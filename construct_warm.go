package economysim

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"slices"

	"github.com/Whealinghold/economy-sim/internal/checked"
	"github.com/Whealinghold/economy-sim/internal/ledger"
	"github.com/Whealinghold/economy-sim/internal/logkeys"
)

// This file is the world's side of the warm start (spec-world.md 5.6,
// spec-viability.md 8; plan-phase-1-11.md Phase 3 deliverable 3, D191). It is
// the one file of the world's construction path that reads Config.WarmStart,
// apart from the validator, the solver and the wage helper of welfare.go
// (FR-VIA-047): no rule after tick 0 depends on the table. Nothing here draws a
// random number or calls the solver (FR-VIA-021, 040).
//
// The pieces, in the order of spec-world.md 5.6.1: the pure helpers (3a:
// apportion, phase, pickOffset, staggerKey), the seed prices (3b: warmPriceTable),
// the wages (3c: keepWages), the money and stock of a group with the stagger (3d:
// fundWarm), the jobs (3e: placeJobs) and their books (3f: grantInputs and
// takeInputs, kept apart so a test can stop between them), the pools (3g:
// seedPools) and the line (3h: logWarm). warm runs the pieces after the agents
// exist and before the creation draws.

// apportion shares n agents among the entries of a group's jobs by one joint
// largest-remainder apportionment (spec-viability.md 8.5.2, FR-VIA-034): the
// quotas are share * n / 10 000 for each entry in the given order and, last, the
// idle share 10 000 minus the shares; the floors are taken first and each agent
// left goes to the largest remainder, a tie to the earlier entry (the idle
// share is the last, so it loses every tie; architecture.md 7.6). It returns
// the busy count of each entry, so every count is within one of its quota and
// the sum never exceeds n. A share outside [0, 10 000], shares summing above
// 10 000 or a negative n is ErrOutOfBounds, and a product that does not fit is
// ErrOverflow, never a wrapped value. It takes no random number.
func apportion(shares []int64, n int64) ([]int64, error) {
	sum, err := checkShares(shares, n)
	if err != nil {
		return nil, err
	}
	k := len(shares)
	counts, rems := make([]int64, k+1), make([]int64, k+1)
	var given int64
	for i := range counts {
		share := wholeBP - sum // the idle share, last
		if i < k {
			share = shares[i]
		}
		q, r, ok := checked.MulDivRem(share, n, wholeBP)
		if !ok {
			// Excluded from coverage (testing.md 9.5): a share is at most 10 000, so
			// the quotient is at most n and always fits.
			return nil, fmt.Errorf("apportion: quota of share %d over %d agents: %w", share, n, ErrOverflow)
		}
		counts[i], rems[i] = q, r
		given += q
	}
	order := make([]int, k+1)
	for i := range order {
		order[i] = i
	}
	// The remainders and the indexes make a total order, so the sort is stable
	// by construction: a larger remainder first, then the earlier entry.
	slices.SortFunc(order, func(a, b int) int { return cmp.Or(cmp.Compare(rems[b], rems[a]), cmp.Compare(a, b)) })
	for _, i := range order[:min(n-given, int64(k+1))] {
		counts[i]++
	}
	return counts[:k], nil
}

// checkShares returns the sum of the shares, or ErrOutOfBounds for a share
// outside [0, 10 000], a sum above 10 000 or a negative n.
func checkShares(shares []int64, n int64) (int64, error) {
	var sum int64
	for _, s := range shares {
		if s < 0 || s > wholeBP {
			return 0, fmt.Errorf("apportion: share %d outside [0, %d]: %w", s, wholeBP, ErrOutOfBounds)
		}
		sum += s // at most len(shares) * 10 000: a count of configured jobs
	}
	if sum > wholeBP || n < 0 {
		return 0, fmt.Errorf("apportion: shares sum to %d over %d agents, want at most %d and a non-negative count: %w", sum, n, wholeBP, ErrOutOfBounds)
	}
	return sum, nil
}

