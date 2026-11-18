package economysim

import (
	"errors"
	"math"
	"reflect"
	"slices"
	"testing"
)

// Class indexes of facConfig, in the order of its sorted IDs.
const (
	clsBarn FacilityClassID = iota
	clsDrain
	clsMill
	clsRoad
)

// facMiller is the first miller of home, a household agent of weight 1.
func facMiller(w *World) AgentID { return facAgents(w, 0, "miller")[0] }

// facIDs are the IDs of all the facilities of the world, in ascending order.
func facIDs(w *World) []FacilityID {
	var out []FacilityID
	for _, f := range w.infra.fac {
		out = append(out, f.id)
	}
	return out
}

// fcBrief is the part of a FacilityChanged that the tests compare: the facility, the kind
// and cause and the condition and effect around the change.
type fcBrief struct {
	ID                    FacilityID
	Kind                  FacilityKind
	Cause                 Cause
	Before, After, Effect int64
}

// briefs projects the FacilityChanged events of the log, in order.
func briefs(obs *evLog) []fcBrief {
	var out []fcBrief
	for _, e := range of[FacilityChanged](obs) {
		out = append(out, fcBrief{e.Facility, e.Kind, e.Cause, e.Before, e.After, e.Effect})
	}
	return out
}

// TestAddFacilities_CreatesInSliceOrderAndRebuilds verifies that the call creates the
// facilities with the next IDs in slice order, with the full durability for a condition of
// 0 and the given condition otherwise, emits FacilityChanged (added, consumer) for each in
// that order, rebuilds the effect tables at once (the road gives its gain to the edge
// before the next step) and returns the IDs.
//
// Arithmetic: a road of 6 000 at condition 500 of 2 000 gives floor(6 000 * 500 / 2 000) =
// 1 500; a mill of capacity 2 at condition 500 of 1 000 serves ceil(2 * 500 / 1 000) = 1.
//
// Requirement: FR-CAP-034, FR-CAP-005
func TestAddFacilities_CreatesInSliceOrderAndRebuilds(t *testing.T) {
	t.Parallel()
	obs := &evLog{}
	w := facWorld(t, nil, WithObserver(obs))
	m := facMiller(w)
	obs.events = nil
	ids, err := w.AddFacilities([]NewFacility{
		{Class: clsRoad, Region: 0, Toward: 1, Owner: 0, Condition: 500},
		{Class: clsMill, Region: 0, Toward: NoRegion, Owner: m},
		{Class: clsMill, Region: 0, Toward: NoRegion, Owner: m, Condition: 500},
	})
	if err != nil || !slices.Equal(ids, []FacilityID{1, 2, 3}) {
		t.Fatalf("AddFacilities = %v, %v, want IDs 1, 2, 3", ids, err)
	}
	want := []fcBrief{
		{1, FacilityAdded, CauseConsumer, 0, 500, 1_500},
		{2, FacilityAdded, CauseConsumer, 0, 1_000, 2},
		{3, FacilityAdded, CauseConsumer, 0, 500, 1},
	}
	if got := briefs(obs); !reflect.DeepEqual(got, want) {
		t.Errorf("events = %+v, want %+v", got, want)
	}
	n := w.infra
	gain, capacity := n.route[0*n.regions*n.modes+1*n.modes], n.site[siteKey{m, 0, clsMill}]
	if gain != 1_500 || capacity != 3 {
		t.Errorf("route gain home to yonder = %d and site capacity of the miller = %d, want 1500 and 2 + 1 = 3", gain, capacity)
	}
}

// TestAddFacilities_StoreInstalledAtTheCall verifies that a store facility is installed in
// the call: storage's StoreChanged (added, cause facility) comes first and then
// FacilityChanged (added, consumer) with the facility installed, and the owner holds the
// store.
//
// Requirement: FR-CAP-026, FR-CAP-034
func TestAddFacilities_StoreInstalledAtTheCall(t *testing.T) {
	t.Parallel()
	obs := &evLog{}
	w := facWorld(t, nil, WithObserver(obs))
	m := facMiller(w)
	before := w.sto.owned(m)
	obs.events = nil
	ids, err := w.AddFacilities([]NewFacility{{Class: clsBarn, Region: 0, Toward: NoRegion, Owner: m}})
	if err != nil {
		t.Fatalf("AddFacilities error = %v, want nil", err)
	}
	var seq []string
	for _, e := range obs.events {
		switch v := e.(type) {
		case StoreChanged:
			seq = append(seq, "store:"+string(v.Kind)+":"+string(v.Cause))
		case FacilityChanged:
			seq = append(seq, "facility:"+string(v.Kind))
		}
	}
	if want := []string{"store:added:facility", "facility:added"}; !reflect.DeepEqual(seq, want) {
		t.Errorf("events = %v, want %v", seq, want)
	}
	if st, _ := w.Facility(ids[0]); !st.Installed || w.sto.owned(m) != before+1 {
		t.Errorf("facility %+v and %d stores, want it installed and %d stores", st, w.sto.owned(m), before+1)
	}
}

