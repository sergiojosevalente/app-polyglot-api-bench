// Package metrics menghitung request dan membaca statistik proses dari /proc
// (Linux/Docker). Padanannya: rust-service/src/metrics.rs
package metrics

import (
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

// Metrics menyimpan counter global. atomic.Uint64 = lock-free, aman lintas goroutine.
type Metrics struct {
	started  time.Time
	requests atomic.Uint64
}

func New() *Metrics {
	return &Metrics{started: time.Now()}
}

// Middleware membungkus handler lain: pola decorator func(http.Handler) http.Handler.
func (m *Metrics) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		m.requests.Add(1)
		next.ServeHTTP(w, r)
	})
}

func (m *Metrics) Requests() uint64 { return m.requests.Load() }

func (m *Metrics) Uptime() time.Duration { return time.Since(m.started) }

// ProcStats adalah statistik proses dari sudut pandang OS (sama untuk Go dan Rust,
// jadi perbandingannya adil).
type ProcStats struct {
	RSSBytes   uint64
	CPUSeconds float64
	Threads    int
}

// clockTicks = USER_HZ, hampir selalu 100 di Linux.
const clockTicks = 100.0

// ReadProc membaca /proc/self/stat. Di luar Linux (mis. go run di Windows) hasilnya nol.
func ReadProc() ProcStats {
	var ps ProcStats
	data, err := os.ReadFile("/proc/self/stat")
	if err != nil {
		return ps
	}
	s := string(data)
	// Field ke-2 (nama proses) bisa berisi spasi, jadi mulai setelah ')' terakhir.
	i := strings.LastIndexByte(s, ')')
	if i < 0 || i+2 > len(s) {
		return ps
	}
	f := strings.Fields(s[i+2:]) // f[0] = field 3 (state), jadi field k = f[k-3]
	if len(f) < 22 {
		return ps
	}
	utime, _ := strconv.ParseFloat(f[11], 64) // field 14
	stime, _ := strconv.ParseFloat(f[12], 64) // field 15
	threads, _ := strconv.Atoi(f[17])         // field 20
	rssPages, _ := strconv.ParseUint(f[21], 10, 64)

	ps.CPUSeconds = (utime + stime) / clockTicks
	ps.Threads = threads
	ps.RSSBytes = rssPages * uint64(os.Getpagesize())
	return ps
}
