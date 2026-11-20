# `collab`

Import: `github.com/krewire/krewire/pkg/collab`

## Purpose

Team collaboration primitives for the `team` open-core tier: member
management, hierarchical roles, and per-environment authorization policies.
It supplies the rules a team-scale deployment needs without embedding an
identity provider or a database.

`collab` declares `Tier = model.TierTeam`; it is a paid-tier package and is
never a prerequisite for any `free` module.

## Main API

- `collab.NewTeam(name, slug, ownerEmail string) (*Team, error)`: Creates a team
  with the owner as `admin` and the default environment policies.
- `(*Team).AddMember(email, name string, role Role) (*Member, error)` /
  `(*Team).RemoveMember(email string) error`: Manage membership.
- `(*Team).FindMember(email string) (*Member, bool)`: Look up a member.
- `(*Team).CanDeploy(email string, env Environment) error`: Checks whether a
  member may deploy to an environment.
- `collab.Authorize(role Role, env Environment, perm Permission, policies map[Environment]EnvironmentPolicy) error`:
  Evaluates a permission against the active policy set.
- `collab.ParseRole(s string) (Role, error)` and `collab.DefaultEnvironmentPolicies()`.

### Roles

Ascending privilege: `viewer` → `contributor` → `maintainer` → `admin`.
`Role.AtLeast(target)` and `Role.Rank()` provide the ordering.

### Environments and permissions

Environments are `dev`, `staging`, and `prod`; permissions are `read`, `write`,
`deploy`, and `admin`. Default policies require `contributor` to deploy to
`dev`, `maintainer` to deploy to `staging`, and `admin` to deploy to `prod`;
`admin` bypasses policy checks.

## Example

```go
package main

import (
	"fmt"

	"github.com/krewire/krewire/pkg/collab"
)

func main() {
	team, err := collab.NewTeam("Acme", "acme", "owner@acme.test")
	if err != nil {
		panic(err)
	}

	if _, err := team.AddMember("dev@acme.test", "Dev", collab.RoleMaintainer); err != nil {
		panic(err)
	}

	if err := team.CanDeploy("dev@acme.test", collab.EnvProd); err != nil {
		fmt.Println("denied:", err) // maintainer cannot deploy to prod
	}
}
```

## Design boundary

- **Policy, not storage:** `Team` is a serializable model (`json`/`yaml` tags);
  persistence and transport belong to the consuming application.
- **Tenant-aware by construction:** roles and environment policies are explicit
  values, so callers cannot silently fall back to an implicit superuser.
- **Downward only:** imports `pkg/model` and the standard library; never imports
  `tools/kiw` or a web framework.
