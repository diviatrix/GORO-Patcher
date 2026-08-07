package engine

import (
	"encoding/json"
	"sort"

	"github.com/gomarkdown/markdown"
)

type NotesManifest struct {
	NotesBaseURL string      `json:"notes_base_url"`
	NotesCSSURL  string      `json:"notes_css_url"`
	Notes        []NoteEntry `json:"notes"`
}

type NoteEntry struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type RenderedNote struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	Content string `json:"content"`
}

func ParseNotesManifest(data []byte) (*NotesManifest, error) {
	var m NotesManifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

func RenderMarkdown(md []byte) string {
	return string(markdown.ToHTML(md, nil, nil))
}

func SortNotesByIDDesc(notes []RenderedNote) {
	sort.Slice(notes, func(i, j int) bool {
		return notes[i].ID > notes[j].ID
	})
}
