package widget

import (
	"strings"
	"testing"
)

// TestSEC_WIDGET_001_SafeURL_BlocksExecutableSchemes verifies that a link target
// cannot carry a script-executing scheme. HTML-escaping alone does not help
// here: "javascript:alert(1)" contains no character that needs escaping, so it
// survives attribute encoding and still runs when the link is clicked.
func TestSEC_WIDGET_001_SafeURL_BlocksExecutableSchemes(t *testing.T) {
	unsafe := []string{
		"javascript:alert(1)",
		"JaVaScRiPt:alert(1)",
		"  javascript:alert(1)",
		"java\tscript:alert(1)",
		"vbscript:msgbox(1)",
		"data:text/html,<script>alert(1)</script>",
		"file:///etc/passwd",
	}
	for _, raw := range unsafe {
		if got := SafeURL(raw); got != "#" {
			t.Errorf("SafeURL(%q) = %q, want %q", raw, got, "#")
		}
	}
}

// TestSEC_WIDGET_002_SafeURL_AllowsLegitimateSchemes guards against the
// validator breaking ordinary navigation.
func TestSEC_WIDGET_002_SafeURL_AllowsLegitimateSchemes(t *testing.T) {
	cases := map[string]string{
		"/dashboard":     "/dashboard",
		"../relative":    "../relative",
		"#section":       "#section",
		"?q=1":           "?q=1",
		"https://ok.ex":  "https://ok.ex",
		"http://ok.ex":   "http://ok.ex",
		"mailto:a@b.ex":  "mailto:a@b.ex",
		"//cdn.ex/x.png": "//cdn.ex/x.png",
	}
	for raw, want := range cases {
		if got := SafeURL(raw); got != want {
			t.Errorf("SafeURL(%q) = %q, want %q", raw, got, want)
		}
	}
}

// TestSEC_WIDGET_003_SafeURL_EscapesQuoteInjection verifies that a URL
// containing a double quote cannot break out of the href attribute.
func TestSEC_WIDGET_003_SafeURL_EscapesQuoteInjection(t *testing.T) {
	got := SafeURL(`https://ok.example/" onmouseover="alert(1)`)
	if strings.Contains(got, `" onmouseover=`) {
		t.Errorf("SafeURL must escape quotes, got %q", got)
	}
	if !strings.Contains(got, "&#34;") {
		t.Errorf("SafeURL must entity-encode quotes, got %q", got)
	}
}

// TestSEC_WIDGET_004_ButtonLinkUsesSafeURL verifies the button link path routes
// through SafeURL rather than raw escaping.
func TestSEC_WIDGET_004_ButtonLinkUsesSafeURL(t *testing.T) {
	out := string(NewButton("click").Link("javascript:alert(1)").Render())
	if strings.Contains(out, "javascript:") {
		t.Errorf("button link must not carry a javascript: URL:\n%s", out)
	}
}
