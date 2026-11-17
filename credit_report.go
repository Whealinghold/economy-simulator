package economysim

import (
	"context"
	"log/slog"
	"math"
	"slices"

	"github.com/Whealinghold/economy-sim/internal/checked"
	crd "github.com/Whealinghold/economy-sim/internal/credit"
	"github.com/Whealinghold/economy-sim/internal/ledger"
	"github.com/Whealinghold/economy-sim/internal/logkeys"
)

// This file is the reporting of the credit module (spec-credit.md 15.2; the
// continuation file of credit.go, architecture.md 2.6): the construction warnings.
// Warnings never fail validation (spec-config.md 7.2). Every threshold is a
// placeholder (N-015, N-027). A warning whose arithmetic cannot be done (an
// undeclared commodity, an overflow, a calendar that fails) is not raised.

// creditRoles are the roles of a configuration that lend and that borrow, in the
// order of Config.Roles.
type creditRoles struct {
	lenders, borrowers []*Role
}

// creditRolesOf sorts the roles with a credit block by the block they hold.
func creditRolesOf(c *Config) creditRoles {
	var cr creditRoles
	for i := range c.Roles {
		ro := &c.Roles[i]
		switch {
		case ro.Credit == nil:
		case ro.Credit.Lend != nil:
			cr.lenders = append(cr.lenders, ro)
		case ro.Credit.Borrow != nil:
			cr.borrowers = append(cr.borrowers, ro)
		}
	}
	return cr
}

// creditWarnings returns the warnings of 15.2 that this module raises, unsorted
// (the caller sorts). It is empty with the module off. rate_limit_below_floor is
// the institutions module's.
func creditWarnings(c *Config, lk lookup) []Warning {
	m := c.Modules.Credit
	if m == nil {
		return nil
	}
	cr := creditRolesOf(c)
	out := creditSetWarnings(c, cr)
	for _, ro := range cr.borrowers {
		out = append(out, borrowerWarnings(c, lk, m, cr, ro)...)
	}
	for _, ro := range cr.lenders {
		out = append(out, lenderWarnings(c, lk, m, ro)...)
	}
	return append(out, creditEconomyWarnings(c, m, cr)...)
}

// creditWarn is a warning of the module itself, at modules.credit.
func creditWarn(rule string) Warning {
	return Warning{Path: []string{"modules", "credit"}, Rule: rule}
}

// creditSetWarnings are the warnings about the set of roles: no_credit_roles,
// credit_no_lender, credit_no_borrower, rate_above_limits and credit_free.
func creditSetWarnings(c *Config, cr creditRoles) []Warning {
	var out []Warning
	switch {
	case len(cr.lenders)+len(cr.borrowers) == 0:
		out = append(out, creditWarn("no_credit_roles"))
	case len(cr.lenders) == 0:
		out = append(out, creditWarn("credit_no_lender"))
	case len(cr.borrowers) == 0:
		out = append(out, creditWarn("credit_no_borrower"))
	}
	if len(cr.lenders) == 0 {
		return out
	}
	lowest, free := lowestRate(cr.lenders)
	if free {
		out = append(out, creditWarn("credit_free"))
	}
	if len(cr.borrowers) > 0 && lowest > highestLimit(cr.borrowers) {
		out = append(out, creditWarn("rate_above_limits"))
	}
	return out
}

// lowestRate is the lowest posted rate of the lenders and whether every one is 0.
func lowestRate(lenders []*Role) (lowest int64, free bool) {
	lowest, free = lenders[0].Credit.Lend.RateBP, true
	for _, ro := range lenders {
		lowest = min(lowest, ro.Credit.Lend.RateBP)
		free = free && ro.Credit.Lend.RateBP == 0
	}
	return lowest, free
}

// highestLimit is the highest rate any borrower accepts.
func highestLimit(borrowers []*Role) int64 {
	var hi int64
	for _, ro := range borrowers {
		hi = max(hi, ro.Credit.Borrow.MaxRateBP)
	}
	return hi
}

// creditEconomyWarnings are the warnings that read the world: no_shield (a borrower
// role and ShieldTicks 0), credit_no_upkeep (no upkeep anywhere, so every room is
// 0) and interest_rounds_to_zero.
func creditEconomyWarnings(c *Config, m *Credit, cr creditRoles) []Warning {
	var out []Warning
	if len(cr.borrowers) > 0 && m.ShieldTicks == 0 {
		out = append(out, Warning{Path: []string{"modules", "credit", "shield_ticks"}, Rule: "no_shield"})
	}
	if !upkeepAnywhere(c) {
		out = append(out, creditWarn("credit_no_upkeep"))
	}
	if roundsToZero(c, m, cr) {
		out = append(out, creditWarn("interest_rounds_to_zero"))
	}
	return out
}

