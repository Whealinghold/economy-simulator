package economysim

import (
	"math/rand/v2"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/beliefs"
	"github.com/Whealinghold/economy-sim/internal/ledger"
)

// agentConfig is the canonical configuration of the agent tests. Commodities
// sort to fuel 0, ore 1, staple 2 (spoils in 50 ticks, so lots have born
// ticks). The world upkeep is staple 3 and fuel 1 per household, as in
// spec-agents.md 16.3. Roles sort to clerk 0 (no recipes, trades ore), farmer 1
// (recipe grow, which has two outputs, so two residues; start money 10 000 and
// start stock ore 2, staple 5) and miner 2 (upkeep fuel 1 only). Regions sort
// to cold 0 (extra upkeep fuel 2) and warm 1.
func agentConfig() *Config {
	c := Config{
		World: WorldParams{WelfareWindow: 30, WindowCycles: 2, RestBand: 500,
			Upkeep: []Item{{Commodity: "staple", Quantity: 3}, {Commodity: "fuel", Quantity: 1}}},
		Commodities: []Commodity{{ID: "fuel", ReferencePrice: 10}, {ID: "ore", ReferencePrice: 10},
			{ID: "staple", ReferencePrice: 10, ShelfLife: 50}},
		Recipes: []Recipe{{ID: "grow", Duration: 1,
			Outputs: []Output{{Commodity: "staple", Quantity: 2, CostShare: 10000}, {Commodity: "ore", Quantity: 1}}}},
		Roles: []Role{
			{ID: "farmer", Recipes: []string{"grow"}, StartMoney: 10000, TargetRatio: 15000,
				StartStock: []Item{{Commodity: "ore", Quantity: 2}, {Commodity: "staple", Quantity: 5}}},
			{ID: "clerk", Trades: []string{"ore"}, TargetRatio: 15000},
			{ID: "miner", Upkeep: []Item{{Commodity: "fuel", Quantity: 1}}, TargetRatio: 15000},
		},
		Regions: []Region{{ID: "warm"}, {ID: "cold", ExtraUpkeep: []Item{{Commodity: "fuel", Quantity: 2}}}},
	}
	cc := c.canonical()
	return &cc
}

// agentFixture is a ledger out of construction with the tables of agentConfig.
type agentFixture struct {
	cfg   *Config
	rt    recipeTable
	roles roleTable
	l     *ledger.Ledger
}

func newAgentFixture(t *testing.T, construction bool) *agentFixture {
	t.Helper()
	cfg := agentConfig()
	l, err := newLedger(*cfg)
	if err != nil {
		t.Fatalf("newLedger: %v", err)
	}
	if !construction {
		if err := l.EndConstruction(); err != nil {
			t.Fatalf("EndConstruction: %v", err)
		}
	}
	return &agentFixture{cfg: cfg, rt: newRecipeTable(cfg), roles: newRoleTable(cfg), l: l}
}

func (f *agentFixture) role(t *testing.T, name string) (RoleID, *roleRow) {
	t.Helper()
	id, ok := f.cfg.roleID(name)
	if !ok {
		t.Fatalf("role %q not in the configuration", name)
	}
	return id, f.roles.row(id)
}

func (f *agentFixture) com(t *testing.T, name string) int {
	t.Helper()
	id, ok := f.cfg.commodityID(name)
	if !ok {
		t.Fatalf("commodity %q not in the configuration", name)
	}
	return int(id)
}

func (f *agentFixture) region(t *testing.T, name string) RegionID {
	t.Helper()
	id, ok := f.cfg.regionID(name)
	if !ok {
		t.Fatalf("region %q not in the configuration", name)
	}
	return id
}

// open opens an agent account in a region.
func (f *agentFixture) open(t *testing.T, region string) int {
	t.Helper()
	id, err := f.l.Open(ledger.KindAgent, 0, int(f.region(t, region)))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	return id
}

// give puts n units of a commodity in an account from the Initial boundary.
func (f *agentFixture) give(t *testing.T, account int, name string, n int64) {
	t.Helper()
	c := f.com(t, name)
	from, ok := f.l.Boundary(ledger.Initial, c)
	if !ok {
		t.Fatalf("no Initial boundary for %s", name)
	}
	if err := f.l.Move(from, account, c, n, ledger.Initial); err != nil {
		t.Fatalf("give %d %s: %v", n, name, err)
	}
}

func (f *agentFixture) held(t *testing.T, account int, name string) int64 {
	t.Helper()
	h, _, err := f.l.Holding(account, f.com(t, name))
	if err != nil {
		t.Fatalf("Holding(%s): %v", name, err)
	}
	return h
}

func (f *agentFixture) newAgent(t *testing.T, role string, account int, weight int64) agent {
	t.Helper()
	id, row := f.role(t, role)
	return newAgent(agentInit{id: 1, account: account, role: id, weight: weight}, row, &f.rt)
}

