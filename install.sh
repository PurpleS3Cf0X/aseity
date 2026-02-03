#!/bin/sh
set -e

# Aseity installer
# Usage: curl -fsSL https://raw.githubusercontent.com/PurpleS3Cf0X/aseity/master/install.sh | sh

# Ensure we can interact with the user even if piped
# logic moved to confirm function to avoid breaking "curl | sh"
# (exec < /dev/tty would cause sh to stop reading the script from the pipe)

REPO="PurpleS3Cf0X/aseity"
INSTALL_DIR="/usr/local/bin"
BINARY="aseity"

GREEN='\033[0;32m'
BRIGHT_GREEN='\033[1;32m'
YELLOW='\033[1;33m'
DIM='\033[2m'
RED='\033[0;31m'
NC='\033[0m'

info() { printf "${GREEN}▸${NC} %s\n" "$1"; }
ask() { printf "${YELLOW}▸ %s [Y/n]${NC} " "$1"; }
success() { printf "${BRIGHT_GREEN}✓${NC} %s\n" "$1"; }
error() { printf "${RED}✗${NC} %s\n" "$1" >&2; exit 1; }

# Interactive Yes/No prompt
confirm() {
    ask "$1"
    if [ -t 0 ]; then
        read -r response
    elif [ -e /dev/tty ]; then
        # Read explicitly from TTY if stdin is a pipe
        read -r response < /dev/tty
    else
        read -r response
    fi
    
    case "$response" in
        [yY][eE][sS]|[yY]|"") return 0 ;;
        *) return 1 ;;
    esac
}

# Detect Package Manager
detect_pkg_manager() {
    if command -v brew >/dev/null 2>&1; then
        PkgManager="brew"
        InstallCmd="brew install"
    elif command -v apt-get >/dev/null 2>&1; then
        PkgManager="apt"
        InstallCmd="sudo apt-get install -y"
    elif command -v dnf >/dev/null 2>&1; then
        PkgManager="dnf"
        InstallCmd="sudo dnf install -y"
    elif command -v pacman >/dev/null 2>&1; then
        PkgManager="pacman"
        InstallCmd="sudo pacman -S --noconfirm"
    else
        PkgManager=""
    fi
}

detect_pkg_manager

ensure_cmd() {
    cmd=$1
    pkg=$2
    if ! command -v "$cmd" >/dev/null 2>&1; then
        info "$cmd is missing."
        if [ -n "$PkgManager" ]; then
            if confirm "Would you like to install '$pkg' using $PkgManager?"; then
                info "Installing $pkg..."
                $InstallCmd "$pkg"
                success "$pkg installed"
            else
                error "$cmd is required to proceed. Please install it manually."
            fi
        else
            error "$cmd is required but no supported package manager found. Please install '$pkg' manually."
        fi
    fi
}

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

# Ensure we have curl or wget
if ! command -v curl >/dev/null 2>&1 && ! command -v wget >/dev/null 2>&1; then
    ensure_cmd curl curl
fi

# Get latest release tag
info "Fetching latest release..."
if command -v curl >/dev/null 2>&1; then
    LATEST=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed 's/.*"tag_name": *"//;s/".*//')
elif command -v wget >/dev/null 2>&1; then
    LATEST=$(wget -qO- "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed 's/.*"tag_name": *"//;s/".*//')
fi

if [ -z "$LATEST" ]; then
    # No release yet, build from source
    info "No binary release found for this detection. Building from source..."

    # Ensure dependencies for building
    ensure_cmd git git
    ensure_cmd go golang

    TMPDIR=$(mktemp -d)
    trap 'rm -rf "$TMPDIR"' EXIT

    info "Cloning repository..."
    git clone --depth 1 "https://github.com/${REPO}.git" "$TMPDIR/aseity"

    info "Building..."
    cd "$TMPDIR/aseity"
    CGO_ENABLED=0 go build -ldflags="-s -w" -o "${BINARY}" ./cmd/aseity

    info "Installing to ${INSTALL_DIR}..."
    if [ -w "$INSTALL_DIR" ]; then
        mv "${BINARY}" "${INSTALL_DIR}/${BINARY}"
    else
        if confirm "Install to ${INSTALL_DIR}? (Requires sudo)"; then
            sudo mv "${BINARY}" "${INSTALL_DIR}/${BINARY}"
        else
            error "Installation aborted."
        fi
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
    # Check explicitly if we need sudo
    if confirm "Install to ${INSTALL_DIR}? (Requires sudo)"; then
        sudo mv "$TMPFILE" "${INSTALL_DIR}/${BINARY}"
    else
        error "Installation aborted."
    fi
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
