package markdown

import (
	"fmt"
	"html"
	"regexp"
	"strings"
)

var (
	atxHeadingRe = regexp.MustCompile(`^(#{1,6})\s+(.*?)(?:\s+#+)?$`)
	tableSepRe   = regexp.MustCompile(`^\|?(\s*:?-+:?\s*\|?)+$`)
	hrRe         = regexp.MustCompile(`^(\*{3,}|-{3,}|_{3,})$`)
	ulItemRe     = regexp.MustCompile(`^(\s*)[-*+]\s+(.*)$`)
	olItemRe     = regexp.MustCompile(`^(\s*)[0-9]+\.\s+(.*)$`)
)

type parseContext struct {
	slugs *slugTracker
}

func parseAndRenderBlocks(src []byte) string {
	text := string(src)
	// Normalize CRLF to LF
	text = strings.ReplaceAll(text, "\r\n", "\n")
	lines := strings.Split(text, "\n")

	ctx := &parseContext{
		slugs: newSlugTracker(),
	}

	sb := GetBuffer()
	defer PutBuffer(sb)
	i := 0
	n := len(lines)

	for i < n {
		line := lines[i]
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			i++
			continue
		}

		// 1. Fenced Code Block: ``` or ~~~
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			codeHTML, consumed := parseFencedCode(lines, i)
			sb.WriteString(codeHTML)
			i += consumed
			continue
		}

		// 2. ATX Heading: # Title
		if headingHTML, ok := parseATXHeading(trimmed, ctx); ok {
			sb.WriteString(headingHTML)
			i++
			continue
		}

		// 3. Setext Heading: line followed by === or ---
		if headingHTML, consumed, ok := parseSetextHeading(lines, i, ctx); ok {
			sb.WriteString(headingHTML)
			i += consumed
			continue
		}

		// 4. Horizontal Rule: ---, ***, ___
		if hrRe.MatchString(trimmed) && len(trimmed) >= 3 {
			sb.WriteString("<hr>\n")
			i++
			continue
		}

		// 5. Blockquote: > text
		if strings.HasPrefix(trimmed, ">") {
			bqHTML, consumed := parseBlockquote(lines, i)
			sb.WriteString(bqHTML)
			i += consumed
			continue
		}

		// 6. GFM Table: | a | b |
		if strings.Contains(line, "|") && i+1 < n && isTableSeparator(lines[i+1]) {
			tableHTML, consumed := parseTable(lines[i:])
			if consumed > 0 {
				sb.WriteString(tableHTML)
				i += consumed
				continue
			}
		}

		// 7. Unordered List: - item, * item, + item
		if ulItemRe.MatchString(line) {
			listHTML, consumed := parseList(lines[i:], false)
			if consumed > 0 {
				sb.WriteString(listHTML)
				i += consumed
				continue
			}
		}

		// 8. Ordered List: 1. item
		if olItemRe.MatchString(line) {
			listHTML, consumed := parseList(lines[i:], true)
			if consumed > 0 {
				sb.WriteString(listHTML)
				i += consumed
				continue
			}
		}

		// 9. Raw HTML Block: starts with <tag or </tag
		if isHTMLBlockStart(trimmed) {
			htmlBlock, consumed := parseHTMLBlock(lines, i)
			sb.WriteString(htmlBlock)
			i += consumed
			continue
		}

		// 10. Paragraph
		pHTML, consumed := parseParagraph(lines, i)
		sb.WriteString(pHTML)
		i += consumed
	}

	return sb.String()
}

func parseFencedCode(lines []string, i int) (string, int) {
	n := len(lines)
	trimmed := strings.TrimSpace(lines[i])
	fence := trimmed[:3]
	lang := strings.TrimSpace(trimmed[3:])
	if idx := strings.IndexAny(lang, " \t"); idx != -1 {
		lang = lang[:idx]
	}
	start := i + 1
	var codeLines []string
	curr := start
	for curr < n {
		if strings.HasPrefix(strings.TrimSpace(lines[curr]), fence) {
			curr++
			break
		}
		codeLines = append(codeLines, lines[curr])
		curr++
	}
	codeContent := strings.Join(codeLines, "\n")
	if len(codeLines) > 0 {
		codeContent += "\n"
	}
	sb := GetBuffer()
	defer PutBuffer(sb)
	sb.WriteString("<pre><code")
	if lang != "" {
		sb.WriteString(` class="language-`)
		sb.WriteString(html.EscapeString(lang))
		sb.WriteString(`"`)
	}
	sb.WriteString(">")
	sb.WriteString(html.EscapeString(codeContent))
	sb.WriteString("</code></pre>\n")
	return sb.String(), curr - i
}

