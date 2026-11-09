package indicator_test

import (
	"errors"
	"os"
	"reflect"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/config"
	"github.com/Whealinghold/economy-sim/indicator"
)

// observedWorld returns a world of the given config files (under testdata/chains)
// with the tally as an observer, and its configuration.
func observedWorld(t *testing.T, tl *indicator.Tally, files ...string) (*economysim.World, economysim.Config) {
	t.Helper()
	cfg, err := config.LoadFS(os.DirFS("../testdata/chains"), files...)
	if err != nil {
		t.Fatalf("LoadFS(%v) error = %v, want nil", files, err)
	}
	w, err := economysim.NewWorld(cfg, economysim.WithSeed(1), economysim.WithObserver(tl))
	if err != nil {
		t.Fatalf("NewWorld(%v) error = %v, want nil", files, err)
	}
	return w, cfg
}

// snapshotOf takes a snapshot and fails the test on an error.
func snapshotOf(t *testing.T, w *economysim.World) economysim.Snapshot {
	t.Helper()
	s, err := w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot() error = %v, want nil", err)
	}
	return s
}

// TestTally_IgnoresWhatItDoesNotUse verifies that a tally ignores the per-order
// and per-trade classes and the events no preset reads while a window is open,
// and every event while no window is open, so a consumer pays nothing for the
// classes that are more than 99% of a world's events.
//
// Requirement: FR-IND-005
func TestTally_IgnoresWhatItDoesNotUse(t *testing.T) {
	unused := []economysim.Event{
		economysim.OrderPlaced{Tick: 10}, economysim.OrderRejected{Tick: 10}, economysim.TradeExecuted{Tick: 10},
		economysim.WearApplied{Tick: 10, Commodity: grain, Units: 3}, economysim.EdgeStatusChanged{Tick: 10},
	}
	tl := indicator.NewTally()
	// Closed: even events the tally reads are dropped.
	for _, e := range append([]economysim.Event{
		flow(10, alpha, economysim.ReasonConsumption, grain, 5),
		economysim.AgentSpawned{Tick: 10, Region: alpha, Role: grower, Weight: 9},
		economysim.MarketCleared{Tick: 10, Region: alpha, Commodity: grain, Price: 5, Volume: 5},
	}, unused...) {
		tl.Observe(e)
	}
	openValley(t, tl, 10, [4]int64{40, 10, 2, 0})
	step(tl, 10, unused...)
	w, err := tl.Close(valleySnap(11, [4]int64{40, 10, 2, 0}))
	if err != nil {
		t.Fatalf("Close() error = %v, want nil", err)
	}
	if len(w.Flows) != 0 || len(w.Outputs) != 0 || len(w.Relocations) != 0 {
		t.Errorf("sparse rows = flows %d outputs %d relocations %d, want none", len(w.Flows), len(w.Outputs), len(w.Relocations))
	}
	for i, m := range w.Markets {
		if m != (indicator.MarketTally{}) {
			t.Errorf("Markets[%d] = %+v, want zero", i, m)
		}
	}
	for i, p := range w.People {
		if p.Spawned != 0 || p.Households != w.OpenHouseholds[i] {
			t.Errorf("People[%d] = %+v, want no change from the opening households %d", i, p, w.OpenHouseholds[i])
		}
	}
}

// rowCounts is the number of rows a window holds in every table.
func rowCounts(w indicator.Window) [11]int {
	return [11]int{len(w.Flows), len(w.Construction), len(w.Markets), len(w.Beliefs), len(w.Recipe), len(w.Outputs),
		len(w.People), len(w.Regional), len(w.Relocations), len(w.OpenHouseholds), len(w.ClosePending)}
}

