package clearing_test

import (
	"fmt"
	"math"
	"math/rand/v2"
	"slices"
	"strings"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/clearing"
)

const book = 7 // the book's commodity in these tests

func scratchFor(bids, asks []clearing.Offer) clearing.Scratch {
	n := len(bids) + len(asks)
	return clearing.Scratch{Sums: make([]int64, n), Index: make([]int32, n)}
}

// twoSided is the book of the check tests: bids by owners 1 and 2, asks by
// owner 3 (two named lots) and owner 4 (no lot).
func twoSided() (bids, asks []clearing.Offer) {
	bids = []clearing.Offer{
		{Ref: 0, Owner: 1, Price: 12, Qty: 3},
		{Ref: 1, Owner: 2, Price: 10, Qty: 2},
	}
	asks = []clearing.Offer{
		{Ref: 0, Owner: 3, Price: 6, Qty: 2, Lot: clearing.Lot{Named: true, Born: 1, Durability: 5}},
		{Ref: 1, Owner: 3, Price: 9, Qty: 4, Lot: clearing.Lot{Named: true, Born: 2, Durability: 5}},
		{Ref: 2, Owner: 4, Price: 8, Qty: 1},
	}
	return bids, asks
}

func fill(buyer, seller int64, lot clearing.Lot, price, qty int64) clearing.Fill {
	return clearing.Fill{Commodity: book, Buyer: buyer, Seller: seller, Lot: lot, Price: price, Qty: qty}
}

var (
	lotA = clearing.Lot{Named: true, Born: 1, Durability: 5}
	lotB = clearing.Lot{Named: true, Born: 2, Durability: 5}
)

// TestCheck_Cases breaks exactly one rule of CLR 8.2 per case, and runs valid
// lists: in any order, summing to exactly an order's quantity, empty.
//
// Requirement: FR-CLR-024, FR-CLR-018, FR-CLR-014, FR-CLR-029
func TestCheck_Cases(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		fills []clearing.Fill
		want  clearing.Code
		refs  [][2]int // BidRef, AskRef of each match when valid
	}{
		{"empty list", nil, 0, nil},
		{"valid", []clearing.Fill{fill(1, 3, lotA, 9, 2)}, 0, [][2]int{{0, 0}}},
		{"named lot resolves to its own ask", []clearing.Fill{fill(1, 3, lotB, 10, 1)}, 0, [][2]int{{0, 1}}},
		{"any order of fills", []clearing.Fill{fill(2, 4, clearing.Lot{}, 9, 1), fill(1, 3, lotA, 8, 2)}, 0,
			[][2]int{{1, 2}, {0, 0}}},
		{"several fills sum to exactly the quantity", []clearing.Fill{fill(1, 3, lotB, 9, 1), fill(1, 3, lotB, 9, 2)}, 0,
			[][2]int{{0, 1}, {0, 1}}},
		{"summed one over", []clearing.Fill{fill(1, 3, lotB, 9, 2), fill(1, 3, lotB, 9, 2)}, clearing.OverFilled, nil},
		{"another commodity", []clearing.Fill{{Commodity: book + 1, Buyer: 1, Seller: 4, Price: 9, Qty: 1}}, clearing.BadCommodity, nil},
		{"no bid", []clearing.Fill{fill(9, 4, clearing.Lot{}, 9, 1)}, clearing.NoOrder, nil},
		{"no ask", []clearing.Fill{fill(1, 9, clearing.Lot{}, 9, 1)}, clearing.NoOrder, nil},
		{"ask exists but not of that lot", []clearing.Fill{fill(1, 3, clearing.Lot{}, 9, 1)}, clearing.NoOrder, nil},
		{"quantity 0", []clearing.Fill{fill(1, 4, clearing.Lot{}, 9, 0)}, clearing.BadQuantity, nil},
		{"quantity negative", []clearing.Fill{fill(1, 4, clearing.Lot{}, 9, -1)}, clearing.BadQuantity, nil},
		{"price below ask limit", []clearing.Fill{fill(1, 4, clearing.Lot{}, 7, 1)}, clearing.BadPrice, nil},
		{"price above bid limit", []clearing.Fill{fill(2, 4, clearing.Lot{}, 11, 1)}, clearing.BadPrice, nil},
		{"price at the ask limit and at the bid limit is valid", []clearing.Fill{fill(1, 3, lotA, 6, 1), fill(2, 4, clearing.Lot{}, 10, 1)}, 0,
			[][2]int{{0, 0}, {1, 2}}},
		{"two fills at their limits over one ask's quantity", []clearing.Fill{fill(2, 4, clearing.Lot{}, 10, 1), fill(1, 4, clearing.Lot{}, 8, 1)}, clearing.OverFilled, nil},
		{"qty overflow of a sum", []clearing.Fill{fill(1, 4, clearing.Lot{}, 9, 1<<62), fill(1, 4, clearing.Lot{}, 9, 1<<62)}, clearing.OverFilled, nil},
		{"first failure of a fill wins", []clearing.Fill{{Commodity: book + 1, Buyer: 9, Seller: 9, Qty: 0}}, clearing.BadCommodity, nil},
		{"lowest code over the list", []clearing.Fill{fill(1, 4, clearing.Lot{}, 7, 1), fill(9, 4, clearing.Lot{}, 9, 1)}, clearing.NoOrder, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			bids, asks := twoSided()
			sc := scratchFor(bids, asks)
			pre := []clearing.Match{{Price: 99}}
			got, code := clearing.Check(slices.Clone(pre), sc, book, bids, asks, tt.fills)
			if code != tt.want {
				t.Fatalf("Check(%d bids, %d asks, %+v) code = %d, want %d", len(bids), len(asks), tt.fills, code, tt.want)
			}
			if code != 0 {
				if !slices.Equal(got, pre) {
					t.Errorf("Check(%+v) dst = %v, want it unchanged %v", tt.fills, got, pre)
				}
				return
			}
			if len(got) != 1+len(tt.refs) {
				t.Fatalf("Check(%+v) len(dst) = %d, want %d", tt.fills, len(got), 1+len(tt.refs))
			}
			for i, r := range tt.refs {
				if m := got[1+i]; m.BidRef != r[0] || m.AskRef != r[1] || m.Price != tt.fills[i].Price || m.Qty != tt.fills[i].Qty {
					t.Errorf("Check(%+v) match %d = %+v, want refs %v of fill %+v", tt.fills, i, m, r, tt.fills[i])
				}
			}
			// Independent oracle: the per-order sum of the fills, by the ref
			// of each match (Ref equals position in twoSided), bids first.
			want := make([]int64, len(bids)+len(asks))
			for i, r := range tt.refs {
				want[r[0]] += tt.fills[i].Qty
				want[len(bids)+r[1]] += tt.fills[i].Qty
			}
			if !slices.Equal(sc.Sums, want) {
				t.Errorf("Check(%+v) Sums = %v, want %v", tt.fills, sc.Sums, want)
			}
		})
	}
}

