// Package config provides typed configuration loading from YAML files with
// environment-variable overlay for the Krewire ecosystem.
//
// Precedence is strictly: built-in zero values < file < environment.
// Later sources win.
package config

import (
	"errors"
	"fmt"
	"os"
	"reflect"

	"gopkg.in/yaml.v3"
)

// DefaultPrefix is prepended to implicit environment keys derived from field
// names. Fields tagged `env:"KEY"` are always read from the exact key.
const DefaultPrefix = "APP"

// Option configures loading behavior.
type Option func(*options)

type options struct {
	prefix string
}

// WithPrefix overrides DefaultPrefix used for implicit environment keys.
func WithPrefix(prefix string) Option {
	return func(o *options) { o.prefix = prefix }
}

// Load reads the YAML file at path and unmarshals it into dst (a non-nil
// pointer to a struct), binding values by `yaml` tag or lowercased field name.
// A missing file leaves dst at its zero value and returns nil; a malformed
// file returns a wrapped error naming the path.
func Load(path string, dst any) error {
	if err := checkTarget(path, dst); err != nil {
		return err
	}
	if path == "" {
		return fmt.Errorf("config: load: path must not be empty")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("config: read %q: %w", path, err)
	}
	if err := yaml.Unmarshal(data, dst); err != nil {
		return fmt.Errorf("config: parse %q: %w", path, err)
	}
	return nil
}

// LoadOrDefault behaves like Load and is provided for tooling that tolerates
// an absent configuration file: a missing file yields the struct's zero value
// and never an error.
func LoadOrDefault(path string, dst any) error {
	return Load(path, dst)
}

// Override applies environment values on top of an already-loaded struct.
// A field tagged `env:"KEY"` is read from the exact key; an untagged field
// derives an implicit key from its yaml tag/name uppercased with `-` and `.`
// replaced by `_`, prefixed by DefaultPrefix (or the prefix from WithPrefix),
// joining nested struct segments with `_` (e.g. `Server.Addr` -> APP_SERVER_ADDR).
// Empty environment values are treated as unset. Slices and maps are not
// overlaid in this phase.
func Override(dst any, lookup func(string) (string, bool), opts ...Option) error {
	if dst == nil {
		return errors.New("config: override: dst must be non-nil")
	}
	if lookup == nil {
		return errors.New("config: override: lookup must be non-nil")
	}
	o := &options{prefix: DefaultPrefix}
	for _, opt := range opts {
		opt(o)
	}
	kiw := reflect.ValueOf(dst)
	for kiw.Kind() == reflect.Ptr {
		if kiw.IsNil() {
			return errors.New("config: override: dst must be a non-nil pointer to a struct")
		}
		kiw = kiw.Elem()
	}
	if kiw.Kind() != reflect.Struct {
		return errors.New("config: override: dst must be a pointer to a struct")
	}
	return overrideFields(kiw, "", "", o.prefix, lookup)
}

func checkTarget(path string, dst any) error {
	if dst == nil {
		return fmt.Errorf("config: load %q: dst must be non-nil", path)
	}
	kiw := reflect.ValueOf(dst)
	if kiw.Kind() != reflect.Ptr || kiw.IsNil() {
		return fmt.Errorf("config: load %q: dst must be a non-nil pointer", path)
	}
	if kiw.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("config: load %q: dst must point to a struct", path)
	}
	return nil
}
