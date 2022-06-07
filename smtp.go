package zdpgo_smtp

import (
	"errors"
	"fmt"
	"github.com/zhangdapeng520/zdpgo_cache_http"
	"github.com/zhangdapeng520/zdpgo_json"
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

	// json对象
	Json = zdpgo_json.New()

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

// GetClient 获取SMTP服务客户端
func (s *Smtp) GetClient() (*Client, error) {
	c := &Client{
		Config: s.Config,
		Log:    s.Log,
	}

	// 建立连接
	if c.Config.Client.Host == "" {
		c.Config.Client.Host = "127.0.0.1"
	}
	if c.Config.Client.Port == 0 {
		c.Config.Client.Port = 37333
	}
	addr := fmt.Sprintf("%s:%d", c.Config.Client.Host, c.Config.Client.Port)
	smtpClient, err := smtp.Dial(addr)
	if err != nil {
		c.Log.Error("与SMTP服务建立连接失败", "error", err)
		return nil, err
	}
	c.SmtpClient = smtpClient

	// 校验权限
	if !c.Auth() {
		msg := "权限校验失败，请检查用户名或密码是否正确"
		c.Log.Error(msg)
		return nil, errors.New(msg)
	}

	// 返回
	return c, nil
}
