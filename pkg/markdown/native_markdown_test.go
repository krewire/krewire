package markdown

import (
	"strings"
	"sync"
	"testing"
)

func TestLexerAndLineScanner(t *testing.T) {
	src := []byte("Line 1\n\nLine 2\r\n\r\nLine 3")
	html, err := Render(src)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	if !strings.Contains(html, "<p>Line 1</p>") || !strings.Contains(html, "<p>Line 2</p>") || !strings.Contains(html, "<p>Line 3</p>") {
		t.Errorf("Line scanner did not correctly split paragraphs: %s", html)
	}
}

func TestBlockParsing_Complete(t *testing.T) {
	src := []byte(`# Heading 1
## Heading 2

Alt Heading 1
===

Alt Heading 2
---

> Quote line 1
> Quote line 2

` + "```go\nfunc main() {}\n```\n" + `
- Item A
- Item B
- [ ] Task incomplete
- [x] Task complete

1. Ordered A
2. Ordered B

| Col A | Col B |
| :--- | :---: |
| val1 | val2 |

---
`)

	html, err := Render(src)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}

	assertions := []string{
		`<h1 id="heading-1">Heading 1</h1>`,
		`<h2 id="heading-2">Heading 2</h2>`,
		`<h1 id="alt-heading-1">Alt Heading 1</h1>`,
		`<h2 id="alt-heading-2">Alt Heading 2</h2>`,
		"<blockquote>",
		`<pre><code class="language-go">func main() {}`,
		"<ul>",
		"<ol>",
		`<li>Item A</li>`,
		`<input disabled="" type="checkbox"> Task incomplete`,
		`<input checked="" disabled="" type="checkbox"> Task complete`,
		`<table>`,
		`<th align="left">Col A</th>`,
		`<th align="center">Col B</th>`,
		`<td align="left">val1</td>`,
		`<td align="center">val2</td>`,
		"<hr>",
	}

	for _, want := range assertions {
		if !strings.Contains(html, want) {
			t.Errorf("output missing expected block %q:\n%s", want, html)
		}
	}
}

func TestInlineParsing_Complete(t *testing.T) {
	src := []byte(`This has **bold**, *italic*, ***both***, ~~strike~~, ` + "`code span`" + `, [my link](https://krewire.com "Home"), ![alt img](/pic.png), and <https://krewire.com>.`)
	html, err := Render(src)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}

	assertions := []string{
		"<strong>bold</strong>",
		"<em>italic</em>",
		"<strong><em>both</em></strong>",
		"<del>strike</del>",
		"<code>code span</code>",
		`<a href="https://krewire.com" title="Home">my link</a>`,
		`<img src="/pic.png" alt="alt img">`,
		`<a href="https://krewire.com">https://krewire.com</a>`,
	}

	for _, want := range assertions {
		if !strings.Contains(html, want) {
			t.Errorf("output missing expected inline %q:\n%s", want, html)
		}
	}
}

func TestAutoHeadingIDs_Deduplication(t *testing.T) {
	src := []byte("# Same Title\n\n# Same Title\n\n# Same Title\n")
	html, err := Render(src)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}

	if !strings.Contains(html, `id="same-title"`) {
		t.Errorf("missing first slug: %s", html)
	}
	if !strings.Contains(html, `id="same-title-1"`) {
		t.Errorf("missing deduplicated slug -1: %s", html)
	}
	if !strings.Contains(html, `id="same-title-2"`) {
		t.Errorf("missing deduplicated slug -2: %s", html)
	}
}

func TestHTMLRendering_RenderAndRenderSafe(t *testing.T) {
	trusted := []byte("<div class=\"hero\"><h1>Safe</h1></div>")
	untrusted := []byte("<script>steal()</script><p>Text</p>")

	outTrusted, err := Render(trusted)
	if err != nil {
		t.Fatalf("Render trusted: %v", err)
	}
	if !strings.Contains(outTrusted, `<div class="hero"><h1>Safe</h1></div>`) {
		t.Errorf("Render should preserve trusted HTML: %s", outTrusted)
	}

	outSafe, err := RenderSafe(untrusted)
	if err != nil {
		t.Fatalf("RenderSafe untrusted: %v", err)
	}
	if strings.Contains(outSafe, "<script>") || strings.Contains(outSafe, "steal()") {
		t.Errorf("RenderSafe must strip script tags: %s", outSafe)
	}
	if !strings.Contains(outSafe, "<p>Text</p>") {
		t.Errorf("RenderSafe should keep safe paragraph: %s", outSafe)
	}
}

func TestSecurityControls_ScriptURLs(t *testing.T) {
	cases := []string{
		`[click](javascript:alert(1))`,
		`[click](vbscript:alert(1))`,
		`[click](data:text/html;base64,PHNjcmlwdD5hbGVydCgxKTwvc2NyaXB0Pg==)`,
	}
	for _, src := range cases {
		out, err := Render([]byte(src))
		if err != nil {
			t.Fatalf("Render: %v", err)
		}
		if strings.Contains(out, "javascript:") || strings.Contains(out, "vbscript:") || strings.Contains(out, "data:") {
			t.Errorf("Script URL was not stripped from %q:\n%s", src, out)
		}
	}
}

func TestAPIDropInCompatibility(t *testing.T) {
	// Assert signatures
	var _ func([]byte) (string, error) = Render
	var _ func([]byte) (string, error) = RenderSafe
	var _ func([]byte, string) (string, error) = RenderWithBase
	var _ func([]byte, string) (string, error) = RenderSafeWithBase
	var _ func(string, string) string = PrefixLinks
	var _ func(string) (string, error) = StripScriptURLs
}

func TestConcurrentSafe(t *testing.T) {
	var wg sync.WaitGroup
	src := []byte("# Heading\n\nParagraph with **bold** and `code`.\n\n| A | B |\n|---|---|\n| 1 | 2 |\n")

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			out, err := Render(src)
			if err != nil || !strings.Contains(out, "<h1") {
				t.Errorf("Concurrent render failed: %v", err)
			}
		}()
	}
	wg.Wait()
}
