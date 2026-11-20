package economysim

import (
	"go/parser"
	"go/token"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/ordering"
)

// The tests of the world side of the policy module (spec-policy.md 7; plan 3-plc Phase
// 4): the position, the actors the world resolves, the due test and the interface. The
// view, the checks and the application are tested in policy_apply_internal_test.go.

// policyMillConfig is fiscalMillConfig with the transport module, an edge pair between the
// two regions and the policy module m on: the crown (governed, seat home) holds the head
// tax poll and the transit factor of the road from home to yonder, the shire (governed,
// seat yonder, below the crown) holds the edge from home to yonder.
func policyMillConfig(actors ...PolicyActor) Config {
	c := fiscalMillConfig(&Fiscal{Taxes: []Tax{{ID: "poll", Authority: "crown", Target: homeMillers, Amount: 5}}})
	c.Edges = []Edge{{From: "home", To: "yonder", Distance: 10}, {From: "yonder", To: "home", Distance: 10}}
	c.Modules.Transport = &Transport{Modes: []Mode{{ID: "foot", Speed: 20, UnitMass: 40, UnitBulk: 30, Crew: DefaultCrew}}}
	c.Modules.Policy = &PolicyConfig{Actors: actors}
	return c
}

// crownActor is the actor of the crown: the edge from home to yonder (which it may close),
// the flow poll and the transit factor of the road, by LeverID 0, 1 and 2.
func crownActor(policy string, every int64) PolicyActor {
	return PolicyActor{
		ID: "crown_ctl", Jurisdiction: "crown", Policy: policy, Every: every,
		Levers: []Lever{
			{ID: "gate", Kind: LeverEdge, From: "home", To: "yonder", Min: 5_000, Max: 20_000, MayClose: true},
			{ID: "poll", Kind: LeverFlow, Flow: "poll", Min: 0, Max: 20_000, MaxStep: 3_000},
			{ID: "road", Kind: LeverTransit, From: "home", To: "yonder", Mode: "foot", Min: 0, Max: 20_000},
		},
	}
}

// shireActor is the actor of the shire: the edge from yonder to home, which it may not close.
func shireActor(policy string, every int64) PolicyActor {
	return PolicyActor{
		ID: "shire_ctl", Jurisdiction: "shire", Policy: policy, Every: every,
		Levers: []Lever{{ID: "back", Kind: LeverEdge, From: "yonder", To: "home", Min: 5_000, Max: 20_000}},
	}
}

// TestPositions_P03aListed verifies that P0.3a is a built line position of PLC, after
// P0.3 and before P1.1, and no longer listed as not built (FR-WLD-006, spec-policy.md
// 7.1.1).
//
// Requirement: FR-PLC-050
func TestPositions_P03aListed(t *testing.T) {
	t.Parallel()
	at := slices.IndexFunc(positionList[:], func(p position) bool { return p.id == "P0.3a" })
	if at < 0 {
		t.Fatalf("positionList has no P0.3a")
	}
	if p := positionList[at]; p.owner != "PLC" || p.where != line {
		t.Errorf("P0.3a = %+v, want owner PLC and a line position", p)
	}
	if before, after := positionList[at-1].id, positionList[at+1].id; before != "P0.3" || after != "P1.1" {
		t.Errorf("P0.3a stands between %s and %s, want P0.3 and P1.1", before, after)
	}
	if slices.Contains(notBuiltPositions[:], "P0.3a") {
		t.Errorf("notBuiltPositions lists P0.3a, which has a call")
	}
}

// TestNewPolicy_ResolvesActorsAndKeys verifies what the world derives from the
// configuration once: the actors in ascending ID (the ActorID), the jurisdiction, the seat
// and the treasury of each, the covered regions and the keys of each lever (D7).
//
// Requirement: FR-PLC-003, FR-PLC-020
func TestNewPolicy_ResolvesActorsAndKeys(t *testing.T) {
	t.Parallel()
	w := newMill(t, policyMillConfig(shireActor("hold", 10), crownActor("hold", 30)), WithSeed(1))
	s := w.policy
	if s == nil || len(s.actors) != 2 {
		t.Fatalf("policy state = %+v, want two actors", s)
	}
	crown, shire := &s.actors[0], &s.actors[1]
	t.Run("identity", func(t *testing.T) {
		if crown.id != "crown_ctl" || shire.id != "shire_ctl" {
			t.Errorf("actors = %q, %q, want crown_ctl and shire_ctl in ascending ID order", crown.id, shire.id)
		}
		if crown.every != 30 || shire.every != 10 || crown.kind != kindHold {
			t.Errorf("cadences = %d and %d, kind %v, want 30, 10 and hold", crown.every, shire.every, crown.kind)
		}
	})
	t.Run("place in the polity", func(t *testing.T) {
		if !crown.governed || crown.seat != 0 || shire.seat != 1 {
			t.Errorf("seats = %d and %d, governed %v, want 0, 1 and governed", crown.seat, shire.seat, crown.governed)
		}
		if !slices.Equal(crown.covered, []RegionID{0, 1}) || !slices.Equal(shire.covered, []RegionID{1}) {
			t.Errorf("covered = %v and %v, want [0 1] and [1]", crown.covered, shire.covered)
		}
	})
	t.Run("keys", func(t *testing.T) {
		gate, poll, road := crown.levers[0], crown.levers[1], crown.levers[2]
		if poll.flow != "poll" || road.from != 0 || road.to != 1 || road.mode != 0 {
			t.Errorf("crown levers = %+v and %+v, want the flow poll and the road 0 to 1 by mode 0", poll, road)
		}
		if !slices.Equal(gate.edges, []EdgeID{0}) || !slices.Equal(shire.levers[0].edges, []EdgeID{1}) {
			t.Errorf("gate edges = %v and back edges = %v, want [0] and [1]", gate.edges, shire.levers[0].edges)
		}
	})
}

