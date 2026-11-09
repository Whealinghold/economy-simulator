package indicator_test

import (
	"errors"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/indicator"
)

// basketConfig is the part of the configuration the basket of 18.5 reads: the
// upkeep of 2 grain and 1 loaf, the reference prices alpha (8, 24) and beta
// (12, 26) (a region's override replaces the commodity's price), and two regions.
func basketConfig() economysim.Config {
	return economysim.Config{
		World: economysim.WorldParams{Upkeep: []economysim.Item{{Commodity: "grain", Quantity: 2}, {Commodity: "loaf", Quantity: 1}}},
		Commodities: []economysim.Commodity{
			{ID: "grain", ReferencePrice: 8}, {ID: "loaf", ReferencePrice: 24}, {ID: "wood", ReferencePrice: 6}, {ID: "cart", ReferencePrice: 5},
		},
		Regions:      []economysim.Region{{ID: "alpha"}, {ID: "beta"}},
		RegionPrices: []economysim.RegionPrice{{Region: "beta", Commodity: "grain", Price: 12}, {Region: "beta", Commodity: "loaf", Price: 26}},
	}
}

// priced returns a valley snapshot of the configuration's hash whose markets
// publish the prices given for grain and loaf at alpha and beta (age in ticks).
func priced(cfg economysim.Config, alphaGrain, alphaLoaf, betaGrain, betaLoaf economysim.Price, age int64) economysim.Snapshot {
	s := valleySnap(12, [4]int64{})
	s.ConfigHash = economysim.HashConfig(cfg)
	for i, p := range []economysim.Price{alphaGrain, alphaLoaf, 6, 5, betaGrain, betaLoaf, 8, 6} {
		s.Markets[i].State.Price = p
	}
	s.Markets[int(alpha)*4+int(loaf)].State.Age = age
	return s
}

// TestBasketLevel_ExactlyTenThousandAtReference verifies 18.5: the basket of 2
// grain and 1 loaf costs 45 at alpha and 54 at beta at the published prices
// against a computed reference cost of 40 and 50, so the levels are 11 250 and
// 10 800 and beta against alpha is rdiv(10 000 * 54 * 40, 50 * 45) = 9 600 from
// the exact sums; at the reference prices the level is exactly 10 000. The basket
// is fixed, and a good whose price is carried forward is priced stale.
//
// Requirement: FR-IND-033
func TestBasketLevel_ExactlyTenThousandAtReference(t *testing.T) {
	cfg := basketConfig()
	got, err := indicator.BasketLevel(cfg, priced(cfg, 10, 25, 13, 28, 0))
	if err != nil {
		t.Fatalf("BasketLevel() error = %v, want nil", err)
	}
	a, b := got.Rows[0], got.Rows[1]
	if a.Cost != 45 || a.RefCost != 40 || a.Level.Value != 11250 || b.Cost != 54 || b.RefCost != 50 || b.Level.Value != 10800 {
		t.Errorf("BasketLevel() = alpha %+v, beta %+v, want 45/40 -> 11 250 and 54/50 -> 10 800", a, b)
	}
	if r, err := got.Ratio(beta, alpha); err != nil || r.Value != 9600 || !r.OK {
		t.Errorf("BasketLevel().Ratio(beta, alpha) = %+v, %v, want 9 600", r, err)
	}
	if r, err := got.Ratio(beta, 7); err != nil || r.OK {
		t.Errorf("BasketLevel().Ratio(beta, unknown) = %+v, %v, want undefined", r, err)
	}

	atRef, err := indicator.BasketLevel(cfg, priced(cfg, 8, 24, 12, 26, 0))
	if err != nil || atRef.Rows[0].Level.Value != 10_000 || atRef.Rows[1].Level.Value != 10_000 {
		t.Errorf("BasketLevel(reference prices) = %+v, %v, want exactly 10 000 in both regions", atRef.Rows, err)
	}

	// Must not hide: the loaf at alpha did not trade (age 1): 25 of the basket cost 45 is stale.
	stale, err := indicator.BasketLevel(cfg, priced(cfg, 10, 25, 13, 28, 1))
	if err != nil || !stale.Rows[0].StaleOK || stale.Rows[0].Stale != 5556 || stale.Rows[1].Stale != 0 {
		t.Errorf("BasketLevel(aged loaf) stale = alpha %d (%v) beta %d, %v, want 5 556 (25/45), 0", stale.Rows[0].Stale, stale.Rows[0].StaleOK, stale.Rows[1].Stale, err)
	}
	if !slices.Equal(stale.Warnings, []indicator.Warning{indicator.WarnStale}) || len(got.Warnings) != 0 {
		t.Errorf("BasketLevel() warnings = %v (aged loaf), %v (fresh), want [WarnStale], none", stale.Warnings, got.Warnings)
	}
	// A region with an empty basket (cost 0) has no price level against another.
	empty := indicator.BasketResult{Rows: []indicator.BasketRow{{Region: alpha, RefCost: 40}, {Region: beta, Cost: 54, RefCost: 50}}}
	if r, err := empty.Ratio(alpha, beta); err != nil || r.OK {
		t.Errorf("BasketResult.Ratio(zero cost, beta) = %+v, %v, want undefined", r, err)
	}

	other := priced(cfg, 10, 25, 13, 28, 0)
	other.ConfigHash = [32]byte{1}
	if _, err := indicator.BasketLevel(cfg, other); !errors.Is(err, indicator.ErrConfigMismatch) {
		t.Errorf("BasketLevel(snapshot of another world) error = %v, want ErrConfigMismatch", err)
	}
	cfg.World.Upkeep = append(cfg.World.Upkeep, economysim.Item{Commodity: "gold", Quantity: 1})
	if _, err := indicator.BasketLevel(cfg, priced(cfg, 10, 25, 13, 28, 0)); !errors.Is(err, indicator.ErrRange) {
		t.Errorf("BasketLevel(upkeep of an unknown commodity) error = %v, want ErrRange", err)
	}
}

