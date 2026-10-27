package economysim_test

import (
	"log/slog"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// The tests of the money supply module's step through the public API on small
// Go-built worlds (spec-money-supply.md 5): who a flow reaches, when it runs, how
// much it moves, what it reports. The arithmetic of the dues is the unit tests of
// internal/moneyflow; the internals (scale setter, removal, audit) are in
// moneyflow_step_internal_test.go.

// moneyWorldConfig is the world of the step tests: minimalConfig with two roles,
// farmers and bakers, each household holding 1 000 grain so that nobody goes
// unfed in a few ticks and no money moves by itself (no recipe, no trade) except
// by a flow. flows are the module's flows.
func moneyWorldConfig(farmers, bakers int64, flows ...economysim.MoneyFlow) economysim.Config {
	c := minimalConfig()
	baker := economysim.DefaultRole()
	baker.ID = "baker"
	c.Roles = append(c.Roles, baker)
	for i := range c.Roles {
		c.Roles[i].StartStock = []economysim.Item{{Commodity: "grain", Quantity: 1_000}}
	}
	c.Groups = []economysim.Group{
		{Region: "home", Role: "farmer", Households: farmers},
		{Region: "home", Role: "baker", Households: bakers},
	}
	c.Modules.MoneySupply = &economysim.MoneySupply{Flows: flows}
	return c
}

// flowTo returns a flow of the kind aimed at the households of a role in every region.
func flowTo(id, kind, role string) economysim.MoneyFlow {
	return economysim.MoneyFlow{ID: id, Kind: kind, Target: economysim.FlowTarget{Kind: economysim.TargetHouseholds, Role: role}}
}

// moneyWorld builds a world of the step tests with the audit on.
func moneyWorld(t *testing.T, c economysim.Config, opts ...economysim.Option) *economysim.World {
	t.Helper()
	w, err := economysim.NewWorld(c, append([]economysim.Option{economysim.WithSeed(1), economysim.WithAudit()}, opts...)...)
	if err != nil {
		t.Fatalf("NewWorld error = %v, want nil", err)
	}
	return w
}

// flowRow returns the snapshot row of the flow with the ID.
func flowRow(t *testing.T, w *economysim.World, id string) economysim.FlowRow {
	t.Helper()
	s, err := w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot error = %v, want nil", err)
	}
	for _, f := range s.Flows {
		if f.Flow == id {
			return f
		}
	}
	t.Fatalf("Snapshot has no flow %q in %+v", id, s.Flows)
	return economysim.FlowRow{}
}

// roleMoney returns the money of the non-boundary accounts: the households and the estates.
func roleMoney(t *testing.T, w *economysim.World) economysim.Money {
	t.Helper()
	s, err := w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot error = %v, want nil", err)
	}
	return s.Money
}

// TestFlow_ClassIsRoleRegionAndWeight verifies that a grant reaches the
// households of its role and no other, and pays each account the weight times the
// per-household amount, so the flow's total is the amount times the households of
// the class and the ledger's boundary balance of the flow is minus that.
//
// Requirement: FR-MON-007, FR-MON-012, FR-MON-004
func TestFlow_ClassIsRoleRegionAndWeight(t *testing.T) {
	t.Parallel()
	// 25 farmer households make agents of weight above 1 (MinAgents 10 caps the
	// agents at 10): 7 a household-tick is 7 * 25 = 175 a tick, whatever the split.
	g := flowTo("stipend", economysim.FlowGrant, "farmer")
	g.Amount = 7
	w := moneyWorld(t, moneyWorldConfig(25, 10, g))
	before := roleMoney(t, w)
	if err := w.Step(); err != nil {
		t.Fatalf("Step error = %v, want nil", err)
	}
	if got := flowRow(t, w, "stipend").Total; got != 175 {
		t.Errorf("stipend Total after one tick = %d, want 7 * 25 = 175", got)
	}
	if got, want := roleMoney(t, w)-before, economysim.Money(175); got != want {
		t.Errorf("money held after one tick grew by %d, want %d (only the farmers were paid)", got, want)
	}
	s, err := w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot error = %v, want nil", err)
	}
	for _, b := range s.Boundary {
		if b.Reason == "stipend" && (b.Balance != -175 || b.Kind != economysim.KindFaucet) {
			t.Errorf("boundary row of stipend = %+v, want a faucet at -175", b)
		}
	}
}

