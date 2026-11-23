package apphost

import (
	"bytes"
	"fmt"
	"os"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Parse reads the `topology:` section of a krewire.yaml document, normalizes
// it and validates it. Other top-level keys are ignored, so the topology lives
// next to the rest of the project configuration.
func Parse(data []byte) (*Topology, error) {
	var doc struct {
		Topology *Topology `yaml:"topology"`
	}
	dec := yaml.NewDecoder(bytes.NewReader(data))
	if err := dec.Decode(&doc); err != nil {
		return nil, fmt.Errorf("apphost: parse krewire.yaml: %w", err)
	}
	if doc.Topology == nil {
		return nil, fmt.Errorf("apphost: krewire.yaml has no %q section", TopologyKey)
	}
	t := doc.Topology
	t.Normalize()
	if err := t.Validate(); err != nil {
		return nil, err
	}
	return t, nil
}

// LoadFile reads and parses a krewire.yaml file.
func LoadFile(path string) (*Topology, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("apphost: %w", err)
	}
	return Parse(data)
}

// Normalize fills names and defaults in place. It is idempotent and is called
// by Parse; callers building a Topology in Go should call it before Validate.
func (t *Topology) Normalize() {
	for name, r := range t.Resources {
		if r == nil {
			r = &Resource{}
			t.Resources[name] = r
		}
		r.Name = name
		if r.Restart.Policy == "" {
			r.Restart.Policy = RestartNever
		}
		if r.Restart.MaxRestarts == 0 {
			r.Restart.MaxRestarts = DefaultMaxRestarts
		}
		if r.Restart.Backoff == 0 {
			r.Restart.Backoff = Duration(DefaultRestartBackoff)
		}
		if r.ShutdownTimeout == 0 {
			r.ShutdownTimeout = Duration(DefaultShutdownTimeout)
		}
		for i := range r.Endpoints {
			e := &r.Endpoints[i]
			if e.Host == "" {
				e.Host = DefaultHost
			}
			if e.Scheme == "" {
				e.Scheme = "tcp"
			}
		}
		if r.Readiness == nil && r.External == nil && len(r.Endpoints) > 0 {
			r.Readiness = &Probe{TCP: r.Endpoints[0].Name}
		}
		if p := r.Readiness; p != nil {
			if p.Interval == 0 {
				p.Interval = Duration(DefaultProbeInterval)
			}
			if p.Timeout == 0 {
				p.Timeout = Duration(DefaultReadyTimeout)
			}
		}
	}
}

// Clone returns a deep copy, so a proposal can be edited without touching the
// running topology.
func (t *Topology) Clone() *Topology {
	data, err := yaml.Marshal(t)
	if err != nil {
		panic(err) // a Topology always marshals
	}
	var c Topology
	if err := yaml.Unmarshal(data, &c); err != nil {
		panic(err)
	}
	for name, r := range c.Resources {
		if r != nil {
			r.Name = name
		}
	}
	return &c
}

// MarshalYAML writes durations as strings.
func (d Duration) MarshalYAML() (any, error) { return d.String(), nil }

// Names returns resource names sorted lexically.
func (t *Topology) Names() []string {
	out := make([]string, 0, len(t.Resources))
	for n := range t.Resources {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

func sortedKeys(set map[string]bool) []string {
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// envName turns a resource or endpoint name into an environment fragment:
// "user-db" becomes "USER_DB".
func envName(s string) string {
	return strings.ToUpper(strings.NewReplacer("-", "_", ".", "_").Replace(s))
}
