package economysim

import (
	"math"
	"testing"
)

// worldGoods is the cart of spec-goods.md 3.7: ingot (mass 5, bulk 1, id 0),
// fleece (mass 1, bulk 4, id 1) and a service (mass 0, bulk 0, id 2).
func worldGoods() *Config {
	return &Config{Commodities: []Commodity{
		{ID: "ingot", Mass: 5, Bulk: 1},
		{ID: "fleece", Mass: 1, Bulk: 4},
		{ID: "service"},
	}}
}

const (
	cartMass   = 100
	cartBulk   = 40
	cartFactor = 25_000
)

// TestLoadFits_WorkedExample verifies the cart of spec-goods.md 3.7: Cm 100,
// Cb 40, BulkFactor 25 000, boundaries on each dimension and the mix.
//
// Requirement: FR-GDS-005
func TestLoadFits_WorkedExample(t *testing.T) {
	t.Parallel()
	c := worldGoods()
	tests := []struct {
		name     string
		qty      []Quantity
		mass     int64
		bulk     int64
		wantFits bool
	}{
		// 20 ingots: mass 5*20 = 100 (exactly the capacity), bulk 20.
		{"20 ingots fit", []Quantity{20}, 100, 20, true},
		// 21 ingots: mass 105 > 100.
		{"21st ingot breaks mass", []Quantity{21}, 105, 21, false},
		// 10 fleeces: mass 10, bulk 4*10 = 40 (exactly the capacity).
		{"10 fleeces fit", []Quantity{0, 10}, 10, 40, true},
		// 11 fleeces: bulk 44 > 40.
		{"11th fleece breaks bulk", []Quantity{0, 11}, 11, 44, false},
		// 16 ingots, 6 fleeces: mass 80+6 = 86, bulk 16+24 = 40.
		{"mix fits", []Quantity{16, 6}, 86, 40, true},
		{"empty load", nil, 0, 0, true},
		// A service has no mass and no bulk: adding any of it changes nothing.
		{"service adds nothing", []Quantity{20, 0, 1_000_000_000}, 100, 20, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mass, bulk, ok := load(c, tt.qty)
			if !ok || mass != tt.mass || bulk != tt.bulk {
				t.Fatalf("load(%v) = (%d, %d, %v), want (%d, %d, true)", tt.qty, mass, bulk, ok, tt.mass, tt.bulk)
			}
			if got := fits(mass, bulk, cartMass, cartBulk); got != tt.wantFits {
				t.Errorf("fits(%d, %d, %d, %d) = %v, want %v", mass, bulk, cartMass, cartBulk, got, tt.wantFits)
			}
		})
	}
}

// TestFits_Boundaries verifies each dimension alone: a load exactly at the
// capacity fits, one over on either dimension does not, and a zero capacity
// holds only an empty load.
//
// Requirement: FR-GDS-005
func TestFits_Boundaries(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                       string
		mass, bulk, capMass, capBk int64
		want                       bool
	}{
		{"both exactly met", 10, 20, 10, 20, true},
		{"mass one over", 11, 20, 10, 20, false},
		{"bulk one over", 10, 21, 10, 20, false},
		{"empty in zero capacity", 0, 0, 0, 0, true},
		{"one unit in zero capacity", 1, 0, 0, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := fits(tt.mass, tt.bulk, tt.capMass, tt.capBk); got != tt.want {
				t.Errorf("fits(%d, %d, %d, %d) = %v, want %v", tt.mass, tt.bulk, tt.capMass, tt.capBk, got, tt.want)
			}
		})
	}
}

// TestLoad_OverflowIsRefused verifies that values out of the bounds of
// bounds.go, which the primitives take as int64, give ok false and no wrapped
// number: a product, the mass sum and the bulk sum each overflow.
//
// Requirement: FR-GDS-005
func TestLoad_OverflowIsRefused(t *testing.T) {
	t.Parallel()
	// Mass and bulk near half of MaxInt64 so that one unit fits and the sums
	// of two units, or a product of 3, do not.
	const half = math.MaxInt64 / 2
	big := &Config{Commodities: []Commodity{
		{ID: "a", Mass: half, Bulk: 1},
		{ID: "b", Mass: 1, Bulk: half},
		{ID: "c", Mass: half, Bulk: half},
	}}
	tests := []struct {
		name string
		qty  []Quantity
	}{
		{"mass product overflows", []Quantity{3}},
		// 2 of a: mass 2*half = MaxInt64-1 fits; one more of c adds half.
		{"mass sum overflows", []Quantity{2, 0, 1}},
		{"bulk product overflows", []Quantity{0, 3}},
		// 2 of b: bulk MaxInt64-1 fits; one more of c adds half.
		{"bulk sum overflows", []Quantity{0, 2, 1}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if mass, bulk, ok := load(big, tt.qty); ok || mass != 0 || bulk != 0 {
				t.Errorf("load(%v) = (%d, %d, %v), want (0, 0, false)", tt.qty, mass, bulk, ok)
			}
		})
	}
}

