package orm

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/krewire/krewire/pkg/errs"
)

const (
	// DefaultElasticsearchAddress is the fallback cluster HTTP endpoint.
	DefaultElasticsearchAddress = "http://localhost:9200"

	// DefaultElasticsearchTimeout is the default HTTP request timeout for cluster queries.
	DefaultElasticsearchTimeout = 15 * time.Second

	// DefaultMaxRetries is the number of retry attempts for transient cluster errors.
	DefaultMaxRetries = 3

	// DefaultRetryBackoff is the initial backoff delay before retrying.
	DefaultRetryBackoff = 100 * time.Millisecond
)

// ElasticsearchConfig contains cluster connection, security, and pool parameters.
type ElasticsearchConfig struct {
	Addresses             []string // default: []string{DefaultElasticsearchAddress}
	Username              string
	Password              string
	APIKey                string
	BearerToken           string
	Headers               map[string]string
	Timeout               time.Duration // default: DefaultElasticsearchTimeout (15s)
	MaxRetries            int           // default: DefaultMaxRetries (3)
	RetryBackoff          time.Duration // default: DefaultRetryBackoff (100ms)
	TLSInsecureSkipVerify bool
	Client                *http.Client
}

// ElasticsearchAdapter implements Adapter for full-text search, analytics, and indexing.
type ElasticsearchAdapter struct {
	mu           sync.RWMutex
	addresses    []string
	counter      atomic.Uint64
	client       *http.Client
	basicAuth    string
	apiKey       string
	bearerToken  string
	headers      map[string]string
	maxRetries   int
	retryBackoff time.Duration
	closed       bool
}

// Ensure ElasticsearchAdapter implements Adapter.
var _ Adapter = (*ElasticsearchAdapter)(nil)

// NewElasticsearch initializes a production-ready Elasticsearch adapter.
func NewElasticsearch(config ElasticsearchConfig) (*ElasticsearchAdapter, error) {
	addrs := config.Addresses
	if len(addrs) == 0 {
		addrs = []string{DefaultElasticsearchAddress}
	}

	cleanAddrs := make([]string, len(addrs))
	for i, a := range addrs {
		cleanAddrs[i] = strings.TrimRight(a, "/")
	}

	httpClient := config.Client
	if httpClient == nil {
		timeout := config.Timeout
		if timeout <= 0 {
			timeout = DefaultElasticsearchTimeout
		}

		tr := &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   5 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			MaxIdleConns:          100,
			MaxIdleConnsPerHost:   20,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ResponseHeaderTimeout: 30 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		}

		if config.TLSInsecureSkipVerify {
			tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
		}

		httpClient = &http.Client{
			Transport: tr,
			Timeout:   timeout,
		}
	}

	var auth string
	if config.Username != "" {
		auth = base64.StdEncoding.EncodeToString([]byte(config.Username + ":" + config.Password))
	}

	retries := config.MaxRetries
	if retries <= 0 {
		retries = DefaultMaxRetries
	}

	backoff := config.RetryBackoff
	if backoff <= 0 {
		backoff = DefaultRetryBackoff
	}

	headers := make(map[string]string)
	for k, v := range config.Headers {
		headers[k] = v
	}

	return &ElasticsearchAdapter{
		addresses:    cleanAddrs,
		client:       httpClient,
		basicAuth:    auth,
		apiKey:       config.APIKey,
		bearerToken:  config.BearerToken,
		headers:      headers,
		maxRetries:   retries,
		retryBackoff: backoff,
	}, nil
}

func (a *ElasticsearchAdapter) Name() string {
	return "elasticsearch"
}

func (a *ElasticsearchAdapter) Type() AdapterType {
	return AdapterTypeSearch
}

// nextAddress selects an endpoint in round-robin fashion across available cluster nodes.
func (a *ElasticsearchAdapter) nextAddress() string {
	idx := a.counter.Add(1) - 1
	return a.addresses[idx%uint64(len(a.addresses))]
}

func (a *ElasticsearchAdapter) Ping(ctx context.Context) error {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.closed {
		return errs.FailureError("orm: elasticsearch adapter is closed")
	}

	addr := a.nextAddress()
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, addr, nil)
	if err != nil {
		return errs.FailureError(fmt.Sprintf("orm: ping request build failed: %v", err))
	}
	a.applyHeaders(req)

	res, err := a.client.Do(req)
	if err != nil {
		return errs.FailureError(fmt.Sprintf("orm: ping elasticsearch failed: %v", err))
	}
	defer res.Body.Close()

	if res.StatusCode >= 400 {
		return errs.FailureError(fmt.Sprintf("orm: ping elasticsearch returned status %d", res.StatusCode))
	}
	return nil
}

func (a *ElasticsearchAdapter) Close() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.closed = true
	return nil
}

// ----------------------------------------------------------------------------
// Request Execution & Retry Helper
// ----------------------------------------------------------------------------

func (a *ElasticsearchAdapter) executeRequest(ctx context.Context, method, endpoint string, body io.Reader) ([]byte, error) {
	a.mu.RLock()
	closed := a.closed
	a.mu.RUnlock()
	if closed {
		return nil, errs.FailureError("orm: elasticsearch adapter is closed")
	}

	var bodyBytes []byte
	if body != nil {
		var err error
		bodyBytes, err = io.ReadAll(body)
		if err != nil {
			return nil, err
		}
	}

	var lastErr error
	backoff := a.retryBackoff

	for attempt := 0; attempt <= a.maxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
				backoff *= 2
			}
		}

		addr := a.nextAddress()
		url := addr + endpoint

		var rdr io.Reader
		if bodyBytes != nil {
			rdr = bytes.NewReader(bodyBytes)
		}

		req, err := http.NewRequestWithContext(ctx, method, url, rdr)
		if err != nil {
			return nil, err
		}
		a.applyHeaders(req)

		res, err := a.client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		respBytes, err := io.ReadAll(res.Body)
		res.Body.Close()
		if err != nil {
			lastErr = err
			continue
		}

		// Retry on 429 (Too Many Requests) or 503 (Service Unavailable)
		if res.StatusCode == http.StatusTooManyRequests || res.StatusCode == http.StatusServiceUnavailable {
			lastErr = errs.FailureError(fmt.Sprintf("elasticsearch status %d: %s", res.StatusCode, string(respBytes)))
			continue
		}

		if res.StatusCode == http.StatusNotFound {
			return nil, errs.FailureError("orm: document not found")
		}

		if res.StatusCode >= 400 {
			return nil, errs.FailureError(fmt.Sprintf("elasticsearch error (status %d): %s", res.StatusCode, string(respBytes)))
		}

		return respBytes, nil
	}

	return nil, errs.FailureError(fmt.Sprintf("elasticsearch operation failed after %d retries: %v", a.maxRetries, lastErr))
}

func (a *ElasticsearchAdapter) applyHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	if a.apiKey != "" {
		req.Header.Set("Authorization", "ApiKey "+a.apiKey)
	} else if a.bearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+a.bearerToken)
	} else if a.basicAuth != "" {
		req.Header.Set("Authorization", "Basic "+a.basicAuth)
	}

	for k, v := range a.headers {
		req.Header.Set(k, v)
	}
}
