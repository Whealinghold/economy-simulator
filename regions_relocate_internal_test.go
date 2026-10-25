package economysim

import (
	"errors"
	"log/slog"
	"reflect"
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/ledger"
)

// relocWorld is the two-region world of removalConfig: agents 2 and 3 are the
// millers of home (region 0), 4 the weaver of home, 5 the miller of yonder
// (region 1); 0 and 1 are the estates. It has no module, so the table of
// obligations is empty unless a test adds one.
func relocWorld(t *testing.T, opts ...Option) *World {
	t.Helper()
	return newMill(t, removalConfig(), append([]Option{WithSeed(exitSeed)}, opts...)...)
}

// holdsShipment registers a fake obligation named open_shipment that agent 3
// holds, as two instances (keys 7 and 8): the shape of the real module's.
func holdsShipment(w *World) {
	w.bounds.add("open_shipment", func(a *agent, skip []string) bool {
		return a.id == 3 && (!skipHas(skip, "open_shipment", 7) || !skipHas(skip, "open_shipment", 8))
	})
}

// holdsStore registers a fake obligation named has_store, which has no instances:
// agent 3 holds it whatever skip says, and only a skip by name lifts it.
func holdsStore(w *World) {
	w.bounds.add("has_store", func(a *agent, _ []string) bool { return a.id == 3 })
}

