# User Guide

Guide for RO server administrators setting up GORO-Patcher for their players.

## What You Need

- A web server (or CDN) to host patch files over **HTTPS**
- Your game client files (GRF, executable)
- A GORO-Patcher **release binary that you build yourself** for your server —
  there is no generic pre-built release binary (see [BUILD.md](BUILD.md))
- The `hashfile` tool (built alongside the patcher) to hash, **sign**, and
  verify your manifest

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
| `grf` | Sprites, models, textures, sounds — files inside GRF | Your GRF filename (e.g. `myserver.grf`, `custom.grf`) |
| `raw` | Exe, DLLs, LUBs — files outside GRF | Relative path in game dir |

## Step 3: Calculate Hash and Size

Use the included `hashfile` tool to generate SHA-256 hashes:

```bash
# Linux
./build/hashfile patch_0.grf

# Windows
build\hashfile.exe patch_0.grf
```

Output: `<sha256 hex, 64 chars>  200627214  patch_0.grf`
- First column: hash (64-char hex)
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
      "hash": "GENERATED_HASH",
      "size": 200627214,
      "type": "grf",
      "target": "myserver.grf"
    }
  ]
}
```

| Field | Description |
|-------|-------------|
| `patch_base_url` | Where patch files are hosted |
| `patches[].id` | Sequential number. IS the version — no separate version field. |
| `patches[].name` | Filename on server |
| `patches[].hash` | SHA-256, lowercase hex, 64 chars |
| `patches[].size` | File size in bytes |
| `patches[].type` | `"grf"` or `"raw"` |
| `patches[].target` | **Your GRF filename** or relative path — whatever your server uses |

The `target` field is the file that gets patched. Use whatever GRF name your server uses (e.g. `myserver.grf`, `custom.grf`, etc.). The patcher creates it if it doesn't exist.

### Sign the manifest

A **release** patcher refuses an unsigned or tampered manifest, so you must
sign yours before uploading. This requires the keypair matching the public key
you configure in Step 5.

```bash
# One-time setup: generate an Ed25519 keypair (keep the private key secret)
./build/hashfile genkey -out keys/key.pem -pub keys/pub.pem
#   -> prints the base64 public key (save it for goro-config.json)

# Sign the manifest (sign -in rewrites plist.json in place); then check it
./build/hashfile sign -key keys/key.pem -in plist.json
./build/hashfile verify -key keys/pub.pem -in plist.json
#   -> "signature OK"
```

On Windows use `build\hashfile.exe`. Re-sign the manifest after every edit.

## Step 5: Configure the Patcher

`goro-config.json` (placed next to the patcher binary):

```json
{
  "manifest_url": "https://patches.yourserver.com/plist.json",
  "exe_name": "your-client.exe",
  "notes_url": "https://patches.yourserver.com/notes.json",
  "manifest_public_key": "PASTE_BASE64_PUBLIC_KEY_FROM_GENKEY"
}
```

| Field | Required | Description |
|-------|----------|-------------|
| `manifest_url` | Yes | URL to your `plist.json` — for a release build this **must be `https://`** |
| `exe_name` | Yes | Game executable name |
| `notes_url` | No | URL to `notes.json` for displaying notes |
| `manifest_public_key` | Yes* | Base64 Ed25519 public key the manifest must match. **Required** for a release build (it refuses to trust the manifest otherwise). May also be set via the `GORO_PATCHER_PUBKEY` environment variable, which overrides config. |

Alternatively set `GORO_PATCHER_PUBKEY` to the same base64 string.

## Step 6: Distribute to Players

Package together:
1. Your **release-built** patcher binary (`GORO-Patcher` or `GORO-Patcher.exe`)
2. `goro-config.json` — containing `manifest_url` (`https://`) and
   `manifest_public_key` for your server
3. Your base GRF file (or let the patcher create it on first run)

Players just run the patcher. It checks for updates, downloads, applies, and enables "Launch Game".

If you distribute a **dev** build instead, it skips the signature check and
accepts `http://`/`file://` — convenient for testing, but it has no tamper
protection and must never be given to production players over anything but a
trusted transport you control.

## Adding Updates

1. Create new patch file
2. Calculate hash: `./build/hashfile patch_1.grf`
3. Add entry to `plist.json` with next `id`
4. Re-sign the manifest: `./build/hashfile sign -key keys/key.pem -in plist.json`
5. Upload the patch and the signed manifest to your patch server

```json
{
  "patch_base_url": "https://patches.yourserver.com/",
  "patches": [
    { "id": 1, "name": "patch_0.grf", "hash": "GENERATED_HASH", "size": 200627214, "type": "grf", "target": "myserver.grf" },
    { "id": 2, "name": "patch_1.zip", "hash": "GENERATED_HASH", "size": 542, "type": "raw", "target": "System/itemInfo.lub" }
  ]
}
```

Players who already applied patch 1 only download patch 2.

## Self-Update (Optional)

Add to manifest:

```json
{
  "patcher_url": "https://patches.yourserver.com/GORO-Patcher.exe",
  "patcher_hash": "<sha256 of the patcher binary, 64 hex>",
  "patcher_version": 2,
  "patch_base_url": "...",
  "patches": [...]
}
```

| Field | Description |
|-------|-------------|
| `patcher_url` | HTTPS URL of the new patcher binary |
| `patcher_hash` | SHA-256 of that binary (compute with `hashfile sha256`) |
| `patcher_version` | Version of that binary — must **strictly exceed** the running patcher's `CurrentPatcherVersion` (`src/pkg/engine/version.go`). Same-or-older versions are refused, so the feed cannot force a downgrade. |

