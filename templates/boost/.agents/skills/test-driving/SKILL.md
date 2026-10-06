---
name: test-driving
description: Use when working on tests: adding coverage, running the suite, or fixing failing tests. Trigger on "add tests", "run the tests", "test this", "fix failing test". Covers discovering the harness and iterating.
---

# Test Driving

A reliable loop for working with a project's tests.

## 1. Discover the harness

Read the manifest and scripts first. Common matches:

- Node: `jest`, `vitest`, `mocha` → `npm test`
- Python: `pytest`, `unittest` → `pytest`
- Go: testing stdlib → `go test ./...`
- Rust: built-in → `cargo test`
- Ruby: `rspec`, `minitest` → `bundle exec rspec`

If ambiguous, ask; do not guess a novel command.

## 2. Baseline

Run the existing suite once before changing anything. Record pass/fail counts so you know the starting point.

## 3. Write tests

- Mirror existing test naming, location, and style.
- Test behavior (input/output), not private implementation details.
- Cover the happy path, edge cases, and error paths.
- For bug fixes: write a failing test that reproduces the bug first, then fix — it becomes regression protection.

## 4. Iterate

Run the targeted test, then the full suite. Keep it green.

## 5. Report

State what was run, the results, and what tests were added or changed.

## Rules

Read `rules/key-rules.md` before acting and follow those rules.