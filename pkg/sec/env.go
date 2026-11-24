package sec

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
)

var (
	// ErrReservedEnvOverride is returned when an attempt is made to override a reserved security environment variable.
	ErrReservedEnvOverride = errors.New("sec: cannot override reserved security environment variable")

	// ErrSensitiveEnvOverride is returned when an environment variable key or value contains credentials or secret patterns.
	ErrSensitiveEnvOverride = errors.New("sec: environment variable contains sensitive credentials")
)

// DefaultAllowlistEnvKeys contains standard non-secret environment variable names safe for child processes.
var DefaultAllowlistEnvKeys = []string{
	"PATH",
	"LANG",
	"LC_ALL",
	"LC_CTYPE",
	"TMPDIR",
	"TEMP",
	"TMP",
	"USER",
	"LOGNAME",
	"HOME",
	"SHELL",
	"TERM",
}

// ReservedSecurityEnvKeys contains environment variables that must never be overridden or injected.
var ReservedSecurityEnvKeys = []string{
	"LD_PRELOAD",
	"LD_LIBRARY_PATH",
	"LD_AUDIT",
	"LD_DEBUG",
	"DYLD_INSERT_LIBRARIES",
	"DYLD_LIBRARY_PATH",
	"DYLD_FRAMEWORK_PATH",
	"BASH_ENV",
	"ENV",
	"PS4",
	"PROMPT_COMMAND",
	"IFS",
	"NODE_OPTIONS",
	"PYTHONPATH",
	"RUBYOPT",
	"PERL5OPT",
}

// SensitivePatterns contains keywords indicating sensitive credentials in variable names or contents.
var SensitivePatterns = []string{
	"SECRET",
	"TOKEN",
	"KEY",
	"PASS",
	"PASSWORD",
	"PASSWD",
	"CREDENTIAL",
	"AUTH",
	"PRIVATE",
	"SIGNING",
	"DATABASE_URL",
	"DB_PASS",
	"SSH_AUTH_SOCK",
	"AWS_",
	"GITHUB_",
	"GH_",
	"OPENAI_",
	"ANTHROPIC_",
	"GEMINI_",
	"JWT_",
	"COOKIE",
	"BEARER",
	"ACCESS_KEY",
	"SESSION",
}

// IsReservedSecurityKey checks if the environment variable name is a reserved security variable.
func IsReservedSecurityKey(key string) bool {
	upper := strings.ToUpper(strings.TrimSpace(key))
	for _, reserved := range ReservedSecurityEnvKeys {
		if upper == reserved {
			return true
		}
	}
	return false
}

// IsSensitiveKey checks if the environment variable name matches sensitive credential patterns.
func IsSensitiveKey(key string) bool {
	upper := strings.ToUpper(strings.TrimSpace(key))
	for _, pattern := range SensitivePatterns {
		if strings.Contains(upper, pattern) {
			return true
		}
	}
	return false
}

var (
	sensitiveTokenRegex = regexp.MustCompile(`(?i)(?:sk-[a-zA-Z0-9_\-]{20,}|gh[pousr]_[a-zA-Z0-9_]{30,}|AKIA[0-9A-Z]{16}|bearer\s+[a-zA-Z0-9_\-\.]+|-----BEGIN [A-Z ]+PRIVATE KEY-----)`)
	uriCredentialRegex  = regexp.MustCompile(`(?i)(?:postgres|mysql|redis|mongodb|amqp|http|https)://[^:\s]+:[^@\s]+@`)
)

// IsSensitiveValue checks whether an environment variable's value contains secret patterns.
func IsSensitiveValue(val string) bool {
	if sensitiveTokenRegex.MatchString(val) {
		return true
	}
	if uriCredentialRegex.MatchString(val) {
		return true
	}
	return false
}

