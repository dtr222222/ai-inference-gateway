package checker

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCheckURL(t *testing.T) {
	ctx := context.Background()
	url := "https://www.docker.com/"
	timeout := 60 * time.Second

	result := checkURL(ctx, url, timeout)
	if result.URL != url {
		t.Errorf("url is wrong")
	}

	if result.Status != "200 OK" {
		t.Errorf("status is wrong")
	}

	if result.Err != nil {
		t.Errorf("err")
	}

	if result.Success != true {
		t.Errorf("success is false")
	}

}

func TestCheckFakeHttp(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	defer server.Close()

	result := checkURL(context.Background(), server.URL, 2*time.Second)

	if !result.Success || result.Status != "200 OK" {
		t.Errorf("期望 200 OK，实际得到 %v", result.Status)
	}
}
