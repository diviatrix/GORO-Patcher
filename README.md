# GORO-Patcher

Cross-platform Ragnarok Online patcher. Merges GRF files and extracts raw patches.

Project page: https://goro.1337.plus/

[![Patcher UI](doc/img/patcher.jpg)](doc/img/patcher.jpg)

## Features

- GRF 2.0 merge (direct `.grf` or `.zip` → GRF)
- Raw patch extraction (zip → game directory)
- HTTP downloads with resume support
- SHA-256 verification everywhere
- Crypto-signed manifests (Ed25519) + HTTPS-only release transports
- Self-update with a no-downgrade version gate
- Crash-safe atomic GRF writes
- Auto-check on start
- Patch integrity validation + Repair
- Markdown notes (patch notes, news, announcements)
- Linux and Windows builds

## Quick Start

1. Build: `./scripts/build.sh linux` or `./scripts/build.sh windows`
2. Binaries output to `build/` — copy `build/` to your server
3. Create `goro-config.json` (see [User Guide](doc/USER_GUIDE.md))
4. Run

## Documentation

- **[User Guide](doc/USER_GUIDE.md)** — setup, configuration, patching workflow (admins + players)
- **[API Reference](doc/API.md)** — JavaScript API for custom patcher UIs
- [Architecture](doc/ARCHITECTURE.md) — how the code works, design decisions
- [Build Guide](doc/BUILD.md) — prerequisites, build commands
- [Patch Format](doc/PATCH_FORMAT.md) — manifest and patch types
- [Changelog](doc/CHANGELOG.md) — version history

## License
See [LICENSE](LICENSE).
