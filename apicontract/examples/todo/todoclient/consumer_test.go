package todoclient_test

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/krewire/krewire/apicontract"
	"github.com/krewire/krewire/apicontract/examples/todo/todoapi"
	"github.com/krewire/krewire/apicontract/examples/todo/todoclient"
	"github.com/krewire/krewire/apicontract/examples/todo/todoservice"
	"github.com/krewire/krewire/apicontract/gatewaybind"
	"github.com/krewire/krewire/apicontract/httpbind"
	"github.com/krewire/krewire/auth"
	"github.com/krewire/krewire/web/gateway"
)

var secret = []byte("0123456789abcdef0123456789abcdef")

func token(t *testing.T, roles ...any) string {
	t.Helper()
	c := auth.DefaultClaims("consumer", time.Hour)
	if len(roles) > 0 {
		c["roles"] = roles
	}
	tok, err := auth.SignJWT(secret, c)
	if err != nil {
		t.Fatal(err)
	}
	return tok
}

// TestConsumerThroughGateway runs the generated client against the service
// behind a gateway whose routes come from the same contract.
func TestConsumerThroughGateway(t *testing.T) {
	svc := httptest.NewServer(todoservice.Handler(todoservice.NewStore(nil), secret, httpbind.ValidateResponses()))
	defer svc.Close()
	doc := todoapi.Service().MustCompile()
	gw := gateway.New()
	gw.Reload(gatewaybind.Routes(doc, gatewaybind.Options{TargetURL: svc.URL}))
	front := httptest.NewServer(gw)
	defer front.Close()

	base := httpbind.NewClient(front.URL + doc.BasePath)
	base.ValidateResponses = true
	anon := todoclient.New(base)
	user := todoclient.New(base.WithBearer(token(t)))
	admin := todoclient.New(base.WithBearer(token(t, "admin")))
	ctx := context.Background()

	if todoclient.ContractVersion != todoapi.Version {
		t.Fatalf("generated client is for %s, contract is %s", todoclient.ContractVersion, todoapi.Version)
	}
	if _, err := anon.CreateTodo(ctx, &todoapi.CreateTodoRequest{Title: "x"}); !errors.Is(err, apicontract.ErrUnauthenticated) {
		t.Fatalf("anonymous create: %v", err)
	}
	created, err := user.CreateTodo(ctx, &todoapi.CreateTodoRequest{Title: "ship contracts"})
	if err != nil {
		t.Fatal(err)
	}
	if created.ID != 1 || created.Priority != "normal" {
		t.Fatalf("created = %+v", created)
	}
	if _, err := user.CreateTodo(ctx, &todoapi.CreateTodoRequest{}); !errors.Is(err, apicontract.ErrValidation) {
		t.Fatalf("invalid create: %v", err)
	}
	done := true
	if got, err := user.UpdateTodo(ctx, &todoapi.UpdateTodoRequest{ID: 1, Done: &done}); err != nil || !got.Done {
		t.Fatalf("update: %+v %v", got, err)
	}
	list, err := anon.ListTodos(ctx, &todoapi.ListTodosRequest{Done: &done, Limit: 5})
	if err != nil || len(list.Items) != 1 {
		t.Fatalf("list: %+v %v", list, err)
	}
	if _, err := user.DeleteTodo(ctx, &todoapi.TodoID{ID: 1}); !errors.Is(err, apicontract.ErrForbidden) {
		t.Fatalf("delete without role: %v", err)
	}
	if _, err := admin.DeleteTodo(ctx, &todoapi.TodoID{ID: 1}); err != nil {
		t.Fatalf("delete: %v", err)
	}
	_, err = anon.GetTodo(ctx, &todoapi.TodoID{ID: 1})
	var ce *apicontract.Error
	if !errors.Is(err, todoapi.ErrTodoNotFound) || !errors.As(err, &ce) || ce.Status != 404 {
		t.Fatalf("get deleted: %v", err)
	}
	if _, err := anon.GetTodo(ctx, &todoapi.TodoID{}); err == nil {
		t.Fatal("empty path parameter must fail before sending")
	}
}

func TestGatewayRoutesFromContract(t *testing.T) {
	routes := gatewaybind.Routes(todoapi.Service().MustCompile(), gatewaybind.Options{AuthMiddleware: "jwt"})
	type key struct {
		method, path string
		prefix, auth bool
	}
	want := []key{
		{"GET", "/api/todo/todos", false, false},
		{"POST", "/api/todo/todos", false, true},
		{"DELETE", "/api/todo/todos/", true, true},
		{"GET", "/api/todo/todos/", true, false},
		{"PATCH", "/api/todo/todos/", true, true},
	}
	if len(routes) != len(want) {
		t.Fatalf("routes = %+v", routes)
	}
	for i, w := range want {
		r := routes[i]
		got := key{r.Method, r.Path, r.Prefix, r.Auth}
		if got != w || r.Service != "todo" {
			t.Errorf("route %d = %+v (service %q), want %+v", i, got, r.Service, w)
		}
		if w.auth && (len(r.Middleware) != 1 || r.Middleware[0] != "jwt") {
			t.Errorf("route %d middleware = %v", i, r.Middleware)
		}
	}
}
