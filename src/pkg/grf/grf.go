package grf

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"golang.org/x/text/encoding/korean"
	"golang.org/x/text/transform"
)

const grfHeaderSize = 0x2E

type GRF struct {
	header      grfHeader
	entries     map[string]*FileEntry
	pendingData map[string][]byte
	filePath    string
	file        *os.File
	dataOffset  int64
}

type grfHeader struct {
	Magic     [16]byte
	Key       [8]byte
	Reserved  [20]byte
	Version   uint32
	FileCount uint32
}

func Open(path string) (*GRF, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	var headerBuf [grfHeaderSize]byte
	if _, err := io.ReadFull(f, headerBuf[:]); err != nil {
		f.Close()
		return nil, fmt.Errorf("read header: %w", err)
	}

	if string(headerBuf[0:15]) != "Master of Magic" {
		f.Close()
		return nil, fmt.Errorf("invalid GRF magic")
	}

	version := binary.LittleEndian.Uint32(headerBuf[0x2A:0x2E]) >> 8
	fileCount := int(binary.LittleEndian.Uint32(headerBuf[0x26:0x2A])) - 7

	ftSeekOffset := binary.LittleEndian.Uint32(headerBuf[0x1E:0x22])
	if _, err := f.Seek(int64(ftSeekOffset), io.SeekCurrent); err != nil {
		f.Close()
		return nil, fmt.Errorf("seek to file table: %w", err)
	}

	var ftHeader [8]byte
	if _, err := io.ReadFull(f, ftHeader[:]); err != nil {
		f.Close()
		return nil, fmt.Errorf("read file table header: %w", err)
	}

	compressedSize := binary.LittleEndian.Uint32(ftHeader[0:4])
	uncompressedSize := binary.LittleEndian.Uint32(ftHeader[4:8])

	compressed := make([]byte, compressedSize)
	if _, err := io.ReadFull(f, compressed); err != nil {
		f.Close()
		return nil, fmt.Errorf("read compressed file table: %w", err)
	}

	zr, err := zlib.NewReader(bytes.NewReader(compressed))
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("zlib reader: %w", err)
	}
	defer zr.Close()

	decompressed, err := io.ReadAll(zr)
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("decompress file table: %w", err)
	}

	if len(decompressed) != int(uncompressedSize) {
		f.Close()
		return nil, fmt.Errorf("file table size mismatch: got %d, want %d", len(decompressed), uncompressedSize)
	}

	entries, err := parseFileEntries(decompressed, fileCount)
	if err != nil {
		f.Close()
		return nil, err
	}

	entryMap := make(map[string]*FileEntry, len(entries))
	for i := range entries {
		entryMap[entries[i].FileName] = &entries[i]
	}

	dataOffset := int64(grfHeaderSize)

	return &GRF{
		header: grfHeader{
			Version:   version,
			FileCount: uint32(fileCount),
		},
		entries:    entryMap,
		filePath:   path,
		file:       f,
		dataOffset: dataOffset,
	}, nil
}

func parseFileEntries(data []byte, expectedCount int) ([]FileEntry, error) {
	var entries []FileEntry
	r := bytes.NewReader(data)

	for r.Len() > 0 && len(entries) < expectedCount {

		var nameBytes []byte
		for {
			b, err := r.ReadByte()
			if err != nil {
				return entries, err
			}
			if b == 0 {
				break
			}
			nameBytes = append(nameBytes, b)
		}

		rawName := string(nameBytes)

		displayName := decodeEUCKR(nameBytes)

		var entryMeta [17]byte
		if _, err := io.ReadFull(r, entryMeta[:]); err != nil {
			return entries, fmt.Errorf("read entry meta: %w", err)
		}

		entry := FileEntry{
			CompressedSize:        binary.LittleEndian.Uint32(entryMeta[0:4]),
			CompressedSizeAligned: binary.LittleEndian.Uint32(entryMeta[4:8]),
			UncompressedSize:      binary.LittleEndian.Uint32(entryMeta[8:12]),
			Flags:                 entryMeta[12],
			Offset:                binary.LittleEndian.Uint32(entryMeta[13:17]),
			FileName:              rawName,
			DisplayName:           displayName,
		}

		entries = append(entries, entry)
	}

	return entries, nil
}

