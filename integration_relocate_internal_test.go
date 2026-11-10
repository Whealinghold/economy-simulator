package economysim

import (
	"bytes"
	"errors"
	"slices"
	"strings"
	"testing"
)

// The tests of Relocate (spec-integration.md 4.5, plan 2-17 Phase 6) are built on the
// fixtures of the owners: infoConfig with the migration module (three regions a, b
// and c; agents 3 and 4 in a, 5 and 6 in b, 7 and 8 in c), a journey planted through
// the module's own depart as the migration tests do, and the merchants fixture of
// the contract tests for the open contract. A second obligation is the test-only one
// of the registry of plan 3.

// relocationState is what a refused Relocate must leave: the checkpoint bytes and
// the number of events.
func relocationState(t *testing.T, w *World, rec *evLog) (b []byte, events int) {
	t.Helper()
	return bytesOf(t, w), len(rec.events)
}

// TestRelocate_RefusalOrder verifies the dry run of spec-integration.md 4.5.2 in
// REG's order: the region first (even for an agent that does not exist), then the
// agent (unknown, removed, or a pseudo-agent), then the same region and the
// obligation of a module. Each refusal is the sentinel of the owner, a removed agent
// is ErrAgentRemoved (which is also ErrUnknownAgent), and a refused call changes
// nothing: the checkpoint bytes are those before it and no event came.
//
// Requirement: FR-INT-031, FR-INT-030, FR-INT-005, FR-INT-002
func TestRelocate_RefusalOrder(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		id   AgentID
		to   RegionID
		want error
	}{
		{"region below range", 3, NoRegion, ErrUnknownRegion},
		{"region above range", 3, 9, ErrUnknownRegion},
		{"the region before the agent", 999, 9, ErrUnknownRegion},
		{"agent never issued", 999, infoB, ErrUnknownAgent},
		{"agent below range", NoAgent, infoB, ErrUnknownAgent},
		{"agent removed", 4, infoB, ErrAgentRemoved},
		{"agent removed is unknown too", 4, infoB, ErrUnknownAgent},
		{"an estate", 0, infoB, ErrNotRelocatable},
		{"same region", 3, infoA, ErrSameRegion},
		{"an obligation of a module", 5, infoA, ErrRegionBound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w, _, rec := migWorld(t, nil)
			w.bounds.add("open_hire", func(a *agent, _ []string) bool { return a.id == 5 })
			if err := w.removeAgent(&w.agents[4], CauseConsumer); err != nil {
				t.Fatalf("removeAgent() = %v, want nil", err)
			}
			before, n := relocationState(t, w, rec)
			if err := w.Relocate(tt.id, tt.to); !errors.Is(err, tt.want) {
				t.Fatalf("Relocate(%d, %d) = %v, want %v", tt.id, tt.to, err, tt.want)
			}
			after, m := relocationState(t, w, rec)
			if !bytes.Equal(before, after) || n != m {
				t.Errorf("a refused Relocate changed the world: bytes equal %v, %d events (was %d)", bytes.Equal(before, after), m, n)
			}
		})
	}
	t.Run("the obligation is named", func(t *testing.T) {
		t.Parallel()
		w, _, _ := migWorld(t, nil)
		w.bounds.add("open_hire", func(a *agent, _ []string) bool { return a.id == 5 })
		var rb *RegionBoundError
		if err := w.Relocate(5, infoA); !errors.As(err, &rb) || rb.Obligation != "open_hire" {
			t.Errorf("Relocate(5, a) = %v, want a *RegionBoundError naming open_hire", err)
		}
	})
	t.Run("the outside node", func(t *testing.T) {
		t.Parallel()
		w, rec := intWorld(t)
		before, n := relocationState(t, w, rec)
		household := AgentID(w.firstHousehold)
		if err := w.Relocate(household, regCoast); !errors.Is(err, ErrOutsideNode) {
			t.Errorf("Relocate(%d, the outside node) = %v, want ErrOutsideNode", household, err)
		}
		if after, m := relocationState(t, w, rec); !bytes.Equal(before, after) || n != m {
			t.Error("the refused move into the outside node changed the world")
		}
	})
	t.Run("the error names the call", func(t *testing.T) {
		t.Parallel()
		w, _, _ := migWorld(t, nil)
		if err := w.Relocate(3, infoA); err == nil || !strings.HasPrefix(err.Error(), "Relocate: agent 3") {
			t.Errorf("Relocate error = %v, want it to begin with the call's name and the agent", err)
		}
	})
}

