package policy_test

import (
	"testing"

	"github.com/Whealinghold/economy-sim/internal/policy"
)

// stipend is the lever of spec-policy.md 14.3: bounds 0 to 20 000, step limit
// 1 000, now at 11 000, a flow (no closed key, so every key is "open").
func stipend() policy.Lever {
	return policy.Lever{Min: 0, Max: 20_000, MaxStep: 1_000, Current: 11_000, AnyOpen: true}
}

// TestChecks_Example verifies the worked example of 14.3: [(stipend, 25 000),
// (stipend, 9 000), (lever 7, 1)] gives applied to 12 000 (clamped to 20 000,
// stepped to 11 000 + 1 000), duplicate and not_held, and Entries is the sum.
//
// Requirement: FR-PLC-040
func TestChecks_Example(t *testing.T) {
	t.Parallel()
	var s policy.Scratch
	tally := policy.Check([]policy.Lever{stipend()}, []policy.Entry{{Lever: 0, BP: 25_000}, {Lever: 0, BP: 9_000}, {Lever: 7, BP: 1}}, &s)
	want := policy.Tally{Entries: 3, Applied: 1, NotHeld: 1, Duplicate: 1, Clamped: 1, Stepped: 1}
	if tally != want {
		t.Errorf("tally = %+v, want %+v", tally, want)
	}
	rows := []policy.Result{
		{Outcome: policy.Applied, Value: 12_000, Clamped: true, Stepped: true},
		{Outcome: policy.Duplicate},
		{Outcome: policy.NotHeld},
	}
	for i, w := range rows {
		if s.Results[i] != w {
			t.Errorf("Results[%d] = %+v, want %+v", i, s.Results[i], w)
		}
	}
}

// TestChecks_OneOutcomePerEntry verifies that every entry gets exactly one
// outcome in the order of 6.2.1 and that Entries equals the sum of the five
// outcome counters, including a negative lever index and a duplicate of an entry
// that was itself refused.
//
// Requirement: FR-PLC-040
func TestChecks_OneOutcomePerEntry(t *testing.T) {
	t.Parallel()
	levers := []policy.Lever{stipend(), {Min: 0, Max: 100, Current: 50, AnyOpen: true}}
	entries := []policy.Entry{
		{Lever: -1, BP: 5},               // not held
		{Lever: 1, BP: 60, Closed: true}, // no_close (no MayClose, key open)
		{Lever: 1, BP: 70},               // duplicate of the refused entry
		{Lever: 0, BP: 11_000},           // unchanged
		{Lever: 2, BP: 0},                // not held
	}
	var s policy.Scratch
	tally := policy.Check(levers, entries, &s)
	want := policy.Tally{Entries: 5, NotHeld: 2, NoClose: 1, Duplicate: 1, Unchanged: 1}
	if tally != want {
		t.Errorf("tally = %+v, want %+v", tally, want)
	}
	if sum := tally.Applied + tally.Unchanged + tally.NotHeld + tally.Duplicate + tally.NoClose; sum != tally.Entries {
		t.Errorf("Entries = %d, sum of outcomes = %d", tally.Entries, sum)
	}
}

// TestCut_BoundsThenStep verifies the fixed order of the two cuts (6.2.2):
// bounds first, then Current plus or minus MaxStep. A value that the step cut
// alone would leave outside the bounds shows the order.
//
// Requirement: FR-PLC-041
func TestCut_BoundsThenStep(t *testing.T) {
	t.Parallel()
	l := policy.Lever{Min: 100, Max: 200, MaxStep: 30, Current: 190, AnyOpen: true}
	tests := []struct {
		name        string
		bp          int64
		want        int64
		clamped     bool
		stepped     bool
		wantOutcome policy.Outcome
	}{
		{"inside both", 195, 195, false, false, policy.Applied},
		{"above max is clamped, then inside the step", 500, 200, true, false, policy.Applied},
		{"inside bounds, beyond the step", 100, 160, false, true, policy.Applied},
		{"below min is clamped, then stepped", -1_000, 160, true, true, policy.Applied},
		{"equal to current", 190, 190, false, false, policy.Unchanged},
		{"clamped to the value it already has", 9_999, 200, true, false, policy.Applied},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var s policy.Scratch
			policy.Check([]policy.Lever{l}, []policy.Entry{{Lever: 0, BP: tt.bp}}, &s)
			got := s.Results[0]
			if got.Value != tt.want || got.Clamped != tt.clamped || got.Stepped != tt.stepped || got.Outcome != tt.wantOutcome {
				t.Errorf("Check(%d) = %+v, want value %d clamped %v stepped %v outcome %v", tt.bp, got, tt.want, tt.clamped, tt.stepped, tt.wantOutcome)
			}
		})
	}
}