// TestDeliveredPrice_HiredAndSelfCarried verifies 18.5: a hired shipment that
// redeemed 4 haul units (cart, 5 at alpha) and dispatched 10 units has a freight
// of rdiv(20, 10) = 2 per unit and a delivered price of 10 + 2 = 12; a lane of a
// mode with no haul commodity (the traveller carried it) has Hired false and
// Freight 0, its cost being no price.
//
// Requirement: FR-IND-034
func TestDeliveredPrice_HiredAndSelfCarried(t *testing.T) {
	w, _, _ := workedWindow(t)
	w.Dispatches = []indicator.DispatchTally{
		{Origin: alpha, Destination: beta, Mode: 0, Shipments: 1, Hired: 1, HaulUnits: 4, Units: 10},
		{Origin: alpha, Destination: beta, Mode: 1, Shipments: 1, Units: 3},
		{Origin: alpha, Destination: beta, Mode: 2, Shipments: 1, Hired: 1, HaulUnits: 4},
	}
	p := valleyTable(t, w, alphaP, betaP)
	got, err := indicator.DeliveredPrice(w, []economysim.CommodityID{cart, economysim.NoCommodity, cart}, p)
	if err != nil {
		t.Fatalf("DeliveredPrice() error = %v, want nil", err)
	}
	want := []indicator.Lane{
		{Origin: alpha, Destination: beta, Mode: 0, Commodity: grain, Delivered: 12, Freight: 2, HaulValue: 20, Units: 10, Hired: true, FreightOK: true},
		{Origin: alpha, Destination: beta, Mode: 1, Commodity: grain, Delivered: 10, Units: 3},
		{Origin: alpha, Destination: beta, Mode: 2, Commodity: grain, Delivered: 10, HaulValue: 20, Hired: true},
	}
	if !slices.Equal(got.Lanes, want) {
		t.Errorf("DeliveredPrice().Lanes = %+v, want %+v", got.Lanes, want)
	}
	other := p
	other.ConfigHash = [32]byte{2}
	if _, err := indicator.DeliveredPrice(w, nil, other); !errors.Is(err, indicator.ErrConfigMismatch) {
		t.Errorf("DeliveredPrice(table of another world) error = %v, want ErrConfigMismatch", err)
	}
}