// upkeepAnywhere reports whether the world, a role or a region has an upkeep.
func upkeepAnywhere(c *Config) bool {
	return len(c.World.Upkeep) > 0 ||
		slices.ContainsFunc(c.Roles, func(r Role) bool { return len(r.Upkeep) > 0 }) ||
		slices.ContainsFunc(c.Regions, func(r Region) bool { return len(r.ExtraUpkeep) > 0 })
}

// roundsToZero is interest_rounds_to_zero (spec-credit.md 7.3): the interest of
// MinPrincipal for one tick at the lowest posted rate above 0 is 0.
func roundsToZero(c *Config, m *Credit, cr creditRoles) bool {
	var rate int64
	for _, ro := range cr.lenders {
		if r := ro.Credit.Lend.RateBP; r > 0 && (rate == 0 || r < rate) {
			rate = r
		}
	}
	tpy := c.Calendar.TicksPerYear()
	if rate == 0 || tpy < 1 || m.MinPrincipal < 1 {
		return false
	}
	interest, ok := crd.Interest(m.MinPrincipal, rate, 1, tpy)
	return ok && interest == 0
}

// borrowerWarnings are term_exceeds_lenders and cap_below_horizon for one borrower
// role, which read the horizon of its longest recipe (10.2).
func borrowerWarnings(c *Config, lk lookup, m *Credit, cr creditRoles, ro *Role) []Warning {
	h, ok := creditHorizon(c, lk, ro)
	if !ok {
		return nil
	}
	b := ro.Credit.Borrow
	p := []string{"roles", ro.ID, "credit", "borrow"}
	var out []Warning
	if len(cr.lenders) > 0 && h > longestLenderTerm(cr.lenders) {
		out = append(out, Warning{Path: append(slices.Clone(p), "slack_ticks"), Rule: "term_exceeds_lenders"})
	}
	if b.MaxDebtTicks < h {
		out = append(out, Warning{Path: append(slices.Clone(p), "max_debt_ticks"), Rule: "cap_below_horizon"})
	}
	return out
}

// longestLenderTerm is the longest MaxTermTicks of any lender.
func longestLenderTerm(lenders []*Role) int64 {
	var hi int64
	for _, ro := range lenders {
		hi = max(hi, ro.Credit.Lend.MaxTermTicks)
	}
	return hi
}

// creditHorizon is the horizon H of the default request of a borrower role for its
// longest recipe: Duration + 1 + SlackTicks, and one more where the household can be
// too short of money to buy the inputs at once (Late can be 1, 10.2.1.2). Late can be
// 1 where the role's StartMoney less one tick's upkeep cost is below the cost of the
// inputs of that recipe in some region, at the reference prices. ok is false for a
// role with no recipe or an arithmetic that cannot be done.
func creditHorizon(c *Config, lk lookup, ro *Role) (h int64, ok bool) {
	var rc *Recipe
	for _, id := range ro.Recipes {
		if r := lk.recipe[id]; r != nil && (rc == nil || r.Duration > rc.Duration) {
			rc = r
		}
	}
	if rc == nil || rc.Duration < 1 {
		return 0, false
	}
	h, ok = checked.Add(rc.Duration, 1)
	if ok {
		h, ok = checked.Add(h, ro.Credit.Borrow.SlackTicks)
	}
	if ok && lateCan1(c, lk, ro, rc) {
		h, ok = checked.Add(h, 1)
	}
	return h, ok
}

// lateCan1 reports whether, in some region, the StartMoney of the role less its
// upkeep cost is below the cost of the inputs of the recipe at the reference prices.
func lateCan1(c *Config, lk lookup, ro *Role, rc *Recipe) bool {
	for i := range c.Regions {
		g := &c.Regions[i]
		u, ok1 := itemsCost(lk, g.ID, effectiveUpkeep(c, ro), g.ExtraUpkeep)
		in, ok2 := itemsCost(lk, g.ID, rc.Inputs)
		if ok1 && ok2 && roomAfter(int64(ro.StartMoney), u) < in {
			return true
		}
	}
	return false
}

// roomAfter is start less cost, cut to the int64 bounds where the difference would
// wrap. Warnings runs on a configuration that does not validate, so a StartMoney far
// below 0 must read as short, not as a huge balance.
func roomAfter(start, cost int64) int64 {
	if r, ok := checked.Sub(start, cost); ok {
		return r
	}
	if cost > 0 {
		return math.MinInt64
	}
	return math.MaxInt64
}

