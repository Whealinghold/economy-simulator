package economysim

import (
	"math"
	"slices"

	"github.com/Whealinghold/economy-sim/internal/checked"
	"github.com/Whealinghold/economy-sim/internal/household"
	"github.com/Whealinghold/economy-sim/internal/production"
	sav "github.com/Whealinghold/economy-sim/internal/saving"
)

// This file is the validation of the saving module (spec-saving-and-investment.md 11;
// the continuation file of saving.go, architecture.md 2.6): the rules of 11.1 that
// Validate checks and the eight warnings of 11.2.

// savingModulePath is the path of the module in a configuration.
var savingModulePath = []string{"modules", "saving"}

// savingBlockPath is the path of the Saver block of a role.
func savingBlockPath(r *Role) []string { return []string{"roles", r.ID, "saver"} }

// validateSaving checks the saving module and the Saver blocks (FR-SAV-034, FR-CFG-096).
// The rules run in phases and a phase that finds an error ends the run, so the first
// error is the same for the same mistake: the bounds, the module dependencies, then the
// class names. With the module off only a block is reported.
func (v *validator) validateSaving() {
	steps := []func() bool{v.savingBounds, v.savingDependencies, v.savingWorth}
	for _, step := range steps {
		if !step() {
			return
		}
	}
}

// savingBounds checks the three fields of the module and every field of every block
// against its bound of spec-saving-and-investment.md 10.2, and the rows of a Worth.
func (v *validator) savingBounds() bool {
	n := len(v.rep.errs)
	if m := v.c.Modules.Saving; m != nil {
		v.ranges(savingModulePath, []fieldRange{
			{"safety_ticks", m.SafetyTicks, 0, MaxSafetyTicks},
			{"hurdle_spread_bp", m.HurdleSpreadBP, 0, MaxHurdleSpreadBP},
			{"commit_band_bp", m.CommitBandBP, 0, MaxCommitBandBP},
		})
	}
	for i := range v.c.Roles {
		ro := &v.c.Roles[i]
		if b := ro.Saver; b != nil {
			p := savingBlockPath(ro)
			v.ranges(p, []fieldRange{
				{"precaution_share_bp", b.PrecautionShareBP, 0, MaxPrecautionShareBP},
				{"precaution_cap_ticks", b.PrecautionCapTicks, 0, MaxPrecautionCapTicks},
				{"hurdle_bp", b.HurdleBP, wholeBP, MaxHurdleBP},
				{"payback_ticks", b.PaybackTicks, 1, MaxPaybackTicks},
				{"repair_below_bp", b.RepairBelowBP, 0, wholeBP},
			})
			v.savingRowBounds(p, b)
		}
	}
	return len(v.rep.errs) == n
}

// savingRowBounds checks the count of the Worth rows and the numbers of each.
func (v *validator) savingRowBounds(p []string, b *Saver) {
	if len(b.Worth) > MaxWorthRows {
		v.rep.add(ErrOutOfBounds, path(p, "worth"), itoa(len(b.Worth)), itoa(MaxWorthRows))
		return
	}
	for i, w := range b.Worth {
		row := path(p, "worth["+itoa(i)+"]")
		v.inRange(path(row, "per_tick"), w.PerTick, 1, MaxWorthPerTick)
		v.inRange(path(row, "count"), w.Count, 1, MaxWorthCount)
	}
}

// savingDependencies reports ErrModuleDependency for a Saver block while Modules.Saving
// is nil, and for a Worth row while Modules.Infrastructure is nil, and whether every
// block is right.
func (v *validator) savingDependencies() bool {
	n := len(v.rep.errs)
	for i := range v.c.Roles {
		ro := &v.c.Roles[i]
		b := ro.Saver
		if b == nil {
			continue
		}
		if v.c.Modules.Saving == nil {
			v.rep.add(ErrModuleDependency, savingBlockPath(ro), "saver block", "modules.saving")
		}
		if len(b.Worth) > 0 && v.c.Modules.Infrastructure == nil {
			v.rep.add(ErrModuleDependency, path(savingBlockPath(ro), "worth"), "worth row", "modules.infrastructure")
		}
	}
	return len(v.rep.errs) == n
}

