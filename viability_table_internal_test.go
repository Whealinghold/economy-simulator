package economysim

import (
	"errors"
	"reflect"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/steady"
)

// This file tests the pieces of the warm-start table (spec-viability.md 6 and 8)
// on literals: the seed price, the main recipe and the money, the stock and the
// lean reserve. The whole table of a solved chain is viability_solve_internal_test.go's.

// seedConfig is a configuration with the defaults of the world only: the
// resolution floor is ceil(10 000 / 200) = 50 and the reference band runs from
// 1 000 to 100 000 basis points of the reference price (a tenth to ten times).
func seedConfig() *viaSolve {
	cfg := Config{World: DefaultWorld()}
	return &viaSolve{m: &viaModel{c: &cfg}}
}

// TestSeedPrice verifies the rounding, the floor, the cap and the band of a
// seed (6.3, 6.4, O11): half to even, raised to the resolution floor, capped at
// MaxPrice, and a floored or free price lifted into the reference band.
//
// Requirement: FR-VIA-023, FR-VIA-026 (dual-0 half; non-unique half N-311), FR-VIA-030
func TestSeedPrice(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		dual, scale steady.Frac
		ref         Price
		seed, raw   int64
		lifted      bool
	}{
		// 205/2 = 102.5 is an exact half: the even neighbour is 102. The band of
		// ref 100 is 10 to 1 000, so nothing else acts.
		{"half to even down", steady.Rat(205, 2), steady.Rat(1, 1), 100, 102, 102, false},
		// 207/2 = 103.5: the even neighbour is 104.
		{"half to even up", steady.Rat(207, 2), steady.Rat(1, 1), 100, 104, 104, false},
		// 10 is below the floor 50 and inside the band 10 to 1 000: the floor binds.
		{"floor binds", steady.Rat(10, 1), steady.Rat(1, 1), 100, 50, 50, true},
		// A free good (dual 0) of ref 1 000: floored to 50, then lifted to the band's
		// lower bound 1 000 / 10 = 100; the price before the band is 50.
		{"free good lifted into the band", steady.Rat(0, 1), steady.Rat(1, 1), 1_000, 100, 50, true},
		// 20 000 against ref 100: above the band's upper bound 1 000, capped to it.
		{"capped to the band", steady.Rat(20_000, 1), steady.Rat(1, 1), 100, 1_000, 20_000, false},
		// 2 * 10^9 is above MaxPrice: the price before the band is MaxPrice, and
		// ref 100 caps it to 1 000.
		{"above MaxPrice", steady.Rat(2_000_000_000, 1), steady.Rat(1, 1), 100, 1_000, MaxPrice, false},
		// The same against ref MaxPrice, whose band reaches 10^10: MaxPrice stays.
		{"MaxPrice stays", steady.Rat(3_000_000_000, 1), steady.Rat(1, 1), MaxPrice, MaxPrice, MaxPrice, false},
		// 1/3 * 3 = 1 exactly: floor 50.
		{"scale", steady.Rat(1, 3), steady.Rat(3, 1), 100, 50, 50, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cm := Commodity{ID: "x", ReferencePrice: tt.ref}
			seed, raw, lifted, err := seedConfig().seedPrice(tt.dual, tt.scale, &cm)
			if err != nil || seed != tt.seed || raw != tt.raw || lifted != tt.lifted {
				t.Errorf("seedPrice(%s) = %d, %d, %v, %v, want %d, %d, %v, nil", tt.name, seed, raw, lifted, err, tt.seed, tt.raw, tt.lifted)
			}
		})
	}
}

