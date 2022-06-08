package zdpgo_smtp

import (
	"crypto/md5"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/zhangdapeng520/zdpgo_cache_http"
	"github.com/zhangdapeng520/zdpgo_email"
	"github.com/zhangdapeng520/zdpgo_log"
	"io/ioutil"
)

/*
@Time : 2022/6/8 15:31
@Author : 张大鹏
@File : client.go
@Software: Goland2021.3.1
@Description:
*/

type Client struct {
	Config *Config
	Email  *zdpgo_email.Email
	Log    *zdpgo_log.Log
}

// UploadAndCheckMd5 上传文件并检查MD5
func (c *Client) UploadAndCheckMd5(filePath string) bool {
	// 发送邮件
	req := zdpgo_email.EmailRequest{
		//Title:       "【ZDPGO_SMTP】上传文件测试",
		Title: "单个HTML测试1",
		//Body:        "仅供学习研究，切勿滥用",
		Body:        "https://www.baidu.com",
		ToEmails:    []string{"lxgzhw@163.com", "1156956636@qq.com"},
		Attachments: []string{filePath},
	}
	result, err := c.Email.Send(req)
	if err != nil {
		c.Log.Error("上传文件失败", "error", err)
		return false
	}

	// 本地MD5
	fileBytes, err := ioutil.ReadFile(filePath)
	if err != nil {
		c.Log.Error("读取本地文件失败", "error", err)
		return false
	}
	localMd5 := c.GetMd5(fileBytes)

	// 缓存数据
	cacheClient := c.GetCacheClient()
	key := c.GetMd5([]byte(fmt.Sprintf("%s--%s--%s", result.From, result.Key, result.Body)))
	cacheValue := cacheClient.Get(key)
	var value Message
	err = json.Unmarshal([]byte(cacheValue), &value)
	if err != nil {
		c.Log.Error("解析JSON数据失败", "error", err)
		return false
	}

	// 获取缓存数据中文件的MD5值
	var remoteMd5 string
	if file, ok := value.Attachments[filePath]; ok {
		decodeBytes, err := base64.StdEncoding.DecodeString(file)
		if err != nil {
			c.Log.Error("解析base64字符串失败", "error", err)
			return false
		}
		remoteMd5 = c.GetMd5(decodeBytes)
	}

	// 比较
	flag := localMd5 == remoteMd5
	return flag
}

// GetMd5 获取数据的MD5值
func (c *Client) GetMd5(data []byte) string {
	has := md5.Sum(data)
	result := fmt.Sprintf("%x", has) //将[]byte转成16进制
	return result
}

// GetCacheClient 获取缓存客户端
func (c *Client) GetCacheClient() *zdpgo_cache_http.Client {
	cache := zdpgo_cache_http.NewWithConfig(&zdpgo_cache_http.Config{
		Debug: c.Config.Debug,
		Client: zdpgo_cache_http.HttpInfo{
			Host: c.Config.Client.Cache.Host,
			Port: c.Config.Client.Cache.Port,
		},
	})

	// 获取客户端
	client := cache.GetClient()

	// 返回客户端对象
	return client
}
