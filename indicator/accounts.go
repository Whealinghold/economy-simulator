package indicator

import (
	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/internal/measure"
)

// The reasons of the boundary flows that the transport, outside and risk modules
// add. The root package declares them with those modules (plans 2-12, 2-16 and
// the risk and storage plans); until then they are named here by their string,
// the rule constant of the ledger (spec-indicators.md 7.1).
const (
	reasonHaulUse         economysim.Reason = "haul_use"
	reasonTransitUse      economysim.Reason = "transit_use"
	reasonLoss            economysim.Reason = "loss"
	reasonOutsideOffer    economysim.Reason = "outside_offer"
	reasonOutsideUnsold   economysim.Reason = "outside_unsold"
	reasonOutsidePurchase economysim.Reason = "outside_purchases"
	reasonOutsideBudget   economysim.Reason = "outside_budget"
	reasonOutsideUnspent  economysim.Reason = "outside_unspent"
	reasonOutsideReceipts economysim.Reason = "outside_receipts"
)

// DepreciationBasis says how a net product counts the wear of durables.
type DepreciationBasis int

// BasisRetirement is the version-1 basis: the items retired because their
// durability ran out, valued at the table price (spec-indicators.md 7.6.2).
const BasisRetirement DepreciationBasis = iota + 1

// String returns the name of the basis.
func (b DepreciationBasis) String() string {
	if b == BasisRetirement {
		return "retirement"
	}
	return "unknown"
}

// GDPRow is the production and expenditure views of one region, or of the realm
// (Region -1), in minor units at the price table of the call (spec-indicators.md
// 7.2, 7.3).
type GDPRow struct {
	// Region is the region, or -1 for the realm.
	Region economysim.RegionID
	// Output, Intermediate and Pending are the terms of the production view:
	// the value of the units made, the units used up as inputs, and the signed
	// change of pending output. Cash is Output - Intermediate and ValueAdded
	// (GDP by the production approach, accrual) Cash + Pending.
	Output, Intermediate, Pending, Cash, ValueAdded int64
	// C, StockChange, Written, Carriage, NX, Inter and Endow are the terms of the
	// expenditure view; Reloc is the value of the goods that moved with relocated
	// accounts, the reconciling item of the identity.
	C, StockChange, Written, Carriage, NX, Inter, Endow, Reloc int64
	// NetProduct is ValueAdded less the retired items at the table (Basis).
	NetProduct int64
	// Residual is ValueAdded less the expenditure view plus Reloc, which is 0 by
	// the ledger's conservation. For the realm it is ValueAdded less the
	// expenditure view with TradeValuation as the interregional term: 0 at one
	// price table, and the value of the relocated goods (Reloc, negated) at
	// regional prices.
	Residual int64
	// RelocUnits is the units of the relocation item summed over commodities and
	// RelocEvents the units of AgentRelocated into the region less out of it; they
	// are equal (spec-indicators.md 8.2).
	RelocUnits, RelocEvents int64
	// Construction is the value at the table of the goods of the construction
	// bucket, a memo line that is in no flow (7.1.4).
	Construction int64
	// Binding is the counters of the caps and limits that acted in the region.
	Binding BindingCounts
}

// GDPResult is the three views of a window and the identity between them
// (spec-indicators.md 9.1, row 1).
type GDPResult struct {
	ConfigHash [32]byte
	From, To   int64
	// Regions has one row per region and Realm the sum of them.
	Regions []GDPRow
	Realm   GDPRow
	// TradeValuation is the interregional term of the realm: the value the goods
	// that crossed a lane gain or lose by the difference of the two prices
	// (7.4.2); 0 at one price table.
	TradeValuation int64
	Basis          DepreciationBasis
	Warnings       []Warning
}

// stockTable is the stocks of a snapshot at region * commodities + commodity.
type stockTable struct{ held, pend, estate, transit []int64 }

// stocksOf reads the units a snapshot holds. The units in transit are not in the
// snapshot yet: shortcut: Snapshot.Shipments (plan 2-12) is not in the tree, so
// transit is 0 and a cargo in flight must be counted in Held by the caller's
// fixture; read it from the snapshot when the field exists, in this function.
func stocksOf(s economysim.Snapshot, regions, commodities int) (stockTable, error) {
	n := regions * commodities
	st := stockTable{make([]int64, n), make([]int64, n), make([]int64, n), make([]int64, n)}
	for _, m := range s.Markets {
		r, c := m.State.Region, m.State.Commodity
		if r < 0 || int(r) >= regions || c < 0 || int(c) >= commodities {
			return stockTable{}, wrap("GDP", "market", ErrRange)
		}
		i := int(r)*commodities + int(c)
		st.held[i] = int64(m.HeldByAgents) + int64(m.HeldByEstate)
		st.estate[i], st.pend[i] = int64(m.HeldByEstate), int64(m.Pending)
	}
	return st, nil
}

