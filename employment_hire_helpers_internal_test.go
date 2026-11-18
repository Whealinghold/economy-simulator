package economysim

import (
	"fmt"
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/ledger"
)

// hireScript is the strategy of the hire-form tests: Work returns what plan says
// for the agent (a scripted offer or vacancy, never an own job) and Trade places
// no order, so the only money that moves is the module's.
type hireScript struct {
	plan func(v *AgentView) JobRequest
}

func (s *hireScript) Work(v *AgentView, _ *rand.Rand) JobRequest {
	if s.plan == nil {
		return JobRequest{NoJob: true}
	}
	return s.plan(v)
}

func (*hireScript) Trade(*AgentView, *rand.Rand, *TradePlan) {}

func (*hireScript) Revise(o *OutcomeView, _ *rand.Rand) Belief { return o.Belief }

var _ Strategy = (*hireScript)(nil)

// hireOpts shapes hireWorld.
type hireOpts struct {
	farms, hands int64
	duration     int64
	farmMoney    Money
	maxHires     int64
	maxOpen      int64
	shareBP      int64
	yieldBP      int64
	input        bool // the recipe takes 1 grain an execution and the farm holds farmGrain
	farmGrain    int64
	tool         bool // the recipe needs a plough, which the farm holds farmTools of
	farmTools    int64
	sited        bool // the recipe is sited and has no yield row: no_site
	regions      int
	handRecipe   bool  // the hand has the recipe too, so it has an own job to fall back on
	handScale    int64 // households per hand agent, 0 for the world's scale
}

// hireWorld is a world of one raw commodity "grain" and two roles: "farm", an
// employer whose hire list is the recipe "tend", and "hand", a Hireable role. Each
// household is one agent of weight 1, the slots of the farms first. There is no
// upkeep and no order, so nothing moves money but the module. The strategy
// "script" is plan for both roles.
func hireWorld(t *testing.T, o hireOpts, plan func(v *AgentView) JobRequest, opts ...Option) *World {
	t.Helper()
	if o.regions == 0 {
		o.regions = 1
	}
	world := DefaultWorldParams()
	world.OutputCoverTicks, world.Upkeep, world.MinAgents = 0, nil, 1
	grain, plough := DefaultCommodity(), DefaultCommodity()
	grain.ID, grain.ReferencePrice, grain.Raw = "grain", 1_000, true
	plough.ID, plough.ReferencePrice, plough.Raw, plough.Durability = "plough", 1_000, true, 1_000_000
	farm, hand := DefaultRole(), DefaultRole()
	farm.ID, farm.Recipes, farm.StartMoney, farm.Strategy = "farm", []string{"tend"}, o.farmMoney, "script"
	farm.Employer = &Employer{MaxHires: o.maxHires, WorkerShareBP: o.shareBP, HireYieldBP: o.yieldBP, Recipes: []string{"tend"}}
	hand.ID, hand.StartMoney, hand.Hireable, hand.Strategy = "hand", 0, true, "script"
	if o.handRecipe {
		hand.Recipes = []string{"tend"}
	}
	if o.farmGrain > 0 {
		farm.StartStock = append(farm.StartStock, Item{Commodity: "grain", Quantity: Quantity(o.farmGrain)})
	}
	if o.farmTools > 0 {
		farm.StartStock = append(farm.StartStock, Item{Commodity: "plough", Quantity: Quantity(o.farmTools)})
	}
	rec := Recipe{ID: "tend", Duration: o.duration, Sited: o.sited, TraineeYield: 5_000, Outputs: []Output{{Commodity: "grain", Quantity: 2, CostShare: 10_000}}}
	if o.input {
		rec.Inputs = []Item{{Commodity: "grain", Quantity: 1}}
	}
	if o.tool {
		rec.Requires = []Tool{{Commodity: "plough", WearPerUse: 1}}
	}
	cfg := Config{World: world, Calendar: DefaultCalendar(), Commodities: []Commodity{grain}, Roles: []Role{farm, hand}, Recipes: []Recipe{rec}}
	if o.tool {
		cfg.Commodities = append(cfg.Commodities, plough)
	}
	for g := range o.regions {
		r := DefaultRegion()
		r.ID, r.ReferenceBasket = fmt.Sprintf("r%d", g), 0
		cfg.Regions = append(cfg.Regions, r)
		cfg.Groups = append(cfg.Groups, Group{Region: r.ID, Role: "farm", Households: o.farms}, Group{Region: r.ID, Role: "hand", Households: o.hands, Scale: o.handScale})
	}
	cfg.Modules.Employment = &Employment{Hire: true, MaxOpen: o.maxOpen}
	opts = append([]Option{WithSeed(1), WithAudit(), WithStrategy("script", &hireScript{plan: plan})}, opts...)
	w, err := NewWorld(cfg, opts...)
	if err != nil {
		t.Fatalf("NewWorld() error = %v, want nil", err)
	}
	return w
}

