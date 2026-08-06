# Changelog

## [0.0.1] - 2026-08-07

Initial release of GORO-Patcher.

### Features

- **GRF 2.0 read/write/merge** — full format implementation with 46-byte header (rAthena layout), zlib-compressed file table, EUC-KR filename decoding
- **GRF patching** — merge `.grf` files directly or extract `.zip` contents into target GRF
- **Raw patching** — extract zip files to game directory with backup support
- **HTTP downloader** — resume support (Range headers), retry with backoff, xxhash64 verification
- **Self-update** — patcher can update itself from manifest
- **Process detection** — checks if game is running before patching
- **State machine** — clean state transitions (idle → checking → downloading → patching → ready/error)
- **Polling-based progress** — real-time progress via `GetProgress()` bound method
- **Crash recovery** — atomic GRF save, `.bak` restoration, re-download on hash failure
- **Auto-check on start** — patcher checks for updates automatically on launch
- **Cross-platform builds** — `scripts/build.sh` (Linux/cross-compile) and `scripts/build.bat` (native Windows)
- **Hash tool** — `hashfile` command for generating XXHash64 hashes for manifest

### Technical

- **Stack**: Go + Wails v3 + vanilla HTML/CSS/JS
- **GRF format**: 46-byte header, `FileTableOffset` relative to header end (SEEK_CUR), data blocks before file table
- **Version tracking**: patch ID = version, stored in `goro-patch.json`, no separate version field in manifest
- **Config**: `goro-config.json` with `manifest_url` and `exe_name`
- **Manifest**: `plist.json` with `patch_base_url` and patches array
- **File encoding**: EUC-KR filenames decoded to UTF-8 via `golang.org/x/text/encoding/korean`

### Project Structure

- Source code in `src/` directory
- Build output to `build/` directory
- Documentation in `doc/`
- Build script `build.sh` for Linux and Windows
- Makefile for common tasks (`make build`, `make test`, `make lint`)
