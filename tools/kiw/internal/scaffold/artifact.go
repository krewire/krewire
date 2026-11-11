package scaffold

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"github.com/krewire/krewire/pkg/sec"
)

// ErrUnknownKind is returned when an unsupported artifact kind is requested.
var ErrUnknownKind = errors.New("unknown kind")

// MakeArtifact scaffolds a new component, page, handler, migration, model,
// middleware, or service in the project root.
func MakeArtifact(root, kind, name string) error {
	switch strings.ToLower(kind) {
	case "component":
		return generateComponent(root, name)
	case "page":
		return generatePage(root, name)
	case "handler":
		return generateHandler(root, name)
	case "migration":
		return generateMigration(root, name)
	case "model":
		return generateModel(root, name)
	case "middleware":
		return generateMiddleware(root, name)
	case "service":
		return generateService(root, name)
	case "job", "worker":
		return generateJob(root, name)
	case "chapter", "doc", "docs":
		return generateChapter(root, name)
	default:
		return fmt.Errorf("%w %q — supported: component, page, handler, migration, model, middleware, service, job, chapter, doc", ErrUnknownKind, kind)
	}
}

func generateChapter(root, name string) error {
	name = strings.TrimSuffix(name, ".md")
	title := toTitle(name)
	slug := toSlug(name)

	// Determine destination directory
	destDir := "content"
	if hasDir(root, filepath.Join("content", "docs")) {
		destDir = filepath.Join("content", "docs")
	} else if hasDir(root, filepath.Join("content", "en", "docs")) {
		destDir = filepath.Join("content", "en", "docs")
	}

	fullDir := filepath.Join(root, destDir)
	if err := os.MkdirAll(fullDir, 0o755); err != nil {
		return err
	}

	// Check if any existing file already has the same slug or name
	entries, _ := os.ReadDir(fullDir)
	mdCount := 0
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(strings.ToLower(e.Name()), ".md") {
			mdCount++
			base := strings.TrimSuffix(e.Name(), ".md")
			parts := strings.SplitN(base, "-", 2)
			if len(parts) == 2 && isNumeric(parts[0]) {
				base = parts[1]
			}
			if toSlug(base) == slug {
				return fmt.Errorf("chapter %q already exists at %s", name, filepath.Join(destDir, e.Name()))
			}
		}
	}
	nextOrder := mdCount + 1

	// File name formatting: if user didn't include numbers, prefix with 01-, 02-, etc.
	fileName := slug + ".md"
	if len(slug) > 0 && !unicode.IsDigit(rune(slug[0])) {
		fileName = fmt.Sprintf("%02d-%s.md", nextOrder, slug)
	}

	target := filepath.Join(fullDir, fileName)
	if _, err := os.Stat(target); err == nil {
		return fmt.Errorf("chapter %q already exists at %s", name, filepath.Join(destDir, fileName))
	}

	content := chapterTemplate(title, nextOrder)

	if err := safeWriteArtifact(root, target, []byte(content)); err != nil {
		return err
	}
	fmt.Printf("created %s\n", filepath.ToSlash(filepath.Join(destDir, fileName)))
	return nil
}

func generateComponent(root, name string) error {
	name = strings.TrimSuffix(name, ".kiw")
	name = strings.ReplaceAll(name, "\\", "/")

	// Support compound and nested components e.g. "Card.Header" or "Card/Header"
	if strings.Contains(name, "/") || strings.Contains(name, ".") {
		normalized := strings.ReplaceAll(name, ".", "/")
		parts := strings.Split(normalized, "/")
		var pascalParts []string
		for _, p := range parts {
			if trimmed := strings.TrimSpace(p); trimmed != "" {
				pascalParts = append(pascalParts, toPascal(trimmed))
			}
		}
		if len(pascalParts) > 1 {
			parentSegments := pascalParts[:len(pascalParts)-1]
			basePascal := pascalParts[len(pascalParts)-1]
			dirParts := append([]string{root, "components"}, parentSegments...)
			dir := filepath.Join(dirParts...)
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return err
			}
			target := filepath.Join(dir, basePascal+".kiw")
			if _, err := os.Stat(target); err == nil {
				rel, _ := filepath.Rel(root, target)
				return fmt.Errorf("component already exists at %s", filepath.ToSlash(rel))
			}
			pascal := strings.Join(pascalParts, ".")
			slug := toSlug(basePascal)
			content := componentTemplate(pascal, slug)
			if err := safeWriteArtifact(root, target, []byte(content)); err != nil {
				return err
			}
			rel, _ := filepath.Rel(root, target)
			fmt.Printf("created %s\n", filepath.ToSlash(rel))
			return nil
		}
	}

	pascal := toPascal(name)
	slug := toSlug(name)
	dir := filepath.Join(root, "components")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	target := filepath.Join(dir, pascal+".kiw")
	if _, err := os.Stat(target); err == nil {
		return fmt.Errorf("component %q already exists at components/%s.kiw", pascal, pascal)
	}

	content := componentTemplate(pascal, slug)

	if err := safeWriteArtifact(root, target, []byte(content)); err != nil {
		return err
	}
	fmt.Printf("created components/%s.kiw\n", pascal)
	return nil
}

