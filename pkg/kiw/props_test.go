package kiw

import (
	"reflect"
	"testing"
)

func TestPropAliases(t *testing.T) {
	cases := map[string][]string{
		"variant":    {"variant", "Variant"},
		"Variant":    {"Variant", "variant"},
		"aria-label": {"aria-label", "AriaLabel", "ariaLabel"},
		"show_value": {"show_value", "ShowValue", "showValue"},
		"id":         {"id", "Id", "ID"},
		"":           nil,
	}
	for in, want := range cases {
		if got := PropAliases(in); !reflect.DeepEqual(got, want) {
			t.Errorf("PropAliases(%q) = %v, want %v", in, got, want)
		}
	}
}
