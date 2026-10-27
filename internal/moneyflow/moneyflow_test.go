package moneyflow_test

import (
	"math"
	"math/big"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/moneyflow"
)

// maxMoney is a stand-in for the root's MaxMoney bound: the functions take it
// as a parameter and never read the root (architecture.md 2.9).
const maxMoney = int64(math.MaxInt64 / 4)

// spreadRun runs a rate grant for n ticks from carry c0 and returns the ticks
// elapsed (1-based) when the first non-zero due arrived and the total paid.
func spreadRun(t *testing.T, w, amount, scale, c0, period int64, n int) (first int, total int64) {
	t.Helper()
	c := c0
	for i := 1; i <= n; i++ {
		due, nc, ok := moneyflow.Spread(w, amount, scale, c, period, maxMoney)
		if !ok {
			t.Fatalf("Spread(%d, %d, %d, %d, %d) not ok at tick %d", w, amount, scale, c, period, i)
		}
		if due > 0 && first == 0 {
			first = i
		}
		total += due
		c = nc
	}
	return first, total
}

// TestSpread_YearOf7 verifies the worked example of spec-money-supply.md 17.2:
// 7 a year over 365 ticks, account 0 first paid in the 53rd tick (7*53 = 371 >=
// 365), account 100 in the 38th (100 + 7*38 = 366), each paid 7 in the year,
// and weights 1, 3 and 40 paid 7, 21 and 280.
//
// Requirement: FR-MON-012, FR-MON-013
func TestSpread_YearOf7(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		id, w     int64
		wantFirst int
		wantTotal int64
	}{
		{"account 0, weight 1", 0, 1, 53, 7},
		{"account 100, weight 1", 100, 1, 38, 7},
		{"account 0, weight 3", 0, 3, 18, 21},   // 3*7*18 = 378 >= 365
		{"account 0, weight 40", 0, 40, 2, 280}, // 40*7*2 = 560 >= 365; 280 a tick-year over 365 ticks
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c0 := moneyflow.StartCarry(tt.id, 365)
			first, total := spreadRun(t, tt.w, 7, 10_000, c0, 365, 365)
			if first != tt.wantFirst || total != tt.wantTotal {
				t.Errorf("Spread over a year (id %d, w %d) first = %d, total = %d, want %d, %d",
					tt.id, tt.w, first, total, tt.wantFirst, tt.wantTotal)
			}
		})
	}
}

// TestSpread_TotalIsExact verifies that the total paid over n ticks is
// floor((n*w*amount*scale + c0) / (period*10^4)): no drift and no lump, for a
// rate that is not a whole number of units a tick.
//
// Requirement: FR-MON-012
func TestSpread_TotalIsExact(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                 string
		w, amount, scale, id int64
		period               int64
		ticks                int
	}{
		{"year of 365", 2, 11, 10_000, 17, 365, 1000},
		{"scaled up", 1, 3, 25_000, 4, 52, 333},
		{"scaled down", 5, 9, 1_500, 51, 52, 411},
		{"per tick", 3, 8, 10_000, 9, 1, 50},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c0 := moneyflow.StartCarry(tt.id, tt.period)
			_, total := spreadRun(t, tt.w, tt.amount, tt.scale, c0, tt.period, tt.ticks)
			num := new(big.Int).Mul(big.NewInt(int64(tt.ticks)*tt.w*tt.amount), big.NewInt(tt.scale))
			num.Add(num, big.NewInt(c0))
			want := num.Div(num, big.NewInt(tt.period*10_000)).Int64()
			if total != want {
				t.Errorf("total of %d ticks = %d, want floor((n*w*amount*scale + c0)/(period*1e4)) = %d", tt.ticks, total, want)
			}
		})
	}
}

