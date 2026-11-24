package config

import (
	"reflect"

	"gopkg.in/yaml.v3"
)

func isStructKind(v reflect.Value) bool {
	k := v.Kind()
	if k == reflect.Ptr {
		return isStructKind(derefPtr(v))
	}
	return k == reflect.Struct
}

func derefPtr(v reflect.Value) reflect.Value {
	for v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return reflect.Value{}
		}
		v = v.Elem()
	}
	return v
}

func setScalar(fv reflect.Value, val string) error {
	target := reflect.New(fv.Type())
	if err := yaml.Unmarshal([]byte(val), target.Interface()); err != nil {
		return err
	}
	fv.Set(target.Elem())
	return nil
}
