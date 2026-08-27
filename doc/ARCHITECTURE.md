# Architecture

## Source Structure

```
src/
├── app.go                  # App struct, bound methods, patch pipeline
├── main.go                 # Wails entry point, window setup
├── go.mod
├── go.sum
├── wails.json              # Wails config (frontend.dir, etc.)
├── Taskfile.yml            # Wails build tasks
├── cmd/
│   └── hashfile/           # CLI hash tool
├── frontend/
│   ├── dist/               # Frontend (embedded in binary)
│   │   ├── index.html      # UI structure
│   │   ├── style.css       # RO-themed styles
│   │   ├── main.js         # UI logic (polling, buttons)
│   │   └── bindings/       # Generated TypeScript bindings
└── pkg/
    ├── engine/
    │   ├── engine.go           # State machine, event emitter interface
    │   ├── state.go            # State enum + transitions
    │   ├── manifest.go         # plist.json parser, patch queue builder
    │   ├── config.go           # goro-config.json read/write
    │   ├── localstate.go       # goro-patch.json (patch tracking + validation)
    │   ├── localstate_test.go  # Local state tests
    │   ├── notes.go            # notes.json parser, markdown rendering
    │   ├── notes_test.go       # Notes tests
    │   ├── launcher.go         # Game process launch
    │   ├── launcher_linux.go   # Linux launch implementation
    │   ├── launcher_windows.go # Windows launch implementation
    │   └── rawpatcher.go       # Raw zip extraction + backup
    ├── grf/
    │   ├── grf.go              # GRF read/write/save (46-byte header)
    │   ├── filetable.go        # FileEntry struct
    │   └── patcher.go          # GRF merge + zip→GRF
    ├── downloader/
    │   ├── downloader.go       # HTTP client with resume
    │   └── hash.go             # SHA-256 verification
    ├── detector/               # Process detection
    └── updater/                # Self-update
```

## Component Interaction

```
Frontend (main.js)
    │
    │  GetProgress() polling every 100ms
    │  StartCheck(), StartRepair(), NeedsRepair(), LaunchGame(), etc.
    ▼
App (app.go)
    │
    ├── Engine (state machine)
    ├── Downloader (HTTP + hash)
    ├── GRF (read/write/merge)
    ├── Detector (process check)
    └── Updater (self-update)
```

The `App` struct is the Wails service. All methods on `App` are bound and callable from JavaScript. The `App` holds references to `Engine`, `Downloader`, etc.

## State Machine

```
IDLE ──► CHECKING ──► DOWNLOADING ──► PATCHING ──► READY
  ▲          │              │              │          │
  └──────────┴──────────────┴──────────────┴──────────┘
                            ▼
                          ERROR
```

- **IDLE**: App started, waiting for auto-check or user click
- **CHECKING**: Fetching manifest, evaluating patch queue
- **DOWNLOADING**: Downloading patch files with resume support
- **PATCHING**: Applying GRF merge or raw extraction
- **READY**: All patches applied, game can launch
- **ERROR**: Any failure, shows message with retry

Transitions are enforced in `state.go`. Invalid transitions return an error.

## Progress: Polling vs Events

Wails v3 has an event system (`app.Event.Emit`), but it has a problem: events queue through a mailbox that can't flush while a Go goroutine is CPU-bound. During GRF merge (161K files), events pile up and the UI freezes at 0% until merge completes.

**Solution**: Polling. The frontend calls `App.GetProgress()` every 100ms via `setInterval`. This is a direct RPC call that always works. The `App` writes progress to a mutex-protected struct; the frontend reads it.

Events are still used for `patch_already_updated` as a legacy fallback, but progress is entirely polling-based.

## Crash Recovery

GRF patching uses atomic rename with backup:

```
1. SaveAs(myserver.grf.patching)   — write new GRF to temp file
2. Rename(myserver.grf → .bak)     — backup original
3. Rename(.patching → myserver.grf) — activate new
4. Remove(.bak)                 — cleanup
```

If killed at any point:
- During 1: `.patching` partial, `myserver.grf` untouched → `.patching` deleted on restart
- Between 2-3: `myserver.grf` gone, `.bak` has good copy → restored on restart
- During 4: `myserver.grf` is good, `.bak` exists → deleted on restart

