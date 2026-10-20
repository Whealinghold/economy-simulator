package economysim

import (
	"cmp"
	"context"
	"errors"
	"math"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/steady"
	"github.com/Whealinghold/economy-sim/internal/welfare"
)

// This file kills the mutants of viability_table.go that the neighbouring tests
// of the warm-start table leave alive (gomu: MSI 69.1% before, 92.6% after). Most
// cases call one method of the table builder on a hand-made region context, so a
// case states its numbers and each mutant has one case that sees it.
//
// Equivalent mutants of viability_table.go that no test can kill (gomu, one-line
// why each; the 73 survivors of the last run are these):
//   - scale `steady.Rat(0, 1) -> Rat(0, 2)` (107 both), census (760) and reserves
//     (436): 0/2 is the same fraction as 0/1.
//   - `== 0 -> <=` or `== 0 -> >=` on a value that is never negative (87, 269,
//     408, 427 both, 499, 563, 620, 651, 718 both, 766, 778, 818, 859, 930, 809):
//     `<= 0` and `== 0` agree; so do `> 0 -> !=` and `>= 0` (157, 252, 395, 468,
//     507, 516, 542, 690, 864) and `< 0 -> <= 0` (288, 917, 920, 683:20).
//   - seedPrice `return 0, 0, false, err` values (249): the one caller reads the
//     error only.
//   - reserveTotal `y > 0` -> `true, >=, !=` (463) and `Ceil(0, ...) ->
//     Ceil(1, ...)` (467, 469): a source of yield 0 has no spans, and a value of
//     0 comes back as 0 either way.
//   - jobs `rc.nVirt == nil -> true` (492) and the cache store (496): the cache
//     only saves recomputing the same count.
//   - residual `!ok` (656, 657): Quo fails only on a zero target, and a wage of 0
//     is skipped above it, so the branch is unreachable.
//   - mainRecipe `d == dmax -> >=` (338): `d > dmax ||` already holds for `>`.
//   - need `ok && jd < year -> ||, <=, !=` and `capacity > need -> true, >=, !=`
//     (743, 744): when jd >= year the start capacity x * year / J is at most the
//     labour x * d, and when jd < year it is strictly above it, so the compare
//     alone decides.
//   - recipeCensus `roles > 1`, `roles > 0` (809) and `Ceil(0, ...)` (806): roles
//     is never 1 there, and a need of 0 comes back as 0.
//   - finish `len(v.rep.errs) == 0 -> true, >=` (563): the offsets are then also
//     checked on a table that is already an error, which is the same error.
//   - surpluses `len(p.regions) > 1 -> <, !=, > 2` (840): with free trade the
//     pooled programme is progs[0], which the second clause already names, and
//     without it no programme has more than one region.
//   - ratioOf `Price[b].Sign() == 0 -> <` (930): Quo of a zero b is not ok anyway.

// TestMaxInt64Constant verifies the spelled-out largest int64.
//
// Guard: `1<<63 - 1` mutated to `1<<62 - 1` or `1<<63 - 2` is a different bound
// for every checked conversion that uses it.
//
// Requirement: FR-VIA-015
func TestMaxInt64Constant(t *testing.T) {
	t.Parallel()
	if int64(maxInt64) != math.MaxInt64 {
		t.Errorf("maxInt64 = %d, want %d", int64(maxInt64), int64(math.MaxInt64))
	}
}

// tblSolved validates c, solves its programmes and returns the state build
// starts from.
func tblSolved(t *testing.T, c Config, opts ...SolveOption) *viaSolve {
	t.Helper()
	if err := c.Validate(); err != nil {
		t.Fatalf("Validate = %v, want nil", err)
	}
	o, err := newSolveOptions(opts)
	if err != nil {
		t.Fatalf("newSolveOptions = %v, want nil", err)
	}
	cc := c.canonical()
	s := &viaSolve{m: newViaModel(&cc), o: o}
	s.m.maxPivots = o.maxPivots
	if err := s.solve(context.Background()); err != nil {
		t.Fatalf("solve = %v, want nil", err)
	}
	return s
}

// tblRegion returns the context of region 0 of the first programme of a solve.
func tblRegion(t *testing.T, s *viaSolve) *regionCtx {
	t.Helper()
	p := s.progs[0]
	scale, ok := s.scale(p)
	if !ok {
		t.Fatalf("scale = false, want a price scale")
	}
	return s.newRegion(p, 0, scale)
}

// TestBuild_TableErrorsPropagate verifies that an overflow in any step of a
// region's rows comes back from build, rows and programmeTable with its text and
// no partial result.
//
// Guard: `if err != nil` of build, programmeTable and rows turned to false, their
// bodies emptied or `return err` turned to `return nil` would hand back a table
// that was never built.
//
// Requirement: FR-VIA-030, FR-VIA-032
func TestBuild_TableErrorsPropagate(t *testing.T) {
	t.Parallel()
	t.Run("reference band through build", func(t *testing.T) {
		t.Parallel()
		s := tblSolved(t, viaToy())
		s.m.c.Commodities[0].ReferencePrice = 1 << 62 // the band's upper bound 10 * 2^62 leaves int64
		err := s.build()
		if !errors.Is(err, ErrOverflow) || !strings.Contains(err.Error(), "reference band of a price of") {
			t.Errorf("build with a reference price of 2^62 = %v, want ErrOverflow naming the reference band", err)
		}
	})
	t.Run("wage through build", func(t *testing.T) {
		t.Parallel()
		s := tblSolved(t, viaToy())
		s.progs[0].pr.Wage[0] = steady.Rat(1<<62, 1) // times the scale 1 000 is above MaxMoney
		err := s.build()
		if !errors.Is(err, ErrOverflow) || !strings.Contains(err.Error(), "starting wage of region") {
			t.Errorf("build with a wage dual of 2^62 = %v, want ErrOverflow naming the starting wage", err)
		}
	})
	t.Run("stock through rows", func(t *testing.T) {
		t.Parallel()
		s := tblSolved(t, viaToy())
		rc := tblRegion(t, s)
		rc.groups[0].reserve[0] = math.MaxInt64 // the cover 5 plus the reserve leaves int64
		if err := rc.rows(); !errors.Is(err, ErrOverflow) {
			t.Errorf("rows with a reserve of MaxInt64 = %v, want ErrOverflow", err)
		}
	})
	t.Run("money through rows", func(t *testing.T) {
		t.Parallel()
		s := tblSolved(t, viaToy())
		s.m.c.Commodities[0].ReferencePrice = MaxPrice // the seed is lifted to 10^8
		rc := tblRegion(t, s)
		rc.groups[0].up[0] = 1 << 40 // the cover fits; the upkeep 2^40 * 10^8 leaves int64
		if err := rc.rows(); !errors.Is(err, ErrOverflow) {
			t.Errorf("rows with an upkeep of 2^40 at a seed of 10^8 = %v, want ErrOverflow", err)
		}
	})
}

// TestBand verifies the reference band of a price, its rounding and each way it
// overflows.
//
// Guard: `!ok1 || !ok2` turned to `&&` passes a band with one bound lost, the
// error's text and zero bounds are returned values, and the body of the branch is
// the error itself.
//
// Requirement: FR-VIA-030
func TestBand(t *testing.T) {
	t.Parallel()
	band := func(w World, ref int64) (int64, int64, error) {
		s := &viaSolve{m: &viaModel{c: &Config{World: w}}}
		return s.band(ref)
	}
	w := DefaultWorld() // 1 000 to 100 000 basis points
	if lo, hi, err := band(w, 101); err != nil || lo != 11 || hi != 1_010 {
		t.Errorf("band(101) = %d, %d, %v, want 11, 1 010, nil (10.1 rounded up; 1 010 exactly)", lo, hi, err)
	}
	tests := []struct {
		name string
		edit func(*World)
	}{
		{"only the upper bound overflows", func(*World) {}}, // 2^62 * 100 000 / 10 000 = 10 * 2^62
		{"only the lower bound overflows", func(w *World) { w.RefBandLow, w.RefBandHigh = 100_000, 1_000 }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w := DefaultWorld()
			tt.edit(&w)
			lo, hi, err := band(w, 1<<62)
			if !errors.Is(err, ErrOverflow) || !strings.Contains(err.Error(), "reference band of a price of") || lo != 0 || hi != 0 {
				t.Errorf("band(2^62) = %d, %d, %v, want 0, 0 and ErrOverflow naming the reference band", lo, hi, err)
			}
		})
	}
}

// TestPrices_ReportsTheSeedsTheBandChanged verifies the records of prices: the
// seed rows, the floored, the unpriced, the seeds outside the band (above and
// below the rounded price) and the drift of each.
//
// Guard: `seed != p1` turned to any other compare, its body emptied or its
// assignment removed loses a record of OutsideReferenceBand, or lists a seed the
// band did not change.
//
// Requirement: FR-VIA-023, FR-VIA-025, FR-VIA-026, FR-VIA-030
func TestPrices_ReportsTheSeedsTheBandChanged(t *testing.T) {
	t.Parallel()
	cfg := &Config{World: DefaultWorld(), Commodities: []Commodity{
		{ID: "a", ReferencePrice: 100}, {ID: "b", ReferencePrice: 5_000}, {ID: "c", ReferencePrice: 5},
		{ID: "d", ReferencePrice: 1_000}, {ID: "e", ReferencePrice: 100},
	}}
	duals := []steady.Frac{steady.Rat(100, 1), steady.Rat(100, 1), steady.Rat(100, 1), steady.Rat(0, 1), steady.Rat(10, 1)}
	rc := &regionCtx{
		s: &viaSolve{m: &viaModel{c: cfg}}, p: &viaProgramme{pr: steady.Outcome{Price: duals}}, id: "r",
		scale: steady.Rat(1, 1), price: make([]int64, 5), table: []int{0, 1, 2, 3, 4},
	}
	if err := rc.prices(); err != nil {
		t.Fatalf("prices = %v, want nil", err)
	}
	// Floor 50, band 10 % to 1 000 % of the reference.
	// a: 100 in the band 10 to 1 000, kept. b: 100 against ref 5 000 (500 to 50 000)
	// is lifted to 500. c: 100 against ref 5 (1 to 50) is capped to 50. d: free,
	// floored to 50 and lifted to 100 (the band of 1 000 starts at 100). e: dual 10
	// is floored to 50, inside its band.
	if want := []int64{100, 500, 50, 100, 50}; !slices.Equal(rc.price, want) {
		t.Errorf("seeds = %v, want %v", rc.price, want)
	}
	rep := rc.s.rep
	if want := []RegionCommodity{{"r", "d"}}; !reflect.DeepEqual(rep.UnpricedCommodities, want) {
		t.Errorf("UnpricedCommodities = %v, want %v", rep.UnpricedCommodities, want)
	}
	if want := []RegionCommodity{{"r", "b"}, {"r", "d"}, {"r", "e"}}; !reflect.DeepEqual(rep.Floored, want) {
		t.Errorf("Floored = %v, want %v (raised by the band or the floor; c was only capped)", rep.Floored, want)
	}
	if want := []PriceRecord{{"r", "b", 100, 500}, {"r", "c", 100, 50}, {"r", "d", 50, 100}}; !reflect.DeepEqual(rep.OutsideReferenceBand, want) {
		t.Errorf("OutsideReferenceBand = %v, want %v", rep.OutsideReferenceBand, want)
	}
	// (seed - ref) * 10 000 / ref: b -4 500 / 5 000, c 45 / 5, d -900 / 1 000, e -50 / 100.
	wantDrift := []DriftRow{{"r", "a", 0}, {"r", "b", -9_000}, {"r", "c", 90_000}, {"r", "d", -9_000}, {"r", "e", -5_000}}
	if !reflect.DeepEqual(rep.Drift, wantDrift) {
		t.Errorf("Drift = %v, want %v", rep.Drift, wantDrift)
	}
	if len(rc.s.ws.Prices) != 5 || rc.s.ws.Prices[1] != (SeedPrice{Region: "r", Commodity: "b", Price: 500}) {
		t.Errorf("Prices = %v, want 5 rows, the second b at 500", rc.s.ws.Prices)
	}
}

