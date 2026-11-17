package economysim

import (
	"fmt"
	"slices"

	"github.com/Whealinghold/economy-sim/internal/codec"
	crd "github.com/Whealinghold/economy-sim/internal/credit"
	"github.com/Whealinghold/economy-sim/internal/ledger"
)

// This file is the state of the credit module that is derived or saved (the
// continuation of credit.go, architecture.md 2.6; spec-credit.md 17): the indexes of
// the claim list that make the settlement proportional to the claims due and
// overdue (NFR-CRD-003), the figures of one agent's claims, the audit of the
// indexes, and the module's section of the checkpoint, tag 26. The claims
// themselves are the ledger's list, saved in tag 2 (FR-CRD-088).

// claimIndex is the derived indexes of the open-claim list (spec-credit.md 17.3),
// every list of claim IDs ascending. due is a ring of MaxCreditTerm + 1 buckets by
// Due modulo the ring: a claim whose Due is further than that from the tick shares a
// bucket with an earlier tick and is skipped there by its Due, so a payout claim of
// any term is found too.
type claimIndex struct {
	// byDebtor and byCreditor are the claim IDs of each account, by account ID.
	byDebtor, byCreditor [][]int64
	// due holds the claims that are not overdue, by Due; overdue those that are.
	due     [][]int64
	overdue []int64
}

// newClaimIndex returns the empty indexes.
func newClaimIndex() claimIndex {
	return claimIndex{due: make([][]int64, MaxCreditTerm+1)}
}

// insertID inserts id in the ascending list, once.
func insertID(list []int64, id int64) []int64 {
	i, found := slices.BinarySearch(list, id)
	if found {
		return list
	}
	return slices.Insert(list, i, id)
}

// removeID removes id from the ascending list when it is there.
func removeID(list []int64, id int64) []int64 {
	i, found := slices.BinarySearch(list, id)
	if !found {
		return list
	}
	return slices.Delete(list, i, i+1)
}

// ofAccount returns the list of an account of the table, growing it to hold one.
func ofAccount(table *[][]int64, account int) *[]int64 {
	if account >= len(*table) {
		*table = append(*table, make([][]int64, account+1-len(*table))...)
	}
	return &(*table)[account]
}

// bucket is the ring bucket of a due tick.
func (x *claimIndex) bucket(due int64) *[]int64 { return &x.due[due%int64(len(x.due))] }

// add indexes an open claim: by its two accounts and by its due tick, or in the
// overdue list when it is already overdue.
func (x *claimIndex) add(c ledger.Claim) {
	d, k := ofAccount(&x.byDebtor, c.Debtor), ofAccount(&x.byCreditor, c.Creditor)
	*d, *k = insertID(*d, c.ID), insertID(*k, c.ID)
	if c.Overdue >= 0 {
		x.overdue = insertID(x.overdue, c.ID)
		return
	}
	b := x.bucket(c.Due)
	*b = insertID(*b, c.ID)
}

// remove drops a claim that ended from every index. c is the claim as it was
// indexed (its Due and whether it was overdue decide where it is).
func (x *claimIndex) remove(c ledger.Claim) {
	d, k := ofAccount(&x.byDebtor, c.Debtor), ofAccount(&x.byCreditor, c.Creditor)
	*d, *k = removeID(*d, c.ID), removeID(*k, c.ID)
	x.overdue = removeID(x.overdue, c.ID)
	b := x.bucket(c.Due)
	*b = removeID(*b, c.ID)
}

// reassign moves a claim from its creditor's list to the new creditor's.
func (x *claimIndex) reassign(c ledger.Claim, creditor int) {
	ofAccount(&x.byCreditor, max(c.Creditor, creditor)) // grow once: a second growth would strand the first pointer
	old, now := ofAccount(&x.byCreditor, c.Creditor), ofAccount(&x.byCreditor, creditor)
	*old, *now = removeID(*old, c.ID), insertID(*now, c.ID)
}

// markOverdue moves a claim from its due bucket to the overdue list.
func (x *claimIndex) markOverdue(c ledger.Claim) {
	b := x.bucket(c.Due)
	*b = removeID(*b, c.ID)
	x.overdue = insertID(x.overdue, c.ID)
}

// rebuild replaces the indexes by those of the claims, which are in ascending ID.
func (x *claimIndex) rebuild(claims []ledger.Claim) {
	for i := range x.byDebtor {
		x.byDebtor[i] = x.byDebtor[i][:0]
	}
	for i := range x.byCreditor {
		x.byCreditor[i] = x.byCreditor[i][:0]
	}
	for i := range x.due {
		x.due[i] = x.due[i][:0]
	}
	x.overdue = x.overdue[:0]
	for _, c := range claims {
		x.add(c)
	}
}

