package loadtest

import (
	"math"
	"sync/atomic"
	"time"
)

// Bucket latency dalam mikrodetik dengan resolusi bertingkat:
//
//	0–999µs      : 1µs
//	1ms–99.9ms   : 100µs
//	100ms–10s    : 10ms
//	>10s         : bucket terakhir
const (
	tier1      = 1000
	tier2      = 990
	tier3      = 990
	numBuckets = tier1 + tier2 + tier3 + 1
)

// Histogram lock-free: ribuan goroutine worker bisa Record bersamaan tanpa mutex.
type Histogram struct {
	buckets [numBuckets]atomic.Uint64
	sumUs   atomic.Uint64
	maxUs   atomic.Uint64
}

func NewHistogram() *Histogram { return &Histogram{} }

func bucketIndex(us uint64) int {
	switch {
	case us < 1_000:
		return int(us)
	case us < 100_000:
		return tier1 + int((us-1_000)/100)
	case us < 10_000_000:
		return tier1 + tier2 + int((us-100_000)/10_000)
	default:
		return numBuckets - 1
	}
}

// bucketValue mengembalikan nilai tengah bucket (µs).
func bucketValue(i int) uint64 {
	switch {
	case i < tier1:
		return uint64(i)
	case i < tier1+tier2:
		return 1_000 + uint64(i-tier1)*100 + 50
	case i < tier1+tier2+tier3:
		return 100_000 + uint64(i-tier1-tier2)*10_000 + 5_000
	default:
		return 10_000_000
	}
}

func (h *Histogram) Record(d time.Duration) {
	us := uint64(d.Microseconds())
	h.buckets[bucketIndex(us)].Add(1)
	h.sumUs.Add(us)
	for {
		cur := h.maxUs.Load()
		if us <= cur || h.maxUs.CompareAndSwap(cur, us) {
			return
		}
	}
}

type LatencyStats struct {
	Count  uint64  `json:"count"`
	MeanMs float64 `json:"mean_ms"`
	P50Ms  float64 `json:"p50_ms"`
	P90Ms  float64 `json:"p90_ms"`
	P95Ms  float64 `json:"p95_ms"`
	P99Ms  float64 `json:"p99_ms"`
	MaxMs  float64 `json:"max_ms"`
}

func (h *Histogram) Snapshot() LatencyStats {
	var counts [numBuckets]uint64
	var total uint64
	for i := range h.buckets {
		counts[i] = h.buckets[i].Load()
		total += counts[i]
	}
	s := LatencyStats{Count: total}
	if total == 0 {
		return s
	}
	maxMs := float64(h.maxUs.Load()) / 1000
	s.MeanMs = float64(h.sumUs.Load()) / float64(total) / 1000
	s.MaxMs = maxMs
	s.P50Ms = min(percentile(&counts, total, 0.50), maxMs)
	s.P90Ms = min(percentile(&counts, total, 0.90), maxMs)
	s.P95Ms = min(percentile(&counts, total, 0.95), maxMs)
	s.P99Ms = min(percentile(&counts, total, 0.99), maxMs)
	return s
}

func percentile(counts *[numBuckets]uint64, total uint64, q float64) float64 {
	rank := uint64(math.Ceil(q * float64(total)))
	var cum uint64
	for i, c := range counts {
		cum += c
		if cum >= rank {
			return float64(bucketValue(i)) / 1000
		}
	}
	return float64(bucketValue(numBuckets-1)) / 1000
}
