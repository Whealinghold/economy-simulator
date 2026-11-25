package economysim

import (
	"fmt"
	"log/slog"
	"reflect"
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/ledger"
	"github.com/Whealinghold/economy-sim/internal/logkeys"
)

// The rebuild of an observer that keeps no configuration (spec-fiscal.md 16.5, 4.3.2,
// plan 3-fis Phase 9): the money of every region and of the treasury sector, and the
// goods of the treasuries, rebuilt from the events alone equal the snapshot.

// observerFiscal is a module with every kind that moves money or goods between a
// treasury and the rest, over the fiscal mill (the crown seated at home, the shire at
// yonder): a head tax of each treasury on its millers, a remittance of the shire's money
// up to the crown (money from yonder to home), a payout of money to the yonder millers
// (home to yonder), a ration of grain in kind to the home millers, and a band of the
// crown and of the shire on grain, each in its seat's market.
func observerFiscal() *Fiscal {
	yonder := FlowTarget{Kind: TargetHouseholds, Region: "yonder", Role: "miller"}
	return &Fiscal{
		Taxes: []Tax{
			{ID: "poll", Authority: "crown", Target: homeMillers, Amount: 4},
			{ID: "poll_s", Authority: "shire", Target: yonder, Amount: 6},
			{ID: "remit", Authority: "crown", Target: FlowTarget{Kind: TargetTreasury}, RateBP: 500},
		},
		Payouts: []Payout{
			{ID: "alms", Authority: "crown", Target: yonder, Amount: 3},
			{ID: "ration", Authority: "crown", Target: homeMillers, Commodity: "grain", Amount: 1},
		},
		Purchases: []Purchase{buy("buy", 2, 9_900, 0), {ID: "buy_s", Authority: "shire", Commodity: "grain", Quantity: 2, LimitBP: 9_900}},
		Releases:  []Release{sell("sell", 2, 10_100, 0), {ID: "sell_s", Authority: "shire", Commodity: "grain", Quantity: 2, LimitBP: 10_100}},
	}
}

// stockTreasury puts qty units of grain into the treasury at index j of the sorted
// governed jurisdictions (crown 0, shire 1), by a trade from the agent from.
func stockTreasury(t *testing.T, w *World, j int, from AgentID, qty int64) {
	t.Helper()
	g := payoutGrain(w)
	give(t, w, from, g, qty)
	if err := w.ledger.Move(w.agents[from].account, w.polity.tr[j].account, int(g), qty, ledger.Trade); err != nil {
		t.Fatalf("stock treasury %d with %d grain: %v", j, qty, err)
	}
}

// observerRun builds the world of observerFiscal with 300 in every household and 12 grain
// in each treasury, records 12 steps and returns the world, the snapshot before the first
// step and the events.
func observerRun(t *testing.T) (*World, Snapshot, *evLog) {
	t.Helper()
	w, rec := orderMill(t, observerFiscal(), 5_000)
	c := castOf(t, w)
	money := map[AgentID]int64{}
	w.eachLive(func(a *agent) { money[a.id] = 300 })
	setBalances(t, w, money)
	stockTreasury(t, w, 0, c.homeWeaver, 12)
	stockTreasury(t, w, 1, c.yonderWeaver, 12)
	start, err := w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot() error = %v, want nil", err)
	}
	rec.events = nil
	stepN(t, w, 12)
	return w, start, rec
}

// fiscalFold is what an observer that keeps no configuration rebuilds from the events
// of a run: by region the faucets less the sinks of money, FiscalIn_r and FiscalOut_r,
// the money of the treasury sector, and the units a treasury holds by (region,
// commodity), each as a change from the snapshot the run started at.
type fiscalFold struct {
	boundary, in, out, treasury []int64
	goods                       map[[2]int]int64
	treasuryTrades              int
	seat                        map[AgentID]RegionID
	seatOf                      map[JurisdictionID]RegionID
}

