package economysim_test

import (
	"fmt"
	"log/slog"
	"reflect"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// This file holds the migration scenarios S61 to S66 (plan 2-14 Phase 8), the
// determinism and fork tests of the module, and what they share. The cold packs move
// nobody with the defaults: a household of pack two holds none of three upkeep goods at
// the end of the tick, so every ranked household is unprovisioned for a journey (MIG
// 7.4.1). The scenarios therefore run on movingPack, pack two or three with its trade,
// information and migration overlays whose world upkeep is the staple alone (what a
// household holds at the end of a tick), the review every 10 ticks and the gate open.
// They are short on purpose: 240 ticks and two seeds, except S66, which takes
// scenarioSeedCount() seeds of 200 ticks.

const (
	migTicks   = 240
	migSeeds   = 2
	migS66Tick = 200
	migReview  = 10
)

// movingPack is the module pack of the variant with the changes that let households
// move: the staple alone as the upkeep (with the reference basket of its price), the
// gate open, a review every migReview ticks and the cooldown of the welfare window.
// mut edits the configuration last.
func movingPack(t testing.TB, layers []string, mut func(*economysim.Config)) economysim.Config {
	t.Helper()
	cfg := loadValidPack(t, layers...)
	i := slices.IndexFunc(cfg.Commodities, func(c economysim.Commodity) bool { return c.ID == "staple" })
	if i < 0 {
		t.Fatalf("the pack has no staple")
	}
	cfg.World.Upkeep = []economysim.Item{{Commodity: "staple", Quantity: 1}}
	for r := range cfg.Roles {
		cfg.Roles[r].Upkeep = nil
	}
	for r := range cfg.Regions {
		cfg.Regions[r].ReferenceBasket = economysim.Money(cfg.Commodities[i].ReferencePrice)
	}
	m := cfg.Modules.Migration
	m.MoveBelow, m.MoveReviewTicks, m.MoveCooldown = economysim.MaxBasisPoints, migReview, cfg.World.WelfareWindow
	if mut != nil {
		mut(&cfg)
	}
	return cfg
}

// migTally is what a run of a migration scenario kept.
type migTally struct {
	started, arrived, ended, pushed int64
	outcomes                        [11]int64
	arrivedTo                       []int64
	startedFrom                     map[economysim.RoleID]int64
	households                      [][]int64 // by tick, by region, at the boundary
	peakFlight, endFlight           int64
	capBreaches                     int
	rebuildMismatches               int
	reviewMismatches                int
	events                          []economysim.Event
	lines                           []logLine
}

// runMigration runs the world of cfg for ticks with the audit, an observer and a
// logger, and takes a snapshot at every boundary: the households by region, the
// households on the way, the pipeline against its cap, and the household count of each
// region rebuilt from the events against the snapshot. before is called before each
// step (a shock). The failing seed is in every message.
func runMigration(t testing.TB, cfg economysim.Config, seed uint64, ticks int, before func(w *economysim.World, tick int64)) migTally {
	t.Helper()
	scenSem <- struct{}{}
	defer func() { <-scenSem }()
	rec, lg := &recorder{}, newCaptureHandler()
	w, err := economysim.NewWorld(cfg, economysim.WithSeed(seed), economysim.WithAudit(), economysim.WithObserver(rec), economysim.WithLogger(slog.New(lg)))
	if err != nil {
		t.Fatalf("seed %d: NewWorld() error = %v, want nil", seed, err)
	}
	tl := migTally{arrivedTo: make([]int64, len(cfg.Regions)), startedFrom: map[economysim.RoleID]int64{}}
	rebuilt := make([]int64, len(cfg.Regions))
	seen, maxWeight := 0, int64(1)
	for tick := range int64(ticks) {
		if before != nil {
			before(w, tick)
		}
		if err := w.Step(); err != nil {
			t.Fatalf("seed %d: Step() at tick %d error = %v, want nil (the audit failed)", seed, tick, err)
		}
		for ; seen < len(rec.events); seen++ {
			tl.absorb(rec.events[seen], rebuilt, &maxWeight)
		}
		s, err := w.Snapshot()
		if err != nil {
			t.Fatalf("seed %d: Snapshot() at tick %d error = %v, want nil", seed, tick, err)
		}
		tl.boundary(cfg, &s, rebuilt, maxWeight)
	}
	tl.events = rec.events
	tl.lines = lg.sink.lines
	return tl
}

// absorb counts one event: the starts, the arrivals, the ends, the outcomes and the
// households of each region rebuilt from the creation and relocation events.
func (tl *migTally) absorb(e economysim.Event, rebuilt []int64, maxWeight *int64) {
	switch e := e.(type) {
	case economysim.AgentSpawned:
		rebuilt[e.Region] += e.Weight
		*maxWeight = max(*maxWeight, e.Weight)
	case economysim.AgentBankrupted:
		rebuilt[e.Region] -= e.Weight
	case economysim.WeightChanged:
		rebuilt[e.Region] += e.To - e.From
	case economysim.AgentRelocated:
		rebuilt[e.From] -= e.Weight
		rebuilt[e.To] += e.Weight
		if e.Cause == economysim.CauseMigration {
			tl.arrived += e.Weight
			tl.arrivedTo[e.To] += e.Weight
		}
	case economysim.MigrationStarted:
		tl.started += e.Weight
		tl.startedFrom[e.Role] += e.Weight
	case economysim.MigrationReviewed:
		tl.pushed += e.Pushed
		tl.ended += e.Ended
		o := [11]int64{e.Started, e.NoEstimate, e.NoDestination, e.NoRumour, e.TooFar, e.NoViable, e.NoGain, e.Bound, e.OutCap, e.Unprovisioned, e.InCap}
		var sum int64
		for i, n := range o {
			tl.outcomes[i] += n
			sum += n
		}
		if sum != e.Pushed {
			tl.reviewMismatches++
		}
	}
}

// boundary takes the figures of one boundary: the households by region, the
// pipeline against its cap, the peak households on the way and the event rebuild.
func (tl *migTally) boundary(cfg economysim.Config, s *economysim.Snapshot, rebuilt []int64, maxWeight int64) {
	row := make([]int64, len(s.Regions))
	var inFlight int64
	for g, r := range s.Regions {
		row[g] = r.Households
		inFlight += r.OnJourneyFrom
		if r.Households != rebuilt[g] {
			tl.rebuildMismatches++
		}
		// The pipeline cap counts households at the pass: ceil(households * share) plus one
		// agent's weight less 1. The households fall after the pass by exits of the same
		// tick, so the larger of this boundary and the last is the bound.
		share, households := cfg.Modules.Migration.MoveInShare, r.Households
		if len(tl.households) > 0 {
			households = max(households, tl.households[len(tl.households)-1][g])
		}
		if limit := (households*share+9_999)/10_000 + maxWeight - 1; r.OnJourneyTo > limit {
			tl.capBreaches++
		}
	}
	tl.households = append(tl.households, row)
	tl.peakFlight, tl.endFlight = max(tl.peakFlight, inFlight), inFlight
}

// migCheck records the verdict of an exact check and fails the test when it does not
// hold; a band is checked only when its key has a value (REF 7.1.5).
func migCheck(t testing.TB, rep *scenReport, seed uint64, name string, ok bool, detail string) {
	t.Helper()
	rep.check(seed, name, ok, detail)
	if !ok {
		t.Errorf("seed %d: %s: %s", seed, name, detail)
	}
}

// bandCheck reports a band check: it asserts only when the band has a value.
func bandCheck(t testing.TB, rep *scenReport, seed uint64, bands, key string, ok func(band int64) bool, detail string) {
	t.Helper()
	if v, set := bandIn(t, "testdata/chains/bands/"+bands+".toml", key); set {
		migCheck(t, rep, seed, key, ok(v), detail)
	}
}

// regionNamed returns the RegionID of a region of the configuration.
func regionNamed(t testing.TB, cfg economysim.Config, id string) economysim.RegionID {
	t.Helper()
	i := slices.IndexFunc(cfg.Regions, func(r economysim.Region) bool { return r.ID == id })
	if i < 0 {
		t.Fatalf("the pack has no region %q", id)
	}
	return economysim.RegionID(i)
}

// TestScenarioS61_MigrationCobweb runs scenario S61 (MIG-S1, plan 2-14): pack two with
// the module on, a yield shock that halves the staple of the north from tick 40, and
// households free to move. It asserts that the households on the way never exceed the
// pipeline cap, that every household started is accounted for (arrived, on the way or
// ended), and it reports the households
// that moved against the net change of the north: journeys_factor is the ratio of the
// two, asserted within the band once the band is set.
//
// Requirement: FR-MIG-021, FR-MIG-023, FR-MIG-005
//
// Guard: a cap that counted agents, or a journey that dropped a household, would show
// as a breach or an unaccounted household.
func TestScenarioS61_MigrationCobweb(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	rep := newPackScenReport("two", "S61 the migration cobweb after a yield shock (MIG-S1)")
	cfg := movingPack(t, packMigration("two"), nil)
	north, grow := regionNamed(t, cfg, "north"), slices.IndexFunc(cfg.Recipes, func(r economysim.Recipe) bool { return r.ID == "grow_staple" })
	for seed := uint64(1); seed <= migSeeds; seed++ {
		tl := runMigration(t, cfg, seed, migTicks, func(w *economysim.World, tick int64) {
			if tick == 40 {
				if err := economysim.SetYieldShockForTest(w, north, economysim.RecipeID(grow), 5_000); err != nil {
					t.Fatalf("seed %d: SetYieldShockForTest() = %v, want nil", seed, err)
				}
			}
		})
		first, last := tl.households[0], tl.households[len(tl.households)-1]
		net := max(first[north]-last[north], last[north]-first[north])
		rep.num(seed, "s61.households_started", tl.started)
		rep.num(seed, "s61.net_change_of_the_north", net)
		rep.num(seed, "s61.peak_households_on_the_way", tl.peakFlight)
		migCheck(t, rep, seed, "s61.pipeline_within_cap", tl.capBreaches == 0, fmt.Sprintf("%d boundaries with more households on the way to a region than its cap", tl.capBreaches))
		migCheck(t, rep, seed, "s61.every_start_accounted_for", tl.started >= tl.arrived+tl.endFlight+tl.ended, fmt.Sprintf("started %d, arrived %d, on the way %d, ended %d", tl.started, tl.arrived, tl.endFlight, tl.ended))
		if net > 0 {
			factor := tl.started * 1_000 / net
			rep.num(seed, "s61.journeys_factor", factor)
			bandCheck(t, rep, seed, "two", "s61.journeys_factor", func(b int64) bool { return factor <= b }, fmt.Sprintf("journeys factor %d, band %s", factor, "set"))
		} else {
			rep.absent(seed, "s61.journeys_factor")
		}
	}
	rep.flush(t)
}

// TestScenarioS62_Herding runs scenario S62 (MIG-S2) on pack three: households of the
// three regions free to move. It reports how the arrivals divide among the regions and
// how often the difference of the households of the two other regions changes sign
// (herding would alternate it), and asserts the split within the band once it is set.
// It asserts that the arrivals of the regions add up to the arrivals.
//
// Requirement: FR-MIG-022, FR-MIG-023
func TestScenarioS62_Herding(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	rep := newPackScenReport("three", "S62 herding between equal destinations (MIG-S2)")
	cfg := movingPack(t, packMigration("three"), nil)
	for seed := uint64(1); seed <= migSeeds; seed++ {
		tl := runMigration(t, cfg, seed, migTicks, nil)
		var sum int64
		for _, n := range tl.arrivedTo {
			sum += n
		}
		migCheck(t, rep, seed, "s62.arrivals_add_up", sum == tl.arrived, fmt.Sprintf("arrivals by region add to %d, total %d", sum, tl.arrived))
		migCheck(t, rep, seed, "s62.pipeline_within_cap", tl.capBreaches == 0, fmt.Sprintf("%d boundaries over the pipeline cap", tl.capBreaches))
		a, b := tl.arrivedTo[1], tl.arrivedTo[2]
		flips, prev := int64(0), int64(0)
		for _, row := range tl.households {
			if d := row[1] - row[2]; d != 0 {
				if prev != 0 && (d > 0) != (prev > 0) {
					flips++
				}
				prev = d
			}
		}
		rep.num(seed, "s62.arrivals_to_the_second_region", a)
		rep.num(seed, "s62.arrivals_to_the_third_region", b)
		rep.num(seed, "s62.sign_changes_of_the_difference", flips)
		if a+b > 0 {
			split := max(a-b, b-a) * 10_000 / (a + b)
			rep.num(seed, "s62.split_bp", split)
			bandCheck(t, rep, seed, "three", "s62.split_band_bp", func(band int64) bool { return split <= band }, fmt.Sprintf("split %d bp", split))
		}
	}
	rep.flush(t)
}

// TestScenarioS63_ReliefOnOff runs scenario S63 (MIG-S3) on pack three with and without
// the estates overlay (relief). The module reads the earned ratio, which the relief does
// not touch, so the decision to leave is the same; it reports the households started
// in each run and their difference, asserted within the band once it is set.
//
// Requirement: FR-MIG-009
func TestScenarioS63_ReliefOnOff(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	rep := newPackScenReport("three", "S63 relief on and off (MIG-S3)")
	plain := movingPack(t, packMigration("three"), nil)
	relief := movingPack(t, append(packMigration("three"), "overlay/estates.toml"), nil)
	for seed := uint64(1); seed <= migSeeds; seed++ {
		off, on := runMigration(t, plain, seed, migTicks, nil), runMigration(t, relief, seed, migTicks, nil)
		rep.num(seed, "s63.started_without_relief", off.started)
		rep.num(seed, "s63.started_with_relief", on.started)
		migCheck(t, rep, seed, "s63.outcomes_add_to_pushed", off.reviewMismatches+on.reviewMismatches == 0, "a MigrationReviewed whose outcomes do not add to Pushed")
		if off.started > 0 {
			diff := max(off.started-on.started, on.started-off.started) * 10_000 / off.started
			rep.num(seed, "s63.difference_bp", diff)
			bandCheck(t, rep, seed, "three", "s63.journeys_band_bp", func(band int64) bool { return diff <= band }, fmt.Sprintf("difference %d bp", diff))
		}
	}
	rep.flush(t)
}

// TestScenarioS64_Unprovisioned runs scenario S64 (MIG-S4) on pack two with a journey
// longer than the household holds upkeep for (a speed of 20 over an edge of 100 is 5
// ticks, the world covers 3): no household can start, every pushed household that
// reaches admission is counted Unprovisioned, and the Warn "migration blocked" is logged
// at the tenth consecutive tick, naming the outcome that held most households.
//
// Requirement: FR-MIG-019, FR-MIG-038
func TestScenarioS64_Unprovisioned(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	rep := newPackScenReport("two", "S64 a journey the household cannot provision (MIG-S4)")
	cfg := movingPack(t, packMigration("two"), func(c *economysim.Config) { c.Modules.Migration.MoveSpeed = 20 })
	for seed := uint64(1); seed <= migSeeds; seed++ {
		tl := runMigration(t, cfg, seed, migTicks, nil)
		warns := map[string]int{}
		for _, l := range tl.lines {
			if l.level == slog.LevelWarn && l.msg == "migration blocked" {
				for _, a := range l.attrs {
					if a.Key == "reason" {
						warns[a.Value.String()]++
					}
				}
			}
		}
		rep.num(seed, "s64.unprovisioned", tl.outcomes[9])
		rep.num(seed, "s64.blocked_warns", int64(warns["unprovisioned"]))
		migCheck(t, rep, seed, "s64.nobody_started", tl.started == 0, fmt.Sprintf("%d households started a journey they cannot provision", tl.started))
		migCheck(t, rep, seed, "s64.unprovisioned_counted", tl.outcomes[9] > 0, "no household was counted Unprovisioned")
		migCheck(t, rep, seed, "s64.blocked_warn_at_the_tenth_tick", warns["unprovisioned"] > 0, fmt.Sprintf("Warn 'migration blocked' by reason: %v", warns))
	}
	rep.flush(t)
}

// TestScenarioS65_EmptyingAChain runs scenario S65 (MIG-S5) on pack three: it reports the
// least share of its start that any role keeps in any region (asserted within the band
// once set), and, with every role of the first region fixed, asserts that no household
// leaves that region.
//
// Requirement: FR-MIG-006, FR-MIG-043
func TestScenarioS65_EmptyingAChain(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	rep := newPackScenReport("three", "S65 emptying a chain (MIG-S5)")
	free := movingPack(t, packMigration("three"), nil)
	fixed := movingPack(t, packMigration("three"), func(c *economysim.Config) {
		for r := range c.Roles {
			c.Roles[r].Fixed = true
		}
	})
	for seed := uint64(1); seed <= migSeeds; seed++ {
		tl := runMigration(t, free, seed, migTicks, nil)
		rep.num(seed, "s65.households_started", tl.started)
		fx := runMigration(t, fixed, seed, migTicks, nil)
		migCheck(t, rep, seed, "s65.fixed_roles_never_leave", fx.started == 0 && fx.arrived == 0, fmt.Sprintf("with every role fixed %d households started and %d arrived", fx.started, fx.arrived))
		// The share of its start that each region keeps (a region of the free run).
		least := int64(10_000)
		for g := range tl.households[0] {
			if tl.households[0][g] == 0 {
				continue
			}
			for _, row := range tl.households {
				least = min(least, row[g]*10_000/tl.households[0][g])
			}
		}
		rep.num(seed, "s65.least_share_of_its_start_bp", least)
		bandCheck(t, rep, seed, "three", "s65.role_share_bp", func(band int64) bool { return least >= band }, fmt.Sprintf("least share %d bp", least))
	}
	rep.flush(t)
}

// TestScenarioS66_InvariantsOverSeeds runs scenario S66 (MIG-S6): scenarioSeedCount()
// seeds of pack two and of pack three with the module moving households and the audit
// on (so money and every commodity are conserved exactly at every tick), and asserts that
// the households started reconcile with the arrivals, the households on the way and the
// ends; that the outcomes add to Pushed in every MigrationReviewed; that the household
// count of each region rebuilt from the events equals the snapshot at every boundary;
// and that the pipeline never exceeds its cap. The closure leg is plan 17's.
//
// Requirement: FR-MIG-033, FR-MIG-037, FR-MIG-025, FR-STB-013
func TestScenarioS66_InvariantsOverSeeds(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	for _, variant := range []string{"two", "three"} {
		t.Run(variant, func(t *testing.T) {
			t.Parallel()
			rep := newPackScenReport(variant, "S66 the invariants of the migration module over seeds (MIG-S6)")
			cfg := movingPack(t, packMigration(variant), nil)
			for _, seed := range seedList(scenarioSeedCount()) {
				tl := runMigration(t, cfg, seed, migS66Tick, nil)
				rep.num(seed, "s66.households_started", tl.started)
				migCheck(t, rep, seed, "s66.reconciles", tl.started >= tl.arrived+tl.endFlight+tl.ended, fmt.Sprintf("started %d, arrived %d, on the way %d, ended %d", tl.started, tl.arrived, tl.endFlight, tl.ended))
				migCheck(t, rep, seed, "s66.outcomes_add_to_pushed", tl.reviewMismatches == 0, fmt.Sprintf("%d reviews whose outcomes do not add to Pushed", tl.reviewMismatches))
				migCheck(t, rep, seed, "s66.counts_rebuilt_from_events", tl.rebuildMismatches == 0, fmt.Sprintf("%d region boundaries where the events rebuild a household count the snapshot does not hold", tl.rebuildMismatches))
				migCheck(t, rep, seed, "s66.pipeline_within_cap", tl.capBreaches == 0, fmt.Sprintf("%d boundaries over the pipeline cap", tl.capBreaches))
			}
			rep.flush(t)
		})
	}
}

// TestDeterminism_Migration verifies, for the module pack as it is and with households
// moving, that the final state does not depend on the way the run is driven or watched:
// the cases of detCases(true), seed 1, end in the snapshot and state hash of a bare Step
// run.
//
// Requirement: FR-REF-025, FR-MIG-042
//
// Guard: journeys keep state that no other determinism test reaches; a run that differed
// by observation or chunking would show first here.
func TestDeterminism_Migration(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	for _, variant := range []string{"idle", "moving"} {
		t.Run(variant, func(t *testing.T) {
			t.Parallel()
			cfg := loadValidPack(t, packMigration("two")...)
			if variant == "moving" {
				cfg = movingPack(t, packMigration("two"), nil)
			}
			base := runPack(t, cfg, detStep(runSpec{Seed: 1, Ticks: detTicks}))
			for _, tc := range detCases(true) {
				spec := tc.spec()
				spec.Seed, spec.Ticks = 1, detTicks
				got := runPack(t, cfg, spec)
				if !reflect.DeepEqual(got.Final, base.Final) {
					t.Errorf("%s case %q: final snapshot differs from the bare run", variant, tc.name)
				}
				if economysim.HashWorld(got.World) != economysim.HashWorld(base.World) {
					t.Errorf("%s case %q: state hash differs from the bare run", variant, tc.name)
				}
			}
		})
	}
}

// TestFork_Migration verifies the fork-at-N of the moving pack (FR-MIG-041): at the
// start, just after a first departure (a household is on the way at the boundary), after
// its arrival and after a call between ticks (a yield shock), the original and two
// restores give the same events, snapshot, audit and checkpoint bytes for 15 ticks.
// The cell on the way also checks that a household was on a journey at the boundary
// and that it arrived after it.
//
// Requirement: FR-MIG-041, FR-CHK-050, FR-CHK-052, FR-CHK-053
func TestFork_Migration(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := movingPack(t, packMigration("two"), nil)
	// The first tick at which a household starts a journey, from a probe run.
	probe := runMigration(t, cfg, 1, migTicks, nil)
	first := int64(-1)
	for _, e := range probe.events {
		if s, ok := e.(economysim.MigrationStarted); ok {
			first = s.Tick
			break
		}
	}
	if first < 0 {
		t.Fatalf("the probe run started no journey in %d ticks, want one for the cell on the way", migTicks)
	}
	north, grow := regionNamed(t, cfg, "north"), slices.IndexFunc(cfg.Recipes, func(r economysim.Recipe) bool { return r.ID == "grow_staple" })
	for _, tc := range []struct {
		name     string
		n        int
		inFlight bool
		before   func(*economysim.World)
	}{
		{"before the first step", 0, false, nil},
		{"just after a departure", int(first) + 1, true, nil},
		{"after the arrival", int(first) + 4, false, nil},
		{"after a shock between ticks", int(first) + 2, false, func(w *economysim.World) {
			_ = economysim.SetYieldShockForTest(w, north, economysim.RecipeID(grow), 5_000)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var onWay int
			forkAtN(t, forkCase{
				Name: "migration", Cfg: cfg, Seed: 1, N: tc.n, M: 15, Logs: true,
				Before: func(w *economysim.World) {
					if tc.before != nil {
						tc.before(w)
					}
					s, err := w.Snapshot()
					if err != nil {
						t.Fatalf("Snapshot() error = %v, want nil", err)
					}
					onWay = len(s.Journeys)
				},
				Inspect: func(t *testing.T, restored, since []economysim.Event) {
					if tc.inFlight && onWay == 0 {
						t.Errorf("N %d: no household on a journey at the boundary, want the cell to be inside one", tc.n)
					}
					arrived := slices.ContainsFunc(restored, func(e economysim.Event) bool {
						r, ok := e.(economysim.AgentRelocated)
						return ok && r.Cause == economysim.CauseMigration
					})
					if tc.inFlight && !arrived {
						t.Errorf("N %d: the restored world delivered no migration arrival, want the household on the way to arrive", tc.n)
					}
				},
			})
		})
	}
}
