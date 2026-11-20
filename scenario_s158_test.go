package economysim_test

import (
	"bytes"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// TestScenarioS158_ADirectiveIsAppliedAtTheCadence reports PLC-S4 of spec-policy.md 14.4 on the pack of the
// golden with overlay/policy-directed.toml, seeds 1 to 3: council_a is directed. A player calls
// DirectPolicy at tick 95 (the gate closed at 20 000, the stipend 15 000, the road 12 000);
// Snapshot().Actors[0].Pending is 3 (the settings, FR-PLC-074) and nothing is written until the actor's next due tick, 100,
// when the three levers are written (eight edge events, one flow event, one transit event, the cause policy)
// and Pending is 0. A checkpoint is taken at 97, the mailbox in it; a world restored from it has
// Pending 1, applies the same writes at 100 and its checkpoint at 101 equals the original's byte for byte.
//
// Requirement: FR-PLC-055, FR-PLC-070, FR-PLC-071, FR-PLC-073, FR-PLC-090
//
// Guard: a directive applied before its due tick, applied twice, or lost by a restore fails here.
func TestScenarioS158_ADirectiveIsAppliedAtTheCadence(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end scenario, skipped with -short")
	}
	cfg := loadValidPack(t, fivePolicyDirected...)
	policyScenario(t, "S158 five+policy directive at 95, checkpoint at 97, applied at 100", min(scenarioSeedCount(), 3), func(t *testing.T, rep *scenReport, seed uint64) {
		w, k := policyWorld(t, cfg, seed)
		policyStepTo(t, w, 95, nil)
		directPolicyCouncil(t)(w)
		snap, err := w.Snapshot()
		if err != nil {
			t.Fatalf("Snapshot() error = %v, want nil", err)
		}
		checkExact(t, rep, "S158", "five", seed, "the three settings are pending after the call", len(snap.Actors) > 0 && snap.Actors[0].Pending == 3, "Actors = %+v", snap.Actors)
		policyStepTo(t, w, 97, nil)
		ck, err := w.Checkpoint()
		if err != nil {
			t.Fatalf("Checkpoint() error = %v, want nil", err)
		}
		before := len(k.edges) + len(k.flows) // council_b's rule moves a transit lane of its own
		r, rk := restoreKept(t, cfg, ck)
		rs, err := r.Snapshot()
		if err != nil {
			t.Fatalf("Snapshot() of the restored world error = %v, want nil", err)
		}
		checkExact(t, rep, "S158", "five", seed, "the restored world has the settings pending", len(rs.Actors) > 0 && rs.Actors[0].Pending == 3, "Actors = %+v", rs.Actors)
		policyStepTo(t, w, 101, nil)
		policyStepTo(t, r, 101, nil)
		checkExact(t, rep, "S158", "five", seed, "nothing was written before the due tick", before == 0, "%d edge or flow events before tick 97", before)
		got, want := appliedAt(k, 100), appliedAt(rk, 100)
		checkExact(t, rep, "S158", "five", seed, "the original applies it at tick 100", got == [3]int{8, 1, 1}, "edge, flow and transit events at 100: %v, want [8 1 1]", got)
		checkExact(t, rep, "S158", "five", seed, "the restored world applies the same", got == want, "restored %v, original %v", want, got)
		checkExact(t, rep, "S158", "five", seed, "the checkpoints at 101 are equal", bytes.Equal(policyBytes(t, w), policyBytes(t, r)), "the checkpoint of the restored world differs from the original's")
		after, _ := w.Snapshot()
		checkExact(t, rep, "S158", "five", seed, "the mailbox is empty after the decision", after.Actors[0].Pending == 0, "Pending = %d", after.Actors[0].Pending)
	})
}

// restoreKept restores cfg from ck with the audit on and a kept observer.
func restoreKept(t testing.TB, cfg economysim.Config, ck economysim.Checkpoint) (*economysim.World, *policyKept) {
	t.Helper()
	k := &policyKept{others: map[string]int{}}
	w, err := economysim.RestoreWorld(cfg, ck, economysim.WithAudit(), economysim.WithObserver(k))
	if err != nil {
		t.Fatalf("RestoreWorld() error = %v, want nil", err)
	}
	return w, k
}

// appliedAt counts the edge, flow and transit events of the tick with the cause policy.
func appliedAt(k *policyKept, tick int64) [3]int {
	var n [3]int
	for _, e := range k.edges {
		if e.Tick == tick && e.Cause == economysim.CausePolicy {
			n[0]++
		}
	}
	for _, f := range k.flows {
		if f.Tick == tick && f.Cause == economysim.CausePolicy {
			n[1]++
		}
	}
	for _, f := range k.transit {
		if f.Tick == tick && f.Cause == economysim.CausePolicy {
			n[2]++
		}
	}
	return n
}

// policyBytes is the checkpoint of w, marshalled.
func policyBytes(t testing.TB, w *economysim.World) []byte {
	t.Helper()
	ck, err := w.Checkpoint()
	if err != nil {
		t.Fatalf("Checkpoint() error = %v, want nil", err)
	}
	b, err := ck.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary() error = %v, want nil", err)
	}
	return b
}
