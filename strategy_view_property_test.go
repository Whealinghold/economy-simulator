package economysim_test

import (
	"fmt"
	"maps"
	"math/rand/v2"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/internal/household"
)

// forEachRoleAndRegion calls fn for every role and region of the pack.
func forEachRoleAndRegion(h *economysim.ViewHarness, fn func(role, region int)) {
	for ro := range h.Cfg.Roles {
		for g := range h.Cfg.Regions {
			fn(ro, g)
		}
	}
}

// TestDefaultStrategyTrade_NeverRejectedByTheRealAdmission verifies over 200
// seeds, both packs, every role and region and states drawn by a seeded
// generator (money from none to many ticks of upkeep, stocks from none to above
// the targets, beliefs anywhere in their bounds, prices anywhere, hunger, jobs
// in progress, known and unknown wages) that the plan of DefaultStrategy.Trade,
// converted with orderOf and admitted in plan order into the real book over the
// real ledger with the facts the world builds, has no order rejected, for any
// reason (insufficient_funds, insufficient_stock, duplicate_order,
// not_tradable, overflow, holding_limit, receipt_limit or another), and that the
// money the admission earmarks never exceeds the balance. The orders are in
// canonical commodity order (D86), so this is the evidence that the order of
// submission rejects nothing. The failing seed is in the message.
//
// Requirement: FR-AGT-044
func TestDefaultStrategyTrade_NeverRejectedByTheRealAdmission(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	t.Parallel()
	for _, pk := range viewPacks {
		t.Run(pk.name, func(t *testing.T) {
			t.Parallel()
			h := loadViewPack(t, pk.layers)
			p := newPackIndex(h.Cfg)
			orders := 0
			for seed := range uint64(200) {
				rng := rand.New(rand.NewPCG(seed, 99))
				forEachRoleAndRegion(h, func(ro, g int) {
					s := sampleState(t, h, p, rng, ro, g)
					v, err := h.Fill(s.agent)
					if err != nil {
						t.Fatalf("seed %d: fillCore error = %v, want nil", seed, err)
					}
					plan := h.Plan()
					economysim.DefaultStrategy{}.Trade(v, rng, plan)
					if !slices.IsSortedFunc(plan.Intents, func(a, b economysim.Intent) int { return int(a.Commodity) - int(b.Commodity) }) {
						t.Errorf("seed %d, role %s: intents %v are not in canonical commodity order", seed, h.Cfg.Roles[ro].ID, plan.Intents)
					}
					for _, r := range h.Admit(t, s.agent, plan) {
						orders++
						if !r.OK {
							t.Errorf("seed %d, pack %s, role %s, region %s, weight %d, money %d: order %+v rejected %q",
								seed, pk.name, h.Cfg.Roles[ro].ID, h.Cfg.Regions[g].ID, s.weight, s.money, r.Order, r.Reason)
						}
					}
					if bal, res := h.Balance(t, s.agent); res > bal {
						t.Errorf("seed %d, role %s: reserved %d exceeds the balance %d", seed, h.Cfg.Roles[ro].ID, res, bal)
					}
				})
			}
			if orders == 0 {
				t.Fatalf("no order was planned in 200 seeds, want some for the property to mean something")
			}
		})
	}
}

