package economysim

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"slices"

	"github.com/Whealinghold/economy-sim/internal/logkeys"
)

// This file is the slot calls of spec-integration.md 5, split from integration.go.

// The slot entries of spec-integration.md 5.1: a key and the whole value of the key.

// YieldShock is the yield factor of a recipe in a region, in basis points: 0 to
// MaxYieldBP, 10 000 when none is set (SetYieldShocks).
type YieldShock struct {
	Region RegionID
	Recipe RecipeID
	BP     int64
}

// TransitFactor is the factor on the length of a route by a mode, in basis points:
// 0 to MaxTransitFactorBP, 10 000 when none is set, 0 blocks new dispatches
// (SetTransitFactors).
type TransitFactor struct {
	From, To RegionID
	Mode     ModeID
	BP       int64
}

// FlowScale is the scale of a money flow, named by its ID, in basis points: 0 to
// MaxFlowScale, 10 000 when none is set, 0 pauses the flow (SetFlowScales).
type FlowScale struct {
	Flow string
	BP   int64
}

// RegionScale is a scale of a region in basis points: the birth or the death scale
// of the population module, 0 to MaxBasisPoints, the region's configured value
// when none is set (SetBirthScales, SetDeathScales).
type RegionScale struct {
	Region RegionID
	BP     int64
}

// EndowmentScale is a scale of an endowment in basis points: the capacity or the
// regeneration scale, 0 to MaxShockBP, 10 000 when none is set (SetCapacityScales,
// SetRegenScales).
type EndowmentScale struct {
	Region   RegionID
	Resource ResourceID
	BP       int64
}

// EndowmentAdjustment is a signed change of the stock of an endowment, in units,
// at most MaxEndowment in magnitude; the stock is clamped to [0, its capacity]
// (AdjustEndowments).
type EndowmentAdjustment struct {
	Region   RegionID
	Resource ResourceID
	Delta    int64
}

// OutsideShock is the price shock of an outside line, in basis points: 1 to
// MaxShockBP, 10 000 when none is set (SetOutsideShocks).
type OutsideShock struct {
	Node      RegionID
	Commodity CommodityID
	BP        int64
}

// slotKeys is check 4 of spec-integration.md 3.2 for a slice argument named
// "changes": check validates entry i (known IDs, then bounds, in the order of
// 5.1.3) and returns its key, a number that orders the keys canonically; the first
// entry that fails, or that repeats the key of an earlier one (ErrDuplicateID, its
// path ending dupField), is the error. The keys are returned for the apply.
func slotKeys[E any](entries []E, check func(i int, e E) (int64, error), dupField string) ([]int64, error) {
	keys := make([]int64, 0, len(entries))
	dup := func() error {
		if d := keyedOnce(keys); d >= 0 {
			return argError(ErrDuplicateID, fmt.Sprintf("%+v", entries[d]), "", entry("changes", d), dupField)
		}
		return nil
	}
	for i, e := range entries {
		k, err := check(i, e)
		if err != nil {
			return nil, cmp.Or(dup(), err)
		}
		keys = append(keys, k)
	}
	return keys, dup()
}

// inKeyOrder returns the indexes of keys in ascending key order, the canonical
// order of the events of a call (spec-integration.md 3.3.3). The keys are distinct.
func inKeyOrder(keys []int64) []int {
	idx := make([]int, len(keys))
	for i := range idx {
		idx[i] = i
	}
	slices.SortFunc(idx, func(a, b int) int { return cmp.Compare(keys[a], keys[b]) })
	return idx
}

// slotBP is the check of a value in basis points against its bound.
func slotBP(i int, bp, lo, hi int64, bound string) error {
	if bp < lo || bp > hi {
		return argError(ErrOutOfBounds, itoa(int(bp)), bound, entry("changes", i), "bp")
	}
	return nil
}