// TestFlow_WindowAndStart verifies that a flow is active exactly from Start up to
// but not including Until, and changes no state outside the window.
//
// Requirement: FR-MON-010, FR-MON-044
func TestFlow_WindowAndStart(t *testing.T) {
	t.Parallel()
	g := flowTo("burst", economysim.FlowGrant, "farmer")
	g.Amount, g.Start, g.Until = 3, 2, 4
	w := moneyWorld(t, moneyWorldConfig(10, 10, g))
	// 3 a household-tick over 10 farmer households, ticks 2 and 3 only.
	for tick, want := range []economysim.Money{0, 0, 0, 30, 60, 60} {
		if got := flowRow(t, w, "burst").Total; got != want {
			t.Errorf("burst Total before tick %d = %d, want %d", tick, got, want)
		}
		if err := w.Step(); err != nil {
			t.Fatalf("Step at tick %d error = %v, want nil", tick, err)
		}
	}
}

// TestFlow_ConstructionBucketHoldsInitialOnly verifies that nothing of a flow is
// applied during construction: the bucket of a world built with the module on
// holds the initial reason only, and every flow's total is 0.
//
// Requirement: FR-MON-044
func TestFlow_ConstructionBucketHoldsInitialOnly(t *testing.T) {
	t.Parallel()
	g := flowTo("stipend", economysim.FlowGrant, "farmer")
	g.Amount = 5
	c := moneyWorldConfig(10, 10, g)
	c.Roles[0].StartMoney, c.Roles[1].StartMoney = 100, 100
	w := moneyWorld(t, c)
	s, err := w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot error = %v, want nil", err)
	}
	if len(s.Construction) == 0 {
		t.Fatal("construction bucket is empty, want the initial money")
	}
	for _, f := range s.Construction {
		if f.Reason != economysim.ReasonInitial {
			t.Errorf("construction bucket has reason %q, want only %q", f.Reason, economysim.ReasonInitial)
		}
	}
	if got := flowRow(t, w, "stipend").Total; got != 0 {
		t.Errorf("stipend Total before the first step = %d, want 0", got)
	}
}

// TestFlow_PerYearUsesCalendar verifies that a flow per year spreads over the
// ticks of the calendar's year, and that a flow per tick does not depend on it.
// An Amount of 3 * TicksPerYear a year is 3 a tick for each household, exactly.
//
// Requirement: FR-MON-014, FR-CUR-004
func TestFlow_PerYearUsesCalendar(t *testing.T) {
	t.Parallel()
	for _, daysPerTick := range []int64{1, 5} {
		c := moneyWorldConfig(10, 10)
		c.Calendar.DaysPerTick = daysPerTick
		for i := range c.Roles {
			c.Roles[i].ExpectedWorkTicks, c.Roles[i].MinWorkShare = 50, 1_000
		}
		year := c.Calendar.TicksPerYear()
		yr := flowTo("yearly", economysim.FlowGrant, "farmer")
		yr.Amount, yr.Per = 3*year, economysim.PerYear
		tk := flowTo("tickly", economysim.FlowGrant, "farmer")
		tk.Amount = 3
		c.Modules.MoneySupply.Flows = []economysim.MoneyFlow{tk, yr}
		w := moneyWorld(t, c)
		if err := w.Step(); err != nil {
			t.Fatalf("DaysPerTick %d: Step error = %v, want nil", daysPerTick, err)
		}
		for _, id := range []string{"tickly", "yearly"} {
			if got := flowRow(t, w, id).Total; got != 30 {
				t.Errorf("DaysPerTick %d (year %d ticks): %s Total after one tick = %d, want 3 * 10 = 30", daysPerTick, year, id, got)
			}
		}
	}
}

