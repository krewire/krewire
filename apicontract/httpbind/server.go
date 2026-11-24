// Package httpbind binds typed service contracts (pkg/apicontract) to HTTP:
// Handle registers a typed handler on a pkg/web router with request decoding,
// authentication enforcement, validation, and declared error mapping; Call
// is the matching typed client. Both read the same contract definition, so
// route, method, parameter locations, and error semantics cannot drift.
package httpbind

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"reflect"

	"github.com/krewire/krewire/apicontract"
	"github.com/krewire/krewire/apicontract/internal/bind"
	"github.com/krewire/krewire/validation"
	"github.com/krewire/krewire/web"
)

// HandlerFunc implements one contract endpoint.
type HandlerFunc[Req, Resp any] func(ctx context.Context, req *Req) (*Resp, error)

// Options tunes the server binding.
type Options struct {
	// ValidateResponses validates every response struct with pkg/validation
	// before writing it, turning contract violations into 500s. Enable it in
	// development and tests.
	ValidateResponses bool
	// Logger receives undeclared-error and response-validation reports.
	Logger *slog.Logger
}

// Option configures Options.
type Option func(*Options)

// ValidateResponses enables response validation.
func ValidateResponses() Option { return func(o *Options) { o.ValidateResponses = true } }

// WithLogger sets the logger.
func WithLogger(l *slog.Logger) Option { return func(o *Options) { o.Logger = l } }

// Handle registers fn on r at the endpoint's method and path.
func Handle[Req, Resp any](r *web.Router, def apicontract.Def[Req, Resp], fn HandlerFunc[Req, Resp], opts ...Option) {
	ep := def.Endpoint()
	r.Handle(ep.Method, ep.Path, Handler(def, fn, opts...))
}

// Handler returns the web handler for one endpoint without registering it.
func Handler[Req, Resp any](def apicontract.Def[Req, Resp], fn HandlerFunc[Req, Resp], opts ...Option) web.HandlerFunc {
	o := Options{Logger: slog.Default()}
	for _, f := range opts {
		f(&o)
	}
	ep := def.Endpoint()
	declared := map[string]bool{}
	for _, e := range ep.Errors {
		declared[e.Code] = true
	}
	fields := bind.Fields(ep.Request)
	isEmpty := ep.Request == reflect.TypeFor[apicontract.Empty]()
	hasBody := !isEmpty && hasBodyFields(ep.Request)
	status := ep.SuccessStatus()

	return func(w http.ResponseWriter, hr *http.Request, params web.Params) {
		if err := authorize(hr.Context(), ep.Auth); err != nil {
			writeErr(w, err)
			return
		}
		req := new(Req)
		if !isEmpty {
			if hasBody {
				if err := web.ReadJSON(hr, req); err != nil {
					writeErr(w, apicontract.ErrBadRequest.New(err.Error()))
					return
				}
			}
			if err := bindParams(req, fields, hr, params); err != nil {
				writeErr(w, err)
				return
			}
			if err := validation.Struct(req); err != nil {
				writeErr(w, validationErr(err))
				return
			}
		}
		resp, err := fn(hr.Context(), req)
		if err != nil {
			var ce *apicontract.Error
			if errors.As(err, &ce) && !declared[ce.Code] {
				o.Logger.Warn("apicontract: undeclared error returned", "endpoint", ep.Name, "code", ce.Code)
			}
			writeErr(w, err)
			return
		}
		if ep.Response == nil || ep.Response == reflect.TypeFor[apicontract.Empty]() || resp == nil {
			w.WriteHeader(status)
			return
		}
		if o.ValidateResponses {
			if verr := validation.Struct(resp); verr != nil {
				o.Logger.Error("apicontract: response violates contract", "endpoint", ep.Name, "error", verr)
				writeErr(w, apicontract.ErrInternal.New("response violates contract"))
				return
			}
		}
		web.JSON(w, status, resp)
	}
}

// authorize enforces the endpoint's Auth against the identity installed by
// the pkg/web (pkg/auth) authentication middleware.
func authorize(ctx context.Context, a apicontract.Auth) error {
	if !a.Required && len(a.Roles) == 0 {
		return nil
	}
	id := web.IdentityFrom(ctx)
	if id == nil || !schemeMatches(a.Scheme, id.Method) {
		return apicontract.ErrUnauthenticated.New("authentication required")
	}
	if len(a.Roles) == 0 {
		return nil
	}
	for _, role := range a.Roles {
		if id.HasRole(role) {
			return nil
		}
	}
	return apicontract.ErrForbidden.New("missing required role")
}

func schemeMatches(scheme, method string) bool {
	switch scheme {
	case "bearer":
		return method == "jwt" || method == "bearer"
	case "basic":
		return method == "basic"
	}
	return true
}

func hasBodyFields(t reflect.Type) bool {
	for t != nil && t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t == nil || t.Kind() != reflect.Struct {
		return false
	}
	for i := 0; i < t.NumField(); i++ {
		sf := t.Field(i)
		if sf.Tag.Get("json") == "-" || bind.IsLocated(sf) {
			continue
		}
		if sf.IsExported() || sf.Anonymous {
			return true
		}
	}
	return false
}

func bindParams(req any, fields []bind.Field, hr *http.Request, params web.Params) error {
	rv := reflect.ValueOf(req).Elem()
	q := hr.URL.Query()
	for _, f := range fields {
		var raw []string
		switch f.In {
		case bind.InPath:
			raw = []string{params[f.Name]}
		case bind.InQuery:
			raw = q[f.Name]
		case bind.InHeader:
			raw = hr.Header.Values(f.Name)
		}
		if err := bind.Set(rv.FieldByIndex(f.Index), raw); err != nil {
			return &apicontract.Error{Status: http.StatusBadRequest, Code: apicontract.ErrBadRequest.Code,
				Message: f.In + " parameter " + f.Name + ": " + err.Error()}
		}
	}
	return nil
}

// validationErr maps a pkg/validation failure to the declared
// validation_error, carrying per-field details.
func validationErr(err error) error {
	var ve *validation.ValidationError
	if !errors.As(err, &ve) {
		return apicontract.ErrInternal.New("invalid validation rules")
	}
	type field struct {
		Field string `json:"field"`
		Rule  string `json:"rule"`
	}
	details := make([]field, 0, len(ve.Fields))
	for _, f := range ve.Fields {
		details = append(details, field{Field: f.Field, Rule: f.Rule})
	}
	e := apicontract.ErrValidation.New("request failed validation")
	e.Details = details
	return e
}

// writeErr renders err with the shared ErrorBody wire shape.
func writeErr(w http.ResponseWriter, err error) {
	var ce *apicontract.Error
	if errors.As(err, &ce) {
		if ce.Status == http.StatusUnauthorized {
			w.Header().Set("WWW-Authenticate", "Bearer")
		}
		web.JSON(w, ce.Status, apicontract.ErrorBody{Code: ce.Code, Message: ce.Message, Details: ce.Details})
		return
	}
	var he *web.HTTPError
	if errors.As(err, &he) {
		web.Error(w, he)
		return
	}
	web.JSON(w, http.StatusInternalServerError, apicontract.ErrorBody{Code: apicontract.ErrInternal.Code, Message: "internal server error"})
}
