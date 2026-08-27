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

Build GORO-Patcher (desktop GUI) and the hashfile signing/hashing tool.

Targets:
  linux       Build Linux binaries
  windows     Build Windows binaries (.exe)
  all         Build both (default: current platform only)

Options:
  -c, --clean     Clean build artifacts before building
  -f, --frontend  Regenerate TypeScript bindings + API docs
  -r, --release   Add the \`release\` build tag
  -h, --help      Show this help

Release notes:
  -r selects the hardened \`release\` build tag, enforced at compile time: the
  patcher allows ONLY \`https://\` for fetched content and REQUIRES a valid
  Ed25519 signature on the manifest. Configure the publisher public key at
  runtime via \`manifest_public_key\` in goro-config.json or the
  \`GORO_PATCHER_PUBKEY\` environment variable. Builds without -r (dev) accept
  \`http://\`/`file://\` and skip signature checks, for local testing.

  The windows .exe is built with an embedded icon/manifest via a generated
  .syso; hashfile is a pure-Go tool built for the same platform.

Output:
  Each target writes the canonical files to build/:
    GORO-Patcher (or .exe)  and  hashfile (or .exe)
  plus goro-config.json.example and public/plist.json.example.

Example (all four channels, saved under build/dist):
  ./scripts/build.sh linux        && mv build/GORO-Patcher build/dist/GORO-Patcher-linux-dev
  ./scripts/build.sh linux -r     && mv build/GORO-Patcher build/dist/GORO-Patcher-linux-release
  ./scripts/build.sh windows      && mv build/GORO-Patcher.exe build/dist/GORO-Patcher-windows-dev.exe
  ./scripts/build.sh windows -r   && mv build/GORO-Patcher.exe build/dist/GORO-Patcher-windows-release.exe
  cp build/hashfile build/dist/hashfile-linux
  cp build/hashfile.exe build/dist/hashfile-windows.exe
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
            all) resolved+=(linux windows) ;;
            linux)
                if [[ "$HOST_OS" != "linux" ]]; then
                    err "Linux build only supported on Linux"
                fi
                resolved+=(linux)
                ;;
            windows) resolved+=(windows) ;;
        esac
    done
    printf '%s\n' "${resolved[@]}" | sort -u
}

build_tags() {
    # Mirrors the desktop build: production-optimized (trimpath, stripped).
    # The `release` tag is added for the hardened channel.
    local tags="production"
    $RELEASE && tags="production,release"
    echo "$tags"
}

generate_bindings() {
    log "Generating TypeScript bindings..."
    cd "$SRC_DIR" && wails3 generate bindings -b -d frontend/dist/bindings
    log "Generating API docs..."
    "$SCRIPT_DIR/gen-api-doc.sh"
}

generate_syso() {
    log "Generating Windows .syso (icon/manifest)..."
    rm -f "$SRC_DIR/wails_windows_amd64.syso"
    ( cd "$SRC_DIR/windows" && wails3 generate syso -arch amd64 -icon icon.ico -manifest wails.exe.manifest -info info.json -out ../wails_windows_amd64.syso )
}

clean() {
    log "Cleaning build artifacts..."
    rm -f "$OUT_DIR/GORO-Patcher" "$OUT_DIR/GORO-Patcher.exe"
    rm -f "$OUT_DIR/hashfile" "$OUT_DIR/hashfile.exe"
    rm -f "$SRC_DIR/wails_windows_amd64.syso"
    rm -rf "$SRC_DIR/frontend/wailsjs"
}

build_linux() {
    log "Building for Linux... ($([ "$RELEASE" = true ] && echo release || echo dev))"
    cd "$SRC_DIR"

    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
        go build -trimpath -buildvcs=false -ldflags="-w -s" \
        -o "$OUT_DIR/hashfile" ./cmd/hashfile

    CGO_ENABLED=1 GOOS=linux GOARCH=amd64 \
        go build -tags "$(build_tags)" -trimpath -buildvcs=false -ldflags="-w -s" \
        -o "$OUT_DIR/GORO-Patcher" .

    log "Linux: $OUT_DIR/GORO-Patcher, $OUT_DIR/hashfile"
}

build_windows() {
    log "Building for Windows... ($([ "$RELEASE" = true ] && echo release || echo dev))"
    generate_syso
    cd "$SRC_DIR"

    CGO_ENABLED=0 GOOS=windows GOARCH=amd64 \
        go build -trimpath -buildvcs=false -ldflags="-w -s" \
        -o "$OUT_DIR/hashfile.exe" ./cmd/hashfile

    CGO_ENABLED=0 GOOS=windows GOARCH=amd64 \
        go build -tags "$(build_tags)" -trimpath -buildvcs=false -ldflags="-w -s -H windowsgui" \
        -o "$OUT_DIR/GORO-Patcher.exe" .

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
    case $target in
        linux) build_linux ;;
        windows) build_windows ;;
    esac
done

copy_examples

log "Done!"
ls -lh "$OUT_DIR"/GORO-Patcher* "$OUT_DIR"/hashfile* "$OUT_DIR"/*.example 2>/dev/null