// newFiscalFold reads the treasuries' agents and seats from the Jurisdictions rows, as
// an observer reads them from WorldCreated (spec-observability.md 4.3.3 item 2).
func newFiscalFold(s Snapshot) *fiscalFold {
	n := len(s.Regions)
	f := &fiscalFold{
		boundary: make([]int64, n), in: make([]int64, n), out: make([]int64, n), treasury: make([]int64, n),
		goods: map[[2]int]int64{}, seat: map[AgentID]RegionID{}, seatOf: map[JurisdictionID]RegionID{},
	}
	for _, j := range s.Jurisdictions {
		f.seat[j.Agent], f.seatOf[j.Jurisdiction] = j.Seat, j.Seat
	}
	return f
}

// apply adds one event to the fold.
func (f *fiscalFold) apply(e Event) {
	switch e := e.(type) {
	case BoundaryFlow:
		if e.Tick >= 0 && e.Commodity == NoCommodity {
			f.boundary[e.Region] += map[Kind]int64{KindFaucet: 1, KindSink: -1}[e.Kind] * e.Total
		}
	case TradeExecuted:
		f.trade(e)
	case FiscalFlowApplied:
		f.flow(e)
	}
}

// trade classes the money and the goods of a trade with a treasury to the treasury
// sector and the holdings of its seat; a trade of two households is no concern here.
func (f *fiscalFold) trade(e TradeExecuted) {
	value := int64(e.Price) * int64(e.Quantity)
	if r, ok := f.seat[e.Seller]; ok {
		f.treasury[r] += value
		f.goods[[2]int{int(r), int(e.Commodity)}] -= int64(e.Quantity)
		f.treasuryTrades++
	}
	if r, ok := f.seat[e.Buyer]; ok {
		f.treasury[r] -= value
		f.goods[[2]int{int(r), int(e.Commodity)}] += int64(e.Quantity)
		f.treasuryTrades++
	}
}

// flow classes a FiscalFlowApplied: the money of a tax, a duty or a toll goes from the
// sector of Class in Region to the treasury sector of the seat of Authority, a payout
// the reverse; a class of treasuries is the treasury sector of Region. The units of a
// payout in kind leave the seat's treasuries.
func (f *fiscalFold) flow(e FiscalFlowApplied) {
	seat, a := f.seatOf[e.Authority], int64(e.Applied)
	if e.Commodity != NoCommodity {
		f.goods[[2]int{int(seat), int(e.Commodity)}] -= a
		return
	}
	payer, payee := e.Region, seat
	if e.Kind == KindPayout {
		payer, payee = seat, e.Region
	}
	if e.Kind == KindPayout {
		f.treasury[seat] -= a
		if e.Class == FiscalTreasury {
			f.treasury[e.Region] += a
		}
	} else {
		f.treasury[seat] += a
		if e.Class == FiscalTreasury {
			f.treasury[e.Region] -= a
		}
	}
	if payer != payee {
		f.out[payer] += a
		f.in[payee] += a
	}
}

// regionMoney is the money of every sector of a region the identity of spec-fiscal.md
// 4.3.2 sums: households, pool, contract and treasury.
func regionMoney(r RegionRow) int64 {
	return int64(r.MoneyHouseholds) + int64(r.MoneyPool) + int64(r.MoneyContract) + int64(r.MoneyTreasury)
}

// TestFiscal_RegionalIdentity verifies the regional money identity with the two terms
// the module adds (spec-fiscal.md 4.3.2): over a run of the world in which the shire
// remits to the crown (yonder to home) and the crown pays a stipend to the yonder
// millers (home to yonder), the change of the money of every region, treasury included,
// equals the faucets less the sinks plus FiscalIn_r less FiscalOut_r read from
// FiscalFlowApplied and the seats of the snapshot. The mill has no relocation and no
// contract, so those terms are 0.
//
// Requirement: FR-FIS-046
//
// Guard: a flow attributed to the wrong region (a payout to the recipient's, a tax to
// the seat's) leaves the identity of one region short and the other over by the same sum,
// which the sum over regions would hide.
func TestFiscal_RegionalIdentity(t *testing.T) {
	t.Parallel()
	w, start, rec := observerRun(t)
	f := newFiscalFold(start)
	for _, e := range rec.events {
		f.apply(e)
	}
	end, err := w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot() error = %v, want nil", err)
	}
	var moved int64
	for r := range end.Regions {
		got := regionMoney(end.Regions[r]) - regionMoney(start.Regions[r])
		if want := f.boundary[r] + f.in[r] - f.out[r]; got != want {
			t.Errorf("region %d: money changed by %d, want %d = boundary %d + in %d - out %d", r, got, want, f.boundary[r], f.in[r], f.out[r])
		}
		moved += f.in[r] + f.out[r]
	}
	if moved == 0 || f.in[0] == 0 || f.in[1] == 0 {
		t.Errorf("FiscalIn = %v, FiscalOut = %v, want money crossing both ways between the regions", f.in, f.out)
	}
}

