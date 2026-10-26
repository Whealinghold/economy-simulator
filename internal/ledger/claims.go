package ledger

import (
	"slices"

	"github.com/Whealinghold/economy-sim/internal/checked"
)

// The origins of a claim (spec-credit.md 3.1.5), fixed at formation.
const (
	// OriginLoan is a claim formed by Lend.
	OriginLoan = 0
	// OriginPayout is a claim formed by Obligate.
	OriginPayout = 1
)

// Claim is one claim of the open-claim list, exactly the record of
// spec-credit.md 3.1.1. The ledger knows accounts and numbers, so the parties
// are account IDs.
type Claim struct {
	// ID is assigned in order of formation and never reused.
	ID int64
	// Creditor is an Agent account or an Estate account, Debtor an Agent
	// account, and they differ.
	Creditor, Debtor int
	// Principal and Interest are fixed at formation.
	Principal, Interest int64
	// PrincipalOwed and InterestOwed are what remains, in [0, Principal] and
	// [0, Interest], and not both 0.
	PrincipalOwed, InterestOwed int64
	// RateBP is the effective yearly rate at formation, kept for reports.
	RateBP int64
	// Formed is the tick of formation, Due the tick the whole falls due.
	Formed, Due int64
	// Overdue is the tick the claim first went unpaid, or -1 while it has not.
	Overdue int64
	// Origin is OriginLoan or OriginPayout.
	Origin int
}

// ClaimTotals are the seven cumulative totals of spec-credit.md 3.3.2. They
// are never reset, and make the identity of 3.6 exact: the principal owed
// over the open claims is Lent + Awarded - Repaid - LostPrincipal, and the
// interest owed plus InterestPaid plus LostInterest is InterestFixed.
type ClaimTotals struct {
	Lent, Awarded, Repaid       int64
	InterestPaid, InterestFixed int64
	LostPrincipal, LostInterest int64
}

// Sheet is the balance sheet of one non-boundary account (spec-ledger.md 9.2):
// measurement only, in plain integers.
type Sheet struct {
	// Money is the account's balance, reservations included.
	Money int64
	// Goods is every unit held, earmarked ones included, at the supplied
	// prices.
	Goods int64
	// ClaimsHeld is the amount owed, undiscounted, on the open claims whose
	// creditor is the account, and ClaimsOwed that on the claims whose
	// debtor it is.
	ClaimsHeld, ClaimsOwed int64
	// Assets is Money + Goods + ClaimsHeld, and NetWorth is Assets less
	// ClaimsOwed, which may be negative.
	Assets, NetWorth int64
}

// ClaimTotals returns the cumulative totals.
func (l *Ledger) ClaimTotals() ClaimTotals { return l.claimTot }

// AppendClaims appends the open claims, in ascending ID, to dst and returns
// it. It allocates only when dst has no room.
func (l *Ledger) AppendClaims(dst []Claim) []Claim { return append(dst, l.claims...) }

// findClaim returns the index in the open-claim list of the claim with the ID.
func (l *Ledger) findClaim(id int64) (int, bool) {
	return slices.BinarySearchFunc(l.claims, id, func(c Claim, id int64) int {
		switch {
		case c.ID < id:
			return -1
		case c.ID > id:
			return 1
		}
		return 0
	})
}

// inClaim reports whether the account is the creditor or the debtor of an
// open claim.
func (l *Ledger) inClaim(id int) bool {
	for i := range l.claims {
		if l.claims[i].Creditor == id || l.claims[i].Debtor == id {
			return true
		}
	}
	return false
}

// parties returns the two accounts of a claim, the creditor's checked first:
// ErrUnknownAccount, then ErrClosedAccount.
func (l *Ledger) parties(creditor, debtor int) (c, d *account, err error) {
	if c, err = l.open(creditor); err != nil {
		return nil, nil, err
	}
	if d, err = l.open(debtor); err != nil {
		return nil, nil, err
	}
	return c, d, nil
}

