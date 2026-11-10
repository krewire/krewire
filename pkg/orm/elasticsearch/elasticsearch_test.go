package elasticsearch_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/krewire/krewire/pkg/orm"
	"github.com/krewire/krewire/pkg/orm/elasticsearch"
)

type LogEntry struct {
	Level   string `json:"level"`
	Message string `json:"message"`
	Host    string `json:"host"`
}

func TestElasticsearchProductionCapabilities(t *testing.T) {
	var requestCount atomic.Uint64

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount.Add(1)
		w.Header().Set("Content-Type", "application/json")

		// 1. Cluster Health (GET /_cluster/health)
		if r.Method == http.MethodGet && r.URL.Path == "/_cluster/health" {
			_ = json.NewEncoder(w).Encode(orm.ClusterHealthResponse{
				ClusterName:         "krewire-es-cluster",
				Status:              "green",
				NumberOfNodes:       3,
				NumberOfDataNodes:   3,
				ActivePrimaryShards: 12,
				ActiveShards:        24,
			})
			return
		}

		// 2. Index Exists (HEAD /logs-2026)
		if r.Method == http.MethodHead && r.URL.Path == "/logs-2026" {
			w.WriteHeader(http.StatusOK)
			return
		}

		// 3. Document Exists (HEAD /logs-2026/_doc/log-101)
		if r.Method == http.MethodHead && r.URL.Path == "/logs-2026/_doc/log-101" {
			w.WriteHeader(http.StatusOK)
			return
		}

		// 4. Bulk API (POST /_bulk)
		if r.Method == http.MethodPost && r.URL.Path == "/_bulk" {
			_ = json.NewEncoder(w).Encode(orm.BulkResponse{
				Took:   25,
				Errors: false,
				Items: []map[string]orm.BulkItemResult{
					{"index": {Index: "logs-2026", ID: "log-1", Status: 201, Result: "created"}},
					{"index": {Index: "logs-2026", ID: "log-2", Status: 201, Result: "created"}},
					{"delete": {Index: "logs-2026", ID: "log-old", Status: 200, Result: "deleted"}},
				},
			})
			return
		}

		// 5. Update API (POST /logs-2026/_update/log-101)
		if r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/_update/") {
			_ = json.NewEncoder(w).Encode(orm.IndexResponse{
				Index:   "logs-2026",
				ID:      "log-101",
				Version: 2,
				Result:  "updated",
			})
			return
		}

		// 6. Refresh Index (POST /logs-2026/_refresh)
		if r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/_refresh") {
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"_shards": map[string]int{"total": 2, "successful": 2, "failed": 0},
			})
			return
		}

		// 7. Aliases (POST /_aliases)
		if r.Method == http.MethodPost && r.URL.Path == "/_aliases" {
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]bool{"acknowledged": true})
			return
		}

		// 8. Delete Index (DELETE /logs-2026)
		if r.Method == http.MethodDelete && r.URL.Path == "/logs-2026" {
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]bool{"acknowledged": true})
			return
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	ctx := context.Background()
	adapter, err := elasticsearch.Open(ts.URL)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer adapter.Close()

	// 1. Cluster Health
	health, err := adapter.ClusterHealth(ctx)
	if err != nil {
		t.Fatalf("ClusterHealth failed: %v", err)
	}
	if health.Status != "green" || health.ClusterName != "krewire-es-cluster" {
		t.Errorf("unexpected cluster health: %+v", health)
	}

	// 2. Index Exists
	idxExists, err := adapter.IndexExists(ctx, "logs-2026")
	if err != nil {
		t.Fatalf("IndexExists failed: %v", err)
	}
	if !idxExists {
		t.Error("expected index logs-2026 to exist")
	}

	// 3. Document Exists
	docExists, err := adapter.Exists(ctx, "logs-2026", "log-101")
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}
	if !docExists {
		t.Error("expected document log-101 to exist")
	}

	// 4. Bulk API
	bulkRes, err := adapter.Bulk(ctx, []orm.BulkItem{
		{Action: orm.BulkActionIndex, Index: "logs-2026", ID: "log-1", Doc: LogEntry{Level: "INFO", Message: "Server starting", Host: "node-1"}},
		{Action: orm.BulkActionIndex, Index: "logs-2026", ID: "log-2", Doc: LogEntry{Level: "WARN", Message: "High memory usage", Host: "node-2"}},
		{Action: orm.BulkActionDelete, Index: "logs-2026", ID: "log-old"},
	})
	if err != nil {
		t.Fatalf("Bulk failed: %v", err)
	}
	if bulkRes.Errors || len(bulkRes.Items) != 3 {
		t.Errorf("unexpected bulk response: %+v", bulkRes)
	}

	// 5. Update API
	upRes, err := adapter.Update(ctx, "logs-2026", "log-101", map[string]interface{}{"status": "resolved"})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if upRes.Result != "updated" {
		t.Errorf("expected updated result, got %s", upRes.Result)
	}

	// 6. Refresh Index
	if err := adapter.RefreshIndex(ctx, "logs-2026"); err != nil {
		t.Fatalf("RefreshIndex failed: %v", err)
	}

	// 7. Aliases
	if err := adapter.CreateAlias(ctx, "logs-2026", "current-logs"); err != nil {
		t.Fatalf("CreateAlias failed: %v", err)
	}
	if err := adapter.DeleteAlias(ctx, "logs-2026", "current-logs"); err != nil {
		t.Fatalf("DeleteAlias failed: %v", err)
	}

	// 8. Delete Index
	if err := adapter.DeleteIndex(ctx, "logs-2026"); err != nil {
		t.Fatalf("DeleteIndex failed: %v", err)
	}
}

func TestMultiNodeRoundRobin(t *testing.T) {
	var count1, count2 atomic.Uint64

	ts1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count1.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts1.Close()

	ts2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count2.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts2.Close()

	ctx := context.Background()
	adapter, err := elasticsearch.Open(ts1.URL, ts2.URL)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer adapter.Close()

	for i := 0; i < 6; i++ {
		if err := adapter.Ping(ctx); err != nil {
			t.Fatalf("Ping %d failed: %v", i, err)
		}
	}

	if count1.Load() != 3 || count2.Load() != 3 {
		t.Errorf("expected balanced round-robin (3, 3), got (%d, %d)", count1.Load(), count2.Load())
	}
}
