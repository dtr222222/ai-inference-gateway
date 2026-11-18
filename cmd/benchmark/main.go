package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"
)

// TODO: 队列监测扩容

var client = &http.Client{
	Timeout: 60 * time.Second,
	Transport: &http.Transport{
		DisableKeepAlives:   true,
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 100,
	},
}

func worker(jsonData []byte, first *[]int64, end *[]int64, mu *sync.Mutex) {
	post := func() {
		start := time.Now()
		resp, err := client.Post("http://go-service:8080/api/chat", "application/json", bytes.NewBuffer(jsonData))
		if err != nil {
			slog.Error("发送请求失败", "error", err)
			return
		}
		defer resp.Body.Close()

		flag := true

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			if scanner.Err() != nil {
				break
			}
			text := scanner.Text()
			if text != "" && flag {
				flag = false
				mu.Lock()
				*first = append(*first, time.Since(start).Milliseconds())
				mu.Unlock()
			}
		}
		mu.Lock()
		*end = append(*end, time.Since(start).Milliseconds())
		mu.Unlock()
	}

	for range 5 {
		post()
	}
}

type Request struct {
	Prompt string `json:"prompt"`
}

func main() {
	var wg sync.WaitGroup
	var mu sync.Mutex

	req := Request{"写一首短诗"}

	jsonData, err := json.Marshal(req)
	if err != nil {
		fmt.Print("序列化失败")
		os.Exit(1)
	}

	var (
		first, end []int64
		avgF, avgE int64
	)

	for range 10 {
		wg.Go(func() {
			worker(jsonData, &first, &end, &mu)
		})
	}

	wg.Wait()

	for i := range len(first) {
		avgF, avgE = avgF+first[i], avgE+end[i]
	}

	fmt.Printf("平均首字延迟为：%v\n平均回复时间为：%v\n", avgF/int64(len(first)), avgE/int64(len(end)))
}
