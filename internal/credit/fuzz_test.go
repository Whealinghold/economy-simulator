package credit_test

import (
	"math/big"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/credit"
)

// FuzzInterest verifies, for any principal, rate, term and year, that Interest
// never panics, reports ok exactly when the input is in its domain and the
// quotient fits int64, and then equals the big-integer quotient rounded half to
// even (so it is within one unit of the exact quotient).
//
// Requirement: NFR-CRD-005, FR-CRD-027
func FuzzInterest(f *testing.F) {
	f.Add(int64(5_700), int64(600), int64(65), int64(365))
	f.Add(maxMoney, maxRateBP, maxTerm, int64(365_000))
	f.Add(int64(1), int64(1), int64(1), int64(2))
	f.Add(int64(-1), int64(0), int64(0), int64(0))
	f.Fuzz(func(t *testing.T, x, rate, term, tpy int64) {
		got, ok := credit.Interest(x, rate, term, tpy)
		if msg := interestViolation(x, rate, term, tpy, got, ok); msg != "" {
			t.Fatalf("Interest(%d, %d, %d, %d) = %d, %v: %s", x, rate, term, tpy, got, ok, msg)
		}
	})
}

// interestDomain reports whether the arguments are in the domain of Interest: not
// negative, a year of at least one tick, and a denominator that fits int64.
func interestDomain(x, rate, term, tpy int64) bool {
	if x < 0 || rate < 0 || term < 0 || tpy < 1 {
		return false
	}
	return new(big.Int).Mul(big.NewInt(10_000), big.NewInt(tpy)).IsInt64()
}

// interestViolation reports how a result of Interest breaks its contract, or "".
func interestViolation(x, rate, term, tpy, got int64, ok bool) string {
	if !interestDomain(x, rate, term, tpy) {
		if ok || got != 0 {
			return "a result outside the domain"
		}
		return ""
	}
	want := exactInterest(x, rate, term, tpy)
	if want.IsInt64() != ok || (ok && want.Int64() != got) {
		return "not the exact half-even quotient " + want.String()
	}
	return ""
}

// FuzzAllocate verifies, for any pay and up to four claims, that Allocate never
// panics, that it refuses exactly what is outside its domain, and that otherwise
// the shares sum to the pay, none is above what its claim owes and each is the
// floor of the exact share or one above it.
//
// Requirement: NFR-CRD-005, FR-CRD-032
func FuzzAllocate(f *testing.F) {
	f.Add(int64(1_001), int64(600), int64(900), int64(0), int64(0), uint64(1))
	f.Add(int64(800), int64(500), int64(700), int64(3), int64(0), uint64(0))
	f.Add(int64(0), int64(0), int64(0), int64(0), int64(0), uint64(9))
	f.Fuzz(func(t *testing.T, pay, a, b, c, d int64, key uint64) {
		owed := []int64{a, b, c, d}
		tie := []uint64{key, key >> 1, key >> 2, 0}
		var s credit.Scratch
		shares, ok := s.Allocate(nil, pay, owed, tie)
		total, in := domainSum(owed)
		in = in && pay >= 0 && big.NewInt(pay).Cmp(total) <= 0
		if ok != in {
			t.Fatalf("Allocate(%d, %v) ok = %v, want %v", pay, owed, ok, in)
		}
		if ok {
			if msg := checkShares(pay, owed, shares); msg != "" {
				t.Fatalf("Allocate(%d, %v) = %v: %s", pay, owed, shares, msg)
			}
		}
	})
}

// domainSum returns the sum of the owed and whether every one is not negative and
// the sum fits int64.
func domainSum(owed []int64) (*big.Int, bool) {
	total, in := new(big.Int), true
	for _, o := range owed {
		in = in && o >= 0
		total.Add(total, big.NewInt(o))
	}
	return total, in && total.IsInt64()
}

