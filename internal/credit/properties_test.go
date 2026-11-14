package credit_test

import (
	"fmt"
	"math/big"
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
		if msg := interestProblem(x, rate, term); msg != "" {
			t.Fatalf("seed %d: Interest(%d, %d, %d, %d): %s", seed, x, rate, term, ticksYear, msg)
		}
	}
}

// interestProblem returns what is wrong with Interest at the input, or "": the
// quotient against the oracle, the zero cases and the monotonicity.
func interestProblem(x, rate, term int64) string {
	got, ok := credit.Interest(x, rate, term, ticksYear)
	if msg := oracleProblem(got, ok, exactInterest(x, rate, term, ticksYear)); msg != "" {
		return msg
	}
	for name, in := range map[string][2]int64{"a rate of 0": {0, term}, "a term of 0": {rate, 0}} {
		if zero, _ := credit.Interest(x, in[0], in[1], ticksYear); zero != 0 {
			return fmt.Sprintf("with %s = %d, want 0", name, zero)
		}
	}
	for name, up := range map[string][3]int64{"principal": {x + 1, rate, term}, "rate": {x, rate + 1, term}, "term": {x, rate, term + 1}} {
		if more, ok := credit.Interest(up[0], up[1], up[2], ticksYear); ok && more < got {
			return fmt.Sprintf("fell from %d to %d when the %s rose", got, more, name)
		}
	}
	return ""
}

// oracleProblem compares an Interest result with the exact quotient: a quotient
// above int64 is reported, never wrapped (ok false and the interest 0); otherwise
// the result is the quotient.
func oracleProblem(got int64, ok bool, want *big.Int) string {
	fits := want.IsInt64()
	if ok == fits && ((fits && got == want.Int64()) || (!fits && got == 0)) {
		return ""
	}
	return fmt.Sprintf("= %d, %v, want %s (fits int64: %v)", got, ok, want, fits)
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
		if msg := waterfallProblem(&s, balance, owed, tie); msg != "" {
			t.Fatalf("seed %d: Waterfall(%d, %v): %s", seed, balance, owed, msg)
		}
	}
}

// waterfallProblem returns what is wrong with Waterfall at the input, or "": the
// shares within their claims, summing to what can be paid, the rest completing the
// balance, and the answer repeating.
func waterfallProblem(s *credit.Scratch, balance int64, owed []int64, tie []uint64) string {
	shares, left, ok := s.Waterfall(nil, balance, owed, tie)
	if !ok {
		return "refused"
	}
	var paid, total int64
	for i, sh := range shares {
		if sh < 0 || sh > owed[i] {
			return fmt.Sprintf("share %d = %d, want within [0, %d]", i, sh, owed[i])
		}
		paid += sh
		total += owed[i]
	}
	if want := min(balance, total); paid != want {
		return fmt.Sprintf("paid %d, want min(balance, owed) = %d", paid, want)
	}
	if paid+left != balance || left < 0 {
		return fmt.Sprintf("paid %d + left %d != balance (or left < 0)", paid, left)
	}
	if again, leftAgain, _ := s.Waterfall(nil, balance, owed, tie); !slices.Equal(again, shares) || leftAgain != left {
		return fmt.Sprintf("a second call gave %v, %d, want %v, %d", again, leftAgain, shares, left)
	}
	return ""
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
