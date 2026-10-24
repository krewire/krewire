package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	dirPerm  = 0o755
	filePerm = 0o644
)

// Vars is a flat key-value configuration store with dot-notation keys,
// loadable from and saveable to YAML files (KWL-2X1QZ CFG-KV-001).
type Vars map[string]string

// LoadVars reads the YAML file at path and flattens it into Vars using
// dot-notation keys. A missing file or empty path yields an empty Vars and
// nil error; a malformed file yields a wrapped error naming the path
// (KWL-2X1QZ CFG-KV-002).
func LoadVars(path string) (Vars, error) {
	v := make(Vars)
	if path == "" {
		return v, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return v, nil
		}
		return nil, fmt.Errorf("config: read %q: %w", path, err)
	}
	var raw map[string]any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("config: parse %q: %w", path, err)
	}
	flatten(raw, "", v)
	return v, nil
}

// Save writes the Vars map to a YAML file as a nested structure, creating
// parent directories as needed (KWL-2X1QZ CFG-KV-003).
func (v Vars) Save(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, dirPerm); err != nil {
		return fmt.Errorf("config: mkdir %q: %w", dir, err)
	}
	nested := unflatten(v)
	data, err := yaml.Marshal(nested)
	if err != nil {
		return fmt.Errorf("config: marshal: %w", err)
	}
	return os.WriteFile(path, data, filePerm)
}

// Get returns the value for key, or empty string if not found.
func (v Vars) Get(key string) string {
	return v[key]
}

// GetOr returns the value for key when present and non-empty, otherwise
// fallback (KWL-2X1QZ CFG-KV-005).
func (v Vars) GetOr(key, fallback string) string {
	if val, ok := v[key]; ok && val != "" {
		return val
	}
	return fallback
}

// Set sets a key-value pair.
func (v Vars) Set(key, value string) {
	v[key] = value
}

// Delete removes a key.
func (v Vars) Delete(key string) {
	delete(v, key)
}

// Keys returns all keys in sorted order.
func (v Vars) Keys() []string {
	keys := make([]string, 0, len(v))
	for k := range v {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// Merge merges other into a copy of v, with other taking precedence
// (KWL-2X1QZ CFG-KV-004).
func (v Vars) Merge(other Vars) Vars {
	result := make(Vars)
	for k, val := range v {
		result[k] = val
	}
	for k, val := range other {
		result[k] = val
	}
	return result
}

// EnvOverride applies environment variables with the given prefix on top of
// v. Keys are lower-cased with underscores replaced by dots, e.g.
// APP_SERVER_PORT=8080 -> server.port (KWL-2X1QZ CFG-KV-004).
func (v Vars) EnvOverride(prefix string) Vars {
	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			continue
		}
		k, value := parts[0], parts[1]
		if !strings.HasPrefix(k, prefix+"_") {
			continue
		}
		key := strings.ToLower(strings.TrimPrefix(k, prefix+"_"))
		key = strings.ReplaceAll(key, "_", ".")
		v[key] = value
	}
	return v
}

// WithDefaults returns a copy of v filled with defaults for absent keys
// (KWL-2X1QZ CFG-KV-004).
func (v Vars) WithDefaults(defaults Vars) Vars {
	result := make(Vars)
	for k, val := range v {
		result[k] = val
	}
	for k, val := range defaults {
		if _, exists := result[k]; !exists {
			result[k] = val
		}
	}
	return result
}

// flatten converts a nested map to a flat map with dot-notation keys.
func flatten(m map[string]any, prefix string, out Vars) {
	for k, val := range m {
		key := k
		if prefix != "" {
			key = prefix + "." + k
		}
		switch val := val.(type) {
		case map[string]any:
			flatten(val, key, out)
		case []any:
			// For slices, we store as JSON-like string or indexed keys
			for i, item := range val {
				if m, ok := item.(map[string]any); ok {
					flatten(m, fmt.Sprintf("%s.%d", key, i), out)
				} else {
					out[fmt.Sprintf("%s.%d", key, i)] = fmt.Sprintf("%v", item)
				}
			}
		default:
			out[key] = fmt.Sprintf("%v", val)
		}
	}
}

// unflatten converts a flat map with dot-notation keys back to a nested map.
func unflatten(flat Vars) map[string]any {
	result := make(map[string]any)
	for k, val := range flat {
		parts := strings.Split(k, ".")
		current := result
		for i, part := range parts {
			if i == len(parts)-1 {
				current[part] = val
			} else {
				if next, ok := current[part].(map[string]any); ok {
					current = next
				} else {
					next = make(map[string]any)
					current[part] = next
					current = next
				}
			}
		}
	}
	return result
}
