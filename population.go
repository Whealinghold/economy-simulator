package economysim

import (
	"cmp"
	"slices"

	"github.com/Whealinghold/economy-sim/internal/checked"
	"github.com/Whealinghold/economy-sim/internal/endowment"
	"github.com/Whealinghold/economy-sim/internal/ledger"
	"github.com/Whealinghold/economy-sim/internal/population"
)

// ScaleKind names which of a region's two population scales a call sets.
type ScaleKind string

// The two scales of a region (spec-population.md 3.6).
const (
	// ScaleBirth is the multiplier of the birth schedule.
	ScaleBirth ScaleKind = "birth"
	// ScaleDeath is the multiplier of the death schedule.
	ScaleDeath ScaleKind = "death"
)

// popCount is the counts of one (region, role) in the tick being stepped: the
// fields of PopulationChanged (spec-population.md 10.1). BirthsCeiling is SET's
// and has no counter here.
type popCount struct {
	exposed, unfed, noRatio                     int64
	birthsDue, deathsDue, deathsCut, deathsHeld int64
	birthsCapped, birthsUnfunded, clamped       int64
	birthsDone, deathsDone                      int64
}

// popState is the state of the module, nil on the world when it is off (spec-
// population.md 11). The death carry is indexed by AgentID and grown at the
// step with the start of every id it has not seen (D423); the birth carry by
// region * roles + role; the effective scales, the configured scales and the cap
// by RegionID. The last four slices are scratch for the region being stepped.
type popState struct {
	denom          int64
	birthK, deathK []population.Knot
	minHouseholds  int64
	stakeBP        int64
	roles          int
	deathCarry     []int64
	birthCarry     []int64
	effBirth       []int64
	effDeath       []int64
	rowBirth       []int64
	rowDeath       []int64
	cap            []int64
	cnt            []popCount
	num            []int64
	group          []int64
}

// newPopulation builds the state from a canonical configuration that passed
// validation, or returns nil when the module is off. Every scale starts at its
// configured value (spec-population.md 6.6).
func newPopulation(c *Config) *popState {
	m := c.Modules.Population
	if m == nil {
		return nil
	}
	denom, ok := checked.Mul(wholeBP, c.Calendar.TicksPerYear())
	if !ok {
		invariant("the death carry bound overflows after validation")
	}
	regions, roles := len(c.Regions), len(c.Roles)
	p := &popState{
		denom: denom, birthK: popKnots(m.BirthSchedule), deathK: popKnots(m.DeathSchedule),
		minHouseholds: m.MinHouseholds, stakeBP: m.BirthStake, roles: roles,
		birthCarry: make([]int64, regions*roles),
		rowBirth:   make([]int64, regions), rowDeath: make([]int64, regions), cap: make([]int64, regions),
		cnt: make([]popCount, roles), num: make([]int64, roles), group: make([]int64, roles),
	}
	for g := range p.rowBirth {
		p.rowBirth[g], p.rowDeath[g] = wholeBP, wholeBP
	}
	for _, r := range m.Regions {
		if g, ok := c.regionID(r.Region); ok {
			p.rowBirth[g], p.rowDeath[g], p.cap[g] = r.BirthScale, r.DeathScale, r.HouseholdCap
		}
	}
	p.effBirth, p.effDeath = slices.Clone(p.rowBirth), slices.Clone(p.rowDeath)
	return p
}

// populationReasons is the ledger reasons the module declares.
func populationReasons() []ledger.Reason { return []ledger.Reason{ledger.Deaths} }

// carryOf returns the death carry of agent id: the slice entry, or its start
// while the step has not seen the id (spec-population.md 3.8.3).
func (p *popState) carryOf(id AgentID) int64 {
	if int(id) < len(p.deathCarry) {
		return p.deathCarry[id]
	}
	return population.Offset(int64(id), p.denom)
}

// This file is the population module (spec-population.md): births and deaths as
// the formation and dissolution of households, driven by the household's own
// welfare ratio. The pure rules are in internal/population. The file holds, in
// the order of architecture.md 2.10.1: the state, the configuration surface (the
// canonical form, the validator and the warnings), the step of P7.1(b), the
// hooks the executor calls and the scale slot.

