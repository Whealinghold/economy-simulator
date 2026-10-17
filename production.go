package economysim

import (
	"cmp"
	"fmt"
	"slices"

	"github.com/Whealinghold/economy-sim/internal/checked"
	"github.com/Whealinghold/economy-sim/internal/ledger"
	"github.com/Whealinghold/economy-sim/internal/production"
)

// This file holds the configuration-dependent state of production that the
// steps read (spec-production.md 5, 6; plan-phase-1-5.md Phase 5): the yield
// table, the yield-shock slot, the job type and the residue layout, the
// admission check and the scratch the steps convert into. It emits nothing,
// logs nothing and draws no random number. The tables are derived at
// construction and are not state: they are not hashed and not checkpointed.
// The shock slot is the one piece of state here; it is saved by the
// checkpoint row.

// yieldRow is the profile of one (region, recipe) pair with its seasons as
// calendar indexes.
type yieldRow struct {
	region  RegionID
	recipe  RecipeID
	profile production.Profile
}

// seasonIndex maps a season ID to its index in Calendar.Seasons.
type seasonIndex struct {
	id    string
	index int
}

// yieldTable holds one row per yield profile in (region, recipe) order, the
// canonical order of Config.Yields. Seasons of a profile are ascending by
// calendar index, not by ID (spec-production.md 6.1, 6.2).
type yieldTable struct {
	rows    []yieldRow
	seasons []seasonIndex // ascending by id, searched by binary search
}

// newYieldTable builds the table from a canonical configuration that passed
// Validate. A reference that does not resolve is a programmer error and
// panics with a message that says so. The season index table is built from
// the calendar, because Config.Yields lists a profile's seasons by ID while
// the calendar, and Moment.Season, order them by declaration.
func newYieldTable(c *Config) yieldTable {
	t := yieldTable{
		rows:    make([]yieldRow, len(c.Yields)),
		seasons: make([]seasonIndex, len(c.Calendar.Seasons)),
	}
	for i, s := range c.Calendar.Seasons {
		t.seasons[i] = seasonIndex{id: s.ID, index: i}
	}
	slices.SortFunc(t.seasons, func(a, b seasonIndex) int { return cmp.Compare(a.id, b.id) })
	for i := range c.Yields {
		y := &c.Yields[i]
		region, ok := c.regionID(y.Region)
		if !ok {
			panic("economysim: invariant violated: yield profile names unknown region " + y.Region + " after validation")
		}
		recipe, ok := c.recipeID(y.Recipe)
		if !ok {
			panic("economysim: invariant violated: yield profile names unknown recipe " + y.Recipe + " after validation")
		}
		row := &t.rows[i]
		row.region, row.recipe = region, recipe
		row.profile = production.Profile{HasAll: y.HasAll, All: y.All}
		for _, s := range y.Seasons {
			j, ok := slices.BinarySearchFunc(t.seasons, s.Season, func(e seasonIndex, id string) int { return cmp.Compare(e.id, id) })
			if !ok {
				panic("economysim: invariant violated: yield profile names unknown season " + s.Season + " after validation")
			}
			row.profile.Seasons = append(row.profile.Seasons, production.SeasonBP{Season: t.seasons[j].index, BP: s.BP})
		}
		// Calendar indexes are unique, so the order is total.
		slices.SortFunc(row.profile.Seasons, func(a, b production.SeasonBP) int { return cmp.Compare(a.Season, b.Season) })
	}
	return t
}

// site returns the site factor of a recipe in a region at a season (a
// calendar index, -1 for none) and whether a profile row exists for the pair.
// sited is the recipe's Sited flag. It allocates nothing.
func (t *yieldTable) site(region RegionID, recipe RecipeID, season int, sited bool) (factor int64, has bool) {
	i, has := slices.BinarySearchFunc(t.rows, [2]int{int(region), int(recipe)}, func(r yieldRow, k [2]int) int {
		return cmp.Or(cmp.Compare(int(r.region), k[0]), cmp.Compare(int(r.recipe), k[1]))
	})
	if !has {
		return production.Site(nil, sited, season), false
	}
	return production.Site(&t.rows[i].profile, sited, season), true
}

