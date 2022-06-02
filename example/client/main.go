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
	c, err := smtp.Dial("localhost:1025")
	if err != nil {
		panic(err)
	}
	defer c.Close()

	// 权限
	//auth := sasl.NewPlainClient("", "zhangdapeng520", "zhangdapeng520")
	auth := sasl.NewPlainClient("", "zhangdapeng5201", "zhangdapeng520")
	err = c.Auth(auth)
	if err != nil {
		panic(err)
	}

	// Set the sender and recipient, and send the email all in one step.
	to := []string{"recipient@example.net"}
	msg := strings.NewReader("To: recipient@example.net\r\n" +
		"Subject: discount Gophers!\r\n" +
		"\r\n" +
		"This is the email body.\r\n")
	err = c.SendMail("sender@example.org", to, msg)
	if err != nil {
		log.Fatal(err)
	}
}
