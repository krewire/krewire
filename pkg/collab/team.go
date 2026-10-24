package collab

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/krewire/krewire/pkg/model"
)

// Member represents a collaborator in a team.
type Member struct {
	ID        string    `json:"id" yaml:"id"`
	Email     string    `json:"email" yaml:"email"`
	Name      string    `json:"name,omitempty" yaml:"name,omitempty"`
	Role      Role      `json:"role" yaml:"role"`
	JoinedAt  time.Time `json:"joined_at" yaml:"joined_at"`
	Suspended bool      `json:"suspended,omitempty" yaml:"suspended,omitempty"`
}

// Team represents a collaborative workspace under the team open-core tier.
type Team struct {
	Name      string                            `json:"name" yaml:"name"`
	Slug      string                            `json:"slug" yaml:"slug"`
	Tier      model.Tier                        `json:"tier" yaml:"tier"`
	Members   []Member                          `json:"members" yaml:"members"`
	Policies  map[Environment]EnvironmentPolicy `json:"policies,omitempty" yaml:"policies,omitempty"`
	CreatedAt time.Time                         `json:"created_at" yaml:"created_at"`
	UpdatedAt time.Time                         `json:"updated_at" yaml:"updated_at"`
}

// NewTeam initializes a team with standard policies and a designated owner/admin.
func NewTeam(name, slug, ownerEmail string) (*Team, error) {
	name = strings.TrimSpace(name)
	slug = strings.TrimSpace(slug)
	ownerEmail = strings.TrimSpace(ownerEmail)

	if name == "" {
		return nil, errors.New("collab: team name cannot be empty")
	}
	if slug == "" {
		slug = strings.ToLower(strings.ReplaceAll(name, " ", "-"))
	}
	if ownerEmail == "" {
		return nil, errors.New("collab: team owner email cannot be empty")
	}

	now := time.Now().UTC()
	return &Team{
		Name:      name,
		Slug:      slug,
		Tier:      Tier,
		Policies:  DefaultEnvironmentPolicies(),
		CreatedAt: now,
		UpdatedAt: now,
		Members: []Member{
			{
				ID:       "usr_" + slug + "_owner",
				Email:    ownerEmail,
				Role:     RoleAdmin,
				JoinedAt: now,
			},
		},
	}, nil
}

// AddMember adds or updates a collaborator in the team.
func (t *Team) AddMember(email, name string, role Role) (*Member, error) {
	if !role.IsValid() {
		return nil, fmt.Errorf("collab: invalid role %q", role)
	}
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return nil, errors.New("collab: member email cannot be empty")
	}

	for i, m := range t.Members {
		if strings.EqualFold(m.Email, email) {
			t.Members[i].Role = role
			if name != "" {
				t.Members[i].Name = name
			}
			t.UpdatedAt = time.Now().UTC()
			return &t.Members[i], nil
		}
	}

	now := time.Now().UTC()
	mem := Member{
		ID:       fmt.Sprintf("usr_%d", time.Now().UnixNano()),
		Email:    email,
		Name:     name,
		Role:     role,
		JoinedAt: now,
	}
	t.Members = append(t.Members, mem)
	t.UpdatedAt = now
	return &mem, nil
}

// RemoveMember removes a member by email from the team.
func (t *Team) RemoveMember(email string) error {
	email = strings.ToLower(strings.TrimSpace(email))
	idx := -1
	for i, m := range t.Members {
		if strings.EqualFold(m.Email, email) {
			if m.Role == RoleAdmin && t.adminCount() <= 1 {
				return errors.New("collab: cannot remove the last remaining admin")
			}
			idx = i
			break
		}
	}

	if idx == -1 {
		return fmt.Errorf("collab: member with email %q not found", email)
	}

	t.Members = append(t.Members[:idx], t.Members[idx+1:]...)
	t.UpdatedAt = time.Now().UTC()
	return nil
}

// FindMember locates a member by email.
func (t *Team) FindMember(email string) (*Member, bool) {
	email = strings.ToLower(strings.TrimSpace(email))
	for _, m := range t.Members {
		if strings.EqualFold(m.Email, email) {
			return &m, true
		}
	}
	return nil, false
}

// CanDeploy checks whether a member is allowed to deploy to a specific environment.
func (t *Team) CanDeploy(email string, env Environment) error {
	mem, ok := t.FindMember(email)
	if !ok {
		return fmt.Errorf("collab: user %q is not a member of team %s", email, t.Name)
	}
	if mem.Suspended {
		return fmt.Errorf("collab: user %q is suspended", email)
	}
	return Authorize(mem.Role, env, PermDeploy, t.Policies)
}

func (t *Team) adminCount() int {
	c := 0
	for _, m := range t.Members {
		if m.Role == RoleAdmin && !m.Suspended {
			c++
		}
	}
	return c
}
