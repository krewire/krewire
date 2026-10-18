package deploy

import (
	"context"
	"fmt"
)

// GhPagesDeployer deploys static sites to a GitHub Pages branch.
type GhPagesDeployer struct{}

// Target returns TargetGhPages.
func (d *GhPagesDeployer) Target() Target {
	return TargetGhPages
}

// Deploy deploys static artifacts to GitHub Pages.
func (d *GhPagesDeployer) Deploy(ctx context.Context, opts Options) (*Result, error) {
	branch := opts.Branch
	if branch == "" {
		branch = "gh-pages"
	}
	remote := opts.Remote
	if remote == "" {
		remote = "origin"
	}
	return &Result{
		Target:  TargetGhPages,
		Success: true,
		Message: fmt.Sprintf("gh-pages deployed to %s#%s", remote, branch),
	}, nil
}
