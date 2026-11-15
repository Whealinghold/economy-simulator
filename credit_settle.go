package economysim

import (
	"cmp"
	"slices"

	"github.com/Whealinghold/economy-sim/internal/checked"
	crd "github.com/Whealinghold/economy-sim/internal/credit"
	"github.com/Whealinghold/economy-sim/internal/ledger"
	"github.com/Whealinghold/economy-sim/internal/ordering"
)

// This file is the claims in life of the credit module (the continuation of
// credit.go, architecture.md 2.6; spec-credit.md 8 and 9): the settlement pass at
// P1.7a and the step of a removal that pays a removed debtor's claims from its whole
// balance (2d) and passes a removed creditor's claims to its estate (2e). It takes
// no word of the stream: every tie is the stateless key K(t, claim ID) (FR-CRD-062).
// A refusal of the ledger after the checks is a violated invariant (plan 3-crd D7).

// settleRow is the sums of one region over a pass, the fields of CreditSettled and
// the count that feeds the run of the Warn "claims overdue". Only claims of origin
// loan are in it (spec-credit.md 3.1.5).
type settleRow struct {
	due, repaid, defaulted, writtenOff                     int64
	principal, interest, lost, shielded, headroom, arrears int64
	received, receivedEstate                               int64
	overdueClaims                                          int64
}

// quiet reports whether the row holds no count and no sum that CreditSettled
// carries (spec-credit.md 16.1.1).
func (r settleRow) quiet() bool {
	r.overdueClaims = 0
	return r == settleRow{}
}

// debtorRow is a debtor of the pass: its account, its balance recorded at the entry
// (spec-credit.md 8.1.1.1) and its claims, the slice [lo, hi) of the visits.
type debtorRow struct {
	account, lo, hi int
	balance         int64
}

// visit is a claim the pass settles, as it was before the payment, and the parts
// paid on it.
type visit struct {
	c            ledger.Claim
	paidI, paidP int64
}

// settleScratch is the scratch of the pass and of a waterfall, grown once and
// reused, so that neither allocates in a step once it has grown (NFR-CRD-001).
type settleScratch struct {
	accounts []int
	debtors  []debtorRow
	visits   []visit
	owed     []int64
	tie      []uint64
	shares   []int64
	ids      []int64
	alloc    crd.Scratch
	rows     []settleRow
}

// mustClaim panics when the ledger refuses an operation of the module after the
// checks that make it legal: a refusal is a violated invariant, never an input.
func mustClaim(err error, step string) {
	if err != nil {
		invariant("the ledger refused " + step + " of a checked claim: " + err.Error())
	}
}

// accountView reads an account of a claim, which is open while the claim is.
func (w *World) accountView(id int) ledger.View {
	v, err := w.ledger.Account(id)
	if err != nil {
		invariant("the account of a claim is unreadable: " + err.Error())
	}
	return v
}

// p17aSettle is P1.7a (spec-credit.md 8, FR-CRD-030): for each debtor with a claim
// due this tick or overdue, in ascending agent ID, the pot, the allocation and the
// payments, each from the balance recorded at the entry; then, in ascending claim
// ID, the ends, the defaults and the write-offs; then the events. It runs after the
// money flows and before the outside's walk, so a due claim is paid before any
// order of the tick (FR-CRD-037). It draws no word and returns no error.
func (w *World) p17aSettle() {
	cr, t := w.credit, w.Tick()
	p := &cr.pass
	clear(p.rows)
	w.gatherDebtors(t)
	if len(p.debtors) > 0 {
		if cr.cfg.ShieldTicks > 0 {
			w.valuePrices()
		}
		for i := range p.debtors {
			w.payDebtor(t, &p.debtors[i])
		}
		w.endClaims(t)
	}
	w.reportSettled(t)
}

