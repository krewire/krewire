package markdown

import (
	"html"
	"regexp"
	"strings"
)

var (
	htmlTagRe  = regexp.MustCompile(`^</?[a-zA-Z][a-zA-Z0-9:-]*(\s+[^>]*)?/?>`)
	autolinkRe = regexp.MustCompile(`^<((?:https?|mailto|ftp):[^>]+)>`)
	entityRe   = regexp.MustCompile(`^&(?:[a-zA-Z0-9]+|#[0-9]{1,7}|#[xX][0-9a-fA-F]{1,6});`)
)

// renderInlines parses inline Markdown tokens and writes the resulting HTML into sb.
func renderInlines(s string, sb *strings.Builder) {
	n := len(s)
	i := 0

	for i < n {
		// 1. Backtick code span: `code` or ``code``
		if s[i] == '`' {
			count := 1
			for i+count < n && s[i+count] == '`' {
				count++
			}
			ticks := s[i : i+count]
			end := strings.Index(s[i+count:], ticks)
			if end != -1 {
				code := s[i+count : i+count+end]
				// Trim one leading/trailing space if both present per CommonMark
				if len(code) >= 2 && code[0] == ' ' && code[len(code)-1] == ' ' && strings.Trim(code, " ") != "" {
					code = code[1 : len(code)-1]
				}
				sb.WriteString("<code>")
				sb.WriteString(html.EscapeString(code))
				sb.WriteString("</code>")
				i += count + end + count
				continue
			}
		}

		// 2. Autolink: <http://...> or <mailto:...>
		if s[i] == '<' {
			if m := autolinkRe.FindStringSubmatch(s[i:]); m != nil {
				url := m[1]
				sb.WriteString(`<a href="`)
				sb.WriteString(html.EscapeString(url))
				sb.WriteString(`">`)
				sb.WriteString(html.EscapeString(url))
				sb.WriteString(`</a>`)
				i += len(m[0])
				continue
			}

			// Raw HTML Tag pass-through
			if m := htmlTagRe.FindString(s[i:]); m != "" {
				sb.WriteString(m)
				i += len(m)
				continue
			}
		}

		// 3. Image: ![alt](url "title")
		if s[i] == '!' && i+1 < n && s[i+1] == '[' {
			if imgHTML, advanced := tryParseImage(s[i:]); advanced > 0 {
				sb.WriteString(imgHTML)
				i += advanced
				continue
			}
		}

		// 4. Link: [text](url "title")
		if s[i] == '[' {
			if linkHTML, advanced := tryParseLink(s[i:]); advanced > 0 {
				sb.WriteString(linkHTML)
				i += advanced
				continue
			}
		}

		// 5. Strikethrough: ~~text~~
		if strings.HasPrefix(s[i:], "~~") {
			end := strings.Index(s[i+2:], "~~")
			if end != -1 {
				inner := s[i+2 : i+2+end]
				sb.WriteString("<del>")
				renderInlines(inner, sb)
				sb.WriteString("</del>")
				i += 2 + end + 2
				continue
			}
		}

		// 6. Bold/Italic: ***text***, **text**, *text*, ___text___, __text__, _text_
		if strings.HasPrefix(s[i:], "***") || strings.HasPrefix(s[i:], "___") {
			delim := s[i : i+3]
			end := strings.Index(s[i+3:], delim)
			if end != -1 {
				inner := s[i+3 : i+3+end]
				sb.WriteString("<strong><em>")
				renderInlines(inner, sb)
				sb.WriteString("</em></strong>")
				i += 3 + end + 3
				continue
			}
		}

		if strings.HasPrefix(s[i:], "**") || strings.HasPrefix(s[i:], "__") {
			delim := s[i : i+2]
			end := strings.Index(s[i+2:], delim)
			if end != -1 {
				inner := s[i+2 : i+2+end]
				sb.WriteString("<strong>")
				renderInlines(inner, sb)
				sb.WriteString("</strong>")
				i += 2 + end + 2
				continue
			}
		}

		if (s[i] == '*' || s[i] == '_') && (i == 0 || s[i-1] != s[i]) {
			delim := s[i]
			end := strings.IndexByte(s[i+1:], delim)
			if end != -1 && end > 0 {
				inner := s[i+1 : i+1+end]
				// Must not be empty or start/end with spaces
				if inner[0] != ' ' && inner[len(inner)-1] != ' ' {
					sb.WriteString("<em>")
					renderInlines(inner, sb)
					sb.WriteString("</em>")
					i += 1 + end + 1
					continue
				}
			}
		}

		// 7. HTML entities: &amp;, &#123;, etc.
		if s[i] == '&' {
			if m := entityRe.FindString(s[i:]); m != "" {
				sb.WriteString(m)
				i += len(m)
				continue
			}
			sb.WriteString("&amp;")
			i++
			continue
		}

		// 8. Other special HTML characters
		if s[i] == '<' {
			sb.WriteString("&lt;")
			i++
			continue
		}
		if s[i] == '>' {
			sb.WriteString("&gt;")
			i++
			continue
		}
		if s[i] == '"' {
			sb.WriteString("&quot;")
			i++
			continue
		}

		sb.WriteByte(s[i])
		i++
	}
}

