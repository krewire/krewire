package workgraph

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"path"
	"sort"
	"strings"
)

// CauseKind says why a node is affected.
type CauseKind string

const (
	// CauseChanged: a file owned by the node changed.
	CauseChanged CauseKind = "changed"
	// CauseGlobal: a global input (toolchain pin, shared schema) changed.
	CauseGlobal CauseKind = "global"
	// CauseDependency: a node this node depends on is affected.
	CauseDependency CauseKind = "dependency"
	// CauseFull: full-run mode bypassed affected calculation.
	CauseFull CauseKind = "full"
)

// Cause explains one affected decision. For dependency causes, Via is the
// affected dependency and Provenance the declaring site of the edge.
type Cause struct {
	Kind       CauseKind   `json:"kind"`
	File       string      `json:"file,omitempty"`
	Via        string      `json:"via,omitempty"`
	Edge       EdgeKind    `json:"edge,omitempty"`
	Provenance *Provenance `json:"provenance,omitempty"`
}

func (c Cause) String() string {
	switch c.Kind {
	case CauseChanged:
		return "changed file " + c.File
	case CauseGlobal:
		return "global input " + c.File
	case CauseDependency:
		s := fmt.Sprintf("depends on %s (%s)", c.Via, c.Edge)
		if c.Provenance != nil && c.Provenance.File != "" {
			s += fmt.Sprintf(" at %s:%d", c.Provenance.File, c.Provenance.Line)
		}
		return s
	}
	return "full run"
}

// AffectedNode is one affected node and its first-found cause.
type AffectedNode struct {
	ID    string `json:"id"`
	Cause Cause  `json:"cause"`
}

// Mode values.
const (
	ModeAffected = "affected"
	ModeFull     = "full"
)

// AffectedSet is the machine-readable affected calculation.
type AffectedSet struct {
	APIVersion   string         `json:"api_version"`
	Kind         string         `json:"kind"`
	Mode         string         `json:"mode"`
	Base         string         `json:"base,omitempty"`
	Head         string         `json:"head,omitempty"`
	GraphDigest  string         `json:"graph_digest"`
	ChangedFiles []string       `json:"changed_files,omitempty"`
	GlobalInputs []string       `json:"global_inputs_changed,omitempty"`
	Unowned      []string       `json:"unowned_files,omitempty"`
	Nodes        []AffectedNode `json:"nodes"`
	index        map[string]Cause
}

// AffectedOptions tunes Affected.
type AffectedOptions struct {
	Base, Head string
	// UnownedIsGlobal treats files no node owns as global inputs (the safest
	// policy for repositories whose tests read arbitrary files).
	UnownedIsGlobal bool
}

// Affected maps changed files (slash paths relative to the root) to the set
// of affected nodes: owners of changed files, every node when a global input
// changed, and the transitive dependents of both.
func (ws *Workspace) Affected(changed []string, opt AffectedOptions) *AffectedSet {
	a := ws.newSet(ModeAffected, opt.Base, opt.Head)
	a.ChangedFiles = sortedUnique(cleanAll(changed))
	var queue []string
	mark := func(id string, c Cause) {
		if _, ok := a.index[id]; !ok {
			a.index[id] = c
			queue = append(queue, id)
		}
	}
	global := ws.Manifest.globalInputs()
	for _, f := range a.ChangedFiles {
		owners := ws.Owners(f)
		switch {
		case matchAny(global, f) || (len(owners) == 0 && opt.UnownedIsGlobal):
			a.GlobalInputs = append(a.GlobalInputs, f)
		case len(owners) == 0:
			a.Unowned = append(a.Unowned, f)
		}
		for _, id := range owners {
			mark(id, Cause{Kind: CauseChanged, File: f})
		}
	}
	if len(a.GlobalInputs) > 0 {
		for _, n := range ws.Graph.Nodes() {
			mark(n.ID, Cause{Kind: CauseGlobal, File: a.GlobalInputs[0]})
		}
	}
	sort.Strings(queue)
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for _, dep := range ws.Graph.Dependents(cur, false) {
			e := ws.Graph.EdgesBetween(dep, cur)[0]
			c := Cause{Kind: CauseDependency, Via: cur, Edge: e.Kind}
			if len(e.Provenance) > 0 {
				p := e.Provenance[0]
				c.Provenance = &p
			}
			mark(dep, c)
		}
	}
	return a.finish()
}

// All returns the full-run set: every node, bypassing affected calculation.
func (ws *Workspace) All() *AffectedSet {
	a := ws.newSet(ModeFull, "", "")
	for _, n := range ws.Graph.Nodes() {
		a.index[n.ID] = Cause{Kind: CauseFull}
	}
	return a.finish()
}

func (ws *Workspace) newSet(mode, base, head string) *AffectedSet {
	return &AffectedSet{APIVersion: APIVersion, Kind: KindAffectedDocument, Mode: mode,
		Base: base, Head: head, GraphDigest: ws.Graph.Digest(), index: map[string]Cause{}}
}

func (a *AffectedSet) finish() *AffectedSet {
	a.Nodes = make([]AffectedNode, 0, len(a.index))
	for _, id := range sortedKeysOf(a.index) {
		a.Nodes = append(a.Nodes, AffectedNode{ID: id, Cause: a.index[id]})
	}
	return a
}

// Cause returns why id is affected and whether it is.
func (a *AffectedSet) Cause(id string) (Cause, bool) {
	c, ok := a.index[id]
	return c, ok
}

// WriteJSON writes the set as indented JSON.
func (a *AffectedSet) WriteJSON(w io.Writer) error { return writeJSON(w, a) }

// WriteText writes one line per affected node with its reason.
func (a *AffectedSet) WriteText(w io.Writer) error {
	var b strings.Builder
	fmt.Fprintf(&b, "%s: %d affected node(s) from %d changed file(s)\n", a.Mode, len(a.Nodes), len(a.ChangedFiles))
	for _, f := range a.Unowned {
		fmt.Fprintf(&b, "  unowned %s\n", f)
	}
	for _, n := range a.Nodes {
		fmt.Fprintf(&b, "  %s: %s\n", n.ID, n.Cause)
	}
	_, err := io.WriteString(w, b.String())
	return err
}

func cleanAll(in []string) []string {
	out := make([]string, 0, len(in))
	for _, f := range in {
		if f = strings.TrimSpace(f); f != "" {
			out = append(out, path.Clean(strings.ReplaceAll(f, "\\", "/")))
		}
	}
	return out
}

// ChangedFiles lists files changed between base and head in the Git
// repository at dir, relative to dir. An empty head compares base with the
// working tree and includes untracked files. Renames are split into a
// deletion and an addition so both sides are attributed.
func ChangedFiles(ctx context.Context, dir, base, head string) ([]string, error) {
	if base == "" {
		return nil, fmt.Errorf("workgraph: changed files need a base revision")
	}
	args := []string{"diff", "--name-only", "--no-renames", "--relative", "-z"}
	if head == "" {
		args = append(args, base)
	} else {
		args = append(args, base+"..."+head)
	}
	out, err := git(ctx, dir, args...)
	if err != nil {
		return nil, err
	}
	if head == "" {
		untracked, err := git(ctx, dir, "ls-files", "--others", "--exclude-standard", "-z")
		if err != nil {
			return nil, err
		}
		out = append(out, untracked...)
	}
	var files []string
	for _, f := range bytes.Split(out, []byte{0}) {
		if len(f) > 0 {
			files = append(files, string(f))
		}
	}
	return sortedUnique(files), nil
}

func git(ctx context.Context, dir string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", dir}, args...)...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("workgraph: git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return out, nil
}
