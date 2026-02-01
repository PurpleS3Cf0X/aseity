#!/bin/sh
set -e

# Aseity installer
# Usage: curl -fsSL https://raw.githubusercontent.com/PurpleS3Cf0X/aseity/master/install.sh | sh

REPO="PurpleS3Cf0X/aseity"
INSTALL_DIR="/usr/local/bin"
BINARY="aseity"

GREEN='\033[0;32m'
BRIGHT_GREEN='\033[1;32m'
DIM='\033[2m'
RED='\033[0;31m'
NC='\033[0m'

info() { printf "${GREEN}▸${NC} %s\n" "$1"; }
success() { printf "${BRIGHT_GREEN}✓${NC} %s\n" "$1"; }
error() { printf "${RED}✗${NC} %s\n" "$1" >&2; exit 1; }

# Detect OS
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$OS" in
    linux)  OS="linux" ;;
    darwin) OS="darwin" ;;
    *)      error "Unsupported OS: $OS" ;;
esac

# Detect architecture
ARCH=$(uname -m)
case "$ARCH" in
    x86_64|amd64)  ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *)             error "Unsupported architecture: $ARCH" ;;
esac

info "Detected ${OS}/${ARCH}"

# Get latest release tag
info "Fetching latest release..."
if command -v curl >/dev/null 2>&1; then
    LATEST=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed 's/.*"tag_name": *"//;s/".*//')
elif command -v wget >/dev/null 2>&1; then
    LATEST=$(wget -qO- "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed 's/.*"tag_name": *"//;s/".*//')
else
    error "Neither curl nor wget found. Install one and retry."
fi

if [ -z "$LATEST" ]; then
    # No release yet, build from source
    info "No release found. Building from source..."

    if ! command -v go >/dev/null 2>&1; then
        error "Go is required to build from source. Install Go: https://go.dev/dl/"
    fi

    TMPDIR=$(mktemp -d)
    trap 'rm -rf "$TMPDIR"' EXIT

    info "Cloning repository..."
    git clone --depth 1 "https://github.com/${REPO}.git" "$TMPDIR/aseity" 2>/dev/null

    info "Building..."
    cd "$TMPDIR/aseity"
    CGO_ENABLED=0 go build -ldflags="-s -w" -o "${BINARY}" ./cmd/aseity

    info "Installing to ${INSTALL_DIR}..."
    if [ -w "$INSTALL_DIR" ]; then
        mv "${BINARY}" "${INSTALL_DIR}/${BINARY}"
    else
        sudo mv "${BINARY}" "${INSTALL_DIR}/${BINARY}"
    fi

    success "Installed aseity $(${INSTALL_DIR}/${BINARY} --version)"
    exit 0
fi

info "Latest version: ${LATEST}"

# Download binary
FILENAME="aseity-${OS}-${ARCH}"
URL="https://github.com/${REPO}/releases/download/${LATEST}/${FILENAME}"

TMPFILE=$(mktemp)
trap 'rm -f "$TMPFILE"' EXIT

info "Downloading ${URL}..."
if command -v curl >/dev/null 2>&1; then
    curl -fsSL "$URL" -o "$TMPFILE"
elif command -v wget >/dev/null 2>&1; then
    wget -q "$URL" -O "$TMPFILE"
fi

chmod +x "$TMPFILE"

# Install
info "Installing to ${INSTALL_DIR}/${BINARY}..."
if [ -w "$INSTALL_DIR" ]; then
    mv "$TMPFILE" "${INSTALL_DIR}/${BINARY}"
else
    sudo mv "$TMPFILE" "${INSTALL_DIR}/${BINARY}"
fi

success "Installed aseity ${LATEST}"

# Verify
if command -v aseity >/dev/null 2>&1; then
    printf "\n${BRIGHT_GREEN}"
    cat <<'BANNER'
   ██████╗ ███████╗███████╗██╗████████╗██╗   ██╗
  ██╔══██╗██╔════╝██╔════╝██║╚══██╔══╝╚██╗ ██╔╝
  ███████║███████╗█████╗  ██║   ██║    ╚████╔╝
  ██╔══██║╚════██║██╔══╝  ██║   ██║     ╚██╔╝
  ██║  ██║███████║███████╗██║   ██║      ██║
  ╚═╝  ╚═╝╚══════╝╚══════╝╚═╝   ╚═╝      ╚═╝
BANNER
    printf "${NC}\n"
    printf "  ${DIM}Type ${BRIGHT_GREEN}aseity${DIM} to start.${NC}\n\n"
else
    printf "\n  ${DIM}Make sure ${INSTALL_DIR} is in your PATH.${NC}\n\n"
fi
