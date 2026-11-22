package lifecycle

import (
	"context"
	"errors"
	"testing"

	"github.com/krewire/krewire/pkg/errs"
	"github.com/krewire/krewire/pkg/model"
)

type testModule struct {
	name string
	init func(*Kernel) error
}

func (m testModule) Name() string { return m.name }
func (m testModule) Init(k *Kernel) error {
	if m.init != nil {
		return m.init(k)
	}
	return nil
}

func TestNewValidatesProject(t *testing.T) {
	_, err := New(model.Project{Name: ""})
	if err == nil {
		t.Error("New should fail for invalid project")
	}
	_, err = New(model.Project{Name: "demo"})
	if err != nil {
		t.Errorf("New error = %v, want nil", err)
	}
}

func TestRegistryDuplicate(t *testing.T) {
	r := NewRegistry()
	if err := r.Register(testModule{name: "a"}); err != nil {
		t.Fatalf("Register error = %v", err)
	}
	if err := r.Register(testModule{name: "a"}); err == nil {
		t.Error("duplicate Register should fail")
	}
}

func TestKernelBootAndExecute(t *testing.T) {
	k, _ := New(model.Project{Name: "demo"})
	called := false
	k.Use(testModule{name: "mod", init: func(k *Kernel) error {
		k.RegisterHandler(func(ctx context.Context) errs.ExitCode {
			called = true
			return errs.ExitCodeSuccess
		})
		return nil
	}})
	if err := k.Boot(context.Background()); err != nil {
		t.Fatalf("Boot error = %v", err)
	}
	code := k.Execute(context.Background())
	if code != errs.ExitCodeSuccess || !called {
		t.Errorf("Execute = %v called=%v, want success true", code, called)
	}
}

func TestSupervisor(t *testing.T) {
	s := NewSupervisor()
	hit := false
	s.OnReload(func() { hit = true })
	s.Reload()
	if !hit {
		t.Error("OnReload not triggered")
	}
	// Start/Stop with no modules should succeed
	if err := s.Start(context.Background()); err != nil {
		t.Errorf("Start error = %v", err)
	}
	if err := s.Stop(context.Background()); err != nil {
		t.Errorf("Stop error = %v", err)
	}
}

type depModule struct {
	name string
	deps []string
}

func (m depModule) Name() string        { return m.name }
func (m depModule) Init(*Kernel) error  { return nil }
func (m depModule) DependsOn() []string { return m.deps }

func TestRegistry_CircularDependencyDoesNotHang(t *testing.T) {
	r := NewRegistry()
	_ = r.Register(depModule{name: "modA", deps: []string{"modB"}})
	_ = r.Register(depModule{name: "modB", deps: []string{"modA"}})

	// Must finish immediately and not hang in infinite loop
	mods := r.Ordered()
	if len(mods) != 2 {
		t.Fatalf("expected 2 modules, got %d", len(mods))
	}
}

func TestRegistry_TopologicalOrdering(t *testing.T) {
	r := NewRegistry()
	// C depends on B, B depends on A
	_ = r.Register(depModule{name: "modC", deps: []string{"modB"}})
	_ = r.Register(depModule{name: "modB", deps: []string{"modA"}})
	_ = r.Register(depModule{name: "modA", deps: nil})

	mods := r.Ordered()
	if len(mods) != 3 {
		t.Fatalf("expected 3 modules, got %d", len(mods))
	}
	if mods[0].Name() != "modA" || mods[1].Name() != "modB" || mods[2].Name() != "modC" {
		t.Errorf("unexpected ordering: %s, %s, %s", mods[0].Name(), mods[1].Name(), mods[2].Name())
	}
}

func TestRegistry_ConcurrentAccess(t *testing.T) {
	r := NewRegistry()
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			_ = r.Register(testModule{name: "mod"})
			_, _ = r.Resolve("mod")
			_ = r.Ordered()
			done <- true
		}(i)
	}
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestExecutor_ConcurrentAccess(t *testing.T) {
	e := newExecutor()
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			e.Register(func(ctx context.Context) errs.ExitCode {
				return errs.ExitCodeSuccess
			})
			_ = e.Execute(context.Background())
			done <- true
		}(i)
	}
	for i := 0; i < 10; i++ {
		<-done
	}
}

