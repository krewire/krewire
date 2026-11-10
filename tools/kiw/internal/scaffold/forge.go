package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ForgePanel scaffolds a new administrative panel dashboard under panels/<slug>.
func ForgePanel(root, name string, dryRun bool) error {
	slug := ToSlug(name)
	title := ToTitle(name)
	pascal := ToPascal(name)

	panelDir := filepath.Join(root, "panels", slug)
	goFile := filepath.Join(panelDir, "panel.go")
	kiwFile := filepath.Join(panelDir, "Dashboard.kiw")

	goContent := panelGoTemplate(slug, title, pascal)
	kiwContent := panelKiwTemplate(title)

	if dryRun {
		fmt.Printf("[dry-run] Would create: %s\n", goFile)
		fmt.Printf("[dry-run] Would create: %s\n", kiwFile)
		return nil
	}

	if err := os.MkdirAll(panelDir, 0o755); err != nil {
		return err
	}
	if err := safeWriteArtifact(root, goFile, []byte(goContent)); err != nil {
		return err
	}
	if err := safeWriteArtifact(root, kiwFile, []byte(kiwContent)); err != nil {
		return err
	}

	fmt.Printf("✓ Created administrative panel %s in %s\n", title, filepath.Join("panels", slug))
	fmt.Printf("  └─ %s (Go HTTP router and panel definition)\n", filepath.Join("panels", slug, "panel.go"))
	fmt.Printf("  └─ %s (.kiw Pop-Brutalist dashboard view)\n", filepath.Join("panels", slug, "Dashboard.kiw"))
	return nil
}

// ForgeResource scaffolds a complete CRUD resource with model, HTTP routes, Index.kiw and Form.kiw.
func ForgeResource(root, name, model string, dryRun bool) error {
	slug := ToSlug(name)
	title := ToTitle(name)
	pascal := ToPascal(name)

	if model == "" {
		model = pascal
	} else {
		model = ToPascal(model)
	}

	resDir := filepath.Join(root, "resources", slug)
	goFile := filepath.Join(resDir, "resource.go")
	indexKiw := filepath.Join(resDir, "Index.kiw")
	formKiw := filepath.Join(resDir, "Form.kiw")

	goContent := resourceGoTemplate(slug, title, pascal, model)
	indexContent := resourceIndexKiwTemplate(title, slug)
	formContent := resourceFormKiwTemplate(title, slug)

	if dryRun {
		fmt.Printf("[dry-run] Would create: %s\n", goFile)
		fmt.Printf("[dry-run] Would create: %s\n", indexKiw)
		fmt.Printf("[dry-run] Would create: %s\n", formKiw)
		return nil
	}

	if err := os.MkdirAll(resDir, 0o755); err != nil {
		return err
	}
	if err := safeWriteArtifact(root, goFile, []byte(goContent)); err != nil {
		return err
	}
	if err := safeWriteArtifact(root, indexKiw, []byte(indexContent)); err != nil {
		return err
	}
	if err := safeWriteArtifact(root, formKiw, []byte(formContent)); err != nil {
		return err
	}

	fmt.Printf("✓ Created CRUD resource %s in %s\n", title, filepath.Join("resources", slug))
	fmt.Printf("  └─ %s (REST CRUD routes and Go model)\n", filepath.Join("resources", slug, "resource.go"))
	fmt.Printf("  └─ %s (.kiw table list view)\n", filepath.Join("resources", slug, "Index.kiw"))
	fmt.Printf("  └─ %s (.kiw create/edit form view)\n", filepath.Join("resources", slug, "Form.kiw"))
	return nil
}

// ForgeForm generates a declarative .kiw form and accompanying Go validation struct.
func ForgeForm(root, name, fieldsSpec string, dryRun bool) error {
	slug := ToSlug(name)
	title := ToTitle(name)
	pascal := ToPascal(name)

	formsDir := filepath.Join(root, "forms")
	compDir := filepath.Join(root, "components")
	goFile := filepath.Join(formsDir, slug+"_form.go")
	kiwFile := filepath.Join(compDir, pascal+"Form.kiw")

	fields := parseFormFields(fieldsSpec)

	var goFields strings.Builder
	var kiwFields strings.Builder
	var validateLogic strings.Builder

	for _, f := range fields {
		fTitle := ToTitle(f.Name)
		fPascal := ToPascal(f.Name)
		fTag := fmt.Sprintf(`forge:"label=%s`, fTitle)
		if f.Required {
			fTag += ",required"
		}
		fTag += `"`
		goFields.WriteString(fmt.Sprintf("\t%s %s `json:\"%s\" %s`\n", fPascal, f.Type, ToSlug(f.Name), fTag))

		if f.Required && f.Type == "string" {
			validateLogic.WriteString(fmt.Sprintf("\tif strings.TrimSpace(f.%s) == \"\" {\n\t\terrs[\"%s\"] = \"%s is required\"\n\t}\n", fPascal, ToSlug(f.Name), fTitle))
		}

		if f.Kind == "textarea" {
			kiwFields.WriteString(fmt.Sprintf(`    <div>
      <label class="block text-xs font-bold uppercase tracking-wider text-muted mb-1">%s</label>
      <Textarea Name="%s" Rows="4" Placeholder="Enter %s..." />
      {#if errors.%s}<span class="text-xs text-red-500 font-bold block mt-1">{errors.%s}</span>{/if}
    </div>
`, fTitle, ToSlug(f.Name), fTitle, ToSlug(f.Name), ToSlug(f.Name)))
		} else {
			kiwFields.WriteString(fmt.Sprintf(`    <div>
      <label class="block text-xs font-bold uppercase tracking-wider text-muted mb-1">%s</label>
      <Input Name="%s" Type="%s" Placeholder="Enter %s..." />
      {#if errors.%s}<span class="text-xs text-red-500 font-bold block mt-1">{errors.%s}</span>{/if}
    </div>
`, fTitle, ToSlug(f.Name), f.Kind, fTitle, ToSlug(f.Name), ToSlug(f.Name)))
		}
	}

	goContent := formGoTemplate(pascal, title, goFields.String(), validateLogic.String())
	kiwContent := formKiwTemplate(slug, title, kiwFields.String())

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
	if err := safeWriteArtifact(root, goFile, []byte(goContent)); err != nil {
		return err
	}
	if err := safeWriteArtifact(root, kiwFile, []byte(kiwContent)); err != nil {
		return err
	}

	fmt.Printf("✓ Created declarative form %s\n", title)
	fmt.Printf("  └─ %s (Go validation struct)\n", filepath.Join("forms", slug+"_form.go"))
	fmt.Printf("  └─ %s (.kiw Pop-Brutalist form component)\n", filepath.Join("components", pascal+"Form.kiw"))
	return nil
}

func parseFormFields(fieldsSpec string) []formFieldDef {
	var fields []formFieldDef

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
				fields = append(fields, formFieldDef{
					Name:     fName,
					Type:     fGoType,
					Kind:     fType,
					Required: true,
				})
			}
		}
	}

	if len(fields) == 0 {
		return defaultFormFields
	}
	return fields
}