// canonical returns a copy of the module with the schedules cloned in knot order
// and the region rows sorted by region (spec-population.md 8, spec-config.md
// 3.5). A nil module stays nil; a module that is on stays on.
func (p *Population) canonical() *Population {
	if p == nil {
		return nil
	}
	return &Population{
		BirthSchedule: slices.Clone(p.BirthSchedule),
		DeathSchedule: slices.Clone(p.DeathSchedule),
		MinHouseholds: p.MinHouseholds,
		BirthStake:    p.BirthStake,
		Regions:       sortedBy(p.Regions, func(a, b RegionPopulation) int { return cmp.Compare(a.Region, b.Region) }),
	}
}

// popKnots converts a schedule for internal/population, inline at the call
// (architecture.md 2.9).
func popKnots(ks []Knot) []population.Knot {
	out := make([]population.Knot, len(ks))
	for i, k := range ks {
		out[i] = population.Knot{Ratio: k.Ratio, Rate: k.Rate}
	}
	return out
}

// validatePopulation is the module's part of phase 2 and 3 of spec-config.md
// 7.1.6 (spec-population.md 9.1): the bounds of each field, the schedules and the
// region rows. Every error is ErrOutOfBounds, ErrUnknownRegion or ErrDuplicateID
// with the path of the field; the module adds no sentinel.
func (v *validator) validatePopulation() {
	m := v.c.Modules.Population
	if m == nil {
		return
	}
	base := []string{"modules", "population"}
	v.popSchedule(path(base, "birth_schedule"), m.BirthSchedule)
	v.popSchedule(path(base, "death_schedule"), m.DeathSchedule)
	v.inRange(path(base, "min_households"), m.MinHouseholds, 0, MaxHouseholds)
	v.inRange(path(base, "birth_stake"), m.BirthStake, 0, wholeBP)
	if len(m.Regions) > MaxRegions {
		v.rep.add(ErrOutOfBounds, path(base, "regions"), itoa(len(m.Regions)), itoa(MaxRegions))
		return // the rest is linear in the count it just refused
	}
	seen := make(map[string]bool, len(m.Regions))
	for i := range m.Regions {
		r := &m.Regions[i]
		rb := path(base, "region", r.Region)
		if v.lk.region[r.Region] == nil {
			v.rep.add(ErrUnknownRegion, path(rb, "region"), r.Region, "")
		}
		if seen[r.Region] {
			v.rep.add(ErrDuplicateID, path(rb, "region"), r.Region, "")
		}
		seen[r.Region] = true
		v.inRange(path(rb, "birth_scale"), r.BirthScale, 0, MaxBasisPoints)
		v.inRange(path(rb, "death_scale"), r.DeathScale, 0, MaxBasisPoints)
		v.inRange(path(rb, "household_cap"), r.HouseholdCap, 0, MaxHouseholds)
	}
}

// popSchedule checks one schedule: 1 to MaxKnots knots, each ratio and rate in
// its range, and the ratios strictly increasing (spec-population.md 9.1,
// FR-POP-026). A count out of its bound stops the per-knot rules.
func (v *validator) popSchedule(p []string, ks []Knot) {
	if len(ks) < 1 || len(ks) > MaxKnots {
		v.rep.add(ErrOutOfBounds, p, itoa(len(ks)), itoa(MaxKnots))
		return
	}
	for i, k := range ks {
		kp := path(p, itoa(i))
		v.inRange(path(kp, "ratio"), k.Ratio, 0, MaxBasisPoints)
		v.inRange(path(kp, "rate"), k.Rate, 0, population.MaxRate)
		if i > 0 && k.Ratio <= ks[i-1].Ratio {
			v.rep.add(ErrOutOfBounds, path(kp, "ratio"), i64(k.Ratio), i64(ks[i-1].Ratio))
		}
	}
}

// scheduleOK reports whether a schedule is one the rules can read: 1 to MaxKnots
// knots, ratios strictly increasing and in range, rates in range. The warnings
// run on a configuration that may not validate and read only such schedules.
func scheduleOK(ks []Knot) bool {
	if len(ks) < 1 || len(ks) > MaxKnots {
		return false
	}
	for i, k := range ks {
		if k.Ratio < 0 || k.Ratio > MaxBasisPoints || k.Rate < 0 || k.Rate > population.MaxRate ||
			(i > 0 && k.Ratio <= ks[i-1].Ratio) {
			return false
		}
	}
	return true
}

