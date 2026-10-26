package economysim

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/Whealinghold/economy-sim/internal/checked"
	"github.com/Whealinghold/economy-sim/internal/steady"
)

// This file holds the root's side of the viability rules and the solver
// (spec-viability.md): the conversion of a canonical Config to the primitives of
// internal/steady, inline at the call as architecture.md 2.9 says, the virtual
// start offsets of a job (8.5.2), and the exported surface of SolveSteadyState
// (the options, the result and the report) with the solves it runs. The checks
// that use the conversion are in viability_check.go, the warm-start table made
// from the solves in viability_table.go. Nothing here draws a random number,
// logs or reads a map in an order that reaches a result.

// seasonRun is a stretch of consecutive ticks of the year in one season, -1
// when the calendar has none.
type seasonRun struct {
	season int
	length int64
}

// seasonRuns splits the year of a calendar into stretches of one season, read
// tick by tick with Calendar.SeasonAt and no other rule (FR-VIA-053). A calendar
// with no ticks gives none.
func seasonRuns(cal Calendar) []seasonRun {
	var out []seasonRun
	for t := range cal.TicksPerYear() {
		s := cal.SeasonAt(t)
		if n := len(out); n > 0 && out[n-1].season == s {
			out[n-1].length++
			continue
		}
		out = append(out, seasonRun{season: s, length: 1})
	}
	return out
}

// viaActivity is an activity that exists (J above 0) in a region
// (spec-viability.md 3.1). act has Region 0: the programme that uses it sets the
// region index of its own model.
type viaActivity struct {
	region  RegionID
	recipe  RecipeID
	runs    []steady.Run
	count   steady.StartCount
	act     steady.Activity
	allYear bool // the start is open on every tick of the year
}

// viaModel is a canonical, clean configuration in the form the checks read. It
// is built once per check and holds no state between calls.
type viaModel struct {
	c       *Config
	rt      recipeTable
	roles   roleTable
	yt      yieldTable
	year    int64
	seasons []seasonRun
	// maxCells, maxPivots and maxSteps are the size gate, the pivot cap and the
	// step bound of the cycle-gain enumeration (D126): the bounds of bounds.go,
	// held here so a test can lower them and reach the skipped parts with a small
	// configuration.
	maxCells, maxPivots, maxSteps int64
	// withHouseholds lists, ascending, the regions with at least one household:
	// the only regions that have a programme or an activity (a region with no
	// household has no labour).
	withHouseholds []RegionID
	households     []int64   // by region
	upkeep         [][]int64 // by region, by commodity: D(c, rho)
	acts           [][]viaActivity
	// listed[r][k] says that a role with a group in region r lists recipe k.
	listed [][]bool
	// expected[r] and targets[r] are the sums over the groups of region r of
	// households times ExpectedWorkTicks and times TargetRatio, the numerators of
	// the household-weighted means of the nominal warnings; sumsOK[r] is false
	// when a sum does not fit int64, where those warnings are not run.
	expected, targets []int64
	sumsOK            []bool
}

// newViaModel builds the model of a canonical configuration that passed phases
// 1 to 5 of Validate. It builds the recipe, role and yield tables, which panic
// on anything else.
func newViaModel(c *Config) *viaModel {
	m := &viaModel{
		c:          c,
		rt:         newRecipeTable(c),
		roles:      newRoleTable(c),
		yt:         newYieldTable(c),
		year:       c.Calendar.TicksPerYear(),
		seasons:    seasonRuns(c.Calendar),
		maxCells:   MaxViabilityCells,
		maxPivots:  MaxViabilityPivots,
		maxSteps:   MaxViabilityCells,
		households: make([]int64, len(c.Regions)),
		upkeep:     make([][]int64, len(c.Regions)),
		acts:       make([][]viaActivity, len(c.Regions)),
		listed:     make([][]bool, len(c.Regions)),
		expected:   make([]int64, len(c.Regions)),
		targets:    make([]int64, len(c.Regions)),
		sumsOK:     make([]bool, len(c.Regions)),
	}
	m.addGroups()
	for r := range c.Regions {
		if m.households[r] > 0 {
			m.withHouseholds = append(m.withHouseholds, RegionID(r))
			m.acts[r] = m.activitiesOf(RegionID(r))
		}
	}
	return m
}

