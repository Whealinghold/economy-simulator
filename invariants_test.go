package economysim_test

import (
	"fmt"
	"reflect"
	"slices"
	"sort"
	"strings"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// This file is phase 5 of doc/plans/plan-phase-1-12.md: the invariants of
// testing.md 6.3 over the packs and seeds of REF 7.5, checked with exact
// equality after every tick (D179, D180). The checker is a plain struct fed
// events and snapshots, so its unit tests need no world.

// invTicks is the length of an invariants run (REF 7.5).
const invTicks int64 = 400

// invFlowKey is a boundary account: a reason and a commodity (NoCommodity for money).
type invFlowKey struct {
	reason    economysim.Reason
	commodity economysim.CommodityID
}

// signedFlow is the signed total of a flow in the sign of the account balance:
// a faucet is negative, a sink positive (OBS 5.3.3).
func signedFlow(kind economysim.Kind, total int64) int64 {
	if kind == economysim.KindFaucet {
		return -total
	}
	return total
}

// invChecker accumulates what the invariants need between ticks.
type invChecker struct {
	regions int
	live    map[economysim.AgentID]bool
	// flows is the construction term plus the BoundaryFlow events of every
	// tick since, signed; bucket is the Tick -1 events alone.
	flows, base, bucket map[invFlowKey]int64
	bucketChecked       bool
	// unchecked counts trades that D180 can check against one side only.
	bothSided, oneSided, noSide int
}

// newInvChecker starts from the snapshot taken before the first step, whose
// Construction is the term of FR-OBS-042 (D179).
func newInvChecker(pre economysim.Snapshot) *invChecker {
	c := &invChecker{regions: len(pre.Regions), live: map[economysim.AgentID]bool{},
		flows: map[invFlowKey]int64{}, base: map[invFlowKey]int64{}, bucket: map[invFlowKey]int64{}}
	for _, f := range pre.Construction {
		k := invFlowKey{f.Reason, f.Commodity}
		c.flows[k] += signedFlow(f.Kind, f.Total)
		c.base[k] += signedFlow(f.Kind, f.Total)
	}
	return c
}

// isEstate reports whether an agent ID is an estate pseudo-agent (0 .. regions-1).
func (c *invChecker) isEstate(a economysim.AgentID) bool { return a >= 0 && int(a) < c.regions }

// events folds the events of one batch (the NewWorld events or one step) into
// the checker and returns the failures of the event-only invariants 4 and 5
// and the closed-world checks.
func (c *invChecker) events(evs []economysim.Event) []string {
	type orderKey struct {
		tick   int64
		region economysim.RegionID
		com    economysim.CommodityID
		agent  economysim.AgentID
		side   economysim.Side
	}
	var bad []string
	limits := map[orderKey]economysim.Price{}
	for _, e := range evs {
		switch e := e.(type) {
		case economysim.AgentSpawned:
			c.live[e.Agent] = true
		case economysim.AgentBankrupted:
			delete(c.live, e.Agent)
		case economysim.BoundaryFlow:
			k := invFlowKey{e.Reason, e.Commodity}
			if e.Tick < 0 {
				c.bucket[k] += signedFlow(e.Kind, e.Total)
			} else {
				c.flows[k] += signedFlow(e.Kind, e.Total)
			}
		case economysim.OrderPlaced:
			limits[orderKey{e.Tick, e.Region, e.Commodity, e.Agent, e.Side}] = e.Limit
		case economysim.OrderRejected:
			switch e.Reason {
			case economysim.ReasonOverflow, economysim.ReasonHoldingLimit, economysim.ReasonReceiptLimit:
				bad = append(bad, fmt.Sprintf("tick %d: order of agent %d rejected by an arithmetic bound (%s)", e.Tick, e.Agent, e.Reason))
			}
		case economysim.TradeExecuted:
			for _, who := range [2]economysim.AgentID{e.Buyer, e.Seller} {
				if !c.live[who] && !c.isEstate(who) {
					bad = append(bad, fmt.Sprintf("tick %d: trade names agent %d, who does not exist", e.Tick, who))
				}
			}
			bid, hasBid := limits[orderKey{e.Tick, e.Region, e.Commodity, e.Buyer, economysim.Bid}]
			ask, hasAsk := limits[orderKey{e.Tick, e.Region, e.Commodity, e.Seller, economysim.Ask}]
			if c.isEstate(e.Seller) {
				hasAsk = false // one ask per held lot: the one-order premise fails (D180)
			}
			if msg := priceBetween(e.Price, ask, hasAsk, bid, hasBid); msg != "" {
				bad = append(bad, fmt.Sprintf("tick %d region %d commodity %d: trade of %d for agent %d <- %d: %s", e.Tick, e.Region, e.Commodity, e.Price, e.Buyer, e.Seller, msg))
			}
			switch {
			case hasBid && hasAsk:
				c.bothSided++
			case hasBid || hasAsk:
				c.oneSided++
			default:
				c.noSide++
			}
		}
	}
	return bad
}

// priceBetween is the clearing-price invariant: ask <= price <= bid, each side
// checked when its limit is known. A trade with neither side known is checked
// for a positive price only.
func priceBetween(price, ask economysim.Price, hasAsk bool, bid economysim.Price, hasBid bool) string {
	switch {
	case price <= 0:
		return "price is not positive"
	case hasBid && price > bid:
		return fmt.Sprintf("price above the bid limit %d", bid)
	case hasAsk && price < ask:
		return fmt.Sprintf("price below the ask limit %d", ask)
	}
	return ""
}

// balances checks the conservation identities against a snapshot taken at a
// tick boundary: goods (i), money (ii) and FR-OBS-042 for every boundary
// account. It also checks, once, that the Tick -1 events are the construction
// term, so a recorder never needs both.
func (c *invChecker) balances(s economysim.Snapshot) []string {
	var bad []string
	if !c.bucketChecked && s.Tick > 0 {
		c.bucketChecked = true
		if !reflect.DeepEqual(c.base, c.bucket) {
			bad = append(bad, fmt.Sprintf("the Tick -1 BoundaryFlow events %v differ from the construction bucket %v", c.bucket, c.base))
		}
	}
	held := map[economysim.CommodityID]int64{}
	for _, m := range s.Markets {
		held[m.State.Commodity] += int64(m.HeldByAgents) + int64(m.HeldByEstate)
	}
	sum := map[economysim.CommodityID]int64{economysim.NoCommodity: int64(s.Money)}
	pending := map[invFlowKey]int64{}
	for _, f := range s.Pending {
		pending[invFlowKey{f.Reason, f.Commodity}] += signedFlow(f.Kind, f.Total)
	}
	declared := map[invFlowKey]bool{}
	for _, b := range s.Boundary {
		k := invFlowKey{b.Reason, b.Commodity}
		declared[k] = true
		if want := c.flows[k] + pending[k]; b.Balance != want {
			bad = append(bad, fmt.Sprintf("FR-OBS-042: boundary %s/%d balance %d, construction + flows + pending = %d", b.Reason, b.Commodity, b.Balance, want))
		}
		if _, ok := sum[b.Commodity]; !ok {
			sum[b.Commodity] = held[b.Commodity]
		}
		sum[b.Commodity] += b.Balance
	}
	for k := range c.flows {
		if !declared[k] {
			bad = append(bad, fmt.Sprintf("flow of undeclared boundary %s/%d", k.reason, k.commodity))
		}
	}
	for _, k := range sortedCommodities(sum) {
		if sum[k] != 0 {
			what := fmt.Sprintf("goods of commodity %d", k)
			if k == economysim.NoCommodity {
				what = "money"
			}
			bad = append(bad, fmt.Sprintf("%s: accounts + boundary balances = %d, want 0", what, sum[k]))
		}
	}
	return bad
}

// sortedCommodities lists the keys of m in ascending order, so a failure
// message is stable.
func sortedCommodities(m map[economysim.CommodityID]int64) []economysim.CommodityID {
	keys := make([]economysim.CommodityID, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	return keys
}

// noMoneyFlowAfterConstruction reports the money boundary flows in events of a
// step: the packs have no faucet or sink of money after construction. The tick
// it names is the Tick of the event, the step that ran, which is one less than
// the Tick of the snapshot taken after that step; runInvariants labels its own
// failures with the same step tick.
func noMoneyFlowAfterConstruction(evs []economysim.Event) []string {
	var bad []string
	for _, e := range evs {
		if f, ok := e.(economysim.BoundaryFlow); ok && f.Tick >= 0 && f.Commodity == economysim.NoCommodity {
			bad = append(bad, fmt.Sprintf("money flow %s %s %d in the step of tick %d, after construction", f.Reason, f.Kind, f.Total, f.Tick))
		}
	}
	return bad
}

// withoutConfiguredFlows drops the BoundaryFlow events of the money flows of the
// configuration: a flow of the money supply module is a named faucet or sink, the
// one way money may move after construction (FR-MON-002), and its accounting is
// the module's own tests.
func withoutConfiguredFlows(cfg economysim.Config, evs []economysim.Event) []economysim.Event {
	if cfg.Modules.MoneySupply == nil {
		return evs
	}
	var out []economysim.Event
	for _, e := range evs {
		if f, ok := e.(economysim.BoundaryFlow); ok && slices.ContainsFunc(cfg.Modules.MoneySupply.Flows, func(m economysim.MoneyFlow) bool { return economysim.Reason(m.ID) == f.Reason }) {
			continue
		}
		out = append(out, e)
	}
	return out
}

// invariantCase is one row of the invariants table (REF 7.5).
type invariantCase struct {
	name   string
	layers []string
	seeds  int // seeds 1 .. seeds
	// ticks is the length of a run; 0 is invTicks. script, when set, is a file of
	// testdata/chains/scripts applied between steps.
	ticks  int64
	script string
}

// invariantTable is the Stage 1 table: N-295 lists the rows that wait. The
// three row is the J1 probe's: it builds and runs, its regions do not trade
// (REF 16.2); the two +trade rows are the packs with the edge part of their trade
// overlay (plan 2-3 Phase 9; the edges do not change a run until TRN reads them).
var invariantTable = []invariantCase{
	{"one", packOne, 16, 0, ""},
	{"second", packSecond, 4, 0, ""},
	{"one-demand", packOneDemand, 4, 0, ""},
	{"one+faucet-sunk", packOneFaucetSunk, 4, 0, ""},
	{"three", []string{"base.toml", "three.toml", "warm-three.toml"}, 4, 0, ""},
	{"two+trade", tradeWarm("two"), 4, 0, ""},
	{"three+trade", tradeWarm("three"), 4, 0, ""},
	// REF 7.5: famine runs 800 ticks with the shock of S14 at depth 0.
	{"famine", append(slices.Clone(packLayers["famine"]), "warm-famine.toml"), 4, 800, "s14-0.toml"},
}

// runInvariants runs one (pack, seed) case and fails with the seed, tick and
// invariant of the first tick that breaks one (testing.md 6.2).
func runInvariants(t *testing.T, cfg economysim.Config, seed uint64, ticks int64, script []shockStep) {
	t.Helper()
	rec := &recorder{}
	w, err := economysim.NewWorld(cfg, economysim.WithSeed(seed), economysim.WithObserver(rec), economysim.WithAudit())
	if err != nil {
		t.Fatalf("NewWorld(seed %d) error = %v, want nil", seed, err)
	}
	fail := func(tick int64, invariant string, msgs []string) {
		t.Helper()
		for _, m := range msgs {
			t.Errorf("seed %d tick %d: %s: %s (reproduce with WithSeed(%d), WithAudit)", seed, tick, invariant, m, seed)
		}
		if len(msgs) > 0 {
			t.FailNow()
		}
	}
	pre, err := w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot before the first step error = %v", err)
	}
	c := newInvChecker(pre)
	c.events(rec.events)
	rec.forget()
	fail(0, "conservation (before the first step)", c.balances(pre))
	for i := int64(0); i < ticks; i++ {
		applyScript(t, w, pre.Names, script)
		if err := w.Step(); err != nil {
			t.Fatalf("seed %d tick %d: Step error = %v, want nil (reproduce with WithSeed(%d))", seed, i, err, seed)
		}
		if got := w.Tick(); got != i+1 {
			t.Fatalf("seed %d: Tick() after step %d = %d, want %d", seed, i, got, i+1)
		}
		s, err := w.Snapshot()
		if err != nil {
			t.Fatalf("seed %d tick %d: Snapshot error = %v", seed, i, err)
		}
		fail(i, "money flow", noMoneyFlowAfterConstruction(withoutConfiguredFlows(cfg, rec.events)))
		fail(i, "agents, orders and prices", c.events(rec.events))
		rec.forget()
		fail(i, "conservation", c.balances(s))
		fail(i, "negative balance or stock", economysim.NegativeAccounts(w))
		if i == 0 {
			twin, err := economysim.NewWorld(cfg, economysim.WithSeed(seed))
			if err != nil || twin.Step() != nil {
				t.Fatalf("seed %d: twin world: %v", seed, err)
			}
			ts, err := twin.Snapshot()
			if err != nil || !reflect.DeepEqual(ts, s) {
				fail(i, "determinism of the first tick", []string{"a second world with the same seed has another snapshot"})
			}
		}
	}
	t.Logf("seed %d: %d tick(s); trades checked against both limits: %d, one limit only: %d, none: %d", seed, ticks, c.bothSided, c.oneSided, c.noSide)
}

// TestInvariants_Packs runs the invariants of testing.md 6.3 over the packs
// and seeds of REF 7.5 (one 1 to 16, second and three 1 to 4), 400 ticks
// each: goods and money conserved with exact equality, FR-OBS-042 per reason,
// no negative balance or stock, no trade with a missing party, the clearing
// price between the matched limits, the first tick repeatable, and Step nil
// with the tick advancing by one.
//
// Requirement: FR-REF-028, FR-REF-025, FR-REF-026, FR-STB-008, FR-OBS-042, FR-OBS-015, FR-LED-035, FR-LED-045, FR-LED-046, FR-CLR-019
// Guard: a unit created without a boundary reason breaks conservation; a trade
// price outside the matched limits breaks the price check.
//
// Under -race each row runs its first invariantSeeds(seeds) seeds (plan 2-1
// D226); gate F runs them all.
func TestInvariants_Packs(t *testing.T) {
	if testing.Short() {
		t.Skip("end-to-end: 400 ticks per (pack, seed); run without -short")
	}
	t.Parallel()
	for _, tc := range invariantTable {
		cfg := loadValidPack(t, tc.layers...)
		ticks, script := invTicks, []shockStep(nil)
		if tc.ticks > 0 {
			ticks = tc.ticks
		}
		if tc.script != "" {
			script = loadScript(t, tc.script)
		}
		for seed := uint64(1); seed <= uint64(invariantSeeds(tc.seeds)); seed++ {
			t.Run(fmt.Sprintf("%s/seed%d", tc.name, seed), func(t *testing.T) {
				t.Parallel()
				runInvariants(t, cfg, seed, ticks, script)
			})
		}
	}
}

// TestInvariantChecker_Balances verifies the conservation checks on
// hand-built snapshots: balanced, one unit off, and balanced with a nonzero
// pending that identity (i) must not count.
//
// Requirement: FR-REF-028, FR-OBS-042, FR-LED-045
func TestInvariantChecker_Balances(t *testing.T) {
	t.Parallel()
	const wood = economysim.CommodityID(0)
	market := func(held economysim.Quantity, pending economysim.Quantity) economysim.MarketRow {
		return economysim.MarketRow{State: economysim.MarketState{Commodity: wood}, HeldByAgents: held, Pending: pending}
	}
	snap := func(money economysim.Money, held, pending economysim.Quantity, goodsBalance, moneyBalance int64) economysim.Snapshot {
		return economysim.Snapshot{
			Tick:    1,
			Money:   money,
			Markets: []economysim.MarketRow{market(held, pending)},
			Boundary: []economysim.BoundaryRow{
				{Reason: economysim.ReasonInitial, Kind: economysim.KindFaucet, Commodity: wood, Balance: goodsBalance},
				{Reason: economysim.ReasonInitial, Kind: economysim.KindFaucet, Commodity: economysim.NoCommodity, Balance: moneyBalance},
			},
		}
	}
	// The faucet put 10 wood and 100 money in at construction.
	build := func() *invChecker {
		return newInvChecker(economysim.Snapshot{Construction: []economysim.FlowTotal{
			{Reason: economysim.ReasonInitial, Kind: economysim.KindFaucet, Commodity: wood, Total: 10},
			{Reason: economysim.ReasonInitial, Kind: economysim.KindFaucet, Commodity: economysim.NoCommodity, Total: 100},
		}})
	}
	tests := []struct {
		name string
		s    economysim.Snapshot
		want int
	}{
		{"balanced", snap(100, 10, 0, -10, -100), 0},
		{"one unit off", snap(100, 9, 0, -10, -100), 1},
		{"one coin off", snap(99, 10, 0, -10, -100), 1},
		{"pending excluded from identity (i)", snap(100, 10, 7, -10, -100), 0},
		{"balance moved without a flow", snap(101, 10, 0, -10, -101), 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := build()
			c.bucketChecked = true
			got := c.balances(tt.s)
			if len(got) != tt.want {
				t.Errorf("balances = %q, want %d failure(s)", got, tt.want)
			}
		})
	}
}

