package ledger_test

import (
	"errors"
	"math"
	"reflect"
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/ledger"
)

// goodsFix is the fixture of the relocation tests: newFix after construction
// has ended, with a0 (region 0) holding 100 money, 5 units of commodity 0 and 4
// of commodity 1, so a relocation has all three things to total.
func goodsFix(t *testing.T) *fix {
	t.Helper()
	f := newFix(t, testMaxMoney)
	b1, err := f.l.OpenBoundary(ledger.Initial, 1)
	if err != nil {
		t.Fatalf("setup: OpenBoundary: %v", err)
	}
	if err := f.l.Move(f.goodsBoundary, f.a0, 0, 5, ledger.Initial); err != nil {
		t.Fatalf("setup: goods 0: %v", err)
	}
	if err := f.l.Move(b1, f.a0, 1, 4, ledger.Initial); err != nil {
		t.Fatalf("setup: goods 1: %v", err)
	}
	if err := f.l.EndConstruction(); err != nil {
		t.Fatalf("setup: EndConstruction: %v", err)
	}
	return f
}

// TestRelocate_OnlyRegionChanges verifies that a relocation changes the region
// of the account and nothing else: no balance, lot, reservation, flow total or
// construction bucket, the money and the goods of the ledger as a whole, and the
// audit are all as before, and the relocation totals hold the balance and the
// units of each commodity under (from, to).
//
// Requirement: FR-REG-040, FR-REG-042, FR-LED-054
func TestRelocate_OnlyRegionChanges(t *testing.T) {
	t.Parallel()
	f := goodsFix(t)
	if err := f.l.Transfer(f.a0, f.a1, 10, ledger.Trade); err != nil {
		t.Fatalf("setup: Transfer: %v", err)
	}
	before := snapshot(f.l)
	lotsBefore, err := f.l.AppendLots(nil, f.a0)
	if err != nil {
		t.Fatalf("AppendLots: %v", err)
	}
	moneyBefore := sum(f.l)

	if err := f.l.Relocate(f.a0, 1); err != nil {
		t.Fatalf("Relocate(a0, 1) = %v, want nil", err)
	}

	after := snapshot(f.l)
	// a0 held 100 and paid 10: it holds 90, and only its region differs.
	before.Accounts[f.a0].Region = 1
	if !reflect.DeepEqual(after, before) {
		t.Errorf("state after Relocate = %+v, want the state before with a0 in region 1: %+v", after, before)
	}
	lotsAfter, err := f.l.AppendLots(nil, f.a0)
	if err != nil || !slices.Equal(lotsAfter, lotsBefore) {
		t.Errorf("AppendLots after = %v, %v, want %v", lotsAfter, err, lotsBefore)
	}
	if got := sum(f.l); got != moneyBefore {
		t.Errorf("money over all accounts = %d, want %d", got, moneyBefore)
	}
	if err := f.l.Audit(); err != nil {
		t.Errorf("Audit() = %v, want nil", err)
	}
	want := []ledger.Moved{
		{From: 0, To: 1, Commodity: ledger.Money, Total: 90},
		{From: 0, To: 1, Commodity: 0, Total: 5},
		{From: 0, To: 1, Commodity: 1, Total: 4},
	}
	if got := f.l.AppendRelocations(nil); !slices.Equal(got, want) {
		t.Errorf("AppendRelocations = %v, want %v", got, want)
	}
	if got, err := f.l.Units(f.a0); err != nil || got != 9 {
		t.Errorf("Units(a0) = %d, %v, want 9 (5 + 4)", got, err)
	}
}

