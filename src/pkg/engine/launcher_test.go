package engine

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLaunchGameEmptyPath(t *testing.T) {
	err := LaunchGame("", nil)
	if err == nil {
		t.Error("expected error for empty path")
	}
}

func TestLaunchGameNotFound(t *testing.T) {
	err := LaunchGame("/nonexistent/ragexe.exe", nil)
	if err == nil {
		t.Error("expected error for nonexistent executable")
	}
}

func TestLaunchGameValid(t *testing.T) {
	dir := t.TempDir()
	scriptPath := filepath.Join(dir, "test.sh")
	os.WriteFile(scriptPath, []byte("#!/bin/sh\nexit 0\n"), 0755)

	err := LaunchGame(scriptPath, nil)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}
