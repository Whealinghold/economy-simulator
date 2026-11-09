package measure_test

import (
	"errors"
	"math"
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/measure"
)

// TestQuantile_NearestRank verifies the weighted nearest-rank quantile of
// spec-indicators.md 5.5.2: the rank is k = max(1, ceil(q * W / 10 000)), the
// value is a household's value and not an interpolation, Q(0) is the minimum and
// Q(10 000) the maximum, and a cohort whose total does not divide returns the
// even integer of the half.
//
// Requirement: FR-IND-024
func TestQuantile_NearestRank(t *testing.T) {
	t.Parallel()
	four := []measure.Row{{1, 5}, {1, 1}, {1, 9}, {1, 3}} // sorted 1 3 5 9, W = 4
	tests := []struct {
		name string
		rows []measure.Row
		q    int64
		want int64
	}{
		{"q 0 is the minimum", four, 0, 1},
		{"q 2 500 is rank 1", four, 2500, 1},
		{"q 2 501 is rank 2", four, 2501, 3},
		{"q 5 000 is rank 2", four, 5000, 3},
		{"q 5 001 is rank 3", four, 5001, 5},
		{"q 10 000 is the maximum", four, 10000, 9},
		{"a cohort of 2 holding 41 is 20", []measure.Row{{2, 41}}, 5000, 20},
		{"a cohort of 2 holding 43 is 22", []measure.Row{{2, 43}}, 5000, 22},
		{"weights count as households", []measure.Row{{9, 90}, {1, 500}}, 9000, 10},
		{"the weighted top", []measure.Row{{9, 90}, {1, 500}}, 9001, 500},
		{"negative values are allowed", []measure.Row{{1, -4}, {1, 2}, {1, -9}}, 3400, -4},
		{"equal values in either order", []measure.Row{{1, 5}, {1, 5}, {1, 2}}, 6700, 5},
		// W = 10 001 and q = 1: k = ceil(10 001 / 10 000) = 2, so the second household
		// (5), not the first (3); a rank that rounded 1.0001 down would return 3
		{"the rank rounds up by a single unit", []measure.Row{{1, 3}, {10000, 50000}}, 1, 5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok, err := measure.Quantile(tt.rows, tt.q)
			if err != nil || !ok || got != tt.want {
				t.Errorf("Quantile(%v, %d) = %d, %v, %v, want %d, true, nil", tt.rows, tt.q, got, ok, err, tt.want)
			}
		})
	}
}

// TestQuantile_Errors verifies the outcomes that are not a value: no rows is not
// ok, q outside [0, 10 000] and a weight below 1 are ErrRange, and weights that
// pass uint64 are ErrOverflow.
//
// Requirement: FR-IND-024
func TestQuantile_Errors(t *testing.T) {
	t.Parallel()
	one := []measure.Row{{1, 1}}
	tests := []struct {
		name string
		rows []measure.Row
		q    int64
		err  error
	}{
		{"no rows", nil, 5000, nil},
		{"q below the range", one, -1, measure.ErrRange},
		{"q above the range", one, 10001, measure.ErrRange},
		{"weight zero", []measure.Row{{0, 1}}, 5000, measure.ErrRange},
		{"weights past uint64", []measure.Row{{math.MaxInt64, 1}, {math.MaxInt64, 1}, {2, 1}}, 5000, measure.ErrOverflow},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got, ok, err := measure.Quantile(tt.rows, tt.q); ok || got != 0 || !errors.Is(err, tt.err) {
				t.Errorf("Quantile(%v, %d) = %d, %v, %v, want not ok, error %v", tt.rows, tt.q, got, ok, err, tt.err)
			}
		})
	}
}

