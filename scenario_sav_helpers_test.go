package economysim_test

import (
	"fmt"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// The helpers of the scenarios of the saving module (S142 to S147, spec-saving-and-investment.md
// 14.3, spec-reference-chain.md 7.12, plan 3-sav Phase 7). They run on pack one with the
// infrastructure and saving overlays (the pack of the golden), assert what is exact (the ledger
// identities through the audit, the equality of two worlds, a count that a rule makes zero) and
// report, never assert, what the bands of STY will judge (N-015, N-027). A run keeps counters and
// the sampled snapshot rows, not snapshots (plan 3-sav, gate economics).

// savingStats is what a scenario run keeps of the saving pack: the sums of the SavingPlanned counts
// over the run (household-ticks, as the event counts households), the facilities the world built
// and repaired, and the sampled sums of unfed households and of idle money of the roles with a block.
type savingStats struct {
	Planned                                   int64
	Saving, Buffered, Passed, Building        int64
	NoReturn, Standing, Unreachable, Survival int64
	Builds, Repairs, Built, Repaired          int64
	Goal, Buffer                              economysim.Money
	UnfedSamples, IdleSamples, IdleNegative   int64
}

// Observe adds one event to the stats.
func (s *savingStats) Observe(e economysim.Event) {
	switch v := e.(type) {
	case economysim.SavingPlanned:
		s.Planned++
		s.Saving += v.Saving
		s.Buffered += v.Buffered
		s.Passed += v.Passed
		s.Building += v.Building
		s.NoReturn += v.NoReturn
		s.Standing += v.Standing
		s.Unreachable += v.Unreachable
		s.Survival += v.Survival
		s.Builds += v.Builds
		s.Repairs += v.Repairs
		s.Goal += v.Goal
		s.Buffer += v.Buffer
	case economysim.FacilityChanged:
		switch v.Kind {
		case economysim.FacilityBuilt:
			s.Built++
		case economysim.FacilityRepaired:
			s.Repaired++
		}
	}
}

// savingSampleEvery is the spacing in ticks of the snapshots a scenario run samples.
const savingSampleEvery = 20

// savingRun runs cfg for seed and ticks with the audit on and returns the stats. It samples the
// snapshot every savingSampleEvery ticks and adds the unfed households and the idle money of the
// rows to the stats.
func savingRun(t testing.TB, cfg economysim.Config, seed uint64, ticks int64) *savingStats {
	t.Helper()
	st := &savingStats{}
	w, err := economysim.NewWorld(cfg, economysim.WithSeed(seed), economysim.WithAudit(), economysim.WithObserver(st))
	if err != nil {
		t.Fatalf("seed %d: NewWorld() error = %v, want nil", seed, err)
	}
	for w.Tick() < ticks {
		if err := w.Step(); err != nil {
			t.Fatalf("seed %d: Step() at tick %d error = %v, want nil", seed, w.Tick(), err)
		}
		if w.Tick()%savingSampleEvery != 0 {
			continue
		}
		snap, err := w.Snapshot()
		if err != nil {
			t.Fatalf("seed %d: Snapshot() at tick %d error = %v, want nil", seed, w.Tick(), err)
		}
		for _, r := range snap.Roles {
			st.UnfedSamples += r.Unfed
			st.IdleSamples += int64(r.IdleMoney)
			if r.IdleMoney < 0 {
				st.IdleNegative++
			}
		}
	}
	return st
}

// savingPack loads the pack of the golden and applies edit to it, for a scenario that varies a
// field of the module or of a recipe; each call loads a fresh configuration, so an edit never
// reaches another run.
func savingPack(t testing.TB, edit func(*economysim.Config)) economysim.Config {
	t.Helper()
	cfg := loadValidPack(t, packOneSaving...)
	if edit != nil {
		edit(&cfg)
	}
	return cfg
}

// savingScenario runs fn for the seeds of the scenario as parallel subtests and records the block
// of the pack under title in the report of pack one.
func savingScenario(t *testing.T, title string, seeds int, fn func(t *testing.T, rep *scenReport, seed uint64)) {
	t.Helper()
	rep := newPackScenReport("one", fmt.Sprintf("%s seeds 1-%d", title, seeds))
	t.Run("seeds", func(t *testing.T) {
		for _, seed := range seedList(seeds) {
			t.Run(fmt.Sprintf("seed %d", seed), func(t *testing.T) {
				t.Parallel()
				fn(t, rep, seed)
			})
		}
	})
	rep.flush(t)
}
