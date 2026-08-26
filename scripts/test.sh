#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$SCRIPT_DIR/.."
SRC_DIR="$PROJECT_DIR/src"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log() { echo -e "${GREEN}[test]${NC} $1"; }
err() { echo -e "${RED}[test]${NC} $1"; exit 1; }

cd "$SRC_DIR"

log "Verifying formatting (gofmt)"
unformatted=$(gofmt -l . | grep '\.go$' || true)
if [[ -n "$unformatted" ]]; then
    echo "$unformatted"
    err "Unformatted Go files (run: gofmt -w src/)"
fi

log "go vet ./..."
go vet ./...

log "go test ./... (dev)"
go test -count=1 ./...

log "go test -race ./..."
go test -count=1 -race ./...

log "go test -tags release ./..."
go test -count=1 -tags release ./...

# The Wails GUI app (root main) needs platform toolchains (cgo/gcc on Linux) and is
# built via build.sh; here we cross-compile the reusable packages + CLI tools.
log "cross-compile packages + tools (linux)"
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build ./cmd/... ./pkg/...

log "cross-compile packages + tools (windows)"
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build ./cmd/... ./pkg/...

log "All checks passed"