// TestRelocate_Effects verifies what the call does to an external agent that has a
// job in progress and a sticky directive, in the order of spec-integration.md 4.5.2:
// the job is abandoned with the reason region_changed and then the account moves
// (AgentRelocated, cause consumer, the tick the next step runs), the beliefs are
// those of a fresh seeding at the destination with no draw, RoleSince is the tick,
// and the money, goods, role, weight, strategy, target ratio, residues and the
// mailbox are as they were. The audit passes and the run goes on.
//
// Requirement: FR-INT-031, FR-INT-009
func TestRelocate_Effects(t *testing.T) {
	t.Parallel()
	w, _, rec := migWorld(t, nil)
	id, err := w.AddAgent(NewAgent{Region: infoA, Role: 0, Money: 5_000, Stock: []StockItem{{Commodity: 0, Quantity: 4}}})
	if err != nil {
		t.Fatalf("AddAgent() = %v, want nil", err)
	}
	mailbox := Directive{Hold: []CommodityID{0}, Sticky: true}
	if err := w.Direct(id, mailbox); err != nil {
		t.Fatalf("Direct() = %v, want nil", err)
	}
	if err := w.Step(); err != nil { // a tick, so RoleSince 0 differs from the tick of the call
		t.Fatalf("Step() = %v, want nil", err)
	}
	a := &w.agents[id]
	a.job = job{recipe: 0, executions: 1, remaining: 2}
	a.hunger, a.privation = 2, 1
	role, weight, strategy, target, residues, dir := a.role, a.weight, a.strategy, a.target, slices.Clone(a.residues), a.directive
	money, units := balance2(t, w, id)
	rec.events = nil
	if err := w.Relocate(id, infoB); err != nil {
		t.Fatalf("Relocate(%d, b) = %v, want nil", id, err)
	}
	if got := regionNow(w, id); got != infoB {
		t.Errorf("region after Relocate = %d, want %d", got, infoB)
	}
	if a.hasJob() {
		t.Error("the job is still in progress after Relocate, want it abandoned")
	}
	var order []string
	for _, e := range rec.events {
		switch e := e.(type) {
		case RecipeAbandoned:
			order = append(order, "abandoned:"+string(e.Reason))
			if e.Agent != id || e.Region != infoA || e.Tick != w.Tick() {
				t.Errorf("RecipeAbandoned = %+v, want agent %d in region a at tick %d", e, id, w.Tick())
			}
		case AgentRelocated:
			order = append(order, "relocated:"+string(e.Cause))
			if e.Agent != id || e.From != infoA || e.To != infoB || e.Weight != weight || e.Role != role || e.Money != Money(money) || e.Units != units || e.Tick != w.Tick() {
				t.Errorf("AgentRelocated = %+v, want agent %d from a to b, weight %d, %d money, %d units at tick %d", e, id, weight, money, units, w.Tick())
			}
		default:
			t.Errorf("Relocate emitted %T, want only RecipeAbandoned and AgentRelocated", e)
		}
	}
	if want := []string{"abandoned:region_changed", "relocated:consumer"}; !slices.Equal(order, want) {
		t.Errorf("events of Relocate = %v, want %v", order, want)
	}
	if a.roleSince != w.Tick() {
		t.Errorf("roleSince = %d, want %d, the tick the next step runs", a.roleSince, w.Tick())
	}
	if a.role != role || a.weight != weight || a.strategy != strategy || a.target != target || !slices.Equal(a.residues, residues) || a.hunger != 2 || a.privation != 1 {
		t.Errorf("role %d weight %d strategy %d target %d residues %v hunger %d privation %d, want %d %d %d %d %v 2 1: Relocate changes none of them",
			a.role, a.weight, a.strategy, a.target, a.residues, a.hunger, a.privation, role, weight, strategy, target, residues)
	}
	if m, u := balance2(t, w, id); m != money || u != units {
		t.Errorf("money %d and goods %d after Relocate, want %d and %d: it moves no money and no goods", m, u, money, units)
	}
	if a.directive != dir || !a.directive.Sticky {
		t.Errorf("mailbox after Relocate = %+v, want the one before: a pending directive stays", a.directive)
	}
	// The beliefs are a fresh seeding at the destination: the same as an external
	// agent of the role added there, whose beliefs AddAgent seeds by the same rule.
	fresh, err := w.AddAgent(NewAgent{Region: infoB, Role: 0})
	if err != nil {
		t.Fatalf("AddAgent() = %v, want nil", err)
	}
	if !slices.Equal(a.beliefs, w.agents[fresh].beliefs) {
		t.Errorf("beliefs after Relocate = %+v, want those of a fresh seeding in b, %+v", a.beliefs, w.agents[fresh].beliefs)
	}
	if err := w.Audit(); err != nil {
		t.Errorf("Audit() = %v, want nil", err)
	}
	if err := w.Step(); err != nil {
		t.Errorf("Step() after Relocate = %v, want nil", err)
	}
}

// balance2 returns the money and the units of goods an agent holds.
func balance2(t *testing.T, w *World, id AgentID) (money, units int64) {
	t.Helper()
	money, _ = balance(t, w, id)
	units, err := w.ledger.Units(w.agents[id].account)
	if err != nil {
		t.Fatalf("Units(agent %d) = %v, want nil", id, err)
	}
	return money, units
}

// TestRelocate_OnJourney verifies the forced move of a migrant (spec-integration.md
// 4.5.2 step 2, INT-S5): the journey is dropped by the module's cancel and its
// households are counted in Ended, and the household is moved, with no
// MigrationStarted and no migration arrival; the ledger is not touched by the
// cancel. With a second obligation open, the call changes nothing and the journey
// stays.
//
// Requirement: FR-INT-031
func TestRelocate_OnJourney(t *testing.T) {
	t.Parallel()
	relocateOnJourney(t, 0)
}

