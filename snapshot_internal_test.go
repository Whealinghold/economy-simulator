package economysim

import (
	"reflect"
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/beliefs"
	"github.com/Whealinghold/economy-sim/internal/ledger"
)

// TestSnapshot_BoundaryPendingConstruction verifies the three flow sections on
// the shell world: before the first step Construction holds the endowment's
// totals and Pending is empty; after a step Construction is empty; money minted
// between ticks is in Pending, as the BoundaryFlow of the next tick will carry
// it, and in the boundary balance.
//
// Arithmetic: Construction holds the money the endowment minted, B; the boundary
// balance of `initial` is -B, and minting 5 more makes it -(B + 5) with Pending
// one faucet row of 5.
//
// Requirement: FR-OBS-042, FR-OBS-043, FR-WLD-010
func TestSnapshot_BoundaryPendingConstruction(t *testing.T) {
	t.Parallel()
	w := internalShell(t)
	s, err := w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot() error = %v, want nil", err)
	}
	var built int64
	for _, f := range s.Construction {
		if f.Kind == KindFaucet && f.Commodity == NoCommodity {
			built += f.Total
		}
	}
	if built == 0 || len(s.Pending) != 0 {
		t.Fatalf("before the first step: construction money %d, pending rows %d, want money above 0 and no pending", built, len(s.Pending))
	}
	if err := w.Step(); err != nil {
		t.Fatalf("Step() error = %v, want nil", err)
	}
	acct, err := w.ledger.Open(ledger.KindAgent, 0, 0)
	if err != nil {
		t.Fatalf("Open(agent) = %v, want nil", err)
	}
	src, _ := w.ledger.Boundary(ledger.Initial, ledger.Money)
	if err := w.ledger.Transfer(src, acct, 5, ledger.Initial); err != nil {
		t.Fatalf("Transfer() = %v, want nil", err)
	}
	s, err = w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot() error = %v, want nil", err)
	}
	wantPending := []FlowTotal{{Region: 0, Reason: ReasonInitial, Kind: KindFaucet, Commodity: NoCommodity, Total: 5}}
	if len(s.Construction) != 0 || !reflect.DeepEqual(s.Pending, wantPending) {
		t.Errorf("after a step and a mint: Construction = %+v, Pending = %+v, want none and %+v", s.Construction, s.Pending, wantPending)
	}
	for _, b := range s.Boundary {
		if b.Reason == ReasonInitial && b.Commodity == NoCommodity && b.Balance != -(built+5) {
			t.Errorf("boundary initial money = %d, want %d", b.Balance, -(built + 5))
		}
	}
}

// TestSnapshot_PoolIsTheEstateAccount verifies that the pool a region reports is
// the money of its estate account (FR-EST-013), that the money of the region's
// households is the sum of its agents' accounts, and that the header money is
// the sum of both over the regions.
//
// Requirement: FR-EST-013, FR-OBS-041
func TestSnapshot_PoolIsTheEstateAccount(t *testing.T) {
	t.Parallel()
	w := internalShell(t)
	if err := w.estates.seedPool(w.ledger, 0, 1_234); err != nil {
		t.Fatalf("seedPool() error = %v, want nil", err)
	}
	s, err := w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot() error = %v, want nil", err)
	}
	v, err := w.ledger.Account(w.estates.account(0))
	if err != nil {
		t.Fatalf("Account(estate) error = %v, want nil", err)
	}
	var held int64
	for i := len(w.cfg.Regions); i < len(w.agents); i++ {
		a, _ := w.ledger.Account(w.agents[i].account)
		held += a.Balance
	}
	r := s.Regions[0]
	if int64(r.MoneyPool) != v.Balance || int64(r.MoneyHouseholds) != held || int64(s.Money) != v.Balance+held || v.Balance != 1_234 {
		t.Errorf("pool %d, households %d, header %d, want %d, %d and %d", r.MoneyPool, r.MoneyHouseholds, s.Money, v.Balance, held, v.Balance+held)
	}
}