// TestRelocate_Refusals verifies that every refusal of Relocate is its own
// sentinel, in the documented order (an earlier failing rule hides a later
// one), that CheckRelocate returns the same error, and that a refused call
// leaves the ledger, the totals included, exactly as it was.
//
// Requirement: FR-REG-039, FR-REG-043, FR-REG-044
func TestRelocate_Refusals(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		setup func(f *fix, t *testing.T)
		id    func(f *fix) int
		to    int
		want  error
	}{
		{name: "unknown account", id: func(*fix) int { return 99 }, to: 1, want: ledger.ErrUnknownAccount},
		{name: "negative account", id: func(*fix) int { return -1 }, to: 1, want: ledger.ErrUnknownAccount},
		{
			name: "closed account",
			setup: func(f *fix, t *testing.T) {
				if err := f.l.Close(f.a1); err != nil {
					t.Fatalf("setup: Close: %v", err)
				}
			},
			id: func(f *fix) int { return f.a1 }, to: 0, want: ledger.ErrClosedAccount,
		},
		{name: "estate account", id: func(f *fix) int { return f.e0 }, to: 1, want: ledger.ErrNotRelocatable},
		{name: "boundary account", id: func(f *fix) int { return f.mint }, to: 1, want: ledger.ErrNotRelocatable},
		// The kind is checked before the region: an estate moved to a region that
		// does not exist is not relocatable first.
		{name: "estate to an unknown region", id: func(f *fix) int { return f.e0 }, to: 7, want: ledger.ErrNotRelocatable},
		{name: "region past the last", id: func(f *fix) int { return f.a0 }, to: 2, want: ledger.ErrUnknownRegion},
		{name: "region far past the last", id: func(f *fix) int { return f.a0 }, to: 99, want: ledger.ErrUnknownRegion},
		{name: "no region", id: func(f *fix) int { return f.a0 }, to: -1, want: ledger.ErrUnknownRegion},
		{name: "same region", id: func(f *fix) int { return f.a0 }, to: 0, want: ledger.ErrSameRegion},
		{
			name: "reserved money",
			setup: func(f *fix, t *testing.T) {
				if err := f.l.ReserveMoney(f.a0, 5); err != nil {
					t.Fatalf("setup: ReserveMoney: %v", err)
				}
			},
			id: func(f *fix) int { return f.a0 }, to: 1, want: ledger.ErrOpenReservation,
		},
		{
			name: "earmarked goods",
			setup: func(f *fix, t *testing.T) {
				if err := f.l.ReserveGoods(f.a0, 0, 2); err != nil {
					t.Fatalf("setup: ReserveGoods: %v", err)
				}
			},
			id: func(f *fix) int { return f.a0 }, to: 1, want: ledger.ErrOpenReservation,
		},
		{
			// The region is checked before the reservation, and the same region before it.
			name: "reservation hidden by the same region",
			setup: func(f *fix, t *testing.T) {
				if err := f.l.ReserveMoney(f.a0, 5); err != nil {
					t.Fatalf("setup: ReserveMoney: %v", err)
				}
			},
			id: func(f *fix) int { return f.a0 }, to: 0, want: ledger.ErrSameRegion,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := goodsFix(t)
			if tt.setup != nil {
				tt.setup(f, t)
			}
			before := snapshot(f.l)
			movedBefore := f.l.AppendRelocations(nil)
			id := tt.id(f)
			if err := f.l.CheckRelocate(id, tt.to); !errors.Is(err, tt.want) {
				t.Errorf("CheckRelocate(%d, %d) = %v, want %v", id, tt.to, err, tt.want)
			}
			if err := f.l.Relocate(id, tt.to); !errors.Is(err, tt.want) {
				t.Errorf("Relocate(%d, %d) = %v, want %v", id, tt.to, err, tt.want)
			}
			if got := snapshot(f.l); !reflect.DeepEqual(got, before) {
				t.Errorf("state after the refused Relocate = %+v, want %+v", got, before)
			}
			if got := f.l.AppendRelocations(nil); !slices.Equal(got, movedBefore) {
				t.Errorf("relocation totals after the refusal = %v, want %v", got, movedBefore)
			}
		})
	}
}

// TestRelocate_TotalOverflowIsRefused verifies that a relocation whose money
// would push a row of the totals past int64 is refused with ErrBalanceOverflow
// and changes nothing, instead of wrapping.
//
// Guard: the totals are summed across calls in a tick, so a row can outgrow the
// bound of one balance; architecture.md 3.5 forbids a silent wrap.
func TestRelocate_TotalOverflowIsRefused(t *testing.T) {
	t.Parallel()
	l, err := ledger.New(ledger.Config{Regions: 2, MaxMoney: math.MaxInt64, MaxQuantity: 1, Commodities: []ledger.Commodity{{}}})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	mint, err := l.OpenBoundary(ledger.Initial, ledger.Money)
	if err != nil {
		t.Fatalf("OpenBoundary: %v", err)
	}
	a, err := l.Open(ledger.KindAgent, 1, 0)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	// The (0, 1) row takes half the bound plus one on the first move and the
	// same again on the third, which does not fit.
	const half = math.MaxInt64/2 + 1
	if err := l.Transfer(mint, a, half, ledger.Initial); err != nil {
		t.Fatalf("setup: Transfer: %v", err)
	}
	for _, to := range []int{1, 0} {
		if err := l.Relocate(a, to); err != nil {
			t.Fatalf("Relocate(a, %d) = %v, want nil", to, err)
		}
	}
	before := l.AppendRelocations(nil)
	if err := l.Relocate(a, 1); !errors.Is(err, ledger.ErrBalanceOverflow) {
		t.Errorf("Relocate(a, 1) = %v, want ErrBalanceOverflow", err)
	}
	if v, err := l.Account(a); err != nil || v.Region != 0 {
		t.Errorf("Account(a) = %+v, %v, want region 0 (the refused call changed nothing)", v, err)
	}
	if got := l.AppendRelocations(nil); !slices.Equal(got, before) {
		t.Errorf("rows after the refusal = %v, want %v", got, before)
	}
}

