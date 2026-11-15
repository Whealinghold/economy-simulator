package economysim_test

import (
	"math/rand/v2"
	"strings"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// This file holds the scenarios SET-S1 to SET-S3 of spec-settlements.md 13 (S4 is
// TestScenarioSET4_OneClassEqualsOff and S5 TestFork_Settlements and
// TestCheckpoint_Settlements*). They report and assert the invariants of the spec, not
// a band (every default of the module is a placeholder, N-015, N-027), so they write no
// line of expected/*.scenarios.txt: no scenario harness reads one per pack.

// gateSeen is what a view of the recording spawner said about the gated role.
type gateSeen struct {
	tick     int64
	closed   bool
	headroom int64
	noLimit  bool
}

// gateRecorder is a spawner that records the view of one role in its region at each
// tick and delegates to the default: the gate and the ceiling as the spawner sees them.
type gateRecorder struct {
	role economysim.RoleID
	seen []gateSeen
}

func (g *gateRecorder) Decide(v *economysim.RegionView, rng *rand.Rand, d *economysim.Decisions) {
	for _, r := range v.Roles {
		if r.Role == g.role {
			g.seen = append(g.seen, gateSeen{v.Tick, r.Closed, r.Headroom, r.NoLimit})
		}
	}
	economysim.DefaultSpawner{}.Decide(v, rng, d)
}

// roleIndex returns the RoleID of the role in canonical order.
func setRoleID(t testing.TB, c economysim.Config, id string) economysim.RoleID {
	t.Helper()
	for i := range c.Roles {
		if c.Roles[i].ID == id {
			return economysim.RoleID(i)
		}
	}
	t.Fatalf("role %q is not in the configuration", id)
	return 0
}

// seatedWorld builds a world over c driven by a recording
// spawner of the role.
func seatedWorld(t testing.TB, c economysim.Config, role string, seed uint64) (*economysim.World, *gateRecorder, *recorder) {
	t.Helper()
	rec := &recorder{}
	g := &gateRecorder{role: setRoleID(t, c, role)}
	c.World.Spawner = "seat"
	w, err := economysim.NewWorld(c, economysim.WithSeed(seed), economysim.WithObserver(rec), economysim.WithSpawner("seat", g))
	if err != nil {
		t.Fatalf("NewWorld(seed %d) error = %v, want nil", seed, err)
	}
	return w, g, rec
}

// classChanges returns the SettlementClassChanged events of the recorder after the
// founding, in order.
func classChanges(rec *recorder) []economysim.SettlementClassChanged {
	var out []economysim.SettlementClassChanged
	for _, e := range rec.events {
		if x, ok := e.(economysim.SettlementClassChanged); ok && x.Cause != economysim.CauseFounded {
			out = append(out, x)
		}
	}
	return out
}

// heldTicks counts the SettlementHeld events of the recorder.
func heldTicks(rec *recorder) int {
	n := 0
	for _, e := range rec.events {
		if _, ok := e.(economysim.SettlementHeld); ok {
			n++
		}
	}
	return n
}

// growConfig is the reference chain with the module on: a hamlet, a village (650
// households and the flag fair) and a town (1 000, fair), a dwell of 20 and the grower
// needing fair. The region north holds 600 households, so it starts a hamlet.
func growConfig() economysim.Config {
	c := referenceConfig()
	m := economysim.DefaultSettlements()
	m.Dwell = 20
	m.Classes = []economysim.SettlementClass{
		{ID: "hamlet"},
		{ID: "village", Enter: 650, Flags: []string{"fair"}},
		{ID: "town", Enter: 1_000, Flags: []string{"fair"}},
	}
	c.Modules.Settlements = &m
	roleOf(&c, "grower").Needs = "fair"
	return c
}

// growExternals adds n external households of the role to region 0 and returns their IDs.
func growExternals(t testing.TB, w *economysim.World, role economysim.RoleID, n int) []economysim.AgentID {
	t.Helper()
	var ids []economysim.AgentID
	for range n {
		id, err := w.AddAgent(economysim.NewAgent{Region: 0, Role: role})
		if err != nil {
			t.Fatalf("AddAgent() = %v, want nil", err)
		}
		ids = append(ids, id)
	}
	return ids
}

// TestScenarioSET1_GrowingRegion verifies SET-S1 on the reference chain with external
// households added between ticks (the births of the population module do not lift a
// region of the collapsing five across a threshold, so the growth is the consumer's, as
// in SET-S2): a region that grows by 12 households a tick climbs the staircase in at
// most K - 1 changes, all upwards, no two less than the dwell apart, and the gated role
// is closed until the class carries its flag and open from the tick after the change.
//
// Arithmetic: 600 households and 12 more a tick: the count passes 650 at tick 5 but the
// founding holds the class until tick 20 (4.4.3), when it is 840 and the class becomes
// village (enter 650); the next release is tick 40, when it is 1 080 and the class becomes
// town (enter 1 000): two changes, K - 1 = 2. The grower, which needs fair, is closed in
// the hamlet and open once the village is.
//
// Requirement: FR-SET-010, FR-SET-011, FR-SET-022, FR-SET-024, FR-SET-025
func TestScenarioSET1_GrowingRegion(t *testing.T) {
	t.Parallel()
	if testing.Short() || raceEnabled {
		t.Skip("multi-seed test: runs in gate F, not in the short or race runs (testing.md 9.1)")
	}
	for _, seed := range []uint64{1, 2} {
		c := growConfig()
		w, seen, rec := seatedWorld(t, c, "grower", seed)
		for tick := range 60 {
			growExternals(t, w, setRoleID(t, c, "cook"), 12)
			if err := w.Step(); err != nil {
				t.Fatalf("seed %d: Step() at tick %d = %v, want nil", seed, tick, err)
			}
		}
		changes := classChanges(rec)
		if len(changes) == 0 || len(changes) > 2 {
			t.Fatalf("seed %d: %d changes of class, want 1 or 2 (K - 1)", seed, len(changes))
		}
		checkStaircase(t, seed, changes, 20)
		checkGateOpensAfter(t, seed, seen.seen, changes[0].Tick)
		if heldTicks(rec) == 0 {
			t.Errorf("seed %d: no SettlementHeld, want the founding hold and the dwell to show", seed)
		}
	}
}

// checkGateOpensAfter asserts that the grower is closed in every view up to the tick of
// the first change and open in every view after it.
func checkGateOpensAfter(t testing.TB, seed uint64, seen []gateSeen, first int64) {
	t.Helper()
	for _, s := range seen {
		if want := s.tick <= first; s.closed != want {
			t.Errorf("seed %d: grower closed = %v at tick %d, want %v (the gate opens in the tick after the change at %d)", seed, s.closed, s.tick, want, first)
		}
	}
}

// checkStaircase asserts that every change is a growth to a higher class and that no two
// are less than the dwell apart.
func checkStaircase(t testing.TB, seed uint64, changes []economysim.SettlementClassChanged, dwell int64) {
	t.Helper()
	for i, ch := range changes {
		if ch.Cause != economysim.CauseGrew || ch.To <= ch.From {
			t.Errorf("seed %d: change %d = %+v, want grew and upwards", seed, i, ch)
		}
		if i > 0 && ch.Tick-changes[i-1].Tick < dwell {
			t.Errorf("seed %d: changes at ticks %d and %d are less than the dwell %d apart", seed, changes[i-1].Tick, ch.Tick, dwell)
		}
	}
}

// noiseConfig is the reference chain with a region of 590 households (the cooks are 126
// not 136) under a class big at 600 households, so a count that moves between 590 and 615
// straddles the enter threshold. The band and the dwell are the arguments.
func noiseConfig(band, dwell int64) economysim.Config {
	c := referenceConfig()
	for i := range c.Groups {
		if c.Groups[i].Region == "north" && c.Groups[i].Role == "cook" {
			c.Groups[i].Households -= 10
		}
	}
	m := economysim.DefaultSettlements()
	m.BandBP, m.Dwell = band, dwell
	m.Classes = []economysim.SettlementClass{{ID: "small"}, {ID: "big", Enter: 600, Flags: []string{"fair"}}}
	c.Modules.Settlements = &m
	return c
}

// noisyRun runs ticks steps with 25 external households added on an even tick and
// removed on the next, so the count is 615 and 590 in turn, and returns the changes after
// the founding and the held events.
func noisyRun(t testing.TB, c economysim.Config, ticks int) ([]economysim.SettlementClassChanged, int) {
	t.Helper()
	rec := &recorder{}
	w, err := economysim.NewWorld(c, economysim.WithSeed(1), economysim.WithObserver(rec))
	if err != nil {
		t.Fatalf("NewWorld() = %v, want nil", err)
	}
	var ids []economysim.AgentID
	for tick := range ticks {
		if tick%2 == 0 {
			ids = growExternals(t, w, 0, 25)
		} else {
			for _, id := range ids {
				if err := w.RemoveAgent(id); err != nil {
					t.Fatalf("RemoveAgent(%d) = %v, want nil", id, err)
				}
			}
		}
		if err := w.Step(); err != nil {
			t.Fatalf("Step() at tick %d = %v, want nil", tick, err)
		}
	}
	return classChanges(rec), heldTicks(rec)
}

// TestScenarioSET2_NoisyCountOnAThreshold verifies SET-S2: a count that crosses an enter
// threshold every other tick is kept from flapping by the band and the dwell. With the
// default band (10 %) and a dwell of 30 the class changes once, at the first release of
// the founding, and never back; with a band of 0 and a dwell of 0 it changes at every
// crossing; with a band of 0 and a dwell of 5 it is held (SettlementHeld) and changes at
// most every 5 ticks.
//
// Arithmetic: the count is 615 on even ticks and 590 on odd ones; the enter threshold is
// 600 and the leave threshold 600 - 10 % = 540 with the default band, which 590 never
// reaches; with a band of 0 the leave threshold is 600 and 590 is below it.
//
// Requirement: FR-SET-010, FR-SET-011, FR-SET-025
func TestScenarioSET2_NoisyCountOnAThreshold(t *testing.T) {
	t.Parallel()
	if testing.Short() || raceEnabled {
		t.Skip("multi-seed test: runs in gate F, not in the short or race runs (testing.md 9.1)")
	}
	const ticks = 60
	calm, calmHeld := noisyRun(t, noiseConfig(1_000, 30), ticks)
	if calmHeld == 0 || len(calm) != 1 {
		t.Errorf("default band and dwell: changes %+v, held %d, want one change and holds before it", calm, calmHeld)
	} else if calm[0].Tick != 30 {
		t.Errorf("default band and dwell: the change is at tick %d, want 30 (the first release of the founding)", calm[0].Tick)
	}
	checkFlaps(t, ticks)
}

// checkFlaps asserts the two runs without a band or a dwell that bind: with neither the
// class changes at about every tick and nothing is held; with a dwell of 5 changes are
// held, at least 5 ticks apart and at most one in 5.
func checkFlaps(t testing.TB, ticks int) {
	t.Helper()
	flap, flapHeld := noisyRun(t, noiseConfig(0, 0), ticks)
	if len(flap) < ticks/2 || flapHeld != 0 {
		t.Errorf("band 0, dwell 0: %d changes, %d held, want a change at about every tick and no hold", len(flap), flapHeld)
	}
	slow, slowHeld := noisyRun(t, noiseConfig(0, 5), ticks)
	if n := len(slow); slowHeld == 0 || n < 2 || n > ticks/5+1 {
		t.Errorf("band 0, dwell 5: %d changes, %d held, want holds and 2 to %d changes", n, slowHeld, ticks/5+1)
	}
	checkSpacing(t, slow, 5)
}

// checkSpacing asserts that no two changes are less than the dwell apart.
func checkSpacing(t testing.TB, changes []economysim.SettlementClassChanged, dwell int64) {
	t.Helper()
	for i := 1; i < len(changes); i++ {
		if changes[i].Tick-changes[i-1].Tick < dwell {
			t.Errorf("changes at ticks %d and %d are less than the dwell %d apart", changes[i-1].Tick, changes[i].Tick, dwell)
		}
	}
}

// famineSettlements is the famine pack with the module on: a class small and a class
// big (500 households, the flag farm), the growers needing farm. The pack starts with
// 600 households; the class is big while the count is 500 or more and falls to small
// when it is below 450 (the band), unless the floor holds it at big.
func famineSettlements(t testing.TB, enter int64, floor string) economysim.Config {
	t.Helper()
	c := loadValidPack(t, packLayers["famine"]...)
	m := economysim.DefaultSettlements()
	m.Dwell = 10
	m.Classes = []economysim.SettlementClass{{ID: "small"}, {ID: "big", Enter: enter, Flags: []string{"farm"}}}
	if floor != "" {
		m.Regions = []economysim.RegionSettlement{{Region: "north", ThresholdBP: 10_000, Floor: floor}}
	}
	c.Modules.Settlements = &m
	roleOf(&c, "grower").Needs = "farm"
	return c
}

// TestScenarioSET3_FamineWithAGatedRole verifies SET-S3: on the famine pack with the
// grower, which the upkeep needs, gated by a flag the class of a shrinking region loses,
// the gate follows the class (closed exactly while the class lacks the flag), no household
// is removed by it, a floor keeps the class and so the gate open, and a pack that starts
// without the flag raises needs_unmet_at_start. The grower's households at the end are
// reported, not asserted: whether the gate helps a region recover cannot be told from the
// baseline (R2, R10).
//
// Requirement: FR-SET-007, FR-SET-022, FR-SET-028
func TestScenarioSET3_FamineWithAGatedRole(t *testing.T) {
	t.Parallel()
	if testing.Short() || raceEnabled {
		t.Skip("multi-seed test: runs in gate F, not in the short or race runs (testing.md 9.1)")
	}
	for _, floor := range []string{"", "big"} {
		c := famineSettlements(t, 500, floor)
		w, seen, rec := seatedWorld(t, c, "grower", 1)
		for tick := range 200 {
			if err := w.Step(); err != nil {
				t.Fatalf("floor %q: Step() at tick %d = %v, want nil", floor, tick, err)
			}
		}
		snap, err := w.Snapshot()
		if err != nil {
			t.Fatalf("Snapshot() = %v, want nil", err)
		}
		changes := classChanges(rec)
		t.Logf("floor %q: %d changes of class, final class %d, %d households", floor, len(changes), snap.Regions[0].Class, snap.Regions[0].Households)
		if floor == "big" {
			checkFloorHeld(t, snap, changes)
		}
		checkGateFollowsClass(t, floor, seen.seen, changes)
	}
	warned := func(enter int64) bool {
		for _, w := range famineSettlements(t, enter, "").Warnings() {
			if w.Rule == "needs_unmet_at_start" && strings.Join(w.Path, ".") == "roles.grower.needs" {
				return true
			}
		}
		return false
	}
	if warned(500) || !warned(700) {
		t.Errorf("needs_unmet_at_start = %v with enter 500 and %v with enter 700, want false and true", warned(500), warned(700))
	}
}

// checkFloorHeld asserts that a region with a floor of the class big ends in it and
// never changes to a class below it.
func checkFloorHeld(t testing.TB, snap economysim.Snapshot, changes []economysim.SettlementClassChanged) {
	t.Helper()
	if snap.Regions[0].Class != 1 {
		t.Errorf("floor big: final class %d, want 1 (never below the floor)", snap.Regions[0].Class)
	}
	for _, ch := range changes {
		if ch.To < 1 {
			t.Errorf("floor big: a change to class %d, want none below the floor", ch.To)
		}
	}
}

// checkGateFollowsClass asserts that the grower is closed in a view exactly while the
// class of the region is the small one, the founding class being big.
func checkGateFollowsClass(t testing.TB, floor string, seen []gateSeen, changes []economysim.SettlementClassChanged) {
	t.Helper()
	class := economysim.SettlementClassID(1) // 600 households at the founding: big
	next := 0
	for _, s := range seen {
		for next < len(changes) && changes[next].Tick < s.tick {
			class = changes[next].To
			next++
		}
		if s.closed != (class == 0) {
			t.Errorf("floor %q: grower closed = %v at tick %d in class %d, want closed exactly in small", floor, s.closed, s.tick, class)
		}
	}
}
