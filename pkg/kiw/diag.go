package kiw

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"unicode/utf8"
)

// Diagnostic is a problem located at a position in a .kiw source file.
type Diagnostic struct {
	// File is the source path used in messages (may be empty).
	File string
	// Line and Col are 1-based; Col counts characters.
	Line, Col int
	// Message describes the problem.
	Message string
	// Hint suggests a fix; it may be empty.
	Hint string
	// Err is the underlying template error, if any.
	Err error
}

// Error formats the diagnostic as file:line:col: message, plus a hint line.
func (d *Diagnostic) Error() string {
	var b strings.Builder
	if d.File != "" {
		b.WriteString(d.File + ":")
	}
	fmt.Fprintf(&b, "%d:%d: %s", d.Line, d.Col, d.Message)
	if d.Hint != "" {
		b.WriteString("\n  hint: " + d.Hint)
	}
	return b.String()
}

// Unwrap returns the underlying template error.
func (d *Diagnostic) Unwrap() error { return d.Err }

var (
	// templateErrRe matches the position prefix of text/template and
	// html/template errors: "template: NAME:LINE: msg" (parse),
	// "template: NAME:LINE:COL: msg" (exec) and "html/template:NAME:LINE:COL:
	// msg" (escaping). NAME may contain colons.
	templateErrRe = regexp.MustCompile(`(?:html/)?template: ?(.*?):(\d+)(?::(\d+))?: (.*)`)
	errTokenRe    = regexp.MustCompile(`(?:at|unexpected) <([^>]+)>|function "([^"]+)"|variable "([^"]+)"`)
	frontmatterRe = regexp.MustCompile(`^yaml: line (\d+): (.*)`)
)

// AlignedBody returns the desugared template body of src with every line kept
// at its source line number: the frontmatter and removed <style>, <script>
// and <markdown> blocks become blank lines and multi-line tags are followed by
// the newlines they lost. It is meant for locating errors, not for rendering.
func AlignedBody(src string) string {
	return parseKiw(src, true).Body
}

// Explain turns err, returned by text/template or html/template while parsing
// or executing the body that ParseKiw produced from src under template name
// name, into a *Diagnostic positioned in src. funcs must hold the template
// functions the body was parsed with. Errors that do not belong to name are
// returned unchanged.
func Explain(file, name, src string, funcs map[string]any, err error) error {
	if err == nil {
		return nil
	}
	var d *Diagnostic
	if errors.As(err, &d) {
		return err // already located (a nested component's error)
	}
	m := templateErrRe.FindStringSubmatch(err.Error())
	if m == nil || m[1] != name {
		return err
	}
	if lint := Lint(file, src); len(lint) > 0 {
		lint[0].Err = err
		return &lint[0]
	}
	line, _ := strconv.Atoi(m[2])
	msg := m[4]
	srcLine := 0
	if m[3] != "" {
		col, _ := strconv.Atoi(m[3])
		srcLine = mapExecLine(parseKiw(src, false).Body, AlignedBody(src), line, col)
	} else {
		srcLine = mapParseLine(name, AlignedBody(src), funcs)
	}
	if srcLine <= 0 {
		srcLine = line
	}
	return &Diagnostic{File: file, Line: srcLine, Col: tokenColumn(sourceLine(src, srcLine), msg), Message: msg, Hint: hintFor(msg), Err: err}
}

// mapParseLine re-parses the aligned body and returns the line of the parse error.
func mapParseLine(name, aligned string, funcs map[string]any) int {
	_, err := template.New(name).Funcs(funcs).Parse(aligned)
	if err == nil {
		return 0
	}
	if m := templateErrRe.FindStringSubmatch(err.Error()); m != nil {
		n, _ := strconv.Atoi(m[2])
		return n
	}
	return 0
}

// mapExecLine maps an execution error position (line, 0-based col) in the
// rendered body to a source line. Both bodies hold the same sequence of
// actions, so the k-th "{{" of one is the k-th "{{" of the other.
func mapExecLine(body, aligned string, line, col int) int {
	off := 0
	for i := 1; i < line; i++ {
		nl := strings.IndexByte(body[off:], '\n')
		if nl < 0 {
			return 0
		}
		off += nl + 1
	}
	off = min(off+col+2, len(body))
	k := strings.Count(body[:off], "{{")
	if k == 0 {
		return 0
	}
	idx := 0
	for i := 0; i < k; i++ {
		next := strings.Index(aligned[idx:], "{{")
		if next < 0 {
			return 0
		}
		idx += next
		if i < k-1 {
			idx += 2
		}
	}
	return strings.Count(aligned[:idx], "\n") + 1
}

func sourceLine(src string, n int) string {
	lines := strings.Split(src, "\n")
	if n < 1 || n > len(lines) {
		return ""
	}
	return lines[n-1]
}

// tokenColumn guesses the column of the token an error message names on the
// source line, falling back to the first non-blank character.
func tokenColumn(line, msg string) int {
	var needles []string
	if m := errTokenRe.FindStringSubmatch(msg); m != nil {
		switch tok := m[1] + m[2] + m[3]; tok {
		case "if":
			needles = []string{"{#if", "{{if"}
		case "else":
			needles = []string{"{:else", "{{else"}
		case "end":
			needles = []string{"{/if}", "{/each}", "{{end"}
		case "range":
			needles = []string{"{#each", "{{range"}
		default:
			for _, part := range strings.Split(strings.Trim(tok, ".$"), ".") {
				if part != "" {
					needles = append(needles, part, lowerFirstRune(part))
				}
			}
		}
	}
	for _, n := range needles {
		if i := strings.Index(line, n); i >= 0 {
			return utf8.RuneCountInString(line[:i]) + 1
		}
	}
	return utf8.RuneCountInString(line[:len(line)-len(strings.TrimLeft(line, " \t"))]) + 1
}

func hintFor(msg string) string {
	switch {
	case strings.Contains(msg, `function "dict" not defined`), strings.Contains(msg, `function "component" not defined`):
		return "component tags desugar to {{component}} and {{dict}} calls; render .kiw files through kiw build/dev (pkg/web/ssg) or register those template functions"
	case strings.Contains(msg, "not defined"):
		return "only Go template builtins and the renderer helpers (class, assetLinks, component, dict, …) are available"
	case strings.Contains(msg, "can't evaluate field"):
		return "check the prop name: props are written lower-camel ({variant}) and read as .Variant; inside {#each}, use the loop variable ({item.label})"
	case strings.Contains(msg, "in operand"), strings.Contains(msg, "unexpected"):
		return "a control block ({#if}, {#each}) or brace is out of place; blocks may not appear inside component attributes"
	}
	return ""
}