// TestWages verifies the starting wage of a region: half to even, a wage of 1 is
// a wage, 0 is none, and a wage above MaxMoney is ErrOverflow with its text.
//
// Guard: `w < 1` turned to `w <= 1` or `w < 2` loses a wage of 1; `!ok` emptied or
// its message cleared returns no error.
//
// Requirement: FR-VIA-050
func TestWages(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		dual steady.Frac
		want int64
	}{
		{"one", steady.Rat(1, 1), 1},
		{"half rounds to the even zero", steady.Rat(1, 2), 0},
		{"three halves round to the even two", steady.Rat(3, 2), 2},
		{"none", steady.Rat(0, 1), 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			rc := &regionCtx{s: &viaSolve{}, p: &viaProgramme{pr: steady.Outcome{Wage: []steady.Frac{tt.dual}}}, id: "r", scale: steady.Rat(1, 1)}
			if err := rc.wages(); err != nil || rc.wage != tt.want {
				t.Fatalf("wages(%s) = wage %d, %v, want %d, nil", tt.name, rc.wage, err, tt.want)
			}
			if tt.want == 0 {
				if !slices.Equal(rc.s.rep.UnpricedWages, []string{"r"}) || len(rc.s.ws.Wages) != 0 {
					t.Errorf("wages(%s) = unpriced %v, rows %v, want [r] and none", tt.name, rc.s.rep.UnpricedWages, rc.s.ws.Wages)
				}
				return
			}
			if want := []SeedWage{{Region: "r", Wage: Money(tt.want)}}; !reflect.DeepEqual(rc.s.ws.Wages, want) || len(rc.s.rep.UnpricedWages) != 0 {
				t.Errorf("wages(%s) = rows %v, unpriced %v, want %v and none", tt.name, rc.s.ws.Wages, rc.s.rep.UnpricedWages, want)
			}
		})
	}
	rc := &regionCtx{s: &viaSolve{}, p: &viaProgramme{pr: steady.Outcome{Wage: []steady.Frac{steady.Rat(MaxMoney+1, 1)}}}, id: "r", scale: steady.Rat(1, 1)}
	err := rc.wages()
	if !errors.Is(err, ErrOverflow) || !strings.Contains(err.Error(), "starting wage of region") || rc.wage != 0 || len(rc.s.ws.Wages) != 0 {
		t.Errorf("wages above MaxMoney = %v (wage %d), want ErrOverflow naming the wage and no row", err, rc.wage)
	}
}

// TestNewRegion_NoActivityIsMinusOne verifies the index of the activity of each
// recipe: a recipe with none in the region is -1.
//
// Guard: `-1` turned to `+1` (or to another number) indexes an activity that is
// not the recipe's.
//
// Requirement: FR-VIA-028
func TestNewRegion_NoActivityIsMinusOne(t *testing.T) {
	t.Parallel()
	c := viaToy()
	// A sited recipe with no yield profile for the north has no activity there.
	c.Recipes = append(c.Recipes, viaRecipe("spare", nil, []Output{viaOut("grain", 1)}, func(r *Recipe) { r.Sited = true }))
	rc := tblRegion(t, tblSolved(t, c))
	if want := []int{0, -1}; !slices.Equal(rc.actOf, want) {
		t.Errorf("actOf = %v, want %v", rc.actOf, want)
	}
}

// TestTableCommodities verifies the commodities that have a row: the upkeep
// (including a quantity of 1), an input, a tool and a non-discard output, and not
// a discard output on its own.
//
// Guard: `d > 0` turned to `d < 0` or `d > 1` drops an upkeep-only commodity.
//
// Requirement: FR-VIA-023
func TestTableCommodities(t *testing.T) {
	t.Parallel()
	m := &viaModel{upkeep: [][]int64{{1, 2, 0, 0, 0, 0, 0}}}
	rc := &regionCtx{s: &viaSolve{m: m}, price: make([]int64, 7)}
	rc.acts = []viaActivity{{act: steady.Activity{
		Inputs:  []steady.Item{{Commodity: 2}},
		Tools:   []steady.Tool{{Commodity: 3}},
		Outputs: []steady.Item{{Commodity: 4}, {Commodity: 5, Discard: true}, {Commodity: 1, Discard: true}},
	}}}
	rc.tableCommodities()
	// 0 and 1 are in the upkeep (1 and 2; the discard of 1 does not take it away), 2
	// is an input, 3 a tool, 4 an output; 5 is only a discard and 6 is nothing.
	if want := []int{0, 1, 2, 3, 4}; !slices.Equal(rc.table, want) {
		t.Errorf("table = %v, want %v", rc.table, want)
	}
}

// TestCost verifies the cost of a list and that either overflow is (0, false).
//
// Guard: the two `!ok` branches turned to false, emptied or returning true let a
// wrapped product through, and a non-zero value with the false is a number to use.
//
// Requirement: FR-VIA-032
func TestCost(t *testing.T) {
	t.Parallel()
	rc := &regionCtx{price: []int64{3, 1, 1 << 62}}
	tests := []struct {
		name  string
		items []steady.Item
		want  int64
		ok    bool
	}{
		// 2 * 3 + 5 * 1 = 11.
		{"sum", []steady.Item{{Commodity: 0, Qty: 2}, {Commodity: 1, Qty: 5}}, 11, true},
		{"product overflows", []steady.Item{{Commodity: 2, Qty: 5}}, 0, false},
		// Each product is 2^62 and fits; their sum 2^63 does not.
		{"sum overflows", []steady.Item{{Commodity: 2, Qty: 1}, {Commodity: 2, Qty: 1}}, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got, ok := rc.cost(tt.items); got != tt.want || ok != tt.ok {
				t.Errorf("cost(%s) = %d, %v, want %d, %v", tt.name, got, ok, tt.want, tt.ok)
			}
		})
	}
}

// TestMainRecipe_TieKeepsTheFirstAndOverflowIsAnError verifies that of two
// recipes tied in duration the dearer inputs win and not the later one, and that
// an overflowing cost is an error with the returned values -1, 0, 0.
//
// Guard: `cost > inputs` turned to `!=` takes a cheaper later recipe; the error
// branch emptied, its values changed or its message cleared hides an overflow.
//
// Requirement: FR-VIA-032
func TestMainRecipe_TieKeepsTheFirstAndOverflowIsAnError(t *testing.T) {
	t.Parallel()
	in := func(c int, q int64) []steady.Item { return []steady.Item{{Commodity: c, Qty: q}} }
	// Both take 3 ticks; the first costs 1 000 (1 of b), the second 200 (2 of a).
	rc, gc := moneyContext(steady.Activity{Duration: 3, Inputs: in(1, 1)}, steady.Activity{Duration: 3, Inputs: in(0, 2)})
	if main, dmax, cost, err := rc.mainRecipe(gc); err != nil || main != 0 || dmax != 3 || cost != 1_000 {
		t.Errorf("mainRecipe(dearer first) = %d, %d, %d, %v, want 0, 3, 1 000, nil", main, dmax, cost, err)
	}
	rc, gc = moneyContext(steady.Activity{Duration: 3, Inputs: in(1, 1<<60)}) // 1 000 * 2^60 leaves int64
	main, dmax, cost, err := rc.mainRecipe(gc)
	if !errors.Is(err, ErrOverflow) || !strings.Contains(err.Error(), `money of group "fabber" in region "r"`) || main != -1 || dmax != 0 || cost != 0 {
		t.Errorf("mainRecipe(overflow) = %d, %d, %d, %v, want -1, 0, 0 and ErrOverflow naming the group", main, dmax, cost, err)
	}
}