// FuzzWaterfall verifies, for any balance and up to three claims, that Waterfall
// never panics and that the shares and the rest sum to the balance, no share is
// above its claim and the rest is the balance less what the claims could take.
//
// Requirement: NFR-CRD-005, FR-CRD-040
func FuzzWaterfall(f *testing.F) {
	f.Add(int64(800), int64(500), int64(700), int64(0))
	f.Add(int64(1_500), int64(500), int64(700), int64(0))
	f.Add(int64(0), int64(1), int64(1), int64(1))
	f.Fuzz(func(t *testing.T, balance, a, b, c int64) {
		owed := []int64{a, b, c}
		var s credit.Scratch
		shares, left, ok := s.Waterfall(nil, balance, owed, []uint64{0, 0, 0})
		total, in := domainSum(owed)
		in = in && balance >= 0
		if ok != in {
			t.Fatalf("Waterfall(%d, %v) ok = %v, want %v", balance, owed, ok, in)
		}
		if !ok {
			return
		}
		paid := sum(shares)
		if new(big.Int).Add(paid, big.NewInt(left)).Cmp(big.NewInt(balance)) != 0 || left < 0 {
			t.Fatalf("Waterfall(%d, %v) = %v, %d: shares and rest do not sum to the balance", balance, owed, shares, left)
		}
		if paid.Cmp(total) > 0 || (left > 0 && paid.Cmp(total) != 0) {
			t.Fatalf("Waterfall(%d, %v) = %v, %d: the estate gets money while a claim is short", balance, owed, shares, left)
		}
	})
}

// FuzzRound verifies, for any lane of up to three requests and three offers, that
// Match never panics, never fills below the minimum, above a request's limit or
// past an offer's term, never lends an offer's principal twice, and leaves the
// requests and the offers with exactly the principal that was filled.
//
// Requirement: NFR-CRD-005, FR-CRD-023, FR-CRD-024
func FuzzRound(f *testing.F) {
	f.Add(uint64(1), int64(100))
	f.Add(uint64(7), int64(1))
	f.Add(uint64(99), int64(0))
	f.Fuzz(func(t *testing.T, seed uint64, minP int64) {
		reqs, offs, _ := randomRound(seed)
		credit.SortRequests(reqs)
		credit.SortOffers(offs)
		before := laneBefore(reqs, offs)
		var fills []credit.Fill
		credit.Match(reqs, offs, minP, formAll(&fills))
		if msg := roundViolation(before, reqs, offs, fills, minP); msg != "" {
			t.Fatalf("seed %d, min %d: %s", seed, minP, msg)
		}
	})
}

// lane is a copy of the rows of a round before the matching, by Ref.
type lane struct{ reqs, offs map[int]credit.Row }

// laneBefore copies the rows by Ref.
func laneBefore(reqs, offs []credit.Row) lane {
	l := lane{reqs: map[int]credit.Row{}, offs: map[int]credit.Row{}}
	for _, r := range reqs {
		l.reqs[r.Ref] = r
	}
	for _, o := range offs {
		l.offs[o.Ref] = o
	}
	return l
}

// roundViolation reports the first rule a round broke: a fill below the minimum,
// above the request's limit, past the offer's term, at another rate than the
// offer's or between one agent and itself, or a remainder that is not the
// principal less the fills.
func roundViolation(before lane, reqs, offs []credit.Row, fills []credit.Fill, minP int64) string {
	reqLoss, offLoss := map[int]int64{}, map[int]int64{}
	for _, fl := range fills {
		r, o := before.reqs[fl.Request], before.offs[fl.Offer]
		if fl.Principal < max(minP, 1) || fl.RateBP > r.RateBP || o.Term < r.Term || fl.RateBP != o.RateBP || o.Agent == r.Agent {
			return "a fill breaks the rules"
		}
		reqLoss[fl.Request] += fl.Principal
		offLoss[fl.Offer] += fl.Principal
	}
	if remainderViolation(before.reqs, reqs, reqLoss) || remainderViolation(before.offs, offs, offLoss) {
		return "a remainder is not its principal less the fills"
	}
	return ""
}

// remainderViolation reports whether a row's remainder differs from its principal
// before less the fills it took part in, or is negative.
func remainderViolation(before map[int]credit.Row, rows []credit.Row, loss map[int]int64) bool {
	for _, r := range rows {
		if r.Principal != before[r.Ref].Principal-loss[r.Ref] || r.Principal < 0 {
			return true
		}
	}
	return false
}
