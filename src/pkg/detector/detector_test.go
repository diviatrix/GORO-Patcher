package detector

import (
	"os"
	"runtime"
	"testing"
)

func TestIsGameRunningSelf(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Skip("cannot determine executable")
	}

	name := exe
	if runtime.GOOS == "windows" {
		for i := len(exe) - 1; i >= 0; i-- {
			if exe[i] == '\\' || exe[i] == '/' {
				name = exe[i+1:]
				break
			}
		}
	}

	running, err := IsGameRunning(name)
	if err != nil {
		t.Skipf("detection not available: %v", err)
	}
	if !running {
		t.Error("expected our own process to be detected as running")
	}
}

func TestIsGameRunningNonexistent(t *testing.T) {
	running, err := IsGameRunning("definitely_not_running_process_12345.exe")
	if err != nil {
		t.Skipf("detection not available: %v", err)
	}
	if running {
		t.Error("expected false for nonexistent process")
	}
}
