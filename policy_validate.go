package economysim

import (
	"slices"
)

// This file is the validation of the policy module (spec-policy.md 11.1; the
// continuation file of policy.go, architecture.md 2.6): the rules that Validate checks,
// in the order of 11.1. The warnings of 11.2 are in policy_warn.go, the reach and the
// keys in policy_reach.go. Nothing at run time returns an error from the module. An
// ErrPolicyShape carries the name of its rule as the Bound of the ValidationError, as
// the table of 11.1 names it, and the offending value as the Value.

// The reserved policy names, registered by the world and refused to a consumer
// (spec-policy.md 4.4.1). The built-ins are not exported values (N-481).
const (
	policyHold     = "hold"
	policySchedule = "schedule"
	policyRule     = "rule"
	policyDirected = "directed"
)

// policyBase is the path of the module in a configuration.
var policyBase = []string{"modules", "policy"}

// policyActorPath is the path of a key of the actor at index i of the list as given,
// for example modules.policy.actors[1].every.
func policyActorPath(i int, segs ...string) []string {
	return path(policyBase, append([]string{"actors[" + itoa(i) + "]"}, segs...)...)
}

// policyListPath is the path of a key of an element of one of an actor's lists, for
// example modules.policy.actors[1].levers[0].min.
func policyListPath(i int, list string, j int, segs ...string) []string {
	return path(policyBase, append([]string{"actors[" + itoa(i) + "]", list + "[" + itoa(j) + "]"}, segs...)...)
}

// policyCheck is the state the phases of validatePolicy share: the module, and the
// canonical configuration, the polity and the graph it was built from (zero until the
// dependency phase has found them sound).
type policyCheck struct {
	v       *validator
	m       *PolicyConfig
	cc      Config
	pol     Polity
	g       Graph
	outside map[string]bool // the outside nodes of the external module; a lookup only
}

// validatePolicy checks the policy module (spec-policy.md 11.1, FR-PLC-002 to 012). It
// reports nothing when the module is off. The rules run in the order of 11.1 and a
// phase that finds an error ends the run, so the first error is the same for the same
// mistake. The two rules that need the keys of the levers (the count of an Against and a
// key named twice by one actor) run with the keys, after the reach and the nesting, and
// before the keys shared by two actors.
func (v *validator) validatePolicy() {
	m := v.c.Modules.Policy
	if m == nil {
		return
	}
	k := &policyCheck{v: v, m: m}
	for _, step := range []func() bool{
		k.dependencies, k.counts, k.ids, k.jurisdictions, k.oneActor, k.bounds, k.data,
		k.keyFields, k.entities, k.transitPairs, k.reach, k.nested, k.schedules, k.reactions,
		k.governed, k.keys,
	} {
		if !step() {
			return
		}
	}
}

// dependencies is ErrModuleDependency: the module needs the polity module with a
// jurisdiction, a flow lever needs the money supply or the fiscal module and a transit
// lever the transport module. It also builds the polity, and the graph the Against and
// edge rules read; a polity or a graph that is itself invalid ends the run here, its
// own errors having been reported by their validators.
func (k *policyCheck) dependencies() bool {
	v, n := k.v, len(k.v.rep.errs)
	p := v.c.Modules.Polity
	if p == nil || len(p.Jurisdictions) == 0 {
		v.rep.add(ErrModuleDependency, policyBase, "polity", "")
		return false
	}
	k.needs()
	if len(v.rep.errs) != n {
		return false
	}
	return k.build()
}

// needs reports the lever kinds whose module is off, once for the lever.
func (k *policyCheck) needs() {
	c := &k.v.c
	for i, a := range k.m.Actors {
		for j, l := range a.Levers {
			switch {
			case l.Kind == LeverFlow && c.Modules.MoneySupply == nil && c.Modules.Fiscal == nil:
				k.v.rep.add(ErrModuleDependency, policyListPath(i, "levers", j, "kind"), "money_supply or fiscal", "")
			case l.Kind == LeverTransit && c.Modules.Transport == nil:
				k.v.rep.add(ErrModuleDependency, policyListPath(i, "levers", j, "kind"), "transport", "")
			}
		}
	}
}

// build derives the canonical copy, the polity and the graph, and reports false when
// the polity or the graph is not sound.
func (k *policyCheck) build() bool {
	c := &k.v.c
	pol, ok := soundPolity(c, k.v.lk)
	if !ok || graphRulesError(*c) != nil {
		return false
	}
	k.cc = c.canonical()
	g, err := newGraph(&k.cc, nil)
	if err != nil {
		return false
	}
	k.pol, k.g, k.outside = pol, g, outsideNodeSet(&k.cc)
	return true
}

