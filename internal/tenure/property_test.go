package tenure_test

import (
	"cmp"
	"math"
	"math/big"
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/tenure"
)

// expectedApportion is the largest-remainder apportionment of stock over the
// weights, computed with arbitrary precision and without looking at the order the
// parts are listed in: floor(stock * w / W) each, and the units left to the parts
// of the largest remainder, ties by agents before the commons, ascending key,
// ascending ID.
func expectedApportion(stock int64, parts []tenure.Part) map[int]int64 {
	total := new(big.Int)
	for _, p := range parts {
		total.Add(total, bi(p.Share))
	}
	want := make(map[int]int64, len(parts))
	type ranked struct {
		part tenure.Part
		rem  *big.Int
	}
	var order []ranked
	var given int64
	for _, p := range parts {
		want[p.Ref] = 0
		if p.Share == 0 || total.Sign() == 0 {
			continue
		}
		num := new(big.Int).Mul(bi(stock), bi(p.Share))
		q, r := new(big.Int).QuoRem(num, total, new(big.Int))
		want[p.Ref] = q.Int64()
		given += q.Int64()
		order = append(order, ranked{p, r})
	}
	slices.SortFunc(order, func(a, b ranked) int {
		return cmp.Or(b.rem.Cmp(a.rem), boolInt(a.part.Commons)-boolInt(b.part.Commons), cmp.Compare(a.part.Key, b.part.Key), cmp.Compare(a.part.ID, b.part.ID))
	})
	for i := 0; i < int(stock-given); i++ {
		want[order[i].part.Ref]++
	}
	return want
}

// boolInt is 1 for true and 0 for false.
func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// randomParts builds up to eight parts with weights from basis points up to
// 100 000, a few zero, keys from a small range so that ties are common, distinct
// IDs and at most one commons part.
func randomParts(rng *rand.Rand) []tenure.Part {
	parts := make([]tenure.Part, 1+rng.IntN(8))
	for i := range parts {
		parts[i] = tenure.Part{Ref: i, ID: int64(1 + i), Share: int64(rng.IntN(5)) * int64(rng.IntN(25_000)), Key: uint64(rng.IntN(3))}
	}
	parts[rng.IntN(len(parts))].Commons = rng.IntN(2) == 0
	rng.Shuffle(len(parts), func(i, j int) { parts[i], parts[j] = parts[j], parts[i] })
	return parts
}

// TestApportion_MatchesExactLargestRemainder verifies, over random weights and
// stocks up to MaxInt64 (so stock * weight passes 64 bits), that Apportion gives
// exactly the largest-remainder result computed with arbitrary precision: the
// units sum to the stock, no part is more than one unit from its exact share,
// and the leftover units go to the parts the tie rule ranks first. The parts are
// listed in a random order, so the result is also independent of it.
//
// Requirement: FR-LND-007, NFR-LND-003
//
// Guard: the entitlements are money-like: a unit created or destroyed by the
// rounding would make the units drawn exceed the stock or leave stock unlet.
func TestApportion_MatchesExactLargestRemainder(t *testing.T) {
	t.Parallel()
	for _, seed := range oracleSeeds {
		rng := rand.New(rand.NewPCG(seed, 99))
		for i := range 1500 {
			parts := randomParts(rng)
			stock := []int64{0, 1, 7, 80, 10_000, math.MaxInt64 / 10_000, math.MaxInt64}[rng.IntN(7)]
			if rng.IntN(2) == 0 {
				stock = rng.Int64N(1 << uint(1+rng.IntN(62)))
			}
			var weight int64
			for _, p := range parts {
				weight += p.Share
			}
			out := make([]int64, len(parts))
			ok := tenure.Apportion(stock, slices.Clone(parts), out)
			if weight == 0 && stock > 0 {
				if ok || sum(out) != 0 {
					t.Fatalf("seed %d case %d: Apportion(%d, %+v) = %v, %v, want refused with 0: no weight", seed, i, stock, parts, out, ok)
				}
				continue
			}
			want := expectedApportion(stock, parts)
			for _, p := range parts {
				if !ok || out[p.Ref] != want[p.Ref] {
					t.Fatalf("seed %d case %d: Apportion(%d, %+v) = %v, %v, want part %d to get %d", seed, i, stock, parts, out, ok, p.Ref, want[p.Ref])
				}
			}
			if sum(out) != stock {
				t.Fatalf("seed %d case %d: Apportion(%d, %+v) = %v sums to %d", seed, i, stock, parts, out, sum(out))
			}
		}
	}
}

// TestCredit_PaidPlusForfeitedIsTheAmount verifies over random balances, limits
// and amounts that a credit never creates or destroys money: what the payee
// receives plus what is forfeited is the amount, the payee never passes its
// limit, and nothing is forfeited while there is room.
//
// Requirement: FR-LND-016, FR-LND-017
//
// Guard: the rent is a transfer; a payee at the money cap must not make the
// tenant pay more than the payee received without the forfeit being counted.
func TestCredit_PaidPlusForfeitedIsTheAmount(t *testing.T) {
	t.Parallel()
	g := newCaseGen(11)
	for i := range oracleCases {
		balance, amount, limit := g.val(), g.val(), g.val()
		paid, forfeited, ok := tenure.Credit(balance, amount, limit)
		if balance > limit {
			if ok || paid != 0 || forfeited != 0 {
				t.Fatalf("case %d: Credit(%d, %d, %d) = %d, %d, %v, want refused: balance above limit", i, balance, amount, limit, paid, forfeited, ok)
			}
			continue
		}
		room := limit - balance
		if !ok || paid+forfeited != amount || paid < 0 || forfeited < 0 || paid > room || (forfeited > 0 && paid != room) {
			t.Fatalf("case %d: Credit(%d, %d, %d) = %d, %d, %v, want paid+forfeited = %d within room %d", i, balance, amount, limit, paid, forfeited, ok, amount, room)
		}
	}
}
