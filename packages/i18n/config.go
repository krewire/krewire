package i18n

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config defines the configuration schema for the i18n subsystem.
// It can be loaded from YAML (e.g. krewire.yaml), JSON, environment variables,
// or a dynamic configuration source.
type Config struct {
	// DefaultLocale is the primary locale used when none is specified (default: "en").
	DefaultLocale string `yaml:"default_locale" json:"default_locale" env:"I18N_DEFAULT_LOCALE"`

	// FallbackLocale is the locale to fall back to when a key is missing in the target locale (default: "en").
	FallbackLocale string `yaml:"fallback_locale" json:"fallback_locale" env:"I18N_FALLBACK_LOCALE"`

	// BasePath is the filesystem path where translation files reside (default: "lang").
	BasePath string `yaml:"base_path" json:"base_path" env:"I18N_BASE_PATH"`

	// Locales is the optional explicit list of supported locales (e.g. ["en", "id"]).
	Locales []string `yaml:"locales" json:"locales" env:"I18N_LOCALES"`

	// QueryParam is the URL query parameter name used by middleware (default: "lang").
	QueryParam string `yaml:"query_param" json:"query_param" env:"I18N_QUERY_PARAM"`

	// CookieName is the HTTP cookie name used by middleware (default: "lang").
	CookieName string `yaml:"cookie_name" json:"cookie_name" env:"I18N_COOKIE_NAME"`

	// HeaderName is the HTTP request header used by middleware (default: "Accept-Language").
	HeaderName string `yaml:"header_name" json:"header_name" env:"I18N_HEADER_NAME"`
}

// DefaultConfig returns an initialized Config with production defaults.
func DefaultConfig() Config {
	return Config{
		DefaultLocale:  "en",
		FallbackLocale: "en",
		BasePath:       "lang",
		QueryParam:     "lang",
		CookieName:     "lang",
		HeaderName:     "Accept-Language",
	}
}

// WithConfig applies settings from a Config struct onto the Bundle.
func WithConfig(cfg Config) Option {
	return func(b *Bundle) {
		if cfg.DefaultLocale != "" {
			b.defaultLocale = strings.ToLower(cfg.DefaultLocale)
		}
		if cfg.FallbackLocale != "" {
			b.fallbackLocale = strings.ToLower(cfg.FallbackLocale)
		}
		if cfg.BasePath != "" {
			b.basePath = strings.Trim(cfg.BasePath, "/")
		}
		for _, loc := range cfg.Locales {
			if trimmed := strings.ToLower(strings.TrimSpace(loc)); trimmed != "" {
				b.locales[trimmed] = struct{}{}
			}
		}
	}
}

// NewFromConfig creates an i18n Bundle configured from a Config struct.
func NewFromConfig(fsys fs.FS, cfg Config, opts ...Option) (*Bundle, error) {
	allOpts := append([]Option{WithConfig(cfg)}, opts...)
	return New(fsys, allOpts...)
}

// CancelFunc cancels a watch subscription.
type CancelFunc func()

// ConfigSource is the minimal contract required to load and watch dynamic configuration.
type ConfigSource interface {
	Get(ctx context.Context, key string) (string, error)
	Watch(ctx context.Context, key string) (<-chan string, CancelFunc, error)
}

// LoadConfigFromSource reads an i18n Config from a dynamic ConfigSource.
// It supports values stored as JSON or YAML under the given key.
func LoadConfigFromSource(ctx context.Context, src ConfigSource, key string) (Config, error) {
	cfg := DefaultConfig()
	if src == nil {
		return cfg, fmt.Errorf("i18n: nil config source")
	}

	data, err := src.Get(ctx, key)
	if err != nil {
		return cfg, fmt.Errorf("i18n: read config key %q from source: %w", key, err)
	}

	if err := parseConfigData(data, &cfg); err != nil {
		return cfg, fmt.Errorf("i18n: parse config from source: %w", err)
	}

	return cfg, nil
}

// WatchSource watches a ConfigSource for dynamic updates to i18n configuration.
// When changes occur, it updates the bundle's locales and settings dynamically.
func WatchSource(ctx context.Context, src ConfigSource, key string, bundle *Bundle) (CancelFunc, error) {
	if src == nil || bundle == nil {
		return func() {}, fmt.Errorf("i18n: source and bundle must not be nil")
	}

	changes, cancel, err := src.Watch(ctx, key)
	if err != nil {
		return nil, err
	}

	go func() {
		for data := range changes {
			var newCfg Config
			if err := parseConfigData(data, &newCfg); err == nil {
				bundle.mu.Lock()
				if newCfg.DefaultLocale != "" {
					bundle.defaultLocale = strings.ToLower(newCfg.DefaultLocale)
				}
				if newCfg.FallbackLocale != "" {
					bundle.fallbackLocale = strings.ToLower(newCfg.FallbackLocale)
				}
				for _, loc := range newCfg.Locales {
					if trimmed := strings.ToLower(strings.TrimSpace(loc)); trimmed != "" {
						bundle.locales[trimmed] = struct{}{}
					}
				}
				bundle.mu.Unlock()
			}
		}
	}()

	return cancel, nil
}

func parseConfigData(data string, dst *Config) error {
	trimmed := strings.TrimSpace(data)
	if strings.HasPrefix(trimmed, "{") {
		return json.Unmarshal([]byte(data), dst)
	}
	return yaml.Unmarshal([]byte(data), dst)
}