// TestSpread_ScaleChangeMovesOnlyTheTerm verifies that a change of the scale
// between ticks changes only the term added that tick: the carry stays in
// units that do not depend on it (spec-money-supply.md 6.1).
//
// Requirement: FR-MON-012
func TestSpread_ScaleChangeMovesOnlyTheTerm(t *testing.T) {
	t.Parallel()
	// period 10, amount 3: den = 1e5. Tick 1 at scale 10 000: num 30 000, due 0,
	// carry 30 000. Tick 2 at scale 20 000: num 60 000 + 30 000 = 90 000, due 0,
	// carry 90 000. Tick 3 at scale 10 000: 30 000 + 90 000 = 120 000, due 1,
	// carry 20 000.
	c := int64(0)
	want := []struct{ scale, due, carry int64 }{{10_000, 0, 30_000}, {20_000, 0, 90_000}, {10_000, 1, 20_000}}
	for i, w := range want {
		due, nc, ok := moneyflow.Spread(1, 3, w.scale, c, 10, maxMoney)
		if !ok || due != w.due || nc != w.carry {
			t.Fatalf("tick %d Spread(scale %d, carry %d) = %d, %d, %v, want %d, %d", i+1, w.scale, c, due, nc, ok, w.due, w.carry)
		}
		c = nc
	}
}

// TestStartCarry_ByID verifies the initial carry (ID mod P) * 10^4 and that it
// is a function of the account ID alone.
//
// Requirement: FR-MON-013
func TestStartCarry_ByID(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		id, period int64
		want       int64
	}{
		{"id 0", 0, 365, 0},
		{"id 100", 100, 365, 1_000_000},
		{"id wraps", 366, 365, 10_000},
		{"period 1", 77, 1, 0},
		{"period below 1", 5, 0, 0},
		{"negative id", -1, 365, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := moneyflow.StartCarry(tt.id, tt.period); got != tt.want {
				t.Errorf("StartCarry(%d, %d) = %d, want %d", tt.id, tt.period, got, tt.want)
			}
		})
	}
}

// TestRateLevy_Trace171 verifies the four ticks of spec-money-supply.md 17.1:
// a weight-2 household, a rate levy of 5 000 basis points a year over 365
// ticks, exempt 500 a household: dues 0, 2, 2, 4 and carries 3.0e10, 1.7e10,
// 3.39e10, 0.77e10 (denominator 3.65e10).
//
// Requirement: FR-MON-014
func TestRateLevy_Trace171(t *testing.T) {
	t.Parallel()
	bases := []int64{600, 1200, 1798, 2396}
	wantDue := []int64{0, 2, 2, 4}
	wantCarry := []int64{30_000_000_000, 17_000_000_000, 33_900_000_000, 7_700_000_000}
	c := int64(0)
	for i, base := range bases {
		due, nc, ok := moneyflow.RateLevy(base, 5_000, 10_000, c, 365, maxMoney)
		if !ok || due != wantDue[i] || nc != wantCarry[i] {
			t.Fatalf("tick %d RateLevy(%d, carry %d) = %d, %d, %v, want %d, %d", i, base, c, due, nc, ok, wantDue[i], wantCarry[i])
		}
		c = nc
	}
}

// TestRateLevy_NeverTakesPastBase verifies that a rate levy cut by the
// available balance never applies more than the base, including a MaxMoney
// base at the largest rate and scale, where the numerator reaches 10^27
// (plan D327).
//
// Requirement: FR-MON-015
func TestRateLevy_NeverTakesPastBase(t *testing.T) {
	t.Parallel()
	const maxBP, maxScale = 10_000, 100_000
	tests := []struct {
		name string
		base int64
	}{
		{"small base at the largest rate", 7},
		{"zero base", 0},
		{"large base", 1_000_000_000_000},
		{"MaxMoney base", maxMoney},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			due, carry, ok := moneyflow.RateLevy(tt.base, maxBP, maxScale, 0, 1, maxMoney)
			if !ok {
				t.Fatalf("RateLevy(%d, %d, %d) not ok", tt.base, maxBP, maxScale)
			}
			if carry < 0 || carry >= 100_000_000 {
				t.Errorf("RateLevy(%d) carry = %d, want in [0, 1e8)", tt.base, carry)
			}
			// At 10 000 bp a tick and scale 10x the due is ten bases (saturated
			// at maxMoney); the room of the account is the base.
			if tt.base > 0 && due < tt.base {
				t.Errorf("RateLevy(%d) due = %d, want at least the base (rate 10x)", tt.base, due)
			}
			if applied, cut := moneyflow.Clamp(due, tt.base, maxMoney); applied > tt.base || (due > tt.base && cut != moneyflow.CutRoom) {
				t.Errorf("Clamp(%d, base %d) = %d, %v, want at most the base, and CutRoom when the due exceeds it", due, tt.base, applied, cut)
			}
		})
	}
}

