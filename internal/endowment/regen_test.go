package endowment_test

import (
	"math"
	"math/big"
	"math/rand/v2"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/endowment"
)

// bigNumerator is the oracle of the numerator of one tick (spec-resources.md
// 4.2), written with math/big and no shared code with the package.
func bigNumerator(p endowment.Params, stock, season int64) *big.Int {
	n := new(big.Int)
	m, s, rate := big.NewInt(p.ScaleBP), big.NewInt(season), big.NewInt(p.Rate)
	switch p.Shape {
	case endowment.Constant:
		n.Mul(rate, m).Mul(n, s).Mul(n, big.NewInt(10_000))
	case endowment.Refill:
		if stock < p.Capacity {
			n.Mul(rate, big.NewInt(p.Capacity-stock)).Mul(n, m).Mul(n, s)
		}
	case endowment.Logistic:
		sp := max(stock, min(p.Refuge, p.Capacity-1))
		if sp > 0 && sp < p.Capacity {
			w := new(big.Int).Mul(big.NewInt(sp), big.NewInt(p.Capacity-sp))
			c := big.NewInt(p.Capacity)
			w.Add(w, new(big.Int).Sub(c, big.NewInt(1))).Div(w, c) // ceiling
			n.Mul(rate, w).Mul(n, m).Mul(n, s)
		}
	}
	return n
}

func denom(tpy int64) *big.Int {
	return new(big.Int).Mul(big.NewInt(1_000_000_000_000), big.NewInt(tpy))
}

// TestRegen_NumeratorsOfTheThreeShapes verifies the integer rule of each shape
// on a 365-tick year (D = 3.65e14): the units added and the carry left.
//
// Requirement: FR-RES-008
func TestRegen_NumeratorsOfTheThreeShapes(t *testing.T) {
	t.Parallel()
	const tpy = 365
	tests := []struct {
		name string
		p    endowment.Params
		in   endowment.Step
		want endowment.Result
	}{
		// 365 units a year at m = s = 10 000: n = 365 * 10^4 * 10^4 * 10^4 = 3.65e14 = D,
		// exactly one unit a tick and no carry.
		{"constant one unit a tick",
			endowment.Params{Shape: endowment.Constant, Rate: 365, Capacity: 100, ScaleBP: 10_000, TicksPerYear: tpy},
			endowment.Step{Stock: 10, SeasonBP: 10_000}, endowment.Result{Stock: 11, Added: 1}},
		// worked.out A0, first autumn tick: refill 10 000 bp, C = 2 000, S = 1 320, s = 20 000:
		// n = 10 000 * 680 * 10 000 * 20 000 = 1.36e15; 1.36e15 / 3.65e14 = 3 rem 2.65e14.
		{"refill first autumn tick of the arable stock",
			endowment.Params{Shape: endowment.Refill, Rate: 10_000, Capacity: 2000, ScaleBP: 10_000, TicksPerYear: tpy},
			endowment.Step{Stock: 1320, SeasonBP: 20_000},
			endowment.Result{Stock: 1323, Carry: 265_000_000_000_000, Added: 3}},
		// worked.out A0, second tick: n = 10 000 * 677 * 10 000 * 20 000 = 1.354e15, plus the
		// carry 2.65e14 = 1.619e15; 1.619e15 / 3.65e14 = 4 rem 1.59e14.
		{"refill second autumn tick carries the remainder",
			endowment.Params{Shape: endowment.Refill, Rate: 10_000, Capacity: 2000, ScaleBP: 10_000, TicksPerYear: tpy},
			endowment.Step{Stock: 1323, Carry: 265_000_000_000_000, SeasonBP: 20_000},
			endowment.Result{Stock: 1327, Carry: 159_000_000_000_000, Added: 4}},
		// logistic 3 000 bp, C = 50 000, S = 20 000: w = ceil(20 000 * 30 000 / 50 000) = 12 000;
		// n = 3 000 * 12 000 * 10^4 * 10^4 = 3.6e15; 3.6e15 / 3.65e14 = 9 rem 3.15e14.
		{"logistic at 20 000 of 50 000",
			endowment.Params{Shape: endowment.Logistic, Rate: 3000, Capacity: 50_000, ScaleBP: 10_000, TicksPerYear: tpy},
			endowment.Step{Stock: 20_000, SeasonBP: 10_000},
			endowment.Result{Stock: 20_009, Carry: 315_000_000_000_000, Added: 9}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := endowment.Regen(tt.p, tt.in)
			if !ok || got != tt.want {
				t.Errorf("Regen(%+v, %+v) = %+v, %v, want %+v, true", tt.p, tt.in, got, ok, tt.want)
			}
		})
	}
}

