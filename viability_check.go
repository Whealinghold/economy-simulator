package economysim

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/Whealinghold/economy-sim/internal/checked"
	"github.com/Whealinghold/economy-sim/internal/steady"
)

// This file holds the viability checks V1 to V4 as Validate and Warnings run
// them (spec-viability.md 4, 10; spec-config.md 7.6): necessary and optimistic,
// pooled free trade and the best yield across regions, each programme under the
// size gate. checkViability is the one entry point; the solves and the exported
// result of SolveSteadyState are viability.go's, its table viability_table.go's.

// viabilityOutcome is what one run of the checks found: the errors of the five
// rules, the warnings, and fault, set only for a defect of the solver (a
// certificate that failed), which no input can cause.
type viabilityOutcome struct {
	errs     []*ViabilityError
	warnings []Warning
	fault    error
}

// fail records the first defect of the solver.
func (o *viabilityOutcome) fail(err error) {
	if o.fault != nil {
		return
	}
	if errors.Is(err, steady.ErrInconsistent) {
		o.fault = ErrSolveInconsistent
		return
	}
	o.fault = fmt.Errorf("%w: %v", ErrSolveInconsistent, err)
}

// warn adds a warning.
func (o *viabilityOutcome) warn(rule string, path ...string) {
	o.warnings = append(o.warnings, Warning{Path: path, Rule: rule})
}

// skipped adds viability_check_skipped naming the part not run.
func (o *viabilityOutcome) skipped(part string, ids ...string) {
	o.warn("viability_check_skipped", append([]string{"viability", part}, ids...)...)
}

// reject adds an error of one of the five rules.
func (o *viabilityOutcome) reject(rule string, path []string, value, bound string) {
	o.errs = append(o.errs, &ViabilityError{Rule: rule, Path: path, Value: value, Bound: bound})
}

// checkViability runs V1 to V4 and the viability warnings on a canonical
// configuration that phases 1 to 5 of Validate found clean (D125): V1 and V4
// always, V2 and V3 under the size gate, V3 only when V1 and V2 found nothing.
// It has no side effect and draws nothing. lk is the lookup of c.
func checkViability(c *Config, lk lookup) viabilityOutcome {
	return newViaModel(c).check(lk)
}

// check runs the checks on a model; Validate builds the model first because the
// empty-offset rule of its phase 6 reads it too.
func (m *viaModel) check(lk lookup) viabilityOutcome {
	var o viabilityOutcome
	unproducible := m.sourced(&o)
	unproductive := m.cycles(&o)
	m.storage(&o, lk)
	if !unproducible && !unproductive {
		m.labour(&o)
		m.wantsExceed(&o)
	}
	m.regions(&o)
	return o
}

// realm builds the producibility problem of the existing activities (4.2): a
// Raw commodity is sourced, a tool is an input, and an activity is Listed when a
// role with a group in its region lists the recipe.
func (m *viaModel) realm() steady.Realm {
	c := m.c
	r := steady.Realm{Commodities: len(c.Commodities), Regions: len(c.Regions), Sourced: make([]bool, len(c.Commodities))}
	for i := range c.Commodities {
		r.Sourced[i] = c.Commodities[i].Raw
	}
	for _, reg := range m.withHouseholds {
		for _, a := range m.acts[reg] {
			mk := steady.Maker{Region: int(reg), Listed: m.listed[reg][a.recipe]}
			for _, in := range a.act.Inputs {
				mk.Needs = append(mk.Needs, in.Commodity)
			}
			for _, t := range a.act.Tools {
				mk.Needs = append(mk.Needs, t.Commodity)
			}
			for _, out := range a.act.Outputs {
				if !out.Discard {
					mk.Makes = append(mk.Makes, out.Commodity)
				}
			}
			r.Makers = append(r.Makers, mk)
		}
	}
	return r
}

// upkeepGoods returns, ascending, the commodities some group's upkeep needs.
func (m *viaModel) upkeepGoods() []int {
	var out []int
	for ci := range m.c.Commodities {
		for _, r := range m.withHouseholds {
			if m.upkeep[r][ci] > 0 {
				out = append(out, ci)
				break
			}
		}
	}
	return out
}

// sourced is V1 (4.2): the rules unproducible_in_realm and role_cannot_work. It
// reports whether either rule fired, which gates V3.
func (m *viaModel) sourced(o *viabilityOutcome) (unproducible bool) {
	c := m.c
	realm := m.realm()
	missing := steady.Unproducible(realm, m.upkeepGoods())
	for _, ms := range missing {
		id := c.Commodities[ms.Commodity].ID
		o.reject("unproducible_in_realm", []string{"commodities", id}, id, "")
	}
	uses, exists := m.roleUses()
	cannot := steady.CannotWork(uses, exists)
	for _, st := range cannot {
		o.reject("role_cannot_work", []string{"roles", c.Roles[st.Role].ID}, c.Regions[st.Region].ID, "")
	}
	return len(missing) > 0 || len(cannot) > 0
}

