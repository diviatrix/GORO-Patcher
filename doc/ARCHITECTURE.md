# Architecture

## Source Structure

```
src/
├── app.go              # App struct, bound methods, patch pipeline
├── main.go             # Wails entry point, window setup
├── go.mod
├── frontend/
│   ├── index.html      # UI structure
│   ├── style.css       # Styles
│   ├── src/main.js     # UI logic (polling, buttons, events)
│   └── dist/           # Built frontend (embedded in binary)
└── pkg/
    ├── engine/
    │   ├── engine.go       # State machine, event emitter interface
    │   ├── state.go        # State enum + transitions
    │   ├── manifest.go     # plist.json parser, patch queue builder
    │   ├── config.go       # goro-config.json read/write
    │   ├── localstate.go   # goro-patch.json (version tracking)
    │   ├── launcher.go     # Game process launch
    │   └── rawpatcher.go   # Raw zip extraction + backup
    ├── grf/
    │   ├── grf.go          # GRF read/write/save (46-byte header)
    │   ├── filetable.go    # FileEntry struct
    │   └── patcher.go      # GRF merge + zip→GRF
    ├── downloader/
    │   ├── downloader.go   # HTTP client with resume
    │   └── hash.go         # XXHash64 verification
    ├── detector/           # Process detection
    └── updater/            # Self-update
```

## Component Interaction

```
Frontend (main.js)
    │
    │  GetProgress() polling every 100ms
    │  StartCheck(), LaunchGame(), etc.
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
1. SaveAs(data.grf.patching)   — write new GRF to temp file
2. Rename(data.grf → .bak)     — backup original
3. Rename(.patching → data.grf) — activate new
4. Remove(.bak)                 — cleanup
```

If killed at any point:
- During 1: `.patching` partial, `data.grf` untouched → `.patching` deleted on restart
- Between 2-3: `data.grf` gone, `.bak` has good copy → restored on restart
- During 4: `data.grf` is good, `.bak` exists → deleted on restart

On startup, `CleanupStaleFiles()`:
- Deletes `.patching` and `.new` files
- Restores `.bak` if target is missing or corrupt (checks GRF magic bytes)

Hash verify failures trigger re-download: delete the patch file, download again, verify once more.

## Version = Patch ID

There is no separate `version` field in the manifest. The version IS the highest patch ID.

- `goro-patch.json` stores the last applied patch ID
- `BuildQueue()` returns patches where `id > localVersion`
- After applying patch N, writes N to `goro-patch.json`

This is simpler than maintaining a separate version counter. Each patch = one version bump.

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
