package httpbind_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/krewire/krewire/apicontract"
	"github.com/krewire/krewire/apicontract/httpbind"
	"github.com/krewire/krewire/auth"
	"github.com/krewire/krewire/web"
)

type echoReq struct {
	Trace string   `json:"-" header:"X-Trace"`
	Tags  []string `json:"-" query:"tag"`
	Mode  string   `json:"-" query:"mode"`
}

type echoResp struct {
	Trace string   `json:"trace" validate:"required"`
	Tags  []string `json:"tags"`
}

var (
	echo      = apicontract.Define[echoReq, echoResp]("Echo", "GET", "/echo")
	basicOnly = apicontract.Define[apicontract.Empty, apicontract.Empty]("Basic", "GET", "/basic", apicontract.RequireAuth("basic"))
)

func serve(t *testing.T, h web.HandlerFunc, req *http.Request) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	h(rec, req, web.Params{})
	return rec
}

func TestHandlerErrorsAndBinding(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, nil))
	h := httpbind.Handler(echo, func(_ context.Context, r *echoReq) (*echoResp, error) {
		switch r.Mode {
		case "undeclared":
			return nil, apicontract.ErrorSpec{Status: 418, Code: "teapot"}.New("short and stout")
		case "web":
			return nil, web.Conflict("busy")
		case "plain":
			return nil, errors.New("secret detail")
		case "invalid":
			return &echoResp{}, nil
		}
		return &echoResp{Trace: r.Trace, Tags: r.Tags}, nil
	}, httpbind.ValidateResponses(), httpbind.WithLogger(logger))

	req := httptest.NewRequest("GET", "/echo?tag=a&tag=b", nil)
	req.Header.Set("X-Trace", "t1")
	if rec := serve(t, h, req); rec.Code != 200 || !strings.Contains(rec.Body.String(), `"tags":["a","b"]`) || !strings.Contains(rec.Body.String(), `"trace":"t1"`) {
		t.Fatalf("bind: %d %s", rec.Code, rec.Body)
	}
	cases := map[string]struct {
		status int
		code   string
	}{
		"undeclared": {418, "teapot"},
		"web":        {409, "conflict"},
		"plain":      {500, "internal_error"},
		"invalid":    {500, "internal_error"},
	}
	for mode, want := range cases {
		rec := serve(t, h, httptest.NewRequest("GET", "/echo?mode="+mode, nil))
		if rec.Code != want.status || !strings.Contains(rec.Body.String(), `"code":"`+want.code+`"`) {
			t.Errorf("%s: %d %s", mode, rec.Code, rec.Body)
		}
		if strings.Contains(rec.Body.String(), "secret detail") {
			t.Error("plain errors must not leak to clients")
		}
	}
	if !strings.Contains(logs.String(), "undeclared error") || !strings.Contains(logs.String(), "response violates contract") {
		t.Fatalf("expected contract warnings in logs: %s", logs.String())
	}
}

func TestAuthSchemeIsEnforced(t *testing.T) {
	h := httpbind.Handler(basicOnly, func(context.Context, *apicontract.Empty) (*apicontract.Empty, error) { return nil, nil })
	req := httptest.NewRequest("GET", "/basic", nil)
	jwtID := &auth.Identity{Subject: "u", Method: "jwt"}
	if rec := serve(t, h, req.WithContext(auth.WithIdentity(req.Context(), jwtID))); rec.Code != 401 {
		t.Fatalf("jwt identity on basic endpoint: %d", rec.Code)
	}
	basicID := &auth.Identity{Subject: "u", Method: "basic"}
	if rec := serve(t, h, req.WithContext(auth.WithIdentity(req.Context(), basicID))); rec.Code != 204 {
		t.Fatalf("basic identity: %d %s", rec.Code, rec.Body)
	}
}

func TestClientDecodesUnexpectedStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "upstream down", http.StatusBadGateway)
	}))
	defer srv.Close()
	_, err := httpbind.Call(context.Background(), httpbind.NewClient(srv.URL+"/"), echo, nil)
	var ce *apicontract.Error
	if !errors.As(err, &ce) || ce.Status != 502 || ce.Code != "unexpected_status" || ce.Message != "upstream down" {
		t.Fatalf("err = %#v", err)
	}
}