// roleUses lists, for each role, the regions of its groups and the recipes on
// its list, and for each region which recipes exist there.
func (m *viaModel) roleUses() (uses []steady.RoleUse, exists [][]bool) {
	c := m.c
	uses = make([]steady.RoleUse, len(c.Roles))
	for _, g := range c.Groups {
		r, _ := c.regionID(g.Region)
		ro, _ := c.roleID(g.Role)
		if n := len(uses[ro].Regions); n == 0 || uses[ro].Regions[n-1] != int(r) {
			uses[ro].Regions = append(uses[ro].Regions, int(r))
		}
	}
	exists = make([][]bool, len(c.Regions))
	for r := range exists {
		exists[r] = make([]bool, len(c.Recipes))
		for _, a := range m.acts[r] {
			exists[r][a.recipe] = true
		}
	}
	for i := range uses {
		for _, k := range m.roles.row(RoleID(i)).recipes {
			uses[i].Recipes = append(uses[i].Recipes, int(k))
		}
	}
	return uses, exists
}

// candidates lists every existing activity for the cycle checks (4.3).
func (m *viaModel) candidates() []steady.Candidate {
	var cs []steady.Candidate
	for _, r := range m.withHouseholds {
		for _, a := range m.acts[r] {
			cs = append(cs, steady.Candidate{Recipe: int(a.recipe), Activity: a.act})
		}
	}
	return cs
}

// cycles is V2 (4.3) and the cycle gain (7.6 item 3): cycle_not_productive for a
// component with no surplus, and the warning cycle_cost_gain for a productive
// one whose gain reaches 1. It reports whether any component was unproductive.
func (m *viaModel) cycles(o *viabilityOutcome) (unproductive bool) {
	c := m.c
	cs := m.candidates()
	comps, err := steady.Cycles(context.Background(), len(c.Commodities), cs, m.maxCells, m.maxPivots)
	if errors.Is(err, steady.ErrOverflow) { // an input size, not a solver defect
		if o.fault == nil {
			o.fault = fmt.Errorf("%w: a cycle floor out of int64 range", ErrOverflow)
		}
		return false
	}
	if err != nil {
		o.fail(err)
		return false
	}
	for _, comp := range comps {
		low := c.Commodities[comp.Commodities[0]].ID
		switch {
		case comp.Skipped:
			o.skipped("v2", low)
		case !comp.Productive:
			unproductive = true
			o.reject("cycle_not_productive", []string{"commodities", low}, i64(comp.SBP), "0")
		default:
			m.gain(o, comp.Commodities, cs)
		}
	}
	return unproductive
}

// gain warns for each simple cycle of a productive component whose gain
// CostFloor / 10 000 * q_in / q_out is 1 or more, once per lowest commodity.
func (m *viaModel) gain(o *viabilityOutcome, comp []int, cs []steady.Candidate) {
	c := m.c
	gains, ok := steady.CycleGains(comp, cs, m.maxSteps)
	if !ok {
		o.skipped("cycle_gain", c.Commodities[comp[0]].ID)
		return
	}
	var warned []int
	for _, g := range gains {
		low := g.Commodities[0]
		if c.World.CostFloor >= 1 && g.Ratio.AtLeast(wholeBP, c.World.CostFloor) && !slices.Contains(warned, low) {
			warned = append(warned, low)
			o.warn("cycle_cost_gain", "commodities", c.Commodities[low].ID)
		}
	}
}

// model builds the programme of the given regions at the work share phi: one
// region is the autarky form, several the pooled free-trade form (D129).
func (m *viaModel) model(regions []RegionID, phi steady.Ratio) steady.Model {
	md := steady.Model{Commodities: len(m.c.Commodities), Year: m.year, Phi: phi}
	for k, r := range regions {
		md.Regions = append(md.Regions, steady.Region{Households: m.households[r], Upkeep: m.upkeep[r]})
		for _, a := range m.acts[r] {
			act := a.act
			act.Region = k
			md.Activities = append(md.Activities, act)
		}
	}
	return md
}

// solve runs the programme under the size gate and the pivot cap. skipped says
// the gate or the cap kept it from running; a returned error is a defect.
func (m *viaModel) solve(md steady.Model) (out steady.Outcome, skipped bool, err error) {
	rows, cols := steady.Size(md)
	if cells, ok := checked.Mul(int64(rows), int64(cols)); !ok || cells > m.maxCells {
		return steady.Outcome{}, true, nil
	}
	out, err = steady.SolveModel(context.Background(), md, m.maxPivots)
	if errors.Is(err, steady.ErrLimit) {
		return steady.Outcome{}, true, nil
	}
	return out, false, err
}