// TestSnapshot_EstateAtFloorCountsUnitsListedAtTheResolutionFloor verifies that
// EstateAtFloor is 0 while the estate's lots are listed above the resolution floor
// or not listed yet, equals HeldByEstate once every listing sits at the floor, and
// is never above HeldByEstate.
//
// Arithmetic: agent 3's goods go to the estate of region 0; its first asks list
// each lot at the reference price, well above the floor, so the field is 0. Setting
// every listing price to the floor makes every held unit count, so the field equals
// HeldByEstate.
//
// Requirement: FR-OBS-041
// Guard: without the field the estate's markdown at its floor is invisible to a
// consumer, and economysim.estate.at_floor has nothing to read.
func TestSnapshot_EstateAtFloorCountsUnitsListedAtTheResolutionFloor(t *testing.T) {
	t.Parallel()
	w := newTrader(t, tradeConfig(3), func(*AgentView, *TradePlan) {})
	if _, _, err := w.estates.receive(w.ledger, w.agents[3].account, 0); err != nil {
		t.Fatalf("receive() error = %v, want nil", err)
	}
	at := func() (held, atFloor Quantity) {
		s, err := w.Snapshot()
		if err != nil {
			t.Fatalf("Snapshot() error = %v, want nil", err)
		}
		m := s.Markets[w.markets.index(0, grain)]
		return m.HeldByEstate, m.EstateAtFloor
	}
	if held, got := at(); held < 1 || got != 0 {
		t.Fatalf("before listing: HeldByEstate %d, EstateAtFloor %d, want held above 0 and 0", held, got)
	}
	if _, err := w.estates.asks(w.ledger, &w.cfg, w.refs, 0, nil); err != nil {
		t.Fatalf("asks() error = %v, want nil", err)
	}
	if held, got := at(); held < 1 || got != 0 {
		t.Errorf("listed at the reference price: HeldByEstate %d, EstateAtFloor %d, want held above 0 and 0", held, got)
	}
	floor, _ := resolutionFloor(w.cfg.World)
	for i := range w.estates.listed[0] {
		w.estates.listed[0][i].price = Price(floor)
	}
	if held, got := at(); got != held {
		t.Errorf("listed at the floor: EstateAtFloor %d, want HeldByEstate %d", got, held)
	}
}

// TestSnapshot_MarketLotsEqualTheLotsTheLedgerHolds verifies that Markets[].Lots
// of a commodity equals the lots of that commodity in the accounts the held
// fields count (the agents' and the estate's), counted here straight from the
// ledger, and that the estate's lots are among them.
//
// Arithmetic: three households each start with 10 flour and 10 grain, so the
// start stock is lots of both commodities in every account; agent 3's goods then
// move to the estate of region 0, which keeps them as lots of the same keys. The
// expected count is a fresh walk over the agent accounts and the estate account,
// not the snapshot's accumulator.
//
// Requirement: FR-OBS-093
// Guard: Lots is the denominator of economysim.ledger.lots; a count that drifted
// from the ledger would make the gauge disagree with the ledger silently.
func TestSnapshot_MarketLotsEqualTheLotsTheLedgerHolds(t *testing.T) {
	t.Parallel()
	w := newTrader(t, tradeConfig(3), func(*AgentView, *TradePlan) {})
	if _, _, err := w.estates.receive(w.ledger, w.agents[3].account, 0); err != nil {
		t.Fatalf("receive() error = %v, want nil", err)
	}
	want := make([]int64, len(w.cfg.Commodities))
	accounts := []int{w.estates.accounts[0]}
	for i := len(w.cfg.Regions); i < len(w.agents); i++ {
		accounts = append(accounts, w.agents[i].account)
	}
	for _, a := range accounts {
		lots, err := w.ledger.AppendLots(nil, a)
		if err != nil {
			t.Fatalf("AppendLots(%d) error = %v, want nil", a, err)
		}
		for _, l := range lots {
			want[l.Commodity]++
		}
	}
	s, err := w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot() error = %v, want nil", err)
	}
	var total int64
	for c := range want {
		got := s.Markets[w.markets.index(0, CommodityID(c))].Lots
		if got != want[c] {
			t.Errorf("Snapshot().Markets[commodity %d].Lots = %d, want %d (the ledger's lots)", c, got, want[c])
		}
		total += got
	}
	if total < 1 {
		t.Errorf("sum of Markets[].Lots = %d, want above 0: the scenario holds goods", total)
	}
}

