package indicator_test

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/internal/verdict"
)

// The provoked runs of the separation protocol (STY 6.5.3, plan 2-19 D598) and the
// proposal of the first setting run. A provoked run causes a regime on purpose; its
// statistic is recorded and never asserted: a regime is a property of a run, a
// failed test a property of an assertion (STY 6.4). Nothing here writes a value
// line into a band file: a human reads the proposal (STY 8.4.5).

// sepKey is the separated key of a detector, whether its bad side is low, and the
// kind of its value (STY 6.5.3).
type sepKey struct {
	name string
	low  bool
	kind verdict.Kind
}

var separated = map[verdict.Detector]sepKey{
	verdict.R1: {"r1.rise_bp", false, verdict.Share}, verdict.R2: {"r2.share_bp", false, verdict.Share},
	verdict.R3: {"r3.fall_bp", false, verdict.Share}, verdict.R4: {"r4.vol_bp", true, verdict.Share},
	verdict.R5a: {"r5.pinned_ticks", false, verdict.Ticks}, verdict.R5b: {"r5.freeze_ticks", false, verdict.Ticks},
	verdict.R6: {"r6.factor", false, verdict.Share}, verdict.R7: {"r7.disp_bp", true, verdict.Share},
	verdict.R8: {"r8.offset_bp", false, verdict.Share}, verdict.R9: {"r9.fall_bp", false, verdict.Share},
}

// maxKey bounds the search for a statistic: no key of 6.2 reaches it.
const maxKey = int64(1) << 40

// statisticOf returns the value the series of a run reaches for the detector to
// count it as the regime (6.5.3): the largest value of the separated key at which
// the detector still fires, the held keys at their priors; for a detector whose
// bad side is low the smallest value at which it fires, less one (Separate
// negates it). ok is false when the detector
// cannot fire on the run at all (a guard makes it unknown, the series is empty)
// or fires at any key.
func statisticOf(in verdict.Input, d verdict.Detector) (stat int64, ok bool) {
	k := separated[d]
	fires := func(v int64) bool {
		th := workingThresholds()
		th.Set(k.name, v)
		return verdict.Detect(d, in, th).Class == verdict.Detected
	}
	if k.low {
		if fires(0) || !fires(maxKey) {
			return 0, false
		}
		lo, hi := int64(0), maxKey // fires(lo) is false, fires(hi) is true
		for hi-lo > 1 {
			if mid := lo + (hi-lo)/2; fires(mid) {
				hi = mid
			} else {
				lo = mid
			}
		}
		return hi - 1, true
	}
	if !fires(0) || fires(maxKey) {
		return 0, false
	}
	lo, hi := int64(0), maxKey // fires(lo) is true, fires(hi) is false
	for hi-lo > 1 {
		if mid := lo + (hi-lo)/2; fires(mid) {
			lo = mid
		} else {
			hi = mid
		}
	}
	return lo, true
}

// provokedRun is one run that causes a regime (the table of 6.5.3).
type provokedRun struct {
	Detector verdict.Detector
	Label    string
	Ticks    int64
	Run      func(t testing.TB, seed uint64) *stylisedRun
}

// scaleUp is the large SetFlowScales of R1's provoked run, in basis points: five
// times the configured grant, from tick 365.
const scaleUp = int64(50_000)

