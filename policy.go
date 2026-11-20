package economysim

import (
	"log/slog"
	"math/rand/v2" // nosemgrep: go.lang.security.audit.crypto.math_random.math-random-used -- only the *rand.Rand type of the injected, seeded simulation source, never a secret (architecture.md 7.3, 7.7)
	"slices"

	"github.com/Whealinghold/economy-sim/internal/logkeys"
	"github.com/Whealinghold/economy-sim/internal/policy"
)

// This file is the module file of the policy module (spec-policy.md; plan 3-plc): the
// types of the interface, the state, its construction and the position P0.3a. The
// view is in policy_view.go, the checks and the application in policy_apply.go, the
// built-in policies in policy_builtin.go, the validation in policy_validate.go,
// policy_reach.go and policy_warn.go.

// ActorID is the index of a policy actor in ascending order of its ID string. It is not
// stable across configurations; use the string ID to persist or exchange one
// (spec-policy.md 3.2.2).
type ActorID int

// LeverID is the index of a lever in the ascending order of the actor's lever IDs.
type LeverID int

// Policy decides the settings of one policy actor, at the actor's decision tick. It
// returns settings of the actor's levers; it does not bid, offer, produce or move
// money. Decide is called between the previous step and the first position that reads a
// slot (P0.3a), on the world's goroutine, once per due actor, in ascending ActorID,
// with rng reseeded at the key (sitePolicy, tick, ActorID).
//
// Randomness: an implementation MUST take it from rng and MUST NOT create a source, use
// a global one or keep rng after Decide returns (architecture.md 7.7). State: an
// implementation that keeps state between calls MUST also implement Stateful, MUST NOT
// share an instance between worlds and MUST NOT read the clock, the environment, a
// map's order or an address (architecture.md 7.1, 7.4). The view is a deep copy the
// implementation may keep or change; the returned Settings is copied by the world.
type Policy interface {
	Decide(v PolicyView, rng *rand.Rand) Settings
}

// HoldPolicy is the default policy: it returns no settings and keeps no state, so every
// lever keeps the value its slot holds (spec-policy.md 8.1, ADR 0001 decision 4).
type HoldPolicy struct{}

// Decide returns no settings.
func (HoldPolicy) Decide(PolicyView, *rand.Rand) Settings { return Settings{} }

var _ Policy = HoldPolicy{}

// PolicyView is what Decide receives: plain data, a deep copy built for this call
// (spec-policy.md 4.2).
type PolicyView struct {
	// Tick is the tick the step runs, and Moment is Calendar().At(Tick).
	Tick   int64
	Moment Moment
	// Actor and Jurisdiction say who is deciding and where.
	Actor        ActorID
	Jurisdiction JurisdictionID
	// Seat is the seat region of the jurisdiction, NoRegion for a grouping.
	Seat RegionID
	// HasTreasury says the jurisdiction is governed; Treasury is the balance of its
	// treasury at the boundary, 0 without one.
	HasTreasury bool
	Treasury    Money
	// Levers are the actor's levers by LeverID.
	Levers []LeverView
	// Regions are the states of the regions the jurisdiction covers that are not outside
	// nodes, ascending RegionID; empty unless the actor has ReadsRegions.
	Regions []RegionState
}

// LeverView is one lever as the actor holds it now (spec-policy.md 4.2).
type LeverView struct {
	Lever LeverID
	Kind  LeverKind
	// BP is the current value (of a group, that of its first key) and Closed whether
	// the key is closed now (of a group, whether the first key is).
	BP     int64
	Closed bool
	// Default is the value of an unset key.
	Default int64
	// Min, Max and MaxStep are the lever's limits; MayClose says the actor may close it.
	Min, Max, MaxStep int64
	MayClose          bool
}

// Settings is what Decide returns: values for some of the actor's levers. A lever with
// no entry keeps its value (spec-policy.md 4.2).
type Settings struct {
	Levers []LeverSetting
}

// LeverSetting asks that one lever take a value: the whole value, BP and Closed, as an
// integration setter takes a whole value (spec-policy.md 4.2). To change BP of a closed
// edge and keep it closed the entry carries Closed true, which is not a closing.
type LeverSetting struct {
	Lever LeverID
	BP    int64
	// Closed is the closed flag of an edge lever; true needs MayClose, and is refused on
	// another kind (no_close).
	Closed bool
}

// policyKind says which built-in policy an actor names, or that a consumer's does.
type policyKind int

const (
	kindConsumer policyKind = iota
	kindHold
	kindSchedule
	kindRule
	kindDirected
)

// policyKindOf is the kind a policy name selects: a reserved name, else a consumer's.
func policyKindOf(name string) policyKind {
	switch name {
	case policyHold:
		return kindHold
	case policySchedule:
		return kindSchedule
	case policyRule:
		return kindRule
	case policyDirected:
		return kindDirected
	}
	return kindConsumer
}

// policyLever is one lever resolved to its keys at construction (plan 3-plc D7): the
// tree is fixed for a run (spec-polity.md position 9), so nothing here is rebuilt or
// saved.
type policyLever struct {
	id                string
	kind              LeverKind
	min, max, maxStep int64
	mayClose          bool
	flow              string   // kind flow: the flow ID
	edges             []EdgeID // kind edge: the keys in ascending EdgeID, never empty unless the group is
	from, to          RegionID // kind transit
	mode              ModeID   // kind transit
}

