package config_test

import (
	"os"
	"reflect"
	"strconv"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/config"
)

// institutionsDoc is the file form of spec-institutions.md 9: two bodies, a produce
// right and a sell right, a control and a rate limit, with every key.
const institutionsDoc = minimal + `
[modules.institutions]
own_use_bound = true

[modules.institutions.body.weavers]
roles = ["maker"]
regions = ["north"]
places = 12
max_trainees = 2
needs = "fair"
label = "Weavers"

[modules.institutions.body.league]
roles = ["cook", "grower"]
jurisdictions = ["crown"]
regions = ["south"]

[modules.institutions.right.weave_in_north]
body = "weavers"
activity = "produce"
subject = "make_tool_a"
region = "north"
who = "both"
label = "Charter"

[modules.institutions.right.no_sale]
activity = "sell"
subject = "tool_a"
jurisdiction = "crown"

[modules.institutions.control.assize]
commodity = "staple"
jurisdiction = "crown"
cap_bp = 12000
floor_bp = 8000

[modules.institutions.rate_limit.interest_law]
region = "north"
cap_bp = 1200
`

// institutionsSame is institutionsDoc written the other way: the tables in another
// order, the lists unsorted and the default Who of the produce right spelled out.
const institutionsSame = minimal + `
[modules.institutions.rate_limit.interest_law]
cap_bp = 1200
region = "north"
[modules.institutions.control.assize]
floor_bp = 8000
cap_bp = 12000
jurisdiction = "crown"
commodity = "staple"
[modules.institutions.right.no_sale]
jurisdiction = "crown"
subject = "tool_a"
activity = "sell"
who = "principal"
[modules.institutions.right.weave_in_north]
who = "both"
region = "north"
subject = "make_tool_a"
activity = "produce"
body = "weavers"
[modules.institutions.body.league]
regions = ["south"]
jurisdictions = ["crown"]
roles = ["grower", "cook"]
[modules.institutions.body.weavers]
needs = "fair"
max_trainees = 2
places = 12
regions = ["north"]
roles = ["maker"]
[modules.institutions]
own_use_bound = true
`

// TestLoad_Institutions verifies that the tables of spec-institutions.md 9 load: the
// rows in ID order with every key, the labels read, a key-less [modules.institutions]
// table switching the module on with nothing declared, no table leaving it off, two
// spellings of one module hashing alike, and the file and the Go value hashing alike.
//
// Requirement: FR-ORG-051, FR-CFG-094
func TestLoad_Institutions(t *testing.T) {
	t.Parallel()
	got, err := parseStrings(institutionsDoc)
	if err != nil {
		t.Fatalf("Parse(institutionsDoc) error = %v, want nil", err)
	}
	want := &economysim.Institutions{
		OwnUseBound: true,
		Bodies: []economysim.Body{
			{ID: "league", Roles: []string{"cook", "grower"}, Regions: []string{"south"}, Jurisdictions: []string{"crown"}},
			{ID: "weavers", Roles: []string{"maker"}, Regions: []string{"north"}, Places: 12, MaxTrainees: 2, Needs: "fair", Label: "Weavers"},
		},
		Rights: []economysim.Right{
			{ID: "no_sale", Activity: "sell", Subject: "tool_a", Jurisdiction: "crown"},
			{ID: "weave_in_north", Body: "weavers", Activity: "produce", Subject: "make_tool_a", Region: "north", Who: "both", Label: "Charter"},
		},
		Controls:   []economysim.Control{{ID: "assize", Commodity: "staple", Jurisdiction: "crown", CapBP: 12000, FloorBP: 8000}},
		RateLimits: []economysim.RateLimit{{ID: "interest_law", Region: "north", CapBP: 1200}},
	}
	if !reflect.DeepEqual(got.Modules.Institutions, want) {
		t.Errorf("Parse(institutionsDoc).Modules.Institutions = %+v, want %+v", got.Modules.Institutions, want)
	}
	if same, err := parseStrings(institutionsSame); err != nil || economysim.HashConfig(same) != economysim.HashConfig(got) {
		t.Errorf("Parse(institutionsSame) = %+v, %v, want the same hash as institutionsDoc: the lists hash in canonical order", same.Modules.Institutions, err)
	}
	goCfg := economysim.Config{World: economysim.DefaultWorldParams(), Calendar: economysim.DefaultCalendar()}
	goCfg.Modules.Institutions = want
	if economysim.HashConfig(goCfg) != economysim.HashConfig(got) {
		t.Error("HashConfig of the file differs from that of the same Go value")
	}
	if on, err := parseStrings(minimal + "[modules.institutions]\n"); err != nil || !reflect.DeepEqual(on.Modules.Institutions, &economysim.Institutions{}) {
		t.Errorf("Parse(empty table).Modules.Institutions = %+v, %v, want the module on and empty", on.Modules.Institutions, err)
	}
	if off, _ := parseStrings(minimal); off.Modules.Institutions != nil {
		t.Errorf("Parse(minimal).Modules.Institutions = %+v, want nil: the module is off", off.Modules.Institutions)
	}
}