// TestMoney_OverflowsReturnZeroAndSayWhere verifies each overflow of the money:
// the main recipe's cost, the sum of the upkeep and the span, each as ErrOverflow
// with its own text and a money of 0.
//
// Guard: the mainRecipe error ignored, the upkeep sum's `!ok` turned to false, or
// a returned `0` turned to `1` pays out a number that was never computed.
//
// Requirement: FR-VIA-032
func TestMoney_OverflowsReturnZeroAndSayWhere(t *testing.T) {
	t.Parallel()
	t.Run("main recipe", func(t *testing.T) {
		t.Parallel()
		rc, gc := moneyContext(steady.Activity{Duration: 3, Inputs: []steady.Item{{Commodity: 1, Qty: 1 << 60}}})
		got, err := rc.money(gc)
		if !errors.Is(err, ErrOverflow) || got != 0 {
			t.Errorf("money with an overflowing recipe = %d, %v, want 0 and ErrOverflow", got, err)
		}
	})
	t.Run("upkeep product", func(t *testing.T) {
		t.Parallel()
		rc, gc := moneyContext()
		gc.up[0], rc.price[0] = 1<<40, 1<<40
		got, err := rc.money(gc)
		if !errors.Is(err, ErrOverflow) || !strings.Contains(err.Error(), "upkeep of group") || got != 0 {
			t.Errorf("money with an upkeep product of 2^80 = %d, %v, want 0 and ErrOverflow naming the upkeep", got, err)
		}
	})
	t.Run("upkeep sum", func(t *testing.T) {
		t.Parallel()
		rc, gc := moneyContext()
		gc.up[0], gc.up[1], rc.price[0], rc.price[1] = 1, 1, math.MaxInt64, 1 // each product fits, the sum does not
		got, err := rc.money(gc)
		if !errors.Is(err, ErrOverflow) || !strings.Contains(err.Error(), "upkeep of group") || got != 0 {
			t.Errorf("money with an upkeep sum above int64 = %d, %v, want 0 and ErrOverflow naming the upkeep", got, err)
		}
	})
	t.Run("span", func(t *testing.T) {
		t.Parallel()
		rc, gc := moneyContext()
		rc.price[0], rc.s.o.moneyBuffer = 1<<62, 10_000
		got, err := rc.money(gc)
		if !errors.Is(err, ErrOverflow) || !strings.Contains(err.Error(), `money of group "fabber" in region "r"`) || got != 0 {
			t.Errorf("money with U * span above int64 = %d, %v, want 0 and ErrOverflow naming the group", got, err)
		}
	})
}

// TestCover verifies the cover of a commodity and its cut to the shelf life: a
// cover equal to the life is not cut, a longer one is and is listed, a life that
// overflows cuts nothing, and an overflowing cover is not ok.
//
// Guard: `cover > life` turned to `>=` lists a cover that was not cut.
//
// Requirement: FR-VIA-039
func TestCover(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		up       int64
		shelf    int64
		want     int64
		ok       bool
		shortend bool
	}{
		// The cover is 5 * 1 = 5.
		{"life equal to the cover", 1, 5, 5, true, false},
		{"life below the cover", 1, 4, 4, true, true},
		{"no shelf life", 1, 0, 5, true, false},
		// 5 * 2^40 fits; the life 2^40 * 2^30 does not, so nothing is cut.
		{"life overflows", 1 << 40, 1 << 30, 5 << 40, true, false},
		{"cover overflows", 1 << 62, 1, 0, false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg := &Config{Commodities: []Commodity{{ID: "c", ShelfLife: tt.shelf}}}
			rc := &regionCtx{s: &viaSolve{m: &viaModel{c: cfg}}, id: "r"}
			rc.s.o.stockCover = 5
			got, ok := rc.cover(&groupCtx{up: []int64{tt.up}}, 0)
			if got != tt.want || ok != tt.ok {
				t.Errorf("cover(%s) = %d, %v, want %d, %v", tt.name, got, ok, tt.want, tt.ok)
			}
			if listed := len(rc.s.rep.CapShortened) == 1; listed != tt.shortend {
				t.Errorf("cover(%s) CapShortened = %v, want listed = %v", tt.name, rc.s.rep.CapShortened, tt.shortend)
			}
		})
	}
}

// tblReservesCtx returns a region of one group of 10 households with three
// commodities: a, a spring crop (the season gap of TestReserveTotal; the shelf
// life is a parameter), b, which no activity makes, and c, made every tick by a
// 3-tick recipe, which has a first-completion shortfall but no supply gap.
func tblReservesCtx(shelfA int64) *regionCtx {
	cfg := &Config{Commodities: []Commodity{{ID: "a", ShelfLife: shelfA}, {ID: "b", ShelfLife: 5}, {ID: "c"}}}
	spring := viaActivity{
		recipe: 0, runs: []steady.Run{{Length: 59}, {Length: 92, FactorBP: 10_000}, {Length: 214}},
		count: steady.StartCount{Starts: 2, FactorSum: 20_000, Completions: []int64{118, 209}},
		act:   steady.Activity{Duration: 60, Starts: 2, FactorSum: 20_000, Outputs: []steady.Item{{Commodity: 0, Qty: 1_800}}},
	}
	slow := viaActivity{
		recipe: 1, allYear: true, runs: []steady.Run{{Length: 365, FactorBP: 10_000}},
		count: steady.StartCount{Starts: 365, FactorSum: 3_650_000},
		act:   steady.Activity{Duration: 3, Starts: 365, FactorSum: 3_650_000, Outputs: []steady.Item{{Commodity: 2, Qty: 4}}},
	}
	m := &viaModel{c: cfg, year: 365, upkeep: [][]int64{{10, 10, 4}}, acts: [][]viaActivity{{spring, slow}}}
	p := &viaProgramme{base: []int{0}, out: steady.Outcome{Flow: []steady.Frac{steady.Rat(1, 2), steady.Rat(1, 1)}, Active: []bool{true, true}}}
	return &regionCtx{
		s: &viaSolve{m: m}, p: p, id: "r", acts: m.acts[0],
		groups: []groupCtx{{g: &Group{Households: 10}, reserve: make([]int64, 3), active: []activeRecipe{{i: 0, share: steady.Rat(1, 2)}, {i: 1, share: steady.Rat(1, 1)}}}},
	}
}

// TestReserves verifies the lean reserve of a region: only a commodity that is
// made with a supply gap has one, a commodity no activity makes has none, nor one
// made every tick, and a reserve cut to the shelf life is listed.
//
// Guard: `len(prods[c]) == 0` turned to `<` reserves an imported commodity,
// `Gap == 0` turned to `<` reserves one made every tick, and the `if cut` body
// emptied or its assignment removed loses the CapShortened row.
//
// Requirement: FR-VIA-039
func TestReserves(t *testing.T) {
	t.Parallel()
	t.Run("no shelf life", func(t *testing.T) {
		t.Parallel()
		rc := tblReservesCtx(0)
		rc.reserves()
		// a: the whole use of 10 a tick until tick 118 is 1 180, 118 for each of the 10
		// households. b has no maker. c is made every tick: no gap, so no reserve (it
		// would be ceil(8 / 10) = 1: 4 a tick for the 2 ticks before the first
		// completion).
		if want := []int64{118, 0, 0}; !slices.Equal(rc.groups[0].reserve, want) || len(rc.s.rep.CapShortened) != 0 {
			t.Errorf("reserve = %v, CapShortened = %v, want %v and none", rc.groups[0].reserve, rc.s.rep.CapShortened, want)
		}
	})
	t.Run("cut to the shelf life", func(t *testing.T) {
		t.Parallel()
		rc := tblReservesCtx(5)
		rc.reserves()
		// 5 ticks of the use of 10 is 50: 5 for each household. b (shelf life 5) has no
		// maker and is not listed.
		if want := []int64{5, 0, 0}; !slices.Equal(rc.groups[0].reserve, want) {
			t.Errorf("reserve = %v, want %v", rc.groups[0].reserve, want)
		}
		if want := []RegionCommodity{{"r", "a"}}; !reflect.DeepEqual(rc.s.rep.CapShortened, want) {
			t.Errorf("CapShortened = %v, want %v", rc.s.rep.CapShortened, want)
		}
	})
}

// TestReserveTotal_Bounds verifies the lean reserve against its edge cases: an
// inactive activity adds no use, a source of one unit a tick counts, a total
// above MaxInt64 / MaxHouseholds is kept, and the cut to the shelf life at 1,
// at the total exactly and above the same bound.
//
// Guard: `!rc.activeAt(i)` turned to false adds the use of a stopped recipe;
// `y > 0` turned to `y > 1` drops a source of 1; `maxInt64-MaxHouseholds` turned
// to `/` loses a large total; `shelf > 0` to `> 1` and `total > life` to `>=` move
// the cut.
//
// Requirement: FR-VIA-039
func TestReserveTotal_Bounds(t *testing.T) {
	t.Parallel()
	spring := viaActivity{
		runs: []steady.Run{{Length: 59}, {Length: 92, FactorBP: 10_000}, {Length: 214}},
		act:  steady.Activity{Duration: 60, Starts: 2, FactorSum: 20_000, Outputs: []steady.Item{{Commodity: 0, Qty: 1_800}}},
	}
	consumer := viaActivity{
		runs: []steady.Run{{Length: 365, FactorBP: 10_000}},
		act:  steady.Activity{Duration: 1, Starts: 365, FactorSum: 3_650_000, Inputs: []steady.Item{{Commodity: 0, Qty: 3}}},
	}
	unit := viaActivity{
		runs: []steady.Run{{Length: 365, FactorBP: 10_000}},
		act:  steady.Activity{Duration: 1, Starts: 365, FactorSum: 3_650_000, Outputs: []steady.Item{{Commodity: 0, Qty: 1}}},
	}
	half, one := steady.Rat(1, 2), steady.Rat(1, 1)
	tests := []struct {
		name   string
		shelf  int64
		upkeep int64
		acts   []viaActivity
		flows  []steady.Frac
		active []bool
		total  int64
		cut    bool
	}{
		// The use of 10 a tick until the harvest at tick 118 is 1 180 (TestReserveTotal);
		// the stopped consumer's 3 per execution would add to it.
		{"inactive activity", 0, 10, []viaActivity{spring, consumer}, []steady.Frac{half, one}, []bool{true, false}, 1_180, false},
		// 10 executions a tick of 1 unit make 10 a tick against a use of 10: no shortfall.
		{"source of one unit", 0, 10, []viaActivity{unit}, []steady.Frac{steady.Rat(10, 1)}, []bool{true}, 0, false},
		// The use is 10^9 a tick for 365 ticks and the harvest brings 0.5 * 365 * 1 800
		// = 328 500 back in total: 365 * 10^9 - 328 500, which is above MaxInt64 / 10^9.
		{"total above the divided bound", 0, 1_000_000_000, []viaActivity{spring}, []steady.Frac{half}, []bool{true}, 364_999_671_500, false},
		// 1 tick of a use of 10 is 10, below the total 1 180.
		{"shelf life of one", 1, 10, []viaActivity{spring}, []steady.Frac{half}, []bool{true}, 10, true},
		// 118 ticks of 10 is exactly the total: no cut.
		{"shelf life equal to the total", 118, 10, []viaActivity{spring}, []steady.Frac{half}, []bool{true}, 1_180, false},
		// 10 ticks of 10^9 is 10^10, above MaxInt64 / 10^9 and below the total.
		{"life above the divided bound", 10, 1_000_000_000, []viaActivity{spring}, []steady.Frac{half}, []bool{true}, 10_000_000_000, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg := &Config{Commodities: []Commodity{{ID: "g", ShelfLife: tt.shelf}}}
			m := &viaModel{c: cfg, year: 365, upkeep: [][]int64{{tt.upkeep}}}
			p := &viaProgramme{base: []int{0}, out: steady.Outcome{Flow: tt.flows, Active: tt.active}}
			rc := &regionCtx{s: &viaSolve{m: m}, p: p, acts: tt.acts}
			if total, cut := rc.reserveTotal(0); total != tt.total || cut != tt.cut {
				t.Errorf("reserveTotal(%s) = %d, %v, want %d, %v", tt.name, total, cut, tt.total, tt.cut)
			}
		})
	}
}

