package sec

import (
	"bytes"
	"strings"
	"testing"
)

func TestMaskingWriter_Basic(t *testing.T) {
	var buf bytes.Buffer
	w := NewMaskingWriter(&buf, "secret123")
	w.Write([]byte("hello secret123 world"))
	w.Flush()
	if got := buf.String(); got != "hello [REDACTED] world" {
		t.Fatalf("got %q", got)
	}
}

func TestMaskingWriter_SplitBoundary(t *testing.T) {
	var buf bytes.Buffer
	w := NewMaskingWriter(&buf, "secret")
	w.Write([]byte("sec"))
	w.Write([]byte("ret"))
	w.Flush()
	if got := buf.String(); !strings.Contains(got, "[REDACTED]") || strings.Contains(got, "secret") {
		t.Fatalf("got %q", got)
	}
}

func TestMaskingWriter_NoSecret(t *testing.T) {
	var buf bytes.Buffer
	w := NewMaskingWriter(&buf, "xxx")
	w.Write([]byte("hello world"))
	w.Flush()
	if got := buf.String(); got != "hello world" {
		t.Fatalf("got %q", got)
	}
}
