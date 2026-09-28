#!/usr/bin/env sh
# ─────────────────────────────────────────────────────────────────────────────
# install.sh — Krewire kiw installer
#
# Usage:
#   curl -fsSL https://krewire.com/scripts/install.sh | sh
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

# ── Colors ───────────────────────────────────────────────────────────────────
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

# ── Fetch tool ───────────────────────────────────────────────────────────────
if command -v curl >/dev/null 2>&1; then
  FETCH="curl -fsSL"
elif command -v wget >/dev/null 2>&1; then
  FETCH="wget -qO-"
else
  die "curl or wget is required. Please install one and try again."
fi

# ── Detect OS & Arch ─────────────────────────────────────────────────────────
OS="$(uname -s)"
case "${OS}" in
  Linux*)  OS="linux"  ;;
  Darwin*) OS="darwin" ;;
  *)       die "Unsupported OS: ${OS}. Please install manually from https://github.com/${REPO}" ;;
esac

ARCH="$(uname -m)"
case "${ARCH}" in
  x86_64|amd64)  ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *)             die "Unsupported architecture: ${ARCH}. Please install manually from https://github.com/${REPO}" ;;
esac

info "Platform detected: ${OS}/${ARCH}"

# ── Query latest release tag ─────────────────────────────────────────────────
LATEST=""
if [ -n "${FETCH}" ]; then
  LATEST=$(${FETCH} "https://api.github.com/repos/${REPO}/releases/latest" 2>/dev/null \
    | grep '"tag_name"' \
    | head -1 \
    | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/' || true)
fi

TARGET_TAG="${LATEST:-latest}"
if [ -n "${LATEST}" ]; then
  info "Latest release version: ${LATEST}"
fi

# ── Check for precompiled binary asset in GitHub Release ─────────────────────
TARBALL="${BINARY}_${OS}_${ARCH}.tar.gz"
DOWNLOAD_URL=""
ASSET_EXISTS=false

if [ -n "${LATEST}" ]; then
  DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${LATEST}/${TARBALL}"
  # Check if asset actually exists (avoid 404 failure)
  if command -v curl >/dev/null 2>&1; then
    HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -I "${DOWNLOAD_URL}" 2>/dev/null || true)
    if [ "${HTTP_CODE}" = "200" ] || [ "${HTTP_CODE}" = "302" ]; then
      ASSET_EXISTS=true
    fi
  fi
fi

INSTALLED_AT=""

if [ "${ASSET_EXISTS}" = "true" ]; then
  # ── Path A: Precompiled binary release available ───────────────────────────
  info "Downloading ${DOWNLOAD_URL} ..."
  TMP_DIR="$(mktemp -d)"
  trap 'rm -rf "${TMP_DIR}"' EXIT

  if command -v curl >/dev/null 2>&1; then
    curl -fsSL "${DOWNLOAD_URL}" -o "${TMP_DIR}/${TARBALL}"
  else
    wget -qO "${TMP_DIR}/${TARBALL}" "${DOWNLOAD_URL}"
  fi

  info "Extracting..."
  tar -xzf "${TMP_DIR}/${TARBALL}" -C "${TMP_DIR}"

  BINARY_PATH="${TMP_DIR}/${BINARY}"
  if [ ! -f "${BINARY_PATH}" ]; then
    BINARY_PATH="$(find "${TMP_DIR}" -type f -name "${BINARY}" | head -1)"
  fi
  [ -f "${BINARY_PATH}" ] || die "Binary '${BINARY}' not found in release archive."

  chmod +x "${BINARY_PATH}"

  if [ -w "${INSTALL_DIR}" ]; then
    mv "${BINARY_PATH}" "${INSTALL_DIR}/${BINARY}"
    INSTALLED_AT="${INSTALL_DIR}/${BINARY}"
  elif command -v sudo >/dev/null 2>&1; then
    info "Installing to ${INSTALL_DIR} via sudo..."
    sudo mv "${BINARY_PATH}" "${INSTALL_DIR}/${BINARY}"
    INSTALLED_AT="${INSTALL_DIR}/${BINARY}"
  else
    mkdir -p "${FALLBACK_DIR}"
    mv "${BINARY_PATH}" "${FALLBACK_DIR}/${BINARY}"
    INSTALLED_AT="${FALLBACK_DIR}/${BINARY}"
  fi

