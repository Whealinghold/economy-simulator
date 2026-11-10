package config_test

import (
	"errors"
	"reflect"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// merchantsDoc is a [modules.merchants] table and a merchant role in the form of
// spec-merchants.md 19.1, with a role that has no block.
const merchantsDoc = minimal + `
[modules.merchants]
max_open = 500
[role.broker]
strategy = "merchant"
informed = true
trades = [ "x", "haul" ]
merchant_fee_share = 6000
[role.broker.merchant]
modes = [ "coach", "barge" ]
lot_units = 2
fee_share_spread_bp = 1500
max_source_age = 3
time_cost_bp = 8000
consignee_cut_bp = 400
consigns = true
take_margin_bp = 900
take_units = 150
[role.farmer]
`

// TestLoad_Merchants verifies that the keys of spec-config.md 6.2.1 and
// spec-merchants.md 13 load the merchants module and the block of a role: the
// table switches the module on, its key is read, the block is read over
// DefaultMerchant, a role with a block and no share takes the default 5 000
// while a role with no block keeps 0, with no table the module is off, and a
// file and the same Go value hash alike.
//
// Requirement: FR-MER-054, FR-CAR-012, FR-CFG-079
func TestLoad_Merchants(t *testing.T) {
	t.Parallel()
	got, err := parseStrings(merchantsDoc)
	if err != nil {
		t.Fatalf("Parse(merchantsDoc) error = %v, want nil", err)
	}
	wantBlock := &economysim.Merchant{Modes: []string{"coach", "barge"}, LotUnits: 2, FeeShareSpreadBP: 1_500, MaxSourceAge: 3, TimeCostBP: 8_000,
		ConsigneeCutBP: 400, Consigns: true, TakeMarginBP: 900, TakeUnits: 150}
	if len(got.Roles) != 2 || !reflect.DeepEqual(got.Roles[0].Merchant, wantBlock) || got.Roles[0].MerchantFeeShare != 6_000 || got.Roles[1].Merchant != nil || got.Roles[1].MerchantFeeShare != 0 {
		t.Errorf("Parse(merchantsDoc).Roles = %+v, want broker with the block and share 6000 and farmer with neither", got.Roles)
	}
	if want := (&economysim.Merchants{MaxOpen: 500}); !reflect.DeepEqual(got.Modules.Merchants, want) {
		t.Errorf("Parse(merchantsDoc).Modules.Merchants = %+v, want %+v", got.Modules.Merchants, want)
	}
	only, err := parseStrings(minimal + "[modules.merchants]\n[role.a]\n[role.a.merchant]\nmodes = [ \"m\" ]\n")
	if err != nil {
		t.Fatalf("Parse(defaults) error = %v, want nil", err)
	}
	wantDefault := economysim.DefaultMerchant()
	wantDefault.Modes = []string{"m"}
	if only.Modules.Merchants.MaxOpen != 1_000 || !reflect.DeepEqual(*only.Roles[0].Merchant, wantDefault) || only.Roles[0].MerchantFeeShare != economysim.DefaultMerchantFeeShare {
		t.Errorf("Parse(defaults) = %+v, %+v, want the defaults of DefaultMerchants, DefaultMerchant and share 5000", only.Modules.Merchants, only.Roles[0])
	}
	if off, _ := parseStrings(minimal); off.Modules.Merchants != nil {
		t.Errorf("Parse(minimal).Modules.Merchants = %+v, want nil: the module is off", off.Modules.Merchants)
	}
	role := func(id string) economysim.Role {
		r := economysim.DefaultRole()
		r.ID = id
		return r
	}
	broker, farmer := role("broker"), role("farmer")
	broker.Strategy, broker.Informed, broker.Trades, broker.MerchantFeeShare, broker.Merchant = "merchant", true, []string{"x", "haul"}, 6_000, wantBlock
	goValue := economysim.Config{World: economysim.DefaultWorldParams(), Calendar: economysim.DefaultCalendar(), Roles: []economysim.Role{broker, farmer}}
	goValue.Modules.Merchants = &economysim.Merchants{MaxOpen: 500}
	if economysim.HashConfig(got) != economysim.HashConfig(goValue) {
		t.Errorf("HashConfig(file) != HashConfig(the same Go value), want equal (FR-MER-054)")
	}
}

// TestLoad_MerchantErrors verifies that an unknown key in the module table or in
// the block is ErrUnknownKey, that a float is ErrNotInteger and that a flag that
// is not a boolean is ErrWrongType.
//
// Requirement: FR-MER-054, FR-CFG-024
func TestLoad_MerchantErrors(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		doc  string
		want error
	}{
		{"unknown key in the module", "[modules.merchants]\nbogus = 1\n", economysim.ErrUnknownKey},
		{"unknown key in the block", "[role.a]\n[role.a.merchant]\nbogus = 1\n", economysim.ErrUnknownKey},
		{"a float max_open", "[modules.merchants]\nmax_open = 1.5\n", economysim.ErrNotInteger},
		{"a float lot", "[role.a]\n[role.a.merchant]\nlot_units = 1.5\n", economysim.ErrNotInteger},
		{"a float share", "[role.a]\nmerchant_fee_share = 5000.5\n", economysim.ErrNotInteger},
		{"consigns that is not a flag", "[role.a]\n[role.a.merchant]\nconsigns = 1\n", economysim.ErrWrongType},
		{"a string mode list", "[role.a]\n[role.a.merchant]\nmodes = \"m\"\n", economysim.ErrWrongType},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if _, err := parseStrings(minimal + tt.doc); !errors.Is(err, tt.want) {
				t.Errorf("Parse(%q) error = %v, want %v", tt.doc, err, tt.want)
			}
		})
	}
}

// FuzzLoadMerchants is FuzzLoad with the seeds of the merchants module: the
// document of spec-merchants.md 19.1 and documents that break each key of the
// module and of the block. It asserts the same properties.
//
// Requirement: FR-MER-054, FR-CFG-024, NFR-MER-005
func FuzzLoadMerchants(f *testing.F) {
	f.Add([]byte(merchantsDoc), []byte("[modules.merchants]\nmax_open = 0\n"), []byte("[role.broker.merchant]\nlot_units = 1.5\nconsigns = 1\n"))
	f.Add([]byte(minimal), []byte("[role.a]\nmerchant_fee_share = -1\n[role.a.merchant]\nmodes = [ \"a\", \"a\" ]\n"), []byte("[modules.merchants]\nbogus = 1\n"))
	f.Fuzz(loadProperties)
}
