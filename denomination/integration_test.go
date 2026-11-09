package denomination_test

import (
	"math"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/denomination"
)

// TestFormat_RootMoneyThroughInt64 verifies the consumer path of the package:
// money computed with the root's checked value types is converted with
// int64(m) and formatted, with no import in the other direction.
//
// Requirement: FR-CUR-012, FR-CUR-017
// Guard: Format takes int64, not Money (the root may not import this package).
// If Money stopped being an int64 count of minor units, the consumer's edge
// conversion would break and nothing else in this package would notice.
func TestFormat_RootMoneyThroughInt64(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	t.Parallel()
	f := mustFormatter(t, denomination.RPG(), 2)

	// 123 units at a price of 100: 123 * 100 = 12 300 = 1 gp (10 000) + 2 sp (2 000) + 3 cp (300).
	total, ok := economysim.Price(100).Times(economysim.Quantity(123))
	if !ok {
		t.Fatalf("Price(100).Times(123) overflowed")
	}
	if got, want := f.Format(int64(total)), "1 gp 2 sp 3.00 cp"; got != want {
		t.Errorf("Format(int64(%v)) = %q, want %q", total, got, want)
	}

	// 12 300 - 15 220 = -2 920, the spec's negative row.
	diff, ok := total.Sub(economysim.Money(15_220))
	if !ok {
		t.Fatalf("Money.Sub overflowed")
	}
	if got, want := f.Format(int64(diff)), "-2 sp 9.20 cp"; got != want {
		t.Errorf("Format(int64(%v)) = %q, want %q", diff, got, want)
	}

	// The extreme of the value type formats without overflow.
	if got := f.Format(int64(economysim.Money(math.MinInt64))); !strings.HasPrefix(got, "-922337203685477 gp") {
		t.Errorf("Format(int64(MinInt64 Money)) = %q, want a -922337203685477 gp prefix", got)
	}
}

// TestRPG_MatchesTheSpecTableInTheSpecFile verifies, end to end, that the RPG
// preset and the formatter reproduce every row of the table in section 4.3 of
// spec-currency.md, read from the file itself.
//
// Requirement: FR-CUR-017
// Guard: the unit test copies the table by hand; this one fails when the spec's
// table and the preset drift apart in either direction.
func TestRPG_MatchesTheSpecTableInTheSpecFile(t *testing.T) {
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	t.Parallel()
	raw, err := os.ReadFile("../doc/specs/spec-currency.md")
	if err != nil {
		t.Fatalf("read spec: %v", err)
	}
	row := regexp.MustCompile("^\\| (-?[0-9 ]+?) +\\| `([^`]+)` +\\| `([^`]+)` +\\|$")
	f0 := mustFormatter(t, denomination.RPG(), 0)
	f2 := mustFormatter(t, denomination.RPG(), 2)
	rows := 0
	for _, line := range strings.Split(string(raw), "\n") {
		m := row.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		amount, err := strconv.ParseInt(strings.ReplaceAll(m[1], " ", ""), 10, 64)
		if err != nil {
			t.Fatalf("parse amount %q: %v", m[1], err)
		}
		rows++
		if got := f0.Format(amount); got != m[2] {
			t.Errorf("RPG Format(%d) at precision 0 = %q, spec says %q", amount, got, m[2])
		}
		if got := f2.Format(amount); got != m[3] {
			t.Errorf("RPG Format(%d) at precision 2 = %q, spec says %q", amount, got, m[3])
		}
	}
	if rows != 5 {
		t.Errorf("read %d rows of the spec table, want 5: the table moved or its format changed", rows)
	}
}
