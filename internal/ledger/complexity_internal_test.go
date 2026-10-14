package ledger

import (
	"errors"
	"math"
	"reflect"
	"slices"
	"strings"
	"testing"
)

// TestOutsideRange verifies the range predicate of the lot audit at its
// boundaries: a zero limit admits only 0, a positive limit admits [lo, hi],
// and a negative limit (never produced by a valid table) admits anything.
//
// Requirement: FR-LED-024, FR-LED-035
func TestOutsideRange(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name            string
		limit, v, lo, h int64
		want            bool
	}{
		{"zero limit, zero", 0, 0, 1, 5, false},
		{"zero limit, nonzero", 0, 1, 1, 5, true},
		{"zero limit, negative", 0, -1, 1, 5, true},
		{"low edge", 5, 1, 1, 5, false},
		{"below low edge", 5, 0, 1, 5, true},
		{"high edge", 5, 5, 1, 5, false},
		{"above high edge", 5, 6, 1, 5, true},
		{"born oldest allowed", 3, -2, -2, -1, false},
		{"born one too old", 3, -3, -2, -1, true},
		{"negative limit", -1, 99, 1, 5, false},
	}
	for _, tc := range tests {
		if got := outsideRange(tc.limit, tc.v, tc.lo, tc.h); got != tc.want {
			t.Errorf("%s: outsideRange(%d, %d, %d, %d) = %v, want %v", tc.name, tc.limit, tc.v, tc.lo, tc.h, got, tc.want)
		}
	}
}

// TestEarlierAndAddFits verifies the two small folds the audit and the wear
// check use.
//
// Requirement: FR-LED-035
func TestEarlierAndAddFits(t *testing.T) {
	t.Parallel()
	if e, seen := earlier(0, false, 7); e != 7 || !seen {
		t.Errorf("first expiry = %d, %v, want 7, true", e, seen)
	}
	if e, seen := earlier(5, true, 7); e != 5 || !seen {
		t.Errorf("later expiry replaced the earliest: %d, %v", e, seen)
	}
	if e, seen := earlier(5, true, 2); e != 2 || !seen {
		t.Errorf("earlier expiry = %d, %v, want 2, true", e, seen)
	}
	if !addFits(math.MaxInt64-1, 1) || addFits(math.MaxInt64, 1) {
		t.Errorf("addFits does not stop exactly at MaxInt64")
	}
	if !addFits(math.MinInt64+1, -1) || addFits(math.MinInt64, -1) {
		t.Errorf("addFits does not stop exactly at MinInt64")
	}
}

// TestAuditLots_RunSumAndHint verifies, on a synthetic account, that the
// quantity bound is per commodity (a run restarts at each commodity), that two
// lots of one commodity add, and that the hint is checked against the earliest
// expiry wherever in canonical order that lot sits.
//
// Requirement: FR-LED-024, FR-LED-046
func TestAuditLots_RunSumAndHint(t *testing.T) {
	t.Parallel()
	g := newRig(t, scriptMaxMoney)
	l := g.l
	big := int64(scriptMaxQty - 40)
	perCommodity := &account{kind: KindAgent, lots: []Lot{{Commodity: 1, Quantity: big, Durability: 5}, {Commodity: 2, Quantity: big}}}
	if err := l.auditLots(0, perCommodity); err != nil {
		t.Errorf("two commodities of %d each under MaxQuantity %d: %v", big, scriptMaxQty, err)
	}
	sameCommodity := &account{kind: KindAgent, lots: []Lot{{Commodity: 1, Quantity: big, Durability: 1}, {Commodity: 1, Quantity: big, Durability: 2}}}
	var ae *AuditError
	if err := l.auditLots(0, sameCommodity); !errors.As(err, &ae) || ae.Identity != IdentityStructure || ae.Commodity != 1 {
		t.Errorf("two lots of one commodity above MaxQuantity: %v", err)
	}
	tests := []struct {
		name    string
		lots    []Lot
		hasHint bool
		hint    int64
		want    Identity
		ok      bool
	}{
		{"spoiling lot, no hint", []Lot{{Commodity: 0, Quantity: 1, Born: -1}}, false, 0, IdentityHint, false},
		{"hint equals the expiry", []Lot{{Commodity: 0, Quantity: 1, Born: -1}}, true, 1, 0, true},
		{"hint one late", []Lot{{Commodity: 0, Quantity: 1, Born: -1}}, true, 2, IdentityHint, false},
		{"hint early is allowed", []Lot{{Commodity: 0, Quantity: 1, Born: -1}}, true, -5, 0, true},
		// Commodity 3 sorts after 0 and expires first: the minimum is not
		// the first spoiling lot.
		{"earliest is a later lot", []Lot{{Commodity: 0, Quantity: 1, Born: -1}, {Commodity: 3, Quantity: 1, Born: -3, Durability: 6}}, true, 1, IdentityHint, false},
		{"earliest is a later lot, hint fits", []Lot{{Commodity: 0, Quantity: 1, Born: -1}, {Commodity: 3, Quantity: 1, Born: -3, Durability: 6}}, true, 0, 0, true},
		{"nothing spoils, no hint", []Lot{{Commodity: 2, Quantity: 1}}, false, 0, 0, true},
	}
	for _, tc := range tests {
		a := &account{kind: KindAgent, lots: tc.lots, hasHint: tc.hasHint, hint: tc.hint}
		err := l.auditLots(0, a)
		switch {
		case tc.ok && err != nil:
			t.Errorf("%s: %v, want nil", tc.name, err)
		case !tc.ok && (!errors.As(err, &ae) || ae.Identity != tc.want):
			t.Errorf("%s: %v, want identity %d", tc.name, err, tc.want)
		}
	}
}

