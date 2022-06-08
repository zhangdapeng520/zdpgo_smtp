package main

import (
	"fmt"
	"github.com/zhangdapeng520/zdpgo_email"
)

/*
@Time : 2022/6/8 15:27
@Author : 张大鹏
@File : main.go
@Software: Goland2021.3.1
@Description:
*/

func main() {
	e, _ := zdpgo_email.NewWithConfig(&zdpgo_email.Config{
		Debug:    true,
		Email:    "zhangdapeng520@zhangdapeng520.com",
		Username: "zhangdapeng520",
		Password: "zhangdapeng520",
		Host:     "127.0.0.1",
		Port:     3333,
		IsSSL:    false,
	})
	req := zdpgo_email.EmailRequest{
		Title:       "单个HTML测试1",
		Body:        "https://www.baidu.com",
		ToEmails:    []string{"lxgzhw@163.com", "1156956636@qq.com"},
		Attachments: []string{"README.md", "global.go", "message.go"},
	}
	result, err := e.Send(req)
	if err != nil {
		e.Log.Error("发送邮件失败", "error", err)
	}
	fmt.Println(result.Title, result.SendStatus, result.Key, result.StartTime, result.EndTime)
}
