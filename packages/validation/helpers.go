package validation

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
)

func interfaceValue(v reflect.Value) any {
	if v.IsValid() && v.CanInterface() {
		return v.Interface()
	}
	return nil
}

// splitRules splits a validate tag into its individual rules, honouring commas
// inside quotes and inside (), {}, or [] so an argument such as
// oneof=admin,editor or pattern=^\d{2,4}$ stays in one piece.
func splitRules(tag string) []string {
	var rules []string
	var cur strings.Builder
	s := ruleSplitter{cur: &cur}

	for i := 0; i < len(tag); i++ {
		s.consume(tag[i], &rules)
	}
	if cur.Len() > 0 {
		rules = append(rules, cur.String())
	}
	return rules
}

// ruleSplitter tracks the nesting state needed to decide whether a comma
// separates rules or belongs to a rule argument. Separating this state from the
// scanning loop keeps splitRules a single-pass reader of the tag.
type ruleSplitter struct {
	cur      *strings.Builder
	quote    byte
	braces   int
	brackets int
	parens   int
}

// consume advances the machine by one byte, flushing the pending rule when a
// top-level comma is reached.
func (s *ruleSplitter) consume(c byte, rules *[]string) {
	switch {
	case s.quote != 0:
		if c == s.quote {
			s.quote = 0
		}
		s.cur.WriteByte(c)
	case c == '"' || c == '\'':
		s.quote = c
		s.cur.WriteByte(c)
	case c == '{':
		s.braces++
		s.cur.WriteByte(c)
	case c == '}':
		closeLevel(&s.braces)
		s.cur.WriteByte(c)
	case c == '[':
		s.brackets++
		s.cur.WriteByte(c)
	case c == ']':
		closeLevel(&s.brackets)
		s.cur.WriteByte(c)
	case c == '(':
		s.parens++
		s.cur.WriteByte(c)
	case c == ')':
		closeLevel(&s.parens)
		s.cur.WriteByte(c)
	case c == ',' && s.braces == 0 && s.brackets == 0 && s.parens == 0:
		*rules = append(*rules, s.cur.String())
		s.cur.Reset()
	default:
		s.cur.WriteByte(c)
	}
}

// closeLevel pops a nesting level without letting an unbalanced closer drive the
// counter negative.
func closeLevel(counter *int) {
	if *counter > 0 {
		*counter--
	}
}

func hasRule(tag, want string) bool {
	for _, r := range splitRules(tag) {
		name, _ := parseRule(r)
		if name == want {
			return true
		}
	}
	return false
}

func parseRule(rule string) (name, arg string) {
	rule = strings.TrimSpace(rule)
	if i := strings.IndexAny(rule, "=:"); i >= 0 {
		name = strings.TrimSpace(rule[:i])
		arg = strings.TrimSpace(rule[i+1:])
		if (strings.HasPrefix(arg, "'") && strings.HasSuffix(arg, "'")) ||
			(strings.HasPrefix(arg, "\"") && strings.HasSuffix(arg, "\"")) {
			if len(arg) >= 2 {
				arg = arg[1 : len(arg)-1]
			}
		}
		return name, arg
	}
	return rule, ""
}

func effective(v reflect.Value) (reflect.Value, bool) {
	nilPtr := false
	for v.Kind() == reflect.Ptr {
		if v.IsNil() {
			nilPtr = true
			break
		}
		v = v.Elem()
	}
	return v, nilPtr
}

func valueKind(v reflect.Value) reflect.Kind {
	t := v.Type()
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t.Kind()
}

func isZero(v reflect.Value) bool {
	if !v.IsValid() {
		return true
	}
	switch v.Kind() {
	case reflect.String, reflect.Slice, reflect.Map:
		return v.Len() == 0
	}
	return v.IsZero()
}

func sortMapKeys(keys []reflect.Value) {
	sort.Slice(keys, func(i, j int) bool {
		return fmt.Sprint(keys[i].Interface()) < fmt.Sprint(keys[j].Interface())
	})
}
