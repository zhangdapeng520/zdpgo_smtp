package zdpgo_smtp

import (
	"fmt"
	"github.com/zhangdapeng520/zdpgo_cache_http"
	"github.com/zhangdapeng520/zdpgo_email"
	"github.com/zhangdapeng520/zdpgo_requests"
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
	Cache  *zdpgo_cache_http.Client
}

func New() *Smtp {
	return NewWitchConfig(&Config{})
}

func NewWitchConfig(config *Config) *Smtp {
	s := &Smtp{}

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
		config.Auths["zhangdapeng520@zhangdapeng520.com"] = Auth{
			Username: "zhangdapeng520@zhangdapeng520.com",
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

	// 返回
	return s
}

// RunCache 运行缓存服务
func (s *Smtp) RunCache() error {
	cacheServer := zdpgo_cache_http.NewWithConfig(&zdpgo_cache_http.Config{
		Debug: s.Config.Debug,
		Server: zdpgo_cache_http.HttpInfo{
			Host: s.Config.Cache.Host,
			Port: s.Config.Cache.Port,
		},
	})
	err := cacheServer.Run()
	if err != nil {
		return err
	}
	return nil
}

// GetCacheClient 获取缓存客户端对象
func (s *Smtp) GetCacheClient() *zdpgo_cache_http.Client {
	cacheServer := zdpgo_cache_http.NewWithConfig(&zdpgo_cache_http.Config{
		Debug: s.Config.Debug,
		Client: zdpgo_cache_http.HttpInfo{
			Port: s.Config.Cache.Port,
		},
	})

	// 获取客户端
	client := cacheServer.GetClient()

	// 返回客户端对象
	return client
}

func (s *Smtp) Run() error {

	// 启动缓存服务
	go func() {
		err := s.RunCache()
		if err != nil {
			fmt.Println("运行缓存服务失败", "error", err)
		}
	}()

	// 创建缓存客户端
	cache = zdpgo_cache_http.NewClient(zdpgo_requests.New(), &zdpgo_cache_http.Config{
		Debug: s.Config.Debug,
		Client: zdpgo_cache_http.HttpInfo{
			Host: s.Config.Cache.Host,
			Port: s.Config.Cache.Port,
		},
	})

	// 初始化密码
	for _, auth := range s.Config.Auths {
		cache.Set(auth.Username, auth.Password)
	}

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
	err := s.Server.ListenAndServe()
	if err != nil {
		return err
	}

	// 返回
	return nil
}

// GetClient 获取客户端
func (s *Smtp) GetClient() (*Client, error) {
	// 客户端配置
	if s.Config.Client.Email == "" {
		s.Config.Client.Email = "zhangdapeng520@zhangdapeng520.com"
	}
	if s.Config.Client.Username == "" {
		s.Config.Client.Username = "zhangdapeng520@zhangdapeng520.com"
	}
	if s.Config.Client.Password == "" {
		s.Config.Client.Password = "zhangdapeng520"
	}
	if s.Config.Client.Host == "" {
		s.Config.Client.Host = "127.0.0.1"
	}
	if s.Config.Client.Port == 0 {
		s.Config.Client.Port = 37333
	}
	if s.Config.Client.Cache.Host == "" {
		s.Config.Client.Cache.Host = "127.0.0.1"
	}
	if s.Config.Client.Cache.Port == 0 {
		s.Config.Client.Cache.Port = 37334
	}

	// 邮件
	e, err := zdpgo_email.NewWithConfig(&zdpgo_email.Config{
		Email:    s.Config.Client.Email,
		Username: s.Config.Client.Username,
		Password: s.Config.Client.Password,
		Host:     s.Config.Client.Host,
		Port:     s.Config.Client.Port,
		IsSSL:    false,
	})

	// 客户端
	return &Client{
		Config: s.Config,
		Email:  e,
		Cache:  s.GetCacheClient(),
	}, err
}