// TestCheck_LotKey asserts that the lot half of the index key is complete: asks
// of one owner that differ only in Named, or only in Durability, each resolve
// from a fill of exactly that lot.
//
// Requirement: FR-CLR-024
func TestCheck_LotKey(t *testing.T) {
	t.Parallel()
	lots := []clearing.Lot{
		{Named: true, Born: 0, Durability: 2},
		{Named: true, Born: 0, Durability: 0},
		{},
		{Named: true, Born: 0, Durability: 1},
	}
	var asks []clearing.Offer
	for i, l := range lots {
		asks = append(asks, clearing.Offer{Ref: i, Owner: 3, Lot: l, Price: 5, Qty: 1})
	}
	bids := []clearing.Offer{{Ref: 0, Owner: 1, Price: 9, Qty: 4}}
	for i, l := range lots {
		t.Run(fmt.Sprintf("lot %d", i), func(t *testing.T) {
			t.Parallel()
			got, code := clearing.Check(nil, scratchFor(bids, asks), book, bids, asks,
				[]clearing.Fill{fill(1, 3, l, 6, 1)})
			if code != 0 || len(got) != 1 || got[0].AskRef != i {
				t.Errorf("Check(fill of lot %+v) = (%v, %d), want one match with AskRef %d", l, got, code, i)
			}
		})
	}
}

