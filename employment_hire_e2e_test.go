package economysim_test

import (
	"fmt"
	"math/rand/v2"
	"reflect"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// hirePackStrategy wraps the default strategy and adds the labour market the pack
// overlay configures: a cook posts one vacancy of the recipe on its hire list at a
// fixed bid, and a digger or forester offers one household at a low reserve. It
// is a extension of a consumer, written through the public API only, and keeps no
// state.
type hirePackStrategy struct {
	def                  economysim.DefaultStrategy
	cook, digger, forest economysim.RoleID
	recipe               economysim.RecipeID
	bid, reserve         economysim.Price
}

func (s hirePackStrategy) Work(v *economysim.AgentView, rng *rand.Rand) economysim.JobRequest {
	r := s.def.Work(v, rng)
	switch v.Role {
	case s.cook:
		r.Hire = economysim.HireRequest{Recipe: s.recipe, Positions: 1, Bid: s.bid}
	case s.digger, s.forest:
		r.Offer = economysim.LabourOffer{Households: 1, Reserve: s.reserve}
	}
	return r
}

func (s hirePackStrategy) Trade(v *economysim.AgentView, rng *rand.Rand, p *economysim.TradePlan) {
	s.def.Trade(v, rng, p)
}

func (s hirePackStrategy) Revise(o *economysim.OutcomeView, rng *rand.Rand) economysim.Belief {
	return s.def.Revise(o, rng)
}

var _ economysim.Strategy = hirePackStrategy{}

// hireTape records every event of a run in order.
type hireTape struct{ events []economysim.Event }

func (h *hireTape) Observe(e economysim.Event) { h.events = append(h.events, e) }

// hirePackStrategyFor resolves the roles and the recipe of the pack by their IDs.
// The typed IDs are the positions in the slices of the configuration.
func hirePackStrategyFor(t *testing.T, cfg economysim.Config) hirePackStrategy {
	t.Helper()
	idx := func(ids []string, id string) int {
		i := slices.Index(ids, id)
		if i < 0 {
			t.Fatalf("pack has no %q among %v", id, ids)
		}
		return i
	}
	var roles, recipes []string
	for _, r := range cfg.Roles {
		roles = append(roles, r.ID)
	}
	for _, r := range cfg.Recipes {
		recipes = append(recipes, r.ID)
	}
	return hirePackStrategy{
		cook:    economysim.RoleID(idx(roles, "cook")),
		digger:  economysim.RoleID(idx(roles, "digger")),
		forest:  economysim.RoleID(idx(roles, "forester")),
		recipe:  economysim.RecipeID(idx(recipes, "prepare")),
		bid:     6,
		reserve: 1,
	}
}

// hirePackWorld builds the pack one-demand with the hire overlay and a consumer
// strategy that posts vacancies and offers, with the audit on. A nil tape means
// no observer.
func hirePackWorld(t *testing.T, seed uint64, tape *hireTape) *economysim.World {
	t.Helper()
	cfg := loadPack(t, packOneDemandHire...)
	s := hirePackStrategyFor(t, cfg)
	for i := range cfg.Roles {
		switch cfg.Roles[i].ID {
		case "cook", "digger", "forester":
			cfg.Roles[i].Strategy = "hire_pack"
		}
	}
	opts := []economysim.Option{economysim.WithSeed(seed), economysim.WithAudit(), economysim.WithStrategy("hire_pack", s)}
	if tape != nil {
		opts = append(opts, economysim.WithObserver(tape))
	}
	w, err := economysim.NewWorld(cfg, opts...)
	if err != nil {
		t.Fatalf("NewWorld(one-demand + hire overlay) = %v, want nil", err)
	}
	return w
}

// hirePackRun steps a hire-pack world for ticks, with the audit on, and returns
// the tape and, for each step, the money of every sector of the world.
func hirePackRun(t *testing.T, seed uint64, ticks int) (*hireTape, []hireSnap) {
	t.Helper()
	var tape hireTape
	w := hirePackWorld(t, seed, &tape)
	var snaps []hireSnap
	for range ticks {
		if err := w.Step(); err != nil {
			t.Fatalf("Step() at tick %d (seed %d) = %v, want nil", w.Tick(), seed, err)
		}
		sn, err := w.Snapshot()
		if err != nil {
			t.Fatalf("Snapshot() at tick %d (seed %d) = %v, want nil", w.Tick(), seed, err)
		}
		hs := hireSnap{tick: int64(w.Tick()) - 1}
		for _, r := range sn.Regions {
			hs.total += r.MoneyHouseholds + r.MoneyPool + r.MoneyContract + r.MoneyTreasury
			hs.escrow += r.MoneyContract
		}
		snaps = append(snaps, hs)
	}
	return &tape, snaps
}

// hireSnap is the money of a world at the end of a step.
type hireSnap struct {
	tick          int64
	total, escrow economysim.Money
}

// TestHirePack_HiresFormAndEndOnThePack verifies, end to end on the pack
// one-demand with overlay/employment-hire.toml and a consumer strategy that posts
// vacancies and offers, that hires form and end through the public API: every
// HireFormed is followed by exactly one HireEnded of the same hire, employer,
// worker and recipe; the bill escrowed is wage times ticks times households; the
// wages paid plus the refund equal that bill; HireFormed comes after TickStarted
// and before the first OrderPlaced (P2.5a is before the trade phase); and
// the formations of a tick are in ascending hire ID.
//
// Pack one-demand has one-tick recipes, so a hire forms, pays at P2.7 and
// completes in one tick: the shortest whole life. The multi-tick life is a unit
// test of the hand-built world (TestHire_FormsPaysAndCompletes).
//
// Requirement: FR-EMP-008, FR-EMP-024, FR-EMP-029, FR-EMP-030, FR-EMP-036, FR-EMP-051, FR-EMP-052
func TestHirePack_HiresFormAndEndOnThePack(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	for _, seed := range []uint64{1, 2, 3} {
		tape, _ := hirePackRun(t, seed, 25)
		c := newHireChecker()
		for i, e := range tape.events {
			hireStep(c, i, e)
		}
		if len(c.formed) == 0 {
			t.Fatalf("seed %d: no hire formed in 25 ticks, want some (the cooks of the pack post, the hands offer)", seed)
		}
		if len(c.ended) != len(c.formed) {
			t.Errorf("seed %d: %d hires formed, %d ended, want equal at the end of a run of one-tick jobs", seed, len(c.formed), len(c.ended))
		}
		for _, p := range c.problems {
			t.Errorf("seed %d: %s", seed, p)
		}
	}
}

// hireChecker walks the events of a run in order and collects what breaks the
// life of a hire, as text.
type hireChecker struct {
	formed      map[economysim.HireID]economysim.HireFormed
	ended       map[economysim.HireID]economysim.HireEnded
	tick        int64
	orderPlaced bool
	lastFormed  economysim.HireID
	problems    []string
}

func newHireChecker() *hireChecker {
	return &hireChecker{
		formed: make(map[economysim.HireID]economysim.HireFormed),
		ended:  make(map[economysim.HireID]economysim.HireEnded),
	}
}

func hireBad(c *hireChecker, format string, args ...any) {
	c.problems = append(c.problems, fmt.Sprintf(format, args...))
}

func hireStep(c *hireChecker, i int, e economysim.Event) {
	switch e := e.(type) {
	case economysim.TickStarted:
		c.tick, c.orderPlaced, c.lastFormed = e.Tick, false, -1
	case economysim.OrderPlaced:
		c.orderPlaced = true
	case economysim.HireFormed:
		onFormed(c, i, e)
	case economysim.HireEnded:
		onEnded(c, e)
	}
}

func onFormed(c *hireChecker, i int, e economysim.HireFormed) {
	if e.Tick != c.tick || c.orderPlaced {
		hireBad(c, "event %d HireFormed of tick %d outside P2.5a of tick %d (an order was placed first: %v)", i, e.Tick, c.tick, c.orderPlaced)
	}
	if e.Hire <= c.lastFormed {
		hireBad(c, "HireFormed hire %d after hire %d in tick %d, want ascending formation order", e.Hire, c.lastFormed, e.Tick)
	}
	c.lastFormed = e.Hire
	if _, dup := c.formed[e.Hire]; dup {
		hireBad(c, "hire %d formed twice", e.Hire)
	}
	checkBill(c, e)
	c.formed[e.Hire] = e
}

func onEnded(c *hireChecker, e economysim.HireEnded) {
	f, ok := c.formed[e.Hire]
	if !ok {
		hireBad(c, "HireEnded for hire %d before any HireFormed", e.Hire)
		return
	}
	if _, dup := c.ended[e.Hire]; dup {
		hireBad(c, "hire %d ended twice", e.Hire)
	}
	c.ended[e.Hire] = e
	checkEnd(c, e, f)
}

// checkBill checks the money of a formation: the bill is the wage for every
// household-tick, and the wage lies between the two limits that fixed it.
func checkBill(c *hireChecker, e economysim.HireFormed) {
	if want := economysim.Money(int64(e.Wage) * e.Ticks * e.Households); e.Escrowed != want {
		hireBad(c, "hire %d Escrowed = %d, want Wage*Ticks*Households = %d", e.Hire, e.Escrowed, want)
	}
	if lo, hi := economysim.Money(e.Reserve), economysim.Money(e.Bid); !(lo <= e.Wage && e.Wage <= hi) {
		hireBad(c, "hire %d Wage = %d outside Reserve %d to Bid %d", e.Hire, e.Wage, e.Reserve, e.Bid)
	}
}

// checkEnd compares a end of a hire with its formation.
func checkEnd(c *hireChecker, e economysim.HireEnded, f economysim.HireFormed) {
	if e.Employer != f.Employer || e.Worker != f.Worker || e.Recipe != f.Recipe || e.Households != f.Households {
		hireBad(c, "hire %d ended as %+v, formed as %+v, want the same employer, worker, recipe and households", e.Hire, e, f)
	}
	if e.Cause != economysim.CauseCompleted {
		hireBad(c, "hire %d Cause = %q, want %q", e.Hire, e.Cause, economysim.CauseCompleted)
	}
	if want := economysim.Money(int64(f.Wage) * e.Worked * e.Households); e.Paid != want {
		hireBad(c, "hire %d Paid = %d, want Wage*Worked*Households = %d", e.Hire, e.Paid, want)
	}
	if e.Paid+e.Refunded != f.Escrowed {
		hireBad(c, "hire %d Paid %d + Refunded %d != Escrowed %d, want the bill back whole", e.Hire, e.Paid, e.Refunded, f.Escrowed)
	}
}

// TestHirePack_MoneyIsConservedAndEscrowEmptiesEachTick verifies the ledger side
// of a hire on the pack: with the audit on, the money of all sectors (households,
// pool, contract escrow, treasury) changes in a tick only by the faucets and sinks
// of that tick BoundaryFlow, so the wage escrow, pay and refund (internal
// reasons, FR-EMP-039) create and destroy nothing; and, the jobs being one tick,
// no money is left in an Escrow account at the end of a step.
//
// Requirement: FR-EMP-036, FR-EMP-039, FR-EMP-040, FR-EMP-041
func TestHirePack_MoneyIsConservedAndEscrowEmptiesEachTick(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	for _, seed := range []uint64{1, 2} {
		tape, snaps := hirePackRun(t, seed, 25)
		net := boundaryNet(tape.events)
		for i, s := range snaps {
			if s.escrow != 0 {
				t.Errorf("seed %d: tick %d MoneyContract = %d at the end of the step, want 0 (one-tick hires)", seed, s.tick, s.escrow)
			}
			if i == 0 {
				continue
			}
			if got, want := s.total-snaps[i-1].total, net[s.tick]; got != want {
				t.Errorf("seed %d: tick %d total money moved by %d, BoundaryFlow net is %d, want equal", seed, s.tick, got, want)
			}
		}
	}
}

// boundaryNet is the money the faucets put in less the sinks took out, by tick.
func boundaryNet(events []economysim.Event) map[int64]economysim.Money {
	net := make(map[int64]economysim.Money)
	for _, e := range events {
		b, ok := e.(economysim.BoundaryFlow)
		if !ok || b.Commodity != economysim.NoCommodity {
			continue
		}
		if b.Kind == economysim.KindFaucet {
			net[b.Tick] += economysim.Money(b.Total)
		} else {
			net[b.Tick] -= economysim.Money(b.Total)
		}
	}
	return net
}

// TestHirePack_SameSeedGivesTheSameHiresWithAndWithoutObserver verifies that the
// hire events and the final snapshot of a hire-pack run are identical for the same
// seed on two runs, and that attaching the observer does not change the snapshot.
//
// Requirement: FR-EMP-018, FR-EMP-061
func TestHirePack_SameSeedGivesTheSameHiresWithAndWithoutObserver(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	hires := func(tape *hireTape) []economysim.Event {
		return slices.DeleteFunc(slices.Clone(tape.events), func(e economysim.Event) bool {
			switch e.(type) {
			case economysim.HireFormed, economysim.HireEnded:
				return false
			}
			return true
		})
	}
	a, _ := hirePackRun(t, 7, 20)
	b, _ := hirePackRun(t, 7, 20)
	if ha, hb := hires(a), hires(b); len(ha) == 0 || !reflect.DeepEqual(ha, hb) {
		t.Errorf("hire events of seed 7 twice: %d and %d events, want equal and non-empty", len(ha), len(hb))
	}
	snapOf := func(tape *hireTape) economysim.Snapshot {
		w := hirePackWorld(t, 7, tape)
		for range 20 {
			if err := w.Step(); err != nil {
				t.Fatalf("Step() = %v, want nil", err)
			}
		}
		sn, err := w.Snapshot()
		if err != nil {
			t.Fatalf("Snapshot() = %v, want nil", err)
		}
		return sn
	}
	if with, without := snapOf(&hireTape{}), snapOf(nil); !reflect.DeepEqual(with, without) {
		t.Errorf("Snapshot() of seed 7 with an observer differs from the one without, want equal")
	}
}

// TestHirePack_ModuleOffFormsNoHire verifies that the same consumer strategy on
// the pack without the overlay (employment off) forms no hire and emits no hire
// event: its offers and vacancies are refused at Debug, and the world steps
// without error.
//
// Requirement: FR-EMP-001, FR-EMP-010, FR-EMP-011
func TestHirePack_ModuleOffFormsNoHire(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := loadPack(t, packOneDemand...)
	for i := range cfg.Roles {
		cfg.Roles[i].Strategy = "hire_pack"
	}
	var tape hireTape
	s := hirePackStrategyFor(t, cfg)
	w, err := economysim.NewWorld(cfg, economysim.WithSeed(1), economysim.WithAudit(),
		economysim.WithStrategy("hire_pack", s), economysim.WithObserver(&tape))
	if err != nil {
		t.Fatalf("NewWorld(one-demand, module off) = %v, want nil", err)
	}
	for range 10 {
		if err := w.Step(); err != nil {
			t.Fatalf("Step() = %v, want nil", err)
		}
	}
	for _, e := range tape.events {
		switch e.(type) {
		case economysim.HireFormed, economysim.HireEnded:
			t.Fatalf("event %T with the module off, want none", e)
		}
	}
}