// TestTopUp_Trace174 verifies spec-money-supply.md 17.4: a level of 5 000 and a
// balance of 3 800 owes 1 200; at 6 300 it owes 0, and the sweep (a rate levy
// of 10 000 basis points a tick, exempt 5 000) then takes the 1 300 above.
//
// Requirement: FR-MON-016
func TestTopUp_Trace174(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct{ balance, want int64 }{{3_800, 1_200}, {6_300, 0}, {5_000, 0}, {0, 5_000}} {
		got, ok := moneyflow.TopUp(1, 5_000, 10_000, tt.balance, maxMoney)
		if !ok || got != tt.want {
			t.Errorf("TopUp(1, 5000, 10000, %d) = %d, %v, want %d", tt.balance, got, ok, tt.want)
		}
	}
	// Weight and scale: w*level*S/10^4 = 3*5000*5000/10^4 = 7 500, less 500.
	if got, ok := moneyflow.TopUp(3, 5_000, 5_000, 500, maxMoney); !ok || got != 7_000 {
		t.Errorf("TopUp(3, 5000, 5000, 500) = %d, %v, want 7000", got, ok)
	}
	due, _, ok := moneyflow.RateLevy(6_300-5_000, 10_000, 10_000, 0, 1, maxMoney)
	if !ok || due != 1_300 {
		t.Errorf("sweep RateLevy(1300, 10000 bp) = %d, %v, want 1300", due, ok)
	}
}

// TestSpread_SaturatesAtMaxMoney verifies that a due past maxMoney is returned
// as maxMoney while the carry stays the exact remainder (spec-money-supply.md
// 6.4.1), and that an input outside the bounds is not ok.
//
// Requirement: FR-MON-017
func TestSpread_SaturatesAtMaxMoney(t *testing.T) {
	t.Parallel()
	// w*amount = 1e18, scale 1e5, period 1: due 1e19 past maxMoney (2.3e18).
	due, carry, ok := moneyflow.Spread(1_000_000, 1_000_000_000_000, 100_000, 0, 1, maxMoney)
	if !ok || due != maxMoney || carry != 0 {
		t.Errorf("Spread(1e6, 1e12, 1e5) = %d, %d, %v, want %d, 0, true", due, carry, ok, maxMoney)
	}
	// An odd remainder survives the saturation: 1e18*1e5 + 7 = q*1e4 + 7.
	_, carry, ok = moneyflow.Spread(1_000_000, 1_000_000_000_000, 100_000, 7, 1, maxMoney)
	if !ok || carry != 7 {
		t.Errorf("Spread saturated carry = %d, %v, want 7, true", carry, ok)
	}
	if got, ok := moneyflow.TopUp(1, math.MaxInt64, 100_000, 0, maxMoney); !ok || got != maxMoney {
		t.Errorf("TopUp(1, MaxInt64, 1e5) = %d, %v, want %d, true", got, ok, maxMoney)
	}
	if _, _, ok := moneyflow.Spread(-1, 1, 1, 0, 1, maxMoney); ok {
		t.Error("Spread with a negative weight is ok, want not ok")
	}
	if _, _, ok := moneyflow.Spread(1, 1, 1, 0, 0, maxMoney); ok {
		t.Error("Spread with period 0 is ok, want not ok")
	}
	if _, _, ok := moneyflow.Spread(math.MaxInt64, 2, 1, 0, 1, maxMoney); ok {
		t.Error("Spread with w*amount past int64 is ok, want not ok")
	}
	if _, _, ok := moneyflow.RateLevy(1, 1, 1, 0, math.MaxInt64, maxMoney); ok {
		t.Error("RateLevy with period*10^8 past int64 is ok, want not ok")
	}
	if _, ok := moneyflow.TopUp(math.MaxInt64, 2, 1, 0, maxMoney); ok {
		t.Error("TopUp with w*level past int64 is ok, want not ok")
	}
	if _, ok := moneyflow.TopUp(1, 1, 1, -1, maxMoney); ok {
		t.Error("TopUp with a negative balance is ok, want not ok")
	}
}

