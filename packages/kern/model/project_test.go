// Tests for KWL-CORE-K1N2Q
package model

import "testing"

// Spec: KWL-CORE-K1N2Q KWL-CORE-020 Scope: Unit
func TestProjectValidate(t *testing.T) {
	cases := []struct {
		p  Project
		ok bool
	}{
		{Project{Name: "demo", Kind: KindApp}, true},
		{Project{Name: "my-app", Kind: KindCLI, ModulePath: "example.com/my-app"}, true},
		{Project{Name: "", Kind: KindApp}, false},
		{Project{Name: "Bad_Name", Kind: KindApp}, false},
		{Project{Name: "demo", Kind: Kind("unknown")}, false},
		{Project{Name: "demo", Kind: KindApp, ConfigPath: "krewire.yaml"}, true},
		{Project{Name: "demo", Kind: KindApp, ConfigPath: "ssg.yaml"}, false},
		{Project{Name: "demo", Kind: KindApp, ModulePath: "example.com/a/../b"}, false},
		{Project{Name: "demo", Kind: KindApp, ModulePath: "example.com//a"}, false},
		{Project{Name: "demo", Kind: KindApp, ModulePath: "/leading"}, false},
		{Project{Name: "demo", Kind: KindApp, ModulePath: "trailing/"}, false},
	}
	for i, c := range cases {
		err := c.p.Validate()
		if c.ok && err != nil {
			t.Errorf("case %d Validate error = %v, want nil", i, err)
		}
		if !c.ok && err == nil {
			t.Errorf("case %d Validate succeeded, want error", i)
		}
	}
}

// Spec: KWL-CORE-K1N2Q KWL-CORE-021 Scope: Unit
func TestValidateKrewireYamlPath(t *testing.T) {
	cases := []struct {
		in string
		ok bool
	}{
		{"krewire.yaml", true},
		{"./krewire.yaml", true},
		{"/abs/dir/krewire.yaml", true},
		{`C:\proj\krewire.yaml`, true}, // Windows-style path validates on any host
		{"ssg.yaml", false},
		{"", false},
	}
	for _, c := range cases {
		err := ValidateKrewireYamlPath(c.in)
		if c.ok && err != nil {
			t.Errorf("ValidateKrewireYamlPath(%q) error = %v, want nil", c.in, err)
		}
		if !c.ok && err == nil {
			t.Errorf("ValidateKrewireYamlPath(%q) succeeded, want error", c.in)
		}
	}
}

// batteryRules is an ecosystem's opt-in policy, declared by the ecosystem. The
// kernel only knows the rule shape; it holds no battery list of its own.
func batteryRules() []OptInRule[string] {
	return []OptInRule[string]{
		{For: KindApp, Owner: KindService, Blocked: []string{"github.com/krewire/cloud/service"}},
		{For: KindApp, Owner: KindInfra, Blocked: []string{"github.com/krewire/cloud/infra"}},
	}
}

// Spec: KWL-CORE-K1N2Q KWL-CORE-022 Scope: Unit
// A rule blocks an import for every kind except the one that owns it, matching
// on a path-segment boundary so a subpackage is covered and a mere prefix is not.
func TestViolatesOptIn(t *testing.T) {
	cases := []struct {
		name     string
		kind     Kind
		imported []string
		want     bool
	}{
		{"owner kind imports its own battery", KindService, []string{"github.com/krewire/cloud/service"}, false},
		{"infra kind imports infra", KindInfra, []string{"github.com/krewire/cloud/infra"}, false},
		{"app imports service", KindApp, []string{"github.com/krewire/cloud/service"}, true},
		{"app imports infra", KindApp, []string{"github.com/krewire/cloud/infra"}, true},
		{"app imports service subpackage", KindApp, []string{"github.com/krewire/cloud/service/gateway"}, true},
		{"service among many imports", KindApp, []string{"github.com/krewire/web", "github.com/krewire/cloud/service"}, true},
		{"prefix-only match is not a violation", KindApp, []string{"github.com/krewire/cloud/serviceless"}, false},
		{"unrelated import is fine", KindApp, []string{"github.com/krewire/tui"}, false},
		{"nothing imported", KindApp, nil, false},
		{"site kind imports service", KindSite, []string{"github.com/krewire/cloud/service"}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := ViolatesOptIn(c.kind, c.imported, batteryRules()); got != c.want {
				t.Errorf("ViolatesOptIn(%q, %v) = %v, want %v", c.kind, c.imported, got, c.want)
			}
		})
	}
}

// Spec: KWL-CORE-K1N2Q KWL-CORE-022 Scope: Unit
// The rule shape is generic over the import-path type, so a caller may use its
// own named string type rather than plain strings.
func TestViolatesOptInIsGeneric(t *testing.T) {
	type importPath string

	rules := []OptInRule[importPath]{
		{For: KindApp, Owner: KindService, Blocked: []importPath{"example.com/battery"}},
	}
	if !ViolatesOptIn(KindApp, []importPath{"example.com/battery/inner"}, rules) {
		t.Error("a named string path type should behave exactly like a plain string path")
	}
}
