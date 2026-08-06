# User Guide

Guide for RO server administrators setting up GORO-Patcher for their players.

## What You Need

- A web server (or CDN) to host patch files
- Your game client files (GRF, executable)
- GORO-Patcher binary (download or [build](BUILD.md))

## Step 1: Set Up Your Patch Server

Static file host only. No server-side code needed.

```
https://patches.yourserver.com/
├── plist.json              # Manifest
├── patch_0.grf             # First patch
├── patch_1.zip             # Second patch
└── patch_2.grf             # Third patch
```

nginx, Apache, S3, Cloudflare, GitHub Pages — anything works.

## Step 2: Create Patches

### GRF patches (sprites, models, textures)

A GRF patch is a standard GRF 2.0 file containing the files you want to add or replace.

**Creating a GRF patch file:**

1. Use a GRF editor (e.g. GRF Editor, Thor Patcher) to create a new GRF
2. Add your files to it (sprites, textures, models, sounds)
3. Save as a `.grf` file

The patcher merges this GRF into the player's existing GRF. Files with the same path are replaced; new files are added.

**Example contents:**
```
patch_0.grf
├── data\texture\effect\new_effect.bmp
├── data\sprite\npc\custom_npc.spr
└── data\model\building\new_house.rsm
```

### Raw patches (exe, dll, lub files)

A raw patch is a zip archive containing files to extract directly to the game directory.

**Creating a raw patch:**

1. Create a zip file
2. Add files with the correct folder structure
3. Save as `.zip`

**Example contents:**
```
patch_1.zip
├── System/itemInfo.lub
├── dinput.dll
└── data/sprite/poring.spr
```

Files are extracted to the game folder. Existing files are backed up to `.bak` before overwrite.

### When to use which

| Type | Use for | `target` field |
|------|---------|----------------|
| `grf` | Sprites, models, textures, sounds — files inside GRF | Your GRF filename (e.g. `sdata.grf`, `custom.grf`) |
| `raw` | Exe, DLLs, LUBs — files outside GRF | Relative path in game dir |

## Step 3: Calculate Hash and Size

Use the included `hashfile` tool to generate XXHash64 hashes:

```bash
# Linux
./build/hashfile patch_0.grf

# Windows
build\hashfile.exe patch_0.grf
```

Output: `a899ed08439de698  200627214  patch_0.grf`
- First column: hash (16-char hex)
- Second column: file size in bytes
- Third column: filename

Copy the hash and size into your manifest.

## Step 4: Create the Manifest

`plist.json`:

```json
{
  "patch_base_url": "https://patches.yourserver.com/",
  "patches": [
    {
      "id": 1,
      "name": "patch_0.grf",
      "hash": "a899ed08439de698",
      "size": 200627214,
      "type": "grf",
      "target": "sdata.grf"
    }
  ]
}
```

| Field | Description |
|-------|-------------|
| `patch_base_url` | Where patch files are hosted |
| `patches[].id` | Sequential number. IS the version — no separate version field. |
| `patches[].name` | Filename on server |
| `patches[].hash` | XXHash64, lowercase hex, 16 chars |
| `patches[].size` | File size in bytes |
| `patches[].type` | `"grf"` or `"raw"` |
| `patches[].target` | **Your GRF filename** or relative path — whatever your server uses |

The `target` field is the file that gets patched. Use whatever GRF name your server uses (e.g. `sdata.grf`, `custom.grf`, `data.grf`, etc.). The patcher creates it if it doesn't exist.

## Step 5: Configure the Patcher

`goro-config.json` (placed next to the patcher binary):

```json
{
  "manifest_url": "https://patches.yourserver.com/plist.json",
  "exe_name": "your-client.exe"
}
```

## Step 6: Distribute to Players

Package together:
1. Patcher binary (`GORO-Patcher` or `GORO-Patcher.exe`)
2. `goro-config.json`
3. Your base GRF file (or let the patcher create it on first run)

Players just run the patcher. It checks for updates, downloads, applies, and enables "Launch Game".

## Adding Updates

1. Create new patch file
2. Calculate hash: `./build/hashfile patch_1.grf`
3. Add entry to `plist.json` with next `id`
4. Upload both to your patch server

