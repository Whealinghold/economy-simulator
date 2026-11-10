package economysim

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"slices"

	"github.com/Whealinghold/economy-sim/internal/checked"
	"github.com/Whealinghold/economy-sim/internal/ledger"
	"github.com/Whealinghold/economy-sim/internal/logkeys"
	"github.com/Whealinghold/economy-sim/internal/outside"
)

// This file is the external-sector module (spec-external-sector.md;
// architecture.md 2.10.1): the canonical form of its configuration, its
// validator and its warnings. The arithmetic of the price and the quantities is
// in internal/outside. Outside this file the module appears only in the
// catalogue files (config.go, hash.go, bounds.go, defaults.go, errors.go,
// validate.go) and at the named hooks of ADR 0010.

// external is the state of the module, nil while it is off (spec-external-sector.md
// 16). The shapes are flat slices in the canonical order of the configuration,
// with no map (D529): the lines by (node, commodity), the counterparties by
// (node, gate, side with buy before sell) and the side rows by counterparty and
// then by line. The mid and the quotes are not stored: they are functions of the
// tick, the drift and the shock.
type external struct {
	// lines are the lines of every node, ascending by node then commodity.
	lines []outsideLine
	// cps are the counterparties in agent ID order (spec-external-sector.md 4.2.2).
	cps []counterparty
	// rows are the side rows (3.3.4): the rows of counterparty i are
	// rows[cps[i].first : cps[i].first+cps[i].n], by commodity.
	rows []sideRow
	// ticksPerYear is the length of the year in ticks, for the trend.
	ticksPerYear int64
}

// outsideLine is the runtime state of one line, and the means to find its
// configuration.
type outsideLine struct {
	node      RegionID
	commodity CommodityID
	cfg       *OutsideLine
	// drift is the walk state, 0 without a walk (6.4); shock is the shock slot,
	// 10 000 until set (6.5).
	drift, shock int64
	// mid and clamped are the mid of the tick and whether a clamp acted, set at P3.5.
	mid     int64
	clamped bool
}

// counterparty is a pseudo-agent bound to one Outside account: the buyer or the
// seller of one gate of one node (4.2.1).
type counterparty struct {
	node, gate RegionID
	// buy is true for the counterparty of a buy side, which places bids.
	buy bool
	// agent is its agent ID, in the range between the estates and the households.
	agent AgentID
	// account is its ledger account, 0 until openAccounts or attachAccounts.
	account int
	// scaleBP is the gate scale of its quotes (3.2.4).
	scaleBP int64
	// first and n are its rows.
	first, n int
}

// sideRow is the state of one side at one gate (3.3.4): the carry of the wanted
// units, the units and the list value traded over the run, and whether it ended.
// t is the scratch of the tick, written at P3.5 and read and cleared at P4.2.
type sideRow struct {
	line                    int
	carry, units, listValue int64
	ended                   bool
	t                       rowTick
}

// rowTick is what a side row knows of the tick between its order and its sweep:
// the quote, the units wanted and offered, the cuts, the money of its trades (from
// the trades of P4.1), and what the sweep found.
type rowTick struct {
	mid, limit, wanted, offered int64
	cutClosed, cutCap, cutPrice int64
	clamped                     bool
	// paid is the money of the row's trades, by the trades of the tick.
	paid int64
	// traded and value are set by the sweep; endCause is the cause when the row ended
	// in this tick, 0 otherwise.
	traded, value, listValue int64
	endCause                 int
}

// newExternal builds the state of the module from the canonical configuration:
// the lines, and the counterparties with their rows. It returns nil while the
// module is off. Agent IDs are assigned after the estates, in ascending
// (node, gate, side) with the buy side before the sell side of a gate. No draw is
// taken and no account is opened (FR-EXT-044).
func newExternal(c *Config) *external {
	m := c.Modules.External
	if m == nil {
		return nil
	}
	x := &external{ticksPerYear: c.Calendar.TicksPerYear()}
	for ni := range m.Nodes {
		n := &m.Nodes[ni]
		node, _ := c.regionID(n.Region)
		first := len(x.lines)
		for li := range n.Lines {
			commodity, _ := c.commodityID(n.Lines[li].Commodity)
			x.lines = append(x.lines, outsideLine{node: node, commodity: commodity, cfg: &n.Lines[li], shock: wholeBP})
		}
		x.addCounterparties(c, n, node, first)
	}
	return x
}

// addCounterparties appends the counterparties of one node: for each gate in
// ascending order, the buyer and then the seller, each only if some line has the
// side present and the gate is a gate for it (4.2.1). first is the index of the
// node's first line.
func (x *external) addCounterparties(c *Config, n *OutsideNode, node RegionID, first int) {
	sell, buy := edgeGates(c, n.Region)
	hasSell, hasBuy := n.sides()
	gates := slices.Compact(slices.Sorted(slices.Values(slices.Concat(sell, buy))))
	for _, g := range gates {
		gate, _ := c.regionID(g)
		scale := n.gateScale(g)
		if hasBuy && slices.Contains(buy, g) {
			x.addCounterparty(counterparty{node: node, gate: gate, buy: true, scaleBP: scale}, n, first)
		}
		if hasSell && slices.Contains(sell, g) {
			x.addCounterparty(counterparty{node: node, gate: gate, scaleBP: scale}, n, first)
		}
	}
}

// addCounterparty appends a counterparty and its rows: one for each line of the
// node whose side is present, in commodity order. The agent ID is set later, by
// the number of regions.
func (x *external) addCounterparty(cp counterparty, n *OutsideNode, firstLine int) {
	cp.first = len(x.rows)
	for li := range n.Lines {
		s := &n.Lines[li].Sell
		if cp.buy {
			s = &n.Lines[li].Buy
		}
		if s.present() {
			x.rows = append(x.rows, sideRow{line: firstLine + li})
		}
	}
	cp.n = len(x.rows) - cp.first
	x.cps = append(x.cps, cp)
}

// counterpartyCount is the number of counterparties, 0 for a nil module.
func (x *external) counterpartyCount() int {
	if x == nil {
		return 0
	}
	return len(x.cps)
}

// assignAgents gives the counterparties the agent IDs from first on, in order.
func (x *external) assignAgents(first int) {
	for i := range x.cps {
		x.cps[i].agent = AgentID(first + i)
	}
}

// openAccounts opens the Outside account of each counterparty, in agent ID order,
// right after the estate accounts and before any agent's (spec-external-sector.md
// 4.2.3). The owner of the account is the agent ID and its region the gate.
func (x *external) openAccounts(l *ledger.Ledger) error {
	for i := range x.cps {
		cp := &x.cps[i]
		id, err := l.Open(ledger.KindOutside, int(cp.agent), int(cp.gate))
		if err != nil {
			return fmt.Errorf("open the account of counterparty %d: %w", cp.agent, err)
		}
		cp.account = id
	}
	return nil
}

// attachAccounts finds the Outside account of each counterparty in a ledger that a
// restore has read: the accounts right after the estates, in agent ID order, each
// open, of the kind, owned by the agent and in the gate region.
func (x *external) attachAccounts(l *ledger.Ledger, estates *estates) error {
	first := estates.accounts[0] + len(estates.accounts)
	for i := range x.cps {
		cp := &x.cps[i]
		id := first + i
		v, err := l.Account(id)
		if err != nil || v.Kind != ledger.KindOutside || v.Closed || v.Owner != int(cp.agent) || v.Region != int(cp.gate) {
			return fmt.Errorf("%w: account %d is not the open Outside account of counterparty %d", ErrCheckpointCorrupt, id, cp.agent)
		}
		cp.account = id
	}
	return nil
}

