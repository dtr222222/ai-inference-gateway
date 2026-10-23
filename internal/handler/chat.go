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

	job := Job{Prompt: myRequest.Prompt, RequestID: ReqID, ResultChan: make(chan string, 1), Ctx: c.Request.Context()}

	select {
	case JobQueue <- job:
		metrics.AddQueueSize()
	case <-c.Request.Context().Done():
		slog.Warn("client disconnected before enqueue")
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	select {
	case reply := <-job.ResultChan:
		metrics.RecordJobOutcome("delivered")
		c.JSON(http.StatusOK, gin.H{"reply": reply})
	case <-ctx.Done():
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
