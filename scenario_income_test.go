package economysim_test

import (
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/internal/checked"
)

// This file is the realised income per household per role per window of
// doc/plans/plan-2-8.md D364 (1), (3), (4) and (5), built from events only
// (FR-REF-027): a recording observer joins the TradeExecuted and RecipeExecuted
// events to the role the agent held at the time, which AgentSpawned,
// RoleChanged, WeightChanged and AgentBankrupted keep. It is the static income
// of CensusRow.IncomeBP seen from the run.
//
// Two figures are kept, never one net figure, because a TradeExecuted does not say
// whether a purchase is a recipe input or upkeep (staple and fuel are both): the
// gross sales proceeds of the role's sellers, and the cost of the inputs of the
// recipes it executed, the input quantities times the executions priced at the
// window's volume-weighted mean trade price of the commodity. The margin is their
// difference. The share of all household money that a role holds (D364 (2)) is
// not here: neither the events nor the snapshot say what one role holds.

// The mean trade price is pooled across regions, as the role cells are, so the
// margins are right for a single-region pack (one) only; key both by region
// before reusing the observer for a multi-region pack such as five.

// incomeWindow is the window of D364 in ticks.
const incomeWindow = 100

// incomeCell is what the observer kept for one role in one window.
type incomeCell struct {
	Sales          int64                         // gross money received by the role's sellers
	HouseholdTicks int64                         // households of the role, summed over the ticks of the window
	Unfed          int64                         // households unfed, summed over the ticks (UpkeepShortfall)
	Executions     map[economysim.RecipeID]int64 // executions started and completed, by recipe
}

// incomeAgent is the role and the households of a live agent.
type incomeAgent struct {
	role   economysim.RoleID
	weight int64
}

// incomeObs is the recording observer of the realised income.
type incomeObs struct {
	agents map[economysim.AgentID]incomeAgent
	live   map[economysim.RoleID]int64 // households of each role now
	cells  map[[2]int64]*incomeCell    // (window, role)
	// vwm is the money and the units traded in a window, by commodity, for the
	// volume-weighted mean price of the window.
	vwmValue, vwmUnits map[[2]int64]int64 // (window, commodity)
	inputs             map[economysim.RecipeID][]economysim.Amount
	overflow           bool // a sum of the stream did not fit int64: every Row is then absent
}

// newIncomeObs makes the observer for cfg whose names are names (the Names of a
// snapshot of the world). The inputs of each recipe come from the configuration.
func newIncomeObs(cfg economysim.Config, names economysim.Names) *incomeObs {
	o := &incomeObs{
		agents: map[economysim.AgentID]incomeAgent{}, live: map[economysim.RoleID]int64{},
		cells: map[[2]int64]*incomeCell{}, vwmValue: map[[2]int64]int64{}, vwmUnits: map[[2]int64]int64{},
		inputs: map[economysim.RecipeID][]economysim.Amount{},
	}
	index := func(list []string, id string) int {
		for i, s := range list {
			if s == id {
				return i
			}
		}
		panic("incomeObs: " + id + " is not in the names of the world") // a mismatched probe, not input
	}
	for _, r := range cfg.Recipes {
		ri := index(names.Recipes, r.ID)
		for _, in := range r.Inputs {
			o.inputs[economysim.RecipeID(ri)] = append(o.inputs[economysim.RecipeID(ri)],
				economysim.Amount{Commodity: economysim.CommodityID(index(names.Commodities, in.Commodity)), Quantity: in.Quantity})
		}
	}
	return o
}

func (o *incomeObs) cell(tick int64, role economysim.RoleID) *incomeCell {
	k := [2]int64{tick / incomeWindow, int64(role)}
	if o.cells[k] == nil {
		o.cells[k] = &incomeCell{Executions: map[economysim.RecipeID]int64{}}
	}
	return o.cells[k]
}

// Observe keeps what the income measure reads.
func (o *incomeObs) Observe(e economysim.Event) {
	switch e := e.(type) {
	case economysim.AgentSpawned:
		o.agents[e.Agent] = incomeAgent{e.Role, e.Weight}
		o.live[e.Role] += e.Weight
	case economysim.RoleChanged:
		a, ok := o.agents[e.Agent]
		if !ok { // an agent the observer never saw spawn: nothing to move
			return
		}
		o.live[a.role] -= a.weight
		o.agents[e.Agent] = incomeAgent{e.To, a.weight}
		o.live[e.To] += a.weight
	case economysim.WeightChanged:
		a, ok := o.agents[e.Agent]
		if !ok {
			return
		}
		o.live[a.role] += e.To - e.From
		o.agents[e.Agent] = incomeAgent{a.role, e.To}
	case economysim.AgentBankrupted:
		a, ok := o.agents[e.Agent]
		if !ok {
			return
		}
		o.live[a.role] -= a.weight
		delete(o.agents, e.Agent)
	case economysim.TradeExecuted:
		k := [2]int64{e.Tick / incomeWindow, int64(e.Commodity)}
		v, ok1 := checked.Mul(int64(e.Price), int64(e.Quantity))
		value, ok2 := checked.Add(o.vwmValue[k], v)
		units, ok3 := checked.Add(o.vwmUnits[k], int64(e.Quantity))
		if !ok1 || !ok2 || !ok3 {
			o.overflow = true
			return
		}
		o.vwmValue[k], o.vwmUnits[k] = value, units
		if a, ok := o.agents[e.Seller]; ok { // an estate is not an agent of a role
			c := o.cell(e.Tick, a.role)
			sales, ok := checked.Add(c.Sales, v)
			if !ok {
				o.overflow = true
				return
			}
			c.Sales = sales
		}
	case economysim.RecipeExecuted:
		if a, ok := o.agents[e.Agent]; ok {
			o.cell(e.Tick, a.role).Executions[e.Recipe] += e.Executions
		}
	case economysim.UpkeepShortfall:
		o.cell(e.Tick, e.Role).Unfed += e.Unfed
	case economysim.TickEnded:
		for role, hh := range o.live {
			o.cell(e.Tick, role).HouseholdTicks += hh
		}
	}
}

