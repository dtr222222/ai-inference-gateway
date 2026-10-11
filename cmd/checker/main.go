package main

import (
	"context"
	"errors"
	"log"
	"log/slog"
	"my-checker/internal/config"
	handler2 "my-checker/internal/handler"
	"my-checker/internal/middleware"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gin-gonic/gin"
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

	// 初始化 slog
	jsonHandler := slog.NewJSONHandler(os.Stdout, nil)
	logger := slog.New(jsonHandler)
	slog.SetDefault(logger)
	slog.Info("配置加载成功")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	// gin 框架
	router := gin.New()
	// 设置成发布模式，减少调试信息输出
	gin.SetMode(gin.ReleaseMode)
	// 加入中间件
	router.Use(middleware.SlogMiddleware())
	handler := &handler2.Handler{Cfg: cfg}
	router.POST("/api/v1/check", handler.CheckHandler)

	// graceful shutdown
	srv := &http.Server{
		Addr:              ":8080",
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	<-ctx.Done()

	stop()
	log.Println("shutting down gracefully, press Ctrl+C again to force")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err = srv.Shutdown(shutdownCtx); err != nil {
		log.Println("Server forced to shutdown: ", err)
	}
	log.Println("Server exiting")
}