// replacementOf returns the replacement ratio r* of the module and the slope of
// birth - death on the segment that holds it: dR the rise in rate and dX the run
// in ratio (spec-population.md 5.4, 5.7). lo is the lower end of the segment. ok
// is false when a schedule is unreadable or the schedules have no replacement
// ratio.
func (p *Population) replacementOf() (rstar, lo, dR, dX int64, ok bool) {
	if !scheduleOK(p.BirthSchedule) || !scheduleOK(p.DeathSchedule) {
		return 0, 0, 0, 0, false
	}
	b, d := popKnots(p.BirthSchedule), popKnots(p.DeathSchedule)
	rstar, lo, hi, ok := population.Replacement(b, d)
	if !ok {
		return 0, 0, 0, 0, false
	}
	g := func(x int64) int64 {
		br, _, _ := population.Rate(b, x, wholeBP)
		dr, _, _ := population.Rate(d, x, wholeBP)
		return br - dr
	}
	return rstar, lo, g(hi) - g(lo), hi - lo, true
}

// populationWarnings returns the warnings of spec-population.md 9.2 and the two
// diagnostics (5.7, 4.6.1), in no particular order. They read a configuration
// that may not validate, so each guards what it reads. None fails Validate.
func populationWarnings(c *Config) []Warning {
	m := c.Modules.Population
	if m == nil {
		return nil
	}
	var out []Warning
	add := func(rule string, segs ...string) { out = append(out, Warning{Path: segs, Rule: rule}) }
	base := []string{"modules", "population"}
	popRenewalWarnings(c, add)
	if !hasUpkeep(c) {
		add("population_without_upkeep", base...)
	}
	if !hasDraws(c) && !hasCap(m) {
		add("no_fixed_factor", base...)
	}
	if scheduleOK(m.BirthSchedule) && scheduleOK(m.DeathSchedule) {
		popScheduleWarnings(c, m, base, add)
	}
	return out
}

// popRenewalWarnings raises renewal_with_population for a non-default Renewal
// above 0 and the informational renewal_ignored at the default (spec-population.md
// 4.6.1, 9.2). At 0 neither fires.
func popRenewalWarnings(c *Config, add func(string, ...string)) {
	switch r := c.World.Renewal; {
	case r == DefaultWorldParams().Renewal:
		add("renewal_ignored", "world", "renewal")
	case r > 0:
		add("renewal_with_population", "world", "renewal")
	}
}

// popScheduleWarnings raises the warnings that read the two schedules:
// schedule_direction, no_replacement_ratio, replacement_ratio,
// replacement_above_closed_ceiling, population_gain and the two plan warnings of
// the resources module.
func popScheduleWarnings(c *Config, m *Population, base []string, add func(string, ...string)) {
	if fallsAnywhere(m.BirthSchedule) {
		add("schedule_direction", append(slices.Clone(base), "birth_schedule")...)
	}
	if risesAnywhere(m.DeathSchedule) {
		add("schedule_direction", append(slices.Clone(base), "death_schedule")...)
	}
	rstar, lo, dR, dX, ok := m.replacementOf()
	if !ok {
		add("no_replacement_ratio", base...)
		return
	}
	// The path ends in the lower knot of the birth schedule at or below the
	// segment, the index of the segment the figure is read at (D434).
	idx := max(0, sortedSearchBelow(m.BirthSchedule, lo))
	add("replacement_ratio", append(slices.Clone(base), "birth_schedule", itoa(idx))...)
	if rstar > wholeBP && c.Modules.Demand == nil {
		add("replacement_above_closed_ceiling", base...)
	}
	if population.GainWarn(rstar, dR, dX, longestWindow(c), c.Calendar.TicksPerYear()) {
		add("population_gain", base...)
	}
	popPlanWarnings(c, rstar, dR, dX, add)
}

// sortedSearchBelow returns the index of the last knot with a ratio at or below
// x, -1 when there is none.
func sortedSearchBelow(ks []Knot, x int64) int {
	i, _ := slices.BinarySearchFunc(ks, x, func(k Knot, t int64) int { return cmp.Compare(k.Ratio, t+1) })
	return i - 1
}

// fallsAnywhere reports whether a schedule's rate falls from one knot to the next.
func fallsAnywhere(ks []Knot) bool {
	for i := 1; i < len(ks); i++ {
		if ks[i].Rate < ks[i-1].Rate {
			return true
		}
	}
	return false
}

