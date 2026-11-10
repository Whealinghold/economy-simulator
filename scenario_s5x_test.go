package economysim_test

import (
	"fmt"
	"reflect"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/strategy"
)

// This file holds the scenarios of the external module that run a pack (spec-stability.md
// 8, S51, S55 and S71; plan 2-16 Phase 7, D539), the fork of the module-alone pack and
// its determinism row. S52 to S54 are built in Go on small worlds and are in
// scenario_s5x_internal_test.go. A run is short (150 ticks) and uses the first seeds only:
// the identities hold whatever the economy does, and gate F carries the time.

// outsideBalances returns the balance of the boundary accounts of the six reasons of
// the external module, by reason and commodity (NoCommodity for the money reasons).
func outsideBalances(s economysim.Snapshot) map[economysim.Reason]map[economysim.CommodityID]int64 {
	out := map[economysim.Reason]map[economysim.CommodityID]int64{}
	for _, r := range s.Boundary {
		switch r.Reason {
		case "outside_budget", "outside_unspent", "outside_receipts", "outside_offer", "outside_unsold", "outside_purchases":
			if out[r.Reason] == nil {
				out[r.Reason] = map[economysim.CommodityID]int64{}
			}
			out[r.Reason][r.Commodity] += r.Balance
		}
	}
	return out
}

// checkOutsideIdentities asserts the identities of spec-external-sector.md 8.2 on a
// snapshot and the events up to it, with exact equality: the units imported and
// exported per commodity equal the goods reasons and the sides' units, and the money
// of the buy sides' trades equals the budget less the unspent, and the sell sides'
// equals the receipts.
func checkOutsideIdentities(t *testing.T, seed uint64, s economysim.Snapshot, events []economysim.Event) {
	t.Helper()
	b := outsideBalances(s)
	for _, tr := range s.Outside.Trade {
		imports := -b["outside_offer"][tr.Commodity] - b["outside_unsold"][tr.Commodity]
		exports := b["outside_purchases"][tr.Commodity]
		if int64(tr.Imports) != imports || int64(tr.Exports) != exports {
			t.Errorf("seed %d tick %d commodity %d: Trade imports %d exports %d, boundary says %d and %d (FR-EXT-019)", seed, s.Tick, tr.Commodity, tr.Imports, tr.Exports, imports, exports)
		}
	}
	var exportValue, importValue int64
	for _, e := range events {
		if v, ok := e.(economysim.OutsideTraded); ok {
			if v.Side == economysim.Bid {
				exportValue += int64(v.Value)
			} else {
				importValue += int64(v.Value)
			}
		}
	}
	nc := economysim.NoCommodity
	if got := -b["outside_budget"][nc] - b["outside_unspent"][nc]; got != exportValue {
		t.Errorf("seed %d tick %d: export value from the boundary = %d, from the events = %d (FR-EXT-019)", seed, s.Tick, got, exportValue)
	}
	if got := b["outside_receipts"][nc]; got != importValue {
		t.Errorf("seed %d tick %d: import value from the boundary = %d, from the events = %d (FR-EXT-019)", seed, s.Tick, got, importValue)
	}
	for _, l := range s.Outside.Lines {
		if l.Mid < 1 || l.Mid > economysim.MaxPrice {
			t.Errorf("seed %d tick %d: mid %d of line %d outside [1, MaxPrice] (NFR-EXT-004)", seed, s.Tick, l.Mid, l.Commodity)
		}
	}
}

// TestScenarioS55_InvariantRun (EXT-S5) runs pack one with the external overlay over the
// first seeds (one under -race), 150 ticks with WithAudit, which holds the counterparties
// empty at every boundary and the sectoral identity of LED 9.3 at every step, and checks
// at the end the identities of EXT 8.2 with exact equality, the mid
// within [1, MaxPrice] and the drift within its bound. This is the module-alone entry
// of the invariants matrix of REF 7.5: the generic checker of that matrix treats a
// boundary money flow after construction as a violation, which this module makes by
// design, so the module has its own checker.
//
// Requirement: FR-EXT-019, FR-EXT-020, NFR-EXT-004
//
// Guard: an order or a sweep that created or lost a unit or a coin would break the
// equality at once; the seed is in the message.
func TestScenarioS55_InvariantRun(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := loadValidPack(t, packExternal()...)
	bound := cfg.Modules.External.Nodes[0].Lines[0].WalkBoundBP
	for _, seed := range seedList(invariantSeeds(3)) {
		t.Run(fmt.Sprintf("seed %d", seed), func(t *testing.T) {
			t.Parallel()
			res := runPack(t, cfg, runSpec{Seed: seed, Ticks: 150, Drive: driveStep, Observe: true, Audit: true, Snapshots: snapEvery})
			for tick, s := range res.Samples {
				for _, l := range s.Outside.Lines {
					if l.DriftBP < -bound || l.DriftBP > bound {
						t.Errorf("seed %d tick %d: drift %d outside ±%d (FR-EXT-022)", seed, tick, l.DriftBP, bound)
					}
				}
			}
			checkOutsideIdentities(t, seed, res.Final, res.Events)
		})
	}
}

