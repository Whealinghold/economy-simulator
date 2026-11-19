package config

import (
	"strconv"

	economysim "github.com/Whealinghold/economy-sim"
)

// The keys of the policy module (spec-policy.md 3.1.3, 10), in a file of their own so
// that mapping.go does not grow.

// policy reads [modules.policy] (spec-config.md 6.2, spec-policy.md 3.1.3): an actor as
// [actor.<id>] with the keys jurisdiction, policy, every, reads_regions, cycle and
// label; its levers as [actor.<id>.lever.<id>], its schedule steps as
// [actor.<id>.schedule.<lever>.<tick>] and its reactions as [actor.<id>.rule.<lever>].
// A key-less table switches the module on with no actor. Every level is read in the
// order of the keys; Validate and the hash apply the canonical order.
func (l *loader) policy(t *tbl) *economysim.PolicyConfig {
	var m economysim.PolicyConfig
	t.entities("actor", func(id string, e *tbl) {
		a := economysim.PolicyActor{ID: id}
		e.need("jurisdiction", e.text("jurisdiction", &a.Jurisdiction))
		e.need("policy", e.text("policy", &a.Policy))
		e.need("every", integer(e, "every", &a.Every))
		e.flag("reads_regions", &a.ReadsRegions)
		integer(e, "cycle", &a.Cycle)
		e.text("label", &a.Label)
		l.policyLevers(e, &a)
		l.policySteps(e, &a)
		l.policyRules(e, &a)
		m.Actors = append(m.Actors, a)
	})
	return &m
}

// policyLevers reads [actor.<id>.lever.<id>].
func (l *loader) policyLevers(e *tbl, a *economysim.PolicyActor) {
	e.entities("lever", func(id string, t *tbl) {
		lv := economysim.Lever{ID: id}
		var kind string
		t.need("kind", t.text("kind", &kind))
		lv.Kind = economysim.LeverKind(kind)
		t.text("flow", &lv.Flow)
		t.text("from", &lv.From)
		t.text("to", &lv.To)
		t.text("mode", &lv.Mode)
		t.text("against", &lv.Against)
		t.need("min", integer(t, "min", &lv.Min))
		t.need("max", integer(t, "max", &lv.Max))
		integer(t, "max_step", &lv.MaxStep)
		t.flag("may_close", &lv.MayClose)
		a.Levers = append(a.Levers, lv)
	})
}

// policySteps reads [actor.<id>.schedule.<lever>.<tick>]: the second key is the tick, an
// integer written as the table's name (a name that is not one is ErrWrongType at its
// path).
func (l *loader) policySteps(e *tbl, a *economysim.PolicyActor) {
	e.entities("schedule", func(lever string, t *tbl) {
		t.eachSub(func(tick string, s *tbl) {
			at, err := strconv.ParseInt(tick, 10, 64)
			if err != nil {
				l.fail(economysim.ErrWrongType, s.path)
				return
			}
			step := economysim.ScheduleStep{Lever: lever, At: at}
			s.need("bp", integer(s, "bp", &step.BP))
			s.flag("closed", &step.Closed)
			a.Schedule = append(a.Schedule, step)
		})
	})
}

// policyRules reads [actor.<id>.rule.<lever>].
func (l *loader) policyRules(e *tbl, a *economysim.PolicyActor) {
	e.entities("rule", func(lever string, t *tbl) {
		r := economysim.Reaction{Lever: lever}
		var gauge string
		t.need("gauge", t.text("gauge", &gauge))
		r.Gauge = economysim.Gauge(gauge)
		t.text("commodity", &r.Commodity)
		readInts(t, []intField{
			{"max_age", &r.MaxAge}, {"target", &r.Target}, {"deadband", &r.Deadband}, {"span", &r.Span}, {"move", &r.Move},
		})
		t.flag("raise", &r.Raise)
		a.Rules = append(a.Rules, r)
	})
}
