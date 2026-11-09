package migration_test

import (
	"math"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/migration"
)

// grow is the recipe of MIG 15.1 at a destination: 4 ticks, 2 seed in, 12 staple
// out, priced as the rumours give.
func grow(yield, staple, seed int64) migration.Recipe {
	return migration.Recipe{
		Duration: 4,
		Yield:    yield,
		Outputs:  []migration.Line{{Qty: 12, Price: staple}},
		Inputs:   []migration.Line{{Qty: 2, Price: seed}},
	}
}

// TestIncome_WorkedExampleA verifies the estimate of MIG 15.1 step by step: the
// income per working tick and Vhat at both destinations of the river valley.
//
// Requirement: FR-MIG-014
func TestIncome_WorkedExampleA(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                 string
		recipe               migration.Recipe
		upkeep               []migration.Line
		wantIncome, wantVhat int64
		wantUpkeepCost       int64
	}{
		// value floor(12 * 10 000 * 321 / 10 000) = 3 852; cost 2 * 80 = 160; net
		// 3 692; over 4 ticks 923; upkeep 2 * 321 = 642; 923 * 10 000 / 642 = 14 376.
		{"middle", grow(10_000, 321, 80), []migration.Line{{Qty: 2, Price: 321}}, 923, 14_376, 642},
		// value floor(12 * 12 000 * 366 / 10 000) = 5 270; cost 2 * 82 = 164; net
		// 5 106; over 4 ticks 1 276; upkeep 2 * 366 = 732; 1 276 * 10 000 / 732 = 17 431.
		{"lower", grow(12_000, 366, 82), []migration.Line{{Qty: 2, Price: 366}}, 1_276, 17_431, 732},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			income, priced, ok := migration.Income([]migration.Recipe{tt.recipe})
			if !ok || !priced || income != tt.wantIncome {
				t.Fatalf("Income = %d, %v, %v, want %d, true, true", income, priced, ok, tt.wantIncome)
			}
			u, ok := migration.Cost(tt.upkeep)
			if !ok || u != tt.wantUpkeepCost {
				t.Fatalf("Cost(upkeep) = %d, %v, want %d, true", u, ok, tt.wantUpkeepCost)
			}
			if v, ok := migration.Vhat(income, u); !ok || v != tt.wantVhat {
				t.Errorf("Vhat(%d, %d) = %d, %v, want %d, true", income, u, v, ok, tt.wantVhat)
			}
		})
	}
	// The home figure has the same shape: 640 * 10 000 / (2 * 298 + 405) = 6 393.
	home, _ := migration.Cost([]migration.Line{{Qty: 2, Price: 298}, {Qty: 1, Price: 405}})
	if v, ok := migration.Vhat(640, home); !ok || v != 6_393 {
		t.Errorf("Vhat(640, %d) = %d, %v, want 6393, true", home, v, ok)
	}
}

