package indicator

import (
	"cmp"
	"slices"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/internal/measure"
)

// Observe folds one event into the open window. It ignores every event while no
// window is open (but a WorldRestored opens one at the event's snapshot,
// spec-indicators.md 10.5.2) and every event type no preset reads, among them
// the per-order and per-trade classes. A Stage 3 event adds one case to one of
// the folds below. Observe allocates nothing for an event of an existing
// row (NFR-IND-002).
func (t *Tally) Observe(e economysim.Event) {
	if !t.open {
		if wr, ok := e.(economysim.WorldRestored); ok && t.Open(wr.Snapshot) != nil {
			*t = Tally{}
		}
		return
	}
	if t.foldStep(e) || t.foldPeople(e) || t.foldTransport(e) {
		return
	}
	t.foldLimits(e)
}

// foldStep folds the events that carry the sums of a step.
func (t *Tally) foldStep(e economysim.Event) bool {
	switch e := e.(type) {
	case economysim.TickStarted:
		t.tickStarted(e)
	case economysim.TickEnded:
		t.tickEnded(e)
	case economysim.RunEnded:
		t.failed = t.failed || e.Failed
	case economysim.BoundaryFlow:
		t.boundaryFlow(e)
	case economysim.MarketCleared:
		t.marketCleared(e)
	case economysim.BeliefsUpdated:
		t.beliefsUpdated(e)
	case economysim.RecipeExecuted:
		t.recipeExecuted(e)
	default:
		return false
	}
	return true
}

// foldPeople folds the events that move households.
func (t *Tally) foldPeople(e economysim.Event) bool {
	switch e := e.(type) {
	case economysim.AgentSpawned:
		t.agentSpawned(e)
	case economysim.AgentBankrupted:
		t.agentBankrupted(e)
	case economysim.WeightChanged:
		t.weightChanged(e)
	case economysim.RoleChanged:
		t.roleChanged(e)
	case economysim.AgentRelocated:
		t.agentRelocated(e)
	case economysim.UpkeepShortfall:
		t.upkeepShortfall(e)
	default:
		return false
	}
	return true
}

// foldTransport folds the shipments of the transport module and the contracts of
// the merchant module (the rows of Dispatches, Arrivals and Settlements).
func (t *Tally) foldTransport(e economysim.Event) bool {
	switch e := e.(type) {
	case economysim.ShipmentDispatched:
		t.shipmentDispatched(e)
	case economysim.ShipmentArrived:
		t.shipmentArrived(e)
	case economysim.ContractFormed:
		if r := t.settlement(e.Tick, e.Region, e.To, e.Escrowed != 0); r != nil {
			t.add((*int64)(&r.Escrowed), int64(e.Escrowed))
		}
	case economysim.ContractSettled:
		if r := t.settlement(e.Tick, e.Region, e.To, e.Paid != 0 || e.Refunded != 0); r != nil {
			t.add((*int64)(&r.Paid), int64(e.Paid))
			t.add((*int64)(&r.Refunded), int64(e.Refunded))
		}
	default:
		return false
	}
	return true
}

// foldLimits folds the aggregates of the caps, limits and relief.
func (t *Tally) foldLimits(e economysim.Event) {
	switch e := e.(type) {
	case economysim.PopulationChanged:
		t.populationChanged(e)
	case economysim.EntryLimited:
		t.entryLimited(e)
	case economysim.MoneyFlowLimited:
		t.moneyFlowLimited(e)
	case economysim.ReliefPaid:
		t.reliefPaid(e)
	}
}

func (t *Tally) tickStarted(e economysim.TickStarted) {
	if !t.step(e.Tick) {
		return
	}
	for i := range t.w.People {
		p := &t.w.People[i]
		t.add(&p.HouseholdTicks, p.Households)
	}
}

func (t *Tally) tickEnded(e economysim.TickEnded) {
	if !t.step(e.Tick) {
		return
	}
	// The steps of a window are consecutive from its first tick.
	if (t.steps == 0 && e.Tick != t.w.From) || (t.steps > 0 && e.Tick != t.last+1) {
		t.stray = true
	}
	t.steps++
	t.last = e.Tick
	t.failed = t.failed || e.Failed
}

// flowKey orders the flow rows by (region, reason, commodity).
func flowKey(r economysim.FlowTotal, e economysim.BoundaryFlow) int {
	return cmp.Or(cmp.Compare(r.Region, e.Region), cmp.Compare(r.Reason, e.Reason), cmp.Compare(r.Commodity, e.Commodity))
}

