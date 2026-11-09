package outside_test

import (
	"math/rand/v2"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/outside"
)

// line is a MidIn at tick 0 with the defaults of the module: no season, no
// shock, no walk, the factor held between 1 000 and 100 000 basis points, 360
// ticks a year and a price bound of 10^9.
func line(base int64) outside.MidIn {
	return outside.MidIn{
		Base: base, Season: 10_000, Shock: 10_000, FloorBP: 1_000, CeilBP: 100_000,
		MaxPrice: 1_000_000_000, TicksPerYear: 360,
	}
}

// TestMid_SpecVectors verifies the mids and the quotes of the worked examples
// of spec-external-sector.md 18.1 and 18.3, with the arithmetic of each.
//
// Requirement: FR-EXT-024, FR-EXT-026, FR-EXT-010
func TestMid_SpecVectors(t *testing.T) {
	t.Parallel()
	reagent := line(300)
	reagent.TrendBP, reagent.Tick = -200, 3
	tests := []struct {
		name      string
		in        outside.MidIn
		wantMid   int64
		marginBP  int64
		sell      bool
		wantQuote int64
	}{
		// 18.1: salt, Base 40, a fixed price; ask = ceil(40 * 10 500 / 10 000) = 42.
		{"salt ask", line(40), 40, 500, true, 42},
		// 18.1: grain, Base 20; bid = floor(20 * 9 000 / 10 000) = 18.
		{"grain bid", line(20), 20, 1_000, false, 18},
		// 18.3: ore, Base 120; bid = floor(120 * 8 000 / 10 000) = 96.
		{"ore bid", line(120), 120, 2_000, false, 96},
		// 18.3: reagent, Base 300, TrendBP -200, tick 3 of 360: the trend is
		// trunc(-200 * 3 / 360) = trunc(-1.67) = -1, so F = 9 999 and the mid is
		// round(300 * 9 999 / 10 000) = round(299.97) = 300; ask = ceil(300 * 10 800 / 10 000) = 324.
		{"reagent ask", reagent, 300, 800, true, 324},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mid, clamped, ok := outside.Mid(tt.in)
			if !ok || mid != tt.wantMid || clamped {
				t.Errorf("Mid(%+v) = %d, %v, %v, want %d, false, true", tt.in, mid, clamped, ok, tt.wantMid)
			}
			got, ok := outside.Quote(mid, tt.marginBP, 10_000, tt.in.MaxPrice, tt.sell)
			if !ok || got != tt.wantQuote {
				t.Errorf("Quote(%d, %d, 10000, sell %v) = %d, %v, want %d, true", mid, tt.marginBP, tt.sell, got, ok, tt.wantQuote)
			}
		})
	}
}

// TestMid_Factors verifies the season, the trend, the walk drift and the shock
// each as a factor of the mid, and the half-to-even rounding of the single
// division.
//
// Requirement: FR-EXT-024, FR-EXT-025
func TestMid_Factors(t *testing.T) {
	t.Parallel()
	with := func(f func(*outside.MidIn)) outside.MidIn {
		m := line(1_000)
		f(&m)
		return m
	}
	tests := []struct {
		name string
		in   outside.MidIn
		want int64
	}{
		// 1 000 * 12 500 / 10 000 = 1 250.
		{"season 125 percent", with(func(m *outside.MidIn) { m.Season = 12_500 }), 1_250},
		// TrendBP 3 650 a year, tick 73 of 365: trunc(3 650 * 73 / 365) = 730, F = 10 730, mid 1 073.
		{"trend", with(func(m *outside.MidIn) { m.TrendBP, m.Tick, m.TicksPerYear = 3_650, 73, 365 }), 1_073},
		// The trend truncates toward zero: trunc(-100 * 1 / 360) = 0, not -1.
		{"negative trend truncates toward zero", with(func(m *outside.MidIn) { m.TrendBP, m.Tick = -100, 1 }), 1_000},
		// Drift -500: F_walk = 9 500, F = round(10 000 * 9 500 / 10 000) = 9 500, mid 950.
		{"walk drift", with(func(m *outside.MidIn) { m.Drift = -500 }), 950},
		// Shock 15 000: 1 000 * 15 000 / 10 000 = 1 500.
		{"shock", with(func(m *outside.MidIn) { m.Shock = 15_000 }), 1_500},
		// 90 * 12 500 / 10 000 = 112.5, a tie, goes to the even 112; 110 gives 137.5, to 138.
		{"half to even down", func() outside.MidIn { m := line(90); m.Season = 12_500; return m }(), 112},
		{"half to even up", func() outside.MidIn { m := line(110); m.Season = 12_500; return m }(), 138},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if mid, clamped, ok := outside.Mid(tt.in); !ok || mid != tt.want || clamped {
				t.Errorf("Mid(%+v) = %d, %v, %v, want %d, false, true", tt.in, mid, clamped, ok, tt.want)
			}
		})
	}
}

