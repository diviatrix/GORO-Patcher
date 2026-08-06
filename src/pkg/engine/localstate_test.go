package engine

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadLocalVersionNotExist(t *testing.T) {
	dir := t.TempDir()
	v, err := ReadLocalVersion(dir)
	if err != nil {
		t.Fatal(err)
	}
	if v != 0 {
		t.Errorf("expected 0, got %d", v)
	}
}

func TestWriteReadLocalVersion(t *testing.T) {
	dir := t.TempDir()
	if err := WriteLocalVersion(dir, 105); err != nil {
		t.Fatal(err)
	}

	v, err := ReadLocalVersion(dir)
	if err != nil {
		t.Fatal(err)
	}
	if v != 105 {
		t.Errorf("expected 105, got %d", v)
	}
}

func TestReadLocalVersionEmpty(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "goro-patch.json"), []byte("   \n"), 0644)

	v, err := ReadLocalVersion(dir)
	if err != nil {
		t.Fatal(err)
	}
	if v != 0 {
		t.Errorf("expected 0 for empty file, got %d", v)
	}
}

func TestReadLocalVersionMalformed(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "goro-patch.json"), []byte("abc"), 0644)

	_, err := ReadLocalVersion(dir)
	if err == nil {
		t.Error("expected error for non-numeric content")
	}
}
