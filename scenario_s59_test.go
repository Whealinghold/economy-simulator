package economysim_test

import (
	"fmt"
	"log/slog"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// s59 sizes (the figures of S44, spec-resources.md 19.4): a refill stock of capacity
// 20 000 growing 1 000% a year, a year of 73 ticks, blighted to a capacity of 3 650
// before the first tick, so an empty stock regrows 500 a tick. A thousand
// households, as 1 000 agents of weight 1 or as 20 cohorts of weight 50, each in the
// own-use mode and each asking for one unit a tick (an execution of "gather" draws
// one): half of the demand is served.
const (
	s59Households = 1_000
	s59Ticks      = 80
	s59Warmup     = 10
	s59Cohort     = 50
)

// s59World is the forage world with the resources module on and the own-use recipe
// drawing the commons "field". Each household holds 500 grain, 250 ticks of upkeep,
// and EnterTicks 300 and ExitTicks 1 000 keep it in the mode for the whole run, so the
// demand is a constant 1 000 requests a tick whatever the ration grants (a household
// that was fed would leave the mode and the demand of the units and of the cohorts
// would part). The market recipe draws nothing: the commons is drawn by an own-use
// recipe alone.
func s59World(scale int64) economysim.Config {
	cfg := forageWorld(forageSpec{Households: s59Households, StartGrain: 500, OwnUseBP: 10_000, Duration: 1, Module: true, RetryJobs: 10, EnterTicks: 300, ExitTick: 1_000})
	res := economysim.DefaultResource()
	res.Shape, res.RateBP = economysim.ShapeRefill, 100_000
	res.ID, res.Capacity = "field", 20_000
	cfg.Calendar.DaysPerTick = 5
	cfg.Roles[0].ExpectedWorkTicks = 70 // a year is 73 ticks
	cfg.Recipes[0].Draws = []economysim.Draw{{Resource: "field", Quantity: 1}}
	cfg.Modules.Resources = &economysim.Resources{
		HorizonYears: 50, Resources: []economysim.Resource{res},
		Endowments: []economysim.Endowment{{Region: "home", Resource: "field", HasInitial: true, Initial: 20_000}},
	}
	cfg.World.Scale, cfg.World.MinAgents = scale, s59Households/scale
	return cfg
}

// s59Result is what S59 keeps of one run.
type s59Result struct {
	requested, granted, blocked, started int64
	// short, denied and reduced are the sums over the window of the units asked and not
	// granted, the agents granted nothing and the agents granted less than they asked.
	short, denied, reduced int64
	// leaked counts the ticks of the window in which the units granted and the units
	// short did not add up to the units asked.
	leaked       int64
	warnLines    int
	snapshotRows int
}

// runS59 runs the world with the commons blighted at tick 0 and keeps the units
// requested and granted after the warm-up, the households blocked and the
// executions started by own-use jobs, and the Warn lines of the module.
func runS59(t testing.TB, seed uint64, scale int64) s59Result {
	t.Helper()
	cfg := s59World(scale)
	rec, log := &recorder{}, newCaptureHandler()
	w, err := economysim.NewWorld(cfg, economysim.WithSeed(seed), economysim.WithObserver(rec), economysim.WithLogger(slog.New(log)), economysim.WithAudit())
	if err != nil {
		t.Fatalf("NewWorld(seed %d, scale %d) error = %v, want nil", seed, scale, err)
	}
	economysim.ShockCapacity(w, 0, 0, 1_825)
	for range s59Ticks {
		if err := w.Step(); err != nil {
			t.Fatalf("seed %d, scale %d: Step() at tick %d error = %v, want nil", seed, scale, w.Tick(), err)
		}
	}
	var r s59Result
	for _, e := range rec.events {
		switch e := e.(type) {
		case economysim.EndowmentChanged:
			if e.Tick >= s59Warmup {
				r.requested += e.Requested
				r.granted += e.Granted
				r.short += e.Short
				r.denied += e.AgentsDenied
				r.reduced += e.AgentsReduced
				if e.Granted+e.Short != e.Requested {
					r.leaked++
				}
			}
		case economysim.OwnUseChanged:
			if e.Tick >= s59Warmup {
				r.blocked += e.Blocked
				r.started += e.Started
			}
		}
	}
	for _, l := range log.records() {
		if l.level == slog.LevelWarn && (l.msg == "own use blocked" || l.msg == "role on own use") {
			r.warnLines++
		}
	}
	s, err := w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot() error = %v, want nil", err)
	}
	r.snapshotRows = len(s.Endowments)
	return r
}

