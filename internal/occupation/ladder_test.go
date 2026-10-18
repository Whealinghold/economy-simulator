package occupation_test

import (
	"math"
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/occupation"
)

// TestPastCooldown_Cases verifies that an agent acts only after the tick it
// took up its role and after the cooldown, and that a difference beyond int64
// counts as long enough.
//
// Tick 10: a role taken at 10 is not before this tick whatever the cooldown; at
// 9 it is 1 tick old; with a cooldown of 5, 10 - 5 = 5 passes and 10 - 6 = 4
// does not.
//
// Requirement: FR-SPN-050
func TestPastCooldown_Cases(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                      string
		tick, roleSince, cooldown int64
		want                      bool
	}{
		{"created this tick", 10, 10, 0, false},
		{"created next tick", 10, 11, 0, false},
		{"one tick old, no cooldown", 10, 9, 0, true},
		{"exactly the cooldown", 10, 5, 5, true},
		{"one short of the cooldown", 10, 6, 5, false},
		{"difference beyond int64", math.MaxInt64, math.MinInt64, math.MaxInt64, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := occupation.PastCooldown(tt.tick, tt.roleSince, tt.cooldown); got != tt.want {
				t.Errorf("PastCooldown(%d, %d, %d) = %v, want %v", tt.tick, tt.roleSince, tt.cooldown, got, tt.want)
			}
		})
	}
}

// TestRung3_Cases verifies the trigger of rung 3: privation at the threshold and
// the threshold's number of ticks in the role, each alone.
//
// Threshold 20, tick 100: privation 20 with a role taken at 80 (20 ticks)
// passes; 19 privation fails; 19 ticks in the role fails. A threshold of 0 still
// needs roleSince < tick.
//
// Requirement: FR-SPN-013, FR-SPN-050
func TestRung3_Cases(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                                string
		privation, tick, roleSince, removeP int64
		want                                bool
	}{
		{"both at the threshold", 20, 100, 80, 20, true},
		{"privation one short", 19, 100, 80, 20, false},
		{"ticks one short", 20, 100, 81, 20, false},
		{"threshold 0 in the creation tick", 5, 10, 10, 0, false},
		{"threshold 0 the tick after", 5, 10, 9, 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := occupation.Rung3(tt.privation, tt.tick, tt.roleSince, tt.removeP); got != tt.want {
				t.Errorf("Rung3(%d, %d, %d, %d) = %v, want %v", tt.privation, tt.tick, tt.roleSince, tt.removeP, got, tt.want)
			}
		})
	}
}

// TestReviewDue_Rotation verifies the review rotation against the worked case
// and a nonzero bucket.
//
// Agent 17, period 10: Weyl2(17) = 178 801 879 and (178 801 879 * 10) >> 32 = 0,
// so it reviews on ticks 0, 10, 20 and no other of 0 to 20. Agent 1:
// Weyl2(1) = 1 779 033 703, times 10 is 17 790 337 030, >> 32 = 4 (4 * 2^32 =
// 17 179 869 184), so it reviews where (t + 4) mod 10 = 0: ticks 6 and 16. Period
// 1 is every tick; a period below 1 never.
//
// Requirement: FR-SPN-018
func TestReviewDue_Rotation(t *testing.T) {
	t.Parallel()
	due := func(id, period int64) []int64 {
		var ts []int64
		for tick := int64(0); tick <= 20; tick++ {
			if occupation.ReviewDue(id, tick, period) {
				ts = append(ts, tick)
			}
		}
		return ts
	}
	tests := []struct {
		name       string
		id, period int64
		want       []int64
	}{
		{"agent 17, bucket 0", 17, 10, []int64{0, 10, 20}},
		{"agent 1, bucket 4", 1, 10, []int64{6, 16}},
		{"period 1", 3, 1, slices.Collect(func(yield func(int64) bool) {
			for i := int64(0); i <= 20; i++ {
				if !yield(i) {
					return
				}
			}
		})},
		{"period 0", 17, 0, nil},
		{"negative period", 17, -3, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := due(tt.id, tt.period); !slices.Equal(got, tt.want) {
				t.Errorf("ReviewDue(%d, tick, %d) over ticks 0..20 = %v, want %v", tt.id, tt.period, got, tt.want)
			}
		})
	}
}