// phase is the stagger rule of spec-viability.md 8.4.3 (FR-VIA-037): agent i
// (0-based, in agent order) of n gets the phase floor(i * p / n) of a period p, a
// job's duration, a tool's durability or a lot's shelf life. It is in [0, p), is
// 0 for agent 0 and does not decrease with i. An i outside [0, n), an n below 1
// or a p below 0 is ErrOutOfBounds, and a product that does not fit is
// ErrOverflow. It takes no random number.
func phase(i, n, p int64) (int64, error) {
	if i < 0 || i >= n || p < 0 {
		return 0, fmt.Errorf("phase(%d, %d, %d): want 0 <= i < n and p >= 0: %w", i, n, p, ErrOutOfBounds)
	}
	q, ok := checked.MulDivFloor(i, p, n)
	if !ok {
		// Excluded from coverage (testing.md 9.5): i < n makes the quotient less than p.
		return 0, fmt.Errorf("phase(%d, %d, %d): %w", i, n, p, ErrOverflow)
	}
	return q, nil
}

// pickOffset returns the virtual start offset of the jth of busy agents of a
// spec (spec-viability.md 8.5.2, FR-VIA-034): V[floor(j * |V| / busy)] of the
// ascending set v, which is an index inside v. An empty v is ErrWarmStartJob (a
// table with one cannot pass Validate), and a j outside [0, busy) is
// ErrOutOfBounds.
func pickOffset(v []int64, j, busy int64) (int64, error) {
	if len(v) == 0 {
		return 0, fmt.Errorf("no virtual start offset for the job: %w", ErrWarmStartJob)
	}
	if j < 0 || j >= busy {
		return 0, fmt.Errorf("pickOffset(%d, %d): want 0 <= j < busy: %w", j, busy, ErrOutOfBounds)
	}
	k, ok := checked.MulDivFloor(j, int64(len(v)), busy)
	if !ok {
		// Excluded from coverage (testing.md 9.5): j < busy makes the quotient less than |v|.
		return 0, fmt.Errorf("pickOffset(%d, %d): %w", j, busy, ErrOverflow)
	}
	return v[k], nil
}

// staggerKey is the stated lot key of the stock of agent i of n for a commodity
// (spec-viability.md 8.4.3, FR-VIA-038): a perishable lot is born at
// -phase(i, n, ShelfLife) and a durable item has the durability
// Durability - phase(i, n, Durability), both inside the ranges of
// spec-ledger.md 5.2.4; a commodity that neither spoils nor wears has the
// default key.
func staggerKey(c *Commodity, id CommodityID, i, n int64) (ledger.Key, error) {
	k := ledger.Key{Commodity: int(id)}
	if c.ShelfLife > 0 {
		p, err := phase(i, n, c.ShelfLife)
		if err != nil {
			return ledger.Key{}, err
		}
		k.Born = -p
	}
	if c.Durability > 0 {
		p, err := phase(i, n, c.Durability)
		if err != nil {
			return ledger.Key{}, err
		}
		k.Durability = c.Durability - p
	}
	return k, nil
}

// warmGroupOf returns the entry of the warm start for the group (region, role),
// or nil when the table has none or there is no table (FR-VIA-042). It points
// into the configuration, which is never written after NewWorld.
func warmGroupOf(c *Config, region, role string) *WarmGroup {
	if c.WarmStart == nil {
		return nil
	}
	for i := range c.WarmStart.Groups {
		if g := &c.WarmStart.Groups[i]; g.Region == region && g.Role == role {
			return g
		}
	}
	return nil
}