// Lend transfers principal from the creditor to the debtor under LoanOut and
// appends a claim of origin OriginLoan with the next ID, PrincipalOwed
// principal, InterestOwed interest, Formed the current tick and Overdue -1,
// and returns its ID (spec-credit.md 3.2). It is atomic (2.6). The checks run
// in this order and the first that holds is returned:
//
//  1. ErrInvalidAmount, if principal < 1 or interest < 0;
//  2. ErrUnknownAccount, then ErrClosedAccount, for the creditor and then the
//     debtor;
//  3. ErrClaimShape, for the same account on both sides or a due not after
//     the current tick;
//  4. ErrReasonMismatch, if LoanOut is not declared or the accounts are not
//     both Agent accounts;
//  5. ErrTooManyClaims, if MaxClaims claims are open (a MaxClaims of 0, as in
//     every world with no module that lends, opens none);
//  6. ErrBalanceOverflow, if principal + interest passes MaxMoney or a total
//     would leave int64;
//  7. the refusals of the transfer: ErrInsufficientFunds and
//     ErrBalanceOverflow, for the debtor's balance.
func (l *Ledger) Lend(creditor, debtor int, principal, interest, rateBP, due int64) (int64, error) {
	// #lizard forgives the parameter count: the signature is exported API.
	if principal < 1 || interest < 0 {
		return 0, ErrInvalidAmount
	}
	c, d, err := l.parties(creditor, debtor)
	if err != nil {
		return 0, err
	}
	if err := l.checkLend(creditor, debtor, due, c, d); err != nil {
		return 0, err
	}
	lent, fixed, ok := l.lentTotals(principal, interest)
	if !ok {
		return 0, ErrBalanceOverflow
	}
	if err := l.transfer(creditor, debtor, principal, LoanOut, false); err != nil {
		return 0, err
	}
	// Nothing below can fail.
	id := l.nextClaim
	l.nextClaim++
	l.claims = append(l.claims, Claim{
		ID: id, Creditor: creditor, Debtor: debtor,
		Principal: principal, Interest: interest,
		PrincipalOwed: principal, InterestOwed: interest,
		RateBP: rateBP, Formed: l.tick, Due: due, Overdue: -1, Origin: OriginLoan,
	})
	l.claimTot.Lent, l.claimTot.InterestFixed = lent, fixed
	return id, nil
}

// checkLend makes checks 3 to 5 of Lend, in that order, once the accounts are
// found.
func (l *Ledger) checkLend(creditor, debtor int, due int64, c, d *account) error {
	if l.sameOrDue(creditor, debtor, due) {
		return ErrClaimShape
	}
	if !l.moneyFits(LoanOut, c, d) {
		return ErrReasonMismatch
	}
	if len(l.claims) >= l.maxClaims {
		return ErrTooManyClaims
	}
	return nil
}

// sameOrDue reports whether a claim has the same account on both sides or a
// due not after the current tick.
func (l *Ledger) sameOrDue(creditor, debtor int, due int64) bool {
	return creditor == debtor || due <= l.tick
}

// lentTotals returns the Lent and InterestFixed totals after a loan, and false
// when the amount owed passes MaxMoney or a total would leave int64.
func (l *Ledger) lentTotals(principal, interest int64) (lent, fixed int64, ok bool) {
	owed, ok1 := checked.Add(principal, interest)
	lent, ok2 := checked.Add(l.claimTot.Lent, principal)
	fixed, ok3 := checked.Add(l.claimTot.InterestFixed, interest)
	return lent, fixed, ok1 && ok2 && ok3 && owed <= l.maxMoney
}

// Obligate appends a claim of origin OriginPayout with the next ID, Principal
// and PrincipalOwed amount, Interest 0, Formed the current tick and Overdue
// -1, with no transfer, and returns its ID (spec-credit.md 3.2). It is atomic.
// It is refused, in this order, with ErrInvalidAmount (amount below 1),
// ErrUnknownAccount, ErrClosedAccount, ErrClaimShape (the same account on
// both sides, a due not after the current tick, or a creditor that is not an
// Agent or an Estate account or a debtor that is not an Agent account),
// ErrTooManyClaims and ErrBalanceOverflow (the Awarded total would leave
// int64).
func (l *Ledger) Obligate(creditor, debtor int, amount, due int64) (int64, error) {
	if amount < 1 {
		return 0, ErrInvalidAmount
	}
	c, d, err := l.parties(creditor, debtor)
	if err != nil {
		return 0, err
	}
	if l.sameOrDue(creditor, debtor, due) || !setAE.has(c.kind) || d.kind != KindAgent {
		return 0, ErrClaimShape
	}
	if len(l.claims) >= l.maxClaims {
		return 0, ErrTooManyClaims
	}
	awarded, ok := checked.Add(l.claimTot.Awarded, amount)
	if !ok {
		return 0, ErrBalanceOverflow
	}
	id := l.nextClaim
	l.nextClaim++
	l.claims = append(l.claims, Claim{
		ID: id, Creditor: creditor, Debtor: debtor,
		Principal: amount, PrincipalOwed: amount,
		Formed: l.tick, Due: due, Overdue: -1, Origin: OriginPayout,
	})
	l.claimTot.Awarded = awarded
	return id, nil
}

