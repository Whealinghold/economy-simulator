package transit_test

import (
	"math"
	"math/big"
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/transit"
)

// TestTicks_CeilingAndMinimum verifies k = max(1, ceil(D / v)) at the default
// factor: the table of spec-transport.md 17.1, the asymmetry of 90 against 100
// and the minimum of one tick.
//
// Requirement: FR-TRN-005
func TestTicks_CeilingAndMinimum(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name            string
		distance, speed int64
		want            int64
	}{
		{"foot north to mid", 40, 20, 2},
		{"foot mid to south", 60, 20, 3},
		{"foot north to south, the direct 90", 90, 20, 5}, // 4.5 rounds up
		{"foot south to north, 100 through mid", 100, 20, 5},
		{"wagon north to south", 90, 30, 3},
		{"wagon south to north", 100, 30, 4}, // 3.33 rounds up
		{"exact division", 60, 30, 2},
		{"short hop is one tick", 1, 100, 1},
		{"zero distance is one tick", 0, 5, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := transit.Ticks(tt.distance, tt.speed, 10_000)
			if !ok || got != tt.want {
				t.Errorf("Ticks(%d, %d, 10000) = %d, %v, want %d, true", tt.distance, tt.speed, got, ok, tt.want)
			}
		})
	}
}

// TestTicks_FactorAndOverflow verifies the transit factor scales the speed
// (5 000 halves it), that the product D * 10 000 is held in 128 bits, that the
// largest speed times the largest factor fits, and that every rejected input is
// reported with ok false.
//
// Requirement: FR-TRN-005
func TestTicks_FactorAndOverflow(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                      string
		distance, speed, factorBP int64
		want                      int64
		ok                        bool
	}{
		// 45 * 10000 / (10 * 5000) = 9: half the speed doubles 4.5 to 9.
		{"half factor", 45, 10, 5_000, 9, true},
		{"double factor", 45, 10, 20_000, 3, true}, // 450000/200000 = 2.25 up
		// D * 10 000 = 10^19 > MaxInt64, the quotient 10^19 / 10^13 = 10^6.
		{"128-bit product", 1_000_000_000_000_000, 1_000_000_000, 10_000, 1_000_000, true},
		{"largest speed and factor", 5, 1_000_000_000, 100_000, 1, true},
		{"blocked factor", 10, 5, 0, 0, false},
		{"zero speed", 10, 0, 10_000, 0, false},
		{"negative distance", -1, 5, 10_000, 0, false},
		{"speed times factor overflows", 1, math.MaxInt64, 2, 0, false},
		{"quotient overflows", math.MaxInt64, 1, 1, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := transit.Ticks(tt.distance, tt.speed, tt.factorBP)
			if ok != tt.ok || got != tt.want {
				t.Errorf("Ticks(%d, %d, %d) = %d, %v, want %d, %v", tt.distance, tt.speed, tt.factorBP, got, ok, tt.want, tt.ok)
			}
		})
	}
}

// TestUnitsNeeded_BindingFill verifies n = max(1, ceil(mass / unitMass),
// ceil(bulk / unitBulk)): a capacity of 0 leaves its term out, the binding
// dimension decides, and an empty load still needs one unit.
//
// Requirement: FR-TRN-010
func TestUnitsNeeded_BindingFill(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                     string
		mass, bulk, uMass, uBulk int64
		want                     int64
		ok                       bool
	}{
		// 36/30 = 2 against 40/30 = 2 (both ceil to 2): spec 17.2 reads a unit of
		// (40, 30), where 36 over 40 and 30 over 30 both fit in one.
		{"17.2 fits one unit", 36, 30, 40, 30, 1, true},
		{"bulk binds", 36, 42, 40, 30, 2, true},  // ceil(42/30) = 2
		{"mass binds", 100, 10, 40, 30, 3, true}, // ceil(100/40) = 3
		{"coach: 17.4", 300, 200, 1_200, 900, 1, true},
		{"mass term left out", 1_000, 31, 0, 30, 2, true},
		{"bulk term left out", 41, 1_000, 40, 0, 2, true},
		{"empty load needs one", 0, 0, 40, 30, 1, true},
		{"both capacities zero", 1, 1, 0, 0, 0, false},
		{"negative load", -1, 0, 40, 30, 0, false},
		{"negative capacity", 1, 1, -1, 30, 0, false},
		{"negative bulk", 1, -1, 40, 30, 0, false},
		{"negative bulk capacity", 1, 1, 40, -1, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := transit.UnitsNeeded(tt.mass, tt.bulk, tt.uMass, tt.uBulk)
			if ok != tt.ok || got != tt.want {
				t.Errorf("UnitsNeeded(%d, %d, %d, %d) = %d, %v, want %d, %v", tt.mass, tt.bulk, tt.uMass, tt.uBulk, got, ok, tt.want, tt.ok)
			}
		})
	}
}