// TestNewPolicy_ModuleOffIsNil verifies that a world with no policy module has no state,
// so no position runs and nothing is allocated (NFR-PLC-001).
//
// Requirement: NFR-PLC-001
func TestNewPolicy_ModuleOffIsNil(t *testing.T) {
	t.Parallel()
	c := policyMillConfig()
	c.Modules.Policy = nil
	if w := newMill(t, c, WithSeed(1)); w.policy != nil {
		t.Errorf("policy state of a world with the module off = %+v, want nil", w.policy)
	}
	if got := modulesOn(&c); slices.Contains(got, "policy") {
		t.Errorf("modulesOn with the module off = %v, want no policy", got)
	}
	c.Modules.Policy = &PolicyConfig{}
	if got := modulesOn(&c); !slices.Contains(got, "policy") {
		t.Errorf("modulesOn with the module on = %v, want policy", got)
	}
}

// TestDue_WorldActorsFollowTheRotation verifies the due test in the world: an actor with
// ActorID a and cadence every is due at the ticks where (t + bucket(a, every)) mod every
// is 0, so the actor of ActorID 0 is due at tick 0 and each multiple of its cadence, and
// the actor of ActorID 1 at its bucket's offset (spec-policy.md 7.2, worked example
// 14.1).
//
// Requirement: FR-PLC-051
func TestDue_WorldActorsFollowTheRotation(t *testing.T) {
	t.Parallel()
	w := newMill(t, policyMillConfig(crownActor("hold", 10), shireActor("hold", 10)), WithSeed(1))
	for id, a := range w.policy.actors {
		bucket := ordering.Bucket(int64(id), a.every)
		for tick := range int64(40) {
			want := (tick+bucket)%a.every == 0
			if got := w.policy.due(tick, ActorID(id)); got != want {
				t.Errorf("actor %d due at tick %d = %v, want %v (bucket %d)", id, tick, got, want, bucket)
			}
		}
	}
	if !w.policy.due(0, 0) || !w.policy.due(10, 0) || w.policy.due(1, 0) {
		t.Errorf("the actor of ActorID 0 is not due at ticks 0 and 10 only")
	}
}

// TestPolicy_NeutralHoldAndIdle verifies that a world with the policy module on and only
// hold actors runs exactly as the same world with the module off: the same snapshot,
// compared without the fields that name the modules and the configuration, and the same events after the
// construction, for 12 ticks (the first half of FR-PLC-010; the idle half is in
// policy_apply_internal_test.go).
//
// Requirement: FR-PLC-010
func TestPolicy_NeutralHoldAndIdle(t *testing.T) {
	t.Parallel()
	on := policyMillConfig(crownActor("hold", 3), shireActor("hold", 1))
	off := on
	off.Modules.Policy = nil
	a, b := &evLog{}, &evLog{}
	wa := newMill(t, on, WithSeed(7), WithObserver(a))
	wb := newMill(t, off, WithSeed(7), WithObserver(b))
	for tick := range 12 {
		for _, w := range []*World{wa, wb} {
			if err := w.Step(); err != nil {
				t.Fatalf("tick %d: Step() error = %v, want nil", tick, err)
			}
		}
	}
	sa, err := wa.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot() error = %v, want nil", err)
	}
	sb, err := wb.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot() error = %v, want nil", err)
	}
	if len(sb.Actors) != 0 || len(sa.Actors) != 2 {
		t.Errorf("Actors = %d rows with hold actors, %d with the module off, want 2 and 0", len(sa.Actors), len(sb.Actors))
	}
	sa.Modules, sb.Modules, sa.ConfigHash, sb.ConfigHash, sa.Actors = nil, nil, [32]byte{}, [32]byte{}, nil
	if !reflect.DeepEqual(sa, sb) {
		t.Errorf("snapshots differ between a world with only hold actors and one with the module off")
	}
	if len(a.events) != len(b.events) {
		t.Errorf("events = %d with hold actors, %d with the module off, want equal", len(a.events), len(b.events))
	}
}

