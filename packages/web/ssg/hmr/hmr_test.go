package hmr

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewServer(t *testing.T) {
	s := NewServer()
	if s == nil {
		t.Fatal("expected non-nil server")
	}
	if s.ClientCount() != 0 {
		t.Fatalf("expected 0 clients, got %d", s.ClientCount())
	}
}

func TestBroadcast_NoClients(t *testing.T) {
	s := NewServer()
	s.Broadcast(Event{Type: EventReload})
	if s.ClientCount() != 0 {
		t.Fatal("expected 0 clients")
	}
}

func TestServeHTTP_SSE(t *testing.T) {
	s := NewServer()
	req := httptest.NewRequest("GET", "/__krewire/hmr", nil)
	w := httptest.NewRecorder()

	// Use a channel to signal when handler is done
	done := make(chan struct{})
	go func() {
		s.ServeHTTP(w, req)
		close(done)
	}()

	// Give handler time to start
	time.Sleep(50 * time.Millisecond)

	if s.ClientCount() != 1 {
		t.Fatalf("expected 1 client, got %d", s.ClientCount())
	}

	s.Broadcast(Event{Type: EventReload, Path: "/index.html"})

	// Cancel request to stop handler
	cancelReq := req.Clone(req.Context())
	_ = cancelReq

	time.Sleep(50 * time.Millisecond)
}

func TestEvent_Types(t *testing.T) {
	tests := []struct {
		ev   Event
		want string
	}{
		{Event{Type: EventReload}, "reload"},
		{Event{Type: EventStyleUpdate}, "style-update"},
		{Event{Type: EventContentUpdate}, "content-update"},
	}
	for _, tt := range tests {
		if string(tt.ev.Type) != tt.want {
			t.Fatalf("event type = %q, want %q", tt.ev.Type, tt.want)
		}
	}
}

func TestEventLog_Truncation(t *testing.T) {
	s := NewServer()
	for i := 0; i < 100; i++ {
		s.Broadcast(Event{Type: EventReload})
	}
	s.mu.RLock()
	count := len(s.eventLog)
	s.mu.RUnlock()
	if count > 64 {
		t.Fatalf("event log should be truncated to 64, got %d", count)
	}
}

func TestServeHTTP_ClientScript(t *testing.T) {
	s := NewServer()
	req := httptest.NewRequest("GET", "/__krewire/hmr.js", nil)
	w := httptest.NewRecorder()

	s.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d", w.Code, http.StatusOK)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/javascript" {
		t.Errorf("content-type = %q, want application/javascript", ct)
	}
	if !strings.Contains(w.Body.String(), "/__krewire/hmr") {
		t.Errorf("expected body to contain /__krewire/hmr SSE endpoint")
	}
}

func TestHandler_InjectsHMRIntoHTML(t *testing.T) {
	s := NewServer()

	// Mock handler serving HTML
	htmlHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte("<!DOCTYPE html><html><body><h1>Hello</h1></body></html>"))
	})

	wrapped := s.Handler(htmlHandler)

	req := httptest.NewRequest("GET", "/index.html", nil)
	w := httptest.NewRecorder()
	wrapped.ServeHTTP(w, req)

	body := w.Body.String()
	if !strings.Contains(body, `<script src="/__krewire/hmr.js" defer></script></body>`) {
		t.Errorf("expected script injected before closing body, got: %s", body)
	}

	// Mock handler serving non-HTML (e.g. CSS)
	cssHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/css")
		_, _ = w.Write([]byte("body { color: red; }"))
	})
	wrappedCSS := s.Handler(cssHandler)

	reqCSS := httptest.NewRequest("GET", "/style.css", nil)
	wCSS := httptest.NewRecorder()
	wrappedCSS.ServeHTTP(wCSS, reqCSS)

	if strings.Contains(wCSS.Body.String(), "hmr.js") {
		t.Errorf("non-HTML response must not have HMR script injected")
	}
}
