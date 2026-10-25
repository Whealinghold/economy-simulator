package economysim_test

import (
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// threeRegionEdges is a three-region configuration with edges and an override:
// the reference configuration's north, with east and south, a ring of edges and
// the staple dearer in south. It is the second pinned vector of the hash.
func threeRegionEdges() economysim.Config {
	c := withRegions(referenceConfig(), "east", "south")
	c.Edges = []economysim.Edge{edgeOf("north", "east", 100), edgeOf("east", "south", 100), edgeOf("south", "north", 130)}
	c.RegionPrices = []economysim.RegionPrice{{Region: "south", Commodity: "staple", Price: 196}}
	return c
}

// pinnedThreeRegionHash is the hash of threeRegionEdges, recorded when the
// edges and the overrides were added (hash ids 8 and 9). It moves only with a
// new encoding version (spec-config.md 8.2.1).
const pinnedThreeRegionHash = "4cf517d140fbe207d2d50e94a9a47e12fed22c527c0be278f50389b831f5bce7"

// TestHashConfig_EdgesAndPrices verifies that the edges and the overrides are
// hashed under their ids: changing a distance, an end or a price moves the hash,
// the order in which rows are listed does not, and a pinned vector of a
// three-region configuration holds.
//
// Requirement: FR-REG-053, FR-REG-002
func TestHashConfig_EdgesAndPrices(t *testing.T) {
	t.Parallel()
	base := hexHash(threeRegionEdges())
	if base != pinnedThreeRegionHash {
		t.Errorf("HashConfig(threeRegionEdges()) = %s, want %s", base, pinnedThreeRegionHash)
	}
	changes := map[string]func(c *economysim.Config){
		"a distance":    func(c *economysim.Config) { c.Edges[0].Distance++ },
		"an end":        func(c *economysim.Config) { c.Edges[0].To = "south" },
		"a start":       func(c *economysim.Config) { c.Edges[0].From = "south" },
		"an edge added": func(c *economysim.Config) { c.Edges = append(c.Edges, edgeOf("south", "east", 5)) },
		"an edge gone":  func(c *economysim.Config) { c.Edges = c.Edges[1:] },
		"a price":       func(c *economysim.Config) { c.RegionPrices[0].Price++ },
		"a region":      func(c *economysim.Config) { c.RegionPrices[0].Region = "east" },
		"a commodity":   func(c *economysim.Config) { c.RegionPrices[0].Commodity = "fuel" },
		"a price gone":  func(c *economysim.Config) { c.RegionPrices = nil },
	}
	for name, change := range changes {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			c := threeRegionEdges()
			change(&c)
			if hexHash(c) == base {
				t.Errorf("HashConfig(%s changed) = the hash of the unchanged configuration, want a different hash", name)
			}
		})
	}
	t.Run("listing order", func(t *testing.T) {
		t.Parallel()
		c := threeRegionEdges()
		slices.Reverse(c.Edges)
		c.RegionPrices = append(c.RegionPrices, economysim.RegionPrice{Region: "east", Commodity: "fuel", Price: 200})
		want := hexHash(c)
		slices.Reverse(c.RegionPrices)
		if got := hexHash(c); got != want {
			t.Errorf("HashConfig(rows listed in another order) = %s, want %s", got, want)
		}
	})
	t.Run("two ids", func(t *testing.T) {
		t.Parallel()
		// The same single row under the edge list and under the price list hashes
		// differently: the ids 8 and 9 keep the lists apart.
		a := withRegions(referenceConfig(), "south")
		a.Edges = []economysim.Edge{edgeOf("north", "south", 7)}
		b := withRegions(referenceConfig(), "south")
		b.RegionPrices = []economysim.RegionPrice{{Region: "north", Commodity: "south", Price: 7}}
		if hexHash(a) == hexHash(b) {
			t.Errorf("HashConfig(edge north>south 7) = HashConfig(price north/south 7), want the ids to differ")
		}
	})
}

// TestHashConfig_EdgesOffLeavesNoTrace verifies that no edge and no override
// leave no trace in the hash: a nil and an empty list hash alike, and both equal
// the pin of the reference configuration, which was recorded before the lists
// existed (architecture.md 2.10.5).
//
// Requirement: FR-REG-053, FR-CFG-105
//
// Guard: a module-less append must not move the hash of every existing
// configuration; written as a count of 0 the lists would.
func TestHashConfig_EdgesOffLeavesNoTrace(t *testing.T) {
	t.Parallel()
	nilLists := referenceConfig()
	emptyLists := referenceConfig()
	emptyLists.Edges, emptyLists.RegionPrices = []economysim.Edge{}, []economysim.RegionPrice{}
	if hexHash(nilLists) != pinnedReferenceHash {
		t.Errorf("HashConfig(nil lists) = %s, want the pin %s", hexHash(nilLists), pinnedReferenceHash)
	}
	if hexHash(emptyLists) != pinnedReferenceHash {
		t.Errorf("HashConfig(empty lists) = %s, want the pin %s", hexHash(emptyLists), pinnedReferenceHash)
	}
}