// TestScenarioS71_GateRegionWithMerchants (MER-S5, FR-MER-049) runs the S71 stack for 40
// ticks with the audit on: pack three with the trade, transport, information, merchants
// and external overlays, the merchant strategy registered, an outside node `outer` on the
// gate `far_south`. It asserts that no shipment, contract or offer names the node
// (neither as origin nor as destination), that every ShipmentRejected toward the node,
// if any, is `outside_destination`, that the outside's quantity per tick never exceeds
// its capacity (it is shared by whoever trades at the gate), and the identities of EXT
// 8.2. That the merchants buy imports at the gate and sell inland is reported only: the
// cold pack's households die in the first ticks (the golden of the merchants shows it).
//
// Requirement: FR-MER-049, FR-EXT-013, FR-EXT-019
//
// Guard: a candidate lane toward a node would be refused only by the admission; this
// test pins that none is planned and that the node is never a lane end.
func TestScenarioS71_GateRegionWithMerchants(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := loadValidPack(t, packExternalThree()...)
	const seed = 1
	res := runPack(t, cfg, runSpec{Seed: seed, Ticks: 40, Drive: driveStep, Observe: true, Audit: true, Extra: strategy.Options()})
	outer := economysim.RegionID(slices.Index(res.Final.Names.Regions, "outer"))
	if outer < 0 {
		t.Fatalf("the names hold no region outer: %v", res.Final.Names.Regions)
	}
	capacity := map[economysim.CommodityID]int64{}
	for _, l := range cfg.Modules.External.Nodes[0].Lines {
		c := economysim.CommodityID(slices.Index(res.Final.Names.Commodities, l.Commodity))
		capacity[c] = max(l.Sell.Capacity, l.Buy.Capacity)
	}
	for _, e := range res.Events {
		switch v := e.(type) {
		case economysim.ShipmentDispatched:
			if v.Region == outer || v.To == outer {
				t.Errorf("seed %d tick %d: ShipmentDispatched %+v names the node", seed, v.Tick, v)
			}
		case economysim.ContractFormed:
			if v.Region == outer || v.To == outer {
				t.Errorf("seed %d tick %d: ContractFormed %+v names the node", seed, v.Tick, v)
			}
		case economysim.ShipmentRejected:
			if v.To == outer && v.Reason != economysim.ReasonOutsideDestination {
				t.Errorf("seed %d tick %d: ShipmentRejected toward the node with reason %q, want outside_destination", seed, v.Tick, v.Reason)
			}
		case economysim.OutsideTraded:
			if v.Traded > capacity[v.Commodity] {
				t.Errorf("seed %d tick %d: outside traded %d of commodity %d, above its capacity %d", seed, v.Tick, v.Traded, v.Commodity, capacity[v.Commodity])
			}
		}
	}
	checkOutsideIdentities(t, seed, res.Final, res.Events)
	t.Logf("seed %d: %d OutsideTraded, %d ShipmentDispatched, %d ContractFormed (reported, no band: the pack is cold)", seed,
		count[economysim.OutsideTraded](res.Events), count[economysim.ShipmentDispatched](res.Events), count[economysim.ContractFormed](res.Events))
}

// count returns the number of events of type T.
func count[T economysim.Event](events []economysim.Event) int {
	n := 0
	for _, e := range events {
		if _, ok := e.(T); ok {
			n++
		}
	}
	return n
}

