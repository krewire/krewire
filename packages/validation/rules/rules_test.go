package rules

import (
	"reflect"
	"testing"
)

// str returns a reflect.Value for a string, the most common rule input.
func str(s string) reflect.Value { return reflect.ValueOf(s) }

// num returns a reflect.Value for the given int, used by numeric rules.
func num(i int) reflect.Value { return reflect.ValueOf(i) }

// TestRULES_001_Required verifies the zero-value semantics for each kind a
// required check must treat as absent.
func TestRULES_001_Required(t *testing.T) {
	for _, tc := range []struct {
		name  string
		v     reflect.Value
		fails bool
	}{
		{"empty string", str(""), true},
		{"empty slice", reflect.ValueOf([]string{}), true},
		{"empty map", reflect.ValueOf(map[string]string{}), true},
		{"zero int", num(0), true},
		{"false bool", reflect.ValueOf(false), true},
		{"nil pointer", reflect.ValueOf((*string)(nil)), true},
		{"non-empty string", str("x"), false},
		{"populated slice", reflect.ValueOf([]string{"a"}), false},
		{"non-zero int", num(1), false},
		{"true bool", reflect.ValueOf(true), false},
	} {
		failed, err := EvalRequired("", tc.v)
		if err != nil {
			t.Errorf("%s: unexpected error %v", tc.name, err)
			continue
		}
		if failed != tc.fails {
			t.Errorf("%s: failed = %v, want %v", tc.name, failed, tc.fails)
		}
	}
}

// TestRULES_002_Min verifies the bound applies to numbers and to the length of
// strings, slices, arrays, and maps.
func TestRULES_002_Min(t *testing.T) {
	for _, tc := range []struct {
		name  string
		arg   string
		v     reflect.Value
		fails bool
	}{
		{"int at bound passes", "5", num(5), false},
		{"int below bound fails", "5", num(4), true},
		{"uint at bound passes", "5", reflect.ValueOf(uint(5)), false},
		{"uint below bound fails", "5", reflect.ValueOf(uint(4)), true},
		{"float at bound passes", "1.5", reflect.ValueOf(1.5), false},
		{"float below bound fails", "1.5", reflect.ValueOf(1.0), true},
		{"string length at bound passes", "3", str("abc"), false},
		{"string too short fails", "3", str("ab"), true},
		{"slice at bound passes", "2", reflect.ValueOf([]int{1, 2}), false},
		{"slice too short fails", "2", reflect.ValueOf([]int{1}), true},
		{"map at bound passes", "1", reflect.ValueOf(map[string]int{"a": 1}), false},
		{"array at bound passes", "2", reflect.ValueOf([2]int{1, 2}), false},
	} {
		failed, err := EvalMin(tc.arg, tc.v)
		if err != nil {
			t.Errorf("%s: unexpected error %v", tc.name, err)
			continue
		}
		if failed != tc.fails {
			t.Errorf("%s: failed = %v, want %v", tc.name, failed, tc.fails)
		}
	}
}

// TestRULES_003_MinRejectsBadInput verifies a non-numeric bound and an
// unsupported kind are reported rather than silently passing.
func TestRULES_003_MinRejectsBadInput(t *testing.T) {
	if _, err := EvalMin("abc", num(5)); err == nil {
		t.Error("a non-numeric bound must return an error")
	}
	if _, err := EvalMin("1", reflect.ValueOf(true)); err == nil {
		t.Error("an unsupported kind must return an error")
	}
}

// TestRULES_004_Max mirrors the min coverage for the upper bound.
func TestRULES_004_Max(t *testing.T) {
	for _, tc := range []struct {
		name  string
		arg   string
		v     reflect.Value
		fails bool
	}{
		{"int at bound passes", "5", num(5), false},
		{"int above bound fails", "5", num(6), true},
		{"uint at bound passes", "5", reflect.ValueOf(uint(5)), false},
		{"uint above bound fails", "5", reflect.ValueOf(uint(6)), true},
		{"float at bound passes", "1.5", reflect.ValueOf(1.5), false},
		{"float above bound fails", "1.5", reflect.ValueOf(2.0), true},
		{"string at bound passes", "3", str("abc"), false},
		{"string too long fails", "3", str("abcd"), true},
		{"slice at bound passes", "2", reflect.ValueOf([]int{1, 2}), false},
		{"slice too long fails", "1", reflect.ValueOf([]int{1, 2}), true},
		{"map at bound passes", "1", reflect.ValueOf(map[string]int{"a": 1}), false},
	} {
		failed, err := EvalMax(tc.arg, tc.v)
		if err != nil {
			t.Errorf("%s: unexpected error %v", tc.name, err)
			continue
		}
		if failed != tc.fails {
			t.Errorf("%s: failed = %v, want %v", tc.name, failed, tc.fails)
		}
	}
	if _, err := EvalMax("abc", num(5)); err == nil {
		t.Error("a non-numeric bound must return an error")
	}
	if _, err := EvalMax("1", reflect.ValueOf(true)); err == nil {
		t.Error("an unsupported kind must return an error")
	}
}

