package economysim

import (
	"log/slog"
	"reflect"
	"slices"
	"testing"
)

// This file tests the observability of the population module (plan 2-11 Phase 5):
// PopulationChanged, the three bounded Warns, the Debug line and the snapshot
// figures. The worlds are those of population_internal_test.go.

// churnConfig is exitConfig with 60 household millers fed for a whole run and both
// schedules flat at 10 000, so births and deaths are frequent: the world of the
// conservation and non-interference tests.
func churnConfig() Config {
	c := exitConfig()
	c.Groups[0].Households = 60
	c.Roles[0].StartStock[0].Quantity = 1_000
	mod := DefaultPopulation()
	mod.BirthSchedule = []Knot{{Ratio: 0, Rate: 10_000}}
	mod.DeathSchedule = []Knot{{Ratio: 0, Rate: 10_000}}
	c.Modules.Population = &mod
	return c
}

// TestPopulationChanged_Emission verifies the event: one per region and role that
// has a non-zero count, in role order, after EntryLimited, with the counts of the
// step, and none in a quiet tick.
//
// Arithmetic: the scenario of TestExec_DeathAndBirth: the millers expose 2 + 1 = 3
// households, two deaths are due and done, one birth is due and done. The closed
// weaver exposes 1 household and has no count to report, so it gets no event.
//
// Requirement: FR-POP-031
func TestPopulationChanged_Emission(t *testing.T) {
	t.Parallel()
	rec := &evLog{}
	w := popWorld(t, DefaultPopulation(), WithObserver(rec))
	d := w.pop.denom
	setCarries(w, map[AgentID]int64{1: d - 150, 2: d - 100, 3: 0}, map[RoleID]int64{0: d - 1})
	runEntry(t, w, rec)
	got := of[PopulationChanged](rec)
	want := []PopulationChanged{{
		Tick: 0, Region: 0, Role: 0, Exposed: 3, BirthsDue: 1, DeathsDue: 2, BirthsDone: 1, DeathsDone: 2,
	}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("seed %d: PopulationChanged = %+v, want %+v", popSeed, got, want)
	}
	lim, pc := -1, -1
	for i, e := range rec.events {
		switch e.(type) {
		case EntryLimited:
			lim = i
		case PopulationChanged:
			pc = i
		}
	}
	if lim < 0 || pc < lim {
		t.Errorf("seed %d: EntryLimited at %d, PopulationChanged at %d, want the second after the first", popSeed, lim, pc)
	}

	quiet := &evLog{}
	w = popWorld(t, DefaultPopulation(), WithObserver(quiet))
	runEntry(t, w, quiet)
	if got := of[PopulationChanged](quiet); len(got) != 0 {
		t.Errorf("seed %d: a quiet tick emitted %+v, want none", popSeed, got)
	}
}

// TestPopulationChanged_DoneEqualsExecuted verifies that, over a run, the sum of
// DeathsDone equals the households removed with the cause population and the sum
// of BirthsDone the households added with the cause births, read from
// WeightChanged, AgentSpawned and AgentBankrupted.
//
// Requirement: FR-POP-032
func TestPopulationChanged_DoneEqualsExecuted(t *testing.T) {
	t.Parallel()
	rec := &evLog{}
	w := newMill(t, churnConfig(), WithSeed(popSeed), WithObserver(rec))
	if err := w.estates.seedPool(w.ledger, 0, 500_000); err != nil {
		t.Fatalf("seed %d: seedPool() error = %v, want nil", popSeed, err)
	}
	rec.events = nil
	step(t, w, 100)
	var births, deaths, doneB, doneD int64
	for _, e := range rec.events {
		switch e := e.(type) {
		case WeightChanged:
			if e.Cause == CauseBirths {
				births += e.To - e.From
			}
			if e.Cause == CausePopulation {
				deaths += e.From - e.To
			}
		case AgentSpawned:
			births += e.Weight
		case AgentBankrupted:
			if e.Cause == CausePopulation {
				deaths += e.Weight
			}
		case PopulationChanged:
			doneB, doneD = doneB+e.BirthsDone, doneD+e.DeathsDone
		}
	}
	if births == 0 || deaths == 0 || births != doneB || deaths != doneD {
		t.Errorf("seed %d: executed %d births and %d deaths, events say %d and %d, want equal and non-zero", popSeed, births, deaths, doneB, doneD)
	}
}