// TestTally_SizeIsAFunctionOfTheConfiguration verifies that the same
// configuration gives the same row counts whether 120 or 2 400 agents spawn,
// trade, produce and relocate in the window: the tally's size does not grow with
// agents, orders or trades.
//
// Requirement: FR-IND-004
func TestTally_SizeIsAFunctionOfTheConfiguration(t *testing.T) {
	run := func(agents int) indicator.Window {
		tl := indicator.NewTally()
		openValley(t, tl, 0, [4]int64{})
		for tick := int64(0); tick < 3; tick++ {
			var evs []economysim.Event
			for a := 0; a < agents/3; a++ {
				r, role := economysim.RegionID(a%2), economysim.RoleID(a%2)
				evs = append(evs,
					economysim.AgentSpawned{Tick: tick, Agent: economysim.AgentID(a), Region: r, Role: role, Weight: 1},
					economysim.TradeExecuted{Tick: tick},
					flow(tick, r, economysim.ReasonConsumption, grain, 1),
					economysim.RecipeExecuted{Tick: tick, Region: r, Recipe: recGrow, Executions: 1,
						Outputs: []economysim.Amount{{Commodity: grain, Quantity: 2}}},
					economysim.AgentRelocated{Tick: tick, Agent: economysim.AgentID(a), Role: role, From: r, To: 1 - r, Weight: 1},
				)
			}
			step(tl, tick, evs...)
		}
		w, err := tl.Close(valleySnap(3, [4]int64{}))
		if err != nil {
			t.Fatalf("Close() with %d agents error = %v, want nil", agents, err)
		}
		return w
	}
	small, large := run(120), run(2400)
	if got, want := rowCounts(large), rowCounts(small); got != want {
		t.Errorf("row counts at 2400 agents = %v, want %v (those at 120 agents)", got, want)
	}
	if small.People[0].Spawned == large.People[0].Spawned {
		t.Errorf("People[0].Spawned equal at both sizes (%d): the run did not scale", small.People[0].Spawned)
	}
}

// TestTally_ObserveAllocatesNothing verifies that Observe allocates nothing for
// an event of a row that already exists, so a consumer that wants every preset
// every tick pays no garbage for the tally.
//
// Requirement: NFR-IND-002
func TestTally_ObserveAllocatesNothing(t *testing.T) {
	tl := indicator.NewTally()
	openValley(t, tl, 0, [4]int64{5, 5, 5, 5})
	events := []economysim.Event{
		economysim.TickStarted{Tick: 0},
		flow(0, alpha, economysim.ReasonConsumption, grain, 3),
		flow(-1, alpha, economysim.ReasonInitial, grain, 3),
		economysim.MarketCleared{Tick: 0, Region: beta, Commodity: wood, Price: 8, Volume: 4},
		economysim.BeliefsUpdated{Tick: 0, Region: alpha, Commodity: grain, Outcomes: 5, MeanCentre: 10, LowCentre: 9, HighCentre: 12},
		economysim.RecipeExecuted{Tick: 0, Region: alpha, Recipe: recGrow, Executions: 2, Outputs: []economysim.Amount{{Commodity: grain, Quantity: 4}}, Disposed: []economysim.Amount{{Commodity: grain, Quantity: 1}}},
		economysim.AgentSpawned{Tick: 0, Region: alpha, Role: grower, Weight: 1},
		economysim.AgentBankrupted{Tick: 0, Region: alpha, Role: grower, Weight: 1},
		economysim.WeightChanged{Tick: 0, Region: alpha, Role: grower, From: 1, To: 2},
		economysim.RoleChanged{Tick: 0, Region: alpha, From: grower, To: baker, Weight: 1},
		economysim.AgentRelocated{Tick: 0, Role: baker, From: alpha, To: beta, Weight: 1, Money: 4, Units: 2},
		economysim.UpkeepShortfall{Tick: 0, Region: alpha, Role: grower, Unfed: 2, MaxHunger: 3},
		economysim.PopulationChanged{Tick: 0, Region: alpha, Role: grower, BirthsDone: 1, DeathsDone: 1},
		economysim.EntryLimited{Tick: 0, Region: alpha, RegionCap: 1},
		economysim.MoneyFlowLimited{Tick: 0, Region: alpha, Short: 1},
		economysim.ReliefPaid{Tick: 0, Region: alpha, Paid: 5},
		economysim.TradeExecuted{Tick: 0},
		economysim.TickEnded{Tick: 0},
	}
	for _, e := range events { // create every row once
		tl.Observe(e)
	}
	if got := testing.AllocsPerRun(50, func() {
		for _, e := range events {
			tl.Observe(e)
		}
	}); got != 0 {
		t.Errorf("Observe of events of existing rows allocated %v times per pass, want 0", got)
	}
}

