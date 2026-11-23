package todoservice_test

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/krewire/krewire/pkg/apicontract/contracttest"
	"github.com/krewire/krewire/pkg/apicontract/examples/todo/todoapi"
	"github.com/krewire/krewire/pkg/apicontract/examples/todo/todoservice"
	"github.com/krewire/krewire/pkg/apicontract/httpbind"
	"github.com/krewire/krewire/pkg/auth"
)

var secret = []byte("0123456789abcdef0123456789abcdef")

func bearer(t *testing.T, roles ...string) http.Header {
	t.Helper()
	claims := auth.DefaultClaims("tester", time.Hour)
	if len(roles) > 0 {
		r := make([]any, len(roles))
		for i, s := range roles {
			r[i] = s
		}
		claims["roles"] = r
	}
	tok, err := auth.SignJWT(secret, claims)
	if err != nil {
		t.Fatal(err)
	}
	return http.Header{"Authorization": {"Bearer " + tok}}
}

// TestContractBaseline is the CI gate: a contract change without the
// version bump the compatibility report requires fails here, and so does any
// change whose baseline was not refreshed.
func TestContractBaseline(t *testing.T) {
	doc := todoapi.Service().MustCompile()
	contracttest.CheckBaseline(t, doc, "../testdata/todo.contract.json")
	oas, err := doc.OpenAPI()
	if err != nil {
		t.Fatal(err)
	}
	contracttest.Golden(t, "../testdata/todo.openapi.json", oas)
}

func TestConformance(t *testing.T) {
	doc := todoapi.Service().MustCompile()
	h := todoservice.Handler(todoservice.NewStore(nil), secret, httpbind.ValidateResponses())
	user, admin := bearer(t), bearer(t, "admin")
	id := map[string]string{"id": "1"}
	missing := map[string]string{"id": "99"}
	cases := []contracttest.Case{
		{Name: "create", Endpoint: "CreateTodo", Header: user, Body: map[string]any{"title": "write docs", "priority": "high"}},
		{Name: "create unauthenticated", Endpoint: "CreateTodo", Body: map[string]any{"title": "x"}, WantStatus: 401, WantCode: "unauthenticated"},
		{Name: "create invalid", Endpoint: "CreateTodo", Header: user, Body: map[string]any{"title": ""}, WantStatus: 400, WantCode: "validation_error"},
		{Name: "create bad priority", Endpoint: "CreateTodo", Header: user, Body: map[string]any{"title": "x", "priority": "urgent"}, WantStatus: 400, WantCode: "validation_error"},
		{Name: "create unknown field", Endpoint: "CreateTodo", Header: user, Body: `{"title":"x","owner":"me"}`, WantStatus: 400, WantCode: "bad_request"},
		{Name: "list", Endpoint: "ListTodos", Query: url.Values{"done": {"false"}, "limit": {"10"}}},
		{Name: "list invalid limit", Endpoint: "ListTodos", Query: url.Values{"limit": {"500"}}, WantStatus: 400, WantCode: "validation_error"},
		{Name: "list malformed bool", Endpoint: "ListTodos", Query: url.Values{"done": {"maybe"}}, WantStatus: 400, WantCode: "bad_request"},
		{Name: "get", Endpoint: "GetTodo", Path: id},
		{Name: "get missing", Endpoint: "GetTodo", Path: missing, WantStatus: 404, WantCode: "todo_not_found"},
		{Name: "get invalid id", Endpoint: "GetTodo", Path: map[string]string{"id": "0"}, WantStatus: 400, WantCode: "validation_error"},
		{Name: "update", Endpoint: "UpdateTodo", Path: id, Header: user, Body: map[string]any{"done": true}},
		{Name: "update unauthenticated", Endpoint: "UpdateTodo", Path: id, Body: map[string]any{"done": true}, WantStatus: 401, WantCode: "unauthenticated"},
		{Name: "update invalid", Endpoint: "UpdateTodo", Path: id, Header: user, Body: map[string]any{"title": strings.Repeat("x", 201)}, WantStatus: 400, WantCode: "validation_error"},
		{Name: "update missing", Endpoint: "UpdateTodo", Path: missing, Header: user, Body: map[string]any{"done": true}, WantStatus: 404, WantCode: "todo_not_found"},
		{Name: "delete forbidden", Endpoint: "DeleteTodo", Path: id, Header: user, WantStatus: 403, WantCode: "forbidden"},
		{Name: "delete unauthenticated", Endpoint: "DeleteTodo", Path: id, WantStatus: 401, WantCode: "unauthenticated"},
		{Name: "delete", Endpoint: "DeleteTodo", Path: id, Header: admin},
		{Name: "delete missing", Endpoint: "DeleteTodo", Path: id, Header: admin, WantStatus: 404, WantCode: "todo_not_found"},
	}
	contracttest.Run(t, h, doc, cases)
	contracttest.RequireCoverage(t, doc, cases)
}

func TestCoverageReportsGaps(t *testing.T) {
	doc := todoapi.Service().MustCompile()
	missing := contracttest.Uncovered(doc, []contracttest.Case{{Endpoint: "ListTodos"}})
	want := []string{
		"CreateTodo: authentication failure",
		"DeleteTodo: forbidden role",
		"GetTodo: declared error todo_not_found",
		"UpdateTodo: validation failure",
	}
	got := strings.Join(missing, "\n")
	for _, w := range want {
		if !strings.Contains(got, w) {
			t.Errorf("Uncovered missing %q:\n%s", w, got)
		}
	}
	if strings.Contains(got, "ListTodos: success") {
		t.Error("ListTodos success is covered")
	}
}