// TestMid_SingleRounding verifies that the mid is rounded once, at the end:
// rounding the season first would give another price.
//
// Guard: spec-external-sector.md 6.1 asks for a single rounding. Base 10 at a
// season of 4 500 and a shock of 20 000 is 10 * 4 500 * 20 000 / 10^8 = 9
// exactly, but a mid rounded after the season (4.5, to the even 4) and then
// doubled would be 8. A change to two roundings moves every price with a season.
func TestMid_SingleRounding(t *testing.T) {
	t.Parallel()
	m := line(10)
	m.Season, m.Shock = 4_500, 20_000
	if mid, _, ok := outside.Mid(m); !ok || mid != 9 {
		t.Errorf("Mid(%+v) = %d, %v, want 9, true", m, mid, ok)
	}
}

// TestMid_Clamps verifies that the factor is held between the floor and the
// ceiling, the mid between 1 and the price bound, and that clamped says so,
// while an unclamped mid is not flagged.
//
// Requirement: FR-EXT-026
func TestMid_Clamps(t *testing.T) {
	t.Parallel()
	with := func(base int64, f func(*outside.MidIn)) outside.MidIn {
		m := line(base)
		f(&m)
		return m
	}
	tests := []struct {
		name        string
		in          outside.MidIn
		want        int64
		wantClamped bool
	}{
		// Trend 100 000 a year over two years: F_trend = 10 000 + 200 000, held at the
		// ceiling 100 000, so the mid is ten times the base.
		{"ceiling", with(5, func(m *outside.MidIn) { m.TrendBP, m.Tick = 100_000, 720 }), 50, true},
		// Trend -100 000 over one year: 10 000 - 100 000 is below 0, so F_trend is 0 and F is
		// held at the floor 1 000, a tenth of the base.
		{"floor", with(100, func(m *outside.MidIn) { m.TrendBP, m.Tick = -100_000, 360 }), 10, true},
		// The same floor on a base of 3: 3 * 1 000 / 10 000 = 0.3 rounds to 0, held at 1.
		{"mid at least one", with(3, func(m *outside.MidIn) { m.TrendBP, m.Tick = -100_000, 360 }), 1, true},
		// 900 * 100 000 / 10 000 = 9 000, held at the price bound 5 000.
		{"price bound", with(900, func(m *outside.MidIn) { m.Season, m.MaxPrice = 100_000, 5_000 }), 5_000, true},
		{"no clamp", line(100), 100, false},
		// A season of 0 makes the product 0, held at 1 although the factor was not clamped.
		{"mid at least one from a zero season", with(100, func(m *outside.MidIn) { m.Season = 0 }), 1, true},
		// With a floor of 0 the factor reaches 0 and the mid is held at 1: 10^9 * 1 / 10^4 would be
		// 100 if a negative trend factor were held at 1 instead of 0.
		{"factor of zero", with(1_000_000_000, func(m *outside.MidIn) { m.FloorBP, m.TrendBP, m.Tick = 0, -100_000, 360 }), 1, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mid, clamped, ok := outside.Mid(tt.in)
			if !ok || mid != tt.want || clamped != tt.wantClamped {
				t.Errorf("Mid(%+v) = %d, %v, %v, want %d, %v, true", tt.in, mid, clamped, ok, tt.want, tt.wantClamped)
			}
		})
	}
}

