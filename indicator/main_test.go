package indicator_test

import (
	"flag"
	"fmt"
	"os"
	"testing"
)

// updateFlag rewrites the golden file of this package (testing.md 7.2: read it
// first). It mirrors the flag of the root test package (plan 2-18 D576).
var updateFlag = flag.Bool("update", false, "rewrite the golden files (testing.md 7.2: read it first)")

// TestMain refuses -update under -race before any test runs: the seed set is
// reduced there, and a written golden would say "seeds 1-4" of a run that did not
// make it (plan 1 D228, mirrored).
func TestMain(m *testing.M) {
	flag.Parse()
	if raceEnabled && *updateFlag {
		fmt.Fprintln(os.Stderr, "-update is refused under -race: the seed set is reduced; run without -race")
		os.Exit(2)
	}
	os.Exit(m.Run())
}

// seedCount is the seed count of a run-level test with `full` seeds in this build:
// all of them, or the first alone under -race.
func seedCount(full int) int {
	if raceEnabled {
		return 1
	}
	return full
}