// TestRelocationRefusal_Order verifies the order of the checks of 6.1.3: each row
// is returned when it is the first to fail, a later failing row is hidden by an
// earlier one, and the reason and the error are the row's.
//
// Requirement: FR-REG-039, FR-REG-043, FR-REG-044, FR-REG-059, FR-REG-047
func TestRelocationRefusal_Order(t *testing.T) {
	t.Parallel()
	reserve := func(w *World, id AgentID) {
		if err := w.ledger.ReserveMoney(w.agents[id].account, 5); err != nil {
			t.Fatalf("setup: ReserveMoney: %v", err)
		}
	}
	tests := []struct {
		name       string
		setup      func(w *World)
		id         AgentID
		to         RegionID
		skip       []string
		wantReason Reason
		wantErr    error
		// wantName is the obligation a region_bound refusal names.
		wantName string
	}{
		{name: "none", id: 2, to: 1},
		{name: "unknown region", id: 2, to: 2, wantReason: ReasonUnknownRegion, wantErr: ErrUnknownRegion},
		{name: "no region", id: 2, to: NoRegion, wantReason: ReasonUnknownRegion, wantErr: ErrUnknownRegion},
		// The region is row 1: it hides an unknown agent.
		{name: "unknown region before unknown agent", id: 99, to: 2, wantReason: ReasonUnknownRegion, wantErr: ErrUnknownRegion},
		{name: "unknown agent", id: 99, to: 1, wantReason: ReasonUnknownAgent, wantErr: ErrUnknownAgent},
		{name: "negative agent", id: -1, to: 1, wantReason: ReasonUnknownAgent, wantErr: ErrUnknownAgent},
		{name: "first ID not issued", id: 6, to: 1, wantReason: ReasonUnknownAgent, wantErr: ErrUnknownAgent},
		{name: "estate pseudo-agent", id: 0, to: 1, wantReason: ReasonNotRelocatable, wantErr: ErrNotRelocatable},
		{name: "estate of the destination", id: 1, to: 1, wantReason: ReasonNotRelocatable, wantErr: ErrNotRelocatable},
		{
			name:  "removed agent",
			setup: func(w *World) { removeForTest(t, w, 4) },
			id:    4, to: 1, wantReason: ReasonClosedAccount, wantErr: ErrClosedAccount,
		},
		{name: "same region", id: 2, to: 0, wantReason: ReasonSameRegion, wantErr: ErrSameRegion},
		{
			name:  "same region before reservation",
			setup: func(w *World) { reserve(w, 2) },
			id:    2, to: 0, wantReason: ReasonSameRegion, wantErr: ErrSameRegion,
		},
		{
			name:  "open reservation",
			setup: func(w *World) { reserve(w, 2) },
			id:    2, to: 1, wantReason: ReasonOpenReservation, wantErr: ErrOpenReservation,
		},
		{
			name:  "open reservation before region bound",
			setup: func(w *World) { holdsShipment(w); reserve(w, 3) },
			id:    3, to: 1, wantReason: ReasonOpenReservation, wantErr: ErrOpenReservation,
		},
		{
			name:  "region bound",
			setup: holdsShipment,
			id:    3, to: 1, wantReason: ReasonRegionBound, wantErr: ErrRegionBound, wantName: "open_shipment",
		},
		{
			name:  "obligation with no instances",
			setup: holdsStore,
			id:    3, to: 1, wantReason: ReasonRegionBound, wantErr: ErrRegionBound, wantName: "has_store",
		},
		{name: "obligation with no instances skipped by name", setup: holdsStore, id: 3, to: 1, skip: []string{"has_store"}},
		{
			name:  "obligation with no instances not skipped by a key",
			setup: holdsStore,
			id:    3, to: 1, skip: []string{"has_store:1"}, wantReason: ReasonRegionBound, wantErr: ErrRegionBound, wantName: "has_store",
		},
		{
			name:  "first registered obligation wins",
			setup: func(w *World) { holdsShipment(w); holdsStore(w) },
			id:    3, to: 1, wantReason: ReasonRegionBound, wantErr: ErrRegionBound, wantName: "open_shipment",
		},
		{
			name:  "second obligation when the first is skipped",
			setup: func(w *World) { holdsShipment(w); holdsStore(w) },
			id:    3, to: 1, skip: []string{"open_shipment"}, wantReason: ReasonRegionBound, wantErr: ErrRegionBound, wantName: "has_store",
		},
		{name: "region bound not held by another agent", setup: holdsShipment, id: 2, to: 1},
		{name: "region bound skipped by name", setup: holdsShipment, id: 3, to: 1, skip: []string{"open_shipment"}},
		{
			name:  "region bound skipped by both keys",
			setup: holdsShipment,
			id:    3, to: 1, skip: []string{"open_shipment:7", "open_shipment:8"},
		},
		{
			name:  "region bound held by the other key",
			setup: holdsShipment,
			id:    3, to: 1, skip: []string{"open_shipment:7"}, wantReason: ReasonRegionBound, wantErr: ErrRegionBound, wantName: "open_shipment",
		},
		{
			// A skip names obligations only: it cannot lift the reservation.
			name:  "skip cannot lift another refusal",
			setup: func(w *World) { reserve(w, 2) },
			id:    2, to: 1, skip: []string{"open_reservation", "same_region", "closed_account"},
			wantReason: ReasonOpenReservation, wantErr: ErrOpenReservation,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w := relocWorld(t)
			if tt.setup != nil {
				tt.setup(w)
			}
			reason, err := w.RelocationRefusal(tt.id, tt.to, tt.skip...)
			if reason != string(tt.wantReason) || !errors.Is(err, tt.wantErr) {
				t.Errorf("RelocationRefusal(%d, %d, %v) = %q, %v, want %q, %v", tt.id, tt.to, tt.skip, reason, err, tt.wantReason, tt.wantErr)
			}
			if tt.wantReason == ReasonRegionBound {
				var rb *RegionBoundError
				if !errors.As(err, &rb) || rb.Obligation != tt.wantName {
					t.Errorf("RelocationRefusal error = %v, want a *RegionBoundError naming %s", err, tt.wantName)
				}
			}
		})
	}
}

// removeForTest removes agent id as a step would, with the world in a step.
func removeForTest(t *testing.T, w *World, id AgentID) {
	t.Helper()
	if err := w.beginStep(); err != nil {
		t.Fatalf("setup: beginStep: %v", err)
	}
	if err := w.removeAgent(&w.agents[id], CauseInsolvent); err != nil {
		t.Fatalf("setup: removeAgent: %v", err)
	}
}

