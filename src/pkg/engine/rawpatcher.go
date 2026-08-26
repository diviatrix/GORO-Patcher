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
	realBase, err := filepath.EvalSymlinks(gamePath)
	if err != nil {
		return fmt.Errorf("resolve game dir: %w", err)
	}

	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("open zip: %w", err)
	}
	defer zr.Close()

	for _, file := range zr.File {
		if file.FileInfo().IsDir() {
			continue
		}

		name := SanitizePath(file.Name)
		if name == "" {
			continue
		}

		dest := filepath.Join(gamePath, name)

		dir := filepath.Dir(dest)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("mkdir %s: %w", dir, err)
		}

		realDir, err := filepath.EvalSymlinks(dir)
		if err != nil {
			return fmt.Errorf("resolve dir %s: %w", dir, err)
		}
		if !withinPath(realBase, realDir) {
			return fmt.Errorf("refusing to write outside game dir: %s", name)
		}

		if fi, err := os.Lstat(dest); err == nil && fi.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("refusing to write through symlink: %s", name)
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

		tmp, err := os.CreateTemp(dir, filepath.Base(dest)+".goro-*")
		if err != nil {
			rc.Close()
			return fmt.Errorf("create temp for %s: %w", name, err)
		}

		_, err = io.Copy(tmp, rc)
		closeErr := tmp.Close()
		rc.Close()
		if err != nil {
			os.Remove(tmp.Name())
			return fmt.Errorf("write %s: %w", dest, err)
		}
		if closeErr != nil {
			os.Remove(tmp.Name())
			return fmt.Errorf("close temp for %s: %w", dest, closeErr)
		}

		if file.Mode()&0111 != 0 {
			os.Chmod(tmp.Name(), 0755)
		}

		if err := os.Rename(tmp.Name(), dest); err != nil {
			os.Remove(tmp.Name())
			return fmt.Errorf("commit %s: %w", dest, err)
		}
	}

	return nil
}

func withinPath(base, target string) bool {
	rel, err := filepath.Rel(base, target)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)
}

func SanitizePath(path string) string {

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
		if strings.Contains(part, ":") {
			return ""
		}
		clean = append(clean, part)
	}

	if len(clean) == 0 {
		return ""
	}

	return strings.Join(clean, "/")
}

func SafePatchComponent(name string) (string, error) {
	normalized := strings.ReplaceAll(name, "\\", "/")
	if normalized == "" || normalized == "." || normalized == ".." {
		return "", fmt.Errorf("invalid patch filename %q", name)
	}
	if strings.Contains(normalized, "/") {
		return "", fmt.Errorf("patch filename %q must be a bare filename without path separators", name)
	}
	if strings.Contains(name, ":") {
		return "", fmt.Errorf("patch filename %q must not contain ':' (NTFS alternate data stream)", name)
	}
	if strings.TrimRight(name, ". ") != name {
		return "", fmt.Errorf("patch filename %q must not end with a dot or space", name)
	}
	lower := strings.ToLower(name)
	for _, reserved := range windowsReservedNames {
		if lower == reserved || strings.HasPrefix(lower, reserved+".") {
			return "", fmt.Errorf("patch filename %q is a reserved name on Windows", name)
		}
	}
	return name, nil
}

var windowsReservedNames = []string{
	"con", "prn", "aux", "nul",
	"conin$", "conout$", "clock$",
	"com1", "com2", "com3", "com4", "com5", "com6", "com7", "com8", "com9",
	"lpt1", "lpt2", "lpt3", "lpt4", "lpt5", "lpt6", "lpt7", "lpt8", "lpt9",
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
		if !strings.HasSuffix(strings.ToLower(target), ".grf") {
			continue
		}
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