// TestShareBelow_Strict verifies that the share counts the households strictly
// below x (Total < x * Weight), so a household at x is not below it, and that
// weights count as households.
//
// Requirement: FR-IND-025
func TestShareBelow_Strict(t *testing.T) {
	t.Parallel()
	four := []measure.Row{{1, 5}, {1, 1}, {1, 9}, {1, 3}}
	tests := []struct {
		name string
		rows []measure.Row
		x    int64
		want int64
	}{
		{"x at the minimum counts none", four, 1, 0},
		{"x above the minimum", four, 2, 2500},
		{"x at a value excludes it", four, 5, 5000},
		{"x above all", four, 10, 10000},
		{"a cohort at x is not below", []measure.Row{{2, 10}}, 5, 0},
		{"a cohort just below", []measure.Row{{2, 9}, {1, 40}}, 5, 6667},
		{"negative x with negative values", []measure.Row{{1, -4}, {1, 2}, {1, -9}}, -3, 6667},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok, err := measure.ShareBelow(tt.rows, tt.x)
			if err != nil || !ok || got != tt.want {
				t.Errorf("ShareBelow(%v, %d) = %d, %v, %v, want %d, true, nil", tt.rows, tt.x, got, ok, err, tt.want)
			}
		})
	}
	if got, ok, err := measure.ShareBelow(nil, 1); got != 0 || ok || err != nil {
		t.Errorf("ShareBelow(nil, 1) = %d, %v, %v, want 0, not ok, nil", got, ok, err)
	}
	if got, ok, err := measure.ShareBelow([]measure.Row{{0, 1}}, 1); got != 0 || ok || !errors.Is(err, measure.ErrRange) {
		t.Errorf("ShareBelow with a zero weight = %d, %v, %v, want 0, not ok, ErrRange", got, ok, err)
	}
}

// TestTopBottom_SplitsTheBoundaryRow verifies the shares of the top and bottom
// k households when the boundary falls inside a row: k' of the row's w households
// hold k' * T / w. Rows {1, 0} and {4, 40} (four households of 10): W = 5,
// S = 40; q = 3 000 gives k = ceil(1.5) = 2; the top two hold 20 of 40 (5 000),
// the bottom two hold 0 + 10 (2 500).
//
// Requirement: FR-IND-024
func TestTopBottom_SplitsTheBoundaryRow(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		rows        []measure.Row
		q           int64
		top, bottom int64
	}{
		{"boundary inside a cohort", []measure.Row{{1, 0}, {4, 40}}, 3000, 5000, 2500},
		// values 10, 30, 30, 30: S = 100, k = 2; top two hold 60, bottom two 10 + 30
		{"boundary on a row edge", []measure.Row{{1, 10}, {1, 30}, {2, 60}}, 5000, 6000, 4000},
		{"top decile of ten equal households", []measure.Row{{10, 100}}, 1000, 1000, 1000},
		{"all households", []measure.Row{{1, 10}, {1, 30}}, 10000, 10000, 10000},
		{"q 0 takes one household", []measure.Row{{1, 10}, {1, 30}}, 0, 7500, 2500},
		{"one household holds all of it", []measure.Row{{1, 10}}, 5000, 10000, 10000},
		{"one cohort of two", []measure.Row{{2, 10}}, 5000, 5000, 5000},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			top, bottom, ok, err := measure.TopBottom(tt.rows, tt.q)
			if err != nil || !ok || top != tt.top || bottom != tt.bottom {
				t.Errorf("TopBottom(%v, %d) = %d, %d, %v, %v, want %d, %d, true, nil", tt.rows, tt.q, top, bottom, ok, err, tt.top, tt.bottom)
			}
		})
	}
}

// TestTopBottom_Errors verifies the undefined and error outcomes of the shares.
//
// Requirement: FR-IND-024, FR-IND-009
func TestTopBottom_Errors(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		rows   []measure.Row
		q      int64
		wantOK bool
		err    error
	}{
		{"no rows", nil, 1000, false, nil},
		{"a total of zero", []measure.Row{{1, 0}, {2, 0}}, 1000, false, nil},
		{"a negative total", []measure.Row{{1, 5}, {1, -1}}, 1000, false, measure.ErrNegative},
		{"a zero weight", []measure.Row{{0, 5}}, 1000, false, measure.ErrRange},
		{"q out of range", []measure.Row{{1, 5}}, 10001, false, measure.ErrRange},
		{"weights past uint64", []measure.Row{{math.MaxInt64, 1}, {math.MaxInt64, 1}, {2, 1}}, 1000, false, measure.ErrOverflow},
		{"w times the total past 128 bits", []measure.Row{{math.MaxInt64, math.MaxInt64}, {math.MaxInt64, math.MaxInt64}}, 5000, false, measure.ErrOverflow},
		// four totals of 2^63 - 1 make S about 2^65; the boundary row's weight 2^62
		// times S passes 2^127 and the numerator times 10 000 passes 2^128
		{"five rows of the largest totals", []measure.Row{{1, math.MaxInt64}, {1, math.MaxInt64}, {1, math.MaxInt64}, {1, math.MaxInt64}, {1 << 62, math.MaxInt64}}, 5000, false, measure.ErrOverflow},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			top, bottom, ok, err := measure.TopBottom(tt.rows, tt.q)
			if ok != tt.wantOK || !errors.Is(err, tt.err) || top != 0 || bottom != 0 {
				t.Errorf("TopBottom(%v, %d) = %d, %d, %v, %v, want not ok, error %v", tt.rows, tt.q, top, bottom, ok, err, tt.err)
			}
		})
	}
}