// TestAgent_HoldsNoMoneyQuantityOrRegion verifies that the agent struct has no
// field of a money, price or quantity type and none named for a balance, a
// stock, money or a region, so that a test that sums the agent's fields cannot
// find money and the region cannot be stored twice.
//
// Requirement: FR-AGT-002
//
// Guard: the ledger account is the only home of money and goods and of the
// region; a field added for convenience would be a second balance that drifts
// from the account, and the audit would not see it.
func TestAgent_HoldsNoMoneyQuantityOrRegion(t *testing.T) {
	t.Parallel()
	typ := reflect.TypeFor[agent]()
	banned := []reflect.Type{reflect.TypeFor[Money](), reflect.TypeFor[Price](), reflect.TypeFor[Quantity](), reflect.TypeFor[RegionID]()}
	words := []string{"money", "balance", "stock", "quantity", "qty", "holding", "region", "price"}
	for i := range typ.NumField() {
		f := typ.Field(i)
		t.Run(f.Name, func(t *testing.T) {
			t.Parallel()
			if slices.Contains(banned, f.Type) {
				t.Errorf("agent field %s has type %v, want no money, price, quantity or region type", f.Name, f.Type)
			}
			for _, w := range words {
				if strings.Contains(strings.ToLower(f.Name), w) {
					t.Errorf("agent field %s is named for %q, want no balance, stock or region field", f.Name, w)
				}
			}
		})
	}
}

// TestAgent_HasJob verifies the job predicate: a job is in progress when it has
// executions and ticks remaining, and the cleared job is not one.
//
// Requirement: FR-AGT-001
func TestAgent_HasJob(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		job  job
		want bool
	}{
		{"cleared job", job{}, false},
		{"running job", job{executions: 2, remaining: 3}, true},
		{"no ticks remaining", job{executions: 2}, false},
		{"no executions", job{remaining: 3}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a := agent{job: tt.job}
			if got := a.hasJob(); got != tt.want {
				t.Errorf("hasJob() with %+v = %v, want %v", tt.job, got, tt.want)
			}
		})
	}
}

// TestNewAgent_BuildsFromRoleRow verifies that newAgent copies the creation
// fields, sizes the residues by the role's recipes (grow has two outputs, so
// 2; a role with no recipes has 0 and is valid), sizes the history by the
// role's window (30, or 2 * 1 = 2 against the world's 30 floor), leaves the
// job cleared and the counters 0, sets roleSince to the creation tick and
// reserves room for the beliefs.
//
// Requirement: FR-AGT-004, FR-AGT-005, FR-AGT-010, FR-AGT-053, FR-AGT-061
func TestNewAgent_BuildsFromRoleRow(t *testing.T) {
	t.Parallel()
	f := newAgentFixture(t, false)
	tests := []struct {
		name         string
		role         string
		wantResidues int
	}{
		{"role with a recipe", "farmer", 2},
		{"role with an empty recipe list", "clerk", 0},
		{"role with a smaller tradable set", "miner", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			id, row := f.role(t, tt.role)
			a := newAgent(agentInit{id: 9, account: 4, role: id, weight: 7, strategy: 2, external: true, mortal: true, tick: 12}, row, &f.rt)
			if a.id != 9 || a.account != 4 || a.role != id || a.weight != 7 || a.strategy != 2 || !a.external || !a.mortal {
				t.Errorf("newAgent identity = %+v, want id 9 account 4 role %d weight 7 strategy 2 external mortal", a, id)
			}
			if len(a.residues) != tt.wantResidues || slices.Max(append([]int64{0}, a.residues...)) != 0 {
				t.Errorf("residues = %v, want %d zeros", a.residues, tt.wantResidues)
			}
			if got := a.history.Window(); got != row.window {
				t.Errorf("history window = %d, want the role's %d", got, row.window)
			}
			if a.hasJob() || a.directive != nil || a.unfed != 0 || a.hunger != 0 || a.privation != 0 || a.target != 0 {
				t.Errorf("newAgent state = job %v unfed %d hunger %d privation %d target %d, want all clear", a.job, a.unfed, a.hunger, a.privation, a.target)
			}
			if a.roleSince != 12 {
				t.Errorf("roleSince = %d, want the creation tick 12", a.roleSince)
			}
			if len(a.beliefs) != 0 || cap(a.beliefs) != len(row.tradable) {
				t.Errorf("beliefs len %d cap %d, want 0 and %d", len(a.beliefs), cap(a.beliefs), len(row.tradable))
			}
		})
	}
}

// TestNewAgent_Weight verifies the weight bound: 1 and MaxWeight build, and a
// weight below 1 is a violated invariant that panics with a message that says so.
//
// Requirement: FR-AGT-001
func TestNewAgent_Weight(t *testing.T) {
	t.Parallel()
	f := newAgentFixture(t, false)
	id, row := f.role(t, "farmer")
	for _, w := range []int64{1, MaxWeight} {
		a := newAgent(agentInit{id: 1, role: id, weight: w}, row, &f.rt)
		if a.weight != w {
			t.Errorf("newAgent weight %d gave weight %d", w, a.weight)
		}
	}
	for _, w := range []int64{0, -3} {
		mustPanicInvariant(t, "newAgent with a weight below 1", func() {
			newAgent(agentInit{id: 1, role: id, weight: w}, row, &f.rt)
		})
	}
}