// TestRegen_LogisticWRoundsUp verifies the logistic factor is rounded up, so a
// stock of one unit, and one unit below the capacity, both regrow where
// rounding down would leave them stuck.
//
// Requirement: FR-RES-009
func TestRegen_LogisticWRoundsUp(t *testing.T) {
	t.Parallel()
	// TPY 1, rate 10 000 bp (a whole a year), m = s = 10 000: D = 10^12 and
	// n = 10^4 * w * 10^8 = w * 10^12, so w units are added.
	p := endowment.Params{Shape: endowment.Logistic, Rate: 10_000, Capacity: 1_000_000, ScaleBP: 10_000, TicksPerYear: 1}
	tests := []struct {
		name  string
		stock int64
		want  int64
	}{
		// w = ceil(1 * 999 999 / 1 000 000) = 1; floor would be 0.
		{"one unit regrows", 1, 2},
		// w = ceil(999 999 * 1 / 1 000 000) = 1; floor would be 0.
		{"one below the capacity regrows to it", 999_999, 1_000_000},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := endowment.Regen(p, endowment.Step{Stock: tt.stock, SeasonBP: 10_000})
			if !ok || got.Stock != tt.want {
				t.Errorf("Regen(%+v, stock %d) = %+v, %v, want stock %d", p, tt.stock, got, ok, tt.want)
			}
		})
	}
}

// TestRegen_CarryIsExact verifies that over 400 random runs the units added are
// exactly floor((carry0 + sum of numerators) / D) and the carry is the
// remainder, against a math/big oracle, whatever the draws between ticks did to
// the stock.
//
// Requirement: FR-RES-010
func TestRegen_CarryIsExact(t *testing.T) {
	t.Parallel()
	rng := rand.New(rand.NewPCG(11, 7))
	shapes := []endowment.Shape{endowment.Constant, endowment.Refill, endowment.Logistic}
	tpys := []int64{1, 12, 365, 640, 100_000}
	ms := []int64{2500, 10_000, 31_000}
	ss := []int64{5000, 10_000, 40_000}
	for run := range 400 {
		p := endowment.Params{Shape: shapes[rng.IntN(3)], TicksPerYear: tpys[rng.IntN(len(tpys))],
			Capacity: 10_000 + rng.Int64N(1_000_000_000_000), ScaleBP: ms[rng.IntN(3)]}
		if p.Shape == endowment.Constant {
			p.Rate = 1 + rng.Int64N(1_000_000)
		} else {
			p.Rate = 1 + rng.Int64N(100_000)
		}
		season := ss[rng.IntN(3)]
		d := denom(p.TicksPerYear)
		stock := 1 + rng.Int64N(p.Capacity/2)
		carry := rng.Int64N(d.Int64())
		total := big.NewInt(carry)
		var units int64
		capped := false
		for range 300 {
			total.Add(total, bigNumerator(p, stock, season))
			got, ok := endowment.Regen(p, endowment.Step{Stock: stock, Carry: carry, SeasonBP: season})
			if !ok {
				t.Fatalf("run %d: Regen(%+v) not ok", run, p)
			}
			if got.Stock >= p.Capacity {
				capped = true
				break
			}
			units += got.Added
			carry = got.Carry
			// A draw between ticks that keeps at least one unit, as the model does.
			stock = got.Stock - min(got.Stock-1, rng.Int64N(got.Added+3))
		}
		if capped {
			continue
		}
		q, r := new(big.Int).DivMod(total, d, new(big.Int))
		if q.Int64() != units || r.Int64() != carry {
			t.Errorf("run %d %+v: added %d carry %d, oracle %v rem %v", run, p, units, carry, q, r)
		}
	}
}

