package ledger

import (
	"errors"
	"math"
	"reflect"
	"testing"
)

// trRig is a ledger in construction with commodity 0 spoiling (shelf life 10)
// and wearing (durability 5), commodities 1 and 2 plain, two regions, the
// Initial, RecipeInput, RecipeOutput and Wear boundaries of every commodity,
// and an agent in each region.
type trRig struct {
	l      *Ledger
	a0, a1 int
}

// newTrRig opens the boundary of reason first and commodity 0 before every
// other boundary, so that it is account 0.
func newTrRig(t *testing.T, first Reason) *trRig {
	t.Helper()
	l, err := New(Config{
		Commodities: []Commodity{{ShelfLife: 10, Durability: 5}, {}, {}},
		Regions:     2,
		MaxMoney:    1000,
		MaxQuantity: 100,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	open := func(r Reason, c int) {
		t.Helper()
		if _, err := l.OpenBoundary(r, c); err != nil {
			t.Fatalf("OpenBoundary(%d, %d): %v", r, c, err)
		}
	}
	open(first, 0)
	for _, r := range []Reason{Initial, RecipeInput, RecipeOutput, Wear} {
		for c := 0; c < 3; c++ {
			if r != first || c != 0 {
				open(r, c)
			}
		}
	}
	g := &trRig{l: l}
	if g.a0, err = l.Open(KindAgent, 0, 0); err != nil {
		t.Fatalf("Open: %v", err)
	}
	if g.a1, err = l.Open(KindAgent, 0, 1); err != nil {
		t.Fatalf("Open: %v", err)
	}
	return g
}

// give credits qty units under key k to account id from the Initial boundary.
func (g *trRig) give(t *testing.T, id int, k Key, qty int64) {
	t.Helper()
	b, _ := g.l.Boundary(Initial, k.Commodity)
	if err := g.l.MoveStated(b, id, k, qty, Initial); err != nil {
		t.Fatalf("MoveStated(%+v, %d): %v", k, qty, err)
	}
}

// lots returns the lots of account id.
func (g *trRig) lots(t *testing.T, id int) []Lot {
	t.Helper()
	got, err := g.l.AppendLots(nil, id)
	if err != nil {
		t.Fatalf("AppendLots: %v", err)
	}
	return got
}

// TestTransformAndWear_BoundaryAccountWithIDZeroIsOpen verifies that a
// recipe_input, recipe_output or wear boundary account that happens to be
// account 0 is an opened boundary.
//
// Guard: account IDs start at 0, so the "never opened" marker must be
// negative, not zero.
func TestTransformAndWear_BoundaryAccountWithIDZeroIsOpen(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		first Reason
		op    func(*Ledger, int) error
	}{
		{"recipe_input", RecipeInput, func(l *Ledger, a int) error { return l.Transform(a, []Line{{0, 1}}, nil) }},
		{"recipe_output", RecipeOutput, func(l *Ledger, a int) error { return l.Transform(a, nil, []Line{{0, 1}}) }},
		{"wear", Wear, func(l *Ledger, a int) error { return l.Wear(a, 0, 1) }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			g := newTrRig(t, tc.first)
			if id, ok := g.l.Boundary(tc.first, 0); !ok || id != 0 {
				t.Fatalf("Boundary(%d, 0) = %d, %v, want 0, true", tc.first, id, ok)
			}
			g.give(t, g.a0, Key{Commodity: 0, Born: 0, Durability: 5}, 1)
			if err := tc.op(g.l, g.a0); err != nil {
				t.Errorf("operation on boundary account 0 = %v, want nil", err)
			}
		})
	}
}

// TestTransform_InputsAreCheckedInOrderOfFirstLine verifies that the
// commodities of the inputs are checked once each in the order of their first
// line: a short commodity named first is refused before the balance overflow
// of a commodity named between its two lines.
//
// Guard: the first line of a commodity is the one that carries its check.
func TestTransform_InputsAreCheckedInOrderOfFirstLine(t *testing.T) {
	t.Parallel()
	g := newTrRig(t, Initial)
	g.give(t, g.a0, Key{Commodity: 1}, 1)
	b, _ := g.l.Boundary(RecipeInput, 1)
	g.l.accounts[b].balance = math.MaxInt64
	// Commodity 2 is held 0 times: ErrInsufficientStock (check 6). Commodity 1
	// would overflow its boundary (check 7), and is named after commodity 2's
	// first line.
	err := g.l.Transform(g.a0, []Line{{2, 1}, {1, 1}, {2, 1}}, nil)
	if !errors.Is(err, ErrInsufficientStock) {
		t.Errorf("Transform = %v, want ErrInsufficientStock", err)
	}
}

