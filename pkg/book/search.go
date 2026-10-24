package book

import (
	"encoding/json"
	"html"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

var (
	headingRegex = regexp.MustCompile(`(?i)<h([1-6])(?:\s+id=["']([^"']+)["'])?[^>]*>(.*?)</h[1-6]>`)
	htmlTagRegex = regexp.MustCompile(`<[^>]+>`)
	wordRegex    = regexp.MustCompile(`[\p{L}\p{N}_\-]+`)
)

// SearchDoc represents a searchable document or section in search-index.json.
type SearchDoc struct {
	ID       string   `json:"id"`
	Title    string   `json:"title"`
	Anchor   string   `json:"anchor,omitempty"`
	Path     string   `json:"path"`
	Headings []string `json:"headings,omitempty"`
	Snippet  string   `json:"snippet"`
	Tokens   []string `json:"tokens"`
}

// SearchIndex is the top-level manifest written to search-index.json.
type SearchIndex struct {
	Version     string      `json:"version"`
	BookTitle   string      `json:"book_title"`
	GeneratedAt int64       `json:"generated_at"`
	Docs        []SearchDoc `json:"docs"`
}

// NavItem represents a hierarchical navigation entry in nav.json.
type NavItem struct {
	Label    string    `json:"label"`
	Title    string    `json:"title"`
	Path     string    `json:"path"`
	PrevPath string    `json:"prev,omitempty"`
	NextPath string    `json:"next,omitempty"`
	Subs     []NavItem `json:"subs,omitempty"`
}

// NavManifest represents the complete navigation structure in nav.json.
type NavManifest struct {
	Title    string    `json:"title"`
	Author   string    `json:"author,omitempty"`
	BasePath string    `json:"base_path"`
	Items    []NavItem `json:"items"`
}

// BuildSearchIndex constructs a structured SearchIndex from all chapters in the book.
func (b *Book) BuildSearchIndex() SearchIndex {
	var docs []SearchDoc

	for _, ch := range b.flattened() {
		rawHTML := string(ch.Body)

		// 1. Extract headings and anchor fragments
		var headingTitles []string
		headingMatches := headingRegex.FindAllStringSubmatch(rawHTML, -1)
		for _, m := range headingMatches {
			hText := cleanHTMLText(m[3])
			if hText != "" {
				headingTitles = append(headingTitles, hText)
			}
		}

		// 2. Extract clean plain text for body snippet
		plainText := cleanHTMLText(rawHTML)
		snippet := makeSnippet(plainText, 180)

		// 3. Extract keyword tokens
		tokenSet := make(map[string]struct{})
		addTokens(tokenSet, ch.Title)
		for _, ht := range headingTitles {
			addTokens(tokenSet, ht)
		}
		addTokens(tokenSet, plainText)

		tokens := make([]string, 0, len(tokenSet))
		for tok := range tokenSet {
			tokens = append(tokens, tok)
		}
		sort.Strings(tokens)

		chPath := b.link(ch.Path())
		doc := SearchDoc{
			ID:       ch.Path(),
			Title:    ch.Title,
			Path:     chPath,
			Headings: headingTitles,
			Snippet:  snippet,
			Tokens:   tokens,
		}
		docs = append(docs, doc)

		// Also index individual subheadings (H2, H3) that have anchors
		for _, m := range headingMatches {
			level := m[1]
			anchorID := m[2]
			hText := cleanHTMLText(m[3])

			if (level == "2" || level == "3") && anchorID != "" && hText != "" {
				subTokenSet := make(map[string]struct{})
				addTokens(subTokenSet, ch.Title)
				addTokens(subTokenSet, hText)

				subTokens := make([]string, 0, len(subTokenSet))
				for tok := range subTokenSet {
					subTokens = append(subTokens, tok)
				}
				sort.Strings(subTokens)

				subPath := chPath + "#" + anchorID
				subDoc := SearchDoc{
					ID:      ch.Path() + "#" + anchorID,
					Title:   hText + " (" + ch.Title + ")",
					Anchor:  anchorID,
					Path:    subPath,
					Snippet: hText + " section in " + ch.Title,
					Tokens:  subTokens,
				}
				docs = append(docs, subDoc)
			}
		}
	}

	return SearchIndex{
		Version:     "1.0.0",
		BookTitle:   b.Title,
		GeneratedAt: time.Now().Unix(),
		Docs:        docs,
	}
}

// BuildNavManifest constructs a structural NavManifest representing the book's hierarchy.
func (b *Book) BuildNavManifest() NavManifest {
	manifest := NavManifest{
		Title:    b.Title,
		Author:   b.Author,
		BasePath: b.base,
	}

	for i := range b.Chapters {
		ch := &b.Chapters[i]
		manifest.Items = append(manifest.Items, b.chapterToNavItem(ch))
	}

	return manifest
}

func (b *Book) chapterToNavItem(ch *Chapter) NavItem {
	item := NavItem{
		Label: ch.Label(),
		Title: ch.Title,
		Path:  b.link(ch.Path()),
	}
	if ch.Prev != nil {
		item.PrevPath = b.link(ch.Prev.Path())
	}
	if ch.Next != nil {
		item.NextPath = b.link(ch.Next.Path())
	}
	for _, sub := range ch.Subs {
		item.Subs = append(item.Subs, b.chapterToNavItem(sub))
	}
	return item
}

func exportManifests(outDir, mount string, searchIdx SearchIndex, navManifest NavManifest) error {
	baseOut := outDir
	if mount != "" {
		baseOut = filepath.Join(outDir, filepath.FromSlash(mount))
	}
	if err := os.MkdirAll(baseOut, 0o755); err != nil {
		return err
	}

	searchBytes, err := json.MarshalIndent(searchIdx, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(baseOut, "search-index.json"), searchBytes, 0o644); err != nil {
		return err
	}

	navBytes, err := json.MarshalIndent(navManifest, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(baseOut, "nav.json"), navBytes, 0o644); err != nil {
		return err
	}

	return nil
}

func cleanHTMLText(rawHTML string) string {
	stripped := htmlTagRegex.ReplaceAllString(rawHTML, " ")
	unescaped := html.UnescapeString(stripped)
	fields := strings.Fields(unescaped)
	return strings.Join(fields, " ")
}

func makeSnippet(text string, maxRunes int) string {
	runes := []rune(text)
	if len(runes) <= maxRunes {
		return text
	}
	// Try to cut at space
	cutoff := maxRunes
	for i := maxRunes; i > maxRunes-30 && i < len(runes); i-- {
		if runes[i] == ' ' {
			cutoff = i
			break
		}
	}
	return string(runes[:cutoff]) + "..."
}

func addTokens(set map[string]struct{}, text string) {
	words := wordRegex.FindAllString(text, -1)
	for _, w := range words {
		w = strings.ToLower(strings.TrimSpace(w))
		if len(w) >= 2 && len(w) <= 30 {
			set[w] = struct{}{}
		}
	}
}
