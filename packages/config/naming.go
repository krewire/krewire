package config

import (
	"reflect"
	"strings"
)

func fieldName(sf reflect.StructField) string {
	if tag, ok := sf.Tag.Lookup("yaml"); ok {
		if name := strings.Split(tag, ",")[0]; name != "" {
			return name
		}
	}
	return strings.ToLower(sf.Name)
}

func envKey(sf reflect.StructField, baseKey, prefix string) string {
	if tag, ok := sf.Tag.Lookup("env"); ok && tag != "" {
		return tag
	}
	return implicitKey(baseKey, fieldName(sf), prefix)
}

func implicitKey(baseKey, name, prefix string) string {
	upper := strings.NewReplacer("-", "_", ".", "_").Replace(strings.ToUpper(name))
	if baseKey != "" {
		return baseKey + "_" + upper
	}
	return prefix + "_" + upper
}
