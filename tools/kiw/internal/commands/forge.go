package commands

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unicode"

	"github.com/krewire/krewire/pkg/errs"
)

// RegisterForge registers flags for the forge command.
func RegisterForge(fs *flag.FlagSet) {
	fs.String("model", "", "resource: model name to bind to (e.g. User, Product)")
	fs.String("fields", "", "form: comma-separated field definitions (e.g. title:string,price:float,status:select)")
	fs.Int("port", 8089, "studio: HTTP bind port (default: 8089)")
	fs.Bool("dry-run", false, "preview generated files without writing to disk")
}

const forgeUsage = `Krewire Forge — Declarative Web, Form, and Admin Panel Builder for Go & .kiw

Usage:
  kiw forge panel <name>             Scaffold a new admin panel dashboard
  kiw forge resource <name>          Scaffold a CRUD resource (model, table view, and forms)
  kiw forge form <name>              Generate a declarative .kiw form with validation struct
  kiw forge studio [--port=8089]     Launch the local interactive visual builder studio

Examples:
  kiw forge panel admin
  kiw forge resource product --model=Product
  kiw forge form contact --fields="name:string,email:string,message:textarea"
  kiw forge studio --port=8089
`

// RunForge dispatches Krewire Forge scaffolding and studio subcommands.
func RunForge(fs *flag.FlagSet) errs.ExitCode {
	args := fs.Args()
	if len(args) == 0 {
		fmt.Print(forgeUsage)
		return errs.ExitCodeUsage
	}

	sub := strings.ToLower(args[0])
	dryRun := flagBool(fs, "dry-run")
	model := flagValue(fs, "model")
	fields := flagValue(fs, "fields")
	port := flagInt(fs, "port")
	if port <= 0 {
		port = 8089
	}

	root, err := findRoot()
	if err != nil {
		root, _ = os.Getwd()
	}

	switch sub {
	case "panel":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "error: panel name required (e.g. kiw forge panel admin)")
			return errs.ExitCodeUsage
		}
		if err := forgePanel(root, args[1], dryRun); err != nil {
			fmt.Fprintf(os.Stderr, "kiw forge panel: %v\n", err)
			return errs.ExitCodeFailure
		}
		return errs.ExitCodeSuccess

	case "resource":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "error: resource name required (e.g. kiw forge resource product)")
			return errs.ExitCodeUsage
		}
		if err := forgeResource(root, args[1], model, dryRun); err != nil {
			fmt.Fprintf(os.Stderr, "kiw forge resource: %v\n", err)
			return errs.ExitCodeFailure
		}
		return errs.ExitCodeSuccess

	case "form":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "error: form name required (e.g. kiw forge form contact)")
			return errs.ExitCodeUsage
		}
		if err := forgeForm(root, args[1], fields, dryRun); err != nil {
			fmt.Fprintf(os.Stderr, "kiw forge form: %v\n", err)
			return errs.ExitCodeFailure
		}
		return errs.ExitCodeSuccess

	case "studio":
		if err := forgeStudio(port); err != nil {
			fmt.Fprintf(os.Stderr, "kiw forge studio: %v\n", err)
			return errs.ExitCodeFailure
		}
		return errs.ExitCodeSuccess

	case "help", "-h", "--help":
		fmt.Print(forgeUsage)
		return errs.ExitCodeSuccess

	default:
		fmt.Fprintf(os.Stderr, "kiw forge: unknown subcommand %q\n\n%s", args[0], forgeUsage)
		return errs.ExitCodeUsage
	}
}

// ── 1. Panel Generator ────────────────────────────────────────────────────────

