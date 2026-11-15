package economysim_test

import (
	"fmt"
	"reflect"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// diffusionOnly is pack three with the four-mechanism overlay and the trend and the
// discoveries switched off, so the only way a cell without a leader's practice gains is
// contact: learning makes the gaps and the catch-up closes them.
func diffusionOnly(t *testing.T) economysim.Config {
	t.Helper()
	cfg := loadValidPack(t, technologyThree...)
	f := &cfg.Modules.Technology.Fields[0]
	f.TrendBP, f.DiscoveryBP, f.StepBP = 0, 0, 0
	f.Base = 10
	return cfg
}

// closeAllEdges closes every edge of the configuration, so no region has contact.
func closeAllEdges(t *testing.T, cfg economysim.Config, w *economysim.World) {
	t.Helper()
	ids := make([]string, len(cfg.Regions))
	for i, r := range cfg.Regions {
		ids[i] = r.ID
	}
	slices.Sort(ids) // a RegionID is the rank of the region's ID (spec-regions.md 3.1)
	id := func(s string) economysim.RegionID { return economysim.RegionID(slices.Index(ids, s)) }
	cut := make([]economysim.EdgeStatus, len(cfg.Edges))
	for i, e := range cfg.Edges {
		cut[i] = economysim.EdgeStatus{From: id(e.From), To: id(e.To), Closed: true, FactorBP: 10_000}
	}
	if err := w.SetEdgeStatuses(cut); err != nil {
		t.Fatalf("SetEdgeStatuses() error = %v, want nil", err)
	}
}

// TestScenarioS150_DiffusionAndAClosedEdge reports TEC-S3 of spec-technology.md 14.4 on
// pack three with the trade edges: with learning and catch-up only, a year, seeds 1 to
// 10, the catch-up adds a positive gain while the edges are open, and not one basis point
// when every edge is closed by SetEdgeStatuses before the first step; in both worlds the
// gains chain and stay within the cap, and with contact no cell's advance passes the
// largest advance of any cell (FR-TEC-016: the frontier is a share of a leader). The
// totals are recorded.
//
// Requirement: FR-TEC-013, FR-TEC-014, FR-TEC-015, FR-TEC-016, FR-TEC-036
//
// Guard: a closed edge that still carried knowledge, or a catch-up that passed its
// frontier, fails here.
func TestScenarioS150_DiffusionAndAClosedEdge(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end scenario, skipped with -short")
	}
	cfg := diffusionOnly(t)
	techScenario(t, "three", "S150 three+trade+technology contact, 365 ticks", func(t *testing.T, rep *scenReport, seed uint64) {
		open := runTechnology(t, cfg, seed, techTicks, nil)
		shut := runTechnology(t, cfg, seed, techTicks, func(w *economysim.World, i int) {
			if i == 0 {
				closeAllEdges(t, cfg, w)
			}
		})
		reportTech(rep, seed, open)
		rep.num(seed, "gain added by catch-up with every edge closed bp", shut.catchUp)
		checkChain(t, rep, "S150", "three", seed, open)
		checkChain(t, rep, "S150", "three", seed, shut)
		checkExact(t, rep, "S150", "three", seed, "contact carries knowledge", open.catchUp > 0, "catch-up %d bp with the edges open", open.catchUp)
		checkExact(t, rep, "S150", "three", seed, "a closed edge carries none", shut.catchUp == 0, "catch-up %d bp with every edge closed", shut.catchUp)
		var most int64
		for _, r := range open.final.Tech {
			most = max(most, r.AdvanceBP)
		}
		for _, r := range open.final.Tech {
			checkExact(t, rep, "S150", "three", seed, fmt.Sprintf("region %d advance within the largest", r.Region), r.AdvanceBP <= most, "advance %d, largest %d", r.AdvanceBP, most)
		}
	})
}

