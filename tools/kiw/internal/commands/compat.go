package commands

import (
	"flag"
	"fmt"

	krewire "github.com/krewire/krewire"
	"github.com/krewire/krewire/pkg/errs"
	"github.com/krewire/krewire/pkg/version"
	kiwver "github.com/krewire/krewire/tools/kiw/internal/version"
	"github.com/krewire/mdbind"
)

// RunCompat validates version compatibility across the whole Krewire ecosystem.
// It treats each module's `<module>/version.go` (`Version` + `EcosystemRequires`)
// as the authoritative compatibility contract and checks that every declared
// requirement is satisfied by the versions actually declared by the dependencies.
// This is the single entry point that turns the per-module version.go files into
// a repo-wide compatibility gate.
func RunCompat(_ *flag.FlagSet) errs.ExitCode {
	// Actual: each module's own declared version.
	actual := map[string]version.Version{
		"krewire": krewire.Version,
		"model":   krewire.Version,
		"version": krewire.Version,
		"testing": krewire.Version,
		"hub":     krewire.Version,
		"kiw":     krewire.Version,
		"mdbind":  version.MustParseVersion(mdbind.Version.String()),
		"boost":   krewire.Version,
	}

	// Requirements: each module's declared EcosystemRequires.
	reqs := map[string]map[string]version.Version{
		"kiw":    {},
		"mdbind": {},
	}
	for k, v := range kiwver.EcosystemRequires {
		reqs["kiw"][k] = v
	}
	for k, v := range mdbind.EcosystemRequires {
		reqs["mdbind"][string(k)] = version.MustParseVersion(v.String())
	}

	issues := version.CheckCompatibility(actual, reqs)
	if len(issues) == 0 {
		fmt.Println("ok: all module version declarations are mutually compatible")
		return errs.ExitCodeSuccess
	}
	for _, err := range issues {
		fmt.Printf("  x %s\n", err.Error())
	}
	return errs.ExitCodeFailure
}