// TestRegen_ClampBanksNothing verifies a stock that reaches the capacity is
// clamped to it with a carry of 0 and only the units kept reported, that a
// stock already full banks no carry, and that a capacity of 0 gives nothing.
//
// Requirement: FR-RES-011
func TestRegen_ClampBanksNothing(t *testing.T) {
	t.Parallel()
	base := endowment.Params{Shape: endowment.Refill, Rate: 100_000, Capacity: 100, ScaleBP: 10_000, TicksPerYear: 1}
	tests := []struct {
		name string
		p    endowment.Params
		in   endowment.Step
		want endowment.Result
	}{
		// rate 10x a year with one tick a year: n = 10^5 * 40 * 10^8 = 4e14... D = 10^12, so 400
		// units are due and only the 60 of room are kept.
		{"overshoot is clamped", base, endowment.Step{Stock: 40, Carry: 5, SeasonBP: 10_000},
			endowment.Result{Stock: 100, Added: 60}},
		{"full stock banks no carry", base, endowment.Step{Stock: 100, Carry: 999, SeasonBP: 10_000},
			endowment.Result{Stock: 100}},
		{"capacity 0 constant", endowment.Params{Shape: endowment.Constant, Rate: 5, TicksPerYear: 1}, endowment.Step{SeasonBP: 10_000}, endowment.Result{}},
		// Capacity 0 returns before the numerator is computed (4.4.2): a rate, scale and season
		// whose product leaves 128 bits do not make it an error.
		{"capacity 0 skips the numerator", endowment.Params{Shape: endowment.Constant, Rate: math.MaxInt64, ScaleBP: math.MaxInt64, TicksPerYear: 1}, endowment.Step{SeasonBP: math.MaxInt64}, endowment.Result{}},
		// Capacity 1 is not 0: a rate of 10x a year with one tick a year fills the one unit.
		{"capacity 1 fills", endowment.Params{Shape: endowment.Refill, Rate: 100_000, Capacity: 1, ScaleBP: 10_000, TicksPerYear: 1}, endowment.Step{SeasonBP: 10_000}, endowment.Result{Stock: 1, Added: 1}},
		{"capacity 0 logistic", endowment.Params{Shape: endowment.Logistic, Rate: 5, Refuge: 9, TicksPerYear: 1}, endowment.Step{SeasonBP: 10_000}, endowment.Result{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := endowment.Regen(tt.p, tt.in)
			if !ok || got != tt.want {
				t.Errorf("Regen(%+v, %+v) = %+v, %v, want %+v, true", tt.p, tt.in, got, ok, tt.want)
			}
		})
	}
}

// TestRegen_RefusesUnknownShape verifies a Shape outside the closed set is
// refused with ok false, not regenerated as a constant flow.
//
// Guard: the numerator once fell through to the constant rule for any other
// value, so a corrupt Shape produced a plausible stock instead of a refusal.
func TestRegen_RefusesUnknownShape(t *testing.T) {
	t.Parallel()
	for _, shape := range []endowment.Shape{-1, 3, 7} {
		// A capacity of 0 returns early, and must not let a corrupt Shape through.
		for _, capacity := range []int64{100, 0} {
			p := endowment.Params{Shape: shape, Rate: 5, Capacity: capacity, ScaleBP: 10_000, TicksPerYear: 1}
			if got, ok := endowment.Regen(p, endowment.Step{SeasonBP: 10_000}); ok {
				t.Errorf("Regen(%+v) = %+v, true, want not ok", p, got)
			}
		}
	}
}

