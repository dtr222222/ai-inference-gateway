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
	"sync/atomic"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var isShuttingDown atomic.Bool

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

	// Root context canceled on SIGINT, used to trigger graceful shutdown.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 设置成发布模式，减少调试信息输出
	gin.SetMode(gin.ReleaseMode)
	// gin 框架
	router := gin.New()
	// 加入中间件
	// 注意这里的 recovery 中间件要放在最后面，
	// 同时 slog 和 metrics 的中间件在 next 后要加 defer func(), 确保日志记录能被打印出来
	router.Use(middleware.ShutdownMiddleware(&isShuttingDown))
	router.Use(middleware.SlogMiddleware())
	router.Use(middleware.MetricsMiddleware())
	router.Use(middleware.RecoveryMiddleware())
	// Background batcher aggregates async results before flushing to storage.
	var wg sync.WaitGroup
	wg.Go(func() {
		defer func() { recover() }()
		h.StartBatcher(ctx)
	})

	{
		api := router.Group("/api")
		api.POST("/chat", h.ChatHandler)
	}

	// 注册 metrics 路由，prometheus 会定时来抓取数据
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))
	router.GET("/healthz", h.HealthHandler(&isShuttingDown))
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
	isShuttingDown.Store(true)
	time.Sleep(5 * time.Second)
	// 这里睡 5s 的作用
	// 收到 SIGTERM -> 状态变更为“正在停机”。
	// 就绪探针下一次探测时，发现状态变了，返回 503。
	// K8s 接收到 503，开始更新路由表。
	// 在更新路由表延迟的几秒内，ShutdownMiddleware 拦截漏网之鱼（返回 503 让客户端重试）。
	// Sleep 时间结束，端口安全关闭（服务彻底不通），存量请求处理完，Pod 销毁。

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

	slog.Info("server exiting")
	// 关闭空闲链接
	h.Client.CloseIdleConnections()
}
