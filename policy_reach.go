package economysim

import (
	"cmp"
	"slices"
)

// This file is the second half of the validator of the policy module (spec-policy.md
// 11.1; the continuation file of policy.go, architecture.md 2.6): the bounds, the
// reach of a lever (3.5), the nesting of an Against, the schedule and reaction rules and
// the keys a lever holds. The phases of policy_validate.go run them in the order of 11.1.

// policyDefault is the value of an unset key of every kind of lever: a flow's scale, an
// edge's factor and a transit factor are all 10 000 at their default (spec-policy.md
// 6.3.1).
const policyDefault = wholeBP

// slotBound is the lowest and highest value of the slot a kind of lever writes
// (spec-policy.md 6.3.1), and false for a kind that is not offered.
func slotBound(kind LeverKind) (lo, hi int64, ok bool) {
	switch kind {
	case LeverFlow:
		return 0, MaxFlowScale, true
	case LeverEdge:
		return 1, MaxEdgeFactorBP, true
	case LeverTransit:
		return 0, MaxTransitFactorBP, true
	}
	return 0, 0, false
}

// bounds is ErrOutOfBounds: a kind that is not offered (a reserved kind of 6.3.2 is
// refused here, path .kind), and a number outside its bound.
func (k *policyCheck) bounds() bool {
	n := len(k.v.rep.errs)
	for i, a := range k.m.Actors {
		k.v.inRange(policyActorPath(i, "every"), a.Every, 1, MaxPolicyEvery)
		k.v.inRange(policyActorPath(i, "cycle"), a.Cycle, 0, MaxPolicyCycle)
		for j, l := range a.Levers {
			k.leverBounds(i, j, l)
		}
		for j, s := range a.Schedule {
			k.stepBounds(i, j, a, s)
		}
		for j, r := range a.Rules {
			k.ruleBounds(i, j, r)
		}
	}
	return len(k.v.rep.errs) == n
}

// leverBounds checks the kind, Min, Max, the default and MaxStep of one lever.
func (k *policyCheck) leverBounds(i, j int, l Lever) {
	lo, hi, ok := slotBound(l.Kind)
	if !ok {
		k.v.rep.add(ErrOutOfBounds, policyListPath(i, "levers", j, "kind"), string(l.Kind), "flow, edge or transit")
		return
	}
	okMin := k.v.inRange(policyListPath(i, "levers", j, "min"), l.Min, lo, hi)
	okMax := k.v.inRange(policyListPath(i, "levers", j, "max"), l.Max, lo, hi)
	if okMin && okMax {
		k.leverOrder(i, j, l)
	}
	k.v.inRange(policyListPath(i, "levers", j, "max_step"), l.MaxStep, 0, hi)
}

// leverOrder checks Min <= Max and Min <= default <= Max, for a Min and a Max that are
// inside the slot's bound.
func (k *policyCheck) leverOrder(i, j int, l Lever) {
	pmin, pmax := policyListPath(i, "levers", j, "min"), policyListPath(i, "levers", j, "max")
	switch {
	case l.Min > l.Max:
		k.v.rep.add(ErrOutOfBounds, pmin, i64(l.Min), i64(l.Max))
	case l.Min > policyDefault:
		k.v.rep.add(ErrOutOfBounds, pmin, i64(l.Min), i64(policyDefault))
	case l.Max < policyDefault:
		k.v.rep.add(ErrOutOfBounds, pmax, i64(l.Max), i64(policyDefault))
	}
}

// stepBounds checks the tick of a step against MaxScheduleTick and the cycle, and its
// value against the slot of its lever when the lever is named.
func (k *policyCheck) stepBounds(i, j int, a PolicyActor, s ScheduleStep) {
	if k.v.inRange(policyListPath(i, "schedule", j, "at"), s.At, 0, MaxScheduleTick) && a.Cycle > 0 && s.At >= a.Cycle {
		k.v.rep.add(ErrOutOfBounds, policyListPath(i, "schedule", j, "at"), i64(s.At), i64(a.Cycle-1))
	}
	if l, ok := actorLever(a, s.Lever); ok {
		if lo, hi, known := slotBound(l.Kind); known {
			k.v.inRange(policyListPath(i, "schedule", j, "bp"), s.BP, lo, hi)
		}
	}
}

// ruleBounds checks the numbers of a reaction.
func (k *policyCheck) ruleBounds(i, j int, r Reaction) {
	for _, f := range []struct {
		key    string
		val    int64
		lo, hi int64
	}{
		{"max_age", r.MaxAge, 0, MaxPolicyEvery},
		{"target", r.Target, 0, MaxMoney},
		{"deadband", r.Deadband, 0, MaxMoney},
		{"span", r.Span, 1, MaxMoney},
		{"move", r.Move, 0, MaxMoveBP},
	} {
		k.v.inRange(policyListPath(i, "rules", j, f.key), f.val, f.lo, f.hi)
	}
}