// TestRelocate_FlowsFollowTheNewRegion verifies that a transfer before a
// relocation is attributed to the old region and one after it to the new region,
// and that the same holds for the spoilage of goods the account holds.
//
// Requirement: FR-REG-041
//
// Guard: Age attributes spoilage to the region of the account (ageing.go); a
// relocation that left a cached region behind would split the figures of a
// regional report.
func TestRelocate_FlowsFollowTheNewRegion(t *testing.T) {
	t.Parallel()
	f := goodsFix(t)
	if _, err := f.l.OpenBoundary(ledger.Spoilage, 0); err != nil {
		t.Fatalf("setup: OpenBoundary(Spoilage): %v", err)
	}
	if err := f.l.Transfer(f.a0, f.a1, 10, ledger.Trade); err != nil {
		t.Fatalf("Transfer before: %v", err)
	}
	if err := f.l.Relocate(f.a0, 1); err != nil {
		t.Fatalf("Relocate: %v", err)
	}
	if err := f.l.Transfer(f.a0, f.a1, 20, ledger.Trade); err != nil {
		t.Fatalf("Transfer after: %v", err)
	}
	// Commodity 0 lasts 3 ticks: a lot born in tick 0 is gone by tick 2. The
	// ticks pass with a0 in region 1, so its 5 units spoil there.
	f.l.AdvanceTick()
	f.l.AdvanceTick()
	if err := f.l.Age(); err != nil {
		t.Fatalf("Age: %v", err)
	}
	want := []ledger.Flow{
		{Region: 0, Reason: ledger.Trade, Commodity: ledger.Money, Total: 10},
		{Region: 1, Reason: ledger.Trade, Commodity: ledger.Money, Total: 20},
		{Region: 1, Reason: ledger.Spoilage, Commodity: 0, Total: 5},
	}
	got := f.l.AppendFlows(nil, false)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("flow totals = %v, want %v", got, want)
	}
}

// TestRelocationTotals_OrderAndClear verifies that the totals add up across
// relocations of one tick, are listed in ascending from, to, then money before
// the commodities, are cleared with the flow totals at the end of the tick, and
// that a relocation after the clearing starts them again.
//
// Requirement: FR-REG-042, FR-LED-054
func TestRelocationTotals_OrderAndClear(t *testing.T) {
	t.Parallel()
	f := goodsFix(t)
	// a1 holds nothing, so only the region pair is totalled: no row of 0.
	steps := []struct {
		id, to int
	}{{f.a0, 1}, {f.a1, 0}, {f.a0, 0}, {f.a0, 1}}
	for _, s := range steps {
		if err := f.l.Relocate(s.id, s.to); err != nil {
			t.Fatalf("Relocate(%d, %d) = %v, want nil", s.id, s.to, err)
		}
	}
	// a0 moved 0 to 1 twice, and 1 to 0 once, with 100 money, 5 and 4 units.
	want := []ledger.Moved{
		{From: 0, To: 1, Commodity: ledger.Money, Total: 200},
		{From: 0, To: 1, Commodity: 0, Total: 10},
		{From: 0, To: 1, Commodity: 1, Total: 8},
		{From: 1, To: 0, Commodity: ledger.Money, Total: 100},
		{From: 1, To: 0, Commodity: 0, Total: 5},
		{From: 1, To: 0, Commodity: 1, Total: 4},
	}
	if got := f.l.AppendRelocations(nil); !slices.Equal(got, want) {
		t.Errorf("AppendRelocations = %v, want %v", got, want)
	}
	if got := f.l.AppendRelocations(want[:1]); len(got) != 7 || got[0] != want[0] {
		t.Errorf("AppendRelocations(dst) = %v, want dst kept and the six rows appended", got)
	}

	f.l.EndTick()
	if got := f.l.AppendRelocations(nil); len(got) != 0 {
		t.Errorf("AppendRelocations after EndTick = %v, want none", got)
	}
	if err := f.l.Relocate(f.a0, 0); err != nil {
		t.Fatalf("Relocate after EndTick = %v, want nil", err)
	}
	if got := f.l.AppendRelocations(nil); len(got) != 3 || got[0].Total != 100 {
		t.Errorf("AppendRelocations in the next tick = %v, want 3 rows starting at 100 money", got)
	}
}

