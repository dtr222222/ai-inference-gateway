package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config 将这几个配置打包成结构体
type Config struct {
	// 要导出的变量首字母需要大写
	Timeout     int
	Concurrency int
	DSN         string
}

// Load error 类型最好和结构体分开，便于调用时处理错误
func Load() (*Config, error) {
	timeoutStr := os.Getenv("TIMEOUT_SECONDS")
	concurrencyStr := os.Getenv("CONCURRENCY")
	dsn := os.Getenv("DB_DSN")

	if timeoutStr == "" || concurrencyStr == "" || dsn == "" {
		return nil, fmt.Errorf("环境变量缺失：TIMEOUT_SECONDS=%q，CONCURRENCY=%q，DB_DSN=%q",
			timeoutStr, concurrencyStr, dsn)
	}
	timeout, err := strconv.Atoi(timeoutStr)
	if err != nil {
		return nil, fmt.Errorf("TIMEOUT_SECONDS 格式错误: %w", err)
	}
	concurrency, err := strconv.Atoi(concurrencyStr)
	if err != nil {
		return nil, fmt.Errorf("CONCURRENCY 格式错误: %w", err)
	}
	return &Config{Timeout: timeout, Concurrency: concurrency, DSN: dsn}, nil
}
