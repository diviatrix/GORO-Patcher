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
| `patches[].file_hashes` | **Optional.** Map of entry/file → XXHash64 of its content. For `grf` the keys are GRF entry paths (content hash); for `raw` the keys are extracted relative paths (file hash). Enables installed-file verification. Omitted (empty) when a patch writes nothing new. |
| `patches[].mutable` | **Optional.** List of `raw` relative paths the game rewrites at runtime. Those are verified at **apply time only** and excluded from the **startup re-check** (avoids false "corrupt → repair"). |

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

The `hash`/`size` fields verify the **downloaded archive** (checked at download time against the patch file).

The optional `file_hashes` verifies the **installed result on disk**:

- **grf**: each GRF entry the patch writes is read back from the target GRF and its content hash is compared to `file_hashes`. Content is base-independent (dictated by the patch), so this works regardless of GRF serialization details.
- **raw**: each extracted file is hashed and compared to `file_hashes` of its **highest writer** (last patch that wrote it, last-wins).
- `mutable` raw files are checked at **apply time only**, not on the startup re-check, so runtime-rewritten configs/lub/ini don't trigger false repairs.
- Patches **without** `file_hashes` are verified by archive hash only — no installed check, no failure reported.
- An applied patch **pruned** from the manifest is treated as history, not an error.
- Raw `file_hashes` keys that fail path sanitization (e.g. `../`) are ignored.

If any installed check fails (missing/corrupt file, changed content), the UI shows a **Repair** button from the first failing patch.

### Full GRF Check (Settings)

`App.FullGRFCheck` (Settings → Full GRF Check) performs a deep scan of every patcher-owned GRF: it opens the GRF and reads the content of **all** entries, reporting any that fail. This catches corruption in entries that no patch ever wrote (which per-patch `file_hashes` deliberately does not). The result is surfaced in the status line.

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

### Authoring installed-result hashes (`gen-plist`)

`src/cmd/gen-plist` fills the optional `file_hashes` field by hashing the **patch's own content** (entry content for `grf`, extracted file bytes for `raw`) using the patcher's own `pkg/grf`/`pkg/engine` code. No server-side apply is needed because the installed entry/file content is dictated by the patch:

```sh
go run ./cmd/gen-plist -in plist.json -patches /path/to/patches -out plist.json
```

- For each grf patch it records `{ grf_entry_path: content_hash }` for every entry the patch writes; an empty patch produces an empty map (not a version boundary).
- For each raw patch it extracts and hashes every file as `file_hashes`.
- The tool emits the original plist with the new fields; `hash`/`size` are left untouched. `hashfile` remains the hasher for patch archives.

## Notes (notes.json)

Optional. Displays markdown notes in the patcher UI. Configured via `notes_url` in `goro-config.json`.

### Configuration

`notes.json`:
```json
{
  "notes_base_url": "https://your-server.com/notes/",
  "notes_css_url": "https://your-server.com/notes/design.css",
  "notes": [
    { "id": 0, "name": "note_0.md" },
    { "id": 1, "name": "note_1.md" }
  ]
}
```

### Fields

| Field | Description |
|-------|-------------|
| `notes_base_url` | Base URL for note files. Appended with `name`. |
| `notes_css_url` | Optional. Custom CSS for note styling. Empty or omitted = no overrides. |
| `notes[].id` | Note ID. Used for sorting — highest ID displayed first. |
| `notes[].name` | Markdown filename (fetched from `notes_base_url + name`). |

### Protocol Support

`notes_base_url` and `notes_css_url` support `file://`, `http://`, and `https://` protocols. Bare file paths (without `file://`) are not supported.

### Display Order

Notes are sorted by ID descending — latest note appears at the top.

### Links

Links in notes open in the user's default external browser, not inside the patcher window.

### Custom CSS

If `notes_css_url` is set, the CSS is injected into the notes section. It can override any HTML element styling within notes. If empty or omitted, browser defaults are used.

### Markdown

Standard CommonMark: headings, bold, italic, links, lists, code blocks, blockquotes, tables, images.

## Example: Adding a New Patch

1. Create the patch file (GRF or zip)
2. Generate hash: `downloader.HashFile("patch.grf")`
3. Get file size: `stat -c%s patch.grf`
4. Add entry to `plist.json` with next sequential `id`
5. Upload patch file to `patch_base_url`
6. Upload updated `plist.json` to manifest URL
