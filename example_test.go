package economysim_test

import (
	"errors"
	"fmt"

	economysim "github.com/Whealinghold/economy-sim"
)

// exampleConfig is the smallest valid configuration: one raw commodity that
// households consume, one role, one region and one group of ten households.
func exampleConfig() economysim.Config {
	world := economysim.DefaultWorld()
	world.Upkeep = []economysim.Item{{Commodity: "grain", Quantity: 1}}
	grain := economysim.DefaultCommodity()
	grain.ID, grain.ReferencePrice, grain.Raw = "grain", 1_000, true
	region := economysim.DefaultRegion()
	region.ID = "home"
	role := economysim.DefaultRole()
	role.ID = "farmer"
	return economysim.Config{
		World:       world,
		Calendar:    economysim.DefaultCalendar(),
		Commodities: []economysim.Commodity{grain},
		Roles:       []economysim.Role{role},
		Regions:     []economysim.Region{region},
		Groups:      []economysim.Group{{Region: "home", Role: "farmer", Households: 10}},
	}
}

// A configuration is plain data. Validate returns every problem it finds, and
// each can be matched with errors.Is against a sentinel of the package.
func ExampleConfig_Validate() {
	c := exampleConfig()
	fmt.Println("valid:", c.Validate() == nil)

	c.Groups = append(c.Groups, economysim.Group{Region: "home", Role: "baker", Households: 5})
	err := c.Validate()
	fmt.Println("unknown role:", errors.Is(err, economysim.ErrUnknownRole))
	// Output:
	// valid: true
	// unknown role: true
}

// Warnings reports what a valid configuration does that is probably not what
// its author meant. A warning never fails the configuration.
func ExampleConfig_Warnings() {
	c := exampleConfig()
	fmt.Println("valid:", c.Validate() == nil)
	for _, w := range c.Warnings() {
		fmt.Println(w.Rule, w.Path)
	}
	// Output:
	// valid: true
	// raw_without_source [commodities grain]
	// start_stock_below_target [roles farmer start_stock]
}

// HashConfig identifies the configuration: the same value gives the same hash,
// and any field that can change a result changes it.
func ExampleHashConfig() {
	a, b := exampleConfig(), exampleConfig()
	fmt.Println("same configuration, same hash:", economysim.HashConfig(a) == economysim.HashConfig(b))

	b.Groups[0].Households++
	fmt.Println("more households, different hash:", economysim.HashConfig(a) != economysim.HashConfig(b))
	// Output:
	// same configuration, same hash: true
	// more households, different hash: true
}
