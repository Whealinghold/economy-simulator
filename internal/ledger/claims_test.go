package ledger_test

import (
	"errors"
	"math"
	"reflect"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/ledger"
)

// cfix is the fixture of the claim tests: the money fixture of fix with the
// three loan reasons declared and room for four claims, and the accounts a
// claim runs between. Money: lender 100, everyone else 0, so the boundary is
// at -100. The tick is 0.
type cfix struct {
	*fix
	// lender and borrower are agents of region 0, third another agent of
	// region 0.
	lender, borrower, third int
}

func newCfix(t *testing.T) *cfix {
	t.Helper()
	f := newFix(t, testMaxMoney)
	c := &cfix{fix: f, lender: f.a0}
	var err error
	if c.borrower, err = f.l.Open(ledger.KindAgent, 20, 0); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if c.third, err = f.l.Open(ledger.KindAgent, 21, 0); err != nil {
		t.Fatalf("setup: %v", err)
	}
	return c
}

// lend forms a loan of principal 60 and interest 6, due at tick 10, and
// returns its ID.
func (c *cfix) lend(t *testing.T) int64 {
	t.Helper()
	id, err := c.l.Lend(c.lender, c.borrower, 60, 6, 500, 10)
	if err != nil {
		t.Fatalf("setup: Lend: %v", err)
	}
	return id
}

// claimIdentity checks the identities of spec-credit.md 3.6 items 1 and 4 by
// exact equality, and that money over the non-boundary accounts is unchanged
// at wantMoney (item 2: the loan reasons are internal).
func claimIdentity(t *testing.T, l *ledger.Ledger, wantMoney int64) {
	t.Helper()
	tot := l.ClaimTotals()
	var principal, interest int64
	last := int64(-1)
	for _, c := range l.AppendClaims(nil) {
		principal += c.PrincipalOwed
		interest += c.InterestOwed
		switch {
		case c.ID <= last:
			t.Errorf("claim IDs not ascending: %d after %d", c.ID, last)
		case c.PrincipalOwed < 0 || c.PrincipalOwed > c.Principal || c.InterestOwed < 0 || c.InterestOwed > c.Interest:
			t.Errorf("claim %d owed parts out of range: %+v", c.ID, c)
		case c.PrincipalOwed == 0 && c.InterestOwed == 0:
			t.Errorf("claim %d has nothing owed and still exists", c.ID)
		case c.Creditor == c.Debtor:
			t.Errorf("claim %d has the same account on both sides", c.ID)
		}
		last = c.ID
	}
	if want := tot.Lent + tot.Awarded - tot.Repaid - tot.LostPrincipal; principal != want {
		t.Errorf("principal owed = %d, want Lent+Awarded-Repaid-LostPrincipal = %d (%+v)", principal, want, tot)
	}
	if want := tot.InterestFixed; interest+tot.InterestPaid+tot.LostInterest != want {
		t.Errorf("interest owed + paid + lost = %d, want InterestFixed = %d (%+v)", interest+tot.InterestPaid+tot.LostInterest, want, tot)
	}
	var money int64
	for id := 0; id < l.Len(); id++ {
		v, _ := l.Account(id)
		if v.Kind != ledger.KindBoundary {
			money += v.Balance
		}
	}
	if money != wantMoney {
		t.Errorf("money over non-boundary accounts = %d, want %d", money, wantMoney)
	}
	if got := sum(l); got != 0 {
		t.Errorf("money over all accounts = %d, want 0", got)
	}
}

