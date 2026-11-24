package apicontract_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/krewire/krewire/apicontract"
)

// mutate compiles the test service and applies edit to a JSON round-trip of
// the document, the same path a committed baseline takes.
func mutate(t *testing.T, version string, edit func(d *apicontract.Document)) *apicontract.Document {
	t.Helper()
	raw, _ := testService("1.2.0").MustCompile().JSON()
	var d apicontract.Document
	if err := json.Unmarshal(raw, &d); err != nil {
		t.Fatal(err)
	}
	d.Version = version
	edit(&d)
	return &d
}

func kinds(r *apicontract.Report) map[string]apicontract.Severity {
	out := map[string]apicontract.Severity{}
	for _, c := range r.Changes {
		out[c.Kind+"@"+c.Endpoint+":"+c.Location] = c.Severity
	}
	return out
}

func TestDiffClassification(t *testing.T) {
	base := testService("1.2.0").MustCompile()
	B, N := apicontract.Breaking, apicontract.NonBreaking
	cases := []struct {
		name string
		edit func(d *apicontract.Document)
		want map[string]apicontract.Severity
	}{
		{"endpoint removed", func(d *apicontract.Document) { d.Endpoints = d.Endpoints[1:] },
			map[string]apicontract.Severity{"endpoint.removed@ListItems:": B}},
		{"endpoint added", func(d *apicontract.Document) {
			e := *d.Endpoint("GetItem")
			e.Name, e.Path = "GetItemV2", "/v2/items/{id}"
			d.Endpoints = append(d.Endpoints, e)
		}, map[string]apicontract.Severity{"endpoint.added@GetItemV2:": N}},
		{"path changed", func(d *apicontract.Document) { d.Endpoint("ListItems").Path = "/all" },
			map[string]apicontract.Severity{"endpoint.path_changed@ListItems:": B}},
		{"param renamed only", func(d *apicontract.Document) {
			e := d.Endpoint("GetItem")
			e.Path, e.Params[0].Name = "/items/{itemId}", "itemId"
		}, map[string]apicontract.Severity{}},
		{"auth required", func(d *apicontract.Document) { d.Endpoint("ListItems").Auth.Required = true },
			map[string]apicontract.Severity{"auth.required@ListItems:auth": B}},
		{"role removed", func(d *apicontract.Document) { d.Endpoint("CreateItem").Auth.Roles = []string{"admin"} },
			map[string]apicontract.Severity{"auth.roles_removed@CreateItem:auth.roles": B}},
		{"role added", func(d *apicontract.Document) {
			d.Endpoint("CreateItem").Auth.Roles = append(d.Endpoint("CreateItem").Auth.Roles, "owner")
		}, map[string]apicontract.Severity{"auth.roles_added@CreateItem:auth.roles": N}},
		{"required query param", func(d *apicontract.Document) { d.Endpoint("ListItems").Params[0].Required = true },
			map[string]apicontract.Severity{"param.required@ListItems:query.limit": B}},
		{"query param removed", func(d *apicontract.Document) { d.Endpoint("ListItems").Params = d.Endpoint("ListItems").Params[:1] },
			map[string]apicontract.Severity{"param.removed@ListItems:query.tag": N}},
		{"request property required", func(d *apicontract.Document) {
			b := d.Endpoint("CreateItem").Body
			b.Required = append(b.Required, "priority")
		}, map[string]apicontract.Severity{"property.required@CreateItem:body.priority": B}},
		{"request max tightened", func(d *apicontract.Document) {
			n := 60
			d.Endpoint("CreateItem").Body.Properties["title"].MaxLength = &n
		}, map[string]apicontract.Severity{"constraint.maxLength@CreateItem:body.title": B}},
		{"request max loosened", func(d *apicontract.Document) {
			n := 500
			d.Endpoint("CreateItem").Body.Properties["title"].MaxLength = &n
		}, map[string]apicontract.Severity{"constraint.maxLength@CreateItem:body.title": N}},
		{"request enum shrinks", func(d *apicontract.Document) {
			d.Endpoint("CreateItem").Body.Properties["priority"].Enum = []any{"low", "normal"}
		}, map[string]apicontract.Severity{"enum.value_removed@CreateItem:body.priority": B}},
		{"response property removed", func(d *apicontract.Document) { delete(d.Schemas["Item"].Properties, "created") },
			map[string]apicontract.Severity{
				"property.removed@CreateItem:response.created":        B,
				"property.removed@GetItem:response.created":           B,
				"property.removed@ListItems:response.items[].created": B,
			}},
		{"response enum grows", func(d *apicontract.Document) {
			p := d.Schemas["Item"].Properties["priority"]
			p.Enum = append(p.Enum, "urgent")
		}, map[string]apicontract.Severity{
			"enum.value_added@CreateItem:response.priority":        B,
			"enum.value_added@GetItem:response.priority":           B,
			"enum.value_added@ListItems:response.items[].priority": B,
		}},
		{"type changed", func(d *apicontract.Document) {
			d.Schemas["Item"].Properties["id"] = &apicontract.Schema{Type: "string"}
		},
			map[string]apicontract.Severity{
				"schema.type_changed@CreateItem:response.id":        B,
				"schema.type_changed@GetItem:response.id":           B,
				"schema.type_changed@ListItems:response.items[].id": B,
			}},
		{"error declared", func(d *apicontract.Document) {
			e := d.Endpoint("ListItems")
			e.Errors = append(e.Errors, apicontract.ErrorSpec{Status: 429, Code: "rate_limited"})
		}, map[string]apicontract.Severity{"error.added@ListItems:errors.rate_limited": N}},
		{"status changed", func(d *apicontract.Document) { d.Endpoint("CreateItem").Status = 200 },
			map[string]apicontract.Severity{"endpoint.status_changed@CreateItem:": B}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := kinds(apicontract.Diff(base, mutate(t, "9.0.0", tc.edit)))
			if len(got) != len(tc.want) {
				t.Fatalf("changes = %v, want %v", got, tc.want)
			}
			for k, sev := range tc.want {
				if got[k] != sev {
					t.Errorf("%s: got %q, want %q (all: %v)", k, got[k], sev, got)
				}
			}
		})
	}
}

