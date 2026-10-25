package economysim_test

import (
	"errors"
	"log/slog"
	"slices"
	"strings"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// withRegions returns the reference configuration with the regions east and
// south added to north. A region needs a basket within the tolerance of the
// upkeep (999 at the reference prices), so each takes north's.
func withRegions(c economysim.Config, ids ...string) economysim.Config {
	for _, id := range ids {
		r := economysim.DefaultRegion()
		r.ID = id
		c.Regions = append(c.Regions, r)
	}
	return c
}

// edgeOf is an edge for a table.
func edgeOf(from, to string, d int64) economysim.Edge {
	return economysim.Edge{From: from, To: to, Distance: d}
}

// TestValidate_EdgeRules verifies the rules of an edge list (spec-regions.md
// 4.1.3, 9.1): each rule rejects a list that breaks only it with its sentinel
// and the path of the row, and the lists the spec allows are accepted: both
// directions of a pair with different distances, a one-way edge, and a distance
// of exactly 1 and exactly MaxDistance.
//
// Requirement: FR-REG-001, FR-REG-009, FR-REG-010, FR-REG-011, FR-REG-012, FR-REG-013
func TestValidate_EdgeRules(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		edges []economysim.Edge
		want  error // nil: accepted
		path  []string
	}{
		{"asymmetric pair and a one-way edge", []economysim.Edge{
			edgeOf("north", "south", 3), edgeOf("south", "north", 9), edgeOf("east", "north", 1),
		}, nil, nil},
		{"shortest and longest distance", []economysim.Edge{
			edgeOf("north", "south", 1), edgeOf("south", "north", economysim.MaxDistance),
		}, nil, nil},
		{"unknown From", []economysim.Edge{edgeOf("nowhere", "south", 3)}, economysim.ErrUnknownRegion, []string{"edges[0]", "from"}},
		{"unknown To", []economysim.Edge{edgeOf("north", "nowhere", 3)}, economysim.ErrUnknownRegion, []string{"edges[0]", "to"}},
		{"a loop", []economysim.Edge{edgeOf("south", "south", 3)}, economysim.ErrSelfEdge, []string{"edges[0]", "to"}},
		{"a pair twice", []economysim.Edge{edgeOf("north", "south", 3), edgeOf("east", "south", 1), edgeOf("north", "south", 4)},
			economysim.ErrDuplicateID, []string{"edges[2]"}},
		{"a distance of 0", []economysim.Edge{edgeOf("north", "south", 0)}, economysim.ErrOutOfBounds, []string{"edges[0]", "distance"}},
		{"a negative distance", []economysim.Edge{edgeOf("north", "south", -5)}, economysim.ErrOutOfBounds, []string{"edges[0]", "distance"}},
		{"a distance over MaxDistance", []economysim.Edge{edgeOf("north", "south", economysim.MaxDistance+1)}, economysim.ErrOutOfBounds, []string{"edges[0]", "distance"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := withRegions(referenceConfig(), "east", "south")
			c.Edges = tt.edges
			err := c.Validate()
			if tt.want == nil {
				if err != nil {
					t.Errorf("Validate(%s) = %v, want nil", tt.name, err)
				}
				return
			}
			vs, other := violations(err)
			if !errors.Is(err, tt.want) || len(other) != 0 || len(vs) != 1 || !slices.Equal(vs[0].Path, tt.path) {
				t.Errorf("Validate(%s) = %v, want only %v at %q", tt.name, err, tt.want, tt.path)
			}
		})
	}
}

// TestValidate_EdgeRulesReportEveryBreak verifies that a list with two broken
// rows reports both, not the first: a row with an unknown end and a distance of
// 0 gives two errors, at two paths.
//
// Requirement: FR-REG-001, FR-REG-009
func TestValidate_EdgeRulesReportEveryBreak(t *testing.T) {
	t.Parallel()
	c := withRegions(referenceConfig(), "south")
	c.Edges = []economysim.Edge{edgeOf("north", "nowhere", 0)}
	vs, _ := violations(c.Validate())
	want := [][]string{{"edges[0]", "distance"}, {"edges[0]", "to"}}
	if got := pathsOf(vs); !slices.EqualFunc(got, want, slices.Equal) {
		t.Errorf("Validate(unknown end and distance 0) paths = %q, want %q (sorted by path)", got, want)
	}
}

