package checker

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"proxypool/internal/model"
)

func TestChecker_Check(t *testing.T) {
	// 1. Target Server (The destination, e.g., google.com)
	targetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer targetServer.Close()

	// 2. Dummy Proxy Server (Acts as the proxy)
	// Standard HTTP proxying: Client sends requests to Proxy with absolute URL.
	proxyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "HEAD" && strings.TrimSuffix(r.URL.String(), "/") == strings.TrimSuffix(targetServer.URL, "/") {
			time.Sleep(1 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
			return
		}
		// Echo request for debugging
		t.Logf("Proxy received: %s %s. Expected: HEAD %s", r.Method, r.URL.String(), targetServer.URL)
		w.WriteHeader(http.StatusForbidden)
	}))
	defer proxyServer.Close()

	proxyURL, _ := url.Parse(proxyServer.URL)
	host := proxyURL.Hostname()
	port, _ := strconv.Atoi(proxyURL.Port())

	// 3. Test Proxy struct
	p := &model.Proxy{
		IP:       host,
		Port:     port,
		Protocol: "http",
	}

	c := NewChecker(targetServer.URL, 2*time.Second)
	
	result, err := c.Check(context.Background(), p)
	if err != nil {
		t.Fatalf("Check returned error: %v", err)
	}

	if !result.Alive {
		t.Errorf("Expected proxy to be alive")
	}

	if result.LatencyMS <= 0 {
		t.Errorf("Expected valid latency, got %d", result.LatencyMS)
	}
}

func TestChecker_Check_Dead(t *testing.T) {
	// Use a closed port
	p := &model.Proxy{
		IP:       "127.0.0.1",
		Port:     54321, // Unlikely to be open
		Protocol: "http",
	}

	c := NewChecker("http://google.com", 100*time.Millisecond) // Short timeout
	
	result, err := c.Check(context.Background(), p)
	if err != nil {
		t.Fatalf("Unexpected error structure (should return Alive:false, not err): %v", err)
	}

	if result.Alive {
		t.Errorf("Expected proxy to be dead")
	}
}