// graphOf builds the region graph of the given regions and directed edges.
func graphOf(t *testing.T, regions []string, edges ...economysim.Edge) economysim.Graph {
	t.Helper()
	cfg := economysim.Config{Edges: edges}
	for _, id := range regions {
		cfg.Regions = append(cfg.Regions, economysim.Region{ID: id})
	}
	g, err := economysim.NewGraph(cfg)
	if err != nil {
		t.Fatalf("NewGraph(%v) error = %v, want nil", regions, err)
	}
	return g
}

// TestPriceGap_SpreadFromExactSums verifies 18.5: grain at 10 and 13 has a
// spread of rdiv(2 * 10 000 * 3, 13 + 10) = 2 609 from the exact sums (a share
// of the rounded mean 12 would give 2 500), a local gap of 3 and a gap after
// freight of 13 - 12 = 1, rdiv(10 000, 12) = 833 of the delivered price; a pair
// with no route is a row with Connected false that the average leaves out, with
// the NoRoute warning.
//
// Requirement: FR-IND-035
func TestPriceGap_SpreadFromExactSums(t *testing.T) {
	w, _, _ := workedWindow(t)
	w.Dispatches = []indicator.DispatchTally{{Origin: alpha, Destination: beta, Mode: 0, Shipments: 1, Hired: 1, HaulUnits: 4, Units: 10}}
	p := valleyTable(t, w, [4]economysim.Price{10, 0, 0, 0}, [4]economysim.Price{13, 0, 0, 0})
	dl, err := indicator.DeliveredPrice(w, []economysim.CommodityID{cart}, p)
	if err != nil {
		t.Fatalf("DeliveredPrice() error = %v, want nil", err)
	}
	// The haul commodity has no price in this table: the freight is 0 and the delivered price 10.
	if dl.Lanes[0].Delivered != 10 {
		t.Fatalf("DeliveredPrice().Lanes[0].Delivered = %d, want 10 (cart is unpriced in this table)", dl.Lanes[0].Delivered)
	}
	p = valleyTable(t, w, [4]economysim.Price{10, 0, 0, 5}, [4]economysim.Price{13, 0, 0, 0})
	dl, _ = indicator.DeliveredPrice(w, []economysim.CommodityID{cart}, p)
	linked := graphOf(t, []string{"alpha", "beta"}, economysim.Edge{From: "alpha", To: "beta", Distance: 1}, economysim.Edge{From: "beta", To: "alpha", Distance: 1})
	got, err := indicator.PriceGap(w, p, dl, linked)
	if err != nil {
		t.Fatalf("PriceGap() error = %v, want nil", err)
	}
	var fwd indicator.GapRow
	for _, r := range got.Rows {
		if r.Origin == alpha && r.Destination == beta && r.Commodity == grain {
			fwd = r
		}
	}
	if fwd.Spread.Value != 2609 || fwd.GapLocal != 3 || fwd.GapAfterFreight != 1 || fwd.GapAfterFreightBP.Value != 833 || !fwd.Connected || !fwd.AfterFreightOK {
		t.Errorf("PriceGap() alpha->beta grain = %+v, want spread 2 609, gap 3, after freight 1 (833 bp), connected", fwd)
	}
	if got.MeanSpread.Value != 2609 || got.Pairs != 2 || got.Unconnected != 0 {
		t.Errorf("PriceGap() mean %+v over %d pairs, %d unconnected, want 2 609 over 2, 0", got.MeanSpread, got.Pairs, got.Unconnected)
	}

	// Must not hide: a closed road. Two regions with no edge are two components.
	cut, err := indicator.PriceGap(w, p, dl, graphOf(t, []string{"alpha", "beta"}))
	if err != nil {
		t.Fatalf("PriceGap(cut) error = %v, want nil", err)
	}
	if cut.Unconnected != 2 || cut.Pairs != 0 || cut.MeanSpread.OK || cut.Rows[0].Connected || !slices.Contains(cut.Warnings, indicator.WarnNoRoute) {
		t.Errorf("PriceGap(cut) = unconnected %d pairs %d mean %+v warnings %v, want 2, 0, undefined, NoRoute", cut.Unconnected, cut.Pairs, cut.MeanSpread, cut.Warnings)
	}
	if _, err := indicator.PriceGap(w, p, dl, graphOf(t, []string{"alpha"})); !errors.Is(err, indicator.ErrRange) {
		t.Errorf("PriceGap(graph of another size) error = %v, want ErrRange", err)
	}
}

