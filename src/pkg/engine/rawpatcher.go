package engine

import (
	"archive/zip"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func PatchRaw(gamePath, zipPath string) error {
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("open zip: %w", err)
	}
	defer zr.Close()

	for _, file := range zr.File {
		if file.FileInfo().IsDir() {
			continue
		}

		name := sanitizePath(file.Name)
		if name == "" {
			continue
		}

		dest := filepath.Join(gamePath, name)

		dir := filepath.Dir(dest)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("mkdir %s: %w", dir, err)
		}

		if _, err := os.Stat(dest); err == nil {
			bakPath := filepath.Join(gamePath, ".goro-backups", name+".bak")
			if err := os.MkdirAll(filepath.Dir(bakPath), 0755); err != nil {
				return fmt.Errorf("mkdir backup: %w", err)
			}
			if err := copyFile(dest, bakPath); err != nil {
				return fmt.Errorf("backup %s: %w", dest, err)
			}
		}

		rc, err := file.Open()
		if err != nil {
			return fmt.Errorf("open zip entry %s: %w", file.Name, err)
		}

		out, err := os.Create(dest)
		if err != nil {
			rc.Close()
			return fmt.Errorf("create %s: %w", dest, err)
		}

		_, err = io.Copy(out, rc)
		out.Close()
		rc.Close()
		if err != nil {
			return fmt.Errorf("write %s: %w", dest, err)
		}

		if file.Mode()&0111 != 0 {
			os.Chmod(dest, 0755)
		}
	}

	return nil
}

func sanitizePath(path string) string {

	result := make([]byte, len(path))
	for i, b := range []byte(path) {
		if b == '\\' {
			result[i] = '/'
		} else {
			result[i] = b
		}
	}
	path = string(result)

	for strings.HasPrefix(path, "../") {
		path = path[3:]
	}
	for strings.HasPrefix(path, "/") {
		path = path[1:]
	}

	parts := strings.Split(path, "/")
	var clean []string
	for _, part := range parts {
		if part == ".." || part == "." || part == "" {
			continue
		}
		clean = append(clean, part)
	}

	if len(clean) == 0 {
		return ""
	}

	return strings.Join(clean, "/")
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

func CleanupBackups(gamePath string) error {

	backupsDir := filepath.Join(gamePath, ".goro-backups")

	if _, err := os.Stat(backupsDir); os.IsNotExist(err) {
		return nil
	}

	return os.RemoveAll(backupsDir)
}

func CleanupStaleFiles(gamePath string) error {

	for _, pattern := range []string{"*.patching", "*.new"} {
		matches, _ := filepath.Glob(filepath.Join(gamePath, pattern))
		for _, m := range matches {
			os.Remove(m)
		}
	}

	baks, _ := filepath.Glob(filepath.Join(gamePath, "*.bak"))
	for _, bak := range baks {
		target := bak[:len(bak)-4]
		if needsRestore(target) {
			log.Printf("[recovery] Restoring %s from %s", target, bak)
			os.Rename(bak, target)
		} else {
			os.Remove(bak)
		}
	}

	return reconcileBackups(filepath.Join(gamePath, ".goro-backups"), gamePath)
}

func needsRestore(target string) bool {
	info, err := os.Stat(target)
	if os.IsNotExist(err) {
		return true
	}
	if err == nil {
		if info.Size() == 0 {
			return true
		}
		if isGRFFile(target) && !isValidGRF(target) {
			return true
		}
	}
	return false
}

func reconcileBackups(backupsDir, gamePath string) error {
	if _, err := os.Stat(backupsDir); os.IsNotExist(err) {
		return nil
	}

	return filepath.Walk(backupsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() || !strings.HasSuffix(path, ".bak") {
			return nil
		}

		rel, err := filepath.Rel(backupsDir, path)
		if err != nil {
			return nil
		}
		rel = strings.TrimSuffix(rel, ".bak")

		target := filepath.Join(gamePath, rel)
		if needsRestore(target) {
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return nil
			}
			log.Printf("[recovery] Restoring %s from %s", target, path)
			os.Rename(path, target)
		} else {
			os.Remove(path)
		}
		return nil
	})
}

func isGRFFile(path string) bool {
	return strings.HasSuffix(strings.ToLower(path), ".grf")
}

func isValidGRF(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()

	var magic [15]byte
	if _, err := io.ReadFull(f, magic[:]); err != nil {
		return false
	}
	return string(magic[:]) == "Master of Magic"
}