// TestClaims_Refusals verifies that every refusal of Lend, Settle, WriteOff,
// Reassign, MarkOverdue and Obligate is its own sentinel, that each case
// breaks only the one rule, that a refused call leaves the ledger (the
// accounts, the flow totals, the list and the seven totals) equal to a copy
// of itself, and that the money total is unchanged.
//
// Requirement: FR-LED-007, FR-LED-063, FR-CRD-004
func TestClaims_Refusals(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		setup func(c *cfix, t *testing.T)
		do    func(c *cfix) error
		want  error
	}{
		{name: "lend principal 0", do: func(c *cfix) error { _, err := c.l.Lend(c.lender, c.borrower, 0, 1, 0, 5); return err }, want: ledger.ErrInvalidAmount},
		{name: "lend negative interest", do: func(c *cfix) error { _, err := c.l.Lend(c.lender, c.borrower, 1, -1, 0, 5); return err }, want: ledger.ErrInvalidAmount},
		{name: "lend unknown creditor", do: func(c *cfix) error { _, err := c.l.Lend(99, c.borrower, 1, 0, 0, 5); return err }, want: ledger.ErrUnknownAccount},
		{name: "lend unknown debtor", do: func(c *cfix) error { _, err := c.l.Lend(c.lender, -1, 1, 0, 0, 5); return err }, want: ledger.ErrUnknownAccount},
		{
			name: "lend to a closed debtor",
			setup: func(c *cfix, t *testing.T) {
				if err := c.l.Close(c.third); err != nil {
					t.Fatalf("Close: %v", err)
				}
			},
			do:   func(c *cfix) error { _, err := c.l.Lend(c.lender, c.third, 1, 0, 0, 5); return err },
			want: ledger.ErrClosedAccount,
		},
		{name: "lend to oneself", do: func(c *cfix) error { _, err := c.l.Lend(c.lender, c.lender, 1, 0, 0, 5); return err }, want: ledger.ErrClaimShape},
		{name: "lend due now", do: func(c *cfix) error { _, err := c.l.Lend(c.lender, c.borrower, 1, 0, 0, 0); return err }, want: ledger.ErrClaimShape},
		{name: "lend from an estate", do: func(c *cfix) error { _, err := c.l.Lend(c.e0, c.borrower, 1, 0, 0, 5); return err }, want: ledger.ErrReasonMismatch},
		{name: "lend from a boundary", do: func(c *cfix) error { _, err := c.l.Lend(c.mint, c.borrower, 1, 0, 0, 5); return err }, want: ledger.ErrReasonMismatch},
		{
			name: "lend past MaxClaims",
			setup: func(c *cfix, t *testing.T) {
				for i := 0; i < 4; i++ {
					if _, err := c.l.Lend(c.lender, c.borrower, 1, 0, 0, 5); err != nil {
						t.Fatalf("setup: %v", err)
					}
				}
			},
			do:   func(c *cfix) error { _, err := c.l.Lend(c.lender, c.borrower, 1, 0, 0, 5); return err },
			want: ledger.ErrTooManyClaims,
		},
		{
			// MaxMoney is 1000: principal 1000 with interest 1 is one above.
			name: "lend with principal plus interest above MaxMoney",
			setup: func(c *cfix, t *testing.T) {
				if err := c.l.Transfer(c.mint, c.lender, 900, ledger.Initial); err != nil {
					t.Fatalf("fund: %v", err)
				}
			},
			do:   func(c *cfix) error { _, err := c.l.Lend(c.lender, c.borrower, 1000, 1, 0, 5); return err },
			want: ledger.ErrBalanceOverflow,
		},
		{name: "lend with principal plus interest overflowing int64", do: func(c *cfix) error {
			_, err := c.l.Lend(c.lender, c.borrower, math.MaxInt64, 1, 0, 5)
			return err
		}, want: ledger.ErrBalanceOverflow},
		{name: "lend more than the lender has", do: func(c *cfix) error { _, err := c.l.Lend(c.lender, c.borrower, 101, 0, 0, 5); return err }, want: ledger.ErrInsufficientFunds},
		{
			name: "lend more than the lender has available",
			setup: func(c *cfix, t *testing.T) {
				if err := c.l.ReserveMoney(c.lender, 50); err != nil {
					t.Fatalf("ReserveMoney: %v", err)
				}
			},
			do:   func(c *cfix) error { _, err := c.l.Lend(c.lender, c.borrower, 51, 0, 0, 5); return err },
			want: ledger.ErrInsufficientFunds,
		},
		{
			// The debtor holds exactly MaxMoney; one more unit passes it.
			name: "lend to a debtor at MaxMoney",
			setup: func(c *cfix, t *testing.T) {
				if err := c.l.Transfer(c.mint, c.borrower, testMaxMoney, ledger.Initial); err != nil {
					t.Fatalf("fund: %v", err)
				}
			},
			do:   func(c *cfix) error { _, err := c.l.Lend(c.lender, c.borrower, 1, 0, 0, 5); return err },
			want: ledger.ErrBalanceOverflow,
		},

		{name: "settle an unknown claim", do: func(c *cfix) error { return c.l.Settle(7, 1, 1) }, want: ledger.ErrUnknownClaim},
		{name: "settle both parts 0", setup: lendSetup, do: func(c *cfix) error { return c.l.Settle(0, 0, 0) }, want: ledger.ErrInvalidAmount},
		{name: "settle negative interest", setup: lendSetup, do: func(c *cfix) error { return c.l.Settle(0, -1, 5) }, want: ledger.ErrInvalidAmount},
		{name: "settle negative principal", setup: lendSetup, do: func(c *cfix) error { return c.l.Settle(0, 1, -5) }, want: ledger.ErrInvalidAmount},
		{name: "settle interest above owed", setup: lendSetup, do: func(c *cfix) error { return c.l.Settle(0, 7, 0) }, want: ledger.ErrInvalidAmount},
		{name: "settle principal above owed", setup: lendSetup, do: func(c *cfix) error { return c.l.Settle(0, 0, 61) }, want: ledger.ErrInvalidAmount},
		{
			// The loan moved 60 to the borrower; they hold 60 and owe 66.
			name: "settle more than the debtor has",
			setup: func(c *cfix, t *testing.T) {
				lendSetup(c, t)
				if err := c.l.Transfer(c.borrower, c.third, 55, ledger.Trade); err != nil {
					t.Fatalf("setup: %v", err)
				}
			},
			do:   func(c *cfix) error { return c.l.Settle(0, 6, 60) },
			want: ledger.ErrInsufficientFunds,
		},
		{
			// The creditor holds 40 and is paid 66: 106, and with MaxMoney at
			// 1000 it fits; fund it to 935 so that 66 is one above.
			name: "settle into a creditor at MaxMoney",
			setup: func(c *cfix, t *testing.T) {
				lendSetup(c, t)
				if err := c.l.Transfer(c.mint, c.lender, 935-40, ledger.Initial); err != nil {
					t.Fatalf("fund: %v", err)
				}
				// The borrower holds 60 and must be able to pay 66.
				if err := c.l.Transfer(c.mint, c.borrower, 6, ledger.Initial); err != nil {
					t.Fatalf("fund: %v", err)
				}
			},
			do:   func(c *cfix) error { return c.l.Settle(0, 6, 60) },
			want: ledger.ErrBalanceOverflow,
		},
		{
			// Nobody else in the fixture is a claim party, so the same
			// claim, ended, is simply unknown.
			name: "settle a claim that ended",
			setup: func(c *cfix, t *testing.T) {
				lendSetup(c, t)
				if err := c.l.Transfer(c.mint, c.borrower, 6, ledger.Initial); err != nil {
					t.Fatalf("fund: %v", err)
				}
				if err := c.l.Settle(0, 6, 60); err != nil {
					t.Fatalf("setup: %v", err)
				}
			},
			do:   func(c *cfix) error { return c.l.Settle(0, 1, 0) },
			want: ledger.ErrUnknownClaim,
		},

		{name: "write off an unknown claim", do: func(c *cfix) error { _, _, err := c.l.WriteOff(0); return err }, want: ledger.ErrUnknownClaim},
		{name: "reassign an unknown claim", do: func(c *cfix) error { return c.l.Reassign(0, c.e0) }, want: ledger.ErrUnknownClaim},
		{name: "reassign to an unknown account", setup: lendSetup, do: func(c *cfix) error { return c.l.Reassign(0, 99) }, want: ledger.ErrUnknownAccount},
		{name: "reassign to an agent", setup: lendSetup, do: func(c *cfix) error { return c.l.Reassign(0, c.third) }, want: ledger.ErrReasonMismatch},
		{
			name: "reassign to a closed estate",
			setup: func(c *cfix, t *testing.T) {
				lendSetup(c, t)
				id, err := c.l.Open(ledger.KindEstate, 0, 0)
				if err != nil {
					t.Fatalf("Open: %v", err)
				}
				if err := c.l.Close(id); err != nil {
					t.Fatalf("Close: %v", err)
				}
				c.third = id
			},
			do:   func(c *cfix) error { return c.l.Reassign(0, c.third) },
			want: ledger.ErrClosedAccount,
		},

		{name: "mark overdue an unknown claim", do: func(c *cfix) error { return c.l.MarkOverdue(0, 12) }, want: ledger.ErrUnknownClaim},
		// The claim is due at tick 10: 9 is one below.
		{name: "mark overdue below due", setup: lendSetup, do: func(c *cfix) error { return c.l.MarkOverdue(0, 9) }, want: ledger.ErrClaimShape},

		{name: "obligate amount 0", do: func(c *cfix) error { _, err := c.l.Obligate(c.lender, c.borrower, 0, 5); return err }, want: ledger.ErrInvalidAmount},
		{name: "obligate unknown creditor", do: func(c *cfix) error { _, err := c.l.Obligate(99, c.borrower, 1, 5); return err }, want: ledger.ErrUnknownAccount},
		{name: "obligate to oneself", do: func(c *cfix) error { _, err := c.l.Obligate(c.lender, c.lender, 1, 5); return err }, want: ledger.ErrClaimShape},
		{name: "obligate due now", do: func(c *cfix) error { _, err := c.l.Obligate(c.lender, c.borrower, 1, 0); return err }, want: ledger.ErrClaimShape},
		{name: "obligate with an estate debtor", do: func(c *cfix) error { _, err := c.l.Obligate(c.lender, c.e0, 1, 5); return err }, want: ledger.ErrClaimShape},
		{name: "obligate with a boundary creditor", do: func(c *cfix) error { _, err := c.l.Obligate(c.mint, c.borrower, 1, 5); return err }, want: ledger.ErrClaimShape},
		{
			name: "obligate past MaxClaims",
			setup: func(c *cfix, t *testing.T) {
				for i := 0; i < 4; i++ {
					if _, err := c.l.Obligate(c.lender, c.borrower, 1, 5); err != nil {
						t.Fatalf("setup: %v", err)
					}
				}
			},
			do:   func(c *cfix) error { _, err := c.l.Obligate(c.lender, c.borrower, 1, 5); return err },
			want: ledger.ErrTooManyClaims,
		},
		{
			name: "obligate with Awarded at the edge of int64",
			setup: func(c *cfix, t *testing.T) {
				if _, err := c.l.Obligate(c.lender, c.borrower, math.MaxInt64, 5); err != nil {
					t.Fatalf("setup: %v", err)
				}
			},
			do:   func(c *cfix) error { _, err := c.l.Obligate(c.lender, c.borrower, 1, 5); return err },
			want: ledger.ErrBalanceOverflow,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := newCfix(t)
			if tt.setup != nil {
				tt.setup(c, t)
			}
			before := snapshot(c.l)
			err := tt.do(c)
			if !errors.Is(err, tt.want) {
				t.Fatalf("got error %v, want %v", err, tt.want)
			}
			if after := snapshot(c.l); !reflect.DeepEqual(before, after) {
				t.Errorf("refused call changed the ledger:\n before %+v\n after  %+v", before, after)
			}
		})
	}
}

