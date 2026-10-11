package handler

import (
	"my-checker/internal/checker"
	"my-checker/internal/config"
	"net/http"

	"github.com/gin-gonic/gin"
)

// CheckRequest 请求结构体（定义客户端传什么）
type CheckRequest struct {
	URLs []string `json:"urls"`
}

type Handler struct{ Cfg *config.Config }

func (h *Handler) CheckHandler(c *gin.Context) {
	// 获取数据

	req := CheckRequest{}
	// ShouldBindWith 的作用是将用户传上来的 json 数据解析并绑定到 req 中
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 处理数据

	// 获取上下文
	// 来源： Go 标准库和 Gin 框架在底层传好了
	ctx := c.Request.Context()
	// 由信息重复而引起的 Check 函数的重新调整
	results := checker.Check(ctx, req.URLs, h.Cfg)
	c.JSON(http.StatusOK, results)
}
