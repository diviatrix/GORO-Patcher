package engine

import (
	"strings"
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

// TestRenderMarkdownSanitizes verifies that untrusted markdown — the feed is an
// unauthenticated content channel — cannot survive as live markup or a
// script-capable URL when injected into the DOM via innerHTML.
func TestRenderMarkdownSanitizes(t *testing.T) {
	tests := []struct {
		name        string
		md          string
		notContain  []string
		mustContain []string
	}{
		{
			name:       "raw script tag",
			md:         "<script>alert(1)</script>",
			notContain: []string{"<script", "<script>"},
		},
		{
			name:       "img onerror",
			md:         "<img src=x onerror=alert(1)>",
			notContain: []string{"onerror", "<img"},
		},
		{
			name:       "div onclick attribute",
			md:         "<div onclick=alert(1)>hi</div>",
			notContain: []string{"onclick", "<div"},
		},
		{
			name:       "markdown javascript URL",
			md:         "[click](javascript:alert(1))",
			notContain: []string{"javascript:"},
		},
		{
			name:       "markdown data image URL",
			md:         "![pwn](data:image/svg+xml;base64,SSbmVytex)",
			notContain: []string{"data:", "src="},
		},
		{
			name:        "safe https link preserved",
			md:          "[docs](https://example.com/x)",
			notContain:  []string{"javascript:"},
			mustContain: []string{"https://example.com/x"},
		},
		{
			name:        "code span stays escaped",
			md:          "run `<script>alert(1)</script>`",
			notContain:  []string{"<script>"},
			mustContain: []string{"&lt;script&gt;"},
		},
		{
			name:        "normal prose intact",
			md:          "**bold** and a https link",
			notContain:  []string{"javascript:"},
			mustContain: []string{"<strong>bold</strong>"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RenderMarkdown([]byte(tt.md))
			for _, bad := range tt.notContain {
				if c := strings.Count(got, bad); c > 0 {
					t.Errorf("RenderMarkdown(%q) contains %q (%d occurrence(s)):\n%s",
						tt.md, bad, c, got)
				}
			}
			for _, good := range tt.mustContain {
				if !strings.Contains(got, good) {
					t.Errorf("RenderMarkdown(%q) missing %q:\n%s", tt.md, good, got)
				}
			}
		})
	}
}

func TestSortNotesByIDDescSingle(t *testing.T) {
	notes := []RenderedNote{{ID: 5, Name: "only.md"}}
	SortNotesByIDDesc(notes)
	if notes[0].ID != 5 {
		t.Errorf("notes[0].ID = %d, want 5", notes[0].ID)
	}
}