// TestAuditLotShape_Order verifies that a lot with several faults reports the
// first in the documented order: commodity, then quantity and earmark, then
// zero, then order, then born, then durability.
//
// Requirement: FR-LED-024, FR-LED-035
func TestAuditLotShape_Order(t *testing.T) {
	t.Parallel()
	g := newRig(t, scriptMaxMoney)
	l := g.l
	tests := []struct {
		name string
		lots []Lot
		want Identity
		desc string
	}{
		{"commodity first", []Lot{{Commodity: 9, Quantity: -1}}, IdentityCanonical, "names a commodity"},
		{"negative before zero", []Lot{{Commodity: 2, Quantity: 0, Reserved: -1}}, IdentityNonNegative, "earmarked"},
		{"zero before order", []Lot{{Commodity: 2, Quantity: 1}, {Commodity: 2, Quantity: 0}}, IdentityCanonical, "quantity 0"},
		{"order before born", []Lot{{Commodity: 0, Quantity: 1, Born: -1}, {Commodity: 0, Quantity: 1, Born: -1}}, IdentityCanonical, "not after"},
		{"born before durability", []Lot{{Commodity: 3, Quantity: 1, Born: 7, Durability: 99}}, IdentityCanonical, "born"},
		{"durability", []Lot{{Commodity: 1, Quantity: 1, Durability: 99}}, IdentityCanonical, "durability"},
		{"durability 0 on a durable", []Lot{{Commodity: 1, Quantity: 1, Durability: 0}}, IdentityCanonical, "durability"},
		{"durability on a plain commodity", []Lot{{Commodity: 2, Quantity: 1, Durability: 1}}, IdentityCanonical, "durability"},
	}
	for _, tc := range tests {
		a := &account{kind: KindAgent, lots: tc.lots, hasHint: true, hint: -10}
		err := l.auditLots(0, a)
		var ae *AuditError
		if !errors.As(err, &ae) || ae.Identity != tc.want || !strings.Contains(ae.Detail, tc.desc) {
			t.Errorf("%s: %v, want identity %d mentioning %q", tc.name, err, tc.want, tc.desc)
		}
	}
}

// TestAuditCounters_Split verifies each failure of the counter audit: a table
// of the wrong length, each of the four flow tables with a negative entry, and
// a negative claim total, in that order.
//
// Requirement: FR-LED-035, FR-LED-064
func TestAuditCounters_Split(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		mut  func(l *Ledger)
		desc string
	}{
		{"short table", func(l *Ledger) { l.base = l.base[:len(l.base)-1] }, "wrong length"},
		{"short bucket", func(l *Ledger) { l.bucket = l.bucket[:len(l.bucket)-1] }, "wrong length"},
		{"negative total", func(l *Ledger) { l.totals[1] = -1 }, "flow total"},
		{"negative bucket", func(l *Ledger) { l.bucket[1] = -1 }, "construction bucket"},
		{"negative wear", func(l *Ledger) { l.wear[1] = -1 }, "wear units"},
		{"negative construction wear", func(l *Ledger) { l.wearBucket[1] = -1 }, "construction wear units"},
		{"negative claim total", func(l *Ledger) { l.claimTot.LostInterest = -1 }, "claim total 6"},
	}
	for _, tc := range tests {
		l := newRig(t, scriptMaxMoney).l
		if err := l.auditCounters(); err != nil {
			t.Fatalf("%s: a fresh ledger fails the counter audit: %v", tc.name, err)
		}
		tc.mut(l)
		err := l.auditCounters()
		var ae *AuditError
		if !errors.As(err, &ae) || ae.Identity != IdentityStructure || !strings.Contains(ae.Detail, tc.desc) {
			t.Errorf("%s: %v, want a structure failure mentioning %q", tc.name, err, tc.desc)
		}
	}
}

// TestAuditClaims_PartiesAndOwed verifies the claim identities by the account
// and claim each names: an owed amount outside its range, a claim that owes
// nothing, the same account on both sides, a creditor or debtor of the wrong
// kind, and the claims over MaxClaims.
//
// Requirement: FR-LED-064
func TestAuditClaims_PartiesAndOwed(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		mut     func(l *Ledger, g *rig)
		account bool
	}{
		{"owed above principal", func(l *Ledger, _ *rig) { l.claims[0].PrincipalOwed = l.claims[0].Principal + 1 }, false},
		{"interest owed above interest", func(l *Ledger, _ *rig) { l.claims[0].InterestOwed = l.claims[0].Interest + 1 }, false},
		{"owed negative", func(l *Ledger, _ *rig) { l.claims[0].InterestOwed = -1 }, false},
		{"owes nothing", func(l *Ledger, _ *rig) { l.claims[0].PrincipalOwed, l.claims[0].InterestOwed = 0, 0 }, false},
		{"same account", func(l *Ledger, _ *rig) { l.claims[0].Creditor = l.claims[0].Debtor }, true},
		{"creditor is a boundary", func(l *Ledger, g *rig) { l.claims[0].Creditor = g.mint }, true},
		{"debtor is an estate", func(l *Ledger, g *rig) { l.claims[0].Debtor = g.holders[4] }, true},
		{"IDs not ascending", func(l *Ledger, _ *rig) { l.claims[0].ID = l.nextClaim }, false},
		{"principal below 1", func(l *Ledger, _ *rig) { l.claims[0].Principal = 0 }, false},
		{"payout with interest", func(l *Ledger, _ *rig) { l.claims[0].Origin = OriginPayout; l.claims[0].Interest = 1 }, false},
	}
	for _, tc := range tests {
		g := newRig(t, scriptMaxMoney)
		g.fund(t)
		if _, err := g.l.Lend(g.holders[0], g.holders[1], 50, 5, 500, 3); err != nil {
			t.Fatalf("setup: %v", err)
		}
		tc.mut(g.l, g)
		var ae *AuditError
		if err := g.l.auditClaims(); !errors.As(err, &ae) || ae.Identity != IdentityClaim {
			t.Errorf("%s: %v, want an IdentityClaim failure", tc.name, err)
			continue
		}
		if ae.Claim != g.l.claims[0].ID {
			t.Errorf("%s: claim = %d, want %d", tc.name, ae.Claim, g.l.claims[0].ID)
		}
		if (ae.Account != None) != tc.account {
			t.Errorf("%s: account = %d, names an account = %v, want %v", tc.name, ae.Account, ae.Account != None, tc.account)
		}
		if ae.Commodity != None {
			t.Errorf("%s: commodity = %d, want None", tc.name, ae.Commodity)
		}
	}
	g := newRig(t, scriptMaxMoney)
	g.l.maxClaims = 0
	g.l.claims = []Claim{{ID: 0}}
	var ae *AuditError
	if err := g.l.auditClaims(); !errors.As(err, &ae) || ae.Claim != -1 || ae.Account != None {
		t.Errorf("claims over MaxClaims: %v, want an IdentityClaim failure with no claim", err)
	}
}

