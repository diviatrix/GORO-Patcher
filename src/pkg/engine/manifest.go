package engine

import (
	"encoding/json"
	"sort"
)

type Manifest struct {
	PatchBaseURL   string  `json:"patch_base_url"`
	PatcherURL     string  `json:"patcher_url"`
	PatcherHash    string  `json:"patcher_hash"`
	PatcherVersion int     `json:"patcher_version"`
	Signature      string  `json:"signature,omitempty"`
	Patches        []Patch `json:"patches"`
}

type Patch struct {
	ID         int               `json:"id"`
	Name       string            `json:"name"`
	Hash       string            `json:"hash"`
	Size       int64             `json:"size"`
	Type       string            `json:"type"`
	Target     string            `json:"target"`
	FileHashes map[string]string `json:"file_hashes,omitempty"`
	Mutable    []string          `json:"mutable,omitempty"`
}

func ParseManifest(data []byte) (*Manifest, error) {
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

func BuildQueue(m *Manifest, localVersion int) []Patch {
	if m == nil {
		return nil
	}

	seen := make(map[int]struct{}, len(m.Patches))
	var queue []Patch
	for _, p := range m.Patches {
		if p.ID < 0 || p.ID <= localVersion {
			continue
		}
		if _, dup := seen[p.ID]; dup {
			continue
		}
		seen[p.ID] = struct{}{}
		queue = append(queue, p)
	}

	sort.Slice(queue, func(i, j int) bool {
		return queue[i].ID < queue[j].ID
	})

	return queue
}

func NeedsSelfUpdate(m *Manifest) bool {
	return m != nil && m.PatcherURL != "" && m.PatcherHash != "" && m.PatcherVersion > CurrentPatcherVersion
}

func FindPatchByID(patches []Patch, id int) *Patch {
	for i := range patches {
		if patches[i].ID == id {
			return &patches[i]
		}
	}
	return nil
}