// TestUnits_Errors verifies the unit count of an account and its errors.
//
// Requirement: FR-REG-046
func TestUnits_Errors(t *testing.T) {
	t.Parallel()
	f := goodsFix(t)
	if got, err := f.l.Units(99); !errors.Is(err, ledger.ErrUnknownAccount) || got != 0 {
		t.Errorf("Units(99) = %d, %v, want 0, ErrUnknownAccount", got, err)
	}
	if got, err := f.l.Units(f.a1); err != nil || got != 0 {
		t.Errorf("Units(a1) = %d, %v, want 0", got, err)
	}
	if err := f.l.ReserveGoods(f.a0, 0, 2); err != nil {
		t.Fatalf("setup: ReserveGoods: %v", err)
	}
	if got, err := f.l.Units(f.a0); err != nil || got != 9 {
		t.Errorf("Units(a0) = %d, %v, want 9: earmarked units are held", got, err)
	}
}

// FuzzRelocate runs a byte-driven sequence of transfers, relocations (some
// refused for a reservation), goods moves and ticks, and checks after each step
// that the audit holds, that the money total is unchanged, that the totals are
// strictly ascending with no zero row, and that Relocate returns exactly what
// CheckRelocate said.
func FuzzRelocate(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11})
	f.Add([]byte{1, 2, 1, 2, 3, 3, 4, 0, 1, 5, 2, 1, 6, 7, 1, 2})
	f.Fuzz(func(t *testing.T, data []byte) {
		fx := newFix(t, testMaxMoney)
		b1, err := fx.l.OpenBoundary(ledger.Initial, 1)
		if err != nil {
			t.Fatal(err)
		}
		if err := fx.l.EndConstruction(); err != nil {
			t.Fatal(err)
		}
		agents := []int{fx.a0, fx.a1}
		// sum counts the balance of the goods boundaries too, which are units, so
		// the money is read from the four accounts that hold or mint it.
		moneyOf := func() (total int64) {
			for _, id := range []int{fx.mint, fx.a0, fx.a1, fx.e0} {
				v, err := fx.l.Account(id)
				if err != nil {
					t.Fatal(err)
				}
				total += v.Balance
			}
			return total
		}
		money := moneyOf()
		for i := 0; i+1 < len(data); i += 2 {
			op, arg := data[i]%6, int(data[i+1])
			from, to := agents[arg%2], agents[(arg/2)%2]
			var err error
			switch op {
			case 0:
				_ = fx.l.Transfer(from, to, int64(arg%7+1), ledger.Trade)
			case 1:
				want := fx.l.CheckRelocate(from, arg/4%3)
				if got := fx.l.Relocate(from, arg/4%3); !errors.Is(got, want) && got != want {
					t.Fatalf("op %d: Relocate = %v, CheckRelocate = %v", i, got, want)
				}
			case 2:
				// A reservation blocks the move; releasing it allows the move again.
				if fx.l.ReserveMoney(from, 1) == nil {
					if err = fx.l.Relocate(from, 1-arg%2); err == nil {
						t.Fatalf("op %d: Relocate with a reservation = nil", i)
					}
					if err := fx.l.ReleaseMoney(from, 1); err != nil {
						t.Fatal(err)
					}
				}
			case 3:
				_ = fx.l.Move(fx.goodsBoundary, from, 0, int64(arg%5+1), ledger.Initial)
			case 4:
				_ = fx.l.Move(b1, from, 1, int64(arg%5+1), ledger.Initial)
			case 5:
				fx.l.EndTick()
			}
			if err := fx.l.Audit(); err != nil {
				t.Fatalf("op %d: Audit = %v", i, err)
			}
			if got := moneyOf(); got != money {
				t.Fatalf("op %d: money over all accounts = %d, want %d", i, got, money)
			}
			rows := fx.l.AppendRelocations(nil)
			for j, r := range rows {
				prev := ledger.Moved{}
				if j > 0 {
					prev = rows[j-1]
				}
				ascending := j == 0 || r.From > prev.From || r.From == prev.From && (r.To > prev.To || r.To == prev.To && r.Commodity > prev.Commodity)
				if r.Total < 1 || r.From == r.To || !ascending {
					t.Fatalf("op %d: relocation rows %v: row %d is zero, within a region or out of order", i, rows, j)
				}
			}
		}
	})
}

