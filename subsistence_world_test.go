package economysim_test

import (
	"log/slog"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// The world the subsistence scenarios run on (plan 2-10 Phase 8). The scenarios
// need a household that can be starved by an outage of its market recipe and
// rescued by an own-use recipe, so they use one small region rather than pack
// one: pack one is large and its S-scenarios take minutes (the cost matters, the
// F run has a limit of 120 minutes), and its households hold stock for hundreds
// of ticks before a shortage reaches them.

// forageSpec says how forageWorld is built.
type forageSpec struct {
	Households int64
	// StartGrain is the grain each household holds at the start; the upkeep is 2 a tick.
	StartGrain int64
	// OwnUseBP is the yield of the own-use recipe in basis points: Cov = OwnUseBP.
	OwnUseBP int64
	// Duration is the ticks of the own-use recipe.
	Duration int64
	// Module switches the subsistence module on; RetryJobs, EnterTicks and
	// ExitTicks are its tunables.
	Module                          bool
	RetryJobs, EnterTicks, ExitTick int64
	// Away adds a second region "yonder" with no household (it sorts after "home", so it
	// is region 1), in which the own-use recipe has the same yield.
	Away bool
	// ChangePrivation sets the privation of rung 1 of the failure ladder (0 keeps
	// the default).
	ChangePrivation int64
	// MinWork is the role's MinWorkShare in basis points.
	MinWork int64
}

// forageWorld is one region "home" of foragers whose upkeep is 2 "grain" a tick.
// The market recipe "tend" makes 2 grain a tick at a yield of 10 000, so with
// nothing wrong a household feeds itself; a script that sets its yield to 0 is an
// outage. The own-use recipe "gather" (sited, 2 grain, spec.Duration ticks) is on
// the list of the module when it is on.
func forageWorld(spec forageSpec) economysim.Config {
	world := economysim.DefaultWorldParams()
	world.Upkeep = []economysim.Item{{Commodity: "grain", Quantity: 2}}
	world.OutputCoverTicks = 0
	grain := economysim.DefaultCommodity()
	grain.ID, grain.ReferencePrice, grain.Raw = "grain", 1_000, true
	region := economysim.DefaultRegion()
	region.ID, region.ReferenceBasket = "home", 2_000
	role := economysim.DefaultRole()
	role.ID, role.Recipes, role.StartMoney, role.MinWorkShare = "forager", []string{"tend"}, 0, spec.MinWork
	if spec.StartGrain > 0 {
		role.StartStock = []economysim.Item{{Commodity: "grain", Quantity: economysim.Quantity(spec.StartGrain)}}
	}
	dur := max(spec.Duration, 1)
	cfg := economysim.Config{
		World: world, Calendar: economysim.DefaultCalendar(),
		Commodities: []economysim.Commodity{grain}, Roles: []economysim.Role{role}, Regions: []economysim.Region{region},
		Groups: []economysim.Group{{Region: "home", Role: "forager", Households: spec.Households}},
		Recipes: []economysim.Recipe{
			{ID: "gather", Duration: dur, Sited: true, TraineeYield: 5_000, Outputs: []economysim.Output{{Commodity: "grain", Quantity: 2 * economysim.Quantity(dur), CostShare: 10_000}}},
			{ID: "tend", Duration: 1, TraineeYield: 5_000, Outputs: []economysim.Output{{Commodity: "grain", Quantity: 2, CostShare: 10_000}}},
		},
		Yields: []economysim.YieldProfile{{Region: "home", Recipe: "gather", HasAll: true, All: spec.OwnUseBP}},
	}
	if spec.Away {
		away := economysim.DefaultRegion()
		away.ID, away.ReferenceBasket = "yonder", 2_000
		cfg.Regions = append(cfg.Regions, away)
		cfg.Yields = append(cfg.Yields, economysim.YieldProfile{Region: "yonder", Recipe: "gather", HasAll: true, All: spec.OwnUseBP})
	}
	if spec.ChangePrivation > 0 {
		cfg.World.ChangePrivation = spec.ChangePrivation
	}
	if spec.Module {
		m := economysim.DefaultSubsistence()
		m.Recipes, m.RetryJobs = []string{"gather"}, spec.RetryJobs
		if spec.EnterTicks > 0 {
			m.EnterTicks = spec.EnterTicks
		}
		if spec.ExitTick > 0 {
			m.ExitTicks = spec.ExitTick
		}
		cfg.Modules.Subsistence = &m
	}
	return cfg
}

// outage is the script of a failure of the market recipe of forageWorld: its
// yield is 0 from tick 10 to tick 49 and whole again at 50
// (testdata/chains/scripts/s56.toml).
func outage(t testing.TB) []shockStep { return loadScript(t, "s56.toml") }

// forageStats is what the subsistence scenarios keep of a run of forageWorld.
type forageStats struct {
	// Unfed is the unfed household-ticks (the sum of UpkeepShortfall.Unfed),
	// Removed the households removed, Units the units output by own-use jobs,
	// Entered, Blocked and Retried the sums of the counts of OwnUseChanged, PeakOn
	// the largest number of households in the mode at the end of a tick.
	Unfed, Removed, Units, Entered, Blocked, Retried, PeakOn int64
	// Abandoned is the executions abandoned, by recipe ID and reason.
	Abandoned map[string]int64
	// Mid is the snapshot after tick midTick (the zero Snapshot when midTick is
	// negative or past the run), Final the one at the end.
	Mid, Final economysim.Snapshot
	Log        *captureHandler
}

// runForage runs cfg for ticks under the script with the audit on and keeps the
// forageStats (midTick is the step after which Mid is taken, or -1); a failure of a step or of the audit fails t with the seed.
func runForage(t testing.TB, cfg economysim.Config, seed uint64, ticks, midTick int64, script []shockStep, opts ...economysim.Option) forageStats {
	t.Helper()
	scenSem <- struct{}{}
	defer func() { <-scenSem }()
	rec, log := &recorder{}, newCaptureHandler()
	w, err := economysim.NewWorld(cfg, append([]economysim.Option{economysim.WithSeed(seed), economysim.WithObserver(rec), economysim.WithLogger(slog.New(log)), economysim.WithAudit()}, opts...)...)
	if err != nil {
		t.Fatalf("NewWorld(seed %d) error = %v, want nil", seed, err)
	}
	s0, err := w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot() error = %v, want nil", err)
	}
	st := forageStats{Abandoned: map[string]int64{}, Log: log}
	for i := range ticks {
		applyScript(t, w, s0.Names, script)
		if err := w.Step(); err != nil {
			t.Fatalf("seed %d: Step() at tick %d error = %v, want nil", seed, w.Tick(), err)
		}
		if i == midTick {
			if st.Mid, err = w.Snapshot(); err != nil {
				t.Fatalf("Snapshot() error = %v, want nil", err)
			}
		}
	}
	for _, e := range rec.events {
		switch e := e.(type) {
		case economysim.UpkeepShortfall:
			st.Unfed += e.Unfed
		case economysim.AgentBankrupted:
			st.Removed += e.Weight
		case economysim.RecipeAbandoned:
			st.Abandoned[s0.Names.Recipes[e.Recipe]+"/"+string(e.Reason)] += e.Executions
		case economysim.OwnUseChanged:
			st.Entered += e.Entered
			st.Blocked += e.Blocked
			st.Retried += e.Retried
			st.PeakOn = max(st.PeakOn, e.OnOwnUse)
			for _, a := range e.Output {
				st.Units += int64(a.Quantity)
			}
		}
	}
	if st.Final, err = w.Snapshot(); err != nil {
		t.Fatalf("Snapshot() error = %v, want nil", err)
	}
	return st
}
