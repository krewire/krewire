package observe

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"testing"

	klog "github.com/krewire/krewire/log"
)

func TestIsSensitiveKey(t *testing.T) {
	for _, k := range []string{"password", "DB-Password", "access_token", "Authorization", "api.key", "client_secret", "user_email", "set_cookie", "DATABASE_DSN", "aws_access_key_id"} {
		if !IsSensitiveKey(k) {
			t.Errorf("%q should be sensitive", k)
		}
	}
	for _, k := range []string{"input_tokens", "status", "listen_address", "session_id", "task_id", "duration"} {
		if IsSensitiveKey(k) {
			t.Errorf("%q should not be sensitive", k)
		}
	}
}

func TestRedactString(t *testing.T) {
	in := "dial postgres://app:hunter2@db:5432/x failed; Authorization: Bearer abc.def; token=xyz&a=1 mail ops@example.com"
	out := RedactString(in)
	for _, leak := range []string{"hunter2", "abc.def", "xyz", "ops@example.com"} {
		if strings.Contains(out, leak) {
			t.Errorf("leaked %q in %q", leak, out)
		}
	}
	if RedactString("") != "" {
		t.Fatal("empty")
	}
	if got := RedactURL("https://u:p@h/x?q=secret&id=1"); strings.Contains(got, "secret") || strings.Contains(got, "u:p") || !strings.Contains(got, "/x?") {
		t.Fatalf("RedactURL %q", got)
	}
	if got := RedactURL("://bad url token=abc"); strings.Contains(got, "abc") {
		t.Fatalf("RedactURL fallback %q", got)
	}
	m := RedactMap(map[string]string{"API_KEY": "k", "HOME": "/home/x", "DB_URL": "mysql://a:b@h/d"})
	if m["API_KEY"] != Redacted || m["HOME"] != "/home/x" || strings.Contains(m["DB_URL"], "a:b") {
		t.Fatalf("RedactMap %v", m)
	}
}

func TestHandlerContract(t *testing.T) {
	var buf bytes.Buffer
	l := NewLogger(slog.New(slog.NewJSONHandler(&buf, nil)), Identity{Service: "billing", Workload: "service", Resource: "api", Version: "1.2.0"})
	ctx, _ := StartSpan(WithIDs(context.Background(), IDs{RequestID: "r1", TaskID: "t1"}))
	l.With("api_key", "k1", slog.Group("db", slog.String("password", "p"))).InfoContext(ctx, "charged token=zzz",
		"amount_cents", 4500, "err", errors.New("dial mysql://u:pw@h/db"), slog.Group("g", slog.String("email", "a@b.co")))
	var rec map[string]any
	if err := json.Unmarshal(buf.Bytes(), &rec); err != nil {
		t.Fatal(err)
	}
	for k, want := range map[string]string{FieldService: "billing", FieldWorkload: "service", FieldResource: "api", FieldVersion: "1.2.0", FieldRequestID: "r1", FieldTaskID: "t1", "api_key": Redacted} {
		if rec[k] != want {
			t.Errorf("%s = %v, want %v", k, rec[k], want)
		}
	}
	if rec[FieldTraceID] == nil || rec[FieldSpanID] == nil {
		t.Errorf("missing trace fields: %v", rec)
	}
	for _, leak := range []string{"zzz", "u:pw", "a@b.co", `"p"`, "k1"} {
		if strings.Contains(buf.String(), leak) {
			t.Errorf("leaked %q: %s", leak, buf.String())
		}
	}
	if !NewHandler(nil, Identity{}).Enabled(context.Background(), slog.LevelError) {
		t.Fatal("enabled")
	}
	buf.Reset()
	NewLogger(slog.New(slog.NewJSONHandler(&buf, nil)), Identity{}).WithGroup("x").Info("m", "k", 1)
	if !strings.Contains(buf.String(), `"x":{"k":1}`) {
		t.Fatalf("group: %s", buf.String())
	}
	if NewLogger(nil, Identity{}) == nil {
		t.Fatal("nil base")
	}
}

func TestContextAttrsFallsBackToPkgLog(t *testing.T) {
	ctx := WithIDs(context.Background(), IDs{})
	if len(ContextAttrs(ctx)) != 0 {
		t.Fatal("expected no attrs")
	}
	attrs := ContextAttrs(klog.WithCorrelation(context.Background(), "", "legacy-trace"))
	if len(attrs) != 1 || attrs[0].Value.String() != "legacy-trace" {
		t.Fatalf("fallback attrs %v", attrs)
	}
}