// tblJobsCtx returns a region with one group whose active recipes j0, j1, ...
// run all year, each of the given flow and with the given count of virtual start
// offsets, over one household that lists each. An executions-in-progress share is
// then flow * v * 10 000 basis points (365 * v / 365).
func tblJobsCtx(flows []steady.Frac, v map[RecipeID]int64) (*regionCtx, *groupCtx) {
	cfg := &Config{}
	m := &viaModel{c: cfg, year: 365}
	rc := &regionCtx{s: &viaSolve{m: m}, nVirt: v, listing: make([]int64, len(flows))}
	p := &viaProgramme{base: []int{0}, out: steady.Outcome{Flow: flows, Active: make([]bool, len(flows))}}
	rc.p = p
	gc := &groupCtx{}
	for i := range flows {
		cfg.Recipes = append(cfg.Recipes, Recipe{ID: "j" + string(rune('0'+i))})
		rc.acts = append(rc.acts, viaActivity{recipe: RecipeID(i), runs: []steady.Run{{Length: 365, FactorBP: 10_000}}, act: steady.Activity{Duration: 2}})
		rc.listing[i] = 1
		gc.active = append(gc.active, activeRecipe{i: i})
	}
	return rc, gc
}

// TestJobs_Edges verifies the jobs in progress of a group at the edges of 8.5.1:
// a recipe with no virtual start, an overflowing denominator, shares that round
// to 0 and 1, a share beyond int64, a sum of exactly 10 000, and a scale-down that
// drops the shares that fall to 0.
//
// Guard: `continue` turned to `break` stops at the first skipped recipe; the
// `!ok` check removed divides by 0; `Floor(0, ...)` to `Floor(1, ...)` makes a
// job of 10 000 out of a share of 0; `bp > 0` to `bp > 1` or `>= 0`, `sum >
// wholeBP` to `>=`, `InProgressBP > 0` to `>= 0` or `> 1`, the `fits` branch
// emptied and `jobs = scaled` removed each change a job or the Short flag.
//
// Requirement: FR-VIA-033, FR-VIA-034
func TestJobs_Edges(t *testing.T) {
	t.Parallel()
	r := steady.Rat
	tests := []struct {
		name  string
		flows []steady.Frac
		v     map[RecipeID]int64
		edit  func(*regionCtx)
		want  []JobSpec
		short bool
	}{
		{"no virtual start is skipped, the next kept", []steady.Frac{r(1, 1), r(1, 2)}, map[RecipeID]int64{0: 0, 1: 1}, nil,
			[]JobSpec{{Recipe: "j1", InProgressBP: 5_000}}, false},
		// No virtual start and no household that lists the recipe: the denominator is 0
		// too, and the recipe is skipped before it is divided by.
		{"no virtual start and no household", []steady.Frac{r(1, 1), r(1, 2)}, map[RecipeID]int64{0: 0, 1: 1},
			func(rc *regionCtx) { rc.listing[0] = 0 },
			[]JobSpec{{Recipe: "j1", InProgressBP: 5_000}}, false},
		// The first recipe's denominator open ticks * households = 2^40 * 2^40 overflows.
		{"overflowing denominator is skipped, the next kept", []steady.Frac{r(1, 1), r(1, 2)}, map[RecipeID]int64{0: 1, 1: 1},
			func(rc *regionCtx) {
				rc.acts[0].runs = []steady.Run{{Length: 1 << 40, FactorBP: 10_000}}
				rc.listing[0] = 1 << 40
			},
			[]JobSpec{{Recipe: "j1", InProgressBP: 5_000}}, false},
		// 1/20 000 * 1 * 10 000 = 0.5 basis points, rounded down to 0: no job.
		{"share of 0", []steady.Frac{r(1, 20_000)}, map[RecipeID]int64{0: 1}, nil, nil, false},
		{"share of 1", []steady.Frac{r(1, 10_000)}, map[RecipeID]int64{0: 1}, nil, []JobSpec{{Recipe: "j0", InProgressBP: 1}}, false},
		// 2^62 * 10 000 leaves int64: clamped to 10 000, a sum of exactly 10 000.
		{"share beyond int64", []steady.Frac{r(1<<62, 1)}, map[RecipeID]int64{0: 1}, nil, []JobSpec{{Recipe: "j0", InProgressBP: 10_000}}, false},
		{"sum of exactly 10 000", []steady.Frac{r(1, 2), r(1, 2)}, map[RecipeID]int64{0: 1, 1: 1}, nil,
			[]JobSpec{{Recipe: "j0", InProgressBP: 5_000}, {Recipe: "j1", InProgressBP: 5_000}}, false},
		// 10 000, 2 and 1 sum to 10 003: scaled to 10 000 * 10 000 / 10 003 = 9 997.0,
		// 2 * 10 000 / 10 003 = 1.99 -> 1, and 1 * 10 000 / 10 003 = 0.99 -> 0, dropped.
		{"scale-down drops the zero share", []steady.Frac{r(1, 1), r(2, 10_000), r(1, 10_000)}, map[RecipeID]int64{0: 1, 1: 1, 2: 1}, nil,
			[]JobSpec{{Recipe: "j0", InProgressBP: 9_997}, {Recipe: "j1", InProgressBP: 1}}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			rc, gc := tblJobsCtx(tt.flows, tt.v)
			if tt.edit != nil {
				tt.edit(rc)
			}
			if got := rc.jobs(gc); !slices.Equal(got, tt.want) || gc.short != tt.short {
				t.Errorf("jobs(%s) = %v, short %v, want %v, short %v", tt.name, got, gc.short, tt.want, tt.short)
			}
		})
	}
}

// TestOutQty verifies the quantity one execution outputs: the sum of the
// non-discard outputs of the commodity.
//
// Guard: `q += o.Qty` turned to `q = o.Qty` keeps the last of several outputs.
//
// Requirement: FR-VIA-039
func TestOutQty(t *testing.T) {
	t.Parallel()
	a := steady.Activity{Outputs: []steady.Item{{Commodity: 0, Qty: 3}, {Commodity: 0, Qty: 4}, {Commodity: 1, Qty: 5}, {Commodity: 0, Qty: 2, Discard: true}}}
	if got := outQty(a, 0); got != 7 {
		t.Errorf("outQty(0) = %d, want 7 (3 + 4; the discard and commodity 1 do not count)", got)
	}
	if got := outQty(a, 2); got != 0 {
		t.Errorf("outQty(2) = %d, want 0", got)
	}
}

// TestOpenTicks verifies the count of ticks with a positive factor, a factor of 1
// included.
//
// Guard: `FactorBP > 0` turned to `> 1` drops a run at the lowest positive yield.
//
// Requirement: FR-VIA-033
func TestOpenTicks(t *testing.T) {
	t.Parallel()
	runs := []steady.Run{{Length: 10, FactorBP: 1}, {Length: 20}, {Length: 30, FactorBP: 10_000}}
	if got := openTicks(runs); got != 40 {
		t.Errorf("openTicks = %d, want 40 (10 at 1 bp and 30 at 10 000; the 20 closed ticks do not count)", got)
	}
}

// TestFinish_ChecksTheTable verifies that finish returns the error of a table
// that breaks a structural bound, and the error of a job that no virtual start
// can place, which only the configuration with the table in it can see.
//
// Guard: `mm.c = &full` removed makes emptyOffsets read a configuration with no
// table; `len(v.rep.errs) == 0` turned to false, `<` or `> 0` never runs it; the
// call removed does not run it.
//
// Requirement: FR-VIA-030, FR-VIA-033
func TestFinish_ChecksTheTable(t *testing.T) {
	t.Parallel()
	t.Run("over a bound", func(t *testing.T) {
		t.Parallel()
		s := tblSolved(t, viaToy())
		if err := s.build(); err != nil {
			t.Fatalf("build = %v, want nil", err)
		}
		s.ws.Prices[0].Price = MaxPrice + 1
		if st, err := s.finish(); err == nil || !reflect.DeepEqual(st, SteadyState{}) {
			t.Errorf("finish with a price above MaxPrice = %v, %v, want an error and no table", st, err)
		}
	})
	t.Run("job with no virtual start", func(t *testing.T) {
		t.Parallel()
		c := viaToy()
		c.Recipes[0].Duration, c.Recipes[0].Outputs[0].Quantity = 2, 20
		c.Yields = []YieldProfile{viaSpring("north", "farm")}
		s := tblSolved(t, c)
		if err := s.build(); err != nil {
			t.Fatalf("build = %v, want nil", err)
		}
		if len(s.ws.Groups[0].Jobs) != 0 {
			t.Fatalf("Jobs = %v, want none: the spring recipe has no virtual start", s.ws.Groups[0].Jobs)
		}
		s.ws.Groups[0].Jobs = []JobSpec{{Recipe: "farm", InProgressBP: 5_000}}
		if st, err := s.finish(); !errors.Is(err, ErrWarmStartJob) || !reflect.DeepEqual(st, SteadyState{}) {
			t.Errorf("finish with a job at an unplaceable recipe = %v, %v, want ErrWarmStartJob and no table", st, err)
		}
	})
}

