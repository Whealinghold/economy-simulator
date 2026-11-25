package economysim

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"slices"

	"github.com/Whealinghold/economy-sim/internal/fiscal"
	"github.com/Whealinghold/economy-sim/internal/ledger"
	"github.com/Whealinghold/economy-sim/internal/logkeys"
)

// This file is the fiscal module (spec-fiscal.md, FIS; architecture.md 2.10.1): the
// treasury's taxes, duties, tolls, payouts and standing orders. The configuration
// types are in config.go, the validator in fiscal_validate.go and its class rules in
// fiscal_class.go, the warnings in fiscal_warnings.go and the hash in fiscal_hash.go.
// The module is off unless Config.Modules.Fiscal is present, and while it is off it
// leaves no trace (architecture.md 2.10.5).

// fiscalReasons returns the ledger's reason of every flow that moves money or goods
// between accounts, in the canonical order: the taxes, the duties, the tolls and the
// payouts, each list by ID (spec-fiscal.md 3.4.1, plan 3-fis D5). A purchase and a
// release use the reason trade and declare none. The reasons are Internal and open no
// boundary account, so no account ID shifts; fiscal flow j is the ledger's reason
// FlowReason(len(money flows) + j). A payout in kind moves goods and no money. It
// returns nil for a configuration with the module off.
func fiscalReasons(c *Config) []ledger.FiscalFlow {
	m := c.Modules.Fiscal
	if m == nil {
		return nil
	}
	money := len(m.Taxes) + len(m.Duties) + len(m.Tolls)
	out := make([]ledger.FiscalFlow, money+len(m.Payouts))
	for i, p := range m.Payouts {
		out[money+i].Goods = p.Commodity != ""
	}
	return out
}

// fiscalKind is the kind of a flow in the module's tables. The kinds are in the
// canonical order of the six lists (spec-fiscal.md 17): a flow's position is its
// index in the concatenation taxes, duties, tolls, payouts, purchases, releases,
// each by ID, and the sparse scale list of the checkpoint names a flow by it.
type fiscalKind uint8

const (
	fkTax fiscalKind = iota
	fkDuty
	fkToll
	fkPayout
	fkPurchase
	fkRelease
)

// carryPair is the carry of one account on a flow: the account ID and the
// remainder of the rate, in the units of the flow's denominator (spec-fiscal.md
// 5.2). A flow keeps its pairs in ascending account ID, for every open account it
// has acted on, zeros included (17).
type carryPair struct {
	account int
	carry   int64
}

// fiscalWindow is the Window of a flow with its seasons resolved to indexes of the
// calendar's seasons (spec-fiscal.md 3.1.2).
type fiscalWindow struct {
	start, until int64
	seasons      []int
}

// fiscalTarget is the class of a tax or a payout, resolved once (spec-fiscal.md 3.2).
// in says which regions a class of households or a pool is read in (the region it
// names, or every region the authority covers); from lists, for a class of
// treasuries, the indexes into polity.tr of the treasuries that pay or receive, in
// ascending JurisdictionID.
type fiscalTarget struct {
	kind string
	role int
	in   []bool
	from []int
}

// fiscalFlowState is the state of one flow and what the step derives from its
// configuration once. scale is saved, the carries are saved (17), the rest is
// derived.
type fiscalFlowState struct {
	id        string
	kind      fiscalKind
	at        int // index in its list
	authority JurisdictionID
	treasury  int // index into polity.tr of the authority's treasury
	reason    ledger.Reason
	resource  int // a payout's resource: -1 is money, else the commodity index
	period    int64
	shape     fiscal.Shape
	window    fiscalWindow
	target    fiscalTarget
	// scale is the flow's runtime scale in basis points, 10 000 by default.
	scale int64
	// carry holds the pairs of a flow that keeps a carry per account; single is the
	// carry of a purchase or a release.
	carry  []carryPair
	single int64
	// duty is a duty's payer, scope, commodity, rate and the trades of the tick; toll
	// a toll's side, mode and amount.
	duty fiscalDuty
	toll fiscalToll
	// order is a purchase's or a release's commodity, quantity, limit and bound.
	order fiscalOrder
}

// fiscalAgg is the aggregate of one flow in one region in the tick: the counters of
// FiscalFlowApplied (spec-fiscal.md 16.1). It is scratch, cleared when its flow runs.
type fiscalAgg struct {
	due, applied                int64
	parties, short, cut, headrm int64
	exempt                      int64
}

// fiscalPayer is one account a flow acts on in the tick: the account, its weight
// (1 for a pool or a treasury), its region and the agent that owns it, which keys
// a cut (spec-fiscal.md 8.4.2).
type fiscalPayer struct {
	account int
	weight  int64
	region  RegionID
	owner   AgentID
}