func parseATXHeading(trimmed string, ctx *parseContext) (string, bool) {
	m := atxHeadingRe.FindStringSubmatch(trimmed)
	if m == nil {
		return "", false
	}
	level := len(m[1])
	title := m[2]
	slug := ctx.slugs.Slug(title)
	sb := GetBuffer()
	defer PutBuffer(sb)
	fmt.Fprintf(sb, `<h%d id="%s">`, level, html.EscapeString(slug))
	renderInlines(title, sb)
	fmt.Fprintf(sb, "</h%d>\n", level)
	return sb.String(), true
}

func parseSetextHeading(lines []string, i int, ctx *parseContext) (string, int, bool) {
	n := len(lines)
	if i+1 >= n || strings.TrimSpace(lines[i+1]) == "" {
		return "", 0, false
	}
	trimmed := strings.TrimSpace(lines[i])
	nextTrimmed := strings.TrimSpace(lines[i+1])
	if !isSetextUnderline(nextTrimmed) || strings.HasPrefix(trimmed, ">") || strings.HasPrefix(trimmed, "-") {
		return "", 0, false
	}
	level := 1
	if nextTrimmed[0] == '-' {
		level = 2
	}
	title := trimmed
	slug := ctx.slugs.Slug(title)
	sb := GetBuffer()
	defer PutBuffer(sb)
	fmt.Fprintf(sb, `<h%d id="%s">`, level, html.EscapeString(slug))
	renderInlines(title, sb)
	fmt.Fprintf(sb, "</h%d>\n", level)
	return sb.String(), 2, true
}

func parseBlockquote(lines []string, i int) (string, int) {
	n := len(lines)
	var bqLines []string
	curr := i
	for curr < n {
		cur := strings.TrimSpace(lines[curr])
		if !strings.HasPrefix(cur, ">") {
			break
		}
		rest := strings.TrimPrefix(cur, ">")
		rest = strings.TrimPrefix(rest, " ")
		bqLines = append(bqLines, rest)
		curr++
	}
	inner := parseAndRenderBlocks([]byte(strings.Join(bqLines, "\n")))
	return "<blockquote>\n" + inner + "</blockquote>\n", curr - i
}

func parseHTMLBlock(lines []string, i int) (string, int) {
	n := len(lines)
	var htmlLines []string
	curr := i
	for curr < n {
		cur := lines[curr]
		if strings.TrimSpace(cur) == "" {
			break
		}
		htmlLines = append(htmlLines, cur)
		curr++
	}
	return strings.Join(htmlLines, "\n") + "\n", curr - i
}

func parseParagraph(lines []string, i int) (string, int) {
	n := len(lines)
	var pLines []string
	curr := i
	for curr < n {
		cur := lines[curr]
		curTrim := strings.TrimSpace(cur)
		if curTrim == "" {
			break
		}
		if len(pLines) > 0 {
			if atxHeadingRe.MatchString(curTrim) ||
				strings.HasPrefix(curTrim, "```") ||
				strings.HasPrefix(curTrim, "~~~") ||
				strings.HasPrefix(curTrim, ">") ||
				hrRe.MatchString(curTrim) ||
				isSetextUnderline(curTrim) {
				break
			}
		}
		pLines = append(pLines, cur)
		curr++
	}
	sb := GetBuffer()
	defer PutBuffer(sb)
	pText := strings.Join(pLines, "\n")
	sb.WriteString("<p>")
	renderInlines(pText, sb)
	sb.WriteString("</p>\n")
	return sb.String(), curr - i
}

func isSetextUnderline(s string) bool {
	if len(s) < 2 {
		return false
	}
	ch := s[0]
	if ch != '=' && ch != '-' {
		return false
	}
	for i := 1; i < len(s); i++ {
		if s[i] != ch {
			return false
		}
	}
	return true
}

func isTableSeparator(s string) bool {
	trimmed := strings.TrimSpace(s)
	if !strings.Contains(trimmed, "-") || !strings.Contains(trimmed, "|") {
		return false
	}
	cols := splitTableRow(trimmed)
	if len(cols) == 0 {
		return false
	}
	for _, col := range cols {
		clean := strings.Trim(col, " :")
		if clean == "" || strings.Trim(clean, "-") != "" {
			return false
		}
	}
	return true
}

func splitTableRow(s string) []string {
	trimmed := strings.TrimSpace(s)
	trimmed = strings.TrimPrefix(trimmed, "|")
	trimmed = strings.TrimSuffix(trimmed, "|")
	rawCols := strings.Split(trimmed, "|")
	var cols []string
	for _, c := range rawCols {
		cols = append(cols, strings.TrimSpace(c))
	}
	return cols
}