// lendSetup forms the loan of cfix.lend, claim 0.
func lendSetup(c *cfix, t *testing.T) { t.Helper(); c.lend(t) }

// TestLend_FormsClaimAndTransfer verifies that Lend moves the principal under
// LoanOut and appends the claim of CRD 3.1.1 with the next ID, the current
// tick, Overdue -1 and the origin loan, that the IDs are ascending and not
// reused after an end, and that the totals and the money total are exact.
//
// Requirement: FR-LED-063, FR-CRD-004, FR-CRD-008
func TestLend_FormsClaimAndTransfer(t *testing.T) {
	t.Parallel()
	c := newCfix(t)
	c.l.AdvanceTick()
	c.l.AdvanceTick() // the tick is 2
	id0, err := c.l.Lend(c.lender, c.borrower, 60, 6, 500, 10)
	if err != nil || id0 != 0 {
		t.Fatalf("Lend = %d, %v, want 0, nil", id0, err)
	}
	id1, err := c.l.Lend(c.lender, c.third, 10, 0, 0, 3)
	if err != nil || id1 != 1 {
		t.Fatalf("Lend = %d, %v, want 1, nil", id1, err)
	}
	want := []ledger.Claim{
		{ID: 0, Creditor: c.lender, Debtor: c.borrower, Principal: 60, Interest: 6, PrincipalOwed: 60, InterestOwed: 6, RateBP: 500, Formed: 2, Due: 10, Overdue: -1, Origin: ledger.OriginLoan},
		{ID: 1, Creditor: c.lender, Debtor: c.third, Principal: 10, PrincipalOwed: 10, Formed: 2, Due: 3, Overdue: -1, Origin: ledger.OriginLoan},
	}
	if got := c.l.AppendClaims(nil); !reflect.DeepEqual(got, want) {
		t.Errorf("claims = %+v, want %+v", got, want)
	}
	for id, bal := range map[int]int64{c.lender: 30, c.borrower: 60, c.third: 10} {
		if v, _ := c.l.Account(id); v.Balance != bal {
			t.Errorf("balance of account %d = %d, want %d", id, v.Balance, bal)
		}
	}
	// Lent = 60 + 10, InterestFixed = 6.
	if got, want := c.l.ClaimTotals(), (ledger.ClaimTotals{Lent: 70, InterestFixed: 6}); got != want {
		t.Errorf("totals = %+v, want %+v", got, want)
	}
	// The principal is a LoanOut flow in the construction bucket, in region 0.
	var out int64
	for _, fl := range c.l.AppendFlows(nil, true) {
		if fl.Reason == ledger.LoanOut && fl.Commodity == ledger.Money {
			out += fl.Total
		}
	}
	if out != 70 {
		t.Errorf("LoanOut flow = %d, want 70", out)
	}
	claimIdentity(t, c.l, 100)

	// An ended claim's ID is not reused.
	if _, _, err := c.l.WriteOff(1); err != nil {
		t.Fatalf("WriteOff: %v", err)
	}
	if id, err := c.l.Lend(c.lender, c.third, 1, 0, 0, 3); err != nil || id != 2 {
		t.Errorf("Lend after an end = %d, %v, want 2, nil", id, err)
	}
	claimIdentity(t, c.l, 100)
}

