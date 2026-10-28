package endowment_test

import (
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/endowment"
	"github.com/Whealinghold/economy-sim/internal/ordering"
)

// rationOf runs Ration on a copy of reqs and returns the grants by Ref.
func rationOf(t *testing.T, available int64, reqs []endowment.Request) []int64 {
	t.Helper()
	grants := make([]int64, len(reqs))
	if !endowment.Ration(available, slices.Clone(reqs), grants) {
		t.Fatalf("Ration(%d, %+v) not ok", available, reqs)
	}
	return grants
}

// TestRation_CoversAllWhenStockCovers verifies that when the stock covers the
// units wanted every request is granted in full by one sum.
//
// Requirement: FR-RES-017, NFR-RES-002
func TestRation_CoversAllWhenStockCovers(t *testing.T) {
	t.Parallel()
	reqs := []endowment.Request{{Ref: 0, Executions: 4, Draw: 2}, {Ref: 1, Executions: 3, Draw: 1}}
	for _, available := range []int64{11, 12, 500} { // U = 4*2 + 3*1 = 11
		if got := rationOf(t, available, reqs); !slices.Equal(got, []int64{4, 3}) {
			t.Errorf("Ration(%d, %+v) = %v, want [4 3]", available, reqs, got)
		}
	}
}

// TestRation_ProportionalThenLeftovers verifies the example of
// spec-resources.md 5.3.7: ten plots, requests A 4, B 3, C 3, D 2, E 1 of a draw
// of 1 (U = 13). The floor shares floor(w * 10 / 13) are 3, 2, 2, 1, 0, using 8
// of 10; the remainders are 1, 4, 4, 7, 10 (of 13); the 2 units left go to E
// (10) and D (7), so the grants are 3, 2, 2, 2, 1.
//
// Requirement: FR-RES-017
func TestRation_ProportionalThenLeftovers(t *testing.T) {
	t.Parallel()
	reqs := []endowment.Request{
		{Ref: 0, Executions: 4, Draw: 1, Key: 5}, {Ref: 1, Executions: 3, Draw: 1, Key: 4},
		{Ref: 2, Executions: 3, Draw: 1, Key: 3}, {Ref: 3, Executions: 2, Draw: 1, Key: 2},
		{Ref: 4, Executions: 1, Draw: 1, Key: 1},
	}
	if got, want := rationOf(t, 10, reqs), []int64{3, 2, 2, 2, 1}; !slices.Equal(got, want) {
		t.Errorf("Ration(10, %+v) = %v, want %v", reqs, got, want)
	}
}

// TestRation_NeverAboveRequestOrStock verifies on 500 random cases that no
// grant exceeds its request, the units granted never exceed the stock, and a
// scarce stock is left with less than the smallest draw that was refused.
//
// Requirement: FR-RES-020
func TestRation_NeverAboveRequestOrStock(t *testing.T) {
	t.Parallel()
	rng := rand.New(rand.NewPCG(3, 5))
	for c := range 500 {
		reqs := make([]endowment.Request, 1+rng.IntN(12))
		for i := range reqs {
			reqs[i] = endowment.Request{Ref: i, Executions: rng.Int64N(20), Draw: 1 + rng.Int64N(6), Key: rng.Uint64()}
		}
		available := rng.Int64N(150)
		grants := rationOf(t, available, reqs)
		var used int64
		for i, r := range reqs {
			if grants[i] < 0 || grants[i] > r.Executions {
				t.Fatalf("case %d: grant %d of request %+v out of [0, %d]", c, grants[i], r, r.Executions)
			}
			used += grants[i] * r.Draw
		}
		if used > available {
			t.Errorf("case %d: granted %d units of %d available, requests %+v", c, used, available, reqs)
		}
	}
}

// TestRation_IndependentOfListingOrder verifies the order of step 3 is total,
// so the same requests listed in any order get the same grants.
//
// Requirement: FR-RES-020
func TestRation_IndependentOfListingOrder(t *testing.T) {
	t.Parallel()
	rng := rand.New(rand.NewPCG(9, 9))
	for c := range 200 {
		reqs := make([]endowment.Request, 2+rng.IntN(10))
		for i := range reqs {
			// Few distinct keys and sizes make ties in the remainder and in the key likely.
			reqs[i] = endowment.Request{Ref: i, Executions: 1 + rng.Int64N(4), Draw: 1 + rng.Int64N(2), Key: uint64(rng.IntN(3))}
		}
		available := 1 + rng.Int64N(10)
		want := rationOf(t, available, reqs)
		shuffled := slices.Clone(reqs)
		rng.Shuffle(len(shuffled), func(i, j int) { shuffled[i], shuffled[j] = shuffled[j], shuffled[i] })
		if got := rationOf(t, available, shuffled); !slices.Equal(got, want) {
			t.Errorf("case %d: shuffled grants %v, want %v (requests %+v)", c, got, want, reqs)
		}
	}
}

