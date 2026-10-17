package economysim_test

import (
	"fmt"
	"math/rand/v2"
	"os"
	"slices"
	"sort"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/config"
	"github.com/Whealinghold/economy-sim/internal/beliefs"
	"github.com/Whealinghold/economy-sim/internal/welfare"
)

// viewPacks are the two settings the view and the strategy are checked over: the
// first consumer's chain and an unrelated one, with no code specific to either
// (architecture.md 9.4).
var viewPacks = []struct {
	name   string
	layers []string
}{
	{"one", []string{"base.toml", "one.toml", "warm-one.toml"}},
	{"second", []string{"second/second.toml", "second/warm-second.toml"}},
}

// loadViewPack loads a pack and builds the harness for it.
func loadViewPack(t *testing.T, layers []string) *economysim.ViewHarness {
	t.Helper()
	raw, err := config.LoadFS(os.DirFS("testdata/chains"), layers...)
	if err != nil {
		t.Fatalf("LoadFS(%v) error = %v, want nil", layers, err)
	}
	return economysim.NewViewHarness(t, raw)
}

// packIndex resolves the IDs of a canonical configuration, which are the
// positions in its slices.
type packIndex struct {
	cfg       *economysim.Config
	commodity map[string]int
	recipe    map[string]int
	roles     economysim.RoleTable
}

func newPackIndex(cfg *economysim.Config) *packIndex {
	p := &packIndex{cfg: cfg, commodity: map[string]int{}, recipe: map[string]int{}, roles: economysim.NewRoleTable(cfg)}
	for i, c := range cfg.Commodities {
		p.commodity[c.ID] = i
	}
	for i, r := range cfg.Recipes {
		p.recipe[r.ID] = i
	}
	return p
}

// upkeep is the effective upkeep of one household of a role in a region: the
// role's own list when it has one, else the world's, plus the region's extra,
// summed by commodity in ascending commodity ID.
func (p *packIndex) upkeep(role economysim.Role, region economysim.Region) []economysim.UpkeepRow {
	base := p.cfg.World.Upkeep
	if len(role.Upkeep) > 0 {
		base = role.Upkeep
	}
	sum := map[int]int64{}
	for _, it := range slices.Concat(base, region.ExtraUpkeep) {
		sum[p.commodity[it.Commodity]] += int64(it.Quantity)
	}
	var rows []economysim.UpkeepRow
	for c, q := range sum {
		rows = append(rows, economysim.UpkeepRow{Commodity: economysim.CommodityID(c), Quantity: q})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].Commodity < rows[j].Commodity })
	return rows
}

// makes reports whether a recipe of the role makes the commodity as a
// non-discard output.
func (p *packIndex) makes(role economysim.Role, commodity string) bool {
	for _, rn := range role.Recipes {
		for _, o := range p.cfg.Recipes[p.recipe[rn]].Outputs {
			if o.Commodity == commodity && !o.Discard {
				return true
			}
		}
	}
	return false
}

// isTool reports whether any recipe of the configuration requires the commodity.
func (p *packIndex) isTool(commodity string) bool {
	for _, r := range p.cfg.Recipes {
		for _, tl := range r.Requires {
			if tl.Commodity == commodity {
				return true
			}
		}
	}
	return false
}

// meanHalfEven is the mean of the prices rounded half to even
// (spec-beliefs.md 7.2).
func meanHalfEven(prices []int64) int64 {
	var sum int64
	for _, p := range prices {
		sum += p
	}
	n := int64(len(prices))
	q, r := sum/n, sum%n
	if 2*r > n || (2*r == n && q%2 == 1) {
		q++
	}
	return q
}

// state is what a test put in one agent, so the expectation can be computed
// from it and not from the view.
type state struct {
	agent      int
	roleID     int
	region     int
	weight     int64
	money      int64
	incomeSum  int64 // the welfare ring's sums and length
	costSum    int64
	held       int64
	hunger     int64
	stock      map[int]int64
	beliefs    map[int]economysim.Belief
	reports    map[int]economysim.MarketReport
	window     map[int][]int64
	wage       int64
	wantJob    bool
	jobRecipe  int
	jobLeft    int64
	jobRuns    int64
	tradableID []int
}

