package indicator

import (
	"cmp"

	economysim "github.com/Whealinghold/economy-sim"
)

// This file folds LabourRound of the employment module into the tally (spec-indicators.md
// 9.1 rows 22 to 24, FR-IND-056 to FR-IND-058): the households offered, hired and left
// with no vacancy and the wages paid and refunded by region, the same three counts by
// worker role, and the positions each formation check closed.

// LabourTally is LabourRound summed over the window for one region. The three counts
// are households (the sum of weights), summed over ticks, so they are household-ticks.
type LabourTally struct {
	Region economysim.RegionID
	// Offered, Hired and NoVacancy are the households offered, hired and left
	// unmatched; NotStarted the households of a hire whose job started nothing.
	Offered, Hired, NoVacancy, NotStarted int64
	// HouseholdTicks is the working household-ticks paid; Paid the wages released
	// and Refunded the refunds of a cut start.
	HouseholdTicks int64
	Paid, Refunded economysim.Money
	// Refusals are the positions the formation checks closed.
	Refusals LabourRefusals
}

// LabourRefusals are the positions each formation check of the hire closed, summed
// over LabourRound (spec-employment.md 5.5). The fields of a module that is not built
// stay 0.
type LabourRefusals struct {
	TooMany, NotPermitted, NoSite, ZeroYield, NoTool, NoFacility, NoInput, NoEndowment    int64
	NoTitle, RentFunds, NotUnlocked, FacilityCap, NoTarget, StoreRefused, NothingToRepair int64
	NoRoom, EmployerFunds                                                                 int64
}

// LabourRoleTally is the households of one worker role in one region: offered, hired
// and left with no vacancy, summed over the rows of LabourRound.
type LabourRoleTally struct {
	Region                    economysim.RegionID
	Role                      economysim.RoleID
	Offered, Hired, NoVacancy int64
}

func labourKey(r LabourTally, k LabourTally) int { return cmp.Compare(r.Region, k.Region) }

func labourRoleKey(r LabourRoleTally, k LabourRoleTally) int {
	return cmp.Or(cmp.Compare(r.Region, k.Region), cmp.Compare(r.Role, k.Role))
}

// labourRoundOK reports whether a LabourRound names a region and worker roles of the
// window, carries no negative count and takes the step it belongs to; a bad one marks
// the window stray.
func (t *Tally) labourRoundOK(e economysim.LabourRound) bool {
	ok := e.Region >= 0 && int(e.Region) < t.w.Regions && t.step(e.Tick) && t.labourCountsOK(e)
	for _, r := range e.Workers {
		ok = ok && r.Role >= 0 && int(r.Role) < t.w.Roles && t.nonneg(r.Offered, r.Hired, r.NoVacancy)
	}
	if !ok {
		t.stray = true
	}
	return ok
}

// labourCountsOK reports whether no count or sum of money of a LabourRound is negative.
func (t *Tally) labourCountsOK(e economysim.LabourRound) bool {
	return t.nonneg(e.Offered, e.Hired, e.NoVacancy, e.NotStarted, e.HouseholdTicks, int64(e.Paid), int64(e.Refunded)) &&
		t.nonneg(e.TooMany, e.NotPermitted, e.NoSite, e.ZeroYield, e.NoTool, e.NoFacility, e.NoInput, e.NoEndowment) &&
		t.nonneg(e.NoTitle, e.RentFunds, e.NotUnlocked, e.FacilityCap, e.NoTarget, e.StoreRefused, e.NothingToRepair) &&
		t.nonneg(e.NoRoom, e.EmployerFunds)
}

// labourRound folds a LabourRound: the region's sums and the rows of its worker roles.
func (t *Tally) labourRound(e economysim.LabourRound) {
	if !t.labourRoundOK(e) {
		return
	}
	l := upsert(&t.w.Labour, LabourTally{Region: e.Region}, labourKey)
	t.add(&l.Offered, e.Offered)
	t.add(&l.Hired, e.Hired)
	t.add(&l.NoVacancy, e.NoVacancy)
	t.add(&l.NotStarted, e.NotStarted)
	t.add(&l.HouseholdTicks, e.HouseholdTicks)
	t.add((*int64)(&l.Paid), int64(e.Paid))
	t.add((*int64)(&l.Refunded), int64(e.Refunded))
	t.refusals(&l.Refusals, e)
	for _, r := range e.Workers {
		w := upsert(&t.w.LabourRoles, LabourRoleTally{Region: e.Region, Role: r.Role}, labourRoleKey)
		t.add(&w.Offered, r.Offered)
		t.add(&w.Hired, r.Hired)
		t.add(&w.NoVacancy, r.NoVacancy)
	}
}

// refusals adds the formation refusals of a LabourRound to the row.
func (t *Tally) refusals(r *LabourRefusals, e economysim.LabourRound) {
	for _, f := range [...]struct {
		dst *int64
		v   int64
	}{
		{&r.TooMany, e.TooMany}, {&r.NotPermitted, e.NotPermitted}, {&r.NoSite, e.NoSite},
		{&r.ZeroYield, e.ZeroYield}, {&r.NoTool, e.NoTool}, {&r.NoFacility, e.NoFacility},
		{&r.NoInput, e.NoInput}, {&r.NoEndowment, e.NoEndowment}, {&r.NoTitle, e.NoTitle},
		{&r.RentFunds, e.RentFunds}, {&r.NotUnlocked, e.NotUnlocked}, {&r.FacilityCap, e.FacilityCap},
		{&r.NoTarget, e.NoTarget}, {&r.StoreRefused, e.StoreRefused}, {&r.NothingToRepair, e.NothingToRepair},
		{&r.NoRoom, e.NoRoom}, {&r.EmployerFunds, e.EmployerFunds},
	} {
		t.add(f.dst, f.v)
	}
}
