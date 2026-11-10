package economysim

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"

	"github.com/Whealinghold/economy-sim/internal/codec"
	"github.com/Whealinghold/economy-sim/internal/ledger"
	"github.com/Whealinghold/economy-sim/internal/logkeys"
)

// This file is the integration calls of spec-integration.md: what a consumer
// does to a world between two steps. It is a core concept file with no module,
// no Config field and no default (plan 2-17 D540).

// enter is checks 1 to 3 of spec-integration.md 3.2 for a call that changes the
// world, in the order guard has: a running step, a failed world, then the module
// the call needs (module names it and on says whether it is on; an empty module
// needs none). The error carries the call's name (architecture.md 8.2).
func (w *World) enter(call, module string, on bool) error {
	if err := w.guard(); err != nil {
		return fmt.Errorf("%s: %w", call, err)
	}
	if module != "" && !on {
		return fmt.Errorf("%s: module %s: %w", call, module, ErrModuleOff)
	}
	return nil
}

// argError is an error of check 4 of spec-integration.md 3.2: a *ValidationError
// whose path names the argument, one segment each (entry names the slice entry
// as "changes[2]"), with the sentinel as its rule.
func argError(rule error, value, bound string, segs ...string) error {
	return &ValidationError{Path: segs, Rule: rule, Value: value, Bound: bound}
}

// entry names the i-th entry of the slice argument name, as the segment of a
// path: "changes[2]".
func entry(name string, i int) string { return name + "[" + itoa(i) + "]" }

// keyedOnce returns the index of the first key that appears an earlier time in
// keys, or -1 if every key is distinct: the check that a key appears once in a
// call (spec-integration.md 3.3.2). It sorts a copy, so the caller's slice is
// untouched, and it uses no map (architecture.md 7.5).
func keyedOnce[K ~int | ~int32 | ~int64](keys []K) int {
	idx := make([]int, len(keys))
	for i := range idx {
		idx[i] = i
	}
	slices.SortStableFunc(idx, func(a, b int) int { return cmp.Compare(keys[a], keys[b]) })
	dup := -1
	for j := 1; j < len(idx); j++ {
		if keys[idx[j]] == keys[idx[j-1]] && (dup < 0 || idx[j] < dup) {
			dup = idx[j]
		}
	}
	return dup
}

// NewAgent describes an external agent that a consumer adds to a world
// (spec-integration.md 4.1).
type NewAgent struct {
	// Region is the agent's region: an ordinary region, not an outside node.
	Region RegionID
	// Role is the agent's role: any role of the configuration.
	Role RoleID
	// Strategy is a registered name, or "default" or "directed"; "" means
	// "directed".
	Strategy string
	// Money is the money the agent starts with, 0 to MaxMoney, created by the
	// reason initial. Role.StartMoney is not applied.
	Money Money
	// Stock is the goods the agent starts with, one entry per commodity,
	// created by the reason initial.
	Stock []StockItem
	// Mortal says the failure ladder may act on the agent. An external agent is
	// not mortal by default.
	Mortal bool
}

// StockItem is a quantity of one commodity.
type StockItem struct {
	// Commodity is the commodity held.
	Commodity CommodityID
	// Quantity is the units, 1 to MaxQuantity.
	Quantity Quantity
}

// AddAgent adds an external agent of weight 1 and returns its ID, or NoAgent and
// the first error of the checks of spec-integration.md 4.1.1: ErrInStep,
// ErrWorldFailed, ErrUnknownRegion, ErrOutsideNode, ErrUnknownRole,
// ErrUnknownStrategy, ErrOutOfBounds, ErrUnknownCommodity, ErrDuplicateID and
// ErrOverflow. A refused call changes nothing and draws nothing. It acts between
// steps only: the agent exists when the call returns, takes the next ID, is
// created with its money and goods by the reason initial, and takes its two
// creation words from the stream keyed (siteAddAgent, Tick(), the new agent), so
// the order of calls changes no agent's words. It emits AgentSpawned with
// External true, at the tick the next step runs.
func (w *World) AddAgent(n NewAgent) (AgentID, error) {
	if err := w.enter("AddAgent", "", false); err != nil {
		return NoAgent, err
	}
	strategy, err := w.checkNewAgent(n)
	if err != nil {
		return NoAgent, fmt.Errorf("AddAgent: %w", err)
	}
	stock := make([]itemRow, len(n.Stock))
	funds := make([]ledger.Fund, len(n.Stock))
	for i, it := range n.Stock {
		stock[i] = itemRow{commodity: it.Commodity, quantity: it.Quantity}
		funds[i] = ledger.Fund{Commodity: int(it.Commodity), Qty: int64(it.Quantity)}
	}
	if !w.ledger.FundFits(int(n.Region), int64(n.Money), funds) {
		return NoAgent, fmt.Errorf("AddAgent: creating the agent's money and goods: %w", ErrOverflow)
	}
	id := w.createExternal(n, strategy, stock)
	return id, nil
}

// checkNewAgent is check 4 of spec-integration.md 3.2 for AddAgent, in the order
// of 4.1.1, before the overflow probe. It returns the index of the agent's
// strategy in the strategy table.
func (w *World) checkNewAgent(n NewAgent) (int, error) {
	if n.Region < 0 || int(n.Region) >= len(w.cfg.Regions) {
		return 0, argError(ErrUnknownRegion, itoa(int(n.Region)), "", "n", "region")
	}
	if w.isOutsideNode(n.Region) {
		return 0, argError(ErrOutsideNode, itoa(int(n.Region)), "", "n", "region")
	}
	if n.Role < 0 || int(n.Role) >= len(w.cfg.Roles) {
		return 0, argError(ErrUnknownRole, itoa(int(n.Role)), "", "n", "role")
	}
	name := n.Strategy
	if name == "" {
		name = directedStrategyName
	}
	strategy, ok := w.strategyIndex(name)
	if !ok {
		return 0, argError(ErrUnknownStrategy, name, "", "n", "strategy")
	}
	if n.Money < 0 || n.Money > MaxMoney {
		return 0, argError(ErrOutOfBounds, itoa(int(n.Money)), "MaxMoney", "n", "money")
	}
	return strategy, w.checkNewStock(n.Stock)
}

// checkNewStock checks the stock of a NewAgent: each quantity in [1, MaxQuantity],
// then each commodity known, then each commodity once.
func (w *World) checkNewStock(stock []StockItem) error {
	for i, it := range stock {
		if it.Quantity < 1 || it.Quantity > MaxQuantity {
			return argError(ErrOutOfBounds, itoa(int(it.Quantity)), "MaxQuantity", "n", entry("stock", i), "quantity")
		}
	}
	for i, it := range stock {
		if it.Commodity < 0 || int(it.Commodity) >= len(w.cfg.Commodities) {
			return argError(ErrUnknownCommodity, itoa(int(it.Commodity)), "", "n", entry("stock", i), "commodity")
		}
	}
	keys := make([]CommodityID, len(stock))
	for i, it := range stock {
		keys[i] = it.Commodity
	}
	if i := keyedOnce(keys); i >= 0 {
		return argError(ErrDuplicateID, itoa(int(stock[i].Commodity)), "", "n", entry("stock", i), "commodity")
	}
	return nil
}

