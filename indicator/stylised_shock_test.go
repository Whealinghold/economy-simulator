package indicator_test

import (
	"cmp"
	"fmt"
	"maps"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/internal/verdict"
)

// The fact tests of group B, the shocks (plan 2-19 Phase 7, D596): S01 and P05 on
// `one`, S02 and S06 on `famine` with the estates overlay. A shock is set between
// steps through the public calls of plan 2-17 and cleared by the same call; none
// edits state (STY 5.3). A pack that has collapsed (N-312, N-314) makes a fact
// that needs a living economy a finding in the log, never an edit to a test.

const (
	shockRunTicks = int64(700) // S01 and S08 run 700 ticks (REF 7.2)
	baseFrom      = int64(300) // P0 of S01 is the mean of ticks 300 to 399
	meanWindow    = int64(10)  // the recovery is read on a ten-tick mean (REF 7.2)

	// The priors that stand in for an unset key while a shock fact is measured
	// (labelled working values, STY 6.5.1): they never assert.
	workS01Band   = int64(1000) // s01.band_bp
	workS08Band   = int64(1000) // s08.band_bp
	workS14Recov  = int64(1000) // s14.recover_band_bp
	famineTicks   = int64(1800)
	famineShockAt = int64(424)
	famineShockTo = int64(516)
)

// marketSeries returns the price series of a commodity in a region, by tick.
func marketSeries(r *stylisedRun, region, commodity string) (price []int64, ok bool) {
	g, c := indexOf(r.names.Regions, region), indexOf(r.names.Commodities, commodity)
	if g < 0 || c < 0 {
		return nil, false
	}
	m := r.obs.markets[[2]int{g, c}]
	if m == nil {
		return nil, false
	}
	return m.Price, true
}

// outputOf returns the commodity a recipe makes, other than a discard.
func outputOf(cfg economysim.Config, recipe string) []string {
	var out []string
	for _, rc := range cfg.Recipes {
		if rc.ID != recipe {
			continue
		}
		for _, o := range rc.Outputs {
			if !o.Discard {
				out = append(out, o.Commodity)
			}
		}
	}
	return out
}

// chainDepths returns, for the commodities that the recipe's outputs feed
// through other recipes, their depth: 0 for the output itself, 1 for what is
// made from it, and so on. No identifier of a commodity or a recipe is named
// (architecture 9.1): the chain comes from the recipes.
func chainDepths(cfg economysim.Config, recipe string) map[string]int {
	depth := map[string]int{}
	for _, c := range outputOf(cfg, recipe) {
		depth[c] = 0
	}
	for changed := true; changed; {
		changed = false
		for _, rc := range cfg.Recipes {
			d, fed := -1, false
			for _, in := range rc.Inputs {
				if v, ok := depth[in.Commodity]; ok && (!fed || v < d) {
					d, fed = v, true
				}
			}
			if !fed {
				continue
			}
			for _, o := range rc.Outputs {
				if _, seen := depth[o.Commodity]; !o.Discard && !seen {
					depth[o.Commodity], changed = d+1, true
				}
			}
		}
	}
	return depth
}

// outsideBand reports whether v is farther than bandBP of ref.
func outsideBand(v, ref, bandBP int64) bool { return abs(v-ref)*10_000 > bandBP*ref }

// firstOutside returns the first tick in [from, to) at which the price is outside
// the band of p0, and whether it is above.
func firstOutside(s []int64, from, to, p0, bandBP int64) (tick int64, above, ok bool) {
	for t := from; t < min(to, int64(len(s))); t++ {
		if outsideBand(s[t], p0, bandBP) {
			return t, s[t] > p0, true
		}
	}
	return 0, false, false
}

// settledAt returns the first tick from from at which the price is inside the band
// of p0 and stays there to the end of the series; ok is false if it never is.
func settledAt(s []int64, from, p0, bandBP int64) (int64, bool) {
	at := int64(len(s))
	for t := int64(len(s)) - 1; t >= from; t-- {
		if outsideBand(s[t], p0, bandBP) {
			break
		}
		at = t
	}
	return at, at < int64(len(s))
}

