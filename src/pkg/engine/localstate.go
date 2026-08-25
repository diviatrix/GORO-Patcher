package engine

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
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
		log.Printf("[localstate] %s unreadable, removing: %v", localStateFile, err)
		os.Remove(path)
		return &LocalState{}, nil
	}

	var state LocalState
	if err := json.Unmarshal(data, &state); err != nil {
		log.Printf("[localstate] %s corrupt, removing: %v", localStateFile, err)
		os.Remove(path)
		return &LocalState{}, nil
	}
	return &state, nil
}

func WriteLocalState(dir string, state *LocalState) error {
	path := filepath.Join(dir, localStateFile)
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0644)
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

func ValidateAgainstManifest(state *LocalState, manifest *Manifest) (int, bool) {
	if state == nil || len(state.AppliedPatches) == 0 {
		return -1, true
	}

	for _, applied := range state.AppliedPatches {
		mp := FindPatchByID(manifest.Patches, applied.ID)
		if mp == nil {
			return applied.ID, false
		}
		if mp.Hash != applied.Hash || mp.Size != applied.Size {
			return applied.ID, false
		}
	}

	return -1, true
}
