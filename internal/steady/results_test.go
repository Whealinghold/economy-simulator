package steady_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/steady"
)

// planting is the hand model behind the capacity-rent case (spec-viability.md
// 5.3.4). Commodities: grain 0, food 1 (and extra unused ones). One region of 10
// households eats 1 food each. P plants: a 10-tick window, d 10, so J 1 and a
// capacity row x_P <= 10 * 1 / 365 = 2/73; it makes 200 grain. M mills: d 1, one
// grain in, one food out, all year.
func planting(commodities int, extra ...steady.Activity) steady.Model {
	year := int64(365)
	p := steady.Starts(10, []steady.Run{{Length: 10, FactorBP: 10000}, {Length: 355}})
	mill := steady.Starts(1, []steady.Run{{Length: 365, FactorBP: 10000}})
	m := steady.Model{
		Commodities: commodities,
		Regions:     []steady.Region{{Households: 10, Upkeep: []int64{0, 10}}},
		Year:        year,
		Phi:         steady.Ratio{Num: 1, Den: 1},
		Activities: []steady.Activity{
			{Region: 0, Duration: 10, Starts: p.Starts, FactorSum: p.FactorSum, Outputs: []steady.Item{{Commodity: 0, Qty: 200}}},
			{Region: 0, Duration: 1, Starts: mill.Starts, FactorSum: mill.FactorSum,
				Inputs: []steady.Item{{Commodity: 0, Qty: 1}}, Outputs: []steady.Item{{Commodity: 1, Qty: 1}}},
		},
	}
	m.Activities = append(m.Activities, extra...)
	return m
}

// TestSolveModel_CapacityRentAndNominal verifies the case of spec-viability.md
// 5.3.4: with full labour the start-capacity row is tight, labour is slack, the
// wage is 0 and the capacity row has a rent, and the labour share is absent; with
// the nominal work share 1/2 labour binds, the wage is positive and the rent is 0
// (FR-VIA-058 as a unit).
//
// Full labour: x_P = 2/73 grain 200 * 2/73 = 400/73, food the same, so lambda* =
// (400/73) / 10 = 40/73 = 0.5479, SurplusBP = floor(5479.45) = 5479. Duals:
// sum p D = 1 gives p_food = 1/10; M active gives p_grain = p_food = 1/10 (wage 0);
// P active gives 200 * p_grain = 10 * w + g, so the rent g = 20; check: g * 2/73
// = 40/73. Flows at lambda 1: x_P / lambda* = (2/73) / (40/73) = 1/20, x_M = 10.
//
// Nominal (phi 1/2, labour 5): 10 x_P + 200 x_P <= 5 gives x_P = 1/42 (below the
// capacity 2/73), lambda* = 200/42 / 10 = 10/21, SurplusBP = floor(4761.9) = 4761,
// labour share = ceil(10^8 / 4761) = 21004. Duals: p_food = 1/10, M: 1/10 - p_grain
// = w, P: 200 p_grain = 10 w (rent 0), so w = 2/21 and p_grain = 1/210; check:
// 5 * 2/21 = 10/21. Flows: (1/42) / (10/21) = 1/20 and 10.
//
// Requirement: FR-VIA-003, FR-VIA-012, FR-VIA-058
func TestSolveModel_CapacityRentAndNominal(t *testing.T) {
	t.Parallel()
	nominal := planting(2)
	nominal.Phi = steady.Ratio{Num: 1, Den: 2}
	tests := []struct {
		name string
		m    steady.Model
		want rentWant
	}{
		{"full labour", planting(2), rentWant{"40/73", 5479, 0, false, []string{"1/10", "1/10"}, "0", []string{"20", "0"}, []string{"1/20", "10"}, []bool{true, false}}},
		{"nominal", nominal, rentWant{"10/21", 4761, 21004, true, []string{"1/210", "1/10"}, "2/21", []string{"0", "0"}, []string{"1/20", "10"}, []bool{false, false}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			o, err := steady.SolveModel(context.Background(), tt.m, 1000)
			if err != nil {
				t.Fatalf("SolveModel(%s) error = %v, want nil", tt.name, err)
			}
			for _, p := range append(tt.want.problems(o), dualProblems(tt.m, o)...) {
				t.Errorf("SolveModel(%s): %s", tt.name, p)
			}
		})
	}
}

// rentWant is the expected outcome of the planting model.
type rentWant struct {
	lambda    string
	bp, share int64
	hasShare  bool
	price     []string
	wage      string
	rent      []string
	flow      []string
	tight     []bool
}