// TestAuditFail_NamesNoClaim verifies that auditFail leaves the claim at -1
// and auditClaimFail names a claim and no commodity.
//
// Requirement: FR-LED-064
func TestAuditFail_NamesNoClaim(t *testing.T) {
	t.Parallel()
	var ae *AuditError
	if !errors.As(auditFail(IdentityZeroTotal, 3, Money, "x %d", 1), &ae) || ae.Claim != -1 || ae.Account != 3 || ae.Commodity != Money || ae.Detail != "x 1" {
		t.Errorf("auditFail = %+v", ae)
	}
	if !errors.As(auditClaimFail(4, 9, "y"), &ae) || ae.Identity != IdentityClaim || ae.Claim != 9 || ae.Account != 4 || ae.Commodity != None {
		t.Errorf("auditClaimFail = %+v", ae)
	}
}

// TestLend_RefusalOrder verifies that when two checks of Lend would fail, the
// earlier one is returned, and that no refusal changes the ledger.
//
// Requirement: FR-LED-060, FR-LED-061
func TestLend_RefusalOrder(t *testing.T) {
	t.Parallel()
	const due = 5
	tests := []struct {
		name string
		prep func(g *rig)
		op   func(g *rig) error
		want error
	}{
		{"amount before account", nil, func(g *rig) error {
			_, err := g.l.Lend(999, g.holders[1], 0, 0, 0, due)
			return err
		}, ErrInvalidAmount},
		{"negative interest", nil, func(g *rig) error {
			_, err := g.l.Lend(g.holders[0], g.holders[1], 1, -1, 0, due)
			return err
		}, ErrInvalidAmount},
		{"account before shape", nil, func(g *rig) error {
			_, err := g.l.Lend(999, 999, 1, 0, 0, 0)
			return err
		}, ErrUnknownAccount},
		{"unknown debtor", nil, func(g *rig) error {
			_, err := g.l.Lend(g.holders[0], 999, 1, 0, 0, due)
			return err
		}, ErrUnknownAccount},
		{"shape before reason", nil, func(g *rig) error {
			_, err := g.l.Lend(g.holders[4], g.holders[0], 1, 0, 0, g.l.tick)
			return err
		}, ErrClaimShape},
		{"same account", nil, func(g *rig) error {
			_, err := g.l.Lend(g.holders[0], g.holders[0], 1, 0, 0, due)
			return err
		}, ErrClaimShape},
		{"reason before count", func(g *rig) { g.l.maxClaims = 0 }, func(g *rig) error {
			_, err := g.l.Lend(g.holders[4], g.holders[0], 1, 0, 0, due)
			return err
		}, ErrReasonMismatch},
		{"count before overflow", func(g *rig) { g.l.maxClaims = 0 }, func(g *rig) error {
			_, err := g.l.Lend(g.holders[0], g.holders[1], scriptMaxMoney, 1, 0, due)
			return err
		}, ErrTooManyClaims},
		{"owed above MaxMoney", nil, func(g *rig) error {
			_, err := g.l.Lend(g.holders[0], g.holders[1], scriptMaxMoney, 1, 0, due)
			return err
		}, ErrBalanceOverflow},
		{"total overflow", func(g *rig) { g.l.claimTot.Lent = math.MaxInt64 }, func(g *rig) error {
			_, err := g.l.Lend(g.holders[0], g.holders[1], 1, 0, 0, due)
			return err
		}, ErrBalanceOverflow},
		{"interest total overflow", func(g *rig) { g.l.claimTot.InterestFixed = math.MaxInt64 }, func(g *rig) error {
			_, err := g.l.Lend(g.holders[0], g.holders[1], 1, 1, 0, due)
			return err
		}, ErrBalanceOverflow},
		{"transfer refusal", func(g *rig) { g.l.accounts[g.holders[0]].balance = 0 }, func(g *rig) error {
			_, err := g.l.Lend(g.holders[0], g.holders[1], 1, 0, 0, due)
			return err
		}, ErrInsufficientFunds},
	}
	for _, tc := range tests {
		g := newRig(t, scriptMaxMoney)
		g.fund(t)
		if tc.prep != nil {
			tc.prep(g)
		}
		t.Run(tc.name, func(t *testing.T) { refusedNoAudit(t, g, tc.want, func() error { return tc.op(g) }) })
	}
	g := newRig(t, scriptMaxMoney)
	g.fund(t)
	id, err := g.l.Lend(g.holders[0], g.holders[1], 10, 2, 300, due)
	if err != nil {
		t.Fatalf("Lend: %v", err)
	}
	want := Claim{ID: id, Creditor: g.holders[0], Debtor: g.holders[1], Principal: 10, Interest: 2, PrincipalOwed: 10, InterestOwed: 2, RateBP: 300, Formed: g.l.tick, Due: due, Overdue: -1, Origin: OriginLoan}
	if got := g.l.claims; len(got) != 1 || got[0] != want {
		t.Errorf("claims = %+v, want %+v", got, want)
	}
	if g.l.claimTot.Lent != 10 || g.l.claimTot.InterestFixed != 2 {
		t.Errorf("totals = %+v", g.l.claimTot)
	}
}

