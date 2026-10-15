package kiw

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

var (
	identPathRe       = regexp.MustCompile(`^[a-zA-Z_$][a-zA-Z0-9_$]*(?:\.[a-zA-Z0-9_$]+)*$`)
	eachBlockRe       = regexp.MustCompile(`(?s)\{#each\s+([a-zA-Z0-9_$.]+)\s+as\s+([a-zA-Z0-9_$]+)(?:\s*,\s*([a-zA-Z0-9_$]+))?\s*\}(.*?)\{/each\}`)
	ifTagRe           = regexp.MustCompile(`\{#if\s+([^}]+)\}`)
	elseIfTagRe       = regexp.MustCompile(`\{:else\s+if\s+([^}]+)\}`)
	elseTagRe         = regexp.MustCompile(`\{:else\}`)
	endIfTagRe        = regexp.MustCompile(`\{/if\}`)
	classDirRe        = regexp.MustCompile(`class:([a-zA-Z0-9_-]+)(?:=\{([^}]*)\}|="([^"]*)"|='([^']*)')?`)
	classAttrRe       = regexp.MustCompile(`\bclass=(?:"([^"]*)"|'([^']*)')`)
	tagWithClassDirRe = regexp.MustCompile(`(?s)<([a-zA-Z0-9_-]+)([^>]*?\sclass:[^>]*?)>`)
	punnedPropRe      = regexp.MustCompile(`(?:^|[\s/])\{([a-zA-Z_][a-zA-Z0-9_]*)\}`)
)