// TestInvariantChecker_Flows verifies that flows of later ticks and Pending
// move the expected balance, that the Tick -1 events must equal the
// construction bucket, and that an undeclared boundary is reported.
//
// Requirement: FR-OBS-042, FR-OBS-015
func TestInvariantChecker_Flows(t *testing.T) {
	t.Parallel()
	const wood = economysim.CommodityID(0)
	pre := economysim.Snapshot{Construction: []economysim.FlowTotal{{Reason: economysim.ReasonInitial, Kind: economysim.KindFaucet, Commodity: wood, Total: 10}}}
	row := func(r economysim.Reason, bal int64) economysim.BoundaryRow {
		return economysim.BoundaryRow{Reason: r, Kind: economysim.KindFaucet, Commodity: wood, Balance: bal}
	}
	flow := func(tick int64, r economysim.Reason, k economysim.Kind, n int64) economysim.Event {
		return economysim.BoundaryFlow{Tick: tick, Reason: r, Kind: k, Commodity: wood, Total: n}
	}
	c := newInvChecker(pre)
	c.events([]economysim.Event{flow(-1, economysim.ReasonInitial, economysim.KindFaucet, 10)})
	c.events([]economysim.Event{flow(0, economysim.ReasonConsumption, economysim.KindSink, 4)})
	s := economysim.Snapshot{
		Tick:     1,
		Markets:  []economysim.MarketRow{{State: economysim.MarketState{Commodity: wood}, HeldByAgents: 6}},
		Boundary: []economysim.BoundaryRow{row(economysim.ReasonInitial, -10), row(economysim.ReasonConsumption, 4)},
	}
	if got := c.balances(s); len(got) != 0 {
		t.Errorf("balances = %q, want none", got)
	}
	s.Pending = []economysim.FlowTotal{{Reason: economysim.ReasonConsumption, Kind: economysim.KindSink, Commodity: wood, Total: 1}}
	if got := c.balances(s); len(got) != 1 {
		t.Errorf("balances with an unbalanced pending = %q, want 1 failure", got)
	}
	bad := newInvChecker(pre)
	bad.events([]economysim.Event{flow(-1, economysim.ReasonInitial, economysim.KindFaucet, 9)})
	s.Pending = nil
	if got := bad.balances(s); len(got) == 0 {
		t.Error("a Tick -1 bucket that differs from Construction was not reported")
	}
	und := newInvChecker(pre)
	und.events([]economysim.Event{flow(0, economysim.ReasonDisposal, economysim.KindSink, 1)})
	if got := und.balances(s); len(got) == 0 {
		t.Error("a flow of an undeclared boundary was not reported")
	}
}

