package steady_test

import (
	"fmt"
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/steady"
)

// tickFactors expands runs to one factor per tick.
func tickFactors(year []steady.Run) []int64 {
	var f []int64
	for _, r := range year {
		for i := int64(0); i < r.Length; i++ {
			f = append(f, r.FactorBP)
		}
	}
	return f
}

// naiveStarts is the reference: walk the year tick by tick, start at the first
// free tick whose factor is positive, stay busy for d ticks (the predicate of A3:
// a start counts at every tick of the open window).
func naiveStarts(d int64, year []steady.Run) (starts []int64, factorSum int64) {
	f := tickFactors(year)
	for t := int64(0); t < int64(len(f)); {
		if f[t] > 0 {
			starts = append(starts, t)
			factorSum += f[t]
			t += d
		} else {
			t++
		}
	}
	return starts, factorSum
}

// startProblems checks a StartCount against the naive loop and against the
// structural properties of the greedy sequence, and returns a description of
// each that fails.
func startProblems(d int64, year []steady.Run, maxFactor int64) []string {
	var out []string
	sc := steady.Starts(d, year)
	wantStarts, wantSum := naiveStarts(d, year)
	if sc.Starts != int64(len(wantStarts)) || sc.FactorSum != wantSum {
		out = append(out, fmt.Sprintf("Starts = (%d, %d), naive loop gives (%d, %d)", sc.Starts, sc.FactorSum, len(wantStarts), wantSum))
	}
	if int64(len(sc.Completions)) != sc.Starts {
		return append(out, fmt.Sprintf("%d completions for %d starts", len(sc.Completions), sc.Starts))
	}
	out = append(out, sequenceProblems(d, year, sc)...)
	if sc.FactorSum > sc.Starts*maxFactor {
		out = append(out, fmt.Sprintf("FactorSum %d above J*max %d", sc.FactorSum, sc.Starts*maxFactor))
	}
	return out
}

// earth is the Earth default year of 365 ticks whose spring starts on day 59:
// 59 ticks of winter, spring 92, summer 92, autumn 91, winter 31.
func earth(winter, spring, summer, autumn int64) []steady.Run {
	return []steady.Run{{Length: 59, FactorBP: winter}, {Length: 92, FactorBP: spring}, {Length: 92, FactorBP: summer}, {Length: 91, FactorBP: autumn}, {Length: 31, FactorBP: winter}}
}

// TestStarts verifies the greedy start count of one household: where each job
// starts and completes, how many there are and the sum of their factors, for no
// profile, a sited recipe, an all-year profile, a spring window, a window at the
// end of the year, two plantings and a profile whose first open tick is the last.
//
// Requirement: FR-VIA-002
func TestStarts(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		d        int64
		year     []steady.Run
		starts   int64
		factor   int64
		complete []int64
	}{
		// No profile, d 60: starts at 0, 60, ..., 360, that is ceil(365/60) = 7;
		// the last completes at 360 + 59 = 419, past the year end, and counts.
		{"no profile", 60, []steady.Run{{Length: 365, FactorBP: 10000}}, 7, 70000, []int64{59, 119, 179, 239, 299, 359, 419}},
		// A sited recipe has factor 0 everywhere: J = 0, no activity.
		{"sited", 60, []steady.Run{{Length: 365, FactorBP: 0}}, 0, 0, nil},
		// All-year at d 1: a start every tick, 365 starts of 5 000.
		{"all year d 1", 1, []steady.Run{{Length: 365, FactorBP: 5000}}, 365, 365 * 5000, nil},
		// Spring only, d 60: spring is ticks 59..150; starts at 59 and 119 (119 + 60
		// = 179 is past the window), completions 59+59 = 118 and 119+59 = 178.
		{"spring only", 60, earth(0, 10000, 0, 0), 2, 20000, []int64{118, 178}},
		// A window on ticks 340..364, d 60: one start at 340, completing at 399 past
		// the year end.
		{"window at year end", 60, []steady.Run{{Length: 340, FactorBP: 0}, {Length: 25, FactorBP: 10000}}, 1, 10000, []int64{399}},
		// Spring 10 000 and autumn 7 000, d 120: start 59 (busy to 178), summer is
		// closed, autumn opens at 243 (59+92+92): start 243. Completions 59+119 =
		// 178 and 243+119 = 362; factors 10 000 + 7 000.
		{"two plantings", 120, earth(0, 10000, 0, 7000), 2, 17000, []int64{178, 362}},
		// The only open tick is the last: one start at 364, completion 364 + 4.
		{"first open tick is the last", 5, []steady.Run{{Length: 364, FactorBP: 0}, {Length: 1, FactorBP: 10000}}, 1, 10000, []int64{368}},
		// A job longer than the year starts once.
		{"duration beyond year", 500, []steady.Run{{Length: 365, FactorBP: 10000}}, 1, 10000, []int64{499}},
		// Zero-length runs and an empty year change nothing.
		{"zero length runs", 2, []steady.Run{{Length: 0, FactorBP: 3}, {Length: 4, FactorBP: 100}, {Length: 0, FactorBP: 9}}, 2, 200, []int64{1, 3}},
		{"empty year", 2, nil, 0, 0, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := steady.Starts(tt.d, tt.year)
			if got.Starts != tt.starts || got.FactorSum != tt.factor {
				t.Errorf("Starts(%d, %v) = (J %d, factor %d), want (J %d, factor %d)", tt.d, tt.year, got.Starts, got.FactorSum, tt.starts, tt.factor)
			}
			if tt.complete != nil && !slices.Equal(got.Completions, tt.complete) {
				t.Errorf("Starts(%d, %v).Completions = %v, want %v", tt.d, tt.year, got.Completions, tt.complete)
			}
			if p := startProblems(tt.d, tt.year, 10000); len(p) > 0 {
				t.Errorf("Starts(%d, %v) breaks the properties of the greedy sequence: %v", tt.d, tt.year, p)
			}
		})
	}
}

