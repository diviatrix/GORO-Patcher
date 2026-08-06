# GRF File Format (Version 2.0)

> Source: [Ragnarok Research Lab](https://ragnarokresearchlab.github.io/file-formats/grf/)

GRF files are virtual file system containers (like ZIP archives) used by Ragnarok Online.

## Header (46 bytes)

| Field | Offset | Length | Type | Description |
|-------|--------|--------|------|-------------|
| `Signature` | 0 | 15 | `string` | Always `"Master of Magic"` (no null-terminator) |
| `EncryptionKey` | 15 | 15 | `byte[15]` | All zeroes if encryption isn't used |
| `FileTableOffset` | 30 | 4 | `uint32` | Offset to file table, relative to end of header |
| `ScramblingSeed` | 34 | 4 | `uint32` | Used for obfuscation |
| `ScrambledFileCount` | 38 | 4 | `uint32` | Real count = `ScrambledFileCount - ScramblingSeed - 7` |
| `Version` | 42 | 4 | `uint32` | `0x0200` for version 2.0 (stored as minor=0x00 at 0x2A, major=0x02 at 0x2B) |

## File Table

Stored at `header_end + FileTableOffset`. Compressed with zlib.

| Field | Offset | Length | Type | Description |
|-------|--------|--------|------|-------------|
| `CompressedSize` | 0 | 4 | `uint32` | Size of compressed file table |
| `DecompressedSize` | 4 | 4 | `uint32` | Size after decompression |
| `CompressedData` | 8 | variable | `blob` | zlib-compressed list of file entries |

## File Entries

After decompressing the file table, each entry:

| Field | Offset | Length | Type | Description |
|-------|--------|--------|------|-------------|
| `CompressedSize` | 0 | 4 | `uint32` | Compressed file size |
| `CompressedSizeAligned` | 4 | 4 | `uint32` | Bytes to read (padded to 8-byte boundary) |
| `DecompressedSize` | 8 | 4 | `uint32` | Uncompressed file size |
| `Flags` | 12 | 1 | `byte` | 0x01 = zlib compressed |
| `Offset` | 13 | 4 | `uint32` | Offset to file data (relative to data section start) |
| `FileName` | 17 | variable | `string` | Null-terminated EUC-KR encoded path |

## Data Layout

```
[Header 46 bytes] [Data blocks] [File table]
```

- Data section starts immediately after header (offset 46)
- Entry offsets are relative to data section start
- File names use EUC-KR encoding (Korean) with backslash path separators
