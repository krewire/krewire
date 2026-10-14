# Workload: Kernel (`kernel`)

The `kernel` kind represents the minimal viable Krewire project seed. Every new project created via `kiw new` starts as a kernel before being progressively enhanced to a specialized workload.

---

## 1. Key Characteristics

- **Zero Bloat**: Contains only standard Go tooling and the declarative `krewire.yaml` file.
- **Immediate Compilation**: Builds instantly with `go build` without external dependencies.
- **Seamless Progression**: Upgrades to any of the other seven workload kinds in-place using `kiw init`.

---

## 2. Directory Layout

```text
my-kernel/
├── go.mod                # Go module descriptor
├── krewire.yaml          # Project configuration (kind: kernel)
├── main.go               # Minimal runnable Go program
└── .gitignore            # Git exclusion rules
```

---

## 3. Configuration (`krewire.yaml`)

```yaml
version: "0.1.0"
project:
  name: my-kernel
  kind: kernel
```

---

## 4. Progressive Upgrades

From a clean kernel, equip the project with the desired workload:

```bash
# Upgrade to static website
kiw init --site

# Upgrade to documentation book
kiw init --book

# Upgrade to CLI utility
kiw init --cli

# Install AI-Agent Boost template
kiw boost install
```
