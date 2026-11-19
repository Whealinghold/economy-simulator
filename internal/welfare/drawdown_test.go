package welfare_test

import (
	"math"
	"math/big"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/welfare"
)

// cycle returns the records of one 20-tick job of spec-saving-and-investment.md
// 16.3: inputs of 300 at its start, no income for 19 ticks, sales of 2 500 at
// the end, upkeep 100 a tick. income is net of the inputs, so net_i is -400,
// -100 (18 times) and +2 400.
func cycle() []welfare.Record {
	recs := make([]welfare.Record, 20)
	for i := range recs {
		recs[i].Cost = 100
	}
	recs[0].Income = -300
	recs[19].Income = 2_500
	return recs
}

// TestDrawdown_Example16_3 checks the drawdown of 2 200 over a window of 40
// (two cycles), and over one cycle.
//
// Requirement: FR-SAV-006
func TestDrawdown_Example16_3(t *testing.T) {
	t.Parallel()
	one := cycle()
	if got := welfare.Drawdown(one); got != 2_200 {
		t.Errorf("Drawdown over one cycle = %d, want 2200", got)
	}
	two := append(cycle(), cycle()...)
	if got := welfare.Drawdown(two); got != 2_200 {
		t.Errorf("Drawdown over two cycles = %d, want 2200 (the peak of 200 falls to -2000)", got)
	}
}

// TestDrawdown_NoHoleIsZero checks that a run that never fell, or fell and
// never from a peak above its start, gives 0.
//
// Requirement: FR-SAV-006
func TestDrawdown_NoHoleIsZero(t *testing.T) {
	t.Parallel()
	steady := make([]welfare.Record, 10)
	for i := range steady {
		steady[i] = welfare.Record{Income: 160, Cost: 100}
	}
	if got := welfare.Drawdown(steady); got != 0 {
		t.Errorf("Drawdown of a steady surplus = %d, want 0", got)
	}
	flat := []welfare.Record{{Income: 100, Cost: 100}, {Income: 100, Cost: 100}}
	if got := welfare.Drawdown(flat); got != 0 {
		t.Errorf("Drawdown of a flat run = %d, want 0", got)
	}
}

// TestDrawdown_EmptyRing checks that no records give 0, from a slice and from a
// ring that holds nothing.
//
// Requirement: FR-SAV-006
func TestDrawdown_EmptyRing(t *testing.T) {
	t.Parallel()
	if got := welfare.Drawdown(nil); got != 0 {
		t.Errorf("Drawdown(nil) = %d, want 0", got)
	}
	r, err := welfare.NewRing(30, 3_650)
	if err != nil {
		t.Fatal(err)
	}
	got, scratch := welfare.DrawdownRing(&r, nil)
	if got != 0 || len(scratch) != 0 {
		t.Errorf("DrawdownRing of an empty ring = %d with %d records, want 0 and none", got, len(scratch))
	}
}

// TestDrawdown_RingMatchesSlice checks that the ring form reads the records
// oldest first through the caller's scratch, and reuses it.
//
// Requirement: FR-SAV-006, FR-SAV-008
func TestDrawdown_RingMatchesSlice(t *testing.T) {
	t.Parallel()
	r, err := welfare.NewRing(40, 3_650)
	if err != nil {
		t.Fatal(err)
	}
	for _, rec := range append(cycle(), cycle()...) {
		r.Push(rec)
	}
	got, scratch := welfare.DrawdownRing(&r, make([]welfare.Record, 0, 40))
	if got != 2_200 {
		t.Errorf("DrawdownRing = %d, want 2200", got)
	}
	if cap(scratch) != 40 {
		t.Errorf("scratch capacity %d, want the caller's 40 reused", cap(scratch))
	}
}

// TestDrawdown_Saturates checks that a window whose net leaves int64 gives the
// extreme and no wrap (N-491).
//
// Requirement: FR-SAV-006, FR-SAV-043
func TestDrawdown_Saturates(t *testing.T) {
	t.Parallel()
	recs := []welfare.Record{
		{Income: -math.MaxInt64, Cost: math.MaxInt64},
		{Income: -math.MaxInt64, Cost: math.MaxInt64},
	}
	if got := welfare.Drawdown(recs); got != math.MaxInt64 {
		t.Errorf("Drawdown beyond int64 = %d, want %d", got, int64(math.MaxInt64))
	}
	up := []welfare.Record{{Income: math.MaxInt64}, {Income: math.MaxInt64}, {Cost: 5}}
	if got := welfare.Drawdown(up); got != 5 {
		t.Errorf("Drawdown after a saturated peak = %d, want 5", got)
	}
}

// TestDrawdown_WrappedRing checks that the ring form reads a ring whose oldest
// record is not at slot 0: 12 pushes into a window of 5 leave start at 12 mod
// 5 = 2, so AppendRecords must read slots 2, 3, 4, 0, 1 in that order.
//
// The five newest nets are -60, +20, -10, -5, +15, so the running sum is -60,
// -40, -50, -55, -40, the peak stays at 0 and the drawdown is 60. The seven
// dropped records (+10, -20, -30, +5, -40, +100, -50) would have given 110
// (peak 25, then -85), so a ring that kept them, or read in slot order, fails.
//
// Requirement: FR-SAV-006, FR-SAV-008
// Guard: the ring is the production input of the drawdown and wraps every
// window after the first; a read in slot order would give a different number
// that no test of an unwrapped ring would notice.
func TestDrawdown_WrappedRing(t *testing.T) {
	t.Parallel()
	nets := []int64{10, -20, -30, 5, -40, 100, -50, -60, 20, -10, -5, 15}
	r, err := welfare.NewRing(5, 3_650)
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range nets {
		r.Push(welfare.Record{Income: n})
	}
	got, _ := welfare.DrawdownRing(&r, nil)
	if got != 60 {
		t.Errorf("DrawdownRing over a wrapped ring = %d, want 60", got)
	}
	kept := make([]welfare.Record, 0, 5)
	for _, n := range nets[len(nets)-5:] {
		kept = append(kept, welfare.Record{Income: n})
	}
	if want := welfare.Drawdown(kept); got != want {
		t.Errorf("DrawdownRing = %d, Drawdown of the newest five = %d, want equal", got, want)
	}
}

