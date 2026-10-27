// Monitor: load generator + collector metrics + dashboard live untuk
// membandingkan service Go, Rust, PHP, Node.js, Java, dan Python secara berdampingan.
package main

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gorust/monitor/internal/collector"
	"github.com/gorust/monitor/internal/hub"
	"github.com/gorust/monitor/internal/loadtest"
	"github.com/gorust/monitor/internal/target"
	"github.com/gorust/monitor/internal/verify"
)

//go:embed web
var webFS embed.FS

const historySize = 600 // 10 menit titik data per detik

// Point adalah satu titik data per detik yang dikirim ke dashboard.
type Point struct {
	T       int64                  `json:"t"`
	Status  loadtest.Status        `json:"status"`
	Targets map[string]TargetPoint `json:"targets"`
}

type TargetPoint struct {
	collector.Sample
	Load *loadtest.WindowStats `json:"load,omitempty"`
}

type history struct {
	mu     sync.Mutex
	points []Point
}

func (h *history) add(p Point) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.points = append(h.points, p)
	if len(h.points) > historySize {
		h.points = h.points[len(h.points)-historySize:]
	}
}

func (h *history) all() []Point {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]Point(nil), h.points...)
}

func main() {
	targets := []target.Target{
		{Name: "go", URL: getenv("GO_URL", "http://localhost:8081")},
		{Name: "rust", URL: getenv("RUST_URL", "http://localhost:8082")},
		{Name: "php", URL: getenv("PHP_URL", "http://localhost:8083")},
		{Name: "node", URL: getenv("NODE_URL", "http://localhost:8084")},
		{Name: "java", URL: getenv("JAVA_URL", "http://localhost:8085")},
		{Name: "python", URL: getenv("PYTHON_URL", "http://localhost:8086")},
	}

	events := hub.New()
	coll := collector.New(targets)
	hist := &history{}
	mgr := loadtest.NewManager(targets, func(s loadtest.Summary) {
		log.Printf("test %s finished", s.ID)
		events.Broadcast("summary", s)
	})

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go tickLoop(ctx, coll, mgr, events, hist)

	static, _ := fs.Sub(webFS, "web")
	mux := http.NewServeMux()
	mux.Handle("GET /", http.FileServerFS(static))
	mux.Handle("GET /api/stream", events)
	mux.HandleFunc("GET /api/history", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, hist.all())
	})
	mux.HandleFunc("GET /api/targets", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, targets)
	})
	mux.HandleFunc("GET /api/status", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, mgr.Status())
	})
	mux.HandleFunc("GET /api/results", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, mgr.Results())
	})
	mux.HandleFunc("POST /api/test/start", func(w http.ResponseWriter, r *http.Request) {
		var cfg loadtest.Config
		if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		id, err := mgr.Start(cfg)
		switch {
		case errors.Is(err, loadtest.ErrBusy):
			writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
		case err != nil:
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		default:
			writeJSON(w, http.StatusAccepted, map[string]string{"run_id": id})
		}
	})
	mux.HandleFunc("POST /api/test/stop", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]bool{"stopped": mgr.Stop()})
	})
	mux.HandleFunc("POST /api/verify", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, verify.Run(r.Context(), targets))
	})

	addr := getenv("LISTEN_ADDR", ":8080")
	srv := &http.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	go func() {
		log.Printf("monitor dashboard on http://localhost%s, targets: %v", addr, targets)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server error: %v", err)
		}
	}()

	<-ctx.Done()
	mgr.Stop()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx) // koneksi SSE akan diputus setelah timeout
}

// tickLoop: tiap detik scrape metrics, tutup jendela load test, siarkan ke dashboard.
func tickLoop(ctx context.Context, coll *collector.Collector, mgr *loadtest.Manager, events *hub.Hub, hist *history) {
	t := time.NewTicker(time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}

		sctx, cancel := context.WithTimeout(ctx, 900*time.Millisecond)
		samples := coll.Sample(sctx)
		cancel()

		res := make(map[string]loadtest.ResourceSample, len(samples))
		for name, s := range samples {
			if s.Up {
				res[name] = loadtest.ResourceSample{CPUPercent: s.CPUPercent, RSSBytes: s.RSSBytes, CPUSeconds: s.CPUSeconds}
			}
		}
		status, windows := mgr.Tick(res)

		p := Point{T: time.Now().UnixMilli(), Status: status, Targets: make(map[string]TargetPoint, len(samples))}
		for name, s := range samples {
			tp := TargetPoint{Sample: s}
			if w, ok := windows[name]; ok {
				tp.Load = &w
			}
			p.Targets[name] = tp
		}
		hist.add(p)
		events.Broadcast("tick", p)
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("write response: %v", err)
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
