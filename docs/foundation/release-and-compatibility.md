# Release, Versioning, and Compatibility Specification

This specification documents the canonical release procedure, semantic compatibility expectations, and mandatory quality gates for the Krewire core repository and ecosystem.

---

## 1. Canonical Version Single Source of Truth (SSOT)

The definitive, single source of truth for the core ecosystem version is declared in:
- [`version/version.go`](../../version/version.go) via `version.CurrentVersion = MustParseVersion("0.2.0")`.

As established in ADR-E8P9D and the pure-module monorepo reorganization, `krewire` is a pure domain library monorepo (35 modular packages). Standalone executables are distributed from `krewire/krew` (canonical developer CLI) and `krewire/agent` (autonomous AI agent runtime). Core releases publish reproducible multi-module source bundles (`krewire-${VERSION}-source.tar.gz`), machine-readable release manifests (`release-manifest.json`), checksums (`SHA256SUMS`), and signed SLSA build provenance attestations (`provenance.sigstore.json`).

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

## 3. The Six Version Release Tracks & Cadence Model (SemVer 2.0.0)

Every module in the ecosystem releases across six designated tracks strictly adhering to **Semantic Versioning 2.0.0** (`vMAJOR.MINOR.PATCH[-PRERELEASE]`) ([ADR-007](../../../internal/docs/adr/ADR-007-versioning-release-and-deployment-pipelines.md), [Release Cadence Roadmap](../../../internal/docs/roadmap/release-and-deployment-cadence.md)):

1. **`lts` (Long-Term Support)**: Extended enterprise lifecycle published once every two years on **even-numbered calendar years** (e.g., `v2.0.0-lts` in 2028, `v4.0.0-lts` in 2030) with a 5-year active support window (`vX.0.0-lts` or `lts/vX.0` branch). Strict bug fixes and security patches only; zero breaking API changes.
2. **`stable` (General Availability)**: Production baseline published on two synchronized cadences:
   - **Quarterly Stable Releases** (`vX.Y.0`): Minor version increments tagged from `main` at the conclusion of each quarter once all quarterly engineering phases pass 100% of quality gates, advancing additively based on the authentic scope of backwards-compatible capabilities delivered.
   - **Annual Stable Releases** (`vX.0.0`): Authoritative major version increments tagged at the close of Q4 (December).
3. **`pre-release` (Alpha, Beta, RC)**: Formal qualification builds published during the pre-release window preceding stable GA releases (`vX.Y.0-alpha.N`, `vX.Y.0-beta.N`, `vX.Y.0-rc.N`).
4. **`canary` (`vX.Y.Z-canary.N`)**: Automated canary checkpoint builds published **immediately upon completing each individual quarterly engineering phase** (`vX.Y.0-canary.0` for Phase 0, `vX.Y.0-canary.1` for Phase 1, ..., `vX.Y.0-canary.N`) for automated canary deployment and immediate workspace validation.
5. **`nightly` (Daily Automated Builds)**: Bleeding-edge automated builds (`vX.Y.Z-nightly.YYYYMMDD`), published daily from `main`.
6. **`development` (`dev`)**: Local, in-progress workspace builds (`vX.Y.Z-dev.<sha>`).

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
   go test -count=1 -v . ./modules/errs ./modules/model
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

1. **Bump Version**: Update `version.CurrentVersion` in `version/version.go`.
2. **Run Verification**:
   ```bash
   go test -v ./...
   ```
3. **Commit & Tag**:
   ```bash
   git add version/version.go CHANGELOG.md
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
