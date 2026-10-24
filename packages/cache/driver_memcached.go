package cache

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

// MemcachedConfig contains connection parameters for Memcached servers.
type MemcachedConfig struct {
	Address string // "host:port", default: "127.0.0.1:11211"
	Timeout time.Duration
}

// MemcachedCache implements Cache using Memcached text protocol.
type MemcachedCache struct {
	mu      sync.Mutex
	cfg     MemcachedConfig
	conn    net.Conn
	reader  *bufio.Reader
	timeout time.Duration
}

// NewMemcachedCache creates a new Memcached Cache client.
func NewMemcachedCache(cfg MemcachedConfig) (*MemcachedCache, error) {
	if cfg.Address == "" {
		cfg.Address = "127.0.0.1:11211"
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 5 * time.Second
	}

	return &MemcachedCache{
		cfg:     cfg,
		timeout: cfg.Timeout,
	}, nil
}

func (m *MemcachedCache) getConnection() (net.Conn, *bufio.Reader, error) {
	if m.conn != nil {
		return m.conn, m.reader, nil
	}

	conn, err := net.DialTimeout("tcp", m.cfg.Address, m.timeout)
	if err != nil {
		return nil, nil, fmt.Errorf("cache: memcached connect to %s failed: %w", m.cfg.Address, err)
	}

	m.conn = conn
	m.reader = bufio.NewReader(conn)
	return m.conn, m.reader, nil
}

func (m *MemcachedCache) closeConn() {
	if m.conn != nil {
		_ = m.conn.Close()
		m.conn = nil
		m.reader = nil
	}
}

func (m *MemcachedCache) Get(_ context.Context, key string) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	conn, reader, err := m.getConnection()
	if err != nil {
		return nil, err
	}

	_ = conn.SetDeadline(time.Now().Add(m.timeout))
	cmd := fmt.Sprintf("get %s\r\n", key)
	if _, err := conn.Write([]byte(cmd)); err != nil {
		m.closeConn()
		return nil, err
	}

	// Response: VALUE <key> <flags> <bytes>\r\n<data>\r\nEND\r\n
	line, err := reader.ReadString('\n')
	if err != nil {
		m.closeConn()
		return nil, err
	}
	line = strings.TrimSpace(line)

	if line == "END" {
		return nil, ErrCacheMiss
	}

	if !strings.HasPrefix(line, "VALUE ") {
		return nil, fmt.Errorf("cache: unexpected memcached response: %s", line)
	}

	parts := strings.Split(line, " ")
	if len(parts) < 4 {
		return nil, fmt.Errorf("cache: malformed value line: %s", line)
	}

	bytesLen, err := strconv.Atoi(parts[3])
	if err != nil {
		return nil, err
	}

	buf := make([]byte, bytesLen+2) // include trailing \r\n
	n := 0
	for n < len(buf) {
		readN, err := reader.Read(buf[n:])
		if err != nil {
			m.closeConn()
			return nil, err
		}
		n += readN
	}

	// Read final END\r\n
	_, _ = reader.ReadString('\n')

	return buf[:bytesLen], nil
}

func (m *MemcachedCache) GetString(ctx context.Context, key string) (string, error) {
	b, err := m.Get(ctx, key)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (m *MemcachedCache) Set(_ context.Context, key string, val []byte, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	conn, reader, err := m.getConnection()
	if err != nil {
		return err
	}

	_ = conn.SetDeadline(time.Now().Add(m.timeout))
	seconds := int(ttl.Seconds())

	// set <key> <flags> <exptime> <bytes>\r\n<data>\r\n
	cmd := fmt.Sprintf("set %s 0 %d %d\r\n%s\r\n", key, seconds, len(val), string(val))
	if _, err := conn.Write([]byte(cmd)); err != nil {
		m.closeConn()
		return err
	}

	line, err := reader.ReadString('\n')
	if err != nil {
		m.closeConn()
		return err
	}

	if !strings.HasPrefix(line, "STORED") {
		return fmt.Errorf("cache: memcached set failed: %s", strings.TrimSpace(line))
	}
	return nil
}

func (m *MemcachedCache) SetString(ctx context.Context, key string, val string, ttl time.Duration) error {
	return m.Set(ctx, key, []byte(val), ttl)
}

func (m *MemcachedCache) Delete(_ context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	conn, reader, err := m.getConnection()
	if err != nil {
		return err
	}

	_ = conn.SetDeadline(time.Now().Add(m.timeout))
	cmd := fmt.Sprintf("delete %s\r\n", key)
	if _, err := conn.Write([]byte(cmd)); err != nil {
		m.closeConn()
		return err
	}

	line, err := reader.ReadString('\n')
	if err != nil {
		m.closeConn()
		return err
	}

	if !strings.HasPrefix(line, "DELETED") && !strings.HasPrefix(line, "NOT_FOUND") {
		return fmt.Errorf("cache: memcached delete failed: %s", strings.TrimSpace(line))
	}
	return nil
}

func (m *MemcachedCache) Has(ctx context.Context, key string) (bool, error) {
	_, err := m.Get(ctx, key)
	if err == nil {
		return true, nil
	}
	if err == ErrCacheMiss {
		return false, nil
	}
	return false, err
}

func (m *MemcachedCache) Clear(_ context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	conn, reader, err := m.getConnection()
	if err != nil {
		return err
	}

	_ = conn.SetDeadline(time.Now().Add(m.timeout))
	if _, err := conn.Write([]byte("flush_all\r\n")); err != nil {
		m.closeConn()
		return err
	}

	line, err := reader.ReadString('\n')
	if err != nil {
		m.closeConn()
		return err
	}

	if !strings.HasPrefix(line, "OK") {
		return fmt.Errorf("cache: memcached flush_all failed: %s", strings.TrimSpace(line))
	}
	return nil
}

func (m *MemcachedCache) Remember(ctx context.Context, key string, ttl time.Duration, fn func() ([]byte, error)) ([]byte, error) {
	val, err := m.Get(ctx, key)
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

	_ = m.Set(ctx, key, computed, ttl)
	return computed, nil
}

func (m *MemcachedCache) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closeConn()
	return nil
}

var _ Cache = (*MemcachedCache)(nil)
