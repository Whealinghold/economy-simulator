package ledger

import "github.com/Whealinghold/economy-sim/internal/checked"

// Transfer moves amount of money from one account to another under a reason
// (spec-ledger.md 4.2). It spends only available money, balance less
// reserved. The checks run in a fixed order so that one call has one answer,
// and the first that holds is returned; a refusal changes nothing (2.6):
//
//  1. ErrInvalidAmount, if amount < 1;
//  2. ErrUnknownAccount, then ErrClosedAccount, for from and then to;
//  3. ErrReasonMismatch, if the reason is unknown or does not move money, the
//     two kinds do not fit its row, the accounts are the same, or a boundary
//     account is not that reason's money account;
//  4. ErrInsufficientFunds;
//  5. ErrBalanceOverflow, in the credited account (above MaxMoney, or int64
//     for a boundary), the debited boundary, or the flow total.
//
// The flow is attributed to the region of the non-boundary side, and of the
// debited account for an internal flow (7.4, FR-LED-053), and added to the
// construction bucket while the ledger is in construction.
func (l *Ledger) Transfer(from, to int, amount int64, r Reason) error {
	return l.transfer(from, to, amount, r, false)
}

// TransferReserved is Transfer marked "from reserved" (4.3): it draws on the
// reservation of the debited account first, lowering the reservation and the
// balance together. It is ErrBoundaryReserve for a boundary account, and
// ErrReservationExceeds, in place of ErrInsufficientFunds, when the
// reservation is smaller than amount.
func (l *Ledger) TransferReserved(from, to int, amount int64, r Reason) error {
	return l.transfer(from, to, amount, r, true)
}

// moneyFits reports whether reason r has a money row in this ledger (a
// declared reason only if it was declared) that runs from the kind of f to
// the kind of t.
func (l *Ledger) moneyFits(r Reason, f, t *account) bool {
	rw, ok := rowOf(r, false)
	return ok && l.off[r][thingMoney] >= 0 && rw.fits(f.kind, t.kind)
}

// transferParties runs the checks of transfer that precede the debit, in the
// documented order, and returns the two open accounts.
func (l *Ledger) transferParties(from, to int, amount int64, r Reason) (f, t *account, err error) {
	if amount < 1 {
		return nil, nil, ErrInvalidAmount
	}
	if f, t, err = l.parties(from, to); err != nil {
		return nil, nil, err
	}
	if from == to || !l.moneyFits(r, f, t) || !boundariesFit(f, t, r, Money) {
		return nil, nil, ErrReasonMismatch
	}
	return f, t, nil
}

// boundariesFit reports whether each boundary account among f and t is the
// account of reason r and thing (Money or a commodity).
func boundariesFit(f, t *account, r Reason, thing int) bool {
	for _, b := range [2]*account{f, t} {
		if b.kind == KindBoundary && (b.reason != r || b.thing != thing) {
			return false
		}
	}
	return true
}

// debitRefusal returns the refusal of the debit side of a transfer, or nil.
func debitRefusal(f *account, amount int64, marked bool) error {
	switch {
	case marked && f.kind == KindBoundary:
		return ErrBoundaryReserve
	case marked:
		if f.reserved < amount {
			return ErrReservationExceeds
		}
	case f.kind != KindBoundary && f.balance-f.reserved < amount:
		return ErrInsufficientFunds
	}
	return nil
}

// transfer runs a money transfer: the checks in the order Transfer documents,
// then the effects, which cannot fail.
func (l *Ledger) transfer(from, to int, amount int64, r Reason, marked bool) error {
	f, t, err := l.transferParties(from, to, amount, r)
	if err != nil {
		return err
	}
	if err := debitRefusal(f, amount, marked); err != nil {
		return err
	}
	newFrom, ok := checked.Sub(f.balance, amount)
	if !ok {
		return ErrBalanceOverflow
	}
	newTo, err := l.credited(t, amount)
	if err != nil {
		return err
	}
	slot := l.flowSlot(f, t, l.off[r][thingMoney])
	flows := l.flows()
	total, ok := checked.Add(flows[slot], amount)
	if !ok {
		return ErrBalanceOverflow
	}
	f.balance, t.balance, flows[slot] = newFrom, newTo, total
	if marked {
		f.reserved -= amount
	}
	return nil
}

// credited returns the balance of t after it takes amount, or
// ErrBalanceOverflow when it passes int64, or MaxMoney for a non-boundary
// account.
func (l *Ledger) credited(t *account, amount int64) (int64, error) {
	n, ok := checked.Add(t.balance, amount)
	if !ok || (t.kind != KindBoundary && n > l.maxMoney) {
		return 0, ErrBalanceOverflow
	}
	return n, nil
}

// flowSlot is the flow slot at offset off in the region a flow is attributed
// to: the non-boundary side's, and the debited account's for an internal flow.
func (l *Ledger) flowSlot(f, t *account, off int) int {
	region := f.region
	if f.kind == KindBoundary {
		region = t.region
	}
	return region*l.stride + off
}

// ReserveMoney earmarks amount of an account's available money so that an
// order placed now can be settled later (spec-ledger.md 6.1). It is refused
// with ErrInvalidAmount, ErrUnknownAccount or ErrClosedAccount,
// ErrBoundaryReserve for a boundary account, and ErrInsufficientFunds when
// more than the available amount is asked.
func (l *Ledger) ReserveMoney(id int, amount int64) error {
	if amount < 1 {
		return ErrInvalidAmount
	}
	a, err := l.open(id)
	if err != nil {
		return err
	}
	switch {
	case a.kind == KindBoundary:
		return ErrBoundaryReserve
	case a.balance-a.reserved < amount:
		return ErrInsufficientFunds
	}
	a.reserved += amount
	return nil
}

// ReleaseMoney returns amount of an account's reservation to available
// (spec-ledger.md 6.2). Releasing more than is reserved is refused with
// ErrReservationExceeds.
func (l *Ledger) ReleaseMoney(id int, amount int64) error {
	if amount < 1 {
		return ErrInvalidAmount
	}
	a, err := l.open(id)
	if err != nil {
		return err
	}
	if a.reserved < amount {
		return ErrReservationExceeds
	}
	a.reserved -= amount
	return nil
}