// actorLever returns the lever of an actor with an ID, in the order as given.
func actorLever(a PolicyActor, id string) (Lever, bool) {
	i := slices.IndexFunc(a.Levers, func(l Lever) bool { return l.ID == id })
	if i < 0 {
		return Lever{}, false
	}
	return a.Levers[i], true
}

// flowLookup returns what owns a flow ID: the authority of a fiscal flow, or the money
// flow. The two share one namespace (spec-fiscal.md 3.1.1).
func (k *policyCheck) flowLookup(id string) (authority string, money *MoneyFlow, ok bool) {
	if f := k.v.c.Modules.Fiscal; f != nil {
		for _, ff := range f.flows() {
			if ff.id == id {
				return ff.authority, nil, true
			}
		}
	}
	if ms := k.v.c.Modules.MoneySupply; ms != nil {
		for i := range ms.Flows {
			if ms.Flows[i].ID == id {
				return "", &ms.Flows[i], true
			}
		}
	}
	return "", nil, false
}

// descends says jurisdiction a is j or a descendant of it.
func (k *policyCheck) descends(a, j JurisdictionID) bool {
	for steps := 0; steps <= k.pol.Len(); steps++ {
		if a == j {
			return true
		}
		p, ok := k.pol.Parent(a)
		if !ok {
			return false
		}
		a = p
	}
	return false
}

// reach is ErrLeverNotInReach: the key of a lever is inside what the jurisdiction
// covers (spec-policy.md 3.5.1).
func (k *policyCheck) reach() bool {
	n := len(k.v.rep.errs)
	for i, a := range k.m.Actors {
		j, _ := k.pol.Lookup(a.Jurisdiction)
		for li, l := range a.Levers {
			if !k.inReach(j, l) {
				k.v.rep.add(ErrLeverNotInReach, policyListPath(i, "levers", li, "kind"), leverKeyText(l), a.Jurisdiction)
			}
		}
	}
	return len(k.v.rep.errs) == n
}

// inReach is the table of 3.5.1 for one lever of an actor on jurisdiction j.
func (k *policyCheck) inReach(j JurisdictionID, l Lever) bool {
	switch l.Kind {
	case LeverFlow:
		return k.flowInReach(j, l.Flow)
	case LeverEdge:
		return l.Against != "" || k.coversEnd(j, l.From, l.To)
	case LeverTransit:
		return k.coversEnd(j, l.From, l.To)
	}
	return true
}

// coversEnd says j covers the region from or the region to.
func (k *policyCheck) coversEnd(j JurisdictionID, from, to string) bool {
	a, _ := k.cc.regionID(from)
	b, _ := k.cc.regionID(to)
	return k.pol.Covers(j, a) || k.pol.Covers(j, b)
}

// flowInReach is the flow rows of 3.5.1: a fiscal flow whose authority is j or a
// descendant; a money flow whose target regions are all covered by j (every region if it
// names none), or, for a treasury target, whose treasury is that of j or of a descendant
// (every governed one if it names none).
func (k *policyCheck) flowInReach(j JurisdictionID, id string) bool {
	auth, mf, _ := k.flowLookup(id)
	if mf == nil {
		a, ok := k.pol.Lookup(auth)
		return ok && k.descends(a, j)
	}
	if mf.Target.Kind == TargetTreasury {
		return k.treasuriesInReach(j, mf.Target.Jurisdiction)
	}
	if mf.Target.Region != "" {
		r, _ := k.cc.regionID(mf.Target.Region)
		return k.pol.Covers(j, r)
	}
	for r := range k.cc.Regions {
		if !k.outside[k.cc.Regions[r].ID] && !k.pol.Covers(j, RegionID(r)) {
			return false
		}
	}
	return true
}

// treasuriesInReach says the treasury of the named jurisdiction (every governed one when
// the name is empty) is that of j or of a descendant.
func (k *policyCheck) treasuriesInReach(j JurisdictionID, name string) bool {
	if name != "" {
		t, ok := k.pol.Lookup(name)
		return ok && k.descends(t, j)
	}
	for _, g := range k.pol.Governed() {
		if !k.descends(g, j) {
			return false
		}
	}
	return true
}