// TestDrawCreation_TakesTwoWordsInOrder verifies that the creation draw takes
// the target ratio from the first word of the agent's creation stream and the
// residue value from the second, copies the residue to every residue, in
// [0, 10 000), and takes nothing else: the next word of the stream is the
// third, whatever the spread (0 or 2 000), the seed and the agent ID.
//
// Requirement: FR-AGT-006, FR-AGT-005
func TestDrawCreation_TakesTwoWordsInOrder(t *testing.T) {
	t.Parallel()
	f := newAgentFixture(t, false)
	id, row := f.role(t, "farmer")
	for _, seed := range []uint64{42, 7} {
		for _, agentID := range []int64{1, 2} {
			for _, spread := range []int64{0, 2000} {
				s := newStream(seed)
				s.keyed(siteConstruction, 0, agentID)
				a := newAgent(agentInit{id: AgentID(agentID), role: id, weight: 1}, row, &f.rt)
				drawCreation(s, &a, 15000, spread)
				third := s.word()

				// Replay: the first word gives the target in [lo, hi], the second the
				// residue, the third is left.
				r := newStream(seed)
				r.keyed(siteConstruction, 0, agentID)
				wantTarget := newTarget(r, 15000, spread)
				wantResidue := r.draw(10_000)
				wantThird := r.word()
				if a.target != wantTarget || third != wantThird {
					t.Errorf("seed %d agent %d spread %d: target %d next word %d, want %d and %d", seed, agentID, spread, a.target, third, wantTarget, wantThird)
				}
				if len(a.residues) != 2 || a.residues[0] != wantResidue || a.residues[1] != wantResidue || wantResidue < 0 || wantResidue >= 10_000 {
					t.Errorf("seed %d agent %d spread %d: residues %v, want two copies of %d in [0, 10000)", seed, agentID, spread, a.residues, wantResidue)
				}
			}
		}
	}
}

// TestDrawCreation_RoleWithoutResiduesStillDrawsBoth verifies that a role with
// no recipes takes both words all the same, so the stream does not depend on
// the role (spec-agents.md 3.5.2, "whatever the configuration").
//
// Requirement: FR-AGT-006
func TestDrawCreation_RoleWithoutResiduesStillDrawsBoth(t *testing.T) {
	t.Parallel()
	f := newAgentFixture(t, false)
	idF, rowF := f.role(t, "farmer")
	idC, rowC := f.role(t, "clerk")
	next := func(id RoleID, row *roleRow) uint64 {
		s := newStream(5)
		s.keyed(siteConstruction, 0, 3)
		a := newAgent(agentInit{id: 3, role: id, weight: 1}, row, &f.rt)
		drawCreation(s, &a, 15000, 1000)
		return s.word()
	}
	if got, want := next(idC, rowC), next(idF, rowF); got != want {
		t.Errorf("word after drawCreation for a role with no residues = %d, want %d as for one with residues", got, want)
	}
}

// TestDefaultEndowment verifies the endowment of an initial agent: weight times
// the start money and weight times each start quantity, in the role's stock
// order, and the overflow flag. Farmer: money 10 000, stock ore 2 and staple 5;
// weight 7 gives 70 000, ore 14, staple 35.
//
// Requirement: FR-AGT-005, FR-AGT-061
func TestDefaultEndowment(t *testing.T) {
	t.Parallel()
	f := newAgentFixture(t, false)
	farmer := &f.cfg.Roles[mustRole(t, f.cfg, "farmer")]
	ore, staple := CommodityID(f.com(t, "ore")), CommodityID(f.com(t, "staple"))
	tests := []struct {
		name      string
		weight    int64
		wantMoney Money
		wantStock []itemRow
		wantOK    bool
	}{
		{"weight 1", 1, 10000, []itemRow{{ore, 2}, {staple, 5}}, true},
		{"weight 7", 7, 70000, []itemRow{{ore, 14}, {staple, 35}}, true},
		{"money overflows", 1 << 62, 0, nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			money, stock, ok := defaultEndowment(f.cfg, farmer, tt.weight)
			if ok != tt.wantOK || money != tt.wantMoney || !slices.Equal(stock, tt.wantStock) {
				t.Errorf("defaultEndowment(farmer, %d) = (%d, %v, %v), want (%d, %v, %v)", tt.weight, money, stock, ok, tt.wantMoney, tt.wantStock, tt.wantOK)
			}
		})
	}
	t.Run("stock overflows with no start money", func(t *testing.T) {
		t.Parallel()
		r := Role{ID: "farmer", StartStock: []Item{{Commodity: "ore", Quantity: 4}}}
		if _, _, ok := defaultEndowment(f.cfg, &r, 1<<62); ok {
			t.Errorf("defaultEndowment with quantity 4 and weight 2^62 reported ok, want overflow")
		}
	})
}

func mustRole(t *testing.T, c *Config, name string) RoleID {
	t.Helper()
	id, ok := c.roleID(name)
	if !ok {
		t.Fatalf("role %q not in the configuration", name)
	}
	return id
}

