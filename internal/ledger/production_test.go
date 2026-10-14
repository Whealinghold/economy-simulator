package ledger_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/ledger"
)

// The commodities of the production fixture. cSpoil, cWear and cPlain are those
// of the goods fixture, with the same shelf life and durability.
const (
	cBoth = 3 // shelf life 4 and durability 6
	nComm = 4
)

// pfix is a ledger for Transform, wear and ageing: four commodities, two
// regions, and the boundary accounts of Initial, RecipeInput, RecipeOutput,
// Wear and Spoilage for every commodity.
type pfix struct {
	l          *ledger.Ledger
	a0, a1, e0 int
	initial    [nComm]int
}

const prodMaxQuantity = 100

// newProd builds the fixture in its construction phase. The boundary account
// of reason omit for commodity omitC is not opened, to test the refusal of a
// ledger that lacks it; pass ledger.NumReasons for none.
func newProd(t *testing.T, omit ledger.Reason, omitC int) *pfix {
	t.Helper()
	l, err := ledger.New(ledger.Config{
		Commodities: []ledger.Commodity{{ShelfLife: 3}, {Durability: 5}, {}, {ShelfLife: 4, Durability: 6}},
		Regions:     2,
		MaxMoney:    1000,
		MaxQuantity: prodMaxQuantity,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	f := &pfix{l: l}
	for c := 0; c < nComm; c++ {
		for _, r := range []ledger.Reason{ledger.Initial, ledger.RecipeInput, ledger.RecipeOutput, ledger.Wear, ledger.Spoilage} {
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
	f.a0, f.a1, f.e0 = open(ledger.KindAgent, 0), open(ledger.KindAgent, 1), open(ledger.KindEstate, 0)
	return f
}

// stated creates qty units under a stated key, in construction.
func (f *pfix) stated(t *testing.T, to, c int, born, dur, qty int64) {
	t.Helper()
	k := ledger.Key{Commodity: c, Born: born, Durability: dur}
	if err := f.l.MoveStated(f.initial[c], to, k, qty, ledger.Initial); err != nil {
		t.Fatalf("setup: stated c%d born %d dur %d x%d: %v", c, born, dur, qty, err)
	}
}

// holdings is the standard holding of a0, in construction:
//
//	c0: (born -2 x2), (born -1 x1), (born 0 x3)
//	c1: (dur 2 x1), (dur 5 x4)
//	c2: 10
//	c3: (born -1, dur 3 x2), (born 0, dur 3 x1), (born 0, dur 6 x4)
func (f *pfix) holdings(t *testing.T) {
	t.Helper()
	f.stated(t, f.a0, cSpoil, -2, 0, 2)
	f.stated(t, f.a0, cSpoil, -1, 0, 1)
	f.stated(t, f.a0, cSpoil, 0, 0, 3)
	f.stated(t, f.a0, cWear, 0, 2, 1)
	f.stated(t, f.a0, cWear, 0, 5, 4)
	f.stated(t, f.a0, cPlain, 0, 0, 10)
	f.stated(t, f.a0, cBoth, -1, 3, 2)
	f.stated(t, f.a0, cBoth, 0, 3, 1)
	f.stated(t, f.a0, cBoth, 0, 6, 4)
}

// run ends construction and advances the tick n times without ageing.
func (f *pfix) run(t *testing.T, n int) {
	t.Helper()
	if err := f.l.EndConstruction(); err != nil {
		t.Fatalf("setup: EndConstruction: %v", err)
	}
	for ; n > 0; n-- {
		f.l.EndTick()
	}
}

func (f *pfix) lots(t *testing.T, id int) []ledger.Lot {
	t.Helper()
	got, err := f.l.AppendLots(nil, id)
	if err != nil {
		t.Fatalf("AppendLots(%d): %v", id, err)
	}
	return got
}

// balance is the balance of the boundary account of a reason and commodity.
func (f *pfix) balance(t *testing.T, r ledger.Reason, c int) int64 {
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

// pstate is everything observable, for comparing a ledger with its earlier
// self with reflect.DeepEqual.
type pstate struct {
	Accounts       []ledger.View
	Lots           [][]ledger.Lot
	Totals, Bucket []ledger.Flow
	Wear, WearB    []int64
	Construction   bool
	Tick           int64
}

func (f *pfix) state() pstate {
	s := pstate{
		Totals:       f.l.AppendFlows(nil, false),
		Bucket:       f.l.AppendFlows(nil, true),
		Construction: f.l.InConstruction(),
		Tick:         f.l.Tick(),
	}
	for c := 0; c < nComm; c++ {
		w, _ := f.l.WearUnits(c, false)
		b, _ := f.l.WearUnits(c, true)
		s.Wear, s.WearB = append(s.Wear, w), append(s.WearB, b)
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

// conserved reports the commodities whose total over every account,
// boundaries included, is not 0 (FR-LED-002).
func (f *pfix) conserved() (bad []int) {
	for c := 0; c < nComm; c++ {
		var total int64
		for _, r := range []ledger.Reason{ledger.Initial, ledger.RecipeInput, ledger.RecipeOutput, ledger.Wear, ledger.Spoilage} {
			if id, ok := f.l.Boundary(r, c); ok {
				v, _ := f.l.Account(id)
				total += v.Balance
			}
		}
		for id := 0; id < f.l.Len(); id++ {
			if v, _ := f.l.Account(id); v.Kind == ledger.KindBoundary {
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
			bad = append(bad, c)
		}
	}
	return bad
}

func lot(c int, qty, born, dur int64) ledger.Lot {
	return ledger.Lot{Commodity: c, Quantity: qty, Born: born, Durability: dur}
}

// TestTransform_AppliesInputsAndOutputsTogether verifies that a Transform takes
// its inputs oldest first to the recipe_input boundary and creates its outputs
// from the recipe_output boundary as 5.2.3 says, that either side may be empty,
// that a commodity may repeat across lines, and that an output is checked
// against the holding left after the inputs.
//
// Requirement: FR-LED-021, FR-LED-039, FR-LED-002
func TestTransform_AppliesInputsAndOutputsTogether(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		ticks   int
		inputs  []ledger.Line
		outputs []ledger.Line
		// want is the lots of a0 of commodity wantC after the call.
		wantC   int
		want    []ledger.Lot
		balance map[[2]int]int64 // (reason, commodity) -> boundary balance
	}{
		{
			// c0 x3 takes (born -2 x2) and (born -1 x1), so (born 0 x3) stays.
			// c2 is 10 - 4 = 6. The two outputs of c1 are made at full
			// durability 5 and merge into (dur 5 x4).
			name:   "inputs oldest first",
			inputs: []ledger.Line{{Commodity: cSpoil, Quantity: 3}, {Commodity: cPlain, Quantity: 4}},
			outputs: []ledger.Line{
				{Commodity: cWear, Quantity: 2},
			},
			wantC: cSpoil,
			want:  []ledger.Lot{lot(cSpoil, 3, 0, 0)},
			balance: map[[2]int]int64{
				{int(ledger.RecipeInput), cSpoil}:  3,
				{int(ledger.RecipeInput), cPlain}:  4,
				{int(ledger.RecipeOutput), cWear}:  -2,
				{int(ledger.RecipeOutput), cPlain}: 0,
			},
		},
		{
			// 2 + 1 from the two oldest lots, then 1 of the 3 in born 0: 2 left.
			name:   "split of the youngest lot",
			inputs: []ledger.Line{{Commodity: cSpoil, Quantity: 4}},
			wantC:  cSpoil,
			want:   []ledger.Lot{lot(cSpoil, 2, 0, 0)},
		},
		{
			// The outputs are made in tick 2: born 2 for the spoiling c0, and
			// full durability 5 for the durable c1, which merges with the
			// (dur 5 x4) lot: 4 + 2 = 6.
			name:    "outputs born in the current tick",
			ticks:   2,
			outputs: []ledger.Line{{Commodity: cSpoil, Quantity: 1}, {Commodity: cWear, Quantity: 2}},
			wantC:   cWear,
			want:    []ledger.Lot{lot(cWear, 1, 0, 2), lot(cWear, 6, 0, 5)},
		},
		{
			// A source: no inputs. c2 10 + 5.
			name:    "a source",
			outputs: []ledger.Line{{Commodity: cPlain, Quantity: 5}},
			wantC:   cPlain,
			want:    []ledger.Lot{lot(cPlain, 15, 0, 0)},
		},
		{
			// A job start: no outputs, and the emptied lot disappears.
			name:   "no outputs empties a lot",
			inputs: []ledger.Line{{Commodity: cPlain, Quantity: 10}},
			wantC:  cPlain,
			want:   nil,
		},
		{
			// The input is taken first: 10 - 10 + 100 = 100, which fits
			// MaxQuantity 100, though 10 + 100 would not.
			name:    "output is checked after the inputs",
			inputs:  []ledger.Line{{Commodity: cPlain, Quantity: 10}},
			outputs: []ledger.Line{{Commodity: cPlain, Quantity: 100}},
			wantC:   cPlain,
			want:    []ledger.Lot{lot(cPlain, 100, 0, 0)},
		},
		{
			// 3 + 4 = 7 taken from the 10, in two lines of one commodity.
			name:   "repeated commodity adds",
			inputs: []ledger.Line{{Commodity: cPlain, Quantity: 3}, {Commodity: cPlain, Quantity: 4}},
			wantC:  cPlain,
			want:   []ledger.Lot{lot(cPlain, 3, 0, 0)},
			balance: map[[2]int]int64{
				{int(ledger.RecipeInput), cPlain}: 7,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := newProd(t, ledger.NumReasons, -1)
			f.holdings(t)
			f.run(t, tt.ticks)
			if err := f.l.Transform(f.a0, tt.inputs, tt.outputs); err != nil {
				t.Fatalf("Transform(%v, %v) = %v, want nil", tt.inputs, tt.outputs, err)
			}
			var got []ledger.Lot
			for _, lt := range f.lots(t, f.a0) {
				if lt.Commodity == tt.wantC {
					got = append(got, lt)
				}
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("lots of commodity %d = %v, want %v", tt.wantC, got, tt.want)
			}
			for k, want := range tt.balance {
				if b := f.balance(t, ledger.Reason(k[0]), k[1]); b != want {
					t.Errorf("boundary (reason %d, commodity %d) = %d, want %d", k[0], k[1], b, want)
				}
			}
			if bad := f.conserved(); len(bad) != 0 {
				t.Errorf("commodities %v do not total 0", bad)
			}
		})
	}
}

// TestTransform_Refusals verifies one refusal per rule, each breaking only
// that rule, leaves the ledger unchanged, and that a failing output leaves
// the inputs untouched.
//
// Requirement: FR-LED-021, FR-LED-039, FR-LED-005, FR-LED-006, FR-LED-007
func TestTransform_Refusals(t *testing.T) {
	t.Parallel()
	one := func(c int, q int64) []ledger.Line { return []ledger.Line{{Commodity: c, Quantity: q}} }
	tests := []struct {
		name    string
		omit    ledger.Reason
		omitC   int
		prep    func(*pfix)
		account func(*pfix) int
		inputs  []ledger.Line
		outputs []ledger.Line
		want    error
	}{
		{name: "both empty", want: ledger.ErrEmptyTransform},
		{name: "input of 0", inputs: one(cPlain, 0), want: ledger.ErrInvalidAmount},
		{name: "negative output", outputs: one(cPlain, -1), want: ledger.ErrInvalidAmount},
		{name: "unknown account", account: func(*pfix) int { return 99 }, inputs: one(cPlain, 1), want: ledger.ErrUnknownAccount},
		{name: "negative account", account: func(*pfix) int { return -1 }, inputs: one(cPlain, 1), want: ledger.ErrUnknownAccount},
		{
			name:    "closed account",
			account: func(f *pfix) int { return f.a1 },
			prep: func(f *pfix) {
				if err := f.l.Close(f.a1); err != nil {
					panic(err)
				}
			},
			inputs: one(cPlain, 1),
			want:   ledger.ErrClosedAccount,
		},
		{name: "unknown input commodity", inputs: one(9, 1), want: ledger.ErrUnknownCommodity},
		{name: "negative output commodity", outputs: one(-1, 1), want: ledger.ErrUnknownCommodity},
		{name: "estate cannot run a recipe", account: func(f *pfix) int { return f.e0 }, inputs: one(cPlain, 1), want: ledger.ErrReasonMismatch},
		{
			name:    "a boundary cannot run a recipe",
			account: func(f *pfix) int { return f.initial[cPlain] },
			outputs: one(cPlain, 1),
			want:    ledger.ErrReasonMismatch,
		},
		{name: "no input boundary", omit: ledger.RecipeInput, omitC: cPlain, inputs: one(cPlain, 1), want: ledger.ErrUnknownAccount},
		{name: "no output boundary", omit: ledger.RecipeOutput, omitC: cPlain, outputs: one(cPlain, 1), want: ledger.ErrUnknownAccount},
		{name: "input above the holding", inputs: one(cPlain, 11), want: ledger.ErrInsufficientStock},
		{
			// 6 + 5 = 11 > 10: each line alone fits.
			name:   "repeated lines are summed",
			inputs: []ledger.Line{{Commodity: cPlain, Quantity: 6}, {Commodity: cPlain, Quantity: 5}},
			want:   ledger.ErrInsufficientStock,
		},
		{
			// 5 of the 10 are earmarked, so 6 are not available.
			name:   "earmarked units are not available",
			prep:   func(f *pfix) { mustNil(f.l.ReserveGoods(f.a0, cPlain, 5)) },
			inputs: one(cPlain, 6),
			want:   ledger.ErrInsufficientStock,
		},
		{
			// 10 - 1 + 92 = 101 > MaxQuantity 100. The input of c0 would
			// have been valid on its own.
			name:    "output overflows and the input is untouched",
			inputs:  []ledger.Line{{Commodity: cSpoil, Quantity: 1}, {Commodity: cPlain, Quantity: 1}},
			outputs: one(cPlain, 92),
			want:    ledger.ErrBalanceOverflow,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := newProd(t, tt.omit, tt.omitC)
			f.holdings(t)
			f.run(t, 0)
			if tt.prep != nil {
				tt.prep(f)
			}
			id := f.a0
			if tt.account != nil {
				id = tt.account(f)
			}
			before := f.state()
			err := f.l.Transform(id, tt.inputs, tt.outputs)
			if !errors.Is(err, tt.want) {
				t.Errorf("Transform(%d, %v, %v) = %v, want %v", id, tt.inputs, tt.outputs, err, tt.want)
			}
			if after := f.state(); !reflect.DeepEqual(after, before) {
				t.Errorf("ledger changed by a refused Transform:\n got %+v\nwant %+v", after, before)
			}
		})
	}
}

func mustNil(err error) {
	if err != nil {
		panic(err)
	}
}

// TestWearItems_Selection verifies the choice and effect of wear: the most
// worn items first and then canonical order, selected before any is worn,
// destruction at 0 and below without carrying the excess, survivors merged
// into the lot of their new durability, earmarked items skipped, and the units
// counted being the durability actually removed.
//
// Requirement: FR-LED-015, FR-LED-016, FR-LED-040, FR-LED-041, FR-LED-042
func TestWearItems_Selection(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		prep      func(*pfix)
		commodity int
		n, units  int64
		want      []ledger.Lot
		wantUnits int64
		// wantWear is the quantity moved to the wear boundary.
		wantWear int64
	}{
		{
			// The (dur 2) item is the most worn: 2 - 1 = 1.
			name: "most worn first", commodity: cWear, n: 1, units: 1,
			want:      []ledger.Lot{lot(cWear, 1, 0, 1), lot(cWear, 4, 0, 5)},
			wantUnits: 1,
		},
		{
			// 2 - 2 = 0: destroyed exactly at 0; counted 2.
			name: "destroyed at exactly zero", commodity: cWear, n: 1, units: 2,
			want:      []ledger.Lot{lot(cWear, 4, 0, 5)},
			wantUnits: 2, wantWear: 1,
		},
		{
			// 2 - 7 < 0: destroyed; counted 2, the 5 left over is not carried
			// to the (dur 5) items.
			name: "destroyed below zero without carry", commodity: cWear, n: 1, units: 7,
			want:      []ledger.Lot{lot(cWear, 4, 0, 5)},
			wantUnits: 2, wantWear: 1,
		},
		{
			// Three items: the dur 2 one and two of dur 5. Each loses 1:
			// 2->1, 5->4 x2. No item is worn twice: the dur 1 result is not
			// destroyed.
			name: "n items over two lots", commodity: cWear, n: 3, units: 1,
			want:      []ledger.Lot{lot(cWear, 1, 0, 1), lot(cWear, 2, 0, 4), lot(cWear, 2, 0, 5)},
			wantUnits: 3,
		},
		{
			// All five: 2 -> destroyed (counted 2), 4 x (5 - 5 = 0 destroyed,
			// counted 5 each): 2 + 20 = 22.
			name: "every item destroyed", commodity: cWear, n: 5, units: 5,
			want:      nil,
			wantUnits: 22, wantWear: 5,
		},
		{
			// Two items of c3 at the lowest durability, 3, in canonical order:
			// both of (born -1, dur 3). They become (born -1, dur 2).
			name: "ties go in canonical order", commodity: cBoth, n: 2, units: 1,
			want:      []ledger.Lot{lot(cBoth, 2, -1, 2), lot(cBoth, 1, 0, 3), lot(cBoth, 4, 0, 6)},
			wantUnits: 2,
		},
		{
			// Three: the two of born -1 and then the one of born 0, all dur 3,
			// before any of the dur 6 items.
			name: "all lots of one durability", commodity: cBoth, n: 3, units: 1,
			want:      []ledger.Lot{lot(cBoth, 2, -1, 2), lot(cBoth, 1, 0, 2), lot(cBoth, 4, 0, 6)},
			wantUnits: 3,
		},
		{
			// One item of (born -1, dur 3) loses 3 and is destroyed: counted 3.
			name: "oldest of equal wear is destroyed", commodity: cBoth, n: 1, units: 3,
			want:      []ledger.Lot{lot(cBoth, 1, -1, 3), lot(cBoth, 1, 0, 3), lot(cBoth, 4, 0, 6)},
			wantUnits: 3, wantWear: 1,
		},
		{
			// The dur 2 item is earmarked, so the most worn available item is
			// of dur 5: 5 - 1 = 4.
			name: "earmarked item is not worn", commodity: cWear, n: 1, units: 1,
			prep:      func(f *pfix) { mustNil(f.l.ReserveGoods(f.a0, cWear, 1)) },
			want:      []ledger.Lot{{Commodity: cWear, Quantity: 1, Durability: 2, Reserved: 1}, lot(cWear, 1, 0, 4), lot(cWear, 3, 0, 5)},
			wantUnits: 1,
		},
		{
			// Survivors of dur 5 lose 3 and land on 2, which is the key of the
			// earmarked lot: they merge into it (1 + 1 = 2, 1 earmarked).
			name: "survivor merges into an existing lot", commodity: cWear, n: 1, units: 3,
			prep:      func(f *pfix) { mustNil(f.l.ReserveGoods(f.a0, cWear, 1)) },
			want:      []ledger.Lot{{Commodity: cWear, Quantity: 2, Durability: 2, Reserved: 1}, lot(cWear, 3, 0, 5)},
			wantUnits: 3,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := newProd(t, ledger.NumReasons, -1)
			f.holdings(t)
			f.run(t, 0)
			if tt.prep != nil {
				tt.prep(f)
			}
			if err := f.l.WearItems(f.a0, tt.commodity, tt.n, tt.units); err != nil {
				t.Fatalf("WearItems(%d, %d, %d) = %v, want nil", tt.commodity, tt.n, tt.units, err)
			}
			var got []ledger.Lot
			for _, lt := range f.lots(t, f.a0) {
				if lt.Commodity == tt.commodity {
					got = append(got, lt)
				}
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("lots = %v, want %v", got, tt.want)
			}
			if u, _ := f.l.WearUnits(tt.commodity, false); u != tt.wantUnits {
				t.Errorf("WearUnits = %d, want %d", u, tt.wantUnits)
			}
			if b := f.balance(t, ledger.Wear, tt.commodity); b != tt.wantWear {
				t.Errorf("wear boundary = %d, want %d", b, tt.wantWear)
			}
			if bad := f.conserved(); len(bad) != 0 {
				t.Errorf("commodities %v do not total 0", bad)
			}
		})
	}
}

// TestWearItems_Refusals verifies each refusal of WearItems, that it breaks
// only its rule, and that the ledger is unchanged after it.
//
// Requirement: FR-LED-041, FR-LED-007
func TestWearItems_Refusals(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		omit      ledger.Reason
		omitC     int
		prep      func(*pfix)
		account   func(*pfix) int
		commodity int
		n, units  int64
		want      error
	}{
		{name: "n of 0", commodity: cWear, n: 0, units: 1, want: ledger.ErrInvalidAmount},
		{name: "units of 0", commodity: cWear, n: 1, units: 0, want: ledger.ErrInvalidAmount},
		{name: "negative units", commodity: cWear, n: 1, units: -1, want: ledger.ErrInvalidAmount},
		{name: "unknown account", account: func(*pfix) int { return 99 }, commodity: cWear, n: 1, units: 1, want: ledger.ErrUnknownAccount},
		{
			name: "closed account", account: func(f *pfix) int { return f.a1 },
			prep:      func(f *pfix) { mustNil(f.l.Close(f.a1)) },
			commodity: cWear, n: 1, units: 1, want: ledger.ErrClosedAccount,
		},
		{name: "unknown commodity", commodity: 9, n: 1, units: 1, want: ledger.ErrUnknownCommodity},
		{name: "estate holds no tools", account: func(f *pfix) int { return f.e0 }, commodity: cWear, n: 1, units: 1, want: ledger.ErrReasonMismatch},
		{name: "commodity does not wear", commodity: cPlain, n: 1, units: 1, want: ledger.ErrNotDurable},
		{name: "spoiling commodity does not wear", commodity: cSpoil, n: 1, units: 1, want: ledger.ErrNotDurable},
		{name: "no wear boundary", omit: ledger.Wear, omitC: cWear, commodity: cWear, n: 1, units: 1, want: ledger.ErrUnknownAccount},
		{name: "more items than held", commodity: cWear, n: 6, units: 1, want: ledger.ErrInsufficientStock},
		{
			// 5 items, 3 earmarked: 2 available.
			name:      "earmarked items do not count",
			prep:      func(f *pfix) { mustNil(f.l.ReserveGoods(f.a0, cWear, 3)) },
			commodity: cWear, n: 3, units: 1, want: ledger.ErrInsufficientStock,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := newProd(t, tt.omit, tt.omitC)
			f.holdings(t)
			f.run(t, 0)
			if tt.prep != nil {
				tt.prep(f)
			}
			id := f.a0
			if tt.account != nil {
				id = tt.account(f)
			}
			before := f.state()
			err := f.l.WearItems(id, tt.commodity, tt.n, tt.units)
			if !errors.Is(err, tt.want) {
				t.Errorf("WearItems(%d, %d, %d, %d) = %v, want %v", id, tt.commodity, tt.n, tt.units, err, tt.want)
			}
			if after := f.state(); !reflect.DeepEqual(after, before) {
				t.Errorf("ledger changed by a refused WearItems:\n got %+v\nwant %+v", after, before)
			}
		})
	}
}

// TestWear_EqualsWearItemsOfOne verifies that Wear gives the result of
// WearItems with n = 1, the wear units included, on two copies of one ledger,
// and that wear applied while in construction is counted in the bucket and not
// in the totals.
//
// Requirement: FR-LED-042, FR-LED-016
func TestWear_EqualsWearItemsOfOne(t *testing.T) {
	t.Parallel()
	for _, units := range []int64{1, 2, 3, 9} {
		for _, c := range []int{cWear, cBoth} {
			a, b := newProd(t, ledger.NumReasons, -1), newProd(t, ledger.NumReasons, -1)
			for _, f := range []*pfix{a, b} {
				f.holdings(t)
			}
			if err := a.l.Wear(a.a0, c, units); err != nil {
				t.Fatalf("Wear(%d, %d) = %v", c, units, err)
			}
			if err := b.l.WearItems(b.a0, c, 1, units); err != nil {
				t.Fatalf("WearItems(%d, 1, %d) = %v", c, units, err)
			}
			if sa, sb := a.state(), b.state(); !reflect.DeepEqual(sa, sb) {
				t.Errorf("Wear(c%d, %d) differs from WearItems(.., 1, ..):\n got %+v\nwant %+v", c, units, sa, sb)
			}
		}
	}

	f := newProd(t, ledger.NumReasons, -1)
	f.holdings(t)
	// In construction: the dur 2 item is destroyed, counted 2.
	if err := f.l.Wear(f.a0, cWear, 5); err != nil {
		t.Fatal(err)
	}
	if u, _ := f.l.WearUnits(cWear, true); u != 2 {
		t.Errorf("bucket wear units = %d, want 2", u)
	}
	if u, _ := f.l.WearUnits(cWear, false); u != 0 {
		t.Errorf("totals wear units = %d, want 0 in construction", u)
	}
	f.run(t, 0)
	if err := f.l.Wear(f.a0, cWear, 1); err != nil {
		t.Fatal(err)
	}
	if u, _ := f.l.WearUnits(cWear, false); u != 1 {
		t.Errorf("totals wear units = %d, want 1", u)
	}
	f.l.EndTick()
	if u, _ := f.l.WearUnits(cWear, false); u != 0 {
		t.Errorf("totals wear units after EndTick = %d, want 0", u)
	}
	if u, _ := f.l.WearUnits(cWear, true); u != 2 {
		t.Errorf("bucket wear units after EndTick = %d, want 2 (cleared by nothing)", u)
	}
	if _, err := f.l.WearUnits(9, false); !errors.Is(err, ledger.ErrUnknownCommodity) {
		t.Errorf("WearUnits(9) = %v, want ErrUnknownCommodity", err)
	}
}

// TestAge_ExpiryTickAndOrder verifies that a lot made in tick t with shelf
// life S leaves at the end of tick t + S - 1 and not before, that a negative
// born tick follows the same rule, that a commodity that does not spoil never
// leaves, that every expired lot goes whole to the spoilage boundary in the
// region of its account, and that two runs give identical states.
//
// Requirement: FR-LED-014, FR-LED-044, FR-LED-002
func TestAge_ExpiryTickAndOrder(t *testing.T) {
	t.Parallel()
	var a0 int
	script := func(t *testing.T) (steps [][]ledger.Flow, final pstate) {
		f := newProd(t, ledger.NumReasons, -1)
		// a0: c0 born 0 x2 (S=3: leaves at the end of tick 2), c3 born -3 x1
		// (S=4: -3+4-1 = 0, leaves at the end of tick 0) and c3 born -2 x2
		// (leaves at the end of tick 1). a1: c0 born -1 x3 (-1+3-1 = 1).
		f.stated(t, f.a0, cSpoil, 0, 0, 2)
		f.stated(t, f.a0, cBoth, -3, 6, 1)
		f.stated(t, f.a0, cBoth, -2, 6, 2)
		f.stated(t, f.a1, cSpoil, -1, 0, 3)
		f.stated(t, f.a0, cPlain, 0, 0, 5)
		f.stated(t, f.a0, cWear, 0, 5, 5)
		f.run(t, 0)
		a0 = f.a0
		for tick := 0; tick < 4; tick++ {
			if err := f.l.Age(); err != nil {
				t.Fatalf("Age at tick %d = %v", tick, err)
			}
			steps = append(steps, f.l.AppendFlows(nil, false))
			f.l.EndTick()
		}
		return steps, f.state()
	}
	steps, final := script(t)
	// Region 1's c0 leaves in tick 1 with region 0's c3 lot of born -2, and
	// the flows list ascending region.
	want := [][]ledger.Flow{
		{{Region: 0, Reason: ledger.Spoilage, Commodity: cBoth, Total: 1}},
		{
			{Region: 0, Reason: ledger.Spoilage, Commodity: cBoth, Total: 2},
			{Region: 1, Reason: ledger.Spoilage, Commodity: cSpoil, Total: 3},
		},
		{{Region: 0, Reason: ledger.Spoilage, Commodity: cSpoil, Total: 2}},
		nil,
	}
	for i := range want {
		if !reflect.DeepEqual(steps[i], want[i]) {
			t.Errorf("flows moved by ageing at tick %d = %v, want %v", i, steps[i], want[i])
		}
	}
	// Nothing that does not spoil moved, and the state is reproducible.
	if got := final.Lots[a0]; !reflect.DeepEqual(got, []ledger.Lot{lot(cWear, 5, 0, 5), lot(cPlain, 5, 0, 0)}) {
		t.Errorf("a0 lots at the end = %v, want the durable and the plain only", got)
	}
	if _, again := script(t); !reflect.DeepEqual(final, again) {
		t.Errorf("two runs of the same script differ:\n got %+v\nwant %+v", again, final)
	}
}

// TestAge_WholeLotIncludingEarmark verifies that an expired lot goes whole,
// its earmarked units included, and that nothing is left behind.
//
// Requirement: FR-LED-014
func TestAge_WholeLotIncludingEarmark(t *testing.T) {
	t.Parallel()
	f := newProd(t, ledger.NumReasons, -1)
	f.stated(t, f.a0, cSpoil, -2, 0, 4) // expires at the end of tick 0
	f.run(t, 0)
	if err := f.l.ReserveGoods(f.a0, cSpoil, 3); err != nil {
		t.Fatal(err)
	}
	if err := f.l.Age(); err != nil {
		t.Fatal(err)
	}
	if got := f.lots(t, f.a0); len(got) != 0 {
		t.Errorf("lots after ageing = %v, want none", got)
	}
	if b := f.balance(t, ledger.Spoilage, cSpoil); b != 4 {
		t.Errorf("spoilage boundary = %d, want 4", b)
	}
}

// TestAge_RefusesWithoutSpoilageBoundaryAndChangesNothing verifies that an
// expiring lot of a commodity with no spoilage boundary account is refused
// before anything moves, including other expiring lots.
//
// Requirement: FR-LED-007, FR-LED-014
func TestAge_RefusesWithoutSpoilageBoundaryAndChangesNothing(t *testing.T) {
	t.Parallel()
	f := newProd(t, ledger.Spoilage, cBoth)
	f.stated(t, f.a0, cSpoil, -2, 0, 4)
	f.stated(t, f.a0, cBoth, -3, 6, 1)
	f.run(t, 0)
	before := f.state()
	if err := f.l.Age(); !errors.Is(err, ledger.ErrUnknownAccount) {
		t.Errorf("Age() = %v, want ErrUnknownAccount", err)
	}
	if after := f.state(); !reflect.DeepEqual(after, before) {
		t.Errorf("ledger changed by a refused Age:\n got %+v\nwant %+v", after, before)
	}
}

// TestExpiring verifies the units that spoil by a tick: only unearmarked
// units, 0 for a commodity that does not spoil and for a past tick, the two
// account errors, and that for the current tick the sum over commodities is
// what ageing then moves.
//
// Requirement: FR-LED-048, FR-LED-049
func TestExpiring(t *testing.T) {
	t.Parallel()
	setup := func(t *testing.T) *pfix {
		f := newProd(t, ledger.NumReasons, -1)
		// c0: (born -1 x4) expires at 1, (born 0 x3) at 2. c3: (born -2 x2)
		// at -2+4-1 = 1, (born 0 x1) at 3.
		f.stated(t, f.a0, cSpoil, -1, 0, 4)
		f.stated(t, f.a0, cSpoil, 0, 0, 3)
		f.stated(t, f.a0, cBoth, -2, 6, 2)
		f.stated(t, f.a0, cBoth, 0, 6, 1)
		f.stated(t, f.a0, cWear, 0, 5, 2)
		f.stated(t, f.a0, cPlain, 0, 0, 2)
		f.run(t, 0)
		return f
	}
	tests := []struct {
		name      string
		prep      func(*pfix)
		commodity int
		tick      int64
		want      int64
	}{
		{name: "before any expiry", commodity: cSpoil, tick: 0, want: 0},
		{name: "first lot", commodity: cSpoil, tick: 1, want: 4},
		{name: "both lots", commodity: cSpoil, tick: 2, want: 7},
		{name: "far future", commodity: cSpoil, tick: 100, want: 7},
		{name: "second commodity, first lot", commodity: cBoth, tick: 1, want: 2},
		{name: "second commodity, both", commodity: cBoth, tick: 3, want: 3},
		{name: "durable does not spoil", commodity: cWear, tick: 100, want: 0},
		{name: "plain does not spoil", commodity: cPlain, tick: 100, want: 0},
		{
			// The 5 earmarked units are the 4 of born -1 and 1 of born 0, the
			// oldest first: 4 of the first lot and 1 of the second. By tick 1
			// none is available; by tick 2, 7 - 5 = 2.
			name:      "earmarked units are left out, first lot",
			prep:      func(f *pfix) { mustNil(f.l.ReserveGoods(f.a0, cSpoil, 5)) },
			commodity: cSpoil, tick: 1, want: 0,
		},
		{
			name:      "earmarked units are left out, both lots",
			prep:      func(f *pfix) { mustNil(f.l.ReserveGoods(f.a0, cSpoil, 5)) },
			commodity: cSpoil, tick: 2, want: 2,
		},
		{
			// Tick 2 is current, so a question about tick 1 is 0: that
			// ageing is over, though the lots are still held here.
			name:      "a tick before the current one",
			prep:      func(f *pfix) { f.l.EndTick(); f.l.EndTick() },
			commodity: cSpoil, tick: 1, want: 0,
		},
		{
			name:      "the current tick after the clock moved",
			prep:      func(f *pfix) { f.l.EndTick(); f.l.EndTick() },
			commodity: cSpoil, tick: 2, want: 7,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := setup(t)
			if tt.prep != nil {
				tt.prep(f)
			}
			before := f.state()
			got, err := f.l.Expiring(f.a0, tt.commodity, tt.tick)
			if err != nil || got != tt.want {
				t.Errorf("Expiring(a0, c%d, %d) = %d, %v, want %d, nil", tt.commodity, tt.tick, got, err, tt.want)
			}
			if after := f.state(); !reflect.DeepEqual(after, before) {
				t.Errorf("Expiring changed the ledger")
			}
		})
	}

	t.Run("errors", func(t *testing.T) {
		t.Parallel()
		f := setup(t)
		mustNil(f.l.Close(f.a1))
		for _, c := range []struct {
			name      string
			id, comm  int
			want      error
			wantValue int64
		}{
			{"unknown account", 99, cSpoil, ledger.ErrUnknownAccount, 0},
			{"closed account", f.a1, cSpoil, ledger.ErrClosedAccount, 0},
			{"unknown commodity", f.a0, 9, ledger.ErrUnknownCommodity, 0},
			{"estate holds none", f.e0, cSpoil, nil, 0},
		} {
			got, err := f.l.Expiring(c.id, c.comm, 100)
			if !errors.Is(err, c.want) || got != c.wantValue {
				t.Errorf("%s: Expiring(%d, %d, 100) = %d, %v, want %d, %v", c.name, c.id, c.comm, got, err, c.wantValue, c.want)
			}
		}
	})

	t.Run("sum over commodities is what ageing moves", func(t *testing.T) {
		t.Parallel()
		f := setup(t)
		f.l.EndTick() // current tick 1
		var want [nComm]int64
		for c := 0; c < nComm; c++ {
			n, err := f.l.Expiring(f.a0, c, 1)
			if err != nil {
				t.Fatal(err)
			}
			want[c] = n
		}
		// c0 4 and c3 2 by the lots above.
		if want != [nComm]int64{4, 0, 0, 2} {
			t.Fatalf("Expiring per commodity = %v, want [4 0 0 2]", want)
		}
		if err := f.l.Age(); err != nil {
			t.Fatal(err)
		}
		for c := 0; c < nComm; c++ {
			if got := f.balance(t, ledger.Spoilage, c); got != want[c] {
				t.Errorf("spoilage boundary of c%d = %d, want %d (Expiring)", c, got, want[c])
			}
		}
	})
}

// TestEndTick_TotalsAfterTheClearBelongToTheNextTick verifies that the tick
// end advances the counter then clears the totals, that a move made after it is
// in the next tick's totals, and that the construction bucket is untouched.
//
// Requirement: FR-LED-045, FR-LED-053
func TestEndTick_TotalsAfterTheClearBelongToTheNextTick(t *testing.T) {
	t.Parallel()
	f := newProd(t, ledger.NumReasons, -1)
	f.stated(t, f.a0, cPlain, 0, 0, 1)
	bucket := f.l.AppendFlows(nil, true)
	f.run(t, 0)
	if err := f.l.Move(f.initial[cPlain], f.a0, cPlain, 3, ledger.Initial); err != nil {
		t.Fatal(err)
	}
	f.l.EndTick()
	if f.l.Tick() != 1 {
		t.Errorf("Tick() = %d, want 1", f.l.Tick())
	}
	if got := f.l.AppendFlows(nil, false); len(got) != 0 {
		t.Errorf("totals after EndTick = %v, want none", got)
	}
	if err := f.l.Move(f.initial[cPlain], f.a1, cPlain, 2, ledger.Initial); err != nil {
		t.Fatal(err)
	}
	want := []ledger.Flow{{Region: 1, Reason: ledger.Initial, Commodity: cPlain, Total: 2}}
	if got := f.l.AppendFlows(nil, false); !reflect.DeepEqual(got, want) {
		t.Errorf("totals of the next tick = %v, want %v", got, want)
	}
	f.l.EndTick()
	if got := f.l.AppendFlows(nil, false); len(got) != 0 {
		t.Errorf("totals after the second EndTick = %v, want none", got)
	}
	if got := f.l.AppendFlows(nil, true); !reflect.DeepEqual(got, bucket) {
		t.Errorf("bucket = %v, want %v, unchanged", got, bucket)
	}
}