// recoveredAt is the recovery tick of S01: the first tick from from at which the
// ten-tick mean ending there is inside the band of p0 and stays so to the end.
func recoveredAt(s []int64, from, p0, bandBP int64) (int64, bool) {
	at := int64(len(s))
	for t := int64(len(s)) - 1; t >= from; t-- {
		m, ok := meanOver(s, t-meanWindow+1, t+1)
		if !ok || outsideBand(m, p0, bandBP) {
			break
		}
		at = t
	}
	return at, at < int64(len(s))
}

// bandOrWork reads a band value, or its prior as the labelled working value while
// the key is unset.
func bandOrWork(bands map[string]verdict.BandKey, key string, work int64) int64 {
	if k := bands[key]; k.Set {
		return k.Value
	}
	return work
}

// collapsedSeeds lists the seeds of the runs whose households fell under a tenth.
func collapsedSeeds(seeds []uint64, runs []*stylisedRun) []uint64 {
	var out []uint64
	for i, r := range runs {
		if r.collapsed() {
			out = append(out, seeds[i])
		}
	}
	return out
}

// livingNote is the finding that a fact needing a living economy is
// undetermined on the collapsed seeds.
func livingNote(id, pack string, collapsed []uint64) string {
	return fmt.Sprintf("FINDING %s %s pack %s: the households fell under a tenth of the start on seeds %v; the shock facts on those seeds are findings, not passes", issueCollapse, id, pack, collapsed)
}

// shockRuns runs `one` with a script over the seeds of the tier.
func shockRuns(t testing.TB, seeds []uint64, script string) []*stylisedRun {
	t.Helper()
	runs := make([]*stylisedRun, len(seeds))
	for i, s := range seeds {
		runs[i] = scriptedRun(t, packOne, nil, script, s, shockRunTicks, false)
	}
	return runs
}

// shockFacts judges STY-S01 (both legs) and STY-P05 on `one`.
func shockFacts(t testing.TB) *factSet {
	seeds := packSeeds(t, packOne)
	return memoFacts(seedKey("shock", seeds), func() *factSet {
		f, bands := newFactSet(), readBands(t, "one")
		legA, legB, p05 := shockRuns(t, seeds, "s01.toml"), shockRuns(t, seeds, "s01b.toml"), shockRuns(t, seeds, "s08.toml")
		for _, runs := range [][]*stylisedRun{legA, legB, p05} {
			if c := collapsedSeeds(seeds, runs); len(c) > 0 {
				f.notes = append(f.notes, livingNote("STY-S01/P05", "one", c))
				break
			}
		}
		f.s01(t, seeds, bands, legA, "s01.toml", "leg (a) supply cut")
		f.s01(t, seeds, bands, legB, "s01b.toml", "leg (b) demand step")
		f.p05(t, seeds, bands, p05)
		return f
	})
}

// s01 judges one leg: O, the mean price during the shock is above P0, and B, the
// recovery time and the band in which it is read.
func (f *factSet) s01(t testing.TB, seeds []uint64, bands map[string]verdict.BandKey, runs []*stylisedRun, script, leg string) {
	band := bandOrWork(bands, "s01.band_bp", workS01Band)
	steps := readScript(t, script)
	sign := make([]string, len(runs))
	rec := make([]stat, len(runs))
	for i, r := range runs {
		from, to, region, commodity := s01Window(r, steps)
		s, ok := marketSeries(r, region, commodity)
		p0, okP := meanOver(s, baseFrom, from)
		during, okD := meanOver(s, from, to)
		switch {
		case !ok || !okP || !okD || r.collapsed():
			rec[i] = stat{Why: "no living market to read"}
		default:
			if during <= p0 {
				sign[i] = fmt.Sprintf("%s: mean price %d during the shock, P0 %d", leg, during, p0)
			}
			if at, ok := recoveredAt(s, to, p0, band); ok {
				rec[i] = stat{Value: at - to}
			} else {
				rec[i] = stat{Why: "no recovery within the run"}
			}
		}
	}
	f.add("S01", judgeExact(factDef{ID: "STY-S01", Statistic: "price above P0 during the shock: " + leg, Class: verdict.ClassO}, "one", seeds, sign))
	f.add("S01", judge(factDef{ID: "STY-S01", Statistic: "s01.recover_ticks: " + leg, Class: verdict.ClassB, Prior: 90, HasPrior: true}, "one", seeds, rec, bands["s01.recover_ticks"]))
}

