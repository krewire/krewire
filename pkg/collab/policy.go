package collab

import (
	"errors"
	"fmt"
	"strings"

	"github.com/krewire/krewire/pkg/model"
)

// Tier identifies the minimum required tier for the collab package.
const Tier = model.TierTeam

// Role defines the access level of a team member.
type Role string

const (
	RoleAdmin       Role = "admin"
	RoleMaintainer  Role = "maintainer"
	RoleContributor Role = "contributor"
	RoleViewer      Role = "viewer"
)

// AllRoles lists all valid roles in ascending order of privilege.
var AllRoles = []Role{RoleViewer, RoleContributor, RoleMaintainer, RoleAdmin}

var roleRanks = map[Role]int{
	RoleViewer:      1,
	RoleContributor: 2,
	RoleMaintainer:  3,
	RoleAdmin:       4,
}

// IsValid reports whether r is a known Role.
func (r Role) IsValid() bool {
	_, ok := roleRanks[r]
	return ok
}

// Rank returns the numeric privilege level (1-4).
func (r Role) Rank() int {
	return roleRanks[r]
}

// AtLeast reports whether r has at least the privilege of target.
func (r Role) AtLeast(target Role) bool {
	return r.Rank() >= target.Rank()
}

// Environment represents a deployment target environment.
type Environment string

const (
	EnvDev     Environment = "dev"
	EnvStaging Environment = "staging"
	EnvProd    Environment = "prod"
)

// Permission represents an action allowed within a team.
type Permission string

const (
	PermRead   Permission = "read"
	PermWrite  Permission = "write"
	PermDeploy Permission = "deploy"
	PermAdmin  Permission = "admin"
)

// EnvironmentPolicy configures minimum roles required per environment.
type EnvironmentPolicy struct {
	MinDeployRole Role `json:"min_deploy_role" yaml:"min_deploy_role"`
	MinWriteRole  Role `json:"min_write_role" yaml:"min_write_role"`
}

// DefaultEnvironmentPolicies returns standard recommended environment policies.
func DefaultEnvironmentPolicies() map[Environment]EnvironmentPolicy {
	return map[Environment]EnvironmentPolicy{
		EnvDev: {
			MinDeployRole: RoleContributor,
			MinWriteRole:  RoleContributor,
		},
		EnvStaging: {
			MinDeployRole: RoleMaintainer,
			MinWriteRole:  RoleContributor,
		},
		EnvProd: {
			MinDeployRole: RoleAdmin,
			MinWriteRole:  RoleMaintainer,
		},
	}
}

// Authorize checks whether a role is authorized to perform an action in an environment.
func Authorize(role Role, env Environment, perm Permission, policies map[Environment]EnvironmentPolicy) error {
	if !role.IsValid() {
		return fmt.Errorf("collab: invalid role %q", role)
	}

	if role == RoleAdmin {
		return nil // Admins bypass all restrictions
	}

	pol, ok := policies[env]
	if !ok {
		// Fallback to dev if unknown environment
		pol = DefaultEnvironmentPolicies()[EnvDev]
	}

	switch perm {
	case PermRead:
		return nil // All team members can read
	case PermWrite:
		if !role.AtLeast(pol.MinWriteRole) {
			return fmt.Errorf("collab: role %s insufficient for write in %s (requires %s)", role, env, pol.MinWriteRole)
		}
	case PermDeploy:
		if !role.AtLeast(pol.MinDeployRole) {
			return fmt.Errorf("collab: role %s insufficient to deploy to %s (requires %s)", role, env, pol.MinDeployRole)
		}
	case PermAdmin:
		return errors.New("collab: admin permission requires admin role")
	default:
		return fmt.Errorf("collab: unknown permission %q", perm)
	}

	return nil
}

// ParseRole parses a string into a Role.
func ParseRole(s string) (Role, error) {
	r := Role(strings.ToLower(strings.TrimSpace(s)))
	if !r.IsValid() {
		return "", fmt.Errorf("unknown role: %q", s)
	}
	return r, nil
}
