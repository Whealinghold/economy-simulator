package indicator

import (
	"slices"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/internal/measure"
)

// HouseholdsRow is the households of one (region, role) over a window, with the
// account of change that closes them (spec-indicators.md 9.1 row 9). Region and
// Role are -1 on the row of the whole realm. Open and Close are households;
// OpenAgents and CloseAgents are agents, which are not households.
type HouseholdsRow struct {
	Region                  economysim.RegionID
	Role                    economysim.RoleID
	Open, Close             int64
	OpenAgents, CloseAgents int64
	Spawned, Bankrupted     int64
	WeightChanges           int64
	RoleChanges, Relocated  int64
	Births, Deaths          int64
	Residual                int64
}

// HouseholdsResult is the households of a window by (region, role) and in all,
// and the annual rate of natural increase in basis points of the household-ticks
// (Births - Deaths over the households there were, stated per year).
type HouseholdsResult struct {
	ConfigHash [32]byte
	From, To   int64
	Rows       []HouseholdsRow
	Total      HouseholdsRow
	Rate       Index
}

// Households returns the households and agents of each (region, role) at the two
// boundaries and the account of change of the window: Close - Open equals the
// weights spawned less those bankrupted, plus the weight changes (births done
// are among them), plus the net role changes and relocations; Residual is the
// difference and is 0 when the tally and the snapshots agree. A migrant on a
// journey is in its origin until it is relocated, and an estate is in neither
// count (spec-indicators.md 6.1, 6.3). ticksPerYear is the calendar's.
func Households(w Window, open, closing economysim.Snapshot, ticksPerYear int64) (HouseholdsResult, error) {
	oa, ca, err := householdsInputs(w, open, closing, ticksPerYear)
	if err != nil {
		return HouseholdsResult{}, err
	}
	res := HouseholdsResult{ConfigHash: w.ConfigHash, From: w.From, To: w.To, Total: HouseholdsRow{Region: -1, Role: -1}}
	var ticks acc
	for i, p := range w.People {
		row, err := householdsRow(w, i, p, oa[i], ca[i])
		if err != nil {
			return HouseholdsResult{}, err
		}
		res.Rows = append(res.Rows, row)
		if err = res.Total.add(row); err != nil {
			return HouseholdsResult{}, err
		}
		ticks.addInt(measure.FromInt64(p.HouseholdTicks))
	}
	growth, err := net("Households", "rate", []int64{res.Total.Births}, res.Total.Deaths)
	if err != nil {
		return HouseholdsResult{}, err
	}
	res.Rate, err = ratioWide("Households", "rate", measure.MulInt64(growth, ticksPerYear), ticks.v)
	if ticks.over {
		err = wrap("Households", "ticks", ErrOverflow)
	}
	return res, err
}

// householdsInputs checks the inputs of Households and returns the agent counts at both boundaries.
func householdsInputs(w Window, open, closing economysim.Snapshot, ticksPerYear int64) (oa, ca []int64, err error) {
	if err := w.checkSnapshots("Households", open, closing); err != nil {
		return nil, nil, err
	}
	if ticksPerYear < 1 {
		return nil, nil, wrap("Households", "ticks per year", ErrRange)
	}
	if err := w.checkPeople("Households"); err != nil {
		return nil, nil, err
	}
	if oa, err = agentsOf(w, open); err != nil {
		return nil, nil, err
	}
	ca, err = agentsOf(w, closing)
	return oa, ca, err
}

// checkPeople returns ErrRange for a hand-built window whose people table cannot
// be indexed by (region, role) or whose household tables are shorter than it,
// which the presets would otherwise index out of range or divide by Roles 0.
func (w Window) checkPeople(preset string) error {
	n := len(w.People)
	if n > 0 && (w.Roles < 1 || n > w.Regions*w.Roles || len(w.OpenHouseholds) < n || len(w.CloseHouseholds) < n) {
		return wrap(preset, "people", ErrRange)
	}
	return nil
}

// householdsRow is the row of the people cell i.
func householdsRow(w Window, i int, p PeopleTally, openAgents, closeAgents int64) (HouseholdsRow, error) {
	r := HouseholdsRow{
		Region: economysim.RegionID(i / w.Roles), Role: economysim.RoleID(i % w.Roles),
		Open: w.OpenHouseholds[i], Close: w.CloseHouseholds[i], OpenAgents: openAgents, CloseAgents: closeAgents,
		Spawned: p.Spawned, Bankrupted: p.Bankrupted, WeightChanges: p.WeightDelta,
		Births: p.BirthsDone, Deaths: p.DeathsDone,
	}
	var err error
	if r.RoleChanges, err = net("Households", "role_changes", []int64{p.RoleIn}, p.RoleOut); err != nil {
		return HouseholdsRow{}, err
	}
	if r.Relocated, err = net("Households", "relocated", []int64{p.RelocIn}, p.RelocOut); err != nil {
		return HouseholdsRow{}, err
	}
	r.Residual, err = net("Households", "residual", []int64{r.Close, r.Bankrupted}, r.Open, r.Spawned, r.WeightChanges, r.RoleChanges, r.Relocated)
	return r, err
}

