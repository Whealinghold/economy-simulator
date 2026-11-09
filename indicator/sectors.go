package indicator

import (
	"cmp"
	"slices"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/internal/measure"
)

// SectorWorth is an amount by sector: the households, the estate pools, the
// contracts and hires (escrow and cargo in transit) and the treasuries of
// governed jurisdictions (spec-indicators.md 7.5.3).
type SectorWorth struct{ Households, Pool, Contract, Treasury int64 }

// total is the sum of the four sectors, or ErrOverflow.
func (s SectorWorth) total() (int64, error) {
	return net("Sectors", "net_worth", []int64{s.Households, s.Pool, s.Contract, s.Treasury})
}

// RegionMoney is the money account of a region over a window (spec-indicators.md
// 8.2): its change closes exactly on the boundary flows, the money that moved
// with relocated accounts, the contract payments settled in and out and the
// money that crossed the region under a fiscal flow (FR-IND-019). Residual is
// Change less all of them and is 0.
type RegionMoney struct {
	Change, Faucets, Sinks int64
	RelocIn, RelocOut      int64
	SettledIn, SettledOut  int64
	FiscalIn, FiscalOut    int64
	Residual               int64
}

// SectorRow is the money and the net worth of a region, or of the realm
// (Region -1), at the close of a window. MoneyContract and MoneyTreasury are 0
// until the contract and fiscal modules add their snapshot fields; PerHousehold
// is MoneyHouseholds over Households, defined where PerHouseholdOK.
type SectorRow struct {
	Region                                                   economysim.RegionID
	MoneyHouseholds, MoneyPool, MoneyContract, MoneyTreasury int64
	Money                                                    int64
	Households                                               int64
	PerHousehold                                             int64
	PerHouseholdOK                                           bool
	// NW is the net worth by sector at the closing snapshot and DNW its change
	// over the window, both at the price table of the call: money plus the
	// goods the sector holds (the households' include work in progress).
	NW, DNW SectorWorth
	// Account is the money account of the row.
	Account RegionMoney
	// Binding is the counters of the caps that acted in the region.
	Binding BindingCounts
}

// ReasonTotal is the corrected window total of one boundary reason.
type ReasonTotal struct {
	Reason economysim.Reason
	Kind   economysim.Kind
	Total  int64
}

// NetWorthClosing is the closing of the identity of 7.5.3 for the realm at the
// table of the call: the change of the sectors' net worth equals the money
// faucets less sinks, plus the value of the goods faucets less sinks, plus the
// change of work in progress. Residual is 0 at one table (FR-IND-018).
type NetWorthClosing struct {
	DNW, Money, Goods, Pending, Residual int64
}

// SectorsResult is the money by sector, per household, the boundary balances by
// reason, the net worth and its change, and the closing of the identities
// (spec-indicators.md 9.1, row 7). Money held out of circulation (pool, escrow,
// treasury) is its own field, not "the economy".
type SectorsResult struct {
	ConfigHash [32]byte
	From, To   int64
	Regions    []SectorRow
	Realm      SectorRow
	// Flows is the money faucets and sinks of the window by reason, and Balances
	// the cumulative boundary balance of each money reason at the close: how much
	// entered and left.
	Flows    []ReasonTotal
	Balances []economysim.BoundaryRow
	Closing  NetWorthClosing
	Warnings []Warning
}

// Sectors returns the money by sector and the net worth of the sectors at a
// price table, by region and for the realm, with the money account of each
// region and the closing of the identity of 7.5.3. open and closing bound the
// window.
func Sectors(w Window, open, closing economysim.Snapshot, p Prices) (SectorsResult, error) {
	in, err := loadAccounts("Sectors", w, open, closing, p)
	if err != nil {
		return SectorsResult{}, err
	}
	if len(open.Regions) != w.Regions || len(closing.Regions) != w.Regions {
		return SectorsResult{}, wrap("Sectors", "regions", ErrRange)
	}
	f := in.f
	res := SectorsResult{ConfigHash: w.ConfigHash, From: w.From, To: w.To, Warnings: withWarnings(w)}
	var realm sectorAcc
	for r := 0; r < w.Regions; r++ {
		a := in.sectors(w, open, closing, economysim.RegionID(r), p)
		row, err := a.row(w, open, closing, economysim.RegionID(r), f)
		if err != nil {
			return SectorsResult{}, err
		}
		res.Regions = append(res.Regions, row)
		realm.merge(a)
	}
	if res.Realm, err = realm.row(w, open, closing, -1, f); err != nil {
		return SectorsResult{}, err
	}
	if res.Flows, err = reasonTotals(f); err != nil {
		return SectorsResult{}, err
	}
	res.Balances = moneyBalances(closing)
	res.Closing, err = res.close(f, p, realm)
	return res, err
}