// TestReport_Normalise verifies that every list of the report is sorted by region then
// commodity or recipe and has its repeats dropped, field by field, and that the
// census orders a role and a recipe by the text "role/recipe".
//
// Guard: each `r.X = sortedUnique(...)` removed leaves that list as visited; the
// two "/" of the census key turned to "" make "a" + "z" sort after "a0".
//
// Requirement: FR-VIA-029
func TestReport_Normalise(t *testing.T) {
	t.Parallel()
	rcs := func(r, c string) RegionCommodity { return RegionCommodity{r, c} }
	// Unsorted with a repeat: the region decides first (q before r), then the
	// commodity (a before b).
	unsorted := func() Report {
		return Report{
			UnpricedCommodities:  []RegionCommodity{rcs("r", "b"), rcs("q", "z"), rcs("r", "a"), rcs("r", "a")},
			UnpricedWages:        []string{"r", "q", "r"},
			Floored:              []RegionCommodity{rcs("r", "b"), rcs("q", "z"), rcs("r", "a"), rcs("r", "a")},
			OutsideReferenceBand: []PriceRecord{{"r", "b", 1, 2}, {"q", "z", 1, 2}, {"r", "a", 1, 2}, {"r", "a", 1, 2}},
			SeedBelowCostFloor:   []CostRecord{{"r", "b", 1, 2}, {"q", "z", 1, 2}, {"r", "a", 1, 2}, {"r", "a", 1, 2}},
			SeedAboveCostCeiling: []CostRecord{{"r", "b", 1, 2}, {"q", "z", 1, 2}, {"r", "a", 1, 2}, {"r", "a", 1, 2}},
			Drift:                []DriftRow{{"r", "b", 1}, {"q", "z", 1}, {"r", "a", 1}, {"r", "a", 1}},
			ResidualOver:         []ResidualRow{{"r", "b", 1}, {"q", "z", 1}, {"r", "a", 1}, {"r", "a", 1}},
			CapacityTight:        []RegionRecipe{{"r", "b"}, {"q", "z"}, {"r", "a"}, {"r", "a"}},
			CapShortened:         []RegionCommodity{rcs("r", "b"), rcs("q", "z"), rcs("r", "a"), rcs("r", "a")},
			NominalSeed:          []string{"r", "q", "r"},
			Census:               []CensusRow{{Region: "r", Role: "b"}, {Region: "q", Role: "z"}, {Region: "r", Role: "a"}, {Region: "r", Role: "a"}},
			RegionSurplusBP:      []RegionSurplus{{Region: "r"}, {Region: "q"}, {Region: "r"}},
			WageBelowTarget:      []RoleWage{{"r", "b", 1, 2}, {"q", "z", 1, 2}, {"r", "a", 1, 2}, {"r", "a", 1, 2}},
		}
	}
	want := Report{
		UnpricedCommodities:  []RegionCommodity{rcs("q", "z"), rcs("r", "a"), rcs("r", "b")},
		UnpricedWages:        []string{"q", "r"},
		Floored:              []RegionCommodity{rcs("q", "z"), rcs("r", "a"), rcs("r", "b")},
		OutsideReferenceBand: []PriceRecord{{"q", "z", 1, 2}, {"r", "a", 1, 2}, {"r", "b", 1, 2}},
		SeedBelowCostFloor:   []CostRecord{{"q", "z", 1, 2}, {"r", "a", 1, 2}, {"r", "b", 1, 2}},
		SeedAboveCostCeiling: []CostRecord{{"q", "z", 1, 2}, {"r", "a", 1, 2}, {"r", "b", 1, 2}},
		Drift:                []DriftRow{{"q", "z", 1}, {"r", "a", 1}, {"r", "b", 1}},
		ResidualOver:         []ResidualRow{{"q", "z", 1}, {"r", "a", 1}, {"r", "b", 1}},
		CapacityTight:        []RegionRecipe{{"q", "z"}, {"r", "a"}, {"r", "b"}},
		CapShortened:         []RegionCommodity{rcs("q", "z"), rcs("r", "a"), rcs("r", "b")},
		NominalSeed:          []string{"q", "r"},
		Census:               []CensusRow{{Region: "q", Role: "z"}, {Region: "r", Role: "a"}, {Region: "r", Role: "b"}},
		RegionSurplusBP:      []RegionSurplus{{Region: "q"}, {Region: "r"}},
		WageBelowTarget:      []RoleWage{{"q", "z", 1, 2}, {"r", "a", 1, 2}, {"r", "b", 1, 2}},
	}
	got := unsorted()
	got.normalise()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("normalise =\n%+v\nwant\n%+v", got, want)
	}
	// Role "a" with recipe "z" is "a/z"; role "a0" with no recipe is "a0/". '/' is
	// below '0', so the first sorts first; without the separators "az" sorts after "a0".
	x, y := CensusRow{Region: "r", Role: "a", Recipe: "z"}, CensusRow{Region: "r", Role: "a0"}
	for _, in := range [][]CensusRow{{x, y}, {y, x}} {
		r := Report{Census: slices.Clone(in)}
		r.normalise()
		if !reflect.DeepEqual(r.Census, []CensusRow{x, y}) {
			t.Errorf("normalise(census %v) = %v, want %v then %v", in, r.Census, x, y)
		}
	}
}

// TestSortedUnique verifies that an empty list is nil, a list is sorted and its
// repeats are dropped.
//
// Guard: `len(s) == 0` turned to false or `<` returns an empty non-nil list;
// `slices.SortStableFunc` removed leaves the order as given.
//
// Requirement: FR-VIA-029
func TestSortedUnique(t *testing.T) {
	t.Parallel()
	if got := sortedUnique([]int{}, cmp.Compare[int]); got != nil {
		t.Errorf("sortedUnique(empty) = %#v, want nil", got)
	}
	if got := sortedUnique([]int{3, 1, 3, 2, 1}, cmp.Compare[int]); !slices.Equal(got, []int{1, 2, 3}) {
		t.Errorf("sortedUnique(3 1 3 2 1) = %v, want [1 2 3]", got)
	}
}

// tblResidualCtx returns a region of one commodity priced 100 and the given
// wage, whose activities each make `out` units of it in one tick (a value added
// of 100 * out) and of which the first `active` ... are given by the flags.
func tblResidualCtx(wage int64, rent steady.Frac, warn int64, outs []int64, active []bool) *regionCtx {
	cfg := &Config{}
	rc := &regionCtx{
		s: &viaSolve{m: &viaModel{c: cfg}}, p: &viaProgramme{base: []int{0}}, id: "r", wage: wage, scale: steady.Rat(1, 1), price: []int64{100},
	}
	rc.s.o.residualWarn = warn
	rc.p.pr.Active = active
	for i, q := range outs {
		cfg.Recipes = append(cfg.Recipes, Recipe{ID: "x" + string(rune('0'+i))})
		rc.acts = append(rc.acts, viaActivity{recipe: RecipeID(i), act: steady.Activity{
			Duration: 1, Starts: 1, FactorSum: 10_000, Outputs: []steady.Item{{Commodity: 0, Qty: q}},
		}})
		rc.p.pr.Rent = append(rc.p.pr.Rent, rent)
	}
	return rc
}

// TestResidual verifies the residual of the activities: the largest, the rows
// above the warning only, an inactive activity and a region with no wage skipped
// without stopping the loop, a wage of 1 computed, and a residual beyond int64
// as MaxInt64.
//
// Guard: `continue` turned to `break` stops at an inactive activity; `bp >
// residualWarn` turned to `!=` lists a residual at or below the limit; `rc.wage
// == 0` turned to `<` or `== 1` computes with no wage or skips a wage of 1; the
// `!fits` branch emptied or its assignment removed lets a wrapped value through.
//
// Requirement: FR-VIA-025
func TestResidual(t *testing.T) {
	t.Parallel()
	zero := steady.Rat(0, 1)
	t.Run("rows above the warning", func(t *testing.T) {
		t.Parallel()
		// x0 is inactive and would be far off. Against a wage of 200: x1 makes 4 (value
		// 400), |400 - 200| / 200 = 10 000 bp; x2 makes 3 (300), 5 000 bp, which is the
		// limit and not above; x3 makes 2 (200), 0 bp.
		rc := tblResidualCtx(200, zero, 5_000, []int64{40, 4, 3, 2}, []bool{false, true, true, true})
		rc.residual()
		if want := []ResidualRow{{"r", "x1", 10_000}}; !reflect.DeepEqual(rc.s.rep.ResidualOver, want) || rc.s.rep.ResidualBP != 10_000 {
			t.Errorf("residual = %v, largest %d, want %v and 10 000", rc.s.rep.ResidualOver, rc.s.rep.ResidualBP, want)
		}
	})
	t.Run("region with no wage", func(t *testing.T) {
		t.Parallel()
		// The rent 100 would make a target of 100 and a residual of 30 000 if a wage of
		// 0 were not skipped.
		rc := tblResidualCtx(0, steady.Rat(100, 1), 5_000, []int64{4}, []bool{true})
		rc.residual()
		if len(rc.s.rep.ResidualOver) != 0 || rc.s.rep.ResidualBP != 0 {
			t.Errorf("residual with no wage = %v, %d, want none", rc.s.rep.ResidualOver, rc.s.rep.ResidualBP)
		}
	})
	t.Run("wage of one", func(t *testing.T) {
		t.Parallel()
		// Value 400 against a target of 1: 399 * 10 000 = 3 990 000 bp.
		rc := tblResidualCtx(1, zero, 5_000, []int64{4}, []bool{true})
		rc.residual()
		if want := []ResidualRow{{"r", "x0", 3_990_000}}; !reflect.DeepEqual(rc.s.rep.ResidualOver, want) || rc.s.rep.ResidualBP != 3_990_000 {
			t.Errorf("residual = %v, largest %d, want %v and 3 990 000", rc.s.rep.ResidualOver, rc.s.rep.ResidualBP, want)
		}
	})
	t.Run("beyond int64", func(t *testing.T) {
		t.Parallel()
		// Value 10^9 * 10^9 = 10^18 against a target of 1: 10^22 bp.
		rc := tblResidualCtx(1, zero, 5_000, []int64{1_000_000_000}, []bool{true})
		rc.price[0] = 1_000_000_000
		rc.residual()
		if want := []ResidualRow{{"r", "x0", math.MaxInt64}}; !reflect.DeepEqual(rc.s.rep.ResidualOver, want) || rc.s.rep.ResidualBP != math.MaxInt64 {
			t.Errorf("residual = %v, largest %d, want %v and MaxInt64", rc.s.rep.ResidualOver, rc.s.rep.ResidualBP, want)
		}
	})
}

