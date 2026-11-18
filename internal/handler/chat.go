package handler

import (
	chat "ai-inference-gateway/pb"
	"fmt"
	"io"
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

func ChatHandler(client chat.ChatServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		myRequest := ChatRequest{}

		// 获取客户端发来的 json 格式请求并解析到 myRequest 上
		if err := c.ShouldBindJSON(&myRequest); err != nil {
			slog.Error("failed to parse JSON request body", "error", err)
			c.JSON(http.StatusBadRequest, "Bad Request")
			return
		}

		// 拿 id
		rawReqID, exist := c.Get("req_id")
		var ReqID string
		if exist {
			ReqID = rawReqID.(string)
		}

		// 设置 SSE 响应头
		// 在写第一行数据之前，得告诉浏览器：这是一个流，如果没这句，浏览器会转圈圈，直到全部生成完才会返回
		// 下面的是告诉浏览器和一些代理不要缓存这个流，否则可能会导致前端一直卡住，直到断联才能看到所有回复
		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-cache")
		flusher := c.Writer.(http.Flusher)

		req := &chat.ChatRequest{
			RequestId:    ReqID,
			Prompt:       myRequest.Prompt,
			MaxNewTokens: 50,
			Temperature:  0,
		}

		streamer, err := client.StreamChat(c.Request.Context(), req)
		if err != nil {
			// 这里不知道该怎么描述错误
			slog.Error("failed to receive the response", "error", err)
			c.JSON(500, "")
			return
		}

		for {
			// 接受流
			response, err := streamer.Recv()

			// 判断回复是否结束
			if err == io.EOF {
				break
			}

			// 处理网络或服务异常
			if err != nil {
				slog.Error("error receiving stream form gRPC", "error", err)
				break
			}

			// 处理 python 端返回的异常
			if response.ErrorMsg != "" {
				slog.Error("error message returned by python", "error", response.ErrorMsg)
				break
			}

			// 构造 json 格式数据
			data := fmt.Sprintf(`{"token": "%s", "is_finish": %t}`+"\n\n", response.Token, response.IsFinish)
			// 把消息推到前端
			fmt.Fprintf(c.Writer, "data: %s", data)
			flusher.Flush()

			if response.IsFinish {
				break
			}
		}
	}
}
