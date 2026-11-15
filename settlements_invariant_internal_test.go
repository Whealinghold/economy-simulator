package economysim

import (
	"math"
	"strconv"
	"strings"
	"testing"
)

// The tests of the guards of the settlements module and of the first release of the
// dwell, over the world of setConfig (settlements_review_internal_test.go). The
// guards are violated programmer invariants: validation rules the states out, so
// only a Config that skipped validation reaches them.

// TestNewSettlements_ViolatedInvariantsPanic verifies that newSettlements refuses, with
// an invariant message and not a wrong table, a configuration that validation would have
// rejected: a region row that names no region, a scale whose threshold overflows, and
// more distinct limit names than a table holds. A valid configuration builds the state.
//
// Requirement: FR-SET-002, FR-SET-005
//
// Guard: validation is the only thing between a bad configuration and a table indexed
// at -1 (the row), a negative threshold (the scale) or a limit written in the next
// class's cell (the table); without the panic each would give results, not a failure.
func TestNewSettlements_ViolatedInvariantsPanic(t *testing.T) {
	t.Parallel()
	manyLimits := make([]ClassLimit, 65) // one more than the 64 a table holds (internal/settlement.MaxBits)
	for i := range manyLimits {
		manyLimits[i] = ClassLimit{Name: "limit" + strconv.Itoa(i), Value: 1}
	}
	for _, tt := range []struct {
		name   string
		mutate func(*Settlements)
		want   string // "" builds the state
	}{
		{"valid", func(*Settlements) {}, ""},
		{"row names no region", func(m *Settlements) {
			m.Regions = append([]RegionSettlement{{Region: "nowhere", ThresholdBP: 10_000}}, m.Regions...)
		}, "names a region that does not exist"},
		{"threshold overflows", func(m *Settlements) {
			// The product is divided by 10 000 in 128 bits, so a scale alone fits; a scale and an
			// enter that are both near MaxInt64 give 8.5e33, which does not fit an int64.
			m.Classes = append([]SettlementClass(nil), m.Classes...)
			m.Classes[1].Enter = math.MaxInt64
			m.Regions = []RegionSettlement{{Region: "a", ThresholdBP: math.MaxInt64}}
		}, "effective thresholds"},
		{"too many limit names", func(m *Settlements) {
			m.Classes = append([]SettlementClass(nil), m.Classes...)
			m.Classes[1].Limits = manyLimits
		}, "flag and limit tables"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := setConfig()
			m := *c.Modules.Settlements
			tt.mutate(&m)
			c.Modules.Settlements = &m
			var s *settlements
			got := invariantMessage(func() { s = newSettlements(&c) })
			if tt.want == "" {
				if got != "" || s == nil {
					t.Errorf("newSettlements(valid) = %v, panic %q, want a state and no panic", s, got)
				}
				return
			}
			if !strings.Contains(got, "violated programmer invariant") || !strings.Contains(got, tt.want) {
				t.Errorf("newSettlements(%s) panic = %q, want an invariant message containing %q", tt.name, got, tt.want)
			}
		})
	}
}

// TestReview_TicksSinceOverflowIsAnInvariant verifies that a review whose elapsed
// ticks do not fit in an int64 panics with an invariant message instead of deciding on
// a wrapped number, and that the same review with a Since that fits changes the class.
//
// Arithmetic: a Since of the smallest int64 and a tick of 10 give 10 - MinInt64, which
// is above MaxInt64.
//
// Requirement: FR-SET-012
//
// Guard: a restore bounds Since by the tick (FR-SET-029), so the state cannot arise
// from a checkpoint; a wrapped elapsed count would make a held change look due.
func TestReview_TicksSinceOverflowIsAnInvariant(t *testing.T) {
	t.Parallel()
	w, _ := setWorld(t)
	setHouseholds(t, w, setA, 13) // a hamlet at 13 households is due to be a village
	w.set.hh[setA] = 13
	w.set.since[setA] = math.MinInt64
	if got := invariantMessage(func() { w.set.review(w, setA, 10) }); !strings.Contains(got, "overflow") {
		t.Errorf("review(since MinInt64, tick 10) panic = %q, want an invariant message naming the overflow", got)
	}
	w.set.since[setA] = 0
	w.set.review(w, setA, 10)
	if w.set.class[setA] != 1 || w.set.since[setA] != 10 {
		t.Errorf("review(since 0, tick 10) class, since = %d, %d, want 1, 10", w.set.class[setA], w.set.since[setA])
	}
}

