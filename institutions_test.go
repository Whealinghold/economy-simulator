package economysim_test

import (
	"errors"
	"slices"
	"strconv"
	"strings"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// The tests of the institutions module's configuration surface (spec-institutions.md
// 9, 10): the failing case of each rule of 10.1, their order and the warnings of 10.2.
// The hash is in institutions_hash_test.go and the loader in config/institutions_test.go.

// institutionsConfig returns referenceConfig with the polity, settlements and credit
// modules and the institutions module on. The polity has the jurisdiction crown over
// every region; the settlements module puts north, with 600 households, in the class
// town, which carries the flag fair; the cooks lend at 600 basis points. The body
// guild holds the role maker in north with 60 places (55 households start); its
// rights are a produce right on make_tool_a and a sell right on tool_a in north; the
// control caps staple at 12 000 basis points of its reference price and floors it at
// 8 000, for the crown; the rate limit is 1 200 basis points for the crown. None of
// the module's warnings is raised.
func institutionsConfig() economysim.Config {
	c := polityConfig()
	m := economysim.DefaultSettlements()
	m.Dwell = 400 // above the effective welfare window of every gated role
	m.Classes = []economysim.SettlementClass{
		{ID: "hamlet", Enter: 0},
		{ID: "village", Enter: 300},
		{ID: "town", Enter: 500, Flags: []string{"fair"}},
	}
	c.Modules.Settlements = &m
	c.Modules.Credit = &economysim.Credit{MinPrincipal: 5_000, ShieldTicks: 2, MaxClaims: 1_000}
	creditRole(&c, "grower").Credit = &economysim.CreditTerms{Borrow: &economysim.BorrowTerms{MaxRateBP: 1_500, MaxDebtTicks: 90, SlackTicks: 4}}
	creditRole(&c, "cook").Credit = &economysim.CreditTerms{Lend: &economysim.LendTerms{RateBP: 600, MaxTermTicks: 90, KeepTicks: 3, ShareBP: 5_000}}
	c.Modules.Institutions = &economysim.Institutions{
		Bodies: []economysim.Body{{ID: "guild", Roles: []string{"maker"}, Regions: []string{"north"}, Places: 60, MaxTrainees: 2, Needs: "fair", Label: "Guild"}},
		Rights: []economysim.Right{
			{ID: "make_tools", Body: "guild", Activity: economysim.ActivityProduce, Subject: "make_tool_a", Region: "north", Who: economysim.WhoPrincipal},
			{ID: "sell_tools", Body: "guild", Activity: economysim.ActivitySell, Subject: "tool_a", Region: "north"},
		},
		Controls:   []economysim.Control{{ID: "assize", Commodity: "staple", Jurisdiction: "crown", CapBP: 12_000, FloorBP: 8_000}},
		RateLimits: []economysim.RateLimit{{ID: "usury", Jurisdiction: "crown", CapBP: 1_200}},
	}
	return c
}

// orgNoPolity switches the polity module off and keys every control and rate limit by the
// region north, so that a jurisdiction named afterwards is the only one the module lacks.
func orgNoPolity(c *economysim.Config, m *economysim.Institutions) {
	c.Modules.Polity = nil
	m.Controls[0].Region, m.Controls[0].Jurisdiction = "north", ""
	m.RateLimits[0].Region, m.RateLimits[0].Jurisdiction = "north", ""
}

// orgAt is the path of a key of element i of a list of the module.
func orgAt(list string, i int, key ...string) []string {
	return append([]string{"modules", "institutions", list + "[" + strconv.Itoa(i) + "]"}, key...)
}

// orgSet is the change of a case: it gets the configuration and its module.
type orgSet func(c *economysim.Config, m *economysim.Institutions)

// orgCase builds one failing case that starts from institutionsConfig.
func orgCase(field, name string, rule error, p []string, value, bound string, set orgSet) fieldCase {
	return fieldCase{field: field, name: name, rules: []error{rule}, paths: [][]string{p}, value: value, bound: bound,
		mutate: func(c *economysim.Config) {
			*c = institutionsConfig()
			set(c, c.Modules.Institutions)
		}}
}

// institutionsFieldCases are the failing cases of the rows of the institutions module in
// the parameter-owner table (spec-institutions.md 9, 10.1, FR-ORG-047): each breaks one
// field of the valid configuration of institutionsConfig. buildFieldCases appends them
// to the table that the registry test reads.
func institutionsFieldCases() []fieldCase {
	var cs []fieldCase
	cs = append(cs, orgCountCases()...)
	cs = append(cs, orgBodyCases()...)
	cs = append(cs, orgBodyBoundCases()...)
	cs = append(cs, orgRightCases()...)
	cs = append(cs, orgRightScopeCases()...)
	cs = append(cs, orgControlCases()...)
	return append(cs, orgRateLimitCases()...)
}

// orgCountCases are the cases of the four lists.
func orgCountCases() []fieldCase {
	tooMany := func(list string) []string { return []string{"modules", "institutions", list} }
	return []fieldCase{
		orgCase("Institutions.Bodies", "more than MaxBodies", economysim.ErrTooMany, tooMany("bodies"), "257", "256",
			func(_ *economysim.Config, m *economysim.Institutions) {
				m.Bodies = slices.Repeat(m.Bodies, economysim.MaxBodies+1)
			}),
		orgCase("Institutions.Rights", "more than MaxRights", economysim.ErrTooMany, tooMany("rights"), "4097", "4096",
			func(_ *economysim.Config, m *economysim.Institutions) {
				m.Rights = slices.Repeat(m.Rights[:1], economysim.MaxRights+1)
			}),
		orgCase("Institutions.Controls", "more than MaxControls", economysim.ErrTooMany, tooMany("controls"), "4097", "4096",
			func(_ *economysim.Config, m *economysim.Institutions) {
				m.RateLimits = nil
				m.Controls = slices.Repeat(m.Controls, economysim.MaxControls+1)
			}),
		orgCase("Institutions.RateLimits", "controls and rate limits together above MaxControls", economysim.ErrTooMany, tooMany("controls"), "4097", "4096",
			func(_ *economysim.Config, m *economysim.Institutions) {
				m.RateLimits = slices.Repeat(m.RateLimits, economysim.MaxControls)
			}),
	}
}

// orgBodyCases are the cases of the identity, roles, regions, jurisdictions and need of a
// body, and of the overlap.
func orgBodyCases() []fieldCase {
	return []fieldCase{
		orgCase("Body.ID", "upper case letter", economysim.ErrInvalidID, orgAt("bodies", 0, "id"), "Guild", "",
			func(_ *economysim.Config, m *economysim.Institutions) { m.Bodies[0].ID = "Guild" }),
		orgCase("Body.ID", "twice", economysim.ErrDuplicateID, orgAt("bodies", 1, "id"), "guild", "",
			func(_ *economysim.Config, m *economysim.Institutions) {
				m.Bodies = append(m.Bodies, economysim.Body{ID: "guild", Roles: []string{"cook"}, Regions: []string{"north"}})
			}),
		orgCase("Body.Roles", "none", economysim.ErrBodyShape, orgAt("bodies", 0, "roles"), "0", "1",
			func(_ *economysim.Config, m *economysim.Institutions) { m.Bodies[0].Roles = nil }),
		orgCase("Body.Roles", "a role that is not defined", economysim.ErrUnknownRole, orgAt("bodies", 0, "roles[0]"), "nowhere", "",
			func(_ *economysim.Config, m *economysim.Institutions) { m.Bodies[0].Roles = []string{"nowhere"} }),
		orgCase("Body.Roles", "a role and a region in two bodies", economysim.ErrBodyOverlap, orgAt("bodies", 1, "roles"), "maker in north", "guild",
			func(_ *economysim.Config, m *economysim.Institutions) {
				m.Bodies = append(m.Bodies, economysim.Body{ID: "league", Roles: []string{"maker", "cook"}, Regions: []string{"north"}})
			}),
		orgCase("Body.Roles", "a role in two bodies, one scope by jurisdiction", economysim.ErrBodyOverlap, orgAt("bodies", 1, "roles"), "maker in north", "guild",
			func(_ *economysim.Config, m *economysim.Institutions) {
				m.Bodies = append(m.Bodies, economysim.Body{ID: "league", Roles: []string{"maker"}, Jurisdictions: []string{"crown"}})
			}),
		orgCase("Body.Regions", "no scope", economysim.ErrBodyShape, orgAt("bodies", 0, "regions"), "0", "1",
			func(_ *economysim.Config, m *economysim.Institutions) { m.Bodies[0].Regions = nil }),
		orgCase("Body.Regions", "a region that is not defined", economysim.ErrUnknownRegion, orgAt("bodies", 0, "regions[0]"), "nowhere", "",
			func(_ *economysim.Config, m *economysim.Institutions) { m.Bodies[0].Regions = []string{"nowhere"} }),
		orgCase("Body.Regions", "an outside node", economysim.ErrOutsideNode, orgAt("bodies", 0, "regions[0]"), "coast", "",
			func(c *economysim.Config, m *economysim.Institutions) {
				withOutsideNode(c)
				m.Bodies[0].Regions = []string{"coast"}
			}),
		orgCase("Body.Jurisdictions", "a jurisdiction that is not defined", economysim.ErrUnknownJurisdiction, orgAt("bodies", 0, "jurisdictions[0]"), "nowhere", "",
			func(_ *economysim.Config, m *economysim.Institutions) {
				m.Bodies[0].Jurisdictions = []string{"nowhere"}
			}),
		orgCase("Body.Jurisdictions", "the polity module off", economysim.ErrModuleDependency, orgAt("bodies", 0, "jurisdictions"), "crown", "polity",
			func(c *economysim.Config, m *economysim.Institutions) {
				orgNoPolity(c, m)
				m.Bodies[0].Jurisdictions = []string{"crown"}
			}),
		orgCase("Body.Needs", "a flag that no class carries", economysim.ErrUnknownSettlementFlag, orgAt("bodies", 0, "needs"), "market", "",
			func(_ *economysim.Config, m *economysim.Institutions) { m.Bodies[0].Needs = "market" }),
		orgCase("Body.Needs", "the settlements module off", economysim.ErrModuleDependency, orgAt("bodies", 0, "needs"), "fair", "settlements",
			func(c *economysim.Config, _ *economysim.Institutions) { c.Modules.Settlements = nil }),
	}
}

// orgBodyBoundCases are the bound cases of a body.
func orgBodyBoundCases() []fieldCase {
	return []fieldCase{
		orgCase("Body.Places", "above MaxHouseholds", economysim.ErrOutOfBounds, orgAt("bodies", 0, "places"), "1000000001", "1000000000",
			func(_ *economysim.Config, m *economysim.Institutions) {
				m.Bodies[0].Places = economysim.MaxHouseholds + 1
			}),
		orgCase("Body.Places", "below 0", economysim.ErrOutOfBounds, orgAt("bodies", 0, "places"), "-1", "0",
			func(_ *economysim.Config, m *economysim.Institutions) { m.Bodies[0].Places = -1 }),
		orgCase("Body.MaxTrainees", "above 100", economysim.ErrOutOfBounds, orgAt("bodies", 0, "max_trainees"), "101", "100",
			func(_ *economysim.Config, m *economysim.Institutions) { m.Bodies[0].MaxTrainees = 101 }),
		orgCase("Body.MaxTrainees", "below 0", economysim.ErrOutOfBounds, orgAt("bodies", 0, "max_trainees"), "-1", "0",
			func(_ *economysim.Config, m *economysim.Institutions) { m.Bodies[0].MaxTrainees = -1 }),
		orgCase("Body.Label", "over MaxLabelBytes", economysim.ErrOutOfBounds, orgAt("bodies", 0, "label"), "257", "256",
			func(_ *economysim.Config, m *economysim.Institutions) {
				m.Bodies[0].Label = strings.Repeat("x", economysim.MaxLabelBytes+1)
			}),
	}
}

// orgRightCases are the cases of the identity, holder, activity, subject and Who of a
// right.
func orgRightCases() []fieldCase {
	return []fieldCase{
		orgCase("Right.ID", "upper case letter", economysim.ErrInvalidID, orgAt("rights", 0, "id"), "Make", "",
			func(_ *economysim.Config, m *economysim.Institutions) { m.Rights[0].ID = "Make" }),
		orgCase("Right.ID", "twice", economysim.ErrDuplicateID, orgAt("rights", 1, "id"), "make_tools", "",
			func(_ *economysim.Config, m *economysim.Institutions) { m.Rights[1].ID = "make_tools" }),
		orgCase("Right.Body", "a body that is not defined", economysim.ErrUnknownBody, orgAt("rights", 0, "body"), "nowhere", "",
			func(_ *economysim.Config, m *economysim.Institutions) { m.Rights[0].Body = "nowhere" }),
		orgCase("Right.Activity", "outside the four", economysim.ErrRightShape, orgAt("rights", 0, "activity"), "ship", "",
			func(_ *economysim.Config, m *economysim.Institutions) { m.Rights[0].Activity = "ship" }),
		orgCase("Right.Subject", "a recipe that is not defined", economysim.ErrUnknownRecipe, orgAt("rights", 0, "subject"), "nowhere", "",
			func(_ *economysim.Config, m *economysim.Institutions) { m.Rights[0].Subject = "nowhere" }),
		orgCase("Right.Subject", "a commodity that is not defined", economysim.ErrUnknownCommodity, orgAt("rights", 1, "subject"), "nowhere", "",
			func(_ *economysim.Config, m *economysim.Institutions) { m.Rights[1].Subject = "nowhere" }),
		orgCase("Right.Subject", "a role that is not defined", economysim.ErrUnknownRole, orgAt("rights", 0, "subject"), "nowhere", "",
			func(_ *economysim.Config, m *economysim.Institutions) {
				m.Rights[0].Activity, m.Rights[0].Subject = economysim.ActivityTrain, "nowhere"
			}),
		orgCase("Right.Subject", "a commodity for a produce right", economysim.ErrRightShape, orgAt("rights", 0, "subject"), "staple", "produce",
			func(_ *economysim.Config, m *economysim.Institutions) { m.Rights[0].Subject = "staple" }),
		orgCase("Right.Subject", "two rights with one activity, subject and scope key", economysim.ErrDuplicateID, orgAt("rights", 2, "subject"), "tool_anorth", "",
			func(_ *economysim.Config, m *economysim.Institutions) {
				m.Rights = append(m.Rights, economysim.Right{ID: "sell_again", Activity: economysim.ActivitySell, Subject: "tool_a", Region: "north"})
			}),
		orgCase("Right.Who", "executor on a sell right", economysim.ErrRightShape, orgAt("rights", 1, "who"), "executor", "principal",
			func(_ *economysim.Config, m *economysim.Institutions) { m.Rights[1].Who = economysim.WhoExecutor }),
		orgCase("Right.Who", "a value outside the three", economysim.ErrRightShape, orgAt("rights", 0, "who"), "all", "principal",
			func(_ *economysim.Config, m *economysim.Institutions) { m.Rights[0].Who = "all" }),
		orgCase("Right.Label", "over MaxLabelBytes", economysim.ErrOutOfBounds, orgAt("rights", 0, "label"), "257", "256",
			func(_ *economysim.Config, m *economysim.Institutions) {
				m.Rights[0].Label = strings.Repeat("x", economysim.MaxLabelBytes+1)
			}),
	}
}

// orgRightScopeCases are the cases of the scope key of a right.
func orgRightScopeCases() []fieldCase {
	return []fieldCase{
		orgCase("Right.Region", "both a region and a jurisdiction", economysim.ErrRightShape, orgAt("rights", 0, "region"), "northcrown", "1",
			func(_ *economysim.Config, m *economysim.Institutions) { m.Rights[0].Jurisdiction = "crown" }),
		orgCase("Right.Region", "neither a region nor a jurisdiction", economysim.ErrRightShape, orgAt("rights", 0, "region"), "", "1",
			func(_ *economysim.Config, m *economysim.Institutions) { m.Rights[0].Region = "" }),
		orgCase("Right.Region", "a region that is not defined", economysim.ErrUnknownRegion, orgAt("rights", 0, "region"), "nowhere", "",
			func(_ *economysim.Config, m *economysim.Institutions) { m.Rights[0].Region = "nowhere" }),
		orgCase("Right.Region", "an outside node", economysim.ErrOutsideNode, orgAt("rights", 0, "region"), "coast", "",
			func(c *economysim.Config, m *economysim.Institutions) {
				withOutsideNode(c)
				m.Rights[0].Region = "coast"
			}),
		orgCase("Right.Jurisdiction", "a jurisdiction that is not defined", economysim.ErrUnknownJurisdiction, orgAt("rights", 0, "jurisdiction"), "nowhere", "",
			func(_ *economysim.Config, m *economysim.Institutions) {
				m.Rights[0].Region, m.Rights[0].Jurisdiction = "", "nowhere"
			}),
		orgCase("Right.Jurisdiction", "the polity module off", economysim.ErrModuleDependency, orgAt("rights", 0, "jurisdiction"), "crown", "polity",
			func(c *economysim.Config, m *economysim.Institutions) {
				orgNoPolity(c, m)
				m.Rights[0].Region, m.Rights[0].Jurisdiction = "", "crown"
			}),
	}
}

// orgControlCases are the cases of a control.
func orgControlCases() []fieldCase {
	return []fieldCase{
		orgCase("Control.ID", "upper case letter", economysim.ErrInvalidID, orgAt("controls", 0, "id"), "Assize", "",
			func(_ *economysim.Config, m *economysim.Institutions) { m.Controls[0].ID = "Assize" }),
		orgCase("Control.ID", "the ID of a rate limit", economysim.ErrDuplicateID, orgAt("rate_limits", 0, "id"), "assize", "",
			func(_ *economysim.Config, m *economysim.Institutions) { m.RateLimits[0].ID = "assize" }),
		orgCase("Control.Commodity", "a commodity that is not defined", economysim.ErrUnknownCommodity, orgAt("controls", 0, "commodity"), "nowhere", "",
			func(_ *economysim.Config, m *economysim.Institutions) { m.Controls[0].Commodity = "nowhere" }),
		orgCase("Control.Commodity", "two controls with one commodity and scope key", economysim.ErrDuplicateID, orgAt("controls", 1, "commodity"), "staplecrown", "",
			func(_ *economysim.Config, m *economysim.Institutions) {
				m.Controls = append(m.Controls, economysim.Control{ID: "other", Commodity: "staple", Jurisdiction: "crown", CapBP: 9_000})
			}),
		orgCase("Control.Region", "both a region and a jurisdiction", economysim.ErrControlShape, orgAt("controls", 0, "region"), "northcrown", "1",
			func(_ *economysim.Config, m *economysim.Institutions) { m.Controls[0].Region = "north" }),
		orgCase("Control.Region", "neither a region nor a jurisdiction", economysim.ErrControlShape, orgAt("controls", 0, "region"), "", "1",
			func(_ *economysim.Config, m *economysim.Institutions) { m.Controls[0].Jurisdiction = "" }),
		orgCase("Control.Region", "an outside node", economysim.ErrOutsideNode, orgAt("controls", 0, "region"), "coast", "",
			func(c *economysim.Config, m *economysim.Institutions) {
				withOutsideNode(c)
				m.Controls[0].Region, m.Controls[0].Jurisdiction = "coast", ""
			}),
		orgCase("Control.Jurisdiction", "a jurisdiction that is not defined", economysim.ErrUnknownJurisdiction, orgAt("controls", 0, "jurisdiction"), "nowhere", "",
			func(_ *economysim.Config, m *economysim.Institutions) { m.Controls[0].Jurisdiction = "nowhere" }),
		orgCase("Control.Jurisdiction", "the polity module off", economysim.ErrModuleDependency, orgAt("controls", 0, "jurisdiction"), "crown", "polity",
			func(c *economysim.Config, m *economysim.Institutions) {
				orgNoPolity(c, m)
				m.Controls[0].Region, m.Controls[0].Jurisdiction = "", "crown"
			}),
		orgCase("Control.CapBP", "above MaxBasisPoints", economysim.ErrOutOfBounds, orgAt("controls", 0, "cap_bp"), "1000001", "1000000",
			func(_ *economysim.Config, m *economysim.Institutions) {
				m.Controls[0].CapBP = economysim.MaxBasisPoints + 1
			}),
		orgCase("Control.FloorBP", "above the cap", economysim.ErrControlShape, orgAt("controls", 0, "floor_bp"), "12001", "12000",
			func(_ *economysim.Config, m *economysim.Institutions) { m.Controls[0].FloorBP = 12_001 }),
		orgCase("Control.FloorBP", "above MaxBasisPoints", economysim.ErrOutOfBounds, orgAt("controls", 0, "floor_bp"), "1000001", "1000000",
			func(_ *economysim.Config, m *economysim.Institutions) {
				m.Controls[0].CapBP, m.Controls[0].FloorBP = 0, economysim.MaxBasisPoints+1
			}),
		orgCase("Control.Label", "over MaxLabelBytes", economysim.ErrOutOfBounds, orgAt("controls", 0, "label"), "257", "256",
			func(_ *economysim.Config, m *economysim.Institutions) {
				m.Controls[0].Label = strings.Repeat("x", economysim.MaxLabelBytes+1)
			}),
	}
}

// orgRateLimitCases are the cases of a rate limit.
func orgRateLimitCases() []fieldCase {
	return []fieldCase{
		orgCase("RateLimit.ID", "upper case letter", economysim.ErrInvalidID, orgAt("rate_limits", 0, "id"), "Usury", "",
			func(_ *economysim.Config, m *economysim.Institutions) { m.RateLimits[0].ID = "Usury" }),
		orgCase("RateLimit.Region", "both a region and a jurisdiction", economysim.ErrControlShape, orgAt("rate_limits", 0, "region"), "northcrown", "1",
			func(_ *economysim.Config, m *economysim.Institutions) { m.RateLimits[0].Region = "north" }),
		orgCase("RateLimit.Region", "a region that is not defined", economysim.ErrUnknownRegion, orgAt("rate_limits", 0, "region"), "nowhere", "",
			func(_ *economysim.Config, m *economysim.Institutions) {
				m.RateLimits[0].Region, m.RateLimits[0].Jurisdiction = "nowhere", ""
			}),
		orgCase("RateLimit.Jurisdiction", "a jurisdiction that is not defined", economysim.ErrUnknownJurisdiction, orgAt("rate_limits", 0, "jurisdiction"), "nowhere", "",
			func(_ *economysim.Config, m *economysim.Institutions) { m.RateLimits[0].Jurisdiction = "nowhere" }),
		orgCase("RateLimit.Jurisdiction", "the polity module off", economysim.ErrModuleDependency, orgAt("rate_limits", 0, "jurisdiction"), "crown", "polity",
			func(c *economysim.Config, m *economysim.Institutions) {
				orgNoPolity(c, m)
				m.RateLimits[0].Region, m.RateLimits[0].Jurisdiction = "", "crown"
			}),
		orgCase("RateLimit.CapBP", "above MaxCreditRateBP", economysim.ErrOutOfBounds, orgAt("rate_limits", 0, "cap_bp"), "100001", "100000",
			func(_ *economysim.Config, m *economysim.Institutions) {
				m.RateLimits[0].CapBP = economysim.MaxCreditRateBP + 1
			}),
		orgCase("RateLimit.CapBP", "the credit module off", economysim.ErrModuleDependency, orgAt("rate_limits", 0, "cap_bp"), "1200", "credit",
			func(c *economysim.Config, _ *economysim.Institutions) {
				c.Modules.Credit = nil
				for i := range c.Roles {
					c.Roles[i].Credit = nil
				}
			}),
		orgCase("RateLimit.Label", "over MaxLabelBytes", economysim.ErrOutOfBounds, orgAt("rate_limits", 0, "label"), "257", "256",
			func(_ *economysim.Config, m *economysim.Institutions) {
				m.RateLimits[0].Label = strings.Repeat("x", economysim.MaxLabelBytes+1)
			}),
	}
}

// TestValidateInstitutions_ConfigIsValid verifies that the configuration the
// institutions tests start from validates and raises none of the module's warnings, and
// that the empty module is valid: it is on, declares nothing and does nothing
// (FR-ORG-001, FR-ORG-002).
//
// Requirement: FR-ORG-001, FR-ORG-002, FR-ORG-047
//
// Guard: the field cases start from institutionsConfig; if it did not validate, each
// case would pass for the wrong reason.
func TestValidateInstitutions_ConfigIsValid(t *testing.T) {
	t.Parallel()
	c := institutionsConfig()
	if err := c.Validate(); err != nil {
		t.Fatalf("institutionsConfig().Validate() = %v, want nil", err)
	}
	if got := institutionsWarningRules(c); len(got) != 0 {
		t.Errorf("institutionsConfig().Warnings() has %v, want none of the module's", got)
	}
	empty := referenceConfig()
	empty.Modules.Institutions = &economysim.Institutions{}
	if err := empty.Validate(); err != nil {
		t.Errorf("the empty module .Validate() = %v, want nil", err)
	}
}

// TestValidateInstitutions_Errors verifies that each rule of spec-institutions.md 10.1
// has a configuration that breaks only that rule: the sentinel by errors.Is, the path of
// the offending field by its index, and the value and bound.
//
// Requirement: FR-ORG-002, FR-ORG-003, FR-ORG-004, FR-ORG-047
func TestValidateInstitutions_Errors(t *testing.T) {
	t.Parallel()
	for _, tt := range institutionsFieldCases() {
		t.Run(tt.field+" "+tt.name, func(t *testing.T) {
			t.Parallel()
			var c economysim.Config
			tt.mutate(&c)
			err := c.Validate()
			if !errors.Is(err, tt.rules[0]) {
				t.Fatalf("Validate() = %v, want %v", err, tt.rules[0])
			}
			if !errorAt(err, tt) {
				t.Errorf("Validate() = %v, want %v at %v with value %q and bound %q", err, tt.rules[0], tt.paths[0], tt.value, tt.bound)
			}
		})
	}
}

// TestValidateInstitutions_PhaseOrder verifies that a phase that finds an error ends the
// run, so the first error is the same for the same mistake: a body with an unknown role
// and a place count out of bounds reports the unknown role only, and a body with no
// role and an unknown region reports the unknown region only.
//
// Requirement: FR-ORG-047
func TestValidateInstitutions_PhaseOrder(t *testing.T) {
	t.Parallel()
	c := institutionsConfig()
	c.Modules.Institutions.Bodies[0].Roles = []string{"nowhere"}
	c.Modules.Institutions.Bodies[0].Places = -1
	if err := c.Validate(); !errors.Is(err, economysim.ErrUnknownRole) || errors.Is(err, economysim.ErrOutOfBounds) {
		t.Errorf("reference and bound mistakes: Validate() = %v, want ErrUnknownRole only", err)
	}
	c = institutionsConfig()
	c.Modules.Institutions.Bodies[0].Roles = nil
	c.Modules.Institutions.Bodies[0].Regions = []string{"nowhere"}
	if err := c.Validate(); !errors.Is(err, economysim.ErrUnknownRegion) || errors.Is(err, economysim.ErrBodyShape) {
		t.Errorf("shape and reference mistakes: Validate() = %v, want ErrUnknownRegion only", err)
	}
}

// TestSentinels_Institutions verifies that the seven sentinels of the module are
// distinct and say what they are, so errors.Is tells one from another and from the
// sentinels they sit beside.
//
// Requirement: FR-ORG-047
func TestSentinels_Institutions(t *testing.T) {
	t.Parallel()
	all := []error{
		economysim.ErrUnknownBody, economysim.ErrBodyShape, economysim.ErrBodyOverlap, economysim.ErrRightShape,
		economysim.ErrControlShape, economysim.ErrUnknownControl, economysim.ErrUnknownRight,
	}
	for i, a := range all {
		if a == nil || a.Error() == "" {
			t.Fatalf("sentinel %d is nil or has no message", i)
		}
		for j, b := range all {
			if i != j && (errors.Is(a, b) || a.Error() == b.Error()) {
				t.Errorf("sentinels %q and %q are not distinct", a, b)
			}
		}
	}
}

// TestCanonical_InstitutionsSortedByID verifies that the world's configuration holds the
// bodies, rights, controls and rate limits sorted by ID and the lists of a body sorted,
// whatever order they were written in, so the index of a body is its BodyID.
//
// Requirement: FR-ORG-003
func TestCanonical_InstitutionsSortedByID(t *testing.T) {
	t.Parallel()
	c := institutionsConfig()
	m := c.Modules.Institutions
	m.Bodies = []economysim.Body{
		{ID: "zeta", Roles: []string{"maker"}, Regions: []string{"north"}},
		{ID: "guild", Roles: []string{"grower", "cook"}, Regions: []string{"north", "farm"}, Jurisdictions: []string{"inland", "coast"}},
	} // the roles of the two bodies differ, so they overlap nowhere
	slices.Reverse(m.Rights)
	w, err := economysim.NewWorld(c)
	if err != nil {
		t.Fatalf("NewWorld() = %v, want nil: the module builds and does nothing yet", err)
	}
	got := w.Config().Modules.Institutions
	if ids := []string{got.Bodies[0].ID, got.Bodies[1].ID}; !slices.Equal(ids, []string{"guild", "zeta"}) {
		t.Errorf("bodies in the world = %v, want [guild zeta]: the BodyID is the index in ID order", ids)
	}
	if !slices.Equal(got.Bodies[0].Roles, []string{"cook", "grower"}) ||
		!slices.Equal(got.Bodies[0].Regions, []string{"farm", "north"}) ||
		!slices.Equal(got.Bodies[0].Jurisdictions, []string{"coast", "inland"}) {
		t.Errorf("lists of the first body = %v, want the roles, regions and jurisdictions sorted", got.Bodies[0])
	}
	if got.Rights[0].ID != "make_tools" || got.Rights[1].ID != "sell_tools" {
		t.Errorf("rights in the world = %v, want make_tools then sell_tools", got.Rights)
	}
	if want := economysim.WhoPrincipal; got.Rights[0].Who != want || got.Rights[1].Who != "" {
		t.Errorf("Who of the produce and sell rights = %q, %q, want %q and empty: the default is written for produce only", got.Rights[0].Who, got.Rights[1].Who, want)
	}
}

// TestValidateInstitutions_WorldIgnoresModule verifies that a world built with the
// module on and everything declared steps without error: PR A adds no behaviour.
//
// Requirement: FR-ORG-001
//
// Guard: the configuration surface lands before any reader; a world that failed to
// build or to step with the module on would block every other module's PR.
func TestValidateInstitutions_WorldIgnoresModule(t *testing.T) {
	t.Parallel()
	w, err := economysim.NewWorld(institutionsConfig())
	if err != nil {
		t.Fatalf("NewWorld() = %v, want nil", err)
	}
	for range 3 {
		if err := w.Step(); err != nil {
			t.Fatalf("Step() = %v, want nil", err)
		}
	}
}

// FuzzValidateInstitutions verifies, for any values of the module's numbers and counts,
// that Validate and Warnings never panic, that an accepted configuration is within
// every bound of spec-institutions.md 9 and that its hash is stable.
//
// Requirement: FR-ORG-047, FR-ORG-048
func FuzzValidateInstitutions(f *testing.F) {
	f.Add(int64(60), int64(2), int64(12_000), int64(8_000), int64(1_200), uint8(0))
	f.Add(int64(-1), int64(101), int64(1<<62), int64(-1), int64(-1), uint8(255))
	f.Fuzz(func(t *testing.T, places, trainees, capBP, floorBP, limit int64, shape uint8) {
		c := institutionsConfig()
		m := c.Modules.Institutions
		m.Bodies[0].Places, m.Bodies[0].MaxTrainees = places, trainees
		m.Controls[0].CapBP, m.Controls[0].FloorBP = capBP, floorBP
		m.RateLimits[0].CapBP = limit
		if shape&1 != 0 {
			m.Rights[0].Region, m.Rights[0].Jurisdiction = "", ""
		}
		if shape&2 != 0 {
			m.Bodies = append(m.Bodies, m.Bodies[0])
		}
		err := c.Validate()
		_ = c.Warnings()
		if err == nil && !orgInBounds(places, trainees, capBP, floorBP, limit) {
			t.Fatalf("Validate accepted a value outside the bounds: %+v", m)
		}
		if first, second := economysim.HashConfig(c), economysim.HashConfig(c); first != second {
			t.Fatalf("HashConfig is not stable")
		}
	})
}

// orgInBounds reports whether the numbers of the fuzz test are within the bounds of
// spec-institutions.md 9, with a floor at most the cap when a cap is set.
func orgInBounds(places, trainees, capBP, floorBP, limit int64) bool {
	for _, f := range []fieldRange{
		{"places", places, 0, economysim.MaxHouseholds}, {"max_trainees", trainees, 0, 100},
		{"cap_bp", capBP, 0, economysim.MaxBasisPoints}, {"floor_bp", floorBP, 0, economysim.MaxBasisPoints},
		{"limit", limit, 0, economysim.MaxCreditRateBP},
	} {
		if f.value < f.lo || f.value > f.hi {
			return false
		}
	}
	return capBP == 0 || floorBP <= capBP
}
