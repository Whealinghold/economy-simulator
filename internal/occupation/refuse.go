package occupation

// Kind is the kind of a spawner decision, as Refuse reads it.
type Kind int

// The decision kinds of spec-spawner.md 3.3 in the slice.
const (
	KindRemove Kind = iota
	KindReduce
	KindSwitch
	KindEnter
	KindApply // appended with SKL: join a role's traineeship queue
)

// Refusal is the first check of spec-spawner.md 4 that a decision fails. The
// values are appended in module order, so they are not the order of the checks
// (the functions below fix that order); the root maps each to a Reason.
// on_journey is appended, with its Facts field, by the migration module, and
// in_traineeship, needs_weight_one, needs_traineeship and already_queued by the
// skill barriers.
type Refusal int

// The slice's refusals. RefusalNone means the decision passes every check.
const (
	RefusalNone Refusal = iota
	UnknownAgent
	UnknownRole
	ExternalAgent
	SameRole
	BadHouseholds
	UnsupportedCause // appended last (spec-spawner.md 4)
	OnJourney        // appended with MIG; checked after SameRole for a switch

	// Appended with SKL (spec-spawner.md 4). They are checked in the order of the
	// table: in_traineeship after same_role and before on_journey, then
	// needs_weight_one, needs_traineeship and already_queued, before
	// bad_households.
	InTraineeship
	NeedsWeightOne
	NeedsTraineeship
	AlreadyQueued

	// RefusalEnd is one past the last refusal, for a test that enumerates them.
	RefusalEnd
)

// Facts are what the executor knows about a decision against live state.
// External is true for an agent that is external and not mortal. Weight is the
// agent's weight. CauseSupported is false only for a reduction whose cause is
// neither starved nor insolvent. A field a kind does not use is ignored.
type Facts struct {
	AgentKnown, RoleKnown, External, SameRole, CauseSupported bool
	// OnJourney is true for an agent on a migration journey; only a switch reads it.
	OnJourney bool
	// InTraineeship is true for a trainee (a switch or an apply) or a trainer with
	// trainees (a switch); the root decides which per kind.
	InTraineeship bool
	// TrainingRole is true when the role a switch, an enter or an apply names has
	// TrainingTicks above 0. Queued is true for an agent already in a queue.
	TrainingRole, Queued bool
	Households, Weight   int64
}

// Refuse returns the first failing check for a decision of kind k in the order
// of spec-spawner.md 4, or RefusalNone. A removal names an agent only, a
// reduction an agent and a count, a switch an agent and a role, an entry a role
// and a count.
func Refuse(k Kind, f Facts) Refusal {
	switch k {
	case KindRemove:
		return refuseAgent(f, false)
	case KindReduce:
		return refuseReduce(f)
	case KindSwitch:
		return refuseSwitch(f)
	case KindEnter:
		return refuseEnter(f)
	case KindApply:
		return refuseApply(f)
	}
	return RefusalNone
}

// refuseSwitch runs the checks of a switch in the order of spec-spawner.md 4:
// the agent and the role, same_role, in_traineeship, on_journey, then
// needs_traineeship for a role that is entered only by Apply.
func refuseSwitch(f Facts) Refusal {
	if r := refuseAgent(f, true); r != RefusalNone {
		return r
	}
	switch {
	case f.SameRole:
		return SameRole
	case f.InTraineeship:
		return InTraineeship
	case f.OnJourney:
		return OnJourney
	case f.TrainingRole:
		return NeedsTraineeship
	}
	return RefusalNone
}

// refuseEnter runs the checks of an entry: the role, needs_traineeship, then the
// count.
func refuseEnter(f Facts) Refusal {
	switch {
	case !f.RoleKnown:
		return UnknownRole
	case f.TrainingRole:
		return NeedsTraineeship
	case f.Households < 1:
		return BadHouseholds
	}
	return RefusalNone
}

// refuseApply runs the checks of an Apply in the order of spec-spawner.md 4:
// the agent, the role and the external check, then in_traineeship, on_journey,
// needs_weight_one and already_queued. An apply to the agent's own role is not
// a same_role case: the table names a switch only.
func refuseApply(f Facts) Refusal {
	if r := refuseAgent(f, true); r != RefusalNone {
		return r
	}
	switch {
	case f.InTraineeship:
		return InTraineeship
	case f.OnJourney:
		return OnJourney
	case f.Weight > 1:
		return NeedsWeightOne
	case f.Queued:
		return AlreadyQueued
	}
	return RefusalNone
}

func refuseReduce(f Facts) Refusal {
	if r := refuseAgent(f, false); r != RefusalNone {
		return r
	}
	if f.Households < 1 || f.Households > f.Weight {
		return BadHouseholds
	}
	if !f.CauseSupported {
		return UnsupportedCause
	}
	return RefusalNone
}

// refuseAgent runs the checks on the agent a decision names: known, then the
// role when the decision names one, then external.
func refuseAgent(f Facts, needRole bool) Refusal {
	switch {
	case !f.AgentKnown:
		return UnknownAgent
	case needRole && !f.RoleKnown:
		return UnknownRole
	case f.External:
		return ExternalAgent
	}
	return RefusalNone
}
