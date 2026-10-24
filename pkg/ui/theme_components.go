package ui

// componentCSS holds the component-level rules for Krewire UI widget, form, and
// panel classes. It is kept separate from the design tokens in tokens.go so a
// palette change does not touch layout rules and vice versa. The rules are
// static: every value resolves through a --ui-* custom property declared in
// tokens.go, so a caller can override the palette without editing this block.
const componentCSS = `
.ui-app {
  font-family: var(--ui-font-sans);
  background: var(--ui-bg);
  color: var(--ui-fg);
  line-height: 1.5;
  min-height: 100vh;
}

/* Widgets */
.ui-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  padding: 0.55rem 1.15rem;
  font-size: 0.875rem;
  font-weight: 700;
  border-radius: var(--ui-radius);
  border: var(--ui-pop-border);
  box-shadow: var(--ui-pop-shadow-sm);
  background: var(--ui-surface);
  color: var(--ui-fg);
  cursor: pointer;
  text-decoration: none;
  transition: transform 0.12s ease, box-shadow 0.12s ease;
}
.ui-btn:hover {
  transform: translate(-1px, -1px);
  box-shadow: var(--ui-pop-shadow);
}
.ui-btn-primary {
  background: var(--ui-primary);
  color: var(--ui-primary-content);
}
.ui-btn-secondary {
  background: var(--ui-secondary);
  color: var(--ui-secondary-content);
}
.ui-btn-error {
  background: var(--ui-error);
  color: #ffffff;
}
.ui-btn-sm { padding: 0.35rem 0.75rem; font-size: 0.78rem; }
.ui-btn-lg { padding: 0.75rem 1.6rem; font-size: 1rem; }

.ui-badge {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
  padding: 0.2rem 0.6rem;
  font-size: 0.75rem;
  font-weight: 800;
  border-radius: 999px;
  border: 1.5px solid var(--ui-border);
  background: var(--ui-surface);
  color: var(--ui-fg);
  font-family: var(--ui-font-mono);
}
.ui-badge-primary { background: var(--ui-primary); color: var(--ui-primary-content); }
.ui-badge-secondary { background: var(--ui-secondary); color: var(--ui-secondary-content); }
.ui-badge-success { background: var(--ui-success); color: var(--ui-primary-content); }
.ui-badge-warning { background: var(--ui-warning); color: #000; }
.ui-badge-error { background: var(--ui-error); color: #fff; }

.ui-card {
  background: var(--ui-bg);
  border: var(--ui-pop-border);
  border-radius: var(--ui-radius);
  box-shadow: var(--ui-pop-shadow);
  padding: 1.25rem;
}
.ui-card-header {
  border-bottom: 1.5px solid var(--ui-border);
  padding-bottom: 0.75rem;
  margin-bottom: 1rem;
}
.ui-card-title {
  font-size: 1.15rem;
  font-weight: 800;
  margin: 0;
}
.ui-card-desc {
  font-size: 0.85rem;
  color: var(--ui-muted);
  margin: 0.25rem 0 0;
}

.ui-alert {
  padding: 0.9rem 1.2rem;
  border-radius: var(--ui-radius);
  border: var(--ui-pop-border);
  box-shadow: var(--ui-pop-shadow-sm);
  display: flex;
  align-items: center;
  gap: 0.75rem;
  margin-bottom: 1rem;
  font-size: 0.9rem;
  font-weight: 600;
}
.ui-alert-info { background: color-mix(in srgb, var(--ui-secondary) 25%, var(--ui-bg)); }
.ui-alert-success { background: color-mix(in srgb, var(--ui-success) 25%, var(--ui-bg)); }
.ui-alert-warning { background: color-mix(in srgb, var(--ui-warning) 25%, var(--ui-bg)); }
.ui-alert-error { background: color-mix(in srgb, var(--ui-error) 25%, var(--ui-bg)); }

/* Forms */
.ui-form {
  display: flex;
  flex-direction: column;
  gap: 1.15rem;
}
.ui-field-group {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
}
.ui-label {
  font-size: 0.85rem;
  font-weight: 700;
  color: var(--ui-fg);
}
.ui-required {
  color: var(--ui-error);
}
.ui-input, .ui-select, .ui-textarea {
  width: 100%;
  padding: 0.6rem 0.85rem;
  font-size: 0.9rem;
  font-family: inherit;
  border-radius: calc(var(--ui-radius) - 2px);
  border: var(--ui-pop-border);
  background: var(--ui-surface);
  color: var(--ui-fg);
  box-shadow: inset 1px 1px 0px rgba(0,0,0,0.1);
  outline: none;
  box-sizing: border-box;
}
.ui-input:focus, .ui-select:focus, .ui-textarea:focus {
  border-color: var(--ui-primary);
  box-shadow: 0 0 0 2px var(--ui-primary);
}
.ui-field-error {
  font-size: 0.78rem;
  font-weight: 600;
  color: var(--ui-error);
  margin-top: 0.2rem;
}
.ui-field-help {
  font-size: 0.78rem;
  color: var(--ui-muted);
}

/* Panels */
.ui-panel {
  border: var(--ui-pop-border);
  border-radius: var(--ui-radius);
  background: var(--ui-bg);
  box-shadow: var(--ui-pop-shadow);
  overflow: hidden;
}
.ui-panel-head {
  padding: 1rem 1.25rem;
  background: var(--ui-surface);
  border-bottom: var(--ui-pop-border);
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
}
.ui-panel-title {
  font-size: 1.1rem;
  font-weight: 800;
  margin: 0;
}
.ui-panel-body {
  padding: 1.25rem;
}

/* Stats */
.ui-stat {
  padding: 1.25rem;
  border: var(--ui-pop-border);
  border-radius: var(--ui-radius);
  background: var(--ui-bg);
  box-shadow: var(--ui-pop-shadow-sm);
  display: flex;
  flex-direction: column;
  gap: 0.4rem;
}
.ui-stat-label {
  font-size: 0.8rem;
  font-weight: 700;
  color: var(--ui-muted);
  text-transform: uppercase;
  letter-spacing: 0.04em;
}
.ui-stat-value {
  font-size: 1.75rem;
  font-weight: 900;
  color: var(--ui-fg);
}
.ui-stat-change {
  font-size: 0.8rem;
  font-weight: 700;
}
.ui-stat-change.up { color: var(--ui-success); }
.ui-stat-change.down { color: var(--ui-error); }

/* Tables */
.ui-table-wrap {
  width: 100%;
  overflow-x: auto;
  border: var(--ui-pop-border);
  border-radius: var(--ui-radius);
  background: var(--ui-bg);
}
.ui-table {
  width: 100%;
  border-collapse: collapse;
  text-align: left;
  font-size: 0.875rem;
}
.ui-table th {
  padding: 0.75rem 1rem;
  background: var(--ui-surface);
  font-weight: 800;
  border-bottom: var(--ui-pop-border);
  color: var(--ui-fg);
}
.ui-table td {
  padding: 0.75rem 1rem;
  border-bottom: 1px solid var(--ui-border);
}
.ui-table tr:hover {
  background: var(--ui-surface);
}

/* Layouts */
.ui-stack-v { display: flex; flex-direction: column; gap: 1rem; }
.ui-stack-h { display: flex; flex-direction: row; align-items: center; gap: 1rem; flex-wrap: wrap; }
.ui-grid-2 { display: grid; grid-template-columns: repeat(auto-fit, minmax(280px, 1fr)); gap: 1rem; }
.ui-grid-3 { display: grid; grid-template-columns: repeat(auto-fit, minmax(240px, 1fr)); gap: 1rem; }
.ui-grid-4 { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 1rem; }
`