// TestPriceGap_UnconnectedPairIsLeftOutOfTheAverage verifies that with three
// regions, one cut off, the mean spread is that of the connected pair alone.
//
// Requirement: FR-IND-035
func TestPriceGap_UnconnectedPairIsLeftOutOfTheAverage(t *testing.T) {
	w := indicator.Window{ConfigHash: valleyHash, Regions: 3, Commodities: 1, Regional: make([]indicator.RegionTally, 3)}
	p, err := indicator.NewPrices(valleyHash, 0, 0, 3, 1, []economysim.Price{10, 13, 40})
	if err != nil {
		t.Fatalf("NewPrices() error = %v, want nil", err)
	}
	g := graphOf(t, []string{"a", "b", "c"}, economysim.Edge{From: "a", To: "b", Distance: 1}, economysim.Edge{From: "b", To: "a", Distance: 1})
	got, err := indicator.PriceGap(w, p, indicator.DeliveredResult{}, g)
	if err != nil {
		t.Fatalf("PriceGap() error = %v, want nil", err)
	}
	if got.Pairs != 2 || got.Unconnected != 4 || got.MeanSpread.Value != 2609 {
		t.Errorf("PriceGap() pairs %d unconnected %d mean %+v, want 2, 4, 2 609: the far region is out of the average", got.Pairs, got.Unconnected, got.MeanSpread)
	}
}

// windowWithMarkets returns the window [10, 12) in which the alpha grain market
// cleared at 10 for 30 and at 11 for 20 (18.1), with the beliefs, the money and
// the shortfalls of 18.5 to 18.7.
func windowWithMarkets(t *testing.T) (indicator.Window, economysim.Snapshot) {
	t.Helper()
	tl := indicator.NewTally()
	openValley(t, tl, 10, [4]int64{40, 0, 0, 0})
	belief := func(tick int64, out, floor, ceil int64, low, high, mean economysim.Price) economysim.Event {
		return economysim.BeliefsUpdated{Tick: tick, Region: alpha, Commodity: grain, Outcomes: out, AtFloor: floor, AtCeiling: ceil, LowCentre: low, HighCentre: high, MeanCentre: mean}
	}
	step(tl, 10, marketDay(10, alpha, grain, 10, 30), marketDay(10, alpha, loaf, 25, 80), marketDay(10, alpha, wood, 6, 80), marketDay(10, alpha, cart, 5, 0),
		belief(10, 30, 4, 1, 8, 12, 10), economysim.UpkeepShortfall{Tick: 10, Region: alpha, Role: grower, Unfed: 4},
		// Three births done and one bankruptcy at the end of the tick: 40 households at the start of tick 10, 42 at tick 11.
		economysim.WeightChanged{Tick: 10, Region: alpha, Role: grower, From: 4, To: 7},
		economysim.AgentBankrupted{Tick: 10, Region: alpha, Role: grower, Weight: 1})
	step(tl, 11, marketDay(11, alpha, grain, 11, 20), marketDay(11, alpha, loaf, 25, 0), marketDay(11, alpha, wood, 6, 0), marketDay(11, alpha, cart, 5, 0),
		belief(11, 20, 2, 0, 9, 12, 10), economysim.UpkeepShortfall{Tick: 11, Region: alpha, Role: grower, Unfed: 6})
	closing := valleySnap(12, [4]int64{41, 0, 0, 0})
	closing.Regions[0].MoneyHouseholds, closing.Regions[0].MoneyPool = 5200, 800
	w, err := tl.Close(closing)
	if err != nil {
		t.Fatalf("Close() error = %v, want nil", err)
	}
	return w, closing
}