// TestClamp_FirstBoundWins verifies that the room bound acts first and the
// limit second, that a due is classed by the first bound that cut it only, and
// that a negative room or remaining is 0 (plan D328).
//
// Requirement: FR-MON-020, FR-MON-021, FR-MON-022
func TestClamp_FirstBoundWins(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                 string
		due, room, remaining int64
		want                 int64
		wantCut              moneyflow.Cut
	}{
		{"nothing cut", 10, 50, 50, 10, moneyflow.CutNone},
		{"room cuts", 10, 4, 50, 4, moneyflow.CutRoom},
		{"limit cuts", 10, 50, 6, 6, moneyflow.CutLimit},
		{"both bind, room first and counted once", 10, 8, 3, 3, moneyflow.CutRoom},
		{"both bind, limit tighter than room", 10, 5, 2, 2, moneyflow.CutRoom},
		{"due equals room", 10, 10, 50, 10, moneyflow.CutNone},
		{"due equals remaining", 10, 50, 10, 10, moneyflow.CutNone},
		{"no room", 10, 0, 50, 0, moneyflow.CutRoom},
		{"negative room is zero", 10, -5, 50, 0, moneyflow.CutRoom},
		{"limit spent", 10, 50, 0, 0, moneyflow.CutLimit},
		{"negative remaining is zero", 10, 50, -1, 0, moneyflow.CutLimit},
		{"zero due", 0, 0, 0, 0, moneyflow.CutNone},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, cut := moneyflow.Clamp(tt.due, tt.room, tt.remaining)
			if got != tt.want || cut != tt.wantCut {
				t.Errorf("Clamp(%d, %d, %d) = %d, %v, want %d, %v", tt.due, tt.room, tt.remaining, got, cut, tt.want, tt.wantCut)
			}
		})
	}
}

// FuzzSpread checks, for any bounded operands, that the carry stays in
// [0, den) and that sum of dues * den + final carry = sum of numerators, over a
// run of ticks with the weight and scale unchanged.
//
// Requirement: NFR-MON-003
func FuzzSpread(f *testing.F) {
	f.Add(int64(1), int64(7), int64(10_000), int64(0), int64(365), uint8(40))
	f.Add(int64(40), int64(1_000_000), int64(100_000), int64(1_000_000), int64(52), uint8(7))
	f.Add(int64(3), int64(0), int64(10_000), int64(5), int64(1), uint8(3))
	f.Fuzz(func(t *testing.T, w, amount, scale, id, period int64, ticks uint8) {
		if w < 0 || w > 1_000_000 || amount < 0 || amount > 1_000_000_000_000 || scale < 0 || scale > 100_000 || period < 1 || period > 100_000 {
			t.Skip("outside the bounds the validator enforces")
		}
		den := new(big.Int).Mul(big.NewInt(period), big.NewInt(10_000))
		c := moneyflow.StartCarry(id, period)
		sumDue, sumNum := new(big.Int), new(big.Int)
		term := new(big.Int).Mul(big.NewInt(w*amount), big.NewInt(scale))
		c0 := c
		for i := 0; i < int(ticks); i++ {
			due, nc, ok := moneyflow.Spread(w, amount, scale, c, period, math.MaxInt64)
			if !ok {
				t.Fatalf("Spread(%d, %d, %d, %d, %d) not ok", w, amount, scale, c, period)
			}
			if nc < 0 || big.NewInt(nc).Cmp(den) >= 0 {
				t.Fatalf("carry %d outside [0, %v)", nc, den)
			}
			sumDue.Add(sumDue, big.NewInt(due))
			sumNum.Add(sumNum, term)
			c = nc
		}
		lhs := new(big.Int).Mul(sumDue, den)
		lhs.Add(lhs, big.NewInt(c))
		sumNum.Add(sumNum, big.NewInt(c0))
		if lhs.Cmp(sumNum) != 0 {
			t.Fatalf("sum(due)*den + carry = %v, want sum(num) + c0 = %v", lhs, sumNum)
		}
	})
}

