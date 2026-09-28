#!/usr/bin/env sh
# ─────────────────────────────────────────────────────────────────────────────
# install.sh — Krewire kiw installer
#
# Usage:
#   curl -fsSL https://krewire.com/scripts/install.sh | sh
#
# What it does:
#   1. Detects OS and CPU architecture
#   2. Downloads the correct kiw binary from GitHub Releases
#   3. Installs to /usr/local/bin/kiw (or ~/.local/bin/kiw if no sudo)
#   4. Verifies installation
#
# Supported:
#   Linux   — amd64, arm64
#   macOS   — amd64 (Intel), arm64 (Apple Silicon)
# ─────────────────────────────────────────────────────────────────────────────
set -eu

REPO="krewire/kiw"
BINARY="kiw"
INSTALL_DIR="/usr/local/bin"
FALLBACK_DIR="${HOME}/.local/bin"

# ── Colours ──────────────────────────────────────────────────────────────────
RED='\033[0;31m'
GRN='\033[0;32m'
YLW='\033[0;33m'
BLD='\033[1m'
RST='\033[0m'

info()    { printf "${BLD}  →${RST}  %s\n" "$*"; }
success() { printf "${GRN}${BLD}  ✓${RST}  %s\n" "$*"; }
warn()    { printf "${YLW}${BLD}  ⚠${RST}  %s\n" "$*"; }
die()     { printf "${RED}${BLD}  ✗${RST}  %s\n" "$*" >&2; exit 1; }

# ── Banner ───────────────────────────────────────────────────────────────────
printf "\n"
printf "${BLD}  Krewire kiw Installer${RST}\n"
printf "  One Go Framework for Every Workload\n"
printf "  https://krewire.com\n\n"

# ── Detect OS ────────────────────────────────────────────────────────────────
OS="$(uname -s)"
case "${OS}" in
  Linux*)  OS="linux"  ;;
  Darwin*) OS="darwin" ;;
  *)       die "Unsupported OS: ${OS}. Please install manually from https://github.com/${REPO}/releases" ;;
esac

# ── Detect Arch ──────────────────────────────────────────────────────────────
ARCH="$(uname -m)"
case "${ARCH}" in
  x86_64|amd64) ARCH="amd64"  ;;
  aarch64|arm64) ARCH="arm64" ;;
  *)             die "Unsupported architecture: ${ARCH}. Please install manually from https://github.com/${REPO}/releases" ;;
esac

info "Detected: ${OS}/${ARCH}"

# ── Fetch latest version ──────────────────────────────────────────────────────
info "Fetching latest kiw version..."

if command -v curl >/dev/null 2>&1; then
  FETCH="curl -fsSL"
elif command -v wget >/dev/null 2>&1; then
  FETCH="wget -qO-"
else
  die "curl or wget is required. Please install one and try again."
fi

LATEST=$(${FETCH} "https://api.github.com/repos/${REPO}/releases/latest" \
  | grep '"tag_name"' \
  | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/')

if [ -z "${LATEST}" ]; then
  # Fallback: use go install if API is unavailable or no releases yet
  warn "Could not fetch latest release. Falling back to: go install"
  printf "\n"
  if command -v go >/dev/null 2>&1; then
    info "Installing via go install github.com/${REPO}/cmd/kiw@latest ..."
    go install "github.com/${REPO}/cmd/kiw@latest"
    GOPATH="${GOPATH:-$(go env GOPATH)}"
    if echo "${PATH}" | grep -q "${GOPATH}/bin"; then
      success "kiw installed via go install"
    else
      warn "Add ${GOPATH}/bin to your PATH:"
      printf "\n    export PATH=\"\$PATH:${GOPATH}/bin\"\n\n"
    fi
  else
    die "Go is not installed. Install Go 1.22+ from https://go.dev/dl/ then re-run this script, or install kiw manually from https://github.com/${REPO}/releases"
  fi
  exit 0
fi

info "Latest version: ${LATEST}"

# ── Build download URL ────────────────────────────────────────────────────────
TARBALL="${BINARY}_${OS}_${ARCH}.tar.gz"
DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${LATEST}/${TARBALL}"

# ── Download ──────────────────────────────────────────────────────────────────
TMP_DIR="$(mktemp -d)"
trap 'rm -rf "${TMP_DIR}"' EXIT

info "Downloading ${DOWNLOAD_URL} ..."
if command -v curl >/dev/null 2>&1; then
  curl -fsSL "${DOWNLOAD_URL}" -o "${TMP_DIR}/${TARBALL}"
else
  wget -qO "${TMP_DIR}/${TARBALL}" "${DOWNLOAD_URL}"
fi

# ── Extract ───────────────────────────────────────────────────────────────────
info "Extracting..."
tar -xzf "${TMP_DIR}/${TARBALL}" -C "${TMP_DIR}"

# ── Install ───────────────────────────────────────────────────────────────────
BINARY_PATH="${TMP_DIR}/${BINARY}"
if [ ! -f "${BINARY_PATH}" ]; then
  # Some releases put binaries in a subdirectory
  BINARY_PATH="$(find "${TMP_DIR}" -type f -name "${BINARY}" | head -1)"
fi
[ -f "${BINARY_PATH}" ] || die "Binary '${BINARY}' not found in release archive."

chmod +x "${BINARY_PATH}"

if [ -w "${INSTALL_DIR}" ] || command -v sudo >/dev/null 2>&1; then
  if [ -w "${INSTALL_DIR}" ]; then
    mv "${BINARY_PATH}" "${INSTALL_DIR}/${BINARY}"
    INSTALLED_AT="${INSTALL_DIR}/${BINARY}"
  else
    info "sudo required to install to ${INSTALL_DIR}..."
    sudo mv "${BINARY_PATH}" "${INSTALL_DIR}/${BINARY}"
    INSTALLED_AT="${INSTALL_DIR}/${BINARY}"
  fi
else
  # No sudo — install to ~/.local/bin
  mkdir -p "${FALLBACK_DIR}"
  mv "${BINARY_PATH}" "${FALLBACK_DIR}/${BINARY}"
  INSTALLED_AT="${FALLBACK_DIR}/${BINARY}"
  warn "Installed to ${FALLBACK_DIR} (no sudo available)."
  if ! echo "${PATH}" | grep -q "${FALLBACK_DIR}"; then
    warn "Add to your PATH (add to ~/.bashrc or ~/.zshrc):"
    printf "\n    export PATH=\"\$PATH:${FALLBACK_DIR}\"\n\n"
  fi
fi

# ── Verify ────────────────────────────────────────────────────────────────────
if command -v "${BINARY}" >/dev/null 2>&1; then
  VERSION_OUT="$("${BINARY}" version 2>/dev/null || "${BINARY}" --version 2>/dev/null || echo "${LATEST}")"
  success "kiw ${VERSION_OUT} installed at ${INSTALLED_AT}"
else
  success "kiw installed at ${INSTALLED_AT}"
  warn "Make sure ${INSTALLED_AT%/*} is in your PATH."
fi

# ── Next steps ────────────────────────────────────────────────────────────────
printf "\n"
printf "${BLD}  Get started:${RST}\n\n"
printf "    kiw new my-app --site   # scaffold a new site\n"
printf "    cd my-app\n"
printf "    kiw dev                 # start dev server\n"
printf "    kiw build               # build for production\n"
printf "\n"
printf "  Docs → https://krewire.com/docs/getting-started\n\n"