// TestValidate_RegionPriceRules verifies the rules of the overrides (spec-regions.md
// 5.2, 5.3, 9.1): a price of 1 and of MaxPrice is accepted, a price far above or
// below the commodity's own is accepted (no relation is required, 5.2.4), a key
// twice, an unknown region, an unknown commodity and a price outside [1,
// MaxPrice] are rejected, and more than MaxRegionPrices rows is ErrTooMany at
// the path of the list.
//
// Requirement: FR-REG-033, FR-REG-035
func TestValidate_RegionPriceRules(t *testing.T) {
	t.Parallel()
	row := func(region, commodity string, p economysim.Price) economysim.RegionPrice {
		return economysim.RegionPrice{Region: region, Commodity: commodity, Price: p}
	}
	tests := []struct {
		name string
		rows []economysim.RegionPrice
		want error
		path []string
	}{
		// seed has no upkeep or basket role, so a wild price breaks no other rule.
		{"lowest and highest price", []economysim.RegionPrice{row("north", "seed", 60), row("north", "waste", economysim.MaxPrice)}, nil, nil},
		{"far from the commodity's price", []economysim.RegionPrice{row("north", "seed", 2_000)}, nil, nil},
		{"a key twice", []economysim.RegionPrice{row("north", "seed", 230), row("north", "seed", 231)}, economysim.ErrDuplicateID, []string{"region_prices[1]"}},
		{"unknown region", []economysim.RegionPrice{row("nowhere", "seed", 230)}, economysim.ErrUnknownRegion, []string{"region_prices[0]", "region"}},
		{"unknown commodity", []economysim.RegionPrice{row("north", "ghost", 230)}, economysim.ErrUnknownCommodity, []string{"region_prices[0]", "commodity"}},
		{"price 0", []economysim.RegionPrice{row("north", "seed", 0)}, economysim.ErrOutOfBounds, []string{"region_prices[0]", "price"}},
		{"price over MaxPrice", []economysim.RegionPrice{row("north", "seed", economysim.MaxPrice+1)}, economysim.ErrOutOfBounds, []string{"region_prices[0]", "price"}},
		{"more than MaxRegionPrices", make([]economysim.RegionPrice, economysim.MaxRegionPrices+1), economysim.ErrTooMany, []string{"region_prices"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := referenceConfig()
			c.RegionPrices = tt.rows
			err := c.Validate()
			if tt.want == nil {
				if err != nil {
					t.Errorf("Validate(%s) = %v, want nil", tt.name, err)
				}
				return
			}
			vs, other := violations(err)
			if !errors.Is(err, tt.want) || len(other) != 0 || len(vs) != 1 || !slices.Equal(vs[0].Path, tt.path) {
				t.Errorf("Validate(%s) = %v, want only %v at %q", tt.name, err, tt.want, tt.path)
			}
		})
	}
}