// boundaryFlow folds a BoundaryFlow into the flows, or the construction bucket
// for Tick -1 (spec-indicators.md 7.1.4).
func (t *Tally) boundaryFlow(e economysim.BoundaryFlow) {
	rows := &t.w.Flows
	if e.Tick == -1 {
		rows = &t.w.Construction
	} else if !t.step(e.Tick) {
		return
	}
	i, found := slices.BinarySearchFunc(*rows, e, flowKey)
	if !found {
		*rows = slices.Insert(*rows, i, economysim.FlowTotal{Region: e.Region, Reason: e.Reason, Kind: e.Kind, Commodity: e.Commodity})
	}
	t.add(&(*rows)[i].Total, e.Total)
}

func (t *Tally) marketCleared(e economysim.MarketCleared) {
	i, ok := t.w.market(e.Region, e.Commodity)
	if !ok || !t.step(e.Tick) || e.Price < 0 || e.Volume < 0 {
		t.stray = true
		return
	}
	m := &t.w.Markets[i]
	sum, ok := m.SumPV.u().Add(measure.Mul64(uint64(e.Price), uint64(e.Volume)))
	if !ok {
		t.over = true
	}
	m.SumPV = Wide{Hi: sum.Hi, Lo: sum.Lo}
	t.add(&m.SumV, int64(e.Volume))
	m.Ticks++
	if e.Volume > 0 {
		m.Active++
	} else if e.UnmatchedBid > 0 && e.UnmatchedAsk > 0 {
		m.Stuck++
	}
	if m.Ticks == 1 {
		m.Low, m.High = e.Price, e.Price
	}
	m.Low, m.High = min(m.Low, e.Price), max(m.High, e.Price)
	m.Price, m.PriceAge = e.Price, e.PriceAge
}

func (t *Tally) beliefsUpdated(e economysim.BeliefsUpdated) {
	i, ok := t.w.market(e.Region, e.Commodity)
	if !ok || !t.step(e.Tick) {
		t.stray = true
		return
	}
	b := &t.w.Beliefs[i]
	b.Ticks++
	t.add(&b.Outcomes, e.Outcomes)
	t.add(&b.AtFloor, e.AtFloor)
	t.add(&b.AtCeiling, e.AtCeiling)
	b.LowCentre, b.HighCentre, b.MeanCentre = e.LowCentre, e.HighCentre, e.MeanCentre
	if e.MeanCentre <= 0 {
		return
	}
	span, fits := measure.ToInt64(mustSub(int64(e.HighCentre), int64(e.LowCentre)))
	if !fits {
		t.over = true // reported by Close as ErrOverflow
		return
	}
	q, ok := measure.RoundDiv(measure.MulInt64(10_000, span), measure.FromInt64(int64(e.MeanCentre)))
	if v, fits := measure.ToInt64(q); ok && fits && (!b.HasLowest || BP(v) < b.Lowest) {
		b.Lowest, b.HasLowest = BP(v), true
	}
}

// outputKey orders the output rows by (region, recipe, commodity).
func outputKey(r OutputTally, k OutputTally) int {
	return cmp.Or(cmp.Compare(r.Region, k.Region), cmp.Compare(r.Recipe, k.Recipe), cmp.Compare(r.Commodity, k.Commodity))
}

func (t *Tally) recipeExecuted(e economysim.RecipeExecuted) {
	if e.Region < 0 || int(e.Region) >= t.w.Regions || e.Recipe < 0 || int(e.Recipe) >= t.w.Recipes || !t.step(e.Tick) {
		t.stray = true
		return
	}
	t.add(&t.w.Recipe[int(e.Region)*t.w.Recipes+int(e.Recipe)].Executions, e.Executions)
	for _, a := range e.Outputs {
		t.output(e, a, false)
	}
	for _, a := range e.Disposed {
		t.output(e, a, true)
	}
}

// output folds one output amount of a RecipeExecuted into its row.
func (t *Tally) output(e economysim.RecipeExecuted, a economysim.Amount, disposed bool) {
	if a.Commodity < 0 || int(a.Commodity) >= t.w.Commodities {
		t.stray = true
		return
	}
	k := OutputTally{Region: e.Region, Recipe: e.Recipe, Commodity: a.Commodity}
	i, found := slices.BinarySearchFunc(t.w.Outputs, k, outputKey)
	if !found {
		t.w.Outputs = slices.Insert(t.w.Outputs, i, k)
	}
	if disposed {
		t.add(&t.w.Outputs[i].Disposed, int64(a.Quantity))
		return
	}
	t.add(&t.w.Outputs[i].Output, int64(a.Quantity))
}

