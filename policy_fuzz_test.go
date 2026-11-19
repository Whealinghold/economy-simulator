package economysim_test

import (
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// FuzzValidatePolicy verifies, for random fields of an actor, a lever, a schedule step and
// a reaction laid over a valid configuration, that Validate and Warnings never panic, that
// a configuration Validate accepts hashes the same twice and survives the canonical order
// of its lists, and that a rejected one reports an error that names a sentinel.
//
// Requirement: NFR-PLC-003
func FuzzValidatePolicy(f *testing.F) {
	f.Add("crown", int64(10), int64(0), int64(5_000), int64(120), int64(15_000), "foot")
	f.Add("", int64(-1), int64(-1), int64(-1), int64(-1), int64(-1), "")
	f.Add("x", int64(1)<<62, int64(1)<<62, int64(1)<<62, int64(1)<<62, int64(1)<<62, "boat")
	f.Fuzz(func(t *testing.T, id string, every, lo, step, at, bp int64, mode string) {
		jurisdiction, flow, hi := id, id, lo+bp
		c := policyConfig()
		a := &c.Modules.Policy.Actors[0]
		a.ID, a.Jurisdiction, a.Every = id, jurisdiction, every
		a.Levers[0].Min, a.Levers[0].Max, a.Levers[0].MaxStep, a.Levers[0].Flow = lo, hi, step, flow
		a.Levers[1].Mode = mode
		a.Schedule[1].At, a.Schedule[1].BP = at, bp
		c.Modules.Policy.Actors[2].Rules[0].Span, c.Modules.Policy.Actors[2].Rules[0].Move = bp, at
		err := c.Validate()
		_ = c.Warnings() // must not panic, valid or not
		if err != nil {
			if vs, _ := violations(err); len(vs) == 0 {
				t.Fatalf("Validate() = %v, want a *ValidationError", err)
			}
			return
		}
		first := economysim.HashConfig(c)
		if first != economysim.HashConfig(c) {
			t.Fatalf("HashConfig of an accepted configuration is not stable: %+v", c.Modules.Policy)
		}
		m := c.Modules.Policy
		m.Actors[0], m.Actors[2] = m.Actors[2], m.Actors[0]
		if c.Validate() != nil {
			t.Fatalf("Validate rejected an accepted configuration with its actors in another order: %+v", m)
		}
	})
}
