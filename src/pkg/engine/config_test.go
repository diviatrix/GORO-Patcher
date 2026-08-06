package engine

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigNotExist(t *testing.T) {
	dir := t.TempDir()
	cfg, err := LoadConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ExeName != "ragexe.exe" {
		t.Errorf("expected ragexe.exe, got %s", cfg.ExeName)
	}
}

func TestSaveLoadConfig(t *testing.T) {
	dir := t.TempDir()
	cfg := &Config{
		ManifestURL: "https://example.com/plist.json",
		ExeName:     "myclient.exe",
	}

	if err := SaveConfig(dir, cfg); err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadConfig(dir)
	if err != nil {
		t.Fatal(err)
	}

	if loaded.ManifestURL != cfg.ManifestURL {
		t.Errorf("manifest_url: got %s, want %s", loaded.ManifestURL, cfg.ManifestURL)
	}
	if loaded.ExeName != cfg.ExeName {
		t.Errorf("exe_name: got %s, want %s", loaded.ExeName, cfg.ExeName)
	}
}

func TestLoadConfigMalformed(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "goro-config.json"), []byte("not json"), 0644)
	_, err := LoadConfig(dir)
	if err == nil {
		t.Error("expected error for malformed JSON")
	}
}
