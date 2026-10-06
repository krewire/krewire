# Specification — Forge Atomic Component Library

| Field       | Value                                        |
| ----------- | -------------------------------------------- |
| SpecID      | KWF-M4R8T                                    |
| Title       | Forge Atomic `.kiw` Component Library        |
| Status      | Draft                                        |
| Date        | 2026-10-03                                   |
| Author      | Krewire Contributors                          |
| Domain      | Forge — Components — Atomic                   |

## 1. Context

Forge ships two component surfaces:

- **Atomic `.kiw` components** in `components/` — embedded via `components.FS`
  and loaded into every file-based site by `framework/web/ssg/load.go`.
- **Programmatic Go widgets** in `widget/` — `Button`, `Card`, `Table`, `Text`,
  `Heading`, `Divider`, `Stack`, `Grid`, `Alert`.

The `.kiw` set grew one component at a time with no written contract, so its
shape is only discoverable by reading 49 files. Two consequences showed up in
practice:

1. A component that fails to parse **disappears silently**. `load.go` calls
   `continue` when the module fails to parse, so a broken component is not a
   build failure — it is simply absent from every site. A draft of `Progress`
   called a `{{progressPercent}}` helper that no func map defines; nothing
   reported it.
2. Nesting a component inside another's body emitted the child as literal text
   instead of rendering it, because the body travelled as a string argument to
   `(dict ...)`. Fixed alongside this spec in `kiw/dsl` and `ssg.renderFragment`.

This document states what a component in this library is allowed to be, so the
next one can be written against a contract instead of a neighbour.

## 2. Problem Statement

- **No inventory.** A user cannot tell what exists without listing the directory.
- **Silent failure.** A malformed component yields a page that is quietly missing
  an element, with no warning anywhere in the build output.
- **Inconsistent identity.** 19 of 49 components carry no
  `data-kiw-component` hook, so scoped CSS and tests cannot target them.
- **Two vocabularies for one concept.** `Alert` used `danger`, `Callout` used
  `warn`, `Text` used `error`, while the theme tokens are `--error`. Resolved:
  `danger` and `warn` were removed (FRK-FA-031).
- **Inconsistent class naming.** Newer components prefix with `kiw-`; older ones
  ship bare `.btn`, `.card`, `.alert`.

## 3. Goals

- G1 — Every built-in component is discoverable from one table
- G2 — A broken component is caught by a test, not discovered in production
- G3 — Components compose: a child inside a parent's body renders
- G4 — No-JS components stay no-JS
- G5 — Accessible by default (`role` / `aria-*` where a widget has the semantics)

## 4. Non-Goals

- NG1 — Not a general design system; no token pipeline, no theme authoring
- NG2 — No JavaScript is emitted by any component in this spec
- NG3 — Not a replacement for `widget/`; the two APIs are not unified here
## 5. Requirements

### 5.1 Contract

| ID          | Requirement                                                          | Priority |
| ----------- | ------------------------------------------------------------------- | -------- |
| FRK-FA-001  | Every built-in component is a `.kiw` module embedded in `components.FS` | Must |
| FRK-FA-002  | Every component is registered into a file-based site with no import or registration step | Must |
| FRK-FA-003  | Every component carries a `data-kiw-component="<Name>"` hook on its root element | Must |
| FRK-FA-004  | A component that fails to parse fails a test; the loader must not be the only detector | Must |
| FRK-FA-005  | Every component ships its own `<style>` block, except pure passthrough fragments | Must |
| FRK-FA-006  | A component may contain other components; the child must render, not print as text | Must |

### 5.2 Composition

| ID          | Requirement                                                          | Priority |
| ----------- | ------------------------------------------------------------------- | -------- |
| FRK-FA-010  | `Body` accepts markup and renders it as HTML, not escaped text       | Must     |
| FRK-FA-011  | Props are addressed by name; an unset prop falls back to a documented default | Must |
| FRK-FA-012  | `Class` is accepted by every component that renders a container, so a caller can always add utility classes | Must |
| FRK-FA-013  | List components (`List`, `Tabs`, `Breadcrumb`) pass their children through `Body` | Must |