// shockEntry is a yield shock other than neutral.
type shockEntry struct {
	region RegionID
	recipe RecipeID
	bp     int64
}

// yieldShocks is the shock slot (spec-production.md 6.8): a sparse list sorted
// by (region, recipe) of the factors other than 10 000, which is the value of
// an absent key. The zero value holds no shock.
type yieldShocks struct {
	entries []shockEntry
}

func (s *yieldShocks) find(region RegionID, recipe RecipeID) (int, bool) {
	return slices.BinarySearchFunc(s.entries, [2]int{int(region), int(recipe)}, func(e shockEntry, k [2]int) int {
		return cmp.Or(cmp.Compare(int(e.region), k[0]), cmp.Compare(int(e.recipe), k[1]))
	})
}

// factor returns the shock of a (region, recipe) pair in basis points: 10 000
// when none is set. It allocates nothing.
func (s *yieldShocks) factor(region RegionID, recipe RecipeID) int64 {
	if i, ok := s.find(region, recipe); ok {
		return s.entries[i].bp
	}
	return production.BP
}

// set sets the shock of a pair. A value outside [0, MaxYieldBP] is refused
// with ErrOutOfBounds and changes nothing. A value of 10 000 removes the key.
func (s *yieldShocks) set(region RegionID, recipe RecipeID, bp int64) error {
	if bp < 0 || bp > MaxYieldBP {
		return fmt.Errorf("yield shock %d outside [0, %d]: %w", bp, MaxYieldBP, ErrOutOfBounds)
	}
	i, ok := s.find(region, recipe)
	switch {
	case bp == production.BP && ok:
		s.entries = slices.Delete(s.entries, i, i+1)
	case bp == production.BP:
	case ok:
		s.entries[i].bp = bp
	default:
		s.entries = slices.Insert(s.entries, i, shockEntry{region: region, recipe: recipe, bp: bp})
	}
	return nil
}

// ordered returns the shocks in (region, recipe) order for the checkpoint.
// The caller must not change the slice.
func (s *yieldShocks) ordered() []shockEntry { return s.entries }

// effective returns the effective factor of a recipe in a region at a season
// (spec-production.md 6.3) and whether a profile row exists for the pair: the
// site factor, the shock, then the technology, trainee and abundance terms,
// which are neutral (10 000) in the slice. ok is false when a step does not
// fit int64, which validation of the bounds makes unreachable.
func (t *yieldTable) effective(sh *yieldShocks, region RegionID, recipe RecipeID, season int, sited bool) (e int64, has, ok bool) {
	site, has := t.site(region, recipe, season, sited)
	e, ok = production.Effective(site, sh.factor(region, recipe), production.BP, production.BP, production.BP)
	return e, has, ok
}

// yieldRefusal is the yield pair of the count (spec-production.md 4.2.2): the
// cap of the first limit and, when it is 0, the reason. no_site is a sited
// recipe with no profile row; zero_yield is any other effective factor of 0.
func yieldRefusal(sited, has bool, effective int64) (limit int64, reason Reason) {
	limit, noSite := production.YieldLimit(sited, has, effective)
	switch {
	case noSite:
		return limit, ReasonNoSite
	case limit == 0:
		return limit, ReasonZeroYield
	}
	return limit, ""
}

// job is the work in progress of one agent (spec-production.md 5.1). The
// start tick is not stored: a job started in s completes in s + duration - 1,
// so it is derived at completion. yield is the effective factor fixed at the
// start. pending holds one quantity per output item in output order, discards
// included, and is reused by the next start of the same agent.
type job struct {
	recipe     RecipeID
	remaining  int64
	executions int64
	principal  AgentID
	yield      int64
	pending    []int64
}