// tblCostCtx returns a region whose one recipe, listed there and started once a
// year at a mean factor of 10 000, makes the commodities of outs; the wage is the
// given one, the duration 1 tick and the seed prices those given.
func tblCostCtx(wage int64, prices []int64, outs []outputRow, floor, ceiling int64) *regionCtx {
	cfg := &Config{World: World{CostFloor: floor, CostCeiling: ceiling}, Commodities: make([]Commodity, len(prices)), Recipes: []Recipe{{ID: "mk"}}}
	for i := range cfg.Commodities {
		cfg.Commodities[i].ID = string(rune('a' + i))
	}
	m := &viaModel{c: cfg, listed: [][]bool{{true}}, rt: recipeTable{rows: []recipeRow{{outputs: outs}}}}
	return &regionCtx{
		s: &viaSolve{m: m}, id: "r", wage: wage, price: prices,
		acts: []viaActivity{{count: steady.StartCount{Starts: 1, FactorSum: 10_000}, act: steady.Activity{Duration: 1}}},
	}
}

// TestCostBand verifies the seeds outside the cost band: below and above, the
// edges that are inside, a seed of 0 or an unknown cost that is skipped without
// stopping the loop, a unit cost and a seed of 1 that count, and each overflow.
//
// Guard: `have < lo` to `<=` or `have > hi` to `>=` list an edge; `unit == 0 ||
// seed == 0` to `&&`, to a compare against 1 or to `<` lists an unpriced good or
// skips a cost of 1; `continue` to `break` stops the scan; `!ok || !ok1 || !ok2`
// to `&&` keeps a wrapped product.
//
// Requirement: FR-VIA-027, FR-VIA-054
func TestCostBand(t *testing.T) {
	t.Parallel()
	// Wage 400, 4 units an execution: a share of 10 000 basis points costs
	// 400 * 10 000 / (4 * 10 000) = 100 a unit, and a share of 1 costs ceil(0.01) = 1.
	out := func(c CommodityID, share int64) outputRow {
		return outputRow{commodity: c, quantity: 4, costShare: share}
	}
	t.Run("band", func(t *testing.T) {
		t.Parallel()
		// Floor 50 % and ceiling 200 % of 100: below 50, above 200.
		outs := []outputRow{out(0, 10_000), out(1, 10_000), out(2, 10_000), out(3, 10_000), out(4, 10_000), out(6, 10_000), out(7, 1), out(8, 10_000)}
		rc := tblCostCtx(400, []int64{49, 50, 200, 201, 0, 500, 300, 3, 1}, outs, 5_000, 20_000)
		rc.costBand()
		// a 49 is below; b 50 and c 200 are on the edges; d 201 is above; e has no seed;
		// f (500) has no output, so no cost; g 300 is above; h has a cost of 1 (share 1)
		// and a seed of 3 against a ceiling of 2: above; i has a seed of 1: below.
		wantBelow := []CostRecord{{"r", "a", 49, 100}, {"r", "i", 1, 100}}
		wantAbove := []CostRecord{{"r", "d", 201, 100}, {"r", "g", 300, 100}, {"r", "h", 3, 1}}
		rep := rc.s.rep
		if !reflect.DeepEqual(rep.SeedBelowCostFloor, wantBelow) || !reflect.DeepEqual(rep.SeedAboveCostCeiling, wantAbove) {
			t.Errorf("costBand = below %v, above %v, want %v and %v", rep.SeedBelowCostFloor, rep.SeedAboveCostCeiling, wantBelow, wantAbove)
		}
	})
	overflows := []struct {
		name            string
		prices          []int64
		floor, ceiling  int64
		wantBelow       []CostRecord
		wantAboveRecord bool
	}{
		// 2^60 * 10 000 leaves int64 for a, then b is below as usual.
		{"seed times whole", []int64{1 << 60, 1}, 5_000, 20_000, []CostRecord{{"r", "b", 1, 100}}, false},
		{"unit times floor", []int64{3_000, 1}, 1 << 62, 20_000, nil, false},
		{"unit times ceiling", []int64{3_000, 1}, 5_000, 1 << 62, nil, false},
	}
	for _, tt := range overflows {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			rc := tblCostCtx(400, tt.prices, []outputRow{out(0, 10_000), out(1, 10_000)}, tt.floor, tt.ceiling)
			rc.costBand()
			rep := rc.s.rep
			if !reflect.DeepEqual(rep.SeedBelowCostFloor, tt.wantBelow) || len(rep.SeedAboveCostCeiling) != 0 {
				t.Errorf("costBand(%s) = below %v, above %v, want below %v and none above", tt.name, rep.SeedBelowCostFloor, rep.SeedAboveCostCeiling, tt.wantBelow)
			}
		})
	}
}

// TestUnitCosts verifies the least unit cost over the recipes of a region: a
// lower later cost wins, an unknown cost never wipes a known one, a start count
// of 1 counts and of 0 is skipped, an unlisted recipe is skipped without
// stopping the scan, and a discarded output has none.
//
// Guard: `u < best` to `==` keeps the first; `u > 0` to `>= 0` lets 0 replace a
// cost; `Starts < 1` to `< 2` or `< 0` skips a single start or divides by 0;
// `!listed ||` and the output `continue`s to `break` stop early; `o.discard ||`
// to `&&` costs a discard.
//
// Requirement: FR-VIA-027
func TestUnitCosts(t *testing.T) {
	t.Parallel()
	cfg := &Config{Commodities: []Commodity{{ID: "a"}, {ID: "b"}, {ID: "c"}}, Recipes: make([]Recipe, 7)}
	one := func(share int64) []outputRow { return []outputRow{{commodity: 0, quantity: 4, costShare: share}} }
	m := &viaModel{
		c: cfg,
		// r3 is on no role's list. r5 makes a discarded b and then c.
		listed: [][]bool{{true, true, true, false, true, true, true}},
		rt: recipeTable{rows: []recipeRow{
			{outputs: one(10_000)}, // 400 * 10 000 / (4 * 10 000) = 100
			{outputs: one(10_000), inputs: []itemRow{{commodity: 1, quantity: 1}}},
			{outputs: one(5_000)}, // 50
			{outputs: one(1)},
			{outputs: one(10_000)},
			{outputs: []outputRow{{commodity: 1, quantity: 4, costShare: 10_000, discard: true}, {commodity: 2, quantity: 4, costShare: 10_000}}},
			{outputs: one(10_000)},
		}},
	}
	st := steady.StartCount{Starts: 1, FactorSum: 10_000}
	act := func(recipe RecipeID, count steady.StartCount, in ...steady.Item) viaActivity {
		return viaActivity{recipe: recipe, count: count, act: steady.Activity{Duration: 1, Inputs: in}}
	}
	rc := &regionCtx{s: &viaSolve{m: m}, wage: 400, price: []int64{0, 0, 0}, acts: []viaActivity{
		act(3, st),                  // unlisted: skipped first, the scan goes on
		act(0, st),                  // 100
		act(2, st),                  // 50: lower, later
		act(4, steady.StartCount{}), // no start: skipped, no division by 0
		act(5, st),                  // discarded b, then c at 100
		act(6, steady.StartCount{Starts: 1, FactorSum: 20_000}),
		// An input of unknown price: cost 0, which must not wipe the 50 (last, so
		// nothing after it could put it back).
		act(1, st, steady.Item{Commodity: 1, Qty: 1}),
	}}
	// a: 100 (r0), 50 (r2), r6 has a yield of 20 000: 400 * 10 000 / (4 * 20 000) =
	// 50, not below 50, and r1 is unknown. b: only a discard. c: 100.
	if got, want := rc.unitCosts(), []int64{50, 0, 100}; !slices.Equal(got, want) {
		t.Errorf("unitCosts = %v, want %v", got, want)
	}
}

// tblCensusCtx returns a region of the given groups (each with its active
// recipes already set) over the given activities. No activity is active in the
// solver's sense, so the recipe rows of the census are not added.
func tblCensusCtx(groups []groupCtx, acts []viaActivity) *regionCtx {
	cfg := &Config{Roles: make([]Role, len(groups))}
	for i := range groups {
		cfg.Roles[i].ID = "role" + string(rune('a'+i))
		groups[i].role = RoleID(i)
		groups[i].row = &roleRow{}
	}
	return &regionCtx{
		s: &viaSolve{m: &viaModel{c: cfg, year: 365}}, id: "r", acts: acts, groups: groups,
		p: &viaProgramme{base: []int{0}, out: steady.Outcome{Active: make([]bool, len(acts))}},
	}
}

// tblAct is an activity of the given duration and start count.
func tblAct(duration, starts int64) viaActivity {
	return viaActivity{act: steady.Activity{Duration: duration}, count: steady.StartCount{Starts: starts}}
}

