package ledger_test

import (
	"errors"
	"math"
	"reflect"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/ledger"
)

// TestClaims_UnknownIDBelowAnOpenClaim verifies that the ID of an ended claim
// is unknown even while a claim with a higher ID is open, for every call that
// takes a claim ID.
//
// Requirement: FR-LED-063, FR-CRD-004
func TestClaims_UnknownIDBelowAnOpenClaim(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		do   func(c *cfix) error
	}{
		{"settle", func(c *cfix) error { return c.l.Settle(0, 0, 1) }},
		{"write off", func(c *cfix) error { _, _, err := c.l.WriteOff(0); return err }},
		{"reassign", func(c *cfix) error { return c.l.Reassign(0, c.e0) }},
		{"mark overdue", func(c *cfix) error { return c.l.MarkOverdue(0, 20) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := newCfix(t)
			// Claims 0 and 1 are open; ending 0 leaves only 1, whose ID is above 0.
			c.lend(t)
			if _, err := c.l.Lend(c.lender, c.third, 10, 0, 0, 3); err != nil {
				t.Fatalf("setup: Lend: %v", err)
			}
			if _, _, err := c.l.WriteOff(0); err != nil {
				t.Fatalf("setup: WriteOff: %v", err)
			}
			if err := tt.do(c); !errors.Is(err, ledger.ErrUnknownClaim) {
				t.Errorf("got %v, want ErrUnknownClaim", err)
			}
		})
	}
}

// TestClose_AccountOutsideEveryClaim verifies that an account that is neither
// the creditor nor the debtor of any open claim closes, whatever its ID is
// relative to the parties' IDs.
//
// Requirement: FR-LED-009
func TestClose_AccountOutsideEveryClaim(t *testing.T) {
	t.Parallel()
	c := newCfix(t)
	// Claim 0: lender (2) to borrower (5). Claim 1: borrower (5) to third (6).
	// The estate (4) is above the creditor of claim 0 and below its debtor and
	// both parties of claim 1; the other agent (3) is likewise.
	c.lend(t)
	if _, err := c.l.Lend(c.borrower, c.third, 10, 0, 0, 3); err != nil {
		t.Fatalf("setup: Lend: %v", err)
	}
	if err := c.l.Close(c.e0); err != nil {
		t.Errorf("Close(estate outside every claim) = %v, want nil", err)
	}
	if err := c.l.Close(c.a1); err != nil {
		t.Errorf("Close(agent outside every claim) = %v, want nil", err)
	}
}

// TestClaims_DueAndMaxMoneyEdges verifies the edges of the formation checks:
// a due below the current tick is refused as well as one equal to it, and a
// loan owing exactly MaxMoney is accepted.
//
// Requirement: FR-LED-063, FR-CRD-004
func TestClaims_DueAndMaxMoneyEdges(t *testing.T) {
	t.Parallel()
	t.Run("lend due below the tick", func(t *testing.T) {
		t.Parallel()
		c := newCfix(t)
		for range 3 {
			c.l.AdvanceTick()
		}
		// The tick is 3: a due of 2 is below it, a due of 3 equals it.
		for _, due := range []int64{2, 3} {
			if _, err := c.l.Lend(c.lender, c.borrower, 1, 0, 0, due); !errors.Is(err, ledger.ErrClaimShape) {
				t.Errorf("Lend(due %d) = %v, want ErrClaimShape", due, err)
			}
		}
	})
	t.Run("obligate due below the tick", func(t *testing.T) {
		t.Parallel()
		c := newCfix(t)
		for range 3 {
			c.l.AdvanceTick()
		}
		for _, due := range []int64{2, 3} {
			if _, err := c.l.Obligate(c.lender, c.borrower, 1, due); !errors.Is(err, ledger.ErrClaimShape) {
				t.Errorf("Obligate(due %d) = %v, want ErrClaimShape", due, err)
			}
		}
	})
	t.Run("lend owing exactly MaxMoney", func(t *testing.T) {
		t.Parallel()
		c := newCfix(t)
		// Principal 100 (all the lender holds) plus interest 900 is MaxMoney, 1000.
		if _, err := c.l.Lend(c.lender, c.borrower, 100, 900, 0, 5); err != nil {
			t.Errorf("Lend(owing MaxMoney) = %v, want nil", err)
		}
		if _, err := c.l.Lend(c.lender, c.borrower, 1, 1000, 0, 5); !errors.Is(err, ledger.ErrBalanceOverflow) {
			t.Errorf("Lend(owing MaxMoney+1) = %v, want ErrBalanceOverflow", err)
		}
	})
}