// TestDriftOf verifies the drift of a seed from its reference price in basis
// points, rounded toward zero for a seed below and above it.
//
// Requirement: FR-VIA-025
func TestDriftOf(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		seed, ref int64
		want      int64
	}{
		{"equal", 100, 100, 0},
		// (85 - 80) * 10 000 / 80 = 625.
		{"above", 85, 80, 625},
		// (50 - 80) * 10 000 / 80 = -3 750.
		{"below", 50, 80, -3_750},
		// (101 - 3) * 10 000 / 3 = 326 666.67, toward zero.
		{"rounds toward zero", 101, 3, 326_666},
		// (3 - 101) * 10 000 / 101 = -9 702.97, toward zero.
		{"negative rounds toward zero", 3, 101, -9_702},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := driftOf(RegionCommodity{"r", "c"}, tt.seed, tt.ref).DriftBP; got != tt.want {
				t.Errorf("driftOf(%d, %d) = %d, want %d", tt.seed, tt.ref, got, tt.want)
			}
		})
	}
}

// moneyContext returns a region context whose activities have the given
// durations and inputs: the commodity a priced 100 and b priced 1 000, so an
// input of 2 of a costs 200 and 5 of b 5 000.
func moneyContext(acts ...steady.Activity) (*regionCtx, *groupCtx) {
	cfg := &Config{Roles: []Role{{ID: "fabber"}}}
	rc := &regionCtx{s: &viaSolve{m: &viaModel{c: cfg}}, id: "r", price: []int64{100, 1_000}}
	gc := &groupCtx{up: []int64{1, 0}}
	for i, a := range acts {
		rc.acts = append(rc.acts, viaActivity{act: a})
		gc.active = append(gc.active, activeRecipe{i: i})
	}
	return rc, gc
}

// TestMainRecipe verifies the main recipe of a group (8.3.1, JC3): the active
// recipe with the longest duration, ties by the largest inputs cost, then by the
// lower recipe ID; and that the longest duration decides over a dearer input
// (the `second` fabber, 19 423 and not 52 189).
//
// Requirement: FR-VIA-032
func TestMainRecipe(t *testing.T) {
	t.Parallel()
	in := func(c int, q int64) []steady.Item { return []steady.Item{{Commodity: c, Qty: q}} }
	tests := []struct {
		name       string
		acts       []steady.Activity
		main       int
		dmax, cost int64
	}{
		// board takes 3 ticks and 2 of a (200); the fabricator takes 1 tick and 5 of
		// b (5 000): the longest duration wins, though its inputs are cheaper.
		{"longest duration over dearer inputs", []steady.Activity{{Duration: 3, Inputs: in(0, 2)}, {Duration: 1, Inputs: in(1, 5)}}, 0, 3, 200},
		// Equal durations: 1 of b (1 000) is dearer than 2 of a (200).
		{"tie in duration falls to the dearer inputs", []steady.Activity{{Duration: 3, Inputs: in(0, 2)}, {Duration: 3, Inputs: in(1, 1)}}, 1, 3, 1_000},
		// Equal in both: the lower recipe ID, the earlier of the active list.
		{"tie in both falls to the lower ID", []steady.Activity{{Duration: 3, Inputs: in(0, 2)}, {Duration: 3, Inputs: in(0, 2)}}, 0, 3, 200},
		// The longest comes last: 4 beats 3 and 1.
		{"longest last", []steady.Activity{{Duration: 3, Inputs: in(1, 5)}, {Duration: 1}, {Duration: 4, Inputs: in(0, 1)}}, 2, 4, 100},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			rc, gc := moneyContext(tt.acts...)
			main, dmax, cost, err := rc.mainRecipe(gc)
			if err != nil || main != tt.main || dmax != tt.dmax || cost != tt.cost {
				t.Errorf("mainRecipe(%s) = %d, %d, %d, %v, want %d, %d, %d, nil", tt.name, main, dmax, cost, err, tt.main, tt.dmax, tt.cost)
			}
		})
	}
	rc, gc := moneyContext()
	if main, dmax, cost, err := rc.mainRecipe(gc); err != nil || main != -1 || dmax != 0 || cost != 0 {
		t.Errorf("mainRecipe(no active recipe) = %d, %d, %d, %v, want -1, 0, 0, nil", main, dmax, cost, err)
	}
}

