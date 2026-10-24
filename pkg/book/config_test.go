package book

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFindConfigFile(t *testing.T) {
	dir := t.TempDir()
	if got := FindConfigFile(dir); got != "" {
		t.Fatalf("expected empty, got %q", got)
	}

	bookYaml := filepath.Join(dir, "book.yaml")
	if err := os.WriteFile(bookYaml, []byte("title: Test Book\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if got := FindConfigFile(dir); got != bookYaml {
		t.Fatalf("expected %q, got %q", bookYaml, got)
	}
}

func TestLoadConfigFile(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "book.yaml")
	content := `
title: "The Distributed Systems Manual"
author: "Alice Gopher"
input: "manuscript"
output: "dist"
base: "/book/"
theme: "dark"
nav:
  - text: "GitHub"
    url: "https://github.com/example/book"
footer: "Copyright 2026 Alice"
`
	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfigFile(cfgPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Title != "The Distributed Systems Manual" {
		t.Errorf("expected title 'The Distributed Systems Manual', got %q", cfg.Title)
	}
	if cfg.Author != "Alice Gopher" {
		t.Errorf("expected author 'Alice Gopher', got %q", cfg.Author)
	}
	if cfg.Input != "manuscript" {
		t.Errorf("expected input 'manuscript', got %q", cfg.Input)
	}
	if cfg.Output != "dist" {
		t.Errorf("expected output 'dist', got %q", cfg.Output)
	}
	if cfg.BasePath != "/book/" {
		t.Errorf("expected base '/book/', got %q", cfg.BasePath)
	}
	if cfg.FooterText != "Copyright 2026 Alice" {
		t.Errorf("expected footer 'Copyright 2026 Alice', got %q", cfg.FooterText)
	}
	if cfg.Theme == nil || cfg.Theme.Default != "dark" {
		t.Errorf("expected dark theme, got %v", cfg.Theme)
	}
	if len(cfg.NavLinks) != 1 || cfg.NavLinks[0].Text != "GitHub" {
		t.Errorf("expected nav link to GitHub, got %v", cfg.NavLinks)
	}
}

func TestLiveHandler(t *testing.T) {
	dir := t.TempDir()
	chapFile := filepath.Join(dir, "01-intro.md")
	if err := os.WriteFile(chapFile, []byte("# Hello Live\n\nInitial content."), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := Config{
		Input: dir,
		Title: "Live Book",
	}

	handler := LiveHandler(cfg)

	// 1. Request chapter page
	req := httptest.NewRequest(http.MethodGet, "/intro", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Initial content.") {
		t.Fatalf("body missing 'Initial content.': %s", rec.Body.String())
	}

	// 2. Modify chapter file dynamically (simulating live editing)
	if err := os.WriteFile(chapFile, []byte("# Hello Live\n\nUpdated live on save!"), 0o644); err != nil {
		t.Fatal(err)
	}

	// 3. Request again, should reflect changes immediately
	req2 := httptest.NewRequest(http.MethodGet, "/intro", nil)
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec2.Code)
	}
	if !strings.Contains(rec2.Body.String(), "Updated live on save!") {
		t.Fatalf("live reload failed, got: %s", rec2.Body.String())
	}
}