// slotRegion checks a region key of an entry: known, and not an outside node when
// ordinary says the call is for households (a region key that names an outside
// node has no recipes, no households and no endowments).
func (w *World) slotRegion(i int, r RegionID, field string, ordinary bool) error {
	if r < 0 || int(r) >= len(w.cfg.Regions) {
		return argError(ErrUnknownRegion, itoa(int(r)), "", entry("changes", i), field)
	}
	if ordinary && w.isOutsideNode(r) {
		return argError(ErrOutsideNode, itoa(int(r)), "", entry("changes", i), field)
	}
	return nil
}

// slotNote delivers the events of a call that the owner's form does not emit, after
// the whole call has been applied, and writes the call's Debug line (3.3.3, 12.2).
func slotNote[E Event](w *World, msg string, events []E) {
	if w.obs.supplied {
		for _, e := range events {
			w.obs.deliver(e)
		}
	}
	if ctx := context.Background(); w.log.Enabled(ctx, slog.LevelDebug) {
		w.log.LogAttrs(ctx, slog.LevelDebug, msg, slog.Int64(logkeys.Tick, w.Tick()))
	}
}

// SetYieldShocks sets the yield factor of recipes in regions (spec-integration.md
// 5.1): a key at 10 000 is cleared, a key at the value it holds is no change. The
// yield is read when a job starts, so a job in progress keeps its pending output.
// It returns ErrInStep, ErrWorldFailed, then for the first entry that fails
// ErrUnknownRegion, ErrOutsideNode, ErrUnknownRecipe, ErrOutOfBounds or
// ErrDuplicateID, as a *ValidationError; a refused call changes nothing. It emits
// YieldShocked for each changed key in (region, recipe) order.
func (w *World) SetYieldShocks(changes []YieldShock) error {
	if err := w.enter("SetYieldShocks", "", false); err != nil {
		return err
	}
	keys, err := slotKeys(changes, func(i int, c YieldShock) (int64, error) {
		if err := w.slotRegion(i, c.Region, "region", true); err != nil {
			return 0, err
		}
		if c.Recipe < 0 || int(c.Recipe) >= len(w.cfg.Recipes) {
			return 0, argError(ErrUnknownRecipe, itoa(int(c.Recipe)), "", entry("changes", i), "recipe")
		}
		return int64(c.Region)*int64(len(w.cfg.Recipes)) + int64(c.Recipe), slotBP(i, c.BP, 0, MaxYieldBP, "MaxYieldBP")
	}, "region")
	if err != nil {
		return fmt.Errorf("SetYieldShocks: %w", err)
	}
	var events []YieldShocked
	for _, i := range inKeyOrder(keys) {
		c := changes[i]
		before := w.shocks.factor(c.Region, c.Recipe)
		if before == c.BP {
			continue
		}
		if err := w.shocks.set(c.Region, c.Recipe, c.BP); err != nil {
			invariant("a validated yield shock was refused: " + err.Error())
		}
		events = append(events, YieldShocked{Tick: w.Tick(), Region: c.Region, Recipe: c.Recipe, BeforeBP: before, AfterBP: c.BP})
	}
	slotNote(w, "yield shocks set", events)
	return nil
}

// SetEdgeStatuses closes edges or changes their length by a factor (the table of
// spec-regions.md 7.3): an entry that opens an edge says Closed false and FactorBP
// 10 000. It returns ErrInStep, ErrWorldFailed, then for the first entry that fails
// ErrUnknownEdge, ErrOutOfBounds (FactorBP outside 1 to MaxEdgeFactorBP) or
// ErrDuplicateID; a refused call changes nothing. It rebuilds the graph at the call
// and emits EdgeStatusChanged, cause consumer, for each changed edge in EdgeID
// order.
func (w *World) SetEdgeStatuses(changes []EdgeStatus) error {
	if err := w.enter("SetEdgeStatuses", "", false); err != nil {
		return err
	}
	_, err := slotKeys(changes, func(i int, c EdgeStatus) (int64, error) {
		e, ok := w.geo.g.Edge(c.From, c.To)
		if !ok {
			return 0, argError(ErrUnknownEdge, fmt.Sprintf("%d to %d", c.From, c.To), "", entry("changes", i), "from")
		}
		return int64(e.ID), slotBP(i, c.FactorBP, 1, MaxEdgeFactorBP, "MaxEdgeFactorBP")
	}, "from")
	if err != nil {
		return fmt.Errorf("SetEdgeStatuses: %w", err)
	}
	if err := w.setEdgeStatuses(changes, CauseConsumer); err != nil {
		invariant("validated edge statuses were refused: " + err.Error())
	}
	return nil
}