// TestFiscal_MoneyRebuild verifies that an observer that keeps no configuration
// rebuilds the money of the treasury sector of every region and the goods the
// treasuries hold from the events alone (spec-fiscal.md 16.5, items 2 and 12): the money
// of a trade with a treasury is classed to the treasury sector of its seat by the
// Jurisdictions rows, that of a FiscalFlowApplied by its kind and class, and the units
// of a trade and of a payout in kind move the treasury's holding. The result equals
// RegionRow.MoneyTreasury, the rows of Jurisdictions and MarketRow.HeldByTreasury.
//
// Requirement: FR-FIS-047, FR-FIS-048
func TestFiscal_MoneyRebuild(t *testing.T) {
	t.Parallel()
	w, start, rec := observerRun(t)
	f := newFiscalFold(start)
	for _, e := range rec.events {
		f.apply(e)
	}
	end, err := w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot() error = %v, want nil", err)
	}
	if f.treasuryTrades == 0 || len(of[FiscalFlowApplied](rec)) == 0 {
		t.Fatalf("the run had %d trades of a treasury and %d fiscal events, want both above 0", f.treasuryTrades, len(of[FiscalFlowApplied](rec)))
	}
	for _, fault := range append(f.moneyFaults(start, end), f.goodsFaults(start, end)...) {
		t.Error(fault)
	}
}

// moneyFaults lists where the treasury sector the events rebuild differs from the
// snapshot: the money of each region and the row of each jurisdiction.
func (f *fiscalFold) moneyFaults(start, end Snapshot) (out []string) {
	for r := range end.Regions {
		if got, want := int64(end.Regions[r].MoneyTreasury), int64(start.Regions[r].MoneyTreasury)+f.treasury[r]; got != want {
			out = append(out, fmt.Sprintf("region %d: MoneyTreasury = %d, the events rebuild %d", r, got, want))
		}
	}
	for i, j := range end.Jurisdictions { // the rows are in ascending jurisdiction order in both
		if got, want := int64(j.Money), int64(start.Jurisdictions[i].Money)+f.treasury[j.Seat]; got != want {
			out = append(out, fmt.Sprintf("jurisdiction %d: Money = %d, the events rebuild %d (one treasury in each seat)", j.Jurisdiction, got, want))
		}
	}
	return out
}

// goodsFaults lists where the units of the treasuries the events rebuild differ from
// HeldByTreasury of the market rows (the rows are in the same order in both snapshots).
func (f *fiscalFold) goodsFaults(start, end Snapshot) (out []string) {
	for i, m := range end.Markets {
		got := int64(m.HeldByTreasury)
		want := int64(start.Markets[i].HeldByTreasury) + f.goods[[2]int{int(m.State.Region), int(m.State.Commodity)}]
		if got != want {
			out = append(out, fmt.Sprintf("market %d/%d: HeldByTreasury = %d, the events rebuild %d", m.State.Region, m.State.Commodity, got, want))
		}
	}
	return out
}