// checkSnapshots verifies that the two snapshots bound the window.
func (w Window) checkSnapshots(preset string, open, closing economysim.Snapshot) error {
	if open.ConfigHash != w.ConfigHash || closing.ConfigHash != w.ConfigHash {
		return wrap(preset, "snapshots", ErrConfigMismatch)
	}
	if open.Tick != w.From || closing.Tick != w.To {
		return wrap(preset, "snapshots", ErrWindowOrder)
	}
	return nil
}

// checkClosing verifies that one snapshot closes the window: its configuration
// (ErrConfigMismatch) and its tick (ErrWindowOrder).
func (w Window) checkClosing(preset string, closing economysim.Snapshot) error {
	if closing.ConfigHash != w.ConfigHash {
		return wrap(preset, "snapshot", ErrConfigMismatch)
	}
	if closing.Tick != w.To {
		return wrap(preset, "snapshot", ErrWindowOrder)
	}
	return nil
}

// unitTables is the corrected unit flows of a window by (region, commodity).
type unitTables struct {
	out, in, cons, want, spoil, disp, wear, haul, feed, loss []int64
	offer, unsold, purch, init                               []int64
	arrIn, arrOut                                            []int64
}

// loadUnits reads the flows and the arrivals of a window.
func loadUnits(w Window, f flowSet) (unitTables, error) {
	var u unitTables
	for _, t := range []struct {
		dst    *[]int64
		reason economysim.Reason
	}{
		{&u.out, economysim.ReasonRecipeOutput}, {&u.in, economysim.ReasonRecipeInput},
		{&u.cons, economysim.ReasonConsumption}, {&u.want, economysim.ReasonWantConsumption},
		{&u.spoil, economysim.ReasonSpoilage}, {&u.disp, economysim.ReasonDisposal}, {&u.wear, economysim.ReasonWear},
		{&u.haul, reasonHaulUse}, {&u.feed, reasonTransitUse}, {&u.loss, reasonLoss},
		{&u.offer, reasonOutsideOffer}, {&u.unsold, reasonOutsideUnsold}, {&u.purch, reasonOutsidePurchase},
		{&u.init, economysim.ReasonInitial},
	} {
		v, err := f.units(t.reason)
		if err != nil {
			return unitTables{}, err
		}
		*t.dst = v
	}
	n := w.Regions * w.Commodities
	u.arrIn, u.arrOut = make([]int64, n), make([]int64, n)
	for _, a := range w.Arrivals {
		o, d := int(a.Origin)*w.Commodities+int(a.Commodity), int(a.Destination)*w.Commodities+int(a.Commodity)
		if a.Origin < 0 || int(a.Origin) >= w.Regions || a.Destination < 0 || int(a.Destination) >= w.Regions || a.Commodity < 0 || int(a.Commodity) >= w.Commodities {
			return unitTables{}, wrap("GDP", "arrivals", ErrRange)
		}
		moved, err := net("GDP", "arrivals", []int64{a.Delivered, a.Diverted})
		if err != nil {
			return unitTables{}, err
		}
		if u.arrOut[o], err = net("GDP", "arrivals", []int64{u.arrOut[o], moved}); err != nil {
			return unitTables{}, err
		}
		if u.arrIn[d], err = net("GDP", "arrivals", []int64{u.arrIn[d], moved}); err != nil {
			return unitTables{}, err
		}
	}
	return u, nil
}

// regionAcc accumulates the terms of one region.
type regionAcc struct {
	output, interm, pending, cons, change, written, carriage, nx, arrIn, arrOut, endow, reloc, wear, construction acc
	relocUnits                                                                                                    measure.Int128
	relocOver                                                                                                     bool
}

