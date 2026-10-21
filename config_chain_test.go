package economysim_test

import (
	"strconv"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// x4Ticks, x4Window and x4Seeds are the length, the evaluation window and the
// seeds of X4 (spec-reference-chain.md 7.4 and 8.2: 1 095 ticks, the last 365,
// seeds 1 to 4). They are test constants (FR-REF-025).
const (
	x4Ticks  = 1095
	x4Window = 365
)

var x4Seeds = []uint64{1, 2, 3, 4}

// x4Band is the value of the band key key in bands/second.toml; ok is false when
// the key has no value line (FR-REF-022: the measure is then reported only).
func x4Band(t testing.TB, key string) (v float64, ok bool) {
	t.Helper()
	for _, tb := range plainTOML(t, chainsDir+"/bands/second.toml") {
		if tb.name != "x4."+key {
			continue
		}
		s, has := tb.kv["value"]
		if !has {
			return 0, false
		}
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			t.Fatalf("bands/second.toml [x4.%s] value %q: %v", key, s, err)
		}
		return f, true
	}
	t.Fatalf("bands/second.toml has no table [x4.%s]", key)
	return 0, false
}

// x4Measures is what X4 computes over the window of one seed.
type x4Measures struct {
	minTradeShareBP int64   // least share of ticks traded among producible tradable commodities
	leastTraded     string  // that commodity
	meanVolume      float64 // mean units per tick per active market
	active          int     // active markets
	maxPinnedBP     int64   // most share of ticks a market is pinned
	mostPinned      string  // that market
	minHouseholdsBP int64   // least households of the window, share of the start
	maxRange        float64 // largest high/low price ratio of a commodity
	widest          string  // that commodity
}

// x4Producible returns the commodities that some recipe makes and does not
// discard, from the configuration alone: the tradable commodities with a producer.
func x4Producible(cfg economysim.Config) map[string]bool {
	out := map[string]bool{}
	for _, r := range cfg.Recipes {
		for _, o := range r.Outputs {
			if !o.Discard {
				out[o.Commodity] = true
			}
		}
	}
	return out
}

// x4Measure evaluates criteria (b) to (f) over the last x4Window snapshots of res
// and checks the conservation identities of criterion (a) after every tick.
func x4Measure(t *testing.T, cfg economysim.Config, res result) x4Measures {
	t.Helper()
	var start int64
	for _, g := range cfg.Groups {
		start += g.Households
	}
	producible := x4Producible(cfg)
	type key struct{ region, commodity int }
	traded := map[key]int64{}
	volume := map[key]int64{}
	pinned := map[key]int64{}
	low, high := map[int]economysim.Price{}, map[int]economysim.Price{}
	m := x4Measures{minHouseholdsBP: -1, minTradeShareBP: -1}
	for tick := int64(0); tick < x4Ticks; tick++ {
		s, ok := res.Samples[tick]
		if !ok {
			t.Fatalf("no snapshot after tick %d", tick)
		}
		x4Conservation(t, s)
		if tick < x4Ticks-x4Window {
			continue
		}
		if bp := s.Households * 10_000 / start; m.minHouseholdsBP < 0 || bp < m.minHouseholdsBP {
			m.minHouseholdsBP = bp
		}
		for _, row := range s.Markets {
			k := key{int(row.State.Region), int(row.State.Commodity)}
			if row.State.Volume > 0 {
				traded[k]++
				volume[k] += int64(row.State.Volume)
			}
			if row.Beliefs > 0 && 2*(row.AtFloor+row.AtCeiling) > row.Beliefs {
				pinned[k]++
			}
			if p := row.State.Price; p > 0 {
				c := k.commodity
				if lo, seen := low[c]; !seen || p < lo {
					low[c] = p
				}
				if p > high[c] {
					high[c] = p
				}
			}
		}
	}
	final := res.Samples[x4Ticks-1]
	name := func(k key) string {
		return final.Names.Regions[k.region] + "/" + final.Names.Commodities[k.commodity]
	}
	// (b): per commodity, the best market of its regions.
	best := map[int]int64{}
	for k, n := range traded {
		best[k.commodity] = max(best[k.commodity], n)
	}
	for c, id := range final.Names.Commodities {
		if !producible[id] {
			continue
		}
		if bp := best[c] * 10_000 / x4Window; m.minTradeShareBP < 0 || bp < m.minTradeShareBP {
			m.minTradeShareBP, m.leastTraded = bp, id
		}
	}
	// (c): the mean volume of the markets that traded in the window.
	var total int64
	for k, v := range volume {
		total += v
		m.active++
		_ = k
	}
	if m.active > 0 {
		m.meanVolume = float64(total) / float64(int64(m.active)*x4Window)
	}
	// (d)
	for k, n := range pinned {
		if bp := n * 10_000 / x4Window; bp > m.maxPinnedBP || (bp == m.maxPinnedBP && m.mostPinned == "") {
			m.maxPinnedBP, m.mostPinned = bp, name(k)
		}
	}
	// (f)
	for c, hi := range high {
		if r := float64(hi) / float64(low[c]); r > m.maxRange {
			m.maxRange, m.widest = r, final.Names.Commodities[c]
		}
	}
	return m
}

