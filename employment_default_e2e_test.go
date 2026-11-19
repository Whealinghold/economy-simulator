package economysim_test

import (
	"math/rand/v2"
	"reflect"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// The end-to-end and integration tests of the hire form with the default strategy and
// no consumer code (plan 3-emp, PR 5): the pack one-demand with overlay/employment-hire.toml,
// where the cooks are the employers and the diggers and foresters are Hireable. The hire
// life of a consumer's own strategy is in employment_hire_e2e_test.go; the arithmetic of
// the offer, the vacancy and H* is unit tested in employment_strategy_internal_test.go.

// silentStrategy is the default strategy of a consumer that does not know the hire form:
// it sees the agent without the employment group of the view (Hireable, HasEmployer and
// Employer cleared), so it offers no household, posts no vacancy and plans no hire
// (H*, spec-employment.md 9.2). It keeps no state.
type silentStrategy struct{ def economysim.DefaultStrategy }

// unaware returns a copy of the view with the employment group cleared.
func unaware(v *economysim.AgentView) *economysim.AgentView {
	c := *v
	c.Hireable, c.HasEmployer, c.Employer = false, false, economysim.EmployerView{}
	return &c
}

func (s silentStrategy) Work(v *economysim.AgentView, rng *rand.Rand) economysim.JobRequest {
	return s.def.Work(unaware(v), rng)
}

func (s silentStrategy) Trade(v *economysim.AgentView, rng *rand.Rand, p *economysim.TradePlan) {
	s.def.Trade(unaware(v), rng, p)
}

func (s silentStrategy) Revise(o *economysim.OutcomeView, rng *rand.Rand) economysim.Belief {
	return s.def.Revise(o, rng)
}

var _ economysim.Strategy = silentStrategy{}

// defaultHireIDs are the typed IDs the default-strategy tests read events by: the
// employer role and the two Hireable roles of the overlay, and the recipes of its hire
// list. The typed IDs are the positions in the slices of the configuration.
type defaultHireIDs struct {
	cook, digger, forester economysim.RoleID
	hireList               []economysim.RecipeID
}

func defaultHireIDsOf(t *testing.T, cfg economysim.Config) defaultHireIDs {
	t.Helper()
	role := func(id string) economysim.RoleID {
		for i := range cfg.Roles {
			if cfg.Roles[i].ID == id {
				return economysim.RoleID(i)
			}
		}
		t.Fatalf("the pack has no role %q", id)
		return 0
	}
	ids := defaultHireIDs{cook: role("cook"), digger: role("digger"), forester: role("forester")}
	for i := range cfg.Recipes {
		if slices.Contains(cfg.Roles[ids.cook].Employer.Recipes, cfg.Recipes[i].ID) {
			ids.hireList = append(ids.hireList, economysim.RecipeID(i))
		}
	}
	return ids
}

// defaultHireRun builds one-demand with the hire overlay, with the strategy of the
// roles left at the default (name "") unless silent is true, and the audit on, and steps
// it for ticks. With observe false no observer is attached and the tape is empty.
func defaultHireRun(t *testing.T, seed uint64, ticks int, silent, observe bool) (*economysim.World, *hireTape, defaultHireIDs) {
	t.Helper()
	cfg := loadPack(t, packOneDemandHire...)
	opts := []economysim.Option{economysim.WithSeed(seed), economysim.WithAudit()}
	if silent {
		for i := range cfg.Roles {
			cfg.Roles[i].Strategy = "silent"
		}
		opts = append(opts, economysim.WithStrategy("silent", silentStrategy{}))
	}
	tape := &hireTape{}
	if observe {
		opts = append(opts, economysim.WithObserver(tape))
	}
	w, err := economysim.NewWorld(cfg, opts...)
	if err != nil {
		t.Fatalf("NewWorld(one-demand + hire overlay, seed %d) = %v, want nil", seed, err)
	}
	for range ticks {
		if err := w.Step(); err != nil {
			t.Fatalf("Step() at tick %d (seed %d) = %v, want nil", w.Tick(), seed, err)
		}
	}
	return w, tape, defaultHireIDsOf(t, cfg)
}

// TestDefaultHire_LifeOnThePack verifies, end to end through the public API and with
// the default strategy alone, the life of a hire: the default's offers (a Hireable
// role's households at its reservation) and vacancies (an employer's positions at the
// bid of its best hire-list recipe) meet and form hires that end Completed with the
// bill back whole, in the order the checker of employment_hire_e2e_test.go fixes. On
// top of that it checks what only the default decides: the employer is a cook, the
// worker a digger or a forester, the recipe is on the cook's hire list, the bid is at
// least 1 (a vacancy below 1 is not posted), the reservation is not negative, and the
// households an employer hires in a tick do not exceed its MaxHires of 4
// (overlay/employment-hire.toml; the jobs of the pack are one tick, so none is open
// from an earlier tick).
//
// Requirement: FR-EMP-013, FR-EMP-015, FR-EMP-016, FR-EMP-021, FR-EMP-024, FR-EMP-029, FR-EMP-036
func TestDefaultHire_LifeOnThePack(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	for _, seed := range []uint64{1, 2, 3} {
		_, tape, ids := defaultHireRun(t, seed, 25, false, true)
		c := newHireChecker()
		hiredBy := make(map[[2]int64]int64) // (tick, employer) -> households hired
		for i, e := range tape.events {
			hireStep(c, i, e)
			if h, ok := e.(economysim.HireFormed); ok {
				checkDefaultHire(t, seed, h, ids)
				hiredBy[[2]int64{h.Tick, int64(h.Employer)}] += h.Households
			}
		}
		if len(c.formed) == 0 {
			t.Fatalf("seed %d: no hire formed in 25 ticks, want some: the default offers and posts", seed)
		}
		if len(c.ended) != len(c.formed) {
			t.Errorf("seed %d: %d hires formed, %d ended, want equal at the end of a run of one-tick jobs", seed, len(c.formed), len(c.ended))
		}
		for _, p := range c.problems {
			t.Errorf("seed %d: %s", seed, p)
		}
		for k, n := range hiredBy {
			if n > defaultMaxHires {
				t.Errorf("seed %d: employer %d hired %d households in tick %d, want at most MaxHires %d", seed, k[1], n, k[0], defaultMaxHires)
			}
		}
	}
}

// defaultMaxHires is max_hires of the cooks in overlay/employment-hire.toml.
const defaultMaxHires = 4

// checkDefaultHire checks what only the default strategy decides about one formation.
func checkDefaultHire(t *testing.T, seed uint64, h economysim.HireFormed, ids defaultHireIDs) {
	t.Helper()
	if h.EmployerRole != ids.cook {
		t.Errorf("seed %d: hire %d EmployerRole = %d, want the cook %d", seed, h.Hire, h.EmployerRole, ids.cook)
	}
	if h.WorkerRole != ids.digger && h.WorkerRole != ids.forester {
		t.Errorf("seed %d: hire %d WorkerRole = %d, want the digger %d or the forester %d", seed, h.Hire, h.WorkerRole, ids.digger, ids.forester)
	}
	if !slices.Contains(ids.hireList, h.Recipe) {
		t.Errorf("seed %d: hire %d Recipe = %d, want one of the hire list %v", seed, h.Hire, h.Recipe, ids.hireList)
	}
	if h.Bid < 1 || h.Reserve < 0 || h.Households < 1 || h.Ticks < 1 {
		t.Errorf("seed %d: hire %d (Bid, Reserve, Households, Ticks) = (%d, %d, %d, %d), want Bid >= 1, Reserve >= 0, Households >= 1, Ticks >= 1",
			seed, h.Hire, h.Bid, h.Reserve, h.Households, h.Ticks)
	}
}

// labourCell is what the hire events of one region and tick add up to, to be set
// against the LabourRound of that region and tick.
type labourCell struct {
	formed, households, householdTicks, paid int64
}

// TestDefaultHire_LabourRoundAgreesWithTheHireEvents verifies, over whole runs of the
// default strategy, that the aggregate LabourRound (spec-employment.md 15.1) says what
// the per-hire events say: Matched is the hires formed in the region and tick, Hired and
// Filled their households, Offered is Hired plus NoVacancy, Posted is Filled plus
// Unfilled, HouseholdTicks the households times the ticks of the one-tick jobs, Paid
// the wages the HireEnded events of the tick paid; the rows by role sum to the totals,
// stand in ascending RoleID, and name only the cooks as employers and the diggers and
// foresters as workers; and a region and tick with a hire has a round while a round
// appears at most once per region and tick.
//
// Requirement: FR-EMP-053, FR-EMP-060
func TestDefaultHire_LabourRoundAgreesWithTheHireEvents(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	for _, seed := range []uint64{1, 2, 3} {
		_, tape, ids := defaultHireRun(t, seed, 25, false, true)
		cells := hireCells(tape.events)
		seen := make(map[[2]int64]bool)
		for _, e := range tape.events {
			if r, ok := e.(economysim.LabourRound); ok {
				k := [2]int64{r.Tick, int64(r.Region)}
				if seen[k] {
					t.Errorf("seed %d: two LabourRound events for region %d in tick %d, want one", seed, r.Region, r.Tick)
				}
				seen[k] = true
				checkRound(t, seed, r, cells[k], ids)
			}
		}
		for k, c := range cells {
			if c.formed > 0 && !seen[k] {
				t.Errorf("seed %d: tick %d region %d formed %d hires and has no LabourRound, want one", seed, k[0], k[1], c.formed)
			}
		}
		if len(seen) == 0 {
			t.Fatalf("seed %d: no LabourRound in 25 ticks, want some", seed)
		}
	}
}

// hireCells adds up the hire events of a run by (tick, region).
func hireCells(events []economysim.Event) map[[2]int64]labourCell {
	cells := make(map[[2]int64]labourCell)
	for _, e := range events {
		switch e := e.(type) {
		case economysim.HireFormed:
			k := [2]int64{e.Tick, int64(e.Region)}
			c := cells[k]
			c.formed++
			c.households += e.Households
			c.householdTicks += e.Households * e.Ticks
			cells[k] = c
		case economysim.HireEnded:
			k := [2]int64{e.Tick, int64(e.Region)}
			c := cells[k]
			c.paid += int64(e.Paid)
			cells[k] = c
		}
	}
	return cells
}

// checkRound sets one LabourRound against the hire events of its region and tick.
func checkRound(t *testing.T, seed uint64, r economysim.LabourRound, want labourCell, ids defaultHireIDs) {
	t.Helper()
	if got := (labourCell{r.Matched, r.Hired, r.HouseholdTicks, int64(r.Paid)}); got != want {
		t.Errorf("seed %d: tick %d region %d LabourRound (Matched, Hired, HouseholdTicks, Paid) = %+v, want %+v from the hire events",
			seed, r.Tick, r.Region, got, want)
	}
	if r.Filled != r.Hired || r.Offered != r.Hired+r.NoVacancy || r.Posted != r.Filled+r.Unfilled {
		t.Errorf("seed %d: tick %d LabourRound Offered %d Hired %d NoVacancy %d Posted %d Filled %d Unfilled %d, want Filled = Hired, Offered = Hired + NoVacancy, Posted = Filled + Unfilled",
			seed, r.Tick, r.Offered, r.Hired, r.NoVacancy, r.Posted, r.Filled, r.Unfilled)
	}
	checkWorkerRows(t, seed, r, ids)
	checkEmployerRows(t, seed, r, ids)
}

// checkWorkerRows checks the rows by worker role of one LabourRound: ascending RoleID,
// the Hireable roles, summing to the offered, hired and unmatched households.
func checkWorkerRows(t *testing.T, seed uint64, r economysim.LabourRound, ids defaultHireIDs) {
	t.Helper()
	var off, hired, none int64
	for i, w := range r.Workers {
		if w.Role != ids.digger && w.Role != ids.forester {
			t.Errorf("seed %d: tick %d Workers row of role %d, want the digger %d or the forester %d", seed, r.Tick, w.Role, ids.digger, ids.forester)
		}
		if i > 0 && w.Role <= r.Workers[i-1].Role {
			t.Errorf("seed %d: tick %d Workers rows not in ascending RoleID: %d after %d", seed, r.Tick, w.Role, r.Workers[i-1].Role)
		}
		off, hired, none = off+w.Offered, hired+w.Hired, none+w.NoVacancy
	}
	if off != r.Offered || hired != r.Hired || none != r.NoVacancy {
		t.Errorf("seed %d: tick %d Workers rows sum to (offered, hired, no vacancy) = (%d, %d, %d), want (%d, %d, %d)",
			seed, r.Tick, off, hired, none, r.Offered, r.Hired, r.NoVacancy)
	}
}

// checkEmployerRows checks the rows by employer role of one LabourRound: the cooks,
// summing to the positions posted, filled and unfilled.
func checkEmployerRows(t *testing.T, seed uint64, r economysim.LabourRound, ids defaultHireIDs) {
	t.Helper()
	var posted, filled, unfilled int64
	for _, v := range r.Employers {
		if v.Role != ids.cook {
			t.Errorf("seed %d: tick %d Employers row of role %d, want the cook %d", seed, r.Tick, v.Role, ids.cook)
		}
		posted, filled, unfilled = posted+v.Posted, filled+v.Filled, unfilled+v.Unfilled
	}
	if posted != r.Posted || filled != r.Filled || unfilled != r.Unfilled {
		t.Errorf("seed %d: tick %d Employers rows sum to (posted, filled, unfilled) = (%d, %d, %d), want (%d, %d, %d)",
			seed, r.Tick, posted, filled, unfilled, r.Posted, r.Filled, r.Unfilled)
	}
}

// marketThrough returns the orders, trades and clearings of the ticks up to and including
// last, with the cost anchor of a clearing zeroed: the one field the module moves with no
// hire (the pooled wage of the cost anchor leaves the employer roles out, spec-employment.md
// 8.4.1, FR-EMP-047).
func marketThrough(tape *hireTape, last int64) []economysim.Event {
	var out []economysim.Event
	for _, e := range tape.events {
		switch e := e.(type) {
		case economysim.OrderPlaced:
			if e.Tick <= last {
				out = append(out, e)
			}
		case economysim.TradeExecuted:
			if e.Tick <= last {
				out = append(out, e)
			}
		case economysim.MarketCleared:
			if e.Tick <= last {
				e.CostAnchor = 0
				out = append(out, e)
			}
		}
	}
	return out
}

// TestEmploymentOn_StrategyThatOffersNothingFormsNoHire verifies that the module in
// its hire form leaves the labour market empty and the household sector as it was for a
// world whose strategy offers no household and posts no vacancy (spec-employment.md 4.1,
// FR-EMP-009 and FR-EMP-010: an empty offer and an empty hire are no request): no
// HireFormed, HireEnded, LabourRound or PayrollPaid is emitted, no hire is open and no
// escrow is held at the end, and the orders, trades and clearings of the first two ticks
// equal those of the same strategy on the pack with the module off, but for the cost
// anchor, which moves with the module on because the pooled wage leaves the employer
// roles out (FR-EMP-047).
//
// Requirement: FR-EMP-009, FR-EMP-010, FR-EMP-047, FR-EMP-064
//
// Guard: the view group, the hired-tick welfare rules and the planned hires H* are read
// for every agent of a world with the module on; this pins that with no offer and no
// vacancy none of them reaches the market, and that the one effect of the module that
// does not need a hire is the documented one.
func TestEmploymentOn_StrategyThatOffersNothingFormsNoHire(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	const ticks = 15
	for _, seed := range []uint64{1, 2} {
		w, on, _ := defaultHireRun(t, seed, ticks, true, true)
		for _, e := range on.events {
			switch e.(type) {
			case economysim.HireFormed, economysim.HireEnded, economysim.LabourRound, economysim.PayrollPaid:
				t.Fatalf("seed %d: event %T from a strategy that offers nothing, want none", seed, e)
			}
		}
		checkNoHireHeld(t, seed, w)
		a, b := marketThrough(on, 1), marketThrough(moduleOffRun(t, seed, ticks), 1)
		if len(a) == 0 || !reflect.DeepEqual(a, b) {
			i := 0
			for i < min(len(a), len(b)) && reflect.DeepEqual(a[i], b[i]) {
				i++
			}
			t.Errorf("seed %d: market of ticks 0 and 1 with the module on and unused (%d events) differs from the module off (%d), first at %d, want equal and non-empty",
				seed, len(a), len(b), i)
		}
	}
}

// checkNoHireHeld checks that the world holds no hire and no escrow.
func checkNoHireHeld(t *testing.T, seed uint64, w *economysim.World) {
	t.Helper()
	sn, err := w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot() = %v, want nil", err)
	}
	if len(sn.Hires) != 0 {
		t.Errorf("seed %d: Snapshot().Hires = %d rows, want none", seed, len(sn.Hires))
	}
	for _, r := range sn.Regions {
		if r.MoneyContract != 0 {
			t.Errorf("seed %d: region %d MoneyContract = %d, want 0 with no hire", seed, r.Region, r.MoneyContract)
		}
	}
}

