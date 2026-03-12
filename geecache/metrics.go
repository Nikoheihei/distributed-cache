package geecache

import (
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
	"time"
)

type counter struct {
	v uint64
}

func (c *counter) Inc() {
	atomic.AddUint64(&c.v, 1)
}

func (c *counter) Add(n uint64) {
	atomic.AddUint64(&c.v, n)
}

func (c *counter) Get() uint64 {
	return atomic.LoadUint64(&c.v)
}

type histogram struct {
	bounds    []float64
	counts    []uint64 // cumulative counts per bound
	sumMicros uint64
	count     uint64
}

func newHistogram(bounds []float64) *histogram {
	return &histogram{
		bounds: bounds,
		counts: make([]uint64, len(bounds)),
	}
}

func (h *histogram) Observe(d time.Duration) {
	secs := d.Seconds()
	for i, b := range h.bounds {
		if secs <= b {
			atomic.AddUint64(&h.counts[i], 1)
		}
	}
	atomic.AddUint64(&h.count, 1)
	atomic.AddUint64(&h.sumMicros, uint64(d/time.Microsecond))
}

func (h *histogram) writeProm(sb *strings.Builder, name, help string) {
	sb.WriteString("# HELP ")
	sb.WriteString(name)
	sb.WriteString(" ")
	sb.WriteString(help)
	sb.WriteString("\n")
	sb.WriteString("# TYPE ")
	sb.WriteString(name)
	sb.WriteString(" histogram\n")
	for i, b := range h.bounds {
		sb.WriteString(fmt.Sprintf("%s_bucket{le=\"%g\"} %d\n", name, b, atomic.LoadUint64(&h.counts[i])))
	}
	sb.WriteString(fmt.Sprintf("%s_bucket{le=\"+Inf\"} %d\n", name, atomic.LoadUint64(&h.count)))
	sum := float64(atomic.LoadUint64(&h.sumMicros)) / 1e6
	sb.WriteString(fmt.Sprintf("%s_sum %g\n", name, sum))
	sb.WriteString(fmt.Sprintf("%s_count %d\n", name, atomic.LoadUint64(&h.count)))
}

type metrics struct {
	requests   counter
	hits       counter
	misses     counter
	peerReqs   counter
	peerErrors counter
	loadHist   *histogram
}

var metricsState = &metrics{
	loadHist: newHistogram([]float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1, 2, 5}),
}

func IncRequests()          { metricsState.requests.Inc() }
func IncHits()              { metricsState.hits.Inc() }
func IncMisses()            { metricsState.misses.Inc() }
func IncPeerRequests()      { metricsState.peerReqs.Inc() }
func IncPeerErrors()        { metricsState.peerErrors.Inc() }
func ObserveLoad(d time.Duration) { metricsState.loadHist.Observe(d) }

func MetricsHandler(w http.ResponseWriter, r *http.Request) {
	var sb strings.Builder

	sb.WriteString("# HELP gopherstore_cache_requests_total Total cache requests\n")
	sb.WriteString("# TYPE gopherstore_cache_requests_total counter\n")
	sb.WriteString(fmt.Sprintf("gopherstore_cache_requests_total %d\n", metricsState.requests.Get()))

	sb.WriteString("# HELP gopherstore_cache_hits_total Cache hits\n")
	sb.WriteString("# TYPE gopherstore_cache_hits_total counter\n")
	sb.WriteString(fmt.Sprintf("gopherstore_cache_hits_total %d\n", metricsState.hits.Get()))

	sb.WriteString("# HELP gopherstore_cache_misses_total Cache misses\n")
	sb.WriteString("# TYPE gopherstore_cache_misses_total counter\n")
	sb.WriteString(fmt.Sprintf("gopherstore_cache_misses_total %d\n", metricsState.misses.Get()))

	sb.WriteString("# HELP gopherstore_peer_requests_total Peer requests\n")
	sb.WriteString("# TYPE gopherstore_peer_requests_total counter\n")
	sb.WriteString(fmt.Sprintf("gopherstore_peer_requests_total %d\n", metricsState.peerReqs.Get()))

	sb.WriteString("# HELP gopherstore_peer_errors_total Peer request errors\n")
	sb.WriteString("# TYPE gopherstore_peer_errors_total counter\n")
	sb.WriteString(fmt.Sprintf("gopherstore_peer_errors_total %d\n", metricsState.peerErrors.Get()))

	metricsState.loadHist.writeProm(&sb, "gopherstore_cache_load_seconds", "Cache load duration in seconds")

	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(sb.String()))
}
