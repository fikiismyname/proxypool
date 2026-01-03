package scraper

import (
	"context"

	"proxypool/internal/model"
)

// Source defines the interface that all proxy sources must implement.
type Source interface {
	// Name returns the unique name of the source.
	Name() string

	// Fetch retrieves proxies from the source.
	// It should use the context for timeout/cancellation.
	Fetch(ctx context.Context) ([]*model.Proxy, error)
}
