package economysim

import (
	"go/ast"
	"go/parser"
	"go/token"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/welfare"
)

// welfareConfig is a small canonical configuration for the welfare glue: a
// staple (commodity 0) and two tools (1 and 2) of durability 4, and three roles
// whose recipes last 60, 5 and no ticks. The tests of the packs `one` and
// `second` need config/ and live in welfare_pack_test.go.
func welfareConfig() *Config {
	c := Config{
		World: WorldParams{WelfareWindow: 30, WindowCycles: 2, RestBand: 500, Upkeep: []Item{{Commodity: "food", Quantity: 2}}},
		Commodities: []Commodity{
			{ID: "food", ReferencePrice: 10},
			{ID: "hammer", Durability: 4, ReferencePrice: 10},
			{ID: "saw", Durability: 4, ReferencePrice: 10},
		},
		Recipes: []Recipe{
			{ID: "crop", Duration: 60, Outputs: []Output{{Commodity: "food", Quantity: 1, CostShare: 10000}}},
			{ID: "loaf", Duration: 5, Outputs: []Output{{Commodity: "food", Quantity: 1, CostShare: 10000}}},
		},
		Roles: []Role{
			{ID: "farmer", Recipes: []string{"crop"}, TargetRatio: 15000, MinWorkShare: 4000, DefaultRatio: 10000},
			{ID: "baker", Recipes: []string{"loaf"}, TargetRatio: 15000},
			{ID: "trader", Trades: []string{"food"}, TargetRatio: 15000},
		},
		Regions:   []Region{{ID: "r"}},
		WarmStart: &WarmStart{Wages: []SeedWage{{Region: "r", Wage: 77}}},
	}
	cc := c.canonical()
	return &cc
}

// TestRoleTable_EffectiveWindow verifies the stored window of a role against
// hand-computed values: a 60-tick recipe at WindowCycles 2 gives 2 * 60 = 120,
// a role whose longest recipe is 5 ticks gives max(30, 2 * 5) = 30, and a role
// with no recipe gives the world's 30.
//
// Requirement: FR-WEL-009
func TestRoleTable_EffectiveWindow(t *testing.T) {
	t.Parallel()
	c := welfareConfig()
	tab := newRoleTable(c)
	// Canonical order: baker, farmer, trader.
	for i, want := range []int64{30, 120, 30} {
		if got := tab.row(RoleID(i)).window; got != want {
			t.Errorf("role %d: window = %d, want %d", i, got, want)
		}
	}
	h := newHistory(tab.row(1))
	if h.Window() != 120 || h.Len() != 0 {
		t.Errorf("newHistory(farmer) = window %d, len %d, want 120, 0", h.Window(), h.Len())
	}
}

// TestNewHistory_RefusesBadWindowAsInvariant verifies that a role row whose
// window is outside the welfare bounds, which validation prevents, panics with
// a message that says it is a violated invariant.
//
// Guard: architecture.md 8.1, a programmer error must say so.
func TestNewHistory_RefusesBadWindowAsInvariant(t *testing.T) {
	t.Parallel()
	for _, w := range []int64{0, MaxWelfareWindow + 1} {
		func() {
			defer func() {
				msg, _ := recover().(string)
				if !strings.Contains(msg, "invariant") {
					t.Errorf("newHistory(window %d) panic = %q, want a message containing \"invariant\"", w, msg)
				}
			}()
			newHistory(&roleRow{window: w})
			t.Errorf("newHistory(window %d) returned, want a panic", w)
		}()
	}
}

// TestWorkingTick verifies the cases of 4.3 and the no-recipe clauses of D70.
//
// Requirement: FR-WEL-017, FR-WEL-018
func TestWorkingTick(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                      string
		hasRecipes                bool
		k                         int64
		ownUse, traded, inTransit bool
		weight                    int64
		wantWorking               bool
		wantWorkHH, wantOwnHH     int64
	}{
		{"recipes, role job of 2 executions at weight 3", true, 2, false, false, false, 3, true, 2, 0},
		{"recipes, role job k 3 at weight 4: workHH 3, ownHH 0 (D70)", true, 3, false, false, false, 4, true, 3, 0},
		{"recipes, own-use job k 3 at weight 4: workHH 3, ownHH 3 (D70)", true, 3, true, false, false, 4, true, 3, 3},
		{"recipes, own-use job of 3 at weight 4", true, 3, true, false, false, 4, true, 3, 3},
		// Fix: PR 6 review. A negative k was returned as workHH and ownHH.
		{"recipes, negative own-use executions count as none", true, -1, true, false, false, 3, false, 0, 0},
		{"recipes, negative executions count as none", true, -1, false, false, false, 3, false, 0, 0},
		{"recipes, no job is rest", true, 0, false, false, false, 3, false, 0, 0},
		{"recipes, trade alone is not work", true, 0, false, true, false, 3, false, 0, 0},
		{"no recipes, traded at weight 3", false, 0, false, true, false, 3, true, 3, 0},
		{"no recipes, did not trade", false, 0, false, false, false, 3, false, 0, 0},
		{"no recipes, own-use job of 0 executions is not work", false, 0, true, false, false, 4, false, 0, 0},
		{"no recipes, negative own-use executions is not work", false, -1, true, false, false, 4, false, 0, 0},
		{"no recipes, executions without own-use and no trade is not work", false, 3, false, false, false, 4, false, 0, 0},
		{"no recipes, own-use job k 2 alone: workHH = ownHH, no wage", false, 2, true, false, false, 4, true, 2, 2},
		{"no recipes, traded and own-use k 2 at weight 4: denominator 2", false, 2, true, true, false, 4, true, 4, 2},
		{"no recipes, in transit on a trip", false, 0, false, false, true, 3, true, 3, 0},
		{"recipes, in transit on a trip has no household-ticks rule", true, 0, false, false, true, 3, true, 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w, whh, ohh := workingTick(tt.hasRecipes, tt.k, tt.ownUse, tt.traded, tt.inTransit, tt.weight)
			if w != tt.wantWorking || whh != tt.wantWorkHH || ohh != tt.wantOwnHH {
				t.Errorf("workingTick(%v, %d, %v, %v, %v, %d) = %v, %d, %d, want %v, %d, %d",
					tt.hasRecipes, tt.k, tt.ownUse, tt.traded, tt.inTransit, tt.weight, w, whh, ohh,
					tt.wantWorking, tt.wantWorkHH, tt.wantOwnHH)
			}
			if whh-ohh < 0 {
				t.Errorf("workHH - ownHH = %d, want it never negative", whh-ohh)
			}
		})
	}
}