// residueLen returns the number of residues of an agent whose role lists the
// given recipes (ascending): one per output item of each recipe, discards
// included (spec-production.md 15). The layout is a function of the role and
// the recipe table and is never saved.
func residueLen(roleRecipes []RecipeID, rt *recipeTable) int {
	n := 0
	for _, r := range roleRecipes {
		n += len(rt.row(r).outputs)
	}
	return n
}

// residueOffset returns the index of the first residue of a recipe in the
// layout of the role's ascending recipe list, and false when the recipe is not
// on the list.
func residueOffset(roleRecipes []RecipeID, rt *recipeTable, recipe RecipeID) (offset int, ok bool) {
	i, ok := slices.BinarySearch(roleRecipes, recipe)
	if !ok {
		return 0, false
	}
	return residueLen(roleRecipes[:i], rt), true
}

// setResidues sets every residue of an agent to v, the value of the creation
// draw (spec-production.md 6.6.2, FR-PRO-026). v is in [0, 10 000).
func setResidues(residues []int64, v int64) {
	for i := range residues {
		residues[i] = v
	}
}

// admit is the first check of a request (spec-production.md 4.1, 4.2): a
// recipe not in the role's ascending recipe list is refused with
// ReasonNotOnRole. It returns "" for an admitted request. The world calls it
// and logs the refusal.
func admit(roleRecipes []RecipeID, recipe RecipeID) Reason {
	if _, ok := slices.BinarySearch(roleRecipes, recipe); !ok {
		return ReasonNotOnRole
	}
	return ""
}

// productionScratch is the one home of the buffers the production steps
// convert a recipe row into (architecture.md 2.9). It reads the configuration
// and the recipe table and owns the slices; the conversion is written inline
// in each step. One per world: the buffers are mutable and never shared. It is
// not saved and not hashed.
type productionScratch struct {
	cfg     *Config
	recipes *recipeTable

	terms   []production.Term
	wears   []production.Wear
	inputs  []ledger.Line
	outputs []ledger.Line
	limits  []production.Limit
	anchor  []production.Recipe
	row     []int64

	// The cost anchor step (spec-production.md 9.3): the flat argument
	// slices the entries of anchor point into, and the table it fills.
	aTerms []production.Term
	aWears []production.Wear
	aOuts  []production.Out
	costs  []int64 // region x commodity

	// The results of the steps, valid until the next call of the same step.
	wear     []wearFact
	spent    []Amount
	newRes   []int64
	created  []int64
	disposed []int64
	overflow []int64
}

// newProductionScratch sizes the buffers once from the largest recipe and the
// recipe and commodity counts. It converts nothing. The steps reuse each
// buffer with [:0] and never grow it.
func newProductionScratch(cfg *Config, rt *recipeTable) *productionScratch {
	var items, tools, outs, allItems, allTools, allOuts int
	for i := range rt.rows {
		r := &rt.rows[i]
		allItems += len(r.inputs)
		allTools += len(r.tools)
		allOuts += len(r.outputs)
		items = max(items, len(r.inputs))
		tools = max(tools, len(r.tools))
		outs = max(outs, len(r.outputs))
	}
	return &productionScratch{
		cfg:     cfg,
		recipes: rt,
		terms:   make([]production.Term, 0, items),
		wears:   make([]production.Wear, 0, tools),
		inputs:  make([]ledger.Line, 0, items),
		outputs: make([]ledger.Line, 0, outs),
		// The yield pair, a limit per tool and one per input.
		limits: make([]production.Limit, 0, 1+tools+items),
		anchor: make([]production.Recipe, 0, len(rt.rows)),
		row:    make([]int64, len(cfg.Commodities)),
		aTerms: make([]production.Term, 0, allItems),
		aWears: make([]production.Wear, 0, allTools),
		aOuts:  make([]production.Out, 0, allOuts),
		costs:  make([]int64, len(cfg.Regions)*len(cfg.Commodities)),

		wear:     make([]wearFact, 0, tools),
		spent:    make([]Amount, 0, items),
		newRes:   make([]int64, 0, outs),
		created:  make([]int64, outs),
		disposed: make([]int64, outs),
		overflow: make([]int64, outs),
	}
}

