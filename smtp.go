package zdpgo_smtp

import (
	"fmt"
	"github.com/zhangdapeng520/zdpgo_log"
	"github.com/zhangdapeng520/zdpgo_smtp/smtp"
	"os"
)

/*
@Time : 2022/6/6 10:23
@Author : 张大鹏
@File : smtp.go
@Software: Goland2021.3.1
@Description:
*/

type Smtp struct {
	Config *Config
	Server *smtp.Server
	Log    *zdpgo_log.Log
}

func New() {

}

func NewWitchConfig(config *Config) *Smtp {
	s := &Smtp{}

	// 日志
	if config.LogFilePath == "" {
		config.LogFilePath = "logs/zdpgo/zdpgo_smtp.log"
	}
	s.Log = zdpgo_log.NewWithDebug(config.Debug, config.LogFilePath)

	// 服务
	s.Server = smtp.NewServer(&Backend{})
	if config.Server.Domain == "" {
		config.Server.Domain = "localhost"
	}
	if config.Server.Host == "" {
		config.Server.Host = "0.0.0.0"
	}
	if config.Server.Port == 0 {
		config.Server.Port = 37333
	}
	s.Server.Addr = fmt.Sprintf("%s:%d", config.Server.Host, config.Server.Port)
	s.Server.Domain = config.Server.Domain
	s.Server.AllowInsecureAuth = true
	s.Server.Debug = os.Stdout

	// 配置
	s.Config = config

	// 返回
	return s
}

func (s *Smtp) Run() error {
	// 创建服务
	if s.Server == nil {
		s.Server = smtp.NewServer(&Backend{})
		if s.Config.Server.Domain == "" {
			s.Config.Server.Domain = "localhost"
		}
		if s.Config.Server.Host == "" {
			s.Config.Server.Host = "0.0.0.0"
		}
		if s.Config.Server.Port == 0 {
			s.Config.Server.Port = 37333
		}
		s.Server.Addr = fmt.Sprintf("%s:%d", s.Config.Server.Host, s.Config.Server.Port)
		s.Server.Domain = s.Config.Server.Domain
		s.Server.AllowInsecureAuth = true
		s.Server.Debug = os.Stdout
	}

	// 启动服务
	s.Log.Debug("启动SMTP服务", "port", s.Config.Server.Port)
	err := s.Server.ListenAndServe()
	if err != nil {
		s.Log.Error("启动SMTP服务失败", "error", err)
		return err
	}

	// 返回
	return nil
}