// addCell adds the terms of one (region, commodity) cell at price pr.
func (a *regionAcc) addCell(u unitTables, st, op stockTable, i int, pr economysim.Price) {
	a.output.add(u.out[i], pr)
	a.interm.add(u.in[i], pr)
	a.cons.add(u.cons[i], pr)
	a.cons.add(u.want[i], pr)
	// The sums and differences of units are added to the wide accumulators one
	// term at a time: (x + y) * p is x * p + y * p, so nothing wraps (FR-IND-009).
	a.pending.add(st.pend[i], pr)
	a.pending.sub(op.pend[i], pr)
	for _, t := range [...]struct {
		q    int64
		plus bool
	}{{st.held[i], true}, {op.held[i], false}, {st.transit[i], true}, {op.transit[i], false}, {st.pend[i], true}, {op.pend[i], false}} {
		if t.plus {
			a.change.add(t.q, pr)
		} else {
			a.change.sub(t.q, pr)
		}
	}
	for _, q := range [...]int64{u.spoil[i], u.disp[i], u.wear[i], u.loss[i]} {
		a.written.add(q, pr)
	}
	a.wear.add(u.wear[i], pr)
	a.carriage.add(u.haul[i], pr)
	a.carriage.add(u.feed[i], pr)
	a.nx.add(u.purch[i], pr)
	a.nx.sub(u.offer[i], pr)
	a.nx.add(u.unsold[i], pr)
	a.arrIn.add(u.arrIn[i], pr)
	a.arrOut.add(u.arrOut[i], pr)
	a.endow.add(u.init[i], pr)
	// The units of the relocation item: what the ledger's identity (7.1.3, 8.2)
	// leaves after every flow and arrival.
	res, ok := measure.FromInt64(st.held[i]).Add(measure.FromInt64(op.held[i]).Neg())
	dTransit, ok2 := measure.FromInt64(st.transit[i]).Add(measure.FromInt64(op.transit[i]).Neg())
	res, ok3 := res.Add(dTransit)
	if !ok || !ok2 || !ok3 {
		a.relocOver = true
	}
	for _, t := range [...]struct {
		v    int64
		plus bool
	}{{u.out[i], false}, {u.in[i], true}, {u.cons[i], true}, {u.want[i], true}, {u.spoil[i], true},
		{u.disp[i], true}, {u.wear[i], true}, {u.haul[i], true}, {u.feed[i], true}, {u.loss[i], true}, {u.purch[i], true},
		{u.offer[i], false}, {u.unsold[i], true}, {u.init[i], false}, {u.arrIn[i], false}, {u.arrOut[i], true}} {
		v := measure.FromInt64(t.v)
		if !t.plus {
			v = v.Neg()
		}
		var ok bool
		if res, ok = res.Add(v); !ok {
			a.relocOver = true
		}
	}
	if units, ok := measure.ToInt64(res); ok {
		a.reloc.add(units, pr)
	} else {
		a.relocOver = true
	}
	if a.relocUnits, ok = a.relocUnits.Add(res); !ok {
		a.relocOver = true
	}
}

// merge adds the terms of another region.
func (a *regionAcc) merge(b regionAcc) {
	for _, p := range [...][2]*acc{{&a.output, &b.output}, {&a.interm, &b.interm}, {&a.pending, &b.pending}, {&a.cons, &b.cons},
		{&a.change, &b.change}, {&a.written, &b.written}, {&a.carriage, &b.carriage}, {&a.nx, &b.nx}, {&a.arrIn, &b.arrIn},
		{&a.arrOut, &b.arrOut}, {&a.endow, &b.endow}, {&a.reloc, &b.reloc}, {&a.wear, &b.wear}, {&a.construction, &b.construction}} {
		p[0].merge(*p[1])
	}
	var ok bool
	a.relocUnits, ok = a.relocUnits.Add(b.relocUnits)
	a.relocOver = a.relocOver || b.relocOver || !ok
}

// row converts the accumulated terms to a result row.
func (a regionAcc) row(region economysim.RegionID) (GDPRow, error) {
	const name = "GDP"
	var err error
	get := func(x acc, field string) int64 {
		v, e := x.int64(name, field)
		if err == nil {
			err = e
		}
		return v
	}
	row := GDPRow{Region: region}
	row.Output, row.Intermediate, row.Pending = get(a.output, "output"), get(a.interm, "intermediate"), get(a.pending, "pending")
	row.C, row.StockChange, row.Written = get(a.cons, "c"), get(a.change, "stock_change"), get(a.written, "written")
	row.Carriage, row.NX, row.Endow = get(a.carriage, "carriage"), get(a.nx, "nx"), get(a.endow, "endow")
	row.Reloc, row.Construction = get(a.reloc, "reloc"), get(a.construction, "construction")
	row.Inter = get(arrNet(a.arrOut, a.arrIn), "inter")
	wear := get(a.wear, "wear")
	units, ok := measure.ToInt64(a.relocUnits)
	if !ok || a.relocOver {
		err = wrap(name, "reloc_units", ErrOverflow)
	}
	row.RelocUnits = units
	if err != nil {
		return GDPRow{}, err
	}
	if row.Cash, err = net(name, "cash", []int64{row.Output}, row.Intermediate); err != nil {
		return GDPRow{}, err
	}
	if row.ValueAdded, err = net(name, "value_added", []int64{row.Cash, row.Pending}); err != nil {
		return GDPRow{}, err
	}
	row.NetProduct, err = net(name, "net_product", []int64{row.ValueAdded}, wear)
	return row, err
}

