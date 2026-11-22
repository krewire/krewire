package model

import "testing"

func TestProjectValidate(t *testing.T) {
	cases := []struct {
		p  Project
		ok bool
	}{
		{Project{Name: "demo"}, true},
		{Project{Name: "my-app", ModulePath: "example.com/my-app"}, true},
		{Project{Name: "fluid-proj"}, true},
		{Project{Name: ""}, false},
		{Project{Name: "Bad_Name"}, false},
		{Project{Name: "demo", ConfigPath: "krewire.yaml"}, true},
		{Project{Name: "demo", ConfigPath: "ssg.yaml"}, false},
		{Project{Name: "demo", ModulePath: "example.com/a/../b"}, false},
		{Project{Name: "demo", ModulePath: "example.com//a"}, false},
		{Project{Name: "demo", ModulePath: "/leading"}, false},
		{Project{Name: "demo", ModulePath: "trailing/"}, false},
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