// TestMarkWorking_SecondCallIsAnInvariantViolation verifies that setting the
// working scratch twice for one slot in a tick panics with the invariant
// message, and that a reset allows the next tick.
//
// Requirement: FR-WEL-017
// Guard: an agent has one job in a tick (FR-PRO-014, D70); a second call would
// be a bug in the caller that a sum or a maximum would hide.
func TestMarkWorking_SecondCallIsAnInvariantViolation(t *testing.T) {
	t.Parallel()
	s := newWelfareScratch(welfareConfig(), 2)
	s.markWorking(1, 3, false)
	if k, own, traded := s.workRead(1); k != 3 || own || traded {
		t.Errorf("workRead = %d, %v, %v, want 3, false, false", k, own, traded)
	}
	func() {
		defer func() {
			msg, _ := recover().(string)
			if !strings.Contains(msg, "invariant") {
				t.Errorf("second markWorking panic = %q, want a message containing \"invariant\"", msg)
			}
		}()
		s.markWorking(1, 1, true)
		t.Errorf("second markWorking returned, want a panic")
	}()
	s.reset(1)
	s.markWorking(1, 1, true)
	s.markTraded(1)
	if k, own, traded := s.workRead(1); k != 1 || !own || !traded {
		t.Errorf("workRead after reset = %d, %v, %v, want 1, true, true", k, own, traded)
	}
}

// TestRecord_WearValuedAtP63 verifies that the wear rows are facts and are
// valued at the prices handed to record: the tool was 100 when the job started
// and is 120 at P6.3, so the record carries 120. With ceil(4 * 120 / 4) = 120
// per execution and 1 execution the income is -120.
//
// Requirement: FR-WEL-003
func TestRecord_WearValuedAtP63(t *testing.T) {
	t.Parallel()
	s := newWelfareScratch(welfareConfig(), 1)
	s.addWear(0, 1, 1, 4) // P2.7: no money is passed
	got := s.record(0, 1, true, 1, 0, []int64{10, 120, 10}, nil)
	if got.Income != -120 {
		t.Errorf("record income at tool price 120 = %d, want -120 = -(ceil(4 * 120 / 4))", got.Income)
	}
	got = s.record(0, 1, true, 1, 0, []int64{10, 10, 10}, nil)
	if got.Income != -10 {
		t.Errorf("record income at the reference price 10 = %d, want -10", got.Income)
	}
}

// TestRecord_WearRoundingAndWeight verifies one ceiling per execution and no
// second weight factor: WearPerUse 3, Durability 4, price 10, 2 executions,
// weight 3 cost 2 * ceil(30 / 4) = 2 * 8 = 16 (not ceil(60 / 4) = 15 and not
// 16 * 3 = 48); a row with another WearPerUse (5: ceil(50 / 4) = 13, 1
// execution) stays a separate row and adds 13; the rows of the same commodity
// and WearPerUse merge.
//
// Requirement: FR-WEL-003
func TestRecord_WearRoundingAndWeight(t *testing.T) {
	t.Parallel()
	s := newWelfareScratch(welfareConfig(), 1)
	s.addWear(0, 1, 1, 3)
	s.addWear(0, 1, 1, 3) // merges: 2 executions of wear 3
	if f := s.at(0); f.nWear != 1 || f.wear[0].executions != 2 {
		t.Fatalf("after two addWear of one row: %d rows, executions %d, want 1 row of 2", f.nWear, f.wear[0].executions)
	}
	prices := []int64{10, 10, 10}
	if got := s.record(0, 3, true, 2, 0, prices, nil).Income; got != -16 {
		t.Errorf("income with wear 2 * ceil(30/4) at weight 3 = %d, want -16", got)
	}
	s.addWear(0, 1, 1, 5) // another WearPerUse: a row of its own
	if f := s.at(0); f.nWear != 2 {
		t.Fatalf("rows = %d, want 2", f.nWear)
	}
	if got := s.record(0, 3, true, 2, 0, prices, nil).Income; got != -29 {
		t.Errorf("income with the second row = %d, want -29 = -(16 + 13)", got)
	}
}

// TestRecord_WearRowsOverCapacityIsAnInvariantViolation verifies that more
// distinct wear rows than a recipe has tools panics with the invariant message.
//
// Guard: an agent starts one job in a tick, so a fifth row cannot happen; the
// scratch is fixed in size so the tick allocates nothing.
func TestRecord_WearRowsOverCapacityIsAnInvariantViolation(t *testing.T) {
	t.Parallel()
	s := newWelfareScratch(welfareConfig(), 1)
	defer func() {
		msg, _ := recover().(string)
		if !strings.Contains(msg, "invariant") {
			t.Errorf("addWear over capacity panic = %q, want a message containing \"invariant\"", msg)
		}
	}()
	for i := range int64(MaxRequires + 1) {
		s.addWear(0, 1, 1, i+1)
	}
	t.Errorf("addWear over capacity returned, want a panic")
}