// itemsCost is the cost of the lists at the reference prices of a region: the sum of
// quantity times price. ok is false for an undeclared commodity or an overflow.
func itemsCost(lk lookup, region string, lists ...[]Item) (cost int64, ok bool) {
	for _, list := range lists {
		for _, it := range list {
			cm := lk.commodity[it.Commodity]
			if cm == nil {
				return 0, false
			}
			line, ok1 := checked.Mul(int64(it.Quantity), int64(lk.price(region, cm)))
			sum, ok2 := checked.Add(cost, line)
			if !ok1 || !ok2 {
				return 0, false
			}
			cost = sum
		}
	}
	return cost, true
}

// lenderWarnings are lender_without_cash, lender_without_income and
// lender_keep_below_term for one lender role.
func lenderWarnings(c *Config, lk lookup, m *Credit, ro *Role) []Warning {
	l := ro.Credit.Lend
	p := []string{"roles", ro.ID, "credit", "lend"}
	var out []Warning
	if lenderWithoutCash(c, lk, m, ro) {
		out = append(out, Warning{Path: append(slices.Clone(p), "keep_ticks"), Rule: "lender_without_cash"})
	}
	if len(ro.Recipes) == 0 && !flowTargets(c, ro.ID) {
		out = append(out, Warning{Path: p, Rule: "lender_without_income"})
		if l.KeepTicks < l.MaxTermTicks {
			out = append(out, Warning{Path: append(slices.Clone(p), "keep_ticks"), Rule: "lender_keep_below_term"})
		}
	}
	return out
}

// lenderWithoutCash is lender_without_cash: in no region does the StartMoney of the
// role less KeepTicks of its upkeep cost at the reference prices reach MinPrincipal.
// A role with no region whose cost can be computed is not warned of.
func lenderWithoutCash(c *Config, lk lookup, m *Credit, ro *Role) bool {
	computed := false
	for i := range c.Regions {
		g := &c.Regions[i]
		u, ok1 := itemsCost(lk, g.ID, effectiveUpkeep(c, ro), g.ExtraUpkeep)
		keep, ok2 := checked.Mul(ro.Credit.Lend.KeepTicks, u)
		if !ok1 || !ok2 {
			continue
		}
		computed = true
		if roomAfter(int64(ro.StartMoney), keep) >= m.MinPrincipal {
			return false
		}
	}
	return computed
}

// flowTargets reports whether a money flow targets the households of the role.
func flowTargets(c *Config, role string) bool {
	ms := c.Modules.MoneySupply
	return ms != nil && slices.ContainsFunc(ms.Flows, func(f MoneyFlow) bool {
		return f.Target.Kind == TargetHouseholds && f.Target.Role == role
	})
}

// endRec is how a claim ended, for LoanEnded and its Debug line: the cause, the
// parts a waterfall paid and the parts lost.
type endRec struct {
	cause                      Cause
	paidP, paidI, lostP, lostI int64
}

// reportLoanEnded reports the end of claim c (spec-credit.md 16.1, 16.2): the Debug
// line "loan ended" and, with an observer, LoanEnded. The regions are those of the
// two accounts at the event.
func (w *World) reportLoanEnded(t int64, c *ledger.Claim, e endRec) {
	if ctx := context.Background(); w.log.Enabled(ctx, slog.LevelDebug) {
		w.log.LogAttrs(ctx, slog.LevelDebug, "loan ended", slog.Int64(logkeys.Claim, c.ID),
			slog.Int64(logkeys.Agent, int64(w.accountView(c.Debtor).Owner)), slog.String(logkeys.Cause, string(e.cause)),
			slog.Int64(logkeys.Lost, e.lostP+e.lostI))
	}
	if !w.obs.supplied {
		return
	}
	dv, cv := w.accountView(c.Debtor), w.accountView(c.Creditor)
	w.obs.deliver(LoanEnded{
		Tick: t, Claim: ClaimID(c.ID), Origin: originOf(c.Origin),
		Creditor: AgentID(cv.Owner), Debtor: AgentID(dv.Owner),
		Region: RegionID(dv.Region), CreditorRegion: RegionID(cv.Region), ToEstate: cv.Kind == ledger.KindEstate,
		Principal: Money(c.Principal), PaidPrincipal: Money(e.paidP), PaidInterest: Money(e.paidI),
		LostPrincipal: Money(e.lostP), LostInterest: Money(e.lostI), Cause: e.cause,
	})
}

