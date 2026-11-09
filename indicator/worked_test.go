package indicator_test

import (
	"cmp"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/indicator"
)

// The worked example of spec-indicators.md 18.1, built once as the window and
// the two snapshots every account test reads (plan 2-18 D573). The figures are
// those that doc/specs/models/ind/worked.out asserts. One change from the
// spec's tables, forced by the tree: the 6 units of grain in flight at the close
// are held by alpha (223 rather than 217 held and 6 in transit), because the
// snapshot has no Shipments field until plan 2-12. The sum Held + Transit is
// what every account reads, so no figure of 18.2 moves.

// sinkReasons are the goods reasons whose boundary flow is a sink; every other
// goods reason of the fixture is a faucet.
var sinkReasons = map[economysim.Reason]bool{
	economysim.ReasonRecipeInput: true, economysim.ReasonConsumption: true, economysim.ReasonWantConsumption: true,
	economysim.ReasonSpoilage: true, economysim.ReasonDisposal: true, economysim.ReasonWear: true,
	"haul_use": true, "outside_unsold": true, "outside_purchases": true, "loss": true,
}

// goodsFlow is a BoundaryFlow of goods whose kind follows the reason.
func goodsFlow(tick int64, r economysim.RegionID, reason economysim.Reason, c economysim.CommodityID, total int64) economysim.BoundaryFlow {
	kind := economysim.KindFaucet
	if sinkReasons[reason] {
		kind = economysim.KindSink
	}
	return economysim.BoundaryFlow{Tick: tick, Region: r, Reason: reason, Kind: kind, Commodity: c, Total: total}
}

// moneyFlow is a BoundaryFlow of money.
func moneyFlow(tick int64, r economysim.RegionID, reason economysim.Reason, kind economysim.Kind, total int64) economysim.BoundaryFlow {
	return economysim.BoundaryFlow{Tick: tick, Region: r, Reason: reason, Kind: kind, Commodity: economysim.NoCommodity, Total: total}
}

// setStock sets the stock of one market of a valley snapshot.
func setStock(s *economysim.Snapshot, r economysim.RegionID, c economysim.CommodityID, agents, estate, pending int64) {
	m := &s.Markets[int(r)*4+int(c)]
	m.HeldByAgents, m.HeldByEstate, m.Pending = economysim.Quantity(agents), economysim.Quantity(estate), economysim.Quantity(pending)
}

// workedWindow returns the window [10, 12) of 18.1 and the snapshots that bound it.
func workedWindow(t *testing.T) (indicator.Window, economysim.Snapshot, economysim.Snapshot) {
	t.Helper()
	open, closing := valleySnap(10, [4]int64{40, 10, 0, 0}), valleySnap(12, [4]int64{39, 11, 0, 0})
	// Stocks: grain, loaf, wood, cart (agents, estate, pending).
	for c, v := range [][3]int64{{180, 20, 0}, {40, 0, 12}, {25, 0, 0}, {3, 0, 0}} {
		setStock(&open, alpha, economysim.CommodityID(c), v[0], v[1], v[2])
	}
	for c, v := range [][3]int64{{186, 37, 0}, {50, 0, 8}, {38, 0, 0}, {5, 0, 0}} {
		setStock(&closing, alpha, economysim.CommodityID(c), v[0], v[1], v[2])
	}
	for c, v := range [][3]int64{{60, 0, 0}, {10, 0, 0}, {40, 0, 0}, {0, 0, 0}} {
		setStock(&open, beta, economysim.CommodityID(c), v[0], v[1], v[2])
	}
	for c, v := range [][3]int64{{84, 0, 0}, {4, 0, 0}, {75, 0, 0}, {0, 0, 0}} {
		setStock(&closing, beta, economysim.CommodityID(c), v[0], v[1], v[2])
	}
	tl := indicator.NewTally()
	if err := tl.Open(open); err != nil {
		t.Fatalf("Open() error = %v, want nil", err)
	}
	step(tl, 10,
		goodsFlow(10, alpha, economysim.ReasonRecipeOutput, grain, 100), goodsFlow(10, alpha, economysim.ReasonRecipeOutput, loaf, 30),
		goodsFlow(10, alpha, economysim.ReasonRecipeOutput, wood, 20), goodsFlow(10, alpha, economysim.ReasonRecipeOutput, cart, 6),
		goodsFlow(10, alpha, economysim.ReasonRecipeInput, grain, 60),
		goodsFlow(10, alpha, economysim.ReasonConsumption, grain, 10), goodsFlow(10, alpha, economysim.ReasonConsumption, loaf, 20),
		goodsFlow(10, alpha, economysim.ReasonWantConsumption, wood, 5), goodsFlow(10, alpha, economysim.ReasonSpoilage, grain, 3),
		goodsFlow(10, alpha, economysim.ReasonDisposal, wood, 2), goodsFlow(10, alpha, "haul_use", cart, 4),
		goodsFlow(10, beta, economysim.ReasonRecipeOutput, grain, 50), goodsFlow(10, beta, economysim.ReasonRecipeOutput, wood, 30),
		goodsFlow(10, beta, economysim.ReasonConsumption, grain, 35), goodsFlow(10, beta, economysim.ReasonWantConsumption, wood, 10),
		goodsFlow(10, beta, "outside_offer", wood, 20), goodsFlow(10, beta, "outside_unsold", wood, 5), goodsFlow(10, beta, "outside_purchases", loaf, 6),
		recipeRun(10, alpha, recGrow, 50, grain, 100), recipeRun(10, alpha, recBake, 30, loaf, 30),
		recipeRun(10, alpha, recCut, 10, wood, 20), recipeRun(10, alpha, recMakeCart, 6, cart, 6),
		recipeRun(10, beta, recGrow, 25, grain, 50), recipeRun(10, beta, recCut, 15, wood, 30),
	)
	// An external agent added at tick 11 brings 5 grain.
	step(tl, 11, goodsFlow(11, alpha, economysim.ReasonInitial, grain, 5))
	w, err := tl.Close(closing)
	if err != nil {
		t.Fatalf("Close() error = %v, want nil", err)
	}
	// 9 of the 10 units of grain shipped alpha to beta arrived and 1 spoiled.
	w.Arrivals = []indicator.ArrivalTally{{Origin: alpha, Destination: beta, Commodity: grain, Shipped: 10, Delivered: 9, Spoiled: 1}}
	return w, open, closing
}