// TestClose_WindowOrder verifies that Close refuses a window whose events do not
// cover its steps exactly once.
//
// Requirement: FR-IND-006
func TestClose_WindowOrder(t *testing.T) {
	hh := [4]int64{}
	tests := []struct {
		name  string
		feed  func(*indicator.Tally)
		close int64
	}{
		{"missing TickEnded", func(tl *indicator.Tally) { tl.Observe(economysim.TickStarted{Tick: 10}) }, 11},
		{"two TickEnded for one step", func(tl *indicator.Tally) { step(tl, 10); tl.Observe(economysim.TickEnded{Tick: 11}) }, 11},
		{"a step that is not consecutive", func(tl *indicator.Tally) { step(tl, 10); step(tl, 12) }, 13},
		{"a window that starts late", func(tl *indicator.Tally) { step(tl, 11) }, 12},
		{"a failed step", func(tl *indicator.Tally) { tl.Observe(economysim.TickEnded{Tick: 10, Failed: true}) }, 11},
		{"a failed run", func(tl *indicator.Tally) { step(tl, 10); tl.Observe(economysim.RunEnded{Tick: 11, Failed: true}) }, 11},
		{"an event before the window", func(tl *indicator.Tally) { step(tl, 10, flow(9, alpha, economysim.ReasonConsumption, grain, 1)) }, 11},
		{"a step event at the close", func(tl *indicator.Tally) {
			step(tl, 10)
			tl.Observe(economysim.MarketCleared{Tick: 11, Region: alpha, Commodity: grain})
		}, 11},
		{"a call event past the close", func(tl *indicator.Tally) {
			step(tl, 10)
			tl.Observe(economysim.AgentSpawned{Tick: 12, Region: alpha, Role: grower, Weight: 1})
		}, 11},
		{"an ID outside the names table", func(tl *indicator.Tally) {
			step(tl, 10, economysim.AgentSpawned{Tick: 10, Region: 5, Role: grower, Weight: 1})
		}, 11},
		{"open tick not below the close tick", func(tl *indicator.Tally) {}, 10},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tl := indicator.NewTally()
			openValley(t, tl, 10, hh)
			tt.feed(tl)
			if _, err := tl.Close(valleySnap(tt.close, hh)); !errors.Is(err, indicator.ErrWindowOrder) {
				t.Errorf("Close(%d) error = %v, want ErrWindowOrder", tt.close, err)
			}
		})
	}
	t.Run("a call event at the close is part of the window", func(t *testing.T) {
		tl := indicator.NewTally()
		openValley(t, tl, 10, hh)
		step(tl, 10, economysim.AgentSpawned{Tick: 10, Region: alpha, Role: grower, Weight: 1})
		// A call after the last step carries the next step's tick, which is the close.
		tl.Observe(economysim.AgentSpawned{Tick: 11, Region: alpha, Role: grower, Weight: 2})
		w, err := tl.Close(valleySnap(11, [4]int64{3, 0, 0, 0}))
		if err != nil {
			t.Fatalf("Close() error = %v, want nil", err)
		}
		if got := w.People[0].Households; got != 3 {
			t.Errorf("People[0].Households = %d, want 3 (1 in the step and 2 by the call after it)", got)
		}
	})
	t.Run("no window open", func(t *testing.T) {
		if _, err := indicator.NewTally().Close(valleySnap(11, hh)); !errors.Is(err, indicator.ErrWindowOrder) {
			t.Errorf("Close() on a closed tally error = %v, want ErrWindowOrder", err)
		}
	})
}

