package economysim_test

import (
	"fmt"
	"slices"
	"strconv"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// This file is the X4 stable-state predicate of spec-reference-chain.md 7.4 as a
// function over a run (doc/plans/plan-2-8.md, D362): the six criteria (a) to (f)
// over the last x4Window of x4Ticks snapshots, with the thresholds passed in. It
// was extracted from config_chain_test.go, where X4 first ran on pack second, so
// the test of the second pack and the recorded run of one share one definition
// (FR-REF-027: events and the snapshot only, no unexported state). The word
// "stable" is not used of a pack (STY 7.3.2): a run passes the predicate.

// x4Ticks, x4Window and x4Seeds are the length, the evaluation window and the
// seeds of X4 (spec-reference-chain.md 7.4 and 8.2: 1 095 ticks, the last 365,
// seeds 1 to 4). They are test constants (FR-REF-025).
const (
	x4Ticks  = 1095
	x4Window = 365
)

var x4Seeds = []uint64{1, 2, 3, 4}

// x4Threshold is the value of one threshold key and whether it has one: a key
// with no value is reported and not asserted (FR-REF-022).
type x4Threshold struct {
	V   float64
	Set bool
}

// x4Thresholds are the four keys of X4 that a band file gives (spec-reference-chain.md
// 7.4): the least share of ticks a producible commodity trades in, the most share
// a market is pinned, the least households as a share of the start, and the
// price range above which a commodity fails (criteria (b), (d), (e), (f)).
type x4Thresholds struct {
	TradeShareBP, PinnedShareBP, HouseholdsBP, RangeFactor x4Threshold
}

// x4Priors are the screening values of plan 2-8 D363 (STY priors): they are
// passed to the predicate and logged as priors; passing them is not setting
// them, and no bands file carries them (FR-REF-024).
var x4Priors = x4Thresholds{
	TradeShareBP:  x4Threshold{8_000, true},
	PinnedShareBP: x4Threshold{1_000, true},
	HouseholdsBP:  x4Threshold{9_000, true},
	RangeFactor:   x4Threshold{2, true}, // x4.range_factor 20 000 bp is a ratio of 2 (STY unit bp; the band file's unit is "ratio")
}

// x4Criterion is the verdict on one of the six criteria. Asserted is false when
// the criterion has no threshold (its value is reported only), and Pass is then
// true.
type x4Criterion struct {
	Key      string // "a" to "f"
	Name     string
	Asserted bool
	Pass     bool
	Detail   string // the measure, and the failure when Pass is false
}

// stableRun is what the predicate reads of a run: the snapshot after every tick
// (Samples is keyed by the tick of the step, as runPack keeps it), the households
// of the start, and the commodities a recipe makes and does not discard.
type stableRun struct {
	Samples    map[int64]economysim.Snapshot
	Start      int64
	Producible map[string]bool
	Ticks      int64 // x4Ticks
	Window     int64 // x4Window
}

// newStableRun makes the run of cfg that res kept, of the X4 length and window.
func newStableRun(cfg economysim.Config, res result) stableRun {
	var start int64
	for _, g := range cfg.Groups {
		start += g.Households
	}
	return stableRun{Samples: res.Samples, Start: start, Producible: x4Producible(cfg), Ticks: x4Ticks, Window: x4Window}
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

// stablePredicate judges a run against X4: it returns the six criteria (a) to
// (f) in that order. (a) and (c) always have a threshold (the identities are
// exact, the mean volume is at least one unit a tick); (b), (d), (e) and (f) are
// asserted when th gives their key a value.
func stablePredicate(run stableRun, th x4Thresholds) []x4Criterion {
	m, bad := x4Measure(run)
	crit := func(key, name string, asserted, pass bool, measure, fail string) x4Criterion {
		c := x4Criterion{Key: key, Name: name, Asserted: asserted, Pass: pass || !asserted, Detail: measure}
		if !c.Pass {
			c.Detail += "; " + fail
		}
		return c
	}
	a := crit("a", "invariants", true, len(bad) == 0, fmt.Sprintf("%d conservation violations", len(bad)), fmt.Sprint(bad))
	b := crit("b", "trade share", th.TradeShareBP.Set, float64(m.minTradeShareBP) >= th.TradeShareBP.V,
		fmt.Sprintf("%s trades in %d bp of the ticks", m.leastTraded, m.minTradeShareBP), fmt.Sprintf("want >= %v", th.TradeShareBP.V))
	c := crit("c", "mean volume", true, m.meanVolume >= 1,
		fmt.Sprintf("mean volume %.3f units a tick over %d active markets", m.meanVolume, m.active), "want >= 1")
	d := crit("d", "pinned share", th.PinnedShareBP.Set, float64(m.maxPinnedBP) <= th.PinnedShareBP.V,
		fmt.Sprintf("market %s pinned in %d bp of the ticks", m.mostPinned, m.maxPinnedBP), fmt.Sprintf("want <= %v", th.PinnedShareBP.V))
	e := crit("e", "households", th.HouseholdsBP.Set, float64(m.minHouseholdsBP) >= th.HouseholdsBP.V,
		fmt.Sprintf("households fell to %d bp of the start", m.minHouseholdsBP), fmt.Sprintf("want >= %v", th.HouseholdsBP.V))
	f := crit("f", "price range", th.RangeFactor.Set, m.maxRange < th.RangeFactor.V,
		fmt.Sprintf("%s price range x%.2f", m.widest, m.maxRange), fmt.Sprintf("want < %v", th.RangeFactor.V))
	return []x4Criterion{a, b, c, d, e, f}
}

// x4Measure evaluates criteria (b) to (f) over the last run.Window snapshots and
// returns the violations of the conservation identities of criterion (a), found
// after every tick.
func x4Measure(run stableRun) (x4Measures, []string) {
	type key struct{ region, commodity int }
	var bad []string
	traded := map[key]int64{}
	volume := map[key]int64{}
	pinned := map[key]int64{}
	low, high := map[int]economysim.Price{}, map[int]economysim.Price{}
	m := x4Measures{minHouseholdsBP: -1, minTradeShareBP: -1}
	for tick := int64(0); tick < run.Ticks; tick++ {
		s, ok := run.Samples[tick]
		if !ok {
			bad = append(bad, fmt.Sprintf("no snapshot after tick %d", tick))
			continue
		}
		bad = append(bad, x4Conservation(s)...)
		if tick < run.Ticks-run.Window {
			continue
		}
		if run.Start > 0 { // a start of 0 households has no share to fall from
			if bp := s.Households * 10_000 / run.Start; m.minHouseholdsBP < 0 || bp < m.minHouseholdsBP {
				m.minHouseholdsBP = bp
			}
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
	if m.minHouseholdsBP < 0 { // no start households: nothing fell
		m.minHouseholdsBP = 10_000
	}
	final, ok := run.Samples[run.Ticks-1]
	if !ok {
		return m, bad
	}
	name := func(k key) string {
		return final.Names.Regions[k.region] + "/" + final.Names.Commodities[k.commodity]
	}
	// (b): per commodity, the best market of its regions.
	best := map[int]int64{}
	for k, n := range traded {
		best[k.commodity] = max(best[k.commodity], n)
	}
	for c, id := range final.Names.Commodities {
		if !run.Producible[id] {
			continue
		}
		if bp := best[c] * 10_000 / run.Window; m.minTradeShareBP < 0 || bp < m.minTradeShareBP {
			m.minTradeShareBP, m.leastTraded = bp, id
		}
	}
	if m.minTradeShareBP < 0 { // no producible commodity: (b) is vacuous, not a failure at -1
		m.minTradeShareBP = 10_000
	}
	// (c): the mean volume of the markets that traded in the window.
	var total int64
	for _, v := range volume {
		total += v
		m.active++
	}
	if m.active > 0 {
		m.meanVolume = float64(total) / float64(int64(m.active)*run.Window)
	}
	// (d)
	for k, n := range pinned {
		if bp := n * 10_000 / run.Window; bp > m.maxPinnedBP || (bp == m.maxPinnedBP && m.mostPinned == "") {
			m.maxPinnedBP, m.mostPinned = bp, name(k)
		}
	}
	// (f)
	for c, hi := range high {
		if r := float64(hi) / float64(low[c]); r > m.maxRange {
			m.maxRange, m.widest = r, final.Names.Commodities[c]
		}
	}
	return m, bad
}

// x4Conservation returns the violations of the exact identities of criterion (a)
// on one snapshot (spec-ledger.md 9.1): per commodity, the units in accounts plus
// the boundary balances are 0, and so is the money of the accounts plus the money
// boundary.
func x4Conservation(s economysim.Snapshot) []string {
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
	var bad []string
	for c, v := range goods {
		if v != 0 {
			bad = append(bad, fmt.Sprintf("tick %d: commodity %q: units in accounts plus boundary = %d, want 0", s.Tick, s.Names.Commodities[c], v))
		}
	}
	if v := int64(s.Money) + money; v != 0 {
		bad = append(bad, fmt.Sprintf("tick %d: money of the accounts plus boundary = %d, want 0", s.Tick, v))
	}
	return bad
}

// flatRun builds a ten-tick run (window 5) of one region with two commodities,
// "a" producible and "b" not: every tick each market trades 5 units at 100,
// holds 10 belief outcomes with none at a bound, and the households stay at the
// 1 000 of the start, with no boundary and no money. edit may change a snapshot
// before it is stored.
func flatRun(edit func(tick int64, s *economysim.Snapshot)) stableRun {
	const ticks, window = 10, 5
	run := stableRun{Samples: map[int64]economysim.Snapshot{}, Start: 1_000, Producible: map[string]bool{"a": true}, Ticks: ticks, Window: window}
	for tick := int64(0); tick < ticks; tick++ {
		s := economysim.Snapshot{
			Tick: tick + 1, Households: 1_000, Names: economysim.Names{Regions: []string{"r"}, Commodities: []string{"a", "b"}},
			Markets: []economysim.MarketRow{
				{State: economysim.MarketState{Commodity: 0, Price: 100, Volume: 5}, Beliefs: 10},
				{State: economysim.MarketState{Commodity: 1, Price: 100, Volume: 5}, Beliefs: 10},
			},
		}
		if edit != nil {
			edit(tick, &s)
		}
		run.Samples[tick] = s
	}
	return run
}

// failing returns the keys of the criteria that do not pass, in order.
func failing(crit []x4Criterion) string {
	var keys string
	for _, c := range crit {
		if !c.Pass {
			keys += c.Key
		}
	}
	return keys
}

// TestScenarioHelpers_StablePredicate verifies the X4 predicate on hand-built
// series: a flat series passes (a) to (f) with the D363 priors, six variants each
// break exactly one criterion and the predicate names it, and a threshold set
// with no value is reported and not asserted.
//
// Window of 5 ticks, two markets. (a) one unit of money outside the boundary;
// (b) "a" never trades, a trade share of 0; (c) both markets trade 1 unit in 4 of
// the 5 ticks, a mean volume of 8 / (2 * 5) = 0.8 below 1 while each still trades
// in 8 000 bp of the ticks, the prior of (b); (d) 6 of 10 outcomes at the floor is
// more than half, a market pinned in every tick; (e) households at 800 of 1 000;
// (f) a price of 250 against 100 is a range of 2.5, above the prior 2. The last
// case breaks (b), (c) with no trade at all, (d), (e) and (f), and only the two
// whose thresholds are set (the households, and (c) which always is) are named.
//
// Requirement: FR-REF-027, FR-REF-042 (proposed)
func TestScenarioHelpers_StablePredicate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		edit func(tick int64, s *economysim.Snapshot)
		th   x4Thresholds
		want string // the keys that fail, in order
		not  string // the keys that are not asserted
	}{
		{"flat series passes", nil, x4Priors, "", ""},
		{"a: money outside the boundary", func(tick int64, s *economysim.Snapshot) {
			if tick == 3 {
				s.Money = 1
			}
		}, x4Priors, "a", ""},
		{"b: the producible commodity does not trade", func(tick int64, s *economysim.Snapshot) { s.Markets[0].State.Volume = 0 }, x4Priors, "b", ""},
		{"c: thin markets that still trade in 8 000 bp of the ticks", func(tick int64, s *economysim.Snapshot) {
			for i := range s.Markets {
				s.Markets[i].State.Volume = 1
				if tick%5 == 4 {
					s.Markets[i].State.Volume = 0
				}
			}
		}, x4Priors, "c", ""},
		{"d: a market pinned at the floor", func(tick int64, s *economysim.Snapshot) { s.Markets[1].AtFloor = 6 }, x4Priors, "d", ""},
		{"e: households fall to 800 bp of the start", func(tick int64, s *economysim.Snapshot) {
			if tick == 9 {
				s.Households = 800
			}
		}, x4Priors, "e", ""},
		{"f: the price range reaches 2.5", func(tick int64, s *economysim.Snapshot) {
			if tick == 9 {
				s.Markets[0].State.Price = 250
			}
		}, x4Priors, "f", ""},
		{"unset thresholds are reported, not asserted", func(tick int64, s *economysim.Snapshot) {
			s.Markets[0].State.Volume, s.Markets[1].State.Volume, s.Markets[1].AtFloor = 0, 0, 6
			s.Households = 800
		}, x4Thresholds{HouseholdsBP: x4Threshold{V: 9_000, Set: true}}, "ce", "bdf"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			crit := stablePredicate(flatRun(tt.edit), tt.th)
			if len(crit) != 6 {
				t.Fatalf("stablePredicate returned %d criteria, want 6", len(crit))
			}
			if got := failing(crit); got != tt.want {
				t.Errorf("stablePredicate failing criteria = %q, want %q: %+v", got, tt.want, crit)
			}
			var notAsserted string
			for _, c := range crit {
				if !c.Asserted {
					notAsserted += c.Key
				}
			}
			if notAsserted != tt.not {
				t.Errorf("stablePredicate unasserted criteria = %q, want %q", notAsserted, tt.not)
			}
		})
	}
}

// stableOneSkip is the reason TestScenarioStable_OneWarm is skipped while `one`
// collapses under its own rules (N-312, N-314); plan 2-8 Phase 4 empties it with
// the accepted rebalance, or the supervisor re-points the issue ID on a stop
// (D369).
const stableOneSkip = "N-314: `one` collapses under its own rules; un-skipped when the rebalance lands"

// TestScenarioStable_OneWarm records the run that a human reviews for pack `one`
// (STY 8.4): `one` warm, 1 095 ticks, seeds 1 to 4, audit on, no shock, judged by
// the X4 predicate at the D363 priors (which are priors, not bands: no value line
// is written to a bands file, FR-REF-024). It logs the realised table of plan 2-8
// D364 for every role and window of 100 ticks (sales and recipe-input cost per
// household-tick, the margin as basis points of the upkeep cost, unfed households,
// executions of each recipe), the households at ticks 100, 300, 600 and 1 095, and
// the last price of each market against its reference. The words "stable" and
// "stable state" are the predicate's name, not a claim: a pack passes the
// predicate, it is not stable (STY 7.3.2).
//
// Requirement: FR-REF-042 (proposed), FR-REF-039, FR-STB-015
//
// Fix: N-314. `one` fell from about 600 households to 20 to 100 by tick 600; this
// test fails on the Stage 1 pack and passes on the rebalanced one.
func TestScenarioStable_OneWarm(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test (1 095 ticks, four seeds), skipped with -short")
	}
	if raceEnabled {
		t.Skip("gate F runs it without -race (plan 2-1 D229)")
	}
	if stableOneSkip != "" {
		t.Skip(stableOneSkip)
	}
	cfg := loadValidPack(t, packOne...)
	probe, err := economysim.NewWorld(cfg)
	if err != nil {
		t.Fatalf("NewWorld error = %v, want nil", err)
	}
	s0, err := probe.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot error = %v, want nil", err)
	}
	t.Logf("X4 thresholds are the priors of plan 2-8 D363, not bands: %+v", x4Priors)
	for _, seed := range x4Seeds {
		t.Run("seed"+strconv.FormatUint(seed, 10), func(t *testing.T) {
			t.Parallel()
			obs := newIncomeObs(cfg, s0.Names)
			res := runPack(t, cfg, runSpec{Seed: seed, Ticks: x4Ticks, Drive: driveStep, Audit: true, Snapshots: snapEvery,
				Extra: []economysim.Option{economysim.WithObserver(obs)}})
			logRealised(t, cfg, res, obs, s0.Names)
			for _, c := range stablePredicate(newStableRun(cfg, res), x4Priors) {
				t.Logf("X4 one seed %d: (%s) %s: %s", seed, c.Key, c.Name, c.Detail)
				if !c.Pass {
					t.Errorf("X4 (%s) one seed %d: %s", c.Key, seed, c.Detail)
				}
			}
		})
	}
}

