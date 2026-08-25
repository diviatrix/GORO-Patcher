# Plan: Password-Protected ZIP Support

## Goal

Add PKWARE traditional ZIP encryption (ZipCrypto) support to both GRF merge and raw extract paths. Password from `goro-config.json`.

## How PKWARE ZIP Encryption Works

1. 3 key registers initialized from password: `key0=0x12345678`, `key1=0x23456789`, `key2=0x34567890`
2. Each password byte feeds through `updateKeys()` (CRC32-based)
3. 12-byte encryption header before compressed data (byte 11 = high byte of CRC32 for verification)
4. Stream cipher: each byte XORed with keystream from `decryptByte()`, then `updateKeys(plaintext_byte)`

## Changes

### New file: `src/pkg/grf/zipcrypto.go` (~60 lines)

```go
package grf

// ZipCrypto - PKWARE traditional ZIP encryption
type ZipCrypto struct {
    key0, key1, key2 uint32
}

func NewZipCrypto(password []byte) *ZipCrypto  // init keys from password
func (z *ZipCrypto) updateKeys(c byte)          // CRC32-based key update
func (z *ZipCrypto) decryptByte() byte          // keystream generator
func (z *ZipCrypto) DecryptHeader(h [12]byte) [12]byte  // decrypt 12-byte header
func (z *ZipCrypto) VerifyHeader(decrypted [12]byte, fileCRC32 uint32) bool
func (z *ZipCrypto) DecryptStream(r io.Reader) io.Reader // wrap reader with decryption
```

### New file: `src/pkg/grf/zipcrypto_test.go`

- Encrypt-then-decrypt roundtrip
- Wrong password detection (CRC check)
- Stream decryption with known test vectors

### New helper: `src/pkg/grf/zipopener.go`

```go
// OpenZipEntry reads a ZIP entry, decrypting if necessary.
// Returns decompressed data.
func OpenZipEntry(file *zip.File, password string) ([]byte, error) {
    if file.Flags&0x0001 == 0 {
        // Not encrypted - use standard path
        rc, err := file.Open()
        // ... read all ...
    }

    // Encrypted - use ZipCrypto
    raw, _ := file.OpenRaw()
    zc := NewZipCrypto([]byte(password))

    var encHeader [12]byte
    io.ReadFull(raw, encHeader[:])
    decHeader := zc.DecryptHeader(encHeader)

    if !zc.VerifyHeader(decHeader, file.CRC32) {
        return nil, fmt.Errorf("wrong password")
    }

    decrypted := zc.DecryptStream(raw)
    // decompress based on file.Method (Store or Deflate)
    // verify CRC32
}
```

### Modified: `src/pkg/grf/patcher.go`

- `patchGRFFromZip(grfPath, zipPath, password, progress)` - add password parameter
- For each entry: use `OpenZipEntry(file, password)` instead of `file.Open()`
- Signature changes:

```go
func PatchGRF(grfPath, patchPath, password string) error
func PatchGRFWithProgress(grfPath, patchPath, password string, progress ProgressFunc) error
func patchGRFFromZip(grfPath, zipPath, password string, progress ProgressFunc) error
```

### Modified: `src/pkg/engine/rawpatcher.go`

- `PatchRaw(gamePath, zipPath, password)` - add password parameter
- Use `OpenZipEntry(file, password)` instead of `file.Open()`

### Modified: `src/pkg/engine/config.go`

```go
type Config struct {
    ManifestURL string `json:"manifest_url"`
    ExeName     string `json:"exe_name"`
    NotesURL    string `json:"notes_url,omitempty"`
    Password    string `json:"password,omitempty"`  // NEW
}
```

### Modified: `src/app.go`

- `applyPatch()` passes `a.config.Password` to `grf.PatchGRFWithProgress()` and `engine.PatchRaw()`

### Modified: `build/goro-config.json`

```json
{
  "manifest_url": "...",
  "exe_name": "ragnarok.exe",
  "notes_url": "...",
  "password": "optional-zip-password"
}
```

## Execution Order

1. `zipcrypto.go` + `zipcrypto_test.go` - implement and test PKWARE algorithm
2. `zipopener.go` - add `OpenZipEntry()` helper
3. `patcher.go` - update signatures, use `OpenZipEntry()`
4. `rawpatcher.go` - update signature, use `OpenZipEntry()`
5. `config.go` - add `Password` field
6. `app.go` - wire password through to patch functions
7. Run tests: `cd src && go test ./...`
8. Run lint: `cd src && go vet ./...`

## Key Technical Details

- `zip.File.OpenRaw()` returns raw reader starting at the 12-byte encryption header
- `file.Flags` is `uint16` - bit 0 = encrypted
- `file.CRC32` is available for header verification
- `file.Method` tells us Store (0) or Deflate (8) for decompression
- CRC32 uses standard PKZIP polynomial `0xEDB88320` - Go's `hash/crc32.IEEE` matches
- Password is optional - empty string = no encryption (backward compatible)
- Strong encryption (AES, bit 6) is NOT supported - only traditional PKWARE

## References

- PKWARE APPNOTE.TXT Section 6.0/6.1: Traditional encryption specification
- Medium article by physxhacker: GRF encryption reverse engineering
- Tokeiburu/GRFEditor: GRF format and encryption orchestration code
- rpatchur: GRF DES encryption reference implementation
