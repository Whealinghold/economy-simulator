package migration_test

import (
	"math"
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/migration"
)

// ranking is a one-candidate ranking to destination dest with journey time k.
func ranking(dest int, k int64) []migration.Ranked {
	return []migration.Ranked{{Ref: dest, K: k, VEff: 1}}
}

// agent is a household of weight 1 at origin 0 with a one-candidate ranking and a
// stock that provisions any journey.
func agent(ref int, earned int64, dest int, k int64) migration.Agent {
	return migration.Agent{Ref: ref, ID: int64(ref), Weight: 1, EarnedBP: earned, HoldTicks: math.MaxInt64, Ranked: ranking(dest, k)}
}

// outcomes returns the outcome of each Ref in the results.
func outcomes(rs []migration.Result) map[int]migration.Outcome {
	m := map[int]migration.Outcome{}
	for _, r := range rs {
		m[r.Ref] = r.Outcome
	}
	return m
}

// TestAdmit_OrderByNeed verifies that the agents are considered by earned ratio
// ascending and, for equal ratios, by agent ID ascending, whatever the input
// order, and that the poorest is the one admitted when the cap admits one.
//
// Requirement: FR-MIG-019
func TestAdmit_OrderByNeed(t *testing.T) {
	t.Parallel()
	mk := func(ref int, id, earned int64) migration.Agent {
		a := agent(ref, earned, 1, 1)
		a.ID = id
		return a
	}
	// Refs are the input positions; earned ratios 3 000, 1 000, 1 000, 2 000 and IDs
	// 40, 30, 20, 10: need order is (1 000, ID 20), (1 000, ID 30), 2 000, 3 000.
	agents := []migration.Agent{mk(0, 40, 3_000), mk(1, 30, 1_000), mk(2, 20, 1_000), mk(3, 10, 2_000)}
	caps := migration.Caps{Out: []int64{1}, In: []int64{0, 100}, Pipeline: []int64{0, 0}}
	var s migration.Scratch
	got := migration.Admit(nil, agents, caps, &s)
	var order []int
	for _, r := range got {
		order = append(order, r.Ref)
	}
	if want := []int{2, 1, 3, 0}; !slices.Equal(order, want) {
		t.Fatalf("order of consideration = %v, want %v", order, want)
	}
	if got[0].Outcome != migration.Started || got[0].Dest != 1 || got[0].K != 1 {
		t.Errorf("first result = %+v, want Started to 1 in 1 tick", got[0])
	}
	for _, r := range got[1:] {
		if r.Outcome != migration.OutCap {
			t.Errorf("result %+v, want OutCap after the cap of one is reached", r)
		}
	}
}

// TestAdmit_OutCapReachAdmitsOne verifies that a cap admits while the running
// count is below it: cap 3 admits the third and refuses the fourth, a cap of 0
// admits none, and the count of an agent that fails later tests is not spent.
//
// Requirement: FR-MIG-021
func TestAdmit_OutCapReachAdmitsOne(t *testing.T) {
	t.Parallel()
	var agents []migration.Agent
	for i := range 5 {
		agents = append(agents, agent(i, int64(i), 1, 1))
	}
	var s migration.Scratch
	caps := migration.Caps{Out: []int64{3}, In: []int64{0, 100}, Pipeline: []int64{0, 0}}
	got := outcomes(migration.Admit(nil, agents, caps, &s))
	want := map[int]migration.Outcome{0: migration.Started, 1: migration.Started, 2: migration.Started, 3: migration.OutCap, 4: migration.OutCap}
	for ref, o := range want {
		if got[ref] != o {
			t.Errorf("cap 3: agent %d outcome %d, want %d", ref, got[ref], o)
		}
	}
	caps.Out[0] = 0
	for ref, o := range outcomes(migration.Admit(nil, agents, caps, &s)) {
		if o != migration.OutCap {
			t.Errorf("cap 0: agent %d outcome %d, want OutCap", ref, o)
		}
	}
	// An unprovisioned agent spends no outflow: the first two are unprovisioned, the
	// next three fill a cap of 3.
	agents[0].HoldTicks, agents[1].HoldTicks = 0, 0
	caps.Out[0] = 3
	got = outcomes(migration.Admit(nil, agents, caps, &s))
	if got[0] != migration.Unprovisioned || got[1] != migration.Unprovisioned || got[2] != migration.Started || got[4] != migration.Started {
		t.Errorf("unprovisioned agents spent the outflow: %v", got)
	}
}

