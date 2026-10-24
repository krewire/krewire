package orm

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/krewire/krewire/packages/errs"
)

// ElasticsearchConfig contains cluster connection and authentication parameters.
type ElasticsearchConfig struct {
	Addresses []string
	Username  string
	Password  string
	APIKey    string
	Timeout   time.Duration
	Client    *http.Client
}

// ElasticsearchAdapter implements Adapter for full-text search and document analytics.
type ElasticsearchAdapter struct {
	mu        sync.RWMutex
	addresses []string
	client    *http.Client
	basicAuth string
	apiKey    string
	closed    bool
}

// Ensure ElasticsearchAdapter implements Adapter.
var _ Adapter = (*ElasticsearchAdapter)(nil)

// NewElasticsearch initializes an Elasticsearch adapter.
func NewElasticsearch(cfg ElasticsearchConfig) (*ElasticsearchAdapter, error) {
	addrs := cfg.Addresses
	if len(addrs) == 0 {
		addrs = []string{"http://localhost:9200"}
	}

	for i, a := range addrs {
		addrs[i] = strings.TrimRight(a, "/")
	}

	httpClient := cfg.Client
	if httpClient == nil {
		timeout := cfg.Timeout
		if timeout <= 0 {
			timeout = 15 * time.Second
		}
		httpClient = &http.Client{Timeout: timeout}
	}

	var auth string
	if cfg.Username != "" {
		auth = base64.StdEncoding.EncodeToString([]byte(cfg.Username + ":" + cfg.Password))
	}

	return &ElasticsearchAdapter{
		addresses: addrs,
		client:    httpClient,
		basicAuth: auth,
		apiKey:    cfg.APIKey,
	}, nil
}

func (a *ElasticsearchAdapter) Name() string {
	return "elasticsearch"
}

func (a *ElasticsearchAdapter) Type() AdapterType {
	return AdapterTypeSearch
}

func (a *ElasticsearchAdapter) Ping(ctx context.Context) error {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.closed {
		return errs.FailureError("orm: elasticsearch adapter is closed")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodHead, a.addresses[0], nil)
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

// IndexResponse holds results from an indexing operation.
type IndexResponse struct {
	Index   string `json:"_index"`
	ID      string `json:"_id"`
	Version int64  `json:"_version"`
	Result  string `json:"result"` // e.g. "created", "updated"
}

// Index stores or updates a document in an Elasticsearch index.
func (a *ElasticsearchAdapter) Index(ctx context.Context, index, id string, doc interface{}) (*IndexResponse, error) {
	if a.closed {
		return nil, errs.FailureError("orm: elasticsearch adapter is closed")
	}

	body, err := json.Marshal(doc)
	if err != nil {
		return nil, errs.FailureError(fmt.Sprintf("orm: failed to serialize doc: %v", err))
	}

	var path string
	method := http.MethodPost
	if id != "" {
		path = fmt.Sprintf("%s/%s/_doc/%s", a.addresses[0], index, id)
		method = http.MethodPut
	} else {
		path = fmt.Sprintf("%s/%s/_doc", a.addresses[0], index)
	}

	req, err := http.NewRequestWithContext(ctx, method, path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	a.applyHeaders(req)

	res, err := a.client.Do(req)
	if err != nil {
		return nil, errs.FailureError(fmt.Sprintf("orm: elasticsearch index failed: %v", err))
	}
	defer res.Body.Close()

	respBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, errs.FailureError(fmt.Sprintf("elasticsearch index error (status %d): %s", res.StatusCode, string(respBytes)))
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
	if a.closed {
		return errs.FailureError("orm: elasticsearch adapter is closed")
	}

	path := fmt.Sprintf("%s/%s/_doc/%s", a.addresses[0], index, id)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, path, nil)
	if err != nil {
		return err
	}
	a.applyHeaders(req)

	res, err := a.client.Do(req)
	if err != nil {
		return errs.FailureError(fmt.Sprintf("orm: elasticsearch get failed: %v", err))
	}
	defer res.Body.Close()

	if res.StatusCode == http.StatusNotFound {
		return errs.FailureError("orm: document not found")
	}

	respBytes, err := io.ReadAll(res.Body)
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

// Delete removes a document by ID.
func (a *ElasticsearchAdapter) Delete(ctx context.Context, index, id string) error {
	if a.closed {
		return errs.FailureError("orm: elasticsearch adapter is closed")
	}

	path := fmt.Sprintf("%s/%s/_doc/%s", a.addresses[0], index, id)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return err
	}
	a.applyHeaders(req)

	res, err := a.client.Do(req)
	if err != nil {
		return errs.FailureError(fmt.Sprintf("orm: elasticsearch delete failed: %v", err))
	}
	defer res.Body.Close()

	if res.StatusCode >= 400 && res.StatusCode != http.StatusNotFound {
		respBytes, _ := io.ReadAll(res.Body)
		return errs.FailureError(fmt.Sprintf("elasticsearch delete error (status %d): %s", res.StatusCode, string(respBytes)))
	}
	return nil
}

// SearchRequest holds parameters for an Elasticsearch query.
type SearchRequest struct {
	Index string
	Query map[string]interface{}
	From  int
	Size  int
	Sort  []map[string]interface{}
}

// SearchHit represents a single matched document in search results.
type SearchHit struct {
	ID     string          `json:"_id"`
	Index  string          `json:"_index"`
	Score  float64         `json:"_score"`
	Source json.RawMessage `json:"_source"`
}

// SearchResult holds search execution results.
type SearchResult struct {
	TotalHits int64
	Hits      []SearchHit
}

type esSearchRawResponse struct {
	Hits struct {
		Total struct {
			Value int64 `json:"value"`
		} `json:"total"`
		Hits []SearchHit `json:"hits"`
	} `json:"hits"`
}

// Search runs a query using the Elasticsearch Search API.
func (a *ElasticsearchAdapter) Search(ctx context.Context, sreq SearchRequest) (*SearchResult, error) {
	if a.closed {
		return nil, errs.FailureError("orm: elasticsearch adapter is closed")
	}

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

	bodyBytes, err := json.Marshal(bodyMap)
	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf("%s/%s/_search", a.addresses[0], sreq.Index)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, path, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}
	a.applyHeaders(req)

	res, err := a.client.Do(req)
	if err != nil {
		return nil, errs.FailureError(fmt.Sprintf("orm: elasticsearch search failed: %v", err))
	}
	defer res.Body.Close()

	respBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, errs.FailureError(fmt.Sprintf("elasticsearch search error (status %d): %s", res.StatusCode, string(respBytes)))
	}

	var rawRes esSearchRawResponse
	if err := json.Unmarshal(respBytes, &rawRes); err != nil {
		return nil, err
	}

	return &SearchResult{
		TotalHits: rawRes.Hits.Total.Value,
		Hits:      rawRes.Hits.Hits,
	}, nil
}

