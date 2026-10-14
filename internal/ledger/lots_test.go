package ledger_test

import (
	"errors"
	"fmt"
	"math"
	"reflect"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/ledger"
)

// The commodities of the goods fixture.
const (
	cSpoil = 0 // shelf life 3: a lot made in tick t spoils at the end of t+2
	cWear  = 1 // durability 5
	cPlain = 2 // neither
)

// gfix is a ledger for the goods rules, in its construction phase: three
// commodities, two regions, one estate and the boundary accounts of the
// reasons the tests use.
type gfix struct {
	l *ledger.Ledger
	// a0 is an agent of region 0, a1 an agent of region 1, e0 an estate of
	// region 0.
	a0, a1, e0 int
	initial    [3]int // boundary of Initial, per commodity
	disposal   [3]int // boundary of Disposal, per commodity
	spoilage0  int    // boundary of Spoilage for cSpoil
}

const goodsMaxQuantity = 100

func newGoods(t *testing.T, maxQuantity int64) *gfix {
	t.Helper()
	l, err := ledger.New(ledger.Config{
		Commodities: []ledger.Commodity{{ShelfLife: 3}, {Durability: 5}, {}},
		Regions:     2,
		MaxMoney:    1000,
		MaxQuantity: maxQuantity,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	f := &gfix{l: l}
	must := func(id int, err error) int {
		if err != nil {
			t.Fatalf("setup: %v", err)
		}
		return id
	}
	for c := 0; c < 3; c++ {
		f.initial[c] = must(l.OpenBoundary(ledger.Initial, c))
		f.disposal[c] = must(l.OpenBoundary(ledger.Disposal, c))
	}
	f.spoilage0 = must(l.OpenBoundary(ledger.Spoilage, cSpoil))
	f.a0 = must(l.Open(ledger.KindAgent, 0, 0))
	f.a1 = must(l.Open(ledger.KindAgent, 1, 1))
	f.e0 = must(l.Open(ledger.KindEstate, 0, 0))
	return f
}

// stated creates qty units under a stated key, in construction.
func (f *gfix) stated(t *testing.T, to, c int, born, dur, qty int64) {
	t.Helper()
	err := f.l.MoveStated(f.initial[c], to, ledger.Key{Commodity: c, Born: born, Durability: dur}, qty, ledger.Initial)
	if err != nil {
		t.Fatalf("setup: stated lot c%d born %d dur %d x%d: %v", c, born, dur, qty, err)
	}
}

// create creates qty units under the defaults of 5.2.3.
func (f *gfix) create(t *testing.T, to, c int, qty int64) {
	t.Helper()
	if err := f.l.Move(f.initial[c], to, c, qty, ledger.Initial); err != nil {
		t.Fatalf("setup: create c%d x%d: %v", c, qty, err)
	}
}

// holdings builds the standard holdings of the tables, in construction:
//
//	a0: cSpoil (born -2: 4), (born -1: 3), (born 0: 5)
//	    cWear  (dur 2: 3), (dur 5: 6)
//	    cPlain 7
func (f *gfix) holdings(t *testing.T) {
	t.Helper()
	f.stated(t, f.a0, cSpoil, -2, 0, 4)
	f.stated(t, f.a0, cSpoil, -1, 0, 3)
	f.stated(t, f.a0, cSpoil, 0, 0, 5)
	f.stated(t, f.a0, cWear, 0, 2, 3)
	f.stated(t, f.a0, cWear, 0, 5, 6)
	f.create(t, f.a0, cPlain, 7)
}

func (f *gfix) endConstruction(t *testing.T) {
	t.Helper()
	if err := f.l.EndConstruction(); err != nil {
		t.Fatalf("setup: EndConstruction: %v", err)
	}
}

// lots returns a copy of the lots of an account.
func (f *gfix) lots(t *testing.T, id int) []ledger.Lot {
	t.Helper()
	got, err := f.l.AppendLots(nil, id)
	if err != nil {
		t.Fatalf("AppendLots(%d): %v", id, err)
	}
	return got
}

// gstate is everything observable about a goods ledger, for comparing a
// ledger with its earlier self.
type gstate struct {
	Accounts       []ledger.View
	Lots           [][]ledger.Lot
	Totals, Bucket []ledger.Flow
	Construction   bool
	Tick           int64
}

func (f *gfix) state() gstate {
	s := gstate{
		Totals:       f.l.AppendFlows(nil, false),
		Bucket:       f.l.AppendFlows(nil, true),
		Construction: f.l.InConstruction(),
		Tick:         f.l.Tick(),
	}
	for id := 0; id < f.l.Len(); id++ {
		v, err := f.l.Account(id)
		if err != nil {
			panic(err)
		}
		lots, err := f.l.AppendLots(nil, id)
		if err != nil {
			panic(err)
		}
		s.Accounts = append(s.Accounts, v)
		s.Lots = append(s.Lots, lots)
	}
	return s
}

// problems lists what is wrong with the goods of a ledger: a commodity whose
// total over all accounts, boundaries included, is not 0 (FR-LED-002), a
// negative or oversized non-boundary holding (FR-LED-003) and a holding that
// is not in canonical form (FR-LED-010).
func (f *gfix) problems() []string {
	var out []string
	boundaries := [][]int{}
	for c := 0; c < 3; c++ {
		boundaries = append(boundaries, []int{f.initial[c], f.disposal[c]})
	}
	boundaries[cSpoil] = append(boundaries[cSpoil], f.spoilage0)
	isBoundary := map[int]bool{}
	for c, ids := range boundaries {
		var total int64
		for _, id := range ids {
			v, _ := f.l.Account(id)
			total += v.Balance
			isBoundary[id] = true
		}
		for id := 0; id < f.l.Len(); id++ {
			if isBoundary[id] {
				continue
			}
			lots, _ := f.l.AppendLots(nil, id)
			for _, lt := range lots {
				if lt.Commodity == c {
					total += lt.Quantity
				}
			}
		}
		if total != 0 {
			out = append(out, fmt.Sprintf("commodity %d totals %d, want 0", c, total))
		}
	}
	for id := 0; id < f.l.Len(); id++ {
		lots, _ := f.l.AppendLots(nil, id)
		for i, lt := range lots {
			switch {
			case lt.Quantity < 1 || lt.Reserved < 0 || lt.Reserved > lt.Quantity:
				out = append(out, fmt.Sprintf("account %d lot %v: bad quantity or earmark", id, lt))
			case i > 0 && !lessKey(lots[i-1], lt):
				out = append(out, fmt.Sprintf("account %d lots %v, %v are not in canonical order", id, lots[i-1], lt))
			}
		}
	}
	return out
}

func lessKey(a, b ledger.Lot) bool {
	if a.Commodity != b.Commodity {
		return a.Commodity < b.Commodity
	}
	if a.Born != b.Born {
		return a.Born < b.Born
	}
	return a.Durability < b.Durability
}

// TestMove_TakesOldestThenMostWornAndKeepsTheKey verifies that a move takes
// lots oldest first and then lowest remaining durability, that the moved lots
// keep their born tick and durability, that a lot is split and one that is
// emptied disappears, and that an unspoiling, unwearing commodity stays one
// lot.
//
// Requirement: FR-LED-010, FR-LED-011, FR-LED-012, FR-LED-002
func TestMove_TakesOldestThenMostWornAndKeepsTheKey(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		commodity int
		qty       int64
		from, to  []ledger.Lot
	}{
		{
			// Oldest first: born -2 holds 4, so 5 takes all 4 of it and 1 of
			// born -1 (3 -> 2). born 0 is untouched.
			name: "oldest first across lots, emptying one and splitting the next", commodity: cSpoil, qty: 5,
			from: []ledger.Lot{
				{Commodity: cSpoil, Quantity: 2, Born: -1},
				{Commodity: cSpoil, Quantity: 5, Born: 0},
				{Commodity: cWear, Quantity: 3, Durability: 2},
				{Commodity: cWear, Quantity: 6, Durability: 5},
				{Commodity: cPlain, Quantity: 7},
			},
			to: []ledger.Lot{
				{Commodity: cSpoil, Quantity: 4, Born: -2},
				{Commodity: cSpoil, Quantity: 1, Born: -1},
			},
		},
		{
			// Most worn first: durability 2 holds 3, so 4 takes all 3 and 1 of
			// durability 5 (6 -> 5). The items keep their durability.
			name: "most worn first, wear travels", commodity: cWear, qty: 4,
			from: []ledger.Lot{
				{Commodity: cSpoil, Quantity: 4, Born: -2},
				{Commodity: cSpoil, Quantity: 3, Born: -1},
				{Commodity: cSpoil, Quantity: 5, Born: 0},
				{Commodity: cWear, Quantity: 5, Durability: 5},
				{Commodity: cPlain, Quantity: 7},
			},
			to: []ledger.Lot{
				{Commodity: cWear, Quantity: 3, Durability: 2},
				{Commodity: cWear, Quantity: 1, Durability: 5},
			},
		},
		{
			// A commodity that neither spoils nor wears is one lot, so 3 of 7
			// leaves 4 in one lot and arrives as one lot of 3.
			name: "an unspoiling unwearing commodity is one lot", commodity: cPlain, qty: 3,
			from: []ledger.Lot{
				{Commodity: cSpoil, Quantity: 4, Born: -2},
				{Commodity: cSpoil, Quantity: 3, Born: -1},
				{Commodity: cSpoil, Quantity: 5, Born: 0},
				{Commodity: cWear, Quantity: 3, Durability: 2},
				{Commodity: cWear, Quantity: 6, Durability: 5},
				{Commodity: cPlain, Quantity: 4},
			},
			to: []ledger.Lot{{Commodity: cPlain, Quantity: 3}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := newGoods(t, goodsMaxQuantity)
			f.holdings(t)
			if err := f.l.Move(f.a0, f.a1, tt.commodity, tt.qty, ledger.Trade); err != nil {
				t.Fatalf("Move(a0, a1, c%d, %d, Trade) = %v, want nil", tt.commodity, tt.qty, err)
			}
			if got := f.lots(t, f.a0); !reflect.DeepEqual(got, tt.from) {
				t.Errorf("source lots = %v, want %v", got, tt.from)
			}
			if got := f.lots(t, f.a1); !reflect.DeepEqual(got, tt.to) {
				t.Errorf("target lots = %v, want %v", got, tt.to)
			}
			if p := f.problems(); len(p) > 0 {
				t.Errorf("after Move: %v", p)
			}
		})
	}
}

// TestMove_MergesIntoAnEqualKey verifies that a move onto an account that
// already holds the key adds to that lot and does not make a second one.
//
// Requirement: FR-LED-010
func TestMove_MergesIntoAnEqualKey(t *testing.T) {
	t.Parallel()
	f := newGoods(t, goodsMaxQuantity)
	f.holdings(t)
	f.stated(t, f.a1, cSpoil, 0, 0, 2) // a1 holds born 0 x2
	// a0 holds born -2 x4, born -1 x3, born 0 x5; 8 takes -2 (4), -1 (3), 1 of
	// born 0, which joins the 2 already in a1: 2+1 = 3.
	if err := f.l.Move(f.a0, f.a1, cSpoil, 8, ledger.Trade); err != nil {
		t.Fatalf("Move: %v", err)
	}
	want := []ledger.Lot{
		{Commodity: cSpoil, Quantity: 4, Born: -2},
		{Commodity: cSpoil, Quantity: 3, Born: -1},
		{Commodity: cSpoil, Quantity: 3, Born: 0},
	}
	if got := f.lots(t, f.a1); !reflect.DeepEqual(got, want) {
		t.Errorf("target lots = %v, want %v", got, want)
	}
}

// TestCanonicalForm_SameHoldingsSameRepresentation verifies that the same
// holdings reached by different sequences of moves have an identical
// representation.
//
// Requirement: FR-LED-010, FR-LED-011
func TestCanonicalForm_SameHoldingsSameRepresentation(t *testing.T) {
	t.Parallel()
	// Direct: the holdings are created in one order, with the lot of born 0
	// made of two creations.
	direct := newGoods(t, goodsMaxQuantity)
	direct.create(t, direct.a0, cPlain, 7)
	direct.stated(t, direct.a0, cWear, 0, 5, 4)
	direct.stated(t, direct.a0, cSpoil, 0, 0, 2)
	direct.stated(t, direct.a0, cSpoil, 0, 0, 3)
	direct.stated(t, direct.a0, cSpoil, -1, 0, 3)

	// Moved: the same holdings are created in a1, in the reverse order, then
	// carried to a0 one unit and several units at a time.
	moved := newGoods(t, goodsMaxQuantity)
	moved.stated(t, moved.a1, cSpoil, -1, 0, 3)
	moved.stated(t, moved.a1, cSpoil, 0, 0, 5)
	moved.stated(t, moved.a1, cWear, 0, 5, 4)
	moved.create(t, moved.a1, cPlain, 7)
	for _, step := range []struct {
		c   int
		qty int64
	}{{cPlain, 1}, {cSpoil, 1}, {cWear, 3}, {cPlain, 6}, {cSpoil, 7}, {cWear, 1}} {
		if err := moved.l.Move(moved.a1, moved.a0, step.c, step.qty, ledger.Trade); err != nil {
			t.Fatalf("Move(c%d, %d): %v", step.c, step.qty, err)
		}
	}
	if a, b := direct.lots(t, direct.a0), moved.lots(t, moved.a0); !reflect.DeepEqual(a, b) {
		t.Errorf("lots reached directly %v, by moves %v, want equal", a, b)
	}
	if got := len(moved.lots(t, moved.a0)); got != 4 {
		// born -1 and born 0 of cSpoil, one cWear lot, one cPlain lot.
		t.Errorf("a0 has %d lots, want 4", got)
	}
	if got := moved.lots(t, moved.a1); len(got) != 0 {
		t.Errorf("a1 lots = %v, want none", got)
	}
}

// TestCreation_DefaultsToTheCurrentTickAndFullDurability verifies that goods
// that enter from a boundary are born in the current tick if they spoil, have
// full durability if they wear and 0 otherwise, that the boundary goes more
// negative by the quantity, and that initial goods may go to an estate.
//
// Requirement: FR-LED-013, FR-LED-038, FR-LED-002, FR-LED-053
func TestCreation_DefaultsToTheCurrentTickAndFullDurability(t *testing.T) {
	t.Parallel()
	f := newGoods(t, goodsMaxQuantity)
	f.endConstruction(t)
	f.l.AdvanceTick()
	f.l.AdvanceTick() // tick 2
	f.create(t, f.a0, cSpoil, 4)
	f.create(t, f.a0, cWear, 2)
	f.create(t, f.e0, cPlain, 6)
	f.l.AdvanceTick() // tick 3
	f.create(t, f.a0, cSpoil, 1)

	wantA0 := []ledger.Lot{
		{Commodity: cSpoil, Quantity: 4, Born: 2},
		{Commodity: cSpoil, Quantity: 1, Born: 3},
		{Commodity: cWear, Quantity: 2, Durability: 5},
	}
	if got := f.lots(t, f.a0); !reflect.DeepEqual(got, wantA0) {
		t.Errorf("a0 lots = %v, want %v", got, wantA0)
	}
	wantE0 := []ledger.Lot{{Commodity: cPlain, Quantity: 6}}
	if got := f.lots(t, f.e0); !reflect.DeepEqual(got, wantE0) {
		t.Errorf("e0 lots = %v, want %v", got, wantE0)
	}
	for _, tt := range []struct {
		c    int
		want int64
	}{{cSpoil, -5}, {cWear, -2}, {cPlain, -6}} {
		v, _ := f.l.Account(f.initial[tt.c])
		if v.Balance != tt.want {
			t.Errorf("boundary of Initial for c%d = %d, want %d", tt.c, v.Balance, tt.want)
		}
	}
	// Region 0 for the agent and for the estate: the region of the
	// non-boundary side. Reasons are in registry order, commodities
	// ascending: 5 of cSpoil, 2 of cWear, 6 of cPlain.
	wantFlows := []ledger.Flow{
		{Region: 0, Reason: ledger.Initial, Commodity: cSpoil, Total: 5},
		{Region: 0, Reason: ledger.Initial, Commodity: cWear, Total: 2},
		{Region: 0, Reason: ledger.Initial, Commodity: cPlain, Total: 6},
	}
	if got := f.l.AppendFlows(nil, false); !reflect.DeepEqual(got, wantFlows) {
		t.Errorf("flow totals = %v, want %v", got, wantFlows)
	}
	if p := f.problems(); len(p) > 0 {
		t.Errorf("after creation: %v", p)
	}
}

// TestFlows_GoodsAreAttributedAndBucketed verifies that a move of goods is
// totalled in units, to the region of the non-boundary side or of the debited
// account, and that in construction it goes to the bucket and not to the
// totals.
//
// Requirement: FR-LED-023, FR-LED-045, FR-LED-053
func TestFlows_GoodsAreAttributedAndBucketed(t *testing.T) {
	t.Parallel()
	f := newGoods(t, goodsMaxQuantity)
	f.create(t, f.a1, cPlain, 5) // region 1, in construction
	if got := f.l.AppendFlows(nil, false); len(got) != 0 {
		t.Errorf("flow totals after a construction creation = %v, want none", got)
	}
	wantBucket := []ledger.Flow{{Region: 1, Reason: ledger.Initial, Commodity: cPlain, Total: 5}}
	if got := f.l.AppendFlows(nil, true); !reflect.DeepEqual(got, wantBucket) {
		t.Errorf("bucket = %v, want %v", got, wantBucket)
	}
	f.endConstruction(t)
	// a1 (region 1) to a0 (region 0): an internal flow, region of the
	// debited account, 1. Then a1 sinks 2 to Disposal: region 1.
	if err := f.l.Move(f.a1, f.a0, cPlain, 1, ledger.Trade); err != nil {
		t.Fatalf("Move: %v", err)
	}
	if err := f.l.Move(f.a1, f.disposal[cPlain], cPlain, 2, ledger.Disposal); err != nil {
		t.Fatalf("Move to Disposal: %v", err)
	}
	want := []ledger.Flow{
		{Region: 1, Reason: ledger.Trade, Commodity: cPlain, Total: 1},
		{Region: 1, Reason: ledger.Disposal, Commodity: cPlain, Total: 2},
	}
	if got := f.l.AppendFlows(nil, false); !reflect.DeepEqual(got, want) {
		t.Errorf("flow totals = %v, want %v", got, want)
	}
	if got := f.l.AppendFlows(nil, true); !reflect.DeepEqual(got, wantBucket) {
		t.Errorf("bucket after the run began = %v, want %v, unchanged", got, wantBucket)
	}
	v, _ := f.l.Account(f.disposal[cPlain])
	if v.Balance != 2 {
		t.Errorf("Disposal boundary = %d, want 2", v.Balance)
	}
}

// TestMoveStated_Ranges verifies that a creation under Initial in
// construction may state its lot key within the ranges of 5.2.4 and that one
// outside each range is refused with ErrInvalidLot, leaving the ledger
// unchanged.
//
// Requirement: FR-LED-043, FR-LED-007
func TestMoveStated_Ranges(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		key  ledger.Key
		want error
	}{
		// Shelf life 3: -(3-1) = -2 to 0.
		{"spoiling, oldest born accepted", ledger.Key{Commodity: cSpoil, Born: -2}, nil},
		{"spoiling, born 0 accepted", ledger.Key{Commodity: cSpoil, Born: 0}, nil},
		{"spoiling, one before the oldest", ledger.Key{Commodity: cSpoil, Born: -3}, ledger.ErrInvalidLot},
		{"spoiling, born after construction tick", ledger.Key{Commodity: cSpoil, Born: 1}, ledger.ErrInvalidLot},
		{"spoiling, durability stated", ledger.Key{Commodity: cSpoil, Born: 0, Durability: 1}, ledger.ErrInvalidLot},
		// Durability 5: 1 to 5.
		{"durable, durability 1 accepted", ledger.Key{Commodity: cWear, Durability: 1}, nil},
		{"durable, full durability accepted", ledger.Key{Commodity: cWear, Durability: 5}, nil},
		{"durable, durability 0", ledger.Key{Commodity: cWear, Durability: 0}, ledger.ErrInvalidLot},
		{"durable, above full", ledger.Key{Commodity: cWear, Durability: 6}, ledger.ErrInvalidLot},
		{"durable, born stated", ledger.Key{Commodity: cWear, Born: -1, Durability: 5}, ledger.ErrInvalidLot},
		{"plain, key 0 accepted", ledger.Key{Commodity: cPlain}, nil},
		{"plain, born stated", ledger.Key{Commodity: cPlain, Born: -1}, ledger.ErrInvalidLot},
		{"plain, durability stated", ledger.Key{Commodity: cPlain, Durability: 1}, ledger.ErrInvalidLot},
		{"unknown commodity", ledger.Key{Commodity: 9}, ledger.ErrUnknownCommodity},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := newGoods(t, goodsMaxQuantity)
			before := f.state()
			from := 0
			if tt.key.Commodity >= 0 && tt.key.Commodity < 3 {
				from = f.initial[tt.key.Commodity]
			}
			err := f.l.MoveStated(from, f.e0, tt.key, 3, ledger.Initial)
			if !errors.Is(err, tt.want) && (tt.want != nil || err != nil) {
				t.Fatalf("MoveStated(%+v) = %v, want %v", tt.key, err, tt.want)
			}
			if tt.want == nil {
				want := []ledger.Lot{{Commodity: tt.key.Commodity, Quantity: 3, Born: tt.key.Born, Durability: tt.key.Durability}}
				if got := f.lots(t, f.e0); !reflect.DeepEqual(got, want) {
					t.Errorf("estate lots = %v, want %v", got, want)
				}
				return
			}
			if after := f.state(); !reflect.DeepEqual(before, after) {
				t.Errorf("a refused MoveStated changed the ledger:\n before %+v\n after  %+v", before, after)
			}
		})
	}
}

