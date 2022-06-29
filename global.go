package zdpgo_smtp

import "github.com/zhangdapeng520/zdpgo_cache_http"

/*
@Time : 2022/6/7 17:08
@Author : 张大鹏
@File : global.go
@Software: Goland2021.3.1
@Description:
*/

var (
	authMap  = make(map[string]string)
	cache    *zdpgo_cache_http.Client
	gMessage = &Message{}
)
