package engine

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	"github.com/krewire/krewire/cloud/infra/contract"
	"github.com/krewire/krewire/cloud/infra/store"
	"github.com/krewire/krewire/errs"
)

// DriftStatus classifies one recorded resource against actual state.
type DriftStatus string

// Drift statuses.
const (
	DriftInSync     DriftStatus = "in-sync"
	DriftModified   DriftStatus = "modified"
	DriftMissing    DriftStatus = "missing"
	DriftPending    DriftStatus = "pending"
	DriftUnreadable DriftStatus = "unreadable"
)

// DriftEntry is the inspection result for one recorded resource. Changes
// hold recorded values in Before and observed values in After.
type DriftEntry struct {
	Address string      `json:"address"`
	Status  DriftStatus `json:"status"`
	Changes []Change    `json:"changes,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// DriftReport is the result of Inspect. It is never a validation error:
// configuration problems are reported by Plan as *ValidationError.
type DriftReport struct {
	Environment string       `json:"environment"`
	Serial      uint64       `json:"serial"`
	Drifted     bool         `json:"drifted"`
	Entries     []DriftEntry `json:"entries"`
	Digest      string       `json:"digest"`
}

func (d *DriftReport) entry(addr string) (DriftEntry, bool) {
	if d == nil {
		return DriftEntry{}, false
	}
	for _, e := range d.Entries {
		if e.Address == addr {
			return e, true
		}
	}
	return DriftEntry{}, false
}

// ErrStaleDrift reports a drift report computed against older state.
var ErrStaleDrift = errors.New("infra: drift report is stale")

// Inspect refreshes actual state through provider reads and compares it with
// recorded state. It never locks or writes state and never mutates objects.
func (e *Engine) Inspect(ctx context.Context) (*DriftReport, error) {
	snap, err := e.Store.Read(ctx)
	if err != nil {
		return nil, err
	}
	return e.inspect(ctx, snap)
}

func (e *Engine) inspect(ctx context.Context, snap contract.Snapshot) (*DriftReport, error) {
	rep := &DriftReport{Environment: e.Environment, Serial: snap.Serial, Entries: []DriftEntry{}}
	for _, rs := range snap.Resources {
		ent := DriftEntry{Address: rs.Address(), Status: DriftInSync}
		obs, err := e.Provider.Read(ctx, rs)
		switch {
		case rs.Status == contract.StatusPending:
			ent.Status = DriftPending
		case errors.Is(err, contract.ErrNotFound):
			ent.Status = DriftMissing
		case err != nil:
			ent.Status, ent.Error = DriftUnreadable, err.Error()
		default:
			observed, cerr := canonicalMap(obs.Properties)
			if cerr != nil {
				return nil, cerr
			}
			for _, k := range sortedAddrs(rs.Properties) {
				before, after := rs.Properties[k], observed[k]
				if reflect.DeepEqual(before, after) {
					continue
				}
				if contract.IsSensitiveName(k) {
					before, after = redactIfSet(before), redactIfSet(after)
				}
				ent.Changes = append(ent.Changes, Change{Path: k, Before: before, After: after})
			}
			if len(ent.Changes) > 0 {
				ent.Status = DriftModified
			}
		}
		rep.Drifted = rep.Drifted || ent.Status != DriftInSync
		rep.Entries = append(rep.Entries, ent)
	}
	data, err := json.Marshal(rep)
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256(data)
	rep.Digest = "sha256:" + hex.EncodeToString(sum[:])
	return rep, nil
}

// observedProps rebuilds the observed property map from recorded properties
// and drift changes.
func observedProps(recorded map[string]any, changes []Change) map[string]any {
	out := make(map[string]any, len(recorded))
	for k, v := range recorded {
		out[k] = v
	}
	for _, c := range changes {
		if c.After == nil {
			delete(out, c.Path)
		} else {
			out[c.Path] = c.After
		}
	}
	return out
}

// PlanReconcile plans the explicit reconciliation of drift back to desired
// state: missing objects are recreated and modified objects are updated (or
// replaced) using the observed values as the baseline. Apply re-inspects and
// refuses to run if actual state moved again.
func (e *Engine) PlanReconcile(ctx context.Context, desired []contract.Resource, drift *DriftReport) (*Plan, error) {
	snap, err := e.Store.Read(ctx)
	if err != nil {
		return nil, err
	}
	if drift == nil || drift.Serial != snap.Serial || drift.Environment != e.Environment {
		return nil, errs.WithHint(ErrStaleDrift, "run inspect again before planning reconciliation")
	}
	return e.planFrom(snap, ModeReconcile, desired, drift)
}

// AcceptDrift is the other explicit reconciliation: it records actual state
// as the new recorded state. Modified properties take their observed values
// and missing objects leave state. Pending and unreadable entries are left
// for apply to resume. The next Plan then shows what desired state requires.
func (e *Engine) AcceptDrift(ctx context.Context, drift *DriftReport, opts ApplyOptions) (uint64, error) {
	lock, err := e.Store.Lock(ctx, store.LockInfo{Who: e.Principal.ID, Operation: "accept-drift", Created: e.now()})
	if err != nil {
		return 0, err
	}
	defer lock.Unlock()
	snap, err := e.Store.Read(ctx)
	if err != nil {
		return 0, err
	}
	again, err := e.inspect(ctx, snap)
	if err != nil {
		return 0, err
	}
	if drift == nil || again.Digest != drift.Digest {
		return 0, errs.WithHint(ErrStaleDrift, "actual or recorded state changed; inspect again and review the new report")
	}
	if e.Principal.Agent && drift.Drifted {
		pseudo := &Plan{Environment: e.Environment, Mode: "accept-drift", Digest: drift.Digest,
			Actions: []Action{{Op: OpUpdate, Risk: RiskHigh}}}
		if err := e.authorize(pseudo, opts.Approval); err != nil {
			return 0, err
		}
	}
	if !drift.Drifted {
		return snap.Serial, nil
	}
	next := snap.Clone()
	for _, ent := range drift.Entries {
		switch ent.Status {
		case DriftMissing:
			next.Remove(ent.Address)
		case DriftModified:
			rs, _ := next.Find(ent.Address)
			rs.Properties = observedProps(rs.Properties, ent.Changes)
			next.Put(rs)
		}
	}
	next.Serial = snap.Serial + 1
	written, err := e.Store.Write(ctx, lock, next)
	if err != nil {
		return 0, fmt.Errorf("record state: %w", err)
	}
	return written.Serial, nil
}
