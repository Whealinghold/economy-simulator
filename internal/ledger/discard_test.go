package ledger_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/ledger"
)

// The commodities of the discard fixture: cDcSpoil spoils, cDcWear wears and
// cDcPlain does neither. The bound is dcMaxQty units of one commodity in one
// non-boundary account.
const (
	cDcSpoil = 0
	cDcWear  = 1
	cDcPlain = 2
	dcMaxQty = 100
)

// dcFix is a ledger with two regions and three commodities, the Initial,
// RecipeOutput and Disposal boundary accounts of every commodity, an agent in
// each region, an estate in region 0 and an empty agent of region 0 to close.
type dcFix struct {
	l                *ledger.Ledger
	a0, a1, e0, gone int
	initial          [3]int
}

// newDC builds the fixture in its construction phase. The boundary account of
// reason omit for commodity omitC is not opened, to test the refusal of a
// ledger that lacks it; pass ledger.NumReasons for none. gone is closed.
func newDC(t *testing.T, omit ledger.Reason, omitC int) *dcFix {
	t.Helper()
	l, err := ledger.New(ledger.Config{
		Commodities: []ledger.Commodity{{ShelfLife: 3}, {Durability: 5}, {}},
		Regions:     2,
		MaxMoney:    1000,
		MaxQuantity: dcMaxQty,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	f := &dcFix{l: l}
	for c := 0; c < 3; c++ {
		for _, r := range []ledger.Reason{ledger.Initial, ledger.RecipeOutput, ledger.Disposal} {
			if r == omit && c == omitC {
				continue
			}
			id, err := l.OpenBoundary(r, c)
			if err != nil {
				t.Fatalf("setup: OpenBoundary(%d, %d): %v", r, c, err)
			}
			if r == ledger.Initial {
				f.initial[c] = id
			}
		}
	}
	open := func(k ledger.Kind, region int) int {
		id, err := l.Open(k, 0, region)
		if err != nil {
			t.Fatalf("setup: Open: %v", err)
		}
		return id
	}
	f.a0, f.a1, f.e0, f.gone = open(ledger.KindAgent, 0), open(ledger.KindAgent, 1), open(ledger.KindEstate, 0), open(ledger.KindAgent, 0)
	if err := l.Close(f.gone); err != nil {
		t.Fatalf("setup: Close: %v", err)
	}
	return f
}

// give creates qty units of commodity c in account id from the Initial
// boundary, in whatever phase the ledger is in.
func (f *dcFix) give(t *testing.T, id, c int, qty int64) {
	t.Helper()
	if err := f.l.Move(f.initial[c], id, c, qty, ledger.Initial); err != nil {
		t.Fatalf("setup: give %d of c%d to %d: %v", qty, c, id, err)
	}
}

// lots returns the lots of an account.
func (f *dcFix) lots(t *testing.T, id int) []ledger.Lot {
	t.Helper()
	got, err := f.l.AppendLots(nil, id)
	if err != nil {
		t.Fatalf("AppendLots(%d): %v", id, err)
	}
	return got
}

// balance is the balance of the boundary account of a reason and commodity.
func (f *dcFix) balance(t *testing.T, r ledger.Reason, c int) int64 {
	t.Helper()
	id, ok := f.l.Boundary(r, c)
	if !ok {
		t.Fatalf("no boundary for reason %d commodity %d", r, c)
	}
	v, err := f.l.Account(id)
	if err != nil {
		t.Fatalf("Account(%d): %v", id, err)
	}
	return v.Balance
}

// total is the sum of commodity c over every boundary account and every
// holding: zero while goods are conserved (FR-LED-002).
func (f *dcFix) total(t *testing.T, c int) int64 {
	t.Helper()
	var n int64
	for id := 0; id < f.l.Len(); id++ {
		v, err := f.l.Account(id)
		if err != nil {
			t.Fatalf("Account(%d): %v", id, err)
		}
		if v.Kind == ledger.KindBoundary {
			continue
		}
		for _, lt := range f.lots(t, id) {
			if lt.Commodity == c {
				n += lt.Quantity
			}
		}
	}
	for _, r := range []ledger.Reason{ledger.Initial, ledger.RecipeOutput, ledger.Disposal} {
		if _, ok := f.l.Boundary(r, c); ok {
			n += f.balance(t, r, c)
		}
	}
	return n
}

// dcState is everything observable about the ledger, lots included, for
// comparing a ledger with its earlier self with reflect.DeepEqual.
type dcState struct {
	Accounts       []ledger.View
	Lots           [][]ledger.Lot
	Totals, Bucket []ledger.Flow
	Construction   bool
	Tick           int64
}

func (f *dcFix) state(t *testing.T) dcState {
	t.Helper()
	s := dcState{
		Totals:       f.l.AppendFlows(nil, false),
		Bucket:       f.l.AppendFlows(nil, true),
		Construction: f.l.InConstruction(),
		Tick:         f.l.Tick(),
	}
	for id := 0; id < f.l.Len(); id++ {
		v, err := f.l.Account(id)
		if err != nil {
			t.Fatalf("Account(%d): %v", id, err)
		}
		s.Accounts = append(s.Accounts, v)
		s.Lots = append(s.Lots, f.lots(t, id))
	}
	return s
}

// TestDiscard_CreatesAndSinksWithoutHolding verifies that a discard moves the
// quantity in from recipe_output and out to disposal in one step: the
// account's lots do not change, the two boundaries move by the quantity, goods
// stay conserved, and both flows are attributed to the account's region.
//
// Setup: the account holds 7 units of the plain commodity (7 created by
// initial, so initial is -7). Discard of 5 takes recipe_output to -5 and
// disposal to +5; the sum is -7 + 7 - 5 + 5 = 0. The totals show 5 under each
// of the two reasons, in the region of the account.
//
// Requirement: FR-LED-001, FR-LED-037, FR-LED-066
func TestDiscard_CreatesAndSinksWithoutHolding(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		holder func(f *dcFix) int
		region int
	}{
		{"agent of region 0", func(f *dcFix) int { return f.a0 }, 0},
		{"agent of region 1", func(f *dcFix) int { return f.a1 }, 1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			f := newDC(t, ledger.NumReasons, 0)
			id := tc.holder(f)
			f.give(t, id, cDcPlain, 7)
			if err := f.l.EndConstruction(); err != nil {
				t.Fatalf("EndConstruction: %v", err)
			}
			before := f.lots(t, id)

			if err := f.l.Discard(id, cDcPlain, 5); err != nil {
				t.Fatalf("Discard(%d, plain, 5) = %v, want nil", id, err)
			}
			if got := f.lots(t, id); !reflect.DeepEqual(got, before) {
				t.Errorf("lots after Discard = %v, want %v unchanged", got, before)
			}
			if got := f.balance(t, ledger.RecipeOutput, cDcPlain); got != -5 {
				t.Errorf("recipe_output boundary = %d, want -5", got)
			}
			if got := f.balance(t, ledger.Disposal, cDcPlain); got != 5 {
				t.Errorf("disposal boundary = %d, want 5", got)
			}
			if got := f.total(t, cDcPlain); got != 0 {
				t.Errorf("total of the commodity = %d, want 0", got)
			}
			want := []ledger.Flow{
				{Region: tc.region, Reason: ledger.RecipeOutput, Commodity: cDcPlain, Total: 5},
				{Region: tc.region, Reason: ledger.Disposal, Commodity: cDcPlain, Total: 5},
			}
			if got := f.l.AppendFlows(nil, false); !reflect.DeepEqual(got, want) {
				t.Errorf("AppendFlows(totals) = %v, want %v", got, want)
			}
			if err := f.l.Audit(); err != nil {
				t.Errorf("Audit() = %v, want nil", err)
			}
		})
	}
}