// TestCheck_SameOwner asserts that an owner holding both a bid and an ask
// (the treasury case of CLR 3.1.2) cannot be matched with itself, and that the
// same fill passes with two owners. Prices at both limits are valid.
//
// Requirement: FR-CLR-018, FR-CLR-014
func TestCheck_SameOwner(t *testing.T) {
	t.Parallel()
	bids := []clearing.Offer{{Ref: 0, Owner: 1, Price: 10, Qty: 1}, {Ref: 1, Owner: 2, Price: 10, Qty: 1}}
	asks := []clearing.Offer{{Ref: 0, Owner: 1, Price: 5, Qty: 1}}
	if _, code := clearing.Check(nil, scratchFor(bids, asks), book, bids, asks,
		[]clearing.Fill{fill(1, 1, clearing.Lot{}, 7, 1)}); code != clearing.SameOwner {
		t.Errorf("Check(buyer 1 and seller 1) code = %d, want %d", code, clearing.SameOwner)
	}
	got, code := clearing.Check(nil, scratchFor(bids, asks), book, bids, asks,
		[]clearing.Fill{fill(2, 1, clearing.Lot{}, 5, 1)})
	if code != 0 || len(got) != 1 {
		t.Errorf("Check(buyer 2, seller 1) = (%v, %d), want one match and 0", got, code)
	}
}

// TestCheck_DirtyScratch asserts that Check zeroes Sums itself: a scratch
// pre-filled with garbage, then reused across a second call, yields the sums
// of the list alone.
//
// Requirement: FR-CLR-024
func TestCheck_DirtyScratch(t *testing.T) {
	t.Parallel()
	bids, asks := twoSided()
	sc := scratchFor(bids, asks)
	for i := range sc.Sums {
		sc.Sums[i] = 99
	}
	// Per-order sums: bid 0 gets 2; ask 0 gets 2; the rest 0.
	want := []int64{2, 0, 2, 0, 0}
	for call := 0; call < 2; call++ {
		_, code := clearing.Check(nil, sc, book, bids, asks, []clearing.Fill{fill(1, 3, lotA, 9, 2)})
		if code != 0 {
			t.Fatalf("call %d: Check(fill of lotA) code = %d, want 0", call, code)
		}
		if !slices.Equal(sc.Sums, want) {
			t.Errorf("call %d: Check(fill of lotA) Sums = %v, want %v", call, sc.Sums, want)
		}
	}
}

// TestCheck_Allocs asserts that Check allocates nothing with a warmed scratch,
// for a valid list and an invalid one.
//
// Requirement: FR-CLR-024
func TestCheck_Allocs(t *testing.T) {
	bids, asks := twoSided()
	sc := scratchFor(bids, asks)
	dst := make([]clearing.Match, 0, 8)
	valid := []clearing.Fill{fill(1, 3, lotA, 9, 2), fill(2, 4, clearing.Lot{}, 9, 1)}
	invalid := []clearing.Fill{fill(1, 3, lotA, 9, 2), fill(1, 3, lotA, 9, 2)}
	for name, fills := range map[string][]clearing.Fill{"valid": valid, "invalid": invalid} {
		if n := testing.AllocsPerRun(100, func() { clearing.Check(dst[:0], sc, book, bids, asks, fills) }); n != 0 {
			t.Errorf("Check(%s list) = %v allocations, want 0", name, n)
		}
	}
}

// TestCheck_ShortScratch asserts the documented panic for a scratch shorter
// than the book.
//
// Requirement: FR-CLR-024
//
// Guard: a violated caller invariant must be loud, not an index error later.
func TestCheck_ShortScratch(t *testing.T) {
	t.Parallel()
	defer func() {
		if v, _ := recover().(string); !strings.Contains(v, "violated programmer invariant") {
			t.Errorf("Check with a scratch shorter than the book recovered %q, want a violated programmer invariant message", v)
		}
	}()
	bids, asks := twoSided()
	clearing.Check(nil, clearing.Scratch{}, book, bids, asks, nil)
}

// BenchmarkCheck reports the growth of Check over books of 10 to 10 000
// orders, one fill per bid. It gates nothing.
func BenchmarkCheck(b *testing.B) {
	for _, n := range []int{10, 100, 1000, 10000} {
		bids := make([]clearing.Offer, n/2)
		asks := make([]clearing.Offer, n/2)
		fills := make([]clearing.Fill, n/2)
		for i := range bids {
			bids[i] = clearing.Offer{Ref: i, Owner: int64(i), Price: 10, Qty: 1}
			asks[i] = clearing.Offer{Ref: i, Owner: int64(n + i), Price: 5, Qty: 1}
			fills[i] = fill(int64(i), int64(n+i), clearing.Lot{}, 7, 1)
		}
		sc := scratchFor(bids, asks)
		dst := make([]clearing.Match, 0, len(fills))
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				clearing.Check(dst[:0], sc, book, bids, asks, fills)
			}
		})
	}
}