// TestFits_PairRule verifies that a load fits only when both sums are within the
// pooled pair: the load (36, 30) of spec-transport.md 17.2 fits one unit of (40,
// 30) with bulk exactly at its limit, (40, 42) does not, and pooling scales the
// capacity with the units.
//
// Requirement: FR-TRN-009
func TestFits_PairRule(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                            string
		mass, bulk, units, uMass, uBulk int64
		fits, ok                        bool
	}{
		{"17.2 fits at the bulk limit", 36, 30, 1, 40, 30, true, true},
		{"mass fits, bulk 42 over 30", 40, 42, 1, 40, 30, false, true},
		{"bulk fits, mass over", 41, 30, 1, 40, 30, false, true},
		{"two units pool", 70, 50, 2, 40, 30, true, true},
		{"no units fits nothing", 1, 1, 0, 40, 30, false, true},
		{"no units fits an empty load", 0, 0, 0, 40, 30, true, true},
		{"mass unlimited", 1_000_000, 30, 1, 0, 30, true, true},
		{"bulk unlimited", 40, 1_000_000, 1, 40, 0, true, true},
		{"both zero", 1, 1, 1, 0, 0, false, false},
		{"pooled mass overflows", 1, 1, math.MaxInt64, 2, 1, false, false},
		{"pooled bulk overflows", 1, 1, math.MaxInt64, 1, 2, false, false},
		{"negative load", -1, 1, 1, 40, 30, false, false},
		{"negative bulk", 1, -1, 1, 40, 30, false, false},
		{"negative units", 1, 1, -1, 40, 30, false, false},
		{"negative mass capacity", 1, 1, 1, -1, 30, false, false},
		{"negative bulk capacity", 1, 1, 1, 40, -1, false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			fits, ok := transit.Fits(tt.mass, tt.bulk, tt.units, tt.uMass, tt.uBulk)
			if fits != tt.fits || ok != tt.ok {
				t.Errorf("Fits(%d, %d, %d, %d, %d) = %v, %v, want %v, %v", tt.mass, tt.bulk, tt.units, tt.uMass, tt.uBulk, fits, ok, tt.fits, tt.ok)
			}
		})
	}
}

// FuzzFitsAgreesWithUnits verifies that, for at least one unit, a load fits the
// pooled capacity of u units exactly when UnitsNeeded is at most u, and that
// neither function panics.
//
// Requirement: FR-TRN-009, FR-TRN-010
func FuzzFitsAgreesWithUnits(f *testing.F) {
	f.Add(int64(36), int64(30), int64(1), int64(40), int64(30))
	f.Add(int64(40), int64(42), int64(1), int64(40), int64(30))
	f.Add(int64(0), int64(0), int64(3), int64(0), int64(5))
	f.Add(int64(1_000_000), int64(7), int64(9), int64(1_000), int64(0))
	f.Fuzz(func(t *testing.T, mass, bulk, units, uMass, uBulk int64) {
		n, nok := transit.UnitsNeeded(mass, bulk, uMass, uBulk)
		fits, fok := transit.Fits(mass, bulk, units, uMass, uBulk)
		if !nok || !fok || units < 1 {
			return
		}
		if fits != (n <= units) {
			t.Fatalf("Fits(%d, %d, %d, %d, %d) = %v but UnitsNeeded = %d", mass, bulk, units, uMass, uBulk, fits, n)
		}
	})
}

// TestHaulPerUnit_Steps verifies s = ceil(distance / span): the cost of carriage
// grows in steps, 6 haul units for 60 over a span of 10 and 7 for 61.
//
// Requirement: FR-TRN-012
func TestHaulPerUnit_Steps(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		distance, span int64
		want           int64
		ok             bool
	}{
		{"exact", 60, 10, 6, true},
		{"one over", 61, 10, 7, true},
		{"one under", 59, 10, 6, true},
		{"zero distance", 0, 10, 0, true},
		{"span of one", 5, 1, 5, true},
		{"zero span", 5, 0, 0, false},
		{"negative span", 5, -1, 0, false},
		{"negative distance", -5, 1, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := transit.HaulPerUnit(tt.distance, tt.span)
			if got != tt.want || ok != tt.ok {
				t.Errorf("HaulPerUnit(%d, %d) = %d, %v, want %d, %v", tt.distance, tt.span, got, ok, tt.want, tt.ok)
			}
		})
	}
}

