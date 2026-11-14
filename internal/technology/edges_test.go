package technology_test

import (
	"math"
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/technology"
)

// TestWeight_Edges verifies the edges of the contact weight that a table of typical
// values leaves open: a distance of 1 is the nearest contact, a reach of 1 or 2 has
// at most the one distance below it, and a reach of 0 or a distance of 0 is no
// contact.
//
// The arithmetic: reach 2, d 1 gives (2 - 1) * 10 000 / 2 = 5 000; reach 1, d 1 is at
// the reach and gives 0; reach 10 000, d 1 gives 9 999 * 10 000 / 10 000 = 9 999.
//
// Requirement: FR-TEC-014
func TestWeight_Edges(t *testing.T) {
	t.Parallel()
	tests := []struct{ reach, d, want int64 }{
		{2, 1, 5000}, {1, 1, 0}, {10_000, 1, 9999}, {100, 0, 0}, {0, 0, 0}, {-5, 1, 0}, {100, -3, 0},
	}
	for _, tt := range tests {
		if got := technology.Weight(tt.reach, tt.d, true); got != tt.want {
			t.Errorf("Weight(%d, %d, true) = %d, want %d", tt.reach, tt.d, got, tt.want)
		}
	}
}

// TestFrontier_SkipsSelfAndNonPositive verifies that the frontier ignores the region
// itself, a region with no gain and a region out of contact, and that a contact at
// weight 0 (at the reach) adds nothing, so the maximum is over the regions that give
// something.
//
// Requirement: FR-TEC-013
func TestFrontier_SkipsSelfAndNonPositive(t *testing.T) {
	t.Parallel()
	dist := func(a, b int) (int64, bool) { return int64(10 * (a + b + 1)), a+b != 5 }
	gains := []int64{0, 4000, 9000, 0}
	// Region 0 reads region 1 (d 20, w 8 000 at reach 100: 3 200), region 2 (d 30,
	// w 7 000: 6 300) and region 3 (gain 0); region 0's own gain is not read.
	if got := technology.Frontier(0, gains, 100, dist); got != 6300 {
		t.Errorf("Frontier(0) = %d, want 6300", got)
	}
	// Region 3 reads region 2 at no path (2 + 3 = 5), region 1 at d 50 (w 5 000: 2 000).
	if got := technology.Frontier(3, gains, 100, dist); got != 2000 {
		t.Errorf("Frontier(3) = %d, want 2000", got)
	}
	// At the reach the weight is 0: nothing is read.
	if got := technology.Frontier(0, gains, 20, dist); got != 0 {
		t.Errorf("Frontier(0) at reach 20 = %d, want 0 (a contact at the reach has weight 0)", got)
	}
}

// TestHit_HalfTheCellTicks verifies the hit at a rate of one half, where the boolean
// is sensitive to every term of the key: with DiscoveryBP = 10 000 * ticksPerYear / 2
// about half of the cell-ticks hit, equal to the math/big evaluation on every one,
// and the first and last cell of the grid and tick 0 are in range.
//
// Requirement: FR-TEC-009
func TestHit_HalfTheCellTicks(t *testing.T) {
	t.Parallel()
	const rate, tpy, cells = 10_000 * 365 / 2, 365, 7
	hits, total := 0, 0
	for _, key := range []uint64{0, 12345, ^uint64(0)} {
		for tick := int64(0); tick < 120; tick++ {
			for cell := int64(0); cell < cells; cell++ {
				got := technology.Hit(key, tick, cell, cells, rate, tpy)
				if want := oracleHit(key, tick, cell, cells, rate, tpy); got != want {
					t.Fatalf("Hit(%#x, %d, %d, %d, %d, %d) = %v, want %v", key, tick, cell, cells, rate, tpy, got, want)
				}
				total++
				if got {
					hits++
				}
			}
		}
	}
	if hits < total*4/10 || hits > total*6/10 {
		t.Errorf("%d hits of %d cell-ticks at a rate of one half, want between 40%% and 60%%", hits, total)
	}
}

// TestStep_WritesBackTheCarry verifies that a successful step writes the trend carry
// back, adds the trend to every cell of the field and nothing to a field with the
// trend off, and that the carry is cumulative over steps.
//
// The arithmetic: 100 a year at 365 ticks per year: the first step has sum 100, add 0
// and carry 100; the fourth has sum 400, add 1 and carry 35.
//
// Requirement: FR-TEC-008
func TestStep_WritesBackTheCarry(t *testing.T) {
	t.Parallel()
	in := technology.Input{
		Cells: make([]technology.Cell, 4), Counters: make([]int64, 4), TrendCarry: make([]int64, 2),
		Fields:  []technology.Field{{CapBP: 1000, TrendBP: 100}, {CapBP: 1000}},
		Regions: 2, TicksPerYear: 365, MaxExperience: 1 << 50,
	}
	var s technology.Scratch
	out := make([]technology.Result, 4)
	for step := 1; step <= 4; step++ {
		if !technology.Step(in, &s, out) {
			t.Fatalf("Step %d refused", step)
		}
	}
	if want := []int64{35, 0}; !slices.Equal(in.TrendCarry, want) {
		t.Errorf("TrendCarry after four steps = %v, want %v", in.TrendCarry, want)
	}
	adv := []int64{in.Cells[0].Advance, in.Cells[1].Advance, in.Cells[2].Advance, in.Cells[3].Advance}
	if want := []int64{1, 0, 1, 0}; !slices.Equal(adv, want) {
		t.Errorf("advance of the four cells = %v, want %v: the trend adds to the cells of its field only", adv, want)
	}
}

