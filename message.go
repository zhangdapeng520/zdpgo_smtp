package zdpgo_smtp

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

/*
@Time : 2022/6/6 10:43
@Author : 张大鹏
@File : message.go
@Software: Goland2021.3.1
@Description:
*/

type Message struct {
	From    string `json:"from"`
	To      string `json:"to"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
	Time    int    `json:"time"`
}

// ToReader 转换为读取流
func (m *Message) ToReader() *strings.Reader {
	data := fmt.Sprintf(
		"To: %s\r\nSubject: %s\r\n\r\n%s\r\n",
		m.To, m.Subject, m.Body,
	)
	return strings.NewReader(data)
}

// ParseString 解析字符串
func (m *Message) ParseString(data string) error {
	// 拆分字符串
	dataArr := strings.Split(data, "\r\n")
	if len(dataArr) < 3 {
		Log.Error("拆分字符串失败", "dataArr", dataArr)
		return errors.New("数据格式错误")
	}

	// 提取字符串
	m.To = strings.Replace(dataArr[0], "To: ", "", 1)
	m.Subject = strings.Replace(dataArr[1], "Subject: ", "", 1)
	m.Body = strings.TrimSpace(dataArr[3])
	m.Time = int(time.Now().Unix())

	// 返回
	return nil
}
