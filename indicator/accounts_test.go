package indicator_test

import (
	"errors"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/indicator"
)

// TestFlowSelValue_ClassAndRegion verifies the selection of flows: a nil slice
// means all, a class is turned into its commodities by the class table, and the
// value is the units at the price of the row's region.
//
// Requirement: FR-IND-029
func TestFlowSelValue_ClassAndRegion(t *testing.T) {
	w, _, _ := workedWindow(t)
	p := valleyTable(t, w, alphaP, betaP)
	food, err := indicator.NewClasses(valleyConfig()).Of("food")
	if err != nil {
		t.Fatalf("Classes.Of(food) error = %v, want nil", err)
	}
	consumed := []economysim.Reason{economysim.ReasonConsumption, economysim.ReasonWantConsumption}
	tests := []struct {
		name string
		sel  indicator.FlowSel
		want economysim.Money
	}{
		// alpha 10*10 + 20*25 + 5*6 = 630, beta 35*13 + 10*8 = 535.
		{"all regions and commodities", indicator.FlowSel{Reasons: consumed}, 1165},
		{"one region", indicator.FlowSel{Reasons: consumed, Regions: []economysim.RegionID{alpha}}, 630},
		// food = grain and loaf: alpha 100 + 500, beta 35*13 = 455.
		{"a class", indicator.FlowSel{Reasons: consumed, Commodities: food}, 1055},
		{"a class in one region", indicator.FlowSel{Reasons: consumed, Commodities: food, Regions: []economysim.RegionID{beta}}, 455},
		{"no row matches", indicator.FlowSel{Reasons: []economysim.Reason{"nothing"}}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := w.Value(tt.sel, p)
			if err != nil || got != tt.want {
				t.Errorf("Value(%+v) = %d, %v, want %d, nil", tt.sel, got, err, tt.want)
			}
		})
	}
	other := p
	other.ConfigHash = [32]byte{3}
	if _, err := w.Value(indicator.FlowSel{}, other); !errors.Is(err, indicator.ErrConfigMismatch) {
		t.Errorf("Value(table of another world) error = %v, want ErrConfigMismatch", err)
	}
}

// TestGDP_ThreeViewsAgree verifies the worked example of 18.2: value added of
// 1 200 (alpha) and 890 (beta), the expenditure components, a zero residual for
// each region, 2 090 for the realm, and at regional prices a trade valuation of
// -27 = (10 - 13) * 9; at one table for every region the interregional terms
// cancel and the realm's identity has no term.
//
// Requirement: FR-IND-017, FR-IND-030
func TestGDP_ThreeViewsAgree(t *testing.T) {
	w, open, closing := workedWindow(t)
	got, err := indicator.GDP(w, open, closing, valleyTable(t, w, alphaP, betaP))
	if err != nil {
		t.Fatalf("GDP() error = %v, want nil", err)
	}
	type terms struct{ va, c, sc, written, carriage, nx, inter, endow int64 }
	for _, tt := range []struct {
		name string
		row  indicator.GDPRow
		want terms
	}{
		// Alpha: C = 10*10 + 20*25 + 5*6; StockChange = 508 + 60 - 100; Written = 3*10 + 2*6.
		{"alpha", got.Regions[0], terms{1200, 630, 468, 42, 20, 0, 90, 50}},
		// Beta: StockChange = 24*13 - 6*28 + 35*8; NX = 6*28 - 15*8; Inter = -9*13.
		{"beta", got.Regions[1], terms{890, 535, 424, 0, 0, 48, -117, 0}},
		{"realm", got.Realm, terms{2090, 1165, 892, 42, 20, 48, -27, 50}},
	} {
		have := terms{tt.row.ValueAdded, tt.row.C, tt.row.StockChange, tt.row.Written, tt.row.Carriage, tt.row.NX, tt.row.Inter, tt.row.Endow}
		if have != tt.want {
			t.Errorf("GDP().%s = %+v, want %+v", tt.name, have, tt.want)
		}
		if tt.row.Residual != 0 {
			t.Errorf("GDP().%s.Residual = %d, want 0", tt.name, tt.row.Residual)
		}
	}
	if got.TradeValuation != -27 {
		t.Errorf("GDP().TradeValuation = %d, want -27 = (10 - 13) * 9", got.TradeValuation)
	}
	if r := got.Regions[0]; r.RelocUnits != 0 || r.RelocEvents != 0 || r.Reloc != 0 {
		t.Errorf("GDP().alpha relocation item = %d units, %d in events, value %d, want 0 in a world with no relocation", r.RelocUnits, r.RelocEvents, r.Reloc)
	}

	one, err := indicator.GDP(w, open, closing, valleyTable(t, w, alphaP, alphaP))
	if err != nil {
		t.Fatalf("GDP(one table) error = %v, want nil", err)
	}
	if one.TradeValuation != 0 || one.Realm.Inter != 0 || one.Realm.Residual != 0 {
		t.Errorf("GDP(one table) realm = TradeValuation %d Inter %d Residual %d, want 0, 0, 0", one.TradeValuation, one.Realm.Inter, one.Realm.Residual)
	}
	// At one table the realm's production equals its expenditure with no interregional term.
	r := one.Realm
	if want := r.C + r.StockChange + r.Written + r.Carriage + r.NX - r.Endow; r.ValueAdded != want {
		t.Errorf("GDP(one table) realm ValueAdded = %d, want %d = C + StockChange + Written + Carriage + NX - Endow", r.ValueAdded, want)
	}
}

