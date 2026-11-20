package economysim

import "testing"

// policyGroupConfig is policyMillConfig with a third region, far, and a second jurisdiction
// below the crown, marsh (seat far), joined to the shire's seat by an edge pair. The shire's
// actor holds one lever, border, an Against group on marsh: the two edges between yonder
// and far, which it may close.
func policyGroupConfig(step ScheduleStep) Config {
	c := policyMillConfig()
	far := DefaultRegion()
	far.ID = "far"
	c.Regions = append(c.Regions, far)
	c.Edges = append(c.Edges, Edge{From: "yonder", To: "far", Distance: 10}, Edge{From: "far", To: "yonder", Distance: 10})
	c.Modules.Polity.Jurisdictions = append(c.Modules.Polity.Jurisdictions, Jurisdiction{ID: "marsh", Parent: "crown", Seat: "far", StartMoney: 1_000})
	c.Modules.Polity.Members = append(c.Modules.Polity.Members, Member{Region: "far", Jurisdiction: "marsh"})
	step.Lever = "border"
	c.Modules.Policy = &PolicyConfig{Actors: []PolicyActor{{
		ID: "shire_ctl", Jurisdiction: "shire", Policy: "schedule", Every: 1,
		Levers:   []Lever{{ID: "border", Kind: LeverEdge, Against: "marsh", Min: 5_000, Max: 20_000, MayClose: true}},
		Schedule: []ScheduleStep{step},
	}}}
	return c
}

// TestApply_GraphRebuiltOnce verifies that an Against group is applied as one batch: the
// two keys of the border change in one call of setEdgeStatuses, so the graph is built once
// for the whole application, both edges emit their event with the cause policy, and a
// second decision that returns the same value rebuilds nothing.
//
// Requirement: FR-PLC-008, FR-PLC-043
func TestApply_GraphRebuiltOnce(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name   string
		step   ScheduleStep
		factor int64
		closed bool
	}{
		{"a factor", ScheduleStep{At: 0, BP: 15_000}, 15_000, false},
		{"closed", ScheduleStep{At: 0, BP: 8_000, Closed: true}, 8_000, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			rec := &evLog{}
			w := newMill(t, policyGroupConfig(tt.step), WithSeed(3), WithObserver(rec))
			rec.events = nil
			if err := w.Step(); err != nil {
				t.Fatalf("Step() error = %v, want nil", err)
			}
			got := of[EdgeStatusChanged](rec)
			if len(got) != 2 || w.geo.rebuilds != 1 {
				t.Fatalf("after the first decision: %d edge events and %d rebuilds, want 2 and 1 (the group moves once)", len(got), w.geo.rebuilds)
			}
			want := EdgeStatusChanged{FactorBP: tt.factor, Closed: tt.closed, Cause: CausePolicy}
			for _, e := range got {
				e.From, e.To = 0, 0 // the pair is not the point here
				if e != want {
					t.Errorf("EdgeStatusChanged = %+v, want %+v", e, want)
				}
			}
			if err := w.Step(); err != nil {
				t.Fatalf("second Step() error = %v, want nil", err)
			}
			if n := len(of[EdgeStatusChanged](rec)); n != 2 || w.geo.rebuilds != 1 {
				t.Errorf("after an unchanged decision: %d edge events and %d rebuilds, want 2 and 1", n, w.geo.rebuilds)
			}
		})
	}
}