// TestIncome_EligibleRules verifies which recipes count: a recipe with a missing
// price is not eligible and not priced, a yield of 0 is priced and not eligible,
// a recipe with inputs needs a positive net, a source recipe (no inputs) counts
// at net 0, the greatest income wins, and an income truncates toward zero.
//
// Requirement: FR-MIG-014
func TestIncome_EligibleRules(t *testing.T) {
	t.Parallel()
	src := func(duration, out, price int64, tools ...migration.Tool) migration.Recipe {
		return migration.Recipe{Duration: duration, Yield: 10_000, Outputs: []migration.Line{{Qty: out, Price: price}}, Tools: tools}
	}
	tests := []struct {
		name       string
		recipes    []migration.Recipe
		wantIncome int64
		wantPriced bool
	}{
		{"no recipe", nil, 0, false},
		{"missing output price", []migration.Recipe{grow(10_000, 0, 80)}, 0, false},
		{"missing input price", []migration.Recipe{grow(10_000, 321, 0)}, 0, false},
		{"missing tool price", []migration.Recipe{src(1, 5, 10, migration.Tool{Wear: 1, Durability: 10, Price: 0})}, 0, false},
		{"yield 0 is priced and ineligible", []migration.Recipe{grow(0, 321, 80)}, 0, true},
		// 12 staple at 10 is 120, the 2 seed at 80 cost 160: net -40, with an input.
		{"inputs and a negative net", []migration.Recipe{grow(10_000, 10, 80)}, 0, true},
		// 12 * 14 = 168 against 160: net 8, over 4 ticks 2.
		{"inputs and a positive net", []migration.Recipe{grow(10_000, 14, 80)}, 2, true},
		// value 2 * 6 = 12 against an input costing 12: net exactly 0, with an input.
		{"inputs and a net of 0 is ineligible", []migration.Recipe{{Duration: 1, Yield: 10_000, Outputs: []migration.Line{{Qty: 2, Price: 6}}, Inputs: []migration.Line{{Qty: 1, Price: 12}}}}, 0, true},
		{"source at net 0", []migration.Recipe{src(1, 0, 10)}, 0, true},
		// value 5 * 10 = 50, wear ceil(3 * 10 / 4) = 8: net 42 over 3 ticks, 14.
		{"source with wear", []migration.Recipe{src(3, 5, 10, migration.Tool{Wear: 3, Durability: 4, Price: 10})}, 14, true},
		// net 7 over 2 ticks is 3.5: truncated to 3.
		{"quotient truncates", []migration.Recipe{src(2, 7, 1)}, 3, true},
		// The wear 100 * 10 / 1 = 1 000 exceeds the value 50: net -950 over 3 ticks is -316.67, truncated toward zero to -316 (not floor, -317).
		{"source with a negative net truncates toward zero", []migration.Recipe{src(3, 5, 10, migration.Tool{Wear: 100, Durability: 1, Price: 10})}, -316, true},
		// The negative source recipe first: a priced recipe of yield 0 is not eligible, so it
		// does not lift the income to 0 and the later recipes still count.
		{"yield 0 does not lift a negative income", []migration.Recipe{src(3, 5, 10, migration.Tool{Wear: 100, Durability: 1, Price: 10}), func() migration.Recipe { r := src(1, 5, 10); r.Yield = 0; return r }()}, -316, true},
		{"yield 0 then an eligible recipe", []migration.Recipe{func() migration.Recipe { r := src(1, 5, 10); r.Yield = 0; return r }(), src(1, 2, 10)}, 20, true},
		// Yield 1 is eligible: 10 000 * 1 * 10 / 10 000 = 10.
		{"yield 1", []migration.Recipe{func() migration.Recipe { r := src(1, 10_000, 10); r.Yield = 1; return r }()}, 10, true},
		// An input recipe at net 0 or below is skipped and the next one still counts; a net
		// of 1 is eligible. Output 2 * 6 = 12 against inputs 12 is 0; against 11 is 1.
		{"input recipe at net 0 does not lift a negative income", []migration.Recipe{src(3, 5, 10, migration.Tool{Wear: 100, Durability: 1, Price: 10}), {Duration: 1, Yield: 10_000, Outputs: []migration.Line{{Qty: 2, Price: 6}}, Inputs: []migration.Line{{Qty: 1, Price: 12}}}}, -316, true},
		{"input recipe at net 0 then a good one", []migration.Recipe{{Duration: 1, Yield: 10_000, Outputs: []migration.Line{{Qty: 2, Price: 6}}, Inputs: []migration.Line{{Qty: 1, Price: 12}}}, src(1, 2, 10)}, 20, true},
		{"input recipe at net 1", []migration.Recipe{{Duration: 1, Yield: 10_000, Outputs: []migration.Line{{Qty: 2, Price: 6}}, Inputs: []migration.Line{{Qty: 1, Price: 11}}}}, 1, true},
		{"greatest over several", []migration.Recipe{src(1, 3, 10), src(1, 9, 10), grow(10_000, 14, 80), src(1, 5, 10)}, 90, true},
		{"an unpriced recipe among priced ones is skipped", []migration.Recipe{src(1, 9, 0), src(1, 2, 10)}, 20, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			income, priced, ok := migration.Income(tt.recipes)
			if !ok || income != tt.wantIncome || priced != tt.wantPriced {
				t.Errorf("Income = %d, %v, %v, want %d, %v, true", income, priced, ok, tt.wantIncome, tt.wantPriced)
			}
		})
	}
}

