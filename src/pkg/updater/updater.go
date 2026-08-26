package updater

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/diviatrix/GORO-Patcher/pkg/downloader"
)

type Updater struct {
	dl          *downloader.Downloader
	currentPath string
}

func New(dl *downloader.Downloader) (*Updater, error) {
	exe, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("get executable path: %w", err)
	}
	abs, err := filepath.Abs(exe)
	if err != nil {
		return nil, fmt.Errorf("resolve executable path: %w", err)
	}

	return &Updater{
		dl:          dl,
		currentPath: abs,
	}, nil
}

func (u *Updater) Update(ctx context.Context, url, expectedHash string) (bool, error) {
	f, err := os.CreateTemp(filepath.Dir(u.currentPath), filepath.Base(u.currentPath)+".update-*")
	if err != nil {
		return false, fmt.Errorf("create update temp: %w", err)
	}
	tmpPath := f.Name()
	f.Close()

	if err := u.dl.Fetch(ctx, url, tmpPath, nil); err != nil {
		os.Remove(tmpPath)
		return false, fmt.Errorf("download update: %w", err)
	}

	if err := syncFile(tmpPath); err != nil {
		os.Remove(tmpPath)
		return false, fmt.Errorf("sync update: %w", err)
	}

	if err := downloader.VerifyFile(tmpPath, expectedHash); err != nil {
		os.Remove(tmpPath)
		return false, fmt.Errorf("verify update: %w", err)
	}

	if err := u.replaceBinary(tmpPath); err != nil {
		os.Remove(tmpPath)
		return false, fmt.Errorf("replace binary: %w", err)
	}

	return true, nil
}

func syncFile(path string) error {
	f, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		return err
	}
	defer f.Close()
	return f.Sync()
}

func (u *Updater) replaceBinary(newPath string) error {
	if runtime.GOOS == "windows" {
		return u.replaceBinaryWindows(newPath)
	}
	return u.replaceBinaryUnix(newPath)
}

func (u *Updater) replaceBinaryUnix(newPath string) error {
	bakPath := u.currentPath + ".bak"

	if err := os.Rename(u.currentPath, bakPath); err != nil {
		return fmt.Errorf("backup current: %w", err)
	}

	if err := os.Rename(newPath, u.currentPath); err != nil {
		os.Rename(bakPath, u.currentPath)
		return fmt.Errorf("move new binary: %w", err)
	}

	os.Remove(bakPath)
	return nil
}

func (u *Updater) replaceBinaryWindows(newPath string) error {
	batPath := u.currentPath + "_update.bat"

	bat := fmt.Sprintf(`@echo off
:loop
tasklist /FI "PID eq %d" 2>nul | find /i "%s" >nul
if not errorlevel 1 (
    timeout /t 1 /nobreak >nul
    goto loop
)
move /y "%s" "%s.bak" >nul
move /y "%s" "%s" >nul
del "%s.bak" >nul
start "" "%s"
del "%%~f0"
`, os.Getpid(), filepath.Base(u.currentPath),
		u.currentPath, u.currentPath,
		newPath, u.currentPath,
		u.currentPath,
		u.currentPath)

	if err := os.WriteFile(batPath, []byte(bat), 0755); err != nil {
		return fmt.Errorf("write update script: %w", err)
	}

	cmd := exec.Command("cmd", "/C", batPath)
	cmd.SysProcAttr = getSysProcAttrWindows()
	if err := cmd.Start(); err != nil {
		os.Remove(batPath)
		return fmt.Errorf("launch update script: %w", err)
	}

	return nil
}

func (u *Updater) Restart() (bool, error) {
	cmd := exec.Command(u.currentPath)
	if err := cmd.Start(); err != nil {
		return false, fmt.Errorf("restart: %w", err)
	}
	return true, nil
}
