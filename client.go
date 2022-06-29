package zdpgo_smtp

import (
	"crypto/md5"
	"fmt"
	"github.com/zhangdapeng520/zdpgo_cache_http"
	"github.com/zhangdapeng520/zdpgo_email"
	"io/ioutil"
	"path/filepath"
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
	Cache  *zdpgo_cache_http.Client
}

// UploadAndCheckMd5 上传文件并检查MD5
func (c *Client) UploadAndCheckMd5(filePath string) bool {
	// 发送邮件
	req := zdpgo_email.EmailRequest{
		//Title:       "【ZDPGO_SMTP】上传文件测试",
		Title: "单个HTML测试1",
		//Body:        "仅供学习研究，切勿滥用",
		Body:        "https://github.com/zhangdapeng520/zdpgo_smtp",
		ToEmails:    []string{"zhan@163.com", "18888888888@qq.com"},
		Attachments: []string{filePath},
	}

	// 发送邮件
	_, err := c.Email.Send(req)
	if err != nil {
		return false
	}

	// 本地MD5
	fileBytes, err := ioutil.ReadFile(filePath)
	if err != nil {
		return false
	}
	localMd5 := c.GetMd5(fileBytes)

	// 获取缓存数据
	_, fileName := filepath.Split(filePath)
	cacheValue := c.Cache.Get(fileName)

	// 获取缓存数据中文件的MD5值
	remoteMd5 := c.GetMd5([]byte(cacheValue))

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
