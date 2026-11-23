package apphost

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// Issue codes reported by Validate. They are stable identifiers tooling and
// AI workflows can match on.
const (
	CodeVersion       = "unsupported-version"
	CodeName          = "invalid-name"
	CodeKind          = "invalid-kind"
	CodeRunner        = "invalid-runner"
	CodeEndpoint      = "invalid-endpoint"
	CodeMissingDep    = "missing-dependency"
	CodeCycle         = "dependency-cycle"
	CodePortCollision = "port-collision"
	CodeEnvReference  = "invalid-env-reference"
	CodeProbe         = "invalid-probe"
	CodeRestart       = "invalid-restart"
)

// Issue is one validation finding.
type Issue struct {
	Code     string `json:"code"`
	Resource string `json:"resource,omitempty"`
	Field    string `json:"field,omitempty"`
	Message  string `json:"message"`
	Hint     string `json:"hint,omitempty"`
}

func (i Issue) String() string {
	s := i.Code + ": "
	if i.Resource != "" {
		s += i.Resource
		if i.Field != "" {
			s += "." + i.Field
		}
		s += ": "
	}
	s += i.Message
	if i.Hint != "" {
		s += " (hint: " + i.Hint + ")"
	}
	return s
}

// ValidationError aggregates every Issue found, sorted deterministically.
type ValidationError struct {
	Issues []Issue `json:"issues"`
}

func (e *ValidationError) Error() string {
	lines := make([]string, len(e.Issues))
	for i, is := range e.Issues {
		lines[i] = is.String()
	}
	return "apphost: invalid topology:\n  " + strings.Join(lines, "\n  ")
}

// Has reports whether an issue with the code was found.
func (e *ValidationError) Has(code string) bool {
	for _, i := range e.Issues {
		if i.Code == code {
			return true
		}
	}
	return false
}

var resourceNameRe = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

// Validate checks the whole graph before anything is started: schema version,
// names, kinds, runners, endpoints, missing dependencies, cycles, fixed-port
// collisions, environment references, probes and restart policy. It returns
// nil or a *ValidationError listing every issue.
func (t *Topology) Validate() error {
	var issues []Issue
	add := func(code, res, field, hint, format string, args ...any) {
		issues = append(issues, Issue{Code: code, Resource: res, Field: field, Message: fmt.Sprintf(format, args...), Hint: hint})
	}
	if t.Version != SchemaVersion {
		add(CodeVersion, "", "version", fmt.Sprintf("set `version: %d`", SchemaVersion), "schema version %d is not supported", t.Version)
	}
	if len(t.Resources) == 0 {
		add(CodeName, "", "resources", "declare at least one resource", "topology has no resources")
	}
	ports := map[int]string{}
	for _, name := range t.Names() {
		r := t.Resources[name]
		if !resourceNameRe.MatchString(name) || name == selfRef {
			add(CodeName, name, "", "use kebab-case, not \"self\"", "invalid resource name")
		}
		if !validKind(r.Kind) {
			add(CodeKind, name, "kind", fmt.Sprintf("one of %v", Kinds), "unknown kind %q", r.Kind)
		}
		n := 0
		for _, set := range []bool{r.Run != nil, r.Container != nil, r.External != nil} {
			if set {
				n++
			}
		}
		switch {
		case n != 1:
			add(CodeRunner, name, "", "set exactly one of run, container or external", "resource declares %d launch blocks", n)
		case r.Run != nil && r.Run.Command == "":
			add(CodeRunner, name, "run.command", "", "command is required")
		case r.Container != nil && r.Container.Image == "":
			add(CodeRunner, name, "container.image", "", "image is required")
		case r.External != nil && r.External.URL == "" && len(r.Endpoints) == 0:
			add(CodeRunner, name, "external", "give external.url or a fixed endpoint", "external resource has no address")
		}
		seen := map[string]bool{}
		for i, e := range r.Endpoints {
			field := fmt.Sprintf("endpoints[%d]", i)
			if e.Name == "" || seen[e.Name] {
				add(CodeEndpoint, name, field, "endpoint names must be unique and non-empty", "invalid endpoint name %q", e.Name)
			}
			seen[e.Name] = true
			if e.Port < 0 || e.Port > 65535 {
				add(CodeEndpoint, name, field, "", "port %d out of range", e.Port)
			}
			if r.External != nil && e.Port == 0 && e.URL == "" {
				add(CodeEndpoint, name, field, "external endpoints need a fixed port or url", "dynamic port on external resource")
			}
			if r.Container != nil && e.TargetPort == 0 {
				add(CodeEndpoint, name, field, "set targetPort to the port inside the container", "container endpoint without targetPort")
			}
			if e.Port > 0 && r.External == nil {
				if other, dup := ports[e.Port]; dup {
					add(CodePortCollision, name, field, "use port 0 to allocate a free port", "port %d already used by %s", e.Port, other)
				} else {
					ports[e.Port] = name + "." + e.Name
				}
			}
		}
		for _, d := range append(append([]string{}, r.References...), r.DependsOn...) {
			if _, ok := t.Resources[d]; !ok {
				add(CodeMissingDep, name, "", fmt.Sprintf("declare resource %q or remove the reference", d), "depends on unknown resource %q", d)
			} else if d == name {
				add(CodeCycle, name, "", "remove the self dependency", "resource depends on itself")
			}
		}
		for _, key := range sortedEnvKeys(r.Env) {
			t.validateEnv(name, key, r.Env[key], add)
		}
		t.validateProbe(r, add)
		switch r.Restart.Policy {
		case RestartNever, RestartOnFailure, RestartAlways:
		default:
			add(CodeRestart, name, "restart.policy", "never, on-failure or always", "unknown restart policy %q", r.Restart.Policy)
		}
		if r.Restart.MaxRestarts < 0 {
			add(CodeRestart, name, "restart.maxRestarts", "", "must not be negative")
		}
	}
	if cyc := t.cycle(); cyc != nil {
		add(CodeCycle, cyc[0], "", "break the loop with an external resource or remove a dependency", "dependency cycle %s", strings.Join(cyc, " -> "))
	}
	if len(issues) == 0 {
		return nil
	}
	sort.SliceStable(issues, func(i, j int) bool {
		if issues[i].Resource != issues[j].Resource {
			return issues[i].Resource < issues[j].Resource
		}
		return issues[i].Code < issues[j].Code
	})
	return &ValidationError{Issues: issues}
}