// policyActor is one actor resolved at construction: its place in the polity, its
// cadence, its levers with their keys, and the policy that decides for it.
type policyActor struct {
	id       string
	j        JurisdictionID
	seat     RegionID
	governed bool
	every    int64
	reads    bool
	kind     policyKind
	builtin  Policy      // the schedule or the rule built from the row; nil for the others
	consumer Policy      // the registered policy of a consumer's actor; nil for the others
	rule     *rulePolicy // the same value when the policy is a rule: the world reads what it held
	levers   []policyLever
	covered  []RegionID // the covered regions that are not outside nodes, ascending
}

// policyState is the state of the module, nil when it is off: the actors by ActorID, the
// mailbox of each directed actor and the scratch the decisions reuse. The mailbox, by
// ActorID and sorted by lever, is the only state that is saved (tag 32, policy_state.go).
type policyState struct {
	actors  []policyActor
	mailbox [][]LeverSetting
	scratch policyScratch
}

// newPolicy builds the state of a canonical configuration the validator accepted, nil
// when the module is off. Each lever is resolved to its keys once (plan 3-plc D7);
// reg holds the consumer's policies, found by name.
func newPolicy(c *Config, pol *polity, g Graph, reg registry[Policy], outside []bool) *policyState {
	m := c.Modules.Policy
	if m == nil {
		return nil
	}
	s := &policyState{actors: make([]policyActor, len(m.Actors)), mailbox: make([][]LeverSetting, len(m.Actors))}
	for i := range m.Actors {
		s.actors[i] = newPolicyActor(c, &m.Actors[i], pol.v, g, reg, outside)
	}
	return s
}

// newPolicyActor resolves the row a to its place in the polity and its levers.
func newPolicyActor(c *Config, a *PolicyActor, pol Polity, g Graph, reg registry[Policy], outside []bool) policyActor {
	j, _ := pol.Lookup(a.Jurisdiction)
	seat, governed := pol.Seat(j)
	out := policyActor{id: a.ID, j: j, seat: seat, governed: governed, every: a.Every, reads: a.ReadsRegions, kind: policyKindOf(a.Policy)}
	if !governed {
		out.seat = NoRegion
	}
	switch out.kind {
	case kindSchedule:
		out.builtin = newSchedulePolicy(a)
	case kindRule:
		out.rule = newRulePolicy(c, a)
		out.builtin = out.rule
	case kindConsumer:
		out.consumer, _ = reg.find(a.Policy)
	}
	for _, r := range pol.Regions(j) {
		if int(r) >= len(outside) || !outside[r] {
			out.covered = append(out.covered, r)
		}
	}
	out.levers = make([]policyLever, len(a.Levers))
	for i := range a.Levers {
		out.levers[i] = newPolicyLever(c, &a.Levers[i], j, pol, g)
	}
	return out
}

// newPolicyLever resolves one lever to its keys: a flow ID, the edges in ascending
// EdgeID, or the transit triple.
func newPolicyLever(c *Config, l *Lever, j JurisdictionID, pol Polity, g Graph) policyLever {
	out := policyLever{id: l.ID, kind: l.Kind, min: l.Min, max: l.Max, maxStep: l.MaxStep, mayClose: l.MayClose, flow: l.Flow}
	switch l.Kind {
	case LeverEdge:
		out.edges = policyEdgeIDs(pol, g, j, *l)
	case LeverTransit:
		out.from, _ = g.Lookup(l.From)
		out.to, _ = g.Lookup(l.To)
		modes := c.Modules.Transport.Modes
		out.mode = ModeID(slices.IndexFunc(modes, func(m Mode) bool { return m.ID == l.Mode }))
	}
	return out
}

// due reports whether the actor is due at tick: (tick + bucket(ActorID, Every)) mod Every
// is 0 (spec-policy.md 7.2.1). It is a pure function of the tick, the cadence and the
// index, so a restored world is due at the same ticks.
func (s *policyState) due(tick int64, actor ActorID) bool {
	return policy.Due(tick, int64(actor), s.actors[actor].every)
}

// p03aPolicy is P0.3a: each due actor in ascending ActorID reads, decides and writes
// (spec-policy.md 7.1). An actor is due when (t + bucket(ActorID, Every)) mod Every is 0
// (7.2); a hold actor does nothing and builds nothing.
func (w *World) p03aPolicy() {
	p := w.policy
	tick := w.Tick()
	var pass policyPass
	for i := range p.actors {
		a := &p.actors[i]
		if a.kind == kindHold || !p.due(tick, ActorID(i)) {
			continue
		}
		w.policyDecide(ActorID(i), a, &pass)
	}
}

// checkPolicyNames is the name check of the policy module, run by NewWorld after
// Validate succeeds (spec-policy.md 4.5.1, spec-world.md 3.2.2): an actor's policy that
// is neither a reserved name nor registered is a *ValidationError of ErrUnknownPolicy at
// modules.policy.actors[i].policy, the first in ID order.
func checkPolicyNames(c *Config, o *options) error {
	m := c.Modules.Policy
	if m == nil {
		return nil
	}
	for i := range m.Actors {
		name := m.Actors[i].Policy
		if _, ok := o.policies.find(name); !ok && policyKindOf(name) == kindConsumer {
			return &ValidationError{Path: policyActorPath(i, "policy"), Rule: ErrUnknownPolicy, Value: name}
		}
	}
	return nil
}

// policySummary appends the counts of the module to the attributes of the line "world
// created" (spec-policy.md 12.2.2): the actors and the levers.
func (w *World) policySummary(attrs []slog.Attr) []slog.Attr {
	if w.policy == nil {
		return attrs
	}
	levers := 0
	for i := range w.policy.actors {
		levers += len(w.policy.actors[i].levers)
	}
	return append(attrs, slog.Int(logkeys.Actors, len(w.policy.actors)), slog.Int(logkeys.Levers, levers))
}
