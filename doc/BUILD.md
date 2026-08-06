# Building GORO-Patcher

## Prerequisites

- Go 1.22+ — https://go.dev/dl/
- Wails v3 CLI: `go install github.com/wailsapp/wails/v3/cmd/wails3@latest`

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
  -h, --help      Show help
```

## Output

Each build produces:
- `build/GORO-Patcher` (Linux) or `build/GORO-Patcher.exe` (Windows)
- `build/hashfile` (Linux) or `build/hashfile.exe` (Windows) — hash calculation tool
- `build/goro-config.json.example` — example config
- `build/public/plist.json.example` — example manifest (copied from `example/`)
- `build/public/data/` — place patch files here

The `build/` directory is self-contained — copy it to your server as-is.

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
│   └── cmd/hashfile/       # Hash tool
├── scripts/                # Build scripts
│   ├── build.sh            # Linux / cross-compile
│   └── build.bat           # Windows native
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
