package economysim_test

import (
	"fmt"
	"reflect"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// roomBinds is what a run of the storage pack shows of a room that binds.
type roomBinds struct {
	firstRejected, firstCut int64 // the tick of the first no_room bid and of the first role change that listed goods; -1 for none
	rejected                int   // OrderRejected events with the reason no_room
}

// bindsOf reads the events of a run of the storage pack for the two signs of a room
// that binds: a bid rejected for room and a role change that moved goods to the estate.
func bindsOf(events []economysim.Event) roomBinds {
	b := roomBinds{firstRejected: -1, firstCut: -1}
	for _, e := range events {
		switch e := e.(type) {
		case economysim.OrderRejected:
			if e.Reason != economysim.ReasonNoRoom {
				continue
			}
			b.rejected++
			if b.firstRejected < 0 {
				b.firstRejected = e.Tick
			}
		case economysim.RoleChanged:
			if e.GoodsListed > 0 && b.firstCut < 0 {
				b.firstCut = e.Tick
			}
		}
	}
	return b
}

// TestStorage_RoomBindsOnTheGoldenPack verifies, without reading the golden file,
// that the provisional golden of the module shows a room that binds: in 400 ticks of
// pack one with the module on, at least one bid is rejected for room and at least one
// role change moves goods above the new room to the estate, and the pack with the
// module off has neither. A golden whose content never showed a refusal would pass
// its own byte comparison and prove nothing about the module.
//
// Requirement: FR-STO-008, FR-STO-018, FR-STO-001
//
// Guard: the golden is recorded once and compared as bytes; if a change loosened the
// room until it never bound, re-recording it would hide that the module does nothing.
func TestStorage_RoomBindsOnTheGoldenPack(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	on := runPack(t, loadValidPack(t, packOneStorage...), runSpec{Seed: goldenSeed, Ticks: goldenTicks, Drive: driveStep, Observe: true, Audit: true})
	got := bindsOf(on.Events)
	if got.rejected < 1 {
		t.Errorf("bids rejected no_room in %d ticks, seed %d = %d, want at least 1", goldenTicks, goldenSeed, got.rejected)
	}
	if got.firstCut < 0 {
		t.Errorf("role changes that listed goods to the estate in %d ticks, seed %d: none, want at least 1", goldenTicks, goldenSeed)
	}
	off := runPack(t, loadValidPack(t, packOne...), runSpec{Seed: goldenSeed, Ticks: goldenTicks, Drive: driveStep, Observe: true})
	if o := bindsOf(off.Events); o.rejected != 0 {
		t.Errorf("bids rejected no_room with the module off = %d, want 0", o.rejected)
	}
}

// TestStorage_LoadNeverAboveRoomAtAnyTick verifies the invariant of FR-STO-005 on the
// snapshot after every one of 400 ticks, for several seeds, rather than every tenth
// as TestInvariant_RoomNeverExceeded does: on every role row the load is at most the
// room in both dimensions, and a role with limited households has a room above 0.
// The audit is on, so the pledge and the reserved room are checked by the world at
// every position as well.
//
// Requirement: FR-STO-005, FR-STO-010, FR-STO-040
//
// Guard: a transient overshoot between two sampled ticks (a credit that took room
// from a commitment and was repaired a tick later) is invisible to a sparse sample.
func TestStorage_LoadNeverAboveRoomAtAnyTick(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := loadValidPack(t, packOneStorage...)
	for _, seed := range seedList(scenarioSeedCount()) {
		res := runPack(t, cfg, runSpec{Seed: seed, Ticks: goldenTicks, Drive: driveStep, Audit: true, Snapshots: snapEvery})
		for tick, snap := range res.Samples {
			for _, r := range snap.Roles {
				if r.LoadMass > r.RoomMass || r.LoadBulk > r.RoomBulk {
					t.Errorf("seed %d tick %d: role row %+v has a load above its room", seed, tick, r)
				}
				if r.LimitedHouseholds > 0 && (r.RoomMass <= 0 && r.RoomBulk <= 0) {
					t.Errorf("seed %d tick %d: role row %+v has limited households and no room", seed, tick, r)
				}
			}
		}
	}
}

// TestDeterminism_Storage verifies that a run of the storage pack is the same run
// however it is observed: for each seed, a run with a logger, an observer or the
// audit, and a run split into Run calls (seeds 1 and 2, the reduced case list), end in the snapshot and the state hash of the
// bare Step run, the events of the observed runs are equal, and a different seed gives
// a different run.
//
// Requirement: NFR-STO-004, FR-STO-037, FR-STO-041, FR-OBS-023
//
// Guard: the room is state read by decisions (bids, starts, dispatches). A logger or an
// observer that changed what the module counted or when it released a reservation would
// change the run, and only a run of the module on can show it.
func TestDeterminism_Storage(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := loadValidPack(t, packOneStorage...)
	cases := detCases(true) // the reduced list: seeds 1 and 2 are the cost, as for the money supply
	seeds := seedList(2)
	finals := make([]economysim.Snapshot, len(seeds))
	for i, seed := range seeds {
		t.Run(fmt.Sprintf("seed %d", seed), func(t *testing.T) {
			base := runPack(t, cfg, detStep(runSpec{Seed: seed, Ticks: detTicks}))
			finals[i] = base.Final
			ref := runPack(t, cfg, detStep(runSpec{Seed: seed, Ticks: detTicks, Observe: true}))
			refEvents := renderEvents(withoutRunBrackets(ref))
			if d := firstTickDiff(base.Samples, ref.Samples); d != "" {
				t.Fatalf("seed %d: observed Step run differs from the bare one at %s", seed, d)
			}
			for _, tc := range cases {
				spec := tc.spec()
				spec.Seed, spec.Ticks = seed, detTicks
				got := runPack(t, cfg, spec)
				if !reflect.DeepEqual(got.Final, base.Final) {
					t.Errorf("seed %d, case %q: final snapshot differs from the bare run", seed, tc.name)
				}
				if economysim.HashWorld(got.World) != economysim.HashWorld(base.World) {
					t.Errorf("seed %d, case %q: state hash of the world differs from the bare run", seed, tc.name)
				}
				if spec.Observe {
					if d := firstLineDiff(refEvents, renderEvents(withoutRunBrackets(got))); d != "" {
						t.Errorf("seed %d, case %q: events differ from the reference at %s", seed, tc.name, d)
					}
				}
			}
		})
	}
	if len(seeds) > 1 && reflect.DeepEqual(finals[0], finals[1]) {
		t.Errorf("seeds %d and %d end in the same snapshot, want different runs", seeds[0], seeds[1])
	}
}

// storageForkCells are the ticks at which the storage pack is checkpointed: before any
// job, inside the grow jobs whose output is pledged, and after the first room that
// binds (tick 118 of seed 1, see the golden), so the restore meets pledges, a room
// that has cut goods and a run that has refused bids.
var storageForkCells = []struct {
	name string
	n    int
}{
	{"before any job", 1},
	{"inside a job", 100},
	{"after the room bound", 130},
	{"across a season change", 150},
}

// TestFork_Storage verifies that a world of the storage pack restored from its
// checkpoint continues exactly as the world that never stopped: same events,
// snapshot, audit and checkpoint bytes for 30 ticks, and the restore of a restore
// too. The module has no checkpoint section yet (the stores are Phase 8) and a run of
// the pack owns no store, so what a restore must rebuild is the derived state: the
// room of each limited account from its weight, the load from the lots, and the pledge
// from the jobs in progress; the audit of the restored world names any drift.
//
// Requirement: FR-STO-043, FR-STO-020, FR-STO-039, FR-CHK-050
//
// Guard: the pledge is not saved (FR-STO-020). A restore that left it at 0 would pass
// until the first completion credited from a pledge that was never made, and fail only
// in a run that checkpoints inside a job.
func TestFork_Storage(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	t.Skip("plan 3-sto Phase 8 (PR 5), the checkpoint of the stores: until then RestoreWorld of a storage world returns ErrCheckpointCorrupt (TestRestoreWorld_StorageWorldIsRefusedNotPanicked)")
	cfg := loadValidPack(t, packOneStorage...)
	for _, cell := range storageForkCells {
		t.Run(cell.name, func(t *testing.T) {
			t.Parallel()
			forkAtN(t, forkCase{
				Name: "storage", Cfg: cfg, Seed: 1, N: cell.n, M: forkM,
				Opts: func() []economysim.Option { return []economysim.Option{economysim.WithAudit()} },
			})
		})
	}
}
