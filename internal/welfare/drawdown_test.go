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
