package economysim_test

import (
	"log/slog"
	"reflect"
	"strings"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// treasuryRows is the Jurisdictions rows of example A in polityWorldConfig: six
// regions (estates are agents 0 to 5), so crown (JurisdictionID 1, seat capital 0)
// has the treasury agent 6 and inland (2, seat mill 4) agent 7.
func treasuryRows() []economysim.JurisdictionRow {
	return []economysim.JurisdictionRow{
		{Jurisdiction: jCrown, Seat: rCapital, Agent: 6, Money: 50_000},
		{Jurisdiction: jInland, Seat: rMill, Agent: 7, Money: 5_000},
	}
}

// TestPolity_Snapshot verifies the polity rows of a snapshot (spec-polity.md 10.5):
// MoneyTreasury is the money of the treasuries seated in each region, Jurisdictions
// has one row for each governed jurisdiction in ascending JurisdictionID, with the
// seat, the pseudo-agent and the money, the agent counts hold no treasury, the money
// of the world counts the treasuries, a second snapshot is a deep copy, and Region
// agrees with the full snapshot (FR-OBS-044).
//
// Arithmetic: crown holds 50 000 in capital and inland 5 000 in mill; no other
// transfer touches them, so the two seats' MoneyTreasury are 50 000 and 5 000 and the
// other four regions 0.
//
// Requirement: FR-POL-016, FR-POL-028, FR-OBS-041, FR-OBS-044
func TestPolity_Snapshot(t *testing.T) {
	t.Parallel()
	w := polityWorld(t, economysim.WithAudit())
	for range 5 {
		if err := w.Step(); err != nil {
			t.Fatalf("Step() error = %v, want nil", err)
		}
	}
	s, err := w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot() error = %v, want nil", err)
	}
	if !reflect.DeepEqual(s.Jurisdictions, treasuryRows()) {
		t.Errorf("Snapshot().Jurisdictions = %+v, want %+v", s.Jurisdictions, treasuryRows())
	}
	want := map[int]economysim.Money{rCapital: 50_000, rMill: 5_000}
	var households, pool economysim.Money
	for _, r := range s.Regions {
		if r.MoneyTreasury != want[int(r.Region)] {
			t.Errorf("Regions[%d].MoneyTreasury = %d, want %d", r.Region, r.MoneyTreasury, want[int(r.Region)])
		}
		households += r.MoneyHouseholds
		pool += r.MoneyPool
	}
	if s.Money != households+pool+55_000 {
		t.Errorf("Snapshot().Money = %d, want households %d + pool %d + treasuries 55 000", s.Money, households, pool)
	}
	again, _ := w.Snapshot()
	s.Jurisdictions[0].Money = 1
	if again.Jurisdictions[0].Money != 50_000 {
		t.Errorf("a second Snapshot shares the Jurisdictions rows of the first")
	}
	rs, err := w.Region(rMill)
	if err != nil || rs.Region.MoneyTreasury != 5_000 {
		t.Errorf("Region(mill) = %+v, %v, want MoneyTreasury 5 000", rs.Region, err)
	}
}

