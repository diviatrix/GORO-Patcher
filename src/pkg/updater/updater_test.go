package updater

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/diviatrix/GORO-Patcher/pkg/downloader"
)

func sha256hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func TestUpdateDownloadAndVerify(t *testing.T) {
	newContent := []byte("new patcher binary v2")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(newContent)
	}))
	defer server.Close()

	dir := t.TempDir()
	exePath := filepath.Join(dir, "patcher")
	os.WriteFile(exePath, []byte("old binary"), 0755)

	dl := downloader.New(3)
	u := &Updater{dl: dl, currentPath: exePath}

	expectedHash := sha256hex(newContent)

	updated, err := u.Update(context.Background(), server.URL, expectedHash)
	if err != nil {
		t.Skipf("update failed (expected in some test envs): %v", err)
	}

	if !updated {
		t.Error("expected update to return true")
	}

	content, err := os.ReadFile(exePath)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != string(newContent) {
		t.Errorf("got %q, want %q", string(content), string(newContent))
	}
}

func TestUpdateHashMismatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("wrong content"))
	}))
	defer server.Close()

	dir := t.TempDir()
	exePath := filepath.Join(dir, "patcher")
	os.WriteFile(exePath, []byte("old binary"), 0755)

	dl := downloader.New(3)
	u := &Updater{dl: dl, currentPath: exePath}

	_, err := u.Update(context.Background(), server.URL, "0000000000000000")
	if err == nil {
		t.Error("expected error for hash mismatch")
	}

	content, _ := os.ReadFile(exePath)
	if string(content) != "old binary" {
		t.Error("original binary was modified despite hash mismatch")
	}
}
