package scaffold

import "fmt"

// Default form fallback fields when no fields are specified.
var defaultFormFields = []formFieldDef{
	{Name: "Name", Type: "string", Kind: "text", Required: true},
	{Name: "Email", Type: "string", Kind: "email", Required: true},
	{Name: "Message", Type: "string", Kind: "textarea", Required: false},
}

type formFieldDef struct {
	Name     string
	Type     string
	Kind     string
	Required bool
}

func panelGoTemplate(slug, title, pascal string) string {
	return fmt.Sprintf(`package %s

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
}

func panelKiwTemplate(title string) string {
	return fmt.Sprintf(`<script>
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
}

func resourceGoTemplate(slug, title, pascal, model string) string {
	return fmt.Sprintf(`package %s

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
}

func resourceIndexKiwTemplate(title, slug string) string {
	return fmt.Sprintf(`<script>
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
}

func resourceFormKiwTemplate(title, slug string) string {
	return fmt.Sprintf(`<script>
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
}

func formGoTemplate(pascal, title, goFields, validateLogic string) string {
	return fmt.Sprintf(`package forms

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
`, pascal, title, pascal, goFields, pascal, validateLogic)
}

func formKiwTemplate(slug, title, kiwFields string) string {
	return fmt.Sprintf(`<script>
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
`, slug, title, kiwFields, title)
}