// SetTransitFactors sets the factor on the length of routes by modes (the
// transport module): new dispatches read it, a shipment in flight keeps its
// arrival tick. It returns ErrInStep, ErrWorldFailed, ErrModuleOff, then for the
// first entry that fails ErrUnknownRegion, ErrOutsideNode, ErrSameRegion,
// ErrUnknownMode, ErrOutOfBounds or ErrDuplicateID; a refused call changes
// nothing. It emits TransitFactorChanged, cause consumer, for each changed key in
// (from, to, mode) order.
func (w *World) SetTransitFactors(changes []TransitFactor) error {
	if err := w.enter("SetTransitFactors", "transport", w.transport != nil); err != nil {
		return err
	}
	modes := int64(len(w.transport.modes))
	keys, err := slotKeys(changes, func(i int, c TransitFactor) (int64, error) {
		// Both regions are known before either is checked for an outside node.
		for _, ordinary := range [...]bool{false, true} {
			if err := w.slotRegion(i, c.From, "from", ordinary); err != nil {
				return 0, err
			}
			if err := w.slotRegion(i, c.To, "to", ordinary); err != nil {
				return 0, err
			}
		}
		switch {
		case c.From == c.To:
			return 0, argError(ErrSameRegion, itoa(int(c.To)), "", entry("changes", i), "to")
		case c.Mode < 0 || int64(c.Mode) >= modes:
			return 0, argError(ErrUnknownMode, itoa(int(c.Mode)), "", entry("changes", i), "mode")
		}
		regions := int64(len(w.cfg.Regions))
		return (int64(c.From)*regions+int64(c.To))*modes + int64(c.Mode), slotBP(i, c.BP, 0, MaxTransitFactorBP, "MaxTransitFactorBP")
	}, "from")
	if err != nil {
		return fmt.Errorf("SetTransitFactors: %w", err)
	}
	for _, i := range inKeyOrder(keys) {
		c := changes[i]
		if err := w.setTransitFactor(c.From, c.To, c.Mode, c.BP, CauseConsumer); err != nil {
			invariant("a validated transit factor was refused: " + err.Error())
		}
	}
	return nil
}

// SetFlowScales sets the scale of money flows by ID (the money supply module): 0
// pauses a flow, 10 000 is the configured rate. It returns ErrInStep,
// ErrWorldFailed, ErrModuleOff, then for the first entry that fails ErrUnknownFlow,
// ErrOutOfBounds or ErrDuplicateID; a refused call changes nothing. It emits
// MoneyFlowScaled, cause consumer, for each changed flow in flow ID order, and the
// flow is skipped whole or scaled from the next P1.7.
func (w *World) SetFlowScales(changes []FlowScale) error {
	if err := w.enter("SetFlowScales", "money_supply", w.money != nil); err != nil {
		return err
	}
	keys, err := slotKeys(changes, func(i int, c FlowScale) (int64, error) {
		at := slices.IndexFunc(w.money.flows, func(f flowState) bool { return f.flow.ID == c.Flow })
		if at < 0 {
			return 0, argError(ErrUnknownFlow, c.Flow, "", entry("changes", i), "flow")
		}
		return int64(at), slotBP(i, c.BP, 0, MaxFlowScale, "MaxFlowScale")
	}, "flow")
	if err != nil {
		return fmt.Errorf("SetFlowScales: %w", err)
	}
	for _, i := range inKeyOrder(keys) {
		if err := w.setFlowScale(int(keys[i]), changes[i].BP, CauseConsumer); err != nil {
			invariant("a validated flow scale was refused: " + err.Error())
		}
	}
	return nil
}