// TestGDP_PendingShownApartFromCash verifies that work in progress is shown
// apart from cash: alpha's cash form is 1 300 and its accrual 1 200, the
// difference being the fall of pending output, (8 - 12) * 25 = -100.
//
// Requirement: FR-IND-030
func TestGDP_PendingShownApartFromCash(t *testing.T) {
	w, open, closing := workedWindow(t)
	got, err := indicator.GDP(w, open, closing, valleyTable(t, w, alphaP, betaP))
	if err != nil {
		t.Fatalf("GDP() error = %v, want nil", err)
	}
	if a := got.Regions[0]; a.Cash != 1300 || a.Pending != -100 || a.ValueAdded != 1200 || a.Output != 1900 || a.Intermediate != 600 {
		t.Errorf("GDP().alpha = cash %d pending %d value added %d output %d intermediate %d, want 1300, -100, 1200, 1900, 600", a.Cash, a.Pending, a.ValueAdded, a.Output, a.Intermediate)
	}
}

// TestGDP_BasisIsRetirement verifies that the net product is on the retirement
// basis and says so: wear of 2 units of cart at 5 gives a net product of
// 1 200 - 10 for alpha, with the Basis warning and field.
//
// Requirement: FR-IND-030
func TestGDP_BasisIsRetirement(t *testing.T) {
	w, open, closing := workedWindow(t)
	addFlow(&w, goodsFlow(10, alpha, economysim.ReasonWear, cart, 2))
	got, err := indicator.GDP(w, open, closing, valleyTable(t, w, alphaP, betaP))
	if err != nil {
		t.Fatalf("GDP() error = %v, want nil", err)
	}
	if got.Basis != indicator.BasisRetirement || got.Basis.String() != "retirement" {
		t.Errorf("GDP().Basis = %v, want retirement", got.Basis)
	}
	if got.Regions[0].NetProduct != 1190 || got.Realm.NetProduct != 2080 {
		t.Errorf("GDP().NetProduct = %d (alpha), %d (realm), want 1190, 2080", got.Regions[0].NetProduct, got.Realm.NetProduct)
	}
	if len(got.Warnings) == 0 || got.Warnings[0] != indicator.WarnBasis {
		t.Errorf("GDP().Warnings = %v, want WarnBasis first", got.Warnings)
	}
}