// TestDefaultStrategyWork_AcceptedByProduction verifies over the same states
// that the job the default requests is on the role's list, is never refused by
// the count (it is for at least one execution and at most the count k0 allows),
// and starts: a request above k0 is cut to k0, not refused. A role with no
// recipe requests nothing, and any request it made would be refused
// not_on_role by the first check of production.
//
// Requirement: FR-AGT-010, FR-AGT-028, FR-AGT-074
func TestDefaultStrategyWork_AcceptedByProduction(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	t.Parallel()
	for _, pk := range viewPacks {
		t.Run(pk.name, func(t *testing.T) {
			t.Parallel()
			h := loadViewPack(t, pk.layers)
			p := newPackIndex(h.Cfg)
			requests := 0
			for seed := range uint64(60) {
				rng := rand.New(rand.NewPCG(seed, 5))
				forEachRoleAndRegion(h, func(ro, g int) {
					s := sampleState(t, h, p, rng, ro, g)
					if s.wantJob {
						return // Work is called only for an agent with no job
					}
					role := h.Cfg.Roles[ro]
					v, err := h.Fill(s.agent)
					if err != nil {
						t.Fatalf("seed %d: fillCore error = %v, want nil", seed, err)
					}
					req := economysim.DefaultStrategy{}.Work(v, rng)
					if len(role.Recipes) == 0 {
						if !req.NoJob {
							t.Errorf("seed %d, role %s: a role with no recipe requested %+v, want none", seed, role.ID, req)
						}
						if r := h.RequestAdmission(s.agent, 0); r != economysim.ReasonNotOnRole {
							t.Errorf("role %s: admission of recipe 0 = %q, want %q", role.ID, r, economysim.ReasonNotOnRole)
						}
						return
					}
					if req.NoJob {
						return
					}
					requests++
					if r := h.RequestAdmission(s.agent, req.Recipe); r != "" {
						t.Fatalf("seed %d, role %s: request %+v refused %q on the role's own list", seed, role.ID, req, r)
					}
					name := h.Cfg.Recipes[req.Recipe].ID
					k0, _ := h.MaxExecutions(t, s.agent, name)
					if req.Executions < 1 || k0 < 1 || req.Executions > k0 {
						t.Errorf("seed %d, role %s: request %+v with k0 = %d, want 1 <= executions <= k0", seed, role.ID, req, k0)
					}
					if k, reason := h.Start(t, s.agent, req.Recipe, req.Executions+s.weight+1); k != k0 {
						t.Errorf("seed %d, role %s: a request above k0 started %d (%q), want it cut to k0 = %d", seed, role.ID, k, reason, k0)
					}
				})
			}
			if requests == 0 {
				t.Fatalf("no job was requested, want some for the test to mean something")
			}
		})
	}
}

// TestDefaultStrategyTrade_CohortInvariance verifies, on the real tables, the
// claims of spec-agents.md 9.1 inside their conditions (FR-AGT-052): a cohort of
// weight 7 holds 7 times the stock and money of a household of weight 1, with
// the same beliefs, prices and drawn words. The conditions are asserted for
// every pair and a pair that breaks one fails: the money is ample (twice the
// money the plan can spend and twice every ceiling, so no limit is lowered by
// Afford and no tier is money-bound), and the planned recipe is the same at
// both weights, or the pair is a state at the halt threshold, which is counted
// and not compared. Then the commodities, sides, limits and drawn tiers agree:
// tiers 1, 2 and 3 before the buy share, asks and reserves are exactly 7 times
// as large, a Target from the use target is exactly 7 times as large, a Target
// from the output target is within its band and, for a role with one recipe of
// one output, ceil(7*X) (checkOutputTarget), the quantity bought equals, at
// each weight, the closed form of the allocation on the totals (tiers 1 and 2
// in full, the spend cap of the money left, tier 3 at the buy share), a bid is
// within [7*q1, 7*q1 + 6], and the executions of a recipe with no tool equal
// min(weight, floor(stock/q) of its inputs).
//
// X is q*E*OutputCoverTicks/(10 000*Duration) for the output o of a role's one
// recipe; for the packs' recipes of duration 3 and output 1 it is an integer, so
// the target is exactly 7 times as large, and for the others the band
// [7*q1 - 6, 7*q1] holds.
//
// Requirement: FR-AGT-052
func TestDefaultStrategyTrade_CohortInvariance(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	t.Parallel()
	const w = 7
	for _, pk := range viewPacks {
		t.Run(pk.name, func(t *testing.T) {
			t.Parallel()
			h := loadViewPack(t, pk.layers)
			p := newPackIndex(h.Cfg)
			compared, skipped, targets := 0, 0, 0
			for seed := range uint64(60) {
				forEachRoleAndRegion(h, func(ro, g int) {
					rng := rand.New(rand.NewPCG(seed, 3))
					one, seven := cohortPair(t, h, p, rng, ro, g, w)
					a, b := tradeOnce(t, h, one, seed), tradeOnce(t, h, seven, seed)
					where := fmt.Sprintf("seed %d, pack %s, role %s, region %s", seed, pk.name, h.Cfg.Roles[ro].ID, h.Cfg.Regions[g].ID)
					if a.rstar != b.rstar {
						skipped++
						return
					}
					compared++
					checkCohortPair(t, a, b, w, where)
					targets += checkOutputTarget(t, h, p, ro, g, a.intents, b.intents, where)
				})
			}
			if targets == 0 {
				t.Errorf("no output target was checked, want some for the closed form to be tested")
			}
			if compared == 0 || compared < skipped {
				t.Errorf("compared %d pairs and skipped %d at the halt threshold, want most pairs inside the conditions", compared, skipped)
			}
		})
	}
}

