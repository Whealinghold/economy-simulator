package economysim_test

import (
	"errors"
	"slices"
	"strconv"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// The tests of the tenure module's configuration surface (spec-land.md 3.3, 10, 11):
// the failing case of each rule of 11.1, their order and the canonical order. The
// warnings are in tenure_warnings_test.go, the hash in tenure_hash_test.go and the
// loader in config/tenure_test.go. The module does nothing at run time in this change.

// tenureConfig returns referenceConfig with the resources module, the estates module
// and the tenure module on: the grower households of north hold 20 basis points of
// the arable land each (253 households, 5 060 of 10 000), the pool of north holds 1 000,
// the arable land lets at 3 a unit and the posted rent steps by 500, inside a band of
// 2 000 to 15 000. The estates module is on so that rent_hoarded is not raised, and no
// other warning of the module is raised either.
func tenureConfig() economysim.Config {
	c := referenceConfig()
	withResources(&c)
	c.Modules.Estates = &economysim.Estates{}
	c.Modules.Tenure = &economysim.TenureConfig{
		Terms: []economysim.TenureTerm{{Region: "north", Resource: "arable", HasRent: true, Rent: 3}},
		Holdings: []economysim.Holding{
			{Region: "north", Resource: "arable", Holder: economysim.HolderRef{Kind: economysim.HolderKindRole, Role: "grower"}, ShareBP: 20},
			{Region: "north", Resource: "arable", Holder: economysim.HolderRef{Kind: economysim.HolderKindPool}, ShareBP: 1_000},
		},
		RentStepBP: 500, RentFloorBP: 2_000, RentCeilingBP: 15_000,
	}
	return c
}

// tenureCase builds one failing case from a mutation of the module of tenureConfig.
type tenureCase func(field, name string, rule error, p []string, value, bound string, set func(c *economysim.Config)) fieldCase

// tenureAt is the path of a field of the module, or of one of its rows.
func tenureAt(list string, i int, key ...string) []string {
	return append([]string{"modules", "tenure", list + "[" + strconv.Itoa(i) + "]"}, key...)
}

// tenureFieldCases are the failing cases of the rows of the tenure module in the
// parameter-owner table (spec-land.md 10, 11.1, FR-LND-038): each breaks one field of
// the valid configuration of tenureConfig. buildFieldCases appends them to the table
// that the registry test reads.
func tenureFieldCases() []fieldCase {
	ten := func(field, name string, rule error, p []string, value, bound string, set func(c *economysim.Config)) fieldCase {
		return fieldCase{field: field, name: name, rules: []error{rule}, paths: [][]string{p}, value: value, bound: bound,
			mutate: func(c *economysim.Config) {
				*c = tenureConfig()
				set(c)
			}}
	}
	var cs []fieldCase
	cs = append(cs, tenureTunableCases(ten)...)
	cs = append(cs, tenureTermCases(ten)...)
	cs = append(cs, tenureHoldingCases(ten)...)
	return append(cs, tenureShareCases(ten)...)
}

// tenureTunableCases are the cases of the module's dependency, its two counts and its
// five tunables.
func tenureTunableCases(ten tenureCase) []fieldCase {
	m := func(c *economysim.Config) *economysim.TenureConfig { return c.Modules.Tenure }
	base := []string{"modules", "tenure"}
	ob := economysim.ErrOutOfBounds
	return []fieldCase{
		ten("Modules.Tenure", "without the resources module", economysim.ErrModuleDependency, base, "resources", "",
			func(c *economysim.Config) {
				recipeOf(c, "grow_staple").Draws, recipeOf(c, "cut_fuel").Draws = nil, nil
				c.Modules.Resources = nil
			}),
		ten("TenureConfig.Terms", "more than MaxHoldings", ob, append(slices.Clone(base), "terms"), "4097", "4096",
			func(c *economysim.Config) {
				m(c).Terms = slices.Repeat(m(c).Terms, economysim.MaxHoldings+1)
			}),
		ten("TenureConfig.Holdings", "more than MaxHoldings", ob, append(slices.Clone(base), "holdings"), "4097", "4096",
			func(c *economysim.Config) {
				m(c).Holdings = slices.Repeat(m(c).Holdings[:1], economysim.MaxHoldings+1)
			}),
		ten("TenureConfig.RentStepBP", "above MaxRentStepBP", ob, append(slices.Clone(base), "rent_step_bp"), "10001", "10000",
			func(c *economysim.Config) { m(c).RentStepBP = economysim.MaxRentStepBP + 1 }),
		ten("TenureConfig.RentStepBP", "below 0", ob, append(slices.Clone(base), "rent_step_bp"), "-1", "0",
			func(c *economysim.Config) { m(c).RentStepBP = -1 }),
		ten("TenureConfig.RentFloorBP", "above 10 000", ob, append(slices.Clone(base), "rent_floor_bp"), "10001", "10000",
			func(c *economysim.Config) { m(c).RentFloorBP = 10_001 }),
		ten("TenureConfig.RentFloorBP", "below 0", ob, append(slices.Clone(base), "rent_floor_bp"), "-1", "0",
			func(c *economysim.Config) { m(c).RentFloorBP = -1 }),
		ten("TenureConfig.RentCeilingBP", "below 10 000", ob, append(slices.Clone(base), "rent_ceiling_bp"), "9999", "10000",
			func(c *economysim.Config) { m(c).RentCeilingBP = 9_999 }),
		ten("TenureConfig.RentCeilingBP", "above MaxRentCeilingBP", ob, append(slices.Clone(base), "rent_ceiling_bp"), "100001", "100000",
			func(c *economysim.Config) { m(c).RentCeilingBP = economysim.MaxRentCeilingBP + 1 }),
		ten("TenureConfig.RentCeilingBP", "below 0", ob, append(slices.Clone(base), "rent_ceiling_bp"), "-1", "10000",
			func(c *economysim.Config) { m(c).RentCeilingBP = -1 }),
		ten("TenureConfig.MaxHolderShareBP", "above 10 000", ob, append(slices.Clone(base), "max_holder_share_bp"), "10001", "10000",
			func(c *economysim.Config) { m(c).MaxHolderShareBP = 10_001 }),
		ten("TenureConfig.MaxHolderShareBP", "below 0", ob, append(slices.Clone(base), "max_holder_share_bp"), "-1", "1",
			func(c *economysim.Config) { m(c).MaxHolderShareBP = -1 }),
	}
}

// tenureTermCases are the cases of the fields of a term row.
func tenureTermCases(ten tenureCase) []fieldCase {
	term := func(c *economysim.Config) *economysim.TenureTerm { return &c.Modules.Tenure.Terms[0] }
	return []fieldCase{
		ten("TenureTerm.Region", "not defined", economysim.ErrUnknownRegion, tenureAt("term", 0, "region"), "nowhere", "",
			func(c *economysim.Config) { term(c).Region = "nowhere" }),
		ten("TenureTerm.Region", "an outside node", economysim.ErrOutsideNode, tenureAt("term", 0, "region"), "coast", "",
			func(c *economysim.Config) { withOutsideNode(c); term(c).Region = "coast" }),
		ten("TenureTerm.Resource", "not defined", economysim.ErrUnknownResource, tenureAt("term", 0, "resource"), "nowhere", "",
			func(c *economysim.Config) { term(c).Resource = "nowhere" }),
		ten("TenureTerm.Resource", "twice in a region", economysim.ErrDuplicateID, tenureAt("term", 1, "resource"), "arable", "",
			func(c *economysim.Config) {
				c.Modules.Tenure.Terms = append(c.Modules.Tenure.Terms, c.Modules.Tenure.Terms[0])
			}),
		ten("TenureTerm.Rent", "above MaxPrice", economysim.ErrOutOfBounds, tenureAt("term", 0, "rent"), "1000000001", "1000000000",
			func(c *economysim.Config) { term(c).Rent = economysim.MaxPrice + 1 }),
		ten("TenureTerm.Rent", "below 0", economysim.ErrOutOfBounds, tenureAt("term", 0, "rent"), "-1", "0",
			func(c *economysim.Config) { term(c).Rent = -1 }),
		ten("TenureTerm.Rent", "times the largest draw and weight above MaxMoney", economysim.ErrOverflow, tenureAt("term", 0, "rent"), "", "1000000000000000000",
			func(c *economysim.Config) { heavyDraw(c); term(c).Rent = economysim.MaxPrice }),
	}
}

// withOutsideNode adds the region coast to c as the outside node of an external module
// that trades nothing, with the edges that the module needs, as externalConfig has them.
func withOutsideNode(c *economysim.Config) {
	r := economysim.DefaultRegion()
	r.ID = "coast"
	c.Regions = append(c.Regions, r)
	c.Edges = []economysim.Edge{{From: "north", To: "coast", Distance: 30}, {From: "coast", To: "north", Distance: 30}}
	ore := economysim.DefaultOutsideLine("mineral", 837)
	ore.Sell = economysim.OutsideSide{MarginBP: 500, Capacity: 10}
	c.Modules.External = &economysim.External{Nodes: []economysim.OutsideNode{{Region: "coast", Lines: []economysim.OutsideLine{ore}}}}
}

// heavyDraw makes the grower draw the most of the arable land that one agent may, and
// one group very heavy: no product the validator checks alone is above its bound, but
// the rent times the largest draw times the largest weight is above MaxMoney. Each
// grower is an agent of weight 1, and the cooks are agents of a hundred thousand
// households each.
func heavyDraw(c *economysim.Config) {
	recipeOf(c, "grow_staple").Draws = []economysim.Draw{{Resource: "arable", Quantity: economysim.MaxQuantity}}
	for i := range c.Groups {
		switch c.Groups[i].Role {
		case "grower":
			c.Groups[i].Scale = 1
		case "cook":
			c.Groups[i].Households, c.Groups[i].Scale = 1_000_000, 100_000
		}
	}
}

// tenureHoldingCases are the cases of the key and the holder of a holding row.
func tenureHoldingCases(ten tenureCase) []fieldCase {
	h := func(c *economysim.Config) *economysim.Holding { return &c.Modules.Tenure.Holdings[0] }
	return []fieldCase{
		ten("Holding.Region", "not defined", economysim.ErrUnknownRegion, tenureAt("holding", 0, "region"), "nowhere", "",
			func(c *economysim.Config) { h(c).Region = "nowhere" }),
		ten("Holding.Region", "an outside node", economysim.ErrOutsideNode, tenureAt("holding", 0, "region"), "coast", "",
			func(c *economysim.Config) { withOutsideNode(c); h(c).Region = "coast" }),
		ten("Holding.Resource", "not defined", economysim.ErrUnknownResource, tenureAt("holding", 0, "resource"), "nowhere", "",
			func(c *economysim.Config) { h(c).Resource = "nowhere" }),
		ten("Holding.Holder", "a role that is not defined", economysim.ErrUnknownRole, tenureAt("holding", 0, "holder", "role"), "serf", "",
			func(c *economysim.Config) { h(c).Holder.Role = "serf" }),
		ten("Holding.Holder", "a kind that is neither role nor pool", economysim.ErrOutOfBounds, tenureAt("holding", 0, "holder", "kind"), "guild", "role or pool",
			func(c *economysim.Config) { h(c).Holder = economysim.HolderRef{Kind: "guild"} }),
		ten("Holding.Holder", "the pool with a role", economysim.ErrOutOfBounds, tenureAt("holding", 1, "holder", "role"), "grower", "",
			func(c *economysim.Config) { c.Modules.Tenure.Holdings[1].Holder.Role = "grower" }),
		ten("Holding.Holder", "twice for one endowment", economysim.ErrDuplicateID, tenureAt("holding", 2, "holder"), "grower", "",
			func(c *economysim.Config) {
				c.Modules.Tenure.Holdings = append(c.Modules.Tenure.Holdings, c.Modules.Tenure.Holdings[0])
			}),
		ten("Holding.Holder", "a role named pool", economysim.ErrDuplicateID, []string{"roles", "pool", "id"}, "pool", "",
			func(c *economysim.Config) {
				ro := economysim.DefaultRole()
				ro.ID = "pool"
				c.Roles = append(c.Roles, ro)
			}),
	}
}

// tenureShareCases are the cases of the share of a holding and of the sum and the cap.
func tenureShareCases(ten tenureCase) []fieldCase {
	share := economysim.ErrTitleShares
	return []fieldCase{
		ten("Holding.ShareBP", "0", share, tenureAt("holding", 0, "share_bp"), "0", "1",
			func(c *economysim.Config) { c.Modules.Tenure.Holdings[0].ShareBP = 0 }),
		ten("Holding.ShareBP", "above 10 000", share, tenureAt("holding", 1, "share_bp"), "10001", "10000",
			func(c *economysim.Config) { c.Modules.Tenure.Holdings[1].ShareBP = 10_001 }),
		ten("Holding.ShareBP", "with the census above 10 000 on one endowment", share, tenureAt("holding", 1, "share_bp"), "11060", "10000",
			func(c *economysim.Config) { c.Modules.Tenure.Holdings[1].ShareBP = 6_000 }),
		ten("Holding.ShareBP", "of the largest agent above MaxHolderShareBP", economysim.ErrTitleCap, tenureAt("holding", 0, "share_bp"), "5060", "5000",
			func(c *economysim.Config) {
				// One grower agent of 253 households holds 253 * 20 = 5 060 basis points.
				c.World.Scale, c.World.MinAgents = 1_000, 1
				c.Modules.Tenure.MaxHolderShareBP = 5_000
			}),
	}
}

// TestValidate_TenureBaseIsValid verifies that tenureConfig validates and raises no
// warning of the tenure module, so that each failing case above breaks one rule only.
//
// Guard: every case of tenureFieldCases starts from it; a base that already warns or
// fails would hide the one rule a case is for.
func TestValidate_TenureBaseIsValid(t *testing.T) {
	t.Parallel()
	c := tenureConfig()
	if err := c.Validate(); err != nil {
		t.Fatalf("Validate(tenureConfig) = %v, want nil", err)
	}
	for _, w := range c.Warnings() {
		if len(w.Path) > 1 && w.Path[1] == "tenure" {
			t.Errorf("Warnings(tenureConfig) holds %v, want none of the module's", w)
		}
	}
}

// TestValidate_TenureOffAndEmpty verifies that the module off validates with no
// trace, and that a module that is on with nothing in it is legal and does nothing
// (spec-land.md 3.3.3): zero for the ceiling and the cap are their defaults.
//
// Requirement: FR-LND-001, FR-LND-038
func TestValidate_TenureOffAndEmpty(t *testing.T) {
	t.Parallel()
	c := tenureConfig()
	c.Modules.Tenure = &economysim.TenureConfig{}
	if err := c.Validate(); err != nil {
		t.Errorf("Validate(empty tenure module) = %v, want nil", err)
	}
	c.Modules.Tenure = nil
	if err := c.Validate(); err != nil {
		t.Errorf("Validate(tenure off) = %v, want nil", err)
	}
}

// TestValidate_TenureFlowCannotNameAReason verifies that a money flow whose ID is rent
// or title_sale is ErrReasonCollision, the reasons of spec-land.md 5.8 being reserved.
//
// Requirement: FR-LND-017, FR-LND-038
func TestValidate_TenureFlowCannotNameAReason(t *testing.T) {
	t.Parallel()
	for _, id := range []string{"rent", "title_sale"} {
		t.Run(id, func(t *testing.T) {
			t.Parallel()
			c := tenureConfig()
			withMoney(&c)
			c.Modules.MoneySupply.Flows[0].ID = id
			if err := c.Validate(); !errors.Is(err, economysim.ErrReasonCollision) {
				t.Errorf("Validate(flow %q) = %v, want ErrReasonCollision", id, err)
			}
		})
	}
}

// TestValidate_TenureErrorsInPhases verifies that a phase that finds an error ends
// the run, so the first error is the same for the same mistake: a configuration with
// a count over its bound and a bad tunable reports only the count.
//
// Guard: the order of the rules is part of the error a consumer sees first.
func TestValidate_TenureErrorsInPhases(t *testing.T) {
	t.Parallel()
	c := tenureConfig()
	c.Modules.Tenure.RentStepBP = -1
	c.Modules.Tenure.Terms[0].Region = "nowhere"
	err := c.Validate()
	vs, _ := violations(err)
	if got := pathsOf(vs); len(got) != 1 || got[0][2] != "rent_step_bp" {
		t.Errorf("Validate(bad tunable and bad term) paths = %q, want only the tunable", got)
	}
	c.Modules.Tenure.RentStepBP = 0
	if !errors.Is(c.Validate(), economysim.ErrUnknownRegion) {
		t.Errorf("Validate(bad term) is not ErrUnknownRegion, want it once the tunables are fixed")
	}
}

// TestSentinels_Tenure verifies that the four sentinels of the module are distinct,
// non-nil errors that name their rule.
//
// Requirement: FR-LND-038
func TestSentinels_Tenure(t *testing.T) {
	t.Parallel()
	all := []error{economysim.ErrTitleShares, economysim.ErrTitleCap, economysim.ErrNoTitle, economysim.ErrNotAHolder}
	for i, a := range all {
		if a == nil || a.Error() == "" {
			t.Fatalf("sentinel %d is nil or has no text", i)
		}
		for j, b := range all {
			if i != j && errors.Is(a, b) {
				t.Errorf("errors.Is(sentinel %d, sentinel %d) = true, want false", i, j)
			}
		}
	}
}
