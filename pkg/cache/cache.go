package cache

import (
	"context"
	"errors"
	"time"
)

// ErrCacheMiss is returned when a requested key is absent or has expired.
var ErrCacheMiss = errors.New("cache: key not found or expired")

// Cache defines the contract for key-value caching backends.
type Cache interface {
	// Get retrieves raw bytes for a key. Returns ErrCacheMiss if missing or expired.
	Get(ctx context.Context, key string) ([]byte, error)

	// GetString retrieves string data for a key.
	GetString(ctx context.Context, key string) (string, error)

	// Set stores value with an expiration TTL (0 = no expiration).
	Set(ctx context.Context, key string, val []byte, ttl time.Duration) error

	// SetString stores a string value with TTL.
	SetString(ctx context.Context, key string, val string, ttl time.Duration) error

	// Delete removes a key from cache.
	Delete(ctx context.Context, key string) error

	// Has reports whether a key exists and has not expired.
	Has(ctx context.Context, key string) (bool, error)

	// Clear removes all keys from the cache.
	Clear(ctx context.Context) error

	// Remember returns the cached value or computes it using fn, caching the result.
	Remember(ctx context.Context, key string, ttl time.Duration, fn func() ([]byte, error)) ([]byte, error)

	// Close releases connection pool or background resources.
	Close() error
}
