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
	configPath := filepath.Join(dir, "book.yaml")
	content := `
title: "The Distributed Systems Manual"
author: "Alice Gopher"
input: "docs"
output: "dist"
base: "/book/"
theme: "dark"
nav:
  - text: "GitHub"
    url: "https://github.com/example/book"
footer: "Copyright 2026 Alice"
`
	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	config, err := LoadConfigFile(configPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if config.Title != "The Distributed Systems Manual" {
		t.Errorf("expected title 'The Distributed Systems Manual', got %q", config.Title)
	}
	if config.Author != "Alice Gopher" {
		t.Errorf("expected author 'Alice Gopher', got %q", config.Author)
	}
	if config.Input != "docs" {
		t.Errorf("expected input 'docs', got %q", config.Input)
	}
	if config.Output != "dist" {
		t.Errorf("expected output 'dist', got %q", config.Output)
	}
	if config.BasePath != "/book/" {
		t.Errorf("expected base '/book/', got %q", config.BasePath)
	}
	if config.FooterText != "Copyright 2026 Alice" {
		t.Errorf("expected footer 'Copyright 2026 Alice', got %q", config.FooterText)
	}
	if config.Theme == nil || config.Theme.Default != "dark" {
		t.Errorf("expected dark theme, got %v", config.Theme)
	}
	if len(config.NavLinks) != 1 || config.NavLinks[0].Text != "GitHub" {
		t.Errorf("expected nav link to GitHub, got %v", config.NavLinks)
	}
}

func TestLiveHandler(t *testing.T) {
	dir := t.TempDir()
	chapFile := filepath.Join(dir, "01-intro.md")
	if err := os.WriteFile(chapFile, []byte("# Hello Live\n\nInitial content."), 0o644); err != nil {
		t.Fatal(err)
	}

	config := Config{
		Input: dir,
		Title: "Live Book",
	}

	handler := LiveHandler(config)

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
