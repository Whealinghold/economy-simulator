//go:build stylised_long

package indicator_test

import (
	"fmt"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/internal/verdict"
)

// Tier L of the stylised facts (plan 2-19 Phase 8, D595, D596; STY 4.5.1): the
// facts of the population loop, W02, W03, W06 and the population leg of S02, which
// need a hundred years or more. The file is behind the build tag stylised_long and
// is never in the merge gate; the only user of a run longer than gateTicks.
//
//	go test -tags stylised_long -timeout 120m -run TestStylisedLong ./indicator/
//
// Seeds 1 to 10 (STY 8.2), 36 500 ticks, about 24 minutes a run of ten seeds at the
// placeholder 4 ms a tick (counted, not measured: N-015). W02 runs each seed twice,
// from above and from below.

// longTicks is a hundred years of 365 ticks (STY 4.5.1).
const longTicks = int64(36_500)

const (
	// The starts of W02 as shares of the configured households (STY W02): the
	// households at which the stock binds are observed, never assumed (RES 8.5), so
	// the configured count stands in for them. Decision taken, reversible here.
	startAbove = int64(16_000)
	startBelow = int64(6_000)

	// The shock of W03: a death scale of four times for a year at the middle of the
	// run (the model of POP 5.6 item 3).
	deathShockAt, deathShockFor, deathScaleBP = int64(18_250), int64(365), int64(40_000)

	workW02N, workW02Ratio = int64(500), int64(500) // w02.n_band_bp, w02.ratio_band_bp
)

var longLayers = []string{"overlay/resources-binding.toml", "overlay/population.toml"}

// longPack is pack `one-demand` with the binding resources and the population
// overlay, ten seeds.
var longPack = stylisedPack{"one-demand", packOneDemand.files, 10, 10}

// scaleHouseholds starts every group at start/10 000 of its configured households
// and scales the capacity and the starting stock of every endowment by the same
// share, so the fixed factor keeps the headroom its overlay sized and the world
// stays viable (the two runs of S37, scenario_pop_helpers_test.go).
func scaleHouseholds(start int64) func(*economysim.Config) {
	return func(c *economysim.Config) {
		c.Groups = slices.Clone(c.Groups)
		for i := range c.Groups {
			c.Groups[i].Households = max(c.Groups[i].Households*start/10_000, 1)
		}
		if c.Modules.Resources != nil {
			res := *c.Modules.Resources
			res.Endowments = slices.Clone(res.Endowments)
			for i := range res.Endowments {
				e := &res.Endowments[i]
				e.Capacity, e.Initial = e.Capacity*start/10_000, e.Initial*start/10_000
			}
			c.Modules.Resources = &res
		}
	}
}

// longRun runs the long pack from start (a share of the configured households, in
// basis points) for longTicks, with the death shock of W03 when shock is set.
func longRun(t testing.TB, seed uint64, start int64, shock bool) *stylisedRun {
	t.Helper()
	return mutatedRun(t, longPack, longLayers, fmt.Sprintf("long start=%d shock=%v", start, shock), seed, longTicks, scaleHouseholds(start), func(c *capture) {
		if !shock {
			return
		}
		c.before = func(w *economysim.World) {
			scales := func(bp int64) []economysim.RegionScale {
				out := make([]economysim.RegionScale, len(c.run.names.Regions))
				for i := range out {
					out[i] = economysim.RegionScale{Region: economysim.RegionID(i), BP: bp}
				}
				return out
			}
			switch w.Tick() {
			case deathShockAt:
				if err := w.SetDeathScales(scales(deathScaleBP)); err != nil {
					t.Fatalf("SetDeathScales(%d) error = %v, want nil", deathScaleBP, err)
				}
			case deathShockAt + deathShockFor:
				if err := w.SetDeathScales(scales(10_000)); err != nil {
					t.Fatalf("SetDeathScales(10000) error = %v, want nil", err)
				}
			}
		}
	})
}

// endLevel is the mean of the last year of a per-tick series.
func endLevel(s []int64) (int64, bool) {
	return meanOver(s, int64(len(s))-yearTicks, int64(len(s)))
}

// crossings counts the times a series crosses a level, ignoring the ticks on it.
func crossings(s []int64, level int64) int {
	n, side := 0, 0
	for _, v := range s {
		cur := 0
		switch {
		case v > level:
			cur = 1
		case v < level:
			cur = -1
		}
		if cur != 0 && side != 0 && cur != side {
			n++
		}
		if cur != 0 {
			side = cur
		}
	}
	return n
}

