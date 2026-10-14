package ledger_test

import (
	"errors"
	"math"
	"reflect"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/ledger"
)

// TestTransfer_Refusals verifies that every refusal of Transfer is its own
// sentinel, that each case breaks only the one rule, and that a refused
// transfer leaves the ledger exactly as it was and the total at zero.
//
// Requirement: FR-LED-002, FR-LED-004, FR-LED-005, FR-LED-006, FR-LED-007, FR-LED-017, FR-LED-019
func TestTransfer_Refusals(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		// big builds the fixture with the largest money bound, to reach the
		// edge of int64.
		big   bool
		setup func(f *fix, t *testing.T)
		do    func(f *fix) error
		want  error
	}{
		{name: "amount zero", do: func(f *fix) error { return f.l.Transfer(f.a0, f.a1, 0, ledger.Trade) }, want: ledger.ErrInvalidAmount},
		{name: "amount negative", do: func(f *fix) error { return f.l.Transfer(f.a0, f.a1, -5, ledger.Trade) }, want: ledger.ErrInvalidAmount},
		{name: "unknown debited account", do: func(f *fix) error { return f.l.Transfer(99, f.a1, 1, ledger.Trade) }, want: ledger.ErrUnknownAccount},
		{name: "negative debited account", do: func(f *fix) error { return f.l.Transfer(-1, f.a1, 1, ledger.Trade) }, want: ledger.ErrUnknownAccount},
		{name: "unknown credited account", do: func(f *fix) error { return f.l.Transfer(f.a0, 99, 1, ledger.Trade) }, want: ledger.ErrUnknownAccount},
		{
			name: "closed debited account",
			setup: func(f *fix, t *testing.T) {
				if err := f.l.Close(f.a1); err != nil {
					t.Fatalf("Close: %v", err)
				}
			},
			do:   func(f *fix) error { return f.l.Transfer(f.a1, f.a0, 1, ledger.Trade) },
			want: ledger.ErrClosedAccount,
		},
		{
			name: "closed credited account",
			setup: func(f *fix, t *testing.T) {
				if err := f.l.Close(f.a1); err != nil {
					t.Fatalf("Close: %v", err)
				}
			},
			do:   func(f *fix) error { return f.l.Transfer(f.a0, f.a1, 1, ledger.Trade) },
			want: ledger.ErrClosedAccount,
		},
		{name: "reason out of range", do: func(f *fix) error { return f.l.Transfer(f.a0, f.a1, 1, ledger.NumReasons) }, want: ledger.ErrReasonMismatch},
		{name: "goods-only reason moving money", do: func(f *fix) error { return f.l.Transfer(f.a0, f.a1, 1, ledger.Consumption) }, want: ledger.ErrReasonMismatch},
		{name: "trade to a boundary", do: func(f *fix) error { return f.l.Transfer(f.a0, f.mint, 1, ledger.Trade) }, want: ledger.ErrReasonMismatch},
		{name: "faucet between agents", do: func(f *fix) error { return f.l.Transfer(f.a0, f.a1, 1, ledger.Initial) }, want: ledger.ErrReasonMismatch},
		{name: "start money from an agent", do: func(f *fix) error { return f.l.Transfer(f.a0, f.e0, 1, ledger.StartMoney) }, want: ledger.ErrReasonMismatch},
		{name: "births to an estate", do: func(f *fix) error { return f.l.Transfer(f.a0, f.e0, 1, ledger.Births) }, want: ledger.ErrReasonMismatch},
		{name: "same account", do: func(f *fix) error { return f.l.Transfer(f.a0, f.a0, 1, ledger.Trade) }, want: ledger.ErrReasonMismatch},
		{name: "boundary of goods under a money transfer", do: func(f *fix) error { return f.l.Transfer(f.goodsBoundary, f.a1, 1, ledger.Initial) }, want: ledger.ErrReasonMismatch},
		{name: "insufficient funds", do: func(f *fix) error { return f.l.Transfer(f.a0, f.a1, 101, ledger.Trade) }, want: ledger.ErrInsufficientFunds},
		{
			name: "earmarked funds cannot be spent unmarked",
			setup: func(f *fix, t *testing.T) {
				if err := f.l.ReserveMoney(f.a0, 60); err != nil {
					t.Fatalf("ReserveMoney: %v", err)
				}
			},
			// 100 held, 60 reserved: 40 available.
			do:   func(f *fix) error { return f.l.Transfer(f.a0, f.a1, 41, ledger.Trade) },
			want: ledger.ErrInsufficientFunds,
		},
		{
			name: "credit above MaxMoney",
			setup: func(f *fix, t *testing.T) {
				if err := f.l.Transfer(f.mint, f.a1, testMaxMoney, ledger.Initial); err != nil {
					t.Fatalf("fund: %v", err)
				}
			},
			// a1 holds exactly MaxMoney; one more is one above.
			do:   func(f *fix) error { return f.l.Transfer(f.a0, f.a1, 1, ledger.Trade) },
			want: ledger.ErrBalanceOverflow,
		},
		{name: "faucet of one above MaxMoney", do: func(f *fix) error { return f.l.Transfer(f.mint, f.e0, testMaxMoney+1, ledger.Initial) }, want: ledger.ErrBalanceOverflow},
		{
			name: "boundary at the edge of int64",
			big:  true,
			setup: func(f *fix, t *testing.T) {
				// The boundary holds -100; a faucet of MaxInt64 - 100 takes it
				// to exactly MinInt64 + 1, and one more unit is the last that
				// fits.
				if err := f.l.Transfer(f.mint, f.a1, math.MaxInt64-100, ledger.Initial); err != nil {
					t.Fatalf("fund: %v", err)
				}
			},
			// -MaxInt64 - 2 does not fit in int64.
			do:   func(f *fix) error { return f.l.Transfer(f.mint, f.e0, 2, ledger.Initial) },
			want: ledger.ErrBalanceOverflow,
		},
		{
			name: "marked transfer with a reservation too small",
			setup: func(f *fix, t *testing.T) {
				if err := f.l.ReserveMoney(f.a0, 10); err != nil {
					t.Fatalf("ReserveMoney: %v", err)
				}
			},
			do:   func(f *fix) error { return f.l.TransferReserved(f.a0, f.a1, 11, ledger.Trade) },
			want: ledger.ErrReservationExceeds,
		},
		{name: "marked transfer from a boundary", do: func(f *fix) error { return f.l.TransferReserved(f.mint, f.a1, 1, ledger.Initial) }, want: ledger.ErrBoundaryReserve},
		{name: "marked transfer with no reservation", do: func(f *fix) error { return f.l.TransferReserved(f.a0, f.a1, 1, ledger.Trade) }, want: ledger.ErrReservationExceeds},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			maxMoney := int64(testMaxMoney)
			if tt.big {
				maxMoney = math.MaxInt64
			}
			f := newFix(t, maxMoney)
			if tt.setup != nil {
				tt.setup(f, t)
			}
			before := snapshot(f.l)
			err := tt.do(f)
			if !errors.Is(err, tt.want) {
				t.Fatalf("Transfer: got %v, want %v", err, tt.want)
			}
			if after := snapshot(f.l); !reflect.DeepEqual(before, after) {
				t.Errorf("refused transfer changed the ledger:\n before %+v\n after  %+v", before, after)
			}
			if got := sum(f.l); got != 0 {
				t.Errorf("total of money = %d, want 0", got)
			}
		})
	}
}