func decodeEUCKR(data []byte) string {
	decoder := korean.EUCKR.NewDecoder()
	utf8Bytes, _, err := transform.Bytes(decoder, data)
	if err != nil {

		return string(data)
	}

	return strings.ReplaceAll(string(utf8Bytes), "\\", "/")
}

func (g *GRF) ListFiles() []string {
	names := make([]string, 0, len(g.entries))
	for name := range g.entries {
		names = append(names, name)
	}
	return names
}

func (g *GRF) DisplayName(rawName string) string {
	if e, ok := g.entries[rawName]; ok {
		return e.DisplayName
	}
	return rawName
}

func (g *GRF) HasFile(name string) bool {
	_, ok := g.entries[name]
	return ok
}

func (g *GRF) FileCount() int {
	return len(g.entries)
}

func (g *GRF) ReadFile(name string) ([]byte, error) {
	entry, ok := g.entries[name]
	if !ok {
		return nil, fmt.Errorf("file not found: %s", name)
	}

	dataPos := g.dataOffset + int64(entry.Offset)

	if _, err := g.file.Seek(dataPos, io.SeekStart); err != nil {
		return nil, fmt.Errorf("seek to data: %w", err)
	}

	compressed := make([]byte, entry.CompressedSize)
	if _, err := io.ReadFull(g.file, compressed); err != nil {
		return nil, fmt.Errorf("read compressed data: %w", err)
	}

	if entry.UncompressedSize == 0 {
		return compressed, nil
	}

	zr, err := zlib.NewReader(bytes.NewReader(compressed))
	if err != nil {
		return nil, fmt.Errorf("zlib reader: %w", err)
	}
	defer zr.Close()

	decompressed, err := io.ReadAll(zr)
	if err != nil {
		return nil, fmt.Errorf("decompress: %w", err)
	}

	return decompressed, nil
}

func (g *GRF) AddFile(name string, data []byte) error {
	compressed := compressZlibData(data)

	entry := &FileEntry{
		CompressedSize:        uint32(len(compressed)),
		CompressedSizeAligned: uint32(len(compressed)),
		UncompressedSize:      uint32(len(data)),
		Flags:                 1,
		Offset:                0,
		FileName:              name,
	}

	g.entries[name] = entry
	if g.pendingData == nil {
		g.pendingData = make(map[string][]byte)
	}
	g.pendingData[name] = compressed

	return nil
}

func (g *GRF) Save(dest string) error {
	tmpPath := dest + ".patching"
	bakPath := dest + ".bak"

	if err := g.SaveAs(tmpPath); err != nil {
		os.Remove(tmpPath)
		return err
	}

	g.file.Close()
	g.file = nil

	if err := os.Rename(dest, bakPath); err != nil && !os.IsNotExist(err) {
		g.file, _ = os.Open(dest)
		return fmt.Errorf("rename to bak: %w", err)
	}

	if err := os.Rename(tmpPath, dest); err != nil {
		os.Rename(bakPath, dest)
		g.file, _ = os.Open(dest)
		return fmt.Errorf("rename temp to dest: %w", err)
	}

	os.Remove(bakPath)

	g.filePath = dest
	var err error
	g.file, err = os.Open(dest)
	if err != nil {
		return fmt.Errorf("reopen saved file: %w", err)
	}

	return nil
}

