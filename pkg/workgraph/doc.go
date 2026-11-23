// Package workgraph is the workspace dependency graph and affected-task
// engine shared by the Krewire developer toolchain and its AI-native
// workflows (issue #135).
//
// The package is deterministic and LLM-free. It:
//
//   - discovers Go modules, packages, templates, and workloads in a workspace
//     (one repository or several sibling repositories joined by go.work or a
//     krewire.workspace.yaml manifest) and records every dependency edge with
//     its provenance (file and line);
//   - validates the graph, rejecting cycles the architecture forbids;
//   - maps changed files (or a base...head Git range) to affected nodes and
//     explains each decision;
//   - expands task definitions (format, lint, test, build, generate, package,
//     release) into a target graph, computes content-addressed cache keys, and
//     runs targets in topological order with bounded parallelism;
//   - reports executed, restored, skipped, failed, and blocked targets with the
//     reason for each decision, as JSON for CI and AI workflows and as text for
//     humans.
//
// The CLI surface lives in krew; see docs/pkg/workgraph.md for the command
// wiring and the integration point with the toolchain contracts.
package workgraph
