package storage

import (
	"context"
	"strings"

	"github.com/krewire/krewire/pkg/redis"
)

// RedisKV implements the KV storage interface over a Redis client.
type RedisKV struct {
	client *redis.Client
	prefix string
}

// NewRedisKV creates a Redis-backed KV store.
func NewRedisKV(client *redis.Client, prefix string) *RedisKV {
	return &RedisKV{
		client: client,
		prefix: prefix,
	}
}

func (r *RedisKV) key(k string) string {
	clean := CleanKey(k)
	if r.prefix != "" {
		return r.prefix + ":" + clean
	}
	return clean
}

func (r *RedisKV) Get(ctx context.Context, key string) ([]byte, bool, error) {
	val, err := r.client.Get(ctx, r.key(key))
	if err != nil {
		if err == redis.ErrNil {
			return nil, false, nil
		}
		return nil, false, err
	}
	return []byte(val), true, nil
}

func (r *RedisKV) Put(ctx context.Context, key string, val []byte) error {
	return r.client.Set(ctx, r.key(key), string(val), 0)
}

func (r *RedisKV) Delete(ctx context.Context, key string) error {
	_, err := r.client.Del(ctx, r.key(key))
	return err
}

func (r *RedisKV) List(ctx context.Context, prefix string) ([]string, error) {
	searchPrefix := r.key(prefix)
	res, err := r.client.Do(ctx, "KEYS", searchPrefix+"*")
	if err != nil {
		return nil, err
	}

	arr, ok := res.([]interface{})
	if !ok {
		return nil, nil
	}

	out := make([]string, 0, len(arr))
	for _, item := range arr {
		if s, ok := item.(string); ok {
			clean := strings.TrimPrefix(s, r.prefix+":")
			out = append(out, clean)
		}
	}
	return out, nil
}

var _ KV = (*RedisKV)(nil)
