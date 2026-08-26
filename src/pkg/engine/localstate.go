package engine

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/diviatrix/GORO-Patcher/pkg/downloader"
	"github.com/diviatrix/GORO-Patcher/pkg/grf"
)

const localStateFile = "goro-patch.json"

type LocalPatch struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Hash string `json:"hash"`
	Size int64  `json:"size"`
}

type LocalState struct {
	AppliedPatches []LocalPatch `json:"applied_patches"`
}

func ReadLocalState(dir string) (*LocalState, error) {
	path := filepath.Join(dir, localStateFile)
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &LocalState{}, nil
	}
	if err != nil {
		return nil, err
	}

	var state LocalState
	if err := json.Unmarshal(data, &state); err != nil {
		log.Printf("[localstate] %s corrupt, quarantining: %v", localStateFile, err)
		quarantineCorruptState(path)
		return &LocalState{}, nil
	}
	return &state, nil
}

func quarantineCorruptState(path string) {
	quarantine := path + ".corrupt"
	_ = os.Remove(quarantine)
	if err := os.Rename(path, quarantine); err != nil {
		log.Printf("[localstate] failed to quarantine corrupt state: %v", err)
	}
}

func WriteLocalState(dir string, state *LocalState) error {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if err := atomicWriteFile(filepath.Join(dir, localStateFile), data); err != nil {
		return err
	}
	_ = os.Remove(filepath.Join(dir, localStateFile+".corrupt"))
	return nil
}

func atomicWriteFile(path string, data []byte) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		os.Remove(tmpName)
		return err
	}
	syncDir(dir)
	return nil
}

func syncDir(dir string) {
	d, err := os.Open(dir)
	if err != nil {
		return
	}
	defer d.Close()
	fi, err := d.Stat()
	if err == nil && fi.IsDir() {
		d.Sync()
	}
}

func LocalVersion(state *LocalState) int {
	max := 0
	for _, p := range state.AppliedPatches {
		if p.ID > max {
			max = p.ID
		}
	}
	return max
}

func ValidateAgainstDisk(gamePath string, state *LocalState, manifest *Manifest) (int, bool) {
	if state == nil || manifest == nil || len(state.AppliedPatches) == 0 {
		return -1, true
	}

	V := -1
	for _, applied := range state.AppliedPatches {
		if FindPatchByID(manifest.Patches, applied.ID) != nil && applied.ID > V {
			V = applied.ID
		}
	}
	if V < 0 {
		return -1, true
	}

	mutable := make(map[string]bool)
	lastGRF := make(map[string]pendingCheck)
	lastRaw := make(map[string]pendingCheck)

	for i := range manifest.Patches {
		p := &manifest.Patches[i]
		if p.ID < 0 || p.ID > V {
			continue
		}
		for _, m := range p.Mutable {
			if n := strings.ReplaceAll(m, "\\", "/"); n != "" {
				mutable[n] = true
			}
		}
		switch p.Type {
		case "grf":
			for entry, h := range p.FileHashes {
				key := p.Target + "\x00" + entry
				if cur, ok := lastGRF[key]; !ok || p.ID > cur.id {
					lastGRF[key] = pendingCheck{grfTarget: p.Target, name: entry, hash: h, id: p.ID}
				}
			}
		case "raw":
			for f, h := range p.FileHashes {
				s := SanitizePath(f)
				if s == "" || s != f {
					continue
				}
				if cur, ok := lastRaw[s]; !ok || p.ID > cur.id {
					lastRaw[s] = pendingCheck{name: s, hash: h, id: p.ID}
				}
			}
		}
	}

	mismatchID := -1
	note := func(c pendingCheck) {
		if mismatchID < 0 || c.id < mismatchID {
			mismatchID = c.id
		}
	}

	for _, c := range lastRaw {
		if mutable[c.name] {
			continue
		}
		if err := downloader.VerifyFile(filepath.Join(gamePath, c.name), c.hash); err != nil {
			note(c)
		}
	}

	openGRFs := make(map[string]*grf.GRF)
	defer func() {
		for _, g := range openGRFs {
			g.Close()
		}
	}()
	for _, c := range lastGRF {
		if mutable[strings.ReplaceAll(c.name, "\\", "/")] {
			continue
		}
		g, ok := openGRFs[c.grfTarget]
		if !ok {
			var err error
			g, err = grf.Open(filepath.Join(gamePath, c.grfTarget))
			if err != nil {
				note(c)
				continue
			}
			openGRFs[c.grfTarget] = g
		}
		data, err := g.ReadFile(c.name)
		if err != nil {
			note(c)
			continue
		}
		if downloader.HashBytes(data) != c.hash {
			note(c)
		}
	}

	if mismatchID >= 0 {
		return mismatchID, false
	}
	return -1, true
}

type pendingCheck struct {
	grfTarget string
	name      string
	hash      string
	id        int
}
