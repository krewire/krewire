package migrate

import (
	"strings"
	"testing"
	"testing/fstest"
)

// generated mirrors the template `kiw make migration` writes.
const generated = `-- Migration: create_users
-- Created at: 2026-11-09T12:00:00Z

-- +up
-- Write your UP migration SQL here:
CREATE TABLE users (id INTEGER PRIMARY KEY);

-- +down
-- Write your DOWN migration SQL here:
DROP TABLE users;
`

func TestParseGeneratedFormat(t *testing.T) {
	m, err := Parse("migrations/20261109120000_create_users.sql", []byte(generated))
	if err != nil {
		t.Fatal(err)
	}
	if m.Version != "20261109120000" || m.Name != "create_users" || m.File != "20261109120000_create_users.sql" {
		t.Fatalf("identity = %q %q %q", m.Version, m.Name, m.File)
	}
	if !strings.Contains(m.Up, "CREATE TABLE users") || strings.Contains(m.Up, "DROP") || strings.Contains(m.Up, "Created at") {
		t.Errorf("up section = %q", m.Up)
	}
	if !strings.Contains(m.Down, "DROP TABLE users") || strings.Contains(m.Down, "CREATE") {
		t.Errorf("down section = %q", m.Down)
	}
	if len(m.Checksum) != 64 {
		t.Errorf("checksum = %q", m.Checksum)
	}
	crlf, _ := Parse("20261109120000_create_users.sql", []byte(strings.ReplaceAll(generated, "\n", "\r\n")))
	if crlf.Checksum != m.Checksum {
		t.Error("CRLF line endings changed the checksum")
	}
}

func TestParseEdgeCases(t *testing.T) {
	m, err := Parse("1_plain.sql", []byte("CREATE TABLE a (id INT);"))
	if err != nil || m.Up != "CREATE TABLE a (id INT);" || m.Down != "" {
		t.Errorf("no markers: up=%q down=%q err=%v", m.Up, m.Down, err)
	}
	if m, _ := Parse("2_x.sql", []byte("--+UP\nSELECT 1;\n--  +Down\nSELECT 2;\n")); !strings.Contains(m.Up, "SELECT 1") || !strings.Contains(m.Down, "SELECT 2") {
		t.Errorf("marker spelling variants: up=%q down=%q", m.Up, m.Down)
	}
	if _, err := Parse("3_x.sql", []byte("-- +up\n-- +down\n-- +up\n")); err == nil {
		t.Error("duplicate marker accepted")
	}
	for _, bad := range []string{"create_users.sql", "v1_users.sql", "20261109_.sql"} {
		if _, err := Parse(bad, nil); err == nil {
			t.Errorf("file name %q accepted", bad)
		}
	}
	if !isBlank("\n-- Write your UP migration SQL here:\n\n") || isBlank("-- c\nSELECT 1;") {
		t.Error("isBlank misjudged comment-only SQL")
	}
}

func TestLoadOrdersAndValidates(t *testing.T) {
	fsys := fstest.MapFS{
		"20261109120000_b.sql": {Data: []byte("SELECT 1;")},
		"9_a.sql":              {Data: []byte("SELECT 1;")},
		"embed.go":             {Data: []byte("package migrations")},
		"sub/1_ignored.sql":    {Data: []byte("SELECT 1;")},
	}
	ms, err := Load(fsys)
	if err != nil {
		t.Fatal(err)
	}
	if len(ms) != 2 || ms[0].Version != "9" || ms[1].Version != "20261109120000" {
		t.Fatalf("order = %v", ms)
	}
	fsys["009_dup.sql"] = &fstest.MapFile{Data: []byte("SELECT 1;")}
	if _, err := Load(fsys); err == nil || !strings.Contains(err.Error(), "share version") {
		t.Errorf("duplicate version: %v", err)
	}
	fsys["notes.sql"] = &fstest.MapFile{Data: []byte("SELECT 1;")}
	if _, err := Load(fsys); err == nil {
		t.Error("unversioned .sql file accepted")
	}
}

func TestVersionLess(t *testing.T) {
	if !versionLess("9", "10") || !versionLess("009", "10") || versionLess("10", "9") || versionLess("1", "001") {
		t.Error("versions must compare numerically")
	}
}