// relocateOnJourney is the body of TestRelocate_OnJourney and of INT-S5 (S83) for one seed.
func relocateOnJourney(t *testing.T, seed uint64) {
	t.Helper()
	t.Run("the journey is dropped and the household moves", func(t *testing.T) {
		t.Parallel()
		w, _, rec := migWorldSeed(t, seed, nil)
		w.migration.depart(3, infoB, 9)
		weight := w.agents[3].weight
		rec.events = nil
		if err := w.Relocate(3, infoC); err != nil {
			t.Fatalf("Relocate(3, c) = %v, want nil", err)
		}
		if w.migration.on(3) || len(w.migration.active) != 0 || w.migration.ended[infoA] != weight {
			t.Errorf("after Relocate on(3) %v, active %v, ended %v, want none, none and %d in region a, where the account was", w.migration.on(3), w.migration.active, w.migration.ended, weight)
		}
		if got := regionNow(w, 3); got != infoC {
			t.Errorf("region after Relocate = %d, want %d, the target of the call and not the journey's destination", got, infoC)
		}
		moved := of[AgentRelocated](rec)
		if len(moved) != 1 || moved[0].Cause != CauseConsumer || moved[0].From != infoA || moved[0].To != infoC {
			t.Errorf("AgentRelocated = %+v, want one from a to c with the cause consumer", moved)
		}
		if len(of[MigrationStarted](rec)) != 0 {
			t.Error("Relocate emitted MigrationStarted, want none")
		}
		step(t, w, 12) // past the planted arrival tick: the dropped journey never arrives
		if got := regionNow(w, 3); got != infoC {
			t.Errorf("region after the planted arrival tick = %d, want %d: the journey is gone", got, infoC)
		}
		if err := w.Audit(); err != nil {
			t.Errorf("Audit() = %v, want nil", err)
		}
	})
	t.Run("another obligation keeps the journey", func(t *testing.T) {
		t.Parallel()
		w, _, rec := migWorldSeed(t, seed, nil)
		w.bounds.add("open_hire", func(a *agent, _ []string) bool { return a.id == 3 })
		w.migration.depart(3, infoB, 9)
		before, n := relocationState(t, w, rec)
		var rb *RegionBoundError
		if err := w.Relocate(3, infoC); !errors.As(err, &rb) || rb.Obligation != "open_hire" {
			t.Fatalf("Relocate(3, c) = %v, want ErrRegionBound naming open_hire, not on_journey", err)
		}
		if !w.migration.on(3) || w.migration.ended[infoA] != 0 {
			t.Errorf("after the refusal on(3) %v, ended %v, want the journey kept and none ended", w.migration.on(3), w.migration.ended)
		}
		if after, m := relocationState(t, w, rec); !bytes.Equal(before, after) || n != m {
			t.Error("the refused Relocate changed the world")
		}
	})
}

// TestRelocate_OpenContract verifies that the payer of an open delivery contract is
// refused with the obligation of the merchants module (spec-merchants.md 9.8.1),
// that the call and the query RelocationRefusal agree for every input
// (FR-REG-059), and that nothing changes.
//
// Requirement: FR-INT-031; INT-S5
func TestRelocate_OpenContract(t *testing.T) {
	t.Parallel()
	offer := DeliveryOffer{Mode: modeCoach, To: regFar, Commodity: cGrain, Quantity: 5, Limit: 10}
	plan := roundPlan{
		offers: map[AgentID][]DeliveryOffer{d32: {offer}},
		takes:  map[AgentID][]DeliveryTake{p61: {{Commodity: cGrain, Quantity: 10, Limit: 100}}},
	}
	w, rec, _ := roundWorld(t, trio, plan, d32)
	step(t, w, 1)
	if len(w.mer.open) == 0 {
		t.Fatal("no contract is open after the round, so the refusal cannot be observed")
	}
	before, n := relocationState(t, w, rec)
	for _, party := range []struct {
		name string
		id   AgentID
		to   RegionID
	}{{"payer", p61, regHome}, {"deliverer", d32, regFar}} {
		_, wantErr := w.RelocationRefusal(party.id, party.to)
		err := w.Relocate(party.id, party.to)
		var rb, wantRB *RegionBoundError
		if !errors.Is(err, ErrRegionBound) || !errors.As(err, &rb) || !errors.As(wantErr, &wantRB) || rb.Obligation != wantRB.Obligation {
			t.Errorf("Relocate(%s) = %v, want the refusal of the query, %v", party.name, err, wantErr)
		}
	}
	var rb *RegionBoundError
	if _, err := w.RelocationRefusal(p61, regHome, "open_shipment"); !errors.As(err, &rb) || rb.Obligation != "open_contract" {
		t.Errorf("RelocationRefusal(payer, skipping open_shipment) = %v, want ErrRegionBound naming open_contract", err)
	}
	if after, m := relocationState(t, w, rec); !bytes.Equal(before, after) || n != m {
		t.Error("a refused Relocate changed the world")
	}
}