// refusedNoAudit is refused for a ledger a test has broken on purpose (a
// total set to MaxInt64 fails the audit's other identities).
func refusedNoAudit(t *testing.T, g *rig, want error, op func() error) {
	t.Helper()
	flows, claims, tot := g.l.AppendFlows(nil, false), slices.Clone(g.l.claims), g.l.claimTot
	bal := make([]int64, len(g.l.accounts))
	for i := range bal {
		bal[i] = g.l.accounts[i].balance
	}
	if got := op(); !errors.Is(got, want) {
		t.Fatalf("error = %v, want %v", got, want)
	}
	if !reflect.DeepEqual(flows, g.l.AppendFlows(nil, false)) || !reflect.DeepEqual(claims, g.l.claims) || tot != g.l.claimTot {
		t.Errorf("a refused call changed flows, claims or totals")
	}
	for i := range bal {
		if g.l.accounts[i].balance != bal[i] {
			t.Errorf("a refused call changed the balance of account %d", i)
		}
	}
}

// TestObligate_RefusalOrder verifies the order of the checks of Obligate and
// that a refusal changes nothing.
//
// Requirement: FR-LED-062
func TestObligate_RefusalOrder(t *testing.T) {
	t.Parallel()
	const due = 5
	tests := []struct {
		name string
		prep func(g *rig)
		op   func(g *rig) error
		want error
	}{
		{"amount before account", nil, func(g *rig) error { _, err := g.l.Obligate(999, 998, 0, due); return err }, ErrInvalidAmount},
		{"unknown account", nil, func(g *rig) error { _, err := g.l.Obligate(g.holders[0], 999, 1, due); return err }, ErrUnknownAccount},
		{"account before shape", nil, func(g *rig) error { _, err := g.l.Obligate(999, 999, 1, 0); return err }, ErrUnknownAccount},
		{"same account", nil, func(g *rig) error { _, err := g.l.Obligate(g.holders[0], g.holders[0], 1, due); return err }, ErrClaimShape},
		{"due not after tick", nil, func(g *rig) error { _, err := g.l.Obligate(g.holders[0], g.holders[1], 1, g.l.tick); return err }, ErrClaimShape},
		{"creditor a boundary", nil, func(g *rig) error { _, err := g.l.Obligate(g.mint, g.holders[1], 1, due); return err }, ErrClaimShape},
		{"debtor an estate", nil, func(g *rig) error { _, err := g.l.Obligate(g.holders[0], g.holders[4], 1, due); return err }, ErrClaimShape},
		{"shape before count", func(g *rig) { g.l.maxClaims = 0 }, func(g *rig) error { _, err := g.l.Obligate(g.holders[0], g.holders[4], 1, due); return err }, ErrClaimShape},
		{"count before overflow", func(g *rig) { g.l.maxClaims = 0; g.l.claimTot.Awarded = math.MaxInt64 }, func(g *rig) error { _, err := g.l.Obligate(g.holders[0], g.holders[1], 1, due); return err }, ErrTooManyClaims},
		{"awarded overflow", func(g *rig) { g.l.claimTot.Awarded = math.MaxInt64 }, func(g *rig) error { _, err := g.l.Obligate(g.holders[0], g.holders[1], 1, due); return err }, ErrBalanceOverflow},
	}
	for _, tc := range tests {
		g := newRig(t, scriptMaxMoney)
		g.fund(t)
		if tc.prep != nil {
			tc.prep(g)
		}
		t.Run(tc.name, func(t *testing.T) { refusedNoAudit(t, g, tc.want, func() error { return tc.op(g) }) })
	}
	g := newRig(t, scriptMaxMoney)
	g.fund(t)
	// An estate may be the creditor of a payout claim.
	id, err := g.l.Obligate(g.holders[4], g.holders[1], 9, due)
	if err != nil {
		t.Fatalf("Obligate: %v", err)
	}
	want := Claim{ID: id, Creditor: g.holders[4], Debtor: g.holders[1], Principal: 9, PrincipalOwed: 9, Formed: g.l.tick, Due: due, Overdue: -1, Origin: OriginPayout}
	if got := g.l.claims; len(got) != 1 || got[0] != want || g.l.claimTot.Awarded != 9 {
		t.Errorf("claims = %+v, awarded = %d, want %+v, 9", got, g.l.claimTot.Awarded, want)
	}
}