// TestInvariantChecker_Events verifies the live set rebuilt from
// AgentSpawned and AgentBankrupted, the estate exemption, and the
// price-between-limits check on constructed events: equal to the bid, equal
// to the ask, one above, one below.
//
// Requirement: FR-REF-028, FR-CLR-019
func TestInvariantChecker_Events(t *testing.T) {
	t.Parallel()
	const wood = economysim.CommodityID(0)
	order := func(a economysim.AgentID, side economysim.Side, limit economysim.Price) economysim.Event {
		return economysim.OrderPlaced{Tick: 3, Commodity: wood, Agent: a, Side: side, Limit: limit}
	}
	trade := func(buyer, seller economysim.AgentID, p economysim.Price) economysim.Event {
		return economysim.TradeExecuted{Tick: 3, Commodity: wood, Buyer: buyer, Seller: seller, Price: p, Quantity: 1}
	}
	spawn := func(a economysim.AgentID) economysim.Event { return economysim.AgentSpawned{Agent: a} }
	newC := func() *invChecker { return &invChecker{regions: 1, live: map[economysim.AgentID]bool{}} }
	tests := []struct {
		name string
		evs  []economysim.Event
		want int
	}{
		{"price equals bid", []economysim.Event{spawn(5), spawn(6), order(5, economysim.Bid, 10), order(6, economysim.Ask, 8), trade(5, 6, 10)}, 0},
		{"price equals ask", []economysim.Event{spawn(5), spawn(6), order(5, economysim.Bid, 10), order(6, economysim.Ask, 8), trade(5, 6, 8)}, 0},
		{"one above the bid", []economysim.Event{spawn(5), spawn(6), order(5, economysim.Bid, 10), order(6, economysim.Ask, 8), trade(5, 6, 11)}, 1},
		{"one below the ask", []economysim.Event{spawn(5), spawn(6), order(5, economysim.Bid, 10), order(6, economysim.Ask, 8), trade(5, 6, 7)}, 1},
		{"unknown buyer", []economysim.Event{spawn(6), trade(5, 6, 9)}, 1},
		{"bankrupt seller is gone", []economysim.Event{spawn(5), spawn(6), economysim.AgentBankrupted{Agent: 6}, trade(5, 6, 9)}, 1},
		{"estate seller needs no live entry", []economysim.Event{spawn(5), order(5, economysim.Bid, 10), trade(5, 0, 9)}, 0},
		{"estate seller above the bid", []economysim.Event{spawn(5), order(5, economysim.Bid, 10), trade(5, 0, 11)}, 1},
		{"rejected by overflow", []economysim.Event{economysim.OrderRejected{Tick: 3, Agent: 5, Reason: economysim.ReasonOverflow}}, 1},
		{"rejected by funds is fine", []economysim.Event{economysim.OrderRejected{Tick: 3, Agent: 5, Reason: economysim.ReasonInsufficientFunds}}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := newC().events(tt.evs); len(got) != tt.want {
				t.Errorf("events = %q, want %d failure(s)", got, tt.want)
			}
		})
	}
}

