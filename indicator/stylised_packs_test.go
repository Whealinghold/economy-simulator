package indicator_test

import (
	"testing"

	"github.com/Whealinghold/economy-sim/internal/verdict"
)

// The fact tests of group C, the larger packs (plan 2-19 Phase 7, D596): the
// umbrella fact G01 and the warm-start fact H02 at 1 095 ticks on `two`, `three`,
// `one-demand` (seeds 1 to 4) and `five` (seeds 1 and 2, REF 8.2), and the
// demand facts W01 and M04. A pack that has collapsed under its own rules
// (N-312, N-314) makes the facts that need a living economy findings.

// largerPacks are the packs of G01 and H02 with the band file each reads.
var largerPacks = []stylisedPack{packTwo, packThree, packOneDemand, packFive}

// packFacts is the fact set of a larger pack beyond G01: P01, M01 and M02 (exact,
// reused from the invariants) and H02 over the year windows.
func packFacts(t testing.TB, p stylisedPack) *factSet {
	seeds := packSeeds(t, p)
	return memoFacts(seedKey("pack"+p.name, seeds), func() *factSet {
		f, runs, bands := newFactSet(), loadRuns(t, p, seeds), readBands(t, p.name)
		f.exactRows(p, seeds, runs)
		h02(f, p.name, seeds, runs, bands)
		return f
	})
}

// TestStylised_Packs_G01_H02 runs `two`, `three`, `one-demand` and `five` warm for
// 1 095 ticks and judges STY-G01 (every detector over the year windows: R7 and R8
// are `none` on these unperturbed runs, an unset threshold is reported, any other
// `unknown` or a `detected` regime fails the fact), STY-H02 (the offset from the
// warm table, the money per household, the least households) and the exact facts
// P01, M01 and M02.
//
// Requirement: FR-STY-054, FR-STY-057, FR-STY-060, FR-STY-082, FR-STY-086, FR-REF-022, FR-REF-026
func TestStylised_Packs_G01_H02(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("run-level test, skipped with -short")
	}
	for _, p := range largerPacks {
		t.Run(p.name, func(t *testing.T) {
			t.Parallel()
			if raceEnabled && p.raceSeeds == 0 {
				t.Skip("plan 2-19 D595: the heaviest pack (5 000 households, about 14 s a seed, over 100 s under -race) is run by gate F only; -race gets the other three packs")
			}
			g := g01Facts(t, p, p.name)
			for _, v := range g.regimes["G01"] {
				t.Logf("pack %s: %s", p.name, v)
				if *stylisedRelease && v.Class != verdict.None {
					t.Errorf("-stylised.release: pack %s: %s, want none (an unset key or an unknown verdict is not a pass, STY 8.1.1)", p.name, v)
				}
			}
			g.report(t)
			packFacts(t, p).report(t)
		})
	}
}

// unskilledMean is the household-weighted mean welfare ratio of the unskilled roles
// at the close of the last year window (WelfareDistribution.Mean is the same
// weighted mean over the roles' rows; the snapshot rows carry it), in basis
// points; ok is false when no household of an unskilled role has a ratio.
func unskilledMean(r *stylisedRun) (mean int64, ok bool) {
	var sum, n int64
	for _, row := range lastYear(r).closing.Roles {
		if r.cfg.Roles[row.Role].Unskilled && row.HasMeanRatio {
			sum, n = sum+row.MeanRatioBP*row.RatioHouseholds, n+row.RatioHouseholds
		}
	}
	if n == 0 {
		return 0, false
	}
	return (2*sum + n) / (2 * n), true
}

// demandFacts judges STY-W01 on `one-demand` (the only demand pack REF 4.1 has) and
// reports STY-M04.
func demandFacts(t testing.TB) *factSet {
	seeds := packSeeds(t, packOneDemand)
	return memoFacts(seedKey("demand", seeds), func() *factSet {
		f, runs, bands := newFactSet(), loadRuns(t, packOneDemand, seeds), readBands(t, "one-demand")
		mean := make([]stat, len(runs))
		var gone []uint64
		for i, r := range runs {
			m, ok := unskilledMean(r)
			mean[i] = stat{Value: m}
			if !ok || r.collapsed() {
				mean[i] = stat{Why: "no household with a ratio at the close of the last year"}
				gone = append(gone, seeds[i])
			}
		}
		if len(gone) > 0 {
			f.notes = append(f.notes, livingNote("STY-W01", "one-demand", gone))
		}
		f.add("W01", judge(factDef{ID: "STY-W01", Statistic: "w01.mean_low_bp", Class: verdict.ClassB, Side: atLeast, Prior: 10_000, HasPrior: true}, "one-demand", seeds, mean, bands["w01.mean_low_bp"]))
		f.add("W01", judge(factDef{ID: "STY-W01", Statistic: "w01.mean_high_bp", Class: verdict.ClassB, Side: atMost, Prior: 30_000, HasPrior: true}, "one-demand", seeds, mean, bands["w01.mean_high_bp"]))
		f.notApplicable("M04", "STY-M04", "R", "aggregate ratio above 10 000 and at most the ceiling",
			"the aggregate ratio needs HouseholdIncome, which is not OK in version 1 (IND 9 row 21; FR-STY-046)")
		return f
	})
}

// TestStylised_Demand judges STY-W01 (the household-weighted mean ratio of the
// unskilled roles, within w01.mean_low_bp and w01.mean_high_bp) on `one-demand`
// over the seeds of the tier; STY-M04 is a report row (the aggregate ratio exists
// only with HouseholdIncome). `five` with the demand overlay is not a pack REF
// builds (STY 5.1 W01), so the second pack of the plan is not run.
//
// Requirement: FR-STY-029, FR-STY-046, FR-STY-047, FR-STY-082
func TestStylised_Demand(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("run-level test, skipped with -short")
	}
	f := demandFacts(t)
	for _, j := range f.scenarios["W01"] {
		t.Logf("%s %s: outside %v values %d..%d (median %d)", j.Row.ID, j.Row.Statistic, j.Row.Outside, j.Row.Min, j.Row.Max, j.Row.Median)
	}
	f.report(t)
}
