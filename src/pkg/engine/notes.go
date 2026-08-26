package engine

import (
	"encoding/json"
	"net/url"
	"sort"
	"strings"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/ast"
	"github.com/gomarkdown/markdown/html"
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
	doc := markdown.Parse(md, nil)
	if doc != nil {
		sanitizeHTMLAST(doc)
	}
	return string(markdown.Render(doc, html.NewRenderer(html.RendererOptions{})))
}

func sanitizeHTMLAST(n ast.Node) {
	children := n.GetChildren()
	kept := make([]ast.Node, 0, len(children))
	for _, child := range children {
		switch node := child.(type) {
		case *ast.HTMLBlock, *ast.HTMLSpan:
			continue
		case *ast.Link:
			if !isSafeURL(node.Destination) {
				node.Destination = nil
				node.Title = nil
			}
		case *ast.Image:
			if !isSafeURL(node.Destination) {
				continue
			}
		}
		kept = append(kept, child)
		sanitizeHTMLAST(child)
	}
	n.SetChildren(kept)
}

func isSafeURL(dest []byte) bool {
	if len(dest) == 0 {
		return false
	}
	u, err := url.Parse(string(dest))
	if err != nil {
		return false
	}
	switch strings.ToLower(u.Scheme) {
	case "http", "https":
		return u.Host != ""
	default:
		return false
	}
}

func SortNotesByIDDesc(notes []RenderedNote) {
	sort.Slice(notes, func(i, j int) bool {
		return notes[i].ID > notes[j].ID
	})
}
