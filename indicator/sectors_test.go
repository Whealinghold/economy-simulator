package indicator_test

import (
	"errors"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/indicator"
)

// withMoney sets the money of the regions of the two snapshots, in the order
// alpha households, alpha pool, beta households, beta pool, and the households
// of alpha at the close.
func withMoney(open, closing *economysim.Snapshot, o, c [4]economysim.Money, alphaHouseholds int64) {
	for i, s := range []*economysim.Snapshot{open, closing} {
		m := [2][4]economysim.Money{o, c}[i]
		s.Regions[0].MoneyHouseholds, s.Regions[0].MoneyPool = m[0], m[1]
		s.Regions[1].MoneyHouseholds, s.Regions[1].MoneyPool = m[2], m[3]
	}
	closing.Regions[0].Households = alphaHouseholds
}

// TestSectors_MoneyByHolderAndNetWorth verifies the money by sector of 18.6
// (without the contract sector, which has no snapshot field until plan 2-15):
// 5 000 in alpha's households and 800 in its pool, 50 per household over 100
// households; the net worth of each sector at the table; and the closing of the
// identity of 7.5.3 exactly, at one price table.
//
// Requirement: FR-IND-018, FR-IND-036
func TestSectors_MoneyByHolderAndNetWorth(t *testing.T) {
	w, open, closing := workedWindow(t)
	// Money: alpha's households gain 100 (a faucet), beta's lose 20 (faucet 50, sink 70).
	withMoney(&open, &closing, [4]economysim.Money{4900, 800, 2000, 0}, [4]economysim.Money{5000, 800, 1980, 0}, 100)
	addFlow(&w, moneyFlow(10, alpha, economysim.ReasonStartMoney, economysim.KindFaucet, 100))
	addFlow(&w, moneyFlow(10, beta, economysim.ReasonStartMoney, economysim.KindFaucet, 50))
	addFlow(&w, moneyFlow(10, beta, economysim.ReasonEstateMoney, economysim.KindSink, 70))
	closing.Boundary = []economysim.BoundaryRow{{Reason: economysim.ReasonStartMoney, Kind: economysim.KindFaucet, Commodity: economysim.NoCommodity, Balance: -150}, {Reason: economysim.ReasonConsumption, Kind: economysim.KindSink, Commodity: grain, Balance: 9}}
	got, err := indicator.Sectors(w, open, closing, valleyTable(t, w, alphaP, alphaP))
	if err != nil {
		t.Fatalf("Sectors() error = %v, want nil", err)
	}
	a := got.Regions[0]
	if a.MoneyHouseholds != 5000 || a.MoneyPool != 800 || a.MoneyContract != 0 || a.MoneyTreasury != 0 || a.Money != 5800 {
		t.Errorf("Sectors().alpha money = %d %d %d %d (total %d), want 5000, 800, 0, 0 (5800)", a.MoneyHouseholds, a.MoneyPool, a.MoneyContract, a.MoneyTreasury, a.Money)
	}
	if a.PerHousehold != 50 || !a.PerHouseholdOK {
		t.Errorf("Sectors().alpha.PerHousehold = %d, %v, want 50, true", a.PerHousehold, a.PerHouseholdOK)
	}
	// Net worth at alpha's table. Households: 5000 + 186*10 + 50*25 + 38*6 + 5*5 + 8*25 (work in progress) = 8563.
	// Pool: 800 + 37 grain at 10 = 1170.
	if a.NW.Households != 8563 || a.NW.Pool != 1170 {
		t.Errorf("Sectors().alpha.NW = households %d pool %d, want 8563, 1170", a.NW.Households, a.NW.Pool)
	}
	// Change of the households' worth: money +100; agents' goods +6*10 +10*25 +13*6 +2*5; work in progress -4*25.
	if a.DNW.Households != 398 || a.DNW.Pool != 170 {
		t.Errorf("Sectors().alpha.DNW = households %d pool %d, want 398, 170", a.DNW.Households, a.DNW.Pool)
	}
	if got.Realm.Money != 5000+800+1980 || got.Realm.Households != 100 {
		t.Errorf("Sectors().Realm = money %d households %d, want 7780, 100", got.Realm.Money, got.Realm.Households)
	}
	if c := got.Closing; c.Residual != 0 || c.Money != 80 {
		t.Errorf("Sectors().Closing = %+v, want residual 0 and money flows of 80 (150 in, 70 out)", c)
	}
	if want := []indicator.ReasonTotal{{Reason: economysim.ReasonEstateMoney, Kind: economysim.KindSink, Total: 70}, {Reason: economysim.ReasonStartMoney, Kind: economysim.KindFaucet, Total: 150}}; !slices.Equal(got.Flows, want) {
		t.Errorf("Sectors().Flows = %v, want %v", got.Flows, want)
	}
	if len(got.Balances) != 1 || got.Balances[0].Balance != -150 {
		t.Errorf("Sectors().Balances = %v, want the one money row of -150", got.Balances)
	}
	if got.Regions[0].Account.Residual != 0 || got.Regions[1].Account.Residual != 0 || got.Realm.Account.Residual != 0 {
		t.Errorf("Sectors() accounts = %+v / %+v / %+v, want residual 0", got.Regions[0].Account, got.Regions[1].Account, got.Realm.Account)
	}
}

