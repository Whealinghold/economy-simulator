package economysim_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// The tests of the transport module's configuration surface: its modes, bounds,
// shape rules, warnings and the ledger rows it declares (spec-transport.md 3.1,
// 11, 12, FR-TRN-001 to 003, 037, 038). The module has no world effect yet.

// transportConfig returns referenceConfig with a second region, edges of 40 in
// both directions, and the transport module on with one self-carried mode, foot
// (speed 20, unit (40, 30), crew 1).
func transportConfig() economysim.Config {
	c := referenceConfig()
	south := economysim.DefaultRegion()
	south.ID = "south"
	c.Regions = append(c.Regions, south)
	c.Edges = []economysim.Edge{{From: "north", To: "south", Distance: 40}, {From: "south", To: "north", Distance: 40}}
	c.Modules.Transport = &economysim.Transport{Modes: []economysim.Mode{footMode()}}
	return c
}

func footMode() economysim.Mode {
	return economysim.Mode{ID: "foot", Speed: 20, UnitMass: 40, UnitBulk: 30, Crew: economysim.DefaultCrew}
}

// hiredMode returns a valid hired mode that hauls the commodity "seed".
func hiredMode() economysim.Mode {
	return economysim.Mode{ID: "coach", Speed: 30, UnitMass: 1_200, UnitBulk: 900, Haul: "seed", HaulSpan: 10}
}