// TestStarts_PanicsOnBadInput verifies that a duration below 1 and a negative run
// are reported as violated programmer invariants.
//
// Guard: a zero duration would loop forever in a naive caller and a negative
// length would corrupt the tick count; the panic names the invariant.
func TestStarts_PanicsOnBadInput(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		d    int64
		year []steady.Run
	}{
		{"zero duration", 0, []steady.Run{{Length: 5, FactorBP: 1}}},
		{"negative length", 1, []steady.Run{{Length: -1, FactorBP: 1}}},
		{"negative factor", 1, []steady.Run{{Length: 1, FactorBP: -1}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("Starts(%d, %v) did not panic, want an invariant panic", tt.d, tt.year)
				}
			}()
			steady.Starts(tt.d, tt.year)
		})
	}
}

// TestStarts_MatchesNaiveLoop verifies the run-length start count against the
// naive tick loop on 200 random profiles and durations for each of three seeds.
// A failure names the seed and the case, so one rand.NewPCG(seed, 0) replays it.
//
// Requirement: FR-VIA-002
func TestStarts_MatchesNaiveLoop(t *testing.T) {
	t.Parallel()
	factors := []int64{0, 0, 3000, 10000, 12000}
	for _, seed := range []uint64{1, 2, 3} {
		t.Run(fmt.Sprintf("seed %d", seed), func(t *testing.T) {
			t.Parallel()
			rng := rand.New(rand.NewPCG(seed, 0))
			for i := 0; i < 200; i++ {
				var year []steady.Run
				for n := 1 + rng.IntN(8); n > 0; n-- {
					year = append(year, steady.Run{Length: int64(rng.IntN(81)), FactorBP: factors[rng.IntN(len(factors))]})
				}
				d := int64(1 + rng.IntN(150))
				if p := startProblems(d, year, 12000); len(p) > 0 {
					t.Fatalf("seed %d case %d: Starts(%d, %v): %v", seed, i, d, year, p)
				}
			}
		})
	}
}

// FuzzStarts verifies, for any year and duration, that the start count agrees
// with the naive loop and keeps the properties that replace "J * d is at most the
// year": starts strictly increase, each at least d after the previous, each at a
// positive-factor tick before the year end, the first is the first positive
// tick, no further start fits, (J-1)*d is below the year and the factor sum is at
// most J times the largest factor.
//
// Requirement: FR-VIA-002
func FuzzStarts(f *testing.F) {
	f.Add([]byte{60, 0, 100, 10, 0, 91, 5, 0, 0, 2, 4, 5})
	f.Add([]byte{1, 255, 255})
	f.Add([]byte{120, 59, 0, 92, 7, 92, 0, 91, 9})
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) == 0 {
			return
		}
		d := int64(data[0]%200) + 1
		var year []steady.Run
		var max int64
		for i := 1; i+1 < len(data) && len(year) < 12; i += 2 {
			fb := int64(data[i+1]%5) * 3000
			year = append(year, steady.Run{Length: int64(data[i]), FactorBP: fb})
			max = max64(max, fb)
		}
		if p := startProblems(d, year, max); len(p) > 0 {
			t.Fatalf("Starts(%d, %v): %v", d, year, p)
		}
	})
}

// max64 returns the larger of a and b.
func max64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

// sequenceProblems checks that the starts implied by the completions are in
// order, at least d apart, at positive-factor ticks before the year end, begin
// at the first positive tick and leave no room for another.
func sequenceProblems(d int64, year []steady.Run, sc steady.StartCount) []string {
	var out []string
	f := tickFactors(year)
	n := int64(len(f))
	for i, c := range sc.Completions {
		s := c - d + 1
		switch {
		case s < 0 || s >= n || f[s] <= 0:
			out = append(out, fmt.Sprintf("start %d is not at a tick of positive factor before the year end", s))
		case i > 0 && s < sc.Completions[i-1]+1:
			out = append(out, fmt.Sprintf("start %d is less than %d after the previous", s, d))
		}
	}
	if sc.Starts == 0 {
		return out
	}
	return append(out, edgeProblems(d, f, sc)...)
}

// edgeProblems checks the first start is the first positive tick and the last
// leaves no room for another.
func edgeProblems(d int64, f []int64, sc steady.StartCount) []string {
	var out []string
	n := int64(len(f))
	first := slices.IndexFunc(f, func(v int64) bool { return v > 0 })
	if got := sc.Completions[0] - d + 1; got != int64(first) {
		out = append(out, fmt.Sprintf("first start %d, first positive tick %d", got, first))
	}
	last := sc.Completions[sc.Starts-1] - d + 1
	for u := last + d; u < n; u++ {
		if f[u] > 0 {
			out = append(out, fmt.Sprintf("a further start was possible at tick %d", u))
			break
		}
	}
	if (sc.Starts-1)*d >= n {
		out = append(out, fmt.Sprintf("(J-1)*d = %d reaches the year %d", (sc.Starts-1)*d, n))
	}
	return out
}