// savingWorth reports ErrUnknownFacilityClass for a Worth row that names a class not
// defined and ErrDuplicateID for a class twice in one role's Worth.
func (v *validator) savingWorth() bool {
	n := len(v.rep.errs)
	inf := v.c.Modules.Infrastructure
	if inf == nil {
		return true
	}
	known := make(map[string]bool, len(inf.Classes)) // a membership test only
	for _, cl := range inf.Classes {
		known[cl.ID] = true
	}
	for i := range v.c.Roles {
		ro := &v.c.Roles[i]
		if ro.Saver == nil {
			continue
		}
		seen := make(map[string]bool, len(ro.Saver.Worth)) // a membership test only
		for _, w := range ro.Saver.Worth {
			p := path(savingBlockPath(ro), "worth", w.Class)
			switch {
			case !known[w.Class]:
				v.rep.add(ErrUnknownFacilityClass, path(p, "class"), w.Class, "")
			case seen[w.Class]:
				v.rep.add(ErrDuplicateID, path(p, "class"), w.Class, "")
			}
			seen[w.Class] = true
		}
	}
	return len(v.rep.errs) == n
}

// savingWarnings returns the warnings of spec-saving-and-investment.md 11.2, unsorted
// (the caller sorts), and none with the module off. It reads the configuration as given
// and never fails validation (FR-SAV-035).
func savingWarnings(c *Config, lk lookup) []Warning {
	m := c.Modules.Saving
	if m == nil {
		return nil
	}
	var out []Warning
	for i := range c.Roles {
		if ro := &c.Roles[i]; ro.Saver != nil {
			out = append(out, savingRoleWarnings(c, lk, m, ro)...)
		}
	}
	if savingTargetExceedsMoney(c, lk, m) {
		out = append(out, Warning{Path: savingModulePath, Rule: "saving_target_exceeds_money"})
	}
	return out
}

// savingRoleWarnings are the warnings about one role with a block: the shape of its
// projects, its Worth rows, its repair gate and the safety against the credit slack.
func savingRoleWarnings(c *Config, lk lookup, m *Saving, ro *Role) []Warning {
	p := savingBlockPath(ro)
	var out []Warning
	add := func(rule string, p []string) { out = append(out, Warning{Path: p, Rule: rule}) }
	builds, repairs := savingClauses(lk, ro)
	if len(builds)+len(repairs) == 0 {
		add("saver_no_project", p)
	}
	if len(repairs) > 0 && ro.Saver.RepairBelowBP == 0 {
		add("repair_never", p)
	}
	if bw := ro.Credit; bw != nil && bw.Borrow != nil && m.SafetyTicks < bw.Borrow.SlackTicks {
		add("safety_below_slack", p)
	}
	out = append(out, savingWorthWarnings(c, ro, p, builds, repairs)...)
	return append(out, savingProjectWarnings(c, lk, m, ro, p)...)
}

// savingWorthWarnings are worth_unused and worth_on_site, one per Worth row.
func savingWorthWarnings(c *Config, ro *Role, p, builds, repairs []string) []Warning {
	var out []Warning
	for _, w := range ro.Saver.Worth {
		wp := path(p, "worth", w.Class)
		if !slices.Contains(builds, w.Class) && !slices.Contains(repairs, w.Class) {
			out = append(out, Warning{Path: wp, Rule: "worth_unused"})
		}
		if inf := c.Modules.Infrastructure; inf != nil && facilityKindOf(inf, w.Class) == kindSite {
			out = append(out, Warning{Path: wp, Rule: "worth_on_site"})
		}
	}
	return out
}

// savingClauses are the classes the recipes on the role's list build and repair.
func savingClauses(lk lookup, ro *Role) (builds, repairs []string) {
	for _, id := range ro.Recipes {
		rc := lk.recipe[id]
		switch {
		case rc == nil:
		case rc.Builds != nil:
			builds = append(builds, rc.Builds.Class)
		case rc.Repairs != nil:
			repairs = append(repairs, rc.Repairs.Class)
		}
	}
	return builds, repairs
}

// facilityKindOf is the kind of the class, or "" for a class not defined.
func facilityKindOf(m *Infrastructure, class string) string {
	for _, cl := range m.Classes {
		if cl.ID == class {
			return cl.Kind
		}
	}
	return ""
}

// savingProject is one build project of a role valued at the reference prices of a
// region (spec-saving-and-investment.md 5.3 to 5.6, for a weight of 1 that holds no
// facility and the yield of the recipe as written).
type savingProject struct {
	benefit, cost, outlay int64
}

// savingProjectWarnings are project_below_hurdle and spread_below_margin: the test of
// 5.5 at the reference prices on the role's best build project. A role with no build
// project that can be valued has neither.
func savingProjectWarnings(c *Config, lk lookup, m *Saving, ro *Role, p []string) []Warning {
	best, ok := bestSavingProject(c, lk, m, ro)
	if !ok {
		return nil
	}
	b := ro.Saver
	if !sav.Passes(best.benefit, best.cost, b.HurdleBP, MaxMoney) {
		return []Warning{{Path: p, Rule: "project_below_hurdle"}}
	}
	ratio, ok := checked.MulDiv(best.benefit, wholeBP, max(best.cost, 1))
	margin, mok := checked.Sub(ratio, b.HurdleBP)
	if !mok {
		margin = math.MaxInt64 // a hurdle below the bounds, unvalidated: the margin saturates
	}
	if ok && m.HurdleSpreadBP < margin {
		return []Warning{{Path: p, Rule: "spread_below_margin"}}
	}
	return nil
}