// TestPriceRange_Pinned_Dispersion_Offset verifies the market figures of 18.5
// and 18.7 for the alpha grain market: the range of the published prices 10 and
// 11 is rdiv(10 000 * 11, 10) = 11 000; pinned is rdiv(10 000 * (6 + 1), 50) =
// 1 400; the dispersion of the last tick is rdiv(10 000 * (12 - 9), 10) = 3 000
// and the window's lowest is the same (the first tick's 4 000 is higher); the
// offset from a price of 9 is rdiv(10 000 * (10 - 9), 9) = 1 111, beside the unmet
// need rdiv(10 000 * (4 + 6), 40 + 42) = 1 220 of the running households.
//
// Requirement: FR-IND-046, FR-IND-047, FR-IND-048, FR-IND-049
func TestPriceRange_Pinned_Dispersion_Offset(t *testing.T) {
	w, _ := windowWithMarkets(t)
	sel := indicator.CellSel{Regions: []economysim.RegionID{alpha}, Commodities: []economysim.CommodityID{grain}}

	rng, err := indicator.PriceRange(w, sel)
	if err != nil || len(rng.Rows) != 1 || rng.Rows[0].Low != 10 || rng.Rows[0].High != 11 || rng.Rows[0].Range.Value != 11000 || rng.Rows[0].Active != 2 {
		t.Errorf("PriceRange() = %+v, %v, want low 10, high 11, range 11 000, 2 active ticks", rng.Rows, err)
	}
	// Must not hide: the cart market published a price and never traded (Active 0).
	all, err := indicator.PriceRange(w, indicator.CellSel{Regions: []economysim.RegionID{alpha}, Commodities: []economysim.CommodityID{cart}})
	if err != nil || all.Rows[0].Active != 0 || all.Rows[0].Range.Value != 10_000 || !slices.Contains(all.Warnings, indicator.WarnThin) {
		t.Errorf("PriceRange(cart) = %+v warnings %v, %v, want range 10 000 with no active tick and the Thin warning", all.Rows, all.Warnings, err)
	}

	pin, err := indicator.Pinned(w, sel)
	if err != nil || len(pin.Rows) != 1 || pin.Rows[0].Share.Value != 1400 || pin.Rows[0].Outcomes != 50 || pin.Rows[0].AtFloor != 6 || pin.Rows[0].AtCeiling != 1 {
		t.Errorf("Pinned() = %+v, %v, want share 1 400 of 50 outcomes (6 floor, 1 ceiling)", pin.Rows, err)
	}

	disp, err := indicator.Dispersion(w, sel)
	if err != nil || len(disp.Rows) != 1 || disp.Rows[0].Spread.Value != 3000 || disp.Rows[0].Lowest.Value != 3000 || disp.Rows[0].MeanCentre != 10 {
		t.Errorf("Dispersion() = %+v, %v, want spread 3 000 and lowest 3 000", disp.Rows, err)
	}

	p, err := indicator.PricesOfWindow(w)
	if err != nil {
		t.Fatalf("PricesOfWindow() error = %v, want nil", err)
	}
	ref := valleyTable(t, w, [4]economysim.Price{9, 0, 0, 0}, [4]economysim.Price{0, 0, 0, 0})
	off, err := indicator.Offset(w, p, ref, sel)
	if err != nil || len(off.Rows) != 1 || off.Rows[0].Diff.Value != 1111 || !off.UnmetOK || off.Unmet != 1220 {
		t.Errorf("Offset() = %+v unmet %d (%v), %v, want diff 1 111 beside the unmet need 1 220", off.Rows, off.Unmet, off.UnmetOK, err)
	}
	// Must not hide: a reference at or below 0 is undefined, not a huge offset.
	noRef := valleyTable(t, w, [4]economysim.Price{0, 0, 0, 0}, [4]economysim.Price{0, 0, 0, 0})
	if u, err := indicator.Offset(w, p, noRef, sel); err != nil || u.Rows[0].Diff.OK || u.Rows[0].Diff.Value != 0 {
		t.Errorf("Offset(reference 0) = %+v, %v, want the diff undefined", u.Rows, err)
	}
	other := ref
	other.ConfigHash = [32]byte{8}
	if _, err := indicator.Offset(w, p, other, sel); !errors.Is(err, indicator.ErrConfigMismatch) {
		t.Errorf("Offset(reference of another world) error = %v, want ErrConfigMismatch", err)
	}

	// Must not hide: no belief outcome is undefined, not 0.
	empty, err := indicator.Pinned(w, indicator.CellSel{Regions: []economysim.RegionID{beta}})
	if err != nil || len(empty.Rows) != 0 {
		t.Errorf("Pinned(beta) = %+v, %v, want no row for a market with no outcome", empty.Rows, err)
	}
	w.Beliefs[int(alpha)*4+int(grain)].Outcomes = 0
	if u, _ := indicator.Pinned(w, sel); u.Rows[0].Share.OK || u.Rows[0].Share.Value != 0 {
		t.Errorf("Pinned(no outcome) = %+v, want the share undefined", u.Rows[0].Share)
	}
	w.Beliefs[int(alpha)*4+int(grain)].MeanCentre = 0
	if u, _ := indicator.Dispersion(w, sel); u.Rows[0].Spread.OK {
		t.Errorf("Dispersion(mean centre 0) = %+v, want the spread undefined", u.Rows[0].Spread)
	}
}

