package engine

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGRFTargetsDistinct(t *testing.T) {
	state := &LocalState{AppliedPatches: []LocalPatch{{ID: 1, Name: "p1.grf"}, {ID: 2, Name: "p2.zip"}, {ID: 3, Name: "p3.grf"}}}
	manifest := &Manifest{Patches: []Patch{
		{ID: 1, Type: "grf", Target: "a.grf"},
		{ID: 2, Type: "raw", Target: "x.lub"},
		{ID: 3, Type: "grf", Target: "b.grf"},
		{ID: 4, Type: "grf", Target: "a.grf"},
	}}

	got := GRFTargets(state, manifest)
	want := map[string]bool{"a.grf": true, "b.grf": true}
	if len(got) != len(want) {
		t.Fatalf("expected %d targets, got %d: %v", len(want), len(got), got)
	}
	for _, g := range got {
		if !want[g] {
			t.Errorf("unexpected target %q", g)
		}
	}
}

func TestCheckGRFIntegrityOK(t *testing.T) {
	dir := t.TempDir()
	writeGRF(t, dir, "myserver.grf", map[string]string{"a.txt": "1", "b/c.txt": "2"})

	res := CheckGRFIntegrity(dir, "myserver.grf")
	if !res.OK {
		t.Errorf("expected OK, got failed=%v error=%q", res.Failed, res.Error)
	}
	if res.Checked != 2 {
		t.Errorf("expected checked=2, got %d", res.Checked)
	}
}

func TestCheckGRFIntegrityNotAGRF(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "junk.grf"), []byte("not a grf"), 0644)

	res := CheckGRFIntegrity(dir, "junk.grf")
	if res.OK {
		t.Error("expected failure for non-GRF file")
	}
	if res.Error == "" {
		t.Error("expected an error message")
	}
}

func TestCheckGRFIntegrityMissing(t *testing.T) {
	dir := t.TempDir()
	res := CheckGRFIntegrity(dir, "nope.grf")
	if res.OK {
		t.Error("expected failure for missing GRF")
	}
	if res.Error == "" {
		t.Error("expected an error message")
	}
}