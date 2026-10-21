package economysim_test

import (
	"context"
	"errors"
	"fmt"

	economysim "github.com/Whealinghold/economy-sim"
)

// exampleConfig is the smallest valid configuration: one raw commodity that
// households consume, one role, one region and one group of ten households.
func exampleConfig() economysim.Config {
	world := economysim.DefaultWorldParams()
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

// At reads the calendar's moment of a tick, SeasonAt only its season, and
// TicksPerYear the derived length of a year. Tick 59 of the Earth default is
// the first day of March and of the first season.
func ExampleCalendar_At() {
	cal := economysim.DefaultCalendar()
	m, err := cal.At(59)
	fmt.Println("error:", err)
	fmt.Println("month:", m.Month, "day of month:", m.DayOfMonth, "season:", m.Season)
	fmt.Println("season of tick 0:", cal.SeasonAt(0))
	fmt.Println("ticks per year:", cal.TicksPerYear())
	// Output:
	// error: <nil>
	// month: 2 day of month: 0 season: 0
	// season of tick 0: 3
	// ticks per year: 365
}

// SolveSteadyState solves the reference programme of a configuration and makes
// the warm-start table: here ten households that each make 4 grain a tick from
// nothing and eat 1, so the surplus factor is 4 and the grain is priced at the
// reference basket. The configuration it is given is not changed, and the table
// is plain data that Validate accepts.
func ExampleSolveSteadyState() {
	c := exampleConfig()
	c.Commodities[0].Raw = false
	farm := economysim.DefaultRecipe()
	farm.ID = "farm"
	farm.Outputs = []economysim.Output{{Commodity: "grain", Quantity: 4, CostShare: 10_000}}
	c.Recipes = []economysim.Recipe{farm}
	c.Roles[0].Recipes = []string{"farm"}
	c.Regions[0].ReferenceBasket = 1_000

	st, err := economysim.SolveSteadyState(context.Background(), c)
	if err != nil {
		fmt.Println(err)
		return
	}
	p := st.Programmes[0]
	fmt.Printf("surplus %d bp, labour share %d bp\n", p.SurplusBP, p.LabourShareBP)
	fmt.Println("grain price:", st.Table.Prices[0].Price, "wage:", st.Table.Wages[0].Wage)
	c.WarmStart = &st.Table
	fmt.Println("valid with its table:", c.Validate() == nil)
	// Output:
	// surplus 40000 bp, labour share 2500 bp
	// grain price: 1000 wage: 4000
	// valid with its table: true
}