// outsideMarks returns the outside nodes by RegionID: all false while the module
// is off (D526).
func outsideMarks(c *Config) []bool {
	marks := make([]bool, len(c.Regions))
	if c.Modules.External == nil {
		return marks
	}
	for i := range c.Modules.External.Nodes {
		if r, ok := c.regionID(c.Modules.External.Nodes[i].Region); ok {
			marks[r] = true
		}
	}
	return marks
}

// isOutsideNode reports that the region is an outside node: a region that is not
// a market (spec-external-sector.md 3.1). It is false for an ID that is no region.
func (w *World) isOutsideNode(r RegionID) bool {
	return r >= 0 && int(r) < len(w.outside) && w.outside[r]
}

// outsideReasons returns the six ledger reasons the module declares while it is on
// (spec-external-sector.md 8.1), or nil while it is off. They are declared reasons:
// a ledger has them only when Config.Declared names them (D522).
func outsideReasons(c *Config) []ledger.Reason {
	if c.Modules.External == nil {
		return nil
	}
	return []ledger.Reason{
		ledger.OutsideBudget, ledger.OutsideUnspent, ledger.OutsideReceipts,
		ledger.OutsideOffer, ledger.OutsideUnsold, ledger.OutsidePurchases,
	}
}

// outsideGroup is the plan of group 3 of the boundary accounts: the commodities
// that some line has a side for, ascending (spec-external-sector.md 4.2.4).
type outsideGroup struct{ goods []int }

// outsideGroupOf returns the plan of group 3 for c, nil while the module is off.
func outsideGroupOf(c *Config) *outsideGroup {
	m := c.Modules.External
	if m == nil {
		return nil
	}
	has := make([]bool, len(c.Commodities))
	for ni := range m.Nodes {
		for li := range m.Nodes[ni].Lines {
			l := &m.Nodes[ni].Lines[li]
			if k, ok := c.commodityID(l.Commodity); ok && (l.Sell.present() || l.Buy.present()) {
				has[k] = true
			}
		}
	}
	g := &outsideGroup{}
	for k, h := range has {
		if h {
			g.goods = append(g.goods, k)
		}
	}
	return g
}

// open opens the boundary accounts of group 3: the money reasons first, then the
// goods reasons for each commodity that has a line side, in commodity order. It
// opens nothing for a nil plan.
func (g *outsideGroup) open(l *ledger.Ledger) error {
	if g == nil {
		return nil
	}
	money := []ledger.Reason{ledger.OutsideBudget, ledger.OutsideUnspent, ledger.OutsideReceipts}
	goods := []ledger.Reason{ledger.OutsideOffer, ledger.OutsideUnsold, ledger.OutsidePurchases}
	for _, r := range money {
		if err := openBoundary(l, r, ledger.Money); err != nil {
			return err
		}
	}
	for _, k := range g.goods {
		for _, r := range goods {
			if err := openBoundary(l, r, k); err != nil {
				return err
			}
		}
	}
	return nil
}

// audit is the module's check of World.Audit (spec-external-sector.md 8.5,
// FR-EXT-050): every counterparty account is empty, the boundary balance of each
// of the six reasons has the sign of its kind, and for each commodity the balance
// of outside_purchases equals the units of the buy rows and that of outside_offer
// less outside_unsold the units of the sell rows, exactly. A violation names the
// node and the commodity.
func (x *external) audit(w *World) error {
	if err := x.auditEmpty(w); err != nil {
		return err
	}
	if err := x.auditState(w); err != nil {
		return err
	}
	if err := x.auditMoney(w); err != nil {
		return err
	}
	for k := range w.cfg.Commodities {
		if err := x.auditUnits(w, CommodityID(k)); err != nil {
			return err
		}
	}
	return nil
}

// auditState checks the saved state of the module against its bounds: the drift of
// a line within its bound, the shock in 1 to MaxShockBP, the carry of a side row in
// [0, 10 000) and its totals not negative.
func (x *external) auditState(w *World) error {
	for i := range x.lines {
		l := &x.lines[i]
		if l.drift < -l.cfg.WalkBoundBP || l.drift > l.cfg.WalkBoundBP || l.shock < 1 || l.shock > MaxShockBP {
			return fmt.Errorf("outside node %q, commodity %q: drift %d (bound %d) or shock %d is out of its bounds", w.cfg.Regions[l.node].ID, l.cfg.Commodity, l.drift, l.cfg.WalkBoundBP, l.shock)
		}
	}
	return x.auditRows(w)
}

// auditRows checks the side rows of auditState.
func (x *external) auditRows(w *World) error {
	for i := range x.cps {
		cp := &x.cps[i]
		for j := cp.first; j < cp.first+cp.n; j++ {
			r := &x.rows[j]
			if r.carry < 0 || r.carry >= wholeBP || r.units < 0 || r.listValue < 0 {
				return fmt.Errorf("outside node %q, commodity %q: side row has carry %d, units %d, list value %d out of bounds",
					w.cfg.Regions[cp.node].ID, x.lines[r.line].cfg.Commodity, r.carry, r.units, r.listValue)
			}
		}
	}
	return nil
}

// auditEmpty checks that no counterparty account holds money or goods.
func (x *external) auditEmpty(w *World) error {
	for i := range x.cps {
		cp := &x.cps[i]
		name := w.cfg.Regions[cp.node].ID
		v, err := w.ledger.Account(cp.account)
		if err != nil {
			invariant("a counterparty account is unreadable: " + err.Error())
		}
		if v.Balance != 0 || v.Reserved != 0 {
			return fmt.Errorf("outside node %q, gate %q: counterparty %d holds money %d (reserved %d), want an empty account", name, w.cfg.Regions[cp.gate].ID, cp.agent, v.Balance, v.Reserved)
		}
		for k := range w.cfg.Commodities {
			if held, _, err := w.ledger.Holding(cp.account, k); err != nil || held != 0 {
				return fmt.Errorf("outside node %q, gate %q: counterparty %d holds %d units of %q, want an empty account", name, w.cfg.Regions[cp.gate].ID, cp.agent, held, w.cfg.Commodities[k].ID)
			}
		}
	}
	return nil
}

// boundaryBalance returns the balance of the boundary account of a reason and a
// thing, and 0 when the account is not open (no line deals in the commodity).
func (w *World) boundaryBalance(r ledger.Reason, thing int) int64 {
	id, ok := w.ledger.Boundary(r, thing)
	if !ok {
		return 0
	}
	v, err := w.ledger.Account(id)
	if err != nil {
		invariant("a boundary account is unreadable: " + err.Error())
	}
	return v.Balance
}

// auditMoney checks the signs of the three money reasons: a faucet's balance is
// not positive and a sink's is not negative.
func (x *external) auditMoney(w *World) error {
	if w.boundaryBalance(ledger.OutsideBudget, ledger.Money) > 0 {
		return fmt.Errorf("outside_budget: boundary balance %d is positive, want a faucet's balance (not positive)", w.boundaryBalance(ledger.OutsideBudget, ledger.Money))
	}
	for _, r := range []ledger.Reason{ledger.OutsideUnspent, ledger.OutsideReceipts} {
		if b := w.boundaryBalance(r, ledger.Money); b < 0 {
			return fmt.Errorf("reason %d (outside money sink): boundary balance %d is negative, want a sink's balance (not negative)", r, b)
		}
	}
	return nil
}