// TestAddFacilities_Errors verifies one case for each error of 11.1, with the sentinel
// matched by errors.Is, the path naming the argument as a *ValidationError, and that a
// refused call changes nothing: the world has the facilities it had and emits nothing.
//
// Requirement: FR-CAP-034, FR-CAP-037
func TestAddFacilities_Errors(t *testing.T) {
	t.Parallel()
	base := NewFacility{Class: clsMill, Region: 0, Toward: NoRegion, Owner: 0}
	tests := []struct {
		name string
		mut  func(w *World, e *NewFacility)
		want error
	}{
		{"unknown class", func(_ *World, e *NewFacility) { e.Class = 9 }, ErrUnknownFacilityClass},
		{"unknown region", func(_ *World, e *NewFacility) { e.Region = 9 }, ErrUnknownRegion},
		{"toward on a site", func(_ *World, e *NewFacility) { e.Toward = 1 }, ErrFacilityShape},
		{"route toward itself", func(_ *World, e *NewFacility) { e.Class, e.Toward = clsRoad, 0 }, ErrSameRegion},
		{"route toward an unknown region", func(_ *World, e *NewFacility) { e.Class, e.Toward = clsRoad, 9 }, ErrUnknownRegion},
		{"condition above durability", func(_ *World, e *NewFacility) { e.Condition = 1_001 }, ErrOutOfBounds},
		{"negative condition", func(_ *World, e *NewFacility) { e.Condition = -1 }, ErrOutOfBounds},
		{"owner never issued", func(_ *World, e *NewFacility) { e.Owner = 999 }, ErrUnknownAgent},
		{"owner is another estate", func(_ *World, e *NewFacility) { e.Owner = 1 }, ErrFacilityOwner},
		{"store with the estate", func(_ *World, e *NewFacility) { e.Class = clsBarn }, ErrFacilityShape},
		{"store owner in another region", func(w *World, e *NewFacility) {
			e.Class, e.Owner = clsBarn, facAgents(w, 1, "miller")[0]
		}, ErrFacilityRegion},
		{"store owner with no base room", func(w *World, e *NewFacility) {
			e.Class, e.Owner = clsBarn, facAgents(w, 0, "weaver")[0]
		}, ErrNotLimited},
		{"ground with no endowment of the resource", func(_ *World, e *NewFacility) {
			e.Class, e.Region, e.Owner = clsDrain, 1, 1 // the estate of yonder
		}, ErrFacilityTarget},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			obs := &evLog{}
			w := facWorld(t, nil, WithObserver(obs))
			if tt.want == ErrFacilityTarget {
				w.res.capacity[w.res.cellOf(1, 0)] = 0 // yonder has no ore
			}
			e := base
			tt.mut(w, &e)
			obs.events = nil
			ids, err := w.AddFacilities([]NewFacility{{Class: clsMill, Region: 0, Toward: NoRegion, Owner: facMiller(w)}, e})
			if !errors.Is(err, tt.want) || ids != nil {
				t.Fatalf("AddFacilities = %v, %v, want %v and no IDs", ids, err, tt.want)
			}
			var ve *ValidationError
			if !errors.As(err, &ve) || len(ve.Path) == 0 || ve.Path[0] != "adds[1]" {
				t.Errorf("error %v is not a ValidationError of adds[1]", err)
			}
			if len(w.infra.fac) != 0 || len(obs.events) != 0 {
				t.Errorf("a refused call left %d facilities and %d events, want none: the first entry is valid and must not be applied", len(w.infra.fac), len(obs.events))
			}
		})
	}
}