// createExternal is the creation of AddAgent after its checks (spec-integration.md
// 4.1.2), in that order and mirroring spawnEntrant: the account, the funding by
// initial, the agent with its beliefs and its two creation words, its place in the
// world's tables and counters, and AgentSpawned. The probe has passed, so a
// refusal here is a violated invariant.
func (w *World) createExternal(n NewAgent, strategy int, stock []itemRow) AgentID {
	id, g, row := w.state.nextAgent, n.Region, w.roles.row(n.Role)
	if int(id) != len(w.agents) {
		invariant("the agent ID space is not dense")
	}
	acct, err := w.ledger.Open(ledger.KindAgent, int(id), int(g))
	if err != nil {
		invariant("the ledger refused the opening of an external agent's account: " + err.Error())
	}
	if err := fundInitial(w.ledger, acct, n.Money, stock); err != nil {
		invariant("the ledger refused the funding of an external agent after the probe: " + err.Error())
	}
	a := newAgent(agentInit{id: id, account: acct, role: n.Role, weight: 1, strategy: strategy, external: true, mortal: n.Mortal, tick: w.Tick()}, row, &w.recipes)
	w.fillSeeds(g, row)
	reseedBeliefs(&a, row, nil, w.ex.seeds, w.bp)
	role := &w.cfg.Roles[n.Role]
	w.rnd.keyed(siteAddAgent, w.Tick(), int64(id))
	drawCreation(w.rnd, &a, role.TargetRatio, role.TargetSpread)
	w.agents = append(w.agents, a)
	w.state.nextAgent++
	w.live.agents++
	w.live.households++
	cell := int(g)*len(w.cfg.Roles) + int(n.Role)
	w.live.inRole[cell]++
	w.state.hadAgent[cell] = true
	w.agentJoined(&w.agents[id], g)
	w.integ.watch(int32(id))
	if w.obs.supplied {
		w.obs.deliver(AgentSpawned{Tick: w.Tick(), Agent: id, Region: g, Role: n.Role, Weight: 1, StartMoney: n.Money, External: true})
	}
	if ctx := context.Background(); w.log.Enabled(ctx, slog.LevelDebug) {
		w.log.LogAttrs(ctx, slog.LevelDebug, "agent added", slog.Int64(logkeys.Tick, w.Tick()),
			slog.Int64(logkeys.Agent, int64(id)), slog.Int64(logkeys.Region, int64(g)), slog.Int64(logkeys.Role, int64(n.Role)))
	}
	return id
}

// externalAgent finds the live agent an ID names for a call that needs an
// external one (spec-integration.md 3.2 check 5, 7.1.1): ErrUnknownAgent for an ID
// never issued or a pseudo-agent, ErrAgentRemoved for one that was removed,
// ErrNotExternal for a household of the configuration or the spawner.
func (w *World) externalAgent(id AgentID) (*agent, error) {
	switch {
	case id < 0 || id >= w.state.nextAgent || int(id) < w.firstHousehold:
		return nil, ErrUnknownAgent
	case w.agents[id].weight < 1:
		return nil, ErrAgentRemoved
	case !w.agents[id].external:
		return nil, ErrNotExternal
	}
	return &w.agents[id], nil
}

// RemoveAgent removes an external agent the consumer no longer wants, mortal or
// not (spec-integration.md 4.4). It returns ErrInStep, ErrWorldFailed,
// ErrUnknownAgent (never issued, or a pseudo-agent), ErrAgentRemoved,
// ErrNotExternal, or ErrOverflow when the money or goods it hands to the estate
// would pass a bound; a refused call changes nothing. It is the one removal path
// of the world (removeAgent), with the cause consumer, so its events and their
// order are those of every removal; it draws nothing and the ID is never reused.
func (w *World) RemoveAgent(id AgentID) error {
	if err := w.enter("RemoveAgent", "", false); err != nil {
		return err
	}
	a, err := w.externalAgent(id)
	if err != nil {
		return fmt.Errorf("RemoveAgent: agent %d: %w", id, err)
	}
	if !w.estateFits(a) {
		return fmt.Errorf("RemoveAgent: agent %d: handing its money and goods to the estate: %w", id, ErrOverflow)
	}
	if err := w.removeAgent(a, CauseConsumer); err != nil {
		invariant("a removal that fits the estate failed: " + err.Error())
	}
	return nil
}

// estateFits is the read-only probe of a removal (plan 2-17 D547): whether the
// estate of the agent's region can take the agent's money and goods without
// passing a balance bound. The ledger has no undo, and the removal changes the
// job before the intake.
func (w *World) estateFits(a *agent) bool {
	region := w.regionOf(a)
	est, err := w.ledger.Account(w.estates.account(region))
	if err != nil {
		invariant("the estate account is unreadable: " + err.Error())
	}
	mine, err := w.ledger.Account(a.account)
	if err != nil {
		invariant("the account of an agent being removed is unreadable: " + err.Error())
	}
	if mine.Balance > MaxMoney-est.Balance {
		return false
	}
	lots, err := w.ledger.AppendLots(nil, a.account)
	if err != nil {
		invariant("the lots of an agent being removed are unreadable: " + err.Error())
	}
	held := make([]int64, len(w.cfg.Commodities))
	for _, lt := range lots {
		held[lt.Commodity] += lt.Quantity
	}
	for c, q := range held {
		if q == 0 {
			continue
		}
		have, _, err := w.ledger.Holding(w.estates.account(region), c)
		if err != nil {
			invariant("the estate holding is unreadable: " + err.Error())
		}
		if q > MaxQuantity-have {
			return false
		}
	}
	return true
}

// integrationState is what INT keeps of the world's own (plan 2-17): the agents
// position P1.9 visits.
type integrationState struct {
	// externals are the slots, ascending, of the agents P1.9 reads a mailbox of:
	// the external agents and any agent that holds a directive (only the test seam
	// gives a household one). It is derived from the agents, so it is not saved
	// and a restore rebuilds it (rebuildExternals); a removed agent stays in it
	// until the next P1.9, which drops it. tick.go calls P1.9 only while the list is
	// not empty, so a world that never held one records no P1.9 line.
	externals []int32
	// visit says externals is not empty: the guard of the P1.9 call in tick.go,
	// which may not call a function.
	visit bool
	// warnings are the warnings of the configuration, sorted by path then rule, as
	// Config.Warnings returned them at construction or restore. They are derived
	// from the configuration alone, so they are not saved.
	warnings []Warning
	// views holds, by agent slot, the rumours an external agent of an informed role
	// was given at its last view (spec-integration.md 4.7.3, plan 2-17 D553). Saved
	// in tag 22; it affects no result.
	views []rumourView
}

// rumourView is the last view built for an external agent of an informed role: the
// tick it was built at and a copy of its rumours, in the order of the view.
type rumourView struct {
	has     bool
	tick    int64
	rumours []Rumour
}