// TestSkipHas verifies the textual form of an instance key: a name alone matches
// every instance, "name:number" matches that number in canonical decimal only,
// and a key below 0 (an obligation with no instances) matches the name alone.
//
// Requirement: FR-REG-059, FR-REG-060
func TestSkipHas(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		skip []string
		key  int64
		want bool
	}{
		{"empty", nil, 7, false},
		{"name alone", []string{"on_journey"}, 7, true},
		{"name alone, no key", []string{"on_journey"}, -1, true},
		{"exact key", []string{"on_journey:7"}, 7, true},
		{"zero key", []string{"on_journey:0"}, 0, true},
		{"other key", []string{"on_journey:8"}, 7, false},
		{"key of a no-instance obligation", []string{"on_journey:7"}, -1, false},
		{"other name", []string{"open_hire:7"}, 7, false},
		{"name that is a prefix", []string{"on_journey_x:7"}, 7, false},
		{"name with no colon and a key", []string{"on_journey7"}, 7, false},
		{"empty number", []string{"on_journey:"}, 0, false},
		{"leading zero", []string{"on_journey:07"}, 7, false},
		{"plus sign", []string{"on_journey:+7"}, 7, false},
		{"minus sign", []string{"on_journey:-7"}, 7, false},
		{"hexadecimal", []string{"on_journey:0x7"}, 7, false},
		{"trailing text", []string{"on_journey:7a"}, 7, false},
		{"int64 maximum", []string{"on_journey:9223372036854775807"}, 1<<63 - 1, true},
		{"past int64", []string{"on_journey:9223372036854775808"}, 1<<63 - 1, false},
		{"digit then minus is not 7", []string{"on_journey:1-"}, 7, false},
		{"digit then colon is not 20", []string{"on_journey:1:"}, 20, false},
		{"letter is not 49", []string{"on_journey:a"}, 49, false},
		{"same length, no colon", []string{"xx_journey"}, 7, false},
		{"same length, other name", []string{"xx_journey:7"}, 7, false},
		{"second element matches", []string{"open_hire", "on_journey:7"}, 7, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := skipHas(tt.skip, "on_journey", tt.key); got != tt.want {
				t.Errorf("skipHas(%q, on_journey, %d) = %v, want %v", tt.skip, tt.key, got, tt.want)
			}
		})
	}
}

// TestRelocationRegistry_AddTwiceIsAnInvariant verifies that registering one
// obligation name twice is a violated invariant, and that the table of a world
// with no module is empty.
//
// Guard: the table is closed at construction by the modules that are on; two
// registrations of a name would make the refusal name the first one for both.
func TestRelocationRegistry_AddTwiceIsAnInvariant(t *testing.T) {
	t.Parallel()
	w := relocWorld(t)
	if n := len(w.bounds.rows); n != 0 {
		t.Fatalf("obligations of a world with no module = %d, want 0", n)
	}
	holdsShipment(w)
	if msg := invariantMessage(func() { holdsShipment(w) }); msg == "" {
		t.Errorf("adding open_shipment twice did not panic, want a violated invariant")
	}
}

// TestRelocationRegistry_MalformedIsAnInvariant verifies that an obligation with
// an empty name, a name with a colon or no test is a violated invariant at
// registration, and leaves the table unchanged.
//
// Guard: a colon would make a skip entry ambiguous, and a nil test would panic in
// the refusal path instead of at the module that registered it.
func TestRelocationRegistry_MalformedIsAnInvariant(t *testing.T) {
	t.Parallel()
	never := func(*agent, []string) bool { return false }
	cases := []struct {
		name string
		reg  string
		held func(*agent, []string) bool
	}{
		{"empty name", "", never},
		{"colon in name", "open:shipment", never},
		{"nil test", "open_shipment", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			w := relocWorld(t)
			if msg := invariantMessage(func() { w.bounds.add(tc.reg, tc.held) }); msg == "" {
				t.Errorf("add(%q) did not panic, want a violated invariant", tc.reg)
			}
			if n := len(w.bounds.rows); n != 0 {
				t.Errorf("obligations after the refused add = %d, want 0", n)
			}
		})
	}
}

// TestRelocationRefusal_AllocatesNothing verifies that the query allocates
// nothing for a success and for each refusal, with an obligation registered and
// with a skip by key.
//
// Requirement: FR-REG-059
func TestRelocationRefusal_AllocatesNothing(t *testing.T) {
	w := relocWorld(t)
	holdsShipment(w)
	if err := w.ledger.ReserveMoney(w.agents[2].account, 5); err != nil {
		t.Fatalf("setup: %v", err)
	}
	cases := []struct {
		id   AgentID
		to   RegionID
		skip []string
	}{{5, 0, nil}, {4, 1, nil}, {3, 1, nil}, {3, 1, []string{"open_shipment:7", "open_shipment:8"}}, {2, 1, nil}, {2, 0, nil}, {0, 1, nil}, {99, 1, nil}, {2, 9, nil}}
	for _, c := range cases {
		if got := testing.AllocsPerRun(50, func() { _, _ = w.RelocationRefusal(c.id, c.to, c.skip...) }); got != 0 {
			t.Errorf("RelocationRefusal(%d, %d, %v) allocates %v times, want 0", c.id, c.to, c.skip, got)
		}
	}
}