// TestSnapshot_RolesRegionsAndBeliefsOfAScriptedWorld verifies the counts, the
// hunger signals and the belief statistics of the snapshot on the two-region world
// of removalConfig, with each agent's state set by hand so that every total has
// one right answer and a total that leaks from one region or role into another, or
// misses an agent, shows.
//
// Agents (weight, region, role): 2 (2, home, miller), 3 (1, home, miller),
// 4 (1, home, weaver), 5 (2, yonder, miller). Set by hand, (unfed, hunger,
// privation): agent 2 (3, 2, 5), agent 3 (0, 7, 1), agent 4 (1, 1, 0), agent 5
// (2, 3, 4). Expected role rows: home miller 2 agents, 3 households, Unfed 3 + 0 =
// 3, UnfedAgents 1 (only agent 2), MaxHunger max(2, 7) = 7, MaxPrivation max(5, 1) =
// 5; home weaver 1 agent, 1 household, Unfed 1, UnfedAgents 1, MaxHunger 1,
// MaxPrivation 0; yonder miller 1 agent, 2 households, Unfed 2, UnfedAgents 1,
// MaxHunger 3, MaxPrivation 4; yonder weaver empty. Region rows: home 3 agents and
// 4 households, yonder 1 and 2.
//
// Beliefs of flour (commodity 0, only millers trade it): agent 2 holds 3 000 and
// agent 3 holds 2 001 (home), agent 5 holds 3 500 (yonder). Home: n 2, low 2 001
// (not the first agent's), high 3 000 (the first agent's), mean (3 000 + 2 001) / 2
// = 2 500.5, which rounds half to even to 2 500. Yonder: n 1 and all three
// statistics 3 500. Of pan (commodity 2) agent 2 holds 4 000 and agent 3 holds
// 5 000: low 4 000 (the first), high 5 000, mean 4 500. Of grain (commodity 1)
// agent 2 sits on the ceiling of its range, agent 3 on the floor and agent 4 at
// 1 000 (home, n 3): AtFloor 1, AtCeiling 1, low the floor, high the ceiling, mean
// (floor + ceiling + 1 000) / 3 half to even.
//
// Requirement: FR-OBS-040, FR-OBS-041, FR-OBS-042
// Guard: the snapshot's totals are what the gauges and the spawner read; the
// per-region accumulator must start empty for each region.
func TestSnapshot_RolesRegionsAndBeliefsOfAScriptedWorld(t *testing.T) {
	t.Parallel()
	w := newMill(t, removalConfig())
	const flourC, grainC, panC = 0, 1, 2
	for id, st := range map[AgentID][3]int64{2: {3, 2, 5}, 3: {0, 7, 1}, 4: {1, 1, 0}, 5: {2, 3, 4}} {
		a := &w.agents[id]
		a.unfed, a.hunger, a.privation = st[0], st[1], st[2]
	}
	setCentre := func(id AgentID, c CommodityID, centre int64) {
		i := slices.Index(w.roles.row(w.agents[id].role).tradable, c)
		if i < 0 {
			t.Fatalf("agent %d does not trade commodity %d", id, c)
		}
		w.agents[id].beliefs[i].Centre = centre
	}
	grain := beliefs.Limits(int64(w.cfg.Commodities[grainC].ReferencePrice), 0, w.bp)
	setCentre(2, flourC, 3_000)
	setCentre(3, flourC, 2_001)
	setCentre(5, flourC, 3_500)
	setCentre(2, panC, 4_000)
	setCentre(3, panC, 5_000)
	setCentre(2, grainC, grain.Ceiling)
	setCentre(3, grainC, grain.Floor)
	setCentre(4, grainC, 1_000)
	s, err := w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot() error = %v, want nil", err)
	}
	roles := []struct {
		region                          RegionID
		role                            RoleID
		agents, households, unfed, ufag int64
		hunger, privation               int64
	}{
		{0, 0, 2, 3, 3, 1, 7, 5}, {0, 1, 1, 1, 1, 1, 1, 0}, {1, 0, 1, 2, 2, 1, 3, 4}, {1, 1, 0, 0, 0, 0, 0, 0},
	}
	if len(s.Roles) != len(roles) {
		t.Fatalf("Snapshot().Roles has %d rows, want %d", len(s.Roles), len(roles))
	}
	for i, want := range roles {
		got := s.Roles[i]
		if got.Region != want.region || got.Role != want.role || got.Agents != want.agents || got.Households != want.households ||
			got.Unfed != want.unfed || got.UnfedAgents != want.ufag || got.MaxHunger != want.hunger || got.MaxPrivation != want.privation {
			t.Errorf("Snapshot().Roles[%d] = %+v, want region %d role %d agents %d households %d unfed %d unfed agents %d max hunger %d max privation %d",
				i, got, want.region, want.role, want.agents, want.households, want.unfed, want.ufag, want.hunger, want.privation)
		}
	}
	for g, want := range [][2]int64{{3, 4}, {1, 2}} {
		if got := s.Regions[g]; got.Agents != want[0] || got.Households != want[1] {
			t.Errorf("Snapshot().Regions[%d] agents, households = %d, %d, want %d, %d", g, got.Agents, got.Households, want[0], want[1])
		}
	}
	if s.Agents != 4 || s.Households != 6 {
		t.Errorf("Snapshot() agents, households = %d, %d, want 4, 6", s.Agents, s.Households)
	}
	// The mean of three is the sum over 3, rounded half to even; a remainder of 0 to
	// 2 never sits on a half, so the nearest integer is the answer.
	sum := grain.Ceiling + grain.Floor + 1_000
	grainMeanRounded := sum / 3
	if sum%3 == 2 {
		grainMeanRounded++
	}
	markets := []struct {
		region                RegionID
		commodity             CommodityID
		n, atFloor, atCeiling int64
		low, high, mean       Price
	}{
		{0, flourC, 2, 0, 0, 2_001, 3_000, 2_500},
		{0, panC, 2, 0, 0, 4_000, 5_000, 4_500},
		{0, grainC, 3, 1, 1, Price(grain.Floor), Price(grain.Ceiling), Price(grainMeanRounded)},
		{1, flourC, 1, 0, 0, 3_500, 3_500, 3_500},
	}
	for _, want := range markets {
		got := s.Markets[w.markets.index(want.region, want.commodity)]
		if got.Beliefs != want.n || got.AtFloor != want.atFloor || got.AtCeiling != want.atCeiling ||
			got.LowCentre != want.low || got.HighCentre != want.high || got.MeanCentre != want.mean {
			t.Errorf("Snapshot().Markets[region %d, commodity %d] beliefs = n %d, floor %d, ceiling %d, low %d, high %d, mean %d, want n %d, floor %d, ceiling %d, low %d, high %d, mean %d",
				want.region, want.commodity, got.Beliefs, got.AtFloor, got.AtCeiling, got.LowCentre, got.HighCentre, got.MeanCentre,
				want.n, want.atFloor, want.atCeiling, want.low, want.high, want.mean)
		}
	}
}