// TestLoad_InstitutionsRejections verifies the keys and types the loader rejects in the
// institutions tables, each with the rule and the path of the key.
//
// Requirement: FR-CFG-094
func TestLoad_InstitutionsRejections(t *testing.T) {
	t.Parallel()
	base := []string{"modules", "institutions"}
	at := func(more ...string) []string { return append(append([]string{}, base...), more...) }
	tests := []struct {
		name string
		doc  string
		rule error
		path []string
	}{
		{"an unknown key of the module", minimal + "[modules.institutions]\nlevel = 1\n", economysim.ErrUnknownKey, at("level")},
		{"an unknown key of a body", minimal + "[modules.institutions.body.a]\nseats = 1\n", economysim.ErrUnknownKey, at("body", "a", "seats")},
		{"a float of places", minimal + "[modules.institutions.body.a]\nplaces = 1.5\n", economysim.ErrNotInteger, at("body", "a", "places")},
		{"a string of roles", minimal + "[modules.institutions.body.a]\nroles = \"maker\"\n", economysim.ErrWrongType, at("body", "a", "roles")},
		{"a right with no activity", minimal + "[modules.institutions.right.a]\nsubject = \"x\"\n", economysim.ErrMissingField, at("right", "a", "activity")},
		{"a right with no subject", minimal + "[modules.institutions.right.a]\nactivity = \"sell\"\n", economysim.ErrMissingField, at("right", "a", "subject")},
		{"a control with no commodity", minimal + "[modules.institutions.control.a]\ncap_bp = 1\n", economysim.ErrMissingField, at("control", "a", "commodity")},
		{"a rate limit with no cap", minimal + "[modules.institutions.rate_limit.a]\nregion = \"north\"\n", economysim.ErrMissingField, at("rate_limit", "a", "cap_bp")},
		{"a rate limit with a floor", minimal + "[modules.institutions.rate_limit.a]\ncap_bp = 1\nfloor_bp = 1\n", economysim.ErrUnknownKey, at("rate_limit", "a", "floor_bp")},
		{"a scalar where a body table", minimal + "[modules.institutions.body]\na = 1\n", economysim.ErrWrongType, at("body", "a")},
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

// TestLoad_InstitutionsOverBounds verifies that a document that loads and breaks a rule
// of spec-institutions.md 10.1 is reported by Validate, not by the loader: the loader
// checks the shape of the file and the rules are the validator's.
//
// Requirement: FR-ORG-047
func TestLoad_InstitutionsOverBounds(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		doc  string
		rule error
		path []string
	}{
		{"places over MaxHouseholds", "[modules.institutions.body.a]\nroles = [\"maker\"]\nregions = [\"north\"]\nplaces = 1000000001\n",
			economysim.ErrOutOfBounds, []string{"modules", "institutions", "bodies[0]", "places"}},
		{"a body with no role", "[modules.institutions.body.a]\nregions = [\"north\"]\n",
			economysim.ErrBodyShape, []string{"modules", "institutions", "bodies[0]", "roles"}},
		{"a jurisdiction with the polity module off", "[modules.institutions.body.a]\nroles = [\"maker\"]\njurisdictions = [\"crown\"]\n",
			economysim.ErrModuleDependency, []string{"modules", "institutions", "bodies[0]", "jurisdictions"}},
		{"a right of an unknown body", "[modules.institutions.right.a]\nbody = \"nowhere\"\nactivity = \"sell\"\nsubject = \"tool_a\"\nregion = \"north\"\n",
			economysim.ErrUnknownBody, []string{"modules", "institutions", "rights[0]", "body"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			doc, err := parseStrings(minimal, tt.doc)
			if err != nil {
				t.Fatalf("Parse() error = %v, want nil: the loader does not check the rules", err)
			}
			cfg, err := config.LoadFS(os.DirFS(chains), "base.toml", "three.toml", "warm-three.toml")
			if err != nil {
				t.Fatalf("LoadFS(pack three) error = %v, want nil", err)
			}
			cfg.Modules.Institutions = doc.Modules.Institutions
			if err := cfg.Validate(); err == nil || !hasError(err, tt.rule, tt.path) {
				t.Errorf("Validate() errors = %s, want %v at %q", describe(err), tt.rule, tt.path)
			}
		})
	}
}

// FuzzLoadInstitutions verifies, for random tables of a body, a right and a control,
// that the loader never panics, that a rejected document returns an error, that an
// accepted one loads again to an equal Config with an equal hash, and that Validate
// does not panic and accepts only a body with a role and a shape that 10.1 allows.
//
// Requirement: FR-ORG-047, FR-CFG-024
func FuzzLoadInstitutions(f *testing.F) {
	f.Add("weavers", "north", int64(12), "produce", "make_tool_a", int64(12000), int64(8000))
	f.Add("", "", int64(-1), "ship", "", int64(-1), int64(-1))
	f.Add("x", "nowhere", int64(1)<<62, "sell", "tool_a", int64(1)<<62, int64(1)<<62)
	f.Fuzz(func(t *testing.T, id, region string, places int64, activity, subject string, capBP, floorBP int64) {
		doc := minimal + "[modules.institutions.body." + strconv.Quote(id) + "]\nroles = [ \"maker\" ]\nregions = [ " +
			strconv.Quote(region) + " ]\nplaces = " + strconv.FormatInt(places, 10) +
			"\n[modules.institutions.right.r]\nactivity = " + strconv.Quote(activity) + "\nsubject = " + strconv.Quote(subject) +
			"\nregion = " + strconv.Quote(region) +
			"\n[modules.institutions.control.c]\ncommodity = \"staple\"\nregion = \"north\"\ncap_bp = " + strconv.FormatInt(capBP, 10) +
			"\nfloor_bp = " + strconv.FormatInt(floorBP, 10) + "\n"
		got, err := parseStrings(doc)
		if err != nil {
			return
		}
		again, err := parseStrings(doc)
		if err != nil || !reflect.DeepEqual(got, again) || economysim.HashConfig(got) != economysim.HashConfig(again) {
			t.Fatalf("Parse(%q) is not repeatable: %v", doc, err)
		}
		if got.Validate() != nil || got.Modules.Institutions == nil {
			return
		}
		if b, bad := institutionsBadBody(got.Modules.Institutions); bad {
			t.Fatalf("Validate accepted the body %+v of %q, want a role and places in bounds", b, doc)
		}
	})
}

// institutionsBadBody returns the first body of a validated module that has no role or
// has places outside 0 to MaxHouseholds, which Validate must have refused.
func institutionsBadBody(m *economysim.Institutions) (economysim.Body, bool) {
	for _, b := range m.Bodies {
		if len(b.Roles) == 0 || b.Places < 0 || b.Places > economysim.MaxHouseholds {
			return b, true
		}
	}
	return economysim.Body{}, false
}