// fiscalState is the state of the module, held by World.fiscal and nil when it is
// off. The scratch is reused every tick and never saved.
type fiscalState struct {
	pol     *polity
	flows   []fiscalFlowState // canonical position order
	tolls   [2]int            // the positions of the tolls: from, to
	orders  [2]int            // the positions of the purchases then the releases: from, to
	regions int
	agg     []fiscalAgg // flow position * regions + region
	ran     []int       // the positions that ran at P1.7b, in event order
	// receipts are the units of goods the payouts of the tick gave to households,
	// valued and added to the welfare scratch at P6.3 (spec-fiscal.md 12.3); scratch.
	receipts []fiscalReceipt
	// scratch
	paid    []tollCharge // the tolls of the shipment being checked or dispatched
	view    []tollCharge // the tolls of a route row
	payers  []fiscalPayer
	bases   []fiscal.Base // a duty's base of each payer
	dues    []int64
	take    []int64
	applied []int64
	next    []carryPair
	rec     []fiscal.Recipient
	order   []int      // the active payouts, grouped by authority and resource
	ent     []payEntry // the recipients of one group of payouts
	cut     fiscal.Scratch
}

// newFiscal builds the state of a canonical configuration the validator accepted,
// nil when the module is off. Every name a flow uses resolves after validation.
func newFiscal(c *Config, p *polity) *fiscalState {
	m := c.Modules.Fiscal
	if m == nil {
		return nil
	}
	s := &fiscalState{pol: p, regions: len(c.Regions)}
	money := 0
	if ms := c.Modules.MoneySupply; ms != nil {
		money = len(ms.Flows)
	}
	for i, t := range m.Taxes {
		f := s.add(c, fkTax, i, t.ID, t.Authority, t.Window)
		f.period, f.shape = flowPeriod(c, t.Per), fiscal.Fixed
		if t.RateBP != 0 {
			f.shape = fiscal.Rate
		}
		f.target = s.resolve(c, f.authority, t.Target)
	}
	for i := range m.Duties {
		d := &m.Duties[i]
		f := s.add(c, fkDuty, i, d.ID, d.Authority, d.Window)
		f.duty = newDuty(c, d)
		// a duty is paid by households, in the regions of the trades it takes.
		f.target = fiscalTarget{kind: TargetHouseholds, role: -1, in: s.regionsOf(c, f.authority, d.Region)}
	}
	s.tolls[0] = len(s.flows)
	for i := range m.Tolls {
		t := &m.Tolls[i]
		f := s.add(c, fkToll, i, t.ID, t.Authority, t.Window)
		f.toll = newToll(c, t)
		// a toll is paid by the shipper, a household; the regions its authority covers
		// are where a crossing is told.
		f.target = fiscalTarget{kind: TargetHouseholds, role: -1, in: s.regionsOf(c, f.authority, "")}
	}
	s.tolls[1] = len(s.flows)
	for i, o := range m.Payouts {
		f := s.add(c, fkPayout, i, o.ID, o.Authority, o.Window)
		f.period = flowPeriod(c, o.Per)
		f.target = s.resolve(c, f.authority, o.Target)
		f.resource = payoutResource(c, o)
	}
	s.orders[0] = len(s.flows)
	for i, o := range m.Purchases {
		f := s.add(c, fkPurchase, i, o.ID, o.Authority, o.Window)
		f.order = newOrder(c, o.Commodity, o.Quantity, o.LimitBP, o.Hold)
	}
	for i, o := range m.Releases {
		f := s.add(c, fkRelease, i, o.ID, o.Authority, o.Window)
		f.order = newOrder(c, o.Commodity, o.Quantity, o.LimitBP, o.Reserve)
	}
	s.orders[1] = len(s.flows)
	s.numberReasons(money)
	s.agg = make([]fiscalAgg, len(s.flows)*s.regions)
	return s
}

// payoutResource is the index of the commodity a payout gives, or -1 for money. A name
// that is not defined is a violated invariant: the validator rejects it, and -1 would
// pay money in its place.
func payoutResource(c *Config, o Payout) int {
	if o.Commodity == "" {
		return -1
	}
	i := slices.IndexFunc(c.Commodities, func(x Commodity) bool { return x.ID == o.Commodity })
	if i < 0 {
		invariant("a payout names a commodity that is not defined: " + o.ID)
	}
	return i
}

// add appends the flow of a kind and index to the table and returns it, resolved
// as far as the fields every kind shares: the authority and its treasury, the
// window with its seasons, the scale and the period of a tick.
func (s *fiscalState) add(c *Config, k fiscalKind, i int, id, authority string, w Window) *fiscalFlowState {
	j, ok := s.pol.v.Lookup(authority)
	if !ok || s.pol.byJ[j] < 0 {
		invariant("a fiscal flow's authority is not a governed jurisdiction after validation: " + id)
	}
	win := fiscalWindow{start: w.Start, until: w.Until}
	for _, name := range w.Seasons {
		at := slices.IndexFunc(c.Calendar.Seasons, func(x Season) bool { return x.ID == name })
		if at < 0 {
			invariant("a fiscal flow's season is not in the calendar after validation: " + id)
		}
		win.seasons = append(win.seasons, at)
	}
	s.flows = append(s.flows, fiscalFlowState{
		id: id, kind: k, at: i, authority: j, treasury: int(s.pol.byJ[j]), window: win, period: 1, scale: wholeBP, resource: -1,
	})
	return &s.flows[len(s.flows)-1]
}

