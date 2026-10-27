package loadtest

import (
	"bytes"
	"context"
	"io"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorust/monitor/internal/target"
)

// WindowStats adalah statistik satu jendela 1 detik (untuk grafik live).
type WindowStats struct {
	RPS       float64      `json:"rps"`
	Errors    uint64       `json:"errors"`
	ErrorRate float64      `json:"error_rate"`
	VUs       int          `json:"vus"`
	Latency   LatencyStats `json:"latency"`
}

// ResourceSample adalah data resource service yang diambil collector tiap detik.
type ResourceSample struct {
	CPUPercent float64
	RSSBytes   uint64
	CPUSeconds float64
}

type TargetSummary struct {
	Target       string       `json:"target"`
	Requests     uint64       `json:"requests"`
	Errors       uint64       `json:"errors"`
	ErrorRate    float64      `json:"error_rate"`
	AvgRPS       float64      `json:"avg_rps"`
	PeakRPS      float64      `json:"peak_rps"`
	Latency      LatencyStats `json:"latency"`
	AvgCPU       float64      `json:"avg_cpu"`
	PeakCPU      float64      `json:"peak_cpu"`
	PeakRSSMB    float64      `json:"peak_rss_mb"`
	CPUSeconds   float64      `json:"cpu_seconds"`
	ReqPerCPUSec float64      `json:"req_per_cpu_sec"`
}

type Summary struct {
	ID          string          `json:"id"`
	Config      Config          `json:"config"`
	StartedAt   time.Time       `json:"started_at"`
	DurationSec float64         `json:"duration_sec"`
	Stopped     bool            `json:"stopped"`
	Targets     []TargetSummary `json:"targets"`
}

type targetRun struct {
	t           target.Target
	client      *http.Client
	window      atomic.Pointer[Histogram] // jendela 1 detik saat ini, di-swap tiap tick
	winErrors   atomic.Uint64
	total       *Histogram
	totalErrors atomic.Uint64

	// di bawah ini dijaga run.mu, diperbarui sekali per tick
	peakRPS, peakCPU, cpuSum float64
	cpuN                     int
	peakRSS                  uint64
	haveBaseline             bool
	baseCPUSec, lastCPUSec   float64
	reqSinceBaseline         uint64
	reqAtLastCPU             uint64 // request s/d sampel CPU terakhir yang berhasil
}

func (tr *targetRun) record(lat time.Duration, failed bool) {
	tr.window.Load().Record(lat)
	tr.total.Record(lat)
	if failed {
		tr.winErrors.Add(1)
		tr.totalErrors.Add(1)
	}
}

type run struct {
	id      string
	cfg     Config
	started time.Time
	ended   time.Time
	ctx     context.Context
	cancel  context.CancelFunc
	targets []*targetRun
	desired atomic.Int64 // jumlah virtual user aktif saat ini
	stopped atomic.Bool

	mu       sync.Mutex
	lastTick time.Time
}

func newRun(id string, cfg Config, targets []target.Target) *run {
	ctx, cancel := context.WithCancel(context.Background())
	r := &run{id: id, cfg: cfg, started: time.Now(), ctx: ctx, cancel: cancel}
	r.lastTick = r.started
	for _, t := range targets {
		tr := &targetRun{t: t, client: newClient(cfg.Concurrency), total: NewHistogram()}
		tr.window.Store(NewHistogram())
		r.targets = append(r.targets, tr)
	}
	return r
}

// newClient: connection pool sebesar jumlah VU. Default MaxIdleConnsPerHost=2
// akan membuat koneksi baru terus-menerus (port exhaustion) saat load tinggi.
func newClient(conc int) *http.Client {
	return &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			DialContext:         (&net.Dialer{Timeout: 5 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
			MaxIdleConns:        conc * 2,
			MaxIdleConnsPerHost: conc * 2,
			IdleConnTimeout:     30 * time.Second,
			DisableCompression:  true,
		},
	}
}

// profile menentukan jumlah VU aktif berdasarkan waktu berjalan.
func (r *run) profile(elapsed time.Duration) int {
	c := r.cfg.Concurrency
	frac := elapsed.Seconds() / float64(r.cfg.DurationSec)
	switch r.cfg.Mode {
	case "stress": // 10 anak tangga: 10%, 20%, ... 100%
		step := min(int(frac*10), 9)
		return max(1, c*(step+1)/10)
	case "spike": // baseline 10%, lonjakan 100% di 40–60% durasi
		if frac >= 0.4 && frac < 0.6 {
			return c
		}
		return max(1, c/10)
	default:
		return c
	}
}

func (r *run) execute() {
	r.desired.Store(int64(r.profile(0)))

	var wg sync.WaitGroup
	for _, tr := range r.targets {
		for i := 0; i < r.cfg.Concurrency; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				r.worker(tr, i)
			}()
		}
	}

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	deadline := time.NewTimer(time.Duration(r.cfg.DurationSec) * time.Second)
	defer deadline.Stop()
