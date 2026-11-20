package config_test

import (
	"reflect"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// insuranceDoc is the file form of spec-insurance.md 4.1 and 4.2: the module, an
// insured role with its block and an underwriter role with its block. The roles are
// not valid on their own (no recipes): the loader reads the shape of the file,
// Validate the rest.
const insuranceDoc = minimal + `
[modules.insurance]
value_cap_bp = 20000
payout_ticks = 3
min_sum = 100
max_policies = 1000
max_cover_requests = 8

[role.grower.cover.insure]
min_loss_bp = 1000
ruin_share_bp = 3000
max_loading_bp = 15000
min_cover_bp = 2000
store_share_bp = 5000
max_store_rate_bp = 200

[role.cook.cover.underwrite]
loading_bp = 12000
store_rate_bp = 50
cover_bp = 8000
cap_bp = 5000
share_bp = 5000
`

// insuranceSame is insuranceDoc written the other way: the roles first, the blocks as
// inline tables and the keys in another order.
const insuranceSame = minimal + `
[role.cook.cover]
underwrite = { share_bp = 5000, cap_bp = 5000, cover_bp = 8000, store_rate_bp = 50, loading_bp = 12000 }
[role.grower.cover]
insure = { max_store_rate_bp = 200, store_share_bp = 5000, min_cover_bp = 2000, max_loading_bp = 15000, ruin_share_bp = 3000, min_loss_bp = 1000 }
[modules.insurance]
max_cover_requests = 8
max_policies = 1000
min_sum = 100
payout_ticks = 3
value_cap_bp = 20000
`

// insuranceWant is the Go value insuranceDoc loads to.
func insuranceWant() economysim.Config {
	c := economysim.Config{World: economysim.DefaultWorldParams(), Calendar: economysim.DefaultCalendar()}
	c.Modules.Insurance = &economysim.Insurance{ValueCapBP: 20_000, PayoutTicks: 3, MinSum: 100, MaxPolicies: 1_000, MaxCoverRequests: 8}
	cook, grower := economysim.DefaultRole(), economysim.DefaultRole()
	cook.ID, grower.ID = "cook", "grower"
	cook.Cover = &economysim.CoverTerms{Underwrite: &economysim.UnderwriteTerms{LoadingBP: 12_000, StoreRateBP: 50, CoverBP: 8_000, CapBP: 5_000, ShareBP: 5_000}}
	grower.Cover = &economysim.CoverTerms{Insure: &economysim.InsureTerms{
		MinLossBP: 1_000, RuinShareBP: 3_000, MaxLoadingBP: 15_000, MinCoverBP: 2_000, StoreShareBP: 5_000, MaxStoreRateBP: 200}}
	c.Roles = []economysim.Role{cook, grower}
	return c
}

// TestLoad_Insurance verifies that the tables of spec-insurance.md 4.1 and 4.2 load:
// the module with its five keys, an Insure and an Underwrite block on two roles, two
// spellings of one configuration loading to one Config and one hash, and the file and
// the Go value hashing alike.
//
// Requirement: FR-INS-010, FR-INS-011, FR-CFG-091
func TestLoad_Insurance(t *testing.T) {
	t.Parallel()
	got, err := parseStrings(insuranceDoc)
	if err != nil {
		t.Fatalf("Parse(insuranceDoc) error = %v, want nil", err)
	}
	want := insuranceWant()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Parse(insuranceDoc) = %+v, want %+v", got, want)
	}
	if same, err := parseStrings(insuranceSame); err != nil || !reflect.DeepEqual(same, got) ||
		economysim.HashConfig(same) != economysim.HashConfig(got) {
		t.Errorf("Parse(insuranceSame) = %+v, %v, want the same Config and hash as insuranceDoc", same, err)
	}
	if economysim.HashConfig(want) != economysim.HashConfig(got) {
		t.Error("HashConfig of the file differs from that of the same Go value")
	}
}

// TestLoad_InsuranceOnAndOff verifies that a key-less [modules.insurance] table
// switches the module on with the two defaults of spec-insurance.md 4.1.2 (a payout
// due in 1 tick and 4 requests in a plan, the three required fields 0 for Validate to
// refuse), that no table leaves it off, and that a field whose bound admits 0 loads as
// 0 when written.
//
// Requirement: FR-INS-001, FR-INS-011, FR-CFG-091
func TestLoad_InsuranceOnAndOff(t *testing.T) {
	t.Parallel()
	on, err := parseStrings(minimal + "[modules.insurance]\n")
	if want := economysim.DefaultInsurance(); err != nil || !reflect.DeepEqual(on.Modules.Insurance, &want) {
		t.Errorf("Parse(empty table).Modules.Insurance = %+v, %v, want the defaults %+v", on.Modules.Insurance, err, want)
	}
	if off, _ := parseStrings(minimal); off.Modules.Insurance != nil {
		t.Errorf("Parse(minimal).Modules.Insurance = %+v, want nil: the module is off", off.Modules.Insurance)
	}
	zero, err := parseStrings(minimal + "[role.a.cover.underwrite]\nloading_bp = 0\nstore_rate_bp = 0\n" +
		"[role.b.cover.insure]\nmin_loss_bp = 0\nruin_share_bp = 0\nmax_loading_bp = 0\nmax_store_rate_bp = 0\n")
	if err != nil {
		t.Fatalf("Parse(fields of 0) error = %v, want nil: 0 is written, not missing", err)
	}
	if u := zero.Roles[0].Cover.Underwrite; u == nil || u.LoadingBP != 0 || u.StoreRateBP != 0 {
		t.Errorf("Parse(loading_bp = 0).Underwrite = %+v, want a block with the fields 0", u)
	}
	if s := zero.Roles[1].Cover.Insure; s == nil || s.MinLossBP != 0 || s.StoreShareBP != 0 {
		t.Errorf("Parse(min_loss_bp = 0).Insure = %+v, want a block with the fields 0", s)
	}
}

