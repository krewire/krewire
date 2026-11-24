// Package hmr provides live reload and hot module replacement for the SSG
// dev server (KWF-209JV). It serves a WebSocket endpoint at /__krewire/hmr
// that broadcasts reload/style-update/content-update events to connected
// browsers when incremental builds detect changes.
package hmr

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const DefaultHeartbeatInterval = 25 * time.Second

// ClientScript is the client-side JavaScript snippet that connects to the
// SSE live reload endpoint at /__krewire/hmr and triggers hot reload or
// stylesheet updates.
const ClientScript = `(function() {
  if (!window.EventSource) return;
  var es = new EventSource('/__krewire/hmr');
  es.addEventListener('reload', function() {
    console.log('[krewire:hmr] reloading page...');
    window.location.reload();
  });
  es.addEventListener('style-update', function() {
    console.log('[krewire:hmr] updating stylesheets...');
    var links = document.querySelectorAll('link[rel="stylesheet"]');
    for (var i = 0; i < links.length; i++) {
      var href = links[i].getAttribute('href');
      if (href) {
        var base = href.split('?')[0];
        links[i].setAttribute('href', base + '?v=' + Date.now());
      }
    }
  });
  es.onerror = function() {
    // EventSource handles reconnection automatically
  };
})();`

// EventType is the kind of HMR event (KWF-209JV).
type EventType string

const (
	// EventReload signals a full page reload.
	EventReload EventType = "reload"
	// EventStyleUpdate signals a CSS-only update (no reload needed).
	EventStyleUpdate EventType = "style-update"
	// EventContentUpdate signals a content change with path.
	EventContentUpdate EventType = "content-update"
)

// Event is an HMR message sent to connected clients (KWF-209JV).
type Event struct {
	Type    EventType `json:"type"`
	Path    string    `json:"path,omitempty"`
	Message string    `json:"message,omitempty"`
}

// Server is an HMR WebSocket endpoint (KWF-209JV).
type Server struct {
	mu       sync.RWMutex
	clients  map[chan Event]struct{}
	eventLog []Event
}

// NewServer creates an HMR server (KWF-209JV).
func NewServer() *Server {
	return &Server{
		clients:  map[chan Event]struct{}{},
		eventLog: make([]Event, 0, 64),
	}
}

// Handler wraps an existing http.Handler, intercepting /__krewire/hmr and
// /__krewire/hmr.js, and injecting the HMR client script into served HTML documents.
func (s *Server) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/__krewire/hmr" || r.URL.Path == "/__krewire/hmr.js" {
			s.ServeHTTP(w, r)
			return
		}

		rw := &htmlInjectWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}
		next.ServeHTTP(rw, r)
		rw.flush()
	})
}

type htmlInjectWriter struct {
	http.ResponseWriter
	buf         bytes.Buffer
	statusCode  int
	wroteHeader bool
}

func (w *htmlInjectWriter) WriteHeader(code int) {
	if !w.wroteHeader {
		w.statusCode = code
		w.wroteHeader = true
	}
}

func (w *htmlInjectWriter) Write(b []byte) (int, error) {
	if !w.wroteHeader {
		w.wroteHeader = true
	}
	return w.buf.Write(b)
}

func (w *htmlInjectWriter) flush() {
	body := w.buf.Bytes()
	contentType := w.Header().Get("Content-Type")

	isHTML := strings.Contains(contentType, "text/html") ||
		(contentType == "" && len(body) > 0 && http.DetectContentType(body) == "text/html; charset=utf-8")

	if isHTML && len(body) > 0 {
		tag := []byte(`<script src="/__krewire/hmr.js" defer></script>`)
		bodyLower := bytes.ToLower(body)
		if idx := bytes.LastIndex(bodyLower, []byte("</body>")); idx >= 0 {
			var newBody bytes.Buffer
			newBody.Write(body[:idx])
			newBody.Write(tag)
			newBody.Write(body[idx:])
			body = newBody.Bytes()
		} else {
			body = append(body, tag...)
		}
		w.Header().Set("Content-Length", strconv.Itoa(len(body)))
	}

	w.ResponseWriter.WriteHeader(w.statusCode)
	_, _ = w.ResponseWriter.Write(body)
}

// ServeHTTP handles WebSocket upgrade requests (KWF-209JV).
// For simplicity, this uses Server-Sent Events (SSE) instead of raw
// WebSockets — SSE is sufficient for one-directional HMR broadcasts
// and requires no external dependency.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/__krewire/hmr.js" {
		w.Header().Set("Content-Type", "application/javascript")
		w.Header().Set("Cache-Control", "no-cache")
		_, _ = w.Write([]byte(ClientScript))
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	ch := make(chan Event, 16)
	s.mu.Lock()
	s.clients[ch] = struct{}{}
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.clients, ch)
		s.mu.Unlock()
		close(ch)
	}()

	// Send recent event log to catch up a reconnecting client
	s.mu.RLock()
	for _, ev := range s.eventLog {
		s.writeEvent(w, ev, flusher)
	}
	s.mu.RUnlock()

	// Heartbeat to keep connection alive
	ticker := time.NewTicker(DefaultHeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case ev, ok := <-ch:
			if !ok {
				return
			}
			s.writeEvent(w, ev, flusher)
		case <-ticker.C:
			fmt.Fprintf(w, ":\n\n")
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}

func (s *Server) writeEvent(w http.ResponseWriter, ev Event, flusher http.Flusher) {
	data, err := json.Marshal(ev)
	if err != nil {
		return
	}
	fmt.Fprintf(w, "event: %s\n", ev.Type)
	fmt.Fprintf(w, "data: %s\n\n", data)
	flusher.Flush()
}

// Broadcast sends an event to all connected clients (KWF-209JV).
func (s *Server) Broadcast(ev Event) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.eventLog = append(s.eventLog, ev)
	if len(s.eventLog) > 64 {
		s.eventLog = s.eventLog[len(s.eventLog)-64:]
	}
	for ch := range s.clients {
		select {
		case ch <- ev:
		default:
		}
	}
	slog.Debug("hmr broadcast", "type", ev.Type, "path", ev.Path)
}

// ClientCount returns the number of connected clients (for observability).
func (s *Server) ClientCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.clients)
}
