package economysim_test

import (
	"errors"
	"reflect"
	"slices"
	"strconv"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// merchantConfig returns informationConfig with the transport module (a hired
// mode "coach" that hauls "seed" and the self-carried "foot"), the merchants
// module and a merchant role "broker" of one household per region: it trades
// "fuel" and "seed", carries by coach and consigns.
func merchantConfig() economysim.Config {
	c := informationConfig()
	c.Modules.Transport = &economysim.Transport{Modes: []economysim.Mode{footMode(), hiredMode()}}
	m := economysim.DefaultMerchants()
	c.Modules.Merchants = &m
	role := economysim.DefaultRole()
	role.ID, role.Strategy, role.Informed = "broker", "merchant", true
	role.Trades = []string{"fuel", "seed"}
	role.MerchantFeeShare = economysim.DefaultMerchantFeeShare
	b := economysim.DefaultMerchant()
	b.Modes, b.Consigns, b.TakeUnits = []string{"coach"}, true, 150
	role.Merchant = &b
	c.Roles = append(c.Roles, role)
	c.Groups = append(c.Groups, economysim.Group{Region: "north", Role: "broker", Households: 2}, economysim.Group{Region: "south", Role: "broker", Households: 2})
	return c
}

// TestValidate_MerchantConfigIsValid verifies that the configuration the merchant
// tests start from validates, so every failing case breaks one rule only.
//
// Guard: the field cases of the registry start from merchantConfig; if it did not
// validate, each case would pass for the wrong reason.
func TestValidate_MerchantConfigIsValid(t *testing.T) {
	t.Parallel()
	if err := merchantConfig().Validate(); err != nil {
		t.Fatalf("merchantConfig().Validate() = %v, want nil", err)
	}
}

// broker returns the merchant role of c.
func broker(c *economysim.Config) *economysim.Role {
	for i := range c.Roles {
		if c.Roles[i].ID == "broker" {
			return &c.Roles[i]
		}
	}
	panic("merchantConfig has no broker")
}

// merchantFieldCases are the failing cases of the rows of the merchants module in
// the parameter-owner table (spec-merchants.md 13, 14.1, FR-CFG-079): each breaks
// one field of a valid configuration. buildFieldCases appends them to the table
// that the registry test reads.
func merchantFieldCases() []fieldCase {
	mb := []string{"roles", "broker", "merchant"}
	field := func(field, name string, rule error, p []string, value, bound string, set func(c *economysim.Config)) fieldCase {
		return fieldCase{field: field, name: name, rules: []error{rule}, paths: [][]string{p}, value: value, bound: bound,
			mutate: func(c *economysim.Config) {
				*c = merchantConfig()
				set(c)
			}}
	}
	bounds := economysim.ErrOutOfBounds
	block := func(set func(b *economysim.Merchant)) func(c *economysim.Config) {
		return func(c *economysim.Config) { set(broker(c).Merchant) }
	}
	return []fieldCase{
		field("Merchants.MaxOpen", "0", bounds, []string{"modules", "merchants", "max_open"}, "0", "1",
			func(c *economysim.Config) { c.Modules.Merchants.MaxOpen = 0 }),
		field("Merchants.MaxOpen", "above MaxContracts", bounds, []string{"modules", "merchants", "max_open"}, strconv.Itoa(economysim.MaxContracts+1), strconv.Itoa(economysim.MaxContracts),
			func(c *economysim.Config) { c.Modules.Merchants.MaxOpen = economysim.MaxContracts + 1 }),
		field("Role.MerchantFeeShare", "above 10 000", bounds, []string{"roles", "broker", "merchant_fee_share"}, "10001", "10000",
			func(c *economysim.Config) {
				broker(c).MerchantFeeShare, broker(c).Merchant.FeeShareSpreadBP = 10_001, 0
			}),
		field("Role.MerchantFeeShare", "below 0", bounds, []string{"roles", "broker", "merchant_fee_share"}, "-1", "0",
			func(c *economysim.Config) { broker(c).MerchantFeeShare, broker(c).Merchant.FeeShareSpreadBP = -1, 0 }),
		field("Role.Merchant", "a role with recipes", economysim.ErrMerchantShape, []string{"roles", "broker", "recipes"}, "", "",
			func(c *economysim.Config) { broker(c).Recipes = []string{c.Recipes[0].ID} }),
		field("Role.Merchant", "strategy merchant with no block", economysim.ErrMerchantShape, []string{"roles", "broker", "merchant"}, "missing", "",
			func(c *economysim.Config) { broker(c).Merchant = nil }),
		field("Merchant.Modes", "none", bounds, append(slices.Clone(mb), "modes"), "0", strconv.Itoa(economysim.MaxModes),
			block(func(b *economysim.Merchant) { b.Modes = nil })),
		field("Merchant.Modes", "a mode that is not defined", economysim.ErrUnknownMode, append(slices.Clone(mb), "modes"), "nowhere", "",
			block(func(b *economysim.Merchant) { b.Modes = []string{"nowhere"} })),
		field("Merchant.Modes", "a mode twice", economysim.ErrDuplicateID, append(slices.Clone(mb), "modes"), "coach", "",
			block(func(b *economysim.Merchant) { b.Modes = []string{"coach", "coach"} })),
		field("Merchant.LotUnits", "0", bounds, append(slices.Clone(mb), "lot_units"), "0", "1", block(func(b *economysim.Merchant) { b.LotUnits = 0 })),
		field("Merchant.LotUnits", "above MaxLotUnits", bounds, append(slices.Clone(mb), "lot_units"), strconv.Itoa(economysim.MaxLotUnits+1), strconv.Itoa(economysim.MaxLotUnits),
			block(func(b *economysim.Merchant) { b.LotUnits = economysim.MaxLotUnits + 1 })),
		field("Merchant.FeeShareSpreadBP", "above 5 000", bounds, append(slices.Clone(mb), "fee_share_spread_bp"), "5001", "5000",
			func(c *economysim.Config) {
				broker(c).MerchantFeeShare, broker(c).Merchant.FeeShareSpreadBP = 9_000, 5_001
			}),
		field("Merchant.FeeShareSpreadBP", "above MerchantFeeShare", bounds, append(slices.Clone(mb), "fee_share_spread_bp"), "2500", "2000",
			func(c *economysim.Config) { broker(c).MerchantFeeShare = 2_000 }),
		field("Merchant.FeeShareSpreadBP", "share plus spread above 10 000", bounds, append(slices.Clone(mb), "fee_share_spread_bp"), "2500", "2000",
			func(c *economysim.Config) { broker(c).MerchantFeeShare = 8_000 }),
		field("Merchant.MaxSourceAge", "below 0", bounds, append(slices.Clone(mb), "max_source_age"), "-1", "0", block(func(b *economysim.Merchant) { b.MaxSourceAge = -1 })),
		field("Merchant.MaxSourceAge", "above MaxRumourAge", bounds, append(slices.Clone(mb), "max_source_age"), strconv.Itoa(economysim.MaxRumourAge+1), strconv.Itoa(economysim.MaxRumourAge),
			block(func(b *economysim.Merchant) { b.MaxSourceAge = economysim.MaxRumourAge + 1 })),
		field("Merchant.TimeCostBP", "above MaxTimeCostBP", bounds, append(slices.Clone(mb), "time_cost_bp"), strconv.Itoa(economysim.MaxTimeCostBP+1), strconv.Itoa(economysim.MaxTimeCostBP),
			block(func(b *economysim.Merchant) { b.TimeCostBP = economysim.MaxTimeCostBP + 1 })),
		field("Merchant.ConsigneeCutBP", "above 5 000", bounds, append(slices.Clone(mb), "consignee_cut_bp"), "5001", "5000", block(func(b *economysim.Merchant) { b.ConsigneeCutBP = 5_001 })),
		field("Merchant.Consigns", "true while the module is off", economysim.ErrModuleDependency, append(slices.Clone(mb), "consigns"), "true", "",
			func(c *economysim.Config) {
				c.Modules.Merchants = nil
				broker(c).Merchant.Modes = []string{"foot"}
			}),
		field("Merchant.TakeMarginBP", "above 9 000", bounds, append(slices.Clone(mb), "take_margin_bp"), "9001", "9000", block(func(b *economysim.Merchant) { b.TakeMarginBP = 9_001 })),
		field("Merchant.TakeUnits", "0 with Consigns", economysim.ErrMerchantShape, append(slices.Clone(mb), "take_units"), "0", "1", block(func(b *economysim.Merchant) { b.TakeUnits = 0 })),
		field("Merchant.TakeUnits", "above MaxQuantity", bounds, append(slices.Clone(mb), "take_units"), strconv.Itoa(economysim.MaxQuantity+1), strconv.Itoa(economysim.MaxQuantity),
			block(func(b *economysim.Merchant) { b.TakeUnits = economysim.MaxQuantity + 1 })),
	}
}

// TestValidate_MerchantRules verifies the rules of spec-merchants.md 14.1 that no
// row of the parameter-owner table owns, each broken alone: the shape of the role
// (Informed, the trade inputs, the cargo set, the crew of a self-carried mode),
// the weight of a group and the module dependencies.
//
// Requirement: FR-MER-003, FR-MER-004, FR-MER-050, FR-CFG-079
func TestValidate_MerchantRules(t *testing.T) {
	t.Parallel()
	mb := []string{"roles", "broker", "merchant"}
	tests := []struct {
		name   string
		mutate func(c *economysim.Config)
		rule   error
		path   []string
	}{
		{"Informed false", func(c *economysim.Config) { broker(c).Informed = false }, economysim.ErrMerchantShape, []string{"roles", "broker", "informed"}},
		{"a trade input outside Trades", func(c *economysim.Config) { broker(c).Trades = []string{"fuel"} }, economysim.ErrMerchantShape, []string{"roles", "broker", "trades"}},
		{"an empty cargo set", func(c *economysim.Config) { broker(c).Trades = []string{"seed"} }, economysim.ErrMerchantShape, []string{"roles", "broker", "trades"}},
		{"a self-carried mode of Crew 2", func(c *economysim.Config) {
			c.Modules.Transport.Modes[0].Crew = 2
			broker(c).Merchant.Modes = []string{"coach", "foot"}
		}, economysim.ErrMerchantShape, append(slices.Clone(mb), "modes", "foot", "crew")},
		{"a group of scale 2", func(c *economysim.Config) {
			for i := range c.Groups {
				if c.Groups[i].Role == "broker" {
					c.Groups[i].Households, c.Groups[i].Scale = 4, 2
				}
			}
		}, economysim.ErrMerchantWeight, []string{"groups", "north", "broker", "scale"}},
		{"a hired mode while the module is off", func(c *economysim.Config) { c.Modules.Merchants = nil }, economysim.ErrModuleDependency, append(slices.Clone(mb), "modes")},
		{"the module while transport is off", func(c *economysim.Config) {
			c.Modules.Transport = nil
			broker(c).Merchant = nil
			broker(c).Strategy = "default"
		}, economysim.ErrModuleDependency, []string{"modules", "merchants"}},
		{"a block while information is off", func(c *economysim.Config) {
			c.Modules.Information = nil
			for i := range c.Roles {
				c.Roles[i].Informed = false
			}
			broker(c).Informed = true
		}, economysim.ErrModuleDependency, mb},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := merchantConfig()
			tt.mutate(&c)
			err := c.Validate()
			if !errors.Is(err, tt.rule) {
				t.Fatalf("errors.Is(Validate(%s), %v) = false, want true; error: %v", tt.name, tt.rule, err)
			}
			vs, _ := violations(err)
			if got := pathsOf(vs); !slices.ContainsFunc(got, func(p []string) bool { return slices.Equal(p, tt.path) }) {
				t.Errorf("Validate(%s) paths = %q, want one of them to be %q", tt.name, got, tt.path)
			}
		})
	}
}

