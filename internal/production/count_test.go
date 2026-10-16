package production_test

import (
	"math"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/production"
)

// Reasons of the tests, in the master order of PRO 4.2.1 (the package
// interprets none of them).
const (
	rZeroYield = iota + 1
	rNoSite
	rNoTool
	rNoInput
)

const none = production.NoLimit

// TestCount_Cases verifies the least of the weight, the maximum and the caps,
// and the reason of the first cap that is 0.
//
// Requirement: FR-PRO-009
func TestCount_Cases(t *testing.T) {
	t.Parallel()
	L := func(c int64, r int) production.Limit { return production.Limit{Cap: c, Reason: r} }
	tests := []struct {
		name       string
		weight     int64
		maximum    int64
		limits     []production.Limit
		wantK      int64
		wantReason int
	}{
		// min(10, 4 (grain 9 / 2), 3 (one tool item per execution, 3 items)) = 3.
		{"least of weight and caps", 10, 0, []production.Limit{L(none, rZeroYield), L(3, rNoTool), L(4, rNoInput)}, 3, production.NoReason},
		// min(5, 2) = 2: a partial k is not an error and carries no reason.
		{"partial k below the weight", 5, 0, []production.Limit{L(2, rNoInput)}, 2, production.NoReason},
		// min(5, 3 (maximum), 4) = 3.
		{"maximum binds", 5, 3, []production.Limit{L(4, rNoInput)}, 3, production.NoReason},
		// A maximum of 0 means none: min(5, 4) = 4.
		{"zero maximum is none", 5, 0, []production.Limit{L(4, rNoInput)}, 4, production.NoReason},
		// A negative maximum is none and does not wrap: min(5, 4) = 4.
		{"negative maximum is none", 5, -1, []production.Limit{L(4, rNoInput)}, 4, production.NoReason},
		{"min int64 maximum is none", 5, math.MinInt64, nil, 5, production.NoReason},
		// A maximum above the weight changes nothing: min(5, 99) = 5.
		{"maximum above the weight", 5, 99, nil, 5, production.NoReason},
		{"weight 1", 1, 0, []production.Limit{L(7, rNoInput)}, 1, production.NoReason},
		// no_tool (position 5) before no_input (position 7), both 0.
		{"no_tool before no_input", 5, 0, []production.Limit{L(0, rNoTool), L(0, rNoInput)}, 0, rNoTool},
		// zero_yield before both.
		{"zero_yield before both", 5, 0, []production.Limit{L(0, rZeroYield), L(0, rNoTool), L(0, rNoInput)}, 0, rZeroYield},
		{"only no_input is 0", 5, 0, []production.Limit{L(none, rZeroYield), L(2, rNoTool), L(0, rNoInput)}, 0, rNoInput},
		// The yield pair first: a sited recipe with no profile and no tool says no_site.
		{"no_site before no_tool", 5, 0, []production.Limit{L(0, rNoSite), L(0, rNoTool)}, 0, rNoSite},
		// A pair with no limit is ignored, not read as 0: min(5, 2) = 2.
		{"no limit then cap 2", 5, 0, []production.Limit{L(none, rNoTool), L(2, rNoInput)}, 2, production.NoReason},
		// A stock of 0 gives cap 0.
		{"stock 0", 5, 0, []production.Limit{L(production.Cap(0, 2), rNoInput)}, 0, rNoInput},
		// An input quantity larger than the stock: 5 / 7 = 0.
		{"need above stock", 5, 0, []production.Limit{L(production.Cap(5, 7), rNoInput)}, 0, rNoInput},
		// Weight 0 with every cap open: k = 0 and no limit is the reason.
		{"weight 0", 0, 0, []production.Limit{L(3, rNoInput)}, 0, production.NoReason},
		{"negative weight", -4, 0, nil, 0, production.NoReason},
		// A negative cap counts as 0.
		{"negative cap", 5, 0, []production.Limit{L(-3, rNoTool)}, 0, rNoTool},
		// D57: stock 7, input quantity 2, one tool item: 7 / (2 + 1) = 2, no_tool.
		// WearPerUse is durability, not items, so 5 changes nothing.
		{"both input and tool", 5, 0, []production.Limit{L(production.CapInputAndTool(7, 2), rNoTool)}, 2, production.NoReason},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			k, reason := production.Count(tt.weight, tt.maximum, tt.limits)
			if k != tt.wantK || reason != tt.wantReason {
				t.Errorf("Count(%d, %d, %+v) = %d, %d, want %d, %d", tt.weight, tt.maximum, tt.limits, k, reason, tt.wantK, tt.wantReason)
			}
		})
	}
}