// TestAddFacilities_ModuleOffAndGuards verifies ErrModuleOff with the module off and
// ErrInStep and ErrWorldFailed from the guard.
//
// Requirement: FR-CAP-034
func TestAddFacilities_ModuleOffAndGuards(t *testing.T) {
	t.Parallel()
	c := facConfig(nil)
	c.Modules.Infrastructure = nil
	off := newMill(t, c)
	if _, err := off.AddFacilities(nil); !errors.Is(err, ErrModuleOff) {
		t.Errorf("AddFacilities with the module off = %v, want ErrModuleOff", err)
	}
	w := facWorld(t, nil)
	w.stepping = true
	if _, err := w.AddFacilities(nil); !errors.Is(err, ErrInStep) {
		t.Errorf("AddFacilities inside a step = %v, want ErrInStep", err)
	}
	w.stepping, w.failed = false, errors.New("failed")
	if _, err := w.AddFacilities(nil); !errors.Is(err, ErrWorldFailed) {
		t.Errorf("AddFacilities on a failed world = %v, want ErrWorldFailed", err)
	}
}

// TestAddFacilities_Caps verifies ErrFacilityCap for an owner that would hold more than
// MaxFacilitiesPerAgent, counting the entries of the same call, and that the estate of a
// region has no per-owner bound.
//
// Requirement: FR-CAP-034, FR-CAP-005
func TestAddFacilities_Caps(t *testing.T) {
	t.Parallel()
	w := facWorld(t, nil)
	m := facMiller(w)
	adds := make([]NewFacility, MaxFacilitiesPerAgent+1)
	for i := range adds {
		adds[i] = NewFacility{Class: clsMill, Region: 0, Toward: NoRegion, Owner: m}
	}
	if _, err := w.AddFacilities(adds); !errors.Is(err, ErrFacilityCap) || len(w.infra.fac) != 0 {
		t.Errorf("AddFacilities of %d to one owner = %v with %d facilities, want ErrFacilityCap and none", len(adds), err, len(w.infra.fac))
	}
	// Fix: the per-owner breach named the world total and MaxFacilities; it names the
	// owner's count (64 held + 1 = 65 here) and MaxFacilitiesPerAgent.
	_, err := w.AddFacilities(adds)
	var ve *ValidationError
	if !errors.As(err, &ve) || ve.Value != itoa(MaxFacilitiesPerAgent+1) || ve.Bound != itoa(MaxFacilitiesPerAgent) {
		t.Errorf("AddFacilities per-owner breach = %+v, want value %d and bound %d", ve, MaxFacilitiesPerAgent+1, MaxFacilitiesPerAgent)
	}
	for i := range adds {
		adds[i].Owner = 0
	}
	if _, err := w.AddFacilities(adds); err != nil {
		t.Errorf("AddFacilities of %d to the estate = %v, want nil: the estate has no per-owner bound", len(adds), err)
	}
}

// TestAddFacilities_NextIDNearTheTop verifies that a call refuses to take IDs that would
// wrap past the largest int64 (a next ID restored near the top) and creates nothing.
//
// Requirement: FR-CAP-034
func TestAddFacilities_NextIDNearTheTop(t *testing.T) {
	t.Parallel()
	w := facWorld(t, nil)
	w.infra.next = math.MaxInt64 - 1
	adds := []NewFacility{
		{Class: clsMill, Region: 0, Toward: NoRegion, Owner: facMiller(w)},
		{Class: clsMill, Region: 0, Toward: NoRegion, Owner: facMiller(w)},
	}
	if _, err := w.AddFacilities(adds); !errors.Is(err, ErrFacilityCap) || len(w.infra.fac) != 0 {
		t.Errorf("AddFacilities near the top = %v with %d facilities, want ErrFacilityCap and none", err, len(w.infra.fac))
	}
	if _, err := w.AddFacilities(adds[:1]); err != nil {
		t.Errorf("AddFacilities of one entry with room for it = %v, want nil", err)
	}
}

