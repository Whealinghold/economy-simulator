package economysim_test

import (
	"context"
	"fmt"
	"math/rand/v2"

	economysim "github.com/Whealinghold/economy-sim"
)

// A world is built from a configuration, stepped one tick at a time or run for
// a number of ticks, and read through its snapshot. With no logger and no
// observer it writes nothing and emits nothing.
func ExampleNewWorld() {
	w, err := economysim.NewWorld(exampleConfig(), economysim.WithSeed(1))
	if err != nil {
		fmt.Println("build:", err)
		return
	}
	fmt.Println("tick:", w.Tick())
	if err := w.Step(); err != nil {
		fmt.Println("step:", err)
		return
	}
	done, err := w.Run(context.Background(), 4)
	fmt.Println("ran:", done, err, "tick:", w.Tick())
	snap, err := w.Snapshot()
	fmt.Println("snapshot:", snap.Tick, snap.Agents, snap.Households, err)
	// Output:
	// tick: 0
	// ran: 4 <nil> tick: 5
	// snapshot: 5 10 10 <nil>
}

// retirer is a consumer Spawner: in the first tick it removes the first agent of
// the region, and nothing after.
type retirer struct{}

// Decide appends the one removal.
func (retirer) Decide(v *economysim.RegionView, _ *rand.Rand, d *economysim.Decisions) {
	if v.Tick == 0 && len(v.Agents) > 0 {
		d.List = append(d.List, economysim.Decision{Kind: economysim.DecideRemove, Agent: v.Agents[0].Agent, Cause: economysim.CauseConsumer})
	}
}

// A consumer registers its Spawner under a name, and the configuration selects
// it by that name. The world executes the plan the spawner returns and reports
// the removal as an event.
func ExampleWithSpawner() {
	c := exampleConfig()
	c.World.Spawner = "retire"
	watch := economysim.ObserverFunc(func(e economysim.Event) {
		if b, ok := e.(economysim.AgentBankrupted); ok {
			fmt.Println("removed agent", b.Agent, "households", b.Weight, "cause", b.Cause)
		}
	})
	w, err := economysim.NewWorld(c, economysim.WithSpawner("retire", retirer{}), economysim.WithObserver(watch))
	if err != nil {
		fmt.Println("build:", err)
		return
	}
	if err := w.Step(); err != nil {
		fmt.Println("step:", err)
		return
	}
	snap, _ := w.Snapshot()
	fmt.Println("agents:", snap.Agents, "households:", snap.Households)
	// Output:
	// removed agent 1 households 1 cause consumer
	// agents: 9 households: 9
}

// counting is a consumer Clearing that counts its calls and delegates to the
// default.
type counting struct{ calls *int }

// Clear counts the call and matches as AverageClearing does.
func (c counting) Clear(bids, asks []economysim.Order, rng *rand.Rand) []economysim.Trade {
	*c.calls++
	return economysim.AverageClearing{}.Clear(bids, asks, rng)
}

// A consumer registers a Clearing under a name and the configuration selects it.
// The world calls it once for each market in each tick.
func ExampleWithClearing_world() {
	var calls int
	c := exampleConfig()
	c.World.Clearing = "counting"
	w, err := economysim.NewWorld(c, economysim.WithClearing("counting", counting{calls: &calls}))
	if err != nil {
		fmt.Println("build:", err)
		return
	}
	if _, err := w.Run(context.Background(), 3); err != nil {
		fmt.Println("run:", err)
		return
	}
	fmt.Println("clearing calls in 3 ticks of 1 market:", calls)
	// Output:
	// clearing calls in 3 ticks of 1 market: 3
}
