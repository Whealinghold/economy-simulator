package economysim

import "github.com/Whealinghold/economy-sim/internal/occupation"

// This file holds what the world reads to report on the spawner: the value of
// EntryLimited, the runs of the three bounded Warns and the largest privation of
// a role. It emits nothing; the world emits (spec-spawner.md 13).

// tallyRow is the counts of one region in one tick: the executor's, and the
// plan's that the default spawner leaves in Decisions.
type tallyRow struct {
	changed, entered, removed, unfunded, refused int64
	noViableRole, regionCap, roleCap             int64
	cooldown, hysteresis, exitCap                int64
	stranded                                     int64
}

// entryTally adds, per region, the counts of the executor to those of the plan
// (D119). It is sized once by the number of regions and holds no map.
type entryTally struct {
	rows []tallyRow
}

// newEntryTally returns a tally for regions regions, all zero.
func newEntryTally(regions int) *entryTally {
	return &entryTally{rows: make([]tallyRow, regions)}
}

// reset zeroes every region's counts, keeping the backing array.
func (t *entryTally) reset() { clear(t.rows) }

// row returns the counts of region r, or nil for a region outside the tally.
func (t *entryTally) row(r RegionID) *tallyRow {
	if int(r) < 0 || int(r) >= len(t.rows) {
		return nil
	}
	return &t.rows[r]
}

// addPlan adds the counts a spawner left in d for region r.
func (t *entryTally) addPlan(r RegionID, d *Decisions) {
	if w := t.row(r); w != nil {
		w.noViableRole = satAdd(w.noViableRole, d.noViableRole)
		w.regionCap = satAdd(w.regionCap, d.regionCap)
		w.roleCap = satAdd(w.roleCap, d.roleCap)
		w.cooldown = satAdd(w.cooldown, d.cooldown)
		w.hysteresis = satAdd(w.hysteresis, d.hysteresis)
		w.exitCap = satAdd(w.exitCap, d.exitCap)
		w.stranded = satAdd(w.stranded, d.stranded)
	}
}

// addExecuted adds the households the executor changed, entered, removed, found
// unfunded and refused in region r.
func (t *entryTally) addExecuted(r RegionID, changed, entered, removed, unfunded, refused int64) {
	if w := t.row(r); w != nil {
		w.changed = satAdd(w.changed, changed)
		w.entered = satAdd(w.entered, entered)
		w.removed = satAdd(w.removed, removed)
		w.unfunded = satAdd(w.unfunded, unfunded)
		w.refused = satAdd(w.refused, refused)
	}
}

// limited returns the EntryLimited of region r at tick, and false when every
// count is 0 (FR-SPN-035).
func (t *entryTally) limited(tick int64, r RegionID) (EntryLimited, bool) {
	w := t.row(r)
	if w == nil || *w == (tallyRow{}) {
		return EntryLimited{}, false
	}
	return EntryLimited{
		Tick: tick, Region: r, Changed: w.changed, Entered: w.entered, Removed: w.removed,
		Unfunded: w.unfunded, NoViableRole: w.noViableRole, RegionCap: w.regionCap, RoleCap: w.roleCap,
		Cooldown: w.cooldown, Hysteresis: w.hysteresis, ExitCap: w.exitCap, Refused: w.refused, Stranded: w.stranded,
	}, true
}

// extinctRun advances the run of the Warn "role extinct" of a (region, role),
// pattern A: the first tick and every tenth (FR-SPN-042).
func extinctRun(before, agents int64, everHad, configured bool) (run int64, warn bool) {
	return advanceRun(before, occupation.Extinct(agents, everHad, configured), true)
}

// unfundedRun advances the run of the Warn "entry unfunded" of a region,
// pattern B: ten consecutive ticks, then every tenth (FR-SPN-036).
func unfundedRun(before, due, funded int64) (run int64, warn bool) {
	return advanceRun(before, occupation.Unfunded(due, funded), false)
}

// belowFloorRun advances the run of the Warn "group below floor" of a (region,
// role), pattern A (FR-SPN-047).
func belowFloorRun(before, weight, agents, minAgents int64) (run int64, warn bool) {
	return advanceRun(before, occupation.BelowFloor(weight, agents, minAgents), true)
}

// maxPrivation is the largest privation among the agents of role (FR-SPN-046),
// 0 if the role has none.
func maxPrivation(agents []RegionAgent, role RoleID) (m int64) {
	for i := range agents {
		if agents[i].Role == role && agents[i].Privation > m {
			m = agents[i].Privation
		}
	}
	return m
}