// TestValidate_EffectivePriceFloorAndBasket verifies that the rules which read
// the reference price read the effective one (spec-regions.md 5.2.2, 5.2.3,
// example A): a region whose override makes the staple dearer needs a dearer
// basket, and with it validates; the basket of another region is still checked
// at the commodity's price; an override below the resolution floor is rejected at
// its row; a default below the floor is excused only where every region
// overrides it; and the warm-start band is the band of the seed's region.
//
// The upkeep is 2 staple + 1 prepared + 1 fuel = 2*96 + 628 + 179 = 999.
// Raising the staple to 196 in the second region costs 2*196 + 628 + 179 =
// 1 199, and the tolerance of a basket of 1 200 is 1 200 * 1 000 / 10 000 = 120.
//
// Requirement: FR-REG-037, FR-REG-036
func TestValidate_EffectivePriceFloorAndBasket(t *testing.T) {
	t.Parallel()
	dearStaple := economysim.RegionPrice{Region: "middle", Commodity: "staple", Price: 196}
	middle := func(basket economysim.Money) economysim.Config {
		c := withRegions(referenceConfig(), "middle")
		c.Regions[1].ReferenceBasket = basket
		c.RegionPrices = []economysim.RegionPrice{dearStaple}
		return c
	}
	t.Run("the basket of the dearer region matches its prices", func(t *testing.T) {
		t.Parallel()
		if err := middle(1_200).Validate(); err != nil {
			t.Errorf("Validate(middle basket 1200, staple 196) = %v, want nil", err)
		}
	})
	t.Run("the commodity's basket is a mismatch at the effective price", func(t *testing.T) {
		t.Parallel()
		// |1 000 - 1 199| = 199 is above 1 000 * 1 000 / 10 000 = 100.
		err := middle(1_000).Validate()
		vs, _ := violations(err)
		if !errors.Is(err, economysim.ErrBasketMismatch) || len(vs) != 1 || !slices.Equal(vs[0].Path, []string{"regions", "middle", "reference_basket"}) || vs[0].Bound != "1199" {
			t.Errorf("Validate(middle basket 1000, staple 196) = %v, want only ErrBasketMismatch at regions/middle/reference_basket with cost 1199", err)
		}
	})
	t.Run("another region keeps the commodity's price", func(t *testing.T) {
		t.Parallel()
		c := middle(1_200)
		c.Regions[0].ReferenceBasket = 1_200 // north: cost 999, |1 200 - 999| = 201 > 120
		err := c.Validate()
		vs, _ := violations(err)
		if !errors.Is(err, economysim.ErrBasketMismatch) || len(vs) != 1 || !slices.Equal(vs[0].Path, []string{"regions", "north", "reference_basket"}) {
			t.Errorf("Validate(north basket 1200 at the default price) = %v, want only ErrBasketMismatch at regions/north/reference_basket", err)
		}
	})
	t.Run("an override below the floor", func(t *testing.T) {
		t.Parallel()
		c := referenceConfig()
		c.RegionPrices = []economysim.RegionPrice{{Region: "north", Commodity: "seed", Price: 40}}
		err := c.Validate()
		vs, _ := violations(err)
		if !errors.Is(err, economysim.ErrOutOfBounds) || len(vs) != 1 || !slices.Equal(vs[0].Path, []string{"region_prices[0]", "price"}) || vs[0].Value != "40" || vs[0].Bound != "50" {
			t.Errorf("Validate(seed override 40) = %v, want only ErrOutOfBounds at region_prices[0].price, value 40, bound 50", err)
		}
	})
	t.Run("a default below the floor, overridden in the only region", func(t *testing.T) {
		t.Parallel()
		c := referenceConfig()
		for i := range c.Commodities {
			if c.Commodities[i].ID == "seed" {
				c.Commodities[i].ReferencePrice = 40
			}
		}
		c.RegionPrices = []economysim.RegionPrice{{Region: "north", Commodity: "seed", Price: 222}}
		if err := c.Validate(); err != nil {
			t.Errorf("Validate(seed default 40, overridden to 222 in north) = %v, want nil", err)
		}
		// A second region that does not override it uses the default of 40.
		two := withRegions(c, "south")
		err := two.Validate()
		vs, _ := violations(err)
		if !errors.Is(err, economysim.ErrOutOfBounds) || len(vs) != 1 || !slices.Equal(vs[0].Path, []string{"commodities", "seed", "reference_price"}) {
			t.Errorf("Validate(seed default 40, south uses it) = %v, want only ErrOutOfBounds at commodities/seed/reference_price", err)
		}
	})
	t.Run("the warm-start band follows the override", func(t *testing.T) {
		t.Parallel()
		// The warm start seeds grain-like prices near the commodity's 96. A band of
		// 1 000 to 100 000 basis points of an effective price of 96 * 100 = 9 600
		// puts 96 below 960 and rejects it, which the commodity's price alone does not.
		c := referenceConfig()
		c.RegionPrices = []economysim.RegionPrice{{Region: "north", Commodity: "staple", Price: 9_600}}
		c.Regions[0].ReferenceBasket = 9_600*2 + 628 + 179
		err := c.Validate()
		if !errors.Is(err, economysim.ErrWarmStartPrice) {
			t.Errorf("Validate(staple override 9600 against a warm seed price) = %v, want ErrWarmStartPrice", err)
		}
	})
}