// TestRegionReads_ModuleOffHasNoFlagAndNoLimit verifies that with the module off
// regionHas is false for any flag and regionLimit reports no limit (0, false), so a
// reader needs no check of its own that the module is on.
//
// Requirement: FR-SET-020, FR-SET-001
func TestRegionReads_ModuleOffHasNoFlagAndNoLimit(t *testing.T) {
	t.Parallel()
	w := newMill(t, shellConfig())
	if w.set != nil {
		t.Fatalf("world with the module off has set = %v, want nil", w.set)
	}
	if w.regionHas(0, 0) {
		t.Errorf("regionHas(0, 0) = true with the module off, want false")
	}
	if v, ok := w.regionLimit(0, 0); v != 0 || ok {
		t.Errorf("regionLimit(0, 0) = %d, %v with the module off, want 0, false", v, ok)
	}
}

// TestDwell_FirstReleaseAtTickDwell verifies that regions founded together are held
// together and released together: with Since 0 for every region and a Dwell of 5, two
// regions whose counts cross a threshold at tick 0 are held at ticks 0 to 4, with the
// ticks left counting down from 5 to 1 in both, and both change at tick 5 in ascending
// RegionID, each with Since 5 and the cause grew.
//
// Arithmetic: a goes from 2 to 13 households, over the enter of village (10); b goes
// from 10 to 30, over the enter of town (25). Elapsed ticks at tick 4 are 4, below 5;
// at tick 5 they are 5, which meets the dwell.
//
// Requirement: FR-SET-011, FR-SET-012
func TestDwell_FirstReleaseAtTickDwell(t *testing.T) {
	t.Parallel()
	w, rec := setWorld(t)
	rec.events = nil
	setHouseholds(t, w, setA, 13)
	setHouseholds(t, w, setB, 30)
	for tick := range int64(5) {
		atTick(w, tick)
		w.p72aSettlement()
	}
	held := of[SettlementHeld](rec)
	if len(held) != 10 {
		t.Fatalf("held events before the release = %d, want 10 (2 regions, ticks 0 to 4)", len(held))
	}
	for i, h := range held {
		if wantLeft := int64(5) - h.Tick; h.TicksLeft != wantLeft || h.Region != RegionID(i%2) {
			t.Errorf("held[%d] = %+v, want region %d with %d ticks left", i, h, i%2, wantLeft)
		}
	}
	if got := of[SettlementClassChanged](rec); len(got) != 0 {
		t.Fatalf("changes during the hold = %+v, want none", got)
	}
	atTick(w, 5)
	w.p72aSettlement()
	want := []SettlementClassChanged{
		{Tick: 5, Region: setA, From: 0, To: 1, Households: 13, Cause: CauseGrew},
		{Tick: 5, Region: setB, From: 1, To: 2, Households: 30, Cause: CauseGrew},
	}
	got := of[SettlementClassChanged](rec)
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("changes at tick 5 = %+v, want %+v", got, want)
	}
	if w.set.since[setA] != 5 || w.set.since[setB] != 5 || len(of[SettlementHeld](rec)) != 10 {
		t.Errorf("since = %d, %d and %d held events after the release, want 5, 5 and 10", w.set.since[setA], w.set.since[setB], len(of[SettlementHeld](rec)))
	}
}
