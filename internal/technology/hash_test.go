package technology_test

import (
	"math/big"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/ordering"
	"github.com/Whealinghold/economy-sim/internal/technology"
)

// oracleHit is the hit of spec-technology.md 4.4.2 and 4.4.3 written with
// math/big, so the test does not share the implementation's uint64 shortcuts.
func oracleHit(key uint64, tick, cell, cells, rate, tpy int64) bool {
	mod := new(big.Int).Lsh(big.NewInt(1), 64)
	idx := new(big.Int).Mul(big.NewInt(tick), big.NewInt(cells))
	idx.Add(idx, big.NewInt(cell+1))
	arg := new(big.Int).Mul(idx, new(big.Int).SetUint64(ordering.Increment))
	arg.Add(arg, new(big.Int).SetUint64(key))
	arg.Mod(arg, mod)
	k := new(big.Int).SetUint64(ordering.Finalise(arg.Uint64()))
	k.Mul(k, big.NewInt(10_000*tpy))
	k.Rsh(k, 64)
	return k.Cmp(big.NewInt(rate)) < 0
}

// TestHit_Threshold verifies the hit against a math/big evaluation of the
// spec's formula for two keys, many ticks and cells, and that a rate of 0 or a
// cell outside the grid never hits.
//
// Requirement: FR-TEC-009
func TestHit_Threshold(t *testing.T) {
	t.Parallel()
	for _, key := range []uint64{0, 0xDEADBEEFCAFEF00D, ^uint64(0)} {
		for tick := int64(0); tick < 50; tick++ {
			for cell := int64(0); cell < 6; cell++ {
				got := technology.Hit(key, tick, cell, 6, 5_000, 365)
				if want := oracleHit(key, tick, cell, 6, 5_000, 365); got != want {
					t.Fatalf("Hit(%#x, %d, %d, 6, 5000, 365) = %v, want %v", key, tick, cell, got, want)
				}
			}
		}
	}
}

// TestHit_GuardsNeverHit verifies the refusal form of Hit.
//
// Requirement: FR-TEC-009
func TestHit_GuardsNeverHit(t *testing.T) {
	t.Parallel()
	if technology.Hit(1, 3, 0, 6, 0, 365) || technology.Hit(1, 3, 6, 6, 10_000, 365) ||
		technology.Hit(1, 3, -1, 6, 10_000, 365) || technology.Hit(1, -1, 0, 6, 10_000, 365) ||
		technology.Hit(1, 3, 0, 6, 10_000, 0) || technology.Hit(1, 3, 0, 6, 10_000, 1<<62) {
		t.Errorf("Hit accepted a rate of 0, a cell outside the grid, a negative tick, no ticks per year or a span that overflows")
	}
}

// TestHit_RateOverCellTicks verifies the stated rate: 10 cells over 20 years
// of 365 ticks at DiscoveryBP 5 000 is 73 000 cell-ticks at a probability of
// 5000 / (10 000 * 365) each, so about 100 hits, and two keys and two cells
// give different hit sets (models/tec/law.out part 3).
//
// The window: the count is binomial with mean 100 and standard deviation about
// 10, so the fixed keys are checked to sit inside 60 to 140 (four deviations).
//
// Requirement: FR-TEC-009
func TestHit_RateOverCellTicks(t *testing.T) {
	t.Parallel()
	count := func(key uint64) (n int, on [2][]int64) {
		for tick := int64(0); tick < 20*365; tick++ {
			for cell := int64(0); cell < 10; cell++ {
				if technology.Hit(key, tick, cell, 10, 5_000, 365) {
					n++
					if cell < 2 {
						on[cell] = append(on[cell], tick)
					}
				}
			}
		}
		return n, on
	}
	n1, on1 := count(technology.Key(1, 0xF68B3E6A63DA053C))
	n2, on2 := count(technology.Key(2, 0xF68B3E6A63DA053C))
	for i, n := range []int{n1, n2} {
		if n < 60 || n > 140 {
			t.Errorf("key %d: %d hits over 73000 cell-ticks, want about 100 (60 to 140)", i+1, n)
		}
	}
	if len(on1[0]) == len(on2[0]) && len(on1[0]) > 0 && on1[0][0] == on2[0][0] {
		t.Errorf("two keys hit the same first tick %d on cell 0", on1[0][0])
	}
	if len(on1[0]) > 0 && len(on1[1]) > 0 && on1[0][0] == on1[1][0] {
		t.Errorf("cells 0 and 1 hit the same first tick %d", on1[0][0])
	}
}

// TestKey_FollowsTheSeed verifies that the key is the high word of the stream
// key of (site, tick 0, entity 0) and that a different seed gives a different
// key, which is how a reseed changes it.
//
// Requirement: FR-TEC-011, FR-TEC-012
func TestKey_FollowsTheSeed(t *testing.T) {
	t.Parallel()
	const site = 0xF68B3E6A63DA053C
	hi, _ := ordering.StreamKey(42, site, 0, 0)
	if got := technology.Key(42, site); got != hi {
		t.Errorf("Key(42, site) = %#x, want the high word %#x", got, hi)
	}
	if technology.Key(42, site) == technology.Key(43, site) {
		t.Errorf("Key(42) equals Key(43): a reseed would not change the key")
	}
}
