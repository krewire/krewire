# Workload: Kernel (`kernel`)

The `kernel` kind is the minimal, pre-scaffold Krewire layout. It is the state
that `scaffold.New` writes before a variant is equipped, and the state every
`kiw init` / `kiw new` variant upgrades in place.

---

## 1. Key Characteristics

- **Zero Bloat**: standard Go tooling plus the declarative `krewire.yaml`; no Krewire imports.
- **Immediate Compilation**: `go build` succeeds with no external dependencies.
- **Seamless Progression**: every other kind is reached by equipping a variant, not by rewriting the project.

---

## 2. Directory Layout

```text
my-kernel/
├── go.mod                # Go module descriptor
├── krewire.yaml          # project name and version
├── main.go               # stdlib-only starter printing "hello, krewire"
└── .gitignore            # Git exclusion rules
```

---

## 3. Detection

`shape.Detect` reports `kernel` when all of the following hold:

- `go.mod`, `krewire.yaml`, `.gitignore`, and `main.go` exist;
- `main.go` still contains `hello, krewire`;
- none of `pages/`, `content/`, `cmd/`, `internal/`, or `components/` exists.

An explicit `project.kind: kernel` always wins over this inference.

---

## 4. Configuration (`krewire.yaml`)

The scaffold writes only the project name and version:

```yaml
project:
  name: my-kernel
  version: 0.1.0
```

Declaring the kind explicitly is valid but unnecessary:

```yaml
project:
  name: my-kernel
  kind: kernel
```

---

## 5. Progressive Upgrades

`kiw new <name>` writes this layout and then equips the fullstack `app` variant
by default; `kiw init` without a variant stops after the config files. To move a
kernel to another kind, equip it explicitly:

```bash
kiw init --site      # static website
kiw init --book      # documentation book
kiw init --cli       # command-line tool
kiw init --worker    # background worker
kiw init --service   # microservice
kiw init --infra     # infrastructure blueprint
kiw boost install    # AI-agent Boost template
```
