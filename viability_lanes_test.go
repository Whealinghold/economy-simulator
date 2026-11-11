package economysim_test

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/big"
	"reflect"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// completeLanes returns the complete set of ordered pairs of distinct regions
// and commodities, with the given labour and loss: the free-trade set when both
// are 0.
func completeLanes(cfg economysim.Config, labour, lossBP int64) []economysim.Lane {
	var out []economysim.Lane
	for _, a := range cfg.Regions {
		for _, b := range cfg.Regions {
			for _, c := range cfg.Commodities {
				if a.ID != b.ID {
					out = append(out, economysim.Lane{From: a.ID, To: b.ID, Commodity: c.ID, Labour: labour, LossBP: lossBP})
				}
			}
		}
	}
	return out
}

// solveLanes solves cfg with the options and fails the test on an error.
func solveLanes(t *testing.T, cfg economysim.Config, opts ...economysim.SolveOption) economysim.SteadyState {
	t.Helper()
	st, err := economysim.SolveSteadyState(context.Background(), cfg, opts...)
	if err != nil {
		t.Fatalf("SolveSteadyState error = %v, want nil", err)
	}
	return st
}

// TestSolve_LanesOracle verifies, on the pack `two`, that the complete set of
// zero-cost lanes in the lane form gives the surplus, the seed prices and the
// seed wages of WithFreeTrade (the collapsed form), that no lane gives the
// autarky result byte for byte, and that a lossy complete set has a surplus
// between the lowest autarky one and the free-trade one. The programme has one
// surplus factor for all its regions, so the autarky solutions, which use no
// lane, are a feasible point at the lowest of their factors: the pooled optimum
// cannot be below it, and a lossy lane cannot beat a lossless one. The census
// rows (stock, jobs) are not compared: where the optimum is not unique the two
// forms may pick different flows.
//
// Requirement: FR-VIA-043, FR-VIA-044
func TestSolve_LanesOracle(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	cfg := loadPack(t, packLayers["two"]...)
	free := solveLanes(t, cfg, economysim.WithFreeTrade())
	lanes := solveLanes(t, cfg, economysim.WithLanes(completeLanes(cfg, 0, 0)))
	if free.Programmes[0].SurplusBP != lanes.Programmes[0].SurplusBP || !reflect.DeepEqual(free.Table.Prices, lanes.Table.Prices) || !reflect.DeepEqual(free.Table.Wages, lanes.Table.Wages) {
		t.Errorf("complete zero-cost lanes: surplus %d, prices %+v, wages %+v, want the surplus %d, prices %+v and wages %+v of WithFreeTrade",
			lanes.Programmes[0].SurplusBP, lanes.Table.Prices, lanes.Table.Wages, free.Programmes[0].SurplusBP, free.Table.Prices, free.Table.Wages)
	}

	autarky := solveLanes(t, cfg)
	if none := solveLanes(t, cfg, economysim.WithLanes(nil)); !reflect.DeepEqual(autarky, none) {
		t.Errorf("WithLanes(nil) = %+v, want the autarky result %+v", none, autarky)
	}

	lossy := solveLanes(t, cfg, economysim.WithLanes(completeLanes(cfg, 0, 5000)))
	lo := slices.MinFunc(autarky.Programmes, func(a, b economysim.ProgrammeResult) int { return int(a.SurplusBP - b.SurplusBP) }).SurplusBP
	if got, hi := lossy.Programmes[0].SurplusBP, lanes.Programmes[0].SurplusBP; len(lossy.Programmes) != 1 || got < lo || got > hi {
		t.Errorf("lossy complete lanes: %+v, want one programme with a surplus between the lowest autarky %d and the free-trade %d", lossy.Programmes, lo, hi)
	}
}

