package engine

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestPatchRaw(t *testing.T) {
	gameDir := t.TempDir()
	zipPath := createRawTestZip(t, map[string]string{
		"System/itemInfo.lub": "new lua bytecode",
		"data/texture.bmp":    "new bitmap",
	})

	if err := PatchRaw(gameDir, zipPath); err != nil {
		t.Fatal(err)
	}

	content, err := os.ReadFile(filepath.Join(gameDir, "System/itemInfo.lub"))
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "new lua bytecode" {
		t.Errorf("got %q, want %q", string(content), "new lua bytecode")
	}

	content, err = os.ReadFile(filepath.Join(gameDir, "data/texture.bmp"))
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "new bitmap" {
		t.Errorf("got %q, want %q", string(content), "new bitmap")
	}
}

func TestPatchRawCreatesDirs(t *testing.T) {
	gameDir := t.TempDir()
	zipPath := createRawTestZip(t, map[string]string{
		"deep/nested/dir/file.txt": "deep content",
	})

	if err := PatchRaw(gameDir, zipPath); err != nil {
		t.Fatal(err)
	}

	content, err := os.ReadFile(filepath.Join(gameDir, "deep/nested/dir/file.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "deep content" {
		t.Errorf("got %q, want %q", string(content), "deep content")
	}
}

func TestPatchRawBackup(t *testing.T) {
	gameDir := t.TempDir()

	existing := filepath.Join(gameDir, "existing.txt")
	os.MkdirAll(filepath.Dir(existing), 0755)
	os.WriteFile(existing, []byte("old content"), 0644)

	zipPath := createRawTestZip(t, map[string]string{
		"existing.txt": "new content",
	})

	if err := PatchRaw(gameDir, zipPath); err != nil {
		t.Fatal(err)
	}

	content, _ := os.ReadFile(existing)
	if string(content) != "new content" {
		t.Errorf("got %q, want %q", string(content), "new content")
	}

	bak, err := os.ReadFile(existing + ".bak")
	if err != nil {
		t.Fatal(err)
	}
	if string(bak) != "old content" {
		t.Errorf("backup: got %q, want %q", string(bak), "old content")
	}
}

func TestPatchRawPathTraversal(t *testing.T) {
	gameDir := t.TempDir()

	zipPath := createRawTestZip(t, map[string]string{
		"../../../etc/passwd": "malicious",
	})

	if err := PatchRaw(gameDir, zipPath); err != nil {
		t.Fatal(err)
	}

	// Check that /etc/passwd was not overwritten with "malicious"
	content, err := os.ReadFile("/etc/passwd")
	if err == nil && string(content) == "malicious" {
		t.Error("path traversal succeeded — /etc/passwd was overwritten")
	}

	// Verify file was extracted to safe location
	safePath := filepath.Join(gameDir, "etc/passwd")
	if _, err := os.Stat(safePath); os.IsNotExist(err) {
		t.Error("file should have been extracted to safe location")
	}
}

func TestSanitizePath(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"System/itemInfo.lub", "System/itemInfo.lub"},
		{"../etc/passwd", "etc/passwd"},
		{"../../secret.txt", "secret.txt"},
		{"/absolute/path.txt", "absolute/path.txt"},
		{"data\\windows\\file.txt", "data/windows/file.txt"},
		{"", ""},
	}

	for _, tt := range tests {
		got := sanitizePath(tt.input)
		if got != tt.want {
			t.Errorf("sanitizePath(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestCleanupStaleFiles(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "data.grf.patching"), []byte("stale"), 0644)
	os.WriteFile(filepath.Join(dir, "patcher.exe.new"), []byte("stale"), 0644)
	os.WriteFile(filepath.Join(dir, "valid.txt"), []byte("keep"), 0644)

	CleanupStaleFiles(dir)

	if _, err := os.Stat(filepath.Join(dir, "data.grf.patching")); !os.IsNotExist(err) {
		t.Error("expected .patching to be deleted")
	}
	if _, err := os.Stat(filepath.Join(dir, "patcher.exe.new")); !os.IsNotExist(err) {
		t.Error("expected .new to be deleted")
	}
	if _, err := os.Stat(filepath.Join(dir, "valid.txt")); err != nil {
		t.Error("valid file should not be deleted")
	}
}

func TestCleanupStaleFilesRestoresBak(t *testing.T) {
	dir := t.TempDir()
	// Target is missing, .bak has good data
	os.WriteFile(filepath.Join(dir, "data.grf.bak"), []byte("good data here"), 0644)

	CleanupStaleFiles(dir)

	data, err := os.ReadFile(filepath.Join(dir, "data.grf"))
	if err != nil {
		t.Fatalf("expected data.grf to be restored: %v", err)
	}
	if string(data) != "good data here" {
		t.Error("restored data does not match .bak")
	}
	if _, err := os.Stat(filepath.Join(dir, "data.grf.bak")); !os.IsNotExist(err) {
		t.Error("expected .bak to be deleted after restore")
	}
}

func TestCleanupStaleFilesDeletesBakWhenTargetOK(t *testing.T) {
	dir := t.TempDir()
	// Target exists and looks like a valid GRF (has magic bytes)
	grfData := make([]byte, 2048)
	copy(grfData, "Master of Magic")
	os.WriteFile(filepath.Join(dir, "data.grf"), grfData, 0644)
	os.WriteFile(filepath.Join(dir, "data.grf.bak"), []byte("old backup"), 0644)

	CleanupStaleFiles(dir)

	if _, err := os.Stat(filepath.Join(dir, "data.grf.bak")); !os.IsNotExist(err) {
		t.Error("expected .bak to be deleted when target is valid")
	}
	// Target should be untouched
	info, _ := os.Stat(filepath.Join(dir, "data.grf"))
	if info.Size() != 2048 {
		t.Error("target should be untouched")
	}
}

func TestCleanupStaleFilesRestoresCorruptGRF(t *testing.T) {
	dir := t.TempDir()
	// Target exists but is corrupt (no magic bytes)
	os.WriteFile(filepath.Join(dir, "data.grf"), []byte("corrupt"), 0644)
	os.WriteFile(filepath.Join(dir, "data.grf.bak"), make([]byte, 2048), 0644)

	CleanupStaleFiles(dir)

	info, err := os.Stat(filepath.Join(dir, "data.grf"))
	if err != nil {
		t.Fatalf("expected data.grf to be restored: %v", err)
	}
	if info.Size() != 2048 {
		t.Error("expected restored file to match .bak size")
	}
}

// --- Test helpers ---

func createRawTestZip(t *testing.T, files map[string]string) string {
	t.Helper()

	dir := t.TempDir()
	zipPath := filepath.Join(dir, "raw.zip")

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