// TestCensus_NeedOfAGroup verifies the need of a group: the larger of the labour
// and the largest start capacity among its recipes, whichever comes first or last,
// also for a group of one recipe, and a region with no flow keeps the configured
// households.
//
// Guard: `n == 0 || c > capacity` turned to `n == 1` or `c == capacity` takes the
// wrong capacity; `len(gc.active) > 0` to `> 1` ignores a capacity of one recipe;
// `slices.Max(balanced) == 0` to false, `<`, `== 1`, emptied or `b = h` removed
// lose the households of a region with no flow.
//
// Requirement: FR-VIA-028
func TestCensus_NeedOfAGroup(t *testing.T) {
	t.Parallel()
	one := steady.Rat(1, 1)
	share := func(i int) activeRecipe { return activeRecipe{i: i, share: one} }
	t.Run("largest capacity first or last", func(t *testing.T) {
		t.Parallel()
		// A start every year of a 2-tick recipe (J * d = 2 < 365) needs 365 / 1 = 365
		// households; 5 starts need 365 / 5 = 73. The labour of a group is 2 + 2 = 4.
		acts := []viaActivity{tblAct(2, 1), tblAct(2, 5), tblAct(2, 5), tblAct(2, 1)}
		groups := []groupCtx{
			{g: &Group{Households: 20}, active: []activeRecipe{share(0), share(1)}},
			{g: &Group{Households: 20}, active: []activeRecipe{share(2), share(3)}},
		}
		rc := tblCensusCtx(groups, acts)
		rc.census()
		// Both groups need 365: 40 households split 20 and 20.
		want := []CensusRow{{Region: "r", Role: "rolea", Households: 20, Balanced: 20}, {Region: "r", Role: "roleb", Households: 20, Balanced: 20}}
		if !reflect.DeepEqual(rc.s.rep.Census, want) {
			t.Errorf("Census = %+v, want %+v", rc.s.rep.Census, want)
		}
	})
	t.Run("one recipe", func(t *testing.T) {
		t.Parallel()
		// The first group's one recipe needs 365 by its start capacity (labour 2); the
		// second's 365-tick recipe needs 365 by labour and has no capacity row.
		acts := []viaActivity{tblAct(2, 1), tblAct(365, 1)}
		groups := []groupCtx{
			{g: &Group{Households: 20}, active: []activeRecipe{share(0)}},
			{g: &Group{Households: 20}, active: []activeRecipe{share(1)}},
		}
		rc := tblCensusCtx(groups, acts)
		rc.census()
		want := []CensusRow{{Region: "r", Role: "rolea", Households: 20, Balanced: 20}, {Region: "r", Role: "roleb", Households: 20, Balanced: 20}}
		if !reflect.DeepEqual(rc.s.rep.Census, want) {
			t.Errorf("Census = %+v, want %+v", rc.s.rep.Census, want)
		}
	})
	t.Run("no flow keeps the households", func(t *testing.T) {
		t.Parallel()
		groups := []groupCtx{{g: &Group{Households: 7}, short: true}, {g: &Group{Households: 3}}}
		rc := tblCensusCtx(groups, nil)
		rc.census()
		want := []CensusRow{{Region: "r", Role: "rolea", Households: 7, Balanced: 7, Short: true}, {Region: "r", Role: "roleb", Households: 3, Balanced: 3}}
		if !reflect.DeepEqual(rc.s.rep.Census, want) {
			t.Errorf("Census = %+v, want %+v", rc.s.rep.Census, want)
		}
	})
	t.Run("a balanced household count of 1", func(t *testing.T) {
		t.Parallel()
		// One configured household goes to the group that has flow, whose own count is 0.
		acts := []viaActivity{tblAct(1, 365)}
		groups := []groupCtx{{g: &Group{Households: 0}, active: []activeRecipe{share(0)}}, {g: &Group{Households: 1}}}
		rc := tblCensusCtx(groups, acts)
		rc.census()
		want := []CensusRow{{Region: "r", Role: "rolea", Households: 0, Balanced: 1, Imbalance: -1}, {Region: "r", Role: "roleb", Households: 1, Balanced: 0, Imbalance: 1}}
		if !reflect.DeepEqual(rc.s.rep.Census, want) {
			t.Errorf("Census = %+v, want %+v", rc.s.rep.Census, want)
		}
	})
}

// TestRecipeCensus verifies the recipe rows: an inactive recipe is skipped, a
// recipe on one role's list has no row (and does not stop the scan), one on none
// is unlisted and one on two lists is ambiguous, each with the rounded-up need.
//
// Guard: `!rc.activeAt(i)` to false or emptied lists a stopped recipe;
// `continue` to `break` stops at a recipe with one role; `roles := 0` or
// `roles++` changed miscounts the lists.
//
// Requirement: FR-VIA-028
func TestRecipeCensus(t *testing.T) {
	t.Parallel()
	cfg := &Config{Recipes: []Recipe{{ID: "r0"}, {ID: "r1"}, {ID: "r2"}, {ID: "r3"}}}
	acts := make([]viaActivity, 4)
	for i := range acts {
		acts[i] = viaActivity{recipe: RecipeID(i), count: steady.StartCount{Starts: 365}, act: steady.Activity{Duration: 2}}
	}
	acts[2].act.Duration = 3
	rc := &regionCtx{
		s: &viaSolve{m: &viaModel{c: cfg, year: 365}}, id: "r", acts: acts, listing: []int64{0, 10, 0, 30},
		groups: []groupCtx{{row: &roleRow{recipes: []RecipeID{1, 3}}}, {row: &roleRow{recipes: []RecipeID{3}}}},
		p: &viaProgramme{base: []int{0}, out: steady.Outcome{
			Active: []bool{false, true, true, true},
			Flow:   []steady.Frac{steady.Rat(1, 1), steady.Rat(1, 1), steady.Rat(1, 1), steady.Rat(1, 2)},
		}},
	}
	rc.recipeCensus()
	// r2 on no list: needs 3 * 1 = 3, no households. r3 on two lists: needs 2 / 2 =
	// 1 of the 30 households that list it. r0 is stopped; r1 is on one list.
	want := []CensusRow{
		{Region: "r", Recipe: "r2", Households: 0, Balanced: 3, Imbalance: -3, Unlisted: true},
		{Region: "r", Recipe: "r3", Households: 30, Balanced: 1, Imbalance: 29, Ambiguous: true},
	}
	if !reflect.DeepEqual(rc.s.rep.Census, want) {
		t.Errorf("recipe Census = %+v, want %+v", rc.s.rep.Census, want)
	}
}

// TestWageTargets verifies the roles whose seed wage is below the target: below
// is listed, equal is not, a wage of 0 lists none and a wage of 1 is compared.
//
// Guard: `rc.wage == 0` to false, `<`, emptied or `== 1` lists a region with no
// wage or skips a wage of 1; `rc.wage < target` to `<=` lists an equal wage.
//
// Requirement: FR-WEL-031, FR-WEL-038
func TestWageTargets(t *testing.T) {
	t.Parallel()
	// 10 000 * 100 * 365 / (10 000 * 250) = 146.
	target, ok := welfare.Wage(10_000, 100, 365, 250)
	if !ok || target != 146 {
		t.Fatalf("welfare.Wage = %d, %v, want 146, true", target, ok)
	}
	tests := []struct {
		wage int64
		want []RoleWage
	}{
		{145, []RoleWage{{"r", "w", 145, 146}}},
		{146, nil},
		{147, nil},
		{0, nil},
		{1, []RoleWage{{"r", "w", 1, 146}}},
	}
	for _, tt := range tests {
		cfg := &Config{
			Roles:   []Role{{ID: "w", TargetRatio: 10_000, ExpectedWorkTicks: 250}},
			Regions: []Region{{ID: "r", ReferenceBasket: 100}},
		}
		rc := &regionCtx{s: &viaSolve{m: &viaModel{c: cfg, year: 365}}, id: "r", wage: tt.wage, groups: []groupCtx{{role: 0}}}
		rc.wageTargets()
		if !reflect.DeepEqual(rc.s.rep.WageBelowTarget, tt.want) {
			t.Errorf("wageTargets(wage %d) = %v, want %v", tt.wage, rc.s.rep.WageBelowTarget, tt.want)
		}
	}
}

// tblTwoRegionChain returns two regions that make grain at different rates: the
// north farmer 4 an execution (and a weak recipe of 1 that is never active), the
// south farmer 2. Alone the north has a surplus factor of 4 and the south of 2;
// pooled, 3 (60 made against 20 eaten).
func tblTwoRegionChain() Config {
	c := viaToy()
	south := DefaultRegion()
	south.ID, south.ReferenceBasket = "south", 100
	c.Regions = append(c.Regions, south)
	c.Recipes = append(c.Recipes,
		viaRecipe("farm2", nil, []Output{viaOut("grain", 2)}, nil),
		viaRecipe("weak", nil, []Output{viaOut("grain", 1)}, nil),
	)
	c.Roles[0].Recipes = []string{"farm", "weak"}
	// Each recipe is sited: the farm and the weak recipe in the north, farm2 in the south.
	for i := range c.Recipes {
		c.Recipes[i].Sited = true
	}
	c.Yields = []YieldProfile{
		{Region: "north", Recipe: "farm", HasAll: true, All: 10_000},
		{Region: "north", Recipe: "weak", HasAll: true, All: 10_000},
		{Region: "south", Recipe: "farm2", HasAll: true, All: 10_000},
	}
	slow := DefaultRole()
	slow.ID, slow.Recipes = "slowfarmer", []string{"farm2"}
	c.Roles = append(c.Roles, slow)
	c.Groups = append(c.Groups, Group{Region: "south", Role: "slowfarmer", Households: 10})
	return c
}

// TestSurplusesAndAdvantage verifies the surplus of each region in autarky and
// pooled, and the comparative advantage: without free trade there is none and
// each region has only its own factor; with it every region has both, and only
// the recipes with a flow are listed, each in the regions it runs in.
//
// Guard: `len(p.regions) > 1 || freeTrade && p == progs[0]` turned to `&&`, or
// its inner `&&` to `||`, moves a programme to the wrong side; `!freeTrade ||
// len(progs) == 0` to false or `&&` computes an advantage without free trade;
// `len(regions) > 0` to `true`, `>= 0` or `> 1` lists a recipe that is stopped or
// drops one that runs in one region; `a.recipe == recipe` to any other compare
// lists the regions of the wrong recipes.
//
// Requirement: FR-VIA-043, FR-VIA-044, FR-VIA-045
func TestSurplusesAndAdvantage(t *testing.T) {
	t.Parallel()
	t.Run("alone", func(t *testing.T) {
		t.Parallel()
		st := solveToy(t, tblTwoRegionChain())
		want := []RegionSurplus{{Region: "north", AutarkyBP: 40_000}, {Region: "south", AutarkyBP: 20_000}}
		if !reflect.DeepEqual(st.Report.RegionSurplusBP, want) || !reflect.DeepEqual(st.Report.Advantage, Advantage{}) {
			t.Errorf("alone = %+v, advantage %+v, want %+v and none", st.Report.RegionSurplusBP, st.Report.Advantage, want)
		}
	})
	t.Run("pooled", func(t *testing.T) {
		t.Parallel()
		st := solveToy(t, tblTwoRegionChain(), WithFreeTrade())
		want := []RegionSurplus{
			{Region: "north", AutarkyBP: 40_000, PooledBP: 30_000, HasPooled: true},
			{Region: "south", AutarkyBP: 20_000, PooledBP: 30_000, HasPooled: true},
		}
		if !reflect.DeepEqual(st.Report.RegionSurplusBP, want) {
			t.Errorf("pooled RegionSurplusBP = %+v, want %+v", st.Report.RegionSurplusBP, want)
		}
		// farm runs in the north, farm2 in the south; weak never runs; one basket good
		// makes no pair.
		wantAdv := Advantage{Active: []ActiveIn{{Recipe: "farm", Regions: []string{"north"}}, {Recipe: "farm2", Regions: []string{"south"}}}}
		if !reflect.DeepEqual(st.Report.Advantage, wantAdv) {
			t.Errorf("pooled Advantage = %+v, want %+v", st.Report.Advantage, wantAdv)
		}
	})
	t.Run("one region pooled", func(t *testing.T) {
		t.Parallel()
		st := solveToy(t, viaToy(), WithFreeTrade())
		want := []RegionSurplus{{Region: "north", AutarkyBP: 40_000, PooledBP: 40_000, HasPooled: true}}
		if !reflect.DeepEqual(st.Report.RegionSurplusBP, want) {
			t.Errorf("one-region pooled RegionSurplusBP = %+v, want %+v", st.Report.RegionSurplusBP, want)
		}
		wantAdv := Advantage{Active: []ActiveIn{{Recipe: "farm", Regions: []string{"north"}}}}
		if !reflect.DeepEqual(st.Report.Advantage, wantAdv) {
			t.Errorf("one-region pooled Advantage = %+v, want %+v", st.Report.Advantage, wantAdv)
		}
	})
}