// TestClose_ConfigMismatch verifies that Close refuses a snapshot of another
// configuration, leaves the window open, and accepts the right snapshot after.
//
// Requirement: FR-IND-006
func TestClose_ConfigMismatch(t *testing.T) {
	tl := indicator.NewTally()
	openValley(t, tl, 10, [4]int64{})
	step(tl, 10)
	other := valleySnap(11, [4]int64{})
	other.ConfigHash = [32]byte{9}
	if _, err := tl.Close(other); !errors.Is(err, indicator.ErrConfigMismatch) {
		t.Fatalf("Close(other world) error = %v, want ErrConfigMismatch", err)
	}
	w, err := tl.Close(valleySnap(11, [4]int64{}))
	if err != nil {
		t.Fatalf("Close(right snapshot) after a refusal error = %v, want nil", err)
	}
	if w.ConfigHash != valleyHash || w.From != 10 || w.To != 11 || w.Steps != 1 {
		t.Errorf("window = hash %x from %d to %d steps %d, want the valley's, 10, 11, 1", w.ConfigHash[:2], w.From, w.To, w.Steps)
	}
}

// TestClose_Overflow verifies that a sum past int64 is reported by Close and
// not wrapped.
//
// Requirement: FR-IND-009
func TestClose_Overflow(t *testing.T) {
	tl := indicator.NewTally()
	openValley(t, tl, 10, [4]int64{})
	big := int64(1) << 62
	step(tl, 10, flow(10, alpha, economysim.ReasonConsumption, grain, big), flow(10, alpha, economysim.ReasonConsumption, grain, big), flow(10, alpha, economysim.ReasonConsumption, grain, big))
	if _, err := tl.Close(valleySnap(11, [4]int64{})); !errors.Is(err, indicator.ErrOverflow) {
		t.Errorf("Close() error = %v, want ErrOverflow", err)
	}
}

// TestTally_ConstructionBucketIsAMemoLine verifies that the BoundaryFlow rows of
// Tick -1 are kept apart and are not in the window's flows, and that `initial`
// at tick 0 or later is in the flows (the Endow term of the accounts).
//
// Requirement: FR-IND-007
func TestTally_ConstructionBucketIsAMemoLine(t *testing.T) {
	tl := indicator.NewTally()
	openValley(t, tl, 0, [4]int64{})
	tl.Observe(flow(-1, alpha, economysim.ReasonInitial, grain, 200))
	step(tl, 0, flow(0, alpha, economysim.ReasonInitial, grain, 5))
	w, err := tl.Close(valleySnap(1, [4]int64{}))
	if err != nil {
		t.Fatalf("Close() error = %v, want nil", err)
	}
	if got := w.Construction; len(got) != 1 || got[0].Total != 200 {
		t.Errorf("Construction = %v, want the one memo row of 200", got)
	}
	flows, err := w.Corrected()
	if err != nil || len(flows) != 1 || flows[0].Total != 5 || flows[0].Reason != economysim.ReasonInitial {
		t.Errorf("Corrected() = %v, %v, want the one initial row of 5 and no construction", flows, err)
	}
}

// TestTally_RunningHouseholdsEqualTheSnapshot verifies, at every boundary of a
// run, that the households the tally rebuilt from the events of the step equal
// the households of the snapshot, for a pack with a role change, bankruptcies
// and weight changes and for one with the population module (births, deaths).
//
// Requirement: FR-IND-011
func TestTally_RunningHouseholdsEqualTheSnapshot(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	packs := map[string][]string{
		"one":            {"base.toml", "one.toml", "warm-one.toml"},
		"one+population": {"base.toml", "one.toml", "warm-one.toml", "overlay/population.toml"},
	}
	changes := int64(0)
	for name, files := range packs {
		t.Run(name, func(t *testing.T) {
			tl := indicator.NewTally()
			w, _ := observedWorld(t, tl, files...)
			prev := snapshotOf(t, w)
			if err := tl.Open(prev); err != nil {
				t.Fatalf("Open() error = %v, want nil", err)
			}
			for tick := 0; tick < 60; tick++ {
				if err := w.Step(); err != nil {
					t.Fatalf("Step() at %d error = %v, want nil", tick, err)
				}
				s := snapshotOf(t, w)
				win, err := tl.Close(s)
				if err != nil {
					t.Fatalf("Close() at %d error = %v, want nil", s.Tick, err)
				}
				for _, r := range s.Roles {
					i := int(r.Region)*win.Roles + int(r.Role)
					if got := win.People[i].Households; got != r.Households {
						t.Fatalf("tick %d: running households of (%d,%d) = %d, want %d (the snapshot's)", s.Tick, r.Region, r.Role, got, r.Households)
					}
					if win.CloseHouseholds[i] != r.Households {
						t.Fatalf("tick %d: CloseHouseholds of (%d,%d) = %d, want %d", s.Tick, r.Region, r.Role, win.CloseHouseholds[i], r.Households)
					}
					p := win.People[i]
					changes += p.Spawned + p.Bankrupted + p.RoleIn + p.RoleOut + p.BirthsDone + p.DeathsDone
					if p.WeightDelta != 0 {
						changes++
					}
				}
				if err := tl.Open(s); err != nil {
					t.Fatalf("Open() at %d error = %v, want nil", s.Tick, err)
				}
			}
		})
	}
	if changes == 0 {
		t.Errorf("no household changed in either run: the check was vacuous")
	}
}

