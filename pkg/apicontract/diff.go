package apicontract

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/krewire/krewire/pkg/version"
)

// Severity classifies a contract change.
type Severity string

const (
	// Breaking changes can fail an existing, correct consumer.
	Breaking Severity = "breaking"
	// NonBreaking changes are additive or relax a requirement.
	NonBreaking Severity = "non-breaking"
)

// Bump is a semantic-version increment.
type Bump string

// Bumps, ordered from weakest to strongest.
const (
	BumpNone  Bump = "none"
	BumpPatch Bump = "patch"
	BumpMinor Bump = "minor"
	BumpMajor Bump = "major"
)

var bumpRank = map[Bump]int{BumpNone: 0, BumpPatch: 1, BumpMinor: 2, BumpMajor: 3}

// Change is one classified difference between two contract versions.
type Change struct {
	Severity Severity `json:"severity"`
	// Kind is a stable machine-readable identifier, e.g. "endpoint.removed".
	Kind     string `json:"kind"`
	Endpoint string `json:"endpoint,omitempty"`
	// Location points into the endpoint, e.g. "body.title" or "query.limit".
	Location string `json:"location,omitempty"`
	Message  string `json:"message"`
}

// Report is the result of comparing two documents. It is designed to be
// printed as JSON by CI and consumed as structured context by planning and
// review workflows.
type Report struct {
	Service     string   `json:"service"`
	OldVersion  string   `json:"oldVersion"`
	NewVersion  string   `json:"newVersion"`
	Breaking    int      `json:"breaking"`
	NonBreaking int      `json:"nonBreaking"`
	Changes     []Change `json:"changes"`
	// RequiredBump is the minimum version increment the changes demand.
	RequiredBump Bump `json:"requiredBump"`
	// ActualBump is the increment between OldVersion and NewVersion.
	ActualBump Bump `json:"actualBump"`
	// VersionOK reports whether ActualBump satisfies RequiredBump.
	VersionOK bool `json:"versionOk"`
	// Decision explains VersionOK in one sentence.
	Decision string `json:"decision"`
}

// JSON renders the report for CI logs and agent context.
func (r *Report) JSON() ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	err := enc.Encode(r)
	return buf.Bytes(), err
}

// Err returns nil when the version decision is sound, and an error naming
// the required bump otherwise.
func (r *Report) Err() error {
	if r.VersionOK {
		return nil
	}
	return fmt.Errorf("apicontract: %s: %s", r.Service, r.Decision)
}

// Diff compares an old (baseline) and a new contract document.
func Diff(oldDoc, newDoc *Document) *Report {
	c := &differ{old: oldDoc, new: newDoc, seen: map[[2]*Schema]bool{}}
	if oldDoc.Service != newDoc.Service {
		c.add(Breaking, "service.renamed", "", "", "service renamed from %q to %q", oldDoc.Service, newDoc.Service)
	}
	if oldDoc.BasePath != newDoc.BasePath {
		c.add(Breaking, "service.base_path_changed", "", "", "base path changed from %q to %q", oldDoc.BasePath, newDoc.BasePath)
	}
	for _, oe := range oldDoc.Endpoints {
		ne := newDoc.Endpoint(oe.Name)
		if ne == nil {
			c.add(Breaking, "endpoint.removed", oe.Name, "", "endpoint %s %s removed", oe.Method, oe.Path)
			continue
		}
		c.endpoint(oe, *ne)
	}
	for _, ne := range newDoc.Endpoints {
		if oldDoc.Endpoint(ne.Name) == nil {
			c.add(NonBreaking, "endpoint.added", ne.Name, "", "endpoint %s %s added", ne.Method, ne.Path)
		}
	}
	sort.SliceStable(c.changes, func(i, j int) bool {
		a, b := c.changes[i], c.changes[j]
		if a.Endpoint != b.Endpoint {
			return a.Endpoint < b.Endpoint
		}
		if a.Location != b.Location {
			return a.Location < b.Location
		}
		return a.Kind < b.Kind
	})
	r := &Report{
		Service: newDoc.Service, OldVersion: oldDoc.Version, NewVersion: newDoc.Version,
		Changes: c.changes,
	}
	if r.Changes == nil {
		r.Changes = []Change{}
	}
	for _, ch := range r.Changes {
		if ch.Severity == Breaking {
			r.Breaking++
		} else {
			r.NonBreaking++
		}
	}
	r.decide()
	return r
}

func (r *Report) decide() {
	ov, err1 := version.ParseVersion(r.OldVersion)
	nv, err2 := version.ParseVersion(r.NewVersion)
	if err1 != nil || err2 != nil {
		r.RequiredBump, r.ActualBump = BumpNone, BumpNone
		r.Decision = "versions are not valid semantic versions"
		return
	}
	switch {
	case r.Breaking > 0 && ov.Major == 0:
		// 0.x: SemVer allows breaking changes in a minor bump.
		r.RequiredBump = BumpMinor
	case r.Breaking > 0:
		r.RequiredBump = BumpMajor
	case r.NonBreaking > 0:
		r.RequiredBump = BumpMinor
	default:
		r.RequiredBump = BumpNone
	}
	switch {
	case nv.Major != ov.Major:
		r.ActualBump = BumpMajor
	case nv.Minor != ov.Minor:
		r.ActualBump = BumpMinor
	case nv.Patch != ov.Patch || nv.PreRelease != ov.PreRelease:
		r.ActualBump = BumpPatch
	default:
		r.ActualBump = BumpNone
	}
	switch {
	case nv.Less(ov):
		r.Decision = fmt.Sprintf("version went backwards from %s to %s", r.OldVersion, r.NewVersion)
	case bumpRank[r.ActualBump] < bumpRank[r.RequiredBump]:
		r.Decision = fmt.Sprintf("%d breaking and %d non-breaking changes require a %s bump, but %s -> %s is %s",
			r.Breaking, r.NonBreaking, r.RequiredBump, r.OldVersion, r.NewVersion, r.ActualBump)
	default:
		r.VersionOK = true
		r.Decision = fmt.Sprintf("%s bump satisfies required %s bump", r.ActualBump, r.RequiredBump)
	}
}

type differ struct {
	old, new *Document
	changes  []Change
	seen     map[[2]*Schema]bool
	ep       string
}

func (c *differ) add(sev Severity, kind, ep, loc, format string, args ...any) {
	c.changes = append(c.changes, Change{Severity: sev, Kind: kind, Endpoint: ep, Location: loc, Message: fmt.Sprintf(format, args...)})
}

// pick returns breaking when cond holds, non-breaking otherwise.
func pick(cond bool) Severity {
	if cond {
		return Breaking
	}
	return NonBreaking
}

// DiffFiles loads two canonical documents (baseline first) and diffs them.
// It is the entry point for CLI wiring such as `krew contract diff`.
func DiffFiles(oldPath, newPath string) (*Report, error) {
	oldDoc, err := LoadDocument(oldPath)
	if err != nil {
		return nil, fmt.Errorf("apicontract: baseline %s: %w", oldPath, err)
	}
	newDoc, err := LoadDocument(newPath)
	if err != nil {
		return nil, fmt.Errorf("apicontract: candidate %s: %w", newPath, err)
	}
	return Diff(oldDoc, newDoc), nil
}
