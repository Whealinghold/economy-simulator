package economysim_test

import (
	"fmt"
	"log/slog"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/internal/ledger"
)

// warmPack is a pack with a warm start and the same pack without it. The layers
// are the ones the other pack tests load: the warm table is its own layer.
type warmPack struct {
	name         string
	layers, cold []string
}

var warmPacks = []warmPack{
	{"warm-one", []string{"base.toml", "one.toml", "warm-one.toml"}, []string{"base.toml", "one.toml"}},
	{"warm-second", []string{"second/second.toml", "second/warm-second.toml"}, []string{"second/second.toml"}},
	{"warm-two", []string{"base.toml", "two.toml", "warm-two.toml"}, []string{"base.toml", "two.toml"}},
}

// idxOf returns the index of the entity with the ID in a list of n.
func idxOf(n int, id string, name func(int) string) int {
	for i := range n {
		if name(i) == id {
			return i
		}
	}
	return -1
}

func regionIdx(cfg *economysim.Config, id string) int {
	return idxOf(len(cfg.Regions), id, func(i int) string { return cfg.Regions[i].ID })
}

func commodityIdx(cfg *economysim.Config, id string) int {
	return idxOf(len(cfg.Commodities), id, func(i int) string { return cfg.Commodities[i].ID })
}

func roleIdx(cfg *economysim.Config, id string) int {
	return idxOf(len(cfg.Roles), id, func(i int) string { return cfg.Roles[i].ID })
}

func recipeIdx(cfg *economysim.Config, id string) int {
	return idxOf(len(cfg.Recipes), id, func(i int) string { return cfg.Recipes[i].ID })
}

// groupRows returns the rows of the agents of the group (region, role) in
// ascending agent ID, which is agent order.
func groupRows(rows []economysim.AgentRow, region, role int) []economysim.AgentRow {
	var out []economysim.AgentRow
	for _, r := range rows {
		if int(r.Region) == region && int(r.Role) == role {
			out = append(out, r)
		}
	}
	return out
}

// buildWarm builds the world of a warm pack with the audit on.
func buildWarm(t *testing.T, layers []string, opts ...economysim.Option) (*economysim.World, economysim.Config) {
	t.Helper()
	raw := loadPack(t, layers...)
	w, err := economysim.NewWorld(raw, append([]economysim.Option{economysim.WithAudit()}, opts...)...)
	if err != nil {
		t.Fatalf("NewWorld(%v, WithAudit) error = %v, want nil", layers, err)
	}
	return w, economysim.Canonical(raw)
}