// TestFlows_PendingCorrection verifies that the flows of a window are the
// tally's rows less the opening pending plus the closing pending: a call before
// Open (in the opening balances and in the first tick's flow) is not counted
// again, and one after the last step (in the closing balances and in no flow)
// is. Pending-only rows appear.
//
// Requirement: FR-IND-016
func TestFlows_PendingCorrection(t *testing.T) {
	pend := func(r economysim.RegionID, reason economysim.Reason, c economysim.CommodityID, n int64) economysim.FlowTotal {
		return economysim.FlowTotal{Region: r, Reason: reason, Kind: economysim.KindFaucet, Commodity: c, Total: n}
	}
	open := valleySnap(10, [4]int64{})
	open.Pending = []economysim.FlowTotal{pend(alpha, economysim.ReasonInitial, grain, 5)}
	closing := valleySnap(11, [4]int64{})
	closing.Pending = []economysim.FlowTotal{pend(alpha, economysim.ReasonInitial, grain, 2), pend(beta, economysim.ReasonInitial, wood, 7)}
	tl := indicator.NewTally()
	if err := tl.Open(open); err != nil {
		t.Fatalf("Open() error = %v, want nil", err)
	}
	// The first tick's flow carries the 5 of the call before Open besides its own 1.
	step(tl, 10, flow(10, alpha, economysim.ReasonInitial, grain, 6))
	w, err := tl.Close(closing)
	if err != nil {
		t.Fatalf("Close() error = %v, want nil", err)
	}
	got, err := w.Corrected()
	if err != nil {
		t.Fatalf("Corrected() error = %v, want nil", err)
	}
	// alpha grain: 6 - 5 + 2 = 3; beta wood: 0 - 0 + 7 = 7 (no event carried it).
	want := []economysim.FlowTotal{pend(alpha, economysim.ReasonInitial, grain, 3), pend(beta, economysim.ReasonInitial, wood, 7)}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Corrected() = %v, want %v", got, want)
	}
	if w.Flows[0].Total != 6 {
		t.Errorf("Flows[0].Total = %d, want 6: the raw rows are as the events carried them", w.Flows[0].Total)
	}
}

// marketDay is one clearing of a market.
func marketDay(tick int64, r economysim.RegionID, c economysim.CommodityID, price economysim.Price, volume economysim.Quantity) economysim.MarketCleared {
	return economysim.MarketCleared{Tick: tick, Region: r, Commodity: c, Price: price, Volume: volume}
}