type addFunc func(code, res, field, hint, format string, args ...any)

func (t *Topology) validateEnv(name, key, val string, add addFunc) {
	field := "env." + key
	refs, err := parseExpressions(val)
	if err != nil {
		add(CodeEnvReference, name, field, "use ${resource.url|host|port} or ${resource.endpoint.port}", "%v", err)
		return
	}
	for _, x := range refs {
		target := t.Resources[name]
		if x.Resource != selfRef {
			var ok bool
			if target, ok = t.Resources[x.Resource]; !ok {
				add(CodeEnvReference, name, field, fmt.Sprintf("declare resource %q", x.Resource), "${%s} names unknown resource %q", x.Raw, x.Resource)
				continue
			}
		}
		if x.Endpoint == "" && x.Property == propURL && target.External != nil && target.External.URL != "" {
			continue
		}
		if _, ok := target.endpoint(x.Endpoint); !ok {
			add(CodeEnvReference, name, field, "declare the endpoint on "+target.Name, "${%s}: %q has no endpoint %q", x.Raw, target.Name, x.Endpoint)
		}
	}
}

func (t *Topology) validateProbe(r *Resource, add addFunc) {
	p := r.Readiness
	if p == nil {
		return
	}
	n := 0
	if p.TCP != "" {
		n++
		if _, ok := r.endpoint(p.TCP); !ok {
			add(CodeProbe, r.Name, "readiness.tcp", "", "unknown endpoint %q", p.TCP)
		}
	}
	if p.HTTP != nil {
		n++
		if _, ok := r.endpoint(p.HTTP.Endpoint); !ok || p.HTTP.Endpoint == "" {
			add(CodeProbe, r.Name, "readiness.http.endpoint", "", "unknown endpoint %q", p.HTTP.Endpoint)
		}
	}
	if p.Exec != nil {
		n++
		if p.Exec.Command == "" {
			add(CodeProbe, r.Name, "readiness.exec.command", "", "command is required")
		}
	}
	if n != 1 {
		add(CodeProbe, r.Name, "readiness", "set exactly one of tcp, http or exec", "probe declares %d checks", n)
	}
}

func validKind(k Kind) bool {
	for _, v := range Kinds {
		if v == k {
			return true
		}
	}
	return false
}

func sortedEnvKeys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