// moduleOffRun runs the silent strategy on one-demand without the employment module.
func moduleOffRun(t *testing.T, seed uint64, ticks int) *hireTape {
	t.Helper()
	cfg := loadPack(t, packOneDemand...)
	for i := range cfg.Roles {
		cfg.Roles[i].Strategy = "silent"
	}
	var tape hireTape
	w, err := economysim.NewWorld(cfg, economysim.WithSeed(seed), economysim.WithAudit(),
		economysim.WithStrategy("silent", silentStrategy{}), economysim.WithObserver(&tape))
	if err != nil {
		t.Fatalf("NewWorld(one-demand, module off) = %v, want nil", err)
	}
	for range ticks {
		if err := w.Step(); err != nil {
			t.Fatalf("Step() = %v, want nil", err)
		}
	}
	return &tape
}

// TestDefaultHire_DeterministicOverRunsAndSeeds verifies that the default strategy's
// labour market is deterministic (architecture.md 7.4): for each seed, two runs give the
// same events from the first tick to the last, the labour events included, and
// attaching no observer gives the same snapshot; and the seed matters, since the hire
// events of the seeds are not all the same.
//
// Requirement: FR-EMP-018, FR-EMP-061, FR-EMP-075
func TestDefaultHire_DeterministicOverRunsAndSeeds(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	labour := func(tape *hireTape) []economysim.Event {
		return slices.DeleteFunc(slices.Clone(tape.events), func(e economysim.Event) bool {
			switch e.(type) {
			case economysim.HireFormed, economysim.HireEnded, economysim.LabourRound:
				return false
			}
			return true
		})
	}
	var streams [][]economysim.Event
	for _, seed := range []uint64{1, 2, 3, 4} {
		wa, a, _ := defaultHireRun(t, seed, 20, false, true)
		_, b, _ := defaultHireRun(t, seed, 20, false, true)
		if !reflect.DeepEqual(a.events, b.events) {
			t.Errorf("seed %d: two runs gave different events (%d and %d), want identical", seed, len(a.events), len(b.events))
		}
		la := labour(a)
		if len(la) == 0 {
			t.Errorf("seed %d: no labour event in 20 ticks, want some", seed)
		}
		streams = append(streams, la)

		wn, _, _ := defaultHireRun(t, seed, 20, false, false)
		sa, err := wa.Snapshot()
		if err != nil {
			t.Fatalf("Snapshot() = %v, want nil", err)
		}
		sn, err := wn.Snapshot()
		if err != nil {
			t.Fatalf("Snapshot() = %v, want nil", err)
		}
		if !reflect.DeepEqual(sa, sn) {
			t.Errorf("seed %d: Snapshot() with an observer differs from the one without, want equal", seed)
		}
	}
	distinct := false
	for _, s := range streams[1:] {
		distinct = distinct || !reflect.DeepEqual(s, streams[0])
	}
	if !distinct {
		t.Errorf("the labour events of seeds 1 to 4 are all the same, want the seed to matter")
	}
}