// transportFieldCases are the failing cases of the rows of the transport module in
// the parameter-owner table (spec-transport.md 11, 12.1, FR-CFG-031): each breaks
// one field of a valid configuration. buildFieldCases appends them to the table
// that the registry test reads.
func transportFieldCases() []fieldCase {
	base := []string{"modules", "transport"}
	// with replaces the modes by the one the case builds from start.
	with := func(field, name string, rule error, start func() economysim.Mode, mutate func(m *economysim.Mode), path ...string) fieldCase {
		return fieldCase{field: field, name: name, rules: []error{rule}, paths: [][]string{append(slices.Clone(base), path...)},
			mutate: func(c *economysim.Config) {
				md := start()
				mutate(&md)
				c.Modules.Transport = &economysim.Transport{Modes: []economysim.Mode{md}}
			}}
	}
	self := func(field, name string, rule error, mutate func(m *economysim.Mode), path ...string) fieldCase {
		return with(field, name, rule, footMode, mutate, append([]string{"mode", "foot"}, path...)...)
	}
	hired := func(field, name string, rule error, mutate func(m *economysim.Mode), path ...string) fieldCase {
		return with(field, name, rule, hiredMode, mutate, append([]string{"mode", "coach"}, path...)...)
	}
	wagon := func() economysim.Mode {
		m := footMode()
		m.ID, m.Vehicle, m.WearPerTick = "wagon", "tool_a", 5
		return m
	}
	vehicle := func(field, name string, rule error, mutate func(m *economysim.Mode), path ...string) fieldCase {
		return with(field, name, rule, wagon, mutate, append([]string{"mode", "wagon"}, path...)...)
	}
	bounds := economysim.ErrOutOfBounds
	five := []economysim.Item{{Commodity: "fuel", Quantity: 1}, {Commodity: "mineral", Quantity: 1}, {Commodity: "prepared", Quantity: 1}, {Commodity: "refined", Quantity: 1}, {Commodity: "seed", Quantity: 1}}
	return []fieldCase{
		{field: "Transport.Modes", name: "more than MaxModes", rules: []error{bounds}, paths: [][]string{{"modules", "transport", "modes"}},
			mutate: func(c *economysim.Config) {
				c.Modules.Transport = &economysim.Transport{Modes: make([]economysim.Mode, economysim.MaxModes+1)}
			}},
		{field: "Mode.ID", name: "not an ID", rules: []error{economysim.ErrInvalidID}, paths: [][]string{{"modules", "transport", "mode", "Bad ID", "id"}},
			mutate: func(c *economysim.Config) {
				m := footMode()
				m.ID = "Bad ID"
				c.Modules.Transport = &economysim.Transport{Modes: []economysim.Mode{m}}
			}},
		{field: "Mode.ID", name: "twice", rules: []error{economysim.ErrDuplicateID}, paths: [][]string{{"modules", "transport", "mode", "foot", "id"}},
			mutate: func(c *economysim.Config) {
				c.Modules.Transport = &economysim.Transport{Modes: []economysim.Mode{footMode(), footMode()}}
			}},
		self("Mode.Speed", "below 1", bounds, func(m *economysim.Mode) { m.Speed = 0 }, "speed"),
		self("Mode.Speed", "above MaxSpeed", bounds, func(m *economysim.Mode) { m.Speed = economysim.MaxSpeed + 1 }, "speed"),
		self("Mode.UnitMass", "below 0", bounds, func(m *economysim.Mode) { m.UnitMass = -1 }, "unit_mass"),
		self("Mode.UnitMass", "above MaxCapacity", bounds, func(m *economysim.Mode) { m.UnitMass = economysim.MaxCapacity + 1 }, "unit_mass"),
		self("Mode.UnitMass", "both capacities 0", bounds, func(m *economysim.Mode) { m.UnitMass, m.UnitBulk = 0, 0 }, "unit_mass"),
		self("Mode.UnitBulk", "below 0", bounds, func(m *economysim.Mode) { m.UnitBulk = -1 }, "unit_bulk"),
		self("Mode.UnitBulk", "above MaxCapacity", bounds, func(m *economysim.Mode) { m.UnitBulk = economysim.MaxCapacity + 1 }, "unit_bulk"),
		self("Mode.Crew", "0 in a self-carried mode", bounds, func(m *economysim.Mode) { m.Crew = 0 }, "crew"),
		self("Mode.Crew", "above MaxWeight", bounds, func(m *economysim.Mode) { m.Crew = economysim.MaxWeight + 1 }, "crew"),
		hired("Mode.Crew", "set in a hired mode", economysim.ErrModeShape, func(m *economysim.Mode) { m.Crew = 1 }, "crew"),
		vehicle("Mode.Vehicle", "unknown", economysim.ErrUnknownCommodity, func(m *economysim.Mode) { m.Vehicle = "nothing" }, "vehicle"),
		vehicle("Mode.Vehicle", "not durable", economysim.ErrVehicleNotDurable, func(m *economysim.Mode) { m.Vehicle = "staple" }, "vehicle"),
		hired("Mode.Vehicle", "set in a hired mode", economysim.ErrModeShape, func(m *economysim.Mode) { m.Vehicle, m.WearPerTick = "tool_a", 0 }, "vehicle"),
		vehicle("Mode.WearPerTick", "above the durability", bounds, func(m *economysim.Mode) { m.WearPerTick = 11 }, "wear_per_tick"),
		vehicle("Mode.WearPerTick", "0 with a vehicle", economysim.ErrModeShape, func(m *economysim.Mode) { m.WearPerTick = 0 }, "wear_per_tick"),
		self("Mode.WearPerTick", "set without a vehicle", economysim.ErrModeShape, func(m *economysim.Mode) { m.WearPerTick = 1 }, "wear_per_tick"),
		vehicle("Mode.WearPerTick", "below 0", bounds, func(m *economysim.Mode) { m.WearPerTick = -1 }, "wear_per_tick"),
		self("Mode.Feed", "more than MaxFeed", bounds, func(m *economysim.Mode) { m.Feed = five }, "feed"),
		self("Mode.Feed", "unknown commodity", economysim.ErrUnknownCommodity, func(m *economysim.Mode) {
			m.Feed = []economysim.Item{{Commodity: "nothing", Quantity: 1}}
		}, "feed", "0", "commodity"),
		self("Mode.Feed", "commodity twice", economysim.ErrDuplicateID, func(m *economysim.Mode) {
			m.Feed = []economysim.Item{{Commodity: "fuel", Quantity: 1}, {Commodity: "fuel", Quantity: 2}}
		}, "feed", "1", "commodity"),
		self("Mode.Feed", "quantity 0", bounds, func(m *economysim.Mode) { m.Feed = []economysim.Item{{Commodity: "fuel"}} }, "feed", "0", "quantity"),
		self("Mode.Feed", "quantity above MaxQuantity", bounds, func(m *economysim.Mode) {
			m.Feed = []economysim.Item{{Commodity: "fuel", Quantity: economysim.MaxQuantity + 1}}
		}, "feed", "0", "quantity"),
		hired("Mode.Feed", "set in a hired mode", economysim.ErrModeShape, func(m *economysim.Mode) {
			m.Feed = []economysim.Item{{Commodity: "fuel", Quantity: 1}}
		}, "feed"),
		hired("Mode.Haul", "unknown commodity", economysim.ErrUnknownCommodity, func(m *economysim.Mode) { m.Haul = "nothing" }, "haul"),
		hired("Mode.HaulSpan", "0 with a haul commodity", economysim.ErrModeShape, func(m *economysim.Mode) { m.HaulSpan = 0 }, "haul_span"),
		hired("Mode.HaulSpan", "above MaxHaulSpan", bounds, func(m *economysim.Mode) { m.HaulSpan = economysim.MaxHaulSpan + 1 }, "haul_span"),
		hired("Mode.HaulSpan", "below 0", bounds, func(m *economysim.Mode) { m.HaulSpan = -1 }, "haul_span"),
		self("Mode.HaulSpan", "set without a haul commodity", economysim.ErrModeShape, func(m *economysim.Mode) { m.HaulSpan = 10 }, "haul_span"),
	}
}