// addGroups sums the households of each region, their upkeep per commodity and
// the recipes the roles that live there list. The sums are at most 10^5 groups
// of 10^9 households of 2 000 units (the base upkeep and the region's extra, each
// within MaxQuantity / MaxWeight), 2 * 10^17, inside int64.
func (m *viaModel) addGroups() {
	c := m.c
	for i := range c.Regions {
		m.upkeep[i] = make([]int64, len(c.Commodities))
		m.listed[i] = make([]bool, len(c.Recipes))
		m.sumsOK[i] = true
	}
	for _, g := range c.Groups {
		r, _ := c.regionID(g.Region) // resolved: phase 3 passed
		ro, _ := c.roleID(g.Role)
		row := m.roles.row(ro)
		m.households[r] += g.Households
		for _, it := range row.upkeepIn(r) {
			m.upkeep[r][it.commodity] += int64(it.quantity) * g.Households
		}
		for _, k := range row.recipes {
			m.listed[r][k] = true
		}
		m.addMeans(r, g.Households, &c.Roles[ro])
	}
}

// addMeans adds a group to the weighted sums of its region, with checked
// arithmetic: a sum that does not fit switches the nominal warnings off for the
// region.
func (m *viaModel) addMeans(r RegionID, households int64, ro *Role) {
	e, ok1 := checked.Mul(households, ro.ExpectedWorkTicks)
	t, ok2 := checked.Mul(households, ro.TargetRatio)
	se, ok3 := checked.Add(m.expected[r], e)
	st, ok4 := checked.Add(m.targets[r], t)
	if !(ok1 && ok2 && ok3 && ok4) {
		m.sumsOK[r] = false
		return
	}
	m.expected[r], m.targets[r] = se, st
}

// activitiesOf returns the activities that exist in a region, in recipe order.
func (m *viaModel) activitiesOf(region RegionID) []viaActivity {
	var out []viaActivity
	for k := range m.c.Recipes {
		if a, ok := m.activity(region, RecipeID(k)); ok {
			out = append(out, a)
		}
	}
	return out
}

// runsFor returns the year of a pair as runs of one yield factor: the effective
// factor of the recipe in the region in each season (spec-production.md 6.3, the
// shock and the other terms neutral), adjacent equal runs merged.
func (m *viaModel) runsFor(region RegionID, recipe RecipeID) []steady.Run {
	sited := m.c.Recipes[recipe].Sited
	var out []steady.Run
	for _, sr := range m.seasons {
		// An effective factor that does not fit int64 is 0 here; the bounds make
		// it unreachable.
		f, _, _ := m.yt.effective(&yieldShocks{}, region, recipe, sr.season, sited)
		if n := len(out); n > 0 && out[n-1].FactorBP == f {
			out[n-1].Length += sr.length
			continue
		}
		out = append(out, steady.Run{Length: sr.length, FactorBP: f})
	}
	return out
}

// activity builds the activity of a pair, or reports that it does not exist
// (J = 0, spec-viability.md 3.1).
func (m *viaModel) activity(region RegionID, recipe RecipeID) (viaActivity, bool) {
	rc := &m.c.Recipes[recipe]
	runs := m.runsFor(region, recipe)
	count := steady.Starts(rc.Duration, runs)
	if count.Starts < 1 {
		return viaActivity{}, false
	}
	row := m.rt.row(recipe)
	a := steady.Activity{Duration: rc.Duration, Starts: count.Starts, FactorSum: count.FactorSum}
	for _, in := range row.inputs {
		a.Inputs = append(a.Inputs, steady.Item{Commodity: int(in.commodity), Qty: int64(in.quantity)})
	}
	for _, o := range row.outputs {
		a.Outputs = append(a.Outputs, steady.Item{Commodity: int(o.commodity), Qty: int64(o.quantity), Discard: o.discard})
	}
	for _, t := range row.tools {
		a.Tools = append(a.Tools, steady.Tool{Commodity: int(t.commodity), Wear: t.wearPerUse, Durability: m.c.Commodities[t.commodity].Durability})
	}
	return viaActivity{region: region, recipe: recipe, runs: runs, count: count, act: a, allYear: allOpen(runs)}, true
}

// allOpen reports that every tick of the year has a positive factor.
func allOpen(runs []steady.Run) bool {
	for _, r := range runs {
		if r.FactorBP <= 0 {
			return false
		}
	}
	return len(runs) > 0
}