// watch adds slot to the agents P1.9 visits, keeping the list ascending and free
// of repeats.
func (s *integrationState) watch(slot int32) {
	i, found := slices.BinarySearch(s.externals, slot)
	if !found {
		s.externals = slices.Insert(s.externals, i, slot)
	}
	s.visit = true
}

// rebuildExternals sets the agents P1.9 visits from the agents of a restored
// world: the live external agents and the live agents that hold a directive.
func (w *World) rebuildExternals() {
	w.integ.externals = w.integ.externals[:0]
	for i := range w.agents {
		if a := &w.agents[i]; a.weight >= 1 && (a.external || a.directive != nil) {
			w.integ.externals = append(w.integ.externals, int32(i))
		}
	}
	w.integ.visit = len(w.integ.externals) > 0
}

// p19Directives is P1.9 (spec-world.md 6, spec-integration.md 4.2.3): for each
// agent that has a mailbox, in ascending ID, the directive of the tick is fixed
// (a.live) and a directive that is not Sticky leaves the mailbox, so it applies to
// this step only. The positions that read a directive (P2.5, P3.2) read a.live. A
// removed agent leaves the list here.
func (w *World) p19Directives() {
	keep := w.integ.externals[:0]
	for _, s := range w.integ.externals {
		a := &w.agents[s]
		if a.weight < 1 {
			continue
		}
		keep = append(keep, s)
		a.live = a.directive
		if a.directive != nil && !a.directive.Sticky {
			a.directive = nil
		}
	}
	w.integ.externals, w.integ.visit = keep, len(keep) > 0
}

// Direct leaves a directive in the mailbox of a directed external agent
// (spec-integration.md 4.3). It replaces what the mailbox held, an empty directive
// empties it, and it applies to the next step only unless Sticky. It returns
// ErrInStep, ErrWorldFailed, then, for the agent, ErrUnknownAgent, ErrAgentRemoved,
// ErrNotExternal and ErrNotDirected, then for the directive ErrUnknownRecipe,
// ErrOutOfBounds, ErrNotOnRole, ErrUnknownCommodity, ErrDuplicateID and
// ErrNotTradable, in the order of 4.3.2; a refused call changes nothing. It
// emits no event and checks nothing a tick can change (whether an order is
// affordable): the step refuses that.
func (w *World) Direct(id AgentID, d Directive) error {
	if err := w.enter("Direct", "", false); err != nil {
		return err
	}
	a, err := w.externalAgent(id)
	if err == nil && !isDirected(w.strategyOf(a)) {
		err = ErrNotDirected
	}
	if err != nil {
		return fmt.Errorf("Direct: agent %d: %w", id, err)
	}
	if err := w.checkDirective(a, &d); err != nil {
		return fmt.Errorf("Direct: agent %d: %w", id, err)
	}
	a.directive = d.mailbox()
	if ctx := context.Background(); w.log.Enabled(ctx, slog.LevelDebug) {
		w.log.LogAttrs(ctx, slog.LevelDebug, "directive set", slog.Int64(logkeys.Tick, w.Tick()), slog.Int64(logkeys.Agent, int64(id)))
	}
	return nil
}

// mailbox is the directive as the agent keeps it: a deep copy, so the caller's
// slices can change after the call, or nil for a directive with nothing in it.
func (d *Directive) mailbox() *Directive {
	if d.Job == nil && len(d.Intents) == 0 && len(d.Discards) == 0 && len(d.Hold) == 0 {
		return nil
	}
	c := Directive{Sticky: d.Sticky}
	if d.Job != nil {
		j := *d.Job
		c.Job = &j
	}
	c.Intents = slices.Clone(d.Intents)
	c.Discards = slices.Clone(d.Discards)
	c.Hold = slices.Clone(d.Hold)
	return &c
}

// checkDirective is rows 2 to 6 of the table of spec-integration.md 4.3.2, in
// order. Row 1 (the module of Offer, Hire, Borrow and Lend) and row 7 wait for the
// modules that own those fields (plan 2-17 Scope).
func (w *World) checkDirective(a *agent, d *Directive) error {
	if err := w.checkDirectedJob(a, d.Job); err != nil {
		return err
	}
	if err := w.checkDirectedIntents(d.Intents); err != nil {
		return err
	}
	row := w.roles.row(a.role)
	for i, in := range d.Intents {
		if _, ok := slices.BinarySearch(row.tradable, in.Commodity); !ok {
			return argError(ErrNotTradable, itoa(int(in.Commodity)), "", "d", entry("intents", i), "commodity")
		}
	}
	if err := w.checkDirectedDiscards(d.Discards); err != nil {
		return err
	}
	return w.checkDirectedHold(d.Hold)
}

// checkDirectedJob is rows 2 and 3: a recipe that exists and is on the role's list
// (or its own-use list), and Executions from 0 to the agent's weight. A request for
// no job has nothing to check.
func (w *World) checkDirectedJob(a *agent, j *JobRequest) error {
	if j == nil || j.NoJob {
		return nil
	}
	if j.Recipe < 0 || int(j.Recipe) >= len(w.cfg.Recipes) {
		return argError(ErrUnknownRecipe, itoa(int(j.Recipe)), "", "d", "job", "recipe")
	}
	if j.Executions < 0 || j.Executions > a.weight {
		return argError(ErrOutOfBounds, itoa(int(j.Executions)), "weight", "d", "job", "executions")
	}
	if reason := admit(w.roles.row(a.role).jobs, j.Recipe); reason != "" {
		return argError(ErrNotOnRole, itoa(int(j.Recipe)), "", "d", "job", "recipe")
	}
	return nil
}

// known reports whether c names a commodity of the configuration.
func (w *World) known(c CommodityID) bool { return c >= 0 && int(c) < len(w.cfg.Commodities) }

// checkDirectedIntents is row 4: for each intent, in order, a known commodity, a
// valid side and purpose, and each amount in its bound; then the commodity not
// seen in an earlier intent.
func (w *World) checkDirectedIntents(intents []Intent) error {
	for i, in := range intents {
		seg := entry("intents", i)
		if !w.known(in.Commodity) {
			return argError(ErrUnknownCommodity, itoa(int(in.Commodity)), "", "d", seg, "commodity")
		}
		for _, f := range [...]struct {
			name, bound string
			v, lo, hi   int64
		}{
			{"side", "", int64(in.Side), int64(Bid), int64(Ask)},
			{"purpose", "", int64(in.Purpose), int64(PurposeOther), int64(PurposeTool)},
			{"quantity", "MaxQuantity", int64(in.Quantity), 1, MaxQuantity},
			{"limit", "MaxPrice", int64(in.Limit), 1, MaxPrice},
			{"target", "MaxQuantity", int64(in.Target), 0, MaxQuantity},
			{"reserve", "MaxQuantity", int64(in.Reserve), 0, MaxQuantity},
			{"unit_cost", "MaxPrice", int64(in.UnitCost), 0, MaxPrice},
		} {
			if f.v < f.lo || f.v > f.hi {
				return argError(ErrOutOfBounds, itoa(int(f.v)), f.bound, "d", seg, f.name)
			}
		}
		for _, earlier := range intents[:i] {
			if earlier.Commodity == in.Commodity {
				return argError(ErrDuplicateID, itoa(int(in.Commodity)), "", "d", seg, "commodity")
			}
		}
	}
	return nil
}