// person returns the cell of a household event, or nil (and notes the fault) for
// an ID outside the names table or a tick before the window.
func (t *Tally) person(r economysim.RegionID, role economysim.RoleID, tick int64, callEvent bool) *PeopleTally {
	i, ok := t.w.cell(r, role)
	if !ok {
		t.stray = true
		return nil
	}
	if (callEvent && !t.call(tick)) || (!callEvent && !t.step(tick)) {
		return nil
	}
	return &t.w.People[i]
}

func (t *Tally) agentSpawned(e economysim.AgentSpawned) {
	if p := t.person(e.Region, e.Role, e.Tick, true); p != nil {
		t.add(&p.Spawned, e.Weight)
		t.add(&p.Households, e.Weight)
	}
}

func (t *Tally) agentBankrupted(e economysim.AgentBankrupted) {
	if p := t.person(e.Region, e.Role, e.Tick, true); p != nil {
		t.add(&p.Bankrupted, e.Weight)
		t.sub(&p.Households, e.Weight)
	}
}

func (t *Tally) weightChanged(e economysim.WeightChanged) {
	if p := t.person(e.Region, e.Role, e.Tick, true); p != nil {
		d := t.diff(e.To, e.From)
		t.add(&p.WeightDelta, d)
		t.add(&p.Households, d)
	}
}

func (t *Tally) roleChanged(e economysim.RoleChanged) {
	out, in := t.person(e.Region, e.From, e.Tick, true), t.person(e.Region, e.To, e.Tick, true)
	if out == nil || in == nil {
		return
	}
	t.add(&out.RoleOut, e.Weight)
	t.sub(&out.Households, e.Weight)
	t.add(&in.RoleIn, e.Weight)
	t.add(&in.Households, e.Weight)
}

// relocKey orders the relocation rows by (from, to).
func relocKey(r RelocTally, k RelocTally) int {
	return cmp.Or(cmp.Compare(r.From, k.From), cmp.Compare(r.To, k.To))
}

func (t *Tally) agentRelocated(e economysim.AgentRelocated) {
	out, in := t.person(e.From, e.Role, e.Tick, true), t.person(e.To, e.Role, e.Tick, true)
	if out == nil || in == nil {
		return
	}
	t.add(&out.RelocOut, e.Weight)
	t.sub(&out.Households, e.Weight)
	t.add(&in.RelocIn, e.Weight)
	t.add(&in.Households, e.Weight)
	k := RelocTally{From: e.From, To: e.To}
	i, found := slices.BinarySearchFunc(t.w.Relocations, k, relocKey)
	if !found {
		t.w.Relocations = slices.Insert(t.w.Relocations, i, k)
	}
	r := &t.w.Relocations[i]
	t.add((*int64)(&r.Money), int64(e.Money))
	t.add(&r.Units, e.Units)
	t.add(&r.Weight, e.Weight)
}

func (t *Tally) upkeepShortfall(e economysim.UpkeepShortfall) {
	if p := t.person(e.Region, e.Role, e.Tick, false); p != nil {
		t.add(&p.Unfed, e.Unfed)
		p.MaxHunger = max(p.MaxHunger, e.MaxHunger)
	}
}

// region returns the regional row of a step event, or nil for a region outside
// the names table or a tick before the window.
func (t *Tally) region(r economysim.RegionID, tick int64) *RegionTally {
	if r < 0 || int(r) >= t.w.Regions || !t.step(tick) {
		t.stray = true
		return nil
	}
	return &t.w.Regional[r]
}

func (t *Tally) populationChanged(e economysim.PopulationChanged) {
	rt := t.region(e.Region, e.Tick)
	p := t.person(e.Region, e.Role, e.Tick, false)
	if rt == nil || p == nil {
		return
	}
	t.add(&p.BirthsDone, e.BirthsDone)
	t.add(&p.DeathsDone, e.DeathsDone)
	b := &rt.Binding
	t.add(&b.BirthsCapped, e.BirthsCapped)
	t.add(&b.BirthsUnfunded, e.BirthsUnfunded)
	t.add(&b.DeathsCut, e.DeathsCut)
	t.add(&b.DeathsHeld, e.DeathsHeld)
}

func (t *Tally) entryLimited(e economysim.EntryLimited) {
	rt := t.region(e.Region, e.Tick)
	if rt == nil {
		return
	}
	b := &rt.Binding
	t.add(&b.Unfunded, e.Unfunded)
	t.add(&b.RegionCap, e.RegionCap)
	t.add(&b.RoleCap, e.RoleCap)
	t.add(&b.Cooldown, e.Cooldown)
	t.add(&b.Hysteresis, e.Hysteresis)
	t.add(&b.ExitCap, e.ExitCap)
	t.add(&b.Refused, e.Refused)
}

