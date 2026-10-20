package steady_test

import (
	"math"
	"math/big"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/steady"
)

// earthSpring is the default year of 365 ticks with spring on days 59 to 150.
var earthSpring = []steady.Run{{Length: 59, FactorBP: 0}, {Length: 92, FactorBP: 10000}, {Length: 214, FactorBP: 0}}

// centreSeasons is twoPlantings with both windows at factor 8 000.
var centreSeasons = []steady.Run{{Length: 59}, {Length: 92, FactorBP: 8000}, {Length: 92}, {Length: 92, FactorBP: 8000}, {Length: 30}}

// twoPlantings adds an autumn window on days 243 to 334 (starts 243 and 303).
var twoPlantings = []steady.Run{{Length: 59}, {Length: 92, FactorBP: 10000}, {Length: 92}, {Length: 92, FactorBP: 10000}, {Length: 30}}

// TestGap verifies the supply gap of D122: the greedy completions, the wrap of a
// late completion, the all-year exemption and the union of producers.
//
// Requirement: FR-VIA-008
func TestGap(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		year int64
		ps   []steady.Producer
		want int64
	}{
		// Starts 59 and 119, d 60: completions 118 and 178. The run across the year
		// end is 118 + 365 - 178 - 1 = 304; all open ticks as completions would give
		// 365 - 92 = 273, which is not the rule.
		{"spring only d 60", 365, []steady.Producer{steady.NewProducer(60, earthSpring)}, 304},
		// Completions 118, 178, 302, 362: 178 to 302 is 302 - 178 - 1 = 123; across the
		// year end 118 + 365 - 362 - 1 = 120.
		{"two plantings", 365, []steady.Producer{steady.NewProducer(60, twoPlantings)}, 123},
		// One start at 340 completes at 399 = tick 34 of the next year: one completion
		// a year, gap 34 + 365 - 34 - 1 = 364.
		{"late window wraps", 365, []steady.Producer{steady.NewProducer(60, []steady.Run{{Length: 340}, {Length: 25, FactorBP: 10000}})}, 364},
		// Two activities, one planting each: the union is the two plantings.
		{"union of two producers", 365, []steady.Producer{
			{Completions: []int64{118, 178}}, {Completions: []int64{302, 362}}}, 123},
		{"all year d 2", 365, []steady.Producer{steady.NewProducer(2, []steady.Run{{Length: 365, FactorBP: 10000}})}, 0},
		{"all year d 12", 365, []steady.Producer{steady.NewProducer(12, []steady.Run{{Length: 365, FactorBP: 10000}})}, 0},
		// One all-year producer among others makes no gap.
		{"one every-tick producer", 365, []steady.Producer{{Completions: []int64{5}}, {Every: true}}, 0},
		{"nothing completes", 365, []steady.Producer{steady.NewProducer(60, []steady.Run{{Length: 365}})}, 365},
		{"no producers", 365, nil, 365},
		{"zero-length year", 0, []steady.Producer{{Every: false, Completions: []int64{1}}}, 0},
		// Completions 10 and 20 of a year of 100: across the end 10 + 100 - 20 - 1 = 89.
		{"run across the year end", 100, []steady.Producer{{Completions: []int64{10, 20}}}, 89},
		// 10, 20 and 80: gaps 9, 59 and 10 + 100 - 80 - 1 = 29; the middle one wins.
		{"run inside the year", 100, []steady.Producer{{Completions: []int64{80, 10, 20, 10}}}, 59},
		// Completions 5 and 150 of a year of 100 are 5 and 50 modulo the year: gaps 44
		// and 5 + 100 - 50 - 1 = 54 (without the reduction the gap would be 144).
		{"modulo the year", 100, []steady.Producer{{Completions: []int64{5, 150}}}, 54},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := steady.Gap(tt.year, tt.ps); got != tt.want {
				t.Errorf("Gap(%d, %s) = %d, want %d", tt.year, tt.name, got, tt.want)
			}
		})
	}
}

// TestExceedsShelf verifies the storage_gap condition: a shelf life of 200
// against a gap of 304 fails, 700 passes, and 0 (non-perishable) never fails.
//
// Requirement: FR-VIA-008
func TestExceedsShelf(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		gap, shelf int64
		want       bool
	}{
		{"shelf 200 against 304", 304, 200, true},
		{"shelf 700 against 304", 304, 700, false},
		{"shelf equal to the gap", 304, 304, false},
		{"one below the gap", 304, 303, true},
		{"non-perishable", 304, 0, false},
		{"no gap", 0, 5, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := steady.ExceedsShelf(tt.gap, tt.shelf); got != tt.want {
				t.Errorf("ExceedsShelf(%d, %d) = %v, want %v", tt.gap, tt.shelf, got, tt.want)
			}
		})
	}
}

// fr builds a Frac n/d.
func fr(n, d int64) steady.Frac { return steady.Frac{Num: big.NewInt(n), Den: big.NewInt(d)} }