// auditUnits checks the signs and the units of one commodity.
func (x *external) auditUnits(w *World, k CommodityID) error {
	offer, unsold, bought := w.boundaryBalance(ledger.OutsideOffer, int(k)), w.boundaryBalance(ledger.OutsideUnsold, int(k)), w.boundaryBalance(ledger.OutsidePurchases, int(k))
	name := w.cfg.Commodities[k].ID
	if offer > 0 || unsold < 0 || bought < 0 {
		return fmt.Errorf("commodity %q: outside boundary balances offer %d, unsold %d, purchases %d have the wrong sign for their kinds", name, offer, unsold, bought)
	}
	var sellUnits, buyUnits int64
	node := ""
	for i := range x.cps {
		cp := &x.cps[i]
		for j := cp.first; j < cp.first+cp.n; j++ {
			row := &x.rows[j]
			if x.lines[row.line].commodity != k {
				continue
			}
			node = w.cfg.Regions[cp.node].ID
			if cp.buy {
				buyUnits += row.units
			} else {
				sellUnits += row.units
			}
		}
	}
	if bought != buyUnits {
		return fmt.Errorf("outside node %q, commodity %q: purchases balance %d, want the units of the buy sides %d", node, name, bought, buyUnits)
	}
	if -offer-unsold != sellUnits {
		return fmt.Errorf("outside node %q, commodity %q: offer less unsold is %d, want the units of the sell sides %d", node, name, -offer-unsold, sellUnits)
	}
	return nil
}

// outsideNodeRow is row 5a of spec-regions.md 6.1.3: the destination is an outside
// node, and no earlier row (closed_account, same_region) has refused the
// relocation. reason is the ledger's answer for rows 5 and 6.
func (w *World) outsideNodeRow(to RegionID, reason Reason) bool {
	return w.isOutsideNode(to) && reason != ReasonClosedAccount && reason != ReasonSameRegion
}

// outsideMarket reports that the market at dense index i is the slot of an outside
// node, which has a slot but no market in effect (D526).
func (w *World) outsideMarket(i int) bool {
	return w.external != nil && w.isOutsideNode(RegionID(i/w.markets.commodities))
}

// dropOutsideRows removes the Roles and Markets rows of the outside nodes from a
// snapshot built densely by (region, role) and (region, commodity): a node has no
// such row (spec-observability.md 5.3.4, 5.3.5). It keeps the order of the rest.
func (w *World) dropOutsideRows(s *Snapshot) {
	if w.external == nil {
		return
	}
	nr, nc := len(w.cfg.Roles), len(w.cfg.Commodities)
	roles, markets := s.Roles[:0], s.Markets[:0]
	for g := range w.cfg.Regions {
		if w.isOutsideNode(RegionID(g)) {
			continue
		}
		roles = append(roles, s.Roles[g*nr:(g+1)*nr]...)
		markets = append(markets, s.Markets[g*nc:(g+1)*nc]...)
	}
	s.Roles, s.Markets = roles, markets
}

// canonical returns the module with nodes by region, gates by region, lines by
// commodity and season factors by the order of the calendar's seasons (a season
// the calendar lacks last, by ID); the receiver is not changed. It returns nil
// for a nil module (spec-external-sector.md 3.3.2).
func (m *External) canonical(calendar []Season) *External {
	if m == nil {
		return nil
	}
	index := func(season string) int {
		if i := slices.IndexFunc(calendar, func(s Season) bool { return s.ID == season }); i >= 0 {
			return i
		}
		return len(calendar)
	}
	bySeason := func(a, b SeasonFactor) int {
		return cmp.Or(cmp.Compare(index(a.Season), index(b.Season)), cmp.Compare(a.Season, b.Season))
	}
	out := &External{Nodes: sortedBy(m.Nodes, func(a, b OutsideNode) int { return cmp.Compare(a.Region, b.Region) })}
	for i := range out.Nodes {
		n := &out.Nodes[i]
		n.Gates = sortedBy(n.Gates, func(a, b OutsideGate) int { return cmp.Compare(a.Region, b.Region) })
		n.Lines = sortedBy(n.Lines, func(a, b OutsideLine) int { return cmp.Compare(a.Commodity, b.Commodity) })
		for j := range n.Lines {
			l := &n.Lines[j]
			l.PriceBySeason = sortedBy(l.PriceBySeason, bySeason)
			l.Sell.Window = sortedBy(l.Sell.Window, bySeason)
			l.Buy.Window = sortedBy(l.Buy.Window, bySeason)
		}
	}
	return out
}

// gateScale returns the scale of the gate region g of the node: its row, else the
// neutral default (spec-external-sector.md 3.2.4).
func (n *OutsideNode) gateScale(g string) int64 {
	if i := slices.IndexFunc(n.Gates, func(o OutsideGate) bool { return o.Region == g }); i >= 0 {
		return n.Gates[i].ScaleBP
	}
	return DefaultGateScaleBP
}

// present reports that a side exists: a Capacity of 0 is an absent side
// (spec-external-sector.md 3.3.3).
func (s *OutsideSide) present() bool { return s.Capacity > 0 }

// edgeGates returns the gates of a node read from the edges as given
// (spec-external-sector.md 3.2.1): sell are the regions the node has a direct
// edge to (the outside sells there), buy the regions with a direct edge to the
// node (the outside buys there). Both are in ascending order, without repeats.
func edgeGates(c *Config, node string) (sell, buy []string) {
	for _, e := range c.Edges {
		switch {
		case e.From == node && e.To != node:
			sell = append(sell, e.To)
		case e.To == node && e.From != node:
			buy = append(buy, e.From)
		}
	}
	slices.Sort(sell)
	slices.Sort(buy)
	return slices.Compact(sell), slices.Compact(buy)
}

// outsideNodeSet returns the regions the module marks as outside, nil when the
// module is off.
func outsideNodeSet(c *Config) map[string]bool {
	if c.Modules.External == nil {
		return nil
	}
	set := make(map[string]bool, len(c.Modules.External.Nodes))
	for i := range c.Modules.External.Nodes {
		set[c.Modules.External.Nodes[i].Region] = true
	}
	return set
}

// sides reports whether some line of the node has a sell side present, and
// whether some has a buy side present.
func (n *OutsideNode) sides() (sell, buy bool) {
	for i := range n.Lines {
		sell = sell || n.Lines[i].Sell.present()
		buy = buy || n.Lines[i].Buy.present()
	}
	return sell, buy
}

// windowAllClosed reports that a window is 0 in every season of the calendar
// (spec-external-sector.md 7.2): every season has a row and every row is 0. A
// calendar with no season has no such window.
func windowAllClosed(w []SeasonFactor, seasons []Season) bool {
	if len(seasons) == 0 {
		return false
	}
	for _, s := range seasons {
		if !slices.ContainsFunc(w, func(f SeasonFactor) bool { return f.Season == s.ID && f.BP == 0 }) {
			return false
		}
	}
	return true
}

