package economysim

import (
	"slices"

	"github.com/Whealinghold/economy-sim/internal/checked"
	"github.com/Whealinghold/economy-sim/internal/production"
)

// This file holds the reporting side of production: the event values, the
// refusal tally, the snapshot aggregate and the cost anchor table
// (spec-production.md 9, 14).

// recipeAbandoned writes the RecipeAbandoned value of an abandon.
func recipeAbandoned(tick int64, agent AgentID, region RegionID, recipe RecipeID, executions int64, reason Reason) RecipeAbandoned {
	return RecipeAbandoned{Tick: tick, Agent: agent, Region: region, Recipe: recipe, Executions: executions, Reason: reason}
}

// The columns of a refusalTally, in the master order of the reasons
// (spec-production.md 4.2.1). A reason of a later module takes its place in
// that order when the module is built.
const (
	refusedNotOnRole = iota
	refusedZeroYield
	refusedNoSite
	refusedNoTool
	refusedNoInput
	refusedNoEndowment
	refusedColumns
)

// refusalTally counts the households whose job request was refused, per
// region, recipe and reason, for the RecipeRefused event of one tick
// (spec-production.md 14.1). Its size is regions x recipes x reasons, fixed at
// construction, so it is bounded by the configuration and allocates nothing
// after it. The world row adds a refusal where a start returns k = 0 and
// emits the rows once at P2.8a. It emits nothing itself.
type refusalTally struct {
	recipes int
	counts  []int64 // region x recipe x column
	dirty   bool
}

func newRefusalTally(regions, recipes int) *refusalTally {
	return &refusalTally{recipes: recipes, counts: make([]int64, regions*recipes*refusedColumns)}
}

// refusalColumn is the column of a reason the tally counts; any other is a
// programmer error.
func refusalColumn(reason Reason) int {
	switch reason {
	case ReasonNotOnRole:
		return refusedNotOnRole
	case ReasonZeroYield:
		return refusedZeroYield
	case ReasonNoSite:
		return refusedNoSite
	case ReasonNoTool:
		return refusedNoTool
	case ReasonNoInput:
		return refusedNoInput
	case ReasonNoEndowment:
		return refusedNoEndowment
	default:
		panic("economysim: invariant violated: refusal of a start under the reason " + string(reason) + ", which the first slice does not count")
	}
}

// add counts weight households refused for reason. A reason the slice does not
// count, or a pair outside the tables, is a programmer error.
func (t *refusalTally) add(region RegionID, recipe RecipeID, reason Reason, weight int64) {
	col := refusalColumn(reason)
	i := (int(region)*t.recipes+int(recipe))*refusedColumns + col
	if region < 0 || recipe < 0 || int(recipe) >= t.recipes || i >= len(t.counts) || weight < 1 {
		panic("economysim: invariant violated: refusal outside the tables or not a positive count")
	}
	sum, ok := checked.Add(t.counts[i], weight)
	if !ok {
		panic("economysim: invariant violated: refusal count does not fit int64")
	}
	t.counts[i] = sum
	t.dirty = true
}

// flush calls emit with one RecipeRefused value for each (region, recipe) with
// a non-zero count, in ascending RegionID then RecipeID, and zeroes the tally.
// It does nothing, and costs nothing beyond the flag, in a tick with no
// refusal.
func (t *refusalTally) flush(tick int64, emit func(RecipeRefused)) {
	if !t.dirty {
		return
	}
	for p := 0; p*refusedColumns < len(t.counts); p++ {
		c := t.counts[p*refusedColumns : (p+1)*refusedColumns]
		if !slices.ContainsFunc(c, func(n int64) bool { return n != 0 }) {
			continue
		}
		emit(RecipeRefused{
			Tick: tick, Region: RegionID(p / t.recipes), Recipe: RecipeID(p % t.recipes),
			NotOnRole: c[refusedNotOnRole], ZeroYield: c[refusedZeroYield], NoSite: c[refusedNoSite],
			NoTool: c[refusedNoTool], NoInput: c[refusedNoInput], NoEndowment: c[refusedNoEndowment],
		})
		clear(c)
	}
	t.dirty = false
}

// addPending adds the job into the snapshot aggregates (spec-production.md
// 14.3): its pending quantities into pending, indexed by commodity, and one
// into jobs, indexed by recipe. A cleared job adds nothing. The caller zeroes
// the slices and calls it for the agents of one region. It returns false when
// a sum would overflow; the slices are then unspecified.
func (s *productionScratch) addPending(j *job, pending, jobs []int64) bool {
	if j.executions <= 0 {
		return true
	}
	for i, o := range s.recipes.row(j.recipe).outputs {
		sum, ok := checked.Add(pending[o.commodity], j.pending[i])
		if !ok {
			return false
		}
		pending[o.commodity] = sum
	}
	jobs[j.recipe]++
	return true
}

