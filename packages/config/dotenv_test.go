// Tests for KWL-2X1QZ
package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Spec: KWL-2X1QZ CFG-DOTV-002 Scope: Unit
func TestCFG_DOTV_002_ParseDotEnv_ToleratesCommentsQuotesAndExport(t *testing.T) {
	body := `# comment line

PLAIN=value
export EXPORTED=yes
QUOTED_DOUBLE="hello world"
QUOTED_SINGLE='single # not comment'
UNQUOTED_COMMENTED=abc # trailing comment
`
	pairs, err := ParseDotEnv([]byte(body))
	if err != nil {
		t.Fatalf("ParseDotEnv: %v", err)
	}
	got := map[string]string{}
	for _, kv := range pairs {
		got[kv.Key] = kv.Value
	}
	want := map[string]string{
		"PLAIN":              "value",
		"EXPORTED":           "yes",
		"QUOTED_DOUBLE":      "hello world",
		"QUOTED_SINGLE":      "single # not comment",
		"UNQUOTED_COMMENTED": "abc",
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("key %q = %q, want %q", k, got[k], v)
		}
	}
}

// Spec: KWL-2X1QZ CFG-DOTV-001 CFG-DOTV-002 Scope: Unit
func TestCFG_DOTV_001_LoadDotEnv_SetsMissingKeepsExistingAndToleratesAbsentFile(t *testing.T) {
	t.Setenv("EXISTING", "from-shell")
	t.Setenv("SET_BY_FILE", "")
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	body := "EXISTING=from-dotenv\nNEW=loaded\nSET_BY_FILE=file\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := LoadDotEnv(path); err != nil {
		t.Fatalf("LoadDotEnv: %v", err)
	}
	if got := os.Getenv("EXISTING"); got != "from-shell" {
		t.Errorf("EXISTING = %q, want shell value to win over .env", got)
	}
	if got := os.Getenv("NEW"); got != "loaded" {
		t.Errorf("NEW = %q, want %q", got, "loaded")
	}
	// SET_BY_FILE exists but is empty in the process env; LookupEnv reports
	// existence, so .env must not overwrite it.
	if got := os.Getenv("SET_BY_FILE"); got != "" {
		t.Errorf("SET_BY_FILE = %q, want empty (existing env never overwritten)", got)
	}

	if err := LoadDotEnv(filepath.Join(dir, "missing.env")); err != nil {
		t.Errorf("LoadDotEnv(missing) = %v, want nil for absent file", err)
	}
}

// Spec: KWL-2X1QZ CFG-DOTV-003 Scope: Unit
func TestCFG_DOTV_003_ParseDotEnv_MalformedLineErrorsWithLineNumber(t *testing.T) {
	_, err := ParseDotEnv([]byte("GOOD=1\nnot-a-pair\n"))
	if err == nil {
		t.Fatal("ParseDotEnv(malformed) = nil error, want error")
	}
	if !strings.Contains(err.Error(), "line 2") {
		t.Errorf("error %q should name the offending line number 2", err.Error())
	}
}

func TestParseDotEnv_MultilineValues(t *testing.T) {
	body := `
PRIVATE_KEY="-----BEGIN RSA PRIVATE KEY-----
MIIEowIBAAKCAQEA0
some-key-data
-----END RSA PRIVATE KEY-----"
CERT='-----BEGIN CERTIFICATE-----
MIIBkTCB...
-----END CERTIFICATE-----'
SINGLE_LINE="normal value"
`
	pairs, err := ParseDotEnv([]byte(body))
	if err != nil {
		t.Fatalf("ParseDotEnv failed: %v", err)
	}
	got := map[string]string{}
	for _, p := range pairs {
		got[p.Key] = p.Value
	}

	wantKey := "-----BEGIN RSA PRIVATE KEY-----\nMIIEowIBAAKCAQEA0\nsome-key-data\n-----END RSA PRIVATE KEY-----"
	if got["PRIVATE_KEY"] != wantKey {
		t.Errorf("PRIVATE_KEY =\n%q\nwant:\n%q", got["PRIVATE_KEY"], wantKey)
	}

	wantCert := "-----BEGIN CERTIFICATE-----\nMIIBkTCB...\n-----END CERTIFICATE-----"
	if got["CERT"] != wantCert {
		t.Errorf("CERT =\n%q\nwant:\n%q", got["CERT"], wantCert)
	}

	if got["SINGLE_LINE"] != "normal value" {
		t.Errorf("SINGLE_LINE = %q, want %q", got["SINGLE_LINE"], "normal value")
	}
}