// TestGDP_DiscardsAreWrittenNotConsumed verifies that the 2 units of wood
// disposed of at alpha are in Written (2 * 6, with the 3 spoiled grain) and not
// in C, which holds the 5 consumed as a want.
//
// Requirement: FR-IND-030
func TestGDP_DiscardsAreWrittenNotConsumed(t *testing.T) {
	w, open, closing := workedWindow(t)
	for i, f := range w.Flows {
		if f.Region == alpha && f.Reason == economysim.ReasonDisposal {
			w.Flows[i].Total = 4 // two more units thrown away
		}
	}
	// The two units left stock without being made into anything: stock falls by 2.
	closing.Markets[int(alpha)*4+int(wood)].HeldByAgents -= 2
	got, err := indicator.GDP(w, open, closing, valleyTable(t, w, alphaP, betaP))
	if err != nil {
		t.Fatalf("GDP() error = %v, want nil", err)
	}
	a := got.Regions[0]
	if a.Written != 54 || a.C != 630 || a.Residual != 0 {
		t.Errorf("GDP().alpha = Written %d C %d Residual %d, want 54 (3*10 + 4*6), 630 (unchanged), 0", a.Written, a.C, a.Residual)
	}
}

// TestGDP_Misuse verifies the preconditions as errors: a table of another world
// or shape, snapshots that do not bound the window, and a snapshot row outside
// the names table.
//
// Requirement: FR-IND-006, FR-IND-009
func TestGDP_Misuse(t *testing.T) {
	w, open, closing := workedWindow(t)
	p := valleyTable(t, w, alphaP, betaP)
	other := p
	other.ConfigHash = [32]byte{3}
	short, _ := indicator.NewPrices(w.ConfigHash, 10, 12, 1, 4, make([]economysim.Price, 4))
	late := closing
	late.Tick = 13
	foreign := open
	foreign.ConfigHash = [32]byte{4}
	badMarket := closing
	badMarket.Markets = append([]economysim.MarketRow(nil), closing.Markets...)
	badMarket.Markets[0].State.Region = 9
	tests := []struct {
		name          string
		open, closing economysim.Snapshot
		p             indicator.Prices
		want          error
	}{
		{"table of another world", open, closing, other, indicator.ErrConfigMismatch},
		{"table of another shape", open, closing, short, indicator.ErrRange},
		{"closing snapshot at the wrong tick", open, late, p, indicator.ErrWindowOrder},
		{"snapshot of another world", foreign, closing, p, indicator.ErrConfigMismatch},
		{"market row outside the names", open, badMarket, p, indicator.ErrRange},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := indicator.GDP(w, tt.open, tt.closing, tt.p); !errors.Is(err, tt.want) {
				t.Errorf("GDP() error = %v, want %v", err, tt.want)
			}
		})
	}
}

