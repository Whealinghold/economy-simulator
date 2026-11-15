package tenure_test

import (
	"math"
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/tenure"
)

// TestAttribute_LessorSupplyByUnusedWeight verifies the supply of each lessor
// is the apportionment of the let units taken by the weights of the unused
// entitlements (spec-land.md 4.4.1): of 5 units taken against unused 6 and 2
// (weights 3 : 1), 5 * 6 / 8 = 3 rem 6 and 5 * 2 / 8 = 1 rem 2, and the unit left
// goes to the larger remainder, so the supplies are 4 and 1, never above 6 and 2.
//
// Requirement: FR-LND-013
func TestAttribute_LessorSupplyByUnusedWeight(t *testing.T) {
	t.Parallel()
	out := make([]int64, 2)
	lessors := []tenure.Part{{Ref: 0, ID: 1, Share: 6}, {Ref: 1, ID: 2, Share: 2}}
	if !tenure.Supply(5, lessors, out) {
		t.Fatalf("Supply(5, %+v) not ok", lessors)
	}
	if want := []int64{4, 1}; !slices.Equal(out, want) {
		t.Errorf("Supply(5, %+v) = %v, want %v", lessors, out, want)
	}
	if tenure.Supply(9, lessors, out) || out[0] != 0 || out[1] != 0 {
		t.Errorf("Supply(9, %+v) = %v, want refused with 0: more than the let supply", lessors, out)
	}
}

// TestAttribute_SupplyNeverAboveUnused checks over random weights that the
// supply of a lessor never exceeds its unused entitlement and the supplies sum
// to the units taken, since ceil(V * U / L) <= U for V <= L.
//
// Requirement: FR-LND-013, NFR-LND-003
func TestAttribute_SupplyNeverAboveUnused(t *testing.T) {
	t.Parallel()
	rng := rand.New(rand.NewPCG(5, 6))
	for range 300 {
		n := 1 + rng.IntN(8)
		parts := make([]tenure.Part, n)
		var total int64
		for i := range parts {
			parts[i] = tenure.Part{Ref: i, ID: int64(i + 1), Share: int64(rng.IntN(40)), Key: uint64(rng.IntN(3))}
			total += parts[i].Share
		}
		v := int64(rng.IntN(int(total) + 1))
		out := make([]int64, n)
		if !tenure.Supply(v, slices.Clone(parts), out) {
			t.Fatalf("Supply(%d, %+v) not ok", v, parts)
		}
		for i, s := range out {
			if s > parts[i].Share {
				t.Fatalf("Supply(%d, %+v) = %v, lessor %d above its unused", v, parts, out, i)
			}
		}
		if sum(out) != v {
			t.Fatalf("Supply(%d, %+v) = %v, sums to %d", v, parts, out, sum(out))
		}
	}
}

// TestAttribute_AscendingTenantsAgainstLessors verifies the two-pointer walk:
// tenants 5 (3 units) and 7 (2 units) against lessors 2 (4 units) and 9 (1
// unit), in ascending ID, give 5 -> 2 for 3, 7 -> 2 for 1 and 7 -> 9 for 1. The
// indices are those of the slices as passed in, which are listed out of order.
//
// Requirement: FR-LND-013
func TestAttribute_AscendingTenantsAgainstLessors(t *testing.T) {
	t.Parallel()
	tenants := []tenure.Taker{{ID: 7, Units: 2}, {ID: 5, Units: 3}}
	lessors := []tenure.Taker{{ID: 9, Units: 1}, {ID: 2, Units: 4}}
	got, ok := new(tenure.Scratch).Attribute(nil, tenants, lessors)
	if !ok {
		t.Fatalf("Attribute(%+v, %+v) not ok", tenants, lessors)
	}
	want := []tenure.Triple{{Tenant: 1, Lessor: 1, Units: 3}, {Tenant: 0, Lessor: 1, Units: 1}, {Tenant: 0, Lessor: 0, Units: 1}}
	if !slices.Equal(got, want) {
		t.Errorf("Attribute(%+v, %+v) = %v, want %v", tenants, lessors, got, want)
	}
	if tenants[0].ID != 7 || lessors[0].ID != 9 {
		t.Errorf("Attribute reordered its input: %+v %+v", tenants, lessors)
	}
}

// TestAttribute_SelfTransferSkippedUnitsStillLet verifies a tenant that is its
// own lessor is assigned its units like any other, marked Self so the transfer
// is skipped, and that the units still count in the triples (let units).
//
// Requirement: FR-LND-017
func TestAttribute_SelfTransferSkippedUnitsStillLet(t *testing.T) {
	t.Parallel()
	got, ok := new(tenure.Scratch).Attribute(nil, []tenure.Taker{{ID: 4, Units: 2}}, []tenure.Taker{{ID: 4, Units: 2}})
	if want := []tenure.Triple{{Tenant: 0, Lessor: 0, Units: 2, Self: true}}; !ok || !slices.Equal(got, want) {
		t.Errorf("Attribute(self) = %v, %v, want %v", got, ok, want)
	}
}

// TestAttribute_Refusals verifies unequal sums, a repeated ID, a negative unit
// count and a sum beyond int64 are refused with the destination unchanged.
//
// Requirement: FR-LND-013
func TestAttribute_Refusals(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name             string
		tenants, lessors []tenure.Taker
	}{
		{"sums differ", []tenure.Taker{{ID: 1, Units: 3}}, []tenure.Taker{{ID: 2, Units: 2}}},
		{"repeated tenant", []tenure.Taker{{ID: 1, Units: 1}, {ID: 1, Units: 1}}, []tenure.Taker{{ID: 2, Units: 2}}},
		{"repeated lessor", []tenure.Taker{{ID: 1, Units: 2}}, []tenure.Taker{{ID: 2, Units: 1}, {ID: 2, Units: 1}}},
		{"negative units", []tenure.Taker{{ID: 1, Units: -1}}, []tenure.Taker{{ID: 2, Units: -1}}},
		{"tenant sum overflows", []tenure.Taker{{ID: 1, Units: math.MaxInt64}, {ID: 3, Units: 1}}, []tenure.Taker{{ID: 2, Units: 1}}},
		{"lessor sum overflows", []tenure.Taker{{ID: 2, Units: 1}}, []tenure.Taker{{ID: 1, Units: math.MaxInt64}, {ID: 3, Units: 1}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dst := []tenure.Triple{{Units: 1}}
			got, ok := new(tenure.Scratch).Attribute(dst, tt.tenants, tt.lessors)
			if ok || len(got) != 1 {
				t.Errorf("Attribute(%s) = %v, %v, want refused with the destination unchanged", tt.name, got, ok)
			}
		})
	}
}
