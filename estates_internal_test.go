package economysim

import (
	"errors"
	"math"
	"slices"
	"strings"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/ledger"
)

func k(c int, born, dur int64) ledger.Key {
	return ledger.Key{Commodity: c, Born: born, Durability: dur}
}

func lotOf(c int, born, dur int64) ledger.Lot {
	return ledger.Lot{Commodity: c, Born: born, Durability: dur, Quantity: 1}
}

// TestEstates_Identity verifies that the estate pseudo-agent of a region has
// the RegionID as its agent ID, that regionOf is true exactly below the region
// count and that account reads the slot of the region.
//
// Guard: a regionOf that accepted the count itself, or a negative ID, would
// make a household the estate of a region that does not exist (FR-EST-005).
//
// Requirement: FR-EST-005, FR-EST-011
func TestEstates_Identity(t *testing.T) {
	t.Parallel()
	e := &estates{accounts: []int{64, 65, 66}, listed: make([][]listing, 3)}
	tests := []struct {
		name   string
		a      AgentID
		region RegionID
		ok     bool
	}{
		{"first region", 0, 0, true},
		{"last region", 2, 2, true},
		{"the region count", 3, 0, false},
		{"far above", 1 << 40, 0, false},
		{"negative", -1, 0, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			r, ok := e.regionOf(tc.a)
			if r != tc.region || ok != tc.ok {
				t.Fatalf("regionOf(%d) = %d, %v; want %d, %v", tc.a, r, ok, tc.region, tc.ok)
			}
			if ok && e.agentID(r) != tc.a {
				t.Fatalf("agentID(%d) = %d, want %d", r, e.agentID(r), tc.a)
			}
		})
	}
	if got := e.account(1); got != 65 {
		t.Fatalf("account(1) = %d, want 65", got)
	}
}

// TestFirstPrice verifies the first price of a lot: the reference price scaled
// by remaining over full durability, truncated, never below the floor, and the
// unscaled reference price (also floored) for a commodity that does not wear.
//
// Arithmetic: 100 * 3 / 10 = 30; 100 * 7 / 10 = 70; 60 * 1 / 10 = 6, raised to
// the floor 50; 7 * 1 / 3 = 2.33 truncated to 2, raised to the floor 1 never
// lowers it; reference 3 with floor 50 gives 50.
//
// Requirement: FR-EST-008
func TestFirstPrice(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                          string
		ref                           Price
		durability, left, floor, want int64
	}{
		{"durable, 3 of 10", 100, 10, 3, 4, 30},
		{"durable, 7 of 10", 100, 10, 7, 4, 70},
		{"durable, full", 100, 10, 10, 4, 100},
		{"floor binds", 60, 10, 1, 50, 50},
		{"truncates", 7, 3, 1, 1, 2},
		{"non-durable is the reference", 100, 0, 0, 4, 100},
		{"non-durable below the floor", 3, 0, 0, 50, 50},
		{"worn out lot is the floor", 100, 10, 0, 4, 4},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, ok := firstPrice(tc.ref, tc.durability, tc.left, tc.floor)
			if !ok || int64(got) != tc.want {
				t.Fatalf("firstPrice(%d, %d, %d, %d) = %d, %v; want %d", tc.ref, tc.durability, tc.left, tc.floor, got, ok, tc.want)
			}
		})
	}
	// Out-of-contract input (left > durability): 2^62 * 4 / 1 = 2^64 does not
	// fit, and no in-contract input overflows.
	if _, ok := firstPrice(1<<62, 1, 4, 1); ok {
		t.Fatal("firstPrice overflow reported ok")
	}
}

// TestMarkedDown verifies the estate markdown: the reduction
// trunc(price * md / 10 000) is taken off the price, the result is never below
// the floor, and a price at or below the floor stays.
//
// Arithmetic: 60 * 500 / 10 000 = 30 000 / 10 000 = 3 so 60 goes to 57; 19 * 500 /
// 10 000 = 0.95 truncates to 0 so 19 stays; 1 000 * 500 / 10 000 = 50 gives
// 950; 100 * 10 000 / 10 000 = 100 gives 0, raised to the floor 4; 60 with
// floor 58 and md 500: 60 - 3 = 57, raised to 58.
//
// Requirement: FR-EST-009
func TestMarkedDown(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name            string
		price           Price
		md, floor, want int64
	}{
		{"60 at 500", 60, 500, 4, 57},
		{"reduction truncates to 0", 19, 500, 4, 19},
		{"1000 at 500", 1000, 500, 4, 950},
		{"zero basis points", 60, 0, 4, 60},
		{"full markdown goes to the floor", 100, 10_000, 4, 4},
		{"at the floor stays", 4, 500, 4, 4},
		{"below the floor stays", 3, 500, 4, 3},
		{"floor binds", 60, 500, 58, 58},
		{"largest price", MaxPrice, 10_000, 4, 4},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := markedDown(tc.price, tc.md, tc.floor); int64(got) != tc.want {
				t.Fatalf("markedDown(%d, %d, %d) = %d, want %d", tc.price, tc.md, tc.floor, got, tc.want)
			}
		})
	}
	// An out-of-bound basis point count that overflows is the floor.
	if got := markedDown(MaxPrice, math.MaxInt64, 4); got != 4 {
		t.Fatalf("overflowing markdown = %d, want the floor 4", got)
	}
}