// TestStep_RefusesEachShapeMistake verifies that each mismatch of the input and each
// bad number is refused alone: a negative region count, a cells, counters, carry or
// result slice of the wrong length, no ticks per year, and a refusal at each stage of
// the fold (a before-gain, a before-learning and an after-learning that overflow).
//
// Requirement: NFR-TEC-005
func TestStep_RefusesEachShapeMistake(t *testing.T) {
	t.Parallel()
	base := func() technology.Input {
		return technology.Input{
			Cells: make([]technology.Cell, 2), Counters: make([]int64, 2), TrendCarry: make([]int64, 1),
			Fields:  []technology.Field{{CapBP: 1000, Base: 10, DoublingBP: 100}},
			Regions: 2, TicksPerYear: 365, MaxExperience: 1 << 50,
		}
	}
	tests := map[string]func(in *technology.Input){
		"cells short":         func(in *technology.Input) { in.Cells = in.Cells[:1] },
		"cells long":          func(in *technology.Input) { in.Cells = make([]technology.Cell, 3) },
		"counters long":       func(in *technology.Input) { in.Counters = make([]int64, 3) },
		"carry long":          func(in *technology.Input) { in.TrendCarry = make([]int64, 2) },
		"carry short":         func(in *technology.Input) { in.TrendCarry = nil },
		"negative regions":    func(in *technology.Input) { in.Regions = -1 },
		"ticks per year 0":    func(in *technology.Input) { in.TicksPerYear = 0 },
		"ticks per year -1":   func(in *technology.Input) { in.TicksPerYear = -1 },
		"negative counter":    func(in *technology.Input) { in.Counters[0] = -1 },
		"negative experience": func(in *technology.Input) { in.Cells[0].Experience = -1 },
		"before-gain overflows": func(in *technology.Input) {
			in.Fields[0].Base, in.Cells[1].Experience = math.MaxInt64-4, 5
		},
		"learning after the fold overflows": func(in *technology.Input) {
			in.Fields[0].Base, in.Counters[1] = math.MaxInt64-4, 5
		},
		"advance plus learned overflows": func(in *technology.Input) {
			in.Fields[0].Base, in.Cells[0].Experience, in.Cells[0].Advance = 1, 1<<40, math.MaxInt64
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			in := base()
			mutate(&in)
			out := make([]technology.Result, 2)
			var s technology.Scratch
			if technology.Step(in, &s, out) {
				t.Errorf("Step accepted the %s case", name)
			}
		})
	}
}

// TestStep_OneCellOfTwoDiscovers verifies the cell index of the hash: with two cells
// in one field and a rate that always hits, a practising cell discovers at its own
// canonical position and an idle cell does not, and a field with the rate off is
// skipped.
//
// Requirement: FR-TEC-009
func TestStep_OneCellOfTwoDiscovers(t *testing.T) {
	t.Parallel()
	fields := []technology.Field{{CapBP: 1000, DiscoveryBP: 10_000 * 365, StepBP: 7}, {CapBP: 1000, StepBP: 0}}
	in := technology.Input{
		Cells: make([]technology.Cell, 4), Counters: []int64{0, 5, 3, 3}, TrendCarry: make([]int64, 2),
		Fields: fields, Regions: 2, TicksPerYear: 365, MaxExperience: 1 << 50,
	}
	out := make([]technology.Result, 4)
	var s technology.Scratch
	if !technology.Step(in, &s, out) {
		t.Fatalf("Step refused")
	}
	got := []int64{out[0].Discovery, out[1].Discovery, out[2].Discovery, out[3].Discovery}
	if want := []int64{0, 0, 7, 0}; !slices.Equal(got, want) {
		t.Errorf("discoveries = %v, want %v (cell 0 idle, cell 1 field with the rate off, cell 2 practises, cell 3 field with the rate off)", got, want)
	}
}

// TestFrontier_ScansPastARegionOutOfContact verifies that a region with no weight (at
// or past the reach) does not end the scan: the frontier still reads the regions after
// it.
//
// Requirement: FR-TEC-013
func TestFrontier_ScansPastARegionOutOfContact(t *testing.T) {
	t.Parallel()
	dist := func(a, b int) (int64, bool) { return map[int]int64{0: 500, 1: 10}[a], true }
	// Region 0 is 500 away (reach 100: no weight) and region 1 is 10 away (w 9 000).
	if got := technology.Frontier(2, []int64{9000, 2000, 0}, 100, dist); got != 1800 {
		t.Errorf("Frontier(2) = %d, want 1800 (2000 * 9000 / 10000): the region out of reach must not stop the scan", got)
	}
}

