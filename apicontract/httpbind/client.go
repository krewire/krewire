package httpbind

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"

	"github.com/krewire/krewire/apicontract"
	"github.com/krewire/krewire/apicontract/internal/bind"
	"github.com/krewire/krewire/validation"
)

// maxResponseBytes bounds decoded response bodies.
const maxResponseBytes = 8 << 20

// Client is a typed contract client. It is safe for concurrent use.
type Client struct {
	// BaseURL is the service root, including any gateway base path
	// (e.g. "https://api.example.com/api/todo").
	BaseURL string
	// HTTP performs requests; nil means http.DefaultClient.
	HTTP *http.Client
	// Prepare, when set, decorates each outgoing request (credentials,
	// tracing headers).
	Prepare func(*http.Request)
	// ValidateResponses validates decoded responses with pkg/validation.
	ValidateResponses bool
}

// NewClient returns a client for baseURL.
func NewClient(baseURL string) *Client {
	return &Client{BaseURL: strings.TrimSuffix(baseURL, "/")}
}

// WithBearer returns a copy of c that sends token as a bearer credential.
func (c *Client) WithBearer(token string) *Client {
	cp := *c
	prev := c.Prepare
	cp.Prepare = func(r *http.Request) {
		if prev != nil {
			prev(r)
		}
		r.Header.Set("Authorization", "Bearer "+token)
	}
	return &cp
}

// Call invokes the endpoint described by def. A non-success response returns
// an *apicontract.Error, so callers match declared errors with errors.Is(err,
// spec). Transport failures are returned unchanged.
func Call[Req, Resp any](ctx context.Context, c *Client, def apicontract.Def[Req, Resp], req *Req) (*Resp, error) {
	ep := def.Endpoint()
	if req == nil {
		req = new(Req)
	}
	hr, err := newRequest(ctx, c, ep, reflect.ValueOf(req).Elem())
	if err != nil {
		return nil, err
	}
	hc := c.HTTP
	if hc == nil {
		hc = http.DefaultClient
	}
	res, err := hc.Do(hr)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(io.LimitReader(res.Body, maxResponseBytes))
	if err != nil {
		return nil, err
	}
	if res.StatusCode != ep.SuccessStatus() {
		return nil, decodeError(res.StatusCode, body)
	}
	out := new(Resp)
	if ep.Response == reflect.TypeFor[apicontract.Empty]() || len(bytes.TrimSpace(body)) == 0 {
		return out, nil
	}
	if err := json.Unmarshal(body, out); err != nil {
		return nil, fmt.Errorf("httpbind: %s: decode response: %w", ep.Name, err)
	}
	if c.ValidateResponses {
		if err := validation.Struct(out); err != nil {
			return nil, fmt.Errorf("httpbind: %s: response violates contract: %w", ep.Name, err)
		}
	}
	return out, nil
}

func newRequest(ctx context.Context, c *Client, ep apicontract.Endpoint, rv reflect.Value) (*http.Request, error) {
	path := ep.Path
	q := url.Values{}
	header := http.Header{}
	isEmpty := ep.Request == reflect.TypeFor[apicontract.Empty]()
	if !isEmpty {
		for _, f := range bind.Fields(ep.Request) {
			vals := bind.Format(rv.FieldByIndex(f.Index))
			switch f.In {
			case bind.InPath:
				if len(vals) == 0 || vals[0] == "" {
					return nil, fmt.Errorf("httpbind: %s: path parameter %q is empty", ep.Name, f.Name)
				}
				path = strings.Replace(path, "{"+f.Name+"}", url.PathEscape(vals[0]), 1)
			case bind.InQuery:
				for _, v := range vals {
					q.Add(f.Name, v)
				}
			case bind.InHeader:
				for _, v := range vals {
					header.Add(f.Name, v)
				}
			}
		}
	}
	u := c.BaseURL + path
	if len(q) > 0 {
		u += "?" + q.Encode()
	}
	var body io.Reader
	if !isEmpty && hasBodyFields(ep.Request) {
		raw, err := json.Marshal(rv.Interface())
		if err != nil {
			return nil, fmt.Errorf("httpbind: %s: encode request: %w", ep.Name, err)
		}
		body = bytes.NewReader(raw)
		header.Set("Content-Type", "application/json")
	}
	hr, err := http.NewRequestWithContext(ctx, ep.Method, u, body)
	if err != nil {
		return nil, err
	}
	for k, vs := range header {
		hr.Header[k] = vs
	}
	hr.Header.Set("Accept", "application/json")
	if c.Prepare != nil {
		c.Prepare(hr)
	}
	return hr, nil
}

func decodeError(status int, body []byte) error {
	var eb apicontract.ErrorBody
	if err := json.Unmarshal(body, &eb); err != nil || eb.Code == "" {
		return &apicontract.Error{Status: status, Code: "unexpected_status",
			Message: strings.TrimSpace(string(body))}
	}
	return &apicontract.Error{Status: status, Code: eb.Code, Message: eb.Message, Details: eb.Details}
}