// TestMoney verifies the money of a household (8.3.1): the inputs of the main
// recipe plus the upkeep for the longest job and the buffer, capped at
// MaxMoney, and ErrOverflow when a product leaves int64.
//
// Requirement: FR-VIA-032
func TestMoney(t *testing.T) {
	t.Parallel()
	t.Run("composition", func(t *testing.T) {
		t.Parallel()
		// Inputs 200 (2 of a) + U * (d_max + buffer) = 200 + 100 * (3 + 10) = 1 500.
		rc, gc := moneyContext(steady.Activity{Duration: 3, Inputs: []steady.Item{{Commodity: 0, Qty: 2}}})
		rc.s.o.moneyBuffer = 10
		if got, err := rc.money(gc); err != nil || got != 1_500 {
			t.Errorf("money = %d, %v, want 1 500 (200 + 100 * 13)", got, err)
		}
	})
	t.Run("capped at MaxMoney", func(t *testing.T) {
		t.Parallel()
		// U = 5 * 10^17, no active recipe, buffer 10: 5 * 10^18 fits int64 and is
		// above MaxMoney 10^18.
		rc, gc := moneyContext()
		rc.price[0], rc.s.o.moneyBuffer = MaxMoney/2, 10
		if got, err := rc.money(gc); err != nil || got != MaxMoney {
			t.Errorf("money = %d, %v, want MaxMoney %d", got, err, int64(MaxMoney))
		}
	})
	t.Run("overflow of the upkeep", func(t *testing.T) {
		t.Parallel()
		rc, gc := moneyContext()
		gc.up[0], rc.price[0] = 1<<40, 1<<40
		if _, err := rc.money(gc); !errors.Is(err, ErrOverflow) {
			t.Errorf("money with an upkeep of 2^80 = %v, want ErrOverflow", err)
		}
	})
	t.Run("overflow of the span", func(t *testing.T) {
		t.Parallel()
		rc, gc := moneyContext()
		rc.price[0], rc.s.o.moneyBuffer = 1<<62, 10_000
		if _, err := rc.money(gc); !errors.Is(err, ErrOverflow) {
			t.Errorf("money with U * span above int64 = %v, want ErrOverflow", err)
		}
	})
}

// stockContext returns a region context with three commodities (bread, grain,
// plough) for the stock tests: bread spoils in 4 ticks, the upkeep is 1 bread
// and 2 grain, and the one active recipe takes 20 grain and wears a plough.
func stockContext() (*regionCtx, *groupCtx) {
	cfg := &Config{
		Commodities: []Commodity{{ID: "bread", ShelfLife: 4}, {ID: "grain", ShelfLife: 700}, {ID: "plough"}},
		Roles:       []Role{{ID: "grower"}},
	}
	rc := &regionCtx{s: &viaSolve{m: &viaModel{c: cfg}}, id: "r", price: make([]int64, 3), table: []int{0, 1, 2}}
	rc.s.o.stockCover = 5
	rc.acts = []viaActivity{{act: steady.Activity{
		Inputs: []steady.Item{{Commodity: 1, Qty: 20}}, Tools: []steady.Tool{{Commodity: 2, Wear: 1, Durability: 10}},
	}}}
	gc := &groupCtx{up: []int64{1, 2, 0}, reserve: []int64{0, 644, 0}, active: []activeRecipe{{i: 0}}}
	return rc, gc
}