// TestTransfer_FlowTotalOverflow verifies that a transfer whose flow total
// would pass int64 is refused although both balances fit, so that a total is
// never wrapped and never silently wrong.
//
// Requirement: FR-LED-006, FR-LED-007
func TestTransfer_FlowTotalOverflow(t *testing.T) {
	t.Parallel()
	f := newFix(t, math.MaxInt64)
	// x is below MaxInt64, so one trade of x fits and two do not add up.
	const x = math.MaxInt64 - 100
	other, err := f.l.Open(ledger.KindAgent, 9, 1)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	for _, step := range []struct {
		from, to int
		amount   int64
		reason   ledger.Reason
	}{
		{f.mint, f.a1, x, ledger.Initial},
		{f.a1, other, x, ledger.Trade},
	} {
		if err := f.l.Transfer(step.from, step.to, step.amount, step.reason); err != nil {
			t.Fatalf("setup Transfer: %v", err)
		}
	}
	before := snapshot(f.l)
	// Region 1's trade total is x; paying x back from region 1's account
	// makes 2x, which is above MaxInt64.
	err = f.l.Transfer(other, f.a1, x, ledger.Trade)
	if !errors.Is(err, ledger.ErrBalanceOverflow) {
		t.Fatalf("Transfer: got %v, want ErrBalanceOverflow", err)
	}
	if after := snapshot(f.l); !reflect.DeepEqual(before, after) {
		t.Errorf("refused transfer changed the ledger")
	}
}