### 5.3 Accessibility & degradation

| ID          | Requirement                                                          | Priority |
| ----------- | ------------------------------------------------------------------- | -------- |
| FRK-FA-020  | `Progress` exposes `role="progressbar"` with `aria-valuenow`         | Must     |
| FRK-FA-021  | `Spinner` exposes `role="status"` when a label is given, else `aria-hidden` | Must |
| FRK-FA-022  | `Tooltip` is reachable by keyboard (`tabindex` + `:focus-visible`)   | Must     |
| FRK-FA-023  | `Breadcrumb` marks the current item with `aria-current="page"`       | Must     |
| FRK-FA-024  | `Tabs` renders the `tablist` relationship; panel switching is left to the host | Must |
| FRK-FA-025  | `Collapse` is built on `<details>` so disclosure works with JS disabled | Must |
| FRK-FA-026  | `Spinner` respects `prefers-reduced-motion`                         | Must     |

### 5.4 Vocabulary

| ID          | Requirement                                                          | Priority |
| ----------- | ------------------------------------------------------------------- | -------- |
| FRK-FA-030  | The semantic status set is `info`, `success`, `warning`, `error` — matching the `--error` theme token and the names `framework/ui` already uses | Must |
| FRK-FA-031  | `danger` and `warn` are removed, not aliased. `Alert.Danger()`, `Button.Danger()`, `Badge.Danger()` and the `.kiw` variants are gone; `Error()` and `Variant="error"` replace them | Must |
| FRK-FA-032  | New components use the `kiw-` class prefix; legacy bare class names are not renamed in place (breaking) | Must |

## 6. Non-Functional Requirements

- NFR1 — Deterministic: the set is defined by files on disk, not a hand-kept
## 7. Component Inventory

49 components. `*` marks a component missing the `data-kiw-component` hook
required by FRK-FA-003.

**Text & inline** — `Text`, `Heading`, `Code`, `Kbd`, `List`, `Link`,
`Badge`\*, `Terminal`, `CodeWindow`, `Head`\*
**Layout** — `Box`, `Stack`, `Flex`, `Grid`, `Container`, `Section`, `Divider`,
`Card`\*, `Table`
**Display & status** — `Alert`\*, `Callout`\*, `Avatar`, `Progress`, `Spinner`,
`ThemeSwitch`\*, `Toggle`\*
**Navigation** — `Navbar`\*, `NavMenu`\*, `Brand`\*, `Breadcrumb`,
`BreadcrumbItem`, `Sidebar`\*, `SideMenu`\*, `Drawer`, `Hamburger`,
`LangSwitch`, `Footer`\*, `Ecosystem`
**Forms** — `Form`, `Input`\*, `Textarea`\*, `Select`\*, `Checkbox`, `Radio`
**Overlays** — `Modal`\*, `Tooltip`, `Collapse`, `Tabs`

## 8. Success Criteria

- S1 — `TestAllForgeComponentsLoadIntoSite` fails when any component stops parsing
- S2 — `<Breadcrumb><BreadcrumbItem Label="Home" Href="/" /></Breadcrumb>` renders
  two `<li>`, not literal `{{component ...}}` text
- S3 — Every component renders through `LoadFromDir` + `Build` with no
  missing-template-function error
- S4 — A site still passing `Variant="danger"` renders an element with no
  matching CSS. It does **not** error: `.kiw` props are untyped, so a retired
  name degrades silently. That gap is the open item NG4 (prop-schema
  validation), not something this spec claims to solve.

## 9. Related Specifications

| SpecID      | Relationship |
| ----------- | ------------ |
| KWF-DF3PL | Loads a file-based site and registers these components |
| KWL-KIW-001 | The `.kiw` parser every component here is written against |
| KWL-LAYER-001 | Keeps that parser reachable from `framework` |
  list, so the two cannot drift
- NFR2 — A component embeds no script; behaviour comes from CSS or the host
- NFR3 — Backward compatible: renaming nothing is a hard requirement
- NG4 — No per-component prop schema validation at build time (open)