func forgePanel(root, name string, dryRun bool) error {
	slug := forgeToSlug(name)
	title := forgeToTitle(name)
	pascal := forgeToPascal(name)

	panelDir := filepath.Join(root, "panels", slug)
	goFile := filepath.Join(panelDir, "panel.go")
	kiwFile := filepath.Join(panelDir, "Dashboard.kiw")

	goContent := fmt.Sprintf(`package %s

import (
	"net/http"

	"github.com/krewire/krewire/pkg/ui/panel"
)

// Panel represents the %s administrative dashboard.
type Panel struct {
	Title string
	Slug  string
}

// New creates and initializes the %s panel.
func New() *Panel {
	return &Panel{
		Title: "%s Panel",
		Slug:  "%s",
	}
}

// Routes mounts the panel HTTP handlers onto the given mux.
func (p *Panel) Routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /%s", p.handleDashboard)
}

func (p *Panel) handleDashboard(w http.ResponseWriter, r *http.Request) {
	pnl := panel.New(p.Title).
		WithDesc("Central administrative console for managing resources and operational telemetry.")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(pnl.Render()))
}
`, slug, title, pascal, title, slug, slug)

	kiwContent := fmt.Sprintf(`<script>
  export let title = "%s Panel"
  export let metrics = {
    users: "1,248",
    activeWorkspaces: "42",
    uptime: "99.98%%"
  }
</script>

<div class="panel-layout min-h-screen bg-bg dark:bg-bg-dark text-fg dark:text-fg-dark">
  <!-- Top Navigation Header -->
  <header class="h-16 px-6 border-b border-border flex items-center justify-between bg-bg-2 dark:bg-bg-dark-2">
    <div class="flex items-center gap-3">
      <span class="icon-box bg-primary text-primary-content font-black w-8 h-8 flex items-center justify-center rounded">◈</span>
      <h1 class="text-base font-black tracking-tight">{title}</h1>
    </div>
    <div class="flex items-center gap-3">
      <Badge Variant="primary" Size="sm" Class="font-mono">v0.2.0</Badge>
    </div>
  </header>

  <!-- Main Content Area -->
  <main class="p-6 max-w-7xl mx-auto space-y-6">
    <!-- Stat Metrics Grid -->
    <div class="grid grid-cols-1 sm:grid-cols-3 gap-4">
      <div class="p-5 rounded-xl border border-border bg-bg-2 dark:bg-bg-dark-2 shadow-sm">
        <span class="text-2xs font-mono font-bold uppercase text-muted dark:text-muted-dark">Total Users</span>
        <div class="text-2xl font-black mt-1 text-primary">{metrics.users}</div>
      </div>
      <div class="p-5 rounded-xl border border-border bg-bg-2 dark:bg-bg-dark-2 shadow-sm">
        <span class="text-2xs font-mono font-bold uppercase text-muted dark:text-muted-dark">Active Workspaces</span>
        <div class="text-2xl font-black mt-1 text-accent">{metrics.activeWorkspaces}</div>
      </div>
      <div class="p-5 rounded-xl border border-border bg-bg-2 dark:bg-bg-dark-2 shadow-sm">
        <span class="text-2xs font-mono font-bold uppercase text-muted dark:text-muted-dark">System SLA</span>
        <div class="text-2xl font-black mt-1 text-emerald-500">{metrics.uptime}</div>
      </div>
    </div>

    <!-- Panel Actions & Resources -->
    <div class="p-6 rounded-xl border border-border bg-bg-2 dark:bg-bg-dark-2">
      <h2 class="text-lg font-black mb-2">Registered Resources</h2>
      <p class="text-sm text-muted dark:text-muted-dark mb-4">
        Manage database resources, forms, and workflows registered under this panel.
      </p>
      <div class="text-xs font-mono text-muted dark:text-muted-dark p-3 rounded bg-bg dark:bg-bg-dark border border-border-subtle">
        Run 'kiw forge resource <name>' to scaffold a new CRUD resource.
      </div>
    </div>
  </main>
</div>
`, title)

	if dryRun {
		fmt.Printf("[dry-run] Would create: %s\n", goFile)
		fmt.Printf("[dry-run] Would create: %s\n", kiwFile)
		return nil
	}

	if err := os.MkdirAll(panelDir, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(goFile, []byte(goContent), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(kiwFile, []byte(kiwContent), 0o644); err != nil {
		return err
	}

	fmt.Printf("✓ Created panel %s in %s\n", title, filepath.Join("panels", slug))
	fmt.Printf("  └─ %s (Go handler & route configuration)\n", filepath.Join("panels", slug, "panel.go"))
	fmt.Printf("  └─ %s (.kiw Pop-Brutalist dashboard layout)\n", filepath.Join("panels", slug, "Dashboard.kiw"))
	return nil
}

// ── 2. Resource & CRUD Generator ──────────────────────────────────────────────

func forgeResource(root, name, model string, dryRun bool) error {
	slug := forgeToSlug(name)
	title := forgeToTitle(name)
	pascal := forgeToPascal(name)
	if model == "" {
		model = pascal
	}

	resDir := filepath.Join(root, "resources", slug)
	goFile := filepath.Join(resDir, "resource.go")
	indexKiw := filepath.Join(resDir, "Index.kiw")
	formKiw := filepath.Join(resDir, "Form.kiw")

	goContent := fmt.Sprintf(`package %s

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

// %s represents the underlying resource domain model.
type %s struct {
	ID        int64     `+"`json:\"id\" orm:\"id,primary\"`"+`
	Name      string    `+"`json:\"name\" orm:\"name\" forge:\"label=Name,required\"`"+`
	Status    string    `+"`json:\"status\" orm:\"status\" forge:\"label=Status,type=select,options=active|draft|archived\"`"+`
	CreatedAt time.Time `+"`json:\"created_at\" orm:\"created_at\"`"+`
}

// Resource handles CRUD operations for %s.
type Resource struct {
	BaseURL string
}

// NewResource creates a new %s CRUD handler.
func NewResource(baseURL string) *Resource {
	return &Resource{BaseURL: strings.TrimRight(baseURL, "/")}
}

// Routes registers the REST endpoints on the given ServeMux.
func (r *Resource) Routes(mux *http.ServeMux) {
	mux.HandleFunc("GET " + r.BaseURL, r.handleList)
	mux.HandleFunc("POST " + r.BaseURL, r.handleCreate)
	mux.HandleFunc("GET " + r.BaseURL + "/{id}", r.handleShow)
	mux.HandleFunc("PUT " + r.BaseURL + "/{id}", r.handleUpdate)
	mux.HandleFunc("DELETE " + r.BaseURL + "/{id}", r.handleDelete)
}

func (r *Resource) handleList(w http.ResponseWriter, req *http.Request) {
	items := []%s{
		{ID: 1, Name: "Sample %s", Status: "active", CreatedAt: time.Now()},
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(items)
}

func (r *Resource) handleCreate(w http.ResponseWriter, req *http.Request) {
	var item %s
	if err := json.NewDecoder(req.Body).Decode(&item); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	item.ID = time.Now().UnixMilli()
	item.CreatedAt = time.Now()
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(item)
}

func (r *Resource) handleShow(w http.ResponseWriter, req *http.Request) {
	item := %s{ID: 1, Name: "Sample %s", Status: "active", CreatedAt: time.Now()}
	_ = json.NewEncoder(w).Encode(item)
}

func (r *Resource) handleUpdate(w http.ResponseWriter, req *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (r *Resource) handleDelete(w http.ResponseWriter, req *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}
`, slug, model, model, title, pascal, model, title, model, model, title)

	indexContent := fmt.Sprintf(`<script>
  export let title = "%s List"
  export let items = []
</script>

<div class="resource-index space-y-4">
  <div class="flex items-center justify-between">
    <div>
      <h2 class="text-xl font-black tracking-tight">{title}</h2>
      <p class="text-xs text-muted dark:text-muted-dark">Manage and filter %s records.</p>
    </div>
    <Button Variant="primary" Size="sm" Href="/%s/new">+ New %s</Button>
  </div>

  <!-- Data Table -->
  <div class="rounded-xl border border-border overflow-hidden bg-bg-2 dark:bg-bg-dark-2">
    <table class="w-full text-left text-xs font-mono">
      <thead class="bg-bg dark:bg-bg-dark border-b border-border text-muted uppercase">
        <tr>
          <th class="p-3">ID</th>
          <th class="p-3">Name</th>
          <th class="p-3">Status</th>
          <th class="p-3 text-right">Actions</th>
        </tr>
      </thead>
      <tbody class="divide-y divide-border">
        {#each items as item}
          <tr class="hover:bg-bg dark:hover:bg-bg-dark transition-colors">
            <td class="p-3 font-bold">{item.id}</td>
            <td class="p-3 font-sans font-bold">{item.name}</td>
            <td class="p-3"><Badge Variant="neutral" Size="sm">{item.status}</Badge></td>
            <td class="p-3 text-right">
              <a href="/%s/{item.id}" class="text-primary hover:underline mr-2">Edit</a>
              <button class="text-red-500 hover:underline">Delete</button>
            </td>
          </tr>
        {/each}
      </tbody>
    </table>
  </div>
</div>
`, title, title, slug, title, slug)

	formContent := fmt.Sprintf(`<script>
  export let title = "Edit %s"
  export let item = { name: "", status: "active" }
  export let errors = {}
</script>

<div class="resource-form max-w-xl mx-auto p-6 rounded-xl border border-border bg-bg-2 dark:bg-bg-dark-2">
  <h2 class="text-lg font-black mb-4">{title}</h2>

  <form method="POST" class="space-y-4">
    <div>
      <label class="block text-xs font-bold uppercase tracking-wider text-muted mb-1">Name</label>
      <Input Name="name" Value="{item.name}" Placeholder="Enter %s name..." />
      {#if errors.name}
        <span class="text-xs text-red-500 font-bold mt-1 block">{errors.name}</span>
      {/if}
    </div>

    <div>
      <label class="block text-xs font-bold uppercase tracking-wider text-muted mb-1">Status</label>
      <Select Name="status" Value="{item.status}">
        <option value="active">Active</option>
        <option value="draft">Draft</option>
        <option value="archived">Archived</option>
      </Select>
    </div>

    <div class="pt-4 border-t border-border flex items-center justify-end gap-2">
      <Button Variant="neutral" Size="md" Href="/%s">Cancel</Button>
      <Button Variant="primary" Size="md" Type="submit">Save Record</Button>
    </div>
  </form>
</div>
`, title, title, slug)

	if dryRun {
		fmt.Printf("[dry-run] Would create: %s\n", goFile)
		fmt.Printf("[dry-run] Would create: %s\n", indexKiw)
		fmt.Printf("[dry-run] Would create: %s\n", formKiw)
		return nil
	}

	if err := os.MkdirAll(resDir, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(goFile, []byte(goContent), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(indexKiw, []byte(indexContent), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(formKiw, []byte(formContent), 0o644); err != nil {
		return err
	}

	fmt.Printf("✓ Created CRUD resource %s in %s\n", title, filepath.Join("resources", slug))
	fmt.Printf("  └─ %s (REST CRUD routes and Go model)\n", filepath.Join("resources", slug, "resource.go"))
	fmt.Printf("  └─ %s (.kiw table list view)\n", filepath.Join("resources", slug, "Index.kiw"))
	fmt.Printf("  └─ %s (.kiw create/edit form view)\n", filepath.Join("resources", slug, "Form.kiw"))
	return nil
}

// ── 3. Declarative Form Generator ─────────────────────────────────────────────

func forgeForm(root, name, fieldsSpec string, dryRun bool) error {
	slug := forgeToSlug(name)
	title := forgeToTitle(name)
	pascal := forgeToPascal(name)

	formsDir := filepath.Join(root, "forms")
	compDir := filepath.Join(root, "components")
	goFile := filepath.Join(formsDir, slug+"_form.go")
	kiwFile := filepath.Join(compDir, pascal+"Form.kiw")

	// Parse field specifications
	type fieldDef struct {
		Name     string
		Type     string
		Kind     string
		Required bool
	}
	var fields []fieldDef

	if fieldsSpec != "" {
		for _, f := range strings.Split(fieldsSpec, ",") {
			parts := strings.Split(strings.TrimSpace(f), ":")
			if len(parts) >= 2 {
				fName := strings.TrimSpace(parts[0])
				fType := strings.ToLower(strings.TrimSpace(parts[1]))
				fGoType := "string"
				if fType == "number" || fType == "int" {
					fGoType = "int"
				} else if fType == "float" {
					fGoType = "float64"
				} else if fType == "bool" || fType == "checkbox" {
					fGoType = "bool"
				}
				fields = append(fields, fieldDef{
					Name:     fName,
					Type:     fGoType,
					Kind:     fType,
					Required: true,
				})
			}
		}
	}

	// Default fallback fields if none provided
	if len(fields) == 0 {
		fields = []fieldDef{
			{Name: "Name", Type: "string", Kind: "text", Required: true},
			{Name: "Email", Type: "string", Kind: "email", Required: true},
			{Name: "Message", Type: "string", Kind: "textarea", Required: false},
		}
	}

	var goFields strings.Builder
	var kiwFields strings.Builder
	var validateLogic strings.Builder

	for _, f := range fields {
		fTitle := forgeToTitle(f.Name)
		fPascal := forgeToPascal(f.Name)
		fTag := fmt.Sprintf(`forge:"label=%s`, fTitle)
		if f.Required {
			fTag += ",required"
		}
		fTag += `"`
		goFields.WriteString(fmt.Sprintf("\t%s %s `json:\"%s\" %s`\n", fPascal, f.Type, forgeToSlug(f.Name), fTag))

		if f.Required && f.Type == "string" {
			validateLogic.WriteString(fmt.Sprintf("\tif strings.TrimSpace(f.%s) == \"\" {\n\t\terrs[\"%s\"] = \"%s is required\"\n\t}\n", fPascal, forgeToSlug(f.Name), fTitle))
		}

		if f.Kind == "textarea" {
			kiwFields.WriteString(fmt.Sprintf(`    <div>
      <label class="block text-xs font-bold uppercase tracking-wider text-muted mb-1">%s</label>
      <Textarea Name="%s" Rows="4" Placeholder="Enter %s..." />
      {#if errors.%s}<span class="text-xs text-red-500 font-bold block mt-1">{errors.%s}</span>{/if}
    </div>
`, fTitle, forgeToSlug(f.Name), fTitle, forgeToSlug(f.Name), forgeToSlug(f.Name)))
		} else {
			kiwFields.WriteString(fmt.Sprintf(`    <div>
      <label class="block text-xs font-bold uppercase tracking-wider text-muted mb-1">%s</label>
      <Input Name="%s" Type="%s" Placeholder="Enter %s..." />
      {#if errors.%s}<span class="text-xs text-red-500 font-bold block mt-1">{errors.%s}</span>{/if}
    </div>
`, fTitle, forgeToSlug(f.Name), f.Kind, fTitle, forgeToSlug(f.Name), forgeToSlug(f.Name)))
		}
	}

	goContent := fmt.Sprintf(`package forms

import (
	"strings"
)

// %sForm defines the declarative schema and validation rules for %s.
type %sForm struct {
%s}

// Validate checks the submitted values against schema constraints.
func (f *%sForm) Validate() map[string]string {
	errs := make(map[string]string)
%s	return errs
}
`, pascal, title, pascal, goFields.String(), pascal, validateLogic.String())

	kiwContent := fmt.Sprintf(`<script>
  export let action = "/api/%s"
  export let errors = {}
</script>

<form method="POST" action="{action}" class="p-6 rounded-xl border border-border bg-bg-2 dark:bg-bg-dark-2 space-y-4 shadow-sm">
  <h3 class="text-lg font-black tracking-tight mb-2">%s</h3>

%s
  <div class="pt-3 border-t border-border flex justify-end">
    <Button Variant="primary" Size="md" Type="submit">Submit %s</Button>
  </div>
</form>
`, slug, title, kiwFields.String(), title)

	if dryRun {
		fmt.Printf("[dry-run] Would create: %s\n", goFile)
		fmt.Printf("[dry-run] Would create: %s\n", kiwFile)
		return nil
	}

	if err := os.MkdirAll(formsDir, 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(compDir, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(goFile, []byte(goContent), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(kiwFile, []byte(kiwContent), 0o644); err != nil {
		return err
	}

	fmt.Printf("✓ Created declarative form %s\n", title)
	fmt.Printf("  └─ %s (Go validation struct)\n", filepath.Join("forms", slug+"_form.go"))
	fmt.Printf("  └─ %s (.kiw Pop-Brutalist form component)\n", filepath.Join("components", pascal+"Form.kiw"))
	return nil
}

// ── 4. Interactive Visual Studio ──────────────────────────────────────────────

func forgeStudio(port int) error {
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	mux := http.NewServeMux()

	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <title>Krewire Forge Studio</title>
  <style>
    :root { --bg: #071426; --card: #0c203b; --fg: #e2e8f0; --accent: #ff3366; --border: #1e3a5f; }
    body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; background: var(--bg); color: var(--fg); margin: 0; padding: 2rem; }
    .container { max-width: 900px; margin: 0 auto; }
    .card { background: var(--card); border: 2px solid var(--border); border-radius: 12px; padding: 2rem; margin-top: 1.5rem; box-shadow: 4px 4px 0 #000; }
    h1 { font-size: 2rem; font-weight: 900; margin: 0; }
    .badge { background: var(--accent); color: white; padding: 0.2rem 0.6rem; border-radius: 4px; font-size: 0.75rem; font-weight: 800; }
    pre { background: #040d1a; padding: 1rem; border-radius: 8px; border: 1px solid var(--border); font-family: monospace; overflow-x: auto; color: #38bdf8; }
  </style>
</head>
<body>
  <div class="container">
    <div style="display: flex; align-items: center; justify-content: space-between;">
      <h1>Krewire Forge Studio</h1>
      <span class="badge">LOCAL ENGINE</span>
    </div>
    <p style="color: #94a3b8;">Declarative Web, Form, and Admin Panel Builder for Go & .kiw</p>

    <div class="card">
      <h2 style="margin-top: 0;">Quick Scaffolding Commands</h2>
      <p style="color: #94a3b8; font-size: 0.9rem;">Run these commands directly in your project terminal:</p>
      <pre>$ kiw forge panel admin
$ kiw forge resource product --model=Product
$ kiw forge form contact --fields="name:string,email:string,message:textarea"</pre>
    </div>
  </div>
</body>
</html>`))
	})

	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	fmt.Printf("⚡ Krewire Forge Studio running on http://%s\n", addr)
	fmt.Println("Press Ctrl+C to stop.")

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		}
	}()

	<-stop
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return server.Shutdown(ctx)
}

// ── Casing Helpers ────────────────────────────────────────────────────────────

func forgeToSlug(s string) string {
	var b strings.Builder
	prevDash := false
	for i, r := range s {
		if unicode.IsUpper(r) {
			if i > 0 && !prevDash {
				b.WriteRune('-')
			}
			b.WriteRune(unicode.ToLower(r))
			prevDash = false
		} else if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(unicode.ToLower(r))
			prevDash = false
		} else if !prevDash {
			b.WriteRune('-')
			prevDash = true
		}
	}
	res := strings.Trim(b.String(), "-")
	if res == "" {
		return "item"
	}
	return res
}

func forgeToPascal(s string) string {
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return r == '-' || r == '_' || r == ' ' || r == '.'
	})
	var b strings.Builder
	for _, p := range parts {
		if len(p) == 0 {
			continue
		}
		runes := []rune(p)
		runes[0] = unicode.ToUpper(runes[0])
		b.WriteString(string(runes))
	}
	if b.Len() == 0 {
		return "Item"
	}
	return b.String()
}

func forgeToTitle(s string) string {
	slug := forgeToSlug(s)
	words := strings.Split(slug, "-")
	for i, w := range words {
		if len(w) > 0 {
			runes := []rune(w)
			runes[0] = unicode.ToUpper(runes[0])
			words[i] = string(runes)
		}
	}
	return strings.Join(words, " ")
}
