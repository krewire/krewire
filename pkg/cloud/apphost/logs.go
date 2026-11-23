package apphost

import (
	"bytes"
	"fmt"
	"io"
	"sync"
	"time"
)

// Log streams.
const (
	StreamStdout = "stdout"
	StreamStderr = "stderr"
	// StreamSystem carries orchestrator messages about a resource.
	StreamSystem = "system"
)

// LogLine is one line of aggregated output.
type LogLine struct {
	Time     time.Time `json:"time"`
	Resource string    `json:"resource"`
	Stream   string    `json:"stream"`
	Text     string    `json:"text"`
}

// logHub keeps a bounded per-resource history and fans lines out to the
// aggregated sink.
type logHub struct {
	mu    sync.Mutex
	max   int
	lines map[string][]LogLine
	sink  io.Writer
	hook  func(LogLine)
}

func newLogHub(max int, sink io.Writer, hook func(LogLine)) *logHub {
	if max <= 0 {
		max = 1000
	}
	return &logHub{max: max, lines: map[string][]LogLine{}, sink: sink, hook: hook}
}

func (h *logHub) add(l LogLine) {
	h.mu.Lock()
	buf := append(h.lines[l.Resource], l)
	if len(buf) > h.max {
		buf = append([]LogLine(nil), buf[len(buf)-h.max:]...)
	}
	h.lines[l.Resource] = buf
	if h.sink != nil {
		fmt.Fprintf(h.sink, "%s | %s\n", l.Resource, l.Text)
	}
	h.mu.Unlock()
	if h.hook != nil {
		h.hook(l)
	}
}

func (h *logHub) system(res, format string, args ...any) {
	h.add(LogLine{Time: time.Now(), Resource: res, Stream: StreamSystem, Text: fmt.Sprintf(format, args...)})
}

// tail returns up to n recent lines of res (all when n <= 0).
func (h *logHub) tail(res string, n int) []LogLine {
	h.mu.Lock()
	defer h.mu.Unlock()
	buf := h.lines[res]
	if n > 0 && len(buf) > n {
		buf = buf[len(buf)-n:]
	}
	return append([]LogLine(nil), buf...)
}

// writer returns an io.Writer that splits output into lines for res.
func (h *logHub) writer(res, stream string) *lineWriter {
	return &lineWriter{hub: h, res: res, stream: stream}
}

type lineWriter struct {
	mu     sync.Mutex
	hub    *logHub
	res    string
	stream string
	buf    []byte
}

func (w *lineWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.buf = append(w.buf, p...)
	for {
		i := bytes.IndexByte(w.buf, '\n')
		if i < 0 {
			break
		}
		text := string(bytes.TrimRight(w.buf[:i], "\r"))
		w.buf = w.buf[i+1:]
		w.hub.add(LogLine{Time: time.Now(), Resource: w.res, Stream: w.stream, Text: text})
	}
	return len(p), nil
}

// flush emits a trailing partial line.
func (w *lineWriter) flush() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if len(w.buf) > 0 {
		w.hub.add(LogLine{Time: time.Now(), Resource: w.res, Stream: w.stream, Text: string(w.buf)})
		w.buf = nil
	}
}
