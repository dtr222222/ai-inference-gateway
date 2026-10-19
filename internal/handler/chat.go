package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// 用来限制并发数
var sem = make(chan struct{}, 3)

// 这里的结构体需要与 python 的 GenRequest对应
// 用来接收客户端传来的 JSON
type ChatRequest struct {
	Prompt string `json:"prompt"`
}

type PythonResponse struct {
	Result []struct {
		GeneratedText string `json:"generated_text"`
	} `json:"result"`
}

// 每一个错误客户端和服务端两端都要记录到
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

	// 处理成 json 格式，并利用 NewBuffer 来变成 io.reader，使能够做好下一步发送请求的格式
	jsonData, err := json.Marshal(myRequest)
	if err != nil {
		slog.Error("转换到 JSON 格式失败", "error", err)
		c.JSON(http.StatusInternalServerError, "Internal Server Error")
		return
	}
	reader := bytes.NewBuffer(jsonData)

	select {
	case <-time.After(1 * time.Second):
		slog.Info("请求太多")
		c.JSON(429, "Too Many Requests")
		return
	case <-c.Request.Context().Done():
		slog.Info("用户自行断开连接")
		return
	case sem <- struct{}{}:
		// 这里这个 case 的作用是抢到了通道就往下走
		// 没抢到的就直接返回
		// 不要把剩下的代码全都塞进这里
		// 这在工程上叫 “箭头型代码”（严重向右缩进），非常难读。
	}
	// 请求完就释放
	defer func() { <-sem }()
	// 定义超时上下文
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	// 创建请求
	req, err := http.NewRequestWithContext(ctx, "POST", "http://python_service:8000/prompt", reader)
	if err != nil {
		slog.Error("初始化请求失败", "error", err)
		c.JSON(http.StatusInternalServerError, "Internal Server Error")
		return
	}

	// 给 python 发暗号
	// Python 的 FastAPI 默认只吃 application/json。
	// 如果不加这行头，它会认为你发的是纯文本，直接给你返回一个 422 Unprocessable Entity。
	req.Header.Set("Content-Type", "application/json")
	// 发送请求
	resp, err := http.DefaultClient.Do(req)
	if errors.Is(err, context.DeadlineExceeded) {
		slog.Error("调用 python 服务超时", "error", err)
		c.JSON(http.StatusGatewayTimeout, "Timeout")
		return
	} else if err != nil {
		slog.Error("调用 python 服务失败", "error", err)
		c.JSON(http.StatusBadGateway, "Bad Gateway")
		return
	}
	defer resp.Body.Close()

	// 从 resp.Body 里读取 python 后端回复的内容
	reply, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.Error("加载回复失败", "error", err)
		c.JSON(http.StatusBadGateway, "Bad Gateway")
		return
	}

	var pyResp PythonResponse
	// 把 reply 里的内容用 pyResp 结构体接住
	// 这里要注意：PythonResponse的结构体的内容跟 python 后端返回的格式相同
	// 因此才能存在 pyResp 中
	err = json.Unmarshal(reply, &pyResp)
	if err != nil {
		slog.Error("反 JSON 格式解析失败", "error", err)
		c.JSON(http.StatusBadGateway, "Bad Gateway")
		return
	}

	if len(pyResp.Result) == 0 {
		slog.Error("Python 返回了空结果")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "模型未返回结果"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"reply": pyResp.Result[0].GeneratedText})
}
