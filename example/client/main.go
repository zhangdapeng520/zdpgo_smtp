package main

import (
	"github.com/zhangdapeng520/zdpgo_smtp"
)

/*
@Time : 2022/6/2 17:28
@Author : 张大鹏
@File : main.go
@Software: Goland2021.3.1
@Description:
*/

func main() {
	s := zdpgo_smtp.NewWitchConfig(&zdpgo_smtp.Config{
		Debug: true,
		Client: zdpgo_smtp.ClientInfo{
			Port:     3333,
			Username: "zhangdapeng520",
			Password: "zhangdapeng520",
		},
	})

	// 获取客户端
	client, err := s.GetClient()
	if err != nil {
		panic(err)
	}

	// 发送邮件
	err = client.Send(zdpgo_smtp.SendRequest{
		From: "1156956636@qq.com",
		To:   []string{"lxgzhw@163.com"},
		Msg: zdpgo_smtp.Message{
			To:      "lxgzhw@163.com",
			Subject: "这是一封测试邮件",
			Body:    "这是邮件的内容",
		},
	})

	if err != nil {
		panic(err)
	}
}