// TestAdmit_Provisioning verifies the test of MIG 15.1: an agent whose stock
// provisions 3 ticks is refused a journey of 4 (to lower) and passes a journey of
// 2 (to middle), and Hold computes those ticks from the stock.
//
// Requirement: FR-MIG-022
func TestAdmit_Provisioning(t *testing.T) {
	t.Parallel()
	// 6 staple against 2 a tick for one household is 3 ticks; 3 fuel against 1 is
	// 3; the least is 3.
	staple, ok1 := migration.Hold(6, 1, 2)
	fuel, ok2 := migration.Hold(3, 1, 1)
	if !ok1 || !ok2 || staple != 3 || fuel != 3 {
		t.Fatalf("Hold = %d, %v and %d, %v, want 3 and 3", staple, ok1, fuel, ok2)
	}
	tests := []struct {
		name    string
		ranking []migration.Ranked
		want    migration.Outcome
		dest    int
	}{
		{"lower alone needs 4 ticks", []migration.Ranked{{Ref: 0, K: 4}}, migration.Unprovisioned, 0},
		{"middle alone needs 2", []migration.Ranked{{Ref: 1, K: 2}}, migration.Started, 1},
		{"lower first, then middle", []migration.Ranked{{Ref: 0, K: 4}, {Ref: 1, K: 2}}, migration.Started, 1},
		{"exactly 3 ticks passes", []migration.Ranked{{Ref: 0, K: 3}}, migration.Started, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a := migration.Agent{Ref: 7, ID: 7, Weight: 1, HoldTicks: 3, Ranked: tt.ranking}
			caps := migration.Caps{Out: []int64{10}, In: []int64{10, 10}, Pipeline: []int64{0, 0}}
			var s migration.Scratch
			got := migration.Admit(nil, []migration.Agent{a}, caps, &s)
			if len(got) != 1 || got[0].Outcome != tt.want || (tt.want == migration.Started && got[0].Dest != tt.dest) {
				t.Errorf("Admit = %+v, want %d to %d", got, tt.want, tt.dest)
			}
		})
	}
	if got, ok := migration.Hold(6, 0, 2); ok || got != 0 {
		t.Errorf("Hold(6, 0, 2) = %d, %v, want 0, false (a zero divisor)", got, ok)
	}
	if got, ok := migration.Hold(6, math.MaxInt64, 2); ok || got != 0 {
		t.Errorf("Hold(6, MaxInt64, 2) = %d, %v, want 0, false (an overflowing divisor)", got, ok)
	}
	if got, ok := migration.Hold(0, 1, 1); !ok || got != 0 {
		t.Errorf("Hold(0, 1, 1) = %d, %v, want 0, true (nothing held)", got, ok)
	}
	if got, ok := migration.Hold(-5, 1, 2); !ok || got != 0 {
		t.Errorf("Hold(-5, 1, 2) = %d, %v, want 0, true", got, ok)
	}
}

// TestAdmit_InCapCountsEarlierAdmissions verifies that the pipeline of a
// destination counts the households already on the way and those admitted earlier
// in the pass, that the household that reaches the cap is admitted, and that a
// full destination falls through to the next candidate.
//
// Requirement: FR-MIG-023
func TestAdmit_InCapCountsEarlierAdmissions(t *testing.T) {
	t.Parallel()
	// Destination 1 has a pipeline cap of 10 and 8 on the way: two more are
	// admitted (the second reaches the cap), the third is refused there and goes
	// to destination 2.
	agents := []migration.Agent{
		{Ref: 0, ID: 0, Weight: 1, EarnedBP: 0, HoldTicks: 9, Ranked: []migration.Ranked{{Ref: 1, K: 2}, {Ref: 2, K: 2}}},
		{Ref: 1, ID: 1, Weight: 1, EarnedBP: 1, HoldTicks: 9, Ranked: []migration.Ranked{{Ref: 1, K: 2}, {Ref: 2, K: 2}}},
		{Ref: 2, ID: 2, Weight: 1, EarnedBP: 2, HoldTicks: 9, Ranked: []migration.Ranked{{Ref: 1, K: 2}, {Ref: 2, K: 2}}},
		{Ref: 3, ID: 3, Weight: 1, EarnedBP: 3, HoldTicks: 9, Ranked: []migration.Ranked{{Ref: 1, K: 2}}},
	}
	caps := migration.Caps{Out: []int64{100}, In: []int64{0, 10, 10}, Pipeline: []int64{0, 8, 0}}
	var s migration.Scratch
	got := migration.Admit(nil, agents, caps, &s)
	want := []migration.Result{
		{Ref: 0, Outcome: migration.Started, Dest: 1, K: 2},
		{Ref: 1, Outcome: migration.Started, Dest: 1, K: 2},
		{Ref: 2, Outcome: migration.Started, Dest: 2, K: 2},
		{Ref: 3, Outcome: migration.InCap},
	}
	if !slices.Equal(got, want) {
		t.Errorf("Admit = %+v, want %+v", got, want)
	}
	if caps.Pipeline[1] != 8 {
		t.Errorf("the caller's Pipeline was changed to %v", caps.Pipeline)
	}
	caps.In[1] = 0 // InShare 0: nobody may enter.
	got = migration.Admit(got, agents[3:], caps, &s)
	if len(got) != 1 || got[0].Outcome != migration.InCap {
		t.Errorf("Admit with a closed destination = %+v, want InCap", got)
	}
}