// TestNewWorld_WarmStartPricesMoneyAndPools verifies, over the packs warm-one,
// warm-second and warm-two, the parts of the warm start that need no job: a seed
// price is the market's published price and the centre of each agent's belief in
// it, and a market with no row takes the reference price (FR-VIA-031, 042); the
// money of an agent of a group with an entry is the entry's Money times its
// weight, in place of StartMoney, and a group with no entry takes the defaults
// (FR-VIA-032, 042); a pool is WarmPool.Money and 0 when absent (FR-EST-015); the
// starting wage is kept for the regions that have one (FR-VIA-052); no money
// exists but the sum of the agents' and the pools', which is the construction
// bucket's `initial` money, and the totals of tick 0 are empty (FR-LED-046); the
// audit passes at tick 0 (FR-WLD-038).
//
// Requirement: FR-VIA-031, FR-VIA-032, FR-VIA-042, FR-VIA-052, FR-EST-015, FR-WLD-038, FR-LED-046
func TestNewWorld_WarmStartPricesMoneyAndPools(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	t.Parallel()
	for _, pk := range warmPacks {
		t.Run(pk.name, func(t *testing.T) {
			t.Parallel()
			w, cfg := buildWarm(t, pk.layers)
			rows := economysim.ConstructionRows(w)
			if err := w.Audit(); err != nil {
				t.Errorf("Audit() at tick 0 = %v, want nil", err)
			}
			checkWarmPrices(t, w, &cfg, rows)
			var total economysim.Money
			for _, g := range cfg.Groups {
				region, role := regionIdx(&cfg, g.Region), roleIdx(&cfg, g.Role)
				want := cfg.Roles[role].StartMoney // the default of a group with no entry
				for _, wg := range cfg.WarmStart.Groups {
					if wg.Region == g.Region && wg.Role == g.Role {
						want = wg.Money
					}
				}
				for _, r := range groupRows(rows, region, role) {
					if got := economysim.Money(r.Money); got != want*economysim.Money(r.Weight) {
						t.Errorf("%s: agent %d of %s/%s holds %d money, want %d * weight %d = %d", pk.name, r.ID, g.Region, g.Role, got, want, r.Weight, want*economysim.Money(r.Weight))
					}
					total += economysim.Money(r.Money)
				}
			}
			for ri := range cfg.Regions {
				var want economysim.Money
				for _, p := range cfg.WarmStart.Pools {
					if p.Region == cfg.Regions[ri].ID {
						want = p.Money
					}
				}
				if got := economysim.PoolMoney(w, economysim.RegionID(ri)); got != want {
					t.Errorf("%s: pool of %s = %d, want %d (the row's Money, 0 when absent)", pk.name, cfg.Regions[ri].ID, got, want)
				}
				total += want
			}
			bucket, tick := economysim.ConstructionFlows(w)
			var created economysim.Money
			for _, f := range bucket {
				if f.Reason == ledger.Initial && f.Commodity == ledger.Money {
					created += economysim.Money(f.Total)
				}
			}
			if created != total || len(tick) != 0 {
				t.Errorf("%s: construction bucket created %d money and tick 0 has %d flows, want the sum of the agents' and pools' money %d and no flow in tick 0", pk.name, created, len(tick), total)
			}
			wages := economysim.SeedWages(w)
			for ri := range cfg.Regions {
				var want int64
				for _, sw := range cfg.WarmStart.Wages {
					if sw.Region == cfg.Regions[ri].ID {
						want = int64(sw.Wage)
					}
				}
				if wages[ri] != want {
					t.Errorf("%s: seed wage of %s = %d, want %d", pk.name, cfg.Regions[ri].ID, wages[ri], want)
				}
			}
		})
	}
}

// checkWarmPrices checks the published price of every market and the centre of
// every belief against the seed table, the reference price where there is no row.
func checkWarmPrices(t *testing.T, w *economysim.World, cfg *economysim.Config, rows []economysim.AgentRow) {
	t.Helper()
	want := make([]economysim.Price, len(cfg.Regions)*len(cfg.Commodities))
	for ri := range cfg.Regions {
		for ci := range cfg.Commodities {
			want[ri*len(cfg.Commodities)+ci] = cfg.Commodities[ci].ReferencePrice
		}
	}
	for _, p := range cfg.WarmStart.Prices {
		want[regionIdx(cfg, p.Region)*len(cfg.Commodities)+commodityIdx(cfg, p.Commodity)] = p.Price
	}
	for i, m := range economysim.MarketRecords(w) {
		if m.Price != want[i] {
			t.Errorf("market %d (region %d, commodity %d) price = %d, want %d", i, m.Region, m.Commodity, m.Price, want[i])
		}
	}
	rt := economysim.NewRoleTable(cfg)
	for _, r := range rows {
		trades := economysim.RoleTableRow(&rt, r.Role).Tradable
		for bi, b := range r.Beliefs {
			if exp := want[int(r.Region)*len(cfg.Commodities)+trades[bi]]; b.Centre != exp {
				t.Errorf("agent %d belief in commodity %d has centre %d, want the seed price %d", r.ID, trades[bi], b.Centre, exp)
			}
		}
	}
}

// refBusy is the joint largest-remainder apportionment of n agents over the
// shares of a group's jobs, the idle share last, written out as the spec words
// it (spec-viability.md 8.5.2) and not as apportion computes it: the quotas
// share * n / 10 000, the floors first, then one agent at a time to the largest
// remainder, the earlier entry on a tie.
func refBusy(shares []int64, n int64) []int64 {
	var sum int64
	for _, s := range shares {
		sum += s
	}
	all := append(slices.Clone(shares), 10_000-sum)
	got, rem := make([]int64, len(all)), make([]int64, len(all))
	left := n
	for i, s := range all {
		got[i], rem[i] = s*n/10_000, s*n%10_000
		left -= got[i]
	}
	for ; left > 0; left-- {
		best := 0
		for i := range rem {
			if rem[i] > rem[best] {
				best = i
			}
		}
		got[best]++
		rem[best] = -1
	}
	return got[:len(shares)]
}