// bestSavingProject is the build project of the role with the highest benefit over cost
// at the reference prices of the regions the role has a group in, and whether there is
// one. A project whose benefit is 0 is still a project: it fails the test.
func bestSavingProject(c *Config, lk lookup, m *Saving, ro *Role) (savingProject, bool) {
	var best savingProject
	found := false
	for _, region := range savingRegions(c, ro) {
		if pr, ok := bestSavingProjectIn(c, lk, m, ro, region); ok && (!found || sav.Better(savingCandidate(pr, 0), savingCandidate(best, 0))) {
			best, found = pr, true
		}
	}
	return best, found
}

// savingCandidate is the project as the pure rule compares it: by benefit over cost.
func savingCandidate(p savingProject, id int64) sav.Candidate {
	return sav.Candidate{Benefit: p.benefit, Cost: p.cost, ID: id}
}

// savingRegions are the distinct regions in which the role has a group, in the order of
// the groups.
func savingRegions(c *Config, ro *Role) []string {
	var out []string
	for _, g := range c.Groups {
		if g.Role == ro.ID && !slices.Contains(out, g.Region) {
			out = append(out, g.Region)
		}
	}
	return out
}

// valueSavingProject values one build recipe in a region: its benefit by 5.3, its cost
// by 5.4 and its outlay by 5.6.1, all at the reference prices. ok is false when a
// price is not declared or a product does not fit.
func valueSavingProject(c *Config, lk lookup, m *Saving, ro *Role, region string, rc *Recipe) (savingProject, bool) {
	cl, ok := savingClass(c, rc.Builds.Class)
	if !ok {
		return savingProject{}, false
	}
	inputs, tools, ok := savingInputs(lk, region, rc)
	if !ok {
		return savingProject{}, false
	}
	pvB, pvA := savingJobValues(lk, region, ro, cl.ID)
	b := ro.Saver
	var benefit int64
	if row, has := savingWorthRow(b, cl.ID); has {
		rate := sav.WorthRate(cl.Decay, cl.WearPerUse, 1, cl.Capacity)
		benefit = sav.WorthBenefit(row.PerTick, sav.ServiceTime(cl.Durability, rate, b.PaybackTicks), 0, row.Count, MaxMoney)
	} else if cl.Kind == kindSite {
		gain := sav.BuildGain(sav.JobValue(pvB, pvA, 1, 0), sav.JobValue(pvB, pvA, 1, cl.Capacity))
		rate := sav.Rate(cl.Decay, cl.WearPerUse, min(1, cl.Capacity), max(rc.Duration, 1))
		benefit = sav.SiteBenefit(gain, sav.ServiceTime(cl.Durability, rate, b.PaybackTicks), MaxMoney)
	}
	cost := sav.Cost(inputs, sav.Opportunity(sav.JobValue(pvB, pvA, 1, 0), rc.Duration), MaxMoney)
	upkeep, uok := savingUpkeep(c, lk, region, ro)
	if !uok {
		return savingProject{}, false
	}
	return savingProject{benefit: benefit, cost: cost, outlay: sav.Outlay(inputs, tools, upkeep, rc.Duration, m.SafetyTicks)}, true
}

// savingClass is the facility class with the ID.
func savingClass(c *Config, id string) (FacilityClass, bool) {
	if inf := c.Modules.Infrastructure; inf != nil {
		for _, cl := range inf.Classes {
			if cl.ID == id {
				return cl, true
			}
		}
	}
	return FacilityClass{}, false
}

// savingWorthRow is the role's Worth row for the class.
func savingWorthRow(b *Saver, class string) (ProjectWorth, bool) {
	for _, w := range b.Worth {
		if w.Class == class {
			return w, true
		}
	}
	return ProjectWorth{}, false
}

