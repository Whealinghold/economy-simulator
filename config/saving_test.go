package config_test

import (
	"reflect"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// savingDoc is the file form of spec-saving-and-investment.md 3.2.1: the module, a
// saver role with every key and two Worth rows, written out of class order, and a saver
// role with an inert block. The roles are not valid on their own (no recipes): the
// loader reads the shape of the file, Validate the rest.
const savingDoc = minimal + `
[modules.saving]
safety_ticks = 3
hurdle_spread_bp = 40000
commit_band_bp = 1000

[role.grower.saver]
precaution_share_bp = 10000
precaution_cap_ticks = 20
hurdle_bp = 20000
payback_ticks = 90
repair_below_bp = 3000

[[role.grower.saver.worth]]
class = "road"
per_tick = 12
count = 2

[[role.grower.saver.worth]]
class = "barn"
per_tick = 5
count = 1

[role.cook.saver]
`

// savingSame is savingDoc written the other way: the roles first, the keys in another
// order and the rows as inline tables.
const savingSame = minimal + `
[role.cook]
saver = {}
[role.grower.saver]
repair_below_bp = 3000
payback_ticks = 90
hurdle_bp = 20000
precaution_cap_ticks = 20
precaution_share_bp = 10000
worth = [{ class = "road", per_tick = 12, count = 2 }, { count = 1, per_tick = 5, class = "barn" }]
[modules.saving]
commit_band_bp = 1000
hurdle_spread_bp = 40000
safety_ticks = 3
`

// savingWant is the Go value savingDoc loads to: the block of the cook has the defaults.
func savingWant() economysim.Config {
	c := economysim.Config{World: economysim.DefaultWorldParams(), Calendar: economysim.DefaultCalendar()}
	c.Modules.Saving = &economysim.Saving{SafetyTicks: 3, HurdleSpreadBP: 40_000, CommitBandBP: 1_000}
	cook, grower := economysim.DefaultRole(), economysim.DefaultRole()
	cook.ID, grower.ID = "cook", "grower"
	inert := economysim.DefaultSaver()
	cook.Saver = &inert
	grower.Saver = &economysim.Saver{
		PrecautionShareBP: 10_000, PrecautionCapTicks: 20, HurdleBP: 20_000, PaybackTicks: 90, RepairBelowBP: 3_000,
		Worth: []economysim.ProjectWorth{{Class: "road", PerTick: 12, Count: 2}, {Class: "barn", PerTick: 5, Count: 1}},
	}
	c.Roles = []economysim.Role{cook, grower}
	return c
}

// TestLoad_SavingToml verifies that the tables of spec-saving-and-investment.md 3.2.1
// load: the module with its three keys, a block with every key and rows, an empty block
// with the defaults, two spellings of one configuration loading to one Config and one
// hash, and the file and the Go value hashing alike. The rows are in the order written;
// the canonical copy sorts them.
//
// Requirement: FR-CFG-096
func TestLoad_SavingToml(t *testing.T) {
	t.Parallel()
	got, err := parseStrings(savingDoc)
	if err != nil {
		t.Fatalf("Parse(savingDoc) error = %v, want nil", err)
	}
	want := savingWant()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Parse(savingDoc) = %+v, want %+v", got, want)
	}
	if same, err := parseStrings(savingSame); err != nil || !reflect.DeepEqual(same, got) ||
		economysim.HashConfig(same) != economysim.HashConfig(got) {
		t.Errorf("Parse(savingSame) = %+v, %v, want the same Config and hash as savingDoc", same, err)
	}
	if economysim.HashConfig(want) != economysim.HashConfig(got) {
		t.Error("HashConfig of the file differs from that of the same Go value")
	}
}