// checkDirectedDiscards is the discards of row 6: a known commodity, a quantity in
// [1, MaxQuantity], each commodity once.
func (w *World) checkDirectedDiscards(discards []Discard) error {
	for i, ds := range discards {
		seg := entry("discards", i)
		switch {
		case !w.known(ds.Commodity):
			return argError(ErrUnknownCommodity, itoa(int(ds.Commodity)), "", "d", seg, "commodity")
		case ds.Quantity < 1 || ds.Quantity > MaxQuantity:
			return argError(ErrOutOfBounds, itoa(int(ds.Quantity)), "MaxQuantity", "d", seg, "quantity")
		}
		for _, earlier := range discards[:i] {
			if earlier.Commodity == ds.Commodity {
				return argError(ErrDuplicateID, itoa(int(ds.Commodity)), "", "d", seg, "commodity")
			}
		}
	}
	return nil
}

// checkDirectedHold is the hold list of row 6: each a known commodity, once.
func (w *World) checkDirectedHold(hold []CommodityID) error {
	for i, c := range hold {
		if !w.known(c) {
			return argError(ErrUnknownCommodity, itoa(int(c)), "", "d", entry("hold", i))
		}
		if slices.Contains(hold[:i], c) {
			return argError(ErrDuplicateID, itoa(int(c)), "", "d", entry("hold", i))
		}
	}
	return nil
}

// The slot entries of spec-integration.md 5.1: a key and the whole value of the key.

// YieldShock is the yield factor of a recipe in a region, in basis points: 0 to
// MaxYieldBP, 10 000 when none is set (SetYieldShocks).
type YieldShock struct {
	Region RegionID
	Recipe RecipeID
	BP     int64
}

// TransitFactor is the factor on the length of a route by a mode, in basis points:
// 0 to MaxTransitFactorBP, 10 000 when none is set, 0 blocks new dispatches
// (SetTransitFactors).
type TransitFactor struct {
	From, To RegionID
	Mode     ModeID
	BP       int64
}

// FlowScale is the scale of a money flow, named by its ID, in basis points: 0 to
// MaxFlowScale, 10 000 when none is set, 0 pauses the flow (SetFlowScales).
type FlowScale struct {
	Flow string
	BP   int64
}

// RegionScale is a scale of a region in basis points: the birth or the death scale
// of the population module, 0 to MaxBasisPoints, the region's configured value
// when none is set (SetBirthScales, SetDeathScales).
type RegionScale struct {
	Region RegionID
	BP     int64
}

// EndowmentScale is a scale of an endowment in basis points: the capacity or the
// regeneration scale, 0 to MaxShockBP, 10 000 when none is set (SetCapacityScales,
// SetRegenScales).
type EndowmentScale struct {
	Region   RegionID
	Resource ResourceID
	BP       int64
}

// EndowmentAdjustment is a signed change of the stock of an endowment, in units,
// at most MaxEndowment in magnitude; the stock is clamped to [0, its capacity]
// (AdjustEndowments).
type EndowmentAdjustment struct {
	Region   RegionID
	Resource ResourceID
	Delta    int64
}

// OutsideShock is the price shock of an outside line, in basis points: 1 to
// MaxShockBP, 10 000 when none is set (SetOutsideShocks).
type OutsideShock struct {
	Node      RegionID
	Commodity CommodityID
	BP        int64
}

// slotKeys is check 4 of spec-integration.md 3.2 for a slice argument named
// "changes": check validates entry i (known IDs, then bounds, in the order of
// 5.1.3) and returns its key, a number that orders the keys canonically; the first
// entry that fails, or that repeats the key of an earlier one (ErrDuplicateID, its
// path ending dupField), is the error. The keys are returned for the apply.
func slotKeys[E any](entries []E, check func(i int, e E) (int64, error), dupField string) ([]int64, error) {
	keys := make([]int64, 0, len(entries))
	dup := func() error {
		if d := keyedOnce(keys); d >= 0 {
			return argError(ErrDuplicateID, fmt.Sprintf("%+v", entries[d]), "", entry("changes", d), dupField)
		}
		return nil
	}
	for i, e := range entries {
		k, err := check(i, e)
		if err != nil {
			return nil, cmp.Or(dup(), err)
		}
		keys = append(keys, k)
	}
	return keys, dup()
}

// inKeyOrder returns the indexes of keys in ascending key order, the canonical
// order of the events of a call (spec-integration.md 3.3.3). The keys are distinct.
func inKeyOrder(keys []int64) []int {
	idx := make([]int, len(keys))
	for i := range idx {
		idx[i] = i
	}
	slices.SortFunc(idx, func(a, b int) int { return cmp.Compare(keys[a], keys[b]) })
	return idx
}

// slotBP is the check of a value in basis points against its bound.
func slotBP(i int, bp, lo, hi int64, bound string) error {
	if bp < lo || bp > hi {
		return argError(ErrOutOfBounds, itoa(int(bp)), bound, entry("changes", i), "bp")
	}
	return nil
}

// slotRegion checks a region key of an entry: known, and not an outside node when
// ordinary says the call is for households (a region key that names an outside
// node has no recipes, no households and no endowments).
func (w *World) slotRegion(i int, r RegionID, field string, ordinary bool) error {
	if r < 0 || int(r) >= len(w.cfg.Regions) {
		return argError(ErrUnknownRegion, itoa(int(r)), "", entry("changes", i), field)
	}
	if ordinary && w.isOutsideNode(r) {
		return argError(ErrOutsideNode, itoa(int(r)), "", entry("changes", i), field)
	}
	return nil
}

// slotNote delivers the events of a call that the owner's form does not emit, after
// the whole call has been applied, and writes the call's Debug line (3.3.3, 12.2).
func slotNote[E Event](w *World, msg string, events []E) {
	if w.obs.supplied {
		for _, e := range events {
			w.obs.deliver(e)
		}
	}
	if ctx := context.Background(); w.log.Enabled(ctx, slog.LevelDebug) {
		w.log.LogAttrs(ctx, slog.LevelDebug, msg, slog.Int64(logkeys.Tick, w.Tick()))
	}
}