// TestAdmit_OrderOfTests verifies the fixed order of the tests: the guard before
// the outflow cap, the cap before provisioning, Unprovisioned only when every
// candidate failed provisioning and InCap when at least one passed it.
//
// Requirement: FR-MIG-024
func TestAdmit_OrderOfTests(t *testing.T) {
	t.Parallel()
	full := []int64{0, 0, 0} // pipeline caps of 0: every destination is full
	tests := []struct {
		name  string
		agent migration.Agent
		out   int64
		in    []int64
		want  migration.Outcome
	}{
		{"guard before cap", migration.Agent{Bound: true, HoldTicks: 0, Ranked: ranking(1, 1)}, 0, []int64{0, 9, 9}, migration.Bound},
		{"cap before provisioning", migration.Agent{HoldTicks: 0, Ranked: ranking(1, 1)}, 0, []int64{0, 9, 9}, migration.OutCap},
		{"provisioning before pipeline cap", migration.Agent{HoldTicks: 0, Ranked: ranking(1, 1)}, 5, full, migration.Unprovisioned},
		{"all fail provisioning", migration.Agent{HoldTicks: 1, Ranked: []migration.Ranked{{Ref: 1, K: 2}, {Ref: 2, K: 3}}}, 5, full, migration.Unprovisioned},
		{"one passes provisioning and is full", migration.Agent{HoldTicks: 2, Ranked: []migration.Ranked{{Ref: 1, K: 2}, {Ref: 2, K: 3}}}, 5, full, migration.InCap},
		{"one fails provisioning and the other is open", migration.Agent{HoldTicks: 2, Ranked: []migration.Ranked{{Ref: 1, K: 3}, {Ref: 2, K: 2}}}, 5, []int64{0, 0, 9}, migration.Started},
		{"no ranking", migration.Agent{HoldTicks: 9}, 5, full, migration.NoDestination},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.agent.Weight = 1
			caps := migration.Caps{Out: []int64{tt.out}, In: tt.in, Pipeline: []int64{0, 0, 0}}
			var s migration.Scratch
			got := migration.Admit(nil, []migration.Agent{tt.agent}, caps, &s)
			if len(got) != 1 || got[0].Outcome != tt.want {
				t.Errorf("Admit = %+v, want outcome %d", got, tt.want)
			}
		})
	}
}