// gatherDebtors fills the debtors of the pass in ascending account ID, which is
// ascending agent ID, each with its balance and its due and overdue claims in
// ascending claim ID. The claims due at t and the overdue claims are found from the
// indexes, not by a scan of the open claims (FR-CRD-030, NFR-CRD-003).
func (w *World) gatherDebtors(t int64) {
	p, x := &w.credit.pass, &w.credit.idx
	p.accounts, p.debtors, p.visits = p.accounts[:0], p.debtors[:0], p.visits[:0]
	for _, id := range *x.bucket(t) {
		if c := w.indexed(id); c.Due == t {
			p.accounts = append(p.accounts, c.Debtor)
		}
	}
	for _, id := range x.overdue {
		p.accounts = append(p.accounts, w.indexed(id).Debtor)
	}
	slices.Sort(p.accounts)
	for _, acct := range slices.Compact(p.accounts) {
		d := debtorRow{account: acct, lo: len(p.visits), balance: w.accountView(acct).Balance}
		for _, id := range x.byDebtor[acct] {
			if c := w.indexed(id); c.Due == t || c.Overdue >= 0 {
				p.visits = append(p.visits, visit{c: c})
			}
		}
		d.hi = len(p.visits)
		p.debtors = append(p.debtors, d)
	}
}

// indexed reads an indexed claim from the ledger's list; the indexes name open
// claims only.
func (w *World) indexed(id int64) ledger.Claim {
	c, ok := w.ledger.Claim(id)
	if !ok {
		invariant("an indexed claim is not in the ledger's list")
	}
	return c
}

// payDebtor settles the due and overdue claims of one debtor (spec-credit.md 8.2,
// 8.3): the pot of the recorded balance less the shield, shared among the claims
// pro rata by what each owes with the largest remainder, each share paid interest
// first.
func (w *World) payDebtor(t int64, d *debtorRow) {
	p, vs := &w.credit.pass, w.credit.pass.visits[d.lo:d.hi]
	p.owed, p.tie = p.owed[:0], p.tie[:0]
	var total int64
	loans := false
	for i := range vs {
		c := &vs[i].c
		o := c.PrincipalOwed + c.InterestOwed // each part is at most MaxMoney, so the sum fits
		var ok bool
		if total, ok = checked.Add(total, o); !ok {
			invariant("the owed of a debtor's claims does not fit in int64")
		}
		p.owed, p.tie = append(p.owed, o), append(p.tie, ordering.Key(t, c.ID))
		loans = loans || c.Origin == ledger.OriginLoan
	}
	view := w.accountView(d.account)
	var upkeep int64
	if w.credit.cfg.ShieldTicks > 0 {
		upkeep = w.creditUpkeep(&w.agents[view.Owner])
	}
	pay, ok := crd.Pay(d.balance, w.credit.cfg.ShieldTicks, upkeep, total, MaxMoney)
	if !ok {
		invariant("a negative figure in the pot of a debtor")
	}
	if loans {
		p.rows[view.Region].shielded += min(d.balance, total) - pay
	}
	if p.shares, ok = p.alloc.Allocate(p.shares, pay, p.owed, p.tie); !ok {
		invariant("the allocation of a debtor's payment was refused")
	}
	for i := range vs {
		w.tallyPaid(&vs[i], w.payClaim(&vs[i], p.shares[i], view), view)
	}
}

// payClaim pays one claim its share, cut to the creditor's headroom
// (spec-credit.md 8.3, 8.4): interest first, then principal. A part of 0 moves no
// money, and a share cut to 0 is no settlement. The interest is a cost of the
// debtor and unearned income of an agent creditor in the welfare record of the tick
// (spec-credit.md 11.1, FR-CRD-054). It returns the money cut at the creditor's
// headroom.
func (w *World) payClaim(v *visit, share int64, debtor ledger.View) (cut int64) {
	c := &v.c
	ip, pp, ok := crd.Split(share, c.InterestOwed)
	if !ok {
		invariant("a negative share in a settlement")
	}
	cv := w.accountView(c.Creditor)
	if ip, pp, cut, ok = crd.Fit(ip, pp, MaxMoney-cv.Balance); !ok {
		invariant("a negative figure in the headroom of a creditor")
	}
	if ip+pp > 0 {
		mustClaim(w.ledger.Settle(c.ID, ip, pp), "the payment")
	}
	v.paidI, v.paidP = ip, pp
	if ip > 0 {
		w.welfare.addInputSpend(debtor.Owner, ip)
		if cv.Kind == ledger.KindAgent {
			w.welfare.addUnearned(cv.Owner, ip)
		}
	}
	return cut
}