// TestFiscal_Snapshot verifies the one field the module adds to the snapshot
// (spec-fiscal.md 16.3, FR-FIS-048): HeldByTreasury of a market row is the units the
// treasuries seated in the region hold, a function of the saved ledger, and their lots
// are in Lots; nothing else of the module is a row of the snapshot; a restored world's
// snapshot equals the original's.
//
// Arithmetic: the crown (seat home) holds 7 grain and the shire (seat yonder) 3, so the
// home grain row has 7 and the yonder one 3; the lots of a treasury count in Lots beside
// the households' and the estate's.
//
// Requirement: FR-FIS-048
func TestFiscal_Snapshot(t *testing.T) {
	t.Parallel()
	cfg := fiscalMillConfig(&Fiscal{Purchases: []Purchase{buy("b", 1, 8_000, 0)}})
	w := newMill(t, cfg, WithSeed(exitSeed), WithAudit())
	c := castOf(t, w)
	before := mustSnapshot(t, w)
	stockTreasury(t, w, 0, c.homeWeaver, 7)
	stockTreasury(t, w, 1, c.yonderWeaver, 3)
	got := mustSnapshot(t, w)
	g := payoutGrain(w)
	home, yonder, b := got.Markets[w.markets.index(0, g)], got.Markets[w.markets.index(1, g)], before.Markets[w.markets.index(0, g)]
	if home.HeldByTreasury != 7 || yonder.HeldByTreasury != 3 || b.HeldByTreasury != 0 {
		t.Errorf("HeldByTreasury of home and yonder grain = %d and %d (before %d), want 7 and 3 (before 0)", home.HeldByTreasury, yonder.HeldByTreasury, b.HeldByTreasury)
	}
	if home.Lots != b.Lots+1 {
		t.Errorf("Lots of home grain = %d, want %d: the treasury's lot counts beside the households' and the estate's", home.Lots, b.Lots+1)
	}
	ck, err := w.Checkpoint()
	if err != nil {
		t.Fatalf("Checkpoint() error = %v, want nil", err)
	}
	r, err := RestoreWorld(cfg, ck, WithAudit())
	if err != nil {
		t.Fatalf("RestoreWorld() error = %v, want nil", err)
	}
	if !reflect.DeepEqual(mustSnapshot(t, r), got) {
		t.Error("snapshot of the restored world differs from the original's, want equal")
	}
}

// mustSnapshot takes the snapshot of a world, failing the test on an error.
func mustSnapshot(t *testing.T, w *World) Snapshot {
	t.Helper()
	s, err := w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot() error = %v, want nil", err)
	}
	return s
}

// TestFiscal_Logs verifies the logs of the module (spec-fiscal.md 16.2, FR-FIS-049): the
// construction line names fiscal among the modules and counts the flows; a step logs
// nothing of the module at Info, Warn or Error; the Debug line "treasury ordered" has a
// constant message and the keys authority and region.
//
// Requirement: FR-FIS-049
func TestFiscal_Logs(t *testing.T) {
	t.Parallel()
	h := newCapture()
	cfg := fiscalMillConfig(&Fiscal{Purchases: []Purchase{buy("b", 1, 8_000, 0)}, Taxes: []Tax{{ID: "poll", Authority: "crown", Target: homeMillers, Amount: 1}}})
	w := newMill(t, cfg, WithSeed(exitSeed), WithAudit(), WithLogger(slog.New(h)))
	created := h.take("world created")
	if len(created) != 1 || created[0].attrs[logkeys.FiscalFlows] != 2 || !slices.Contains(modulesOn(&w.cfg), "fiscal") {
		t.Errorf("world created lines = %+v, modules %v, want one with fiscal_flows = 2 (a tax and a purchase) and fiscal among the modules", created, modulesOn(&w.cfg))
	}
	stepN(t, w, 3)
	debug, loud := orderedLines(*h.lines)
	if debug != 3 || loud != 0 {
		t.Errorf("in 3 steps: %d Debug lines \"treasury ordered\" for the crown and %d lines of the module above Debug, want 3 and 0", debug, loud)
	}
}

// orderedLines counts the Debug lines "treasury ordered" of the crown at home and the lines
// of the module above Debug.
func orderedLines(lines []logRec) (debug, loud int) {
	for _, l := range lines {
		switch {
		case l.msg != "treasury ordered" && l.msg != "fiscal flow applied":
		case l.level != slog.LevelDebug:
			loud++
		case l.msg == "treasury ordered" && l.strs["authority"] == "crown" && l.attrs["region"] == 0:
			debug++
		}
	}
	return debug, loud
}
