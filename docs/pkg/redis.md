# `redis`

Import: `github.com/krewire/krewire/pkg/redis`

## Purpose

A lightweight, pure-Go Redis client library adhering to the REdis Serialization Protocol (RESP). Supports Redis, KeyDB, Dragonfly, and Valkey without external dependencies. Provides unified APIs for String keys, Hashes, Lists, Sets, and Pub/Sub.

## Main API

- **Client**:
  - `NewClient(cfg Config) (*Client, error)`
  - `Client.Ping(ctx context.Context) error`
  - `Client.Close() error`
  - `Client.Do(ctx context.Context, args ...string) (interface{}, error)`
- **Strings**:
  - `Get(ctx context.Context, key string) (string, error)`
  - `Set(ctx context.Context, key, val string, ttl time.Duration) error`
  - `Del(ctx context.Context, keys ...string) (int64, error)`
  - `Exists(ctx context.Context, keys ...string) (int64, error)`
  - `Incr(ctx context.Context, key string) (int64, error)`
- **Hashes**:
  - `HGet(ctx context.Context, key, field string) (string, error)`
  - `HSet(ctx context.Context, key, field, val string) error`
  - `HGetAll(ctx context.Context, key string) (map[string]string, error)`
  - `HDel(ctx context.Context, key string, fields ...string) (int64, error)`
- **Lists & Queues**:
  - `LPush(ctx context.Context, key string, values ...string) (int64, error)`
  - `RPop(ctx context.Context, key string) (string, error)`
  - `LLen(ctx context.Context, key string) (int64, error)`
- **Pub/Sub**:
  - `Publish(ctx context.Context, channel, message string) (int64, error)`

## Example

```go
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/krewire/krewire/pkg/redis"
)

func main() {
	ctx := context.Background()

	client, err := redis.NewClient(redis.Config{
		Address: "127.0.0.1:6379",
	})
	if err != nil {
		panic(err)
	}
	defer client.Close()

	_ = client.Set(ctx, "session:123", "logged_in", 30*time.Minute)
	val, _ := client.Get(ctx, "session:123")
	fmt.Println("Session:", val)
}
```

## Security & Architecture Boundaries

- **Pure Go**: No CGO dependencies.
- **Downward Dependency Pyramid**: Depends only on Level 1 (`pkg/errs`, `pkg/model`) and standard library `net`.
