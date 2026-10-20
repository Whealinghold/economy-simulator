package economysim

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/steady"
)

// Equivalent mutants of viability.go that no test can kill (gomu, one-line why each):
//   - seasonRuns `len(out) > 0 -> !=` (36) and runsFor (182): a length is never
//     negative.
//   - addGroups households `> 0 -> !=` (112): Validate keeps Households positive.
//   - allOpen `<= 0 -> ==` (217), `len(runs) > 0 -> !=` (221), virtualStartOffsets
//     `f > 0 -> !=` (235), noVirtualStart `<= 0 -> ==` (252): a yield factor is
//     never negative, and a length is never negative.
//   - noVirtualStart `i == 0 -> <=` (255): i is never negative.
//   - boundedOption inner `n < lo -> <=` (297): the branch is reached only past a
//     bound, so n == lo never gets there.
//   - SolveSteadyState `s.build()` error branch (602, 603): every overflow build
//     checks is held off by the bounds Validate has already passed.
//   - sets `len(g) == 0` variants (621) and `continue -> break` (622): a group of
//     regions is never empty, as withHouseholds lists only regions with households,
//     and noProgramme/unprogrammed of an empty group do nothing.
//   - unprogrammed `upkeep > 0` variants (639): programmeTable lists the same
//     unpriced goods again and normalise drops the repeats.
//   - autarkies `continue -> break` (674): a pooled set whose every region is
//     alone without a programme but whose pool has one is not reachable with a
//     viable configuration that has a source in both.
//   - solveStrict `ctx.Err()` check (759): the simplex loop checks the context
//     before every pivot, so a cancelled solve fails there with the same error.
//   - solve `len(sets) == 0 -> <=` (651): a length is never negative.
//   - programme `len(p.nominalSeed) > 0 -> !=` (723): likewise.
//   - capacityScarce `Sign() != 0 -> >` (732) and `Sign() > 0 -> !=` (736): a labour
//     dual and a rent are never negative.

// viaCtx is a context that reports Canceled from its (limit+1)th Err call on and
// counts the calls: the solver checks Err before each solve and each pivot.
type viaCtx struct {
	context.Context
	limit, calls int
}

func (c *viaCtx) Err() error {
	c.calls++
	if c.calls > c.limit {
		return context.Canceled
	}
	return nil
}

// viaManyRegions is the toy with n farming regions of ten households each.
func viaManyRegions(n int) Config {
	c := viaToy()
	for i := 1; i < n; i++ {
		r := DefaultRegion()
		r.ID, r.ReferenceBasket = string(rune('a'+i)), 100
		c.Regions = append(c.Regions, r)
		c.Groups = append(c.Groups, Group{Region: r.ID, Role: "farmer", Households: 10})
	}
	return c
}

// TestViaModel_RunsForMergesEqualNeighbours verifies that adjacent seasons with
// one factor are one run, and that runs of different factors are not merged.
//
// Arithmetic: the default calendar has a year of 365 ticks in four seasons; a
// recipe with no yield profile has factor 10 000 in all four.
//
// Guard: a merge that never happens, starts a run late or drops the length leaves
// several runs for one factor.
//
// Requirement: FR-VIA-053
func TestViaModel_RunsForMergesEqualNeighbours(t *testing.T) {
	t.Parallel()
	flat := viaToy().canonical()
	mf := newViaModel(&flat)
	if got, want := mf.runsFor(0, 0), []steady.Run{{Length: 365, FactorBP: 10_000}}; !reflect.DeepEqual(got, want) {
		t.Errorf("runsFor(flat) = %+v, want %+v", got, want)
	}
	sp := viaSeasonal(10, viaSpring("", "")).canonical()
	ms := newViaModel(&sp)
	runs := ms.runsFor(0, 0)
	var sum int64
	for i, r := range runs {
		sum += r.Length
		if i > 0 && runs[i-1].FactorBP == r.FactorBP {
			t.Errorf("runsFor(spring) = %+v, want no two neighbours of one factor", runs)
		}
	}
	if sum != 365 || len(runs) < 2 {
		t.Errorf("runsFor(spring) = %+v, want a total of 365 ticks in at least two runs", runs)
	}
}