// SetYieldShocks sets the yield factor of recipes in regions (spec-integration.md
// 5.1): a key at 10 000 is cleared, a key at the value it holds is no change. The
// yield is read when a job starts, so a job in progress keeps its pending output.
// It returns ErrInStep, ErrWorldFailed, then for the first entry that fails
// ErrUnknownRegion, ErrOutsideNode, ErrUnknownRecipe, ErrOutOfBounds or
// ErrDuplicateID, as a *ValidationError; a refused call changes nothing. It emits
// YieldShocked for each changed key in (region, recipe) order.
func (w *World) SetYieldShocks(changes []YieldShock) error {
	if err := w.enter("SetYieldShocks", "", false); err != nil {
		return err
	}
	keys, err := slotKeys(changes, func(i int, c YieldShock) (int64, error) {
		if err := w.slotRegion(i, c.Region, "region", true); err != nil {
			return 0, err
		}
		if c.Recipe < 0 || int(c.Recipe) >= len(w.cfg.Recipes) {
			return 0, argError(ErrUnknownRecipe, itoa(int(c.Recipe)), "", entry("changes", i), "recipe")
		}
		return int64(c.Region)*int64(len(w.cfg.Recipes)) + int64(c.Recipe), slotBP(i, c.BP, 0, MaxYieldBP, "MaxYieldBP")
	}, "region")
	if err != nil {
		return fmt.Errorf("SetYieldShocks: %w", err)
	}
	var events []YieldShocked
	for _, i := range inKeyOrder(keys) {
		c := changes[i]
		before := w.shocks.factor(c.Region, c.Recipe)
		if before == c.BP {
			continue
		}
		if err := w.shocks.set(c.Region, c.Recipe, c.BP); err != nil {
			invariant("a validated yield shock was refused: " + err.Error())
		}
		events = append(events, YieldShocked{Tick: w.Tick(), Region: c.Region, Recipe: c.Recipe, BeforeBP: before, AfterBP: c.BP})
	}
	slotNote(w, "yield shocks set", events)
	return nil
}

// SetEdgeStatuses closes edges or changes their length by a factor (the table of
// spec-regions.md 7.3): an entry that opens an edge says Closed false and FactorBP
// 10 000. It returns ErrInStep, ErrWorldFailed, then for the first entry that fails
// ErrUnknownEdge, ErrOutOfBounds (FactorBP outside 1 to MaxEdgeFactorBP) or
// ErrDuplicateID; a refused call changes nothing. It rebuilds the graph at the call
// and emits EdgeStatusChanged, cause consumer, for each changed edge in EdgeID
// order.
func (w *World) SetEdgeStatuses(changes []EdgeStatus) error {
	if err := w.enter("SetEdgeStatuses", "", false); err != nil {
		return err
	}
	_, err := slotKeys(changes, func(i int, c EdgeStatus) (int64, error) {
		e, ok := w.geo.g.Edge(c.From, c.To)
		if !ok {
			return 0, argError(ErrUnknownEdge, fmt.Sprintf("%d to %d", c.From, c.To), "", entry("changes", i), "from")
		}
		return int64(e.ID), slotBP(i, c.FactorBP, 1, MaxEdgeFactorBP, "MaxEdgeFactorBP")
	}, "from")
	if err != nil {
		return fmt.Errorf("SetEdgeStatuses: %w", err)
	}
	if err := w.setEdgeStatuses(changes, CauseConsumer); err != nil {
		invariant("validated edge statuses were refused: " + err.Error())
	}
	return nil
}

// SetTransitFactors sets the factor on the length of routes by modes (the
// transport module): new dispatches read it, a shipment in flight keeps its
// arrival tick. It returns ErrInStep, ErrWorldFailed, ErrModuleOff, then for the
// first entry that fails ErrUnknownRegion, ErrOutsideNode, ErrSameRegion,
// ErrUnknownMode, ErrOutOfBounds or ErrDuplicateID; a refused call changes
// nothing. It emits TransitFactorChanged, cause consumer, for each changed key in
// (from, to, mode) order.
func (w *World) SetTransitFactors(changes []TransitFactor) error {
	if err := w.enter("SetTransitFactors", "transport", w.transport != nil); err != nil {
		return err
	}
	modes := int64(len(w.transport.modes))
	keys, err := slotKeys(changes, func(i int, c TransitFactor) (int64, error) {
		for _, r := range [...]struct {
			id    RegionID
			field string
		}{{c.From, "from"}, {c.To, "to"}} {
			if err := w.slotRegion(i, r.id, r.field, false); err != nil {
				return 0, err
			}
		}
		for _, r := range [...]struct {
			id    RegionID
			field string
		}{{c.From, "from"}, {c.To, "to"}} {
			if err := w.slotRegion(i, r.id, r.field, true); err != nil {
				return 0, err
			}
		}
		switch {
		case c.From == c.To:
			return 0, argError(ErrSameRegion, itoa(int(c.To)), "", entry("changes", i), "to")
		case c.Mode < 0 || int64(c.Mode) >= modes:
			return 0, argError(ErrUnknownMode, itoa(int(c.Mode)), "", entry("changes", i), "mode")
		}
		regions := int64(len(w.cfg.Regions))
		return (int64(c.From)*regions+int64(c.To))*modes + int64(c.Mode), slotBP(i, c.BP, 0, MaxTransitFactorBP, "MaxTransitFactorBP")
	}, "from")
	if err != nil {
		return fmt.Errorf("SetTransitFactors: %w", err)
	}
	for _, i := range inKeyOrder(keys) {
		c := changes[i]
		if err := w.setTransitFactor(c.From, c.To, c.Mode, c.BP, CauseConsumer); err != nil {
			invariant("a validated transit factor was refused: " + err.Error())
		}
	}
	return nil
}

// SetFlowScales sets the scale of money flows by ID (the money supply module): 0
// pauses a flow, 10 000 is the configured rate. It returns ErrInStep,
// ErrWorldFailed, ErrModuleOff, then for the first entry that fails ErrUnknownFlow,
// ErrOutOfBounds or ErrDuplicateID; a refused call changes nothing. It emits
// MoneyFlowScaled, cause consumer, for each changed flow in flow ID order, and the
// flow is skipped whole or scaled from the next P1.7.
func (w *World) SetFlowScales(changes []FlowScale) error {
	if err := w.enter("SetFlowScales", "money_supply", w.money != nil); err != nil {
		return err
	}
	keys, err := slotKeys(changes, func(i int, c FlowScale) (int64, error) {
		at := slices.IndexFunc(w.money.flows, func(f flowState) bool { return f.flow.ID == c.Flow })
		if at < 0 {
			return 0, argError(ErrUnknownFlow, c.Flow, "", entry("changes", i), "flow")
		}
		return int64(at), slotBP(i, c.BP, 0, MaxFlowScale, "MaxFlowScale")
	}, "flow")
	if err != nil {
		return fmt.Errorf("SetFlowScales: %w", err)
	}
	for _, i := range inKeyOrder(keys) {
		if err := w.setFlowScale(int(keys[i]), changes[i].BP, CauseConsumer); err != nil {
			invariant("a validated flow scale was refused: " + err.Error())
		}
	}
	return nil
}