func tryParseLink(s string) (string, int) {
	// Look for [text](url)
	if len(s) < 4 || s[0] != '[' {
		return "", 0
	}
	closeBracket := -1
	depth := 0
	for j := 0; j < len(s); j++ {
		if s[j] == '[' {
			depth++
		} else if s[j] == ']' {
			depth--
			if depth == 0 {
				closeBracket = j
				break
			}
		}
	}
	if closeBracket == -1 || closeBracket+1 >= len(s) || s[closeBracket+1] != '(' {
		return "", 0
	}
	closeParen := strings.IndexByte(s[closeBracket+2:], ')')
	if closeParen == -1 {
		return "", 0
	}
	closeParen += closeBracket + 2

	text := s[1:closeBracket]
	rawDest := strings.TrimSpace(s[closeBracket+2 : closeParen])
	urlPart, titlePart := splitURLAndTitle(rawDest)

	var sb strings.Builder
	sb.WriteString(`<a href="`)
	sb.WriteString(html.EscapeString(urlPart))
	sb.WriteString(`"`)
	if titlePart != "" {
		sb.WriteString(` title="`)
		sb.WriteString(html.EscapeString(titlePart))
		sb.WriteString(`"`)
	}
	sb.WriteString(`>`)
	renderInlines(text, &sb)
	sb.WriteString(`</a>`)

	return sb.String(), closeParen + 1
}

func tryParseImage(s string) (string, int) {
	// Look for ![alt](url)
	if len(s) < 5 || s[0] != '!' || s[1] != '[' {
		return "", 0
	}
	linkStr, adv := tryParseLink(s[1:])
	if adv == 0 {
		return "", 0
	}
	_ = linkStr

	closeBracket := strings.IndexByte(s[2:], ']')
	if closeBracket == -1 {
		return "", 0
	}
	closeBracket += 2
	alt := s[2:closeBracket]
	closeParen := strings.IndexByte(s[closeBracket+2:], ')')
	if closeParen == -1 {
		return "", 0
	}
	closeParen += closeBracket + 2
	rawDest := strings.TrimSpace(s[closeBracket+2 : closeParen])
	urlPart, titlePart := splitURLAndTitle(rawDest)

	var sb strings.Builder
	sb.WriteString(`<img src="`)
	sb.WriteString(html.EscapeString(urlPart))
	sb.WriteString(`" alt="`)
	sb.WriteString(html.EscapeString(alt))
	sb.WriteString(`"`)
	if titlePart != "" {
		sb.WriteString(` title="`)
		sb.WriteString(html.EscapeString(titlePart))
		sb.WriteString(`"`)
	}
	sb.WriteString(`>`)

	return sb.String(), closeParen + 1
}

func splitURLAndTitle(s string) (string, string) {
	s = strings.TrimSpace(s)
	// Check for title in quotes e.g. "https://example.com" "title"
	if idx := strings.IndexAny(s, `"'`); idx != -1 {
		url := strings.TrimSpace(s[:idx])
		title := strings.Trim(s[idx:], `"' `)
		return url, title
	}
	return s, ""
}
