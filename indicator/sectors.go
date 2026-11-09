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

// total is the sum of the four sectors.
func (s SectorWorth) total() int64 { return s.Households + s.Pool + s.Contract + s.Treasury }

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
	f, err := flowsOf(w)
	if err != nil {
		return SectorsResult{}, err
	}
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
	res.Flows, res.Balances = reasonTotals(f), moneyBalances(closing)
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
		agentsClose, agentsOpen := in.st.held[i]-in.st.estate[i], in.op.held[i]-in.op.estate[i]
		a.nw[0].add(agentsClose+in.st.pend[i], pr)
		a.dnw[0].add(agentsClose-agentsOpen+in.st.pend[i]-in.op.pend[i], pr)
		a.nw[1].add(in.st.estate[i], pr)
		a.dnw[1].add(in.st.estate[i]-in.op.estate[i], pr)
		a.nw[2].add(in.st.transit[i], pr)
		a.dnw[2].add(in.st.transit[i]-in.op.transit[i], pr)
		a.pend.add(in.st.pend[i]-in.op.pend[i], pr)
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
	for _, g := range regionsOf(w, r) {
		cr, or := closing.Regions[g], open.Regions[g]
		row.MoneyHouseholds += int64(cr.MoneyHouseholds)
		row.MoneyPool += int64(cr.MoneyPool)
		row.Households += cr.Households
		row.DNW.Households += int64(cr.MoneyHouseholds) - int64(or.MoneyHouseholds)
		row.DNW.Pool += int64(cr.MoneyPool) - int64(or.MoneyPool)
		row.Binding = addBinding(row.Binding, w.Regional[g].Binding)
	}
	// The contract and treasury money are not in the snapshot yet (plans 2-15, POL):
	// MoneyContract and MoneyTreasury stay 0 and are read from RegionRow when it has them.
	row.Money = row.MoneyHouseholds + row.MoneyPool + row.MoneyContract + row.MoneyTreasury
	row.NW.Households += row.MoneyHouseholds
	row.NW.Pool += row.MoneyPool
	if q, ok := measure.RoundDiv(measure.FromInt64(row.MoneyHouseholds), measure.FromInt64(row.Households)); ok {
		row.PerHousehold, row.PerHouseholdOK = measure.ToInt64(q)
	}
	row.Account = account(w, open, closing, r, f)
	return row, nil
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

// addBinding adds two sets of counters.
func addBinding(a, b BindingCounts) BindingCounts {
	for _, p := range [...][2]*int64{{&a.Unfunded, &b.Unfunded}, {&a.RegionCap, &b.RegionCap}, {&a.RoleCap, &b.RoleCap},
		{&a.Cooldown, &b.Cooldown}, {&a.Hysteresis, &b.Hysteresis}, {&a.ExitCap, &b.ExitCap}, {&a.Refused, &b.Refused},
		{&a.BirthsCapped, &b.BirthsCapped}, {&a.BirthsUnfunded, &b.BirthsUnfunded}, {&a.DeathsCut, &b.DeathsCut},
		{&a.DeathsHeld, &b.DeathsHeld}, {&a.FlowShort, &b.FlowShort}, {&a.FlowLimit, &b.FlowLimit}} {
		*p[0] += *p[1]
	}
	return a
}

// account builds the money account of a region (or the realm): the change of
// its money against the boundary flows and the money that crossed it.
func account(w Window, open, closing economysim.Snapshot, r economysim.RegionID, f flowSet) RegionMoney {
	var m RegionMoney
	for _, g := range regionsOf(w, r) {
		cr, or := closing.Regions[g], open.Regions[g]
		m.Change += int64(cr.MoneyHouseholds) + int64(cr.MoneyPool) - int64(or.MoneyHouseholds) - int64(or.MoneyPool)
		for _, x := range f.rows {
			if x.Region != g || x.Commodity != economysim.NoCommodity {
				continue
			}
			if x.Kind == economysim.KindFaucet {
				m.Faucets += x.Total
			} else {
				m.Sinks += x.Total
			}
		}
		m.FiscalIn += int64(w.Regional[g].FiscalIn)
		m.FiscalOut += int64(w.Regional[g].FiscalOut)
	}
	for _, x := range w.Relocations {
		m.RelocIn, m.RelocOut = m.RelocIn+inRegion(r, x.To, int64(x.Money)), m.RelocOut+inRegion(r, x.From, int64(x.Money))
	}
	for _, x := range w.Settlements {
		m.SettledIn, m.SettledOut = m.SettledIn+inRegion(r, x.Region, int64(x.Paid)), m.SettledOut+inRegion(r, x.To, int64(x.Paid))
	}
	m.Residual = m.Change - (m.Faucets - m.Sinks + m.RelocIn - m.RelocOut + m.SettledIn - m.SettledOut + m.FiscalIn - m.FiscalOut)
	return m
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
func reasonTotals(f flowSet) []ReasonTotal {
	var out []ReasonTotal
	for _, x := range f.rows {
		if x.Commodity != economysim.NoCommodity {
			continue
		}
		i, found := slices.BinarySearchFunc(out, x.Reason, func(r ReasonTotal, k economysim.Reason) int { return cmp.Compare(r.Reason, k) })
		if !found {
			out = slices.Insert(out, i, ReasonTotal{Reason: x.Reason, Kind: x.Kind})
		}
		out[i].Total += x.Total
	}
	return out
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
		sign := int64(1)
		if x.Kind != economysim.KindFaucet {
			sign = -1
		}
		goods.add(sign*x.Total, p.At(x.Region, x.Commodity))
	}
	var c NetWorthClosing
	var err error
	c.DNW = res.Realm.DNW.total()
	c.Money = res.Realm.Account.Faucets - res.Realm.Account.Sinks
	if c.Goods, err = goods.int64("Sectors", "goods"); err != nil {
		return c, err
	}
	if c.Pending, err = realm.pend.int64("Sectors", "pending"); err != nil {
		return c, err
	}
	c.Residual = c.DNW - (c.Money + c.Goods + c.Pending)
	return c, nil
}
