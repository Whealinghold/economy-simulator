package economysim

import (
	"math/rand/v2"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/ledger"
	"github.com/Whealinghold/economy-sim/internal/welfare"
)

// This file tests the step of the population module (population.go, plan 2-11
// Phase 4) on the small world of exitConfig: agents 1 and 2 are millers of
// weight 2 and 1, agent 3 is the closed weaver of weight 1, and the pool of the
// region holds 25 000. The upkeep is one grain at the published price 1 000. The
// default schedules give, at the welfare ratio 14 000 that setRatio fixes, a
// birth rate of 250 and a death rate of 450 + trunc(-400 * 6 000 / 8 000) = 150.

// popSeed is the seed of these tests; the failure messages name it (testing.md 6.2).
const popSeed = 20261108

// popWorld builds the world of exitConfig with the module on at mod and the
// whole pool as the entry budget, the pool seeded at 25 000, and every agent at
// the welfare ratio 14 000. The first view is taken once so the carry slice has
// grown; the caller sets the carries it wants.
func popWorld(t *testing.T, mod Population, opts ...Option) *World {
	t.Helper()
	c := exitConfig()
	c.World.EntryPoolShare = 10_000
	c.Modules.Population = &mod
	w := newMill(t, c, append([]Option{WithSeed(popSeed)}, opts...)...)
	if err := w.estates.seedPool(w.ledger, 0, 25_000); err != nil {
		t.Fatalf("seed %d: seedPool() error = %v, want nil", popSeed, err)
	}
	for id := 1; id <= 3; id++ {
		setRatio(w, AgentID(id), 14_000)
	}
	popView(w, 0)
	return w
}

// popView builds the view of region g and runs the step on it, as decideRegion
// does; buildRegionView alone is what the snapshot reads and moves no carry.
func popView(w *World, g RegionID) *RegionView {
	v := w.buildRegionView(g)
	w.popFill(g, v)
	return v
}

// setRatio fills the welfare ring of agent id so that its ratio is exactly
// ratio: a full window of records with income ratio and cost 10 000.
func setRatio(w *World, id AgentID, ratio int64) {
	a := &w.agents[id]
	for range w.roles.row(a.role).window {
		a.history.Push(welfare.Record{Income: ratio, Cost: 10_000})
	}
}

// setCarries sets the carries a test wants: the death carry of each agent and the
// birth carry of each role of region 0.
func setCarries(w *World, death map[AgentID]int64, birth map[RoleID]int64) {
	for id, c := range death {
		w.pop.deathCarry[id] = c
	}
	for r, c := range birth {
		w.pop.birthCarry[int(r)] = c
	}
}

// TestStep_CarryAndHooks verifies the step of one region: the death carry of each
// agent and the birth carry of each role advance by exposed households times the
// rate, a whole household leaves the remainder in the carry, the hooks of the view
// hold the counts after the floor, and a closed role's birth carry does not
// advance while its deaths run.
//
// Arithmetic, D = 10 000 * TicksPerYear: agent 1 (weight 2) adds 2 * 150 = 300 to
// a carry of D - 100, which is D + 200: one death, carry 200. Agent 2 adds 150 to
// 0. The closed weaver (agent 3) adds 150 to D - 150: one death due, and the
// group floor of 1 household holds it (DeathsHeld 1). The miller role has 3
// exposed households at 250: 750, added to D - 700, is D + 50: one birth, carry
// 50. The weaver's birth carry stays 5.
//
// Requirement: FR-POP-002, FR-POP-007, FR-POP-008, FR-POP-009, FR-POP-011, FR-POP-013, FR-POP-023
func TestStep_CarryAndHooks(t *testing.T) {
	t.Parallel()
	w := popWorld(t, DefaultPopulation())
	d := w.pop.denom
	setCarries(w, map[AgentID]int64{1: d - 100, 2: 0, 3: d - 150}, map[RoleID]int64{0: d - 700, 1: 5})
	v := popView(w, 0)
	if got := []int64{v.Agents[0].PopDeaths, v.Agents[1].PopDeaths, v.Agents[2].PopDeaths}; !slices.Equal(got, []int64{1, 0, 0}) {
		t.Errorf("seed %d: PopDeaths = %v, want [1 0 0]", popSeed, got)
	}
	if got := []int64{v.Roles[0].PopBirths, v.Roles[1].PopBirths}; !slices.Equal(got, []int64{1, 0}) {
		t.Errorf("seed %d: PopBirths = %v, want [1 0]", popSeed, got)
	}
	if got := w.pop.deathCarry[1:4]; !slices.Equal(got, []int64{200, 150, 0}) {
		t.Errorf("seed %d: death carries = %v, want [200 150 0]", popSeed, got)
	}
	if got := w.pop.birthCarry[:2]; !slices.Equal(got, []int64{50, 5}) {
		t.Errorf("seed %d: birth carries = %v, want [50 5]", popSeed, got)
	}
	want := []popCount{
		{exposed: 3, birthsDue: 1, deathsDue: 1},
		{exposed: 1, deathsDue: 1, deathsHeld: 1},
	}
	if !slices.Equal(w.pop.cnt, want) {
		t.Errorf("seed %d: counts = %+v, want %+v", popSeed, w.pop.cnt, want)
	}
}