// The codes the count's limits carry. The count returns the code of the first
// limit at 0, and limitReason maps it to the Reason of the master order of
// spec-production.md 4.2.1.
const (
	limitZeroYield = iota
	limitNoSite
	limitNoTool
	limitNoInput
)

func limitReason(code int) Reason {
	switch code {
	case limitZeroYield:
		return ReasonZeroYield
	case limitNoSite:
		return ReasonNoSite
	case limitNoTool:
		return ReasonNoTool
	case limitNoInput:
		return ReasonNoInput
	}
	return ""
}

// startReq is one request to start a recipe (spec-production.md 4.2, 4.3).
// account is the ledger account of the principal: inputs and tool items are
// read from it and taken from it (4.3.2, 4.4.1), and the outputs go to it.
// effective and hasProfile come from yieldTable.effective for the executor's
// region and the season of the start.
type startReq struct {
	recipe     RecipeID
	account    int
	principal  AgentID
	weight     int64
	maximum    int64
	effective  int64
	hasProfile bool
}

// startResult is what a start tells the world. reason is set only when k is 0
// and a limit of the list is 0; a weight of 0 gives k 0 and no reason.
type startResult struct {
	k         int64
	effective int64
	reason    Reason
}

// wearFact is the wear one start put on a tool: items distinct items worn by
// units each (spec-production.md 4.4.2).
type wearFact struct {
	commodity CommodityID
	items     int64
	units     int64
}

// limitList builds the ordered limit list of a request in the master order of
// spec-production.md 4.2.1: the yield pair (D59), a limit per tool, then one
// per input. Available units are held - reserved (D58). A commodity that is
// both a tool and an input gets the defensive cap of 4.2.3 as its tool limit.
// It reads the ledger and writes only the scratch.
func (s *productionScratch) limitList(l *ledger.Ledger, q *startReq) error {
	row := s.recipes.row(q.recipe)
	s.limits = s.limits[:0]
	yieldCap, yieldReason := yieldRefusal(s.cfg.Recipes[q.recipe].Sited, q.hasProfile, q.effective)
	code := limitZeroYield
	if yieldReason == ReasonNoSite {
		code = limitNoSite
	}
	s.limits = append(s.limits, production.Limit{Cap: yieldCap, Reason: code})
	for _, t := range row.tools {
		held, reserved, err := l.Holding(q.account, int(t.commodity))
		if err != nil {
			return fmt.Errorf("read tool stock of recipe %d: %w", q.recipe, err)
		}
		c := production.Cap(held-reserved, 1)
		for _, in := range row.inputs {
			if in.commodity == t.commodity {
				c = production.CapInputAndTool(held-reserved, int64(in.quantity))
			}
		}
		s.limits = append(s.limits, production.Limit{Cap: c, Reason: limitNoTool})
	}
	for _, in := range row.inputs {
		held, reserved, err := l.Holding(q.account, int(in.commodity))
		if err != nil {
			return fmt.Errorf("read input stock of recipe %d: %w", q.recipe, err)
		}
		s.limits = append(s.limits, production.Limit{Cap: production.Cap(held-reserved, int64(in.quantity)), Reason: limitNoInput})
	}
	return nil
}

// maxExecutions is the read of the limit (spec-production.md 4.2): the
// executions k0 the request could start and, when k0 is 0, the first
// applicable reason. It changes nothing, and start acts on its result, so the
// world's views and the start cannot drift (D52).
func (s *productionScratch) maxExecutions(l *ledger.Ledger, q *startReq) (k0 int64, reason Reason, err error) {
	if err := s.limitList(l, q); err != nil {
		return 0, "", err
	}
	k, code := production.Count(q.weight, q.maximum, s.limits)
	return k, limitReason(code), nil
}

