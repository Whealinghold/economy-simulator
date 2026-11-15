package economysim_test

import (
	"errors"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// The end-to-end tests of the technology module's second slice (plan 3-tec, PR B): a
// consumer loads a pack with the module on, runs it and reads events, snapshot rows
// and the yields of production, and calls AdjustTechnology. The unit and white-box
// tests of the slice pin each rule; these prove the parts work together through the
// public API only.

// fastLearning loads the four-mechanism overlay on pack three and lowers the doubling
// base to 10 executions, so the gain moves inside 120 ticks (the overlay's own base of
// 1 000 is a design placeholder, N-015).
func fastLearning(t *testing.T) economysim.Config {
	t.Helper()
	cfg := loadValidPack(t, technologyThree...)
	for i := range cfg.Modules.Technology.Fields {
		cfg.Modules.Technology.Fields[i].Base = 10
	}
	return cfg
}

// TestTechnology_GainReachesProduction verifies end to end that a pack with the module
// on learns and that production sees it: over 120 ticks of pack three (seed 1) the
// per-cell TechnologyAdvanced events chain (each BeforeBP is the previous AfterBP of its
// cell, the gain never falls and stays within the cap), the final snapshot rows hold
// the last AfterBP of each cell, and the RecipeExecuted yields of the module-on run
// hold values the module-off run never has, each a yield of the off run lifted by at
// most the cap (a yield of 10 000 x site factor x (10 000 + G) / 10 000, so above its
// off value and at most the cap above it).
//
// Requirement: FR-TEC-010, FR-TEC-017, FR-TEC-018, FR-TEC-024, FR-TEC-026
func TestTechnology_GainReachesProduction(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := fastLearning(t)
	capBP := cfg.Modules.Technology.Fields[0].CapBP
	on := runPack(t, cfg, runSpec{Seed: 1, Ticks: 120, Observe: true, Audit: true})
	off := cfg
	off.Modules.Technology = nil
	base := runPack(t, off, runSpec{Seed: 1, Ticks: 120, Observe: true, Audit: true})

	type cell struct {
		r economysim.RegionID
		f economysim.TechFieldID
	}
	last := map[cell]int64{}
	adv := 0
	onYields, offYields := map[int64]bool{}, map[int64]bool{}
	for _, e := range on.Events {
		switch x := e.(type) {
		case economysim.TechnologyAdvanced:
			adv++
			k := cell{x.Region, x.Field}
			if x.BeforeBP != last[k] || x.AfterBP < x.BeforeBP || x.AfterBP > capBP {
				t.Errorf("TechnologyAdvanced %+v: BeforeBP want %d, AfterBP want in [BeforeBP, %d]", x, last[k], capBP)
			}
			last[k] = x.AfterBP
		case economysim.RecipeExecuted:
			onYields[x.EffectiveYield] = true
		}
	}
	for _, e := range base.Events {
		switch e.(type) {
		case economysim.TechnologyAdvanced, economysim.TechnologyAdjusted:
			t.Errorf("module off emitted %T, want no technology event", e)
		}
		if x, ok := e.(economysim.RecipeExecuted); ok {
			offYields[x.EffectiveYield] = true
		}
	}
	if adv == 0 {
		t.Fatalf("the run emitted no TechnologyAdvanced, want the cells to learn in 120 ticks")
	}
	rows := map[cell]int64{}
	for _, r := range on.Final.Tech {
		rows[cell{r.Region, r.Field}] = r.GainBP
	}
	for k, g := range last {
		if rows[k] != g {
			t.Errorf("Snapshot.Tech gain of %+v = %d, want %d of the last TechnologyAdvanced", k, rows[k], g)
		}
	}
	if len(base.Final.Tech) != 0 {
		t.Errorf("module off Snapshot.Tech = %v, want none", base.Final.Tech)
	}
	lifted := 0
	for y := range onYields {
		if offYields[y] {
			continue
		}
		lifted++
		ok := false
		for b := range offYields {
			if y > b && y <= b*(10_000+capBP)/10_000 {
				ok = true
			}
		}
		if !ok {
			t.Errorf("yield %d of the module-on run is not an off yield lifted by at most %d bp", y, capBP)
		}
	}
	if lifted == 0 {
		t.Errorf("no yield of the module-on run differs from the module off, want the gain to reach production")
	}
}

// TestTechnology_AdjustThroughThePublicAPI verifies a consumer's use of the call on a
// running world of pack three: with the module off the call is ErrModuleOff; with it
// on, a raise of 500 bp in region 0 emits TechnologyAdjusted with the tick of the next
// step, shows in Snapshot.Tech at once, and a fall of MaxGainBP takes the advance to 0, leaves the
// gain learned from experience and reports LostBP, the part the advance did not absorb. A refused call (an unknown region) changes the snapshot not
// at all.
//
// Requirement: FR-TEC-021, FR-TEC-022, FR-TEC-023
func TestTechnology_AdjustThroughThePublicAPI(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := fastLearning(t)
	offCfg := cfg
	offCfg.Modules.Technology = nil
	offW := mustWorld(t, offCfg, economysim.WithSeed(1))
	if err := offW.AdjustTechnology(nil); !errors.Is(err, economysim.ErrModuleOff) {
		t.Errorf("AdjustTechnology with the module off = %v, want ErrModuleOff", err)
	}

	rec := &recorder{}
	w := mustWorld(t, cfg, economysim.WithSeed(1), economysim.WithObserver(rec))
	for range 20 {
		if err := w.Step(); err != nil {
			t.Fatalf("Step() error = %v, want nil", err)
		}
	}
	rowOf := func() economysim.TechRow {
		s, err := w.Snapshot()
		if err != nil {
			t.Fatalf("Snapshot() error = %v, want nil", err)
		}
		for _, r := range s.Tech {
			if r.Region == 0 && r.Field == 0 {
				return r
			}
		}
		t.Fatalf("Snapshot.Tech has no row for region 0, field 0 after 20 ticks, want one")
		return economysim.TechRow{}
	}
	gainOf := func() int64 { return rowOf().GainBP }
	before := gainOf()
	if err := w.AdjustTechnology([]economysim.TechnologyAdjustment{{Region: 0, Field: 0, DeltaBP: 500}}); err != nil {
		t.Fatalf("AdjustTechnology(+500) error = %v, want nil", err)
	}
	if got, want := gainOf(), min(before+500, cfg.Modules.Technology.Fields[0].CapBP); got != want {
		t.Errorf("gain after +500 = %d, want %d (the gain before, %d, plus 500, cut at the cap)", got, want, before)
	}
	var ev economysim.TechnologyAdjusted
	for _, e := range rec.events {
		if x, ok := e.(economysim.TechnologyAdjusted); ok {
			ev = x
		}
	}
	if ev.Tick != w.Tick() || ev.BeforeBP != before || ev.AfterBP != gainOf() || ev.DeltaBP != 500 {
		t.Errorf("TechnologyAdjusted = %+v, want tick %d, DeltaBP 500, BeforeBP %d, AfterBP %d", ev, w.Tick(), before, gainOf())
	}

	row := rowOf()
	if err := w.AdjustTechnology([]economysim.TechnologyAdjustment{{Region: 0, Field: 0, DeltaBP: -economysim.MaxGainBP}}); err != nil {
		t.Fatalf("AdjustTechnology(-MaxGainBP) error = %v, want nil", err)
	}
	if got := rowOf(); got.AdvanceBP != 0 || got.GainBP != row.GainBP-row.AdvanceBP {
		t.Errorf("row after the fall = %+v, want AdvanceBP 0 and the gain %d (the gain learned from experience stays)", got, row.GainBP-row.AdvanceBP)
	}
	var cut economysim.TechnologyAdjusted
	for _, e := range rec.events {
		if x, ok := e.(economysim.TechnologyAdjusted); ok {
			cut = x
		}
	}
	if want := economysim.MaxGainBP - row.AdvanceBP; cut.LostBP != want {
		t.Errorf("LostBP of the fall = %d, want %d (the part of %d the advance of %d did not absorb)", cut.LostBP, want, economysim.MaxGainBP, row.AdvanceBP)
	}

	n := len(rec.events)
	if err := w.AdjustTechnology([]economysim.TechnologyAdjustment{{Region: 99, Field: 0, DeltaBP: 1}}); !errors.Is(err, economysim.ErrUnknownRegion) {
		t.Errorf("AdjustTechnology(region 99) = %v, want ErrUnknownRegion", err)
	}
	if len(rec.events) != n {
		t.Errorf("a refused call emitted %d events, want 0", len(rec.events)-n)
	}
}