// TestValidate_TransportRules verifies each row of spec-transport.md 12.1 on a
// configuration that breaks only that rule: the sentinel is returned, at the path
// of the field, and no other error is reported; and that a valid self-carried,
// hired and vehicle mode validate.
//
// Requirement: FR-TRN-001, FR-TRN-002, FR-TRN-003, FR-CFG-070
func TestValidate_TransportRules(t *testing.T) {
	t.Parallel()
	valid := transportConfig()
	valid.Modules.Transport.Modes = append(valid.Modules.Transport.Modes, hiredMode(), economysim.Mode{
		ID: "wagon", Speed: 30, UnitMass: 1_200, UnitBulk: 900, Crew: 2, Vehicle: "tool_a", WearPerTick: 5,
		Feed: []economysim.Item{{Commodity: "fuel", Quantity: 3}},
	})
	if err := valid.Validate(); err != nil {
		t.Fatalf("Validate(foot, coach and wagon) = %v, want nil", err)
	}
	for _, tt := range transportFieldCases() {
		t.Run(tt.field+" "+tt.name, func(t *testing.T) {
			t.Parallel()
			c := transportConfig()
			tt.mutate(&c)
			err := c.Validate()
			if !errors.Is(err, tt.rules[0]) {
				t.Fatalf("Validate() = %v, want %v", err, tt.rules[0])
			}
			vs, other := violations(err)
			if len(other) != 0 || len(vs) != 1 || !slices.Equal(vs[0].Path, tt.paths[0]) {
				t.Errorf("Validate() = %v, want only %v at %v", err, tt.rules[0], tt.paths[0])
			}
		})
	}
}