On startup, `CleanupStaleFiles()`:
- Deletes `.patching` and `.new` files
- Restores `.bak` if target is missing or corrupt (checks GRF magic bytes)

Hash verify failures trigger re-download: delete the patch file, download again, verify once more.

## Version = Patch ID

There is no separate `version` field in the manifest. The version IS the highest patch ID.

- `goro-patch.json` stores metadata for each applied patch (id, name, hash, size)
- `LocalVersion()` returns the highest ID from applied patches
- `BuildQueue()` returns patches where `id > localVersion`
- After applying patch N, appends patch metadata to `goro-patch.json`
- `ValidateAgainstDisk()` hashes installed GRF entries / raw files and compares them to the manifest's optional `file_hashes` for integrity (see PATCH_FORMAT.md)
- `App.FullGRFCheck` (Settings) deep-scans every patcher-owned GRF by reading all entries

This is simpler than maintaining a separate version counter. Each patch = one version bump.

## Repair

On start, the patcher validates each applied patch's hash and size against the manifest. If any mismatch is found:

1. UI button changes from "Check for Updates" to "Repair"
2. Repair fetches manifest, finds first mismatching patch ID
3. Downloads and applies all patches from that ID onward
4. Updates `goro-patch.json` after each successful patch

## File Name Validation (Security)

`patch.Name` and `patch.Target` come straight from the (unauthenticated) manifest, so before either is joined onto the game directory they pass through `engine.SafePatchComponent()` (`pkg/engine/rawpatcher.go`). The manifest is the trust root: an attacker who controls it could otherwise write or read arbitrary paths on the user's machine. Validation is **fail-closed** (an invalid name errors the patch/check) and applied at every filesystem join:

| Call site | Field | Direction |
|---|---|---|
| `App.acquirePatch` | `Name` | WRITE (patch download) |
| `App.applyPatch` (grf branch) | `Target` | WRITE (merged GRF) |
| `engine.CheckGRFIntegrity` | `Target` | READ (GRF check) |

`SafePatchComponent` requires a single bare relative filename — no path segments — and rejects:

- **Path separators** — after normalizing `\` → `/`: any `/` (covers `../` traversal, absolute paths, nested dirs, Windows drive letters like `C:\…`)
- **Empty / `.` / `..`**
- **`:`** — NTFS alternate-data-stream channel
- **Trailing `.` or ` `** — Win32 trims these and can silently alias another file
- **Windows reserved device names** — `CON`, `PRN`, `AUX`, `NUL`, `CONIN$`, `CONOUT$`, `CLOCK$`, `COM1–9`, `LPT1–9`, case-insensitive and with any extension (`NUL.txt`, `CON.grf`)

The reserved-name, colon, and trailing-dot checks apply on **every OS** (not just Windows) so patcher behavior is identical across platforms, at the cost of a Windows-relevant-only restriction on Linux. The raw patch path (`engine.PatchRaw`) separately sanitizes every zip entry name via `engine.SanitizePath`.

## Self-Update & Manifest Authenticity

The self-updater (`pkg/updater`) can replace the running executable, so it is the
sharpest edge of manifest-content trust. It is protected by four layered gates:

1. **Manifest signature (authenticity — the primary defense).** The manifest is
   the trust root and is now authenticated. `engine.SignManifest` (called by the
   publisher's `hashfile sign`) computes an **Ed25519** signature over a canonical
   form of the manifest — the struct re-marshalled with the `signature` field
   cleared. `engine.VerifyManifestSignatureWithKey` re-derives that same
   canonical form and checks it against a public key, and `engine.VerifyManifest`
   (used by `App.fetchManifest`) resolves that key at runtime via
   `engine.ResolveManifestPublicKey` — from `manifest_public_key` in
   `goro-config.json`, or the `GORO_PATCHER_PUBKEY` environment variable.
   Verification runs in `App.fetchManifest` before any field is trusted, so a
   tampered manifest can no longer influence `PatcherHash`, `PatcherURL`, or any
   patch. Because signer and verifier share the same Go canonicalization code,
   there is no cross-language skew. The public key is **not** compiled into the
   binary; in dev builds (`!release`) an absent key allows unsigned manifests
   (`MissingKeyAllowsUnsigned` → true), while release builds fail closed
   (`MissingKeyAllowsUnsigned` → false). The private key lives only with the
   publisher and is never embedded — its loss is the operator's risk, not the
   patcher's.
2. **Release-only HTTPS transport.** `downloader.validateURL` gates every fetch
   (manifest, patches, patcher binary, notes). Dev builds (`//go:build !release`)
   allow `http://`, `https://`, `file://` and local paths for testing; release
   builds (`//go:build release`) allow **only `https://`**. The channel is chosen
   at compile time via the `release` build tag (`build.sh --release`), so a
   shipped binary cannot be downgraded to plaintext.