// fundWarm gives agent i of group p, which has an entry in the warm start, the
// entry's money and stock in place of StartMoney and StartStock, by one
// initial creation (spec-world.md 5.6.1 item 3, FR-VIA-032): the money times the
// weight, and each stock item times the weight at the stated lot key of the
// stagger (staggerKey). It returns the money. A validated table cannot make the
// ledger refuse a step, so a refusal is a violated invariant.
func (b *builder) fundWarm(a *agent, p *groupPlan, i int) Money {
	w := b.w
	money, ok := checked.Mul(a.weight, int64(p.warm.Money))
	if !ok {
		invariant("the warm money of a group overflows after validation")
	}
	mustBuild(fundInitial(w.ledger, a.account, Money(money), nil), "the warm money of an agent")
	for _, it := range p.warm.Stock {
		id, ok := w.cfg.commodityID(it.Commodity)
		q, qok := checked.Mul(a.weight, int64(it.Quantity))
		if !ok || !qok {
			invariant("the warm stock of a group names an unknown commodity or overflows after validation")
		}
		key, err := staggerKey(&w.cfg.Commodities[id], id, int64(i), int64(len(p.weights)))
		mustBuild(err, "the stagger of a warm stock")
		from, ok := w.ledger.Boundary(ledger.Initial, int(id))
		if !ok {
			invariant("the boundary account of initial goods is not open")
		}
		mustBuild(w.ledger.MoveStated(from, a.account, key, q, ledger.Initial), "the warm stock of an agent")
	}
	return Money(money)
}

// warm applies the rest of the warm start after the agents exist and before the
// creation draws (spec-world.md 5.6.1, items 2, 4 and 5; items 1 and 3 are the
// seed prices of allocate and fundWarm). It takes no random number. A job whose
// tools the group's stock does not hold is an error, ErrWarmStartJob, since
// Validate does not look at the stock.
func (b *builder) warm() error {
	b.keepWages()
	// Excluded from coverage (testing.md 9.5): the error is a ledger or overflow
	// refusal of the world's own accounts, which a valid configuration cannot cause
	// and no test can inject from the root (D190); the branch only wraps and returns
	// it.
	if err := b.placeJobs(); err != nil {
		return err
	}
	// Excluded from coverage (testing.md 9.5): the error is a ledger or overflow
	// refusal of the world's own accounts, which a valid configuration cannot cause
	// and no test can inject from the root (D190); the branch only wraps and returns
	// it.
	if err := b.grantInputs(); err != nil {
		return err
	}
	if err := b.takeInputs(); err != nil {
		return err
	}
	b.seedPools()
	return nil
}

// keepWages keeps the starting wage of each region for the cost anchor (3c,
// FR-VIA-052). The choice between it and the pooled wage is made at every read
// of the anchor and never latched.
func (b *builder) keepWages() { b.w.seedWage = seedWages(&b.w.cfg) }

// placeJobs gives the jobs in progress of every group with an entry (3e).
func (b *builder) placeJobs() error {
	for pi := range b.plans {
		if p := &b.plans[pi]; p.warm != nil && len(p.warm.Jobs) > 0 {
			// Excluded from coverage (testing.md 9.5): the error is a ledger or overflow
			// refusal of the world's own accounts, which a valid configuration cannot cause
			// and no test can inject from the root (D190); the branch only wraps and returns
			// it.
			if err := b.placeGroupJobs(p); err != nil {
				return err
			}
		}
	}
	return nil
}

// placeGroupJobs apportions the agents of the group among its JobSpecs and gives
// the busy agents their jobs, spec by spec in agent order (spec-viability.md
// 8.5.2, FR-VIA-033 to 035). It touches no ledger.
func (b *builder) placeGroupJobs(p *groupPlan) error {
	w := b.w
	shares := make([]int64, len(p.warm.Jobs))
	for i, j := range p.warm.Jobs {
		shares[i] = j.InProgressBP
	}
	busy, err := apportion(shares, int64(len(p.weights)))
	// Excluded from coverage (testing.md 9.5): the error is a ledger or overflow
	// refusal of the world's own accounts, which a valid configuration cannot cause
	// and no test can inject from the root (D190); the branch only wraps and returns
	// it.
	if err != nil {
		return fmt.Errorf("warm start group %s/%s: %w", p.warm.Region, p.warm.Role, err)
	}
	next := p.first
	for si, js := range p.warm.Jobs {
		recipe, ok := w.cfg.recipeID(js.Recipe)
		if !ok {
			invariant("a warm-start job names an unknown recipe after validation")
		}
		v := virtualStartOffsets(&w.cfg, &w.yields, p.region, recipe)
		for j := range busy[si] {
			// Excluded from coverage (testing.md 9.5): the error is a ledger or overflow
			// refusal of the world's own accounts, which a valid configuration cannot cause
			// and no test can inject from the root (D190); the branch only wraps and returns
			// it.
			if err := b.startWarmJob(&w.agents[next], p.region, recipe, v, [2]int64{j, busy[si]}); err != nil {
				return fmt.Errorf("warm start group %s/%s: %w", p.warm.Region, p.warm.Role, err)
			}
			next++
		}
	}
	return nil
}