// TestWarnings_RegionRules verifies the five warnings of spec-regions.md 9.2: a
// world of one region and no edge draws none of the first three; two regions
// with no edge draw region_isolated for each and graph_not_strongly_connected;
// an edge longer than a detour draws edge_not_shortest at its row (as written);
// a region with no group draws region_without_agents; an override equal to the
// commodity's price draws override_equals_default.
//
// Requirement: FR-REG-007, FR-REG-031, FR-REG-034
func TestWarnings_RegionRules(t *testing.T) {
	t.Parallel()
	rules := map[string]bool{
		"region_isolated": true, "graph_not_strongly_connected": true, "edge_not_shortest": true,
		"region_without_agents": true, "override_equals_default": true,
	}
	// got returns the region warnings as "rule path" strings in the order Warnings gives.
	got := func(c economysim.Config) []string {
		var out []string
		for _, w := range c.Warnings() {
			if rules[w.Rule] {
				out = append(out, w.Rule+" "+strings.Join(w.Path, "/"))
			}
		}
		return out
	}
	grouped := func(c economysim.Config, ids ...string) economysim.Config {
		c = withRegions(c, ids...)
		for _, id := range ids {
			c.Groups = append(c.Groups, economysim.Group{Region: id, Role: "cook", Households: 10})
		}
		return c
	}
	detour := []economysim.Edge{edgeOf("east", "north", 30), edgeOf("north", "south", 30), edgeOf("east", "south", 90)}
	tests := []struct {
		name string
		cfg  economysim.Config
		want []string
	}{
		{"one region and no edge", referenceConfig(), nil},
		{"two regions with no edge", grouped(referenceConfig(), "south"), []string{
			"graph_not_strongly_connected edges", "region_isolated regions/north", "region_isolated regions/south",
		}},
		{"two regions joined one way", func() economysim.Config {
			c := grouped(referenceConfig(), "south")
			c.Edges = []economysim.Edge{edgeOf("north", "south", 5)}
			return c
		}(), []string{"graph_not_strongly_connected edges"}},
		{"two regions joined both ways", func() economysim.Config {
			c := grouped(referenceConfig(), "south")
			c.Edges = []economysim.Edge{edgeOf("north", "south", 5), edgeOf("south", "north", 6)}
			return c
		}(), nil},
		{"an edge longer than a detour", func() economysim.Config {
			c := grouped(referenceConfig(), "east", "south")
			c.Edges = append(detour, edgeOf("north", "east", 30), edgeOf("south", "north", 30))
			return c
		}(), []string{"edge_not_shortest edges[2]"}},
		{"an edge as long as the detour", func() economysim.Config {
			c := grouped(referenceConfig(), "east", "south")
			c.Edges = []economysim.Edge{edgeOf("east", "north", 30), edgeOf("north", "south", 30), edgeOf("east", "south", 60),
				edgeOf("north", "east", 30), edgeOf("south", "north", 30)}
			return c
		}(), nil},
		{"a region with no group", withRegions(referenceConfig(), "south"), []string{
			"graph_not_strongly_connected edges", "region_isolated regions/north", "region_isolated regions/south", "region_without_agents regions/south",
		}},
		{"an override equal to the default", func() economysim.Config {
			c := referenceConfig()
			c.RegionPrices = []economysim.RegionPrice{{Region: "north", Commodity: "seed", Price: 223}, {Region: "north", Commodity: "waste", Price: 50}}
			return c
		}(), []string{"override_equals_default region_prices[1]/price"}},
		{"invalid edges are skipped", func() economysim.Config {
			c := grouped(referenceConfig(), "south")
			c.Edges = []economysim.Edge{edgeOf("north", "nowhere", 5), edgeOf("south", "south", 3), edgeOf("north", "south", 0)}
			return c
		}(), []string{"graph_not_strongly_connected edges", "region_isolated regions/north", "region_isolated regions/south"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if g := got(tt.cfg); !slices.Equal(g, tt.want) {
				t.Errorf("Warnings(%s) region rules = %q, want %q", tt.name, g, tt.want)
			}
		})
	}
}

