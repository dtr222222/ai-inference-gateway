package main

import (
	h "ai-inference-gateway/internal/handler"
	"ai-inference-gateway/internal/metrics"
	"ai-inference-gateway/internal/middleware"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
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

	//err := godotenv.Load()
	//if errors.Is(err, os.ErrNotExist) {
	//	// A missing .env is tolerated: in containers/prod env vars are injected externally.
	//	slog.Warn(".env file not found", "error", err)
	//} else if err != nil {
	//	slog.Error("failed to load .env file", "error", err)
	//	return
	//}

	//cfg, err := config.Load()
	//if err != nil {
	//	slog.Error("failed to load config", "error", err)
	//	return
	//}

	// Root context canceled on SIGINT, used to trigger graceful shutdown.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 初始化数据库（已废除）
	//db, err := repository.InitDB(cfg.DSN, ctx)
	//if err != nil {
	//	slog.Error("failed to initialize database", "error", err)
	//	os.Exit(1)
	//}
	//sqlDB, err := db.DB()
	//if err != nil {
	//	slog.Error("failed to initialize database", "error", err)
	//	os.Exit(1)
	//}

	// 实例化仓库
	// repo := repository.NewCheckRepository(db)

	// 设置成发布模式，减少调试信息输出
	gin.SetMode(gin.ReleaseMode)
	// gin 框架
	router := gin.New()
	// 加入中间件
	// 注意这里的 recovery 中间件要放在最后面，
	// 同时 slog 和 metrics 的中间件在 next 后要加 defer func(), 确保日志记录能被打印出来
	router.Use(middleware.SlogMiddleware())
	router.Use(middleware.MetricsMiddleware())
	router.Use(middleware.RecoveryMiddleware())
	// handler := &h.Handler{Cfg: cfg, CheckRepo: repo}
	// Background batcher aggregates async results before flushing to storage.
	var wg sync.WaitGroup
	wg.Go(func() {
		h.StartBatcher(ctx)
	})

	{
		v1 := router.Group("/api/v1")
		// v1.POST("/check", handler.CheckHandler)
		// v1.GET("/results", handler.GetResultsHandler)
		v1.POST("/chat", h.ChatHandler)
	}

	// 注册 metrics 路由
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))
	// graceful shutdown
	srv := &http.Server{
		// 这个地方真正决定了开放哪个端口
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
	shutdownHttpCtx, cancelHttp := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelHttp()
	if err := srv.Shutdown(shutdownHttpCtx); err != nil {
		slog.Info("server forced to shutdown", "error", err)
	}
	shutdownBatchCtx, cancelBatch := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancelBatch()
	sign := make(chan struct{})
	go func() {
		wg.Wait()
		close(sign)
	}()

	select {
	case <-shutdownBatchCtx.Done():
	case <-sign:
	}
	// sqlDB.Close()
	slog.Info("server exiting")
}
