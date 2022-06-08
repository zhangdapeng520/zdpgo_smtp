package zdpgo_smtp

import (
	"github.com/zhangdapeng520/zdpgo_log"
)

/*
@Time : 2022/6/7 17:08
@Author : 张大鹏
@File : global.go
@Software: Goland2021.3.1
@Description:
*/

var (
	gConfig  *Config
	Log      *zdpgo_log.Log
	gMessage = &Message{}
)