// SetBirthScales sets the birth scale of regions (the population module), which
// replaces the region's configured BirthScale; a scale equal to the configured one
// clears the key. It returns ErrInStep, ErrWorldFailed, ErrModuleOff, then for the
// first entry that fails ErrUnknownRegion, ErrOutsideNode, ErrOutOfBounds or
// ErrDuplicateID; a refused call changes nothing. It emits PopulationScaled for
// each changed region in ascending order; P7.1 reads the scale from the next step.
func (w *World) SetBirthScales(changes []RegionScale) error {
	return w.setPopulationScales("SetBirthScales", ScaleBirth, changes)
}

// SetDeathScales is SetBirthScales for the death scale.
func (w *World) SetDeathScales(changes []RegionScale) error {
	return w.setPopulationScales("SetDeathScales", ScaleDeath, changes)
}

// setPopulationScales is the call of SetBirthScales and SetDeathScales.
func (w *World) setPopulationScales(call string, kind ScaleKind, changes []RegionScale) error {
	if err := w.enter(call, "population", w.pop != nil); err != nil {
		return err
	}
	keys, err := slotKeys(changes, func(i int, c RegionScale) (int64, error) {
		if err := w.slotRegion(i, c.Region, "region", true); err != nil {
			return 0, err
		}
		return int64(c.Region), slotBP(i, c.BP, 0, MaxBasisPoints, "MaxBasisPoints")
	}, "region")
	if err != nil {
		return fmt.Errorf("%s: %w", call, err)
	}
	var events []PopulationScaled
	for _, i := range inKeyOrder(keys) {
		c := changes[i]
		before, ok := w.setPopulationScale(c.Region, kind, c.BP)
		if !ok {
			invariant("a validated population scale was refused")
		}
		if before != c.BP {
			events = append(events, PopulationScaled{Tick: w.Tick(), Region: c.Region, Which: kind, BeforeBP: before, AfterBP: c.BP})
		}
	}
	slotNote(w, "population scales set", events)
	return nil
}

// SetCapacityScales sets the capacity scale of endowments (the resources module):
// the effective capacity is recomputed at the call and a stock above it is clamped,
// the excess lost. It returns ErrInStep, ErrWorldFailed, ErrModuleOff, then for the
// first entry that fails ErrUnknownRegion, ErrUnknownResource, ErrOutOfBounds or
// ErrDuplicateID; a refused call changes nothing. It emits EndowmentAdjusted, kind
// shock, for each changed key in (region, resource) order.
func (w *World) SetCapacityScales(changes []EndowmentScale) error {
	return w.setEndowmentScales("SetCapacityScales", changes, (*World).setCapacityScales)
}

// SetRegenScales sets the regeneration scale of endowments, read at P8.3; the stock
// is not touched. Its errors and its event are those of SetCapacityScales.
func (w *World) SetRegenScales(changes []EndowmentScale) error {
	return w.setEndowmentScales("SetRegenScales", changes, (*World).setRegenScales)
}

// setEndowmentScales is the call of the two scales of an endowment: apply is the
// owner's form, which emits its own event.
func (w *World) setEndowmentScales(call string, changes []EndowmentScale, apply func(*World, RegionID, ResourceID, int64)) error {
	if err := w.enter(call, "resources", w.res != nil); err != nil {
		return err
	}
	keys, err := slotKeys(changes, func(i int, c EndowmentScale) (int64, error) {
		key, err := w.slotEndowment(i, c.Region, c.Resource)
		if err != nil {
			return 0, err
		}
		return key, slotBP(i, c.BP, 0, MaxShockBP, "MaxShockBP")
	}, "region")
	if err != nil {
		return fmt.Errorf("%s: %w", call, err)
	}
	for _, i := range inKeyOrder(keys) {
		apply(w, changes[i].Region, changes[i].Resource, changes[i].BP)
	}
	return nil
}

