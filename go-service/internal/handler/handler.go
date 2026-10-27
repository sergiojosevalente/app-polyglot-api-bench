// Package handler adalah lapisan HTTP: parsing request, validasi, panggil
// business logic, tulis response. Padanannya: rust-service/src/handlers.rs
package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"runtime"
	"strconv"
	"strings"

	"github.com/gorust/go-service/internal/compute"
	"github.com/gorust/go-service/internal/metrics"
	"github.com/gorust/go-service/internal/store"
)

const (
	lang        = "go"
	defaultFibN = 25
	maxBodySize = 1 << 20 // 1 MiB, sama dengan DefaultBodyLimit di Rust
)

// Handler memegang dependency (dependency injection manual lewat struct).
type Handler struct {
	store   *store.Store
	metrics *metrics.Metrics
}

func New(s *store.Store, m *metrics.Metrics) *Handler {
	return &Handler{store: s, metrics: m}
}

// Routes memakai ServeMux bawaan Go 1.22+ yang sudah mendukung method & path param.
func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", h.health)
	mux.HandleFunc("GET /api/hello", h.hello)
	mux.HandleFunc("POST /api/json", h.jsonStats)
	mux.HandleFunc("GET /api/cpu", h.cpu)
	mux.HandleFunc("GET /api/items/{id}", h.getItem)
	mux.HandleFunc("PUT /api/items/{id}", h.putItem)
	mux.HandleFunc("GET /metrics", h.metricsHandler)
	return h.metrics.Middleware(mux)
}

// ---- response types: struct biasa + json tag ----

type healthResponse struct {
	Status string `json:"status"`
	Lang   string `json:"lang"`
}

type helloResponse struct {
	Message string `json:"message"`
	Lang    string `json:"lang"`
}

type cpuResponse struct {
	N      uint32 `json:"n"`
	Result uint64 `json:"result"`
	Lang   string `json:"lang"`
}

// Embedding: field compute.Stats "diratakan" ke JSON (setara #[serde(flatten)]).
type statsResponse struct {
	compute.Stats
	Lang string `json:"lang"`
}

type statsRequest struct {
	Numbers []float64 `json:"numbers"`
}

// Price berupa pointer supaya bisa membedakan "tidak dikirim" vs 0.
// Tanpa pointer, Go diam-diam memakai zero value. Rust (serde) menolak field yang hilang.
type upsertRequest struct {
	Name  string   `json:"name"`
	Price *float64 `json:"price"`
}

type metricsResponse struct {
	Lang          string    `json:"lang"`
	Version       string    `json:"version"`
	UptimeSec     float64   `json:"uptime_sec"`
	RequestsTotal uint64    `json:"requests_total"`
	RSSBytes      uint64    `json:"rss_bytes"`
	CPUSeconds    float64   `json:"cpu_seconds"`
	Threads       int       `json:"threads"`
	Items         int       `json:"items"`
	Runtime       goRuntime `json:"runtime"`
}

type goRuntime struct {
	Goroutines     int    `json:"goroutines"`
	HeapAllocBytes uint64 `json:"heap_alloc_bytes"`
	GCCycles       uint32 `json:"gc_cycles"`
	GOMAXPROCS     int    `json:"gomaxprocs"`
}

// ---- handlers ----

func (h *Handler) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, healthResponse{Status: "ok", Lang: lang})
}

func (h *Handler) hello(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if name == "" {
		name = "World"
	}
	writeJSON(w, http.StatusOK, helloResponse{Message: "Hello, " + name + "!", Lang: lang})
}

func (h *Handler) cpu(w http.ResponseWriter, r *http.Request) {
	n := uint64(defaultFibN)
	if raw := r.URL.Query().Get("n"); raw != "" {
		v, err := strconv.ParseUint(raw, 10, 32)
		if err != nil {
			writeError(w, http.StatusBadRequest, "n must be a non-negative integer")
			return
		}
		n = v
	}
	if n > compute.MaxFib {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("n must be <= %d", compute.MaxFib))
		return
	}
	// Goroutine bersifat preemptive: kerja CPU berat di sini tidak "membekukan"
	// request lain. Bandingkan dengan catatan di handlers.rs (tokio cooperative).
	writeJSON(w, http.StatusOK, cpuResponse{N: uint32(n), Result: compute.Fib(uint32(n)), Lang: lang})
}

func (h *Handler) jsonStats(w http.ResponseWriter, r *http.Request) {
	var req statsRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	stats, err := compute.Summarize(req.Numbers)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, statsResponse{Stats: stats, Lang: lang})
}

func (h *Handler) getItem(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "id must be a non-negative integer")
		return
	}
	item, err := h.store.Get(id)
	switch {
	case errors.Is(err, store.ErrNotFound):
		writeError(w, http.StatusNotFound, fmt.Sprintf("item %d not found", id))
	case err != nil:
		writeError(w, http.StatusInternalServerError, err.Error())
	default:
		writeJSON(w, http.StatusOK, item)
	}
}

func (h *Handler) putItem(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "id must be a non-negative integer")
		return
	}
	var req upsertRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	switch {
	case strings.TrimSpace(req.Name) == "":
		writeError(w, http.StatusBadRequest, "name must not be empty")
		return
	case req.Price == nil:
		writeError(w, http.StatusBadRequest, "price is required")
		return
	case *req.Price < 0:
		writeError(w, http.StatusBadRequest, "price must be >= 0")
		return
	}
	item := h.store.Upsert(store.Item{ID: id, Name: req.Name, Price: *req.Price})
	writeJSON(w, http.StatusOK, item)
}

func (h *Handler) metricsHandler(w http.ResponseWriter, _ *http.Request) {
	ps := metrics.ReadProc()
	// ReadMemStats sempat menghentikan dunia sebentar; aman karena hanya dipanggil 1x/detik.
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)

	writeJSON(w, http.StatusOK, metricsResponse{
		Lang:          lang,
		Version:       runtime.Version(),
		UptimeSec:     h.metrics.Uptime().Seconds(),
		RequestsTotal: h.metrics.Requests(),
		RSSBytes:      ps.RSSBytes,
		CPUSeconds:    ps.CPUSeconds,
		Threads:       ps.Threads,
		Items:         h.store.Count(),
		Runtime: goRuntime{
			Goroutines:     runtime.NumGoroutine(),
			HeapAllocBytes: ms.HeapAlloc,
			GCCycles:       ms.NumGC,
			GOMAXPROCS:     runtime.GOMAXPROCS(0),
		},
	})
}

// ---- helpers ----

func parseID(r *http.Request) (uint64, error) {
	return strconv.ParseUint(r.PathValue("id"), 10, 64)
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)
	return json.NewDecoder(r.Body).Decode(dst)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("write response: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
