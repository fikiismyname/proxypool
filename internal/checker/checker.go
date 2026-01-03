package checker

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"proxypool/internal/model"
)

type CheckResult struct {
	Alive     bool
	LatencyMS int
	Country   string // Placeholder for GeoIP
	Anonymity string // Placeholder for header analysis
}

type Checker struct {
	TargetURL string
	Timeout   time.Duration
}

func NewChecker(targetURL string, timeout time.Duration) *Checker {
	return &Checker{
		TargetURL: targetURL,
		Timeout:   timeout,
	}
}

// Check validates the proxy by attempting to make a request to the target URL.
func (c *Checker) Check(ctx context.Context, p *model.Proxy) (*CheckResult, error) {
	// Construct Proxy URL
	// Assume HTTP for simplicity if not set, or handle SOCKS later
	// For this task, we assume HTTP/HTTPS proxies.
	proxyStr := fmt.Sprintf("http://%s:%d", p.IP, p.Port)
	if p.Protocol != "" {
		proxyStr = fmt.Sprintf("%s://%s:%d", p.Protocol, p.IP, p.Port)
	}

	proxyURL, err := url.Parse(proxyStr)
	if err != nil {
		return nil, fmt.Errorf("invalid proxy url: %w", err)
	}

	transport := &http.Transport{
		Proxy: http.ProxyURL(proxyURL),
		// Disable KeepAlives for checkers to save resources
		DisableKeepAlives: true,
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   c.Timeout,
	}

	start := time.Now()
	
	// Create a new context with timeout for the request
	reqCtx, cancel := context.WithTimeout(ctx, c.Timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, "HEAD", c.TargetURL, nil)
	if err != nil {
		return nil, fmt.Errorf("bad request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		// Connection failed
		return &CheckResult{Alive: false}, nil 
		// Note: We return Alive: false instead of error to indicate "checked but failed"
	}
	defer resp.Body.Close()

	Latency := time.Since(start).Milliseconds()

	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		return &CheckResult{
			Alive:     true,
			LatencyMS: int(Latency),
			// Anonymity detection requires inspecting returned headers (e.g. from httpbin), HEAD doesn't show body.
			// For basic liveness, this is enough.
		}, nil
	}

	return &CheckResult{Alive: false}, nil
}
