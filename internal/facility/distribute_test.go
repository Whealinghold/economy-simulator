package facility_test

import (
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/facility"
)

// TestDistribute_MostWornFirst verifies the repair distribution of 4.3.2: the
// least condition first, ties to the lower Ref, each at most what it misses,
// and the remainder voided (spec-infrastructure.md 18, repair rows).
//
// Requirement: FR-CAP-015
func TestDistribute_MostWornFirst(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		units      int64
		need       []facility.Need
		want       []facility.Placement
		wantVoided int64
	}{
		{"fills the worst first", 150, []facility.Need{{Ref: 1, Condition: 900, Missing: 100}, {Ref: 2, Condition: 400, Missing: 600}},
			[]facility.Placement{{Ref: 2, N: 150}}, 0},
		{"spills to the next", 700, []facility.Need{{Ref: 1, Condition: 900, Missing: 100}, {Ref: 2, Condition: 400, Missing: 600}},
			[]facility.Placement{{Ref: 2, N: 600}, {Ref: 1, N: 100}}, 0},
		{"remainder voided", 900, []facility.Need{{Ref: 1, Condition: 900, Missing: 100}, {Ref: 2, Condition: 400, Missing: 600}},
			[]facility.Placement{{Ref: 2, N: 600}, {Ref: 1, N: 100}}, 200},
		{"tie to the lower ref", 5, []facility.Need{{Ref: 7, Condition: 10, Missing: 3}, {Ref: 4, Condition: 10, Missing: 3}},
			[]facility.Placement{{Ref: 4, N: 3}, {Ref: 7, N: 2}}, 0},
		{"full ones skipped", 4, []facility.Need{{Ref: 1, Condition: 10, Missing: 0}, {Ref: 2, Condition: 20, Missing: 9}},
			[]facility.Placement{{Ref: 2, N: 4}}, 0},
		{"nothing to repair", 6, nil, nil, 6},
		{"no units", 0, []facility.Need{{Ref: 1, Condition: 1, Missing: 5}}, nil, 0},
		{"negative units", -3, []facility.Need{{Ref: 1, Condition: 1, Missing: 5}}, nil, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, voided := facility.Distribute(nil, tt.units, tt.need)
			if !slices.Equal(got, tt.want) || voided != tt.wantVoided {
				t.Errorf("Distribute(%d, %v) = %v, %d, want %v, %d", tt.units, tt.need, got, voided, tt.want, tt.wantVoided)
			}
		})
	}
}

// TestDistribute_Conservation verifies over random lists that the units given
// plus the voided units equal the units offered, no facility gets more than it
// misses, and the order is by condition then Ref.
//
// Requirement: FR-CAP-015
func TestDistribute_Conservation(t *testing.T) {
	t.Parallel()
	for seed := uint64(1); seed <= 2000; seed++ {
		r := rand.New(rand.NewPCG(seed, 5))
		need := make([]facility.Need, 1+r.IntN(6))
		for i := range need {
			need[i] = facility.Need{Ref: i + 1, Condition: r.Int64N(5), Missing: r.Int64N(6)}
		}
		missing := map[int]int64{}
		for _, n := range need {
			missing[n.Ref] = n.Missing
		}
		units := r.Int64N(30)
		got, voided := facility.Distribute(nil, units, need)
		var sum int64
		for i, p := range got {
			sum += p.N
			if p.N < 1 || p.N > missing[p.Ref] {
				t.Fatalf("seed %d: placement %v outside [1, %d]", seed, p, missing[p.Ref])
			}
			if i > 0 && need[i].Condition < need[i-1].Condition {
				t.Fatalf("seed %d: order broken at %d: %v", seed, i, need)
			}
		}
		if sum+voided != units {
			t.Fatalf("seed %d: given %d + voided %d != offered %d", seed, sum, voided, units)
		}
	}
}