// siteFactor is the yield factor of a recipe in a region in a season, as
// spec-production.md 6.2 words it: the profile's entry for the season, else All,
// else 0 when the profile lists seasons, else 10 000 for a recipe that is not
// sited and 0 for one that is. The shock, the technology and the trainee terms
// are neutral at construction.
func siteFactor(cfg *economysim.Config, region string, rc *economysim.Recipe, season string) int64 {
	for _, y := range cfg.Yields {
		if y.Region != region || y.Recipe != rc.ID {
			continue
		}
		for _, s := range y.Seasons {
			if s.Season == season {
				return s.BP
			}
		}
		switch {
		case y.HasAll:
			return y.All
		case len(y.Seasons) > 0:
			return 0
		}
	}
	if rc.Sited {
		return 0
	}
	return 10_000
}

// mLot is a lot of the model of a warm stock: the lots of one commodity of one
// agent, ascending by (born, durability), merged where the key is equal.
type mLot struct{ born, dur, qty int64 }

func addLot(lots []mLot, l mLot) []mLot {
	for i := range lots {
		if lots[i].born == l.born && lots[i].dur == l.dur {
			lots[i].qty += l.qty
			return lots
		}
	}
	lots = append(lots, l)
	slices.SortFunc(lots, func(a, b mLot) int {
		if a.born != b.born {
			return int(a.born - b.born)
		}
		return int(a.dur - b.dur)
	})
	return lots
}

// takeOldest consumes n units from the front of the lots, oldest first.
func takeOldest(lots []mLot, n int64) []mLot {
	var out []mLot
	for _, l := range lots {
		take := min(n, l.qty)
		n -= take
		if l.qty > take {
			out = append(out, mLot{l.born, l.dur, l.qty - take})
		}
	}
	return out
}

// wearItems wears n items by units each, the lowest remaining durability first;
// an item at 0 or below is destroyed, the survivors merge into the lots of their
// new durability.
func wearItems(lots []mLot, n, units int64) []mLot {
	byDur := slices.Clone(lots)
	slices.SortFunc(byDur, func(a, b mLot) int { return int(a.dur-b.dur)*1000 + int(a.born-b.born) })
	var out []mLot
	var worn []mLot
	for _, l := range byDur {
		take := min(n, l.qty)
		n -= take
		if l.qty > take {
			out = addLot(out, mLot{l.born, l.dur, l.qty - take})
		}
		if take > 0 && l.dur-units > 0 {
			worn = append(worn, mLot{l.born, l.dur - units, take})
		}
	}
	for _, l := range worn {
		out = addLot(out, l)
	}
	return out
}