// TestSectors_Misuse verifies that snapshots with a different number of regions
// than the window are refused.
//
// Requirement: FR-IND-006
func TestSectors_Misuse(t *testing.T) {
	w, open, closing := workedWindow(t)
	closing.Regions = closing.Regions[:1]
	if _, err := indicator.Sectors(w, open, closing, valleyTable(t, w, alphaP, betaP)); !errors.Is(err, indicator.ErrRange) {
		t.Errorf("Sectors(closing with one region) error = %v, want ErrRange", err)
	}
}

// TestRegionalMoneyIdentity_Terms verifies that the change of a region's money
// closes exactly on its faucets and sinks, the money relocated with accounts,
// the contract payments settled in and out and the money that crossed it under a
// fiscal flow, and that the realm closes with the transfers cancelling. The
// terms are written on a literal: the tally fills the fiscal terms only when the
// fiscal module adds its case.
//
// Requirement: FR-IND-019
func TestRegionalMoneyIdentity_Terms(t *testing.T) {
	w, open, closing := workedWindow(t)
	// alpha: +100 faucet -40 relocated out +30 settled in +7 fiscal in = +97.
	// beta: +50 faucet -70 sink +40 relocated in -30 settled out -7 fiscal out = -17.
	withMoney(&open, &closing, [4]economysim.Money{4900, 800, 2000, 0}, [4]economysim.Money{4997, 800, 1983, 0}, 100)
	addFlow(&w, moneyFlow(10, alpha, economysim.ReasonStartMoney, economysim.KindFaucet, 100))
	addFlow(&w, moneyFlow(10, beta, economysim.ReasonStartMoney, economysim.KindFaucet, 50))
	addFlow(&w, moneyFlow(10, beta, economysim.ReasonEstateMoney, economysim.KindSink, 70))
	w.Relocations = []indicator.RelocTally{{From: alpha, To: beta, Money: 40, Weight: 2}}
	w.Settlements = []indicator.SettlementTally{{Region: alpha, To: beta, Paid: 30}}
	w.Regional[alpha].FiscalIn, w.Regional[beta].FiscalOut = 7, 7
	got, err := indicator.Sectors(w, open, closing, valleyTable(t, w, alphaP, alphaP))
	if err != nil {
		t.Fatalf("Sectors() error = %v, want nil", err)
	}
	want := []indicator.RegionMoney{
		{Change: 97, Faucets: 100, RelocOut: 40, SettledIn: 30, FiscalIn: 7},
		{Change: -17, Faucets: 50, Sinks: 70, RelocIn: 40, SettledOut: 30, FiscalOut: 7},
		{Change: 80, Faucets: 150, Sinks: 70, RelocIn: 40, RelocOut: 40, SettledIn: 30, SettledOut: 30, FiscalIn: 7, FiscalOut: 7},
	}
	for i, row := range []indicator.SectorRow{got.Regions[0], got.Regions[1], got.Realm} {
		if row.Account != want[i] {
			t.Errorf("Sectors().account[%d] = %+v, want %+v", i, row.Account, want[i])
		}
	}
}