// TestAllOpen verifies each way a year is or is not open all through.
//
// Guard: a factor of 1 is positive, so <= 1 would close it, and no run is not an
// open year.
//
// Requirement: FR-VIA-053
func TestAllOpen(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		runs []steady.Run
		want bool
	}{
		{"none", nil, false},
		{"one basis point", []steady.Run{{Length: 1, FactorBP: 1}}, true},
		{"a closed run", []steady.Run{{Length: 3, FactorBP: 5}, {Length: 2, FactorBP: 0}}, false},
		{"all open", []steady.Run{{Length: 3, FactorBP: 5}, {Length: 2, FactorBP: 9}}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := allOpen(tt.runs); got != tt.want {
				t.Errorf("allOpen(%s) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

// TestVirtualStartOffsets_OneBasisPointIsOpen verifies that a yield factor of 1
// basis point opens every offset, in the list and in the run-based test.
//
// Arithmetic: a yield of 1 BP on a base factor of 10 000 is a factor of 1.
//
// Guard: the factor is positive at 1, so a bound of > 1 or <= 1 drops every offset.
//
// Requirement: FR-VIA-053
func TestVirtualStartOffsets_OneBasisPointIsOpen(t *testing.T) {
	t.Parallel()
	c := viaSeasonal(3, YieldProfile{HasAll: true, All: 1}).canonical()
	m := newViaModel(&c)
	if got, want := virtualStartOffsets(&c, &m.yt, 0, 0), []int64{0, 1, 2}; !slices.Equal(got, want) {
		t.Errorf("virtualStartOffsets(1 BP) = %v, want %v", got, want)
	}
	if noVirtualStart(m.runsFor(0, 0), m.year, 3) {
		t.Errorf("noVirtualStart(1 BP) = true, want false")
	}
}

// TestNoVirtualStart_Runs verifies the run-based test on runs given directly.
//
// Arithmetic: year 1 and duration 5 give a tail window starting at max(-3, 0) = 0.
//
// Guard: a window start of 1 instead of 0 and a factor bound of 1 instead of 0
// each turn an open run into a closed one.
//
// Requirement: FR-VIA-053
func TestNoVirtualStart_Runs(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		runs           []steady.Run
		year, duration int64
		want           bool
	}{
		{"first run of 1 BP", []steady.Run{{Length: 1, FactorBP: 1}}, 1, 5, false},
		{"later run of 1 BP", []steady.Run{{Length: 4, FactorBP: 0}, {Length: 1, FactorBP: 1}}, 5, 2, false},
		{"window starts at tick 0", []steady.Run{{Length: 0, FactorBP: 0}, {Length: 1, FactorBP: 5}}, 1, 5, false},
		{"all closed", []steady.Run{{Length: 4, FactorBP: 0}}, 4, 2, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := noVirtualStart(tt.runs, tt.year, tt.duration); got != tt.want {
				t.Errorf("noVirtualStart(%s) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

// TestSolveOptions_OutOfBounds verifies the value and the bound each option
// reports: the lower bound below the range, the upper one above it, and that the
// bounds themselves are accepted.
//
// Guard: a swapped or missing bound reports the wrong one, and a strict
// comparison refuses the end of the range.
//
// Requirement: FR-VIA-017
func TestSolveOptions_OutOfBounds(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		opt          SolveOption
		value, bound string
	}{
		{"cover below", WithStockCoverTicks(-1), "-1", "0"},
		{"cover above", WithStockCoverTicks(maxStockCoverTicks + 1), i64(maxStockCoverTicks + 1), i64(maxStockCoverTicks)},
		{"buffer below", WithMoneyBufferTicks(-5), "-5", "0"},
		{"warn above", WithResidualWarnBP(maxResidualWarnBP + 1), i64(maxResidualWarnBP + 1), i64(maxResidualWarnBP)},
		{"pivots below", WithMaxPivots(0), "0", "1"},
		{"pivots above", WithMaxPivots(MaxViabilityPivots + 1), i64(MaxViabilityPivots + 1), i64(MaxViabilityPivots)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := newSolveOptions([]SolveOption{tt.opt})
			var ve *ValidationError
			if !errors.As(err, &ve) || !errors.Is(err, ErrOutOfBounds) || ve.Value != tt.value || ve.Bound != tt.bound {
				t.Errorf("newSolveOptions(%s) = %v, want ErrOutOfBounds with value %s and bound %s", tt.name, err, tt.value, tt.bound)
			}
		})
	}
	edges := []SolveOption{
		WithStockCoverTicks(0), WithStockCoverTicks(maxStockCoverTicks), WithMoneyBufferTicks(maxMoneyBufferTicks),
		WithResidualWarnBP(maxResidualWarnBP), WithMaxPivots(1), WithMaxPivots(MaxViabilityPivots),
	}
	for i, opt := range edges {
		if _, err := newSolveOptions([]SolveOption{opt}); err != nil {
			t.Errorf("newSolveOptions(edge %d) = %v, want nil: the bounds are inclusive", i, err)
		}
	}
}

// TestMapSolveErr_CertificateIsTheSentinel verifies that a failed certificate is
// ErrSolveInconsistent itself, with no wrapping text, where any other solver
// error is wrapped with its text.
//
// Guard: with the certificate case gone the error falls to the wrapping default
// and still matches errors.Is.
//
// Requirement: FR-VIA-020
func TestMapSolveErr_CertificateIsTheSentinel(t *testing.T) {
	t.Parallel()
	if got := mapSolveErr(steady.ErrInconsistent); got != ErrSolveInconsistent {
		t.Errorf("mapSolveErr(ErrInconsistent) = %v, want the sentinel itself", got)
	}
	if got := mapSolveErr(steady.ErrBadProgramme); got == ErrSolveInconsistent || !errors.Is(got, ErrSolveInconsistent) {
		t.Errorf("mapSolveErr(ErrBadProgramme) = %v, want it wrapped", got)
	}
}

// TestNominalPhi verifies the work share and each way it is not available.
//
// Arithmetic: expected 100 + 200 = 300 over 365 * (10 + 20) = 10 950 household-ticks.
//
// Guard: each of the three reasons (a region's sums, the expected sum, the
// household sum) and the product must give not-ok by itself.
//
// Requirement: FR-VIA-022
func TestNominalPhi(t *testing.T) {
	t.Parallel()
	const maxI = int64(1<<63 - 1)
	tests := []struct {
		name       string
		expected   []int64
		households []int64
		sumsOK     []bool
		year       int64
		want       steady.Ratio
		ok         bool
	}{
		{"fits", []int64{100, 200}, []int64{10, 20}, []bool{true, true}, 365, steady.Ratio{Num: 300, Den: 10_950}, true},
		{"sums of a region", []int64{100, 200}, []int64{10, 20}, []bool{true, false}, 365, steady.Ratio{}, false},
		{"expected overflows", []int64{maxI, 1}, []int64{10, 20}, []bool{true, true}, 365, steady.Ratio{}, false},
		{"households overflow", []int64{1, 1}, []int64{maxI, 1}, []bool{true, true}, 365, steady.Ratio{}, false},
		{"product overflows", []int64{1, 1}, []int64{maxI / 2, 1}, []bool{true, true}, 365, steady.Ratio{Num: 2}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := &viaModel{year: tt.year, expected: tt.expected, households: tt.households, sumsOK: tt.sumsOK}
			got, ok := m.nominalPhi([]RegionID{0, 1})
			if ok != tt.ok || (ok && got != tt.want) {
				t.Errorf("nominalPhi(%s) = %+v, %v, want %+v, %v", tt.name, got, ok, tt.want, tt.ok)
			}
		})
	}
}

// TestViaSolve_ProgrammeWithoutNominal verifies that a programme whose nominal
// sums do not fit is the full-labour one alone: no nominal solve, no nominal
// seed, and a solver error is still returned.
//
// Guard: with the early return gone the nominal solve runs on a zero ratio, and
// with the error test gone a failed solve returns a nil programme.
//
// Requirement: FR-VIA-022
func TestViaSolve_ProgrammeWithoutNominal(t *testing.T) {
	t.Parallel()
	c := viaToy().canonical()
	m := newViaModel(&c)
	m.sumsOK[0] = false
	s := &viaSolve{m: m, o: solveOptions{maxPivots: MaxViabilityPivots}}
	p, err := s.programme(context.Background(), m.withHouseholds)
	if err != nil || p == nil || p.nominalBP != 0 || len(p.nominalSeed) != 0 || p.out.SurplusBP != 40_000 {
		t.Errorf("programme(no nominal) = %+v, %v, want the full programme with no nominal", p, err)
	}
	p, err = s.programme(&viaCtx{Context: context.Background()}, m.withHouseholds)
	if !errors.Is(err, context.Canceled) || p != nil {
		t.Errorf("programme(cancelled, no nominal) = %+v, %v, want a nil programme and Canceled", p, err)
	}
}

// TestViaSolve_FullBases verifies the first column of each region's activities
// in a programme of three regions.
//
// Arithmetic: each region has one activity, so the bases are 0, 1 and 2.
//
// Guard: a base that is the last count and not the running total gives the third
// region the second's columns.
//
// Requirement: FR-VIA-012
func TestViaSolve_FullBases(t *testing.T) {
	t.Parallel()
	c := viaManyRegions(3).canonical()
	m := newViaModel(&c)
	s := &viaSolve{m: m, o: solveOptions{maxPivots: MaxViabilityPivots}}
	p, err := s.full(context.Background(), m.withHouseholds)
	if err != nil || !slices.Equal(p.base, []int{0, 1, 2}) {
		t.Errorf("full(three regions) base = %v, %v, want [0 1 2]", p.base, err)
	}
}

// TestSolveSteadyState_CancelledAnywhere verifies that a context cancelled at any
// check of a solve, in the full solve, the nominal one or a region's autarky
// solve, is the error of the call and never a partial table.
//
// Guard: each of the error returns of solve, autarkies, full, programme and the
// strict solve; with one dropped, the cancelled solve goes on to a nil
// programme or returns a table.
//
// Requirement: FR-VIA-018
func TestSolveSteadyState_CancelledAnywhere(t *testing.T) {
	t.Parallel()
	c := viaManyRegions(2)
	for name, opts := range map[string][]SolveOption{"autarky": nil, "pooled": {WithFreeTrade()}} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			base := &viaCtx{Context: context.Background(), limit: 1 << 30}
			if _, err := SolveSteadyState(base, c, opts...); err != nil {
				t.Fatalf("SolveSteadyState(%s) = %v, want nil", name, err)
			}
			for limit := range base.calls {
				st, err := SolveSteadyState(&viaCtx{Context: context.Background(), limit: limit}, c, opts...)
				if !errors.Is(err, context.Canceled) || !reflect.DeepEqual(st, SteadyState{}) {
					t.Fatalf("SolveSteadyState(%s, cancelled at check %d of %d) = %v, want Canceled and no table", name, limit+1, base.calls, err)
				}
			}
		})
	}
}

// TestSolveSteadyState_AutarkiesOnlyWithFreeTrade verifies that the solves of
// the regions alone run beside a pooled programme and not otherwise: the pooled
// call of one region makes one more full solve.
//
// Guard: with the autarky solves always on, a call with no free trade does the
// same work as the pooled one.
//
// Requirement: FR-VIA-045
func TestSolveSteadyState_AutarkiesOnlyWithFreeTrade(t *testing.T) {
	t.Parallel()
	count := func(opts ...SolveOption) int {
		ctx := &viaCtx{Context: context.Background(), limit: 1 << 30}
		if _, err := SolveSteadyState(ctx, viaToy(), opts...); err != nil {
			t.Fatalf("SolveSteadyState = %v, want nil", err)
		}
		return ctx.calls
	}
	if alone, pooled := count(), count(WithFreeTrade()); alone >= pooled {
		t.Errorf("context checks without and with free trade = %d, %d, want fewer without", alone, pooled)
	}
}

// TestSolveSteadyState_PoolCapFallsToTheRegions verifies that a pivot cap the
// pooled programme exceeds and one region alone does not is the pooled error.
//
// Guard: with the pooled error dropped the call goes on to the autarky solves,
// which fit, and returns a table or fails elsewhere.
//
// Requirement: FR-VIA-010
func TestSolveSteadyState_PoolCapFallsToTheRegions(t *testing.T) {
	t.Parallel()
	c := viaManyRegions(3)
	alone := solveToy(t, c)
	pooled := solveToy(t, c, WithFreeTrade())
	most := alone.Programmes[0].Pivots
	if pooled.Programmes[0].Pivots <= most {
		t.Skipf("the pooled programme (%d pivots) is no larger than a region's (%d)", pooled.Programmes[0].Pivots, most)
	}
	_, err := SolveSteadyState(context.Background(), c, WithFreeTrade(), WithMaxPivots(most))
	if !errors.Is(err, ErrSolveLimit) {
		t.Errorf("SolveSteadyState(pooled, cap %d) = %v, want ErrSolveLimit", most, err)
	}
}

// viaUnprogrammed is a configuration whose first region, "alpha", has a basket of
// one Raw good that only the farmers of "north" make, and so no programme alone, beside a farming "north".
func viaUnprogrammed() Config {
	c := viaToy()
	c.Commodities = append(c.Commodities, viaCommodity("seed", 50, func(m *Commodity) { m.Raw = true }))
	alpha := DefaultRegion()
	alpha.ID, alpha.ReferenceBasket = "alpha", 100
	c.Regions = append(c.Regions, alpha)
	trader := DefaultRole()
	trader.ID, trader.Upkeep = "trader", []Item{{Commodity: "seed", Quantity: 1}}
	c.Roles = append(c.Roles, trader)
	c.Groups = append(c.Groups, Group{Region: "alpha", Role: "trader", Households: 5})
	// Farmers dig the seed, so the pooled realm has a source; alpha alone has none.
	c.Recipes = append(c.Recipes, viaRecipe("dig", nil, []Output{viaOut("seed", 4)}, func(r *Recipe) { r.Sited = true }))
	c.Yields = []YieldProfile{{Region: "north", Recipe: "dig", HasAll: true, All: 10_000}}
	c.Roles[0].Recipes = []string{"farm", "dig"}
	return c
}

// TestSolveSteadyState_UnprogrammedRegionIsListedAndTheRestSolved verifies that a
// region with no programme is reported unpriced, for its own basket goods only,
// and that the regions after it are still solved, alone or pooled.
//
// Guard: stopping at the first such region drops the others; listing a commodity
// with no upkeep, or one with an upkeep of 1, gets the list wrong.
//
// Requirement: FR-VIA-050
func TestSolveSteadyState_UnprogrammedRegionIsListedAndTheRestSolved(t *testing.T) {
	t.Parallel()
	st := solveToy(t, viaUnprogrammed())
	if want := []string{"alpha"}; !slices.Equal(st.Report.UnpricedWages, want) {
		t.Errorf("UnpricedWages = %v, want %v", st.Report.UnpricedWages, want)
	}
	var inAlpha []RegionCommodity
	for _, rc := range st.Report.UnpricedCommodities {
		if rc.Region == "alpha" {
			inAlpha = append(inAlpha, rc)
		}
	}
	if want := []RegionCommodity{{Region: "alpha", Commodity: "seed"}}; !slices.Equal(inAlpha, want) {
		t.Errorf("UnpricedCommodities of alpha = %v, want %v", inAlpha, want)
	}
	if len(st.Programmes) != 1 || !slices.Equal(st.Programmes[0].Regions, []string{"north"}) {
		t.Errorf("Programmes = %+v, want north alone", st.Programmes)
	}
}

// TestViaSolve_SetsOfNoHouseholds verifies that a realm with no household has no
// set, pooled or not, and lists nothing as unprogrammed.
//
// Guard: an empty group tested for a programme is a set of no regions.
//
// Requirement: FR-VIA-018
func TestViaSolve_SetsOfNoHouseholds(t *testing.T) {
	t.Parallel()
	c := viaToy()
	c.Groups = nil
	cc := c.canonical()
	m := newViaModel(&cc)
	s := &viaSolve{m: m, o: solveOptions{freeTrade: true}}
	if got := s.sets(); len(got) != 0 || len(s.rep.UnpricedWages) != 0 {
		t.Errorf("sets(no households) = %v, unpriced %v, want none", got, s.rep.UnpricedWages)
	}
}