// TestAdjustFacilities verifies the clamp to [0, Durability], that a result of 0 ruins the
// facility at once with the cause consumer and the record removed, that an entry that
// changes nothing emits nothing, that the changes are applied and reported in ascending
// facility ID whatever the slice order, and that the tables are rebuilt.
//
// Arithmetic: the mill has durability 1 000 and starts at 1 000, so a delta of 0 or +50
// clamps to 1 000 (no change), -400 gives 600 (it serves ceil(2 * 600 / 1 000) = 2) and
// -5 000 gives 0; the road at 2 000 with -1 000 gives 1 000 and floor(6 000 * 1 000 /
// 2 000) = 3 000, and +5 000 clamps to 2 000 and gives its whole 6 000.
//
// Requirement: FR-CAP-035
func TestAdjustFacilities(t *testing.T) {
	t.Parallel()
	obs := &evLog{}
	w := facWorld(t, func(m *Infrastructure) {
		m.Initial = []InitialFacility{{Region: "home", Class: "mill", Role: "miller", Count: 2}, {Region: "home", Class: "road", Toward: "yonder", Count: 1}}
	}, WithObserver(obs))
	ids := facIDs(w) // the road (class 3 sorts after the mills): IDs 1 and 2 are the mills, 3 the road
	obs.events = nil
	err := w.AdjustFacilities([]FacilityAdjustment{{ID: ids[2], Delta: -1_000}, {ID: ids[1], Delta: -5_000}, {ID: ids[0], Delta: -400}})
	if err != nil {
		t.Fatalf("AdjustFacilities error = %v, want nil", err)
	}
	want := []fcBrief{
		{ids[0], FacilityAdjusted, CauseConsumer, 1_000, 600, 2},
		{ids[1], FacilityRuined, CauseConsumer, 1_000, 0, 0},
		{ids[2], FacilityAdjusted, CauseConsumer, 2_000, 1_000, 3_000},
	}
	if got := briefs(obs); !reflect.DeepEqual(got, want) {
		t.Errorf("events = %+v, want %+v: ascending facility ID whatever the slice order", got, want)
	}
	_, gone := w.Facility(ids[1])
	if !errors.Is(gone, ErrUnknownFacility) || w.infra.route[1*w.infra.modes] != 3_000 {
		t.Errorf("Facility of a ruined facility = %v and route gain = %d, want ErrUnknownFacility (the record is removed) and floor(6000 * 1000 / 2000) = 3000", gone, w.infra.route[1*w.infra.modes])
	}
	obs.events = nil
	if err := w.AdjustFacilities([]FacilityAdjustment{{ID: ids[0], Delta: 0}, {ID: ids[2], Delta: 5_000}}); err != nil {
		t.Fatalf("AdjustFacilities of no change error = %v, want nil", err)
	}
	if got, want := briefs(obs), []fcBrief{{ids[2], FacilityAdjusted, CauseConsumer, 1_000, 2_000, 6_000}}; !reflect.DeepEqual(got, want) {
		t.Errorf("events of a no-op and a clamped rise = %+v, want only the road restored to 2000: %+v", got, want)
	}
}

// TestAdjustFacilities_Errors verifies ErrUnknownFacility, ErrOutOfBounds for a delta
// above MaxDurability and ErrDuplicateID, each atomic.
//
// Requirement: FR-CAP-035
func TestAdjustFacilities_Errors(t *testing.T) {
	t.Parallel()
	w := facWorld(t, func(m *Infrastructure) {
		m.Initial = []InitialFacility{{Region: "home", Class: "mill", Role: "miller", Count: 2}}
	})
	tests := []struct {
		name string
		in   []FacilityAdjustment
		want error
	}{
		{"unknown facility", []FacilityAdjustment{{ID: 1, Delta: -1}, {ID: 9}}, ErrUnknownFacility},
		{"delta too large", []FacilityAdjustment{{ID: 1, Delta: -1}, {ID: 2, Delta: MaxDurability + 1}}, ErrOutOfBounds},
		{"delta too small", []FacilityAdjustment{{ID: 2, Delta: -MaxDurability - 1}}, ErrOutOfBounds},
		{"duplicate", []FacilityAdjustment{{ID: 1, Delta: -1}, {ID: 2, Delta: -1}, {ID: 1, Delta: -2}}, ErrDuplicateID},
	}
	for _, tt := range tests {
		if err := w.AdjustFacilities(tt.in); !errors.Is(err, tt.want) {
			t.Errorf("%s: AdjustFacilities = %v, want %v", tt.name, err, tt.want)
		}
		for _, f := range w.infra.fac {
			if f.condition != 1_000 {
				t.Errorf("%s: a refused call changed facility %d to %d, want it untouched", tt.name, f.id, f.condition)
			}
		}
	}
}

