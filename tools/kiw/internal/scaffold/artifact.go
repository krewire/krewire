package scaffold

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"
)

// ErrUnknownKind is returned when an unsupported artifact kind is requested.
var ErrUnknownKind = errors.New("unknown kind")

// MakeArtifact scaffolds a new component, page, handler, migration, model,
// middleware, or service in the project root.
func MakeArtifact(root, kind, name string) error {
	switch strings.ToLower(kind) {
	case "component":
		return generateComponent(root, name)
	case "page":
		return generatePage(root, name)
	case "handler":
		return generateHandler(root, name)
	case "migration":
		return generateMigration(root, name)
	case "model":
		return generateModel(root, name)
	case "middleware":
		return generateMiddleware(root, name)
	case "service":
		return generateService(root, name)
	default:
		return fmt.Errorf("%w %q — supported: component, page, handler, migration, model, middleware, service", ErrUnknownKind, kind)
	}
}

func generateComponent(root, name string) error {
	name = strings.TrimSuffix(name, ".kiw")
	pascal := toPascal(name)
	slug := toSlug(name)
	dir := filepath.Join(root, "components")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	target := filepath.Join(dir, pascal+".kiw")
	if _, err := os.Stat(target); err == nil {
		return fmt.Errorf("component %q already exists at components/%s.kiw", pascal, pascal)
	}

	content := fmt.Sprintf(`{{/* components/%s.kiw */}}
<div class="%s">
  <h3>{{.Title}}</h3>
  <slot></slot>
</div>

<style>
.%s {
  display: block;
  padding: 1rem;
}
</style>
`, pascal, slug, slug)

	if err := os.WriteFile(target, []byte(content), 0o644); err != nil {
		return err
	}
	fmt.Printf("created components/%s.kiw\n", pascal)
	return nil
}

func generatePage(root, name string) error {
	name = strings.TrimSuffix(name, ".kiw")
	slug := toSlug(name)
	title := toTitle(name)
	dir := filepath.Join(root, "pages")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	target := filepath.Join(dir, slug+".kiw")
	if _, err := os.Stat(target); err == nil {
		return fmt.Errorf("page %q already exists at pages/%s.kiw", slug, slug)
	}

	content := fmt.Sprintf(`---
title: "%s"
layout: "Base"
---

<div class="%s-page">
  <h1>{{.Title}}</h1>
  <p>Content for %s page.</p>
</div>

<style>
.%s-page {
  max-width: 64rem;
  margin: 0 auto;
  padding: 2rem 1rem;
}
</style>
`, title, slug, title, slug)

	if err := os.WriteFile(target, []byte(content), 0o644); err != nil {
		return err
	}
	fmt.Printf("created pages/%s.kiw\n", slug)
	return nil
}

func generateHandler(root, name string) error {
	name = strings.TrimSuffix(name, ".go")
	pascal := toPascal(name)
	slug := toSlug(name)

	destDir := filepath.Join(root, "internal", "handlers")
	if hasDir(root, "handlers") {
		destDir = filepath.Join(root, "handlers")
	}
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return err
	}

	fileName := strings.ReplaceAll(slug, "-", "_") + ".go"
	target := filepath.Join(destDir, fileName)
	if _, err := os.Stat(target); err == nil {
		rel, _ := filepath.Rel(root, target)
		return fmt.Errorf("handler already exists at %s", rel)
	}

	content := fmt.Sprintf(`package handlers

import (
	"net/http"
)

// %sHandler handles HTTP requests for %s.
func %sHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`+"`"+`{"status":"ok","handler":"%sHandler"}`+"`"+`))
}
`, pascal, name, pascal, pascal)

	if err := os.WriteFile(target, []byte(content), 0o644); err != nil {
		return err
	}
	rel, _ := filepath.Rel(root, target)
	fmt.Printf("created %s\n", filepath.ToSlash(rel))
	return nil
}

func generateMigration(root, name string) error {
	name = strings.TrimSuffix(name, ".sql")
	slug := strings.ReplaceAll(toSlug(name), "-", "_")
	dir := filepath.Join(root, "migrations")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	ts := time.Now().UTC().Format("20060102150405")
	fileName := fmt.Sprintf("%s_%s.sql", ts, slug)
	target := filepath.Join(dir, fileName)

	content := fmt.Sprintf(`-- Migration: %s
-- Created at: %s

-- +up
-- Write your UP migration SQL here:


-- +down
-- Write your DOWN migration SQL here:

`, name, time.Now().UTC().Format(time.RFC3339))

	if err := os.WriteFile(target, []byte(content), 0o644); err != nil {
		return err
	}
	fmt.Printf("created migrations/%s\n", fileName)
	return nil
}

