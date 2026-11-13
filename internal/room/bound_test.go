package room_test

import (
	"math"
	"math/rand/v2"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/room"
)

// dims lists the two dimensions of a pair, so a table can be written once.
var dims = [2]func(room.Pair) int64{
	func(p room.Pair) int64 { return p.Mass },
	func(p room.Pair) int64 { return p.Bulk },
}

// bruteK is the definition of 5.3.2 by search: the largest k up to k0 with
// k*c_d <= R_d in both dimensions, where R_d uses the residue only for outputs
// whose q*E is not a multiple of D. Values are small enough for int64.
func bruteK(ins []room.Item, outs []room.Output, e int64, free, upkeep room.Pair, k0 int64) int64 {
	best := int64(0)
	for k := int64(1); k <= k0; k++ {
		fits := true
		for _, d := range dims {
			var in, a, s int64
			for _, it := range ins {
				in += it.Qty * d(it.Size)
			}
			for _, o := range outs {
				if o.Discard {
					continue
				}
				a += o.Qty * e * d(o.Size)
				if (o.Qty*e)%room.Denominator != 0 {
					s += d(o.Size) * o.Residue
				}
			}
			c := a - room.Denominator*in
			r := room.Denominator*(d(free)-d(upkeep)) - s
			if k*c > r {
				fits = false
			}
		}
		if fits {
			best = k
		}
	}
	return best
}

// exactFits reports whether the exact pending output of k executions, with the
// executor's residues, fits the free room less the upkeep room once the inputs
// are taken (the soundness claim of 5.3.2).
func exactFits(ins []room.Item, outs []room.Output, e int64, free, upkeep room.Pair, k int64) bool {
	for _, d := range dims {
		var in, out int64
		for _, it := range ins {
			in += k * it.Qty * d(it.Size)
		}
		for _, o := range outs {
			if o.Discard {
				continue
			}
			out += d(o.Size) * ((k*o.Qty*e + o.Residue) / room.Denominator)
		}
		if out > d(free)-d(upkeep)+in {
			return false
		}
	}
	return true
}

func randomRecipe(rng *rand.Rand) (ins []room.Item, outs []room.Output, e int64, free, upkeep room.Pair, k0 int64) {
	size := func() room.Pair { return room.Pair{Mass: rng.Int64N(4), Bulk: rng.Int64N(4)} }
	for range rng.IntN(3) {
		ins = append(ins, room.Item{Qty: 1 + rng.Int64N(4), Size: size()})
	}
	yields := []int64{3333, 5000, 7500, 10_000, 12_345, 20_000}
	e = yields[rng.IntN(len(yields))]
	for range 1 + rng.IntN(2) {
		o := room.Output{Item: room.Item{Qty: 1 + rng.Int64N(5), Size: size()}, Residue: rng.Int64N(room.Denominator), Discard: rng.IntN(6) == 0}
		if (o.Qty*e)%room.Denominator == 0 {
			o.Residue = 0 // an exempt output carries no remainder (6.6.1)
		}
		outs = append(outs, o)
	}
	free = room.Pair{Mass: rng.Int64N(40), Bulk: rng.Int64N(40)}
	upkeep = room.Pair{Mass: rng.Int64N(6), Bulk: rng.Int64N(6)}
	return ins, outs, e, free, upkeep, 1 + rng.Int64N(12)
}

// TestKRoom_MatchesExactLargestK verifies the bound of 5.3.2 against two
// searches over 20 000 random recipes: it equals the largest k up to k0 that
// satisfies the inequality k*c <= R in both dimensions (the definition), and
// it is sound, because the exact pending output of the k chosen, with the
// executor's own residues, fits once the inputs are taken. Where every output
// is exempt (q*E a multiple of D) the inequality is the exact condition, so the
// bound equals the exact largest k.
//
// Requirement: FR-STO-012
func TestKRoom_MatchesExactLargestK(t *testing.T) {
	t.Parallel()
	rng := rand.New(rand.NewPCG(1, 2))
	exempt := 0
	for n := range 20_000 {
		ins, outs, e, free, upkeep, k0 := randomRecipe(rng)
		got := room.KRoom(ins, outs, e, free, upkeep, k0)
		if want := bruteK(ins, outs, e, free, upkeep, k0); got != want {
			t.Fatalf("case %d: KRoom(%v, %v, %d, %v, %v, %d) = %d, want %d (the largest k with k*c <= R)", n, ins, outs, e, free, upkeep, k0, got, want)
		}
		if got > 0 && !exactFits(ins, outs, e, free, upkeep, got) {
			t.Fatalf("case %d: KRoom = %d is not sound: the exact output of %d executions does not fit (%v, %v, %d, %v, %v)", n, got, got, ins, outs, e, free, upkeep)
		}
		if allExempt(outs, e) {
			exempt++
			exact := int64(0)
			for k := int64(1); k <= k0; k++ {
				if exactFits(ins, outs, e, free, upkeep, k) {
					exact = k
				}
			}
			if got != exact {
				t.Fatalf("case %d: all outputs exempt, KRoom = %d, exact largest k = %d", n, got, exact)
			}
		}
	}
	if exempt < 500 {
		t.Fatalf("only %d of 20000 cases had exempt outputs only: the tightness check is not exercised", exempt)
	}
}

