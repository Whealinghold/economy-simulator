package indicator

import (
	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/internal/measure"
)

// EmploymentRow is the hired households of one (region, role) at the closing
// boundary. Employed (a hireable role) is the households of the role on a hired job
// and Employing (an employer role) the hired households working for agents of the
// role; Households is the role's households. CloseRate is rdiv(10 000 * Employed,
// Households) for a hireable role, OK false where it has no households.
type EmploymentRow struct {
	Region                          economysim.RegionID
	Role                            economysim.RoleID
	Hireable                        bool
	Employed, Employing, Households int64
	CloseRate                       Index
}

// EmploymentRegion is the window form for one region: the working household-ticks
// paid (LabourRound.HouseholdTicks) over the running households of the hireable roles,
// the sum over the ticks of their count at each tick's start (spec-indicators.md 6.3).
// Rate is undefined (OK false) where the region has no running hireable household.
type EmploymentRegion struct {
	Region                  economysim.RegionID
	HouseholdTicks, Running int64
	Rate                    Index
}

// EmploymentResult is the employment of the hireable roles at the closing boundary and
// over the window (spec-indicators.md 9.1 row 24).
type EmploymentResult struct {
	ConfigHash [32]byte
	From, To   int64
	// Rows are the hireable and the employer roles, in (region, role) order; a role of
	// neither kind is absent.
	Rows []EmploymentRow
	// Regions are the window rates by region, in region order, for the regions with a
	// hireable role.
	Regions []EmploymentRegion
	// Total is the window form over all regions (Region -1).
	Total    EmploymentRegion
	Warnings []Warning
}

// Employment returns the households of the hireable roles that were on a hired job: at
// the closing boundary rdiv(10 000 * Employed, Households) of the role from
// RoleRow.Employed, and over the window the paid household-ticks over the running
// households at the start of each tick, by region (FR-IND-058). The count of paid
// household-ticks is a region figure in LabourRound, so the window rate has no role
// split. Households, not agents: a household that works its own recipe is not employed
// in this sense, and employment is not "has income". A role that is neither hireable nor
// an employer is absent, not 0. cfg is the world's configuration (its hash is the
// window's) and closing the snapshot that closes the window.
func Employment(w Window, cfg economysim.Config, closing economysim.Snapshot) (EmploymentResult, error) {
	const name = "Employment"
	if err := w.checkClosing(name, closing); err != nil {
		return EmploymentResult{}, err
	}
	if len(cfg.Roles) != w.Roles || economysim.HashConfig(cfg) != w.ConfigHash {
		return EmploymentResult{}, wrap(name, "config", ErrConfigMismatch)
	}
	if err := w.checkPeople(name); err != nil {
		return EmploymentResult{}, err
	}
	res := EmploymentResult{ConfigHash: w.ConfigHash, From: w.From, To: w.To, Warnings: withWarnings(w), Total: EmploymentRegion{Region: -1}}
	var err error
	if res.Rows, err = employmentRows(w, cfg, closing); err != nil {
		return EmploymentResult{}, err
	}
	if err = res.window(w, cfg); err != nil {
		return EmploymentResult{}, err
	}
	return res, nil
}

// employmentRows reads the role rows of the closing snapshot of the hireable and the
// employer roles.
func employmentRows(w Window, cfg economysim.Config, closing economysim.Snapshot) ([]EmploymentRow, error) {
	var out []EmploymentRow
	for _, r := range closing.Roles {
		if r.Region < 0 || int(r.Region) >= w.Regions || r.Role < 0 || int(r.Role) >= w.Roles {
			return nil, wrap("Employment", "role row", ErrRange)
		}
		role := cfg.Roles[r.Role]
		if !role.Hireable && role.Employer == nil {
			continue
		}
		row := EmploymentRow{Region: r.Region, Role: r.Role, Hireable: role.Hireable, Employed: r.Employed, Employing: r.Employing, Households: r.Households}
		if role.Hireable {
			var err error
			if row.CloseRate, err = ratioWide("Employment", "close_rate", measure.FromInt64(r.Employed), measure.FromInt64(r.Households)); err != nil {
				return nil, err
			}
		}
		out = append(out, row)
	}
	return out, nil
}

// window fills the window form by region and in all.
func (res *EmploymentResult) window(w Window, cfg economysim.Config) error {
	running := make([]acc, w.Regions)
	has := make([]bool, w.Regions)
	for i, p := range w.People {
		r, role := i/w.Roles, i%w.Roles
		if cfg.Roles[role].Hireable {
			running[r].addInt(measure.FromInt64(p.HouseholdTicks))
			has[r] = true
		}
	}
	ticks := make([]int64, w.Regions)
	for _, l := range w.Labour {
		if l.Region < 0 || int(l.Region) >= w.Regions {
			return wrap("Employment", "labour", ErrRange)
		}
		ticks[l.Region] = l.HouseholdTicks
	}
	var allTicks, allRun acc
	for r := range w.Regions {
		if !has[r] {
			continue
		}
		row, err := employmentRegion(economysim.RegionID(r), ticks[r], running[r])
		if err != nil {
			return err
		}
		res.Regions = append(res.Regions, row)
		allTicks.addInt(measure.FromInt64(row.HouseholdTicks))
		allRun.merge(running[r])
	}
	return res.total(allTicks, allRun)
}

// employmentRegion is the window row of a region.
func employmentRegion(r economysim.RegionID, ticks int64, running acc) (EmploymentRegion, error) {
	run, err := running.int64("Employment", "running")
	if err != nil {
		return EmploymentRegion{}, err
	}
	row := EmploymentRegion{Region: r, HouseholdTicks: ticks, Running: run}
	row.Rate, err = ratioWide("Employment", "rate", measure.FromInt64(ticks), running.v)
	return row, err
}

// total fills the realm row of the window form.
func (res *EmploymentResult) total(ticks, running acc) error {
	var err error
	if res.Total.HouseholdTicks, err = ticks.int64("Employment", "household_ticks"); err != nil {
		return err
	}
	if res.Total.Running, err = running.int64("Employment", "running"); err != nil {
		return err
	}
	res.Total.Rate, err = ratioWide("Employment", "rate", ticks.v, running.v)
	res.Warnings = clampWarning(res.Warnings, res.Total.Rate)
	return err
}
