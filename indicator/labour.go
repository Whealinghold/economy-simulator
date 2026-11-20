package indicator

import (
	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/internal/measure"
)

// UnemploymentRow is the households that offered their labour in a window and what
// became of them, for one (region, worker role). Region or Role is -1 on the row that
// sums over it. Offered, Hired and NoVacancy are household-ticks. Rate is
// rdiv(10 000 * NoVacancy, Offered), OK false where nothing was offered.
type UnemploymentRow struct {
	Region                    economysim.RegionID
	Role                      economysim.RoleID
	Offered, Hired, NoVacancy int64
	Rate                      Index
}

// RefusalRow is the positions a formation check closed in the window, by the reason of
// the check (spec-employment.md 5.5). Only reasons with a count above 0 are listed.
type RefusalRow struct {
	Reason economysim.Reason
	Count  int64
}

// UnemploymentResult is the rate by (region, worker role), by region and in all, with
// the formation refusals by reason beside it (spec-indicators.md 9.1 row 22).
type UnemploymentResult struct {
	ConfigHash [32]byte
	From, To   int64
	// Rows are the worker roles that offered, by region and role; Regions the sums by
	// region (Role -1) and Total the realm (Region and Role -1).
	Rows, Regions []UnemploymentRow
	Total         UnemploymentRow
	// ByReason are the refusals of the formation, summed over the realm, in the order
	// of the checks.
	ByReason []RefusalRow
	Warnings []Warning
}

// Unemployment returns the share of the households that offered their labour for hire
// in a window and were not matched, by region and worker role (spec-indicators.md 9.1
// row 22, FR-IND-056): rdiv(10 000 * NoVacancy, Offered) of the household-ticks summed
// over the window. A household that rested, was in the own-use mode, was a trainee, away
// or on a journey offered nothing and is not counted, so a row with nothing offered has
// no rate. The rate does not say whether the unemployment is structural or frictional;
// ByReason gives the refusals to split it. Households, not agents.
func Unemployment(w Window) (UnemploymentResult, error) {
	const name = "Unemployment"
	if err := w.checkDims(name); err != nil {
		return UnemploymentResult{}, err
	}
	res := UnemploymentResult{ConfigHash: w.ConfigHash, From: w.From, To: w.To, Warnings: withWarnings(w), Total: UnemploymentRow{Region: -1, Role: -1}}
	var err error
	for _, r := range w.LabourRoles {
		if r.Region < 0 || int(r.Region) >= w.Regions || r.Role < 0 || int(r.Role) >= w.Roles {
			return UnemploymentResult{}, wrap(name, "labour_roles", ErrRange)
		}
		row := UnemploymentRow{Region: r.Region, Role: r.Role, Offered: r.Offered, Hired: r.Hired, NoVacancy: r.NoVacancy}
		if row.Rate, err = unemploymentRate(row); err != nil {
			return UnemploymentResult{}, err
		}
		res.Rows = append(res.Rows, row)
	}
	if err = res.sums(w); err != nil {
		return UnemploymentResult{}, err
	}
	res.ByReason, err = refusalRows(w)
	res.Warnings = clampWarning(res.Warnings, res.Total.Rate)
	return res, err
}

// unemploymentRate fills the rate of a row.
func unemploymentRate(r UnemploymentRow) (Index, error) {
	return ratioWide("Unemployment", "rate", measure.FromInt64(r.NoVacancy), measure.FromInt64(r.Offered))
}

// sums fills the region rows and the realm row from the LabourRound sums by region.
func (res *UnemploymentResult) sums(w Window) error {
	var off, hired, none acc
	for _, l := range w.Labour {
		if l.Region < 0 || int(l.Region) >= w.Regions {
			return wrap("Unemployment", "labour", ErrRange)
		}
		row := UnemploymentRow{Region: l.Region, Role: -1, Offered: l.Offered, Hired: l.Hired, NoVacancy: l.NoVacancy}
		var err error
		if row.Rate, err = unemploymentRate(row); err != nil {
			return err
		}
		res.Regions = append(res.Regions, row)
		off.addInt(measure.FromInt64(l.Offered))
		hired.addInt(measure.FromInt64(l.Hired))
		none.addInt(measure.FromInt64(l.NoVacancy))
	}
	var err error
	t := &res.Total
	if t.Offered, err = off.int64("Unemployment", "offered"); err != nil {
		return err
	}
	if t.Hired, err = hired.int64("Unemployment", "hired"); err != nil {
		return err
	}
	if t.NoVacancy, err = none.int64("Unemployment", "no_vacancy"); err != nil {
		return err
	}
	t.Rate, err = unemploymentRate(*t)
	return err
}

// refusalRows sums the formation refusals of the window over the regions and lists the
// reasons with a count above 0, in the order of the checks of spec-employment.md 5.5.
func refusalRows(w Window) ([]RefusalRow, error) {
	var sum [17]acc
	for _, l := range w.Labour {
		for i, v := range l.Refusals.counts() {
			sum[i].addInt(measure.FromInt64(v))
		}
	}
	var out []RefusalRow
	for i, reason := range refusalReasons() {
		n, err := sum[i].int64("Unemployment", string(reason))
		if err != nil {
			return nil, err
		}
		if n > 0 {
			out = append(out, RefusalRow{Reason: reason, Count: n})
		}
	}
	return out, nil
}