// TestFlow_GrantsBeforeLevies verifies that the grants run before the levies
// whatever the flow IDs, so a levy sees what the tick's grants gave, and that a
// grant is in the account before the tick's trade. The levy "a" sorts before the
// grant "z"; it still collects exactly the 10 the grant gave, so nothing is short.
//
// Requirement: FR-MON-018, FR-MON-019
func TestFlow_GrantsBeforeLevies(t *testing.T) {
	t.Parallel()
	levy := flowTo("a", economysim.FlowLevy, "farmer")
	levy.Amount = 10
	grant := flowTo("z", economysim.FlowGrant, "farmer")
	grant.Amount = 10
	rec := &recorder{}
	w := moneyWorld(t, moneyWorldConfig(10, 10, levy, grant), economysim.WithObserver(rec))
	rec.forget()
	if err := w.Step(); err != nil {
		t.Fatalf("Step error = %v, want nil", err)
	}
	for _, id := range []string{"a", "z"} {
		if got := flowRow(t, w, id).Total; got != 100 {
			t.Errorf("flow %q Total = %d, want 10 * 10 = 100", id, got)
		}
	}
	for _, e := range rec.events {
		if l, ok := e.(economysim.MoneyFlowLimited); ok {
			t.Errorf("MoneyFlowLimited %+v emitted, want none: the levy found the grant's money", l)
		}
	}
}

// TestFlow_ShortHeadroomLimitCounts verifies the three counts of MoneyFlowLimited
// for a levy that finds too little and a grant that hits its limit, that nothing
// is emitted when everything is applied, and that a household is in the first
// count that cut it only.
//
// Requirement: FR-MON-020, FR-MON-022, FR-MON-030
func TestFlow_ShortHeadroomLimitCounts(t *testing.T) {
	t.Parallel()
	// Each farmer holds 30. The levy asks 100 each: 30 each is taken (Short 10
	// households), Due 1 000, Applied 300. The grant to the bakers asks 10 each
	// (100) against a limit of 55: the first five households get 10, the sixth 5,
	// the rest nothing: Limit counts 5 households (the sixth and four more), the
	// flow ends with the limit.
	levy := flowTo("tax", economysim.FlowLevy, "farmer")
	levy.Amount = 100
	grant := flowTo("aid", economysim.FlowGrant, "baker")
	grant.Amount, grant.Limit = 10, 55
	c := moneyWorldConfig(10, 10, levy, grant)
	c.Roles[0].StartMoney, c.Roles[1].StartMoney = 30, 0
	rec := &recorder{}
	w := moneyWorld(t, c, economysim.WithObserver(rec))
	rec.forget()
	if err := w.Step(); err != nil {
		t.Fatalf("Step error = %v, want nil", err)
	}
	var got []economysim.Event
	for _, e := range rec.events {
		switch e.(type) {
		case economysim.MoneyFlowLimited, economysim.MoneyFlowEnded:
			got = append(got, e)
		}
	}
	want := []economysim.Event{
		economysim.MoneyFlowLimited{Tick: 0, Flow: "aid", Due: 100, Applied: 55, Limit: 5},
		economysim.MoneyFlowEnded{Tick: 0, Flow: "aid", Cause: economysim.CauseLimit, Total: 55},
		economysim.MoneyFlowLimited{Tick: 0, Flow: "tax", Due: 1000, Applied: 300, Short: 10},
	}
	if !slices.Equal(got, want) {
		t.Errorf("limited and ended events = %+v, want %+v (per flow in ID order, the end after its flow's limited rows)", got, want)
	}
	if err := w.Step(); err != nil {
		t.Fatalf("second Step error = %v, want nil", err)
	}
	// The ended flow applies nothing from now on; the levy found 0 above the
	// exemption and so is still short, with a due of 100 each.
	if got := flowRow(t, w, "aid"); got.Total != 55 || !got.Ended {
		t.Errorf("aid row after the second tick = %+v, want Total 55 and Ended", got)
	}
}

