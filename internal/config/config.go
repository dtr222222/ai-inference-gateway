package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config 将这几个配置打包成结构体
type Config struct {
	// 要导出的变量首字母需要大写
	URLs        []string
	Timeout     int
	Concurrency int
}

// Load error 类型最好和结构体分开，便于调用时处理错误
func Load() (*Config, error) {
	urls := strings.Split(os.Getenv("TARGET_URLS"), ",")
	for i, url := range urls {
		urls[i] = strings.TrimSpace(url)
	}

	timeout, err := strconv.Atoi(os.Getenv("TIMEOUT_SECONDS"))
	if err != nil {
		return nil, fmt.Errorf("TIMEOUT_SECONDS 格式错误: %w", err)
	}
	concurrency, err := strconv.Atoi(os.Getenv("CONCURRENCY"))
	if err != nil {
		return nil, fmt.Errorf("CONCURRENCY 格式错误: %w", err)
	}

	return &Config{URLs: urls, Timeout: timeout, Concurrency: concurrency}, nil
}