// TestPolity_WorldCreatedRebuild verifies FR-POL-035 and FR-OBS-041 at construction:
// an observer that keeps no configuration rebuilds the pool of each region from the
// construction bucket's initial money, less the StartMoney of each initial
// AgentSpawned, less the Money of each WorldCreated.Jurisdictions row seated there
// (spec-observability.md 4.3.3 item 4), and the result is the snapshot's MoneyPool.
// The rows of WorldRestored's snapshot are those of the original after a restore.
//
// Arithmetic: initial money of the seat's region = households' start money + the
// pool's seed + the treasury's StartMoney; the test subtracts the first and the last.
//
// Requirement: FR-POL-035, FR-OBS-041
func TestPolity_WorldCreatedRebuild(t *testing.T) {
	t.Parallel()
	rec := &recorder{}
	w := polityWorld(t, economysim.WithObserver(rec))
	var created economysim.WorldCreated
	spawned := make([]economysim.Money, 6) // the six regions of polityWorldConfig
	for _, e := range rec.events {
		switch e := e.(type) {
		case economysim.WorldCreated:
			created = e
		case economysim.AgentSpawned:
			spawned[e.Region] += e.StartMoney
		}
	}
	if !reflect.DeepEqual(created.Jurisdictions, treasuryRows()) {
		t.Fatalf("WorldCreated.Jurisdictions = %+v, want %+v", created.Jurisdictions, treasuryRows())
	}
	s, err := w.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	initial := make([]economysim.Money, len(s.Regions))
	for _, f := range s.Construction {
		if f.Reason == economysim.ReasonInitial && f.Commodity == economysim.NoCommodity {
			initial[f.Region] += economysim.Money(f.Total)
		}
	}
	for _, j := range created.Jurisdictions {
		initial[j.Seat] -= j.Money
	}
	for g, r := range s.Regions {
		if pool := initial[g] - spawned[g]; pool != r.MoneyPool {
			t.Errorf("region %d: rebuilt pool = %d, snapshot MoneyPool = %d", g, pool, r.MoneyPool)
		}
	}
	ck, err := w.Checkpoint()
	if err != nil {
		t.Fatal(err)
	}
	rr := &recorder{}
	if _, err := economysim.RestoreWorld(polityWorldConfig(), ck, economysim.WithObserver(rr)); err != nil {
		t.Fatalf("RestoreWorld() error = %v, want nil", err)
	}
	wr, ok := rr.events[0].(economysim.WorldRestored)
	if !ok || !reflect.DeepEqual(wr.Snapshot.Jurisdictions, treasuryRows()) {
		t.Errorf("first event of the restore = %+v, want WorldRestored with the rows %+v", rr.events[0], treasuryRows())
	}
}

// TestPolity_Cardinality verifies that every JurisdictionID of a row is in range of
// Names.Jurisdictions, every seat in range of Names.Regions, and every treasury agent
// ID between the estates and the first household, in the snapshot and the
// world-created event (spec-polity.md 10.3).
//
// Requirement: FR-POL-027, FR-POL-028
func TestPolity_Cardinality(t *testing.T) {
	t.Parallel()
	rec := &recorder{}
	w := polityWorld(t, economysim.WithObserver(rec))
	s, err := w.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	var created economysim.WorldCreated
	for _, e := range rec.events {
		if wc, ok := e.(economysim.WorldCreated); ok {
			created = wc
		}
	}
	names := s.Names
	for _, rows := range [][]economysim.JurisdictionRow{s.Jurisdictions, created.Jurisdictions} {
		for _, r := range rows {
			if r.Jurisdiction < 0 || int(r.Jurisdiction) >= len(names.Jurisdictions) {
				t.Errorf("row %+v: Jurisdiction not in 0 to %d", r, len(names.Jurisdictions))
			}
			if r.Seat < 0 || int(r.Seat) >= len(names.Regions) {
				t.Errorf("row %+v: Seat not in 0 to %d", r, len(names.Regions))
			}
			if int(r.Agent) < len(names.Regions) || int(r.Agent) >= len(names.Regions)+len(rows) {
				t.Errorf("row %+v: Agent not in the %d pseudo-agent IDs after the %d estates", r, len(rows), len(names.Regions))
			}
		}
	}
}

// TestPolity_Summary verifies the construction summary line (spec-polity.md 10.2,
// FR-POL-025): the one Info line "world created" carries jurisdictions and governed
// and polity in modules, and no other line mentions the module.
//
// Requirement: FR-POL-025
func TestPolity_Summary(t *testing.T) {
	t.Parallel()
	h := newCaptureHandler()
	polityWorld(t, economysim.WithLogger(slog.New(h)))
	found := false
	for _, l := range h.sink.lines {
		if l.msg != "world created" {
			continue
		}
		found = true
		attrs := map[string]string{}
		for _, a := range l.attrs {
			attrs[a.Key] = a.Value.String()
		}
		if attrs["jurisdictions"] != "4" || attrs["governed"] != "2" || !strings.Contains(attrs["modules"], "polity") {
			t.Errorf("world created attrs = %v, want jurisdictions 4, governed 2 and polity in modules", attrs)
		}
	}
	if !found {
		t.Error("no \"world created\" line")
	}
}