// TestTradeBalance_GrossAndNet verifies 18.8: at gate beta the outside's budget
// of 200 less 32 unspent is 168 exported and 120 received for imports, a net of
// +48; 15 units of wood imported (20 offered, 5 unsold) and 6 of loaf exported,
// each at the table; and the openness (168 + 120) * 10 000 / 2 090 = 1 378.
//
// Requirement: FR-IND-042
func TestTradeBalance_GrossAndNet(t *testing.T) {
	w, open, closing := workedWindow(t)
	addFlow(&w, moneyFlow(10, beta, "outside_budget", economysim.KindFaucet, 200))
	addFlow(&w, moneyFlow(10, beta, "outside_unspent", economysim.KindSink, 32))
	addFlow(&w, moneyFlow(10, beta, "outside_receipts", economysim.KindSink, 120))
	p := valleyTable(t, w, alphaP, betaP)
	gdp, err := indicator.GDP(w, open, closing, p)
	if err != nil {
		t.Fatalf("GDP() error = %v, want nil", err)
	}
	got, err := indicator.TradeBalance(w, p, gdp)
	if err != nil {
		t.Fatalf("TradeBalance() error = %v, want nil", err)
	}
	wantLines := []indicator.TradeLine{
		{Gate: beta, Commodity: loaf, Exp: 6, ExpValue: 168},
		{Gate: beta, Commodity: wood, Imp: 15, ImpValue: 120},
	}
	if !slices.Equal(got.Lines, wantLines) {
		t.Errorf("TradeBalance().Lines = %+v, want %+v", got.Lines, wantLines)
	}
	if len(got.Gates) != 1 || got.Gates[0].ImportValue != 120 || got.Gates[0].ExportValue != 168 || got.Gates[0].Net != 48 {
		t.Errorf("TradeBalance().Gates = %+v, want one gate, import 120, export 168, net +48", got.Gates)
	}
	if got.Total.Net != 48 || got.Openness != 1378 || !got.OpennessOK {
		t.Errorf("TradeBalance() = net %d openness %d (%v), want 48, 1378, true", got.Total.Net, got.Openness, got.OpennessOK)
	}
	// The money value (168) equals the units at the table here (6 * 28); price improvement
	// of the midpoint rule would make them differ, and is a transfer, not output.
	if got.Lines[0].ExpValue != got.Gates[0].ExportValue {
		t.Errorf("exported units at the table = %d, money = %d, want equal in this fixture", got.Lines[0].ExpValue, got.Gates[0].ExportValue)
	}
	zero := gdp
	zero.Realm.ValueAdded = 0
	if undefined, err := indicator.TradeBalance(w, p, zero); err != nil || undefined.OpennessOK || undefined.Openness != 0 {
		t.Errorf("TradeBalance(GDP of 0) = openness %d ok %v err %v, want 0, false, nil", undefined.Openness, undefined.OpennessOK, err)
	}
	stale := gdp
	stale.From = 0
	if _, err := indicator.TradeBalance(w, p, stale); !errors.Is(err, indicator.ErrWindowOrder) {
		t.Errorf("TradeBalance(GDP of another window) error = %v, want ErrWindowOrder", err)
	}
}

