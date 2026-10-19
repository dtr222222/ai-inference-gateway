package handler

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

// ChatRequest 这里的结构体需要与 python 的 GenRequest对应
// 用来接收客户端传来的 JSON
type ChatRequest struct {
	Prompt string `json:"prompt"`
}

// ChatHandler 每一个错误客户端和服务端两端都要记录到
// 客户端模糊，服务端详尽
func ChatHandler(c *gin.Context) {
	// 定义一个跟 python 结构体相同的变量
	myRequest := ChatRequest{}

	// 获取客户端发来的 json 格式请求并解析到 myRequest 上
	if err := c.ShouldBindJSON(&myRequest); err != nil {
		slog.Error("JSON 格式解析失败", "error", err)
		c.JSON(http.StatusBadRequest, "Bad Request")
		return
	}

	job := Job{Prompt: myRequest.Prompt, ResultChan: make(chan string, 1)}
	JobQueue <- job

	select {
	case reply := <-job.ResultChan:
		c.JSON(http.StatusOK, gin.H{"reply": reply})
	case <-c.Request.Context().Done():
		slog.Warn("客户端断开连接")
	}
}
