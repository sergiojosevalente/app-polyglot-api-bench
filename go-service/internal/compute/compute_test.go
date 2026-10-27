package compute

import (
	"errors"
	"testing"
)

// Table-driven test: idiom standar Go.
func TestFib(t *testing.T) {
	cases := []struct {
		n    uint32
		want uint64
	}{{0, 0}, {1, 1}, {2, 1}, {10, 55}, {25, 75025}}

	for _, c := range cases {
		if got := Fib(c.n); got != c.want {
			t.Errorf("Fib(%d) = %d, want %d", c.n, got, c.want)
		}
	}
}

func TestSummarize(t *testing.T) {
	s, err := Summarize([]float64{1, 2, 3, 4.5, -2})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := Stats{Count: 5, Sum: 8.5, Min: -2, Max: 4.5, Mean: 1.7}
	if s != want {
		t.Errorf("got %+v, want %+v", s, want)
	}
}

func TestSummarizeEmpty(t *testing.T) {
	if _, err := Summarize(nil); !errors.Is(err, ErrEmpty) {
		t.Errorf("expected ErrEmpty, got %v", err)
	}
}