// sampleState puts a random state in a new agent of the role in the region: a
// weight, stocks, money, hunger, market reports and windows, a wage, beliefs
// anywhere in their bounds and, sometimes, a job in progress.
func sampleState(t *testing.T, h *economysim.ViewHarness, p *packIndex, rng *rand.Rand, roleID, regionID int) *state {
	t.Helper()
	cfg := h.Cfg
	role, region := cfg.Roles[roleID], cfg.Regions[regionID]
	rr := economysim.RoleRow(&p.roles, economysim.RoleID(roleID))
	weights := []int64{1, 2, 7, 40, 500}
	s := &state{
		roleID: roleID, region: regionID, weight: weights[rng.IntN(len(weights))], tradableID: rr.Tradable,
		stock: map[int]int64{}, beliefs: map[int]economysim.Belief{}, reports: map[int]economysim.MarketReport{}, window: map[int][]int64{},
	}
	s.agent = h.Spawn(t, role.ID, region.ID, s.weight)
	moneys := []int64{0, int64(rng.IntN(2000)), s.weight * int64(rng.IntN(20_000)), s.weight * int64(rng.IntN(5_000_000))}
	s.money = moneys[rng.IntN(len(moneys))]
	if s.money > 0 {
		h.Fund(t, s.agent, s.money)
	}
	if rng.IntN(2) == 0 {
		s.hunger = int64(1 + rng.IntN(4))
	}
	h.SetHunger(s.agent, s.hunger, min(s.hunger, s.weight))
	for range min(rng.IntN(4), int(economysim.RoleWindow(&p.roles, economysim.RoleID(roleID)))) {
		inc, cost := rng.Int64N(5000), 1+rng.Int64N(5000)
		h.PushHistory(s.agent, inc, cost, rng.IntN(2) == 0)
		s.incomeSum, s.costSum, s.held = s.incomeSum+inc, s.costSum+cost, s.held+1
	}
	for _, c := range rr.Tradable {
		name := cfg.Commodities[c].ID
		ref := int64(cfg.Commodities[c].ReferencePrice)
		if rng.IntN(4) > 0 {
			s.stock[c] = s.weight * int64(rng.IntN(40))
			if s.stock[c] > 0 {
				h.Give(t, s.agent, name, s.stock[c])
			}
		}
		rep := economysim.MarketReport{Price: 1 + rng.Int64N(3*ref), Age: int64(rng.IntN(8))}
		if rng.IntN(2) == 0 {
			rep.BestBid, rep.HasBestBid = 1+rng.Int64N(3*ref), true
		}
		if rng.IntN(2) == 0 {
			rep.BestAsk, rep.HasBestAsk = 1+rng.Int64N(3*ref), true
		}
		s.reports[c] = rep
		h.SetReport(t, region.ID, name, rep)
		h.ResetWindow(t, region.ID, name)
		for range rng.IntN(4) {
			pr := 1 + rng.Int64N(3*ref)
			s.window[c] = append(s.window[c], pr)
			h.PushWindow(t, region.ID, name, pr)
		}
	}
	if rng.IntN(2) == 0 {
		s.wage = 1 + rng.Int64N(200)
	}
	h.Anchors(s.wage)
	if len(role.Recipes) > 0 && rng.IntN(4) == 0 {
		s.wantJob, s.jobRecipe = true, p.recipe[role.Recipes[rng.IntN(len(role.Recipes))]]
		s.jobLeft, s.jobRuns = int64(1+rng.IntN(3)), 1+rng.Int64N(s.weight)
		h.SetJob(t, s.agent, cfg.Recipes[s.jobRecipe].ID, s.jobLeft, s.jobRuns)
	}
	// The beliefs are anywhere in their bounds, which the first fill gives.
	v, err := h.Fill(s.agent)
	if err != nil {
		t.Fatalf("fillCore error = %v, want nil", err)
	}
	bt := v.Tunables.Beliefs
	for k, c := range rr.Tradable {
		bd := v.Commodities[k].Bounds
		b := economysim.Belief{
			Centre: bd.Floor + economysim.Price(rng.Int64N(int64(bd.Ceiling-bd.Floor)+1)),
			Width:  bt.WidthMin + rng.Int64N(bt.WidthMax-bt.WidthMin+1),
		}
		s.beliefs[c] = b
		h.SetBelief(s.agent, k, int64(b.Centre), b.Width)
	}
	return s
}

