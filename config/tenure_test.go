package config_test

import (
	"os"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/config"
)

// tenureDoc is the file form of spec-land.md 3.3.2: a term with a rent and a closing,
// a term with a closing only, a holding of a role and a holding of the pool, and the
// five tunables of the module.
const tenureDoc = minimal + `
[modules.tenure]
rent_step_bp = 500
rent_floor_bp = 2000
rent_ceiling_bp = 15000
own_use_exempt = true
max_holder_share_bp = 5000

[modules.tenure.term.vale.plots]
rent = 40
closed = true

[modules.tenure.term.vale.stand]
closed = false

[modules.tenure.holding.vale.plots.freeholder]
share_bp = 500

[modules.tenure.holding.vale.plots.pool]
share_bp = 1000
`

// tenureSame is tenureDoc written the other way: the tables in another order and the
// keys in another order.
const tenureSame = minimal + `
[modules.tenure.holding.vale.plots.pool]
share_bp = 1000
[modules.tenure.holding.vale.plots.freeholder]
share_bp = 500
[modules.tenure.term.vale.stand]
closed = false
[modules.tenure.term.vale.plots]
closed = true
rent = 40
[modules.tenure]
max_holder_share_bp = 5000
own_use_exempt = true
rent_ceiling_bp = 15000
rent_floor_bp = 2000
rent_step_bp = 500
`

// TestLoad_Tenure verifies that the tables of spec-land.md 3.3.2 load: the five tunables,
// a term with each key setting its Has flag and a key absent leaving it clear, a holding
// of a role and of the pool with the holder taken from the last key, a key-less
// [modules.tenure] table switching the module on with nothing in it, no table leaving it
// off, two spellings of one module hashing alike, and the file and the Go value hashing
// alike.
//
// Requirement: FR-LND-001, FR-LND-038, FR-CFG-085
func TestLoad_Tenure(t *testing.T) {
	t.Parallel()
	got, err := parseStrings(tenureDoc)
	if err != nil {
		t.Fatalf("Parse(tenureDoc) error = %v, want nil", err)
	}
	want := &economysim.TenureConfig{
		Terms: []economysim.TenureTerm{
			{Region: "vale", Resource: "plots", HasRent: true, Rent: 40, HasClosed: true, Closed: true},
			{Region: "vale", Resource: "stand", HasClosed: true},
		},
		Holdings: []economysim.Holding{
			{Region: "vale", Resource: "plots", Holder: economysim.HolderRef{Kind: economysim.HolderKindRole, Role: "freeholder"}, ShareBP: 500},
			{Region: "vale", Resource: "plots", Holder: economysim.HolderRef{Kind: economysim.HolderKindPool}, ShareBP: 1000},
		},
		RentStepBP: 500, RentFloorBP: 2000, RentCeilingBP: 15000, OwnUseExempt: true, MaxHolderShareBP: 5000,
	}
	if !reflect.DeepEqual(got.Modules.Tenure, want) {
		t.Errorf("Parse(tenureDoc).Modules.Tenure = %+v, want %+v", got.Modules.Tenure, want)
	}
	if same, err := parseStrings(tenureSame); err != nil || economysim.HashConfig(same) != economysim.HashConfig(got) {
		t.Errorf("Parse(tenureSame) = %+v, %v, want the same hash as tenureDoc: the lists hash in canonical order", same.Modules.Tenure, err)
	}
	goCfg := economysim.Config{World: economysim.DefaultWorldParams(), Calendar: economysim.DefaultCalendar()}
	goCfg.Modules.Tenure = want
	if economysim.HashConfig(goCfg) != economysim.HashConfig(got) {
		t.Error("HashConfig of the file differs from that of the same Go value")
	}
	if on, err := parseStrings(minimal + "[modules.tenure]\n"); err != nil || !reflect.DeepEqual(on.Modules.Tenure, &economysim.TenureConfig{}) {
		t.Errorf("Parse(empty table).Modules.Tenure = %+v, %v, want the module on and empty", on.Modules.Tenure, err)
	}
	if off, _ := parseStrings(minimal); off.Modules.Tenure != nil {
		t.Errorf("Parse(minimal).Modules.Tenure = %+v, want nil: the module is off", off.Modules.Tenure)
	}
}