// TestLoad_ForeignQuantityPanics verifies that a quantity list longer than the
// commodity table is refused as a programmer error with the invariant message.
//
// Guard: load indexes the commodities by position; without the check a
// longer list would be an index panic with no hint of the cause.
func TestLoad_ForeignQuantityPanics(t *testing.T) {
	t.Parallel()
	defer func() {
		if recover() == nil {
			t.Errorf("load with 4 quantities over 3 commodities did not panic")
		}
	}()
	load(worldGoods(), []Quantity{1, 1, 1, 1})
}

// TestChargeable_WorkedExample verifies the charges and the capacity of
// spec-goods.md 3.7 and the mix that is over-charged.
//
// Requirement: FR-GDS-006, FR-GDS-007
func TestChargeable_WorkedExample(t *testing.T) {
	t.Parallel()
	c := worldGoods()
	capacity, ok := chargeableCapacity(cartMass, cartBulk, cartFactor)
	// min(100, floor(40 * 25 000 / 10 000)) = min(100, 100) = 100.
	if !ok || capacity != 100 {
		t.Fatalf("chargeableCapacity(100, 40, 25000) = (%d, %v), want (100, true)", capacity, ok)
	}
	tests := []struct {
		name string
		id   CommodityID
		qty  Quantity
		want int64
	}{
		// ingot: max(5, ceil(1 * 25 000 / 10 000) = 3) = 5; 20 * 5 = 100.
		{"20 ingots", 0, 20, 100},
		// fleece: max(1, ceil(4 * 25 000 / 10 000) = 10) = 10; 10 * 10 = 100.
		{"10 fleeces", 1, 10, 100},
		// 16 ingots 80 + 6 fleeces 60 = 140 (over-charged mix).
		{"16 ingots", 0, 16, 80},
		{"6 fleeces", 1, 6, 60},
		{"service", 2, 5, 0},
		{"no units", 0, 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := chargeableSize(c, tt.id, tt.qty, cartFactor)
			if !ok || got != tt.want {
				t.Errorf("chargeableSize(%d, %d, %d) = (%d, %v), want (%d, true)", tt.id, tt.qty, cartFactor, got, ok, tt.want)
			}
		})
	}
}

// TestChargeable_Rounding verifies ceil on a unit's size and floor on the
// capacity, with cases where truncation would differ.
//
// Requirement: FR-GDS-006, FR-GDS-007
func TestChargeable_Rounding(t *testing.T) {
	t.Parallel()
	// bulk 3 at factor 3 333: 3 * 3 333 / 10 000 = 0.9999, ceil 1, trunc 0.
	c := &Config{Commodities: []Commodity{{ID: "x", Mass: 0, Bulk: 3}}}
	if got, ok := chargeableSize(c, 0, 1, 3_333); !ok || got != 1 {
		t.Errorf("chargeableSize(bulk 3, factor 3333) = (%d, %v), want (1, true): ceil of 0.9999", got, ok)
	}
	// Cb 3 at factor 3 333: floor(0.9999) = 0, so min(50, 0) = 0.
	if got, ok := chargeableCapacity(50, 3, 3_333); !ok || got != 0 {
		t.Errorf("chargeableCapacity(50, 3, 3333) = (%d, %v), want (0, true): floor of 0.9999", got, ok)
	}
	// The mass binds: Cm 7, Cb 40 at 10 000 gives min(7, 40) = 7.
	if got, ok := chargeableCapacity(7, 40, wholeBP); !ok || got != 7 {
		t.Errorf("chargeableCapacity(7, 40, 10000) = (%d, %v), want (7, true)", got, ok)
	}
}

// TestChargeable_FactorBoundsAndOverflow verifies BulkFactor 1 and
// MaxBasisPoints, and that out-of-bound inputs give ok false.
//
// Requirement: FR-GDS-006, FR-GDS-007
func TestChargeable_FactorBoundsAndOverflow(t *testing.T) {
	t.Parallel()
	c := &Config{Commodities: []Commodity{{ID: "x", Mass: 1, Bulk: 1_000_000}}}
	// factor 1: ceil(1 000 000 * 1 / 10 000) = 100.
	if got, ok := chargeableSize(c, 0, 1, 1); !ok || got != 100 {
		t.Errorf("chargeableSize(factor 1) = (%d, %v), want (100, true)", got, ok)
	}
	// factor MaxBasisPoints: 1 000 000 * 1 000 000 / 10 000 = 100 000 000.
	if got, ok := chargeableSize(c, 0, 2, MaxBasisPoints); !ok || got != 200_000_000 {
		t.Errorf("chargeableSize(factor max) = (%d, %v), want (200000000, true)", got, ok)
	}
	if got, ok := chargeableCapacity(math.MaxInt64, 1_000_000, MaxBasisPoints); !ok || got != 100_000_000 {
		t.Errorf("chargeableCapacity(max, 1e6, max) = (%d, %v), want (100000000, true)", got, ok)
	}
	o := &Config{Commodities: []Commodity{{ID: "o", Mass: math.MaxInt64, Bulk: 1}}}
	if got, ok := chargeableSize(o, 0, 2, wholeBP); ok {
		t.Errorf("chargeableSize(mass MaxInt64, qty 2) = (%d, true), want ok false", got)
	}
	// bulk MaxInt64 times 20 000 over 10 000 is 2 * MaxInt64: no int64.
	h := &Config{Commodities: []Commodity{{ID: "h", Mass: 1, Bulk: math.MaxInt64}}}
	if got, ok := chargeableSize(h, 0, 1, 2*wholeBP); ok {
		t.Errorf("chargeableSize(bulk MaxInt64, factor 20000) = (%d, true), want ok false", got)
	}
	if got, ok := chargeableCapacity(10, math.MaxInt64, math.MaxInt64); ok {
		t.Errorf("chargeableCapacity(10, MaxInt64, MaxInt64) = (%d, true), want ok false", got)
	}
}