func capitalize(s string) string {
	if s == "" {
		return ""
	}
	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

func toGoField(path string, loopVars map[string]bool) string {
	if path == "" {
		return ""
	}
	parts := strings.Split(path, ".")
	root := parts[0]

	// Preserve explicit $ variables
	if strings.HasPrefix(root, "$") {
		var res []string
		res = append(res, root)
		for _, p := range parts[1:] {
			res = append(res, capitalize(p))
		}
		return strings.Join(res, ".")
	}

	// Loop variable reference (e.g. item -> $item)
	if loopVars != nil && loopVars[root] {
		var res []string
		res = append(res, "$"+root)
		for _, p := range parts[1:] {
			res = append(res, capitalize(p))
		}
		return strings.Join(res, ".")
	}

	// Preserve leading dot if already present
	if strings.HasPrefix(root, ".") {
		var res []string
		res = append(res, root)
		for _, p := range parts[1:] {
			res = append(res, capitalize(p))
		}
		return strings.Join(res, ".")
	}

	// In loop context, outer variables need $. prefix to escape shifting scope
	prefix := "."
	if len(loopVars) > 0 {
		prefix = "$."
	}

	var res []string
	res = append(res, prefix+capitalize(root))
	for _, p := range parts[1:] {
		res = append(res, capitalize(p))
	}
	return strings.Join(res, ".")
}

// desugarInterpolation converts single-brace {var} and {path.field} to Go template {{.Path.Field}}.
func desugarInterpolation(src string, loopVars map[string]bool) string {
	var buf strings.Builder
	n := len(src)
	i := 0

	for i < n {
		// Preserve double-brace {{...}} verbatim
		if i+1 < n && src[i] == '{' && src[i+1] == '{' {
			end := strings.Index(src[i+2:], "}}")
			if end != -1 {
				buf.WriteString(src[i : i+2+end+2])
				i += 2 + end + 2
				continue
			}
		}

		// Check for single brace {expr}
		if src[i] == '{' && (i+1 < n && src[i+1] != '{') {
			closeIdx := -1
			depth := 1
			for j := i + 1; j < n; j++ {
				if src[j] == '{' {
					depth++
				} else if src[j] == '}' {
					depth--
					if depth == 0 {
						closeIdx = j
						break
					}
				}
			}

			if closeIdx != -1 {
				content := src[i+1 : closeIdx]
				trimmed := strings.TrimSpace(content)
				if identPathRe.MatchString(trimmed) && trimmed != "true" && trimmed != "false" && trimmed != "nil" && trimmed != "null" {
					goField := toGoField(trimmed, loopVars)
					buf.WriteString("{{")
					buf.WriteString(goField)
					buf.WriteString("}}")
					i = closeIdx + 1
					continue
				}
			}
		}

		buf.WriteByte(src[i])
		i++
	}

	return buf.String()
}

// desugarClassDirectives handles class:name={cond} on HTML tags.
func desugarClassDirectives(src string) string {
	return tagWithClassDirRe.ReplaceAllStringFunc(src, func(tagMatch string) string {
		sub := tagWithClassDirRe.FindStringSubmatch(tagMatch)
		if len(sub) < 3 {
			return tagMatch
		}
		tagName := sub[1]
		attrContent := sub[2]

		cleanAttr := classDirRe.ReplaceAllString(attrContent, "")
		cleanAttr = strings.TrimRight(cleanAttr, " \t\r\n")

		hasClassAttr := classAttrRe.MatchString(cleanAttr)
		var condClasses []string
		classDirs := classDirRe.FindAllStringSubmatch(attrContent, -1)
		for i, m := range classDirs {
			className := m[1]
			expr := ""
			if len(m) > 2 && m[2] != "" {
				expr = m[2]
			} else if len(m) > 3 && m[3] != "" {
				expr = m[3]
			} else if len(m) > 4 && m[4] != "" {
				expr = m[4]
			}
			expr = strings.TrimSpace(expr)
			if expr == "" {
				expr = className
			}
			goCond := desugarInfixExpr(expr, nil)
			space := " "
			if i == 0 && !hasClassAttr {
				space = ""
			}
			condClasses = append(condClasses, fmt.Sprintf("{{if %s}}%s%s{{end}}", goCond, space, className))
		}

		joinedConds := strings.Join(condClasses, "")
		if classMatch := classAttrRe.FindStringSubmatchIndex(cleanAttr); classMatch != nil {
			valStart := classMatch[2]
			valEnd := classMatch[3]
			quote := `"`
			if valStart == -1 {
				valStart = classMatch[4]
				valEnd = classMatch[5]
				quote = `'`
			}
			origVal := cleanAttr[valStart:valEnd]
			newClassVal := fmt.Sprintf(`class=%s%s%s%s`, quote, origVal, joinedConds, quote)
			cleanAttr = cleanAttr[:classMatch[0]] + newClassVal + cleanAttr[classMatch[1]:]
		} else {
			cleanAttr += fmt.Sprintf(` class="%s"`, joinedConds)
		}

		return fmt.Sprintf("<%s%s>", tagName, cleanAttr)
	})
}

// desugarLoops translates {#each items as item, index}...{/each} blocks.
func desugarLoops(src string) string {
	res := src
	for {
		m := eachBlockRe.FindStringSubmatchIndex(res)
		if m == nil {
			break
		}
		itemsExpr := strings.TrimSpace(res[m[2]:m[3]])
		itemVar := strings.TrimSpace(res[m[4]:m[5]])
		indexVar := ""
		if m[6] != -1 && m[7] != -1 {
			indexVar = strings.TrimSpace(res[m[6]:m[7]])
		}
		body := res[m[8]:m[9]]

		loopVars := map[string]bool{itemVar: true}
		if indexVar != "" {
			loopVars[indexVar] = true
		}

		goItems := toGoField(itemsExpr, nil)
		var rangeHeader string
		if indexVar != "" {
			rangeHeader = fmt.Sprintf("{{range $%s, $%s := %s}}", indexVar, itemVar, goItems)
		} else {
			rangeHeader = fmt.Sprintf("{{range $%s := %s}}", itemVar, goItems)
		}

		// Recursively desugar body with loop variable scope
		desugaredBody, _ := DesugarTemplateWithScope(body, loopVars)

		replacement := rangeHeader + desugaredBody + "{{end}}"
		res = res[:m[0]] + replacement + res[m[1]:]
	}
	return res
}

// desugarConditionals converts {#if expr}, {:else if expr}, {:else}, {/if}.
func desugarConditionals(src string, loopVars map[string]bool) string {
	res := ifTagRe.ReplaceAllStringFunc(src, func(match string) string {
		sub := ifTagRe.FindStringSubmatch(match)
		if len(sub) < 2 {
			return match
		}
		expr := strings.TrimSpace(sub[1])
		return fmt.Sprintf("{{if %s}}", desugarInfixExpr(expr, loopVars))
	})

	res = elseIfTagRe.ReplaceAllStringFunc(res, func(match string) string {
		sub := elseIfTagRe.FindStringSubmatch(match)
		if len(sub) < 2 {
			return match
		}
		expr := strings.TrimSpace(sub[1])
		return fmt.Sprintf("{{else if %s}}", desugarInfixExpr(expr, loopVars))
	})

	res = elseTagRe.ReplaceAllString(res, "{{else}}")
	res = endIfTagRe.ReplaceAllString(res, "{{end}}")
	return res
}

// desugarInfixExpr converts infix logic expressions (role == 'admin' && active) to Go template prefix funcs.
func desugarInfixExpr(expr string, loopVars map[string]bool) string {
	p := newExprParser(expr, loopVars)
	node := p.parse()
	if node == nil {
		return toGoField(strings.TrimSpace(expr), loopVars)
	}
	res := node.toGo(loopVars)
	// Strip outer wrapping parens if present for clean {{if ...}} output
	if strings.HasPrefix(res, "(") && strings.HasSuffix(res, ")") {
		// check if matching
		depth := 0
		allWrapped := true
		for i := 0; i < len(res)-1; i++ {
			if res[i] == '(' {
				depth++
			} else if res[i] == ')' {
				depth--
				if depth == 0 {
					allWrapped = false
					break
				}
			}
		}
		if allWrapped {
			res = res[1 : len(res)-1]
		}
	}
	return res
}

// Expression Parser

type exprTokenKind int

const (
	tokEnd exprTokenKind = iota
	tokIdentifier
	tokStrLit
	tokNumLit
	tokBoolLit
	tokOpEq
	tokOpNe
	tokOpLt
	tokOpLe
	tokOpGt
	tokOpGe
	tokOpAnd
	tokOpOr
	tokOpNot
	tokLParen
	tokRParen
)

type exprToken struct {
	kind exprTokenKind
	val  string
}

type exprParser struct {
	src      string
	pos      int
	cur      exprToken
	loopVars map[string]bool
}

func newExprParser(src string, loopVars map[string]bool) *exprParser {
	p := &exprParser{src: src, loopVars: loopVars}
	p.next()
	return p
}

func (p *exprParser) next() {
	p.cur = p.lexToken()
}

func (p *exprParser) lexToken() exprToken {
	n := len(p.src)
	for p.pos < n && unicode.IsSpace(rune(p.src[p.pos])) {
		p.pos++
	}
	if p.pos >= n {
		return exprToken{kind: tokEnd}
	}

	ch := p.src[p.pos]

	if ch == '"' || ch == '\'' {
		quote := ch
		p.pos++
		start := p.pos
		for p.pos < n && p.src[p.pos] != quote {
			p.pos++
		}
		val := p.src[start:p.pos]
		if p.pos < n {
			p.pos++ // skip closing quote
		}
		return exprToken{kind: tokStrLit, val: val}
	}

	if p.pos+1 < n {
		pair := p.src[p.pos : p.pos+2]
		switch pair {
		case "==":
			p.pos += 2
			return exprToken{kind: tokOpEq, val: "=="}
		case "!=":
			p.pos += 2
			return exprToken{kind: tokOpNe, val: "!="}
		case "<=":
			p.pos += 2
			return exprToken{kind: tokOpLe, val: "<="}
		case ">=":
			p.pos += 2
			return exprToken{kind: tokOpGe, val: ">="}
		case "&&":
			p.pos += 2
			return exprToken{kind: tokOpAnd, val: "&&"}
		case "||":
			p.pos += 2
			return exprToken{kind: tokOpOr, val: "||"}
		}
	}

	switch ch {
	case '<':
		p.pos++
		return exprToken{kind: tokOpLt, val: "<"}
	case '>':
		p.pos++
		return exprToken{kind: tokOpGt, val: ">"}
	case '!':
		p.pos++
		return exprToken{kind: tokOpNot, val: "!"}
	case '(':
		p.pos++
		return exprToken{kind: tokLParen, val: "("}
	case ')':
		p.pos++
		return exprToken{kind: tokRParen, val: ")"}
	}

	if unicode.IsDigit(rune(ch)) {
		start := p.pos
		for p.pos < n && (unicode.IsDigit(rune(p.src[p.pos])) || p.src[p.pos] == '.') {
			p.pos++
		}
		return exprToken{kind: tokNumLit, val: p.src[start:p.pos]}
	}

	if unicode.IsLetter(rune(ch)) || ch == '_' || ch == '$' || ch == '.' {
		start := p.pos
		for p.pos < n && (unicode.IsLetter(rune(p.src[p.pos])) || unicode.IsDigit(rune(p.src[p.pos])) || p.src[p.pos] == '_' || p.src[p.pos] == '$' || p.src[p.pos] == '.') {
			p.pos++
		}
		val := p.src[start:p.pos]
		if val == "true" || val == "false" {
			return exprToken{kind: tokBoolLit, val: val}
		}
		return exprToken{kind: tokIdentifier, val: val}
	}

	p.pos++
	return exprToken{kind: tokEnd}
}

type astNode interface {
	toGo(loopVars map[string]bool) string
}

type astIdent struct{ val string }

func (a astIdent) toGo(loopVars map[string]bool) string {
	return toGoField(a.val, loopVars)
}

type astLiteral struct {
	kind exprTokenKind
	val  string
}

func (a astLiteral) toGo(loopVars map[string]bool) string {
	if a.kind == tokStrLit {
		return fmt.Sprintf("%q", a.val)
	}
	return a.val
}

type astUnary struct {
	op  exprTokenKind
	sub astNode
}

func (a astUnary) toGo(loopVars map[string]bool) string {
	return fmt.Sprintf("(not %s)", a.sub.toGo(loopVars))
}

type astBinary struct {
	op    exprTokenKind
	left  astNode
	right astNode
}

func (a astBinary) toGo(loopVars map[string]bool) string {
	fn := ""
	switch a.op {
	case tokOpEq:
		fn = "eq"
	case tokOpNe:
		fn = "ne"
	case tokOpLt:
		fn = "lt"
	case tokOpLe:
		fn = "le"
	case tokOpGt:
		fn = "gt"
	case tokOpGe:
		fn = "ge"
	case tokOpAnd:
		fn = "and"
	case tokOpOr:
		fn = "or"
	}
	return fmt.Sprintf("(%s %s %s)", fn, a.left.toGo(loopVars), a.right.toGo(loopVars))
}

func (p *exprParser) parse() astNode {
	return p.parseOr()
}

func (p *exprParser) parseOr() astNode {
	left := p.parseAnd()
	for p.cur.kind == tokOpOr {
		p.next()
		right := p.parseAnd()
		left = astBinary{op: tokOpOr, left: left, right: right}
	}
	return left
}

func (p *exprParser) parseAnd() astNode {
	left := p.parseRel()
	for p.cur.kind == tokOpAnd {
		p.next()
		right := p.parseRel()
		left = astBinary{op: tokOpAnd, left: left, right: right}
	}
	return left
}

func (p *exprParser) parseRel() astNode {
	left := p.parseUnary()
	for p.cur.kind == tokOpEq || p.cur.kind == tokOpNe || p.cur.kind == tokOpLt || p.cur.kind == tokOpLe || p.cur.kind == tokOpGt || p.cur.kind == tokOpGe {
		op := p.cur.kind
		p.next()
		right := p.parseUnary()
		left = astBinary{op: op, left: left, right: right}
	}
	return left
}

func (p *exprParser) parseUnary() astNode {
	if p.cur.kind == tokOpNot {
		p.next()
		sub := p.parseUnary()
		return astUnary{op: tokOpNot, sub: sub}
	}
	return p.parsePrimary()
}

func (p *exprParser) parsePrimary() astNode {
	switch p.cur.kind {
	case tokLParen:
		p.next()
		node := p.parseOr()
		if p.cur.kind == tokRParen {
			p.next()
		}
		return node
	case tokIdentifier:
		val := p.cur.val
		p.next()
		return astIdent{val: val}
	case tokStrLit, tokNumLit, tokBoolLit:
		k := p.cur.kind
		val := p.cur.val
		p.next()
		return astLiteral{kind: k, val: val}
	default:
		return nil
	}
}
