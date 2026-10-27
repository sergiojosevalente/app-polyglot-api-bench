// Package compute berisi business logic murni tanpa pengetahuan tentang HTTP,
// sehingga mudah di-unit-test dan dipakai ulang. Padanannya: rust-service/src/compute.rs
package compute

import (
	"errors"
	"math"
)

// MaxFib membatasi n supaya satu request tidak memonopoli CPU berdetik-detik.
const MaxFib = 35

// ErrEmpty dikembalikan Summarize bila input kosong (sentinel error, dicek dengan errors.Is).
var ErrEmpty = errors.New("numbers must not be empty")

// Fib sengaja dibuat rekursif naif (eksponensial): ini benchmark CPU murni,
// bukan contoh algoritma yang baik.
func Fib(n uint32) uint64 {
	if n < 2 {
		return uint64(n)
	}
	return Fib(n-1) + Fib(n-2)
}

// Stats adalah ringkasan statistik dari sekumpulan angka.
type Stats struct {
	Count int     `json:"count"`
	Sum   float64 `json:"sum"`
	Min   float64 `json:"min"`
	Max   float64 `json:"max"`
	Mean  float64 `json:"mean"`
}

// Summarize menghitung count/sum/min/max/mean. Go memakai pola (value, error).
func Summarize(nums []float64) (Stats, error) {
	if len(nums) == 0 {
		return Stats{}, ErrEmpty
	}
	s := Stats{Count: len(nums), Min: math.Inf(1), Max: math.Inf(-1)}
	for _, n := range nums {
		s.Sum += n
		s.Min = min(s.Min, n)
		s.Max = max(s.Max, n)
	}
	s.Mean = s.Sum / float64(s.Count)
	return s, nil
}