// inputCost is the cost of the inputs of the recipes the role executed in the
// window, at the window's volume-weighted mean price of each commodity (half to
// even); ok is false when a product does not fit int64.
func (o *incomeObs) inputCost(window int64, c *incomeCell) (cost int64, ok bool) {
	for recipe, n := range c.Executions {
		for _, in := range o.inputs[recipe] {
			k := [2]int64{window, int64(in.Commodity)}
			if o.vwmUnits[k] == 0 { // nothing traded: no price to value the inputs at
				continue
			}
			units, ok0 := checked.Mul(int64(in.Quantity), n)
			v, ok1 := checked.MulDivHalfEven(o.vwmValue[k], units, o.vwmUnits[k])
			sum, ok2 := checked.Add(cost, v)
			if !ok0 || !ok1 || !ok2 {
				return 0, false
			}
			cost = sum
		}
	}
	return cost, true
}

// incomeRow is the figures of one role and window: the two gross figures per
// household-tick and the margin as a share of the upkeep cost.
type incomeRow struct {
	SalesPerHousehold, InputCostPerHousehold int64 // minor units per household-tick
	MarginBP                                 int64 // (sales - input cost) / upkeep cost, basis points
	OK                                       bool  // false when the role had no household-ticks in the window or a figure overflowed
}

// Row returns the income of a role in a window (the ticks from window*incomeWindow
// on), with upkeep the upkeep cost of a household-tick at the published prices.
func (o *incomeObs) Row(window int64, role economysim.RoleID, upkeep int64) incomeRow {
	c := o.cells[[2]int64{window, int64(role)}]
	if o.overflow || c == nil || c.HouseholdTicks == 0 || upkeep < 1 {
		return incomeRow{}
	}
	cost, ok := o.inputCost(window, c)
	if !ok {
		return incomeRow{}
	}
	sales, ok1 := checked.MulDivHalfEven(c.Sales, 1, c.HouseholdTicks)
	inputs, ok2 := checked.MulDivHalfEven(cost, 1, c.HouseholdTicks)
	den, ok3 := checked.Mul(c.HouseholdTicks, upkeep)
	margin, ok4 := checked.MulDivHalfEven(c.Sales-cost, 10_000, den)
	return incomeRow{sales, inputs, margin, ok1 && ok2 && ok3 && ok4}
}

