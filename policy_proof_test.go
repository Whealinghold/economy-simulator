package economysim_test

import (
	"fmt"
	"reflect"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// The proofs of the policy module on the pack of its golden (plan 3-plc Phase 8): the run does not
// depend on how it is driven or watched, and a fork at a tick N of every kind of boundary continues
// as the uninterrupted world does. The actors are council_a (ActorID 0, a schedule, due at every
// multiple of 10) and council_b (ActorID 1, a rule on the unfed share, every 30 ticks).

// TestDeterminism_Policy verifies that the final state of the module-alone pack does not depend on
// the way the run is driven or watched: the cases of detCases(true), seed 1, end in the snapshot and
// the state hash of a bare Step run, and their events equal an observed Step run's.
//
// Requirement: FR-PLC-055, NFR-PLC-004, NFR-PLC-005, FR-OBS-023, FR-WLD-063
//
// Guard: the actors' views are built whether or not an observer is attached; a decision that read
// the observer, or a state the chunking of Run changed, would fail in the case that differs.
func TestDeterminism_Policy(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := loadValidPack(t, fivePolicy...)
	base := runPack(t, cfg, detStep(runSpec{Seed: 1, Ticks: detTicks}))
	ref := runPack(t, cfg, detStep(runSpec{Seed: 1, Ticks: detTicks, Observe: true}))
	refEvents := renderEvents(withoutRunBrackets(ref))
	if d := firstTickDiff(base.Samples, ref.Samples); d != "" {
		t.Fatalf("observed Step run differs from the bare one at %s (FR-OBS-023)", d)
	}
	for _, tc := range detCases(true) {
		spec := tc.spec()
		spec.Seed, spec.Ticks = 1, detTicks
		got := runPack(t, cfg, spec)
		if !reflect.DeepEqual(got.Final, base.Final) {
			t.Errorf("case %q: final snapshot differs from the bare run", tc.name)
		}
		if economysim.HashWorld(got.World) != economysim.HashWorld(base.World) {
			t.Errorf("case %q: state hash of the world differs from the bare run", tc.name)
		}
		if spec.Observe {
			if d := firstLineDiff(refEvents, renderEvents(withoutRunBrackets(got))); d != "" {
				t.Errorf("case %q: events differ from the reference at %s", tc.name, d)
			}
		}
	}
}

// directPolicyCouncil is the call of the fork's Before hook: the directed actor of
// overlay/policy-directed.toml (ActorID 0) is told to close its gate at a factor of 20 000, to move
// the grant by 5 000 and the road to 12 000. Levers are indices in the actor's ascending lever IDs:
// gate 0, grant 1, road 2.
func directPolicyCouncil(t *testing.T) func(w *economysim.World) {
	return func(w *economysim.World) {
		t.Helper()
		err := w.DirectPolicy([]economysim.PolicyDirective{{Actor: 0, Levers: []economysim.LeverSetting{
			{Lever: 0, BP: 20_000, Closed: true}, {Lever: 1, BP: 15_000}, {Lever: 2, BP: 12_000},
		}}})
		if err != nil {
			t.Fatalf("DirectPolicy() error = %v, want nil", err)
		}
	}
}

// TestFork_Policy verifies the fork at N of CHK 8.1 on the pack of the golden (S158's boundary
// cells): the original and two restores give the same events, snapshot, audit and checkpoint bytes
// for 30 ticks after N of 0 (the construction state), 9 (before the first due tick that follows
// tick 0), 101 (just after the schedule's first applied change, at tick 100), 19 (after the rule's
// stepped cut at tick 18, seed 1 as probed) and, with the directed overlay, 97 with a directive
// pending since 95 (applied at the next due tick, 100).
//
// Requirement: FR-PLC-055, FR-PLC-073, FR-PLC-090, FR-CHK-048
//
// Guard: a restore that lost the mailbox, re-read a lever the owners' sections hold, or restarted an
// actor's cadence would differ at the first decision after the boundary.
func TestFork_Policy(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	audit := func() []economysim.Option { return []economysim.Option{economysim.WithAudit()} }
	cfg := loadValidPack(t, fivePolicy...)
	for _, n := range []int{0, 9, 19, 101} {
		t.Run(fmt.Sprintf("schedule and rule at %d", n), func(t *testing.T) {
			t.Parallel()
			forkAtN(t, forkCase{Name: fmt.Sprintf("policy at %d", n), Cfg: cfg, Seed: 1, N: n, M: forkM, Logs: true, Opts: audit})
		})
	}
	t.Run("directive pending at 97", func(t *testing.T) {
		t.Parallel()
		forkAtN(t, forkCase{
			Name: "policy directed at 97", Cfg: loadValidPack(t, fivePolicyDirected...), Seed: 1, N: 97, M: forkM, Logs: true, Opts: audit,
			Before: directPolicyCouncil(t),
			Inspect: func(t *testing.T, restored, _ []economysim.Event) {
				var applied int
				for _, e := range restored {
					if e, ok := e.(economysim.PolicyDecided); ok && e.Actor == 0 && e.Applied > 0 {
						applied++
					}
				}
				if applied != 1 {
					t.Errorf("the restored world applied the directive %d times, want once (at tick 100)", applied)
				}
			},
		})
	})
}