// TestMoveStated_OnlyInConstructionUnderInitial verifies that a stated key
// after construction is ErrNotConstruction even when the key is valid, that
// it is only for a creation under Initial, and that a negative born stays in
// canonical order with the others.
//
// Requirement: FR-LED-043, FR-LED-044
func TestMoveStated_OnlyInConstructionUnderInitial(t *testing.T) {
	t.Parallel()
	f := newGoods(t, goodsMaxQuantity)
	f.stated(t, f.a0, cSpoil, 0, 0, 1)
	f.stated(t, f.a0, cSpoil, -2, 0, 1)
	f.stated(t, f.a0, cSpoil, -1, 0, 1)
	want := []ledger.Lot{
		{Commodity: cSpoil, Quantity: 1, Born: -2},
		{Commodity: cSpoil, Quantity: 1, Born: -1},
		{Commodity: cSpoil, Quantity: 1, Born: 0},
	}
	if got := f.lots(t, f.a0); !reflect.DeepEqual(got, want) {
		t.Errorf("lots = %v, want %v, negative born first", got, want)
	}
	// The key is valid but the source is an account, not the boundary, and
	// the reason is not Initial.
	if err := f.l.MoveStated(f.a0, f.a1, ledger.Key{Commodity: cSpoil, Born: -2}, 1, ledger.Trade); !errors.Is(err, ledger.ErrReasonMismatch) {
		t.Errorf("MoveStated from an account = %v, want ErrReasonMismatch", err)
	}
	if err := f.l.MoveStated(f.disposal[cSpoil], f.a1, ledger.Key{Commodity: cSpoil}, 1, ledger.Disposal); !errors.Is(err, ledger.ErrReasonMismatch) {
		t.Errorf("MoveStated under Disposal = %v, want ErrReasonMismatch", err)
	}
	f.endConstruction(t)
	before := f.state()
	err := f.l.MoveStated(f.initial[cSpoil], f.a1, ledger.Key{Commodity: cSpoil, Born: -1}, 1, ledger.Initial)
	if !errors.Is(err, ledger.ErrNotConstruction) {
		t.Errorf("MoveStated after construction = %v, want ErrNotConstruction", err)
	}
	if after := f.state(); !reflect.DeepEqual(before, after) {
		t.Errorf("a refused MoveStated changed the ledger")
	}
}