// TestBounds_Constants verifies the bounds of spec-regions.md 4.2 and the claim
// behind them: a shortest path of MaxRegions-1 edges of the largest effective
// length sums below 10^13, far below int64, and the derived counts follow their
// factors.
//
// Requirement: FR-REG-055, FR-REG-029
func TestBounds_Constants(t *testing.T) {
	t.Parallel()
	if economysim.MaxDistance != 1_000_000_000 || economysim.MaxEdgeFactorBP != 100_000 {
		t.Errorf("MaxDistance, MaxEdgeFactorBP = %d, %d, want 1e9, 100000", economysim.MaxDistance, economysim.MaxEdgeFactorBP)
	}
	if economysim.MaxEdges != 20*economysim.MaxRegions {
		t.Errorf("MaxEdges = %d, want 20 * MaxRegions = %d", economysim.MaxEdges, 20*economysim.MaxRegions)
	}
	if economysim.MaxRegionPrices != economysim.MaxRegions*economysim.MaxCommodities {
		t.Errorf("MaxRegionPrices = %d, want MaxRegions * MaxCommodities = %d", economysim.MaxRegionPrices, economysim.MaxRegions*economysim.MaxCommodities)
	}
	// 10^9 * 10^5 / 10^4 = 10^10 per edge; 999 of them: 9.99e12.
	const effective = int64(economysim.MaxDistance) * economysim.MaxEdgeFactorBP / 10_000
	if path := int64(economysim.MaxRegions-1) * effective; path >= 10_000_000_000_000 {
		t.Errorf("longest path = %d, want below 10^13", path)
	}
}