// TestInputs_OutsideTheBoundsAreNotOK verifies, for Spread, RateLevy, TopUp and
// StartCarry, that each argument below its bound (a negative amount, balance,
// weight, carry or scale, a period below 1, a negative maxMoney, a negative
// account ID) gives ok false and zero results, that the lowest valid value of each
// is accepted, and that a product that leaves int64 is refused the same way: ok
// is false only for an input outside the bounds the caller validated.
//
// Requirement: FR-MON-017, FR-MON-012, FR-MON-015, FR-MON-016
//
// Guard: the root passes validated values, so no run reaches these branches; a
// mutant that dropped one check would change nothing but the contract.
func TestInputs_OutsideTheBoundsAreNotOK(t *testing.T) {
	t.Parallel()
	const big = math.MaxInt64
	// Spread(w, amount, scale, carry, period, maxMoney) and RateLevy(base, rateBP,
	// scale, carry, period, maxMoney) take their six arguments in the same order of
	// bounds: four non-negative, a period of at least 1, a non-negative maxMoney.
	six := func(name string, overflow [][6]int64, f func(a [6]int64) (int64, int64, bool)) {
		valid := [6]int64{1, 1, 1, 0, 1, 10}
		if _, _, ok := f([6]int64{0, 0, 0, 0, 1, 0}); !ok {
			t.Errorf("%s with every argument at its lowest valid value: not ok, want ok", name)
		}
		for i := range valid {
			bad := valid
			bad[i] = -1
			if i == 4 {
				bad[i] = 0 // a period of 0
			}
			if due, c, ok := f(bad); ok || due != 0 || c != 0 {
				t.Errorf("%s(%v) = %d, %d, %v, want 0, 0, false", name, bad, due, c, ok)
			}
		}
		for _, bad := range overflow {
			if due, c, ok := f(bad); ok || due != 0 || c != 0 {
				t.Errorf("%s(%v) = %d, %d, %v, want 0, 0, false: the product leaves int64", name, bad, due, c, ok)
			}
		}
	}
	six("Spread", [][6]int64{{big, 2, 1, 0, 1, 10}, {1, 1, 1, 0, big, 10}}, func(a [6]int64) (int64, int64, bool) {
		return moneyflow.Spread(a[0], a[1], a[2], a[3], a[4], a[5])
	})
	six("RateLevy", [][6]int64{{1, big, 2, 0, 1, 10}, {1, 1, 1, 0, 100_000_000_000, 10}}, func(a [6]int64) (int64, int64, bool) {
		return moneyflow.RateLevy(a[0], a[1], a[2], a[3], a[4], a[5])
	})
	// TopUp(w, level, scale, balance, maxMoney).
	if _, ok := moneyflow.TopUp(0, 0, 0, 0, 0); !ok {
		t.Error("TopUp with every argument at its lowest valid value: not ok, want ok")
	}
	valid := [5]int64{1, 1, 1, 0, 10}
	for i := range valid {
		bad := valid
		bad[i] = -1
		if due, ok := moneyflow.TopUp(bad[0], bad[1], bad[2], bad[3], bad[4]); ok || due != 0 {
			t.Errorf("TopUp(%v) = %d, %v, want 0, false", bad, due, ok)
		}
	}
	// At or above the level nothing is due: 5 000 * 10 000 / 10 000 = 5 000.
	for _, balance := range []int64{5_000, 6_300} {
		if due, ok := moneyflow.TopUp(1, 5_000, 10_000, balance, maxMoney); !ok || due != 0 {
			t.Errorf("TopUp(level 5000, balance %d) = %d, %v, want 0, true", balance, due, ok)
		}
	}
	if due, ok := moneyflow.TopUp(big, 2, 1, 0, 10); ok || due != 0 {
		t.Errorf("TopUp(w*level overflows) = %d, %v, want 0, false", due, ok)
	}
	// StartCarry: a negative ID or a period below 1 starts at 0, not at a value the
	// formula would give for the number.
	for _, tt := range []struct{ id, period, want int64 }{{-1, 5, 0}, {7, 0, 0}, {7, -3, 0}, {7, 5, 20_000}} {
		if got := moneyflow.StartCarry(tt.id, tt.period); got != tt.want {
			t.Errorf("StartCarry(%d, %d) = %d, want %d", tt.id, tt.period, got, tt.want)
		}
	}
}
