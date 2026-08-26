package grf

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRealGRFPatch(t *testing.T) {

	grfPath := "../../../example/data/patch_0.grf"
	if _, err := os.Stat(grfPath); os.IsNotExist(err) {
		t.Skip("patch_0.grf not found")
	}

	g, err := Open(grfPath)
	if err != nil {
		t.Fatalf("failed to open real GRF: %v", err)
	}
	defer g.Close()

	t.Logf("GRF opened successfully")
	t.Logf("File count: %d", g.FileCount())
	t.Logf("Files: %v", g.ListFiles())

	files := g.ListFiles()
	for _, name := range files {
		if _, err := g.ReadFile(name); err != nil {
			t.Fatalf("could not read %s from real GRF: %v", name, err)
		}
	}

	tmpDir := t.TempDir()
	targetPath := filepath.Join(tmpDir, "myserver.grf")

	grfData, err := os.ReadFile(grfPath)
	if err != nil {
		t.Fatal(err)
	}
	os.WriteFile(targetPath, grfData, 0644)

	zipPath := createTestZip(t, map[string]string{
		"data\\new_file.txt": "patched content",
	})

	if err := PatchGRF(targetPath, zipPath); err != nil {
		t.Fatalf("failed to patch GRF: %v", err)
	}

	g2, err := Open(targetPath)
	if err != nil {
		t.Fatalf("failed to open patched GRF: %v", err)
	}
	defer g2.Close()

	t.Logf("Patched GRF: %d files", g2.FileCount())

	if !g2.HasFile("data\\new_file.txt") {
		t.Error("new file not found after patch")
	}

	for _, name := range files {
		if !g2.HasFile(name) {
			t.Errorf("original file %s missing after patch", name)
		}
	}

	content, err := g2.ReadFile("data\\new_file.txt")
	if err != nil {
		t.Fatalf("failed to read new file: %v", err)
	}
	if string(content) != "patched content" {
		t.Errorf("new file content: got %q, want %q", string(content), "patched content")
	}
}

func TestRealGRFMerge(t *testing.T) {

	grfPath := "../../../example/data/patch_0.grf"
	if _, err := os.Stat(grfPath); os.IsNotExist(err) {
		t.Skip("patch_0.grf not found")
	}

	tmpDir := t.TempDir()
	patchGrfPath := filepath.Join(tmpDir, "patch.grf")

	patchGrf := buildTestGRFWithFiles(t, map[string][]byte{
		"data\\patched.txt": []byte("merged from patch GRF"),
	})
	os.WriteFile(patchGrfPath, patchGrf, 0644)

	targetPath := filepath.Join(tmpDir, "myserver.grf")
	grfData, _ := os.ReadFile(grfPath)
	os.WriteFile(targetPath, grfData, 0644)

	if err := PatchGRF(targetPath, patchGrfPath); err != nil {
		t.Fatalf("failed to merge GRFs: %v", err)
	}

	g, err := Open(targetPath)
	if err != nil {
		t.Fatalf("failed to open merged GRF: %v", err)
	}
	defer g.Close()

	if !g.HasFile("data\\patched.txt") {
		t.Error("merged file not found")
	}

	content, err := g.ReadFile("data\\patched.txt")
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "merged from patch GRF" {
		t.Errorf("got %q, want %q", string(content), "merged from patch GRF")
	}
}
