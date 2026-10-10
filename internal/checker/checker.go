package checker

import (
	"context"
	"errors"
	"my-checker/internal/config"
	"net/http"
	"sync"
	"time"
)

type Result struct {
	URL      string
	Status   string
	Duration time.Duration
	Err      error
	Success  bool
}

func checkURL(ctx context.Context, url string, timeout time.Duration) Result {
	// 具体操作
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return Result{URL: url, Err: err, Success: false}
	}

	start := time.Now()
	resp, err := http.DefaultClient.Do(req)
	duration := time.Since(start)
	if errors.Is(err, context.DeadlineExceeded) {
		return Result{URL: url, Status: "timeout", Duration: duration, Err: err, Success: false}
	} else if err != nil {
		return Result{URL: url, Status: "error", Duration: duration, Err: err, Success: false}
	}
	defer resp.Body.Close()

	statusCode := resp.StatusCode
	if statusCode >= 200 && statusCode < 300 {
		return Result{URL: url, Status: resp.Status, Duration: duration, Err: nil, Success: true}
	}
	return Result{URL: url, Status: resp.Status, Duration: duration, Err: nil, Success: false}
}

func Check(ctx context.Context, cfg *config.Config) []Result {
	taskChan := make(chan string)
	resultChan := make(chan Result)
	// 对数据解包
	urls := cfg.URLs
	timeout := time.Duration(cfg.Timeout) * time.Second
	concurrency := cfg.Concurrency

	var wg sync.WaitGroup
	// 部署工人
	for range concurrency {
		wg.Go(func() {
			for {
				select {
				case task, ok := <-taskChan:
					if !ok {
						return
					}
					resultChan <- checkURL(ctx, task, timeout)
				case <-ctx.Done():
					return
				}
			}
		})
	}

	go func() {
		wg.Wait()
		close(resultChan)
	}()

	go func() {
		// 这里的 defer 得放在这里，不然被 close 后就不能被执行了
		// 必须得执行到这段代码 defer 才会触发
		defer close(taskChan)
		for _, url := range urls {
			select {
			case taskChan <- url:
			case <-ctx.Done():
				return
			}
		}
	}()

	// 这里处理结果
	resultArray := make([]Result, 0, len(urls))
	for result := range resultChan {
		resultArray = append(resultArray, result)
	}

	return resultArray
}
