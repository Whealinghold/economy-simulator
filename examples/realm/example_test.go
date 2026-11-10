package main

import (
	"fmt"
	"log"

	"github.com/Whealinghold/economy-sim/denomination"
)

// ExampleGame loads the realm, adds a player with 12 345 minor units, steps ten ticks
// and prints the tick and the balance in the ladder of the RPG preset: the first lines
// of the README. The player has no directive, so its fallback strategy spends most of
// the money on upkeep goods in ten ticks.
func ExampleGame() {
	g, err := NewGame(1)
	if err != nil {
		log.Fatal(err)
	}
	if _, err := g.AddPlayer(PlayerSpec{Region: "crownlands", Role: "traveller", Money: 12_345}); err != nil {
		log.Fatal(err)
	}
	if _, err := g.Advance(10); err != nil {
		log.Fatal(err)
	}
	p, err := g.Player()
	if err != nil {
		log.Fatal(err)
	}
	f, err := denomination.NewFormatter(denomination.RPG())
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("tick %d, balance %s\n", g.Tick(), f.Format(int64(p.Money)))
	// Output: tick 10, balance 1 sp 6 cp
}