// TestCut_OutsideBoundsMovesByStep verifies 6.2.2 for a current value that a
// consumer's call put outside the bounds: the bounds cut wins and the step is
// taken from Current toward the bound, by at most MaxStep a decision.
//
// Requirement: FR-PLC-041
func TestCut_OutsideBoundsMovesByStep(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		l    policy.Lever
		bp   int64
		want int64
	}{
		{"above max, asks for max", policy.Lever{Min: 0, Max: 20_000, MaxStep: 1_000, Current: 30_000, AnyOpen: true}, 20_000, 29_000},
		{"above max, asks for more than max", policy.Lever{Min: 0, Max: 20_000, MaxStep: 1_000, Current: 30_000, AnyOpen: true}, 25_000, 29_000},
		{"below min, asks for zero", policy.Lever{Min: 100, Max: 200, MaxStep: 30, Current: 0, AnyOpen: true}, 0, 30},
		{"above max, near enough to land inside", policy.Lever{Min: 0, Max: 20_000, MaxStep: 1_000, Current: 20_500, AnyOpen: true}, 0, 19_500},
		{"no step limit lands on the bound", policy.Lever{Min: 0, Max: 20_000, Current: 30_000, AnyOpen: true}, 99_999, 20_000},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var s policy.Scratch
			policy.Check([]policy.Lever{tt.l}, []policy.Entry{{Lever: 0, BP: tt.bp}}, &s)
			if got := s.Results[0]; got.Value != tt.want || got.Outcome != policy.Applied {
				t.Errorf("Check(%d) = %+v, want applied to %d", tt.bp, got, tt.want)
			}
		})
	}
}

// TestChecks_NoCloseRefusedWhole verifies check 3: closing a key without
// MayClose is refused whole (the number is not applied either), opening is never
// refused, and closing a key that is already closed is no closing, so a rule can
// move the factor of an edge a consumer closed and keep it closed.
//
// Requirement: FR-PLC-042
func TestChecks_NoCloseRefusedWhole(t *testing.T) {
	t.Parallel()
	edge := policy.Lever{Min: 10_000, Max: 40_000, Current: 10_000, AnyOpen: true}
	closedEdge := policy.Lever{Min: 10_000, Max: 40_000, Current: 10_000, AnyClosed: true}
	mayClose := policy.Lever{Min: 10_000, Max: 40_000, Current: 10_000, MayClose: true, AnyOpen: true}
	tests := []struct {
		name    string
		l       policy.Lever
		e       policy.Entry
		want    policy.Outcome
		wantVal int64
		wantTop bool
	}{
		{"closing without may_close is refused whole", edge, policy.Entry{BP: 20_000, Closed: true}, policy.NoClose, 0, false},
		{"closing with may_close applies", mayClose, policy.Entry{BP: 10_000, Closed: true}, policy.Applied, 10_000, true},
		{"opening a closed edge is never refused", closedEdge, policy.Entry{BP: 10_000, Closed: false}, policy.Applied, 10_000, false},
		{"keeping a consumer-closed edge closed moves the factor", closedEdge, policy.Entry{BP: 25_000, Closed: true}, policy.Applied, 25_000, true},
		{"keeping it closed at the same factor is unchanged", closedEdge, policy.Entry{BP: 10_000, Closed: true}, policy.Unchanged, 10_000, true},
		{"open edge, same number, open is unchanged", edge, policy.Entry{BP: 10_000}, policy.Unchanged, 10_000, false},
		{"a mixed group is a change even at the same number", policy.Lever{Min: 0, Max: 9, Current: 5, Mixed: true, AnyOpen: true}, policy.Entry{BP: 5}, policy.Applied, 5, false},
		{"a half-closed group may be opened with may_close off", policy.Lever{Min: 0, Max: 9, Current: 5, AnyOpen: true, AnyClosed: true}, policy.Entry{BP: 5}, policy.Applied, 5, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var s policy.Scratch
			policy.Check([]policy.Lever{tt.l}, []policy.Entry{tt.e}, &s)
			got := s.Results[0]
			if got.Outcome != tt.want {
				t.Fatalf("outcome = %v, want %v", got.Outcome, tt.want)
			}
			if tt.want == policy.NoClose {
				if got.Value != 0 || got.Clamped || got.Stepped {
					t.Errorf("a refused entry carries a value or a cut flag: %+v", got)
				}
				return
			}
			if got.Value != tt.wantVal || got.Closed != tt.wantTop {
				t.Errorf("result = %+v, want value %d closed %v", got, tt.wantVal, tt.wantTop)
			}
		})
	}
}

