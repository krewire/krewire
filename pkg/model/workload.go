// Package core — workload registry (KWF-M8K2Q, KWL-K1N2Q).
package model

import (
	"strings"
)

// Kind is a descriptive copywriting taxonomy for common starter archetypes.
// Per ADR-R8C2P, workload kinds exist exclusively as educational and documentation
// copywriting, never as programmatic execution constraints or architectural silos.
// Krewire projects are fluid, composable Go packages that can evolve without restriction.
type Kind string

const (
	KindApp     Kind = "app"
	KindCLI     Kind = "cli"
	KindSite    Kind = "site"
	KindBook    Kind = "book"
	KindWorker  Kind = "worker"
	KindService Kind = "service"
	KindGateway Kind = "gateway"
	KindInfra   Kind = "infra"
)

// AllKinds lists every canonical starter Kind in recommended order.
var AllKinds = []Kind{KindApp, KindCLI, KindSite, KindBook, KindWorker, KindService, KindGateway, KindInfra}

// IsCanonical reports whether k is one of the eight canonical starter kinds.
func (k Kind) IsCanonical() bool {
	for _, kind := range AllKinds {
		if k == kind {
			return true
		}
	}
	return false
}

// IsValid reports whether k is a valid Kind name. Any non-empty string is valid,
// enabling projects to evolve freely.
func (k Kind) IsValid() bool {
	return strings.TrimSpace(string(k)) != ""
}

// ParseKind parses s as a Kind, trimming surrounding whitespace.
// Unknown kinds are accepted to ensure projects can evolve fluidly.
func ParseKind(s string) (Kind, error) {
	k := Kind(strings.TrimSpace(s))
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

// MaturityLevel represents the objective readiness and support status of a workload or capability.
type MaturityLevel string

const (
	// MaturityStable: Production-qualified, full automated test coverage, performance baseline verified, backwards compatible.
	MaturityStable MaturityLevel = "Stable"
	// MaturityUsable: Functional and tested for single-node / standard use cases, documented, active production use.
	MaturityUsable MaturityLevel = "Usable"
	// MaturityExperimental: Working prototype / preview, public API may change, opt-in flag required.
	MaturityExperimental MaturityLevel = "Experimental"
	// MaturityPlanned: Defined in roadmap and specifications, implementation not yet complete.
	MaturityPlanned MaturityLevel = "Planned"
	// MaturityUnsupported: Explicitly out of scope or deprecated.
	MaturityUnsupported MaturityLevel = "Unsupported"
)

// WorkloadReadiness represents the evidence-backed maturity record for a workload kind.
type WorkloadReadiness struct {
	Kind        Kind          `json:"kind"`
	Title       string        `json:"title"`
	Status      Status        `json:"status"`
	Maturity    MaturityLevel `json:"maturity"`
	Package     string        `json:"package"`
	Evidence    string        `json:"evidence"`
	Limitations []string      `json:"limitations,omitempty"`
	Owner       string        `json:"owner"`
}

// CanonicalWorkloadReadiness returns the official evidence-backed readiness records for all 8 canonical workloads.
func CanonicalWorkloadReadiness() []WorkloadReadiness {
	return []WorkloadReadiness{
		{
			Kind:     KindSite,
			Title:    "Static Website (SSG)",
			Status:   StatusShipped,
			Maturity: MaturityStable,
			Package:  "pkg/web/ssg",
			Evidence: "Powers krewire.com in production. SSG compilation, assets pipeline, client HMR, multi-locale i18n.",
			Limitations: []string{
				"Dynamic server-side API routes require pairing with an app kind or gateway kind.",
			},
			Owner: "@krewire/team-runtime",
		},
		{
			Kind:     KindBook,
			Title:    "Documentation & Books",
			Status:   StatusShipped,
			Maturity: MaturityStable,
			Package:  "pkg/book",
			Evidence: "Powers ecosystem documentation in English & Indonesian with automated search indexing and TOC.",
			Limitations: []string{
				"Manuscript compilation currently focuses on Markdown; AsciiDoc support is planned.",
			},
			Owner: "@krewire/team-runtime",
		},
		{
			Kind:     KindApp,
			Title:    "Fullstack Web Application",
			Status:   StatusShipped,
			Maturity: MaturityStable,
			Package:  "pkg/app, pkg/web, pkg/runtime",
			Evidence: "Fullstack SSR + WebAssembly islands compilation, routing middleware, 18 .kiw conformance tests passing.",
			Limitations: []string{
				"SSR state hydration for deeply nested reactive loops is under ongoing optimization.",
			},
			Owner: "@krewire/team-runtime",
		},
		{
			Kind:     KindCLI,
			Title:    "Command-Line Tool",
			Status:   StatusShipped,
			Maturity: MaturityStable,
			Package:  "pkg/tui, pkg/term",
			Evidence: "Unified toolchain krew drives all ecosystem workflows. Reproducible pure-Go binary compilation verified.",
			Limitations: []string{
				"Terminal graphics protocols (Sixel, Kitty) are unbundled and optional.",
			},
			Owner: "@krewire/team-runtime",
		},
		{
			Kind:     KindGateway,
			Title:    "API Gateway & Reverse Proxy",
			Status:   StatusShipped,
			Maturity: MaturityStable,
			Package:  "pkg/web/gateway",
			Evidence: "Reverse proxy ingress routing, rate limiting token bucket, header transformation, and health checks.",
			Limitations: []string{
				"Dynamic service discovery beyond static route tables requires external DNS or consul adapters.",
			},
			Owner: "@krewire/team-runtime",
		},
		{
			Kind:     KindWorker,
			Title:    "Background Queue Worker",
			Status:   StatusShipped,
			Maturity: MaturityUsable,
			Package:  "pkg/cloud/worker, pkg/lifecycle",
			Evidence: "Priority queue processing, concurrency bounds, exponential backoff retries, Redis/Postgres drivers.",
			Limitations: []string{
				"Redis and PostgreSQL queue backends supported; distributed Kafka/RabbitMQ adapters planned.",
			},
			Owner: "@krewire/team-cloud",
		},
		{
			Kind:     KindService,
			Title:    "Headless Microservice",
			Status:   StatusShipped,
			Maturity: MaturityUsable,
			Package:  "pkg/cloud/service, pkg/resilience",
			Evidence: "Headless REST microservice runtime, request correlation tracing, circuit breakers, rate limiting.",
			Limitations: []string{
				"Direct HTTP/REST and SSE supported; gRPC and protobuf support in roadmap.",
			},
			Owner: "@krewire/team-cloud",
		},
		{
			Kind:     KindInfra,
			Title:    "Declarative Cloud Infrastructure",
			Status:   StatusShipped,
			Maturity: MaturityExperimental,
			Package:  "pkg/cloud/infra, pkg/cloud/deploy",
			Evidence: "Declarative resource topology planning (kiw build --plan) and state reconciliation parser.",
			Limitations: []string{
				"Provider drivers currently target local Docker and Linux systemd; cloud PaaS drivers are experimental.",
			},
			Owner: "@krewire/team-cloud",
		},
	}
}

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