// TestHash_MerchantsOffLeavesNoTrace verifies that a configuration with no
// merchants module and no block hashes as it did before the module was added,
// the pin of the all-off reference configuration.
//
// Requirement: FR-MER-063, FR-CFG-105
//
// Guard: the append of Role.MerchantFeeShare and Role.Merchant and of the
// Modules pointer is a trace only while it is on (architecture.md 2.10.5).
func TestHash_MerchantsOffLeavesNoTrace(t *testing.T) {
	t.Parallel()
	c := referenceConfig()
	for i := range c.Roles {
		c.Roles[i].MerchantFeeShare = economysim.DefaultMerchantFeeShare
	}
	if got := hexHash(c); got != pinnedReferenceHash {
		t.Errorf("HashConfig(referenceConfig() with a fee share on every role and no block) = %s, want the pin %s", got, pinnedReferenceHash)
	}
}

// TestMerchantsConfig_DefaultsAndBounds verifies the placeholder defaults of the
// module and of the block (spec-merchants.md 13), that MerchantFeeShare defaults
// to 5 000 (FR-CAR-012), and that each field is accepted at both ends of its
// range and a Go zero value of the block's Modes is not.
//
// Requirement: FR-MER-052, FR-CAR-012
func TestMerchantsConfig_DefaultsAndBounds(t *testing.T) {
	t.Parallel()
	if economysim.DefaultMerchantFeeShare != 5_000 {
		t.Errorf("DefaultMerchantFeeShare = %d, want 5000", economysim.DefaultMerchantFeeShare)
	}
	if got := economysim.DefaultMerchants(); got != (economysim.Merchants{MaxOpen: 1_000}) {
		t.Errorf("DefaultMerchants() = %+v, want MaxOpen 1000", got)
	}
	want := economysim.Merchant{LotUnits: 1, FeeShareSpreadBP: 2_500, MaxSourceAge: 2, TimeCostBP: 10_000, ConsigneeCutBP: 500, TakeMarginBP: 1_000}
	if got := economysim.DefaultMerchant(); !reflect.DeepEqual(got, want) {
		t.Errorf("DefaultMerchant() = %+v, want %+v", got, want)
	}
	if economysim.MaxLotUnits != 64 || economysim.MaxContracts != 1_000_000 || economysim.MaxTimeCostBP != 100_000 {
		t.Errorf("bounds = %d, %d, %d, want 64, 1000000, 100000", economysim.MaxLotUnits, economysim.MaxContracts, economysim.MaxTimeCostBP)
	}
	tests := []struct {
		name   string
		mutate func(c *economysim.Config)
	}{
		{"MaxOpen at both ends", func(c *economysim.Config) { c.Modules.Merchants.MaxOpen = economysim.MaxContracts }},
		{"MaxOpen 1", func(c *economysim.Config) { c.Modules.Merchants.MaxOpen = 1 }},
		{"the share 0 with spread 0", func(c *economysim.Config) { broker(c).MerchantFeeShare, broker(c).Merchant.FeeShareSpreadBP = 0, 0 }},
		{"the share 10 000 with spread 0", func(c *economysim.Config) {
			broker(c).MerchantFeeShare, broker(c).Merchant.FeeShareSpreadBP = 10_000, 0
		}},
		{"the spread equal to the share", func(c *economysim.Config) {
			broker(c).MerchantFeeShare, broker(c).Merchant.FeeShareSpreadBP = 5_000, 5_000
		}},
		{"LotUnits at its bounds", func(c *economysim.Config) { broker(c).Merchant.LotUnits = economysim.MaxLotUnits }},
		{"MaxSourceAge at its bounds", func(c *economysim.Config) { broker(c).Merchant.MaxSourceAge = economysim.MaxRumourAge }},
		{"MaxSourceAge 0", func(c *economysim.Config) { broker(c).Merchant.MaxSourceAge = 0 }},
		{"TimeCostBP at its bounds", func(c *economysim.Config) { broker(c).Merchant.TimeCostBP = economysim.MaxTimeCostBP }},
		{"the cut and the margin at their bounds", func(c *economysim.Config) {
			broker(c).Merchant.ConsigneeCutBP, broker(c).Merchant.TakeMarginBP = 5_000, 9_000
		}},
		{"TakeUnits at its bound", func(c *economysim.Config) { broker(c).Merchant.TakeUnits = economysim.MaxQuantity }},
		{"no Consigns and no TakeUnits", func(c *economysim.Config) { broker(c).Merchant.Consigns, broker(c).Merchant.TakeUnits = false, 0 }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := merchantConfig()
			tt.mutate(&c)
			if err := c.Validate(); err != nil {
				t.Errorf("Validate(%s) = %v, want nil", tt.name, err)
			}
		})
	}
}

