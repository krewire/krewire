# `cache`

Import: `github.com/krewire/krewire/pkg/cache`

## Purpose

High-performance, pure-Go multi-driver caching library. Provides a unified `Cache` interface supporting Memory, File, Redis (and compatible engines such as Valkey, Dragonfly, and KeyDB), and Memcached. Implements standard caching operations, TTL-based eviction, and safe concurrency.

## Main API

- **`Cache` Interface**:
  - `Get(ctx context.Context, key string) ([]byte, error)`
  - `GetString(ctx context.Context, key string) (string, error)`
  - `Set(ctx context.Context, key string, val []byte, ttl time.Duration) error`
  - `SetString(ctx context.Context, key string, val string, ttl time.Duration) error`
  - `Delete(ctx context.Context, key string) error`
  - `Has(ctx context.Context, key string) (bool, error)`
  - `Clear(ctx context.Context) error`
  - `Remember(ctx context.Context, key string, ttl time.Duration, fn func() ([]byte, error)) ([]byte, error)`
  - `Close() error`
- **Drivers**:
  - **Memory Cache**: `NewMemoryCache(cleanupInterval time.Duration) *MemoryCache`
  - **File Cache**: `NewFileCache(cfg FileCacheConfig) (*FileCache, error)`
  - **Redis Cache**: `NewRedisCache(cfg RedisConfig) (*RedisCache, error)`
  - **Memcached Cache**: `NewMemcachedCache(cfg MemcachedConfig) (*MemcachedCache, error)`

## Example

```go
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/krewire/krewire/pkg/cache"
)

func main() {
	ctx := context.Background()

	// In-Memory cache with periodic TTL eviction
	c := cache.NewMemoryCache(1 * time.Minute)
	defer c.Close()

	// Store with TTL
	_ = c.SetString(ctx, "session:100", "user_payload", 15*time.Minute)

	// Fetch or compute pattern
	val, err := c.Remember(ctx, "top_articles", 5*time.Minute, func() ([]byte, error) {
		return []byte(`[{"id": 1, "title": "Krewire PaaS"}]`), nil
	})
	if err != nil {
		panic(err)
	}

	fmt.Println("Cached articles:", string(val))
}
```

## Security & Architecture Boundaries

- **Pure Go**: No CGO dependencies. Redis and Memcached drivers use standard TCP and native protocols (RESP and text protocol).
- **Downward Dependency Pyramid**: Depends only on Level 1 (`packages/errs`, `packages/model`) and the Go standard library.