// TestLoad_SavingDefaults verifies that a key-less [modules.saving] table switches the
// module on with the defaults of spec-saving-and-investment.md 10.2, that a key written
// as 0 is the value 0 and not the default, that no table leaves the module off, and that
// a key-less block has the defaults of the block.
//
// Requirement: FR-CFG-096
func TestLoad_SavingDefaults(t *testing.T) {
	t.Parallel()
	d := economysim.DefaultSaving()
	if on, err := parseStrings(minimal + "[modules.saving]\n"); err != nil || !reflect.DeepEqual(on.Modules.Saving, &d) {
		t.Errorf("Parse(empty table).Modules.Saving = %+v, %v, want the module on with %+v", on.Modules.Saving, err, d)
	}
	if off, _ := parseStrings(minimal); off.Modules.Saving != nil {
		t.Errorf("Parse(minimal).Modules.Saving = %+v, want nil: the module is off", off.Modules.Saving)
	}
	zero, err := parseStrings(minimal + "[modules.saving]\nhurdle_spread_bp = 0\n[role.a.saver]\nrepair_below_bp = 0\n")
	if err != nil {
		t.Fatalf("Parse(keys of 0) error = %v, want nil", err)
	}
	if zero.Modules.Saving.HurdleSpreadBP != 0 || zero.Modules.Saving.SafetyTicks != d.SafetyTicks {
		t.Errorf("Parse(hurdle_spread_bp = 0).Modules.Saving = %+v, want a spread of 0 and the other defaults", zero.Modules.Saving)
	}
	if b := zero.Roles[0].Saver; b == nil || b.RepairBelowBP != 0 || b.HurdleBP != economysim.DefaultSaver().HurdleBP {
		t.Errorf("Parse(repair_below_bp = 0).Saver = %+v, want a gate of 0 and the other defaults", b)
	}
}

// TestLoad_SavingRejections verifies the keys and types the loader rejects in the saving
// tables, each with the rule and the path of the key, and ErrMissingField for a Worth
// row with no class, no per_tick or no count.
//
// Requirement: FR-CFG-096
func TestLoad_SavingRejections(t *testing.T) {
	t.Parallel()
	row := "[[role.a.saver.worth]]\nclass = \"road\"\nper_tick = 1\ncount = 1\n"
	tests := []struct {
		name string
		doc  string
		rule error
		path []string
	}{
		{"an unknown key of the module", minimal + "[modules.saving]\nrate = 1\n", economysim.ErrUnknownKey, []string{"modules", "saving", "rate"}},
		{"an unknown key of the block", minimal + "[role.a.saver]\nhorizon = 1\n", economysim.ErrUnknownKey, []string{"role", "a", "saver", "horizon"}},
		{"an unknown key of a row", minimal + row + "weight = 1\n", economysim.ErrUnknownKey, []string{"role", "a", "saver", "worth", "[0]", "weight"}},
		{"an unknown sub-table of the block", minimal + "[role.a.saver.goal]\n", economysim.ErrUnknownKey, []string{"role", "a", "saver", "goal"}},
		{"a float spread", minimal + "[modules.saving]\nhurdle_spread_bp = 1.5\n", economysim.ErrNotInteger, []string{"modules", "saving", "hurdle_spread_bp"}},
		{"a string hurdle", minimal + "[role.a.saver]\nhurdle_bp = \"1\"\n", economysim.ErrWrongType, []string{"role", "a", "saver", "hurdle_bp"}},
		{"a row with no class", minimal + "[[role.a.saver.worth]]\nper_tick = 1\ncount = 1\n", economysim.ErrMissingField, []string{"role", "a", "saver", "worth", "[0]", "class"}},
		{"a row with no per_tick", minimal + "[[role.a.saver.worth]]\nclass = \"road\"\ncount = 1\n", economysim.ErrMissingField, []string{"role", "a", "saver", "worth", "[0]", "per_tick"}},
		{"a row with no count", minimal + "[[role.a.saver.worth]]\nclass = \"road\"\nper_tick = 1\n", economysim.ErrMissingField, []string{"role", "a", "saver", "worth", "[0]", "count"}},
		{"an integer where the block table", minimal + "[role.a]\nsaver = 1\n", economysim.ErrWrongType, []string{"role", "a", "saver"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := parseStrings(tt.doc)
			if err == nil || !hasError(err, tt.rule, tt.path) {
				t.Errorf("Parse(%q) errors = %s, want %v at %q", tt.doc, describe(err), tt.rule, tt.path)
			}
		})
	}
}

// FuzzLoadSaving feeds the saving tables, well formed and not, to the loader and checks
// the properties every load has (loadProperties): a sentinel for a refusal, a repeatable
// load, and a Validate that is nil, a sentinel or a ValidationError.
func FuzzLoadSaving(f *testing.F) {
	f.Add([]byte(savingDoc), []byte("[modules.saving]\nbogus = 1\n"), []byte("[[role.a.saver.worth]]\nclass = 1\n"))
	f.Add([]byte(minimal), []byte("[role.a.saver]\nhurdle_bp = 9999\n"), []byte("[modules.saving]\nsafety_ticks = -1\n"))
	f.Add([]byte(savingSame), []byte("[role.a.saver]\nworth = [{ class = \"x\" }]\n"), []byte("[modules.saving]\ncommit_band_bp = 9223372036854775807\n"))
	f.Fuzz(loadProperties)
}