// ScrubSensitiveValue redacts sensitive information from KEY=VALUE formatted strings.
func ScrubSensitiveValue(v string) string {
	parts := strings.SplitN(v, "=", 2)
	if len(parts) == 2 {
		if IsSensitiveKey(parts[0]) || IsSensitiveValue(parts[1]) {
			return parts[0] + "=[REDACTED]"
		}
	}
	return RedactSensitiveOutput(v)
}

// EnvConfig specifies options for child-process environment sanitization.
type EnvConfig struct {
	ParentEnv    []string          // Source environment (defaults to os.Environ() if nil)
	Allowlist    []string          // Explicit variable keys to inherit from parent (defaults to DefaultAllowlistEnvKeys if empty)
	Overrides    map[string]string // Explicit non-secret variable overrides/additions
	Workspace    string            // Workspace directory (binds PWD/HOME safely)
	FallbackPath string            // Fallback PATH if none resolved
}

// SanitizeChildEnv constructs a minimal, isolated child-process environment vector
// adhering strictly to explicit allowlisting and secret redaction.
func SanitizeChildEnv(cfg EnvConfig) ([]string, error) {
	parentEnv := cfg.ParentEnv
	if parentEnv == nil {
		parentEnv = os.Environ()
	}

	allowlist := cfg.Allowlist
	if len(allowlist) == 0 {
		allowlist = DefaultAllowlistEnvKeys
	}

	allowMap := make(map[string]bool, len(allowlist))
	for _, k := range allowlist {
		allowMap[strings.ToUpper(strings.TrimSpace(k))] = true
	}

	envMap := make(map[string]string)

	// Filter parent environment against explicit allowlist and sensitive checks
	for _, entry := range parentEnv {
		parts := strings.SplitN(entry, "=", 2)
		if len(parts) != 2 {
			continue
		}
		k := parts[0]
		v := parts[1]
		upperKey := strings.ToUpper(strings.TrimSpace(k))

		// Must be explicitly allowlisted
		if !allowMap[upperKey] {
			continue
		}

		// Never inherit reserved security keys
		if IsReservedSecurityKey(k) {
			continue
		}

		// Never inherit sensitive keys or values
		if IsSensitiveKey(k) || IsSensitiveValue(v) {
			continue
		}

		envMap[k] = v
	}

	// Process explicit overrides
	for k, v := range cfg.Overrides {
		if IsReservedSecurityKey(k) {
			return nil, fmt.Errorf("%w: %s", ErrReservedEnvOverride, k)
		}
		if IsSensitiveKey(k) {
			return nil, fmt.Errorf("%w: key %q matches sensitive pattern", ErrSensitiveEnvOverride, k)
		}
		if IsSensitiveValue(v) {
			return nil, fmt.Errorf("%w: value for %q contains sensitive credentials", ErrSensitiveEnvOverride, k)
		}
		envMap[k] = v
	}

	// Ensure baseline variables are set
	if _, ok := envMap["PATH"]; !ok || strings.TrimSpace(envMap["PATH"]) == "" {
		fallback := cfg.FallbackPath
		if fallback == "" {
			fallback = "/usr/local/go/bin:/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin"
		}
		envMap["PATH"] = fallback
	}
	if _, ok := envMap["LANG"]; !ok || strings.TrimSpace(envMap["LANG"]) == "" {
		envMap["LANG"] = "C.UTF-8"
	}
	if _, ok := envMap["TMPDIR"]; !ok || strings.TrimSpace(envMap["TMPDIR"]) == "" {
		envMap["TMPDIR"] = os.TempDir()
	}
	if cfg.Workspace != "" {
		if _, ok := envMap["PWD"]; !ok {
			envMap["PWD"] = cfg.Workspace
		}
	}

	// Sort keys for deterministic output
	keys := make([]string, 0, len(envMap))
	for k := range envMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	out := make([]string, 0, len(keys))
	for _, k := range keys {
		out = append(out, fmt.Sprintf("%s=%s", k, envMap[k]))
	}
	return out, nil
}
