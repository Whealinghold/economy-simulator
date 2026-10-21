package economysim

// The test-only seam of the invariants test (plan 1-12, phase 5, D179): a
// one-line wrapper over invariants_internal_test.go for the external package.

// NegativeAccounts lists the open non-boundary accounts that hold a negative
// balance or stock.
func NegativeAccounts(w *World) []string { return negativeAccounts(w) }