// anchors fills the cost anchor table (spec-production.md 9.3, FR-PRO-037,
// 047 to 049): per region and commodity, the least non-zero unit cost over
// the recipes that make the commodity and can run in the region. prices and
// the table are region x commodity, the previous tick's published prices; wages
// is per region (0 is unknown and gives an all-zero row); runnable lists per
// region the recipes some agent has on its role's list, in any order. The
// effective factor of each recipe is taken for the region and the season (a
// calendar index, -1 for none). The recipe data is converted inline into the
// scratch, once per call; nothing of it is kept. It allocates nothing.
func (s *productionScratch) anchors(prices, wages []int64, runnable [][]RecipeID, yt *yieldTable, sh *yieldShocks, season int) {
	rt := s.recipes
	s.loadAnchorRecipes()
	nc := len(s.cfg.Commodities)
	for g := range s.cfg.Regions {
		s.markRunnable(g, runnable, yt, sh, season)
		production.Anchors(s.costs[g*nc:(g+1)*nc], s.anchor, rt.byOutput, prices[g*nc:(g+1)*nc], wages[g])
	}
}

// loadAnchorRecipes converts the recipe data inline into the scratch, once per
// anchors call.
func (s *productionScratch) loadAnchorRecipes() {
	rt := s.recipes
	s.anchor, s.aTerms, s.aWears, s.aOuts = s.anchor[:len(rt.rows)], s.aTerms[:0], s.aWears[:0], s.aOuts[:0]
	for i := range rt.rows {
		row := &rt.rows[i]
		r := &s.anchor[i]
		t0, w0, o0 := len(s.aTerms), len(s.aWears), len(s.aOuts)
		for _, in := range row.inputs {
			s.aTerms = append(s.aTerms, production.Term{Commodity: int(in.commodity), Quantity: int64(in.quantity)})
		}
		for _, tl := range row.tools {
			s.aWears = append(s.aWears, production.Wear{Commodity: int(tl.commodity), WearPerUse: tl.wearPerUse,
				Durability: s.cfg.Commodities[tl.commodity].Durability})
		}
		for _, o := range row.outputs {
			s.aOuts = append(s.aOuts, production.Out{Commodity: int(o.commodity), Quantity: int64(o.quantity), ShareBP: o.costShare, Discard: o.discard})
		}
		r.Inputs, r.Tools, r.Outputs = s.aTerms[t0:len(s.aTerms):len(s.aTerms)], s.aWears[w0:len(s.aWears):len(s.aWears)], s.aOuts[o0:len(s.aOuts):len(s.aOuts)]
		r.Duration = s.cfg.Recipes[i].Duration
	}
}

// markRunnable sets Runnable and YieldBP of every recipe for region g.
func (s *productionScratch) markRunnable(g int, runnable [][]RecipeID, yt *yieldTable, sh *yieldShocks, season int) {
	for i := range s.anchor {
		r := &s.anchor[i]
		r.Runnable, r.YieldBP = false, 0
	}
	if g < len(runnable) {
		for _, id := range runnable[g] {
			if int(id) < 0 || int(id) >= len(s.anchor) {
				continue
			}
			r := &s.anchor[id]
			ab := s.abundance(RegionID(g), id, true)
			e, _, ok := yt.effectiveAt(sh, RegionID(g), id, season, s.cfg.Recipes[id].Sited, ab)
			if ok && e == 0 {
				e = s.bestSeasonFactor(yt, sh, g, id, ab)
			}
			if ok {
				r.Runnable, r.YieldBP = true, e
			}
		}
	}
}

// bestSeasonFactor is the largest effective factor of recipe id in region g over
// the seasons of the calendar, 0 when it has none (spec-production.md 9.3): a
// recipe with a factor of 0 only in this season is costed at the factor it has in
// its open season, so a seasonal crop keeps its anchor all year. It is a function
// of the table and the shocks, with no state.
func (s *productionScratch) bestSeasonFactor(yt *yieldTable, sh *yieldShocks, g int, id RecipeID, abundance int64) (best int64) {
	for season := range s.cfg.Calendar.Seasons {
		if e, _, ok := yt.effectiveAt(sh, RegionID(g), id, season, s.cfg.Recipes[id].Sited, abundance); ok {
			best = max(best, e)
		}
	}
	return best
}

// unitCost is the root's read of the cost anchor table (spec-production.md
// 9.4): the anchor of a commodity in a region as of the last anchors call,
// 0 when unknown or outside the table. It is not API.
func (s *productionScratch) unitCost(region RegionID, c CommodityID) int64 {
	nc := len(s.cfg.Commodities)
	if region < 0 || c < 0 || int(c) >= nc || (int(region)+1)*nc > len(s.costs) {
		return 0
	}
	return s.costs[int(region)*nc+int(c)]
}
