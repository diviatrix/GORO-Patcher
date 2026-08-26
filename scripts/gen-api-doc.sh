#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$SCRIPT_DIR/.."
BINDINGS_DIR="$PROJECT_DIR/src/frontend/dist/bindings/github.com/diviatrix/GORO-Patcher"
OUT="$PROJECT_DIR/doc/API.md"
APP_JS="$BINDINGS_DIR/app.js"

if [[ ! -f "$APP_JS" ]]; then
    echo "Error: Bindings not found at $APP_JS"
    echo "Run 'wails3 generate bindings' first."
    exit 1
fi

> "$OUT"

cat >> "$OUT" << 'EOF'
# API Reference

All methods are callable from JavaScript via the generated bindings:

```js
import { App } from './bindings/github.com/diviatrix/GORO-Patcher/index.js';
```

Methods return `CancellablePromise<T>` — use `await` to get the result.

## Typical Flow

### Normal update check

```js
// 1. Start checking for updates
await App.StartCheck();

// 2. Poll progress until complete
setInterval(async () => {
    const p = await App.GetProgress();
    updateUI(p);
    if (p.totalPercent >= 100) {
        // Done — game ready to launch
    }
}, 100);
```

### Repair flow

```js
// 1. After StartCheck() completes, check if repair is needed
const needsRepair = await App.NeedsRepair();

if (needsRepair) {
    // 2. Start repair — redownloads from first mismatching patch
    await App.StartRepair();
}
```

### Status values to watch for

| Status | Meaning |
|--------|---------|
| `"Ready (patch N)"` | All patches applied, ready to play |
| `"Up to date (patch N)"` | No new patches |
| `"Patch N corrupted — Repair needed"` | Integrity mismatch, call `StartRepair()` |
| `"Error: ..."` | Something failed |

## Data Models

### ProgressInfo

Returned by `GetProgress()`. Updated every 100ms during patching.

| Field | Type | Description |
|-------|------|-------------|
| `status` | `string` | Current status text (e.g. "Downloading...", "Ready (patch 3)") |
| `currentFile` | `string` | Currently processing filename |
| `filePercent` | `number` | Progress of current file (0–100) |
| `totalPercent` | `number` | Overall progress (0–100) |
| `speed` | `string` | Download speed or file count |

## Methods

EOF

grep -E '^export function ' "$APP_JS" | while IFS= read -r line; do
    name=$(echo "$line" | sed 's/export function \([A-Za-z]*\).*/\1/')
    params=$(echo "$line" | sed 's/.*(\(.*\)) {.*/\1/' | xargs)
    returns=$(grep -B 20 "export function $name" "$APP_JS" | grep -oP '@returns \{\K[^}]+' | tail -1)
    returns=$(echo "$returns" | sed 's/\$CancellablePromise<\(.*\)>/\1/' | sed 's/\$models\.//')

    case "$name" in
        GetProgress)     desc="Returns current patching progress state. Poll every 100ms for real-time updates." ;;
        StartCheck)      desc="Start checking for updates and applying patches. Runs asynchronously." ;;
        StartRepair)     desc="Start repair -- redownloads and reapplies patches from first mismatch. Runs asynchronously." ;;
        NeedsRepair)     desc="Returns whether patch integrity check detected corruption. Call after StartCheck() completes." ;;
        LaunchGame)      desc="Launch the game executable. Fails if game is already running." ;;
        CancelDownload)  desc="Cancel the current download/patch operation." ;;
        GetLocalVersion) desc="Returns the highest applied patch ID from local state." ;;
        GetManifestURL)  desc="Returns the current manifest URL from config." ;;
        SetManifestURL)  desc="Set the manifest URL and save to config. Supports file://, http://, https://." ;;
        GetExeName)      desc="Returns the game executable name from config." ;;
        SetExeName)      desc="Set the game executable name and save to config." ;;
        IsGameRunning)   desc="Returns whether the game process is currently running." ;;
        FullGRFCheck)    desc="Run a full integrity check over all GRF targets referenced by applied patches." ;;
        *)               desc="" ;;
    esac

    echo "### \`${name}()\`" >> "$OUT"
    echo "" >> "$OUT"
    echo "$desc" >> "$OUT"
    echo "" >> "$OUT"

    if [[ -n "$params" ]]; then
        echo "- **Parameters**: \`${params}\`" >> "$OUT"
    else
        echo "- **Parameters**: none" >> "$OUT"
    fi

    if [[ -n "$returns" && "$returns" != "void" ]]; then
        echo "- **Returns**: \`${returns}\`" >> "$OUT"
    else
        echo "- **Returns**: \`void\`" >> "$OUT"
    fi

    echo "" >> "$OUT"
done

echo "Generated: $OUT"
