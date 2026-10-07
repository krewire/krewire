package sec

// Tests for SEC-A10-001.
import (
	"errors"
	"testing"
)

func TestValidateOutboundURL(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		allowed []string
		wantErr error
	}{
		{name: "allowed HTTPS", raw: "https://api.example.test/v1", allowed: []string{"api.example.test"}},
		{name: "rejects non HTTP scheme", raw: "file:///etc/passwd", allowed: []string{"api.example.test"}, wantErr: ErrOutboundURLInvalid},
		{name: "rejects user info", raw: "https://user:pass@api.example.test", allowed: []string{"api.example.test"}, wantErr: ErrOutboundURLInvalid},
		{name: "rejects non allowlisted host", raw: "https://attacker.example", allowed: []string{"api.example.test"}, wantErr: ErrOutboundHostDenied},
		{name: "rejects loopback literal", raw: "https://127.0.0.1/admin", allowed: []string{"127.0.0.1"}, wantErr: ErrOutboundHostDenied},
		{name: "rejects private literal", raw: "https://10.0.0.1", allowed: []string{"10.0.0.1"}, wantErr: ErrOutboundHostDenied},
		{name: "rejects localhost hostname", raw: "http://localhost/metrics", allowed: []string{"localhost"}, wantErr: ErrOutboundHostDenied},
		{name: "rejects cloud metadata hostname", raw: "http://metadata.google.internal/computeMetadata/v1/", allowed: []string{"metadata.google.internal"}, wantErr: ErrOutboundHostDenied},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateOutboundURL(tt.raw, tt.allowed...)
			if tt.wantErr == nil {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want errors.Is(..., %v)", err, tt.wantErr)
			}
		})
	}
}

func TestNewSafeHTTPClient(t *testing.T) {
	client := NewSafeHTTPClient(
		WithAllowedHosts("example.com"),
		WithDialTimeout(5),
	)
	if client == nil || client.Transport == nil {
		t.Fatal("expected non-nil safe client and transport")
	}

	// Outbound request to blocked host should fail at dial time
	_, err := client.Get("http://localhost:8080/metrics")
	if err == nil {
		t.Fatal("expected error connecting to localhost via safe client, got nil")
	}
}