// TestSnapshot_JobsPendingEstateAndMoneyByRegion verifies, on the two-region world
// of removalConfig after one step with a recipe of 3 ticks, that the job rows, the
// pending output, the estate's holdings and the money are counted for the region
// they belong to, and that nothing carried over from the region before.
//
// Arithmetic: a mill execution takes 2 grain and gives 3 flour after 3 ticks. The
// default strategy starts every miller's request, so after tick 0 agent 2 (home,
// weight 2) has 2 executions, agent 3 (home, weight 1) has 1 and agent 5 (yonder,
// weight 2) has 2. Job rows: home 2 agents, 2 + 1 = 3 executions, 2 + 1 = 3
// households; yonder 1 agent, 2 executions, 2 households. Pending flour: home
// 3 * 3 = 9, yonder 2 * 3 = 6. Then agent 3's goods go to the estate of home: its
// units of each commodity are HeldByEstate of home, read from the estate's lots,
// and yonder's estate holds none. Listing every lot at the resolution floor makes
// EstateAtFloor of home equal to HeldByEstate and leaves yonder's 0.
//
// Requirement: FR-OBS-041, FR-OBS-042, FR-PRO-042
// Guard: a per-region accumulator that kept the last region's jobs, pending output
// or estate would report them twice.
func TestSnapshot_JobsPendingEstateAndMoneyByRegion(t *testing.T) {
	t.Parallel()
	c := removalConfig()
	c.Recipes[0].Duration = 3
	w := newMill(t, c)
	step(t, w, 1)
	if _, _, err := w.estates.receive(w.ledger, w.agents[3].account, 0); err != nil {
		t.Fatalf("receive() error = %v, want nil", err)
	}
	if _, err := w.estates.asks(w.ledger, &w.cfg, w.refs, 0, nil); err != nil {
		t.Fatalf("asks() error = %v, want nil", err)
	}
	floor, _ := resolutionFloor(w.cfg.World)
	for i := range w.estates.listed[0] {
		w.estates.listed[0][i].price = Price(floor)
	}
	s, err := w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot() error = %v, want nil", err)
	}
	if want := []JobRow{{Region: 0, Recipe: 0, Agents: 2, Executions: 3, Households: 3}, {Region: 1, Recipe: 0, Agents: 1, Executions: 2, Households: 2}}; !slices.Equal(s.Jobs, want) {
		t.Errorf("Snapshot().Jobs = %+v, want %+v", s.Jobs, want)
	}
	estate := map[CommodityID]Quantity{}
	for _, l := range w.estates.heldLots(w.ledger, 0) {
		estate[CommodityID(l.Commodity)] += Quantity(l.Quantity)
	}
	if len(estate) == 0 {
		t.Fatalf("the estate of home holds nothing after the receive, want agent 3's goods")
	}
	for c := range w.cfg.Commodities {
		home, yonder := s.Markets[w.markets.index(0, CommodityID(c))], s.Markets[w.markets.index(1, CommodityID(c))]
		if home.HeldByEstate != estate[CommodityID(c)] || home.EstateAtFloor != home.HeldByEstate {
			t.Errorf("home commodity %d: HeldByEstate %d, EstateAtFloor %d, want %d and the same", c, home.HeldByEstate, home.EstateAtFloor, estate[CommodityID(c)])
		}
		if yonder.HeldByEstate != 0 || yonder.EstateAtFloor != 0 {
			t.Errorf("yonder commodity %d: HeldByEstate %d, EstateAtFloor %d, want 0 and 0: nothing carries over from home", c, yonder.HeldByEstate, yonder.EstateAtFloor)
		}
	}
	if got, want := [2]Quantity{s.Markets[w.markets.index(0, flour)].Pending, s.Markets[w.markets.index(1, flour)].Pending}, [2]Quantity{9, 6}; got != want {
		t.Errorf("pending flour of home and yonder = %v, want %v", got, want)
	}
	for _, g := range []RegionID{0, 1} {
		if p := s.Markets[w.markets.index(g, grain)].Pending; p != 0 {
			t.Errorf("pending grain of region %d = %d, want 0: no recipe makes it", g, p)
		}
	}
	// What the agents hold, counted from the ledger one agent at a time and grouped by
	// the region of the account, and the lots of the agents and of the estate.
	held := make([][]Quantity, 2)
	lots := make([][]int64, 2)
	for g := range held {
		held[g], lots[g] = make([]Quantity, len(w.cfg.Commodities)), make([]int64, len(w.cfg.Commodities))
		for _, l := range w.estates.heldLots(w.ledger, RegionID(g)) {
			lots[g][l.Commodity]++
		}
	}
	for i := len(w.cfg.Regions); i < len(w.agents); i++ {
		a := &w.agents[i]
		g := w.regionOf(a)
		ls, err := w.ledger.AppendLots(nil, a.account)
		if err != nil {
			t.Fatalf("AppendLots(agent %d) error = %v, want nil", i, err)
		}
		for _, l := range ls {
			held[g][l.Commodity] += Quantity(l.Quantity)
			lots[g][l.Commodity]++
		}
	}
	for g := range held {
		for c := range w.cfg.Commodities {
			m := s.Markets[w.markets.index(RegionID(g), CommodityID(c))]
			if m.HeldByAgents != held[g][c] || m.Lots != lots[g][c] {
				t.Errorf("region %d commodity %d: HeldByAgents %d, Lots %d, want %d and %d (counted from the ledger)", g, c, m.HeldByAgents, m.Lots, held[g][c], lots[g][c])
			}
		}
	}
	var households, pool [2]int64
	for i := len(w.cfg.Regions); i < len(w.agents); i++ {
		a := &w.agents[i]
		v, err := w.ledger.Account(a.account)
		if err != nil {
			t.Fatalf("Account(agent %d) error = %v, want nil", i, err)
		}
		households[w.regionOf(a)] += v.Balance
	}
	var total int64
	for g := range households {
		v, _ := w.ledger.Account(w.estates.accounts[g])
		pool[g] = v.Balance
		if got := s.Regions[g]; int64(got.MoneyHouseholds) != households[g] || int64(got.MoneyPool) != pool[g] {
			t.Errorf("region %d money = households %d, pool %d, want %d and %d (the ledger's)", g, got.MoneyHouseholds, got.MoneyPool, households[g], pool[g])
		}
		total += households[g] + pool[g]
	}
	if int64(s.Money) != total || total == 0 {
		t.Errorf("Snapshot().Money = %d, want %d, the households' money plus the pools'", s.Money, total)
	}
}

