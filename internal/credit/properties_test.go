package credit_test

import (
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/credit"
)

// TestInterest_Property verifies, over 500 random cases inside the legal domain
// with a seed of their own (the failing seed is in the message), that the interest
// equals the exact big-integer quotient rounded half to even when that fits int64,
// that a quotient above int64 is reported (ok false, 0) and never wraps, that it is
// 0 for a rate or a term of 0, and that it does not fall when the principal, the
// rate or the term rises.
//
// Requirement: FR-CRD-027, NFR-CRD-005
func TestInterest_Property(t *testing.T) {
	t.Parallel()
	for seed := uint64(1); seed <= 500; seed++ {
		r := rand.New(rand.NewPCG(seed, 13))
		x, rate, term := r.Int64N(maxMoney+1), r.Int64N(maxRateBP+1), r.Int64N(maxTerm+1)
		got, ok := credit.Interest(x, rate, term, ticksYear)
		// A quotient above int64 is reported, never wrapped: ok is false and the
		// interest 0; otherwise it is the exact quotient.
		want := exactInterest(x, rate, term, ticksYear)
		if fits := want.IsInt64(); ok != fits || (fits && got != want.Int64()) || (!fits && got != 0) {
			t.Fatalf("seed %d: Interest(%d, %d, %d, %d) = %d, %v, want %s (fits int64: %v)", seed, x, rate, term, ticksYear, got, ok, want, fits)
		}
		if zero, _ := credit.Interest(x, 0, term, ticksYear); zero != 0 {
			t.Fatalf("seed %d: Interest(%d, 0, %d) = %d, want 0 (interest-free)", seed, x, term, zero)
		}
		if zero, _ := credit.Interest(x, rate, 0, ticksYear); zero != 0 {
			t.Fatalf("seed %d: Interest(%d, %d, 0) = %d, want 0 (no term)", seed, x, rate, zero)
		}
		for name, up := range map[string][3]int64{"principal": {x + 1, rate, term}, "rate": {x, rate + 1, term}, "term": {x, rate, term + 1}} {
			if more, ok := credit.Interest(up[0], up[1], up[2], ticksYear); ok && more < got {
				t.Fatalf("seed %d: Interest fell from %d to %d when the %s rose (%d, %d, %d)", seed, got, more, name, x, rate, term)
			}
		}
	}
}

// TestPay_Property verifies, over 500 random cases with a seed of their own, that a
// debtor never pays more than it owes or holds, pays 0 while the balance is inside
// the shield, and pays the whole debt once the balance clears the debt plus the
// shield.
//
// Requirement: FR-CRD-031, NFR-CRD-005
func TestPay_Property(t *testing.T) {
	t.Parallel()
	for seed := uint64(1); seed <= 500; seed++ {
		r := rand.New(rand.NewPCG(seed, 17))
		balance, owed := r.Int64N(1_000_000), r.Int64N(1_000_000)
		ticks, upkeep := r.Int64N(10), r.Int64N(50_000)
		shield := ticks * upkeep
		pay, ok := credit.Pay(balance, ticks, upkeep, owed, maxMoney)
		switch {
		case !ok || pay < 0 || pay > owed || pay > balance:
			t.Fatalf("seed %d: Pay(%d, %d, %d, %d) = %d, %v, want within [0, min(owed, balance)]", seed, balance, ticks, upkeep, owed, pay, ok)
		case balance <= shield && pay != 0:
			t.Fatalf("seed %d: Pay(%d) = %d with the balance inside the shield %d, want 0", seed, balance, pay, shield)
		case balance >= owed+shield && pay != owed:
			t.Fatalf("seed %d: Pay(%d) = %d with the balance above debt %d plus shield %d, want the whole debt", seed, balance, pay, owed, shield)
		}
	}
}