// TestSolve_LaneValidation verifies the refusals of the lane list, each with its
// sentinel: the config-free rules fail in the option, before the configuration is
// read (an empty Config would fail Validate, so reaching the lane error proves
// it came first); the unknown region and commodity fail in SolveSteadyState after
// the configuration is validated; and WithLanes with WithFreeTrade is
// ErrOptionContext in either order.
//
// Requirement: FR-VIA-063
func TestSolve_LaneValidation(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	cfg := loadPack(t, packLayers["two"]...)
	ok := economysim.Lane{From: "north", To: "south", Commodity: "staple", Labour: 100, LossBP: 100}
	pre := []struct {
		name  string
		lanes []economysim.Lane
		want  error
	}{
		{"self lane", []economysim.Lane{{From: "north", To: "north", Commodity: "staple"}}, economysim.ErrOutOfBounds},
		{"negative labour", []economysim.Lane{{From: "north", To: "south", Commodity: "staple", Labour: -1}}, economysim.ErrOutOfBounds},
		{"negative loss", []economysim.Lane{{From: "north", To: "south", Commodity: "staple", LossBP: -1}}, economysim.ErrOutOfBounds},
		{"loss of everything", []economysim.Lane{{From: "north", To: "south", Commodity: "staple", LossBP: 10000}}, economysim.ErrOutOfBounds},
		{"duplicate", []economysim.Lane{ok, ok}, economysim.ErrDuplicateID},
	}
	for _, tt := range pre {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := economysim.SolveSteadyState(context.Background(), economysim.Config{}, economysim.WithLanes(tt.lanes))
			var ve *economysim.ValidationError
			if !errors.Is(err, tt.want) || !errors.As(err, &ve) || ve.Path[0] != "solve_options" {
				t.Errorf("SolveSteadyState(empty config, WithLanes(%s)) error = %v, want %v from solve_options before the configuration is read", tt.name, err, tt.want)
			}
		})
	}
	post := []struct {
		name string
		lane economysim.Lane
		want error
	}{
		{"unknown origin", economysim.Lane{From: "nowhere", To: "south", Commodity: "staple"}, economysim.ErrUnknownRegion},
		{"unknown destination", economysim.Lane{From: "north", To: "nowhere", Commodity: "staple"}, economysim.ErrUnknownRegion},
		{"unknown commodity", economysim.Lane{From: "north", To: "south", Commodity: "nothing"}, economysim.ErrUnknownCommodity},
	}
	for _, tt := range post {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := economysim.SolveSteadyState(context.Background(), cfg, economysim.WithLanes([]economysim.Lane{tt.lane}))
			if !errors.Is(err, tt.want) {
				t.Errorf("SolveSteadyState(two, lane %+v) error = %v, want %v", tt.lane, err, tt.want)
			}
		})
	}
	t.Run("config error first", func(t *testing.T) {
		t.Parallel()
		_, err := economysim.SolveSteadyState(context.Background(), economysim.Config{}, economysim.WithLanes([]economysim.Lane{post[0].lane}))
		if errors.Is(err, economysim.ErrUnknownRegion) || err == nil {
			t.Errorf("SolveSteadyState(empty config, unknown region) error = %v, want the error of Validate", err)
		}
	})
	for _, order := range [][]economysim.SolveOption{
		{economysim.WithLanes([]economysim.Lane{ok}), economysim.WithFreeTrade()},
		{economysim.WithFreeTrade(), economysim.WithLanes([]economysim.Lane{ok})},
	} {
		if _, err := economysim.SolveSteadyState(context.Background(), economysim.Config{}, order...); !errors.Is(err, economysim.ErrOptionContext) {
			t.Errorf("SolveSteadyState(WithLanes with WithFreeTrade) error = %v, want ErrOptionContext", err)
		}
	}
	// Empty lanes with free trade is not a conflict: there is no lane to refuse.
	if _, err := economysim.SolveSteadyState(context.Background(), cfg, economysim.WithLanes(nil), economysim.WithFreeTrade()); err != nil {
		t.Errorf("SolveSteadyState(WithLanes(nil), WithFreeTrade) error = %v, want nil", err)
	}
}

