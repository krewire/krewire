package migrate

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"path"
	"regexp"
	"sort"
	"strings"
)

// Migration is one SQL file in the format `kiw make migration` writes:
//
//	migrations/<version>_<name>.sql
//
//	-- +up
//	CREATE TABLE ...;
//
//	-- +down
//	DROP TABLE ...;
//
// Text before the first marker (the generated header comments) belongs to
// neither section. A file without markers is treated as up-only.
type Migration struct {
	// Version is the leading number of the file name, e.g. "20261109123854".
	Version string
	// Name is the rest of the file name without the extension, e.g. "create_tasks".
	Name string
	// File is the base file name.
	File string
	// Up and Down hold the SQL of the two sections.
	Up, Down string
	// Checksum is the hex SHA-256 of the file with line endings normalized to
	// LF, so a CRLF checkout does not count as an edit.
	Checksum string
}

// String returns the file stem, e.g. "20261109123854_create_tasks".
func (m Migration) String() string { return m.Version + "_" + m.Name }

var (
	fileNameRe = regexp.MustCompile(`^([0-9]+)_([A-Za-z0-9_.-]+)\.sql$`)
	markerRe   = regexp.MustCompile(`(?i)^--\s*\+(up|down)\s*$`)
)

// Load reads every *.sql file at the root of fsys (subdirectories and other
// files, such as an embed.go, are ignored) and returns the migrations sorted
// by version. A file whose name does not start with a version, or two files
// sharing a version, is an error.
func Load(fsys fs.FS) ([]Migration, error) {
	names, err := fs.Glob(fsys, "*.sql")
	if err != nil {
		return nil, err
	}
	out := make([]Migration, 0, len(names))
	seen := map[string]string{}
	for _, name := range names {
		src, err := fs.ReadFile(fsys, name)
		if err != nil {
			return nil, err
		}
		m, err := Parse(name, src)
		if err != nil {
			return nil, err
		}
		key := normVersion(m.Version)
		if prev, dup := seen[key]; dup {
			return nil, fmt.Errorf("migrate: %s and %s share version %s", prev, name, m.Version)
		}
		seen[key] = name
		out = append(out, m)
	}
	sort.Slice(out, func(i, j int) bool { return versionLess(out[i].Version, out[j].Version) })
	return out, nil
}

// Parse parses one migration file. file is its name, used for the version
// and the name; src is its content.
func Parse(file string, src []byte) (Migration, error) {
	base := path.Base(file)
	match := fileNameRe.FindStringSubmatch(base)
	if match == nil {
		return Migration{}, fmt.Errorf("migrate: %s: file name must look like <version>_<name>.sql (create it with `kiw make migration <name>`)", file)
	}
	text := strings.ReplaceAll(string(src), "\r\n", "\n")
	sum := sha256.Sum256([]byte(text))
	m := Migration{
		Version:  match[1],
		Name:     match[2],
		File:     base,
		Checksum: hex.EncodeToString(sum[:]),
	}

	current := "" // "", "up" or "down": the section the next line belongs to
	sections := map[string]*strings.Builder{"up": {}, "down": {}}
	seen := map[string]bool{}
	for _, line := range strings.SplitAfter(text, "\n") {
		if mk := markerRe.FindStringSubmatch(strings.TrimSpace(line)); mk != nil {
			dir := strings.ToLower(mk[1])
			if seen[dir] {
				return Migration{}, fmt.Errorf("migrate: %s: duplicate -- +%s marker", file, dir)
			}
			current, seen[dir] = dir, true
			continue
		}
		if b := sections[current]; b != nil {
			b.WriteString(line)
		}
	}
	if current == "" {
		m.Up = text // no markers: the whole file is the up section
		return m, nil
	}
	m.Up, m.Down = sections["up"].String(), sections["down"].String()
	return m, nil
}

// isBlank reports whether sql holds nothing but whitespace and `--` comments,
// such as the placeholder lines of a freshly generated migration.
func isBlank(sql string) bool {
	for _, line := range strings.Split(sql, "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "--") {
			return false
		}
	}
	return true
}

// normVersion drops leading zeros so "001" and "1" compare equal.
func normVersion(v string) string {
	if t := strings.TrimLeft(v, "0"); t != "" {
		return t
	}
	return "0"
}

// versionLess orders versions numerically without overflowing on long
// timestamps: a shorter normalized number is smaller.
func versionLess(a, b string) bool {
	a, b = normVersion(a), normVersion(b)
	if len(a) != len(b) {
		return len(a) < len(b)
	}
	return a < b
}
