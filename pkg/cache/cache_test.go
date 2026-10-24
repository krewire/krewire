package cache_test

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/krewire/krewire/pkg/cache"
)

func TestMemoryCache_Operations(t *testing.T) {
	c := cache.NewMemoryCache(10 * time.Millisecond)
	defer c.Close()

	ctx := context.Background()

	// 1. Set & Get
	err := c.Set(ctx, "session:user1", []byte("data_1"), 0)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	val, err := c.GetString(ctx, "session:user1")
	if err != nil {
		t.Fatalf("GetString failed: %v", err)
	}
	if val != "data_1" {
		t.Errorf("expected data_1, got %s", val)
	}

	// 2. Has
	has, err := c.Has(ctx, "session:user1")
	if err != nil || !has {
		t.Errorf("expected Has to be true, got %v (err: %v)", has, err)
	}

	// 3. TTL Expiration
	err = c.Set(ctx, "temp_key", []byte("expires_fast"), 20*time.Millisecond)
	if err != nil {
		t.Fatalf("Set with TTL failed: %v", err)
	}

	time.Sleep(30 * time.Millisecond)
	_, err = c.Get(ctx, "temp_key")
	if err != cache.ErrCacheMiss {
		t.Errorf("expected ErrCacheMiss on expired key, got %v", err)
	}

	// 4. Remember
	calls := 0
	remembered, err := c.Remember(ctx, "computed", 1*time.Minute, func() ([]byte, error) {
		calls++
		return []byte("expensive_result"), nil
	})
	if err != nil || string(remembered) != "expensive_result" {
		t.Fatalf("Remember failed: %v", err)
	}
	if calls != 1 {
		t.Errorf("expected 1 call, got %d", calls)
	}

	// Second Remember call must hit cache and NOT call fn
	remembered2, err := c.Remember(ctx, "computed", 1*time.Minute, func() ([]byte, error) {
		calls++
		return []byte("new_result"), nil
	})
	if err != nil || string(remembered2) != "expensive_result" {
		t.Fatalf("second Remember failed: %v", err)
	}
	if calls != 1 {
		t.Errorf("expected still 1 call (cached), got %d", calls)
	}

	// 5. Delete & Clear
	_ = c.Delete(ctx, "session:user1")
	_, err = c.Get(ctx, "session:user1")
	if err != cache.ErrCacheMiss {
		t.Errorf("expected ErrCacheMiss after delete, got %v", err)
	}

	_ = c.Clear(ctx)
	has, _ = c.Has(ctx, "computed")
	if has {
		t.Error("expected cache to be empty after clear")
	}
}

func TestFileCache_Operations(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "krewire_cache_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	c, err := cache.NewFileCache(cache.FileCacheConfig{Directory: tempDir})
	if err != nil {
		t.Fatalf("NewFileCache failed: %v", err)
	}
	defer c.Close()

	ctx := context.Background()

	// Set & Get
	err = c.SetString(ctx, "site_config", "theme=dark", 0)
	if err != nil {
		t.Fatalf("SetString failed: %v", err)
	}

	val, err := c.GetString(ctx, "site_config")
	if err != nil {
		t.Fatalf("GetString failed: %v", err)
	}
	if val != "theme=dark" {
		t.Errorf("expected theme=dark, got %s", val)
	}

	// TTL Expiration
	err = c.SetString(ctx, "token", "abc", 15*time.Millisecond)
	if err != nil {
		t.Fatalf("SetString TTL failed: %v", err)
	}

	time.Sleep(25 * time.Millisecond)
	_, err = c.GetString(ctx, "token")
	if err != cache.ErrCacheMiss {
		t.Errorf("expected ErrCacheMiss, got %v", err)
	}
}

func TestRedisCache_MockServer(t *testing.T) {
	// Start in-memory TCP mock RESP server
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	mockStore := make(map[string]string)

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				reader := bufio.NewReader(c)
				for {
					line, err := reader.ReadString('\n')
					if err != nil {
						return
					}
					line = strings.TrimSpace(line)
					if strings.HasPrefix(line, "*") { // array length
						// Read command args
						var numArgs int
						_, _ = fmt.Sscanf(line, "*%d", &numArgs)
						args := make([]string, numArgs)
						for i := 0; i < numArgs; i++ {
							_, _ = reader.ReadString('\n') // arg len
							valLine, _ := reader.ReadString('\n')
							args[i] = strings.TrimSpace(valLine)
						}

						if len(args) == 0 {
							continue
						}

						cmd := strings.ToUpper(args[0])
						switch cmd {
						case "PING":
							_, _ = c.Write([]byte("+PONG\r\n"))
						case "SET", "SETEX":
							key := args[1]
							var val string
							if cmd == "SETEX" {
								val = args[3]
							} else {
								val = args[2]
							}
							mockStore[key] = val
							_, _ = c.Write([]byte("+OK\r\n"))
						case "GET":
							key := args[1]
							val, ok := mockStore[key]
							if !ok {
								_, _ = c.Write([]byte("$-1\r\n"))
							} else {
								resp := fmt.Sprintf("$%d\r\n%s\r\n", len(val), val)
								_, _ = c.Write([]byte(resp))
							}
						case "DEL":
							delete(mockStore, args[1])
							_, _ = c.Write([]byte(":1\r\n"))
						case "EXISTS":
							_, ok := mockStore[args[1]]
							if ok {
								_, _ = c.Write([]byte(":1\r\n"))
							} else {
								_, _ = c.Write([]byte(":0\r\n"))
							}
						case "FLUSHDB":
							mockStore = make(map[string]string)
							_, _ = c.Write([]byte("+OK\r\n"))
						default:
							_, _ = c.Write([]byte("+OK\r\n"))
						}
					}
				}
			}(conn)
		}
	}()

	rc, err := cache.NewRedisCache(cache.RedisConfig{
		Address: listener.Addr().String(),
	})
	if err != nil {
		t.Fatal(err)
	}
	defer rc.Close()

	ctx := context.Background()

	// 1. Set & Get
	err = rc.SetString(ctx, "app:user:1", "Alice", 10*time.Second)
	if err != nil {
		t.Fatalf("Redis Set failed: %v", err)
	}

	got, err := rc.GetString(ctx, "app:user:1")
	if err != nil {
		t.Fatalf("Redis Get failed: %v", err)
	}
	if got != "Alice" {
		t.Errorf("expected Alice, got %s", got)
	}

	// 2. Has
	has, err := rc.Has(ctx, "app:user:1")
	if err != nil || !has {
		t.Errorf("expected true, got %v (err %v)", has, err)
	}

	// 3. Delete
	_ = rc.Delete(ctx, "app:user:1")
	_, err = rc.Get(ctx, "app:user:1")
	if err != cache.ErrCacheMiss {
		t.Errorf("expected ErrCacheMiss, got %v", err)
	}
}