// TestBinding_WarningCarriesTheCounters verifies that a regional comparison
// carries the warning Binding and the counters of the region when a cap acted in
// the window, and carries neither when none did.
//
// Requirement: FR-IND-051
func TestBinding_WarningCarriesTheCounters(t *testing.T) {
	run := func(events ...economysim.Event) (indicator.Window, economysim.Snapshot, economysim.Snapshot) {
		open, closing := valleySnap(10, [4]int64{}), valleySnap(11, [4]int64{})
		tl := indicator.NewTally()
		if err := tl.Open(open); err != nil {
			t.Fatalf("Open() error = %v, want nil", err)
		}
		step(tl, 10, events...)
		w, err := tl.Close(closing)
		if err != nil {
			t.Fatalf("Close() error = %v, want nil", err)
		}
		return w, open, closing
	}
	has := func(ws []indicator.Warning) bool { return slices.Contains(ws, indicator.WarnBinding) }

	w, open, closing := run(economysim.EntryLimited{Tick: 10, Region: beta, RegionCap: 3},
		economysim.PopulationChanged{Tick: 10, Region: beta, Role: grower, BirthsCapped: 2},
		economysim.MoneyFlowLimited{Tick: 10, Region: beta, Short: 1, Limit: 4})
	p := valleyTable(t, w, [4]economysim.Price{1, 1, 1, 1}, [4]economysim.Price{1, 1, 1, 1})
	gdp, err := indicator.GDP(w, open, closing, p)
	if err != nil {
		t.Fatalf("GDP() error = %v, want nil", err)
	}
	sec, err := indicator.Sectors(w, open, closing, p)
	if err != nil {
		t.Fatalf("Sectors() error = %v, want nil", err)
	}
	tb, err := indicator.TradeBalance(w, p, gdp)
	if err != nil {
		t.Fatalf("TradeBalance() error = %v, want nil", err)
	}
	if !has(gdp.Warnings) || !has(sec.Warnings) || !has(tb.Warnings) {
		t.Errorf("warnings = GDP %v Sectors %v TradeBalance %v, want Binding in each", gdp.Warnings, sec.Warnings, tb.Warnings)
	}
	wantBeta := indicator.BindingCounts{RegionCap: 3, BirthsCapped: 2, FlowShort: 1, FlowLimit: 4}
	if got := gdp.Regions[1].Binding; got != wantBeta {
		t.Errorf("GDP().beta.Binding = %+v, want %+v", got, wantBeta)
	}
	if got := sec.Regions[0].Binding; got.Any() {
		t.Errorf("Sectors().alpha.Binding = %+v, want no counter", got)
	}
	if !slices.IsSorted(gdp.Warnings) {
		t.Errorf("GDP().Warnings = %v, want ascending", gdp.Warnings)
	}

	w, open, closing = run()
	gdp, err = indicator.GDP(w, open, closing, valleyTable(t, w, [4]economysim.Price{1, 1, 1, 1}, [4]economysim.Price{1, 1, 1, 1}))
	if err != nil {
		t.Fatalf("GDP() error = %v, want nil", err)
	}
	if has(gdp.Warnings) {
		t.Errorf("GDP().Warnings = %v with no cap acting, want no Binding", gdp.Warnings)
	}
	for k, want := range map[indicator.Warning]string{indicator.WarnStale: "stale", indicator.WarnBinding: "binding", 99: "warning_99"} {
		if k.String() != want {
			t.Errorf("Warning(%d).String() = %q, want %q", int(k), k.String(), want)
		}
	}
}

// TestPendingDependencies records the checks that wait for modules not yet in
// the tree. Each names the plan that unblocks it.
func TestPendingDependencies(t *testing.T) {
	t.Run("transit stock from Snapshot.Shipments", func(t *testing.T) {
		t.Skip("plan 2-12: Snapshot.Shipments is not in the tree; GDP and Sectors read the transit stock from it and the worked fixture then holds the 6 units in flight there (task stage-2-18 dependency note)")
	})
	t.Run("contract escrow in Sectors", func(t *testing.T) {
		t.Skip("plan 2-15: RegionRow.MoneyContract is not in the tree; Sectors then reports M_c = 200 and M = 6 000 for alpha (18.6)")
	})
	t.Run("tally fills the shipment and contract rows", func(t *testing.T) {
		t.Skip("plans 2-12 and 2-15: ShipmentDispatched, ShipmentArrived, ContractFormed and ContractSettled are not in the tree; Observe then fills Dispatches, Arrivals and Settlements")
	})
}