// TestRelocate_AgreesWithQuery verifies, for every agent ID from -1 to one past
// the last, every region from NoRegion to one past the last, with a reservation,
// an obligation and a removed agent in the world, that relocate returns exactly
// the error the query names and that a refused call changes nothing.
//
// Requirement: FR-REG-059, FR-REG-039
func TestRelocate_AgreesWithQuery(t *testing.T) {
	t.Parallel()
	build := func() *World {
		w := relocWorld(t)
		holdsShipment(w)
		if err := w.ledger.ReserveMoney(w.agents[2].account, 5); err != nil {
			t.Fatalf("setup: %v", err)
		}
		removeForTest(t, w, 4)
		return w
	}
	n := len(build().agents)
	for id := AgentID(-1); int(id) <= n; id++ {
		for to := NoRegion; to <= 2; to++ {
			w := build()
			_, want := w.RelocationRefusal(id, to)
			before := w.ledger.AppendRelocations(nil)
			got := w.relocate(id, to, CauseConsumer)
			if got != want {
				t.Errorf("relocate(%d, %d) = %v, want the query's %v", id, to, got, want)
			}
			if want != nil && !slices.Equal(w.ledger.AppendRelocations(nil), before) {
				t.Errorf("relocate(%d, %d) refused with %v but changed the relocation totals", id, to, want)
			}
		}
	}
}

// TestRelocate_AgentStateUnchanged verifies that a relocation leaves the whole
// agent record as it was (role, weight, strategy, job, beliefs, welfare ring,
// hunger, privation, directive, role-since), changes the region of the account
// only, moves the (region, role) counters of the agent, and shows in the agents
// of each region at once, each list in ascending agent ID.
//
// Requirement: FR-REG-045, FR-REG-048, FR-REG-040
func TestRelocate_AgentStateUnchanged(t *testing.T) {
	t.Parallel()
	w := relocWorld(t)
	a := &w.agents[2]
	a.job = job{recipe: 0, remaining: 1, executions: 2, principal: a.id}
	a.unfed, a.hunger, a.privation, a.roleSince = 1, 2, 3, 4
	a.directive = &Directive{Sticky: true}
	before := *a
	beliefs, residues, records := slices.Clone(a.beliefs), slices.Clone(a.residues), a.history.AppendRecords(nil)
	roles := len(w.cfg.Roles)
	cell := func(r RegionID) int64 { return w.live.inRole[int(r)*roles+int(a.role)] }
	home, yonder, agents, households := cell(0), cell(1), w.live.agents, w.live.households

	if err := w.relocate(2, 1, CauseConsumer); err != nil {
		t.Fatalf("relocate(2, 1) = %v, want nil", err)
	}

	if !reflect.DeepEqual(*a, before) {
		t.Errorf("agent after the relocation = %+v, want %+v", *a, before)
	}
	if !slices.Equal(a.beliefs, beliefs) || !slices.Equal(a.residues, residues) || !slices.Equal(a.history.AppendRecords(nil), records) {
		t.Errorf("beliefs, residues or welfare ring changed by the relocation")
	}
	if got := w.regionOf(a); got != 1 {
		t.Errorf("region of agent 2 = %d, want 1", got)
	}
	if cell(0) != home-1 || cell(1) != yonder+1 || w.live.agents != agents || w.live.households != households {
		t.Errorf("counters (home, yonder, agents, households) = %d, %d, %d, %d, want %d, %d, %d, %d", cell(0), cell(1), w.live.agents, w.live.households, home-1, yonder+1, agents, households)
	}
	// The weaver (agent 4) is the only one of its role, and yonder has had none.
	weaver := &w.agents[4]
	if w.state.hadAgent[1*roles+int(weaver.role)] {
		t.Fatalf("setup: yonder has had a %d already", weaver.role)
	}
	if err := w.relocate(4, 1, CauseConsumer); err != nil {
		t.Fatalf("relocate(4, 1) = %v, want nil", err)
	}
	if !w.state.hadAgent[1*roles+int(weaver.role)] {
		t.Errorf("hadAgent of (yonder, role %d) = false, want true: the destination cell has had an agent", weaver.role)
	}
	byRegion := map[RegionID][]AgentID{}
	for id := len(w.cfg.Regions); id < len(w.agents); id++ {
		r := w.regionOf(&w.agents[id])
		byRegion[r] = append(byRegion[r], AgentID(id))
	}
	if want := map[RegionID][]AgentID{0: {3}, 1: {2, 4, 5}}; !reflect.DeepEqual(byRegion, want) {
		t.Errorf("agents by region = %v, want %v", byRegion, want)
	}
	if err := w.ledger.Audit(); err != nil {
		t.Errorf("Audit() = %v, want nil", err)
	}
}

