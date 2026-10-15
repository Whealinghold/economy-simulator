package ledger

import (
	"math"

	"github.com/Whealinghold/economy-sim/internal/checked"
)

// Line is one commodity and a quantity of a Transform.
type Line struct {
	Commodity int
	Quantity  int64
}

// boundaryAt returns the boundary account of a goods reason and commodity, or
// nil if none was opened. The commodity must be in range.
func (l *Ledger) boundaryAt(r Reason, c int) *account {
	if id := l.boundary[l.off[r][thingGoods]+c]; id >= 0 {
		return &l.accounts[id]
	}
	return nil
}

// firstOf reports whether line i is the first of its commodity, and sum is the
// total of that commodity over all the lines.
func firstOf(lines []Line, i int) (first bool, sum int64, ok bool) {
	for j, ln := range lines {
		if ln.Commodity != lines[i].Commodity {
			continue
		}
		if j < i {
			return false, 0, true
		}
		if sum, ok = checked.Add(sum, ln.Quantity); !ok {
			return true, 0, false
		}
	}
	return true, sum, true
}

// inTotal is the total of commodity c over the lines. It does not check for
// overflow, so it is called only after firstOf has summed the same lines.
func inTotal(lines []Line, c int) (n int64) {
	for _, ln := range lines {
		if ln.Commodity == c {
			n += ln.Quantity
		}
	}
	return n
}

// Transform is a recipe execution (spec-ledger.md 8.1): it moves each input
// out of the account to the RecipeInput boundary, oldest first as Move does,
// and each output in from the RecipeOutput boundary, created as 5.2.3 says, all
// or nothing. One side may be empty, not both (8.2.1). Inputs take only
// unearmarked units. The same commodity may appear in several lines, and an
// output is checked against the holding left after the inputs.
//
// The checks run in a fixed order, and a refusal changes nothing:
//
//  1. ErrEmptyTransform;
//  2. ErrInvalidAmount, for a line below 1;
//  3. ErrUnknownAccount, then ErrClosedAccount;
//  4. ErrUnknownCommodity;
//  5. ErrReasonMismatch, if the account's kind does not fit the RecipeInput
//     (agent to boundary) or RecipeOutput row, and ErrUnknownAccount if the
//     boundary account of a line's reason and commodity was never opened;
//  6. ErrInsufficientStock, then ErrBalanceOverflow (a flow total, then the
//     boundary), for the inputs, one commodity at a time in the order of its
//     first line, so a commodity that would overflow and is named before a
//     short one is refused with ErrBalanceOverflow;
//  7. ErrBalanceOverflow, for the outputs (the holding, a flow total, then the
//     boundary), one commodity at a time in the same way.
func (l *Ledger) Transform(id int, inputs, outputs []Line) error {
	if err := transformShape(inputs, outputs); err != nil {
		return err
	}
	a, err := l.open(id)
	if err != nil {
		return err
	}
	if err := l.transformRefusal(a, inputs, outputs); err != nil {
		return err
	}

	flows := l.flows()
	if err := l.checkTransformInputs(a, flows, inputs); err != nil {
		return err
	}
	if err := l.checkTransformOutputs(a, flows, inputs, outputs); err != nil {
		return err
	}
	l.applyTransform(a, flows, inputs, outputs)
	return nil
}

// transformRefusal is checks 4 and 5 of Transform: the commodities, the
// account's kind, and the boundary accounts of the lines.
func (l *Ledger) transformRefusal(a *account, inputs, outputs []Line) error {
	if l.linesUnknownCommodity(inputs) || l.linesUnknownCommodity(outputs) {
		return ErrUnknownCommodity
	}
	if !transformKindsFit(a, inputs, outputs) {
		return ErrReasonMismatch
	}
	if !l.boundariesOpen(RecipeInput, inputs) || !l.boundariesOpen(RecipeOutput, outputs) {
		return ErrUnknownAccount
	}
	return nil
}