// importSourced reports that the commodity is sourced by an import
// (spec-external-sector.md 10.1.1, D536): some node has a line for it with a sell
// side present, at least one gate of the node for the sell side, and a window
// that is not 0 in every season. The cap does not matter. It reads c as given
// and never fails.
func importSourced(c *Config, commodity string) bool {
	if c.Modules.External == nil {
		return false
	}
	for i := range c.Modules.External.Nodes {
		n := &c.Modules.External.Nodes[i]
		sell, _ := edgeGates(c, n.Region)
		if len(sell) == 0 {
			continue
		}
		for j := range n.Lines {
			l := &n.Lines[j]
			if l.Commodity == commodity && l.Sell.present() && !windowAllClosed(l.Sell.Window, c.Calendar.Seasons) {
				return true
			}
		}
	}
	return false
}

// validateExternal checks the external module (spec-external-sector.md 14.1,
// FR-CFG-075). It reports nothing when the module is off.
func (v *validator) validateExternal() {
	m := v.c.Modules.External
	if m == nil {
		return
	}
	base := []string{"modules", "external"}
	if len(v.c.Edges) == 0 {
		v.rep.add(ErrModuleDependency, base, "edges", "")
	}
	if len(m.Nodes) > MaxOutsideNodes {
		v.rep.add(ErrOutOfBounds, path(base, "node"), itoa(len(m.Nodes)), itoa(MaxOutsideNodes))
		return // the rest is linear in the count it just refused
	}
	seen := make(map[string]bool, len(m.Nodes))
	counterparties := 0
	for i := range m.Nodes {
		counterparties += v.outsideNode(base, &m.Nodes[i], seen)
	}
	if counterparties > MaxCounterparties {
		v.rep.add(ErrOutOfBounds, path(base, "counterparties"), itoa(counterparties), itoa(MaxCounterparties))
	}
	v.outsideInhabited(m)
}

// outsideNode checks one node, its gates and its lines, and returns the number
// of counterparties the node has: one for each gate of each side that some line
// of the node has present (spec-external-sector.md 4.2.1).
func (v *validator) outsideNode(base []string, n *OutsideNode, seen map[string]bool) int {
	nb := path(base, "node", n.Region)
	if v.lk.region[n.Region] == nil {
		v.rep.add(ErrUnknownRegion, path(nb, "region"), n.Region, "")
	}
	if seen[n.Region] {
		v.rep.add(ErrDuplicateID, path(nb, "region"), n.Region, "")
	}
	seen[n.Region] = true
	v.label(nb, "label", n.Label)
	sell, buy := edgeGates(&v.c, n.Region)
	v.outsideGates(nb, n, sell, buy)
	outside := v.outsideSet()
	for _, g := range slices.Concat(sell, buy) {
		if outside[g] { // no market to clear in: the side would never trade
			v.rep.add(ErrNotAGate, path(nb, "edge", g), g, n.Region)
		}
	}
	if len(n.Lines) > MaxOutsideLines {
		v.rep.add(ErrOutOfBounds, path(nb, "line"), itoa(len(n.Lines)), itoa(MaxOutsideLines))
		return 0 // the rest is linear in the count it just refused
	}
	lines := make(map[string]bool, len(n.Lines))
	for i := range n.Lines {
		v.outsideLine(nb, &n.Lines[i], lines)
	}
	hasSell, hasBuy := n.sides()
	count := 0
	if hasSell {
		count += len(sell)
	}
	if hasBuy {
		count += len(buy)
	}
	return count
}

// outsideSet is outsideNodeSet(&v.c), built once per validation.
func (v *validator) outsideSet() map[string]bool {
	if v.outsideNodes == nil {
		v.outsideNodes = outsideNodeSet(&v.c)
	}
	return v.outsideNodes
}

// outsideGates checks the gate rows of a node: a defined region, once, that is a
// gate of the node, with a scale in 1 to MaxGateScaleBP (3.2.4).
func (v *validator) outsideGates(nb []string, n *OutsideNode, sell, buy []string) {
	seen := make(map[string]bool, len(n.Gates))
	for i := range n.Gates {
		g := &n.Gates[i]
		gb := path(nb, "gate", g.Region)
		known := v.lk.region[g.Region] != nil
		if !known {
			v.rep.add(ErrUnknownRegion, path(gb, "region"), g.Region, "")
		}
		if seen[g.Region] {
			v.rep.add(ErrDuplicateID, path(gb, "region"), g.Region, "")
		}
		seen[g.Region] = true
		if known && !slices.Contains(sell, g.Region) && !slices.Contains(buy, g.Region) {
			v.rep.add(ErrNotAGate, path(gb, "region"), g.Region, n.Region)
		}
		v.inRange(path(gb, "scale_bp"), g.ScaleBP, 1, MaxGateScaleBP)
	}
}

// outsideLine checks one line: its commodity, its price path, its tables and its
// two sides (spec-external-sector.md 13.1, 6.6.1).
func (v *validator) outsideLine(nb []string, l *OutsideLine, seen map[string]bool) {
	lb := path(nb, "line", l.Commodity)
	if v.lk.commodity[l.Commodity] == nil {
		v.rep.add(ErrUnknownCommodity, path(lb, "commodity"), l.Commodity, "")
	}
	if seen[l.Commodity] {
		v.rep.add(ErrDuplicateID, path(lb, "commodity"), l.Commodity, "")
	}
	seen[l.Commodity] = true
	v.ranges(lb, []fieldRange{
		{"base", int64(l.Base), 1, MaxPrice},
		{"trend_bp", l.TrendBP, -MaxTrendBP, MaxTrendBP},
		{"floor_bp", l.FloorBP, 1, wholeBP},
		{"ceiling_bp", l.CeilingBP, wholeBP, MaxFactorBP},
		{"walk_step_bp", l.WalkStepBP, 0, MaxWalkStepBP},
		{"walk_revert_bp", l.WalkRevertBP, 0, wholeBP},
	})
	v.walkBound(lb, l)
	v.seasonTable(path(lb, "price_by_season"), l.PriceBySeason, 1, MaxPriceFactorBP)
	v.outsideSide(path(lb, "sell"), &l.Sell, MaxSellMarginBP)
	v.outsideSide(path(lb, "buy"), &l.Buy, MaxBuyMarginBP)
	if l.Sell.present() && l.Buy.present() && l.Sell.MarginBP >= 0 && l.Buy.MarginBP >= 0 &&
		l.Sell.MarginBP+l.Buy.MarginBP < MinQuoteSpreadBP {
		v.rep.add(ErrCrossedQuotes, lb, i64(l.Sell.MarginBP+l.Buy.MarginBP), itoa(MinQuoteSpreadBP))
	}
}

// walkBound checks WalkBoundBP: WalkStepBP to MaxDriftBP when a walk is on, and
// 0 when it is not (spec-external-sector.md 6.4.1, 14.1).
func (v *validator) walkBound(lb []string, l *OutsideLine) {
	p := path(lb, "walk_bound_bp")
	if l.WalkStepBP > 0 {
		v.inRange(p, l.WalkBoundBP, l.WalkStepBP, MaxDriftBP)
		return
	}
	if l.WalkBoundBP != 0 {
		v.rep.add(ErrOutOfBounds, p, i64(l.WalkBoundBP), "0")
	}
}