func provokedRuns() []provokedRun {
	oneRun := func(tag string, layers []string, ticks int64, mutate func(*economysim.Config), setup func(testing.TB) func(*capture)) func(testing.TB, uint64) *stylisedRun {
		return func(t testing.TB, seed uint64) *stylisedRun {
			var su func(*capture)
			if setup != nil {
				su = setup(t)
			}
			return mutatedRun(t, packOne, layers, tag, seed, ticks, mutate, su)
		}
	}
	costFloor := func(v int64) func(*economysim.Config) {
		return func(c *economysim.Config) { c.World.CostFloor = v }
	}
	return []provokedRun{
		{verdict.R1, "S13 faucet scaled up", 1095, oneRun("r1", []string{"overlay/faucet.toml"}, 1095, nil, func(t testing.TB) func(*capture) {
			return func(c *capture) {
				var flows []economysim.FlowScale
				for _, fl := range c.run.cfg.Modules.MoneySupply.Flows {
					flows = append(flows, economysim.FlowScale{Flow: fl.ID, BP: scaleUp})
				}
				c.before = func(w *economysim.World) {
					if w.Tick() == faucetOffAt/2 {
						if err := w.SetFlowScales(flows); err != nil {
							t.Fatalf("SetFlowScales(%d) error = %v, want nil", scaleUp, err)
						}
					}
				}
			}
		})},
		{verdict.R2, "S19 halt off", 1095, oneRun("r2", nil, 1095, func(c *economysim.Config) { c.World.ProduceHaltRatio = economysim.MaxBasisPoints }, nil)},
		{verdict.R3, "S14 famine depth 0", famineTicks, func(t testing.TB, seed uint64) *stylisedRun {
			return scriptedRun(t, packFamine, nil, "s14-0.toml", seed, famineTicks, true)
		}},
		{verdict.R4, "S25 money at half the warm start", 1095, func(t testing.TB, seed uint64) *stylisedRun {
			return mutatedRun(t, packOneDemand, nil, "r4", seed, 1095, func(c *economysim.Config) {
				if c.WarmStart != nil {
					for i := range c.WarmStart.Groups {
						c.WarmStart.Groups[i].Money /= 2
					}
				}
			}, nil)
		}},
		{verdict.R5a, "S16 overhang", 730, oneRun("r5a", []string{"overlay/overhang.toml"}, 730, nil, nil)},
		{verdict.R5b, "S07 cost floor 10000", 400, oneRun("r5b-10000", []string{"overlay/glut-prepared.toml"}, 400, costFloor(10_000), nil)},
		{verdict.R5b, "S07 cost floor 9000", 400, oneRun("r5b-9000", []string{"overlay/glut-prepared.toml"}, 400, costFloor(9_000), nil)},
		{verdict.R6, "S05 visitor on long-grow", 1095, oneRun("r6", []string{"overlay/long-grow.toml"}, 1095, nil, func(t testing.TB) func(*capture) {
			return func(c *capture) {
				d := &scriptDriver{t: t, steps: readScript(t, "s01b.toml"), cfg: c.run.cfg, names: c.run.names, obs: c.run.obs}
				c.before = d.apply
			}
		})},
		{verdict.R7, "S17 cold one", yearTicks, func(t testing.TB, seed uint64) *stylisedRun {
			return closedRun(t, stylisedPack{"one-cold", []string{"base.toml", "one.toml"}, 10, 2}, seed, yearTicks)
		}},
		{verdict.R8, "S22 perturbed chain, warm table", 1095, oneRun("r8", []string{"overlay/perturbed.toml"}, 1095, nil, nil)},
		{verdict.R9, "overlay/sink.toml", 1095, oneRun("r9", []string{"overlay/sink.toml"}, 1095, nil, nil)},
	}
}

// detectorStatistic reads the statistic of a detector on a run, or why there is none.
func detectorStatistic(r *stylisedRun, d verdict.Detector) (int64, string) {
	in, err := detectorInput(r)
	if err != nil {
		return 0, "detector input: " + err.Error()
	}
	if d == verdict.R3 {
		in.Rate = replacementRate(r.cfg)
	}
	if v, ok := statisticOf(in, d); ok {
		return v, ""
	}
	return 0, fmt.Sprintf("%s cannot be read on this run: %s with the priors", d, verdict.Detect(d, in, workingThresholds()))
}

