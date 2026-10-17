package household_test

import (
	"go/parser"
	"go/token"
	"math"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/household"
)

// TestNet_Examples verifies the value of a recipe: the worked case of AGT 16.1,
// the floor of an output, a tool's wear and a rent added as formed amounts, and
// a negative net.
//
// Requirement: FR-AGT-034
func TestNet_Examples(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		add  func(n *household.Net)
		want int64
	}{
		// 1 * 10 000 * 900 / 10 000 = 900, less 2 * 200 = 400: net 500.
		{"example 16.1", func(n *household.Net) { n.Output(1, 10_000, 900); n.Input(2, 200) }, 500},
		// 3 * 8 500 * 101 / 10 000 = 257.55, floored to 257.
		{"output floors", func(n *household.Net) { n.Output(3, 8500, 101) }, 257},
		// Value 100, inputs 2 * 30 = 60, wear 5, rent 40: 100 - 105 = -5.
		{"wear and rent", func(n *household.Net) {
			n.Output(1, 10_000, 100)
			n.Input(2, 30)
			n.Cost(5)
			n.Cost(40)
		}, -5},
		{"empty", func(n *household.Net) {}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var n household.Net
			tt.add(&n)
			if got, ok := n.Value(); got != tt.want || !ok {
				t.Errorf("Net.Value() = %d, %v, want %d, true", got, ok, tt.want)
			}
		})
	}
}

// TestNet_OverflowIsSticky verifies that each operation reports an overflow and
// that nothing added afterwards clears it.
//
// Requirement: NFR-AGT-003
func TestNet_OverflowIsSticky(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		add  func(n *household.Net)
	}{
		{"output product", func(n *household.Net) { n.Output(math.MaxInt64, 2, 1) }},
		{"output quotient", func(n *household.Net) { n.Output(math.MaxInt64, 10_000, 100) }},
		{"output sum", func(n *household.Net) { n.Output(1, 10_000, math.MaxInt64); n.Output(1, 10_000, 1) }},
		{"input product", func(n *household.Net) { n.Input(math.MaxInt64, 2) }},
		{"cost sum", func(n *household.Net) { n.Cost(math.MaxInt64); n.Cost(1) }},
		{"net difference", func(n *household.Net) { n.Output(1, 10_000, math.MaxInt64); n.Cost(-1) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var n household.Net
			tt.add(&n)
			n.Output(1, 10_000, 1) // after the overflow nothing changes
			n.Input(1, 1)
			n.Cost(1)
			if got, ok := n.Value(); ok || got != 0 {
				t.Errorf("Net.Value() = %d, %v, want 0, false", got, ok)
			}
		})
	}
}

// TestBetter_Examples verifies the comparison of net per tick: equal ratios
// tie, signs, a duration of 1 and the largest values in 128 bits.
//
// Requirement: FR-AGT-034
func TestBetter_Examples(t *testing.T) {
	t.Parallel()
	m := int64(math.MaxInt64)
	tests := []struct {
		name                   string
		netA, durA, netB, durB int64
		want                   bool
	}{
		{"greater", 500, 2, 200, 1, true},
		{"smaller", 200, 1, 500, 2, false},
		// 1/2 = 3/6: a tie is false both ways.
		{"equal ratio", 1, 2, 3, 6, false},
		{"equal ratio reversed", 3, 6, 1, 2, false},
		{"zero against zero", 0, 3, 0, 1, false},
		{"positive over zero", 1, 9, 0, 1, true},
		{"zero over negative", 0, 1, -1, 9, true},
		{"negative over positive", -1, 1, 1, 9, false},
		// -1/9 is above -1/2: the smaller magnitude wins among negatives.
		{"less negative", -1, 9, -1, 2, true},
		{"more negative", -1, 2, -1, 9, false},
		{"duration 1", 7, 1, 6, 1, true},
		// m/m = 1 against (m-1)/m: the products exceed 64 bits.
		{"largest greater", m, m, m - 1, m, true},
		{"largest tie", m, m, m, m, false},
		{"largest smaller", m - 1, m, m, m, false},
		{"min net", math.MinInt64, 1, math.MinInt64 + 1, 1, false},
		{"min net reversed", math.MinInt64 + 1, 1, math.MinInt64, 1, true},
		{"duration below 1", 5, 0, 1, 1, false},
		{"other duration below 1", 5, 1, 1, -1, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := household.Better(tt.netA, tt.durA, tt.netB, tt.durB); got != tt.want {
				t.Errorf("Better(%d, %d, %d, %d) = %v, want %v", tt.netA, tt.durA, tt.netB, tt.durB, got, tt.want)
			}
		})
	}
}