// TestLabour_Exact verifies the lane labour of spec-transport.md 9.4 against
// hand figures: the mass term binding, the bulk term binding (the larger of the
// two ceilings), a result rounded up, a fill of 0, a term left out, a numerator
// past 64 bits, and the rejected inputs.
//
// Requirement: FR-TRN-041
func TestLabour_Exact(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                                                       string
		mass, bulk, uMass, uBulk, span, distance, duration, output int64
		want                                                       int64
		ok                                                         bool
	}{
		// f = max(3/1200, 2/900) = 3/1200 = 1/400; s = 6; 10000 * 6 * 4 / (400 * 4) = 150.
		{"mass binds", 3, 2, 1_200, 900, 10, 60, 4, 4, 150, true},
		// f = max(1/1200, 3/900) = 1/300; 10000 * 6 * 4 / (300 * 4) = 200.
		{"bulk binds", 1, 3, 1_200, 900, 10, 60, 4, 4, 200, true},
		// f = 7/1200 (bulk unlimited), s = 7, 10000 * 7 * 7 * 1 / (1200 * 5) = 81.67 up to 82.
		{"rounds up", 7, 99, 1_200, 0, 10, 61, 1, 5, 82, true},
		{"bulk only", 1, 3, 0, 900, 10, 60, 4, 4, 200, true}, // f = 3/900 = 1/300
		{"nothing to carry", 0, 0, 1_200, 900, 10, 60, 4, 4, 0, true},
		{"zero duration", 3, 2, 1_200, 900, 10, 60, 0, 4, 0, true},
		{"output of one", 3, 2, 1_200, 900, 10, 60, 4, 1, 600, true}, // 10000 * 6 * 4 / 400 = 600
		// f = 1, s = 10^9, k = 10^9 * 10^4 * 10^4 = 10^17, numerator 10^26 over 10^9 * 10^5.
		{"numerator past 64 bits", 1_000_000_000, 1, 1_000_000_000, 1, 1, 1_000_000_000, 10_000, 100_000, 1_000_000_000_000, true},
		{"both capacities zero", 1, 1, 0, 0, 10, 60, 4, 4, 0, false},
		{"zero output", 1, 1, 1, 1, 10, 60, 4, 0, 0, false},
		{"zero span", 1, 1, 1, 1, 0, 60, 4, 4, 0, false},
		{"negative mass", -1, 1, 1, 1, 10, 60, 4, 4, 0, false},
		{"negative duration", 1, 1, 1, 1, 10, 60, -4, 4, 0, false},
		{"s times duration overflows", 1, 1, 1, 1, 1, math.MaxInt64, math.MaxInt64, 1, 0, false},
		{"times 10 000 overflows", 1, 1, 1, 1, 1, math.MaxInt64 / 2, 1, 1, 0, false},
		{"divisor overflows", 1, 0, math.MaxInt64, 1, 1, 10, 1, math.MaxInt64, 0, false},
		{"quotient past 64 bits", math.MaxInt64, 1, 1, 1, 1, 1 << 40, 1 << 20, 1, 0, false},
		{"quotient past int64", 1 << 40, 1, 1, 1, 1, 1 << 20, 1 << 10, 1, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := transit.Labour(tt.mass, tt.bulk, tt.uMass, tt.uBulk, tt.span, tt.distance, tt.duration, tt.output)
			if got != tt.want || ok != tt.ok {
				t.Errorf("Labour(%d, %d, %d, %d, %d, %d, %d, %d) = %d, %v, want %d, %v",
					tt.mass, tt.bulk, tt.uMass, tt.uBulk, tt.span, tt.distance, tt.duration, tt.output, got, ok, tt.want, tt.ok)
			}
		})
	}
}

