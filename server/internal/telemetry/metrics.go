// Package telemetry exposes bounded, process-wide metrics without player labels.
package telemetry

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type Number struct{ value atomic.Int64 }

func (n *Number) Add(v int64)  { n.value.Add(v) }
func (n *Number) Set(v int64)  { n.value.Store(v) }
func (n *Number) Value() int64 { return n.value.Load() }

type Histogram struct {
	count   atomic.Uint64
	nanos   atomic.Uint64
	buckets [8]atomic.Uint64
}

var limits = [8]time.Duration{time.Millisecond, 5 * time.Millisecond, 10 * time.Millisecond, 50 * time.Millisecond, 100 * time.Millisecond, 500 * time.Millisecond, time.Second, 5 * time.Second}

func (h *Histogram) Observe(d time.Duration) {
	if d < 0 {
		return
	}
	h.count.Add(1)
	h.nanos.Add(uint64(d))
	for i, limit := range limits {
		if d <= limit {
			h.buckets[i].Add(1)
		}
	}
}

var registry = struct {
	sync.Mutex
	numbers    map[string]*Number
	histograms map[string]*Histogram
}{numbers: map[string]*Number{}, histograms: map[string]*Histogram{}}

func valid(name string) {
	if len(name) == 0 || len(name) > 100 {
		panic("invalid metric name")
	}
	for _, r := range name {
		if !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '_') {
			panic("invalid metric name")
		}
	}
}
func NumberMetric(name string) *Number {
	valid(name)
	registry.Lock()
	defer registry.Unlock()
	if n := registry.numbers[name]; n != nil {
		return n
	}
	if len(registry.numbers)+len(registry.histograms) >= 128 {
		panic("metric budget exceeded")
	}
	n := &Number{}
	registry.numbers[name] = n
	return n
}
func DurationMetric(name string) *Histogram {
	valid(name)
	registry.Lock()
	defer registry.Unlock()
	if h := registry.histograms[name]; h != nil {
		return h
	}
	if len(registry.numbers)+len(registry.histograms) >= 128 {
		panic("metric budget exceeded")
	}
	h := &Histogram{}
	registry.histograms[name] = h
	return h
}

// WritePrometheus writes the standard text exposition format. Names come only
// from fixed code registrations; user IDs and source addresses are never labels.
func WritePrometheus(w io.Writer) {
	registry.Lock()
	defer registry.Unlock()
	names := make([]string, 0, len(registry.numbers)+len(registry.histograms))
	for name := range registry.numbers {
		names = append(names, name)
	}
	for name := range registry.histograms {
		names = append(names, name)
	}
	sort.Strings(names)
	var b strings.Builder
	for _, name := range names {
		if n := registry.numbers[name]; n != nil {
			fmt.Fprintf(&b, "%s %d\n", name, n.Value())
			continue
		}
		h := registry.histograms[name]
		fmt.Fprintf(&b, "# TYPE %s histogram\n", name)
		for i, limit := range limits {
			fmt.Fprintf(&b, "%s_bucket{le=\"%g\"} %d\n", name, limit.Seconds(), h.buckets[i].Load())
		}
		fmt.Fprintf(&b, "%s_bucket{le=\"+Inf\"} %d\n%s_sum %g\n%s_count %d\n", name, h.count.Load(), name, float64(h.nanos.Load())/float64(time.Second), name, h.count.Load())
	}
	io.WriteString(w, b.String())
}
