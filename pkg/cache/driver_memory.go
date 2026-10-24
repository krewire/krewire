package cache

import (
	"context"
	"sync"
	"time"
)

type memoryItem struct {
	value     []byte
	expiresAt time.Time // zero time means no expiration
}

func (i memoryItem) isExpired() bool {
	if i.expiresAt.IsZero() {
		return false
	}
	return time.Now().After(i.expiresAt)
}

// MemoryCache implements Cache in-memory with automatic TTL eviction.
type MemoryCache struct {
	mu     sync.RWMutex
	items  map[string]memoryItem
	stopCh chan struct{}
}

// NewMemoryCache creates an in-memory Cache with periodic cleanup.
func NewMemoryCache(cleanupInterval time.Duration) *MemoryCache {
	if cleanupInterval <= 0 {
		cleanupInterval = 1 * time.Minute
	}

	mc := &MemoryCache{
		items:  make(map[string]memoryItem),
		stopCh: make(chan struct{}),
	}

	go mc.cleanupLoop(cleanupInterval)
	return mc
}

func (c *MemoryCache) cleanupLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.cleanupExpired()
		case <-c.stopCh:
			return
		}
	}
}

func (c *MemoryCache) cleanupExpired() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for k, item := range c.items {
		if !item.expiresAt.IsZero() && now.After(item.expiresAt) {
			delete(c.items, k)
		}
	}
}

func (c *MemoryCache) Get(_ context.Context, key string) ([]byte, error) {
	c.mu.RLock()
	item, ok := c.items[key]
	c.mu.RUnlock()

	if !ok || item.isExpired() {
		if ok {
			c.mu.Lock()
			delete(c.items, key)
			c.mu.Unlock()
		}
		return nil, ErrCacheMiss
	}

	cp := make([]byte, len(item.value))
	copy(cp, item.value)
	return cp, nil
}

func (c *MemoryCache) GetString(ctx context.Context, key string) (string, error) {
	b, err := c.Get(ctx, key)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (c *MemoryCache) Set(_ context.Context, key string, val []byte, ttl time.Duration) error {
	var exp time.Time
	if ttl > 0 {
		exp = time.Now().Add(ttl)
	}

	cp := make([]byte, len(val))
	copy(cp, val)

	c.mu.Lock()
	defer c.mu.Unlock()

	c.items[key] = memoryItem{
		value:     cp,
		expiresAt: exp,
	}
	return nil
}

func (c *MemoryCache) SetString(ctx context.Context, key string, val string, ttl time.Duration) error {
	return c.Set(ctx, key, []byte(val), ttl)
}

func (c *MemoryCache) Delete(_ context.Context, key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.items, key)
	return nil
}

func (c *MemoryCache) Has(_ context.Context, key string) (bool, error) {
	c.mu.RLock()
	item, ok := c.items[key]
	c.mu.RUnlock()

	if !ok || item.isExpired() {
		return false, nil
	}
	return true, nil
}

func (c *MemoryCache) Clear(_ context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = make(map[string]memoryItem)
	return nil
}

func (c *MemoryCache) Remember(ctx context.Context, key string, ttl time.Duration, fn func() ([]byte, error)) ([]byte, error) {
	val, err := c.Get(ctx, key)
	if err == nil {
		return val, nil
	}
	if !errorsIsCacheMiss(err) {
		return nil, err
	}

	computed, err := fn()
	if err != nil {
		return nil, err
	}

	_ = c.Set(ctx, key, computed, ttl)
	return computed, nil
}

func (c *MemoryCache) Close() error {
	select {
	case <-c.stopCh:
		// already closed
	default:
		close(c.stopCh)
	}
	return nil
}

func errorsIsCacheMiss(err error) bool {
	return err == ErrCacheMiss
}

var _ Cache = (*MemoryCache)(nil)
