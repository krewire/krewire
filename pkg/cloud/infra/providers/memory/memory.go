// Package memory is an in-memory fake infrastructure provider for tests
// (issue #138). It models three kinds with different lifecycle traits and
// supports failure injection and out-of-band changes for drift tests.
package memory

import (
	"context"
	"fmt"
	"maps"
	"sync"

	"github.com/krewire/krewire/pkg/cloud/infra/contract"
)

// Name is the provider name.
const Name = "memory"

// Kinds served by the fake.
const (
	KindBucket  = "bucket"  // replace-only (not updatable)
	KindService = "service" // updatable, takes a secret token
	KindDNS     = "dns"     // externally visible: high risk
)

type object struct {
	props   map[string]any
	secrets map[string]string
}

// Provider is the fake provider. The zero value is not usable; call New.
type Provider struct {
	mu      sync.Mutex
	objects map[string]object
	fail    map[string]error
	seq     int
	// Calls lists mutation calls as "op address", in order.
	Calls []string
	// LeakSecretsInErrors makes injected errors include resolved secret
	// values, simulating a careless provider; the engine must scrub them.
	LeakSecretsInErrors bool
}

// New returns an empty fake provider.
func New() *Provider {
	return &Provider{objects: map[string]object{}, fail: map[string]error{}}
}

// Capabilities implements contract.Provider.
func (p *Provider) Capabilities() contract.Capabilities {
	return contract.Capabilities{
		Contract: contract.Version, Provider: Name, ProviderVersion: "1.0.0",
		Kinds: []contract.KindSchema{
			{Kind: KindBucket, SchemaVersion: "v1", Description: "object bucket",
				Required: []string{"region"}, Optional: []string{"versioning"}, Updatable: false},
			{Kind: KindService, SchemaVersion: "v1", Description: "long-running service",
				Required: []string{"image"}, Optional: []string{"replicas", "port"},
				SecretProps: []string{"token"}, ForceNew: []string{"port"}, Updatable: true,
				SensitiveOutputs: []string{"connection"}},
			{Kind: KindDNS, SchemaVersion: "v1", Description: "public DNS record",
				Required: []string{"target"}, Updatable: true, ExternallyVisible: true},
		},
	}
}

// FailNext makes the next op ("create", "update", "delete", "read") on
// address fail with err.
func (p *Provider) FailNext(op, address string, err error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.fail[op+" "+address] = err
}

// Tamper changes an object's property out of band.
func (p *Provider) Tamper(address, key string, value any) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if o, ok := p.objects[address]; ok {
		o.props[key] = value
	}
}

// Remove deletes an object out of band.
func (p *Provider) Remove(address string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.objects, address)
}

// Exists reports whether address exists.
func (p *Provider) Exists(address string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	_, ok := p.objects[address]
	return ok
}

func (p *Provider) injected(op, address string, secrets map[string]string) error {
	key := op + " " + address
	err, ok := p.fail[key]
	if !ok {
		return nil
	}
	delete(p.fail, key)
	if p.LeakSecretsInErrors {
		for k, v := range secrets {
			err = fmt.Errorf("%w (%s=%s)", err, k, v)
		}
	}
	return err
}

// Read implements contract.Provider.
func (p *Provider) Read(_ context.Context, rec contract.ResourceState) (contract.Observed, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	addr := rec.Address()
	if err := p.injected("read", addr, nil); err != nil {
		return contract.Observed{}, err
	}
	o, ok := p.objects[addr]
	if !ok {
		return contract.Observed{}, contract.ErrNotFound
	}
	return p.observe(addr, o), nil
}

func (p *Provider) observe(addr string, o object) contract.Observed {
	obs := contract.Observed{ID: addr, Properties: maps.Clone(o.props)}
	if _, ok := o.secrets["token"]; ok {
		obs.Outputs = map[string]any{"endpoint": "mem://" + addr, "connection": "mem://" + addr + "?auth=" + o.secrets["token"]}
	}
	return obs
}

func (p *Provider) write(ctx context.Context, op string, r contract.Resource, sr contract.SecretResolver) (contract.Observed, error) {
	secrets := map[string]string{}
	for name, ref := range r.Secrets {
		v, err := sr.Resolve(ctx, ref)
		if err != nil {
			return contract.Observed{}, fmt.Errorf("resolve %s: %w", name, err)
		}
		secrets[name] = v
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	addr := r.Address()
	p.Calls = append(p.Calls, op+" "+addr)
	if err := p.injected(op, addr, secrets); err != nil {
		return contract.Observed{}, err
	}
	o := object{props: maps.Clone(r.Properties), secrets: secrets}
	if o.props == nil {
		o.props = map[string]any{}
	}
	p.objects[addr] = o
	return p.observe(addr, o), nil
}

// Create implements contract.Provider; it converges an existing object.
func (p *Provider) Create(ctx context.Context, r contract.Resource, sr contract.SecretResolver) (contract.Observed, error) {
	return p.write(ctx, "create", r, sr)
}

// Update implements contract.Provider.
func (p *Provider) Update(ctx context.Context, r contract.Resource, _ contract.ResourceState, sr contract.SecretResolver) (contract.Observed, error) {
	return p.write(ctx, "update", r, sr)
}

// Delete implements contract.Provider; deleting an absent object succeeds.
func (p *Provider) Delete(_ context.Context, rec contract.ResourceState) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	addr := rec.Address()
	p.Calls = append(p.Calls, "delete "+addr)
	if err := p.injected("delete", addr, nil); err != nil {
		return err
	}
	delete(p.objects, addr)
	return nil
}