// decodeBook builds a book and a list of fills from bytes: one order per
// (owner, lot, side), fills drawn from small domains that include values outside
// the book.
func decodeBook(data []byte) (bids, asks []clearing.Offer, fills []clearing.Fill) {
	if len(data) == 0 {
		return
	}
	nb, na := int(data[0]>>4)%5, int(data[0]&15)%5
	data = data[1:]
	next := func() int {
		if len(data) == 0 {
			return 0
		}
		v := int(data[0])
		data = data[1:]
		return v
	}
	lotOf := func(v int) clearing.Lot {
		if v%3 == 0 {
			return clearing.Lot{}
		}
		return clearing.Lot{Named: true, Born: int64(v % 3), Durability: 1}
	}
	seenB, seenA := map[int64]bool{}, map[[2]int64]bool{}
	for i := 0; i < nb; i++ {
		o, p, q := int64(next()%6), int64(next()%20), int64(next()%6)
		if !seenB[o] {
			seenB[o] = true
			bids = append(bids, clearing.Offer{Ref: len(bids), Owner: o, Price: p, Qty: q})
		}
	}
	for i := 0; i < na; i++ {
		o, l, p, q := int64(next()%6), lotOf(next()), int64(next()%20), int64(next()%6)
		k := [2]int64{o, l.Born}
		if !seenA[k] {
			seenA[k] = true
			asks = append(asks, clearing.Offer{Ref: len(asks), Owner: o, Lot: l, Price: p, Qty: q})
		}
	}
	for len(data) >= 6 {
		c := int64(book)
		if data[0]%8 == 0 {
			c++
		}
		q := int64(data[5]%7) - 1
		if data[5] == 255 {
			q = 1 << 62
		}
		fills = append(fills, clearing.Fill{Commodity: c, Buyer: int64(data[1] % 7), Seller: int64(data[2] % 7),
			Lot: lotOf(int(data[3])), Price: int64(data[4] % 22), Qty: q})
		data = data[6:]
	}
	return
}

// refLookup finds a fill's bid and ask by linear scan (nil when absent).
func refLookup(bids, asks []clearing.Offer, f clearing.Fill) (b, a *clearing.Offer) {
	for i := range bids {
		if bids[i].Owner == f.Buyer && bids[i].Lot == (clearing.Lot{}) {
			b = &bids[i]
		}
	}
	for i := range asks {
		if asks[i].Owner == f.Seller && asks[i].Lot == f.Lot {
			a = &asks[i]
		}
	}
	return b, a
}

// refFault is the first fault of one fill given its looked-up orders.
func refFault(f clearing.Fill, b, a *clearing.Offer) clearing.Code {
	switch {
	case f.Commodity != book:
		return clearing.BadCommodity
	case b == nil || a == nil:
		return clearing.NoOrder
	case f.Qty < 1:
		return clearing.BadQuantity
	case f.Price < a.Price || f.Price > b.Price:
		return clearing.BadPrice
	case f.Buyer == f.Seller:
		return clearing.SameOwner
	}
	return 0
}

// refCheck is the independent plain-loop restatement of CLR 8.2: linear
// lookups and a map of sums, the lowest failure code over the list.
func refCheck(bids, asks []clearing.Offer, fills []clearing.Fill) (clearing.Code, int) {
	var worst clearing.Code
	type key struct {
		side int
		ref  int
	}
	sums := map[key]int64{}
	over := false
	n := 0
	for _, f := range fills {
		b, a := refLookup(bids, asks, f)
		c := refFault(f, b, a)
		if c != 0 {
			if worst == 0 || c < worst {
				worst = c
			}
			continue
		}
		sums[key{0, b.Ref}] += f.Qty
		sums[key{1, a.Ref}] += f.Qty
		if sums[key{0, b.Ref}] > b.Qty || sums[key{1, a.Ref}] > a.Qty {
			over = true
		}
		n++
	}
	if worst == 0 && over {
		worst = clearing.OverFilled
	}
	return worst, n
}