// TestRecord_FormsIncomeAndCost verifies income_t for a scripted tick: receipts
// 900 and the relief 50 are credits, the input spend 200 and the wear 40 are
// debits, so income = 900 + 50 - 200 - 40 = 710; the wear is
// ceil(4 * 100 / 10) = 40 on a tool of durability 10. The upkeep 2 food at 10 for
// weight 3 costs 2 * 10 * 3 = 60. A one-tick ring reads 710 * 10 000 / 60. An
// Upkeep, Tool or Other spend has no method and so is never netted.
//
// Requirement: FR-WEL-002, FR-WEL-003, FR-WEL-004, FR-SCL-009
func TestRecord_FormsIncomeAndCost(t *testing.T) {
	t.Parallel()
	c := welfareConfig()
	c.Commodities[1].Durability = 10
	s := newWelfareScratch(c, 1)
	s.addReceipt(0, 400)
	s.addReceipt(0, 500)
	s.addInputSpend(0, 200)
	s.at(0).relief = 50
	s.addWear(0, 1, 1, 4)
	upkeep := []itemRow{{commodity: 0, quantity: 2}}
	rec := s.record(0, 3, true, 3, 0, []int64{10, 100, 10}, upkeep)
	want := welfare.Record{Income: 710, Cost: 60, Relief: 50, Working: true, WorkHH: 3}
	if rec != want {
		t.Fatalf("record = %+v, want %+v", rec, want)
	}
	ring, err := welfare.NewRing(1, MaxWelfareWindow)
	if err != nil {
		t.Fatal(err)
	}
	if ring.Push(rec) {
		t.Errorf("Push(%+v) clamped, want not", rec)
	}
	if r, ok := ring.Ratio(60, 10000); !ok || r != 710*10000/60 {
		t.Errorf("ratio = %d, %v, want %d, true", r, ok, 710*10000/60)
	}
}

// TestCostTick verifies the cost of a tick: a two-commodity upkeep (2 at 10 and
// 1 at 50) at weight 3 is (20 + 50) * 3 = 210; an away agent is valued at the
// prices it is given, those of its account's region; an overflow saturates to
// MaxInt64 and has no ok.
//
// Requirement: FR-WEL-004, FR-WEL-005
func TestCostTick(t *testing.T) {
	t.Parallel()
	b := newCostBuf(3)
	upkeep := []itemRow{{commodity: 0, quantity: 2}, {commodity: 2, quantity: 1}}
	if got := b.tick(upkeep, []int64{10, 0, 50}, 3); got != 210 {
		t.Errorf("tick at home prices = %d, want 210", got)
	}
	if got := b.tick(upkeep, []int64{20, 0, 100}, 3); got != 420 {
		t.Errorf("tick at the account's region prices = %d, want 420", got)
	}
	huge := []itemRow{{commodity: 0, quantity: math.MaxInt64}}
	if got := b.tick(huge, []int64{10, 0, 0}, 3); got != math.MaxInt64 {
		t.Errorf("tick of an overflowing upkeep = %d, want MaxInt64", got)
	}
}

// TestRecord_OverflowSaturatesAndPushClamps verifies D63: receipts at MaxInt64
// give an income at the extreme and not a panic or a wrap, Push then stores the
// record clamped and reports it, and a ring holding a field at the clamp is
// counted in ClampedHouseholds.
//
// Requirement: FR-WEL-010, FR-WEL-042
func TestRecord_OverflowSaturatesAndPushClamps(t *testing.T) {
	t.Parallel()
	c := welfareConfig()
	s := newWelfareScratch(c, 1)
	s.addReceipt(0, math.MaxInt64)
	s.addReceipt(0, math.MaxInt64)
	if got := s.at(0).receipts; got != math.MaxInt64 {
		t.Fatalf("receipts after two MaxInt64 = %d, want MaxInt64", got)
	}
	rec := s.record(0, 1, true, 1, 0, []int64{10, 10, 10}, nil)
	if rec.Income != math.MaxInt64 {
		t.Fatalf("income = %d, want MaxInt64", rec.Income)
	}
	ring, err := welfare.NewRing(5, MaxWelfareWindow)
	if err != nil {
		t.Fatal(err)
	}
	if !ring.Push(rec) {
		t.Errorf("Push of an income at MaxInt64 reported no clamp, want one")
	}
	if !ring.AtBound() {
		t.Errorf("AtBound after a clamped push = false, want true")
	}
	b := newReadBuf(3)
	set := roleSet{rings: []*welfare.Ring{&ring}, weights: []int64{4}, targets: []int64{15000}, free: []bool{true}}
	if got := b.clampedHouseholds(set); got != 4 {
		t.Errorf("clampedHouseholds = %d, want 4", got)
	}
	var run int64
	var warns []int64
	for i := range 11 {
		var warn bool
		run, warn = clampRun(run, b.clampedHouseholds(set))
		if warn {
			warns = append(warns, int64(i+1))
		}
	}
	if len(warns) != 2 || warns[0] != 1 || warns[1] != 10 {
		t.Errorf("clamp run warned at ticks %v, want [1 10]", warns)
	}
}

// TestNewTarget verifies the creation of a target: the first word of the
// creation stream is taken whatever the spread, so the next word is the same at
// spread 0 and at spread 3 000, the target at spread 0 is the TargetRatio and
// at spread 3 000 lies in [15 000 * 7 000 / 10 000, 15 000 * 13 000 / 10 000] =
// [10 500, 19 500].
//
// Requirement: FR-WEL-024
func TestNewTarget(t *testing.T) {
	t.Parallel()
	s := newStream(42)
	s.keyed(siteConstruction, 0, 7)
	t0 := newTarget(s, 15000, 0)
	next0 := s.word()
	s.keyed(siteConstruction, 0, 7)
	t3 := newTarget(s, 15000, 3000)
	next3 := s.word()
	if t0 != 15000 {
		t.Errorf("target at spread 0 = %d, want 15000", t0)
	}
	if t3 < 10500 || t3 > 19500 {
		t.Errorf("target at spread 3000 = %d, want within [10500, 19500]", t3)
	}
	if next0 != next3 {
		t.Errorf("residue word differs between spreads: %d vs %d, want the same stream", next0, next3)
	}
}

