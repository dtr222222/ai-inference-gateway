package main

import (
	"context"
	"log"
	"log/slog"
	"my-checker/internal/checker"
	"my-checker/internal/config"
	"os"
	"os/signal"

	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal(err)
		return
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
		return
	}

	jsonHandler := slog.NewJSONHandler(os.Stdout, nil)
	logger := slog.New(jsonHandler)
	slog.SetDefault(logger)
	slog.Info("配置加载成功")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	results := checker.Check(ctx, cfg)
	cntSuccess, cntFail := 0, 0
	for _, result := range results {
		if result.Success {
			cntSuccess++
		} else {
			cntFail++
		}
		slog.Info("summary", "url", result.URL,
			"status", result.Status, "duration", result.Duration,
			"error", result.Err, "success", result.Success)
	}
	slog.Info("检查汇总", "total", cntFail+cntSuccess, "success", cntSuccess, "fail", cntFail)
}
