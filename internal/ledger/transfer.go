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

func (l *Ledger) transfer(from, to int, amount int64, r Reason, marked bool) error {
	if amount < 1 {
		return ErrInvalidAmount
	}
	f, err := l.open(from)
	if err != nil {
		return err
	}
	t, err := l.open(to)
	if err != nil {
		return err
	}
	if from == to || !l.moneyFits(r, f, t) {
		return ErrReasonMismatch
	}
	for _, b := range [2]*account{f, t} {
		if b.kind == KindBoundary && (b.reason != r || b.thing != Money) {
			return ErrReasonMismatch
		}
	}
	// The debit.
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
	newFrom, ok := checked.Sub(f.balance, amount)
	if !ok {
		return ErrBalanceOverflow
	}
	// The credit.
	newTo, ok := checked.Add(t.balance, amount)
	if !ok || (t.kind != KindBoundary && newTo > l.maxMoney) {
		return ErrBalanceOverflow
	}
	// The flow total.
	region := f.region
	if f.kind == KindBoundary {
		region = t.region
	}
	slot := region*l.stride + l.off[r][thingMoney]
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
