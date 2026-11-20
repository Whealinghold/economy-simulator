package economysim

import "testing"

// The test of what the four deferred instruments `economysim.policy.*` need (spec-policy.md
// 12.4, plan 3-plc Q5): the instruments themselves are built later in otelobserver, so this
// test derives them from the recorded PolicyDecided events alone, as the adapter will.

// policySeries is what the adapter derives for one actor: a counter per instrument and
// attribute value.
type policySeries struct {
	decisions int64
	entries   map[string]int64 // by economysim.entry_outcome
	cut       map[string]int64 // by economysim.policy_cut
	held      map[string]int64 // by economysim.policy_hold
}

// derivePolicySeries applies the rows of observability-and-logging.md 5.3.1 to the events:
// no state of the world is read, and the actor attribute is the ActorID the event carries.
func derivePolicySeries(events []PolicyDecided) map[ActorID]*policySeries {
	out := map[ActorID]*policySeries{}
	for _, e := range events {
		s := out[e.Actor]
		if s == nil {
			s = &policySeries{entries: map[string]int64{}, cut: map[string]int64{}, held: map[string]int64{}}
			out[e.Actor] = s
		}
		s.decisions++
		for k, v := range map[string]int64{"applied": e.Applied, "unchanged": e.Unchanged, "not_held": e.NotHeld, "duplicate": e.Duplicate, "no_close": e.NoClose} {
			s.entries[k] += v
		}
		s.cut["clamped"] += e.Clamped
		s.cut["stepped"] += e.Stepped
		s.held["in_band"] += e.InBand
		s.held["no_reading"] += e.NoReading
	}
	return out
}

// TestPolicy_MetricsDerivableFromDecided verifies FR-PLC-086: over 30 ticks of a scripted
// actor that exercises every outcome and a rule that holds reactions, the four instruments
// are derived from PolicyDecided alone, each series is on an actor of the configuration and
// a value of its closed set, and the entry outcomes of every event sum to its Entries (so
// `economysim.policy.entries` loses no entry).
//
// Requirement: FR-PLC-082, FR-PLC-086
func TestPolicy_MetricsDerivableFromDecided(t *testing.T) {
	t.Parallel()
	shire := shireActor("rule", 1)
	shire.ReadsRegions = true
	shire.Rules = []Reaction{{Lever: "back", Gauge: GaugeMeanRatio, Target: 20_000, Span: 1_000, Move: 500, Raise: false}}
	rec := &evLog{}
	w := newMill(t, policyMillConfig(crownActor("schedule", 1), shire), WithSeed(3), WithObserver(rec))
	w.policy.actors[0].builtin = &scriptedPolicy{set: Settings{Levers: []LeverSetting{
		{Lever: 7, BP: 1}, {Lever: 1, BP: 99_999}, {Lever: 1, BP: 11_000},
		{Lever: 2, BP: 4_000, Closed: true}, {Lever: 0, BP: 1_000, Closed: true},
	}}}
	const ticks = 30
	for range ticks {
		if err := w.Step(); err != nil {
			t.Fatalf("Step() error = %v, want nil", err)
		}
	}
	events := of[PolicyDecided](rec)
	checkDecidedCounts(t, events)
	checkDerivedSeries(t, derivePolicySeries(events), len(w.policy.actors), len(events), ticks)
}

// checkDecidedCounts checks that the outcomes of every event sum to its Entries and that no
// count is negative or a cut above the entries.
func checkDecidedCounts(t *testing.T, events []PolicyDecided) {
	t.Helper()
	for _, e := range events {
		if sum := e.Applied + e.Unchanged + e.NotHeld + e.Duplicate + e.NoClose; sum != e.Entries {
			t.Errorf("PolicyDecided %+v: outcomes sum to %d, want Entries %d", e, sum, e.Entries)
		}
		if min(e.Clamped, e.Stepped, e.InBand, e.NoReading) < 0 || max(e.Clamped, e.Stepped) > e.Entries {
			t.Errorf("PolicyDecided %+v: a cut or hold count is negative, or a cut count exceeds the entries", e)
		}
	}
}

// checkDerivedSeries checks that every series is on a configured actor (an ActorID is its
// index in the world's table, because Snapshot.Actors of spec-policy.md 12.3 is not built
// yet), that the decisions add up to the events, and the crown's counts.
func checkDerivedSeries(t *testing.T, series map[ActorID]*policySeries, actors, events, ticks int) {
	t.Helper()
	var total int64
	for id, s := range series {
		if int(id) < 0 || int(id) >= actors {
			t.Errorf("series on actor %d, want one of the %d configured actors", id, actors)
		}
		total += s.decisions
	}
	checkCrownSeries(t, series[0], ticks)
	if int(total) != events {
		t.Errorf("decisions summed to %d, want one per PolicyDecided (%d)", total, events)
	}
}

// checkCrownSeries checks the series of the scripted actor: a decision in every tick, and in
// each of them one entry for an unknown lever, a repeated lever and a closing the lever may
// not make, and the bounds cutting at least one entry.
func checkCrownSeries(t *testing.T, crown *policySeries, ticks int) {
	t.Helper()
	n := int64(ticks)
	if crown == nil || crown.decisions != n || crown.entries["not_held"] != n || crown.entries["duplicate"] != n ||
		crown.entries["no_close"] != n || crown.cut["clamped"] < n {
		t.Errorf("crown series = %+v, want %d decisions, and one not_held, duplicate and no_close entry in each", crown, ticks)
	}
}
