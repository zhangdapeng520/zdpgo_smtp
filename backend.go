package zdpgo_smtp

import "github.com/zhangdapeng520/zdpgo_smtp/smtp"

/*
@Time : 2022/6/6 10:24
@Author : 张大鹏
@File : backend.go
@Software: Goland2021.3.1
@Description:
*/

// Backend 后台实现
type Backend struct {
}

func (bkd *Backend) NewSession(c smtp.ConnectionState) (smtp.Session, error) {
	return &Session{}, nil
}
