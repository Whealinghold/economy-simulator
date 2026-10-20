package economysim

import (
	"errors"
	"reflect"
	"slices"
	"testing"
)

// viaPerturb applies one edit of the menu to the toy configuration: op picks the
// edit and arg its size. Every edit keeps the references valid, so most inputs
// reach phase 6; the ones that do not still must not panic.
func viaPerturb(c *Config, op, arg uint8) {
	switch op % 12 {
	case 0:
		c.Recipes[0].Duration = 1 + int64(arg)%100
	case 1:
		c.Recipes[0].Outputs[0].Quantity = 1 + Quantity(arg)
	case 2:
		c.Recipes[0].Sited = arg%4 != 0
		switch arg % 4 {
		case 1:
			c.Yields = []YieldProfile{viaSpring("north", "farm")}
		case 2:
			c.Yields = []YieldProfile{{Region: "north", Recipe: "farm", Seasons: []SeasonYield{{Season: "autumn", BP: 10_000}, {Season: "spring", BP: 5_000}}}}
		case 3:
			c.Yields = []YieldProfile{{Region: "north", Recipe: "farm", HasAll: true, All: int64(arg) * 100}}
		}
	case 3:
		c.Commodities[0].ShelfLife = int64(arg) * 4
	case 4:
		c.Groups[0].Households = 1 + int64(arg)
	case 5:
		viaTwoRegions(c)
	case 6:
		c.Commodities = append(c.Commodities, viaCommodity("berry", 50, func(m *Commodity) { m.ShelfLife = int64(arg) * 2 }))
		c.Recipes = append(c.Recipes,
			viaRecipe("pick", nil, []Output{viaOut("berry", 8)}, func(r *Recipe) { r.Sited = true }),
			viaRecipe("jam", []Item{{Commodity: "berry", Quantity: 1}}, []Output{viaOut("grain", 4)}, nil))
		c.Roles[0].Recipes = append(c.Roles[0].Recipes, "pick", "jam")
		c.Yields = append(c.Yields, viaSpring("north", "pick"))
	case 7:
		c.Commodities = append(c.Commodities, viaCommodity("seed", 50, func(m *Commodity) { m.Raw = arg%2 == 0 }))
		c.Recipes = append(c.Recipes,
			viaRecipe("grow", []Item{{Commodity: "seed", Quantity: 1 + Quantity(arg%3)}}, []Output{viaOut("grain", 5)}, nil),
			viaRecipe("thresh", []Item{{Commodity: "grain", Quantity: 4}}, []Output{viaOut("seed", 1+Quantity(arg%3))}, nil))
		c.Roles[0].Recipes = append(c.Roles[0].Recipes, "grow", "thresh")
	case 8:
		c.World.CostFloor = 1_000 + int64(arg)*100
	case 9:
		c.Roles[0].ExpectedWorkTicks, c.Roles[0].MinWorkShare = 1+int64(arg), 0
	case 10:
		c.Commodities = append(c.Commodities, viaCommodity("rock", 50, nil))
		c.Recipes = append(c.Recipes, viaRecipe("quarry", nil, []Output{viaOut("rock", 1)}, func(r *Recipe) { r.Sited = true }))
		idle := DefaultRole()
		idle.ID, idle.Recipes = "idle", []string{"quarry"}
		c.Roles = append(c.Roles, idle)
		c.Groups = append(c.Groups, Group{Region: "north", Role: "idle", Households: 1 + int64(arg)})
	case 11:
		switch arg % 3 {
		case 0:
			c.Calendar.Seasons = nil
		case 1:
			c.Calendar.TicksPerDay = 2
		case 2:
			c.Calendar.StartDay = int64(arg)
		}
		c.Roles[0].ExpectedWorkTicks, c.Roles[0].MinWorkShare = 50, 0
	}
}

// viaReversed returns the configuration with every entity list reversed, which a
// canonical configuration is not.
func viaReversed(c Config) Config {
	r := c
	r.Commodities, r.Recipes, r.Yields = slices.Clone(c.Commodities), slices.Clone(c.Recipes), slices.Clone(c.Yields)
	r.Roles, r.Regions, r.Groups = slices.Clone(c.Roles), slices.Clone(c.Regions), slices.Clone(c.Groups)
	slices.Reverse(r.Commodities)
	slices.Reverse(r.Recipes)
	slices.Reverse(r.Yields)
	slices.Reverse(r.Roles)
	slices.Reverse(r.Regions)
	slices.Reverse(r.Groups)
	return r
}