// TestDiscard_AccountAtTheBound verifies that an account holding exactly
// MaxQuantity of a commodity accepts a discard of 1 and of MaxQuantity with its
// holding unchanged, where a Move of the same quantity in from recipe_output is
// refused with ErrBalanceOverflow. This is the case the operation exists for:
// the completion of a job never fails for room.
//
// Requirement: FR-LED-006, FR-LED-066
func TestDiscard_AccountAtTheBound(t *testing.T) {
	t.Parallel()
	for _, qty := range []int64{1, dcMaxQty} {
		f := newDC(t, ledger.NumReasons, 0)
		f.give(t, f.a0, cDcPlain, dcMaxQty)
		out, _ := f.l.Boundary(ledger.RecipeOutput, cDcPlain)

		// 100 held + 1 would be 101 > 100, whatever qty is: the Move is refused.
		if err := f.l.Move(out, f.a0, cDcPlain, qty, ledger.RecipeOutput); !errors.Is(err, ledger.ErrBalanceOverflow) {
			t.Errorf("Move(recipe_output, a0, %d) at the bound = %v, want ErrBalanceOverflow", qty, err)
		}
		if err := f.l.Discard(f.a0, cDcPlain, qty); err != nil {
			t.Errorf("Discard(a0, plain, %d) at the bound = %v, want nil", qty, err)
		}
		held, reserved, err := f.l.Holding(f.a0, cDcPlain)
		if err != nil || held != dcMaxQty || reserved != 0 {
			t.Errorf("Holding after Discard(%d) = %d, %d, %v, want %d, 0, nil", qty, held, reserved, err, dcMaxQty)
		}
	}
}

