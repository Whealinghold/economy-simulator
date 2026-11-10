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
	slices.SortFunc(stock, func(a, b itemRow) int { return cmp.Compare(a.commodity, b.commodity) }) // the order of the input changes no ledger order
	// The belief seeds read the cost anchors, which are scratch and not saved: derive
	// them from the saved state as Snapshot does, so a restored world agrees.
	if _, err := w.prepareAnchors(); err != nil {
		return NoAgent, fmt.Errorf("AddAgent: %w", err)
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
	a, err := w.queryAgent(id)
	if err == nil && !a.external {
		err = ErrNotExternal
	}
	if err != nil {
		return nil, err
	}
	return a, nil
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

// integrationState is what INT keeps of the world's own (plan 2-17): the agents
// position P1.9 visits.
type integrationState struct {
	// externals are the slots, ascending, of the agents P1.9 reads a mailbox of:
	// the external agents and any agent that holds a directive (only the test seam
	// gives a household one). It is derived from the agents, so it is not saved
	// and a restore rebuilds it (rebuildExternals); a removed external agent leaves it at RemoveAgent, any other
	// removed agent at the next P1.9. tick.go calls P1.9 only while the list is
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

// unwatch drops slot from the agents P1.9 visits.
func (s *integrationState) unwatch(slot int32) {
	if i, found := slices.BinarySearch(s.externals, slot); found {
		s.externals = slices.Delete(s.externals, i, i+1)
	}
	s.visit = len(s.externals) > 0
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
	// reseedAtArrival reads the cost anchors, which are scratch: derive them from the
	// saved state, so a restored world agrees (as in AddAgent).
	if _, err := w.prepareAnchors(); err != nil {
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
	w.integ.dropView(a.id) // the rumours were read in the old region
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

// externalRoleChanged is the step of a role change for a mortal external agent the
// spawner switched: the mailbox was checked against the old role, so it is emptied,
// and the last view is dropped, so neither comes back with a later role.
func (w *World) externalRoleChanged(a *agent) {
	a.directive = nil
	w.integ.dropView(a.id)
}

// dropExternal is the step of a removal for an external agent, whatever the cause:
// its view goes and P1.9 stops visiting it, so a restore, which rebuilds the list
// from live agents, agrees with the original world.
func (w *World) dropExternal(a *agent) {
	if a.external {
		w.integ.dropView(a.id)
		w.integ.unwatch(int32(a.id))
	}
}

// dropView forgets the last view of an agent and frees its rumours.
func (s *integrationState) dropView(id AgentID) {
	if int(id) < len(s.views) {
		s.views[id] = rumourView{}
	}
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
		rm := w.readRumour(rs, r)
		if r.Err() != nil {
			return
		}
		if !validRumour(rm, rumours) {
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

// readRumour reads one rumour of tag 22 against its own bounds.
func (w *World) readRumour(rs *restoreCtx, r *codec.Reader) Rumour {
	return Rumour{
		Region: RegionID(r.Index(uint64(len(w.cfg.Regions)))), Commodity: CommodityID(r.Index(uint64(len(w.cfg.Commodities)))),
		Price: Price(r.Bounded("price", MaxPrice)), Mean: Price(r.Bounded("mean", MaxPrice)),
		Lag: r.Bounded("lag", w.info.maxAge), SourceAge: r.Bounded("source age", rs.tick),
	}
}

// validRumour says rm is a rumour a view can hold after the rumours before it: a
// price of at least 1, a lag of at least 2, and strictly ascending by (region,
// commodity), which is the order of a view.
func validRumour(rm Rumour, before []Rumour) bool {
	if rm.Price < 1 || rm.Lag < 2 {
		return false
	}
	if len(before) == 0 {
		return true
	}
	p := before[len(before)-1]
	return p.Region < rm.Region || p.Region == rm.Region && p.Commodity < rm.Commodity
}