// TestTransfer_RefusalOrder verifies the order of the checks of Transfer, the
// reserved variant, and that the overflow checks of the credit and the flow
// total refuse before any effect.
//
// Requirement: FR-LED-003, FR-LED-004
func TestTransfer_RefusalOrder(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		prep func(g *rig)
		op   func(g *rig) error
		want error
	}{
		{"amount before account", nil, func(g *rig) error { return g.l.Transfer(999, 998, 0, Trade) }, ErrInvalidAmount},
		{"from before to", nil, func(g *rig) error { return g.l.Transfer(999, 998, 1, Trade) }, ErrUnknownAccount},
		{"to unknown", nil, func(g *rig) error { return g.l.Transfer(g.holders[0], 998, 1, Trade) }, ErrUnknownAccount},
		{"account before reason", nil, func(g *rig) error { return g.l.Transfer(g.holders[0], 998, 1, Wear) }, ErrUnknownAccount},
		{"same account", nil, func(g *rig) error { return g.l.Transfer(g.holders[0], g.holders[0], 1, Trade) }, ErrReasonMismatch},
		{"goods reason", nil, func(g *rig) error { return g.l.Transfer(g.holders[0], g.holders[1], 1, Wear) }, ErrReasonMismatch},
		{"loan between agent and estate", nil, func(g *rig) error { return g.l.Transfer(g.holders[0], g.holders[4], 1, LoanOut) }, ErrReasonMismatch},
		{"boundary of another thing", nil, func(g *rig) error { return g.l.Transfer(g.bnd[Initial][0], g.holders[0], 1, Initial) }, ErrReasonMismatch},
		{"boundary of another reason", nil, func(g *rig) error { return g.l.Transfer(g.mint, g.holders[0], 1, Trade) }, ErrReasonMismatch},
		{"reason before funds", func(g *rig) { g.l.accounts[g.holders[0]].balance = 0 }, func(g *rig) error { return g.l.Transfer(g.holders[0], g.holders[0], 1, Trade) }, ErrReasonMismatch},
		{"funds before overflow", func(g *rig) {
			g.l.accounts[g.holders[0]].balance = 0
			g.l.accounts[g.holders[1]].balance = scriptMaxMoney
		}, func(g *rig) error { return g.l.Transfer(g.holders[0], g.holders[1], 1, Trade) }, ErrInsufficientFunds},
		{"reserved money is not available", func(g *rig) { g.l.accounts[g.holders[0]].reserved = g.l.accounts[g.holders[0]].balance }, func(g *rig) error { return g.l.Transfer(g.holders[0], g.holders[1], 1, Trade) }, ErrInsufficientFunds},
		{"credit above MaxMoney", func(g *rig) { g.l.accounts[g.holders[1]].balance = scriptMaxMoney }, func(g *rig) error { return g.l.Transfer(g.holders[0], g.holders[1], 1, Trade) }, ErrBalanceOverflow},
		{"boundary debit overflow", func(g *rig) { g.l.accounts[g.mint].balance = math.MinInt64 }, func(g *rig) error { return g.l.Transfer(g.mint, g.holders[0], 1, Initial) }, ErrBalanceOverflow},
		{"flow total overflow", func(g *rig) {
			s, _ := g.l.slotOf(Trade, Money)
			g.l.bucket[g.l.accounts[g.holders[0]].region*g.l.stride+s] = math.MaxInt64
		}, func(g *rig) error { return g.l.Transfer(g.holders[0], g.holders[1], 1, Trade) }, ErrBalanceOverflow},
		{"reserved from a boundary", nil, func(g *rig) error { return g.l.TransferReserved(g.mint, g.holders[0], 1, Initial) }, ErrBoundaryReserve},
		{"reserved exceeds", func(g *rig) { g.l.accounts[g.holders[0]].reserved = 1 }, func(g *rig) error { return g.l.TransferReserved(g.holders[0], g.holders[1], 2, Trade) }, ErrReservationExceeds},
		{"reason before reserved boundary", nil, func(g *rig) error { return g.l.TransferReserved(g.mint, g.holders[0], 1, Trade) }, ErrReasonMismatch},
	}
	for _, tc := range tests {
		g := newRig(t, scriptMaxMoney)
		g.fund(t)
		if tc.prep != nil {
			tc.prep(g)
		}
		t.Run(tc.name, func(t *testing.T) { refusedNoAudit(t, g, tc.want, func() error { return tc.op(g) }) })
	}
}

// TestTransfer_ReservedAndRegion verifies a transfer drawn on a reservation
// lowers the reservation and the balance together, and that the flow lands in
// the region of the non-boundary side (of the debited account when internal).
//
// Requirement: FR-LED-004, FR-LED-053
func TestTransfer_ReservedAndRegion(t *testing.T) {
	t.Parallel()
	g := newRig(t, scriptMaxMoney)
	g.fund(t)
	l := g.l
	a0, a1 := g.holders[0], g.holders[1] // regions 0 and 1
	if err := l.ReserveMoney(a0, 20); err != nil {
		t.Fatal(err)
	}
	before := l.accounts[a0]
	if err := l.TransferReserved(a0, a1, 15, Trade); err != nil {
		t.Fatalf("TransferReserved: %v", err)
	}
	if a := l.accounts[a0]; a.balance != before.balance-15 || a.reserved != 5 {
		t.Errorf("after a reserved transfer balance = %d, reserved = %d, want %d, 5", a.balance, a.reserved, before.balance-15)
	}
	if err := l.Transfer(g.mint, a1, 1, Initial); err != nil {
		t.Fatalf("Transfer: %v", err)
	}
	want := map[Flow]bool{}
	for _, f := range l.AppendFlows(nil, true) {
		want[f] = true
	}
	for _, f := range []Flow{{Region: 0, Reason: Trade, Commodity: Money, Total: 15}} {
		if !want[f] {
			t.Errorf("construction bucket %v lacks %v: the internal flow is attributed to the debited region", l.AppendFlows(nil, true), f)
		}
	}
	for _, f := range l.AppendFlows(nil, true) {
		if f.Reason == Initial && f.Commodity == Money && f.Region != 1 && f.Total == 1 {
			t.Errorf("a faucet flow to region 1 is attributed to region %d", f.Region)
		}
	}
}