// TestWarns_Population verifies the three bounded Warns: each is emitted at the
// first tick of its condition and then every tenth, with a constant message and
// the keys of spec-population.md 10.3, no Info is emitted, and a Debug line
// "population step" is written per region.
//
// Arithmetic: a region with no households, two births due and none done (the pool
// funded none) and one death held. The run counter reaches 1 on the first tick and
// 10 on the tenth, so over 12 ticks each Warn is logged twice.
//
// Requirement: FR-POP-033
func TestWarns_Population(t *testing.T) {
	t.Parallel()
	lg := newCapture()
	w := popWorld(t, DefaultPopulation(), WithLogger(slog.New(lg)))
	*lg.lines = nil
	for range 12 {
		w.pop.cnt[0] = popCount{birthsDue: 2, deathsDue: 1, deathsHeld: 1}
		w.popReport(0, 0)
	}
	counts := map[string]int{}
	for _, l := range *lg.lines {
		counts[l.msg]++
		switch l.msg {
		case "region extinct":
			if l.level != slog.LevelWarn || l.attrs["region"] != 0 {
				t.Errorf("seed %d: %+v, want a Warn with region 0", popSeed, l)
			}
		case "births unfunded":
			if l.level != slog.LevelWarn || l.attrs["region"] != 0 || l.attrs["births"] != 2 {
				t.Errorf("seed %d: %+v, want a Warn with region 0 and births 2", popSeed, l)
			}
		case "deaths held":
			if l.level != slog.LevelWarn || l.attrs["region"] != 0 || l.attrs["role"] != 0 || l.attrs["deaths"] != 1 {
				t.Errorf("seed %d: %+v, want a Warn with region 0, role 0 and deaths 1", popSeed, l)
			}
		case "population step":
			if l.level != slog.LevelDebug {
				t.Errorf("seed %d: %+v, want a Debug line", popSeed, l)
			}
		default:
			t.Errorf("seed %d: unexpected line %+v", popSeed, l)
		}
		if l.level == slog.LevelInfo {
			t.Errorf("seed %d: Info line %+v inside the tick, want none", popSeed, l)
		}
	}
	want := map[string]int{"region extinct": 2, "births unfunded": 2, "deaths held": 2, "population step": 12}
	if !reflect.DeepEqual(counts, want) {
		t.Errorf("seed %d: lines by message = %v, want %v", popSeed, counts, want)
	}
}

// TestSnapshot_Population verifies the figures of the snapshot: the weighted mean
// rates, the natural increase and the birth carry per role, the cap, and r* with
// the gain; and that taking a snapshot moves no carry.
//
// Arithmetic: every agent is at the ratio 14 000, birth 250 and death 150, so the
// mean rates are 250 and 150 and the natural increase is 100. r* is 12 000 (12.1),
// dRate = (250 - 50) - (250 - 450) = 400 over dRatio = 8 000, so the gain is
// trunc(12 000 * 400 / 8 000) = 600.
//
// Requirement: FR-POP-034
func TestSnapshot_Population(t *testing.T) {
	t.Parallel()
	mod := DefaultPopulation()
	mod.Regions = []RegionPopulation{{Region: "home", BirthScale: 10_000, DeathScale: 10_000, HouseholdCap: 50}}
	w := popWorld(t, mod)
	setCarries(w, nil, map[RoleID]int64{0: 77})
	carries := slices.Clone(w.pop.deathCarry)
	s, err := w.Snapshot()
	if err != nil {
		t.Fatalf("seed %d: Snapshot() error = %v, want nil", popSeed, err)
	}
	for r := range 2 {
		row := s.Roles[r]
		if row.BirthRateBP != 250 || row.DeathRateBP != 150 || row.NaturalIncreaseBP != 100 {
			t.Errorf("seed %d: role %d rates = %d, %d, %d, want 250, 150, 100", popSeed, r, row.BirthRateBP, row.DeathRateBP, row.NaturalIncreaseBP)
		}
	}
	if s.Roles[0].BirthCarry != 77 || s.Regions[0].HouseholdCap != 50 {
		t.Errorf("seed %d: BirthCarry = %d and HouseholdCap = %d, want 77 and 50", popSeed, s.Roles[0].BirthCarry, s.Regions[0].HouseholdCap)
	}
	if want := (PopulationState{ReplacementRatioBP: 12_000, HasReplacement: true, GainBP: 600}); s.Population != want {
		t.Errorf("seed %d: Population = %+v, want %+v", popSeed, s.Population, want)
	}
	if !slices.Equal(carries, w.pop.deathCarry) || w.pop.birthCarry[0] != 77 {
		t.Errorf("seed %d: a snapshot moved a carry", popSeed)
	}
	off, err := newMill(t, exitConfig(), WithSeed(popSeed)).Snapshot()
	if err != nil || off.Population != (PopulationState{}) || off.Roles[0].BirthRateBP != 0 {
		t.Errorf("seed %d: off snapshot Population = %+v, error %v, want the zero value", popSeed, off.Population, err)
	}
}