// TestLoad_InsuranceRejections verifies the keys and types the loader rejects in the
// insurance tables, each with the rule and the path of the key, and ErrMissingField for
// each of the six required fields whose bound admits 0 and that a block omits.
//
// Requirement: FR-INS-011, FR-CFG-091
func TestLoad_InsuranceRejections(t *testing.T) {
	t.Parallel()
	under := "[role.a.cover.underwrite]\nloading_bp = 1\nstore_rate_bp = 1\n"
	insure := "[role.a.cover.insure]\nmin_loss_bp = 1\nruin_share_bp = 1\nmax_loading_bp = 1\nmax_store_rate_bp = 1\n"
	tests := []struct {
		name string
		doc  string
		rule error
		path []string
	}{
		{"an unknown key of the module", minimal + "[modules.insurance]\ncap = 1\n", economysim.ErrUnknownKey, []string{"modules", "insurance", "cap"}},
		{"an unknown key of an underwrite block", minimal + under + "term = 1\n", economysim.ErrUnknownKey, []string{"role", "a", "cover", "underwrite", "term"}},
		{"an unknown key of an insure block", minimal + insure + "limit = 1\n", economysim.ErrUnknownKey, []string{"role", "a", "cover", "insure", "limit"}},
		{"an unknown sub-table of the block", minimal + "[role.a.cover.reinsure]\n", economysim.ErrUnknownKey, []string{"role", "a", "cover", "reinsure"}},
		{"a float sum", minimal + "[modules.insurance]\nmin_sum = 1.5\n", economysim.ErrNotInteger, []string{"modules", "insurance", "min_sum"}},
		{"a string share", minimal + under + "share_bp = \"1\"\n", economysim.ErrWrongType, []string{"role", "a", "cover", "underwrite", "share_bp"}},
		{"an underwrite block with no loading", minimal + "[role.a.cover.underwrite]\nstore_rate_bp = 1\n", economysim.ErrMissingField, []string{"role", "a", "cover", "underwrite", "loading_bp"}},
		{"an underwrite block with no store rate", minimal + "[role.a.cover.underwrite]\nloading_bp = 1\n", economysim.ErrMissingField, []string{"role", "a", "cover", "underwrite", "store_rate_bp"}},
		{"an insure block with no min loss", minimal + "[role.a.cover.insure]\nruin_share_bp = 1\nmax_loading_bp = 1\nmax_store_rate_bp = 1\n", economysim.ErrMissingField, []string{"role", "a", "cover", "insure", "min_loss_bp"}},
		{"an insure block with no ruin share", minimal + "[role.a.cover.insure]\nmin_loss_bp = 1\nmax_loading_bp = 1\nmax_store_rate_bp = 1\n", economysim.ErrMissingField, []string{"role", "a", "cover", "insure", "ruin_share_bp"}},
		{"an insure block with no max loading", minimal + "[role.a.cover.insure]\nmin_loss_bp = 1\nruin_share_bp = 1\nmax_store_rate_bp = 1\n", economysim.ErrMissingField, []string{"role", "a", "cover", "insure", "max_loading_bp"}},
		{"an insure block with no max store rate", minimal + "[role.a.cover.insure]\nmin_loss_bp = 1\nruin_share_bp = 1\nmax_loading_bp = 1\n", economysim.ErrMissingField, []string{"role", "a", "cover", "insure", "max_store_rate_bp"}},
		{"an integer where the block table", minimal + "[role.a]\ncover = 1\n", economysim.ErrWrongType, []string{"role", "a", "cover"}},
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

// TestLoad_InsuranceShape verifies that a role table with both blocks, or with a cover
// table that has neither, loads as written: the loader checks the shape of the file
// and Validate the rule of 4.2.1 (ErrCoverShape).
//
// Requirement: FR-INS-010, FR-INS-011
func TestLoad_InsuranceShape(t *testing.T) {
	t.Parallel()
	both, err := parseStrings(minimal + "[role.a.cover.underwrite]\nloading_bp = 1\nstore_rate_bp = 1\n" +
		"[role.a.cover.insure]\nmin_loss_bp = 1\nruin_share_bp = 1\nmax_loading_bp = 1\nmax_store_rate_bp = 1\n")
	if err != nil || both.Roles[0].Cover == nil || both.Roles[0].Cover.Underwrite == nil || both.Roles[0].Cover.Insure == nil {
		t.Errorf("Parse(both blocks) = %+v, %v, want a block with Underwrite and Insure", both.Roles, err)
	}
	neither, err := parseStrings(minimal + "[role.a.cover]\n")
	if err != nil || !reflect.DeepEqual(neither.Roles[0].Cover, &economysim.CoverTerms{}) {
		t.Errorf("Parse(empty cover table) = %+v, %v, want an empty non-nil block", neither.Roles, err)
	}
}

// TestLoad_InsuranceEveryKeyRejectsAString verifies that each of the sixteen keys of the
// module and of the two blocks, written as a string, is ErrWrongType at its own path:
// no key is read by a path that skips the type check.
//
// Requirement: FR-INS-011, FR-CFG-091
func TestLoad_InsuranceEveryKeyRejectsAString(t *testing.T) {
	t.Parallel()
	tables := []struct {
		head string
		path []string
		keys []string
	}{
		{"[modules.insurance]", []string{"modules", "insurance"}, []string{"value_cap_bp", "payout_ticks", "min_sum", "max_policies", "max_cover_requests"}},
		{"[role.a.cover.underwrite]", []string{"role", "a", "cover", "underwrite"}, []string{"loading_bp", "store_rate_bp", "cover_bp", "cap_bp", "share_bp"}},
		{"[role.a.cover.insure]", []string{"role", "a", "cover", "insure"}, []string{"min_loss_bp", "ruin_share_bp", "max_loading_bp", "min_cover_bp", "store_share_bp", "max_store_rate_bp"}},
	}
	for _, tb := range tables {
		for _, key := range tb.keys {
			t.Run(key, func(t *testing.T) {
				t.Parallel()
				doc := minimal + tb.head + "\n" + key + " = \"1\"\n"
				_, err := parseStrings(doc)
				if !hasError(err, economysim.ErrWrongType, append(slices.Clone(tb.path), key)) {
					t.Errorf("Parse(%s = \"1\") errors = %s, want ErrWrongType at %q", key, describe(err), key)
				}
			})
		}
	}
}

// TestLoad_InsurancePartialTableKeepsDefaults verifies that a [modules.insurance] table
// that writes some keys keeps the two defaults of the others and leaves the three
// required fields 0 when they are not written, and that a written key replaces its
// default: the table is read over DefaultInsurance, key by key.
//
// Requirement: FR-INS-001, FR-INS-011
func TestLoad_InsurancePartialTableKeepsDefaults(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		body string
		want economysim.Insurance
	}{
		{"only the three required", "value_cap_bp = 7\nmin_sum = 8\nmax_policies = 9\n",
			economysim.Insurance{ValueCapBP: 7, PayoutTicks: 1, MinSum: 8, MaxPolicies: 9, MaxCoverRequests: 4}},
		{"only payout_ticks", "payout_ticks = 5\n", economysim.Insurance{PayoutTicks: 5, MaxCoverRequests: 4}},
		{"only max_cover_requests", "max_cover_requests = 2\n", economysim.Insurance{PayoutTicks: 1, MaxCoverRequests: 2}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := parseStrings(minimal + "[modules.insurance]\n" + tt.body)
			if err != nil || got.Modules.Insurance == nil || *got.Modules.Insurance != tt.want {
				t.Errorf("Parse(%q).Modules.Insurance = %+v, %v, want %+v", tt.body, got.Modules.Insurance, err, tt.want)
			}
		})
	}
}

