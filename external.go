package economysim

import (
	"cmp"
	"slices"

	"github.com/Whealinghold/economy-sim/internal/checked"
	"github.com/Whealinghold/economy-sim/internal/outside"
)

// This file is the external-sector module (spec-external-sector.md;
// architecture.md 2.10.1): the canonical form of its configuration, its
// validator and its warnings. The arithmetic of the price and the quantities is
// in internal/outside. Outside this file the module appears only in the
// catalogue files (config.go, hash.go, bounds.go, defaults.go, errors.go,
// validate.go) and at the named hooks of ADR 0010.

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
	nodes := outsideNodeSet(&v.c)
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
		scale := int64(DefaultGateScaleBP)
		if i := slices.IndexFunc(n.Gates, func(o OutsideGate) bool { return o.Region == g }); i >= 0 {
			scale = n.Gates[i].ScaleBP
		}
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
