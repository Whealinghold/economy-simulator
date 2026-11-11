package economysim_test

import (
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/ordering"
)

// TestScenarioS73_BindingCapFairness is the unit model of WLD-S1 (spec-world.md 17): 100
// tied candidates, a cap of 5, 1 000 ticks, the tie broken by the keyed order K(t, id).
// Pass when no candidate is admitted fewer than half or more than twice its expected
// share (50). The places O1 to O5, O9 and O10 of 10.2 on a built world are not run here:
// the device is the same ordering.Key at each of them.
//
// Requirement: FR-WLD-090, FR-WLD-091
//
// Guard: a tie broken by the lower ID admits ids 0 to 4 every tick and nobody else.
func TestScenarioS73_BindingCapFairness(t *testing.T) {
	t.Parallel()
	const n, capN, ticks = 100, 5, 1000
	got := make([]int, n)
	ids := make([]int64, n)
	for tick := int64(0); tick < ticks; tick++ {
		for i := range ids {
			ids[i] = int64(i)
		}
		slices.SortFunc(ids, func(a, b int64) int {
			ka, kb := ordering.Key(tick, a), ordering.Key(tick, b)
			switch {
			case ka < kb:
				return -1
			case ka > kb:
				return 1
			}
			return int(a - b)
		})
		for _, id := range ids[:capN] {
			got[id]++
		}
	}
	want := ticks * capN / n
	lo, hi := slices.Min(got), slices.Max(got)
	t.Logf("S73 admissions per candidate: min %d max %d, expected %d", lo, hi, want)
	if lo < want/2 || hi > want*2 {
		t.Errorf("admissions min %d max %d, want within [%d, %d]", lo, hi, want/2, want*2)
	}
}

// TestScenarioS74_CadenceLoad is the unit model of WLD-S2: agents at periods 10 and 30 with
// IDs spaced by every stride of 1 to 31 from three starting IDs; no tick has more than
// ceil(1.2 n / P) reviews under the rule bucket of 9.1.1, for n of 1 000 and 5 000.
//
// Requirement: FR-WLD-080
//
// Guard: a review rule on the raw ID puts every agent of an equal stride on one tick.
func TestScenarioS74_CadenceLoad(t *testing.T) {
	t.Parallel()
	for _, n := range []int64{1000, 5000} {
		for _, p := range []int64{10, 30} {
			bound := (12*n + 10*p - 1) / (10 * p)
			worst := int64(0)
			for _, start := range []int64{0, 100, 12345} {
				for stride := int64(1); stride <= 31; stride++ {
					load := make([]int64, p)
					for i := int64(0); i < n; i++ {
						b := ordering.Bucket(start+i*stride, p)
						load[(p-b)%p]++ // the agent reviews on the ticks t with (t+b) mod p == 0
					}
					worst = max(worst, slices.Max(load))
				}
			}
			t.Logf("S74 n %d period %d: worst tick %d, bound %d", n, p, worst, bound)
			if worst > bound {
				t.Errorf("n %d period %d: %d reviews on one tick, want at most %d", n, p, worst, bound)
			}
		}
	}
}

// TestScenarioS76_StreamAfterChange is the unit model of WLD-S4: 200 agents over 50 ticks,
// one draw each a tick from a stream per (site, tick, agent); one agent takes one extra
// draw in tick 1. Reported: the share of later draws that differ (the model gives 0.01%).
// The two runs of `one` with a Strategy double wait on the code of the double.
//
// Requirement: FR-WLD-069
//
// Guard: one positional stream moves 98% of the later draws.
func TestScenarioS76_StreamAfterChange(t *testing.T) {
	t.Parallel()
	const agents, ticks, seed, site = 200, 50, 7, 1
	run := func(extraTick int64) [][]uint64 {
		out := make([][]uint64, ticks)
		for tick := int64(0); tick < ticks; tick++ {
			for a := int64(0); a < agents; a++ {
				hi, lo := ordering.StreamKey(seed, site, tick, a)
				r := rand.NewPCG(hi, lo)
				if tick == extraTick && a == 0 {
					r.Uint64()
				}
				out[tick] = append(out[tick], r.Uint64())
			}
		}
		return out
	}
	base, changed := run(-1), run(1)
	diff := 0
	for i := range base {
		for j := range base[i] {
			if base[i][j] != changed[i][j] {
				diff++
			}
		}
	}
	t.Logf("S76 draws that differ: %d of %d", diff, agents*ticks)
	if diff != 1 {
		t.Errorf("%d later draws differ, want exactly the one of the changed stream", diff)
	}
}
