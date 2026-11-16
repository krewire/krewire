package kiw

import (
	"fmt"
	"strings"
	"unicode"
)

// desugarInfixExpr converts JavaScript-like boolean & comparison expressions
// (e.g., `count > 0 && active`) into nested Go html/template prefix function calls
// (e.g., `and (gt .Count 0) .Active`).
func desugarInfixExpr(expr string, loopVars map[string]bool) string {
	p := newExprParser(expr, loopVars)
	node := p.parse()
	if node == nil {
		return toGoField(strings.TrimSpace(expr), loopVars)
	}
	res := node.toGo(loopVars)
	// Strip outer wrapping parens if present for clean {{if ...}} output
	if strings.HasPrefix(res, "(") && strings.HasSuffix(res, ")") {
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

// ----------------------------------------------------------------------------
// Expression Lexer & Parser
// ----------------------------------------------------------------------------

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

// ----------------------------------------------------------------------------
// AST Nodes
// ----------------------------------------------------------------------------

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