// slotEndowment checks the key of an endowment entry, a region and a resource of
// the configuration, and returns the number that orders it by (region, resource).
func (w *World) slotEndowment(i int, region RegionID, resource ResourceID) (int64, error) {
	if err := w.slotRegion(i, region, "region", false); err != nil {
		return 0, err
	}
	if resource < 0 || int(resource) >= w.res.resources {
		return 0, argError(ErrUnknownResource, itoa(int(resource)), "", entry("changes", i), "resource")
	}
	return int64(w.res.cellOf(region, resource)), nil
}

// AdjustEndowments changes the stock of endowments now, by a signed number of
// units clamped so the stock stays in [0, its capacity] (the resources module). It
// holds no value and has no default. It returns ErrInStep, ErrWorldFailed,
// ErrModuleOff, then for the first entry that fails ErrUnknownRegion,
// ErrUnknownResource, ErrOutOfBounds (a Delta above MaxEndowment in magnitude) or
// ErrDuplicateID; a refused call changes nothing. It emits EndowmentAdjusted, kind
// adjust, for each stock that changed, in (region, resource) order.
func (w *World) AdjustEndowments(changes []EndowmentAdjustment) error {
	if err := w.enter("AdjustEndowments", "resources", w.res != nil); err != nil {
		return err
	}
	keys, err := slotKeys(changes, func(i int, c EndowmentAdjustment) (int64, error) {
		key, err := w.slotEndowment(i, c.Region, c.Resource)
		if err != nil {
			return 0, err
		}
		if c.Delta < -MaxEndowment || c.Delta > MaxEndowment {
			return 0, argError(ErrOutOfBounds, itoa(int(c.Delta)), "MaxEndowment", entry("changes", i), "delta")
		}
		return key, nil
	}, "region")
	if err != nil {
		return fmt.Errorf("AdjustEndowments: %w", err)
	}
	for _, i := range inKeyOrder(keys) {
		w.adjustEndowments(changes[i].Region, changes[i].Resource, changes[i].Delta)
	}
	return nil
}

// SetOutsideShocks sets the price shock of outside lines (the external module):
// the walk and the quotes read it from the next step. It returns ErrInStep,
// ErrWorldFailed, ErrModuleOff, then for the first entry that fails
// ErrUnknownRegion, ErrUnknownCommodity, ErrUnknownLine (the node has no line for
// the commodity, and an ordinary region has none), ErrOutOfBounds or
// ErrDuplicateID; a refused call changes nothing. It emits OutsideShocked for each
// changed line in (node, commodity) order.
func (w *World) SetOutsideShocks(changes []OutsideShock) error {
	if err := w.enter("SetOutsideShocks", "external", w.external != nil); err != nil {
		return err
	}
	keys, err := slotKeys(changes, func(i int, c OutsideShock) (int64, error) {
		if err := w.slotRegion(i, c.Node, "node", false); err != nil {
			return 0, err
		}
		if !w.known(c.Commodity) {
			return 0, argError(ErrUnknownCommodity, itoa(int(c.Commodity)), "", entry("changes", i), "commodity")
		}
		if !slices.ContainsFunc(w.external.lines, func(l outsideLine) bool { return l.node == c.Node && l.commodity == c.Commodity }) {
			return 0, argError(ErrUnknownLine, fmt.Sprintf("%d, %d", c.Node, c.Commodity), "", entry("changes", i), "commodity")
		}
		return int64(c.Node)*int64(len(w.cfg.Commodities)) + int64(c.Commodity), slotBP(i, c.BP, 1, MaxShockBP, "MaxShockBP")
	}, "node")
	if err != nil {
		return fmt.Errorf("SetOutsideShocks: %w", err)
	}
	ordered := make([]outsideShock, 0, len(changes))
	for _, i := range inKeyOrder(keys) {
		ordered = append(ordered, outsideShock{node: changes[i].Node, commodity: changes[i].Commodity, bp: changes[i].BP})
	}
	slotNote(w, "outside shocks set", w.external.applyShocks(w.Tick(), ordered))
	return nil
}