// TestReserve verifies the lean reserve of D123 on hand years, with the
// cumulative shortfall in each comment.
//
// Requirement: FR-VIA-039
func TestReserve(t *testing.T) {
	t.Parallel()
	// Ten ticks, ticks 4 and 5 open. d 1: a start at t completes at t. N_open 2, so
	// each open tick starts flow * 10 / 2 = 5 executions, each yielding Yield.
	short := []steady.Run{{Length: 4}, {Length: 2, FactorBP: 10000}, {Length: 4}}
	tests := []struct {
		name string
		year int64
		cons steady.Frac
		srcs []steady.Source
		want string
	}{
		// No source: the whole year is consumed, 10 * 5/2 = 25.
		{"no production", 10, fr(5, 2), nil, "25"},
		// cons 2: ticks 0..3 add 8; ticks 4 and 5 produce 5 * 2 = 10, delta -8 each, so
		// the level is 0 then -8; ticks 6..9 add 8 for a final 0. The peak is 8 at tick 3.
		{"first harvest is the peak", 10, fr(2, 1), []steady.Source{{Duration: 1, Flow: fr(1, 1), Yield: 2, Year: short}}, "8"},
		// cons 2, Yield 1: production 5 a tick, delta -3: 8, 2, then 10 at tick 9.
		{"year end is the peak", 10, fr(2, 1), []steady.Source{{Duration: 1, Flow: fr(1, 1), Yield: 1, Year: short}}, "10"},
		// Factor 5 000: production 2.5 * 2 = 5 a tick, same as the case above.
		{"half factor", 10, fr(2, 1), []steady.Source{{Duration: 1, Flow: fr(1, 1), Yield: 2, Year: []steady.Run{{Length: 4}, {Length: 2, FactorBP: 5000}, {Length: 4}}}}, "10"},
		// Open ticks 8 and 9, d 3: completions at 10 and 11, past the year: nothing is
		// credited though N_open counts both, so the reserve is the whole 10 * 1.
		{"completion past the year adds nothing", 10, fr(1, 1), []steady.Source{{Duration: 3, Flow: fr(1, 1), Yield: 9, Year: []steady.Run{{Length: 8}, {Length: 2, FactorBP: 10000}}}}, "10"},
		// A source with no flow produces nothing.
		{"zero flow", 10, fr(1, 1), []steady.Source{{Duration: 1, Flow: fr(0, 1), Yield: 9, Year: short}}, "10"},
		// Spring on an Earth year: the peak is 118 000 at tick 117 (production
		// 365 * 1 000 / 92 = 3 967.39 a tick afterwards exceeds the 1 000 used).
		{"first harvest of one", 365, fr(1000, 1), []steady.Source{{Duration: 60, Flow: fr(1, 1), Yield: 1000, Year: earthSpring}}, "118000"},
		// Spring and autumn, Yield 184: 365 * 184 / 184 = 365 a start tick, cons 200.
		// First peak 118 * 200 = 23 600 at tick 117. Spring (118..209, 92 ticks) nets
		// 92 * (200 - 365) = -15 180, 210..301 adds 92 * 200 = 18 400: level at 301 =
		// 23 600 - 15 180 + 18 400 = 26 820 > 23 600, the case of `centre` in `five`.
		{"second peak above the first", 365, fr(200, 1), []steady.Source{{Duration: 60, Flow: fr(1, 1), Yield: 184, Year: twoPlantings}}, "26820"},
		// Open ticks 7..9, d 2: completions 8, 9, 10; N_open 3, so 10 * 1 / 3 = 10/3 a
		// tick at Yield 1. The last span [9, 10+1) is clipped at the year end 10 (the
		// run is 3 long, so hi = 7+3+2-1 = 11 clips to 10). Ticks 0..7 add 8 * 5 = 40,
		// ticks 8 and 9 add 2 * (5 - 10/3) = 10/3: 130/3. Unclipped, a tick 10 adds 5/3 more: 45.
		{"span straddling the year end is clipped", 10, fr(5, 1), []steady.Source{{Duration: 2, Flow: fr(1, 1), Yield: 1, Year: []steady.Run{{Length: 7}, {Length: 3, FactorBP: 10000}}}}, "130/3"},
		// famine: 1 350 a tick for the 118 ticks to the first harvest: 1 350 * 118 = 159 300
		// (spec-viability 8.4.1; 159 300 / 417 households = 382.01, rounded up in Phase 6).
		{"famine", 365, fr(1350, 1), []steady.Source{{Duration: 60, Flow: fr(1, 1), Yield: 1000, Year: earthSpring}}, "159300"},
		// two south: the first harvest at tick 118 is the peak, 118 * 10995929/11800 =
		// 109 959.29; the autumn windows (243, 303) complete later and do not exceed it.
		{"two south", 365, fr(10995929, 11800), []steady.Source{{Duration: 60, Flow: fr(1, 1), Yield: 1000, Year: twoPlantings}}, "10995929/100"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := steady.Reserve(tt.year, tt.cons, tt.srcs)
			want, _ := new(big.Rat).SetString(tt.want)
			if new(big.Rat).SetFrac(got.Num, got.Den).Cmp(want) != 0 {
				t.Errorf("Reserve(%s) = %s/%s, want %s", tt.name, got.Num, got.Den, tt.want)
			}
		})
	}
}

// TestReserveCentreOfFive pins the pack figure where the cumulative maximum is
// not at the first harvest.
//
// Requirement: FR-VIA-039
func TestReserveCentreOfFive(t *testing.T) {
	t.Parallel()
	// cons 16200/7 = 2 314.29 a tick; the first harvest at tick 118 leaves 118 *
	// 16200/7 = 273 085.71 at tick 117. Flow 2908665108/1000000 at Yield 1 makes the
	// spring/autumn rate 365 * 0.8 * flow / 184 = 4 615.925 a tick, which the spring
	// harvest (92 ticks) nets down by 211 750.82; 210..301 use 92 * 16200/7 =
	// 212 914.29, so the level at tick 301 is 274 249.18 (spec-viability 8.4.1).
	srcs := []steady.Source{{Duration: 60, Flow: fr(2908665108, 1000000), Yield: 1, Year: centreSeasons}}
	got := steady.Reserve(365, fr(16200, 7), srcs)
	g, _ := new(big.Rat).SetFrac(got.Num, got.Den).Float64()
	if math.Abs(g-274249.18) > 0.005 {
		t.Errorf("Reserve(centre) = %.4f, want 274249.18", g)
	}
}
