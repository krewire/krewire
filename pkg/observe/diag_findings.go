package observe

import "fmt"

// runbook is the operational guide section each finding points to.
const runbook = "docs/guides/operational-visibility-and-recovery.md"

var probeHints = map[Probe]string{
	ProbeStartup: "Initialization has not completed. Filter startup logs by trace_id, then check configuration and the dependencies bound to the startup probe. See " +
		runbook + "#diagnosing-startup-failure.",
	ProbeReadiness: "The instance is removed from routing but not restarted. Check the failing dependency named in the code and its retry/breaker metrics. See " +
		runbook + "#diagnosing-readiness-failure.",
	ProbeLiveness: "The orchestrator will restart this process. Look for deadlocks, a stalled heartbeat loop, or resource exhaustion in runtime metrics. See " +
		runbook + "#diagnosing-readiness-failure.",
}

// probeFindings turns a probe report into findings. A "starting" reason is
// reported once, by the startup probe.
func probeFindings(rep Report, traceID string) []Finding {
	var out []Finding
	switch {
	case rep.Reason == "starting" && rep.Probe == ProbeStartup:
		out = append(out, Finding{Code: "health.startup.incomplete", Severity: SeverityError, Origin: OriginRuntime,
			Class: FailureWorkload, Message: "startup has not been marked complete", Hint: probeHints[ProbeStartup], TraceID: traceID})
	case rep.Reason == "draining":
		out = append(out, Finding{Code: "health.readiness.draining", Severity: SeverityWarn, Origin: OriginRuntime,
			Message: "instance is draining for shutdown", Hint: "Expected during shutdown. If it persists, the drain exceeded its timeout; see " +
				runbook + "#diagnosing-shutdown.", TraceID: traceID})
	}
	for _, c := range rep.Checks {
		if c.Status == StatusPass {
			continue
		}
		sev := SeverityError
		if c.Status == StatusWarn {
			sev = SeverityWarn
		}
		out = append(out, Finding{Code: fmt.Sprintf("health.%s.%s", rep.Probe, c.Name), Severity: sev, Origin: OriginRuntime,
			Class: FailureWorkload, Message: c.Name + ": " + c.Error, Hint: probeHints[rep.Probe], TraceID: traceID})
	}
	return out
}

// metricFindings flags conditions visible only in metrics.
func metricFindings(fams []Family) []Finding {
	var out []Finding
	for _, f := range fams {
		for _, s := range f.Samples {
			for _, v := range s.Labels {
				if v == OverflowValue {
					out = append(out, Finding{Code: "metrics.cardinality_overflow", Severity: SeverityWarn, Origin: OriginRuntime,
						Message: f.Name + " exceeded its series bound", Hint: "A label carries unbounded values (IDs, raw paths, user input); replace it with a bounded label."})
					break
				}
			}
			switch {
			case f.Name == MetricJobs && s.Labels["outcome"] == OutcomeDeadLetter && s.Value > 0:
				out = append(out, Finding{Code: "worker.dead_letters", Severity: SeverityWarn, Origin: OriginRuntime, Class: FailureWorkload,
					Message: fmt.Sprintf("queue %s dead-lettered %g jobs", s.Labels["queue"], s.Value),
					Hint:    "Inspect the DLQ and the job's retry policy. See " + runbook + "#diagnosing-retries."})
			case f.Name == MetricRetries && s.Value > 0:
				out = append(out, Finding{Code: "resilience.retries", Severity: SeverityInfo, Origin: OriginRuntime,
					Message: fmt.Sprintf("operation %s retried %g times", s.Labels["operation"], s.Value),
					Hint:    "Retries are expected in moderation; correlate with dependency latency. See " + runbook + "#diagnosing-retries."})
			case f.Name == MetricBreakerState && s.Value >= 2:
				out = append(out, Finding{Code: "resilience.breaker_open", Severity: SeverityWarn, Origin: OriginRuntime, Class: FailureProvider,
					Message: "circuit breaker " + s.Labels["breaker"] + " is open",
					Hint:    "Calls to the protected dependency are short-circuited. See " + runbook + "#diagnosing-retries."})
			}
		}
	}
	return out
}
