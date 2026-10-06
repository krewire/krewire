// Tests for KWL-Q3N8P (KWL-MD-001..003) — Scope: Unit libs/markdown.
package markdown

import (
	"strings"
	"testing"
)

// TestSEC_MD_001_RenderSafe_RemovesScriptBearing verifies that RenderSafe drops
// executable markup. Render keeps raw HTML for trusted .kiw sources, so any
// Markdown supplied by an untrusted author must go through RenderSafe.
func TestSEC_MD_001_RenderSafe_RemovesScriptBearing(t *testing.T) {
	cases := []struct {
		name    string
		src     string
		absent  []string
		present []string
	}{
		{
			name:   "script element and body removed",
			src:    `<script>alert(document.cookie)</script>`,
			absent: []string{"<script", "alert(document.cookie)"},
		},
		{
			name:    "event handler attribute stripped",
			src:     `<img src=x onerror=alert(1)>`,
			absent:  []string{"onerror", "alert(1)"},
			present: []string{"<img"},
		},
		{
			name:   "iframe removed",
			src:    `<iframe src="https://evil.example"></iframe>`,
			absent: []string{"<iframe"},
		},
		{
			name:    "javascript href dropped",
			src:     `<a href="javascript:alert(1)">click</a>`,
			absent:  []string{"javascript:"},
			present: []string{"click"},
		},
		{
			name:   "mixed-case javascript scheme dropped",
			src:    `<a href="JaVaScRiPt:alert(1)">x</a>`,
			absent: []string{"JaVaScRiPt:", "alert(1)"},
		},
		{
			name:   "control-character obfuscated scheme dropped",
			src:    "<a href=\"java\tscript:alert(1)\">x</a>",
			absent: []string{"alert(1)"},
		},
		{
			name:   "vbscript href dropped",
			src:    `<a href="vbscript:msgbox(1)">x</a>`,
			absent: []string{"vbscript:"},
		},
		{
			name:    "style attribute stripped",
			src:     `<div style="background:url(javascript:alert(1))">d</div>`,
			absent:  []string{"style=", "javascript:"},
			present: []string{"<div"},
		},
		{
			name:   "form removed entirely",
			src:    `<form action="javascript:alert(1)"><input name=a></form>`,
			absent: []string{"<form", "<input"},
		},
		{
			// Goldmark treats "<svg/onload=...>" as literal text and escapes it,
			// so the payload survives only as inert characters, never as a tag.
			name:   "svg onload never becomes live markup",
			src:    `<svg/onload=alert(1)>`,
			absent: []string{"<svg"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out, err := RenderSafe([]byte(tc.src))
			if err != nil {
				t.Fatalf("RenderSafe: %v", err)
			}
			for _, bad := range tc.absent {
				if strings.Contains(out, bad) {
					t.Errorf("output must not contain %q:\n%s", bad, out)
				}
			}
			for _, good := range tc.present {
				if !strings.Contains(out, good) {
					t.Errorf("output should still contain %q:\n%s", good, out)
				}
			}
		})
	}
}

// TestSEC_MD_002_RenderSafe_KeepsSafeMarkup guards against over-sanitizing:
// ordinary documentation markup must survive intact.
func TestSEC_MD_002_RenderSafe_KeepsSafeMarkup(t *testing.T) {
	src := []byte("# Title\n\nA **bold** and [link](https://ok.example) and `code`.\n\n| a | b |\n|---|---|\n| 1 | 2 |\n")
	out, err := RenderSafe(src)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`<h1 id="title">Title</h1>`,
		"<strong>bold</strong>",
		`href="https://ok.example"`,
		"<code>code</code>",
		"<table>",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("RenderSafe dropped expected markup %q:\n%s", want, out)
		}
	}
}

// TestSEC_MD_003_Render_StripsScriptURLs verifies that even the trusted Render
// path neutralizes javascript: links, which the ecosystem needs because Markdown
// authors link with [text](javascript:...) by mistake or maliciously.
func TestSEC_MD_003_Render_StripsScriptURLs(t *testing.T) {
	cases := []struct {
		src     string
		absent  string
		present string
	}{
		{`<a href="javascript:alert(1)">x</a>`, "javascript:", "x"},
		{`<a href="https://ok.example">y</a>`, "", `href="https://ok.example"`},
		{`<a href="/relative">y</a>`, "", `href="/relative"`},
		{`<a href="#anchor">y</a>`, "", `href="#anchor"`},
		{`<a href="mailto:a@b.example">y</a>`, "", `mailto:a@b.example`},
		{`<a href="//cdn.example/x">y</a>`, "", `//cdn.example/x`},
	}
	for _, tc := range cases {
		out, err := Render([]byte(tc.src))
		if err != nil {
			t.Fatalf("Render(%q): %v", tc.src, err)
		}
		if tc.absent != "" && strings.Contains(out, tc.absent) {
			t.Errorf("Render(%q) still contains %q:\n%s", tc.src, tc.absent, out)
		}
		if tc.present != "" && !strings.Contains(out, tc.present) {
			t.Errorf("Render(%q) should preserve %q:\n%s", tc.src, tc.present, out)
		}
	}
}