// transformShape is checks 1 and 2 of Transform: not empty, every line at
// least 1.
func transformShape(inputs, outputs []Line) error {
	if len(inputs) == 0 && len(outputs) == 0 {
		return ErrEmptyTransform
	}
	for _, side := range [2][]Line{inputs, outputs} {
		for _, ln := range side {
			if ln.Quantity < 1 {
				return ErrInvalidAmount
			}
		}
	}
	return nil
}

// linesUnknownCommodity reports whether any line names a commodity outside the
// table.
func (l *Ledger) linesUnknownCommodity(lines []Line) bool {
	for _, ln := range lines {
		if ln.Commodity < 0 || ln.Commodity >= len(l.comms) {
			return true
		}
	}
	return false
}

// transformKindsFit reports whether the account's kind fits the RecipeInput and
// RecipeOutput rows for the sides that are present.
func transformKindsFit(a *account, inputs, outputs []Line) bool {
	if len(inputs) > 0 && !Allowed(RecipeInput, true, a.kind, KindBoundary) {
		return false
	}
	return len(outputs) == 0 || Allowed(RecipeOutput, true, KindBoundary, a.kind)
}

// boundariesOpen reports whether every line has an opened boundary account for
// the reason.
func (l *Ledger) boundariesOpen(r Reason, lines []Line) bool {
	for _, ln := range lines {
		if l.boundaryAt(r, ln.Commodity) == nil {
			return false
		}
	}
	return true
}

// slot is the index into the flow totals of reason r and commodity c in the
// region of account a.
func (l *Ledger) slot(a *account, r Reason, c int) int {
	return a.region*l.stride + l.off[r][thingGoods] + c
}

// checkTransformInputs is check 6: per commodity, the stock and then the two
// overflow checks of its totals. The commodities are visited once each, at
// their first line. A line count is at
// most MaxRecipeItems, so the quadratic scan is small.
func (l *Ledger) checkTransformInputs(a *account, flows []int64, inputs []Line) error {
	for i, ln := range inputs {
		first, tot, ok := firstOf(inputs, i)
		if !first {
			continue
		}
		if !ok || a.held(ln.Commodity)-a.earmarkedOf(ln.Commodity) < tot {
			return ErrInsufficientStock
		}
		if _, ok := checked.Add(flows[l.slot(a, RecipeInput, ln.Commodity)], tot); !ok {
			return ErrBalanceOverflow
		}
		if _, ok := checked.Add(l.boundaryAt(RecipeInput, ln.Commodity).balance, tot); !ok {
			return ErrBalanceOverflow
		}
	}
	return nil
}

// checkTransformOutputs is check 7: an output is checked
// against the holding left after the inputs.
func (l *Ledger) checkTransformOutputs(a *account, flows []int64, inputs, outputs []Line) error {
	for i, ln := range outputs {
		first, tot, ok := firstOf(outputs, i)
		if !first {
			continue
		}
		if !ok {
			return ErrBalanceOverflow
		}
		after := a.held(ln.Commodity) - inTotal(inputs, ln.Commodity)
		if n, ok := checked.Add(after, tot); !ok || n > l.maxQty {
			return ErrBalanceOverflow
		}
		if _, ok := checked.Add(flows[l.slot(a, RecipeOutput, ln.Commodity)], tot); !ok {
			return ErrBalanceOverflow
		}
		if _, ok := checked.Sub(l.boundaryAt(RecipeOutput, ln.Commodity).balance, tot); !ok {
			return ErrBalanceOverflow
		}
	}
	return nil
}

// applyTransform makes the moves of a Transform whose checks passed; it cannot
// fail.
func (l *Ledger) applyTransform(a *account, flows []int64, inputs, outputs []Line) {
	for _, ln := range inputs {
		a.drain(ln.Commodity, ln.Quantity)
		flows[l.slot(a, RecipeInput, ln.Commodity)] += ln.Quantity
		l.boundaryAt(RecipeInput, ln.Commodity).balance += ln.Quantity
	}
	a.compact()
	for _, ln := range outputs {
		l.add(a, l.creationKey(ln.Commodity), ln.Quantity)
		flows[l.slot(a, RecipeOutput, ln.Commodity)] += ln.Quantity
		l.boundaryAt(RecipeOutput, ln.Commodity).balance -= ln.Quantity
	}
}