// TestMove_Refusals verifies that every refusal of a goods move, in its four
// forms, is its own sentinel, that each case breaks only that rule, and that a
// refusal leaves the ledger as it was with every commodity still totalling 0.
//
// Requirement: FR-LED-002, FR-LED-004, FR-LED-005, FR-LED-006, FR-LED-007, FR-LED-009, FR-LED-019, FR-LED-033, FR-LED-034, FR-LED-037
func TestMove_Refusals(t *testing.T) {
	t.Parallel()
	named := ledger.Key{Commodity: cSpoil, Born: -1} // a0 holds 3 of it
	tests := []struct {
		name string
		// big builds the ledger with MaxQuantity at the edge of int64.
		big   bool
		setup func(f *gfix, t *testing.T)
		do    func(f *gfix) error
		want  error
	}{
		{name: "quantity zero", do: func(f *gfix) error { return f.l.Move(f.a0, f.a1, cPlain, 0, ledger.Trade) }, want: ledger.ErrInvalidAmount},
		{name: "quantity negative", do: func(f *gfix) error { return f.l.Move(f.a0, f.a1, cPlain, -2, ledger.Trade) }, want: ledger.ErrInvalidAmount},
		{name: "unknown source", do: func(f *gfix) error { return f.l.Move(99, f.a1, cPlain, 1, ledger.Trade) }, want: ledger.ErrUnknownAccount},
		{name: "unknown target", do: func(f *gfix) error { return f.l.Move(f.a0, -1, cPlain, 1, ledger.Trade) }, want: ledger.ErrUnknownAccount},
		{
			name:  "closed target",
			setup: func(f *gfix, t *testing.T) { f.closeA1(t) },
			do:    func(f *gfix) error { return f.l.Move(f.a0, f.a1, cPlain, 1, ledger.Trade) },
			want:  ledger.ErrClosedAccount,
		},
		{
			name:  "closed source",
			setup: func(f *gfix, t *testing.T) { f.closeA1(t) },
			do:    func(f *gfix) error { return f.l.Move(f.a1, f.a0, cPlain, 1, ledger.Trade) },
			want:  ledger.ErrClosedAccount,
		},
		{name: "commodity above the table", do: func(f *gfix) error { return f.l.Move(f.a0, f.a1, 3, 1, ledger.Trade) }, want: ledger.ErrUnknownCommodity},
		{name: "money index as a commodity", do: func(f *gfix) error { return f.l.Move(f.a0, f.a1, ledger.Money, 1, ledger.Trade) }, want: ledger.ErrUnknownCommodity},
		{name: "reason out of range", do: func(f *gfix) error { return f.l.Move(f.a0, f.a1, cPlain, 1, ledger.NumReasons) }, want: ledger.ErrReasonMismatch},
		{name: "money-only reason moving goods", do: func(f *gfix) error { return f.l.Move(f.a0, f.e0, cPlain, 1, ledger.StartMoney) }, want: ledger.ErrReasonMismatch},
		{name: "sink between agents", do: func(f *gfix) error { return f.l.Move(f.a0, f.a1, cPlain, 1, ledger.Disposal) }, want: ledger.ErrReasonMismatch},
		{name: "disposal from a boundary", do: func(f *gfix) error {
			return f.l.Move(f.disposal[cPlain], f.a1, cPlain, 1, ledger.Disposal)
		}, want: ledger.ErrReasonMismatch},
		{name: "trade into a boundary", do: func(f *gfix) error { return f.l.Move(f.a0, f.disposal[cPlain], cPlain, 1, ledger.Trade) }, want: ledger.ErrReasonMismatch},
		{name: "same account", do: func(f *gfix) error { return f.l.Move(f.a0, f.a0, cPlain, 1, ledger.Trade) }, want: ledger.ErrReasonMismatch},
		{name: "boundary of another commodity", do: func(f *gfix) error {
			return f.l.Move(f.a0, f.disposal[cWear], cPlain, 1, ledger.Disposal)
		}, want: ledger.ErrReasonMismatch},
		{name: "insufficient stock", do: func(f *gfix) error { return f.l.Move(f.a0, f.a1, cPlain, 8, ledger.Trade) }, want: ledger.ErrInsufficientStock},
		{
			name: "earmarked units are not available",
			setup: func(f *gfix, t *testing.T) {
				// 7 held, 5 earmarked: 2 available.
				if err := f.l.ReserveGoods(f.a0, cPlain, 5); err != nil {
					t.Fatalf("ReserveGoods: %v", err)
				}
			},
			do:   func(f *gfix) error { return f.l.Move(f.a0, f.a1, cPlain, 3, ledger.Trade) },
			want: ledger.ErrInsufficientStock,
		},
		{
			name:  "target would exceed MaxQuantity",
			setup: func(f *gfix, t *testing.T) { f.create(t, f.a1, cPlain, goodsMaxQuantity) },
			do:    func(f *gfix) error { return f.l.Move(f.a0, f.a1, cPlain, 1, ledger.Trade) },
			want:  ledger.ErrBalanceOverflow,
		},
		{
			name:  "creation would exceed MaxQuantity",
			setup: func(f *gfix, t *testing.T) { f.create(t, f.a1, cPlain, goodsMaxQuantity) },
			do:    func(f *gfix) error { return f.l.Move(f.initial[cPlain], f.a1, cPlain, 1, ledger.Initial) },
			want:  ledger.ErrBalanceOverflow,
		},
		{
			name: "boundary would leave int64",
			big:  true,
			setup: func(f *gfix, t *testing.T) {
				f.create(t, f.a0, cPlain, math.MaxInt64)
			},
			// The boundary is at -MaxInt64; 2 more is below MinInt64, and the
			// target a1 is empty so only the boundary overflows.
			do:   func(f *gfix) error { return f.l.Move(f.initial[cPlain], f.a1, cPlain, 2, ledger.Initial) },
			want: ledger.ErrBalanceOverflow,
		},
		{name: "marked from a boundary", do: func(f *gfix) error {
			return f.l.MoveReserved(f.initial[cPlain], f.a1, cPlain, 1, ledger.Initial)
		}, want: ledger.ErrBoundaryReserve},
		{name: "marked with nothing earmarked", do: func(f *gfix) error { return f.l.MoveReserved(f.a0, f.a1, cPlain, 1, ledger.Trade) }, want: ledger.ErrReservationExceeds},
		{
			name: "marked above the earmark",
			setup: func(f *gfix, t *testing.T) {
				if err := f.l.ReserveGoods(f.a0, cPlain, 2); err != nil {
					t.Fatalf("ReserveGoods: %v", err)
				}
			},
			do:   func(f *gfix) error { return f.l.MoveReserved(f.a0, f.a1, cPlain, 3, ledger.Trade) },
			want: ledger.ErrReservationExceeds,
		},
		{name: "named lot not held", do: func(f *gfix) error {
			return f.l.MoveLot(f.a0, f.a1, ledger.Key{Commodity: cSpoil, Born: -2, Durability: 1}, 1, ledger.Trade)
		}, want: ledger.ErrUnknownLot},
		{name: "named lot from a boundary", do: func(f *gfix) error {
			return f.l.MoveLot(f.initial[cSpoil], f.a1, ledger.Key{Commodity: cSpoil}, 1, ledger.Initial)
		}, want: ledger.ErrUnknownLot},
		{name: "named lot with too few units", do: func(f *gfix) error { return f.l.MoveLot(f.a0, f.a1, named, 4, ledger.Trade) }, want: ledger.ErrInsufficientStock},
		{
			name: "named lot with too few unearmarked units",
			setup: func(f *gfix, t *testing.T) {
				if err := f.l.ReserveLot(f.a0, named, 2); err != nil {
					t.Fatalf("ReserveLot: %v", err)
				}
			},
			do:   func(f *gfix) error { return f.l.MoveLot(f.a0, f.a1, named, 2, ledger.Trade) },
			want: ledger.ErrInsufficientStock,
		},
		{name: "named marked with no earmark", do: func(f *gfix) error { return f.l.MoveLotReserved(f.a0, f.a1, named, 1, ledger.Trade) }, want: ledger.ErrReservationExceeds},
		{name: "named marked, unknown lot", do: func(f *gfix) error {
			return f.l.MoveLotReserved(f.a0, f.a1, ledger.Key{Commodity: cPlain, Born: 4}, 1, ledger.Trade)
		}, want: ledger.ErrUnknownLot},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			maxQuantity := int64(goodsMaxQuantity)
			if tt.big {
				maxQuantity = math.MaxInt64
			}
			f := newGoods(t, maxQuantity)
			if !tt.big {
				f.holdings(t)
			}
			f.endConstruction(t)
			if tt.setup != nil {
				tt.setup(f, t)
			}
			before := f.state()
			if err := tt.do(f); !errors.Is(err, tt.want) {
				t.Fatalf("got %v, want %v", err, tt.want)
			}
			if after := f.state(); !reflect.DeepEqual(before, after) {
				t.Errorf("refusal changed the ledger:\n before %+v\n after  %+v", before, after)
			}
			if p := f.problems(); len(p) > 0 {
				t.Errorf("after refusal: %v", p)
			}
		})
	}
}