// TestInvariantChecker_NoMoneyFlowAfterConstruction verifies the money-flow
// check on constructed events: a money flow in a step (Tick >= 0) is flagged and
// names that tick; the construction bucket (Tick -1), a goods flow and an empty
// step are not.
//
// Requirement: FR-REF-028, FR-OBS-042
// Guard: the packs have no money faucet or sink after construction; without the
// test a check that flagged nothing would let one through.
func TestInvariantChecker_NoMoneyFlowAfterConstruction(t *testing.T) {
	t.Parallel()
	money := func(tick int64) economysim.Event {
		return economysim.BoundaryFlow{Tick: tick, Reason: "grant", Kind: economysim.KindFaucet, Commodity: economysim.NoCommodity, Total: 5}
	}
	goods := economysim.BoundaryFlow{Tick: 4, Reason: "harvest", Kind: economysim.KindFaucet, Commodity: 0, Total: 5}
	tests := []struct {
		name string
		evs  []economysim.Event
		want int
	}{
		{"no events", nil, 0},
		{"construction bucket", []economysim.Event{money(-1)}, 0},
		{"goods flow in a step", []economysim.Event{goods}, 0},
		{"money flow at tick 0", []economysim.Event{money(0)}, 1},
		{"money flow in a later step", []economysim.Event{goods, money(7)}, 1},
		{"two money flows", []economysim.Event{money(2), money(2)}, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := noMoneyFlowAfterConstruction(tt.evs)
			if len(got) != tt.want {
				t.Fatalf("noMoneyFlowAfterConstruction = %q, want %d message(s)", got, tt.want)
			}
			if tt.name == "money flow in a later step" && !strings.Contains(got[0], "tick 7") {
				t.Errorf("message = %q, want it to name tick 7", got[0])
			}
		})
	}
}