// outsideSide checks one side. An absent side (Capacity 0) must have every other
// field 0 (3.3.3); a present one has its margin in 0 to maxMargin, its totals in
// 0 to MaxLineTotal and its window in 0 to 10 000 by known seasons.
func (v *validator) outsideSide(sb []string, s *OutsideSide, maxMargin int64) {
	if !v.inRange(path(sb, "capacity"), s.Capacity, 0, MaxQuantity) {
		return
	}
	if !s.present() {
		v.absentSide(sb, s)
		return
	}
	v.ranges(sb, []fieldRange{
		{"margin_bp", s.MarginBP, 0, maxMargin},
		{"total_units", s.TotalUnits, 0, MaxLineTotal},
		{"total_value", s.TotalValue, 0, MaxLineTotal},
	})
	v.seasonTable(path(sb, "window"), s.Window, 0, wholeBP)
}

// absentSide reports every field set on an absent side as ErrOutOfBounds
// (spec-external-sector.md 3.3.3): a margin with no side is a mistake.
func (v *validator) absentSide(sb []string, s *OutsideSide) {
	for _, f := range []fieldRange{
		{"margin_bp", s.MarginBP, 0, 0}, {"total_units", s.TotalUnits, 0, 0}, {"total_value", s.TotalValue, 0, 0},
		{"window", int64(len(s.Window)), 0, 0},
	} {
		v.inRange(path(sb, f.name), f.value, f.lo, f.hi)
	}
}

// seasonTable checks a table of season factors: each season defined by the
// calendar (ErrUnknownSeason, also for any season when the calendar has none),
// once (ErrDuplicateID), with the factor in lo to hi.
func (v *validator) seasonTable(p []string, rows []SeasonFactor, lo, hi int64) {
	seen := make(map[string]bool, len(rows))
	for _, r := range rows {
		rp := path(p, r.Season)
		if !slices.ContainsFunc(v.c.Calendar.Seasons, func(s Season) bool { return s.ID == r.Season }) {
			v.rep.add(ErrUnknownSeason, path(rp, "season"), r.Season, "")
		}
		if seen[r.Season] {
			v.rep.add(ErrDuplicateID, path(rp, "season"), r.Season, "")
		}
		seen[r.Season] = true
		v.inRange(path(rp, "bp"), r.BP, lo, hi)
	}
}

// outsideInhabited reports ErrOutsideNodeInhabited for every table row that names
// an outside node (spec-external-sector.md 3.1.2): a group, a warm-start row, a
// population region row, a taste row or a money-flow target. A later module that
// adds a table of rows by region adds its check here.
func (v *validator) outsideInhabited(m *External) {
	nodes := v.outsideSet()
	if len(nodes) == 0 {
		return
	}
	hit := func(region string, table ...string) {
		if nodes[region] {
			v.rep.add(ErrOutsideNodeInhabited, table, region, "")
		}
	}
	c := &v.c
	for i := range c.Groups {
		hit(c.Groups[i].Region, "groups", c.Groups[i].Region, c.Groups[i].Role)
	}
	if w := c.WarmStart; w != nil {
		for _, r := range w.Prices {
			hit(r.Region, "warm_start", "price", r.Region, r.Commodity)
		}
		for _, r := range w.Wages {
			hit(r.Region, "warm_start", "wage", r.Region)
		}
		for _, r := range w.Groups {
			hit(r.Region, "warm_start", "group", r.Region, r.Role)
		}
		for _, r := range w.Pools {
			hit(r.Region, "warm_start", "pool", r.Region)
		}
	}
	v.outsideInhabitedModules(hit)
}

// outsideInhabitedModules is outsideInhabited for the tables of the modules: the
// population region rows, the taste rows and the targets of the money flows.
func (v *validator) outsideInhabitedModules(hit func(region string, table ...string)) {
	mods := &v.c.Modules
	if mods.Population != nil {
		for _, r := range mods.Population.Regions {
			hit(r.Region, "modules", "population", "region", r.Region)
		}
	}
	if mods.Demand != nil {
		for _, r := range mods.Demand.RegionTastes {
			hit(r.Region, "modules", "demand", "region_taste", r.Region, r.Want)
		}
	}
	if mods.MoneySupply != nil {
		for _, f := range mods.MoneySupply.Flows {
			hit(f.Target.Region, "modules", "money_supply", "flow", f.ID, "target", "region")
		}
	}
}

// externalWarnings returns the warnings of spec-external-sector.md 14.2 for c
// read as given. It never fails: an ID that is not declared is skipped.
func externalWarnings(c *Config, lk lookup) []Warning {
	if c.Modules.External == nil {
		return nil
	}
	w := &outsideWarner{c: c, lk: lk}
	w.sets = tradableSets(c, lk)
	for i := range c.Modules.External.Nodes {
		w.node(&c.Modules.External.Nodes[i])
	}
	w.global()
	return w.out
}

// outsideWarner collects the warnings of the module. The quotes it gathers are
// the ask and the bid at Base for each gate, by commodity, for the arbitrage
// test (6.6.1).
type outsideWarner struct {
	c       *Config
	lk      lookup
	sets    []map[string]bool
	out     []Warning
	asks    map[string][]gateQuote
	bids    map[string][]gateQuote
	anyBuy  bool
	anySell bool
}

// gateQuote is the quote of one gate at the base price.
type gateQuote struct {
	gate  string
	price int64
}

func (w *outsideWarner) add(rule string, segs ...string) {
	w.out = append(w.out, Warning{Path: segs, Rule: rule})
}

// node raises the warnings of one node and its lines.
func (w *outsideWarner) node(n *OutsideNode) {
	nb := []string{"modules", "external", "node", n.Region}
	sell, buy := edgeGates(w.c, n.Region)
	hasSell, hasBuy := n.sides()
	if (hasSell || hasBuy) && !(hasSell && len(sell) > 0) && !(hasBuy && len(buy) > 0) {
		w.add("node_without_gate", nb...)
	}
	for i := range n.Lines {
		w.line(nb, n, &n.Lines[i], sell, buy)
	}
}

// line raises the warnings of one line.
func (w *outsideWarner) line(nb []string, n *OutsideNode, l *OutsideLine, sell, buy []string) {
	lb := path(nb, "line", l.Commodity)
	if !l.Sell.present() && !l.Buy.present() {
		w.add("line_without_side", lb...)
	}
	w.anySell = w.anySell || l.Sell.present()
	w.anyBuy = w.anyBuy || l.Buy.present()
	w.direction(lb, l, sell, buy)
	w.side(lb, n, l, "sell", &l.Sell, sell)
	w.side(lb, n, l, "buy", &l.Buy, buy)
	w.walk(lb, l)
	if !slices.ContainsFunc(w.sets, func(set map[string]bool) bool { return set[l.Commodity] }) {
		w.add("line_untraded", lb...)
	}
}

// direction raises gate_edge_wrong_direction for a side whose needed edge is
// absent while the opposite edge exists.
func (w *outsideWarner) direction(lb []string, l *OutsideLine, sell, buy []string) {
	if l.Sell.present() && len(sell) == 0 && len(buy) > 0 {
		w.add("gate_edge_wrong_direction", path(lb, "sell")...)
	}
	if l.Buy.present() && len(buy) == 0 && len(sell) > 0 {
		w.add("gate_edge_wrong_direction", path(lb, "buy")...)
	}
}

