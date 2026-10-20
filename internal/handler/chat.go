package handler

import (
	"log/slog"
	"net/http"

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

	job := Job{Prompt: myRequest.Prompt, ResultChan: make(chan string, 1)}
	// 这里如果队列满了会出现问题，这一步会一直卡在这里，执行不到客户端主动断开连接的语句
	go func() { JobQueue <- job }()

	select {
	case JobQueue <- job:
	case <-c.Request.Context().Done():
		slog.Warn("client disconnected before enqueue")
		return
	}

	// 这里还有 bug，孤儿请求白白烧硬件
	select {
	case reply := <-job.ResultChan:
		c.JSON(http.StatusOK, gin.H{"reply": reply})
	case <-c.Request.Context().Done():
		slog.Warn("client disconnected while waiting for result")
	}
}
