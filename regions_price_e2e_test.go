package economysim_test

import (
	"reflect"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// priceCutoverTicks is the length of the end-to-end runs of the price cutover:
// long enough for several clearings and the first belief updates of every market.
const priceCutoverTicks = 30

// marketEvents returns the MarketCleared events of the region, in emission order.
func marketEvents(events []economysim.Event, region economysim.RegionID) []economysim.MarketCleared {
	var out []economysim.MarketCleared
	for _, e := range events {
		if m, ok := e.(economysim.MarketCleared); ok && m.Region == region {
			out = append(out, m)
		}
	}
	return out
}

// TestPriceCutover_RegionOverrideRunsThroughAWholeWorld verifies, on the cold
// two-region pack run for 30 ticks through the public API, the effective-price
// cutover of spec-regions.md 5.2.2 end to end. The same world is run three
// times: with no override, with an override of south's staple price that equals
// the commodity's price, and with one at about twice it (196).
//
//   - the equal override changes nothing the world reports (events and final
//     snapshot are deep-equal to the run without a row): the cutover is
//     invisible until a row differs, as spec-config.md 8.2.3 asks of an append;
//   - the dear override moves south's staple market from construction (the seed
//     price of tick 0 is the effective price, 196), leaves the north's
//     market events untouched for as long as no household has entered in
//     either run (the regions share no edge or transport, so the only coupling
//     the override could cause is a leak), and changes the final staple market
//     of south.
//
// Requirement: FR-REG-032, FR-REG-036, FR-REG-038
func TestPriceCutover_RegionOverrideRunsThroughAWholeWorld(t *testing.T) {
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	t.Parallel()
	const north, south economysim.RegionID = 0, 1
	base := loadValidPack(t, "base.toml", "two.toml")
	staple := -1
	for i, c := range base.Commodities {
		if c.ID == "staple" {
			staple = i
		}
	}
	if staple < 0 || base.Commodities[staple].ReferencePrice != 96 {
		t.Fatalf("pack commodity staple not found at price 96, the arithmetic of this test depends on it")
	}
	withRow := func(price economysim.Price) economysim.Config {
		c := base
		c.Regions = slices.Clone(base.Regions)
		if price != 96 {
			// The upkeep basket of south must match the dearer staple: 2 * 196 + 628 + 179 = 1 199,
			// inside the tolerance of a basket of 1 200 (spec-regions.md 5.3.3).
			c.Regions[south].ReferenceBasket = 1_200
		}
		c.RegionPrices = []economysim.RegionPrice{{Region: "south", Commodity: "staple", Price: price}}
		return c
	}
	run := func(cfg economysim.Config) result {
		return runPack(t, cfg, runSpec{Seed: 11, Ticks: priceCutoverTicks, Observe: true})
	}
	plain, same, dear := run(base), run(withRow(96)), run(withRow(196))

	if !reflect.DeepEqual(plain.Events[1:], same.Events[1:]) {
		t.Errorf("events with an override equal to the commodity's price differ from the run without a row, want none (the WorldCreated event, which carries the configuration hash, is left out)")
	}
	sameFinal := same.Final
	sameFinal.ConfigHash = plain.Final.ConfigHash // the hash covers the row, equal or not
	if !reflect.DeepEqual(plain.Final, sameFinal) {
		t.Errorf("final snapshot with an override equal to the commodity's price differs from the run without a row, want equal")
	}

	if got, want := dear.World.ReferencePrice(south, economysim.CommodityID(staple)), economysim.Price(196); got != want {
		t.Errorf("ReferencePrice(south, staple) = %d, want %d", got, want)
	}
	if got, want := dear.World.ReferencePrice(north, economysim.CommodityID(staple)), economysim.Price(96); got != want {
		t.Errorf("ReferencePrice(north, staple) = %d, want %d", got, want)
	}

	// The AgentID space is one counter for the world, so the first household that
	// enters after construction takes the next ID whichever region it is in: when
	// south's entries differ between the runs, north's later agents carry other IDs
	// and draw other streams. Until the first entry in either run, north cannot
	// have been touched by an override in south.
	settled := int64(priceCutoverTicks)
	for _, r := range []result{plain, dear} {
		for _, e := range r.Events {
			if a, ok := e.(economysim.AgentSpawned); ok && a.Tick > 0 {
				settled = min(settled, a.Tick)
				break
			}
		}
	}
	if settled < 5 {
		t.Fatalf("the first entry after construction is at tick %d, want a window of at least 5 ticks to compare north in", settled)
	}
	before := func(ms []economysim.MarketCleared) []economysim.MarketCleared {
		i := slices.IndexFunc(ms, func(m economysim.MarketCleared) bool { return m.Tick >= settled })
		return ms[:i]
	}
	pn, dn := before(marketEvents(plain.Events, north)), before(marketEvents(dear.Events, north))
	if len(pn) == 0 || !reflect.DeepEqual(pn, dn) {
		t.Errorf("MarketCleared events of north before tick %d: %d with an override in south, %d without, want the same non-empty sequence", settled, len(dn), len(pn))
	}
	ps, ds := marketEvents(plain.Events, south), marketEvents(dear.Events, south)
	if len(ps) != len(ds) || len(ps) == 0 {
		t.Fatalf("MarketCleared events of south: %d with the override, %d without, want the same non-zero count", len(ds), len(ps))
	}
	if reflect.DeepEqual(ps, ds) {
		t.Errorf("MarketCleared events of south with a dearer staple price equal those without, want them to differ")
	}
	idx := func(r economysim.RegionID) int { return int(r)*len(base.Commodities) + staple }
	if p, d := plain.Final.Markets[idx(south)], dear.Final.Markets[idx(south)]; reflect.DeepEqual(p, d) {
		t.Errorf("final staple market of south with a dearer price equals the run without (%+v), want it to differ", d)
	}
}
