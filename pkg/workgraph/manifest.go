package workgraph

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// ManifestFile is the optional workspace manifest at the workspace root.
const ManifestFile = "krewire.workspace.yaml"

// DefaultGlobalInputs are files whose change invalidates every target: the
// workspace/toolchain pins and the manifest itself. go.mod and go.sum are
// handled per module through member edges, not globally.
var DefaultGlobalInputs = []string{
	"go.work", "go.work.sum", ".go-version", ".tool-versions", ManifestFile,
}

// Manifest declares what plain Go discovery cannot infer. Every field is
// optional; a workspace with only go.mod gets a correct package graph.
type Manifest struct {
	Version int `yaml:"version" json:"version"`
	// Modules lists extra module roots relative to the workspace root (for
	// example sibling repositories "../krew"), in addition to go.work uses.
	Modules []string `yaml:"modules,omitempty" json:"modules,omitempty"`
	// Exclude lists directory globs that discovery skips.
	Exclude []string `yaml:"exclude,omitempty" json:"exclude,omitempty"`
	// GlobalInputs are file globs whose change affects every target (shared
	// configuration schemas, CI workflow definitions, toolchain pins). They
	// extend DefaultGlobalInputs.
	GlobalInputs []string `yaml:"global_inputs,omitempty" json:"global_inputs,omitempty"`
	// Inputs adds file globs to the node owning a directory, e.g. a root
	// guard test that reads every pkg/**/*.go file: {".": ["pkg/**/*.go"]}.
	Inputs map[string][]string `yaml:"inputs,omitempty" json:"inputs,omitempty"`
	// Templates lists template directory globs (each match is one node).
	Templates []string `yaml:"templates,omitempty" json:"templates,omitempty"`
	// Workloads binds workload fixtures to templates and packages.
	Workloads []WorkloadSpec `yaml:"workloads,omitempty" json:"workloads,omitempty"`
	// Tasks overrides DefaultTasks when non-empty.
	Tasks []TaskSpec `yaml:"tasks,omitempty" json:"tasks,omitempty"`
}

// WorkloadSpec declares one workload node.
type WorkloadSpec struct {
	Name string `yaml:"name" json:"name"`
	// Template is the template directory the workload is generated from.
	Template string `yaml:"template,omitempty" json:"template,omitempty"`
	// Packages are package directories (relative to the root) or import
	// paths the workload relies on.
	Packages []string `yaml:"packages,omitempty" json:"packages,omitempty"`
	// Fixtures are directories whose files belong to the workload.
	Fixtures []string `yaml:"fixtures,omitempty" json:"fixtures,omitempty"`
}

// ParseManifest decodes a manifest, rejecting unknown fields so a typo
// cannot silently drop an invalidation rule.
func ParseManifest(data []byte) (*Manifest, error) {
	m := &Manifest{}
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(m); err != nil && !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("workgraph: %s: %w", ManifestFile, err)
	}
	if m.Version > 1 {
		return nil, fmt.Errorf("workgraph: %s: unsupported version %d", ManifestFile, m.Version)
	}
	for i, w := range m.Workloads {
		if w.Name == "" {
			return nil, fmt.Errorf("workgraph: %s: workload #%d has no name", ManifestFile, i+1)
		}
	}
	return m, nil
}

// LoadManifest reads root/krewire.workspace.yaml, returning an empty
// manifest when the file does not exist.
func LoadManifest(root string) (*Manifest, error) {
	data, err := os.ReadFile(filepath.Join(root, ManifestFile))
	if errors.Is(err, fs.ErrNotExist) {
		return &Manifest{Version: 1}, nil
	}
	if err != nil {
		return nil, err
	}
	return ParseManifest(data)
}

func (m *Manifest) globalInputs() []string {
	return append(append([]string(nil), DefaultGlobalInputs...), m.GlobalInputs...)
}