// TestWarnings_Merchant verifies the eight warnings of spec-merchants.md 14.2:
// each is raised once, with the path of the field and, for the two that count
// pairs, the Count, and the valid base configuration raises none of them.
//
// Requirement: FR-MER-051
func TestWarnings_Merchant(t *testing.T) {
	t.Parallel()
	merchantRules := []string{"merchant_spread_narrow", "merchant_fee_share_high", "merchant_fee_share_low", "merchant_no_consignee",
		"lane_without_consignee", "cargo_never_delivered", "consigns_without_cargo", "merchants_one_agent"}
	mb := []string{"roles", "broker", "merchant"}
	tests := []struct {
		name   string
		mutate func(c *economysim.Config)
		rule   string
		path   []string
		count  int64
	}{
		{"the bars are within one cost", func(c *economysim.Config) { broker(c).Merchant.FeeShareSpreadBP = 0 }, "merchant_spread_narrow", append(slices.Clone(mb), "fee_share_spread_bp"), 0},
		{"the top share is above 9 000", func(c *economysim.Config) { broker(c).MerchantFeeShare, broker(c).Merchant.FeeShareSpreadBP = 9_500, 0 }, "merchant_fee_share_high", append(slices.Clone(mb), "fee_share_spread_bp"), 0},
		{"the lowest share is below 500", func(c *economysim.Config) { broker(c).MerchantFeeShare, broker(c).Merchant.FeeShareSpreadBP = 400, 0 }, "merchant_fee_share_low", append(slices.Clone(mb), "fee_share_spread_bp"), 0},
		{"a hired mode and nobody consigns", func(c *economysim.Config) { broker(c).Merchant.Consigns, broker(c).Merchant.TakeUnits = false, 0 }, "merchant_no_consignee", append(slices.Clone(mb), "modes"), 0},
		{"two routes end where nobody consigns", func(c *economysim.Config) { broker(c).Merchant.Consigns, broker(c).Merchant.TakeUnits = false, 0 }, "lane_without_consignee", append(slices.Clone(mb), "modes"), 2},
		{"a cargo that spoils before the shortest route ends", func(c *economysim.Config) {
			for i := range c.Commodities {
				if c.Commodities[i].ID == "fuel" {
					c.Commodities[i].ShelfLife = 2
				}
			}
		}, "cargo_never_delivered", append(slices.Clone(mb), "modes"), 1},
		{"a consignee that nothing is sent to", func(c *economysim.Config) {
			c.Groups = slices.DeleteFunc(c.Groups, func(g economysim.Group) bool { return g.Role == "broker" && g.Region == "south" })
		}, "consigns_without_cargo", append(slices.Clone(mb), "consigns"), 0},
		{"one merchant in the world", func(c *economysim.Config) {
			c.Groups = slices.DeleteFunc(c.Groups, func(g economysim.Group) bool { return g.Role == "broker" && g.Region == "south" })
			for i := range c.Groups {
				if c.Groups[i].Role == "broker" {
					c.Groups[i].Households = 1
				}
			}
		}, "merchants_one_agent", []string{"roles", "broker"}, 0},
	}
	base := merchantConfig()
	for _, w := range base.Warnings() {
		if slices.Contains(merchantRules, w.Rule) {
			t.Errorf("Warnings() of the valid base raised %+v, want no merchant warning", w)
		}
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := merchantConfig()
			tt.mutate(&c)
			if err := c.Validate(); err != nil {
				t.Fatalf("Validate(%s) = %v, want nil", tt.name, err)
			}
			var got []economysim.Warning
			for _, w := range c.Warnings() {
				if w.Rule == tt.rule {
					got = append(got, w)
				}
			}
			if len(got) != 1 || !slices.Equal(got[0].Path, tt.path) || got[0].Count != tt.count {
				t.Errorf("Warnings() rule %s = %+v, want once at %v with Count %d", tt.rule, got, tt.path, tt.count)
			}
		})
	}
}

// TestModulesOn_ListsMerchants verifies that the snapshot lists "merchants" when
// Modules.Merchants is set and does not when it is not.
//
// Requirement: FR-MER-052
func TestModulesOn_ListsMerchants(t *testing.T) {
	t.Parallel()
	for _, on := range []bool{true, false} {
		c := merchantConfig()
		if !on {
			c.Modules.Merchants = nil
			broker(&c).Merchant.Modes, broker(&c).Merchant.Consigns, broker(&c).Merchant.TakeUnits = []string{"foot"}, false, 0
		}
		w, err := economysim.NewWorld(c, economysim.WithStrategy("merchant", economysim.DefaultStrategy{}))
		if err != nil {
			t.Fatalf("NewWorld(merchants %v) = %v, want nil", on, err)
		}
		snap, err := w.Snapshot()
		if err != nil {
			t.Fatalf("Snapshot() = %v, want nil", err)
		}
		if got := slices.Contains(snap.Modules, "merchants"); got != on {
			t.Errorf("Snapshot().Modules = %v with the module %v, want merchants listed %v", snap.Modules, on, on)
		}
	}
}