// FuzzValidateGraph verifies that Validate, Warnings and HashConfig never panic
// on any edge list and any list of overrides, whatever the ends, the distances
// and the prices, that the hash does not depend on the order the rows are
// listed in, and that a configuration Validate accepts has only edges with
// known, distinct ends and a distance in [1, MaxDistance] and only overrides
// with a price in [1, MaxPrice].
//
// The input is read as rows of 8 bytes: an edge (two end indices into a list of
// four region IDs and one unknown one, and a signed distance), then an override
// (an index of region, one of commodity and a signed price).
//
// Requirement: NFR-REG-003
func FuzzValidateGraph(f *testing.F) {
	f.Add([]byte{0, 1, 0, 0, 0, 0, 0, 5, 1, 0, 0, 0, 0, 0, 0, 9})
	f.Add([]byte{2, 2, 0, 0, 0, 0, 0, 1, 0, 4, 255, 255, 255, 255, 255, 255})
	f.Add([]byte{3, 0, 127, 255, 255, 255, 255, 255, 3, 0, 127, 255, 255, 255, 255, 255})
	regions := []string{"north", "east", "south", "west", "nowhere"}
	commodities := []string{"fuel", "seed", "staple", "ghost"}
	f.Fuzz(func(t *testing.T, data []byte) {
		c := withRegions(referenceConfig(), "east", "south", "west")
		for len(data) >= 8 && len(c.Edges)+len(c.RegionPrices) < 64 {
			row := data[:8]
			data = data[8:]
			dist := int64(int32(uint32(row[2])<<24|uint32(row[3])<<16|uint32(row[4])<<8|uint32(row[5]))) * int64(row[6]+1)
			c.Edges = append(c.Edges, edgeOf(regions[int(row[0])%len(regions)], regions[int(row[1])%len(regions)], dist))
			c.RegionPrices = append(c.RegionPrices, economysim.RegionPrice{
				Region: regions[int(row[7])%len(regions)], Commodity: commodities[int(row[6])%len(commodities)], Price: economysim.Price(dist),
			})
		}
		err := c.Validate()
		_ = c.Warnings()
		// Two rows of one key are a configuration that fails validation, and the
		// stable sort keeps their written order, so the hash may depend on it; the
		// order independence is claimed for the rows of distinct keys.
		keys := map[[2]string]bool{}
		distinct := true
		for _, e := range c.Edges {
			distinct = distinct && !keys[[2]string{e.From, e.To}]
			keys[[2]string{e.From, e.To}] = true
		}
		for _, p := range c.RegionPrices {
			distinct = distinct && !keys[[2]string{"price", p.Region + "/" + p.Commodity}]
			keys[[2]string{"price", p.Region + "/" + p.Commodity}] = true
		}
		want := hexHash(c)
		slices.Reverse(c.Edges)
		slices.Reverse(c.RegionPrices)
		if got := hexHash(c); distinct && got != want {
			t.Errorf("HashConfig(rows of distinct keys reversed) = %s, want %s", got, want)
		}
		if err != nil {
			return
		}
		seen := map[[2]string]bool{}
		for _, e := range c.Edges {
			key := [2]string{e.From, e.To}
			if e.From == e.To || e.From == "nowhere" || e.To == "nowhere" || e.Distance < 1 || e.Distance > economysim.MaxDistance || seen[key] {
				t.Errorf("Validate accepted the edge %+v (or a repeat of it), want it rejected", e)
			}
			seen[key] = true
		}
		for _, p := range c.RegionPrices {
			if p.Price < 1 || p.Price > economysim.MaxPrice || p.Region == "nowhere" || p.Commodity == "ghost" {
				t.Errorf("Validate accepted the override %+v, want it rejected", p)
			}
		}
	})
}

// TestWarnings_ToolUnaffordableReadsTheRegionsPrice verifies that tool_unaffordable
// compares StartMoney with the effective price in the regions the role has a
// group in (spec-regions.md 5.2.2): a dear override raises it where the
// commodity's own price would not, and a cheap one silences it.
//
// The cook's tool is tool_b at 17 105 and its StartMoney is set to 17 105, which
// buys it at the commodity's price.
//
// Requirement: FR-REG-036
func TestWarnings_ToolUnaffordableReadsTheRegionsPrice(t *testing.T) {
	t.Parallel()
	fires := func(c economysim.Config) bool {
		for _, w := range c.Warnings() {
			if w.Rule == "tool_unaffordable" && slices.Equal(w.Path, []string{"roles", "cook", "start_money"}) {
				return true
			}
		}
		return false
	}
	for _, tt := range []struct {
		name  string
		price economysim.Price
		want  bool
	}{
		{"no override", 0, false},
		{"dear override in the group's region", 17_106, true},
		{"cheap override in the group's region", 100, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := referenceConfig()
			for i := range c.Roles {
				if c.Roles[i].ID == "cook" {
					c.Roles[i].StartMoney = 17_105
				}
			}
			if tt.price != 0 {
				c.RegionPrices = []economysim.RegionPrice{{Region: "north", Commodity: "tool_b", Price: tt.price}}
			}
			if got := fires(c); got != tt.want {
				t.Errorf("tool_unaffordable fires = %v, want %v (tool_b override %d)", got, tt.want, tt.price)
			}
		})
	}
}

