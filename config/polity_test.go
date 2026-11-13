package config_test

import (
	"reflect"
	"strconv"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// polityDoc is the polity module of example A of spec-polity.md 12.1 (without the
// regions it names, which Validate checks and Parse does not): the jurisdictions as
// tables with parent, seat, start_money and label, and the membership as one
// table keyed by region.
const polityDoc = minimal + `
[modules.polity.jurisdiction.crown]
seat = "capital"
start_money = 50000
label = "the crown"

[modules.polity.jurisdiction.coast]
parent = "crown"

[modules.polity.jurisdiction.inland]
parent = "crown"
seat = "mill"
start_money = 5000

[modules.polity.jurisdiction.uplands]
parent = "inland"

[modules.polity.member]
capital = "crown"
farm    = "inland"
harbour = "coast"
hills   = "uplands"
mill    = "inland"
`

// politySame is polityDoc written the other way: the membership as dotted keys
// under [modules.polity.member] in another order and the jurisdictions as inline
// parts of one document.
const politySame = minimal + `
[modules.polity.member]
mill    = "inland"
hills   = "uplands"
harbour = "coast"
farm    = "inland"
capital = "crown"

[modules.polity.jurisdiction.uplands]
parent = "inland"
[modules.polity.jurisdiction.inland]
parent = "crown"
seat = "mill"
start_money = 5000
[modules.polity.jurisdiction.coast]
parent = "crown"
[modules.polity.jurisdiction.crown]
seat = "capital"
start_money = 50000
label = "the crown"
`

// TestLoad_Polity verifies that the tables of spec-polity.md 3.4.2 and 12.1 load: the
// jurisdictions with their keys, in ID order, the membership one row per region, a
// key-less [modules.polity] table switching the module on, no table leaving it off,
// an unknown key and a non-string member rejected, two spellings of one polity
// loading to one Config and one hash, and the file and the Go value hashing alike.
//
// Requirement: FR-POL-022, FR-CFG-092
func TestLoad_Polity(t *testing.T) {
	t.Parallel()
	got, err := parseStrings(polityDoc)
	if err != nil {
		t.Fatalf("Parse(polityDoc) error = %v, want nil", err)
	}
	want := &economysim.PolityConfig{
		Jurisdictions: []economysim.Jurisdiction{
			{ID: "coast", Parent: "crown"},
			{ID: "crown", Seat: "capital", StartMoney: 50_000, Label: "the crown"},
			{ID: "inland", Parent: "crown", Seat: "mill", StartMoney: 5_000},
			{ID: "uplands", Parent: "inland"},
		},
		Members: []economysim.Member{
			{Region: "capital", Jurisdiction: "crown"}, {Region: "farm", Jurisdiction: "inland"},
			{Region: "harbour", Jurisdiction: "coast"}, {Region: "hills", Jurisdiction: "uplands"},
			{Region: "mill", Jurisdiction: "inland"},
		},
	}
	if !reflect.DeepEqual(got.Modules.Polity, want) {
		t.Errorf("Parse(polityDoc).Modules.Polity = %+v, want %+v", got.Modules.Polity, want)
	}
	if same, err := parseStrings(politySame); err != nil || !reflect.DeepEqual(same, got) ||
		economysim.HashConfig(same) != economysim.HashConfig(got) {
		t.Errorf("Parse(politySame) = %+v, %v, want the same Config and hash as polityDoc", same, err)
	}
	if on, err := parseStrings(minimal + "[modules.polity]\n"); err != nil || !reflect.DeepEqual(on.Modules.Polity, &economysim.PolityConfig{}) {
		t.Errorf("Parse(empty table).Modules.Polity = %+v, %v, want the module on and empty", on.Modules.Polity, err)
	}
	if off, _ := parseStrings(minimal); off.Modules.Polity != nil {
		t.Errorf("Parse(minimal).Modules.Polity = %+v, want nil: the module is off", off.Modules.Polity)
	}
	// The file and the Go value hash alike (the labels are not hashed either way).
	goCfg := economysim.Config{World: economysim.DefaultWorldParams(), Calendar: economysim.DefaultCalendar()}
	goCfg.Modules.Polity = want
	if economysim.HashConfig(goCfg) != economysim.HashConfig(got) {
		t.Error("HashConfig of the file differs from that of the same Go value")
	}
	for name, doc := range map[string]string{
		"an unknown key":       minimal + "[modules.polity.jurisdiction.a]\nregion = \"r\"\n",
		"a table as a member":  minimal + "[modules.polity.member.r]\nx = 1\n",
		"an integer as member": minimal + "[modules.polity.member]\nr = 1\n",
		"a float start_money":  minimal + "[modules.polity.jurisdiction.a]\nstart_money = 1.5\n",
		"an unknown table key": minimal + "[modules.polity]\nlevel = 1\n",
	} {
		if _, err := parseStrings(doc); err == nil {
			t.Errorf("Parse(%s) error = nil, want it rejected", name)
		}
	}
}

// FuzzLoadPolity verifies, for random jurisdiction and member tables, that the
// loader never panics, that a rejected document returns an error, and that an
// accepted one loads again to an equal Config with an equal hash and its Validate
// does not panic.
//
// Requirement: FR-CFG-024
func FuzzLoadPolity(f *testing.F) {
	f.Add("crown", "", "capital", int64(5), "capital", "crown")
	f.Add("a", "b", "", int64(0), "r", "a")
	f.Add("x", "x", "r", int64(-1), "r", "")
	f.Fuzz(func(t *testing.T, id, parent, seat string, money int64, region, member string) {
		doc := minimal + "[modules.polity.jurisdiction." + strconv.Quote(id) + "]\nparent = " + strconv.Quote(parent) + "\nseat = " + strconv.Quote(seat) +
			"\nstart_money = " + strconv.FormatInt(money, 10) + "\n[modules.polity.member]\n" + strconv.Quote(region) + " = " + strconv.Quote(member) + "\n"
		got, err := parseStrings(doc)
		if err != nil {
			return
		}
		again, err := parseStrings(doc)
		if err != nil || !reflect.DeepEqual(got, again) || economysim.HashConfig(got) != economysim.HashConfig(again) {
			t.Fatalf("Parse(%q) is not repeatable: %v", doc, err)
		}
		_ = got.Validate()
	})
}
