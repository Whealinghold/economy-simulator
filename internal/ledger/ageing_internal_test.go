package ledger

import (
	"math"
	"math/rand/v2"
	"reflect"
	"testing"
)

// iledger builds a ledger for the white-box tests: the four commodities of the
// production fixture, two regions, two agents and the boundary accounts the
// operations use. It returns the ledger and the agent IDs.
func iledger(t testing.TB) (*Ledger, [2]int) {
	t.Helper()
	l, err := New(Config{
		Commodities: []Commodity{{ShelfLife: 3}, {Durability: 5}, {}, {ShelfLife: 4, Durability: 6}},
		Regions:     2,
		MaxMoney:    1000,
		MaxQuantity: 100,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	for c := range l.comms {
		for _, r := range []Reason{Initial, RecipeInput, RecipeOutput, Wear, Spoilage} {
			if _, err := l.OpenBoundary(r, c); err != nil {
				t.Fatalf("OpenBoundary(%d, %d): %v", r, c, err)
			}
		}
	}
	var agents [2]int
	for i := range agents {
		id, err := l.Open(KindAgent, i, i)
		if err != nil {
			t.Fatalf("Open: %v", err)
		}
		agents[i] = id
	}
	return l, agents
}

// exactHint is the earliest expiry over the lots of an account, from scratch.
func exactHint(l *Ledger, a *account) (int64, bool) {
	var h int64
	var has bool
	for _, lt := range a.lots {
		if s := l.comms[lt.Commodity].ShelfLife; s > 0 {
			if e := expiry(lt.Born, s); !has || e < h {
				h, has = e, true
			}
		}
	}
	return h, has
}

// checkHints fails if any account's hint is later than its true earliest
// expiry, or missing while a spoiling lot exists: the one way a hint can be
// wrong is late, because an account is visited only when it is reached.
func checkHints(t *testing.T, l *Ledger, context string) {
	t.Helper()
	for id := range l.accounts {
		a := &l.accounts[id]
		h, has := exactHint(l, a)
		if has && (!a.hasHint || a.hint > h) {
			t.Fatalf("%s: account %d hint = %d (set %v), earliest expiry %d", context, id, a.hint, a.hasHint, h)
		}
	}
}

// scriptOps drives a ledger with a fixed pseudo-random sequence of operations,
// refusals included, calling after after every one. The seed is the test's.
func scriptOps(l *Ledger, agents [2]int, seed uint64, n int, after func(op string)) {
	rng := rand.New(rand.NewPCG(seed, 99))
	initial := func(c int) int {
		id, _ := l.Boundary(Initial, c)
		return id
	}
	for i := 0; i < n; i++ {
		c := rng.IntN(len(l.comms))
		who := agents[rng.IntN(2)]
		q := int64(1 + rng.IntN(6))
		switch rng.IntN(7) {
		case 0, 1:
			_ = l.Move(initial(c), who, c, q, Initial)
			after("create")
		case 2:
			_ = l.Transform(who, []Line{{Commodity: c, Quantity: q}}, []Line{{Commodity: rng.IntN(len(l.comms)), Quantity: q}})
			after("transform")
		case 3:
			_ = l.WearItems(who, 1+2*rng.IntN(2), q, int64(1+rng.IntN(5)))
			after("wear")
		case 4:
			_ = l.Move(who, agents[rng.IntN(2)], c, q, Trade)
			after("move")
		case 5:
			_ = l.Age()
			after("age")
			l.EndTick()
			after("end tick")
		default:
			_ = l.ReserveGoods(who, c, q)
			after("reserve")
			_ = l.ReleaseGoods(who, c, q)
			after("release")
		}
	}
}

// TestHints_NeverLateAndRebuiltExactly verifies, after every operation of a
// scripted run over several seeds, that no account's expiry hint is later than
// its earliest expiring lot, and that RebuildHints makes every hint exact and
// is idempotent.
//
// Requirement: NFR-LED-002, FR-LED-030
func TestHints_NeverLateAndRebuiltExactly(t *testing.T) {
	t.Parallel()
	for seed := uint64(1); seed <= 20; seed++ {
		l, agents := iledger(t)
		scriptOps(l, agents, seed, 300, func(op string) { checkHints(t, l, op) })
		l.RebuildHints()
		for id := range l.accounts {
			a := &l.accounts[id]
			h, has := exactHint(l, a)
			if a.hasHint != has || a.hint != h {
				t.Fatalf("seed %d: after RebuildHints account %d hint = %d (%v), want %d (%v)", seed, id, a.hint, a.hasHint, h, has)
			}
		}
		before := make([]account, len(l.accounts))
		copy(before, l.accounts)
		l.RebuildHints()
		if !reflect.DeepEqual(before, l.accounts) {
			t.Errorf("seed %d: RebuildHints is not idempotent", seed)
		}
	}
}

// TestRestore_RebuiltHintsContinueExactly verifies that a ledger whose derived
// hints were discarded and rebuilt from its lots continues exactly as the
// uninterrupted one does: the same states after further operations.
//
// Requirement: FR-LED-030
func TestRestore_RebuiltHintsContinueExactly(t *testing.T) {
	t.Parallel()
	for seed := uint64(1); seed <= 10; seed++ {
		a, agentsA := iledger(t)
		b, agentsB := iledger(t)
		scriptOps(a, agentsA, seed, 120, func(string) {})
		scriptOps(b, agentsB, seed, 120, func(string) {})
		for i := range b.accounts {
			b.accounts[i].hint, b.accounts[i].hasHint = 0, false
		}
		b.RebuildHints()
		// Same continuation on both: ageing every tick for a while.
		for tick := 0; tick < 12; tick++ {
			for _, l := range []*Ledger{a, b} {
				if err := l.Age(); err != nil {
					t.Fatalf("seed %d: Age: %v", seed, err)
				}
				l.EndTick()
			}
		}
		strip := func(l *Ledger) []account {
			out := make([]account, len(l.accounts))
			copy(out, l.accounts)
			for i := range out {
				out[i].hint, out[i].hasHint = 0, false
			}
			return out
		}
		if !reflect.DeepEqual(strip(a), strip(b)) || !reflect.DeepEqual(a.totals, b.totals) || a.tick != b.tick {
			t.Errorf("seed %d: restored ledger diverged from the uninterrupted one", seed)
		}
	}
}

// TestAge_DoesNotVisitAnAccountWhoseHintIsInTheFuture verifies that ageing
// reads an account only when its hint has been reached: a lot planted past the
// hint is left alone, and one in an account whose hint is due is moved.
//
// Guard: the cost of ageing must not grow with the number of live lots
// (NFR-LED-002). If Age scanned every lot, an account with a future hint would
// be read, and this test would see the planted lot move.
//
// Requirement: NFR-LED-002
func TestAge_DoesNotVisitAnAccountWhoseHintIsInTheFuture(t *testing.T) {
	t.Parallel()
	l, agents := iledger(t)
	a0, a1 := &l.accounts[agents[0]], &l.accounts[agents[1]]
	// Both accounts hold a lot that expired at tick -5, planted directly. a0's
	// hint is that tick (due), a1's is far in the future (not due).
	planted := Lot{Commodity: 0, Quantity: 2, Born: -7}
	a0.lots = []Lot{planted}
	a1.lots = []Lot{planted}
	a0.hint, a0.hasHint = -5, true
	a1.hint, a1.hasHint = 1000, true
	if err := l.Age(); err != nil {
		t.Fatalf("Age() = %v", err)
	}
	if len(a0.lots) != 0 {
		t.Errorf("a0 (hint due) holds %v after ageing, want none", a0.lots)
	}
	if !reflect.DeepEqual(a1.lots, []Lot{planted}) {
		t.Errorf("a1 (hint in the future) holds %v, want the planted lot untouched", a1.lots)
	}
}

// TestOverflowOnBoundariesAndFlowTotals verifies that a Transform, a wear and
// an ageing whose boundary balance or flow total would leave int64 are refused
// and leave the ledger unchanged.
//
// Requirement: FR-LED-006, FR-LED-007
func TestOverflowOnBoundariesAndFlowTotals(t *testing.T) {
	t.Parallel()
	slot := func(l *Ledger, region int, r Reason, c int) int {
		return region*l.stride + l.off[r][thingGoods] + c
	}
	tests := []struct {
		name string
		poke func(l *Ledger, agent int)
		op   func(l *Ledger, agent int) error
	}{
		{
			name: "transform input boundary",
			poke: func(l *Ledger, _ int) { l.boundaryAt(RecipeInput, 2).balance = math.MaxInt64 },
			op: func(l *Ledger, a int) error {
				return l.Transform(a, []Line{{Commodity: 2, Quantity: 1}}, nil)
			},
		},
		{
			name: "transform input flow",
			poke: func(l *Ledger, _ int) { l.totals[slot(l, 0, RecipeInput, 2)] = math.MaxInt64 },
			op: func(l *Ledger, a int) error {
				return l.Transform(a, []Line{{Commodity: 2, Quantity: 1}}, nil)
			},
		},
		{
			name: "transform output boundary",
			poke: func(l *Ledger, _ int) { l.boundaryAt(RecipeOutput, 2).balance = math.MinInt64 },
			op: func(l *Ledger, a int) error {
				return l.Transform(a, nil, []Line{{Commodity: 2, Quantity: 1}})
			},
		},
		{
			name: "transform output flow",
			poke: func(l *Ledger, a int) { l.totals[slot(l, 0, RecipeOutput, 2)] = math.MaxInt64 },
			op: func(l *Ledger, a int) error {
				return l.Transform(a, nil, []Line{{Commodity: 2, Quantity: 1}})
			},
		},
		{
			name: "wear boundary",
			poke: func(l *Ledger, _ int) { l.boundaryAt(Wear, 1).balance = math.MaxInt64 },
			op:   func(l *Ledger, a int) error { return l.Wear(a, 1, 99) },
		},
		{
			name: "wear flow",
			poke: func(l *Ledger, _ int) { l.totals[slot(l, 0, Wear, 1)] = math.MaxInt64 },
			op:   func(l *Ledger, a int) error { return l.Wear(a, 1, 99) },
		},
		{
			name: "wear units",
			poke: func(l *Ledger, _ int) { l.wear[1] = math.MaxInt64 },
			op:   func(l *Ledger, a int) error { return l.Wear(a, 1, 1) },
		},
		{
			name: "spoilage boundary",
			poke: func(l *Ledger, _ int) { l.boundaryAt(Spoilage, 0).balance = math.MaxInt64 },
			op:   func(l *Ledger, _ int) error { return l.Age() },
		},
		{
			name: "spoilage flow",
			poke: func(l *Ledger, _ int) { l.totals[slot(l, 0, Spoilage, 0)] = math.MaxInt64 },
			op:   func(l *Ledger, _ int) error { return l.Age() },
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			l, agents := iledger(t)
			a := agents[0]
			if err := l.EndConstruction(); err != nil {
				t.Fatal(err)
			}
			for _, m := range []struct {
				c   int
				qty int64
			}{{0, 2}, {1, 1}, {2, 5}} {
				id, _ := l.Boundary(Initial, m.c)
				if err := l.Move(id, a, m.c, m.qty, Initial); err != nil {
					t.Fatal(err)
				}
			}
			// c0 was born at tick 0 with shelf life 3: it is due at tick 2.
			l.tick = 2
			tt.poke(l, a)
			l.RebuildHints()
			before := snapshotInternal(l)
			err := tt.op(l, a)
			if err != ErrBalanceOverflow {
				t.Errorf("operation = %v, want ErrBalanceOverflow", err)
			}
			if after := snapshotInternal(l); !reflect.DeepEqual(after, before) {
				t.Errorf("ledger changed by a refused operation")
			}
		})
	}
}

// snapshotInternal copies everything an operation may change.
func snapshotInternal(l *Ledger) any {
	type s struct {
		Accounts []account
		Totals   []int64
		Bucket   []int64
		Wear     []int64
		Spoil    []int64
	}
	out := s{
		Accounts: make([]account, len(l.accounts)),
		Totals:   append([]int64(nil), l.totals...),
		Bucket:   append([]int64(nil), l.bucket...),
		Wear:     append([]int64(nil), l.wear...),
		Spoil:    append([]int64(nil), l.spoil...),
	}
	for i, a := range l.accounts {
		a.lots = append([]Lot(nil), a.lots...)
		out.Accounts[i] = a
	}
	return out
}

// TestExpiry_SaturatesAndLineSumsAreChecked verifies that an expiry that does
// not fit int64 means never, and that lines whose sum does not fit int64 are
// refused by a Transform, not wrapped.
//
// Requirement: FR-LED-006
func TestExpiry_SaturatesAndLineSumsAreChecked(t *testing.T) {
	t.Parallel()
	if got := expiry(math.MaxInt64, 3); got != math.MaxInt64 {
		t.Errorf("expiry(MaxInt64, 3) = %d, want MaxInt64", got)
	}
	l, agents := iledger(t)
	big := []Line{{Commodity: 2, Quantity: math.MaxInt64}, {Commodity: 2, Quantity: math.MaxInt64}}
	if err := l.Transform(agents[0], big, nil); err != ErrInsufficientStock {
		t.Errorf("Transform(big inputs) = %v, want ErrInsufficientStock", err)
	}
	if err := l.Transform(agents[0], nil, big); err != ErrBalanceOverflow {
		t.Errorf("Transform(big outputs) = %v, want ErrBalanceOverflow", err)
	}
}