// TestStep_Exposure verifies who is exposed (spec-population.md 3.7): an unfed
// household is not, an external agent counts nowhere and keeps its carry, and an
// agent with no ratio is counted as NoRatio and keeps its carry.
//
// Arithmetic: agent 1 has weight 2 and one unfed household, so it exposes 1 and
// adds 1 * 150 to its carry. With every price at 0 the cost of the upkeep is 0 and
// the rings hold only empty records, so they have no ratio: all four households are NoRatio.
//
// Requirement: FR-POP-003, FR-POP-006, FR-POP-023
func TestStep_Exposure(t *testing.T) {
	t.Parallel()
	w := popWorld(t, DefaultPopulation())
	w.agents[1].unfed, w.agents[3].external = 1, true
	start := w.pop.carryOf(1)
	setCarries(w, map[AgentID]int64{1: start}, nil)
	external := w.pop.deathCarry[3]
	popView(w, 0)
	if got := w.pop.deathCarry[1] - start; got != 150 {
		t.Errorf("seed %d: carry of the agent with an unfed household moved by %d, want 150", popSeed, got)
	}
	if w.pop.deathCarry[3] != external || w.pop.cnt[1] != (popCount{}) {
		t.Errorf("seed %d: external agent: carry %d (was %d), counts %+v, want it unexposed and uncounted", popSeed, w.pop.deathCarry[3], external, w.pop.cnt[1])
	}
	if c := w.pop.cnt[0]; c.exposed != 2 || c.unfed != 1 {
		t.Errorf("seed %d: miller counts = %+v, want exposed 2 (1 + 1) and unfed 1", popSeed, c)
	}

	w = popWorld(t, DefaultPopulation())
	for i := range w.markets.reports {
		w.markets.reports[i].Price = 0
	}
	for id := 1; id <= 3; id++ {
		for range w.roles.row(w.agents[id].role).window {
			w.agents[id].history.Push(welfare.Record{})
		}
	}
	before := slices.Clone(w.pop.deathCarry)
	popView(w, 0)
	if c := w.pop.cnt; c[0].noRatio != 3 || c[1].noRatio != 1 || c[0].exposed != 0 || !slices.Equal(before, w.pop.deathCarry) {
		t.Errorf("seed %d: no-ratio counts = %+v, carries %v (was %v), want 3 and 1 households with no ratio and no carry moved", popSeed, c, w.pop.deathCarry, before)
	}
}