else
  # ── Path B: Build & install via Go toolchain ───────────────────────────────
  if ! command -v go >/dev/null 2>&1; then
    printf "\n"
    warn "Precompiled release binary not yet published for ${OS}/${ARCH}."
    die "Go 1.22+ is required to install from source. Please install Go from https://go.dev/dl/ and run this script again."
  fi

  GO_VER="$(go version 2>/dev/null || true)"
  info "Found ${GO_VER}"
  info "Installing kiw via: go install github.com/${REPO}/cmd/kiw@${TARGET_TAG} ..."

  go install "github.com/${REPO}/cmd/kiw@${TARGET_TAG}"

  GOPATH_BIN="$(go env GOPATH)/bin"
  GOBIN_DIR="$(go env GOBIN)"
  SRC_BIN=""

  if [ -n "${GOBIN_DIR}" ] && [ -f "${GOBIN_DIR}/${BINARY}" ]; then
    SRC_BIN="${GOBIN_DIR}/${BINARY}"
  elif [ -f "${GOPATH_BIN}/${BINARY}" ]; then
    SRC_BIN="${GOPATH_BIN}/${BINARY}"
  elif command -v "${BINARY}" >/dev/null 2>&1; then
    SRC_BIN="$(command -v "${BINARY}")"
  else
    die "go install succeeded but binary '${BINARY}' was not found in ${GOPATH_BIN}."
  fi

  # Attempt to symlink/copy to /usr/local/bin for global convenience
  if [ -w "${INSTALL_DIR}" ]; then
    cp "${SRC_BIN}" "${INSTALL_DIR}/${BINARY}"
    INSTALLED_AT="${INSTALL_DIR}/${BINARY}"
  elif command -v sudo >/dev/null 2>&1 && [ -t 0 ]; then
    info "Linking to ${INSTALL_DIR} for global PATH access..."
    sudo cp "${SRC_BIN}" "${INSTALL_DIR}/${BINARY}" 2>/dev/null || true
    if [ -f "${INSTALL_DIR}/${BINARY}" ]; then
      INSTALLED_AT="${INSTALL_DIR}/${BINARY}"
    else
      INSTALLED_AT="${SRC_BIN}"
    fi
  else
    INSTALLED_AT="${SRC_BIN}"
  fi
fi

# ── Verification ─────────────────────────────────────────────────────────────
printf "\n"
if command -v "${BINARY}" >/dev/null 2>&1; then
  ACTIVE_BIN="$(command -v "${BINARY}")"
  VER_TXT="$("${ACTIVE_BIN}" version 2>/dev/null || echo "ok")"
  success "kiw successfully installed!"
  info "Binary: ${ACTIVE_BIN}"
  info "Version: ${VER_TXT}"
elif [ -n "${INSTALLED_AT}" ] && [ -f "${INSTALLED_AT}" ]; then
  success "kiw binary installed at ${INSTALLED_AT}"
  BIN_DIR="$(dirname "${INSTALLED_AT}")"
  warn "Please ensure '${BIN_DIR}' is in your PATH. Add to ~/.bashrc or ~/.zshrc:"
  printf "\n    export PATH=\"\$PATH:${BIN_DIR}\"\n\n"
else
  die "Could not verify '${BINARY}' installation."
fi

# ── Next steps ───────────────────────────────────────────────────────────────
printf "\n"
printf "${BLD}  Get started in 30 seconds:${RST}\n\n"
printf "    kiw new my-app --site   # scaffold a new static site\n"
printf "    cd my-app\n"
printf "    kiw dev                 # start hot-reload dev server\n"
printf "    kiw build               # build static output to .krewire/build\n"
printf "\n"
printf "  Documentation → https://krewire.com/docs/getting-started\n\n"