// checkCohortPair compares the runs of a household and of a cohort of weight w
// of one state, which have the same planned recipe.
func checkCohortPair(t *testing.T, a, b *cohortRun, w int64, where string) {
	t.Helper()
	for _, r := range []*cohortRun{a, b} {
		if r.money < 2*max(r.spend(), r.ceiling) {
			t.Errorf("%s: money %d is not ample for a spend of %d and a ceiling of %d, want twice as much", where, r.money, r.spend(), r.ceiling)
		}
		if got, want := r.quantities(), r.expected(); !maps.Equal(got, want) {
			t.Errorf("%s: bids at weight %d = %v, want the closed form %v", where, r.weight, got, want)
		}
		for _, x := range r.intents {
			if x.Side == economysim.Bid && int64(x.Limit) != r.rowOf(x.Commodity).Limit {
				t.Errorf("%s: limit of %+v is %d, want the drawn %d (a limit lowered by Afford)", where, x, x.Limit, r.rowOf(x.Commodity).Limit)
			}
		}
	}
	if len(a.rows) != len(b.rows) {
		t.Errorf("%s: %d candidates at weight 1 and %d at weight %d, want the same", where, len(a.rows), len(b.rows), w)
		return
	}
	for k, x := range a.rows {
		y := b.rows[k]
		if y.Commodity != x.Commodity || y.Bid != x.Bid || y.Limit != x.Limit || y.UseSourced != x.UseSourced ||
			y.T1 != w*x.T1 || y.T2 != w*x.T2 || y.T3 != w*x.T3 || y.Ask != w*x.Ask {
			t.Errorf("%s: row %+v at weight 1 and %+v at weight %d, want the same side and limit and tiers and asks %d times as large", where, x, y, w, w)
		}
	}
	for _, row := range a.rows {
		x, okx := a.intentOf(row.Commodity)
		y, oky := b.intentOf(row.Commodity)
		switch {
		case !row.Bid && (!okx || !oky):
			t.Errorf("%s: ask for %d present at weight 1 %v and at weight %d %v, want both", where, row.Commodity, okx, w, oky)
		case row.Bid && (okx && !oky || oky && int64(y.Quantity) < w*int64(x.Quantity) || oky && int64(y.Quantity) > w*int64(x.Quantity)+w-1):
			// A bid that is 0 at weight 1 (absent) is 0 to w-1 at weight w: the floor of tier 3.
			t.Errorf("%s: bid for %d is %+v at weight 1 and %+v at weight %d, want q within [w*q1, w*q1+w-1]", where, row.Commodity, x, y, w)
		}
		if !okx || !oky {
			continue
		}
		switch {
		case x.Limit != y.Limit:
			t.Errorf("%s: limit of %+v = %d at weight %d, want %d", where, x, y.Limit, w, x.Limit)
		case int64(y.Reserve) != w*int64(x.Reserve):
			t.Errorf("%s: reserve of %+v = %d at weight %d, want %d", where, x, y.Reserve, w, w*int64(x.Reserve))
		case x.Side == economysim.Ask && int64(y.Quantity) != w*int64(x.Quantity):
			t.Errorf("%s: ask quantity %d at weight %d, want %d", where, y.Quantity, w, w*int64(x.Quantity))
		case row.UseSourced && int64(y.Target) != w*int64(x.Target):
			t.Errorf("%s: target %d at weight %d, want exactly %d (from the use target)", where, y.Target, w, w*int64(x.Target))
		case !row.UseSourced && (int64(y.Target) > w*int64(x.Target) || int64(y.Target) < w*int64(x.Target)-(w-1)):
			t.Errorf("%s: target %d at weight %d, want within [%d, %d] (from the output target)", where, y.Target, w, w*int64(x.Target)-(w-1), w*int64(x.Target))
		}
	}
	for _, r := range []*cohortRun{a, b} {
		for _, e := range r.execs {
			if e.tools == 0 && e.k0 != e.want {
				t.Errorf("%s: executions of a recipe with %d inputs at weight %d = %d, want min(weight, stock/quantity) = %d", where, e.inputs, r.weight, e.k0, e.want)
			}
		}
	}
}

