package economysim_test

import (
	"log/slog"
	"reflect"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/internal/logkeys"
)

// storageLogScenario is pack one with the storage overlay for 400 ticks: a role change
// at about tick 118 cuts a room, and from then the default strategy cuts bids and
// discards. The log tests read it with the other scenarios (logScenarios).
func storageLogScenario() scenario {
	return scenario{name: "storage", ticks: 400, tail: 2, cfg: func(t *testing.T) economysim.Config {
		return loadValidPack(t, packOneStorage...)
	}}
}

// TestNonInterference_Storage verifies FR-STO-041 and NFR-STO-004: a run of the pack
// with the storage module on, with an observer and a logger that reports every level,
// ends in the snapshot a run with neither ends in.
//
// Requirement: FR-STO-041, NFR-STO-004
func TestNonInterference_Storage(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end: 400 ticks of pack one twice; skipped under -short")
	}
	sc := storageLogScenario()
	plain := play(t, sc, who{})
	seen := play(t, sc, who{observer: true, logger: true})
	if a, b := mustSnapshot(t, plain.w), mustSnapshot(t, seen.w); !reflect.DeepEqual(a, b) {
		t.Errorf("a run of the storage pack with an observer and a Debug logger ends in a snapshot that differs from one with neither (seed %d)", obsSeed)
	}
}

// TestDebug_Lines verifies the Debug lines of spec-storage.md 14.3: "bid cut for room"
// with the tick, agent, commodity and quantity, and "store added" and "store removed"
// with the tick, agent, store class and cause, each a constant message with the keys
// of the table, written at Debug and with no observer needed.
//
// Requirement: FR-STO-041
func TestDebug_Lines(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end: 400 ticks of pack one; skipped under -short")
	}
	sc := storageLogScenario()
	w, _, h := build(t, sc, who{logger: true})
	for range 300 {
		if err := w.Step(); err != nil {
			t.Fatalf("Step() = %v, want nil", err)
		}
	}
	id, ok := firstOwnerCandidate(w)
	if !ok {
		t.Fatalf("no household accepted a store of class 0")
	}
	if err := w.RemoveStore(id, 0); err != nil {
		t.Fatalf("RemoveStore(%d) = %v, want nil", id, err)
	}
	want := map[string][]string{
		"bid cut for room": {logkeys.Tick, logkeys.Agent, logkeys.Commodity, logkeys.Quantity},
		"store added":      {logkeys.Tick, logkeys.Agent, logkeys.StoreClass, logkeys.Cause},
		"store removed":    {logkeys.Tick, logkeys.Agent, logkeys.StoreClass, logkeys.Cause},
	}
	for msg, keys := range want {
		if !checkDebugLine(t, h.records(), msg, keys) {
			t.Errorf("no Debug line %q after 300 ticks, a store added and a store removed", msg)
		}
	}
}

// checkDebugLine checks every line of the message: Debug, with exactly the keys. It
// reports whether there was one.
func checkDebugLine(t *testing.T, lines []logLine, msg string, keys []string) bool {
	t.Helper()
	found := false
	for _, l := range lines {
		if l.msg != msg {
			continue
		}
		found = true
		if l.level != slog.LevelDebug || !slices.Equal(keysOfLine(l), keys) {
			t.Errorf("line %q at %v has the keys %v, want Debug with %v", l.msg, l.level, keysOfLine(l), keys)
		}
	}
	return found
}

// firstOwnerCandidate adds a store of class 0 to the first household that accepts one
// and returns it.
func firstOwnerCandidate(w *economysim.World) (economysim.AgentID, bool) {
	for id := economysim.AgentID(0); id < 200; id++ {
		if w.AddStore(id, 0) == nil {
			return id, true
		}
	}
	return 0, false
}

// TestLogKeys_StoreClass verifies that the two keys of the module are rows of the table
// 3.3.2 of the rules and that their constants are those strings.
//
// Requirement: FR-STO-041
func TestLogKeys_StoreClass(t *testing.T) {
	t.Parallel()
	keys := ruleKeys(t)
	for _, k := range []string{logkeys.StoreClass, logkeys.StoreClasses} {
		if !keys[k] {
			t.Errorf("log key %q is not a row of the table 3.3.2 of observability-and-logging.md", k)
		}
	}
	if logkeys.StoreClass != "store_class" || logkeys.StoreClasses != "store_classes" {
		t.Errorf("keys = %q and %q, want store_class and store_classes", logkeys.StoreClass, logkeys.StoreClasses)
	}
}
