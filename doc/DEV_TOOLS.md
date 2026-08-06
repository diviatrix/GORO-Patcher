# EUC-KR Path Conversion

> Source: [Ragnarok Research Lab](https://ragnarokresearchlab.github.io/tools/)

GRF file names are stored as EUC-KR encoded Windows paths. To display them properly, decode EUC-KR to UTF-8.

## Example (EUC-KR to UTF-8)

Raw GRF path: `data\model\¿ÜºÎ¼ÒÇ°\Æ®·¦01.rsm`
Decoded: `data/model/외부소품/트랩01.rsm`

Path separators (`\` → `/`) should also be normalized after decoding.

## Implementation in GORO-Patcher

Uses `golang.org/x/text/encoding/korean` for decoding:

```go
decoder := korean.EUCKR.NewDecoder()
utf8Bytes, _, err := transform.Bytes(decoder, rawEUCKRBytes)
```

Raw filenames are kept as map keys (preserves uniqueness), decoded names used for display only.