// TestRegen_FiniteExtinctRefuge verifies a constant stock of rate 0 never
// changes, that a logistic stock of 0 with no refuge is extinct for ever, that
// a refuge restarts growth, and that a refuge at or above the capacity is
// capped below it.
//
// Requirement: FR-RES-031
func TestRegen_FiniteExtinctRefuge(t *testing.T) {
	t.Parallel()
	run := func(p endowment.Params, stock int64, ticks int) int64 {
		var carry int64
		for range ticks {
			got, ok := endowment.Regen(p, endowment.Step{Stock: stock, Carry: carry, SeasonBP: 10_000})
			if !ok {
				t.Fatalf("Regen(%+v) not ok", p)
			}
			stock, carry = got.Stock, got.Carry
		}
		return stock
	}
	finite := endowment.Params{Shape: endowment.Constant, Capacity: 1000, ScaleBP: 10_000, TicksPerYear: 12}
	if got := run(finite, 400, 500); got != 400 {
		t.Errorf("finite stock after 500 ticks = %d, want 400", got)
	}
	extinct := endowment.Params{Shape: endowment.Logistic, Rate: 30_000, Capacity: 1000, ScaleBP: 10_000, TicksPerYear: 12}
	if got := run(extinct, 0, 500); got != 0 {
		t.Errorf("extinct stock without a refuge after 500 ticks = %d, want 0", got)
	}
	refuge := extinct
	refuge.Refuge = 20
	if got := run(refuge, 0, 500); got == 0 {
		t.Errorf("stock with a refuge of 20 after 500 ticks = 0, want growth")
	}
	// Refuge 100 at C = 10 is capped to C - 1 = 9: w = ceil(9 * 1 / 10) = 1, so the stock
	// of 0 grows by 1 when D allows (TPY 1, rate 10 000: n = w * 10^12 = D).
	capped := endowment.Params{Shape: endowment.Logistic, Rate: 10_000, Refuge: 100, Capacity: 10, ScaleBP: 10_000, TicksPerYear: 1}
	if got, ok := endowment.Regen(capped, endowment.Step{SeasonBP: 10_000}); !ok || got.Stock != 1 {
		t.Errorf("Regen(%+v, stock 0) = %+v, %v, want stock 1", capped, got, ok)
	}
}

// TestRegen_ArableYearsOfWorkedOut verifies the arable stock of example A1
// (spec-resources.md 16.1) through four years: refill 100% a year, regenerating
// only in autumn and winter at twice the rate, 19 plots drawn a tick for the
// first 20 ticks of spring. The spring-start stocks are 1 700, 1 748, 1 766 and
// 1 773 (models/res/worked.out), and year 0 passes through 1 320, 1 320, 1 587
// and 1 748 at the end of each season.
//
// Requirement: FR-RES-007, FR-RES-012
func TestRegen_ArableYearsOfWorkedOut(t *testing.T) {
	t.Parallel()
	p := endowment.Params{Shape: endowment.Refill, Rate: 10_000, Capacity: 2000, ScaleBP: 10_000, TicksPerYear: 365}
	seasonBP := func(day int) int64 {
		if day >= 184 { // autumn from day 184, winter from 275
			return 20_000
		}
		return 0
	}
	stock, carry := int64(1700), int64(0)
	wantSpring := []int64{1700, 1748, 1766, 1773}
	wantEnds := map[int]int64{91: 1320, 183: 1320, 274: 1587, 364: 1748}
	for year, want := range wantSpring {
		if stock != want {
			t.Errorf("spring start of year %d = %d, want %d", year, stock, want)
		}
		for day := range 365 {
			if day < 20 {
				stock -= min(19, stock)
			}
			got, ok := endowment.Regen(p, endowment.Step{Stock: stock, Carry: carry, SeasonBP: seasonBP(day)})
			if !ok {
				t.Fatalf("year %d day %d: Regen not ok", year, day)
			}
			stock, carry = got.Stock, got.Carry
			if w, hit := wantEnds[day]; hit && year == 0 && stock != w {
				t.Errorf("year 0 day %d stock = %d, want %d", day, stock, w)
			}
		}
	}
}

