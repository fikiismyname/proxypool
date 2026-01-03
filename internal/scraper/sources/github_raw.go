package sources

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"proxypool/internal/model"
)

// GithubRawSource scrapes proxies from a raw text file URL.
type GithubRawSource struct {
	name             string
	url              string
	protocolOverride string // If set, forces this protocol. Otherwise, tries to detect or defaults.
}

func NewGithubRawSource(name, url, protocolOverride string) *GithubRawSource {
	return &GithubRawSource{
		name:             name,
		url:              url,
		protocolOverride: protocolOverride,
	}
}

func (s *GithubRawSource) Name() string {
	return s.name
}

func (s *GithubRawSource) Fetch(ctx context.Context) ([]*model.Proxy, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", s.url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	var proxies []*model.Proxy
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		host, portStr, found := strings.Cut(line, ":")
		if !found {
			continue
		}

		port, err := strconv.Atoi(portStr)
		if err != nil {
			continue
		}

		proxies = append(proxies, &model.Proxy{
			IP:       host,
			Port:     port,
			Protocol: s.protocolOverride,
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan error: %w", err)
	}

	return proxies, nil
}
