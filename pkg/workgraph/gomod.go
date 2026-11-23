package workgraph

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// Require is one go.mod require directive.
type Require struct {
	Path     string `json:"path"`
	Version  string `json:"version"`
	Indirect bool   `json:"indirect,omitempty"`
	Line     int    `json:"line"`
}

// ModFile is the subset of go.mod the graph needs.
type ModFile struct {
	Module    string    `json:"module"`
	Go        string    `json:"go,omitempty"`
	Toolchain string    `json:"toolchain,omitempty"`
	Requires  []Require `json:"requires,omitempty"`
}

// ParseModFile parses go.mod content. It understands module, go, toolchain,
// and single-line or block require directives; other directives are ignored
// (their bytes still feed the module's hash).
func ParseModFile(data []byte) (*ModFile, error) {
	mf := &ModFile{}
	inRequire, inOther := false, false
	for i, raw := range strings.Split(string(data), "\n") {
		line, comment, _ := strings.Cut(raw, "//")
		line = strings.TrimSpace(line)
		indirect := strings.TrimSpace(comment) == "indirect"
		switch {
		case line == "":
			continue
		case inRequire || inOther:
			if line == ")" {
				inRequire, inOther = false, false
			} else if inRequire {
				if err := mf.addRequire(line, indirect, i+1); err != nil {
					return nil, err
				}
			}
			continue
		}
		verb, rest, _ := strings.Cut(line, " ")
		rest = strings.TrimSpace(rest)
		switch verb {
		case "module":
			mf.Module = strings.Trim(rest, `"`)
		case "go":
			mf.Go = rest
		case "toolchain":
			mf.Toolchain = rest
		case "require":
			if rest == "(" {
				inRequire = true
			} else if err := mf.addRequire(rest, indirect, i+1); err != nil {
				return nil, err
			}
		default:
			inOther = strings.HasSuffix(rest, "(")
		}
	}
	if mf.Module == "" {
		return nil, fmt.Errorf("workgraph: go.mod declares no module path")
	}
	return mf, nil
}

func (mf *ModFile) addRequire(spec string, indirect bool, line int) error {
	f := strings.Fields(spec)
	if len(f) != 2 {
		return fmt.Errorf("workgraph: go.mod:%d: malformed require %q", line, spec)
	}
	mf.Requires = append(mf.Requires, Require{Path: strings.Trim(f[0], `"`), Version: f[1], Indirect: indirect, Line: line})
	return nil
}

// ParseWorkFile returns the `use` directories of a go.work file, as written.
func ParseWorkFile(data []byte) []string {
	var dirs []string
	inUse := false
	for _, raw := range strings.Split(string(data), "\n") {
		line, _, _ := strings.Cut(raw, "//")
		line = strings.TrimSpace(line)
		switch {
		case inUse && line == ")":
			inUse = false
		case inUse && line != "":
			dirs = append(dirs, strings.Trim(line, `"`))
		case line == "use (":
			inUse = true
		case strings.HasPrefix(line, "use "):
			dirs = append(dirs, strings.Trim(strings.TrimSpace(line[4:]), `"`))
		}
	}
	return dirs
}

// readModFile loads and parses dir/go.mod.
func readModFile(dir string) (*ModFile, error) {
	data, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	if err != nil {
		return nil, err
	}
	return ParseModFile(data)
}

// matchGlob reports whether the slash path name matches pattern. Segments
// use path.Match syntax; a "**" segment matches zero or more segments.
func matchGlob(pattern, name string) bool {
	return matchSegs(strings.Split(pattern, "/"), strings.Split(name, "/"))
}

func matchSegs(pat, name []string) bool {
	for len(pat) > 0 {
		if pat[0] == "**" {
			for i := 0; i <= len(name); i++ {
				if matchSegs(pat[1:], name[i:]) {
					return true
				}
			}
			return false
		}
		if len(name) == 0 {
			return false
		}
		if ok, _ := path.Match(pat[0], name[0]); !ok {
			return false
		}
		pat, name = pat[1:], name[1:]
	}
	return len(name) == 0
}

func matchAny(patterns []string, name string) bool {
	for _, p := range patterns {
		if matchGlob(p, name) {
			return true
		}
	}
	return false
}
