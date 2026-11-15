package technology_test

import (
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/technology"
)

const maxExp = int64(1) << 50

// matrix is a symmetric distance table as a DistanceFunc; a negative entry is
// no path.
func matrix(d [][]int64) technology.DistanceFunc {
	return func(a, b int) (int64, bool) { return d[a][b], d[a][b] >= 0 }
}

// diffusionInput is the three-region case of models/tec/law.out part 4: one
// field with only diffusion on (Reach 100, CatchUpBP 2 000), region 0 at a
// gain of 3 000 held by its advance, and the distances 0-1 20, 0-2 60, 1-2 50.
func diffusionInput(dist technology.DistanceFunc) technology.Input {
	return technology.Input{
		Cells:         []technology.Cell{{Advance: 3000}, {}, {}},
		Counters:      make([]int64, 3),
		TrendCarry:    make([]int64, 1),
		Fields:        []technology.Field{{CapBP: 100_000, CatchUpBP: 2000, Reach: 100}},
		Regions:       3,
		TicksPerYear:  365,
		MaxExperience: maxExp,
		Dist:          dist,
	}
}

var lawDistances = [][]int64{{-1, 20, 60}, {20, -1, 50}, {60, 50, -1}}

// TestCatchUp_NeverPassesFrontier verifies the diffusion step against the
// printed years of models/tec/law.out part 4: with the leader at 3 000, a
// lagging cell reaches 435 and 217 after one year, 1 517 and 758 after five,
// 2 075 and 1 038 after ten and 2 356 and 1 178 after twenty, and never passes
// w times the leader (2 400 for the near cell).
//
// Requirement: FR-TEC-016, FR-TEC-015, FR-TEC-013
func TestCatchUp_NeverPassesFrontier(t *testing.T) {
	t.Parallel()
	in := diffusionInput(matrix(lawDistances))
	var s technology.Scratch
	out := make([]technology.Result, 3)
	marks := map[int][2]int64{1: {435, 217}, 5: {1517, 758}, 10: {2075, 1038}, 20: {2356, 1178}}
	for tick := 1; tick <= 20*365; tick++ {
		in.Tick = int64(tick)
		if !technology.Step(in, &s, out) {
			t.Fatalf("Step refused at tick %d", tick)
		}
		if out[1].After > 2400 {
			t.Fatalf("tick %d: near cell at %d, above w * leader = 2400", tick, out[1].After)
		}
		if want, ok := marks[tick/365]; ok && tick%365 == 0 {
			if out[1].After != want[0] || out[2].After != want[1] || out[0].After != 3000 {
				t.Errorf("year %d: gains %d, %d, %d, want 3000, %d, %d", tick/365, out[0].After, out[1].After, out[2].After, want[0], want[1])
			}
		}
	}
}

// TestContact_ClosedEdgeStopsAndResumes verifies that a closed edge is no
// contact: with 0-1 closed (no path either way) the step adds nothing to any
// cell and the carries are 0, 0 and 2 400 000 (law.out part 4: cell 2 carries
// 1 200 * 2 000 toward its first unit, cell 1 has no frontier), and the same
// world with the edge open adds to cell 1.
//
// Requirement: FR-TEC-036, FR-TEC-014
func TestContact_ClosedEdgeStopsAndResumes(t *testing.T) {
	t.Parallel()
	closed := [][]int64{{-1, -1, 60}, {-1, -1, 50}, {60, 50, -1}}
	in := diffusionInput(matrix(closed))
	var s technology.Scratch
	out := make([]technology.Result, 3)
	if !technology.Step(in, &s, out) {
		t.Fatalf("Step refused")
	}
	for i, r := range out {
		if r.CatchUp != 0 {
			t.Errorf("closed edge: cell %d added %d, want 0", i, r.CatchUp)
		}
	}
	got := []int64{in.Cells[0].CatchUp, in.Cells[1].CatchUp, in.Cells[2].CatchUp}
	if want := []int64{0, 0, 2_400_000}; !slices.Equal(got, want) {
		t.Errorf("carries after a closed edge = %v, want %v", got, want)
	}
	in.Dist = matrix(lawDistances)
	if !technology.Step(in, &s, out) || in.Cells[1].CatchUp == 0 {
		t.Errorf("with the edge open cell 1 carries %d, want contact to resume", in.Cells[1].CatchUp)
	}
}

