package zdpgo_smtp

/*
@Time : 2022/6/6 10:26
@Author : 张大鹏
@File : config.go
@Software: Goland2021.3.1
@Description:
*/

type Config struct {
	Debug       bool     `yaml:"debug" json:"debug"`
	LogFilePath string   `yaml:"log_file_path" json:"log_file_path"`
	Server      HttpInfo `yaml:"server" json:"server"`
}

type HttpInfo struct {
	Domain string `yaml:"domain" json:"domain"`
	Host   string `yaml:"host" json:"host"`
	Port   int    `yaml:"port" json:"port"`
}