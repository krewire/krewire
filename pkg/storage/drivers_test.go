package storage_test

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/krewire/krewire/pkg/storage"
)

func TestMemoryStore_Operations(t *testing.T) {
	store := storage.NewMemoryStore()

	// Put
	err := store.Put("docs/readme.txt", []byte("Hello Krewire"))
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	// Get
	data, err := store.Get("docs/readme.txt")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if string(data) != "Hello Krewire" {
		t.Errorf("expected Hello Krewire, got %s", string(data))
	}

	// Stat
	info, err := store.Stat("docs/readme.txt")
	if err != nil {
		t.Fatalf("Stat failed: %v", err)
	}
	if info.Size != int64(len("Hello Krewire")) {
		t.Errorf("expected size %d, got %d", len("Hello Krewire"), info.Size)
	}

	// Delete
	err = store.Delete("docs/readme.txt")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Get after delete must return ErrNotFound
	_, err = store.Get("docs/readme.txt")
	if err != storage.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestS3Store_OperationsWithMockServer(t *testing.T) {
	var mu sync.RWMutex
	s3Objects := make(map[string][]byte)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "AWS4-HMAC-SHA256") {
			t.Errorf("expected AWS4-HMAC-SHA256 signature in Authorization header, got %s", auth)
		}

		key := strings.TrimPrefix(r.URL.Path, "/test-bucket/")

		switch r.Method {
		case http.MethodPut:
			body, _ := io.ReadAll(r.Body)
			s3Objects[key] = body
			w.WriteHeader(http.StatusOK)
		case http.MethodGet:
			body, ok := s3Objects[key]
			if !ok {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(body)
		case http.MethodHead:
			body, ok := s3Objects[key]
			if !ok {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Length", string(rune(len(body))))
			w.WriteHeader(http.StatusOK)
		case http.MethodDelete:
			delete(s3Objects, key)
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusBadRequest)
		}
	}))
	defer ts.Close()

	store, err := storage.NewS3Store(storage.S3Config{
		Endpoint:        ts.URL,
		Bucket:          "test-bucket",
		Region:          "us-east-1",
		AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
		SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
		UsePathStyle:    true,
		HTTPClient:      ts.Client(),
	})
	if err != nil {
		t.Fatalf("NewS3Store failed: %v", err)
	}

	// Put
	payload := []byte("Cloud Object Payload")
	err = store.Put("uploads/image.png", payload)
	if err != nil {
		t.Fatalf("S3 Put failed: %v", err)
	}

	// Get
	got, err := store.Get("uploads/image.png")
	if err != nil {
		t.Fatalf("S3 Get failed: %v", err)
	}
	if !bytes.Equal(got, payload) {
		t.Errorf("expected %s, got %s", string(payload), string(got))
	}

	// Delete
	err = store.Delete("uploads/image.png")
	if err != nil {
		t.Fatalf("S3 Delete failed: %v", err)
	}

	// Get after Delete
	_, err = store.Get("uploads/image.png")
	if err != storage.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}
