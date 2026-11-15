package tenure_test

import (
	"math"
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/tenure"
)

// sum adds a slice of units.
func sum(v []int64) int64 {
	var s int64
	for _, x := range v {
		s += x
	}
	return s
}

// TestApportion_SumsToStock verifies the example of spec-land.md 4.2.2: fifty
// holders of 100 basis points of a stock of 80 and a commons of 5 000. Each
// holder is owed 80 * 100 / 10 000 = 0.8 unit and the commons 40 exactly, so
// the 40 units left go to forty holders and the entitlements sum to 80.
//
// Requirement: FR-LND-007
func TestApportion_SumsToStock(t *testing.T) {
	t.Parallel()
	parts := []tenure.Part{{Ref: 50, Share: 5000, Commons: true}}
	for i := range 50 {
		parts = append(parts, tenure.Part{Ref: i, ID: int64(100 + i), Share: 100, Key: uint64(i * 7 % 50)})
	}
	out := make([]int64, 51)
	if !tenure.Apportion(80, parts, out) {
		t.Fatalf("Apportion(80, 50 holders) not ok")
	}
	ones := 0
	for _, e := range out[:50] {
		if e != 0 && e != 1 {
			t.Errorf("Apportion(80, 50 holders) gave a holder %d, want 0 or 1", e)
		}
		ones += int(e)
	}
	if ones != 40 || out[50] != 40 || sum(out) != 80 {
		t.Errorf("Apportion(80, 50 holders): %d holders got a unit, commons %d, sum %d, want 40, 40, 80", ones, out[50], sum(out))
	}
}

// TestApportion_LargestRemainder_KeyThenIDThenCommonsLast verifies the order of
// the leftover units: a larger remainder first, then the smaller key, then the
// smaller ID, and the commons after every agent whatever the key. Stock 3 over
// four equal weights of 2 500 owes each 0.75, so one unit is left for each
// of three of the four parts, and the part left out is the last in the order.
//
// Requirement: FR-LND-007
func TestApportion_LargestRemainder_KeyThenIDThenCommonsLast(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		stock int64
		parts []tenure.Part
		want  []int64
	}{
		{"smaller key first, then ID", 3, []tenure.Part{
			{Ref: 0, ID: 10, Share: 2500, Key: 5}, {Ref: 1, ID: 11, Share: 2500, Key: 5},
			{Ref: 2, ID: 12, Share: 2500, Key: 9}, {Ref: 3, ID: 13, Share: 2500, Key: 1},
		}, []int64{1, 1, 0, 1}},
		{"commons after every agent", 3, []tenure.Part{
			{Ref: 0, ID: 10, Share: 2500, Key: 99}, {Ref: 1, ID: 11, Share: 2500, Key: 98},
			{Ref: 2, ID: 12, Share: 2500, Key: 97}, {Ref: 3, Share: 2500, Key: 0, Commons: true},
		}, []int64{1, 1, 1, 0}},
		// 3 * 3000 / 10 000 = 0 rem 9000 twice and 3 * 4000 / 10 000 = 1 rem 2000: the two
		// units left go to the two remainders of 9 000, whatever the keys (9, 8, 7).
		{"larger remainder first", 3, []tenure.Part{
			{Ref: 0, ID: 1, Share: 3000, Key: 9}, {Ref: 1, ID: 2, Share: 3000, Key: 8},
			{Ref: 2, ID: 3, Share: 4000, Key: 7},
		}, []int64{1, 1, 1}},
		// 5 * 5000 / 10 000 = 2 rem 5000 twice; one unit left, to the smaller key.
		{"one unit to the smaller key", 5, []tenure.Part{
			{Ref: 0, ID: 1, Share: 5000, Key: 9}, {Ref: 1, ID: 2, Share: 5000, Key: 3},
		}, []int64{2, 3}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			out := make([]int64, len(tt.parts))
			if !tenure.Apportion(tt.stock, slices.Clone(tt.parts), out) {
				t.Fatalf("Apportion(%d, %+v) not ok", tt.stock, tt.parts)
			}
			if !slices.Equal(out, tt.want) {
				t.Errorf("Apportion(%d, %+v) = %v, want %v", tt.stock, tt.parts, out, tt.want)
			}
		})
	}
}