// TestRank_OrderOfRowsDoesNotMatter verifies that Quantile, ShareBelow and
// TopBottom give the same answer for any order of the rows, with ties.
//
// Requirement: FR-IND-010
func TestRank_OrderOfRowsDoesNotMatter(t *testing.T) {
	t.Parallel()
	r := rand.New(rand.NewPCG(17, 18))
	for i := range 300 {
		rows := randRows(r, 2+r.IntN(8), 4, 8)
		q := r.Int64N(10001)
		v1, _, _ := measure.Quantile(rows, q)
		s1, _, _ := measure.ShareBelow(rows, 3)
		t1, b1, _, _ := measure.TopBottom(rows, q)
		shuffled := slices.Clone(rows)
		r.Shuffle(len(shuffled), func(a, b int) { shuffled[a], shuffled[b] = shuffled[b], shuffled[a] })
		v2, _, _ := measure.Quantile(shuffled, q)
		s2, _, _ := measure.ShareBelow(shuffled, 3)
		t2, b2, _, _ := measure.TopBottom(shuffled, q)
		if v1 != v2 || s1 != s2 || t1 != t2 || b1 != b2 {
			t.Fatalf("case %d: rows %v and %v disagree at q %d: (%d %d %d %d) against (%d %d %d %d)", i, rows, shuffled, q, v1, s1, t1, b1, v2, s2, t2, b2)
		}
	}
}

// FuzzRank verifies that Quantile, ShareBelow and TopBottom never panic and keep
// their bounds for any rows: the quantile lies between the least and greatest
// household value, a share is in [0, 10 000], and the top share is not below the
// bottom share of the same k.
//
// Requirement: NFR-IND-003
func FuzzRank(f *testing.F) {
	f.Add([]byte{1, 5, 1, 9, 2, 0}, uint16(5000), int8(3))
	f.Add([]byte{}, uint16(0), int8(0))
	f.Fuzz(func(t *testing.T, data []byte, q uint16, x int8) {
		var rows []measure.Row
		for i := 0; i+1 < len(data) && len(rows) < 12; i += 2 {
			rows = append(rows, measure.Row{Weight: int64(data[i]%7) + 1, Total: int64(data[i+1]) - 100})
		}
		qq := int64(q) % 10001
		if v, ok, err := measure.Quantile(rows, qq); err != nil {
			t.Fatalf("Quantile(%v, %d) error = %v", rows, qq, err)
		} else if ok {
			lo, hi := int64(math.MaxInt64), int64(math.MinInt64)
			for _, r := range rows {
				lo, hi = min(lo, r.Total/r.Weight-1), max(hi, r.Total/r.Weight+1)
			}
			if v < lo || v > hi {
				t.Errorf("Quantile(%v, %d) = %d, want within [%d, %d]", rows, qq, v, lo, hi)
			}
		}
		if s, ok, err := measure.ShareBelow(rows, int64(x)); err != nil || (ok && (s < 0 || s > 10000)) {
			t.Errorf("ShareBelow(%v, %d) = %d, %v, %v", rows, x, s, ok, err)
		}
		for i := range rows {
			rows[i].Total += 100 // shares need non-negative totals
		}
		top, bottom, ok, err := measure.TopBottom(rows, qq)
		if err != nil || (ok && (top < 0 || top > 10000 || bottom < 0 || bottom > 10000 || top+1 < bottom)) {
			t.Errorf("TopBottom(%v, %d) = %d, %d, %v, %v", rows, qq, top, bottom, ok, err)
		}
	})
}
