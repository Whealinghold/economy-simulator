package economysim

import "testing"

// overflowConfig is one employer role with one group and one recipe whose hire
// value is the sum of n outputs of one commodity at the given price and quantity,
// less five units of the same commodity as inputs. It is built by hand, because a
// validated configuration keeps these products inside int64 (architecture.md 3.5)
// and the warnings must still be right on arithmetic the validator cannot reach.
func overflowConfig(n int, price Price, qty Quantity) Config {
	outs := make([]Output, n)
	for i := range outs {
		outs[i] = Output{Commodity: "g", Quantity: qty, CostShare: 10_000 / int64(n)}
	}
	return Config{
		Commodities: []Commodity{{ID: "g", ReferencePrice: price}},
		Recipes:     []Recipe{{ID: "r", Duration: 1, Inputs: []Item{{Commodity: "g", Quantity: 5}}, Outputs: outs}},
		Roles: []Role{{ID: "boss", Recipes: []string{"r"}, StartMoney: 1, Employer: &Employer{
			Recipes: []string{"r"}, MaxHires: 1, WorkerShareBP: 5_000, HireYieldBP: 10_000}}},
		Groups: []Group{{Region: "a", Role: "boss", Households: 1}},
	}
}

// TestEmployerPriceWarnings_OverflowIsNotNeverProfitable verifies that a hire
// list valued with arithmetic that overflows draws no price warning, and that a
// recipe that is truly unprofitable still draws hire_never_profitable.
//
// Fix: a hire value that did not fit int64 was returned as 0, so the recipe read
// as net below 1 and the warning claimed "never profitable" for a computation
// that had failed. The warning must say nothing when the value is unknown.
//
// Requirement: FR-EMP-059
func TestEmployerPriceWarnings_OverflowIsNotNeverProfitable(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		c    Config
		want int
	}{
		// Ten outputs of 10^9 units at 10^9 are 10^19 in all, above the int64 limit
		// of about 9.22 * 10^18, so the sum overflows although each product fits.
		{"sum of outputs overflows", overflowConfig(10, MaxPrice, MaxQuantity), 0},
		// The product of one output is 10^9 * 10^9 = 10^18 and fits; only the
		// first-level check passes, so the control is not an overflow case.
		// Output 1 * 1 = 1 less inputs 5 * 1 = 5 is -4 per tick: truly never profitable.
		{"truly unprofitable still warns", overflowConfig(1, 1, 1), 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			lk := newLookup(&tt.c)
			got := employerPriceWarnings(&tt.c, lk, &tt.c.Roles[0], nil)
			n := 0
			for _, w := range got {
				if w.Rule == "hire_never_profitable" {
					n++
				}
			}
			if n != tt.want {
				t.Errorf("employerPriceWarnings(%s) hire_never_profitable count = %d, want %d (warnings %v)", tt.name, n, tt.want, got)
			}
			if tt.want == 0 && len(got) != 0 {
				t.Errorf("employerPriceWarnings(%s) = %v, want none on arithmetic that overflowed", tt.name, got)
			}
		})
	}
}
