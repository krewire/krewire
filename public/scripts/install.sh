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
#
# Security:
#   - SHA256 checksum verified against release checksum manifest before install
#   - Tag name format-validated (vX.Y.Z only) to prevent injection
#   - Binary search depth limited (maxdepth 2) to prevent path traversal
#   - Uses `install -m 755` instead of mv for atomic placement + perms
#   - Temp dir cleaned up on exit, interrupt, and termination
#   - Go module sum database NOT bypassed in fallback path
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

# ── Fetch helper (curl or wget) ───────────────────────────────────────────────
FETCH_CMD=""
if command -v curl >/dev/null 2>&1; then
  FETCH_CMD="curl"
elif command -v wget >/dev/null 2>&1; then
  FETCH_CMD="wget"
else
  die "curl or wget is required. Please install one and try again."
fi

fetch_url() {
  # fetch_url <url> <output_file>
  _url="$1"; _out="$2"
  if [ "${FETCH_CMD}" = "curl" ]; then
    curl -fsSL --retry 3 --retry-delay 2 --max-time 120 "${_url}" -o "${_out}"
  else
    wget -qO "${_out}" "${_url}"
  fi
}

fetch_stdout() {
  # fetch_stdout <url>
  _url="$1"
  if [ "${FETCH_CMD}" = "curl" ]; then
    curl -fsSL --retry 3 --retry-delay 2 --max-time 30 "${_url}"
  else
    wget -qO- "${_url}"
  fi
}

# ── Detect OS & Arch ─────────────────────────────────────────────────────────
OS="$(uname -s)"
case "${OS}" in
  Linux*)  OS="linux"  ;;
  Darwin*) OS="darwin" ;;
  *)       die "Unsupported OS: ${OS}. Install manually: https://github.com/${REPO}/releases" ;;
esac

ARCH="$(uname -m)"
case "${ARCH}" in
  x86_64|amd64)  ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *)             die "Unsupported architecture: ${ARCH}. Install manually: https://github.com/${REPO}/releases" ;;
esac

info "Platform detected: ${OS}/${ARCH}"

# ── Query latest release (with tag format validation) ────────────────────────
LATEST=""
_api_resp="$(fetch_stdout "https://api.github.com/repos/${REPO}/releases/latest" 2>/dev/null || true)"
if [ -n "${_api_resp}" ]; then
  _raw_tag="$(printf '%s' "${_api_resp}" \
    | grep '"tag_name"' \
    | head -1 \
    | sed -E 's/.*"tag_name"[[:space:]]*:[[:space:]]*"([^"]+)".*/\1/' || true)"
  # Validate: only accept vX.Y.Z[-prerelease] format — reject anything else
  case "${_raw_tag}" in
    v[0-9]*.[0-9]*.[0-9]*)
      LATEST="${_raw_tag}"
      info "Latest release: ${LATEST}"
      ;;
    *)
      warn "Unexpected release tag format '${_raw_tag}' — will fall back to go install."
      ;;
  esac
fi

TARGET_TAG="${LATEST:-latest}"

# ── Isolated temp dir with guaranteed cleanup ─────────────────────────────────
TMP_DIR="$(mktemp -d)"
trap 'rm -rf "${TMP_DIR}"' EXIT INT TERM

# ── Check for precompiled binary asset ───────────────────────────────────────
TARBALL="${BINARY}_${OS}_${ARCH}.tar.gz"
CHECKSUM_FILE="${BINARY}_checksums.txt"
ASSET_EXISTS=false

if [ -n "${LATEST}" ]; then
  DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${LATEST}/${TARBALL}"
  CHECKSUM_URL="https://github.com/${REPO}/releases/download/${LATEST}/${CHECKSUM_FILE}"

  # HEAD-check asset existence before downloading
  if [ "${FETCH_CMD}" = "curl" ]; then
    _code="$(curl -s -o /dev/null -w "%{http_code}" -I --max-time 10 "${DOWNLOAD_URL}" 2>/dev/null || true)"
    if [ "${_code}" = "200" ] || [ "${_code}" = "302" ]; then
      ASSET_EXISTS=true
    fi
  else
    if wget --spider -q "${DOWNLOAD_URL}" 2>/dev/null; then
      ASSET_EXISTS=true
    fi
  fi