func generatePage(root, name string) error {
	name = strings.TrimSuffix(name, ".kiw")
	slug := toSlug(name)
	title := toTitle(name)
	dir := filepath.Join(root, "pages")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	target := filepath.Join(dir, slug+".kiw")
	if _, err := os.Stat(target); err == nil {
		return fmt.Errorf("page %q already exists at pages/%s.kiw", slug, slug)
	}

	content := pageTemplate(title, slug)

	if err := safeWriteArtifact(root, target, []byte(content)); err != nil {
		return err
	}
	fmt.Printf("created pages/%s.kiw\n", slug)
	return nil
}

func generateHandler(root, name string) error {
	name = strings.TrimSuffix(name, ".go")
	pascal := toPascal(name)
	slug := toSlug(name)

	destDir := filepath.Join(root, "internal", "handlers")
	if hasDir(root, "handlers") {
		destDir = filepath.Join(root, "handlers")
	}
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return err
	}

	fileName := strings.ReplaceAll(slug, "-", "_") + ".go"
	target := filepath.Join(destDir, fileName)
	if _, err := os.Stat(target); err == nil {
		rel, _ := filepath.Rel(root, target)
		return fmt.Errorf("handler already exists at %s", rel)
	}

	content := handlerTemplate(pascal, name)

	if err := safeWriteArtifact(root, target, []byte(content)); err != nil {
		return err
	}
	rel, _ := filepath.Rel(root, target)
	fmt.Printf("created %s\n", filepath.ToSlash(rel))
	return nil
}

func generateMigration(root, name string) error {
	_, err := MakeMigration(root, "migrations", name)
	return err
}

// MakeMigration writes a timestamped migration into dir (relative to root)
// and returns its path relative to root. By default, it generates a programmatic
// Go migration (.go) with fluent schema.Builder syntax identical to Laravel.
// If name has a ".sql" extension, it creates a raw SQL migration with "-- +up"
// and "-- +down" sections.
// When another migration already uses the current second, the timestamp moves
// forward so versions stay unique.
func MakeMigration(root, dir, name string) (string, error) {
	isSQL := strings.HasSuffix(name, ".sql")
	name = strings.TrimSuffix(name, ".sql")
	name = strings.TrimSuffix(name, ".go")
	slug := strings.ReplaceAll(toSlug(name), "-", "_")
	if slug == "" {
		return "", fmt.Errorf("migration name %q has no letters or digits", name)
	}
	full := filepath.Join(root, dir)
	if err := os.MkdirAll(full, 0o755); err != nil {
		return "", err
	}

	ext := ".go"
	if isSQL {
		ext = ".sql"
	}

	now := time.Now().UTC()
	for {
		takenGo, err := filepath.Glob(filepath.Join(full, now.Format("20060102150405")+"_*.go"))
		if err != nil {
			return "", err
		}
		takenSQL, err := filepath.Glob(filepath.Join(full, now.Format("20060102150405")+"_*.sql"))
		if err != nil {
			return "", err
		}
		if len(takenGo) == 0 && len(takenSQL) == 0 {
			break
		}
		now = now.Add(time.Second)
	}
	version := now.Format("20060102150405")
	fileName := fmt.Sprintf("%s_%s%s", version, slug, ext)
	target := filepath.Join(full, fileName)

	var content string
	if isSQL {
		content = migrationTemplate(name, now.Format(time.RFC3339))
	} else {
		pkgName := "migrations"
		if base := filepath.Base(dir); base != "" && base != "." {
			cleanBase := strings.ReplaceAll(toSlug(base), "-", "_")
			if cleanBase != "" {
				pkgName = cleanBase
			}
		}

		// Ensure a package entry point exists if dir has no other Go files
		entryFile := filepath.Join(full, pkgName+".go")
		if _, err := os.Stat(entryFile); os.IsNotExist(err) {
			_ = safeWriteArtifact(root, entryFile, []byte(migrationsPackageTemplate(pkgName)))
		}

		content = goMigrationTemplate(pkgName, version, slug)
	}

	if err := safeWriteArtifact(root, target, []byte(content)); err != nil {
		return "", err
	}
	rel, _ := filepath.Rel(root, target)
	rel = filepath.ToSlash(rel)
	fmt.Printf("created %s\n", rel)
	return rel, nil
}