// refusalReasons returns the reasons of LabourRefusals.counts, in the order of its
// fields (the order of the checks of spec-employment.md 5.5).
func refusalReasons() [17]economysim.Reason {
	return [17]economysim.Reason{
		economysim.ReasonTooManyHires, economysim.ReasonNotPermitted, economysim.ReasonNoSite,
		economysim.ReasonZeroYield, economysim.ReasonNoTool, economysim.ReasonNoFacility,
		economysim.ReasonNoInput, economysim.ReasonNoEndowment, economysim.ReasonNoTitle,
		economysim.ReasonRentFunds, economysim.ReasonNotUnlocked, economysim.ReasonFacilityCap,
		economysim.ReasonNoTarget, economysim.ReasonStoreRefused, economysim.ReasonNothingToRepair,
		economysim.ReasonNoRoom, economysim.ReasonEmployerFunds,
	}
}

// counts returns the refusals in the order of refusalReasons.
func (r LabourRefusals) counts() [17]int64 {
	return [17]int64{
		r.TooMany, r.NotPermitted, r.NoSite, r.ZeroYield, r.NoTool, r.NoFacility, r.NoInput,
		r.NoEndowment, r.NoTitle, r.RentFunds, r.NotUnlocked, r.FacilityCap, r.NoTarget,
		r.StoreRefused, r.NothingToRepair, r.NoRoom, r.EmployerFunds,
	}
}

// WageBillRow is the wage bill of one region (Region -1: the realm): the wages released and
// the refunds of cut starts of the window, and the shares of the value added of the
// region in its accrual and cash forms (spec-indicators.md 7.2.1). A share is undefined
// (OK false) for a value added at or below 0; above 10 000 it is a result, not an error.
type WageBillRow struct {
	Region           economysim.RegionID
	Paid, Refunded   economysim.Money
	ValueAdded       int64
	Cash             int64
	Share, ShareCash Index
}

// WageBillResult is the wage bill by region and in all (spec-indicators.md 9.1 row 23).
type WageBillResult struct {
	ConfigHash [32]byte
	From, To   int64
	Regions    []WageBillRow
	Realm      WageBillRow
	Warnings   []Warning
}

// WageBill returns the money paid to hired households in a window, the sum of
// LabourRound.Paid by region of the hire, with the Refunded of cut starts beside it, and
// the labour share rdiv(10 000 * Paid, ValueAdded) against the accrual form of
// spec-indicators.md 7.2.1 and against the cash form (FR-IND-057). g is the GDP of the
// same window; the value added holds the wage and the employer's profit together, so the
// share is the wage bill over it and not a share of income. A hire is paid by the tick
// and its value is booked at the job start, so a window that cuts a hire shows the two
// on different edges.
func WageBill(w Window, g GDPResult) (WageBillResult, error) {
	const name = "WageBill"
	if err := w.checkDims(name); err != nil {
		return WageBillResult{}, err
	}
	switch {
	case g.ConfigHash != w.ConfigHash:
		return WageBillResult{}, wrap(name, "gdp", ErrConfigMismatch)
	case g.From != w.From || g.To != w.To:
		return WageBillResult{}, wrap(name, "gdp", ErrWindowOrder)
	}
	res := WageBillResult{ConfigHash: w.ConfigHash, From: w.From, To: w.To, Warnings: withWarnings(w), Realm: WageBillRow{Region: -1}}
	var paid, refunded acc
	for _, l := range w.Labour {
		row, err := wageRow(g, l)
		if err != nil {
			return WageBillResult{}, err
		}
		res.Regions = append(res.Regions, row)
		paid.addInt(measure.FromInt64(int64(l.Paid)))
		refunded.addInt(measure.FromInt64(int64(l.Refunded)))
	}
	err := res.finish(g, paid, refunded)
	res.Warnings = clampWarning(res.Warnings, res.Realm.Share, res.Realm.ShareCash)
	return res, err
}

// wageRow is the wage row of a region: the GDP row of the region gives the value added.
func wageRow(g GDPResult, l LabourTally) (WageBillRow, error) {
	row := WageBillRow{Region: l.Region, Paid: l.Paid, Refunded: l.Refunded}
	for _, r := range g.Regions {
		if r.Region == l.Region {
			row.ValueAdded, row.Cash = r.ValueAdded, r.Cash
			return row.shares()
		}
	}
	return WageBillRow{}, wrap("WageBill", "gdp", ErrRange)
}

// shares fills the two shares of a row.
func (r WageBillRow) shares() (WageBillRow, error) {
	var err error
	if r.Share, err = wageShare("share", r.Paid, r.ValueAdded); err != nil {
		return WageBillRow{}, err
	}
	r.ShareCash, err = wageShare("share_cash", r.Paid, r.Cash)
	return r, err
}

// wageShare is rdiv(10 000 * paid, base), undefined for a base at or below 0.
func wageShare(field string, paid economysim.Money, base int64) (Index, error) {
	if base <= 0 {
		return Index{}, nil
	}
	return ratioWide("WageBill", field, measure.FromInt64(int64(paid)), measure.FromInt64(base))
}

// finish fills the realm row from the sums and the realm row of the GDP.
func (res *WageBillResult) finish(g GDPResult, paid, refunded acc) error {
	p, err := paid.int64("WageBill", "paid")
	if err != nil {
		return err
	}
	f, err := refunded.int64("WageBill", "refunded")
	if err != nil {
		return err
	}
	res.Realm.Paid, res.Realm.Refunded = economysim.Money(p), economysim.Money(f)
	res.Realm.ValueAdded, res.Realm.Cash = g.Realm.ValueAdded, g.Realm.Cash
	res.Realm, err = res.Realm.shares()
	return err
}
