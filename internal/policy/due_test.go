package policy_test

import (
	"math/big"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/policy"
)

// TestDue_Bucket verifies the due ticks of spec-policy.md 14.1: actor 0 is due
// at every multiple of its cadence, actor 1 at 6 mod 10 (bucket 4 of 10) and
// actor 2 at 6 mod 30 (bucket 24 of 30), and a cadence below 1 is never due.
//
// bucket(a, P) = (Weyl2(a) * P) >> 32 with Weyl2(1) = 1 779 033 703, so
// (1 779 033 703 * 10) >> 32 = 4 and (3 558 067 406 * 30) >> 32 = 24.
//
// Requirement: FR-PLC-051
func TestDue_Bucket(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		actor, every int64
		due          []int64
		notDue       []int64
	}{
		{"actor 0 every 10", 0, 10, []int64{0, 10, 20}, []int64{1, 6, 15}},
		{"actor 1 every 10", 1, 10, []int64{6, 16, 26}, []int64{0, 4, 10}},
		{"actor 2 every 30", 2, 30, []int64{6, 36, 66, 96, 126}, []int64{0, 30, 7}},
		{"actor 2 every 10 shares bucket 8", 2, 10, []int64{2, 12}, []int64{8}},
		{"every 1 is every tick", 5, 1, []int64{0, 1, 2, 3}, nil},
		{"every 0 is never due", 0, 0, nil, []int64{0, 1}},
		{"negative every is never due", 3, -4, nil, []int64{0, 4}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			for _, tick := range tt.due {
				if !policy.Due(tick, tt.actor, tt.every) {
					t.Errorf("Due(%d, %d, %d) = false, want true", tick, tt.actor, tt.every)
				}
			}
			for _, tick := range tt.notDue {
				if policy.Due(tick, tt.actor, tt.every) {
					t.Errorf("Due(%d, %d, %d) = true, want false", tick, tt.actor, tt.every)
				}
			}
		})
	}
}

// refDue is the rotation of spec-policy.md 7.2.1 in exact integers.
func refDue(tick, actor, every int64) bool {
	if every < 1 {
		return false
	}
	w := new(big.Int).SetUint64(uint64(uint32(actor) * 0x6A09E667))
	b := new(big.Int).Rsh(w.Mul(w, big.NewInt(every)), 32).Int64()
	return (tick+b)%every == 0
}

// FuzzDue fuzzes Due against the reference of 7.2.1, and checks that exactly one
// tick in every window of `every` ticks is due.
//
// Requirement: FR-PLC-051
func FuzzDue(f *testing.F) {
	f.Add(int64(6), int64(1), int64(10))
	f.Add(int64(0), int64(0), int64(1))
	f.Add(int64(1_000_000), int64(63), int64(3650))
	f.Fuzz(func(t *testing.T, tick, actor, every int64) {
		if tick < 0 || tick > 1<<40 || every > 3650 {
			t.Skip()
		}
		if got, want := policy.Due(tick, actor, every), refDue(tick, actor, every); got != want {
			t.Fatalf("Due(%d, %d, %d) = %v, want %v", tick, actor, every, got, want)
		}
		if every < 1 {
			return
		}
		n := 0
		for k := int64(0); k < every; k++ {
			if policy.Due(tick+k, actor, every) {
				n++
			}
		}
		if n != 1 {
			t.Fatalf("%d due ticks in a window of %d, want 1", n, every)
		}
	})
}