// recipeRun is a RecipeExecuted of a recipe that made one commodity.
func recipeRun(tick int64, r economysim.RegionID, rec economysim.RecipeID, executions int64, c economysim.CommodityID, units economysim.Quantity) economysim.RecipeExecuted {
	return economysim.RecipeExecuted{Tick: tick, Region: r, Recipe: rec, Executions: executions, Outputs: []economysim.Amount{{Commodity: c, Quantity: units}}}
}

// valleyTable returns the window unit values of 18.1 as a price table of the
// window: alpha grain 10, loaf 25, wood 6, cart 5; beta 13, 28, 8, 6.
func valleyTable(t *testing.T, w indicator.Window, alphaPrices, betaPrices [4]economysim.Price) indicator.Prices {
	t.Helper()
	cells := append(append([]economysim.Price(nil), alphaPrices[:]...), betaPrices[:]...)
	p, err := indicator.NewPrices(w.ConfigHash, w.From, w.To, 2, 4, cells)
	if err != nil {
		t.Fatalf("NewPrices() error = %v, want nil", err)
	}
	return p
}

var (
	alphaP = [4]economysim.Price{10, 25, 6, 5}
	betaP  = [4]economysim.Price{13, 28, 8, 6}
)

// valleyConfig is the part of the configuration the value-added table reads:
// the commodities with their classes and the four recipes (18.3).
func valleyConfig() economysim.Config {
	item := func(c string, q economysim.Quantity) economysim.Item {
		return economysim.Item{Commodity: c, Quantity: q}
	}
	out := func(c string, q economysim.Quantity) economysim.Output {
		return economysim.Output{Commodity: c, Quantity: q, CostShare: 10_000}
	}
	return economysim.Config{
		Commodities: []economysim.Commodity{
			{ID: "grain", Classes: []string{"food"}}, {ID: "loaf", Classes: []string{"food"}},
			{ID: "wood", Classes: []string{"fuel"}}, {ID: "cart", Classes: []string{"carriage"}},
		},
		Recipes: []economysim.Recipe{
			{ID: "grow", Outputs: []economysim.Output{out("grain", 2)}},
			{ID: "bake", Inputs: []economysim.Item{item("grain", 2)}, Outputs: []economysim.Output{out("loaf", 1)}},
			{ID: "cut", Outputs: []economysim.Output{out("wood", 2)}},
			{ID: "make_cart", Outputs: []economysim.Output{out("cart", 1)}},
		},
	}
}

// addFlow adds a flow row to a window, keeping the rows in (region, reason,
// commodity) order as a tally does.
func addFlow(w *indicator.Window, e economysim.BoundaryFlow) {
	w.Flows = append(w.Flows, economysim.FlowTotal{Region: e.Region, Reason: e.Reason, Kind: e.Kind, Commodity: e.Commodity, Total: e.Total})
	slices.SortFunc(w.Flows, func(a, b economysim.FlowTotal) int {
		return cmp.Or(cmp.Compare(a.Region, b.Region), cmp.Compare(a.Reason, b.Reason), cmp.Compare(a.Commodity, b.Commodity))
	})
}