// TestFillCore_AgreesWithIndependentComputation verifies, over both packs, every
// role in every region and a range of random states (stock, money, hunger,
// weights, a job in progress, published prices, empty and filled price windows,
// known and unknown wages), that each core group of the filled view equals the
// same quantity computed in the test from the state and the configuration:
// the tradable set, stocks, beliefs, bounds (with the zeroed unit cost), the
// reference prices, the cost anchors of the makers only, the tool flag over the
// whole configuration, the market reports and windows, the merged upkeep, the
// welfare ratio, the recipes with their lines and yields and the executions a
// direct count gives.
//
// Requirement: FR-AGT-024, FR-AGT-048, FR-AGT-071, FR-AGT-073, FR-AGT-038
func TestFillCore_AgreesWithIndependentComputation(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	t.Parallel()
	for _, pk := range viewPacks {
		t.Run(pk.name, func(t *testing.T) {
			t.Parallel()
			h := loadViewPack(t, pk.layers)
			p := newPackIndex(h.Cfg)
			tab := economysim.NewYieldTable(h.Cfg)
			rng := rand.New(rand.NewPCG(7, 11))
			checked := 0
			for ro := range h.Cfg.Roles {
				for g := range h.Cfg.Regions {
					for range 5 {
						s := sampleState(t, h, p, rng, ro, g)
						checkView(t, h, p, &tab, s)
						checked++
					}
				}
			}
			if checked == 0 {
				t.Fatalf("no state was checked")
			}
		})
	}
}

// checkView fills the view for the state and compares every group.
func checkView(t *testing.T, h *economysim.ViewHarness, p *packIndex, tab *economysim.YieldTable, s *state) {
	t.Helper()
	cfg := h.Cfg
	role, region := cfg.Roles[s.roleID], cfg.Regions[s.region]
	where := fmt.Sprintf("role %s, region %s, weight %d", role.ID, region.ID, s.weight)
	v, err := h.Fill(s.agent)
	if err != nil {
		t.Fatalf("%s: fillCore error = %v, want nil", where, err)
	}
	if v.Agent != economysim.AgentID(s.agent) || int(v.Role) != s.roleID || int(v.Region) != s.region || v.Weight != s.weight ||
		v.Money != s.money || v.Hunger != s.hunger || v.Window != int64(economysim.RoleWindow(&p.roles, v.Role)) || v.TargetRatio != role.TargetRatio ||
		v.MinWorkShare != role.MinWorkShare {
		t.Errorf("%s: identity, money and state = %+v, want agent %d, money %d, hunger %d", where, v, s.agent, s.money, s.hunger)
	}
	wantStarts := int64(1)
	if s.wantJob {
		wantStarts = s.jobLeft + 1
	}
	if v.HasJob != s.wantJob || v.StartsIn != wantStarts || (s.wantJob && (int(v.Recipe) != s.jobRecipe || v.Remaining != s.jobLeft || v.Executions != s.jobRuns)) {
		t.Errorf("%s: job = (%v, recipe %d, %d left, %d runs, starts in %d), want job %v, starts in %d", where, v.HasJob, v.Recipe, v.Remaining, v.Executions, v.StartsIn, s.wantJob, wantStarts)
	}
	up := p.upkeep(role, region)
	if !slices.Equal(v.Upkeep, up) {
		t.Errorf("%s: Upkeep = %v, want %v", where, v.Upkeep, up)
	}
	q, pr := make([]int64, len(up)), make([]int64, len(up))
	for i, u := range up {
		q[i] = u.Quantity
		pr[i] = int64(h.PublishedPrice(economysim.RegionID(s.region), u.Commodity))
	}
	if ratio, present := welfare.Ratio(s.incomeSum, s.costSum, v.Window-s.held, welfare.TickCost(q, pr, s.weight), role.DefaultRatio); v.Ratio != ratio || v.HasRatio != present {
		t.Errorf("%s: ratio = %d (%v), want %d (%v)", where, v.Ratio, v.HasRatio, ratio, present)
	}
	checkCommodities(t, h, p, s, v, where)
	checkRecipes(t, h, p, tab, s, v, where)
}

