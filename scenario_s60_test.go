package economysim_test

import (
	"fmt"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// s60Ticks is the length of the invariant run of S60 on pack one with the overlay:
// the households whose stock of prepared is 0 enter the mode at tick 2 and the
// run covers their first jobs and the market's answer.
const s60Ticks int64 = 120

// boundaryReasons returns the reasons of the BoundaryFlow events, sorted and
// without repeats.
func boundaryReasons(events []economysim.Event) []economysim.Reason {
	var out []economysim.Reason
	for _, e := range events {
		if b, ok := e.(economysim.BoundaryFlow); ok && !slices.Contains(out, b.Reason) {
			out = append(out, b.Reason)
		}
	}
	slices.Sort(out)
	return out
}

// TestScenarioS60_InvariantsCheckpointRelocation runs scenario S60 (SUB-S5, plan 2-10
// D417) in three parts. (a) The invariant run: pack one with
// overlay/own-use.toml for 120 ticks over seeds 1 to scenarioSeedCount(), with the
// audit, the invariants of testing.md 6.3 and the recorded events checked against
// the snapshot every tick; households enter the mode; and the ledger reasons of the
// boundary flows are within those of the module off (no reason of its own,
// FR-SUB-003). (b) A checkpoint in the middle of the mode: pack one with the
// overlay is forked at tick 5, inside a job of a household in the mode, and the
// restored world and the restore of a restore are bit-identical to the original in
// events, snapshot and checkpoint bytes for 15 ticks (FR-SUB-044). (c) A
// relocation: a household in the mode of a two-region forage world is moved by the
// one relocation path, and its flag is dropped at once, before the next flag step
// (FR-SUB-031). The outcomes of (a) are written to
// testdata/chains/expected/one.scenarios.txt with -update.
//
// Requirement: FR-SUB-003, FR-SUB-031, FR-SUB-044
//
// Guard: a module that opened a reason of its own in the books, a checkpoint that
// forgot the mode, or a relocation that carried the flag to a region with another
// floor, fails here.
func TestScenarioS60_InvariantsCheckpointRelocation(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end scenario, skipped with -short")
	}
	const scenario, pack = "S60", "one+own-use"
	cfg := loadValidPack(t, packOneSubsistence...)
	off := loadValidPack(t, packOne...)
	rep := newScenReport(fmt.Sprintf("S60 one+own-use invariants seeds 1-%d, reasons seeds 1-%d, %d ticks", scenarioSeedCount(), min(2, scenarioSeedCount()), s60Ticks))
	// The parts run in a group, so that every part has finished when the report is flushed.
	t.Run("parts", func(t *testing.T) {
		t.Run("invariants", func(t *testing.T) {
			for _, seed := range seedList(scenarioSeedCount()) {
				t.Run(fmt.Sprintf("seed %d", seed), func(t *testing.T) {
					t.Parallel()
					runInvariants(t, cfg, seed, s60Ticks, nil)
				})
			}
		})
		t.Run("reasons and the mode", func(t *testing.T) {
			t.Parallel()
			for _, seed := range seedList(min(2, scenarioSeedCount())) {
				recOn, recOff := &recorder{}, &recorder{}
				for _, r := range []struct {
					cfg economysim.Config
					rec *recorder
				}{{cfg, recOn}, {off, recOff}} {
					w, err := economysim.NewWorld(r.cfg, economysim.WithSeed(seed), economysim.WithObserver(r.rec), economysim.WithAudit())
					if err != nil {
						t.Fatalf("NewWorld(seed %d) error = %v, want nil", seed, err)
					}
					step(t, w, int(s60Ticks))
				}
				on, base := boundaryReasons(recOn.events), boundaryReasons(recOff.events)
				extra := slices.DeleteFunc(slices.Clone(on), func(r economysim.Reason) bool { return slices.Contains(base, r) })
				checkExact(t, rep, scenario, pack, seed, "no ledger reason beyond the module off", len(extra) == 0, "reasons only with the module: %v", extra)
				entered := subsistenceEntered(recOn.events)
				rep.num(seed, "households entered the mode", entered)
				checkExact(t, rep, scenario, pack, seed, "households enter the mode", entered > 0, "entered %d", entered)
			}
		})
		t.Run("checkpoint in the middle of the mode", func(t *testing.T) {
			t.Parallel()
			forkAtN(t, forkCase{Name: "S60 one+own-use", Cfg: cfg, Seed: 1, N: 5, M: 15, Logs: true,
				Opts: func() []economysim.Option { return []economysim.Option{economysim.WithAudit()} },
				Inspect: func(t *testing.T, restored, since []economysim.Event) {
					if subsistenceEntered(restored)+countOwnUse(since) == 0 {
						t.Errorf("S60: no OwnUseChanged after the boundary, want the mode to be running in it")
					}
				}})
		})
		t.Run("relocation of a household in the mode", func(t *testing.T) {
			t.Parallel()
			w, err := economysim.NewWorld(forageWorld(forageSpec{Households: 4, OwnUseBP: 10_000, Duration: 3, Module: true, RetryJobs: 10, ExitTick: 40, Away: true}),
				economysim.WithSeed(1), economysim.WithAudit())
			if err != nil {
				t.Fatalf("NewWorld() error = %v, want nil", err)
			}
			step(t, w, 2)
			onOwnUse := func(region economysim.RegionID) (n int64) {
				s, err := w.Snapshot()
				if err != nil {
					t.Fatalf("Snapshot() error = %v, want nil", err)
				}
				for _, r := range s.Roles {
					if r.Region == region {
						n += r.OnOwnUse
					}
				}
				return n
			}
			if home := onOwnUse(0); home != 4 {
				t.Fatalf("households in the mode at home before the move = %d, want 4 (no grain, a job of 3 ticks, ExitTicks 40)", home)
			}
			// Agent 0 and 1 are the estates of the two regions; the first household is 2.
			if err := economysim.RelocateForTest(w, 2, 1, economysim.CauseConsumer); err != nil {
				t.Fatalf("RelocateForTest() error = %v, want nil", err)
			}
			if home, away := onOwnUse(0), onOwnUse(1); home != 3 || away != 0 {
				t.Errorf("households in the mode after the move: home %d, away %d, want 3 and 0 (the flag is dropped with the relocation)", home, away)
			}
			step(t, w, 4)
		})
	})
	rep.flush(t)
}

// countOwnUse returns the OwnUseChanged events in events.
func countOwnUse(events []economysim.Event) (n int64) {
	for _, e := range events {
		if _, ok := e.(economysim.OwnUseChanged); ok {
			n++
		}
	}
	return n
}