// drain takes n unearmarked units of commodity c, oldest first. The caller has
// checked that they exist and compacts afterwards.
func (a *account) drain(c int, n int64) {
	lo, hi := a.span(c)
	for i := lo; i < hi && n > 0; i++ {
		k := min(a.lots[i].Quantity-a.lots[i].Reserved, n)
		a.lots[i].Quantity -= k
		n -= k
	}
}

// wearSel is one selected lot of a WearItems: the lot, how many of its items
// are worn, and the key of the survivors.
type wearSel struct {
	lot       int
	n         int64
	survivor  Key
	destroyed bool
	removed   int64
}

// WearItems wears the n most worn available items of a durable commodity by
// units each (spec-ledger.md 5.4.3): lowest remaining durability first, then
// canonical order, all selected before any is worn, so none is worn twice.
// An item that reaches durability 0 or below is destroyed to the Wear boundary
// (a move of its quantity, a sink) and excess wear is not carried; survivors
// merge into the lots of their new durability. Earmarked items are never worn.
// The durability actually removed is added to the wear units of the commodity
// (5.4.2), in the construction bucket while in construction.
//
// The checks run in a fixed order, and a refusal changes nothing:
//
//  1. ErrInvalidAmount, for n or units below 1;
//  2. ErrUnknownAccount, then ErrClosedAccount;
//  3. ErrUnknownCommodity;
//  4. ErrReasonMismatch, if the kind does not fit the Wear row (agent to
//     boundary);
//  5. ErrNotDurable, for a commodity with no durability, then
//     ErrUnknownAccount if the commodity has no Wear boundary account;
//  6. ErrInsufficientStock, with fewer than n available items;
//  7. ErrBalanceOverflow, for the Wear boundary, the flow total or the wear
//     units.
func (l *Ledger) WearItems(id, commodity int, n, units int64) error {
	a, b, err := l.wearTarget(id, commodity, n, units)
	if err != nil {
		return err
	}
	l.selectWorn(a, commodity, n, units)
	destroyed, removed, ok := l.wearTotals()
	if !ok {
		return ErrBalanceOverflow
	}
	flows, wear := l.flows(), l.wearUnitsSlice()
	slot := l.slot(a, Wear, commodity)
	if !(addFits(wear[commodity], removed) && addFits(flows[slot], destroyed) && addFits(b.balance, destroyed)) {
		return ErrBalanceOverflow
	}

	// Nothing below can fail.
	l.applyWorn(a)
	wear[commodity] += removed
	flows[slot] += destroyed
	b.balance += destroyed
	return nil
}

// addFits reports whether base + delta stays within int64.
func addFits(base, delta int64) bool {
	_, ok := checked.Add(base, delta)
	return ok
}

// wearUnitsSlice returns the wear units WearItems adds to: the construction
// bucket while in construction, else the tick's.
func (l *Ledger) wearUnitsSlice() []int64 {
	if l.construction {
		return l.wearBucket
	}
	return l.wear
}

// applyWorn removes the selected items from their lots and adds the survivors
// at their new durability.
func (l *Ledger) applyWorn(a *account) {
	for _, s := range l.sel {
		a.lots[s.lot].Quantity -= s.n
	}
	a.compact()
	for _, s := range l.sel {
		if !s.destroyed {
			l.add(a, s.survivor, s.n)
		}
	}
}

