#!/usr/bin/env bash
# Reproducible release build for the executables shipped by the core module.
#
# Usage: build-release.sh <version> <channel> [out-dir]
#   version  release version, e.g. v0.2.0 or v0.3.0-nightly.20261123
#   channel  stable | lts | rc | beta | alpha | nightly | experimental | dev
#   out-dir  output directory (default: dist)
#
# Environment:
#   TARGETS            space-separated GOOS/GOARCH list (default: the support matrix)
#   SOURCE_DATE_EPOCH  build timestamp; defaults to the commit time of HEAD
#   SBOM               1 (default) writes a CycloneDX SBOM per binary; 0 skips it
#
# The same script is used by the stable and nightly pipelines and can be run
# locally; two runs on the same commit produce bit-for-bit identical binaries.
set -euo pipefail

VERSION="${1:?usage: build-release.sh <version> <channel> [out-dir]}"
CHANNEL="${2:?usage: build-release.sh <version> <channel> [out-dir]}"
OUT="${3:-dist}"

# Support matrix (keep in sync with docs/foundation/release-and-compatibility.md).
DEFAULT_TARGETS="linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64"
TARGETS="${TARGETS:-$DEFAULT_TARGETS}"

# Executables shipped by this module: <binary name>=<main package>.
BINARIES="ai=./cmd/ai"

PKG="github.com/krewire/krewire/version"
SBOM="${SBOM:-1}"
CYCLONEDX_GOMOD="github.com/CycloneDX/cyclonedx-gomod/cmd/cyclonedx-gomod@v1.12.0"

if ! git diff --quiet HEAD -- || [ -n "$(git status --porcelain --untracked-files=normal)" ]; then
  echo "error: working tree is not clean; release builds must come from a committed tree" >&2
  git status --short >&2
  exit 1
fi

COMMIT="$(git rev-parse HEAD)"
export SOURCE_DATE_EPOCH="${SOURCE_DATE_EPOCH:-$(git log -1 --format=%ct HEAD)}"
if date -u -d "@0" >/dev/null 2>&1; then
  BUILD_DATE="$(date -u -d "@${SOURCE_DATE_EPOCH}" +%Y-%m-%dT%H:%M:%SZ)"
else
  BUILD_DATE="$(date -u -r "${SOURCE_DATE_EPOCH}" +%Y-%m-%dT%H:%M:%SZ)"
fi

sha256() {
  if command -v sha256sum >/dev/null 2>&1; then sha256sum "$@"; else shasum -a 256 "$@"; fi
}

LDFLAGS="-s -w -buildid=
 -X ${PKG}.buildVersion=${VERSION}
 -X ${PKG}.buildCommit=${COMMIT}
 -X ${PKG}.buildDate=${BUILD_DATE}
 -X ${PKG}.buildChannel=${CHANNEL}"

mkdir -p "${OUT}"

if [ "${SBOM}" = "1" ]; then
  TOOLS="$(mktemp -d)"
  trap 'rm -rf "${TOOLS}"' EXIT
  GOBIN="${TOOLS}" GOFLAGS="" CGO_ENABLED=0 go install "${CYCLONEDX_GOMOD}"
fi

export CGO_ENABLED=0 GOFLAGS="-mod=readonly"

for bin in ${BINARIES}; do
  name="${bin%%=*}"
  main="${bin#*=}"
  for target in ${TARGETS}; do
    goos="${target%/*}"
    goarch="${target#*/}"
    ext=""
    [ "${goos}" = "windows" ] && ext=".exe"
    file="${OUT}/${name}_${VERSION}_${goos}_${goarch}${ext}"
    echo "building ${file}"
    GOOS="${goos}" GOARCH="${goarch}" go build \
      -trimpath -buildvcs=true \
      -ldflags "${LDFLAGS}" \
      -o "${file}" "${main}"
    if [ "${SBOM}" = "1" ]; then
      "${TOOLS}/cyclonedx-gomod" bin -json -noserial -version "${VERSION}" \
        -output "${file%.exe}.cdx.json" "${file}"
      # Pin the SBOM timestamp to SOURCE_DATE_EPOCH so it is reproducible too.
      sed -E "s/\"timestamp\": \"[^\"]*\"/\"timestamp\": \"${BUILD_DATE}\"/" \
        "${file%.exe}.cdx.json" > "${file%.exe}.cdx.tmp"
      mv "${file%.exe}.cdx.tmp" "${file%.exe}.cdx.json"
    fi
  done
done

GO_VERSION="$(go env GOVERSION)"

# Release manifest: machine-readable record of what was built and how.
{
  printf '{\n'
  printf '  "schema": "krewire.release-manifest/v1",\n'
  printf '  "module": "github.com/krewire/krewire",\n'
  printf '  "version": "%s",\n' "${VERSION}"
  printf '  "channel": "%s",\n' "${CHANNEL}"
  printf '  "commit": "%s",\n' "${COMMIT}"
  printf '  "source_date_epoch": %s,\n' "${SOURCE_DATE_EPOCH}"
  printf '  "build_date": "%s",\n' "${BUILD_DATE}"
  printf '  "go_version": "%s",\n' "${GO_VERSION}"
  printf '  "build_flags": "CGO_ENABLED=0 -trimpath -buildvcs=true -ldflags=-s -w -buildid=",\n'
  printf '  "targets": ['
  sep=""
  for target in ${TARGETS}; do printf '%s"%s"' "${sep}" "${target}"; sep=", "; done
  printf '],\n'
  printf '  "executables": ['
  sep=""
  for bin in ${BINARIES}; do printf '%s"%s"' "${sep}" "${bin%%=*}"; sep=", "; done
  printf '],\n'
  printf '  "toolchain": {\n'
  printf '    "name": "krew",\n'
  printf '    "repository": "github.com/krewire/krew",\n'
  printf '    "provisioning_mode": "separate-distribution",\n'
  printf '    "adapter_contract": "unversioned",\n'
  printf '    "note": "krew is released from its own repository; the toolchain-to-engine adapter contract is tracked in krewire/krewire#131 and #140"\n'
  printf '  }\n'
  printf '}\n'
} > "${OUT}/release-manifest.json"

# Checksums over every published file except the checksum file itself.
(
  cd "${OUT}"
  sums="$(mktemp)"
  find . -maxdepth 1 -type f ! -name SHA256SUMS -print \
    | sed 's|^\./||' | LC_ALL=C sort | while read -r f; do sha256 "$f"; done > "${sums}"
  mv "${sums}" SHA256SUMS
)

echo "artifacts in ${OUT}:"
cat "${OUT}/SHA256SUMS"
