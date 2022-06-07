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
	gMessage.From = from
	return nil
}

func (s *Session) Rcpt(to string) error {
	gMessage.To = to
	return nil
}

func (s *Session) Data(r io.Reader) error {
	// 读取客户端数据
	data, err := ioutil.ReadAll(r)
	if err != nil {
		Log.Error("读取客户端数据失败", "error", err)
		return err
	}

	// 解析客户端数据
	err = gMessage.ParseString(string(data))
	if err != nil {
		Log.Error("解析客户端数据失败", "error", err)
		return err
	}
	Log.Debug("解析数据成功", "msg", gMessage)
	return nil
}

func (s *Session) Reset() {
	gMessage = &Message{}
}

func (s *Session) Logout() error {
	gMessage = &Message{}
	return nil
}