// TestFundInitial verifies the initial endowment through the ledger: agents of
// weight 1 and 7 hold w * StartMoney and w * each start quantity, the
// construction bucket holds the same under the initial faucet and the audit
// passes.
//
// Requirement: FR-AGT-005, FR-AGT-066
func TestFundInitial(t *testing.T) {
	t.Parallel()
	f := newAgentFixture(t, true)
	farmer := &f.cfg.Roles[mustRole(t, f.cfg, "farmer")]
	var wantMoney int64
	wantGoods := map[string]int64{}
	for _, w := range []int64{1, 7} {
		acct := f.open(t, "warm")
		money, stock, ok := defaultEndowment(f.cfg, farmer, w)
		if !ok {
			t.Fatalf("defaultEndowment(%d) overflowed", w)
		}
		if err := fundInitial(f.l, acct, money, stock); err != nil {
			t.Fatalf("fundInitial(weight %d): %v", w, err)
		}
		v, err := f.l.Account(acct)
		if err != nil || v.Balance != w*10000 {
			t.Errorf("weight %d account balance = %d (err %v), want %d", w, v.Balance, err, w*10000)
		}
		if got := f.held(t, acct, "ore"); got != w*2 {
			t.Errorf("weight %d ore = %d, want %d", w, got, w*2)
		}
		if got := f.held(t, acct, "staple"); got != w*5 {
			t.Errorf("weight %d staple = %d, want %d", w, got, w*5)
		}
		wantMoney += w * 10000
		wantGoods["ore"] += w * 2
		wantGoods["staple"] += w * 5
	}
	// 1 * 10 000 + 7 * 10 000 = 80 000 money; ore 2 + 14 = 16; staple 5 + 35 = 40.
	gotMoney, gotGoods := int64(0), map[int]int64{}
	for _, fl := range f.l.AppendFlows(nil, true) {
		if fl.Reason != ledger.Initial {
			t.Errorf("construction bucket holds a flow of reason %v, want only initial", fl.Reason)
			continue
		}
		if fl.Commodity == ledger.Money {
			gotMoney += fl.Total
		} else {
			gotGoods[fl.Commodity] += fl.Total
		}
	}
	if gotMoney != wantMoney || gotGoods[f.com(t, "ore")] != wantGoods["ore"] || gotGoods[f.com(t, "staple")] != wantGoods["staple"] {
		t.Errorf("construction bucket = money %d goods %v, want money %d goods %v", gotMoney, gotGoods, wantMoney, wantGoods)
	}
	if err := f.l.Audit(); err != nil {
		t.Errorf("Audit after funding: %v", err)
	}
}

// TestFundInitial_SkipsZeroAmounts verifies that a start money of 0 and a
// zero-quantity item move nothing and return no error, since the ledger refuses
// an amount below 1.
//
// Requirement: FR-AGT-005, FR-AGT-061
func TestFundInitial_SkipsZeroAmounts(t *testing.T) {
	t.Parallel()
	f := newAgentFixture(t, true)
	acct := f.open(t, "warm")
	if err := fundInitial(f.l, acct, 0, []itemRow{{CommodityID(f.com(t, "ore")), 0}}); err != nil {
		t.Errorf("fundInitial with zero amounts = %v, want nil", err)
	}
	if v, _ := f.l.Account(acct); v.Balance != 0 || f.held(t, acct, "ore") != 0 {
		t.Errorf("account after a zero endowment holds %d money and %d ore, want 0 and 0", v.Balance, f.held(t, acct, "ore"))
	}
	if flows := f.l.AppendFlows(nil, true); len(flows) != 0 {
		t.Errorf("construction bucket after a zero endowment = %v, want empty", flows)
	}
}

// TestFundInitial_LedgerRefusalIsReturned verifies that a refusal of the
// ledger comes back as an error that wraps the ledger's sentinel: a funding
// above the money bound, and an unknown account.
//
// Requirement: FR-AGT-005
//
// Guard: the world fails construction on this error; swallowing it would
// leave an agent unfunded with no trace.
func TestFundInitial_LedgerRefusalIsReturned(t *testing.T) {
	t.Parallel()
	f := newAgentFixture(t, true)
	acct := f.open(t, "warm")
	if err := fundInitial(f.l, acct, Money(MaxMoney+1), nil); err == nil {
		t.Errorf("fundInitial above MaxMoney = nil, want an error")
	}
	if err := fundInitial(f.l, acct, 0, []itemRow{{CommodityID(f.com(t, "ore")), Quantity(MaxQuantity + 1)}}); err == nil {
		t.Errorf("fundInitial above MaxQuantity = nil, want an error")
	}
	if err := fundInitial(f.l, 9999, 1, nil); err == nil {
		t.Errorf("fundInitial into an unknown account = nil, want an error")
	}
}

// TestFundInitial_NoBoundaryIsAnError verifies the error when the ledger has no
// initial boundary account, for money and for goods (a ledger that was not
// opened by newLedger).
//
// Requirement: FR-AGT-005
//
// Guard: the boundary accounts are opened once by newLedger; their absence
// must be reported, not read as a zero account.
func TestFundInitial_NoBoundaryIsAnError(t *testing.T) {
	t.Parallel()
	l, err := ledger.New(ledger.Config{Commodities: []ledger.Commodity{{}}, Regions: 1, MaxMoney: MaxMoney, MaxQuantity: MaxQuantity})
	if err != nil {
		t.Fatalf("ledger.New: %v", err)
	}
	acct, err := l.Open(ledger.KindAgent, 0, 0)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if err := fundInitial(l, acct, 5, nil); err == nil {
		t.Errorf("fundInitial of money with no boundary = nil, want an error")
	}
	if err := fundInitial(l, acct, 0, []itemRow{{0, 5}}); err == nil {
		t.Errorf("fundInitial of goods with no boundary = nil, want an error")
	}
}

