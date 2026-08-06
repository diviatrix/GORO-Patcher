package engine

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

func LaunchGame(exePath string, args []string) error {
	if exePath == "" {
		return fmt.Errorf("executable path is empty")
	}

	if !filepath.IsAbs(exePath) {
		abs, err := filepath.Abs(exePath)
		if err != nil {
			return fmt.Errorf("resolve path: %w", err)
		}
		exePath = abs
	}

	if _, err := os.Stat(exePath); os.IsNotExist(err) {
		return fmt.Errorf("executable not found: %s", exePath)
	}

	cmd := exec.Command(exePath, args...)

	if runtime.GOOS == "windows" {
		cmd.SysProcAttr = getSysProcAttr()
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("launch game: %w", err)
	}

	return nil
}
