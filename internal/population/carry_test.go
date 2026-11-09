package population_test

import (
	"math/bits"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/ordering"
	"github.com/Whealinghold/economy-sim/internal/population"
)

// TestStep_Carry verifies the worked examples of spec-population.md 12.1: an
// agent's death carry and a role's birth carry, both with D = 3 650 000.
//
// Requirement: FR-POP-008, FR-POP-009, FR-POP-011, FR-POP-012
func TestStep_Carry(t *testing.T) {
	t.Parallel()
	const d = 3_650_000
	t.Run("agent 17 first death at tick 334", func(t *testing.T) {
		t.Parallel()
		// offset(17) = ((17 * 2 654 435 769) mod 2^32) * D / 2^32 = 1 849 008.
		carry := population.Offset(17, d)
		if carry != 1_849_008 {
			t.Fatalf("Offset(17, %d) = %d, want 1849008", d, carry)
		}
		// 36 exposed households at death rate 150: 5 400 a tick.
		num, ok := population.Term(36, 150)
		if !ok || num != 5_400 {
			t.Fatalf("Term(36, 150) = %d, %v, want 5400", num, ok)
		}
		for tick := 1; tick <= 334; tick++ {
			events, next, ok := population.Step(carry, num, d)
			if !ok {
				t.Fatalf("Step at tick %d not ok", tick)
			}
			switch {
			case tick == 1 && (events != 0 || next != 1_854_408):
				t.Errorf("tick 1: events %d carry %d, want 0 and 1854408", events, next)
			case tick == 2 && next != 1_859_808:
				t.Errorf("tick 2: carry %d, want 1859808", next)
			case tick == 3 && next != 1_865_208:
				t.Errorf("tick 3: carry %d, want 1865208", next)
			case tick < 334 && events != 0:
				t.Fatalf("tick %d: %d events before tick 334", tick, events)
			case tick == 334 && (events != 1 || next != 2_608):
				// 1 849 008 + 334 * 5 400 = 3 652 608 = D + 2 608.
				t.Errorf("tick 334: events %d carry %d, want 1 and 2608", events, next)
			}
			carry = next
		}
	})
	t.Run("role of 1000 households at 250", func(t *testing.T) {
		t.Parallel()
		num, _ := population.Term(1_000, 250)
		var carry, born int64
		for tick := 1; tick <= 15; tick++ {
			var events int64
			events, carry, _ = population.Step(carry, num, d)
			born += events
		}
		// 15 * 250 000 = 3 750 000 = D + 100 000.
		if born != 1 || carry != 100_000 {
			t.Errorf("after 15 ticks: %d births, carry %d, want 1 and 100000", born, carry)
		}
	})
	t.Run("numerator 0 keeps the carry", func(t *testing.T) {
		t.Parallel()
		if e, n, ok := population.Step(1_234, 0, d); !ok || e != 0 || n != 1_234 {
			t.Errorf("Step(1234, 0, %d) = %d, %d, %v, want 0, 1234, true", d, e, n, ok)
		}
	})
}

// TestStep_CumulativeProperty verifies that over 10 000 ticks of changing
// exposure and rate the events equal floor((carry0 + sum) / D) less
// floor(carry0 / D), whatever the path.
//
// Requirement: FR-POP-012, FR-SCL-012
func TestStep_CumulativeProperty(t *testing.T) {
	t.Parallel()
	const d = 6_400_000
	carry0 := population.Offset(9, d)
	carry, sum, events := carry0, int64(0), int64(0)
	for tick := int64(0); tick < 10_000; tick++ {
		exposed := 1 + tick*7%400
		rate := (tick * 37) % 3_001
		num, _ := population.Term(exposed, rate)
		var e int64
		var ok bool
		e, carry, ok = population.Step(carry, num, d)
		if !ok || carry < 0 || carry >= d {
			t.Fatalf("tick %d: ok %v, carry %d outside [0, %d)", tick, ok, carry, d)
		}
		sum += num
		events += e
	}
	if want := (carry0+sum)/d - carry0/d; events != want {
		t.Errorf("events over 10000 ticks = %d, want %d", events, want)
	}
}

// TestStep_Refuses verifies Step reports an invalid denominator, a negative
// input and an overflow instead of wrapping.
//
// Guard: a wrapped carry would pass a negative remainder to the next tick.
func TestStep_Refuses(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                  string
		carry, num, denom, wh int64
	}{
		{"denominator 0", 1, 1, 0, 0},
		{"negative carry", -1, 1, 10, 0},
		{"negative numerator", 1, -1, 10, 0},
		{"overflow", 1<<63 - 1, 1, 10, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if _, _, ok := population.Step(tt.carry, tt.num, tt.denom); ok {
				t.Errorf("Step(%d, %d, %d) ok, want not ok", tt.carry, tt.num, tt.denom)
			}
		})
	}
	if _, ok := population.Term(1<<62, 4); ok {
		t.Errorf("Term(2^62, 4) ok, want overflow")
	}
}

// TestOffset_Range verifies the start of a death carry is in [0, D) and equals
// Weyl32(id) * D >> 32 through the 128-bit product.
//
// Requirement: FR-POP-010
func TestOffset_Range(t *testing.T) {
	t.Parallel()
	for _, d := range []int64{3_650_000, 6_400_000, 1} {
		for id := int64(0); id <= 100_000; id++ {
			got := population.Offset(id, d)
			hi, lo := bits.Mul64(uint64(ordering.Weyl32(id)), uint64(d))
			if want := int64(hi<<32 | lo>>32); got != want || got < 0 || got >= d {
				t.Fatalf("Offset(%d, %d) = %d, want %d in [0, %d)", id, d, got, want, d)
			}
		}
	}
	if got := population.Offset(5, 0); got != 0 {
		t.Errorf("Offset(5, 0) = %d, want 0", got)
	}
}

// FuzzStep verifies Step never panics, that an accepted step leaves a
// remainder in [0, denom), and that ok is false only for an invalid input or an
// overflow.
//
// Requirement: FR-POP-008, FR-POP-012
func FuzzStep(f *testing.F) {
	f.Add(int64(1_849_008), int64(5_400), int64(3_650_000))
	f.Add(int64(1<<63-1), int64(1), int64(7))
	f.Fuzz(func(t *testing.T, carry, num, denom int64) {
		events, next, ok := population.Step(carry, num, denom)
		valid := denom >= 1 && carry >= 0 && num >= 0
		if !valid {
			if ok {
				t.Errorf("Step(%d, %d, %d) ok on an invalid input", carry, num, denom)
			}
			return
		}
		if !ok {
			if carry+num >= 0 {
				t.Errorf("Step(%d, %d, %d) not ok without overflow", carry, num, denom)
			}
			return
		}
		if next < 0 || next >= denom || events < 0 || events*denom+next != carry+num {
			t.Errorf("Step(%d, %d, %d) = %d, %d: not a division", carry, num, denom, events, next)
		}
	})
}
