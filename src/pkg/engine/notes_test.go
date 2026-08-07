package engine

import (
	"testing"
)

func TestParseNotesManifest(t *testing.T) {
	data := []byte(`{
		"notes_base_url": "https://example.com/notes/",
		"notes_css_url": "https://example.com/notes/design.css",
		"notes": [
			{"id": 0, "name": "note_0.md"},
			{"id": 1, "name": "note_1.md"}
		]
	}`)

	m, err := ParseNotesManifest(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if m.NotesBaseURL != "https://example.com/notes/" {
		t.Errorf("NotesBaseURL = %q, want %q", m.NotesBaseURL, "https://example.com/notes/")
	}
	if m.NotesCSSURL != "https://example.com/notes/design.css" {
		t.Errorf("NotesCSSURL = %q, want %q", m.NotesCSSURL, "https://example.com/notes/design.css")
	}
	if len(m.Notes) != 2 {
		t.Fatalf("len(Notes) = %d, want 2", len(m.Notes))
	}
	if m.Notes[0].ID != 0 || m.Notes[0].Name != "note_0.md" {
		t.Errorf("Notes[0] = {%d, %q}, want {0, note_0.md}", m.Notes[0].ID, m.Notes[0].Name)
	}
}

func TestParseNotesManifestEmptyCSS(t *testing.T) {
	data := []byte(`{
		"notes_base_url": "https://example.com/notes/",
		"notes": []
	}`)

	m, err := ParseNotesManifest(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if m.NotesCSSURL != "" {
		t.Errorf("NotesCSSURL = %q, want empty", m.NotesCSSURL)
	}
	if len(m.Notes) != 0 {
		t.Errorf("len(Notes) = %d, want 0", len(m.Notes))
	}
}

func TestParseNotesManifestInvalid(t *testing.T) {
	_, err := ParseNotesManifest([]byte(`{invalid`))
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}

func TestRenderMarkdown(t *testing.T) {
	tests := []struct {
		name string
		md   string
		want string
	}{
		{
			name: "heading",
			md:   "# Hello",
			want: "<h1>Hello</h1>\n",
		},
		{
			name: "bold",
			md:   "**bold**",
			want: "<p><strong>bold</strong></p>\n",
		},
		{
			name: "paragraph",
			md:   "Hello world",
			want: "<p>Hello world</p>\n",
		},
		{
			name: "list",
			md:   "- item 1\n- item 2",
			want: "<ul>\n<li>item 1</li>\n<li>item 2</li>\n</ul>\n",
		},
		{
			name: "link",
			md:   "[link](https://example.com)",
			want: `<p><a href="https://example.com">link</a></p>` + "\n",
		},
		{
			name: "empty",
			md:   "",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RenderMarkdown([]byte(tt.md))
			if got != tt.want {
				t.Errorf("RenderMarkdown(%q) = %q, want %q", tt.md, got, tt.want)
			}
		})
	}
}

func TestSortNotesByIDDesc(t *testing.T) {
	notes := []RenderedNote{
		{ID: 1, Name: "b.md"},
		{ID: 3, Name: "d.md"},
		{ID: 0, Name: "a.md"},
		{ID: 2, Name: "c.md"},
	}

	SortNotesByIDDesc(notes)

	expected := []int{3, 2, 1, 0}
	for i, id := range expected {
		if notes[i].ID != id {
			t.Errorf("notes[%d].ID = %d, want %d", i, notes[i].ID, id)
		}
	}
}

func TestSortNotesByIDDescEmpty(t *testing.T) {
	var notes []RenderedNote
	SortNotesByIDDesc(notes)
	if len(notes) != 0 {
		t.Errorf("expected empty slice, got %d items", len(notes))
	}
}

func TestSortNotesByIDDescSingle(t *testing.T) {
	notes := []RenderedNote{{ID: 5, Name: "only.md"}}
	SortNotesByIDDesc(notes)
	if notes[0].ID != 5 {
		t.Errorf("notes[0].ID = %d, want 5", notes[0].ID)
	}
}
