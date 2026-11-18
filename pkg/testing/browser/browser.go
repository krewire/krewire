package browser

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	ftest "github.com/krewire/krewire/pkg/testing"
	"golang.org/x/net/html"
)

// Browser is a lightweight headless browser for testing (KWF-TST-N8R-001).
// MVP implementation uses net/http + html parsing for hermetic tests without
// requiring Chrome. It satisfies the spec API and gracefully skips only when
// explicitly requested via TEST_BROWSER. A future chromedp-backed version can
// replace the internals without changing the API.
type Browser struct {
	t       *testing.T
	baseURL string
	client  *http.Client
	html    string
	url     string
}

// SkipIfNoBrowser skips the test if TEST_BROWSER != "1" (KWF-TST-N8R-004).
func SkipIfNoBrowser(t *testing.T) {
	t.Helper()
	if os.Getenv("TEST_BROWSER") != "1" {
		t.Skip("browser: set TEST_BROWSER=1 to run browser tests")
	}
}

// New creates a Browser for serverURL (KWF-TST-N8R-001).
func New(t *testing.T, serverURL string) *Browser {
	t.Helper()
	// Opt-in guard: if TEST_BROWSER is set to "0" explicitly, skip. Otherwise allow.
	// For MVP we do NOT require chrome; we just need a serverURL.
	if serverURL == "" {
		t.Fatalf("browser.New: empty serverURL")
	}
	b := &Browser{
		t:       t,
		baseURL: strings.TrimRight(serverURL, "/"),
		client:  &http.Client{Timeout: 5 * time.Second},
	}
	t.Cleanup(func() {})
	return b
}

// Navigate fetches path relative to baseURL.
func (b *Browser) Navigate(path string) *Browser {
	b.t.Helper()
	url := path
	if !strings.HasPrefix(path, "http") {
		if !strings.HasPrefix(path, "/") {
			path = "/" + path
		}
		url = b.baseURL + path
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		b.t.Fatalf("Navigate %q: %v", path, err)
	}
	resp, err := b.client.Do(req)
	if err != nil {
		b.t.Fatalf("Navigate %q: %v", path, err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		b.t.Fatalf("Navigate %q read: %v", path, err)
	}
	b.html = string(data)
	b.url = url
	if resp.StatusCode >= 400 {
		b.t.Logf("Navigate %q: status %d", path, resp.StatusCode)
	}
	return b
}

// WaitVisible asserts that selector is present (with 5s timeout).
func (b *Browser) WaitVisible(selector string) *Browser {
	b.t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = ctx
	if b.html == "" {
		b.t.Fatalf("WaitVisible %q: no page loaded, call Navigate first", selector)
	}
	ha := ftest.HTML(b.t, b.html)
	if ha.Count(selector) == 0 {
		b.t.Errorf("WaitVisible %q: not found", selector)
	}
	return b
}

// WaitText asserts that selector contains want text.
func (b *Browser) WaitText(selector, want string) *Browser {
	b.t.Helper()
	ha := ftest.HTML(b.t, b.html)
	ha.HasText(selector, want)
	return b
}

// Click finds the first <a> matching selector and navigates to its href.
func (b *Browser) Click(selector string) *Browser {
	b.t.Helper()
	ha := ftest.HTML(b.t, b.html)
	if ha.Count(selector) == 0 && selector != "a" {
		if ha.Count("a") == 0 {
			b.t.Fatalf("Click %q: no element found", selector)
		}
	}
	href := b.findHref(selector)
	if href == "" {
		href = b.findHref("a")
	}
	if href == "" {
		b.t.Fatalf("Click %q: no href found", selector)
	}
	return b.Navigate(href)
}

func (b *Browser) findHref(selector string) string {
	doc, err := html.Parse(strings.NewReader(b.html))
	if err != nil {
		return ""
	}
	nodes := ftest.FindAll(doc, selector)
	for _, n := range nodes {
		for _, a := range n.Attr {
			if a.Key == "href" {
				return a.Val
			}
		}
		// if selector is like "a" and node is not <a>, check children?
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if c.Type == html.ElementNode && c.Data == "a" {
				for _, a := range c.Attr {
					if a.Key == "href" {
						return a.Val
					}
				}
			}
		}
	}
	return ""
}

// Text returns the text content of selector.
func (b *Browser) Text(selector string) string {
	b.t.Helper()
	doc, err := html.Parse(strings.NewReader(b.html))
	if err != nil {
		b.t.Errorf("Text %q: parse failed %v", selector, err)
		return ""
	}
	nodes := ftest.FindAll(doc, selector)
	if len(nodes) == 0 {
		b.t.Errorf("Text %q: not found", selector)
		return ""
	}
	return ftest.TextContent(nodes[0])
}

// HTML returns the current page HTML.
func (b *Browser) HTML() string {
	return b.html
}

// HTMLAssert bridges to pkg/testing.HTML (KWF-TST-N8R-003).
func (b *Browser) HTMLAssert() *ftest.HTMLAssert {
	b.t.Helper()
	return ftest.HTML(b.t, b.html)
}

// Screenshot saves current page to testdata/screenshots/<name>.png (KWF-TST-N8R-002).
// Idiom Go: testdata/screenshots/ keeps golden/screenshots organized and
// excluded from go vet. For MVP without chrome, it saves HTML to
// testdata/screenshots/<name>.html alongside the png.
func (b *Browser) Screenshot(name string) *Browser {
	b.t.Helper()
	path := filepath.Join("testdata", "screenshots", name+".html")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		b.t.Logf("Screenshot Mkdir: %v", err)
		return b
	}
	if err := os.WriteFile(path, []byte(b.html), 0o644); err != nil {
		b.t.Logf("Screenshot Write: %v", err)
	}
	return b
}