// SetBirthScales sets the birth scale of regions (the population module), which
// replaces the region's configured BirthScale; a scale equal to the configured one
// clears the key. It returns ErrInStep, ErrWorldFailed, ErrModuleOff, then for the
// first entry that fails ErrUnknownRegion, ErrOutsideNode, ErrOutOfBounds or
// ErrDuplicateID; a refused call changes nothing. It emits PopulationScaled for
// each changed region in ascending order; P7.1 reads the scale from the next step.
func (w *World) SetBirthScales(changes []RegionScale) error {
	return w.setPopulationScales("SetBirthScales", ScaleBirth, changes)
}

// SetDeathScales is SetBirthScales for the death scale.
func (w *World) SetDeathScales(changes []RegionScale) error {
	return w.setPopulationScales("SetDeathScales", ScaleDeath, changes)
}

// setPopulationScales is the call of SetBirthScales and SetDeathScales.
func (w *World) setPopulationScales(call string, kind ScaleKind, changes []RegionScale) error {
	if err := w.enter(call, "population", w.pop != nil); err != nil {
		return err
	}
	keys, err := slotKeys(changes, func(i int, c RegionScale) (int64, error) {
		if err := w.slotRegion(i, c.Region, "region", true); err != nil {
			return 0, err
		}
		return int64(c.Region), slotBP(i, c.BP, 0, MaxBasisPoints, "MaxBasisPoints")
	}, "region")
	if err != nil {
		return fmt.Errorf("%s: %w", call, err)
	}
	var events []PopulationScaled
	for _, i := range inKeyOrder(keys) {
		c := changes[i]
		before, ok := w.setPopulationScale(c.Region, kind, c.BP)
		if !ok {
			invariant("a validated population scale was refused")
		}
		if before != c.BP {
			events = append(events, PopulationScaled{Tick: w.Tick(), Region: c.Region, Which: kind, BeforeBP: before, AfterBP: c.BP})
		}
	}
	slotNote(w, "population scales set", events)
	return nil
}

// SetCapacityScales sets the capacity scale of endowments (the resources module):
// the effective capacity is recomputed at the call and a stock above it is clamped,
// the excess lost. It returns ErrInStep, ErrWorldFailed, ErrModuleOff, then for the
// first entry that fails ErrUnknownRegion, ErrUnknownResource, ErrOutOfBounds or
// ErrDuplicateID; a refused call changes nothing. It emits EndowmentAdjusted, kind
// shock, for each changed key in (region, resource) order.
func (w *World) SetCapacityScales(changes []EndowmentScale) error {
	return w.setEndowmentScales("SetCapacityScales", changes, (*World).setCapacityScales)
}

// SetRegenScales sets the regeneration scale of endowments, read at P8.3; the stock
// is not touched. Its errors and its event are those of SetCapacityScales.
func (w *World) SetRegenScales(changes []EndowmentScale) error {
	return w.setEndowmentScales("SetRegenScales", changes, (*World).setRegenScales)
}

// setEndowmentScales is the call of the two scales of an endowment: apply is the
// owner's form, which emits its own event.
func (w *World) setEndowmentScales(call string, changes []EndowmentScale, apply func(*World, RegionID, ResourceID, int64)) error {
	if err := w.enter(call, "resources", w.res != nil); err != nil {
		return err
	}
	keys, err := slotKeys(changes, func(i int, c EndowmentScale) (int64, error) {
		key, err := w.slotEndowment(i, c.Region, c.Resource)
		if err != nil {
			return 0, err
		}
		return key, slotBP(i, c.BP, 0, MaxShockBP, "MaxShockBP")
	}, "region")
	if err != nil {
		return fmt.Errorf("%s: %w", call, err)
	}
	for _, i := range inKeyOrder(keys) {
		apply(w, changes[i].Region, changes[i].Resource, changes[i].BP)
	}
	return nil
}

// slotEndowment checks the key of an endowment entry, a region and a resource of
// the configuration, and returns the number that orders it by (region, resource).
func (w *World) slotEndowment(i int, region RegionID, resource ResourceID) (int64, error) {
	if err := w.slotRegion(i, region, "region", false); err != nil {
		return 0, err
	}
	if resource < 0 || int(resource) >= w.res.resources {
		return 0, argError(ErrUnknownResource, itoa(int(resource)), "", entry("changes", i), "resource")
	}
	return int64(w.res.cellOf(region, resource)), nil
}

// AdjustEndowments changes the stock of endowments now, by a signed number of
// units clamped so the stock stays in [0, its capacity] (the resources module). It
// holds no value and has no default. It returns ErrInStep, ErrWorldFailed,
// ErrModuleOff, then for the first entry that fails ErrUnknownRegion,
// ErrUnknownResource, ErrOutOfBounds (a Delta above MaxEndowment in magnitude) or
// ErrDuplicateID; a refused call changes nothing. It emits EndowmentAdjusted, kind
// adjust, for each stock that changed, in (region, resource) order.
func (w *World) AdjustEndowments(changes []EndowmentAdjustment) error {
	if err := w.enter("AdjustEndowments", "resources", w.res != nil); err != nil {
		return err
	}
	keys, err := slotKeys(changes, func(i int, c EndowmentAdjustment) (int64, error) {
		key, err := w.slotEndowment(i, c.Region, c.Resource)
		if err != nil {
			return 0, err
		}
		if c.Delta < -MaxEndowment || c.Delta > MaxEndowment {
			return 0, argError(ErrOutOfBounds, itoa(int(c.Delta)), "MaxEndowment", entry("changes", i), "delta")
		}
		return key, nil
	}, "region")
	if err != nil {
		return fmt.Errorf("AdjustEndowments: %w", err)
	}
	for _, i := range inKeyOrder(keys) {
		w.adjustEndowments(changes[i].Region, changes[i].Resource, changes[i].Delta)
	}
	return nil
}

// SetOutsideShocks sets the price shock of outside lines (the external module):
// the walk and the quotes read it from the next step. It returns ErrInStep,
// ErrWorldFailed, ErrModuleOff, then for the first entry that fails
// ErrUnknownRegion, ErrUnknownCommodity, ErrUnknownLine (the node has no line for
// the commodity, and an ordinary region has none), ErrOutOfBounds or
// ErrDuplicateID; a refused call changes nothing. It emits OutsideShocked for each
// changed line in (node, commodity) order.
func (w *World) SetOutsideShocks(changes []OutsideShock) error {
	if err := w.enter("SetOutsideShocks", "external", w.external != nil); err != nil {
		return err
	}
	keys, err := slotKeys(changes, func(i int, c OutsideShock) (int64, error) {
		if err := w.slotRegion(i, c.Node, "node", false); err != nil {
			return 0, err
		}
		if !w.known(c.Commodity) {
			return 0, argError(ErrUnknownCommodity, itoa(int(c.Commodity)), "", entry("changes", i), "commodity")
		}
		if !slices.ContainsFunc(w.external.lines, func(l outsideLine) bool { return l.node == c.Node && l.commodity == c.Commodity }) {
			return 0, argError(ErrUnknownLine, fmt.Sprintf("%d, %d", c.Node, c.Commodity), "", entry("changes", i), "commodity")
		}
		return int64(c.Node)*int64(len(w.cfg.Commodities)) + int64(c.Commodity), slotBP(i, c.BP, 1, MaxShockBP, "MaxShockBP")
	}, "node")
	if err != nil {
		return fmt.Errorf("SetOutsideShocks: %w", err)
	}
	ordered := make([]outsideShock, 0, len(changes))
	for _, i := range inKeyOrder(keys) {
		ordered = append(ordered, outsideShock{node: changes[i].Node, commodity: changes[i].Commodity, bp: changes[i].BP})
	}
	slotNote(w, "outside shocks set", w.external.applyShocks(w.Tick(), ordered))
	return nil
}

