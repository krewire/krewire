# Package: `config` & `validation` (Configuration & Validation)

The `packages/config` and `packages/validation` libraries handle type-safe YAML configuration loading, environment overrides, and struct-level validation rules.

---

## 1. Declarative Config Loader (`packages/config`)

- **SSOT Loading**: Reads the authoritative [`krewire.yaml`](../../packages/kern/model/project.go) file.
- **Environment Overrides**: Automatically reads `.env` and environment variables prefixed with `KREWIRE_`.
- **Kind-Specific Schema Mapping**: Decodes workload sections (`ssg`, `book`, `server`, `worker`, `service`) into strictly typed Go structs.

---

## 2. Struct Validation Engine (`packages/validation`)

Validates invariants using struct tags (`validate:"required,min=3"`):

```go
type ServerConfig struct {
	Host string `yaml:"host" validate:"required,ip|hostname"`
	Port int    `yaml:"port" validate:"required,min=1,max=65535"`
}

func Validate(val any) error
```

### Extensible Rules (`packages/validation/rules`)
Includes built-in validators for URLs, email addresses, IP addresses, alphanumeric slugs, and kebab-case strings, with support for custom validator functions.