// x4Conservation checks the exact identities of criterion (a) on one snapshot
// (spec-ledger.md 9.1): per commodity, the units in accounts plus the boundary
// balances are 0, and so is the money of the accounts plus the money boundary.
func x4Conservation(t *testing.T, s economysim.Snapshot) {
	t.Helper()
	goods := map[economysim.CommodityID]int64{}
	var money int64
	for _, b := range s.Boundary {
		if b.Commodity == economysim.NoCommodity {
			money += b.Balance
		} else {
			goods[b.Commodity] += b.Balance
		}
	}
	for _, row := range s.Markets {
		goods[row.State.Commodity] += int64(row.HeldByAgents) + int64(row.HeldByEstate)
	}
	for c, v := range goods {
		if v != 0 {
			t.Errorf("tick %d: commodity %q: units in accounts plus boundary = %d, want 0", s.Tick, s.Names.Commodities[c], v)
		}
	}
	if v := int64(s.Money) + money; v != 0 {
		t.Errorf("tick %d: money of the accounts plus boundary = %d, want 0", s.Tick, v)
	}
}

// TestConfigOnly_SecondRunsToTheStablePredicate verifies X4, the
// configuration-only chain test: pack second, loaded from testdata/chains/second
// alone, builds a world and runs 1 095 ticks for seeds 1 to 4 with no Go specific
// to the pack, and over the last 365 ticks the stable-state predicate holds. The
// test names no commodity, recipe or role of the pack: every identifier comes from
// the configuration and the snapshot's Names (architecture.md 9.4).
//
// Asserted: (a) the audit passes every tick and the conservation identities hold
// exactly, (c) the mean volume per active market is at least one unit a tick (the
// paper's number, spec-beliefs.md 19) and the Go-built secondConfig has the file's
// hash (FR-CFG-004). Reported, and asserted only once bands/second.toml gives the
// key a value (FR-REF-022): (b) x4.trade_share_bp, (d) x4.pinned_share_bp, (e)
// x4.households_bp, (f) x4.range_factor.
//
// Requirement: FR-REF-031, FR-REF-017, FR-REF-001, FR-CFG-004, FR-PRO-046
//
// Guard: a branch on a commodity, recipe or role identifier anywhere in the
// library would fail here on a setting it never heard of.
func TestConfigOnly_SecondRunsToTheStablePredicate(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test (1 095 ticks, four seeds), skipped with -short")
	}
	cfg := loadValidPack(t, packSecond...)
	if economysim.HashConfig(cfg) != economysim.HashConfig(secondConfig()) {
		t.Fatalf("HashConfig(file) != HashConfig(secondConfig()): the file and the Go build disagree")
	}
	for _, seed := range x4Seeds {
		t.Run("seed"+strconv.FormatUint(seed, 10), func(t *testing.T) {
			t.Parallel()
			res := runPack(t, cfg, runSpec{Seed: seed, Ticks: x4Ticks, Drive: driveStep, Audit: true, Snapshots: snapEvery})
			m := x4Measure(t, cfg, res)
			var traj []int64
			for _, tick := range []int64{0, 50, 100, 200, 365, 730, x4Ticks - 1} {
				traj = append(traj, res.Samples[tick].Households)
			}
			t.Logf("X4 second seed %d: households at ticks 0, 50, 100, 200, 365, 730, %d = %v (start %d)", seed, x4Ticks-1, traj, res.Samples[0].Households)
			t.Logf("X4 second seed %d: households %d bp of start (min), trade share %d bp (least: %s), "+
				"mean volume %.2f over %d active markets, pinned %d bp (most: %s), price range x%.1f (widest: %s)",
				seed, m.minHouseholdsBP, m.minTradeShareBP, m.leastTraded, m.meanVolume, m.active, m.maxPinnedBP, m.mostPinned, m.maxRange, m.widest)
			if m.meanVolume < 1 {
				t.Errorf("X4 (c) second seed %d: mean volume per active market = %.3f units a tick over %d active markets, want >= 1", seed, m.meanVolume, m.active)
			}
			if v, ok := x4Band(t, "trade_share_bp"); ok && float64(m.minTradeShareBP) < v {
				t.Errorf("X4 (b) second seed %d: %s trades in %d bp of the ticks, want >= %v", seed, m.leastTraded, m.minTradeShareBP, v)
			}
			if v, ok := x4Band(t, "pinned_share_bp"); ok && float64(m.maxPinnedBP) > v {
				t.Errorf("X4 (d) second seed %d: market %s pinned in %d bp of the ticks, want <= %v", seed, m.mostPinned, m.maxPinnedBP, v)
			}
			if v, ok := x4Band(t, "households_bp"); ok && float64(m.minHouseholdsBP) < v {
				t.Errorf("X4 (e) second seed %d: households fell to %d bp of the start, want >= %v", seed, m.minHouseholdsBP, v)
			}
			if v, ok := x4Band(t, "range_factor"); ok && m.maxRange >= v {
				t.Errorf("X4 (f) second seed %d: %s price range x%.2f, want < %v", seed, m.widest, m.maxRange, v)
			}
		})
	}
}