// listed reports whether the ascending list holds id.
func listed(list []int64, id int64) bool {
	_, found := slices.BinarySearch(list, id)
	return found
}

// check compares the indexes with the claim list, which is in ascending ID: every
// claim is in the list of its debtor, in that of its creditor and in the bucket of
// its due tick or the overdue list, and the indexes hold nothing else
// (spec-credit.md 17.3). It names the first claim that fails.
func (x *claimIndex) check(claims []ledger.Claim) error {
	for _, c := range claims {
		if err := x.holds(c); err != nil {
			return err
		}
	}
	if d, k, n := x.entries(); d != len(claims) || k != len(claims) || n != len(claims) {
		return fmt.Errorf("the indexes hold %d, %d and %d entries for %d open claims", d, k, n, len(claims))
	}
	return nil
}

// holds checks that the indexes list one claim where it belongs.
func (x *claimIndex) holds(c ledger.Claim) error {
	switch {
	case c.Debtor >= len(x.byDebtor) || !listed(x.byDebtor[c.Debtor], c.ID):
		return fmt.Errorf("claim %d is not in the index of its debtor account %d", c.ID, c.Debtor)
	case c.Creditor >= len(x.byCreditor) || !listed(x.byCreditor[c.Creditor], c.ID):
		return fmt.Errorf("claim %d is not in the index of its creditor account %d", c.ID, c.Creditor)
	case c.Overdue >= 0 && !listed(x.overdue, c.ID):
		return fmt.Errorf("claim %d is overdue and not in the overdue list", c.ID)
	case c.Overdue < 0 && !listed(*x.bucket(c.Due), c.ID):
		return fmt.Errorf("claim %d is not in the bucket of its due tick %d", c.ID, c.Due)
	}
	return nil
}

// entries counts what the three kinds of index hold: the claims of the debtor
// lists, of the creditor lists, and of the due buckets and the overdue list.
func (x *claimIndex) entries() (debtors, creditors, dated int) {
	for _, l := range x.byDebtor {
		debtors += len(l)
	}
	for _, l := range x.byCreditor {
		creditors += len(l)
	}
	for _, l := range x.due {
		dated += len(l)
	}
	return debtors, creditors, dated + len(x.overdue)
}

// claimFigures is what one agent owes and is owed as a debtor and as a creditor
// (the fields of the Credit group of the view, spec-credit.md 10.1), read from the
// indexes. NextDue is -1 when it owes nothing.
type claimFigures struct {
	// Debt is the principal owed as debtor, Owed principal and interest owed, Overdue
	// the owed on the overdue claims and NextDue the earliest due tick.
	Debt, Owed, Overdue, NextDue int64
	// Lent is the principal owed to the agent as creditor and Receivable the
	// principal and interest.
	Lent, Receivable int64
}

// figuresOf reads the figures of an account from the indexes in time proportional
// to its claims. It is nil-safe: a world without the module has no claims.
func (w *World) figuresOf(account int) claimFigures {
	f := claimFigures{NextDue: -1}
	if w.credit == nil {
		return f
	}
	x := &w.credit.idx
	if account < len(x.byDebtor) {
		for _, id := range x.byDebtor[account] {
			c := w.indexed(id)
			f.Debt += c.PrincipalOwed
			f.Owed += c.PrincipalOwed + c.InterestOwed
			if c.Overdue >= 0 {
				f.Overdue += c.PrincipalOwed + c.InterestOwed
			}
			if f.NextDue < 0 || c.Due < f.NextDue {
				f.NextDue = c.Due
			}
		}
	}
	if account < len(x.byCreditor) {
		for _, id := range x.byCreditor[account] {
			c := w.indexed(id)
			f.Lent += c.PrincipalOwed
			f.Receivable += c.PrincipalOwed + c.InterestOwed
		}
	}
	return f
}

// creditRoom is the exposure room of a borrower agent (spec-credit.md 5.3.3): the
// cap of its block's MaxDebtTicks of upkeep cost less its debt, 0 for an agent with
// no Borrow block. The prices of the tick must be valued (valuePrices).
func (w *World) creditRoom(a *agent, debt int64) int64 {
	t := w.cfg.Roles[a.role].Credit
	if t == nil || t.Borrow == nil {
		return 0
	}
	return w.creditRoomAt(t.Borrow, w.creditUpkeep(a), debt)
}

// creditRoomAt is the exposure room of a borrow block at an upkeep cost and a debt
// (spec-credit.md 5.3.3), for a caller that has the upkeep cost already.
func (w *World) creditRoomAt(b *BorrowTerms, upkeep, debt int64) int64 {
	room, ok := crd.Room(b.MaxDebtTicks, upkeep, debt, MaxMoney)
	if !ok {
		invariant("a negative figure in the exposure room of an agent")
	}
	return room
}

