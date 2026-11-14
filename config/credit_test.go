package config_test

import (
	"reflect"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// creditDoc is the file form of spec-credit.md 4.1 and 4.2: the module, a borrower
// role with its block and a lender role with its block. The roles are not valid on
// their own (no recipes): the loader reads the shape of the file, Validate the rest.
const creditDoc = minimal + `
[modules.credit]
min_principal = 5000
shield_ticks = 2
write_off_ticks = 7
max_claims = 1000

[role.grower.credit.borrow]
max_rate_bp = 1500
max_debt_ticks = 90
slack_ticks = 4

[role.cook.credit.lend]
rate_bp = 600
max_term_ticks = 90
keep_ticks = 3
share_bp = 5000
`

// creditSame is creditDoc written the other way: the roles first, the blocks as
// inline tables and the keys in another order.
const creditSame = minimal + `
[role.cook.credit]
lend = { share_bp = 5000, keep_ticks = 3, max_term_ticks = 90, rate_bp = 600 }
[role.grower.credit]
borrow = { slack_ticks = 4, max_debt_ticks = 90, max_rate_bp = 1500 }
[modules.credit]
max_claims = 1000
write_off_ticks = 7
shield_ticks = 2
min_principal = 5000
`

// creditWant is the Go value creditDoc loads to.
func creditWant() economysim.Config {
	c := economysim.Config{World: economysim.DefaultWorldParams(), Calendar: economysim.DefaultCalendar()}
	c.Modules.Credit = &economysim.Credit{MinPrincipal: 5_000, ShieldTicks: 2, WriteOffTicks: 7, MaxClaims: 1_000}
	cook, grower := economysim.DefaultRole(), economysim.DefaultRole()
	cook.ID, grower.ID = "cook", "grower"
	cook.Credit = &economysim.CreditTerms{Lend: &economysim.LendTerms{RateBP: 600, MaxTermTicks: 90, KeepTicks: 3, ShareBP: 5_000}}
	grower.Credit = &economysim.CreditTerms{Borrow: &economysim.BorrowTerms{MaxRateBP: 1_500, MaxDebtTicks: 90, SlackTicks: 4}}
	c.Roles = []economysim.Role{cook, grower}
	return c
}

// TestLoad_Credit verifies that the tables of spec-credit.md 4.1 and 4.2 load: the
// module with its four keys, a Borrow and a Lend block on two roles, two spellings of
// one configuration loading to one Config and one hash, and the file and the Go value
// hashing alike.
//
// Requirement: FR-CRD-011, FR-CRD-012, FR-CFG-088
func TestLoad_Credit(t *testing.T) {
	t.Parallel()
	got, err := parseStrings(creditDoc)
	if err != nil {
		t.Fatalf("Parse(creditDoc) error = %v, want nil", err)
	}
	want := creditWant()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Parse(creditDoc) = %+v, want %+v", got, want)
	}
	if same, err := parseStrings(creditSame); err != nil || !reflect.DeepEqual(same, got) ||
		economysim.HashConfig(same) != economysim.HashConfig(got) {
		t.Errorf("Parse(creditSame) = %+v, %v, want the same Config and hash as creditDoc", same, err)
	}
	if economysim.HashConfig(want) != economysim.HashConfig(got) {
		t.Error("HashConfig of the file differs from that of the same Go value")
	}
}

