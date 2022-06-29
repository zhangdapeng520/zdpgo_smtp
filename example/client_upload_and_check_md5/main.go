package main

import (
	"fmt"
	"github.com/zhangdapeng520/zdpgo_smtp"
)

/*
@Time : 2022/6/8 15:27
@Author : 张大鹏
@File : main.go
@Software: Goland2021.3.1
@Description:
*/

func main() {
	s := zdpgo_smtp.NewWitchConfig(&zdpgo_smtp.Config{
		Debug: true,
		Client: zdpgo_smtp.ClientConfig{
			Port: 3333,
		},
	})

	client, err := s.GetClient()
	if err != nil {
		panic(err)
	}

	var flag bool
	flag = client.UploadAndCheckMd5("README.md")
	if !flag {
		panic("上传文件失败")
	}
	fmt.Println("上传文件成功")

	flag = client.UploadAndCheckMd5("example/server/main.go")
	if !flag {
		panic("上传文件失败")
	}
	fmt.Println("上传文件成功")
}