// FuzzLabourOracle verifies Labour against a math/big evaluation of the formula
// whenever it reports a result, and that it never panics.
//
// Requirement: FR-TRN-041
func FuzzLabourOracle(f *testing.F) {
	f.Add(int64(3), int64(2), int64(1_200), int64(900), int64(10), int64(60), int64(4), int64(4))
	f.Add(int64(7), int64(99), int64(1_200), int64(0), int64(10), int64(61), int64(1), int64(5))
	f.Fuzz(func(t *testing.T, mass, bulk, uMass, uBulk, span, distance, duration, output int64) {
		got, ok := transit.Labour(mass, bulk, uMass, uBulk, span, distance, duration, output)
		if !ok {
			return
		}
		s := new(big.Int).Div(new(big.Int).Add(big.NewInt(distance), big.NewInt(span-1)), big.NewInt(span))
		// f as the larger of two fractions, compared exactly.
		fn, fd := big.NewInt(mass), big.NewInt(uMass)
		if uMass == 0 || (uBulk > 0 && new(big.Int).Mul(big.NewInt(bulk), big.NewInt(uMass)).Cmp(new(big.Int).Mul(big.NewInt(mass), big.NewInt(uBulk))) > 0) {
			fn, fd = big.NewInt(bulk), big.NewInt(uBulk)
		}
		num := new(big.Int).Mul(fn, s)
		num.Mul(num, big.NewInt(duration)).Mul(num, big.NewInt(10_000))
		den := new(big.Int).Mul(fd, big.NewInt(output))
		want := new(big.Int)
		want.Add(num, den).Sub(want, big.NewInt(1)).Div(want, den)
		if want.Cmp(big.NewInt(got)) != 0 {
			t.Fatalf("Labour(%d, %d, %d, %d, %d, %d, %d, %d) = %d, want %s", mass, bulk, uMass, uBulk, span, distance, duration, output, got, want)
		}
	})
}

// TestFree_LowestFirst verifies that Take returns the lowest open account ID
// first, that Put keeps the order and refuses a duplicate, that Rebuild sorts and
// drops repeats and reuses its storage, and that an empty set takes nothing.
//
// Requirement: FR-TRN-020, FR-TRN-023
func TestFree_LowestFirst(t *testing.T) {
	t.Parallel()
	var f transit.Free
	if id, ok := f.Take(); ok || id != 0 {
		t.Fatalf("Take on an empty set = %d, %v, want 0, false", id, ok)
	}
	f.Rebuild([]int{9, 3, 7, 3})
	if f.Len() != 3 {
		t.Fatalf("Len after Rebuild([9 3 7 3]) = %d, want 3 (the repeat dropped)", f.Len())
	}
	if f.Put(7) {
		t.Errorf("Put(7) of a member = true, want false")
	}
	if !f.Put(5) || !f.Put(1) || !f.Put(12) {
		t.Fatalf("Put of a new ID = false, want true")
	}
	var got []int
	for {
		id, ok := f.Take()
		if !ok {
			break
		}
		got = append(got, id)
	}
	if want := []int{1, 3, 5, 7, 9, 12}; !slices.Equal(got, want) {
		t.Errorf("Take order = %v, want %v", got, want)
	}
	f.Rebuild([]int{4})
	f.Rebuild(nil)
	if f.Len() != 0 {
		t.Errorf("Len after Rebuild(nil) = %d, want 0", f.Len())
	}
}

// TestSchedule_DueInTickThenIDOrder verifies that the entries come out by tick and,
// within a tick, by ascending ID whatever the order they were pushed in, that nothing
// comes out before its tick, and that Reset empties the schedule.
//
// Requirement: FR-TRN-024
func TestSchedule_DueInTickThenIDOrder(t *testing.T) {
	t.Parallel()
	var s transit.Schedule
	if id, ok := s.Due(100); ok || id != 0 {
		t.Fatalf("Due on an empty schedule = %d, %v, want 0, false", id, ok)
	}
	for _, e := range [][2]int64{{5, 9}, {3, 4}, {5, 2}, {7, 1}, {3, 8}, {5, 6}} {
		s.Push(e[0], e[1])
	}
	if id, ok := s.Due(2); ok {
		t.Fatalf("Due(2) = %d, true, want none before tick 3", id)
	}
	var got []int64
	for tick := int64(3); tick <= 7; tick++ {
		for {
			id, ok := s.Due(tick)
			if !ok {
				break
			}
			got = append(got, id)
		}
	}
	// Tick 3: IDs 4 and 8; tick 5: 2, 6 and 9; tick 7: 1.
	if want := []int64{4, 8, 2, 6, 9, 1}; !slices.Equal(got, want) {
		t.Errorf("Due order = %v, want %v", got, want)
	}
	s.Push(1, 1)
	s.Reset()
	if s.Len() != 0 {
		t.Errorf("Len after Reset = %d, want 0", s.Len())
	}
}

