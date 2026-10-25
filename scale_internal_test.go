package economysim

import (
	"math"
	"slices"
	"testing"
)

// TestGroupWeights verifies the pure derivation of the agents and weights of a
// group: ceil(H / S) agents, raised to min(K, H), weights that sum to H and
// differ by at most 1, the first H mod a agents taking the extra household
// (spec-scale.md 3.2). The expected weights are worked out in each row.
//
// Requirement: FR-SCL-001, FR-SCL-002, FR-SCL-003, FR-SCL-004, FR-SCL-005
func TestGroupWeights(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name              string
		households, scale int64
		minAgents         int64
		want              []int64
	}{
		// H 1, S 1: one agent of weight 1 (the floor min(K, H) is 1).
		{"one household", 1, 1, 10, []int64{1}},
		// H 10, S 4: ceil(10 / 4) = 3 agents; 10 = 3 * 3 + 1, so the first agent takes
		// the extra household: 4, 3, 3.
		{"ten over scale four", 10, 4, 1, []int64{4, 3, 3}},
		// H 2 below K 3: ceil(2 / 100) = 1, raised to min(3, 2) = 2 agents of 1.
		{"fewer households than the floor", 2, 100, 3, []int64{1, 1}},
		// H 40, S 40, K 1: one agent holding all forty households.
		{"one agent holds the group", 40, 40, 1, []int64{40}},
		// H 7, S 100, K 3: ceil(7 / 100) = 1, raised to min(3, 7) = 3 agents; 7 = 3 * 2 + 1:
		// 3, 2, 2.
		{"floor raises the count", 7, 100, 3, []int64{3, 2, 2}},
		// H 100, S 1, K 10: ceil(100 / 1) = 100 agents of weight 1 (the floor is below).
		{"scale one gives one household each", 100, 1, 10, slices.Repeat([]int64{1}, 100)},
		// H 10, S 3, K 1: ceil(10 / 3) = 4 agents; 10 = 4 * 2 + 2: 3, 3, 2, 2.
		{"two extras", 10, 3, 1, []int64{3, 3, 2, 2}},
		// No household or no scale is a group validation rejects: no agents.
		{"no household", 0, 1, 1, nil},
		{"no scale", 5, 0, 1, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := groupWeights(tt.households, tt.scale, tt.minAgents)
			if !slices.Equal(got, tt.want) {
				t.Errorf("groupWeights(%d, %d, %d) = %v, want %v", tt.households, tt.scale, tt.minAgents, got, tt.want)
			}
			if tt.want == nil {
				return
			}
			var sum int64
			for _, w := range got {
				sum += w
			}
			if lo, hi := slices.Min(got), slices.Max(got); sum != tt.households || hi-lo > 1 || lo < 1 {
				t.Errorf("groupWeights(%d, %d, %d) = %v: sum %d, spread %d, min %d, want sum %d, spread at most 1, min at least 1",
					tt.households, tt.scale, tt.minAgents, got, sum, hi-lo, lo, tt.households)
			}
		})
	}
}

// TestAgentCount verifies the one formula of the number of agents of a group
// (spec-scale.md 3.2): ceil(H / S), raised to min(K, H), and 0 for a group
// validation rejects. The expected counts are worked out in each row.
//
// Requirement: FR-SCL-002, FR-SCL-020
func TestAgentCount(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                         string
		households, scale, minAgents int64
		want                         int64
	}{
		{"no household", 0, 1, 1, 0},
		{"no scale", 5, 0, 1, 0},
		{"negative scale", 5, -1, 1, 0},
		// H 10, S 5: 10 / 5 = 2 exactly, no remainder, so no extra agent.
		{"exact division", 10, 5, 1, 2},
		// H 11, S 5: 11 / 5 = 2 remainder 1, so ceil = 3.
		{"remainder adds one", 11, 5, 1, 3},
		// H 100, S 50, K 10: ceil = 2, below the floor, so min(10, 100) = 10.
		{"floor wins", 100, 50, 10, 10},
		// H 4, S 100, K 10: the floor is above the households, so min(10, 4) = 4
		// agents of weight 1 (legal and silent, FR-SCL-020).
		{"floor above households", 4, 100, 10, 4},
		{"one household", 1, 1, 10, 1},
		// H MaxHouseholds at S MaxHouseholds: ceil = 1, raised to the floor 10.
		{"MaxHouseholds at the largest scale", MaxHouseholds, MaxHouseholds, 10, 10},
		{"MaxHouseholds at scale 1 and 3", MaxHouseholds, 3, 1, 333_333_334},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := agentCount(tt.households, tt.scale, tt.minAgents); got != tt.want {
				t.Errorf("agentCount(%d, %d, %d) = %d, want %d", tt.households, tt.scale, tt.minAgents, got, tt.want)
			}
		})
	}
}