// TestInsertRow verifies that an insert keeps the rows sorted by key and that
// a duplicate key is one row whose existing price is kept.
//
// Requirement: FR-EST-007
func TestInsertRow(t *testing.T) {
	t.Parallel()
	var rows []listing
	for _, in := range []listing{
		{key: k(1, 0, 3), price: 30},
		{key: k(0, 0, 0), price: 10},
		{key: k(1, 0, 6), price: 60},
		{key: k(1, -2, 9), price: 90},
		{key: k(1, 0, 3), price: 99}, // duplicate: ignored
	} {
		rows = insertRow(rows, in)
	}
	want := []listing{
		{key: k(0, 0, 0), price: 10},
		{key: k(1, -2, 9), price: 90},
		{key: k(1, 0, 3), price: 30},
		{key: k(1, 0, 6), price: 60},
	}
	if !slices.Equal(rows, want) {
		t.Fatalf("rows = %v, want %v", rows, want)
	}
	if i, ok := findRow(rows, k(1, 0, 6)); !ok || i != 3 {
		t.Fatalf("findRow = %d, %v; want 3, true", i, ok)
	}
	if i, ok := findRow(rows, k(1, 0, 4)); ok || i != 3 {
		t.Fatalf("findRow of absent = %d, %v; want 3, false", i, ok)
	}
}

// TestReconcile verifies the merge-walk: with lots {a, b, c} and rows {b, d}
// the insert-enabled walk gives {a, b, c} with b's price and mark kept and a, c
// at the first price, and the filter-only walk gives {b}.
//
// Requirement: FR-EST-007, FR-EST-008
func TestReconcile(t *testing.T) {
	t.Parallel()
	a, b, c, d := k(0, 0, 0), k(1, 0, 3), k(1, 0, 6), k(2, 0, 0)
	lots := []ledger.Lot{lotOf(a.Commodity, a.Born, a.Durability), lotOf(b.Commodity, b.Born, b.Durability), lotOf(c.Commodity, c.Born, c.Durability)}
	rows := []listing{{key: b, price: 17, asked: true}, {key: d, price: 5}}
	first := func(l ledger.Lot) Price { return Price(100 + l.Durability) }
	tests := []struct {
		name  string
		first func(ledger.Lot) Price
		want  []listing
	}{
		{"insert-enabled", first, []listing{{key: a, price: 100}, {key: b, price: 17, asked: true}, {key: c, price: 106}}},
		{"filter-only", nil, []listing{{key: b, price: 17, asked: true}}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := reconcile(nil, slices.Clone(rows), lots, tc.first)
			if !slices.Equal(got, tc.want) {
				t.Fatalf("reconcile = %v, want %v", got, tc.want)
			}
		})
	}
	if got := reconcile(nil, rows, nil, first); len(got) != 0 {
		t.Fatalf("no lots left rows %v", got)
	}
}

// TestReconcile_NoAllocation verifies that a walk into a warmed destination
// allocates nothing, in both modes (NFR-EST-001).
//
// Requirement: NFR-EST-001
func TestReconcile_NoAllocation(t *testing.T) {
	lots := []ledger.Lot{lotOf(0, 0, 0), lotOf(1, 0, 3), lotOf(1, 0, 6)}
	rows := []listing{{key: k(1, 0, 3), price: 17}, {key: k(2, 0, 0), price: 5}}
	first := func(l ledger.Lot) Price { return Price(l.Durability + 1) }
	dst := make([]listing, 0, 8)
	for _, f := range []func(ledger.Lot) Price{first, nil} {
		if n := testing.AllocsPerRun(100, func() { dst = reconcile(dst, rows, lots, f) }); n != 0 {
			t.Errorf("reconcile allocated %v times per run, want 0", n)
		}
	}
}

// keysFrom decodes a byte string to a sorted, duplicate-free key set: each
// byte is a commodity (2 bits), a born tick (2 bits) and a durability (3 bits).
func keysFrom(b []byte) []ledger.Key {
	ks := make([]ledger.Key, 0, len(b))
	for _, x := range b {
		ks = append(ks, k(int(x>>6&3), int64(x>>3&3)-1, int64(x&7)))
	}
	slices.SortFunc(ks, cmpKey)
	return slices.CompactFunc(ks, func(a, b ledger.Key) bool { return a == b })
}

