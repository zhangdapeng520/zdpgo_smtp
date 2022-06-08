package smtp

import (
	"io"
)

var (
	ErrAuthRequired = &SMTPError{
		Code:         502,
		EnhancedCode: EnhancedCode{5, 7, 0},
		Message:      "Please authenticate first",
	}
	ErrAuthUnsupported = &SMTPError{
		Code:         502,
		EnhancedCode: EnhancedCode{5, 7, 0},
		Message:      "Authentication not supported",
	}
)

// A SMTP server backend.
type Backend interface {
	NewSession(c ConnectionState) (Session, error)
}

type BodyType string

const (
	Body7Bit       BodyType = "7BIT"
	Body8BitMIME   BodyType = "8BITMIME"
	BodyBinaryMIME BodyType = "BINARYMIME"
)

// MailOptions 邮件参数
type MailOptions struct {
	Body       BodyType // 内容参数：7BIT, 8BITMIME or BINARYMIME.
	Size       int      // 内容大小
	RequireTLS bool     // 是否需要TLS
	UTF8       bool     // 是否为UTF-8
	Auth       *string  // 权限字符串
}

// Session 会话接口
type Session interface {
	Reset()                                    // 启用当前的消息
	Logout() error                             // 注销
	AuthPlain(username, password string) error // 权限校验
	Mail(from string, opts *MailOptions) error // 发件人
	Rcpt(to string) error                      // 收件人
	Data(r io.Reader) error                    // 读取数据
}

// LMTPSession 会话
type LMTPSession interface {
	LMTPData(r io.Reader, status StatusCollector) error
}

// StatusCollector 状态控制
type StatusCollector interface {
	SetStatus(rcptTo string, err error)
}
