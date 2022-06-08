package zdpgo_smtp

import (
	"errors"
	"fmt"
	"mime"
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
	From    string   `json:"from"`
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	Body    string   `json:"body"`
	Time    int      `json:"time"`
	Author  string   `json:"author"` // zdpgo_email发过来的唯一标识
}

// ToReader 转换为读取流
func (m *Message) ToReader() *strings.Reader {
	data := fmt.Sprintf(
		"To: %s\r\nSubject: %s\r\n\r\n%s\r\n",
		strings.Join(m.To, ","), m.Subject, m.Body,
	)
	return strings.NewReader(data)
}

// ParseString 解析字符串
func (m *Message) ParseString(data string) error {
	// 拆分字符串
	dataArr := strings.Split(data, "\r\n\r\n")
	if len(dataArr) < 2 {
		Log.Error("拆分字符串失败", "dataArr", dataArr)
		return errors.New("数据格式错误")
	}

	// 处理请求头
	fmt.Println("==============开始=======================")
	fmt.Println(dataArr[0])
	headerArr := strings.Split(dataArr[0], "\r\n")
	for _, v := range headerArr {
		if strings.HasPrefix(v, "To:") {
			// 处理消息收件人
			to := strings.Replace(v, "To: ", "", 1)
			toArr := strings.Split(to, ",")
			var toResult []string
			for _, toTemp := range toArr {
				toResult = append(toResult, strings.TrimSpace(toTemp))
			}
			m.To = toResult
		} else if strings.HasPrefix(v, "X-ZdpgoEmail-Auther") {
			// 处理zdpgo_email作者
			author := strings.Replace(v, "X-ZdpgoEmail-Auther: ", "", 1)
			m.Author = strings.TrimSpace(author)
		} else if strings.HasPrefix(v, "X-ZdpgoSmtp-Auther") {
			// 处理zdpgo_smtp作者
			author := strings.Replace(v, "X-ZdpgoSmtp-Auther: ", "", 1)
			m.Author = strings.TrimSpace(author)
		} else if strings.HasPrefix(v, "Subject: ") {
			// 处理标题
			subject := strings.Replace(v, "Subject: ", "", 1)
			title, err := m.ParseTitle(strings.TrimSpace(subject))
			if err != nil {
				return err
			}
			m.Subject = title
		}
		fmt.Println("请求头：", v)
	}
	fmt.Println("==============内容=======================")
	fmt.Println(dataArr[1])
	m.Body = strings.TrimSpace(dataArr[1])
	fmt.Println("==============结束=======================")

	//// 提取字符串
	//m.To = strings.Replace(dataArr[0], "To: ", "", 1)
	//m.Subject = strings.Replace(dataArr[1], "Subject: ", "", 1)
	//m.Body = strings.TrimSpace(dataArr[3])
	m.Time = int(time.Now().Unix())

	// 返回
	return nil
}

// ParseTitle 解析邮件标题
func (m *Message) ParseTitle(title string) (string, error) {
	dec := new(mime.WordDecoder)
	result, err := dec.Decode(title)
	if err != nil {
		Log.Error("解析标题失败", "error", err)
		return "", err
	}
	return result, nil
}
