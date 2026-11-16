package orm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/krewire/krewire/pkg/errs"
)

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