// TestStylised_Provoked_Runs runs each provoked run of 6.5.3 on the seeds of the
// tier and logs the statistic of its detector next to the run's label. It asserts
// nothing: the statistics are data for the separation (D598), tier N.
//
// Requirement: FR-STY-071, FR-STY-073, FR-STY-082
func TestStylised_Provoked_Runs(t *testing.T) {
	t.Parallel()
	requireTierN(t)
	seeds := packSeeds(t, packOne)
	for _, pr := range provokedRuns() {
		t.Run(pr.Label, func(t *testing.T) {
			t.Parallel()
			for _, s := range seeds {
				r := pr.Run(t, s)
				v, why := detectorStatistic(r, pr.Detector)
				if why != "" {
					t.Logf("provoked %s (%s) seed %d: no statistic: %s", pr.Detector, pr.Label, s, why)
					continue
				}
				t.Logf("provoked %s (%s) seed %d: statistic %d (collapsed %v)", pr.Detector, pr.Label, s, v, r.collapsed())
			}
		})
	}
}

// extreme is the worst value of the healthy runs (t_hi) or the least bad of the
// provoked (t_lo): the largest when the bad side is high, else the smallest.
func extreme(v []int64, smallest bool) int64 {
	if len(v) == 0 {
		return 0
	}
	if smallest {
		return slices.Min(v)
	}
	return slices.Max(v)
}

// healthyRuns are the warm, no-shock runs of the shipped packs (the set H).
func healthyRuns(t testing.TB) []*stylisedRun {
	t.Helper()
	var out []*stylisedRun
	for _, p := range []stylisedPack{packOne, packSecond, packTwo, packThree} {
		out = append(out, loadRuns(t, p, packSeeds(t, p))...)
	}
	return out
}

// proposalOf runs the separation protocol over the healthy and provoked runs and
// returns the text of the proposal: for each detector t_hi, t_lo and the key
// Separate proposes, or the refusal and why. Nothing here sets a value.
func proposalOf(t testing.TB, seeds []uint64) []string {
	t.Helper()
	hi := map[verdict.Detector][]int64{}
	lo := map[verdict.Detector][]int64{}
	var lines []string
	for _, r := range healthyRuns(t) {
		for _, d := range verdict.Detectors() {
			if v, why := detectorStatistic(r, d); why == "" {
				hi[d] = append(hi[d], v)
			}
		}
	}
	for _, pr := range provokedRuns() {
		for _, s := range seeds {
			if v, why := detectorStatistic(pr.Run(t, s), pr.Detector); why == "" {
				lo[pr.Detector] = append(lo[pr.Detector], v)
			}
		}
	}
	for _, d := range verdict.Detectors() {
		k := separated[d]
		got, err := verdict.Separate(hi[d], lo[d], k.kind, k.low)
		tHi, tLo := extreme(hi[d], k.low), extreme(lo[d], !k.low)
		switch {
		case errors.Is(err, verdict.ErrNoRuns):
			lines = append(lines, fmt.Sprintf("%s %s: no proposal, healthy runs %d and provoked runs %d with a statistic (no threshold is set from one set alone); the key stays unset", d, k.name, len(hi[d]), len(lo[d])))
		case err != nil:
			lines = append(lines, fmt.Sprintf("%s %s: does not separate (t_hi %d, t_lo %d): %v; the key stays unset and G01 reports %s unknown", d, k.name, tHi, tLo, err, d))
		default:
			lines = append(lines, fmt.Sprintf("%s %s: propose %d (t_hi %d, t_lo %d, %d healthy and %d provoked runs); a human reads it and sets the value in a commit of their own", d, k.name, got, tHi, tLo, len(hi[d]), len(lo[d])))
		}
	}
	return lines
}