// TestSchedule_CostFollowsArrivals verifies NFR-TRN-002 on the structure: a tick
// with nothing due costs one comparison however many shipments are in flight, and the
// cost of the arrivals of a tick grows with their number and the logarithm of what
// is in flight, not with what is in flight.
//
// Requirement: NFR-TRN-002
func TestSchedule_CostFollowsArrivals(t *testing.T) {
	t.Parallel()
	cost := func(inFlight int) (idle, busy int64) {
		var s transit.Schedule
		for i := range inFlight {
			s.Push(1_000+int64(i), int64(i)) // far in the future
		}
		for i := range 3 {
			s.Push(10, int64(inFlight+i)) // three due at tick 10
		}
		before := s.Steps()
		s.Due(9)
		idle = s.Steps() - before
		before = s.Steps()
		for {
			if _, ok := s.Due(10); !ok {
				break
			}
		}
		return idle, s.Steps() - before
	}
	smallIdle, smallBusy := cost(8)
	largeIdle, largeBusy := cost(8_000)
	if smallIdle != 1 || largeIdle != 1 {
		t.Errorf("a tick with no arrival cost %d and %d comparisons for 8 and 8 000 in flight, want 1 and 1", smallIdle, largeIdle)
	}
	// 8 000 is 1 000 times 8 and its logarithm about 4 times: the cost of three
	// arrivals may grow with the logarithm and must not grow with the count.
	if largeBusy > 6*smallBusy {
		t.Errorf("three arrivals cost %d comparisons among 8 000 in flight and %d among 8, want growth by the logarithm (at most 6 times)", largeBusy, smallBusy)
	}
}

// TestSchedule_OrderOfManyEntries verifies the heap against a sort: 300 entries pushed in
// a scrambled order (ticks that repeat, IDs that do not) come out in ascending (tick,
// ID), whatever interleaving of pushes and pops precedes, and an entry due before the
// tick asked for comes out first.
//
// Requirement: FR-TRN-024
func TestSchedule_OrderOfManyEntries(t *testing.T) {
	t.Parallel()
	type pair struct{ at, id int64 }
	var s transit.Schedule
	var all []pair
	x := uint64(88172645463325252) // xorshift: a fixed scramble, not a source of randomness for results
	next := func(n uint64) int64 {
		x ^= x << 13
		x ^= x >> 7
		x ^= x << 17
		return int64(x % n)
	}
	for id := range int64(300) {
		p := pair{at: 1 + next(40), id: id}
		all = append(all, p)
		s.Push(p.at, p.id)
	}
	slices.SortFunc(all, func(a, b pair) int {
		if a.at != b.at {
			return int(a.at - b.at)
		}
		return int(a.id - b.id)
	})
	var got []int64
	for {
		id, ok := s.Due(100)
		if !ok {
			break
		}
		got = append(got, id)
	}
	want := make([]int64, len(all))
	for i, p := range all {
		want[i] = p.id
	}
	if !slices.Equal(got, want) {
		t.Errorf("Due order of 300 entries differs from the sorted order at %d places", countDiff(got, want))
	}
	s.Push(2, 7)
	if id, ok := s.Due(5); !ok || id != 7 {
		t.Errorf("Due(5) with an entry at tick 2 = %d, %v, want 7, true (an entry before the tick comes out)", id, ok)
	}
}

func countDiff(a, b []int64) (n int) {
	for i := range min(len(a), len(b)) {
		if a[i] != b[i] {
			n++
		}
	}
	return n + max(len(a), len(b)) - min(len(a), len(b))
}

// TestSchedule_StepsCountComparisons verifies the counter on hand-counted cases: a first
// Push compares nothing, a Push below its parent compares once and moves up, a Due on an
// empty schedule or with nothing due is one look, and a Due that removes the only entry
// is that look and the two comparisons of the sift (left and right child).
//
// Requirement: NFR-TRN-002
func TestSchedule_StepsCountComparisons(t *testing.T) {
	t.Parallel()
	var s transit.Schedule
	s.Push(5, 1)
	if got := s.Steps(); got != 0 {
		t.Errorf("Steps after the first Push = %d, want 0", got)
	}
	s.Push(3, 2) // one comparison with its parent, then at the root
	if got := s.Steps(); got != 1 {
		t.Errorf("Steps after a Push below its parent = %d, want 1", got)
	}
	s.Push(9, 3) // one comparison with the root, which is lower: stays
	if got := s.Steps(); got != 2 {
		t.Errorf("Steps after a Push above its parent = %d, want 2", got)
	}
	before := s.Steps()
	s.Due(2) // nothing due: one look
	if got := s.Steps() - before; got != 1 {
		t.Errorf("Steps of a Due with nothing due = %d, want 1", got)
	}
	var one transit.Schedule
	one.Push(1, 1)
	before = one.Steps()
	one.Due(1)
	if got := one.Steps() - before; got != 3 { // the look and the sift of an empty heap, 2
		t.Errorf("Steps of the Due of the only entry = %d, want 3", got)
	}
}
