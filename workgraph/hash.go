package workgraph

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os/exec"
	"path"
	"strings"
	"sync"
)

// KeySchema versions the cache-key derivation. Bump it whenever KeyInputs
// or its hashing changes so old entries can never be restored.
const KeySchema = "workgraph-key/v1"

// KeyInputs is everything a target's cache key covers. It is emitted in
// reports so humans and agents can see why a key changed; it cannot be
// edited to force a restore because the key is recomputed from disk.
type KeyInputs struct {
	Schema       string   `json:"schema"`
	Target       string   `json:"target"`
	Argv         []string `json:"argv"`
	WorkDir      string   `json:"work_dir"`
	FailOnOutput bool     `json:"fail_on_output,omitempty"`
	GoVersion    string   `json:"go_version"`
	// Env maps each declared variable to its value ("" when unset).
	Env map[string]string `json:"env"`
	// Lock maps go.mod/go.sum files in the dependency closure to digests.
	Lock map[string]string `json:"lock"`
	// Externals maps third-party modules in the closure to versions.
	Externals map[string]string `json:"externals,omitempty"`
	// Files maps every source/input file to its content digest.
	Files map[string]string `json:"files"`
}

// Key returns the content address: SHA-256 over the canonical JSON (Go maps
// marshal with sorted keys, so the encoding is deterministic).
func (k KeyInputs) Key() string {
	data, _ := json.Marshal(k)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// Hasher derives KeyInputs for targets, memoizing file digests.
type Hasher struct {
	ws        *Workspace
	goVersion string
	getenv    func(string) string

	mu    sync.Mutex
	files map[string]string
}

// NewHasher returns a hasher. goVersion is the active toolchain (see
// GoVersion); getenv supplies environment inputs (os.Getenv in production).
func NewHasher(ws *Workspace, goVersion string, getenv func(string) string) *Hasher {
	return &Hasher{ws: ws, goVersion: goVersion, getenv: getenv, files: map[string]string{}}
}

// GoVersion returns `go env GOVERSION`, the toolchain that will actually run
// (after GOTOOLCHAIN switching).
func GoVersion(ctx context.Context, dir string) (string, error) {
	cmd := exec.CommandContext(ctx, "go", "env", "GOVERSION")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("workgraph: go env GOVERSION: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// Inputs computes the key inputs of t: its node and every transitive
// dependency (all edge kinds), global inputs, and task-declared inputs.
func (h *Hasher) Inputs(t *Target) (KeyInputs, error) {
	spec := t.spec
	k := KeyInputs{
		Schema: KeySchema, Target: t.ID, Argv: t.Argv, WorkDir: t.WorkDir,
		FailOnOutput: spec.FailOnOutput, GoVersion: h.goVersion,
		Env: map[string]string{}, Lock: map[string]string{}, Externals: map[string]string{}, Files: map[string]string{},
	}
	for _, name := range append(append([]string(nil), DefaultEnv...), spec.Env...) {
		k.Env[name] = h.getenv(name)
	}
	g := h.ws.Graph
	var files []string
	for _, id := range append([]string{t.Node}, g.Dependencies(t.Node, true)...) {
		n, _ := g.Node(id)
		if n.Kind == KindExternal {
			k.Externals[n.Name] = n.Version
			continue
		}
		files = append(files, n.Files...)
	}
	global := h.ws.Manifest.globalInputs()
	for _, f := range h.ws.Files {
		if matchAny(global, f) || matchAny(spec.Inputs, f) {
			files = append(files, f)
		}
	}
	for _, f := range sortedUnique(files) {
		sum, err := h.digest(f)
		if err != nil {
			return KeyInputs{}, err
		}
		if base := path.Base(f); base == "go.mod" || base == "go.sum" {
			k.Lock[f] = sum
		} else {
			k.Files[f] = sum
		}
	}
	return k, nil
}

func (h *Hasher) digest(rel string) (string, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if s, ok := h.files[rel]; ok {
		return s, nil
	}
	data, err := readRel(h.ws.Root, rel)
	if err != nil {
		return "", fmt.Errorf("workgraph: hash %s: %w", rel, err)
	}
	sum := sha256.Sum256(data)
	s := hex.EncodeToString(sum[:])
	h.files[rel] = s
	return s, nil
}
