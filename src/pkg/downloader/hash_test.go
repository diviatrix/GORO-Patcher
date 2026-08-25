package downloader

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHashBytes(t *testing.T) {
	hash := HashBytes([]byte("hello world"))
	if hash == "" {
		t.Fatal("expected non-empty hash")
	}
	if len(hash) != 64 {
		t.Fatalf("expected 64-char SHA-256 hex, got %d chars: %q", len(hash), hash)
	}

	if hash != "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9" {
		t.Errorf("unexpected hash %q (algorithm drift?)", hash)
	}

	hash2 := HashBytes([]byte("hello world"))
	if hash != hash2 {
		t.Errorf("same input produced different hashes: %s != %s", hash, hash2)
	}

	hash3 := HashBytes([]byte("hello worle"))
	if hash == hash3 {
		t.Error("different inputs produced same hash")
	}
}

func TestHashFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	content := []byte("test content for hashing")
	os.WriteFile(path, content, 0644)

	hash, err := HashFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if hash == "" {
		t.Fatal("expected non-empty hash")
	}

	bytesHash := HashBytes(content)
	if hash != bytesHash {
		t.Errorf("HashFile (%s) != HashBytes (%s)", hash, bytesHash)
	}
}

func TestHashFileNotExist(t *testing.T) {
	_, err := HashFile("/nonexistent/file.bin")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestVerifyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.bin")
	content := []byte("verify me")
	os.WriteFile(path, content, 0644)

	expected := HashBytes(content)
	if err := VerifyFile(path, expected); err != nil {
		t.Errorf("expected match, got error: %v", err)
	}

	if err := VerifyFile(path, "0000000000000000"); err == nil {
		t.Error("expected mismatch error")
	}
}

func TestVerifyBytes(t *testing.T) {
	data := []byte("test data")
	expected := HashBytes(data)

	if err := VerifyBytes(data, expected); err != nil {
		t.Errorf("expected match, got error: %v", err)
	}

	if err := VerifyBytes(data, "deadbeefdeadbeef"); err == nil {
		t.Error("expected mismatch error")
	}
}

func TestHashDeterministic(t *testing.T) {
	data := []byte("deterministic test input 12345")
	h1 := HashBytes(data)
	h2 := HashBytes(data)
	if h1 != h2 {
		t.Errorf("non-deterministic: %s != %s", h1, h2)
	}
}