// TestSolve_LaneComponents verifies, on the pack `three` (far_south, north, south
// in the realm's order), that lanes make one programme for each weakly connected
// component: a lane from north to south alone (one way) joins them, far_south
// stays in autarky with the rows of the plain autarky solve, the programmes are
// ordered by first region, and the order of the lanes changes nothing.
//
// Requirement: FR-VIA-061
func TestSolve_LaneComponents(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	cfg := loadPack(t, packLayers["three"]...)
	autarky := solveLanes(t, cfg)
	oneWay := []economysim.Lane{{From: "north", To: "south", Commodity: "staple", Labour: 500}, {From: "north", To: "south", Commodity: "tool_a", Labour: 500}}
	st := solveLanes(t, cfg, economysim.WithLanes(oneWay))
	var groups [][]string
	for _, p := range st.Programmes {
		groups = append(groups, p.Regions)
	}
	if want := [][]string{{"far_south"}, {"north", "south"}}; !reflect.DeepEqual(groups, want) {
		t.Fatalf("programmes of a one-way lane north to south = %v, want %v", groups, want)
	}
	rows := func(s economysim.SteadyState) []economysim.SeedPrice {
		return slices.DeleteFunc(slices.Clone(s.Table.Prices), func(p economysim.SeedPrice) bool { return p.Region != "far_south" })
	}
	if a, b := autarky.Programmes[0], st.Programmes[0]; !reflect.DeepEqual(a, b) || !reflect.DeepEqual(rows(autarky), rows(st)) {
		t.Errorf("far_south with lanes elsewhere = %+v and %v, want the autarky %+v and %v", b, rows(st), a, rows(autarky))
	}
	reversed := solveLanes(t, cfg, economysim.WithLanes(slices.Clone([]economysim.Lane{oneWay[1], oneWay[0]})))
	if !reflect.DeepEqual(st, reversed) {
		t.Errorf("result with the lanes reversed = %+v, want the result %+v", reversed, st)
	}
	all := solveLanes(t, cfg, economysim.WithLanes(completeLanes(cfg, 0, 0)))
	if len(all.Programmes) != 1 || !slices.Equal(all.Programmes[0].Regions, []string{"far_south", "north", "south"}) {
		t.Errorf("programmes of the complete lane set = %+v, want one over the three regions", all.Programmes)
	}
}

// TestSolve_LaneGrouplessRegion verifies that a lane with an end in a region that
// has no group is ignored: the result equals the result without the lane, and the
// region still has no programme and no table row.
//
// Requirement: FR-VIA-061
func TestSolve_LaneGrouplessRegion(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	cfg := loadPack(t, packLayers["two"]...)
	cfg.Groups = slices.DeleteFunc(slices.Clone(cfg.Groups), func(g economysim.Group) bool { return g.Region == "south" })
	cfg.WarmStart = nil
	plain := solveLanes(t, cfg)
	lane := economysim.Lane{From: "north", To: "south", Commodity: "staple", Labour: 100}
	with := solveLanes(t, cfg, economysim.WithLanes([]economysim.Lane{lane}))
	if !reflect.DeepEqual(plain, with) {
		t.Errorf("result with a lane into a region with no group = %+v, want the result without it %+v", with, plain)
	}
	if len(with.Programmes) != 1 || !slices.Equal(with.Programmes[0].Regions, []string{"north"}) {
		t.Errorf("programmes = %+v, want north alone", with.Programmes)
	}
}

// TestSolve_LanePricesWithinLaneCost verifies, on the table of the pack `two` with
// the lanes of labour 2 000 and loss 1 000 in both directions, that for every
// lane the seed prices satisfy  p[To] * (10 000 - loss) <= (p[From] + wage *
// labour / 10 000) * 10 000  within the rounding of the prices and the wage
// (FR-VIA-062, spec 9.6). Each seed is the exact price rounded half to even, so
// it is off by at most 1/2: p[To] by 1/2 * 9 000, p[From] by 1/2 * 10 000 and the
// wage term by 1/2 * 2 000, in all 10 500 in the scale of 10 000.
//
// Requirement: FR-VIA-062
func TestSolve_LanePricesWithinLaneCost(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	cfg := loadPack(t, packLayers["two"]...)
	const labour, loss = 2000, 1000
	lanes := completeLanes(cfg, labour, loss)
	st := solveLanes(t, cfg, economysim.WithLanes(lanes))
	if len(st.Report.Floored)+len(st.Report.OutsideReferenceBand) > 0 {
		t.Skipf("the table lifts %v and clamps %v: the bound holds for the duals before the band", st.Report.Floored, st.Report.OutsideReferenceBand)
	}
	price := map[[2]string]int64{}
	for _, p := range st.Table.Prices {
		price[[2]string{p.Region, p.Commodity}] = int64(p.Price)
	}
	wage := map[string]int64{}
	for _, w := range st.Table.Wages {
		wage[w.Region] = int64(w.Wage)
	}
	for _, ln := range lanes {
		pFrom, okF := price[[2]string{ln.From, ln.Commodity}]
		pTo, okT := price[[2]string{ln.To, ln.Commodity}]
		if !okF || !okT {
			continue // a commodity with no row in a region has no price to compare
		}
		lhs := pTo * (10000 - loss)
		rhs := pFrom*10000 + wage[ln.From]*labour
		if lhs > rhs+10500 {
			t.Errorf("lane %+v: p[To] * (1 - loss) = %d (x10 000), p[From] + wage * labour = %d (x10 000), want at most the latter plus 10 500", ln, lhs, rhs)
		}
	}
}