// cohortRun is what a test needs of one agent's trade: the intents, the first
// pass of the strategy, and the state the closed forms read.
type cohortRun struct {
	weight, money, cost, ceiling int64
	rstar                        int
	intents                      []economysim.Intent
	rows                         []economysim.TradeRow
	par                          household.Params
	execs                        []execCheck
}

// execCheck is the executions the view gives for a recipe and the closed form.
type execCheck struct {
	inputs, tools int
	k0, want      int64
}

func (r *cohortRun) intentOf(c economysim.CommodityID) (economysim.Intent, bool) {
	for _, x := range r.intents {
		if x.Commodity == c {
			return x, true
		}
	}
	return economysim.Intent{}, false
}

func (r *cohortRun) rowOf(c economysim.CommodityID) economysim.TradeRow {
	for _, x := range r.rows {
		if x.Commodity == c {
			return x
		}
	}
	return economysim.TradeRow{}
}

// spend is the money the bids of the plan reserve.
func (r *cohortRun) spend() (sum int64) {
	for _, x := range r.intents {
		if x.Side == economysim.Bid {
			sum += int64(x.Limit) * int64(x.Quantity)
		}
	}
	return sum
}

// quantities is the quantity of each bid of the plan.
func (r *cohortRun) quantities() map[economysim.CommodityID]int64 {
	m := map[economysim.CommodityID]int64{}
	for _, x := range r.intents {
		if x.Side == economysim.Bid {
			m[x.Commodity] = int64(x.Quantity)
		}
	}
	return m
}

// expected is the allocation of the money by its closed form on the totals
// (spec-agents.md 7.4.3, 7.5.3): tiers 1 and 2 of every bid in full at the drawn
// limit, then the spend cap of the money left over the savings, then tier 3 of
// each bid in order at the buy share, bounded by the money and the cap.
func (r *cohortRun) expected() map[economysim.CommodityID]int64 {
	m := map[economysim.CommodityID]int64{}
	avail := r.money
	for _, x := range r.rows {
		if x.Bid {
			avail -= x.Limit * (x.T1 + x.T2)
			m[x.Commodity] = x.T1 + x.T2
		}
	}
	savings, _ := household.ReserveMoney(r.cost, r.par.ReserveTicks)
	spendCap, _ := household.SpendCap(household.Excess(avail, savings), r.cost, r.par)
	for _, x := range r.rows {
		if !x.Bid {
			continue
		}
		want := x.T3 * household.BuyShare(x.WindowPos, r.par) / household.Basis
		if q := min(want, min(avail, spendCap)/x.Limit); q >= 1 {
			m[x.Commodity] += q
			avail -= q * x.Limit
			spendCap -= q * x.Limit
		}
	}
	for c, q := range m {
		if q < 1 {
			delete(m, c)
		}
	}
	return m
}

// cohortPair builds two agents of the role and region from one drawn state:
// weight 1 and weight w, the second with w times the stock and money, with equal
// beliefs, and returns their indexes. The money is ample, 10^6 per household of
// the larger.
func cohortPair(t *testing.T, h *economysim.ViewHarness, p *packIndex, rng *rand.Rand, ro, g int, w int64) (one, many int) {
	t.Helper()
	cfg := h.Cfg
	role, region := cfg.Roles[ro], cfg.Regions[g]
	rr := economysim.RoleRow(&p.roles, economysim.RoleID(ro))
	one, many = h.Spawn(t, role.ID, region.ID, 1), h.Spawn(t, role.ID, region.ID, w)
	h.Fund(t, one, 1_000_000)
	h.Fund(t, many, 1_000_000*w)
	for _, c := range rr.Tradable {
		name := cfg.Commodities[c].ID
		if n := int64(rng.IntN(60)); n > 0 {
			h.Give(t, one, name, n)
			h.Give(t, many, name, w*n)
		}
		ref := int64(cfg.Commodities[c].ReferencePrice)
		h.SetReport(t, region.ID, name, economysim.MarketReport{Price: 1 + rng.Int64N(3*ref), Age: int64(rng.IntN(5))})
		h.ResetWindow(t, region.ID, name)
		for range rng.IntN(3) {
			h.PushWindow(t, region.ID, name, 1+rng.Int64N(3*ref))
		}
	}
	h.Anchors(int64(rng.IntN(100)))
	v, err := h.Fill(one)
	if err != nil {
		t.Fatalf("fillCore error = %v, want nil", err)
	}
	for k := range rr.Tradable {
		bd, bt := v.Commodities[k].Bounds, v.Tunables.Beliefs
		centre := bd.Floor + economysim.Price(rng.Int64N(int64(bd.Ceiling-bd.Floor)+1))
		width := bt.WidthMin + rng.Int64N(bt.WidthMax-bt.WidthMin+1)
		h.SetBelief(one, k, int64(centre), width)
		h.SetBelief(many, k, int64(centre), width)
	}
	return one, many
}