// TestDiffusion_SnapshotAndPermutations verifies that the result of a step
// does not depend on the order of the regions: for all six renamings of three
// regions the results of each region equal those of the identity renaming,
// whatever the trend, the experience and the diffusion.
//
// Requirement: FR-TEC-013, NFR-TEC-002
func TestDiffusion_SnapshotAndPermutations(t *testing.T) {
	t.Parallel()
	gains := []technology.Cell{{Experience: 40_000, Advance: 3000}, {Experience: 5_000, Advance: 400}, {}}
	counters := []int64{70, 3, 0}
	field := technology.Field{CapBP: 100_000, Base: 100_000, DoublingBP: 200, TrendBP: 137, CatchUpBP: 2000, Reach: 100}
	run := func(perm []int) []technology.Result {
		cells := make([]technology.Cell, 3)
		ctr := make([]int64, 3)
		for pos, r := range perm {
			cells[pos], ctr[pos] = gains[r], counters[r]
		}
		dist := func(a, b int) (int64, bool) { return lawDistances[perm[a]][perm[b]], true }
		in := technology.Input{Cells: cells, Counters: ctr, TrendCarry: make([]int64, 1), Fields: []technology.Field{field},
			Regions: 3, Tick: 7, TicksPerYear: 365, MaxExperience: maxExp, Dist: dist}
		out := make([]technology.Result, 3)
		var s technology.Scratch
		if !technology.Step(in, &s, out) {
			t.Fatalf("Step refused for %v", perm)
		}
		byRegion := make([]technology.Result, 3)
		for pos, r := range perm {
			byRegion[r] = out[pos]
		}
		return byRegion
	}
	want := run([]int{0, 1, 2})
	for _, perm := range [][]int{{0, 2, 1}, {1, 0, 2}, {1, 2, 0}, {2, 0, 1}, {2, 1, 0}} {
		if got := run(perm); !slices.Equal(got, want) {
			t.Errorf("order %v: results %+v, want those of the identity order %+v", perm, got, want)
		}
	}
}

// TestStep_EventIdentity verifies the identity of 4.6.3 over 100 000 random
// cells with a fixed seed: After = Before + Doing + Trend + Discovery +
// CatchUp - Clamped with Clamped never negative and After never below Before,
// the stored advance within [0, CapBP], and the state written back equal to
// the result.
//
// Requirement: FR-TEC-017, FR-TEC-005
func TestStep_EventIdentity(t *testing.T) {
	t.Parallel()
	rng := rand.New(rand.NewPCG(3, 4))
	var s technology.Scratch
	out := make([]technology.Result, 1)
	for n := range 100_000 {
		f := technology.Field{CapBP: 1 + rng.Int64N(5000), Base: 1 + rng.Int64N(200_000), DoublingBP: rng.Int64N(400),
			TrendBP: rng.Int64N(300), DiscoveryBP: rng.Int64N(20_000), StepBP: 1 + rng.Int64N(400)}
		c := technology.Cell{Experience: rng.Int64N(5_000_000), Advance: rng.Int64N(f.CapBP + 1)}
		in := technology.Input{Cells: []technology.Cell{c}, Counters: []int64{rng.Int64N(3000)}, TrendCarry: []int64{rng.Int64N(365)},
			Fields: []technology.Field{f}, Regions: 1, Tick: int64(n), TicksPerYear: 365, MaxExperience: maxExp, Key: rng.Uint64()}
		if !technology.Step(in, &s, out) {
			t.Fatalf("case %d: Step refused %+v", n, in)
		}
		r := out[0]
		if got := r.Before + r.Doing + r.Trend + r.Discovery + r.CatchUp - r.Clamped; got != r.After {
			t.Fatalf("case %d: identity %d != After %d (%+v)", n, got, r.After, r)
		}
		if r.Clamped < 0 || r.After < r.Before || r.Advance < 0 || r.Advance > f.CapBP || r.After > f.CapBP {
			t.Fatalf("case %d: result out of range %+v for cap %d", n, r, f.CapBP)
		}
		if in.Cells[0] != (technology.Cell{Experience: r.Experience, Advance: r.Advance, CatchUp: r.CatchUpCarry}) {
			t.Fatalf("case %d: state %+v does not match the result %+v", n, in.Cells[0], r)
		}
	}
}

// TestExperience_SaturatesNoWrap verifies that the experience stops at
// MaxExperience however large the counter and never wraps, and that a counter
// that would add less than the room left adds exactly.
//
// Requirement: FR-TEC-035
func TestExperience_SaturatesNoWrap(t *testing.T) {
	t.Parallel()
	f := technology.Field{CapBP: 1000, Base: 1, DoublingBP: 100}
	var s technology.Scratch
	out := make([]technology.Result, 1)
	for _, tt := range []struct{ x, counter, want int64 }{
		{maxExp - 5, 3, maxExp - 2},
		{maxExp - 5, 5, maxExp},
		{maxExp - 5, 6, maxExp},
		{maxExp, 1 << 62, maxExp},
		{0, 1<<63 - 1, maxExp},
	} {
		in := technology.Input{Cells: []technology.Cell{{Experience: tt.x}}, Counters: []int64{tt.counter}, TrendCarry: []int64{0},
			Fields: []technology.Field{f}, Regions: 1, TicksPerYear: 365, MaxExperience: maxExp}
		if !technology.Step(in, &s, out) || in.Cells[0].Experience != tt.want {
			t.Errorf("Step with X %d and counter %d: experience %d, want %d", tt.x, tt.counter, in.Cells[0].Experience, tt.want)
		}
	}
}

