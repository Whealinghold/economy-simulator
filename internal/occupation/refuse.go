package occupation

// Kind is the kind of a spawner decision, as Refuse reads it.
type Kind int

// The decision kinds of spec-spawner.md 3.3 in the slice.
const (
	KindRemove Kind = iota
	KindReduce
	KindSwitch
	KindEnter
)

// Refusal is the first check of spec-spawner.md 4 that a decision fails. The
// values are in the order of the checks; the root maps each to a Reason.
// in_traineeship, needs_weight_one and on_journey are appended, with their Facts
// field, by the skills and migration modules.
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

	// RefusalEnd is one past the last refusal, for a test that enumerates them.
	RefusalEnd
)

// Facts are what the executor knows about a decision against live state.
// External is true for an agent that is external and not mortal. Weight is the
// agent's weight. CauseSupported is false only for a reduction whose cause is
// neither starved nor insolvent. A field a kind does not use is ignored.
type Facts struct {
	AgentKnown, RoleKnown, External, SameRole, CauseSupported bool
	Households, Weight                                        int64
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
		if r := refuseAgent(f, true); r != RefusalNone {
			return r
		}
		if f.SameRole {
			return SameRole
		}
	case KindEnter:
		if !f.RoleKnown {
			return UnknownRole
		}
		if f.Households < 1 {
			return BadHouseholds
		}
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