// TestApportion_Product128 verifies the product stock * share is held in 128
// bits: a stock of MaxInt64 over a share of 10 000 of 10 000 gives the stock, and
// over two halves gives the halves, which an int64 product would wrap.
//
// Requirement: FR-LND-007
func TestApportion_Product128(t *testing.T) {
	t.Parallel()
	out := make([]int64, 2)
	parts := []tenure.Part{{Ref: 0, ID: 1, Share: 5000}, {Ref: 1, ID: 2, Share: 5000}}
	if !tenure.Apportion(math.MaxInt64, parts, out) {
		t.Fatalf("Apportion(MaxInt64, two halves) not ok")
	}
	// MaxInt64 = 2^63 - 1 is odd: each half is 2^62 - 1 and the one unit left goes to ID 1.
	if want := []int64{1<<62 - 1 + 1, 1<<62 - 1}; !slices.Equal(out, want) {
		t.Errorf("Apportion(MaxInt64, halves) = %v, want %v", out, want)
	}
}

// TestApportion_ZeroShareIsNoPart verifies a part with a share of 0 gets 0 and
// takes no leftover, and that a stock of 0 gives every part 0.
//
// Requirement: FR-LND-007
func TestApportion_ZeroShareIsNoPart(t *testing.T) {
	t.Parallel()
	out := make([]int64, 3)
	parts := []tenure.Part{{Ref: 0, ID: 1, Share: 0, Key: 0}, {Ref: 1, ID: 2, Share: 3333}, {Ref: 2, ID: 3, Share: 6667}}
	if !tenure.Apportion(1, parts, out) {
		t.Fatalf("Apportion(1, %+v) not ok", parts)
	}
	// 1 * 3333 / 10 000 = 0 rem 3333; 1 * 6667 / 10 000 = 0 rem 6667: the unit goes to the larger remainder.
	if want := []int64{0, 0, 1}; !slices.Equal(out, want) {
		t.Errorf("Apportion(1, %+v) = %v, want %v", parts, out, want)
	}
	if !tenure.Apportion(0, parts, out) || sum(out) != 0 {
		t.Errorf("Apportion(0, %+v) = %v, want all 0", parts, out)
	}
}

// TestApportion_Refusals verifies the inputs that cannot be applied: a negative
// stock or share, a Ref outside out, a sum of weights beyond int64 and a stock
// with no weight, each reporting not ok and leaving 0 in every slot.
//
// Requirement: FR-LND-007
func TestApportion_Refusals(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		stock int64
		parts []tenure.Part
	}{
		{"negative stock", -1, []tenure.Part{{Ref: 0, ID: 1, Share: 1}}},
		{"negative share", 5, []tenure.Part{{Ref: 0, ID: 1, Share: -1}}},
		{"ref outside", 5, []tenure.Part{{Ref: 1, ID: 1, Share: 1}}},
		{"negative ref", 5, []tenure.Part{{Ref: -1, ID: 1, Share: 1}}},
		{"weights overflow", 5, []tenure.Part{{Ref: 0, ID: 1, Share: math.MaxInt64}, {Ref: 0, ID: 2, Share: 1}}},
		{"no weight", 5, []tenure.Part{{Ref: 0, ID: 1, Share: 0}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			out := []int64{7}
			if tenure.Apportion(tt.stock, tt.parts, out) {
				t.Errorf("Apportion(%d, %+v) = ok, want refused", tt.stock, tt.parts)
			}
			if tt.parts[0].Ref == 0 && out[0] != 0 {
				t.Errorf("Apportion(%d, %+v) left %d in a slot it names, want 0", tt.stock, tt.parts, out[0])
			}
		})
	}
}

// TestApportion_IndependentOfListingOrder verifies a shuffled listing gives the
// same entitlements by ID, the order being total.
//
// Requirement: FR-LND-007, NFR-LND-003
func TestApportion_IndependentOfListingOrder(t *testing.T) {
	t.Parallel()
	rng := rand.New(rand.NewPCG(1, 2))
	for range 50 {
		n := 2 + rng.IntN(12)
		base := make([]tenure.Part, n)
		for i := range base {
			base[i] = tenure.Part{Ref: i, ID: int64(i + 1), Share: int64(1 + rng.IntN(2000)), Key: uint64(rng.IntN(4))}
		}
		stock := int64(rng.IntN(500))
		want := make([]int64, n)
		tenure.Apportion(stock, slices.Clone(base), want)
		shuffled := slices.Clone(base)
		rng.Shuffle(n, func(i, j int) { shuffled[i], shuffled[j] = shuffled[j], shuffled[i] })
		got := make([]int64, n)
		tenure.Apportion(stock, shuffled, got)
		if !slices.Equal(got, want) {
			t.Fatalf("Apportion(%d, %+v) = %v, shuffled %v", stock, base, want, got)
		}
	}
}