// closeA1 closes the empty agent a1.
func (f *gfix) closeA1(t *testing.T) {
	t.Helper()
	if err := f.l.Close(f.a1); err != nil {
		t.Fatalf("setup: Close: %v", err)
	}
}

// TestEarmarks_CanonicalOrderAndUnmarkedMovesSkipThem verifies that a goods
// reservation earmarks units in canonical order, that an unmarked move takes
// only unearmarked units and fails when only earmarked ones are left, that a
// marked move consumes earmarked units, and that a release returns them.
//
// Requirement: FR-LED-017, FR-LED-018, FR-LED-019, FR-LED-034, FR-LED-007
func TestEarmarks_CanonicalOrderAndUnmarkedMovesSkipThem(t *testing.T) {
	t.Parallel()
	f := newGoods(t, goodsMaxQuantity)
	f.stated(t, f.a0, cSpoil, -2, 0, 4)
	f.stated(t, f.a0, cSpoil, -1, 0, 3)
	f.stated(t, f.a0, cSpoil, 0, 0, 5)
	f.endConstruction(t)

	// Reserve 6 of 12: the oldest lot (4) and 2 of the next.
	if err := f.l.ReserveGoods(f.a0, cSpoil, 6); err != nil {
		t.Fatalf("ReserveGoods(6): %v", err)
	}
	want := []ledger.Lot{
		{Commodity: cSpoil, Quantity: 4, Born: -2, Reserved: 4},
		{Commodity: cSpoil, Quantity: 3, Born: -1, Reserved: 2},
		{Commodity: cSpoil, Quantity: 5, Born: 0},
	}
	if got := f.lots(t, f.a0); !reflect.DeepEqual(got, want) {
		t.Fatalf("after reserve lots = %v, want %v", got, want)
	}

	// Unmarked 5 of the 6 available: born -1 has 1 unearmarked, born 0 has 5,
	// so it takes 1 and then 4. The earmarks stay where they are.
	if err := f.l.Move(f.a0, f.a1, cSpoil, 5, ledger.Trade); err != nil {
		t.Fatalf("Move(5): %v", err)
	}
	want = []ledger.Lot{
		{Commodity: cSpoil, Quantity: 4, Born: -2, Reserved: 4},
		{Commodity: cSpoil, Quantity: 2, Born: -1, Reserved: 2},
		{Commodity: cSpoil, Quantity: 1, Born: 0},
	}
	if got := f.lots(t, f.a0); !reflect.DeepEqual(got, want) {
		t.Fatalf("after unmarked move lots = %v, want %v", got, want)
	}
	wantTo := []ledger.Lot{
		{Commodity: cSpoil, Quantity: 1, Born: -1},
		{Commodity: cSpoil, Quantity: 4, Born: 0},
	}
	if got := f.lots(t, f.a1); !reflect.DeepEqual(got, wantTo) {
		t.Errorf("target lots = %v, want %v", got, wantTo)
	}

	// 7 held, 6 earmarked, 1 available: 2 is refused, 1 succeeds and leaves
	// only earmarked units, which an unmarked move cannot take.
	before := f.state()
	if err := f.l.Move(f.a0, f.a1, cSpoil, 2, ledger.Trade); !errors.Is(err, ledger.ErrInsufficientStock) {
		t.Errorf("Move(2) = %v, want ErrInsufficientStock", err)
	}
	if after := f.state(); !reflect.DeepEqual(before, after) {
		t.Errorf("refused Move changed the ledger")
	}
	if err := f.l.Move(f.a0, f.a1, cSpoil, 1, ledger.Trade); err != nil {
		t.Fatalf("Move(1): %v", err)
	}
	if err := f.l.Move(f.a0, f.a1, cSpoil, 1, ledger.Trade); !errors.Is(err, ledger.ErrInsufficientStock) {
		t.Errorf("Move of an earmarked unit = %v, want ErrInsufficientStock", err)
	}

	// A marked move of 5 consumes the earmark in canonical order: 4 of born
	// -2 and 1 of born -1. A lot that is emptied is gone, and the earmark of
	// what is left is 1.
	if err := f.l.MoveReserved(f.a0, f.a1, cSpoil, 5, ledger.Trade); err != nil {
		t.Fatalf("MoveReserved(5): %v", err)
	}
	want = []ledger.Lot{{Commodity: cSpoil, Quantity: 1, Born: -1, Reserved: 1}}
	if got := f.lots(t, f.a0); !reflect.DeepEqual(got, want) {
		t.Errorf("after marked move lots = %v, want %v", got, want)
	}
	if err := f.l.ReleaseGoods(f.a0, cSpoil, 1); err != nil {
		t.Fatalf("ReleaseGoods(1): %v", err)
	}
	if err := f.l.Move(f.a0, f.a1, cSpoil, 1, ledger.Trade); err != nil {
		t.Errorf("Move of a released unit = %v, want nil", err)
	}
	if p := f.problems(); len(p) > 0 {
		t.Errorf("after the script: %v", p)
	}
}

