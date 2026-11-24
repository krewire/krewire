package book

import "strings"

// crumb is one step in a page's breadcrumb trail.
type crumb struct {
	Title   string
	Link    string
	Current bool
}

// linkRef is a prev/next navigation target.
type linkRef struct {
	Title string
	Link  string
}

// crumbs builds the breadcrumb trail for a chapter page: Home, the containing
// chapter(s), and the page itself.
func (b *Book) crumbs(ch *Chapter) []crumb {
	out := []crumb{{Title: "Home", Link: b.base}}
	var parents []*Chapter
	for p := ch.Parent; p != nil; p = p.Parent {
		parents = append(parents, p)
	}
	for i := len(parents) - 1; i >= 0; i-- {
		out = append(out, crumb{Title: parents[i].Title, Link: b.link(parents[i].Path())})
	}
	if ch != nil {
		out = append(out, crumb{Title: ch.Title, Link: b.link(ch.Path()), Current: true})
	}
	return out
}

func (b *Book) findPage(target string) (*Chapter, bool) {
	cleanTarget := strings.Trim(target, "/")
	if cleanTarget == "" || cleanTarget == "docs" {
		return b.indexChapter, b.indexChapter != nil
	}
	for _, c := range b.flattened() {
		rel := strings.Trim(c.Path(), "/")
		if rel == cleanTarget || c.Slug == cleanTarget {
			return c, true
		}
		if strings.Trim(b.link(c.Path()), "/") == cleanTarget {
			return c, true
		}
	}
	return nil, false
}

func (b *Book) resolveLinkRef(custom string, fallback *Chapter) *linkRef {
	if custom == "none" || custom == "false" {
		return nil
	}
	if custom != "" {
		if ch, ok := b.findPage(custom); ok && ch != nil {
			link := b.link(ch.Path())
			if ch == b.indexChapter {
				link = b.base
			}
			return &linkRef{Title: ch.Title, Link: link}
		}
		return &linkRef{Title: titleFromLink(custom), Link: custom}
	}
	if fallback != nil {
		link := b.link(fallback.Path())
		if fallback == b.indexChapter {
			link = b.base
		}
		return &linkRef{Title: fallback.Title, Link: link}
	}
	return nil
}

func titleFromLink(u string) string {
	parts := strings.Split(strings.Trim(u, "/"), "/")
	if len(parts) == 0 || parts[len(parts)-1] == "" {
		return "Home"
	}
	last := parts[len(parts)-1]
	words := strings.Split(last, "-")
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}

func (b *Book) locale() string {
	for _, part := range []string{b.base, b.mount} {
		trimmed := strings.Trim(part, "/")
		if trimmed == "" {
			continue
		}
		segments := strings.Split(trimmed, "/")
		if len(segments) > 0 && len(segments[0]) == 2 && segments[0] != "to" && segments[0] != "at" {
			return segments[0]
		}
	}
	return "en"
}

func (b *Book) localizedNavLinks() []Link {
	loc := b.locale()
	if loc == "" || loc == "en" {
		return b.navLinks
	}
	prefix := "/" + loc + "/"
	out := make([]Link, len(b.navLinks))
	for i, l := range b.navLinks {
		out[i] = l
		if strings.HasPrefix(l.URL, "/") && !strings.HasPrefix(l.URL, "/#") && !strings.HasPrefix(l.URL, prefix) {
			out[i].URL = "/" + loc + l.URL
		}
	}
	return out
}
