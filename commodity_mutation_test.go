package economysim

import "testing"

// Equivalent mutants of commodity.go that no test can kill (gomu, one-line why each):
//   - load, chargeableSize, chargeableCapacity: every `return 0, ...` -> `return 0, ...`
//     (lines 27, 30, 33, 54, 67) is a no-op: the replacement is the same literal.

// TestLoad_ForeignQuantityPanicMessage pins the exact invariant message of a
// quantity list longer than the commodity table.
//
// Guard: without the length check the runtime's index-out-of-range panic
// (a runtime.Error, not this string) would surface instead, so asserting the
// recovered value kills the mutant that removes the check.
//
// Requirement: FR-GDS-005
func TestLoad_ForeignQuantityPanicMessage(t *testing.T) {
	t.Parallel()
	const want = "economysim: invariant violated: load of a quantity list longer than the commodities"
	defer func() {
		if got := recover(); got != want {
			t.Errorf("load panic = %#v, want %q", got, want)
		}
	}()
	load(worldGoods(), []Quantity{1, 1, 1, 1})
}