// replacementRatio is the ratio at which the birth and death schedules cross
// (POP 5.4), by linear interpolation between the knots, rounded down; ok is false
// when they do not cross.
func replacementRatio(birth, death []economysim.Knot) (int64, bool) {
	at := func(ks []economysim.Knot, r int64) int64 {
		if r <= ks[0].Ratio {
			return ks[0].Rate
		}
		for i := 1; i < len(ks); i++ {
			if r <= ks[i].Ratio {
				span := ks[i].Ratio - ks[i-1].Ratio
				return ks[i-1].Rate + (ks[i].Rate-ks[i-1].Rate)*(r-ks[i-1].Ratio)/span
			}
		}
		return ks[len(ks)-1].Rate
	}
	var prev int64
	for r := int64(0); r <= economysim.MaxBasisPoints/10; r += 10 {
		d := at(birth, r) - at(death, r)
		if r > 0 && (prev < 0) != (d < 0) || d == 0 && r > 0 {
			return r, true
		}
		prev = d
	}
	return 0, false
}

// warnedAny reports whether validation raised one of the rules.
func warnedAny(cfg economysim.Config, rules ...string) bool {
	return slices.ContainsFunc(cfg.Warnings(), func(w economysim.Warning) bool { return slices.Contains(rules, w.Rule) })
}

// TestStylisedLong_W02 verifies the approach to the fixed point (STY-W02): households
// started above (1.6) and below (0.6) end within w02.n_band_bp of each other, the
// mean ratio there is within w02.ratio_band_bp of r*, and each series crosses its end
// level at most once while validation raised neither population_gain nor
// resource_overshoot. B parts are asserted only where their key has a value.
//
// Requirement: FR-STY-030, FR-STY-083
func TestStylisedLong_W02(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("tier L run, skipped with -short")
	}
	bands := readBands(t, "one-demand")
	seeds := packSeeds(t, longPack)
	crossErr, gap, ratio := make([]string, len(seeds)), make([]stat, len(seeds)), make([]stat, len(seeds))
	f := newFactSet()
	for i, s := range seeds {
		up, down := longRun(t, s, startAbove, false), longRun(t, s, startBelow, false)
		eu, okU := endLevel(up.households)
		ed, okD := endLevel(down.households)
		gap[i], ratio[i] = stat{Why: "no end level"}, stat{Why: "no end level"}
		if !okU || !okD || eu+ed == 0 {
			continue
		}
		gap[i] = stat{Value: abs(eu-ed) * 20_000 / (eu + ed)}
		m := up.cfg.Modules.Population
		if rs, ok := replacementRatio(m.BirthSchedule, m.DeathSchedule); ok && rs > 0 {
			if r, ok := endLevel(up.ratio); ok {
				ratio[i] = stat{Value: abs(r-rs) * 10_000 / rs}
			}
		}
		if warnedAny(up.cfg, "population_gain", "resource_overshoot") {
			f.notes = append(f.notes, fmt.Sprintf("STY-W02 seed %d: validation raised population_gain or resource_overshoot; the crossing bound does not apply (POP 5.4)", s))
			continue
		}
		if c := crossings(up.households, eu); c > 1 {
			crossErr[i] = fmt.Sprintf("the series from above crosses its end level %d times", c)
		} else if c := crossings(down.households, ed); c > 1 {
			crossErr[i] = fmt.Sprintf("the series from below crosses its end level %d times", c)
		}
	}
	f.add("W02", judgeExact(factDef{ID: "STY-W02", Statistic: "at most one crossing of the end level", Class: verdict.ClassO}, "one-demand", seeds, crossErr))
	f.add("W02", judge(factDef{ID: "STY-W02", Statistic: "w02.n_band_bp", Class: verdict.ClassB, Prior: workW02N, HasPrior: true}, "one-demand", seeds, gap, bands["w02.n_band_bp"]))
	f.add("W02", judge(factDef{ID: "STY-W02", Statistic: "w02.ratio_band_bp", Class: verdict.ClassB, Prior: workW02Ratio, HasPrior: true}, "one-demand", seeds, ratio, bands["w02.ratio_band_bp"]))
	f.report(t)
}

// regrowth is the ticks from the trough of the households after the death shock
// to the first tick they are back within the recovery band of their level before it.
func regrowth(r *stylisedRun, bandBP int64) (troughAt, backAt int64, ok bool) {
	hh := r.households
	if int64(len(hh)) <= deathShockAt+deathShockFor || hh[deathShockAt-1] == 0 {
		return 0, 0, false
	}
	level := hh[deathShockAt-1]
	troughAt = deathShockAt
	for i := deathShockAt; i < int64(len(hh)); i++ {
		if hh[i] < hh[troughAt] {
			troughAt = i
		}
	}
	for i := troughAt; i < int64(len(hh)); i++ {
		if hh[i]*10_000 >= level*(10_000-bandBP) {
			return troughAt, i, true
		}
	}
	return troughAt, 0, false
}

