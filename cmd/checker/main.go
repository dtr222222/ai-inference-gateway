package main

import (
	"ai-inference-gateway/internal/config"
	h "ai-inference-gateway/internal/handler"
	"ai-inference-gateway/internal/metrics"
	"ai-inference-gateway/internal/middleware"
	"ai-inference-gateway/internal/repository"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	// 初始化 slog
	// 同时要注意应该先初始化 slog, 不然后面的 slog 的格式会有差异
	// Initialize slog first so every subsequent log uses the same JSON handler.
	jsonHandler := slog.NewJSONHandler(os.Stdout, nil)
	logger := slog.New(jsonHandler)
	slog.SetDefault(logger)
	slog.Info("slog configured successfully")

	// 初始化 metrics
	// Init metrics before any business logic so all code paths can record them.
	metrics.Init()

	err := godotenv.Load()
	if errors.Is(err, os.ErrNotExist) {
		// A missing .env is tolerated: in containers/prod env vars are injected externally.
		slog.Warn(".env file not found", "error", err)
	} else if err != nil {
		slog.Error("failed to load .env file", "error", err)
		return
	}

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		return
	}

	// 初始化数据库
	db, err := repository.InitDB(cfg.DSN)
	if err != nil {
		slog.Error("failed to initialize database", "error", err)
		os.Exit(1)
	}

	// 实例化仓库
	repo := repository.NewCheckRepository(db)

	// Root context canceled on SIGINT, used to trigger graceful shutdown.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	// 设置成发布模式，减少调试信息输出
	gin.SetMode(gin.ReleaseMode)
	// gin 框架
	router := gin.New()
	// 加入中间件
	// Order matters: recovery -> structured logging -> metrics.
	router.Use(middleware.RecoveryMiddleware())
	router.Use(middleware.SlogMiddleware())
	router.Use(middleware.MetricsMiddleware())
	handler := &h.Handler{Cfg: cfg, CheckRepo: repo}
	// Background batcher aggregates async results before flushing to storage.
	go h.StartBatcher()

	{
		v1 := router.Group("/api/v1")
		v1.POST("/check", handler.CheckHandler)
		v1.GET("/results", handler.GetResultsHandler)
		v1.POST("/chat", h.ChatHandler)
	}

	// 注册 metrics 路由
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))
	// graceful shutdown
	srv := &http.Server{
		// 这个地方真正决定了开放哪个端口
		// TODO: read the listen address from config instead of hardcoding it.
		Addr:              ":8080",
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("listen failed", "error", err)
		}
	}()

	// Block until SIGINT is received.
	<-ctx.Done()

	slog.Info("shutting down gracefully, press Ctrl+C again to force")

	// Give in-flight requests up to 5 seconds to finish.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err = srv.Shutdown(shutdownCtx); err != nil {
		slog.Info("server forced to shutdown", "error", err)
	}
	slog.Info("server exiting")
}