func TestDiffVersionDecision(t *testing.T) {
	base := testService("1.2.0").MustCompile()
	removeList := func(d *apicontract.Document) { d.Endpoints = d.Endpoints[1:] }
	addErr := func(d *apicontract.Document) {
		e := d.Endpoint("ListItems")
		e.Errors = append(e.Errors, apicontract.ErrorSpec{Status: 429, Code: "rate_limited"})
	}
	cases := []struct {
		name, version string
		edit          func(*apicontract.Document)
		required      apicontract.Bump
		ok            bool
	}{
		{"no change same version", "1.2.0", func(*apicontract.Document) {}, apicontract.BumpNone, true},
		{"breaking needs major", "1.3.0", removeList, apicontract.BumpMajor, false},
		{"breaking with major", "2.0.0", removeList, apicontract.BumpMajor, true},
		{"additive needs minor", "1.2.1", addErr, apicontract.BumpMinor, false},
		{"additive with minor", "1.3.0", addErr, apicontract.BumpMinor, true},
		{"backwards", "1.1.0", func(*apicontract.Document) {}, apicontract.BumpNone, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := apicontract.Diff(base, mutate(t, tc.version, tc.edit))
			if r.RequiredBump != tc.required || r.VersionOK != tc.ok || (r.Err() == nil) != tc.ok {
				t.Fatalf("required=%s ok=%t decision=%q", r.RequiredBump, r.VersionOK, r.Decision)
			}
		})
	}

	zero := testService("0.4.0").MustCompile()
	r := apicontract.Diff(zero, mutate(t, "0.5.0", func(d *apicontract.Document) { d.Version = "0.5.0"; d.Endpoints = d.Endpoints[1:] }))
	if r.RequiredBump != apicontract.BumpMinor || !r.VersionOK {
		t.Fatalf("0.x breaking change should accept a minor bump: %+v", r)
	}
}

func TestReportJSONIsStable(t *testing.T) {
	base := testService("1.2.0").MustCompile()
	next := mutate(t, "2.0.0", func(d *apicontract.Document) {
		d.Endpoints = d.Endpoints[1:]
		delete(d.Schemas["Item"].Properties, "created")
		d.Endpoint("GetItem").Errors = append(d.Endpoint("GetItem").Errors, apicontract.ErrorSpec{Status: 410, Code: "gone"})
	})
	out, err := apicontract.Diff(base, next).JSON()
	if err != nil {
		t.Fatal(err)
	}
	golden(t, "items.diff.json", out)
}

func TestDiffFiles(t *testing.T) {
	dir := t.TempDir()
	write := func(name string, d *apicontract.Document) string {
		raw, err := d.JSON()
		if err != nil {
			t.Fatal(err)
		}
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, raw, 0o644); err != nil {
			t.Fatal(err)
		}
		return p
	}
	oldPath := write("old.json", testService("1.2.0").MustCompile())
	newPath := write("new.json", mutate(t, "1.3.0", func(d *apicontract.Document) { d.Endpoints = d.Endpoints[1:] }))
	r, err := apicontract.DiffFiles(oldPath, newPath)
	if err != nil || r.Breaking != 1 || r.Err() == nil {
		t.Fatalf("report=%+v err=%v", r, err)
	}
	if _, err := apicontract.DiffFiles(filepath.Join(dir, "missing.json"), newPath); err == nil {
		t.Fatal("missing baseline must fail")
	}
}
