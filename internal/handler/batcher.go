package handler

import (
	"ai-inference-gateway/internal/metrics"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"time"
)

// Job 定义 Job 和队列
type Job struct {
	Prompt     string
	ResultChan chan string
}

// JobQueue 定义全局缓冲通道
var JobQueue = make(chan Job, 1000)

// BatchPyRequest 批量请求结构体
type BatchPyRequest struct {
	Prompts []string `json:"prompts"`
}

// BatchPyResponse 批量响应结构体
type BatchPyResponse struct {
	Results []string `json:"results"`
}

func processBatch(batch []Job) {
	metrics.ObserveBatchSize(len(batch))
	// 定义失败函数，简化错误处理
	notifyFail := func(errMsg string, err error) {
		if err != nil {
			slog.Error(errMsg, "error", err)
		} else {
			slog.Error(errMsg)
		}
		for _, job := range batch {
			job.ResultChan <- "错误：" + errMsg
		}
	}

	// 把字符串都传到 batchRequest 里
	var batchRequest BatchPyRequest
	for _, job := range batch {
		batchRequest.Prompts = append(batchRequest.Prompts, job.Prompt)
	}

	// JSON 化
	buf, err := json.Marshal(batchRequest)
	if err != nil {
		notifyFail("JSON 格式化失败", err)
		return
	}

	buffer := bytes.NewBuffer(buf)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	//  不要在循环里使用，而是单独开一个函数
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", "http://python_service:8000/prompt", buffer)
	if err != nil {
		notifyFail("初始化请求失败", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		notifyFail("发送请求失败", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		notifyFail("模型服务返回非 200 状态", nil)
		return
	}
	reader, err := io.ReadAll(resp.Body)
	if err != nil {
		notifyFail("读取回复失败", err)
		return
	}

	var batchResponse BatchPyResponse
	err = json.Unmarshal(reader, &batchResponse)
	if err != nil {
		notifyFail("反 JSON 格式化失败", err)
		return
	}

	for i, job := range batch {
		if i < len(batchResponse.Results) {
			job.ResultChan <- batchResponse.Results[i]
		} else {
			job.ResultChan <- "错误：未能获取有效结果"
		}
	}
}

func StartBatcher() {
	for {
		select {
		case <-time.After(100 * time.Millisecond):
			// 提取 batch
			var batch []Job
			batchSize := len(JobQueue)
			metrics.SetQueueSize(batchSize)
			// 这里不能写 for job := range JobQueue
			// 因为直到 JobQueue 通道被关闭后，循环才会结束，会一直被堵塞
			for range batchSize {
				batch = append(batch, <-JobQueue)
			}

			// 如果 batch > 0,调用独立函数来处理，防止 defer 在循环里无法退出
			if len(batch) > 0 {
				processBatch(batch)
			}
		}
	}
}