// TestEligible verifies that a source is eligible at any net and a recipe with
// inputs only above 0.
//
// Requirement: FR-AGT-035
func TestEligible(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		net    int64
		source bool
		want   bool
	}{
		{"source with negative net", -10, true, true},
		{"source with zero net", 0, true, true},
		{"inputs with negative net", -10, false, false},
		{"inputs with zero net", 0, false, false},
		{"inputs with positive net", 1, false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := household.Eligible(tt.net, tt.source); got != tt.want {
				t.Errorf("Eligible(%d, %v) = %v, want %v", tt.net, tt.source, got, tt.want)
			}
		})
	}
}

// TestOutputTarget verifies the rounding up, a zero cover, the weight as a
// factor and the overflow.
//
// Requirement: FR-AGT-036, NFR-AGT-003
func TestOutputTarget(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                string
		w, q, e, cover, dur int64
		want                int64
		wantOK              bool
	}{
		// 1 * 1 * 10 000 * 1 / (10 000 * 1) = 1.
		{"one", 1, 1, 10_000, 1, 1, 1, true},
		// 1 * 3 * 8 500 * 1 / (10 000 * 2) = 1.275, rounded up to 2.
		{"rounds up", 1, 3, 8500, 1, 2, 2, true},
		// 40 * 3 * 10 000 * 2 / (10 000 * 4) = 60 exactly.
		{"exact", 40, 3, 10_000, 2, 4, 60, true},
		// Large exact quotient: a changed 10 000 or a perturbed denominator moves it.
		{"large exact", 1_000_000, 1, 10_000, 1, 1, 1_000_000, true},
		// 1 000 000 * 10 001 / 10 000 = 1 000 100 exactly.
		{"large exact 2", 1_000_000, 1, 10_001, 1, 1, 1_000_100, true},
		// 999 999 * 10 001 / 10 000 = 1 000 098.9999, rounded up to 1 000 099.
		{"large rounds up", 999_999, 1, 10_001, 1, 1, 1_000_099, true},
		// 1e10 / 30 000 = 333 333.33, rounded up to 333 334.
		{"large duration", 1_000_000, 1, 10_000, 1, 3, 333_334, true},
		{"no cover", 5, 3, 10_000, 0, 1, 0, true},
		{"no weight", 0, 3, 10_000, 1, 1, 0, true},
		{"negative factors count as 0", -5, 3, 10_000, 1, 1, 0, true},
		{"no duration", 1, 1, 10_000, 1, 0, 0, false},
		{"product overflows", math.MaxInt64, 2, 10_000, 1, 1, 0, false},
		{"third product overflows", 1 << 32, 1 << 31, 10_000, 1, 1, 0, false},
		{"denominator overflows", 1, 1, 10_000, 1, math.MaxInt64, 0, false},
		{"quotient overflows", 1 << 40, 1, 1 << 20, 1 << 20, 1, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := household.OutputTarget(tt.w, tt.q, tt.e, tt.cover, tt.dur)
			if got != tt.want || ok != tt.wantOK {
				t.Errorf("OutputTarget(%d, %d, %d, %d, %d) = %d, %v, want %d, %v", tt.w, tt.q, tt.e, tt.cover, tt.dur, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

// TestHalted verifies the halt at its exact boundary, the 128-bit comparison
// and that a target of 0 never halts.
//
// Requirement: FR-AGT-036, FR-AGT-070
func TestHalted(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                 string
		stock, target, ratio int64
		want                 bool
	}{
		// Ratio 5 000, target 10: the boundary is stock 5: 5 * 10 000 == 5 000 * 10.
		{"at the boundary", 5, 10, 5000, true},
		{"just below", 4, 10, 5000, false},
		{"above", 6, 10, 5000, true},
		// Target 0 never halts, whatever the stock and the ratio.
		{"target 0 stock 0", 0, 0, 5000, false},
		{"target 0 stock above", 100, 0, 5000, false},
		{"target 0 ratio 0", 0, 0, 0, false},
		{"negative target", 5, -1, 5000, false},
		// Ratio 0 halts at any stock against a positive target.
		{"ratio 0", 0, 10, 0, true},
		{"negative stock counts as 0", -5, 10, 5000, false},
		// 128 bits: Max * 10 000 against Max * 10 000 is the boundary.
		{"large boundary", math.MaxInt64, math.MaxInt64, 10_000, true},
		{"large below", math.MaxInt64 - 1, math.MaxInt64, 10_000, false},
		{"large ratio", 1, math.MaxInt64, math.MaxInt64, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := household.Halted(tt.stock, tt.target, tt.ratio); got != tt.want {
				t.Errorf("Halted(%d, %d, %d) = %v, want %v", tt.stock, tt.target, tt.ratio, got, tt.want)
			}
		})
	}
}

// TestHalted_NoCoverNeverHalts verifies the OutputCoverTicks 0 case end to end:
// the target is 0, so no stock halts the recipe.
//
// Requirement: FR-AGT-036
//
// Guard: the literal inequality Stock * 10 000 >= Ratio * 0 is always true, so
// without the target-0 rule every recipe with OutputCoverTicks 0 would halt.
func TestHalted_NoCoverNeverHalts(t *testing.T) {
	t.Parallel()
	target, ok := household.OutputTarget(40, 3, 10_000, 0, 1)
	if target != 0 || !ok {
		t.Fatalf("OutputTarget(.., cover 0, ..) = %d, %v, want 0, true", target, ok)
	}
	for _, stock := range []int64{0, 1, 1000} {
		if household.Halted(stock, target, 30_000) {
			t.Errorf("Halted(%d, %d, 30000) = true, want false", stock, target)
		}
	}
}

// TestKeepFoodCap verifies the cap at stock == U (0), U + q (1) and between.
//
// Requirement: FR-AGT-037
func TestKeepFoodCap(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		stock, u, q int64
		want        int64
	}{
		{"stock equals upkeep", 12, 12, 3, 0},
		{"stock below upkeep", 5, 12, 3, 0},
		{"one execution", 15, 12, 3, 1},
		{"between", 17, 12, 3, 1},
		{"several", 30, 12, 3, 6},
		{"no upkeep", 7, 0, 2, 3},
		{"negative upkeep counts as 0", 7, -4, 2, 3},
		{"no quantity", 7, 0, 0, 0},
		{"largest", math.MaxInt64, 0, 1, math.MaxInt64},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := household.KeepFoodCap(tt.stock, tt.u, tt.q); got != tt.want {
				t.Errorf("KeepFoodCap(%d, %d, %d) = %d, want %d", tt.stock, tt.u, tt.q, got, tt.want)
			}
		})
	}
}

