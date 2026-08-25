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

// RenderMarkdown renders a note's markdown to HTML, sanitized so the result is
// safe to inject into the page DOM (the frontend uses innerHTML). Notes are an
// unauthenticated content channel, so raw HTML from the source must never
// survive as live markup. Sanitizing at the markdown AST keeps the output
// limited to safe gomarkdown tags with http(s)-only destinations, and avoids
// the subtle bugs of hand-parsing an HTML string.
func RenderMarkdown(md []byte) string {
	doc := markdown.Parse(md, nil)
	if doc != nil {
		sanitizeHTMLAST(doc)
	}
	return string(markdown.Render(doc, html.NewRenderer(html.RendererOptions{})))
}

// sanitizeHTMLAST prunes a parsed markdown tree so it can render nothing that
// executes script or exfiltrates data. Raw-HTML nodes (the vehicle for DOM
// XSS: <script>, <img onerror>, <div onclick>, …) are dropped entirely, and
// link/image destinations are kept only when they are http(s) URLs.
// Inline code spans are left untouched — gomarkdown HTML-escapes their contents,
// so `` `<script>` `` cannot smuggle markup out.
func sanitizeHTMLAST(n ast.Node) {
	children := n.GetChildren()
	kept := make([]ast.Node, 0, len(children))
	for _, child := range children {
		switch node := child.(type) {
		case *ast.HTMLBlock, *ast.HTMLSpan:
			// Raw HTML never reaches the rendered string.
			continue
		case *ast.Link:
			if !isSafeURL(node.Destination) {
				// Keep the link but strip its href — renders as bare text.
				node.Destination = nil
				node.Title = nil
			}
		case *ast.Image:
			if !isSafeURL(node.Destination) {
				// Drop the whole image so no src (empty or data:) is emitted.
				continue
			}
		}
		kept = append(kept, child)
		sanitizeHTMLAST(child)
	}
	n.SetChildren(kept)
}

// isSafeURL reports whether a link/image destination is an http(s) URL and not
// a script-capable or local scheme (javascript:, data:, file:, vbscript:,
// mailto:, relative paths, …).
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