// side raises the warnings of one present side and records its quotes.
func (w *outsideWarner) side(lb []string, n *OutsideNode, l *OutsideLine, name string, s *OutsideSide, gates []string) {
	if !s.present() {
		return
	}
	if windowAllClosed(s.Window, w.c.Calendar.Seasons) {
		w.add("window_all_closed", path(lb, name, "window")...)
	}
	sellSide := name == "sell"
	for _, g := range gates {
		scale := n.gateScale(g)
		q, ok := outside.Quote(int64(l.Base), s.MarginBP, scale, MaxPrice, sellSide)
		if !ok {
			continue
		}
		w.quote(lb, name, g, l.Commodity, q, sellSide)
	}
}

// quote records a quote for the arbitrage test and raises
// quote_outside_reference_band when it lies outside the band of reference
// prices of its gate (spec-external-sector.md 6.6.2).
func (w *outsideWarner) quote(lb []string, name, gate, commodity string, q int64, sellSide bool) {
	if w.asks == nil {
		w.asks, w.bids = map[string][]gateQuote{}, map[string][]gateQuote{}
	}
	if sellSide {
		w.asks[commodity] = append(w.asks[commodity], gateQuote{gate, q})
	} else {
		w.bids[commodity] = append(w.bids[commodity], gateQuote{gate, q})
	}
	cm := w.lk.commodity[commodity]
	if cm == nil || w.lk.region[gate] == nil {
		return
	}
	ref := int64(w.lk.price(gate, cm))
	lo, okLo := checked.MulDiv(ref, w.c.World.RefBandLow, wholeBP)
	hi, okHi := checked.MulDiv(ref, w.c.World.RefBandHigh, wholeBP)
	if okLo && okHi && (q < lo || q > hi) {
		w.add("quote_outside_reference_band", path(lb, name, "gate", gate)...)
	}
}

// walk raises walk_faster_than_beliefs: the step plus the trend per tick above
// one belief step (6.4.3). It needs a valid calendar.
func (w *outsideWarner) walk(lb []string, l *OutsideLine) {
	ticks := w.c.Calendar.TicksPerYear()
	if ticks < 1 {
		return
	}
	trend := l.TrendBP
	if trend < 0 {
		trend = -trend
	}
	if l.WalkStepBP+(trend+ticks-1)/ticks > w.c.World.BeliefStep {
		w.add("walk_faster_than_beliefs", path(lb, "walk_step_bp")...)
	}
}

// global raises the warnings that read the whole module: the faucet and the drain
// without their counterpart, an arbitrage between two gates, and an import that
// is the only source of a commodity something consumes.
func (w *outsideWarner) global() {
	base := []string{"modules", "external"}
	if w.anyBuy && !w.anySell {
		w.add("outside_faucet_without_sink", base...)
	}
	if w.anySell && !w.anyBuy {
		w.add("outside_drain_without_source", base...)
	}
	made, consumed := w.recipeUse()
	for _, cm := range w.c.Commodities {
		if w.arbitrage(cm.ID) {
			w.add("outside_arbitrage", "commodities", cm.ID)
		}
		if consumed[cm.ID] && !made[cm.ID] && importSourced(w.c, cm.ID) {
			w.add("import_only_source", "commodities", cm.ID)
		}
	}
}

// arbitrage reports that the lowest ask of a commodity is below the highest bid
// at another gate (spec-external-sector.md 6.6.1, 14.2).
func (w *outsideWarner) arbitrage(commodity string) bool {
	for _, a := range w.asks[commodity] {
		for _, b := range w.bids[commodity] {
			if a.gate != b.gate && a.price < b.price {
				return true
			}
		}
	}
	return false
}

// recipeUse returns the commodities some recipe makes and keeps, and those that a
// recipe, an upkeep or a tool consumes.
func (w *outsideWarner) recipeUse() (made, consumed map[string]bool) {
	c := w.c
	made, consumed = map[string]bool{}, map[string]bool{}
	for i := range c.Recipes {
		rc := &c.Recipes[i]
		for _, o := range rc.Outputs {
			made[o.Commodity] = made[o.Commodity] || !o.Discard
		}
		for _, it := range rc.Inputs {
			consumed[it.Commodity] = true
		}
		for _, t := range rc.Requires {
			consumed[t.Commodity] = true
		}
	}
	for _, id := range upkeepNamed(c) {
		consumed[id] = true
	}
	return made, consumed
}

// upkeepNamed returns the commodities named by an upkeep list of the world,
// a role or a region.
func upkeepNamed(c *Config) []string {
	var out []string
	for _, it := range c.World.Upkeep {
		out = append(out, it.Commodity)
	}
	for i := range c.Roles {
		for _, it := range c.Roles[i].Upkeep {
			out = append(out, it.Commodity)
		}
	}
	for i := range c.Regions {
		for _, it := range c.Regions[i].ExtraUpkeep {
			out = append(out, it.Commodity)
		}
	}
	return out
}

// seasonBP returns the factor of a season table for the season id, or def when the
// season has no row or the tick has no season (id is empty).
func seasonBP(table []SeasonFactor, id string, def int64) int64 {
	if i := slices.IndexFunc(table, func(f SeasonFactor) bool { return f.Season == id }); i >= 0 && id != "" {
		return table[i].BP
	}
	return def
}

// seasonID returns the ID of the season of the tick's moment, empty when the
// calendar has none.
func (w *World) seasonID() string {
	if s := w.tk.moment.Season; s >= 0 && s < len(w.cfg.Calendar.Seasons) {
		return w.cfg.Calendar.Seasons[s].ID
	}
	return ""
}

// p18Walk is P1.8 (spec-external-sector.md 5.1.1, 6.4): for each line with a walk,
// ascending by (node, commodity), one word of the keyed stream of (tick, node,
// commodity) moves the drift. The count of words in a tick is the number of
// walking lines, whatever the state, and none is drawn at construction.
func (w *World) p18Walk() {
	x := w.external
	for i := range x.lines {
		l := &x.lines[i]
		step := l.cfg.WalkStepBP
		if step < 1 {
			continue
		}
		w.rnd.keyedPair(siteOutsideWalk, w.Tick(), int64(l.node), int64(l.commodity))
		d := w.rnd.drawRange(-step, step)
		l.drift = outside.Walk(l.drift, l.cfg.WalkRevertBP, d, l.cfg.WalkBoundBP)
		if ctx := context.Background(); w.log.Enabled(ctx, slog.LevelDebug) {
			w.log.LogAttrs(ctx, slog.LevelDebug, "outside drift", slog.Int64(logkeys.Tick, w.Tick()),
				slog.Int64(logkeys.Node, int64(l.node)), slog.Int64(logkeys.Commodity, int64(l.commodity)),
				slog.Int64(logkeys.Drift, l.drift))
		}
	}
}

// p35Orders is P3.5 (spec-external-sector.md 5.2): the counterparties place their
// orders, in ascending agent ID, each for the lines of its node in commodity order
// whose side is present. Every input is the tick, the state of the line and the
// status of an edge; nothing of the realm is read (FR-EXT-023).
func (w *World) p35Orders() error {
	x := w.external
	x.setMids(w)
	for i := range x.cps {
		cp := &x.cps[i]
		for j := cp.first; j < cp.first+cp.n; j++ {
			if err := w.lineOrder(cp, &x.rows[j]); err != nil {
				return err
			}
		}
	}
	return nil
}