// TestValidate_TransportOffAndOverflow verifies that a module that is off checks
// nothing, that MaxWeight times MaxCapacity fits int64 (the ErrOverflow row of
// spec-transport.md 12.1 is true at the bounds, so no check can fail), that the
// modes may be given in any order and that a label past its bound is refused.
//
// Requirement: FR-TRN-001, FR-TRN-003
func TestValidate_TransportOffAndOverflow(t *testing.T) {
	t.Parallel()
	if economysim.MaxWeight*int64(economysim.MaxCapacity) <= 0 || int64(economysim.MaxSpeed)*economysim.MaxTransitFactorBP <= 0 {
		t.Errorf("MaxWeight*MaxCapacity or MaxSpeed*MaxTransitFactorBP overflows int64")
	}
	off := transportConfig()
	off.Modules.Transport = nil
	if err := off.Validate(); err != nil {
		t.Errorf("Validate(module off) = %v, want nil", err)
	}
	c := transportConfig()
	c.Modules.Transport.Modes = []economysim.Mode{hiredMode(), footMode()}
	if err := c.Validate(); err != nil {
		t.Errorf("Validate(coach before foot) = %v, want nil: the order of the list is free", err)
	}
	c.Modules.Transport.Modes[1].Label = string(make([]byte, economysim.MaxLabelBytes+1))
	if err := c.Validate(); !errors.Is(err, economysim.ErrOutOfBounds) {
		t.Errorf("Validate(label of %d bytes) = %v, want ErrOutOfBounds", economysim.MaxLabelBytes+1, err)
	}
}

// haulOf adds a commodity that no recipe makes, with the given shelf life and
// durability, and makes the module's only mode a hired one that hauls it. The
// result may not validate (nobody makes or uses the commodity); Warnings reads it
// all the same.
func haulOf(c *economysim.Config, id string, shelfLife, durability int64) {
	h := economysim.DefaultCommodity()
	h.ID, h.ReferencePrice, h.ShelfLife, h.Durability = id, 100, shelfLife, durability
	c.Commodities = append(c.Commodities, h)
	m := hiredMode()
	m.Haul = id
	c.Modules.Transport.Modes = []economysim.Mode{m}
}

// warnPaths returns the paths, joined by "/", of the warnings of a rule.
func warnPaths(c economysim.Config, rule string) []string {
	var out []string
	for _, w := range c.Warnings() {
		if w.Rule == rule {
			p := ""
			for i, s := range w.Path {
				if i > 0 {
					p += "/"
				}
				p += s
			}
			out = append(out, p)
		}
	}
	return out
}