// TestScenarioS51_CorridorReport (EXT-S1) runs pack one with and without the external
// overlay for 150 ticks and reports the price range of the staple, the balance of trade
// and the share of ticks at a quote. It asserts only that both runs completed with the
// audit on and that the outside traded: the bands are the stability register's
// (t03.pull_bp, N-113).
//
// Requirement: FR-EXT-049
func TestScenarioS51_CorridorReport(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	with := runPack(t, loadValidPack(t, packExternal()...), runSpec{Seed: 1, Ticks: 150, Drive: driveStep, Observe: true, Audit: true})
	without := runPack(t, loadValidPack(t, packLayers["one"]...), runSpec{Seed: 1, Ticks: 150, Drive: driveStep, Observe: true, Audit: true})
	staple := economysim.CommodityID(slices.Index(with.Final.Names.Commodities, "staple"))
	rng := func(r result) (lo, hi economysim.Price) {
		lo = economysim.MaxPrice
		for _, e := range r.Events {
			if v, ok := e.(economysim.TradeExecuted); ok && v.Commodity == staple {
				lo, hi = min(lo, v.Price), max(hi, v.Price)
			}
		}
		return lo, hi
	}
	lo1, hi1 := rng(with)
	lo0, hi0 := rng(without)
	atQuote := 0
	for _, e := range with.Events {
		if v, ok := e.(economysim.OutsideTraded); ok && v.Traded > 0 && v.Commodity == staple {
			atQuote++
		}
	}
	b := outsideBalances(with.Final)
	t.Logf("staple trade price range with the outside %d..%d, without %d..%d; balance of trade %d; ticks with a staple export %d of 150 (reported, no band)",
		lo1, hi1, lo0, hi0, -b["outside_budget"][economysim.NoCommodity]-b["outside_unspent"][economysim.NoCommodity]-b["outside_receipts"][economysim.NoCommodity], atQuote)
	if count[economysim.OutsideTraded](with.Events) == 0 || count[economysim.OutsideTraded](without.Events) != 0 {
		t.Errorf("OutsideTraded with the outside %d and without %d, want some and none", count[economysim.OutsideTraded](with.Events), count[economysim.OutsideTraded](without.Events))
	}
}

// TestFork_External verifies FR-EXT-016 on the module-alone pack through the fork-at-N of
// CHK 8.1, seed 1, 10 ticks after the boundary: at N = 0 (before the first step) and at
// N = 100 (inside a job, with the walk, the carry and the cumulative units of the sides
// in tag 18) the original, the restored world and the restore of a restore agree on
// events, snapshot, tick and checkpoint bytes after every step. The window, the closed
// season and the ended side are the internal TestFork_ExternalRestore's.
//
// Requirement: FR-EXT-016
func TestFork_External(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := loadValidPack(t, packExternal()...)
	for _, n := range []int{0, 100} {
		t.Run(fmt.Sprintf("N %d", n), func(t *testing.T) {
			t.Parallel()
			forkAtN(t, forkCase{Name: "external", Cfg: cfg, Seed: goldenSeed, N: n, M: 10})
		})
	}
}

// TestDeterminism_ExternalAlone verifies the determinism row of REF 8.2 for pack one with
// the external overlay, seed 1, 400 ticks: a run split into Run calls, with a logger, an
// observer or the audit, ends in the snapshot and the state hash of the bare Step run,
// and its events equal the reference's.
//
// Requirement: NFR-EXT-004, FR-OBS-023, FR-WLD-063
//
// Guard: the walk's draws and the carries are state; a logger, an observer or the
// chunking of Run that moved them would fail here.
func TestDeterminism_ExternalAlone(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := loadValidPack(t, packExternal()...)
	seed := uint64(1)
	base := runPack(t, cfg, detStep(runSpec{Seed: seed, Ticks: detTicks}))
	ref := runPack(t, cfg, detStep(runSpec{Seed: seed, Ticks: detTicks, Observe: true}))
	refEvents := renderEvents(withoutRunBrackets(ref))
	if d := firstTickDiff(base.Samples, ref.Samples); d != "" {
		t.Fatalf("seed %d: observed Step run differs from the bare one at %s (FR-OBS-023)", seed, d)
	}
	for _, tc := range detCases(true) {
		spec := tc.spec()
		spec.Seed, spec.Ticks = seed, detTicks
		got := runPack(t, cfg, spec)
		if !reflect.DeepEqual(got.Final, base.Final) {
			t.Errorf("seed %d, case %q: final snapshot differs from the bare run", seed, tc.name)
		}
		if economysim.HashWorld(got.World) != economysim.HashWorld(base.World) {
			t.Errorf("seed %d, case %q: state hash of the world differs from the bare run", seed, tc.name)
		}
		if spec.Observe {
			if d := firstLineDiff(refEvents, renderEvents(withoutRunBrackets(got))); d != "" {
				t.Errorf("seed %d, case %q: events differ from the reference at %s", seed, tc.name, d)
			}
		}
	}
}
