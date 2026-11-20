package economysim

import (
	"cmp"
	"fmt"
	"slices"
)

// This file is the state of the policy module that a call or a checkpoint touches
// (spec-policy.md 8.4, 9.1, 12.3): the mailbox of a directed actor, DirectPolicy that fills
// it and the rows of Snapshot.Actors. The levers' values are not here: they are the
// owners' slots.

// PolicyDirective is the whole content of one actor's mailbox.
type PolicyDirective struct {
	Actor ActorID
	// Levers is the new content: sorted by Lever or not (the world sorts); empty clears
	// the mailbox.
	Levers []LeverSetting
}

// ActorRow is one policy actor of a snapshot (spec-policy.md 12.3.1).
type ActorRow struct {
	Actor        ActorID
	Jurisdiction JurisdictionID
	// Pending is the number of settings in the mailbox of a directed actor, 0 for an
	// actor of another policy.
	Pending int64
}

// DirectPolicy replaces the mailbox of each named actor. It acts between ticks, is atomic,
// draws nothing and emits no event; the directive is applied at the actor's next decision
// tick, through the checks of spec-policy.md 6.2. It returns ErrInStep, ErrWorldFailed and
// ErrModuleOff, then, for the first entry that fails, ErrUnknownActor, ErrDuplicateID (an
// actor named twice), ErrNotDirected, and for its settings ErrUnknownLever, ErrDuplicateID
// (a lever named twice) and ErrOutOfBounds (the value outside [Min, Max] of the lever, or
// Closed on a lever that may not close), in the order of 9.1.2. The step limit is not a
// check of the call: it is taken against the value the lever has at the decision. A nil or
// empty d is a no-op; a refused call changes nothing.
func (w *World) DirectPolicy(d []PolicyDirective) error {
	if err := w.enter("DirectPolicy", "policy", w.policy != nil); err != nil {
		return err
	}
	for i := range d {
		if err := w.checkPolicyDirective(d, i); err != nil {
			return fmt.Errorf("DirectPolicy: %w", err)
		}
	}
	for i := range d {
		w.policy.setMailbox(d[i].Actor, d[i].Levers)
	}
	return nil
}

// checkPolicyDirective is the check of entry i of d, in the order of spec-policy.md 9.1.2:
// the actor, a repeat of it in the call, its policy, then each setting.
func (w *World) checkPolicyDirective(d []PolicyDirective, i int) error {
	e := &d[i]
	p := w.policy
	if e.Actor < 0 || int(e.Actor) >= len(p.actors) {
		return argError(ErrUnknownActor, itoa(int(e.Actor)), "", "d", entry("d", i), "actor")
	}
	for _, earlier := range d[:i] {
		if earlier.Actor == e.Actor {
			return argError(ErrDuplicateID, itoa(int(e.Actor)), "", "d", entry("d", i), "actor")
		}
	}
	a := &p.actors[e.Actor]
	if a.kind != kindDirected {
		return argError(ErrNotDirected, itoa(int(e.Actor)), "", "d", entry("d", i), "actor")
	}
	for k, s := range e.Levers {
		if f := checkPolicySetting(a, e.Levers, k, s); f != nil {
			return argError(f.rule, f.value, f.bound, "d", entry("d", i), entry("levers", k), f.field)
		}
	}
	return nil
}

// settingFault is what is wrong with one setting of a directive: the sentinel, the value,
// the bound and the field the path ends in.
type settingFault struct {
	rule         error
	value, bound string
	field        string
}

// checkPolicySetting checks setting k of levers against the actor's lever: it exists, no
// earlier setting names it, the value is inside [Min, Max] and Closed needs MayClose. It
// returns nil for a setting that passes.
func checkPolicySetting(a *policyActor, levers []LeverSetting, k int, s LeverSetting) *settingFault {
	if s.Lever < 0 || int(s.Lever) >= len(a.levers) {
		return &settingFault{rule: ErrUnknownLever, value: itoa(int(s.Lever)), field: "lever"}
	}
	if slices.ContainsFunc(levers[:k], func(x LeverSetting) bool { return x.Lever == s.Lever }) {
		return &settingFault{rule: ErrDuplicateID, value: itoa(int(s.Lever)), field: "lever"}
	}
	l := &a.levers[s.Lever]
	if s.BP < l.min || s.BP > l.max {
		return &settingFault{rule: ErrOutOfBounds, value: itoa(int(s.BP)), bound: "[" + itoa(int(l.min)) + ", " + itoa(int(l.max)) + "]", field: "bp"}
	}
	if s.Closed && !l.mayClose {
		return &settingFault{rule: ErrOutOfBounds, value: "true", bound: "the lever may not close", field: "closed"}
	}
	return nil
}

// setMailbox replaces the mailbox of actor with a copy of levers sorted by lever, reusing
// the storage the mailbox holds. The checks have passed: the levers are distinct.
func (s *policyState) setMailbox(actor ActorID, levers []LeverSetting) {
	m := append(s.mailbox[actor][:0], levers...)
	slices.SortFunc(m, func(x, y LeverSetting) int { return cmp.Compare(x.Lever, y.Lever) })
	s.mailbox[actor] = m
}

// takeMailbox returns the settings of actor's mailbox and empties it: a directive acts once,
// whatever the checks make of it (spec-policy.md 8.4.1). The result is valid until the
// next DirectPolicy, which cannot run before the checks of the decision that reads it.
func (s *policyState) takeMailbox(actor ActorID) Settings {
	m := s.mailbox[actor]
	s.mailbox[actor] = m[:0]
	return Settings{Levers: m}
}

// snapshotActors fills Snapshot.Actors: one row per actor in ascending ActorID (spec-policy.md
// 12.3.1). A nil module fills nothing.
func (w *World) snapshotActors(s *Snapshot) {
	p := w.policy
	if p == nil {
		return
	}
	s.Actors = make([]ActorRow, len(p.actors))
	for i := range p.actors {
		s.Actors[i] = ActorRow{Actor: ActorID(i), Jurisdiction: p.actors[i].j, Pending: int64(len(p.mailbox[i]))}
	}
}
