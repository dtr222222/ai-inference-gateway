package handler

import (
	"ai-inference-gateway/internal/metrics"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"time"
)

var URL = "http://python-service:8000/prompt"

// 定义全局客户端，避免多次创建和销毁，增加开销、
var tr = &http.Transport{
	MaxIdleConnsPerHost: 10,
	MaxIdleConns:        100,
}
var Client = &http.Client{Transport: tr}

// Job 定义 Job 和队列
// Job represents a single prompt request and the channel its result is written back to.
type Job struct {
	Prompt     string
	RequestID  string
	ResultChan chan JobResult
	Ctx        context.Context
}

type JobResult struct {
	Reply string
	Err   error
}

// JobQueue 定义全局缓冲通道
// JobQueue is a bounded buffer; writers block once it is full (backpressure).
var JobQueue = make(chan Job, 1000)

// BatchPyRequest 批量请求结构体
type BatchPyRequest struct {
	Prompts    []string `json:"prompts"`
	RequestIDs []string `json:"request_ids"`
}

// BatchPyResponse 批量响应结构体
type BatchPyResponse struct {
	Results    []string `json:"results"`
	RequestIDs []string `json:"request_ids"`
}

func processBatch(batch []Job) {
	metrics.ObserveBatchSize(len(batch))

	// 这里先过滤一下 batch，把没人要的 job 剔除
	var activeJobs []Job
	for _, job := range batch {
		if job.Ctx.Err() != nil {
			job.ResultChan <- JobResult{"error: canceled", job.Ctx.Err()}
		} else {
			activeJobs = append(activeJobs, job)
		}
	}

	// 定义失败函数，简化错误处理
	// notifyFail broadcasts the same error to every job in the batch and logs it once.
	notifyFail := func(errMsg string, err error) {
		if err != nil {
			slog.Error(errMsg, "error", err)
		} else {
			slog.Error(errMsg)
		}
		for _, job := range activeJobs {
			job.ResultChan <- JobResult{"error: " + errMsg, errors.New(errMsg)}
		}
	}
	// 把字符串都传到 batchRequest 里
	var batchRequest BatchPyRequest
	for _, job := range activeJobs {
		batchRequest.Prompts = append(batchRequest.Prompts, job.Prompt)
		batchRequest.RequestIDs = append(batchRequest.RequestIDs, job.RequestID)
	}

	if len(batchRequest.Prompts) == 0 {
		return
	}

	// JSON 化
	buf, err := json.Marshal(batchRequest)
	if err != nil {
		notifyFail("failed to marshal request to JSON", err)
		return
	}

	buffer := bytes.NewBuffer(buf)
	// 10s is the per-batch RPC budget; it must stay below the caller's HTTP timeout.
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	//  不要在循环里使用，而是单独开一个函数
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", URL, buffer)
	if err != nil {
		notifyFail("failed to build upstream request", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	// 默认的客户端在底层只允许同一个目标主机保持最多 2 个空闲长连接
	// 如果瞬间有 10 个 Batch 要发给 Python，
	// 会有 8 个 Batch 每次都要重新进行 TCP 三次握手，
	// 极大地浪费时间和端口资源。

	resp, err := Client.Do(req)
	if err != nil {
		notifyFail("failed to call model service", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		notifyFail("model service returned non-200 status", nil)
		return
	}
	reader, err := io.ReadAll(resp.Body)
	if err != nil {
		notifyFail("failed to read response body", err)
		return
	}

	var batchResponse BatchPyResponse
	err = json.Unmarshal(reader, &batchResponse)
	if err != nil {
		notifyFail("failed to unmarshal response JSON", err)
		return
	}

	for i, job := range activeJobs {

		if job.Ctx.Err() != nil {
			metrics.RecordJobOutcome("orphaned")
			continue
		}

		if i >= len(batchResponse.RequestIDs) {
			errMsg := "request id miss"
			job.ResultChan <- JobResult{"error: " + errMsg, errors.New(errMsg)}
			slog.Error("request id miss")
			continue
		}

		if batchResponse.RequestIDs[i] != job.RequestID {
			errMsg := "request id mismatch"
			job.ResultChan <- JobResult{"error: " + errMsg, errors.New(errMsg)}
			slog.Error("request id mismatch", "expected", job.RequestID, "got", batchResponse.RequestIDs[i])
			continue
		}

		if i < len(batchResponse.Results) {
			job.ResultChan <- JobResult{batchResponse.Results[i], nil}
		} else {
			errMsg := "no valid result returned"
			job.ResultChan <- JobResult{"error: " + errMsg, errors.New(errMsg)}
		}
	}
}

func collectJobs() {
	var batch []Job
Loop:
	for {
		select {
		case job := <-JobQueue:

			batch = append(batch, job)
			if len(batch) >= 50 {
				break Loop
			}
		default:
			break Loop
		}
	}

	// 如果 batch > 0,调用独立函数来处理，防止 defer 在循环里无法退出
	if len(batch) > 0 {
		metrics.SubQueueSize(len(batch))
		processBatch(batch)
	}
}

func StartBatcher(ctx context.Context) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			collectJobs()
			return
		case <-ticker.C:
			collectJobs()
		}
	}
}