// savingInputs is the cost of the inputs and of the wear of the tools of one execution,
// and the price of the tools the household would have to buy (it holds none), at the
// reference prices of the region.
func savingInputs(lk lookup, region string, rc *Recipe) (inputs, tools int64, ok bool) {
	var net household.Net
	for _, in := range rc.Inputs {
		cm := lk.commodity[in.Commodity]
		if cm == nil {
			return 0, 0, false
		}
		net.Input(int64(in.Quantity), int64(lk.price(region, cm)))
	}
	for _, t := range rc.Requires {
		cm := lk.commodity[t.Commodity]
		if cm == nil {
			return 0, 0, false
		}
		price := int64(lk.price(region, cm))
		w, wok := production.WearCost(production.Wear{WearPerUse: t.WearPerUse, Durability: cm.Durability}, price)
		if !wok {
			return 0, 0, false
		}
		net.Cost(w)
		var tok bool
		if tools, tok = checked.Add(tools, price); !tok {
			return 0, 0, false
		}
	}
	cost, ok := net.Value()
	if cost == math.MinInt64 {
		return 0, 0, false
	}
	return -cost, tools, ok
}

// savingJobValues are pv_b and pv_a of spec-saving-and-investment.md 5.3.1 for a class
// at the reference prices: the best value per tick of a valued recipe on the role's
// list that needs the class, and of one that needs nothing.
func savingJobValues(lk lookup, region string, ro *Role, class string) (pvB, pvA int64) {
	for _, id := range ro.Recipes {
		rc := lk.recipe[id]
		if rc == nil || rc.Builds != nil || rc.Repairs != nil {
			continue
		}
		net, ok := savingNet(lk, region, rc)
		if !ok || net <= 0 {
			continue
		}
		pv := sav.PerTick(net, rc.Duration)
		switch {
		case slices.Contains(rc.Needs, class):
			pvB = max(pvB, pv)
		case len(rc.Needs) == 0:
			pvA = max(pvA, pv)
		}
	}
	return pvB, pvA
}

// savingNet is the net value of one execution of a recipe at the reference prices: the
// non-discard outputs at the yield as written, less the inputs and the wear of the tools.
func savingNet(lk lookup, region string, rc *Recipe) (int64, bool) {
	var net household.Net
	for _, o := range rc.Outputs {
		if cm := lk.commodity[o.Commodity]; cm != nil && !o.Discard {
			net.Output(int64(o.Quantity), wholeBP, int64(lk.price(region, cm)))
		}
	}
	inputs, _, ok := savingInputs(lk, region, rc)
	if !ok {
		return 0, false
	}
	value, vok := net.Value()
	out, sok := checked.Sub(value, inputs)
	return out, vok && sok
}

// savingUpkeep is the cost of one tick of upkeep of a household of the role in a region
// at the reference prices, with the region's extra upkeep.
func savingUpkeep(c *Config, lk lookup, region string, ro *Role) (int64, bool) {
	rg := lk.region[region]
	if rg == nil {
		return 0, false
	}
	return listCost(lk, region, effectiveUpkeep(c, ro), rg.ExtraUpkeep)
}

// savingTargetExceedsMoney is saving_target_exceeds_money (spec-saving-and-investment.md
// 8.2.2): the households of the roles with a block times the floor of the target
// (the reserve) plus the outlay of the role's best build project, at the reference
// prices, is more than the money the households of the start hold.
func savingTargetExceedsMoney(c *Config, lk lookup, m *Saving) bool {
	var target, money int64
	for _, g := range c.Groups {
		ro := lk.role[g.Role]
		if ro == nil {
			continue
		}
		held, hok := checked.Mul(g.Households, int64(ro.StartMoney))
		if !hok {
			held = math.MaxInt64 // saturate: no target of the same configuration can pass it unseen
		}
		money = satAdd64(money, held)
		if ro.Saver == nil {
			continue
		}
		upkeep, ok := savingUpkeep(c, lk, g.Region, ro)
		if !ok {
			continue
		}
		reserve, rok := checked.Mul(upkeep, c.World.ReserveTicks)
		if !rok {
			return true
		}
		floor := sav.Floor(reserve, 0)
		if pr, found := bestSavingProjectIn(c, lk, m, ro, g.Region); found {
			floor = satAdd64(floor, pr.outlay)
		}
		need, nok := checked.Mul(g.Households, floor)
		if !nok {
			return true
		}
		target = satAdd64(target, need)
	}
	return target > money
}

// bestSavingProjectIn is the best build project of the role in one region.
func bestSavingProjectIn(c *Config, lk lookup, m *Saving, ro *Role, region string) (savingProject, bool) {
	var best savingProject
	found := false
	var bestAt int64
	for n, id := range ro.Recipes {
		if rc := lk.recipe[id]; rc != nil && rc.Builds != nil {
			pr, ok := valueSavingProject(c, lk, m, ro, region, rc)
			if ok && (!found || sav.Better(savingCandidate(pr, int64(n)), savingCandidate(best, bestAt))) {
				best, bestAt, found = pr, int64(n), true
			}
		}
	}
	return best, found
}