```json
{
  "patch_base_url": "https://patches.yourserver.com/",
  "patches": [
    { "id": 1, "name": "patch_0.grf", "hash": "a899ed08439de698", "size": 200627214, "type": "grf", "target": "sdata.grf" },
    { "id": 2, "name": "patch_1.zip", "hash": "ca7c90486d2f7a42", "size": 542, "type": "raw", "target": "System/itemInfo.lub" }
  ]
}
```

Players who already applied patch 1 only download patch 2.

## Self-Update (Optional)

Add to manifest:

```json
{
  "patcher_url": "https://patches.yourserver.com/GORO-Patcher.exe",
  "patcher_hash": "abc123...",
  "patcher_size": 16376064,
  "patch_base_url": "...",
  "patches": [...]
}
```

Patcher compares its own hash and updates if different.

## Branding (Optional)

1. Change window title in `src/main.go`
2. Edit styles in `src/frontend/style.css`
3. Modify layout in `src/frontend/index.html`
4. Rebuild: `./build.sh`

## Crash Recovery

If patcher crashes or power is lost during patching, just restart. Recovery is automatic:

- Partial download → re-downloaded
- Interrupted merge → original restored from backup
- Corrupt file → restored from backup

## Local Testing

Use `file://` URLs and absolute paths:

```json
// goro-config.json
{
  "manifest_url": "file:///home/you/patches/plist.json",
  "exe_name": "your-client.exe"
}
```

```json
// plist.json
{
  "patch_base_url": "/home/you/patches/data/",
  "patches": [...]
}
```

## Running under Wine/Proton

GORO-Patcher uses Wails v3, which requires Microsoft WebView2 on Windows. Wine/Proton doesn't include it by default. Install it using one of these methods.

### Method 1: winetricks

```bash
# Set your Wine prefix
export WINEPREFIX=~/.wine

# Install WebView2
winetricks webview2

# Run the patcher
wine GORO-Patcher.exe
```

For Proton prefixes (e.g. Steam):
```bash
export WINEPREFIX=~/.local/share/Steam/steamapps/compatdata/<appid>/pfx
winetricks webview2
```

### Method 2: Heroic Games Launcher

Heroic uses protontricks or winetricks with the game's prefix.

**Find your prefix:**
- Default: `~/Games/Heroic/Prefixes/<game-name>`
- Check: Heroic → Game Settings → Wine → Prefix path

**Install via protontricks:**
```bash
# Find the app ID (for non-Steam games, protontricks uses UMU IDs)
protontricks --list

# Install WebView2
protontricks <appid> webview2
```

**Install via winetricks (with Heroic's prefix):**
```bash
WINEPREFIX="/home/$USER/Games/Heroic/Prefixes/<game-name>" winetricks webview2
```

**Install via "Run EXE on Prefix":**
1. Download [MicrosoftEdgeWebview2Setup.exe](https://go.microsoft.com/fwlink/p/?LinkId=2124703)
2. In Heroic: Game Settings → Other → Run EXE on Prefix
3. Select the downloaded file

### Method 3: Lutris

**Via Wine configuration:**
1. Right-click game → Wine configuration
2. Install WebView2 from there

**Via Run EXE inside prefix:**
1. Download [MicrosoftEdgeWebview2Setup.exe](https://go.microsoft.com/fwlink/p/?LinkId=2124703)
2. Right-click game → Run EXE inside Wine prefix
3. Select the downloaded file

**Via winetricks:**
```bash
WINEPREFIX="/home/$USER/Games/<game-name>" winetricks webview2
```

### Download

Microsoft Edge WebView2 Evergreen Bootstrapper:
https://go.microsoft.com/fwlink/p/?LinkId=2124703

## Troubleshooting

| Problem | Solution |
|---------|----------|
| "Hash mismatch" | Regenerate hash with `hashfile`, update `plist.json` |
| Stays at "Starting..." | Check `manifest_url` is accessible |
| Files not merging | Ensure `type` is `"grf"` and `target` matches your GRF |
| Crashes during merge | Restart — recovery restores automatically |
| Game won't launch | Check `exe_name` matches your executable |
| Won't start on Wine/Proton | Install WebView2 — see [Running under Wine/Proton](#running-under-wineproton) |