// sectorAcc accumulates the stocks of a region's sectors.
type sectorAcc struct {
	nw, dnw [4]acc // households, pool, contract, treasury
	goods   acc    // the value of the goods flows, faucets less sinks
	pend    acc    // the value of the change of work in progress
}

// merge adds another region.
func (a *sectorAcc) merge(b sectorAcc) {
	for i := range a.nw {
		a.nw[i].merge(b.nw[i])
		a.dnw[i].merge(b.dnw[i])
	}
	a.goods.merge(b.goods)
	a.pend.merge(b.pend)
}

// sectors values the goods the sectors of a region hold at the two boundaries.
func (in accounts) sectors(w Window, open, closing economysim.Snapshot, r economysim.RegionID, p Prices) sectorAcc {
	var a sectorAcc
	for c := 0; c < w.Commodities; c++ {
		i, pr := int(r)*w.Commodities+c, p.At(r, economysim.CommodityID(c))
		// Term by term, so a difference of units never wraps (FR-IND-009).
		a.nw[0].add(in.st.held[i], pr)
		a.nw[0].sub(in.st.estate[i], pr)
		a.nw[0].add(in.st.pend[i], pr)
		a.dnw[0].add(in.st.held[i], pr)
		a.dnw[0].sub(in.st.estate[i], pr)
		a.dnw[0].sub(in.op.held[i], pr)
		a.dnw[0].add(in.op.estate[i], pr)
		a.dnw[0].add(in.st.pend[i], pr)
		a.dnw[0].sub(in.op.pend[i], pr)
		a.nw[1].add(in.st.estate[i], pr)
		a.dnw[1].add(in.st.estate[i], pr)
		a.dnw[1].sub(in.op.estate[i], pr)
		a.nw[2].add(in.st.transit[i], pr)
		a.dnw[2].add(in.st.transit[i], pr)
		a.dnw[2].sub(in.op.transit[i], pr)
		a.pend.add(in.st.pend[i], pr)
		a.pend.sub(in.op.pend[i], pr)
	}
	return a
}

// row builds the result row of a region, or of the realm for region -1.
func (a sectorAcc) row(w Window, open, closing economysim.Snapshot, r economysim.RegionID, f flowSet) (SectorRow, error) {
	row := SectorRow{Region: r}
	var err error
	worth := func(x [4]acc) SectorWorth {
		var s SectorWorth
		for i, dst := range []*int64{&s.Households, &s.Pool, &s.Contract, &s.Treasury} {
			v, e := x[i].int64("Sectors", "net_worth")
			*dst, err = v, firstErr(err, e)
		}
		return s
	}
	row.NW, row.DNW = worth(a.nw), worth(a.dnw)
	if err != nil {
		return SectorRow{}, err
	}
	dh, dp, err := row.fillMoney(w, open, closing, r)
	if err != nil {
		return SectorRow{}, err
	}
	dh.addInt(measure.FromInt64(row.DNW.Households))
	dp.addInt(measure.FromInt64(row.DNW.Pool))
	if row.DNW.Households, err = dh.int64("Sectors", "net_worth"); err != nil {
		return SectorRow{}, err
	}
	if row.DNW.Pool, err = dp.int64("Sectors", "net_worth"); err != nil {
		return SectorRow{}, err
	}
	if row.NW.Households, err = net("Sectors", "net_worth", []int64{row.NW.Households, row.MoneyHouseholds}); err != nil {
		return SectorRow{}, err
	}
	if row.NW.Pool, err = net("Sectors", "net_worth", []int64{row.NW.Pool, row.MoneyPool}); err != nil {
		return SectorRow{}, err
	}
	if q, ok := measure.RoundDiv(measure.FromInt64(row.MoneyHouseholds), measure.FromInt64(row.Households)); ok {
		row.PerHousehold, row.PerHouseholdOK = measure.ToInt64(q)
	}
	row.Account, err = account(w, open, closing, r, f)
	return row, err
}

