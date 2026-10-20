package checker

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestCheckURL hits a real external endpoint; keep it for manual/e2e runs,
// but note it is flaky in CI without network access.
func TestCheckURL(t *testing.T) {
	ctx := context.Background()
	url := "https://www.docker.com/"
	timeout := 60 * time.Second

	result := checkURL(ctx, url, timeout)
	if result.URL != url {
		t.Errorf("expected url %q, got %q", url, result.URL)
	}

	if result.Status != "200 OK" {
		t.Errorf("expected status %q, got %q", "200 OK", result.Status)
	}

	if result.Err != nil {
		t.Errorf("expected no error, got %v", result.Err)
	}

	if result.Success != true {
		t.Errorf("expected success=true, got %v", result.Success)
	}
}

// TestCheckFakeHttp uses an in-process server so the test is hermetic and fast.
func TestCheckFakeHttp(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	defer server.Close()

	result := checkURL(context.Background(), server.URL, 2*time.Second)

	if !result.Success || result.Status != "200 OK" {
		t.Errorf("expected 200 OK, got %v", result.Status)
	}
}
