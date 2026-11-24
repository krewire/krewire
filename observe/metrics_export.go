package observe

import (
	"bufio"
	"context"
	"errors"
	"io"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
)

// Bucket is one cumulative histogram bucket.
type Bucket struct {
	UpperBound float64 `json:"le"`
	Count      uint64  `json:"count"`
}

// Sample is one series of a family at snapshot time. Value is the counter or
// gauge value, or the histogram sum.
type Sample struct {
	Labels  map[string]string `json:"labels,omitempty"`
	Value   float64           `json:"value"`
	Count   uint64            `json:"count,omitempty"`
	Buckets []Bucket          `json:"buckets,omitempty"`
}

// Family is a point-in-time copy of one metric family. It is the neutral
// shape exporters translate: its fields map one-to-one onto OpenTelemetry
// Sum (monotonic), Gauge, and explicit-bucket Histogram data points.
type Family struct {
	Desc
	Samples []Sample `json:"samples"`
}

// Exporter pushes snapshots to an external backend (an OTLP bridge, a log
// sink, a test). Exporters are optional: workloads that configure none keep
// the same instrumentation code and pay only for in-memory counters.
type Exporter interface {
	Export(ctx context.Context, families []Family) error
}

// ExporterFunc adapts a function to Exporter.
type ExporterFunc func(ctx context.Context, families []Family) error

// Export calls f.
func (f ExporterFunc) Export(ctx context.Context, families []Family) error { return f(ctx, families) }

// ExportTo snapshots r once and hands the result to every exporter, joining
// their errors. A nil registry exports nothing.
func (r *Registry) ExportTo(ctx context.Context, exporters ...Exporter) error {
	if r == nil {
		return nil
	}
	snap := r.Snapshot()
	var errs []error
	for _, e := range exporters {
		if e != nil {
			errs = append(errs, e.Export(ctx, snap))
		}
	}
	return errors.Join(errs...)
}

// Snapshot returns every family sorted by name, series sorted by labels.
func (r *Registry) Snapshot() []Family {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	hooks := append([]func(){}, r.hooks...)
	r.mu.Unlock()
	for _, h := range hooks {
		h()
	}
	r.mu.Lock()
	fams := make([]*family, 0, len(r.families))
	for _, f := range r.families {
		fams = append(fams, f)
	}
	r.mu.Unlock()
	sort.Slice(fams, func(i, j int) bool { return fams[i].desc.Name < fams[j].desc.Name })

	out := make([]Family, 0, len(fams))
	for _, f := range fams {
		out = append(out, f.snapshot())
	}
	return out
}

func (f *family) snapshot() Family {
	f.mu.Lock()
	defer f.mu.Unlock()
	keys := make([]string, 0, len(f.series))
	for k := range f.series {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	fam := Family{Desc: f.desc, Samples: make([]Sample, 0, len(keys))}
	for _, k := range keys {
		s := f.series[k]
		smp := Sample{Value: s.value, Count: s.count}
		if len(s.labels) > 0 {
			smp.Labels = make(map[string]string, len(s.labels))
			for i, l := range f.desc.Labels {
				smp.Labels[l] = s.labels[i]
			}
		}
		for i, b := range f.desc.Buckets {
			smp.Buckets = append(smp.Buckets, Bucket{UpperBound: b, Count: s.buckets[i]})
		}
		fam.Samples = append(fam.Samples, smp)
	}
	return fam
}

// PrometheusContentType is the text exposition format version 0.0.4.
const PrometheusContentType = "text/plain; version=0.0.4; charset=utf-8"

// WritePrometheus writes r in the Prometheus text exposition format, which
// Prometheus, VictoriaMetrics, Grafana Agent, and the OpenTelemetry Collector
// prometheus receiver all scrape.
func (r *Registry) WritePrometheus(w io.Writer) error {
	bw := bufio.NewWriter(w)
	for _, f := range r.Snapshot() {
		bw.WriteString("# HELP " + f.Name + " " + escapeHelp(f.Help) + "\n")
		bw.WriteString("# TYPE " + f.Name + " " + string(f.Kind) + "\n")
		for _, s := range f.Samples {
			if f.Kind != KindHistogram {
				writeLine(bw, f.Name, f.Labels, s.Labels, "", "", s.Value)
				continue
			}
			for _, b := range s.Buckets {
				writeLine(bw, f.Name+"_bucket", f.Labels, s.Labels, "le", formatFloat(b.UpperBound), float64(b.Count))
			}
			writeLine(bw, f.Name+"_bucket", f.Labels, s.Labels, "le", "+Inf", float64(s.Count))
			writeLine(bw, f.Name+"_sum", f.Labels, s.Labels, "", "", s.Value)
			writeLine(bw, f.Name+"_count", f.Labels, s.Labels, "", "", float64(s.Count))
		}
	}
	return bw.Flush()
}

func writeLine(w *bufio.Writer, name string, order []string, labels map[string]string, extraK, extraV string, v float64) {
	w.WriteString(name)
	var parts []string
	for _, l := range order {
		parts = append(parts, l+`="`+escapeLabel(labels[l])+`"`)
	}
	if extraK != "" {
		parts = append(parts, extraK+`="`+extraV+`"`)
	}
	if len(parts) > 0 {
		w.WriteString("{" + strings.Join(parts, ",") + "}")
	}
	w.WriteString(" " + formatFloat(v) + "\n")
}

func formatFloat(v float64) string {
	switch {
	case math.IsInf(v, 1):
		return "+Inf"
	case math.IsInf(v, -1):
		return "-Inf"
	case math.IsNaN(v):
		return "NaN"
	}
	return strconv.FormatFloat(v, 'g', -1, 64)
}

var (
	labelEscaper = strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", `\n`)
	helpEscaper  = strings.NewReplacer(`\`, `\\`, "\n", `\n`)
)

func escapeLabel(s string) string { return labelEscaper.Replace(s) }
func escapeHelp(s string) string  { return helpEscaper.Replace(s) }

// MetricsHandler serves r in Prometheus text format (conventionally at
// /metrics). A nil registry serves 404 so a disabled exporter is visible.
func MetricsHandler(r *Registry) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if r == nil {
			http.NotFound(w, req)
			return
		}
		w.Header().Set("Content-Type", PrometheusContentType)
		_ = r.WritePrometheus(w)
	})
}