// TestDiscard_Refusals verifies each refusal with a case that breaks only that
// rule, and that a refusal changes nothing: the ledger is DeepEqual to a copy
// taken before, flows and bucket included.
//
// Requirement: FR-LED-005, FR-LED-007, FR-LED-066
func TestDiscard_Refusals(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		// omit and omitC choose the boundary account the fixture leaves out.
		omit  ledger.Reason
		omitC int
		id    func(f *dcFix) int
		c     int
		qty   int64
		want  error
	}{
		{"quantity zero", ledger.NumReasons, 0, func(f *dcFix) int { return f.a0 }, cDcPlain, 0, ledger.ErrInvalidAmount},
		{"quantity negative", ledger.NumReasons, 0, func(f *dcFix) int { return f.a0 }, cDcPlain, -3, ledger.ErrInvalidAmount},
		{"account below the table", ledger.NumReasons, 0, func(*dcFix) int { return -1 }, cDcPlain, 1, ledger.ErrUnknownAccount},
		{"account above the table", ledger.NumReasons, 0, func(f *dcFix) int { return f.l.Len() }, cDcPlain, 1, ledger.ErrUnknownAccount},
		{"closed account", ledger.NumReasons, 0, func(f *dcFix) int { return f.gone }, cDcPlain, 1, ledger.ErrClosedAccount},
		{"commodity below the table", ledger.NumReasons, 0, func(f *dcFix) int { return f.a0 }, -1, 1, ledger.ErrUnknownCommodity},
		{"commodity above the table", ledger.NumReasons, 0, func(f *dcFix) int { return f.a0 }, 3, 1, ledger.ErrUnknownCommodity},
		{"commodity well above the table", ledger.NumReasons, 0, func(f *dcFix) int { return f.a0 }, 4, 1, ledger.ErrUnknownCommodity},
		{"estate is not an agent", ledger.NumReasons, 0, func(f *dcFix) int { return f.e0 }, cDcPlain, 1, ledger.ErrReasonMismatch},
		{"boundary account", ledger.NumReasons, 0, func(f *dcFix) int { return f.initial[cDcPlain] }, cDcPlain, 1, ledger.ErrReasonMismatch},
		{"recipe_output boundary never opened", ledger.RecipeOutput, cDcPlain, func(f *dcFix) int { return f.a0 }, cDcPlain, 1, ledger.ErrUnknownAccount},
		{"disposal boundary never opened", ledger.Disposal, cDcPlain, func(f *dcFix) int { return f.a0 }, cDcPlain, 1, ledger.ErrUnknownAccount},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			f := newDC(t, tc.omit, tc.omitC)
			f.give(t, f.a0, cDcPlain, 7)
			before := f.state(t)
			id := tc.id(f)
			if err := f.l.Discard(id, tc.c, tc.qty); !errors.Is(err, tc.want) {
				t.Errorf("Discard(%d, %d, %d) = %v, want %v", id, tc.c, tc.qty, err, tc.want)
			}
			if after := f.state(t); !reflect.DeepEqual(after, before) {
				t.Errorf("Discard(%d, %d, %d) refused but changed the ledger:\n got %+v\nwant %+v", id, tc.c, tc.qty, after, before)
			}
		})
	}
}

