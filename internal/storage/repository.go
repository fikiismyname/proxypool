package storage

import (
	"context"

	"proxypool/internal/model"
)

// ProxyRepository defines the methods for interacting with the proxy storage.
type ProxyRepository interface {
	// SaveBatch saves a batch of proxies. It should handle duplicates (e.g., ON CONFLICT DO NOTHING).
	SaveBatch(ctx context.Context, proxies []*model.Proxy) error

	// GetUnchecked returns a list of proxies that need to be checked.
	// This should include proxies that have never been checked OR haven't been checked in a while.
	GetProxiesToCheck(ctx context.Context, limit int) ([]*model.Proxy, error)

	// Update updates the validation status (latency, anonymity, etc.) of a proxy.
	Update(ctx context.Context, proxy *model.Proxy) error

	// UpdateBatch updates a batch of proxies.
	UpdateBatch(ctx context.Context, proxies []*model.Proxy) error
	
	// Count returns the total number of proxies.
	Count(ctx context.Context) (int64, error)
}
