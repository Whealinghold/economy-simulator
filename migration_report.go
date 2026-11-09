package economysim

import (
	"github.com/Whealinghold/economy-sim/internal/checked"
	mig "github.com/Whealinghold/economy-sim/internal/migration"
)

// This file is the reporting side of the migration module (spec-migration.md 12.3,
// 13, 14): the warnings of the configuration, and below them the events, logs,
// snapshot rows and the checkpoint encoder (MIG, a continuation of migration.go
// for the Lizard limits, architecture.md 2.6).

// migrationFlowFactor is the factor 3 of the warning migration_flow_window
// (spec-migration.md 11.3, 12.3.1): a rule constant of the model of 16.6, not a
// parameter.
const migrationFlowFactor = 3

// migrationWarnings returns the five warnings of spec-migration.md 12.3 that
// read the configuration alone; migration_provision and migration_far need the
// shortest distances and are migrationRouteWarnings. They read a configuration
// that may not validate, so each guards its inputs. Each carries rule and path
// only (the product of migration_flow_window has no carrier, N-327).
func migrationWarnings(c *Config, lk lookup) []Warning {
	m := c.Modules.Migration
	if m == nil {
		return nil
	}
	base := []string{"modules", "migration"}
	var out []Warning
	if len(c.Regions) < 2 || len(c.Edges) == 0 {
		out = append(out, Warning{Path: base, Rule: "migration_without_routes"})
	}
	roles, longest := migrationRoleWarnings(c, lk)
	out = append(out, roles...)
	if p, ok := checked.Mul(m.MoveOutShare, longest); ok && p > migrationFlowFactor*m.MoveGain {
		out = append(out, Warning{Path: []string{"modules", "migration", "move_out_share"}, Rule: "migration_flow_window"})
	}
	if m.MoveReviewTicks >= 1 && m.MoveOutShare >= wholeBP/m.MoveReviewTicks && m.MoveInShare >= wholeBP {
		out = append(out, Warning{Path: base, Rule: "migration_uncapped"})
	}
	return out
}

// migrationRoleWarnings returns the warnings that read the mobile roles,
// migration_no_mobile_role and migration_cohort, and the longest effective welfare
// window of a mobile role (0 when none), which migration_flow_window reads.
func migrationRoleWarnings(c *Config, lk lookup) (out []Warning, longest int64) {
	any := false
	for i := range c.Roles {
		r := &c.Roles[i]
		if !r.mobile() {
			continue
		}
		any = true
		if w, ok := effectiveWindow(c, lk, r); ok {
			longest = max(longest, w)
		}
		if migrationCohort(c, r) {
			out = append(out, Warning{Path: []string{"roles", r.ID}, Rule: "migration_cohort"})
		}
	}
	if !any {
		out = append(out, Warning{Path: []string{"modules", "migration"}, Rule: "migration_no_mobile_role"})
	}
	return out, longest
}

// migrationCohort reports whether some group of the role has a scale above 1: its
// agents are cohorts, which move whole (spec-migration.md 3.2.2). A group with no
// scale of its own has the world's.
func migrationCohort(c *Config, r *Role) bool {
	for _, g := range c.Groups {
		scale := g.Scale
		if scale == 0 {
			scale = c.World.Scale
		}
		if g.Role == r.ID && scale > 1 {
			return true
		}
	}
	return false
}

// b2i is 1 for true and 0 for false.
func b2i(b bool) int64 {
	if b {
		return 1
	}
	return 0
}

// migrationRouteWarnings returns migration_provision and migration_far
// (spec-migration.md 12.3): the ordered pairs of regions with a route in the
// configured graph, no edge status applied, whose journey time is above CoverTicks
// and is the horizon or more. Count is the number of pairs. Warnings appends them
// after the structural phases come back clean, as it appends the horizon warning;
// the module being off, a field out of its bound or a graph that does not build
// gives nothing.
func migrationRouteWarnings(c *Config) []Warning {
	m := c.Modules.Migration
	if m == nil || m.MoveSpeed < 1 || m.MoveHorizon < 2 {
		return nil
	}
	g, err := NewGraph(*c)
	if err != nil {
		return nil
	}
	provision, far := journeyPairs(c, g, m)
	var out []Warning
	if provision > 0 {
		out = append(out, Warning{Path: []string{"modules", "migration", "move_speed"}, Rule: "migration_provision", Count: provision})
	}
	if far > 0 {
		out = append(out, Warning{Path: []string{"modules", "migration", "move_horizon"}, Rule: "migration_far", Count: far})
	}
	return out
}

// journeyPairs counts the ordered pairs of regions with a route whose journey time
// is above CoverTicks and the pairs whose journey time is the horizon or more.
func journeyPairs(c *Config, g Graph, m *Migration) (provision, far int64) {
	for from := range RegionID(g.Len()) {
		for to := range RegionID(g.Len()) {
			d, ok := g.Distance(from, to)
			if !ok || from == to {
				continue
			}
			if k, ok := mig.Ticks(d, m.MoveSpeed); ok {
				provision += b2i(k > c.World.CoverTicks)
				far += b2i(k >= m.MoveHorizon)
			}
		}
	}
	return provision, far
}