// TestStep_FloorCapAndExtinct verifies the cuts in the order of D430: the group
// floor holds deaths in agent ID order and drops them, and the household cap of
// the region drops the births that would exceed it.
//
// Arithmetic: the millers hold 3 households and MinHouseholds is 2. Agent 1 takes
// 1 (3 -> 2), agent 2's death would leave 1, so it is held (DeathsHeld 1). The
// region then holds 4 - 1 = 3 households; with a cap of 3 the one birth due is
// dropped (BirthsCapped 1).
//
// Requirement: FR-POP-013, FR-POP-014, FR-POP-015, FR-POP-038
func TestStep_FloorCapAndExtinct(t *testing.T) {
	t.Parallel()
	mod := DefaultPopulation()
	mod.MinHouseholds = 2
	mod.Regions = []RegionPopulation{{Region: "home", BirthScale: 10_000, DeathScale: 10_000, HouseholdCap: 3}}
	w := popWorld(t, mod)
	d := w.pop.denom
	setCarries(w, map[AgentID]int64{1: d - 300, 2: d - 150, 3: 0}, map[RoleID]int64{0: d - 1})
	v := popView(w, 0)
	if v.Agents[0].PopDeaths != 1 || v.Agents[1].PopDeaths != 0 || v.Roles[0].PopBirths != 0 {
		t.Errorf("seed %d: PopDeaths = %d, %d and PopBirths = %d, want 1, 0 and 0", popSeed, v.Agents[0].PopDeaths, v.Agents[1].PopDeaths, v.Roles[0].PopBirths)
	}
	if c := w.pop.cnt[0]; c.deathsHeld != 1 || c.birthsCapped != 1 || c.birthsDue != 1 || c.deathsDue != 2 {
		t.Errorf("seed %d: counts = %+v, want deathsDue 2, deathsHeld 1, birthsDue 1, birthsCapped 1", popSeed, c)
	}

	// A region with no households gets no birth: no agent, no numerator.
	w = popWorld(t, DefaultPopulation())
	for id := 1; id <= 3; id++ {
		w.agents[id].weight = 0
	}
	if v := popView(w, 0); v.Roles[0].PopBirths != 0 || v.Roles[1].PopBirths != 0 {
		t.Errorf("seed %d: PopBirths = %d, %d in a region with no households, want 0, 0", popSeed, v.Roles[0].PopBirths, v.Roles[1].PopBirths)
	}
}

// TestScale_EffectiveReadEachTick verifies that the scales of a region are read
// each tick and that setPopulationScale returns the value it replaced and refuses
// what it cannot apply.
//
// Arithmetic: at the ratio 9 000 the death rate is 450 + trunc(-400 * 1 000 /
// 8 000) = 400, so agent 1 (2 exposed households) adds 800. At the scale 30 000
// the rate is min(10 000, trunc(400 * 30 000 / 10 000)) = 1 200 and it adds 2 400;
// cleared, 800 again.
//
// Requirement: FR-POP-005, FR-POP-039
func TestScale_EffectiveReadEachTick(t *testing.T) {
	t.Parallel()
	w := popWorld(t, DefaultPopulation())
	setRatio(w, 1, 9_000)
	moved := func() int64 {
		setCarries(w, map[AgentID]int64{1: 0}, nil)
		popView(w, 0)
		return w.pop.deathCarry[1]
	}
	if got := moved(); got != 800 {
		t.Fatalf("seed %d: carry moved by %d at the row's scale, want 800", popSeed, got)
	}
	if before, ok := w.setPopulationScale(0, ScaleDeath, 30_000); !ok || before != 10_000 {
		t.Fatalf("seed %d: setPopulationScale(0, death, 30000) = %d, %v, want 10000, true", popSeed, before, ok)
	}
	if got := moved(); got != 2_400 {
		t.Errorf("seed %d: carry moved by %d at the scale 30 000, want 2400", popSeed, got)
	}
	w.setPopulationScale(0, ScaleDeath, w.pop.rowDeath[0])
	if got := moved(); got != 800 {
		t.Errorf("seed %d: carry moved by %d after the scale was cleared, want 800", popSeed, got)
	}
	for name, call := range map[string]func() bool{
		"unknown region": func() bool { _, ok := w.setPopulationScale(5, ScaleBirth, 1); return ok },
		"negative":       func() bool { _, ok := w.setPopulationScale(0, ScaleBirth, -1); return ok },
		"over the bound": func() bool { _, ok := w.setPopulationScale(0, ScaleBirth, MaxBasisPoints+1); return ok },
		"unknown kind":   func() bool { _, ok := w.setPopulationScale(0, "x", 1); return ok },
	} {
		if call() {
			t.Errorf("seed %d: setPopulationScale(%s) = true, want false", popSeed, name)
		}
	}
}