// TestPolity_NoEventsOfItsOwn verifies that the module emits nothing of its own
// (spec-polity.md 10.1, FR-POL-025, 026, NFR-POL-001): over construction and ticks
// the event types are those of the core, and the records at Info or above are those of
// a world without the module (the one summary line gains two attributes and no
// record is added).
//
// Requirement: FR-POL-025, FR-POL-026, NFR-POL-001
func TestPolity_NoEventsOfItsOwn(t *testing.T) {
	t.Parallel()
	rec, h := &recorder{}, newCaptureHandler()
	w := polityWorld(t, economysim.WithObserver(rec), economysim.WithLogger(slog.New(h)))
	for range 5 {
		if err := w.Step(); err != nil {
			t.Fatal(err)
		}
	}
	for _, e := range rec.events {
		if name := reflect.TypeOf(e).Name(); strings.Contains(name, "Jurisdiction") || strings.Contains(name, "Polity") || strings.Contains(name, "Treasury") {
			t.Errorf("event %s: the module has no event of its own", name)
		}
	}
	base := newCaptureHandler()
	off := polityWorldConfig()
	off.Modules.Polity = nil
	ow, err := economysim.NewWorld(off, economysim.WithLogger(slog.New(base)))
	if err != nil {
		t.Fatal(err)
	}
	for range 5 {
		if err := ow.Step(); err != nil {
			t.Fatal(err)
		}
	}
	if got, want := loudLines(h), loudLines(base); !reflect.DeepEqual(got, want) {
		t.Errorf("records at Info or above = %v, want those of the world without the module, %v", got, want)
	}
}

// loudLines counts the records at Info or above by level and message.
func loudLines(h *captureHandler) map[string]int {
	out := map[string]int{}
	for _, l := range h.sink.lines {
		if l.level >= slog.LevelInfo {
			out[l.level.String()+" "+l.msg]++
		}
	}
	return out
}

// TestCheckpoint_Polity verifies FR-POL-030 and FR-CHK-048 end to end: a world with
// two treasuries checkpoints and restores, the restore attaches both, its snapshot,
// audit and checkpoint bytes equal the original's, and a restore with a different
// StartMoney is refused by the configuration hash.
//
// Requirement: FR-POL-030, FR-CHK-048
func TestCheckpoint_Polity(t *testing.T) {
	t.Parallel()
	w := polityWorld(t, economysim.WithAudit())
	for range 7 {
		if err := w.Step(); err != nil {
			t.Fatal(err)
		}
	}
	ck, err := w.Checkpoint()
	if err != nil {
		t.Fatal(err)
	}
	r, err := economysim.RestoreWorld(polityWorldConfig(), ck, economysim.WithAudit())
	if err != nil {
		t.Fatalf("RestoreWorld() error = %v, want nil", err)
	}
	a, _ := w.Snapshot()
	b, _ := r.Snapshot()
	if !reflect.DeepEqual(a, b) {
		t.Error("the restored world's snapshot differs from the original's")
	}
	if got, ok := r.TreasuryAgent(jInland); !ok || got != 7 {
		t.Errorf("restored TreasuryAgent(inland) = %d, %v, want 7, true", got, ok)
	}
	other := polityWorldConfig()
	other.Modules.Polity.Jurisdictions[0].StartMoney++
	if _, err := economysim.RestoreWorld(other, ck); err == nil {
		t.Error("RestoreWorld with another StartMoney error = nil, want a refusal")
	}
}