// TestObligate_PartyKinds verifies that a boundary account is neither a
// creditor nor a debtor of a payout claim.
//
// Requirement: FR-CRD-004
func TestObligate_PartyKinds(t *testing.T) {
	t.Parallel()
	c := newCfix(t)
	if _, err := c.l.Obligate(c.mint, c.borrower, 5, 5); !errors.Is(err, ledger.ErrClaimShape) {
		t.Errorf("Obligate(boundary creditor) = %v, want ErrClaimShape", err)
	}
	if _, err := c.l.Obligate(c.lender, c.mint, 5, 5); !errors.Is(err, ledger.ErrClaimShape) {
		t.Errorf("Obligate(boundary debtor) = %v, want ErrClaimShape", err)
	}
	if _, err := c.l.Obligate(c.lender, c.e0, 5, 5); !errors.Is(err, ledger.ErrClaimShape) {
		t.Errorf("Obligate(estate debtor) = %v, want ErrClaimShape", err)
	}
}

// TestSettle_MoneyEdges verifies the money checks of Settle at their edges:
// reserved money is not available to pay, exactly the available money is, and
// a creditor ending at exactly MaxMoney is accepted.
//
// Requirement: FR-LED-063, FR-CRD-005
func TestSettle_MoneyEdges(t *testing.T) {
	t.Parallel()
	t.Run("reserved money is not available", func(t *testing.T) {
		t.Parallel()
		c := newCfix(t)
		id := c.lend(t) // the borrower holds 60
		if err := c.l.ReserveMoney(c.borrower, 50); err != nil {
			t.Fatalf("ReserveMoney: %v", err)
		}
		// Available is 60 - 50 = 10.
		if err := c.l.Settle(id, 0, 20); !errors.Is(err, ledger.ErrInsufficientFunds) {
			t.Errorf("Settle(20) = %v, want ErrInsufficientFunds", err)
		}
		if err := c.l.Settle(id, 0, 10); err != nil {
			t.Errorf("Settle(10) = %v, want nil", err)
		}
	})
	t.Run("creditor ends at MaxMoney", func(t *testing.T) {
		t.Parallel()
		c := newCfix(t)
		id := c.lend(t) // the lender holds 40 and is owed 66
		// The borrower holds 60 and needs 6 more to pay 66.
		if err := c.l.Transfer(c.mint, c.borrower, 6, ledger.Initial); err != nil {
			t.Fatalf("setup: Transfer: %v", err)
		}
		// 40 + 894 = 934, and 934 + 66 = 1000 = MaxMoney.
		if err := c.l.Transfer(c.mint, c.lender, 894, ledger.Initial); err != nil {
			t.Fatalf("setup: Transfer: %v", err)
		}
		if err := c.l.Settle(id, 6, 60); err != nil {
			t.Errorf("Settle to exactly MaxMoney = %v, want nil", err)
		}
	})
}

