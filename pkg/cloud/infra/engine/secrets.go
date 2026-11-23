package engine

import (
	"context"
	"errors"
	"strings"
	"sync"

	"github.com/krewire/krewire/pkg/cloud/infra/contract"
)

// ErrNoSecrets reports a secret reference with no resolver configured.
var ErrNoSecrets = errors.New("infra: no secret resolver configured")

// scrubber wraps the configured resolver and remembers every value it hands
// out, so provider errors can be scrubbed before they reach results, logs,
// or evidence.
type scrubber struct {
	inner  contract.SecretResolver
	mu     sync.Mutex
	values []string
}

func newScrubber(inner contract.SecretResolver) *scrubber { return &scrubber{inner: inner} }

// Resolve implements contract.SecretResolver.
func (s *scrubber) Resolve(ctx context.Context, ref contract.SecretRef) (string, error) {
	if s.inner == nil {
		return "", ErrNoSecrets
	}
	v, err := s.inner.Resolve(ctx, ref)
	if err != nil {
		return "", err
	}
	if v != "" {
		s.mu.Lock()
		s.values = append(s.values, v)
		s.mu.Unlock()
	}
	return v, nil
}

// scrub replaces every resolved secret value in msg.
func (s *scrubber) scrub(msg string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, v := range s.values {
		msg = strings.ReplaceAll(msg, v, contract.Redacted)
	}
	return msg
}

// MapSecrets is a SecretResolver backed by a map keyed by SecretRef.String().
// It is intended for tests and local development.
type MapSecrets map[string]string

// Resolve implements contract.SecretResolver.
func (m MapSecrets) Resolve(_ context.Context, ref contract.SecretRef) (string, error) {
	v, ok := m[ref.String()]
	if !ok {
		return "", errors.New("infra: secret " + ref.String() + " not found")
	}
	return v, nil
}