// TestRelocate_EventAndOrder verifies that one AgentRelocated is emitted per
// success with the figures the account held, after the move is complete (the
// observer reads the region already changed), and none for a refusal, for a
// pseudo-agent, or for a world with no observer, and that the Debug lines are
// written for the move and for the refusal.
//
// Arithmetic: agent 2 starts with weight 2 and 2 * 7 000 = 14 000 money, 2 * 10 =
// 20 grain and 2 * 1 = 2 pans (removalConfig), so Units is 22.
//
// Requirement: FR-REG-046, FR-REG-056, FR-REG-057
func TestRelocate_EventAndOrder(t *testing.T) {
	t.Parallel()
	var w *World
	var regionAtDelivery []RegionID
	rec := &evLog{}
	obs := ObserverFunc(func(e Event) {
		if _, ok := e.(AgentRelocated); ok {
			regionAtDelivery = append(regionAtDelivery, w.regionOf(&w.agents[2]))
		}
		rec.Observe(e)
	})
	cap := newCapture()
	w = relocWorld(t, WithObserver(obs), WithLogger(slog.New(cap)))
	rec.events = nil
	if err := w.relocate(2, 1, CauseTransport); err != nil {
		t.Fatalf("relocate(2, 1) = %v, want nil", err)
	}
	for _, refused := range []struct {
		id AgentID
		to RegionID
	}{{2, 1}, {0, 1}, {99, 1}, {2, 7}} {
		if err := w.relocate(refused.id, refused.to, CauseConsumer); err == nil {
			t.Fatalf("relocate(%d, %d) = nil, want a refusal", refused.id, refused.to)
		}
	}
	want := AgentRelocated{Tick: 0, Agent: 2, Role: w.agents[2].role, From: 0, To: 1, Weight: 2, Money: 14_000, Units: 22, Cause: CauseTransport}
	if got := of[AgentRelocated](rec); !slices.Equal(got, []AgentRelocated{want}) {
		t.Errorf("AgentRelocated events = %+v, want exactly %+v", got, want)
	}
	if !slices.Equal(regionAtDelivery, []RegionID{1}) {
		t.Errorf("region of agent 2 when the event was delivered = %v, want [1]: the event follows the move", regionAtDelivery)
	}
	// take forgets every line, so the refusals are copied first.
	all := slices.Clone(*cap.lines)
	moved := cap.take("agent relocated")
	*cap.lines = all
	if len(moved) != 1 || moved[0].attrs["agent"] != 2 || moved[0].attrs["from_region"] != 0 || moved[0].attrs["to_region"] != 1 ||
		moved[0].attrs["households"] != 2 || moved[0].strs["cause"] != "transport" || moved[0].level != slog.LevelDebug {
		t.Errorf("\"agent relocated\" lines = %+v, want one Debug line with agent 2, from 0, to 1, 2 households, cause transport", moved)
	}
	refused := cap.take("relocation refused")
	if len(refused) != 4 || refused[0].strs["reason"] != "same_region" || refused[1].strs["reason"] != "not_relocatable" ||
		refused[2].strs["reason"] != "unknown_agent" || refused[3].strs["reason"] != "unknown_region" || refused[3].attrs["to_region"] != 7 {
		t.Errorf("\"relocation refused\" lines = %+v, want same_region, not_relocatable, unknown_agent, unknown_region", refused)
	}
}