// TestNewGraph_ErrorPathIndexesTheListAsGiven verifies that NewGraph reports a
// row by its index in the list as given, as Validate does, not by its place in
// the canonical order.
//
// Guard: the edges are given as b->a, a->a; sorted, the self edge is first. A
// NewGraph that read the canonical copy would say edges[0] where Validate says
// edges[1].
//
// Requirement: FR-REG-001, FR-REG-009
func TestNewGraph_ErrorPathIndexesTheListAsGiven(t *testing.T) {
	t.Parallel()
	c := withRegions(referenceConfig(), "south")
	c.Edges = []economysim.Edge{edgeOf("south", "north", 5), edgeOf("north", "north", 5)}
	_, err := economysim.NewGraph(c)
	var ve *economysim.ValidationError
	if !errors.Is(err, economysim.ErrSelfEdge) || !errors.As(err, &ve) {
		t.Fatalf("NewGraph(self edge second) error = %v, want ErrSelfEdge", err)
	}
	if want := []string{"edges[1]", "to"}; !slices.Equal(ve.Path, want) {
		t.Errorf("NewGraph(self edge second) path = %v, want %v", ve.Path, want)
	}
	vs, _ := violations(c.Validate())
	found := false
	for _, v := range vs {
		found = found || slices.Equal(v.Path, []string{"edges[1]", "to"})
	}
	if !found {
		t.Errorf("Validate paths = %v, want one at edges[1].to as NewGraph reports", vs)
	}
}

// TestWarnings_OverrideEqualsDefaultUsesTheFirstOfARepeatedCommodity verifies
// that on a configuration with a repeated commodity ID, override_equals_default
// compares with the first row, the one the validator's lookup keeps.
//
// Guard: the last row would make the warning depend on the order of a list that
// Validate rejects anyway (ErrDuplicateID), and Warnings is documented to work
// on a configuration that does not validate.
func TestWarnings_OverrideEqualsDefaultUsesTheFirstOfARepeatedCommodity(t *testing.T) {
	t.Parallel()
	c := referenceConfig()
	first := c.Commodities[0]
	dup := first
	dup.ReferencePrice = first.ReferencePrice + 7
	c.Commodities = append(c.Commodities, dup)
	c.RegionPrices = []economysim.RegionPrice{{Region: "north", Commodity: first.ID, Price: first.ReferencePrice}}
	for _, w := range c.Warnings() {
		if w.Rule == "override_equals_default" {
			return
		}
	}
	t.Errorf("Warnings() has no override_equals_default for an override equal to the first row's price %d", first.ReferencePrice)
}

// TestLog_WorldCreatedCarriesTheEdgesAndComponents verifies that the Info line
// "world created" carries the number of edges and the number of strongly
// connected components of the region graph, 0 for a world of one region as
// Snapshot.Components is, and that the same count is on the line whose warning
// graph_not_strongly_connected cannot carry it.
//
// Requirement: FR-REG-056
//
// Guard: the warning has a rule and a path only, so without these two keys the
// log would say the graph is split and not into how many parts.
func TestLog_WorldCreatedCarriesTheEdgesAndComponents(t *testing.T) {
	t.Parallel()
	grouped := func(edges ...economysim.Edge) economysim.Config {
		c := withRegions(referenceConfig(), "south")
		c.Groups = append(c.Groups, economysim.Group{Region: "south", Role: "cook", Households: 10})
		c.Edges = edges
		return c
	}
	tests := []struct {
		name              string
		cfg               economysim.Config
		edges, components int64
		warned            bool
	}{
		{"one region", referenceConfig(), 0, 0, false},
		{"two regions and no edge", grouped(), 0, 2, true},
		{"two regions joined one way", grouped(edgeOf("north", "south", 5)), 1, 2, true},
		{"two regions joined both ways", grouped(edgeOf("north", "south", 5), edgeOf("south", "north", 6)), 2, 1, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			h := newCaptureHandler()
			if _, err := economysim.NewWorld(tt.cfg, economysim.WithLogger(slog.New(h))); err != nil {
				t.Fatalf("NewWorld(%s) error = %v, want nil", tt.name, err)
			}
			var line logLine
			warned := false
			for _, l := range h.records() {
				switch {
				case l.msg == "world created":
					line = l
				case l.msg == "configuration warning":
					if rule, _ := l.attr("rule"); rule.String() == "graph_not_strongly_connected" {
						warned = true
					}
				}
			}
			if got := line.intAttr("edges"); got != tt.edges {
				t.Errorf("world created edges = %d, want %d", got, tt.edges)
			}
			if got := line.intAttr("components"); got != tt.components {
				t.Errorf("world created components = %d, want %d", got, tt.components)
			}
			if warned != tt.warned {
				t.Errorf("graph_not_strongly_connected logged = %v, want %v", warned, tt.warned)
			}
		})
	}
}