// TestLoad_TenureRejections verifies the keys and types the loader rejects in the tenure
// tables, each with the rule and the path of the key.
//
// Requirement: FR-CFG-085
func TestLoad_TenureRejections(t *testing.T) {
	t.Parallel()
	base := []string{"modules", "tenure"}
	tests := []struct {
		name string
		doc  string
		rule error
		path []string
	}{
		{"an unknown key of the module", minimal + "[modules.tenure]\nlevel = 1\n", economysim.ErrUnknownKey, append(slices.Clone(base), "level")},
		{"an unknown key of a term", minimal + "[modules.tenure.term.a.b]\ncost = 1\n", economysim.ErrUnknownKey, append(slices.Clone(base), "term", "a", "b", "cost")},
		{"an unknown key of a holding", minimal + "[modules.tenure.holding.a.b.pool]\nshare_bp = 1\nrole = \"x\"\n", economysim.ErrUnknownKey, append(slices.Clone(base), "holding", "a", "b", "pool", "role")},
		{"a holding with no share", minimal + "[modules.tenure.holding.a.b.pool]\n", economysim.ErrMissingField, append(slices.Clone(base), "holding", "a", "b", "pool", "share_bp")},
		{"a float rent", minimal + "[modules.tenure.term.a.b]\nrent = 1.5\n", economysim.ErrNotInteger, append(slices.Clone(base), "term", "a", "b", "rent")},
		{"a string share", minimal + "[modules.tenure.holding.a.b.pool]\nshare_bp = \"1\"\n", economysim.ErrWrongType, append(slices.Clone(base), "holding", "a", "b", "pool", "share_bp")},
		{"an integer closed", minimal + "[modules.tenure.term.a.b]\nclosed = 1\n", economysim.ErrWrongType, append(slices.Clone(base), "term", "a", "b", "closed")},
		{"a string exemption", minimal + "[modules.tenure]\nown_use_exempt = \"yes\"\n", economysim.ErrWrongType, append(slices.Clone(base), "own_use_exempt")},
		{"a scalar where a term table", minimal + "[modules.tenure.term]\na = 1\n", economysim.ErrWrongType, append(slices.Clone(base), "term", "a")},
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

// TestLoad_TenureToml verifies that overlay/tenure.toml loads on pack one and on pack
// five after the resources overlay, validates, holds the rows it says and raises no
// warning of the module but rent_hoarded (the packs have no estates module, so the
// pool's title has no relief to go to, spec-land.md 11.2).
//
// Requirement: FR-LND-001, FR-LND-039, FR-CFG-085
func TestLoad_TenureToml(t *testing.T) {
	t.Parallel()
	wantTerm := economysim.TenureTerm{Region: "north", Resource: "arable", HasRent: true, Rent: 2}
	// The loader reads the holders in the order of their keys, grower before pool;
	// canonical puts the pool first.
	wantHoldings := []economysim.Holding{
		{Region: "north", Resource: "arable", Holder: economysim.HolderRef{Kind: economysim.HolderKindRole, Role: "grower"}, ShareBP: 20},
		{Region: "north", Resource: "arable", Holder: economysim.HolderRef{Kind: economysim.HolderKindPool}, ShareBP: 1000},
	}
	packs := map[string][]string{
		"one":  {"base.toml", "one.toml", "warm-one.toml", "overlay/resources.toml", "overlay/tenure.toml"},
		"five": {"base.toml", "lines.toml", "five.toml", "warm-five.toml", "overlay/resources-five.toml", "overlay/tenure.toml"},
	}
	for name, layers := range packs {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			cfg := loadTechnologyPack(t, layers...)
			m := cfg.Modules.Tenure
			if m == nil || !reflect.DeepEqual(m.Terms, []economysim.TenureTerm{wantTerm}) || !reflect.DeepEqual(m.Holdings, wantHoldings) {
				t.Fatalf("overlay/tenure.toml on %s loaded %+v, want the term %+v and the holdings %+v", name, m, wantTerm, wantHoldings)
			}
			if m.RentStepBP != 0 || m.OwnUseExempt {
				t.Errorf("overlay/tenure.toml on %s loaded a step %d and an exemption %v, want the rent not to move", name, m.RentStepBP, m.OwnUseExempt)
			}
			var warned []string
			for _, w := range cfg.Warnings() {
				if len(w.Path) > 1 && w.Path[1] == "tenure" {
					warned = append(warned, strings.Join(w.Path, ".")+" "+w.Rule)
				}
			}
			if want := []string{"modules.tenure.holding[0] rent_hoarded"}; !slices.Equal(warned, want) {
				t.Errorf("Warnings() of the tenure module on %s = %q, want %q", name, warned, want)
			}
		})
	}
}