func checkCommodities(t *testing.T, h *economysim.ViewHarness, p *packIndex, s *state, v *economysim.AgentView, where string) {
	t.Helper()
	cfg := h.Cfg
	role := cfg.Roles[s.roleID]
	if len(v.Commodities) != len(s.tradableID) {
		t.Fatalf("%s: %d commodities, want the role's %d", where, len(v.Commodities), len(s.tradableID))
	}
	for k, c := range s.tradableID {
		cv := v.Commodities[k]
		name, ref := cfg.Commodities[c].ID, cfg.Commodities[c].ReferencePrice
		var cost int64
		if p.makes(role, name) {
			cost = h.AnchorOf(economysim.RegionID(s.region), economysim.CommodityID(c))
		}
		bd := beliefs.Limits(int64(ref), cost, h.BeliefParams())
		want := economysim.CommodityView{
			Commodity: economysim.CommodityID(c), Stock: s.stock[c], Ref: ref, Tool: p.isTool(name), UnitCost: economysim.Price(cost),
			Belief: s.beliefs[c], Bounds: economysim.Bounds{Floor: economysim.Price(bd.Floor), Ceiling: economysim.Price(bd.Ceiling)},
			Market: s.reports[c],
		}
		if prices := s.window[c]; len(prices) > 0 {
			lo, hi := slices.Min(prices), slices.Max(prices)
			want.Window = economysim.PriceWindowView{Mean: meanHalfEven(prices), Low: lo, High: hi, Position: 5000, HasWindow: true}
			if hi > lo {
				want.Window.Position = min(max((s.reports[c].Price-lo)*10_000/(hi-lo), 0), 10_000)
			}
		}
		if cv != want {
			t.Errorf("%s: commodity %s =\n%+v\nwant\n%+v", where, name, cv, want)
		}
	}
}

func checkRecipes(t *testing.T, h *economysim.ViewHarness, p *packIndex, tab *economysim.YieldTable, s *state, v *economysim.AgentView, where string) {
	t.Helper()
	cfg := h.Cfg
	role := cfg.Roles[s.roleID]
	var ids []int
	for _, rn := range role.Recipes {
		ids = append(ids, p.recipe[rn])
	}
	slices.Sort(ids)
	if len(v.Recipes) != len(ids) {
		t.Fatalf("%s: %d recipes, want the role's %d", where, len(v.Recipes), len(ids))
	}
	for k, id := range ids {
		r, cr := v.Recipes[k], cfg.Recipes[id]
		site, _ := economysim.YieldSite(tab, economysim.RegionID(s.region), economysim.RecipeID(id), h.Season(), cr.Sited)
		k0, reason := h.MaxExecutions(t, s.agent, cr.ID)
		if int(r.Recipe) != id || r.Duration != cr.Duration || r.Effective != site || r.MaxExecutions != k0 || r.Reason != reason {
			t.Errorf("%s: recipe %s = %+v, want duration %d, yield %d, executions %d (%q)", where, cr.ID, r, cr.Duration, site, k0, reason)
		}
		var in []economysim.RecipeLine
		for _, it := range cr.Inputs {
			in = append(in, economysim.RecipeLine{Commodity: economysim.CommodityID(p.commodity[it.Commodity]), Quantity: int64(it.Quantity)})
		}
		var out []economysim.RecipeOutput
		for _, o := range cr.Outputs {
			out = append(out, economysim.RecipeOutput{Commodity: economysim.CommodityID(p.commodity[o.Commodity]), Quantity: int64(o.Quantity), CostShare: o.CostShare, Discard: o.Discard})
		}
		var tools []economysim.RecipeTool
		for _, tl := range cr.Requires {
			c := p.commodity[tl.Commodity]
			tools = append(tools, economysim.RecipeTool{Commodity: economysim.CommodityID(c), WearPerUse: tl.WearPerUse, Durability: cfg.Commodities[c].Durability})
		}
		sort.SliceStable(in, func(i, j int) bool { return in[i].Commodity < in[j].Commodity })
		sort.SliceStable(out, func(i, j int) bool { return out[i].Commodity < out[j].Commodity })
		sort.SliceStable(tools, func(i, j int) bool { return tools[i].Commodity < tools[j].Commodity })
		if !slices.Equal(r.Inputs, in) || !slices.Equal(r.Outputs, out) || !slices.Equal(r.Tools, tools) {
			t.Errorf("%s: recipe %s lines = %v %v %v, want %v %v %v", where, cr.ID, r.Inputs, r.Outputs, r.Tools, in, out, tools)
		}
	}
}
