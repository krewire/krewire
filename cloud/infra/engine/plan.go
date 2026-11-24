package engine

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"slices"

	"github.com/krewire/krewire/cloud/infra/contract"
)

// PlanFormatVersion is the version of the JSON plan document.
const PlanFormatVersion = 1

// Op is the action a plan step performs.
type Op string

// Plan operations.
const (
	OpCreate  Op = "create"
	OpUpdate  Op = "update"
	OpReplace Op = "replace"
	OpDelete  Op = "delete"
	OpNoOp    Op = "noop"
)

// Mode says how a plan was computed; Apply recomputes it the same way.
type Mode string

// Plan modes.
const (
	ModeApply     Mode = "apply"
	ModeDestroy   Mode = "destroy"
	ModeReconcile Mode = "reconcile"
)

// Change is one field difference inside an action.
type Change struct {
	Path   string `json:"path"`
	Before any    `json:"before,omitempty"`
	After  any    `json:"after,omitempty"`
}

// Action is one ordered step of a plan.
type Action struct {
	Address string   `json:"address"`
	Op      Op       `json:"op"`
	Risk    Risk     `json:"risk"`
	Reason  string   `json:"reason,omitempty"`
	Changes []Change `json:"changes,omitempty"`
	// Pending marks a resume of an interrupted operation; apply re-reads the
	// object before deciding between create, update, and delete.
	Pending bool `json:"pending,omitempty"`
	// Resource is the desired resource; nil for deletes.
	Resource *contract.Resource `json:"resource,omitempty"`
}

// Summary counts actions by operation.
type Summary struct {
	Create  int `json:"create"`
	Update  int `json:"update"`
	Replace int `json:"replace"`
	Delete  int `json:"delete"`
	NoOp    int `json:"noop"`
}

// Plan is the deterministic, reviewable result of planning. Identical
// inputs always produce byte-identical JSON and the same Digest.
type Plan struct {
	FormatVersion   int      `json:"formatVersion"`
	Contract        string   `json:"contract"`
	Provider        string   `json:"provider"`
	ProviderVersion string   `json:"providerVersion"`
	Environment     string   `json:"environment"`
	Mode            Mode     `json:"mode"`
	Lineage         string   `json:"lineage,omitempty"`
	BaseSerial      uint64   `json:"baseSerial"`
	DriftDigest     string   `json:"driftDigest,omitempty"`
	Actions         []Action `json:"actions"`
	Summary         Summary  `json:"summary"`
	Digest          string   `json:"digest"`
}

// HasChanges reports whether applying the plan would mutate anything.
func (p *Plan) HasChanges() bool {
	for _, a := range p.Actions {
		if a.Op != OpNoOp {
			return true
		}
	}
	return false
}

// MaxRisk returns the highest risk of any action.
func (p *Plan) MaxRisk() Risk {
	max := RiskNone
	for _, a := range p.Actions {
		if a.Risk.rank() > max.rank() {
			max = a.Risk
		}
	}
	return max
}