func generateModel(root, name string) error {
	name = strings.TrimSuffix(name, ".go")
	pascal := toPascal(name)
	slug := toSlug(name)

	destDir := filepath.Join(root, "internal", "models")
	if hasDir(root, "models") {
		destDir = filepath.Join(root, "models")
	}
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return err
	}

	fileName := strings.ReplaceAll(slug, "-", "_") + ".go"
	target := filepath.Join(destDir, fileName)
	if _, err := os.Stat(target); err == nil {
		rel, _ := filepath.Rel(root, target)
		return fmt.Errorf("model already exists at %s", rel)
	}

	content := fmt.Sprintf(`package models

import (
	"time"
)

// %s represents the %s domain entity.
type %s struct {
	ID        int64     `+"`"+`json:"id" db:"id"`+"`"+`
	CreatedAt time.Time `+"`"+`json:"created_at" db:"created_at"`+"`"+`
	UpdatedAt time.Time `+"`"+`json:"updated_at" db:"updated_at"`+"`"+`
}
`, pascal, pascal, pascal)

	if err := os.WriteFile(target, []byte(content), 0o644); err != nil {
		return err
	}
	rel, _ := filepath.Rel(root, target)
	fmt.Printf("created %s\n", filepath.ToSlash(rel))
	return nil
}

func generateMiddleware(root, name string) error {
	name = strings.TrimSuffix(name, ".go")
	pascal := toPascal(name)
	slug := toSlug(name)

	destDir := filepath.Join(root, "internal", "middleware")
	if hasDir(root, "middleware") {
		destDir = filepath.Join(root, "middleware")
	}
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return err
	}

	fileName := strings.ReplaceAll(slug, "-", "_") + ".go"
	target := filepath.Join(destDir, fileName)
	if _, err := os.Stat(target); err == nil {
		rel, _ := filepath.Rel(root, target)
		return fmt.Errorf("middleware already exists at %s", rel)
	}

	content := fmt.Sprintf(`package middleware

import (
	"net/http"
)

// %s is an HTTP middleware that wraps the next handler.
func %s(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Pre-handler logic
		next.ServeHTTP(w, r)
		// Post-handler logic
	})
}
`, pascal, pascal)

	if err := os.WriteFile(target, []byte(content), 0o644); err != nil {
		return err
	}
	rel, _ := filepath.Rel(root, target)
	fmt.Printf("created %s\n", filepath.ToSlash(rel))
	return nil
}

func generateService(root, name string) error {
	name = strings.TrimSuffix(name, ".go")
	pascal := toPascal(name)
	slug := toSlug(name)
	lowerFirst := strings.ToLower(string(pascal[0])) + pascal[1:]

	destDir := filepath.Join(root, "internal", "services")
	if hasDir(root, "services") {
		destDir = filepath.Join(root, "services")
	}
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return err
	}

	fileName := strings.ReplaceAll(slug, "-", "_") + ".go"
	target := filepath.Join(destDir, fileName)
	if _, err := os.Stat(target); err == nil {
		rel, _ := filepath.Rel(root, target)
		return fmt.Errorf("service already exists at %s", rel)
	}

	content := fmt.Sprintf(`package services

import (
	"context"
)

// %sService defines business operations for %s.
type %sService interface {
	Execute(ctx context.Context) error
}

type %sService struct {
}

// New%sService constructs a new %sService instance.
func New%sService() %sService {
	return &%sService{}
}

func (s *%sService) Execute(ctx context.Context) error {
	return nil
}
`, pascal, pascal, pascal, lowerFirst, pascal, pascal, pascal, pascal, lowerFirst, lowerFirst)

	if err := os.WriteFile(target, []byte(content), 0o644); err != nil {
		return err
	}
	rel, _ := filepath.Rel(root, target)
	fmt.Printf("created %s\n", filepath.ToSlash(rel))
	return nil
}

func hasDir(root, path string) bool {
	info, err := os.Stat(filepath.Join(root, path))
	return err == nil && info.IsDir()
}

func toPascal(s string) string {
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return r == '-' || r == '_' || r == ' ' || r == '.'
	})
	var b strings.Builder
	for _, p := range parts {
		if len(p) == 0 {
			continue
		}
		runes := []rune(p)
		runes[0] = unicode.ToUpper(runes[0])
		b.WriteString(string(runes))
	}
	if b.Len() == 0 {
		return "Item"
	}
	return b.String()
}

func toSlug(s string) string {
	var b strings.Builder
	prevDash := false
	for i, r := range s {
		if unicode.IsUpper(r) {
			if i > 0 && !prevDash {
				b.WriteRune('-')
			}
			b.WriteRune(unicode.ToLower(r))
			prevDash = false
		} else if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(unicode.ToLower(r))
			prevDash = false
		} else if !prevDash {
			b.WriteRune('-')
			prevDash = true
		}
	}
	res := strings.Trim(b.String(), "-")
	if res == "" {
		return "item"
	}
	return res
}

func toTitle(s string) string {
	slug := toSlug(s)
	words := strings.Split(slug, "-")
	for i, w := range words {
		if len(w) > 0 {
			runes := []rune(w)
			runes[0] = unicode.ToUpper(runes[0])
			words[i] = string(runes)
		}
	}
	return strings.Join(words, " ")
}
