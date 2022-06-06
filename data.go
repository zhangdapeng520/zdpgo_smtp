package zdpgo_smtp

/*
@Time : 2022/6/6 10:21
@Author : 张大鹏
@File : data.go
@Software: Goland2021.3.1
@Description:
*/

type Data struct {
	From    string   `json:"from"`
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	Body    string   `json:"body"`
}