// TestStylisedLong_W03 verifies that a shock that lowers the households raises the
// ratio of those who remain (STY-W03): from the trough to the tick the households
// are back within the recovery band, the sum over ticks of the product of the
// change of the households and the change of the mean ratio is negative (exact
// integers). A sum that is not negative means the loop is absent in the pack
// (no_fixed_factor): a finding about the pack, not a regime.
//
// Requirement: FR-STY-031, FR-STY-083
func TestStylisedLong_W03(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("tier L run, skipped with -short")
	}
	seeds := packSeeds(t, longPack)
	errs := make([]string, len(seeds))
	f := newFactSet()
	for i, s := range seeds {
		r := longRun(t, s, 10_000, true)
		from, to, ok := regrowth(r, workS14Recov)
		if !ok {
			f.notes = append(f.notes, fmt.Sprintf("STY-W03 pack one-demand seed %d: the households did not come back within the run; the sum is undetermined", s))
			continue
		}
		var sum int64
		for k := from + 1; k <= to; k++ {
			sum += (r.households[k] - r.households[k-1]) * (r.ratio[k] - r.ratio[k-1])
		}
		if sum >= 0 {
			errs[i] = fmt.Sprintf("the sum of dH * dR from tick %d to %d is %d, want negative (a finding about the pack: no_fixed_factor)", from, to, sum)
		}
	}
	f.add("W03", judgeExact(factDef{ID: "STY-W03", Statistic: "households and ratio move in opposite directions over the regrowth", Class: verdict.ClassO}, "one-demand", seeds, errs))
	f.report(t)
}

// TestStylisedLong_W06 verifies that natural increase is bounded by the schedules
// (STY-W06): in every year the households formed are at most the largest birth rate
// of the schedule in basis points of the largest households of the year, plus one
// household for the rounding of the carry of each region (X, every seed); and the
// signature of a closed region with an empty pool, BirthsUnfunded non-zero, is
// recorded with the households against M over the stake (B, w06.money_cap_bp, only
// where the key has a value).
//
// Requirement: FR-STY-034, FR-STY-083
func TestStylisedLong_W06(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("tier L run, skipped with -short")
	}
	bands := readBands(t, "one-demand")
	seeds := packSeeds(t, longPack)
	bound, cap := make([]string, len(seeds)), make([]stat, len(seeds))
	f := newFactSet()
	for i, s := range seeds {
		r := longRun(t, s, 10_000, false)
		m := r.cfg.Modules.Population
		var top int64
		for _, k := range m.BirthSchedule {
			top = max(top, k.Rate)
		}
		for y := int64(0); (y+1)*yearTicks <= int64(len(r.households)); y++ {
			var done, peak, unfunded int64
			for tick := y * yearTicks; tick < (y+1)*yearTicks; tick++ {
				done += at(r.obs.births, tick)
				unfunded += at(r.obs.unfunded, tick)
				peak = max(peak, r.households[tick])
			}
			if limit := peak*top/10_000 + int64(len(r.names.Regions)); done > limit {
				bound[i] = fmt.Sprintf("year %d formed %d households, above %d (the schedule's maximum rate %d bp of %d, plus the carry)", y+1, done, limit, top, peak)
				break
			}
			if y == int64(len(r.households))/yearTicks-1 && unfunded > 0 {
				stake := r.cfg.Modules.Population.BirthStake
				f.notes = append(f.notes, fmt.Sprintf("STY-W06 pack one-demand seed %d: BirthsUnfunded %d in the last year; households %d, money %d, birth stake %d bp of the role's start money", s, unfunded, r.final.Households, r.final.Money, stake))
			}
		}
		cap[i] = stat{Why: "no unfunded birth: the region is not stopped by its money"}
		if sum(r.obs.unfunded) > 0 {
			if stake := meanStartMoney(r.cfg) * m.BirthStake / 10_000; stake > 0 {
				limit := int64(r.final.Money) / stake
				cap[i] = stat{Value: max(0, (limit-r.final.Households)*10_000/max(limit, 1))}
			}
		}
	}
	f.add("W06", judgeExact(factDef{ID: "STY-W06", Statistic: "births in a year at most the schedule's maximum", Class: verdict.ClassX}, "one-demand", seeds, bound))
	f.add("W06", judge(factDef{ID: "STY-W06", Statistic: "w06.money_cap_bp", Class: verdict.ClassB, Prior: 1000, HasPrior: true}, "one-demand", seeds, cap, bands["w06.money_cap_bp"]))
	f.report(t)
}

