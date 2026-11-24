package observe

import (
	"fmt"
	"regexp"
	"slices"
	"sort"
	"strings"
	"sync"
)

// Kind is the metric instrument type.
type Kind string

const (
	KindCounter   Kind = "counter"
	KindGauge     Kind = "gauge"
	KindHistogram Kind = "histogram"
)

// Unit is the UCUM-style unit a metric is measured in. Names carry the unit
// as a suffix (Prometheus convention) and the Desc records it explicitly for
// OpenTelemetry adapters.
type Unit string

const (
	UnitSeconds Unit = "s"
	UnitBytes   Unit = "By"
	UnitCount   Unit = "1"
)

// Desc describes one metric family. Labels are the only dimensions allowed;
// every label must have a bounded value set (route patterns, outcomes, queue
// names), never IDs, paths with parameters, or user input.
type Desc struct {
	Name    string    `json:"name"`
	Help    string    `json:"help"`
	Kind    Kind      `json:"kind"`
	Unit    Unit      `json:"unit"`
	Labels  []string  `json:"labels,omitempty"`
	Buckets []float64 `json:"buckets,omitempty"`
}

// DefaultMaxSeries bounds the label combinations per family. New combinations
// past the bound are folded into one series whose labels are all OverflowValue.
const DefaultMaxSeries = 200

// OverflowValue labels the series that absorbs over-cardinality samples.
const OverflowValue = "_overflow"

// DefaultBuckets are latency buckets in seconds, 5ms to 10s.
var DefaultBuckets = []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10}

var nameRe = regexp.MustCompile(`^[a-z_][a-z0-9_]*$`)

// Registry owns metric families. A nil *Registry is a disabled registry: it
// hands out nil instruments whose methods are no-ops.
type Registry struct {
	mu        sync.Mutex
	families  map[string]*family
	maxSeries int
	hooks     []func()
}

// NewRegistry returns an empty registry with DefaultMaxSeries.
func NewRegistry() *Registry {
	return &Registry{families: map[string]*family{}, maxSeries: DefaultMaxSeries}
}

// SetMaxSeries sets the cardinality bound (minimum 1) for families registered
// after the call; call it before registering instruments.
func (r *Registry) SetMaxSeries(n int) {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.maxSeries = max(n, 1)
}

// OnCollect registers fn to run before every exposition or snapshot, used to
// refresh gauges such as runtime statistics.
func (r *Registry) OnCollect(fn func()) {
	if r == nil || fn == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.hooks = append(r.hooks, fn)
}

type series struct {
	labels  []string
	value   float64
	count   uint64
	buckets []uint64
}

type family struct {
	desc   Desc
	mu     sync.Mutex
	series map[string]*series
	max    int
}

// register validates d and returns its family. Registering an identical Desc
// twice returns the existing family; a conflicting one panics, as it is a
// programming error that would corrupt the exposition.
func (r *Registry) register(d Desc) *family {
	if !nameRe.MatchString(d.Name) {
		panic(fmt.Sprintf("observe: invalid metric name %q", d.Name))
	}
	if d.Kind == KindCounter && !strings.HasSuffix(d.Name, "_total") {
		panic(fmt.Sprintf("observe: counter %q must end in _total", d.Name))
	}
	for _, l := range d.Labels {
		if !nameRe.MatchString(l) || strings.HasPrefix(l, "__") || l == "le" {
			panic(fmt.Sprintf("observe: invalid label %q on %q", l, d.Name))
		}
	}
	if d.Kind == KindHistogram {
		if len(d.Buckets) == 0 {
			d.Buckets = DefaultBuckets
		}
		d.Buckets = append([]float64(nil), d.Buckets...)
		sort.Float64s(d.Buckets)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if f, ok := r.families[d.Name]; ok {
		if !sameDesc(f.desc, d) {
			panic(fmt.Sprintf("observe: metric %q re-registered with a different description", d.Name))
		}
		return f
	}
	f := &family{desc: d, series: map[string]*series{}, max: r.maxSeries}
	r.families[d.Name] = f
	return f
}

func sameDesc(a, b Desc) bool {
	return a.Name == b.Name && a.Help == b.Help && a.Kind == b.Kind && a.Unit == b.Unit &&
		slices.Equal(a.Labels, b.Labels) && slices.Equal(a.Buckets, b.Buckets)
}

// get returns the series for values, folding malformed or over-bound label
// sets into the overflow series. Callers hold f.mu.
func (f *family) get(values []string) *series {
	if len(values) != len(f.desc.Labels) {
		values = overflowLabels(len(f.desc.Labels))
	}
	key := strings.Join(values, "\xff")
	if s, ok := f.series[key]; ok {
		return s
	}
	if len(f.series) >= f.max {
		values = overflowLabels(len(f.desc.Labels))
		key = strings.Join(values, "\xff")
		if s, ok := f.series[key]; ok {
			return s
		}
	}
	s := &series{labels: append([]string(nil), values...)}
	if f.desc.Kind == KindHistogram {
		s.buckets = make([]uint64, len(f.desc.Buckets))
	}
	f.series[key] = s
	return s
}

func overflowLabels(n int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = OverflowValue
	}
	return out
}