// TestSnapshot_BoundaryRowsAfterTwoSteps verifies the whole boundary section for
// the mill world (2 households, recipe of 1 tick) after two steps: money first,
// then the goods by commodity (flour 0, grain 1), the reasons in registry order,
// each row a faucet (balance 0 or negative: what entered) or a sink (positive:
// what left), and the money rows only for the reasons that are open.
//
// Arithmetic: each of 2 agents starts with 10 000 money and 10 grain, so initial
// money is -20 000 and initial grain -20. In each of the 2 ticks each agent runs
// 1 execution (2 grain in, 3 flour out) and eats 1 grain: recipe_input grain
// 2 * 2 * 2 = 8, recipe_output flour 2 * 2 * 3 = 12 (a faucet, so -12) and
// consumption grain 2 * 2 * 1 = 4. Nothing is discarded, spoiled or worn.
//
// Requirement: FR-OBS-042, FR-LED-052
// Guard: the section is what a consumer's conservation check reads; a row in the
// wrong order, of the wrong kind or for the wrong commodity would break it.
func TestSnapshot_BoundaryRowsAfterTwoSteps(t *testing.T) {
	t.Parallel()
	w := newMill(t, millConfig(1, 2))
	step(t, w, 2)
	s, err := w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot() error = %v, want nil", err)
	}
	fa, si := KindFaucet, KindSink
	want := []BoundaryRow{
		{ReasonInitial, fa, NoCommodity, -20_000},
		{ReasonInitial, fa, flour, 0}, {"recipe_input", si, flour, 0}, {"recipe_output", fa, flour, -12},
		{"disposal", si, flour, 0}, {"consumption", si, flour, 0}, {"spoilage", si, flour, 0}, {"wear", si, flour, 0},
		{ReasonInitial, fa, grain, -20}, {"recipe_input", si, grain, 8}, {"recipe_output", fa, grain, 0},
		{"disposal", si, grain, 0}, {"consumption", si, grain, 4}, {"spoilage", si, grain, 0}, {"wear", si, grain, 0},
	}
	if !slices.Equal(s.Boundary, want) {
		t.Errorf("Snapshot().Boundary = %+v, want %+v", s.Boundary, want)
	}
}