// SlotState lists every slot value that differs from its default, in the canonical
// order of its owner, the order a checkpoint saves it in (spec-integration.md 5.3).
// A later module appends its field at the end.
type SlotState struct {
	YieldShocks    []YieldShock
	EdgeStatuses   []EdgeStatus
	TransitFactors []TransitFactor
	FlowScales     []FlowScale
	BirthScales    []RegionScale
	DeathScales    []RegionScale
	CapacityScales []EndowmentScale
	RegenScales    []EndowmentScale
	OutsideShocks  []OutsideShock
}

// Slots returns a deep copy of the slots that hold a value other than their default
// (spec-integration.md 5.3): a key set back to its default is not listed, and a
// world that was never shocked has an empty SlotState. It returns ErrInStep or
// ErrWorldFailed, changes nothing and draws nothing. It reads the lists the owners
// save, so a restored world reports what the original did. The one-off
// AdjustEndowments holds no value and is not listed.
func (w *World) Slots() (SlotState, error) {
	if err := w.enter("Slots", "", false); err != nil {
		return SlotState{}, err
	}
	var s SlotState
	for _, e := range w.shocks.ordered() {
		s.YieldShocks = append(s.YieldShocks, YieldShock{Region: e.region, Recipe: e.recipe, BP: e.bp})
	}
	for _, e := range w.edgeRows() {
		s.EdgeStatuses = append(s.EdgeStatuses, EdgeStatus(e))
	}
	w.transitSlots(&s)
	w.flowSlots(&s)
	w.populationSlots(&s)
	w.endowmentSlots(&s)
	w.outsideSlots(&s)
	return s, nil
}

// transitSlots, flowSlots, populationSlots, endowmentSlots and outsideSlots add the
// non-default keys of one owner to the slot list; each is empty with its module off.
func (w *World) transitSlots(s *SlotState) {
	if w.transport == nil {
		return
	}
	for _, f := range w.transport.factors {
		s.TransitFactors = append(s.TransitFactors, TransitFactor{From: f.from, To: f.to, Mode: f.mode, BP: f.bp})
	}
}

func (w *World) flowSlots(s *SlotState) {
	if w.money == nil {
		return
	}
	for _, f := range w.money.flows {
		if f.scale != wholeBP {
			s.FlowScales = append(s.FlowScales, FlowScale{Flow: f.flow.ID, BP: f.scale})
		}
	}
}

func (w *World) populationSlots(s *SlotState) {
	p := w.pop
	if p == nil {
		return
	}
	for g := range p.effBirth {
		if p.effBirth[g] != p.rowBirth[g] {
			s.BirthScales = append(s.BirthScales, RegionScale{Region: RegionID(g), BP: p.effBirth[g]})
		}
		if p.effDeath[g] != p.rowDeath[g] {
			s.DeathScales = append(s.DeathScales, RegionScale{Region: RegionID(g), BP: p.effDeath[g]})
		}
	}
}

func (w *World) endowmentSlots(s *SlotState) {
	r := w.res
	if r == nil {
		return
	}
	for cell := range r.capScale {
		if r.capScale[cell] != wholeBP {
			s.CapacityScales = append(s.CapacityScales, EndowmentScale{Region: r.regionOf(cell), Resource: r.resourceOf(cell), BP: r.capScale[cell]})
		}
		if r.regenScale[cell] != wholeBP {
			s.RegenScales = append(s.RegenScales, EndowmentScale{Region: r.regionOf(cell), Resource: r.resourceOf(cell), BP: r.regenScale[cell]})
		}
	}
}

func (w *World) outsideSlots(s *SlotState) {
	if w.external == nil {
		return
	}
	for _, l := range w.external.lines {
		if l.shock != wholeBP {
			s.OutsideShocks = append(s.OutsideShocks, OutsideShock{Node: l.node, Commodity: l.commodity, BP: l.shock})
		}
	}
}