// ringWith returns a ring of the given window holding the records.
func ringWith(t *testing.T, window int64, recs ...welfare.Record) *welfare.Ring {
	t.Helper()
	r, err := welfare.FromRecords(recs, window, MaxWelfareWindow)
	if err != nil {
		t.Fatalf("FromRecords error = %v, want nil", err)
	}
	return &r
}

// TestRegionWage verifies the wage the cost anchor reads: the pooled wage when
// known and positive, the warm-start wage when the pooled one is not, and
// unknown with neither. The pooled wage of two rings is (900 + 300) / (3 + 1)
// = 300; a ring that worked at a loss gives a pooled wage that is not positive
// and so the seed; a ring that never worked gives the seed; the same answer
// follows a rebuild of the rings from their records (no latch). With a floor of
// 5 000 bp and a seed of 1 000 the floor is 500: a pooled wage of 300 is raised to
// 500, one of 600 is kept, and a floor with no seed changes nothing.
//
// Requirement: FR-WEL-018, FR-WEL-019, FR-WEL-034, FR-PRO-055
func TestRegionWage(t *testing.T) {
	t.Parallel()
	good1 := welfare.Record{Income: 900, Working: true, WorkHH: 3}
	good2 := welfare.Record{Income: 300, Working: true, WorkHH: 1}
	loss := welfare.Record{Income: -50, Working: true, WorkHH: 1}
	idle := welfare.Record{Income: 10}
	tests := []struct {
		name      string
		rings     []*welfare.Ring
		seed      int64
		floorBP   int64
		wantWage  int64
		wantKnown bool
	}{
		{"pooled over two agents", []*welfare.Ring{ringWith(t, 5, good1), ringWith(t, 5, good2)}, 77, 0, 300, true},
		{"pooled below the floor is raised", []*welfare.Ring{ringWith(t, 5, good1), ringWith(t, 5, good2)}, 1_000, 5_000, 500, true},
		{"pooled above the floor is kept", []*welfare.Ring{ringWith(t, 5, good1), ringWith(t, 5, good2)}, 500, 5_000, 300, true},
		{"a floor with no seed changes nothing", []*welfare.Ring{ringWith(t, 5, good1), ringWith(t, 5, good2)}, 0, 5_000, 300, true},
		{"loss falls back to the seed", []*welfare.Ring{ringWith(t, 5, loss)}, 77, 0, 77, true},
		{"nobody worked falls back to the seed", []*welfare.Ring{ringWith(t, 5, idle)}, 77, 0, 77, true},
		{"loss and no seed is unknown", []*welfare.Ring{ringWith(t, 5, loss)}, 0, 0, 0, false},
		{"no agents and no seed is unknown", nil, 0, 0, 0, false},
		{"a negative seed is not a starting wage", []*welfare.Ring{ringWith(t, 5, loss)}, -5, 0, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			b := newReadBuf(3)
			w, known := b.regionWage(tt.rings, tt.seed, tt.floorBP)
			if w != tt.wantWage || known != tt.wantKnown {
				t.Errorf("regionWage = %d, %v, want %d, %v", w, known, tt.wantWage, tt.wantKnown)
			}
			rebuilt := make([]*welfare.Ring, len(tt.rings))
			for i, r := range tt.rings {
				rebuilt[i] = ringWith(t, r.Window(), r.AppendRecords(nil)...)
			}
			if w2, k2 := b.regionWage(rebuilt, tt.seed, tt.floorBP); w2 != w || k2 != known {
				t.Errorf("regionWage after a rebuild = %d, %v, want %d, %v", w2, k2, w, known)
			}
		})
	}
}

// TestSeedWages verifies the starting wage by region and that a configuration
// with no warm start gives none.
//
// Requirement: FR-WEL-019
func TestSeedWages(t *testing.T) {
	t.Parallel()
	c := welfareConfig()
	if got := seedWages(c); len(got) != 1 || got[0] != 77 {
		t.Errorf("seedWages = %v, want [77]", got)
	}
	c.WarmStart = nil
	if got := seedWages(c); len(got) != 1 || got[0] != 0 {
		t.Errorf("seedWages without a warm start = %v, want [0]", got)
	}
}

// TestFloorRun verifies the run of "work floor binding": ten ticks above half
// warn at the tenth and the twentieth and not between; a tick at or below half
// resets; exactly half does not warn (2 * 5 = 10 is not above 10); a role with
// no households has no run.
//
// Requirement: FR-WEL-033
func TestFloorRun(t *testing.T) {
	t.Parallel()
	var run int64
	var warns []int
	for i := 1; i <= 20; i++ {
		var w bool
		run, w = floorRun(run, 6, 10) // 12 > 10
		if w {
			warns = append(warns, i)
		}
	}
	if len(warns) != 2 || warns[0] != 10 || warns[1] != 20 {
		t.Errorf("floorRun warned at ticks %v, want [10 20]", warns)
	}
	for range 9 {
		run, _ = floorRun(run, 6, 10)
	}
	if run, _ = floorRun(run, 5, 10); run != 0 {
		t.Errorf("floorRun at exactly half: run = %d, want 0 (reset)", run)
	}
	if run, w := floorRun(3, 0, 0); run != 0 || w {
		t.Errorf("floorRun with no households = %d, %v, want 0, false", run, w)
	}
	if run, w := floorRun(math.MaxInt64, 6, 10); run != math.MaxInt64 || w {
		t.Errorf("floorRun at the saturated run = %d, %v, want MaxInt64, false (MaxInt64 is not a multiple of 10)", run, w)
	}
}