// TestTransform_StockAndOverflowAreCheckedPerCommodity verifies that the
// stock and overflow checks of the inputs run one commodity at a time in the
// order of its first line: an overflowing commodity named before a short one
// is refused with ErrBalanceOverflow, and the reverse order with
// ErrInsufficientStock.
//
// Guard: the order is defined only by the doc comment of Transform; it decides
// which error a caller that has two faults sees, and nothing else pins it.
func TestTransform_StockAndOverflowAreCheckedPerCommodity(t *testing.T) {
	t.Parallel()
	g := newTrRig(t, Initial)
	g.give(t, g.a0, Key{Commodity: 1}, 1)
	b, _ := g.l.Boundary(RecipeInput, 1)
	g.l.accounts[b].balance = math.MaxInt64
	// Commodity 1 is held and would overflow its boundary by 1; commodity 2 is
	// not held at all.
	tests := []struct {
		name   string
		inputs []Line
		want   error
	}{
		{"overflowing commodity first", []Line{{1, 1}, {2, 1}}, ErrBalanceOverflow},
		{"short commodity first", []Line{{2, 1}, {1, 1}}, ErrInsufficientStock},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := g.l.Transform(g.a0, tt.inputs, nil); !errors.Is(err, tt.want) {
				t.Errorf("Transform(%v) = %v, want %v", tt.inputs, err, tt.want)
			}
		})
	}
}

// TestTransformAndWear_FlowsAreTotalledInTheRegionOfTheAccount verifies that
// the recipe and wear flows of an account in region 1 land in region 1.
//
// Requirement: FR-LED-053
func TestTransformAndWear_FlowsAreTotalledInTheRegionOfTheAccount(t *testing.T) {
	t.Parallel()
	g := newTrRig(t, Initial)
	g.give(t, g.a1, Key{Commodity: 1}, 4)
	g.give(t, g.a1, Key{Commodity: 0, Born: 0, Durability: 5}, 1)
	if err := g.l.Transform(g.a1, []Line{{1, 2}}, []Line{{2, 3}}); err != nil {
		t.Fatalf("Transform: %v", err)
	}
	// The one item of commodity 0 has durability 5, worn by 5: destroyed.
	if err := g.l.Wear(g.a1, 0, 5); err != nil {
		t.Fatalf("Wear: %v", err)
	}
	want := map[Flow]bool{
		{Region: 1, Reason: RecipeInput, Commodity: 1, Total: 2}:  true,
		{Region: 1, Reason: RecipeOutput, Commodity: 2, Total: 3}: true,
		{Region: 1, Reason: Wear, Commodity: 0, Total: 1}:         true,
	}
	got := map[Flow]bool{}
	for _, f := range g.l.AppendFlows(nil, true) {
		if f.Reason == RecipeInput || f.Reason == RecipeOutput || f.Reason == Wear {
			got[f] = true
		}
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("flows = %v, want %v", got, want)
	}
}

// TestTransform_InputsTakeOnlyUnearmarkedUnitsOfAPartlyEarmarkedLot verifies
// that an input drains the unearmarked part of the oldest lot and then the
// next lot.
//
// Requirement: FR-LED-034
func TestTransform_InputsTakeOnlyUnearmarkedUnitsOfAPartlyEarmarkedLot(t *testing.T) {
	t.Parallel()
	g := newTrRig(t, Initial)
	old, young := Key{Commodity: 0, Born: -1, Durability: 5}, Key{Commodity: 0, Born: 0, Durability: 5}
	g.give(t, g.a0, old, 3)
	g.give(t, g.a0, young, 5)
	if err := g.l.ReserveLot(g.a0, old, 2); err != nil {
		t.Fatalf("ReserveLot: %v", err)
	}
	if err := g.l.Transform(g.a0, []Line{{0, 4}}, nil); err != nil {
		t.Fatalf("Transform: %v", err)
	}
	// Old lot: 3 held, 2 earmarked, so 1 is taken (2 left, all earmarked); the
	// other 3 of the 4 come from the young lot (5 - 3 = 2).
	want := []Lot{
		{Commodity: 0, Quantity: 2, Born: -1, Durability: 5, Reserved: 2},
		{Commodity: 0, Quantity: 2, Born: 0, Durability: 5},
	}
	if got := g.lots(t, g.a0); !reflect.DeepEqual(got, want) {
		t.Errorf("lots = %+v, want %+v", got, want)
	}
}

