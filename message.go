package zdpgo_smtp

import (
	"fmt"
	"strings"
)

/*
@Time : 2022/6/6 10:43
@Author : 张大鹏
@File : message.go
@Software: Goland2021.3.1
@Description:
*/

type Message struct {
	To      string `json:"to"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
}

func (m *Message) ToReader() *strings.Reader {
	data := fmt.Sprintf(
		"To: %s\r\nSubject: %s\r\n\r\n%s\r\n",
		m.To, m.Subject, m.Body,
	)
	return strings.NewReader(data)
}
