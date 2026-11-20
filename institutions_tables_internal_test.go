package economysim

import (
	"slices"
	"testing"
)

// The tests of the resolved tables of the institutions module (spec-institutions.md 5.4,
// 6.1.3, 6.2): the right in force and the legal range for every (activity, subject,
// region), resolved once at construction and again when a level or a suspension is set.

// orgTablesToy is orgToy with the polity module (crown over a grouping inland, a in inland,
// b and c in crown), the credit module, and rights, controls and rate limits keyed by
// region and by jurisdiction. Grain has the reference price 100.
func orgTablesToy() Config {
	c := orgToy()
	c.Modules.Polity = &PolityConfig{
		Jurisdictions: []Jurisdiction{{ID: "crown"}, {ID: "inland", Parent: "crown"}},
		Members:       []Member{{Region: "a", Jurisdiction: "inland"}, {Region: "b", Jurisdiction: "crown"}, {Region: "c", Jurisdiction: "crown"}},
	}
	c.Modules.Credit = &Credit{MinPrincipal: 5_000, ShieldTicks: 2, MaxClaims: 1_000}
	c.Roles[orgFarmer].Credit = &CreditTerms{Borrow: &BorrowTerms{MaxRateBP: 1_500, MaxDebtTicks: 90, SlackTicks: 4}}
	c.Roles[orgSmith].Credit = &CreditTerms{Lend: &LendTerms{RateBP: 600, MaxTermTicks: 90, KeepTicks: 3, ShareBP: 5_000}}
	m := c.Modules.Institutions
	m.Rights = []Right{
		{ID: "ban_a", Activity: ActivityBuy, Subject: "grain", Region: "a"},
		{ID: "farm_b", Body: "guild", Activity: ActivityProduce, Subject: "farm", Region: "b", Who: WhoExecutor},
		{ID: "sell_crown", Body: "guild", Activity: ActivitySell, Subject: "grain", Jurisdiction: "crown"},
		{ID: "sell_inland", Body: "lodge", Activity: ActivitySell, Subject: "grain", Jurisdiction: "inland"},
		{ID: "train_c", Body: "guild", Activity: ActivityTrain, Subject: "smith", Region: "c"},
	}
	m.Controls = []Control{
		{ID: "assize", Commodity: "grain", Jurisdiction: "crown", CapBP: 12_000, FloorBP: 8_000},
		{ID: "assize_a", Commodity: "grain", Region: "a", CapBP: 10_000},
	}
	m.RateLimits = []RateLimit{{ID: "usury", Jurisdiction: "crown", CapBP: 1_200}, {ID: "usury_a", Region: "a", CapBP: 900}}
	return c
}

// orgRight returns the RightID of the right with the ID.
func orgRight(t *testing.T, w *World, id string) RightID {
	t.Helper()
	for i, r := range w.cfg.Modules.Institutions.Rights {
		if r.ID == id {
			return RightID(i)
		}
	}
	t.Fatalf("no right %q in the configuration", id)
	return 0
}

