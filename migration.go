package economysim

import (
	"slices"

	"github.com/Whealinghold/economy-sim/internal/checked"
)

// This file is the migration module (spec-migration.md, MIG): households that
// move between regions on a timed journey. It holds the module's validation and,
// below it, its state, the guard, the arrivals and the life of a journey
// (migration_pass.go is the pass of P7.2, migration_report.go the warnings, the
// events and the checkpoint). The arithmetic is internal/migration; this file
// converts at the call (architecture.md 2.9, 2.10.1).

// migrationCommodities returns K(role) of spec-migration.md 6.1 as commodity IDs
// in ascending order: the upkeep commodities of the role in any region (its own
// list or the world's, and every region's extra upkeep) and, for every recipe on
// its list, the inputs, the required tools and the non-discard outputs. It reads
// a configuration that may not validate, so an unknown recipe adds nothing.
func migrationCommodities(c *Config, lk lookup, r *Role) []string {
	set := map[string]bool{}
	for _, it := range effectiveUpkeep(c, r) {
		set[it.Commodity] = true
	}
	for i := range c.Regions {
		for _, it := range c.Regions[i].ExtraUpkeep {
			set[it.Commodity] = true
		}
	}
	for _, id := range r.Recipes {
		if rc := lk.recipe[id]; rc != nil {
			addRecipeCommodities(set, rc)
		}
	}
	out := make([]string, 0, len(set))
	for id := range set { // the sort below erases the map order
		out = append(out, id)
	}
	slices.Sort(out)
	return out
}

// addRecipeCommodities adds the inputs, the required tools and the non-discard
// outputs of a recipe to set.
func addRecipeCommodities(set map[string]bool, rc *Recipe) {
	for _, it := range rc.Inputs {
		set[it.Commodity] = true
	}
	for _, t := range rc.Requires {
		set[t.Commodity] = true
	}
	for _, o := range rc.Outputs {
		if !o.Discard {
			set[o.Commodity] = true
		}
	}
}

// mobile reports whether the module can move the households of the role: it is
// not Fixed and has a recipe (spec-migration.md 3.3, 3.3.2).
func (r *Role) mobile() bool { return !r.Fixed && len(r.Recipes) > 0 }

// validateMigration is the module's part of phases 2 and 3 of spec-config.md
// 7.1.6 (spec-migration.md 12.1, 12.2): the bounds of each field, the module
// dependency, the region rows and the derived checks. It reports nothing while
// the module is off; Role.Fixed has no error.
func (v *validator) validateMigration() {
	c := &v.c
	m := c.Modules.Migration
	if m == nil {
		return
	}
	base := []string{"modules", "migration"}
	if c.Modules.Information == nil {
		v.rep.add(ErrModuleDependency, base, "information", "")
	}
	fields := []struct {
		key         string
		val, lo, hi int64
	}{
		{"move_speed", m.MoveSpeed, 1, MaxSpeed},
		{"move_review_ticks", m.MoveReviewTicks, 1, MaxMoveReviewTicks},
		{"move_below", m.MoveBelow, 0, MaxBasisPoints},
		{"move_gain", m.MoveGain, 0, MaxBasisPoints},
		{"move_width", m.MoveWidth, 0, MaxBasisPoints},
		{"move_cooldown", m.MoveCooldown, 0, MaxWelfareWindow},
		{"move_horizon", m.MoveHorizon, 2, MaxTravelTicks},
		{"move_out_share", m.MoveOutShare, 0, wholeBP},
		{"move_in_share", m.MoveInShare, 0, wholeBP},
	}
	ok := true
	for _, f := range fields {
		ok = v.inRange(path(base, f.key), f.val, f.lo, f.hi) && ok
	}
	if len(m.Regions) > MaxRegions {
		v.rep.add(ErrTooMany, path(base, "regions"), itoa(len(m.Regions)), itoa(MaxRegions))
		return // the rest is linear in the count it just refused
	}
	rowsOK := v.migrationRows(base, m)
	if ok && rowsOK {
		v.migrationLimits(m)
	}
}

// migrationRows checks the region rows of 11.2: a defined region, once, and the
// two shares in 0 to 10 000. It reports whether every row is sound.
func (v *validator) migrationRows(base []string, m *Migration) bool {
	ok := true
	seen := make(map[string]bool, len(m.Regions))
	for i := range m.Regions {
		r := &m.Regions[i]
		rb := path(base, "region", r.Region)
		if v.lk.region[r.Region] == nil {
			v.rep.add(ErrUnknownRegion, path(rb, "region"), r.Region, "")
			ok = false
		}
		if seen[r.Region] {
			v.rep.add(ErrDuplicateID, path(rb, "region"), r.Region, "")
			ok = false
		}
		seen[r.Region] = true
		ok = v.inRange(path(rb, "out_share"), r.OutShare, 0, wholeBP) && ok
		ok = v.inRange(path(rb, "in_share"), r.InShare, 0, wholeBP) && ok
	}
	return ok
}

// migrationLimits is the derived checks of spec-migration.md 12.2, run only when
// the fields hold: the candidates of each mobile role against the bound of one
// view (ErrTooMany), and the households of the largest region times the largest
// share against MaxQuantity (ErrOverflow). The estimate's own products are
// covered by the unit-cost check of spec-production.md 10.3.
func (v *validator) migrationLimits(m *Migration) {
	c := &v.c
	others := int64(max(0, len(c.Regions)-1))
	for i := range c.Roles {
		r := &c.Roles[i]
		if !r.mobile() {
			continue
		}
		if n := int64(len(migrationCommodities(c, v.lk, r))) * others; n > MaxRumoursPerView {
			v.rep.add(ErrTooMany, path([]string{"roles", r.ID}, "recipes"), i64(n), i64(MaxRumoursPerView))
		}
	}
	share := max(m.MoveOutShare, m.MoveInShare)
	for _, r := range m.Regions {
		share = max(share, r.OutShare, r.InShare)
	}
	byRegion := make(map[string]int64, len(c.Regions))
	most := int64(0)
	for _, g := range c.Groups {
		sum, ok := checked.Add(byRegion[g.Region], g.Households)
		if !ok {
			sum = MaxQuantity + 1
		}
		byRegion[g.Region] = sum
		most = max(most, sum)
	}
	p, ok := checked.Mul(most, share)
	if !ok {
		p = MaxQuantity + 1
	}
	if p > MaxQuantity {
		v.rep.add(ErrOverflow, []string{"modules", "migration", "move_out_share"}, i64(p), i64(MaxQuantity))
	}
}
