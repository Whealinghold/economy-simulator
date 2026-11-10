package economysim

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"slices"

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
