package economysim_test

import (
	"fmt"

	economysim "github.com/Whealinghold/economy-sim"
)

// ExampleWorld_DirectPolicy shows a player governing the example jurisdictions of spec-policy.md 14: the
// coast's council is a directed actor holding a border against the inland province, and the player
// closes it at a factor of 20 000 by one directive. The call stores it; at the council's next due tick
// (the first, tick 0, for an actor of the first rotation slot) the border's two edges are written in
// one application with the cause policy, and the mailbox is empty again.
func ExampleWorld_DirectPolicy() {
	rec := &recorder{}
	w, err := economysim.NewWorld(policyConfig(), economysim.WithSeed(11), economysim.WithObserver(rec))
	if err != nil {
		fmt.Println("NewWorld:", err)
		return
	}
	// ActorID 0 is the coast council and LeverID 0 its gate (ascending IDs, spec-policy.md 3.2).
	err = w.DirectPolicy([]economysim.PolicyDirective{{Actor: 0, Levers: []economysim.LeverSetting{{Lever: 0, BP: 20_000, Closed: true}}}})
	if err != nil {
		fmt.Println("DirectPolicy:", err)
		return
	}
	pending := func() int64 {
		s, _ := w.Snapshot()
		return s.Actors[0].Pending
	}
	fmt.Println("pending after the call:", pending())
	if err := w.Step(); err != nil { // tick 0: the council is due
		fmt.Println("Step:", err)
		return
	}
	for _, e := range recordedOf[economysim.EdgeStatusChanged](rec) {
		fmt.Printf("tick %d: edge %d to %d closed=%t factor=%d cause=%s\n", e.Tick, e.From, e.To, e.Closed, e.FactorBP, e.Cause)
	}
	fmt.Println("pending after tick 0:", pending())
	// Output:
	// pending after the call: 1
	// tick 0: edge 2 to 4 closed=true factor=20000 cause=policy
	// tick 0: edge 4 to 2 closed=true factor=20000 cause=policy
	// pending after tick 0: 0
}
