package grf

import (
	"archive/zip"
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

func TestGRFListAndRead(t *testing.T) {
	grfData := buildTestGRF(t)
	path := writeTempGRF(t, grfData)

	g, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer g.Close()

	files := g.ListFiles()
	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(files))
	}

	if !g.HasFile("data\\test.txt") {
		t.Error("expected to find data\\test.txt")
	}
	if !g.HasFile("data\\image.bmp") {
		t.Error("expected to find data\\image.bmp")
	}
	if g.HasFile("nonexistent.txt") {
		t.Error("should not find nonexistent file")
	}
}

func TestGRFReadFileContent(t *testing.T) {
	grfData := buildTestGRF(t)
	path := writeTempGRF(t, grfData)

	g, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer g.Close()

	content, err := g.ReadFile("data\\test.txt")
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "hello world" {
		t.Errorf("got %q, want %q", string(content), "hello world")
	}
}

func TestGRFReadFileNotFound(t *testing.T) {
	grfData := buildTestGRF(t)
	path := writeTempGRF(t, grfData)

	g, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer g.Close()

	_, err = g.ReadFile("nonexistent.txt")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestGRFOpenInvalidFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "invalid.grf")
	os.WriteFile(path, []byte("not a grf file"), 0644)

	_, err := Open(path)
	if err == nil {
		t.Error("expected error for invalid GRF")
	}
}

func TestPatchGRFFromZip(t *testing.T) {
	grfData := buildTestGRF(t)
	srcPath := writeTempGRF(t, grfData)

	zipPath := createTestZip(t, map[string]string{
		"data\\new_file.txt": "patched content",
		"data\\another.lub":  "lua bytecode here",
	})

	if err := PatchGRF(srcPath, zipPath); err != nil {
		t.Fatal(err)
	}

	g, err := Open(srcPath)
	if err != nil {
		t.Fatal(err)
	}
	defer g.Close()

	if !g.HasFile("data\\test.txt") {
		t.Error("original file missing after patch")
	}
	if !g.HasFile("data\\new_file.txt") {
		t.Error("patched file not found")
	}
	if !g.HasFile("data\\another.lub") {
		t.Error("second patched file not found")
	}
}

func TestPatchGRFFromGRF(t *testing.T) {
	grfData := buildTestGRF(t)
	srcPath := writeTempGRF(t, grfData)

	patchGrf := buildTestGRFWithFiles(t, map[string][]byte{
		"data\\new_file.txt": []byte("patched content"),
	})
	patchPath := writeTempGRF(t, patchGrf)

	if err := PatchGRF(srcPath, patchPath); err != nil {
		t.Fatal(err)
	}

	g, err := Open(srcPath)
	if err != nil {
		t.Fatal(err)
	}
	defer g.Close()

	if !g.HasFile("data\\new_file.txt") {
		t.Error("patched file not found")
	}
}

func TestPatchGRFFromGRFNoTarget(t *testing.T) {
	dir := t.TempDir()
	targetPath := filepath.Join(dir, "myserver.grf")

	patchGrf := buildTestGRFWithFiles(t, map[string][]byte{
		"data\\test.txt": []byte("hello world"),
	})
	patchPath := writeTempGRF(t, patchGrf)

	if err := PatchGRF(targetPath, patchPath); err != nil {
		t.Fatal(err)
	}

	g, err := Open(targetPath)
	if err != nil {
		t.Fatal(err)
	}
	defer g.Close()

	if !g.HasFile("data\\test.txt") {
		t.Error("file not found after patch")
	}
}

func TestNormalizeGRFPath(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"data/texture.bmp", "data\\texture.bmp"},
		{"data\\sound.wav", "data\\sound.wav"},
		{"data/sub/file.txt", "data\\sub\\file.txt"},
		{"file.txt", "file.txt"},
	}

	for _, tt := range tests {
		got := normalizeGRFPath(tt.input)
		if got != tt.want {
			t.Errorf("normalizeGRFPath(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func buildTestGRF(t *testing.T) []byte {
	t.Helper()
	return buildTestGRFWithFiles(t, map[string][]byte{
		"data\\test.txt":  []byte("hello world"),
		"data\\image.bmp": []byte("fake bitmap data here"),
	})
}

func buildTestGRFWithFiles(t *testing.T, files map[string][]byte) []byte {
	t.Helper()

	type fileBlock struct {
		name       string
		compressed []byte
	}

	var blocks []fileBlock
	for name, content := range files {
		compressed := compressZlib(t, content)
		blocks = append(blocks, fileBlock{name: name, compressed: compressed})
	}

	var fileTable bytes.Buffer
	dataOffset := uint32(0)
	for _, b := range blocks {
		fileTable.WriteString(b.name)
		fileTable.WriteByte(0)
		binary.Write(&fileTable, binary.LittleEndian, uint32(len(b.compressed)))
		binary.Write(&fileTable, binary.LittleEndian, uint32(len(b.compressed)))
		binary.Write(&fileTable, binary.LittleEndian, uint32(len(b.compressed)))
		fileTable.WriteByte(1)

		binary.Write(&fileTable, binary.LittleEndian, dataOffset)
		dataOffset += uint32(len(b.compressed))
	}

	compressedFT := compressZlib(t, fileTable.Bytes())

	ftSeekOffset := dataOffset

	var grf bytes.Buffer

	header := make([]byte, grfHeaderSize)
	copy(header, "Master of Magic")
	header[15] = 0
	binary.LittleEndian.PutUint32(header[0x2A:0x2E], 0x00020000)
	binary.LittleEndian.PutUint32(header[0x26:0x2A], uint32(len(files))+7)
	binary.LittleEndian.PutUint32(header[0x1E:0x22], ftSeekOffset)
	grf.Write(header)

	for _, b := range blocks {
		grf.Write(b.compressed)
	}

	binary.Write(&grf, binary.LittleEndian, uint32(len(compressedFT)))
	binary.Write(&grf, binary.LittleEndian, uint32(fileTable.Len()))
	grf.Write(compressedFT)

	return grf.Bytes()
}

func compressZlib(t *testing.T, data []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zlib.NewWriter(&buf)
	w.Write(data)
	w.Close()
	return buf.Bytes()
}

func writeTempGRF(t *testing.T, data []byte) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.grf")
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func createTestZip(t *testing.T, files map[string]string) string {
	t.Helper()

	dir := t.TempDir()
	zipPath := filepath.Join(dir, "patch.zip")

	var buf bytes.Buffer
	w := zip.NewWriter(&buf)

	for name, content := range files {
		f, err := w.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		f.Write([]byte(content))
	}
	w.Close()

	os.WriteFile(zipPath, buf.Bytes(), 0644)
	return zipPath
}