// TestTransfer_Applies verifies the effect of an accepted transfer: both
// balances, the total of zero, the exact boundary values (one unit, exactly
// the available balance, exactly MaxMoney, the edge of int64) and the
// reservation drawn down by the marked form.
//
// Requirement: FR-LED-002, FR-LED-003, FR-LED-005, FR-LED-006, FR-LED-019
func TestTransfer_Applies(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		big   bool
		setup func(f *fix, t *testing.T)
		do    func(f *fix) error
		// want is the balances of a0, a1, e0 and the mint, and the
		// reservation of a0 afterwards.
		a0, a1, e0, mint, reserved int64
	}{
		{
			name: "one unit",
			do:   func(f *fix) error { return f.l.Transfer(f.a0, f.a1, 1, ledger.Trade) },
			// 100 - 1, 0 + 1.
			a0: 99, a1: 1, e0: 0, mint: -100,
		},
		{
			name: "exactly the available balance",
			do:   func(f *fix) error { return f.l.Transfer(f.a0, f.a1, 100, ledger.Trade) },
			a0:   0, a1: 100, e0: 0, mint: -100,
		},
		{
			name: "exactly MaxMoney",
			do:   func(f *fix) error { return f.l.Transfer(f.mint, f.e0, testMaxMoney, ledger.Initial) },
			// The mint was at -100 and goes to -100 - 1000.
			a0: 100, a1: 0, e0: testMaxMoney, mint: -100 - testMaxMoney,
		},
		{
			name: "boundary to the bottom of int64",
			big:  true,
			do: func(f *fix) error {
				// -100 - (MaxInt64 - 99) = MinInt64: the last value that fits.
				// a1 is in region 1, so the region 0 faucet total of the
				// setup (100) is not added to it.
				return f.l.Transfer(f.mint, f.a1, math.MaxInt64-99, ledger.Initial)
			},
			a0: 100, a1: math.MaxInt64 - 99, e0: 0, mint: math.MinInt64,
		},
		{
			name: "estate to agent",
			setup: func(f *fix, t *testing.T) {
				if err := f.l.Transfer(f.a0, f.e0, 30, ledger.EstateMoney); err != nil {
					t.Fatalf("setup: %v", err)
				}
			},
			do: func(f *fix) error { return f.l.Transfer(f.e0, f.a1, 30, ledger.StartMoney) },
			a0: 70, a1: 30, e0: 0, mint: -100,
		},
		{
			name: "marked transfer draws the reservation down",
			setup: func(f *fix, t *testing.T) {
				if err := f.l.ReserveMoney(f.a0, 50); err != nil {
					t.Fatalf("setup: %v", err)
				}
			},
			// 50 of 100 reserved; a marked 30 leaves 70 held and 20 reserved.
			do: func(f *fix) error { return f.l.TransferReserved(f.a0, f.a1, 30, ledger.Trade) },
			a0: 70, a1: 30, e0: 0, mint: -100, reserved: 20,
		},
		{
			name: "unmarked transfer of the unreserved part",
			setup: func(f *fix, t *testing.T) {
				if err := f.l.ReserveMoney(f.a0, 50); err != nil {
					t.Fatalf("setup: %v", err)
				}
			},
			do: func(f *fix) error { return f.l.Transfer(f.a0, f.a1, 50, ledger.Trade) },
			a0: 50, a1: 50, e0: 0, mint: -100, reserved: 50,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			maxMoney := int64(testMaxMoney)
			if tt.big {
				maxMoney = math.MaxInt64
			}
			f := newFix(t, maxMoney)
			if tt.setup != nil {
				tt.setup(f, t)
			}
			if err := tt.do(f); err != nil {
				t.Fatalf("Transfer: %v", err)
			}
			for _, c := range []struct {
				name string
				id   int
				want int64
			}{{"a0", f.a0, tt.a0}, {"a1", f.a1, tt.a1}, {"e0", f.e0, tt.e0}, {"mint", f.mint, tt.mint}} {
				v, err := f.l.Account(c.id)
				if err != nil {
					t.Fatalf("Account(%s): %v", c.name, err)
				}
				if v.Balance != c.want {
					t.Errorf("balance of %s = %d, want %d", c.name, v.Balance, c.want)
				}
			}
			if v, _ := f.l.Account(f.a0); v.Reserved != tt.reserved {
				t.Errorf("reserved of a0 = %d, want %d", v.Reserved, tt.reserved)
			}
			// With the edge of int64 in play the sum itself would wrap, so
			// compare the unsigned sum, which wraps to the same bits.
			if got := uint64(sum(f.l)); got != 0 {
				t.Errorf("total of money = %d, want 0", int64(got))
			}
		})
	}
}

