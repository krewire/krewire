#!/usr/bin/env bash
# Render release notes for a built dist directory.
# Usage: release-notes.sh <version> <channel> <dist-dir>
set -euo pipefail

VERSION="${1:?usage: release-notes.sh <version> <channel> <dist-dir>}"
CHANNEL="${2:?usage: release-notes.sh <version> <channel> <dist-dir>}"
DIST="${3:?usage: release-notes.sh <version> <channel> <dist-dir>}"
REPO="${GITHUB_REPOSITORY:-krewire/krewire}"
COMMIT="$(sed -n 's/.*"commit": "\([0-9a-f]*\)".*/\1/p' "${DIST}/release-manifest.json")"
GO_VERSION="$(sed -n 's/.*"go_version": "\([^"]*\)".*/\1/p' "${DIST}/release-manifest.json")"
BASE="https://github.com/${REPO}/releases/download/${VERSION}"

cat <<EOF
## Krewire Core — ${VERSION}

| | |
|---|---|
| **Release channel** | \`${CHANNEL}\` |
| **Source commit** | \`${COMMIT}\` |
| **Go toolchain** | \`${GO_VERSION}\`, \`CGO_ENABLED=0\`, \`-trimpath\` (reproducible) |

EOF

if ls "${DIST}"/ai_* >/dev/null 2>&1; then
cat <<EOF
This release publishes the Go library module \`github.com/krewire/krewire\` and
the \`ai\` command (\`cmd/ai\`), the only executable the core module ships.

### Supported platforms

| OS | Architecture | Asset |
|---|---|---|
EOF

for f in "${DIST}"/ai_*; do
  name="$(basename "${f}")"
  case "${name}" in *.cdx.json) continue ;; esac
  target="${name#ai_"${VERSION}"_}"
  target="${target%.exe}"
  # shellcheck disable=SC2016 # backticks are Markdown, not command substitution
  printf '| %s | %s | `%s` |\n' "${target%_*}" "${target##*_}" "${name}"
done

cat <<EOF

### Install

Library:

\`\`\`bash
go get github.com/krewire/krewire@${VERSION}
\`\`\`

\`ai\` binary (Linux/macOS; pick your platform from the table above):

\`\`\`bash
curl -fsSLO ${BASE}/ai_${VERSION}_linux_amd64
curl -fsSLO ${BASE}/SHA256SUMS
sha256sum -c --ignore-missing SHA256SUMS     # macOS: shasum -a 256 -c --ignore-missing SHA256SUMS
install -m 0755 ai_${VERSION}_linux_amd64 /usr/local/bin/ai
ai --version
\`\`\`

Windows: download \`ai_${VERSION}_windows_<arch>.exe\`, compare
\`Get-FileHash -Algorithm SHA256\` with \`SHA256SUMS\`, and rename it to \`ai.exe\`.

### Upgrade

Replace the existing \`ai\` binary with the new one and confirm with
\`ai --version\`, which prints the release version, channel, and source commit.
Library consumers bump the module with \`go get github.com/krewire/krewire@${VERSION}\`
and run \`go mod tidy\`.

### Verify

- **Checksums**: \`SHA256SUMS\` covers every asset.
- **Provenance**: each asset has a GitHub artifact attestation (SLSA build
  provenance); \`provenance.sigstore.json\` is the signed bundle.
  \`gh attestation verify ai_${VERSION}_linux_amd64 --repo ${REPO}\`
- **SBOM**: \`ai_${VERSION}_<os>_<arch>.cdx.json\` (CycloneDX) per binary.
- **Manifest**: \`release-manifest.json\` records version, channel, commit,
  toolchain, build flags, and targets.
- **Reproduce**: check out \`${COMMIT}\` and run
  \`.github/scripts/build-release.sh ${VERSION} ${CHANNEL} dist\` with Go
  \`${GO_VERSION}\`; the checksums must match.
EOF
else
cat <<EOF
This release publishes the reproducible multi-module pure library source bundle
for the Krewire foundational ecosystem (\`github.com/krewire/krewire\`).

### Published Assets

- \`krewire-${VERSION}-source.tar.gz\`: Full reproducible multi-module library source bundle.
- \`release-manifest.json\`: Machine-readable metadata of constituent modules, toolchain, and commit.
- \`SHA256SUMS\`: Cryptographic SHA-256 checksums covering all published assets.
- \`provenance.sigstore.json\`: Cryptographically signed SLSA build provenance attestation.

### Consume Modules

Import constituent modules in your \`go.mod\`:

\`\`\`bash
go get github.com/krewire/krewire/model@${VERSION}
go get github.com/krewire/krewire/web@${VERSION}
\`\`\`

### Verify

- **Checksums**: \`SHA256SUMS\` covers every asset.
- **Provenance**: \`gh attestation verify krewire-${VERSION}-source.tar.gz --repo ${REPO}\`
- **Manifest**: \`release-manifest.json\` records version, channel, commit, toolchain, and module inventory.
- **Reproduce**: check out \`${COMMIT}\` and run
  \`.github/scripts/build-release.sh ${VERSION} ${CHANNEL} dist\` with Go
  \`${GO_VERSION}\`; the checksums must match bit-for-bit.
EOF
fi

cat <<EOF

### Known limitations

- The unified developer toolchain \`krew\` is released from its own repository
  (\`krewire/krew\`) and is not bundled here; the toolchain-to-engine adapter
  contract is tracked in krewire/krewire#131 and #140.
- Library modules are pure Go and standard-library based; external database
  drivers enter through \`krewire/hub\`.
EOF