// TestChargeable_SingleCommodityWithinCapacityFits verifies, over seeds 1 to
// 200 of a small deterministic generator, that any load of one commodity whose
// chargeable size is within the chargeable capacity fits the container.
//
// Requirement: FR-GDS-007
//
// Guard: the conservative bound of spec-goods.md 3.4 is what a later per-unit
// price would rely on; a rounding change in either function could make a
// priced load that does not fit.
func TestChargeable_SingleCommodityWithinCapacityFits(t *testing.T) {
	t.Parallel()
	next := func(s *uint64) int64 { // splitmix64 step
		*s += 0x9e3779b97f4a7c15
		z := *s
		z = (z ^ z>>30) * 0xbf58476d1ce4e5b9
		z = (z ^ z>>27) * 0x94d049bb133111eb
		return int64((z ^ z>>31) >> 40) // below 2^24
	}
	for seed := uint64(1); seed <= 200; seed++ {
		s := seed
		mass, bulk := next(&s)%50, next(&s)%50
		capMass, capBulk := next(&s)%2_000, next(&s)%2_000
		factor := 1 + next(&s)%40_000
		c := &Config{Commodities: []Commodity{{ID: "x", Mass: mass, Bulk: bulk}}}
		capacity, ok := chargeableCapacity(capMass, capBulk, factor)
		if !ok {
			t.Fatalf("seed %d: chargeableCapacity overflowed", seed)
		}
		for q := int64(0); q <= 400; q++ {
			size, ok := chargeableSize(c, 0, Quantity(q), factor)
			if !ok {
				t.Fatalf("seed %d: chargeableSize overflowed", seed)
			}
			if size > capacity {
				break
			}
			m, b, ok := load(c, []Quantity{Quantity(q)})
			if !ok || !fits(m, b, capMass, capBulk) {
				t.Fatalf("seed %d: %d units (mass %d bulk %d factor %d) have chargeable size %d <= capacity %d but do not fit (%d, %d)",
					seed, q, mass, bulk, factor, size, capacity, capMass, capBulk)
			}
		}
	}
}

// TestLoadAndChargeable_AllocateNothing verifies that the primitives allocate
// nothing on a read (NFR-GDS-001).
//
// Requirement: NFR-GDS-001
func TestLoadAndChargeable_AllocateNothing(t *testing.T) {
	c := worldGoods()
	qty := []Quantity{16, 6, 3}
	n := testing.AllocsPerRun(100, func() {
		load(c, qty)
		if !fits(86, 40, 100, 40) {
			t.Errorf("fits(86, 40, 100, 40) = false, want true")
		}
		chargeableSize(c, 1, 6, cartFactor)
		chargeableCapacity(100, 40, cartFactor)
	})
	if n != 0 {
		t.Errorf("load, fits, chargeableSize and chargeableCapacity allocated %v times, want 0", n)
	}
}

// TestLoad_ZeroMassOrZeroBulkCommodity verifies that a commodity with zero mass
// but some bulk, and one with zero bulk but some mass, add only their non-zero
// dimension to a load (the lower bound of Mass and Bulk is 0, not 1).
//
// Requirement: FR-GDS-004
func TestLoad_ZeroMassOrZeroBulkCommodity(t *testing.T) {
	t.Parallel()
	c := &Config{Commodities: []Commodity{
		{ID: "gas", Mass: 0, Bulk: 3},
		{ID: "ore", Mass: 4, Bulk: 0},
	}}
	// 5 gas: mass 0, bulk 15. 2 ore: mass 8, bulk 0.
	mass, bulk, ok := load(c, []Quantity{5, 2})
	if !ok || mass != 8 || bulk != 15 {
		t.Errorf("load(5 gas, 2 ore) = (%d, %d, %v), want (8, 15, true)", mass, bulk, ok)
	}
	// A capacity of exactly that load fits; one unit less of bulk does not.
	if !fits(mass, bulk, 8, 15) || fits(mass, bulk, 8, 14) {
		t.Errorf("fits(8, 15, 8, 15) and fits(8, 15, 8, 14) = %v, %v, want true, false", fits(mass, bulk, 8, 15), fits(mass, bulk, 8, 14))
	}
}
