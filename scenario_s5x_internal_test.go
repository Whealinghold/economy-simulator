package economysim

import (
	"slices"
	"testing"
)

// This file holds the scenarios of the external module that are built in Go on small
// worlds (spec-stability.md 8, S52 to S54; plan 2-16 Phase 7, D539): they need the
// public edge-status setter and worlds with the shape
// the cold packs lack (an import-only upkeep, a cap, a window). The packs' scenarios
// (S51, S55, S71) are in scenario_s5x_test.go.

// importOnly returns the world of example A with the upkeep salt, which only the outside
// sells, and every household bidding for one unit a tick with money to pay.
func importOnly(t *testing.T) (*World, *evLog) {
	t.Helper()
	c := extConfig()
	c.World.Upkeep = []Item{{Commodity: "salt", Quantity: 1}}
	c.Commodities = c.Commodities[1:] // bread, the old upkeep, is gone
	for i := range c.Regions {
		c.Regions[i].ReferenceBasket = 40 // the reference price of salt times the upkeep of 1
	}
	w, s, rec := extWorld(t, c)
	salt := commodityOf(t, w, "salt")
	for i := range 7 {
		id := AgentID(w.firstHousehold + i)
		fund(t, w, id, 100_000)
		s.plan[id] = []Intent{{Commodity: salt, Side: Bid, Quantity: 2, Limit: 60}}
	}
	return w, rec
}

// TestScenarioS52_ImportOnlyUpkeep (EXT-S2) runs a realm whose only upkeep good is
// imported, twice for 12 ticks with the audit on: with the edge open throughout, and
// with the edge from the node closed from tick 6. It asserts that while the edge is open
// the outside sells the units the households want and the counter CutClosed is 0, that
// after the closure the sell side offers nothing and counts its whole capacity as
// CutClosed every tick (30 a tick), and that no household goes unfed with the edge open
// in the ticks the closure has not reached. The tick of the first unfed households and
// the failure ladder after it are reported.
//
// Requirement: FR-EXT-032, FR-EXT-035
//
// Guard: a closed edge that still placed an order, or one that counted nothing, would
// hide the single-source hazard (L101) from the report.
func TestScenarioS52_ImportOnlyUpkeep(t *testing.T) {
	t.Parallel()
	const closeAt, ticks = 6, 12
	for _, closed := range []bool{false, true} {
		w, rec := importOnly(t)
		for i := range ticks {
			if closed && i == closeAt {
				if err := w.SetEdgeStatuses([]EdgeStatus{{From: 0, To: 1, Closed: true, FactorBP: 10_000}}); err != nil {
					t.Fatalf("SetEdgeStatuses() = %v, want nil", err)
				}
			}
			step(t, w, 1)
		}
		var cut int64
		first := int64(-1)
		for _, e := range of[OutsideTraded](rec) {
			if e.Commodity != commodityOf(t, w, "salt") {
				continue
			}
			cut += e.CutClosed
			if e.Tick < closeAt || !closed {
				if e.CutClosed != 0 || e.Offered != 30 {
					t.Errorf("closed %t tick %d: salt event %+v, want 30 offered and none cut while the edge is open", closed, e.Tick, e)
				}
			} else if e.CutClosed != 30 || e.Offered != 0 {
				t.Errorf("closed %t tick %d: salt event %+v, want 30 cut by the closed edge and none offered", closed, e.Tick, e)
			}
		}
		for _, e := range of[UpkeepShortfall](rec) {
			if e.Unfed > 0 && first < 0 {
				first = e.Tick
			}
		}
		if want := int64(30 * (ticks - closeAt)); closed && cut != want || !closed && cut != 0 {
			t.Errorf("closed %t: CutClosed over %d ticks = %d, want %d", closed, ticks, cut, map[bool]int64{false: 0, true: want}[closed])
		}
		t.Logf("closed from tick %d = %t: first tick with unfed households %d (-1 none in %d ticks), salt units cut by the edge %d (reported)", closeAt, closed, first, ticks, cut)
	}
}

