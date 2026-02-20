// Package cache provides caching for upstream metadata and responses.
package cache

import (
	"context"
	"time"
)

// CacheStats holds cache statistics.
type CacheStats struct {
	Hits    int64 `json:"hits"`
	Misses  int64 `json:"misses"`
	Entries int64 `json:"entries"`
	Bytes   int64 `json:"bytes"`
}

// Cache defines the interface for caching data.
type Cache interface {
	// Get retrieves a value by key. Returns nil if not found or expired.
	Get(ctx context.Context, key string) ([]byte, error)

	// Set stores a value with the given TTL. If ttl is 0, uses the default TTL.
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error

	// Delete removes a value by key.
	Delete(ctx context.Context, key string) error

	// Exists checks if a key exists and is not expired.
	Exists(ctx context.Context, key string) (bool, error)

	// Flush removes all entries from the cache.
	Flush(ctx context.Context) error

	// Stats returns cache usage statistics.
	Stats(ctx context.Context) (*CacheStats, error)
}