// TestSettle_InterestFirstThenPrincipal verifies that Settle pays interest
// under LoanInterest and principal under LoanRepay, keeps the claim while
// anything is owed, removes it when both parts reach 0, and counts each part.
//
// Requirement: FR-LED-062, FR-LED-063, FR-LED-028, FR-CRD-004, FR-CRD-005
func TestSettle_InterestFirstThenPrincipal(t *testing.T) {
	t.Parallel()
	c := newCfix(t)
	c.lend(t) // 60 + 6, the borrower holds 60
	// Fund the borrower so that it can pay 66: 6 more.
	if err := c.l.Transfer(c.mint, c.borrower, 6, ledger.Initial); err != nil {
		t.Fatalf("fund: %v", err)
	}
	money := int64(106)

	steps := []struct {
		name                string
		interest, principal int64
		wantOwed            [2]int64 // principal, interest; zero zero means gone
		tot                 ledger.ClaimTotals
	}{
		{"part of the interest", 2, 0, [2]int64{60, 4}, ledger.ClaimTotals{Lent: 60, InterestFixed: 6, InterestPaid: 2}},
		{"part of the principal", 0, 25, [2]int64{35, 4}, ledger.ClaimTotals{Lent: 60, InterestFixed: 6, InterestPaid: 2, Repaid: 25}},
		{"both parts, the rest", 4, 35, [2]int64{0, 0}, ledger.ClaimTotals{Lent: 60, InterestFixed: 6, InterestPaid: 6, Repaid: 60}},
	}
	for _, st := range steps {
		if err := c.l.Settle(0, st.interest, st.principal); err != nil {
			t.Fatalf("%s: Settle: %v", st.name, err)
		}
		claims := c.l.AppendClaims(nil)
		if st.wantOwed == [2]int64{} {
			if len(claims) != 0 {
				t.Errorf("%s: claims = %+v, want none", st.name, claims)
			}
		} else if len(claims) != 1 || claims[0].PrincipalOwed != st.wantOwed[0] || claims[0].InterestOwed != st.wantOwed[1] {
			t.Errorf("%s: claims = %+v, want owed %v", st.name, claims, st.wantOwed)
		}
		if got := c.l.ClaimTotals(); got != st.tot {
			t.Errorf("%s: totals = %+v, want %+v", st.name, got, st.tot)
		}
		claimIdentity(t, c.l, money)
	}
	// The lender has 40 + 66 and the borrower nothing; the interest flow is 6
	// and the principal flow 60 (plus the 60 lent).
	if v, _ := c.l.Account(c.lender); v.Balance != 106 {
		t.Errorf("lender balance = %d, want 106", v.Balance)
	}
	flow := map[ledger.Reason]int64{}
	for _, fl := range c.l.AppendFlows(nil, true) {
		if fl.Commodity == ledger.Money {
			flow[fl.Reason] += fl.Total
		}
	}
	if flow[ledger.LoanInterest] != 6 || flow[ledger.LoanRepay] != 60 || flow[ledger.LoanOut] != 60 {
		t.Errorf("flows = %v, want interest 6, repay 60, out 60", flow)
	}
}