// startWarmJob makes the job of agent a, the jth of busy (jb): Remaining is
// Duration - o for the offset o of pickOffset, the factor is that of the season
// of the virtual tick -o (Calendar.SeasonAt and no other rule, FR-VIA-053), the
// executions are the agent's weight and the principal is the agent
// (FR-VIA-035). The pending output has the residue 0 (spec-world.md 5.6.2).
func (b *builder) startWarmJob(a *agent, region RegionID, recipe RecipeID, v []int64, jb [2]int64) error {
	w := b.w
	o, err := pickOffset(v, jb[0], jb[1])
	// Excluded from coverage (testing.md 9.5): the error is a ledger or overflow
	// refusal of the world's own accounts, which a valid configuration cannot cause
	// and no test can inject from the root (D190); the branch only wraps and returns
	// it.
	if err != nil {
		return err
	}
	remaining := w.cfg.Recipes[recipe].Duration - o
	return w.prod.warmJob(&a.job, &w.yields, region, w.cfg.Calendar.SeasonAt(-o), recipe, a.id, a.weight, remaining)
}

// grantInputs grants every warm job the inputs of its executions by initial, as
// lots at the default key (3f, spec-viability.md 8.5.3). takeInputs takes them.
// They are two functions so that a test can look at the stagger before the
// oldest-first take of the Transform (O11).
func (b *builder) grantInputs() error {
	w := b.w
	for i := range w.agents {
		a := &w.agents[i]
		if !a.hasJob() {
			continue
		}
		// Excluded from coverage (testing.md 9.5): the error is a ledger or overflow
		// refusal of the world's own accounts, which a valid configuration cannot cause
		// and no test can inject from the root (D190); the branch only wraps and returns
		// it.
		if err := w.prod.formInputs(&a.job, w.recipes.row(a.job.recipe), a.job.executions, a.job.recipe); err != nil {
			return err
		}
		for _, ln := range w.prod.inputs {
			from, ok := w.ledger.Boundary(ledger.Initial, ln.Commodity)
			if !ok {
				invariant("the boundary account of initial goods is not open")
			}
			mustBuild(w.ledger.Move(from, a.account, ln.Commodity, ln.Quantity, ledger.Initial), "the grant of the inputs of a warm job")
		}
	}
	return nil
}

// takeInputs takes the granted inputs of every warm job by recipe_input in one
// construction Transform, which consumes the oldest lots first, and wears the
// tools of its executions (3f, FR-VIA-036), all in the construction phase, so
// the moves are in the bucket and the audit passes at tick 0 (FR-LED-046).
func (b *builder) takeInputs() error {
	w := b.w
	for i := range w.agents {
		a := &w.agents[i]
		if !a.hasJob() {
			continue
		}
		row := w.recipes.row(a.job.recipe)
		if err := b.checkTools(a, row); err != nil {
			return err
		}
		// Excluded from coverage (testing.md 9.5): the error is a ledger or overflow
		// refusal of the world's own accounts, which a valid configuration cannot cause
		// and no test can inject from the root (D190); the branch only wraps and returns
		// it.
		if err := w.prod.formInputs(&a.job, row, a.job.executions, a.job.recipe); err != nil {
			return err
		}
		if len(w.prod.inputs) > 0 {
			mustBuild(w.ledger.Transform(a.account, w.prod.inputs, nil), "the take of the inputs of a warm job")
		}
		w.prod.wearTools(w.ledger, row, a.job.executions, a.account)
	}
	return nil
}

