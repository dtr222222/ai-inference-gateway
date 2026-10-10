package config

import "testing"

// 有待改成表驱动形式
func TestLoad(t *testing.T) {
	t.Setenv("TARGET_URLS", "https://github.com/, https://chat.deepseek.com/")
	t.Setenv("TIMEOUT_SECONDS", "10")
	t.Setenv("CONCURRENCY", "2")

	config, err := Load()

	switch {
	case err != nil:
		t.Errorf("fail to load the env")
	case config == nil:
		t.Errorf("some error occurred")
	case config.Timeout != 10:
		t.Errorf("timeout seconds get wrong answer: %d", config.Timeout)
	case config.Concurrency != 2:
		t.Errorf("number of concurrency get wrong answer: %d", config.Concurrency)
	}
}