// JSON renders the plan as indented JSON for review and golden tests.
func (p *Plan) JSON() ([]byte, error) {
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

// ComputeDigest returns the digest of every field except Digest itself.
func (p *Plan) ComputeDigest() (string, error) {
	c := *p
	c.Digest = ""
	data, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

// Plan computes the actions that converge recorded state to desired. It reads
// state but never locks, writes, or calls provider mutations.
func (e *Engine) Plan(ctx context.Context, desired []contract.Resource) (*Plan, error) {
	return e.plan(ctx, ModeApply, desired, nil)
}

// PlanDestroy computes the deletion of every recorded resource.
func (e *Engine) PlanDestroy(ctx context.Context) (*Plan, error) {
	return e.plan(ctx, ModeDestroy, nil, nil)
}

func (e *Engine) plan(ctx context.Context, mode Mode, desired []contract.Resource, drift *DriftReport) (*Plan, error) {
	snap, err := e.Store.Read(ctx)
	if err != nil {
		return nil, err
	}
	return e.planFrom(snap, mode, desired, drift)
}

func (e *Engine) planFrom(snap contract.Snapshot, mode Mode, desired []contract.Resource, drift *DriftReport) (*Plan, error) {
	caps := e.Provider.Capabilities()
	byAddr, err := validate(caps, desired)
	if err != nil {
		return nil, err
	}
	p := &Plan{
		FormatVersion: PlanFormatVersion, Contract: caps.Contract, Provider: caps.Provider,
		ProviderVersion: caps.ProviderVersion, Environment: e.Environment, Mode: mode,
		Lineage: snap.Lineage, BaseSerial: snap.Serial, Actions: []Action{},
	}
	if drift != nil {
		p.DriftDigest = drift.Digest
	}
	recorded := map[string][]string{}
	for _, rs := range snap.Resources {
		if _, keep := byAddr[rs.Address()]; !keep {
			recorded[rs.Address()] = rs.DependsOn
		}
	}
	delOrder, err := order(recorded)
	if err != nil {
		return nil, fmt.Errorf("infra: recorded state: %w", err)
	}
	slices.Reverse(delOrder)
	for _, addr := range delOrder {
		rs, _ := snap.Find(addr)
		p.add(Action{Address: addr, Op: OpDelete, Risk: RiskDestructive, Pending: rs.Status == contract.StatusPending,
			Reason: "not in desired state"})
	}
	addOrder, _ := order(deps(byAddr))
	for _, addr := range addOrder {
		r := byAddr[addr]
		schema, _ := caps.Kind(r.Kind)
		p.add(diffAction(r, schema, snap, drift))
	}
	if p.Digest, err = p.ComputeDigest(); err != nil {
		return nil, err
	}
	return p, nil
}

func (p *Plan) add(a Action) {
	p.Actions = append(p.Actions, a)
	switch a.Op {
	case OpCreate:
		p.Summary.Create++
	case OpUpdate:
		p.Summary.Update++
	case OpReplace:
		p.Summary.Replace++
	case OpDelete:
		p.Summary.Delete++
	default:
		p.Summary.NoOp++
	}
}

// diffAction decides the action for one desired resource.
func diffAction(r contract.Resource, schema contract.KindSchema, snap contract.Snapshot, drift *DriftReport) Action {
	desired := r
	a := Action{Address: r.Address(), Resource: &desired, Risk: kindRisk(schema)}
	rec, ok := snap.Find(a.Address)
	if !ok {
		a.Op, a.Reason, a.Changes = OpCreate, "not in recorded state", diff(contract.Resource{}, r)
		return a
	}
	if rec.Status == contract.StatusPending {
		a.Op, a.Pending = OpCreate, true
		if rec.PendingOp == string(OpUpdate) {
			a.Op = OpUpdate
		}
		a.Reason = "resume interrupted " + rec.PendingOp
		a.Changes = diff(rec.Resource, r)
		return a
	}
	base := rec.Resource
	if d, ok := drift.entry(a.Address); ok {
		switch d.Status {
		case DriftMissing:
			a.Op, a.Reason, a.Changes = OpCreate, "drift: object missing", diff(contract.Resource{}, r)
			return a
		case DriftModified:
			base.Properties = observedProps(base.Properties, d.Changes)
			a.Reason = "drift: object modified"
		}
	}
	a.Changes = diff(base, r)
	if len(a.Changes) == 0 {
		a.Op, a.Risk, a.Reason = OpNoOp, RiskNone, ""
		return a
	}
	a.Op = OpUpdate
	for _, c := range a.Changes {
		if c.Path == "dependsOn" {
			continue
		}
		if !schema.Updatable || slices.Contains(schema.ForceNew, c.Path) {
			a.Op, a.Risk = OpReplace, RiskDestructive
			a.Reason = joinReason(a.Reason, "change to "+c.Path+" requires replacement")
			break
		}
	}
	if a.Reason == "" {
		a.Reason = "configuration changed"
	}
	return a
}

func joinReason(a, b string) string {
	if a == "" {
		return b
	}
	return a + "; " + b
}

// diff lists field changes from before to after, sorted by path. Values under
// sensitive-looking names are redacted.
func diff(before, after contract.Resource) []Change {
	var out []Change
	addChange := func(path string, b, a any) {
		if reflect.DeepEqual(b, a) {
			return
		}
		if contract.IsSensitiveName(path) {
			b, a = redactIfSet(b), redactIfSet(a)
		}
		out = append(out, Change{Path: path, Before: b, After: a})
	}
	if before.SchemaVersion != "" {
		addChange("schemaVersion", before.SchemaVersion, after.SchemaVersion)
	}
	for _, k := range unionKeys(before.Properties, after.Properties) {
		addChange(k, before.Properties[k], after.Properties[k])
	}
	for _, k := range unionKeys(before.Secrets, after.Secrets) {
		addChange("secrets."+k, refString(before.Secrets, k), refString(after.Secrets, k))
	}
	if !slices.Equal(before.DependsOn, after.DependsOn) {
		addChange("dependsOn", nilIfEmpty(before.DependsOn), nilIfEmpty(after.DependsOn))
	}
	return out
}

func redactIfSet(v any) any {
	if v == nil {
		return nil
	}
	return contract.Redacted
}

func refString(m map[string]contract.SecretRef, k string) any {
	if r, ok := m[k]; ok {
		return r.String()
	}
	return nil
}

func nilIfEmpty(s []string) any {
	if len(s) == 0 {
		return nil
	}
	return s
}

func unionKeys[V any](a, b map[string]V) []string {
	seen := map[string]bool{}
	for k := range a {
		seen[k] = true
	}
	for k := range b {
		seen[k] = true
	}
	return sortedAddrs(seen)
}