// TestValueAdded_ByRecipeAndWIP verifies 18.3: the jobs completed in the window
// add up to 1 300 at alpha (grow 1 000, bake 30*25 - 30*2*10 = 150, cut 120,
// make_cart 30), by class food 1 150, fuel 120 and carriage 30; the WIP row is
// +100 = 10 * (60 - 60) - 25 * (8 - 12); a class has no value added from output
// and input.
//
// Requirement: FR-IND-031
func TestValueAdded_ByRecipeAndWIP(t *testing.T) {
	w, open, closing := workedWindow(t)
	cfg := valleyConfig()
	got, err := indicator.ValueAdded(w, cfg, open, closing, valleyTable(t, w, alphaP, betaP), indicator.NewClasses(cfg))
	if err != nil {
		t.Fatalf("ValueAdded() error = %v, want nil", err)
	}
	wantRecipes := map[economysim.RecipeID]int64{recGrow: 1000, recBake: 150, recCut: 120, recMakeCart: 30}
	for _, r := range got.ByRecipe {
		if r.Region == alpha && r.Value != wantRecipes[r.Recipe] {
			t.Errorf("ValueAdded().ByRecipe[alpha, %d] = %d, want %d", r.Recipe, r.Value, wantRecipes[r.Recipe])
		}
	}
	class := func(rows []indicator.ClassRow, r economysim.RegionID, name string) int64 {
		for _, c := range rows {
			if c.Region == r && c.Class == name {
				return c.Value
			}
		}
		t.Errorf("no class row for region %d class %q", r, name)
		return 0
	}
	for name, want := range map[string]int64{"": 1300, "food": 1150, "fuel": 120, "carriage": 30} {
		if v := class(got.ByClass, alpha, name); v != want {
			t.Errorf("ValueAdded().ByClass[alpha, %q] = %d, want %d", name, v, want)
		}
	}
	// Output by class is gross: alpha food 100*10 + 30*25; intermediate food 60*10.
	if v := class(got.Output, alpha, "food"); v != 1750 {
		t.Errorf("ValueAdded().Output[alpha, food] = %d, want 1750", v)
	}
	if v := class(got.Intermediate, alpha, "food"); v != 600 {
		t.Errorf("ValueAdded().Intermediate[alpha, food] = %d, want 600", v)
	}
	a, b := got.Regions[0], got.Regions[1]
	if a.Completed != 1300 || a.Accrual != 1200 || a.WIP != 100 || a.Mismatch != 0 {
		t.Errorf("ValueAdded().alpha = %+v, want completed 1300, accrual 1200, WIP 100, mismatch 0", a)
	}
	if b.Completed != 890 || b.WIP != 0 || b.Mismatch != 0 {
		t.Errorf("ValueAdded().beta = %+v, want completed 890 = accrual, WIP 0", b)
	}
}

// TestValueAdded_ToolRecipeOverstates verifies D575 item 6: the by-recipe value
// of a recipe that uses a tool does not net the tool's wear, so it overstates
// the net product by the wear retired. The overstatement is the wear at the
// table, here 3 units of cart at 5, plus the WIP row.
//
// Requirement: FR-IND-031
func TestValueAdded_ToolRecipeOverstates(t *testing.T) {
	w, open, closing := workedWindow(t)
	addFlow(&w, goodsFlow(10, alpha, economysim.ReasonWear, cart, 3))
	closing.Markets[int(alpha)*4+int(cart)].HeldByAgents -= 3
	cfg := valleyConfig()
	p := valleyTable(t, w, alphaP, betaP)
	va, err := indicator.ValueAdded(w, cfg, open, closing, p, indicator.NewClasses(cfg))
	if err != nil {
		t.Fatalf("ValueAdded() error = %v, want nil", err)
	}
	gdp, err := indicator.GDP(w, open, closing, p)
	if err != nil {
		t.Fatalf("GDP() error = %v, want nil", err)
	}
	a := va.Regions[0]
	if got, want := a.Completed-gdp.Regions[0].NetProduct, a.WIP+15; got != want {
		t.Errorf("by-recipe total less the net product = %d, want %d = WIP %d + wear 3*5", got, want, a.WIP)
	}
}

// TestValueAdded_Misuse verifies that a configuration of another shape than the
// window is refused.
//
// Requirement: FR-IND-006
func TestValueAdded_Misuse(t *testing.T) {
	w, open, closing := workedWindow(t)
	cfg := valleyConfig()
	cfg.Recipes = cfg.Recipes[:2]
	if _, err := indicator.ValueAdded(w, cfg, open, closing, valleyTable(t, w, alphaP, betaP), indicator.Classes{}); !errors.Is(err, indicator.ErrConfigMismatch) {
		t.Errorf("ValueAdded(config with fewer recipes) error = %v, want ErrConfigMismatch", err)
	}
}