// TestWarnings_Transport verifies each of the seven warnings of spec-transport.md
// 12.2 on its condition, at its path, and that it is quiet on the case beside it.
// The trip is 40 units over a graph of two regions, so a speed of 20 takes 2
// ticks and a speed of 5 takes 8.
//
// Requirement: FR-TRN-038
func TestWarnings_Transport(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		mutate func(c *economysim.Config)
		rule   string
		want   []string
	}{
		{"one region", func(c *economysim.Config) {
			*c = referenceConfig()
			c.Modules.Transport = &economysim.Transport{Modes: []economysim.Mode{footMode()}}
		},
			"transport_without_routes", []string{"modules/transport"}},
		{"no modes", func(c *economysim.Config) { c.Modules.Transport.Modes = nil }, "transport_without_routes", []string{"modules/transport"}},
		{"two regions and a mode", func(c *economysim.Config) {}, "transport_without_routes", nil},
		{"haul that nobody makes", func(c *economysim.Config) { haulOf(c, "haul_unit", 0, 0) },
			"haul_unsourced", []string{"modules/transport/mode/coach/haul"}},
		{"haul that is raw", func(c *economysim.Config) {
			m := hiredMode()
			m.Haul = "fuel"
			c.Modules.Transport.Modes = []economysim.Mode{m}
		}, "haul_unsourced", nil},
		{"haul never spoils", func(c *economysim.Config) { haulOf(c, "haul_unit", 0, 0) },
			"haul_never_spoils", []string{"modules/transport/mode/coach/haul"}},
		{"haul that spoils", func(c *economysim.Config) {
			m := hiredMode()
			m.Haul = "staple"
			c.Modules.Transport.Modes = []economysim.Mode{m}
		}, "haul_never_spoils", nil},
		{"haul that wears", func(c *economysim.Config) { haulOf(c, "haul_unit", 0, 3) }, "haul_durable", []string{"modules/transport/mode/coach/haul"}},
		{"haul that does not wear", func(c *economysim.Config) { haulOf(c, "haul_unit", 0, 0) }, "haul_durable", nil},
		// Wear 6 a tick over the longest trip of 2 ticks is 12, above the vehicle's durability of 10.
		{"vehicle wears out in one trip", func(c *economysim.Config) {
			c.Modules.Transport.Modes = []economysim.Mode{{ID: "wagon", Speed: 20, UnitMass: 40, UnitBulk: 30, Crew: 1, Vehicle: "tool_a", WearPerTick: 6}}
		}, "vehicle_wears_in_one_trip", []string{"modules/transport/mode/wagon/wear_per_tick"}},
		{"vehicle wears 5 in 2 ticks", func(c *economysim.Config) {
			c.Modules.Transport.Modes = []economysim.Mode{{ID: "wagon", Speed: 20, UnitMass: 40, UnitBulk: 30, Crew: 1, Vehicle: "tool_a", WearPerTick: 5}}
		}, "vehicle_wears_in_one_trip", nil},
		// Speed 1 over 40 is 40 ticks, above a window of 30; a hired mode is exempt.
		{"trip longer than the window", func(c *economysim.Config) { c.Modules.Transport.Modes[0].Speed = 1 },
			"trip_longer_than_window", []string{"modules/transport/mode/foot"}},
		{"trip within the window", func(c *economysim.Config) {}, "trip_longer_than_window", nil},
		{"hired trip is not a gap in income", func(c *economysim.Config) {
			m := hiredMode()
			m.Speed = 1
			c.Modules.Transport.Modes = []economysim.Mode{m}
		}, "trip_longer_than_window", nil},
		// Speed 5 over 40 is 8 ticks; prepared lasts 4, delivered iff k <= 3, so no route can carry it.
		{"route slower than the shelf life", func(c *economysim.Config) { c.Modules.Transport.Modes[0].Speed = 5 },
			"route_too_slow_for_shelf_life", []string{"modules/transport/mode/foot/shelf_life/prepared"}},
		// Speed 10 over 40 is 4 ticks, still no delivery for a shelf life of 4 (k <= 3 is needed); speed 20 gives 2.
		{"route within the shelf life", func(c *economysim.Config) {}, "route_too_slow_for_shelf_life", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := transportConfig()
			tt.mutate(&c)
			got := warnPaths(c, tt.rule)
			if !slices.Equal(got, tt.want) {
				t.Errorf("Warnings() paths of %s = %v, want %v", tt.rule, got, tt.want)
			}
		})
	}
	c := transportConfig()
	c.Modules.Transport.Modes[0].Speed = 5
	if err := c.Validate(); err != nil {
		t.Errorf("Validate(a config that draws a warning) = %v, want nil: a warning never fails validation", err)
	}
}

// TestPacks_TransportOverlaysLoad verifies that each transport overlay loads over
// its variant and its trade overlay, validates, switches only the transport
// module on, defines a self-carried and a hired mode, and gives another
// configuration hash than the trade pack alone.
//
// Requirement: FR-TRN-001, FR-TRN-002, FR-CFG-070
func TestPacks_TransportOverlaysLoad(t *testing.T) {
	t.Parallel()
	for _, pack := range []string{"two", "three", "five"} {
		t.Run(pack, func(t *testing.T) {
			t.Parallel()
			trade := loadPack(t, withTrade(pack)...)
			c := loadPack(t, append(withTrade(pack), "overlay/transport-"+pack+".toml")...)
			if err := c.Validate(); err != nil {
				t.Fatalf("Validate() = %v, want nil", err)
			}
			m := c.Modules.Transport
			if m == nil || len(m.Modes) != 2 || m.Modes[0].ID != "coach" || m.Modes[0].Haul != "haul" || m.Modes[1].ID != "foot" || m.Modes[1].Crew != economysim.DefaultCrew {
				t.Fatalf("Modules.Transport = %+v, want the hired coach and the self-carried foot with crew %d", m, economysim.DefaultCrew)
			}
			on := c.Modules
			on.Transport = nil
			if on != (economysim.Modules{}) {
				t.Errorf("Modules without Transport = %+v, want every other module off", on)
			}
			if economysim.HashConfig(c) == economysim.HashConfig(trade) {
				t.Errorf("HashConfig of the transport overlay equals the trade pack's, want different")
			}
		})
	}
}

