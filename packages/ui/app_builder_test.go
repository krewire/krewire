package ui

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/krewire/krewire/packages/ui/form"
	"github.com/krewire/krewire/packages/ui/panel"
	"github.com/krewire/krewire/packages/ui/widget"
)

func TestAppPageRouting(t *testing.T) {
	app := New("Forge Admin").
		AddNav("Dashboard", "/").
		AddNav("Settings", "/settings").
		Page("/", "Dashboard", widget.NewCard("Welcome to Admin")).
		Page("/settings", "Settings", widget.NewAlert("System settings").Success())

	// Test GET /
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Dashboard — Forge Admin") {
		t.Errorf("expected title, got %s", body)
	}
	if !strings.Contains(body, "Welcome to Admin") {
		t.Errorf("expected card content, got %s", body)
	}
	if !strings.Contains(body, "Settings") {
		t.Errorf("expected nav link, got %s", body)
	}

	// Test 404
	req404 := httptest.NewRequest(http.MethodGet, "/unknown", nil)
	rec404 := httptest.NewRecorder()
	app.ServeHTTP(rec404, req404)
	if rec404.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec404.Code)
	}
}

func TestAppFormRouting(t *testing.T) {
	contactForm := form.New("/contact", "POST").
		Add(form.NewText("name", "Your Name").MakeRequired()).
		Add(form.NewText("message", "Message").MakeRequired())

	submitted := false
	app := New("Portal").
		Form("/contact", contactForm, func(vals map[string]string) (widget.Widget, error) {
			submitted = true
			return widget.NewAlert("Thank you, " + vals["name"]).Success(), nil
		})

	// GET form
	reqGet := httptest.NewRequest(http.MethodGet, "/contact", nil)
	recGet := httptest.NewRecorder()
	app.ServeHTTP(recGet, reqGet)
	if recGet.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", recGet.Code)
	}
	if !strings.Contains(recGet.Body.String(), "Your Name") {
		t.Errorf("expected form markup, got %s", recGet.Body.String())
	}

	// POST form valid
	formData := url.Values{
		"name":    {"Budi"},
		"message": {"Hello Forge!"},
	}
	reqPost := httptest.NewRequest(http.MethodPost, "/contact", strings.NewReader(formData.Encode()))
	reqPost.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recPost := httptest.NewRecorder()
	app.ServeHTTP(recPost, reqPost)

	if !submitted {
		t.Error("expected onSubmit handler to have been invoked")
	}
	if !strings.Contains(recPost.Body.String(), "Thank you, Budi") {
		t.Errorf("expected success alert, got %s", recPost.Body.String())
	}
}

func TestAppDashboard(t *testing.T) {
	dash := panel.NewDashboard("Overview").
		AddStat(panel.NewStat("Users", "250")).
		AddPanel(panel.New("System Health").WithBody(widget.NewBadge("Normal").Success()))

	app := New("Monitor").Dashboard("/overview", dash)

	req := httptest.NewRequest(http.MethodGet, "/overview", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Overview — Monitor") {
		t.Errorf("missing title in %s", body)
	}
	if !strings.Contains(body, "Users") || !strings.Contains(body, "250") {
		t.Errorf("missing stat in %s", body)
	}
}

func TestAppTailwindDefaultAndOverride(t *testing.T) {
	appDefault := New("Default App")
	shellDefault := string(appDefault.RenderShell("Home", ""))
	if !strings.Contains(shellDefault, "https://cdn.tailwindcss.com") {
		t.Errorf("expected default Tailwind CDN injection, got %s", shellDefault)
	}
	if !strings.Contains(shellDefault, "tailwind.config") {
		t.Errorf("expected default tailwind config, got %s", shellDefault)
	}

	appCustom := New("Custom App").WithTailwindURL("/custom/tailwind.css")
	shellCustom := string(appCustom.RenderShell("Home", ""))
	if !strings.Contains(shellCustom, `<link rel="stylesheet" href="/custom/tailwind.css">`) {
		t.Errorf("expected custom stylesheet link, got %s", shellCustom)
	}

	appDisabled := New("No Tailwind").DisableTailwind()
	shellDisabled := string(appDisabled.RenderShell("Home", ""))
	if strings.Contains(shellDisabled, "tailwindcss") {
		t.Errorf("expected no Tailwind when disabled, got %s", shellDisabled)
	}
}