// FuzzSolveLanes verifies, over random lane lists on the pack `two` (valid and
// not, with known and unknown IDs), that SolveSteadyState never panics, that its
// error is a sentinel of the closed set, that it is never ErrSolveInconsistent,
// that the call is deterministic, and that a table it returns passes Validate.
//
// Requirement: NFR-VIA-004, FR-VIA-043, FR-VIA-061, FR-VIA-063
func FuzzSolveLanes(f *testing.F) {
	cfg := loadValidPack(f, packLayers["two"]...)
	regions := []string{"north", "south", "nowhere"}
	var commodities []string
	for _, c := range cfg.Commodities {
		commodities = append(commodities, c.ID)
	}
	commodities = append(commodities, "nothing")
	for _, s := range [][]byte{nil, {0, 1, 0, 0, 0}, {0, 1, 3, 100, 50, 1, 0, 3, 0, 0}, {0, 2, 0, 0, 0}, {0, 0, 0, 0, 0}, {255, 255, 255, 255, 255, 255}} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		var lanes []economysim.Lane
		for i := 0; i+4 < len(data) && len(lanes) < 12; i += 5 {
			lanes = append(lanes, economysim.Lane{
				From: regions[int(data[i])%len(regions)], To: regions[int(data[i+1])%len(regions)],
				Commodity: commodities[int(data[i+2])%len(commodities)],
				Labour:    int64(data[i+3])*50 - 100, LossBP: int64(data[i+4])*45 - 45,
			})
		}
		st, err := economysim.SolveSteadyState(context.Background(), cfg, economysim.WithLanes(lanes))
		again, err2 := economysim.SolveSteadyState(context.Background(), cfg, economysim.WithLanes(lanes))
		if fmt.Sprint(err) != fmt.Sprint(err2) || !reflect.DeepEqual(st, again) {
			t.Fatalf("SolveSteadyState(%+v) twice = %v, %v and %v, %v", lanes, st, err, again, err2)
		}
		if err != nil {
			for _, ok := range []error{economysim.ErrOutOfBounds, economysim.ErrDuplicateID, economysim.ErrUnknownRegion, economysim.ErrUnknownCommodity, economysim.ErrSolveLimit, economysim.ErrNoUpkeep} {
				if errors.Is(err, ok) {
					return
				}
			}
			t.Fatalf("SolveSteadyState(%+v) error = %v, want a sentinel of the closed set", lanes, err)
		}
		c := cfg
		c.WarmStart = &st.Table
		if verr := c.Validate(); verr != nil {
			t.Fatalf("Validate(table of %+v) = %v, want nil", lanes, verr)
		}
	})
}