// TestStep_RefusesAndChangesNothing verifies the atomic refusal: a shape
// mismatch, a negative counter, an experience above the bound and an advance
// whose sum overflows each return false and leave the cells and the carries as
// they were.
//
// Requirement: NFR-TEC-005
func TestStep_RefusesAndChangesNothing(t *testing.T) {
	t.Parallel()
	base := func() technology.Input {
		in := diffusionInput(matrix(lawDistances))
		in.Fields[0].TrendBP = 100
		return in
	}
	tests := map[string]func(*technology.Input){
		"counter length":     func(in *technology.Input) { in.Counters = in.Counters[:2] },
		"negative counter":   func(in *technology.Input) { in.Counters[1] = -1 },
		"experience too big": func(in *technology.Input) { in.Cells[2].Experience = maxExp + 1 },
		"negative advance":   func(in *technology.Input) { in.Cells[0].Advance = -1 },
		"advance overflows":  func(in *technology.Input) { in.Cells[1].Advance, in.TrendCarry[0] = 1<<63-1, 300 },
		"no ticks per year":  func(in *technology.Input) { in.TicksPerYear = 0 },
		"negative cap":       func(in *technology.Input) { in.Fields[0].CapBP = -1 },
		"learning overflows": func(in *technology.Input) {
			in.Fields[0].DoublingBP, in.Fields[0].Base = 200, 1<<63-1
			in.Cells[1].Experience = 5
		},
		"catch-up overflows": func(in *technology.Input) {
			in.Cells[0].Advance, in.Fields[0].CatchUpBP, in.Fields[0].CapBP = 1<<62, 1<<62, 1<<62
		},
		"trend sum overflows": func(in *technology.Input) { in.TrendCarry[0] = 1<<63 - 1 },
		"discovery overflows": func(in *technology.Input) {
			in.Fields[0].DiscoveryBP, in.Fields[0].StepBP, in.Counters[0], in.Cells[0].Advance = 10_000*365, 1<<62, 1, 1<<62
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			in := base()
			mutate(&in)
			cells, carry := slices.Clone(in.Cells), slices.Clone(in.TrendCarry)
			var s technology.Scratch
			out := make([]technology.Result, len(in.Cells))
			if technology.Step(in, &s, out) {
				t.Fatalf("Step accepted the %s case", name)
			}
			if !slices.Equal(cells, in.Cells) || !slices.Equal(carry, in.TrendCarry) {
				t.Errorf("%s: state changed on a refusal: cells %v carry %v", name, in.Cells, in.TrendCarry)
			}
		})
	}
}

// TestStep_DiscoveryNeedsPractice verifies sub-step 3: a hit adds StepBP only
// to a cell whose counter is above 0, and the cell index of the hash is the
// canonical position (region * fields + field).
//
// Requirement: FR-TEC-009
func TestStep_DiscoveryNeedsPractice(t *testing.T) {
	t.Parallel()
	f := technology.Field{CapBP: 100_000, DiscoveryBP: 10_000 * 365, StepBP: 50}
	in := technology.Input{Cells: make([]technology.Cell, 2), Counters: []int64{0, 4}, TrendCarry: []int64{0},
		Fields: []technology.Field{f}, Regions: 2, TicksPerYear: 365, MaxExperience: maxExp}
	var s technology.Scratch
	out := make([]technology.Result, 2)
	if !technology.Step(in, &s, out) {
		t.Fatalf("Step refused")
	}
	// A rate of 10 000 * ticksPerYear makes the hit certain (the high word is
	// always below it), so only the practice decides.
	if out[0].Discoveries != 0 || out[1].Discoveries != 1 || out[1].Discovery != 50 || out[1].After != 50 {
		t.Errorf("results %+v, want a discovery of 50 in the practising cell only", out)
	}
}

// TestResult_Accrued verifies that a result is due an event exactly when one
// of the four additions or the clamp is not 0.
//
// Requirement: FR-TEC-024
func TestResult_Accrued(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		r    technology.Result
		want bool
	}{
		{"nothing", technology.Result{Before: 5, After: 5}, false},
		{"doing", technology.Result{Doing: 1}, true},
		{"trend", technology.Result{Trend: 1}, true},
		{"discovery", technology.Result{Discovery: 1}, true},
		{"catch-up", technology.Result{CatchUp: 1}, true},
		{"cut only", technology.Result{Clamped: 1}, true},
	}
	for _, tt := range tests {
		if got := tt.r.Accrued(); got != tt.want {
			t.Errorf("%s: Accrued() = %v, want %v", tt.name, got, tt.want)
		}
	}
}

// TestStep_RefusesDiffusionWithoutDistances verifies that a field with diffusion
// on and two regions or more, given no distance function, is refused with nothing
// changed, and not a nil call.
//
// Requirement: FR-TEC-016
func TestStep_RefusesDiffusionWithoutDistances(t *testing.T) {
	t.Parallel()
	in := diffusionInput(nil)
	in.Tick = 1
	out := make([]technology.Result, 3)
	var s technology.Scratch
	if technology.Step(in, &s, out) {
		t.Fatalf("Step accepted diffusion with no distances, want a refusal")
	}
	one := diffusionInput(nil)
	one.Cells, one.Counters, one.Regions, one.Tick = one.Cells[:1], one.Counters[:1], 1, 1
	if !technology.Step(one, &s, out[:1]) {
		t.Errorf("Step refused one region with no distances, want accepted")
	}
}
