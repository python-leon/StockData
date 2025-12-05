package config

import (
	"fmt"

	"github.com/spf13/viper"
)

// Config 全局配置结构
type Config struct {
	Tushare  TushareConfig  `mapstructure:"tushare"`
	Database DatabaseConfig `mapstructure:"database"`
	Server   ServerConfig   `mapstructure:"server"`
	Fetcher  FetcherConfig  `mapstructure:"fetcher"`
	Log      LogConfig      `mapstructure:"log"`
}

// TushareConfig Tushare API 配置
type TushareConfig struct {
	Token   string `mapstructure:"token"`
	BaseURL string `mapstructure:"base_url"`
	Timeout int    `mapstructure:"timeout"`
	Retry   int    `mapstructure:"retry"`
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Type            string `mapstructure:"type"`
	Host            string `mapstructure:"host"`
	Port            int    `mapstructure:"port"`
	User            string `mapstructure:"user"`
	Password        string `mapstructure:"password"`
	DBName          string `mapstructure:"dbname"`
	MaxOpenConns    int    `mapstructure:"max_open_conns"`
	MaxIdleConns    int    `mapstructure:"max_idle_conns"`
	ConnMaxLifetime int    `mapstructure:"conn_max_lifetime"`
}

// ServerConfig 服务配置
type ServerConfig struct {
	Port int    `mapstructure:"port"`
	Mode string `mapstructure:"mode"`
}

// FetcherConfig 数据抓取配置
type FetcherConfig struct {
	Concurrency int    `mapstructure:"concurrency"`
	BatchSize   int    `mapstructure:"batch_size"`
	RateLimit   int    `mapstructure:"rate_limit"`
	StartDate   string `mapstructure:"start_date"`
	EndDate     string `mapstructure:"end_date"`
}

// LogConfig 日志配置
type LogConfig struct {
	Level      string `mapstructure:"level"`
	File       string `mapstructure:"file"`
	MaxSize    int    `mapstructure:"max_size"`
	MaxBackups int    `mapstructure:"max_backups"`
	MaxAge     int    `mapstructure:"max_age"`
	Compress   bool   `mapstructure:"compress"`
}

var GlobalConfig *Config

// LoadConfig 加载配置文件
func LoadConfig(configPath string) (*Config, error) {
	viper.SetConfigFile(configPath)
	viper.SetConfigType("yaml")

	// 读取配置文件
	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	// 解析配置
	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	// 验证配置
	if err := validateConfig(&config); err != nil {
		return nil, err
	}

	GlobalConfig = &config
	return &config, nil
}

// validateConfig 验证配置
func validateConfig(config *Config) error {
	if config.Tushare.Token == "" || config.Tushare.Token == "your_tushare_token_here" {
		return fmt.Errorf("请配置有效的 Tushare Token")
	}

	if config.Database.Type != "postgres" && config.Database.Type != "mysql" {
		return fmt.Errorf("数据库类型必须是 postgres 或 mysql")
	}

	if config.Fetcher.Concurrency <= 0 {
		config.Fetcher.Concurrency = 10
	}

	if config.Fetcher.BatchSize <= 0 {
		config.Fetcher.BatchSize = 1000
	}

	return nil
}

// GetDSN 获取数据库连接字符串
func (c *DatabaseConfig) GetDSN() string {
	switch c.Type {
	case "postgres":
		return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable TimeZone=Asia/Shanghai",
			c.Host, c.Port, c.User, c.Password, c.DBName)
	case "mysql":
		return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
			c.User, c.Password, c.Host, c.Port, c.DBName)
	default:
		return ""
	}
}