// TestExec_DeathAndBirth runs the entry step with the default spawner and checks
// the executed plan: a partial death sheds half the money by the reason deaths, a
// whole removal is an AgentBankrupted with the cause population, and a birth is
// paid the stake from the pool into the host of smallest weight.
//
// Arithmetic: agent 1 (weight 2, 14 000) keeps 7 000 and sends 7 000 to the estate;
// agent 2 (weight 1, 7 000) is removed and sends 7 000. The pool is 25 000 + 14 000
// = 39 000 when the budget is taken (EntryPoolShare 10 000). The stake is trunc(7
// 000 * 5 000 / 10 000) = 3 500, so the pool ends at 39 000 - 3 500 = 35 500 and
// agent 1 returns to weight 2 with 7 000 + 3 500.
//
// Requirement: FR-POP-016, FR-POP-017, FR-POP-018, FR-POP-019, FR-POP-021, FR-POP-022, FR-SCL-014
func TestExec_DeathAndBirth(t *testing.T) {
	t.Parallel()
	rec := &evLog{}
	mod := DefaultPopulation()
	mod.BirthStake = 5_000
	w := popWorld(t, mod, WithObserver(rec))
	d := w.pop.denom
	agents0, hh0 := w.live.agents, w.live.households
	setCarries(w, map[AgentID]int64{1: d - 150, 2: d - 100, 3: 0}, map[RoleID]int64{0: d - 1})
	runEntry(t, w, rec)
	var causes []Cause
	for _, e := range rec.events {
		switch e := e.(type) {
		case WeightChanged:
			causes = append(causes, e.Cause)
			if e.Cause == CausePopulation && (e.Agent != 1 || e.From != 2 || e.To != 1 || e.Money != -7_000 || e.Units < 1) {
				t.Errorf("seed %d: death event = %+v, want agent 1, 2 -> 1, money -7000 and units moved", popSeed, e)
			}
			if e.Cause == CauseBirths && (e.Agent != 1 || e.From != 1 || e.To != 2 || e.Money != 3_500) {
				t.Errorf("seed %d: birth event = %+v, want agent 1, 1 -> 2, money 3500", popSeed, e)
			}
		case AgentBankrupted:
			causes = append(causes, e.Cause)
			if e.Agent != 2 || e.Cause != CausePopulation || e.Weight != 1 {
				t.Errorf("seed %d: removal event = %+v, want agent 2 weight 1 by population", popSeed, e)
			}
		}
	}
	if want := []Cause{CausePopulation, CausePopulation, CauseBirths}; !slices.Equal(causes, want) {
		t.Errorf("seed %d: causes in order = %v, want %v (deaths, then births)", popSeed, causes, want)
	}
	if got := poolOf(w, 0); got != 35_500 {
		t.Errorf("seed %d: pool = %d, want 35500", popSeed, got)
	}
	if c := w.pop.cnt[0]; c.deathsDone != 2 || c.birthsDone != 1 || c.birthsUnfunded != 0 {
		t.Errorf("seed %d: counts = %+v, want deathsDone 2, birthsDone 1, birthsUnfunded 0", popSeed, c)
	}
	if err := w.ledger.Audit(); err != nil {
		t.Errorf("seed %d: Audit() = %v, want nil: money and goods are conserved", popSeed, err)
	}
	checkCounts(t, "death and birth", w, rec, agents0, hh0)
}

// TestExec_BirthUnfundedAndNewAgent verifies that a birth the pool cannot pay is
// neither created nor deferred, that a stake of 0 is not limited by the pool, and
// that with no host below the scale the household becomes a new agent of weight 1
// paid by the reason births.
//
// Arithmetic: a pool of 3 000 against a stake of 7 000 funds floor(3 000 / 7 000)
// = 0 households. With the stake 0 the budget does not limit; both millers are
// at the group scale 2, so the household becomes a new agent.
//
// Requirement: FR-POP-017, FR-POP-021
func TestExec_BirthUnfundedAndNewAgent(t *testing.T) {
	t.Parallel()
	rec := &evLog{}
	w := popWorld(t, DefaultPopulation(), WithObserver(rec))
	if err := w.ledger.Transfer(w.estates.accounts[0], w.agents[3].account, int64(poolOf(w, 0))-3_000, ledger.StartMoney); err != nil {
		t.Fatalf("seed %d: Transfer() error = %v, want nil", popSeed, err)
	}
	setCarries(w, nil, map[RoleID]int64{0: w.pop.denom - 1})
	runEntry(t, w, rec)
	if c := w.pop.cnt[0]; c.birthsDone != 0 || c.birthsUnfunded != 1 || len(of[AgentSpawned](rec)) != 0 {
		t.Errorf("seed %d: counts = %+v, spawned = %d, want no birth and birthsUnfunded 1", popSeed, c, len(of[AgentSpawned](rec)))
	}
	if lim := of[EntryLimited](rec); len(lim) != 1 || lim[0].Unfunded != 1 {
		t.Errorf("seed %d: EntryLimited = %+v, want one with Unfunded 1 (births count there too)", popSeed, lim)
	}

	// A stake of 0 is not limited by the pool, and a full host takes a new agent.
	mod := DefaultPopulation()
	mod.BirthStake = 0
	rec2 := &evLog{}
	w = popWorld(t, mod, WithObserver(rec2))
	w.agents[2].weight, w.agents[1].weight = 2, 2
	setCarries(w, nil, map[RoleID]int64{0: w.pop.denom - 1})
	runEntry(t, w, rec2)
	sp := of[AgentSpawned](rec2)
	if len(sp) != 1 || sp[0].Weight != 1 || sp[0].StartMoney != 0 {
		t.Errorf("seed %d: AgentSpawned = %+v, want one new agent of weight 1 with no money", popSeed, sp)
	}
}

