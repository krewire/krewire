# Release, Versioning, and Compatibility Specification

This specification documents the canonical release procedure, semantic compatibility expectations, and mandatory quality gates for the Krewire core repository and ecosystem.

---

## 1. Canonical Version Single Source of Truth (SSOT)

The definitive, single source of truth for the core ecosystem version is declared in:
- [`pkg/version/version.go`](../../pkg/version/version.go) via `version.CurrentVersion = MustParseVersion("0.2.0")`.

CLI binaries (`tools/kiw`), SSG generators, and metadata reporters derive their version string directly from `pkg/version` or inject it dynamically during build without manual drift:
```bash
go build -ldflags="-s -w -X 'github.com/krewire/krewire/tools/kiw/internal/buildinfo.Version=v0.2.0'" ./tools/kiw/cmd/kiw
```

---

## 2. Semantic Versioning & Compatibility Contract

Krewire adheres strictly to [Semantic Versioning 2.0.0](https://semver.org/):

- **Pre-1.0.0 Phase (`0.y.z`)**:
  - `0.y.0` (Minor): Introduces additive features, new domain packages, or minor behavioral improvements. Breaking changes require migration notes in `docs/overview/upgrade-guide.md`.
  - `0.y.z` (Patch): Backwards-compatible bug fixes, security patches, and performance optimizations.
  - Automated compatibility checking is enforced by `version.IsCompatible()` (for `0.y.z`, minor versions must match).
- **Post-1.0.0 Phase (`x.y.z`)**:
  - `x.0.0` (Major): Incompatible public API changes.
  - `x.y.0` (Minor): Backwards-compatible functionality additions.
  - `x.y.z` (Patch): Backwards-compatible bug fixes.

---

## 3. The Six Version Release Tracks

Every module in the ecosystem releases across six designated tracks ([ADR-R7P3K](../../../internal/docs/adr/ADR-R7P3K-multi-track-release-hotfix-deploy-pipelines.md)):

1. **`lts` (Long-Term Support)**: Extended enterprise lifecycle (`vX.Y.Z-lts` or `lts/vX.Y` branch). Strict bug fixes and security patches only.
2. **`stable` (General Availability)**: Production baseline (`vX.Y.Z`), tagged directly from `main` after passing 100% of quality gates.
3. **`pre-release` (RC, Beta, Alpha)**:
   - `rc`: Release Candidate (`vX.Y.Z-rc.N`), feature-complete, frozen for production evaluation.
   - `beta`: Community preview (`vX.Y.Z-beta.N`), feature-complete for external integration testing.
   - `alpha`: Early preview (`vX.Y.Z-alpha.N`), experimental architecture validation.
4. **`nightly` (Daily Automated Builds)**: Bleeding-edge automated builds (`vX.Y.Z-nightly.YYYYMMDD`), published daily from `main`.
5. **`experimental` (Architecture Spikes & Canary)**: Exploratory features on `exp/*` or `experimental/*` branches (`vX.Y.Z-exp.<feature>`).
6. **`development` (`dev`)**: Continuous integration verification on active `main` and feature branches (`vX.Y.Z-dev.<sha>`).

---

## 4. Automated Release, Hotfix, and Deploy Pipelines

Each repository operates three automated pipelines:
- **Release Pipeline (`release.yaml`)**: Handles multi-track tagging, multi-platform cross-compilation (`linux/amd64`, `linux/arm64`, `darwin/amd64`, `darwin/arm64`, `windows/amd64`), SHA-256 checksums, and Docker image publishing.
- **Hotfix Pipeline (`hotfix.yaml`)**: Fast-track emergency patch pipeline with automatic fallback to `:stable` on failure, patch tagging, and automated back-merge to development branches.
- **Deploy Pipeline (`deploy.yaml`)**: Multi-environment deployment targeting production (`stable`, `lts`), staging (`pre-release`), and canary (`nightly`, `experimental`) with atomic rolling updates and live health polling.

---

## 5. Automated Release Quality Gates

Before any tag or release artifact is published, the codebase must pass all 6 verification gates:

1. **Formatting Gate**:
   ```bash
   gofmt -l .
   ```
   *Must output 0 lines.*
2. **Static Analysis Gate**:
   ```bash
   go vet ./...
   ```
   *Must exit with code 0.*
3. **Layering & Invariant Gate**:
   ```bash
   go test -v ./layering_test.go
   ```
   *Enforces strict downward dependency hierarchy (Layer N cannot import Layer > N).*
4. **Unit & Integration Suite Gate**:
   ```bash
   go test ./...
   ```
   *All packages must pass.*
5. **Pure-Go Compilation Gate**:
   ```bash
   CGO_ENABLED=0 go build ./...
   ```
   *Ensures standalone portability with zero C runtime linkages.*
6. **Bitwise Reproducibility Gate**:
   ```bash
   go test -v -run TestPlatform_BitwiseReproducibility .
   ```
   *Verifies consecutive clean builds produce bitwise identical binaries.*

---

## 4. Release Checklist & Execution

1. **Bump Version**: Update `version.CurrentVersion` in `pkg/version/version.go`.
2. **Run Verification**:
   ```bash
   go test -v ./...
   ```
3. **Commit & Tag**:
   ```bash
   git add pkg/version/version.go CHANGELOG.md
   git commit -m "chore(release): bump version to v0.2.0"
   git tag -a v0.2.0 -m "release: v0.2.0"
   ```
4. **Build Distribution Binaries**:
   ```bash
   GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o dist/kiw-linux-amd64 ./tools/kiw/cmd/kiw
   GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="-s -w" -o dist/kiw-darwin-arm64 ./tools/kiw/cmd/kiw
   GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o dist/kiw-windows-amd64.exe ./tools/kiw/cmd/kiw
   ```