// FuzzCheck decodes a small book and an arbitrary list of fills and asserts
// that Check never panics, agrees with an independent loop on accept and on the
// failure code, leaves dst's length on a failure, and does not depend on the
// order of the fills.
//
// Requirement: FR-CLR-024, FR-CLR-014, FR-CLR-018
//
// Fix: the corpus files testdata/fuzz/FuzzCheck/seed-* pin three hand-written
// cases, not crashes (the same inputs are given with f.Add): seed-valid, a
// list that passes; seed-overfilled, two fills summing above an order's
// quantity; seed-huge-qty, a quantity of 2^62 whose sum must not wrap.
func FuzzCheck(f *testing.F) {
	f.Add([]byte{0x22, 1, 12, 3, 2, 10, 2, 3, 0, 6, 2, 4, 0, 8, 1, 1, 3, 0, 9, 2})
	f.Add([]byte{0x11, 1, 12, 3, 2, 0, 6, 2, 1, 1, 0, 9, 2, 1, 1, 0, 9, 3})
	f.Add([]byte{0x11, 1, 12, 3, 1, 0, 6, 2, 1, 1, 0, 9, 255})
	f.Add([]byte{0x11, 1, 12, 3, 1, 0, 6, 2, 8, 1, 1, 0, 9, 1, 1, 1, 1, 0, 9, 1})
	f.Fuzz(func(t *testing.T, data []byte) {
		bids, asks, fills := decodeBook(data)
		wantCode, wantN := refCheck(bids, asks, fills)
		sc := scratchFor(bids, asks)
		pre := []clearing.Match{{Price: 1}}
		got, code := clearing.Check(slices.Clone(pre), sc, book, bids, asks, fills)
		if code != wantCode {
			t.Fatalf("Check(%+v, %+v, %+v) code = %d, want %d", bids, asks, fills, code, wantCode)
		}
		if code != 0 && len(got) != 1 {
			t.Errorf("Check(%+v, %+v, %+v) len(dst) = %d after a failure, want 1", bids, asks, fills, len(got))
		}
		if code == 0 && len(got) != 1+wantN {
			t.Errorf("Check(%+v, %+v, %+v) len(dst) = %d, want %d", bids, asks, fills, len(got), 1+wantN)
		}
		rev := slices.Clone(fills)
		slices.Reverse(rev)
		_, code2 := clearing.Check(nil, scratchFor(bids, asks), book, bids, asks, rev)
		if code2 != code {
			t.Errorf("Check(%+v, %+v, reversed %+v) code = %d, want %d", bids, asks, rev, code2, code)
		}
	})
}

// TestCheck_IndexOrder asserts that every order resolves from its own fill
// however the book is arranged: asks of two owners that differ in Named, in
// Born and in Durability are listed in many orders, so the sorted index and
// its binary search must agree on one total order of (owner, lot).
//
// Requirement: FR-CLR-024, FR-CLR-029
//
// Guard: a comparator that ignores Born or Durability would still pass a book
// whose lots differ in one field only; a fill would then resolve to the wrong
// ask, or to none.
func TestCheck_IndexOrder(t *testing.T) {
	t.Parallel()
	lots := []clearing.Lot{{}, {Named: true, Born: 1, Durability: 5}, {Named: true, Born: 1, Durability: 9}, {Named: true, Born: 2, Durability: 1}}
	var base []clearing.Offer
	for _, owner := range []int64{3, 4} {
		for _, l := range lots {
			base = append(base, clearing.Offer{Owner: owner, Lot: l, Price: 5, Qty: 1})
		}
	}
	for seed := range 50 {
		perm := rand.New(rand.NewPCG(uint64(seed), 1)).Perm(len(base))
		asks := make([]clearing.Offer, len(base))
		for i, j := range perm {
			asks[i] = base[j]
			asks[i].Ref = i
		}
		bids := []clearing.Offer{{Ref: 0, Owner: 1, Price: 9, Qty: int64(len(base))}}
		fills := make([]clearing.Fill, len(asks))
		for i, a := range asks {
			fills[i] = fill(1, a.Owner, a.Lot, 6, 1)
		}
		got, code := clearing.Check(nil, scratchFor(bids, asks), book, bids, asks, fills)
		if code != 0 || len(got) != len(asks) {
			t.Fatalf("seed %d: Check(%d asks, one fill each) = (%d matches, code %d), want all matches and code 0", seed, len(asks), len(got), code)
		}
		for i, m := range got {
			if m.AskRef != i {
				t.Fatalf("seed %d: fill %d resolved to AskRef %d, want %d", seed, i, m.AskRef, i)
			}
		}
	}
}

