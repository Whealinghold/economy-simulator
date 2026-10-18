package occupation_test

import (
	"math"
	"reflect"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/occupation"
)

// TestBoundaries_Predicates verifies the edges of the three warning predicates
// and of the funded number, Place, Remaining and PastCooldown: the value just
// either side of each comparison.
//
// Guard: a mutation pass (testing.md 9.4) showed these comparisons could be
// flipped (<= for <, a negative argument) with every other test still passing.
// Each case here breaks only its own edge.
//
// Requirement: FR-SPN-008, FR-SPN-042, FR-SPN-047, FR-SPN-050, FR-SPN-051
func TestBoundaries_Predicates(t *testing.T) {
	t.Parallel()
	place := func(hosts []occupation.Host, scale int64) (int, int64, bool) { return occupation.Place(hosts, scale) }
	tests := []struct {
		name string
		got  func() any
		want any
	}{
		{"below floor: weight 0 is not a group", func() any { return occupation.BelowFloor(0, 0, 2) }, false},
		{"below floor: weight 1 is not a group", func() any { return occupation.BelowFloor(1, 0, 2) }, false},
		{"below floor: agents equal to the floor is not below", func() any { return occupation.BelowFloor(5, 2, 2) }, false},
		{"below floor: agents above the floor is not below", func() any { return occupation.BelowFloor(5, 3, 2) }, false},
		{"below floor: weight 2, one agent under a floor of 2", func() any { return occupation.BelowFloor(2, 1, 2) }, true},
		{"unfunded: negative due is not due", func() any { return occupation.Unfunded(-1, 0) }, false},
		{"unfunded: one due and none funded", func() any { return occupation.Unfunded(1, 0) }, true},
		{"unfunded: negative funded is none funded", func() any { return occupation.Unfunded(3, -1) }, true},
		{"unfunded: one funded", func() any { return occupation.Unfunded(3, 1) }, false},
		// min(5, 3/1) = 3: an amount of 1 still limits.
		{"funded: amount 1 limits to the budget", func() any { return occupation.Funded(5, 3, 1) }, int64(3)},
		// A negative budget buys none: without the case, -10/3 = -3 would be returned.
		{"funded: negative budget buys none", func() any { return occupation.Funded(5, -10, 3) }, int64(0)},
		{"funded: zero budget buys none", func() any { return occupation.Funded(5, 0, 3) }, int64(0)},
		// The first host is the lightest, so a later, heavier host must not replace it.
		{"place: a later heavier host does not replace the best", func() any {
			r, room, ok := place([]occupation.Host{{Ref: 7, Weight: 1}, {Ref: 8, Weight: 3}}, 5)
			return []any{r, room, ok}
		}, []any{7, int64(4), true}},
		{"place: a host at the scale is full", func() any {
			_, _, ok := place([]occupation.Host{{Ref: 1, Weight: 4}}, 4)
			return ok
		}, false},
		{"place: a host of weight 0 is below scale 1", func() any {
			r, room, ok := place([]occupation.Host{{Ref: 2, Weight: 0}}, 1)
			return []any{r, room, ok}
		}, []any{2, int64(1), true}},
		{"remaining: a weight of 0 is refused, not divided by", func() any { _, ok := occupation.Remaining(5, 0, 0); return ok }, false},
		{"remaining: a negative new weight is refused", func() any { _, ok := occupation.Remaining(5, 3, -1); return ok }, false},
		{"remaining: a new weight above the weight is refused", func() any { _, ok := occupation.Remaining(5, 3, 4); return ok }, false},
		{"remaining: a negative amount is refused", func() any { _, ok := occupation.Remaining(-1, 3, 2); return ok }, false},
		// A negative cooldown is outside validation; a role taken after the tick must still not be past it.
		{"cooldown: a role taken after the tick is not past a negative cooldown", func() any { return occupation.PastCooldown(5, 7, -5) }, false},
		{"cooldown: a role taken this tick is not past a zero cooldown", func() any { return occupation.PastCooldown(5, 5, 0) }, false},
		{"cooldown: held exactly the cooldown is past", func() any { return occupation.PastCooldown(10, 5, 5) }, true},
		{"cooldown: held one tick short is not past", func() any { return occupation.PastCooldown(9, 5, 5) }, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.got(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("%s = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

// TestBoundaries_Refuse verifies that an entry of exactly 1 household and a
// reduction of exactly 1 household pass the count check.
//
// Guard: the count checks are "below 1"; a flip to "below 2" would refuse the
// smallest legal decision and only a weight-1 cohort would notice.
//
// Requirement: FR-SPN-010
func TestBoundaries_Refuse(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		k    occupation.Kind
		f    occupation.Facts
		want occupation.Refusal
	}{
		{"enter 1 household", occupation.KindEnter, occupation.Facts{RoleKnown: true, Households: 1}, occupation.RefusalNone},
		{"enter 0 households", occupation.KindEnter, occupation.Facts{RoleKnown: true}, occupation.BadHouseholds},
		{"reduce 1 household", occupation.KindReduce, occupation.Facts{AgentKnown: true, CauseSupported: true, Households: 1, Weight: 5}, occupation.RefusalNone},
		{"reduce 0 households", occupation.KindReduce, occupation.Facts{AgentKnown: true, CauseSupported: true, Weight: 5}, occupation.BadHouseholds},
		{"reduce the whole weight", occupation.KindReduce, occupation.Facts{AgentKnown: true, CauseSupported: true, Households: 5, Weight: 5}, occupation.RefusalNone},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := occupation.Refuse(tt.k, tt.f); got != tt.want {
				t.Errorf("Refuse(%v, %+v) = %v, want %v", tt.k, tt.f, got, tt.want)
			}
		})
	}
}

// TestBoundaries_Signals verifies the edges and the saturations of the signals
// and of the capital test: a non-positive upkeep, an overflow that saturates by
// the sign of the wage, and the 128-bit comparison with the high words equal,
// above and below.
//
// 250 ticks over 365: E = 1 000 * 250 * 10 000 / (1 000 * 365) = 6 849 (floor);
// with upkeep MaxInt64/2 the divisor does not fit, so the quotient by upkeep is
// formed first: wage = upkeep gives 250 * 10 000 = 2 500 000, then / 365 = 6 849.
//
// Guard: the saturating paths are unreachable from a validated configuration, so
// only a boundary test notices a flipped comparison in them.
//
// Requirement: FR-SPN-017, FR-SPN-058
func TestBoundaries_Signals(t *testing.T) {
	t.Parallel()
	exp := func(w, work, up, ty int64) any { e, has := occupation.Expected(w, work, up, ty); return []any{e, has} }
	tests := []struct {
		name string
		got  func() any
		want any
	}{
		{"expected: upkeep 0 has no signal", func() any { return exp(1000, 250, 0, 365) }, []any{int64(0), false}},
		{"expected: negative upkeep has no signal", func() any { return exp(1000, 250, -1, 365) }, []any{int64(0), false}},
		{"expected: ticks per year 0 has no signal", func() any { return exp(1000, 250, 1000, 0) }, []any{int64(0), false}},
		{"expected: work ticks overflow saturates up", func() any { return exp(1, math.MaxInt64, 1000, 365) }, []any{int64(math.MaxInt64), true}},
		{"expected: work ticks overflow saturates down", func() any { return exp(-1, math.MaxInt64, 1000, 365) }, []any{int64(math.MinInt64), true}},
		{"expected: work ticks overflow with wage 0 is 0", func() any { return exp(0, math.MaxInt64, 1000, 365) }, []any{int64(0), true}},
		{"expected: the divisor does not fit", func() any { return exp(math.MaxInt64/2, 250, math.MaxInt64/2, 365) }, []any{int64(6849), true}},
		{"expected: the quotient overflows in the long path", func() any { return exp(math.MaxInt64, 250, 1, 365) }, []any{int64(math.MaxInt64), true}},
		{"viability: upkeep 0 has none", func() any { v, has := occupation.Viability(5, 0); return []any{v, has} }, []any{int64(0), false}},
		{"viability: overflow saturates by the wage", func() any { v, _ := occupation.Viability(math.MaxInt64, 1); return v }, int64(math.MaxInt64)},
		{"beats: a ceiling beyond int64 is never reached", func() any { return occupation.Beats(0, math.MaxInt64, 10000) }, false},
		{"beats: a negative ceiling beyond int64 is always reached", func() any { return occupation.Beats(0, math.MinInt64, 10000) }, true},
		{"capital: negative tool price", func() any { return occupation.CapitalOK(10, 0, -1, 1, 1000, 365) }, false},
		{"capital: negative return", func() any { return occupation.CapitalOK(10, 0, 1, -1, 1000, 365) }, false},
		{"capital: destination below own", func() any { return occupation.CapitalOK(0, 10, 1, 1, 1000, 365) }, false},
		{"capital: a zero return passes at equal signals", func() any { return occupation.CapitalOK(5, 5, 100, 0, 1000, 365) }, true},
		{"capital: upkeep 0 is denied even when the right side is 0", func() any { return occupation.CapitalOK(10, 0, 1, 0, 0, 365) }, false},
		{"capital: negative upkeep is denied", func() any { return occupation.CapitalOK(10, 0, 1, 1, -1, 365) }, false},
		{"capital: the product does not fit", func() any { return occupation.CapitalOK(10, 0, 1, 1, math.MaxInt64, 365) }, false},
		// diff 1 * ut 1 = 1 against 2 * 1 = 2: same high word 0, low word below.
		{"capital: equal high words, left low word below", func() any { return occupation.CapitalOK(1, 0, 2, 1, 1, 1) }, false},
		// diff 1 * ut 1 = 1 against 1 * 1 = 1: equal, passes.
		{"capital: equal products pass", func() any { return occupation.CapitalOK(1, 0, 1, 1, 1, 1) }, true},
		// diff 1 * ut 1 = 1 (high word 0) against MaxInt64 * MaxInt64 (high word above 0).
		{"capital: right high word above", func() any { return occupation.CapitalOK(1, 0, math.MaxInt64, math.MaxInt64, 1, 1) }, false},
		// diff 2^64-1 (MaxInt64 - MinInt64) * ut MaxInt64 has a high word above 0 against 1 * 1.
		{"capital: left high word above", func() any { return occupation.CapitalOK(math.MaxInt64, math.MinInt64, 1, 1, math.MaxInt64, 1) }, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.got(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("%s = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

// TestBoundaries_ExitsIgnoreNonPositive verifies that a leaver with a negative
// Unfed or Weight (a hand-built view) is skipped and counts nothing, instead of
// raising the allowance by a negative drop.
//
// Guard: with a negative drop the allowance would grow and a later leaver would
// be served beyond ExitShare.
//
// Requirement: FR-SPN-016, FR-SPN-059
func TestBoundaries_ExitsIgnoreNonPositive(t *testing.T) {
	t.Parallel()
	ls := []occupation.Leaver{
		{Ref: 0, Privation: 20, Unfed: -1, Weight: 5, Money: 1, Upkeep: 1},
		{Ref: 1, Privation: 19, Unfed: 2, Weight: -1, Money: 1, Upkeep: 1},
		{Ref: 2, Privation: 18, Unfed: 1, Weight: 1, Money: 9, Upkeep: 1},
	}
	// Allowance 1: only the third leaver is served, as a whole removal (money 9 >= 1 * 1).
	got, cut := occupation.Exits(nil, ls, 1, 10000)
	want := []occupation.Exit{{Ref: 2, Households: 1, Remove: true, Insolvent: false}}
	if !reflect.DeepEqual(got, want) || cut != 0 {
		t.Errorf("Exits() = %+v cut %d, want %+v cut 0", got, cut, want)
	}
}

// TestBoundaries_AdmitReusesTheBuffers verifies that Admit truncates the slices
// it is given before appending, so a world's scratch can be reused every tick.
//
// Guard: without the truncation a stale admission from the previous region
// would be executed again.
//
// Requirement: FR-SPN-021
func TestBoundaries_AdmitReusesTheBuffers(t *testing.T) {
	t.Parallel()
	stale := []occupation.Admission{{Ref: 99, Admitted: []occupation.DestCount{{Dest: 9, Households: 9}}}}
	staleDC := []occupation.DestCount{{Dest: 9, Households: 9}}
	region := int64(0)
	adm, dc, c := occupation.Admit(stale, staleDC, nil, occupation.Limits{RegionCap: 5, RoleCap: []int64{5}}, &region, []int64{0})
	if len(adm) != 0 || len(dc) != 0 || c != (occupation.Counts{}) {
		t.Errorf("Admit(stale buffers, no candidates) = %+v %+v %+v, want empty", adm, dc, c)
	}
}