// TestScenarioS59_ScarceCommons runs scenario S59 (SUB-S4, plan 2-10 D417) on the
// forage world with the resources module: a thousand households as 1 000 agents of
// weight 1 and as 20 cohorts of weight 50 ask a blighted refill commons, which only an
// own-use recipe draws, for twice what it regrows, seeds 1 to scenarioSeedCount(),
// with the audit on. It asserts that the ration holds (every tick the units
// granted and the units short add up to the units asked, and units are short), that
// the units are blocked at the start step (Blocked and the agents denied are
// non-zero) while a cohort is reduced and never denied (it is granted at least
// floor(w * A / U), FR-RES-021), that the Warn lines of the module are bounded by one
// per period and run (at most 1 + ticks / 10 of each of the two Warns), that the
// configuration raises no endowment_unused warning for a commons only an own-use
// recipe draws and that the snapshot has the endowment (D413). It reports the
// shares of the units asked that were granted, which differ between the units and
// the cohorts (open item of the report: the cohorts leave and enter the mode
// together, so their demand is not the units').
//
// Arithmetic: a stock of 3 650 regrows 13.7% of the gap a tick, 500 units from empty,
// against about 1 000 requests of 1 unit: a share of about 5 000 bp for the units.
//
// Requirement: FR-SUB-023, FR-SUB-036, FR-SUB-042, FR-SUB-050
//
// Guard: an own-use request that the ration ignores, or one the ration serves in
// the order of agent IDs, would give units and cohorts different shares.
func TestScenarioS59_ScarceCommons(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end scenario, skipped with -short")
	}
	const scenario, pack = "S59", "forage+field"
	for _, wn := range s59World(1).Warnings() {
		if wn.Rule == "endowment_unused" {
			t.Errorf("Warnings() holds %+v, want no endowment_unused for a commons an own-use recipe draws (D413)", wn)
		}
	}
	rep := newPackScenReport("subsistence", fmt.Sprintf("S59 forage+field seeds 1-%d, %d ticks", scenarioSeedCount(), s59Ticks))
	t.Run("seeds", func(t *testing.T) {
		for _, seed := range seedList(scenarioSeedCount()) {
			t.Run(fmt.Sprintf("seed %d", seed), func(t *testing.T) {
				t.Parallel()
				units, cohorts := runS59(t, seed, 1), runS59(t, seed, s59Cohort)
				us, cs := units.granted*10_000/max(units.requested, 1), cohorts.granted*10_000/max(cohorts.requested, 1)
				rep.num(seed, "units: share granted, bp", us)
				rep.num(seed, "cohorts: share granted, bp", cs)
				rep.num(seed, "units: households blocked", units.blocked)
				rep.num(seed, "units: agents denied", units.denied)
				rep.num(seed, "cohorts: agents denied", cohorts.denied)
				rep.num(seed, "cohorts: agents reduced", cohorts.reduced)
				rep.num(seed, "units: Warn lines", int64(units.warnLines))
				rep.num(seed, "cohorts: Warn lines", int64(cohorts.warnLines))
				checkExact(t, rep, scenario, pack, seed, "the ration holds", units.leaked == 0 && cohorts.leaked == 0 && units.short > 0 && cohorts.short > 0,
					"ticks with units unaccounted: units %d, cohorts %d; units short: %d and %d", units.leaked, cohorts.leaked, units.short, cohorts.short)
				checkExact(t, rep, scenario, pack, seed, "the units are blocked at the start", units.blocked > 0 && units.denied > 0,
					"units blocked %d, denied %d", units.blocked, units.denied)
				checkExact(t, rep, scenario, pack, seed, "a cohort is reduced, never denied", cohorts.denied == 0 && cohorts.reduced > 0,
					"cohorts denied %d, reduced %d (each gets at least floor(w * A / U), FR-RES-021)", cohorts.denied, cohorts.reduced)
				bound := 2 * (1 + s59Ticks/10)
				checkExact(t, rep, scenario, pack, seed, "the Warns are bounded", units.warnLines <= bound && cohorts.warnLines <= bound,
					"units %d, cohorts %d, bound %d", units.warnLines, cohorts.warnLines, bound)
				checkExact(t, rep, scenario, pack, seed, "the snapshot has the endowment", units.snapshotRows == 1 && cohorts.snapshotRows == 1,
					"units %d rows, cohorts %d rows", units.snapshotRows, cohorts.snapshotRows)
			})
		}
	})
	rep.flush(t)
}