// TestNewWorld_WarmStartStaggerJobsAndBooks verifies, over the packs warm-one,
// warm-second and warm-two, the groups with an entry in the warm table
// (spec-world.md 5.6.1 items 3 and 4, spec-viability.md 8.4.3, 8.5.2, 8.5.3).
// For agent i of a group of n: a perishable lot is born at -floor(i * ShelfLife
// / n) and a durable item has the durability Durability - floor(i * Durability /
// n) (FR-VIA-037, 038); the first busy agents, by one joint largest-remainder
// apportionment with the idle share last, hold the jobs spec by spec, the jth of
// busy with the offset V[floor(j * |V| / busy)], Remaining = Duration - o, the
// effective factor of the season of the tick -o, Executions the weight, the agent
// as principal and the pending output floor(weight * quantity * factor /
// 10 000) with residue 0 (FR-VIA-034, 035); the stock after the warm start is the
// stagger lots plus the granted inputs (born 0) minus the oldest-first take of
// the inputs, with the tools worn most worn first, so the total of an input
// commodity is the stagger total (FR-VIA-036, O11); and the construction bucket
// holds the `initial` goods (the stock plus the inputs granted), the
// `recipe_input` goods (the inputs taken) and the wear, so each commodity's total
// held is initial - recipe_input - wear.
//
// Requirement: FR-VIA-033, FR-VIA-034, FR-VIA-035, FR-VIA-036, FR-VIA-037, FR-VIA-038, FR-VIA-053, FR-WLD-038
func TestNewWorld_WarmStartStaggerJobsAndBooks(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	t.Parallel()
	for _, pk := range warmPacks {
		t.Run(pk.name, func(t *testing.T) {
			t.Parallel()
			w, cfg := buildWarm(t, pk.layers)
			rows := economysim.ConstructionRows(w)
			nc := len(cfg.Commodities)
			initial, taken := make([]int64, nc), make([]int64, nc)
			jobs := 0
			for _, g := range cfg.Groups {
				if hasWarmGroup(&cfg, g.Region, g.Role) {
					continue
				}
				// A group with no entry takes the defaults: weight times StartStock.
				for _, row := range groupRows(rows, regionIdx(&cfg, g.Region), roleIdx(&cfg, g.Role)) {
					for _, it := range cfg.Roles[row.Role].StartStock {
						initial[commodityIdx(&cfg, it.Commodity)] += int64(it.Quantity) * row.Weight
					}
				}
			}
			for _, g := range cfg.WarmStart.Groups {
				jobs += checkWarmGroup(t, pk.name, w, &cfg, rows, &g, initial, taken)
			}
			if jobs == 0 {
				t.Errorf("%s: no warm job placed, want the table's jobs (the pack has them)", pk.name)
			}
			checkWarmBooks(t, pk.name, w, &cfg, rows, initial, taken)
		})
	}
}

// hasWarmGroup reports whether the warm table has an entry for the group.
func hasWarmGroup(cfg *economysim.Config, region, role string) bool {
	for _, g := range cfg.WarmStart.Groups {
		if g.Region == region && g.Role == role {
			return true
		}
	}
	return false
}

// checkWarmGroup checks one group with an entry: the stagger of its stock, its
// jobs and the lots left after the books. It adds the goods the group's agents
// were granted by `initial` to initial and the inputs taken to taken, and
// returns the number of jobs placed.
func checkWarmGroup(t *testing.T, pack string, w *economysim.World, cfg *economysim.Config, rows []economysim.AgentRow, g *economysim.WarmGroup, initial, taken []int64) int {
	t.Helper()
	region, role := regionIdx(cfg, g.Region), roleIdx(cfg, g.Role)
	agents := groupRows(rows, region, role)
	n := int64(len(agents))
	shares := make([]int64, len(g.Jobs))
	for i, j := range g.Jobs {
		shares[i] = j.InProgressBP
	}
	busy := refBusy(shares, n)
	// recipeOf[i] is the recipe index of agent i's job, -1 when idle, and offset
	// the virtual start offset of the jth busy agent of its spec.
	recipeOf, offset := make([]int, n), make([]int64, n)
	next := int64(0)
	for si, j := range g.Jobs {
		rc := recipeIdx(cfg, j.Recipe)
		v := economysim.StartOffsets(w, economysim.RegionID(region), economysim.RecipeID(rc))
		for k := range busy[si] {
			recipeOf[next], offset[next] = rc, v[k*int64(len(v))/busy[si]]
			next++
		}
	}
	for ; next < n; next++ {
		recipeOf[next] = -1
	}
	placed := 0
	for i, a := range agents {
		job := economysim.AgentJob(w, a.ID)
		if recipeOf[i] < 0 {
			if job.Has {
				t.Errorf("%s: agent %d (%d of %d in %s/%s) has a job, want it idle", pack, a.ID, i, n, g.Region, g.Role)
			}
		} else {
			placed++
			checkWarmJob(t, pack, cfg, &a, job, recipeOf[i], offset[i], g)
		}
		checkWarmLots(t, pack, w, cfg, &a, int64(i), n, g, recipeOf[i], initial, taken)
	}
	return placed
}

