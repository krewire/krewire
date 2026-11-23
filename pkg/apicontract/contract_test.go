package apicontract_test

import (
	"errors"
	"flag"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/krewire/krewire/pkg/apicontract"
)

var update = flag.Bool("update", false, "rewrite golden files")

type Label struct {
	Name  string `json:"name" validate:"required,max=20"`
	Color string `json:"color,omitempty" validate:"omitempty,pattern=^#[0-9a-f]{6}$"`
}

type Item struct {
	ID       int64             `json:"id"`
	Title    string            `json:"title" validate:"required,min=1,max=120" doc:"Short title."`
	Priority string            `json:"priority" validate:"oneof=low normal high"`
	Labels   []Label           `json:"labels,omitempty" validate:"max=5"`
	Meta     map[string]string `json:"meta,omitempty"`
	Parent   *Item             `json:"parent,omitempty" validate:"omitempty"`
	Created  time.Time         `json:"created"`
}

type CreateItem struct {
	Title    string  `json:"title" validate:"required,min=1,max=120"`
	Priority string  `json:"priority,omitempty" validate:"omitempty,oneof=low normal high"`
	Labels   []Label `json:"labels,omitempty"`
	Trace    string  `json:"-" header:"X-Trace-Id"`
}

type GetItem struct {
	ID int64 `json:"-" path:"id" validate:"min=1"`
}

type ListItems struct {
	Limit int      `json:"-" query:"limit" validate:"omitempty,min=1,max=100"`
	Tags  []string `json:"-" query:"tag"`
}

type ItemPage struct {
	Items []Item `json:"items"`
	Next  string `json:"next,omitempty"`
}

var ErrItemNotFound = apicontract.ErrorSpec{Status: 404, Code: "item_not_found", Description: "No item has that id."}

func testService(version string) *apicontract.Service {
	s := apicontract.NewService("items", version)
	s.Title = "Items API"
	s.BasePath = "/api/items"
	return s.Add(
		apicontract.Define[ListItems, ItemPage]("ListItems", "GET", "/items", apicontract.Summary("List items"), apicontract.Tags("items")),
		apicontract.Define[GetItem, Item]("GetItem", "GET", "/items/{id}", apicontract.Errors(ErrItemNotFound)),
		apicontract.Define[CreateItem, Item]("CreateItem", "POST", "/items",
			apicontract.Status(http.StatusCreated), apicontract.RequireAuth("bearer", "editor", "admin")),
		apicontract.Define[GetItem, apicontract.Empty]("DeleteItem", "DELETE", "/items/{id}",
			apicontract.RequireAuth(""), apicontract.Errors(ErrItemNotFound)),
	)
}

func golden(t *testing.T, name string, got []byte) {
	t.Helper()
	path := filepath.Join("testdata", name)
	if *update {
		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden %s (run go test -update): %v", path, err)
	}
	if string(want) != string(got) {
		t.Errorf("%s differs from golden; run go test ./pkg/apicontract -update and review the diff", name)
	}
}

func TestCompileGolden(t *testing.T) {
	doc, err := testService("1.2.0").Compile()
	if err != nil {
		t.Fatal(err)
	}
	canon, err := doc.JSON()
	if err != nil {
		t.Fatal(err)
	}
	golden(t, "items.contract.json", canon)
	oas, err := doc.OpenAPI()
	if err != nil {
		t.Fatal(err)
	}
	golden(t, "items.openapi.json", oas)

	again, _ := testService("1.2.0").MustCompile().JSON()
	if string(again) != string(canon) {
		t.Fatal("compilation is not deterministic")
	}
	round, err := apicontract.ReadDocument(strings.NewReader(string(canon)))
	if err != nil {
		t.Fatalf("canonical document does not round-trip: %v", err)
	}
	if r := apicontract.Diff(doc, round); len(r.Changes) != 0 {
		t.Fatalf("round-tripped document differs: %+v", r.Changes)
	}
}

