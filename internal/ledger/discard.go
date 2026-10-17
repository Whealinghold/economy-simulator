package ledger

import "github.com/Whealinghold/economy-sim/internal/checked"

// Discard creates qty units of a commodity under RecipeOutput and sinks them
// under Disposal in one step (spec-ledger.md 8.2.3). The units are never held:
// the account's lots, holding and earmarks do not change, the RecipeOutput
// boundary goes down and the Disposal boundary up by qty, and both flows are
// attributed to the account's region, or to the construction bucket in the
// construction phase. No room check applies, because nothing is held, and
// units earmarked for an order are not touched. It is the composition of a
// Move in from RecipeOutput and a Move out to Disposal, collapsed because the
// account is never credited (FR-LED-001).
//
// The checks run in a fixed order, and a refusal changes nothing:
//
//  1. ErrInvalidAmount, for qty below 1;
//  2. ErrUnknownAccount, then ErrClosedAccount;
//  3. ErrUnknownCommodity;
//  4. ErrReasonMismatch, if the account's kind does not fit the RecipeOutput
//     (boundary to agent) and Disposal (agent to boundary) rows, and
//     ErrUnknownAccount if the RecipeOutput or the Disposal boundary account of
//     the commodity was never opened;
//  5. ErrBalanceOverflow, for the RecipeOutput boundary, the Disposal
//     boundary, or either flow total.
func (l *Ledger) Discard(id, commodity int, qty int64) error {
	if qty < 1 {
		return ErrInvalidAmount
	}
	a, out, sink, err := l.discardTarget(id, commodity)
	if err != nil {
		return err
	}
	flows := l.flows()
	oSlot, dSlot := l.slot(a, RecipeOutput, commodity), l.slot(a, Disposal, commodity)
	_, outOK := checked.Sub(out.balance, qty)
	_, sinkOK := checked.Add(sink.balance, qty)
	if !outOK || !sinkOK || !addFits(flows[oSlot], qty) || !addFits(flows[dSlot], qty) {
		return ErrBalanceOverflow
	}

	// Nothing below can fail.
	out.balance -= qty
	sink.balance += qty
	flows[oSlot] += qty
	flows[dSlot] += qty
	return nil
}

// discardTarget runs checks 2 to 4 of Discard and returns the account and the
// RecipeOutput and Disposal boundary accounts of the commodity.
func (l *Ledger) discardTarget(id, commodity int) (a, out, sink *account, err error) {
	if a, err = l.open(id); err != nil {
		return nil, nil, nil, err
	}
	if commodity < 0 || commodity >= len(l.comms) {
		return nil, nil, nil, ErrUnknownCommodity
	}
	if !Allowed(RecipeOutput, true, KindBoundary, a.kind) || !Allowed(Disposal, true, a.kind, KindBoundary) {
		return nil, nil, nil, ErrReasonMismatch
	}
	out, sink = l.boundaryAt(RecipeOutput, commodity), l.boundaryAt(Disposal, commodity)
	if out == nil || sink == nil {
		return nil, nil, nil, ErrUnknownAccount
	}
	return a, out, sink, nil
}

// Holding returns the units of a commodity that account id holds, earmarked
// ones included, and how many of them are earmarked, so the available units
// are held - reserved (spec-ledger.md 5.2.5). It changes nothing and allocates
// nothing. It returns ErrUnknownAccount for an id outside the table and
// ErrUnknownCommodity for an index outside it; a closed account and a boundary
// hold no lots and return 0 and 0.
func (l *Ledger) Holding(id, commodity int) (held, reserved int64, err error) {
	a, err := l.at(id)
	if err != nil {
		return 0, 0, err
	}
	if commodity < 0 || commodity >= len(l.comms) {
		return 0, 0, ErrUnknownCommodity
	}
	return a.held(commodity), a.earmarkedOf(commodity), nil
}