// setMids computes the mid of every line for the tick, once (6.1).
func (x *external) setMids(w *World) {
	for i := range x.lines {
		l := &x.lines[i]
		l.mid, l.clamped = x.midOf(w, l)
	}
}

// midOf is the mid of a line at the world's tick and whether a clamp acted: a
// function of the tick, the season, the drift and the shock, with no state of its
// own, so a snapshot can read it too.
func (x *external) midOf(w *World, l *outsideLine) (mid int64, clamped bool) {
	mid, clamped, ok := outside.Mid(outside.MidIn{
		Base: int64(l.cfg.Base), Season: seasonBP(l.cfg.PriceBySeason, w.seasonID(), wholeBP), TrendBP: l.cfg.TrendBP,
		FloorBP: l.cfg.FloorBP, CeilBP: l.cfg.CeilingBP, MaxPrice: MaxPrice,
		Tick: w.Tick(), TicksPerYear: x.ticksPerYear, Drift: l.drift, Shock: l.shock,
	})
	if !ok {
		invariant("the mid of an outside line is out of its domain, which validation rules out")
	}
	return mid, clamped
}

// capOf returns what is left of the cumulative caps of a row: the list value is
// always capped, by TotalValue or by MaxLineTotal (7.3).
func (r *sideRow) capOf(s *OutsideSide) outside.Cap {
	value := int64(MaxLineTotal)
	if s.TotalValue > 0 {
		value = min(value, s.TotalValue)
	}
	c := outside.Cap{ValueCap: true, ValueLeft: value - r.listValue}
	if s.TotalUnits > 0 {
		c.UnitCap, c.UnitsLeft = true, s.TotalUnits-r.units
	}
	return c
}

// lineOrder is the order of one side row at P3.5 (5.2.1): the wanted units and the
// carry, whether the edge is open, the quote, the cut of the caps, the top-up from
// the boundary and the ordinary order. A side that ended places nothing.
func (w *World) lineOrder(cp *counterparty, row *sideRow) error {
	x := w.external
	l := &x.lines[row.line]
	row.t = rowTick{}
	if row.ended {
		return nil
	}
	side := &l.cfg.Sell
	if cp.buy {
		side = &l.cfg.Buy
	}
	avail := seasonBP(side.Window, w.seasonID(), wholeBP)
	wanted, carry := outside.Want(side.Capacity, avail, row.carry)
	row.carry, row.t.wanted, row.t.mid, row.t.clamped = carry, wanted, l.mid, l.clamped
	if wanted < 1 {
		return nil
	}
	from, to := l.node, cp.gate
	if cp.buy {
		from, to = cp.gate, l.node
	}
	if e, ok := w.geo.edge(from, to); !ok {
		invariant("a counterparty has no edge to its node")
	} else if e.Closed {
		row.t.cutClosed = wanted
		return nil
	}
	limit, ok := outside.Quote(l.mid, side.MarginBP, cp.scaleBP, MaxPrice, !cp.buy)
	if !ok {
		invariant("the quote of an outside side is out of its domain, which validation rules out")
	}
	if row.t.limit = limit; limit < 1 {
		row.t.cutPrice = wanted
		return nil
	}
	row.t.offered, row.t.cutCap = outside.Offer(row.capOf(side), wanted, limit)
	if row.t.offered < 1 {
		return nil
	}
	return w.placeOutside(cp, l, row)
}

// placeOutside tops up the account of a counterparty for one row and places its
// order through the ordinary path (spec-external-sector.md 5.2.1 steps 5 and 6, D530):
// a buyer receives the budget limit * offered in money, a seller the offered units,
// born in this tick, and the order is admitted like any other.
func (w *World) placeOutside(cp *counterparty, l *outsideLine, row *sideRow) error {
	limit, offered := row.t.limit, row.t.offered
	o := Order{Agent: cp.agent, Commodity: l.commodity, Side: Ask, Price: Price(limit), Quantity: Quantity(offered)}
	if cp.buy {
		o.Side = Bid
		budget, ok := checked.Mul(limit, offered)
		src, open := w.ledger.Boundary(ledger.OutsideBudget, ledger.Money)
		if !ok || !open {
			invariant("the budget of an outside bid overflows or has no boundary account")
		}
		if err := w.ledger.Transfer(src, cp.account, budget, ledger.OutsideBudget); err != nil {
			return fmt.Errorf("budget of counterparty %d at tick %d: %w", cp.agent, w.Tick(), err)
		}
	} else {
		src, open := w.ledger.Boundary(ledger.OutsideOffer, int(l.commodity))
		if !open {
			invariant("an outside sell side has no boundary account for its commodity")
		}
		if err := w.ledger.Move(src, cp.account, int(l.commodity), offered, ledger.OutsideOffer); err != nil {
			return fmt.Errorf("offer of counterparty %d at tick %d: %w", cp.agent, w.Tick(), err)
		}
	}
	mi := w.markets.index(cp.gate, l.commodity)
	b := w.markets.books[mi]
	f := orderFacts{Account: cp.account, AgentKnown: true, SideAllowed: true, CommodityKnown: true, Tradable: true, Tally: &w.receipts}
	if reason, ok := b.admit(w.ledger, f, o); !ok {
		b.emitRejected(&w.obs, w.log, o, reason)
		row.t.offered = 0 // nothing was placed
		return nil
	}
	b.emitPlaced(&w.obs, o)
	w.markets.record(mi, intentRec{})
	return nil
}

// record adds the money of the trades of a market just cleared to the rows of the
// counterparties in them, so the sweep can say what each row paid or received: the
// account of a counterparty holds the money of all its commodities together.
func (x *external) record(w *World, trades []Trade) {
	regions := len(w.cfg.Regions)
	for _, t := range trades {
		for _, a := range [2]AgentID{t.Buyer, t.Seller} {
			if int(a) < regions || int(a) >= w.firstHousehold {
				continue
			}
			cp := &x.cps[int(a)-regions]
			for j := cp.first; j < cp.first+cp.n; j++ {
				if r := &x.rows[j]; x.lines[r.line].commodity == t.Commodity {
					v, ok := checked.Mul(int64(t.Price), int64(t.Quantity))
					if sum, ok2 := checked.Add(r.t.paid, v); ok && ok2 {
						r.t.paid = sum
					} else {
						invariant("a counterparty's paid sum overflowed")
					}
				}
			}
		}
	}
}

// p42Sweep is P4.2 (spec-external-sector.md 5.3): when every market has cleared and
// every reservation is released, each counterparty in agent ID order is emptied to
// its boundary, money then goods in commodity order; then the cumulative units, the
// list value and the ended flags are updated and the events are emitted.
func (w *World) p42Sweep() error {
	x := w.external
	for i := range x.cps {
		if err := x.sweep(w, &x.cps[i]); err != nil {
			return err
		}
	}
	if w.obs.supplied {
		x.emitTraded(w)
	}
	x.emitEnded(w)
	return nil
}

