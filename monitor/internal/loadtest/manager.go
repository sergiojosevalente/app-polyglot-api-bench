// Package loadtest adalah load generator closed-loop: N virtual user per target,
// masing-masing mengirim request berikutnya segera setelah response diterima.
package loadtest

import (
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/gorust/monitor/internal/target"
)

var ErrBusy = errors.New("a test is already running")

type Config struct {
	Targets     []string `json:"targets"`
	Scenario    string   `json:"scenario"`
	Mode        string   `json:"mode"`
	Concurrency int      `json:"concurrency"`
	DurationSec int      `json:"duration_sec"`
	CPUN        int      `json:"cpu_n"`
}

type Status struct {
	Running    bool    `json:"running"`
	RunID      string  `json:"run_id,omitempty"`
	Config     *Config `json:"config,omitempty"`
	ElapsedSec float64 `json:"elapsed_sec,omitempty"`
	VUs        int     `json:"vus,omitempty"`
}

type Manager struct {
	targets []target.Target
	onDone  func(Summary)

	mu      sync.Mutex
	run     *run
	results []Summary // terbaru di depan
}

func NewManager(targets []target.Target, onDone func(Summary)) *Manager {
	return &Manager{targets: targets, onDone: onDone}
}

func (m *Manager) normalize(c *Config) error {
	known := make([]string, 0, len(m.targets))
	for _, t := range m.targets {
		known = append(known, t.Name)
	}
	if len(c.Targets) == 0 {
		c.Targets = known
	}
	for _, t := range c.Targets {
		if !slices.Contains(known, t) {
			return fmt.Errorf("unknown target %q", t)
		}
	}
	if c.Scenario == "" {
		c.Scenario = "hello"
	}
	if !slices.Contains(Scenarios, c.Scenario) {
		return fmt.Errorf("unknown scenario %q", c.Scenario)
	}
	if c.Mode == "" {
		c.Mode = "load"
	}
	if !slices.Contains(Modes, c.Mode) {
		return fmt.Errorf("unknown mode %q", c.Mode)
	}
	if c.Concurrency <= 0 {
		c.Concurrency = 50
	}
	c.Concurrency = min(c.Concurrency, 2000)
	if c.DurationSec <= 0 {
		c.DurationSec = 30
	}
	c.DurationSec = min(max(c.DurationSec, 5), 600)
	if c.CPUN <= 0 {
		c.CPUN = 25
	}
	c.CPUN = min(c.CPUN, 35)
	return nil
}

func (m *Manager) Start(cfg Config) (string, error) {
	if err := m.normalize(&cfg); err != nil {
		return "", err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.run != nil {
		return "", ErrBusy
	}

	var selected []target.Target
	for _, t := range m.targets {
		if slices.Contains(cfg.Targets, t.Name) {
			selected = append(selected, t)
		}
	}
	r := newRun("run-"+time.Now().Format("20060102-150405"), cfg, selected)
	m.run = r

	go func() {
		r.execute()
		s := r.summary()
		m.mu.Lock()
		m.run = nil
		m.results = append([]Summary{s}, m.results...)
		if len(m.results) > 20 {
			m.results = m.results[:20]
		}
		m.mu.Unlock()
		m.onDone(s)
	}()
	return r.id, nil
}

func (m *Manager) Stop() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.run == nil {
		return false
	}
	m.run.stopped.Store(true)
	m.run.cancel()
	return true
}

func (m *Manager) Status() Status {
	m.mu.Lock()
	r := m.run
	m.mu.Unlock()
	if r == nil {
		return Status{}
	}
	return r.status()
}

// Tick dipanggil sekali per detik oleh loop utama.
func (m *Manager) Tick(res map[string]ResourceSample) (Status, map[string]WindowStats) {
	m.mu.Lock()
	r := m.run
	m.mu.Unlock()
	if r == nil {
		return Status{}, nil
	}
	return r.status(), r.tick(res)
}

func (m *Manager) Results() []Summary {
	m.mu.Lock()
	defer m.mu.Unlock()
	return slices.Clone(m.results)
}