// meanStartMoney is the start money of a role averaged over the households of the
// groups: the stake of a born household is a share of it (POP 4.2).
func meanStartMoney(cfg economysim.Config) int64 {
	start := map[string]int64{}
	for _, r := range cfg.Roles {
		start[r.ID] = int64(r.StartMoney)
	}
	var money, n int64
	for _, g := range cfg.Groups {
		money, n = money+start[g.Role]*g.Households, n+g.Households
	}
	if n == 0 {
		return 0
	}
	return money / n
}

func at(s []int64, i int64) int64 {
	if i < int64(len(s)) {
		return s[i]
	}
	return 0
}

func sum(s []int64) int64 {
	var t int64
	for _, v := range s {
		t += v
	}
	return t
}

// TestStylisedLong_S02_Population verifies the population leg of STY-S02: with the
// population module on, the recovery of `famine` after the half harvest of S14 is no
// faster than the speed limit at g_max = max(birth) - min(death) (POP 6.3), by
// verdict.RegrowTicks. The B bands are those of S02 and are asserted only where
// their key has a value.
//
// Requirement: FR-STY-036, FR-STY-083
func TestStylisedLong_S02_Population(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("tier L run, skipped with -short")
	}
	seeds := packSeeds(t, packFamine)
	speed := make([]string, len(seeds))
	f := newFactSet()
	for i, s := range seeds {
		r := scriptedRun(t, packFamine, []string{"overlay/population.toml"}, "s14-5000.toml", s, longTicks/5, false)
		m := r.cfg.Modules.Population
		var hiBirth, loDeath int64 = 0, 10_000
		for _, k := range m.BirthSchedule {
			hiBirth = max(hiBirth, k.Rate)
		}
		for _, k := range m.DeathSchedule {
			loDeath = min(loDeath, k.Rate)
		}
		st, ok := famineRead(r, workS14Recov)
		if !ok {
			f.notes = append(f.notes, fmt.Sprintf("STY-S02 pack famine+population seed %d: no household before the shock; undetermined", s))
			continue
		}
		tmin, reach := verdict.RegrowTicks(st.trough, st.level, workS14Recov, hiBirth-loDeath, r.cfg.Calendar.TicksPerYear())
		if st.recoverAt >= 0 && reach && st.recoverAt-st.troughAt < tmin {
			speed[i] = fmt.Sprintf("recovered in %d ticks, faster than the speed limit %d at g_max %d bp a year", st.recoverAt-st.troughAt, tmin, hiBirth-loDeath)
		}
	}
	f.add("S02 population", judgeExact(factDef{ID: "STY-S02", Statistic: "recovery no faster than the speed limit at g_max", Class: verdict.ClassO}, "famine", seeds, speed))
	f.report(t)
}

// TestStylisedLong_Measures verifies the readings of W02 on built data, without a
// run: the ratio at which the default schedules cross (12 000: birth 250 at every
// ratio, death falling from 450 at 8 000 to 50 at 16 000), the crossings of a level
// by a series and the end level.
//
// Requirement: FR-STY-030
func TestStylisedLong_Measures(t *testing.T) {
	t.Parallel()
	birth := []economysim.Knot{{Ratio: 8000, Rate: 250}, {Ratio: 16000, Rate: 250}}
	death := []economysim.Knot{{Ratio: 8000, Rate: 450}, {Ratio: 16000, Rate: 50}}
	if got, ok := replacementRatio(birth, death); !ok || got != 12000 {
		t.Errorf("replacementRatio(defaults) = %d, %v, want 12000, true", got, ok)
	}
	if _, ok := replacementRatio(birth, []economysim.Knot{{Ratio: 0, Rate: 10}}); ok {
		t.Error("replacementRatio(birth always above death) ok = true, want false")
	}
	if got := crossings([]int64{1, 2, 4, 6, 5, 4, 2}, 3); got != 2 {
		t.Errorf("crossings(up through 3 and down again) = %d, want 2", got)
	}
	if got := crossings([]int64{1, 2, 3, 3, 2, 1}, 3); got != 0 {
		t.Errorf("crossings(touching 3 and leaving below) = %d, want 0: a tick on the level is not a crossing", got)
	}
	s := slices.Repeat([]int64{0}, int(yearTicks))
	s = append(s, slices.Repeat([]int64{10}, int(yearTicks))...)
	if got, ok := endLevel(s); !ok || got != 10 {
		t.Errorf("endLevel(a year of 0 then a year of 10) = %d, %v, want 10, true", got, ok)
	}
}