// floorSet builds a set of agents of the given weights and records: agent i has
// the working ticks given and the same cost, an income that gives a ratio of
// 20 000 (above the band) unless low says otherwise.
func floorSet(t *testing.T, weights []int64, working []int64, income int64) roleSet {
	t.Helper()
	set := roleSet{weights: weights}
	for i, w := range working {
		var recs []welfare.Record
		for j := range 10 {
			recs = append(recs, welfare.Record{Income: income, Cost: 100, Working: int64(j) < w, WorkHH: 1})
		}
		set.rings = append(set.rings, ringWith(t, 10, recs...))
		set.targets = append(set.targets, 15000)
		set.free = append(set.free, true)
		_ = i
	}
	return set
}

// TestHeldByFloor verifies the count in households: window 10, MinWorkShare
// 4 000 so the floor is 4 000 * 10 / 10 000 = 4 working ticks; an agent with 3
// is held, one with 4 is not; the weights 1 and 40 count as households; an
// agent that is busy (not free) is not counted; a ratio below the target
// (share already 10 000) is not held; floor off holds nothing.
//
// Requirement: FR-WEL-020, FR-WEL-027, FR-WEL-033
func TestHeldByFloor(t *testing.T) {
	t.Parallel()
	upkeep := []itemRow{{commodity: 0, quantity: 1}}
	prices := []int64{100, 0, 0}
	p := welfareParams{window: 10, minShareBP: 4000, restBand: 500, defaultBP: 10000}
	// Income 2 000 over cost 100 per tick: ratio 20 000 at weight 1 (cost 100),
	// at or above T_hi = 15 000 * 10 500 / 10 000 = 15 750, so the share is 0.
	set := floorSet(t, []int64{1, 40, 7}, []int64{3, 4, 0}, 200)
	b := newReadBuf(3)
	// Agent costs: weight 1 gives cost 100, but the records hold cost 100 each
	// whatever the weight; the padded upkeep is 0 as the rings are full.
	if got := b.heldByFloor(set, p, upkeep, prices); got != 8 {
		t.Errorf("heldByFloor = %d, want 8 = 1 (3 < 4) + 7 (0 < 4), the weight-40 agent worked 4", got)
	}
	set.free[2] = false
	if got := b.heldByFloor(set, p, upkeep, prices); got != 1 {
		t.Errorf("heldByFloor with the weight-7 agent busy = %d, want 1", got)
	}
	set.free[2] = true
	p0 := p
	p0.minShareBP = 0
	if got := b.heldByFloor(set, p0, upkeep, prices); got != 0 {
		t.Errorf("heldByFloor with the floor off = %d, want 0", got)
	}
	low := floorSet(t, []int64{5}, []int64{0}, 50) // ratio 5 000 < T_lo: share 10 000
	if got := b.heldByFloor(low, p, upkeep, prices); got != 0 {
		t.Errorf("heldByFloor with a ratio below the target = %d, want 0 (the ratio alone already works)", got)
	}
	if got := households(set); got != 48 {
		t.Errorf("households = %d, want 48", got)
	}
}

// TestSummarise verifies a hand-computed summary: two agents of weights 2 and 3
// each with one working tick, one in the role of interest. Agent A: income 600,
// WorkHH 2, so wage 300; agent B: income 150, WorkHH 3, so wage 50; the set
// wage is (600 + 150) / (2 + 3) = 150 (pooled, not the mean of 300 and 50 = 175
// weighted by agent). An unskilled wage of 100 gives a premium of 150 * 10 000 /
// 100 = 15 000; an unskilled wage of 0 gives none. The mean ratio is
// weighted by households.
//
// Requirement: FR-WEL-019, FR-WEL-020, FR-WEL-036, FR-WEL-037
func TestSummarise(t *testing.T) {
	t.Parallel()
	upkeep := []itemRow{{commodity: 0, quantity: 1}}
	prices := []int64{100, 0, 0}
	p := welfareParams{window: 1, minShareBP: 0, restBand: 500, defaultBP: 10000}
	set := roleSet{
		rings: []*welfare.Ring{
			ringWith(t, 1, welfare.Record{Income: 600, Cost: 200, Working: true, WorkHH: 2}),
			ringWith(t, 1, welfare.Record{Income: 150, Cost: 300, Working: true, WorkHH: 3}),
		},
		weights: []int64{2, 3},
		targets: []int64{15000, 15000},
		free:    []bool{true, true},
	}
	b := newReadBuf(3)
	got := b.summarise(set, p, upkeep, prices, 100, true)
	// Ratios: 600 * 10 000 / 200 = 30 000 and 150 * 10 000 / 300 = 5 000; the
	// mean over households is (2 * 30 000 + 3 * 5 000) / 5 = 15 000.
	want := roleSummary{Wage: 150, HasWage: true, Premium: 15000, HasPremium: true, Window: 1,
		MeanRatioBP: 15000, RatioHouseholds: 5, HasMean: true}
	if got != want {
		t.Errorf("summarise = %+v, want %+v", got, want)
	}
	if got := b.summarise(set, p, upkeep, prices, 0, true); got.HasPremium {
		t.Errorf("summarise with an unskilled wage of 0 has a premium %d, want none", got.Premium)
	}
	if got := b.summarise(set, p, upkeep, prices, 0, false); got.HasPremium {
		t.Errorf("summarise with no unskilled agent has a premium %d, want none", got.Premium)
	}
}

