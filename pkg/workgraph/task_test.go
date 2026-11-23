package workgraph

import (
	"reflect"
	"strings"
	"testing"
)

func fixtureTasks(t *testing.T, ws *Workspace) *TaskGraph {
	t.Helper()
	tg, err := BuildTasks(ws.Graph, DefaultTasks())
	if err != nil {
		t.Fatal(err)
	}
	return tg
}

func TestBuildTasksExpansion(t *testing.T) {
	tg := fixtureTasks(t, fixture(t))
	if n := len(tg.Order()); n != 16 {
		t.Fatalf("targets = %d, want 16 (4 packages x 4 tasks)", n)
	}
	cases := map[string]struct {
		workDir string
		argv    string
		deps    []string
	}{
		TargetID("format", pBase): {"core", "gofmt -l ./base", nil},
		TargetID("vet", pTool):    {"tools", "go vet example.com/tools/cmd/tool", nil},
		// build#mid must not wait on build#top: that edge is test-only.
		TargetID("build", pMid):  {"core", "go build example.com/core/mid", []string{TargetID("build", pBase)}},
		TargetID("build", pTool): {"tools", "go build example.com/tools/cmd/tool", []string{TargetID("build", pBase)}},
		TargetID("test", pTop):   {"core", "go test example.com/core/top", []string{TargetID("build", pTop)}},
	}
	for id, want := range cases {
		tgt, ok := tg.Target(id)
		if !ok {
			t.Fatalf("missing target %s", id)
		}
		if tgt.WorkDir != want.workDir || strings.Join(tgt.Argv, " ") != want.argv || !reflect.DeepEqual(tgt.Deps, want.deps) {
			t.Errorf("%s = %s %v %v", id, tgt.WorkDir, tgt.Argv, tgt.Deps)
		}
	}
	pos := map[string]int{}
	for i, id := range tg.Order() {
		pos[id] = i
	}
	for _, tgt := range tg.Targets() {
		for _, d := range tgt.Deps {
			if pos[d] > pos[tgt.ID] {
				t.Errorf("%s ordered before its dependency %s", tgt.ID, d)
			}
		}
	}
}

func TestBuildTasksRejects(t *testing.T) {
	g := fixture(t).Graph
	cmd := []string{"true"}
	for name, specs := range map[string][]TaskSpec{
		"no command": {{Name: "a"}},
		"duplicate":  {{Name: "a", Command: cmd}, {Name: "a", Command: cmd}},
		"unknown":    {{Name: "a", Command: cmd, DependsOn: []string{"^zz"}}},
		"cycle":      {{Name: "a", Command: cmd, DependsOn: []string{"b"}}, {Name: "b", Command: cmd, DependsOn: []string{"a"}}},
	} {
		if _, err := BuildTasks(g, specs); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	tg, err := BuildTasks(g, []TaskSpec{
		{Name: "tidy", Kind: TaskGenerate, On: KindModule, Command: []string{"go", "mod", "tidy"}},
		{Name: "lint", Kind: TaskLint, Command: cmd, Only: []string{"core/*"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if tgt, ok := tg.Target(TargetID("tidy", mTools)); !ok || tgt.WorkDir != "tools" {
		t.Fatalf("module target = %+v", tgt)
	}
	if _, ok := tg.Target(TargetID("lint", pTool)); ok {
		t.Fatal("Only filter ignored")
	}
}

func TestValidatePlan(t *testing.T) {
	tg := fixtureTasks(t, fixture(t))
	ok := []PlanStep{{Target: TargetID("test", pTop), DependsOn: []string{TargetID("build", pBase)}}}
	if err := tg.ValidatePlan(ok); err != nil {
		t.Fatalf("valid plan rejected: %v", err)
	}
	bad := []PlanStep{
		{Target: "test#package:example.com/core/ghost"},
		{Target: TargetID("test", pBase), DependsOn: []string{TargetID("build", pTop)}},
	}
	err := tg.ValidatePlan(bad)
	if err == nil || !strings.Contains(err.Error(), "unknown target") || !strings.Contains(err.Error(), "does not depend on") {
		t.Fatalf("invalid plan: %v", err)
	}
}
