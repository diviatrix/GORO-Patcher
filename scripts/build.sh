#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$SCRIPT_DIR/.."
SRC_DIR="$PROJECT_DIR/src"
OUT_DIR="$PROJECT_DIR/build"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log() { echo -e "${GREEN}[build]${NC} $1"; }
warn() { echo -e "${YELLOW}[build]${NC} $1"; }
err() { echo -e "${RED}[build]${NC} $1"; exit 1; }

detect_os() {
    case "$(uname -s)" in
        Linux*)     echo "linux" ;;
        MINGW*|MSYS*|CYGWIN*)  echo "windows" ;;
        *)          echo "unknown" ;;
    esac
}

HOST_OS=$(detect_os)

usage() {
    cat <<EOF
Usage: $0 [OPTIONS] [TARGET...]

Build GORO-Patcher.

Targets:
  linux       Build Linux binary
  windows     Build Windows binary (.exe)
  all         Build both (default: current platform only)

Options:
  -c, --clean     Clean build artifacts before building
  -f, --frontend  Regenerate frontend bindings
  -r, --release   Build a signed release (release build tag)
  -h, --help      Show this help

Release notes:
  -r adds the `release` build tag, which enforces HTTPS-only fetching and
  requires a valid Ed25519 signature on the manifest. Set the publisher public
  key in src/pkg/engine/manifest_verify_release.go (via `hashfile genkey`)
  before a release build.

Platform notes:
  Linux:  Can build linux (native) and windows (cross-compile via wails3)
  Windows: Can build windows (native). Linux build not supported.
EOF
}

CLEAN=false
REGENERATE=true
RELEASE=false
TARGETS=()

while [[ $# -gt 0 ]]; do
    case $1 in
        -c|--clean) CLEAN=true; shift ;;
        -f|--frontend) REGENERATE=true; shift ;;
        -r|--release) RELEASE=true; shift ;;
        -h|--help) usage; exit 0 ;;
        linux|windows|all) TARGETS+=("$1"); shift ;;
        *) err "Unknown option: $1" ;;
    esac
done

if [[ ${#TARGETS[@]} -eq 0 ]]; then
    case "$HOST_OS" in
        linux)  TARGETS=("linux") ;;
        windows) TARGETS=("windows") ;;
        *) err "Cannot detect OS. Specify target: linux or windows" ;;
    esac
fi

resolve_targets() {
    local resolved=()
    for t in "${TARGETS[@]}"; do
        case $t in
            all)
                resolved+=(linux windows)
                ;;
            linux)
                if [[ "$HOST_OS" != "linux" ]]; then
                    err "Linux build only supported on Linux"
                fi
                resolved+=(linux)
                ;;
            windows)
                resolved+=(windows)
                ;;
        esac
    done
    printf '%s\n' "${resolved[@]}" | sort -u
}

find_wails() {
    if command -v wails3 &>/dev/null; then
        echo "wails3"
    elif [[ -x "$HOME/go/bin/wails3" ]]; then
        echo "$HOME/go/bin/wails3"
    elif [[ -x "/c/Users/$USER/go/bin/wails3.exe" ]]; then
        echo "/c/Users/$USER/go/bin/wails3.exe"
    else
        err "wails3 not found. Install: go install github.com/wailsapp/wails/v3/cmd/wails3@latest"
    fi
}

check_deps() {
    local target=$1
    command -v go &>/dev/null || err "Go not found. Install from https://go.dev/dl/"

    case $target in
        linux)
            if [[ "$HOST_OS" == "linux" ]]; then
                command -v gcc &>/dev/null || err "GCC not found. Install build-essential / base-devel"
            fi
            ;;
        windows)
            ;;
    esac
}

clean() {
    log "Cleaning build artifacts..."
    rm -rf "$SRC_DIR/build/bin" "$SRC_DIR/frontend/wailsjs" "$SRC_DIR/build"
    rm -f "$OUT_DIR/GORO-Patcher" "$OUT_DIR/GORO-Patcher.exe"
    rm -f "$OUT_DIR/hashfile" "$OUT_DIR/hashfile.exe"
}

generate_bindings() {
    log "Generating TypeScript bindings..."
    cd "$SRC_DIR" && $(find_wails) generate bindings -b -d frontend/dist/bindings
    log "Generating API docs..."
    "$SCRIPT_DIR/gen-api-doc.sh"
}

build_linux() {
    log "Building for Linux... ($([ "$RELEASE" = true ] && echo release || echo dev))"
    cd "$SRC_DIR"
    local wails=$(find_wails)

    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
        go build -o "$OUT_DIR/hashfile" ./cmd/hashfile

    local extra_tags=""
    $RELEASE && extra_tags="EXTRA_TAGS=release"

    $wails task linux:build GOOS=linux $extra_tags

    cp "$SRC_DIR/build/bin/GORO-Patcher" "$OUT_DIR/GORO-Patcher"
    rm -rf "$SRC_DIR/build"

    log "Linux: $OUT_DIR/GORO-Patcher, $OUT_DIR/hashfile"
}

build_windows() {
    log "Building for Windows... ($([ "$RELEASE" = true ] && echo release || echo dev))"
    cd "$SRC_DIR"
    local wails=$(find_wails)

    CGO_ENABLED=0 GOOS=windows GOARCH=amd64 \
        go build -o "$OUT_DIR/hashfile.exe" ./cmd/hashfile

    local extra_tags=""
    $RELEASE && extra_tags="EXTRA_TAGS=release"

    $wails task windows:build GOOS=windows $extra_tags

    cp "$SRC_DIR/build/bin/GORO-Patcher.exe" "$OUT_DIR/GORO-Patcher.exe"
    rm -rf "$SRC_DIR/build"

    log "Windows: $OUT_DIR/GORO-Patcher.exe, $OUT_DIR/hashfile.exe"
}

copy_examples() {
    log "Copying example files to build/..."
    cp "$PROJECT_DIR/example/goro-config.json.example" "$OUT_DIR/" 2>/dev/null || true
    mkdir -p "$OUT_DIR/public/data"
    cp "$PROJECT_DIR/example/plist.json.example" "$OUT_DIR/public/"
}

log "GORO-Patcher build script (host: $HOST_OS)"

$CLEAN && clean
$REGENERATE && generate_bindings

for target in $(resolve_targets); do
    check_deps "$target"
    case $target in
        linux) build_linux ;;
        windows) build_windows ;;
    esac
done

copy_examples

log "Done!"
ls -lh "$OUT_DIR"/GORO-Patcher* "$OUT_DIR"/hashfile* "$OUT_DIR"/*.example 2>/dev/null