// flowPeriod is the ticks of the period Per names (spec-fiscal.md 5.2).
func flowPeriod(c *Config, per string) int64 {
	if per != PerYear {
		return 1
	}
	n := c.Calendar.TicksPerYear()
	if n < 1 {
		invariant("a fiscal flow's year has no ticks after validation")
	}
	return n
}

// numberReasons gives the taxes, duties, tolls and payouts their ledger reasons in
// that order after the money flows (plan 3-fis D5); a purchase and a release have
// none, they trade under the reason trade (spec-fiscal.md 3.4.2).
func (s *fiscalState) numberReasons(money int) {
	n := 0
	for i := range s.flows {
		if f := &s.flows[i]; f.kind <= fkPayout {
			f.reason = ledger.FlowReason(money + n)
			n++
		}
	}
}

// active reports whether the flow runs in the tick: inside its window, in a season
// of it and not paused at scale 0 (FR-FIS-006, FR-FIS-044). season is the index of
// the season of the tick, from the moment of P1.1.
func (f *fiscalFlowState) active(tick int64, season int) bool {
	w := &f.window
	if f.scale == 0 || tick < w.start || (w.until != 0 && tick >= w.until) {
		return false
	}
	return len(w.seasons) == 0 || slices.Contains(w.seasons, season)
}

// The flow scale has two owners, the money supply module and this one, in one
// namespace of flow IDs (spec-fiscal.md 3.1.1, 13.1). The calls below are the
// lookups the integration slot uses: the key of an ID in the one flow-ID order, the
// setter, and the non-default scales.

// flowKey is the key of a flow ID in the one flow-ID order of both owners: the
// number of flow IDs below it. It reports false for an ID that is neither a money
// flow nor a fiscal flow (ErrUnknownFlow).
func (w *World) flowKey(id string) (int64, bool) {
	var below int64
	found := false
	count := func(other string) {
		found = found || other == id
		if other < id {
			below++
		}
	}
	for _, f := range w.moneyFlowList() {
		count(f.ID)
	}
	if w.fiscal != nil {
		for i := range w.fiscal.flows {
			count(w.fiscal.flows[i].id)
		}
	}
	return below, found
}

// setScaleOf sets the scale of the flow with the ID, a money flow or a fiscal flow,
// from the next position that reads it. The ID was found by flowKey.
func (w *World) setScaleOf(id string, bp int64, cause Cause) error {
	if at := slices.IndexFunc(w.moneyFlowList(), func(f MoneyFlow) bool { return f.ID == id }); at >= 0 {
		return w.setFlowScale(at, bp, cause)
	}
	at := -1
	if w.fiscal != nil {
		at = slices.IndexFunc(w.fiscal.flows, func(f fiscalFlowState) bool { return f.id == id })
	}
	if at < 0 {
		invariant("a flow ID found by flowKey is neither a money flow nor a fiscal flow: " + id)
	}
	return w.setFiscalScale(at, bp, cause)
}

// setFiscalScale sets the scale of the fiscal flow at position i: ErrOutOfBounds
// outside 0 to MaxFlowScale, changing nothing, and no event when the scale is
// already bp. It emits MoneyFlowScaled with the cause (FR-FIS-043).
func (w *World) setFiscalScale(i int, bp int64, cause Cause) error {
	if bp < 0 || bp > MaxFlowScale {
		return fmt.Errorf("flow scale %d outside 0 to %d: %w", bp, int64(MaxFlowScale), ErrOutOfBounds)
	}
	f := &w.fiscal.flows[i]
	from := f.scale
	if from == bp {
		return nil
	}
	f.scale = bp
	if w.obs.supplied {
		w.obs.deliver(MoneyFlowScaled{Tick: w.Tick(), Flow: f.id, From: from, To: bp, Cause: cause})
	}
	if ctx := context.Background(); w.log.Enabled(ctx, slog.LevelDebug) {
		w.log.LogAttrs(ctx, slog.LevelDebug, "money flow scaled", slog.String(logkeys.Flow, f.id),
			slog.Int64(logkeys.From, from), slog.Int64(logkeys.To, bp))
	}
	return nil
}

// fiscalSlots adds the fiscal flows whose scale is not 10 000 to the slot list and
// puts the list, with the money flows', in one flow-ID order (spec-fiscal.md 13.1).
func (w *World) fiscalSlots(s *SlotState) {
	if w.fiscal == nil {
		return
	}
	for i := range w.fiscal.flows {
		if f := &w.fiscal.flows[i]; f.scale != wholeBP {
			s.FlowScales = append(s.FlowScales, FlowScale{Flow: f.id, BP: f.scale})
		}
	}
	slices.SortFunc(s.FlowScales, func(a, b FlowScale) int { return cmp.Compare(a.Flow, b.Flow) })
}