// TestRights_ResolvedAtConstruction verifies the right in force of each region: the row
// keyed by the region, else the row of the first jurisdiction of its chain, else none.
//
// Arithmetic: the chain of a is [inland, crown] and that of b and c [crown]. For the sale
// of grain the rows are sell_inland (inland) and sell_crown (crown), so a has sell_inland
// and b and c sell_crown. a's purchase of grain has the prohibition ban_a, and the
// purchase in b has no row (open). The produce right is farm_b, for b only.
//
// Requirement: FR-ORG-014
func TestRights_ResolvedAtConstruction(t *testing.T) {
	t.Parallel()
	w, _ := orgWorld(t, orgTablesToy())
	o := w.org
	grain, farm, smith := 0, 0, int(orgSmith)
	cases := []struct {
		name     string
		activity int
		subject  int
		region   RegionID
		want     string // the right in force, "" for none
	}{
		{"sale in a: the innermost jurisdiction", actSell, grain, orgA, "sell_inland"},
		{"sale in b: the outer jurisdiction", actSell, grain, orgB, "sell_crown"},
		{"sale in c", actSell, grain, orgC, "sell_crown"},
		{"purchase in a: the prohibition", actBuy, grain, orgA, "ban_a"},
		{"purchase in b: open", actBuy, grain, orgB, ""},
		{"production in b", actProduce, farm, orgB, "farm_b"},
		{"production in a: open", actProduce, farm, orgA, ""},
		{"training in c", actTrain, smith, orgC, "train_c"},
		{"training in a: open", actTrain, smith, orgA, ""},
	}
	for _, tc := range cases {
		id, ok := o.rightIn(tc.activity, tc.subject, tc.region)
		switch {
		case tc.want == "" && ok:
			t.Errorf("rightIn(%s) = %d, true, want none", tc.name, id)
		case tc.want != "" && (!ok || id != orgRight(t, w, tc.want)):
			t.Errorf("rightIn(%s) = %d, %v, want %s (%d)", tc.name, id, ok, tc.want, orgRight(t, w, tc.want))
		}
	}
	if got := o.rights.holder[orgRight(t, w, "ban_a")]; got != -1 {
		t.Errorf("holder of the prohibition = %d, want -1 (nobody)", got)
	}
	if got := o.rights.holder[orgRight(t, w, "sell_inland")]; got != 1 {
		t.Errorf("holder of sell_inland = %d, want lodge (1)", got)
	}
}

// TestResolve_SuspendedKeyOpen verifies that a suspended right leaves its key open: the
// region it covered has no right in force and the outer jurisdiction's row is not
// consulted in its place, and that lifting the suspension brings the right back.
//
// Arithmetic: a's sale of grain is sell_inland; suspended, a has none, though sell_crown
// covers the crown above it. b's sale (sell_crown) is untouched.
//
// Requirement: FR-ORG-015
func TestResolve_SuspendedKeyOpen(t *testing.T) {
	t.Parallel()
	w, _ := orgWorld(t, orgTablesToy())
	o := w.org
	inland := orgRight(t, w, "sell_inland")
	if err := w.SetRightSuspensions([]RightSuspension{{Right: inland, Suspended: true}}); err != nil {
		t.Fatalf("SetRightSuspensions() error = %v, want nil", err)
	}
	if id, ok := o.rightIn(actSell, 0, orgA); ok {
		t.Errorf("rightIn(sale in a, suspended) = %d, true, want none and no fall back to the crown", id)
	}
	if id, ok := o.rightIn(actSell, 0, orgB); !ok || id != orgRight(t, w, "sell_crown") {
		t.Errorf("rightIn(sale in b) = %d, %v, want sell_crown", id, ok)
	}
	if err := w.SetRightSuspensions([]RightSuspension{{Right: inland}}); err != nil {
		t.Fatalf("SetRightSuspensions(lift) error = %v, want nil", err)
	}
	if id, ok := o.rightIn(actSell, 0, orgA); !ok || id != inland {
		t.Errorf("rightIn(sale in a, lifted) = %d, %v, want sell_inland", id, ok)
	}
}

// TestRange_ResolvedPerRegionAndCommodity verifies the legal range of grain per region
// and the rate cap per region: the row of the region first, else the first of the chain.
//
// Arithmetic: the ControlIDs are the sorted IDs assize 0, assize_a 1, usury 2, usury_a 3.
// Grain's reference price is 100. b and c follow the crown: cap 100 * 12 000 / 10 000 =
// 120 and floor 100 * 8 000 / 10 000 = 80. a has its own row: cap 100 * 10 000 / 10 000 =
// 100 and no floor. The rate caps are 900 in a and 1 200 in b and c.
//
// Requirement: FR-ORG-028, FR-ORG-014
func TestRange_ResolvedPerRegionAndCommodity(t *testing.T) {
	t.Parallel()
	w, _ := orgWorld(t, orgTablesToy())
	o := w.org
	for _, tc := range []struct {
		region           RegionID
		capPrice, floor  int64
		id               ControlID
		rateBP, rateWant int64
	}{
		{orgA, 100, 0, 1, o.legal.rate[orgA], 900},
		{orgB, 120, 80, 0, o.legal.rate[orgB], 1_200},
		{orgC, 120, 80, 0, o.legal.rate[orgC], 1_200},
	} {
		capPrice, floor, id, ok := o.rangeOf(tc.region, 0)
		if !ok || capPrice != tc.capPrice || floor != tc.floor || id != tc.id {
			t.Errorf("rangeOf(region %d, grain) = %d, %d, %d, %v, want %d, %d, %d, true", tc.region, capPrice, floor, id, ok, tc.capPrice, tc.floor, tc.id)
		}
		if tc.rateBP != tc.rateWant {
			t.Errorf("rate cap of region %d = %d, want %d", tc.region, tc.rateBP, tc.rateWant)
		}
	}
	if want := []string{"assize", "assize_a", "usury", "usury_a"}; !slices.Equal(o.legal.ids, want) {
		t.Errorf("control IDs = %v, want %v", o.legal.ids, want)
	}
}