// checkTools reports ErrWarmStartJob when the agent holds fewer items of a tool
// of its job than the job has executions, which the wear would refuse.
func (b *builder) checkTools(a *agent, row *recipeRow) error {
	for _, t := range row.tools {
		held, reserved, err := b.w.ledger.Holding(a.account, int(t.commodity))
		mustBuild(err, "the reading of a tool stock")
		if held-reserved < a.job.executions {
			return fmt.Errorf("agent %d: job of recipe %d needs %d items of tool %d, the warm stock gives %d: %w",
				a.id, a.job.recipe, a.job.executions, t.commodity, held-reserved, ErrWarmStartJob)
		}
	}
	return nil
}

// seedPools creates each region's pool by initial into its estate account (3g,
// spec-viability.md 8.3.3, FR-EST-015); a region with no row has an empty pool.
func (b *builder) seedPools() {
	w := b.w
	if w.cfg.WarmStart == nil {
		return
	}
	for _, p := range w.cfg.WarmStart.Pools {
		r, ok := w.cfg.regionID(p.Region)
		if !ok {
			invariant("a warm-start pool names an unknown region after validation")
		}
		mustBuild(w.estates.seedPool(w.ledger, r, p.Money), "the seed of a pool")
	}
}

// logWarm is the one line the warm start writes (3h, FR-VIA-046): "warm start
// applied" at Info with the counts of the table, once, only with a warm start.
func (b *builder) logWarm(ctx context.Context) {
	w := b.w
	n, present := warmCountsOf(&w.cfg)
	if !present || !w.log.Enabled(ctx, slog.LevelInfo) {
		return
	}
	w.log.LogAttrs(ctx, slog.LevelInfo, "warm start applied",
		slog.Int(logkeys.Regions, len(w.cfg.Regions)), slog.Int64(logkeys.Prices, n.prices),
		slog.Int64(logkeys.Wages, n.wages), slog.Int64(logkeys.Groups, n.groups), slog.Int64(logkeys.Jobs, n.jobs))
}

// warmPriceTable returns the seed price of every market of a canonical
// configuration that passed Validate, region-major (index region * commodities
// + commodity), and 0 for a market with no entry (spec-beliefs.md 8.4,
// FR-VIA-031, 042). The region and commodity strings are mapped to IDs once.
func warmPriceTable(c *Config) []Price {
	t := make([]Price, len(c.Regions)*len(c.Commodities))
	if c.WarmStart == nil {
		return t
	}
	for _, p := range c.WarmStart.Prices {
		r, rok := c.regionID(p.Region)
		k, kok := c.commodityID(p.Commodity)
		if !rok || !kok {
			invariant("a warm-start price names an unknown region or commodity after validation")
		}
		t[int(r)*len(c.Commodities)+int(k)] = p.Price
	}
	return t
}

// warmCounts are the sizes of the warm-start table, the keys of the
// construction summary (observability-and-logging.md 3.3.2: prices, wages,
// groups, jobs).
type warmCounts struct {
	prices, wages, groups, jobs int64
}

// warmCountsOf returns the counts of the table and whether a warm start is
// present at all. A present table that is empty is a warm start with all counts
// 0 (spec-config.md 8.2.1).
func warmCountsOf(c *Config) (n warmCounts, present bool) {
	ws := c.WarmStart
	if ws == nil {
		return warmCounts{}, false
	}
	n = warmCounts{prices: int64(len(ws.Prices)), wages: int64(len(ws.Wages)), groups: int64(len(ws.Groups))}
	for i := range ws.Groups {
		n.jobs += int64(len(ws.Groups[i].Jobs))
	}
	return n, true
}
