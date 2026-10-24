package rules

import (
	"fmt"
	"reflect"
	"regexp"
	"sync"
)

var patternCache sync.Map // map[string]*regexp.Regexp

func compilePattern(pattern string) (*regexp.Regexp, error) {
	if cached, ok := patternCache.Load(pattern); ok {
		return cached.(*regexp.Regexp), nil
	}
	re, err := regexp.Compile(`\A(?:` + pattern + `)\z`)
	if err != nil {
		return nil, err
	}
	patternCache.Store(pattern, re)
	return re, nil
}

// EvalPattern evaluates the 'pattern' rule: string must match regex.
func EvalPattern(arg string, v reflect.Value) (bool, error) {
	if v.Kind() != reflect.String {
		return false, fmt.Errorf("rule \"pattern\" requires a string field")
	}
	re, err := compilePattern(arg)
	if err != nil {
		return false, fmt.Errorf("rule \"pattern\": %w", err)
	}
	return !re.MatchString(v.String()), nil
}