// TestPricesOfWindow_VolumeWeighted verifies the window unit value of 18.1:
// rdiv(10*30 + 11*20, 50) = rdiv(520, 50) = 10 (10.4 rounds to 10), the last
// published price where no unit traded, and the hash and ticks of the window.
//
// Requirement: FR-IND-028
func TestPricesOfWindow_VolumeWeighted(t *testing.T) {
	tl := indicator.NewTally()
	openValley(t, tl, 10, [4]int64{})
	step(tl, 10, marketDay(10, alpha, grain, 10, 30), marketDay(10, alpha, wood, 6, 0), marketDay(10, beta, loaf, 28, 5))
	step(tl, 11, marketDay(11, alpha, grain, 11, 20), marketDay(11, alpha, wood, 7, 0), marketDay(11, beta, loaf, 29, 0))
	w, err := tl.Close(valleySnap(12, [4]int64{}))
	if err != nil {
		t.Fatalf("Close() error = %v, want nil", err)
	}
	p, err := indicator.PricesOfWindow(w)
	if err != nil {
		t.Fatalf("PricesOfWindow() error = %v, want nil", err)
	}
	tests := []struct {
		name string
		r    economysim.RegionID
		c    economysim.CommodityID
		want economysim.Price
	}{
		{"weighted mean rounded half to even", alpha, grain, 10},
		{"no unit traded: the last published price", alpha, wood, 7},
		{"one tick with volume: its price", beta, loaf, 28},
		{"no clearing seen: 0", beta, cart, 0},
	}
	for _, tt := range tests {
		if got := p.At(tt.r, tt.c); got != tt.want {
			t.Errorf("PricesOfWindow().At(%d, %d) = %d, want %d (%s)", tt.r, tt.c, got, tt.want, tt.name)
		}
	}
	if p.ConfigHash != valleyHash || p.From != 10 || p.To != 12 {
		t.Errorf("table = hash %x from %d to %d, want the valley's, 10, 12", p.ConfigHash[:2], p.From, p.To)
	}
	if err := p.Compatible(valleyHash); err != nil {
		t.Errorf("Compatible(window hash) = %v, want nil", err)
	}
}

// TestBaseOf_FreezesPricesAndQuantities verifies that a base holds the table
// and the quantities of the window it was taken from, for each choice of
// weights, and refuses a table of another world or shape.
//
// Requirement: FR-IND-020
func TestBaseOf_FreezesPricesAndQuantities(t *testing.T) {
	tl := indicator.NewTally()
	openValley(t, tl, 10, [4]int64{})
	step(tl, 10, marketDay(10, alpha, grain, 10, 30),
		flow(10, alpha, economysim.ReasonConsumption, grain, 10), flow(10, alpha, economysim.ReasonWantConsumption, grain, 2),
		flow(10, alpha, economysim.ReasonRecipeOutput, grain, 100))
	w, err := tl.Close(valleySnap(11, [4]int64{}))
	if err != nil {
		t.Fatalf("Close() error = %v, want nil", err)
	}
	p, _ := indicator.PricesOfWindow(w)
	b, err := indicator.BaseOf(w, p)
	if err != nil {
		t.Fatalf("BaseOf() error = %v, want nil", err)
	}
	for _, tt := range []struct {
		k    indicator.Weights
		want int64
	}{{indicator.WeightsTraded, 30}, {indicator.WeightsConsumed, 12}, {indicator.WeightsProduced, 100}} {
		if got := b.Quantities(tt.k)[int(alpha)*4+int(grain)]; got != tt.want {
			t.Errorf("Base.Quantities(%v)[alpha grain] = %d, want %d", tt.k, got, tt.want)
		}
	}
	if b.Quantities(indicator.Weights(99)) != nil {
		t.Errorf("Base.Quantities(unknown) = non-nil, want nil")
	}
	other := p
	other.ConfigHash = [32]byte{1}
	if _, err := indicator.BaseOf(w, other); !errors.Is(err, indicator.ErrConfigMismatch) {
		t.Errorf("BaseOf(table of another world) error = %v, want ErrConfigMismatch", err)
	}
	short, _ := indicator.NewPrices(valleyHash, 10, 11, 1, 4, make([]economysim.Price, 4))
	if _, err := indicator.BaseOf(w, short); !errors.Is(err, indicator.ErrRange) {
		t.Errorf("BaseOf(table of another shape) error = %v, want ErrRange", err)
	}
}

