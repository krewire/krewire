package cache

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

// RedisConfig contains connection and authentication options for Redis/Valkey/Dragonfly.
type RedisConfig struct {
	Address  string // "host:port", default: "127.0.0.1:6379"
	Password string
	DB       int // database index, default: 0
	Timeout  time.Duration
}

// RedisCache implements Cache over Redis/RESP protocol in pure Go.
type RedisCache struct {
	mu      sync.Mutex
	cfg     RedisConfig
	conn    net.Conn
	reader  *bufio.Reader
	timeout time.Duration
}

// NewRedisCache creates a new Redis Cache client.
func NewRedisCache(cfg RedisConfig) (*RedisCache, error) {
	if cfg.Address == "" {
		cfg.Address = "127.0.0.1:6379"
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 5 * time.Second
	}

	rc := &RedisCache{
		cfg:     cfg,
		timeout: cfg.Timeout,
	}

	return rc, nil
}

func (r *RedisCache) getConnection() (net.Conn, *bufio.Reader, error) {
	if r.conn != nil {
		return r.conn, r.reader, nil
	}

	conn, err := net.DialTimeout("tcp", r.cfg.Address, r.timeout)
	if err != nil {
		return nil, nil, fmt.Errorf("cache: redis connect to %s failed: %w", r.cfg.Address, err)
	}

	r.conn = conn
	r.reader = bufio.NewReader(conn)

	// Auth if password provided
	if r.cfg.Password != "" {
		if _, err := r.execCommand("AUTH", r.cfg.Password); err != nil {
			r.closeConn()
			return nil, nil, fmt.Errorf("cache: redis auth failed: %w", err)
		}
	}

	// Select DB if > 0
	if r.cfg.DB > 0 {
		if _, err := r.execCommand("SELECT", strconv.Itoa(r.cfg.DB)); err != nil {
			r.closeConn()
			return nil, nil, fmt.Errorf("cache: redis select db %d failed: %w", r.cfg.DB, err)
		}
	}

	return r.conn, r.reader, nil
}

func (r *RedisCache) closeConn() {
	if r.conn != nil {
		_ = r.conn.Close()
		r.conn = nil
		r.reader = nil
	}
}

// encodeRESP serializes a command and arguments as RESP array.
func encodeRESP(args ...string) []byte {
	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("*%d\r\n", len(args)))
	for _, arg := range args {
		buf.WriteString(fmt.Sprintf("$%d\r\n%s\r\n", len(arg), arg))
	}
	return buf.Bytes()
}

// execCommand sends a RESP command and reads the response.
func (r *RedisCache) execCommand(args ...string) (interface{}, error) {
	conn, reader, err := r.getConnection()
	if err != nil {
		return nil, err
	}

	_ = conn.SetDeadline(time.Now().Add(r.timeout))
	cmd := encodeRESP(args...)
	if _, err := conn.Write(cmd); err != nil {
		r.closeConn()
		return nil, err
	}

	res, err := readRESP(reader)
	if err != nil {
		r.closeConn()
		return nil, err
	}
	return res, nil
}

func readRESP(r *bufio.Reader) (interface{}, error) {
	line, err := r.ReadString('\n')
	if err != nil {
		return nil, err
	}
	line = strings.TrimRight(line, "\r\n")
	if len(line) == 0 {
		return nil, fmt.Errorf("empty resp line")
	}

	prefix := line[0]
	payload := line[1:]

	switch prefix {
	case '+': // Simple string
		return payload, nil
	case '-': // Error
		return nil, fmt.Errorf("redis error: %s", payload)
	case ':': // Integer
		return strconv.ParseInt(payload, 10, 64)
	case '$': // Bulk string
		length, err := strconv.Atoi(payload)
		if err != nil {
			return nil, err
		}
		if length == -1 {
			return nil, nil // Nil bulk string
		}
		buf := make([]byte, length+2) // include \r\n
		if _, err := io.ReadFull(r, buf); err != nil {
			return nil, err
		}
		return buf[:length], nil
	case '*': // Array
		count, err := strconv.Atoi(payload)
		if err != nil {
			return nil, err
		}
		if count == -1 {
			return nil, nil
		}
		items := make([]interface{}, count)
		for i := 0; i < count; i++ {
			item, err := readRESP(r)
			if err != nil {
				return nil, err
			}
			items[i] = item
		}
		return items, nil
	default:
		return nil, fmt.Errorf("unsupported resp type: %c", prefix)
	}
}

func (r *RedisCache) Get(_ context.Context, key string) ([]byte, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	res, err := r.execCommand("GET", key)
	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, ErrCacheMiss
	}

	bytesVal, ok := res.([]byte)
	if !ok {
		return nil, ErrCacheMiss
	}
	return bytesVal, nil
}

func (r *RedisCache) GetString(ctx context.Context, key string) (string, error) {
	b, err := r.Get(ctx, key)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (r *RedisCache) Set(_ context.Context, key string, val []byte, ttl time.Duration) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	var err error
	if ttl > 0 {
		seconds := strconv.Itoa(int(ttl.Seconds()))
		_, err = r.execCommand("SETEX", key, seconds, string(val))
	} else {
		_, err = r.execCommand("SET", key, string(val))
	}
	return err
}

func (r *RedisCache) SetString(ctx context.Context, key string, val string, ttl time.Duration) error {
	return r.Set(ctx, key, []byte(val), ttl)
}

func (r *RedisCache) Delete(_ context.Context, key string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	_, err := r.execCommand("DEL", key)
	return err
}

func (r *RedisCache) Has(_ context.Context, key string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	res, err := r.execCommand("EXISTS", key)
	if err != nil {
		return false, err
	}
	count, ok := res.(int64)
	return ok && count > 0, nil
}

func (r *RedisCache) Clear(_ context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	_, err := r.execCommand("FLUSHDB")
	return err
}

func (r *RedisCache) Remember(ctx context.Context, key string, ttl time.Duration, fn func() ([]byte, error)) ([]byte, error) {
	val, err := r.Get(ctx, key)
	if err == nil {
		return val, nil
	}
	if err != ErrCacheMiss {
		return nil, err
	}

	computed, err := fn()
	if err != nil {
		return nil, err
	}

	_ = r.Set(ctx, key, computed, ttl)
	return computed, nil
}

func (r *RedisCache) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.closeConn()
	return nil
}

var _ Cache = (*RedisCache)(nil)
