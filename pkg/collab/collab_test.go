package collab_test

import (
	"testing"

	"github.com/krewire/krewire/pkg/collab"
	"github.com/krewire/krewire/pkg/model"
)

func TestCollab_RolesAndPermissions(t *testing.T) {
	if !collab.RoleAdmin.AtLeast(collab.RoleMaintainer) {
		t.Error("admin should be at least maintainer")
	}
	if collab.RoleViewer.AtLeast(collab.RoleContributor) {
		t.Error("viewer should not be at least contributor")
	}

	policies := collab.DefaultEnvironmentPolicies()

	// Contributor can deploy to dev
	if err := collab.Authorize(collab.RoleContributor, collab.EnvDev, collab.PermDeploy, policies); err != nil {
		t.Errorf("contributor should deploy to dev: %v", err)
	}

	// Contributor cannot deploy to staging
	if err := collab.Authorize(collab.RoleContributor, collab.EnvStaging, collab.PermDeploy, policies); err == nil {
		t.Error("contributor should NOT deploy to staging")
	}

	// Maintainer can deploy to staging
	if err := collab.Authorize(collab.RoleMaintainer, collab.EnvStaging, collab.PermDeploy, policies); err != nil {
		t.Errorf("maintainer should deploy to staging: %v", err)
	}

	// Maintainer cannot deploy to prod
	if err := collab.Authorize(collab.RoleMaintainer, collab.EnvProd, collab.PermDeploy, policies); err == nil {
		t.Error("maintainer should NOT deploy to prod")
	}

	// Admin can deploy anywhere
	if err := collab.Authorize(collab.RoleAdmin, collab.EnvProd, collab.PermDeploy, policies); err != nil {
		t.Errorf("admin should deploy to prod: %v", err)
	}
}

func TestCollab_TeamLifecycle(t *testing.T) {
	team, err := collab.NewTeam("Acme Dev", "acme-dev", "founder@acme.corp")
	if err != nil {
		t.Fatalf("NewTeam failed: %v", err)
	}

	if team.Tier != model.TierTeam {
		t.Errorf("expected tier %s, got %s", model.TierTeam, team.Tier)
	}

	// Add members
	dev, err := team.AddMember("dev@acme.corp", "Alice", collab.RoleContributor)
	if err != nil {
		t.Fatalf("AddMember failed: %v", err)
	}
	if dev.Role != collab.RoleContributor {
		t.Errorf("expected role contributor, got %s", dev.Role)
	}

	// Check deploy permission
	if err := team.CanDeploy("dev@acme.corp", collab.EnvDev); err != nil {
		t.Errorf("dev should deploy to dev: %v", err)
	}
	if err := team.CanDeploy("dev@acme.corp", collab.EnvProd); err == nil {
		t.Error("dev should not deploy to prod")
	}

	// Cannot remove the only admin
	if err := team.RemoveMember("founder@acme.corp"); err == nil {
		t.Error("expected error removing the last admin")
	}

	// Can remove dev
	if err := team.RemoveMember("dev@acme.corp"); err != nil {
		t.Errorf("failed to remove dev: %v", err)
	}
	if _, ok := team.FindMember("dev@acme.corp"); ok {
		t.Error("expected dev to be removed")
	}
}