// TestTally_RestoredWindow verifies that a window that straddles a restore, fed
// by the same tally, equals the window of the uninterrupted run: the
// WorldRestored of the restore changes nothing in an open window.
//
// Requirement: FR-IND-055
func TestTally_RestoredWindow(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	files := []string{"base.toml", "one.toml", "warm-one.toml"}
	whole := indicator.NewTally()
	a, _ := observedWorld(t, whole, files...)
	for i := 0; i < 5; i++ {
		if err := a.Step(); err != nil {
			t.Fatalf("Step() error = %v, want nil", err)
		}
	}
	if err := whole.Open(snapshotOf(t, a)); err != nil {
		t.Fatalf("Open() error = %v, want nil", err)
	}
	for i := 0; i < 15; i++ {
		if err := a.Step(); err != nil {
			t.Fatalf("Step() error = %v, want nil", err)
		}
	}
	want, err := whole.Close(snapshotOf(t, a))
	if err != nil {
		t.Fatalf("Close() of the uninterrupted run error = %v, want nil", err)
	}

	split := indicator.NewTally()
	b, cfg := observedWorld(t, split, files...)
	for i := 0; i < 5; i++ {
		if err := b.Step(); err != nil {
			t.Fatalf("Step() error = %v, want nil", err)
		}
	}
	if err := split.Open(snapshotOf(t, b)); err != nil {
		t.Fatalf("Open() error = %v, want nil", err)
	}
	for i := 0; i < 5; i++ {
		if err := b.Step(); err != nil {
			t.Fatalf("Step() error = %v, want nil", err)
		}
	}
	cp, err := b.Checkpoint()
	if err != nil {
		t.Fatalf("Checkpoint() error = %v, want nil", err)
	}
	b2, err := economysim.RestoreWorld(cfg, cp, economysim.WithObserver(split))
	if err != nil {
		t.Fatalf("RestoreWorld() error = %v, want nil", err)
	}
	for i := 0; i < 10; i++ {
		if err := b2.Step(); err != nil {
			t.Fatalf("Step() after the restore error = %v, want nil", err)
		}
	}
	got, err := split.Close(snapshotOf(t, b2))
	if err != nil {
		t.Fatalf("Close() across the restore error = %v, want nil", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("window across a restore differs from the uninterrupted run:\n got %+v\nwant %+v", got.Steps, want.Steps)
	}
}

// TestTally_OpensAtWorldRestored verifies that a tally with no window open
// opens one at the snapshot of WorldRestored, with its households, and that the
// window then closes like any other.
//
// Requirement: FR-IND-055
func TestTally_OpensAtWorldRestored(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	files := []string{"base.toml", "one.toml", "warm-one.toml"}
	a, cfg := observedWorld(t, indicator.NewTally(), files...)
	for i := 0; i < 4; i++ {
		if err := a.Step(); err != nil {
			t.Fatalf("Step() error = %v, want nil", err)
		}
	}
	cp, err := a.Checkpoint()
	if err != nil {
		t.Fatalf("Checkpoint() error = %v, want nil", err)
	}
	tl := indicator.NewTally()
	b, err := economysim.RestoreWorld(cfg, cp, economysim.WithObserver(tl))
	if err != nil {
		t.Fatalf("RestoreWorld() error = %v, want nil", err)
	}
	for i := 0; i < 3; i++ {
		if err := b.Step(); err != nil {
			t.Fatalf("Step() error = %v, want nil", err)
		}
	}
	s := snapshotOf(t, b)
	w, err := tl.Close(s)
	if err != nil {
		t.Fatalf("Close() error = %v, want nil", err)
	}
	if w.From != 4 || w.To != 7 || w.Steps != 3 {
		t.Errorf("window = [%d, %d) steps %d, want [4, 7) 3 steps: opened at the restored boundary", w.From, w.To, w.Steps)
	}
	var open int64
	for _, h := range w.OpenHouseholds {
		open += h
	}
	if open == 0 {
		t.Errorf("OpenHouseholds sum = 0, want the restored snapshot's households")
	}
	if got := indicator.NewTally().Open(economysim.Snapshot{Names: valleyNames(), Roles: []economysim.RoleRow{{Region: 9}}}); !errors.Is(got, indicator.ErrRange) {
		t.Errorf("Open(role row outside the names) error = %v, want ErrRange", got)
	}
}

// TestClose_StrayEvents verifies that an event whose IDs are outside the names
// table, a negative price or volume, or a tick before the window makes the
// window unclosable rather than writing outside a table: the tally never trusts
// an index it did not size.
//
// Requirement: FR-IND-006
func TestClose_StrayEvents(t *testing.T) {
	out := economysim.Amount{Commodity: 9, Quantity: 1}
	tests := []struct {
		name string
		e    economysim.Event
	}{
		{"market of an unknown region", economysim.MarketCleared{Tick: 10, Region: 5}},
		{"market with a negative price", economysim.MarketCleared{Tick: 10, Price: -1}},
		{"market before the window", economysim.MarketCleared{Tick: 9}},
		{"beliefs of an unknown commodity", economysim.BeliefsUpdated{Tick: 10, Commodity: 9}},
		{"recipe of an unknown region", economysim.RecipeExecuted{Tick: 10, Region: 5}},
		{"recipe of an unknown recipe", economysim.RecipeExecuted{Tick: 10, Recipe: 9}},
		{"recipe output of an unknown commodity", economysim.RecipeExecuted{Tick: 10, Outputs: []economysim.Amount{out}}},
		{"flow before the window", flow(9, alpha, economysim.ReasonConsumption, grain, 1)},
		{"people before the window", economysim.AgentBankrupted{Tick: 9, Region: alpha, Role: grower, Weight: 1}},
		{"role change to an unknown role", economysim.RoleChanged{Tick: 10, Region: alpha, From: grower, To: 9, Weight: 1}},
		{"relocation to an unknown region", economysim.AgentRelocated{Tick: 10, Role: grower, From: alpha, To: 9, Weight: 1}},
		{"upkeep shortfall of an unknown role", economysim.UpkeepShortfall{Tick: 10, Role: 9}},
		{"tick started before the window", economysim.TickStarted{Tick: 9}},
		{"tick ended before the window", economysim.TickEnded{Tick: 9}},
		{"relief of an unknown region", economysim.ReliefPaid{Tick: 10, Region: 5}},
		{"entry limit of an unknown region", economysim.EntryLimited{Tick: 10, Region: 5}},
		{"population of an unknown role", economysim.PopulationChanged{Tick: 10, Role: 9}},
		{"money flow of an unknown region", economysim.MoneyFlowLimited{Tick: 10, Region: 5}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tl := indicator.NewTally()
			openValley(t, tl, 10, [4]int64{})
			step(tl, 10, tt.e)
			if _, err := tl.Close(valleySnap(11, [4]int64{})); !errors.Is(err, indicator.ErrWindowOrder) {
				t.Errorf("Close() after %T error = %v, want ErrWindowOrder", tt.e, err)
			}
		})
	}
}