// TestStock verifies the composition of the stock of a household (8.4.1, D134
// (2)): the larger of the cover and the inputs, plus the reserve and one item
// of each tool, and the cut of the cover to the shelf life (8.4.2).
//
// Requirement: FR-VIA-039
func TestStock(t *testing.T) {
	t.Parallel()
	rc, gc := stockContext()
	got, err := rc.stock(gc)
	// bread: cover 5 * 1 = 5 is cut to the 4 ticks it lasts, 4 * 1 = 4.
	// grain: max(cover 5 * 2 = 10, input 20) + reserve 644 = 664 (the grower of `one`).
	// plough: cover 0, input 0, one item of the tool = 1.
	want := []Item{{Commodity: "bread", Quantity: 4}, {Commodity: "grain", Quantity: 664}, {Commodity: "plough", Quantity: 1}}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Errorf("stock = %v, %v, want %v, nil", got, err, want)
	}
	if want := []RegionCommodity{{"r", "bread"}}; !reflect.DeepEqual(rc.s.rep.CapShortened, want) {
		t.Errorf("CapShortened = %v, want %v: only the bread cover was cut", rc.s.rep.CapShortened, want)
	}
	// A cover of 5 * 2^62 leaves int64: ErrOverflow, not a wrapped number.
	rc, gc = stockContext()
	gc.up[1] = 1 << 62
	if _, err := rc.stock(gc); !errors.Is(err, ErrOverflow) {
		t.Errorf("stock with a cover of 5 * 2^62 = %v, want ErrOverflow", err)
	}
	// A reserve that overflows the sum is ErrOverflow too.
	rc, gc = stockContext()
	gc.reserve[1] = 1<<63 - 1
	if _, err := rc.stock(gc); !errors.Is(err, ErrOverflow) {
		t.Errorf("stock with a reserve of MaxInt64 = %v, want ErrOverflow", err)
	}
}

// TestReserveTotal verifies the lean reserve of a commodity (8.4.1, D123) on a
// spring crop: the whole use of 10 a tick until the first completion at tick 118
// (start 59 plus 60 minus 1), 1 180, where the production then covers it; and the
// cut to what the region uses within the shelf life (8.4.2).
//
// Requirement: FR-VIA-039
func TestReserveTotal(t *testing.T) {
	t.Parallel()
	runs := []steady.Run{{Length: 59}, {Length: 92, FactorBP: 10_000}, {Length: 214}}
	act := steady.Activity{Duration: 60, Starts: 2, FactorSum: 20_000, Outputs: []steady.Item{{Commodity: 0, Qty: 1_800}}}
	tests := []struct {
		name  string
		shelf int64
		total int64
		cut   bool
	}{
		// 10 a tick for the ticks 0 to 117: 118 * 10 = 1 180; the harvest of 92 ticks
		// of 0.5 * 365 / 92 * 1 800 = 3 570 a tick (the flow is 1/2) brings the level
		// far below 0 and it never rises above 1 180 again.
		{"no shelf life", 0, 1_180, false},
		// The region uses 10 a tick; 700 ticks of it is 7 000, above 1 180.
		{"shelf life above the reserve", 700, 1_180, false},
		// 5 ticks of use is 50: the reserve would spoil before it was used.
		{"cut to the shelf life", 5, 50, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg := &Config{Commodities: []Commodity{{ID: "g", ShelfLife: tt.shelf}}}
			m := &viaModel{c: cfg, year: 365, upkeep: [][]int64{{10}}}
			p := &viaProgramme{base: []int{0}, out: steady.Outcome{Flow: []steady.Frac{steady.Rat(1, 2)}, Active: []bool{true}}}
			rc := &regionCtx{s: &viaSolve{m: m}, p: p, acts: []viaActivity{{act: act, runs: runs}}}
			total, cut := rc.reserveTotal(0)
			if total != tt.total || cut != tt.cut {
				t.Errorf("reserveTotal(shelf %d) = %d, %v, want %d, %v", tt.shelf, total, cut, tt.total, tt.cut)
			}
		})
	}
}

// TestCeilDiv verifies the per-household rounding of a lean reserve share: up,
// so the group's households together hold at least the share (8.4.1, D123).
//
// Requirement: FR-VIA-039
func TestCeilDiv(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		n, d int64
		want int64
	}{
		{"zero", 0, 3, 0},
		{"exact", 6, 3, 2},
		// 7 = 2 * 3 + 1: the remainder 1 rounds up to 3.
		{"1 mod 3", 7, 3, 3},
		// 8 = 2 * 3 + 2: the remainder 2 rounds up to 3.
		{"2 mod 3", 8, 3, 3},
		{"one household", 7, 1, 7},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := ceilDiv(tt.n, tt.d); got != tt.want {
				t.Errorf("ceilDiv(%d, %d) = %d, want %d", tt.n, tt.d, got, tt.want)
			}
		})
	}
}