type errStopModule struct {
	stopped *bool
	err     error
}

func (m errStopModule) Start(context.Context) error { return nil }
func (m errStopModule) Stop(context.Context) error {
	*m.stopped = true
	return m.err
}

func TestSupervisor_StopContinuesOnError(t *testing.T) {
	s := NewSupervisor()
	stopped1 := false
	stopped2 := false

	m1 := errStopModule{stopped: &stopped1, err: nil}
	m2 := errStopModule{stopped: &stopped2, err: errs.FailureError("stop error")}

	s.Add(m1)
	s.Add(m2)

	err := s.Stop(context.Background())
	if err == nil {
		t.Error("expected combined stop error")
	}
	if !stopped1 || !stopped2 {
		t.Errorf("all modules should be stopped: m1=%v, m2=%v", stopped1, stopped2)
	}
}

type startableModule struct {
	testModule
	started bool
	stopped bool
}

func (m *startableModule) Start(context.Context) error { m.started = true; return nil }
func (m *startableModule) Stop(context.Context) error  { m.stopped = true; return nil }

type startErrModule struct{}

func (m startErrModule) Start(context.Context) error { return errs.FailureError("start failed") }
func (m startErrModule) Stop(context.Context) error  { return nil }

func TestKernel_StartableAndShutdown(t *testing.T) {
	k, _ := New(model.Project{Name: "demo"})
	sm := &startableModule{testModule: testModule{name: "daemon"}}
	k.Use(sm)

	if err := k.Boot(context.Background()); err != nil {
		t.Fatalf("Boot error: %v", err)
	}
	if err := k.Supervisor.Start(context.Background()); err != nil {
		t.Fatalf("Supervisor.Start error: %v", err)
	}
	if !sm.started {
		t.Errorf("expected startableModule to be started")
	}

	health := k.Supervisor.Health()
	if len(health) == 0 {
		t.Errorf("expected non-empty health map")
	}

	if err := k.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown error: %v", err)
	}
	if !sm.stopped {
		t.Errorf("expected startableModule to be stopped")
	}
}

func TestKernel_BootErrors(t *testing.T) {
	// Boot error with *errs.Error
	k1, _ := New(model.Project{Name: "demo"})
	k1.Use(testModule{name: "err1", init: func(*Kernel) error {
		return errs.UsageError("err1-usage")
	}})
	if err := k1.Boot(context.Background()); err == nil {
		t.Errorf("expected Boot error for *errs.Error")
	}

	// Boot error with generic error
	k2, _ := New(model.Project{Name: "demo"})
	k2.Use(testModule{name: "err2", init: func(*Kernel) error {
		return errors.New("err2-generic")
	}})
	if err := k2.Boot(context.Background()); err == nil {
		t.Errorf("expected Boot error for generic error")
	}
}

func TestExecutor_NilHandler(t *testing.T) {
	e := newExecutor()
	if code := e.Execute(context.Background()); code != errs.ExitCodeSuccess {
		t.Errorf("Execute with nil handler = %v, want ExitCodeSuccess", code)
	}
}

func TestSupervisor_StartError(t *testing.T) {
	s := NewSupervisor()
	s.Add(startErrModule{})
	if err := s.Start(context.Background()); err == nil {
		t.Errorf("expected Start error")
	}
	health := s.Health()
	if health["0"] == nil {
		t.Errorf("expected health[0] to have error")
	}
}

func TestRegistry_ValidationErrors(t *testing.T) {
	r := NewRegistry()
	if err := r.Register(nil); err == nil {
		t.Errorf("expected error registering nil module")
	}
	if err := r.Register(testModule{name: ""}); err == nil {
		t.Errorf("expected error registering empty name module")
	}

	// Module with non-existent dependency is safely ignored in ordering
	_ = r.Register(depModule{name: "orphan", deps: []string{"ghost"}})
	ordered := r.Ordered()
	if len(ordered) != 1 || ordered[0].Name() != "orphan" {
		t.Errorf("expected orphan module in ordered list")
	}
}