func generateModel(root, name string) error {
	name = strings.TrimSuffix(name, ".go")
	pascal := toPascal(name)
	slug := toSlug(name)

	destDir := filepath.Join(root, "internal", "models")
	if hasDir(root, "models") {
		destDir = filepath.Join(root, "models")
	}
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return err
	}

	fileName := strings.ReplaceAll(slug, "-", "_") + ".go"
	target := filepath.Join(destDir, fileName)
	if _, err := os.Stat(target); err == nil {
		rel, _ := filepath.Rel(root, target)
		return fmt.Errorf("model already exists at %s", rel)
	}

	content := modelTemplate(pascal)

	if err := safeWriteArtifact(root, target, []byte(content)); err != nil {
		return err
	}
	rel, _ := filepath.Rel(root, target)
	fmt.Printf("created %s\n", filepath.ToSlash(rel))
	return nil
}

func generateMiddleware(root, name string) error {
	name = strings.TrimSuffix(name, ".go")
	pascal := toPascal(name)
	slug := toSlug(name)

	destDir := filepath.Join(root, "internal", "middleware")
	if hasDir(root, "middleware") {
		destDir = filepath.Join(root, "middleware")
	}
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return err
	}

	fileName := strings.ReplaceAll(slug, "-", "_") + ".go"
	target := filepath.Join(destDir, fileName)
	if _, err := os.Stat(target); err == nil {
		rel, _ := filepath.Rel(root, target)
		return fmt.Errorf("middleware already exists at %s", rel)
	}

	content := middlewareTemplate(pascal)

	if err := safeWriteArtifact(root, target, []byte(content)); err != nil {
		return err
	}
	rel, _ := filepath.Rel(root, target)
	fmt.Printf("created %s\n", filepath.ToSlash(rel))
	return nil
}

func generateService(root, name string) error {
	name = strings.TrimSuffix(name, ".go")
	pascal := toPascal(name)
	slug := toSlug(name)
	lowerFirst := strings.ToLower(string(pascal[0])) + pascal[1:]

	destDir := filepath.Join(root, "internal", "services")
	if hasDir(root, "services") {
		destDir = filepath.Join(root, "services")
	}
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return err
	}

	fileName := strings.ReplaceAll(slug, "-", "_") + ".go"
	target := filepath.Join(destDir, fileName)
	if _, err := os.Stat(target); err == nil {
		rel, _ := filepath.Rel(root, target)
		return fmt.Errorf("service already exists at %s", rel)
	}

	content := serviceTemplate(pascal, lowerFirst)

	if err := safeWriteArtifact(root, target, []byte(content)); err != nil {
		return err
	}
	rel, _ := filepath.Rel(root, target)
	fmt.Printf("created %s\n", filepath.ToSlash(rel))
	return nil
}

func generateJob(root, name string) error {
	name = strings.TrimSuffix(name, ".go")
	name = strings.TrimSuffix(name, "Job")
	name = strings.TrimSuffix(name, "job")
	pascal := toPascal(name)
	slug := toSlug(name)

	destDir := filepath.Join(root, "internal", "jobs")
	if hasDir(root, "jobs") {
		destDir = filepath.Join(root, "jobs")
	}
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return err
	}

	fileName := strings.ReplaceAll(slug, "-", "_") + "_job.go"
	target := filepath.Join(destDir, fileName)
	if _, err := os.Stat(target); err == nil {
		rel, _ := filepath.Rel(root, target)
		return fmt.Errorf("job already exists at %s", filepath.ToSlash(rel))
	}

	content := jobTemplate(pascal)

	if err := safeWriteArtifact(root, target, []byte(content)); err != nil {
		return err
	}
	rel, _ := filepath.Rel(root, target)
	fmt.Printf("created %s\n", filepath.ToSlash(rel))
	return nil
}

func hasDir(root, path string) bool {
	info, err := os.Stat(filepath.Join(root, path))
	return err == nil && info.IsDir()
}

// safeWriteArtifact validates that target resides strictly within root using sec.SafePath before writing.
func safeWriteArtifact(root, target string, content []byte) error {
	safe, err := sec.SafePath(root, target)
	if err != nil {
		return err
	}
	return os.WriteFile(safe, content, 0o644)
}

func isNumeric(s string) bool {
	for _, r := range s {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return len(s) > 0
}

// ToPascal converts strings like "user-profile", "user_profile", or "user/profile" to "UserProfile".
func ToPascal(s string) string {
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return r == '-' || r == '_' || r == ' ' || r == '.' || r == '/' || r == '\\'
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

func toPascal(s string) string { return ToPascal(s) }

// ToSlug converts strings like "UserProfile" to "user-profile".
func ToSlug(s string) string {
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

func toSlug(s string) string { return ToSlug(s) }

// ToTitle converts strings like "user-profile" to "User Profile".
func ToTitle(s string) string {
	slug := ToSlug(s)
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

func toTitle(s string) string { return ToTitle(s) }