// TestAdmit_PartitionExample verifies the accounting of MIG 15.1: of 12 pushed
// households of upper, 3 end no_gain at the stages (Rank), and of the 9 ranked,
// 2 are unprovisioned, 4 are refused by the outflow cap of 3 and 3 start, so the
// outcomes add to 12.
//
// Requirement: FR-MIG-025
func TestAdmit_PartitionExample(t *testing.T) {
	t.Parallel()
	count := map[migration.Outcome]int{}
	for range 3 {
		_, o, _ := migration.Rank(nil, []migration.Candidate{{Ref: 1, Priced: true, K: 2, Vhat: 9_000}}, riverParams)
		count[o]++
	}
	// Need order by earned ratio: a, b start (2 of 3); c, d are unprovisioned (stock
	// for 1 tick, the journey is 2) and spend no outflow; e is the third start; f to
	// i find the cap of 3 reached.
	hold := func(ref int, h int64) migration.Agent {
		a := agent(ref, int64(ref), 1, 2)
		a.HoldTicks = h
		return a
	}
	agents := []migration.Agent{hold(0, 9), hold(1, 9), hold(2, 1), hold(3, 1), hold(4, 9), hold(5, 9), hold(6, 9), hold(7, 9), hold(8, 9)}
	// ceil(600 * 50 / 10 000) = 3 and ceil(1 000 * 100 / 10 000) = 10.
	out, _ := migration.CapOf(600, 50)
	in, _ := migration.CapOf(1_000, 100)
	var s migration.Scratch
	for _, r := range migration.Admit(nil, agents, migration.Caps{Out: []int64{out}, In: []int64{0, in}, Pipeline: []int64{0, 3}}, &s) {
		count[r.Outcome]++
	}
	want := map[migration.Outcome]int{migration.NoGain: 3, migration.Unprovisioned: 2, migration.OutCap: 4, migration.Started: 3}
	total := 0
	for o, n := range want {
		total += n
		if count[o] != n {
			t.Errorf("outcome %d: %d households, want %d", o, count[o], n)
		}
	}
	if sum := count[migration.NoGain] + count[migration.Unprovisioned] + count[migration.OutCap] + count[migration.Started]; sum != 12 || total != 12 {
		t.Errorf("outcomes add to %d, want 12", sum)
	}
}

// TestAdmit_CohortNotSplit verifies that an agent of weight 50 is admitted whole
// at a cap not yet reached, and that the running counts then pass the cap by
// that weight, so the next agent is refused.
//
// Requirement: FR-MIG-043
func TestAdmit_CohortNotSplit(t *testing.T) {
	t.Parallel()
	big := agent(0, 0, 1, 1)
	big.Weight = 50
	next := agent(1, 1, 1, 1)
	caps := migration.Caps{Out: []int64{3}, In: []int64{0, 3}, Pipeline: []int64{0, 0}}
	var s migration.Scratch
	got := migration.Admit(nil, []migration.Agent{big, next}, caps, &s)
	if got[0].Outcome != migration.Started || got[1].Outcome != migration.OutCap {
		t.Errorf("Admit = %+v, want the cohort of 50 started whole and the next refused by the outflow cap", got)
	}
}

// TestAdmit_RunningCountsSaturate verifies that a weight that would overflow the
// running count refuses the next agent instead of wrapping.
//
// Requirement: NFR-MIG-003
func TestAdmit_RunningCountsSaturate(t *testing.T) {
	t.Parallel()
	// The first leaves the count one below the cap, so the second is admitted and
	// its weight would pass int64: the count saturates at the cap.
	huge := agent(0, 0, 1, 1)
	huge.Weight = math.MaxInt64 - 1
	again := agent(1, 1, 1, 1)
	again.Weight = math.MaxInt64
	third := agent(2, 2, 1, 1)
	caps := migration.Caps{Out: []int64{math.MaxInt64}, In: []int64{0, math.MaxInt64}, Pipeline: []int64{0, 0}}
	var s migration.Scratch
	got := migration.Admit(nil, []migration.Agent{huge, again, third}, caps, &s)
	if got[0].Outcome != migration.Started || got[1].Outcome != migration.Started || got[2].Outcome != migration.OutCap {
		t.Errorf("Admit = %+v, want two started and the third refused after saturation", got)
	}
}

// TestCapOf_CeilingAndOverflow verifies ceil(households * share / 10 000), the
// share 0 that admits nobody, and the overflow report.
//
// Requirement: FR-MIG-021
func TestCapOf_CeilingAndOverflow(t *testing.T) {
	t.Parallel()
	tests := []struct {
		households, share int64
		want              int64
		wantOK            bool
	}{
		{600, 50, 3, true},     // 600 * 50 / 10 000 = 3 exactly
		{1_000, 100, 10, true}, // 10 exactly
		{601, 50, 4, true},     // 3.005 rounds up
		{1, 1, 1, true},        // any share of a household is one place
		{600, 0, 0, true},
		{0, 5_000, 0, true},
		{math.MaxInt64, math.MaxInt64, 0, false},
	}
	for _, tt := range tests {
		if got, ok := migration.CapOf(tt.households, tt.share); got != tt.want || ok != tt.wantOK {
			t.Errorf("CapOf(%d, %d) = %d, %v, want %d, %v", tt.households, tt.share, got, ok, tt.want, tt.wantOK)
		}
	}
}
