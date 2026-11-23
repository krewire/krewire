// Package contract defines the versioned infrastructure lifecycle contract
// shared by the infra engine, state stores, and providers (issue #138).
//
// The contract separates three things that the legacy pkg/cloud/infra/provider
// surface mixed together:
//
//   - Desired state: Resource values declared by configuration or a blueprint.
//   - Recorded state: Snapshot / ResourceState values persisted by a store.
//   - Actual state: Observed values returned by a Provider when it reads the
//     real object.
//
// Providers only observe and mutate single resources. Planning, ordering,
// locking, approval, and recovery belong to the engine (pkg/cloud/infra/engine)
// so that every provider inherits the same deterministic lifecycle.
package contract

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
)

// Version is the provider contract version. A provider declares the version it
// implements in Capabilities.Contract; the engine refuses providers that
// declare a different major contract.
const Version = "infra.krewire.dev/v1"

// StateFormatVersion is the on-disk snapshot format version written by stores.
const StateFormatVersion = 1

// SecretRef points at a secret value without carrying it. Plans and snapshots
// store only the reference; providers resolve the value at apply time through
// a SecretResolver.
type SecretRef struct {
	// Source names the secret backend, for example "env" or "file".
	Source string `json:"source"`
	// Key identifies the secret inside the backend.
	Key string `json:"key"`
}

// String renders the reference without any secret material.
func (r SecretRef) String() string { return r.Source + ":" + r.Key }

// SecretResolver resolves secret references at apply time.
type SecretResolver interface {
	Resolve(ctx context.Context, ref SecretRef) (string, error)
}

// Resource is one desired infrastructure object.
type Resource struct {
	// Kind selects the provider schema, for example "file".
	Kind string `json:"kind"`
	// Name is unique within Kind; Kind and Name form the resource address.
	Name string `json:"name"`
	// SchemaVersion is the version of the kind schema the properties follow.
	SchemaVersion string `json:"schemaVersion"`
	// Properties are plain, non-sensitive configuration values.
	Properties map[string]any `json:"properties,omitempty"`
	// Secrets maps a property name to a secret reference resolved at apply time.
	Secrets map[string]SecretRef `json:"secrets,omitempty"`
	// DependsOn lists addresses ("kind/name") that must exist first.
	DependsOn []string `json:"dependsOn,omitempty"`
}

// Address returns the stable "kind/name" identity of the resource.
func (r Resource) Address() string { return Address(r.Kind, r.Name) }

// Address joins a kind and a name into a resource address.
func Address(kind, name string) string { return kind + "/" + name }

// Status is the lifecycle status of a recorded resource.
type Status string

const (
	// StatusReady means the last mutation completed and was recorded.
	StatusReady Status = "ready"
	// StatusPending means a mutation started but its outcome was not recorded;
	// the engine re-reads the object before acting on it again.
	StatusPending Status = "pending"
)

// ResourceState is the recorded view of one resource.
type ResourceState struct {
	Resource
	// ID is the provider-assigned identity, when the provider has one.
	ID string `json:"id,omitempty"`
	// Outputs are provider-reported attributes, already redacted.
	Outputs map[string]any `json:"outputs,omitempty"`
	// Status is ready or pending.
	Status Status `json:"status"`
	// PendingOp is the interrupted operation when Status is pending.
	PendingOp string `json:"pendingOp,omitempty"`
}

// Snapshot is the versioned recorded state for one environment.
type Snapshot struct {
	FormatVersion int    `json:"formatVersion"`
	Lineage       string `json:"lineage"`
	// Serial increases by one on every successful write.
	Serial      uint64          `json:"serial"`
	Environment string          `json:"environment"`
	Provider    string          `json:"provider"`
	Resources   []ResourceState `json:"resources"`
	// Checksum is the integrity digest of every other field, set by stores.
	Checksum string `json:"checksum,omitempty"`
}

// Find returns the recorded resource at address.
func (s *Snapshot) Find(address string) (ResourceState, bool) {
	for _, r := range s.Resources {
		if r.Address() == address {
			return r, true
		}
	}
	return ResourceState{}, false
}

// Put inserts or replaces a recorded resource and keeps the slice sorted.
func (s *Snapshot) Put(rs ResourceState) {
	for i, r := range s.Resources {
		if r.Address() == rs.Address() {
			s.Resources[i] = rs
			return
		}
	}
	s.Resources = append(s.Resources, rs)
	sort.Slice(s.Resources, func(i, j int) bool {
		return s.Resources[i].Address() < s.Resources[j].Address()
	})
}

// Remove deletes the recorded resource at address.
func (s *Snapshot) Remove(address string) {
	out := s.Resources[:0]
	for _, r := range s.Resources {
		if r.Address() != address {
			out = append(out, r)
		}
	}
	s.Resources = out
}

// Clone returns a deep-enough copy for engine bookkeeping: the resource slice
// is copied so Put/Remove on the clone never alias the original.
func (s Snapshot) Clone() Snapshot {
	s.Resources = append([]ResourceState(nil), s.Resources...)
	return s
}

// ErrNotFound reports that the provider found no actual object.
var ErrNotFound = errors.New("infra: resource not found")

// ErrUnsupportedKind reports a kind the provider does not declare.
var ErrUnsupportedKind = errors.New("infra: unsupported resource kind")

// CheckContract verifies that caps declares the contract this package defines.
func CheckContract(caps Capabilities) error {
	if caps.Contract != Version {
		return fmt.Errorf("infra: provider %q implements contract %q, engine requires %q",
			caps.Provider, caps.Contract, Version)
	}
	return nil
}

// sensitiveFragments are property-name fragments that indicate secret
// material. Values under such names must be passed through Secrets.
var sensitiveFragments = []string{
	"password", "passwd", "secret", "token", "apikey", "api_key",
	"privatekey", "private_key", "credential",
}

// IsSensitiveName reports whether a property or output name looks like it
// carries secret material.
func IsSensitiveName(name string) bool {
	n := strings.ToLower(name)
	for _, f := range sensitiveFragments {
		if strings.Contains(n, f) {
			return true
		}
	}
	return false
}

// Redacted is the placeholder written in place of sensitive values.
const Redacted = "(sensitive)"
