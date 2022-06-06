package zdpgo_smtp

import (
	"github.com/zhangdapeng520/zdpgo_log"
	"github.com/zhangdapeng520/zdpgo_smtp/sasl"
	"github.com/zhangdapeng520/zdpgo_smtp/smtp"
)

/*
@Time : 2022/6/6 10:42
@Author : 张大鹏
@File : client.go
@Software: Goland2021.3.1
@Description:
*/

type Client struct {
	Config     *Config
	Log        *zdpgo_log.Log
	SmtpClient *smtp.Client
}

type SendRequest struct {
	From string   `json:"from"`
	To   []string `json:"to"`
	Msg  Message  `json:"msg"`
}

// Auth 权限校验
func (c *Client) Auth() bool {
	// 权限校验
	if c.Config.Client.Username == "" {
		c.Config.Client.Username = "zhangdapeng520"
	}
	if c.Config.Client.Password == "" {
		c.Config.Client.Password = "zhangdapeng520"
	}
	auth := sasl.NewPlainClient("", c.Config.Client.Username, c.Config.Client.Password)

	// 校验失败
	err := c.SmtpClient.Auth(auth)
	if err != nil {
		c.Log.Error("权限校验失败", "error", err)
		return false
	}

	// 校验成功
	return true
}

// Send 发送邮件
func (c *Client) Send(request SendRequest) error {
	err := c.SmtpClient.SendMail(request.From, request.To, request.Msg.ToReader())
	if err != nil {
		c.Log.Error("发送邮件失败", "error", err)
		return err
	}
	return nil
}