// TestEarmarks_ReleaseUndoesTheLastUnits verifies that releasing m of n
// earmarked units leaves the earmarks a reserve of n-m would have made, so the
// units a later marked move takes are still the oldest.
//
// Requirement: FR-LED-018, FR-LED-034
func TestEarmarks_ReleaseUndoesTheLastUnits(t *testing.T) {
	t.Parallel()
	f := newGoods(t, goodsMaxQuantity)
	f.stated(t, f.a0, cSpoil, -2, 0, 4)
	f.stated(t, f.a0, cSpoil, -1, 0, 3)
	if err := f.l.ReserveGoods(f.a0, cSpoil, 6); err != nil {
		t.Fatalf("ReserveGoods: %v", err)
	}
	// 6 reserved: born -2 4, born -1 2. Releasing 3 takes the last first: 2
	// from born -1 and 1 from born -2, leaving 3 on born -2 only.
	if err := f.l.ReleaseGoods(f.a0, cSpoil, 3); err != nil {
		t.Fatalf("ReleaseGoods: %v", err)
	}
	want := []ledger.Lot{
		{Commodity: cSpoil, Quantity: 4, Born: -2, Reserved: 3},
		{Commodity: cSpoil, Quantity: 3, Born: -1},
	}
	if got := f.lots(t, f.a0); !reflect.DeepEqual(got, want) {
		t.Errorf("lots = %v, want %v", got, want)
	}
}