// TestLaneWantsTerm verifies, on the all-modules pack `five`, whose demand module
// gives every group wants, that the fixed term of a lane programme puts the wants
// of each region in the rows of that region: the row of commodity c in region k
// is k*commodities+c and holds what the collapsed term of the region alone gives,
// so summing the regions' rows by commodity gives the collapsed term of all of
// them; and that it is nil with the module off.
//
// Requirement: FR-VIA-055, FR-VIA-062
func TestLaneWantsTerm(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	cfg := economysim.Canonical(loadPack(t, allColdLayers()...))
	n, regions := len(cfg.Commodities), []economysim.RegionID{0, 1, 2, 3, 4}
	got, err := economysim.LaneWantsTerm(&cfg, regions)
	if err != nil || len(got) != len(regions)*n {
		t.Fatalf("LaneWantsTerm(five) = %d rows, %v, want %d rows", len(got), err, len(regions)*n)
	}
	sum := make([]*big.Rat, n)
	for c := range sum {
		sum[c] = new(big.Rat)
	}
	nonzero := 0
	for k, r := range regions {
		own, err := economysim.WantsTerm(&cfg, []economysim.RegionID{r})
		if err != nil {
			t.Fatalf("WantsTerm(region %d) = %v", r, err)
		}
		for c := 0; c < n; c++ {
			if got[k*n+c] != own[c] {
				t.Errorf("LaneWantsTerm row of region %d commodity %d = %v, want %v of the region alone", k, c, got[k*n+c], own[c])
			}
			sum[c].Add(sum[c], big.NewRat(own[c].Num, own[c].Den))
			if own[c].Num != 0 {
				nonzero++
			}
		}
	}
	all, _ := economysim.WantsTerm(&cfg, regions)
	for c := 0; c < n; c++ {
		if sum[c].Cmp(big.NewRat(all[c].Num, all[c].Den)) != 0 {
			t.Errorf("commodity %d: wants of the regions add to %v, want %v of the five together", c, sum[c], all[c])
		}
	}
	if nonzero == 0 {
		t.Errorf("no region has a nonzero fixed term on five, want the demand module's wants")
	}
	cfg.Modules.Demand = nil
	if off, err := economysim.LaneWantsTerm(&cfg, regions); off != nil || err != nil {
		t.Errorf("LaneWantsTerm(demand off) = %v, %v, want nil", off, err)
	}
}

// TestPacks_LanesOracleWants verifies the lane form with the wants of the demand
// module on the all-modules pack `five`: the complete zero-cost lane set and
// WithFreeTrade give the same surplus factor, the same surplus with the wants,
// the same seed prices and the same seed wages. The census rows are not compared
// (the optimum is not unique, see TestSolve_LanesOracle).
//
// Requirement: FR-VIA-043, FR-VIA-044, FR-VIA-055
func TestPacks_LanesOracleWants(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	if raceEnabled {
		t.Skip("about 170 s without -race: gate F runs this test, never -race (plan 2-12b Phase 5)")
	}
	cfg := loadPack(t, allColdLayers()...)
	free := solveLanes(t, cfg, economysim.WithFreeTrade())
	lanes := solveLanes(t, cfg, economysim.WithLanes(completeLanes(cfg, 0, 0)))
	if !free.Report.HasSurplusWants {
		t.Fatalf("WithFreeTrade on the all-modules pack: HasSurplusWants = false, want the wants in the programme")
	}
	if free.Programmes[0].SurplusBP != lanes.Programmes[0].SurplusBP || free.Report.SurplusWantsBP != lanes.Report.SurplusWantsBP ||
		!reflect.DeepEqual(free.Table.Prices, lanes.Table.Prices) || !reflect.DeepEqual(free.Table.Wages, lanes.Table.Wages) {
		t.Errorf("complete zero-cost lanes on five with wants: surplus %d, with wants %d; WithFreeTrade: %d and %d (prices equal %v, wages equal %v)",
			lanes.Programmes[0].SurplusBP, lanes.Report.SurplusWantsBP, free.Programmes[0].SurplusBP, free.Report.SurplusWantsBP,
			reflect.DeepEqual(free.Table.Prices, lanes.Table.Prices), reflect.DeepEqual(free.Table.Wages, lanes.Table.Wages))
	}
}

// TestSolve_LaneExtremeValues verifies that lanes at the edge of int64 labour and
// of the loss range solve or fail with a sentinel, never panic or wrap: the labour
// of 2^63-1 household-ticks per 10 000 units and the largest loss, 9 999.
//
// Requirement: NFR-VIA-004, FR-VIA-063
func TestSolve_LaneExtremeValues(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	cfg := loadPack(t, packLayers["two"]...)
	for _, ln := range []economysim.Lane{
		{From: "north", To: "south", Commodity: "staple", Labour: math.MaxInt64},
		{From: "north", To: "south", Commodity: "staple", Labour: math.MaxInt64, LossBP: 9999},
		{From: "south", To: "north", Commodity: "staple", Labour: 1 << 62, LossBP: 9999},
	} {
		if _, err := economysim.SolveSteadyState(context.Background(), cfg, economysim.WithLanes([]economysim.Lane{ln})); err != nil &&
			!errors.Is(err, economysim.ErrOverflow) && !errors.Is(err, economysim.ErrSolveLimit) {
			t.Errorf("SolveSteadyState(two, lane %+v) error = %v, want nil, ErrOverflow or ErrSolveLimit", ln, err)
		}
	}
}