// TestDrawdown_RingAtClamp checks the drawdown of a ring whose records the
// ring clamped to plus or minus MaxInt64 / maxWindow, the largest values the
// production path can hand to Drawdown (N-491).
//
// With a window of 3 and a bound of 3 the clamp c is 3 074 457 345 618 258 602.
// A net of -c three times is -3c = -(MaxInt64 - 1), an exact drawdown with no
// saturation. A net of -2c twice is -4c, below MinInt64, so the sum and the
// drawdown saturate at MaxInt64.
//
// Requirement: FR-SAV-006, FR-SAV-043
func TestDrawdown_RingAtClamp(t *testing.T) {
	t.Parallel()
	const c = math.MaxInt64 / 3
	tests := []struct {
		name string
		rec  welfare.Record
		n    int
		want int64
	}{
		{"three nets of minus c is exact", welfare.Record{Income: math.MinInt64}, 3, 3 * c},
		{"two nets of minus 2c saturate", welfare.Record{Income: math.MinInt64, Cost: math.MaxInt64}, 2, math.MaxInt64},
		{"nets of plus 2c never fall", welfare.Record{Income: math.MaxInt64, Cost: math.MinInt64}, 3, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r, err := welfare.NewRing(3, 3)
			if err != nil {
				t.Fatal(err)
			}
			for i := 0; i < tt.n; i++ {
				if !r.Push(tt.rec) {
					t.Fatalf("Push(%+v) clamped = false, want true", tt.rec)
				}
			}
			if got, _ := welfare.DrawdownRing(&r, nil); got != tt.want {
				t.Errorf("DrawdownRing(%d clamped records of %+v) = %d, want %d", tt.n, tt.rec, got, tt.want)
			}
		})
	}
}

// TestDrawdown_Boundaries checks the smallest and largest valid inputs: one
// record, a fall of exactly MaxInt64 (no saturation), and a net of MinInt64,
// whose fall of 2^63 does not fit and saturates.
//
// Requirement: FR-SAV-006, FR-SAV-043
func TestDrawdown_Boundaries(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		recs []welfare.Record
		want int64
	}{
		{"one rising record", []welfare.Record{{Income: 7}}, 0},
		{"one falling record", []welfare.Record{{Cost: 7}}, 7},
		{"fall of exactly MaxInt64", []welfare.Record{{Cost: math.MaxInt64}}, math.MaxInt64},
		{"net MinInt64 saturates", []welfare.Record{{Income: math.MinInt64}}, math.MaxInt64},
		// Running sum 5 then -5: the peak is 5, so the fall is 5 - (-5) = 10.
		{"fall is measured from the peak", []welfare.Record{{Income: 5}, {Cost: 10}}, 10},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := welfare.Drawdown(tt.recs); got != tt.want {
				t.Errorf("Drawdown(%+v) = %d, want %d", tt.recs, got, tt.want)
			}
		})
	}
}

// FuzzDrawdown checks that the drawdown is never negative, never panics, and
// equals the exact big-integer drawdown while no sum leaves int64.
//
// Requirement: NFR-SAV-003, FR-SAV-006
func FuzzDrawdown(f *testing.F) {
	f.Add([]byte{0, 1, 2, 3, 4, 5, 6, 7})
	f.Add([]byte{255, 0, 255, 0, 128, 128})
	f.Fuzz(func(t *testing.T, data []byte) {
		recs := make([]welfare.Record, 0, len(data)/2)
		for i := 0; i+1 < len(data); i += 2 {
			recs = append(recs, welfare.Record{Income: int64(data[i]) << 40, Cost: int64(data[i+1]) << 40})
		}
		got := welfare.Drawdown(recs)
		if got < 0 {
			t.Fatalf("Drawdown = %d, want at least 0", got)
		}
		cum, peak, worst := new(big.Int), new(big.Int), new(big.Int)
		for _, r := range recs {
			cum.Add(cum, big.NewInt(r.Income-r.Cost))
			if cum.Cmp(peak) > 0 {
				peak.Set(cum)
			}
			if d := new(big.Int).Sub(peak, cum); d.Cmp(worst) > 0 {
				worst.Set(d)
			}
		}
		if worst.Cmp(big.NewInt(math.MaxInt64)) <= 0 && got != worst.Int64() {
			t.Fatalf("Drawdown = %d, want %s", got, worst)
		}
	})
}

// BenchmarkDrawdown measures the drawdown at the longest window, which must be
// O(n) and allocate nothing (NFR-SAV-004).
func BenchmarkDrawdown(b *testing.B) {
	b.ReportAllocs()
	recs := make([]welfare.Record, 3_650)
	for i := range recs {
		recs[i] = welfare.Record{Income: int64(100 + i%7*40), Cost: 120}
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if welfare.Drawdown(recs) < 0 {
			b.Fatal("negative drawdown")
		}
	}
}
