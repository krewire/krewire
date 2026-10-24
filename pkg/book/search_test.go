package book

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestBuildSearchIndex(t *testing.T) {
	dir := writeContent(t, map[string]string{
		"01-intro.md": "# Introduction\n\nWelcome to Krewire. This guide teaches you how to build fast apps.\n\n## Getting Started\n\nFirst install the Go toolchain.\n",
		"02-guide.md": "# Architecture Guide\n\nDetailed breakdown of components and services.\n\n### Web Layer\n\nHTTP and WebSocket routing protocols.\n",
	})

	out := filepath.Join(t.TempDir(), "out")
	created, err := Build(Config{
		Input:  dir,
		Output: out,
		Title:  "Documentation Book",
		Author: "Krewire Team",
	})
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	searchPath := filepath.Join(out, "search-index.json")
	navPath := filepath.Join(out, "nav.json")

	foundSearch := false
	foundNav := false
	for _, p := range created {
		if p == searchPath {
			foundSearch = true
		}
		if p == navPath {
			foundNav = true
		}
	}
	if !foundSearch {
		t.Errorf("search-index.json not listed in created paths: %v", created)
	}
	if !foundNav {
		t.Errorf("nav.json not listed in created paths: %v", created)
	}

	// Verify search-index.json content
	searchData, err := os.ReadFile(searchPath)
	if err != nil {
		t.Fatalf("failed to read search-index.json: %v", err)
	}

	var searchIdx SearchIndex
	if err := json.Unmarshal(searchData, &searchIdx); err != nil {
		t.Fatalf("failed to unmarshal search-index.json: %v", err)
	}

	if searchIdx.BookTitle != "Documentation Book" {
		t.Errorf("expected BookTitle 'Documentation Book', got %q", searchIdx.BookTitle)
	}
	if len(searchIdx.Docs) < 2 {
		t.Errorf("expected at least 2 search docs, got %d", len(searchIdx.Docs))
	}

	foundAnchor := false
	for _, doc := range searchIdx.Docs {
		if doc.Anchor == "getting-started" {
			foundAnchor = true
			if doc.Path != "/intro#getting-started" {
				t.Errorf("expected path '/intro#getting-started', got %q", doc.Path)
			}
		}
	}
	if !foundAnchor {
		t.Errorf("expected anchor 'getting-started' to be indexed in search docs")
	}

	// Verify nav.json content
	navData, err := os.ReadFile(navPath)
	if err != nil {
		t.Fatalf("failed to read nav.json: %v", err)
	}

	var navManifest NavManifest
	if err := json.Unmarshal(navData, &navManifest); err != nil {
		t.Fatalf("failed to unmarshal nav.json: %v", err)
	}

	if navManifest.Title != "Documentation Book" {
		t.Errorf("expected nav Title 'Documentation Book', got %q", navManifest.Title)
	}
	if len(navManifest.Items) != 2 {
		t.Errorf("expected 2 nav items, got %d", len(navManifest.Items))
	}
	if navManifest.Items[0].Title != "Introduction" {
		t.Errorf("expected first item 'Introduction', got %q", navManifest.Items[0].Title)
	}
	if navManifest.Items[0].NextPath != "/guide" {
		t.Errorf("expected next path '/guide', got %q", navManifest.Items[0].NextPath)
	}
}

func TestBuildSearchIndexMounted(t *testing.T) {
	dir := writeContent(t, map[string]string{
		"index.md": "# Overview\n\nDocs overview.\n",
	})
	out := filepath.Join(t.TempDir(), "out")
	_, err := Build(Config{
		Input:     dir,
		Output:    out,
		Title:     "Docs",
		MountPath: "/docs/",
	})
	if err != nil {
		t.Fatalf("Build mounted failed: %v", err)
	}

	searchPath := filepath.Join(out, "docs", "search-index.json")
	if _, err := os.Stat(searchPath); err != nil {
		t.Errorf("mounted search-index.json missing: %v", err)
	}

	navPath := filepath.Join(out, "docs", "nav.json")
	if _, err := os.Stat(navPath); err != nil {
		t.Errorf("mounted nav.json missing: %v", err)
	}
}
