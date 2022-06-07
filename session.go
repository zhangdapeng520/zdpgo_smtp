package zdpgo_smtp

import (
	"errors"
	"fmt"
	"github.com/zhangdapeng520/zdpgo_smtp/smtp"
	"io"
	"io/ioutil"
)

/*
@Time : 2022/6/6 10:24
@Author : 张大鹏
@File : session.go
@Software: Goland2021.3.1
@Description:
*/

// Session 会话实现
type Session struct {
}

// AuthPlain 用户名和密码校验
func (s *Session) AuthPlain(username, password string) error {
	fmt.Println("校验权限：", username, password, gConfig.Auths)

	// 校验权限
	if auth, ok := gConfig.Auths[username]; ok {
		if auth.Password == password {
			return nil
		}
	}

	// 校验失败
	return errors.New("用户名或密码错误")
}

func (s *Session) Mail(from string, opts *smtp.MailOptions) error {
	fmt.Println("发件人是：", from)
	return nil
}

func (s *Session) Rcpt(to string) error {
	fmt.Println("收件人是：", to)
	return nil
}

func (s *Session) Data(r io.Reader) error {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		panic(err)
	}
	fmt.Println("接收到的数据是：", string(data))
	return nil
}

func (s *Session) Reset() {
}

func (s *Session) Logout() error {
	return nil
}
