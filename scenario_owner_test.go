package economysim_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"testing"
)

// stageTwoScenarios are the scenarios of spec-reference-chain.md 7.6 and 7.11 that Stage 2
// assigns, by range of S-numbers and the plan that owns the tests of each (plan 2-24
// Phase 1 item 4). S77, S78 and S84 to S90 are harness tests that REF does not build and
// are not in it; the legs that plans 6, 9, 14 and 16 hand to plan 17 (S36, S45, S52, S66)
// are in their own plan's range.
var stageTwoScenarios = []struct {
	from, to int
	owner    string
}{
	{28, 29, "plan 3 (REG)"},
	{30, 32, "plans 12 and 15 (TRN, MER)"},
	{33, 36, "plans 6 and 7 (MON)"},
	{37, 40, "plan 11 (POP)"},
	{41, 45, "plan 9 (RES); S41 plan 11, S45 plan 17"},
	{46, 50, "plan 13 (INF)"},
	{51, 55, "plan 16 (EXT)"},
	{56, 60, "plan 10 (SUB)"},
	{61, 66, "plan 14 (MIG)"},
	{67, 72, "plans 15 and 16 (MER)"},
	{73, 76, "plan 24 (WLD), unit models"},
	{79, 83, "plan 17 (INT)"},
}

// scenariosNotBuilt are the numbers of stageTwoScenarios that have no test yet, the staged
// form of architecture.md 9.7: the test fails on a listed number that has a test (a stale
// entry) and on a number with no test that is not listed. Whoever builds a scenario deletes
// its entry.
var scenariosNotBuilt = map[int]string{
	35: "plan 7 (MON-S3, the pool grant); not built when plan 2-24 was written",
	46: "plan 13: S46 (a shipper on a rumoured gap) is a waiting report inside TestScenarioS47_, with no test of its own",
	73: "plan 24: the unit model of the ordering device, WLD-S1 (spec-world.md 17)",
	74: "plan 24: the unit model of the review rule, WLD-S2",
	75: "plan 24: WLD-S3, the cold leg of S18 and the Duration 10 leg, which waits for a pack with such a recipe",
	76: "plan 24: the unit model of the stream and the two runs, WLD-S4",
}

var scenarioTestRe = regexp.MustCompile(`(?m)^func TestScenarioS(\d+)_`)

// scenarioTestNumbers returns the S-numbers that have a TestScenarioS<nn>_ function in the
// root package's test files.
func scenarioTestNumbers(t *testing.T) map[int]bool {
	t.Helper()
	files, err := filepath.Glob("*_test.go")
	if err != nil {
		t.Fatal(err)
	}
	out := map[int]bool{}
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range scenarioTestRe.FindAllSubmatch(b, -1) {
			n, _ := strconv.Atoi(string(m[1]))
			out[n] = true
		}
	}
	return out
}

// TestScenarios_StageTwoAllHaveTests verifies that every scenario number that Stage 2
// assigns (stageTwoScenarios) has a TestScenarioS<nn>_ test or an entry in
// scenariosNotBuilt, that no listed number has a test, and that no scenario test of S28 or
// above names a number outside the table.
//
// Requirement: FR-REF-038
//
// Guard: a scenario assigned to a plan and never written would otherwise be found at the
// exit, not at the plan that owed it.
func TestScenarios_StageTwoAllHaveTests(t *testing.T) {
	t.Parallel()
	have := scenarioTestNumbers(t)
	inTable := map[int]bool{}
	for _, r := range stageTwoScenarios {
		for n := r.from; n <= r.to; n++ {
			inTable[n] = true
			_, listed := scenariosNotBuilt[n]
			switch {
			case !have[n] && !listed:
				t.Errorf("S%d (%s) has no TestScenarioS%d_ test and is not in scenariosNotBuilt", n, r.owner, n)
			case have[n] && listed:
				t.Errorf("S%d has a test and is still in scenariosNotBuilt (%s), want the entry deleted", n, scenariosNotBuilt[n])
			}
		}
	}
	for n := range have {
		if n >= 28 && !inTable[n] {
			t.Errorf("TestScenarioS%d_ is a scenario of no range in stageTwoScenarios, want its owner named in the table", n)
		}
	}
	for n := range scenariosNotBuilt {
		if !inTable[n] {
			t.Errorf("scenariosNotBuilt names S%d, which is in no range of stageTwoScenarios", n)
		}
	}
}
