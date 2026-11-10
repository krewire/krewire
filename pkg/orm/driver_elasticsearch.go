package orm

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
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
func NewElasticsearch(cfg ElasticsearchConfig) (*ElasticsearchAdapter, error) {
	addrs := cfg.Addresses
	if len(addrs) == 0 {
		addrs = []string{DefaultElasticsearchAddress}
	}

	cleanAddrs := make([]string, len(addrs))
	for i, a := range addrs {
		cleanAddrs[i] = strings.TrimRight(a, "/")
	}

	httpClient := cfg.Client
	if httpClient == nil {
		timeout := cfg.Timeout
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

		if cfg.TLSInsecureSkipVerify {
			tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
		}

		httpClient = &http.Client{
			Transport: tr,
			Timeout:   timeout,
		}
	}

	var auth string
	if cfg.Username != "" {
		auth = base64.StdEncoding.EncodeToString([]byte(cfg.Username + ":" + cfg.Password))
	}

	retries := cfg.MaxRetries
	if retries <= 0 {
		retries = DefaultMaxRetries
	}

	backoff := cfg.RetryBackoff
	if backoff <= 0 {
		backoff = DefaultRetryBackoff
	}

	headers := make(map[string]string)
	for k, v := range cfg.Headers {
		headers[k] = v
	}

	return &ElasticsearchAdapter{
		addresses:    cleanAddrs,
		client:       httpClient,
		basicAuth:    auth,
		apiKey:       cfg.APIKey,
		bearerToken:  cfg.BearerToken,
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
// Cluster Diagnostics & Health
// ----------------------------------------------------------------------------

// ClusterHealthResponse holds cluster status metrics.
type ClusterHealthResponse struct {
	ClusterName                 string  `json:"cluster_name"`
	Status                      string  `json:"status"` // "green", "yellow", "red"
	TimedOut                    bool    `json:"timed_out"`
	NumberOfNodes               int     `json:"number_of_nodes"`
	NumberOfDataNodes           int     `json:"number_of_data_nodes"`
	ActivePrimaryShards         int     `json:"active_primary_shards"`
	ActiveShards                int     `json:"active_shards"`
	RelocatingShards            int     `json:"relocating_shards"`
	InitializingShards          int     `json:"initializing_shards"`
	UnassignedShards            int     `json:"unassigned_shards"`
	ActiveShardsPercentAsNumber float64 `json:"active_shards_percent_as_number"`
}

// ClusterHealth queries the _cluster/health API.
func (a *ElasticsearchAdapter) ClusterHealth(ctx context.Context) (*ClusterHealthResponse, error) {
	respBytes, err := a.executeRequest(ctx, http.MethodGet, "/_cluster/health", nil)
	if err != nil {
		return nil, err
	}

	var health ClusterHealthResponse
	if err := json.Unmarshal(respBytes, &health); err != nil {
		return nil, errs.FailureError(fmt.Sprintf("orm: unmarshal cluster health: %v", err))
	}
	return &health, nil
}

// ----------------------------------------------------------------------------
// Document Operations
// ----------------------------------------------------------------------------

// IndexResponse holds results from an indexing or update operation.
type IndexResponse struct {
	Index   string `json:"_index"`
	ID      string `json:"_id"`
	Version int64  `json:"_version"`
	Result  string `json:"result"` // e.g. "created", "updated", "deleted"
}

// Index stores or updates a document in an Elasticsearch index.
func (a *ElasticsearchAdapter) Index(ctx context.Context, index, id string, doc interface{}) (*IndexResponse, error) {
	body, err := json.Marshal(doc)
	if err != nil {
		return nil, errs.FailureError(fmt.Sprintf("orm: failed to serialize doc: %v", err))
	}

	var endpoint string
	method := http.MethodPost
	if id != "" {
		endpoint = fmt.Sprintf("/%s/_doc/%s", index, id)
		method = http.MethodPut
	} else {
		endpoint = fmt.Sprintf("/%s/_doc", index)
	}

	respBytes, err := a.executeRequest(ctx, method, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var idxRes IndexResponse
	if err := json.Unmarshal(respBytes, &idxRes); err != nil {
		return nil, err
	}
	return &idxRes, nil
}

type esGetDocResponse struct {
	Index  string          `json:"_index"`
	ID     string          `json:"_id"`
	Found  bool            `json:"found"`
	Source json.RawMessage `json:"_source"`
}

// Get fetches a document by ID and unmarshals its _source into dest.
func (a *ElasticsearchAdapter) Get(ctx context.Context, index, id string, dest interface{}) error {
	endpoint := fmt.Sprintf("/%s/_doc/%s", index, id)
	respBytes, err := a.executeRequest(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}

	var getRes esGetDocResponse
	if err := json.Unmarshal(respBytes, &getRes); err != nil {
		return err
	}
	if !getRes.Found {
		return errs.FailureError("orm: document not found")
	}

	return json.Unmarshal(getRes.Source, dest)
}

// Update performs a partial document update using the _update API.
func (a *ElasticsearchAdapter) Update(ctx context.Context, index, id string, docOrScript interface{}) (*IndexResponse, error) {
	wrapped := map[string]interface{}{"doc": docOrScript}
	body, err := json.Marshal(wrapped)
	if err != nil {
		return nil, errs.FailureError(fmt.Sprintf("orm: serialize update body: %v", err))
	}

	endpoint := fmt.Sprintf("/%s/_update/%s", index, id)
	respBytes, err := a.executeRequest(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var idxRes IndexResponse
	if err := json.Unmarshal(respBytes, &idxRes); err != nil {
		return nil, err
	}
	return &idxRes, nil
}

// Delete removes a document by ID.
func (a *ElasticsearchAdapter) Delete(ctx context.Context, index, id string) error {
	endpoint := fmt.Sprintf("/%s/_doc/%s", index, id)
	_, err := a.executeRequest(ctx, http.MethodDelete, endpoint, nil)
	return err
}

// Exists checks if a document exists by ID via a HEAD request.
func (a *ElasticsearchAdapter) Exists(ctx context.Context, index, id string) (bool, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.closed {
		return false, errs.FailureError("orm: elasticsearch adapter is closed")
	}

	endpoint := fmt.Sprintf("%s/%s/_doc/%s", a.nextAddress(), index, id)
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, endpoint, nil)
	if err != nil {
		return false, err
	}
	a.applyHeaders(req)

	res, err := a.client.Do(req)
	if err != nil {
		return false, errs.FailureError(fmt.Sprintf("orm: elasticsearch exists request failed: %v", err))
	}
	defer res.Body.Close()

	if res.StatusCode == http.StatusOK {
		return true, nil
	}
	if res.StatusCode == http.StatusNotFound {
		return false, nil
	}
	return false, errs.FailureError(fmt.Sprintf("elasticsearch exists returned status %d", res.StatusCode))
}

// ----------------------------------------------------------------------------
// Bulk Operations API (/_bulk)
// ----------------------------------------------------------------------------

// BulkAction defines the action verb for a bulk operation.
type BulkAction string

const (
	BulkActionIndex  BulkAction = "index"
	BulkActionCreate BulkAction = "create"
	BulkActionUpdate BulkAction = "update"
	BulkActionDelete BulkAction = "delete"
)

// BulkItem represents an individual action in a bulk payload.
type BulkItem struct {
	Action BulkAction
	Index  string
	ID     string
	Doc    interface{}
}

// BulkItemResult holds the status of an item within a BulkResponse.
type BulkItemResult struct {
	Index   string `json:"_index"`
	ID      string `json:"_id"`
	Status  int    `json:"status"`
	Result  string `json:"result,omitempty"`
	Error   any    `json:"error,omitempty"`
	Version int64  `json:"_version,omitempty"`
}

// BulkResponse contains the results of a Bulk API call.
type BulkResponse struct {
	Took   int64                       `json:"took"`
	Errors bool                        `json:"errors"`
	Items  []map[string]BulkItemResult `json:"items"`
}

// Bulk executes a batch of indexing, update, or deletion operations in NDJSON format.
func (a *ElasticsearchAdapter) Bulk(ctx context.Context, items []BulkItem) (*BulkResponse, error) {
	if len(items) == 0 {
		return &BulkResponse{}, nil
	}

	var buf bytes.Buffer
	for _, it := range items {
		actionMeta := map[string]interface{}{}
		metaHeader := map[string]interface{}{}
		if it.Index != "" {
			metaHeader["_index"] = it.Index
		}
		if it.ID != "" {
			metaHeader["_id"] = it.ID
		}
		actionMeta[string(it.Action)] = metaHeader

		metaBytes, err := json.Marshal(actionMeta)
		if err != nil {
			return nil, err
		}
		buf.Write(metaBytes)
		buf.WriteByte('\n')

		if it.Action == BulkActionDelete {
			continue
		}

		var docBytes []byte
		if it.Action == BulkActionUpdate {
			wrapped := map[string]interface{}{"doc": it.Doc}
			docBytes, err = json.Marshal(wrapped)
		} else {
			docBytes, err = json.Marshal(it.Doc)
		}
		if err != nil {
			return nil, err
		}
		buf.Write(docBytes)
		buf.WriteByte('\n')
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.nextAddress()+"/_bulk", &buf)
	if err != nil {
		return nil, err
	}
	a.applyHeaders(req)
	req.Header.Set("Content-Type", "application/x-ndjson")

	res, err := a.client.Do(req)
	if err != nil {
		return nil, errs.FailureError(fmt.Sprintf("orm: bulk request failed: %v", err))
	}
	defer res.Body.Close()

	respBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if res.StatusCode >= 400 {
		return nil, errs.FailureError(fmt.Sprintf("bulk api error (status %d): %s", res.StatusCode, string(respBytes)))
	}

	var bulkRes BulkResponse
	if err := json.Unmarshal(respBytes, &bulkRes); err != nil {
		return nil, errs.FailureError(fmt.Sprintf("orm: unmarshal bulk response: %v", err))
	}
	return &bulkRes, nil
}

// ----------------------------------------------------------------------------
// Search API (/_search)
// ----------------------------------------------------------------------------

// SearchRequest holds parameters for an Elasticsearch query.
type SearchRequest struct {
	Index        string
	Query        map[string]interface{}
	From         int
	Size         int
	Sort         []map[string]interface{}
	Aggregations map[string]interface{}
	Highlight    map[string]interface{}
	Source       []string
	SearchAfter  []interface{}
}

// SearchHit represents a single matched document in search results.
type SearchHit struct {
	ID        string              `json:"_id"`
	Index     string              `json:"_index"`
	Score     float64             `json:"_score"`
	Source    json.RawMessage     `json:"_source"`
	Highlight map[string][]string `json:"highlight,omitempty"`
	Sort      []interface{}       `json:"sort,omitempty"`
}

// SearchResult holds search execution results including hits and aggregations.
type SearchResult struct {
	Took         int64
	TimedOut     bool
	TotalHits    int64
	Hits         []SearchHit
	Aggregations json.RawMessage
}

type esSearchRawResponse struct {
	Took     int64 `json:"took"`
	TimedOut bool  `json:"timed_out"`
	Hits     struct {
		Total struct {
			Value int64 `json:"value"`
		} `json:"total"`
		Hits []SearchHit `json:"hits"`
	} `json:"hits"`
	Aggregations json.RawMessage `json:"aggregations,omitempty"`
}

// Search runs a query using the Elasticsearch Search API.
func (a *ElasticsearchAdapter) Search(ctx context.Context, sreq SearchRequest) (*SearchResult, error) {
	bodyMap := map[string]interface{}{}
	if len(sreq.Query) > 0 {
		bodyMap["query"] = sreq.Query
	} else {
		bodyMap["query"] = map[string]interface{}{"match_all": map[string]interface{}{}}
	}
	if sreq.From > 0 {
		bodyMap["from"] = sreq.From
	}
	if sreq.Size > 0 {
		bodyMap["size"] = sreq.Size
	} else {
		bodyMap["size"] = 10
	}
	if len(sreq.Sort) > 0 {
		bodyMap["sort"] = sreq.Sort
	}
	if len(sreq.Aggregations) > 0 {
		bodyMap["aggs"] = sreq.Aggregations
	}
	if len(sreq.Highlight) > 0 {
		bodyMap["highlight"] = sreq.Highlight
	}
	if len(sreq.Source) > 0 {
		bodyMap["_source"] = sreq.Source
	}
	if len(sreq.SearchAfter) > 0 {
		bodyMap["search_after"] = sreq.SearchAfter
	}

	bodyBytes, err := json.Marshal(bodyMap)
	if err != nil {
		return nil, err
	}

	endpoint := fmt.Sprintf("/%s/_search", sreq.Index)
	respBytes, err := a.executeRequest(ctx, http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}

	var rawRes esSearchRawResponse
	if err := json.Unmarshal(respBytes, &rawRes); err != nil {
		return nil, err
	}

	return &SearchResult{
		Took:         rawRes.Took,
		TimedOut:     rawRes.TimedOut,
		TotalHits:    rawRes.Hits.Total.Value,
		Hits:         rawRes.Hits.Hits,
		Aggregations: rawRes.Aggregations,
	}, nil
}

// ----------------------------------------------------------------------------
// Index Management APIs
// ----------------------------------------------------------------------------

// CreateIndex creates an index with optional mappings and settings.
func (a *ElasticsearchAdapter) CreateIndex(ctx context.Context, index string, body map[string]interface{}) error {
	var r io.Reader
	if len(body) > 0 {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		r = bytes.NewReader(b)
	}

	endpoint := fmt.Sprintf("/%s", index)
	_, err := a.executeRequest(ctx, http.MethodPut, endpoint, r)
	return err
}

// DeleteIndex removes an index.
func (a *ElasticsearchAdapter) DeleteIndex(ctx context.Context, index string) error {
	endpoint := fmt.Sprintf("/%s", index)
	_, err := a.executeRequest(ctx, http.MethodDelete, endpoint, nil)
	return err
}

// IndexExists checks whether an index exists.
func (a *ElasticsearchAdapter) IndexExists(ctx context.Context, index string) (bool, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.closed {
		return false, errs.FailureError("orm: elasticsearch adapter is closed")
	}

	endpoint := fmt.Sprintf("%s/%s", a.nextAddress(), index)
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, endpoint, nil)
	if err != nil {
		return false, err
	}
	a.applyHeaders(req)

	res, err := a.client.Do(req)
	if err != nil {
		return false, errs.FailureError(fmt.Sprintf("orm: index exists failed: %v", err))
	}
	defer res.Body.Close()

	if res.StatusCode == http.StatusOK {
		return true, nil
	}
	if res.StatusCode == http.StatusNotFound {
		return false, nil
	}
	return false, errs.FailureError(fmt.Sprintf("index exists returned status %d", res.StatusCode))
}

// RefreshIndex triggers an explicit refresh to make recent mutations searchable.
func (a *ElasticsearchAdapter) RefreshIndex(ctx context.Context, index string) error {
	endpoint := fmt.Sprintf("/%s/_refresh", index)
	_, err := a.executeRequest(ctx, http.MethodPost, endpoint, nil)
	return err
}

// CreateAlias associates an alias with an index.
func (a *ElasticsearchAdapter) CreateAlias(ctx context.Context, index, alias string) error {
	payload := map[string]interface{}{
		"actions": []map[string]interface{}{
			{"add": map[string]interface{}{"index": index, "alias": alias}},
		},
	}
	b, _ := json.Marshal(payload)
	_, err := a.executeRequest(ctx, http.MethodPost, "/_aliases", bytes.NewReader(b))
	return err
}

// DeleteAlias removes an alias from an index.
func (a *ElasticsearchAdapter) DeleteAlias(ctx context.Context, index, alias string) error {
	payload := map[string]interface{}{
		"actions": []map[string]interface{}{
			{"remove": map[string]interface{}{"index": index, "alias": alias}},
		},
	}
	b, _ := json.Marshal(payload)
	_, err := a.executeRequest(ctx, http.MethodPost, "/_aliases", bytes.NewReader(b))
	return err
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
