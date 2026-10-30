package handler

import (
	"ai-inference-gateway/internal/metrics"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// ChatRequest 这里的结构体需要与 python 的 GenRequest对应
// 用来接收客户端传来的 JSON
// Field names/tags must mirror Python's GenRequest to keep the wire format in sync.
type ChatRequest struct {
	Prompt string `json:"prompt"`
}

// ChatHandler 每一个错误客户端和服务端两端都要记录到
// 客户端模糊，服务端详尽
// Client-facing message stays vague; server log carries the full error detail.
func ChatHandler(c *gin.Context) {
	// 定义一个跟 python 结构体相同的变量
	myRequest := ChatRequest{}

	// 获取客户端发来的 json 格式请求并解析到 myRequest 上
	if err := c.ShouldBindJSON(&myRequest); err != nil {
		slog.Error("failed to parse JSON request body", "error", err)
		c.JSON(http.StatusBadRequest, "Bad Request")
		return
	}

	rawReqID, exist := c.Get("req_id")
	var ReqID string
	if exist {
		ReqID = rawReqID.(string)
	}

	job := Job{Prompt: myRequest.Prompt, RequestID: ReqID, ResultChan: make(chan JobResult, 1), Ctx: c.Request.Context()}

	// 发送 job 给 jobQueue
	select {
	case JobQueue <- job:
		metrics.AddQueueSize()
	case <-c.Request.Context().Done():
		slog.Warn("client disconnected before enqueue")
		c.AbortWithStatus(499) // 用来表示“在服务器发送响应之前，客户端主动关闭了连接”。
		return
	}

	// 设置超时
	ctx, cancel := context.WithTimeout(c.Request.Context(), 25*time.Second)
	defer cancel()

	// 收结果
	select {
	case reply := <-job.ResultChan:
		if reply.Err != nil {
			metrics.RecordJobOutcome("failed")
			slog.Error("wrong error", "reply_error", reply.Err)
			c.JSON(http.StatusBadGateway, gin.H{"error": "internal server error"})
		} else {
			metrics.RecordJobOutcome("delivered")
			c.JSON(http.StatusOK, gin.H{"reply": reply.Reply})
		}
	case <-ctx.Done():
		// ctx 是 c.Request.Context() 派生出来的，这里先要判断特异性，再判断通用性
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			slog.Warn("Timeout")
			metrics.RecordJobOutcome("timeout")
			c.AbortWithStatus(http.StatusGatewayTimeout)
		} else if errors.Is(c.Request.Context().Err(), context.Canceled) {
			slog.Warn("client disconnected while waiting for result")
			metrics.RecordJobOutcome("cancelled")
			c.AbortWithStatus(499)
		}
	}
}
