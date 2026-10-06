package lifecycle

import (
	"context"
	"testing"

	"github.com/krewire/krewire/packages/kern/errs"
	"github.com/krewire/krewire/packages/kern/model"
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
	_, err := New(model.Project{Name: "", Kind: model.KindApp})
	if err == nil {
		t.Error("New should fail for invalid project")
	}
	_, err = New(model.Project{Name: "demo", Kind: model.KindApp})
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
	k, _ := New(model.Project{Name: "demo", Kind: model.KindApp})
	called := false
	k.Use(testModule{name: "mod", init: func(k *Kernel) error {
		k.RegisterHandler(model.KindApp, func(ctx context.Context, w model.Workload[string]) errs.ExitCode {
			called = true
			return errs.ExitCodeSuccess
		})
		return nil
	}})
	if err := k.Boot(context.Background()); err != nil {
		t.Fatalf("Boot error = %v", err)
	}
	// The lifecycle takes the workload the caller declares; the kernel holds no
	// roster of its own to look one up in.
	w := model.Workload[string]{Kind: model.KindApp, Package: "example/app"}
	code := k.Execute(context.Background(), w)
	if code != errs.ExitCodeSuccess || !called {
		t.Errorf("Execute = %v called=%v, want success true", code, called)
	}
	// Unknown workload
	unknown := model.Workload[string]{Kind: "unknown"}
	if code := k.Execute(context.Background(), unknown); code != errs.ExitCodeUsage {
		t.Errorf("Execute unknown = %v, want Usage", code)
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
			e.Register(model.KindApp, func(ctx context.Context, w model.Workload[string]) errs.ExitCode {
				return errs.ExitCodeSuccess
			})
			_ = e.Execute(context.Background(), model.Workload[string]{Kind: model.KindApp})
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