// defaultHire fills the zero fields of the options with the values the tests share.
func defaultHire(o hireOpts) hireOpts {
	for _, f := range []struct {
		v *int64
		d int64
	}{{&o.farms, 1}, {&o.hands, 1}, {&o.duration, 3}, {&o.maxHires, 4}, {&o.maxOpen, 1_000}, {&o.shareBP, 5_000}, {&o.yieldBP, 10_000}} {
		if *f.v == 0 {
			*f.v = f.d
		}
	}
	if o.farmMoney == 0 {
		o.farmMoney = 10_000
	}
	return o
}

// farmWants is the plan of the standard test: a farm posts positions of the recipe at
// bid and a hand offers its household at reserve.
func farmWants(positions int64, bid Price, reserve Price) func(v *AgentView) JobRequest {
	return func(v *AgentView) JobRequest {
		if v.Role == 0 { // farm: the first role of the configuration
			return JobRequest{NoJob: true, Hire: HireRequest{Recipe: 0, Positions: positions, Bid: bid}}
		}
		return JobRequest{NoJob: true, Offer: LabourOffer{Households: 1, Reserve: reserve}}
	}
}

// hireLog records the events of a world.
type hireLog struct {
	formed []HireFormed
	ended  []HireEnded
	all    []Event
}

func (l *hireLog) Observe(e Event) {
	l.all = append(l.all, e)
	switch e := e.(type) {
	case HireFormed:
		l.formed = append(l.formed, e)
	case HireEnded:
		l.ended = append(l.ended, e)
	}
}

// stepN steps the world n ticks, failing the test on an error (the audit is on).
func stepN(t *testing.T, w *World, n int) {
	t.Helper()
	for range n {
		if err := w.Step(); err != nil {
			t.Fatalf("Step() at tick %d error = %v, want nil", w.Tick(), err)
		}
	}
}

// escrowHeld is the money in all the Escrow accounts of the world.
func escrowHeld(t *testing.T, w *World) int64 {
	t.Helper()
	var sum int64
	for id := range w.ledger.Len() {
		if v, err := w.ledger.Account(id); err == nil && v.Kind == ledger.KindEscrow {
			sum += v.Balance
		}
	}
	return sum
}

// byRole returns a plan that returns farm for the farm role and hand for the other.
func byRole(farm, hand JobRequest) func(v *AgentView) JobRequest {
	return func(v *AgentView) JobRequest {
		if v.Role == 0 {
			return farm
		}
		return hand
	}
}

// slotsOfID is slotsOf as agent IDs.
func slotsOfID(w *World, role string) []AgentID {
	var out []AgentID
	for _, s := range slotsOf(w, role) {
		out = append(out, AgentID(s))
	}
	return out
}

// runToRation runs the positions of a step from its start to the labour round
// (P2.5a) by the rows of the replay, and returns with the step open so a test may
// change the world before the ration and the starts.
func runToRation(t *testing.T, w *World) {
	t.Helper()
	i := slices.IndexFunc(replayRows, func(r replayRow) bool { return r.id == "P2.6" })
	if err := runRows(w, w.Tick(), replayRows[:i], func(string) {}); err != nil {
		t.Fatalf("positions up to P2.5a: %v", err)
	}
}

// finishStep runs the rest of a step opened by runToRation, from P2.6 to the end,
// with the audit.
func finishStep(t *testing.T, w *World) {
	t.Helper()
	i := slices.IndexFunc(replayRows, func(r replayRow) bool { return r.id == "P2.6" })
	tick := w.Tick()
	if err := runRows(w, tick, replayRows[i:], func(string) {}); err != nil {
		t.Fatalf("positions from P2.6: %v", err)
	}
	if err := w.endStep(w.p84Audit()); err != nil {
		t.Fatalf("end of the step at tick %d: %v", tick, err)
	}
}