// Settle pays interest and then principal from the claim's debtor to its
// creditor, lowers the owed parts, and removes the claim when both are 0
// (spec-credit.md 3.2). Interest goes under LoanInterest and principal under
// LoanRepay; a part of 0 skips its transfer. It is atomic (2.6). The checks
// run in this order:
//
//  1. ErrUnknownClaim;
//  2. ErrInvalidAmount, for a part below 0 or above what is owed, or both 0;
//  3. ErrReasonMismatch, if a reason of a part to pay is not declared or the
//     accounts do not fit its row;
//  4. ErrInsufficientFunds, for the debtor's available money;
//  5. ErrBalanceOverflow, for the creditor's balance or a flow total.
func (l *Ledger) Settle(claim, interest, principal int64) error {
	i, ok := l.findClaim(claim)
	if !ok {
		return ErrUnknownClaim
	}
	c := &l.claims[i]
	if err := l.checkSettle(c, interest, principal); err != nil {
		return err
	}
	// Nothing below can fail: the checks above are those of transfer.
	if interest > 0 {
		l.mustTransfer(c.Debtor, c.Creditor, interest, LoanInterest)
	}
	if principal > 0 {
		l.mustTransfer(c.Debtor, c.Creditor, principal, LoanRepay)
	}
	c.InterestOwed -= interest
	c.PrincipalOwed -= principal
	l.claimTot.InterestPaid += interest
	l.claimTot.Repaid += principal
	if c.InterestOwed == 0 && c.PrincipalOwed == 0 {
		l.claims = slices.Delete(l.claims, i, i+1)
	}
	return nil
}

// checkSettle makes the checks of Settle after the claim is found, in the
// order its doc comment gives.
func (l *Ledger) checkSettle(c *Claim, interest, principal int64) error {
	if !c.partsValid(interest, principal) {
		return ErrInvalidAmount
	}
	cr, db, err := l.parties(c.Creditor, c.Debtor)
	if err != nil {
		return err
	}
	if !l.settleReasonsFit(cr, db, interest, principal) {
		return ErrReasonMismatch
	}
	// Each part is at most what is owed, and a loan owes at most MaxMoney and
	// a payout claim has no interest, so the sum cannot overflow.
	total := interest + principal
	if db.balance-db.reserved < total {
		return ErrInsufficientFunds
	}
	if n, ok := checked.Add(cr.balance, total); !ok || n > l.maxMoney {
		return ErrBalanceOverflow
	}
	if !l.flowsFit(db.region, interest, principal) {
		return ErrBalanceOverflow
	}
	return nil
}

// settleReasonsFit reports whether the reason of each part to pay is declared
// and fits the accounts.
func (l *Ledger) settleReasonsFit(cr, db *account, interest, principal int64) bool {
	return (interest == 0 || l.moneyFits(LoanInterest, db, cr)) &&
		(principal == 0 || l.moneyFits(LoanRepay, db, cr))
}

// partsValid reports whether the interest and principal parts of a payment
// are each within what is owed and not both 0.
func (c *Claim) partsValid(interest, principal int64) bool {
	return interest >= 0 && principal >= 0 && interest <= c.InterestOwed &&
		principal <= c.PrincipalOwed && (interest != 0 || principal != 0)
}

// flowsFit reports whether the LoanInterest and LoanRepay flow totals of the
// region can take the parts without leaving int64.
func (l *Ledger) flowsFit(region int, interest, principal int64) bool {
	flows := l.flows()
	for _, p := range [2]struct {
		r Reason
		v int64
	}{{LoanInterest, interest}, {LoanRepay, principal}} {
		if p.v == 0 {
			continue
		}
		if _, ok := checked.Add(flows[region*l.stride+l.offOf(p.r, thingMoney)], p.v); !ok {
			return false
		}
	}
	return true
}

// mustTransfer is a transfer whose refusal is a programmer error: the caller
// has made every check of transfer.
func (l *Ledger) mustTransfer(from, to int, amount int64, r Reason) {
	if err := l.transfer(from, to, amount, r, false); err != nil {
		invariantViolated("a claim transfer was refused after its checks: " + err.Error())
	}
}

// WriteOff removes the claim with no transfer, adds what was still owed to
// the LostPrincipal and LostInterest totals, and returns it (spec-credit.md
// 3.2). It is refused with ErrUnknownClaim.
func (l *Ledger) WriteOff(claim int64) (principal, interest int64, err error) {
	i, ok := l.findClaim(claim)
	if !ok {
		return 0, 0, ErrUnknownClaim
	}
	c := l.claims[i]
	// Both totals stay within what was fixed at formation, so they cannot
	// overflow (spec-credit.md 3.6 item 1).
	l.claimTot.LostPrincipal += c.PrincipalOwed
	l.claimTot.LostInterest += c.InterestOwed
	l.claims = slices.Delete(l.claims, i, i+1)
	return c.PrincipalOwed, c.InterestOwed, nil
}