// risesAnywhere reports whether a schedule's rate rises from one knot to the next.
func risesAnywhere(ks []Knot) bool {
	for i := 1; i < len(ks); i++ {
		if ks[i].Rate > ks[i-1].Rate {
			return true
		}
	}
	return false
}

// hasUpkeep reports whether any level of the configuration names an upkeep, the
// no_upkeep case of spec-agents.md 13.
func hasUpkeep(c *Config) bool {
	if len(c.World.Upkeep) > 0 {
		return true
	}
	for i := range c.Roles {
		if len(c.Roles[i].Upkeep) > 0 {
			return true
		}
	}
	for i := range c.Regions {
		if len(c.Regions[i].ExtraUpkeep) > 0 {
			return true
		}
	}
	return false
}

// hasDraws reports whether any recipe draws on an endowment (the first-level test
// of no_fixed_factor, spec-population.md 9.2 and open question 9).
func hasDraws(c *Config) bool {
	for i := range c.Recipes {
		if len(c.Recipes[i].Draws) > 0 {
			return true
		}
	}
	return false
}

// hasCap reports whether any region row sets a household cap.
func hasCap(m *Population) bool {
	for i := range m.Regions {
		if m.Regions[i].HouseholdCap > 0 {
			return true
		}
	}
	return false
}

// longestWindow is the longest effective welfare window in ticks: the world's, or
// a role's WindowCycles times its longest job (spec-welfare-and-wages.md 2.3.3).
func longestWindow(c *Config) int64 {
	t := c.World.WelfareWindow
	lk := newLookup(c)
	for i := range c.Roles {
		if p, ok := checked.Mul(c.World.WindowCycles, longestDurationAny(c, lk, &c.Roles[i])); ok {
			t = max(t, p)
		}
	}
	return t
}

// popPlanWarnings fills endowment.Plan of each resource from the schedules and
// raises resource_overshoot and resource_unstable_plan when the resources module
// is on (spec-resources.md 8.3.1, FR-RES-044; spec-population.md 5.2). The
// regeneration scale is taken at its default 10 000, since a row's scale is per
// region and the warning is per resource.
func popPlanWarnings(c *Config, rstar, dR, dX int64, add func(string, ...string)) {
	m := c.Modules.Resources
	if m == nil {
		return
	}
	for i := range m.Resources {
		r := &m.Resources[i]
		plan, ok := resourcePlan(c, r)
		if !ok {
			continue
		}
		plan.RStar, plan.DR, plan.DX = rstar, dR, dX
		rb := []string{"modules", "resources", "resources", r.ID}
		if endowment.Overshoot(plan) {
			add("resource_overshoot", rb...)
		}
		if endowment.UnstablePlan(plan) {
			add("resource_unstable_plan", rb...)
		}
	}
}

// resourcePlan returns the part of endowment.Plan that a resource gives: its
// shape, the effective annual rate in basis points (the rate times the mean
// season factor), the plan share and the elasticity of the abundance curve there.
// ok is false for a share or rate outside its bound or an overflow.
func resourcePlan(c *Config, r *Resource) (p endowment.Plan, ok bool) {
	if r.PlanShareBP < 1 || r.PlanShareBP > wholeBP || r.RateBP < 0 || r.RateBP > MaxRegenBP {
		return p, false
	}
	eps, ok := endowment.Elasticity(sortedKnots(r.Abundance), r.PlanShareBP)
	if !ok {
		return p, false
	}
	p = endowment.Plan{ShareBP: r.PlanShareBP, ElasticityBP: eps}
	switch r.Shape {
	case ShapeRefill:
		p.Shape = endowment.Refill
	case ShapeLogistic:
		p.Shape = endowment.Logistic
	default:
		p.Shape = endowment.Constant
	}
	p.RateBPEff = r.RateBP * meanSeasonBP(c, r) / wholeBP
	return p, true
}

// meanSeasonBP is the mean regeneration factor of a resource over the calendar's
// seasons in basis points, truncated: 10 000 when it has no profile or the
// calendar no season, 0 for a season the profile does not list.
func meanSeasonBP(c *Config, r *Resource) int64 {
	if len(r.Seasons) == 0 || len(c.Calendar.Seasons) == 0 {
		return wholeBP
	}
	var sum int64
	for _, s := range c.Calendar.Seasons {
		for _, sr := range r.Seasons {
			if sr.Season == s.ID {
				sum += min(max(sr.BP, 0), MaxSeasonRegenBP)
				break
			}
		}
	}
	return sum / int64(len(c.Calendar.Seasons))
}