// CreateIndex creates an index with optional mappings.
func (a *ElasticsearchAdapter) CreateIndex(ctx context.Context, index string, mappings map[string]interface{}) error {
	if a.closed {
		return errs.FailureError("orm: elasticsearch adapter is closed")
	}

	var body io.Reader
	if len(mappings) > 0 {
		b, err := json.Marshal(mappings)
		if err != nil {
			return err
		}
		body = bytes.NewReader(b)
	}

	path := fmt.Sprintf("%s/%s", a.addresses[0], index)
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, path, body)
	if err != nil {
		return err
	}
	a.applyHeaders(req)

	res, err := a.client.Do(req)
	if err != nil {
		return errs.FailureError(fmt.Sprintf("orm: create index failed: %v", err))
	}
	defer res.Body.Close()

	if res.StatusCode >= 400 && res.StatusCode != http.StatusBadRequest {
		respBytes, _ := io.ReadAll(res.Body)
		return errs.FailureError(fmt.Sprintf("create index error (status %d): %s", res.StatusCode, string(respBytes)))
	}
	return nil
}

// DeleteIndex removes an index.
func (a *ElasticsearchAdapter) DeleteIndex(ctx context.Context, index string) error {
	if a.closed {
		return errs.FailureError("orm: elasticsearch adapter is closed")
	}

	path := fmt.Sprintf("%s/%s", a.addresses[0], index)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return err
	}
	a.applyHeaders(req)

	res, err := a.client.Do(req)
	if err != nil {
		return errs.FailureError(fmt.Sprintf("orm: delete index failed: %v", err))
	}
	defer res.Body.Close()

	if res.StatusCode >= 400 && res.StatusCode != http.StatusNotFound {
		respBytes, _ := io.ReadAll(res.Body)
		return errs.FailureError(fmt.Sprintf("delete index error (status %d): %s", res.StatusCode, string(respBytes)))
	}
	return nil
}

func (a *ElasticsearchAdapter) applyHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if a.apiKey != "" {
		req.Header.Set("Authorization", "ApiKey "+a.apiKey)
	} else if a.basicAuth != "" {
		req.Header.Set("Authorization", "Basic "+a.basicAuth)
	}
}