// noProgramme reports that the regions have nothing to be viable for: an empty
// basket, or a basket whose goods are all unsourced Raw markers (4.2.1, 5.3).
func (m *viaModel) noProgramme(regions []RegionID) bool {
	for ci := range m.c.Commodities {
		if slices.ContainsFunc(regions, func(r RegionID) bool { return m.upkeep[r][ci] > 0 }) {
			if !m.c.Commodities[ci].Raw || m.hasSource(regions, ci) {
				return false
			}
		}
	}
	return true
}

// hasSource reports that an existing activity with no inputs in the regions
// outputs the commodity: what stands behind a Raw marker.
func (m *viaModel) hasSource(regions []RegionID, ci int) bool {
	for _, r := range regions {
		for _, a := range m.acts[r] {
			if len(a.act.Inputs) > 0 {
				continue
			}
			if slices.ContainsFunc(a.act.Outputs, func(it steady.Item) bool { return it.Commodity == ci && !it.Discard }) {
				return true
			}
		}
	}
	return false
}

// labour is V3 (4.4): labour_short when the pooled free-trade programme at full
// labour has a surplus factor below 1.
func (m *viaModel) labour(o *viabilityOutcome) {
	regions := m.withHouseholds
	if m.noProgramme(regions) {
		return
	}
	res, skipped, err := m.solve(m.model(regions, steady.Ratio{Num: 1, Den: 1}))
	m.pooled = &pooledSolve{res, skipped, err}
	switch {
	case err != nil:
		o.fail(err)
	case skipped:
		o.skipped("v3_pooled")
	case res.SurplusBP < wholeBP:
		o.reject("labour_short", []string{"regions"}, i64(res.SurplusBP), i64(wholeBP))
	}
}

// pooledSolve is the solve of the pooled upkeep-only programme at full labour,
// kept so labour and wantsExceed do not run the simplex twice.
type pooledSolve struct {
	out     steady.Outcome
	skipped bool
	err     error
}

// wantsExceed warns wants_exceed_labour (FR-VIA-057): the demand module is on,
// the pooled programme with the wants has a surplus factor below 1 or is
// infeasible, and the upkeep-only one reaches 1. Size-gated like V3 and never an
// error.
func (m *viaModel) wantsExceed(o *viabilityOutcome) {
	regions := m.withHouseholds
	if m.c.Modules.Demand == nil || m.noProgramme(regions) {
		return
	}
	one := steady.Ratio{Num: 1, Den: 1}
	if m.pooled == nil { // labour has not run: solve the upkeep-only programme here
		base, skipped, err := m.solve(m.model(regions, one))
		m.pooled = &pooledSolve{base, skipped, err}
	}
	base, skipped, err := m.pooled.out, m.pooled.skipped, m.pooled.err // an error is labour's to report
	if skipped {
		o.skipped("wants_pooled")
	}
	if err != nil || skipped || base.SurplusBP < wholeBP {
		return
	}
	md, err := m.wantsModel(regions, one)
	if err != nil {
		o.fail(err) // the same overflow SolveSteadyState reports
		return
	}
	res, skipped, err := m.solve(md)
	m.wantsVerdict(o, res, skipped, err)
}

// wantsVerdict turns the solve of the programme with the wants into the warning
// of wantsExceed: infeasible, or a surplus factor below 1.
func (m *viaModel) wantsVerdict(o *viabilityOutcome, res steady.Outcome, skipped bool, err error) {
	switch {
	case errors.Is(err, steady.ErrInfeasible):
		o.warn("wants_exceed_labour", "regions")
	case err != nil:
		o.fail(err)
	case skipped:
		o.skipped("wants_pooled")
	case res.SurplusBP < wholeBP:
		o.warn("wants_exceed_labour", "regions")
	}
}

// producers lists, by commodity, the completion pattern of each existing
// activity of a region that outputs it (4.5).
func (m *viaModel) producers(r RegionID) [][]steady.Producer {
	out := make([][]steady.Producer, len(m.c.Commodities))
	for _, a := range m.acts[r] {
		p := steady.Producer{Every: a.allYear, Completions: a.count.Completions}
		for _, it := range a.act.Outputs {
			if !it.Discard {
				out[it.Commodity] = append(out[it.Commodity], p)
			}
		}
	}
	return out
}

// allYearInputs marks the inputs of the activities of a region that are open on
// every tick of the year (the condition of storage_gap_input).
func (m *viaModel) allYearInputs(r RegionID) []bool {
	out := make([]bool, len(m.c.Commodities))
	for _, a := range m.acts[r] {
		if !a.allYear {
			continue
		}
		for _, in := range a.act.Inputs {
			out[in.Commodity] = true
		}
	}
	return out
}

