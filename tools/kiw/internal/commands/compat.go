package commands

import (
	"flag"
	"fmt"

	krewire "github.com/krewire/krewire"
	"github.com/krewire/krewire/packages/kern"
	"github.com/krewire/krewire/tools/kiw/internal/version"
	"github.com/krewire/mdbind"
)

// RunCompat validates version compatibility across the whole Krewire ecosystem.
// It treats each module's `<module>/version.go` (`Version` + `EcosystemRequires`)
// as the authoritative compatibility contract and checks that every declared
// requirement is satisfied by the versions actually declared by the dependencies.
// This is the single entry point that turns the per-module version.go files into
// a repo-wide compatibility gate.
func RunCompat(_ *flag.FlagSet) kern.ExitCode {
	// Actual: each module's own declared version.
	actual := map[string]kern.Version{
		"krewire": krewire.Version,
		"libs":    krewire.Version,
		"kern":    krewire.Version,
		"testing": krewire.Version,
		"hub":     krewire.Version,
		"kiw":     krewire.Version,
		"mdbind":  kern.MustParseVersion(mdbind.Version.String()),
		"boost":   krewire.Version,
	}

	// Requirements: each module's declared EcosystemRequires.
	reqs := map[string]map[string]kern.Version{
		"kiw":    {},
		"mdbind": {},
	}
	for k, v := range version.EcosystemRequires {
		reqs["kiw"][k] = v
	}
	for k, v := range mdbind.EcosystemRequires {
		reqs["mdbind"][string(k)] = kern.MustParseVersion(v.String())
	}

	var issues []error
	for consumer, deps := range reqs {
		for dep, required := range deps {
			act, ok := actual[dep]
			if !ok {
				issues = append(issues, fmt.Errorf("%s requires %s@%s but %s is not in workspace", consumer, dep, required, dep))
				continue
			}
			if !act.IsCompatible(required) {
				issues = append(issues, fmt.Errorf("%s requires %s@%s, got %s", consumer, dep, required, act))
			}
		}
	}
	if len(issues) == 0 {
		fmt.Println("ok: all module version declarations are mutually compatible")
		return kern.ExitCodeSuccess
	}
	for _, err := range issues {
		fmt.Printf("  x %s\n", err.Error())
	}
	return kern.ExitCodeFailure
}