func TestParseDotEnv_EscapeSequences(t *testing.T) {
	body := `
ESCAPED="line1\nline2\ttabbed \"quoted\" and \\slash"
RAW_SINGLE='line1\nline2'
`
	pairs, err := ParseDotEnv([]byte(body))
	if err != nil {
		t.Fatalf("ParseDotEnv failed: %v", err)
	}
	got := map[string]string{}
	for _, p := range pairs {
		got[p.Key] = p.Value
	}

	wantEscaped := "line1\nline2\ttabbed \"quoted\" and \\slash"
	if got["ESCAPED"] != wantEscaped {
		t.Errorf("ESCAPED = %q, want %q", got["ESCAPED"], wantEscaped)
	}

	// Single quotes do not unescape \n
	if got["RAW_SINGLE"] != `line1\nline2` {
		t.Errorf("RAW_SINGLE = %q, want %q", got["RAW_SINGLE"], `line1\nline2`)
	}
}

func TestParseDotEnv_VariableExpansion(t *testing.T) {
	t.Setenv("EXTERNAL_ENV", "production")
	body := `
BASE_HOST=api.example.com
PORT=8443
BASE_URL="https://${BASE_HOST}:${PORT}"
ENDPOINT=$BASE_URL/v1/users
ENV_MODE=${EXTERNAL_ENV}
ESCAPED_DOLLAR="\$LITERAL_DOLLAR"
UNQUOTED_EXPAND=prefix_${BASE_HOST}_suffix
RAW_SINGLE='${BASE_HOST} should not expand'
`
	pairs, err := ParseDotEnv([]byte(body))
	if err != nil {
		t.Fatalf("ParseDotEnv failed: %v", err)
	}
	got := map[string]string{}
	for _, p := range pairs {
		got[p.Key] = p.Value
	}

	if got["BASE_URL"] != "https://api.example.com:8443" {
		t.Errorf("BASE_URL = %q, want %q", got["BASE_URL"], "https://api.example.com:8443")
	}
	if got["ENDPOINT"] != "https://api.example.com:8443/v1/users" {
		t.Errorf("ENDPOINT = %q, want %q", got["ENDPOINT"], "https://api.example.com:8443/v1/users")
	}
	if got["ENV_MODE"] != "production" {
		t.Errorf("ENV_MODE = %q, want %q", got["ENV_MODE"], "production")
	}
	if got["ESCAPED_DOLLAR"] != "$LITERAL_DOLLAR" {
		t.Errorf("ESCAPED_DOLLAR = %q, want %q", got["ESCAPED_DOLLAR"], "$LITERAL_DOLLAR")
	}
	if got["UNQUOTED_EXPAND"] != "prefix_api.example.com_suffix" {
		t.Errorf("UNQUOTED_EXPAND = %q, want %q", got["UNQUOTED_EXPAND"], "prefix_api.example.com_suffix")
	}
	if got["RAW_SINGLE"] != "${BASE_HOST} should not expand" {
		t.Errorf("RAW_SINGLE = %q, want %q", got["RAW_SINGLE"], "${BASE_HOST} should not expand")
	}
}

func TestParseDotEnv_UnclosedQuote(t *testing.T) {
	body := `
KEY1=val1
UNCLOSED="start of multiline
still going
no closing quote
`
	_, err := ParseDotEnv([]byte(body))
	if err == nil {
		t.Fatal("expected error on unclosed quote, got nil")
	}
	if !strings.Contains(err.Error(), "unclosed quote") || !strings.Contains(err.Error(), "line 3") {
		t.Errorf("unexpected error: %v", err)
	}
}
