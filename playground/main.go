package main

import (
	"bytes"
	"net/http"
	"sync"
)

func main() {
	totalRequests := 4
	url := "http://localhost:8080/api/v1/chat"
	var wg sync.WaitGroup
	jsonPrompt := []byte(`{"prompt": "Task"}`)
	for range totalRequests {
		wg.Go(func() {
			// 初始化请求
			req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonPrompt))
			req.Header.Set("Content-Type", "application/json")
			resp, _ := http.DefaultClient.Do(req)
			print(resp.Status)
		})
	}

	wg.Wait()
	print("脚本进程结束")
}