// TestAllowance_Cases verifies the ceiling of a share of households against the
// worked cases and an overflow.
//
// 1 000 * 50 / 10 000 = 5; 50 * 50 / 10 000 = 0.25, up to 1; 400 * 50 / 10 000 =
// 2; 1 000 * 100 / 10 000 = 10; a share of 0 gives 0. MaxInt64 * MaxInt64 /
// 10 000 is beyond int64.
//
// Requirement: FR-SPN-016
func TestAllowance_Cases(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name              string
		households, share int64
		want              int64
		wantOK            bool
	}{
		{"region of 1 000 at entry share 50", 1000, 50, 5, true},
		{"role of 50 rounds up", 50, 50, 1, true},
		{"role of 400", 400, 50, 2, true},
		{"exit share 100", 1000, 100, 10, true},
		{"share 0", 1000, 0, 0, true},
		{"no households", 0, 100, 0, true},
		{"overflow", math.MaxInt64, math.MaxInt64, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := occupation.Allowance(tt.households, tt.share)
			if got != tt.want || ok != tt.wantOK {
				t.Errorf("Allowance(%d, %d) = %d, %v, want %d, %v", tt.households, tt.share, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

// TestExits_Cases verifies the reductions and removals of rung 3: the worked
// cohort, a weight-1 agent removed whole, the floor of one household, the order
// by privation then reference, the allowance that cuts the agent crossing it,
// the agents that find it used up, an agent with nothing unfed, the cause test
// and the cap by weight.
//
// Weight 40, 7 unfed, step 1 000: ceil(40 * 1 000 / 10 000) = 4 and min(7, 4) =
// 4 households leave. Money 12 000 against upkeep 300 * 40 = 12 000 is not below
// it, so starved; 11 999 is insolvent. Weight 1, step 1 000: ceil(0.1) = 1, the
// whole agent. Weight 5 with step 0 still sheds max(1, 0) = 1.
//
// Weight 2 with 9 unfed (more than it holds) and step 50 000: min(9, ceil(2 * 5)
// = 10) = 9, cut to the weight 2, a removal.
//
// Allowance 5 over A (privation 9, wants 4), B (8, wants 4) and C (7, wants 1):
// A sheds 4 (1 left), B sheds 1 and 3 are cut, C finds 0 left and 1 is cut, so 4
// are cut in all. A weight of 1e6 with a step of MaxInt64 overflows the step and
// sheds all the unfed (7).
//
// Requirement: FR-SPN-014, FR-SPN-015, FR-SPN-059
func TestExits_Cases(t *testing.T) {
	t.Parallel()
	cohort := occupation.Leaver{Ref: 0, Privation: 10, Weight: 40, Unfed: 7, Money: 12000, Upkeep: 300}
	with := func(l occupation.Leaver, f func(*occupation.Leaver)) occupation.Leaver {
		f(&l)
		return l
	}
	tests := []struct {
		name      string
		ls        []occupation.Leaver
		allowance int64
		step      int64
		want      []occupation.Exit
		wantCut   int64
	}{
		{"worked cohort", []occupation.Leaver{cohort}, 100, 1000,
			[]occupation.Exit{{Ref: 0, Households: 4}}, 0},
		{"cohort insolvent below upkeep times weight", []occupation.Leaver{with(cohort, func(l *occupation.Leaver) { l.Money = 11999 })}, 100, 1000,
			[]occupation.Exit{{Ref: 0, Households: 4, Insolvent: true}}, 0},
		{"weight one removed whole", []occupation.Leaver{{Ref: 3, Privation: 9, Weight: 1, Unfed: 1, Money: 0, Upkeep: 0}}, 100, 1000,
			[]occupation.Exit{{Ref: 3, Households: 1, Remove: true}}, 0},
		{"at least one household", []occupation.Leaver{{Ref: 0, Weight: 5, Unfed: 5}}, 100, 0,
			[]occupation.Exit{{Ref: 0, Households: 1}}, 0},
		{"capped by the unfed", []occupation.Leaver{{Ref: 0, Weight: 40, Unfed: 2}}, 100, 10000,
			[]occupation.Exit{{Ref: 0, Households: 2}}, 0},
		{"capped by the weight", []occupation.Leaver{{Ref: 0, Weight: 2, Unfed: 9}}, 100, 50000,
			[]occupation.Exit{{Ref: 0, Households: 2, Remove: true}}, 0},
		{"step beyond int64 sheds all unfed", []occupation.Leaver{{Ref: 0, Weight: 1_000_000, Unfed: 7}}, 100, math.MaxInt64,
			[]occupation.Exit{{Ref: 0, Households: 7}}, 0},
		{"upkeep times weight beyond int64 is insolvent", []occupation.Leaver{{Ref: 0, Weight: 2, Unfed: 1, Money: 1e18, Upkeep: math.MaxInt64}}, 100, 1000,
			[]occupation.Exit{{Ref: 0, Households: 1, Insolvent: true}}, 0},
		{"nothing unfed", []occupation.Leaver{{Ref: 0, Privation: 50, Weight: 4, Unfed: 0}}, 100, 1000, nil, 0},
		{"no weight", []occupation.Leaver{{Ref: 0, Privation: 50, Weight: 0, Unfed: 1}}, 100, 1000, nil, 0},
		{"privation descending then reference", []occupation.Leaver{
			{Ref: 5, Privation: 8, Weight: 1, Unfed: 1},
			{Ref: 9, Privation: 10, Weight: 1, Unfed: 1},
			{Ref: 2, Privation: 8, Weight: 1, Unfed: 1},
		}, 100, 1000, []occupation.Exit{
			{Ref: 9, Households: 1, Remove: true},
			{Ref: 2, Households: 1, Remove: true},
			{Ref: 5, Households: 1, Remove: true},
		}, 0},
		{"allowance cuts the agent that crosses it", []occupation.Leaver{
			{Ref: 1, Privation: 9, Weight: 40, Unfed: 7},
			{Ref: 2, Privation: 8, Weight: 40, Unfed: 7},
			{Ref: 3, Privation: 7, Weight: 1, Unfed: 1},
		}, 5, 1000, []occupation.Exit{
			{Ref: 1, Households: 4},
			{Ref: 2, Households: 1},
		}, 4},
		{"allowance 0 cuts everyone", []occupation.Leaver{{Ref: 0, Weight: 40, Unfed: 7}}, 0, 1000, nil, 4},
		{"negative allowance is 0", []occupation.Leaver{{Ref: 0, Weight: 40, Unfed: 7}}, -3, 1000, nil, 4},
		{"no leavers", nil, 10, 1000, nil, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, cut := occupation.Exits(nil, slices.Clone(tt.ls), tt.allowance, tt.step)
			if !slices.Equal(got, tt.want) || cut != tt.wantCut {
				t.Errorf("Exits(%+v, allowance %d, step %d) = %+v, cut %d, want %+v, cut %d",
					tt.ls, tt.allowance, tt.step, got, cut, tt.want, tt.wantCut)
			}
		})
	}
}

// TestExits_ReusesDst verifies that Exits appends to the start of the slice it
// is given, so a caller's scratch is reused and nothing from a previous call
// survives.
//
// Guard: the executor reuses one scratch slice per region and tick. If Exits
// appended after the old length, last tick's exits would be executed again.
func TestExits_ReusesDst(t *testing.T) {
	t.Parallel()
	dst := make([]occupation.Exit, 3, 8)
	for i := range dst {
		dst[i] = occupation.Exit{Ref: 99, Households: 99}
	}
	got, _ := occupation.Exits(dst, []occupation.Leaver{{Ref: 1, Weight: 1, Unfed: 1}}, 5, 1000)
	want := []occupation.Exit{{Ref: 1, Households: 1, Remove: true}}
	if !slices.Equal(got, want) {
		t.Errorf("Exits(dst of length 3) = %+v, want %+v", got, want)
	}
	if &got[0] != &dst[0] {
		t.Errorf("Exits allocated a new slice although dst had capacity")
	}
}

// TestRemaining_Cases verifies the scaling of a reduction against the worked
// case, a floor, the ends and the refused arguments.
//
// 12 000 * 36 / 40 = 10 800 (1 200 go to the estate); 80 * 36 / 40 = 72;
// 100 * 2 / 3 = 66.7, floored to 66; weight 0 keeps 0; the same weight keeps all.
// MaxInt64 * 2 / 3 = 18 446 744 073 709 551 614 / 3 = 6 148 914 691 236 517 204.67,
// floored to ...204.
//
// FR-SPN-005 is covered in part: the floor arithmetic only; the ledger conserving
// the difference is the executor's (row 11).
//
// Requirement: FR-SPN-005, FR-SPN-051
func TestRemaining_Cases(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		x, w, wNew int64
		want       int64
		wantOK     bool
	}{
		{"money of the worked cohort", 12000, 40, 36, 10800, true},
		{"units of the worked cohort", 80, 40, 36, 72, true},
		{"floors", 100, 3, 2, 66, true},
		{"to nothing", 100, 3, 0, 0, true},
		{"same weight", 100, 3, 3, 100, true},
		{"nothing held", 0, 40, 36, 0, true},
		{"large amount stays exact", math.MaxInt64, 3, 2, 6148914691236517204, true},
		{"no weight", 100, 0, 0, 0, false},
		{"weight rises", 100, 3, 4, 0, false},
		{"negative new weight", 100, 3, -1, 0, false},
		{"negative amount", -1, 3, 2, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := occupation.Remaining(tt.x, tt.w, tt.wNew)
			if got != tt.want || ok != tt.wantOK {
				t.Errorf("Remaining(%d, %d, %d) = %d, %v, want %d, %v", tt.x, tt.w, tt.wNew, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

// TestRemaining_ShareNeverFalls verifies, over a grid, that the fed households
// keep at least the share per household they held: floor(x * w' / w) / w' is
// not below floor(x / w), and the estate gets the rest.
//
// For x = w * q + r (r < w), x * w' / w >= w' * q, so the integer share is at
// least q.
//
// Requirement: FR-SPN-051
func TestRemaining_ShareNeverFalls(t *testing.T) {
	t.Parallel()
	for _, x := range []int64{0, 1, 7, 99, 12000, 123456789} {
		for w := int64(1); w <= 12; w++ {
			for wNew := int64(1); wNew <= w; wNew++ {
				kept, ok := occupation.Remaining(x, w, wNew)
				if !ok || kept < 0 || kept > x {
					t.Fatalf("Remaining(%d, %d, %d) = %d, %v, want a value in [0, %d]", x, w, wNew, kept, ok, x)
				}
				if got, want := kept/wNew, x/w; got < want {
					t.Errorf("Remaining(%d, %d, %d) = %d: share %d per household, want at least %d", x, w, wNew, kept, got, want)
				}
			}
		}
	}
}

// TestReviewDue_NegativeTickMatchesResidue verifies that a negative tick reviews
// exactly when its positive equivalent mod period does (-1 is 9 for period 10).
//
// Guard: Go's % is negative for a negative tick, and a reader may fear that the
// sum with the bucket then misses the zero residue. It cannot, since a sum is
// divisible by the period whatever its sign; without this test a "fix" could
// shift the rotation of the construction tick -1.
func TestReviewDue_NegativeTickMatchesResidue(t *testing.T) {
	t.Parallel()
	const period = 10
	for id := int64(0); id < 50; id++ {
		for tick := int64(-25); tick < 0; tick++ {
			want := occupation.ReviewDue(id, tick+3*period, period)
			if got := occupation.ReviewDue(id, tick, period); got != want {
				t.Errorf("ReviewDue(%d, %d, %d) = %v, want %v", id, tick, period, got, want)
			}
		}
	}
}
