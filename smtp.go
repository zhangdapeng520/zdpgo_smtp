package zdpgo_smtp

import (
	"fmt"
	"github.com/zhangdapeng520/zdpgo_cache_http"
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

func New() *Smtp {
	return NewWitchConfig(&Config{})
}

func NewWitchConfig(config *Config) *Smtp {
	s := &Smtp{}

	// 日志
	if config.LogFilePath == "" {
		config.LogFilePath = "logs/zdpgo/zdpgo_smtp.log"
	}
	s.Log = zdpgo_log.NewWithDebug(config.Debug, config.LogFilePath)
	Log = s.Log

	// 服务
	s.Server = smtp.NewServer(&Backend{})
	if config.Domain == "" {
		config.Domain = "localhost"
	}
	if config.Host == "" {
		config.Host = "0.0.0.0"
	}
	if config.Port == 0 {
		config.Port = 37333
	}
	s.Server.Addr = fmt.Sprintf("%s:%d", config.Host, config.Port)
	s.Server.Domain = config.Domain
	s.Server.AllowInsecureAuth = true
	s.Server.Debug = os.Stdout

	// 权限
	if config.Auths == nil {
		config.Auths = make(map[string]Auth)
	}
	if len(config.Auths) == 0 {
		config.Auths["zhangdapeng520"] = Auth{
			Username: "zhangdapeng520",
			Password: "zhangdapeng520",
		}
	}

	// 缓存
	if config.Cache.Host == "" {
		config.Cache.Host = "127.0.0.1"
	}
	if config.Cache.Port == 0 {
		config.Cache.Port = 37334
	}

	// 配置
	s.Config = config
	gConfig = config

	// 返回
	return s
}

// RunCache 运行缓存服务
func (s *Smtp) RunCache() error {
	cache := zdpgo_cache_http.NewWithConfig(&zdpgo_cache_http.Config{
		Debug: s.Config.Debug,
		Server: zdpgo_cache_http.HttpInfo{
			Host: s.Config.Cache.Host,
			Port: s.Config.Cache.Port,
		},
	})
	err := cache.Run()
	if err != nil {
		s.Log.Error("启动缓存服务失败", "error", err)
		return err
	}
	return nil
}

// GetCacheClient 获取缓存客户端对象
func (s *Smtp) GetCacheClient() *zdpgo_cache_http.Client {
	cache := zdpgo_cache_http.NewWithConfig(&zdpgo_cache_http.Config{
		Debug: s.Config.Debug,
		Client: zdpgo_cache_http.HttpInfo{
			Port: s.Config.Cache.Port,
		},
	})

	// 获取客户端
	client := cache.GetClient()

	// 返回客户端对象
	return client
}

func (s *Smtp) Run() error {
	// 运行缓存服务
	go func() {
		err := s.RunCache()
		if err != nil {
			s.Log.Error("运行缓存服务失败", "error", err)
		}
	}()

	// 创建服务
	if s.Server == nil {
		s.Server = smtp.NewServer(&Backend{})
		if s.Config.Domain == "" {
			s.Config.Domain = "localhost"
		}
		if s.Config.Host == "" {
			s.Config.Host = "0.0.0.0"
		}
		if s.Config.Port == 0 {
			s.Config.Port = 37333
		}
		s.Server.Addr = fmt.Sprintf("%s:%d", s.Config.Host, s.Config.Port)
		s.Server.Domain = s.Config.Domain
		s.Server.AllowInsecureAuth = true
		s.Server.Debug = os.Stdout
	}

	// 启动服务
	s.Log.Debug("启动SMTP服务", "port", s.Config.Port)
	err := s.Server.ListenAndServe()
	if err != nil {
		s.Log.Error("启动SMTP服务失败", "error", err)
		return err
	}

	// 返回
	return nil
}