// TestSEC_MD_004_RenderSafeWithBase verifies sanitization composes with the
// base-prefix rewriting used by the book and ssg build.
func TestSEC_MD_004_RenderSafeWithBase(t *testing.T) {
	// The blank line is required: without it Goldmark treats the script tag as
	// an HTML block that swallows the link on the following line.
	out, err := RenderSafeWithBase([]byte("<script>bad()</script>\n\n[x](/y)"), "/guide/")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "script") || strings.Contains(out, "bad()") {
		t.Errorf("RenderSafeWithBase must sanitize:\n%s", out)
	}
	if !strings.Contains(out, `href="/guide/y"`) {
		t.Errorf("RenderSafeWithBase must prefix links:\n%s", out)
	}
}

// TestSEC_MD_005_LinksGetNoopener documents that rendered anchors carry
// rel="noopener" so a target="_blank" link cannot reach back through
// window.opener.
func TestSEC_MD_005_LinksGetNoopener(t *testing.T) {
	out, err := RenderSafe([]byte(`<a href="https://ok.example" target="_blank">x</a>`))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "noopener") {
		t.Errorf("anchor must carry rel=noopener:\n%s", out)
	}
}

func TestKWL_MD_001_Render_GFMAndHeadingIDs(t *testing.T) {
	src := []byte("# Hello\n\nA **bold** and ~~strike~~.\n\n| a | b |\n|---|---|\n| 1 | 2 |\n")
	html, err := Render(src)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`<h1 id="hello">Hello</h1>`,
		`<strong>bold</strong>`,
		`<del>strike</del>`,
		"<table>",
	} {
		if !strings.Contains(html, want) {
			t.Errorf("Render output missing %q:\n%s", want, html)
		}
	}
}

func TestKWL_MD_002_RenderWithBase_PrefixesLinks(t *testing.T) {
	src := []byte("[next](/two) and ![alt](/img.png)")
	html, err := RenderWithBase(src, "/guide/")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(html, `href="/guide/two"`) || !strings.Contains(html, `src="/guide/img.png"`) {
		t.Errorf("RenderWithBase = %s", html)
	}
	root, err := RenderWithBase([]byte("[x](/y)"), "")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(root, `href="/y"`) {
		t.Errorf("empty base must not rewrite: %s", root)
	}
}

func TestKWL_MD_003_PrefixLinks(t *testing.T) {
	tt := []struct{ in, base, want string }{
		{`<a href="/getting-started">`, "/guide/", `<a href="/guide/getting-started">`},
		{`<a href="/">`, "/guide/", `<a href="/guide/">`},
		{`<a href="/guide/y">`, "/guide/", `<a href="/guide/y">`},
		{`<img src="/img.png">`, "/guide/", `<img src="/guide/img.png">`},
		{`<a href="/x">`, "/", `<a href="/x">`},
		{`<a href="//example.com">`, "/guide/", `<a href="//example.com">`},
	}
	for _, tc := range tt {
		if got := PrefixLinks(tc.in, tc.base); got != tc.want {
			t.Errorf("PrefixLinks(%q,%q) = %q, want %q", tc.in, tc.base, got, tc.want)
		}
	}
}

func TestKWL_MD_003_RenderDeterministic(t *testing.T) {
	src := []byte("# T\n\nBody.\n")
	a, _ := Render(src)
	b, _ := Render(src)
	if a != b {
		t.Error("identical input must render identically")
	}
}

func TestPrefixLinks_PreservesCodeBlocks(t *testing.T) {
	in := `<p><a href="/real-link">Link</a></p><pre><code><a href="/code-link">code</a></code></pre>`
	got := PrefixLinks(in, "/base/")
	if !strings.Contains(got, `href="/base/real-link"`) {
		t.Errorf("real link should be prefixed: %s", got)
	}
	if !strings.Contains(got, `href="/code-link"`) {
		t.Errorf("code link must be preserved: %s", got)
	}
}

func TestRender_PreservesRawHTML(t *testing.T) {
	src := []byte("<div class=\"custom-box\"><span class=\"badge\">1.1</span></div>")
	html, err := Render(src)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(html, `<div class="custom-box">`) || !strings.Contains(html, `<span class="badge">1.1</span>`) {
		t.Errorf("expected raw HTML to be preserved, got: %s", html)
	}
}
