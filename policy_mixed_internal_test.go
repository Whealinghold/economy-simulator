package economysim

import "testing"

// The reading of a rule on an edge group whose keys disagree on Closed (plan 3-plc Phase 8, the
// open item of the PR B review; spec-policy.md 8.3.1). The rule sends the lever's Closed as the view
// has it, and the view's value of a group is its first key's (3.4.4), so a decision writes that flag
// to every key of the group.

// policyMixedConfig is policyGroupConfig with the actor's border moved by a rule on the shire's
// treasury (1 000 against a target of 0: a move of +1 000 a decision), which may close the border
// only when mayClose.
func policyMixedConfig(mayClose bool) Config {
	c := policyGroupConfig(ScheduleStep{})
	a := &c.Modules.Policy.Actors[0]
	a.Policy, a.Schedule = "rule", nil
	a.Levers[0].MayClose = mayClose
	a.Rules = []Reaction{{Lever: "border", Gauge: GaugeTreasury, Target: 0, Deadband: 0, Span: 1_000, Move: 1_000, Raise: true}}
	return c
}

// borderEdges returns the EdgeIDs of the border's two keys in ascending order and the world's
// view of each.
func borderEdges(t *testing.T, w *World) [2]EdgeView {
	t.Helper()
	var out [2]EdgeView
	l := w.policy.actors[0].levers[0]
	if len(l.edges) != 2 {
		t.Fatalf("the border holds %d edges, want 2", len(l.edges))
	}
	for i, id := range l.edges {
		out[i] = w.geo.g.view(int(id))
	}
	return out
}

// TestRule_MixedClosedGroupReadsTheFirstKey verifies the reading of 3.4.4 for a rule on a group a
// consumer split: the lever's flag is the first key's (the lower EdgeID), the rule's entry
// carries it, and the one application writes it to every key. With the first key closed and a
// border the actor may not close, the entry would close the open key, so the whole entry is
// refused as NoClose and nothing is written; with the first key open the rule reopens the key the
// consumer closed; with the first key closed and a border the actor may close, both end closed.
//
// Arithmetic: the shire's treasury is 1 000, the target 0, Span 1 000 and Move 1 000: the step is
// trunc(1 000 * 1 000 / 1 000) = +1 000 a decision. The decision of tick 0 takes the border from 10 000
// to 11 000 before the consumer closes a key, and the one of tick 5 takes it to 12 000 when its entry
// is applied and leaves 11 000 when it is refused.
//
// Requirement: FR-PLC-042, FR-PLC-008
func TestRule_MixedClosedGroupReadsTheFirstKey(t *testing.T) {
	t.Parallel()
	for _, tt := range []mixedCase{
		{"first key closed, may not close", false, 0, 1, 0, [2]bool{true, false}, 11_000},
		{"first key open, second closed by a consumer", false, 1, 0, 1, [2]bool{false, false}, 12_000},
		{"first key closed, may close", true, 0, 0, 1, [2]bool{true, true}, 12_000},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.run(t)
		})
	}
}

// mixedCase is one row of the table: the border's MayClose, the key the consumer closes (by index in
// ascending EdgeID), the outcomes counted by the decision of tick 5, the Closed flag of each key
// after it and the factor both keys end with.
type mixedCase struct {
	name      string
	mayClose  bool
	closeKey  int
	wantNo    int64
	wantApply int64
	wantShut  [2]bool
	wantBP    int64
}

// run is the case: a decision at tick 0, a consumer closing one key, a decision at tick 5.
func (tt mixedCase) run(t *testing.T) {
	t.Helper()
	rec := &evLog{}
	c := policyMixedConfig(tt.mayClose)
	c.Modules.Policy.Actors[0].Every = 5 // the first decision is at tick 0, the next at 5
	w := newMill(t, c, WithSeed(3), WithObserver(rec))
	stepUntil(t, w, 1) // tick 0: the rule's first decision, run before the split
	shut := borderEdges(t, w)[tt.closeKey]
	if err := w.SetEdgeStatuses([]EdgeStatus{{From: shut.From, To: shut.To, Closed: true, FactorBP: shut.FactorBP}}); err != nil {
		t.Fatalf("SetEdgeStatuses() error = %v, want nil", err)
	}
	rec.events = nil
	stepUntil(t, w, 6)
	got := of[PolicyDecided](rec)
	if len(got) != 1 || got[0].NoClose != tt.wantNo || got[0].Applied != tt.wantApply {
		t.Fatalf("PolicyDecided = %+v, want one with NoClose %d and Applied %d", got, tt.wantNo, tt.wantApply)
	}
	for i, k := range borderEdges(t, w) {
		if k.Closed != tt.wantShut[i] || k.FactorBP != tt.wantBP {
			t.Errorf("key %d after the decision = closed %v at %d, want closed %v at %d", i, k.Closed, k.FactorBP, tt.wantShut[i], tt.wantBP)
		}
	}
}

// stepUntil steps w until its tick is n.
func stepUntil(t *testing.T, w *World, n int64) {
	t.Helper()
	for w.Tick() < n {
		if err := w.Step(); err != nil {
			t.Fatalf("Step() error = %v, want nil", err)
		}
	}
}
