package scaffold

import "fmt"

func chapterTemplate(title string, nextOrder int) string {
	return fmt.Sprintf(`---
title: "%s"
description: "Overview and guide for %s"
order: %d
tags:
  - docs
  - guide
---

# %s

Write your chapter content here in Markdown.
`, title, title, nextOrder, title)
}

func componentTemplate(pascal, slug string) string {
	return fmt.Sprintf(`{{/* components/%[1]s.kiw — use as <%[1]s title="…" variant="…">…</%[1]s>.
     Props are read lower-camel ({title}, {variant}); callers may write
     title= or Title=, aria-label= or ariaLabel= (lenient prop casing).
     {body} is the child content of a paired tag. */}}
<div class="{{class "%[2]s" .Variant "default" .Class}}">
  {#if title}<h3 class="%[2]s-title">{title}</h3>{/if}
  {#if body}<div class="%[2]s-body">{body}</div>{/if}
</div>

<style>
.%[2]s {
  display: block;
  padding: 1rem;
}
</style>
`, pascal, slug)
}

func pageTemplate(title, slug string) string {
	return fmt.Sprintf(`---
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
}

func handlerTemplate(pascal, name string) string {
	return fmt.Sprintf(`package handlers

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
}

func migrationTemplate(name, createdAt string) string {
	return fmt.Sprintf(`-- Migration: %s
-- Created at: %s

-- +up
-- Write your UP migration SQL here:


-- +down
-- Write your DOWN migration SQL here:

`, name, createdAt)
}

func modelTemplate(pascal string) string {
	return fmt.Sprintf(`package models

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
}

func middlewareTemplate(pascal string) string {
	return fmt.Sprintf(`package middleware

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
}

func serviceTemplate(pascal, lowerFirst string) string {
	return fmt.Sprintf(`package services

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
}

func jobTemplate(pascal string) string {
	return fmt.Sprintf(`package jobs

import (
	"context"
	"log/slog"
)

// %sJob represents a background task.
type %sJob struct {
	Payload string `+"`"+`json:"payload"`+"`"+`
}

// New%sJob creates a new %sJob instance.
func New%sJob(payload string) *%sJob {
	return &%sJob{
		Payload: payload,
	}
}

// Run executes the background job.
func (j *%sJob) Run(ctx context.Context) error {
	slog.Info("executing job", "job", "%sJob", "payload", j.Payload)
	return nil
}
`, pascal, pascal, pascal, pascal, pascal, pascal, pascal, pascal, pascal)
}