The patcher compares its own version/hash and, when a newer signed release is
offered, downloads, verifies, and replaces itself. Bump `CurrentPatcherVersion`
in `src/pkg/engine/version.go` and set the same integer here whenever you ship a
new patcher.

## Notes (Optional)

Display markdown notes (news, patch notes, announcements) in the patcher UI.

### Setup

1. Create `notes.json` and host it on your patch server
2. Add `notes_url` to `goro-config.json`
3. Create markdown files and host them alongside

**notes.json:**
```json
{
  "notes_base_url": "https://patches.yourserver.com/notes/",
  "notes_css_url": "https://patches.yourserver.com/notes/design.css",
  "notes": [
    { "id": 0, "name": "welcome.md" },
    { "id": 1, "name": "patch_v2.md" }
  ]
}
```

**notes/welcome.md:**
```markdown
# Welcome

Welcome to the server! Talk to the Tutorial NPC to get started.
```

### Display Order

Notes are sorted by ID descending. Highest ID (newest) appears at the top.

### Custom CSS

Create a CSS file and set `notes_css_url` to style your notes. The CSS is scoped to the notes section and can override any HTML element rendered from markdown.

```css
h1 { color: #f3e0b5; border-bottom: 1px solid #4a2f1b; }
blockquote { border-left: 3px solid #c5a059; color: #a89070; }
```

If `notes_css_url` is empty or omitted, browser defaults are used.

### Protocols

`notes_base_url` and `notes_css_url` support `file://`, `http://`, and `https://`.

## Branding (Optional)

1. Change window title in `src/main.go`
2. Edit styles in `src/frontend/style.css`
3. Modify layout in `src/frontend/index.html`
4. Rebuild: `./scripts/build.sh`

## Crash Recovery

If patcher crashes or power is lost during patching, just restart. Recovery is automatic:

- Partial download → re-downloaded
- Interrupted merge → original restored from backup
- Corrupt file → restored from backup

## Repair

If patch files become corrupted or the manifest changes, the patcher detects mismatches on start and shows a **Repair** button. Repair redownloads and reapplies patches from the first mismatching one onward.

**When repair triggers:**
- Patch file hash doesn't match manifest
- Patch file size doesn't match manifest
- Applied patch no longer exists in manifest

**How to use:**
- **Main UI** — click "Repair" (appears instead of "Check for Updates" when mismatch detected)
- **Settings** — click "Repair Patches"

The patcher tracks applied patch metadata in `goro-patch.json`. See [PATCH_FORMAT.md](PATCH_FORMAT.md) for the file format.

## Local Testing

Test patches locally before deploying to a remote server.
To speed up testing, use local `file://` URLs and absolute paths.

> **This requires a dev build.** A release build allows only `https://` and
> demands a signed manifest, so use a dev binary (`./scripts/build.sh linux`)
> for `file://`/`http://` local testing. No signature is needed in dev.

### Setup

Create a folder structure:

```
~/test-patches/
├── plist.json
└── data/
    ├── patch_0.grf
    └── patch_1.zip
```

### Step 1: Create a test patch

Prepare your GRF or ZIP file and place it in `~/test-patches/data/`.
You can use test GRF from `example/data/patch_0.grf` or create your own.

### Step 2: Calculate hash

```bash
cd ~/test-patches/data/
/path/to/build/hashfile patch_0.grf
# Output: <sha256 hex, 64 chars>  200627214  patch_0.grf
```

### Step 3: Create manifest

`~/test-patches/plist.json`:

`patch_base_url` can use `file://` `http://` and `https://` protocols. 
Patches referenced from `data/` subfolder.

```json
{
  "patch_base_url": "file:///home/you/test-patches/data/",
  "patches": [
    {
      "id": 0,
      "name": "patch_0.grf",
      "hash": "GENERATED_HASH_64_HEX",
      "size": 2043,
      "type": "grf",
      "target": "server.grf"
    },
    {
      "id": 1,
      "name": "patch_1.zip",
      "hash": "GENERATED_HASH_64_HEX",
      "size": 539,
      "type": "raw",
      "target": "server.grf"
    }
  ]
}
```

### Step 4: Configure patcher

`goro-config.json` (next to patcher binary):

```json
{
  "manifest_url": "file:///home/you/test-patches/plist.json",
  "exe_name": "your-client.exe"
}
```

### Step 5: Run

```bash
cd /path/to/build
./GORO-Patcher
```

### Notes

- `patch_base_url` must end with `/`
- Paths are absolute — `file:///home/you/...` (three slashes)
- On Windows: `file:///C:/Users/you/test-patches/data/`
- Each patch is applied in order by `id`
- First run applies all patches; subsequent runs skip already-applied ones

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
| Manifest rejected / signature fails (release build) | Re-sign `plist.json` (`hashfile sign -key key.pem -in plist.json`) with the key whose public half is in `manifest_public_key`, or fix `manifest_public_key`; re-upload the signed manifest |
| Content refused unless `https://` (release build) | Host every URL (`manifest_url`, `patch_base_url`, `patcher_url`, notes) over HTTPS — release builds reject other schemes; use a dev build only for local testing |
| Stays at "Starting..." | Check `manifest_url` is accessible |
| "Repair needed" shown | Click Repair — patch files don't match manifest |
| Files not merging | Ensure `type` is `"grf"` and `target` matches your GRF |
| Crashes during merge | Restart — recovery restores automatically |
| Game won't launch | Check `exe_name` matches your executable |
| Won't start on Wine/Proton | Install WebView2 — see [Running under Wine/Proton](#running-under-wineproton) |
