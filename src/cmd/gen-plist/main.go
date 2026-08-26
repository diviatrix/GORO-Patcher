package main

import (
	"archive/zip"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/diviatrix/GORO-Patcher/pkg/downloader"
	"github.com/diviatrix/GORO-Patcher/pkg/engine"
	"github.com/diviatrix/GORO-Patcher/pkg/grf"
)

func main() {
	in := flag.String("in", "", "input plist.json")
	patchesDir := flag.String("patches", "", "directory containing the patch files named by plist entries")
	out := flag.String("out", "-", "output plist.json path (\"-\" for stdout)")
	flag.Parse()

	if *in == "" || *patchesDir == "" {
		flag.Usage()
		os.Exit(2)
	}

	data, err := os.ReadFile(*in)
	if err != nil {
		fatal("read input plist: %v", err)
	}

	var manifest engine.Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		fatal("parse input plist: %v", err)
	}

	scratch, err := os.MkdirTemp("", "goro-gen-plist-")
	if err != nil {
		fatal("create scratch dir: %v", err)
	}
	defer os.RemoveAll(scratch)

	for i := range manifest.Patches {
		p := &manifest.Patches[i]
		src := filepath.Join(*patchesDir, p.Name)
		switch p.Type {
		case "grf":
			p.FileHashes = grfEntryHashes(src)
		case "raw":
			hashes, err := extractHashes(src, filepath.Join(scratch, "raw"))
			if err != nil {
				fatal("extract raw patch %d: %v", p.ID, err)
			}
			p.FileHashes = hashes
		}
	}

	outData, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		fatal("marshal plist: %v", err)
	}
	outData = append(outData, '\n')

	if *out == "-" {
		os.Stdout.Write(outData)
		return
	}
	if err := os.WriteFile(*out, outData, 0644); err != nil {
		fatal("write output: %v", err)
	}
}

func grfEntryHashes(patchPath string) map[string]string {
	out := make(map[string]string)
	ext := strings.ToLower(filepath.Ext(patchPath))
	switch ext {
	case ".zip":
		zr, err := zip.OpenReader(patchPath)
		if err != nil {
			fatal("open grf zip %s: %v", patchPath, err)
		}
		defer zr.Close()
		for _, f := range zr.File {
			if f.FileInfo().IsDir() {
				continue
			}
			rc, err := f.Open()
			if err != nil {
				fatal("open zip entry %s: %v", f.Name, err)
			}
			content, err := io.ReadAll(rc)
			rc.Close()
			if err != nil {
				fatal("read zip entry %s: %v", f.Name, err)
			}
			out[normalizeGRFPath(f.Name)] = downloader.HashBytes(content)
		}
	case ".grf":
		g, err := grf.Open(patchPath)
		if err != nil {
			fatal("open grf patch %s: %v", patchPath, err)
		}
		defer g.Close()
		for _, name := range g.ListFiles() {
			content, err := g.ReadFile(name)
			if err != nil {
				fatal("read grf entry %s: %v", name, err)
			}
			out[name] = downloader.HashBytes(content)
		}
	default:
		fatal("unsupported grf patch extension: %s", ext)
	}
	return out
}

func normalizeGRFPath(path string) string {
	return strings.ReplaceAll(path, "/", "\\")
}

func extractHashes(zipPath, scratchDir string) (map[string]string, error) {
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, err
	}
	defer zr.Close()

	out := make(map[string]string)
	for _, f := range zr.File {
		if f.FileInfo().IsDir() {
			continue
		}
		name := engine.SanitizePath(f.Name)
		if name == "" {
			continue
		}

		rc, err := f.Open()
		if err != nil {
			return nil, fmt.Errorf("open %s: %w", f.Name, err)
		}

		dst := filepath.Join(scratchDir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
			rc.Close()
			return nil, fmt.Errorf("mkdir %s: %w", dst, err)
		}

		outFile, err := os.Create(dst)
		if err != nil {
			rc.Close()
			return nil, fmt.Errorf("create %s: %w", dst, err)
		}
		if _, err := io.Copy(outFile, rc); err != nil {
			outFile.Close()
			rc.Close()
			return nil, fmt.Errorf("write %s: %w", dst, err)
		}
		outFile.Close()
		rc.Close()

		hash, err := downloader.HashFile(dst)
		if err != nil {
			return nil, fmt.Errorf("hash %s: %w", dst, err)
		}
		out[name] = hash
	}
	return out, nil
}

func fatal(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "gen-plist: "+strings.TrimSuffix(format, "\n")+"\n", args...)
	os.Exit(1)
}