// start begins a job (spec-production.md 4.3, 4.4, 5.1). j must hold no job
// and residues must be the agent's residues of this recipe's outputs, one per
// output item. On k = 0 it returns the reason and changes nothing. A ledger
// refusal of the input Transform is returned with the job unstarted, the
// residues untouched and the ledger unchanged (FR-PRO-010). The wear facts
// and the inputs spent are left in s.wear and s.spent until the next start.
func (s *productionScratch) start(l *ledger.Ledger, j *job, residues []int64, q *startReq) (startResult, error) {
	if j.executions > 0 {
		panic("economysim: invariant violated: start of a job for an agent that holds one")
	}
	row := s.recipes.row(q.recipe)
	if len(residues) != len(row.outputs) {
		panic("economysim: invariant violated: residues do not match the outputs of the recipe")
	}
	k, reason, err := s.maxExecutions(l, q)
	if err != nil || k == 0 {
		return startResult{reason: reason}, err
	}

	// The pending output and the new residues are formed before the ledger
	// is touched, so a value that does not fit refuses the start whole.
	if err := s.formPending(j, row, residues, k, q); err != nil {
		return startResult{}, err
	}
	if err := s.formInputs(j, row, k, q.recipe); err != nil {
		return startResult{}, err
	}
	// A source or tool-only recipe has no input, and a Transform with both
	// sides empty is refused.
	if len(s.inputs) > 0 {
		if err := l.Transform(q.account, s.inputs, nil); err != nil {
			j.pending = j.pending[:0]
			return startResult{}, fmt.Errorf("take the inputs of recipe %d: %w", q.recipe, err)
		}
	}
	s.wearTools(l, row, k, q.account)
	copy(residues, s.newRes)
	j.recipe, j.executions, j.principal, j.yield = q.recipe, k, q.principal, q.effective
	j.remaining = s.cfg.Recipes[q.recipe].Duration
	return startResult{k: k, effective: q.effective}, nil
}

// formPending fills j.pending and s.newRes for k executions of the recipe. On a
// value that does not fit it clears j.pending and returns ErrOverflow.
func (s *productionScratch) formPending(j *job, row *recipeRow, residues []int64, k int64, q *startReq) error {
	j.pending, s.newRes = j.pending[:0], s.newRes[:0]
	for i, o := range row.outputs {
		produced, rest, ok := production.Output(k, int64(o.quantity), q.effective, residues[i])
		if !ok {
			j.pending = j.pending[:0]
			return fmt.Errorf("pending output of recipe %d does not fit: %w", q.recipe, ErrOverflow)
		}
		j.pending = append(j.pending, produced)
		s.newRes = append(s.newRes, rest)
	}
	return nil
}

// formInputs fills s.inputs and s.spent with k executions of the recipe's
// inputs. On a product that does not fit it clears j.pending and returns
// ErrOverflow.
func (s *productionScratch) formInputs(j *job, row *recipeRow, k int64, recipe RecipeID) error {
	s.inputs, s.spent = s.inputs[:0], s.spent[:0]
	for _, in := range row.inputs {
		n, ok := checked.Mul(k, int64(in.quantity))
		if !ok {
			// Excluded from coverage (testing.md 9.5): k is at most the stock
			// over the quantity, so the product is at most the stock.
			j.pending = j.pending[:0]
			return fmt.Errorf("inputs of recipe %d do not fit: %w", recipe, ErrOverflow)
		}
		s.inputs = append(s.inputs, ledger.Line{Commodity: int(in.commodity), Quantity: n})
		s.spent = append(s.spent, Amount{Commodity: in.commodity, Quantity: Quantity(n)})
	}
	return nil
}

// wearTools wears every tool of the recipe by k executions and records the
// facts in s.wear.
func (s *productionScratch) wearTools(l *ledger.Ledger, row *recipeRow, k int64, account int) {
	s.wear = s.wear[:0]
	for _, t := range row.tools {
		if err := l.WearItems(account, int(t.commodity), k, t.wearPerUse); err != nil {
			// Excluded from coverage (testing.md 9.5): the count showed k
			// unreserved items, so the ledger cannot refuse; the Stage 0
			// determinism guard pins the word "invariant".
			panic("economysim: invariant violated: wear of a counted tool refused: " + err.Error())
		}
		s.wear = append(s.wear, wearFact{commodity: t.commodity, items: k, units: t.wearPerUse})
	}
}

