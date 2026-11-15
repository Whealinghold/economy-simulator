package economysim_test

import (
	"bytes"
	"reflect"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// runSettlementsWorld builds a world from c with the seed, runs ticks steps and
// returns every event after WorldCreated (which carries the configuration hash and
// so differs by design) except the module's own two, and the final snapshot with its
// hash and the fields the module owns zeroed: the names of the classes, the module
// list and the four class fields of the region rows.
func runSettlementsWorld(t *testing.T, c economysim.Config, seed uint64, ticks int) ([]economysim.Event, economysim.Snapshot) {
	t.Helper()
	rec := &recorder{}
	w, err := economysim.NewWorld(c, economysim.WithSeed(seed), economysim.WithObserver(rec))
	if err != nil {
		t.Fatalf("NewWorld(seed %d) error = %v, want nil", seed, err)
	}
	for range ticks {
		if err := w.Step(); err != nil {
			t.Fatalf("Step(seed %d) error = %v, want nil", seed, err)
		}
	}
	snap, err := w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot(seed %d) error = %v, want nil", seed, err)
	}
	snap.ConfigHash, snap.Names.SettlementClasses, snap.Modules = [32]byte{}, nil, nil
	for i := range snap.Regions {
		r := &snap.Regions[i]
		r.Class, r.ClassSince, r.ClassTarget, r.ClassAtFloor = 0, 0, 0, false
	}
	var events []economysim.Event
	for _, e := range rec.events {
		switch e.(type) {
		case economysim.WorldCreated, economysim.SettlementClassChanged, economysim.SettlementHeld:
		default:
			events = append(events, e)
		}
	}
	return events, snap
}

// TestScenario_SettlementsLeaveOtherResultsUnchanged verifies that a configuration
// with the module on (three classes, a flag, a limit, a gated grower and refiner, a
// region row) builds a world, runs 40 ticks and gives the same events, other than the
// module's two, and the same final snapshot, other than the module's fields, as the
// same world with the module off, for three seeds. The review draws nothing, moves
// nothing and, until a reader reads the class (the next pull request), gates nothing.
//
// Requirement: FR-SET-001, FR-SET-015, FR-SET-016
//
// Guard: a class that leaked into a result before its readers exist (a draw, a
// ledger move, a gate) would move the golden of the reference chain; this test
// fails first. The readers' pull request replaces it with the gate's behaviour.
func TestScenario_SettlementsLeaveOtherResultsUnchanged(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	on := settlementsConfig()
	if err := on.Validate(); err != nil {
		t.Fatalf("Validate(settlementsConfig) = %v, want nil", err)
	}
	for _, seed := range []uint64{1, 2, 3} {
		wantEvents, wantSnap := runSettlementsWorld(t, referenceConfig(), seed, 40)
		gotEvents, gotSnap := runSettlementsWorld(t, on, seed, 40)
		if len(wantEvents) == 0 {
			t.Fatalf("seed %d: the off world emitted no event, want a run that means something", seed)
		}
		if !reflect.DeepEqual(gotEvents, wantEvents) {
			t.Errorf("seed %d: events with the module on differ from the module off (%d against %d events)", seed, len(gotEvents), len(wantEvents))
		}
		if !reflect.DeepEqual(gotSnap, wantSnap) {
			t.Errorf("seed %d: final snapshot with the module on differs from the module off", seed)
		}
	}
}

// TestConfigOnly_SettlementsHashDiffersFromOff verifies that the world reports the
// hash of the configuration with the module, which differs from the same world with
// the module off, so a checkpoint of one is refused by the other.
//
// Requirement: FR-SET-030
func TestConfigOnly_SettlementsHashDiffersFromOff(t *testing.T) {
	t.Parallel()
	off, err := economysim.NewWorld(referenceConfig())
	if err != nil {
		t.Fatalf("NewWorld(off) error = %v, want nil", err)
	}
	on, err := economysim.NewWorld(settlementsConfig())
	if err != nil {
		t.Fatalf("NewWorld(on) error = %v, want nil", err)
	}
	if on.ConfigHash() == off.ConfigHash() {
		t.Errorf("ConfigHash with the module on = %x, the same as off, want different", on.ConfigHash())
	}
}

// oneClassConfig is referenceConfig with the module on and one class, which is the
// first and the last: no change is possible and no flag or limit gates anything
// (spec-settlements.md 3.6).
func oneClassConfig() economysim.Config {
	c := referenceConfig()
	m := economysim.DefaultSettlements()
	m.Classes = []economysim.SettlementClass{{ID: "only"}}
	c.Modules.Settlements = &m
	return c
}

// TestScenarioSET4_OneClassEqualsOff verifies SET-S4: a world with the module on and
// one class gives, for every seed, the same events other than the module's two and the
// same snapshot other than the module's fields as the world with the module off, for 60
// ticks (FR-SET-015). The module has nothing to decide with one class, so a difference is
// the module leaking into a result.
//
// Requirement: FR-SET-015
func TestScenarioSET4_OneClassEqualsOff(t *testing.T) {
	t.Parallel()
	if testing.Short() || raceEnabled {
		t.Skip("multi-seed test: runs in gate F, not in the short or race runs (testing.md 9.1)")
	}
	for seed := uint64(1); seed <= 4; seed++ {
		wantEvents, wantSnap := runSettlementsWorld(t, referenceConfig(), seed, 60)
		gotEvents, gotSnap := runSettlementsWorld(t, oneClassConfig(), seed, 60)
		if !reflect.DeepEqual(gotEvents, wantEvents) || !reflect.DeepEqual(gotSnap, wantSnap) {
			t.Errorf("seed %d: the world with one class differs from the module off (%d and %d events)", seed, len(gotEvents), len(wantEvents))
		}
	}
}

// TestDeterminism_SettlementsNoDrawNoLedger verifies that the module draws nothing and
// moves no money or goods: after 30 ticks, every section of the checkpoint of the world
// with one class other than tag 23, which is the module's own, equals the section of the
// module off byte for byte, the ledger, the agents, the markets and the seed among them
// (FR-SET-016).
//
// Requirement: FR-SET-016
func TestDeterminism_SettlementsNoDrawNoLedger(t *testing.T) {
	t.Parallel()
	if testing.Short() || raceEnabled {
		t.Skip("multi-seed test: runs in gate F, not in the short or race runs (testing.md 9.1)")
	}
	sections := func(c economysim.Config) map[uint64][]byte {
		w := newPackWorld(t, c, 3)
		step(t, w, 30)
		out := map[uint64][]byte{}
		for _, s := range decodeFrame(t, mustMarshal(t, mustCheckpoint(t, w))).sections {
			if s.tag != 23 {
				out[s.tag] = s.body
			}
		}
		return out
	}
	want, got := sections(referenceConfig()), sections(oneClassConfig())
	if len(want) != len(got) {
		t.Fatalf("sections with one class = %d, off = %d, want the same set", len(got), len(want))
	}
	for tag, body := range want {
		if !bytes.Equal(body, got[tag]) {
			t.Errorf("section %d differs between the world with one class and the module off", tag)
		}
	}
}