// TestRelocate_TotalOverflowIsRefusedAsOverflow verifies that a relocation whose
// pending totals would pass int64 is refused as ReasonOverflow with ErrOverflow
// by the query and by relocate, and changes nothing.
//
// Guard: the table of 6.1.3 has no row for it; the refusal keeps the query and
// relocate in agreement for a consumer that moves the same stock many times.
func TestRelocate_TotalOverflowIsRefusedAsOverflow(t *testing.T) {
	t.Parallel()
	w := relocWorld(t)
	// 6e17 is under MaxMoney (1e18); the (0, 1) row passes int64 (9.22e18) with its
	// sixteenth addition, which is the thirty-first move: the odd moves go 0 to 1.
	const stake = 600_000_000_000_000_000
	mint, ok := w.ledger.Boundary(ledger.Initial, ledger.Money)
	if !ok {
		t.Fatalf("setup: no Initial boundary account")
	}
	if err := w.ledger.Transfer(mint, w.agents[2].account, stake, ledger.Initial); err != nil {
		t.Fatalf("setup: Transfer: %v", err)
	}
	to := RegionID(1)
	for move := 1; move <= 30; move, to = move+1, 1-to {
		if err := w.relocate(2, to, CauseConsumer); err != nil {
			t.Fatalf("move %d: relocate(2, %d) = %v, want nil", move, to, err)
		}
	}
	reason, err := w.RelocationRefusal(2, to)
	if reason != string(ReasonOverflow) || !errors.Is(err, ErrOverflow) {
		t.Fatalf("RelocationRefusal(2, %d) at move 31 = %q, %v, want overflow, ErrOverflow", to, reason, err)
	}
	if got := w.relocate(2, to, CauseConsumer); got != err {
		t.Errorf("relocate(2, %d) = %v, want the query's %v", to, got, err)
	}
	if got := w.regionOf(&w.agents[2]); got != 1-to {
		t.Errorf("region of agent 2 = %d after the refusal, want %d", got, 1-to)
	}
}

// TestRelocate_CountsByRegionFromEvents verifies that the households of each
// region rebuilt from AgentSpawned, AgentBankrupted, WeightChanged and
// AgentRelocated (and RoleChanged, which keeps the region) equal the snapshot
// after relocations between steps of a run.
//
// Requirement: FR-OBS-040, FR-REG-046, FR-REG-045
func TestRelocate_CountsByRegionFromEvents(t *testing.T) {
	t.Parallel()
	rec := &evLog{}
	w := relocWorld(t, WithObserver(rec))
	moves := []struct {
		after int
		id    AgentID
		to    RegionID
	}{{0, 2, 1}, {2, 5, 0}, {3, 2, 0}, {5, 3, 1}}
	step := 0
	for _, m := range moves {
		for ; step < m.after; step++ {
			if err := w.Step(); err != nil {
				t.Fatalf("seed %d: Step() at tick %d = %v, want nil", exitSeed, w.Tick(), err)
			}
		}
		if err := w.relocate(m.id, m.to, CauseConsumer); err != nil {
			t.Fatalf("seed %d: relocate(%d, %d) after %d steps = %v, want nil", exitSeed, m.id, m.to, m.after, err)
		}
	}
	got := make([]int64, len(w.cfg.Regions))
	for _, e := range rec.events {
		switch e := e.(type) {
		case AgentSpawned:
			got[e.Region] += e.Weight
		case AgentBankrupted:
			got[e.Region] -= e.Weight
		case WeightChanged:
			got[e.Region] += e.To - e.From
		case AgentRelocated:
			got[e.From] -= e.Weight
			got[e.To] += e.Weight
		}
	}
	snap, err := w.Snapshot()
	if err != nil {
		t.Fatalf("seed %d: Snapshot() = %v, want nil", exitSeed, err)
	}
	for r, row := range snap.Regions {
		if got[r] != row.Households {
			t.Errorf("seed %d: households of region %d from the events = %d, snapshot says %d", exitSeed, r, got[r], row.Households)
		}
	}
	if n := len(of[AgentRelocated](rec)); n != len(moves) {
		t.Errorf("seed %d: %d AgentRelocated events, want %d", exitSeed, n, len(moves))
	}
}

// TestRelocate_NextTickPlacesOrdersInNewRegion verifies that an agent relocated
// between ticks places its orders in its new region's markets from the next
// tick on, and in none of the old region's.
//
// Requirement: FR-REG-045, FR-REG-039
func TestRelocate_NextTickPlacesOrdersInNewRegion(t *testing.T) {
	t.Parallel()
	rec := &evLog{}
	w := relocWorld(t, WithObserver(rec))
	if err := w.relocate(3, 1, CauseConsumer); err != nil {
		t.Fatalf("seed %d: relocate(3, 1) = %v, want nil", exitSeed, err)
	}
	rec.events = nil
	for range 3 {
		if err := w.Step(); err != nil {
			t.Fatalf("seed %d: Step() = %v, want nil", exitSeed, err)
		}
	}
	placed := 0
	for _, o := range of[OrderPlaced](rec) {
		if o.Agent != 3 {
			continue
		}
		placed++
		if o.Region != 1 {
			t.Errorf("seed %d: agent 3 placed an order in region %d at tick %d, want only region 1", exitSeed, o.Region, o.Tick)
		}
	}
	if placed == 0 {
		t.Errorf("seed %d: agent 3 placed no order in 3 ticks, so the test checks nothing", exitSeed)
	}
}

