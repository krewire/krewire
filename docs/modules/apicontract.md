# Typed Service Contracts (`apicontract`)

`apicontract` gives the `service` and `gateway` workloads one canonical,
typed, versioned contract per service (ref #137, benchmark: Encore). The
contract is declared in Go, and every other artifact derives from it:

```text
Go Def[Req, Resp] declarations ──Compile──▶ canonical Document (JSON)
                                              │
     ┌──────────────┬───────────────┬─────────┼─────────────┬──────────────────┐
     ▼              ▼               ▼         ▼             ▼                  ▼
 OpenAPI 3.1   JSON Schemas +   httpbind     clientgen    gatewaybind      Diff / Report
 (golden)      ValidateJSON     server+client (Go client)  (gateway routes) (CI + AI context)
```

Nothing in the pipeline calls an LLM. Agent-authored contract edits go
through the same compiler, validators, and compatibility gate as human edits.

| Package | Role | Imports |
|---|---|---|
| `apicontract` | Model, compiler, structural validation, JSON Schema, OpenAPI 3.1, JSON value validation, compatibility diff | stdlib, `validation`, `version` |
| `apicontract/httpbind` | Typed server binding on `web`, generic typed client | + `web` |
| `apicontract/clientgen` | Generated, documented Go client per service | stdlib |
| `apicontract/gatewaybind` | `web/gateway` routes from the contract | + `web/gateway` |
| `apicontract/contracttest` | Baseline, golden, and conformance gates for `go test` | stdlib |
| `apicontract/examples/todo` | Example contract (`todoapi`), service (`todoservice`), generated consumer (`todoclient`) | — |

The core package does not import `web`, so tools (for example the `krew`
CLI) can load, validate, and diff documents without linking the HTTP stack.

## 1. Declaring a contract

```go
package todoapi

type CreateTodoRequest struct {
    Title    string `json:"title" validate:"required,min=1,max=200"`
    Priority string `json:"priority,omitempty" validate:"omitempty,oneof=low normal high"`
}

type TodoID struct {
    ID int64 `json:"-" path:"id" validate:"min=1"`
}

var ErrTodoNotFound = apicontract.ErrorSpec{Status: 404, Code: "todo_not_found"}

var (
    CreateTodo = apicontract.Define[CreateTodoRequest, Todo]("CreateTodo", "POST", "/todos",
        apicontract.Status(201), apicontract.RequireAuth("bearer"))
    GetTodo = apicontract.Define[TodoID, Todo]("GetTodo", "GET", "/todos/{id}",
        apicontract.Errors(ErrTodoNotFound))
    DeleteTodo = apicontract.Define[TodoID, apicontract.Empty]("DeleteTodo", "DELETE", "/todos/{id}",
        apicontract.RequireAuth("bearer", "admin"), apicontract.Errors(ErrTodoNotFound))
)

func Service() *apicontract.Service {
    s := apicontract.NewService("todo", "1.0.0")
    s.BasePath = "/api/todo"
    return s.Add(CreateTodo, GetTodo, DeleteTodo)
}
```

- **Name** is the stable operation id; the diff matches endpoints across
  versions by name. Name each `Def` variable after its endpoint (required by
  `clientgen`).
- **Options**: `Summary`, `Description`, `Tags`, `Status`, `RequireAuth(scheme,
  roles...)`, `Errors(specs...)`, `Deprecated`.
- **Auth**: `scheme` is `bearer` (auth JWT identity), `basic`, or `""`
  (any). Roles are *any-of*. Missing identity → `401 unauthenticated`; no
  matching role → `403 forbidden`.
- **Status**: defaults to 200, or 204 for an `Empty` response.
- **Field locations**: `path:"id"`, `query:"limit"`, `header:"X-Trace"`. Tag
  located fields `json:"-"`; untagged exported fields form the JSON body.
- **Validation**: the existing `validate` tags (`validation`) are the only
  rule source. They run at runtime and map into the schema:

| `validate` rule | JSON Schema |
|---|---|
| `required` (or a non-`omitempty` struct pointer) | listed in `required` |
| `min` / `max` / `len` | `minLength`/`maxLength` (string), `minItems`/`maxItems` (slice), `minimum`/`maximum` (number) |
| `oneof=a b` | `enum` |
| `email` | `format: email` |
| `pattern=re` | `pattern` |

Go type mapping: integers → `integer` (`int32`/`int64` format, unsigned with
`minimum: 0`), floats → `number`, `time.Time` → `string`/`date-time`,
`[]byte` → `string`/`byte`, maps → `additionalProperties`, named structs →
`#/components/schemas/<Name>` (recursive types supported), `doc:"..."` →
`description`. Interfaces and `json.RawMessage` accept any JSON.

### Built-in errors

The compiler adds every error the runtime itself can produce, so documents
never omit an observable failure: `bad_request` and `validation_error` (400)
when the endpoint has a request, `unauthenticated` (401) when auth is
required, `forbidden` (403) when roles are set, and `internal_error` (500)
always. Every error response uses one wire shape, `ErrorBody`
(`{"code","message","details"}`), identical to `web.Error`.

## 2. Compiling and validating

```go
doc, err := todoapi.Service().Compile()   // *apicontract.Document, *InvalidError
raw, _ := doc.JSON()                      // canonical, deterministic bytes
oas, _ := doc.OpenAPI()                   // OpenAPI 3.1 JSON
err = doc.ValidateJSON(doc.Endpoint("GetTodo").Response, body) // *SchemaError
```

`Document.Validate` (run by `Compile`, `ReadDocument`, `LoadDocument`) is the
pre-build/pre-release gate. It reports every issue at once: format version,
service name, SemVer version, identifier names, duplicate names and routes
(`/a/{x}` collides with `/a/{y}`), methods, 2xx success status, bodies on
GET/HEAD/DELETE/OPTIONS, path parameters bound one-to-one to `path:` fields,
scalar parameters, 4xx/5xx error statuses with unique codes, auth schemes,
and unresolved `$ref`s.

OpenAPI output carries Krewire metadata in extensions: `x-krewire-contract`,
`x-krewire-service`, `x-krewire-error-schema`, `x-krewire-roles`, and
`x-krewire-error-codes` per response status.

## 3. Serving and calling

```go
r := web.NewRouter()
r.Use(web.Middleware(auth.JWTAuth(secret, func(o *auth.JWTOptions) { o.ContinueOnMissing = true })))
g := r.Group(todoapi.Service().BasePath)
httpbind.Handle(g, todoapi.CreateTodo, store.Create, httpbind.ValidateResponses())

func (s *Store) Create(ctx context.Context, req *todoapi.CreateTodoRequest) (*todoapi.Todo, error) {
    ...
    return nil, todoapi.ErrTodoNotFound.New("no todo with that id")
}
```

The binding enforces auth, decodes the body strictly (unknown fields are
rejected), binds path/query/header fields, validates, calls the handler, and
maps errors: `*apicontract.Error` keeps its status and code (undeclared codes
are logged), `*web.HTTPError` passes through, anything else becomes a generic
500 without leaking details. `ValidateResponses` turns invalid responses into
500s for development and tests.

Clients use the same definition:

```go
c := httpbind.NewClient("https://api.example.com/api/todo").WithBearer(token)
todo, err := httpbind.Call(ctx, c, todoapi.CreateTodo, &todoapi.CreateTodoRequest{Title: "x"})
if errors.Is(err, todoapi.ErrTodoNotFound) { ... }   // declared error semantics
```

`clientgen.Generate` renders a named method per endpoint on top of `Call`
(see `examples/todo/todoclient/todoclient.go`), including the declared errors
in each method's doc comment and a `ContractVersion` constant. The generated
file is kept current by a golden test.

## 4. Gateway routing

```go
gw.Reload(gatewaybind.Routes(doc, gatewaybind.Options{AuthMiddleware: "jwt"}))
```

Routes come from the contract's `BasePath + Path`; the gateway never restates
paths. Because `web/gateway` matches literal paths or prefixes, a path
with parameters (`/todos/{id}`) becomes a method-scoped prefix route on its
literal head (`/todos/`) and the service router performs the full match.
Authenticated endpoints set `Route.Auth` and append `AuthMiddleware`.

## 5. Compatibility and versioning

`apicontract.Diff(baseline, candidate)` (or `DiffFiles`) returns a `Report`
with classified changes and a versioning decision.

| Change | Request side (input) | Response side (output) |
|---|---|---|
| Endpoint removed / method or path changed / success status changed | breaking | — |
| Endpoint added, deprecated | non-breaking | — |
| Auth became required, roles restricted or an allowed role removed, scheme changed | breaking | — |
| Auth relaxed, allowed role added | non-breaking | — |
| Property removed | breaking (strict decoding) | breaking |
| Property added | breaking if required, else non-breaking | non-breaking |
| Property became required / optional | breaking / non-breaking | non-breaking / breaking |
| Type or format changed | breaking | breaking |
| Enum value removed / added | breaking / non-breaking | non-breaking / breaking |
| Bound tightened / loosened (`min*`, `max*`, `pattern`) | breaking / non-breaking | not compared |
| Query/header parameter added | breaking if required | — |
| Query/header parameter removed | non-breaking (ignored) | — |
| Request body added / removed | breaking if it has required fields / breaking | non-breaking / breaking |
| Error declared / undeclared | non-breaking | — |
| Error status changed for a code | breaking | — |

Path parameter renames (`{id}` → `{todoId}`) are not changes. Schema
comparison follows `$ref`s on both sides and is cycle-safe.

**Version decision.** Breaking changes require a major bump (a minor bump
while the major version is 0, per SemVer §4); any other change requires a
minor bump; no change requires none. A version that goes backwards always
fails. `Report.VersionOK`/`Report.Err()` expose the result.

Report JSON (stable ordering, suitable for CI logs and agent context):

```json
{
  "service": "items",
  "oldVersion": "1.2.0",
  "newVersion": "2.0.0",
  "breaking": 3,
  "nonBreaking": 1,
  "changes": [
    {"severity": "breaking", "kind": "property.removed", "endpoint": "GetItem",
     "location": "response.created", "message": "response property removed"}
  ],
  "requiredBump": "major",
  "actualBump": "major",
  "versionOk": true,
  "decision": "major bump satisfies required major bump"
}
```

## 6. Contract tests (the CI gate)

```go
func TestContractBaseline(t *testing.T) {
    doc := todoapi.Service().MustCompile()
    contracttest.CheckBaseline(t, doc, "testdata/todo.contract.json")
    oas, _ := doc.OpenAPI()
    contracttest.Golden(t, "testdata/todo.openapi.json", oas)
}

func TestConformance(t *testing.T) {
    cases := []contracttest.Case{
        {Name: "create", Endpoint: "CreateTodo", Header: user, Body: map[string]any{"title": "x"}},
        {Name: "create anonymous", Endpoint: "CreateTodo", Body: map[string]any{"title": "x"}, WantStatus: 401, WantCode: "unauthenticated"},
        // ...
    }
    contracttest.Run(t, handler, doc, cases)
    contracttest.RequireCoverage(t, doc, cases)
}
```

- `CheckBaseline` fails when the change needs a larger bump than declared
  (printing the JSON report), and when the contract changed without the
  committed baseline being refreshed. The baseline file therefore records
  every contract version decision in review. Refresh with
  `KREWIRE_CONTRACT_UPDATE=1 go test ./...`.
- `Run` sends each case to the handler and checks the status, the error code,
  that the body validates against the declared schema, and that every error
  is declared by the endpoint.
- `RequireCoverage` demands a success case per endpoint, a validation failure
  per endpoint with a body, an authentication failure per authenticated
  endpoint, a forbidden case per role-restricted endpoint, and a case per
  declared domain error.

Because these are ordinary Go tests, `go test ./...` in pull-request CI
detects breaking changes and requires the explicit versioning decision.

## 7. `krew` CLI wiring

The CLI lives in the `krew` repository and is not changed here. The library
is shaped so `krew` can expose the lifecycle as subcommands without loading
service Go types:

| Command | Library call | Output / exit code |
|---|---|---|
| `krew contract validate <doc.json>` | `apicontract.LoadDocument` | issues from `*InvalidError` as JSON; exit 1 on issues |
| `krew contract openapi <doc.json>` | `(*Document).OpenAPI` | OpenAPI 3.1 on stdout |
| `krew contract diff <base.json> <new.json> [--json]` | `apicontract.DiffFiles` + `(*Report).JSON` | report; exit 1 when `!VersionOK` |
| `krew contract check` | runs `go test` on packages using `contracttest` | test result |
| `krew contract gen client --pkg <name> --defs <import>` | `clientgen.Generate` via a generated `go run` stub | Go source |

Baselines are the committed canonical documents (`testdata/*.contract.json`),
so `krew contract diff origin/main:<path> <path>` needs only JSON. AI-native
workflows consume the same `Report` JSON as structured context: plans and
review reports cite `service`, `newVersion`, and the change list rather than
re-describing the API.

## 8. Design decisions and limits

- **Transport-neutral model, HTTP/JSON first.** `Document` describes
  operations, types, auth, and errors; only `httpbind`/`gatewaybind` are
  HTTP-specific. gRPC/protobuf is a separate extension (a future
  `grpcbind`/proto generator from the same document), not part of this
  release.
- **Reflection over code generation for the core path.** Generic `Def[Req,
  Resp]` carries types to both server and client at compile time;
  `clientgen` is optional sugar. No dependencies beyond the standard library.
- **Nullability** is not modelled: validators accept JSON `null` wherever a
  value may be absent (Go encodes nil pointers, slices, and maps as `null`).
- **Unknown request fields** are rejected at runtime (strict decoding), but
  schemas do not emit `additionalProperties: false`.
- Response-side bound changes are not classified; only response types,
  formats, properties, required-ness, and enums are.
- Generic Go types are supported in schemas (component names are sanitised,
  e.g. `Page_int`) but not by `clientgen`; declare a named type.
