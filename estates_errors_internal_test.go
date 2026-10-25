package economysim

import (
	"math"
	"strings"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/ledger"
)

// smallLedger is a one-region ledger with a money bucket of at most 100 and an
// open boundary account of initial money, used where the pack-one table would
// hide a refusal.
func smallLedger(t *testing.T, boundary bool) *ledger.Ledger {
	t.Helper()
	l, err := ledger.New(ledger.Config{Commodities: []ledger.Commodity{{}}, Regions: 1, MaxMoney: 100, MaxQuantity: 100})
	if err != nil {
		t.Fatal(err)
	}
	if boundary {
		if _, err := l.OpenBoundary(ledger.Initial, ledger.Money); err != nil {
			t.Fatal(err)
		}
	}
	return l
}

// TestOpenEstates_OpenRefusedIsReturned verifies that a ledger that refuses to
// open an estate account (a region it does not have) makes openEstates return
// an error that names the region, and no estates.
//
// Requirement: FR-EST-004
func TestOpenEstates_OpenRefusedIsReturned(t *testing.T) {
	t.Parallel()
	e, err := openEstates(smallLedger(t, true), 2)
	if err == nil || e != nil {
		t.Fatalf("openEstates(1-region ledger, 2) = %v, %v, want nil, an error", e, err)
	}
	if !strings.Contains(err.Error(), "region 1") {
		t.Errorf("openEstates error = %q, want it to name region 1", err)
	}
}

// TestSeedPool_RefusalsAreReturnedOrInvariant verifies that a seed the ledger
// refuses (above the money bound) is returned wrapped, naming the region, and
// that a ledger with no boundary account of initial money is an invariant.
//
// Arithmetic: the ledger holds at most 100 money, so a seed of 101 overflows.
//
// Requirement: FR-EST-015
func TestSeedPool_RefusalsAreReturnedOrInvariant(t *testing.T) {
	t.Parallel()
	t.Run("a seed above the bound is returned", func(t *testing.T) {
		t.Parallel()
		l := smallLedger(t, true)
		e, err := openEstates(l, 1)
		if err != nil {
			t.Fatal(err)
		}
		err = e.seedPool(l, 0, 101)
		if err == nil || !strings.Contains(err.Error(), "region 0") {
			t.Errorf("seedPool(0, 101) = %v, want an error naming region 0", err)
		}
		if got := e.pool(l, 0); got != 0 {
			t.Errorf("pool after a refused seed = %d, want 0", got)
		}
	})
	t.Run("no boundary account of initial money is an invariant", func(t *testing.T) {
		t.Parallel()
		l := smallLedger(t, false)
		e, err := openEstates(l, 1)
		if err != nil {
			t.Fatal(err)
		}
		mustPanicInvariant(t, "seedPool", func() { _ = e.seedPool(l, 0, 5) })
	})
}

// TestEstates_UnreadableAccountIsAnInvariant verifies that every read of an
// estate or of a removed agent's account that the ledger cannot serve is an
// invariant, not a silent zero: the callers hold accounts they opened, so the
// branch can only be reached by a corrupted world.
//
// Guard: without it the branches are unreachable from the public API and a
// refactor could turn them into a quiet wrong answer.
func TestEstates_UnreadableAccountIsAnInvariant(t *testing.T) {
	t.Parallel()
	const bad = 1 << 20
	tests := []struct {
		name string
		call func(f *listFixture)
	}{
		{"pool", func(f *listFixture) { f.e.accounts[0] = bad; f.e.pool(f.l, 0) }},
		{"held lots", func(f *listFixture) { f.e.accounts[0] = bad; f.e.heldLots(f.l, 0) }},
		{"receive from a missing account", func(f *listFixture) { _, _, _ = f.e.receive(f.l, bad, 0) }},
		{"accept goods from a missing account", func(f *listFixture) {
			_, _ = f.e.acceptGoods(f.l, bad, 0, ledger.EstateGoods, nil)
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			f := newListFixture(t, 1)
			mustPanicInvariant(t, tc.name, func() { tc.call(f) })
		})
	}
}

// TestEstates_BadConfigurationIsAnInvariant verifies the invariants that a
// validated configuration cannot reach: a belief step outside its bound in the
// asks and the markdown, a first price that overflows, and an admitted ask for
// a lot with no listing.
//
// Guard: validation (architecture.md 8.5) keeps these out; the test pins that
// the estate fails loudly rather than pricing with a wrong floor.
func TestEstates_BadConfigurationIsAnInvariant(t *testing.T) {
	t.Parallel()
	t.Run("asks with a belief step out of bound", func(t *testing.T) {
		t.Parallel()
		f := newListFixture(t, 1)
		f.cfg.World.BeliefStep = 0
		mustPanicInvariant(t, "asks", func() { _, _ = f.e.asks(f.l, f.cfg, resolveRefPrices(f.cfg), 0, nil) })
	})
	t.Run("markdown with a belief step out of bound", func(t *testing.T) {
		t.Parallel()
		f := newListFixture(t, 1)
		f.cfg.World.BeliefStep = 0
		mustPanicInvariant(t, "markdown", func() { _ = f.e.markdown(f.l, f.cfg, 0) })
	})
	t.Run("a first price that overflows", func(t *testing.T) {
		t.Parallel()
		f := newListFixture(t, 1)
		// The lot is valid for the ledger (remaining durability 6 of 10); the
		// configuration is then changed to a full durability of 1, which a
		// validated one cannot be for this lot, so ref * 6 / 1 exceeds int64.
		f.give(t, 0, lstTool, 6, 1)
		f.cfg.Commodities[lstTool].ReferencePrice = math.MaxInt64 / 2
		f.cfg.Commodities[lstTool].Durability = 1
		mustPanicInvariant(t, "asks", func() { _, _ = f.e.asks(f.l, f.cfg, resolveRefPrices(f.cfg), 0, nil) })
	})
	t.Run("an admitted ask with no listing", func(t *testing.T) {
		t.Parallel()
		f := newListFixture(t, 1)
		mustPanicInvariant(t, "admitted", func() { f.e.admitted(0, Order{Commodity: lstStaple}) })
	})
}

// TestEstateAsks_MarksOfTheLastTickAreCleared verifies that an ask admitted in
// one walk does not make the lot fall at a markdown that follows a later walk
// in which no ask was admitted.
//
// Arithmetic: staple 100 with markdown 500 would fall to 95 if the mark stayed.
//
// Requirement: FR-EST-007, FR-EST-009
func TestEstateAsks_MarksOfTheLastTickAreCleared(t *testing.T) {
	t.Parallel()
	f := newListFixture(t, 1)
	f.threeLots(t)
	for _, o := range ordersOf(t, f) {
		f.e.admitted(0, o)
	}
	ordersOf(t, f) // the next walk clears the marks; nothing is admitted in it
	if err := f.e.markdown(f.l, f.cfg, 0); err != nil {
		t.Fatal(err)
	}
	if got := f.prices()[k(lstStaple, 0, 0)]; got != 100 {
		t.Errorf("price after a walk with no admitted ask = %d, want 100", got)
	}
}
