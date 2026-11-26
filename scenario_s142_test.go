package economysim_test

import (
	"reflect"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// savingBareLayer switches the saving module on with its defaults and gives no role a block.
const savingBareLayer = `
[modules.saving]
`

// TestScenarioS142_InertEqualsOff reports SAV-S1 of spec-saving-and-investment.md 14.3, a harness
// test: pack one with the saving module on and no Saver block on any role is the world with the
// module off, bit for bit, over seeds 1 to 3 and 100 ticks. The snapshot after every 10th tick (but
// its ConfigHash and its list of modules, which differ by design), every event of the run and the
// SavingPlanned count (none) are compared, and no role has a snapshot row of the module.
//
// Arithmetic: a role with no block has no Saving group in its view, so its spend cap reads the flat
// reserve, no project is weighed, no request is made and no event is built (spec 3.2.2, FR-SAV-001,
// FR-SAV-002).
//
// Requirement: FR-SAV-001, FR-SAV-002
//
// Guard: a read of the module's tables for a role with no block, a draw or a count that the module
// made on its own would move every run of a consumer that switched the module on for another role.
func TestScenarioS142_InertEqualsOff(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end scenario, skipped with -short")
	}
	off := loadValidPack(t, packOne...)
	on := loadPackBytes(t, []byte(savingBareLayer), packOne...)
	if on.Modules.Saving == nil || off.Modules.Saving != nil {
		t.Fatalf("Modules.Saving on = %v, off = %v, want set and nil", on.Modules.Saving, off.Modules.Saving)
	}
	const ticks = 100
	savingScenario(t, "S142 one+saving without a block equals off, 100 ticks", min(3, scenarioSeedCount()), func(t *testing.T, rep *scenReport, seed uint64) {
		recOn, recOff := &recorder{}, &recorder{}
		a := mustWorld(t, on, economysim.WithSeed(seed), economysim.WithAudit(), economysim.WithObserver(recOn))
		b := mustWorld(t, off, economysim.WithSeed(seed), economysim.WithAudit(), economysim.WithObserver(recOff))
		recOn.forget() // WorldCreated carries the configuration hash, which differs by design
		recOff.forget()
		equal := true
		for tick := 1; tick <= ticks && equal; tick++ {
			capStep(t, a)
			capStep(t, b)
			if tick%10 != 0 {
				continue
			}
			sa, errA := a.Snapshot()
			sb, errB := b.Snapshot()
			if errA != nil || errB != nil {
				t.Fatalf("seed %d tick %d: Snapshot() = %v, %v, want nil", seed, tick, errA, errB)
			}
			sa.ConfigHash, sb.ConfigHash, sa.Modules, sb.Modules = [32]byte{}, [32]byte{}, nil, nil
			equal = reflect.DeepEqual(sa, sb)
		}
		checkExact(t, rep, "S142", "one", seed, "the snapshots are equal", equal, "the module on with no block changed the snapshot")
		checkExact(t, rep, "S142", "one", seed, "the events are equal", reflect.DeepEqual(recOn.events, recOff.events),
			"%d events with the module on, %d off", len(recOn.events), len(recOff.events))
		var planned int
		for _, e := range recOn.events {
			if _, ok := e.(economysim.SavingPlanned); ok {
				planned++
			}
		}
		checkExact(t, rep, "S142", "one", seed, "no SavingPlanned with no block", planned == 0, "%d SavingPlanned events, want 0", planned)
		rep.num(seed, "events", int64(len(recOn.events)))
		// Not in the report: the equalities above hold for a module that does nothing at all. The control is
		// the same pack with the blocks of the overlay, which must plan, save and build (savingRun fails a
		// run that planned or saved nothing), so a module that has gone inert fails here and not only in S143.
		if st := savingRun(t, savingPack(t, nil), seed, ticks); st.Builds == 0 {
			t.Fatalf("seed %d: the control with blocks made %d build requests in %d ticks, want above 0", seed, st.Builds, ticks)
		}
	})
}
