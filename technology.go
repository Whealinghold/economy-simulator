package economysim

import (
	"fmt"

	"github.com/Whealinghold/economy-sim/internal/checked"
	"github.com/Whealinghold/economy-sim/internal/production"
	tec "github.com/Whealinghold/economy-sim/internal/technology"
)

// This file is the technology module (spec-technology.md; architecture.md
// 2.10.1): the state, the gain reader, the counter and the key. The step is
// p83cTechnology, the call and the validation and the warnings are in
// technology_validate.go, and the events, rows and logs are in
// technology_report.go.

// TechnologyAdjustment is a signed change of the advance of one cell, the entry of
// AdjustTechnology (spec-technology.md 7.1). Plain data.
type TechnologyAdjustment struct {
	// Region is an ordinary region; an outside node has no cell.
	Region RegionID
	// Field is the technology field.
	Field TechFieldID
	// DeltaBP is the signed change, at most MaxGainBP in magnitude.
	DeltaBP int64
}

// technology is the state of the module, nil on a world that has the module off
// (spec-technology.md 3.2, 12.1). The cells are module slices and no agent holds a
// field of it (N-128, N-244). Only cells and trendCarry are saved; the counters are
// scratch, the gains are derived and the key comes from the effective seed (TC3).
type technology struct {
	fields []tec.Field
	nf     int
	// rankOf is the rank of a region among the ordinary regions, -1 for an outside
	// node (no cell, FR-TEC-036); regions is the inverse, rank to RegionID.
	rankOf  []int
	regions []RegionID
	// fieldOf is the field of a recipe, -1 for a recipe in no field.
	fieldOf []int32
	// counted marks the fields with at least one mechanism on: the executions of a
	// field with every rate 0 are not counted, so a module on with such a field is
	// the world with the module off (FR-TEC-001).
	counted []bool
	// cells is region rank * nf + field, in ascending (RegionID, TechFieldID) order.
	cells      []tec.Cell
	trendCarry []int64
	// counter is the executions completed in the tick, per cell (scratch).
	counter []int64
	// gain is the derived gain G of each cell, in basis points (never saved).
	gain []int64
	// results, scratch and dist are the step's working memory, sized once.
	results []tec.Result
	scratch tec.Scratch
	dist    tec.DistanceFunc
	key     uint64
	tpy     int64
}

// newTechnology builds the module's state from a canonical configuration: one zero
// cell per ordinary region and field in ascending order (FR-TEC-004), the field of
// each recipe and the key derived from the effective seed. It returns nil when the
// module is off. outside marks the outside nodes by RegionID and geo is the world's
// region state, read for the distances of diffusion.
func newTechnology(c *Config, outside []bool, geo *regionState, seed uint64) *technology {
	m := c.Modules.Technology
	if m == nil {
		return nil
	}
	t := &technology{nf: len(m.Fields), tpy: c.Calendar.TicksPerYear(), key: tec.Key(seed, siteTechnology)}
	t.fields = make([]tec.Field, t.nf)
	for i, f := range m.Fields {
		t.fields[i] = tec.Field{CapBP: f.CapBP, Base: f.Base, DoublingBP: f.DoublingBP, TrendBP: f.TrendBP,
			DiscoveryBP: f.DiscoveryBP, StepBP: f.StepBP, CatchUpBP: f.CatchUpBP, Reach: f.Reach}
	}
	t.rankOf = make([]int, len(c.Regions))
	for r := range c.Regions {
		t.rankOf[r] = -1
		if !outside[r] {
			t.rankOf[r] = len(t.regions)
			t.regions = append(t.regions, RegionID(r))
		}
	}
	t.fieldOf = fieldsOfRecipes(c)
	t.counted = make([]bool, t.nf)
	for i, f := range t.fields {
		t.counted[i] = f.DoublingBP > 0 || f.TrendBP > 0 || f.DiscoveryBP > 0 || f.CatchUpBP > 0
	}
	n := len(t.regions) * t.nf
	t.cells, t.gain, t.counter = make([]tec.Cell, n), make([]int64, n), make([]int64, n)
	t.results = make([]tec.Result, n)
	t.trendCarry = make([]int64, t.nf)
	t.dist = func(a, b int) (int64, bool) { return geo.distance(t.regions[a], t.regions[b]) }
	return t
}

// fieldsOfRecipes maps each RecipeID to the index of its field, -1 for a recipe in
// no field. Validation guarantees that a recipe is in at most one field.
func fieldsOfRecipes(c *Config) []int32 {
	of := make(map[string]int32)
	for i, f := range c.Modules.Technology.Fields {
		for _, r := range f.Recipes {
			of[r] = int32(i)
		}
	}
	out := make([]int32, len(c.Recipes))
	for i := range c.Recipes {
		out[i] = -1
		if f, ok := of[c.Recipes[i].ID]; ok {
			out[i] = f
		}
	}
	return out
}

// cellOf is the index of the cell of a region and a field: the rank of the region
// among the ordinary regions times the fields, plus the field. ok is false for an
// outside node, which has no cell.
func (t *technology) cellOf(region RegionID, field int) (int, bool) {
	if region < 0 || int(region) >= len(t.rankOf) || t.rankOf[region] < 0 {
		return 0, false
	}
	return t.rankOf[region]*t.nf + field, true
}