// tblPriced returns an outcome that prices the commodities at the given values.
func tblPriced(prices ...steady.Frac) steady.Outcome { return steady.Outcome{Price: prices} }

// tblPairSolve returns a solve over commodities a, b and c with the given
// pooled and autarky prices.
func tblPairSolve(pooled steady.Outcome, autarky ...steady.Outcome) (*viaSolve, *viaProgramme) {
	cfg := &Config{Commodities: []Commodity{{ID: "a"}, {ID: "b"}, {ID: "c"}}}
	s := &viaSolve{m: &viaModel{c: cfg, withHouseholds: []RegionID{0}, upkeep: [][]int64{{1, 1, 1}}}}
	for _, o := range autarky {
		s.aux = append(s.aux, &viaProgramme{out: o})
	}
	p := &viaProgramme{pr: pooled}
	s.progs = []*viaProgramme{p}
	return s, p
}

// TestRatioOf verifies the price ratio of two goods: both must be priced.
//
// Guard: `Sign() == 0 ||` turned to false, `&&` or `<` divides by or returns the
// ratio of an unpriced good.
//
// Requirement: FR-VIA-045
func TestRatioOf(t *testing.T) {
	t.Parallel()
	o := tblPriced(steady.Rat(0, 1), steady.Rat(2, 1), steady.Rat(3, 1))
	tests := []struct {
		a, b int
		want steady.Frac
		ok   bool
	}{
		{1, 2, steady.Rat(2, 3), true},
		{0, 1, steady.Frac{}, false}, // a is free
		{1, 0, steady.Frac{}, false}, // b is free
		{0, 0, steady.Frac{}, false},
	}
	for _, tt := range tests {
		got, ok := ratioOf(o, tt.a, tt.b)
		if ok != tt.ok || (ok && got.Cmp(tt.want) != 0) {
			t.Errorf("ratioOf(%d, %d) ok = %v, want %v (ratio %v, want %v)", tt.a, tt.b, ok, tt.ok, got, tt.want)
		}
	}
}

// TestAutarkyRange verifies the lowest and highest ratio over the autarky
// programmes that price both goods: one that does not is skipped without ending
// the scan, and a later one extends the range both ways.
//
// Guard: `continue` turned to `break` ends at the unpriced programme; `r > high`
// turned to `==` or `> 1` never raises the top.
//
// Requirement: FR-VIA-045
func TestAutarkyRange(t *testing.T) {
	t.Parallel()
	r := steady.Rat
	s, _ := tblPairSolve(tblPriced(r(1, 1), r(1, 1), r(1, 1)),
		tblPriced(r(0, 1), r(1, 1), r(1, 1)), // a free: skipped, first
		tblPriced(r(2, 1), r(1, 1), r(1, 1)), // 2
		tblPriced(r(6, 1), r(2, 1), r(1, 1)), // 3
		tblPriced(r(1, 1), r(2, 1), r(1, 1)), // 1/2
	)
	low, high, found := s.autarkyRange(0, 1)
	if !found || low.Cmp(r(1, 2)) != 0 || high.Cmp(r(3, 1)) != 0 {
		t.Errorf("autarkyRange = %v, %v, %v, want 1/2, 3, true", low, high, found)
	}
	if _, _, found := s.autarkyRange(0, 0); !found {
		// a/a = 1 for the three programmes that price a.
		t.Errorf("autarkyRange(a, a) found = false, want true")
	}
	empty, _ := tblPairSolve(tblPriced(r(1, 1), r(1, 1), r(1, 1)))
	if _, _, found := empty.autarkyRange(0, 1); found {
		t.Errorf("autarkyRange with no programme found = true, want false")
	}
}

// TestPairRatio verifies the pooled ratio against the autarky range: the three
// values in basis points, rounded down and including 0, whether the pooled one
// is outside the range above or below, and ok false for an unpriced pair or one
// no autarky programme prices, or one value beyond int64.
//
// Guard: `return PairRatio{}, false` to true returns a pair that was not
// compared; `Cmp(high) > 0` to `> 1` misses a ratio above the range; `Floor(0,
// ...)` to `Floor(1, ...)` rejects a ratio under one basis point; `Mul(wholeBP,
// 1)` to `Mul(wholeBP, 2)` doubles a value; `ok1 && ok2 && ok3` to `||` accepts
// a value beyond int64.
//
// Requirement: FR-VIA-045
func TestPairRatio(t *testing.T) {
	t.Parallel()
	r := steady.Rat
	two := func(a, b steady.Frac) steady.Outcome { return tblPriced(a, b, r(1, 1)) }
	huge := r(1<<62, 1)
	tests := []struct {
		name    string
		pooled  steady.Outcome
		autarky []steady.Outcome
		want    PairRatio
		ok      bool
	}{
		// The autarky ratios are 2 and 1/2 (20 000 and 5 000 bp).
		{"above the range", two(r(7, 1), r(2, 1)), []steady.Outcome{two(r(2, 1), r(1, 1)), two(r(1, 1), r(2, 1))},
			PairRatio{A: "a", B: "b", PooledBP: 35_000, LowBP: 5_000, HighBP: 20_000, Outside: true}, true},
		{"below the range", two(r(1, 4), r(1, 1)), []steady.Outcome{two(r(2, 1), r(1, 1)), two(r(1, 1), r(2, 1))},
			PairRatio{A: "a", B: "b", PooledBP: 2_500, LowBP: 5_000, HighBP: 20_000, Outside: true}, true},
		{"inside the range", two(r(1, 1), r(1, 1)), []steady.Outcome{two(r(2, 1), r(1, 1)), two(r(1, 1), r(2, 1))},
			PairRatio{A: "a", B: "b", PooledBP: 10_000, LowBP: 5_000, HighBP: 20_000}, true},
		// Under one basis point (1/20 000 and 1/30 000): rounded down to 0, which is a value.
		{"under one basis point", two(r(1, 20_000), r(1, 1)), []steady.Outcome{two(r(1, 30_000), r(1, 1))},
			PairRatio{A: "a", B: "b", Outside: true}, true},
		{"pooled free", two(r(0, 1), r(1, 1)), []steady.Outcome{two(r(1, 1), r(1, 1))}, PairRatio{}, false},
		{"no autarky programme", two(r(1, 1), r(1, 1)), nil, PairRatio{}, false},
		{"autarky free", two(r(1, 1), r(1, 1)), []steady.Outcome{two(r(0, 1), r(1, 1))}, PairRatio{}, false},
		// Only the pooled ratio beyond int64 / 10 000, then only the autarky one.
		{"pooled beyond int64", two(huge, r(1, 1)), []steady.Outcome{two(r(1, 1), r(1, 1))}, PairRatio{}, false},
		{"autarky beyond int64", two(r(1, 1), r(1, 1)), []steady.Outcome{two(huge, r(1, 1))}, PairRatio{}, false},
		{"autarky top beyond int64", two(r(1, 1), r(1, 1)), []steady.Outcome{two(r(1, 1), r(1, 1)), two(huge, r(1, 1))}, PairRatio{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s, p := tblPairSolve(tt.pooled, tt.autarky...)
			got, ok := s.pairRatio(p, 0, 1)
			if ok != tt.ok || (ok && got != tt.want) {
				t.Errorf("pairRatio(%s) = %+v, %v, want %+v, %v", tt.name, got, ok, tt.want, tt.ok)
			}
		})
	}
}

// TestAdvantage_EachPairOnce verifies that every pair of basket goods is
// compared once, a with b but never a with a, and that nothing is compared
// without free trade.
//
// Guard: `basket[x+1:]` turned to `basket[x*1:]`, `x/1` or `x+0` adds the pair
// of a good with itself; `!freeTrade` turned to false computes pairs without free
// trade.
//
// Requirement: FR-VIA-045
func TestAdvantage_EachPairOnce(t *testing.T) {
	t.Parallel()
	r := steady.Rat
	pooled := tblPriced(r(1, 1), r(2, 1), r(4, 1))
	auto := tblPriced(r(1, 1), r(1, 1), r(1, 1))
	s, _ := tblPairSolve(pooled, auto)
	s.o.freeTrade = true
	s.advantage()
	// Pooled ratios a/b 1/2, a/c 1/4, b/c 1/2 against the autarky 1 each.
	want := []PairRatio{
		{A: "a", B: "b", PooledBP: 5_000, LowBP: 10_000, HighBP: 10_000, Outside: true},
		{A: "a", B: "c", PooledBP: 2_500, LowBP: 10_000, HighBP: 10_000, Outside: true},
		{A: "b", B: "c", PooledBP: 5_000, LowBP: 10_000, HighBP: 10_000, Outside: true},
	}
	if !reflect.DeepEqual(s.rep.Advantage.Ratios, want) {
		t.Errorf("Ratios = %+v, want %+v", s.rep.Advantage.Ratios, want)
	}
	off, _ := tblPairSolve(pooled, auto)
	off.advantage()
	if !reflect.DeepEqual(off.rep.Advantage, Advantage{}) {
		t.Errorf("Advantage without free trade = %+v, want none", off.rep.Advantage)
	}
}