// TestPolicy_InterfaceShape verifies that Policy has the one method Decide(PolicyView,
// *rand.Rand) Settings (architecture.md 4.2) and that the default implements it.
//
// Requirement: FR-PLC-001
func TestPolicy_InterfaceShape(t *testing.T) {
	t.Parallel()
	typ := reflect.TypeFor[Policy]()
	if typ.NumMethod() != 1 {
		t.Fatalf("Policy has %d methods, want 1", typ.NumMethod())
	}
	m := typ.Method(0)
	if m.Name != "Decide" || m.Type.NumIn() != 2 || m.Type.NumOut() != 1 ||
		m.Type.In(0) != reflect.TypeFor[PolicyView]() || m.Type.In(1).String() != "*rand.Rand" || m.Type.Out(0) != reflect.TypeFor[Settings]() {
		t.Errorf("Policy method = %v %v, want Decide(PolicyView, *rand.Rand) Settings", m.Name, m.Type)
	}
	if !reflect.TypeFor[HoldPolicy]().Implements(typ) {
		t.Errorf("HoldPolicy does not implement Policy")
	}
}

// TestPolicy_NoWorldInExports verifies that no exported type of the module holds a
// *World, so a policy never receives the world (spec-policy.md 4.3.2, FR-PLC-023).
//
// Requirement: FR-PLC-023, FR-PLC-035
func TestPolicy_NoWorldInExports(t *testing.T) {
	t.Parallel()
	for _, typ := range []reflect.Type{
		reflect.TypeFor[PolicyView](), reflect.TypeFor[LeverView](), reflect.TypeFor[Settings](), reflect.TypeFor[LeverSetting](),
		reflect.TypeFor[PolicyConfig](), reflect.TypeFor[PolicyActor](),
	} {
		for _, name := range reachedTypes(typ, map[reflect.Type]bool{}) {
			if strings.Contains(name, "World") {
				t.Errorf("%v reaches %s, want no world in a policy's data", typ, name)
			}
		}
	}
}

// reachedTypes lists the names of the types reachable from typ through fields, elements
// and pointers.
func reachedTypes(typ reflect.Type, seen map[reflect.Type]bool) []string {
	if seen[typ] {
		return nil
	}
	seen[typ] = true
	out := []string{typ.String()}
	switch typ.Kind() {
	case reflect.Pointer, reflect.Slice, reflect.Array:
		out = append(out, reachedTypes(typ.Elem(), seen)...)
	case reflect.Struct:
		for i := range typ.NumField() {
			out = append(out, reachedTypes(typ.Field(i).Type, seen)...)
		}
	}
	return out
}

// TestPolicy_NoIndicatorImport verifies that the files of the module import neither
// indicator/ nor experiment/: the library never calls an indicator (spec-policy.md 4.3.3,
// FR-PLC-091).
//
// Requirement: FR-PLC-091, FR-PLC-035
func TestPolicy_NoIndicatorImport(t *testing.T) {
	t.Parallel()
	files := []string{"policy.go", "policy_validate.go", "policy_reach.go", "policy_warn.go", "policy_view.go", "policy_apply.go", "policy_builtin.go"}
	fset := token.NewFileSet()
	for _, name := range files {
		f, err := parser.ParseFile(fset, name, nil, parser.ImportsOnly)
		if err != nil {
			continue // a file of a later change
		}
		for _, imp := range f.Imports {
			if p := imp.Path.Value; strings.Contains(p, "/indicator") || strings.Contains(p, "/experiment") {
				t.Errorf("%s imports %s, want neither indicator nor experiment", name, p)
			}
		}
	}
}

// TestCost_PolicyNoActorDueZeroAllocs verifies NFR-PLC-001: a step of a world with the
// policy module on in which no actor is due allocates nothing for the module, with the
// default logger and observer.
//
// Requirement: NFR-PLC-001
func TestCost_PolicyNoActorDueZeroAllocs(t *testing.T) {
	// No t.Parallel: AllocsPerRun changes GOMAXPROCS while it measures.
	w := newMill(t, policyMillConfig(crownActor("schedule", 1_000), shireActor("rule", 1_000)), WithSeed(1))
	if err := w.Step(); err != nil {
		t.Fatalf("Step() error = %v, want nil", err)
	}
	for id := range w.policy.actors {
		if w.policy.due(w.Tick(), ActorID(id)) {
			t.Fatalf("actor %d is due at tick %d, want none due (the cadence is 1 000)", id, w.Tick())
		}
	}
	if got := testing.AllocsPerRun(50, w.p03aPolicy); got != 0 {
		t.Errorf("allocs of P0.3a with no actor due = %v, want 0", got)
	}
}