// TestDiscard_IgnoresEarmarksAndAvailableStock verifies that units earmarked
// for an order stay earmarked and untouched, and that a discard larger than the
// available stock succeeds, because nothing is taken from the account.
//
// Setup: 10 held, 8 earmarked, so 2 are available. Discard of 50 is more than
// the 2 available and more than the 10 held.
//
// Requirement: FR-LED-066
func TestDiscard_IgnoresEarmarksAndAvailableStock(t *testing.T) {
	t.Parallel()
	f := newDC(t, ledger.NumReasons, 0)
	f.give(t, f.a0, cDcPlain, 10)
	if err := f.l.ReserveGoods(f.a0, cDcPlain, 8); err != nil {
		t.Fatalf("ReserveGoods: %v", err)
	}
	before := f.lots(t, f.a0)
	if err := f.l.Discard(f.a0, cDcPlain, 50); err != nil {
		t.Fatalf("Discard(a0, plain, 50) = %v, want nil", err)
	}
	if got := f.lots(t, f.a0); !reflect.DeepEqual(got, before) {
		t.Errorf("lots after Discard = %v, want %v unchanged", got, before)
	}
	if held, reserved, _ := f.l.Holding(f.a0, cDcPlain); held != 10 || reserved != 8 {
		t.Errorf("Holding after Discard = %d, %d, want 10, 8", held, reserved)
	}
}

// TestDiscard_ConstructionBucket verifies that a discard in the construction
// phase lands in the construction bucket and not in the tick's totals, and
// that after EndConstruction it lands in the totals and leaves the bucket.
//
// Requirement: FR-LED-045, FR-LED-053, FR-LED-066
func TestDiscard_ConstructionBucket(t *testing.T) {
	t.Parallel()
	f := newDC(t, ledger.NumReasons, 0)
	if err := f.l.Discard(f.a0, cDcPlain, 4); err != nil {
		t.Fatalf("Discard in construction: %v", err)
	}
	pair := func(n int64) []ledger.Flow {
		return []ledger.Flow{
			{Region: 0, Reason: ledger.RecipeOutput, Commodity: cDcPlain, Total: n},
			{Region: 0, Reason: ledger.Disposal, Commodity: cDcPlain, Total: n},
		}
	}
	if got := f.l.AppendFlows(nil, true); !reflect.DeepEqual(got, pair(4)) {
		t.Errorf("bucket after Discard of 4 in construction = %v, want %v", got, pair(4))
	}
	if got := f.l.AppendFlows(nil, false); len(got) != 0 {
		t.Errorf("totals after Discard in construction = %v, want none", got)
	}
	if err := f.l.EndConstruction(); err != nil {
		t.Fatalf("EndConstruction: %v", err)
	}
	if err := f.l.Discard(f.a0, cDcPlain, 6); err != nil {
		t.Fatalf("Discard after construction: %v", err)
	}
	if got := f.l.AppendFlows(nil, false); !reflect.DeepEqual(got, pair(6)) {
		t.Errorf("totals after Discard of 6 = %v, want %v", got, pair(6))
	}
	if got := f.l.AppendFlows(nil, true); !reflect.DeepEqual(got, pair(4)) {
		t.Errorf("bucket after Discard of 6 = %v, want %v unchanged", got, pair(4))
	}
}