// TestReserveMoney verifies reserving and releasing money: the amount
// available, the exact edges of the available balance and of the
// reservation, and that a boundary can neither reserve nor release.
//
// Requirement: FR-LED-017, FR-LED-018, FR-LED-007
func TestReserveMoney(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		setup func(f *fix, t *testing.T)
		do    func(f *fix) error
		want  error
		// reserved is a0's reservation afterwards.
		reserved int64
	}{
		{name: "reserve one", do: func(f *fix) error { return f.l.ReserveMoney(f.a0, 1) }, reserved: 1},
		{name: "reserve the whole balance", do: func(f *fix) error { return f.l.ReserveMoney(f.a0, 100) }, reserved: 100},
		{name: "reserve one above the balance", do: func(f *fix) error { return f.l.ReserveMoney(f.a0, 101) }, want: ledger.ErrInsufficientFunds},
		{name: "reserve zero", do: func(f *fix) error { return f.l.ReserveMoney(f.a0, 0) }, want: ledger.ErrInvalidAmount},
		{name: "reserve on a boundary", do: func(f *fix) error { return f.l.ReserveMoney(f.mint, 1) }, want: ledger.ErrBoundaryReserve},
		{name: "reserve on an unknown account", do: func(f *fix) error { return f.l.ReserveMoney(99, 1) }, want: ledger.ErrUnknownAccount},
		{
			name: "reserve above the unreserved part",
			setup: func(f *fix, t *testing.T) {
				if err := f.l.ReserveMoney(f.a0, 70); err != nil {
					t.Fatalf("setup: %v", err)
				}
			},
			// 100 held, 70 reserved: 30 available, 31 asked.
			do:       func(f *fix) error { return f.l.ReserveMoney(f.a0, 31) },
			want:     ledger.ErrInsufficientFunds,
			reserved: 70,
		},
		{
			name: "release part",
			setup: func(f *fix, t *testing.T) {
				if err := f.l.ReserveMoney(f.a0, 70); err != nil {
					t.Fatalf("setup: %v", err)
				}
			},
			do:       func(f *fix) error { return f.l.ReleaseMoney(f.a0, 20) },
			reserved: 50,
		},
		{
			name: "release exactly the reservation",
			setup: func(f *fix, t *testing.T) {
				if err := f.l.ReserveMoney(f.a0, 70); err != nil {
					t.Fatalf("setup: %v", err)
				}
			},
			do: func(f *fix) error { return f.l.ReleaseMoney(f.a0, 70) },
		},
		{
			name: "release one above the reservation",
			setup: func(f *fix, t *testing.T) {
				if err := f.l.ReserveMoney(f.a0, 70); err != nil {
					t.Fatalf("setup: %v", err)
				}
			},
			do:       func(f *fix) error { return f.l.ReleaseMoney(f.a0, 71) },
			want:     ledger.ErrReservationExceeds,
			reserved: 70,
		},
		{name: "release with nothing reserved", do: func(f *fix) error { return f.l.ReleaseMoney(f.a0, 1) }, want: ledger.ErrReservationExceeds},
		{name: "release zero", do: func(f *fix) error { return f.l.ReleaseMoney(f.a0, 0) }, want: ledger.ErrInvalidAmount},
		{name: "release on a boundary", do: func(f *fix) error { return f.l.ReleaseMoney(f.mint, 1) }, want: ledger.ErrReservationExceeds},
		{name: "release on an unknown account", do: func(f *fix) error { return f.l.ReleaseMoney(-3, 1) }, want: ledger.ErrUnknownAccount},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := newFix(t, testMaxMoney)
			if tt.setup != nil {
				tt.setup(f, t)
			}
			before := snapshot(f.l)
			err := tt.do(f)
			if !errors.Is(err, tt.want) {
				t.Fatalf("got %v, want %v", err, tt.want)
			}
			v, _ := f.l.Account(f.a0)
			if v.Reserved != tt.reserved {
				t.Errorf("reserved = %d, want %d", v.Reserved, tt.reserved)
			}
			if v.Balance != 100 {
				t.Errorf("balance = %d, want 100 (a reservation moves no money)", v.Balance)
			}
			if tt.want != nil {
				if after := snapshot(f.l); !reflect.DeepEqual(before, after) {
					t.Errorf("refused call changed the ledger")
				}
			}
		})
	}
}
