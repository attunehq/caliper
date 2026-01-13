#!/bin/sh
# Caliper install script
# Usage:
#   curl -sSL https://raw.githubusercontent.com/attunehq/caliper/main/install.sh | sh
#   curl -sSL https://raw.githubusercontent.com/attunehq/caliper/main/install.sh | sh -s -- --version v0.1.0
#   curl -sSL https://raw.githubusercontent.com/attunehq/caliper/main/install.sh | sh -s -- --dir ~/.local/bin

set -e

REPO="attunehq/caliper"
BINARY_NAME="caliper"
INSTALL_DIR="/usr/local/bin"
VERSION=""

# Colors for output (disabled if not a terminal)
if [ -t 1 ]; then
    RED='\033[0;31m'
    GREEN='\033[0;32m'
    YELLOW='\033[0;33m'
    BLUE='\033[0;34m'
    NC='\033[0m' # No Color
else
    RED=''
    GREEN=''
    YELLOW=''
    BLUE=''
    NC=''
fi

info() {
    printf "${BLUE}==>${NC} %s\n" "$1"
}

success() {
    printf "${GREEN}==>${NC} %s\n" "$1"
}

warn() {
    printf "${YELLOW}Warning:${NC} %s\n" "$1"
}

error() {
    printf "${RED}Error:${NC} %s\n" "$1" >&2
    exit 1
}

# Parse command line arguments
while [ $# -gt 0 ]; do
    case "$1" in
        --version)
            VERSION="$2"
            shift 2
            ;;
        --dir)
            INSTALL_DIR="$2"
            shift 2
            ;;
        -h|--help)
            echo "Caliper install script"
            echo ""
            echo "Usage:"
            echo "  curl -sSL https://raw.githubusercontent.com/attunehq/caliper/main/install.sh | sh"
            echo ""
            echo "Options:"
            echo "  --version <version>  Install a specific version (e.g., v0.1.0)"
            echo "  --dir <path>         Install to a custom directory (default: /usr/local/bin)"
            echo "  -h, --help           Show this help message"
            exit 0
            ;;
        *)
            error "Unknown option: $1"
            ;;
    esac
done

# Detect OS
detect_os() {
    OS="$(uname -s)"
    case "$OS" in
        Linux)
            echo "linux"
            ;;
        Darwin)
            echo "darwin"
            ;;
        *)
            error "Unsupported operating system: $OS. Caliper only supports Linux and macOS."
            ;;
    esac
}

# Detect architecture
detect_arch() {
    ARCH="$(uname -m)"
    case "$ARCH" in
        x86_64|amd64)
            echo "amd64"
            ;;
        arm64|aarch64)
            echo "arm64"
            ;;
        *)
            error "Unsupported architecture: $ARCH. Caliper only supports amd64 and arm64."
            ;;
    esac
}

# Check for required commands
check_dependencies() {
    if ! command -v curl >/dev/null 2>&1 && ! command -v wget >/dev/null 2>&1; then
        error "Either 'curl' or 'wget' is required but not installed."
    fi

    if ! command -v tar >/dev/null 2>&1; then
        error "'tar' is required but not installed."
    fi
}

# Download file using curl or wget
download() {
    url="$1"
    output="$2"

    if command -v curl >/dev/null 2>&1; then
        curl -fsSL "$url" -o "$output"
    elif command -v wget >/dev/null 2>&1; then
        wget -q "$url" -O "$output"
    fi
}

# Get the latest version from GitHub API
get_latest_version() {
    if command -v curl >/dev/null 2>&1; then
        curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/'
    elif command -v wget >/dev/null 2>&1; then
        wget -qO- "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/'
    fi
}

