package zdpgo_smtp

/*
@Time : 2022/6/6 10:26
@Author : 张大鹏
@File : config.go
@Software: Goland2021.3.1
@Description:
*/

type Config struct {
	Debug       bool            `yaml:"debug" json:"debug"`
	LogFilePath string          `yaml:"log_file_path" json:"log_file_path"`
	Domain      string          `yaml:"domain" json:"domain"`
	Host        string          `yaml:"host" json:"host"`
	Port        int             `yaml:"port" json:"port"`
	Auths       map[string]Auth `yaml:"auths" json:"auths"`
	Cache       CacheConfig     `yaml:"cache" json:"cache"`
}

type CacheConfig struct {
	Host string `yaml:"host" json:"host"`
	Port int    `yaml:"port" json:"port"`
}

type Auth struct {
	Username string `yaml:"username" json:"username"`
	Password string `yaml:"password" json:"password"`
}

type ServerInfo struct {
	Domain string `yaml:"domain" json:"domain"`
	Host   string `yaml:"host" json:"host"`
	Port   int    `yaml:"port" json:"port"`
}
