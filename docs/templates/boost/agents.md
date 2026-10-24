# Boost AI Agent Roles & Specializations

A comprehensive directory of the 21 specialized subagents defined under `.agents/agents/`.

---

## Agent Roster

| Agent | Role | Boundary & Primary Scope |
|---|---|---|
| `leader` | Outcome Authority | Owns decision gates, scope boundaries, and final acceptance |
| `planner` | Requirements & Backlog | Elicits requirements, PRD, backlog, WBS, MoSCoW, acceptance criteria |
| `thinker` | Architecture Reasoning | Evaluates trade-offs, ADRs, pros/cons (Read-only, never creates files) |
| `designer` | System Architecture | Translates requirements into C4 models, ERDs, APIs, and threat models |
| `builder` | Implementation | Writes production code, adds features, and fixes bugs |
| `decomposer` | SRP Refactoring | Decomposes god classes, long functions, and improves cohesion |
| `tester` | Quality Assurance | Writes unit, integration, and e2e tests; verifies coverage |
| `tracker` | GitHub Issue Tracking | Opens, closes, lists, and views issues; enforces traceability |
| `auditor` | Security & Compliance | Audits against OWASP, CWE, PII redaction, and security policies |
| `automator` | Tooling & Scripting | Creates devtools, batch scripts, generators, and CI automations |
| `customizer` | Agent Store Tailoring | Scaffolds and standardizes agents, skills, commands, rules, and triads |
| `deployer` | CI/CD & Infrastructure | Manages pipelines, Docker, Kubernetes, and cloud deployments |
| `explorer` | Codebase Exploration | Maps unknown repos, locates symbols, and inventories assets |
| `researcher` | External Research | Conducts external research, benchmarks, and market trend analysis |
| `housekeeper` | Memory & Knowledge Hygiene | Tidies memory, prunes outdated context, and maintains index files |
| `learner` | Self-Improvement | Consolidates learnings into workflow rules and memory across sessions |
| `maintainer` | Ops & Incident Response | Handles bug triage, patch management, hotfixes, and tech debt |
| `manager` | Clarification & Routing | Disambiguates vague user requests and delegates to fittest agents |
| `orchestrator` | Multi-Agent Coordination | Coordinates multi-phase execution graphs and dependencies |
| `supervisor` | Runtime Monitoring | Monitors agent timeouts, retry policies, and execution checkpoints |
| `writer` | Documentation & Copy | Writes markdown docs, PHPDoc/godoc docblocks, and UI copy |