// checkWarmJob checks the job of one agent against the rule of 8.5.2.
func checkWarmJob(t *testing.T, pack string, cfg *economysim.Config, a *economysim.AgentRow, job economysim.AgentJobView, recipe int, offset int64, g *economysim.WarmGroup) {
	t.Helper()
	rc := &cfg.Recipes[recipe]
	season := ""
	if s := cfg.Calendar.SeasonAt(-offset); s >= 0 {
		season = cfg.Calendar.Seasons[s].ID
	}
	factor := siteFactor(cfg, g.Region, rc, season)
	if !job.Has || int(job.Recipe) != recipe || job.Remaining != rc.Duration-offset || job.Executions != a.Weight || job.Principal != a.ID || job.Yield != factor {
		t.Errorf("%s: agent %d job = %+v, want recipe %s with Remaining %d - %d, Executions %d, principal itself and factor %d (season %q at tick %d)",
			pack, a.ID, job, rc.ID, rc.Duration, offset, a.Weight, factor, season, -offset)
	}
	want := make([]int64, len(rc.Outputs))
	for k, o := range rc.Outputs {
		want[k] = a.Weight * int64(o.Quantity) * factor / 10_000 // residue 0
	}
	if !slices.Equal(job.Pending, want) {
		t.Errorf("%s: agent %d pending output = %v, want floor(weight %d * quantity * %d / 10 000) per output = %v", pack, a.ID, job.Pending, a.Weight, factor, want)
	}
}

// modelLots returns, per commodity, the lots the model of 8.4.3 and 8.5.3 gives
// agent i of n after the warm start, and adds the goods created by `initial` to
// initial and the inputs taken to taken.
func modelLots(cfg *economysim.Config, a *economysim.AgentRow, i, n int64, g *economysim.WarmGroup, recipe int, initial, taken []int64) [][]mLot {
	lots := make([][]mLot, len(cfg.Commodities))
	for _, it := range g.Stock {
		c := commodityIdx(cfg, it.Commodity)
		cm := &cfg.Commodities[c]
		l := mLot{qty: int64(it.Quantity) * a.Weight}
		if cm.ShelfLife > 0 {
			l.born = -(i * cm.ShelfLife / n)
		}
		if cm.Durability > 0 {
			l.dur = cm.Durability - i*cm.Durability/n
		}
		lots[c] = addLot(lots[c], l)
		initial[c] += l.qty
	}
	if recipe < 0 {
		return lots
	}
	rc := &cfg.Recipes[recipe]
	for _, in := range rc.Inputs {
		c := commodityIdx(cfg, in.Commodity)
		need := int64(in.Quantity) * a.Weight
		lots[c] = addLot(lots[c], mLot{dur: cfg.Commodities[c].Durability, qty: need}) // the grant, born 0
		lots[c] = takeOldest(lots[c], need)
		initial[c] += need
		taken[c] += need
	}
	for _, tl := range rc.Requires {
		c := commodityIdx(cfg, tl.Commodity)
		lots[c] = wearItems(lots[c], a.Weight, tl.WearPerUse)
	}
	return lots
}

// checkWarmLots compares the lots an agent holds with the model.
func checkWarmLots(t *testing.T, pack string, w *economysim.World, cfg *economysim.Config, a *economysim.AgentRow, i, n int64, g *economysim.WarmGroup, recipe int, initial, taken []int64) {
	t.Helper()
	want := modelLots(cfg, a, i, n, g, recipe, initial, taken)
	got := make([][]mLot, len(cfg.Commodities))
	for _, l := range economysim.AgentLots(w, a.ID) {
		got[l.Commodity] = append(got[l.Commodity], mLot{l.Born, l.Durability, l.Quantity})
	}
	for c := range want {
		if !slices.Equal(got[c], want[c]) {
			t.Errorf("%s: agent %d (%d of %d, job recipe %d) lots of %s = %v, want %v (born, durability, quantity)", pack, a.ID, i, n, recipe, cfg.Commodities[c].ID, got[c], want[c])
		}
	}
}

