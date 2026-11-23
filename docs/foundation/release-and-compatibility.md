# Release, Versioning, and Compatibility Specification

This specification documents the canonical release procedure, semantic compatibility expectations, and mandatory quality gates for the Krewire core repository and ecosystem.

---

## 1. Canonical Version Single Source of Truth (SSOT)

The definitive, single source of truth for the core ecosystem version is declared in:
- [`pkg/version/version.go`](../../pkg/version/version.go) via `version.CurrentVersion = MustParseVersion("0.2.0")`.

Executables shipped by this module (currently `cmd/ai`) report their identity through `version.Build()`. The release pipeline stamps the exact release version, source commit, build date, and channel at link time; unstamped local builds fall back to `CurrentVersion` and the VCS metadata embedded by `-buildvcs`:
```bash
go build -trimpath -ldflags="-s -w -buildid= \
  -X github.com/krewire/krewire/pkg/version.buildVersion=v0.2.0 \
  -X github.com/krewire/krewire/pkg/version.buildCommit=$(git rev-parse HEAD) \
  -X github.com/krewire/krewire/pkg/version.buildChannel=stable" ./cmd/ai
ai --version   # ai v0.2.0 (channel stable, commit <sha>, built <date>, go1.27.1, linux/amd64)
```
The `krew` toolchain is built and versioned in its own repository (`krewire/krew`).

---

## 2. Semantic Versioning & Compatibility Contract

Krewire adheres strictly to [Semantic Versioning 2.0.0](https://semver.org/):

- **Pre-1.0.0 Phase (`0.y.z`)**:
  - `0.y.0` (Minor): Introduces additive features, new domain packages, or minor behavioral improvements. Breaking changes require migration notes in `docs/overview/upgrade-guide.md`.
  - `0.y.z` (Patch): Backwards-compatible bug fixes, security patches, and performance optimizations.
  - Automated compatibility checking is enforced by `version.IsCompatible()` (for `0.y.z`, minor versions must match) and the ecosystem [Cross-Repository Compatibility Matrix](./cross-repo-compatibility-matrix.md).
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

This repository operates these automated pipelines:
- **CI (`ci.yaml`)**: The quality-gate contract on every pull request, push to `main`, and merge-queue entry; reused verbatim by the release and nightly pipelines (see [Testing & QA §6](../guides/testing-and-qa.md#6-mandatory-quality-gates)).
- **Release Pipeline (`release.yaml`)**: Multi-track tag classification, then CI, then the shared artifact workflow (§6).
- **Nightly Pipeline (`nightly.yaml`)**: Daily build from `main` through the same CI and artifact workflow, channel `nightly`.
- **Release Artifacts (`release-artifacts.yaml`)**: Reusable build → attest → smoke → publish → verify path shared by release and nightly (§6).
- **Hotfix Pipeline**: Not yet implemented in this repository (no `hotfix.yaml`); hotfixes follow the branch protocol in `AGENTS.md`.
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
   go test -count=1 -v . ./pkg/errs ./pkg/model
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
6. **Bitwise Reproducibility Gate** (CI job `release-snapshot`):
   ```bash
   .github/scripts/build-release.sh v0.0.0-ci dev /tmp/a
   .github/scripts/build-release.sh v0.0.0-ci dev /tmp/b
   diff /tmp/a/SHA256SUMS /tmp/b/SHA256SUMS
   ```
   *Verifies consecutive clean builds produce bitwise identical binaries and SBOMs.*
7. **Cross-Repository Release Qualification Gate**:
   ```bash
   krew release qualify
   ```
   *Validates supported repository pairs, version constraints, and contract tests (see [Compatibility Matrix](./cross-repo-compatibility-matrix.md)).*

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
4. **Push the tag**: `git push origin v0.2.0`. `release.yaml` runs the CI gates, then builds, attests, smoke-tests, and publishes the artifacts (§6). Never build release binaries by hand.

---

## 6. Release Artifacts, Checksums, and Provenance

Every stable, pre-release, and nightly release goes through `.github/workflows/release-artifacts.yaml`; the pipelines differ only by version and channel.

**Support matrix** (declared in `.github/scripts/build-release.sh`): `linux/amd64`, `linux/arm64`, `darwin/amd64`, `darwin/arm64`, `windows/amd64`, `windows/arm64`.

**Executables**: the core module ships one executable, `ai` (`cmd/ai`). The unified developer toolchain `krew` is distributed from `krewire/krew`; the release manifest records it with `provisioning_mode: separate-distribution` until the adapter contract (#131, #140) is versioned.

**Reproducible build** — `.github/scripts/build-release.sh <version> <channel> [dir]`:
- `CGO_ENABLED=0`, `-trimpath`, `-buildvcs=true`, `-ldflags "-s -w -buildid="`, `GOFLAGS=-mod=readonly`, `GOWORK=off`.
- `SOURCE_DATE_EPOCH` defaults to the commit time; the build date stamped into the binary and the SBOM timestamp derive from it.
- Refuses to run from a dirty working tree.

**Published assets** (per release):

| Asset | Purpose |
|---|---|
| `ai_<version>_<os>_<arch>[.exe]` | Binary; `ai --version` prints version, channel, and commit. |
| `ai_<version>_<os>_<arch>.cdx.json` | CycloneDX SBOM generated from the binary's embedded module list. |
| `release-manifest.json` | Version, channel, commit, Go version, build flags, targets, toolchain provisioning. |
| `SHA256SUMS` | SHA-256 of every asset above. |
| `provenance.sigstore.json` | Signed GitHub artifact attestation (SLSA build provenance) covering every file in `SHA256SUMS`. |

**Gates before publishing**: all CI gates; every target builds; on native runners for each platform the checksum verifies and `--version` reports the exact version, channel, and commit, and `--help` works. After publishing, `verify-published` downloads the release, re-checks `SHA256SUMS` and every attestation, and runs the binary.

**Verifying a download**:
```bash
sha256sum -c --ignore-missing SHA256SUMS
gh attestation verify ai_<version>_linux_amd64 --repo krewire/krewire
```

**Maintainer settings required**: Actions must be allowed to create attestations (`id-token: write`, `attestations: write` are granted only to the artifacts job), the `GITHUB_TOKEN` must be allowed to create releases (`contents: write`), and runners `ubuntu-24.04-arm`, `macos-15-intel`, and `windows-11-arm` must be available (they are for public repositories).