// s01Window returns the shock's ticks, region and the commodity watched: leg (a)
// is the output of the shocked recipe, leg (b) the commodity the visitor buys.
func s01Window(r *stylisedRun, steps []scriptStep) (from, to int64, region, commodity string) {
	for _, s := range steps {
		if s.Buyer {
			return s.Tick, s.Tick + s.Ticks, s.Region, s.Commodity
		}
	}
	from, to, region, recipe := shockSpan(steps)
	if out := outputOf(r.cfg, recipe); len(out) > 0 {
		commodity = out[0]
	}
	return from, to, region, commodity
}

// p05 judges the cost shock of S08: O, the input is above its band during the
// shock and the first excursions along the chain are in order; B, the ticks the
// slowest price takes to settle after the shock.
func (f *factSet) p05(t testing.TB, seeds []uint64, bands map[string]verdict.BandKey, runs []*stylisedRun) {
	band := bandOrWork(bands, "s08.band_bp", workS08Band)
	order, settle := make([]string, len(runs)), make([]stat, len(runs))
	from, to, region, recipe := shockSpan(readScript(t, "s08.toml"))
	for i, r := range runs {
		depths := chainDepths(r.cfg, recipe)
		settle[i] = stat{Why: "no living market to read"}
		if r.collapsed() {
			continue
		}
		first := map[string]int64{}
		var worst int64
		settle[i] = stat{}
		for _, c := range slices.Sorted(maps.Keys(depths)) {
			s, ok := marketSeries(r, region, c)
			p0, okP := meanOver(s, from-100, from)
			if !ok || !okP {
				continue
			}
			if tick, above, out := firstOutside(s, from, to, p0, band); out {
				first[c] = tick
				if depths[c] == 0 && !above {
					order[i] = fmt.Sprintf("input %s leaves its band downward at tick %d under a supply cut", c, tick)
				}
			} else if depths[c] == 0 && order[i] == "" {
				order[i] = fmt.Sprintf("input %s never leaves its %d bp band during the shock", c, band)
			}
			if at, ok := settledAt(s, to, p0, band); ok {
				worst = max(worst, at-to)
			} else {
				settle[i] = stat{Why: "a price in the chain does not settle within the run"}
			}
		}
		if order[i] == "" {
			order[i] = excursionOrder(first, depths)
		}
		if settle[i].Why == "" {
			settle[i] = stat{Value: worst}
		}
	}
	f.add("P05", judgeExact(factDef{ID: "STY-P05", Statistic: "input above its band and excursions in order along the chain", Class: verdict.ClassO}, "one", seeds, order))
	f.add("P05", judge(factDef{ID: "STY-P05", Statistic: "s08.settle_ticks", Class: verdict.ClassB, Prior: 90, HasPrior: true}, "one", seeds, settle, bands["s08.settle_ticks"]))
}

// excursionOrder is the error of an excursion that comes before one nearer the
// input: a product meets the shock only through the input's market and the cost
// anchor, so it cannot leave its band first.
func excursionOrder(first map[string]int64, depths map[string]int) string {
	for a, ta := range first {
		for b, tb := range first {
			if depths[a] < depths[b] && tb < ta {
				return fmt.Sprintf("%s (depth %d) leaves its band at tick %d, before %s (depth %d) at tick %d", b, depths[b], tb, a, depths[a], ta)
			}
		}
	}
	return ""
}

