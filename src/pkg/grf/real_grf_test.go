package grf

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRealGRFPatch(t *testing.T) {
	// Find the real patch_0.grf
	grfPath := "../../example/data/patch_0.grf"
	if _, err := os.Stat(grfPath); os.IsNotExist(err) {
		t.Skip("patch_0.grf not found")
	}

	// Open the real GRF
	g, err := Open(grfPath)
	if err != nil {
		t.Fatalf("failed to open real GRF: %v", err)
	}
	defer g.Close()

	t.Logf("GRF opened successfully")
	t.Logf("File count: %d", g.FileCount())
	t.Logf("Files: %v", g.ListFiles())

	// Read a file from the GRF
	files := g.ListFiles()
	if len(files) > 0 {
		content, err := g.ReadFile(files[0])
		if err != nil {
			t.Logf("Warning: could not read %s: %v", files[0], err)
		} else {
			t.Logf("Read %s: %d bytes", files[0], len(content))
		}
	}

	// Now test patching: copy the GRF, then merge a zip into it
	tmpDir := t.TempDir()
	targetPath := filepath.Join(tmpDir, "data.grf")

	// Copy the real GRF to temp
	grfData, err := os.ReadFile(grfPath)
	if err != nil {
		t.Fatal(err)
	}
	os.WriteFile(targetPath, grfData, 0644)

	// Create a patch zip
	zipPath := createTestZip(t, map[string]string{
		"data\\new_file.txt": "patched content",
	})

	// Apply the patch
	if err := PatchGRF(targetPath, zipPath); err != nil {
		t.Fatalf("failed to patch GRF: %v", err)
	}

	// Verify the patched GRF
	g2, err := Open(targetPath)
	if err != nil {
		t.Fatalf("failed to open patched GRF: %v", err)
	}
	defer g2.Close()

	t.Logf("Patched GRF: %d files", g2.FileCount())

	// Check that new file exists
	if !g2.HasFile("data\\new_file.txt") {
		t.Error("new file not found after patch")
	}

	// Check that original files still exist
	for _, name := range files {
		if !g2.HasFile(name) {
			t.Errorf("original file %s missing after patch", name)
		}
	}

	// Read the new file
	content, err := g2.ReadFile("data\\new_file.txt")
	if err != nil {
		t.Fatalf("failed to read new file: %v", err)
	}
	if string(content) != "patched content" {
		t.Errorf("new file content: got %q, want %q", string(content), "patched content")
	}
}

func TestRealGRFMerge(t *testing.T) {
	// Find the real patch_0.grf
	grfPath := "../../example/data/patch_0.grf"
	if _, err := os.Stat(grfPath); os.IsNotExist(err) {
		t.Skip("patch_0.grf not found")
	}

	// Create a patch GRF with some files
	tmpDir := t.TempDir()
	patchGrfPath := filepath.Join(tmpDir, "patch.grf")

	// Build a simple GRF with one file
	patchGrf := buildTestGRFWithFiles(t, map[string][]byte{
		"data\\patched.txt": []byte("merged from patch GRF"),
	})
	os.WriteFile(patchGrfPath, patchGrf, 0644)

	// Copy real GRF to target
	targetPath := filepath.Join(tmpDir, "data.grf")
	grfData, _ := os.ReadFile(grfPath)
	os.WriteFile(targetPath, grfData, 0644)

	// Merge patch GRF into target
	if err := PatchGRF(targetPath, patchGrfPath); err != nil {
		t.Fatalf("failed to merge GRFs: %v", err)
	}

	// Verify
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
