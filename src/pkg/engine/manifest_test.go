package engine

import "testing"

func TestParseManifest(t *testing.T) {
	data := []byte(`{
		"allow_start_on_error": false,
		"news_url": "https://example.com/news.html",
		"patcher_url": "https://cdn.example.com/patcher.exe",
		"patcher_hash": "aabbccdd",
		"patcher_size": 5242880,
		"patches": [
			{
				"id": 104,
				"name": "2026-08-01_items.zip",
				"hash": "8f4e3c2b",
				"size": 1048576,
				"type": "grf",
				"target": "myserver.grf",
				"file_hashes": { "data/items.lub": "abcd1234" }
			},
			{
				"id": 105,
				"name": "2026-08-04_system.zip",
				"hash": "1a2b3c4d",
				"size": 524288,
				"type": "raw",
				"target": "System/itemInfo.lub",
				"file_hashes": { "System/itemInfo.lub": "ef567890", "dinput.dll": "a1b2c3d4" },
				"mutable": ["config.ini"]
			}
		]
	}`)

	m, err := ParseManifest(data)
	if err != nil {
		t.Fatal(err)
	}

	if len(m.Patches) != 2 {
		t.Errorf("patches: got %d, want 2", len(m.Patches))
	}
	if m.Patches[0].Type != "grf" {
		t.Errorf("patch[0].type: got %s, want grf", m.Patches[0].Type)
	}
	if m.Patches[1].Target != "System/itemInfo.lub" {
		t.Errorf("patch[1].target: got %s, want System/itemInfo.lub", m.Patches[1].Target)
	}
	if m.Patches[0].FileHashes["data/items.lub"] != "abcd1234" {
		t.Errorf("patch[0].file_hashes[data/items.lub]: got %s, want abcd1234", m.Patches[0].FileHashes["data/items.lub"])
	}
	if len(m.Patches[1].Mutable) != 1 || m.Patches[1].Mutable[0] != "config.ini" {
		t.Errorf("patch[1].mutable: got %v, want [config.ini]", m.Patches[1].Mutable)
	}
	if m.MaxPatchID() != 105 {
		t.Errorf("MaxPatchID: got %d, want 105", m.MaxPatchID())
	}
}

func TestParseManifestMalformed(t *testing.T) {
	_, err := ParseManifest([]byte("not json"))
	if err == nil {
		t.Error("expected error for malformed JSON")
	}
}

func TestBuildQueue(t *testing.T) {
	m := &Manifest{
		Patches: []Patch{
			{ID: 100, Name: "a.zip"},
			{ID: 102, Name: "b.zip"},
			{ID: 104, Name: "c.zip"},
			{ID: 105, Name: "d.zip"},
		},
	}

	tests := []struct {
		local   int
		wantIDs []int
	}{
		{0, []int{100, 102, 104, 105}},
		{100, []int{102, 104, 105}},
		{102, []int{104, 105}},
		{104, []int{105}},
		{105, nil},
		{200, nil},
	}

	for _, tt := range tests {
		queue := BuildQueue(m, tt.local)
		if len(queue) != len(tt.wantIDs) {
			t.Errorf("local=%d: got %d patches, want %d", tt.local, len(queue), len(tt.wantIDs))
			continue
		}
		for i, p := range queue {
			if p.ID != tt.wantIDs[i] {
				t.Errorf("local=%d: queue[%d].ID = %d, want %d", tt.local, i, p.ID, tt.wantIDs[i])
			}
		}
	}
}

func TestBuildQueueSort(t *testing.T) {
	m := &Manifest{
		Patches: []Patch{
			{ID: 105, Name: "d.zip"},
			{ID: 100, Name: "a.zip"},
			{ID: 104, Name: "c.zip"},
			{ID: 102, Name: "b.zip"},
		},
	}

	queue := BuildQueue(m, 0)
	for i := 1; i < len(queue); i++ {
		if queue[i].ID < queue[i-1].ID {
			t.Errorf("queue not sorted: %d < %d at index %d", queue[i].ID, queue[i-1].ID, i)
		}
	}
}

func TestBuildQueueNil(t *testing.T) {
	if q := BuildQueue(nil, 0); q != nil {
		t.Errorf("expected nil, got %v", q)
	}
}

func TestNeedsUpdate(t *testing.T) {
	m := &Manifest{Patches: []Patch{{ID: 105}}}
	if !NeedsUpdate(m, 104) {
		t.Error("expected true for local < remote")
	}
	if NeedsUpdate(m, 105) {
		t.Error("expected false for local == remote")
	}
	if NeedsUpdate(m, 200) {
		t.Error("expected false for local > remote")
	}
	if NeedsUpdate(nil, 0) {
		t.Error("expected false for nil manifest")
	}
}

func TestNeedsSelfUpdate(t *testing.T) {
	if NeedsSelfUpdate(nil) {
		t.Error("expected false for nil")
	}
	if NeedsSelfUpdate(&Manifest{}) {
		t.Error("expected false for empty patcher fields")
	}
	if NeedsSelfUpdate(&Manifest{PatcherURL: "x", PatcherHash: "y"}) {
		t.Error("expected false when version not newer (no downgrade)")
	}
	if !NeedsSelfUpdate(&Manifest{PatcherURL: "x", PatcherHash: "y", PatcherVersion: CurrentPatcherVersion + 1}) {
		t.Error("expected true when both fields set and version is newer")
	}
}
