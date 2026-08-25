package engine

import (
	"encoding/json"
	"sort"
)

type Manifest struct {
	PatchBaseURL      string  `json:"patch_base_url"`
	AllowStartOnError bool    `json:"allow_start_on_error"`
	NewsURL           string  `json:"news_url"`
	PatcherURL        string  `json:"patcher_url"`
	PatcherHash       string  `json:"patcher_hash"`
	PatcherVersion    int     `json:"patcher_version"`
	PatcherSize       int64   `json:"patcher_size"`
	Signature         string  `json:"signature,omitempty"`
	Patches           []Patch `json:"patches"`
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

func (m *Manifest) MaxPatchID() int {
	max := 0
	for _, p := range m.Patches {
		if p.ID > max {
			max = p.ID
		}
	}
	return max
}

func BuildQueue(m *Manifest, localVersion int) []Patch {
	if m == nil {
		return nil
	}

	var queue []Patch
	for _, p := range m.Patches {
		if p.ID > localVersion {
			queue = append(queue, p)
		}
	}

	sort.Slice(queue, func(i, j int) bool {
		return queue[i].ID < queue[j].ID
	})

	return queue
}

func NeedsUpdate(m *Manifest, localVersion int) bool {
	return m != nil && localVersion < m.MaxPatchID()
}

// NeedsSelfUpdate reports whether the manifest advertises a newer patcher build
// worth applying. A path is only offered when the manifest declares both a URL
// and a SHA-256, and only if PatcherVersion strictly exceeds the current build
// number.
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