// advance counts one tick of the job (spec-production.md 5.2): it lowers
// remaining by one and reports done at 0. working is the executions that are
// a working household-tick this tick (FR-PRO-015). The world calls it once
// for a continuation and once after a start.
func (j *job) advance() (done bool, working int64) {
	if j.executions <= 0 || j.remaining <= 0 {
		panic("economysim: invariant violated: advance of a job that is not in progress")
	}
	j.remaining--
	return j.remaining == 0, j.executions
}

// clear ends the job and keeps the pending slice for the next start (A5).
func (j *job) clear() {
	j.remaining, j.executions, j.pending = 0, 0, j.pending[:0]
}

// completion is what a completed job tells the world: the facts of the job
// that the event needs, and per output item the units created and the units
// never held (D60), of which overflow is the part that is not a discard
// output. created, disposed and overflow are views of the scratch, valid until
// the next complete.
type completion struct {
	recipe     RecipeID
	executions int64
	yield      int64
	created    []int64
	disposed   []int64
	overflow   []int64
}

// cutOutputs sorts the pending output of j into s.created, s.disposed and
// s.overflow and cuts the lines to create into s.outputs, each to the room the
// principal has left for its commodity.
func (s *productionScratch) cutOutputs(l *ledger.Ledger, j *job, row *recipeRow, account int, maxQty int64) {
	n := len(row.outputs)
	s.created, s.disposed, s.overflow, s.outputs = s.created[:n], s.disposed[:n], s.overflow[:n], s.outputs[:0]
	for i, o := range row.outputs {
		q := j.pending[i]
		s.created[i], s.disposed[i], s.overflow[i] = q, 0, 0
		switch {
		case q <= 0:
		case o.discard:
			s.disposed[i] = q
		default:
			held, _, err := l.Holding(account, int(o.commodity))
			if err != nil {
				panic("economysim: invariant violated: holding of the principal unreadable: " + err.Error())
			}
			// The room counts every held unit, reserved ones included, and
			// the lines already cut for the same commodity.
			for _, ln := range s.outputs {
				if ln.Commodity == int(o.commodity) {
					held += ln.Quantity
				}
			}
			fit := min(q, max(0, maxQty-held))
			s.disposed[i], s.overflow[i] = q-fit, q-fit
			if fit > 0 {
				s.outputs = append(s.outputs, ledger.Line{Commodity: int(o.commodity), Quantity: fit})
			}
		}
	}
}

// complete creates the pending output of a finished job in the principal's
// account (spec-production.md 5.3, D54, D58, D60) and clears the job. maxQty
// must equal the ledger's maximum quantity. A refusal by the ledger cannot
// come from input, so it panics.
func (s *productionScratch) complete(l *ledger.Ledger, j *job, account int, maxQty int64) completion {
	if j.executions <= 0 || j.remaining != 0 {
		panic("economysim: invariant violated: completion of a job that is not finished")
	}
	row := s.recipes.row(j.recipe)
	s.cutOutputs(l, j, row, account, maxQty)
	if len(s.outputs) > 0 {
		if err := l.Transform(account, nil, s.outputs); err != nil {
			// Excluded from coverage (testing.md 9.5): each line is cut to the
			// room read in this step; only an int64 overflow of a flow total
			// could refuse it.
			panic("economysim: invariant violated: creation of the outputs refused: " + err.Error())
		}
	}
	for i, o := range row.outputs {
		if s.disposed[i] > 0 {
			if err := l.Discard(account, int(o.commodity), s.disposed[i]); err != nil {
				// Excluded from coverage (testing.md 9.5): as for the Transform.
				panic("economysim: invariant violated: disposal of the surplus refused: " + err.Error())
			}
		}
	}
	c := completion{recipe: j.recipe, executions: j.executions, yield: j.yield, created: s.created, disposed: s.disposed, overflow: s.overflow}
	j.clear()
	return c
}