// counts is ErrTooMany: more than MaxActors actors, MaxLevers levers of an actor or
// MaxScheduleSteps steps of an actor.
func (k *policyCheck) counts() bool {
	r, n := &k.v.rep, len(k.v.rep.errs)
	if len(k.m.Actors) > MaxActors {
		r.add(ErrTooMany, path(policyBase, "actors"), itoa(len(k.m.Actors)), itoa(MaxActors))
		return false
	}
	for i, a := range k.m.Actors {
		if len(a.Levers) > MaxLevers {
			r.add(ErrTooMany, policyActorPath(i, "levers"), itoa(len(a.Levers)), itoa(MaxLevers))
		}
		if len(a.Schedule) > MaxScheduleSteps {
			r.add(ErrTooMany, policyActorPath(i, "schedule"), itoa(len(a.Schedule)), itoa(MaxScheduleSteps))
		}
	}
	return len(r.errs) == n
}

// ids is ErrInvalidID and ErrDuplicateID: the syntax of the actor and lever IDs, an
// actor ID repeated, a lever ID repeated within an actor, a step (Lever, At) repeated
// and a reaction Lever repeated.
func (k *policyCheck) ids() bool {
	r, n := &k.v.rep, len(k.v.rep.errs)
	actors := make(map[string]bool, len(k.m.Actors)) // membership tests only
	for i, a := range k.m.Actors {
		k.id(actors, a.ID, policyActorPath(i, "id"))
		levers := make(map[string]bool, len(a.Levers))
		for j, l := range a.Levers {
			k.id(levers, l.ID, policyListPath(i, "levers", j, "id"))
		}
		steps := make(map[ScheduleStep]bool, len(a.Schedule))
		for j, s := range a.Schedule {
			key := ScheduleStep{Lever: s.Lever, At: s.At}
			if steps[key] {
				r.add(ErrDuplicateID, policyListPath(i, "schedule", j, "at"), i64(s.At), "")
			}
			steps[key] = true
		}
		rules := make(map[string]bool, len(a.Rules))
		for j, x := range a.Rules {
			if rules[x.Lever] {
				r.add(ErrDuplicateID, policyListPath(i, "rules", j, "lever"), x.Lever, "")
			}
			rules[x.Lever] = true
		}
	}
	return len(r.errs) == n
}

// id reports an ID that is not valid or that is already in seen, and records it.
func (k *policyCheck) id(seen map[string]bool, id string, p []string) {
	if !validID(id) {
		k.v.rep.add(ErrInvalidID, p, id, "")
	}
	if seen[id] {
		k.v.rep.add(ErrDuplicateID, p, id, "")
	}
	seen[id] = true
}

// jurisdictions is ErrUnknownJurisdiction: the jurisdiction of an actor and the
// Against of a lever are defined.
func (k *policyCheck) jurisdictions() bool {
	r, n := &k.v.rep, len(k.v.rep.errs)
	for i, a := range k.m.Actors {
		if _, ok := k.pol.Lookup(a.Jurisdiction); !ok {
			r.add(ErrUnknownJurisdiction, policyActorPath(i, "jurisdiction"), a.Jurisdiction, "")
		}
		for j, l := range a.Levers {
			if _, ok := k.pol.Lookup(l.Against); l.Against != "" && !ok {
				r.add(ErrUnknownJurisdiction, policyListPath(i, "levers", j, "against"), l.Against, "")
			}
		}
	}
	return len(r.errs) == n
}

// oneActor is ErrPolicyShape, rule one_actor_per_jurisdiction.
func (k *policyCheck) oneActor() bool {
	n := len(k.v.rep.errs)
	seen := make(map[string]bool, len(k.m.Actors)) // a membership test only
	for i, a := range k.m.Actors {
		if seen[a.Jurisdiction] {
			k.shape("one_actor_per_jurisdiction", policyActorPath(i, "jurisdiction"), a.Jurisdiction)
		}
		seen[a.Jurisdiction] = true
	}
	return len(k.v.rep.errs) == n
}

// shape reports ErrPolicyShape at p for the rule, with the offending value.
func (k *policyCheck) shape(rule string, p []string, value string) {
	k.v.rep.add(ErrPolicyShape, p, value, rule)
}

// data is ErrPolicyShape, rule policy_data: a schedule or a cycle only with the
// schedule policy and rules only with the rule policy.
func (k *policyCheck) data() bool {
	n := len(k.v.rep.errs)
	for i, a := range k.m.Actors {
		if (len(a.Schedule) > 0 || a.Cycle != 0) && a.Policy != policySchedule {
			k.shape("policy_data", policyActorPath(i, "schedule"), a.Policy)
		}
		if len(a.Rules) > 0 && a.Policy != policyRule {
			k.shape("policy_data", policyActorPath(i, "rules"), a.Policy)
		}
	}
	return len(k.v.rep.errs) == n
}

