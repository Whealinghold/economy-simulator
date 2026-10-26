package steady_test

import (
	"errors"
	"fmt"
	"math"
	"slices"
	"strings"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/steady"
)

// oneRegion is a model of two commodities and one region of 10 households whose
// upkeep is 10 of commodity 1, with an activity that has a capacity row (J 1,
// d 10: 10 < 365) and one that has not (J 1, d 365: 365 is not below 365).
func oneRegion() steady.Model {
	return steady.Model{
		Commodities: 2,
		Regions:     []steady.Region{{Households: 10, Upkeep: []int64{0, 10}}},
		Year:        365,
		Phi:         steady.Ratio{Num: 1, Den: 1},
		Activities: []steady.Activity{
			{Region: 0, Duration: 10, Starts: 1, FactorSum: 10000, Outputs: []steady.Item{{Commodity: 0, Qty: 5}}},
			{Region: 0, Duration: 365, Starts: 1, FactorSum: 10000, Inputs: []steady.Item{{Commodity: 0, Qty: 1}}, Outputs: []steady.Item{{Commodity: 1, Qty: 1}}},
		},
	}
}

// TestBuild_Layout verifies the rows and columns of the programme: goods rows
// first, then one labour row per region, then a capacity row only for an activity
// with J * d below the year, the activities in the order given and lambda last,
// and the right-hand sides.
//
// Right-hand sides: labour phi * H = 10; capacity H * J / year = 10 * 1 / 365 =
// 2/73; with the nominal phi = 250/365 and 730 households, labour is 730 * 250 /
// 365 = 500 and capacity 730 * 1 / 365 = 2.
//
// Requirement: FR-VIA-003, FR-VIA-022
func TestBuild_Layout(t *testing.T) {
	t.Parallel()
	nominal := oneRegion()
	nominal.Regions[0].Households = 730
	nominal.Phi = steady.Ratio{Num: 250, Den: 365}
	tests := []struct {
		name     string
		m        steady.Model
		rows     int
		cols     int
		capRow   []int
		labour   steady.Ratio
		capacity steady.Ratio
	}{
		// 2 goods + 1 labour + 1 capacity = 4 rows; 2 activities + lambda + 4 slacks.
		{"full labour", oneRegion(), 4, 7, []int{3, -1}, steady.Ratio{Num: 10, Den: 1}, steady.Ratio{Num: 2, Den: 73}},
		{"nominal", nominal, 4, 7, []int{3, -1}, steady.Ratio{Num: 500, Den: 1}, steady.Ratio{Num: 2, Den: 1}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			for _, p := range layoutProblems(tt.m, tt.rows, tt.cols, tt.capRow, tt.labour, tt.capacity) {
				t.Errorf("Build(%s): %s", tt.name, p)
			}
		})
	}
}

// layoutProblems builds m and compares its size, capacity rows and right-hand
// sides with the expectation.
func layoutProblems(m steady.Model, rows, cols int, capRow []int, labour, capacity steady.Ratio) []string {
	gotRows, gotCols := steady.Size(m)
	p, l, err := steady.Build(m)
	if err != nil {
		return []string{fmt.Sprintf("error = %v, want nil", err)}
	}
	var out []string
	if gotRows != rows || gotCols != cols || p.Rows != rows || len(p.Cols) != 3 || l.Lambda != 2 {
		out = append(out, fmt.Sprintf("Size = (%d, %d), Rows = %d, %d columns, lambda at %d, want (%d, %d) rows and columns, 3 columns, lambda at 2", gotRows, gotCols, p.Rows, len(p.Cols), l.Lambda, rows, cols))
	}
	if !slices.Equal(l.CapRow, capRow) {
		out = append(out, fmt.Sprintf("CapRow = %v, want %v", l.CapRow, capRow))
	}
	for name, pair := range map[string][2]steady.Ratio{
		"labour RHS": {p.RHS[l.LabourRow(0)], labour}, "capacity RHS": {p.RHS[3], capacity}, "goods RHS": {p.RHS[0], {Num: 0, Den: 1}},
	} {
		if pair[0] != pair[1] {
			out = append(out, fmt.Sprintf("%s = %v, want %v", name, pair[0], pair[1]))
		}
	}
	return out
}