// TestLoad_InsuranceBadShapeFailsValidate verifies the two halves of 4.2.1 end to end:
// the file of a role with both blocks, and the file of a role with an empty cover table,
// load, and the Config they give is refused by Validate with ErrCoverShape at the role's
// block, so the rule is the validator's and no file can reach a world with a bad block.
//
// Requirement: FR-INS-010
func TestLoad_InsuranceBadShapeFailsValidate(t *testing.T) {
	t.Parallel()
	under := "[role.a.cover.underwrite]\nloading_bp = 1\nstore_rate_bp = 1\ncover_bp = 1\ncap_bp = 1\nshare_bp = 1\n"
	insure := "[role.a.cover.insure]\nmin_loss_bp = 1\nruin_share_bp = 1\nmax_loading_bp = 1\nmax_store_rate_bp = 1\nmin_cover_bp = 1\n"
	for _, tc := range []struct{ name, doc string }{
		{"both blocks", minimal + under + insure},
		{"an empty cover table", minimal + "[role.a.cover]\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			c, err := parseStrings(tc.doc)
			if err != nil {
				t.Fatalf("Parse(%s) error = %v, want nil", tc.name, err)
			}
			if err := c.Validate(); !hasError(err, economysim.ErrCoverShape, []string{"roles", "a", "cover"}) {
				t.Errorf("Validate(%s) errors = %s, want ErrCoverShape at roles.a.cover", tc.name, describe(err))
			}
		})
	}
}
