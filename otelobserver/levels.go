package otelobserver

import economysim "github.com/Whealinghold/economy-sim"

// unfed keeps the unfed households of the last UpkeepShortfall of each
// (region, role), and whether the step has seen one, so that the level of
// economysim.household.unfed is right in a tick whose households were all fed:
// UpkeepShortfall is not emitted then (spec-observability.md 6.3.1).
type unfed struct {
	roles int
	last  []int64
	seen  []bool
}

func newUnfed(regions, roles int) unfed {
	return unfed{roles: roles, last: make([]int64, regions*roles), seen: make([]bool, regions*roles)}
}

// cell is the place of a (region, role), or -1 outside the table.
func (u *unfed) cell(region, role int32) int {
	if region < 0 || role < 0 || int(role) >= u.roles || int(region)*u.roles+int(role) >= len(u.last) {
		return -1
	}
	return int(region)*u.roles + int(role)
}

// upkeepShortfall adds the change of the unfed households of a (region, role) to
// the level of the role, stores it and marks the cell seen, and counts the units
// short by commodity.
func (o *Observer) upkeepShortfall(ev economysim.UpkeepShortfall) {
	role := ix(ev.Role)
	for _, row := range ev.Short {
		o.count(m2(iUpkeepShort, ix(row.Commodity), role), row.Units)
	}
	if i := o.unfed.cell(ix(ev.Region), role); i >= 0 {
		o.count(m1(iHouseholdUnfed, role), ev.Unfed-o.unfed.last[i])
		o.unfed.last[i], o.unfed.seen[i] = ev.Unfed, true
	}
}

// endUnfed runs at a TickEnded that did not fail: every cell with a last value
// that the step did not see goes back to 0, and the marks are cleared. A failed
// step makes no correction (spec-observability.md 6.3.1).
func (o *Observer) endUnfed() {
	u := &o.unfed
	for i, last := range u.last {
		if !u.seen[i] && last != 0 {
			o.count(m1(iHouseholdUnfed, int32(i%u.roles)), -last)
			u.last[i] = 0
		}
		u.seen[i] = false
	}
}

// seed sets the levels from the snapshot at the restored boundary. A counter is
// not seeded: it counts what this adapter saw (spec-observability.md 6.4.2). The
// series were emptied by reset, so the SDK is brought to the snapshot's level by
// the difference.
func (o *Observer) seed(s economysim.Snapshot) {
	for _, r := range s.Roles {
		role := ix(r.Role)
		o.count(m1(iAgentCount, role), r.Agents)
		o.count(m1(iHouseholdCount, role), r.Households)
		o.count(m1(iHouseholdUnfed, role), r.Unfed)
		if i := o.unfed.cell(ix(r.Region), role); i >= 0 {
			o.unfed.last[i] = r.Unfed
		}
	}
	for _, m := range s.Markets {
		o.set(m2(iMarketClearingPrice, ix(m.State.Commodity), ix(m.State.Region)), int64(m.State.Price))
	}
}