func parseTable(lines []string) (string, int) {
	if len(lines) < 2 {
		return "", 0
	}
	headerCols := splitTableRow(lines[0])
	sepCols := splitTableRow(lines[1])
	if len(headerCols) == 0 || len(sepCols) == 0 {
		return "", 0
	}

	// Determine alignment per column
	aligns := make([]string, len(headerCols))
	for j, col := range sepCols {
		if j >= len(aligns) {
			break
		}
		col = strings.TrimSpace(col)
		hasLeft := strings.HasPrefix(col, ":")
		hasRight := strings.HasSuffix(col, ":")
		if hasLeft && hasRight {
			aligns[j] = "center"
		} else if hasRight {
			aligns[j] = "right"
		} else if hasLeft {
			aligns[j] = "left"
		}
	}

	sb := GetBuffer()
	defer PutBuffer(sb)
	sb.WriteString("<table>\n<thead>\n<tr>\n")
	for j, h := range headerCols {
		sb.WriteString("<th")
		if j < len(aligns) && aligns[j] != "" {
			fmt.Fprintf(sb, ` align="%s"`, aligns[j])
		}
		sb.WriteString(">")
		renderInlines(h, sb)
		sb.WriteString("</th>\n")
	}
	sb.WriteString("</tr>\n</thead>\n")

	// Parse body rows
	var bodyRows [][]string
	i := 2
	for i < len(lines) {
		line := strings.TrimSpace(lines[i])
		if line == "" || !strings.Contains(line, "|") {
			break
		}
		bodyRows = append(bodyRows, splitTableRow(line))
		i++
	}

	if len(bodyRows) > 0 {
		sb.WriteString("<tbody>\n")
		for _, row := range bodyRows {
			sb.WriteString("<tr>\n")
			for j := 0; j < len(headerCols); j++ {
				val := ""
				if j < len(row) {
					val = row[j]
				}
				sb.WriteString("<td")
				if j < len(aligns) && aligns[j] != "" {
					fmt.Fprintf(sb, ` align="%s"`, aligns[j])
				}
				sb.WriteString(">")
				renderInlines(val, sb)
				sb.WriteString("</td>\n")
			}
			sb.WriteString("</tr>\n")
		}
		sb.WriteString("</tbody>\n")
	}
	sb.WriteString("</table>\n")

	return sb.String(), i
}

func parseList(lines []string, ordered bool) (string, int) {
	tag := "ul"
	if ordered {
		tag = "ol"
	}

	var items []string
	i := 0
	re := ulItemRe
	if ordered {
		re = olItemRe
	}

	for i < len(lines) {
		line := lines[i]
		m := re.FindStringSubmatch(line)
		if m == nil {
			break
		}
		items = append(items, m[2])
		i++
	}

	sb := GetBuffer()
	defer PutBuffer(sb)
	fmt.Fprintf(sb, "<%s>\n", tag)
	for _, item := range items {
		// Check for task list item: [ ] or [x]
		itemTrim := strings.TrimSpace(item)
		if strings.HasPrefix(itemTrim, "[ ] ") || strings.HasPrefix(itemTrim, "[x] ") || strings.HasPrefix(itemTrim, "[X] ") {
			checked := !strings.HasPrefix(itemTrim, "[ ] ")
			rest := itemTrim[4:]
			sb.WriteString("<li>")
			if checked {
				sb.WriteString(`<input checked="" disabled="" type="checkbox"> `)
			} else {
				sb.WriteString(`<input disabled="" type="checkbox"> `)
			}
			renderInlines(rest, sb)
			sb.WriteString("</li>\n")
		} else {
			sb.WriteString("<li>")
			renderInlines(item, sb)
			sb.WriteString("</li>\n")
		}
	}
	fmt.Fprintf(sb, "</%s>\n", tag)

	return sb.String(), i
}

func isHTMLBlockStart(s string) bool {
	if !strings.HasPrefix(s, "<") {
		return false
	}
	// Check for common HTML block tags
	lower := strings.ToLower(s)
	blockTags := []string{
		"<div", "<span", "<table", "<p", "<pre", "<script", "<iframe",
		"<header", "<footer", "<nav", "<section", "<article", "<aside",
		"<h1", "<h2", "<h3", "<h4", "<h5", "<h6", "<ul", "<ol", "<li",
		"<blockquote", "<form", "<style", "<button",
	}
	for _, tag := range blockTags {
		if strings.HasPrefix(lower, tag) {
			return true
		}
	}
	return false
}
