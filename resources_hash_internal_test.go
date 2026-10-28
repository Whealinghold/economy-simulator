package economysim

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"reflect"
	"testing"
)

// TestHash_ResourcesFieldIds verifies the bytes of the resources module in the
// hash (spec-config.md 8.2.2): the one-region row with the module on and empty is
// `000c 0005 0000 0000` before the last terminator, the module is written under
// id 5 between the demand module's pair (1) and the estates module's (25), and a
// resource, a knot, a season and a row write the field ids of the table.
//
// The digest of the empty row was computed by an independent encoder (Python's
// hashlib over the bytes of the row of 8.2.4 with the module's pair inserted), as
// the demand module's was.
//
// Requirement: FR-RES-004, FR-CFG-073, FR-CFG-025
func TestHash_ResourcesFieldIds(t *testing.T) {
	t.Parallel()
	const (
		base     = "0100070000000000000001000100000000000000056e6f7274680000" + "0000"
		insert   = "000c" + "0005" + "0000" + "0000"
		wantHash = "43be1091a7d22d268653ea1effeaabf7768c737c467c3d168cf672b7718135af"
	)
	cfg := oneRegion()
	cfg.Modules.Resources = &Resources{}
	want := base[:len(base)-4] + insert + base[len(base)-4:]
	got := encodeConfig(cfg.canonical())
	if hex.EncodeToString(got) != want {
		t.Errorf("encodeConfig(one region, Resources on) = %x, want %s", got, want)
	}
	sum := sha256.Sum256(got)
	if hex.EncodeToString(sum[:]) != wantHash || HashConfig(cfg) != sum {
		t.Errorf("HashConfig(one region, Resources on) = %x, want %s", HashConfig(cfg), wantHash)
	}

	// A populated module: every id of the table appears, strictly ascending in
	// each struct, which the shared decoder checks.
	full := oneRegion()
	full.Calendar.Seasons = []Season{{ID: "wet", Days: 1}}
	full.Modules.Resources = &Resources{
		HorizonYears: 7,
		Resources: []Resource{{
			ID: "r", Shape: ShapeLogistic, RatePerYear: 1, RateBP: 2, Refuge: 3, Capacity: 4, InitialBP: 5, PlanShareBP: 6,
			Abundance: []AbundanceKnot{{Share: 1, Factor: 2}}, Seasons: []SeasonRegen{{Season: "wet", BP: 3}},
		}},
		Endowments: []Endowment{{Region: "north", Resource: "r", HasCapacity: true, Capacity: 1, HasInitial: true, Initial: 2, HasRegenScale: true, RegenScaleBP: 3}},
	}
	r := &idReader{b: encodeConfig(full.canonical())[1:]}
	if err := r.structOf(reflect.TypeFor[Config](), "Config"); err != nil || len(r.b) != 0 {
		t.Errorf("decode of the populated encoding = %v with %d bytes left, want nil and 0", err, len(r.b))
	}
}

// TestHash_DrawsOnlyWhileOn verifies that Recipe.Draws is part of the hash only
// while the resources module is on (spec-config.md 8.2.3): a draw with the module
// off is an invalid configuration and changes no hash, and with the module on two
// recipes that differ in a draw hash differently.
//
// Requirement: FR-RES-004, FR-CFG-073
func TestHash_DrawsOnlyWhileOn(t *testing.T) {
	t.Parallel()
	mk := func(on bool, q Quantity) Config {
		c := oneRegion()
		c.Recipes = []Recipe{{ID: "dig", Draws: []Draw{{Resource: "ore", Quantity: q}}}}
		if on {
			c.Modules.Resources = &Resources{}
		}
		return c
	}
	if HashConfig(mk(false, 1)) != HashConfig(mk(false, 2)) {
		t.Errorf("HashConfig differs by a draw with the module off, want equal: the field is not written")
	}
	if HashConfig(mk(true, 1)) == HashConfig(mk(true, 2)) {
		t.Errorf("HashConfig equal for two draws with the module on, want different")
	}
}

// TestCanonical_ResourceOrder verifies the canonical order of the module
// (spec-resources.md 3.3.3, FR-RES-003): resources by ID, knots by share, season
// entries in the calendar's order, rows by region and resource, draws by
// resource; a value its Has flag does not enable is zeroed; the receiver is not
// changed.
//
// Requirement: FR-RES-003
func TestCanonical_ResourceOrder(t *testing.T) {
	t.Parallel()
	c := oneRegion()
	c.Calendar.Seasons = []Season{{ID: "wet", Days: 1}, {ID: "dry", Days: 1}}
	c.Recipes = []Recipe{{ID: "dig", Draws: []Draw{{Resource: "b", Quantity: 1}, {Resource: "a", Quantity: 2}}}}
	c.Modules.Resources = &Resources{
		Resources: []Resource{
			{ID: "b", Abundance: []AbundanceKnot{{Share: 9, Factor: 1}, {Share: 1, Factor: 2}},
				Seasons: []SeasonRegen{{Season: "dry", BP: 1}, {Season: "zzz", BP: 2}, {Season: "wet", BP: 3}}},
			{ID: "a"},
		},
		Endowments: []Endowment{
			{Region: "south", Resource: "a"}, {Region: "north", Resource: "b", Capacity: 5, Initial: 6, RegenScaleBP: 7},
			{Region: "north", Resource: "a"},
		},
	}
	before := reflect.ValueOf(c.Modules.Resources.Resources[0]).Interface()
	got := c.canonical()
	m := got.Modules.Resources
	ids := []string{m.Resources[0].ID, m.Resources[1].ID}
	if !reflect.DeepEqual(ids, []string{"a", "b"}) {
		t.Errorf("canonical resources = %v, want [a b]", ids)
	}
	if k := m.Resources[1].Abundance; k[0].Share != 1 || k[1].Share != 9 {
		t.Errorf("canonical knots = %v, want by share", k)
	}
	var seasons []string
	for _, s := range m.Resources[1].Seasons {
		seasons = append(seasons, s.Season)
	}
	if !reflect.DeepEqual(seasons, []string{"wet", "dry", "zzz"}) {
		t.Errorf("canonical seasons = %v, want the calendar's order then the unknown one", seasons)
	}
	var rows []string
	for _, e := range m.Endowments {
		rows = append(rows, e.Region+"/"+e.Resource)
	}
	if !reflect.DeepEqual(rows, []string{"north/a", "north/b", "south/a"}) {
		t.Errorf("canonical rows = %v, want by region then resource", rows)
	}
	if e := m.Endowments[1]; e.Capacity != 0 || e.Initial != 0 || e.RegenScaleBP != 0 {
		t.Errorf("canonical row with no Has flag = %+v, want its values zeroed", e)
	}
	if d := got.Recipes[0].Draws; d[0].Resource != "a" || d[1].Resource != "b" {
		t.Errorf("canonical draws = %v, want by resource", d)
	}
	if !reflect.DeepEqual(before, reflect.ValueOf(c.Modules.Resources.Resources[0]).Interface()) || c.Modules.Resources.Resources[0].ID != "b" {
		t.Errorf("canonical changed the receiver")
	}
	if !bytes.Equal(encodeConfig(got), encodeConfig(got.canonical())) {
		t.Errorf("canonical is not idempotent in the hash")
	}
}