// abandon ends the job and moves nothing (spec-production.md 5.4): the inputs
// stay in recipe_input and the pending output is dropped. The reason is the
// caller's. A cleared job gives 0 executions.
func (j *job) abandon() (recipe RecipeID, executions int64) {
	recipe, executions = j.recipe, j.executions
	j.clear()
	return recipe, executions
}

// reweigh applies the fall of the executor's weight to w (spec-production.md
// 5.4): the executions become min(k, w) and the pending output scales down; a
// rise changes nothing. It returns the new executions; 0 ends the job.
func (j *job) reweigh(w int64) int64 {
	if j.executions <= 0 {
		return 0
	}
	k, ok := production.Rescale(j.executions, w, j.pending)
	if !ok {
		panic("economysim: invariant violated: rescale of a job refused")
	}
	j.executions = k
	if k == 0 {
		j.clear()
	}
	return k
}

// warmJob sets the job of a warm start (spec-world.md 5.6.1 item 4,
// spec-production.md 15): the pending output is fixed with the effective
// factor of the virtual start's season, the shock at its default and residue
// 0 on a scratch copy, so the agent's own residues are left for the creation
// draw (FR-PRO-026). It touches no ledger.
func (s *productionScratch) warmJob(j *job, t *yieldTable, region RegionID, season int, recipe RecipeID, principal AgentID, executions, remaining int64) error {
	var neutral yieldShocks
	e, _, ok := t.effective(&neutral, region, recipe, season, s.cfg.Recipes[recipe].Sited)
	if !ok || executions <= 0 || remaining <= 0 {
		return fmt.Errorf("warm job of recipe %d (executions %d, remaining %d): %w", recipe, executions, remaining, ErrOutOfBounds)
	}
	j.pending = j.pending[:0]
	for _, o := range s.recipes.row(recipe).outputs {
		produced, _, ok := production.Output(executions, int64(o.quantity), e, 0)
		if !ok {
			j.pending = j.pending[:0]
			return fmt.Errorf("pending output of warm job of recipe %d does not fit: %w", recipe, ErrOverflow)
		}
		j.pending = append(j.pending, produced)
	}
	j.recipe, j.remaining, j.executions, j.principal, j.yield = recipe, remaining, executions, principal, e
	return nil
}

// recipeExecuted writes the RecipeExecuted value of a completion into the
// caller's slices, which it appends to (spec-production.md 14.1, D60, D67).
// Outputs is the gross units created per output item, Disposed the part
// never held and Overflow the part of Disposed that is not a discard output,
// rows above 0 only. StartTick is derived: the completing tick
// minus duration plus 1. It copies from the completion and allocates nothing
// for a destination with room. It does not emit.
func (s *productionScratch) recipeExecuted(outs, disp, over []Amount, tick int64, agent AgentID, region RegionID, c *completion, duration int64) RecipeExecuted {
	for i, o := range s.recipes.row(c.recipe).outputs {
		if c.created[i] > 0 {
			outs = append(outs, Amount{Commodity: o.commodity, Quantity: Quantity(c.created[i])})
		}
		if c.disposed[i] > 0 {
			disp = append(disp, Amount{Commodity: o.commodity, Quantity: Quantity(c.disposed[i])})
		}
		if c.overflow[i] > 0 {
			over = append(over, Amount{Commodity: o.commodity, Quantity: Quantity(c.overflow[i])})
		}
	}
	return RecipeExecuted{
		Tick: tick, Agent: agent, Region: region, Recipe: c.recipe,
		StartTick: tick - duration + 1, Executions: c.executions,
		Outputs: outs, Disposed: disp, EffectiveYield: c.yield, Overflow: over,
	}
}
