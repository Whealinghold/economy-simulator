package indicator_test

import (
	"errors"
	"math"
	"testing"

	"github.com/Whealinghold/economy-sim/indicator"
)

// TestStats_Growth_Change verifies the window statistics of a series
// (spec-indicators.md 10.3.2): first, last, lowest, highest, count, mean half to
// even and the ratio High * 10 000 / Low, undefined for a Low at or below 0; the
// growth between two points (5.7.1), undefined for a base at or below 0 and
// clamped at MaxIndexBP; and the change of a series, which needs two points.
//
// Requirement: FR-IND-023
func TestStats_Growth_Change(t *testing.T) {
	t.Parallel()
	series := []indicator.Point{{Tick: 10, Value: 40}, {Tick: 20, Value: 100}, {Tick: 30, Value: 25}, {Tick: 40, Value: 25}}
	// mean: 190 / 4 = 47.5, the exact half goes to the even 48; spread: 100 * 10 000 / 25 = 40 000
	got := indicator.Stats(series)
	want := indicator.SeriesStats{
		OK: true, Count: 4,
		First: series[0], Last: series[3], Low: series[2], High: series[1],
		Mean: 48, SpreadOK: true, Spread: 40000,
	}
	if got != want {
		t.Errorf("Stats(%v) = %+v, want %+v", series, got, want)
	}
	if got := indicator.Stats(nil); got != (indicator.SeriesStats{}) {
		t.Errorf("Stats(nil) = %+v, want the zero value, not OK", got)
	}
	for _, low := range []int64{0, -5} {
		s := indicator.Stats([]indicator.Point{{Tick: 1, Value: low}, {Tick: 2, Value: 9}})
		if !s.OK || s.SpreadOK || s.Spread != 0 {
			t.Errorf("Stats with Low %d: OK %v, SpreadOK %v, Spread %d, want OK true and the spread undefined", low, s.OK, s.SpreadOK, s.Spread)
		}
	}
	huge := indicator.Stats([]indicator.Point{{Tick: 1, Value: 1}, {Tick: 2, Value: math.MaxInt64}})
	if !huge.SpreadOK || !huge.SpreadClamped || huge.Spread != indicator.MaxIndexBP {
		t.Errorf("Stats of 1 and MaxInt64: spread %d (ok %v, clamped %v), want MaxIndexBP clamped", huge.Spread, huge.SpreadOK, huge.SpreadClamped)
	}
	if wide := indicator.Stats([]indicator.Point{{Value: math.MaxInt64}, {Value: math.MaxInt64}}); wide.Mean != math.MaxInt64 {
		t.Errorf("Stats mean of two MaxInt64 = %d, want MaxInt64 (the sum needs 128 bits)", wide.Mean)
	}

	growth := []struct {
		name     string
		from, to indicator.Point
		want     indicator.Rate
	}{
		{"rise", indicator.Point{Value: 200}, indicator.Point{Value: 250}, indicator.Rate{OK: true, BP: 2500}},
		{"fall", indicator.Point{Value: 200}, indicator.Point{Value: 150}, indicator.Rate{OK: true, BP: -2500}},
		// 10 000 * 1 / 3 = 3 333.33, rounded to 3 333
		{"rounded", indicator.Point{Value: 3}, indicator.Point{Value: 4}, indicator.Rate{OK: true, BP: 3333}},
		{"zero base", indicator.Point{Value: 0}, indicator.Point{Value: 4}, indicator.Rate{}},
		{"negative base", indicator.Point{Value: -3}, indicator.Point{Value: 4}, indicator.Rate{}},
		{"clamped up", indicator.Point{Value: 1}, indicator.Point{Value: 1 << 40}, indicator.Rate{OK: true, Clamped: true, BP: indicator.MaxIndexBP}},
		{"clamped down", indicator.Point{Value: 1}, indicator.Point{Value: -(1 << 40)}, indicator.Rate{OK: true, Clamped: true, BP: -indicator.MaxIndexBP}},
		{"full range does not wrap", indicator.Point{Value: 1}, indicator.Point{Value: math.MinInt64}, indicator.Rate{OK: true, Clamped: true, BP: -indicator.MaxIndexBP}},
	}
	for _, tt := range growth {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := indicator.Growth(tt.from, tt.to); got != tt.want {
				t.Errorf("Growth(%v, %v) = %+v, want %+v", tt.from, tt.to, got, tt.want)
			}
		})
	}

	d, err := indicator.Change(series)
	if err != nil || d != (indicator.Diff{OK: true, Value: -15, Ticks: 30}) {
		t.Errorf("Change(%v) = %+v, %v, want {OK true Value -15 Ticks 30}", series, d, err)
	}
	if d, err := indicator.Change(series[:1]); err != nil || d.OK {
		t.Errorf("Change of one point = %+v, %v, want not OK", d, err)
	}
	over := []indicator.Point{{Value: math.MinInt64}, {Tick: 1, Value: math.MaxInt64}}
	if _, err := indicator.Change(over); !errors.Is(err, indicator.ErrOverflow) {
		t.Errorf("Change across the whole int64 range error = %v, want ErrOverflow", err)
	}
	ticks := []indicator.Point{{Tick: math.MinInt64}, {Tick: math.MaxInt64}}
	if _, err := indicator.Change(ticks); !errors.Is(err, indicator.ErrOverflow) {
		t.Errorf("Change across the whole tick range error = %v, want ErrOverflow", err)
	}
}