// TestObservation_DoesNotChangePopulation verifies that a run with an observer and
// a logger attached makes the same births and deaths as one without: the carries,
// the household count and the snapshot are equal.
//
// Requirement: NFR-POP-003
func TestObservation_DoesNotChangePopulation(t *testing.T) {
	t.Parallel()
	run := func(opts ...Option) *World {
		w := newMill(t, churnConfig(), append([]Option{WithSeed(popSeed)}, opts...)...)
		if err := w.estates.seedPool(w.ledger, 0, 500_000); err != nil {
			t.Fatalf("seed %d: seedPool() error = %v, want nil", popSeed, err)
		}
		step(t, w, 60)
		return w
	}
	bare := run()
	seen := run(WithObserver(&evLog{}), WithLogger(slog.New(newCapture())))
	a, errA := bare.Snapshot()
	b, errB := seen.Snapshot()
	if errA != nil || errB != nil || !reflect.DeepEqual(a, b) || !slices.Equal(bare.pop.deathCarry, seen.pop.deathCarry) ||
		!slices.Equal(bare.pop.birthCarry, seen.pop.birthCarry) {
		t.Errorf("seed %d: the observed world differs from the bare one (snapshot errors %v, %v)", popSeed, errA, errB)
	}
}

// TestPlan_EventOrderOfARegion verifies the order of the events of a region with a
// ladder reduction, a death, a birth and the aggregates: the ladder's WeightChanged
// (starved), then the module's death, then the birth, then EntryLimited, then
// PopulationChanged.
//
// Arithmetic: agent 1 (weight 2) has one unfed household at privation 50 since long
// before, so with ExitShare 10 000 the ladder takes it (starved, 2 -> 1). Agent 2
// (weight 1, fed) has a death due. Births are due in the millers; the agent of
// smallest weight below the scale takes the household.
//
// Requirement: FR-POP-019
func TestPlan_EventOrderOfARegion(t *testing.T) {
	t.Parallel()
	rec := &evLog{}
	w := popWorld(t, DefaultPopulation(), WithObserver(rec))
	w.cfg.World.ExitShare, w.cfg.World.RemovePrivation = 10_000, 1
	a := &w.agents[1]
	a.unfed, a.privation, a.roleSince = 1, 50, -1_000
	d := w.pop.denom
	setCarries(w, map[AgentID]int64{1: 0, 2: d - 100, 3: 0}, map[RoleID]int64{0: d - 1})
	runEntry(t, w, rec)
	var seq []string
	for _, e := range rec.events {
		switch e := e.(type) {
		case WeightChanged:
			seq = append(seq, "weight:"+string(e.Cause))
		case AgentBankrupted:
			seq = append(seq, "removed:"+string(e.Cause))
		case EntryLimited:
			seq = append(seq, "limited")
		case PopulationChanged:
			seq = append(seq, "population")
		}
	}
	want := []string{"weight:starved", "removed:population", "weight:births", "limited", "population"}
	if !slices.Equal(seq, want) {
		t.Errorf("seed %d: events = %v, want %v", popSeed, seq, want)
	}
}
