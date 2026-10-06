package sec

// Tests for PII masking and safe diagnostic redaction.
import (
	"bytes"
	"context"
	"log/slog"
	"reflect"
	"strings"
	"testing"
)

func TestMaskPII(t *testing.T) {
	input := "email alice@example.com phone +1 (555) 123-4567 card 4111 1111 1111 1111 ip 192.168.1.42"
	want := "email a***@example.com phone ***-***-4567 card ****1111 ip 192.168.xxx.xxx"
	if got := MaskPII(input); got != want {
		t.Fatalf("MaskPII() = %q, want %q", got, want)
	}
}

func TestMaskPIIMapDoesNotMutateAndRedactsNestedValues(t *testing.T) {
	input := map[string]any{
		"email":    "alice@example.com",
		"password": "do-not-log",
		"nested": map[string]any{
			"phone": "+1 555 123 4567",
		},
		"items": []any{"bob@example.com"},
	}
	original := map[string]any{
		"email":    "alice@example.com",
		"password": "do-not-log",
		"nested":   map[string]any{"phone": "+1 555 123 4567"},
		"items":    []any{"bob@example.com"},
	}

	got := MaskPIIMap(input)
	if !reflect.DeepEqual(input, original) {
		t.Fatalf("MaskPIIMap mutated input: %#v", input)
	}
	if got["password"] != "[REDACTED]" {
		t.Fatalf("password = %#v", got["password"])
	}
	if got["email"] != "a***@example.com" {
		t.Fatalf("email = %#v", got["email"])
	}
	nested := got["nested"].(map[string]any)
	if nested["phone"] != "***-***-4567" {
		t.Fatalf("nested phone = %#v", nested["phone"])
	}
}

func TestMaskPIILeavesNonPIIText(t *testing.T) {
	if got := MaskPII("request completed with status 200"); got != "request completed with status 200" {
		t.Fatalf("unexpected masking: %q", got)
	}
}

func TestMaskPIIIPv6(t *testing.T) {
	if got := MaskPII("client [2001:db8::1]"); got != "client [[IPv6 REDACTED]]" {
		t.Fatalf("MaskPII IPv6 = %q", got)
	}
}

func TestMaskPIIAttrsMasksGroupsAndSecrets(t *testing.T) {
	attrs := MaskPIIAttrs(
		slog.String("email", "alice@example.com"),
		slog.String("token", "secret-token"),
		slog.Group("request", slog.String("ip", "2001:db8::1")),
	)
	if attrs[0].Value.String() != "a***@example.com" {
		t.Fatalf("email = %q", attrs[0].Value.String())
	}
	if attrs[1].Value.String() != "[REDACTED]" {
		t.Fatalf("token = %q", attrs[1].Value.String())
	}
	group := attrs[2].Value.Group()
	if group[0].Value.String() != "[IPv6 REDACTED]" {
		t.Fatalf("nested IPv6 = %q", group[0].Value.String())
	}
}

func TestNewPIIHandlerMasksRecordsAndBoundAttrs(t *testing.T) {
	var output bytes.Buffer
	handler := NewPIIHandler(slog.NewTextHandler(&output, nil))
	logger := slog.New(handler).With("request_email", "bound@example.com")
	logger.LogAttrs(context.Background(), slog.LevelInfo, "request", slog.String("email", "alice@example.com"), slog.String("token", "secret"))
	text := output.String()
	for _, leaked := range []string{"bound@example.com", "alice@example.com", "secret"} {
		if strings.Contains(text, leaked) {
			t.Fatalf("PII leaked in log output: %q", text)
		}
	}
	for _, masked := range []string{"a***@example.com", "[REDACTED]"} {
		if !strings.Contains(text, masked) {
			t.Fatalf("masked value %q missing from log output: %q", masked, text)
		}
	}
}
