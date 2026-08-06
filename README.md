# GORO-Patcher
<<<<<<< HEAD

Cross-platform Ragnarok Online patcher. Merges GRF files and extracts raw patches.

## Features

- GRF 2.0 merge (direct `.grf` or `.zip` → GRF)
- Raw patch extraction (zip → game directory)
- HTTP downloads with resume support
- XXHash64 verification
- Crash-safe atomic GRF writes
- Auto-check on start
- Linux and Windows builds

## Quick Start

1. Build: `./build.sh linux` or `./build.sh windows`
2. Copy binary to build folder
3. Create `goro-config.json` (see [User Guide](doc/USER_GUIDE.md))
4. Run

## Documentation

- **[User Guide](doc/USER_GUIDE.md)** — setup, configuration, patching workflow (admins + players)
- [Architecture](doc/ARCHITECTURE.md) — how the code works, design decisions
- [Build Guide](doc/BUILD.md) — prerequisites, build commands
- [Patch Format](doc/PATCH_FORMAT.md) — manifest and patch types
- [GRF Format](doc/GRF_FORMAT.md) — GRF 2.0 binary specification
- [Changelog](doc/CHANGELOG.md) — version history

## License

See [LICENSE](LICENSE).
=======
Ragnarok Online modern GRF patcher with GO and Wails3.
>>>>>>> e95e2724688887a40ed5ea475aa933cbc77dc0c9