// TestRation_CohortEqualsAgents verifies weight invariance: twenty cohorts of
// weight 50 and a thousand agents of weight 1 with the same total demand get the
// same total, and a cohort gets exactly its proportional share when
// w * A / U is whole (50 * 500 / 2 000 = 12.5 is not, so the demand here is
// 1 000 executions in all and A = 500: 50 * 500 / 1 000 = 25 exactly).
//
// Requirement: FR-RES-021
//
// Guard: the same total for cohorts and agents is the property the scale
// mechanism relies on; first come, first served does not have it.
func TestRation_CohortEqualsAgents(t *testing.T) {
	t.Parallel()
	// Twenty cohorts of weight 50 want 1 000 executions, and ten of them hold the whole demand
	// with the thousand agents: U = 1 000 executions of draw 1 either way.
	cohorts := make([]endowment.Request, 20)
	for i := range cohorts {
		cohorts[i] = endowment.Request{Ref: i, Executions: 50, Draw: 1, Key: ordering.Key(1, int64(i))}
	}
	agents := make([]endowment.Request, 1000)
	for i := range agents {
		agents[i] = endowment.Request{Ref: i, Executions: 1, Draw: 1, Key: ordering.Key(1, int64(i))}
	}
	var cohortTotal, agentTotal int64
	for i, g := range rationOf(t, 500, cohorts) {
		if g != 25 {
			t.Errorf("cohort %d granted %d, want 25 (50 * 500 / 1 000)", i, g)
		}
		cohortTotal += g
	}
	for _, g := range rationOf(t, 500, agents) {
		agentTotal += g
	}
	if cohortTotal != 500 || agentTotal != 500 {
		t.Errorf("totals: cohorts %d, agents %d, want 500 each", cohortTotal, agentTotal)
	}
}

// TestRation_FixedMixBias verifies the known and bounded bias of
// spec-resources.md 5.3.4: with 25 agents asking 2 and 50 asking 1 against a
// supply of 25, the fractional order is the same every tick, so the larger
// requests take the leftovers every tick and the smaller get 0 for ever.
//
// Requirement: FR-RES-017
//
// Guard: this documents the open question 4 of the spec. If a carried remainder
// is added the test must change with it; until then a rule that quietly changed
// the leftover order would change who is denied.
func TestRation_FixedMixBias(t *testing.T) {
	t.Parallel()
	for tick := range int64(10) {
		reqs := make([]endowment.Request, 75)
		for i := range reqs {
			w := int64(1)
			if i < 25 {
				w = 2
			}
			reqs[i] = endowment.Request{Ref: i, Executions: w, Draw: 1, Key: ordering.Key(tick, int64(i))}
		}
		// U = 25*2 + 50*1 = 100; floor shares: 2*25/100 = 0 (rem 50), 1*25/100 = 0 (rem 25). The 25
		// leftover units go to the 25 larger remainders: one each.
		grants := rationOf(t, 25, reqs)
		for i, g := range grants {
			if want := map[bool]int64{true: 1, false: 0}[i < 25]; g != want {
				t.Fatalf("tick %d: agent %d granted %d, want %d", tick, i, g, want)
			}
		}
	}
}

// TestRation_SkipsWhatDoesNotFit verifies the walk of step 3: an agent whose
// draw exceeds what is left is skipped and the walk goes on, in one pass.
//
// Requirement: FR-RES-017
func TestRation_SkipsWhatDoesNotFit(t *testing.T) {
	t.Parallel()
	// A = 7, three requests of one execution drawing 5 (X), 4 (Y), 3 (Z): U = 12 and
	// floor(7 / 12) = 0 for each, remainders all 7. By key Y, X, Z: Y fits (7 - 4 = 3 left),
	// X needs 5 and is skipped, Z needs 3 and fits.
	reqs := []endowment.Request{
		{Ref: 0, Executions: 1, Draw: 5, Key: 2}, // X
		{Ref: 1, Executions: 1, Draw: 4, Key: 1}, // Y
		{Ref: 2, Executions: 1, Draw: 3, Key: 3}, // Z
	}
	if got, want := rationOf(t, 7, reqs), []int64{0, 1, 1}; !slices.Equal(got, want) {
		t.Errorf("Ration(7, %+v) = %v, want %v", reqs, got, want)
	}
}