// TestMid_Refuses verifies that an input outside the domain, or a product that
// does not fit in int64, reports ok false and never a price.
//
// Requirement: FR-EXT-040
func TestMid_Refuses(t *testing.T) {
	t.Parallel()
	with := func(f func(*outside.MidIn)) outside.MidIn {
		m := line(10)
		f(&m)
		return m
	}
	tests := []struct {
		name string
		in   outside.MidIn
	}{
		{"base below one", with(func(m *outside.MidIn) { m.Base = 0 })},
		{"price bound below one", with(func(m *outside.MidIn) { m.MaxPrice = 0 })},
		{"negative season", with(func(m *outside.MidIn) { m.Season = -1 })},
		{"negative shock", with(func(m *outside.MidIn) { m.Shock = -1 })},
		{"negative tick", with(func(m *outside.MidIn) { m.Tick = -1 })},
		{"no ticks in a year", with(func(m *outside.MidIn) { m.TicksPerYear = 0 })},
		{"negative floor", with(func(m *outside.MidIn) { m.FloorBP = -1 })},
		{"ceiling below the floor", with(func(m *outside.MidIn) { m.CeilBP = 999 })},
		{"drift of minus one", with(func(m *outside.MidIn) { m.Drift = -10_000 })},
		{"base times season overflows", with(func(m *outside.MidIn) { m.Base, m.Season = 1<<62, 1<<20 })},
		{"factor times shock overflows", with(func(m *outside.MidIn) { m.TrendBP, m.Tick, m.CeilBP, m.Shock = 1<<50, 360, 1<<62, 1<<40 })},
		{"trend overflows", with(func(m *outside.MidIn) { m.TrendBP, m.Tick, m.TicksPerYear = 1<<62, 1<<62, 1 })},
		{"trend plus one overflows", with(func(m *outside.MidIn) { m.TrendBP, m.Tick, m.TicksPerYear = 1<<63-1, 1, 1 })},
		{"drift plus one overflows", with(func(m *outside.MidIn) { m.Drift = 1<<63 - 1 })},
		{"drift below minus one", with(func(m *outside.MidIn) { m.Drift = -10_001 })},
		// Base * Season is 2^62 and factor * shock is 2^62, both fit; 2^124 / 10^12 does not.
		{"quotient overflows", with(func(m *outside.MidIn) {
			m.Base, m.Season, m.TrendBP, m.Tick, m.CeilBP, m.Shock = 1<<42, 1<<20, 1<<42, 360, 1<<62, 1<<20
		})},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if mid, clamped, ok := outside.Mid(tt.in); ok || clamped || mid != 0 {
				t.Errorf("Mid(%+v) = %d, %v, %v, want 0, false, false", tt.in, mid, clamped, ok)
			}
		})
	}
}

// TestWalk_Update verifies one step of the drift: the revert term truncates
// toward zero, the draw is added, and the result is held within the bound.
//
// Requirement: FR-EXT-022
func TestWalk_Update(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                      string
		drift, revertBP, d, bound int64
		want                      int64
	}{
		// No revert: 0 + 100.
		{"free step", 0, 0, 100, 2_000, 100},
		// trunc(99 * 5 000 / 10 000) = trunc(49.5) = 49; 99 - 49 + 0 = 50.
		{"revert truncates toward zero", 99, 5_000, 0, 2_000, 50},
		// trunc(-99 * 5 000 / 10 000) = trunc(-49.5) = -49; -99 + 49 = -50, not -49 - 1.
		{"negative revert truncates toward zero", -99, 5_000, 0, 2_000, -50},
		// 1 000 - trunc(1 000 * 500 / 10 000) = 1 000 - 50 = 950, plus a step of -30.
		{"revert and step", 1_000, 500, -30, 2_000, 920},
		// 1 950 + 100 = 2 050, held at the bound 2 000; the same below.
		{"bound above", 1_950, 0, 100, 2_000, 2_000},
		{"bound below", -1_950, 0, -100, 2_000, -2_000},
		// A full revert undoes the whole drift.
		{"full revert", 700, 10_000, 5, 2_000, 5},
		// A revert past 10 000 is held at 10 000, the whole drift.
		{"revert held at one", 700, 20_000, 0, 2_000, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := outside.Walk(tt.drift, tt.revertBP, tt.d, tt.bound); got != tt.want {
				t.Errorf("Walk(%d, %d, %d, %d) = %d, want %d", tt.drift, tt.revertBP, tt.d, tt.bound, got, tt.want)
			}
		})
	}
}