// add adds the counts of another row, or returns ErrOverflow.
func (r *HouseholdsRow) add(o HouseholdsRow) error {
	for _, p := range [...][2]*int64{{&r.Open, &o.Open}, {&r.Close, &o.Close}, {&r.OpenAgents, &o.OpenAgents},
		{&r.CloseAgents, &o.CloseAgents}, {&r.Spawned, &o.Spawned}, {&r.Bankrupted, &o.Bankrupted},
		{&r.WeightChanges, &o.WeightChanges}, {&r.RoleChanges, &o.RoleChanges}, {&r.Relocated, &o.Relocated},
		{&r.Births, &o.Births}, {&r.Deaths, &o.Deaths}, {&r.Residual, &o.Residual}} {
		v, err := net("Households", "total", []int64{*p[0], *p[1]})
		if err != nil {
			return err
		}
		*p[0] = v
	}
	return nil
}

// agentsOf returns the agents by (region, role) of a snapshot, at region * Roles
// + role. A role row outside the names table is ErrRange.
func agentsOf(w Window, s economysim.Snapshot) ([]int64, error) {
	out := make([]int64, w.Regions*w.Roles)
	for _, r := range s.Roles {
		if r.Region < 0 || int(r.Region) >= w.Regions || r.Role < 0 || int(r.Role) >= w.Roles {
			return nil, wrap("Households", "role row", ErrRange)
		}
		out[int(r.Region)*w.Roles+int(r.Role)] = r.Agents
	}
	return out, nil
}

// UnfedRow is the unfed figures of one (region, role), or of the realm
// (Region and Role -1). Unfed is the household-ticks unfed over the window and
// HouseholdTicks the sum of the running counts at each tick's start; Share is
// their ratio. AtClose is the share of households unfed in the last consumption
// before the closing boundary.
type UnfedRow struct {
	Region         economysim.RegionID
	Role           economysim.RoleID
	Unfed          int64
	HouseholdTicks int64
	Share          Index
	MaxHunger      int64
	AtClose        Index
}

// UnfedResult is the unfed share by (region, role) and in all.
type UnfedResult struct {
	ConfigHash [32]byte
	From, To   int64
	Rows       []UnfedRow
	Total      UnfedRow
	Warnings   []Warning
}

// Unfed returns the share of household-ticks unfed over the window, by (region,
// role) and in all, and the largest hunger (spec-indicators.md 9.1 row 12). The
// denominator is the running count of households at the start of each tick, not
// the households of the events, because UpkeepShortfall is not emitted when all
// are fed; a household is counted by its weight, so an agent of weight 6 unfed
// is 6. closing supplies the share at the boundary.
func Unfed(w Window, closing economysim.Snapshot) (UnfedResult, error) {
	if err := w.checkClosing("Unfed", closing); err != nil {
		return UnfedResult{}, err
	}
	if err := w.checkPeople("Unfed"); err != nil {
		return UnfedResult{}, err
	}
	atClose, err := unfedAtClose(w, closing)
	if err != nil {
		return UnfedResult{}, err
	}
	res := UnfedResult{ConfigHash: w.ConfigHash, From: w.From, To: w.To, Warnings: withWarnings(w), Total: UnfedRow{Region: -1, Role: -1}}
	var unfed, ticks, closeUnfed, closeHH acc
	for i, p := range w.People {
		row := UnfedRow{Region: economysim.RegionID(i / w.Roles), Role: economysim.RoleID(i % w.Roles), Unfed: p.Unfed, HouseholdTicks: p.HouseholdTicks, MaxHunger: p.MaxHunger, AtClose: atClose[i].share}
		if row.Share, err = ratioWide("Unfed", "share", measure.FromInt64(p.Unfed), measure.FromInt64(p.HouseholdTicks)); err != nil {
			return UnfedResult{}, err
		}
		res.Rows = append(res.Rows, row)
		unfed.addInt(measure.FromInt64(p.Unfed))
		ticks.addInt(measure.FromInt64(p.HouseholdTicks))
		closeUnfed.addInt(measure.FromInt64(atClose[i].unfed))
		closeHH.addInt(measure.FromInt64(w.CloseHouseholds[i]))
		res.Total.MaxHunger = max(res.Total.MaxHunger, p.MaxHunger)
	}
	return res.finish(unfed, ticks, closeUnfed, closeHH)
}