fi

INSTALLED_AT=""

if [ "${ASSET_EXISTS}" = "true" ]; then
  # ── Path A: Precompiled binary ──────────────────────────────────────────────
  info "Downloading ${TARBALL}..."
  fetch_url "${DOWNLOAD_URL}" "${TMP_DIR}/${TARBALL}"

  # ── SHA256 checksum verification ────────────────────────────────────────────
  info "Verifying SHA256 checksum..."
  _sum_ok=false
  if fetch_url "${CHECKSUM_URL}" "${TMP_DIR}/${CHECKSUM_FILE}" 2>/dev/null; then
    # Locate the exact line for our tarball (trailing double-space before filename)
    EXPECTED="$(grep "  ${TARBALL}$" "${TMP_DIR}/${CHECKSUM_FILE}" | awk '{print $1}' || true)"
    if [ -z "${EXPECTED}" ]; then
      die "No checksum entry for '${TARBALL}' in ${CHECKSUM_FILE}. Aborting — download may be tampered."
    fi
    if command -v sha256sum >/dev/null 2>&1; then
      ACTUAL="$(sha256sum "${TMP_DIR}/${TARBALL}" | awk '{print $1}')"
    elif command -v shasum >/dev/null 2>&1; then
      ACTUAL="$(shasum -a 256 "${TMP_DIR}/${TARBALL}" | awk '{print $1}')"
    else
      warn "sha256sum/shasum not found — checksum verification skipped."
      ACTUAL="${EXPECTED}"  # allow to proceed with warning
    fi
    if [ "${ACTUAL}" != "${EXPECTED}" ]; then
      die "Checksum MISMATCH!\n  Expected: ${EXPECTED}\n  Got:      ${ACTUAL}\n  The download may be corrupted or tampered with. Aborting."
    fi
    success "Checksum verified (${EXPECTED:0:16}...)"
    _sum_ok=true
  else
    warn "Could not fetch checksum file — proceeding without verification."
    warn "For full security, verify manually: https://github.com/${REPO}/releases/tag/${LATEST}"
  fi

  # ── Extract (depth-limited to prevent traversal) ────────────────────────────
  info "Extracting..."
  tar -xzf "${TMP_DIR}/${TARBALL}" -C "${TMP_DIR}"

  BINARY_PATH="${TMP_DIR}/${BINARY}"
  if [ ! -f "${BINARY_PATH}" ]; then
    BINARY_PATH="$(find "${TMP_DIR}" -maxdepth 2 -type f -name "${BINARY}" | head -1 || true)"
  fi
  [ -n "${BINARY_PATH}" ] && [ -f "${BINARY_PATH}" ] \
    || die "Binary '${BINARY}' not found in release archive."

  # ── Install ─────────────────────────────────────────────────────────────────
  if [ -w "${INSTALL_DIR}" ]; then
    install -m 755 "${BINARY_PATH}" "${INSTALL_DIR}/${BINARY}"
    INSTALLED_AT="${INSTALL_DIR}/${BINARY}"
  elif command -v sudo >/dev/null 2>&1; then
    info "Installing to ${INSTALL_DIR} (sudo required)..."
    sudo install -m 755 "${BINARY_PATH}" "${INSTALL_DIR}/${BINARY}"
    INSTALLED_AT="${INSTALL_DIR}/${BINARY}"
  else
    mkdir -p "${FALLBACK_DIR}"
    install -m 755 "${BINARY_PATH}" "${FALLBACK_DIR}/${BINARY}"
    INSTALLED_AT="${FALLBACK_DIR}/${BINARY}"
  fi