// TestSettle_SkipsAZeroPartAndRefusesBoth verifies D9: a part of 0 skips its
// transfer (so a claim with no interest can be settled, and a part-0 reason
// need not be declared), and both parts 0 is ErrInvalidAmount.
//
// Requirement: FR-LED-063, FR-CRD-004
func TestSettle_SkipsAZeroPartAndRefusesBoth(t *testing.T) {
	t.Parallel()
	// Only LoanRepay is declared: a principal-only settlement works, and an
	// interest part is a reason mismatch.
	l, err := ledger.New(ledger.Config{
		Regions: 1, MaxMoney: 1000, MaxQuantity: 10, MaxClaims: 2,
		Declared: []ledger.Reason{ledger.LoanOut, ledger.LoanRepay},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	mint, _ := l.OpenBoundary(ledger.Initial, ledger.Money)
	a, _ := l.Open(ledger.KindAgent, 0, 0)
	b, _ := l.Open(ledger.KindAgent, 1, 0)
	if err := l.Transfer(mint, a, 100, ledger.Initial); err != nil {
		t.Fatalf("fund: %v", err)
	}
	id, err := l.Lend(a, b, 50, 5, 0, 4)
	if err != nil {
		t.Fatalf("Lend: %v", err)
	}
	if err := l.Settle(id, 0, 0); !errors.Is(err, ledger.ErrInvalidAmount) {
		t.Errorf("Settle(0, 0) = %v, want ErrInvalidAmount", err)
	}
	if err := l.Settle(id, 1, 0); !errors.Is(err, ledger.ErrReasonMismatch) {
		t.Errorf("Settle(interest) with LoanInterest undeclared = %v, want ErrReasonMismatch", err)
	}
	if err := l.Settle(id, 0, 50); err != nil {
		t.Fatalf("Settle(principal only) = %v, want nil", err)
	}
	cl := l.AppendClaims(nil)
	if len(cl) != 1 || cl[0].PrincipalOwed != 0 || cl[0].InterestOwed != 5 {
		t.Errorf("claims = %+v, want one with only the interest owed", cl)
	}
}

// TestObligate_MovesNoMoney verifies that Obligate appends a payout claim
// with Interest 0, counts its principal in Awarded and in no other total, and
// moves no money and no flow total.
//
// Requirement: FR-LED-063, FR-CRD-004, FR-CRD-005
func TestObligate_MovesNoMoney(t *testing.T) {
	t.Parallel()
	c := newCfix(t)
	flows := c.l.AppendFlows(nil, true)
	id, err := c.l.Obligate(c.lender, c.borrower, 40, 7)
	if err != nil || id != 0 {
		t.Fatalf("Obligate = %d, %v, want 0, nil", id, err)
	}
	want := []ledger.Claim{{ID: 0, Creditor: c.lender, Debtor: c.borrower, Principal: 40, PrincipalOwed: 40, Due: 7, Overdue: -1, Origin: ledger.OriginPayout}}
	if got := c.l.AppendClaims(nil); !reflect.DeepEqual(got, want) {
		t.Errorf("claims = %+v, want %+v", got, want)
	}
	if got, want := c.l.ClaimTotals(), (ledger.ClaimTotals{Awarded: 40}); got != want {
		t.Errorf("totals = %+v, want %+v", got, want)
	}
	if got := c.l.AppendFlows(nil, true); !reflect.DeepEqual(got, flows) {
		t.Errorf("flows = %+v, want %+v: Obligate moves no money", got, flows)
	}
	if v, _ := c.l.Account(c.lender); v.Balance != 100 {
		t.Errorf("creditor balance = %d, want 100", v.Balance)
	}
	// Settled like any other claim: the repayment counts in Repaid.
	if err := c.l.Transfer(c.lender, c.borrower, 40, ledger.Trade); err != nil {
		t.Fatalf("fund debtor: %v", err)
	}
	if err := c.l.Settle(0, 0, 40); err != nil {
		t.Fatalf("Settle: %v", err)
	}
	claimIdentity(t, c.l, 100)
	if got := c.l.ClaimTotals(); got.Repaid != 40 || got.Awarded != 40 {
		t.Errorf("totals = %+v, want Repaid 40 and Awarded 40", got)
	}
}

// TestWriteOff_ReturnsWhatWasOwed verifies that WriteOff removes the claim,
// returns the principal and interest still owed, counts them in the lost
// totals, and moves no money.
//
// Requirement: FR-LED-063, FR-CRD-004, FR-CRD-005
func TestWriteOff_ReturnsWhatWasOwed(t *testing.T) {
	t.Parallel()
	c := newCfix(t)
	c.lend(t)
	if err := c.l.Settle(0, 2, 10); err != nil {
		t.Fatalf("Settle: %v", err)
	}
	// Owed after a payment of 2 interest and 10 principal: 50 and 4.
	p, i, err := c.l.WriteOff(0)
	if err != nil || p != 50 || i != 4 {
		t.Fatalf("WriteOff = %d, %d, %v, want 50, 4, nil", p, i, err)
	}
	if got := c.l.AppendClaims(nil); len(got) != 0 {
		t.Errorf("claims = %+v, want none", got)
	}
	want := ledger.ClaimTotals{Lent: 60, InterestFixed: 6, InterestPaid: 2, Repaid: 10, LostPrincipal: 50, LostInterest: 4}
	if got := c.l.ClaimTotals(); got != want {
		t.Errorf("totals = %+v, want %+v", got, want)
	}
	claimIdentity(t, c.l, 100)
}

// TestReassign_ToAnEstate verifies that Reassign changes the creditor to an
// estate account and nothing else, with no transfer, and that a later
// settlement pays the estate.
//
// Requirement: FR-LED-063, FR-CRD-004
func TestReassign_ToAnEstate(t *testing.T) {
	t.Parallel()
	c := newCfix(t)
	c.lend(t)
	flows := c.l.AppendFlows(nil, true)
	if err := c.l.Reassign(0, c.e0); err != nil {
		t.Fatalf("Reassign: %v", err)
	}
	got := c.l.AppendClaims(nil)
	want := []ledger.Claim{{ID: 0, Creditor: c.e0, Debtor: c.borrower, Principal: 60, Interest: 6, PrincipalOwed: 60, InterestOwed: 6, RateBP: 500, Due: 10, Overdue: -1}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("claims = %+v, want %+v", got, want)
	}
	if after := c.l.AppendFlows(nil, true); !reflect.DeepEqual(after, flows) {
		t.Errorf("flows changed: %+v, want %+v", after, flows)
	}
	if err := c.l.Transfer(c.mint, c.borrower, 6, ledger.Initial); err != nil {
		t.Fatalf("fund: %v", err)
	}
	if err := c.l.Settle(0, 6, 60); err != nil {
		t.Fatalf("Settle: %v", err)
	}
	if v, _ := c.l.Account(c.e0); v.Balance != 66 {
		t.Errorf("estate balance = %d, want 66", v.Balance)
	}
	claimIdentity(t, c.l, 106)
}

// TestMarkOverdue_SetsOnlyTheFirstTick verifies D9: MarkOverdue sets Overdue
// when it is -1, leaves it when already set, and changes no other field.
//
// Requirement: FR-LED-063, FR-CRD-004
func TestMarkOverdue_SetsOnlyTheFirstTick(t *testing.T) {
	t.Parallel()
	c := newCfix(t)
	c.lend(t) // due at 10
	before := c.l.AppendClaims(nil)[0]
	for _, st := range []struct{ tick, want int64 }{{10, 10}, {13, 10}} {
		if err := c.l.MarkOverdue(0, st.tick); err != nil {
			t.Fatalf("MarkOverdue(%d): %v", st.tick, err)
		}
		got := c.l.AppendClaims(nil)[0]
		if got.Overdue != st.want {
			t.Errorf("after MarkOverdue(%d) Overdue = %d, want %d", st.tick, got.Overdue, st.want)
		}
		got.Overdue = before.Overdue
		if got != before {
			t.Errorf("MarkOverdue changed another field: %+v, want %+v", got, before)
		}
	}
	claimIdentity(t, c.l, 100)
}

// TestClose_ChecksClaimLast verifies the order of D7 and LED 3.3.1: a
// reservation, then money or goods, then an open claim; and that an account
// closes once its claims have ended, with no account opened for a claim.
//
// Requirement: FR-LED-009
func TestClose_ChecksClaimLast(t *testing.T) {
	t.Parallel()
	c := newCfix(t)
	n := c.l.Len()
	c.lend(t) // lender 40 + a claim held, borrower 60 + a claim owed
	if c.l.Len() != n {
		t.Fatalf("Lend opened an account: Len %d, want %d", c.l.Len(), n)
	}
	if err := c.l.ReserveMoney(c.borrower, 5); err != nil {
		t.Fatalf("ReserveMoney: %v", err)
	}
	steps := []struct {
		name  string
		id    int
		setup func()
		want  error
	}{
		{"reservation, money and claim held", c.borrower, nil, ledger.ErrOpenReservation},
		{"money and claim held", c.borrower, func() {
			if err := c.l.ReleaseMoney(c.borrower, 5); err != nil {
				t.Fatalf("ReleaseMoney: %v", err)
			}
		}, ledger.ErrNotEmpty},
		{"claim owed only", c.borrower, func() {
			if err := c.l.Transfer(c.borrower, c.third, 60, ledger.Trade); err != nil {
				t.Fatalf("Transfer: %v", err)
			}
		}, ledger.ErrOpenClaim},
		{"claim held only", c.lender, func() {
			if err := c.l.Transfer(c.lender, c.third, 40, ledger.Trade); err != nil {
				t.Fatalf("Transfer: %v", err)
			}
		}, ledger.ErrOpenClaim},
	}
	for _, st := range steps {
		if st.setup != nil {
			st.setup()
		}
		before := snapshot(c.l)
		if err := c.l.Close(st.id); !errors.Is(err, st.want) {
			t.Fatalf("%s: Close = %v, want %v", st.name, err, st.want)
		}
		if after := snapshot(c.l); !reflect.DeepEqual(before, after) {
			t.Errorf("%s: a refused Close changed the ledger", st.name)
		}
	}
	// The claim ends by a write-off, and then both parties close.
	if _, _, err := c.l.WriteOff(0); err != nil {
		t.Fatalf("WriteOff: %v", err)
	}
	for _, id := range []int{c.borrower, c.lender} {
		if err := c.l.Close(id); err != nil {
			t.Errorf("Close(%d) after the claim ended = %v, want nil", id, err)
		}
	}
}

// TestClaims_NoModuleNoClaims verifies A1: with MaxClaims 0 (every world of
// Stage 1) nothing can open a claim, and with the loan reasons not declared
// Lend and Settle are a reason mismatch and the reasons move no money.
//
// Requirement: FR-LED-052, FR-LED-063
func TestClaims_NoModuleNoClaims(t *testing.T) {
	t.Parallel()
	build := func(maxClaims int, declared ...ledger.Reason) (*ledger.Ledger, int, int) {
		l, err := ledger.New(ledger.Config{Regions: 1, MaxMoney: 1000, MaxQuantity: 10, MaxClaims: maxClaims, Declared: declared})
		if err != nil {
			t.Fatalf("New: %v", err)
		}
		mint, _ := l.OpenBoundary(ledger.Initial, ledger.Money)
		a, _ := l.Open(ledger.KindAgent, 0, 0)
		b, _ := l.Open(ledger.KindAgent, 1, 0)
		if err := l.Transfer(mint, a, 100, ledger.Initial); err != nil {
			t.Fatalf("fund: %v", err)
		}
		return l, a, b
	}

	t.Run("MaxClaims 0 opens none", func(t *testing.T) {
		t.Parallel()
		l, a, b := build(0, ledger.LoanOut)
		if _, err := l.Lend(a, b, 1, 0, 0, 5); !errors.Is(err, ledger.ErrTooManyClaims) {
			t.Errorf("Lend = %v, want ErrTooManyClaims", err)
		}
		if _, err := l.Obligate(a, b, 1, 5); !errors.Is(err, ledger.ErrTooManyClaims) {
			t.Errorf("Obligate = %v, want ErrTooManyClaims", err)
		}
	})
	t.Run("reasons undeclared", func(t *testing.T) {
		t.Parallel()
		l, a, b := build(2)
		before := snapshot(l)
		if _, err := l.Lend(a, b, 1, 0, 0, 5); !errors.Is(err, ledger.ErrReasonMismatch) {
			t.Errorf("Lend = %v, want ErrReasonMismatch", err)
		}
		for _, r := range []ledger.Reason{ledger.LoanOut, ledger.LoanRepay, ledger.LoanInterest} {
			if err := l.Transfer(a, b, 1, r); !errors.Is(err, ledger.ErrReasonMismatch) {
				t.Errorf("Transfer(reason %d) = %v, want ErrReasonMismatch", r, err)
			}
			if _, err := l.OpenBoundary(r, ledger.Money); !errors.Is(err, ledger.ErrReasonMismatch) {
				t.Errorf("OpenBoundary(reason %d) = %v, want ErrReasonMismatch", r, err)
			}
		}
		if after := snapshot(l); !reflect.DeepEqual(before, after) {
			t.Errorf("a refusal changed the ledger")
		}
	})
	t.Run("an obligation needs no reason", func(t *testing.T) {
		t.Parallel()
		l, a, b := build(2)
		if id, err := l.Obligate(a, b, 5, 3); err != nil || id != 0 {
			t.Errorf("Obligate = %d, %v, want 0, nil", id, err)
		}
		if _, _, err := l.WriteOff(0); err != nil {
			t.Errorf("WriteOff = %v, want nil", err)
		}
	})
}

// TestNew_ChecksClaimConfiguration verifies that New refuses a declared
// reason that is built in, out of range or listed twice, and a negative
// MaxClaims.
//
// Requirement: FR-LED-052
func TestNew_ChecksClaimConfiguration(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		cfg  ledger.Config
		want error
	}{
		{"declared loan reasons", ledger.Config{Declared: []ledger.Reason{ledger.LoanOut, ledger.LoanRepay, ledger.LoanInterest}, MaxClaims: 3}, nil},
		{"a built-in reason", ledger.Config{Declared: []ledger.Reason{ledger.Trade}}, ledger.ErrInvalidConfig},
		{"out of range", ledger.Config{Declared: []ledger.Reason{ledger.NumReasons}}, ledger.ErrInvalidConfig},
		{"twice", ledger.Config{Declared: []ledger.Reason{ledger.LoanOut, ledger.LoanOut}}, ledger.ErrInvalidConfig},
		{"negative MaxClaims", ledger.Config{MaxClaims: -1}, ledger.ErrInvalidConfig},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.cfg.Regions, tt.cfg.MaxMoney, tt.cfg.MaxQuantity = 1, 10, 10
			if _, err := ledger.New(tt.cfg); !errors.Is(err, tt.want) {
				t.Errorf("New(%+v) = %v, want %v", tt.cfg, err, tt.want)
			}
		})
	}
}