// TestChecks_ScratchIsReused verifies that a second call with the same scratch
// leaves no row of the first call and allocates nothing once warm.
//
// Requirement: NFR-PLC-001
func TestChecks_ScratchIsReused(t *testing.T) {
	var s policy.Scratch
	levers := []policy.Lever{stipend()}
	policy.Check(levers, []policy.Entry{{Lever: 0, BP: 12_000}, {Lever: 5}}, &s)
	entries := []policy.Entry{{Lever: 0, BP: 12_000}}
	tally := policy.Check(levers, entries, &s)
	if tally.Entries != 1 || tally.Applied != 1 || tally.NotHeld != 0 || len(s.Results) != 1 {
		t.Errorf("second call = %+v with %d rows, want one applied row", tally, len(s.Results))
	}
	if n := testing.AllocsPerRun(50, func() { policy.Check(levers, entries, &s) }); n != 0 {
		t.Errorf("Check allocates %v times a call once warm, want 0", n)
	}
}

// FuzzChecks fuzzes Check over one lever and a short list: it never panics, one
// row is returned per entry, Entries is the sum of the five outcome counters, and
// a value that was applied or left unchanged lies inside the bounds and within
// MaxStep of Current whenever Current is inside the bounds. The three low bits of
// flags are MayClose, AnyOpen and the first entry's Closed.
//
// Requirement: FR-PLC-040, FR-PLC-041
func FuzzChecks(f *testing.F) {
	f.Add(int64(0), int64(20_000), int64(1_000), int64(11_000), int64(25_000), int64(9_000), uint8(2))
	f.Add(int64(100), int64(200), int64(0), int64(0), int64(-5), int64(300), uint8(7))
	f.Fuzz(func(t *testing.T, lo, hi, step, cur, a, b int64, flags uint8) {
		if lo > hi || step < 0 || step > 1<<40 {
			t.Skip()
		}
		l := policy.Lever{Min: lo, Max: hi, MaxStep: step, Current: cur, MayClose: flags&1 != 0, AnyOpen: flags&2 != 0}
		entries := []policy.Entry{{Lever: 0, BP: a, Closed: flags&4 != 0}, {Lever: 0, BP: b}, {Lever: 3, BP: a}}
		var s policy.Scratch
		tl := policy.Check([]policy.Lever{l}, entries, &s)
		checkTally(t, tl, s.Results, len(entries))
		checkFirstRow(t, l, s.Results[0])
	})
}

// checkTally asserts the shape every Check result has.
func checkTally(t *testing.T, tl policy.Tally, rows []policy.Result, n int) {
	t.Helper()
	if len(rows) != n {
		t.Fatalf("%d rows for %d entries", len(rows), n)
	}
	if sum := tl.Applied + tl.Unchanged + tl.NotHeld + tl.Duplicate + tl.NoClose; sum != tl.Entries {
		t.Fatalf("Entries %d != sum %d", tl.Entries, sum)
	}
	if rows[1].Outcome != policy.Duplicate || rows[2].Outcome != policy.NotHeld {
		t.Fatalf("rows 1 and 2 = %v, %v; want Duplicate, NotHeld", rows[1].Outcome, rows[2].Outcome)
	}
}

// checkFirstRow asserts the bounds and the step limit on a kept value.
func checkFirstRow(t *testing.T, l policy.Lever, r policy.Result) {
	t.Helper()
	if r.Outcome != policy.Applied && r.Outcome != policy.Unchanged {
		return
	}
	if l.Current < l.Min || l.Current > l.Max {
		return
	}
	if r.Value < l.Min || r.Value > l.Max {
		t.Fatalf("value %d outside [%d, %d]", r.Value, l.Min, l.Max)
	}
	if l.MaxStep > 0 && (r.Value > policy.SatAdd(l.Current, l.MaxStep) || r.Value < policy.SatAdd(l.Current, -l.MaxStep)) {
		t.Fatalf("value %d beyond %d of %d", r.Value, l.MaxStep, l.Current)
	}
}