// TestRenderShell_EmitsRealNewlines guards the document shell against literal
// "\n" text leaking into the HTML. Backtick constants that contain an escape
// sequence would emit the two characters backslash-n rather than a line break,
// which is easy to reintroduce when the markup fragments are extracted into
// named constants.
func TestRenderShell_EmitsRealNewlines(t *testing.T) {
	app := New("Ops").AddNav("Home", "/")
	shell := string(app.RenderShell("Dash", widget.NewCard("x").Render()))

	if strings.Contains(shell, `\n`) {
		t.Error("shell must not contain a literal backslash-n sequence")
	}
	// Each structural break that follows a closing tag must be a real newline.
	for _, want := range []string{
		"<!doctype html>\n",
		"\n<head>\n",
		"</head>\n",
		"</header>\n",
		"\n</main>\n",
		"</body>\n</html>",
	} {
		if !strings.Contains(shell, want) {
			t.Errorf("shell missing real newline sequence %q", want)
		}
	}
}

// TestRenderShell_TitleQualification verifies the page title composes with the
// app title and falls back to the app title alone.
func TestRenderShell_TitleQualification(t *testing.T) {
	app := New("Ops")
	if got := app.documentTitle("Dash"); got != "Dash — Ops" {
		t.Errorf("documentTitle(Dash) = %q", got)
	}
	if got := app.documentTitle(""); got != "Ops" {
		t.Errorf("documentTitle(empty) = %q", got)
	}
	// The title is escaped, so markup in a page name cannot break out of <title>.
	esc := string(app.RenderShell(`<script>x</script>`, widget.NewCard("y").Render()))
	if strings.Contains(esc, "<script>x</script>") {
		t.Error("page title must be HTML-escaped so it cannot break out of <title>")
	}
	if !strings.Contains(esc, "&lt;script&gt;") {
		t.Error("page title should appear entity-encoded in <title>")
	}
}

// TestRenderShell_TailwindModes verifies the stylesheet-versus-script decision
// and that disabling Tailwind emits no entry point at all.
func TestRenderShell_TailwindModes(t *testing.T) {
	cdn := string(New("A").RenderShell("P", widget.NewCard("c").Render()))
	if !strings.Contains(cdn, `<script src="`+DefaultTailwindCDN+`">`) {
		t.Error("CDN mode must emit a script tag pointing at the CDN")
	}
	if !strings.Contains(cdn, "tailwind.config") {
		t.Error("CDN mode must emit the Krewire token configuration")
	}

	css := string(New("A").WithTailwindURL("/assets/tailwind.css").RenderShell("P", widget.NewCard("c").Render()))
	if !strings.Contains(css, `<link rel="stylesheet" href="/assets/tailwind.css">`) {
		t.Error("css mode must emit a stylesheet link")
	}
	if strings.Contains(css, "tailwind.config") {
		t.Error("css mode must not emit the CDN configuration script")
	}

	off := string(New("A").DisableTailwind().RenderShell("P", widget.NewCard("c").Render()))
	if strings.Contains(off, "tailwind") {
		t.Error("DisableTailwind must emit no Tailwind markup")
	}
}

// TestRenderShell_NavURLSafety verifies navigation targets are filtered, so a
// javascript: URL supplied through the builder cannot reach the href.
func TestRenderShell_NavURLSafety(t *testing.T) {
	shell := string(New("A").AddNav("Bad", "javascript:alert(1)").RenderShell("P", widget.NewCard("c").Render()))
	if strings.Contains(shell, "javascript:") {
		t.Error("navigation must not render a javascript: URL")
	}
}
