package main

import (
	"errors"
	"flag"
	"fmt"
	"github.com/zhangdapeng520/zdpgo_smtp/smtp"
	"io"
	"log"
	"os"
)

/*
@Time : 2022/6/2 17:21
@Author : 张大鹏
@File : main.go
@Software: Goland2021.3.1
@Description:
*/

var addr = "127.0.0.1:1025"

type backend struct{}

func (bkd *backend) NewSession(c smtp.ConnectionState) (smtp.Session, error) {
	return &session{}, nil
}

type session struct{}

func (s *session) AuthPlain(username, password string) error {
	if username != "zhangdapeng520" || password != "zhangdapeng520" {
		return errors.New("用户名或密码错误")
	}
	fmt.Println("权限校验成功", username, password)
	return nil
}

func (s *session) Mail(from string, opts *smtp.MailOptions) error {
	return nil
}

func (s *session) Rcpt(to string) error {
	return nil
}

func (s *session) Data(r io.Reader) error {
	return nil
}

func (s *session) Reset() {}

func (s *session) Logout() error {
	return nil
}

func main() {
	flag.Parse()

	s := smtp.NewServer(&backend{})

	s.Addr = addr
	s.Domain = "localhost"
	s.AllowInsecureAuth = true
	s.Debug = os.Stdout

	log.Println("Starting SMTP server at", addr)
	log.Fatal(s.ListenAndServe())
}
