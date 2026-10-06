// Package core — workload registry (KWF-M8K2Q, KWL-K1N2Q).
package model

import (
	"fmt"
	"github.com/krewire/krewire/packages/kern/errs"
	"strings"
)

// Kind is a Krewire project kind. Eight kinds cover the unified workload spectrum.
type Kind string

const (
	KindApp     Kind = "app"
	KindCLI     Kind = "cli"
	KindSite    Kind = "site"
	KindBook    Kind = "book"
	KindWorker  Kind = "worker"
	KindService Kind = "service"
	KindInfra   Kind = "infra"
	KindKernel  Kind = "kernel"
)

// AllKinds lists every valid Kind in canonical order.
var AllKinds = []Kind{KindApp, KindCLI, KindSite, KindBook, KindWorker, KindService, KindInfra, KindKernel}

// IsValid reports whether k is one of the eight known kinds. It is derived from
// AllKinds so the kind list has a single owner.
func (k Kind) IsValid() bool {
	for _, kind := range AllKinds {
		if k == kind {
			return true
		}
	}
	return false
}

// ParseKind parses s as a Kind, returning UsageError on unknown.
func ParseKind(s string) (Kind, error) {
	k := Kind(strings.TrimSpace(s))
	if !k.IsValid() {
		return "", errs.UsageError(fmt.Sprintf("unknown project kind %q: want one of %s", s, strings.Join(kindsAsStrings(), ", ")))
	}
	return k, nil
}

func kindsAsStrings() []string {
	out := make([]string, len(AllKinds))
	for i, k := range AllKinds {
		out[i] = string(k)
	}
	return out
}

// Status is the implementation status of a workload.
type Status string

const (
	StatusShipped Status = "shipped"
	StatusPlanned Status = "planned"
)

// Workload is one cell of the workload matrix: a Kind and the package that
// implements it.
//
// P is the caller's own package type, so the kernel never names a concrete
// module. Instantiate it with the module path when that is all you need:
//
//	var m = model.NewMatrix(
//		model.Workload[string]{Kind: model.KindSite, Package: "web/ssg", ...},
//	)
type Workload[P any] struct {
	Kind    Kind   `json:"kind"`
	Package P      `json:"package"`
	Title   string `json:"title"`
	SpecID  string `json:"specId"` // e.g. KWF-5XJFC
	Status  Status `json:"status"`
}

// Matrix is the ordered kind → workload registry for one package type.
//
// The kernel ships no roster: the ecosystem declares its own matrix above this
// layer, so adding a workload never edits the kernel. The first entry wins for
// a given Kind, matching the historical "first match" lookup, and iteration
// follows insertion order so reports built from it are deterministic.
type Matrix[P any] struct {
	cells  []Workload[P]
	byKind map[Kind]int
}

// NewMatrix returns a Matrix seeded with cells.
func NewMatrix[P any](cells ...Workload[P]) *Matrix[P] {
	m := &Matrix[P]{byKind: make(map[Kind]int, len(cells))}
	for _, c := range cells {
		m.Add(c)
	}
	return m
}

// Add appends a cell unless its Kind is already registered, in which case the
// existing entry is kept.
func (m *Matrix[P]) Add(w Workload[P]) {
	if _, seen := m.byKind[w.Kind]; seen {
		return
	}
	m.byKind[w.Kind] = len(m.cells)
	m.cells = append(m.cells, w)
}

// For returns the cell registered for k and whether one was found.
func (m *Matrix[P]) For(k Kind) (Workload[P], bool) {
	i, ok := m.byKind[k]
	if !ok {
		var zero Workload[P]
		return zero, false
	}
	return m.cells[i], true
}

// All returns every cell in insertion order.
func (m *Matrix[P]) All() []Workload[P] {
	out := make([]Workload[P], len(m.cells))
	copy(out, m.cells)
	return out
}

// Packages returns the implementing package of every cell in insertion order.
func (m *Matrix[P]) Packages() []P {
	out := make([]P, 0, len(m.cells))
	for _, c := range m.cells {
		out = append(out, c.Package)
	}
	return out
}

// Len returns the number of cells.
func (m *Matrix[P]) Len() int { return len(m.cells) }
