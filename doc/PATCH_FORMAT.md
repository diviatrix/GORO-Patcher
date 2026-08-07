# Patch Format Reference

## Manifest (plist.json)
plist.json is a file located in remote - basically your patch hosting/server.
It contains the list of patches available for download with hashes, sizes, types and it is source of truth for patch availability.
plist.json address for patcher itself is specified in `goro-config.json`.

```json
{
  "patch_base_url": "https://your-server.com/patches/",
  "patches": [
    {
      "id": 1,
      "name": "patch_0.grf",
      "hash": "a899ed08439de698",
      "size": 200627214,
      "type": "grf",
      "target": "data.grf"
    },
    {
      "id": 2,
      "name": "patch_1.zip",
      "hash": "ca7c90486d2f7a42",
      "size": 542,
      "type": "raw",
      "target": "System/itemInfo.lub"
    }
  ]
}
```

### Fields

| Field | Description |
|-------|-------------|
| `patch_base_url` | Base URL for downloading patch files. |
| `patches[].id` | Unique patch ID. Also the version number. Must be sequential. |
| `patches[].name` | Patch filename (downloaded from `patch_base_url + name`) |
| `patches[].hash` | XXHash64 of the file (16-char hex) |
| `patches[].size` | File size in bytes |
| `patches[].type` | `grf` or `raw` |
| `patches[].target` | Target file for GRF merge, or relative path for raw extract |

### Version = Patch ID

Both `manifest_url` (in `goro-config.json`) and `patch_base_url` support `file://`, `http://`, and `https://` protocols.

There is no separate `version` field. The latest version is the highest `id` in the patches array.

- Patch `id: 1` = version 1
- Patch `id: 2` = version 2
- Client tracks applied patches in `goro-patch.json` (see below)

Only patches with `id > local_version` are applied.

## Local State (goro-patch.json)

The patcher tracks applied patches in `goro-patch.json` (next to the patcher binary). This file stores metadata for each applied patch, enabling integrity validation against the manifest.

```json
{
  "applied_patches": [
    { "id": 0, "name": "patch_0.grf", "hash": "a2ac70ab8dff9e89", "size": 2043 },
    { "id": 1, "name": "patch_1.zip", "hash": "45694d77d7281b59", "size": 539 }
  ]
}
```

### Validation

On start, the patcher compares each applied patch's `hash` and `size` against the manifest (`plist.json`). If any mismatch is found (corrupted file, changed manifest, missing patch), the UI shows a **Repair** button.

### Repair

Redownloads and reapplies all patches from the first mismatching ID onward. Available in two places:
- **Main UI** — button changes from "Check for Updates" to "Repair" when mismatch detected
- **Settings** — "Repair Patches" button

## GRF Patches (`type: "grf"`)

Merges content into target GRF file.

### Format 1: Direct GRF (`.grf`)

Patch file is a GRF. Merged directly into target. Each file in the patch GRF is added/replaced in the target.

TODO: GRF password/encryption is not implemented yet.

### Format 2: Zip archive (`.zip`)

Patch file is a zip containing files to merge into target GRF. Paths are normalized automatically.

```
patch.zip
├── data/texture/sprite.bmp
├── data/luafiles514/iteminfo.lub
└── data/sprite/monster/poring.act
```

### Behavior

- Creates empty GRF if target doesn't exist
- Existing files in target are replaced
- New files are added
- Atomic save: writes to `.patching` temp, renames on completion
- `.bak` backup created before overwrite; cleaned up on success

## Raw Patches (`type: "raw"`)

Extracts zip contents directly to game directory. Use for files that live outside the GRF — executables, DLLs, configs, LUB scripts, etc.

Supported paths:
- **Root files** — `dinput.dll`, `your-client.exe`, `settings.ini`
- **Subdirectories** — `System/itemInfo.lub`, `data/sprite/poring.spr`
- **Any depth** — `data/luafiles514/lua files/skillinfo.lub`

```
patch.zip
├── System/itemInfo.lub
├── dinput.dll
├── data/sprite/poring.spr
└── data/luafiles514/lua files/skillinfo.lub
```

### Behavior

- Existing files backed up to `.bak` before overwrite
- Directories created as needed
- Preserves file permissions (executable bit)
- Paths normalized automatically (backslashes converted to forward slashes)
- Leading `../` and `/` stripped for safety (no directory traversal)

## Hash Generation

XXHash64, lowercase hex, 16 characters.

```go
import "github.com/diviatrix/GORO-Patcher/pkg/downloader"

hash, err := downloader.HashFile("path/to/patch.grf")
// hash = "a899ed08439de698"
```

## Example: Adding a New Patch

1. Create the patch file (GRF or zip)
2. Generate hash: `downloader.HashFile("patch.grf")`
3. Get file size: `stat -c%s patch.grf`
4. Add entry to `plist.json` with next sequential `id`
5. Upload patch file to `patch_base_url`
6. Upload updated `plist.json` to manifest URL