// originOf is the origin constant of a ledger claim's origin.
func originOf(o int) Origin {
	if o == ledger.OriginPayout {
		return OriginPayout
	}
	return OriginLoan
}

// reportPassed reports a claim passed to the estate of region (spec-credit.md 9.2,
// 16.1): the Debug line "claim passed" and, with an observer, LoanPassed.
func (w *World) reportPassed(t int64, c *ledger.Claim, from *agent, region RegionID) {
	if ctx := context.Background(); w.log.Enabled(ctx, slog.LevelDebug) {
		w.log.LogAttrs(ctx, slog.LevelDebug, "claim passed", slog.Int64(logkeys.Claim, c.ID),
			slog.Int64(logkeys.Agent, int64(from.id)), slog.Int64(logkeys.Region, int64(region)))
	}
	if w.obs.supplied {
		w.obs.deliver(LoanPassed{Tick: t, Claim: ClaimID(c.ID), Creditor: from.id, Debtor: AgentID(w.accountView(c.Debtor).Owner),
			Region: region, Owed: Money(c.PrincipalOwed + c.InterestOwed)})
	}
}

// logOverdue is the Debug line "claim overdue" of a claim that went unpaid at its due
// tick (spec-credit.md 16.2).
func (w *World) logOverdue(c *ledger.Claim) {
	if ctx := context.Background(); w.log.Enabled(ctx, slog.LevelDebug) {
		w.log.LogAttrs(ctx, slog.LevelDebug, "claim overdue", slog.Int64(logkeys.Claim, c.ID),
			slog.Int64(logkeys.Agent, int64(w.accountView(c.Debtor).Owner)), slog.Int64(logkeys.Owed, c.PrincipalOwed+c.InterestOwed))
	}
}

// logArrears is the Warn "claims overdue" (spec-credit.md 16.2, pattern B): the
// debtors of region g have owed overdue claims of origin loan at the end of P1.7a in
// each of the last warnRun ticks, and at every warnRun after. Its counterpart is
// CreditSettled.Arrears and the Claims rows of the snapshot.
func (w *World) logArrears(g RegionID, r *settleRow) {
	if ctx := context.Background(); w.log.Enabled(ctx, slog.LevelWarn) {
		w.log.LogAttrs(ctx, slog.LevelWarn, "claims overdue", slog.Int64(logkeys.Region, int64(g)),
			slog.Int64(logkeys.Claims, r.overdueClaims), slog.Int64(logkeys.Arrears, r.arrears))
	}
}

// event is the CreditSettled of a region from its row.
func (r settleRow) event(t int64, g RegionID) CreditSettled {
	return CreditSettled{
		Tick: t, Region: g, Due: r.due, Repaid: r.repaid, Defaulted: r.defaulted, WrittenOff: r.writtenOff,
		PrincipalPaid: Money(r.principal), InterestPaid: Money(r.interest), Lost: Money(r.lost),
		Shielded: Money(r.shielded), Headroom: Money(r.headroom), Arrears: Money(r.arrears),
		Received: Money(r.received), ReceivedByEstate: Money(r.receivedEstate),
	}
}

// logCreditRefused is the Debug line of a request or an offer refused at admission
// (spec-credit.md 5.2, 16.2): msg is "request refused" or "offer refused".
func (w *World) logCreditRefused(msg string, agent AgentID, reason Reason) {
	if ctx := context.Background(); w.log.Enabled(ctx, slog.LevelDebug) {
		w.log.LogAttrs(ctx, slog.LevelDebug, msg, slog.Int64(logkeys.Agent, int64(agent)), slog.String(logkeys.Reason, string(reason)))
	}
}

// reportFormed reports a loan formed (spec-credit.md 6.4 step 4, 16.1): the Debug line
// "loan formed" and, with an observer, LoanFormed in formation order.
func (w *World) reportFormed(region RegionID, c *ledger.Claim, lender, borrower AgentID) {
	if ctx := context.Background(); w.log.Enabled(ctx, slog.LevelDebug) {
		w.log.LogAttrs(ctx, slog.LevelDebug, "loan formed", slog.Int64(logkeys.Claim, c.ID),
			slog.Int64(logkeys.Agent, int64(borrower)), slog.Int64(logkeys.Region, int64(region)),
			slog.Int64(logkeys.Principal, c.Principal), slog.Int64(logkeys.Rate, c.RateBP),
			slog.Int64(logkeys.Term, c.Due-c.Formed))
	}
	if w.obs.supplied {
		w.obs.deliver(LoanFormed{
			Tick: c.Formed, Claim: ClaimID(c.ID), Creditor: lender, Debtor: borrower, Region: region,
			Principal: Money(c.Principal), Interest: Money(c.Interest), RateBP: c.RateBP, Term: c.Due - c.Formed, Due: c.Due,
		})
	}
}