// TestScenarioS53_WalkAndTrend (EXT-S3) checks the warning and the bounds of a walking
// price. The warning walk_faster_than_beliefs is raised for a step of BeliefStep + 1 and
// not for BeliefStep, and for a trend whose step per tick takes it over; over three
// seeds and 40 ticks the drift of a walking line never leaves its bound and the mid
// stays in [1, MaxPrice] (NFR-EXT-004); the range of the mid, the lag the realm's
// belief behind it can reach, is reported.
//
// Requirement: FR-EXT-022, NFR-EXT-004
//
// Guard: a walk step above what beliefs follow is a heuristic warning; it must not be
// lost, and the clamp of the drift is the bound the stability register relies on (D66).
func TestScenarioS53_WalkAndTrend(t *testing.T) {
	t.Parallel()
	warned := func(step, trendBP int64) bool {
		c := extConfig()
		l := &c.Modules.External.Nodes[0].Lines[0]
		l.WalkStepBP, l.WalkBoundBP, l.TrendBP = step, max(step, 1), trendBP
		return slices.ContainsFunc(c.Warnings(), func(w Warning) bool { return w.Rule == "walk_faster_than_beliefs" })
	}
	belief := extConfig().World.BeliefStep
	for _, tt := range []struct {
		name    string
		step    int64
		trendBP int64
		want    bool
	}{
		{"step at one belief step", belief, 0, false},
		{"step above one belief step", belief + 1, 0, true},
		{"a trend that takes it over", belief, 365 * 24 * 24 * 100, true},
	} {
		if got := warned(tt.step, tt.trendBP); got != tt.want {
			t.Errorf("%s: walk_faster_than_beliefs raised = %t, want %t", tt.name, got, tt.want)
		}
	}
	for _, seed := range []uint64{1, 2, 3} {
		c := extConfig()
		l := &c.Modules.External.Nodes[0].Lines[0]
		l.WalkStepBP, l.WalkRevertBP, l.WalkBoundBP = 400, 200, 1_000
		w, _, _ := extWorld(t, c, WithSeed(seed))
		lo, hi := int64(MaxPrice), int64(0)
		for range 40 {
			step(t, w, 1)
			ln := &w.external.lines[1] // the salt line: canonical order is grain, salt
			mid, _ := w.external.midOf(w, ln)
			if ln.drift < -1_000 || ln.drift > 1_000 || mid < 1 || mid > MaxPrice {
				t.Fatalf("seed %d tick %d: drift %d, mid %d, want |drift| <= 1000 and the mid in [1, MaxPrice]", seed, w.Tick(), ln.drift, mid)
			}
			lo, hi = min(lo, mid), max(hi, mid)
		}
		t.Logf("seed %d: salt mid range %d..%d over 40 ticks (reported)", seed, lo, hi)
	}
}

// TestScenarioS54_CapAndWindow (EXT-S4) runs the ore side of example B, which buys 200 a
// tick up to 1 500 units, against a miner that supplies 200 a tick for 10 ticks: it
// asserts the export cliff (200 traded in ticks 0 to 6, 100 in tick 7, none after),
// the glut after the end (the unsold ore stays with the miner: 10 * 200 - 1 500 = 500),
// and OutsideEnded once; then a window of 50 % (a half of the capacity: 100 wanted a
// tick, no unit cut) against the same supply.
//
// Requirement: FR-EXT-029, FR-EXT-030
//
// Guard: the cliff is the hazard L97 names; the end must be counted (one event) and the
// units the outside no longer takes must stay in the realm, not vanish.
func TestScenarioS54_CapAndWindow(t *testing.T) {
	t.Parallel()
	run := func(c Config) (*World, *evLog, CommodityID) {
		w, s, rec := extWorld(t, c)
		ore := commodityOf(t, w, "ore")
		miner := AgentID(w.firstHousehold)
		s.plan[miner] = []Intent{{Commodity: ore, Side: Ask, Quantity: 200, Limit: 90}}
		for range 10 {
			give(t, w, miner, ore, 200)
			step(t, w, 1)
		}
		return w, rec, ore
	}
	w, rec, ore := run(capConfig())
	var series []int64
	for _, e := range of[OutsideTraded](rec) {
		if e.Commodity == ore {
			series = append(series, e.Traded)
		}
	}
	if want := []int64{200, 200, 200, 200, 200, 200, 200, 100}; !slices.Equal(series, want) {
		t.Errorf("ore traded per tick = %v, want %v then nothing: the cliff at the cap", series, want)
	}
	miner := AgentID(w.firstHousehold)
	if held, _, err := w.ledger.Holding(w.agents[miner].account, int(ore)); err != nil || held != 500 {
		t.Errorf("the miner holds %d ore after 10 ticks (%v), want 500 (2 000 supplied less 1 500 bought)", held, err)
	}
	if n := len(of[OutsideEnded](rec)); n < 1 {
		t.Errorf("OutsideEnded events = %d, want the ore side's end counted", n)
	}

	c := capConfig()
	c.Modules.External.Nodes[0].Lines[0].Buy.TotalUnits = 0
	for _, s := range c.Calendar.Seasons {
		c.Modules.External.Nodes[0].Lines[0].Buy.Window = append(c.Modules.External.Nodes[0].Lines[0].Buy.Window, SeasonFactor{Season: s.ID, BP: 5_000})
	}
	_, half, ore2 := run(c)
	for _, e := range of[OutsideTraded](half) {
		if e.Commodity == ore2 && (e.Wanted != 100 || e.CutCap != 0 || e.CutClosed != 0) {
			t.Errorf("windowed ore event %+v, want 100 wanted (half of 200) and no unit cut", e)
		}
	}
}