// checkWarmBooks checks the construction bucket against the goods the model
// created and took: each commodity's `initial` total is the expected one, its
// `recipe_input` total is the inputs taken, and what the agents hold is
// initial - recipe_input - wear.
func checkWarmBooks(t *testing.T, pack string, w *economysim.World, cfg *economysim.Config, rows []economysim.AgentRow, initial, taken []int64) {
	t.Helper()
	bucket, tick := economysim.ConstructionFlows(w)
	gotInitial, gotTaken, gotWear := make([]int64, len(initial)), make([]int64, len(initial)), make([]int64, len(initial))
	for _, f := range bucket {
		if f.Commodity == ledger.Money {
			continue
		}
		switch f.Reason {
		case ledger.Initial:
			gotInitial[f.Commodity] += f.Total
		case ledger.RecipeInput:
			gotTaken[f.Commodity] += f.Total
		case ledger.Wear:
			gotWear[f.Commodity] += f.Total
		}
	}
	held := make([]int64, len(initial))
	for _, r := range rows {
		for c, q := range r.Stock {
			held[c] += q
		}
	}
	for c := range initial {
		id := cfg.Commodities[c].ID
		if gotInitial[c] != initial[c] || gotTaken[c] != taken[c] || held[c] != initial[c]-taken[c]-gotWear[c] {
			t.Errorf("%s: commodity %s: bucket initial %d, recipe_input %d, held %d, wear %d; want initial %d, recipe_input %d and held = initial - recipe_input - wear",
				pack, id, gotInitial[c], gotTaken[c], held[c], gotWear[c], initial[c], taken[c])
		}
	}
	if len(tick) != 0 {
		t.Errorf("%s: tick 0 has %d flows after construction, want none (the bucket holds them)", pack, len(tick))
	}
}

// TestNewWorld_WarmStartTakesNoRandomWord verifies that applying the warm start
// takes no random number and leaves the stream of agent creation as it would be
// without it (FR-VIA-040, FR-WLD-062): a world built from a warm pack and one
// built from the same pack without the table, with the same seed, end with the
// same random state, and every agent has the same target ratio and the same
// residues, which are the two words of its own creation stream.
//
// Requirement: FR-VIA-040, FR-WLD-062, FR-WLD-066
func TestNewWorld_WarmStartTakesNoRandomWord(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	t.Parallel()
	for _, pk := range warmPacks {
		for _, seed := range []uint64{0, 11} {
			t.Run(fmt.Sprintf("%s seed %d", pk.name, seed), func(t *testing.T) {
				t.Parallel()
				warm, _ := buildWarm(t, pk.layers, economysim.WithSeed(seed))
				cold, _ := buildWarm(t, pk.cold, economysim.WithSeed(seed))
				if got, want := economysim.PCGState(warm), economysim.PCGState(cold); !slices.Equal(got, want) {
					t.Errorf("%s seed %d: random state after a warm start = %x, want the cold world's %x", pk.name, seed, got, want)
				}
				wr, cr := economysim.ConstructionRows(warm), economysim.ConstructionRows(cold)
				if len(wr) != len(cr) {
					t.Fatalf("%s seed %d: %d agents with the warm start, %d without, want the same", pk.name, seed, len(wr), len(cr))
				}
				for i := range wr {
					if wr[i].Target != cr[i].Target || !slices.Equal(wr[i].Residues, cr[i].Residues) {
						t.Errorf("%s seed %d: agent %d target %d residues %v, cold world has %d and %v, want equal (the creation draws come after the warm start and are not moved by it)",
							pk.name, seed, wr[i].ID, wr[i].Target, wr[i].Residues, cr[i].Target, cr[i].Residues)
					}
				}
			})
		}
	}
}