// TestIncome_BadInput verifies that a duration below 1, a tool of durability 0
// and products past int64 are reported with ok false and never wrapped.
//
// Requirement: FR-MIG-040, NFR-MIG-003
func TestIncome_BadInput(t *testing.T) {
	t.Parallel()
	big := int64(math.MaxInt64 / 4)
	tests := []struct {
		name   string
		recipe migration.Recipe
	}{
		{"duration 0", migration.Recipe{Duration: 0, Yield: 10_000}},
		{"tool of durability 0", migration.Recipe{Duration: 1, Yield: 10_000, Outputs: []migration.Line{{Qty: 1, Price: 1}}, Tools: []migration.Tool{{Wear: 1, Durability: 0, Price: 1}}}},
		{"quantity times yield overflows", migration.Recipe{Duration: 1, Yield: 10_000, Outputs: []migration.Line{{Qty: big, Price: 1}}}},
		{"value overflows", migration.Recipe{Duration: 1, Yield: 10_000, Outputs: []migration.Line{{Qty: 1 << 40, Price: big}}}},
		{"inputs overflow", migration.Recipe{Duration: 1, Yield: 10_000, Outputs: []migration.Line{{Qty: 1, Price: 1}}, Inputs: []migration.Line{{Qty: big, Price: 8}}}},
		{"the sum of values overflows", migration.Recipe{Duration: 1, Yield: 10_000, Outputs: []migration.Line{{Qty: 1, Price: math.MaxInt64/2 + 1}, {Qty: 1, Price: math.MaxInt64/2 + 1}}}},
		{"the cost with the wear overflows", migration.Recipe{Duration: 1, Yield: 10_000, Outputs: []migration.Line{{Qty: 1, Price: 1}}, Inputs: []migration.Line{{Qty: 1, Price: math.MaxInt64 - 1}}, Tools: []migration.Tool{{Wear: 2, Durability: 1, Price: 1}}}},
		{"net overflows", migration.Recipe{Duration: 1, Yield: 10_000, Outputs: []migration.Line{{Qty: 1, Price: 1}}, Inputs: []migration.Line{{Qty: 1, Price: math.MaxInt64}, {Qty: 1, Price: math.MaxInt64}}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if income, priced, ok := migration.Income([]migration.Recipe{tt.recipe}); ok || income != 0 || priced {
				t.Errorf("Income = %d, %v, %v, want 0, false, false", income, priced, ok)
			}
		})
	}
}

// TestVhat_Overflow verifies that a ratio past int64 and a cost below 1 are
// reported, that a negative income truncates toward zero, and that Cost reports
// an overflowing sum.
//
// Requirement: FR-MIG-040, NFR-MIG-003
func TestVhat_Overflow(t *testing.T) {
	t.Parallel()
	if v, ok := migration.Vhat(math.MaxInt64, 1); ok {
		t.Errorf("Vhat(MaxInt64, 1) = %d, true, want overflow", v)
	}
	for _, cost := range []int64{0, -5} {
		if v, ok := migration.Vhat(100, cost); ok || v != 0 {
			t.Errorf("Vhat(100, %d) = %d, %v, want 0, false (a refused cost)", cost, v, ok)
		}
	}
	// -1 * 10 000 / 3 = -3 333.3, truncated toward zero.
	if v, ok := migration.Vhat(-1, 3); !ok || v != -3_333 {
		t.Errorf("Vhat(-1, 3) = %d, %v, want -3333, true", v, ok)
	}
	if v, ok := migration.Vhat(math.MaxInt64/10_000, 1); !ok || v != math.MaxInt64/10_000*10_000 {
		t.Errorf("Vhat at the largest exact income = %d, %v", v, ok)
	}
	lines := []migration.Line{{Qty: math.MaxInt64 / 2, Price: 1}, {Qty: math.MaxInt64 / 2, Price: 1}, {Qty: 2, Price: 1}}
	if c, ok := migration.Cost(lines); ok || c != 0 {
		t.Errorf("Cost of a sum past int64 = %d, %v, want 0, false", c, ok)
	}
	if c, ok := migration.Cost([]migration.Line{{Qty: math.MaxInt64, Price: 2}}); ok || c != 0 {
		t.Errorf("Cost of a product past int64 = %d, %v, want 0, false", c, ok)
	}
}
