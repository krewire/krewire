// Package contracttest turns a compiled contract into Go test gates (ref
// #137): baseline compatibility checks that force an explicit versioning
// decision, golden artifact checks, and conformance cases that exercise a
// live handler and verify every response against the contract.
//
// All checks are deterministic and need no network or LLM, so they run in
// `go test ./...` and therefore in CI.
package contracttest

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/krewire/krewire/pkg/apicontract"
)

// UpdateEnv, set to "1", makes CheckBaseline and Golden rewrite their files
// instead of comparing. Commit the rewritten files with the change.
const UpdateEnv = "KREWIRE_CONTRACT_UPDATE"

func updating() bool { return os.Getenv(UpdateEnv) == "1" }

// CheckBaseline compares doc with the committed baseline document at path.
//
// It fails when the changes require a larger version bump than the contract
// declares (the compatibility report is logged as JSON), and when the
// contract changed at all without the baseline being refreshed — so every
// contract change, and its version decision, is visible in review. With
// KREWIRE_CONTRACT_UPDATE=1 it rewrites the baseline after the version check
// passes.
func CheckBaseline(t testing.TB, doc *apicontract.Document, path string) *apicontract.Report {
	t.Helper()
	got, err := doc.JSON()
	if err != nil {
		t.Fatalf("contracttest: encode %s: %v", doc.Service, err)
	}
	base, err := apicontract.LoadDocument(path)
	if os.IsNotExist(err) && updating() {
		writeFile(t, path, got)
		return nil
	}
	if err != nil {
		t.Fatalf("contracttest: load baseline %s: %v (create it with %s=1)", path, err, UpdateEnv)
	}
	report := apicontract.Diff(base, doc)
	if !report.VersionOK {
		out, _ := report.JSON()
		t.Fatalf("contracttest: %s\n%s", report.Decision, out)
	}
	want, _ := base.JSON()
	if !bytes.Equal(want, got) {
		if updating() {
			writeFile(t, path, got)
			return report
		}
		out, _ := report.JSON()
		t.Errorf("contracttest: contract %s changed (%s -> %s); refresh the baseline with %s=1\n%s",
			doc.Service, base.Version, doc.Version, UpdateEnv, out)
	}
	return report
}

// Golden compares got with the file at path, rewriting it under
// KREWIRE_CONTRACT_UPDATE=1. Use it for generated artifacts such as the
// OpenAPI document and generated clients.
func Golden(t testing.TB, path string, got []byte) {
	t.Helper()
	if updating() {
		writeFile(t, path, got)
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("contracttest: read golden %s: %v (create it with %s=1)", path, err, UpdateEnv)
	}
	if !bytes.Equal(want, got) {
		t.Errorf("contracttest: %s is stale; regenerate with %s=1", path, UpdateEnv)
	}
}

func writeFile(t testing.TB, path string, data []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

// Case is one conformance request against a contract endpoint.
type Case struct {
	Name     string
	Endpoint string
	// Path fills {param} segments.
	Path   map[string]string
	Query  url.Values
	Header http.Header
	// Body is sent as JSON; a string or []byte is sent verbatim.
	Body any
	// WantStatus defaults to the endpoint's success status.
	WantStatus int
	// WantCode is the expected error code for error responses.
	WantCode string
}

// Run executes each case against h and checks that the response matches the
// contract: the expected status and code, a body that validates against the
// declared schema, and an error that the endpoint declares.
func Run(t *testing.T, h http.Handler, doc *apicontract.Document, cases []Case) {
	t.Helper()
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			ep := doc.Endpoint(tc.Endpoint)
			if ep == nil {
				t.Fatalf("contract %s has no endpoint %q", doc.Service, tc.Endpoint)
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, request(t, doc, ep, tc))
			check(t, doc, ep, tc, rec)
		})
	}
}