// TestWalk_StaysInBounds verifies that for sixty-four seeds of draws the drift
// of 2 000 steps never leaves [-bound, bound], with and without a revert, with
// the step at its bound of 5 000 and the drift bound at 9 999.
//
// Requirement: NFR-EXT-004
func TestWalk_StaysInBounds(t *testing.T) {
	t.Parallel()
	for _, p := range []struct{ step, revert, bound int64 }{
		{100, 0, 2_000}, {100, 500, 2_000}, {5_000, 0, 9_999}, {5_000, 10_000, 9_999}, {1, 0, 1},
	} {
		for seed := uint64(1); seed <= 64; seed++ {
			r := rand.New(rand.NewPCG(seed, 7))
			var drift int64
			for tick := range 2_000 {
				d := r.Int64N(2*p.step+1) - p.step
				drift = outside.Walk(drift, p.revert, d, p.bound)
				if drift < -p.bound || drift > p.bound {
					t.Fatalf("seed %d, step %d, revert %d: drift %d at tick %d is outside [-%d, %d]", seed, p.step, p.revert, drift, tick, p.bound, p.bound)
				}
			}
		}
	}
}

// TestQuote_Rounding verifies the quote of a side at a gate: the ask rounds up
// and the bid down in the margin and again in the gate scale, the price bound
// holds, and a bid can reach 0.
//
// Requirement: FR-EXT-010
func TestQuote_Rounding(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                           string
		mid, marginBP, scaleBP, maxPrc int64
		sell                           bool
		want                           int64
	}{
		// ceil(40 * 10 500 / 10 000) = 42.
		{"ask", 40, 500, 10_000, 1_000_000, true, 42},
		// ceil(33 * 10 100 / 10 000) = ceil(33.33) = 34, rounded up.
		{"ask rounds up", 33, 100, 10_000, 1_000_000, true, 34},
		// floor(33 * 9 900 / 10 000) = floor(32.67) = 32, rounded down.
		{"bid rounds down", 33, 100, 10_000, 1_000_000, false, 32},
		// Scale 12 500 on the ask 42: ceil(42 * 12 500 / 10 000) = ceil(52.5) = 53; on the bid 18: floor(22.5) = 22.
		{"ask scaled up", 40, 500, 12_500, 1_000_000, true, 53},
		{"bid scaled up", 20, 1_000, 12_500, 1_000_000, false, 22},
		// Scale 8 000: ceil(42 * 0.8) = ceil(33.6) = 34; floor(18 * 0.8) = floor(14.4) = 14.
		{"ask scaled down", 40, 500, 8_000, 1_000_000, true, 34},
		{"bid scaled down", 20, 1_000, 8_000, 1_000_000, false, 14},
		// ceil(100 * 110 000 / 10 000) = 1 100, held at the price bound 1 000.
		{"ask held at the bound", 100, 100_000, 10_000, 1_000, true, 1_000},
		// floor(1 * 1 / 10 000) = 0: the side places nothing (a cut by the price).
		{"bid reaches zero", 1, 9_999, 10_000, 1_000_000, false, 0},
		// A margin of 10 000 takes the whole mid.
		{"bid of nothing", 50, 10_000, 10_000, 1_000_000, false, 0},
		// A margin of 0 and a scale of 0 are in the domain: ceil(40 * 10 000 / 10 000) = 40, and a scale of 0 gives 0.
		{"ask with no margin", 40, 0, 10_000, 1_000_000, true, 40},
		{"bid with no margin", 40, 0, 10_000, 1_000_000, false, 40},
		{"scale of zero", 40, 500, 0, 1_000_000, true, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := outside.Quote(tt.mid, tt.marginBP, tt.scaleBP, tt.maxPrc, tt.sell)
			if !ok || got != tt.want {
				t.Errorf("Quote(%d, %d, %d, %d, sell %v) = %d, %v, want %d, true", tt.mid, tt.marginBP, tt.scaleBP, tt.maxPrc, tt.sell, got, ok, tt.want)
			}
		})
	}
	for _, bad := range []struct {
		mid, margin, scale, max int64
		sell                    bool
	}{
		{0, 0, 10_000, 10, true}, {5, -1, 10_000, 10, true}, {5, 0, -1, 10, true},
		{5, 0, 10_000, 0, true}, {5, 10_001, 10_000, 10, false}, {1 << 62, 100_000, 100_000, 1 << 62, true},
	} {
		if got, ok := outside.Quote(bad.mid, bad.margin, bad.scale, bad.max, bad.sell); ok || got != 0 {
			t.Errorf("Quote(%+v) = %d, %v, want 0, false", bad, got, ok)
		}
	}
}