// TestInvariants_SectoralIdentityPerTick verifies, tick by tick over a run with a
// grant and a levy, that the money held by the households changes by exactly the
// grants applied less the levies applied, and that a grant to households changes
// nothing else.
//
// Requirement: FR-MON-026, FR-MON-046, FR-MON-001
func TestInvariants_SectoralIdentityPerTick(t *testing.T) {
	t.Parallel()
	grant := flowTo("g", economysim.FlowGrant, "farmer")
	grant.Amount = 20
	levy := flowTo("l", economysim.FlowLevy, "baker")
	levy.RateBP, levy.Exempt = 2_000, 5
	c := moneyWorldConfig(25, 12, grant, levy)
	c.Roles[0].StartMoney, c.Roles[1].StartMoney = 50, 400
	w := moneyWorld(t, c)
	prev, prevG, prevL := roleMoney(t, w), economysim.Money(0), economysim.Money(0)
	for tick := range 40 {
		if err := w.Step(); err != nil {
			t.Fatalf("seed 1: Step at tick %d error = %v, want nil", tick, err)
		}
		now, g, l := roleMoney(t, w), flowRow(t, w, "g").Total, flowRow(t, w, "l").Total
		if got, want := now-prev, (g-prevG)-(l-prevL); got != want {
			t.Fatalf("seed 1: tick %d: household money changed by %d, want grants - levies = %d", tick, got, want)
		}
		prev, prevG, prevL = now, g, l
	}
	if prevL == 0 {
		t.Error("the levy collected nothing in 40 ticks, want some: the test would prove nothing")
	}
}

// TestSnapshot_FlowRowsAndBoundaryOrder verifies the snapshot rows of the module:
// one FlowRow per flow in ID order with its total, scale and end, and the boundary
// rows of the flows between the built-in rows and the declared ones, in flow-ID
// order, with the kind of each.
//
// Requirement: FR-MON-042, FR-MON-003, FR-MON-029
func TestSnapshot_FlowRowsAndBoundaryOrder(t *testing.T) {
	t.Parallel()
	grant := flowTo("zz", economysim.FlowGrant, "farmer")
	grant.Amount = 1
	levy := flowTo("aa", economysim.FlowLevy, "farmer")
	levy.Amount = 1
	c := moneyWorldConfig(10, 10, grant, levy)
	c.Roles[0].StartMoney = 10
	w := moneyWorld(t, c)
	if err := w.Step(); err != nil {
		t.Fatalf("Step error = %v, want nil", err)
	}
	s, err := w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot error = %v, want nil", err)
	}
	wantFlows := []economysim.FlowRow{{Flow: "aa", Total: 10, ScaleBP: 10_000}, {Flow: "zz", Total: 10, ScaleBP: 10_000}}
	if !slices.Equal(s.Flows, wantFlows) {
		t.Errorf("Snapshot.Flows = %+v, want %+v", s.Flows, wantFlows)
	}
	var seen []economysim.BoundaryRow
	for _, b := range s.Boundary {
		if b.Reason == "aa" || b.Reason == "zz" {
			seen = append(seen, b)
		}
	}
	wantRows := []economysim.BoundaryRow{
		{Reason: "aa", Kind: economysim.KindSink, Commodity: economysim.NoCommodity, Balance: 10},
		{Reason: "zz", Kind: economysim.KindFaucet, Commodity: economysim.NoCommodity, Balance: -10},
	}
	if !slices.Equal(seen, wantRows) {
		t.Errorf("flow boundary rows = %+v, want %+v", seen, wantRows)
	}
	if first := slices.IndexFunc(s.Boundary, func(b economysim.BoundaryRow) bool { return b.Reason == "aa" }); first != len(s.Boundary)-2 {
		t.Errorf("flow rows start at row %d of %d, want the last two: after the built-in rows, and no declared module is on", first, len(s.Boundary))
	}
	off := moneyWorldConfig(10, 10)
	off.Modules.MoneySupply = nil
	if so, err := moneyWorld(t, off).Snapshot(); err != nil || len(so.Flows) != 0 {
		t.Errorf("Snapshot.Flows with the module off = %+v, %v, want none", so.Flows, err)
	}
}

// TestLogs_MoneyFlowDebugOnly verifies that the step logs nothing at Info, Warn
// or Error and that its Debug lines are the constant messages with the keys
// flow, region, due and applied.
//
// Requirement: FR-MON-032
func TestLogs_MoneyFlowDebugOnly(t *testing.T) {
	t.Parallel()
	levy := flowTo("tax", economysim.FlowLevy, "farmer")
	levy.Amount = 100
	c := moneyWorldConfig(10, 10, levy)
	c.Roles[0].StartMoney = 30
	h := newCaptureHandler()
	w := moneyWorld(t, c, economysim.WithLogger(slog.New(h)))
	h.forget()
	if err := w.Step(); err != nil {
		t.Fatalf("Step error = %v, want nil", err)
	}
	var limited int
	for _, l := range h.records() {
		if l.level > slog.LevelDebug {
			t.Errorf("line %q at level %v, want Debug only from the step", l.msg, l.level)
		}
		if l.msg != "money flow limited" {
			continue
		}
		limited++
		for _, key := range []string{"flow", "region", "due", "applied"} {
			if _, ok := l.attr(key); !ok {
				t.Errorf("money flow limited line has no %q attribute: %+v", key, l.attrs)
			}
		}
	}
	if limited != 1 {
		t.Errorf("%d money flow limited lines, want 1", limited)
	}
}