// leverKeyText is the key of a lever as text, for an error value.
func leverKeyText(l Lever) string {
	switch l.Kind {
	case LeverFlow:
		return l.Flow
	case LeverEdge:
		if l.Against != "" {
			return "against " + l.Against
		}
		return l.From + ">" + l.To
	}
	return l.From + ">" + l.To + ":" + l.Mode
}

// nested is ErrPolicyShape, rule against_nested: an Against that is the actor's own
// jurisdiction, an ancestor or a descendant, since their regions are nested and no edge
// crosses.
func (k *policyCheck) nested() bool {
	n := len(k.v.rep.errs)
	for i, a := range k.m.Actors {
		j, _ := k.pol.Lookup(a.Jurisdiction)
		for li, l := range a.Levers {
			t, ok := k.pol.Lookup(l.Against)
			if l.Against != "" && ok && (k.descends(t, j) || k.descends(j, t)) {
				k.shape("against_nested", policyListPath(i, "levers", li, "against"), l.Against)
			}
		}
	}
	return len(k.v.rep.errs) == n
}

// schedules is ErrPolicyShape, rules schedule_lever, schedule_value and schedule_step
// (spec-policy.md 8.2.3).
func (k *policyCheck) schedules() bool {
	n := len(k.v.rep.errs)
	for i, a := range k.m.Actors {
		for j, s := range a.Schedule {
			l, ok := actorLever(a, s.Lever)
			switch {
			case !ok:
				k.shape("schedule_lever", policyListPath(i, "schedule", j, "lever"), s.Lever)
			case s.BP < l.Min || s.BP > l.Max || (s.Closed && !l.MayClose):
				k.shape("schedule_value", policyListPath(i, "schedule", j, "bp"), i64(s.BP))
			}
		}
		if len(k.v.rep.errs) == n {
			k.scheduleSteps(i, a)
		}
	}
	return len(k.v.rep.errs) == n
}

// scheduleSteps is schedule_step for one actor whose steps name levers inside their
// bounds: with a step limit, two consecutive values of a lever differ by at most MaxStep,
// the first from the default and, with a cycle, the first again from the last.
func (k *policyCheck) scheduleSteps(i int, a PolicyActor) {
	order := make([]int, len(a.Schedule))
	for x := range order {
		order[x] = x
	}
	slices.SortStableFunc(order, func(x, y int) int {
		return cmp.Or(cmp.Compare(a.Schedule[x].Lever, a.Schedule[y].Lever), cmp.Compare(a.Schedule[x].At, a.Schedule[y].At))
	})
	for from := 0; from < len(order); {
		to := from
		for to < len(order) && a.Schedule[order[to]].Lever == a.Schedule[order[from]].Lever {
			to++
		}
		l, _ := actorLever(a, a.Schedule[order[from]].Lever)
		if l.MaxStep > 0 {
			k.leverSteps(i, a, l, order[from:to])
		}
		from = to
	}
}

// leverSteps checks the consecutive values of one lever's steps, given as indexes of
// the actor's schedule in ascending At.
func (k *policyCheck) leverSteps(i int, a PolicyActor, l Lever, idx []int) {
	prev := int64(policyDefault)
	for _, x := range idx {
		if abs64(a.Schedule[x].BP-prev) > l.MaxStep {
			k.shape("schedule_step", policyListPath(i, "schedule", x, "bp"), i64(a.Schedule[x].BP))
		}
		prev = a.Schedule[x].BP
	}
	if first := idx[0]; a.Cycle > 0 && abs64(a.Schedule[first].BP-prev) > l.MaxStep {
		k.shape("schedule_step", policyListPath(i, "schedule", first, "bp"), i64(a.Schedule[first].BP))
	}
}

// abs64 is |a| for values inside the slot bounds, which no subtraction can overflow.
func abs64(a int64) int64 {
	if a < 0 {
		return -a
	}
	return a
}

// reactions is ErrUnknownCommodity and ErrPolicyShape, rules rule_lever, rule_gauge and
// rule_reads (spec-policy.md 11.1, rule 13).
func (k *policyCheck) reactions() bool {
	n := len(k.v.rep.errs)
	for i, a := range k.m.Actors {
		for j, r := range a.Rules {
			if _, ok := actorLever(a, r.Lever); !ok {
				k.shape("rule_lever", policyListPath(i, "rules", j, "lever"), r.Lever)
			}
			k.gaugeShape(i, j, a, r)
		}
	}
	return len(k.v.rep.errs) == n
}

