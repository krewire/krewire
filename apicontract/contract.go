// Package apicontract defines typed, versioned service contracts for the
// service and gateway workloads (ref #137).
//
// A contract is declared in Go: each endpoint names its request and response
// types through Define, together with its route, method, success status,
// authentication requirement, and declared errors. Compile turns a Service
// into a canonical, JSON-serialisable Document. Every other artifact derives
// from that one document: the OpenAPI 3.1 description, the JSON Schemas, the
// runtime validators, the compatibility diff, the generated Go client, and the
// gateway route table. Nothing here needs an LLM; AI-assisted edits pass the
// same compiler as human ones.
//
// The package depends only on the standard library and the foundation
// packages pkg/validation and pkg/version. HTTP bindings live in
// pkg/apicontract/httpbind and gateway routing in pkg/apicontract/gatewaybind.
package apicontract

import (
	"fmt"
	"net/http"
	"reflect"
)

// FormatVersion identifies the canonical document format. It changes only
// when the document structure itself changes incompatibly.
const FormatVersion = "krewire.contract/v1"

// Empty marks an endpoint without a request or response body.
type Empty struct{}

// Auth states an endpoint's authentication requirement.
type Auth struct {
	// Required rejects anonymous callers with 401 unauthenticated.
	Required bool `json:"required"`
	// Scheme is "bearer", "basic", or empty when any configured scheme works.
	Scheme string `json:"scheme,omitempty"`
	// Roles, when set, admit only callers holding at least one of them;
	// others receive 403 forbidden. Roles imply Required.
	Roles []string `json:"roles,omitempty"`
}

// ErrorSpec declares an error an endpoint may return. It implements error so
// errors.Is(err, spec) matches any *Error carrying the same status and code.
type ErrorSpec struct {
	Status      int    `json:"status"`
	Code        string `json:"code"`
	Description string `json:"description,omitempty"`
}

// Error implements error.
func (s ErrorSpec) Error() string { return s.Code }

// New returns a runtime error for this declared spec.
func (s ErrorSpec) New(message string) *Error {
	return &Error{Status: s.Status, Code: s.Code, Message: message}
}

// Built-in errors. The compiler adds them to endpoints automatically wherever
// the runtime can produce them, so documents never omit an error a caller can
// observe.
var (
	ErrBadRequest      = ErrorSpec{Status: http.StatusBadRequest, Code: "bad_request", Description: "The request could not be decoded."}
	ErrValidation      = ErrorSpec{Status: http.StatusBadRequest, Code: "validation_error", Description: "The request failed validation."}
	ErrUnauthenticated = ErrorSpec{Status: http.StatusUnauthorized, Code: "unauthenticated", Description: "Authentication is required."}
	ErrForbidden       = ErrorSpec{Status: http.StatusForbidden, Code: "forbidden", Description: "The caller lacks a required role."}
	ErrInternal        = ErrorSpec{Status: http.StatusInternalServerError, Code: "internal_error", Description: "An unexpected server error."}
)