// TestStep_FieldsAreIndependent verifies that a field with a mechanism off does not end
// the pass of the mechanisms over the fields after it: the first field has no trend,
// no discovery and no diffusion, the second has all three, and the second's cells
// receive them.
//
// Requirement: FR-TEC-008, FR-TEC-009, FR-TEC-015
func TestStep_FieldsAreIndependent(t *testing.T) {
	t.Parallel()
	fields := []technology.Field{
		{CapBP: 1000},
		{CapBP: 100_000, TrendBP: 365, DiscoveryBP: 10_000 * 365, StepBP: 5, CatchUpBP: 10_000, Reach: 100},
	}
	in := technology.Input{
		Cells:    []technology.Cell{{}, {Advance: 4000}, {}, {}},
		Counters: []int64{0, 0, 0, 1}, TrendCarry: make([]int64, 2), Fields: fields,
		Regions: 2, TicksPerYear: 365, MaxExperience: 1 << 50,
		Dist: func(a, b int) (int64, bool) { return 10, true },
	}
	out := make([]technology.Result, 4)
	var s technology.Scratch
	if !technology.Step(in, &s, out) {
		t.Fatalf("Step refused")
	}
	// Field 1: the trend adds 365 / 365 = 1 to both cells; the practising cell (region
	// 1) discovers 5; region 0 starts at 4 000 and region 1 at 0. The frontier of
	// region 1 is 4 001 * 9 000 / 10 000 = 3 600 and its gap 3 600 - 6 = 3 594; at 10 000
	// bp a year the carry is 3 594 * 10 000 = 35 940 000 over 3 650 000, so it adds 9.
	if out[1].Trend != 1 || out[3].Trend != 1 || out[3].Discovery != 5 || out[3].CatchUp != 9 || out[0].Trend != 0 {
		t.Errorf("results %+v, want the second field's mechanisms applied to its cells and none to the first", out)
	}
}

// TestStep_AMissedHitDoesNotEndTheScan verifies that a practising cell that does not
// discover leaves the cells after it their chance: for a key at which the first of two
// practising cells misses and the second hits at a rate of one half, only the second
// discovers.
//
// Requirement: FR-TEC-009
func TestStep_AMissedHitDoesNotEndTheScan(t *testing.T) {
	t.Parallel()
	const rate, tpy = 10_000 * 365 / 2, 365
	key := uint64(0)
	for ; technology.Hit(key, 0, 0, 2, rate, tpy) || !technology.Hit(key, 0, 1, 2, rate, tpy); key++ {
		if key > 1000 {
			t.Fatalf("no key below 1000 misses on cell 0 and hits on cell 1")
		}
	}
	in := technology.Input{
		Cells: make([]technology.Cell, 2), Counters: []int64{1, 1}, TrendCarry: make([]int64, 1),
		Fields: []technology.Field{{CapBP: 1000, DiscoveryBP: rate, StepBP: 3}}, Regions: 2,
		TicksPerYear: tpy, MaxExperience: 1 << 50, Key: key,
	}
	out := make([]technology.Result, 2)
	var s technology.Scratch
	if !technology.Step(in, &s, out) || out[0].Discoveries != 0 || out[1].Discoveries != 1 {
		t.Errorf("results %+v, want only the second cell to discover (key %d)", out, key)
	}
}

// TestStep_RegionCountIsBoundedByTheInput verifies that the region count cannot
// make the step allocate more than the input holds: a count whose product with the
// number of fields wraps to the length of the cells, and a huge count with no field,
// are refused or cost nothing, and a negative tick is refused (a hit on it would
// read as no discovery).
//
// The arithmetic: 4 fields * 2^62 regions = 2^64, which wraps to 0, the length of
// empty cells.
//
// Requirement: NFR-TEC-005
func TestStep_RegionCountIsBoundedByTheInput(t *testing.T) {
	t.Parallel()
	fields := []technology.Field{{CapBP: 1}, {CapBP: 1}, {CapBP: 1}, {CapBP: 1}}
	tests := map[string]technology.Input{
		"count wraps with the fields": {
			TrendCarry: make([]int64, 4), Fields: fields, Regions: 1 << 62, TicksPerYear: 1,
		},
		"negative tick": {
			Cells: make([]technology.Cell, 4), Counters: make([]int64, 4), TrendCarry: make([]int64, 4),
			Fields: fields, Regions: 1, Tick: -1, TicksPerYear: 1,
		},
	}
	for name, in := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			var s technology.Scratch
			if technology.Step(in, &s, make([]technology.Result, len(in.Cells))) {
				t.Errorf("Step accepted the %s case", name)
			}
		})
	}
	var s technology.Scratch
	if !technology.Step(technology.Input{Regions: 1 << 62, TicksPerYear: 1}, &s, nil) {
		t.Error("Step refused a world with no field")
	}
}