// TestExec_PopulationCauseRefusedWhenOff verifies that a reduction with the cause
// population is refused with the module off and counted in Refused (N-290).
//
// Requirement: FR-POP-001, FR-POP-016
func TestExec_PopulationCauseRefusedWhenOff(t *testing.T) {
	t.Parallel()
	sp := &scriptSpawner{plan: regionPlans([]Decision{{Kind: DecideReduce, Agent: 1, Households: 1, Cause: CausePopulation}})}
	w, rec, _ := execWorld(t, execConfig(), 0, sp)
	runEntry(t, w, rec)
	if w.agents[1].weight != 2 {
		t.Errorf("seed %d: weight of agent 1 = %d, want 2: the reduction must be refused", execSeed, w.agents[1].weight)
	}
	if lim := of[EntryLimited](rec); len(lim) != 1 || lim[0].Refused != 1 {
		t.Errorf("seed %d: EntryLimited = %+v, want one with Refused 1", execSeed, lim)
	}
}

// TestPopulation_OffLeavesNoTrace verifies that with the module off no state
// exists, Renewal keeps its meaning and the ledger has no deaths reason.
//
// Requirement: FR-POP-001
func TestPopulation_OffLeavesNoTrace(t *testing.T) {
	t.Parallel()
	w := newMill(t, exitConfig(), WithSeed(popSeed))
	if w.pop != nil || w.renewalBP() != w.cfg.World.Renewal || w.popCause(CausePopulation) {
		t.Errorf("seed %d: pop = %v, renewal = %d, population cause accepted = %v, want no state, the configured Renewal and the cause refused",
			popSeed, w.pop, w.renewalBP(), w.popCause(CausePopulation))
	}
	if v := w.buildRegionView(0); v.Renewal != 10_000 || v.Roles[0].PopBirths != 0 || v.Agents[0].PopDeaths != 0 {
		t.Errorf("seed %d: view Renewal = %d, hooks = %d, %d, want 10000, 0, 0", popSeed, v.Renewal, v.Roles[0].PopBirths, v.Agents[0].PopDeaths)
	}
	if err := w.ledger.Transfer(w.agents[1].account, w.estates.accounts[0], 1, ledger.Deaths); err == nil {
		t.Errorf("seed %d: Transfer(deaths) with the module off succeeded, want ErrReasonMismatch or a refusal", popSeed)
	}
}

// TestRenewal_ZeroWhileOn verifies that with the module on the effective Renewal
// is 0 in the view whatever is configured.
//
// Requirement: FR-POP-020
func TestRenewal_ZeroWhileOn(t *testing.T) {
	t.Parallel()
	w := popWorld(t, DefaultPopulation())
	if v := popView(w, 0); v.Renewal != 0 || w.renewalBP() != 0 || w.cfg.World.Renewal != 10_000 {
		t.Errorf("seed %d: view Renewal = %d, renewalBP = %d, configured %d, want 0, 0 and the configured 10000", popSeed, v.Renewal, w.renewalBP(), w.cfg.World.Renewal)
	}
}

// TestPopulation_DrawsNothing verifies that the step takes no word from the
// random source: the PCG state is the same before and after it.
//
// Requirement: FR-POP-036, NFR-POP-004
func TestPopulation_DrawsNothing(t *testing.T) {
	t.Parallel()
	w := popWorld(t, DefaultPopulation())
	before, err := w.rnd.pcg.MarshalBinary()
	if err != nil {
		t.Fatalf("seed %d: MarshalBinary() error = %v, want nil", popSeed, err)
	}
	for range 3 {
		popView(w, 0)
	}
	after, _ := w.rnd.pcg.MarshalBinary()
	if !slices.Equal(before, after) {
		t.Errorf("seed %d: the PCG state moved during the step, want it unchanged", popSeed)
	}
}