// TestRegen_MaxBoundsDoNotOverflow verifies the largest values of
// spec-resources.md 4.6 (numerators near 10^31, D = 10^17) are exact in 128 bits
// and clamp before narrowing, against the math/big oracle.
//
// Requirement: FR-RES-008
func TestRegen_MaxBoundsDoNotOverflow(t *testing.T) {
	t.Parallel()
	const maxC = 1_000_000_000_000_000
	tests := []struct {
		name string
		p    endowment.Params
		in   endowment.Step
	}{
		// n = 10^5 * 10^15 * 10^6 * 10^5 = 10^31; D = 10^17: 10^14 units, below the room 10^15.
		{"refill at the bounds", endowment.Params{Shape: endowment.Refill, Rate: 100_000, Capacity: maxC, ScaleBP: 1_000_000, TicksPerYear: 100_000},
			endowment.Step{SeasonBP: 100_000}},
		// n = 10^15 * 10^6 * 10^5 * 10^4 = 10^30: 10^13 units.
		{"constant at the bounds", endowment.Params{Shape: endowment.Constant, Rate: maxC, Capacity: maxC, ScaleBP: 1_000_000, TicksPerYear: 100_000},
			endowment.Step{SeasonBP: 100_000}},
		// w = ceil(C/2 * C/2 / C) = C/4 = 2.5e14.
		{"logistic at the bounds", endowment.Params{Shape: endowment.Logistic, Rate: 100_000, Capacity: maxC, ScaleBP: 1_000_000, TicksPerYear: 100_000},
			endowment.Step{Stock: maxC / 2, SeasonBP: 100_000}},
		// TPY 1: D = 10^12, so the quotient 10^19 exceeds int64 and is clamped to the room.
		// n = 949 640 990 * 10^15 * 4 257 629 * 455 760 = 1.8e33, q = n / 10^12 has a high word of
		// 99 895 and a low word of 679 713 050 919 680, below the room 10^15: only the high word says
		// the stock is full.
		{"quotient beyond 64 bits with a small low word is clamped", endowment.Params{Shape: endowment.Refill, Rate: 949_640_990, Capacity: maxC, ScaleBP: 4_257_629, TicksPerYear: 1},
			endowment.Step{SeasonBP: 455_760}},
		{"quotient beyond int64 is clamped", endowment.Params{Shape: endowment.Refill, Rate: 100_000, Capacity: maxC, ScaleBP: 1_000_000, TicksPerYear: 1},
			endowment.Step{SeasonBP: 100_000}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := endowment.Regen(tt.p, tt.in)
			if !ok {
				t.Fatalf("Regen(%+v, %+v) not ok", tt.p, tt.in)
			}
			q, r := new(big.Int).DivMod(bigNumerator(tt.p, tt.in.Stock, tt.in.SeasonBP), denom(tt.p.TicksPerYear), new(big.Int))
			room := big.NewInt(tt.p.Capacity - tt.in.Stock)
			if q.Cmp(room) >= 0 {
				if got.Stock != tt.p.Capacity || got.Carry != 0 {
					t.Errorf("Regen(%+v) = %+v, want the stock clamped to %d with no carry", tt.p, got, tt.p.Capacity)
				}
				return
			}
			if big.NewInt(got.Added).Cmp(q) != 0 || big.NewInt(got.Carry).Cmp(r) != 0 {
				t.Errorf("Regen(%+v) = %+v, oracle %v rem %v", tt.p, got, q, r)
			}
		})
	}
}

