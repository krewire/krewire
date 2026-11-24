package apicontract_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/krewire/krewire/apicontract"
)

func TestValidateJSON(t *testing.T) {
	doc := testService("1.0.0").MustCompile()
	item := &apicontract.Schema{Ref: apicontract.RefPrefix + "Item"}
	ok := `{"id":1,"title":"a","priority":"low","created":"2026-11-23T10:00:00Z","labels":[{"name":"x","color":"#00ff00"}],"meta":{"k":"v"},"parent":null}`
	if err := doc.ValidateJSON(item, []byte(ok)); err != nil {
		t.Fatalf("valid item rejected: %v", err)
	}
	cases := map[string]string{
		`{"id":1.5,"title":"a","priority":"low","created":"2026-11-23T10:00:00Z"}`:                                     "/id: want integer",
		`{"id":1,"priority":"low","created":"2026-11-23T10:00:00Z"}`:                                                   "/title: required property is missing",
		`{"id":1,"title":"","priority":"low","created":"2026-11-23T10:00:00Z"}`:                                        "/title: has 0 characters",
		`{"id":1,"title":"a","priority":"urgent","created":"2026-11-23T10:00:00Z"}`:                                    "/priority: value is not one of",
		`{"id":1,"title":"a","priority":"low","created":"yesterday"}`:                                                  "/created: want RFC 3339",
		`{"id":1,"title":"a","priority":"low","created":"2026-11-23T10:00:00Z","labels":[{"name":7}]}`:                 "/labels/0/name: want string",
		`{"id":1,"title":"a","priority":"low","created":"2026-11-23T10:00:00Z","labels":[{"name":"x","color":"red"}]}`: "/labels/0/color: does not match pattern",
		`{"id":1,"title":"a","priority":"low","created":"2026-11-23T10:00:00Z","meta":{"k":1}}`:                        "/meta/k: want string",
		`[1]`:   "/: want object",
		`{oops`: "invalid JSON",
	}
	for in, want := range cases {
		err := doc.ValidateJSON(item, []byte(in))
		var se *apicontract.SchemaError
		if !errors.As(err, &se) || !strings.Contains(err.Error(), want) {
			t.Errorf("%s: want violation %q, got %v", in, want, err)
		}
	}
}
