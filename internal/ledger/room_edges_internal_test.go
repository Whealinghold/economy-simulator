package ledger

import (
	"errors"
	"math"
	"strings"
	"testing"
)

// edgeLedger returns a ledger of one commodity of size (1, 1) with a limited
// agent a, a boundary of reason Initial b and a boundary of reason Disposal d,
// all open, in the run phase.
func edgeLedger(t *testing.T) (l *Ledger, a, b, d int) {
	t.Helper()
	l, err := New(Config{Commodities: []Commodity{{}}, Regions: 1, MaxMoney: 1000, MaxQuantity: 1000})
	if err != nil {
		t.Fatalf("setup: New: %v", err)
	}
	if err := l.SetSizes([]Size{{Mass: 1, Bulk: 1}}); err != nil {
		t.Fatalf("setup: SetSizes: %v", err)
	}
	if b, err = l.OpenBoundary(Initial, 0); err != nil {
		t.Fatalf("setup: OpenBoundary: %v", err)
	}
	if d, err = l.OpenBoundary(Disposal, 0); err != nil {
		t.Fatalf("setup: OpenBoundary: %v", err)
	}
	if a, err = l.Open(KindAgent, 0, 0); err != nil {
		t.Fatalf("setup: Open: %v", err)
	}
	if err := l.EndConstruction(); err != nil {
		t.Fatalf("setup: EndConstruction: %v", err)
	}
	if err := l.SetRoom(a, true, 100, 100); err != nil {
		t.Fatalf("setup: SetRoom: %v", err)
	}
	return l, a, b, d
}

// mustPanicWith runs f and fails the test unless it panics with a message that
// contains want.
func mustPanicWith(t *testing.T, name, want string, f func()) {
	t.Helper()
	defer func() {
		s, _ := recover().(string)
		if !strings.Contains(s, want) {
			t.Errorf("%s: panic = %q, want one containing %q", name, s, want)
		}
	}()
	f()
}

// TestHeadroom_Saturation verifies the arithmetic of the free room at the
// ends of int64: a use that does not fit int64 reads as no room, a freed load
// that overflows the sum reads as the largest room, and a negative result is
// floored at 0.
//
// Guard: headroom is the one place where a free room is computed. A wrap here
// would read a full account as empty and admit a credit the room cannot hold.
//
// Requirement: FR-STO-005
func TestHeadroom_Saturation(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name                       string
		room, a, b, c, freed, want int64
	}{
		{"plain", 10, 2, 3, 1, 0, 4},
		{"freed raises it", 10, 2, 3, 1, 5, 9},
		{"floored at zero", 10, 6, 6, 0, 0, 0},
		{"freed lifts a negative", 10, 6, 6, 0, 5, 3},
		{"freed alone cannot forgive a deeper deficit", 10, 8, 8, 0, 5, 0},
		{"room less use underflows", math.MinInt64, 1, 0, 0, 0, 0},
		{"second use underflows", math.MinInt64 + 1, 1, 1, 0, 0, 0},
		{"third use underflows", math.MinInt64 + 2, 1, 1, 1, 0, 0},
		{"freed overflows the sum", math.MaxInt64, 0, 0, 0, 1, math.MaxInt64},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := headroom(tt.room, tt.a, tt.b, tt.c, tt.freed); got != tt.want {
				t.Errorf("headroom(%d, %d, %d, %d, %d) = %d, want %d", tt.room, tt.a, tt.b, tt.c, tt.freed, got, tt.want)
			}
		})
	}
}

// TestLoadUpdates_ProgrammerErrors verifies that the two updates of the load
// panic, with the invariant message, on a credit that overflows the load and
// on a debit of more than the account carries: the checks before an operation
// make both unreachable, so reaching one is a bug that must not be absorbed.
//
// Guard: a silent clamp would let the load drift from the lots, which the
// audit then reports far from the cause.
//
// Requirement: FR-STO-006
func TestLoadUpdates_ProgrammerErrors(t *testing.T) {
	t.Parallel()
	l, a, _, _ := edgeLedger(t)
	acc := &l.accounts[a]
	mustPanicWith(t, "loadSub of more than the load", "debit took more load", func() { l.loadSub(acc, 0, 1) })
	acc.room.LoadMass = math.MaxInt64
	mustPanicWith(t, "loadAdd overflowing the mass", "overflowed the load", func() { l.loadAdd(acc, 0, 1) })
	acc.room.LoadMass, acc.room.LoadBulk = 0, math.MaxInt64
	mustPanicWith(t, "loadAdd overflowing the bulk", "overflowed the load", func() { l.loadAdd(acc, 0, 1) })
	acc.room.LoadBulk = 0
	l.sizes[0] = Size{Mass: math.MaxInt64, Bulk: 1}
	mustPanicWith(t, "loadSub of a unit load that overflows", "debit took more load", func() { l.loadSub(acc, 0, 2) })
	mustPanicWith(t, "loadAdd of a unit load that overflows", "overflowed the load", func() { l.loadAdd(acc, 0, 2) })
}

// TestRoom_BoundaryBalanceOverflowIsRefusedBeforeTheRoomForm verifies that a
// goods move into a boundary account whose balance would overflow int64 is
// ErrBalanceOverflow, not a room refusal, and changes nothing.
//
// Guard: the boundary branch of the credit check does the balance sum first and
// the form check second; swapping them would report a room refusal for an
// overflow, or accept the move.
//
// Requirement: FR-STO-007
func TestRoom_BoundaryBalanceOverflowIsRefusedBeforeTheRoomForm(t *testing.T) {
	t.Parallel()
	l, a, b, d := edgeLedger(t)
	if err := l.Move(b, a, 0, 5, Initial); err != nil {
		t.Fatalf("setup: Move: %v", err)
	}
	l.accounts[d].balance = math.MaxInt64
	before := l.accounts[a].room.LoadMass
	if err := l.Move(a, d, 0, 1, Disposal); !errors.Is(err, ErrBalanceOverflow) {
		t.Errorf("Move into a boundary at MaxInt64 = %v, want ErrBalanceOverflow", err)
	}
	if got := l.accounts[a].room.LoadMass; got != before {
		t.Errorf("load after the refusal = %d, want %d", got, before)
	}
}
