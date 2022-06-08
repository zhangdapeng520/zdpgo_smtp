package main

import (
	"github.com/zhangdapeng520/zdpgo_smtp"
)

/*
@Time : 2022/6/2 17:21
@Author : 张大鹏
@File : main.go
@Software: Goland2021.3.1
@Description:
*/

func main() {
	s := zdpgo_smtp.NewWitchConfig(&zdpgo_smtp.Config{
		Debug: true,
		Port:  3333,
	})
	err := s.Run()
	if err != nil {
		panic(err)
	}
}