// problems compares an outcome with the expectation.
func (w rentWant) problems(o steady.Outcome) []string {
	var out []string
	check := func(name string, got, want []string) {
		if !slices.Equal(got, want) {
			out = append(out, fmt.Sprintf("%s = %v, want %v", name, got, want))
		}
	}
	check("Lambda", []string{str(o.Lambda)}, []string{w.lambda})
	check("Price", strs(o.Price), w.price)
	check("Wage", []string{str(o.Wage[0])}, []string{w.wage})
	check("Rent", strs(o.Rent), w.rent)
	check("Flow", strs(o.Flow), w.flow)
	if o.SurplusBP != w.bp || o.HasLabourShare != w.hasShare || (w.hasShare && o.LabourShareBP != w.share) {
		out = append(out, fmt.Sprintf("(surplus %d, share %d, has %v), want (%d, %d, %v)", o.SurplusBP, o.LabourShareBP, o.HasLabourShare, w.bp, w.share, w.hasShare))
	}
	if !slices.Equal(o.CapTight, w.tight) || !slices.Equal(o.Active, []bool{true, true}) {
		out = append(out, fmt.Sprintf("CapTight %v Active %v, want %v and both active", o.CapTight, o.Active, w.tight))
	}
	return out
}

// TestSolveModel_Diagnostics verifies Degenerate, Alternative and the zero-priced
// commodities: an unused commodity row of 0 <= 0 leaves a basic slack at 0
// (degenerate) and a price of 0, listed in ascending order; a second copy of an
// activity is a non-basic column with a zero reduced cost (alternative optima);
// the plain model has neither.
//
// Requirement: FR-VIA-019
func TestSolveModel_Diagnostics(t *testing.T) {
	t.Parallel()
	mill := planting(2).Activities[1]
	tests := []struct {
		name        string
		m           steady.Model
		degenerate  bool
		alternative bool
		zero        []int
	}{
		{"plain", planting(2), false, false, nil},
		// Commodities 2, 3 and 4 appear in no activity and no upkeep: their rows are
		// 0 <= 0, their prices 0.
		{"unused commodities", planting(5), true, false, []int{2, 3, 4}},
		{"duplicate activity", planting(2, mill), false, true, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			o, err := steady.SolveModel(context.Background(), tt.m, 1000)
			if err != nil {
				t.Fatalf("SolveModel(%s) error = %v, want nil", tt.name, err)
			}
			if o.Degenerate != tt.degenerate || o.Alternative != tt.alternative {
				t.Errorf("SolveModel(%s) = (degenerate %v, alternative %v), want (%v, %v)", tt.name, o.Degenerate, o.Alternative, tt.degenerate, tt.alternative)
			}
			if !slices.Equal(o.ZeroPriced, tt.zero) {
				t.Errorf("SolveModel(%s).ZeroPriced = %v, want %v", tt.name, o.ZeroPriced, tt.zero)
			}
		})
	}
}

// TestSolveModel_FailureModes verifies that an empty basket is unbounded, the
// pivot cap gives a LimitError and no result, a cancelled context gives its
// wrapped error, and a model Build refuses is refused with its error.
//
// Requirement: FR-VIA-016, FR-VIA-017
func TestSolveModel_FailureModes(t *testing.T) {
	t.Parallel()
	noUpkeep := planting(2)
	noUpkeep.Regions[0].Upkeep = nil
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	badYear := planting(2)
	badYear.Year = 0
	tests := []struct {
		name string
		ctx  context.Context
		m    steady.Model
		max  int64
		want error
	}{
		{"empty basket", context.Background(), noUpkeep, 1000, steady.ErrUnbounded},
		{"pivot cap", context.Background(), planting(2), 0, steady.ErrLimit},
		{"cancelled", cancelled, planting(2), 1000, context.Canceled},
		{"refused model", context.Background(), badYear, 1000, steady.ErrBadProgramme},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			o, err := steady.SolveModel(tt.ctx, tt.m, tt.max)
			if !errors.Is(err, tt.want) {
				t.Errorf("SolveModel(%s) error = %v, want %v", tt.name, err, tt.want)
			}
			if o.Flow != nil || o.Price != nil {
				t.Errorf("SolveModel(%s) returned a partial result %+v with an error", tt.name, o)
			}
		})
	}
}

// TestSolveModel_ZeroSurplusHasZeroFlows verifies that when lambda* is 0 (a
// region that cannot make its basket) the flows at lambda 1 are 0 rather than a
// division by zero, no activity is active and the labour share is absent.
//
// Requirement: FR-VIA-003
func TestSolveModel_ZeroSurplusHasZeroFlows(t *testing.T) {
	t.Parallel()
	m := space().model([]string{"deep_crater"}, []int64{400}, full)
	o, err := steady.SolveModel(context.Background(), m, 1000)
	if err != nil {
		t.Fatalf("SolveModel(deep crater) error = %v, want nil", err)
	}
	for k, f := range o.Flow {
		if f.Num.Sign() != 0 || o.Active[k] {
			t.Errorf("SolveModel(deep crater) activity %d has flow %s active %v, want 0 and inactive", k, str(f), o.Active[k])
		}
	}
	if o.SurplusBP != 0 || o.HasLabourShare {
		t.Errorf("SolveModel(deep crater) = (surplus %d, has share %v), want (0, false)", o.SurplusBP, o.HasLabourShare)
	}
}

