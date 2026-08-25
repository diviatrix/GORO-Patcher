package engine

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"

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
	path := filepath.Join(dir, localStateFile)
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return err
	}
	_ = os.Remove(path + ".corrupt")
	return nil
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
	if state == nil || len(state.AppliedPatches) == 0 {
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
			mutable[m] = true
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

	for _, c := range lastRaw {
		if mutable[c.name] {
			continue
		}
		if err := downloader.VerifyFile(filepath.Join(gamePath, c.name), c.hash); err != nil {
			return c.id, false
		}
	}

	openGRFs := make(map[string]*grf.GRF)
	defer func() {
		for _, g := range openGRFs {
			g.Close()
		}
	}()
	for _, c := range lastGRF {
		g, ok := openGRFs[c.grfTarget]
		if !ok {
			var err error
			g, err = grf.Open(filepath.Join(gamePath, c.grfTarget))
			if err != nil {
				return c.id, false
			}
			openGRFs[c.grfTarget] = g
		}
		data, err := g.ReadFile(c.name)
		if err != nil {
			return c.id, false
		}
		if downloader.HashBytes(data) != c.hash {
			return c.id, false
		}
	}

	return -1, true
}

type pendingCheck struct {
	grfTarget string
	name      string
	hash      string
	id        int
}