// TestReservations_Refusals verifies the refusals of reserving and releasing
// goods, by commodity and by named lot, and that each leaves the ledger
// unchanged.
//
// Requirement: FR-LED-005, FR-LED-007, FR-LED-017, FR-LED-018, FR-LED-033
func TestReservations_Refusals(t *testing.T) {
	t.Parallel()
	named := ledger.Key{Commodity: cSpoil, Born: -1} // a0 holds 3 of it
	tests := []struct {
		name  string
		setup func(f *gfix, t *testing.T)
		do    func(f *gfix) error
		want  error
	}{
		{"reserve zero", nil, func(f *gfix) error { return f.l.ReserveGoods(f.a0, cPlain, 0) }, ledger.ErrInvalidAmount},
		{"release negative", nil, func(f *gfix) error { return f.l.ReleaseGoods(f.a0, cPlain, -1) }, ledger.ErrInvalidAmount},
		{"reserve unknown account", nil, func(f *gfix) error { return f.l.ReserveGoods(99, cPlain, 1) }, ledger.ErrUnknownAccount},
		{"release unknown commodity", nil, func(f *gfix) error { return f.l.ReleaseGoods(f.a0, 7, 1) }, ledger.ErrUnknownCommodity},
		{"reserve on a boundary", nil, func(f *gfix) error { return f.l.ReserveGoods(f.initial[cPlain], cPlain, 1) }, ledger.ErrBoundaryReserve},
		{"named reserve on a boundary", nil, func(f *gfix) error {
			return f.l.ReserveLot(f.initial[cPlain], ledger.Key{Commodity: cPlain}, 1)
		}, ledger.ErrBoundaryReserve},
		{"reserve above the holding", nil, func(f *gfix) error { return f.l.ReserveGoods(f.a0, cPlain, 8) }, ledger.ErrInsufficientStock},
		{"release with nothing earmarked", nil, func(f *gfix) error { return f.l.ReleaseGoods(f.a0, cPlain, 1) }, ledger.ErrReservationExceeds},
		{"release on a boundary", nil, func(f *gfix) error { return f.l.ReleaseGoods(f.initial[cPlain], cPlain, 1) }, ledger.ErrReservationExceeds},
		{"named reserve, no such lot", nil, func(f *gfix) error {
			return f.l.ReserveLot(f.a0, ledger.Key{Commodity: cSpoil, Born: -2, Durability: 3}, 1)
		}, ledger.ErrUnknownLot},
		{"named reserve above the lot", nil, func(f *gfix) error { return f.l.ReserveLot(f.a0, named, 4) }, ledger.ErrInsufficientStock},
		{"named release, no such lot", nil, func(f *gfix) error {
			return f.l.ReleaseLot(f.a0, ledger.Key{Commodity: cPlain, Born: 1}, 1)
		}, ledger.ErrUnknownLot},
		{"named release above the earmark", func(f *gfix, t *testing.T) {
			if err := f.l.ReserveLot(f.a0, named, 1); err != nil {
				t.Fatalf("setup: ReserveLot: %v", err)
			}
		}, func(f *gfix) error {
			return f.l.ReleaseLot(f.a0, named, 2)
		}, ledger.ErrReservationExceeds},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := newGoods(t, goodsMaxQuantity)
			f.holdings(t)
			f.endConstruction(t)
			if tt.setup != nil {
				tt.setup(f, t)
			}
			before := f.state()
			err := tt.do(f)
			if !errors.Is(err, tt.want) {
				t.Fatalf("got %v, want %v", err, tt.want)
			}
			if after := f.state(); !reflect.DeepEqual(before, after) {
				t.Errorf("refusal changed the ledger")
			}
		})
	}
}

