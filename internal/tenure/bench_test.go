package tenure_test

import (
	"testing"

	"github.com/Whealinghold/economy-sim/internal/tenure"
)

// BenchmarkApportion reports the cost of the apportionment of a stock among
// 200 holders and the commons (spec-land.md 4.2.1). The parts are rebuilt each
// round because Apportion sorts them in place.
//
// Requirement: NFR-LND-004
func BenchmarkApportion(b *testing.B) {
	b.ReportAllocs()
	const n = 200
	parts := make([]tenure.Part, n+1)
	out := make([]int64, n+1)
	for b.Loop() {
		for i := range n {
			parts[i] = tenure.Part{Ref: i, ID: int64(i + 1), Share: 40, Key: uint64(i * 31 % 97)}
		}
		parts[n] = tenure.Part{Ref: n, Share: 10_000 - 40*n, Commons: true}
		tenure.Apportion(5_003, parts, out)
	}
}

// BenchmarkTiers reports the cost of the tiers of an endowment of 50 holders and
// 400 requests with a scratch that has warmed up, so it shows the steady cost and
// allocations of a tick (spec-land.md 4.3).
//
// Requirement: NFR-LND-004
func BenchmarkTiers(b *testing.B) {
	b.ReportAllocs()
	var s tenure.Scratch
	holders := make([]tenure.Holder, 50)
	reqs := make([]tenure.Request, 400)
	for b.Loop() {
		for i := range holders {
			holders[i] = tenure.Holder{ID: int64(i + 1), Entitled: 20}
		}
		for i := range reqs {
			reqs[i] = tenure.Request{Ref: i, ID: int64(i + 1), Principal: int64(i%80 + 1), Executions: 3, Draw: 1, Key: uint64(i * 17 % 101), Balance: 1_000}
		}
		s.Tiers(tenure.Params{Stock: 2_000, Commons: 1_000, Rent: 4}, holders, reqs)
	}
}
