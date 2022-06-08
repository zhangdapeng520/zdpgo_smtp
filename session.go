package zdpgo_smtp

import (
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/zhangdapeng520/zdpgo_cache_http"
	"github.com/zhangdapeng520/zdpgo_smtp/smtp"
	"io"
	"io/ioutil"
	"strings"
)

/*
@Time : 2022/6/6 10:24
@Author : 张大鹏
@File : session.go
@Software: Goland2021.3.1
@Description:
*/

// Session 会话实现
type Session struct {
}

// AuthPlain 用户名和密码校验
func (s *Session) AuthPlain(username, password string) error {
	fmt.Println("校验权限：", username, password, gConfig.Auths)

	// 校验权限
	if auth, ok := gConfig.Auths[username]; ok {
		if auth.Password == password {
			return nil
		}
	}

	// 校验失败
	return errors.New("用户名或密码错误")
}

func (s *Session) Mail(from string, opts *smtp.MailOptions) error {
	gMessage.From = from
	return nil
}

func (s *Session) Rcpt(to string) error {
	gMessage.To = strings.Split(to, ",")
	return nil
}

func (s *Session) Data(r io.Reader) error {
	// 读取客户端数据
	data, err := ioutil.ReadAll(r)
	if err != nil {
		Log.Error("读取客户端数据失败", "error", err)
		return err
	}

	// 解析客户端数据
	err = gMessage.ParseString(string(data))
	if err != nil {
		Log.Error("解析客户端数据失败", "error", err)
		return err
	}
	Log.Debug("解析数据成功", "msg", gMessage)

	// 将数据缓存
	cacheClient := s.GetCacheClient()
	key := s.GetMd5([]byte(fmt.Sprintf("%s--%s--%s", gMessage.From, gMessage.Author, gMessage.Body)))
	value, err := json.Marshal(gMessage)
	if err != nil {
		Log.Error("将消息内容序列化为JSON数据失败", "error", err)
		return err
	}
	cacheClient.Set(key, string(value))

	// 返回
	return nil
}

func (s *Session) Reset() {
	gMessage = &Message{}
}

func (s *Session) Logout() error {
	gMessage = &Message{}
	return nil
}

// GetCacheClient 获取缓存客户端
func (s *Session) GetCacheClient() *zdpgo_cache_http.Client {
	cache := zdpgo_cache_http.NewWithConfig(&zdpgo_cache_http.Config{
		Debug: gConfig.Debug,
		Client: zdpgo_cache_http.HttpInfo{
			Port: gConfig.Cache.Port,
		},
	})

	// 获取客户端
	client := cache.GetClient()

	// 返回客户端对象
	return client
}

// GetMd5 获取数据的MD5值
func (s *Session) GetMd5(data []byte) string {
	has := md5.Sum(data)
	result := fmt.Sprintf("%x", has) //将[]byte转成16进制
	return result
}
