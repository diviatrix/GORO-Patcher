package engine

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/diviatrix/GORO-Patcher/pkg/downloader"
	"github.com/diviatrix/GORO-Patcher/pkg/grf"
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
		t.Error("expected corrupt file to be quarantined away from the original path")
	}
	if _, err := os.Stat(filepath.Join(dir, "goro-patch.json.corrupt")); err != nil {
		t.Errorf("expected quarantine file to exist, got: %v", err)
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
	if _, err := os.Stat(filepath.Join(dir, "goro-patch.json.corrupt")); err != nil {
		t.Errorf("expected quarantine file to exist, got: %v", err)
	}
}

func TestReadLocalStateReadErrorNotDeleted(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "goro-patch.json"), 0755); err != nil {
		t.Fatal(err)
	}

	if _, err := ReadLocalState(dir); err == nil {
		t.Fatal("expected error for unreadable (directory) state path")
	}
	if _, err := os.Stat(filepath.Join(dir, "goro-patch.json")); err != nil {
		t.Error("expected unreadable state path to be left untouched")
	}
}

func TestWriteLocalStateClearsQuarantine(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "goro-patch.json.corrupt"), []byte("old garbage"), 0644)

	if err := WriteLocalState(dir, &LocalState{}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "goro-patch.json.corrupt")); !os.IsNotExist(err) {
		t.Error("expected stale quarantine to be removed after a successful write")
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

func TestValidateAgainstDiskEmptyState(t *testing.T) {
	dir := t.TempDir()
	manifest := &Manifest{Patches: []Patch{{ID: 0, Type: "grf", Target: "myserver.grf", FileHashes: map[string]string{"a.txt": "abc"}}}}

	id, ok := ValidateAgainstDisk(dir, &LocalState{}, manifest)
	if !ok {
		t.Errorf("expected ok for empty state, got mismatch at id %d", id)
	}
}

func TestValidateAgainstDiskAllAppliedPruned(t *testing.T) {
	dir := t.TempDir()
	state := &LocalState{AppliedPatches: []LocalPatch{{ID: 5, Name: "gone.grf"}}}
	manifest := &Manifest{}

	id, ok := ValidateAgainstDisk(dir, state, manifest)
	if !ok {
		t.Errorf("expected ok when all applied patches are pruned, got mismatch at id %d", id)
	}
}

func TestValidateAgainstDiskGRFMatch(t *testing.T) {
	dir := t.TempDir()
	writeGRF(t, dir, "myserver.grf", map[string]string{"data/items.lub": "items"})
	hash := downloader.HashBytes([]byte("items"))

	state := &LocalState{AppliedPatches: []LocalPatch{{ID: 3, Name: "p3.grf"}}}
	files := map[string]string{"data/items.lub": hash}
	manifest := &Manifest{Patches: []Patch{{ID: 3, Type: "grf", Target: "myserver.grf", FileHashes: files}}}

	id, ok := ValidateAgainstDisk(dir, state, manifest)
	if !ok {
		t.Errorf("expected ok on matching grf entry, got mismatch at id %d", id)
	}
}

func TestValidateAgainstDiskGRFMismatch(t *testing.T) {
	dir := t.TempDir()
	writeGRF(t, dir, "myserver.grf", map[string]string{"data/items.lub": "real"})

	state := &LocalState{AppliedPatches: []LocalPatch{{ID: 3, Name: "p3.grf"}}}
	files := map[string]string{"data/items.lub": "abcd1234"}
	manifest := &Manifest{Patches: []Patch{{ID: 3, Type: "grf", Target: "myserver.grf", FileHashes: files}}}

	id, ok := ValidateAgainstDisk(dir, state, manifest)
	if ok {
		t.Error("expected grf entry mismatch")
	}
	if id != 3 {
		t.Errorf("expected mismatch at id 3, got %d", id)
	}
}

func TestValidateAgainstDiskGRFMissing(t *testing.T) {
	dir := t.TempDir()
	state := &LocalState{AppliedPatches: []LocalPatch{{ID: 4, Name: "p4.grf"}}}
	manifest := &Manifest{Patches: []Patch{{ID: 4, Type: "grf", Target: "myserver.grf", FileHashes: map[string]string{"a.txt": "abcd1234"}}}}

	id, ok := ValidateAgainstDisk(dir, state, manifest)
	if ok {
		t.Error("expected missing grf to fail verification")
	}
	if id != 4 {
		t.Errorf("expected mismatch at id 4, got %d", id)
	}
}

func TestValidateAgainstDiskGRFEntryMissing(t *testing.T) {
	dir := t.TempDir()
	writeGRF(t, dir, "myserver.grf", map[string]string{"data/other.txt": "x"})
	state := &LocalState{AppliedPatches: []LocalPatch{{ID: 4, Name: "p4.grf"}}}
	manifest := &Manifest{Patches: []Patch{{ID: 4, Type: "grf", Target: "myserver.grf", FileHashes: map[string]string{"data/nope.txt": "abcd1234"}}}}

	id, ok := ValidateAgainstDisk(dir, state, manifest)
	if ok {
		t.Error("expected missing grf entry to fail verification")
	}
	if id != 4 {
		t.Errorf("expected mismatch at id 4, got %d", id)
	}
}

func TestValidateAgainstDiskGRFLastWriter(t *testing.T) {
	dir := t.TempDir()
	writeGRF(t, dir, "myserver.grf", map[string]string{"data/items.lub": "final"})
	hash := downloader.HashBytes([]byte("final"))

	state := &LocalState{AppliedPatches: []LocalPatch{{ID: 1, Name: "p1.grf"}, {ID: 2, Name: "p2.grf"}}}
	manifest := &Manifest{Patches: []Patch{
		{ID: 1, Type: "grf", Target: "myserver.grf", FileHashes: map[string]string{"data/items.lub": "stalehash"}},
		{ID: 2, Type: "grf", Target: "myserver.grf", FileHashes: map[string]string{"data/items.lub": hash}},
	}}

	id, ok := ValidateAgainstDisk(dir, state, manifest)
	if !ok {
		t.Errorf("expected the highest grf entry hash to be used, got mismatch at id %d", id)
	}
}

func TestValidateAgainstDiskRawMatch(t *testing.T) {
	dir := t.TempDir()
	hashA := writeHash(t, dir, "System/itemInfo.lub", []byte("lub data"))
	hashB := writeHash(t, dir, "dinput.dll", []byte("dll bytes"))
	state := &LocalState{AppliedPatches: []LocalPatch{{ID: 2, Name: "p2.zip"}}}
	manifest := &Manifest{Patches: []Patch{
		{ID: 2, Type: "raw", FileHashes: map[string]string{"System/itemInfo.lub": hashA, "dinput.dll": hashB}},
	}}

	id, ok := ValidateAgainstDisk(dir, state, manifest)
	if !ok {
		t.Errorf("expected ok on matching raw files, got mismatch at id %d", id)
	}
}

func TestValidateAgainstDiskRawLastWins(t *testing.T) {
	dir := t.TempDir()
	latest := writeHash(t, dir, "System/itemInfo.lub", []byte("latest content"))
	state := &LocalState{AppliedPatches: []LocalPatch{{ID: 3, Name: "p3.zip"}, {ID: 7, Name: "p7.zip"}}}
	manifest := &Manifest{Patches: []Patch{
		{ID: 3, Type: "raw", FileHashes: map[string]string{"System/itemInfo.lub": "oldhash"}},
		{ID: 7, Type: "raw", FileHashes: map[string]string{"System/itemInfo.lub": latest}},
	}}

	id, ok := ValidateAgainstDisk(dir, state, manifest)
	if !ok {
		t.Errorf("expected last-writer hash to be used, got mismatch at id %d", id)
	}
}

func TestValidateAgainstDiskRawMissingFile(t *testing.T) {
	dir := t.TempDir()
	state := &LocalState{AppliedPatches: []LocalPatch{{ID: 5, Name: "p5.zip"}}}
	manifest := &Manifest{Patches: []Patch{
		{ID: 5, Type: "raw", FileHashes: map[string]string{"System/itemInfo.lub": "abcd1234"}},
	}}

	id, ok := ValidateAgainstDisk(dir, state, manifest)
	if ok {
		t.Error("expected missing raw file to fail verification")
	}
	if id != 5 {
		t.Errorf("expected mismatch at id 5, got %d", id)
	}
}

func TestValidateAgainstDiskRawMutableSkipped(t *testing.T) {
	dir := t.TempDir()
	writeHash(t, dir, "config.ini", []byte("runtime-changed"))
	state := &LocalState{AppliedPatches: []LocalPatch{{ID: 6, Name: "p6.zip"}}}
	manifest := &Manifest{Patches: []Patch{
		{ID: 6, Type: "raw", FileHashes: map[string]string{"config.ini": "abcd1234"}, Mutable: []string{"config.ini"}},
	}}

	id, ok := ValidateAgainstDisk(dir, state, manifest)
	if !ok {
		t.Errorf("expected mutable file to be skipped, got mismatch at id %d", id)
	}
}

func TestValidateAgainstDiskRawTraversalKeyIgnored(t *testing.T) {
	dir := t.TempDir()
	state := &LocalState{AppliedPatches: []LocalPatch{{ID: 6, Name: "p6.zip"}}}
	manifest := &Manifest{Patches: []Patch{
		{ID: 6, Type: "raw", FileHashes: map[string]string{"../../outside.txt": "abcd1234"}},
	}}

	id, ok := ValidateAgainstDisk(dir, state, manifest)
	if !ok {
		t.Errorf("expected traversal key to be ignored, got mismatch at id %d", id)
	}
}

func TestValidateAgainstDiskNoFileHashes(t *testing.T) {
	dir := t.TempDir()
	state := &LocalState{AppliedPatches: []LocalPatch{{ID: 0, Name: "p0.grf"}, {ID: 1, Name: "p1.zip"}}}
	manifest := &Manifest{Patches: []Patch{
		{ID: 0, Type: "grf", Target: "myserver.grf"},
		{ID: 1, Type: "raw", Target: "System/itemInfo.lub"},
	}}

	id, ok := ValidateAgainstDisk(dir, state, manifest)
	if !ok {
		t.Errorf("expected ok when no patches declare file hashes, got mismatch at id %d", id)
	}
}

func writeGRF(t *testing.T, dir, grfName string, entries map[string]string) {
	t.Helper()
	path := filepath.Join(dir, grfName)
	g, err := grf.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	for name, content := range entries {
		if err := g.AddFile(name, []byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := g.Save(path); err != nil {
		t.Fatal(err)
	}
}

func writeHash(t *testing.T, dir, rel string, content []byte) string {
	t.Helper()
	path := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}
	hash, err := downloader.HashFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return hash
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
