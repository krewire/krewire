# Boost AI Agent Roles & Specializations

A comprehensive directory and capability boundary reference for all 21 specialized subagents defined under `.agents/agents/`.

---

## 1. Governance & Leadership

- **`leader` (Outcome Authority)**:
  - Owns final acceptance, release readiness, and scope trade-offs.
  - Verifies that all acceptance criteria are met before closing user requests.
- **`orchestrator` (Multi-Agent Coordinator)**:
  - Coordinates multi-phase execution graphs across subagents.
  - Tracks task dependencies and ensures serial or parallel agent handoffs.
- **`manager` (Clarification & Routing)**:
  - Disambiguates vague user requests through structured questions.
  - Routes tasks to the fittest specialized subagents.
- **`supervisor` (Runtime Monitoring)**:
  - Monitors agent execution timeouts, retry budgets, and state recovery.

---

## 2. Architecture & Design

- **`planner` (Requirements & Backlog)**:
  - Elicits PRDs, work breakdown structures (WBS), MoSCoW prioritization, and user stories.
- **`thinker` (Architectural Reasoning)**:
  - Evaluates technical trade-offs, ADRs, pros/cons, and alternatives. Read-only: never modifies code.
- **`designer` (System Architecture)**:
  - Translates requirements into C4 architectural diagrams, ERDs, API schemas, and threat models.

---

## 3. Implementation & Refactoring

- **`builder` (Feature Implementation)**:
  - Writes production Go code, implements features, and fixes bugs following project conventions.
- **`decomposer` (SRP & Clean Code Refactoring)**:
  - Decomposes oversized god-classes, long functions, and monolithic files into single-responsibility units.
- **`automator` (Tooling & Scripting)**:
  - Scaffolds devtools, code generators, batch scripts, and CI workflows.
- **`customizer` (Agent Store Tailoring)**:
  - Creates and standardizes custom agents, skills, slash commands, and rules.

---

## 4. Verification, Testing & Security

- **`tester` (Quality Assurance)**:
  - Writes unit tests, table-driven tests, HTTP test harnesses, and verifies code coverage gates.
- **`auditor` (Security & Compliance)**:
  - Audits code against OWASP Top 10, CWE-22 (SafePath), CWE-918 (SSRF), secret leaks, and PII masking.
- **`tracker` (GitHub Issues & SSOT)**:
  - Discovers open issues, updates task checklists (`- [ ]` → `- [x]`), and maintains issue traceability.

---

## 5. Operations & Knowledge Hygiene

- **`deployer` (CI/CD & Infrastructure)**:
  - Manages deployment targets (systemd, Docker, binary, GitHub Pages) and infrastructure blueprints.
- **`maintainer` (Operations & Triage)**:
  - Triages bugs, updates dependencies, manages patches, and resolves technical debt.
- **`explorer` (Codebase Exploration)**:
  - Scans unfamiliar repositories, maps directory structures, and locates symbols.
- **`researcher` (External Research)**:
  - Evaluates external libraries, RFC specifications, and industry best practices.
- **`housekeeper` (Memory & Knowledge Hygiene)**:
  - Prunes outdated session context, tidies temporary files, and maintains index documentation.
- **`learner` (Self-Improvement)**:
  - Consolidates developer preferences and corrected agent mistakes into persistent project rules.
- **`writer` (Documentation & Copy)**:
  - Authors documentation, API references, Go docstrings, and UI copy.
