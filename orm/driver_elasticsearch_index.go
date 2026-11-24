package orm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/krewire/krewire/errs"
)

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
