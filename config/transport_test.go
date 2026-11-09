package config_test

import (
	"errors"
	"reflect"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// transportDoc is a [modules.transport] table in the form of spec-transport.md
// 3.1: a self-carried mode with no crew key, a self-carried mode with a vehicle,
// a feed and a crew, and a hired mode, written in an order that is not the
// canonical one.
const transportDoc = minimal + `
[modules.transport.mode.wagon]
speed = 30
unit_mass = 1200
unit_bulk = 900
crew = 2
vehicle = "tool"
wear_per_tick = 5
feed = [ { commodity = "fodder", quantity = 3 }, { commodity = "bale", quantity = 1 } ]
label = "A wagon"
[modules.transport.mode.foot]
speed = 20
unit_mass = 40
unit_bulk = 30
[modules.transport.mode.coach]
speed = 30
unit_mass = 1200
unit_bulk = 900
haul = "haul"
haul_span = 10
`

// TestLoad_TransportMode verifies that the keys of spec-config.md 6.2.1 load the
// transport module: [modules.transport.mode.<id>] switches it on, the modes come
// out in ID order, an absent crew gives DefaultCrew in a self-carried mode and 0
// in a hired one, feed is an array of { commodity, quantity } tables, and with no
// table the module is off.
//
// Requirement: FR-TRN-001, FR-TRN-002, FR-CFG-070
func TestLoad_TransportMode(t *testing.T) {
	t.Parallel()
	got, err := parseStrings(transportDoc)
	if err != nil {
		t.Fatalf("Parse(transportDoc) error = %v, want nil", err)
	}
	want := &economysim.Transport{Modes: []economysim.Mode{
		{ID: "coach", Speed: 30, UnitMass: 1_200, UnitBulk: 900, Haul: "haul", HaulSpan: 10},
		{ID: "foot", Speed: 20, UnitMass: 40, UnitBulk: 30, Crew: economysim.DefaultCrew},
		{ID: "wagon", Speed: 30, UnitMass: 1_200, UnitBulk: 900, Crew: 2, Vehicle: "tool", WearPerTick: 5, Label: "A wagon",
			Feed: []economysim.Item{{Commodity: "fodder", Quantity: 3}, {Commodity: "bale", Quantity: 1}}},
	}}
	if !reflect.DeepEqual(got.Modules.Transport, want) {
		t.Errorf("Parse(transportDoc).Modules.Transport = %+v, want %+v", got.Modules.Transport, want)
	}
	on, _ := parseStrings(minimal + "[modules.transport]\n")
	if on.Modules.Transport == nil || len(on.Modules.Transport.Modes) != 0 {
		t.Errorf("Parse([modules.transport]).Modules.Transport = %+v, want the module on with no mode", on.Modules.Transport)
	}
	if off, _ := parseStrings(minimal); off.Modules.Transport != nil {
		t.Errorf("Parse(minimal).Modules.Transport = %+v, want nil: the module is off", off.Modules.Transport)
	}
}

// TestLoad_TransportCrewHash verifies that an omitted crew and an explicit 1
// give the same Config and the same hash, and that a hired mode that writes a
// crew keeps it, for Validate to refuse (ErrModeShape).
//
// Requirement: FR-TRN-002, FR-TRN-037
func TestLoad_TransportCrewHash(t *testing.T) {
	t.Parallel()
	mode := "[modules.transport.mode.foot]\nspeed = 20\nunit_mass = 40\nunit_bulk = 30\n"
	a, err := parseStrings(minimal + mode)
	if err != nil {
		t.Fatalf("Parse(no crew) error = %v, want nil", err)
	}
	b, err := parseStrings(minimal + mode + "crew = 1\n")
	if err != nil {
		t.Fatalf("Parse(crew = 1) error = %v, want nil", err)
	}
	if !reflect.DeepEqual(a.Modules.Transport, b.Modules.Transport) || economysim.HashConfig(a) != economysim.HashConfig(b) {
		t.Errorf("Parse(no crew) and Parse(crew = 1) differ: %+v against %+v", a.Modules.Transport, b.Modules.Transport)
	}
	h, err := parseStrings(minimal + "[modules.transport.mode.coach]\nspeed = 30\nunit_mass = 1\nhaul = \"haul\"\nhaul_span = 10\ncrew = 2\n")
	if err != nil {
		t.Fatalf("Parse(hired with crew) error = %v, want nil", err)
	}
	if got := h.Modules.Transport.Modes[0].Crew; got != 2 {
		t.Errorf("Parse(hired with crew = 2).Crew = %d, want 2 kept for Validate to refuse", got)
	}
}

// TestLoad_TransportRejects verifies that an unknown key in the module table, in
// a mode and in a feed entry is ErrUnknownKey at its path, and that a feed entry
// with no quantity is a missing key.
//
// Requirement: FR-TRN-001, FR-CFG-024
func TestLoad_TransportRejects(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		doc  string
		want error
		path []string
	}{
		{"unknown key in the module", "[modules.transport]\nbogus = 1\n", economysim.ErrUnknownKey, []string{"modules", "transport", "bogus"}},
		{"unknown key in a mode", "[modules.transport.mode.foot]\nbogus = 1\n", economysim.ErrUnknownKey, []string{"modules", "transport", "mode", "foot", "bogus"}},
		{"unknown key in a feed entry", "[modules.transport.mode.foot]\nfeed = [ { commodity = \"a\", quantity = 1, bogus = 2 } ]\n", economysim.ErrUnknownKey, []string{"modules", "transport", "mode", "foot", "feed", "[0]", "bogus"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := parseStrings(minimal + tt.doc)
			if !errors.Is(err, tt.want) {
				t.Fatalf("Parse(%q) error = %v, want %v", tt.doc, err, tt.want)
			}
		})
	}
	if _, err := parseStrings(minimal + "[modules.transport.mode.foot]\nfeed = [ { commodity = \"a\" } ]\n"); err == nil {
		t.Errorf("Parse(feed entry with no quantity) error = nil, want a missing key")
	}
}