// Counter is a monotonically increasing value.
type Counter struct{ f *family }

// Gauge is a value that can go up and down.
type Gauge struct{ f *family }

// Histogram samples observations into cumulative buckets.
type Histogram struct{ f *family }

// Counter registers (or returns) a counter. On a nil Registry it returns nil.
func (r *Registry) Counter(name, help string, unit Unit, labels ...string) *Counter {
	if r == nil {
		return nil
	}
	return &Counter{r.register(Desc{Name: name, Help: help, Kind: KindCounter, Unit: unit, Labels: labels})}
}

// Gauge registers (or returns) a gauge. On a nil Registry it returns nil.
func (r *Registry) Gauge(name, help string, unit Unit, labels ...string) *Gauge {
	if r == nil {
		return nil
	}
	return &Gauge{r.register(Desc{Name: name, Help: help, Kind: KindGauge, Unit: unit, Labels: labels})}
}

// Histogram registers (or returns) a histogram; nil buckets use
// DefaultBuckets. On a nil Registry it returns nil.
func (r *Registry) Histogram(name, help string, unit Unit, buckets []float64, labels ...string) *Histogram {
	if r == nil {
		return nil
	}
	return &Histogram{r.register(Desc{Name: name, Help: help, Kind: KindHistogram, Unit: unit, Labels: labels, Buckets: buckets})}
}

// Register registers d with the instrument its Kind names and returns it as
// one of *Counter, *Gauge, or *Histogram (nil on a nil Registry).
func (r *Registry) Register(d Desc) any {
	switch d.Kind {
	case KindCounter:
		return r.Counter(d.Name, d.Help, d.Unit, d.Labels...)
	case KindGauge:
		return r.Gauge(d.Name, d.Help, d.Unit, d.Labels...)
	case KindHistogram:
		return r.Histogram(d.Name, d.Help, d.Unit, d.Buckets, d.Labels...)
	}
	panic(fmt.Sprintf("observe: unknown metric kind %q", d.Kind))
}

// Inc adds 1.
func (c *Counter) Inc(labels ...string) { c.Add(1, labels...) }

// Add adds v; negative values are ignored to keep the counter monotonic.
func (c *Counter) Add(v float64, labels ...string) {
	if c == nil || v < 0 {
		return
	}
	c.f.mu.Lock()
	c.f.get(labels).value += v
	c.f.mu.Unlock()
}

// Set replaces the gauge value.
func (g *Gauge) Set(v float64, labels ...string) {
	if g == nil {
		return
	}
	g.f.mu.Lock()
	g.f.get(labels).value = v
	g.f.mu.Unlock()
}

// Add adds v (which may be negative) to the gauge.
func (g *Gauge) Add(v float64, labels ...string) {
	if g == nil {
		return
	}
	g.f.mu.Lock()
	g.f.get(labels).value += v
	g.f.mu.Unlock()
}

// Observe records one sample.
func (h *Histogram) Observe(v float64, labels ...string) {
	if h == nil {
		return
	}
	h.f.mu.Lock()
	s := h.f.get(labels)
	s.count++
	s.value += v
	for i, b := range h.f.desc.Buckets {
		if v <= b {
			s.buckets[i]++
		}
	}
	h.f.mu.Unlock()
}
