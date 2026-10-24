package orm_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/krewire/krewire/packages/orm"
)

type ESDoc struct {
	Title   string `json:"title"`
	Content string `json:"content"`
	Author  string `json:"author"`
}

func TestElasticsearch_Operations(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// 1. Ping (HEAD /)
		if r.Method == http.MethodHead && r.URL.Path == "/" {
			w.WriteHeader(http.StatusOK)
			return
		}

		// 2. Index doc (PUT /articles/_doc/doc-1)
		if r.Method == http.MethodPut && r.URL.Path == "/articles/_doc/doc-1" {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"_index":   "articles",
				"_id":      "doc-1",
				"_version": 1,
				"result":   "created",
			})
			return
		}

		// 3. Get doc (GET /articles/_doc/doc-1)
		if r.Method == http.MethodGet && r.URL.Path == "/articles/_doc/doc-1" {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"_index": "articles",
				"_id":    "doc-1",
				"found":  true,
				"_source": map[string]interface{}{
					"title":   "Building Pure-Go ORM",
					"content": "A guide to multi-driver ORM architecture in Krewire",
					"author":  "Reasvyn",
				},
			})
			return
		}

		// 4. Search (POST /articles/_search)
		if r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/_search") {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"hits": map[string]interface{}{
					"total": map[string]interface{}{
						"value": 1,
					},
					"hits": []map[string]interface{}{
						{
							"_index": "articles",
							"_id":    "doc-1",
							"_score": 1.5,
							"_source": map[string]interface{}{
								"title": "Building Pure-Go ORM",
							},
						},
					},
				},
			})
			return
		}

		// 5. Delete doc (DELETE /articles/_doc/doc-1)
		if r.Method == http.MethodDelete && r.URL.Path == "/articles/_doc/doc-1" {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"_index": "articles",
				"_id":    "doc-1",
				"result": "deleted",
			})
			return
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	adapter, err := orm.NewElasticsearch(orm.ElasticsearchConfig{
		Addresses: []string{ts.URL},
		Client:    ts.Client(),
	})
	if err != nil {
		t.Fatalf("NewElasticsearch failed: %v", err)
	}
	defer adapter.Close()

	if adapter.Name() != "elasticsearch" {
		t.Errorf("expected name elasticsearch, got %s", adapter.Name())
	}
	if adapter.Type() != orm.AdapterTypeSearch {
		t.Errorf("expected search adapter type, got %s", adapter.Type())
	}

	ctx := context.Background()

	// Ping
	if err := adapter.Ping(ctx); err != nil {
		t.Fatalf("Ping failed: %v", err)
	}

	// Index
	idxRes, err := adapter.Index(ctx, "articles", "doc-1", ESDoc{
		Title:   "Building Pure-Go ORM",
		Content: "A guide to multi-driver ORM architecture in Krewire",
		Author:  "Reasvyn",
	})
	if err != nil {
		t.Fatalf("Index failed: %v", err)
	}
	if idxRes.ID != "doc-1" || idxRes.Result != "created" {
		t.Errorf("unexpected index result: %+v", idxRes)
	}

	// Get
	var doc ESDoc
	err = adapter.Get(ctx, "articles", "doc-1", &doc)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if doc.Title != "Building Pure-Go ORM" || doc.Author != "Reasvyn" {
		t.Errorf("unexpected doc: %+v", doc)
	}

	// Search
	sres, err := adapter.Search(ctx, orm.SearchRequest{
		Index: "articles",
		Query: map[string]interface{}{
			"match": map[string]interface{}{
				"title": "ORM",
			},
		},
	})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if sres.TotalHits != 1 || len(sres.Hits) != 1 {
		t.Fatalf("expected 1 hit, got %d", sres.TotalHits)
	}
	if sres.Hits[0].ID != "doc-1" {
		t.Errorf("expected hit id doc-1, got %s", sres.Hits[0].ID)
	}

	// Delete
	err = adapter.Delete(ctx, "articles", "doc-1")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
}
