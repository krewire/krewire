package contract

import (
	"context"
	"fmt"
	"slices"
)

// KindSchema declares one resource kind a provider supports.
type KindSchema struct {
	Kind          string `json:"kind"`
	SchemaVersion string `json:"schemaVersion"`
	Description   string `json:"description,omitempty"`
	// Required lists properties (or secret names) that must be set.
	Required []string `json:"required,omitempty"`
	// Optional lists the remaining accepted properties. Unknown properties are
	// rejected so typos fail validation instead of silently drifting.
	Optional []string `json:"optional,omitempty"`
	// SecretProps lists the properties that may be supplied through Secrets.
	SecretProps []string `json:"secretProps,omitempty"`
	// ForceNew lists properties whose change requires replacement.
	ForceNew []string `json:"forceNew,omitempty"`
	// SensitiveOutputs lists outputs that are always redacted.
	SensitiveOutputs []string `json:"sensitiveOutputs,omitempty"`
	// Updatable reports whether in-place update is supported at all.
	Updatable bool `json:"updatable"`
	// ExternallyVisible marks kinds whose mutation is visible outside the
	// environment (DNS, public endpoints); such changes are high risk.
	ExternallyVisible bool `json:"externallyVisible,omitempty"`
	// CostIncurring marks kinds whose creation incurs spend; high risk.
	CostIncurring bool `json:"costIncurring,omitempty"`
	// Validate performs kind-specific checks on properties.
	Validate func(props map[string]any) error `json:"-"`
}

// Capabilities is the provider's versioned self-description.
type Capabilities struct {
	// Contract must equal Version.
	Contract        string       `json:"contract"`
	Provider        string       `json:"provider"`
	ProviderVersion string       `json:"providerVersion"`
	Kinds           []KindSchema `json:"kinds"`
}

// Kind returns the schema for kind.
func (c Capabilities) Kind(kind string) (KindSchema, bool) {
	for _, k := range c.Kinds {
		if k.Kind == kind {
			return k, true
		}
	}
	return KindSchema{}, false
}

// Observed is the actual state of one object as read by a provider.
type Observed struct {
	ID string `json:"id,omitempty"`
	// Properties are comparable with Resource.Properties; secret-backed
	// properties are never reported.
	Properties map[string]any `json:"properties,omitempty"`
	Outputs    map[string]any `json:"outputs,omitempty"`
}

// Provider is the v1 provider contract. Implementations must satisfy:
//
//   - Read is free of side effects and returns ErrNotFound when the object is
//     absent.
//   - Create converges: if the object already exists it is brought to the
//     desired properties rather than failing.
//   - Update converges the object to the desired properties.
//   - Delete is idempotent: deleting an absent object succeeds.
//   - Errors never include resolved secret values.
//
// The conformance suite in pkg/cloud/infra/conformance verifies these rules.
type Provider interface {
	Capabilities() Capabilities
	Read(ctx context.Context, rec ResourceState) (Observed, error)
	Create(ctx context.Context, r Resource, secrets SecretResolver) (Observed, error)
	Update(ctx context.Context, r Resource, prior ResourceState, secrets SecretResolver) (Observed, error)
	Delete(ctx context.Context, rec ResourceState) error
}

// ValidateResource checks r against the schema: schema version, required and
// known properties, secret placement, and the kind validator.
func (s KindSchema) ValidateResource(r Resource) []string {
	var problems []string
	if r.SchemaVersion != s.SchemaVersion {
		problems = append(problems, fmt.Sprintf("schemaVersion %q is not supported (want %q)", r.SchemaVersion, s.SchemaVersion))
	}
	for _, req := range s.Required {
		_, inProps := r.Properties[req]
		_, inSecrets := r.Secrets[req]
		if !inProps && !inSecrets {
			problems = append(problems, fmt.Sprintf("property %q is required", req))
		}
	}
	for _, name := range sortedKeys(r.Properties) {
		if !slices.Contains(s.Required, name) && !slices.Contains(s.Optional, name) {
			problems = append(problems, fmt.Sprintf("property %q is not declared by schema %s", name, s.SchemaVersion))
		}
		if v, ok := r.Properties[name].(string); ok && v != "" && IsSensitiveName(name) {
			problems = append(problems, fmt.Sprintf("property %q looks sensitive: pass it through secrets", name))
		}
	}
	for _, name := range sortedKeys(r.Secrets) {
		if !slices.Contains(s.SecretProps, name) {
			problems = append(problems, fmt.Sprintf("secret %q is not accepted by kind %s", name, s.Kind))
		}
		if _, dup := r.Properties[name]; dup {
			problems = append(problems, fmt.Sprintf("%q is set both as property and secret", name))
		}
	}
	if s.Validate != nil {
		if err := s.Validate(r.Properties); err != nil {
			problems = append(problems, err.Error())
		}
	}
	return problems
}

func sortedKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	slices.Sort(out)
	return out
}