// TestAppendFlows_Order verifies that flows come out ascending by region, then
// reason, money before goods, then commodity, whatever order they happened in,
// and that the bucket and the tick are separate.
//
// Requirement: FR-LED-053
func TestAppendFlows_Order(t *testing.T) {
	g := newRig(t, scriptMaxMoney)
	g.fund(t)
	l := g.l
	if err := l.EndConstruction(); err != nil {
		t.Fatal(err)
	}
	// Region 1 first, then region 0; goods before money.
	steps := []func() error{
		func() error { return l.Move(g.holders[1], g.bnd[Disposal][2], 2, 1, Disposal) },
		func() error { return l.Move(g.holders[1], g.bnd[Disposal][0], 0, 1, Disposal) },
		func() error { return l.Transfer(g.holders[1], g.holders[3], 2, Trade) },
		func() error { return l.Move(g.holders[0], g.bnd[Disposal][2], 2, 1, Disposal) },
		func() error { return l.Transfer(g.holders[0], g.holders[2], 3, Trade) },
	}
	for i, s := range steps {
		if err := s(); err != nil {
			t.Fatalf("step %d: %v", i, err)
		}
	}
	got := l.AppendFlows(nil, false)
	want := []Flow{
		{Region: 0, Reason: Trade, Commodity: Money, Total: 3},
		{Region: 0, Reason: Disposal, Commodity: 2, Total: 1},
		{Region: 1, Reason: Trade, Commodity: Money, Total: 2},
		{Region: 1, Reason: Disposal, Commodity: 0, Total: 1},
		{Region: 1, Reason: Disposal, Commodity: 2, Total: 1},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("AppendFlows = %v, want %v", got, want)
	}
	if b := l.AppendFlows(nil, true); len(b) == 0 || reflect.DeepEqual(b, got) {
		t.Errorf("the construction bucket = %v, want the funding flows, not the tick's", b)
	}
	// It reuses dst without allocating.
	dst := make([]Flow, 0, 16)
	if n := testing.AllocsPerRun(20, func() { dst = l.AppendFlows(dst[:0], false) }); n != 0 {
		t.Errorf("AppendFlows allocated %v times with room in dst", n)
	}
}

// TestWearItems_RefusalOrder verifies the order of the checks of WearItems and
// that an overflow of any of the three totals refuses before any effect.
//
// Requirement: FR-LED-040, FR-LED-041
func TestWearItems_RefusalOrder(t *testing.T) {
	t.Parallel()
	const durable = 1
	tests := []struct {
		name string
		prep func(g *rig)
		op   func(g *rig) error
		want error
	}{
		{"amount before account", nil, func(g *rig) error { return g.l.WearItems(999, 99, 0, 1) }, ErrInvalidAmount},
		{"units below 1", nil, func(g *rig) error { return g.l.WearItems(g.holders[0], durable, 1, 0) }, ErrInvalidAmount},
		{"account before commodity", nil, func(g *rig) error { return g.l.WearItems(999, 99, 1, 1) }, ErrUnknownAccount},
		{"commodity before kind", nil, func(g *rig) error { return g.l.WearItems(g.holders[4], 99, 1, 1) }, ErrUnknownCommodity},
		{"negative commodity", nil, func(g *rig) error { return g.l.WearItems(g.holders[0], -1, 1, 1) }, ErrUnknownCommodity},
		{"kind before durable", nil, func(g *rig) error { return g.l.WearItems(g.holders[4], 2, 1, 1) }, ErrReasonMismatch},
		{"not durable", nil, func(g *rig) error { return g.l.WearItems(g.holders[0], 2, 1, 1) }, ErrNotDurable},
		{"durable before stock", nil, func(g *rig) error { return g.l.WearItems(g.holders[0], 2, 1000, 1) }, ErrNotDurable},
		{"no wear boundary", func(g *rig) { g.l.boundary[g.l.off[Wear][thingGoods]+durable] = -1 }, func(g *rig) error { return g.l.WearItems(g.holders[0], durable, 1000, 1) }, ErrUnknownAccount},
		{"stock", nil, func(g *rig) error { return g.l.WearItems(g.holders[0], durable, 13, 1) }, ErrInsufficientStock},
		{"earmarked is not available", func(g *rig) {
			if err := g.l.ReserveGoods(g.holders[0], durable, 12); err != nil {
				panic(err)
			}
		}, func(g *rig) error { return g.l.WearItems(g.holders[0], durable, 1, 1) }, ErrInsufficientStock},
		{"wear units overflow", func(g *rig) { g.l.wearBucket[durable] = math.MaxInt64 }, func(g *rig) error { return g.l.WearItems(g.holders[0], durable, 1, 1) }, ErrBalanceOverflow},
		{"flow total overflow", func(g *rig) {
			a := g.l.accounts[g.holders[0]]
			g.l.bucket[g.l.slot(&a, Wear, durable)] = math.MaxInt64
		}, func(g *rig) error { return g.l.WearItems(g.holders[0], durable, 12, 5) }, ErrBalanceOverflow},
		{"boundary overflow", func(g *rig) {
			g.l.accounts[g.bnd[Wear][durable]].balance = math.MaxInt64
		}, func(g *rig) error { return g.l.WearItems(g.holders[0], durable, 12, 5) }, ErrBalanceOverflow},
	}
	for _, tc := range tests {
		g := newRig(t, scriptMaxMoney)
		g.fund(t)
		if tc.prep != nil {
			tc.prep(g)
		}
		t.Run(tc.name, func(t *testing.T) { refusedNoAudit(t, g, tc.want, func() error { return tc.op(g) }) })
	}
}