// TestTurnover_VelocityAndActive verifies 18.6: a trade of 3 000 over money of
// 6 000 (5 200 in households and 800 in the pool; the spec's escrow of 200 is
// folded into households until plan 2-15) has a velocity of rdiv(10 000 * 3 000,
// 6 000) = 5 000 per window and, annualised from the exact sums, rdiv(10 000 *
// 3 000 * 365, 6 000 * 2) = 912 500; a market that never traded is flagged.
//
// Requirement: FR-IND-037
func TestTurnover_VelocityAndActive(t *testing.T) {
	w, closing := windowWithMarkets(t)
	sel := indicator.CellSel{Regions: []economysim.RegionID{alpha}}
	got, err := indicator.Turnover(w, closing, 365, sel)
	if err != nil {
		t.Fatalf("Turnover() error = %v, want nil", err)
	}
	if got.Value != 3000 || got.Money != 6000 || got.Velocity.Value != 5000 || got.Annualised.Value != 912_500 {
		t.Errorf("Turnover() = value %d money %d velocity %d annualised %d, want 3 000, 6 000, 5 000, 912 500", got.Value, got.Money, got.Velocity.Value, got.Annualised.Value)
	}
	if !slices.Contains(got.Warnings, indicator.WarnThin) {
		t.Errorf("Turnover().Warnings = %v, want Thin: the cart market never traded", got.Warnings)
	}
	for _, r := range got.Rows {
		if r.Commodity == grain && (r.Active != 2 || r.Ticks != 2) {
			t.Errorf("Turnover() grain row = %+v, want 2 active ticks of 2", r)
		}
		if r.Commodity == cart && (r.Active != 0 || r.Ticks != 2) {
			t.Errorf("Turnover() cart row = %+v, want 0 active ticks of 2", r)
		}
	}
	for name, tt := range map[string]struct {
		w       indicator.Window
		c       economysim.Snapshot
		perYear int64
		want    error
	}{
		"no ticks in a year":         {w, closing, 0, indicator.ErrRange},
		"snapshot at the wrong tick": {w, valleySnap(5, [4]int64{}), 365, indicator.ErrWindowOrder},
	} {
		if _, err := indicator.Turnover(tt.w, tt.c, tt.perYear, sel); !errors.Is(err, tt.want) {
			t.Errorf("Turnover(%s) error = %v, want %v", name, err, tt.want)
		}
	}
	short := closing
	short.Regions = short.Regions[:1]
	if _, err := indicator.Turnover(w, short, 365, sel); !errors.Is(err, indicator.ErrRange) {
		t.Errorf("Turnover(snapshot with one region) error = %v, want ErrRange", err)
	}
}