// TestRequest verifies the minimum of the demand and the cap and that a cap or
// a demand of 0 is ok false, never (0, true).
//
// Requirement: FR-AGT-037, FR-AGT-034
func TestRequest(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		exec, kcap int64
		want       int64
		wantOK     bool
	}{
		{"cut to the cap", 5, 3, 3, true},
		{"demand below the cap", 2, 3, 2, true},
		{"equal", 4, 4, 4, true},
		{"cap 0", 5, 0, 0, false},
		{"executions 0", 0, 5, 0, false},
		{"negative cap", 5, -1, 0, false},
		{"negative executions", -1, 5, 0, false},
		{"both 0", 0, 0, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := household.Request(tt.exec, tt.kcap)
			if got != tt.want || ok != tt.wantOK || (got == 0 && ok) {
				t.Errorf("Request(%d, %d) = %d, %v, want %d, %v", tt.exec, tt.kcap, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

// TestWorkRules_IgnoreTheWeight verifies that the weight enters the work-side
// rules only through OutputTarget, as ceil(w * X) on the total, and that Halted
// then compares exactly. Better, Eligible, KeepFoodCap and Request take no
// weight, so they cannot depend on it and have nothing to assert here.
//
// Requirement: FR-AGT-070
func TestWorkRules_IgnoreTheWeight(t *testing.T) {
	t.Parallel()
	// X = 3 * 8 500 * 2 / (10 000 * 3) = 51 000 / 30 000 = 1.7 per household.
	const q, e, cover, dur = 3, 8500, 2, 3
	for _, w := range []int64{1, 30, 3000} {
		got, ok := household.OutputTarget(w, q, e, cover, dur)
		// ceil(w * 51 000 / 30 000) with integer arithmetic.
		want := (w*q*e*cover + 10_000*dur - 1) / (10_000 * dur)
		if !ok || got != want {
			t.Errorf("OutputTarget(%d, ...) = %d, %v, want %d, true", w, got, ok, want)
		}
		// Halted is a threshold at stock * 10 000 >= ratio * target: at ratio
		// 5000 the least halting stock is ceil(target / 2).
		least := (got + 1) / 2
		if !household.Halted(least, got, 5000) || household.Halted(least-1, got, 5000) {
			t.Errorf("Halted threshold for target %d is not at stock %d", got, least)
		}
	}
}

// TestOutputTarget_OneCeiling verifies that the ceiling is taken once, on the
// whole product, not per factor.
//
// Requirement: FR-AGT-036
func TestOutputTarget_OneCeiling(t *testing.T) {
	t.Parallel()
	// Make 0.5 is effective 5 000 of 10 000 with q 1; weight 1, duration 1,
	// cover 10: 1 * 1 * 5 000 * 10 / (10 000 * 1) = 5 exactly. Rounding the
	// 0.5 up to 1 first would give 10.
	got, ok := household.OutputTarget(1, 1, 5000, 10, 1)
	if !ok || got != 5 {
		t.Errorf("OutputTarget(1, 1, 5000, 10, 1) = %d, %v, want 5, true", got, ok)
	}
}

// TestWorkRules_AllocateNothing verifies that the work rules allocate nothing.
//
// Requirement: NFR-AGT-001
func TestWorkRules_AllocateNothing(t *testing.T) {
	var sink int64
	allocs := testing.AllocsPerRun(100, func() {
		var n household.Net
		n.Output(1, 10_000, 900)
		n.Input(2, 200)
		n.Cost(5)
		v, _ := n.Value()
		tg, _ := household.OutputTarget(40, 3, 8500, 2, 3)
		r, _ := household.Request(5, household.KeepFoodCap(30, 12, 3))
		if household.Better(v, 2, 3, 1) && household.Eligible(v, false) && household.Halted(5, tg, 5000) {
			sink++
		}
		sink += r
	})
	if allocs != 0 {
		t.Errorf("work rules allocated %v times per run, want 0 (sink %d)", allocs, sink)
	}
}

// TestPackage_ImportsOnlyStdlibAndChecked verifies that no non-test file of
// the package imports anything but the standard library and internal/checked.
//
// Guard: the architecture import guard bans only third-party paths and the
// packages above internal/, so it does not stop the package importing
// internal/production or internal/welfare (D80, D90); the root passes those
// numbers in.
func TestPackage_ImportsOnlyStdlibAndChecked(t *testing.T) {
	t.Parallel()
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	for _, name := range files {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, name, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatal(err)
		}
		for _, imp := range f.Imports {
			path, _ := strconv.Unquote(imp.Path.Value)
			first, _, _ := strings.Cut(path, "/")
			if strings.Contains(first, ".") && path != "github.com/Whealinghold/economy-sim/internal/checked" {
				t.Errorf("%s imports %q, want only the standard library and internal/checked", name, path)
			}
		}
	}
}

// BenchmarkRecipeChoice measures the per-recipe cost of the work rules.
func BenchmarkRecipeChoice(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		var n household.Net
		n.Output(1, 10_000, 900)
		n.Input(2, 200)
		v, _ := n.Value()
		household.Better(v, 2, 300, 1)
	}
}