// TestGroupShape_AgreesWithGroupWeights verifies that the shape validation reads
// (agents and largest weight) is the length and the maximum of the weights the
// construction builds, over a grid of households, scales and floors and the
// edges of the bounds.
//
// Requirement: FR-SCL-002, FR-SCL-003
// Guard: the formula was written twice (construct.go and validate.go); two copies
// that drift apart would let validation bound a weight that construction does not
// build.
func TestGroupShape_AgreesWithGroupWeights(t *testing.T) {
	t.Parallel()
	check := func(h, s, k int64) {
		t.Helper()
		agents, maxWeight, scale, ok := groupShape(WorldParams{MinAgents: k}, Group{Households: h, Scale: s})
		w := groupWeights(h, s, k)
		if !ok || scale != s || agents != int64(len(w)) || maxWeight != slices.Max(w) {
			t.Fatalf("groupShape(K %d, H %d, S %d) = %d agents, max %d, scale %d, ok %t; groupWeights has %d agents, max %d",
				k, h, s, agents, maxWeight, scale, ok, len(w), slices.Max(w))
		}
	}
	for h := int64(1); h <= 400; h += 3 {
		for s := int64(1); s <= 50; s++ {
			for k := int64(1); k <= 20; k++ {
				check(h, s, k)
			}
		}
	}
	// The bounds: the largest group at a scale that keeps the agent count small.
	agents, maxWeight, _, ok := groupShape(WorldParams{MinAgents: 10}, Group{Households: MaxHouseholds, Scale: MaxHouseholds})
	if !ok || agents != 10 || maxWeight != MaxHouseholds/10 {
		t.Errorf("groupShape(MaxHouseholds at MaxHouseholds, K 10) = %d agents, max %d, ok %t, want 10 agents of %d", agents, maxWeight, ok, MaxHouseholds/10)
	}
	// The world's scale applies when the group gives none.
	if _, _, scale, ok := groupShape(WorldParams{Scale: 4, MinAgents: 1}, Group{Households: 40}); !ok || scale != 4 {
		t.Errorf("groupShape(world scale 4, group scale 0) scale = %d, ok %t, want 4", scale, ok)
	}
	for _, g := range []Group{{Households: 0, Scale: 1}, {Households: MaxHouseholds + 1, Scale: 1}, {Households: 5, Scale: -1}, {Households: 5, Scale: MaxHouseholds + 1}} {
		if _, _, _, ok := groupShape(WorldParams{Scale: 1, MinAgents: 1}, g); ok {
			t.Errorf("groupShape(%+v) ok = true, want false", g)
		}
	}
	if _, _, _, ok := groupShape(WorldParams{Scale: 1, MinAgents: 0}, Group{Households: 5}); ok {
		t.Errorf("groupShape(floor 0) ok = true, want false")
	}
}

// TestGroupWeights_RejectsWhatValidationRejects verifies that a group with no
// household or with a scale below 1, which validation rejects, gets no weights.
//
// Requirement: FR-SCL-001
func TestGroupWeights_RejectsWhatValidationRejects(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct{ households, scale int64 }{{0, 1}, {-3, 1}, {5, 0}, {5, -2}, {0, 0}} {
		if got := groupWeights(tt.households, tt.scale, 10); got != nil {
			t.Errorf("groupWeights(%d, %d, 10) = %v, want nil", tt.households, tt.scale, got)
		}
	}
}

// TestGroupWeights_DrawsNothing verifies that the weights of the groups of a
// configuration do not depend on the seed: worlds of seeds 1 and 2 plan the same
// weights, and the plan is built before the creation draws.
//
// Requirement: FR-SCL-004
func TestGroupWeights_DrawsNothing(t *testing.T) {
	t.Parallel()
	c := removalConfig()
	a, b := newMill(t, c, WithSeed(1)), newMill(t, c, WithSeed(2))
	pa, pb := groupPlans(&a.cfg), groupPlans(&b.cfg)
	if len(pa) == 0 || len(pa) != len(pb) {
		t.Fatalf("groupPlans of seeds 1 and 2 = %d and %d plans, want equal and non-zero", len(pa), len(pb))
	}
	for i := range pa {
		if !slices.Equal(pa[i].weights, pb[i].weights) {
			t.Errorf("group %d weights = %v at seed 1 and %v at seed 2, want equal", i, pa[i].weights, pb[i].weights)
		}
	}
}

// FuzzGroupWeights verifies the properties of the derivation over arbitrary
// inputs inside the bounds: the weights sum to the households, differ by at most
// 1, are each at least 1, their count is agentCount, and nothing panics. The
// inputs are mapped into the bounds, and a draw that would allocate more than
// 100 000 agents is skipped so the fuzzer cannot allocate the 10^9 agents of a
// group at scale 1 (plan 2-4 Risks, item 1).
//
// Requirement: FR-SCL-001, FR-SCL-002, FR-SCL-003
func FuzzGroupWeights(f *testing.F) {
	f.Add(uint64(2000), uint64(40), uint64(10))
	f.Add(uint64(105), uint64(40), uint64(10))
	f.Add(uint64(1), uint64(1), uint64(1))
	f.Add(uint64(math.MaxUint64), uint64(math.MaxUint64), uint64(math.MaxUint64))
	f.Fuzz(func(t *testing.T, rh, rs, rk uint64) {
		h, s, k := int64(1+rh%MaxHouseholds), int64(1+rs%MaxHouseholds), int64(1+rk%MaxHouseholds)
		n := agentCount(h, s, k)
		if n > 100_000 {
			return // a bounded input space, not a skipped case
		}
		w := groupWeights(h, s, k)
		var sum int64
		for _, x := range w {
			sum += x
		}
		if int64(len(w)) != n || sum != h || slices.Max(w)-slices.Min(w) > 1 || slices.Min(w) < 1 {
			t.Fatalf("groupWeights(%d, %d, %d): %d weights, sum %d, spread %d, min %d, want %d weights summing to %d, spread at most 1, min at least 1",
				h, s, k, len(w), sum, slices.Max(w)-slices.Min(w), slices.Min(w), n, h)
		}
	})
}