// TestSummarise_NegativeObservedWageIsPresent verifies that a role that worked
// at a loss has an observed wage that is present and negative while the pooled
// regional wage over the same ring is unknown (D62, FR-WEL-037).
//
// Requirement: FR-WEL-037
func TestSummarise_NegativeObservedWageIsPresent(t *testing.T) {
	t.Parallel()
	r := ringWith(t, 1, welfare.Record{Income: -90, Cost: 100, Working: true, WorkHH: 3})
	set := roleSet{rings: []*welfare.Ring{r}, weights: []int64{3}, targets: []int64{15000}, free: []bool{true}}
	b := newReadBuf(3)
	sum := b.summarise(set, welfareParams{window: 1, defaultBP: 10000}, nil, []int64{1, 1, 1}, 100, true)
	if !sum.HasWage || sum.Wage != -30 {
		t.Errorf("observed wage = %d, %v, want -30, true", sum.Wage, sum.HasWage)
	}
	if w, known := b.regionWage(set.rings, 0, 0); known {
		t.Errorf("pooled wage = %d, %v, want unknown", w, known)
	}
}

// TestWelfareAllocations verifies that recording a tick, the working tick, the
// scratch methods, the cost of a tick, the runs, the held count and the region
// wage allocate nothing once the buffers have their size (the scratch and the
// rings are built before the measured closures).
//
// Requirement: NFR-WEL-001
func TestWelfareAllocations(t *testing.T) {
	c := welfareConfig()
	s := newWelfareScratch(c, 2)
	b := newReadBuf(3)
	upkeep := []itemRow{{commodity: 0, quantity: 2}}
	prices := []int64{10, 10, 10}
	p := welfareParams{window: 10, minShareBP: 4000, restBand: 500, defaultBP: 10000}
	set := floorSet(t, []int64{1, 40, 7}, []int64{3, 4, 0}, 200)
	st := newStream(1)
	var sink int64
	tests := []struct {
		name string
		f    func()
	}{
		{"scratch methods", func() {
			s.addReceipt(1, 5)
			s.addInputSpend(1, 5)
			s.addWear(1, 1, 1, 3)
			s.markWorking(1, 1, false)
			s.markTraded(1)
			k, _, _ := s.workRead(1)
			sink += k
			s.reset(1)
		}},
		{"record with wear", func() {
			s.addWear(0, 1, 2, 3)
			sink += s.record(0, 3, true, 2, 0, prices, upkeep).Income
			s.reset(0)
		}},
		{"workingTick", func() {
			_, w, o := workingTick(false, 2, true, true, false, 4)
			sink += w + o
		}},
		{"cost of a tick", func() { sink += s.cost.tick(upkeep, prices, 3) }},
		{"floorRun", func() { r, _ := floorRun(9, 6, 10); sink += r }},
		{"clampRun", func() { r, _ := clampRun(9, 1); sink += r }},
		{"heldByFloor", func() { sink += b.heldByFloor(set, p, upkeep, prices) }},
		{"regionWage", func() { w, _ := b.regionWage(set.rings, 77, 0); sink += w }},
		{"summarise", func() { sink += b.summarise(set, p, upkeep, prices, 100, true).Wage }},
		{"newTarget", func() {
			st.keyed(siteConstruction, 0, 1)
			sink += newTarget(st, 15000, 3000)
		}},
	}
	for _, tt := range tests {
		tt.f() // grow the buffers once
		if got := testing.AllocsPerRun(100, tt.f); got != 0 {
			t.Errorf("AllocsPerRun(%s) = %v, want 0", tt.name, got)
		}
	}
	_ = sink
}

// TestWelfareFiles_Guards verifies, over the syntax tree of welfare.go and of
// the files of internal/welfare, that no .ID selector is read and no string
// literal is compared or named in a case clause, and that welfare.go declares
// no package variable and no function or type named for converting.
//
// Requirement: FR-WEL-013
// Guard: architecture.md 9.1 and 2.9, a branch on a commodity, recipe, role or
// region identifier is a defect, and the welfare arithmetic sees only numbers.
func TestWelfareFiles_Guards(t *testing.T) {
	t.Parallel()
	files := []string{"welfare.go"}
	entries, err := os.ReadDir("internal/welfare")
	if err != nil {
		t.Fatalf("ReadDir error = %v", err)
	}
	for _, e := range entries {
		if n := e.Name(); strings.HasSuffix(n, ".go") && !strings.HasSuffix(n, "_test.go") {
			files = append(files, filepath.Join("internal/welfare", n))
		}
	}
	if len(files) < 8 {
		t.Fatalf("checking %d files, want at least 8: the walk is looking at the wrong place", len(files))
	}
	for _, name := range files {
		f, err := parser.ParseFile(token.NewFileSet(), name, nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		for _, d := range f.Decls {
			if g, ok := d.(*ast.GenDecl); ok && g.Tok == token.VAR && name == "welfare.go" {
				t.Errorf("%s: package variable at %v, want none (architecture.md 7.1)", name, g.Pos())
			}
		}
		ast.Inspect(f, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.FuncDecl:
				if strings.Contains(strings.ToLower(n.Name.Name), "convert") {
					t.Errorf("%s: function %s is named for converting", name, n.Name.Name)
				}
			case *ast.SelectorExpr:
				if n.Sel.Name == "ID" {
					t.Errorf("%s: reads an ID at %v", name, n.Pos())
				}
			case *ast.CaseClause:
				for _, e := range n.List {
					if bl, ok := e.(*ast.BasicLit); ok && bl.Kind == token.STRING {
						t.Errorf("%s: case on a string literal at %v", name, bl.Pos())
					}
				}
			case *ast.BinaryExpr:
				if n.Op == token.EQL || n.Op == token.NEQ {
					for _, e := range []ast.Expr{n.X, n.Y} {
						if bl, ok := e.(*ast.BasicLit); ok && bl.Kind == token.STRING {
							t.Errorf("%s: compares with a string literal at %v", name, bl.Pos())
						}
					}
				}
			}
			return true
		})
	}
}