// FuzzCheck verifies, over the toy configuration with up to four edits from a
// menu, that Validate and Warnings never panic, that every error is a
// *ValidationError, a *ViabilityError that matches ErrNotViable, or the cut, that
// the verdict and the warnings are the same twice and the same when the entity
// lists are in reverse order (the result depends on the configuration, not on
// the order it was written in), and that the receiver is not changed.
//
// Requirement: NFR-VIA-004, FR-VIA-020, FR-VIA-022
func FuzzCheck(f *testing.F) {
	for _, s := range [][]byte{
		nil, {2, 1}, {2, 1, 3, 1}, {5, 0, 0, 2}, {6, 50}, {7, 0}, {7, 1}, {10, 3}, {1, 0, 1, 0}, {3, 25, 2, 1},
		{8, 200, 7, 0}, {11, 0, 5, 0}, {11, 1, 2, 3}, {4, 255, 9, 5}, {0, 99, 1, 200},
	} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, ops []byte) {
		build := func() Config {
			c := viaToy()
			for i := 0; i+1 < len(ops) && i < 8; i += 2 {
				viaPerturb(&c, ops[i], ops[i+1])
			}
			return c
		}
		c := build()
		err := c.Validate()
		if err != nil {
			joined, ok := err.(interface{ Unwrap() []error })
			if !ok {
				joined = joinedOf(err)
			}
			for _, e := range joined.Unwrap() {
				var ve *ValidationError
				var vi *ViabilityError
				switch {
				case errors.As(e, &ve), errors.Is(e, ErrTooManyErrors):
				case errors.As(e, &vi) && errors.Is(e, ErrNotViable):
				default:
					t.Fatalf("Validate(%v) returned %v, want a *ValidationError, a *ViabilityError or the cut", ops, e)
				}
			}
		}
		again, rev := build(), viaReversed(build())
		if err2 := again.Validate(); (err == nil) != (err2 == nil) || (err != nil && err.Error() != err2.Error()) {
			t.Fatalf("Validate(%v) is not deterministic:\n%v\n%v", ops, err, err2)
		}
		if errRev := rev.Validate(); (err == nil) != (errRev == nil) || (err != nil && errors.Is(err, ErrNotViable) != errors.Is(errRev, ErrNotViable)) {
			t.Fatalf("Validate(%v) depends on the order of the lists:\n%v\n%v", ops, err, errRev)
		}
		if !reflect.DeepEqual(c.Warnings(), again.Warnings()) || (err == nil && !reflect.DeepEqual(c.Warnings(), rev.Warnings())) {
			t.Fatalf("Warnings(%v) differ between two runs or between two orders of the lists", ops)
		}
		if !reflect.DeepEqual(c, build()) {
			t.Fatalf("Validate(%v) changed its receiver", ops)
		}
	})
}

// FuzzVirtualStartOffsets verifies, over a recipe of any duration up to 2 000
// ticks on a calendar with any start and any yield per season, that the offsets
// are ascending, inside [0, d) and deterministic, that they match the definition
// by Calendar.SeasonAt, and that the run-based test of an empty set agrees with
// them (FR-VIA-033, FR-VIA-053).
//
// Requirement: NFR-VIA-004, FR-VIA-053, FR-VIA-033
func FuzzVirtualStartOffsets(f *testing.F) {
	f.Add(uint16(60), uint16(59), uint16(0), uint8(0), uint8(100), uint8(0), uint8(0), true)
	f.Add(uint16(365), uint16(59), uint16(0), uint8(0), uint8(100), uint8(0), uint8(0), true)
	f.Add(uint16(1), uint16(0), uint16(300), uint8(100), uint8(0), uint8(100), uint8(0), true)
	f.Add(uint16(2000), uint16(200), uint16(7), uint8(0), uint8(0), uint8(0), uint8(100), false)
	f.Add(uint16(12), uint16(10), uint16(364), uint8(50), uint8(50), uint8(50), uint8(50), true)
	f.Fuzz(func(t *testing.T, d, seasonStart, startDay uint16, spring, summer, autumn, winter uint8, sited bool) {
		dur := 1 + int64(d)%2000
		c := viaToy()
		c.Calendar.SeasonStart, c.Calendar.StartDay = int64(seasonStart)%365, int64(startDay)%365
		c.Recipes[0].Duration, c.Recipes[0].Sited = dur, sited
		c.Recipes[0].Outputs[0].Quantity = Quantity(10 * dur)
		var ys []SeasonYield
		for i, f := range []uint8{spring, summer, autumn, winter} {
			if f > 0 {
				ys = append(ys, SeasonYield{Season: []string{"spring", "summer", "autumn", "winter"}[i], BP: int64(f) * 100})
			}
		}
		if len(ys) > 0 {
			c.Yields = []YieldProfile{{Region: "north", Recipe: "farm", Seasons: ys}}
		}
		v := &validator{c: c}
		v.structural()
		if len(v.rep.errs) > 0 {
			return // not a configuration the tables are built for
		}
		cc := c.canonical()
		m := newViaModel(&cc)
		got := virtualStartOffsets(&cc, &m.yt, 0, 0)
		if !slices.Equal(got, virtualStartOffsets(&cc, &m.yt, 0, 0)) || !slices.Equal(got, viaBruteOffsets(&cc, m, 0, 0)) {
			t.Fatalf("offsets (d=%d) are not deterministic or differ from the definition: %v", dur, got)
		}
		if !slices.IsSorted(got) || (len(got) > 0 && (got[0] < 0 || got[len(got)-1] >= dur)) {
			t.Fatalf("offsets (d=%d) = %v, want ascending and in [0, d)", dur, got)
		}
		if empty := noVirtualStart(m.runsFor(0, 0), m.year, dur); empty != (len(got) == 0) {
			t.Fatalf("noVirtualStart (d=%d) = %v with %d offsets, want them to agree", dur, empty, len(got))
		}
	})
}