// TestWearItems_RoutesWearUnitsByPhase verifies that wear units go to the
// construction bucket while in construction and to the tick's afterwards, and
// that a destroyed item is a sink of its quantity to the Wear boundary.
//
// Requirement: FR-LED-040, FR-LED-041
func TestWearItems_RoutesWearUnitsByPhase(t *testing.T) {
	t.Parallel()
	g := newRig(t, scriptMaxMoney)
	g.fund(t)
	l := g.l
	h := g.holders[0]
	if err := l.WearItems(h, 1, 2, 2); err != nil {
		t.Fatal(err)
	}
	if b, _ := l.WearUnits(1, true); b != 4 {
		t.Errorf("construction wear units = %d, want 4", b)
	}
	if n, _ := l.WearUnits(1, false); n != 0 {
		t.Errorf("tick wear units = %d, want 0 in construction", n)
	}
	if err := l.EndConstruction(); err != nil {
		t.Fatal(err)
	}
	// Durability 5, worn by 5 destroys the items: removed counts 5 each.
	boundary := l.accounts[g.bnd[Wear][1]].balance
	if err := l.WearItems(h, 1, 3, 9); err != nil {
		t.Fatal(err)
	}
	if n, _ := l.WearUnits(1, false); n != 11 {
		t.Errorf("tick wear units = %d, want 11: two items worn out at 3 and one at 5", n)
	}
	if got := l.accounts[g.bnd[Wear][1]].balance; got != boundary+3 {
		t.Errorf("Wear boundary balance = %d, want %d: three destroyed items", got, boundary+3)
	}
	if err := l.Audit(); err != nil {
		t.Errorf("Audit: %v", err)
	}
}

// TestReserve_NamedAndBoundaryRefusals verifies the refusals of the earmark
// calls, including the named-lot release path that shares lotShort with the
// unnamed one, and that release is allowed on a boundary account while
// reserve is not.
//
// Requirement: FR-LED-033, FR-LED-034
func TestReserve_NamedAndBoundaryRefusals(t *testing.T) {
	t.Parallel()
	g := newRig(t, scriptMaxMoney)
	g.fund(t)
	l := g.l
	h := g.holders[1]
	key := Key{Commodity: 1, Durability: 5}
	if _, ok := l.accounts[h].find(key); !ok {
		t.Fatalf("setup: no lot %v", key)
	}
	tests := []struct {
		name string
		op   func() error
		want error
	}{
		{"amount before account", func() error { return l.ReserveGoods(999, 99, 0) }, ErrInvalidAmount},
		{"account before commodity", func() error { return l.ReserveGoods(999, 99, 1) }, ErrUnknownAccount},
		{"commodity before boundary", func() error { return l.ReserveGoods(g.bnd[Initial][0], 99, 1) }, ErrUnknownCommodity},
		{"boundary reserve", func() error { return l.ReserveGoods(g.bnd[Initial][0], 0, 1) }, ErrBoundaryReserve},
		{"boundary named reserve", func() error { return l.ReserveLot(g.bnd[Initial][0], Key{Commodity: 0}, 1) }, ErrBoundaryReserve},
		{"boundary release has nothing earmarked", func() error { return l.ReleaseGoods(g.bnd[Initial][0], 0, 1) }, ErrReservationExceeds},
		{"boundary named release has no lot", func() error { return l.ReleaseLot(g.bnd[Initial][0], Key{Commodity: 0}, 1) }, ErrUnknownLot},
		{"unknown lot reserve", func() error { return l.ReserveLot(h, Key{Commodity: 1, Durability: 3}, 1) }, ErrUnknownLot},
		{"unknown lot release", func() error { return l.ReleaseLot(h, Key{Commodity: 1, Durability: 3}, 1) }, ErrUnknownLot},
		{"named reserve too many", func() error { return l.ReserveLot(h, key, 13) }, ErrInsufficientStock},
		{"named release nothing earmarked", func() error { return l.ReleaseLot(h, key, 1) }, ErrReservationExceeds},
		{"unnamed reserve too many", func() error { return l.ReserveGoods(h, 1, 13) }, ErrInsufficientStock},
		{"unnamed release nothing earmarked", func() error { return l.ReleaseGoods(h, 1, 1) }, ErrReservationExceeds},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) { refusedNoAudit(t, g, tc.want, tc.op) })
	}
	// The limits are exact: reserving everything, then one more, then
	// releasing everything, then one more.
	if err := l.ReserveLot(h, key, 12); err != nil {
		t.Fatalf("ReserveLot all: %v", err)
	}
	if err := l.ReserveLot(h, key, 1); !errors.Is(err, ErrInsufficientStock) {
		t.Errorf("ReserveLot beyond all = %v, want ErrInsufficientStock", err)
	}
	if err := l.ReleaseLot(h, key, 12); err != nil {
		t.Fatalf("ReleaseLot all: %v", err)
	}
	if err := l.ReleaseLot(h, key, 1); !errors.Is(err, ErrReservationExceeds) {
		t.Errorf("ReleaseLot beyond all = %v, want ErrReservationExceeds", err)
	}
	if l.HasReservation() {
		t.Errorf("a reservation is left after releasing it all")
	}
}

// TestExpiring_Guards verifies each guard of Expiring: an unknown commodity,
// a commodity that does not spoil, a tick before the current one, an account
// whose hint is after tick, and that earmarked units are not counted.
//
// Requirement: FR-LED-046
func TestExpiring_Guards(t *testing.T) {
	t.Parallel()
	g := newRig(t, scriptMaxMoney)
	g.fund(t)
	l := g.l
	if err := l.EndConstruction(); err != nil {
		t.Fatal(err)
	}
	h := g.holders[0]
	if _, err := l.Expiring(h, 99, 0); !errors.Is(err, ErrUnknownCommodity) {
		t.Errorf("unknown commodity: %v", err)
	}
	if _, err := l.Expiring(999, 0, 0); !errors.Is(err, ErrUnknownAccount) {
		t.Errorf("unknown account: %v", err)
	}
	if _, err := l.Expiring(h, -1, 0); !errors.Is(err, ErrUnknownCommodity) {
		t.Errorf("negative commodity: %v", err)
	}
	at := func(c int, tick int64) int64 {
		n, err := l.Expiring(h, c, tick)
		if err != nil {
			t.Fatalf("Expiring(%d, %d): %v", c, tick, err)
		}
		return n
	}
	if n := at(2, 9); n != 0 {
		t.Errorf("a commodity that does not spoil: %d, want 0", n)
	}
	if n := at(0, l.tick-1); n != 0 {
		t.Errorf("a tick before the current one: %d, want 0", n)
	}
	if n := at(0, 0); n != 0 {
		t.Errorf("before the hint: %d, want 0", n)
	}
	// 3 stated units were born at -1 and expire at the end of tick 1; the 12
	// created in construction were born at 0 and expire at the end of tick 2.
	if n := at(0, 1); n != 3 {
		t.Errorf("at the first expiry tick: %d, want 3", n)
	}
	if n := at(0, 2); n != 15 {
		t.Errorf("at the second expiry tick: %d, want 15", n)
	}
	if err := l.ReserveGoods(h, 0, 4); err != nil {
		t.Fatal(err)
	}
	if n := at(0, 1); n != 0 {
		t.Errorf("with the oldest lot earmarked: %d, want 0", n)
	}
	if n := at(0, 2); n != 11 {
		t.Errorf("with 4 earmarked: %d, want 11", n)
	}
	if n, err := l.Expiring(g.bnd[Initial][0], 0, 1); err != nil || n != 0 {
		t.Errorf("a boundary account: %d, %v, want 0, nil", n, err)
	}
}

