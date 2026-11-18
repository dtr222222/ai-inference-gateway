package main

import (
	h "ai-inference-gateway/internal/handler"
	"ai-inference-gateway/internal/metrics"
	"ai-inference-gateway/internal/middleware"
	chat "ai-inference-gateway/pb"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	_ "google.golang.org/grpc/balancer/roundrobin"
	"google.golang.org/grpc/credentials/insecure"
)

var isShuttingDown atomic.Bool

func main() {
	// 初始化 slog
	// 同时要注意应该先初始化 slog, 不然后面的 slog 的格式会有差异
	jsonHandler := slog.NewJSONHandler(os.Stdout, nil)
	logger := slog.New(jsonHandler)
	slog.SetDefault(logger)
	slog.Info("slog configured successfully")

	// 导入环境变量
	rawVolume := os.Getenv("VOLUME")
	rawRate := os.Getenv("RATE")
	rawWindow := os.Getenv("WINDOW")
	redisAddr := os.Getenv("REDIS_ADDR")

	volume, errVolume := strconv.Atoi(rawVolume)
	rate, errRate := strconv.Atoi(rawRate)
	window, errWindow := strconv.Atoi(rawWindow)
	if errVolume != nil || errRate != nil || errWindow != nil || redisAddr == "" {
		slog.Error("getting environmental variables failed", "errVolume", errVolume,
			"errRate", errRate, "errWindow", errWindow, "errRedisAddr", "fail to load redis address")
		volume, rate, window, redisAddr = 100, 20, 120, "localhost:6379"
	}

	// 初始化 metrics
	metrics.Init()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 初始化 Redis 客户端
	rdb := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		PoolSize: 20,
	})

	// 建立与 grpc的连接，最后的参数表明是明文通信
	conn, err := grpc.NewClient("dns:///python-service:50051",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy":"round_robin"}`))
	if err != nil {
		slog.Error("failed to connect with grpc", "error", err)
		os.Exit(1)
	}
	defer conn.Close()

	// 建立与 python 端通信的客户端
	client := chat.NewChatServiceClient(conn)

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
	router.Use(middleware.RateLimitBucketMiddleWare(rdb, volume, rate, window))
	router.Use(middleware.RecoveryMiddleware())

	{
		api := router.Group("/api")
		api.POST("/chat", h.ChatHandler(client))
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

	slog.Info("server exiting")
	rdb.Close()
}
