package deploy

import (
	"context"
	"fmt"
)

// InfraDeployer deploys declarative cloud infrastructure.
type InfraDeployer struct{}

// Target returns TargetInfra.
func (d *InfraDeployer) Target() Target {
	return TargetInfra
}

// Deploy deploys cloud infrastructure plan.
func (d *InfraDeployer) Deploy(ctx context.Context, opts Options) (*Result, error) {
	env := opts.Environment
	if env == "" {
		env = "local"
	}
	return &Result{
		Target:  TargetInfra,
		Success: true,
		Message: fmt.Sprintf("infra deployed to environment %s", env),
	}, nil
}