// TestCheckSpoilage_Refusals verifies the three refusals of the spoilage
// check by commodity: a flow total overflow, a missing Spoilage boundary, and a
// boundary balance overflow; and that a commodity with nothing spoiled passes
// whatever its boundary.
//
// Requirement: FR-LED-046
func TestCheckSpoilage_Refusals(t *testing.T) {
	t.Parallel()
	g := newRig(t, scriptMaxMoney)
	l := g.l
	nc := len(l.comms)
	flows := make([]int64, l.regions*l.stride)
	l.spoil = make([]int64, l.regions*nc)
	if err := l.checkSpoilage(flows); err != nil {
		t.Fatalf("nothing spoiled: %v", err)
	}
	l.spoil[1*nc+0] = 5
	slot := 1*l.stride + l.off[Spoilage][thingGoods] + 0
	flows[slot] = math.MaxInt64
	if err := l.checkSpoilage(flows); !errors.Is(err, ErrBalanceOverflow) {
		t.Errorf("flow overflow: %v", err)
	}
	flows[slot] = 0
	l.spoil[0*nc+0] = math.MaxInt64
	if err := l.checkSpoilage(flows); !errors.Is(err, ErrBalanceOverflow) {
		t.Errorf("running total overflow: %v", err)
	}
	l.spoil[0*nc+0] = 0
	b := l.boundary[l.off[Spoilage][thingGoods]]
	l.boundary[l.off[Spoilage][thingGoods]] = -1
	if err := l.checkSpoilage(flows); !errors.Is(err, ErrUnknownAccount) {
		t.Errorf("no boundary: %v", err)
	}
	l.boundary[l.off[Spoilage][thingGoods]] = b
	l.accounts[b].balance = math.MaxInt64
	if err := l.checkSpoilage(flows); !errors.Is(err, ErrBalanceOverflow) {
		t.Errorf("boundary overflow: %v", err)
	}
	// A later commodity is still checked after an earlier one passes.
	l.accounts[b].balance = 0
	l.spoil[1*nc+0] = 0
	l.spoil[0*nc+1] = 7
	l.boundary[l.off[Spoilage][thingGoods]+1] = -1
	if err := l.checkSpoilage(flows); !errors.Is(err, ErrUnknownAccount) {
		t.Errorf("second commodity with no boundary: %v", err)
	}
}

// TestBoundariesFit verifies the boundary match shared by transfer and move.
//
// Requirement: FR-LED-002
func TestBoundariesFit(t *testing.T) {
	t.Parallel()
	agent := &account{kind: KindAgent}
	money := &account{kind: KindBoundary, reason: Initial, thing: Money}
	goods := &account{kind: KindBoundary, reason: Initial, thing: 2}
	tests := []struct {
		name  string
		f, t  *account
		r     Reason
		thing int
		want  bool
	}{
		{"no boundary", agent, agent, Trade, Money, true},
		{"money boundary", money, agent, Initial, Money, true},
		{"destination boundary", agent, money, Initial, Money, true},
		{"wrong reason", money, agent, Trade, Money, false},
		{"wrong thing", goods, agent, Initial, Money, false},
		{"goods boundary", agent, goods, Initial, 2, true},
		{"other commodity", agent, goods, Initial, 1, false},
		{"second is wrong", money, goods, Initial, Money, false},
	}
	for _, tc := range tests {
		if got := boundariesFit(tc.f, tc.t, tc.r, tc.thing); got != tc.want {
			t.Errorf("%s: %v, want %v", tc.name, got, tc.want)
		}
	}
}

// TestBalanceSheet_GoodsValueOverflowAndPrices verifies the goods value is the
// exact sum of quantity times price and refuses an overflow of a product or of
// the sum.
//
// Requirement: FR-LED-044
func TestBalanceSheet_GoodsValueOverflowAndPrices(t *testing.T) {
	t.Parallel()
	lots := []Lot{{Commodity: 0, Quantity: 3}, {Commodity: 1, Quantity: 4, Durability: 5}}
	if v, ok := goodsValue(lots, []int64{10, 100}); !ok || v != 430 {
		t.Errorf("goodsValue = %d, %v, want 430, true", v, ok)
	}
	if _, ok := goodsValue(lots, []int64{math.MaxInt64, 0}); ok {
		t.Errorf("a product beyond int64 was accepted")
	}
	if _, ok := goodsValue(lots, []int64{math.MaxInt64 / 3, math.MaxInt64 / 4}); ok {
		t.Errorf("a sum beyond int64 was accepted")
	}
	if v, ok := goodsValue(nil, nil); !ok || v != 0 {
		t.Errorf("goodsValue of no lots = %d, %v", v, ok)
	}
}
