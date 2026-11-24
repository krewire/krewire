package rules

import (
	"fmt"
	"reflect"
)

// Evaluator is a function that evaluates a validation rule against a reflect.Value.
// It returns true if validation failed, false if it passed, or an error if the rule/argument is malformed.
type Evaluator func(arg string, v reflect.Value) (failed bool, err error)

var registry = map[string]Evaluator{
	"required": EvalRequired,
	"min":      EvalMin,
	"max":      EvalMax,
	"len":      EvalLen,
	"email":    EvalEmail,
	"pattern":  EvalPattern,
	"oneof":    EvalOneof,
}

// Evaluate dispatches a rule evaluation by name.
func Evaluate(name, arg string, v reflect.Value) (bool, error) {
	fn, ok := registry[name]
	if !ok {
		return false, fmt.Errorf("unknown rule %q", name)
	}
	return fn(arg, v)
}

// Register registers or overrides a rule evaluator.
func Register(name string, fn Evaluator) {
	registry[name] = fn
}
