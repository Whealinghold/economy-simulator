package economysim

import (
	"errors"
	"strings"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/ledger"
)

// Equivalent mutants of estates.go that no test can kill (gomu, one-line why each):
//   - regionOf, firstPrice, grant, acceptMoney, acceptUnits, receive
//     `return 0, false` / `return 0, nil` / `return 0, 0, err` -> the same zero
//     literal (lines 53, 72, 221, 224, 260, 263, 303).
//   - reconcile `i < len(rows) && j < len(lots)` -> `!=` (133) and
//     `j < len(lots)` -> `!=` (150): the indexes start at 0 and stop at the
//     length, never past it, so `<` and `!=` are the same test.
//   - reconcile `c < 0 -> <=` (140): the c == 0 case is the branch before it.
//   - openEstates `Kind != KindBoundary -> <` (166): KindBoundary is the largest
//     Kind (Open refuses kind >= KindBoundary), so != and < agree.
//   - receive `money > 0 -> !=` (301): an account balance is never negative.
//   - restore `i > 0 -> !=` (414): i is a range index, never negative.
//   - NOT_VIABLE mutants do not compile, so they are outside the score.

// TestFirstPrice_NegativeDurabilityIsNotNonDurable verifies that a durability
// below zero is divided by, not treated as "does not wear".
//
// Arithmetic: 100 * 5 / -10 = -50, raised to the floor 4.
//
// Guard: the wear test is != 0, so > 0 would return the reference price 100.
//
// Requirement: FR-EST-008
func TestFirstPrice_NegativeDurabilityIsNotNonDurable(t *testing.T) {
	t.Parallel()
	got, ok := firstPrice(100, -10, 5, 4)
	if !ok || got != 4 {
		t.Errorf("firstPrice(100, -10, 5, 4) = %d, %v, want 4, true", got, ok)
	}
}

// TestSeedPool_NegativeAmountIsRefused verifies that only an amount of exactly 0
// is skipped: a negative seed reaches the ledger, which refuses it.
//
// Guard: the skip is == 0, so <= 0 would silently drop a negative seed.
//
// Requirement: FR-EST-015
func TestSeedPool_NegativeAmountIsRefused(t *testing.T) {
	t.Parallel()
	f := newPoolFixture(t)
	err := f.e.seedPool(f.l, 0, -1)
	if err == nil || !strings.Contains(err.Error(), "region 0") {
		t.Errorf("seedPool(0, -1) = %v, want an error naming region 0", err)
	}
}

// TestReceive_FailuresSurface verifies that receive reports an unreadable
// account as its own invariant, and returns the overflow of a lot it cannot
// move.
//
// Arithmetic: the ledger holds at most 100 units of a commodity; the estate
// holds 60 and the agent 60 of the same lot, so 60 + 60 = 120 overflows.
//
// Guard: the first case needs the message of receive (acceptGoods would panic
// too); the second needs the error of acceptGoods passed up.
//
// Requirement: FR-EST-001
func TestReceive_FailuresSurface(t *testing.T) {
	t.Parallel()
	t.Run("an unreadable account names the removed agent", func(t *testing.T) {
		t.Parallel()
		f := newPoolFixture(t)
		defer func() {
			msg, _ := recover().(string)
			if !strings.Contains(msg, "account of a removed agent") {
				t.Errorf("panic = %q, want the removed-agent invariant", msg)
			}
		}()
		_, _, _ = f.e.receive(f.l, 1<<20, 0)
	})
	t.Run("a lot that overflows is returned", func(t *testing.T) {
		t.Parallel()
		l, err := ledger.New(ledger.Config{Commodities: []ledger.Commodity{{}}, Regions: 1, MaxMoney: 100, MaxQuantity: 100})
		if err != nil {
			t.Fatal(err)
		}
		src, err := l.OpenBoundary(ledger.Initial, 0)
		if err != nil {
			t.Fatal(err)
		}
		e, err := openEstates(l, 1)
		if err != nil {
			t.Fatal(err)
		}
		agent, err := l.Open(ledger.KindAgent, 1, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, to := range []int{e.account(0), agent} {
			if err := l.MoveStated(src, to, ledger.Key{}, 60, ledger.Initial); err != nil {
				t.Fatal(err)
			}
		}
		if _, _, err := e.receive(l, agent, 0); !errors.Is(err, ledger.ErrBalanceOverflow) {
			t.Errorf("receive error = %v, want ErrBalanceOverflow", err)
		}
	})
}

// TestFacts_OnlyAskIsAllowed verifies that a side other than Ask, including a
// value beyond Ask, is not allowed.
//
// Guard: the test is == Ask, so >= Ask would admit an invalid side.
//
// Requirement: FR-EST-006
func TestFacts_OnlyAskIsAllowed(t *testing.T) {
	t.Parallel()
	e := &estates{accounts: []int{7}}
	tests := []struct {
		name string
		side Side
		want bool
	}{
		{"ask", Ask, true},
		{"bid", Bid, false},
		{"beyond ask", Ask + 1, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := e.facts(0, tc.side, nil).SideAllowed; got != tc.want {
				t.Errorf("facts(side %d).SideAllowed = %v, want %v", tc.side, got, tc.want)
			}
		})
	}
}

