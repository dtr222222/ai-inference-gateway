package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config 将这几个配置打包成结构体
type Config struct {
	// 要导出的变量首字母需要大写
	// Fields must be exported (capitalized) so callers in other packages can read them.
	Timeout     int
	Concurrency int
	DSN         string
}

// Load error 类型最好和结构体分开，便于调用时处理错误
// Load reads config from environment variables.
// Keeping the error return separate from the struct lets callers handle failures explicitly.
func Load() (*Config, error) {
	timeoutStr := os.Getenv("TIMEOUT_SECONDS")
	concurrencyStr := os.Getenv("CONCURRENCY")
	dsn := os.Getenv("DB_DSN")

	// Fail fast with all missing vars listed at once, so ops can fix them in one pass.
	if timeoutStr == "" || concurrencyStr == "" || dsn == "" {
		return nil, fmt.Errorf("missing required env vars: TIMEOUT_SECONDS=%q, CONCURRENCY=%q, DB_DSN=%q",
			timeoutStr, concurrencyStr, dsn)
	}
	timeout, err := strconv.Atoi(timeoutStr)
	if err != nil {
		return nil, fmt.Errorf("invalid TIMEOUT_SECONDS: %w", err)
	}
	concurrency, err := strconv.Atoi(concurrencyStr)
	if err != nil {
		return nil, fmt.Errorf("invalid CONCURRENCY: %w", err)
	}
	return &Config{Timeout: timeout, Concurrency: concurrency, DSN: dsn}, nil
}