// TestSettle_UndeclaredReasonOfAZeroPart verifies that a part of 0 needs no
// declared reason: with LoanRepay undeclared an interest-only settlement
// works, and a principal part is a reason mismatch.
//
// Requirement: FR-LED-063, FR-CRD-005
func TestSettle_UndeclaredReasonOfAZeroPart(t *testing.T) {
	t.Parallel()
	l, err := ledger.New(ledger.Config{
		Regions: 1, MaxMoney: 1000, MaxQuantity: 10, MaxClaims: 2,
		Declared: []ledger.Reason{ledger.LoanOut, ledger.LoanInterest},
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
	if err := l.Settle(id, 0, 1); !errors.Is(err, ledger.ErrReasonMismatch) {
		t.Errorf("Settle(principal) with LoanRepay undeclared = %v, want ErrReasonMismatch", err)
	}
	if err := l.Settle(id, 5, 0); err != nil {
		t.Errorf("Settle(interest only) = %v, want nil", err)
	}
}

// TestSettle_FlowTotalOverflow verifies that a settlement whose LoanRepay or
// LoanInterest flow total, in the debtor's region, would pass int64 is refused
// with ErrBalanceOverflow and changes nothing.
//
// Requirement: FR-LED-006, FR-LED-063
func TestSettle_FlowTotalOverflow(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name             string
		prefill          ledger.Reason
		interest, repaid int64
	}{
		{"principal flow", ledger.LoanRepay, 0, 50},
		{"interest flow", ledger.LoanInterest, 7, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := newFix(t, math.MaxInt64)
			other, err := f.l.Open(ledger.KindAgent, 9, 1)
			if err != nil {
				t.Fatalf("Open: %v", err)
			}
			// The prefilled reason's total in region 1 is left 10 (principal case)
			// or 3 (interest case) below MaxInt64: x, then a smaller amount back (the
			// Initial boundary cannot take x of MaxInt64 - 10 on top of its -100).
			x := int64(math.MaxInt64 - 200)
			back := int64(190)
			if tt.prefill == ledger.LoanInterest {
				back = 197
			}
			for _, step := range []struct {
				from, to int
				amount   int64
				reason   ledger.Reason
			}{
				{f.mint, f.a1, x, ledger.Initial},
				{f.a1, other, x, tt.prefill},
				{other, f.a1, back, tt.prefill},
			} {
				if err := f.l.Transfer(step.from, step.to, step.amount, step.reason); err != nil {
					t.Fatalf("setup Transfer: %v", err)
				}
			}
			// The debtor a1 is in region 1 and holds 57 after the loan.
			id, err := f.l.Lend(f.a0, f.a1, 57, 7, 0, 5)
			if err != nil {
				t.Fatalf("Lend: %v", err)
			}
			before := snapshot(f.l)
			if err := f.l.Settle(id, tt.interest, tt.repaid); !errors.Is(err, ledger.ErrBalanceOverflow) {
				t.Fatalf("Settle = %v, want ErrBalanceOverflow", err)
			}
			if after := snapshot(f.l); !reflect.DeepEqual(before, after) {
				t.Errorf("a refused Settle changed the ledger")
			}
		})
	}
}

// TestBalanceSheet_ClaimsOfOtherAccounts verifies that an account outside a
// claim has nothing held or owed on it, whatever its ID is relative to the
// parties', and that a price table longer than the commodities is accepted.
//
// Requirement: FR-LED-026
func TestBalanceSheet_ClaimsOfOtherAccounts(t *testing.T) {
	t.Parallel()
	c := newCfix(t)
	// Claim 0: lender (2) to borrower (5). Claim 1: borrower (5) to third (6).
	c.lend(t)
	if _, err := c.l.Lend(c.borrower, c.third, 10, 0, 0, 3); err != nil {
		t.Fatalf("setup: Lend: %v", err)
	}
	above, err := c.l.Open(ledger.KindAgent, 22, 0)
	if err != nil {
		t.Fatalf("setup: Open: %v", err)
	}
	for _, id := range []int{c.a1, c.e0, above} {
		// a1 and e0 are above the creditor of claim 0 and below the parties of
		// claim 1; above is above every party.
		got, err := c.l.BalanceSheet(id, []int64{1, 1, 1})
		if want := (ledger.Sheet{}); err != nil || got != want {
			t.Errorf("BalanceSheet(%d) = %+v, %v, want %+v, nil", id, got, err, want)
		}
	}
}

// TestBalanceSheet_ClaimsOverflow verifies that a sum of claims held, or owed,
// that passes int64 is refused and not wrapped.
//
// Requirement: FR-LED-026
func TestBalanceSheet_ClaimsOverflow(t *testing.T) {
	t.Parallel()
	c := newCfix(t)
	// The borrower holds a payout claim of MaxInt64 on the lender and a loan of
	// 5 on the lender: the borrower's claims held and the lender's claims owed
	// are each MaxInt64 + 5, which passes int64.
	c.lend(t) // gives the borrower 60 to lend
	if _, err := c.l.Obligate(c.borrower, c.lender, math.MaxInt64, 5); err != nil {
		t.Fatalf("setup: Obligate: %v", err)
	}
	if _, err := c.l.Lend(c.borrower, c.lender, 5, 0, 0, 5); err != nil {
		t.Fatalf("setup: Lend: %v", err)
	}
	for _, tt := range []struct {
		name string
		id   int
	}{{"held", c.borrower}, {"owed", c.lender}} {
		if _, err := c.l.BalanceSheet(tt.id, []int64{0, 0}); !errors.Is(err, ledger.ErrBalanceOverflow) {
			t.Errorf("BalanceSheet(%s) = %v, want ErrBalanceOverflow", tt.name, err)
		}
	}
}