// TestHolding_ReadsHeldAndEarmarked verifies the read of an account's units of
// a commodity and how many are earmarked, through a move in, a reservation, a
// marked move and a release, and the three error and empty cases.
//
// Setup: a0 receives 7 (7, 0); ReserveGoods 3 (7, 3); a marked move of 2 to a1
// takes 2 earmarked units, leaving a0 with (5, 1) and a1 with (2, 0);
// ReleaseGoods 1 leaves a0 with (5, 0).
//
// Requirement: FR-LED-067
func TestHolding_ReadsHeldAndEarmarked(t *testing.T) {
	t.Parallel()
	f := newDC(t, ledger.NumReasons, 0)
	steps := []struct {
		name         string
		do           func() error
		held, marked int64
		a1Held       int64
	}{
		{"move in", func() error { return f.l.Move(f.initial[cDcWear], f.a0, cDcWear, 7, ledger.Initial) }, 7, 0, 0},
		{"reserve", func() error { return f.l.ReserveGoods(f.a0, cDcWear, 3) }, 7, 3, 0},
		{"marked move", func() error { return f.l.MoveReserved(f.a0, f.a1, cDcWear, 2, ledger.Trade) }, 5, 1, 2},
		{"release", func() error { return f.l.ReleaseGoods(f.a0, cDcWear, 1) }, 5, 0, 2},
	}
	for _, s := range steps {
		if err := s.do(); err != nil {
			t.Fatalf("%s: %v", s.name, err)
		}
		held, marked, err := f.l.Holding(f.a0, cDcWear)
		if err != nil || held != s.held || marked != s.marked {
			t.Errorf("Holding(a0, wear) after %s = %d, %d, %v, want %d, %d, nil", s.name, held, marked, err, s.held, s.marked)
		}
		if h, m, err := f.l.Holding(f.a1, cDcWear); err != nil || h != s.a1Held || m != 0 {
			t.Errorf("Holding(a1, wear) after %s = %d, %d, %v, want %d, 0, nil", s.name, h, m, err, s.a1Held)
		}
		if h, m, err := f.l.Holding(f.a0, cDcPlain); err != nil || h != 0 || m != 0 {
			t.Errorf("Holding(a0, plain) after %s = %d, %d, %v, want 0, 0, nil: another commodity is not counted", s.name, h, m, err)
		}
	}
	// Reading changes nothing.
	before := f.state(t)
	if _, _, err := f.l.Holding(f.a0, cDcWear); err != nil {
		t.Fatalf("Holding: %v", err)
	}
	if after := f.state(t); !reflect.DeepEqual(after, before) {
		t.Errorf("Holding changed the ledger")
	}
}

// TestHolding_Refusals verifies the error cases and the empty cases of the
// read: an unknown account and an unknown commodity are refused, and a closed
// account and a boundary hold no lots and return 0 and 0.
//
// Requirement: FR-LED-067
func TestHolding_Refusals(t *testing.T) {
	t.Parallel()
	f := newDC(t, ledger.NumReasons, 0)
	f.give(t, f.a0, cDcPlain, 3)
	tests := []struct {
		name         string
		id, c        int
		held, marked int64
		want         error
	}{
		{"account below the table", -1, cDcPlain, 0, 0, ledger.ErrUnknownAccount},
		{"account above the table", f.l.Len(), cDcPlain, 0, 0, ledger.ErrUnknownAccount},
		{"commodity below the table", f.a0, -1, 0, 0, ledger.ErrUnknownCommodity},
		{"commodity above the table", f.a0, 3, 0, 0, ledger.ErrUnknownCommodity},
		{"commodity well above the table", f.a0, 4, 0, 0, ledger.ErrUnknownCommodity},
		{"first commodity is valid", f.a0, cDcSpoil, 0, 0, nil},
		{"closed account", f.gone, cDcPlain, 0, 0, nil},
		{"boundary account", f.initial[cDcPlain], cDcPlain, 0, 0, nil},
		{"agent with goods", f.a0, cDcPlain, 3, 0, nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			held, marked, err := f.l.Holding(tc.id, tc.c)
			if !errors.Is(err, tc.want) || held != tc.held || marked != tc.marked {
				t.Errorf("Holding(%d, %d) = %d, %d, %v, want %d, %d, %v", tc.id, tc.c, held, marked, err, tc.held, tc.marked, tc.want)
			}
		})
	}
}