func request(t *testing.T, doc *apicontract.Document, ep *apicontract.EndpointDoc, tc Case) *http.Request {
	path := doc.BasePath + ep.Path
	for k, v := range tc.Path {
		path = strings.ReplaceAll(path, "{"+k+"}", url.PathEscape(v))
	}
	if len(tc.Query) > 0 {
		path += "?" + tc.Query.Encode()
	}
	var body []byte
	switch b := tc.Body.(type) {
	case nil:
	case string:
		body = []byte(b)
	case []byte:
		body = b
	default:
		var err error
		if body, err = json.Marshal(b); err != nil {
			t.Fatalf("encode body: %v", err)
		}
	}
	r := httptest.NewRequest(ep.Method, path, bytes.NewReader(body))
	if body != nil {
		r.Header.Set("Content-Type", "application/json")
	}
	for k, vs := range tc.Header {
		r.Header[http.CanonicalHeaderKey(k)] = vs
	}
	return r
}

func check(t *testing.T, doc *apicontract.Document, ep *apicontract.EndpointDoc, tc Case, rec *httptest.ResponseRecorder) {
	want := tc.WantStatus
	if want == 0 {
		want = ep.Status
	}
	body := rec.Body.Bytes()
	if rec.Code != want {
		t.Fatalf("status = %d, want %d; body %s", rec.Code, want, body)
	}
	if rec.Code == ep.Status {
		if ep.Response != nil {
			if err := doc.ValidateJSON(ep.Response, body); err != nil {
				t.Errorf("success body violates contract: %v", err)
			}
		}
		return
	}
	if err := doc.ValidateJSON(&apicontract.Schema{Ref: apicontract.RefPrefix + doc.ErrorSchema}, body); err != nil {
		t.Fatalf("error body violates contract: %v", err)
	}
	var eb apicontract.ErrorBody
	_ = json.Unmarshal(body, &eb)
	if tc.WantCode != "" && eb.Code != tc.WantCode {
		t.Errorf("error code = %q, want %q", eb.Code, tc.WantCode)
	}
	for _, e := range ep.Errors {
		if e.Status == rec.Code && e.Code == eb.Code {
			return
		}
	}
	t.Errorf("error %s (%d) is not declared by %s", eb.Code, rec.Code, ep.Name)
}

// Uncovered lists the contract behaviours the cases do not exercise: a
// success per endpoint, a validation failure per endpoint with a request
// body, an authentication failure per authenticated endpoint, a forbidden
// response per role-restricted endpoint, and every domain error the endpoint
// declares. Built-in bad_request and internal_error are exempt.
func Uncovered(doc *apicontract.Document, cases []Case) []string {
	hit := map[string]bool{}
	for _, c := range cases {
		hit[c.Endpoint+"/"+c.WantCode] = true
		if c.WantCode == "" && (c.WantStatus == 0 || c.WantStatus < 300) {
			hit[c.Endpoint+"/"] = true
		}
	}
	var missing []string
	need := func(ep, code, what string) {
		if !hit[ep+"/"+code] {
			missing = append(missing, ep+": "+what)
		}
	}
	for _, ep := range doc.Endpoints {
		need(ep.Name, "", "success")
		if ep.Body != nil {
			need(ep.Name, apicontract.ErrValidation.Code, "validation failure")
		}
		for _, e := range ep.Errors {
			switch e.Code {
			case apicontract.ErrBadRequest.Code, apicontract.ErrInternal.Code, apicontract.ErrValidation.Code:
				continue
			case apicontract.ErrUnauthenticated.Code:
				need(ep.Name, e.Code, "authentication failure")
			case apicontract.ErrForbidden.Code:
				need(ep.Name, e.Code, "forbidden role")
			default:
				need(ep.Name, e.Code, "declared error "+e.Code)
			}
		}
	}
	sort.Strings(missing)
	return missing
}

// RequireCoverage fails t when Uncovered reports gaps.
func RequireCoverage(t testing.TB, doc *apicontract.Document, cases []Case) {
	t.Helper()
	if missing := Uncovered(doc, cases); len(missing) > 0 {
		t.Errorf("contract cases do not cover:\n  %s", strings.Join(missing, "\n  "))
	}
}