// factor is the technology factor of a recipe in a region, 10 000 + G, and 10 000
// for a recipe in no field and for an outside node (spec-technology.md 3.3.1).
func (t *technology) factor(region RegionID, recipe RecipeID) int64 {
	f := t.fieldOf[recipe]
	if f < 0 {
		return production.BP
	}
	i, ok := t.cellOf(region, int(f))
	if !ok {
		return production.BP
	}
	return production.BP + t.gain[i]
}

// techFactor is the technology term of the effective factor of a recipe in a
// region (spec-production.md 6.3, spec-technology.md 5): 10 000 with the module off
// and the gain of the cell otherwise. It reads the gain derived at the last
// step or call, so every position of a tick reads one value (FR-TEC-010).
func (s *productionScratch) techFactor(region RegionID, recipe RecipeID) int64 {
	if s.tec == nil {
		return production.BP
	}
	return s.tec.factor(region, recipe)
}

// count adds the executions of a completed job of a recipe to the counter of its
// cell (spec-technology.md 3.4). A recipe in no field, or in a field with every
// mechanism off, adds nothing. The counter saturates at MaxExperience, which no
// run reaches.
func (t *technology) count(region RegionID, recipe RecipeID, executions int64) {
	f := t.fieldOf[recipe]
	if f < 0 || !t.counted[f] {
		return
	}
	i, ok := t.cellOf(region, int(f))
	if !ok {
		return
	}
	sum, ok := checked.Add(t.counter[i], executions)
	if !ok || sum > MaxExperience {
		sum = MaxExperience
	}
	t.counter[i] = sum
}

// regain derives the gain of cell i.
func (t *technology) regain(i int) {
	g, ok := tec.Gain(t.fields[i%t.nf], t.cells[i].Experience, t.cells[i].Advance)
	if !ok {
		invariant("a technology gain does not fit after validation")
	}
	t.gain[i] = g
}

// p83cTechnology is position P8.3c (spec-technology.md 4.6, 6.1): the step over
// every cell, after the regeneration of P8.3 and before the audit. It draws no word
// and reads no price, wage, stock or count of households (FR-TEC-033). The counters
// are cleared here, once the events have read them (TC7).
func (w *World) p83cTechnology() {
	t := w.tec
	if len(t.cells) == 0 {
		return
	}
	in := tec.Input{Cells: t.cells, Counters: t.counter, TrendCarry: t.trendCarry, Fields: t.fields,
		Regions: len(t.regions), Tick: w.Tick(), TicksPerYear: t.tpy, MaxExperience: MaxExperience,
		Key: t.key, Dist: t.dist}
	if !tec.Step(in, &t.scratch, t.results) {
		invariant("a technology step does not fit after validation")
	}
	w.reportTechnology()
	clear(t.counter)
	for i := range t.results {
		t.gain[i] = t.results[i].After
	}
}

// AdjustTechnology changes the advance of cells between ticks, by a signed number
// of basis points clamped so the advance stays in [0, the field's cap]; the part
// of the change the bounds cut is lost and reported (spec-technology.md 7.1). The
// gain follows at once. It holds no value and has no default. It returns ErrInStep,
// ErrWorldFailed, ErrModuleOff, then for the first entry that fails ErrUnknownRegion,
// ErrOutsideNode, ErrUnknownTechField, ErrOutOfBounds (a DeltaBP above MaxGainBP in
// magnitude) or ErrDuplicateID; a refused call changes nothing. It emits
// TechnologyAdjusted for each cell whose advance changed, in (region, field) order.
func (w *World) AdjustTechnology(changes []TechnologyAdjustment) error {
	if err := w.enter("AdjustTechnology", "technology", w.tec != nil); err != nil {
		return err
	}
	keys, err := slotKeys(changes, w.slotTechnology, "region")
	if err != nil {
		return fmt.Errorf("AdjustTechnology: %w", err)
	}
	for _, i := range inKeyOrder(keys) {
		w.adjustTechnology(int(keys[i]), changes[i].DeltaBP)
	}
	return nil
}

// slotTechnology checks one entry of a call and returns the index of its cell, the
// key of the entry (the order of the events is the cell order).
func (w *World) slotTechnology(i int, c TechnologyAdjustment) (int64, error) {
	if err := w.slotRegion(i, c.Region, "region", true); err != nil {
		return 0, err
	}
	if c.Field < 0 || int(c.Field) >= w.tec.nf {
		return 0, argError(ErrUnknownTechField, itoa(int(c.Field)), "", entry("changes", i), "field")
	}
	if c.DeltaBP < -MaxGainBP || c.DeltaBP > MaxGainBP {
		return 0, argError(ErrOutOfBounds, itoa(int(c.DeltaBP)), "MaxGainBP", entry("changes", i), "delta_bp")
	}
	cell, _ := w.tec.cellOf(c.Region, int(c.Field))
	return int64(cell), nil
}

// adjustTechnology applies one validated entry: A' = min(cap, max(0, A + delta)),
// the gain re-derived at once and the event and the Debug line when A moved.
func (w *World) adjustTechnology(cell int, delta int64) {
	t := w.tec
	before, a := t.gain[cell], t.cells[cell].Advance
	after := min(max(a+delta, 0), t.fields[cell%t.nf].CapBP)
	if after == a {
		return
	}
	t.cells[cell].Advance = after
	t.regain(cell)
	lost := max(delta, -delta) - max(after-a, a-after)
	w.reportAdjusted(cell, delta, before, lost)
}