// TestGDP_RelocatedGoodsAreAReconcilingItem verifies 7.4.1 (b) and (c): 6 units
// of grain that left alpha with a relocated account are the relocation item,
// -60 at alpha and +78 at beta, equal in units to the AgentRelocated events, so
// each region's identity closes with it; at one table the realm's residual is 0,
// and at regional prices it is the flagged revaluation of the moved goods. The
// construction bucket is a memo line of the goods at the table, in no flow.
//
// Requirement: FR-IND-019, FR-IND-007
func TestGDP_RelocatedGoodsAreAReconcilingItem(t *testing.T) {
	w, open, closing := workedWindow(t)
	closing.Markets[int(alpha)*4+int(grain)].HeldByAgents -= 6
	closing.Markets[int(beta)*4+int(grain)].HeldByAgents += 6
	w.Relocations = []indicator.RelocTally{{From: alpha, To: beta, Units: 6, Weight: 1}}
	w.Construction = []economysim.FlowTotal{{Region: alpha, Reason: economysim.ReasonInitial, Kind: economysim.KindFaucet, Commodity: grain, Total: 200}}
	got, err := indicator.GDP(w, open, closing, valleyTable(t, w, alphaP, betaP))
	if err != nil {
		t.Fatalf("GDP() error = %v, want nil", err)
	}
	a, b := got.Regions[0], got.Regions[1]
	if a.RelocUnits != -6 || a.RelocEvents != -6 || a.Reloc != -60 || b.RelocUnits != 6 || b.RelocEvents != 6 || b.Reloc != 78 {
		t.Errorf("GDP() relocation item = alpha %d units (%d events) %d, beta %d units (%d events) %d, want -6 (-6) -60, 6 (6) 78",
			a.RelocUnits, a.RelocEvents, a.Reloc, b.RelocUnits, b.RelocEvents, b.Reloc)
	}
	if a.Residual != 0 || b.Residual != 0 || a.ValueAdded != 1200 {
		t.Errorf("GDP() residuals = %d, %d and alpha value added %d, want 0, 0, 1200: the item closes each identity", a.Residual, b.Residual, a.ValueAdded)
	}
	// At regional prices the realm's residual is the revaluation: -(sum of Reloc) = -18.
	if got.Realm.Residual != -18 {
		t.Errorf("GDP().Realm.Residual = %d, want -18 at regional prices", got.Realm.Residual)
	}
	if a.Construction != 2000 {
		t.Errorf("GDP().alpha.Construction = %d, want 2000 (200 grain at 10, a memo line)", a.Construction)
	}
	one, err := indicator.GDP(w, open, closing, valleyTable(t, w, alphaP, alphaP))
	if err != nil || one.Realm.Residual != 0 || one.Realm.Reloc != 0 {
		t.Errorf("GDP(one table) = residual %d reloc %d err %v, want 0, 0, nil: the moved goods cancel", one.Realm.Residual, one.Realm.Reloc, err)
	}
}

// TestValueAdded_ConfigOfAnotherWorldIsRejected verifies that a configuration
// with the right counts but another hash is ErrConfigMismatch, as BasketLevel
// and RealWage reject it.
//
// Guard: review round 4. Only the counts were compared, so a pack with the same
// shape read its cost shares and class sets from the wrong commodities.
//
// Requirement: FR-IND-031
func TestValueAdded_ConfigOfAnotherWorldIsRejected(t *testing.T) {
	t.Parallel()
	w, open, closing := workedWindow(t)
	cfg := valleyConfig()
	cfg.Commodities[0].ReferencePrice++ // same shape, another hash
	if _, err := indicator.ValueAdded(w, cfg, open, closing, valleyTable(t, w, alphaP, betaP), indicator.NewClasses(cfg)); !errors.Is(err, indicator.ErrConfigMismatch) {
		t.Errorf("ValueAdded(config of another world) error = %v, want ErrConfigMismatch", err)
	}
}