// TestNewWorld_WarmStartEventsAndLogs verifies, over warm-one, that a world with
// a warm start says so once and says nothing else about it (FR-VIA-046,
// FR-OBS-076): WorldCreated has WarmStart true, AgentSpawned carries the warm
// money of each agent, one Info line "warm start applied" with the regions,
// prices, wages, groups and jobs counts of the table comes after the
// configuration warnings and before "world created", and that line carries the
// same counts.
//
// Requirement: FR-VIA-046, FR-OBS-076, FR-WLD-037
func TestNewWorld_WarmStartEventsAndLogs(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	t.Parallel()
	raw := loadPack(t, warmPacks[0].layers...)
	cfg := economysim.Canonical(raw)
	rec := &recorder{}
	h := newCaptureHandler()
	w, err := economysim.NewWorld(raw, economysim.WithObserver(rec), economysim.WithLogger(slog.New(h)))
	if err != nil {
		t.Fatalf("NewWorld() error = %v, want nil", err)
	}
	rows := economysim.ConstructionRows(w)
	if wc, ok := rec.events[0].(economysim.WorldCreated); !ok || !wc.WarmStart {
		t.Errorf("first event = %+v, want WorldCreated with WarmStart true", rec.events[0])
	}
	for i, e := range rec.events[1:] {
		got, ok := e.(economysim.AgentSpawned)
		if !ok || got.Agent != rows[i].ID || int64(got.StartMoney) != rows[i].Money {
			t.Fatalf("event %d = %+v, want AgentSpawned of agent %d with StartMoney %d, the warm money times the weight", i+1, e, rows[i].ID, rows[i].Money)
		}
	}
	jobs := int64(0)
	for _, g := range cfg.WarmStart.Groups {
		jobs += int64(len(g.Jobs))
	}
	want := map[string]int64{
		"regions": int64(len(cfg.Regions)), "prices": int64(len(cfg.WarmStart.Prices)), "wages": int64(len(cfg.WarmStart.Wages)),
		"groups": int64(len(cfg.WarmStart.Groups)), "jobs": jobs,
	}
	lines := h.records()
	applied := -1
	for i, l := range lines {
		if l.msg != "warm start applied" {
			continue
		}
		if applied >= 0 {
			t.Errorf("a second \"warm start applied\" line at %d, want one", i)
		}
		applied = i
		if l.level != slog.LevelInfo || len(l.attrs) != len(want) {
			t.Errorf("line %d = %v, want an Info line with the %d keys of the table", i, l, len(want))
		}
		for _, a := range l.attrs {
			if exp, ok := want[a.Key]; !ok || a.Value.Int64() != exp {
				t.Errorf("warm start applied attribute %v, want one of %v with the table's count", a, want)
			}
		}
	}
	if applied < 0 || applied != len(lines)-2 || lines[len(lines)-1].msg != "world created" {
		t.Errorf("lines = %v, want \"warm start applied\" second to last and \"world created\" last", lines)
	}
}

// TestNewWorld_WorldCreatedLineCarriesTheWarmStartCounts verifies the attributes of
// the Info line "world created" about the warm start (spec-world.md 5.8.1): with a
// warm start the line has warm_start true and the counts of the rows of its table
// (prices, wages, groups, jobs), and without one it has warm_start false and none of
// the four counts.
//
// Requirement: FR-WLD-037, FR-OBS-010
func TestNewWorld_WorldCreatedLineCarriesTheWarmStartCounts(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	t.Parallel()
	line := func(t *testing.T, raw economysim.Config) map[string]slog.Value {
		t.Helper()
		h := newCaptureHandler()
		if _, err := economysim.NewWorld(raw, economysim.WithLogger(slog.New(h))); err != nil {
			t.Fatalf("NewWorld() error = %v, want nil", err)
		}
		for _, l := range h.records() {
			if l.msg == "world created" {
				out := map[string]slog.Value{}
				for _, a := range l.attrs {
					out[a.Key] = a.Value
				}
				return out
			}
		}
		t.Fatalf("no \"world created\" line")
		return nil
	}
	raw := loadPack(t, warmPacks[0].layers...)
	cfg := economysim.Canonical(raw)
	jobs := int64(0)
	for _, g := range cfg.WarmStart.Groups {
		jobs += int64(len(g.Jobs))
	}
	got := line(t, raw)
	want := map[string]int64{"prices": int64(len(cfg.WarmStart.Prices)), "wages": int64(len(cfg.WarmStart.Wages)), "groups": int64(len(cfg.WarmStart.Groups)), "jobs": jobs}
	if v, ok := got["warm_start"]; !ok || !v.Bool() {
		t.Errorf("warm_start = %v, want true", got["warm_start"])
	}
	for k, w := range want {
		if v, ok := got[k]; !ok || v.Int64() != w {
			t.Errorf("attribute %s of \"world created\" = %v (present %v), want %d", k, v, ok, w)
		}
	}
	cold := raw
	cold.WarmStart = nil
	got = line(t, cold)
	if v, ok := got["warm_start"]; !ok || v.Bool() {
		t.Errorf("without a warm start: warm_start = %v, want false", got["warm_start"])
	}
	for k := range want {
		if _, ok := got[k]; ok {
			t.Errorf("without a warm start: attribute %s present, want none", k)
		}
	}
}