// storage is V4 (4.5): storage_gap for an upkeep good whose shelf life is above 0
// and below its longest supply gap in a region, and the warning storage_gap_input
// for an input of an all-year activity. A commodity no activity of the region
// makes has no supply gap to bridge there (it comes from another region), so it
// is left to the pooled reading of the other checks.
func (m *viaModel) storage(o *viabilityOutcome, lk lookup) {
	c := m.c
	for _, r := range m.withHouseholds {
		prods, inputs := m.producers(r), m.allYearInputs(r)
		for ci := range c.Commodities {
			id := c.Commodities[ci].ID
			shelf := lk.commodity[id].ShelfLife
			if shelf == 0 || len(prods[ci]) == 0 {
				continue
			}
			gap := steady.Gap(m.year, prods[ci])
			if !steady.ExceedsShelf(gap, shelf) {
				continue
			}
			p := []string{"regions", c.Regions[r].ID, "commodities", id}
			switch {
			case m.upkeep[r][ci] > 0:
				o.reject("storage_gap", p, i64(gap), i64(shelf))
			case inputs[ci]:
				o.warn("storage_gap_input", p...)
			}
		}
	}
}

// regions runs the per-region warnings (4.4, 9.5) from the autarky programme of
// each region with households.
func (m *viaModel) regions(o *viabilityOutcome) {
	prod := steady.Producible(m.realm())
	for _, r := range m.withHouseholds {
		if m.noProgramme([]RegionID{r}) || m.imports(o, r, prod) {
			continue
		}
		m.autarky(o, r)
	}
}

// imports warns region_not_self_sufficient for each upkeep good the region
// cannot produce from its own activities, and reports whether there was one: such
// a region has no autarky programme worth solving.
func (m *viaModel) imports(o *viabilityOutcome, r RegionID, prod steady.Producibility) bool {
	found := false
	for ci := range m.c.Commodities {
		if m.upkeep[r][ci] > 0 && !prod.Region[r][ci] {
			found = true
			o.warn("region_not_self_sufficient", "regions", m.c.Regions[r].ID, "commodities", m.c.Commodities[ci].ID)
		}
	}
	return found
}

// autarky solves a region alone at full labour and warns
// region_not_self_sufficient when its surplus factor is below 1 and
// subsistence_exceeds_expected_work when the labour share to subsist is above the
// mean expected work share; it then runs the nominal programme.
func (m *viaModel) autarky(o *viabilityOutcome, r RegionID) {
	id := m.c.Regions[r].ID
	res, skipped, err := m.solve(m.model([]RegionID{r}, steady.Ratio{Num: 1, Den: 1}))
	switch {
	case err != nil:
		o.fail(err)
		return
	case skipped:
		o.skipped("v3_region", id)
	case res.SurplusBP < wholeBP:
		o.warn("region_not_self_sufficient", "regions", id)
	}
	if !m.sumsOK[r] {
		return
	}
	if !skipped && res.HasLabourShare && shareAboveMean(res.LabourShareBP, m.expected[r], m.year, m.households[r]) {
		o.warn("subsistence_exceeds_expected_work", "regions", id)
	}
	m.nominal(o, r)
}

// nominal solves the region alone at the work share its households expect and
// warns target_above_surplus when their mean target ratio is above its surplus
// factor.
func (m *viaModel) nominal(o *viabilityOutcome, r RegionID) {
	id := m.c.Regions[r].ID
	den, ok := checked.Mul(m.year, m.households[r])
	if !ok {
		return
	}
	res, skipped, err := m.solve(m.model([]RegionID{r}, steady.Ratio{Num: m.expected[r], Den: den}))
	switch {
	case err != nil:
		o.fail(err)
	case skipped:
		o.skipped("nominal", id)
	default:
		if targetAboveSurplus(m.targets[r], res.SurplusBP, m.households[r]) {
			o.warn("target_above_surplus", "regions", id)
		}
	}
}

// shareAboveMean reports that a labour share in basis points is above the mean
// expected work share of a region, expected / (year * households) of the year,
// in basis points. An integer share is above the exact mean exactly when it is
// above the mean rounded down. It is false when the arithmetic does not fit.
func shareAboveMean(share, expected, year, households int64) bool {
	den, ok1 := checked.Mul(year, households)
	mean, ok2 := checked.MulDiv(expected, wholeBP, den)
	return ok1 && ok2 && share > mean
}

// targetAboveSurplus reports that the household-weighted mean target ratio of a
// region, targets / households, is above its nominal surplus factor in basis
// points: targets > surplus * households. It is false when the product does not
// fit.
func targetAboveSurplus(targets, surplusBP, households int64) bool {
	limit, ok := checked.Mul(surplusBP, households)
	return ok && targets > limit
}
