package deploy

import (
	"context"
	"strings"
	"testing"
)

// stubDeployer is a Deployer that records the Options it was called with and
// returns a canned Result, so no real deployment I/O happens in tests.
type stubDeployer struct {
	target Target
	got    *Options
	result *Result
	err    error
}

func (s *stubDeployer) Target() Target { return s.target }

func (s *stubDeployer) Deploy(_ context.Context, opts Options) (*Result, error) {
	s.got = &opts
	return s.result, s.err
}

// isolateRegistry swaps in an empty Registry for the duration of the test and
// restores the previous one, so tests never mutate package-level state.
func isolateRegistry(t *testing.T) {
	t.Helper()
	prev := Registry
	Registry = map[Target]Deployer{}
	t.Cleanup(func() { Registry = prev })
}

func TestAllTargetsAreDistinct(t *testing.T) {
	targets := []Target{TargetBinary, TargetGhPages, TargetInfra, TargetSystemd}
	seen := map[Target]bool{}
	for _, target := range targets {
		if seen[target] {
			t.Errorf("duplicate target %q", target)
		}
		seen[target] = true
		if target == "" {
			t.Error("target must not be empty")
		}
	}
	if len(seen) != 4 {
		t.Errorf("got %d distinct targets, want 4", len(seen))
	}
}

func TestRegisterAndGetRoundTrip(t *testing.T) {
	isolateRegistry(t)

	want := &stubDeployer{target: TargetBinary, result: &Result{Success: true}}
	Register(want)

	got, err := Get(TargetBinary)
	if err != nil {
		t.Fatalf("Get error = %v", err)
	}
	if got != want {
		t.Errorf("Get returned %v, want the registered deployer", got)
	}
	if got.Target() != TargetBinary {
		t.Errorf("Target() = %q, want %q", got.Target(), TargetBinary)
	}
}

func TestGetUnknownTargetReturnsError(t *testing.T) {
	isolateRegistry(t)

	got, err := Get(Target("nowhere"))
	if err == nil {
		t.Fatal("Get error = nil, want error for an unregistered target")
	}
	if got != nil {
		t.Errorf("Get deployer = %v, want nil on error", got)
	}
	if !strings.Contains(err.Error(), "unknown target") {
		t.Errorf("error = %q, want it to mention %q", err, "unknown target")
	}
}

func TestGetUnregisteredKnownTargetReturnsError(t *testing.T) {
	// An empty registry must not satisfy any of the four declared targets.
	isolateRegistry(t)

	for _, target := range []Target{TargetBinary, TargetGhPages, TargetInfra, TargetSystemd} {
		if _, err := Get(target); err == nil {
			t.Errorf("Get(%q) error = nil, want error when the registry is empty", target)
		}
	}
}

func TestRegisterOverwritesSameTarget(t *testing.T) {
	isolateRegistry(t)

	first := &stubDeployer{target: TargetGhPages}
	second := &stubDeployer{target: TargetGhPages}
	Register(first)
	Register(second)

	got, err := Get(TargetGhPages)
	if err != nil {
		t.Fatalf("Get error = %v", err)
	}
	if got != second {
		t.Error("Register must overwrite the deployer registered for the same target")
	}
}

func TestDeployForwardsOptionsAndReturnsResult(t *testing.T) {
	isolateRegistry(t)

	want := &Result{Target: TargetGhPages, Success: true, URL: "https://example.github.io"}
	d := &stubDeployer{target: TargetGhPages, result: want}
	Register(d)

	opts := Options{
		Root:        "/tmp/site",
		Environment: "production",
		ArtifactDir: ".krewire/dist",
		Branch:      "gh-pages",
		Remote:      "origin",
		PlanOnly:    true,
		AutoApprove: false,
	}
	got, err := d.Deploy(context.Background(), opts)
	if err != nil {
		t.Fatalf("Deploy error = %v", err)
	}
	if got != want {
		t.Errorf("Deploy result = %v, want %v", got, want)
	}
	if d.got == nil {
		t.Fatal("Deploy did not receive Options")
	}
	if *d.got != opts {
		t.Errorf("Deploy options = %+v, want %+v", *d.got, opts)
	}
}

func TestDeployPropagatesError(t *testing.T) {
	isolateRegistry(t)

	d := &stubDeployer{target: TargetInfra, err: context.DeadlineExceeded}
	Register(d)

	got, err := d.Deploy(context.Background(), Options{})
	if err != context.DeadlineExceeded {
		t.Errorf("Deploy error = %v, want %v", err, context.DeadlineExceeded)
	}
	if got != nil {
		t.Errorf("Deploy result = %v, want nil on error", got)
	}
}
