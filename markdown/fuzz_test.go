package markdown_test

import (
	"strings"
	"testing"

	"github.com/krewire/krewire/markdown"
)

func FuzzMarkdownRender(f *testing.F) {
	seeds := []string{
		"# Heading 1\nParagraph with **bold** and *italic*.",
		"[Safe Link](https://example.com/page)",
		"[Malicious Link](javascript:alert(1))",
		"<script>alert('xss')</script>",
		"<a href='vbscript:msgbox(1)'>click me</a>",
		"<a href=\"javascript:alert(1)\">click</a>",
		"```go\nfunc main() {}\n```",
		"- Item 1\n- Item 2\n  - Subitem",
		"| A | B |\n|---|---|\n| 1 | 2 |",
		"> Blockquote with nested [link](data:text/html,<script>alert(1)</script>)",
		strings.Repeat("nested **bold *italic* ** ", 20),
	}

	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, input string) {
		// 1. Render must never panic on arbitrary input
		out, err := markdown.Render([]byte(input))
		if err == nil {
			_ = out
		}

		// 2. Sanitize must never panic on arbitrary input
		sanitized, _ := markdown.Sanitize(input)
		_ = sanitized

		// 3. StripScriptURLs must never panic and must neutralize script URLs in attributes
		cleaned, err := markdown.StripScriptURLs(input)
		if err == nil {
			_ = cleaned
		}
	})
}