// keyFields is ErrPolicyShape, rule lever_key: the key fields of a kind exactly, and
// MayClose only on an edge.
func (k *policyCheck) keyFields() bool {
	n := len(k.v.rep.errs)
	for i, a := range k.m.Actors {
		for j, l := range a.Levers {
			if !leverKeyFieldsOK(l) || (l.MayClose && l.Kind != LeverEdge) {
				k.shape("lever_key", policyListPath(i, "levers", j, "kind"), string(l.Kind))
			}
		}
	}
	return len(k.v.rep.errs) == n
}

// leverKeyFieldsOK says the key fields of the lever are exactly those of its kind: a
// flow has Flow and no other; an edge has From and To, or Against, and no Flow or Mode;
// a transit has From, To and Mode and no Flow or Against.
func leverKeyFieldsOK(l Lever) bool {
	const (
		flow, from, to, mode, against = 1, 2, 4, 8, 16
	)
	present := func(s string, bit uint) uint {
		if s == "" {
			return 0
		}
		return bit
	}
	set := present(l.Flow, flow) | present(l.From, from) | present(l.To, to) | present(l.Mode, mode) | present(l.Against, against)
	switch l.Kind {
	case LeverFlow:
		return set == flow
	case LeverEdge:
		return set == from|to || set == against
	case LeverTransit:
		return set == from|to|mode
	}
	return true // an unknown kind is ErrOutOfBounds, reported by bounds
}

// entities is ErrUnknownFlow, ErrUnknownRegion, ErrUnknownEdge and ErrUnknownMode: a key
// that names nothing configured.
func (k *policyCheck) entities() bool {
	n := len(k.v.rep.errs)
	for i, a := range k.m.Actors {
		for j, l := range a.Levers {
			switch l.Kind {
			case LeverFlow:
				k.flowKnown(i, j, l)
			case LeverEdge:
				k.edgeKnown(i, j, l)
			case LeverTransit:
				k.transitKnown(i, j, l)
			}
		}
	}
	return len(k.v.rep.errs) == n
}

// flowKnown is ErrUnknownFlow: the ID is a money flow or a fiscal flow.
func (k *policyCheck) flowKnown(i, j int, l Lever) {
	if _, _, ok := k.flowLookup(l.Flow); !ok {
		k.v.rep.add(ErrUnknownFlow, policyListPath(i, "levers", j, "flow"), l.Flow, "")
	}
}

// edgeKnown is ErrUnknownRegion and ErrUnknownEdge for an edge named by its pair.
func (k *policyCheck) edgeKnown(i, j int, l Lever) {
	if l.Against != "" {
		return
	}
	known := k.regionKnown(i, j, "from", l.From)
	known = k.regionKnown(i, j, "to", l.To) && known
	if known && !slices.ContainsFunc(k.v.c.Edges, func(e Edge) bool { return e.From == l.From && e.To == l.To }) {
		k.v.rep.add(ErrUnknownEdge, policyListPath(i, "levers", j, "from"), l.From+">"+l.To, "")
	}
}

// transitKnown is ErrUnknownRegion and ErrUnknownMode for a transit key.
func (k *policyCheck) transitKnown(i, j int, l Lever) {
	k.regionKnown(i, j, "from", l.From)
	k.regionKnown(i, j, "to", l.To)
	if !slices.ContainsFunc(k.v.c.Modules.Transport.Modes, func(m Mode) bool { return m.ID == l.Mode }) {
		k.v.rep.add(ErrUnknownMode, policyListPath(i, "levers", j, "mode"), l.Mode, "")
	}
}

// regionKnown reports ErrUnknownRegion for a region ID that is not declared.
func (k *policyCheck) regionKnown(i, j int, key, id string) bool {
	if _, ok := k.cc.regionID(id); !ok {
		k.v.rep.add(ErrUnknownRegion, policyListPath(i, "levers", j, key), id, "")
		return false
	}
	return true
}

// transitPairs is ErrPolicyShape, rule transit_pair: a transit lever with From equal to
// To, or with either an outside node (an edge lever may name a gate edge).
func (k *policyCheck) transitPairs() bool {
	n := len(k.v.rep.errs)
	for i, a := range k.m.Actors {
		for j, l := range a.Levers {
			if l.Kind != LeverTransit {
				continue
			}
			if l.From == l.To || k.outside[l.From] || k.outside[l.To] {
				k.shape("transit_pair", policyListPath(i, "levers", j, "to"), l.From+">"+l.To)
			}
		}
	}
	return len(k.v.rep.errs) == n
}
