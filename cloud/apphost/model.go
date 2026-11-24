// Package apphost is Krewire's declarative application resource graph and
// local orchestrator.
//
// A topology is declared under the `topology:` key of krewire.yaml. It names
// every resource an application needs locally — services, workers, gateways,
// databases, caches, queues, object stores and external dependencies — with
// their endpoints, environment wiring, dependencies, readiness probes, restart
// policy and shutdown timeout. Parse and Validate turn that declaration into a
// checked Topology; NewPlan orders it; Orchestrator starts it in dependency
// order, waits for readiness, injects service discovery environment, supervises
// restarts, aggregates logs, exposes machine-readable state and stops
// everything in reverse order without orphaning child processes.
//
// The same Topology value is the contract for deterministic tooling (krew) and
// AI-assisted workflows: proposals are validated with ValidateProposal, and
// every executed operation passes the optional Approver and is journaled.
package apphost

import (
	"encoding/json"
	"fmt"
	"time"

	"gopkg.in/yaml.v3"
)

// SchemaVersion is the only topology schema version this package understands.
const SchemaVersion = 1

// TopologyKey is the krewire.yaml key that holds the resource graph.
const TopologyKey = "topology"

// Kind classifies a resource. It is descriptive (it drives defaults, plans and
// diagnostics); how a resource runs is chosen by its Run, Container or
// External block.
type Kind string

// Resource kinds.
const (
	KindService     Kind = "service"
	KindWorker      Kind = "worker"
	KindGateway     Kind = "gateway"
	KindDatabase    Kind = "database"
	KindCache       Kind = "cache"
	KindQueue       Kind = "queue"
	KindObjectStore Kind = "objectstore"
	KindExternal    Kind = "external"
)

// Kinds lists every valid Kind, in documentation order.
var Kinds = []Kind{KindService, KindWorker, KindGateway, KindDatabase, KindCache, KindQueue, KindObjectStore, KindExternal}

// RestartPolicy controls what happens when a running resource exits.
type RestartPolicy string

// Restart policies.
const (
	RestartNever     RestartPolicy = "never"
	RestartOnFailure RestartPolicy = "on-failure"
	RestartAlways    RestartPolicy = "always"
)

// Defaults applied by Normalize when a field is left empty.
const (
	DefaultShutdownTimeout = 10 * time.Second
	DefaultReadyTimeout    = 30 * time.Second
	DefaultProbeInterval   = 200 * time.Millisecond
	DefaultMaxRestarts     = 3
	DefaultRestartBackoff  = 500 * time.Millisecond
	DefaultHost            = "127.0.0.1"
)

// Duration is a time.Duration written as a Go duration string ("500ms") in
// YAML and JSON, so the machine-readable state stays human-readable too.
type Duration time.Duration

// D returns the value as a time.Duration.
func (d Duration) D() time.Duration { return time.Duration(d) }

// String formats the duration like time.Duration.
func (d Duration) String() string { return time.Duration(d).String() }

// UnmarshalYAML parses a duration string.
func (d *Duration) UnmarshalYAML(n *yaml.Node) error {
	var s string
	if err := n.Decode(&s); err != nil {
		return err
	}
	return d.parse(s)
}

// MarshalJSON writes the duration string.
func (d Duration) MarshalJSON() ([]byte, error) { return json.Marshal(d.String()) }

// UnmarshalJSON parses a duration string.
func (d *Duration) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	return d.parse(s)
}

func (d *Duration) parse(s string) error {
	v, err := time.ParseDuration(s)
	if err != nil {
		return fmt.Errorf("invalid duration %q: %w", s, err)
	}
	*d = Duration(v)
	return nil
}

// Topology is the declarative resource graph of one application.
type Topology struct {
	Version   int                  `yaml:"version" json:"version"`
	Name      string               `yaml:"name,omitempty" json:"name,omitempty"`
	Resources map[string]*Resource `yaml:"resources" json:"resources"`
}