// TestCap_Cases verifies the cap of a stock and a per-execution need, and the
// defensive cap of D57.
//
// Requirement: FR-PRO-009
func TestCap_Cases(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		stock, need int64
		want        int64
	}{
		{"7 over 2", 7, 2, 3},              // 7 / 2 = 3
		{"exact division", 6, 2, 3},        // 6 / 2 = 3, nothing lost
		{"need above stock", 5, 7, 0},      // 5 / 7 = 0
		{"stock 0", 0, 2, 0},               // 0 / 2 = 0
		{"need 0 is no limit", 7, 0, none}, // zero need
		{"need 0 and stock 0", 0, 0, none}, // still no limit
		{"negative need is no limit", 7, -1, none},
		{"negative stock", -3, 2, 0},
		{"max stock", math.MaxInt64, 1, math.MaxInt64},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := production.Cap(tt.stock, tt.need); got != tt.want {
				t.Errorf("Cap(%d, %d) = %d, want %d", tt.stock, tt.need, got, tt.want)
			}
		})
	}
	both := []struct {
		name     string
		stock, q int64
		want     int64
	}{
		{"stock 7 q 2", 7, 2, 2},                 // 7 / (2 + 1) = 2
		{"stock 6 q 2", 6, 2, 2},                 // 6 / 3 = 2
		{"stock 2 q 2", 2, 2, 0},                 // 2 / 3 = 0
		{"q 0 is a tool alone", 7, 0, 7},         // 7 / 1 = 7
		{"negative q is a tool alone", 7, -4, 7}, // q clamps to 0: 7 / 1 = 7
		{"q max does not wrap", 7, math.MaxInt64, 0},
	}
	for _, tt := range both {
		t.Run("both "+tt.name, func(t *testing.T) {
			t.Parallel()
			if got := production.CapInputAndTool(tt.stock, tt.q); got != tt.want {
				t.Errorf("CapInputAndTool(%d, %d) = %d, want %d", tt.stock, tt.q, got, tt.want)
			}
		})
	}
}

// TestYieldLimit_D59 verifies every combination of D59, each breaking only its
// rule.
//
// Requirement: FR-PRO-009, FR-PRO-020
func TestYieldLimit_D59(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		sited, has bool
		effective  int64
		wantCap    int64
		wantNoSite bool
	}{
		{"sited, profile, 8 500", true, true, 8500, none, false},
		{"sited, no profile", true, false, 0, 0, true},
		{"sited, no profile, shock 0 (the profile decides)", true, false, 0, 0, true},
		{"sited, no profile, positive factor", true, false, 5000, 0, true},
		{"sited, profile all = 0", true, true, 0, 0, false},
		{"sited, seasonal profile without the season", true, true, 0, 0, false},
		{"sited, site 10 000 and shock 0", true, true, 0, 0, false},
		{"not sited, no profile", false, false, 10000, none, false},
		{"not sited, profile all = 0", false, true, 0, 0, false},
		{"not sited, seasonal profile without the season", false, true, 0, 0, false},
		{"not sited, site 10 000 and technology 0", false, true, 0, 0, false},
		{"not sited, no profile, effective 0", false, false, 0, 0, false},
		// A negative effective factor is "0 or less": cap 0, not NoLimit.
		{"not sited, negative effective", false, false, -1, 0, false},
		{"sited, profile, negative effective", true, true, -5000, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c, noSite := production.YieldLimit(tt.sited, tt.has, tt.effective)
			if c != tt.wantCap || noSite != tt.wantNoSite {
				t.Errorf("YieldLimit(%v, %v, %d) = %d, %v, want %d, %v", tt.sited, tt.has, tt.effective, c, noSite, tt.wantCap, tt.wantNoSite)
			}
		})
	}
}

// TestCount_YieldPairFirst verifies that the yield pair, placed first, wins
// over a tool stock that is also 0.
//
// Requirement: FR-PRO-009
func TestCount_YieldPairFirst(t *testing.T) {
	t.Parallel()
	reason := func(sited, has bool, e int64) int {
		c, noSite := production.YieldLimit(sited, has, e)
		r := rZeroYield
		if noSite {
			r = rNoSite
		}
		_, got := production.Count(5, 0, []production.Limit{{Cap: c, Reason: r}, {Cap: production.Cap(0, 1), Reason: rNoTool}})
		return got
	}
	// The arid region of PRO 12.1: sited, no profile, tool stock 0: no_site.
	if got := reason(true, false, 0); got != rNoSite {
		t.Errorf("sited, no profile, no tool: reason %d, want no_site %d", got, rNoSite)
	}
	// Profile all = 0, tool stock 0: zero_yield.
	if got := reason(true, true, 0); got != rZeroYield {
		t.Errorf("sited, all = 0, no tool: reason %d, want zero_yield %d", got, rZeroYield)
	}
	// A working yield and no tool: no_tool.
	if got := reason(true, true, 8500); got != rNoTool {
		t.Errorf("sited, 8 500, no tool: reason %d, want no_tool %d", got, rNoTool)
	}
}