// finish fills the realm row from the sums.
func (res UnfedResult) finish(unfed, ticks, closeUnfed, closeHH acc) (UnfedResult, error) {
	var err error
	if res.Total.Unfed, err = unfed.int64("Unfed", "unfed"); err != nil {
		return UnfedResult{}, err
	}
	if res.Total.HouseholdTicks, err = ticks.int64("Unfed", "household_ticks"); err != nil {
		return UnfedResult{}, err
	}
	if res.Total.Share, err = ratioWide("Unfed", "share", unfed.v, ticks.v); err != nil {
		return UnfedResult{}, err
	}
	res.Total.AtClose, err = ratioWide("Unfed", "at_close", closeUnfed.v, closeHH.v)
	return res, err
}

// closeShare is the unfed households and their share of a (region, role) at the
// closing boundary.
type closeShare struct {
	unfed int64
	share Index
}

// unfedAtClose reads the unfed households of the closing snapshot by cell.
func unfedAtClose(w Window, s economysim.Snapshot) ([]closeShare, error) {
	out := make([]closeShare, w.Regions*w.Roles)
	for _, r := range s.Roles {
		if r.Region < 0 || int(r.Region) >= w.Regions || r.Role < 0 || int(r.Role) >= w.Roles {
			return nil, wrap("Unfed", "role row", ErrRange)
		}
		i := int(r.Region)*w.Roles + int(r.Role)
		sh, err := ratioWide("Unfed", "at_close", measure.FromInt64(r.Unfed), measure.FromInt64(r.Households))
		if err != nil {
			return nil, err
		}
		out[i] = closeShare{unfed: r.Unfed, share: sh}
	}
	return out, nil
}

// ShareRow is the share of a role in the households of its region and of the
// realm, at the two boundaries. A role with no households has share 0 and is
// reported; a region with no households has its shares undefined.
type ShareRow struct {
	Region                economysim.RegionID
	Role                  economysim.RoleID
	Open, Close           int64
	OpenShare, CloseShare Index
	OpenRealm, CloseRealm Index
}

// RoleSharesResult is the share rows of a window.
type RoleSharesResult struct {
	ConfigHash [32]byte
	From, To   int64
	Rows       []ShareRow
}

// RoleShares returns the households of each role as a share of its region's and
// of the realm's, at the start and the end of the window (spec-indicators.md 9.1
// row 16). A role that disappears is a row with share 0, never dropped. The
// swing over a series of windows is Swing of the series of one share.
func RoleShares(w Window) (RoleSharesResult, error) {
	res := RoleSharesResult{ConfigHash: w.ConfigHash, From: w.From, To: w.To}
	if err := w.checkPeople("RoleShares"); err != nil {
		return RoleSharesResult{}, err
	}
	if len(w.OpenHouseholds) > len(w.People) || len(w.CloseHouseholds) > len(w.People) {
		return RoleSharesResult{}, wrap("RoleShares", "people", ErrRange) // byRegion indexes by region * Roles + role
	}
	t, err := shareTotalsOf(w)
	if err != nil {
		return RoleSharesResult{}, err
	}
	openReg, closeReg, openAll, closeAll := t.openReg, t.closeReg, t.openAll, t.closeAll
	for i := range w.People {
		g := i / w.Roles
		row := ShareRow{Region: economysim.RegionID(g), Role: economysim.RoleID(i % w.Roles), Open: w.OpenHouseholds[i], Close: w.CloseHouseholds[i]}
		for _, f := range []struct {
			dst    *Index
			n, den int64
		}{{&row.OpenShare, row.Open, openReg[g]}, {&row.CloseShare, row.Close, closeReg[g]}, {&row.OpenRealm, row.Open, openAll}, {&row.CloseRealm, row.Close, closeAll}} {
			var err error
			if *f.dst, err = ratioWide("RoleShares", "share", measure.FromInt64(f.n), measure.FromInt64(f.den)); err != nil {
				return RoleSharesResult{}, err
			}
		}
		res.Rows = append(res.Rows, row)
	}
	return res, nil
}

// shareTotals are the region and realm household totals at both boundaries.
type shareTotals struct {
	openReg, closeReg []int64
	openAll, closeAll int64
}