// TestNamedLots_ActOnExactlyThatLot verifies that a named reserve, release
// and move act on the one lot, not on the oldest, and that the moved units
// keep the key.
//
// Requirement: FR-LED-033, FR-LED-034
func TestNamedLots_ActOnExactlyThatLot(t *testing.T) {
	t.Parallel()
	f := newGoods(t, goodsMaxQuantity)
	f.holdings(t)
	f.endConstruction(t)
	named := ledger.Key{Commodity: cWear, Durability: 5} // a0: 6 items; durability 2 (3 items) is more worn

	if err := f.l.ReserveLot(f.a0, named, 4); err != nil {
		t.Fatalf("ReserveLot: %v", err)
	}
	want := []ledger.Lot{
		{Commodity: cWear, Quantity: 3, Durability: 2},
		{Commodity: cWear, Quantity: 6, Durability: 5, Reserved: 4},
	}
	if got := f.lots(t, f.a0)[3:5]; !reflect.DeepEqual(got, want) {
		t.Fatalf("after ReserveLot lots = %v, want %v", got, want)
	}
	// A marked move of the named lot consumes 3 of its 4 earmarks.
	if err := f.l.MoveLotReserved(f.a0, f.a1, named, 3, ledger.Trade); err != nil {
		t.Fatalf("MoveLotReserved: %v", err)
	}
	// An unmarked named move takes the 2 unearmarked units of the lot, and
	// not the more worn lot beside it.
	if err := f.l.MoveLot(f.a0, f.a1, named, 2, ledger.Trade); err != nil {
		t.Fatalf("MoveLot: %v", err)
	}
	wantA0 := []ledger.Lot{
		{Commodity: cWear, Quantity: 3, Durability: 2},
		{Commodity: cWear, Quantity: 1, Durability: 5, Reserved: 1},
	}
	if got := f.lots(t, f.a0)[3:5]; !reflect.DeepEqual(got, wantA0) {
		t.Errorf("a0 wear lots = %v, want %v", got, wantA0)
	}
	wantA1 := []ledger.Lot{{Commodity: cWear, Quantity: 5, Durability: 5}}
	if got := f.lots(t, f.a1); !reflect.DeepEqual(got, wantA1) {
		t.Errorf("a1 lots = %v, want %v", got, wantA1)
	}
	if err := f.l.ReleaseLot(f.a0, named, 1); err != nil {
		t.Fatalf("ReleaseLot: %v", err)
	}
	if got := f.lots(t, f.a0)[4].Reserved; got != 0 {
		t.Errorf("Reserved after ReleaseLot = %d, want 0", got)
	}
	if p := f.problems(); len(p) > 0 {
		t.Errorf("after named operations: %v", p)
	}
}