// TestQuote_NeverCrosses verifies that when the two margins of a line sum to
// at least one basis point, the bid is below the ask for every mid, every gate
// scale and every pair of margins tried, so a buyer and a seller counterparty of
// one gate can never match each other.
//
// Requirement: FR-EXT-009
//
// The property holds while the scaled quotes stay below the price bound. Past
// it both are held at the bound and can be equal: see the subtest, which pins
// that edge (the module reports it in its report to the owner of EXT).
func TestQuote_NeverCrosses(t *testing.T) {
	t.Parallel()
	const maxPrice = 1_000_000_000
	margins := []struct{ sell, buy int64 }{{0, 1}, {1, 0}, {500, 500}, {1, 9_999}, {100_000, 0}, {0, 9_999}, {250, 1_000}}
	scales := []int64{1, 1_000, 8_000, 9_999, 10_000, 10_001, 12_500, 100_000}
	for mid := int64(1); mid <= 400; mid++ {
		for _, m := range margins {
			for _, s := range scales {
				ask, okA := outside.Quote(mid, m.sell, s, maxPrice, true)
				bid, okB := outside.Quote(mid, m.buy, s, maxPrice, false)
				if !okA || !okB || bid >= ask {
					t.Fatalf("mid %d margins %+v scale %d: bid %d, ask %d (%v, %v), want bid < ask", mid, m, s, bid, ask, okB, okA)
				}
			}
		}
	}
	t.Run("both held at the price bound", func(t *testing.T) {
		t.Parallel()
		// Mid 90, sell margin 0, buy margin 1, scale 100 000, bound 100: the bid is
		// floor(floor(90 * 9 999 / 10 000) * 10) = 890 and the ask 900, both held at 100.
		ask, _ := outside.Quote(90, 0, 100_000, 100, true)
		bid, _ := outside.Quote(90, 1, 100_000, 100, false)
		if ask != 100 || bid != 100 {
			t.Errorf("held quotes = bid %d, ask %d, want both 100 (the bound): the edge moved", bid, ask)
		}
	})
}

// TestWant_Carry verifies the wanted units of a side with the carry of the
// fraction: a capacity of 3 at an availability of 2 500 wants 0, 1, 1, 1, 0, 1,
// 1, 1 units, nothing is lost over any run, and a carry or a quantity outside
// the domain wants nothing and keeps the carry.
//
// Requirement: FR-EXT-028
func TestWant_Carry(t *testing.T) {
	t.Parallel()
	// 3 * 2 500 = 7 500 a tick: carries 7 500, 5 000, 2 500, 0, 7 500, ... and
	// the units are the carries that crossed 10 000: 0, 1, 1, 1, 0, 1, 1, 1.
	var carry int64
	want := []int64{0, 1, 1, 1, 0, 1, 1, 1}
	for i, w := range want {
		var units int64
		units, carry = outside.Want(3, 2_500, carry)
		if units != w {
			t.Errorf("Want(3, 2500) in tick %d = %d, want %d (carry %d)", i, units, w, carry)
		}
	}
	for _, tt := range []struct{ capacity, avail, carry, units, newCarry int64 }{
		{50, 10_000, 0, 50, 0},
		{30, 0, 0, 0, 0},
		{1, 9_999, 9_999, 1, 9_998},
		{0, 10_000, 5, 0, 5},
		{-1, 10_000, 5, 0, 5},
		{3, 10_000, 10_000, 0, 10_000},
		{3, 10_000, 20_000, 0, 20_000},
		{3, -1, 0, 0, 0},
		{1 << 62, 1 << 62, 0, 0, 0},
		// The product fits and the carry does not: the carry is kept, not lost.
		{1<<63 - 1, 1, 5, 0, 5},
	} {
		if u, c := outside.Want(tt.capacity, tt.avail, tt.carry); u != tt.units || c != tt.newCarry {
			t.Errorf("Want(%d, %d, %d) = %d, %d, want %d, %d", tt.capacity, tt.avail, tt.carry, u, c, tt.units, tt.newCarry)
		}
	}
	// Nothing is lost: units * 10 000 + the final carry is the sum of capacity * avail.
	for _, a := range []int64{1, 333, 2_500, 7_777, 10_000} {
		var sum, total, c int64
		for range 1_000 {
			var u int64
			u, c = outside.Want(7, a, c)
			total += u
			sum += 7 * a
		}
		if total*10_000+c != sum {
			t.Errorf("avail %d: 10 000 * %d units + carry %d != %d", a, total, c, sum)
		}
	}
}