func allExempt(outs []room.Output, e int64) bool {
	for _, o := range outs {
		if !o.Discard && (o.Qty*e)%room.Denominator != 0 {
			return false
		}
	}
	return true
}

// TestKRoom_WorkedExample verifies the table of example 18.1 (a barn): a
// grower with room (400, 200), grain of size (2, 1), an output of 120 per
// execution and an upkeep room of (6, 3) starts one execution with an empty
// barn and none with 100 or 160 grain held.
//
// Requirement: FR-STO-012
func TestKRoom_WorkedExample(t *testing.T) {
	t.Parallel()
	grain := room.Pair{Mass: 2, Bulk: 1}
	outs := []room.Output{{Item: room.Item{Qty: 120, Size: grain}}}
	for _, tt := range []struct {
		name string
		held int64
		want int64
	}{
		{"empty barn", 0, 1},
		{"100 held", 100, 0},
		{"160 held", 160, 0},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			free := room.Pair{Mass: 400 - 2*tt.held, Bulk: 200 - tt.held}
			if got := room.KRoom(nil, outs, room.Denominator, free, room.Pair{Mass: 6, Bulk: 3}, 5); got != tt.want {
				t.Errorf("KRoom(held %d) = %d, want %d", tt.held, got, tt.want)
			}
		})
	}
}

// TestKRoom_ExemptOutputMultipleOfE verifies that an output whose q*E is a
// multiple of D ignores the executor's residue, and one that is not does not.
//
// 3 units at E = 10 000 is a multiple, so a residue of 9 999 adds nothing and 5
// executions of load 3 each fit a free room of 15; at E = 5 000, 3 units give
// 1.5 per execution, so the same residue is reserved: R = 10 000*15 - 9 999 =
// 140 001, c = 15 000, and k <= 9 (the bound asks for the fraction of a unit
// the residue may add).
//
// Requirement: FR-STO-012
func TestKRoom_ExemptOutputMultipleOfE(t *testing.T) {
	t.Parallel()
	one := room.Pair{Mass: 1}
	out := []room.Output{{Item: room.Item{Qty: 3, Size: one}, Residue: 9_999}}
	free := room.Pair{Mass: 15, Bulk: 100}
	if got := room.KRoom(nil, out, 10_000, free, room.Pair{}, 99); got != 5 {
		t.Errorf("KRoom(E=10000, exempt) = %d, want 5", got)
	}
	// E = 5 000: c = 3*5000*1 = 15 000, R = 150 000 - 9 999 = 140 001, k = 9.
	if got := room.KRoom(nil, out, 5_000, room.Pair{Mass: 15, Bulk: 100}, room.Pair{}, 99); got != 9 {
		t.Errorf("KRoom(E=5000, residue reserved) = %d, want 9", got)
	}
}

// TestKRoom_DiscardOutputTakesNone verifies that a discard output is neither in
// the load of the outputs nor in the residue: it is created and disposed at
// the completion and never sits in the account.
//
// Requirement: FR-STO-012, FR-STO-013
func TestKRoom_DiscardOutputTakesNone(t *testing.T) {
	t.Parallel()
	big := room.Pair{Mass: 1000, Bulk: 1000}
	outs := []room.Output{
		{Item: room.Item{Qty: 1, Size: room.Pair{Mass: 2, Bulk: 1}}},
		{Item: room.Item{Qty: 50, Size: big}, Residue: 7, Discard: true},
	}
	// Only the first output counts: 2 mass per execution in a free room of 10.
	if got := room.KRoom(nil, outs, room.Denominator, room.Pair{Mass: 10, Bulk: 10}, room.Pair{}, 99); got != 5 {
		t.Errorf("KRoom with a discard output = %d, want 5", got)
	}
}

