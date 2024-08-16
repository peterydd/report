package config

import (
	"fmt"
	"github.com/spf13/viper"
)

// database 配置
type database struct {
	Driver int
	Source string
}

// smtp 配置
type smtp struct {
	Host     string
	Port     string
	Username string
	Password string
}

// reports 配置
type reports struct {
	Name     string
	WorkBook *workBook
	Sheets   []*sheet
	Message  *message
}

// workBook 配置
type workBook struct {
	Prefix     string
	DateFormat string
	Suffix     string
}

// sheet 配置
type sheet struct {
	Name           string
	Sql            string
	Column         string
	IsSum          bool
	SumBeginColumn int
}

// message 配置
type message struct {
	From        string
	To          []string
	Cc          []string
	Bcc         []string
	Subject     string
	Body        string
	ContentType string
	Attachment  *attach
}

// attach 配置
type attach struct {
	ContentType string
	WithFile    bool
}

// Config 配置数据结构体
type Config struct {
	Database *database
	Smtp     *smtp
	Reports  []*reports
}

// NewConfig 创建配置实例
func NewConfig() (conf *Config, err error) {
	// 加载或创建配置
	// 导入配置文件
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("/")
	viper.AddConfigPath("./")
	viper.AddConfigPath("./configs")
	viper.AddConfigPath("../configs")
	viper.AddConfigPath("../../configs")
	// 读取配置文件
	err = viper.ReadInConfig()
	if err != nil {
		fmt.Println("读取不到配置文件：", err.Error())
		return nil, err
	}

	err = viper.Unmarshal(&conf)
	if err != nil {
		fmt.Println("导出数据有误", err.Error())
		return nil, err
	}
	return
}