// TestRange_LevelsInForceFeedTheRange verifies that a level in force other than the
// configured one gives the range of the new level and is a row of the snapshot, and that
// putting it back gives the configured range and no row.
//
// Arithmetic: assize at cap 15 000 and floor 5 000: cap 100 * 15 000 / 10 000 = 150,
// floor 100 * 5 000 / 10 000 = 50.
//
// Requirement: FR-ORG-028, FR-ORG-038
func TestRange_LevelsInForceFeedTheRange(t *testing.T) {
	t.Parallel()
	w, _ := orgWorld(t, orgTablesToy())
	o := w.org
	if err := w.SetLegalLevels([]LegalLevel{{Control: 0, CapBP: 15_000, FloorBP: 5_000}}); err != nil {
		t.Fatalf("SetLegalLevels() error = %v, want nil", err)
	}
	if capPrice, floor, _, _ := o.rangeOf(orgB, 0); capPrice != 150 || floor != 50 {
		t.Errorf("rangeOf(b, grain) at the new levels = %d, %d, want 150, 50", capPrice, floor)
	}
	if rows := o.legalRows(); !slices.Equal(rows, []LegalRow{{Control: 0, CapBP: 15_000, FloorBP: 5_000}}) {
		t.Errorf("legalRows() = %v, want the one control that differs", rows)
	}
	if err := w.SetLegalLevels([]LegalLevel{{Control: 0, CapBP: 12_000, FloorBP: 8_000}}); err != nil {
		t.Fatalf("SetLegalLevels(the configured level) error = %v, want nil", err)
	}
	if capPrice, floor, _, _ := o.rangeOf(orgB, 0); capPrice != 120 || floor != 80 {
		t.Errorf("rangeOf(b, grain) back at the configured levels = %d, %d, want 120, 80", capPrice, floor)
	}
	if rows := o.legalRows(); rows != nil {
		t.Errorf("legalRows() at the configured levels = %v, want none", rows)
	}
}

// TestNames_ControlsSortedTogether verifies the names of the module: bodies and rights by
// their IDs, and the controls and the rate limits as one list in ascending ID order.
//
// Requirement: FR-ORG-044
func TestNames_ControlsSortedTogether(t *testing.T) {
	t.Parallel()
	on := orgTablesToy().canonical()
	n := namesOf(&on)
	if want := []string{"guild", "lodge"}; !slices.Equal(n.Bodies, want) {
		t.Errorf("Names.Bodies = %v, want %v", n.Bodies, want)
	}
	if want := []string{"ban_a", "farm_b", "sell_crown", "sell_inland", "train_c"}; !slices.Equal(n.Rights, want) {
		t.Errorf("Names.Rights = %v, want %v", n.Rights, want)
	}
	if want := []string{"assize", "assize_a", "usury", "usury_a"}; !slices.Equal(n.Controls, want) {
		t.Errorf("Names.Controls = %v, want %v", n.Controls, want)
	}
	off := orgToy()
	off.Modules.Institutions = nil
	if got := namesOf(&off); got.Bodies != nil || got.Rights != nil || got.Controls != nil {
		t.Errorf("names with the module off = %v %v %v, want nil", got.Bodies, got.Rights, got.Controls)
	}
}
