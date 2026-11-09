package economysim_test

import (
	"fmt"
	"log/slog"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// packOnePopResources is pack one with the fixed factor of overlay/resources-binding.toml
// and the population module at the placeholders of overlay/population.toml (renewal 0):
// the pack of S39 and S41 (plan 2-11 D438).
var packOnePopResources = append(slices.Clone(packOne), "overlay/resources-binding.toml", "overlay/population.toml")

// popStats are the measures of one run of the population scenarios.
type popStats struct {
	// Start, End and Min are the households at the start, at the end and the least
	// seen at a sample (every popSampleEvery ticks).
	Start, End, Min int64
	// Births, Deaths, Unfed, Exposed, Unfunded, Held and Capped are the sums of the
	// fields of PopulationChanged over the run.
	Births, Deaths, Unfed, Exposed, Unfunded, Held, Capped int64
	// Crossings counts the samples at which the household count moved from one
	// side of the start to the other.
	Crossings int
	// Bad is the first PopulationChanged whose counts do not add up (FR-POP-022).
	Bad string
	// Ratio is the household-weighted mean welfare ratio at the end, Pool the money
	// of the pools.
	Ratio, Pool int64
	// Warns counts the Warn lines of the log by message.
	Warns map[string]int
	// Drawn and Short are the units the starts took and the ration refused, summed
	// over the endowments and the run; StockShareBP is the end stock over the
	// capacity of the first endowment row, in basis points.
	Drawn, Short, StockShareBP int64
	Final                      economysim.Snapshot
}

// popSampleEvery is the sampling interval of the household count.
const popSampleEvery = 20

// withHouseholds multiplies the households of every group, and the capacity and the
// starting stock of every endowment, by num/den (at least 1 household), the start
// of the two runs of S37 (0.6 and 1.6 of the pack's count): the fixed factor keeps
// the headroom the overlay sized, so the world stays viable.
func withHouseholds(cfg economysim.Config, num, den int64) economysim.Config {
	cfg.Groups = slices.Clone(cfg.Groups)
	for i := range cfg.Groups {
		cfg.Groups[i].Households = max(cfg.Groups[i].Households*num/den, 1)
	}
	if cfg.Modules.Resources != nil {
		res := *cfg.Modules.Resources
		res.Endowments = slices.Clone(res.Endowments)
		for i := range res.Endowments {
			e := &res.Endowments[i]
			e.Capacity, e.Initial = e.Capacity*num/den, e.Initial*num/den
		}
		cfg.Modules.Resources = &res
	}
	return cfg
}

// runPopulation runs cfg for ticks with the audit on (conservation is checked every
// tick, a failure is a Step error), the recorder and a capturing logger, and keeps
// the measures. It holds a slot of scenSem. Exact checks that hold for any run are
// made here: no negative balance or stock at the end, and per PopulationChanged
// event the identities of the counts.
func runPopulation(t testing.TB, cfg economysim.Config, seed uint64, ticks int64, script []shockStep) popStats {
	t.Helper()
	scenSem <- struct{}{}
	defer func() { <-scenSem }()
	rec, log := &recorder{}, newCaptureHandler()
	w, err := economysim.NewWorld(cfg, economysim.WithSeed(seed), economysim.WithObserver(rec), economysim.WithLogger(slog.New(log)), economysim.WithAudit())
	if err != nil {
		t.Fatalf("NewWorld(seed %d) error = %v, want nil", seed, err)
	}
	s0, err := w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot() error = %v, want nil", err)
	}
	st := popStats{Start: s0.Households, Min: s0.Households, Warns: map[string]int{}}
	side := 0
	for range ticks {
		applyScript(t, w, s0.Names, script)
		if err := w.Step(); err != nil {
			t.Fatalf("seed %d: Step() at tick %d error = %v, want nil", seed, w.Tick(), err)
		}
		if w.Tick()%popSampleEvery != 0 {
			continue
		}
		s, err := w.Snapshot()
		if err != nil {
			t.Fatalf("Snapshot() error = %v, want nil", err)
		}
		st.Min = min(st.Min, s.Households)
		if now := cmpSide(s.Households, st.Start); now != 0 {
			if side != 0 && now != side {
				st.Crossings++
			}
			side = now
		}
	}
	if bad := economysim.NegativeAccounts(w); len(bad) > 0 {
		t.Errorf("seed %d: negative balance or stock at the end: %v", seed, bad)
	}
	if st.Final, err = w.Snapshot(); err != nil {
		t.Fatalf("Snapshot() error = %v, want nil", err)
	}
	st.End = st.Final.Households
	st.tally(eventsAs[economysim.PopulationChanged](rec.events))
	for _, e := range eventsAs[economysim.EndowmentChanged](rec.events) {
		st.Drawn, st.Short = st.Drawn+e.Drawn, st.Short+e.Short
		if e.Region == 0 && e.Resource == 0 && e.Capacity > 0 {
			st.StockShareBP = e.Stock * 10_000 / e.Capacity
		}
	}
	var weighted, hh int64
	for _, r := range st.Final.Roles {
		if r.HasMeanRatio {
			weighted, hh = weighted+r.MeanRatioBP*r.RatioHouseholds, hh+r.RatioHouseholds
		}
	}
	if hh > 0 {
		st.Ratio = weighted / hh
	}
	for _, g := range st.Final.Regions {
		st.Pool += int64(g.MoneyPool)
	}
	for _, l := range log.records() {
		if l.level == slog.LevelWarn {
			st.Warns[l.msg]++
		}
	}
	return st
}

// cmpSide is -1, 0 or 1 as a is below, at or above b.
func cmpSide(a, b int64) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	}
	return 0
}

// tally sums the events and records the first whose counts do not add up: the
// births done, capped and unfunded make the births due, and the deaths done, cut
// and held make the deaths due (FR-POP-022).
func (st *popStats) tally(events []economysim.PopulationChanged) {
	for _, e := range events {
		st.Births, st.Deaths, st.Unfed, st.Exposed = st.Births+e.BirthsDone, st.Deaths+e.DeathsDone, st.Unfed+e.Unfed, st.Exposed+e.Exposed
		st.Unfunded, st.Held, st.Capped = st.Unfunded+e.BirthsUnfunded, st.Held+e.DeathsHeld, st.Capped+e.BirthsCapped
		if st.Bad == "" && (e.BirthsDone+e.BirthsCapped+e.BirthsUnfunded != e.BirthsDue || e.DeathsDone+e.DeathsCut+e.DeathsHeld != e.DeathsDue) {
			st.Bad = fmt.Sprintf("%+v", e)
		}
	}
}

// reportPop records the measures of one run under a label.
func reportPop(rep *scenReport, seed uint64, label string, st popStats) {
	rep.num(seed, label+": households at the start", st.Start)
	rep.num(seed, label+": households at the end", st.End)
	rep.num(seed, label+": least households", st.Min)
	rep.num(seed, label+": births", st.Births)
	rep.num(seed, label+": deaths", st.Deaths)
	rep.num(seed, label+": mean ratio at the end, bp", st.Ratio)
	rep.num(seed, label+": crossings of the start", int64(st.Crossings))
}