// TestWaterfall_Property verifies, over 500 random cases with a seed of their own,
// that the waterfall pays exactly what it can and never more than is owed: the
// shares sum to min(balance, total owed), none exceeds its claim, the shares and
// the rest sum to the balance (nothing is created or lost), and a second call with
// the same input gives the same answer.
//
// Requirement: FR-CRD-040, NFR-CRD-005
func TestWaterfall_Property(t *testing.T) {
	t.Parallel()
	var s credit.Scratch
	for seed := uint64(1); seed <= 500; seed++ {
		r := rand.New(rand.NewPCG(seed, 19))
		owed := make([]int64, r.IntN(9))
		tie := make([]uint64, len(owed))
		var total int64
		for i := range owed {
			owed[i], tie[i] = r.Int64N(1_000_000_000), r.Uint64N(4)
			total += owed[i]
		}
		balance := r.Int64N(2*total + 2)
		shares, left, ok := s.Waterfall(nil, balance, owed, tie)
		if !ok {
			t.Fatalf("seed %d: Waterfall(%d, %v) refused", seed, balance, owed)
		}
		var paid int64
		for i, sh := range shares {
			if sh < 0 || sh > owed[i] {
				t.Fatalf("seed %d: share %d of claim %d = %d, want within [0, %d]", seed, i, i, sh, owed[i])
			}
			paid += sh
		}
		if want := min(balance, total); paid != want {
			t.Fatalf("seed %d: Waterfall(%d, %v) paid %d, want min(balance, owed) = %d", seed, balance, owed, paid, want)
		}
		if paid+left != balance || left < 0 {
			t.Fatalf("seed %d: paid %d + left %d != balance %d (or left < 0)", seed, paid, left, balance)
		}
		again, leftAgain, _ := s.Waterfall(nil, balance, owed, tie)
		if !slices.Equal(again, shares) || leftAgain != left {
			t.Fatalf("seed %d: a second Waterfall gave %v, %d, want %v, %d", seed, again, leftAgain, shares, left)
		}
	}
}

// TestRound_ArrivalOrderIsIrrelevant verifies, over 500 random rounds with a seed of
// their own, that the order the rows arrive in changes neither the sorted lanes nor
// the fills: the sorts are total (they end at Ref) and the matching reads only the
// sorted lanes, so a round is deterministic whatever order the caller built it in.
//
// Requirement: FR-CRD-022, FR-CRD-023, NFR-CRD-005
//
// Guard: the round's input order comes from the world's iteration of agents, which
// is not part of the rules; a sort or a scan that read it would make a run depend on
// map or slot order.
func TestRound_ArrivalOrderIsIrrelevant(t *testing.T) {
	t.Parallel()
	for seed := uint64(1); seed <= 500; seed++ {
		reqs, offs, minP := randomRound(seed)
		credit.SortRequests(reqs)
		credit.SortOffers(offs)
		var want []credit.Fill
		credit.Match(slices.Clone(reqs), slices.Clone(offs), minP, formAll(&want))

		r := rand.New(rand.NewPCG(seed, 23))
		sreqs, soffs := slices.Clone(reqs), slices.Clone(offs)
		r.Shuffle(len(sreqs), func(i, j int) { sreqs[i], sreqs[j] = sreqs[j], sreqs[i] })
		r.Shuffle(len(soffs), func(i, j int) { soffs[i], soffs[j] = soffs[j], soffs[i] })
		credit.SortRequests(sreqs)
		credit.SortOffers(soffs)
		if !slices.Equal(sreqs, reqs) || !slices.Equal(soffs, offs) {
			t.Fatalf("seed %d: the sorted lanes depend on arrival order: %+v vs %+v, %+v vs %+v", seed, sreqs, reqs, soffs, offs)
		}
		var got []credit.Fill
		credit.Match(sreqs, soffs, minP, formAll(&got))
		if !slices.Equal(got, want) {
			t.Fatalf("seed %d: fills after a shuffled arrival = %+v, want %+v", seed, got, want)
		}
	}
}
