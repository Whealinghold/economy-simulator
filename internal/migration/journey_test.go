package migration_test

import (
	"math"
	"math/big"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/migration"
	"github.com/Whealinghold/economy-sim/internal/ordering"
)

// TestTicks_CeilingAndFloorOfOne verifies the journey time k = max(1, ceil(D /
// speed)): a ceiling, never below one tick, and the refusal of a negative
// distance or a speed below 1.
//
// Requirement: FR-MIG-016
func TestTicks_CeilingAndFloorOfOne(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name            string
		distance, speed int64
		want            int64
		wantOK          bool
	}{
		// 70 / 20 = 3.5 rounds up to 4; 40 / 20 is exact.
		{"upper to lower on foot", 70, 20, 4, true},
		{"upper to middle on foot", 40, 20, 2, true},
		{"one speed's worth is one tick", 3_000, 3_000, 1, true},
		{"zero distance is one tick", 0, 20, 1, true},
		{"a short hop is one tick", 1, 3_000, 1, true},
		{"one past an exact multiple rounds up", 41, 20, 3, true},
		{"negative distance", -1, 20, 0, false},
		{"zero speed", 40, 0, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := migration.Ticks(tt.distance, tt.speed)
			if got != tt.want || ok != tt.wantOK {
				t.Errorf("Ticks(%d, %d) = %d, %v, want %d, %v (0 when refused)", tt.distance, tt.speed, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

// TestDiscount_ShareOfHorizonLeft verifies V_eff = Vhat * (H - k) / H truncated
// toward zero, that a journey that takes the whole horizon or more is refused
// (stage 2), and that a negative estimate truncates toward zero too.
//
// Requirement: FR-MIG-016
func TestDiscount_ShareOfHorizonLeft(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		vhat, k, horiz int64
		want           int64
		wantOK         bool
	}{
		// MIG 15.1: 14 376 * 178 / 180 = 14 216.5, truncated to 14 216.
		{"middle", 14_376, 2, 180, 14_216, true},
		// 17 431 * 176 / 180 = 17 043.2.
		{"lower", 17_431, 4, 180, 17_043, true},
		{"k at the horizon is refused", 14_376, 180, 180, 0, false},
		{"k past the horizon is refused", 14_376, 181, 180, 0, false},
		{"k one below the horizon keeps one share", 18_000, 179, 180, 100, true},
		// -1 000 * 9 / 10 = -900 exactly; -7 * 9 / 10 = -6.3 truncates to -6, not -7.
		{"negative estimate truncates toward zero", -7, 1, 10, -6, true},
		{"zero horizon", 100, 1, 0, 0, false},
		{"negative horizon", 100, 0, -5, 0, false},
		{"negative k", 100, -1, 10, 0, false},
		// A horizon of 1 with no journey keeps the whole estimate; a zero-tick journey
		// keeps it at any horizon.
		{"horizon of one", 100, 0, 1, 100, true},
		{"zero-tick journey", 100, 0, 10, 100, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := migration.Discount(tt.vhat, tt.k, tt.horiz)
			if got != tt.want || ok != tt.wantOK {
				t.Errorf("Discount(%d, %d, %d) = %d, %v, want %d, %v", tt.vhat, tt.k, tt.horiz, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

// TestDiscount_ExtremesDoNotOverflow verifies that the largest estimate with the
// smallest discount is exact: the product is held in 128 bits.
//
// Requirement: NFR-MIG-003
func TestDiscount_ExtremesDoNotOverflow(t *testing.T) {
	t.Parallel()
	got, ok := migration.Discount(math.MaxInt64, 1, 2)
	if want := int64(math.MaxInt64 / 2); !ok || got != want {
		t.Errorf("Discount(MaxInt64, 1, 2) = %d, %v, want %d, true", got, ok, want)
	}
	got, ok = migration.Discount(math.MinInt64, 1, 2)
	if want := int64(math.MinInt64 / 2); !ok || got != want {
		t.Errorf("Discount(MinInt64, 1, 2) = %d, %v, want %d, true", got, ok, want)
	}
}

// TestDue_OncePerPeriod verifies that over any window of period ticks every
// agent is due exactly once (the rotation of spec-world.md 9.1.1), that the due
// tick is the same in every period, and that a period below 1 has no turn.
//
// Requirement: FR-MIG-005
func TestDue_OncePerPeriod(t *testing.T) {
	t.Parallel()
	for _, period := range []int64{1, 2, 30, 60, 1_000} {
		for id := int64(0); id < 200; id++ {
			for start := int64(0); start < 2; start++ {
				n, first := 0, int64(-1)
				for tick := start * period; tick < (start+1)*period; tick++ {
					if migration.Due(tick, id, period) {
						n++
						first = tick - start*period
					}
				}
				if n != 1 {
					t.Fatalf("period %d, agent %d, window %d: due %d times, want 1", period, id, start, n)
				}
				if !migration.Due(first+(start+3)*period, id, period) {
					t.Fatalf("period %d, agent %d: due tick %d is not repeated in a later period", period, id, first)
				}
			}
		}
	}
	// The turn is at the tick where tick + bucket is a multiple of the period, not
	// tick - bucket: agent 1042 at period 30 has bucket 18 (Weyl2 2 622 213 950 * 30
	// >> 32), so it is due at 12, 42 and 72 and not at 18, 48 or 78.
	if b := ordering.Bucket(1042, 30); b != 18 {
		t.Fatalf("Bucket(1042, 30) = %d, want 18", b)
	}
	for tick := int64(0); tick < 90; tick++ {
		if got, want := migration.Due(tick, 1042, 30), tick%30 == 12; got != want {
			t.Errorf("Due(%d, 1042, 30) = %v, want %v", tick, got, want)
		}
	}
	if migration.Due(5, 7, 0) || migration.Due(5, 7, -3) {
		t.Errorf("Due with a period below 1 = true, want false")
	}
}

// TestDue_SpreadOverTheBuckets verifies that the reviews of a period are spread:
// with 3 000 agents and a period of 30 no tick has more than twice the mean.
//
// Guard: a rotation that put every agent on one tick would make a region decide
// on the same lagged news at once, which is what the rotation is for.
func TestDue_SpreadOverTheBuckets(t *testing.T) {
	t.Parallel()
	const agents, period = 3_000, 30
	var per [period]int
	for tick := int64(0); tick < period; tick++ {
		for id := int64(0); id < agents; id++ {
			if migration.Due(tick, id, period) {
				per[tick]++
			}
		}
	}
	for tick, n := range per {
		if n > 2*agents/period || n == 0 {
			t.Errorf("tick %d reviews %d agents, want between 1 and %d", tick, n, 2*agents/period)
		}
	}
}

// TestSettled_LongerOfCooldownAndWindow verifies t - RoleSince >= max(cooldown,
// window), at the boundary, with a negative RoleSince and with a distance past
// int64.
//
// Requirement: FR-MIG-006
func TestSettled_LongerOfCooldownAndWindow(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                          string
		tick, since, cooldown, window int64
		want                          bool
	}{
		// max(60, 30) = 60: 60 ticks after the creation is the first settled tick.
		{"cooldown binds, one short", 59, 0, 60, 30, false},
		{"cooldown binds, at the boundary", 60, 0, 60, 30, true},
		// max(10, 90) = 90: the window binds.
		{"window binds, one short", 89, 0, 10, 90, false},
		{"window binds, at the boundary", 90, 0, 10, 90, true},
		{"after an arrival at 100", 150, 100, 60, 30, false},
		{"well past the boundary", 500, 0, 60, 30, true},
		{"a positive role since past int64 below", math.MinInt64, 1, 60, 30, false},
		{"negative role since is settled at once", 0, -60, 60, 30, true},
		{"a difference past int64 is a distant past", math.MaxInt64, math.MinInt64, 60, 30, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := migration.Settled(tt.tick, tt.since, tt.cooldown, tt.window); got != tt.want {
				t.Errorf("Settled(%d, %d, %d, %d) = %v, want %v", tt.tick, tt.since, tt.cooldown, tt.window, got, tt.want)
			}
		})
	}
}

// TestPushed_BelowGateAndAbsentRatio verifies the push gate: strictly below
// MoveBelow, a negative ratio is below any non-negative gate, and an absent ratio
// (no upkeep) is never pushed.
//
// Requirement: FR-MIG-008
func TestPushed_BelowGateAndAbsentRatio(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		earned  int64
		present bool
		below   int64
		want    bool
	}{
		{"below", 5_100, true, 10_000, true},
		{"at the gate is not pushed", 10_000, true, 10_000, false},
		{"one below the gate", 9_999, true, 10_000, true},
		{"above the gate", 15_000, true, 10_000, false},
		{"negative ratio", -3_000, true, 0, true},
		{"gate 0 pushes only a negative ratio", 0, true, 0, false},
		{"absent ratio", -50, false, 10_000, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := migration.Pushed(tt.earned, tt.present, tt.below); got != tt.want {
				t.Errorf("Pushed(%d, %v, %d) = %v, want %v", tt.earned, tt.present, tt.below, got, tt.want)
			}
		})
	}
}

// TestMargin_OffsetSpread verifies g = gain + floor(Offset(id) * width / 2^32)
// with Offset the Weyl value: agent 1042 of MIG 15.1 has Offset 4 258 099 970 and
// a margin of 5 957 at gain 1 000 and width 5 000, every margin lies in [gain,
// gain + width), and the margin overflows are reported.
//
// Requirement: FR-MIG-017
func TestMargin_OffsetSpread(t *testing.T) {
	t.Parallel()
	// 4 258 099 970 * 5 000 / 4 294 967 296 = 4 957.06, floor 4 957; plus 1 000.
	if g, ok := migration.Margin(1042, 1_000, 5_000); !ok || g != 5_957 {
		t.Errorf("Margin(1042, 1000, 5000) = %d, %v, want 5957, true", g, ok)
	}
	if g, ok := migration.Margin(1042, 1_000, 0); !ok || g != 1_000 {
		t.Errorf("Margin(1042, 1000, 0) = %d, %v, want 1000, true", g, ok)
	}
	lo, hi := int64(math.MaxInt64), int64(0)
	for id := int64(0); id < 5_000; id++ {
		g, ok := migration.Margin(id, 1_000, 5_000)
		if !ok || g < 1_000 || g >= 6_000 {
			t.Fatalf("Margin(%d, 1000, 5000) = %d, %v, want in [1000, 6000)", id, g, ok)
		}
		lo, hi = min(lo, g), max(hi, g)
	}
	if lo > 1_050 || hi < 5_950 {
		t.Errorf("margins of 5 000 agents span [%d, %d], want most of [1000, 6000)", lo, hi)
	}
	if g, ok := migration.Margin(1042, math.MaxInt64, 5_000); ok || g != 0 {
		t.Errorf("Margin(1042, MaxInt64, 5000) = %d, %v, want 0, false (overflow)", g, ok)
	}
	if g, ok := migration.Margin(1042, 1_000, -1); ok || g != 0 {
		t.Errorf("Margin(1042, 1000, -1) = %d, %v, want 0, false (negative width)", g, ok)
	}
	// The largest width gives floor(4 258 099 970 * (2^63 - 1) / 2^32), still an int64.
	want := new(big.Int).Rsh(new(big.Int).Mul(big.NewInt(4_258_099_970), big.NewInt(math.MaxInt64)), 32)
	if g, ok := migration.Margin(1042, 0, math.MaxInt64); !ok || g != want.Int64() {
		t.Errorf("Margin(1042, 0, MaxInt64) = %d, %v, want %v, true", g, ok, want)
	}
}

// TestClears_CrossMultiplication verifies the gain test V_eff * 10 000 >= V_a *
// (10 000 + g) against the exact product on a table of edge values, positive,
// zero and negative home figures included, by the ceiling form that needs no
// signed 128-bit product.
//
// Requirement: FR-MIG-017
func TestClears_CrossMultiplication(t *testing.T) {
	t.Parallel()
	// MIG 15.1: V_a 6 393, g 5 957: 6 393 * 15 957 / 10 000 = 10 201.3.
	for _, tt := range []struct {
		veff int64
		want bool
	}{{10_201, false}, {10_202, true}, {14_216, true}} {
		if got, ok := migration.Clears(tt.veff, 6_393, 5_957); !ok || got != tt.want {
			t.Errorf("Clears(%d, 6393, 5957) = %v, %v, want %v, true", tt.veff, got, ok, tt.want)
		}
	}
	vals := []int64{math.MinInt64 / 20_000, -20_000, -10_001, -10_000, -9_999, -1, 0, 1, 9_999, 10_000, 10_001, 20_000, 123_456_789}
	margins := []int64{0, 1, 999, 1_000, 5_957, 10_000, 100_000}
	for _, veff := range vals {
		for _, home := range vals {
			for _, g := range margins {
				got, ok := migration.Clears(veff, home, g)
				lhs := new(big.Int).Mul(big.NewInt(veff), big.NewInt(10_000))
				rhs := new(big.Int).Mul(big.NewInt(home), big.NewInt(10_000+g))
				if want := lhs.Cmp(rhs) >= 0; !ok || got != want {
					t.Errorf("Clears(%d, %d, %d) = %v, %v, want %v, true", veff, home, g, got, ok, want)
				}
			}
		}
	}
	for _, c := range []struct {
		name               string
		veff, home, margin int64
	}{
		{"negative margin", 1, 1, -1},
		{"10 000 plus the margin overflows", 1, 1, math.MaxInt64},
		{"the product overflows", math.MaxInt64, math.MaxInt64, 1_000},
	} {
		if got, ok := migration.Clears(c.veff, c.home, c.margin); ok || got {
			t.Errorf("Clears with %s = %v, %v, want false, false", c.name, got, ok)
		}
	}
}