// TestParamsOf_AndScratchGrowth verifies that the welfare parameters of a role
// come from the role row and the world (farmer: window 2 * 60 = 120, floor
// 4 000, rest band 500, default ratio 10 000), and that the scratch grows for a
// slot it was not built for, as when an agent enters.
//
// Requirement: FR-WEL-009
func TestParamsOf_AndScratchGrowth(t *testing.T) {
	t.Parallel()
	c := welfareConfig()
	tab := newRoleTable(c)
	got := paramsOf(c, 1, &tab) // canonical order: baker, farmer, trader
	if want := (welfareParams{window: 120, minShareBP: 4000, restBand: 500, defaultBP: 10000}); got != want {
		t.Errorf("paramsOf(farmer) = %+v, want %+v", got, want)
	}
	s := newWelfareScratch(c, 1)
	s.addReceipt(5, 9)
	if len(s.slots) != 6 || s.slots[5].receipts != 9 {
		t.Errorf("scratch after a receipt for slot 5: %d slots, receipts %d, want 6 slots, 9", len(s.slots), s.slots[5].receipts)
	}
}

// TestRecord_PlanLiterals verifies the plan's scripted tick: receipts 900,
// input spend 200 and wear 40 (WearPerUse 4 at price 100, durability 10:
// ceil(400 / 10) = 40) give income 900 - 200 - 40 = 660; an upkeep of 100 food
// at 10 at weight 1 is cost 100 * 10 * 1 = 1000; a one-tick ring reads
// 660 * 10 000 / 1000 = 6600. Only addInputSpend nets a spend: the scratch has
// no method for an Upkeep, Tool or Other spend, so no path can change the income
// by one, and the income is unchanged by a record of the same tick read twice.
//
// Requirement: FR-WEL-001, FR-WEL-002, FR-WEL-003, FR-WEL-004
func TestRecord_PlanLiterals(t *testing.T) {
	t.Parallel()
	c := welfareConfig()
	c.Commodities[1].Durability = 10
	s := newWelfareScratch(c, 1)
	s.addReceipt(0, 900)
	s.addInputSpend(0, 200)
	s.addWear(0, 1, 1, 4)
	upkeep := []itemRow{{commodity: 0, quantity: 100}}
	prices := []int64{10, 100, 10}
	rec := s.record(0, 1, true, 1, 0, prices, upkeep)
	if rec.Income != 660 || rec.Cost != 1000 {
		t.Fatalf("record income, cost = %d, %d, want 660, 1000", rec.Income, rec.Cost)
	}
	if again := s.record(0, 1, true, 1, 0, prices, upkeep); again != rec {
		t.Errorf("second record = %+v, want %+v (record does not mutate the scratch)", again, rec)
	}
	ring, err := welfare.NewRing(1, MaxWelfareWindow)
	if err != nil {
		t.Fatal(err)
	}
	ring.Push(rec)
	if r, ok := ring.Ratio(1000, 10000); !ok || r != 6600 {
		t.Errorf("ratio = %d, %v, want 6600, true", r, ok)
	}
}

// TestRecord_SaturatesOnWearAndCostOverflow verifies D63 for the two other
// overflow sites of record: a wear valuation that overflows (executions at
// MaxInt64 of a wear costing 40) gives income -MaxInt64 (the debit saturates at
// MaxInt64 and 0 less that is exact), and a cost of tick that overflows gives MaxInt64;
// neither panics, and Push then clamps the record and reports it.
//
// Requirement: FR-WEL-010, FR-WEL-042
func TestRecord_SaturatesOnWearAndCostOverflow(t *testing.T) {
	t.Parallel()
	c := welfareConfig()
	c.Commodities[1].Durability = 10
	s := newWelfareScratch(c, 2)
	s.addWear(0, 1, math.MaxInt64, 4)
	rec := s.record(0, 1, true, 1, 0, []int64{10, 100, 10}, nil)
	if rec.Income != -math.MaxInt64 {
		t.Errorf("income with an overflowing wear = %d, want -MaxInt64", rec.Income)
	}
	ring, err := welfare.NewRing(5, MaxWelfareWindow)
	if err != nil {
		t.Fatal(err)
	}
	if !ring.Push(rec) {
		t.Errorf("Push of a wear-overflow record reported no clamp, want one")
	}
	huge := []itemRow{{commodity: 0, quantity: math.MaxInt64}}
	rec = s.record(1, 3, true, 1, 0, []int64{10, 100, 10}, huge)
	if rec.Cost != math.MaxInt64 {
		t.Fatalf("cost of an overflowing tick = %d, want MaxInt64", rec.Cost)
	}
	ring, err = welfare.NewRing(5, MaxWelfareWindow)
	if err != nil {
		t.Fatal(err)
	}
	if !ring.Push(rec) || !ring.AtBound() {
		t.Errorf("Push of a cost-overflow record: clamp not reported or ring not at bound, want both")
	}
}

// TestNewTarget_LowestReachableIsAtLeast10000 verifies that, within the bounds
// validation allows (FR-WEL-025: the lowest possible target is at least
// 10 000), no draw falls below 10 000: TargetRatio 10 000 at spread 0 gives
// exactly 10 000, and TargetRatio 15 000 at spread 3 333 has the lowest target
// 15 000 * 6 667 / 10 000 = 10 000 (truncated), so every draw of 200 seeds is
// within [10 000, 19 999]. newTarget itself does not refuse a lower input (a
// TargetRatio 10 000 at spread 9 999 reaches 1); the validator does.
//
// Requirement: FR-WEL-024, FR-WEL-025
func TestNewTarget_LowestReachableIsAtLeast10000(t *testing.T) {
	t.Parallel()
	s := newStream(1)
	s.keyed(siteConstruction, 0, 1)
	if got := newTarget(s, 10000, 0); got != 10000 {
		t.Errorf("target of TargetRatio 10000 at spread 0 = %d, want 10000", got)
	}
	for id := range int64(200) {
		s.keyed(siteConstruction, 0, id)
		if got := newTarget(s, 15000, 3333); got < 10000 || got > 19999 {
			t.Errorf("target at spread 3333, id %d = %d, want within [10000, 19999]", id, got)
		}
	}
}