// wearTarget runs checks 1 to 6 of WearItems and returns the account and the
// Wear boundary account.
func (l *Ledger) wearTarget(id, commodity int, n, units int64) (a, b *account, err error) {
	if n < 1 || units < 1 {
		return nil, nil, ErrInvalidAmount
	}
	if a, err = l.open(id); err != nil {
		return nil, nil, err
	}
	if err = l.wearRefusal(a, commodity); err != nil {
		return nil, nil, err
	}
	if b = l.boundaryAt(Wear, commodity); b == nil {
		return nil, nil, ErrUnknownAccount
	}
	if a.held(commodity)-a.earmarkedOf(commodity) < n {
		return nil, nil, ErrInsufficientStock
	}
	return a, b, nil
}

// wearRefusal is checks 3 to 5 of WearItems before the boundary account: the
// commodity, the kind and that the commodity is durable.
func (l *Ledger) wearRefusal(a *account, commodity int) error {
	if commodity < 0 || commodity >= len(l.comms) {
		return ErrUnknownCommodity
	}
	if !Allowed(Wear, true, a.kind, KindBoundary) {
		return ErrReasonMismatch
	}
	if l.comms[commodity].Durability == 0 {
		return ErrNotDurable
	}
	return nil
}

// selectWorn fills l.sel with the n items to wear, in order of (durability,
// canonical position). A durable's durability is at least 1, so 0 starts below
// every lot.
func (l *Ledger) selectWorn(a *account, commodity int, n, units int64) {
	lo, hi := a.span(commodity)
	l.sel = l.sel[:0]
	remaining, prev := n, int64(0)
	for remaining > 0 {
		next, found := a.nextDurability(lo, hi, prev)
		if !found {
			// Unreachable: the count above and this scan read the same lots.
			invariantViolated("WearItems found fewer available items than it counted")
		}
		for i := lo; i < hi && remaining > 0; i++ {
			lt := a.lots[i]
			if lt.Durability != next || lt.Quantity <= lt.Reserved {
				continue
			}
			m := min(lt.Quantity-lt.Reserved, remaining)
			remaining -= m
			s := wearSel{lot: i, n: m, removed: min(units, lt.Durability)}
			if units >= lt.Durability {
				s.destroyed = true
			} else {
				s.survivor = Key{Commodity: commodity, Born: lt.Born, Durability: lt.Durability - units}
			}
			l.sel = append(l.sel, s)
		}
		prev = next
	}
}

// nextDurability returns the lowest durability above prev among the available
// lots in [lo, hi).
func (a *account) nextDurability(lo, hi int, prev int64) (next int64, found bool) {
	next = math.MaxInt64
	for i := lo; i < hi; i++ {
		lt := a.lots[i]
		if lt.Quantity > lt.Reserved && lt.Durability > prev && lt.Durability < next {
			next, found = lt.Durability, true
		}
	}
	return next, found
}

// wearTotals sums the selection: items destroyed and durability removed. ok is
// false on overflow.
func (l *Ledger) wearTotals() (destroyed, removed int64, ok bool) {
	for _, s := range l.sel {
		w, fits := checked.Mul(s.n, s.removed)
		if !fits {
			return 0, 0, false
		}
		if removed, fits = checked.Add(removed, w); !fits {
			return 0, 0, false
		}
		if s.destroyed {
			destroyed += s.n
		}
	}
	return destroyed, removed, true
}

// Wear is WearItems with n = 1 (spec-ledger.md 5.4.3).
func (l *Ledger) Wear(id, commodity int, units int64) error {
	return l.WearItems(id, commodity, 1, units)
}

// WearUnits returns the wear units of a commodity: the durability removed by
// wear since the last EndTick or ClearTotals, or with bucket true those of the
// construction phase (5.4.2, 7.4.1). It returns ErrUnknownCommodity for an
// index outside the table.
func (l *Ledger) WearUnits(commodity int, bucket bool) (int64, error) {
	if commodity < 0 || commodity >= len(l.comms) {
		return 0, ErrUnknownCommodity
	}
	if bucket {
		return l.wearBucket[commodity], nil
	}
	return l.wear[commodity], nil
}