// SlotState lists every slot value that differs from its default, in the canonical
// order of its owner, the order a checkpoint saves it in (spec-integration.md 5.3).
// A later module appends its field at the end.
type SlotState struct {
	YieldShocks    []YieldShock
	EdgeStatuses   []EdgeStatus
	TransitFactors []TransitFactor
	FlowScales     []FlowScale
	BirthScales    []RegionScale
	DeathScales    []RegionScale
	CapacityScales []EndowmentScale
	RegenScales    []EndowmentScale
	OutsideShocks  []OutsideShock
}

// Slots returns a deep copy of the slots that hold a value other than their default
// (spec-integration.md 5.3): a key set back to its default is not listed, and a
// world that was never shocked has an empty SlotState. It returns ErrInStep or
// ErrWorldFailed, changes nothing and draws nothing. It reads the lists the owners
// save, so a restored world reports what the original did. The one-off
// AdjustEndowments holds no value and is not listed.
func (w *World) Slots() (SlotState, error) {
	if err := w.enter("Slots", "", false); err != nil {
		return SlotState{}, err
	}
	var s SlotState
	for _, e := range w.shocks.ordered() {
		s.YieldShocks = append(s.YieldShocks, YieldShock{Region: e.region, Recipe: e.recipe, BP: e.bp})
	}
	for _, e := range w.edgeRows() {
		s.EdgeStatuses = append(s.EdgeStatuses, EdgeStatus(e))
	}
	w.transitSlots(&s)
	w.flowSlots(&s)
	w.populationSlots(&s)
	w.endowmentSlots(&s)
	w.outsideSlots(&s)
	return s, nil
}

// transitSlots, flowSlots, populationSlots, endowmentSlots and outsideSlots add the
// non-default keys of one owner to the slot list; each is empty with its module off.
func (w *World) transitSlots(s *SlotState) {
	if w.transport == nil {
		return
	}
	for _, f := range w.transport.factors {
		s.TransitFactors = append(s.TransitFactors, TransitFactor{From: f.from, To: f.to, Mode: f.mode, BP: f.bp})
	}
}

func (w *World) flowSlots(s *SlotState) {
	if w.money == nil {
		return
	}
	for _, f := range w.money.flows {
		if f.scale != wholeBP {
			s.FlowScales = append(s.FlowScales, FlowScale{Flow: f.flow.ID, BP: f.scale})
		}
	}
}

func (w *World) populationSlots(s *SlotState) {
	p := w.pop
	if p == nil {
		return
	}
	for g := range p.effBirth {
		if p.effBirth[g] != p.rowBirth[g] {
			s.BirthScales = append(s.BirthScales, RegionScale{Region: RegionID(g), BP: p.effBirth[g]})
		}
		if p.effDeath[g] != p.rowDeath[g] {
			s.DeathScales = append(s.DeathScales, RegionScale{Region: RegionID(g), BP: p.effDeath[g]})
		}
	}
}

func (w *World) endowmentSlots(s *SlotState) {
	r := w.res
	if r == nil {
		return
	}
	for cell := range r.capScale {
		if r.capScale[cell] != wholeBP {
			s.CapacityScales = append(s.CapacityScales, EndowmentScale{Region: r.regionOf(cell), Resource: r.resourceOf(cell), BP: r.capScale[cell]})
		}
		if r.regenScale[cell] != wholeBP {
			s.RegenScales = append(s.RegenScales, EndowmentScale{Region: r.regionOf(cell), Resource: r.resourceOf(cell), BP: r.regenScale[cell]})
		}
	}
}

func (w *World) outsideSlots(s *SlotState) {
	if w.external == nil {
		return
	}
	for _, l := range w.external.lines {
		if l.shock != wholeBP {
			s.OutsideShocks = append(s.OutsideShocks, OutsideShock{Node: l.node, Commodity: l.commodity, BP: l.shock})
		}
	}
}

// Relocate moves an agent, with all the households it stands for, to another region
// at once: the forced move of spec-integration.md 4.5, the guarded relocation of
// spec-regions.md 6 with the cause consumer. It is made for an external agent or a
// household of the configuration, not a pseudo-agent. It returns ErrInStep,
// ErrWorldFailed, then the first refusal of the dry run in REG's order:
// ErrUnknownRegion, ErrUnknownAgent, ErrAgentRemoved (a removed agent),
// ErrNotRelocatable (an estate or another pseudo-agent), ErrSameRegion,
// ErrOutsideNode, ErrOpenReservation or ErrRegionBound (a *RegionBoundError naming
// the obligation: open_shipment, open_contract and the others of the modules that
// are on). A refused call changes nothing and the journey of a migrant is dropped
// only by a call that moves it. After the dry run the call cannot fail: a migration
// journey is cancelled (its households count in Ended of the next MigrationReviewed),
// a job in progress is abandoned with the reason region_changed, the account moves
// (AgentRelocated, cause consumer), the beliefs are reseeded by the arrival rule with
// no draw and RoleSince becomes the tick the next step runs. Money, goods, role,
// weight, strategy and the mailbox stay as they were.
func (w *World) Relocate(id AgentID, to RegionID) error {
	if err := w.enter("Relocate", "", false); err != nil {
		return err
	}
	// The journey is the one obligation the call itself discharges (4.5.2 step 2).
	if _, err := w.relocateRefusal(id, to, []string{onJourneyName}); err != nil {
		if errors.Is(err, ErrClosedAccount) {
			err = ErrAgentRemoved // the dry run's closed account is a removed agent here
		}
		return fmt.Errorf("Relocate: agent %d to region %d: %w", id, to, err)
	}
	a := &w.agents[id]
	from := w.regionOf(a)
	if w.onJourney(a) {
		w.migration.cancelJourney(a, from)
	}
	if a.hasJob() {
		w.abandonJob(a, from, ReasonRegionChanged)
	}
	// moveAgent drops the own-use floor of the old region itself, the second step a
	// departing migrant takes (spec-migration.md 8.2).
	if err := w.moveAgent(a, to, CauseConsumer); err != nil {
		invariant("the ledger refused a relocation the dry run passed: " + err.Error())
	}
	w.reseedAtArrival(a, to)
	a.roleSince = w.Tick()
	return nil
}