// TestCheckRelocate_AllocatesNothing verifies that the dry run, on a success and
// on a refusal, allocates nothing.
//
// Requirement: FR-REG-059
//
// Guard: the world's RelocationRefusal is built on it and must allocate
// nothing; a closure that escaped would break that without changing a result.
func TestCheckRelocate_AllocatesNothing(t *testing.T) {
	f := goodsFix(t)
	for _, to := range []int{1, 0} {
		if got := testing.AllocsPerRun(100, func() {
			if err := f.l.CheckRelocate(f.a0, to); to == 1 && err != nil {
				t.Fatalf("CheckRelocate(a0, 1) = %v, want nil", err)
			}
		}); got != 0 {
			t.Errorf("CheckRelocate(a0, %d) allocates %v times, want 0", to, got)
		}
	}
}

// TestUnits_OverflowIsReported verifies that a unit count past int64 is
// reported as ErrBalanceOverflow, not wrapped.
//
// Guard: each commodity may hold MaxQuantity, so the sum over commodities can
// pass int64 when the bound is large; the event of a relocation reads it.
func TestUnits_OverflowIsReported(t *testing.T) {
	t.Parallel()
	l, err := ledger.New(ledger.Config{Regions: 1, MaxMoney: 1, MaxQuantity: math.MaxInt64, Commodities: []ledger.Commodity{{}, {}}})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	a, err := l.Open(ledger.KindAgent, 1, 0)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	for c := range 2 {
		b, err := l.OpenBoundary(ledger.Initial, c)
		if err != nil {
			t.Fatalf("OpenBoundary: %v", err)
		}
		if err := l.Move(b, a, c, math.MaxInt64/2+1, ledger.Initial); err != nil {
			t.Fatalf("Move: %v", err)
		}
	}
	if got, err := l.Units(a); !errors.Is(err, ledger.ErrBalanceOverflow) || got != 0 {
		t.Errorf("Units(a) = %d, %v, want 0, ErrBalanceOverflow", got, err)
	}
}

// TestRelocationTotals_ExactAmounts verifies the amounts totalled: an account
// with no money and no goods adds no row, a balance of exactly 1 is totalled as
// 1, and the several lots of one commodity (born in different ticks) add up to
// one row.
//
// Requirement: FR-REG-042
//
// Guard: a zero row would break the reader (a total of 0 is refused), and a lot
// counted without its siblings would understate the units a region received.
func TestRelocationTotals_ExactAmounts(t *testing.T) {
	t.Parallel()
	f := goodsFix(t)
	if err := f.l.Relocate(f.a1, 0); err != nil {
		t.Fatalf("Relocate(a1, 0) = %v, want nil", err)
	}
	if got := f.l.AppendRelocations(nil); len(got) != 0 {
		t.Errorf("AppendRelocations after an empty account moved = %v, want none", got)
	}
	// a1 now holds exactly 1 money and, from two ticks, 2 and 3 units of commodity
	// 0 (shelf life 3, so the lots are told apart by their born tick).
	if err := f.l.Transfer(f.mint, f.a1, 1, ledger.Initial); err != nil {
		t.Fatalf("setup: Transfer: %v", err)
	}
	for _, qty := range []int64{2, 3} {
		if err := f.l.Move(f.goodsBoundary, f.a1, 0, qty, ledger.Initial); err != nil {
			t.Fatalf("setup: Move: %v", err)
		}
		f.l.AdvanceTick()
	}
	f.l.ClearTotals()
	if err := f.l.Relocate(f.a1, 1); err != nil {
		t.Fatalf("Relocate(a1, 1) = %v, want nil", err)
	}
	want := []ledger.Moved{
		{From: 0, To: 1, Commodity: ledger.Money, Total: 1},
		{From: 0, To: 1, Commodity: 0, Total: 5},
	}
	if got := f.l.AppendRelocations(nil); !slices.Equal(got, want) {
		t.Errorf("AppendRelocations = %v, want %v (1 money; 2 + 3 units in two lots)", got, want)
	}
}