// sweep empties one counterparty and settles its rows. A buyer returns its unspent
// money and hands over what it bought; a seller hands over what it was paid and
// returns what was not bought.
func (x *external) sweep(w *World, cp *counterparty) error {
	moneyReason, goodsReason := ledger.OutsideReceipts, ledger.OutsideUnsold
	if cp.buy {
		moneyReason, goodsReason = ledger.OutsideUnspent, ledger.OutsidePurchases
	}
	v, err := w.ledger.Account(cp.account)
	if err != nil {
		invariant("a counterparty account is unreadable: " + err.Error())
	}
	if v.Balance > 0 {
		dst, ok := w.ledger.Boundary(moneyReason, ledger.Money)
		if !ok {
			invariant("an outside money reason has no boundary account")
		}
		if err := w.ledger.Transfer(cp.account, dst, v.Balance, moneyReason); err != nil {
			return fmt.Errorf("sweep of the money of counterparty %d at tick %d: %w", cp.agent, w.Tick(), err)
		}
	}
	for j := cp.first; j < cp.first+cp.n; j++ {
		if err := x.sweepRow(w, cp, &x.rows[j], goodsReason); err != nil {
			return err
		}
	}
	return nil
}

// sweepRow moves the goods of one row to the boundary and updates its state.
func (x *external) sweepRow(w *World, cp *counterparty, row *sideRow, reason ledger.Reason) error {
	t := &row.t
	l := &x.lines[row.line]
	k := int(l.commodity)
	held, _, err := w.ledger.Holding(cp.account, k)
	if err != nil {
		invariant("a counterparty account is unreadable: " + err.Error())
	}
	if held > 0 {
		dst, ok := w.ledger.Boundary(reason, k)
		if !ok {
			invariant("an outside goods reason has no boundary account")
		}
		if err := w.ledger.Move(cp.account, dst, k, held, reason); err != nil {
			return fmt.Errorf("sweep of the goods of counterparty %d at tick %d: %w", cp.agent, w.Tick(), err)
		}
	}
	t.traded, t.value = held, t.paid
	if !cp.buy {
		t.traded = t.offered - held
	}
	return x.settleRow(w, cp, row, l)
}

// settleRow adds the units traded in the tick to the cumulative state of a row and
// ends it when its cap is reached (7.3): the list value is the units times the limit,
// not the money.
func (x *external) settleRow(w *World, cp *counterparty, row *sideRow, l *outsideLine) error {
	t := &row.t
	if t.limit < 1 {
		return nil // no quote was made: nothing was offered and nothing can end
	}
	side := &l.cfg.Sell
	if cp.buy {
		side = &l.cfg.Buy
	}
	next, listValue, ok := row.capOf(side).Settle(t.traded, t.limit)
	if !ok {
		invariant("a side traded more than its caps allowed")
	}
	row.units += t.traded
	row.listValue += listValue
	t.listValue = listValue
	if cause, ended := next.Ended(t.limit); ended && !row.ended {
		row.ended = true
		t.endCause = cause
	}
	return nil
}

// sideOf is the Side of a counterparty: Bid for a buyer and Ask for a seller.
func (cp *counterparty) sideOf() Side {
	if cp.buy {
		return Bid
	}
	return Ask
}

// gatePair returns the buyer and the seller of the gate that starts at
// counterparty i (either may be missing, nil), and the index after them: a gate has
// at most two counterparties, the buyer first (4.2.2).
func (x *external) gatePair(i int) (buyer, seller *counterparty, next int) {
	cp := &x.cps[i]
	next = i + 1
	if cp.buy {
		buyer = cp
		if next < len(x.cps) && x.cps[next].node == cp.node && x.cps[next].gate == cp.gate {
			seller, next = &x.cps[next], next+1
		}
		return buyer, seller, next
	}
	return nil, cp, next
}

// eachRowOrdered calls f for every row of the counterparties in the order of the
// events: by node, gate, commodity, buy before sell. A gate's two counterparties hold
// their rows in commodity order, so the two lists are merged.
func (x *external) eachRowOrdered(f func(cp *counterparty, row *sideRow)) {
	for i := 0; i < len(x.cps); {
		b, s, next := x.gatePair(i)
		i = next
		var bj, sj, bn, sn int
		if b != nil {
			bj, bn = b.first, b.first+b.n
		}
		if s != nil {
			sj, sn = s.first, s.first+s.n
		}
		for bj < bn || sj < sn {
			if sj >= sn || (bj < bn && x.rows[bj].line <= x.rows[sj].line) {
				f(b, &x.rows[bj])
				bj++
				continue
			}
			f(s, &x.rows[sj])
			sj++
		}
	}
}

// emitTraded delivers OutsideTraded for every row with a unit wanted or cut
// (spec-external-sector.md 15.1). The caller checks that an observer was supplied.
func (x *external) emitTraded(w *World) {
	x.eachRowOrdered(func(cp *counterparty, row *sideRow) {
		t := &row.t
		if t.wanted < 1 && t.cutClosed+t.cutCap+t.cutPrice < 1 {
			return
		}
		w.obs.deliver(OutsideTraded{
			Tick: w.Tick(), Node: cp.node, Region: cp.gate, Commodity: x.lines[row.line].commodity, Side: cp.sideOf(),
			Mid: Price(t.mid), Price: Price(t.limit), Wanted: t.wanted, Offered: t.offered, Traded: t.traded,
			Value: Money(t.value), ListValue: Money(t.listValue),
			CutClosed: t.cutClosed, CutCap: t.cutCap, CutPrice: t.cutPrice, PriceClamped: t.clamped,
		})
	})
}

// emitEnded delivers OutsideEnded and logs the Warn "outside side ended" for every
// row that ended in this tick, and the Debug line of every side that was active.
func (x *external) emitEnded(w *World) {
	debug := w.log.Enabled(context.Background(), slog.LevelDebug)
	x.eachRowOrdered(func(cp *counterparty, row *sideRow) {
		t := &row.t
		l := &x.lines[row.line]
		if debug && (t.wanted > 0 || t.cutClosed+t.cutCap+t.cutPrice > 0) {
			w.log.LogAttrs(context.Background(), slog.LevelDebug, "outside side", slog.Int64(logkeys.Tick, w.Tick()),
				slog.Int64(logkeys.Node, int64(cp.node)), slog.Int64(logkeys.Region, int64(cp.gate)),
				slog.Int64(logkeys.Commodity, int64(l.commodity)), slog.String(logkeys.Side, cp.sideOf().String()),
				slog.Int64(logkeys.Price, t.limit), slog.Int64(logkeys.Quantity, t.traded), slog.Int64(logkeys.Value, t.value))
		}
		if t.endCause == 0 {
			return
		}
		cause := CauseUnits
		if t.endCause == outside.EndValue {
			cause = CauseValue
		}
		if w.obs.supplied {
			w.obs.deliver(OutsideEnded{
				Tick: w.Tick(), Node: cp.node, Region: cp.gate, Commodity: l.commodity, Side: cp.sideOf(),
				Cause: cause, Units: row.units, ListValue: Money(row.listValue),
			})
		}
		w.log.LogAttrs(context.Background(), slog.LevelWarn, "outside side ended", slog.Int64(logkeys.Tick, w.Tick()),
			slog.Int64(logkeys.Node, int64(cp.node)), slog.Int64(logkeys.Region, int64(cp.gate)),
			slog.Int64(logkeys.Commodity, int64(l.commodity)), slog.String(logkeys.Side, cp.sideOf().String()),
			slog.String(logkeys.Cause, string(cause)))
	})
}