// virtualStartOffsets returns the virtual start offsets V of a recipe in a region
// (spec-viability.md 8.5.2, FR-VIA-053): the offsets o in 0 .. d-1, ascending,
// such that the effective yield factor at the virtual tick -o is positive, the
// season of that tick being Calendar.SeasonAt(-o) and no other rule. An empty
// result means the job cannot be placed (ErrWarmStartJob, FR-VIA-033). c must be
// canonical and clean.
func virtualStartOffsets(c *Config, yt *yieldTable, region RegionID, recipe RecipeID) []int64 {
	rc := &c.Recipes[recipe]
	var out []int64
	for o := range rc.Duration {
		f, _, _ := yt.effective(&yieldShocks{}, region, recipe, c.Calendar.SeasonAt(-o), rc.Sited)
		if f > 0 {
			out = append(out, o)
		}
	}
	return out
}

// noVirtualStart reports that the set of virtual start offsets of a recipe is
// empty, from the runs of its year, in time proportional to the runs rather than
// to the duration (the same answer as len(virtualStartOffsets) == 0). The ticks
// -o for o in 0 .. d-1 are tick 0 and the last d-1 ticks of the year, or the
// whole year when d reaches it.
func noVirtualStart(runs []steady.Run, year, duration int64) bool {
	lo := max(year-duration+1, 0) // the first tick of the tail window
	var at int64
	for i, r := range runs {
		at += r.Length
		if r.FactorBP <= 0 {
			continue
		}
		if i == 0 || at > lo { // tick 0 is in the first run
			return false
		}
	}
	return true
}

// SolveOption changes one call of SolveSteadyState (spec-viability.md 7.1, 11).
// An option is not a Config field and is not hashed: it shapes the table, which
// is then data. It is a separate type from the world's Option.
type SolveOption func(*solveOptions) error

// solveOptions is the state the options change.
type solveOptions struct {
	stockCover, moneyBuffer, residualWarn, maxPivots int64
	freeTrade                                        bool
}

// newSolveOptions applies the options in order over the defaults: a nil option
// is ignored and the last of a kind wins. A value outside its bound is
// ErrOutOfBounds, before any work.
func newSolveOptions(opts []SolveOption) (solveOptions, error) {
	o := solveOptions{
		stockCover: defaultStockCoverTicks, moneyBuffer: defaultMoneyBufferTicks,
		residualWarn: defaultResidualWarnBP, maxPivots: MaxViabilityPivots,
	}
	for _, f := range opts {
		if f == nil {
			continue
		}
		if err := f(&o); err != nil {
			return solveOptions{}, err
		}
	}
	return o, nil
}

// boundedOption returns the option that sets one integer field within [lo, hi].
func boundedOption(name string, n, lo, hi int64, set func(*solveOptions, int64)) SolveOption {
	return func(o *solveOptions) error {
		if n < lo || n > hi {
			bound := hi
			if n < lo {
				bound = lo
			}
			return &ValidationError{Path: []string{"solve_options", name}, Rule: ErrOutOfBounds, Value: i64(n), Bound: i64(bound)}
		}
		set(o, n)
		return nil
	}
}

// WithStockCoverTicks sets the ticks of upkeep a household holds at the start
// (spec-viability.md 8.4), 0 to 1 000; the default is 5.
func WithStockCoverTicks(n int64) SolveOption {
	return boundedOption("stock_cover_ticks", n, 0, maxStockCoverTicks, func(o *solveOptions, v int64) { o.stockCover = v })
}

// WithMoneyBufferTicks sets the ticks of upkeep of money held beyond the job
// (spec-viability.md 8.3), 0 to 10 000; the default is 10.
func WithMoneyBufferTicks(n int64) SolveOption {
	return boundedOption("money_buffer_ticks", n, 0, maxMoneyBufferTicks, func(o *solveOptions, v int64) { o.moneyBuffer = v })
}

// WithResidualWarnBP sets the residual, in basis points, above which the report
// lists an activity (spec-viability.md 6.5), 0 to 100 000; the default is 1 000.
func WithResidualWarnBP(n int64) SolveOption {
	return boundedOption("residual_warn_bp", n, 0, maxResidualWarnBP, func(o *solveOptions, v int64) { o.residualWarn = v })
}

// WithMaxPivots sets the pivots a programme may use before the solve returns
// ErrSolveLimit (spec-viability.md 5.4), 1 to MaxViabilityPivots, which is the
// default.
func WithMaxPivots(n int64) SolveOption {
	return boundedOption("max_pivots", n, 1, MaxViabilityPivots, func(o *solveOptions, v int64) { o.maxPivots = v })
}