// TestNewTarget_SpreadWidensTheDraw verifies that the spread reaches the draw:
// over 200 agents at TargetRatio 15 000 and spread 3 000 the targets take many
// distinct values, both below and above the TargetRatio, inside
// [10 500, 19 500], whereas at spread 0 every target is the TargetRatio.
//
// Guard: the single-draw check of TestNewTarget cannot tell a spread that is
// applied from one that is dropped, since a dropped spread also lands inside
// the range; this one fails if the targets of a whole population collapse to
// the TargetRatio.
//
// Requirement: FR-WEL-024
func TestNewTarget_SpreadWidensTheDraw(t *testing.T) {
	t.Parallel()
	s := newStream(9)
	distinct := map[int64]bool{}
	var below, above bool
	for id := range int64(200) {
		s.keyed(siteConstruction, 0, id)
		got := newTarget(s, 15000, 3000)
		if got < 10500 || got > 19500 {
			t.Fatalf("id %d: target = %d, want within [10500, 19500]", id, got)
		}
		distinct[got] = true
		below, above = below || got < 15000, above || got > 15000
		s.keyed(siteConstruction, 0, id)
		if flat := newTarget(s, 15000, 0); flat != 15000 {
			t.Fatalf("id %d: target at spread 0 = %d, want 15000", id, flat)
		}
	}
	if len(distinct) < 50 || !below || !above {
		t.Errorf("200 targets at spread 3000: %d distinct, below %v, above %v, want at least 50 distinct and both sides of 15000", len(distinct), below, above)
	}
}

// TestAddWear_MergesOnlyOnTheSameCommodityAndWearPerUse verifies that a row of
// another commodity, below or above, and a row of another wear per use, below
// or above, are rows of their own, and that a row with both equal merges.
//
// Requirement: FR-WEL-003
func TestAddWear_MergesOnlyOnTheSameCommodityAndWearPerUse(t *testing.T) {
	t.Parallel()
	s := newWelfareScratch(welfareConfig(), 2)
	s.addWear(0, 1, 1, 4)
	s.addWear(0, 0, 1, 4) // commodity below
	s.addWear(0, 2, 1, 4) // commodity above
	if f := s.at(0); f.nWear != 3 {
		t.Fatalf("rows after three commodities = %d, want 3", f.nWear)
	}
	s.addWear(1, 1, 1, 4)
	s.addWear(1, 1, 1, 3) // wear per use below
	s.addWear(1, 1, 1, 5) // wear per use above
	s.addWear(1, 1, 2, 4) // equal in both: merges
	f := s.at(1)
	if f.nWear != 3 || f.wear[0].executions != 3 || f.wear[1].executions != 1 || f.wear[2].executions != 1 {
		t.Errorf("rows = %d, executions %d, %d, %d, want 3 rows of 3, 1, 1", f.nWear, f.wear[0].executions, f.wear[1].executions, f.wear[2].executions)
	}
}

// TestRecord_WearAtFullCapacity verifies that MaxRequires wear rows are all
// valued and netted: 2 rows of wear 4 and 2 of wear 3 on tools of durability 4
// at price 10 cost 10 + 10 + 8 + 8 = 36.
//
// Requirement: FR-WEL-003
func TestRecord_WearAtFullCapacity(t *testing.T) {
	t.Parallel()
	s := newWelfareScratch(welfareConfig(), 1)
	s.addWear(0, 1, 1, 4)
	s.addWear(0, 2, 1, 4)
	s.addWear(0, 1, 1, 3)
	s.addWear(0, 2, 1, 3)
	if got := s.record(0, 1, true, 1, 0, []int64{10, 10, 10}, nil).Income; got != -36 {
		t.Errorf("income with %d wear rows = %d, want -36", MaxRequires, got)
	}
}

// TestRuns_EdgeInputs verifies the guards of the runs on inputs outside the
// ordinary range: a held count with no (or negative) households, a count not
// above half, and a negative clamped count make no run.
//
// Requirement: FR-WEL-033, FR-WEL-042
func TestRuns_EdgeInputs(t *testing.T) {
	t.Parallel()
	if run, w := floorRun(0, 1, 0); run != 0 || w {
		t.Errorf("floorRun(0, 1, 0) = %d, %v, want 0, false", run, w)
	}
	if run, w := floorRun(0, 1, -2); run != 0 || w {
		t.Errorf("floorRun(0, 1, -2) = %d, %v, want 0, false", run, w)
	}
	if run, w := floorRun(0, 0, 4); run != 0 || w {
		t.Errorf("floorRun(0, 0, 4) = %d, %v, want 0, false", run, w)
	}
	if run, w := clampRun(0, -1); run != 0 || w {
		t.Errorf("clampRun(0, -1) = %d, %v, want 0, false", run, w)
	}
	if run, w := clampRun(0, 1); run != 1 || !w {
		t.Errorf("clampRun(0, 1) = %d, %v, want 1, true", run, w)
	}
}

// TestWorkingTick_NegativeExecutionsIsNotWork verifies that a role with
// recipes is working only for a positive count of executions.
//
// Requirement: FR-WEL-017
func TestWorkingTick_NegativeExecutionsIsNotWork(t *testing.T) {
	t.Parallel()
	if w, _, _ := workingTick(true, -1, false, false, false, 3); w {
		t.Errorf("workingTick with k = -1 working = true, want false")
	}
}
