package smtp

import (
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/textproto"
	"regexp"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Number of errors we'll tolerate per connection before closing. Defaults to 3.
const errThreshold = 3

type ConnectionState struct {
	Hostname   string
	LocalAddr  net.Addr
	RemoteAddr net.Addr
	TLS        tls.ConnectionState
}

type Conn struct {
	conn   net.Conn
	text   *textproto.Conn
	server *Server
	helo   string

	// Number of errors witnessed on this connection
	errCount int

	session    Session
	locker     sync.Mutex
	binarymime bool

	lineLimitReader *lineLimitReader
	bdatPipe        *io.PipeWriter
	bdatStatus      *statusCollector // used for BDAT on LMTP
	dataResult      chan error
	bytesReceived   int // counts total size of chunks when BDAT is used

	fromReceived bool
	recipients   []string
	didAuth      bool
}

func newConn(c net.Conn, s *Server) *Conn {
	sc := &Conn{
		server: s,
		conn:   c,
	}

	sc.init()
	return sc
}

func (c *Conn) init() {
	c.lineLimitReader = &lineLimitReader{
		R:         c.conn,
		LineLimit: c.server.MaxLineLength,
	}
	rwc := struct {
		io.Reader
		io.Writer
		io.Closer
	}{
		Reader: c.lineLimitReader,
		Writer: c.conn,
		Closer: c.conn,
	}

	if c.server.Debug != nil {
		rwc = struct {
			io.Reader
			io.Writer
			io.Closer
		}{
			io.TeeReader(rwc.Reader, c.server.Debug),
			io.MultiWriter(rwc.Writer, c.server.Debug),
			rwc.Closer,
		}
	}

	c.text = textproto.NewConn(rwc)
}

// handle 处理连接命令
func (c *Conn) handle(cmd string, arg string) {
	// 如果抛出了异常，返回421响应并关闭连接对象
	defer func() {
		if err := recover(); err != nil {
			c.WriteResponse(421, EnhancedCode{4, 0, 0}, "服务器内容错误")
			c.Close()

			stack := debug.Stack()
			c.server.ErrorLog.Printf("异常信息 %v: %v\n%s", c.State().RemoteAddr, err, stack)
		}
	}()

	// 命令为空
	if cmd == "" {
		c.protocolError(500, EnhancedCode{5, 5, 2}, "错误：cmd命令不能为空")
		return
	}

	// 处理不同的命令
	cmd = strings.ToUpper(cmd)
	switch cmd {
	case "SEND", "SOML", "SAML", "EXPN", "HELP", "TURN":
		c.WriteResponse(502, EnhancedCode{5, 5, 1}, fmt.Sprintf("%v 命令未实现", cmd))
	case "HELO", "EHLO", "LHLO":
		lmtp := cmd == "LHLO"
		enhanced := lmtp || cmd == "EHLO"
		if c.server.LMTP && !lmtp {
			c.WriteResponse(500, EnhancedCode{5, 5, 1}, "这是一个 LMTP 服务, 使用 LHLO")
			return
		}
		if !c.server.LMTP && lmtp {
			c.WriteResponse(500, EnhancedCode{5, 5, 1}, "这不是一个 LMTP 服务")
			return
		}
		c.handleGreet(enhanced, arg)
	case "MAIL":
		c.handleMail(arg)
	case "RCPT":
		c.handleRcpt(arg)
	case "VRFY":
		c.WriteResponse(252, EnhancedCode{2, 5, 0}, "无法验证用户，但是会接收消息")
	case "NOOP":
		c.WriteResponse(250, EnhancedCode{2, 0, 0}, "成功的处理了NOOP命令")
	case "RSET": // Reset session
		c.reset()
		c.WriteResponse(250, EnhancedCode{2, 0, 0}, "重置会话")
	case "BDAT":
		c.handleBdat(arg)
	case "DATA":
		c.handleData(arg)
	case "QUIT":
		c.WriteResponse(221, EnhancedCode{2, 0, 0}, "再见")
		c.Close()
	case "AUTH":
		if c.server.AuthDisabled {
			c.protocolError(500, EnhancedCode{5, 5, 2}, "语法错误, AUTH 命令不推荐")
		} else {
			c.handleAuth(arg)
		}
	case "STARTTLS":
		c.handleStartTLS()
	default:
		msg := fmt.Sprintf("语法错误, %v 命令不推荐", cmd)
		c.protocolError(500, EnhancedCode{5, 5, 2}, msg)
	}
}

func (c *Conn) Server() *Server {
	return c.server
}

func (c *Conn) Session() Session {
	c.locker.Lock()
	defer c.locker.Unlock()
	return c.session
}

// SetSession 设置会话
func (c *Conn) SetSession(session Session) {
	c.locker.Lock()
	defer c.locker.Unlock()
	c.session = session
}

func (c *Conn) Close() error {
	c.locker.Lock()
	defer c.locker.Unlock()

	if c.bdatPipe != nil {
		c.bdatPipe.CloseWithError(ErrDataReset)
		c.bdatPipe = nil
	}

	if c.session != nil {
		c.session.Logout()
		c.session = nil
	}

	return c.conn.Close()
}

// TLSConnectionState returns the connection's TLS connection state.
// Zero values are returned if the connection doesn't use TLS.
func (c *Conn) TLSConnectionState() (state tls.ConnectionState, ok bool) {
	tc, ok := c.conn.(*tls.Conn)
	if !ok {
		return
	}
	return tc.ConnectionState(), true
}

func (c *Conn) State() ConnectionState {
	state := ConnectionState{}
	tlsState, ok := c.TLSConnectionState()
	if ok {
		state.TLS = tlsState
	}

	state.Hostname = c.helo
	state.LocalAddr = c.conn.LocalAddr()
	state.RemoteAddr = c.conn.RemoteAddr()

	return state
}

func (c *Conn) authAllowed() bool {
	_, isTLS := c.TLSConnectionState()
	return !c.server.AuthDisabled && (isTLS || c.server.AllowInsecureAuth)
}

// protocolError writes errors responses and closes the connection once too many
// have occurred.
func (c *Conn) protocolError(code int, ec EnhancedCode, msg string) {
	c.WriteResponse(code, ec, msg)

	c.errCount++
	if c.errCount > errThreshold {
		c.WriteResponse(500, EnhancedCode{5, 5, 1}, "Too many errors. Quiting now")
		c.Close()
	}
}

// 处理打招呼
func (c *Conn) handleGreet(enhanced bool, arg string) {
	domain, err := parseHelloArgument(arg)
	if err != nil {
		c.WriteResponse(501, EnhancedCode{5, 5, 2}, "Domain/address argument required for HELO")
		return
	}
	c.helo = domain

	sess, err := c.server.Backend.NewSession(c.State())
	if err != nil {
		if smtpErr, ok := err.(*SMTPError); ok {
			c.WriteResponse(smtpErr.Code, smtpErr.EnhancedCode, smtpErr.Message)
			return
		}
		c.WriteResponse(451, EnhancedCode{4, 0, 0}, err.Error())
		return
	}
	c.SetSession(sess)

	if !enhanced {
		c.WriteResponse(250, EnhancedCode{2, 0, 0}, fmt.Sprintf("Hello %s", domain))
		return
	}

	caps := []string{}
	caps = append(caps, c.server.caps...)
	if _, isTLS := c.TLSConnectionState(); c.server.TLSConfig != nil && !isTLS {
		caps = append(caps, "STARTTLS")
	}
	if c.authAllowed() {
		authCap := "AUTH"
		for name := range c.server.auths {
			authCap += " " + name
		}

		caps = append(caps, authCap)
	}
	if c.server.EnableSMTPUTF8 {
		caps = append(caps, "SMTPUTF8")
	}
	if _, isTLS := c.TLSConnectionState(); isTLS && c.server.EnableREQUIRETLS {
		caps = append(caps, "REQUIRETLS")
	}
	if c.server.EnableBINARYMIME {
		caps = append(caps, "BINARYMIME")
	}
	if c.server.MaxMessageBytes > 0 {
		caps = append(caps, fmt.Sprintf("SIZE %v", c.server.MaxMessageBytes))
	} else {
		caps = append(caps, "SIZE")
	}

	args := []string{"Hello " + domain}
	args = append(args, caps...)
	c.WriteResponse(250, NoEnhancedCode, args...)
}

// handleMail 处理邮件内容
func (c *Conn) handleMail(arg string) {
	// 打招呼
	if c.helo == "" {
		c.WriteResponse(502, EnhancedCode{2, 5, 1}, "请先介绍您自己，hello内容不能为空")
		return
	}

	// 管道为空
	if c.bdatPipe != nil {
		c.WriteResponse(502, EnhancedCode{5, 5, 1}, "MAIL 不允许数据传输")
		return
	}

	// 发件人
	if len(arg) < 6 || strings.ToUpper(arg[0:5]) != "FROM:" {
		c.WriteResponse(501, EnhancedCode{5, 5, 2}, "语法错误，期望的格式是 FROM:<address>")
		return
	}
	fromArgs := strings.Split(strings.Trim(arg[5:], " "), " ")
	if c.server.Strict {
		if !strings.HasPrefix(fromArgs[0], "<") || !strings.HasSuffix(fromArgs[0], ">") {
			c.WriteResponse(501, EnhancedCode{5, 5, 2}, "语法错误，期望的格式是 FROM:<address>")
			return
		}
	}
	from := fromArgs[0]
	if from == "" {
		c.WriteResponse(501, EnhancedCode{5, 5, 2}, "语法错误，期望的格式是 FROM:<address>")
		return
	}
	from = strings.Trim(from, "<>")

	// 参数
	opts := &MailOptions{}
	c.binarymime = false
	if len(fromArgs) > 1 {
		// 解析参数
		args, err := parseArgs(fromArgs[1:])
		if err != nil {
			c.WriteResponse(501, EnhancedCode{5, 5, 4}, "解析 MAIL ESMTP 参数失败")
			return
		}

		// 处理参数
		for key, value := range args {
			switch key {
			case "SIZE":
				size, err := strconv.ParseInt(value, 10, 32)
				if err != nil {
					c.WriteResponse(501, EnhancedCode{5, 5, 4}, "无法将SIZE解析为一个整数")
					return
				}
				if c.server.MaxMessageBytes > 0 && int(size) > c.server.MaxMessageBytes {
					c.WriteResponse(552, EnhancedCode{5, 3, 4}, "SIZE超过了最大消息容量限制")
					return
				}
				opts.Size = int(size)
			case "SMTPUTF8":
				if !c.server.EnableSMTPUTF8 {
					c.WriteResponse(504, EnhancedCode{5, 5, 4}, "SMTPUTF8 未实现")
					return
				}
				opts.UTF8 = true
			case "REQUIRETLS":
				if !c.server.EnableREQUIRETLS {
					c.WriteResponse(504, EnhancedCode{5, 5, 4}, "REQUIRETLS 未实现")
					return
				}
				opts.RequireTLS = true
			case "BODY":
				switch value {
				case "BINARYMIME":
					if !c.server.EnableBINARYMIME {
						c.WriteResponse(504, EnhancedCode{5, 5, 4}, "BINARYMIME 未实现")
						return
					}
					c.binarymime = true
				case "7BIT", "8BITMIME":
				default:
					c.WriteResponse(500, EnhancedCode{5, 5, 4}, "未知的 BODY 值")
					return
				}
				opts.Body = BodyType(value)
			case "AUTH":
				value, err = decodeXtext(value)
				if err != nil {
					c.WriteResponse(500, EnhancedCode{5, 5, 4}, "解析权限参数失败")
					return
				}
				if !strings.HasPrefix(value, "<") {
					c.WriteResponse(500, EnhancedCode{5, 5, 4}, "缺少<符号")
					return
				}
				if !strings.HasSuffix(value, ">") {
					c.WriteResponse(500, EnhancedCode{5, 5, 4}, "缺少>符号")
					return
				}
				decodedMbox := value[1 : len(value)-1]
				opts.Auth = &decodedMbox
			default:
				c.WriteResponse(500, EnhancedCode{5, 5, 4}, "未知是收件人信息")
				return
			}
		}
	}

	// 处理邮件
	if err := c.Session().Mail(from, opts); err != nil {
		if smtpErr, ok := err.(*SMTPError); ok {
			c.WriteResponse(smtpErr.Code, smtpErr.EnhancedCode, smtpErr.Message)
			return
		}
		c.WriteResponse(451, EnhancedCode{4, 0, 0}, err.Error())
		return
	}

	c.WriteResponse(250, EnhancedCode{2, 0, 0}, fmt.Sprintf("处理 <%v> 发送的邮件成功", from))
	c.fromReceived = true
}

// This regexp matches 'hexchar' token defined in
// https://tools.ietf.org/html/rfc4954#section-8 however it is intentionally
// relaxed by requiring only '+' to be present.  It allows us to detect
// malformed values such as +A or +HH and report them appropriately.
var hexcharRe = regexp.MustCompile(`\+[0-9A-F]?[0-9A-F]?`)

func decodeXtext(val string) (string, error) {
	if !strings.Contains(val, "+") {
		return val, nil
	}

	var replaceErr error
	decoded := hexcharRe.ReplaceAllStringFunc(val, func(match string) string {
		if len(match) != 3 {
			replaceErr = errors.New("incomplete hexchar")
			return ""
		}
		char, err := strconv.ParseInt(match, 16, 8)
		if err != nil {
			replaceErr = err
			return ""
		}

		return string(rune(char))
	})
	if replaceErr != nil {
		return "", replaceErr
	}

	return decoded, nil
}

func encodeXtext(raw string) string {
	var out strings.Builder
	out.Grow(len(raw))

	for _, ch := range raw {
		if ch == '+' || ch == '=' {
			out.WriteRune('+')
			out.WriteString(strings.ToUpper(strconv.FormatInt(int64(ch), 16)))
		}
		if ch > '!' && ch < '~' { // printable non-space US-ASCII
			out.WriteRune(ch)
		}
		// Non-ASCII.
		out.WriteRune('+')
		out.WriteString(strings.ToUpper(strconv.FormatInt(int64(ch), 16)))
	}
	return out.String()
}

// handleRcpt 处理接收到的消息
func (c *Conn) handleRcpt(arg string) {
	if !c.fromReceived {
		c.WriteResponse(502, EnhancedCode{5, 5, 1}, "发件人不能为空")
		return
	}
	if c.bdatPipe != nil {
		c.WriteResponse(502, EnhancedCode{5, 5, 1}, "RCPT 不允许数据传输")
		return
	}
	if (len(arg) < 4) || (strings.ToUpper(arg[0:3]) != "TO:") {
		c.WriteResponse(501, EnhancedCode{5, 5, 2}, "语法错误，期望的格式是 TO:<address>")
		return
	}

	recipient := strings.Trim(arg[3:], "<> ")
	if c.server.MaxRecipients > 0 && len(c.recipients) >= c.server.MaxRecipients {
		c.WriteResponse(552, EnhancedCode{5, 5, 3}, fmt.Sprintf("超过最大接收数量限制 %v",
			c.server.MaxRecipients))
		return
	}

	// 会话处理接收到的消息
	if err := c.Session().Rcpt(recipient); err != nil {
		if smtpErr, ok := err.(*SMTPError); ok {
			c.WriteResponse(smtpErr.Code, smtpErr.EnhancedCode, smtpErr.Message)
			return
		}
		c.WriteResponse(451, EnhancedCode{4, 0, 0}, err.Error())
		return
	}
	c.recipients = append(c.recipients, recipient)

	// 正确的消息
	c.WriteResponse(250, EnhancedCode{2, 0, 0}, fmt.Sprintf("我会确保 <%v> 接收这条消息", recipient))
}

// handleAuth 处理权限
func (c *Conn) handleAuth(arg string) {
	if c.helo == "" {
		c.WriteResponse(502, EnhancedCode{5, 5, 1}, "请先介绍您自己")
		return
	}
	if c.didAuth {
		c.WriteResponse(503, EnhancedCode{5, 5, 1}, "权限已经校验过")
		return
	}

	parts := strings.Fields(arg)
	if len(parts) == 0 {
		c.WriteResponse(502, EnhancedCode{5, 5, 4}, "缺少参数")
		return
	}

	if _, isTLS := c.TLSConnectionState(); !isTLS && !c.server.AllowInsecureAuth {
		c.WriteResponse(523, EnhancedCode{5, 7, 10}, "需要TLS")
		return
	}

	mechanism := strings.ToUpper(parts[0])

	// Parse client initial response if there is one
	var ir []byte
	if len(parts) > 1 {
		var err error
		ir, err = base64.StdEncoding.DecodeString(parts[1])
		if err != nil {
			return
		}
	}

	newSasl, ok := c.server.auths[mechanism]
	if !ok {
		c.WriteResponse(504, EnhancedCode{5, 7, 4}, "Unsupported authentication mechanism")
		return
	}

	sasl := newSasl(c)

	response := ir
	for {
		challenge, done, err := sasl.Next(response)
		if err != nil {
			if smtpErr, ok := err.(*SMTPError); ok {
				c.WriteResponse(smtpErr.Code, smtpErr.EnhancedCode, smtpErr.Message)
				return
			}
			c.WriteResponse(454, EnhancedCode{4, 7, 0}, err.Error())
			return
		}

		if done {
			break
		}

		encoded := ""
		if len(challenge) > 0 {
			encoded = base64.StdEncoding.EncodeToString(challenge)
		}
		c.WriteResponse(334, NoEnhancedCode, encoded)

		encoded, err = c.ReadLine()
		if err != nil {
			return // TODO: error handling
		}

		if encoded == "*" {
			// https://tools.ietf.org/html/rfc4954#page-4
			c.WriteResponse(501, EnhancedCode{5, 0, 0}, "Negotiation cancelled")
			return
		}

		response, err = base64.StdEncoding.DecodeString(encoded)
		if err != nil {
			c.WriteResponse(454, EnhancedCode{4, 7, 0}, "Invalid base64 data")
			return
		}
	}

	c.WriteResponse(235, EnhancedCode{2, 0, 0}, "Authentication succeeded")
	c.didAuth = true
}

// 处理TLS
func (c *Conn) handleStartTLS() {
	if _, isTLS := c.TLSConnectionState(); isTLS {
		c.WriteResponse(502, EnhancedCode{5, 5, 1}, "Already running in TLS")
		return
	}

	if c.server.TLSConfig == nil {
		c.WriteResponse(502, EnhancedCode{5, 5, 1}, "TLS not supported")
		return
	}

	c.WriteResponse(220, EnhancedCode{2, 0, 0}, "Ready to start TLS")

	// Upgrade to TLS
	tlsConn := tls.Server(c.conn, c.server.TLSConfig)

	if err := tlsConn.Handshake(); err != nil {
		c.WriteResponse(550, EnhancedCode{5, 0, 0}, "Handshake error")
		return
	}

	c.conn = tlsConn
	c.init()

	// Reset all state and close the previous Session.
	// This is different from just calling reset() since we want the Backend to
	// be able to see the information about TLS connection in the
	// ConnectionState object passed to it.
	if session := c.Session(); session != nil {
		session.Logout()
		c.SetSession(nil)
	}
	c.helo = ""
	c.didAuth = false
	c.reset()
}

// 处理数据
func (c *Conn) handleData(arg string) {
	if arg != "" {
		c.WriteResponse(501, EnhancedCode{5, 5, 4}, "参数不能为空")
		return
	}
	if c.bdatPipe != nil {
		c.WriteResponse(502, EnhancedCode{5, 5, 1}, "不允许数据传输")
		return
	}
	if c.binarymime {
		c.WriteResponse(502, EnhancedCode{5, 5, 1}, "不支持BINARYMIME类型的消息")
		return
	}

	if !c.fromReceived || len(c.recipients) == 0 {
		c.WriteResponse(502, EnhancedCode{5, 5, 1}, "缺少 RCPT TO 命令")
		return
	}

	// We have recipients, go to accept data
	c.WriteResponse(354, EnhancedCode{2, 0, 0}, "接收到数据。以以下方式结束数据 <CR><LF>.<CR><LF>")

	defer c.reset()
	if c.server.LMTP {
		c.handleDataLMTP()
		return
	}

	r := newDataReader(c)
	code, enhancedCode, msg := toSMTPStatus(c.Session().Data(r))
	r.limited = false
	io.Copy(ioutil.Discard, r) // Make sure all the data has been consumed
	c.WriteResponse(code, enhancedCode, msg)
}

func (c *Conn) handleBdat(arg string) {
	args := strings.Fields(arg)
	if len(args) == 0 {
		c.WriteResponse(501, EnhancedCode{5, 5, 4}, "缺少chunk size参数")
		return
	}
	if len(args) > 2 {
		c.WriteResponse(501, EnhancedCode{5, 5, 4}, "参数过多")
		return
	}

	if !c.fromReceived || len(c.recipients) == 0 {
		c.WriteResponse(502, EnhancedCode{5, 5, 1}, "缺少RCPT TO命令")
		return
	}

	last := false
	if len(args) == 2 {
		if !strings.EqualFold(args[1], "LAST") {
			c.WriteResponse(501, EnhancedCode{5, 5, 4}, "未知的 BDAT 参数")
			return
		}
		last = true
	}

	// ParseUint instead of Atoi so we will not accept negative values.
	size, err := strconv.ParseUint(args[0], 10, 32)
	if err != nil {
		c.WriteResponse(501, EnhancedCode{5, 5, 4}, "Malformed size argument")
		return
	}

	if c.server.MaxMessageBytes != 0 && c.bytesReceived+int(size) > c.server.MaxMessageBytes {
		c.WriteResponse(552, EnhancedCode{5, 3, 4}, "超过最大消息长度限制")

		// Discard chunk itself without passing it to backend.
		io.Copy(ioutil.Discard, io.LimitReader(c.text.R, int64(size)))

		c.reset()
		return
	}

	if c.bdatStatus == nil && c.server.LMTP {
		c.bdatStatus = c.createStatusCollector()
	}

	if c.bdatPipe == nil {
		var r *io.PipeReader
		r, c.bdatPipe = io.Pipe()

		c.dataResult = make(chan error, 1)

		go func() {
			defer func() {
				if err := recover(); err != nil {
					c.handlePanic(err, c.bdatStatus)

					c.dataResult <- errPanic
					r.CloseWithError(errPanic)
				}
			}()

			var err error
			if !c.server.LMTP {
				err = c.Session().Data(r)
			} else {
				lmtpSession, ok := c.Session().(LMTPSession)
				if !ok {
					err = c.Session().Data(r)
					for _, rcpt := range c.recipients {
						c.bdatStatus.SetStatus(rcpt, err)
					}
				} else {
					err = lmtpSession.LMTPData(r, c.bdatStatus)
				}
			}

			c.dataResult <- err
			r.CloseWithError(err)
		}()
	}

	c.lineLimitReader.LineLimit = 0

	chunk := io.LimitReader(c.text.R, int64(size))
	_, err = io.Copy(c.bdatPipe, chunk)
	if err != nil {
		// Backend might return an error early using CloseWithError without consuming
		// the whole chunk.
		io.Copy(ioutil.Discard, chunk)

		c.WriteResponse(toSMTPStatus(err))

		if err == errPanic {
			c.Close()
		}

		c.reset()
		c.lineLimitReader.LineLimit = c.server.MaxLineLength
		return
	}

	c.bytesReceived += int(size)

	if last {
		c.lineLimitReader.LineLimit = c.server.MaxLineLength

		c.bdatPipe.Close()

		err := <-c.dataResult

		if c.server.LMTP {
			c.bdatStatus.fillRemaining(err)
			for i, rcpt := range c.recipients {
				code, enchCode, msg := toSMTPStatus(<-c.bdatStatus.status[i])
				c.WriteResponse(code, enchCode, "<"+rcpt+"> "+msg)
			}
		} else {
			c.WriteResponse(toSMTPStatus(err))
		}

		if err == errPanic {
			c.Close()
			return
		}

		c.reset()
	} else {
		c.WriteResponse(250, EnhancedCode{2, 0, 0}, "Continue")
	}
}

// ErrDataReset is returned by Reader pased to Data function if client does not
// send another BDAT command and instead closes connection or issues RSET command.
var ErrDataReset = errors.New("smtp: 消息传输失败")

var errPanic = &SMTPError{
	Code:         421,
	EnhancedCode: EnhancedCode{4, 0, 0},
	Message:      "服务器内部错误",
}

func (c *Conn) handlePanic(err interface{}, status *statusCollector) {
	if status != nil {
		status.fillRemaining(errPanic)
	}

	stack := debug.Stack()
	c.server.ErrorLog.Printf("panic serving %v: %v\n%s", c.State().RemoteAddr, err, stack)
}

func (c *Conn) createStatusCollector() *statusCollector {
	rcptCounts := make(map[string]int, len(c.recipients))

	status := &statusCollector{
		statusMap: make(map[string]chan error, len(c.recipients)),
		status:    make([]chan error, 0, len(c.recipients)),
	}
	for _, rcpt := range c.recipients {
		rcptCounts[rcpt]++
	}
	// Create channels with buffer sizes necessary to fit all
	// statuses for a single recipient to avoid deadlocks.
	for rcpt, count := range rcptCounts {
		status.statusMap[rcpt] = make(chan error, count)
	}
	for _, rcpt := range c.recipients {
		status.status = append(status.status, status.statusMap[rcpt])
	}

	return status
}

type statusCollector struct {
	// Contains map from recipient to list of channels that are used for that
	// recipient.
	statusMap map[string]chan error

	// Contains channels from statusMap, in the same
	// order as Conn.recipients.
	status []chan error
}

// fillRemaining sets status for all recipients SetStatus was not called for before.
func (s *statusCollector) fillRemaining(err error) {
	// Amount of times certain recipient was specified is indicated by the channel
	// buffer size, so once we fill it, we can be confident that we sent
	// at least as much statuses as needed. Extra statuses will be ignored anyway.
chLoop:
	for _, ch := range s.statusMap {
		for {
			select {
			case ch <- err:
			default:
				continue chLoop
			}
		}
	}
}

func (s *statusCollector) SetStatus(rcptTo string, err error) {
	ch := s.statusMap[rcptTo]
	if ch == nil {
		panic("SetStatus is called for recipient that was not specified before")
	}

	select {
	case ch <- err:
	default:
		// There enough buffer space to fit all statuses at once, if this is
		// not the case - backend is doing something wrong.
		panic("SetStatus is called more times than particular recipient was specified")
	}
}

func (c *Conn) handleDataLMTP() {
	r := newDataReader(c)
	status := c.createStatusCollector()

	done := make(chan bool, 1)

	lmtpSession, ok := c.Session().(LMTPSession)
	if !ok {
		// Fallback to using a single status for all recipients.
		err := c.Session().Data(r)
		io.Copy(ioutil.Discard, r) // Make sure all the data has been consumed
		for _, rcpt := range c.recipients {
			status.SetStatus(rcpt, err)
		}
		done <- true
	} else {
		go func() {
			defer func() {
				if err := recover(); err != nil {
					status.fillRemaining(&SMTPError{
						Code:         421,
						EnhancedCode: EnhancedCode{4, 0, 0},
						Message:      "Internal server error",
					})

					stack := debug.Stack()
					c.server.ErrorLog.Printf("panic serving %v: %v\n%s", c.State().RemoteAddr, err, stack)
					done <- false
				}
			}()

			status.fillRemaining(lmtpSession.LMTPData(r, status))
			io.Copy(ioutil.Discard, r) // Make sure all the data has been consumed
			done <- true
		}()
	}

	for i, rcpt := range c.recipients {
		code, enchCode, msg := toSMTPStatus(<-status.status[i])
		c.WriteResponse(code, enchCode, "<"+rcpt+"> "+msg)
	}

	// If done gets false, the panic occured in LMTPData and the connection
	// should be closed.
	if !<-done {
		c.Close()
	}
}

func toSMTPStatus(err error) (code int, enchCode EnhancedCode, msg string) {
	if err != nil {
		if smtperr, ok := err.(*SMTPError); ok {
			return smtperr.Code, smtperr.EnhancedCode, smtperr.Message
		} else {
			return 554, EnhancedCode{5, 0, 0}, "Error: transaction failed, blame it on the weather: " + err.Error()
		}
	}

	return 250, EnhancedCode{2, 0, 0}, "OK: queued"
}

func (c *Conn) Reject() {
	c.WriteResponse(421, EnhancedCode{4, 4, 5}, "Too busy. Try again later.")
	c.Close()
}

// greet 准备建立会话
func (c *Conn) greet() {
	c.WriteResponse(220, NoEnhancedCode, fmt.Sprintf("%v ESMTP 服务已准备就绪", c.server.Domain))
}

// WriteResponse 写入响应
func (c *Conn) WriteResponse(code int, enhCode EnhancedCode, text ...string) {
	if c.server.WriteTimeout != 0 {
		c.conn.SetWriteDeadline(time.Now().Add(c.server.WriteTimeout))
	}

	// All responses must include an enhanced code, if it is missing - use
	// a generic code X.0.0.
	if enhCode == EnhancedCodeNotSet {
		cat := code / 100
		switch cat {
		case 2, 4, 5:
			enhCode = EnhancedCode{cat, 0, 0}
		default:
			enhCode = NoEnhancedCode
		}
	}

	for i := 0; i < len(text)-1; i++ {
		c.text.PrintfLine("%d-%v", code, text[i])
	}
	if enhCode == NoEnhancedCode {
		c.text.PrintfLine("%d %v", code, text[len(text)-1])
	} else {
		c.text.PrintfLine("%d %v.%v.%v %v", code, enhCode[0], enhCode[1], enhCode[2], text[len(text)-1])
	}
}

// ReadLine 读取一行 输入
func (c *Conn) ReadLine() (string, error) {
	if c.server.ReadTimeout != 0 {
		if err := c.conn.SetReadDeadline(time.Now().Add(c.server.ReadTimeout)); err != nil {
			return "", err
		}
	}
	return c.text.ReadLine()
}

func (c *Conn) reset() {
	c.locker.Lock()
	defer c.locker.Unlock()

	if c.bdatPipe != nil {
		c.bdatPipe.CloseWithError(ErrDataReset)
		c.bdatPipe = nil
	}
	c.bdatStatus = nil
	c.bytesReceived = 0

	if c.session != nil {
		c.session.Reset()
	}

	c.fromReceived = false
	c.recipients = nil
}
