package contracttest_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/krewire/krewire/apicontract"
	"github.com/krewire/krewire/apicontract/contracttest"
	"github.com/krewire/krewire/apicontract/examples/todo/todoapi"
)

// recorder captures failures instead of stopping the test.
type recorder struct {
	*testing.T
	fatal, errs []string
}

func (r *recorder) Helper()                      {}
func (r *recorder) Fatalf(f string, args ...any) { r.fatal = append(r.fatal, fmt.Sprintf(f, args...)) }
func (r *recorder) Errorf(f string, args ...any) { r.errs = append(r.errs, fmt.Sprintf(f, args...)) }

func withoutDelete(version string) *apicontract.Document {
	s := apicontract.NewService("todo", version)
	s.Title, s.Description, s.BasePath = "Todo API", "Example Krewire service contract.", "/api/todo"
	return s.Add(todoapi.ListTodos, todoapi.GetTodo, todoapi.CreateTodo, todoapi.UpdateTodo).MustCompile()
}

func TestCheckBaseline(t *testing.T) {
	baseline := filepath.Join(t.TempDir(), "todo.contract.json")
	raw, err := todoapi.Service().MustCompile().JSON()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(baseline, raw, 0o644); err != nil {
		t.Fatal(err)
	}

	r := &recorder{T: t}
	contracttest.CheckBaseline(r, todoapi.Service().MustCompile(), baseline)
	if len(r.fatal)+len(r.errs) != 0 {
		t.Fatalf("unchanged contract failed: %v %v", r.fatal, r.errs)
	}

	r = &recorder{T: t}
	contracttest.CheckBaseline(r, withoutDelete("1.1.0"), baseline)
	if len(r.fatal) == 0 || !strings.Contains(r.fatal[0], "require a major bump") || !strings.Contains(r.fatal[0], `"endpoint.removed"`) {
		t.Fatalf("breaking change without major bump must fail with the JSON report: %v", r.fatal)
	}

	r = &recorder{T: t}
	report := contracttest.CheckBaseline(r, withoutDelete("2.0.0"), baseline)
	if len(r.fatal) != 0 || len(r.errs) != 1 || !strings.Contains(r.errs[0], "refresh the baseline") || report.Breaking != 1 {
		t.Fatalf("bumped but unrecorded change must ask for a baseline refresh: %v %v", r.fatal, r.errs)
	}

	t.Setenv(contracttest.UpdateEnv, "1")
	contracttest.CheckBaseline(t, withoutDelete("2.0.0"), baseline)
	doc, err := apicontract.LoadDocument(baseline)
	if err != nil || doc.Version != "2.0.0" || doc.Endpoint("DeleteTodo") != nil {
		t.Fatalf("baseline not refreshed: %v", err)
	}
}

func TestGoldenDetectsStaleFiles(t *testing.T) {
	path := filepath.Join(t.TempDir(), "x.json")
	if err := os.WriteFile(path, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	r := &recorder{T: t}
	contracttest.Golden(r, path, []byte("new"))
	if len(r.errs) != 1 {
		t.Fatalf("stale golden must fail: %v", r.errs)
	}
}