// TestLoad_CreditOnAndOff verifies that a key-less [modules.credit] table switches
// the module on with every field 0 (Validate then refuses MinPrincipal and
// MaxClaims), that no table leaves it off, and that a rate of 0 loads and is a
// rate, not a missing one.
//
// Requirement: FR-CRD-011, FR-CFG-088
func TestLoad_CreditOnAndOff(t *testing.T) {
	t.Parallel()
	if on, err := parseStrings(minimal + "[modules.credit]\n"); err != nil || !reflect.DeepEqual(on.Modules.Credit, &economysim.Credit{}) {
		t.Errorf("Parse(empty table).Modules.Credit = %+v, %v, want the module on and empty", on.Modules.Credit, err)
	}
	if off, _ := parseStrings(minimal); off.Modules.Credit != nil {
		t.Errorf("Parse(minimal).Modules.Credit = %+v, want nil: the module is off", off.Modules.Credit)
	}
	zero, err := parseStrings(minimal + "[role.a.credit.lend]\nrate_bp = 0\n[role.b.credit.borrow]\nmax_rate_bp = 0\n")
	if err != nil {
		t.Fatalf("Parse(rates of 0) error = %v, want nil: 0 is interest-free", err)
	}
	if l := zero.Roles[0].Credit.Lend; l == nil || l.RateBP != 0 {
		t.Errorf("Parse(rate_bp = 0).Lend = %+v, want a block with the rate 0", l)
	}
	if b := zero.Roles[1].Credit.Borrow; b == nil || b.MaxRateBP != 0 {
		t.Errorf("Parse(max_rate_bp = 0).Borrow = %+v, want a block with the limit 0", b)
	}
}

// TestLoad_CreditRejections verifies the keys and types the loader rejects in the
// credit tables, each with the rule and the path of the key, and ErrMissingField for
// a Lend block with no rate_bp and a Borrow block with no max_rate_bp.
//
// Requirement: FR-CRD-011, FR-CFG-088
func TestLoad_CreditRejections(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		doc  string
		rule error
		path []string
	}{
		{"an unknown key of the module", minimal + "[modules.credit]\nrate = 1\n", economysim.ErrUnknownKey, []string{"modules", "credit", "rate"}},
		{"an unknown key of a lend block", minimal + "[role.a.credit.lend]\nrate_bp = 1\nterm = 1\n", economysim.ErrUnknownKey, []string{"role", "a", "credit", "lend", "term"}},
		{"an unknown key of a borrow block", minimal + "[role.a.credit.borrow]\nmax_rate_bp = 1\nlimit = 1\n", economysim.ErrUnknownKey, []string{"role", "a", "credit", "borrow", "limit"}},
		{"an unknown sub-table of the block", minimal + "[role.a.credit.guarantee]\n", economysim.ErrUnknownKey, []string{"role", "a", "credit", "guarantee"}},
		{"a float principal", minimal + "[modules.credit]\nmin_principal = 1.5\n", economysim.ErrNotInteger, []string{"modules", "credit", "min_principal"}},
		{"a string share", minimal + "[role.a.credit.lend]\nrate_bp = 1\nshare_bp = \"1\"\n", economysim.ErrWrongType, []string{"role", "a", "credit", "lend", "share_bp"}},
		{"a lend block with no rate", minimal + "[role.a.credit.lend]\nmax_term_ticks = 5\n", economysim.ErrMissingField, []string{"role", "a", "credit", "lend", "rate_bp"}},
		{"a borrow block with no limit", minimal + "[role.a.credit.borrow]\nmax_debt_ticks = 5\n", economysim.ErrMissingField, []string{"role", "a", "credit", "borrow", "max_rate_bp"}},
		{"an integer where the block table", minimal + "[role.a]\ncredit = 1\n", economysim.ErrWrongType, []string{"role", "a", "credit"}},
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

// TestLoad_CreditShape verifies that a role table with both blocks, or with a credit
// table that has neither, loads as written: the loader checks the shape of the file
// and Validate the rule of 4.2.1 (ErrCreditShape).
//
// Requirement: FR-CRD-010, FR-CRD-011
func TestLoad_CreditShape(t *testing.T) {
	t.Parallel()
	both, err := parseStrings(minimal + "[role.a.credit.lend]\nrate_bp = 1\n[role.a.credit.borrow]\nmax_rate_bp = 1\n")
	if err != nil || both.Roles[0].Credit == nil || both.Roles[0].Credit.Lend == nil || both.Roles[0].Credit.Borrow == nil {
		t.Errorf("Parse(both blocks) = %+v, %v, want a block with Lend and Borrow", both.Roles, err)
	}
	neither, err := parseStrings(minimal + "[role.a.credit]\n")
	if err != nil || !reflect.DeepEqual(neither.Roles[0].Credit, &economysim.CreditTerms{}) {
		t.Errorf("Parse(empty credit table) = %+v, %v, want an empty non-nil block", neither.Roles, err)
	}
}