// TestWarnLines_FiveRules verifies that construction logs each of the five region
// warnings of spec-regions.md 9.2 once, at Warn, as the line "configuration
// warning" with the rule and the path of the Warning, and that running the world
// logs none of them again and nothing at Info (the tick has no Info line).
//
// Setup: north, east and south joined by the detour of TestWarnings_RegionRules
// (the edge east>south, 90, against 30 + 30), a region west with no edge and no
// group, and an override of north's waste at the commodity's own price 50.
//
// Requirement: FR-REG-056, FR-REG-031, FR-REG-034, FR-REG-007
func TestWarnLines_FiveRules(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	c := withRegions(referenceConfig(), "east", "south", "west")
	for _, id := range []string{"east", "south"} {
		c.Groups = append(c.Groups, economysim.Group{Region: id, Role: "cook", Households: 10})
	}
	c.Edges = []economysim.Edge{edgeOf("east", "north", 30), edgeOf("north", "south", 30), edgeOf("east", "south", 90), edgeOf("north", "east", 30), edgeOf("south", "north", 30)}
	c.RegionPrices = []economysim.RegionPrice{{Region: "north", Commodity: "waste", Price: 50}}
	rules := []string{"region_isolated", "graph_not_strongly_connected", "edge_not_shortest", "region_without_agents", "override_equals_default"}
	h := newCaptureHandler()
	w, err := economysim.NewWorld(c, economysim.WithLogger(slog.New(h)))
	if err != nil {
		t.Fatalf("NewWorld() error = %v, want nil", err)
	}
	// The world holds the canonical configuration, so the paths it logs index the
	// sorted lists (the edge east>south is edges[1] there, edges[2] as written).
	var want []string
	for _, wn := range w.Config().Warnings() {
		if slices.Contains(rules, wn.Rule) {
			want = append(want, wn.Rule+" "+strings.Join(wn.Path, "."))
		}
	}
	for _, rule := range rules {
		if !slices.ContainsFunc(want, func(s string) bool { return strings.HasPrefix(s, rule+" ") }) {
			t.Fatalf("setup: the configuration draws no %s, want all five rules (got %q)", rule, want)
		}
	}
	logged := func() (lines []string, infos int) {
		for _, l := range h.records() {
			if l.level == slog.LevelInfo {
				infos++
			}
			if l.msg != "configuration warning" {
				continue
			}
			rule, _ := l.attr("rule")
			path, _ := l.attr("path")
			if l.level != slog.LevelWarn {
				t.Errorf("configuration warning %s logged at %v, want Warn", rule, l.level)
			}
			if slices.Contains(rules, rule.String()) {
				lines = append(lines, rule.String()+" "+path.String())
			}
		}
		return lines, infos
	}
	got, infosAtStart := logged()
	slices.Sort(got)
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Errorf("logged region warnings = %q, want each of %q once", got, want)
	}
	for range 5 {
		if err := w.Step(); err != nil {
			t.Fatalf("Step() error = %v, want nil", err)
		}
	}
	again, infos := logged()
	slices.Sort(again)
	if !slices.Equal(again, want) || infos != infosAtStart {
		t.Errorf("after 5 ticks the region warnings are %q (want the same) and there are %d Info lines (want %d: none in the tick)", again, infos, infosAtStart)
	}
}
