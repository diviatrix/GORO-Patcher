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

### `CancelDownload()`

Cancel the current download/patch operation.

- **Parameters**: none
- **Returns**: `void`

### `GetExeName()`

Returns the game executable name from config.

- **Parameters**: none
- **Returns**: `string`

### `GetLocalVersion()`

Returns the highest applied patch ID from local state.

- **Parameters**: none
- **Returns**: `number`

### `GetManifestURL()`

Returns the current manifest URL from config.

- **Parameters**: none
- **Returns**: `string`

### `GetProgress()`

Returns current patching progress state. Poll every 100ms for real-time updates.

- **Parameters**: none
- **Returns**: `ProgressInfo`

### `IsGameRunning()`

Returns whether the game process is currently running.

- **Parameters**: none
- **Returns**: `boolean`

### `LaunchGame()`

Launch the game executable. Fails if game is already running.

- **Parameters**: none
- **Returns**: `void`

### `NeedsRepair()`

Returns whether patch integrity check detected corruption. Call after StartCheck() completes.

- **Parameters**: none
- **Returns**: `boolean`

### `SetExeName()`

Set the game executable name and save to config.

- **Parameters**: `name`
- **Returns**: `void`

### `SetManifestURL()`

Set the manifest URL and save to config. Supports file://, http://, https://.

- **Parameters**: `url`
- **Returns**: `void`

### `StartCheck()`

Start checking for updates and applying patches. Runs asynchronously.

- **Parameters**: none
- **Returns**: `void`

### `StartRepair()`

Start repair -- redownloads and reapplies patches from first mismatch. Runs asynchronously.

- **Parameters**: none
- **Returns**: `void`