// creditUpkeep is the upkeep cost U of an agent (spec-credit.md 5.3.1): the cost of
// its role's upkeep rows in its region at the prices valued for the tick, weight
// included, the figure relief reads at P1.6. The prices must be valued
// (valuePrices).
func (w *World) creditUpkeep(a *agent) int64 {
	g, nc := w.regionOf(a), len(w.cfg.Commodities)
	return w.welfare.cost.tick(w.roles.row(a.role).upkeepIn(g), w.oc.valued[int(g)*nc:(int(g)+1)*nc], a.weight)
}

// auditCredit is the audit of the module (spec-credit.md 3.6, FR-CRD-082): the
// identities of the claim list are the ledger's, and this adds that the indexes are
// the list. It is nil with the module off and changes nothing.
func (w *World) auditCredit() error {
	if w.credit == nil {
		return nil
	}
	if err := w.credit.idx.check(w.ledger.AppendClaims(nil)); err != nil {
		return fmt.Errorf("credit indexes: %w", err)
	}
	return nil
}

// saveCredit is tag 26 (spec-credit.md 17.1): the rate scales that are not 10 000,
// as (region, basis points) in ascending region, the run of the Warn "claims
// overdue" of each region with a run, as (region, run), and the run of "claim cap
// reached". The claims are in tag 2.
func (w *World) saveCredit(cw *codec.Writer) error {
	c := w.credit
	n := 0
	for _, s := range c.scale {
		if s != wholeBP {
			n++
		}
	}
	cw.Count(n)
	for g, s := range c.scale {
		if s != wholeBP {
			cw.U(uint64(g))
			cw.U(unsigned(s))
		}
	}
	writeSparseByRegion(cw, c.overdueRun)
	cw.U(unsigned(c.capRun))
	return nil
}

// loadCredit reads tag 26 into the state a world with the module has built, which
// holds the defaults: a scale in strictly ascending region order that is 10 000 is
// corrupt (it is a default and is not written), one above MaxShockBP is out of
// bounds, and a run is from 1 to the tick.
func (w *World) loadCredit(rs *restoreCtx, r *codec.Reader) error {
	c := w.credit
	n := r.Count(2)
	prev := uint64(0)
	for i := range n {
		g := r.Index(uint64(len(c.scale)))
		s := r.Bounded("rate scale", MaxShockBP)
		if r.Err() != nil {
			return nil
		}
		if s == wholeBP || (i > 0 && g <= prev) {
			r.Failf("rate scale row (region %d) is at its default or out of order", g)
			return nil
		}
		c.scale[g], prev = s, g
	}
	readSparseByRegion(r, c.overdueRun, rs.tick)
	c.capRun = r.Bounded("run", max(rs.tick, 0))
	return nil
}

// creditCross rebuilds the derived indexes from the ledger's list, which tag 2 has
// loaded and whose accounts and agents are in, and checks what spec-credit.md 17.5
// asks of the claims that the ledger's own audit does not: the due tick after the
// formation, the formation not after the tick, and an overdue tick that is -1 or in
// [Due, tick). Every failure names the claim.
func (w *World) creditCross() error {
	claims := w.ledger.AppendClaims(nil)
	for _, c := range claims {
		if err := claimTimesFault(c, w.Tick()); err != nil {
			return fmt.Errorf("%w: %w", ErrCheckpointCorrupt, err)
		}
	}
	w.credit.idx.rebuild(claims)
	return nil
}

// claimTimesFault checks the ticks of a restored claim against the tick of the
// boundary (spec-credit.md 17.5): the due tick after the formation, the formation not
// after the tick, and an overdue tick that is -1 or in [Due, tick).
func claimTimesFault(c ledger.Claim, tick int64) error {
	switch {
	case c.Due <= c.Formed:
		return fmt.Errorf("claim %d is due at %d, not after its formation %d", c.ID, c.Due, c.Formed)
	case c.Formed > tick:
		return fmt.Errorf("claim %d was formed at %d, after the tick %d", c.ID, c.Formed, tick)
	case c.Overdue != -1 && (c.Overdue < c.Due || c.Overdue >= tick):
		return fmt.Errorf("claim %d is overdue from %d, want -1 or a tick in [%d, %d)", c.ID, c.Overdue, c.Due, tick)
	}
	return nil
}

// creditAgents counts the agents whose role has a Lend block and those whose role has
// a Borrow block, for the construction summary (spec-credit.md 16.2).
func (w *World) creditAgents() (lenders, borrowers int64) {
	for i := w.firstHousehold; i < len(w.agents); i++ {
		a := &w.agents[i]
		if a.weight < 1 {
			continue
		}
		if t := w.cfg.Roles[a.role].Credit; t != nil {
			if t.Lend != nil {
				lenders++
			}
			if t.Borrow != nil {
				borrowers++
			}
		}
	}
	return lenders, borrowers
}