// Resource is one node of the graph.
type Resource struct {
	// Name is filled from the map key by Parse/Normalize.
	Name string `yaml:"-" json:"name"`
	Kind Kind   `yaml:"kind" json:"kind"`

	// Exactly one of Run, Container or External selects how it runs.
	Run       *RunSpec       `yaml:"run,omitempty" json:"run,omitempty"`
	Container *ContainerSpec `yaml:"container,omitempty" json:"container,omitempty"`
	External  *ExternalSpec  `yaml:"external,omitempty" json:"external,omitempty"`

	Endpoints []Endpoint        `yaml:"endpoints,omitempty" json:"endpoints,omitempty"`
	Env       map[string]string `yaml:"env,omitempty" json:"env,omitempty"`
	// References inject service discovery variables for the named resources
	// and make them dependencies.
	References []string `yaml:"references,omitempty" json:"references,omitempty"`
	// DependsOn orders startup without injecting discovery variables.
	DependsOn []string `yaml:"dependsOn,omitempty" json:"dependsOn,omitempty"`

	Readiness       *Probe   `yaml:"readiness,omitempty" json:"readiness,omitempty"`
	Restart         Restart  `yaml:"restart,omitempty" json:"restart"`
	ShutdownTimeout Duration `yaml:"shutdownTimeout,omitempty" json:"shutdownTimeout"`
}

// RunSpec launches a local child process.
type RunSpec struct {
	Command string   `yaml:"command" json:"command"`
	Args    []string `yaml:"args,omitempty" json:"args,omitempty"`
	Dir     string   `yaml:"dir,omitempty" json:"dir,omitempty"`
}

// ContainerSpec launches an OCI container through a ContainerRuntime.
type ContainerSpec struct {
	Image string   `yaml:"image" json:"image"`
	Args  []string `yaml:"args,omitempty" json:"args,omitempty"`
}

// ExternalSpec declares a dependency Krewire does not run (a managed
// database, a shared queue). It is probed, never started or stopped.
type ExternalSpec struct {
	URL string `yaml:"url,omitempty" json:"url,omitempty"`
}

// Endpoint is a named network endpoint. Port 0 asks the orchestrator to
// allocate a free port at start; TargetPort is the in-container port.
type Endpoint struct {
	Name       string `yaml:"name" json:"name"`
	Scheme     string `yaml:"scheme,omitempty" json:"scheme,omitempty"`
	Host       string `yaml:"host,omitempty" json:"host,omitempty"`
	Port       int    `yaml:"port,omitempty" json:"port"`
	TargetPort int    `yaml:"targetPort,omitempty" json:"targetPort,omitempty"`
	URL        string `yaml:"url,omitempty" json:"url,omitempty"`
}

// Probe is a readiness check. Exactly one of TCP, HTTP or Exec is set; TCP
// and HTTP name an endpoint of the same resource.
type Probe struct {
	TCP      string     `yaml:"tcp,omitempty" json:"tcp,omitempty"`
	HTTP     *HTTPProbe `yaml:"http,omitempty" json:"http,omitempty"`
	Exec     *RunSpec   `yaml:"exec,omitempty" json:"exec,omitempty"`
	Interval Duration   `yaml:"interval,omitempty" json:"interval"`
	Timeout  Duration   `yaml:"timeout,omitempty" json:"timeout"`
}

// HTTPProbe succeeds when GET endpoint+Path returns a 2xx/3xx status.
type HTTPProbe struct {
	Endpoint string `yaml:"endpoint" json:"endpoint"`
	Path     string `yaml:"path,omitempty" json:"path,omitempty"`
}

// Restart configures supervision after a resource is ready.
type Restart struct {
	Policy      RestartPolicy `yaml:"policy,omitempty" json:"policy"`
	MaxRestarts int           `yaml:"maxRestarts,omitempty" json:"maxRestarts"`
	Backoff     Duration      `yaml:"backoff,omitempty" json:"backoff"`
}

// Runner reports which launch mechanism the resource uses.
func (r *Resource) Runner() string {
	switch {
	case r.Run != nil:
		return "process"
	case r.Container != nil:
		return "container"
	case r.External != nil:
		return "external"
	}
	return ""
}

// Dependencies returns the union of References, DependsOn and resources named
// in environment expressions, sorted and de-duplicated.
func (r *Resource) Dependencies() []string {
	set := map[string]bool{}
	for _, d := range r.References {
		set[d] = true
	}
	for _, d := range r.DependsOn {
		set[d] = true
	}
	for _, v := range r.Env {
		refs, _ := parseExpressions(v)
		for _, x := range refs {
			if x.Resource != selfRef {
				set[x.Resource] = true
			}
		}
	}
	return sortedKeys(set)
}

// endpoint returns the named endpoint, or the primary one for "".
func (r *Resource) endpoint(name string) (Endpoint, bool) {
	if name == "" {
		if len(r.Endpoints) == 0 {
			return Endpoint{}, false
		}
		return r.Endpoints[0], true
	}
	for _, e := range r.Endpoints {
		if e.Name == name {
			return e, true
		}
	}
	return Endpoint{}, false
}