// popFill is P7.1(b) for region g (spec-population.md 4.1, D422): it computes the
// module's deaths per agent and births per role from the state as it stands and
// puts them in the view's PopDeaths and PopBirths. Deaths are taken per agent in
// ID order under the group floor; the births of each role in role order under the
// region's cap. It draws nothing and allocates nothing once the carry slice has
// grown (NFR-POP-001).
func (w *World) popFill(g RegionID, v *RegionView) {
	p := w.pop
	for id := len(p.deathCarry); id < len(w.agents); id++ {
		p.deathCarry = append(p.deathCarry, population.Offset(int64(id), p.denom))
	}
	clear(p.cnt)
	clear(p.num)
	var households int64
	for r := range v.Roles {
		p.group[r] = v.Roles[r].Households
		households += v.Roles[r].Households
		v.Roles[r].PopBirths = 0
	}
	deaths := int64(0)
	for i := range v.Agents {
		deaths += w.popAgent(g, v, &v.Agents[i])
	}
	p.births(g, v, households-deaths)
}

// popAgent steps one agent of the view: its exposed households, its death carry
// and the deaths it takes, and the numerator it adds to its role's birth carry.
// It returns the households the agent loses.
func (w *World) popAgent(g RegionID, v *RegionView, ra *RegionAgent) (taken int64) {
	p := w.pop
	ra.PopDeaths = 0
	if ra.External {
		return 0
	}
	c := &p.cnt[ra.Role]
	a := &w.agents[ra.Agent]
	u := w.reads.cost.tick(w.roles.row(ra.Role).upkeepIn(g), w.ex.prices, a.weight)
	ratio, ok := a.history.Ratio(u, w.cfg.Roles[ra.Role].DefaultRatio)
	if !ok {
		c.noRatio += ra.Weight
		return 0
	}
	fed := max(ra.Weight-ra.Unfed, 0)
	c.exposed += fed
	c.unfed += ra.Unfed
	if fed == 0 {
		return 0
	}
	rate, dclamp := p.rate(p.deathK, ratio, p.effDeath[g])
	events := p.step(&p.deathCarry[ra.Agent], fed, rate)
	c.deathsDue += events
	take, cut, held := population.Cut(events, fed, p.group[ra.Role], p.minHouseholds)
	p.group[ra.Role] -= take
	c.deathsCut, c.deathsHeld = c.deathsCut+cut, c.deathsHeld+held
	ra.PopDeaths = take
	rate, bclamp := p.rate(p.birthK, ratio, p.effBirth[g])
	if !v.Roles[ra.Role].Closed {
		p.num[ra.Role] = satAdd(p.num[ra.Role], mustTerm(fed, rate))
	}
	if dclamp || bclamp {
		c.clamped += fed
	}
	return take
}

// births advances the birth carry of every role of region g in role order and
// sets PopBirths under the household cap, counting the births due, the births
// dropped and the households the region holds after the tick's deaths
// (spec-population.md 3.8.2, 4.4.2). Earlier roles' births count against the cap
// of later ones.
func (p *popState) births(g RegionID, v *RegionView, households int64) {
	for r := range v.Roles {
		cell := int(g)*p.roles + r
		events := p.step(&p.birthCarry[cell], 1, p.num[r])
		allowed, dropped := population.Allow(events, households, p.cap[g])
		households += allowed
		v.Roles[r].PopBirths = allowed
		p.cnt[r].birthsDue, p.cnt[r].birthsCapped = events, dropped
	}
}

// step adds exposed * rate to the carry at *carry and returns the whole events it
// holds (spec-population.md 3.8). The births pass the numerator as the rate with
// one exposed household.
func (p *popState) step(carry *int64, exposed, rate int64) int64 {
	events, next, ok := population.Step(*carry, mustTerm(exposed, rate), p.denom)
	if !ok {
		invariant("a population carry overflows after validation")
	}
	*carry = next
	return events
}

// rate evaluates a schedule at a ratio under a scale; the rules cannot refuse a
// validated schedule.
func (p *popState) rate(knots []population.Knot, ratio, scale int64) (rate int64, clamped bool) {
	rate, clamped, ok := population.Rate(knots, ratio, scale)
	if !ok {
		invariant("a population rate is unreadable after validation")
	}
	return rate, clamped
}