// TestRestore_LowestPriceIsAccepted verifies that a row priced at exactly 1 is
// restored.
//
// Guard: the lower bound is price < 1, so <= 1 would refuse the smallest price.
//
// Requirement: FR-CHK-023
func TestRestore_LowestPriceIsAccepted(t *testing.T) {
	t.Parallel()
	f := newListFixture(t, 1)
	f.give(t, 0, lstStaple, 0, 3)
	if err := f.e.restore(f.l, 0, []listing{{key: k(lstStaple, 0, 0), price: 1}}); err != nil {
		t.Errorf("restore of price 1 = %v, want nil", err)
	}
	if got := f.prices()[k(lstStaple, 0, 0)]; got != 1 {
		t.Errorf("restored price = %d, want 1", got)
	}
}

// TestEstates_SmallestAmountsMove verifies that the smallest amount of 1 is
// granted, accepted as money and accepted as units, and is not mistaken for
// "nothing".
//
// Arithmetic: a pool of 5 asked for 1 gives 1 and keeps 4; 1 minor unit of money
// and 1 unit of staple move whole.
//
// Guard: the three guards are < 1, so <= 1 would drop an amount of exactly 1.
//
// Requirement: FR-EST-014, FR-EST-016, FR-EST-022
func TestEstates_SmallestAmountsMove(t *testing.T) {
	t.Parallel()
	f := newPoolFixture(t)
	f.fund(t, f.e.account(0), 5)
	if got, err := f.e.grant(f.l, 0, f.a, 1, ledger.StartMoney); err != nil || got != 1 {
		t.Errorf("grant(1) = %d, %v, want 1, nil", got, err)
	}
	if got := f.e.pool(f.l, 0); got != 4 {
		t.Errorf("pool after granting 1 = %d, want 4", got)
	}
	f.fund(t, f.b, 1)
	if err := f.e.acceptMoney(f.l, f.b, 0, 1, ledger.EstateMoney); err != nil {
		t.Fatalf("acceptMoney(1) = %v, want nil", err)
	}
	if got := f.e.pool(f.l, 0); got != 5 {
		t.Errorf("pool after accepting 1 = %d, want 5", got)
	}
	f.stock(t, f.a, poolStaple, 1)
	if got, err := f.e.acceptUnits(f.l, f.a, 0, poolStaple, 1, ledger.EstateGoods); err != nil || got != 1 {
		t.Errorf("acceptUnits(1) = %d, %v, want 1, nil", got, err)
	}
	if lots := f.lots(t, f.e.account(0)); len(lots) != 1 || lots[0].Quantity != 1 {
		t.Errorf("estate lots = %v, want one lot of 1", lots)
	}
}