// TestKRoom_InputsFreeRoom verifies the lower bound: a recipe whose inputs are
// larger than its output frees room as it goes, so a full room (R < 0) admits
// k only from ceil(-R / -c), and none when k0 is below it.
//
// Input 3 of size (4, 0), output 1 of size (2, 0), E = 10 000: c = 2*10 000 -
// 10 000*12 = -100 000; with free room less upkeep -4, R = -40 000 (residue is
// exempt), so k >= 40 000/100 000 rounded up = 1.
//
// Requirement: FR-STO-012
func TestKRoom_InputsFreeRoom(t *testing.T) {
	t.Parallel()
	ins := []room.Item{{Qty: 3, Size: room.Pair{Mass: 4}}}
	outs := []room.Output{{Item: room.Item{Qty: 1, Size: room.Pair{Mass: 2}}}}
	free := room.Pair{Mass: -4, Bulk: 0}
	if got := room.KRoom(ins, outs, room.Denominator, free, room.Pair{}, 5); got != 5 {
		t.Errorf("KRoom(free -4) = %d, want 5 (k >= 1 fits, up to k0)", got)
	}
	// A lower bound above k0 leaves none: free -1000 gives R = -10 000 000, k >= 100.
	if got := room.KRoom(ins, outs, room.Denominator, room.Pair{Mass: -1000}, room.Pair{}, 5); got != 0 {
		t.Errorf("KRoom(lower bound above k0) = %d, want 0", got)
	}
}

// TestKRoom_Edges verifies the boundaries: k0 below 1, an overflow, an input
// load of the minimum int64, c = 0 with R exactly 0, and a saturated lower
// bound all give the safe answer.
//
// Requirement: FR-STO-012
func TestKRoom_Edges(t *testing.T) {
	t.Parallel()
	one := room.Pair{Mass: 1, Bulk: 1}
	out := func(q int64) []room.Output { return []room.Output{{Item: room.Item{Qty: q, Size: one}}} }
	for _, tt := range []struct {
		name  string
		ins   []room.Item
		outs  []room.Output
		e     int64
		free  room.Pair
		upk   room.Pair
		k0    int64
		want  int64
		check string
	}{
		{"k0 zero", nil, out(1), 10_000, room.Pair{Mass: 9, Bulk: 9}, room.Pair{}, 0, 0, ""},
		{"output overflows", nil, out(math.MaxInt64), 10_000, room.Pair{Mass: 9, Bulk: 9}, room.Pair{}, 5, 0, ""},
		{"input overflows", []room.Item{{Qty: math.MaxInt64, Size: room.Pair{Mass: 2, Bulk: 2}}}, out(1), 10_000, room.Pair{Mass: 9, Bulk: 9}, room.Pair{}, 5, 0, ""},
		{"no output room left exactly", nil, out(1), 10_000, room.Pair{Mass: 1, Bulk: 1}, room.Pair{Mass: 1, Bulk: 1}, 5, 0, ""},
		{"zero load fits any k", nil, []room.Output{{Item: room.Item{Qty: 1}}}, 10_000, room.Pair{}, room.Pair{}, 7, 7, ""},
		{"free less upkeep wraps", nil, []room.Output{{Item: room.Item{Qty: 1}}}, 10_000, room.Pair{Mass: math.MinInt64}, room.Pair{Mass: 1}, 7, 0, ""},
		{"c zero R zero", []room.Item{{Qty: 1, Size: one}}, out(1), 10_000, room.Pair{}, room.Pair{}, 4, 4, ""},
		{"c zero R negative", []room.Item{{Qty: 1, Size: one}}, out(1), 10_000, room.Pair{Mass: -1}, room.Pair{}, 4, 0, ""},
		{"huge lower bound saturates", []room.Item{{Qty: 1, Size: room.Pair{Mass: math.MaxInt64 / 10_000}}}, out(0), 10_000, room.Pair{Mass: math.MinInt64 / 2}, room.Pair{}, 4, 0, ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := room.KRoom(tt.ins, tt.outs, tt.e, tt.free, tt.upk, tt.k0); got != tt.want {
				t.Errorf("KRoom(%v, %v, %d, %v, %v, %d) = %d, want %d", tt.ins, tt.outs, tt.e, tt.free, tt.upk, tt.k0, got, tt.want)
			}
		})
	}
}
