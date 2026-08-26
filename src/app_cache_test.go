package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/diviatrix/GORO-Patcher/pkg/downloader"
	"github.com/diviatrix/GORO-Patcher/pkg/engine"
)

func TestCachePathFor(t *testing.T) {
	p := engine.Patch{Name: "foo.patch", Hash: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}
	got := cachePathFor("/g", p)
	want := filepath.Join("/g", ".goro-patches", "foo.patch.aaaaaaaaaaaa.patch")
	if got != want {
		t.Errorf("cachePathFor = %q, want %q", got, want)
	}
}

func TestAcquirePatchUsesCache(t *testing.T) {
	dir := t.TempDir()
	a := &App{gamePath: dir}
	content := []byte("verified patch payload")
	p := engine.Patch{Name: "one.patch", Hash: downloader.HashBytes(content)}

	cacheDir := filepath.Join(dir, ".goro-patches")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cachePathFor(dir, p), content, 0644); err != nil {
		t.Fatal(err)
	}

	path, cached, err := a.acquirePatch(context.Background(), "", p)
	if err != nil {
		t.Fatal(err)
	}
	if !cached {
		t.Error("expected cache hit")
	}
	if path != cachePathFor(dir, p) {
		t.Errorf("path = %q, want %q", path, cachePathFor(dir, p))
	}
}

func TestAcquirePatchCacheMissRejectsEmptyBase(t *testing.T) {
	dir := t.TempDir()
	a := &App{gamePath: dir}
	p := engine.Patch{Name: "two.patch", Hash: downloader.HashBytes([]byte("x"))}

	_, cached, err := a.acquirePatch(context.Background(), "", p)
	if err == nil {
		t.Fatal("expected error when cache misses and patch_base_url is empty")
	}
	if cached {
		t.Error("expected not cached on miss")
	}
}

func TestCachePatchMovesAndSkips(t *testing.T) {
	dir := t.TempDir()
	a := &App{gamePath: dir}
	content := []byte("data")
	p := engine.Patch{Name: "three.patch", Hash: downloader.HashBytes(content)}

	src := filepath.Join(dir, "three.patch")
	if err := os.WriteFile(src, content, 0644); err != nil {
		t.Fatal(err)
	}

	a.cachePatch(p, src, false)
	if _, err := os.Stat(cachePathFor(dir, p)); err != nil {
		t.Fatalf("expected cache entry created: %v", err)
	}
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Error("expected source removed after caching")
	}

	keep := filepath.Join(dir, "keep.patch")
	if err := os.WriteFile(keep, content, 0644); err != nil {
		t.Fatal(err)
	}
	a.cachePatch(p, keep, true)
	if _, err := os.Stat(keep); err != nil {
		t.Error("fromCache must not move the file")
	}
}

func TestPrunePatchCache(t *testing.T) {
	dir := t.TempDir()
	a := &App{gamePath: dir}
	cacheDir := filepath.Join(dir, ".goro-patches")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		t.Fatal(err)
	}

	p1 := engine.Patch{Name: "a.patch", Hash: downloader.HashBytes([]byte("1"))}
	p2 := engine.Patch{Name: "b.patch", Hash: downloader.HashBytes([]byte("2"))}
	os.WriteFile(cachePathFor(dir, p1), []byte("1"), 0644)
	os.WriteFile(cachePathFor(dir, p2), []byte("2"), 0644)
	os.WriteFile(filepath.Join(cacheDir, "stale.patch"), []byte("3"), 0644)

	m := &engine.Manifest{Patches: []engine.Patch{p1}}
	a.prunePatchCache(m)

	if _, err := os.Stat(cachePathFor(dir, p1)); err != nil {
		t.Error("expected p1 retained")
	}
	if _, err := os.Stat(cachePathFor(dir, p2)); !os.IsNotExist(err) {
		t.Error("expected p2 pruned as stale")
	}
	if _, err := os.Stat(filepath.Join(cacheDir, "stale.patch")); !os.IsNotExist(err) {
		t.Error("expected unrelated cache file pruned")
	}
}
