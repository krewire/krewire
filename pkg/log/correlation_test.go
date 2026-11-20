package log

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"
)

func TestCorrelationContext(t *testing.T) {
	// 1. Empty context
	req, trc := CorrelationFrom(context.Background())
	if req != "" || trc != "" {
		t.Errorf("expected empty correlation from background context, got req=%q, trc=%q", req, trc)
	}

	// 2. Injected context
	ctx := WithCorrelation(context.Background(), "req-4fc69929", "trc-a77b6ed")
	req, trc = CorrelationFrom(ctx)
	if req != "req-4fc69929" || trc != "trc-a77b6ed" {
		t.Errorf("unexpected correlation values: req=%q, trc=%q", req, trc)
	}

	// 3. CorrelationAttrs
	attrs := CorrelationAttrs(ctx)
	if len(attrs) != 2 {
		t.Fatalf("expected 2 correlation attributes, got %d", len(attrs))
	}
	if attrs[0].Key != AttrRequestID || attrs[0].Value.String() != "req-4fc69929" {
		t.Errorf("invalid request_id attribute: %+v", attrs[0])
	}
	if attrs[1].Key != AttrTraceID || attrs[1].Value.String() != "trc-a77b6ed" {
		t.Errorf("invalid trace_id attribute: %+v", attrs[1])
	}
}

func TestWithContextLogger(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	baseLogger := slog.New(handler)

	ctx := WithCorrelation(context.Background(), "req-987", "trc-543")
	logger := WithContext(baseLogger, ctx)

	logger.InfoContext(ctx, "processing order", "order_id", "ord-111")

	var record map[string]any
	if err := json.Unmarshal(buf.Bytes(), &record); err != nil {
		t.Fatalf("failed to parse JSON log output: %v\nOutput: %s", err, buf.String())
	}

	if record["msg"] != "processing order" {
		t.Errorf("unexpected msg: %v", record["msg"])
	}
	if record[AttrRequestID] != "req-987" {
		t.Errorf("expected %s='req-987', got %v", AttrRequestID, record[AttrRequestID])
	}
	if record[AttrTraceID] != "trc-543" {
		t.Errorf("expected %s='trc-543', got %v", AttrTraceID, record[AttrTraceID])
	}
	if record["order_id"] != "ord-111" {
		t.Errorf("expected order_id='ord-111', got %v", record["order_id"])
	}
}