// TestDiscard_AuditHoldsThroughAScript verifies that a scripted run that
// interleaves Move, Discard and the reservations keeps the ledger's audit
// passing after every operation, in construction and in a tick.
//
// Requirement: FR-LED-035, FR-LED-036, FR-LED-066
func TestDiscard_AuditHoldsThroughAScript(t *testing.T) {
	t.Parallel()
	f := newDC(t, ledger.NumReasons, 0)
	steps := []struct {
		name string
		do   func() error
	}{
		{"give", func() error { return f.l.Move(f.initial[cDcSpoil], f.a0, cDcSpoil, 9, ledger.Initial) }},
		{"discard in construction", func() error { return f.l.Discard(f.a0, cDcSpoil, 4) }},
		{"reserve", func() error { return f.l.ReserveGoods(f.a0, cDcSpoil, 5) }},
		{"end construction", f.l.EndConstruction},
		{"discard with earmarks", func() error { return f.l.Discard(f.a0, cDcSpoil, 30) }},
		{"marked move", func() error { return f.l.MoveReserved(f.a0, f.a1, cDcSpoil, 2, ledger.Trade) }},
		{"discard from the other region", func() error { return f.l.Discard(f.a1, cDcSpoil, 1) }},
		{"release", func() error { return f.l.ReleaseGoods(f.a0, cDcSpoil, 3) }},
		{"move on", func() error { return f.l.Move(f.a0, f.a1, cDcSpoil, 6, ledger.Trade) }},
	}
	for _, s := range steps {
		if err := s.do(); err != nil {
			t.Fatalf("%s: %v", s.name, err)
		}
		if err := f.l.Audit(); err != nil {
			t.Fatalf("Audit() after %s = %v, want nil", s.name, err)
		}
	}
}

// TestDiscardAndHolding_AllocateNothing verifies that Discard and Holding
// allocate nothing on the path of a completion.
//
// Requirement: NFR-LED-001
func TestDiscardAndHolding_AllocateNothing(t *testing.T) {
	f := newDC(t, ledger.NumReasons, 0)
	f.give(t, f.a0, cDcPlain, 10)
	if n := testing.AllocsPerRun(100, func() {
		if err := f.l.Discard(f.a0, cDcPlain, 1); err != nil {
			t.Fatalf("Discard: %v", err)
		}
	}); n != 0 {
		t.Errorf("Discard allocates %v times per call, want 0", n)
	}
	if n := testing.AllocsPerRun(100, func() {
		if _, _, err := f.l.Holding(f.a0, cDcPlain); err != nil {
			t.Fatalf("Holding: %v", err)
		}
	}); n != 0 {
		t.Errorf("Holding allocates %v times per call, want 0", n)
	}
}

// BenchmarkDiscard measures a discard of one unit.
//
// Requirement: NFR-LED-003
func BenchmarkDiscard(b *testing.B) {
	b.ReportAllocs()
	l, a0 := benchDiscardLedger(b)
	for b.Loop() {
		if err := l.Discard(a0, cDcPlain, 1); err != nil {
			b.Fatalf("Discard: %v", err)
		}
	}
}

// BenchmarkHolding measures the read of an account's holding.
//
// Requirement: NFR-LED-003
func BenchmarkHolding(b *testing.B) {
	b.ReportAllocs()
	l, a0 := benchDiscardLedger(b)
	for b.Loop() {
		if _, _, err := l.Holding(a0, cDcPlain); err != nil {
			b.Fatalf("Holding: %v", err)
		}
	}
}

// benchDiscardLedger returns a ledger with one agent holding 10 units of the
// plain commodity.
func benchDiscardLedger(b *testing.B) (*ledger.Ledger, int) {
	b.Helper()
	l, err := ledger.New(ledger.Config{Commodities: []ledger.Commodity{{}}, Regions: 1, MaxMoney: 1000, MaxQuantity: dcMaxQty})
	if err != nil {
		b.Fatalf("New: %v", err)
	}
	init, err := l.OpenBoundary(ledger.Initial, 0)
	if err == nil {
		_, err = l.OpenBoundary(ledger.RecipeOutput, 0)
	}
	if err == nil {
		_, err = l.OpenBoundary(ledger.Disposal, 0)
	}
	if err != nil {
		b.Fatalf("OpenBoundary: %v", err)
	}
	a, err := l.Open(ledger.KindAgent, 0, 0)
	if err != nil {
		b.Fatalf("Open: %v", err)
	}
	if err := l.Move(init, a, 0, 10, ledger.Initial); err != nil {
		b.Fatalf("Move: %v", err)
	}
	return l, a
}