// TestWeights_String verifies the names of the weights, which a result reports.
func TestWeights_String(t *testing.T) {
	for k, want := range map[indicator.Weights]string{indicator.WeightsTraded: "traded", indicator.WeightsConsumed: "consumed", indicator.WeightsProduced: "produced", 9: "weights_9"} {
		if got := k.String(); got != want {
			t.Errorf("Weights(%d).String() = %q, want %q", int(k), got, want)
		}
	}
	if (indicator.BindingCounts{}).Any() || !(indicator.BindingCounts{Refused: 1}).Any() {
		t.Errorf("BindingCounts.Any() = wrong for the zero value or a counter")
	}
}

// TestFlows_CorrectionOverflow verifies that a correction that leaves int64 is
// ErrOverflow and not a wrapped total, and that no accounts are built from it.
//
// Requirement: FR-IND-009
func TestFlows_CorrectionOverflow(t *testing.T) {
	big := int64(1<<63 - 1)
	w, open, closing := workedWindow(t)
	w.Flows[0].Total = big
	w.ClosePending = []economysim.FlowTotal{{Region: w.Flows[0].Region, Reason: w.Flows[0].Reason, Kind: w.Flows[0].Kind, Commodity: w.Flows[0].Commodity, Total: 1}}
	if _, err := w.Corrected(); !errors.Is(err, indicator.ErrOverflow) {
		t.Errorf("Corrected() error = %v, want ErrOverflow", err)
	}
	if _, err := indicator.GDP(w, open, closing, valleyTable(t, w, alphaP, betaP)); !errors.Is(err, indicator.ErrOverflow) {
		t.Errorf("GDP() error = %v, want ErrOverflow", err)
	}
	if _, err := w.Quantities(indicator.WeightsProduced); !errors.Is(err, indicator.ErrOverflow) {
		t.Errorf("Quantities() error = %v, want ErrOverflow", err)
	}
}