// TestPopulation_NoAllocation verifies that the step allocates nothing once the
// carry slice has grown, with the default observer and logger.
//
// Requirement: NFR-POP-001
func TestPopulation_NoAllocation(t *testing.T) {
	w := popWorld(t, DefaultPopulation())
	v := popView(w, 0)
	if n := testing.AllocsPerRun(50, func() { w.popFill(0, v) }); n != 0 {
		t.Errorf("seed %d: popFill allocated %v times per run, want 0", popSeed, n)
	}
}

// BenchmarkPopulation_Fill reports the cost of the step over a region of 1 000
// agents of weight 1; the cost is proportional to the agents, not to the weights.
//
// Requirement: NFR-POP-002
func BenchmarkPopulation_Fill(b *testing.B) {
	c := exitConfig()
	c.Groups[0].Households, c.Groups[0].Scale = 1_000, 1
	mod := DefaultPopulation()
	c.Modules.Population = &mod
	w, err := NewWorld(c, WithSeed(popSeed))
	if err != nil {
		b.Fatalf("NewWorld() error = %v, want nil", err)
	}
	v := popView(w, 0)
	b.ReportAllocs()
	for b.Loop() {
		w.popFill(0, v)
	}
}

// TestStep_SignalIsOwnRatio verifies that two agents of one role with different
// rings get different rates, and that no rule of the module file reads the mean
// ratio of the snapshot.
//
// Arithmetic: agent 1 at the ratio 14 000 has the death rate 150 and adds 2 * 150
// = 300; agent 2 at 8 000 has 450 and adds 450.
//
// Requirement: FR-POP-003, FR-POP-040
func TestStep_SignalIsOwnRatio(t *testing.T) {
	t.Parallel()
	w := popWorld(t, DefaultPopulation())
	setRatio(w, 2, 8_000)
	a1, a2 := w.pop.carryOf(1), w.pop.carryOf(2)
	setCarries(w, map[AgentID]int64{1: a1, 2: a2}, nil)
	popView(w, 0)
	if d1, d2 := w.pop.deathCarry[1]-a1, w.pop.deathCarry[2]-a2; d1 != 300 || d2 != 450 {
		t.Errorf("seed %d: carries moved by %d and %d, want 300 and 450", popSeed, d1, d2)
	}
	src, err := os.ReadFile("population.go")
	if err != nil {
		t.Fatalf("ReadFile(population.go) error = %v, want nil", err)
	}
	for _, banned := range []string{"MeanRatioBP", "meanRatio", "RatioHouseholds", "HasMean"} {
		if strings.Contains(string(src), banned) {
			t.Errorf("population.go mentions %s, want no rule of the module to read the mean ratio", banned)
		}
	}
}

// TestPopulation_Conservation runs a world of 31 agents with the module on at a
// rate that makes births and deaths frequent, under the audit, for several seeds,
// and checks that money and goods stay conserved every tick (the audit fails the
// step otherwise), that the household count rebuilt from the events equals the
// table, and that both births and deaths happened.
//
// Requirement: FR-POP-022, FR-POP-037
func TestPopulation_Conservation(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	for seed := uint64(1); seed <= 4; seed++ {
		t.Run("seed "+string(rune('0'+seed)), func(t *testing.T) {
			t.Parallel()
			c := exitConfig()
			c.Groups[0].Households = 60
			c.Roles[0].StartStock[0].Quantity = 1_000 // grain for the whole run, so the millers stay fed and exposed
			mod := DefaultPopulation()
			mod.BirthSchedule = []Knot{{Ratio: 0, Rate: 10_000}}
			mod.DeathSchedule = []Knot{{Ratio: 0, Rate: 10_000}}
			c.Modules.Population = &mod
			rec := &evLog{}
			w := newMill(t, c, WithSeed(seed), WithObserver(rec), WithAudit())
			if err := w.estates.seedPool(w.ledger, 0, 500_000); err != nil {
				t.Fatalf("seed %d: seedPool() error = %v, want nil", seed, err)
			}
			rec.events = nil
			agents0, hh0 := w.live.agents, w.live.households
			step(t, w, 200)
			var births, deaths int64
			for _, e := range rec.events {
				switch e := e.(type) {
				case WeightChanged:
					if e.Cause == CauseBirths {
						births += e.To - e.From
					}
					if e.Cause == CausePopulation {
						deaths += e.From - e.To
					}
				case AgentSpawned:
					births += e.Weight
				case AgentBankrupted:
					if e.Cause == CausePopulation {
						deaths += e.Weight
					}
				}
			}
			if deaths == 0 || births == 0 {
				t.Errorf("seed %d: %d deaths and %d births in 200 ticks, want both to happen", seed, deaths, births)
			}
			checkCounts(t, "conservation", w, rec, agents0, hh0)
		})
	}
}