// ErrorBody is the wire shape of every error response. It matches the body
// written by pkg/web.Error, so contract errors and plain web errors agree.
type ErrorBody struct {
	Code    string `json:"code" validate:"required"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

// Error is a runtime contract error, returned by handlers and decoded by
// clients.
type Error struct {
	Status  int
	Code    string
	Message string
	Details any
}

// Error implements error.
func (e *Error) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("%s (%d): %s", e.Code, e.Status, e.Message)
	}
	return fmt.Sprintf("%s (%d)", e.Code, e.Status)
}

// Is matches an ErrorSpec or *Error with the same status and code.
func (e *Error) Is(target error) bool {
	switch t := target.(type) {
	case ErrorSpec:
		return e.Status == t.Status && e.Code == t.Code
	case *Error:
		return e.Status == t.Status && e.Code == t.Code
	}
	return false
}

// Endpoint is the untyped description of one operation.
type Endpoint struct {
	// Name is the stable operation identifier (OpenAPI operationId). The
	// compatibility checker matches endpoints across versions by Name.
	Name        string
	Method      string
	Path        string
	Summary     string
	Description string
	Tags        []string
	// Status is the success status; 0 means 200, or 204 for an Empty response.
	Status     int
	Auth       Auth
	Errors     []ErrorSpec
	Deprecated bool
	Request    reflect.Type
	Response   reflect.Type
}

// Option configures an endpoint.
type Option func(*Endpoint)

// Summary sets the one-line summary.
func Summary(s string) Option { return func(e *Endpoint) { e.Summary = s } }

// Description sets the long description.
func Description(s string) Option { return func(e *Endpoint) { e.Description = s } }

// Tags groups the endpoint in generated documentation.
func Tags(tags ...string) Option { return func(e *Endpoint) { e.Tags = append(e.Tags, tags...) } }

// Status overrides the success status code.
func Status(code int) Option { return func(e *Endpoint) { e.Status = code } }

// RequireAuth requires an authenticated caller using scheme ("bearer",
// "basic", or "" for any) and, when roles are given, one of those roles.
func RequireAuth(scheme string, roles ...string) Option {
	return func(e *Endpoint) {
		e.Auth = Auth{Required: true, Scheme: scheme, Roles: append([]string(nil), roles...)}
	}
}

// Errors declares the domain errors the endpoint may return.
func Errors(specs ...ErrorSpec) Option {
	return func(e *Endpoint) { e.Errors = append(e.Errors, specs...) }
}

// Deprecated marks the endpoint deprecated.
func Deprecated() Option { return func(e *Endpoint) { e.Deprecated = true } }

// Definer is implemented by Def; Service.Add accepts any Definer.
type Definer interface {
	Endpoint() Endpoint
}

// Def is a typed endpoint definition. Its type parameters carry the request
// and response shapes into server bindings and clients at compile time.
type Def[Req, Resp any] struct {
	ep Endpoint
}

// Define declares a typed endpoint.
func Define[Req, Resp any](name, method, path string, opts ...Option) Def[Req, Resp] {
	ep := Endpoint{
		Name:     name,
		Method:   method,
		Path:     path,
		Request:  reflect.TypeFor[Req](),
		Response: reflect.TypeFor[Resp](),
	}
	for _, o := range opts {
		o(&ep)
	}
	return Def[Req, Resp]{ep: ep}
}

// Endpoint returns a copy of the untyped description.
func (d Def[Req, Resp]) Endpoint() Endpoint {
	ep := d.ep
	ep.Tags = append([]string(nil), d.ep.Tags...)
	ep.Errors = append([]ErrorSpec(nil), d.ep.Errors...)
	ep.Auth.Roles = append([]string(nil), d.ep.Auth.Roles...)
	return ep
}

// SuccessStatus resolves the effective success status.
func (e Endpoint) SuccessStatus() int {
	if e.Status != 0 {
		return e.Status
	}
	if e.Response == nil || e.Response == reflect.TypeFor[Empty]() {
		return http.StatusNoContent
	}
	return http.StatusOK
}

// Service groups the endpoints of one versioned contract.
type Service struct {
	Name        string
	Version     string
	Title       string
	Description string
	// BasePath prefixes every endpoint path when the service is mounted
	// behind a gateway (e.g. "/api/todo"). Endpoint paths stay relative.
	BasePath string
	// Errors are declared on every endpoint of the service.
	Errors    []ErrorSpec
	endpoints []Endpoint
}

// NewService starts a contract with a name and a semantic version.
func NewService(name, version string) *Service {
	return &Service{Name: name, Version: version}
}

// Add registers endpoint definitions in order.
func (s *Service) Add(defs ...Definer) *Service {
	for _, d := range defs {
		s.endpoints = append(s.endpoints, d.Endpoint())
	}
	return s
}

// Endpoints returns copies of the registered endpoints.
func (s *Service) Endpoints() []Endpoint {
	return append([]Endpoint(nil), s.endpoints...)
}