// TestBuild_Errors verifies the refusals of Build: a non-zero fixed term, a
// right-hand side or an upkeep total beyond int64 and a malformed year or work
// share.
//
// Requirement: FR-VIA-003
func TestBuild_Errors(t *testing.T) {
	t.Parallel()
	with := func(f func(m *steady.Model)) steady.Model {
		m := oneRegion()
		f(&m)
		return m
	}
	tests := []struct {
		name string
		m    steady.Model
		want error
	}{
		{"fixed term past the goods rows", with(func(m *steady.Model) { m.Fixed = make([]steady.Ratio, m.Commodities+1) }), steady.ErrBadProgramme},
		// phi * H = 2 * MaxInt64 does not fit int64.
		{"labour overflow", with(func(m *steady.Model) { m.Regions[0].Households = math.MaxInt64; m.Phi = steady.Ratio{Num: 2, Den: 1} }), steady.ErrOverflow},
		// H * J / year = 3 * MaxInt64 / 4: the numerator 3 * MaxInt64 does not fit.
		{"capacity overflow", with(func(m *steady.Model) {
			m.Regions[0].Households = math.MaxInt64
			m.Year = 4
			m.Activities[0].Starts, m.Activities[0].Duration = 3, 1
		}), steady.ErrOverflow},
		{"upkeep overflow", with(func(m *steady.Model) {
			m.Regions = append(m.Regions, steady.Region{Households: 1, Upkeep: []int64{0, math.MaxInt64}})
			m.Regions[0].Upkeep[1] = math.MaxInt64
		}), steady.ErrOverflow},
		{"negative households", with(func(m *steady.Model) { m.Regions[0].Households = -1 }), steady.ErrBadProgramme},
		{"negative work share", with(func(m *steady.Model) { m.Phi = steady.Ratio{Num: -1, Den: 1} }), steady.ErrBadProgramme},
		{"zero fixed denominator", with(func(m *steady.Model) { m.Fixed = []steady.Ratio{{Num: 1, Den: 0}} }), steady.ErrBadProgramme},
		{"zero year", with(func(m *steady.Model) { m.Year = 0 }), steady.ErrBadProgramme},
		{"zero work denominator", with(func(m *steady.Model) { m.Phi = steady.Ratio{Num: 1, Den: 0} }), steady.ErrBadProgramme},
		{"commodity out of range", with(func(m *steady.Model) { m.Activities[0].Outputs[0].Commodity = 7 }), steady.ErrBadProgramme},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if _, _, err := steady.Build(tt.m); !errors.Is(err, tt.want) {
				t.Errorf("Build(%s) error = %v, want %v", tt.name, err, tt.want)
			}
		})
	}
}

// TestBuild_PanicsOnNonCanonicalOrder verifies that activities given with
// regions out of order, or in a region that does not exist, are a violated
// programmer invariant and panic with a message that says so.
//
// Guard: the column order decides which optimum Bland's rule finds (FR-VIA-022);
// the caller sorts, and an unsorted input must not be silently accepted.
func TestBuild_PanicsOnNonCanonicalOrder(t *testing.T) {
	t.Parallel()
	unsorted := oneRegion()
	unsorted.Regions = append(unsorted.Regions, steady.Region{Households: 1})
	unsorted.Activities[0].Region = 1
	outOfRange := oneRegion()
	outOfRange.Activities[1].Region = 5
	for name, m := range map[string]steady.Model{"unsorted": unsorted, "region out of range": outOfRange} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			defer func() {
				r := recover()
				s, _ := r.(string)
				if !strings.Contains(s, "invariant") {
					t.Errorf("Build(%s) panicked with %v, want a message containing \"invariant\"", name, r)
				}
			}()
			steady.Build(m)
		})
	}
}
