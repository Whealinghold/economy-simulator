package tenure_test

import (
	"testing"

	"github.com/Whealinghold/economy-sim/internal/tenure"
)

// TestAllocs_RulesAllocateNothingOnceWarm verifies that the rules a tick runs
// for an endowment with a title allocate nothing once the scratch has grown:
// the apportionment, the supply of the lessors, the tiers, the attribution, the
// charge and the step. Each run refills the inputs from a template, because the
// rules sort their slices in place, and uses a Scratch and a dst that have
// already grown.
//
// Requirement: NFR-LND-001
//
// Guard: the benchmarks of this package report allocations but never fail, so
// an allocation added to a per-tick rule would pass every gate and surface only
// as a slower tick far from its cause (observability-and-logging.md 8.2,
// testing.md 8.2).
func TestAllocs_RulesAllocateNothingOnceWarm(t *testing.T) {
	const holders, requests = 20, 120
	holderT := make([]tenure.Holder, holders)
	reqT := make([]tenure.Request, requests)
	for i := range holderT {
		holderT[i] = tenure.Holder{ID: int64(i + 1), Entitled: 5}
	}
	for i := range reqT {
		reqT[i] = tenure.Request{Ref: i, ID: int64(i + 1), Principal: int64(i%40 + 1), Executions: 2, Draw: 1, Key: uint64(i * 17 % 101), Balance: 500}
	}
	partT := make([]tenure.Part, holders+1)
	for i := range holders {
		partT[i] = tenure.Part{Ref: i, ID: int64(i + 1), Share: 40, Key: uint64(i * 31 % 97)}
	}
	partT[holders] = tenure.Part{Ref: holders, Share: 10_000 - 40*holders, Commons: true}
	takers := []tenure.Taker{{ID: 1, Units: 3}, {ID: 2, Units: 4}, {ID: 3, Units: 2}}
	lessors := []tenure.Taker{{ID: 4, Units: 5}, {ID: 1, Units: 4}}
	lines := []tenure.Line{{Rent: 4, Draw: 2, Own: 1, Commons: 1}, {Rent: 3, Draw: 1}}

	hs := make([]tenure.Holder, holders)
	rs := make([]tenure.Request, requests)
	parts := make([]tenure.Part, len(partT))
	out := make([]int64, len(partT))
	var s tenure.Scratch
	dst := make([]tenure.Triple, 0, 16)
	run := func() {
		copy(parts, partT)
		tenure.Apportion(1_003, parts, out)
		copy(parts, partT)
		tenure.Supply(40, parts[:holders], out)
		copy(hs, holderT)
		copy(rs, reqT)
		s.Tiers(tenure.Params{Stock: 400, Commons: 200, Rent: 4}, hs, rs)
		dst, _ = s.Attribute(dst[:0], takers, lessors)
		tenure.Charge(100, 12, lines)
		tenure.Step(tenure.StepIn{Posted: 40, Base: 40, StepBP: 500, FloorBP: 5_000, CeilBP: 20_000, Unserved: 3})
	}
	run() // grow the scratch
	if allocs := testing.AllocsPerRun(50, run); allocs != 0 {
		t.Errorf("one tick of the tenure rules allocates %v per run, want 0", allocs)
	}
}
