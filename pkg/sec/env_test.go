package sec_test

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/krewire/krewire/pkg/sec"
)

func TestSanitizeChildEnv_ExcludesParentSecrets(t *testing.T) {
	parent := []string{
		"PATH=/bin:/usr/bin",
		"USER=developer",
		"HOME=/home/developer",
		"AWS_SECRET_ACCESS_KEY=AKIAIOSFODNN7EXAMPLE/secret123",
		"DATABASE_URL=postgres://user:super_secret_pw@db.internal:5432/prod",
		"OPENAI_API_KEY=sk-test12345678901234567890",
		"GITHUB_TOKEN=ghp_123456789012345678901234567890123456",
		"APP_PRIVATE_KEY=-----BEGIN RSA PRIVATE KEY-----\nMIIEowIBAAKCAQEA...\n-----END RSA PRIVATE KEY-----",
		"LANG=en_US.UTF-8",
	}

	sanitized, err := sec.SanitizeChildEnv(sec.EnvConfig{
		ParentEnv: parent,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sanitizedJoined := strings.Join(sanitized, "\n")

	// Verify allowed variables exist
	if !strings.Contains(sanitizedJoined, "PATH=/bin:/usr/bin") {
		t.Errorf("expected PATH in sanitized env, got:\n%s", sanitizedJoined)
	}
	if !strings.Contains(sanitizedJoined, "USER=developer") {
		t.Errorf("expected USER in sanitized env, got:\n%s", sanitizedJoined)
	}
	if !strings.Contains(sanitizedJoined, "HOME=/home/developer") {
		t.Errorf("expected HOME in sanitized env, got:\n%s", sanitizedJoined)
	}

	// Verify sensitive variables are strictly absent
	forbidden := []string{
		"AWS_SECRET_ACCESS_KEY",
		"DATABASE_URL",
		"OPENAI_API_KEY",
		"GITHUB_TOKEN",
		"APP_PRIVATE_KEY",
		"super_secret_pw",
		"sk-test",
		"ghp_",
	}
	for _, f := range forbidden {
		if strings.Contains(sanitizedJoined, f) {
			t.Errorf("security violation: found leaked secret token %q in sanitized env:\n%s", f, sanitizedJoined)
		}
	}
}

func TestSanitizeChildEnv_CustomAllowlist(t *testing.T) {
	parent := []string{
		"PATH=/bin:/usr/bin",
		"USER=developer",
		"FEATURE_FLAG=enabled",
		"RANDOM_VAR=value",
	}

	sanitized, err := sec.SanitizeChildEnv(sec.EnvConfig{
		ParentEnv: parent,
		Allowlist: []string{"PATH", "FEATURE_FLAG"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sanitizedJoined := strings.Join(sanitized, "\n")
	if !strings.Contains(sanitizedJoined, "FEATURE_FLAG=enabled") {
		t.Errorf("expected FEATURE_FLAG in custom allowlist, got:\n%s", sanitizedJoined)
	}
	if strings.Contains(sanitizedJoined, "RANDOM_VAR") {
		t.Errorf("expected RANDOM_VAR to be excluded, got:\n%s", sanitizedJoined)
	}
	if strings.Contains(sanitizedJoined, "USER=") {
		t.Errorf("expected USER to be excluded when not in custom allowlist, got:\n%s", sanitizedJoined)
	}
}

func TestSanitizeChildEnv_BlocksReservedSecurityOverrides(t *testing.T) {
	reserved := []string{
		"LD_PRELOAD",
		"DYLD_INSERT_LIBRARIES",
		"BASH_ENV",
		"NODE_OPTIONS",
		"PYTHONPATH",
	}

	for _, k := range reserved {
		_, err := sec.SanitizeChildEnv(sec.EnvConfig{
			ParentEnv: []string{"PATH=/bin"},
			Overrides: map[string]string{
				k: "/path/to/malicious.so",
			},
		})
		if err == nil {
			t.Errorf("expected error when overriding reserved key %s, got nil", k)
		}
		if !errors.Is(err, sec.ErrReservedEnvOverride) {
			t.Errorf("expected ErrReservedEnvOverride for %s, got: %v", k, err)
		}
	}
}

func TestSanitizeChildEnv_BlocksSensitiveOverrides(t *testing.T) {
	// 1. Sensitive key override attempt
	_, err := sec.SanitizeChildEnv(sec.EnvConfig{
		ParentEnv: []string{"PATH=/bin"},
		Overrides: map[string]string{
			"MY_API_KEY": "some-key",
		},
	})
	if err == nil || !errors.Is(err, sec.ErrSensitiveEnvOverride) {
		t.Fatalf("expected ErrSensitiveEnvOverride for sensitive key, got: %v", err)
	}

	// 2. Sensitive value in override attempt
	_, err = sec.SanitizeChildEnv(sec.EnvConfig{
		ParentEnv: []string{"PATH=/bin"},
		Overrides: map[string]string{
			"CONFIG": "sk-proj-123456789012345678901234",
		},
	})
	if err == nil || !errors.Is(err, sec.ErrSensitiveEnvOverride) {
		t.Fatalf("expected ErrSensitiveEnvOverride for sensitive token value, got: %v", err)
	}
}

func TestSanitizeChildEnv_BaselineDefaults(t *testing.T) {
	tmp := t.TempDir()
	sanitized, err := sec.SanitizeChildEnv(sec.EnvConfig{
		ParentEnv: []string{}, // Empty parent
		Workspace: tmp,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sanitizedJoined := strings.Join(sanitized, "\n")
	if !strings.Contains(sanitizedJoined, "PATH=") {
		t.Errorf("expected default fallback PATH, got:\n%s", sanitizedJoined)
	}
	if !strings.Contains(sanitizedJoined, "LANG=C.UTF-8") {
		t.Errorf("expected default LANG=C.UTF-8, got:\n%s", sanitizedJoined)
	}
	if !strings.Contains(sanitizedJoined, "TMPDIR="+os.TempDir()) {
		t.Errorf("expected default TMPDIR, got:\n%s", sanitizedJoined)
	}
	if !strings.Contains(sanitizedJoined, "PWD="+tmp) {
		t.Errorf("expected PWD="+tmp+", got:\n%s", sanitizedJoined)
	}
}

func TestRedactSensitiveOutput_Comprehensive(t *testing.T) {
	inputs := []struct {
		raw      string
		expected string
	}{
		{
			raw:      "Connecting to postgres://admin:secretPass123@localhost:5432/mydb",
			expected: "Connecting to postgres://admin:[REDACTED_SECRET]@localhost:5432/mydb",
		},
		{
			raw:      "GitHub token: ghp_123456789012345678901234567890123456 in header",
			expected: "GitHub token: [REDACTED_SECRET] in header",
		},
		{
			raw:      "AWS Key: AKIA1234567890ABCDEF",
			expected: "AWS Key: [REDACTED_SECRET]",
		},
		{
			raw:      "OpenAI Key: sk-proj-12345678901234567890123456",
			expected: "OpenAI Key: [REDACTED_SECRET]",
		},
		{
			raw:      "export client_secret='topsecret1234'",
			expected: "export client_secret=[REDACTED_SECRET]",
		},
	}

	for _, tc := range inputs {
		redacted := sec.RedactSensitiveOutput(tc.raw)
		if redacted != tc.expected {
			t.Errorf("redaction mismatch:\ngot:      %q\nexpected: %q", redacted, tc.expected)
		}
	}
}