// Reassign makes creditor, an Estate account, the claim's creditor, with no
// transfer (spec-credit.md 3.2). It is refused with ErrUnknownClaim,
// ErrUnknownAccount, ErrClosedAccount, or ErrReasonMismatch for an account
// that is not an Estate account.
func (l *Ledger) Reassign(claim int64, creditor int) error {
	i, ok := l.findClaim(claim)
	if !ok {
		return ErrUnknownClaim
	}
	a, err := l.open(creditor)
	if err != nil {
		return err
	}
	if a.kind != KindEstate {
		return ErrReasonMismatch
	}
	l.claims[i].Creditor = creditor
	return nil
}

// MarkOverdue sets the claim's Overdue to tick when it is -1, and leaves it
// when it is already 0 or more: it is the first tick at which the claim went
// unpaid. It moves no money and changes no other field (spec-credit.md 3.2,
// D9 of plan 1-1). It is refused with ErrUnknownClaim, and ErrClaimShape for
// a tick below the claim's Due.
func (l *Ledger) MarkOverdue(claim, tick int64) error {
	i, ok := l.findClaim(claim)
	if !ok {
		return ErrUnknownClaim
	}
	c := &l.claims[i]
	if tick < c.Due {
		return ErrClaimShape
	}
	if c.Overdue < 0 {
		c.Overdue = tick
	}
	return nil
}

// BalanceSheet returns the balance sheet of a non-boundary account
// (spec-ledger.md 9.2). prices holds the price of one unit of each commodity,
// indexed by commodity, of the caller's choosing; every sum is checked. It is
// refused with ErrUnknownAccount, ErrClosedAccount, ErrInvalidKind for a
// boundary account, ErrUnknownCommodity for a table shorter than the
// commodities, ErrInvalidAmount for a negative price and ErrBalanceOverflow
// for a sum outside int64. It changes nothing.
func (l *Ledger) BalanceSheet(id int, prices []int64) (Sheet, error) {
	a, err := l.sheetAccount(id, prices)
	if err != nil {
		return Sheet{}, err
	}
	s := Sheet{Money: a.balance}
	var ok bool
	if s.Goods, ok = goodsValue(a.lots, prices); !ok {
		return Sheet{}, ErrBalanceOverflow
	}
	if !l.sumClaims(id, &s) {
		return Sheet{}, ErrBalanceOverflow
	}
	if s.Assets, ok = checked.Add(s.Money, s.Goods); ok {
		s.Assets, ok = checked.Add(s.Assets, s.ClaimsHeld)
	}
	if ok {
		s.NetWorth, ok = checked.Sub(s.Assets, s.ClaimsOwed)
	}
	if !ok {
		return Sheet{}, ErrBalanceOverflow
	}
	return s, nil
}

// sheetAccount opens account id for a balance sheet and validates the price
// table, in the order BalanceSheet documents.
func (l *Ledger) sheetAccount(id int, prices []int64) (*account, error) {
	a, err := l.open(id)
	if err != nil {
		return nil, err
	}
	if a.kind == KindBoundary {
		return nil, ErrInvalidKind
	}
	if len(prices) < len(l.comms) {
		return nil, ErrUnknownCommodity
	}
	for _, p := range prices[:len(l.comms)] {
		if p < 0 {
			return nil, ErrInvalidAmount
		}
	}
	return a, nil
}

// goodsValue is the sum of quantity times price over the lots, false on
// overflow.
func goodsValue(lots []Lot, prices []int64) (int64, bool) {
	var sum int64
	for _, lt := range lots {
		v, ok := checked.Mul(lt.Quantity, prices[lt.Commodity])
		if !ok {
			return 0, false
		}
		if sum, ok = checked.Add(sum, v); !ok {
			return 0, false
		}
	}
	return sum, true
}

// sumClaims adds what the claims of account id owe and are owed into
// s.ClaimsHeld and s.ClaimsOwed, false on overflow.
func (l *Ledger) sumClaims(id int, s *Sheet) bool {
	for _, c := range l.claims {
		if c.Creditor != id && c.Debtor != id {
			continue
		}
		owed, ok := checked.Add(c.PrincipalOwed, c.InterestOwed)
		if !ok {
			return false
		}
		if c.Creditor == id {
			s.ClaimsHeld, ok = checked.Add(s.ClaimsHeld, owed)
		} else {
			s.ClaimsOwed, ok = checked.Add(s.ClaimsOwed, owed)
		}
		if !ok {
			return false
		}
	}
	return true
}
