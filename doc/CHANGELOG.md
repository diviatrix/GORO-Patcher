# Changelog

## [0.0.3] - 2026-08-25

### Security

- **Patch-note XSS (DOM) — notes are now sanitized** — rendered markdown is scrubbed at the gomarkdown AST stage before it reaches the frontend's `innerHTML`: raw HTML nodes (`<script>`, event attributes, `<img onerror>` …) are dropped, and link/image destinations are kept only when they are `http(s)` URLs (`javascript:`, `data:`, `file:` and relative targets are stripped). Notes are an unauthenticated content channel, so untrusted note text can no longer inject markup into the patcher UI. Inline code is preserved (escaped by the renderer).
- **Manifest signing (Ed25519)** — `plist.json` is now signed by the publisher (`hashfile genkey` / `sign`) and verified against an embedded public key before any field is trusted. Release builds (`build.sh --release`) refuse unsigned or tampered manifests; dev builds skip verification so local, unsigned manifests keep working.
- **HTTPS-only release transport** — release builds reject every scheme except `https://` for all fetched content (manifest, patches, patcher binary, notes). Dev builds still allow `http://`/`file://` for local testing. Selected at compile time via the `release` build tag, so a shipped binary cannot be downgraded.
- **SHA-256 integrity everywhere** — the non-cryptographic XXHash64 is removed entirely. Patch binaries, raw-patch/installed-file validation, and the self-update binary are all verified with SHA-256 (unified `pkg/downloader` functions); `hashfile` now emits SHA-256 by default. Cryptographic hashing resists the collision/preimage craft a 64-bit checksum could not.
- **Self-update version gate** — the manifest's `patcher_version` must strictly exceed the running patcher's `CurrentPatcherVersion`; updates to the same or older versions (rollback/downgrade) are refused.
- **Manifest filename hardening** — `patch.Name` and `patch.Target` (both manifest-controlled) are now validated by `engine.SafePatchComponent()` before any join onto the game directory. Requires a bare relative filename and rejects path separators (`../`, absolute, nested), empty/`.`/`..`, `:` (NTFS ADS), trailing `.`/space, and Windows reserved device names (`CON`, `NUL`, `AUX`, `PRN`, `CONIN$`, `CONOUT$`, `CLOCK$`, `COM1–9`, `LPT1–9`). Fail-closed and applied to the patch download (WRITE), merged GRF write, and GRF-check read paths. See ARCHITECTURE.md → "File Name Validation (Security)".

### Fixes

- **Raw-patch path traversal (Windows)** — `SanitizePath` now rejects `:` (drive letters / NTFS ADS) in zip entry names, so a crafted raw patch can no longer write outside the game directory.
- **Downloader false-success on corrupt transfers** — a `416 Range Not Satisfiable` now resets and refetches fresh instead of returning success with a partial file; bodies that end before `Content-Length` are treated as errors; partial resumes roll back to the last known-good boundary. `FetchBytes` also enforces a 32 MiB response cap.

### Internal

- Remove the dead `EventEmitter` progress/error pipeline (App's `EmitProgress`/`EmitError` were no-ops; the UI polls `GetProgress`). Self-update now surfaces a restart failure instead of stalling on "Checking patcher update…".
- Deduplicate the download→verify→apply loop between check and repair into `App.applyPatches`, replacing a hand-rolled O(n²) bubble sort with `sort.Slice` and dropping unused return values/params.
- Drop production-dead symbols (`downloader.VerifyBytes`, `engine.NeedsUpdate`/`MaxPatchID`, `grf.MergeGRF`) and unread manifest fields (`allow_start_on_error`, `news_url`, `patcher_size`) and `pendingCheck.kind`.
- Skip redundant re-hashing of verified cached patch archives on repeated runs.

## [0.0.2] - 2026-08-08

### Features

- **Notes system** — display markdown notes (patch notes, news, announcements) in the patcher UI
  - Configured via `notes_url` in `goro-config.json` pointing to `notes.json`
  - Notes are markdown files rendered to HTML on the Go side (`gomarkdown/markdown`)
  - Custom CSS support via `notes_css_url` for styling notes
  - Notes sorted by ID descending — newest first
  - Supports `file://`, `http://`, `https://` protocols
- **External link handling** — links in notes open in default browser, not inside patcher
- **Drag-and-drop prevention** — dropping files onto patcher UI no longer opens them

### Fixes

- Fixed list bullet padding in notes display
- Fixed random file dnd to patcher window

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
- **Patch validation** — patcher validates applied patches against manifest on start
- **Repair** — redownload and reapply from first mismatching patch when corruption detected

### Technical

- **Stack**: Go + Wails v3 + vanilla HTML/CSS/JS
- **GRF format**: 46-byte header, `FileTableOffset` relative to header end (SEEK_CUR), data blocks before file table
- **Version tracking**: patch ID = version, stored in `goro-patch.json` with patch metadata, no separate version field in manifest
- **Config**: `goro-config.json` with `manifest_url` and `exe_name`
- **Manifest**: `plist.json` with `patch_base_url` and patches array
- **File encoding**: EUC-KR filenames decoded to UTF-8 via `golang.org/x/text/encoding/korean`

### Project Structure

- Source code in `src/` directory
- Build output to `build/` directory
- Documentation in `doc/`
- Build script `build.sh` for Linux and Windows