// TestStylised_One_Shocks runs the shocks of `one` for 700 ticks on the seeds of
// the tier: STY-S01 with the supply cut (leg a) and the visitor's demand step (leg
// b), and STY-P05 with the cost shock of S08. The O parts are asserted on every
// seed; the recovery and settling times are B, asserted only where their key has a
// value and else reported beside the prior. The shocks are set through
// SetYieldShocks and AddAgent with Direct, between steps.
//
// Requirement: FR-STY-024, FR-STY-035, FR-STY-082, FR-REF-026, FR-REF-027
func TestStylised_One_Shocks(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("run-level test, skipped with -short")
	}
	f := shockFacts(t)
	for sc, js := range f.scenarios {
		for _, j := range js {
			t.Logf("%s %s: %s outside %v values %d..%d (median %d)", sc, j.Row.ID, j.Row.Statistic, j.Row.Outside, j.Row.Min, j.Row.Max, j.Row.Median)
		}
	}
	f.report(t)
}

// ---- the ladder watch (STY-S06) ------------------------------------------------

// ladderWatch polls the agents of the regions that had an unfed household in a
// tick, through Agent(a) of plan 2-17, and counts each agent's unfed ticks. A
// removal for hunger must come at least RemovePrivation unfed ticks after the
// first, since privation rises by 1 a tick when unfed and falls by 1 otherwise
// (SPN 5.1). Polling only the regions with an UpkeepShortfall keeps a healthy
// run free of the poll (a counted cost of 600 queries a tick, STY 5.3, becomes
// the unfed agents only).
type ladderWatch struct {
	removeAt int64
	unfedIn  map[economysim.RegionID]bool
	removed  []removal
	count    map[economysim.AgentID]int64
	errs     []string
	regions  int
}

func newLadderWatch(cfg economysim.Config) *ladderWatch {
	return &ladderWatch{removeAt: cfg.World.RemovePrivation, unfedIn: map[economysim.RegionID]bool{},
		count: map[economysim.AgentID]int64{}, regions: len(cfg.Regions)}
}

// poll is the capture's after hook: it judges the tick's hunger removals against
// the counts polled so far (the removed agent cannot be polled at its removal
// tick, so its own unfed tick is the +1), then counts the unfed ticks of this tick.
func (l *ladderWatch) poll(w *economysim.World) {
	for _, r := range l.removed {
		if (r.Cause == economysim.CauseStarved || r.Cause == economysim.CauseInsolvent) && l.count[r.Agent]+1 < l.removeAt {
			l.errs = append(l.errs, fmt.Sprintf("agent %d removed at tick %d for %s after at most %d unfed ticks, want at least RemovePrivation %d",
				r.Agent, r.Tick, r.Cause, l.count[r.Agent]+1, l.removeAt))
		}
	}
	l.removed = l.removed[:0]
	for g := range l.regions {
		if !l.unfedIn[economysim.RegionID(g)] {
			continue
		}
		ids, err := w.Agents(economysim.RegionID(g))
		if err != nil {
			l.errs = append(l.errs, fmt.Sprintf("Agents(%d) at tick %d: %v", g, w.Tick(), err))
			continue
		}
		for _, id := range ids {
			if st, err := w.Agent(id); err == nil && st.Unfed > 0 {
				l.count[id]++
			}
		}
	}
	clear(l.unfedIn)
}

// ladderOrder is the other X part of S06: relief is paid before the first
// removal for hunger (rung 2 before rung 3). It reads nothing when no relief was
// paid or no agent was removed for hunger.
func ladderOrder(o *runObs) string {
	first := int64(-1)
	for t, v := range o.relief {
		if v > 0 {
			first = int64(t)
			break
		}
	}
	if first < 0 {
		return ""
	}
	for _, r := range o.removed {
		if (r.Cause == economysim.CauseStarved || r.Cause == economysim.CauseInsolvent) && r.Tick < first {
			return fmt.Sprintf("agent %d removed for %s at tick %d, before the first relief at tick %d", r.Agent, r.Cause, r.Tick, first)
		}
	}
	return ""
}

// ---- STY-S02 and STY-S06 on famine ----------------------------------------------