# Verify SHA256 checksum
verify_checksum() {
    archive="$1"
    checksums_file="$2"
    archive_name="$3"

    expected_checksum=$(grep "${archive_name}" "$checksums_file" | awk '{print $1}')
    
    if [ -z "$expected_checksum" ]; then
        error "Could not find checksum for ${archive_name} in checksums file."
    fi

    if command -v sha256sum >/dev/null 2>&1; then
        actual_checksum=$(sha256sum "$archive" | awk '{print $1}')
    elif command -v shasum >/dev/null 2>&1; then
        actual_checksum=$(shasum -a 256 "$archive" | awk '{print $1}')
    else
        warn "Neither sha256sum nor shasum found. Skipping checksum verification."
        return 0
    fi

    if [ "$expected_checksum" != "$actual_checksum" ]; then
        error "Checksum verification failed!\nExpected: ${expected_checksum}\nActual:   ${actual_checksum}"
    fi
}

# Main installation function
main() {
    info "Installing Caliper..."

    check_dependencies

    OS=$(detect_os)
    ARCH=$(detect_arch)
    
    info "Detected OS: ${OS}, Architecture: ${ARCH}"

    # Get version
    if [ -z "$VERSION" ]; then
        info "Fetching latest version..."
        VERSION=$(get_latest_version)
        if [ -z "$VERSION" ]; then
            error "Could not determine latest version. Please specify a version with --version."
        fi
    fi

    # Ensure version starts with 'v'
    case "$VERSION" in
        v*) ;;
        *) VERSION="v${VERSION}" ;;
    esac

    info "Installing version: ${VERSION}"

    # Construct download URLs
    ARCHIVE_NAME="${BINARY_NAME}_${OS}_${ARCH}.tar.gz"
    DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${VERSION}/${ARCHIVE_NAME}"
    CHECKSUMS_URL="https://github.com/${REPO}/releases/download/${VERSION}/checksums.txt"

    # Create temporary directory
    TMP_DIR=$(mktemp -d)
    trap 'rm -rf "$TMP_DIR"' EXIT

    # Download archive
    info "Downloading ${ARCHIVE_NAME}..."
    download "$DOWNLOAD_URL" "${TMP_DIR}/${ARCHIVE_NAME}" || error "Failed to download ${DOWNLOAD_URL}"

    # Download checksums
    info "Downloading checksums..."
    download "$CHECKSUMS_URL" "${TMP_DIR}/checksums.txt" || error "Failed to download checksums"

    # Verify checksum
    info "Verifying checksum..."
    verify_checksum "${TMP_DIR}/${ARCHIVE_NAME}" "${TMP_DIR}/checksums.txt" "$ARCHIVE_NAME"
    success "Checksum verified!"

    # Extract archive
    info "Extracting archive..."
    tar -xzf "${TMP_DIR}/${ARCHIVE_NAME}" -C "$TMP_DIR"

    # Install binary
    info "Installing to ${INSTALL_DIR}..."
    
    # Check if we can write to install directory
    if [ -w "$INSTALL_DIR" ]; then
        mv "${TMP_DIR}/${BINARY_NAME}" "${INSTALL_DIR}/${BINARY_NAME}"
        chmod +x "${INSTALL_DIR}/${BINARY_NAME}"
    elif command -v sudo >/dev/null 2>&1; then
        sudo mv "${TMP_DIR}/${BINARY_NAME}" "${INSTALL_DIR}/${BINARY_NAME}"
        sudo chmod +x "${INSTALL_DIR}/${BINARY_NAME}"
    else
        error "Cannot write to ${INSTALL_DIR}. Please run with sudo or use --dir to specify a writable directory."
    fi

    # Verify installation
    if [ -x "${INSTALL_DIR}/${BINARY_NAME}" ]; then
        success "Caliper ${VERSION} installed successfully to ${INSTALL_DIR}/${BINARY_NAME}"
        
        # Check if install dir is in PATH
        case ":$PATH:" in
            *":${INSTALL_DIR}:"*)
                echo ""
                info "Run 'caliper --help' to get started."
                ;;
            *)
                echo ""
                warn "${INSTALL_DIR} is not in your PATH."
                echo "Add it to your PATH by running:"
                echo ""
                echo "  export PATH=\"${INSTALL_DIR}:\$PATH\""
                echo ""
                echo "Or add this line to your shell profile (~/.bashrc, ~/.zshrc, etc.)"
                ;;
        esac
    else
        error "Installation failed. Binary not found at ${INSTALL_DIR}/${BINARY_NAME}"
    fi
}

main