// tallyPaid adds the payment of a claim of origin loan to the sums of the debtor's
// region and of the creditor's (spec-credit.md 16.1).
func (w *World) tallyPaid(v *visit, cut int64, debtor ledger.View) {
	if v.c.Origin != ledger.OriginLoan {
		return
	}
	rows, cv := w.credit.pass.rows, w.accountView(v.c.Creditor)
	rows[debtor.Region].principal += v.paidP
	rows[debtor.Region].interest += v.paidI
	rows[debtor.Region].headroom += cut
	if cv.Kind == ledger.KindEstate {
		rows[cv.Region].receivedEstate += v.paidI + v.paidP
	} else {
		rows[cv.Region].received += v.paidI + v.paidP
	}
}

// endClaims is the second part of the pass (spec-credit.md 8.5): for each visited
// claim in ascending claim ID, the end of a claim paid in full, the default of a
// claim still owed at its due tick, and the write-off of an overdue claim at its
// write-off tick.
func (w *World) endClaims(t int64) {
	p := &w.credit.pass
	slices.SortFunc(p.visits, func(a, b visit) int { return cmp.Compare(a.c.ID, b.c.ID) })
	for i := range p.visits {
		w.endClaim(t, &p.visits[i])
	}
}

// endClaim decides the fate of one visited claim after the payments.
func (w *World) endClaim(t int64, v *visit) {
	cr, c := w.credit, v.c
	row := &cr.pass.rows[w.accountView(c.Debtor).Region]
	loan := c.Origin == ledger.OriginLoan
	if loan && c.Due == t {
		row.due++
	}
	now, open := w.ledger.Claim(c.ID)
	if !open {
		cr.idx.remove(c)
		if loan {
			row.repaid++
		}
		w.reportLoanEnded(t, &c, endRec{cause: CauseRepaid})
		return
	}
	if now.Overdue < 0 {
		mustClaim(w.ledger.MarkOverdue(c.ID, t), "the mark of an overdue claim")
		cr.idx.markOverdue(now)
		now.Overdue = t
		if loan {
			row.defaulted++
		}
		w.logOverdue(&now)
	}
	if wo := cr.cfg.WriteOffTicks; wo > 0 && t >= now.Due+wo {
		lp, li, err := w.ledger.WriteOff(c.ID)
		mustClaim(err, "the write-off")
		cr.idx.remove(now)
		if loan {
			row.writtenOff++
			row.lost += lp + li
		}
		w.reportLoanEnded(t, &now, endRec{cause: CauseWrittenOff, lostP: lp, lostI: li})
	}
}

// reportSettled closes the pass: the owed on the overdue claims of each region, the
// run of the Warn "claims overdue" of each region and the event CreditSettled of
// each region that has a count or a sum, in ascending RegionID, after the LoanEnded
// events of the pass (spec-credit.md 16.1.1, 16.2).
func (w *World) reportSettled(t int64) {
	cr := w.credit
	rows := cr.pass.rows
	for _, id := range cr.idx.overdue {
		if c := w.indexed(id); c.Origin == ledger.OriginLoan {
			r := &rows[w.accountView(c.Debtor).Region]
			r.arrears += c.PrincipalOwed + c.InterestOwed
			r.overdueClaims++
		}
	}
	for g := range rows {
		run, warn := crd.NextRun(cr.overdueRun[g], rows[g].overdueClaims > 0)
		cr.overdueRun[g] = run
		if warn {
			w.logArrears(RegionID(g), &rows[g])
		}
		if w.obs.supplied && !rows[g].quiet() {
			w.obs.deliver(rows[g].event(t, RegionID(g)))
		}
	}
}