// ledgerTotal is the sum of the balances of every account, the money a facility call
// must leave as it was.
func ledgerTotal(w *World) (n int64) {
	for i := range w.ledger.Len() {
		v, _ := w.ledger.Account(i)
		n += v.Balance
	}
	return n
}

// TestTransferFacilities verifies a transfer to another agent and to the estate of the
// facility's region: no money moves, the events are transferred by the consumer in
// ascending ID whatever the slice order, a transfer to the owner it has is no change, and
// the capacity table follows the owner.
//
// Requirement: FR-CAP-036, FR-CAP-027
func TestTransferFacilities(t *testing.T) {
	t.Parallel()
	obs := &evLog{}
	w := facWorld(t, func(m *Infrastructure) {
		m.Initial = []InitialFacility{{Region: "home", Class: "mill", Role: "miller", Count: 3}}
	}, WithObserver(obs))
	millers := facAgents(w, 0, "miller")
	money := ledgerTotal(w)
	obs.events = nil
	err := w.TransferFacilities([]FacilityTransfer{{ID: 2, To: 0}, {ID: 1, To: millers[1]}, {ID: 3, To: millers[2]}})
	if err != nil {
		t.Fatalf("TransferFacilities error = %v, want nil", err)
	}
	want := []fcBrief{
		{1, FacilityTransferred, CauseConsumer, 1_000, 1_000, 2},
		{2, FacilityTransferred, CauseConsumer, 1_000, 1_000, 2},
	}
	if got := briefs(obs); !reflect.DeepEqual(got, want) {
		t.Errorf("events = %+v, want %+v: facility 3 stays with its owner", got, want)
	}
	evs := of[FacilityChanged](obs)
	if evs[0].Owner != millers[1] || evs[1].Owner != 0 || ledgerTotal(w) != money {
		t.Errorf("new owners = %d and %d, money = %d, want miller %d, the estate and the %d the world held", evs[0].Owner, evs[1].Owner, ledgerTotal(w), millers[1], money)
	}
	n := w.infra
	tables := [3]int64{n.site[siteKey{millers[1], 0, clsMill}], n.site[siteKey{millers[0], 0, clsMill}], n.site[siteKey{0, 0, clsMill}]}
	if tables != [3]int64{2, 0, 2} || len(n.byOwner[millers[0]]) != 0 {
		t.Errorf("site table = %v and list of the first miller = %v, want capacities 2, 0 and 2 and an empty list", tables, n.byOwner[millers[0]])
	}
}

// TestTransferFacilities_Errors verifies one case for each error of 11.3 and that a refused
// call changes nothing.
//
// Requirement: FR-CAP-036
func TestTransferFacilities_Errors(t *testing.T) {
	t.Parallel()
	w := facWorld(t, func(m *Infrastructure) {
		m.Initial = []InitialFacility{{Region: "home", Class: "mill", Role: "miller", Count: 2}, {Region: "home", Class: "barn", Role: "miller", Count: 1}}
	})
	millers := facAgents(w, 0, "miller")
	barn, mill := facilityOf(w, clsBarn), facilityOf(w, clsMill)
	external, err := w.AddAgent(NewAgent{Region: 0})
	if err != nil {
		t.Fatalf("AddAgent error = %v, want nil", err)
	}
	if err := w.RemoveAgent(external); err != nil {
		t.Fatalf("RemoveAgent error = %v, want nil", err)
	}
	tests := []struct {
		name string
		in   []FacilityTransfer
		want error
	}{
		{"unknown facility", []FacilityTransfer{{ID: 99, To: millers[1]}}, ErrUnknownFacility},
		{"store facility", []FacilityTransfer{{ID: barn, To: millers[1]}}, ErrFacilityNotTransferable},
		{"agent never issued", []FacilityTransfer{{ID: mill, To: 999}}, ErrUnknownAgent},
		{"agent removed", []FacilityTransfer{{ID: mill, To: external}}, ErrAgentRemoved},
		{"another estate", []FacilityTransfer{{ID: mill, To: 1}}, ErrFacilityOwner},
		{"duplicate", []FacilityTransfer{{ID: mill, To: millers[1]}, {ID: mill, To: millers[2]}}, ErrDuplicateID},
	}
	before := slices.Clone(w.infra.fac)
	for _, tt := range tests {
		if err := w.TransferFacilities(tt.in); !errors.Is(err, tt.want) {
			t.Errorf("%s: TransferFacilities = %v, want %v", tt.name, err, tt.want)
		}
		if !slices.Equal(before, w.infra.fac) {
			t.Errorf("%s: a refused call changed the records", tt.name)
		}
	}
}