// Warnings returns the warnings of the configuration the world was built from,
// sorted by path then rule (spec-integration.md 6.4): what the construction logged,
// for a consumer with no logger. A restored world holds the same list. The result is
// a deep copy. It works from inside a step and on a failed world.
func (w *World) Warnings() []Warning {
	out := slices.Clone(w.integ.warnings)
	for i := range out {
		out[i].Path = slices.Clone(out[i].Path)
	}
	return out
}

// Calendar returns the calendar of the world: At(tick) gives the moment and
// SeasonAt(tick) the periodic season (spec-integration.md 6.2). It is a deep copy
// of the configuration's, so changing it changes nothing in the world, and it works
// from inside a step and on a failed world.
func (w *World) Calendar() Calendar {
	c := w.cfg.Calendar
	c.Months, c.Seasons = slices.Clone(c.Months), slices.Clone(c.Seasons)
	return c
}

// Agents returns the live agents whose account is in region g, in ascending ID,
// with no estate or counterparty pseudo-agent (spec-integration.md 6.2). It returns
// ErrInStep from inside a step, ErrWorldFailed on a failed world and
// ErrUnknownRegion. An outside node has no agent, so its list is empty.
func (w *World) Agents(g RegionID) ([]AgentID, error) {
	if err := w.guard(); err != nil {
		return nil, fmt.Errorf("Agents: %w", err)
	}
	if g < 0 || int(g) >= len(w.cfg.Regions) {
		return nil, fmt.Errorf("Agents: region %d: %w", g, ErrUnknownRegion)
	}
	out := []AgentID{}
	for i := w.firstHousehold; i < len(w.agents); i++ {
		if a := &w.agents[i]; a.weight >= 1 && w.regionOf(a) == g {
			out = append(out, a.id)
		}
	}
	return out, nil
}

// recordRumours keeps the rumours of the trade view just built for an external agent
// of an informed role (spec-integration.md 4.7.3): a Work view never reaches it, so
// it cannot clear them. It is the one hook INT has in fillAgentView, runs for external
// agents only and reuses the agent's slice once it has grown.
func (w *World) recordRumours(v *AgentView, a *agent) {
	if !a.external || !w.info.informed[a.role] {
		return
	}
	for int(a.id) >= len(w.integ.views) {
		w.integ.views = append(w.integ.views, rumourView{})
	}
	rv := &w.integ.views[a.id]
	rv.has, rv.tick = true, w.Tick()
	rv.rumours = append(rv.rumours[:0], v.Rumours...)
}

// lastView returns the view kept for agent a: none for an agent that has not had a
// trade view, and none once it is not an external agent of an informed role (a role
// change), so that a query and a checkpoint agree.
func (w *World) lastView(a *agent) (*rumourView, bool) {
	if w.info == nil || !a.external || !w.info.informed[a.role] || int(a.id) >= len(w.integ.views) || !w.integ.views[a.id].has {
		return nil, false
	}
	return &w.integ.views[a.id], true
}

// saveRumours writes tag 22 (spec-checkpoint.md 4.2, spec-integration.md 13): for
// each live external agent of an informed role that has had a view, in ascending
// agent ID, the tick of the view and its rumours (region, commodity, price, mean, lag
// and source age) in the order of the view. It is sparse: an agent that has had no
// view has no row.
func (w *World) saveRumours(cw *codec.Writer) error {
	var rows []*agent
	for i := w.firstHousehold; i < len(w.agents); i++ {
		if a := &w.agents[i]; a.weight >= 1 {
			if _, ok := w.lastView(a); ok {
				rows = append(rows, a)
			}
		}
	}
	cw.Count(len(rows))
	for _, a := range rows {
		rv, _ := w.lastView(a)
		cw.U(uint64(a.id))
		cw.I(rv.tick)
		cw.Count(len(rv.rumours))
		for _, r := range rv.rumours {
			cw.U(uint64(r.Region))
			cw.U(uint64(r.Commodity))
			cw.U(unsigned(int64(r.Price)))
			cw.U(unsigned(int64(r.Mean)))
			cw.U(unsigned(r.Lag))
			cw.U(unsigned(r.SourceAge))
		}
	}
	return nil
}

// loadRumours reads tag 22 into the views of a world whose agents are loaded. A row
// is corrupt when its agent is not a live external agent of an informed role, when
// the agents are not strictly ascending, when the tick is not a step the run has
// made, when the count is above MaxRumoursPerView, or when a rumour is not one a view
// can hold: a region and a commodity of the configuration, a price from 1 to
// MaxPrice, a mean up to it, a lag from 2 to the module's maximum age, and the
// rumours not strictly ascending by (region, commodity) (spec-information.md 6.2).
func (w *World) loadRumours(rs *restoreCtx, r *codec.Reader) error {
	n := r.Count(3)
	var prev uint64
	for i := range n {
		id := r.Strict(prev, i > 0)
		prev = id
		if r.Err() != nil {
			return nil
		}
		if id >= uint64(len(w.agents)) || int(id) < w.firstHousehold || !w.liveInformedExternal(&w.agents[id]) {
			r.Failf("tag 22 row for agent %d, which is not a live external agent of an informed role", id)
			return nil
		}
		w.loadView(rs, r, AgentID(id))
	}
	return nil
}

// liveInformedExternal says a is a live external agent whose role is informed.
func (w *World) liveInformedExternal(a *agent) bool {
	return a.weight >= 1 && a.external && w.info != nil && w.info.informed[a.role]
}

// loadView reads the view of one row of tag 22 and keeps it for agent id.
func (w *World) loadView(rs *restoreCtx, r *codec.Reader, id AgentID) {
	tick := r.I()
	if r.Err() == nil && (tick < 0 || tick >= rs.tick) {
		r.Failf("tag 22 row for agent %d: view tick %d, want 0 to %d", id, tick, rs.tick-1)
		return
	}
	k := r.Count(6)
	if r.Err() == nil && k > MaxRumoursPerView {
		r.Failf("tag 22 row for agent %d holds %d rumours, want at most %d", id, k, MaxRumoursPerView)
		return
	}
	rumours := make([]Rumour, 0, k)
	for range k {
		rm := Rumour{
			Region: RegionID(r.Index(uint64(len(w.cfg.Regions)))), Commodity: CommodityID(r.Index(uint64(len(w.cfg.Commodities)))),
			Price: Price(r.Bounded("price", MaxPrice)), Mean: Price(r.Bounded("mean", MaxPrice)),
			Lag: r.Bounded("lag", w.info.maxAge), SourceAge: r.Bounded("source age", rs.tick),
		}
		if r.Err() != nil {
			return
		}
		last := len(rumours) - 1
		if rm.Price < 1 || rm.Lag < 2 || last >= 0 && (rumours[last].Region > rm.Region || rumours[last].Region == rm.Region && rumours[last].Commodity >= rm.Commodity) {
			r.Failf("tag 22 row for agent %d holds an impossible rumour %+v", id, rm)
			return
		}
		rumours = append(rumours, rm)
	}
	for int(id) >= len(w.integ.views) {
		w.integ.views = append(w.integ.views, rumourView{})
	}
	w.integ.views[id] = rumourView{has: true, tick: tick, rumours: rumours}
}