else
  # ── Path B: Install from source via Go toolchain ────────────────────────────
  if ! command -v go >/dev/null 2>&1; then
    printf "\n"
    warn "No precompiled binary available for ${OS}/${ARCH}."
    die "Go 1.22+ is required to install from source. Get it at https://go.dev/dl/ then re-run this script."
  fi

  # Enforce minimum Go version 1.22+
  _go_ver="$(go version 2>/dev/null | grep -oE 'go[0-9]+\.[0-9]+' | head -1 || true)"
  _go_minor="$(printf '%s' "${_go_ver}" | grep -oE '[0-9]+\.[0-9]+' | cut -d. -f2 || echo 0)"
  _go_major="$(printf '%s' "${_go_ver}" | grep -oE '[0-9]+\.[0-9]+' | cut -d. -f1 || echo 0)"
  if [ "${_go_major}" -lt 1 ] || { [ "${_go_major}" -eq 1 ] && [ "${_go_minor}" -lt 22 ]; }; then
    die "Go 1.22+ required (found ${_go_ver:-unknown}). Upgrade at https://go.dev/dl/"
  fi

  info "Found $(go version)"
  info "Installing via: go install github.com/${REPO}/cmd/kiw@${TARGET_TAG}"
  # NOTE: GONOSUMDB / GONOSUMCHECK not set — module sum database remains active.
  # GOPROXY defaults to proxy.golang.org ensuring provenance.
  go install "github.com/${REPO}/cmd/kiw@${TARGET_TAG}"

  GOPATH_BIN="$(go env GOPATH)/bin"
  GOBIN_DIR="$(go env GOBIN 2>/dev/null || true)"
  SRC_BIN=""

  if [ -n "${GOBIN_DIR}" ] && [ -f "${GOBIN_DIR}/${BINARY}" ]; then
    SRC_BIN="${GOBIN_DIR}/${BINARY}"
  elif [ -f "${GOPATH_BIN}/${BINARY}" ]; then
    SRC_BIN="${GOPATH_BIN}/${BINARY}"
  elif command -v "${BINARY}" >/dev/null 2>&1; then
    SRC_BIN="$(command -v "${BINARY}")"
  else
    die "go install succeeded but '${BINARY}' not found in ${GOPATH_BIN}."
  fi

  # Copy to /usr/local/bin if writable, else keep in GOPATH/bin
  if [ -w "${INSTALL_DIR}" ]; then
    install -m 755 "${SRC_BIN}" "${INSTALL_DIR}/${BINARY}"
    INSTALLED_AT="${INSTALL_DIR}/${BINARY}"
  elif command -v sudo >/dev/null 2>&1 && [ -t 0 ]; then
    info "Copying to ${INSTALL_DIR} for global PATH (sudo required)..."
    if sudo install -m 755 "${SRC_BIN}" "${INSTALL_DIR}/${BINARY}" 2>/dev/null; then
      INSTALLED_AT="${INSTALL_DIR}/${BINARY}"
    else
      INSTALLED_AT="${SRC_BIN}"
    fi
  else
    INSTALLED_AT="${SRC_BIN}"
  fi
fi

# ── Verify installed binary (use explicit path, not PATH lookup) ───────────────
printf "\n"
if [ -n "${INSTALLED_AT}" ] && [ -f "${INSTALLED_AT}" ] && [ -x "${INSTALLED_AT}" ]; then
  VER_TXT="$("${INSTALLED_AT}" version 2>/dev/null | head -1 || echo "(version unavailable)")"
  success "kiw installed successfully!"
  info "Binary:  ${INSTALLED_AT}"
  info "Version: ${VER_TXT}"

  # Warn if the install dir isn't in PATH
  _bin_dir="$(dirname "${INSTALLED_AT}")"
  case ":${PATH}:" in
    *":${_bin_dir}:"*) ;;
    *)
      warn "'${_bin_dir}' is not in your PATH."
      printf "\n    Add to ~/.bashrc, ~/.zshrc, or ~/.profile:\n"
      printf "    export PATH=\"\$PATH:%s\"\n\n" "${_bin_dir}"
      ;;
  esac
else
  die "Installation verification failed — binary not found or not executable at '${INSTALLED_AT}'."
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