// fillMoney sums the money and households of the region's cells into row and returns the
// open-to-close deltas of the household and pool money.
func (row *SectorRow) fillMoney(w Window, open, closing economysim.Snapshot, r economysim.RegionID) (dh, dp acc, err error) {
	var mh, mp, hh acc
	for _, g := range regionsOf(w, r) {
		cr, or := closing.Regions[g], open.Regions[g]
		mh.addInt(measure.FromInt64(int64(cr.MoneyHouseholds)))
		mp.addInt(measure.FromInt64(int64(cr.MoneyPool)))
		hh.addInt(measure.FromInt64(cr.Households))
		dh.addInt(measure.FromInt64(int64(cr.MoneyHouseholds)))
		dh.addInt(measure.FromInt64(int64(or.MoneyHouseholds)).Neg())
		dp.addInt(measure.FromInt64(int64(cr.MoneyPool)))
		dp.addInt(measure.FromInt64(int64(or.MoneyPool)).Neg())
		if row.Binding, err = addBinding(row.Binding, w.Regional[g].Binding); err != nil {
			return dh, dp, err
		}
	}
	if row.MoneyHouseholds, err = mh.int64("Sectors", "money_households"); err != nil {
		return dh, dp, err
	}
	if row.MoneyPool, err = mp.int64("Sectors", "money_pool"); err != nil {
		return dh, dp, err
	}
	if row.Households, err = hh.int64("Sectors", "households"); err != nil {
		return dh, dp, err
	}
	// The contract and treasury money are not in the snapshot yet (plans 2-15, POL):
	// MoneyContract and MoneyTreasury stay 0 and are read from RegionRow when it has them.
	if row.Money, err = net("Sectors", "money", []int64{row.MoneyHouseholds, row.MoneyPool, row.MoneyContract, row.MoneyTreasury}); err != nil {
		return dh, dp, err
	}
	return dh, dp, nil
}

// firstErr returns the first non-nil error.
func firstErr(a, b error) error {
	if a != nil {
		return a
	}
	return b
}

// regionsOf returns the regions a row covers: the region, or every region for
// the realm (-1).
func regionsOf(w Window, r economysim.RegionID) []economysim.RegionID {
	if r >= 0 {
		return []economysim.RegionID{r}
	}
	all := make([]economysim.RegionID, w.Regions)
	for i := range all {
		all[i] = economysim.RegionID(i)
	}
	return all
}

// addBinding adds two sets of counters, or ErrOverflow.
func addBinding(a, b BindingCounts) (BindingCounts, error) {
	for _, p := range [...][2]*int64{{&a.Unfunded, &b.Unfunded}, {&a.RegionCap, &b.RegionCap}, {&a.RoleCap, &b.RoleCap},
		{&a.Cooldown, &b.Cooldown}, {&a.Hysteresis, &b.Hysteresis}, {&a.ExitCap, &b.ExitCap}, {&a.Refused, &b.Refused},
		{&a.BirthsCapped, &b.BirthsCapped}, {&a.BirthsUnfunded, &b.BirthsUnfunded}, {&a.DeathsCut, &b.DeathsCut},
		{&a.DeathsHeld, &b.DeathsHeld}, {&a.FlowShort, &b.FlowShort}, {&a.FlowLimit, &b.FlowLimit}} {
		v, err := net("Sectors", "binding", []int64{*p[0], *p[1]})
		if err != nil {
			return BindingCounts{}, err
		}
		*p[0] = v
	}
	return a, nil
}

