package zdpgo_smtp

import (
	"encoding/base64"
	"errors"
	"mime"
	"regexp"
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
	From        string            `json:"from"`
	To          []string          `json:"to"`
	Subject     string            `json:"subject"`
	Body        string            `json:"body"`
	Time        int               `json:"time"`
	Author      string            `json:"author"`      // zdpgo_email发过来的唯一标识
	Attachments map[string]string `json:"attachments"` // 文件名：文件内容的base64字符串
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
	m.Time = int(time.Now().Unix())
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
			// 处理作者
			author := strings.Replace(v, "X-ZdpgoEmail-Auther: ", "", 1)
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
	}

	// 内容：用--切割，第一部分不为空字符串，且第二部分能找不到文件名
	var (
		fileName string
		content  []byte
		err      error
	)
	for i := 1; i < len(dataArr); i++ {
		dataStr := dataArr[i]
		newDataArr := strings.Split(dataStr, "--")

		// 无意义的内容
		if newDataArr[0] == "" {
			continue
		}

		// 提取内容和附件
		if len(newDataArr) >= 2 {
			// 找文件内容
			if fileName != "" {
				content, err = m.ParseFileContent(newDataArr[0])
				if err != nil {
					continue
				}
				if m.Attachments == nil {
					m.Attachments = make(map[string]string)
				}
				m.Attachments[fileName] = base64.StdEncoding.EncodeToString(content)
			}

			// 找文件名
			fileName, err = m.ParseFileName(newDataArr[1])
			if newDataArr[0] != "" && m.Body == "" {
				m.Body = strings.TrimSpace(newDataArr[0])
				continue
			}
		}
	}

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

// ParseFileName 解析文件名
func (m *Message) ParseFileName(dataStr string) (string, error) {
	regexCompile := regexp.MustCompile(`.*?filename="(.*?)".*?`)
	results := regexCompile.FindStringSubmatch(dataStr)
	if len(results) < 2 {
		return "", errors.New("提取文件名失败")
	}
	return results[1], nil
}

// ParseFileContent 解析文件内容
func (m *Message) ParseFileContent(dataStr string) ([]byte, error) {
	// 分割数据内容
	if strings.Contains(dataStr, "--") {
		dataArr := strings.Split(dataStr, "--")
		dataStr = dataArr[0]
	}

	// 提取数据内容
	decodeBytes, err := base64.StdEncoding.DecodeString(strings.TrimSpace(dataStr))
	if err != nil {
		Log.Error("base64解码文件内容失败", "error", err)
		return nil, err
	}

	// 返回
	return decodeBytes, nil
}
