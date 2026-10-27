// Package store adalah repository in-memory yang aman dipakai banyak goroutine.
// Padanannya: rust-service/src/store.rs
package store

import (
	"errors"
	"fmt"
	"sync"
)

// ErrNotFound adalah sentinel error; caller memeriksanya dengan errors.Is.
var ErrNotFound = errors.New("item not found")

// Item adalah model domain. Struct tag `json:"..."` mengatur nama field JSON.
type Item struct {
	ID    uint64  `json:"id"`
	Name  string  `json:"name"`
	Price float64 `json:"price"`
}

// Store melindungi map dengan RWMutex: banyak pembaca ATAU satu penulis.
// Di Go, map biasa TIDAK thread-safe — lupa lock = data race (deteksi: go test -race).
type Store struct {
	mu    sync.RWMutex
	items map[uint64]Item
}

func New() *Store {
	return &Store{items: make(map[uint64]Item)}
}

// Seed mengisi item 1..n supaya benchmark baca punya data.
func (s *Store) Seed(n int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := 1; i <= n; i++ {
		id := uint64(i)
		s.items[id] = Item{ID: id, Name: fmt.Sprintf("item-%d", i), Price: float64(i) * 1.5}
	}
}

func (s *Store) Get(id uint64) (Item, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	item, ok := s.items[id]
	if !ok {
		return Item{}, ErrNotFound
	}
	return item, nil
}

// Upsert membuat atau mengganti item. Item disalin by value ke map.
func (s *Store) Upsert(item Item) Item {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[item.ID] = item
	return item
}

func (s *Store) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.items)
}