// bandProposals applies the band rule of STY 8.4.3 (verdict.SetBand) to every
// banded fact of the fact sets that has a number for every setting seed, in the
// order of the rows. A band is refused, with the reason, when a seed has no number
// (a failed seed: 8.4.1), when fewer than ten seeds ran, or when a threshold it
// backs is unset (the thresholds come first: 10.3). A refusal is data, not an
// error; nothing here sets a value.
func bandProposals(t testing.TB, sets ...*factSet) []string {
	t.Helper()
	th := readThresholds(t)
	var out []string
	for _, f := range sets {
		for _, sc := range slices.Sorted(maps.Keys(f.scenarios)) {
			for _, j := range f.scenarios[sc] {
				if j.Row.Class != verdict.ClassB || j.Seeds == 0 || strings.ContainsAny(j.Row.Statistic, " :") {
					continue
				}
				out = append(out, bandProposal(th, j))
			}
		}
	}
	return out
}

// bandProposal is the line for one band key.
func bandProposal(th verdict.Thresholds, j judged) string {
	key := j.Row.Statistic
	head := fmt.Sprintf("band %s (%s, %s): ", key, j.Row.ID, map[side]string{atMost: "upper", atLeast: "lower"}[j.Side])
	if len(j.Values) != j.Seeds {
		return head + fmt.Sprintf("no band: %d of %d seeds have no number (a failed seed, STY 8.4.1)", j.Seeds-len(j.Values), j.Seeds)
	}
	var limit verdict.Limit
	for _, pr := range verdict.OrderingPairs() {
		if pr.Band != key {
			continue
		}
		v, ok := th.Get(pr.Threshold)
		if !ok {
			return head + fmt.Sprintf("no band: the threshold %s it backs is unset; thresholds are set first (STY 10.3)", pr.Threshold)
		}
		switch pr.Rel {
		case verdict.AtMost:
			limit = verdict.Limit{Set: true, Value: v}
		case verdict.AtLeast:
			limit = verdict.Limit{Set: true, Value: v}
		case verdict.AtLeastComplement:
			limit = verdict.Limit{Set: true, Value: 10_000 - v}
		}
	}
	sd := verdict.Upper
	if j.Side == atLeast {
		sd = verdict.Lower
	}
	v, err := verdict.SetBand(j.Values, sd, j.Seeds, limit)
	switch {
	case errors.Is(err, verdict.ErrAboveThreshold):
		return head + fmt.Sprintf("refused: the bound %d is looser than the threshold %d it backs", v, limit.Value)
	case err != nil:
		return head + "refused: " + err.Error()
	}
	return head + fmt.Sprintf("propose %d (min %d, median %d, max %d over %d seeds); a human reads it and sets the value in a commit of their own", v, j.Row.Min, j.Row.Median, j.Row.Max, j.Seeds)
}

// TestStylised_Propose renders the proposal of the first setting run (STY 8.4,
// plan 2-19 D598) with -stylised.propose: the separation outcome of every detector
// over the healthy and the provoked runs of the setting seeds, and the verdict
// record of the closed packs. It never writes a value line into a band file; a
// human reads the text, and the commit that adds value and set_by lines is the
// reviewer's own, with the three-way decision (band, code or fact wrong) in the pull
// request (testing.md 7.2). It is skipped without the flag.
//
// Requirement: FR-STY-071, FR-STY-087, FR-STY-091
func TestStylised_Propose(t *testing.T) {
	if testing.Short() {
		t.Skip("run-level test, skipped with -short")
	}
	if !*stylisedPropose {
		t.Skip("a proposal is made only with -stylised.propose (plan 2-19 D598)")
	}
	seeds := packSeeds(t, packOne)
	t.Logf("proposal for the setting run: commit and configuration hash are the reviewer's to name; seeds %d..%d", seeds[0], seeds[len(seeds)-1])
	for _, l := range proposalOf(t, seeds) {
		t.Log(l)
	}
	for _, l := range bandProposals(t, oneFacts(t), warmFacts(t), shockFacts(t)) {
		t.Log(l)
	}
	t.Log(recordOf(t, packOne, oneFacts(t), g01Facts(t, packOne, "one"), warmFacts(t), shockFacts(t)))
}
