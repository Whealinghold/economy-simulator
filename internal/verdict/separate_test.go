package verdict_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/verdict"
)

// TestSeparate_Midpoint verifies the separation protocol of STY 6.5.3: the
// integer midpoint of t_hi and t_lo rounded up, then up to a multiple of 5 for
// ticks or to two significant digits for a share, kept in [t_hi+1, t_lo];
// t_hi >= t_lo does not separate; a detector whose bad side is low is negated.
//
// Worked values: ticks 52 and 71: ceil(123/2) = 62, up to 65, inside [53, 71].
// Ticks 100 and 101: ceil(201/2) = 101, up to 105, clamped to t_lo = 101.
// Share 3 100 and 4 290: ceil(7 390/2) = 3 695, two digits up is 3 700.
// Share 3 100 and 3 120: ceil(6 220/2) = 3 110, 2 digits up is 3 200, clamped to 3 120.
// Low is bad: healthy volume shares 6 000, 7 000, 9 000 and provoked 2 000, 3 000
// negate to t_hi = -6 000, t_lo = -3 000; the midpoint ceil(-9 000/2) = -4 500
// is already two digits; the key is 4 500.
//
// Requirement: FR-STY-071, FR-STY-074
func TestSeparate_Midpoint(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name     string
		h, p     []int64
		kind     verdict.Kind
		lowIsBad bool
		want     int64
	}{
		{"ticks round up to 5", []int64{40, 52, 48}, []int64{71, 90}, verdict.Ticks, false, 65},
		{"adjacent values give t_lo", []int64{100}, []int64{101, 150}, verdict.Ticks, false, 101},
		{"a multiple of 5 is kept", []int64{40}, []int64{60}, verdict.Ticks, false, 50},
		{"share to two significant digits", []int64{3100, 2000}, []int64{4290, 5000}, verdict.Share, false, 3700},
		{"share clamped to t_lo", []int64{3100}, []int64{3120}, verdict.Share, false, 3120},
		{"low is bad is negated and back", []int64{6000, 7000, 9000}, []int64{2000, 3000}, verdict.Share, true, 4500},
		{"negative ticks round toward +inf", []int64{-12}, []int64{-3}, verdict.Ticks, false, -5},
	} {
		got, err := verdict.Separate(tc.h, tc.p, tc.kind, tc.lowIsBad)
		if err != nil || got != tc.want {
			t.Errorf("Separate(%s: %v, %v) = %d, %v, want %d", tc.name, tc.h, tc.p, got, err, tc.want)
		}
		if err == nil && !tc.lowIsBad {
			hi := max(tc.h[0], tc.h[len(tc.h)-1])
			for _, x := range tc.h {
				hi = max(hi, x)
			}
			if got < hi+1 {
				t.Errorf("Separate(%s) = %d, below t_hi+1 = %d", tc.name, got, hi+1)
			}
		}
	}

	t.Run("does not separate", func(t *testing.T) {
		for _, tc := range []struct{ h, p []int64 }{
			{[]int64{60}, []int64{60}}, // t_hi = t_lo
			{[]int64{70}, []int64{60}}, // t_hi > t_lo
			{[]int64{10, 90}, []int64{50, 100}},
		} {
			if _, err := verdict.Separate(tc.h, tc.p, verdict.Ticks, false); !errors.Is(err, verdict.ErrNoSeparation) {
				t.Errorf("Separate(%v, %v) error = %v, want ErrNoSeparation", tc.h, tc.p, err)
			}
		}
	})
	t.Run("no threshold from one set alone", func(t *testing.T) {
		if _, err := verdict.Separate([]int64{10}, nil, verdict.Ticks, false); !errors.Is(err, verdict.ErrNoRuns) {
			t.Errorf("Separate(healthy only) error = %v, want ErrNoRuns", err)
		}
		if _, err := verdict.Separate(nil, []int64{10}, verdict.Share, false); !errors.Is(err, verdict.ErrNoRuns) {
			t.Errorf("Separate(provoked only) error = %v, want ErrNoRuns", err)
		}
	})
	t.Run("extremes", func(t *testing.T) {
		const minI, maxI = -1 << 63, 1<<63 - 1
		if _, err := verdict.Separate([]int64{minI}, []int64{1}, verdict.Ticks, true); !errors.Is(err, verdict.ErrOverflow) {
			t.Errorf("Separate(min int64, low is bad) error = %v, want ErrOverflow", err)
		}
		if _, err := verdict.Separate([]int64{1}, []int64{minI}, verdict.Ticks, true); !errors.Is(err, verdict.ErrOverflow) {
			t.Errorf("Separate(provoked min int64, low is bad) error = %v, want ErrOverflow", err)
		}
		// t_lo at the top: rounding up must not wrap; the key falls back to t_lo.
		got, err := verdict.Separate([]int64{maxI - 3}, []int64{maxI}, verdict.Ticks, false)
		if err != nil || got != maxI-1 && got != maxI {
			t.Errorf("Separate(near max) = %d, %v, want a value within [t_hi+1, t_lo]", got, err)
		}
		got, err = verdict.Separate([]int64{maxI - 1}, []int64{maxI}, verdict.Share, false)
		if err != nil || got != maxI {
			t.Errorf("Separate(max-1, max) = %d, %v, want t_lo", got, err)
		}
	})
}
