package credit_test

import (
	"testing"

	"github.com/Whealinghold/economy-sim/internal/credit"
)

// BenchmarkInterest measures the interest of a loan.
//
// Requirement: NFR-CRD-004
func BenchmarkInterest(b *testing.B) {
	b.ReportAllocs()
	for range b.N {
		_, _ = credit.Interest(5_700, 600, 65, 365)
	}
}

// BenchmarkAllocate measures the allocation of a payment among sixteen claims
// with a grown Scratch and a reused destination.
//
// Requirement: NFR-CRD-004
func BenchmarkAllocate(b *testing.B) {
	owed := make([]int64, 16)
	tie := make([]uint64, 16)
	for i := range owed {
		owed[i] = int64(100 + 37*i)
		tie[i] = uint64(i * 7 % 5)
	}
	var s credit.Scratch
	dst := make([]int64, 0, 16)
	b.ReportAllocs()
	for range b.N {
		_, _ = s.Allocate(dst, 1_001, owed, tie)
	}
}

// BenchmarkWaterfall measures the waterfall of a removed debtor with sixteen claims.
//
// Requirement: NFR-CRD-004
func BenchmarkWaterfall(b *testing.B) {
	owed := make([]int64, 16)
	tie := make([]uint64, 16)
	for i := range owed {
		owed[i] = int64(100 + 37*i)
		tie[i] = uint64(i * 7 % 5)
	}
	var s credit.Scratch
	dst := make([]int64, 0, 16)
	b.ReportAllocs()
	for range b.N {
		_, _, _ = s.Waterfall(dst, 900, owed, tie)
	}
}

// BenchmarkMatch measures the matching of a lane of 64 requests and 64 offers.
//
// Requirement: NFR-CRD-004
func BenchmarkMatch(b *testing.B) {
	reqs, offs := grow(3)
	credit.SortRequests(reqs)
	credit.SortOffers(offs)
	r2, o2 := make([]credit.Row, len(reqs)), make([]credit.Row, len(offs))
	form := func(credit.Fill) credit.Outcome { return credit.Formed }
	b.ReportAllocs()
	for range b.N {
		copy(r2, reqs)
		copy(o2, offs)
		credit.Match(r2, o2, 1, form)
	}
}

// grow returns the rows of the random round of a seed, repeated to 64 requests
// and 64 offers with fresh Refs.
func grow(seed uint64) (reqs, offs []credit.Row) {
	r0, o0, _ := randomRound(seed)
	for i := range 64 {
		r, o := r0[i%len(r0)], o0[i%len(o0)]
		r.Ref, o.Ref = i, i
		reqs, offs = append(reqs, r), append(offs, o)
	}
	return reqs, offs
}