// famineDepths are the depths m of S14, in basis points of the staple yield that
// remains; the smallest is the deep depth that expects R3 (STY 6.4).
var famineDepths = []int64{7000, 5000, 2500, 0}

// replacementRate is the fastest regrowth the modules in force allow without the
// population module, in basis points a year for verdict.RegrowTicks: replacement
// of at most Renewal of the households lost in a tick (SPN 7.1), which the caps
// of SPN 6.3 do not limit (FR-SPN-022). The funding limit (EntryPoolShare of the
// pool a tick) is not applied, so the bound is the loosest sound one: an observed
// recovery faster than it is an accounting defect. Decision taken: reverse by
// passing a tighter rate (D585, STY open question 2).
func replacementRate(cfg economysim.Config) int64 {
	return cfg.World.Renewal * cfg.Calendar.TicksPerYear()
}

// famineStats is what S02 reads from the households of one run.
type famineStats struct {
	level, trough, troughAt int64
	recoverAt               int64 // -1 when the households did not come back
}

func famineRead(r *stylisedRun, bandBP int64) (famineStats, bool) {
	hh := r.households
	if int64(len(hh)) <= famineShockAt || hh[famineShockAt-1] == 0 {
		return famineStats{}, false
	}
	st := famineStats{level: hh[famineShockAt-1], trough: hh[famineShockAt-1], troughAt: famineShockAt - 1, recoverAt: -1}
	for i := famineShockAt - 1; i < int64(len(hh)); i++ {
		if hh[i] < st.trough {
			st.trough, st.troughAt = hh[i], i
		}
	}
	for i := st.troughAt; i < int64(len(hh)); i++ {
		if hh[i]*10_000 >= st.level*(10_000-bandBP) {
			st.recoverAt = i
			break
		}
	}
	return st, true
}

// famineFacts judges S02 (O speed limit, B fall and recovery, the expected
// regime of the deep depth, X audit) and S06 (X ladder order) on every depth.
func famineFacts(t testing.TB) *factSet {
	seeds := packSeeds(t, packFamine)
	return memoFacts(seedKey("famine", seeds), func() *factSet {
		f, bands, th := newFactSet(), readBands(t, "famine"), readThresholds(t)
		band := bandOrWork(bands, "s14.recover_band_bp", workS14Recov)
		deep := slices.Min(famineDepths)
		for _, m := range famineDepths {
			sc := fmt.Sprintf("S02 depth %d", m)
			runs := make([]*stylisedRun, len(seeds))
			for i, s := range seeds {
				runs[i] = scriptedRun(t, packFamine, nil, fmt.Sprintf("s14-%d.toml", m), s, famineTicks, true)
			}
			audit, speed, ladder, expect := make([]string, len(runs)), make([]string, len(runs)), make([]string, len(runs)), make([]string, len(runs))
			fall, rec := make([]stat, len(runs)), make([]stat, len(runs))
			var gone []uint64
			var example string
			for i, r := range runs {
				audit[i] = errText(r.stepErr)
				if errs := r.obs.watch.errs; len(errs) > 0 {
					ladder[i] = errs[0]
				} else {
					ladder[i] = ladderOrder(r.obs)
				}
				fall[i], rec[i] = stat{Why: "no household before the shock"}, stat{Why: "no household before the shock"}
				st, ok := famineRead(r, band)
				if !ok {
					// The pack collapsed before the shock (N-312, N-314): what the
					// ladder did is a finding, not a verdict on the ladder.
					if ladder[i] != "" {
						gone, example = append(gone, seeds[i]), cmp.Or(example, ladder[i])
						ladder[i] = ""
					}
					continue
				}
				fall[i] = stat{Value: 10_000 - st.trough*10_000/st.level}
				rec[i] = stat{Why: "the households did not come back: a collapse"}
				tmin, reach := verdict.RegrowTicks(st.trough, st.level, band, replacementRate(r.cfg), r.cfg.Calendar.TicksPerYear())
				if st.recoverAt >= 0 {
					rec[i] = stat{Value: st.recoverAt - st.troughAt}
					if reach && st.recoverAt-st.troughAt < tmin {
						speed[i] = fmt.Sprintf("recovered in %d ticks, faster than the speed limit %d", st.recoverAt-st.troughAt, tmin)
					}
				}
				if m == deep {
					expect[i] = expectedRegime(r, th, verdict.R3, f, seeds[i])
				}
			}
			if len(gone) > 0 {
				f.notes = append(f.notes, fmt.Sprintf("FINDING %s STY-S06 pack famine depth %d seeds %v: the households were gone before tick %d, so the ladder's order is a finding, not a verdict; for example %s", issueCollapse, m, gone, famineShockAt, example))
			}
			f.add(sc, judgeExact(factDef{ID: "STY-S02", Statistic: "audit holds (goods, money, pool, balances)", Class: verdict.ClassX}, "famine", seeds, audit))
			f.add(sc, judgeExact(factDef{ID: "STY-S02", Statistic: "recovery no faster than the speed limit", Class: verdict.ClassO}, "famine", seeds, speed))
			f.add(sc, judgeExact(factDef{ID: "STY-S06", Statistic: "hunger removals after RemovePrivation unfed ticks, relief before removal", Class: verdict.ClassX}, "famine", seeds, ladder))
			if m == deep {
				f.add(sc, judgeExact(factDef{ID: "STY-S02", Statistic: "expected regime R3 occurs", Class: verdict.ClassO}, "famine", seeds, expect))
				f.notApplicable(sc, "STY-S02", "B", "s14.recover_ticks", "the deep depth expects R3; the recovery bands are read on the depths that recover")
				continue
			}
			f.add(sc, judge(factDef{ID: "STY-S02", Statistic: "s14.fall_bp", Class: verdict.ClassB, Prior: 5000, HasPrior: true}, "famine", seeds, fall, bands["s14.fall_bp"]))
			f.add(sc, judge(factDef{ID: "STY-S02", Statistic: "s14.recover_ticks", Class: verdict.ClassB}, "famine", seeds, rec, bands["s14.recover_ticks"]))
		}
		return f
	})
}