3. **SHA-256 binary integrity.** `updater.Update` verifies the downloaded binary
   with `downloader.VerifyFile` against `PatcherHash`. Patch downloads (`app.go`),
   installed-file validation (`LocalState.Validate`, full GRF check) and self-update
   all use the unified SHA-256 integrity functions in `pkg/downloader/hash.go`.
   Because these hashes arrive inside a signed manifest they are trustworthy.
4. **Rollback gate.** `engine.NeedsSelfUpdate` requires
   `PatcherVersion > CurrentPatcherVersion` (a monotonic integer bumped per
   release in `engine/version.go`). Updates to the same or an older version are
   refused, blocking downgrade attacks.

## Notes

Optional feature. Displays markdown content in the notes section of the UI.

1. Frontend calls `App.GetNotes()` on init
2. Go fetches `notes.json` from `notes_url` in config
3. For each note entry, fetches `.md` from `notes_base_url + name`
4. Renders markdown to HTML using `gomarkdown/markdown`, then sanitizes the
   result at the AST stage (`engine.sanitizeHTMLAST`) before it is returned;
   the frontend injects it via `innerHTML`
5. Optionally fetches custom CSS from `notes_css_url`
6. Returns rendered notes (sorted by ID desc) + CSS to frontend
7. Frontend injects HTML and CSS into notes section

**Sanitization (XSS defense).** Notes are an unauthenticated content channel, so
their rendered HTML is cleaned before it reaches the DOM. `RenderMarkdown`
parses to a gomarkdown AST, drops raw-HTML nodes (`HTMLBlock`/`HTMLSpan` — the
vehicle for `<script>`, event attributes, `<img onerror>`) and keeps link/image
destinations only when they are `http(s)` URLs. So `[x](javascript:…)`,
`![x](data:…)`, raw `<script>`, and event-handler attributes all fail to render
as live markup; inline code spans are preserved because the renderer escapes
their contents. This keeps the frontend's `innerHTML` safe without hand-parsing
an HTML string (which is error-prone).

Notes are fetched on every app start. No local caching. If `notes_url` is empty, notes are disabled.

## GRF Merge

The merge happens in-memory:

1. Open target GRF (read entries into memory)
2. Open source GRF (or extract zip)
3. For each file: `target.AddFile(name, data)` — stores compressed data in `pendingData` map
4. `target.Save()` — writes everything: header + all data blocks + file table

`Save()` uses `SaveAs()` internally:
- Iterates all entries (existing + pending)
- For existing entries: reads compressed data from original file
- For pending entries: uses data from `pendingData` map
- Writes header (46 bytes) + data blocks + file table (zlib compressed)
- Atomic rename chain (see Crash Recovery)

## Wails v3 Integration

- `App` implements `EventEmitter` interface (empty stubs — events not used for progress)
- `App` has `SetApp(app *application.App)` called from `main.go` after app creation
- Frontend imports bindings from `./bindings/github.com/diviatrix/GORO-Patcher/index.js`
- Window is frameless, dragging via `--wails-draggable: drag` CSS
- Buttons use `window.wails.Window.Minimise()` and `window.wails.Application.Quit()`
- Links in notes open externally via `window.wails.Browser.OpenURL()`
- Drag-and-drop prevented via `preventDefault()` on `dragover` and `drop` events
