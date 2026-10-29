package economysim_test

import (
	"log/slog"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// The bands of the scenarios S43 and S44 (spec-stylised-facts.md 5.11, which fixes
// their value; testdata/chains/bands/one.toml has no row for them yet and the
// bands file is not edited by this row). They are placeholders (N-027).
const (
	// s43RunwayBP is s43.runway_bp: the run-down time agrees with endowment_runway
	// within this share of it.
	s43RunwayBP = 1_000
	// s44GrantBandBP is s44.grant_band_bp: the grant of a population is within this
	// share of the mean of the ration.
	s44GrantBandBP = 2_000
	// s44DeniedTicks is s44.denied_ticks: the longest run of ticks an agent may be
	// granted nothing.
	s44DeniedTicks = 30
	// exhaustedWarnPeriod is the period of the Warn "endowment exhausted"
	// (spec-resources.md 14.2): the first tick of a run and every tenth after.
	exhaustedWarnPeriod = 10
)

// fieldWorld is the world the resource scenarios run on: one region "home" whose
// households are reapers. A reaper's recipe "reap" takes one tick, draws one unit
// of the resource "field" and makes yield units of "grain", which is also the
// upkeep of a household (1 a tick): a household eats what it reaps and works every
// tick whatever yield is, so the draw of the world is its households a tick while
// the reference programme draws its households over yield. The year is 73 ticks (a
// tick is five days), so a rate a year acts within a few hundred ticks. startGrain
// is the grain each household holds at the start: enough to outlast a run when the
// scenario must not starve, or 6 for one that must. res is given its ID and
// capacity here; the endowment starts at stock units. The configuration is viable
// (SolveSteadyState) only where the scenario says so.
func fieldWorld(res economysim.Resource, capacity, stock, households, startGrain, yield int64) economysim.Config {
	world := economysim.DefaultWorldParams()
	world.Upkeep = []economysim.Item{{Commodity: "grain", Quantity: 1}}
	world.OutputCoverTicks = 0
	grain := economysim.DefaultCommodity()
	grain.ID, grain.ReferencePrice, grain.Raw = "grain", 1_000, true
	region := economysim.DefaultRegion()
	region.ID = "home"
	role := economysim.DefaultRole()
	role.ID, role.Recipes, role.ExpectedWorkTicks = "reaper", []string{"reap"}, 70
	role.StartStock = []economysim.Item{{Commodity: "grain", Quantity: economysim.Quantity(startGrain)}}
	cal := economysim.DefaultCalendar()
	cal.DaysPerTick = 5
	res.ID, res.Capacity = "field", capacity
	return economysim.Config{
		World: world, Calendar: cal,
		Commodities: []economysim.Commodity{grain}, Roles: []economysim.Role{role}, Regions: []economysim.Region{region},
		Groups: []economysim.Group{{Region: "home", Role: "reaper", Households: households}},
		Recipes: []economysim.Recipe{{
			ID: "reap", Duration: 1, TraineeYield: 5_000,
			Outputs: []economysim.Output{{Commodity: "grain", Quantity: economysim.Quantity(yield), CostShare: 10_000}},
			Draws:   []economysim.Draw{{Resource: "field", Quantity: 1}},
		}},
		Modules: economysim.Modules{Resources: &economysim.Resources{
			HorizonYears: 50, Resources: []economysim.Resource{res},
			Endowments: []economysim.Endowment{{Region: "home", Resource: "field", HasInitial: true, Initial: stock}},
		}},
	}
}

// fieldRun is a world of fieldWorld with the observer and the logger attached and
// the audit on, stepped by the scenario.
type fieldRun struct {
	w   *economysim.World
	rec *recorder
	log *captureHandler
}

// newFieldRun builds the world of cfg for seed with a recorder, a capturing
// logger and WithAudit (so the identity of the endowment is audited every tick).
func newFieldRun(t testing.TB, cfg economysim.Config, seed uint64) *fieldRun {
	t.Helper()
	r := &fieldRun{rec: &recorder{}, log: newCaptureHandler()}
	w, err := economysim.NewWorld(cfg, economysim.WithSeed(seed), economysim.WithObserver(r.rec),
		economysim.WithLogger(slog.New(r.log)), economysim.WithAudit())
	if err != nil {
		t.Fatalf("NewWorld(seed %d) error = %v, want nil", seed, err)
	}
	r.w = w
	return r
}

// step advances the world one tick.
func (r *fieldRun) step(t testing.TB, seed uint64) {
	t.Helper()
	if err := r.w.Step(); err != nil {
		t.Fatalf("seed %d: Step() at tick %d error = %v, want nil", seed, r.w.Tick(), err)
	}
}

// stock is the stock of the one endowment of the world.
func (r *fieldRun) stock() int64 { return r.w.Endowment(0, 0) }

// changed returns the EndowmentChanged events recorded so far, in order.
func (r *fieldRun) changed() []economysim.EndowmentChanged {
	return eventsAs[economysim.EndowmentChanged](r.rec.events)
}

// warns returns the Warn lines "endowment exhausted" so far.
func (r *fieldRun) warns() int {
	n := 0
	for _, l := range r.log.records() {
		if l.level == slog.LevelWarn && l.msg == "endowment exhausted" {
			n++
		}
	}
	return n
}

// wantWarns is the number of Warn lines the rule of FR-RES-028 gives for the
// events: one for each tick whose Short is above 0 and whose run of consecutive such
// ticks has length 1 or a multiple of 10.
func wantWarns(ev []economysim.EndowmentChanged) int {
	run, n := 0, 0
	for _, e := range ev {
		if e.Short <= 0 {
			run = 0
			continue
		}
		run++
		if run == 1 || run%exhaustedWarnPeriod == 0 {
			n++
		}
	}
	return n
}

// removedStarved counts the AgentBankrupted events with the cause starved, and
// returns the first tick of one (-1 when none).
func removedStarved(events []economysim.Event) (n int, first int64) {
	first = -1
	for _, b := range eventsAs[economysim.AgentBankrupted](events) {
		if b.Cause != economysim.CauseStarved {
			continue
		}
		if n == 0 {
			first = b.Tick
		}
		n++
	}
	return n, first
}

// noEndowmentRefusals sums the households refused for want of an endowment.
func noEndowmentRefusals(events []economysim.Event) int64 {
	var n int64
	for _, e := range eventsAs[economysim.RecipeRefused](events) {
		n += e.NoEndowment
	}
	return n
}

// eventsAs returns the events of type T in order.
func eventsAs[T economysim.Event](events []economysim.Event) []T {
	var out []T
	for _, e := range events {
		if v, ok := e.(T); ok {
			out = append(out, v)
		}
	}
	return out
}
