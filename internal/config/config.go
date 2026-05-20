package config

import (
	"os"
)

type Config struct {
	ServerPort  string
	DatabasePath string
	WeChatToken  string
	WeChatSecret string
	NotifyInterval int // 分钟
}

func Load() *Config {
	return &Config{
		ServerPort:     getEnv("SERVER_PORT", "38472"),
		DatabasePath:    getEnv("DATABASE_PATH", "./data/price-monitor.db"),
		WeChatToken:    getEnv("WECHAT_TOKEN", ""),
		WeChatSecret:   getEnv("WECHAT_SECRET", ""),
		NotifyInterval: 60, // 默认每小时检查一次
	}
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}