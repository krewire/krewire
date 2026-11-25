package apphost

import (
	"os"
	"strings"
	"testing"
)

// TestDocumentedSchemaParses keeps the schema example in docs/pkg/apphost.md
// valid and dependency-ordered as documented.
func TestDocumentedSchemaParses(t *testing.T) {
	data, err := os.ReadFile("../../docs/modules/apphost.md")
	if err != nil {
		t.Fatal(err)
	}
	_, rest, ok := strings.Cut(string(data), "```yaml\n")
	block, _, ok2 := strings.Cut(rest, "```")
	if !ok || !ok2 {
		t.Fatal("no yaml example in docs/modules/apphost.md")
	}
	top, err := Parse([]byte(block))
	if err != nil {
		t.Fatalf("documented example is invalid: %v", err)
	}
	if got := strings.Join(top.Order(), ","); got != "billing,cache,db,api,worker" {
		t.Fatalf("order = %s", got)
	}
}