// WithFreeTrade solves all the regions with a group as one pooled programme with
// one price scale, the complete set of zero-cost lanes, instead of each region
// alone in autarky (spec-viability.md 7.1, 9.4). Stage 1 realises only this lane
// set: Lane, Lanes and MaxLanes come with the transport module.
func WithFreeTrade() SolveOption {
	return func(o *solveOptions) error { o.freeTrade = true; return nil }
}

// SteadyState is the result of SolveSteadyState: the programmes solved, the
// warm-start table made from them and the report (spec-viability.md 7.1). The
// table passes Validate. No exact flow is exported: the table and the census
// are the only flow-derived outputs (architecture.md 2.8).
type SteadyState struct {
	// Programmes are the programmes solved, by their first region.
	Programmes []ProgrammeResult
	// Table is the warm-start table in canonical order; absent lists are nil.
	Table WarmStart
	// Report holds the diagnostics of spec-viability.md 10.3.
	Report Report
}

// ProgrammeResult is one programme solved: one region in autarky, all the
// regions of the realm with a group when pooled.
type ProgrammeResult struct {
	// Regions are the IDs of the regions of the programme.
	Regions []string
	// SurplusBP is floor(lambda* * 10 000) at full labour.
	SurplusBP int64
	// LabourShareBP is ceil(10 000 * 10 000 / SurplusBP) when HasLabourShare.
	LabourShareBP int64
	// HasLabourShare is false when a capacity row is tight or SurplusBP is 0
	// (FR-VIA-012).
	HasLabourShare bool
	// NominalSurplusBP is SurplusBP at the expected work share, 0 when the sums
	// of the nominal programme do not fit int64.
	NominalSurplusBP int64
	// Pivots is the number of pivots of the full-labour programme.
	Pivots int64
}

// viaProgramme is one programme solved. out holds the flows (full labour, lambda
// scaled to 1, D120); pr holds the prices and wages, which are out's unless the
// nominal fallback rebuilt them (D131).
type viaProgramme struct {
	regions     []RegionID
	md          steady.Model
	out, pr     steady.Outcome
	nominalBP   int64
	nominalSeed []int // indexes into regions
	wants       bool  // the model carries the fixed term of the wants
	base        []int // the first column of each region's activities
}

// viaSolve is the state of one SolveSteadyState call.
type viaSolve struct {
	m     *viaModel
	o     solveOptions
	rep   Report
	progs []*viaProgramme
	aux   []*viaProgramme // autarky programmes beside a pooled one
	ws    WarmStart
}

// SolveSteadyState solves the reference programme for cfg and returns the
// steady state and the warm-start table (spec-viability.md 7.1). It is pure: it
// reads cfg, draws no random number, writes no log and starts no goroutine.
// NewWorld never calls it.
//
// cfg is validated first, on a copy whose WarmStart is nil, so a stale table
// does not make the solve that replaces it fail; the error of a configuration
// that is not viable wraps ErrNotViable. With no WithFreeTrade each region with
// a group is solved alone. ErrNoUpkeep is returned when no group of any region
// has a basket to be viable for, ErrSolveLimit (a *SolveError) at the pivot cap
// or the cell gate, ErrSolveInconsistent when the exact certificate fails (a
// defect), and the wrapped ctx.Err() when ctx is cancelled between pivots. No
// partial result is returned with any error.
func SolveSteadyState(ctx context.Context, cfg Config, opts ...SolveOption) (SteadyState, error) {
	o, err := newSolveOptions(opts)
	if err != nil {
		return SteadyState{}, err
	}
	cold := cfg
	cold.WarmStart = nil
	if err := cold.Validate(); err != nil {
		return SteadyState{}, err
	}
	cc := cold.canonical()
	s := &viaSolve{m: newViaModel(&cc), o: o}
	s.m.maxPivots = o.maxPivots
	if err := s.solve(ctx); err != nil {
		return SteadyState{}, err
	}
	if err := s.build(); err != nil {
		return SteadyState{}, err
	}
	return s.finish()
}

// sets returns the region sets to solve, one region each in autarky and the
// whole realm when pooled, and lists the regions that have no programme: an
// empty basket, or one of unsourced Raw markers (D130, JC5).
func (s *viaSolve) sets() [][]RegionID {
	var sets [][]RegionID
	groups := [][]RegionID{s.m.withHouseholds}
	if !s.o.freeTrade {
		groups = groups[:0]
		for _, r := range s.m.withHouseholds {
			groups = append(groups, []RegionID{r})
		}
	}
	for _, g := range groups {
		if len(g) == 0 {
			continue
		}
		if s.m.noProgramme(g) {
			s.unprogrammed(g)
			continue
		}
		sets = append(sets, g)
	}
	return sets
}

