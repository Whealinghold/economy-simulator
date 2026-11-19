package economysim_test

import (
	"fmt"
	"maps"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// The end-to-end tests of what a body does to a traineeship (PR D, spec-institutions.md 4.5.2
// and 5.7): the train right that makes a trainer a member, the body's own maximum of
// trainees and the completion that waits for a place. They run pack five with the skill
// overlay, where the cooks of north train and households of other roles apply to them, and
// read only the public API: events, the snapshot and the checkpoint.

// trainDoc is the institutions tables over the cooks of north: a body guild, a train right
// held by it, and the places and the trainee limit that the format verbs fill in. north is
// the region of pack five whose cooks are 222 households (testdata/chains/five.toml).
const trainDoc = `
[modules.institutions]

[modules.institutions.body.guild]
roles = ["cook"]
regions = ["north"]
places = %d
max_trainees = %d

[modules.institutions.right.teach]
body = "guild"
activity = "train"
subject = "cook"
region = "north"
`

// trainTicks is the length of the runs below: the course of the overlay is 120 ticks, so
// the first trainees reach their last tick near tick 125 and the rest of the run shows what
// happens at the end of the course.
const trainTicks = 200

// trainRun is what one run of a trainee world kept, for the region north.
type trainRun struct {
	started, completed, held int64 // TraineeshipStarted and Completed of north, sum of MembershipChanged.Held
	worstLoad                int64 // the most trainees one trainer held at a sampled tick
	liveAtEnd                int   // trainees in the snapshot at the end of the run
	servedAtEnd              int64 // the most ticks served by a trainee at the end
	cooks                    int64 // cook households of north at the end
	hash                     [32]byte
}

// runTrain runs cfg for trainTicks from seed with the audit and reads what it did to the
// traineeships of north every fifth tick. The north of the pack is the region ID of its name.
func runTrain(t testing.TB, cfg economysim.Config, seed uint64) trainRun {
	t.Helper()
	var r trainRun
	var north economysim.RegionID = -1
	obs := economysim.ObserverFunc(func(e economysim.Event) {
		switch e := e.(type) {
		case economysim.TraineeshipStarted:
			if e.Region == north {
				r.started++
			}
		case economysim.TraineeshipCompleted:
			if e.Region == north {
				r.completed++
			}
		case economysim.MembershipChanged:
			if e.Region == north {
				r.held += e.Held
			}
		}
	})
	w, err := economysim.NewWorld(cfg, economysim.WithSeed(seed), economysim.WithAudit(), economysim.WithObserver(obs))
	if err != nil {
		t.Fatalf("seed %d: NewWorld() error = %v, want nil", seed, err)
	}
	s, err := w.Snapshot()
	if err != nil {
		t.Fatalf("seed %d: Snapshot() error = %v, want nil", seed, err)
	}
	north = economysim.RegionID(slices.Index(s.Names.Regions, "north"))
	if north < 0 {
		t.Fatalf("seed %d: the names table has no region north: %v", seed, s.Names.Regions)
	}
	for i := range trainTicks {
		if err := w.Step(); err != nil {
			t.Fatalf("seed %d: Step() at tick %d error = %v, want nil", seed, i, err)
		}
		if i%5 != 0 {
			continue
		}
		snap, err := w.Snapshot()
		if err != nil {
			t.Fatalf("seed %d: Snapshot() at tick %d error = %v, want nil", seed, i, err)
		}
		load := map[economysim.AgentID]int64{}
		for _, tr := range snap.Trainees {
			if tr.Region == north {
				load[tr.Trainer]++
			}
		}
		for _, n := range slices.Collect(maps.Values(load)) {
			r.worstLoad = max(r.worstLoad, n)
		}
	}
	final, err := w.Snapshot()
	if err != nil {
		t.Fatalf("seed %d: Snapshot() error = %v, want nil", seed, err)
	}
	cook := slices.Index(final.Names.Roles, "cook")
	for _, row := range final.Roles {
		if int(row.Role) == cook && row.Region == north {
			r.cooks = row.Households
		}
	}
	for _, tr := range final.Trainees {
		if tr.Region == north {
			r.liveAtEnd++
			r.servedAtEnd = max(r.servedAtEnd, tr.Served)
		}
	}
	r.hash = economysim.HashWorld(w)
	return r
}

// trainConfig is pack five with the skill overlay and the institutions tables of trainDoc.
func trainConfig(t testing.TB, places, maxTrainees int64) economysim.Config {
	t.Helper()
	return loadPackBytes(t, fmt.Appendf(nil, trainDoc, places, maxTrainees), packFiveSkills...)
}

// TestEndToEnd_InstitutionsTrainRightLimitsTrainees verifies, from outside, that a body's
// maximum of trainees binds the traineeships of a whole run: with the right held by a body
// that has a place for every cook and a maximum of 1, no trainer of north holds more than
// one trainee at any sampled tick, while the same pack without the module lets a trainer take
// the role's 3, and trainees do start under the body (the limit is not met by refusing all).
//
// Arithmetic: the overlay gives the cooks max_trainees 3; the body's 1 makes the limit
// min(3, 1) = 1 (FR-ORG-026). The body has 400 places for 222 cook households, so every
// cook is a member and every cook may train (FR-ORG-025), and nothing else restricts it.
//
// Requirement: FR-ORG-025, FR-ORG-026, FR-ORG-039
//
// Guard: the unit tests prove the limit over a hand-built cell; this is the proof that the
// limit reaches the application rule of the skill module through the world's own wiring, in a
// run where the households apply by themselves.
func TestEndToEnd_InstitutionsTrainRightLimitsTrainees(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	free := loadValidPack(t, packFiveSkills...)
	limited := trainConfig(t, 400, 1)
	for _, seed := range seedList(invariantSeeds(institutionsWorldSeeds)) {
		t.Run(fmt.Sprintf("seed %d", seed), func(t *testing.T) {
			t.Parallel()
			base, got := runTrain(t, free, seed), runTrain(t, limited, seed)
			t.Logf("seed %d: without the module %+v, with the body %+v", seed, base, got)
			if base.worstLoad < 2 {
				t.Fatalf("seed %d: the most trainees of one trainer without the module = %d, want 2 or more so the limit is not met by chance", seed, base.worstLoad)
			}
			if got.started == 0 {
				t.Errorf("seed %d: TraineeshipStarted in north under the body = 0, want some (the body holds the right and every cook is a member)", seed)
			}
			if got.worstLoad != 1 {
				t.Errorf("seed %d: the most trainees of one trainer under a body maximum of 1 = %d, want 1", seed, got.worstLoad)
			}
		})
	}
}

// TestEndToEnd_InstitutionsHeldCompletion verifies, from outside, the completion that waits
// for a place: with a body whose places are fewer than the cook households of north, a trainee that
// served its whole course completes nowhere in the region, stays in the snapshot at the end
// of its course, and MembershipChanged counts it as held every tick; the same pack without the
// module completes the course. The run is deterministic: two runs of one seed agree in the
// state hash.
//
// Arithmetic: the course is 120 ticks and the run 200, so a trainee that started in the first
// ticks has served 120 by tick 130. The body has 40 places and the cook households of north
// are 222 at the start and about 60 to 130 at the end of the run (the cooks leave), never
// below 40, so the headroom of the cook role is places less households cut at 0 (FR-ORG-027):
// no completion has a place.
//
// Requirement: FR-ORG-027, FR-ORG-039, FR-ORG-033
//
// Guard: a completion that ignored the body's places would give a place to every trainee at the
// end of its course, which no test of a single call sees in a world where the households apply,
// train and leave by themselves.
func TestEndToEnd_InstitutionsHeldCompletion(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	free := loadValidPack(t, packFiveSkills...)
	full := trainConfig(t, 40, 0)
	for _, seed := range seedList(invariantSeeds(institutionsWorldSeeds)) {
		t.Run(fmt.Sprintf("seed %d", seed), func(t *testing.T) {
			t.Parallel()
			base, got := runTrain(t, free, seed), runTrain(t, full, seed)
			t.Logf("seed %d: without the module %+v, with the small body %+v", seed, base, got)
			if base.completed == 0 {
				t.Fatalf("seed %d: TraineeshipCompleted in north without the module = 0, want some in %d ticks so the hold is not vacuous", seed, trainTicks)
			}
			if got.started == 0 || got.held == 0 {
				t.Errorf("seed %d: started = %d, held = %d under the small body, want both above 0", seed, got.started, got.held)
			}
			if got.completed != 0 {
				t.Errorf("seed %d: TraineeshipCompleted in north under the small body = %d, want 0 (no place for a new cook)", seed, got.completed)
			}
			if got.servedAtEnd != 120 {
				t.Errorf("seed %d: the most ticks served by a trainee at the end = %d, want 120 (held at the end of the course)", seed, got.servedAtEnd)
			}
			if again := runTrain(t, full, seed); again != got {
				t.Errorf("seed %d: two runs of one seed differ: %+v and %+v", seed, again, got)
			}
		})
	}
}

// TestFork_InstitutionsTrainRight verifies the checkpoint round trip of a world with the
// module on and the train right in force: for a boundary before the trainees start, in the
// middle of the course and after the first holds, the original and two restores (one of a
// restore) give the same events, snapshot, audit and checkpoint bytes for 30 more ticks, so a
// restored world gives the held completion, the body's maximum and the next round of the
// uninterrupted run (the headroom slot is derived, not saved).
//
// Requirement: FR-ORG-033, FR-ORG-027, FR-CHK-050
func TestFork_InstitutionsTrainRight(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := trainConfig(t, 40, 1)
	for _, n := range []int{3, 60, 125} {
		forkAtN(t, forkCase{Name: "institutions train right", Cfg: cfg, Seed: 1, N: n, M: forkM, Logs: true})
	}
}
