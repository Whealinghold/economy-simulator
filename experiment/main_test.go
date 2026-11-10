package experiment_test

import (
	"flag"
	"fmt"
	"os"
	"testing"
)

// updateFlag rewrites the golden file of this package (testing.md 7.2: read it
// first). It mirrors the flag of the root test package.
var updateFlag = flag.Bool("update", false, "rewrite the golden files (testing.md 7.2: read it first)")

// TestMain refuses -update under -race before any test runs: a golden recorded
// there would not be the one of the gate F build (plan 1 D228, mirrored).
func TestMain(m *testing.M) {
	flag.Parse()
	if raceEnabled && *updateFlag {
		fmt.Fprintln(os.Stderr, "-update is refused under -race; run without -race")
		os.Exit(2)
	}
	os.Exit(m.Run())
}