// TestInvariants_ClosedEconomyMoneyConstant verifies that a world with the module
// on and no active flow (a flow whose window starts after the run, and the module
// with no flow at all) holds the same money as the world with the module off at
// every tick, and the same households, for seeds 1 and 2 over pack one: the
// module changes the money only through a flow that runs, and its presence moves
// nothing.
//
// Requirement: FR-MON-001, FR-MON-028
func TestInvariants_ClosedEconomyMoneyConstant(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end: 3 x 150 ticks of pack one per seed; skipped under -short")
	}
	off := loadValidPack(t, packOne...)
	idle := loadValidPack(t, packOneFaucetSunk...)
	idle.Modules.MoneySupply.Flows[0].Start, idle.Modules.MoneySupply.Flows[1].Start = 1_000, 1_000
	empty := loadValidPack(t, packOne...)
	empty.Modules.MoneySupply = &economysim.MoneySupply{}
	for _, seed := range seedList(min(2, scenarioSeedCount())) {
		for name, cfg := range map[string]economysim.Config{"idle flows": idle, "no flows": empty} {
			w, err := economysim.NewWorld(cfg, economysim.WithSeed(seed), economysim.WithAudit())
			if err != nil {
				t.Fatalf("seed %d: NewWorld(%s) error = %v, want nil", seed, name, err)
			}
			r2, err := economysim.NewWorld(off, economysim.WithSeed(seed))
			if err != nil {
				t.Fatalf("seed %d: NewWorld(off) error = %v, want nil", seed, err)
			}
			for tick := range 150 {
				if err := w.Step(); err != nil {
					t.Fatalf("seed %d, %s: Step at tick %d error = %v, want nil", seed, name, tick, err)
				}
				if err := r2.Step(); err != nil {
					t.Fatalf("seed %d: Step of the off world at tick %d error = %v, want nil", seed, tick, err)
				}
				a, _ := w.Snapshot()
				b, _ := r2.Snapshot()
				if a.Money != b.Money || a.Households != b.Households {
					t.Fatalf("seed %d, %s: tick %d: money %d and households %d against %d and %d with the module off", seed, name, tick, a.Money, a.Households, b.Money, b.Households)
				}
			}
		}
	}
}

// BenchmarkP17Flows reports the time and the allocations of P1.7 over four flows
// (a rate grant, a top-up, a fixed levy and a rate levy) on the census of pack one,
// 600 households in six roles, in steady state: the cost is the flows times the
// accounts of their classes, which this reports for the first consumer's size
// (NFR-MON-002). It reports and asserts nothing (testing.md 8.2).
//
// Requirement: NFR-MON-002
func BenchmarkP17Flows(b *testing.B) {
	cfg := loadValidPack(b, packOne...)
	to := func(role string) economysim.FlowTarget {
		return economysim.FlowTarget{Kind: economysim.TargetHouseholds, Role: role}
	}
	cfg.Modules.MoneySupply = &economysim.MoneySupply{Flows: []economysim.MoneyFlow{
		{ID: "a", Kind: economysim.FlowGrant, Target: to("grower"), Amount: 20},
		{ID: "b", Kind: economysim.FlowGrant, Target: to("cook"), Level: 5_000},
		{ID: "c", Kind: economysim.FlowLevy, Target: to("forester"), Amount: 5},
		{ID: "d", Kind: economysim.FlowLevy, Target: to("maker"), RateBP: 3_650, Per: economysim.PerYear, Exempt: 3_000},
	}}
	w, err := economysim.NewWorld(cfg, economysim.WithSeed(1))
	if err != nil {
		b.Fatalf("NewWorld error = %v, want nil", err)
	}
	for range 3 {
		economysim.P17Flows(w)
	}
	b.ReportAllocs()
	for b.Loop() {
		economysim.P17Flows(w)
	}
}
