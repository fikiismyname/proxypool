package sources

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGithubRawSource_Fetch(t *testing.T) {
	// Mock Server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "1.1.1.1:8080")
		fmt.Fprintln(w, "2.2.2.2:9000")
		fmt.Fprintln(w, "invalid_line")
		fmt.Fprintln(w, "# comment")
	}))
	defer ts.Close()

	source := NewGithubRawSource("test_source", ts.URL, "http")

	proxies, err := source.Fetch(context.Background())
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}

	if len(proxies) != 2 {
		t.Errorf("Expected 2 proxies, got %d", len(proxies))
	}

	if proxies[0].IP != "1.1.1.1" || proxies[0].Port != 8080 {
		t.Errorf("Unexpected proxy 0: %v", proxies[0])
	}
	if proxies[1].IP != "2.2.2.2" || proxies[1].Port != 9000 {
		t.Errorf("Unexpected proxy 1: %v", proxies[1])
	}
}
