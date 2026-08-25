package grf

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type ProgressFunc func(current, total int, filename string)

func PatchGRF(grfPath, patchPath string) error {
	return PatchGRFWithProgress(grfPath, patchPath, nil)
}

func PatchGRFWithProgress(grfPath, patchPath string, progress ProgressFunc) error {

	if _, err := os.Stat(grfPath); os.IsNotExist(err) {
		newGrf, err := Create(grfPath)
		if err != nil {
			return fmt.Errorf("create empty GRF: %w", err)
		}
		newGrf.Close()
	}

	ext := strings.ToLower(filepath.Ext(patchPath))

	switch ext {
	case ".grf":
		return MergeGRFWithProgress(grfPath, patchPath, progress)
	case ".zip":
		return patchGRFFromZip(grfPath, patchPath, progress)
	default:
		return fmt.Errorf("unsupported patch type: %s", ext)
	}
}

func patchGRFFromZip(grfPath, zipPath string, progress ProgressFunc) error {
	target, err := Open(grfPath)
	if err != nil {
		return fmt.Errorf("open target GRF: %w", err)
	}
	defer target.Close()

	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("open zip: %w", err)
	}
	defer zr.Close()

	total := 0
	for _, file := range zr.File {
		if !file.FileInfo().IsDir() {
			total++
		}
	}

	current := 0
	for _, file := range zr.File {
		if file.FileInfo().IsDir() {
			continue
		}

		rc, err := file.Open()
		if err != nil {
			return fmt.Errorf("open zip entry %s: %w", file.Name, err)
		}

		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return fmt.Errorf("read zip entry %s: %w", file.Name, err)
		}

		name := normalizeGRFPath(file.Name)

		if err := target.AddFile(name, data); err != nil {
			return fmt.Errorf("add %s to GRF: %w", name, err)
		}

		current++
		if progress != nil {
			progress(current, total, name)
		}
	}

	return target.Save(grfPath)
}

func MergeGRFWithProgress(targetPath, sourcePath string, progress ProgressFunc) error {
	target, err := Open(targetPath)
	if err != nil {
		return fmt.Errorf("open target GRF: %w", err)
	}
	defer target.Close()

	source, err := Open(sourcePath)
	if err != nil {
		return fmt.Errorf("open source GRF: %w", err)
	}
	defer source.Close()

	files := source.ListFiles()
	total := len(files)

	for i, name := range files {
		data, err := source.ReadFile(name)
		if err != nil {
			return fmt.Errorf("read %s from source: %w", source.DisplayName(name), err)
		}

		if err := target.AddFile(name, data); err != nil {
			return fmt.Errorf("add %s to target: %w", source.DisplayName(name), err)
		}

		if progress != nil {
			progress(i+1, total, source.DisplayName(name))
		}
	}

	return target.Save(targetPath)
}

func normalizeGRFPath(path string) string {
	var result []byte
	for _, b := range []byte(path) {
		if b == '/' {
			result = append(result, '\\')
		} else {
			result = append(result, b)
		}
	}
	return string(result)
}