func (g *GRF) SaveAs(dest string) error {
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()

	names := make([]string, 0, len(g.entries))
	for name := range g.entries {
		names = append(names, name)
	}
	sort.Strings(names)

	type entryDef struct {
		srcOffset uint32
		meta      FileEntry
	}
	defs := make([]entryDef, 0, len(names))
	ftSeekOffset := uint32(0)
	for _, name := range names {
		entry := g.entries[name]
		meta := *entry
		meta.Offset = ftSeekOffset

		dataLen := int(entry.CompressedSize)
		if pending, ok := g.pendingData[name]; ok {
			dataLen = len(pending)
		}

		meta.CompressedSize = uint32(dataLen)
		meta.CompressedSizeAligned = uint32(dataLen)
		ftSeekOffset += uint32(dataLen)

		defs = append(defs, entryDef{srcOffset: entry.Offset, meta: meta})
	}

	header := make([]byte, grfHeaderSize)
	copy(header, "Master of Magic")
	header[15] = 0
	header[0x2A] = 0x00
	header[0x2B] = 0x02
	binary.LittleEndian.PutUint32(header[0x26:0x2A], uint32(len(names))+7)
	binary.LittleEndian.PutUint32(header[0x1E:0x22], ftSeekOffset)
	if _, err := f.Write(header); err != nil {
		return err
	}

	for i, name := range names {
		if pending, ok := g.pendingData[name]; ok {
			if _, err := f.Write(pending); err != nil {
				return err
			}
			continue
		}
		def := defs[i]
		dataOffset := int64(def.srcOffset) + int64(grfHeaderSize)
		if _, err := g.file.Seek(dataOffset, io.SeekStart); err != nil {
			return fmt.Errorf("seek %s: %w", name, err)
		}
		if _, err := io.CopyN(f, g.file, int64(def.meta.CompressedSize)); err != nil {
			return fmt.Errorf("copy %s: %w", name, err)
		}
	}

	allEntries := make([]FileEntry, 0, len(defs))
	for _, def := range defs {
		allEntries = append(allEntries, def.meta)
	}
	fileTableData := buildFileTable(allEntries)
	compressedFT := compressZlibData(fileTableData)

	binary.Write(f, binary.LittleEndian, uint32(len(compressedFT)))
	binary.Write(f, binary.LittleEndian, uint32(len(fileTableData)))
	f.Write(compressedFT)

	return nil
}

func buildFileTable(entries []FileEntry) []byte {
	var buf bytes.Buffer
	for _, entry := range entries {
		buf.WriteString(entry.FileName)
		buf.WriteByte(0)
		binary.Write(&buf, binary.LittleEndian, entry.CompressedSize)
		binary.Write(&buf, binary.LittleEndian, entry.CompressedSizeAligned)
		binary.Write(&buf, binary.LittleEndian, entry.UncompressedSize)
		buf.WriteByte(entry.Flags)
		binary.Write(&buf, binary.LittleEndian, entry.Offset)
	}
	return buf.Bytes()
}

func compressZlibData(data []byte) []byte {
	var buf bytes.Buffer
	w := zlib.NewWriter(&buf)
	w.Write(data)
	w.Close()
	return buf.Bytes()
}

func (g *GRF) Close() error {
	if g.file != nil {
		return g.file.Close()
	}
	return nil
}

func Create(path string) (*GRF, error) {
	f, err := os.Create(path)
	if err != nil {
		return nil, fmt.Errorf("create file: %w", err)
	}

	header := make([]byte, grfHeaderSize)
	copy(header, "Master of Magic")
	header[15] = 0
	header[0x2A] = 0x00
	header[0x2B] = 0x02
	binary.LittleEndian.PutUint32(header[0x26:0x2A], 7)

	binary.LittleEndian.PutUint32(header[0x1E:0x22], 0)

	if _, err := f.Write(header); err != nil {
		f.Close()
		return nil, fmt.Errorf("write header: %w", err)
	}

	emptyData := []byte{}
	compressedFT := compressZlibData(emptyData)

	var ftBuf bytes.Buffer
	binary.Write(&ftBuf, binary.LittleEndian, uint32(len(compressedFT)))
	binary.Write(&ftBuf, binary.LittleEndian, uint32(len(emptyData)))
	ftBuf.Write(compressedFT)

	if _, err := f.Write(ftBuf.Bytes()); err != nil {
		f.Close()
		return nil, fmt.Errorf("write file table: %w", err)
	}

	if err := f.Close(); err != nil {
		return nil, fmt.Errorf("close file: %w", err)
	}

	return Open(path)
}