// TestOverlay_TransportOnly verifies the module-alone configuration, two regions
// with the trade overlay and the transport overlay, is viable by SolveSteadyState
// and loads into a world.
//
// Guard: the module-alone golden of Phase 7 and the scenarios of Phase 9 run on
// this pack; an overlay that stopped being viable would fail them far from the
// cause.
func TestOverlay_TransportOnly(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	c := loadPack(t, append(withTrade("two"), "overlay/transport-two.toml")...)
	if _, err := economysim.SolveSteadyState(context.Background(), c); err != nil {
		t.Fatalf("SolveSteadyState() = %v, want nil: the pack must be viable", err)
	}
	if _, err := economysim.NewWorld(c); err != nil {
		t.Fatalf("NewWorld() = %v, want nil", err)
	}
}

// TestBoundaryFlow_RowOrder verifies the order of the boundary rows the module
// adds (plan 2-5 O1, spec-world.md 5.2 row 3a): after the 1 + 7C built-in rows,
// haul_use for every commodity and then transit_use for every commodity, the two
// sinks in the order of their rows in the table of spec-ledger.md 7.2, and no row
// for transit_out or transit_in, which are internal.
//
// Requirement: FR-TRN-030, FR-LED-050
func TestBoundaryFlow_RowOrder(t *testing.T) {
	t.Parallel()
	w, err := economysim.NewWorld(transportConfig())
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	s, err := w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	n := len(referenceConfig().Commodities)
	if len(s.Boundary) != 1+7*n+2*n {
		t.Fatalf("snapshot boundary rows = %d, want %d: 1 + 7C built in and 2C for the two sinks", len(s.Boundary), 1+7*n+2*n)
	}
	for i, r := range s.Boundary[1+7*n:] {
		want := economysim.BoundaryRow{Reason: economysim.ReasonHaulUse, Kind: economysim.KindSink, Commodity: economysim.CommodityID(i % n)}
		if i >= n {
			want.Reason = economysim.ReasonTransitUse
		}
		if r != want {
			t.Errorf("module row %d = %+v, want %+v", i, r, want)
		}
	}
	if !slices.Contains(s.Modules, "transport") {
		t.Errorf("Snapshot().Modules = %v, want it to list transport", s.Modules)
	}
}

// TestLedger_NoLossFlowWithRiskOff verifies that, with the module on and no
// shipment requested, a run of ticks emits no BoundaryFlow row under a transport
// reason or under loss: loss belongs to the risk module (spec-transport.md 7.2)
// and nothing ships yet.
//
// Requirement: FR-TRN-031
func TestLedger_NoLossFlowWithRiskOff(t *testing.T) {
	t.Parallel()
	var reasons []economysim.Reason
	obs := economysim.ObserverFunc(func(e economysim.Event) {
		if f, ok := e.(economysim.BoundaryFlow); ok {
			reasons = append(reasons, f.Reason)
		}
	})
	w, err := economysim.NewWorld(transportConfig(), economysim.WithObserver(obs), economysim.WithSeed(7))
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	if _, err := w.Run(context.Background(), 20); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(reasons) == 0 {
		t.Fatalf("no BoundaryFlow row in 20 ticks: the test would pass for nothing")
	}
	for _, r := range reasons {
		switch r {
		case "loss", economysim.ReasonTransitOut, economysim.ReasonTransitIn, economysim.ReasonHaulUse, economysim.ReasonTransitUse:
			t.Errorf("BoundaryFlow under %q with no shipment and the risk module off, want none", r)
		}
	}
}