func errText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// expectedRegime is the check of FR-STY-073 for a scenario that expects a regime:
// with the thresholds set, a verdict other than detected is an error (the shock
// was not felt); while a threshold is unset the verdict is unknown and the
// working thresholds (the priors, labelled) say what it would be.
func expectedRegime(r *stylisedRun, th verdict.Thresholds, d verdict.Detector, f *factSet, seed uint64) string {
	in, err := detectorInput(r)
	if err != nil {
		return "detector input: " + err.Error()
	}
	in.Rate = replacementRate(r.cfg)
	switch v := verdict.Detect(d, in, th); v.Class {
	case verdict.Detected:
		return ""
	case verdict.Unknown:
		w := verdict.Detect(d, in, workingThresholds())
		f.notes = append(f.notes, fmt.Sprintf("STY-S02 pack famine seed %d: expected %s is %s; with the prior as a working threshold: %s (report only)", seed, d, v, w))
		return ""
	default:
		return fmt.Sprintf("expected regime %s did not occur: %s", d, v)
	}
}

// TestStylised_Famine_Depths runs `famine` with the estates overlay for 1 800
// ticks at the four depths of S14 (the harvest of `grow_staple` from tick 424 to
// 515): STY-S02 (the speed limit by verdict.RegrowTicks and the replacement rate,
// the fall and the recovery, the expected regime R3 at the deepest depth) and
// STY-S06 (a hunger removal after RemovePrivation unfed ticks, polled through
// Agent(a), and relief before removal). X and O parts are asserted on every seed;
// B parts only where their key has a value.
//
// Requirement: FR-STY-036, FR-STY-040, FR-STY-073, FR-STY-082, FR-SPN-013, FR-SPN-014, FR-SPN-015, FR-REF-026
func TestStylised_Famine_Depths(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("run-level test, skipped with -short")
	}
	famineFacts(t).report(t)
}
