package orm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

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
