package main

import (
	"context"
	"errors"
	"log/slog"
	"my-checker/internal/config"
	h "my-checker/internal/handler"
	"my-checker/internal/middleware"
	"my-checker/internal/repository"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// 初始化 slog
	// 同时要注意应该先初始化 slog, 不然后面的 slog 的格式会有差异
	jsonHandler := slog.NewJSONHandler(os.Stdout, nil)
	logger := slog.New(jsonHandler)
	slog.SetDefault(logger)
	slog.Info("slog 配置加载成功")

	err := godotenv.Load()
	if err != nil {
		slog.Error("环境变量导入失败", "error", err)
		return
	}

	cfg, err := config.Load()
	if err != nil {
		slog.Error("环境变量加载失败", "error", err)
		return
	}

	// 初始化数据库
	db, err := repository.InitDB(cfg.DSN)
	if err != nil {
		slog.Error("初始化数据库失败", "error", err)
		os.Exit(1)
	}

	// 实例化仓库
	repo := repository.NewCheckRepository(db)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	// 设置成发布模式，减少调试信息输出
	gin.SetMode(gin.ReleaseMode)
	// gin 框架
	router := gin.New()
	// 加入中间件
	router.Use(middleware.SlogMiddleware())
	router.Use(middleware.RecoveryMiddleware())
	handler := &h.Handler{Cfg: cfg, CheckRepo: repo}

	{
		v1 := router.Group("/api/v1")
		v1.POST("/check", handler.CheckHandler)
		v1.GET("/results", handler.GetResultsHandler)
	}

	// graceful shutdown
	srv := &http.Server{
		Addr:              ":8080",
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("listen: %s\n", "error", err)
		}
	}()

	<-ctx.Done()

	slog.Info("shutting down gracefully, press Ctrl+C again to force")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err = srv.Shutdown(shutdownCtx); err != nil {
		slog.Info("Server forced to shutdown: ", "error", err)
	}
	slog.Info("Server exiting")
}
