package clientgen_test

import (
	"strings"
	"testing"

	"github.com/krewire/krewire/pkg/apicontract"
	"github.com/krewire/krewire/pkg/apicontract/clientgen"
)

type Page[T any] struct {
	Items []T `json:"items"`
}

func TestGenerateRejectsUnsupportedInput(t *testing.T) {
	svc := apicontract.NewService("svc", "1.0.0").Add(
		apicontract.Define[apicontract.Empty, Page[int]]("List", "GET", "/list"))
	if _, err := clientgen.Generate(svc, clientgen.Options{}); err == nil {
		t.Fatal("missing options must fail")
	}
	_, err := clientgen.Generate(svc, clientgen.Options{Package: "c", DefsImport: "example.com/defs"})
	if err == nil || !strings.Contains(err.Error(), "generic type") {
		t.Fatalf("generic response must be rejected, got %v", err)
	}
	bad := apicontract.NewService("svc", "bad")
	if _, err := clientgen.Generate(bad, clientgen.Options{Package: "c", DefsImport: "example.com/defs"}); err == nil {
		t.Fatal("invalid contract must fail")
	}
}