// mustTerm is population.Term for values the bounds keep far below int64.
func mustTerm(exposed, rate int64) int64 {
	t, ok := population.Term(exposed, rate)
	if !ok {
		invariant("a population numerator overflows after validation")
	}
	return t
}

// popDeaths appends the module's deaths after the ladder's reductions and
// removals and before the switches (spec-population.md 4.8); an agent that loses
// all its households is marked acted on, so no switch is planned for it.
func (c *decider) popDeaths() {
	for i := range c.v.Agents {
		a := &c.v.Agents[i]
		if a.PopDeaths < 1 {
			continue
		}
		c.d.List = append(c.d.List, Decision{Kind: DecideReduce, Agent: a.Agent, Households: a.PopDeaths, Cause: CausePopulation})
		if a.PopDeaths >= a.Weight {
			c.s.acted[i] = true
		}
	}
}

// popBirths appends the module's births, the last operation of the plan, one
// entry per role in role order, with no entry cap and no viability gate
// (spec-population.md 4.3, 4.8).
func (c *decider) popBirths() {
	for r := range c.v.Roles {
		if n := c.v.Roles[r].PopBirths; n > 0 {
			c.d.List = append(c.d.List, Decision{Kind: DecideEnter, Role: c.v.Roles[r].Role, Households: n, Cause: CauseBirths})
		}
	}
}

// popCause reports whether the executor accepts a reduction with cause c: the
// ladder's are always accepted, the module's only while it is on (N-290).
func (w *World) popCause(c Cause) bool { return w.pop != nil && c == CausePopulation }

// reductionReason is the ledger reason of a weight reduction: starvation for the
// ladder, deaths for the module (spec-population.md 4.7).
func (w *World) reductionReason(c Cause) ledger.Reason {
	if c == CausePopulation {
		return ledger.Deaths
	}
	return ledger.Starvation
}

// entryMoney is the money each household of an entry brings and the ledger reason
// it is paid under (spec-population.md 4.3, D427): the stake for a birth, the
// role's StartMoney for every other entry, by births into a host and by
// start_money into a new agent unless the module's births, which are always
// births. With the module off it is what Stage 1 does.
func (w *World) entryMoney(role RoleID, cause Cause, host bool) (Money, ledger.Reason) {
	start := w.cfg.Roles[role].StartMoney
	switch {
	case w.pop != nil && cause == CauseBirths:
		stake, ok := checked.MulDiv(int64(start), w.pop.stakeBP, wholeBP)
		if !ok {
			invariant("the birth stake overflows after validation")
		}
		return Money(stake), ledger.Births
	case host:
		return start, ledger.Births
	}
	return start, ledger.StartMoney
}

// renewalBP is the effective Renewal: 0 while the module is on, whatever is
// configured (spec-population.md 4.6, a rule constant).
func (w *World) renewalBP() int64 {
	if w.pop != nil {
		return 0
	}
	return w.cfg.World.Renewal
}

// popNoteDeaths counts households removed with the module's cause from a role.
func (w *World) popNoteDeaths(role RoleID, n int64, cause Cause) {
	if w.pop != nil && cause == CausePopulation {
		w.pop.cnt[role].deathsDone += n
	}
}

// popNoteBirths counts the households an entry of births added and the ones the
// pool could not fund.
func (w *World) popNoteBirths(d Decision, funded, unfunded int64) {
	if w.pop != nil && d.Cause == CauseBirths {
		c := &w.pop.cnt[d.Role]
		c.birthsDone += funded
		c.birthsUnfunded += unfunded
	}
}

// setPopulationScale sets the effective scale of a region and returns the value it
// replaced; ok is false when the module is off, the region is unknown or the value
// is outside 0 to MaxBasisPoints. It is state only: the integration module emits
// PopulationScaled (D431). The next tick reads the new value (FR-POP-039).
func (w *World) setPopulationScale(region RegionID, kind ScaleKind, bp int64) (before int64, ok bool) {
	p := w.pop
	if p == nil || int(region) >= len(p.effBirth) || region < 0 || bp < 0 || bp > MaxBasisPoints {
		return 0, false
	}
	switch kind {
	case ScaleBirth:
		before, p.effBirth[region] = p.effBirth[region], bp
	case ScaleDeath:
		before, p.effDeath[region] = p.effDeath[region], bp
	default:
		return 0, false
	}
	return before, true
}
