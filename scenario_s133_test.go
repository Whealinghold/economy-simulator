package economysim_test

import (
	"fmt"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// s133Doc is the document of FIS-S2 (spec-fiscal.md 19): a fixed head tax on the cooks of the centre,
// with an exemption, and a payout of the same amount back to the same cooks. The root governs the
// centre (overlay/polity-governed.toml). The amount is a labelled placeholder (N-018, N-027): a
// quarter of a tick's upkeep is not computed here, the exemption is the lever the scenario turns.
func s133Doc(exempt int64) []byte {
	return fmt.Appendf(nil, `
[modules.fiscal.tax.poll]
authority = "root"
target = { kind = "households", region = "centre", role = "cook" }
amount = 8
exempt = %d

[modules.fiscal.payout.alms]
authority = "root"
target = { kind = "households", region = "centre", role = "cook" }
amount = 8
`, exempt)
}

// s133Out is what one run of S133 keeps: counters of the flows, the unfed share of the centre at the
// end and the treasury's start and end.
type s133Out struct {
	taxed, paid, short, cut economysim.Money
	unfed                   int64
	start, end              economysim.Money
}

// runS133 runs the tax with the given exemption for ticks under the audit and returns what it kept.
func runS133(t testing.TB, seed uint64, exempt, ticks int64) s133Out {
	t.Helper()
	cfg := loadPackBytes(t, s133Doc(exempt), fivePolityGoverned...)
	w, k := fiscalWorld(t, cfg, seed)
	start := soleTreasury(t, w)
	fiscalStepTo(t, w, ticks)
	return s133Out{
		taxed: k.applied["poll"], paid: k.applied["alms"],
		short: economysim.Money(k.short["poll"]), cut: economysim.Money(k.cut["alms"]),
		unfed: unfedShare(t, w, regionRank(cfg, "centre")), start: start, end: soleTreasury(t, w),
	}
}

// TestScenarioS133_UpkeepTax reports FIS-S2 of spec-fiscal.md 19 (STB-L174) on the governed warm five,
// seeds 1 to 10, 200 ticks, audit on: a fixed tax on the cooks of the centre and a payout of the same
// amount back to them, with no exemption and with an exemption above every balance. The accounting is
// exact: the sum of Applied of the tax less that of the payout is the change of the root's treasury, no
// flow applies more than its due, and the exemption collects nothing (a tax never takes a balance below
// the exemption, FR-FIS-024) and counts the forgiven payers. The unfed share of the centre, the payers
// short, the recipients cut and the treasury's path are reported, never asserted (N-027).
//
// Requirement: FR-FIS-024, FR-FIS-002, FR-FIS-003, FR-FIS-021, FR-FIS-038
//
// Guard: a collection that left the treasury and the ledger out of step, or an exemption that did not
// protect a balance, shows as a break of the identity or as a positive collection.
func TestScenarioS133_UpkeepTax(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end scenario, skipped with -short")
	}
	const ticks, huge = 200, 1_000_000_000
	fiscalScenario(t, "five", "S133 five+fiscal poll tax with and without an exemption, 200 ticks", scenarioSeedCount(), func(t *testing.T, rep *scenReport, seed uint64) {
		c := fiscalChecker{t, rep, "S133", "five", seed}
		plain, safe := runS133(t, seed, 0, ticks), runS133(t, seed, huge, ticks)
		for _, row := range []struct {
			name string
			o    s133Out
		}{{"plain", plain}, {"exempt", safe}} {
			c.check(row.name+": treasury = start + taxed - paid", row.o.end == row.o.start+row.o.taxed-row.o.paid,
				"end %d, want %d + %d - %d", row.o.end, row.o.start, row.o.taxed, row.o.paid)
		}
		c.check("exemption above every balance collects nothing", safe.taxed == 0, "collected %d", safe.taxed)
		c.check("the plain tax collects", plain.taxed > 0, "collected %d", plain.taxed)
		rep.num(seed, "taxed plain", int64(plain.taxed))
		rep.num(seed, "taxed exempt", int64(safe.taxed))
		rep.num(seed, "paid plain", int64(plain.paid))
		rep.num(seed, "payers short plain", int64(plain.short))
		rep.num(seed, "payers short exempt", int64(safe.short))
		rep.num(seed, "recipients cut plain", int64(plain.cut))
		rep.num(seed, "unfed share centre plain bp", plain.unfed)
		rep.num(seed, "unfed share centre exempt bp", safe.unfed)
		rep.num(seed, "treasury end plain", int64(plain.end))
		rep.num(seed, "treasury end exempt", int64(safe.end))
	})
}
