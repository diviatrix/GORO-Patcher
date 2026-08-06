# Patch Format Reference

## Manifest (plist.json)

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
| `patch_base_url` | Base URL for downloading patch files |
| `patches[].id` | Unique patch ID. Also the version number. Must be sequential. |
| `patches[].name` | Patch filename (downloaded from `patch_base_url + name`) |
| `patches[].hash` | XXHash64 of the file (16-char hex) |
| `patches[].size` | File size in bytes |
| `patches[].type` | `grf` or `raw` |
| `patches[].target` | Target file for GRF merge, or relative path for raw extract |

### Version = Patch ID

There is no separate `version` field. The latest version is the highest `id` in the patches array.

- Patch `id: 1` = version 1
- Patch `id: 2` = version 2
- Client tracks last applied ID in `goro-patch.json`

Only patches with `id > local_version` are applied.

## GRF Patches (`type: "grf"`)

Merges content into target GRF file.

### Format 1: Direct GRF (`.grf`)

Patch file is a GRF. Merged directly into target. Each file in the patch GRF is added/replaced in the target.

### Format 2: Zip archive (`.zip`)

Patch file is a zip. Files are extracted and added to target GRF.

Zip structure uses backslash paths (GRF convention):
```
patch.zip
├── data\texture\sprite.bmp
├── data\luafiles514\iteminfo.lub
└── data\sprite\monster\poring.act
```

### Behavior

- Creates empty GRF if target doesn't exist
- Existing files in target are replaced
- New files are added
- Atomic save: writes to `.patching` temp, renames on completion
- `.bak` backup created before overwrite; cleaned up on success

## Raw Patches (`type: "raw"`)

Extracts zip contents to game directory.

Zip structure uses forward slashes:
```
patch.zip
├── System/itemInfo.lub
├── dinput.dll
└── data/sprite/poring.spr
```

### Behavior

- Existing files backed up to `.bak` before overwrite
- Directories created as needed
- Preserves file permissions (executable bit)

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