// TestUpkeepTotals verifies that the total upkeep is the weight times the
// per-household quantity of each row, in the order of the rows, for weights 1,
// 3 and MaxWeight, for the base list (staple 3, fuel 1) and for the cold
// region's merged list (fuel 1 + 2 = 3), and that a product past int64 reports
// not ok.
//
// Requirement: FR-AGT-011, FR-SCL-007
func TestUpkeepTotals(t *testing.T) {
	t.Parallel()
	f := newAgentFixture(t, false)
	_, farmer := f.role(t, "farmer")
	fuel, staple := CommodityID(f.com(t, "fuel")), CommodityID(f.com(t, "staple"))
	tests := []struct {
		name   string
		per    []itemRow
		weight int64
		want   []itemRow
		wantOK bool
	}{
		{"weight 1", farmer.upkeepIn(f.region(t, "warm")), 1, []itemRow{{fuel, 1}, {staple, 3}}, true},
		{"weight 3", farmer.upkeepIn(f.region(t, "warm")), 3, []itemRow{{fuel, 3}, {staple, 9}}, true},
		{"max weight", farmer.upkeepIn(f.region(t, "warm")), MaxWeight, []itemRow{{fuel, MaxWeight}, {staple, 3 * MaxWeight}}, true},
		{"cold region merges the extra fuel", farmer.upkeepIn(f.region(t, "cold")), 4, []itemRow{{fuel, 12}, {staple, 12}}, true},
		{"no upkeep", nil, 5, nil, true},
		{"product overflows", []itemRow{{fuel, 3}}, 1 << 62, nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := upkeepTotals(make([]itemRow, 0, 4), tt.per, tt.weight)
			if ok != tt.wantOK || (ok && !slices.Equal(got, tt.want)) {
				t.Errorf("upkeepTotals(%v, %d) = (%v, %v), want (%v, %v)", tt.per, tt.weight, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

// TestConsume_Cohort verifies the worked example of spec-agents.md 16.3: a
// cohort of 40 with upkeep staple 3 and fuel 1 per household needs staple 120
// and fuel 40, holds staple 100 and fuel 40, and consumes 100 and 40, so Fed =
// min(floor(100 / 3), floor(40 / 1)) = min(33, 40) = 33 and Unfed = 40 - 33 =
// 7; hunger and privation become 1; the shortfall is 20 staple and 0 fuel.
//
// Requirement: FR-AGT-013, FR-AGT-014, FR-AGT-015, FR-AGT-075, FR-SCL-007
func TestConsume_Cohort(t *testing.T) {
	t.Parallel()
	f := newAgentFixture(t, false)
	acct := f.open(t, "warm")
	f.give(t, acct, "staple", 100)
	f.give(t, acct, "fuel", 40)
	a := f.newAgent(t, "farmer", acct, 40)
	_, row := f.role(t, "farmer")
	var sc consumeScratch
	if err := consume(f.l, &a, acct, row.upkeepIn(f.region(t, "warm")), &sc); err != nil {
		t.Fatalf("consume: %v", err)
	}
	if a.unfed != 7 || a.hunger != 1 || a.privation != 1 {
		t.Errorf("after consume unfed %d hunger %d privation %d, want 7, 1, 1", a.unfed, a.hunger, a.privation)
	}
	if !slices.Equal(sc.short, []int64{0, 20}) {
		t.Errorf("shortfall = %v, want [0 20] (fuel 40 - 40, staple 120 - 100)", sc.short)
	}
	if f.held(t, acct, "staple") != 0 || f.held(t, acct, "fuel") != 0 {
		t.Errorf("holdings after consume = staple %d fuel %d, want 0 and 0", f.held(t, acct, "staple"), f.held(t, acct, "fuel"))
	}
	if err := f.l.Audit(); err != nil {
		t.Errorf("Audit: %v", err)
	}
}

// TestConsume_Table verifies the consumption step over the cases around the
// hand-computed example: a fed tick resets hunger to 0 and lowers privation by
// 1 to a floor of 0; an agent holding nothing is wholly unfed with no error; a
// holding above the need leaves the surplus; an agent with no upkeep is never
// unfed and its counters fall.
//
// Each case is one household of weight 2 with upkeep fuel 1 and staple 3
// (totals fuel 2, staple 6) unless it says otherwise.
//
// Requirement: FR-AGT-013, FR-AGT-014, FR-AGT-015, FR-AGT-075
func TestConsume_Table(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                               string
		noUpkeep                           bool
		fuel, staple                       int64
		hunger, privation                  int64
		wantUnfed, wantHunger, wantPrivate int64
		wantFuelLeft, wantStapleLeft       int64
	}{
		// Holds exactly the need: eats it all, both households fed.
		{name: "fed exactly", fuel: 2, staple: 6, hunger: 3, privation: 5, wantUnfed: 0, wantHunger: 0, wantPrivate: 4},
		// Fed with a surplus: the surplus stays (fuel 5 - 2 = 3, staple 10 - 6 = 4).
		{name: "holding above the need", fuel: 5, staple: 10, wantUnfed: 0, wantFuelLeft: 3, wantStapleLeft: 4},
		// Privation 0 stays 0 on a fed tick.
		{name: "privation floor", fuel: 2, staple: 6, hunger: 0, privation: 0, wantUnfed: 0, wantPrivate: 0},
		// Holds nothing: Fed = min(0/1, 0/3) = 0, both unfed, hunger 4 + 1, privation 4 + 1.
		{name: "holding nothing", hunger: 4, privation: 4, wantUnfed: 2, wantHunger: 5, wantPrivate: 5},
		// Staple 4: floor(4/3) = 1 fed, 1 unfed; fuel 2 fully eaten; staple all 4 eaten.
		{name: "one household short of staple", fuel: 2, staple: 4, wantUnfed: 1, wantHunger: 1, wantPrivate: 1},
		// Staple 3 < 6 but fuel 2: the fuel is still eaten (a household eats what it has).
		{name: "short of one good still eats the other", fuel: 2, staple: 2, wantUnfed: 2, wantHunger: 1, wantPrivate: 1},
		{name: "no upkeep", noUpkeep: true, hunger: 2, privation: 2, wantUnfed: 0, wantHunger: 0, wantPrivate: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := newAgentFixture(t, false)
			acct := f.open(t, "warm")
			if tt.fuel > 0 {
				f.give(t, acct, "fuel", tt.fuel)
			}
			if tt.staple > 0 {
				f.give(t, acct, "staple", tt.staple)
			}
			a := f.newAgent(t, "farmer", acct, 2)
			a.hunger, a.privation = tt.hunger, tt.privation
			_, row := f.role(t, "farmer")
			per := row.upkeepIn(f.region(t, "warm"))
			if tt.noUpkeep {
				per = nil
			}
			var sc consumeScratch
			if err := consume(f.l, &a, acct, per, &sc); err != nil {
				t.Fatalf("consume: %v", err)
			}
			if a.unfed != tt.wantUnfed || a.hunger != tt.wantHunger || a.privation != tt.wantPrivate {
				t.Errorf("unfed %d hunger %d privation %d, want %d, %d, %d", a.unfed, a.hunger, a.privation, tt.wantUnfed, tt.wantHunger, tt.wantPrivate)
			}
			if a.hunger > a.privation {
				t.Errorf("hunger %d above privation %d, want hunger <= privation", a.hunger, a.privation)
			}
			if tt.noUpkeep {
				return
			}
			if got := f.held(t, acct, "fuel"); got != tt.wantFuelLeft {
				t.Errorf("fuel left = %d, want %d", got, tt.wantFuelLeft)
			}
			if got := f.held(t, acct, "staple"); got != tt.wantStapleLeft {
				t.Errorf("staple left = %d, want %d", got, tt.wantStapleLeft)
			}
		})
	}
}

// TestConsume_OldestLotsFirst verifies that the units eaten are the oldest lots
// first: two staple lots born at ticks 0 and 1 (2 units each) and an upkeep of 3
// leave one unit of the lot born at tick 1.
//
// Requirement: FR-AGT-013
func TestConsume_OldestLotsFirst(t *testing.T) {
	t.Parallel()
	f := newAgentFixture(t, false)
	acct := f.open(t, "warm")
	f.give(t, acct, "staple", 2)
	f.l.AdvanceTick()
	f.give(t, acct, "staple", 2)
	a := f.newAgent(t, "farmer", acct, 1)
	var sc consumeScratch
	if err := consume(f.l, &a, acct, []itemRow{{CommodityID(f.com(t, "staple")), 3}}, &sc); err != nil {
		t.Fatalf("consume: %v", err)
	}
	lots, err := f.l.AppendLots(nil, acct)
	if err != nil {
		t.Fatalf("AppendLots: %v", err)
	}
	want := []ledger.Lot{{Commodity: f.com(t, "staple"), Quantity: 1, Born: 1}}
	if len(lots) != 1 || lots[0].Commodity != want[0].Commodity || lots[0].Quantity != 1 || lots[0].Born != 1 {
		t.Errorf("lots after eating 3 of (2 born 0, 2 born 1) = %+v, want %+v", lots, want)
	}
}

// TestConsume_FlowsAreConsumptionOnly verifies that the units eaten appear in
// the flow totals under the consumption reason by commodity and in no other
// sink: 100 staple and 40 fuel eaten, every flow of the tick other than the
// seeding is consumption.
//
// Requirement: FR-AGT-067
func TestConsume_FlowsAreConsumptionOnly(t *testing.T) {
	t.Parallel()
	f := newAgentFixture(t, false)
	acct := f.open(t, "warm")
	f.give(t, acct, "staple", 100)
	f.give(t, acct, "fuel", 40)
	a := f.newAgent(t, "farmer", acct, 40)
	_, row := f.role(t, "farmer")
	var sc consumeScratch
	if err := consume(f.l, &a, acct, row.upkeepIn(f.region(t, "warm")), &sc); err != nil {
		t.Fatalf("consume: %v", err)
	}
	got := map[int]int64{}
	for _, fl := range f.l.AppendFlows(nil, false) {
		switch fl.Reason {
		case ledger.Initial:
		case ledger.Consumption:
			got[fl.Commodity] += fl.Total
		default:
			t.Errorf("flow of reason %v total %d for commodity %d, want only initial and consumption", fl.Reason, fl.Total, fl.Commodity)
		}
	}
	if got[f.com(t, "staple")] != 100 || got[f.com(t, "fuel")] != 40 || len(got) != 2 {
		t.Errorf("consumption flows = %v, want staple 100 and fuel 40", got)
	}
}

// TestConsume_EatsFromTheNamedAccount verifies the account parameter: a call
// with from set to another account (a trainer's) takes the units from that
// account and leaves the agent's own account untouched.
//
// Requirement: FR-AGT-068
func TestConsume_EatsFromTheNamedAccount(t *testing.T) {
	t.Parallel()
	f := newAgentFixture(t, false)
	own, trainer := f.open(t, "warm"), f.open(t, "warm")
	f.give(t, own, "staple", 3)
	f.give(t, trainer, "staple", 3)
	a := f.newAgent(t, "farmer", own, 1)
	var sc consumeScratch
	if err := consume(f.l, &a, trainer, []itemRow{{CommodityID(f.com(t, "staple")), 3}}, &sc); err != nil {
		t.Fatalf("consume: %v", err)
	}
	if f.held(t, own, "staple") != 3 || f.held(t, trainer, "staple") != 0 || a.unfed != 0 {
		t.Errorf("own %d trainer %d unfed %d, want 3, 0, 0", f.held(t, own, "staple"), f.held(t, trainer, "staple"), a.unfed)
	}
}

// TestConsume_Errors verifies the refusals: an unknown account is returned as
// an error and an upkeep times weight past int64 is a violated invariant.
//
// Requirement: FR-AGT-013
//
// Guard: consume has no error path of its own in a valid world; the world
// fails the step on a returned error, so an unreported one would hide a
// corrupt ledger.
func TestConsume_Errors(t *testing.T) {
	t.Parallel()
	f := newAgentFixture(t, false)
	acct := f.open(t, "warm")
	a := f.newAgent(t, "farmer", acct, 1)
	var sc consumeScratch
	per := []itemRow{{CommodityID(f.com(t, "staple")), 3}}
	if err := consume(f.l, &a, 9999, per, &sc); err == nil {
		t.Errorf("consume from an unknown account = nil, want an error")
	}
	if err := consume(f.l, &a, acct, []itemRow{{CommodityID(99), 3}}, &sc); err == nil {
		t.Errorf("consume of an unknown commodity = nil, want an error")
	}
	big := f.newAgent(t, "farmer", acct, 1<<62)
	mustPanicInvariant(t, "consume with an overflowing total", func() { _ = consume(f.l, &big, acct, per, &sc) })
}

// TestConsume_NoAllocsWhenWarm verifies that the consumption step allocates
// nothing once the scratch has grown to the upkeep rows.
//
// Requirement: NFR-AGT-001
func TestConsume_NoAllocsWhenWarm(t *testing.T) {
	f := newAgentFixture(t, false)
	acct := f.open(t, "warm")
	f.give(t, acct, "staple", 500_000)
	f.give(t, acct, "fuel", 500_000)
	a := f.newAgent(t, "farmer", acct, 3)
	_, row := f.role(t, "farmer")
	per := row.upkeepIn(f.region(t, "warm"))
	sc := consumeScratch{}
	if err := consume(f.l, &a, acct, per, &sc); err != nil {
		t.Fatalf("warm-up consume: %v", err)
	}
	if got := testing.AllocsPerRun(100, func() {
		if err := consume(f.l, &a, acct, per, &sc); err != nil {
			t.Fatalf("consume: %v", err)
		}
	}); got != 0 {
		t.Errorf("consume allocates %v times per call with a warm scratch, want 0", got)
	}
}

// TestConsume_ConservesGoodsOverSeeds verifies, over 50 seeds, that random
// consumption on a small fixture conserves goods exactly (what is held plus
// what was eaten equals what was given), that the audit passes after every
// round and that hunger never exceeds privation. A failure names the seed.
//
// Requirement: FR-AGT-066, FR-AGT-075
func TestConsume_ConservesGoodsOverSeeds(t *testing.T) {
	t.Parallel()
	for seed := uint64(1); seed <= 50; seed++ {
		f := newAgentFixture(t, false)
		rng := rand.New(rand.NewPCG(seed, 0))
		_, row := f.role(t, "farmer")
		fuel, staple := f.com(t, "fuel"), f.com(t, "staple")
		given := map[int]int64{}
		var agents []agent
		var accts []int
		for range 5 {
			acct := f.open(t, "warm")
			fu, st := rng.Int64N(60), rng.Int64N(200)
			if fu > 0 {
				f.give(t, acct, "fuel", fu)
			}
			if st > 0 {
				f.give(t, acct, "staple", st)
			}
			given[fuel] += fu
			given[staple] += st
			agents = append(agents, f.newAgent(t, "farmer", acct, 1+rng.Int64N(20)))
			accts = append(accts, acct)
		}
		eaten := map[int]int64{}
		var sc consumeScratch
		for round := range 10 {
			for i := range agents {
				if err := consume(f.l, &agents[i], accts[i], row.upkeepIn(f.region(t, "warm")), &sc); err != nil {
					t.Fatalf("seed %d round %d: consume: %v", seed, round, err)
				}
				for j, it := range sc.totals {
					eaten[int(it.commodity)] += sc.consumed[j]
				}
				if agents[i].hunger > agents[i].privation {
					t.Fatalf("seed %d round %d: hunger %d above privation %d", seed, round, agents[i].hunger, agents[i].privation)
				}
			}
			if err := f.l.Audit(); err != nil {
				t.Fatalf("seed %d round %d: Audit: %v", seed, round, err)
			}
		}
		for _, c := range []int{fuel, staple} {
			var held int64
			for _, acct := range accts {
				h, _, err := f.l.Holding(acct, c)
				if err != nil {
					t.Fatalf("seed %d: Holding: %v", seed, err)
				}
				held += h
			}
			if held+eaten[c] != given[c] {
				t.Errorf("seed %d commodity %d: held %d + eaten %d != given %d", seed, c, held, eaten[c], given[c])
			}
		}
	}
}

func beliefParamsForTest() beliefs.Params {
	return beliefs.Params{Step: 1000, AdjustProb: 5000, Significance: 100, EvidenceShare: 5000,
		StockLow: 2000, StockHigh: 8000, MaxAge: 5, WidthInit: 2000, WidthMin: 500, WidthMax: 6000, WidthStep: 500,
		CostFloor: 5000, CostCeiling: 20000, RefBandLow: 5000, RefBandHigh: 20000, MaxPrice: MaxPrice}
}

// seedsFor returns a seed per commodity of row.tradable with reference price
// 100, no unit cost, and the window empty at the market price price.
func seedsFor(row *roleRow, price int64) []beliefSeed {
	src := make([]beliefSeed, len(row.tradable))
	for i := range src {
		src[i] = beliefSeed{Ref: 100, Empty: true, Price: price}
	}
	return src
}

// TestReseedBeliefs_Creation verifies that a creation (no old role) gives one
// belief per commodity of the tradable set, in canonical order, at the seed
// price with the width BeliefWidthInit: farmer trades fuel, ore, staple, so
// three beliefs of centre 100 and width 2 000 (inside the bounds of reference
// 100 with the band 50% to 200%).
//
// Requirement: FR-AGT-004, FR-AGT-005
func TestReseedBeliefs_Creation(t *testing.T) {
	t.Parallel()
	f := newAgentFixture(t, false)
	_, row := f.role(t, "farmer")
	a := f.newAgent(t, "farmer", 0, 1)
	p := beliefParamsForTest()
	reseedBeliefs(&a, row, nil, seedsFor(row, 100), p)
	want := []beliefs.Belief{{Centre: 100, Width: 2000}, {Centre: 100, Width: 2000}, {Centre: 100, Width: 2000}}
	if len(row.tradable) != 3 || !slices.Equal(a.beliefs, want) {
		t.Errorf("beliefs for %v = %v, want %v", row.tradable, a.beliefs, want)
	}
}

// TestReseedBeliefs_RoleChange verifies that a role change keeps the belief of
// a commodity in the tradable sets of both roles, reseeds the commodities new
// to the set and drops those that left it. Farmer (fuel, ore, staple) to miner
// (fuel): the fuel belief keeps its centre 130 and width 900 and the others are
// dropped; back from miner to farmer, fuel is kept and ore and staple start at
// the seed price 100 with width 2 000.
//
// Requirement: FR-AGT-004
func TestReseedBeliefs_RoleChange(t *testing.T) {
	t.Parallel()
	f := newAgentFixture(t, false)
	_, farmer := f.role(t, "farmer")
	_, miner := f.role(t, "miner")
	a := f.newAgent(t, "farmer", 0, 1)
	p := beliefParamsForTest()
	reseedBeliefs(&a, farmer, nil, seedsFor(farmer, 100), p)
	a.beliefs[0] = beliefs.Belief{Centre: 130, Width: 900} // fuel, the first of the tradable set
	reseedBeliefs(&a, miner, farmer, seedsFor(miner, 100), p)
	if want := []beliefs.Belief{{Centre: 130, Width: 900}}; !slices.Equal(a.beliefs, want) {
		t.Errorf("beliefs after farmer to miner = %v, want %v", a.beliefs, want)
	}
	reseedBeliefs(&a, farmer, miner, seedsFor(farmer, 100), p)
	want := []beliefs.Belief{{Centre: 130, Width: 900}, {Centre: 100, Width: 2000}, {Centre: 100, Width: 2000}}
	if !slices.Equal(a.beliefs, want) {
		t.Errorf("beliefs after miner to farmer = %v, want %v", a.beliefs, want)
	}
}

// TestReseedBeliefs_Invariants verifies the programmer-error panics: too few
// seeds, and stored beliefs that do not match the old role's tradable set.
//
// Requirement: FR-AGT-004
//
// Guard: both mean the caller's tables disagree with the agent; silently
// reseeding would pair a belief with the wrong commodity.
func TestReseedBeliefs_Invariants(t *testing.T) {
	t.Parallel()
	f := newAgentFixture(t, false)
	_, farmer := f.role(t, "farmer")
	_, miner := f.role(t, "miner")
	a := f.newAgent(t, "farmer", 0, 1)
	p := beliefParamsForTest()
	mustPanicInvariant(t, "too few seeds", func() { reseedBeliefs(&a, farmer, nil, nil, p) })
	mustPanicInvariant(t, "beliefs do not match the old row", func() { reseedBeliefs(&a, miner, farmer, seedsFor(miner, 100), p) })
}

// TestReseedBeliefs_ReusesTheSlice verifies that reseeding allocates nothing
// when the agent's slice and the seed scratch have room, including a role
// change to a smaller set.
//
// Requirement: NFR-AGT-001
func TestReseedBeliefs_ReusesTheSlice(t *testing.T) {
	f := newAgentFixture(t, false)
	_, farmer := f.role(t, "farmer")
	a := f.newAgent(t, "farmer", 0, 1)
	p := beliefParamsForTest()
	src := seedsFor(farmer, 100)
	reseedBeliefs(&a, farmer, nil, src, p)
	if got := testing.AllocsPerRun(100, func() { reseedBeliefs(&a, farmer, farmer, src, p) }); got != 0 {
		t.Errorf("reseedBeliefs allocates %v times per call with room, want 0", got)
	}
}

// TestAgentFile_NoRandomNoLoggingNoEvent verifies by reading agent.go that it
// draws no random number except through the stream parameter of drawCreation,
// and logs and emits nothing: it imports neither log/slog nor math/rand.
//
// Guard: the agent step is observation-free (observability-and-logging.md 1.4);
// an import of a logger or a random source here would be the first sign of a
// result-affecting side channel.
func TestAgentFile_NoRandomNoLoggingNoEvent(t *testing.T) {
	t.Parallel()
	b, err := os.ReadFile("agent.go")
	if err != nil {
		t.Fatalf("read agent.go: %v", err)
	}
	src := string(b)
	for _, banned := range []string{`"log/slog"`, `"math/rand`, `"time"`, `"os"`, "Observe(", "slog."} {
		if strings.Contains(src, banned) {
			t.Errorf("agent.go contains %q, want no logger, event or random source", banned)
		}
	}
}