// shareTotalsOf sums the household tables by region and over the realm.
func shareTotalsOf(w Window) (t shareTotals, err error) {
	if t.openReg, err = byRegion(w, w.OpenHouseholds); err != nil {
		return t, err
	}
	if t.closeReg, err = byRegion(w, w.CloseHouseholds); err != nil {
		return t, err
	}
	if t.openAll, err = net("RoleShares", "households", w.OpenHouseholds); err != nil {
		return t, err
	}
	t.closeAll, err = net("RoleShares", "households", w.CloseHouseholds)
	return t, err
}

// byRegion sums a (region, role) table by region, or returns ErrOverflow.
func byRegion(w Window, t []int64) ([]int64, error) {
	out := make([]int64, w.Regions)
	for i, v := range t {
		sum, err := net("RoleShares", "households", []int64{out[i/w.Roles], v})
		if err != nil {
			return nil, err
		}
		out[i/w.Roles] = sum
	}
	return out, nil
}

// Swing returns the largest minus the smallest value of a series of shares, in
// basis points, and false for an empty series (spec-indicators.md 9.1 row 16).
func Swing(points []Point) (int64, bool) {
	if len(points) == 0 {
		return 0, false
	}
	s := Stats(points)
	d, fits := measure.ToInt64(mustSub(s.High.Value, s.Low.Value))
	return d, fits // a span past int64 is undefined, not wrapped
}

// WageRow is the real wage of one (region, role): the observed wage per working
// household-tick over the cost of the role's upkeep, Ratio, in basis points of
// one tick of upkeep. A role with no observed wage has HasWage false and no
// Ratio; it is absent, not zero.
type WageRow struct {
	Region  economysim.RegionID
	Role    economysim.RoleID
	Wage    economysim.Money
	Upkeep  int64
	HasWage bool
	Ratio   Index
}

// RealWageResult is the real wage by (region, role) at a boundary.
type RealWageResult struct {
	ConfigHash [32]byte
	Tick       int64
	Rows       []WageRow
}

// checkWage checks the snapshot and the price table against the configuration.
func checkWage(cfg economysim.Config, s economysim.Snapshot, p Prices) error {
	if s.ConfigHash != economysim.HashConfig(cfg) {
		return wrap("RealWage", "snapshot", ErrConfigMismatch)
	}
	if err := p.Compatible(s.ConfigHash); err != nil {
		return err
	}
	if p.Regions != len(cfg.Regions) || p.Commodities != len(cfg.Commodities) || len(p.Cells) != p.Regions*p.Commodities {
		return wrap("RealWage", "prices", ErrRange)
	}
	return nil
}

// RealWage returns, for each (region, role) of the snapshot, rdiv(10 000 * wage,
// upkeep cost) with the upkeep of the role (its own list, else the world's) plus
// the region's extra, at the prices of p (spec-indicators.md 9.1 row 14). cfg is
// the configuration of the snapshot's world as World.Config returns it.
func RealWage(cfg economysim.Config, s economysim.Snapshot, p Prices) (RealWageResult, error) {
	if err := checkWage(cfg, s, p); err != nil {
		return RealWageResult{}, err
	}
	res, ids := RealWageResult{ConfigHash: s.ConfigHash, Tick: s.Tick}, commodityIDs(cfg)
	for _, r := range s.Roles {
		if r.Role < 0 || int(r.Role) >= len(cfg.Roles) || r.Region < 0 || int(r.Region) >= len(cfg.Regions) {
			return RealWageResult{}, wrap("RealWage", "role row", ErrRange)
		}
		row := WageRow{Region: r.Region, Role: r.Role, Wage: r.WageObserved, HasWage: r.HasWage}
		if r.HasWage {
			if err := row.price(cfg, ids, p); err != nil {
				return RealWageResult{}, err
			}
		}
		res.Rows = append(res.Rows, row)
	}
	return res, nil
}

// price fills the upkeep cost and the ratio of a row with a wage.
func (row *WageRow) price(cfg economysim.Config, ids map[string]economysim.CommodityID, p Prices) error {
	list := cfg.World.Upkeep
	if own := cfg.Roles[row.Role].Upkeep; len(own) > 0 {
		list = own
	}
	var cost acc
	for _, it := range slices.Concat(list, cfg.Regions[row.Region].ExtraUpkeep) {
		c, ok := ids[it.Commodity]
		if !ok {
			return wrap("RealWage", it.Commodity, ErrRange)
		}
		cost.add(int64(it.Quantity), p.At(row.Region, c))
	}
	var err error
	if row.Upkeep, err = cost.int64("RealWage", "upkeep"); err != nil {
		return err
	}
	row.Ratio, err = ratioWide("RealWage", "ratio", measure.FromInt64(int64(row.Wage)), cost.v)
	return err
}
