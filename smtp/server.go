package smtp

import (
	"crypto/tls"
	"errors"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/zhangdapeng520/zdpgo_smtp/sasl"
)

var errTCPAndLMTP = errors.New("smtp: cannot start LMTP server listening on a TCP socket")

// A function that creates SASL servers.
type SaslServerFactory func(conn *Conn) sasl.Server

// Logger 日志接口
type Logger interface {
	Printf(format string, v ...interface{})
	Println(v ...interface{})
}

// Server SMTP服务
type Server struct {
	// TCP or Unix address to listen on.
	Addr string
	// The server TLS configuration.
	TLSConfig *tls.Config
	// Enable LMTP mode, as defined in RFC 2033. LMTP mode cannot be used with a
	// TCP listener.
	LMTP bool

	Domain            string
	MaxRecipients     int
	MaxMessageBytes   int
	MaxLineLength     int
	AllowInsecureAuth bool
	Strict            bool
	Debug             io.Writer
	ErrorLog          Logger
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	EnableSMTPUTF8    bool // 是否开启UTF-8
	EnableREQUIRETLS  bool
	EnableBINARYMIME  bool
	AuthDisabled      bool
	Backend           Backend

	caps  []string
	auths map[string]SaslServerFactory
	done  chan struct{}

	locker    sync.Mutex
	listeners []net.Listener
	conns     map[*Conn]struct{}
}

// NewServer 创建新的SMT服务
func NewServer(be Backend) *Server {
	return &Server{
		MaxLineLength: 2000,

		Backend:  be,
		done:     make(chan struct{}, 1),
		ErrorLog: log.New(os.Stderr, "smtp/server ", log.LstdFlags),
		caps:     []string{"PIPELINING", "8BITMIME", "ENHANCEDSTATUSCODES", "CHUNKING"},
		auths: map[string]SaslServerFactory{
			sasl.Plain: func(conn *Conn) sasl.Server {
				return sasl.NewPlainServer(func(identity, username, password string) error {
					if identity != "" && identity != username {
						return errors.New("Identities not supported")
					}

					sess := conn.Session()
					if sess == nil {
						panic("No session when AUTH is called")
					}

					return sess.AuthPlain(username, password)
				})
			},
		},
		conns: make(map[*Conn]struct{}),
	}
}

// Serve 基于监听器接收消息
func (s *Server) Serve(l net.Listener) error {
	s.locker.Lock()
	s.listeners = append(s.listeners, l)
	s.locker.Unlock()

	var tempDelay time.Duration
	for {
		c, err := l.Accept()
		if err != nil {
			select {
			case <-s.done:
				return nil
			default:
			}
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				if tempDelay == 0 {
					tempDelay = 5 * time.Millisecond
				} else {
					tempDelay *= 2
				}
				if max := 1 * time.Second; tempDelay > max {
					tempDelay = max
				}
				s.ErrorLog.Printf("接收数据失败: %s; 在 %s 后重试", err, tempDelay)
				time.Sleep(tempDelay)
				continue
			}
			return err
		}

		// 开启协程，处理连接
		go func() {
			err = s.handleConn(newConn(c, s))
			if err != nil && !strings.Contains(err.Error(), "closed network connection") {
				s.ErrorLog.Printf("处理客户端连接失败: %s", err)
			}
		}()
	}
}

// handleConn 处理客户端连接
func (s *Server) handleConn(c *Conn) error {
	s.locker.Lock()
	s.conns[c] = struct{}{}
	s.locker.Unlock()

	defer func() {
		c.Close()
		s.locker.Lock()
		delete(s.conns, c)
		s.locker.Unlock()
	}()

	if tlsConn, ok := c.conn.(*tls.Conn); ok {
		if d := s.ReadTimeout; d != 0 {
			c.conn.SetReadDeadline(time.Now().Add(d))
		}
		if d := s.WriteTimeout; d != 0 {
			c.conn.SetWriteDeadline(time.Now().Add(d))
		}
		if err := tlsConn.Handshake(); err != nil {
			return err
		}
	}

	c.greet()

	for {
		line, err := c.ReadLine()
		if err == nil {
			cmd, arg, err := parseCmd(line)
			if err != nil {
				c.protocolError(501, EnhancedCode{5, 5, 2}, "命令错误")
				continue
			}

			c.handle(cmd, arg)
		} else {
			if err == io.EOF {
				return nil
			}
			if err == ErrTooLongLine {
				c.WriteResponse(500, EnhancedCode{5, 4, 0}, "一行太长了，关闭连接")
				return nil
			}

			if neterr, ok := err.(net.Error); ok && neterr.Timeout() {
				c.WriteResponse(221, EnhancedCode{2, 4, 2}, "Idle超时，再见")
				return nil
			}

			c.WriteResponse(221, EnhancedCode{2, 4, 0}, "连接错误，抱歉")
			return err
		}
	}
}

// ListenAndServe 启动SMTP服务
func (s *Server) ListenAndServe() error {
	network := "tcp"
	if s.LMTP {
		network = "unix"
	}

	addr := s.Addr
	if !s.LMTP && addr == "" {
		addr = ":smtp"
	}

	l, err := net.Listen(network, addr)
	if err != nil {
		return err
	}

	return s.Serve(l)
}

// ListenAndServeTLS 启动SMTPS服务
func (s *Server) ListenAndServeTLS() error {
	if s.LMTP {
		return errTCPAndLMTP
	}

	addr := s.Addr
	if addr == "" {
		addr = ":smtps"
	}

	l, err := tls.Listen("tcp", addr, s.TLSConfig)
	if err != nil {
		return err
	}

	return s.Serve(l)
}

// Close 关闭监听器
func (s *Server) Close() error {
	select {
	case <-s.done:
		return errors.New("smtp: 服务已关闭")
	default:
		close(s.done)
	}

	var err error
	s.locker.Lock()
	for _, l := range s.listeners {
		if lerr := l.Close(); lerr != nil && err == nil {
			err = lerr
		}
	}

	for conn := range s.conns {
		conn.Close()
	}
	s.locker.Unlock()

	return err
}

// EnableAuth 开启权限
func (s *Server) EnableAuth(name string, f SaslServerFactory) {
	s.auths[name] = f
}

// ForEachConn iterates through all opened connections.
func (s *Server) ForEachConn(f func(*Conn)) {
	s.locker.Lock()
	defer s.locker.Unlock()
	for conn := range s.conns {
		f(conn)
	}
}