// TestSolveModel_IsDeterministic verifies that two solves of one model give
// identical results.
//
// Requirement: FR-VIA-020
func TestSolveModel_IsDeterministic(t *testing.T) {
	t.Parallel()
	m := bread(breadQty{1500, 60, 100, 12, 6, 12, 18, 12, 18}).model([]string{"north", "south", "far_south"}, []int64{1000, 1000, 1000}, full)
	a, errA := steady.SolveModel(context.Background(), m, 10000)
	b, errB := steady.SolveModel(context.Background(), m, 10000)
	if errA != nil || errB != nil {
		t.Fatalf("SolveModel errors = %v, %v, want nil", errA, errB)
	}
	render := func(o steady.Outcome) string {
		return strings.Join(append(append(append([]string{str(o.Lambda)}, strs(o.Price)...), strs(o.Wage)...), strs(o.Flow)...), ",")
	}
	ra, rb := render(a), render(b)
	if ra != rb || !slices.Equal(a.Active, b.Active) || a.Pivots != b.Pivots {
		t.Errorf("two SolveModel calls differ:\n%s\n%s", ra, rb)
	}
}

// FuzzProgramme verifies, for small random models, that SolveModel never panics
// and never fails the certificate: it returns a result whose duals satisfy
// dual feasibility in exact arithmetic, or ErrUnbounded for an empty basket, and
// two solves agree.
//
// Requirement: FR-VIA-013, FR-VIA-014, NFR-VIA-004
func FuzzProgramme(f *testing.F) {
	f.Add([]byte{3, 2, 1, 10, 20, 5, 3, 7, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12})
	f.Add([]byte{0, 0, 0, 0})
	f.Add([]byte{255, 128, 64, 32, 16, 8, 4, 2, 1, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9})
	f.Fuzz(func(t *testing.T, data []byte) {
		m := fuzzModel(data)
		a, err := steady.SolveModel(context.Background(), m, 100000)
		if errors.Is(err, steady.ErrUnbounded) {
			return
		}
		if err != nil {
			t.Fatalf("SolveModel(%+v) error = %v, want nil or ErrUnbounded", m, err)
		}
		if p := dualProblems(m, a); len(p) > 0 {
			t.Fatalf("SolveModel(%+v) breaks dual feasibility: %v", m, p)
		}
		b, err := steady.SolveModel(context.Background(), m, 100000)
		if err != nil || str(a.Lambda) != str(b.Lambda) || a.Pivots != b.Pivots {
			t.Fatalf("second SolveModel(%+v) = (%v, %v), first (%s, %d)", m, b.Pivots, err, str(a.Lambda), a.Pivots)
		}
	})
}

// fuzzModel decodes up to 3 commodities, 2 regions and 5 activities of small
// numbers, activities in region order.
func fuzzModel(data []byte) steady.Model {
	at := 0
	next := func(mod int) int64 {
		if at >= len(data) {
			return 0
		}
		at++
		return int64(data[at-1]) % int64(mod)
	}
	nc, nr := int(1+next(3)), int(1+next(2))
	m := steady.Model{Commodities: nc, Year: 10 + next(10), Phi: steady.Ratio{Num: 1 + next(4), Den: 4}}
	for r := 0; r < nr; r++ {
		rg := steady.Region{Households: 1 + next(20), Upkeep: make([]int64, nc)}
		for c := range rg.Upkeep {
			rg.Upkeep[c] = next(3)
		}
		m.Regions = append(m.Regions, rg)
	}
	for k := int(next(6)); k > 0; k-- {
		starts := 1 + next(5)
		a := steady.Activity{Region: int(next(int(nr))), Duration: 1 + next(4), Starts: starts, FactorSum: starts * (1 + next(12)) * 1000}
		a.Outputs = []steady.Item{{Commodity: int(next(nc)), Qty: next(20)}}
		a.Inputs = []steady.Item{{Commodity: int(next(nc)), Qty: next(5)}}
		a.Tools = []steady.Tool{{Commodity: int(next(nc)), Wear: next(3), Durability: 1 + next(100)}}
		m.Activities = append(m.Activities, a)
	}
	slices.SortStableFunc(m.Activities, func(a, b steady.Activity) int { return a.Region - b.Region })
	return m
}