// TestTransferFacilities_OwnerCap verifies ErrFacilityCap for a new owner that would hold
// more than MaxFacilitiesPerAgent, counting the entries of the call.
//
// Requirement: FR-CAP-036
func TestTransferFacilities_OwnerCap(t *testing.T) {
	t.Parallel()
	w := facWorld(t, nil)
	m := facMiller(w)
	adds := make([]NewFacility, MaxFacilitiesPerAgent+1)
	for i := range adds {
		adds[i] = NewFacility{Class: clsMill, Region: 0, Toward: NoRegion, Owner: 0}
	}
	ids, err := w.AddFacilities(adds)
	if err != nil {
		t.Fatalf("AddFacilities error = %v, want nil", err)
	}
	moves := make([]FacilityTransfer, len(ids))
	for i, id := range ids {
		moves[i] = FacilityTransfer{ID: id, To: m}
	}
	if err := w.TransferFacilities(moves); !errors.Is(err, ErrFacilityCap) {
		t.Errorf("TransferFacilities of %d to one agent = %v, want ErrFacilityCap", len(moves), err)
	}
	if err := w.TransferFacilities(moves[:MaxFacilitiesPerAgent]); err != nil {
		t.Errorf("TransferFacilities of %d to one agent = %v, want nil", MaxFacilitiesPerAgent, err)
	}
}

// TestQueries_DeepCopy verifies Facility, Facilities and Agent(a).Facilities: copies in
// ascending ID that a change of the caller does not reach, no draw, no event and no change
// of the saved state.
//
// Requirement: FR-CAP-037
func TestQueries_DeepCopy(t *testing.T) {
	t.Parallel()
	obs := &evLog{}
	w := facWorld(t, facRows, WithObserver(obs))
	obs.events = nil
	before := errVal(w.Checkpoint())
	home := errVal(w.Facilities(0))
	if len(home) != 3+8+1+1 || !slices.IsSortedFunc(home, func(a, b FacilityState) int { return int(a.ID - b.ID) }) {
		t.Fatalf("Facilities(home) lists %d facilities, want 13 in ascending ID", len(home))
	}
	home[0].Condition = 1
	m := facMiller(w)
	st := errVal(w.Agent(m))
	st.Facilities[0].Condition = 1
	conditions := [2]int64{errVal(w.Facility(home[0].ID)).Condition, errVal(w.Agent(m)).Facilities[0].Condition}
	if conditions[0] == 1 || conditions[1] == 1 {
		t.Errorf("conditions after a change of the copies = %v, want the records unchanged", conditions)
	}
	if after := errVal(w.Checkpoint()); !reflect.DeepEqual(before, after) || len(obs.events) != 0 {
		t.Errorf("the queries changed the saved state or emitted %d events, want neither", len(obs.events))
	}
}

// TestFacilityQueries_Errors verifies the errors of the two queries: ErrInStep, ErrWorldFailed,
// ErrModuleOff, ErrUnknownFacility, ErrUnknownRegion and ErrOutsideNode.
//
// Requirement: FR-CAP-037
func TestFacilityQueries_Errors(t *testing.T) {
	t.Parallel()
	w := facWorld(t, facRows)
	if _, err := w.Facility(999); !errors.Is(err, ErrUnknownFacility) {
		t.Errorf("Facility(999) = %v, want ErrUnknownFacility", err)
	}
	if _, err := w.Facilities(9); !errors.Is(err, ErrUnknownRegion) {
		t.Errorf("Facilities(9) = %v, want ErrUnknownRegion", err)
	}
	w.stepping = true
	if _, err := w.Facility(1); !errors.Is(err, ErrInStep) {
		t.Errorf("Facility inside a step = %v, want ErrInStep", err)
	}
	w.stepping, w.failed = false, errors.New("failed")
	if _, err := w.Facilities(0); !errors.Is(err, ErrWorldFailed) {
		t.Errorf("Facilities on a failed world = %v, want ErrWorldFailed", err)
	}
	c := facConfig(nil)
	c.Modules.Infrastructure = nil
	if _, err := newMill(t, c).Facility(1); !errors.Is(err, ErrModuleOff) {
		t.Errorf("Facility with the module off = %v, want ErrModuleOff", err)
	}
}
