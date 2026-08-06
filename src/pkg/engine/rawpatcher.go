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
			bakPath := dest + ".bak"
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
	// Convert backslashes to forward slashes
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
	return filepath.Walk(gamePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && strings.HasSuffix(path, ".bak") {
			os.Remove(path)
		}
		return nil
	})
}

func CleanupStaleFiles(gamePath string) error {
	// 1. Delete stale .patching and .new temp files
	for _, pattern := range []string{"*.patching", "*.new"} {
		matches, _ := filepath.Glob(filepath.Join(gamePath, pattern))
		for _, m := range matches {
			os.Remove(m)
		}
	}

	// 2. Recover .bak files: restore if target is missing or corrupt
	baks, _ := filepath.Glob(filepath.Join(gamePath, "*.bak"))
	for _, bak := range baks {
		target := bak[:len(bak)-4] // remove .bak

		needRestore := false
		info, err := os.Stat(target)
		if os.IsNotExist(err) {
			needRestore = true
		} else if err == nil {
			// Target exists — check if it looks corrupt
			if info.Size() == 0 {
				needRestore = true
			} else if isGRFFile(target) && !isValidGRF(target) {
				needRestore = true
			}
		}

		if needRestore {
			log.Printf("[recovery] Restoring %s from %s", target, bak)
			os.Rename(bak, target)
		} else {
			os.Remove(bak)
		}
	}

	return nil
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
