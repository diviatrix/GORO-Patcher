# Building GORO-Patcher

## Prerequisites

- Go 1.25+ — https://go.dev/dl/ (`src/go.mod` requires `go 1.25.0`)
- Wails v3 CLI: `go install github.com/wailsapp/wails/v3/cmd/wails3@latest`
  (used to generate frontend bindings and the Windows `.syso`; the binaries
  themselves are built with plain `go build`)

### Additional for Linux

- GCC (or compatible C compiler)
- GTK4 + WebKitGTK development headers

**Arch / CachyOS:** `pacman -S gtk4 webkitgtk-6.0 base-devel`
**Debian / Ubuntu:** `apt install build-essential libgtk-4-dev libwebkitgtk-6.0-dev`
**Fedora:** `dnf install gcc gtk4-devel webkitgtk6.0-devel`
**openSUSE:** `zypper install gcc gtk4-devel webkit2gtk-6_0-devel`

### Additional for Linux → Windows cross-compile

- mingw-w64

**Arch / CachyOS:** `pacman -S mingw-w64-gcc`
**Debian / Ubuntu:** `apt install gcc-mingw-w64-x86-64`
**Fedora:** `dnf install mingw64-gcc`
**openSUSE:** `zypper install mingw64-cross-gcc`

## Build

### Linux

```bash
./scripts/build.sh linux
```

### Windows (native, from Windows)

```cmd
scripts\build.bat
```

Requires Go, Wails CLI, and a C compiler (MinGW or MSVC) in PATH.

### Windows (cross-compile, from Linux)

```bash
./scripts/build.sh windows
```

### Both platforms (from Linux)

```bash
./scripts/build.sh all
```

## Build Script Options

```
Usage: ./scripts/build.sh [OPTIONS] [TARGET...]

Targets:
  linux       Build Linux binary
  windows     Build Windows binary (.exe)
  all         Build both (default: current platform only)

Options:
  -c, --clean     Clean build artifacts before building
  -f, --frontend  Regenerate frontend bindings
  -r, --release   Add the `release` build tag (HTTPS-only + required signature)
  -h, --help      Show help
```

## Dev vs. Release builds

Two build channels exist, selected at compile time by the presence of the
`release` build tag. A shipped binary cannot have its channel changed at
runtime, so an attacker cannot flip security hardening back on.

- **Dev** (default — plain `./scripts/build.sh`): the patcher accepts `http://`,
  `https://`, `file://` and local paths, and does **not** require a manifest
  signature. Local, unsigned manifests work unchanged.
- **Release** (`./scripts/build.sh --release`): the patcher allows **only
  `https://`** for everything it fetches and **requires** a valid Ed25519
  signature on the manifest. Any other scheme or an unsigned/tampered manifest
  is rejected.

Because the hardened channel cannot be loosened at runtime, a release patcher
is built **per server** by its admin — there is no generic pre-built release
binary to download. Each admin runs the release build themselves:
`./scripts/build.sh --release` (Linux) or `--release windows` / `build.bat -r`
for the `.exe`, then ships that specific binary together with a
`goro-config.json` that carries their public key (below).

Before shipping a release, the publisher must:

1. Generate a keypair (once; keep the **private** key off-repo, never ship it):
   ```bash
   hashfile genkey -out keys/key.pem -pub keys/pub.pem
   ```
   `genkey` prints the base64 public key. The public key is **not** embedded in
   the binary — it is supplied at runtime, next to the patcher, via
   `manifest_public_key` in `goro-config.json` or the `GORO_PATCHER_PUBKEY`
   environment variable (which overrides config). In a release build the patcher
   fails closed when no key is available.
2. Bump `src/pkg/engine/version.go` (`CurrentPatcherVersion`) and record the
   same integer as `patcher_version` in the manifest (the self-update rollback
   gate refuses same/older versions).
3. Compute the patcher binary's **SHA-256** for `patcher_hash`:
   ```bash
   hashfile sha256 build/GORO-Patcher
   ```
4. Sign the manifest and re-upload it:
   ```bash
   hashfile sign -key keys/key.pem -in plist.json
   hashfile verify -key keys/pub.pem -in plist.json
   ```
   In a release build the patcher refuses to run against an unsigned manifest.

All content URLs (`patch_base_url`, `patcher_url`, `manifest_url` in
`goro-config.json`, notes) must be `https://` in the config you ship for a
release build.

## Output

Each build produces:
- `build/GORO-Patcher` (Linux) or `build/GORO-Patcher.exe` (Windows)
- `build/hashfile` (Linux) or `build/hashfile.exe` (Windows) — hash/signing tool
- `build/goro-config.json.example` — example config
- `build/public/plist.json.example` — example manifest (copied from `example/`)
- `build/public/data/` — place patch files here

To keep a **dev** and a **release** version of each platform, rebuild per
channel and move the outputs aside (see the example in `build.sh --help`); the
release build in this repo keeps all four under `build/dist/` as a convenience.

## Development

```bash
cd src && wails3 dev
```

## Testing

```bash
cd src && go test ./...
cd src && go vet ./...
```

## Project Structure

```
GORO-Patcher/
├── src/                    # Source code
│   ├── app.go              # Main app, patch pipeline
│   ├── main.go             # Wails entry point
│   ├── go.mod
│   ├── frontend/           # UI (HTML/CSS/JS)
│   ├── pkg/                # Go packages
│   ├── cmd/hashfile/       # Hash + Ed25519 signing tool (sha256/genkey/sign/verify)
│   └── cmd/gen-plist/      # Authors installed-result file_hashes in a manifest
├── scripts/                # Build scripts
│   ├── build.sh            # Linux / cross-compile
│   ├── build.bat           # Windows native
│   └── gen-api-doc.sh      # API doc generator (runs on build)
├── example/                # Example configs + test data
│   ├── plist.json          # Test manifest
│   ├── plist.json.example  # Template for new servers
│   ├── goro-config.json.example
│   └── data/               # Test patches (patch_0.grf, patch_1.zip)
├── build/                  # Build output (self-contained)
│   ├── GORO-Patcher        # Patcher binary
│   ├── hashfile            # Hash tool
│   ├── goro-config.json.example
│   └── public/             # Server-ready examples
│       ├── plist.json.example
│       └── data/           # Place patches here
├── doc/                    # Documentation
```