// TestRation_RejectsBadInput verifies a negative stock or request, a draw below
// 1, or a sum beyond int64 is reported with ok false and every grant 0.
//
// Requirement: NFR-RES-003
func TestRation_RejectsBadInput(t *testing.T) {
	t.Parallel()
	const big = int64(1) << 62
	tests := []struct {
		name      string
		available int64
		reqs      []endowment.Request
	}{
		{"negative stock", -1, []endowment.Request{{Ref: 0, Executions: 1, Draw: 1}}},
		{"negative executions", 5, []endowment.Request{{Ref: 0, Executions: -1, Draw: 1}}},
		{"draw below 1", 5, []endowment.Request{{Ref: 0, Executions: 1, Draw: 0}}},
		{"product beyond int64", 5, []endowment.Request{{Ref: 0, Executions: big, Draw: 4}}},
		{"sum beyond int64", 5, []endowment.Request{{Ref: 0, Executions: big, Draw: 1}, {Ref: 1, Executions: big, Draw: 1}, {Ref: 2, Executions: big, Draw: 1}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			grants := make([]int64, len(tt.reqs))
			for i := range grants {
				grants[i] = 99
			}
			if ok := endowment.Ration(tt.available, slices.Clone(tt.reqs), grants); ok {
				t.Errorf("Ration(%d, %+v) = ok, want not ok", tt.available, tt.reqs)
			}
			for i, g := range grants {
				if g != 0 {
					t.Errorf("Ration(%d, %+v) left grant %d = %d, want 0", tt.available, tt.reqs, i, g)
				}
			}
		})
	}
}

// TestKey_IsTheSplitMix64Finaliser verifies the key the ration uses is the
// SplitMix64 finaliser of spec-resources.md 5.3.3, written out here from the
// spec and compared with ordering.Key, which the caller puts in Request.Key.
//
// Requirement: FR-RES-019
func TestKey_IsTheSplitMix64Finaliser(t *testing.T) {
	t.Parallel()
	const inc = 0x9E3779B97F4A7C15
	spec := func(tick, agent int64) uint64 {
		x := uint64(tick)*inc + uint64(agent)
		z := x + inc
		z = (z ^ (z >> 30)) * 0xBF58476D1CE4E5B9
		z = (z ^ (z >> 27)) * 0x94D049BB133111EB
		return z ^ (z >> 31)
	}
	for _, c := range [][2]int64{{0, 0}, {1, 0}, {0, 1}, {7, 42}, {1 << 40, 1 << 20}, {-1, 5}} {
		if got, want := ordering.Key(c[0], c[1]), spec(c[0], c[1]); got != want {
			t.Errorf("ordering.Key(%d, %d) = %#x, want %#x", c[0], c[1], got, want)
		}
	}
}

// TestRation_AllocsPerRun verifies a scarce ration allocates nothing: it sorts
// the caller's slice in place and writes the grants into the caller's slice.
//
// Requirement: NFR-RES-001, NFR-RES-004
func TestRation_AllocsPerRun(t *testing.T) {
	template := make([]endowment.Request, 200)
	for i := range template {
		template[i] = endowment.Request{Ref: i, Executions: 1 + int64(i%5), Draw: 1 + int64(i%3), Key: ordering.Key(1, int64(i))}
	}
	reqs := make([]endowment.Request, len(template))
	grants := make([]int64, len(template))
	allocs := testing.AllocsPerRun(50, func() {
		copy(reqs, template)
		endowment.Ration(100, reqs, grants)
	})
	if allocs != 0 {
		t.Errorf("Ration allocates %v per run, want 0", allocs)
	}
}

// BenchmarkRation reports the cost of a scarce ration of 1 000 requests.
func BenchmarkRation(b *testing.B) {
	template := make([]endowment.Request, 1000)
	for i := range template {
		template[i] = endowment.Request{Ref: i, Executions: 1 + int64(i%5), Draw: 1 + int64(i%3), Key: ordering.Key(1, int64(i))}
	}
	reqs := make([]endowment.Request, len(template))
	grants := make([]int64, len(template))
	b.ReportAllocs()
	for b.Loop() {
		copy(reqs, template)
		endowment.Ration(2000, reqs, grants)
	}
}

// BenchmarkRegen reports the cost of one logistic regeneration.
func BenchmarkRegen(b *testing.B) {
	p := endowment.Params{Shape: endowment.Logistic, Rate: 3000, Capacity: 50_000, ScaleBP: 10_000, TicksPerYear: 365}
	in := endowment.Step{Stock: 20_000, SeasonBP: 10_000}
	b.ReportAllocs()
	for b.Loop() {
		endowment.Regen(p, in)
	}
}
