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
//  6. ErrInsufficientStock;
//  7. ErrBalanceOverflow, for the holding, a boundary or a flow total.
func (l *Ledger) Transform(id int, inputs, outputs []Line) error {
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
	a, err := l.open(id)
	if err != nil {
		return err
	}
	for _, side := range [2][]Line{inputs, outputs} {
		for _, ln := range side {
			if ln.Commodity < 0 || ln.Commodity >= len(l.comms) {
				return ErrUnknownCommodity
			}
		}
	}
	if len(inputs) > 0 && !Allowed(RecipeInput, true, a.kind, KindBoundary) ||
		len(outputs) > 0 && !Allowed(RecipeOutput, true, KindBoundary, a.kind) {
		return ErrReasonMismatch
	}
	for _, ln := range inputs {
		if l.boundaryAt(RecipeInput, ln.Commodity) == nil {
			return ErrUnknownAccount
		}
	}
	for _, ln := range outputs {
		if l.boundaryAt(RecipeOutput, ln.Commodity) == nil {
			return ErrUnknownAccount
		}
	}

	flows := l.flows()
	slot := func(r Reason, c int) int { return a.region*l.stride + l.off[r][thingGoods] + c }
	// The commodities are visited once each, at their first line. A line count
	// is at most MaxRecipeItems, so the quadratic scan is small.
	for i, ln := range inputs {
		first, tot, ok := firstOf(inputs, i)
		if !first {
			continue
		}
		if !ok || a.held(ln.Commodity)-a.earmarkedOf(ln.Commodity) < tot {
			return ErrInsufficientStock
		}
		if _, ok := checked.Add(flows[slot(RecipeInput, ln.Commodity)], tot); !ok {
			return ErrBalanceOverflow
		}
		if _, ok := checked.Add(l.boundaryAt(RecipeInput, ln.Commodity).balance, tot); !ok {
			return ErrBalanceOverflow
		}
	}
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
		if _, ok := checked.Add(flows[slot(RecipeOutput, ln.Commodity)], tot); !ok {
			return ErrBalanceOverflow
		}
		if _, ok := checked.Sub(l.boundaryAt(RecipeOutput, ln.Commodity).balance, tot); !ok {
			return ErrBalanceOverflow
		}
	}

	// Nothing below can fail.
	for _, ln := range inputs {
		a.drain(ln.Commodity, ln.Quantity)
		flows[slot(RecipeInput, ln.Commodity)] += ln.Quantity
		l.boundaryAt(RecipeInput, ln.Commodity).balance += ln.Quantity
	}
	a.compact()
	for _, ln := range outputs {
		l.add(a, l.creationKey(ln.Commodity), ln.Quantity)
		flows[slot(RecipeOutput, ln.Commodity)] += ln.Quantity
		l.boundaryAt(RecipeOutput, ln.Commodity).balance -= ln.Quantity
	}
	return nil
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
	if n < 1 || units < 1 {
		return ErrInvalidAmount
	}
	a, err := l.open(id)
	if err != nil {
		return err
	}
	if commodity < 0 || commodity >= len(l.comms) {
		return ErrUnknownCommodity
	}
	if !Allowed(Wear, true, a.kind, KindBoundary) {
		return ErrReasonMismatch
	}
	if l.comms[commodity].Durability == 0 {
		return ErrNotDurable
	}
	b := l.boundaryAt(Wear, commodity)
	if b == nil {
		return ErrUnknownAccount
	}
	if a.held(commodity)-a.earmarkedOf(commodity) < n {
		return ErrInsufficientStock
	}

	// Select, in order of (durability, canonical position), the n items. A
	// durable's durability is at least 1, so 0 starts below every lot.
	lo, hi := a.span(commodity)
	l.sel = l.sel[:0]
	remaining, prev := n, int64(0)
	for remaining > 0 {
		next, found := int64(math.MaxInt64), false
		for i := lo; i < hi; i++ {
			lt := a.lots[i]
			if lt.Quantity > lt.Reserved && lt.Durability > prev && lt.Durability < next {
				next, found = lt.Durability, true
			}
		}
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

	var destroyed, removed int64
	for _, s := range l.sel {
		w, ok := checked.Mul(s.n, s.removed)
		if !ok {
			return ErrBalanceOverflow
		}
		if removed, ok = checked.Add(removed, w); !ok {
			return ErrBalanceOverflow
		}
		if s.destroyed {
			destroyed += s.n
		}
	}
	flows, wear := l.flows(), l.wear
	if l.construction {
		wear = l.wearBucket
	}
	slot := a.region*l.stride + l.off[Wear][thingGoods] + commodity
	if _, ok := checked.Add(wear[commodity], removed); !ok {
		return ErrBalanceOverflow
	}
	if _, ok := checked.Add(flows[slot], destroyed); !ok {
		return ErrBalanceOverflow
	}
	if _, ok := checked.Add(b.balance, destroyed); !ok {
		return ErrBalanceOverflow
	}

	// Nothing below can fail.
	for _, s := range l.sel {
		a.lots[s.lot].Quantity -= s.n
	}
	a.compact()
	for _, s := range l.sel {
		if !s.destroyed {
			l.add(a, s.survivor, s.n)
		}
	}
	wear[commodity] += removed
	flows[slot] += destroyed
	b.balance += destroyed
	return nil
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