// TestRegen_SeasonProfileComplete verifies the season factor scales the
// numerator: over a 365-tick year a factor of 20 000 adds twice what 10 000
// does, and a factor of 0 adds nothing and leaves the carry untouched, so a
// season missing from a profile (factor 0) pauses regeneration.
//
// Requirement: FR-RES-012
func TestRegen_SeasonProfileComplete(t *testing.T) {
	t.Parallel()
	p := endowment.Params{Shape: endowment.Constant, Rate: 365, Capacity: 100_000, ScaleBP: 10_000, TicksPerYear: 365}
	year := func(season int64) int64 {
		stock, carry := int64(10), int64(0)
		for range 365 {
			got, _ := endowment.Regen(p, endowment.Step{Stock: stock, Carry: carry, SeasonBP: season})
			stock, carry = got.Stock, got.Carry
		}
		return stock - 10
	}
	// 365 units a year at s = 10 000: one a tick, 365 in the year.
	for season, want := range map[int64]int64{0: 0, 10_000: 365, 20_000: 730} {
		if got := year(season); got != want {
			t.Errorf("units added in a year at season factor %d = %d, want %d", season, got, want)
		}
	}
	held, ok := endowment.Regen(p, endowment.Step{Stock: 10, Carry: 12345, SeasonBP: 0})
	if !ok || held != (endowment.Result{Stock: 10, Carry: 12345}) {
		t.Errorf("Regen at season factor 0 = %+v, %v, want the stock and carry unchanged", held, ok)
	}
}

// TestRegen_RejectsOutOfRange verifies a state outside its range, or a D that
// leaves int64, is reported with ok false and never computed on.
//
// Requirement: NFR-RES-003
func TestRegen_RejectsOutOfRange(t *testing.T) {
	t.Parallel()
	ok0 := endowment.Params{Shape: endowment.Refill, Rate: 100, Capacity: 100, ScaleBP: 10_000, TicksPerYear: 12}
	tests := []struct {
		name string
		p    endowment.Params
		in   endowment.Step
	}{
		{"negative stock", ok0, endowment.Step{Stock: -1, SeasonBP: 10_000}},
		{"stock above the capacity", ok0, endowment.Step{Stock: 101, SeasonBP: 10_000}},
		{"negative carry", ok0, endowment.Step{Carry: -1, SeasonBP: 10_000}},
		{"carry above D", ok0, endowment.Step{Carry: 12_000_000_000_001, SeasonBP: 10_000}},
		// 2^128 - 1 = (2^32 - 1)(2^32 + 1) * 274 177 * 67 280 421 310 721, so this numerator is the
		// largest 128-bit value and adding any carry overflows.
		{"numerator plus carry beyond 128 bits", endowment.Params{Shape: endowment.Refill, Rate: 67_280_421_310_721, Capacity: 4_294_967_295, ScaleBP: 4_294_967_297, TicksPerYear: 1}, endowment.Step{Carry: 1, SeasonBP: 274_177}},
		{"carry at D", ok0, endowment.Step{Carry: 12_000_000_000_000, SeasonBP: 10_000}},
		{"negative season factor", ok0, endowment.Step{SeasonBP: -1}},
		{"zero ticks a year", endowment.Params{Capacity: 1}, endowment.Step{}},
		{"D beyond int64", endowment.Params{Capacity: 1, TicksPerYear: math.MaxInt64}, endowment.Step{}},
		{"negative rate", endowment.Params{Rate: -1, Capacity: 1, TicksPerYear: 1}, endowment.Step{}},
		{"numerator beyond 128 bits", endowment.Params{Shape: endowment.Refill, Rate: math.MaxInt64, Capacity: math.MaxInt64, ScaleBP: math.MaxInt64, TicksPerYear: 1}, endowment.Step{SeasonBP: math.MaxInt64}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got, ok := endowment.Regen(tt.p, tt.in); ok {
				t.Errorf("Regen(%+v, %+v) = %+v, true, want not ok", tt.p, tt.in, got)
			}
		})
	}
}
