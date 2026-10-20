package economysim

import (
	"cmp"
	"errors"
	"slices"
	"testing"
)

// Equivalent mutants of the phase-6 front of validate.go (Validate, Warnings,
// sortWarnings, structural, viability, addOutcome, emptyOffsets, report;
// gomu, one-line why each) that no test can kill:
//
//   - len(v.rep.errs) == 0 -> <= 0 (Validate, Warnings): a length is never negative.
//   - len(v.rep.errs) > 0 -> != 0 (Warnings, structural): same.
//   - seen[key] = empty removed (emptyOffsets): the cache holds a pure function
//     of the key, so dropping it only recomputes the same answer.
//   - report.err, len(r.errs) == 0 guard (branch_condition, <, <=, empty block):
//     with no errors the rest of err() builds an empty list and errors.Join of
//     nothing is nil, the same result as the early return.
//   - len(shown) > maxReportedErrors -> >=: at exactly the bound the slice
//     shown[:bound] is the whole slice.
//   - make([]error, 0, len(shown)+1) (zero -> one, + -> - * /, one -> zero or
//     two): a capacity hint, and a leading nil that errors.Join drops.
//   - left > 0 -> != 0: left is never negative.

// jobConfig returns a configuration whose warm-start job has no virtual start
// offset in its region (a 60-tick spring recipe), so Validate refuses it.
func jobConfig() Config {
	c := viaSeasonal(60, viaSpring("", ""))
	c.WarmStart = &WarmStart{Groups: []WarmGroup{{Region: "north", Role: "farmer", Jobs: []JobSpec{{Recipe: "farm", InProgressBP: 1_000}}}}}
	return c
}

// TestEmptyOffsets_ErrorCarriesRecipeAndNoBound verifies the one error of an
// empty set of virtual start offsets: its value is the recipe of the job and its
// bound is empty (spec-viability.md 8.5.2).
//
// Requirement: FR-VIA-033
//
// Guard: a bound or a different value would change the text of the error and
// the sort key that ends in them.
func TestEmptyOffsets_ErrorCarriesRecipeAndNoBound(t *testing.T) {
	t.Parallel()
	c := jobConfig()
	var ve *ValidationError
	err := c.Validate()
	if !errors.Is(err, ErrWarmStartJob) || !errors.As(err, &ve) {
		t.Fatalf("Validate = %v, want ErrWarmStartJob", err)
	}
	if ve.Value != "farm" || ve.Bound != "" {
		t.Errorf("Value, Bound = %q, %q, want %q, %q", ve.Value, ve.Bound, "farm", "")
	}
}

// TestAddOutcome_SolverFaultIsReportedAtViability verifies that a defect of the
// solver, which no configuration can cause, is one ErrSolveInconsistent at the
// path viability with no value and no bound, and that the viability errors of
// the outcome come with it.
//
// Requirement: FR-VIA-033
//
// Guard: the fault is reachable only from inside, so a changed path, value or
// bound would go unseen from Validate.
func TestAddOutcome_SolverFaultIsReportedAtViability(t *testing.T) {
	t.Parallel()
	v := &validator{}
	v.addOutcome(viabilityOutcome{
		errs:  []*ViabilityError{{Path: []string{"roles", "idle"}, Rule: "role_cannot_work"}},
		fault: ErrSolveInconsistent,
	})
	if len(v.rep.errs) != 2 {
		t.Fatalf("addOutcome made %d findings, want 2 (the error and the fault)", len(v.rep.errs))
	}
	f := v.rep.errs[1]
	var ve *ValidationError
	if !errors.Is(f.err, ErrSolveInconsistent) || !errors.As(f.err, &ve) {
		t.Fatalf("fault = %v, want a *ValidationError of ErrSolveInconsistent", f.err)
	}
	if !slices.Equal(ve.Path, []string{"viability"}) || ve.Value != "" || ve.Bound != "" {
		t.Errorf("fault = path %v value %q bound %q, want path [viability] and no value or bound", ve.Path, ve.Value, ve.Bound)
	}
}

// TestWarnings_ViabilityWarningsAreSortedIn verifies that the viability warnings
// are appended and then sorted in with the structural ones, by path: a
// structural warning at the path roles sorts after a viability warning at the
// path regions.
//
// Requirement: FR-CFG-042
//
// Guard: without the final sort the structural warning would stay first.
func TestWarnings_ViabilityWarningsAreSortedIn(t *testing.T) {
	t.Parallel()
	c := viaToy()
	viaTwoRegions(&c)
	c.Roles[0].Trades = []string{"grain"} // strategy_default_unused_trades at roles.farmer.trades
	ws := c.Warnings()
	if !viaWarned(ws, "strategy_default_unused_trades", "roles", "farmer", "trades") || !viaWarned(ws, "region_not_self_sufficient", "regions", "south", "commodities", "grain") {
		t.Fatalf("Warnings = %v, want strategy_default_unused_trades and region_not_self_sufficient", ws)
	}
	if !slices.IsSortedFunc(ws, func(a, b Warning) int {
		return cmp.Or(slices.Compare(a.Path, b.Path), cmp.Compare(a.Rule, b.Rule))
	}) {
		t.Errorf("Warnings = %v, want them in order of path and then rule", ws)
	}
}