// FuzzReconcile checks that for any sorted lots and rows the insert-enabled
// walk gives sorted unique rows over exactly the lots, keeping the price of an
// existing row, and the filter-only walk gives exactly the rows that have a
// lot.
//
// Requirement: FR-EST-007
func FuzzReconcile(f *testing.F) {
	f.Add([]byte{1, 2, 3}, []byte{2, 9})
	f.Add([]byte{}, []byte{5, 6})
	f.Add([]byte{0, 0, 0}, []byte{})
	f.Add([]byte{255, 128, 64, 1}, []byte{64, 1, 200})
	f.Fuzz(func(t *testing.T, lotBytes, rowBytes []byte) {
		var lots []ledger.Lot
		for _, key := range keysFrom(lotBytes) {
			lots = append(lots, lotOf(key.Commodity, key.Born, key.Durability))
		}
		var rows []listing
		for i, key := range keysFrom(rowBytes) {
			rows = append(rows, listing{key: key, price: Price(1000 + i)})
		}
		got := reconcile(nil, rows, lots, func(ledger.Lot) Price { return 1 })
		if len(got) != len(lots) {
			t.Fatalf("insert walk gave %d rows for %d lots", len(got), len(lots))
		}
		for i := range got {
			lk := k(lots[i].Commodity, lots[i].Born, lots[i].Durability)
			if got[i].key != lk {
				t.Fatalf("row %d key %v, lot %v", i, got[i].key, lk)
			}
			want := Price(1)
			if j, ok := findRow(rows, lk); ok {
				want = rows[j].price
			}
			if got[i].price != want {
				t.Fatalf("row %v price %d, want %d", lk, got[i].price, want)
			}
		}
		only := reconcile(nil, rows, lots, nil)
		var wantKeys []ledger.Key
		for _, r := range rows {
			for _, l := range lots {
				if k(l.Commodity, l.Born, l.Durability) == r.key {
					wantKeys = append(wantKeys, r.key)
				}
			}
		}
		var gotKeys []ledger.Key
		for _, r := range only {
			gotKeys = append(gotKeys, r.key)
		}
		if !slices.Equal(gotKeys, wantKeys) {
			t.Fatalf("filter walk keys %v, want %v", gotKeys, wantKeys)
		}
	})
}

// FuzzPriceRules checks that the first price is never below the floor, that a
// markdown never raises a price, never takes it below the floor and leaves a
// price at or below the floor alone, and that neither panics.
//
// Requirement: FR-EST-008, FR-EST-009
func FuzzPriceRules(f *testing.F) {
	f.Add(int64(100), int64(10), int64(3), int64(4), int64(60), int64(500))
	f.Add(int64(MaxPrice), int64(1), int64(1), int64(1), int64(MaxPrice), int64(10_000))
	f.Add(int64(0), int64(0), int64(0), int64(50), int64(3), int64(0))
	f.Add(int64(-5), int64(-1), int64(-1), int64(-1), int64(-1), int64(-1))
	f.Fuzz(func(t *testing.T, ref, durability, left, floor, price, md int64) {
		pos := func(v, n int64) int64 { return int64(uint64(v) % uint64(n)) }
		floor = 1 + pos(floor, 10_000)
		ref = pos(ref, MaxPrice+1)
		durability = pos(durability, 1000)
		left = pos(left, durability+1)
		if p, ok := firstPrice(Price(ref), durability, left, floor); !ok || int64(p) < floor {
			t.Fatalf("firstPrice(%d, %d, %d, %d) = %d, %v", ref, durability, left, floor, p, ok)
		}
		price = 1 + pos(price, MaxPrice)
		md = pos(md, 10_001)
		got := markedDown(Price(price), md, floor)
		switch {
		case price <= floor && int64(got) != price:
			t.Fatalf("price %d at or below floor %d moved to %d", price, floor, got)
		case int64(got) > price || (price > floor && int64(got) < floor):
			t.Fatalf("markedDown(%d, %d, %d) = %d", price, md, floor, got)
		}
	})
}

// TestRefused_NamesTheEstateAndAccount verifies that an overflow of an
// estate's balance is returned wrapping ledger.ErrBalanceOverflow with the
// region and the account in the text, and that any other refusal is a violated
// invariant that names them too.
//
// Guard: with several regions a failed step whose error named neither could
// not say which pool overflowed (observability-and-logging.md 3.2.4).
//
// Requirement: FR-EST-001
func TestRefused_NamesTheEstateAndAccount(t *testing.T) {
	t.Run("overflow is returned", func(t *testing.T) {
		err := refused("move money to the estate", 2, 7, ledger.ErrBalanceOverflow)
		if !errors.Is(err, ledger.ErrBalanceOverflow) {
			t.Fatalf("refused(overflow) = %v, want it to wrap ErrBalanceOverflow", err)
		}
		for _, want := range []string{"region 2", "account 7"} {
			if !strings.Contains(err.Error(), want) {
				t.Errorf("refused(overflow) = %q, want it to contain %q", err, want)
			}
		}
	})
	t.Run("other refusal is an invariant", func(t *testing.T) {
		defer func() {
			s, ok := recover().(string)
			if !ok || !strings.Contains(s, "violated programmer invariant") || !strings.Contains(s, "region 2") || !strings.Contains(s, "account 7") {
				t.Errorf("recover() = %v, want an invariant message naming region 2 and account 7", s)
			}
		}()
		_ = refused("move a lot to the estate", 2, 7, ledger.ErrNotConstruction)
	})
}