// recSpawner is a consumer Spawner fake that records the regions it is called for
// and the causes of the plan the default spawner appends, so a test sees the order
// of the plan (spec-population.md 4.8).
type recSpawner struct {
	regions []RegionID
	plans   [][]Cause
}

// Decide records the call and delegates to the default.
func (s *recSpawner) Decide(v *RegionView, rng *rand.Rand, d *Decisions) {
	s.regions = append(s.regions, v.Region)
	DefaultSpawner{}.Decide(v, rng, d)
	var causes []Cause
	for _, x := range d.List {
		causes = append(causes, x.Cause)
	}
	s.plans = append(s.plans, causes)
}

// TestStep_OrderAndHook verifies that the step runs once per region in ascending
// RegionID before Decide, and that the plan lists the module's deaths before its
// births (renewal being 0 there are no entries between them).
//
// Requirement: FR-POP-002, FR-POP-019
func TestStep_OrderAndHook(t *testing.T) {
	t.Parallel()
	c := removalConfig()
	c.World.EntryPoolShare = 10_000
	c.World.Spawner = "rec"
	mod := DefaultPopulation()
	c.Modules.Population = &mod
	sp := &recSpawner{}
	rec := &evLog{}
	w := newMill(t, c, WithSeed(popSeed), WithSpawner("rec", sp), WithObserver(rec))
	for id := 2; id <= 5; id++ {
		setRatio(w, AgentID(id), 14_000)
	}
	popView(w, 0)
	popView(w, 1)
	setCarries(w, map[AgentID]int64{2: w.pop.denom - 150}, map[RoleID]int64{0: w.pop.denom - 1})
	runEntry(t, w, rec)
	if want := []RegionID{0, 1}; !slices.Equal(sp.regions, want) {
		t.Errorf("seed %d: Decide called for regions %v, want %v", popSeed, sp.regions, want)
	}
	if want := []Cause{CausePopulation, CauseBirths}; len(sp.plans) < 1 || !slices.Equal(sp.plans[0], want) {
		t.Errorf("seed %d: plan of region 0 = %v, want %v", popSeed, sp.plans, want)
	}
}

// TestCanonical_PopulationCopies verifies that the canonical form of the module
// copies both schedules (a later change of the caller's slice does not reach it),
// keeps the knots in the order written, sorts the region rows by region, and
// keeps a nil module nil and an empty module on.
//
// Requirement: FR-CFG-069
func TestCanonical_PopulationCopies(t *testing.T) {
	t.Parallel()
	src := DefaultPopulation()
	src.Regions = []RegionPopulation{DefaultRegionPopulation("b"), DefaultRegionPopulation("a")}
	got := src.canonical()
	src.BirthSchedule[0].Rate = 1
	src.DeathSchedule[1].Ratio = 1
	if got.BirthSchedule[0].Rate != 250 || got.DeathSchedule[1].Ratio != 16_000 {
		t.Errorf("canonical schedules = %v, %v, want copies the caller's change does not reach", got.BirthSchedule, got.DeathSchedule)
	}
	if got.Regions[0].Region != "a" || got.Regions[1].Region != "b" || src.Regions[0].Region != "b" {
		t.Errorf("canonical rows = %v from %v, want sorted copies by region", got.Regions, src.Regions)
	}
	if !slices.Equal(got.BirthSchedule, []Knot{{8_000, 250}, {16_000, 250}}) {
		t.Errorf("canonical birth schedule = %v, want the knots in order", got.BirthSchedule)
	}
	var nilModule *Population
	if nilModule.canonical() != nil {
		t.Errorf("canonical of a nil module is not nil")
	}
	if empty := (&Population{}).canonical(); empty == nil {
		t.Errorf("canonical of an empty module is nil, want it on")
	}
}
