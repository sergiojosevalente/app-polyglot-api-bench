package store

import (
	"errors"
	"sync"
	"testing"
)

func TestSeedAndGet(t *testing.T) {
	s := New()
	s.Seed(10)

	item, err := s.Get(7)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if item.Name != "item-7" || item.Price != 10.5 {
		t.Errorf("unexpected item: %+v", item)
	}
	if _, err := s.Get(99); !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// Jalankan dengan `go test -race ./...` untuk membuktikan tidak ada data race.
func TestConcurrentUpsert(t *testing.T) {
	s := New()
	var wg sync.WaitGroup
	for i := 1; i <= 100; i++ {
		wg.Add(1)
		go func(id uint64) {
			defer wg.Done()
			s.Upsert(Item{ID: id, Name: "x", Price: 1})
			_, _ = s.Get(id)
		}(uint64(i))
	}
	wg.Wait()
	if s.Count() != 100 {
		t.Errorf("Count() = %d, want 100", s.Count())
	}
}