// logRealised logs the D364 table of one run: the households at the four ticks of
// N-312, the last price of each market against its reference price, and the income
// of each role in each window.
func logRealised(t *testing.T, cfg economysim.Config, res result, obs *incomeObs, names economysim.Names) {
	t.Helper()
	var hh []int64
	for _, tick := range []int64{100, 300, 600, x4Ticks} {
		hh = append(hh, res.Samples[tick-1].Households)
	}
	t.Logf("households at ticks 100, 300, 600, %d = %v (start %d)", x4Ticks, hh, newStableRun(cfg, res).Start)
	for _, m := range res.Samples[x4Ticks-1].Markets {
		ref := int64(res.World.ReferencePrice(m.State.Region, m.State.Commodity))
		if bp, ok := shareBP(int64(m.State.Price), ref); ok {
			t.Logf("price of %s/%s = %d bp of its reference %d", names.Regions[m.State.Region], names.Commodities[m.State.Commodity], bp, ref)
		}
	}
	// shortcut: region 0's basket for every role, right for the single-region pack one; use a per-region basket before reusing this for five.
	upkeep := int64(cfg.Regions[0].ReferenceBasket)
	for window := int64(0); window < x4Ticks/incomeWindow+1; window++ {
		for role, id := range names.Roles {
			row := obs.Row(window, economysim.RoleID(role), upkeep)
			if !row.OK {
				continue
			}
			c := obs.cells[[2]int64{window, int64(role)}]
			t.Logf("window %d role %s: sales %d and recipe inputs %d per household-tick, margin %d bp of the upkeep %d, unfed %d household-ticks, executions %s",
				window, id, row.SalesPerHousehold, row.InputCostPerHousehold, row.MarginBP, upkeep, c.Unfed, executionsOf(c, names))
		}
	}
}

// executionsOf renders the executions of a cell by recipe name, in recipe order.
func executionsOf(c *incomeCell, names economysim.Names) string {
	var ids []economysim.RecipeID
	for r := range c.Executions {
		ids = append(ids, r)
	}
	slices.Sort(ids)
	out := ""
	for _, r := range ids {
		out += fmt.Sprintf("%s=%d ", names.Recipes[r], c.Executions[r])
	}
	return out
}
