// Command fixture is the infra workload conformance fixture: it validates the
// declarative resource topology in krewire.yaml and round-trips local state.
// Plan, apply and drift are not implemented in core (#138) and are reported by
// the harness as exclusions.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/krewire/krewire/pkg/cloud/infra/schema"
	"github.com/krewire/krewire/pkg/cloud/infra/state"
	"github.com/krewire/krewire/pkg/config"
)

type resource struct {
	Compute  *schema.Compute  `json:"compute,omitempty"`
	Database *schema.Database `json:"database,omitempty"`
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func load() (map[string]resource, error) {
	var raw struct {
		Infra struct {
			Resources map[string]any `yaml:"resources"`
		} `yaml:"infra"`
	}
	if err := config.Load("krewire.yaml", &raw); err != nil {
		return nil, err
	}
	if len(raw.Infra.Resources) == 0 {
		return nil, errors.New("krewire.yaml: infra.resources is empty")
	}
	// Resource schemas carry json tags; route through JSON to honour them.
	b, err := json.Marshal(raw.Infra.Resources)
	if err != nil {
		return nil, err
	}
	res := map[string]resource{}
	if err := json.Unmarshal(b, &res); err != nil {
		return nil, err
	}
	names := make([]string, 0, len(res))
	for n := range res {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		r := res[n]
		var v any = r.Compute
		if r.Database != nil {
			v = r.Database
		}
		if v == nil || (r.Compute == nil && r.Database == nil) {
			return nil, fmt.Errorf("resource %s: no supported kind (compute, database)", n)
		}
		if err := schema.Validate(v); err != nil {
			return nil, fmt.Errorf("resource %s: %w", n, err)
		}
	}
	return res, nil
}

func run(args []string) error {
	res, err := load()
	if err != nil {
		return err
	}
	if len(args) == 0 || args[0] == "check" {
		fmt.Println("ok", len(res), "resources")
		return nil
	}
	if args[0] != "build" || len(args) < 2 {
		return fmt.Errorf("usage: fixture check|build <state.json>")
	}
	if err := os.MkdirAll(".krewire", 0o755); err != nil {
		return err
	}
	st := state.New(&state.FileBackend{Path: args[1]})
	if err := st.Lock(state.LockInfo{ID: "conformance", Who: "fixture", Created: time.Now(), Operation: "build"}); err != nil {
		return err
	}
	if err := st.Lock(state.LockInfo{ID: "second"}); err == nil {
		return errors.New("state: second lock was granted while locked")
	}
	snapshot := map[string]any{}
	for n, r := range res {
		snapshot[n] = r
	}
	if err := st.Save(snapshot); err != nil {
		return err
	}
	back, err := st.Load()
	if err != nil {
		return err
	}
	if len(back) != len(res) {
		return fmt.Errorf("state: reloaded %d resources, saved %d", len(back), len(res))
	}
	if err := st.Unlock(); err != nil {
		return err
	}
	fmt.Println("state ok", len(back), "resources")
	return nil
}