// moneyAndUnits returns the money and the units of goods over every account of
// the world that is open and not a boundary.
func moneyAndUnits(t *testing.T, w *World) (money, units int64) {
	t.Helper()
	for id := range w.ledger.Len() {
		v, err := w.ledger.Account(id)
		if err != nil {
			t.Fatalf("Account(%d) = %v, want nil", id, err)
		}
		if v.Closed || v.Kind == ledger.KindBoundary {
			continue
		}
		n, err := w.ledger.Units(id)
		if err != nil {
			t.Fatalf("Units(%d) = %v, want nil", id, err)
		}
		money, units = money+v.Balance, units+n
	}
	return money, units
}

// TestRelocate_Audit verifies, over a range of seeds, that relocations of
// pseudo-random agents to pseudo-random regions between ticks keep the audit
// clean, conserve the money and the goods exactly (a relocation moves nothing
// between accounts), and that a run with a logger and an observer ends in the
// same snapshot as one without.
//
// Requirement: FR-REG-040, FR-REG-057, FR-LED-054
func TestRelocate_Audit(t *testing.T) {
	t.Parallel()
	run := func(t *testing.T, seed uint64, opts ...Option) Snapshot {
		w := newMill(t, removalConfig(), append([]Option{WithSeed(seed)}, opts...)...)
		x := seed
		for tick := range 8 {
			for range 3 {
				x = x*6364136223846793005 + 1442695040888963407
				id, to := AgentID(x>>33)%AgentID(len(w.agents)+1), RegionID(x>>20)%3
				m0, u0 := moneyAndUnits(t, w)
				_ = w.relocate(id, to, CauseConsumer)
				if m1, u1 := moneyAndUnits(t, w); m1 != m0 || u1 != u0 {
					t.Fatalf("seed %d tick %d: money, units after relocate(%d, %d) = %d, %d, want %d, %d", seed, tick, id, to, m1, u1, m0, u0)
				}
			}
			if err := w.Step(); err != nil {
				t.Fatalf("seed %d: Step() at tick %d = %v, want nil", seed, w.Tick(), err)
			}
			if err := w.ledger.Audit(); err != nil {
				t.Fatalf("seed %d tick %d: Audit() = %v, want nil", seed, tick, err)
			}
		}
		snap, err := w.Snapshot()
		if err != nil {
			t.Fatalf("seed %d: Snapshot() = %v, want nil", seed, err)
		}
		return snap
	}
	for seed := uint64(1); seed <= 6; seed++ {
		plain := run(t, seed)
		watched := run(t, seed, WithObserver(&evLog{}), WithLogger(slog.New(newCapture())))
		if !reflect.DeepEqual(plain, watched) {
			t.Errorf("seed %d: snapshot with a logger and an observer differs from the one without", seed)
		}
	}
}

// TestRelocate_NoObserverAllocatesNothing verifies that a relocation, a move and
// its way back, allocates nothing with the default logger and observer once the
// relocation totals have their rows.
//
// Requirement: FR-REG-057
//
// Guard: the event is built only when an observer was supplied and the Debug
// lines are guarded; a boxed event or an attribute built unguarded would
// allocate on every relocation of a run.
func TestRelocate_NoObserverAllocatesNothing(t *testing.T) {
	w := relocWorld(t)
	for range 2 {
		for _, to := range []RegionID{1, 0} {
			if err := w.relocate(2, to, CauseConsumer); err != nil {
				t.Fatalf("setup: relocate(2, %d) = %v, want nil", to, err)
			}
		}
	}
	if got := testing.AllocsPerRun(50, func() {
		_ = w.relocate(2, 1, CauseConsumer)
		_ = w.relocate(2, 0, CauseConsumer)
	}); got != 0 {
		t.Errorf("a relocation and its way back allocate %v times, want 0", got)
	}
}