// TestBalanceSheet_ValuesGoodsAndClaims verifies the balance sheet on a
// hand-computed account, and that it changes nothing.
//
// Requirement: FR-LED-026
func TestBalanceSheet_ValuesGoodsAndClaims(t *testing.T) {
	t.Parallel()
	c := newCfix(t)
	// The borrower is lent 60 owing 66, holds 3 units of commodity 0 and 2 of
	// commodity 1, and is owed 12 by the lender through a payout claim.
	c.lend(t)
	if err := c.l.Move(c.goodsBoundary, c.borrower, 0, 3, ledger.Initial); err != nil {
		t.Fatalf("Move: %v", err)
	}
	g1, err := c.l.OpenBoundary(ledger.Initial, 1)
	if err != nil {
		t.Fatalf("OpenBoundary: %v", err)
	}
	if err := c.l.Move(g1, c.borrower, 1, 2, ledger.Initial); err != nil {
		t.Fatalf("Move: %v", err)
	}
	if _, err := c.l.Obligate(c.borrower, c.lender, 12, 20); err != nil {
		t.Fatalf("Obligate: %v", err)
	}
	before := snapshot(c.l)
	got, err := c.l.BalanceSheet(c.borrower, []int64{7, 11})
	// Money 60; goods 3*7 + 2*11 = 43; claims held 12; owed 60 + 6 = 66;
	// assets 60 + 43 + 12 = 115; net worth 115 - 66 = 49.
	want := ledger.Sheet{Money: 60, Goods: 43, ClaimsHeld: 12, ClaimsOwed: 66, Assets: 115, NetWorth: 49}
	if err != nil || got != want {
		t.Errorf("BalanceSheet = %+v, %v, want %+v, nil", got, err, want)
	}
	if !reflect.DeepEqual(before, snapshot(c.l)) {
		t.Errorf("BalanceSheet changed the ledger")
	}
	// The lender is the mirror: 40 money, 12 owed to the borrower, 66 held.
	got, err = c.l.BalanceSheet(c.lender, []int64{7, 11})
	want = ledger.Sheet{Money: 40, ClaimsHeld: 66, ClaimsOwed: 12, Assets: 106, NetWorth: 94}
	if err != nil || got != want {
		t.Errorf("BalanceSheet(lender) = %+v, %v, want %+v, nil", got, err, want)
	}
}