func (t *Tally) moneyFlowLimited(e economysim.MoneyFlowLimited) {
	if rt := t.region(e.Region, e.Tick); rt != nil {
		t.add(&rt.Binding.FlowShort, e.Short)
		t.add(&rt.Binding.FlowLimit, e.Limit)
	}
}

func (t *Tally) reliefPaid(e economysim.ReliefPaid) {
	if rt := t.region(e.Region, e.Tick); rt != nil {
		t.add((*int64)(&rt.ReliefPaid), int64(e.Paid))
		t.add(&rt.ReliefRecipients, e.Recipients)
		t.add(&rt.ReliefHouseholds, e.Households)
	}
}

// lane reports whether both regions are in the names table and the tick is in
// the window; a stray event is never added to a row.
func (t *Tally) lane(tick int64, from, to economysim.RegionID) bool {
	if from < 0 || int(from) >= t.w.Regions || to < 0 || int(to) >= t.w.Regions || !t.step(tick) {
		t.stray = true
		return false
	}
	return true
}

func dispatchKey(r DispatchTally, k DispatchTally) int {
	return cmp.Or(cmp.Compare(r.Origin, k.Origin), cmp.Compare(r.Destination, k.Destination), cmp.Compare(r.Mode, k.Mode))
}

func (t *Tally) shipmentDispatched(e economysim.ShipmentDispatched) {
	if !t.lane(e.Tick, e.Region, e.To) {
		return
	}
	for _, l := range e.Lines {
		if l.Commodity < 0 || int(l.Commodity) >= t.w.Commodities {
			t.stray = true
			return
		}
	}
	k := DispatchTally{Origin: e.Region, Destination: e.To, Mode: int64(e.Mode)}
	i, found := slices.BinarySearchFunc(t.w.Dispatches, k, dispatchKey)
	if !found {
		t.w.Dispatches = slices.Insert(t.w.Dispatches, i, k)
	}
	d := &t.w.Dispatches[i]
	t.add(&d.Shipments, 1)
	if e.Hired {
		t.add(&d.Hired, 1)
	}
	t.add(&d.HaulUnits, e.HaulUnits)
	for _, l := range e.Lines {
		t.add(&d.Units, int64(l.Quantity))
	}
	// Struck stays 0 until the risk module appends the loss to the event.
}

func arrivalKey(r ArrivalTally, k ArrivalTally) int {
	return cmp.Or(cmp.Compare(r.Origin, k.Origin), cmp.Compare(r.Destination, k.Destination), cmp.Compare(r.Commodity, k.Commodity))
}

func (t *Tally) shipmentArrived(e economysim.ShipmentArrived) {
	if !t.lane(e.Tick, e.Region, e.To) {
		return
	}
	for _, l := range e.Lines {
		if l.Commodity < 0 || int(l.Commodity) >= t.w.Commodities {
			t.stray = true
			return // an event with a bad line is dropped whole, as the dispatch is
		}
	}
	// Lost and AccountLost stay 0 until the risk module appends them to ArrivedLine.
	for _, l := range e.Lines {
		k := ArrivalTally{Origin: e.Region, Destination: e.To, Commodity: l.Commodity}
		i, found := slices.BinarySearchFunc(t.w.Arrivals, k, arrivalKey)
		if !found {
			t.w.Arrivals = slices.Insert(t.w.Arrivals, i, k)
		}
		a := &t.w.Arrivals[i]
		t.add(&a.Shipped, int64(l.Shipped))
		t.add(&a.Delivered, int64(l.Delivered))
		t.add(&a.Diverted, int64(l.Diverted))
		t.add(&a.Spoiled, int64(l.Spoiled))
	}
}

func settlementKey(r SettlementTally, k SettlementTally) int {
	return cmp.Or(cmp.Compare(r.Region, k.Region), cmp.Compare(r.To, k.To))
}

// settlement returns the row of a contract lane, or nil for a stray event or
// when nothing moved (active false), so that a row means the lane was used.
func (t *Tally) settlement(tick int64, from, to economysim.RegionID, active bool) *SettlementTally {
	if !t.lane(tick, from, to) || !active {
		return nil
	}
	k := SettlementTally{Region: from, To: to}
	i, found := slices.BinarySearchFunc(t.w.Settlements, k, settlementKey)
	if !found {
		t.w.Settlements = slices.Insert(t.w.Settlements, i, k)
	}
	return &t.w.Settlements[i]
}