// arrNet is the value of the units that left a region by shipment less that of
// those that joined it.
func arrNet(out, in acc) acc {
	r := out
	r.addInt(in.v.Neg())
	r.over = out.over || in.over
	return r
}

// expenditure is C + StockChange + Written + Carriage + NX + Inter - Endow, or
// ErrOverflow.
func (r GDPRow) expenditure() (int64, error) {
	return net("GDP", "expenditure", []int64{r.C, r.StockChange, r.Written, r.Carriage, r.NX, r.Inter}, r.Endow)
}

// GDP returns the production and the expenditure views of a window for each
// region and for the realm, the identity between them and its reconciling items
// (spec-indicators.md 7, 9.1 row 1). open and closing are the snapshots that
// bound w (ErrWindowOrder, ErrConfigMismatch otherwise) and p the price table
// (ErrConfigMismatch, ErrRange). With one table for every region the realm's
// Residual is 0 and its Inter and TradeValuation are 0; with regional prices
// TradeValuation is the realm's interregional term. The result carries WarnBasis
// (the net product is on the retirement basis) and WarnBinding when a cap acted.
func GDP(w Window, open, closing economysim.Snapshot, p Prices) (GDPResult, error) {
	in, err := loadAccounts("GDP", w, open, closing, p)
	if err != nil {
		return GDPResult{}, err
	}
	res := GDPResult{ConfigHash: w.ConfigHash, From: w.From, To: w.To, Basis: BasisRetirement, Warnings: withWarnings(w, WarnBasis)}
	var realm regionAcc
	for r := 0; r < w.Regions; r++ {
		a := in.region(w, economysim.RegionID(r), p)
		row, err := a.row(economysim.RegionID(r))
		if err != nil {
			return GDPResult{}, err
		}
		exp, err := row.expenditure()
		if err != nil {
			return GDPResult{}, err
		}
		if row.Residual, err = net("GDP", "residual", []int64{row.ValueAdded, row.Reloc}, exp); err != nil {
			return GDPResult{}, err
		}
		if row.RelocEvents, err = relocEvents(w, economysim.RegionID(r)); err != nil {
			return GDPResult{}, err
		}
		row.Binding = w.Regional[r].Binding
		res.Regions = append(res.Regions, row)
		realm.merge(a)
	}
	if res.Realm, err = realm.row(-1); err != nil {
		return GDPResult{}, err
	}
	res.TradeValuation = res.Realm.Inter
	exp, err := res.Realm.expenditure()
	if err != nil {
		return GDPResult{}, err
	}
	res.Realm.Residual, err = net("GDP", "residual", []int64{res.Realm.ValueAdded}, exp)
	return res, err
}

// accounts is what the three views of a window read: its unit flows and the
// stocks of the two snapshots.
type accounts struct {
	f      flowSet
	u      unitTables
	st, op stockTable
}

// loadAccounts checks the snapshots and the table against the window and reads
// the unit flows and the two stocks.
func loadAccounts(preset string, w Window, open, closing economysim.Snapshot, p Prices) (accounts, error) {
	if err := w.checkSnapshots(preset, open, closing); err != nil {
		return accounts{}, err
	}
	if err := w.checkTable(preset, p); err != nil {
		return accounts{}, err
	}
	f, err := flowsOf(w)
	if err != nil {
		return accounts{}, err
	}
	a := accounts{f: f}
	if a.u, err = loadUnits(w, f); err != nil {
		return accounts{}, err
	}
	if a.st, err = stocksOf(closing, w.Regions, w.Commodities); err != nil {
		return accounts{}, err
	}
	a.op, err = stocksOf(open, w.Regions, w.Commodities)
	return a, err
}

// region accumulates the terms of one region over its commodities.
func (in accounts) region(w Window, r economysim.RegionID, p Prices) regionAcc {
	var a regionAcc
	for c := 0; c < w.Commodities; c++ {
		a.addCell(in.u, in.st, in.op, int(r)*w.Commodities+c, p.At(r, economysim.CommodityID(c)))
	}
	addConstruction(&a, w, r, p)
	return a
}

// addConstruction values the goods of the construction bucket of a region.
func addConstruction(a *regionAcc, w Window, r economysim.RegionID, p Prices) {
	for _, row := range w.Construction {
		if row.Region == r && row.Commodity != economysim.NoCommodity {
			a.construction.add(row.Total, p.At(r, row.Commodity))
		}
	}
}

// relocEvents is the units of AgentRelocated into a region less out of it.
func relocEvents(w Window, r economysim.RegionID) (int64, error) {
	var n acc
	for _, x := range w.Relocations {
		if x.To == r {
			n.addInt(measure.FromInt64(x.Units))
		}
		if x.From == r {
			n.addInt(measure.FromInt64(x.Units).Neg())
		}
	}
	return n.int64("GDP", "reloc_events")
}
