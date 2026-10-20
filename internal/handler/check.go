package handler

import (
	"ai-inference-gateway/internal/checker"
	"ai-inference-gateway/internal/config"
	"ai-inference-gateway/internal/model"
	"ai-inference-gateway/internal/repository"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// 改进：有待增加分页元数据

// CheckRequest 请求结构体（定义客户端传什么）
type CheckRequest struct {
	URLs []string `json:"urls"`
}

type Handler struct {
	Cfg       *config.Config
	CheckRepo *repository.CheckRepository
}

func (h *Handler) CheckHandler(c *gin.Context) {
	// 测试专用
	// panic("测试恢复中间件")

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
	// 遍历 results 把每个元素都变成 model.CheckResult
	var modelResults []model.CheckResult
	for _, result := range results {
		var modelResult model.CheckResult
		// 这里先处理 ErrMsg （只跟 result.Err 有关系）
		var errMsg string
		if result.Err != nil {
			errMsg = result.Err.Error()
		}

		// 然后再解析 Duration （失败只影响 Duration）
		duration, err := time.ParseDuration(result.Duration)
		if err != nil {
			slog.Warn("Duration 解析失败")
			duration = 0
		}

		modelResult = model.CheckResult{
			URL:      result.URL,
			Status:   result.Status,
			Success:  result.Success,
			Duration: int64(duration),
			ErrMsg:   errMsg,
		}

		modelResults = append(modelResults, modelResult)
	}

	err := h.CheckRepo.SaveResults(c.Request.Context(), modelResults)
	if err != nil {
		// 若是存库发生错误
		slog.Error("数据库保存失败",
			"error", err,
			"client_ip", c.ClientIP(),
			"path", c.Request.URL.Path,
			"urls_count", len(modelResults))
	}
	c.JSON(http.StatusOK, results)
}

func (h *Handler) GetResultsHandler(c *gin.Context) {
	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("pageSize", "20")

	page, errPage := strconv.Atoi(pageStr)
	pageSize, errPageSize := strconv.Atoi(pageSizeStr)
	if errPage != nil || errPageSize != nil {
		slog.Error("分页参数解析失败", "errPage", errPage, "errPageSize", errPageSize)
		// 返回客户端的信应该专业但模糊，不要返回数据库的原始错误，可能会被攻击
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid page or pageSize"})
		return
	}

	// 不要用 switch 因为会穿透
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	offset := (page - 1) * pageSize

	results, err := h.CheckRepo.GetResults(c.Request.Context(), pageSize, offset)
	if err != nil {
		slog.Error("获取数据库资源失败", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, results)
}