// TestRULES_005_Len verifies the exact-length rule across the kinds it supports
// and that unsupported kinds are rejected.
func TestRULES_005_Len(t *testing.T) {
	for _, tc := range []struct {
		name  string
		arg   string
		v     reflect.Value
		fails bool
	}{
		{"string at length passes", "3", str("abc"), false},
		{"string wrong length fails", "3", str("ab"), true},
		{"slice at length passes", "2", reflect.ValueOf([]int{1, 2}), false},
		{"slice wrong length fails", "2", reflect.ValueOf([]int{1}), true},
		{"map at length passes", "1", reflect.ValueOf(map[string]int{"a": 1}), false},
		{"array at length passes", "2", reflect.ValueOf([2]int{}), false},
		{"surrounding spaces tolerated", " 3 ", str("abc"), false},
	} {
		failed, err := EvalLen(tc.arg, tc.v)
		if err != nil {
			t.Errorf("%s: unexpected error %v", tc.name, err)
			continue
		}
		if failed != tc.fails {
			t.Errorf("%s: failed = %v, want %v", tc.name, failed, tc.fails)
		}
	}
	if _, err := EvalLen("3", num(5)); err == nil {
		t.Error("an unsupported kind must return an error")
	}
	if _, err := EvalLen("abc", str("x")); err == nil {
		t.Error("a non-numeric length must return an error")
	}
}

// TestRULES_006_Email verifies address parsing and the string-kind guard.
func TestRULES_006_Email(t *testing.T) {
	for _, ok := range []string{"user@example.com", "first.last+tag@sub.example.co.uk"} {
		failed, err := EvalEmail("", str(ok))
		if err != nil {
			t.Errorf("%q: unexpected error %v", ok, err)
			continue
		}
		if failed {
			t.Errorf("%q must pass the email rule", ok)
		}
	}
	for _, bad := range []string{"not-an-email", "@example.com", "user@", ""} {
		failed, err := EvalEmail("", str(bad))
		if err != nil {
			t.Errorf("%q: unexpected error %v", bad, err)
			continue
		}
		if !failed {
			t.Errorf("%q must fail the email rule", bad)
		}
	}
	if _, err := EvalEmail("", num(5)); err == nil {
		t.Error("the email rule must reject a non-string field")
	}
}

// TestRULES_007_Oneof verifies membership testing for strings, signed and
// unsigned integers, plus the guards for an empty option list and a bad kind.
func TestRULES_007_Oneof(t *testing.T) {
	for _, tc := range []struct {
		name  string
		arg   string
		v     reflect.Value
		fails bool
	}{
		{"listed string passes", "admin editor", str("admin"), false},
		{"unlisted string fails", "admin editor", str("viewer"), true},
		{"listed int passes", "1 2 3", num(2), false},
		{"unlisted int fails", "1 2 3", num(9), true},
		{"listed uint passes", "1 2 3", reflect.ValueOf(uint(3)), false},
		{"unlisted uint fails", "1 2 3", reflect.ValueOf(uint(9)), true},
	} {
		failed, err := EvalOneof(tc.arg, tc.v)
		if err != nil {
			t.Errorf("%s: unexpected error %v", tc.name, err)
			continue
		}
		if failed != tc.fails {
			t.Errorf("%s: failed = %v, want %v", tc.name, failed, tc.fails)
		}
	}
	// An empty option list admits nothing.
	if failed, err := EvalOneof("   ", str("anything")); err != nil || !failed {
		t.Errorf("empty options = failed %v err %v, want fail without error", failed, err)
	}
	if _, err := EvalOneof("a b", reflect.ValueOf(true)); err == nil {
		t.Error("an unsupported kind must return an error")
	}
}

// TestRULES_008_Pattern verifies anchored matching, the string-kind guard, the
// compile-error path, and that the compiled-pattern cache is reused.
func TestRULES_008_Pattern(t *testing.T) {
	if failed, _ := EvalPattern(`^\d{3}$`, str("123")); failed {
		t.Error("a matching value must pass pattern")
	}
	// The rule anchors the pattern, so a partial match must fail.
	if failed, _ := EvalPattern(`\d{3}`, str("1234")); !failed {
		t.Error("pattern must be anchored to the whole value")
	}
	if _, err := EvalPattern(`[`, str("x")); err == nil {
		t.Error("an invalid regex must return an error")
	}
	if _, err := EvalPattern(`^a$`, num(5)); err == nil {
		t.Error("pattern must reject a non-string field")
	}
	// A second evaluation hits the cache and must behave identically.
	if failed, err := EvalPattern(`^\d{3}$`, str("123")); err != nil || failed {
		t.Errorf("cached evaluation = failed %v err %v", failed, err)
	}
}

// TestRULES_009_Registry verifies name dispatch, the unknown-rule error, and that
// Register installs a custom evaluator.
func TestRULES_009_Registry(t *testing.T) {
	if _, err := Evaluate("nosuchrule", "", str("x")); err == nil {
		t.Error("an unknown rule must return an error")
	}
	Register("always-fails", func(string, reflect.Value) (bool, error) { return true, nil })
	if failed, err := Evaluate("always-fails", "", str("x")); err != nil || !failed {
		t.Errorf("registered evaluator = failed %v err %v", failed, err)
	}
	Register("custom-rule", func(string, reflect.Value) (bool, error) { return false, nil })
	if failed, err := Evaluate("custom-rule", "", str("x")); err != nil || failed {
		t.Errorf("second registered evaluator = failed %v err %v", failed, err)
	}
}

// TestRULES_010_Helpers covers the internal length and zero-value helpers
// directly, including the kinds they must decline.
func TestRULES_010_Helpers(t *testing.T) {
	if got, ok := lenOf(str("abcd")); !ok || got != 4 {
		t.Errorf("lenOf(string) = %d %v", got, ok)
	}
	if got, ok := lenOf(reflect.ValueOf([]int{1, 2, 3})); !ok || got != 3 {
		t.Errorf("lenOf(slice) = %d %v", got, ok)
	}
	if _, ok := lenOf(num(5)); ok {
		t.Error("lenOf must decline an int")
	}
	// An invalid reflect.Value must count as zero.
	var invalid reflect.Value
	if !isZero(invalid) {
		t.Error("an invalid Value must count as zero")
	}
	if isZero(str("x")) {
		t.Error("a non-empty string must not be zero")
	}
}
