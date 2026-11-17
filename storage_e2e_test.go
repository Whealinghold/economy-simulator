package economysim_test

import (
	"fmt"
	"reflect"
	"slices"
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
// pack one with the module on, at least one bid is cut or rejected for room and at least one
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
	// The default strategy sizes its bids to the free room (FR-STO-023), so the room
	// shows as RoomLimited cuts and discards rather than as rejected bids.
	cuts := 0
	for _, e := range on.Events {
		if rl, ok := e.(economysim.RoomLimited); ok {
			cuts += int(rl.BidsCut + rl.Discarded + rl.BidsRejected)
		}
	}
	if got.rejected+cuts < 1 {
		t.Errorf("bids rejected or cut for room in %d ticks, seed %d = %d, want at least 1", goldenTicks, goldenSeed, got.rejected+cuts)
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
	name  string
	n     int
	store bool // a store is added to a household just before the boundary
}{
	{"before any job", 1, false},
	{"inside a job", 100, false},
	{"after the room bound", 130, false},
	{"after a store call", 120, true},
	{"across a season change", 150, false},
}

// TestFork_Storage verifies that a world of the storage pack restored from its
// checkpoint continues exactly as the world that never stopped: same events,
// snapshot, audit and checkpoint bytes for 30 ticks, and the restore of a restore
// too. The growers own the barn of their role row, so the stores are in the agent record; what
// a restore must rebuild is the derived state: the room of each limited account from its
// weight and stores, the load from the lots, and the pledge from the jobs in progress; the
// audit of the restored world names any drift.
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
	cfg := loadValidPack(t, packOneStorage...)
	for _, cell := range storageForkCells {
		t.Run(cell.name, func(t *testing.T) {
			t.Parallel()
			forkAtN(t, forkCase{
				Name: "storage", Cfg: cfg, Seed: 1, N: cell.n, M: forkM,
				Opts: func() []economysim.Option { return []economysim.Option{economysim.WithAudit()} },
				Before: func(w *economysim.World) {
					if cell.store {
						addBarnToACook(t, cfg, w)
					}
				},
			})
		})
	}
}

// addBarnToACook gives the first household of the role cook in region 0 one more barn, the
// call of a consumer between two ticks (spec-storage.md 3.6.2).
func addBarnToACook(t *testing.T, cfg economysim.Config, w *economysim.World) {
	t.Helper()
	cook := slices.IndexFunc(cfg.Roles, func(r economysim.Role) bool { return r.ID == "cook" })
	ids, err := w.Agents(0)
	if err != nil {
		t.Fatalf("Agents(0) error = %v, want nil", err)
	}
	for _, id := range ids {
		if st, err := w.Agent(id); err == nil && st.Role == economysim.RoleID(cook) && st.Weight >= 1 && !st.External {
			if err := w.AddStore(id, 0); err != nil {
				t.Fatalf("AddStore(%d) error = %v, want nil", id, err)
			}
			return
		}
	}
	t.Fatalf("no household of the role cook in region 0")
}

// TestFork_StorageCargoInFlight verifies the fork-at-N of the storage pack on the variant two
// with the transport module and the scripted shippers (FR-STO-043, FR-STO-014): at a tick with
// cargo in flight to a consignee whose room is the pledge's, the original and two restores give
// the same events, snapshot, audit and checkpoint bytes for 20 ticks. The inbound load and the
// pledge are not saved: the restore derives them from the shipments of tag 17, and the cargo
// that arrives after the boundary is credited from the pledge it rebuilt.
//
// Requirement: FR-STO-043, FR-STO-020, FR-STO-039, FR-CHK-050
//
// Guard: a restore that forgot the cargo in flight would credit it against a room that was
// never pledged, and the audit of the restored world would name the pledge at the arrival.
func TestFork_StorageCargoInFlight(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	base, opts := shipperPackOn(t, packTwoStorage)
	cfg, _ := narrowConsignees(t, base, opts, 1, 4)
	for _, n := range []int{0, 2, 3, 5} {
		t.Run(fmt.Sprintf("N %d", n), func(t *testing.T) {
			t.Parallel()
			var inFlight int
			forkAtN(t, forkCase{
				Name: "storage two shippers", Cfg: cfg, Seed: 1, N: n, M: 20,
				Opts: func() []economysim.Option { return append([]economysim.Option{economysim.WithAudit()}, opts...) },
				Before: func(w *economysim.World) {
					s, err := w.Snapshot()
					if err != nil {
						t.Fatalf("Snapshot() error = %v, want nil", err)
					}
					inFlight = len(s.Shipments)
				},
				Inspect: func(t *testing.T, _, _ []economysim.Event) {
					if (n == 2 || n == 3) && inFlight == 0 {
						t.Errorf("N %d: no shipment in flight at the boundary, want the cell to be inside a journey", n)
					}
				},
			})
		})
	}
}
