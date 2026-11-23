// Command qualify runs the workload conformance harness and writes the
// machine-readable evidence (report.json) and a Markdown summary.
//
//	go run ./internal/conformance/cmd/qualify [-out dir] [-workloads site,book]
//	    [-strict] [-update-baseline]
//
// Exit status: 0 no regression (and, with -strict, workload qualification
// verified); 1 regression or strict failure; 2 usage or environment error.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"time"

	"github.com/krewire/krewire/internal/conformance"
)

func main() { os.Exit(run()) }

func run() int {
	out := flag.String("out", ".krewire/conformance", "directory for report.json and summary.md")
	only := flag.String("workloads", "", "comma-separated workloads (default: all eight)")
	baseline := flag.String("baseline", "", "baseline file (default: internal/conformance/baseline.json)")
	update := flag.Bool("update-baseline", false, "rewrite the baseline from this run (full runs only)")
	strict := flag.Bool("strict", false, "also fail unless every workload gate passed (release qualification)")
	parallel := flag.Int("parallel", 0, "concurrent fixtures (default min(4, NumCPU))")
	timeout := flag.Duration("timeout", 5*time.Minute, "per-workload timeout")
	flag.Parse()

	env, err := conformance.Discover(".")
	if err != nil {
		fmt.Fprintln(os.Stderr, "qualify:", err)
		return 2
	}
	if *baseline == "" {
		*baseline = filepath.Join(env.Root, "internal", "conformance", "baseline.json")
	}
	opts := conformance.Options{Parallel: *parallel, Timeout: *timeout}
	if *only != "" {
		opts.Workloads = strings.Split(*only, ",")
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	rep, err := conformance.Run(ctx, env, opts)
	if err != nil {
		fmt.Fprintln(os.Stderr, "qualify:", err)
		return 2
	}
	if *update {
		if len(opts.Workloads) != 0 {
			fmt.Fprintln(os.Stderr, "qualify: -update-baseline needs a full run")
			return 2
		}
		if err := conformance.WriteBaseline(*baseline, conformance.BaselineFrom(rep)); err != nil {
			fmt.Fprintln(os.Stderr, "qualify:", err)
			return 2
		}
	}
	base, err := conformance.ReadBaseline(*baseline)
	if err != nil {
		fmt.Fprintln(os.Stderr, "qualify:", err)
		return 2
	}
	regressions, improvements := conformance.Compare(rep, base)
	summary := conformance.Summary(rep, regressions, improvements)
	if err := os.MkdirAll(*out, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, "qualify:", err)
		return 2
	}
	if err := rep.WriteJSON(filepath.Join(*out, "report.json")); err != nil {
		fmt.Fprintln(os.Stderr, "qualify:", err)
		return 2
	}
	if err := os.WriteFile(filepath.Join(*out, "summary.md"), []byte(summary), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "qualify:", err)
		return 2
	}
	fmt.Print(summary)
	if len(regressions) > 0 || (*strict && !rep.WorkloadQualification.Verified) {
		return 1
	}
	return 0
}