func TestCompileDerivesSchemas(t *testing.T) {
	doc := testService("1.0.0").MustCompile()
	create := doc.Endpoint("CreateItem")
	if create.Status != 201 || !create.Auth.Required || len(create.Auth.Roles) != 2 {
		t.Fatalf("create metadata: %+v", create)
	}
	if _, ok := create.Body.Properties["Trace"]; ok {
		t.Fatal("header field leaked into the body schema")
	}
	if len(create.Params) != 1 || create.Params[0].In != "header" || create.Params[0].Name != "X-Trace-Id" {
		t.Fatalf("create params: %+v", create.Params)
	}
	codes := map[string]int{}
	for _, e := range create.Errors {
		codes[e.Code] = e.Status
	}
	for code, status := range map[string]int{"bad_request": 400, "validation_error": 400, "unauthenticated": 401, "forbidden": 403, "internal_error": 500} {
		if codes[code] != status {
			t.Errorf("CreateItem missing implicit error %s/%d: %v", code, status, codes)
		}
	}
	del := doc.Endpoint("DeleteItem")
	if del.Status != 204 || del.Response != nil {
		t.Fatalf("delete should be 204 without body: %+v", del)
	}
	item := doc.Schemas["Item"]
	if title := item.Properties["title"]; *title.MinLength != 1 || *title.MaxLength != 120 || title.Description != "Short title." {
		t.Fatalf("title constraints: %+v", title)
	}
	if p := item.Properties["priority"]; len(p.Enum) != 3 {
		t.Fatalf("priority enum: %+v", p)
	}
	if p := item.Properties["parent"]; p.Ref != apicontract.RefPrefix+"Item" {
		t.Fatalf("recursive ref: %+v", p)
	}
	if c := item.Properties["created"]; c.Format != "date-time" {
		t.Fatalf("time format: %+v", c)
	}
	list := doc.Endpoint("ListItems")
	if list.Body != nil || len(list.Params) != 2 || list.Params[1].Schema.Type != "array" {
		t.Fatalf("list params: %+v", list.Params)
	}
}

func TestValidateRejectsBrokenContracts(t *testing.T) {
	type BadPath struct {
		ID string `path:"other"`
	}
	type Body struct {
		X string `json:"x"`
	}
	cases := map[string]struct {
		svc  *apicontract.Service
		want string
	}{
		"version":     {apicontract.NewService("svc", "one"), "version"},
		"name":        {apicontract.NewService("Svc", "1.0.0"), "service"},
		"dup name":    {apicontract.NewService("svc", "1.0.0").Add(apicontract.Define[apicontract.Empty, apicontract.Empty]("A", "GET", "/a"), apicontract.Define[apicontract.Empty, apicontract.Empty]("A", "GET", "/b")), "duplicate endpoint name"},
		"dup route":   {apicontract.NewService("svc", "1.0.0").Add(apicontract.Define[GetItem, apicontract.Empty]("A", "GET", "/a/{id}"), apicontract.Define[GetItem, apicontract.Empty]("B", "GET", "/a/{id}")), "already used"},
		"method":      {apicontract.NewService("svc", "1.0.0").Add(apicontract.Define[apicontract.Empty, apicontract.Empty]("A", "FETCH", "/a")), "unsupported method"},
		"path param":  {apicontract.NewService("svc", "1.0.0").Add(apicontract.Define[BadPath, apicontract.Empty]("A", "GET", "/a/{id}")), "has no request field"},
		"get body":    {apicontract.NewService("svc", "1.0.0").Add(apicontract.Define[Body, apicontract.Empty]("A", "GET", "/a")), "cannot declare a request body"},
		"status":      {apicontract.NewService("svc", "1.0.0").Add(apicontract.Define[apicontract.Empty, apicontract.Empty]("A", "GET", "/a", apicontract.Status(302))), "not 2xx"},
		"error code":  {apicontract.NewService("svc", "1.0.0").Add(apicontract.Define[apicontract.Empty, apicontract.Empty]("A", "GET", "/a", apicontract.Errors(apicontract.ErrorSpec{Status: 200, Code: "x"}))), "not 4xx/5xx"},
		"auth scheme": {apicontract.NewService("svc", "1.0.0").Add(apicontract.Define[apicontract.Empty, apicontract.Empty]("A", "GET", "/a", apicontract.RequireAuth("digest"))), "unsupported scheme"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := tc.svc.Compile()
			var inv *apicontract.InvalidError
			if !errors.As(err, &inv) || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("want InvalidError containing %q, got %v", tc.want, err)
			}
		})
	}
}

func TestErrorMatching(t *testing.T) {
	err := error(ErrItemNotFound.New("item 7"))
	if !errors.Is(err, ErrItemNotFound) || errors.Is(err, apicontract.ErrForbidden) {
		t.Fatal("errors.Is must match on status and code")
	}
	if !errors.Is(err, &apicontract.Error{Status: 404, Code: "item_not_found"}) {
		t.Fatal("errors.Is must match *Error targets")
	}
	if got := err.Error(); got != "item_not_found (404): item 7" {
		t.Fatalf("message: %q", got)
	}
}
