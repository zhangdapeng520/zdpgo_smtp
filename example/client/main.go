package main

import (
	"github.com/zhangdapeng520/zdpgo_smtp/sasl"
	"github.com/zhangdapeng520/zdpgo_smtp/smtp"
	"log"
	"strings"
)

/*
@Time : 2022/6/2 17:28
@Author : 张大鹏
@File : main.go
@Software: Goland2021.3.1
@Description:
*/

func main() {
	// Setup an unencrypted connection to a local mail server.
	c, err := smtp.Dial("localhost:3333")
	if err != nil {
		panic(err)
	}
	defer c.Close()

	// 权限
	auth := sasl.NewPlainClient("", "zhangdapeng520", "zhangdapeng520")
	err = c.Auth(auth)
	if err != nil {
		panic(err)
	}

	// Set the sender and recipient, and send the email all in one step.
	to := []string{"recipient@example.net"}

	// 邮件内容
	msg := strings.NewReader("To: lxgzhw@163.com\r\n" +
		"Subject: 这是一封测试邮件\r\n" +
		"\r\n" +
		"这是邮件的内容\r\n")

	err = c.SendMail("1156956636@qq.com", to, msg)
	if err != nil {
		log.Fatal(err)
	}
}