loop:
	for {
		select {
		case <-r.ctx.Done():
			break loop
		case <-deadline.C:
			break loop
		case <-ticker.C:
			r.desired.Store(int64(r.profile(time.Since(r.started))))
		}
	}
	r.cancel()
	wg.Wait()
	r.ended = time.Now()
	for _, tr := range r.targets {
		tr.client.CloseIdleConnections()
	}
}

// worker adalah satu virtual user: kirim request, tunggu response, ulangi (closed-loop).
func (r *run) worker(tr *targetRun, id int) {
	idle := time.NewTimer(0)
	defer idle.Stop()
	for r.ctx.Err() == nil {
		if int64(id) >= r.desired.Load() {
			idle.Reset(20 * time.Millisecond)
			select {
			case <-r.ctx.Done():
				return
			case <-idle.C:
			}
			continue
		}

		spec := nextRequest(r.cfg.Scenario, r.cfg.CPUN)
		var body io.Reader
		if spec.body != nil {
			body = bytes.NewReader(spec.body)
		}
		req, err := http.NewRequestWithContext(r.ctx, spec.method, tr.t.URL+spec.path, body)
		if err != nil {
			return
		}
		if spec.body != nil {
			req.Header.Set("Content-Type", "application/json")
		}

		start := time.Now()
		resp, err := tr.client.Do(req)
		if err == nil {
			_, _ = io.Copy(io.Discard, resp.Body) // body harus dibaca habis agar koneksi dipakai ulang
			resp.Body.Close()
		}
		if r.ctx.Err() != nil {
			return // request dibatalkan karena test selesai: jangan dihitung error
		}
		tr.record(time.Since(start), err != nil || resp.StatusCode >= 400)
	}
}

func (r *run) status() Status {
	cfg := r.cfg
	return Status{
		Running:    true,
		RunID:      r.id,
		Config:     &cfg,
		ElapsedSec: time.Since(r.started).Seconds(),
		VUs:        int(r.desired.Load()),
	}
}

// tick menutup jendela 1 detik dan mencatat puncak resource.
func (r *run) tick(res map[string]ResourceSample) map[string]WindowStats {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	dt := now.Sub(r.lastTick).Seconds()
	r.lastTick = now
	if dt <= 0 {
		dt = 1
	}
	vus := int(r.desired.Load())

	out := make(map[string]WindowStats, len(r.targets))
	for _, tr := range r.targets {
		lat := tr.window.Swap(NewHistogram()).Snapshot()
		errs := tr.winErrors.Swap(0)
		ws := WindowStats{RPS: float64(lat.Count) / dt, Errors: errs, VUs: vus, Latency: lat}
		if lat.Count > 0 {
			ws.ErrorRate = float64(errs) / float64(lat.Count)
		}
		tr.peakRPS = max(tr.peakRPS, ws.RPS)
		if tr.haveBaseline {
			tr.reqSinceBaseline += lat.Count
		}

		// Sampel bisa hilang bila /metrics service terlalu sibuk untuk menjawab < 900 ms.
		if s, ok := res[tr.t.Name]; ok {
			tr.peakCPU = max(tr.peakCPU, s.CPUPercent)
			tr.cpuSum += s.CPUPercent
			tr.cpuN++
			tr.peakRSS = max(tr.peakRSS, s.RSSBytes)
			// Efisiensi = request / CPU-detik di antara sampel CPU pertama dan terakhir
			// yang berhasil, jadi sampel yang hilang di tengah tidak merusak rasionya.
			if !tr.haveBaseline {
				tr.haveBaseline = true
				tr.baseCPUSec = s.CPUSeconds
			}
			tr.lastCPUSec = s.CPUSeconds
			tr.reqAtLastCPU = tr.reqSinceBaseline
		}
		out[tr.t.Name] = ws
	}
	return out
}

func (r *run) summary() Summary {
	r.mu.Lock()
	defer r.mu.Unlock()

	dur := r.ended.Sub(r.started).Seconds()
	s := Summary{ID: r.id, Config: r.cfg, StartedAt: r.started, DurationSec: dur, Stopped: r.stopped.Load()}
	for _, tr := range r.targets {
		lat := tr.total.Snapshot()
		errs := tr.totalErrors.Load()
		ts := TargetSummary{
			Target:    tr.t.Name,
			Requests:  lat.Count,
			Errors:    errs,
			PeakRPS:   tr.peakRPS,
			Latency:   lat,
			PeakCPU:   tr.peakCPU,
			PeakRSSMB: float64(tr.peakRSS) / (1 << 20),
		}
		if dur > 0 {
			ts.AvgRPS = float64(lat.Count) / dur
		}
		if lat.Count > 0 {
			ts.ErrorRate = float64(errs) / float64(lat.Count)
		}
		if tr.cpuN > 0 {
			ts.AvgCPU = tr.cpuSum / float64(tr.cpuN)
		}
		ts.CPUSeconds = tr.lastCPUSec - tr.baseCPUSec
		if ts.CPUSeconds > 0 {
			ts.ReqPerCPUSec = float64(tr.reqAtLastCPU) / ts.CPUSeconds
		}
		s.Targets = append(s.Targets, ts)
	}
	return s
}
