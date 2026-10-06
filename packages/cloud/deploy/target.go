package deploy

import (
	"context"
	"fmt"
)

// Target identifies a deployment target platform.
type Target string

const (
	TargetBinary  Target = "binary"
	TargetGhPages Target = "gh-pages"
	TargetInfra   Target = "infra"
	TargetSystemd Target = "systemd"
)

// Deployer represents an engine capable of deploying project artifacts.
type Deployer interface {
	Target() Target
	Deploy(ctx context.Context, opts Options) (*Result, error)
}

// Options defines deployment parameters.
type Options struct {
	Root        string
	Environment string
	ArtifactDir string
	Branch      string
	Remote      string
	PlanOnly    bool
	AutoApprove bool
}

// Result describes the outcome of a deployment operation.
type Result struct {
	Target  Target
	Success bool
	Message string
	URL     string
}

// Registry stores registered deployer engines.
var Registry = map[Target]Deployer{}

// Register registers a new deployer.
func Register(d Deployer) {
	Registry[d.Target()] = d
}

// Get returns the deployer for a target.
func Get(target Target) (Deployer, error) {
	d, ok := Registry[target]
	if !ok {
		return nil, fmt.Errorf("deploy: unknown target %q", target)
	}
	return d, nil
}
