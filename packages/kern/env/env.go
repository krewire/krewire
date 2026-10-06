// Package env defines the target environments a Krewire workload runs in and
// how they are resolved from configuration.
package env

import (
	"fmt"
	"strings"

	"github.com/krewire/krewire/packages/kern/errs"
)

// Env is the target environment a workload runs in.
type Env string

const (
	// EnvLocal is the developer-machine default.
	EnvLocal Env = "local"
	// EnvProduction is the live serving environment.
	EnvProduction Env = "production"
	// EnvTesting is the automated-test environment.
	EnvTesting Env = "testing"
)

// DefaultEnv is assumed when no environment is declared anywhere.
const DefaultEnv = EnvLocal

// Envs returns every valid environment in canonical order.
func Envs() []Env {
	return []Env{EnvLocal, EnvProduction, EnvTesting}
}

// String returns the canonical lowercase name.
func (e Env) String() string { return string(e) }

// ParseEnv resolves s into an Env. Empty (after trimming) yields DefaultEnv;
// matching is case-insensitive; anything else is a usage error naming the
// allowed set.
func ParseEnv(s string) (Env, error) {
	switch Env(strings.ToLower(strings.TrimSpace(s))) {
	case "":
		return DefaultEnv, nil
	case EnvLocal:
		return EnvLocal, nil
	case EnvProduction:
		return EnvProduction, nil
	case EnvTesting:
		return EnvTesting, nil
	default:
		return "", errs.UsageError(fmt.Sprintf("unknown environment %q: want one of local, production, testing", s))
	}
}