// TestWearItems_SelectionSkipsEarmarkedAndLessWornLots verifies the lots
// selected: only the available units, of the lowest durability, a partly
// earmarked lot giving its unearmarked units only, and a fully earmarked lot
// of the selected durability giving nothing and leaving no empty lot.
//
// Requirement: FR-LED-040
func TestWearItems_SelectionSkipsEarmarkedAndLessWornLots(t *testing.T) {
	t.Parallel()
	k := func(born, dur int64) Key { return Key{Commodity: 0, Born: born, Durability: dur} }
	tests := []struct {
		name     string
		give     []Lot
		reserve  map[Key]int64
		n, units int64
		want     []Lot
	}{
		{
			// Old lot has 5 - 2 = 3 available, so 3 of the 4 come from it and 1
			// from the young lot; all go from durability 4 to 3.
			name:    "partly earmarked lot",
			give:    []Lot{{Quantity: 5, Born: -1, Durability: 4}, {Quantity: 5, Born: 0, Durability: 4}},
			reserve: map[Key]int64{k(-1, 4): 2},
			n:       4, units: 1,
			want: []Lot{
				{Commodity: 0, Quantity: 3, Born: -1, Durability: 3},
				{Commodity: 0, Quantity: 2, Born: -1, Durability: 4, Reserved: 2},
				{Commodity: 0, Quantity: 1, Born: 0, Durability: 3},
				{Commodity: 0, Quantity: 4, Born: 0, Durability: 4},
			},
		},
		{
			// The older lot has durability 5, the younger 2: the younger is the
			// most worn and the only one worn for n = 1.
			name: "most worn first although younger",
			give: []Lot{{Quantity: 2, Born: -1, Durability: 5}, {Quantity: 2, Born: 0, Durability: 2}},
			n:    1, units: 1,
			want: []Lot{
				{Commodity: 0, Quantity: 2, Born: -1, Durability: 5},
				{Commodity: 0, Quantity: 1, Born: 0, Durability: 1},
				{Commodity: 0, Quantity: 1, Born: 0, Durability: 2},
			},
		},
		{
			// The older lot is fully earmarked at the same durability as the
			// available younger one: it is not worn and no lot of 0 appears.
			name:    "fully earmarked lot of the selected durability",
			give:    []Lot{{Quantity: 2, Born: -1, Durability: 4}, {Quantity: 2, Born: 0, Durability: 4}},
			reserve: map[Key]int64{k(-1, 4): 2},
			n:       1, units: 1,
			want: []Lot{
				{Commodity: 0, Quantity: 2, Born: -1, Durability: 4, Reserved: 2},
				{Commodity: 0, Quantity: 1, Born: 0, Durability: 3},
				{Commodity: 0, Quantity: 1, Born: 0, Durability: 4},
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			g := newTrRig(t, Initial)
			for _, lt := range tc.give {
				g.give(t, g.a0, k(lt.Born, lt.Durability), lt.Quantity)
			}
			for key, q := range tc.reserve {
				if err := g.l.ReserveLot(g.a0, key, q); err != nil {
					t.Fatalf("ReserveLot: %v", err)
				}
			}
			if err := g.l.WearItems(g.a0, 0, tc.n, tc.units); err != nil {
				t.Fatalf("WearItems: %v", err)
			}
			if got := g.lots(t, g.a0); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("lots = %+v, want %+v", got, tc.want)
			}
		})
	}
}

// TestWearUnits_CommodityRange verifies that the first and last commodity are
// known, one before the first and one past the last are
// ErrUnknownCommodity, and that the units are read from the phase asked for.
//
// Requirement: FR-LED-042
func TestWearUnits_CommodityRange(t *testing.T) {
	t.Parallel()
	g := newTrRig(t, Initial)
	g.give(t, g.a0, Key{Commodity: 0, Born: 0, Durability: 5}, 1)
	// In construction, so the 3 units removed go to the bucket.
	if err := g.l.Wear(g.a0, 0, 3); err != nil {
		t.Fatalf("Wear: %v", err)
	}
	tests := []struct {
		name      string
		commodity int
		bucket    bool
		want      int64
		wantErr   error
	}{
		{"first, bucket", 0, true, 3, nil},
		{"first, tick", 0, false, 0, nil},
		{"last, bucket", 2, true, 0, nil},
		{"before the first", -1, true, 0, ErrUnknownCommodity},
		{"one past the last", 3, false, 0, ErrUnknownCommodity},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := g.l.WearUnits(tc.commodity, tc.bucket)
			if !errors.Is(err, tc.wantErr) || got != tc.want {
				t.Errorf("WearUnits(%d, %v) = %d, %v, want %d, %v", tc.commodity, tc.bucket, got, err, tc.want, tc.wantErr)
			}
		})
	}
}