// tradeOnce fills the view of an agent and returns the intents of the default
// strategy for a stream that depends on the seed only, so that the two cohorts
// see the same words, with the first pass of the strategy and the executions of
// the view.
func tradeOnce(t *testing.T, h *economysim.ViewHarness, agent int, seed uint64) *cohortRun {
	t.Helper()
	v, err := h.Fill(agent)
	if err != nil {
		t.Fatalf("fillCore error = %v, want nil", err)
	}
	r := &cohortRun{weight: v.Weight, money: v.Money}
	r.rows, r.rstar, r.cost, r.par = h.TradeFirstPass(v, seed)
	for _, c := range v.Commodities {
		r.ceiling = max(r.ceiling, int64(c.Bounds.Ceiling))
	}
	for _, rv := range v.Recipes {
		e := execCheck{inputs: len(rv.Inputs), tools: len(rv.Tools), k0: rv.MaxExecutions, want: v.Weight}
		for _, in := range rv.Inputs {
			for _, c := range v.Commodities {
				if c.Commodity == in.Commodity {
					e.want = min(e.want, c.Stock/in.Quantity)
				}
			}
		}
		r.execs = append(r.execs, e)
	}
	plan := h.Plan()
	economysim.DefaultStrategy{}.Trade(v, rand.New(rand.NewPCG(seed, 1234)), plan)
	r.intents = slices.Clone(plan.Intents)
	return r
}

// checkOutputTarget checks the closed form of a Target that comes from the
// output target: for a role with one recipe of one non-discard output o, when o
// is neither an upkeep commodity of the region nor an input or a tool of the
// recipe, the Target of the intent for o is ceil(w*X) with
// X = q*E*OutputCoverTicks/(10 000*Duration), at both weights.
func checkOutputTarget(t *testing.T, h *economysim.ViewHarness, p *packIndex, ro, g int, one, many []economysim.Intent, where string) (hits int) {
	t.Helper()
	cfg := h.Cfg
	role := cfg.Roles[ro]
	if len(role.Recipes) != 1 {
		return 0
	}
	rec := cfg.Recipes[p.recipe[role.Recipes[0]]]
	var outs []economysim.Output
	for _, o := range rec.Outputs {
		if !o.Discard {
			outs = append(outs, o)
		}
	}
	if len(outs) != 1 {
		return 0
	}
	o := outs[0]
	c := p.commodity[o.Commodity]
	for _, u := range p.upkeep(role, cfg.Regions[g]) {
		if int(u.Commodity) == c {
			return 0
		}
	}
	for _, it := range rec.Inputs {
		if it.Commodity == o.Commodity {
			return 0
		}
	}
	for _, tl := range rec.Requires {
		if tl.Commodity == o.Commodity {
			return 0
		}
	}
	site, _ := economysim.YieldSite(ptr(economysim.NewYieldTable(cfg)), economysim.RegionID(g), economysim.RecipeID(p.recipe[rec.ID]), h.Season(), rec.Sited)
	cover := cfg.World.OutputCoverTicks
	for i, in := range [][]economysim.Intent{one, many} {
		w := []int64{1, 7}[i]
		for _, x := range in {
			// A Target of 0 is a state in which the recipe is not the planned one
			// (not worth running, or halted), which has no output target.
			if int(x.Commodity) != c || x.Target == 0 {
				continue
			}
			hits++
			// ceil(w * q * E * cover / (10 000 * Duration)).
			num, den := w*int64(o.Quantity)*site*cover, 10_000*rec.Duration
			if want := (num + den - 1) / den; int64(x.Target) != want {
				t.Errorf("%s: target of %s at weight %d = %d, want ceil(%d/%d) = %d", where, o.Commodity, w, x.Target, num, den, want)
			}
		}
	}
	return hits
}

func ptr[T any](v T) *T { return &v }