// TestScenarioS153_CheckpointAndReseed is TEC-S6, a harness test: on pack three with the
// four-mechanism overlay, seeds 1 to 10, a world checkpointed at tick 150 (carries and
// discoveries in the cells) and restored runs 60 more ticks to the same snapshot and the
// same checkpoint bytes as the world that never stopped, and a restore with a reseed
// discovers on other ticks than the original (the key follows the seed).
//
// Requirement: FR-TEC-027, FR-TEC-028, FR-TEC-012
//
// Guard: a carry or an advance left out of the section, or a key that survived a reseed,
// fails here.
func TestScenarioS153_CheckpointAndReseed(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end scenario, skipped with -short")
	}
	cfg := fastLearning(t)
	for _, seed := range seedList(scenarioSeedCount()) {
		t.Run(fmt.Sprintf("seed %d", seed), func(t *testing.T) {
			t.Parallel()
			a := newPackWorld(t, cfg, seed, economysim.WithAudit())
			runTicks(t, a, 150)
			ck := mustCheckpoint(t, a)
			b, err := economysim.RestoreWorld(cfg, ck, economysim.WithAudit())
			if err != nil {
				t.Fatalf("seed %d: RestoreWorld() error = %v, want nil", seed, err)
			}
			runTicks(t, a, 60)
			runTicks(t, b, 60)
			sa, _ := a.Snapshot()
			sb, _ := b.Snapshot()
			if !reflect.DeepEqual(sa, sb) || !reflect.DeepEqual(mustMarshal(t, mustCheckpoint(t, a)), mustMarshal(t, mustCheckpoint(t, b))) {
				t.Errorf("seed %d: the restored world differs from the world that never stopped after 60 ticks", seed)
			}
			var ra, rb recorder
			c := newPackWorld(t, cfg, seed, economysim.WithObserver(&ra))
			d, err := economysim.RestoreWorld(cfg, mustCheckpoint(t, c), economysim.WithReseed(11), economysim.WithObserver(&rb))
			if err != nil {
				t.Fatalf("seed %d: RestoreWorld(WithReseed) error = %v, want nil", seed, err)
			}
			if same, other := discoveryTicks(t, c, &ra, 300), discoveryTicks(t, d, &rb, 300); len(same)+len(other) > 0 && reflect.DeepEqual(same, other) {
				t.Errorf("seed %d: the reseeded restore discovers on the same ticks %v as the original", seed, same)
			}
		})
	}
}

// TestScenarioS154_TheCall reports TEC-S7: on pack one with learning only, seeds 1 to 10,
// a call after 200 ticks (the first jobs have completed) raises the advance of the cell by 800 and the gain follows at once
// by the same amount; a call of -500 lowers both by 500, lost art at once; a call of
// -1 000 when the advance is 300 takes it to 0 and reports LostBP 700. The gains chain
// through the calls (the observer follows TechnologyAdjusted).
//
// Requirement: FR-TEC-021, FR-TEC-022, FR-TEC-023
func TestScenarioS154_TheCall(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end scenario, skipped with -short")
	}
	cfg := loadValidPack(t, packOneTechnology...)
	techScenario(t, "one", "S154 one+technology the call, 230 ticks", func(t *testing.T, rep *scenReport, seed uint64) {
		var gains [4]int64
		var advances [4]int64
		var lost int64
		r := runTechnology(t, cfg, seed, 230, func(w *economysim.World, i int) {
			if i < 200 || (i-200)%10 != 0 {
				return
			}
			k := (i - 200) / 10
			if k > 2 {
				return
			}
			delta := [3]int64{800, -500, -1_000}[k]
			if err := w.AdjustTechnology([]economysim.TechnologyAdjustment{{Region: 0, Field: 0, DeltaBP: delta}}); err != nil {
				t.Fatalf("seed %d: AdjustTechnology(%d) error = %v, want nil", seed, delta, err)
			}
			s, _ := w.Snapshot()
			if len(s.Tech) > 0 {
				gains[k+1], advances[k+1] = s.Tech[0].GainBP, s.Tech[0].AdvanceBP
			}
		})
		lost = r.lost
		reportTech(rep, seed, r)
		rep.num(seed, "bp lost to the bounds over the three calls", lost)
		checkChain(t, rep, "S154", "one", seed, r)
		checkExact(t, rep, "S154", "one", seed, "advances after the three calls are 800, 300 and 0", advances[1] == 800 && advances[2] == 300 && advances[3] == 0, "advances %v", advances)
		checkExact(t, rep, "S154", "one", seed, "the cut of the last call is lost", lost == 700, "lost %d, want 700", lost)
		checkExact(t, rep, "S154", "one", seed, "the gain follows the advance at once", gains[1]-gains[2] == 500 && gains[2] > gains[3], "gains %v", gains)
	})
}
