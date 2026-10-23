package app_test

import (
	"testing"
	"testing/fstest"

	"github.com/krewire/krewire/packages/app"
	"github.com/krewire/krewire/packages/i18n"
)

func TestI18nProvider(t *testing.T) {
	fsys := fstest.MapFS{
		"lang/en.json": &fstest.MapFile{
			Data: []byte(`{"hello": "Hello"}`),
		},
		"lang/id.json": &fstest.MapFile{
			Data: []byte(`{"hello": "Halo"}`),
		},
	}

	bundle, err := i18n.New(fsys)
	if err != nil {
		t.Fatalf("i18n.New: %v", err)
	}

	application, err := app.NewApp(app.I18nProvider(bundle)).Build()
	if err != nil {
		t.Fatalf("app.Build: %v", err)
	}

	resolvedBundle, err := app.Resolve[*i18n.Bundle](application)
	if err != nil {
		t.Fatalf("Resolve[*Bundle]: %v", err)
	}
	if resolvedBundle == nil {
		t.Fatal("expected non-nil bundle")
	}

	tr, err := app.Resolve[*i18n.Translator](application)
	if err != nil {
		t.Fatalf("Resolve[*Translator]: %v", err)
	}
	if tr == nil || tr.Locale() != "en" {
		t.Fatalf("expected translator with locale 'en', got %v", tr)
	}
}

func TestI18nProviderWithConfig(t *testing.T) {
	fsys := fstest.MapFS{
		"lang/en.json": &fstest.MapFile{
			Data: []byte(`{"hello": "Hello"}`),
		},
		"lang/id.json": &fstest.MapFile{
			Data: []byte(`{"hello": "Halo"}`),
		},
	}

	bundle, err := i18n.New(fsys)
	if err != nil {
		t.Fatalf("i18n.New: %v", err)
	}

	cfg := i18n.Config{
		DefaultLocale:  "id",
		FallbackLocale: "en",
	}

	application, err := app.NewApp(app.I18nProviderWithConfig(bundle, cfg)).Build()
	if err != nil {
		t.Fatalf("app.Build: %v", err)
	}

	tr, err := app.Resolve[*i18n.Translator](application)
	if err != nil {
		t.Fatalf("Resolve[*Translator]: %v", err)
	}
	if tr == nil || tr.Locale() != "id" {
		t.Fatalf("expected translator with locale 'id', got %v", tr)
	}
}
