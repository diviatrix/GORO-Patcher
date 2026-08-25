package grf

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestSaveDeterministic(t *testing.T) {
	build := func(path string) {
		g, err := Create(path)
		if err != nil {
			t.Fatal(err)
		}
		files := []struct{ name, data string }{
			{"zeta.txt", "zzz"},
			{"alpha.txt", "aaa"},
			{"beta.txt", "bbb"},
			{"data/sprite/x.spr", "xxxx"},
		}
		for _, f := range files {
			if err := g.AddFile(f.name, []byte(f.data)); err != nil {
				t.Fatal(err)
			}
		}
		if err := g.Save(path); err != nil {
			t.Fatal(err)
		}
	}

	dir := t.TempDir()
	a := filepath.Join(dir, "a.grf")
	b := filepath.Join(dir, "b.grf")
	build(a)
	build(b)

	aBytes, err := os.ReadFile(a)
	if err != nil {
		t.Fatal(err)
	}
	bBytes, err := os.ReadFile(b)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(aBytes, bBytes) {
		t.Error("GRF save is not deterministic across runs")
	}
}