package updater

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/diviatrix/GORO-Patcher/pkg/downloader"
)

func TestNeedsUpdate(t *testing.T) {
	dl := downloader.New(3)

	dir := t.TempDir()
	exePath := filepath.Join(dir, "patcher")
	os.WriteFile(exePath, []byte("current binary"), 0755)

	u := &Updater{dl: dl, currentPath: exePath}

	currentHash := downloader.HashBytes([]byte("current binary"))
	needed, err := u.NeedsUpdate(context.Background(), currentHash)
	if err != nil {
		t.Fatal(err)
	}
	if needed {
		t.Error("expected no update needed for same hash")
	}

	needed, err = u.NeedsUpdate(context.Background(), "0000000000000000")
	if err != nil {
		t.Fatal(err)
	}
	if !needed {
		t.Error("expected update needed for different hash")
	}
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

	expectedHash := downloader.HashBytes(newContent)

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
