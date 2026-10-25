package handler

import (
	"ai-inference-gateway/internal/metrics"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

// 要写一个 handler 函数
func writeJson(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"results":["fake"], "request_ids":["fake-id"]}`))
}

// 这个 TestMain 相当是用来配环境的
func TestMain(m *testing.M) {
	metrics.Init()
	// 运行下面的所有的 TestXxxx函数，并返回结果
	// 0 代表全部成功，非 0 代表有失败的测试
	code := m.Run()
	os.Exit(code)
}

// 写这个测试真是令我头疼
func TestProcessBatchIDMismatch(t *testing.T) {
	var batch []Job
	job := Job{
		Prompt:     "this is a test",
		RequestID:  "123456789",
		ResultChan: make(chan JobResult, 1),
		Ctx:        context.Background(),
	}
	batch = append(batch, job)

	// 创建一个假的服务端
	srv := httptest.NewServer(http.HandlerFunc(writeJson))
	defer srv.Close()

	// 覆盖真的 url，让 processBatch 函数访问假的 url
	URL = srv.URL + "/prompt"
	processBatch(batch)

	res := <-job.ResultChan

	// 别忘了还得先判空
	if res.Err == nil {
		t.Errorf("error 为空，错误拦截失败")
		return
	}

	if res.Reply != "error: request id mismatch" || res.Err.Error() != "request id mismatch" {
		t.Errorf("mismatch 拦截失败")
	}
}