// TestTurnover_VolumeIsUnitsNotValue verifies that the volume is the units
// traded (30 + 20 grain, 80 loaf and 80 wood: 210), not the value (3 000): the
// volume class R of the stylised facts reads units (N-201).
//
// Requirement: FR-IND-037
func TestTurnover_VolumeIsUnitsNotValue(t *testing.T) {
	w, closing := windowWithMarkets(t)
	got, err := indicator.Turnover(w, closing, 365, indicator.CellSel{Regions: []economysim.RegionID{alpha}})
	if err != nil || got.Volume != 210 || got.Value != 3000 {
		t.Errorf("Turnover() = volume %d value %d, %v, want 210 units and 3 000 money", got.Volume, got.Value, err)
	}
}

// TestStockCover_UndefinedWhereNothingIsConsumed verifies 18.6: 217 units of
// grain held, 10 consumed over 2 ticks, give a cover of rdiv(10 000 * 217 * 2,
// 10) = 434 000 (43.4 ticks) and, with 37 in the estate, an unsold share of
// rdiv(10 000 * 37, 217) = 1 705; a commodity with no consumption has its cover
// undefined and not infinite, and an empty pantry (cover 0) is a different
// number from a glut.
//
// Requirement: FR-IND-044
func TestStockCover_UndefinedWhereNothingIsConsumed(t *testing.T) {
	w, _, closing := workedWindow(t)
	setStock(&closing, alpha, grain, 180, 37, 0)
	got, err := indicator.StockCover(w, closing, indicator.CellSel{Regions: []economysim.RegionID{alpha}})
	if err != nil {
		t.Fatalf("StockCover() error = %v, want nil", err)
	}
	byCommodity := map[economysim.CommodityID]indicator.CoverRow{}
	for _, r := range got.Rows {
		byCommodity[r.Commodity] = r
	}
	if g := byCommodity[grain]; g.Cover.Value != 434_000 || g.Unsold.Value != 1705 || g.Held != 217 || g.Consumed != 10 {
		t.Errorf("StockCover().grain = %+v, want cover 434 000, unsold 1 705 (217 held, 10 consumed)", g)
	}
	// Cart: 5 held, none consumed (4 used as haul, a different reason): undefined.
	if c := byCommodity[cart]; c.Cover.OK || c.Cover.Value != 0 || !c.Unsold.OK {
		t.Errorf("StockCover().cart = %+v, want the cover undefined and the unsold share 0 defined", c)
	}
	// Empty pantry: nothing held, something consumed: cover 0, defined.
	setStock(&closing, alpha, loaf, 0, 0, 0)
	empty, err := indicator.StockCover(w, closing, indicator.CellSel{Regions: []economysim.RegionID{alpha}, Commodities: []economysim.CommodityID{loaf}})
	if err != nil || !empty.Rows[0].Cover.OK || empty.Rows[0].Cover.Value != 0 || empty.Total.Cover.Value != 0 {
		t.Errorf("StockCover(empty pantry) = %+v, %v, want a defined cover of 0", empty.Rows, err)
	}
	if _, err := indicator.StockCover(w, valleySnap(3, [4]int64{}), indicator.CellSel{}); !errors.Is(err, indicator.ErrWindowOrder) {
		t.Errorf("StockCover(snapshot at the wrong tick) error = %v, want ErrWindowOrder", err)
	}
}