// TestSnapshot_BeliefBoundsOfAProducedGoodUseTheCostAnchor verifies that a belief
// sitting on the floor of its range is counted at the floor, where the range of a
// good the role makes depends on the cost anchor of its recipe, and the range of a
// good it only buys does not.
//
// Arithmetic: with the starting wage 100 the anchor of flour is ceil((2 * 1 000 +
// 100) / 3) = 700 (see TestP12Anchors_WageAndPrices). The floor of the belief range
// for a reference price of 3 000 and an anchor of 700 differs from the floor with
// no anchor, which the test checks, so only a count that used the anchor finds the
// belief on the floor. Grain, which no recipe makes, has no anchor.
//
// Requirement: FR-OBS-042, FR-BEL-032
func TestSnapshot_BeliefBoundsOfAProducedGoodUseTheCostAnchor(t *testing.T) {
	t.Parallel()
	w := newMill(t, millConfig(1, 2))
	w.seedWage[0] = 100
	w.p12Anchors()
	if got := w.prod.unitCost(0, flour); got != 700 {
		t.Fatalf("unit cost of flour = %d, want 700", got)
	}
	ref := int64(w.cfg.Commodities[flour].ReferencePrice)
	anchored, bare := beliefs.Limits(ref, 700, w.bp), beliefs.Limits(ref, 0, w.bp)
	if anchored == bare {
		t.Fatalf("limits with the anchor %+v equal the limits without %+v, want a different range", anchored, bare)
	}
	i := slices.Index(w.roles.row(0).tradable, CommodityID(flour))
	w.agents[1].beliefs[i].Centre = anchored.Floor
	w.agents[2].beliefs[i].Centre = anchored.Ceiling
	s, err := w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot() error = %v, want nil", err)
	}
	if m := s.Markets[w.markets.index(0, flour)]; m.AtFloor != 1 || m.AtCeiling != 1 {
		t.Errorf("flour beliefs at the floor, ceiling = %d, %d, want 1, 1: the range of a made good uses the anchor", m.AtFloor, m.AtCeiling)
	}
	j := slices.Index(w.roles.row(0).tradable, CommodityID(grain))
	g := beliefs.Limits(int64(w.cfg.Commodities[grain].ReferencePrice), 0, w.bp)
	w.agents[1].beliefs[j].Centre = g.Floor
	s, _ = w.Snapshot()
	if m := s.Markets[w.markets.index(0, grain)]; m.AtFloor != 1 {
		t.Errorf("grain beliefs at the floor = %d, want 1: the range of a bought good has no anchor", m.AtFloor)
	}
}