// gaugeShape checks the gauge of a reaction and the fields it needs.
func (k *policyCheck) gaugeShape(i, j int, a PolicyActor, r Reaction) {
	p := policyListPath(i, "rules", j, "gauge")
	switch r.Gauge {
	case GaugeTreasury, GaugeUnfedShare, GaugeMeanRatio:
		if r.Commodity != "" || r.MaxAge != 0 {
			k.shape("rule_gauge", p, string(r.Gauge))
		}
	case GaugePrice:
		if r.Commodity == "" || r.MaxAge == 0 {
			k.shape("rule_gauge", p, string(r.Gauge))
		} else if _, ok := k.v.lk.commodity[r.Commodity]; !ok {
			k.v.rep.add(ErrUnknownCommodity, policyListPath(i, "rules", j, "commodity"), r.Commodity, "")
		}
	default:
		k.shape("rule_gauge", p, string(r.Gauge))
		return
	}
	if r.Gauge != GaugeTreasury && !a.ReadsRegions {
		k.shape("rule_reads", p, string(r.Gauge))
	}
}

// governed is ErrNotGoverned: a treasury gauge on a jurisdiction with no seat.
func (k *policyCheck) governed() bool {
	n := len(k.v.rep.errs)
	for i, a := range k.m.Actors {
		j, _ := k.pol.Lookup(a.Jurisdiction)
		if _, seat := k.pol.Seat(j); seat {
			continue
		}
		for ri, r := range a.Rules {
			if r.Gauge == GaugeTreasury {
				k.v.rep.add(ErrNotGoverned, policyListPath(i, "rules", ri, "gauge"), a.Jurisdiction, "")
			}
		}
	}
	return len(k.v.rep.errs) == n
}

// policyLeverEdges returns the edges an Against lever of jurisdiction j against k holds:
// Between(g, j, k) and Between(g, k, j), merged in ascending EdgeID (spec-policy.md
// 3.4.3).
func policyLeverEdges(pol Polity, g Graph, j, other JurisdictionID) []EdgeID {
	out := append(pol.Between(g, j, other), pol.Between(g, other, j)...)
	slices.Sort(out)
	return slices.Compact(out)
}

// policyEdgeIDs returns the edges an edge lever of jurisdiction j holds, in ascending
// EdgeID: the group an Against expands to, or the one edge (From, To) (spec-policy.md
// 3.4.3). The lever is valid, so its names are found. The world resolves its levers
// with it at construction, and the validator reads the keys through it.
func policyEdgeIDs(pol Polity, g Graph, j JurisdictionID, l Lever) []EdgeID {
	if l.Against != "" {
		other, _ := pol.Lookup(l.Against)
		return policyLeverEdges(pol, g, j, other)
	}
	from, _ := g.Lookup(l.From)
	to, _ := g.Lookup(l.To)
	e, _ := g.Edge(from, to)
	return []EdgeID{e.ID}
}

// leverKeys returns the keys a lever holds as comparable text, in ascending key order:
// the flow ID, the edge IDs, or the transit triple (spec-policy.md 3.4.4, 3.5.2).
func (k *policyCheck) leverKeys(j JurisdictionID, l Lever) []string {
	switch l.Kind {
	case LeverFlow:
		return []string{"flow:" + l.Flow}
	case LeverTransit:
		return []string{"transit:" + l.From + ">" + l.To + ":" + l.Mode}
	}
	edges := policyEdgeIDs(k.pol, k.g, j, l)
	out := make([]string, len(edges))
	for x, e := range edges {
		out[x] = "edge:" + itoa(int(e))
	}
	return out
}

// keys is ErrTooMany for an Against that expands to more than MaxLeverKeys, ErrDuplicateID
// for a key named by two levers of one actor and ErrLeverShared for a key held by two
// actors, after the expansion of every Against (spec-policy.md 3.5.2). It names the
// first holder as the Bound.
func (k *policyCheck) keys() bool {
	n := len(k.v.rep.errs)
	holder := map[string]string{} // key to the actor that holds it; a lookup, read in actor order
	for i, a := range k.m.Actors {
		j, _ := k.pol.Lookup(a.Jurisdiction)
		for li, l := range a.Levers {
			keys := k.leverKeys(j, l)
			if len(keys) > MaxLeverKeys {
				k.v.rep.add(ErrTooMany, policyListPath(i, "levers", li, "against"), itoa(len(keys)), itoa(MaxLeverKeys))
				continue
			}
			for _, key := range keys {
				switch owner, held := holder[key]; {
				case held && owner == a.ID:
					k.v.rep.add(ErrDuplicateID, policyListPath(i, "levers", li, "id"), key, "")
				case held:
					k.v.rep.add(ErrLeverShared, policyListPath(i, "levers", li, "id"), key, owner)
				default:
					holder[key] = a.ID
				}
			}
		}
	}
	return len(k.v.rep.errs) == n
}