// reportRound delivers CreditRound for each region, in ascending RegionID, that has a
// count other than 0, after the last LoanFormed (spec-credit.md 16.1.1).
func (w *World) reportRound() {
	if !w.obs.supplied {
		return
	}
	t := w.Tick()
	for g, c := range w.credit.round.counts {
		if c.quiet() {
			continue
		}
		w.obs.deliver(CreditRound{
			Tick: t, Region: RegionID(g), Requests: c.requests, Offers: c.offers,
			RequestsRefused: c.requestsRefused, OffersRefused: c.offersRefused, InArrears: c.inArrears, Exposure: c.exposure,
			Formed: c.formed, Principal: Money(c.principal), Interest: Money(c.interest), Unfilled: Money(c.unfilled),
			TooMany: c.tooMany, Overflow: c.overflow,
		})
	}
}

// logCapReached is the Warn "claim cap reached" (spec-credit.md 16.2, pattern A): a
// fill was dropped for want of room in the claim list in each of the last ticks of a
// run, at the tenth and every tenth after. Its counterpart is CreditRound.TooMany and
// the length of the snapshot's Claims.
func (w *World) logCapReached() {
	if ctx := context.Background(); w.log.Enabled(ctx, slog.LevelWarn) {
		w.log.LogAttrs(ctx, slog.LevelWarn, "claim cap reached", slog.Int64(logkeys.Claims, int64(w.ledger.OpenClaims())))
	}
}

// ClaimRow is an open claim in the snapshot (spec-credit.md 16.4): a loan or a payout
// claim, with what was fixed at formation and what is still owed.
type ClaimRow struct {
	// Claim is the claim and Origin what formed it.
	Claim  ClaimID
	Origin Origin
	// Creditor is the creditor's agent ID, or the estate's agent ID, which is its
	// RegionID, and Debtor the debtor's.
	Creditor, Debtor AgentID
	// Region is the debtor's region.
	Region RegionID
	// Principal and Interest are fixed at formation; PrincipalOwed and InterestOwed
	// are what remains.
	Principal, Interest, PrincipalOwed, InterestOwed Money
	// RateBP is the yearly rate at formation in basis points, Formed the tick of
	// formation, Due the tick the whole falls due and Overdue the tick it first went
	// unpaid, or -1 while it has not.
	RateBP, Formed, Due, Overdue int64
}

// CreditTotals are the seven cumulative totals of the claim list since the world was
// built (spec-credit.md 3.3.2): money lent, awarded as a payout claim, repaid, the
// interest paid and fixed, and the principal and interest lost. They are never reset.
type CreditTotals struct {
	Lent, Awarded, Repaid, InterestPaid, InterestFixed, LostPrincipal, LostInterest Money
}

// snapshotClaims fills the claim rows and the totals of the snapshot, a deep copy in
// ascending claim ID bounded by MaxClaims; with the module off it leaves both empty.
func (w *World) snapshotClaims(s *Snapshot) {
	if w.credit == nil {
		return
	}
	for _, c := range w.ledger.AppendClaims(nil) {
		debtor := w.accountView(c.Debtor)
		s.Claims = append(s.Claims, ClaimRow{
			Claim: ClaimID(c.ID), Origin: originOf(c.Origin),
			Creditor: AgentID(w.accountView(c.Creditor).Owner), Debtor: AgentID(debtor.Owner),
			Region:    RegionID(debtor.Region),
			Principal: Money(c.Principal), Interest: Money(c.Interest),
			PrincipalOwed: Money(c.PrincipalOwed), InterestOwed: Money(c.InterestOwed),
			RateBP: c.RateBP, Formed: c.Formed, Due: c.Due, Overdue: c.Overdue,
		})
	}
	t := w.ledger.ClaimTotals()
	s.Credit = CreditTotals{
		Lent: Money(t.Lent), Awarded: Money(t.Awarded), Repaid: Money(t.Repaid), InterestPaid: Money(t.InterestPaid),
		InterestFixed: Money(t.InterestFixed), LostPrincipal: Money(t.LostPrincipal), LostInterest: Money(t.LostInterest),
	}
}