// TestScenarioHelpers_Income verifies the realised-income observer on a
// hand-built stream: gross sales are credited to the seller's role, the recipe
// inputs are valued at the window's volume-weighted mean price, the households
// are summed per tick, a role change moves the agent's later trades to the new
// role, and an estate seller or a later window is not mixed in.
//
// Two agents: 7 (role 0, weight 2) and 8 (role 1, weight 1). Tick 0: 7 sells 3
// units of commodity 0 at 10 (30), 8 sells 1 at 20 (20) and an estate sells 1 at
// 99, which counts in the window's mean price but in no role's sales. 7 executes
// a recipe that takes 2 units of commodity 0 three times. Tick 1: 7 changes to
// role 1, then sells 4 units at 5 (20) that role 1 receives. Tick 100 is the next
// window.
//
// Requirement: FR-REF-027
func TestScenarioHelpers_Income(t *testing.T) {
	t.Parallel()
	cfg := economysim.Config{Recipes: []economysim.Recipe{{ID: "r", Inputs: []economysim.Item{{Commodity: "g", Quantity: 2}}}}}
	names := economysim.Names{Commodities: []string{"g"}, Recipes: []string{"r"}}
	o := newIncomeObs(cfg, names)
	for _, e := range []economysim.Event{
		economysim.AgentSpawned{Agent: 7, Role: 0, Weight: 2},
		economysim.AgentSpawned{Agent: 8, Role: 1, Weight: 1},
		economysim.TradeExecuted{Tick: 0, Seller: 7, Price: 10, Quantity: 3},
		economysim.TradeExecuted{Tick: 0, Seller: 8, Price: 20, Quantity: 1},
		economysim.TradeExecuted{Tick: 0, Seller: 1, Price: 99, Quantity: 1}, // an estate: no role
		economysim.RecipeExecuted{Tick: 0, Agent: 7, Recipe: 0, Executions: 3},
		economysim.UpkeepShortfall{Tick: 0, Role: 1, Unfed: 1},
		economysim.TickEnded{Tick: 0},
		economysim.RoleChanged{Tick: 1, Agent: 7, From: 0, To: 1, Weight: 2},
		economysim.TradeExecuted{Tick: 1, Seller: 7, Price: 5, Quantity: 4},
		economysim.TickEnded{Tick: 1},
		economysim.TradeExecuted{Tick: 100, Seller: 8, Price: 1, Quantity: 1},
		economysim.TickEnded{Tick: 100},
	} {
		o.Observe(e)
	}
	// Mean price of commodity 0 in window 0, the estate's trade included:
	// (30 + 20 + 99 + 20) / (3 + 1 + 1 + 4) = 169 / 9 = 18.78.
	c0 := o.cells[[2]int64{0, 0}]
	if c0 == nil || c0.Sales != 30 || c0.HouseholdTicks != 2 || c0.Executions[0] != 3 {
		t.Fatalf("role 0 window 0 = %+v, want Sales 30, HouseholdTicks 2 (weight 2 at tick 0, none after the change), 3 executions", c0)
	}
	c1 := o.cells[[2]int64{0, 1}]
	// Role 1: 20 at tick 0 and the 20 of agent 7 at tick 1 = 40; households 1 at
	// tick 0 and 3 at tick 1 = 4; 1 unfed.
	if c1 == nil || c1.Sales != 40 || c1.HouseholdTicks != 4 || c1.Unfed != 1 {
		t.Errorf("role 1 window 0 = %+v, want Sales 40, HouseholdTicks 4, Unfed 1", c1)
	}
	// Inputs of role 0: 3 executions * 2 units = 6 units at 169 / 9 = 18.78 per
	// unit = 6 * 169 / 9 = 112.67, half to even 113. Upkeep 100: sales per
	// household-tick 30 / 2 = 15, inputs 113 / 2 = 56.5 -> 56 (half to even),
	// margin (30 - 113) * 10 000 / (2 * 100) = -4 150.
	got := o.Row(0, 0, 100)
	want := incomeRow{SalesPerHousehold: 15, InputCostPerHousehold: 56, MarginBP: -4_150, OK: true}
	if got != want {
		t.Errorf("Row(0, role 0, 100) = %+v, want %+v", got, want)
	}
	if r := o.Row(1, 0, 100); r.OK {
		t.Errorf("Row(1, role 0, 100) = %+v, want not OK: role 0 had no household-tick in window 1", r)
	}
	if c := o.cells[[2]int64{1, 1}]; c == nil || c.Sales != 1 || c.HouseholdTicks != 3 {
		t.Errorf("role 1 window 1 = %+v, want Sales 1, HouseholdTicks 3 (agent 8 weight 1 and agent 7 weight 2)", c)
	}
}

// TestScenarioHelpers_IncomeIgnoresUnknownAgents verifies that a role change, a
// weight change or a removal of an agent the observer never saw spawn moves no
// household, and that a recipe input cost that overflows int64 is reported as
// not OK instead of wrapped.
//
// Guard: a zero-valued agent record would otherwise subtract 0 from role 0 and
// move an unknown agent into the new role; an unchecked quantity times
// executions would wrap and report a wrong margin as valid.
func TestScenarioHelpers_IncomeIgnoresUnknownAgents(t *testing.T) {
	t.Parallel()
	cfg := economysim.Config{Recipes: []economysim.Recipe{{ID: "r", Inputs: []economysim.Item{{Commodity: "g", Quantity: 1 << 40}}}}}
	o := newIncomeObs(cfg, economysim.Names{Commodities: []string{"g"}, Recipes: []string{"r"}})
	for _, e := range []economysim.Event{
		economysim.AgentSpawned{Agent: 1, Role: 0, Weight: 3},
		economysim.RoleChanged{Agent: 99, From: 0, To: 1},
		economysim.WeightChanged{Agent: 99, From: 1, To: 5},
		economysim.AgentBankrupted{Agent: 99},
	} {
		o.Observe(e)
	}
	if o.live[0] != 3 || o.live[1] != 0 {
		t.Errorf("live households = %v after events of an unknown agent, want role 0: 3 and role 1: 0", o.live)
	}
	// 2^40 units per execution times 2^30 executions = 2^70 overflows int64.
	o.vwmValue[[2]int64{0, 0}], o.vwmUnits[[2]int64{0, 0}] = 10, 1
	c := &incomeCell{Executions: map[economysim.RecipeID]int64{0: 1 << 30}}
	if cost, ok := o.inputCost(0, c); ok {
		t.Errorf("inputCost = %d, ok, want not ok on overflow", cost)
	}
}
