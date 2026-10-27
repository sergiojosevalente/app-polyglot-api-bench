// Package collector mengambil /metrics tiap service tiap detik dan menghitung CPU%.
package collector

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/gorust/monitor/internal/target"
)

// Sample: field ber-tag json diisi langsung dari response /metrics service.
type Sample struct {
	Up            bool           `json:"up"`
	Version       string         `json:"version"`
	UptimeSec     float64        `json:"uptime_sec"`
	RequestsTotal uint64         `json:"requests_total"`
	RSSBytes      uint64         `json:"rss_bytes"`
	CPUSeconds    float64        `json:"cpu_seconds"`
	Threads       int            `json:"threads"`
	Items         int            `json:"items"`
	Runtime       map[string]any `json:"runtime"`
	CPUPercent    float64        `json:"cpu_percent"` // dihitung monitor
	ScrapeMs      float64        `json:"scrape_ms"`   // waktu respons /metrics = responsivitas
}

type prevSample struct {
	cpu    float64
	uptime float64
	at     time.Time
}

type Collector struct {
	targets []target.Target
	client  *http.Client

	mu   sync.Mutex
	prev map[string]prevSample
}

func New(targets []target.Target) *Collector {
	return &Collector{
		targets: targets,
		// Transport terpisah dari load generator supaya tidak berebut connection pool.
		client: &http.Client{Timeout: 2 * time.Second, Transport: &http.Transport{MaxIdleConnsPerHost: 4}},
		prev:   make(map[string]prevSample),
	}
}

// Sample men-scrape semua target secara paralel (fan-out goroutine + WaitGroup).
func (c *Collector) Sample(ctx context.Context) map[string]Sample {
	out := make(map[string]Sample, len(c.targets))
	var mu sync.Mutex
	var wg sync.WaitGroup
	for _, t := range c.targets {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s := c.scrape(ctx, t)
			mu.Lock()
			out[t.Name] = s
			mu.Unlock()
		}()
	}
	wg.Wait()
	return out
}

func (c *Collector) scrape(ctx context.Context, t target.Target) Sample {
	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, t.URL+"/metrics", nil)
	if err != nil {
		return Sample{}
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return Sample{}
	}
	defer resp.Body.Close()

	var s Sample
	if resp.StatusCode != http.StatusOK || json.NewDecoder(resp.Body).Decode(&s) != nil {
		return Sample{}
	}
	now := time.Now()
	s.Up = true
	s.ScrapeMs = float64(now.Sub(start).Microseconds()) / 1000

	c.mu.Lock()
	defer c.mu.Unlock()
	// uptime turun = service restart, counter CPU ter-reset.
	if p, ok := c.prev[t.Name]; ok && s.CPUSeconds >= p.cpu && s.UptimeSec >= p.uptime {
		if dt := now.Sub(p.at).Seconds(); dt > 0 {
			s.CPUPercent = (s.CPUSeconds - p.cpu) / dt * 100
		}
	}
	c.prev[t.Name] = prevSample{cpu: s.CPUSeconds, uptime: s.UptimeSec, at: now}
	return s
}