// dropClaims is the step of a removal for the credit module (spec-agents.md 3.6
// steps 2d and 2e, spec-credit.md 9): the removed agent's claims as debtor are paid
// from its whole balance and end, then its claims as creditor pass to the estate of
// its region. It runs before the money moves to the estate, so a creditor comes
// first (FR-CRD-040, 041). It is nil-safe and has no result: a refusal is an
// invariant (plan 3-crd D7). A reduction of weight does not call it.
func (w *World) dropClaims(a *agent) {
	if w.credit == nil {
		return
	}
	t := w.Tick()
	w.waterfall(t, a)
	w.passClaims(t, a)
}

// claimIDs copies the claim IDs of an account's list into the scratch, so that the
// index can change while the claims are visited.
func (w *World) claimIDs(table [][]int64, account int) []int64 {
	p := &w.credit.pass
	p.ids = p.ids[:0]
	if account < len(table) {
		p.ids = append(p.ids, table[account]...)
	}
	return p.ids
}

// waterfall is step 2d (spec-credit.md 9.1.1): the whole balance, with no shield, is
// shared among the removed debtor's claims as the settlement shares a pot, each
// share cut to the creditor's headroom; every claim then ends with the cause
// debtor_removed, what is still owed written off, in ascending claim ID.
func (w *World) waterfall(t int64, a *agent) {
	p := &w.credit.pass
	ids := w.claimIDs(w.credit.idx.byDebtor, a.account)
	if len(ids) == 0 {
		return
	}
	p.visits, p.owed, p.tie = p.visits[:0], p.owed[:0], p.tie[:0]
	for _, id := range ids {
		c := w.indexed(id)
		p.visits = append(p.visits, visit{c: c})
		p.owed = append(p.owed, c.PrincipalOwed+c.InterestOwed)
		p.tie = append(p.tie, ordering.Key(t, c.ID))
	}
	view := w.accountView(a.account)
	shares, _, ok := p.alloc.Waterfall(p.shares, view.Balance, p.owed, p.tie)
	if !ok {
		invariant("the waterfall of a removed debtor was refused")
	}
	p.shares = shares
	for i := range p.visits {
		v := &p.visits[i]
		w.payClaim(v, shares[i], view) // the cut at a headroom is the claim's loss, written off below
		w.endRemoved(t, v)
	}
}

// endRemoved ends a claim of a removed debtor: what is still owed is written off,
// and LoanEnded carries what the waterfall paid and what was lost.
func (w *World) endRemoved(t int64, v *visit) {
	var lp, li int64
	if _, open := w.ledger.Claim(v.c.ID); open {
		var err error
		lp, li, err = w.ledger.WriteOff(v.c.ID)
		mustClaim(err, "the write-off of a removed debtor's claim")
	}
	w.credit.idx.remove(v.c)
	w.reportLoanEnded(t, &v.c, endRec{cause: CauseDebtorRemoved, paidP: v.paidP, paidI: v.paidI, lostP: lp, lostI: li})
}

// passClaims is step 2e (spec-credit.md 9.2): each claim of which the removed agent
// is the creditor becomes the estate account's of the region of its account, in
// ascending claim ID, with no money moved.
func (w *World) passClaims(t int64, a *agent) {
	ids := w.claimIDs(w.credit.idx.byCreditor, a.account)
	if len(ids) == 0 {
		return
	}
	region := w.regionOf(a)
	estate := w.estates.account(region)
	for _, id := range ids {
		c := w.indexed(id)
		mustClaim(w.ledger.Reassign(id, estate), "the passing of a claim")
		w.credit.idx.reassign(c, estate)
		w.reportPassed(t, &c, a, region)
	}
}