// TestLoad_TenureOverBounds verifies that a document that loads and breaks a rule of
// spec-land.md 11.1 is reported by Validate, not by the loader: the loader checks the
// shape of the file and the bounds are the validator's.
//
// Requirement: FR-LND-038
func TestLoad_TenureOverBounds(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		doc  string
		rule error
		path []string
	}{
		{"a share of 0", "[modules.tenure.holding.north.arable.pool]\nshare_bp = 0\n",
			economysim.ErrTitleShares, []string{"modules", "tenure", "holding[0]", "share_bp"}},
		{"the census over 10 000", "[modules.tenure.holding.north.arable.grower]\nshare_bp = 100\n",
			economysim.ErrTitleShares, []string{"modules", "tenure", "holding[0]", "share_bp"}},
		{"a region that is not defined", "[modules.tenure.term.nowhere.arable]\nrent = 1\n",
			economysim.ErrUnknownRegion, []string{"modules", "tenure", "term[0]", "region"}},
		{"a role that is not defined", "[modules.tenure.holding.north.arable.serf]\nshare_bp = 1\n",
			economysim.ErrUnknownRole, []string{"modules", "tenure", "holding[0]", "holder", "role"}},
		{"a rent below 0", "[modules.tenure.term.north.arable]\nrent = -1\n",
			economysim.ErrOutOfBounds, []string{"modules", "tenure", "term[0]", "rent"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			doc, err := parseStrings(minimal, tt.doc)
			if err != nil {
				t.Fatalf("Parse() error = %v, want nil: the loader does not check bounds", err)
			}
			cfg, err := config.LoadFS(os.DirFS(chains), "base.toml", "one.toml", "warm-one.toml", "overlay/resources.toml")
			if err != nil {
				t.Fatalf("LoadFS(pack one with resources) error = %v, want nil", err)
			}
			cfg.Modules.Tenure = doc.Modules.Tenure
			if err := cfg.Validate(); err == nil || !hasError(err, tt.rule, tt.path) {
				t.Errorf("Validate() errors = %s, want %v at %q", describe(err), tt.rule, tt.path)
			}
		})
	}
}

// TestLoad_TenureNeedsResources verifies that the tenure overlay without the resources
// overlay is ErrModuleDependency, the module needing the endowments it titles.
//
// Requirement: FR-LND-001, FR-LND-038
func TestLoad_TenureNeedsResources(t *testing.T) {
	t.Parallel()
	cfg, err := config.LoadFS(os.DirFS(chains), "base.toml", "one.toml", "warm-one.toml", "overlay/tenure.toml")
	if err != nil {
		t.Fatalf("LoadFS(one with tenure only) error = %v, want nil", err)
	}
	if err := cfg.Validate(); err == nil || !hasError(err, economysim.ErrModuleDependency, []string{"modules", "tenure"}) {
		t.Errorf("Validate() errors = %s, want ErrModuleDependency at modules.tenure", describe(err))
	}
}

// FuzzLoadTenure verifies, for random term and holding tables, that the loader never
// panics, that a rejected document returns an error, that an accepted one loads again to
// an equal Config with an equal hash, that Validate and Warnings do not panic, and that a
// holding that validates has a share in 1 to 10 000.
//
// Requirement: FR-LND-038, FR-CFG-024
func FuzzLoadTenure(f *testing.F) {
	f.Add("north", "arable", "grower", int64(20), int64(3), int64(500), int64(5000))
	f.Add("north", "arable", "pool", int64(1000), int64(0), int64(0), int64(0))
	f.Add("", "", "", int64(0), int64(-1), int64(-1), int64(-1))
	f.Add("x", "y", "z", int64(1)<<62, int64(1)<<62, int64(1)<<62, int64(1)<<62)
	f.Fuzz(func(t *testing.T, region, resource, holder string, share, rent, step, maxShare int64) {
		n := func(v int64) string { return strconv.FormatInt(v, 10) }
		doc := minimal + "[modules.tenure]\nrent_step_bp = " + n(step) + "\nmax_holder_share_bp = " + n(maxShare) +
			"\n[modules.tenure.term." + strconv.Quote(region) + "." + strconv.Quote(resource) + "]\nrent = " + n(rent) +
			"\n[modules.tenure.holding." + strconv.Quote(region) + "." + strconv.Quote(resource) + "." + strconv.Quote(holder) + "]\nshare_bp = " + n(share) + "\n"
		got, err := parseStrings(doc)
		if err != nil {
			return
		}
		again, err := parseStrings(doc)
		if err != nil || !reflect.DeepEqual(got, again) || economysim.HashConfig(got) != economysim.HashConfig(again) {
			t.Fatalf("Parse(%q) is not repeatable: %v", doc, err)
		}
		_ = got.Warnings()
		if got.Validate() != nil || got.Modules.Tenure == nil {
			return
		}
		for _, h := range got.Modules.Tenure.Holdings {
			if h.ShareBP < 1 || h.ShareBP > 10_000 {
				t.Fatalf("Validate accepted the holding %+v of %q, want a share of 1 to 10 000", h, doc)
			}
		}
	})
}