// unprogrammed lists regions that have no programme: no wage and, for each
// basket good, no price.
func (s *viaSolve) unprogrammed(regions []RegionID) {
	for _, r := range regions {
		s.rep.UnpricedWages = append(s.rep.UnpricedWages, s.m.c.Regions[r].ID)
		for ci := range s.m.c.Commodities {
			if s.m.upkeep[r][ci] > 0 {
				s.rep.UnpricedCommodities = append(s.rep.UnpricedCommodities, RegionCommodity{s.m.c.Regions[r].ID, s.m.c.Commodities[ci].ID})
			}
		}
	}
}

// solve runs the programmes of the call: each set at full labour, the nominal
// programme beside it, and with free trade the autarky programmes of the
// regions for the comparison of 9.2.
func (s *viaSolve) solve(ctx context.Context) error {
	sets := s.sets()
	if len(sets) == 0 {
		return ErrNoUpkeep
	}
	for _, set := range sets {
		p, err := s.programme(ctx, set)
		if err != nil {
			return err
		}
		s.progs = append(s.progs, p)
		if s.o.freeTrade {
			err = s.autarkies(ctx, set)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// autarkies solves each region of a pooled set alone, at full labour only.
func (s *viaSolve) autarkies(ctx context.Context, set []RegionID) error {
	for _, r := range set {
		if s.m.noProgramme([]RegionID{r}) {
			continue
		}
		p, err := s.full(ctx, []RegionID{r})
		if err != nil {
			return err
		}
		s.aux = append(s.aux, p)
	}
	return nil
}

// full solves a set of regions at full labour.
func (s *viaSolve) full(ctx context.Context, regions []RegionID) (*viaProgramme, error) {
	md, out, wants, err := s.solveAt(ctx, regions, steady.Ratio{Num: 1, Den: 1}, true)
	if err != nil {
		return nil, err
	}
	p := &viaProgramme{regions: regions, md: md, out: out, pr: out, wants: wants}
	var at int
	for _, r := range regions {
		p.base = append(p.base, at)
		at += len(s.m.acts[r])
	}
	return p, nil
}

// programme solves a set at full labour and at the expected work share, and
// rebuilds the prices from the nominal programme where the full-labour wage is 0
// against a tight capacity row (D131, FR-VIA-058).
func (s *viaSolve) programme(ctx context.Context, regions []RegionID) (*viaProgramme, error) {
	p, err := s.full(ctx, regions)
	if err != nil {
		return nil, err
	}
	phi, ok := s.m.nominalPhi(regions)
	if !ok {
		return p, nil
	}
	_, nom, _, err := s.solveAt(ctx, regions, phi, p.wants)
	if err != nil {
		return nil, err
	}
	p.nominalBP = nom.SurplusBP
	for k := range regions {
		if s.capacityScarce(p, k) {
			p.nominalSeed = append(p.nominalSeed, k)
		}
	}
	if len(p.nominalSeed) > 0 {
		p.pr = nom
	}
	return p, nil
}

// solveAt solves the regions at the work share phi, with the fixed term of the
// wants when the module is on and wants is set. A wants programme that is
// infeasible falls back to the upkeep-only one (FR-VIA-057); the bool says which
// was solved.
func (s *viaSolve) solveAt(ctx context.Context, regions []RegionID, phi steady.Ratio, wants bool) (steady.Model, steady.Outcome, bool, error) {
	if wants && s.m.c.Modules.Demand != nil {
		md, err := s.m.wantsModel(regions, phi)
		if err != nil {
			return md, steady.Outcome{}, false, err
		}
		out, err := s.m.solveStrict(ctx, md)
		if !errors.Is(err, errWantsInfeasible) {
			return md, out, err == nil, err
		}
	}
	md := s.m.model(regions, phi)
	out, err := s.m.solveStrict(ctx, md)
	if errors.Is(err, errWantsInfeasible) { // the sentinel is internal: the upkeep-only programme has no fallback
		err = fmt.Errorf("%w: the programme is infeasible", ErrSolveInconsistent)
	}
	return md, out, false, err
}

// wantsModel is the programme of the regions with the fixed term of the wants.
func (m *viaModel) wantsModel(regions []RegionID, phi steady.Ratio) (steady.Model, error) {
	md := m.model(regions, phi)
	fixed, err := wantsTerm(m.c, regions)
	if errors.Is(err, steady.ErrOverflow) {
		return md, fmt.Errorf("%w: the wants term is out of int64 range", ErrOverflow)
	}
	md.Fixed = fixed
	return md, err
}

// wantsTerm (the term of the demand module in the programme, FR-VIA-055) returns the exact fixed term W(c) of the goods rows of a programme
// over the regions (spec-viability.md 3.2): for each group in them, each want of
// its role's ladder with a taste row of nonzero scale, the households times the
// taste and region scales in basis points times the quantity over the period,
// split among the items by weight, summed by commodity. It is nil when the
// module is off. c is canonical and valid, so every reference resolves.
func wantsTerm(c *Config, regions []RegionID) ([]steady.Ratio, error) {
	ix := newDemandIndex(c)
	if ix == nil {
		return nil, nil
	}
	var rows []steady.WantRow
	for _, g := range c.Groups {
		r, _ := c.regionID(g.Region)
		if !slices.Contains(regions, r) {
			continue
		}
		for _, t := range ix.tastes[g.Role] {
			w := ix.want[t.Want]
			if w == nil || t.Scale <= 0 {
				continue
			}
			row := steady.WantRow{Households: g.Households, Taste: t.Scale, Region: ix.regionScale(g.Region, w.ID), Quantity: w.Quantity, Period: w.Period}
			for _, it := range w.Items {
				k, _ := c.commodityID(it.Commodity)
				row.Items = append(row.Items, steady.WantWeight{Commodity: int(k), Weight: it.Weight})
			}
			rows = append(rows, row)
		}
	}
	return steady.FixedTerm(len(c.Commodities), rows)
}

// errWantsInfeasible says the programme with the wants has no feasible point at
// any surplus factor (the first phase of the solver ended above 0).
var errWantsInfeasible = errors.New("the wants programme is infeasible")

// capacityScarce reports that region k of the programme has a labour dual of 0
// while a capacity row of its activities has a positive dual.
func (s *viaSolve) capacityScarce(p *viaProgramme, k int) bool {
	if p.out.Wage[k].Sign() != 0 {
		return false
	}
	n := len(s.m.acts[p.regions[k]])
	return slices.ContainsFunc(p.out.Rent[p.base[k]:p.base[k]+n], func(f steady.Frac) bool { return f.Sign() > 0 })
}

// nominalPhi returns the available work share of the regions, the
// household-weighted mean expected work over the year (spec-viability.md 3.2).
// ok is false when the sums do not fit int64.
func (m *viaModel) nominalPhi(regions []RegionID) (steady.Ratio, bool) {
	var exp, hh int64
	for _, r := range regions {
		e, ok1 := checked.Add(exp, m.expected[r])
		h, ok2 := checked.Add(hh, m.households[r])
		if !m.sumsOK[r] || !ok1 || !ok2 {
			return steady.Ratio{}, false
		}
		exp, hh = e, h
	}
	den, ok := checked.Mul(m.year, hh)
	return steady.Ratio{Num: exp, Den: den}, ok
}

// solveStrict solves a model for SolveSteadyState: over the cell gate, or at the
// pivot cap, it is ErrSolveLimit with the counts, never a skipped part.
func (m *viaModel) solveStrict(ctx context.Context, md steady.Model) (steady.Outcome, error) {
	if err := ctx.Err(); err != nil {
		return steady.Outcome{}, fmt.Errorf("solve: %w", err)
	}
	rows, cols := steady.Size(md)
	if cells, ok := checked.Mul(int64(rows), int64(cols)); !ok || cells > m.maxCells {
		return steady.Outcome{}, &SolveError{Rows: rows, Columns: cols}
	}
	out, err := steady.SolveModel(ctx, md, m.maxPivots)
	return out, mapSolveErr(err)
}

// mapSolveErr turns an error of internal/steady into the root's sentinels.
func mapSolveErr(err error) error {
	var le *steady.LimitError
	switch {
	case err == nil:
		return nil
	case errors.As(err, &le):
		return &SolveError{Pivots: le.Pivots, Rows: le.Rows, Columns: le.Cols}
	case errors.Is(err, steady.ErrInfeasible):
		return errWantsInfeasible
	case errors.Is(err, steady.ErrUnbounded):
		return ErrNoUpkeep
	case errors.Is(err, steady.ErrOverflow):
		return fmt.Errorf("%w: solver value out of int64 range", ErrOverflow)
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return err
	case errors.Is(err, steady.ErrInconsistent):
		return ErrSolveInconsistent
	}
	return fmt.Errorf("%w: %v", ErrSolveInconsistent, err)
}