// TestBalanceSheet_Refusals verifies that a valuation that overflows is
// refused and not wrapped, that a net worth may be negative, and that an
// unknown, closed or boundary account, a short or negative price table are
// each refused with their own sentinel.
//
// Requirement: FR-LED-026
func TestBalanceSheet_Refusals(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		id     func(c *cfix) int
		prices []int64
		want   error
	}{
		{"unknown account", func(c *cfix) int { return 99 }, []int64{1, 1}, ledger.ErrUnknownAccount},
		{"negative account", func(c *cfix) int { return -1 }, []int64{1, 1}, ledger.ErrUnknownAccount},
		{"boundary account", func(c *cfix) int { return c.mint }, []int64{1, 1}, ledger.ErrInvalidKind},
		{"closed account", func(c *cfix) int {
			if err := c.l.Close(c.third); err != nil {
				panic(err)
			}
			return c.third
		}, []int64{1, 1}, ledger.ErrClosedAccount},
		{"price table too short", func(c *cfix) int { return c.lender }, []int64{1}, ledger.ErrUnknownCommodity},
		{"negative price", func(c *cfix) int { return c.lender }, []int64{1, -1}, ledger.ErrInvalidAmount},
		// 5 units at MaxInt64/4 is above MaxInt64.
		{"valuation overflows", func(c *cfix) int {
			if err := c.l.Move(c.goodsBoundary, c.lender, 0, 5, ledger.Initial); err != nil {
				panic(err)
			}
			return c.lender
		}, []int64{math.MaxInt64 / 4, 0}, ledger.ErrBalanceOverflow},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := newCfix(t)
			id := tt.id(c)
			before := snapshot(c.l)
			if _, err := c.l.BalanceSheet(id, tt.prices); !errors.Is(err, tt.want) {
				t.Errorf("BalanceSheet = %v, want %v", err, tt.want)
			}
			if after := snapshot(c.l); !reflect.DeepEqual(before, after) {
				t.Errorf("a refused BalanceSheet changed the ledger")
			}
		})
	}

	t.Run("negative net worth", func(t *testing.T) {
		t.Parallel()
		c := newCfix(t)
		// Spend the borrowed 60: money 0, owed 66, so net worth is -66.
		c.lend(t)
		if err := c.l.Transfer(c.borrower, c.third, 60, ledger.Trade); err != nil {
			t.Fatalf("Transfer: %v", err)
		}
		got, err := c.l.BalanceSheet(c.borrower, []int64{0, 0})
		if want := (ledger.Sheet{ClaimsOwed: 66, NetWorth: -66}); err != nil || got != want {
			t.Errorf("BalanceSheet = %+v, %v, want %+v, nil", got, err, want)
		}
	})
	t.Run("assets overflow", func(t *testing.T) {
		t.Parallel()
		c := newCfix(t)
		// A payout claim of MaxInt64 held by an account with 100 of money:
		// the assets pass int64 and are refused, not wrapped.
		if _, err := c.l.Obligate(c.lender, c.borrower, math.MaxInt64, 5); err != nil {
			t.Fatalf("Obligate: %v", err)
		}
		if _, err := c.l.BalanceSheet(c.lender, []int64{0, 0}); !errors.Is(err, ledger.ErrBalanceOverflow) {
			t.Errorf("BalanceSheet = %v, want ErrBalanceOverflow", err)
		}
	})
}