// TestMove_ToAndFromTheEstateAndSinks verifies that goods run through the
// kinds of the registry: agent to estate, estate to boundary under Spoilage,
// and that a move under a sink empties the lot and raises the boundary.
//
// Requirement: FR-LED-002, FR-LED-037
func TestMove_ToAndFromTheEstateAndSinks(t *testing.T) {
	t.Parallel()
	f := newGoods(t, goodsMaxQuantity)
	f.holdings(t)
	f.endConstruction(t)
	if err := f.l.Move(f.a0, f.e0, cSpoil, 12, ledger.Trade); err != nil {
		t.Fatalf("Move to estate: %v", err)
	}
	if err := f.l.Move(f.e0, f.spoilage0, cSpoil, 12, ledger.Spoilage); err != nil {
		t.Fatalf("Move to Spoilage: %v", err)
	}
	for _, id := range []int{f.a0, f.e0} {
		for _, lt := range f.lots(t, id) {
			if lt.Commodity == cSpoil {
				t.Errorf("account %d still holds %v", id, lt)
			}
		}
	}
	if v, _ := f.l.Account(f.spoilage0); v.Balance != 12 {
		t.Errorf("Spoilage boundary = %d, want 12", v.Balance)
	}
	if p := f.problems(); len(p) > 0 {
		t.Errorf("after sink: %v", p)
	}
}

// TestClose_RefusesGoodsAndEarmarks verifies that Close refuses an account
// holding goods with ErrNotEmpty and one holding an earmark with
// ErrOpenReservation, which comes first, and that both leave the account
// open and the ledger unchanged.
//
// Requirement: FR-LED-009, FR-LED-007
func TestClose_RefusesGoodsAndEarmarks(t *testing.T) {
	t.Parallel()
	f := newGoods(t, goodsMaxQuantity)
	f.create(t, f.a0, cPlain, 2)
	before := f.state()
	if err := f.l.Close(f.a0); !errors.Is(err, ledger.ErrNotEmpty) {
		t.Errorf("Close with goods = %v, want ErrNotEmpty", err)
	}
	if err := f.l.ReserveGoods(f.a0, cPlain, 1); err != nil {
		t.Fatalf("ReserveGoods: %v", err)
	}
	if err := f.l.Close(f.a0); !errors.Is(err, ledger.ErrOpenReservation) {
		t.Errorf("Close with an earmark = %v, want ErrOpenReservation", err)
	}
	if err := f.l.ReleaseGoods(f.a0, cPlain, 1); err != nil {
		t.Fatalf("ReleaseGoods: %v", err)
	}
	if after := f.state(); !reflect.DeepEqual(before, after) {
		t.Errorf("refused Close changed the ledger")
	}
	if err := f.l.Move(f.a0, f.disposal[cPlain], cPlain, 2, ledger.Disposal); err != nil {
		t.Fatalf("Move: %v", err)
	}
	if err := f.l.Close(f.a0); err != nil {
		t.Errorf("Close of an empty account = %v, want nil", err)
	}
	if lots, err := f.l.AppendLots(nil, f.a0); err != nil || len(lots) != 0 {
		t.Errorf("AppendLots of a closed account = %v, %v, want none", lots, err)
	}
}

// TestGoods_ScriptKeepsEveryInvariant verifies, over a fixed script of
// creations, moves, earmarks and a tick advance, that after every step each
// commodity totals 0 and every account is in canonical form.
//
// Requirement: FR-LED-002, FR-LED-003, FR-LED-010, FR-LED-011
func TestGoods_ScriptKeepsEveryInvariant(t *testing.T) {
	t.Parallel()
	f := newGoods(t, goodsMaxQuantity)
	f.holdings(t)
	f.endConstruction(t)
	steps := []struct {
		name string
		do   func() error
	}{
		{"create at tick 0", func() error { return f.l.Move(f.initial[cSpoil], f.a1, cSpoil, 9, ledger.Initial) }},
		{"advance", func() error { f.l.AdvanceTick(); return nil }},
		{"create at tick 1", func() error { return f.l.Move(f.initial[cSpoil], f.a1, cSpoil, 4, ledger.Initial) }},
		{"reserve", func() error { return f.l.ReserveGoods(f.a1, cSpoil, 10) }},
		{"marked move", func() error { return f.l.MoveReserved(f.a1, f.e0, cSpoil, 6, ledger.Trade) }},
		{"unmarked move", func() error { return f.l.Move(f.a0, f.a1, cWear, 5, ledger.Trade) }},
		{"named move", func() error {
			return f.l.MoveLot(f.a0, f.e0, ledger.Key{Commodity: cSpoil, Born: 0}, 5, ledger.Trade)
		}},
		{"release", func() error { return f.l.ReleaseGoods(f.a1, cSpoil, 4) }},
		{"sink", func() error { return f.l.Move(f.a1, f.disposal[cSpoil], cSpoil, 7, ledger.Disposal) }},
		{"refused", func() error {
			if err := f.l.Move(f.a1, f.a0, cSpoil, 1000, ledger.Trade); !errors.Is(err, ledger.ErrInsufficientStock) {
				return fmt.Errorf("got %v, want ErrInsufficientStock", err)
			}
			return nil
		}},
	}
	for _, s := range steps {
		if err := s.do(); err != nil {
			t.Fatalf("step %q: %v", s.name, err)
		}
		if p := f.problems(); len(p) > 0 {
			t.Fatalf("after step %q: %v", s.name, p)
		}
	}
	if got := f.l.Tick(); got != 1 {
		t.Errorf("Tick() = %d, want 1", got)
	}
}
