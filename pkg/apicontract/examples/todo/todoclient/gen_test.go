package todoclient_test

import (
	"testing"

	"github.com/krewire/krewire/pkg/apicontract/clientgen"
	"github.com/krewire/krewire/pkg/apicontract/contracttest"
	"github.com/krewire/krewire/pkg/apicontract/examples/todo/todoapi"
)

// TestGeneratedClientIsCurrent regenerates the client and compares it with
// the committed file; KREWIRE_CONTRACT_UPDATE=1 rewrites it.
func TestGeneratedClientIsCurrent(t *testing.T) {
	src, err := clientgen.Generate(todoapi.Service(), clientgen.Options{
		Package:    "todoclient",
		DefsImport: "github.com/krewire/krewire/pkg/apicontract/examples/todo/todoapi",
	})
	if err != nil {
		t.Fatal(err)
	}
	contracttest.Golden(t, "todoclient.go", src)
}