// account builds the money account of a region (or the realm): the change of
// its money against the boundary flows and the money that crossed it. Every sum
// is checked (FR-IND-009).
func account(w Window, open, closing economysim.Snapshot, r economysim.RegionID, f flowSet) (RegionMoney, error) {
	var change, faucets, sinks, relocIn, relocOut, settledIn, settledOut, fiscalIn, fiscalOut acc
	neg := func(v int64) measure.Int128 { return measure.FromInt64(v).Neg() }
	for _, g := range regionsOf(w, r) {
		cr, or := closing.Regions[g], open.Regions[g]
		change.addInt(measure.FromInt64(int64(cr.MoneyHouseholds)))
		change.addInt(measure.FromInt64(int64(cr.MoneyPool)))
		change.addInt(neg(int64(or.MoneyHouseholds)))
		change.addInt(neg(int64(or.MoneyPool)))
		for _, x := range f.rows {
			if x.Region != g || x.Commodity != economysim.NoCommodity {
				continue
			}
			if x.Kind == economysim.KindFaucet {
				faucets.addInt(measure.FromInt64(x.Total))
			} else {
				sinks.addInt(measure.FromInt64(x.Total))
			}
		}
		fiscalIn.addInt(measure.FromInt64(int64(w.Regional[g].FiscalIn)))
		fiscalOut.addInt(measure.FromInt64(int64(w.Regional[g].FiscalOut)))
	}
	for _, x := range w.Relocations {
		relocIn.addInt(measure.FromInt64(inRegion(r, x.To, int64(x.Money))))
		relocOut.addInt(measure.FromInt64(inRegion(r, x.From, int64(x.Money))))
	}
	for _, x := range w.Settlements {
		settledIn.addInt(measure.FromInt64(inRegion(r, x.Region, int64(x.Paid))))
		settledOut.addInt(measure.FromInt64(inRegion(r, x.To, int64(x.Paid))))
	}
	var m RegionMoney
	var err error
	for _, t := range [...]struct {
		dst *int64
		src acc
	}{{&m.Change, change}, {&m.Faucets, faucets}, {&m.Sinks, sinks}, {&m.RelocIn, relocIn}, {&m.RelocOut, relocOut},
		{&m.SettledIn, settledIn}, {&m.SettledOut, settledOut}, {&m.FiscalIn, fiscalIn}, {&m.FiscalOut, fiscalOut}} {
		if *t.dst, err = t.src.int64("Sectors", "account"); err != nil {
			return RegionMoney{}, err
		}
	}
	m.Residual, err = net("Sectors", "residual", []int64{m.Change, m.Sinks, m.RelocOut, m.SettledOut, m.FiscalOut},
		m.Faucets, m.RelocIn, m.SettledIn, m.FiscalIn)
	return m, err
}

// inRegion returns v when region g is the row's region r, or any region for the
// realm (-1). The realm sums relocations and settlements, which cancel.
func inRegion(r, g economysim.RegionID, v int64) int64 {
	if r < 0 || r == g {
		return v
	}
	return 0
}

// reasonTotals sums the money reasons of the corrected flows over the regions,
// in reason order.
func reasonTotals(f flowSet) ([]ReasonTotal, error) {
	var out []ReasonTotal
	for _, x := range f.rows {
		if x.Commodity != economysim.NoCommodity {
			continue
		}
		i, found := slices.BinarySearchFunc(out, x.Reason, func(r ReasonTotal, k economysim.Reason) int { return cmp.Compare(r.Reason, k) })
		if !found {
			out = slices.Insert(out, i, ReasonTotal{Reason: x.Reason, Kind: x.Kind})
		}
		t, err := net("Sectors", "flows", []int64{out[i].Total, x.Total})
		if err != nil {
			return nil, err
		}
		out[i].Total = t
	}
	return out, nil
}

// moneyBalances returns the cumulative boundary balances of the money reasons.
func moneyBalances(s economysim.Snapshot) []economysim.BoundaryRow {
	var out []economysim.BoundaryRow
	for _, b := range s.Boundary {
		if b.Commodity == economysim.NoCommodity {
			out = append(out, b)
		}
	}
	return out
}

// close builds the closing of the identity of 7.5.3 for the realm at table p:
// the change of the net worth of the sectors against the money flows, the value
// of the goods flows (faucets less sinks) and the change of work in progress.
func (res SectorsResult) close(f flowSet, p Prices, realm sectorAcc) (NetWorthClosing, error) {
	var goods acc
	for _, x := range f.rows {
		if x.Commodity == economysim.NoCommodity {
			continue
		}
		v := measure.MulInt64(x.Total, int64(p.At(x.Region, x.Commodity)))
		if x.Kind != economysim.KindFaucet {
			v = v.Neg()
		}
		goods.addInt(v)
	}
	var c NetWorthClosing
	var err error
	if c.DNW, err = res.Realm.DNW.total(); err != nil {
		return c, err
	}
	if c.Money, err = net("Sectors", "money", []int64{res.Realm.Account.Faucets}, res.Realm.Account.Sinks); err != nil {
		return c, err
	}
	if c.Goods, err = goods.int64("Sectors", "goods"); err != nil {
		return c, err
	}
	if c.Pending, err = realm.pend.int64("Sectors", "pending"); err != nil {
		return c, err
	}
	c.Residual, err = net("Sectors", "residual", []int64{c.DNW}, c.Money, c.Goods, c.Pending)
	return c, err
}
