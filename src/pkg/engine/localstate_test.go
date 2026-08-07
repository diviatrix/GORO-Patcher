package engine

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadLocalStateNotExist(t *testing.T) {
	dir := t.TempDir()
	state, err := ReadLocalState(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(state.AppliedPatches) != 0 {
		t.Errorf("expected 0 patches, got %d", len(state.AppliedPatches))
	}
}

func TestWriteReadLocalState(t *testing.T) {
	dir := t.TempDir()
	state := &LocalState{
		AppliedPatches: []LocalPatch{
			{ID: 0, Name: "patch_0.grf", Hash: "a2ac70ab8dff9e89", Size: 2043},
			{ID: 1, Name: "patch_1.zip", Hash: "45694d77d7281b59", Size: 539},
		},
	}
	if err := WriteLocalState(dir, state); err != nil {
		t.Fatal(err)
	}

	loaded, err := ReadLocalState(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.AppliedPatches) != 2 {
		t.Fatalf("expected 2 patches, got %d", len(loaded.AppliedPatches))
	}
	if loaded.AppliedPatches[0].ID != 0 || loaded.AppliedPatches[0].Hash != "a2ac70ab8dff9e89" {
		t.Errorf("patch 0 mismatch: %+v", loaded.AppliedPatches[0])
	}
	if loaded.AppliedPatches[1].ID != 1 || loaded.AppliedPatches[1].Size != 539 {
		t.Errorf("patch 1 mismatch: %+v", loaded.AppliedPatches[1])
	}
}

func TestReadLocalStateMalformed(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "goro-patch.json"), []byte("not json"), 0644)

	state, err := ReadLocalState(dir)
	if err != nil {
		t.Fatal("expected no error, got:", err)
	}
	if len(state.AppliedPatches) != 0 {
		t.Errorf("expected 0 patches, got %d", len(state.AppliedPatches))
	}
	if _, err := os.Stat(filepath.Join(dir, "goro-patch.json")); !os.IsNotExist(err) {
		t.Error("expected file to be deleted")
	}
}

func TestReadLocalStateOldFormat(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "goro-patch.json"), []byte("105"), 0644)

	state, err := ReadLocalState(dir)
	if err != nil {
		t.Fatal("expected no error, got:", err)
	}
	if len(state.AppliedPatches) != 0 {
		t.Errorf("expected 0 patches, got %d", len(state.AppliedPatches))
	}
	if _, err := os.Stat(filepath.Join(dir, "goro-patch.json")); !os.IsNotExist(err) {
		t.Error("expected file to be deleted")
	}
}

func TestLocalVersion(t *testing.T) {
	state := &LocalState{
		AppliedPatches: []LocalPatch{
			{ID: 0, Name: "patch_0.grf"},
			{ID: 1, Name: "patch_1.zip"},
			{ID: 3, Name: "patch_3.grf"},
		},
	}
	if v := LocalVersion(state); v != 3 {
		t.Errorf("expected 3, got %d", v)
	}
}

func TestLocalVersionEmpty(t *testing.T) {
	state := &LocalState{}
	if v := LocalVersion(state); v != 0 {
		t.Errorf("expected 0, got %d", v)
	}
}

func TestValidateAgainstManifestOK(t *testing.T) {
	state := &LocalState{
		AppliedPatches: []LocalPatch{
			{ID: 0, Name: "patch_0.grf", Hash: "abc123", Size: 100},
			{ID: 1, Name: "patch_1.zip", Hash: "def456", Size: 200},
		},
	}
	manifest := &Manifest{
		Patches: []Patch{
			{ID: 0, Name: "patch_0.grf", Hash: "abc123", Size: 100},
			{ID: 1, Name: "patch_1.zip", Hash: "def456", Size: 200},
			{ID: 2, Name: "patch_2.grf", Hash: "ghi789", Size: 300},
		},
	}

	id, ok := ValidateAgainstManifest(state, manifest)
	if !ok {
		t.Errorf("expected ok, got mismatch at id %d", id)
	}
}

func TestValidateAgainstManifestHashMismatch(t *testing.T) {
	state := &LocalState{
		AppliedPatches: []LocalPatch{
			{ID: 0, Name: "patch_0.grf", Hash: "abc123", Size: 100},
			{ID: 1, Name: "patch_1.zip", Hash: "WRONG", Size: 200},
		},
	}
	manifest := &Manifest{
		Patches: []Patch{
			{ID: 0, Name: "patch_0.grf", Hash: "abc123", Size: 100},
			{ID: 1, Name: "patch_1.zip", Hash: "def456", Size: 200},
		},
	}

	id, ok := ValidateAgainstManifest(state, manifest)
	if ok {
		t.Error("expected mismatch")
	}
	if id != 1 {
		t.Errorf("expected mismatch at id 1, got %d", id)
	}
}

func TestValidateAgainstManifestSizeMismatch(t *testing.T) {
	state := &LocalState{
		AppliedPatches: []LocalPatch{
			{ID: 0, Name: "patch_0.grf", Hash: "abc123", Size: 999},
		},
	}
	manifest := &Manifest{
		Patches: []Patch{
			{ID: 0, Name: "patch_0.grf", Hash: "abc123", Size: 100},
		},
	}

	id, ok := ValidateAgainstManifest(state, manifest)
	if ok {
		t.Error("expected mismatch")
	}
	if id != 0 {
		t.Errorf("expected mismatch at id 0, got %d", id)
	}
}

func TestValidateAgainstManifestMissingFromManifest(t *testing.T) {
	state := &LocalState{
		AppliedPatches: []LocalPatch{
			{ID: 0, Name: "patch_0.grf", Hash: "abc123", Size: 100},
			{ID: 5, Name: "deleted.grf", Hash: "xxx", Size: 50},
		},
	}
	manifest := &Manifest{
		Patches: []Patch{
			{ID: 0, Name: "patch_0.grf", Hash: "abc123", Size: 100},
		},
	}

	id, ok := ValidateAgainstManifest(state, manifest)
	if ok {
		t.Error("expected mismatch for missing patch")
	}
	if id != 5 {
		t.Errorf("expected mismatch at id 5, got %d", id)
	}
}

func TestValidateAgainstManifestEmptyState(t *testing.T) {
	state := &LocalState{}
	manifest := &Manifest{
		Patches: []Patch{
			{ID: 0, Name: "patch_0.grf", Hash: "abc123", Size: 100},
		},
	}

	id, ok := ValidateAgainstManifest(state, manifest)
	if !ok {
		t.Errorf("expected ok for empty state, got mismatch at id %d", id)
	}
	if id != -1 {
		t.Errorf("expected -1, got %d", id)
	}
}

func TestFindPatchByID(t *testing.T) {
	patches := []Patch{
		{ID: 0, Name: "a.grf"},
		{ID: 1, Name: "b.zip"},
		{ID: 3, Name: "c.grf"},
	}

	p := FindPatchByID(patches, 1)
	if p == nil || p.Name != "b.zip" {
		t.Errorf("expected b.zip, got %v", p)
	}

	p = FindPatchByID(patches, 2)
	if p != nil {
		t.Errorf("expected nil for missing id 2, got %v", p)
	}

	p = FindPatchByID(patches, 3)
	if p == nil || p.Name != "c.grf" {
		t.Errorf("expected c.grf, got %v", p)
	}
}
