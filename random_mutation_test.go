package economysim

import (
	"math/bits"
	"testing"
)

// Equivalent mutants of random.go that no test can kill (gomu, one-line why each):
//   - draw: `n < 1` -> `n <= 1`: a bound of 1 yields the high word of w*1, which is 0, the same value.
//   - draw: `return 0` -> `return 0`: the mutation is a no-op.
//   - drawRange: `hi < lo` -> `hi <= lo`: for hi == lo the span is 1 and the draw is lo + 0, the same value.
//   - drawRange: `span == 0` -> `span <= 0`: span is unsigned, so the two tests are the same.

// TestStream_DrawBoundaries pins the values beside the bounds that the
// equivalent mutants above cannot tell apart, and the span of two values on
// each side of the empty-range check.
//
// Requirement: FR-WLD-061
func TestStream_DrawBoundaries(t *testing.T) {
	t.Parallel()
	const w0 uint64 = 0x264d986499a1a1fa // first word of (siteWork, 0, 0), spec-world.md 7.1.5 (b) k1
	hi2, _ := bits.Mul64(w0, 2)
	tests := []struct {
		name string
		take func(s *stream) int64
		want int64
	}{
		{"bound 1", func(s *stream) int64 { return s.draw(1) }, 0},
		{"bound 2", func(s *stream) int64 { return s.draw(2) }, int64(hi2)},
		{"range of one value", func(s *stream) int64 { return s.drawRange(7, 7) }, 7},
		{"range of two values", func(s *stream) int64 { return s.drawRange(7, 8) }, 7 + int64(hi2)},
		{"range one below empty", func(s *stream) int64 { return s.drawRange(8, 7) }, 8},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := newStream(0)
			s.keyed(siteWork, 0, 0)
			if got := tt.take(s); got != tt.want {
				t.Errorf("%s = %d, want %d", tt.name, got, tt.want)
			}
		})
	}
}