// TestCheck_FillLimits asserts the failure code for the boundaries of a fill:
// another commodity just below the book's, an over-fill seen only by the bid or
// only by the ask, a sum that does not fit int64 on either side, the lowest
// code winning over an over-fill, and a scratch longer than the book.
//
// Requirement: FR-CLR-024, FR-CLR-014, FR-CLR-018
func TestCheck_FillLimits(t *testing.T) {
	t.Parallel()
	const huge = math.MaxInt64
	one := func(bq, aq int64) ([]clearing.Offer, []clearing.Offer) {
		return []clearing.Offer{{Ref: 0, Owner: 1, Price: 9, Qty: bq}}, []clearing.Offer{{Ref: 0, Owner: 3, Price: 5, Qty: aq}}
	}
	twoAsks := func() ([]clearing.Offer, []clearing.Offer) {
		return []clearing.Offer{{Ref: 0, Owner: 1, Price: 9, Qty: huge}},
			[]clearing.Offer{{Ref: 0, Owner: 3, Price: 5, Qty: huge}, {Ref: 1, Owner: 4, Price: 5, Qty: 5}}
	}
	twoBids := func() ([]clearing.Offer, []clearing.Offer) {
		return []clearing.Offer{{Ref: 0, Owner: 1, Price: 9, Qty: huge}, {Ref: 1, Owner: 2, Price: 9, Qty: 5}},
			[]clearing.Offer{{Ref: 0, Owner: 3, Price: 5, Qty: huge}}
	}
	type book2 = func() ([]clearing.Offer, []clearing.Offer)
	tests := []struct {
		name  string
		mk    book2
		fills []clearing.Fill
		want  clearing.Code
	}{
		{"commodity below the book's", func() ([]clearing.Offer, []clearing.Offer) { return one(5, 5) },
			[]clearing.Fill{{Commodity: book - 1, Buyer: 1, Seller: 3, Price: 6, Qty: 1}}, clearing.BadCommodity},
		// The bid holds 5 and the ask 10, so 6 units pass the ask and break the bid.
		{"over-fill seen by the bid only", func() ([]clearing.Offer, []clearing.Offer) { return one(5, 10) },
			[]clearing.Fill{fill(1, 3, clearing.Lot{}, 6, 6)}, clearing.OverFilled},
		{"over-fill seen by the ask only", func() ([]clearing.Offer, []clearing.Offer) { return one(10, 5) },
			[]clearing.Fill{fill(1, 3, clearing.Lot{}, 6, 6)}, clearing.OverFilled},
		// MaxInt64 units then one more on the same bid: the bid's sum wraps.
		{"bid sum does not fit int64", twoAsks,
			[]clearing.Fill{fill(1, 3, clearing.Lot{}, 6, huge), fill(1, 4, clearing.Lot{}, 6, 1)}, clearing.OverFilled},
		{"ask sum does not fit int64", twoBids,
			[]clearing.Fill{fill(1, 3, clearing.Lot{}, 6, huge), fill(2, 3, clearing.Lot{}, 6, 1)}, clearing.OverFilled},
		// BadPrice (4) is below OverFilled (6), so it wins whatever the order.
		{"lowest code wins over an over-fill", func() ([]clearing.Offer, []clearing.Offer) { return one(5, 10) },
			[]clearing.Fill{fill(1, 3, clearing.Lot{}, 6, 6), fill(1, 3, clearing.Lot{}, 99, 1)}, clearing.BadPrice},
		{"exactly the quantity passes", func() ([]clearing.Offer, []clearing.Offer) { return one(5, 5) },
			[]clearing.Fill{fill(1, 3, clearing.Lot{}, 6, 5)}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			bids, asks := tt.mk()
			sc := scratchFor(bids, asks)
			// A longer scratch than the book needs must be accepted.
			sc.Sums, sc.Index = append(sc.Sums, 0, 0), append(sc.Index, 0, 0)
			if _, got := clearing.Check(nil, sc, book, bids, asks, tt.fills); got != tt.want {
				t.Errorf("Check(%v, %v, %v) code = %d, want %d", bids, asks, tt.fills, got, tt.want)
			}
		})
	}
}
