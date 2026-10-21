package economysim

import (
	"errors"
	"math"
	"os"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/ledger"
)

// TestApportion verifies the joint largest-remainder apportionment of the agents
// of a group among its jobs (spec-viability.md 8.5.2): the quotas share * n /
// 10 000 per entry with the idle share last, the floors first, the agents left to
// the largest remainders, a tie to the earlier entry, and a sum that never
// exceeds n. Each row works the quotas out.
//
// Requirement: FR-VIA-034
func TestApportion(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		shares []int64
		n      int64
		want   []int64
	}{
		// 5 113 * 20 = 102 260: quota 10, remainder 2 260. Idle 4 887 * 20 = 97 740:
		// quota 9, remainder 7 740. One agent is left (20 - 19), and the idle
		// remainder is the larger, so the job keeps 10.
		{"a share of 5 113 over 20 agents rounds down to 10", []int64{5113}, 20, []int64{10}},
		// 418 * 20 = 8 360: quota 0, remainder 8 360. Idle 9 582 * 20 = 191 640: quota
		// 19, remainder 1 640. The agent left goes to the larger remainder, the job.
		{"a share of 418 over 20 agents rounds up to 1", []int64{418}, 20, []int64{1}},
		// Quotas 0.5, 0.5 and idle 0: floors 0, one agent left, remainders 5 000, 5 000
		// and 0, so the earlier entry takes it.
		{"a tie goes to the earlier entry", []int64{5000, 5000}, 1, []int64{1, 0}},
		// Quotas 3.333, 3.333, 3.334 (remainders 3 330, 3 330, 3 340), idle 0: floors 9,
		// one left, to the third.
		{"the largest remainder takes the agent left", []int64{3333, 3333, 3334}, 10, []int64{3, 3, 4}},
		// Quotas 0.75, 0.75, idle 1.5 (remainders 7 500, 7 500, 5 000): floors 0, 0, 1,
		// two left, to the two jobs.
		{"two agents left go to the two largest remainders", []int64{2500, 2500}, 3, []int64{1, 1}},
		{"no agent", []int64{5000}, 0, []int64{0}},
		{"no job", nil, 5, []int64{}},
		{"every agent busy", []int64{10000}, 7, []int64{7}},
		{"a zero share gets nothing", []int64{0, 10000}, 7, []int64{0, 7}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := apportion(tt.shares, tt.n)
			if err != nil || !slices.Equal(got, tt.want) {
				t.Errorf("apportion(%v, %d) = %v, %v, want %v, nil", tt.shares, tt.n, got, err, tt.want)
			}
		})
	}
	for _, tt := range []struct {
		name   string
		shares []int64
		n      int64
	}{
		{"a negative share", []int64{-1}, 5},
		{"a share above 10 000", []int64{10001}, 5},
		{"shares above 10 000 together", []int64{6000, 4001}, 5},
		{"a negative count", []int64{5000}, -1},
	} {
		if got, err := apportion(tt.shares, tt.n); !errors.Is(err, ErrOutOfBounds) {
			t.Errorf("%s: apportion(%v, %d) = %v, %v, want ErrOutOfBounds", tt.name, tt.shares, tt.n, got, err)
		}
	}
}

// TestPhase verifies the stagger rule floor(i * p / n) (spec-viability.md 8.4.3):
// a perishable of ShelfLife 5 over 4 agents gets 0, 1, 2, 3 (floor(5 i / 4): 0,
// 1.25, 2.5, 3.75) and a durable of Durability 10 gets 0, 2, 5, 7 (floor(10 i /
// 4): 0, 2.5, 5, 7.5), so the remaining durability is 10, 8, 5, 3; the product
// is formed in 128 bits, so the largest operands do not wrap.
//
// Requirement: FR-VIA-037
func TestPhase(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name    string
		n, p    int64
		want    []int64
		wantErr bool
	}{
		{"a shelf life of 5 over 4 agents", 4, 5, []int64{0, 1, 2, 3}, false},
		{"a durability of 10 over 4 agents", 4, 10, []int64{0, 2, 5, 7}, false},
		{"one agent", 1, 10, []int64{0}, false},
		{"a period of 0", 3, 0, []int64{0, 0, 0}, false},
		{"more agents than the period", 4, 2, []int64{0, 0, 1, 1}, false},
	} {
		got := make([]int64, 0, tt.n)
		for i := range tt.n {
			p, err := phase(i, tt.n, tt.p)
			if err != nil {
				t.Fatalf("%s: phase(%d, %d, %d) error = %v, want nil", tt.name, i, tt.n, tt.p, err)
			}
			got = append(got, p)
		}
		if !slices.Equal(got, tt.want) {
			t.Errorf("%s: phases of %d agents over %d = %v, want %v", tt.name, tt.n, tt.p, got, tt.want)
		}
	}
	for _, c := range [][3]int64{{4, 4, 5}, {-1, 4, 5}, {0, 0, 5}, {0, 4, -1}} {
		if got, err := phase(c[0], c[1], c[2]); !errors.Is(err, ErrOutOfBounds) {
			t.Errorf("phase(%d, %d, %d) = %d, %v, want ErrOutOfBounds", c[0], c[1], c[2], got, err)
		}
	}
	// (MaxInt64 - 1) * MaxInt64 / MaxInt64 = MaxInt64 - 1, a product of 126 bits.
	if got, err := phase(math.MaxInt64-1, math.MaxInt64, math.MaxInt64); err != nil || got != math.MaxInt64-1 {
		t.Errorf("phase(MaxInt64-1, MaxInt64, MaxInt64) = %d, %v, want MaxInt64-1, nil", got, err)
	}
}

// TestPickOffset verifies the offset of the jth of busy agents, V[floor(j * |V| /
// busy)] (spec-viability.md 8.5.2): V = [0, 1, 2, 3] and busy 2 give 0 and 2
// (floor(j * 4 / 2)); busy 3 gives 0, 1, 2 (floor(4 j / 3): 0, 1, 2); busy 5, more
// than |V|, gives 0, 0, 1, 2, 3 (floor(4 j / 5): 0, 0, 1, 2, 3).
//
// Requirement: FR-VIA-034
func TestPickOffset(t *testing.T) {
	t.Parallel()
	v := []int64{0, 1, 2, 3}
	for _, tt := range []struct {
		busy int64
		want []int64
	}{{1, []int64{0}}, {2, []int64{0, 2}}, {3, []int64{0, 1, 2}}, {5, []int64{0, 0, 1, 2, 3}}} {
		var got []int64
		for j := range tt.busy {
			o, err := pickOffset(v, j, tt.busy)
			if err != nil {
				t.Fatalf("pickOffset(%v, %d, %d) error = %v, want nil", v, j, tt.busy, err)
			}
			got = append(got, o)
		}
		if !slices.Equal(got, tt.want) {
			t.Errorf("offsets of %d busy agents over %v = %v, want %v", tt.busy, v, got, tt.want)
		}
	}
	if _, err := pickOffset(nil, 0, 1); !errors.Is(err, ErrWarmStartJob) {
		t.Errorf("pickOffset(nil, 0, 1) error = %v, want ErrWarmStartJob", err)
	}
	for _, j := range []int64{-1, 2} {
		if _, err := pickOffset(v, j, 2); !errors.Is(err, ErrOutOfBounds) {
			t.Errorf("pickOffset(%v, %d, 2) error = %v, want ErrOutOfBounds", v, j, err)
		}
	}
}

// TestStaggerKey verifies the stated lot key of a warm stock (spec-viability.md
// 8.4.3, 8.6): a perishable of ShelfLife 4 over 4 agents is born at -floor(4 i /
// 4) = 0, -1, -2, -3 for agent 0 to 3 (the range of the ledger is 0 to
// -(ShelfLife - 1)); a durable of Durability 10 has 10, 8, 5, 3; a commodity with
// both has both; one with neither has the default key.
//
// Requirement: FR-VIA-037, FR-VIA-038
func TestStaggerKey(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name string
		c    Commodity
		want [4]ledger.Key
	}{
		{"perishable", Commodity{ShelfLife: 4}, [4]ledger.Key{{Commodity: 7}, {Commodity: 7, Born: -1}, {Commodity: 7, Born: -2}, {Commodity: 7, Born: -3}}},
		{"durable", Commodity{Durability: 10}, [4]ledger.Key{{Commodity: 7, Durability: 10}, {Commodity: 7, Durability: 8}, {Commodity: 7, Durability: 5}, {Commodity: 7, Durability: 3}}},
		{"both", Commodity{ShelfLife: 4, Durability: 10}, [4]ledger.Key{{Commodity: 7, Durability: 10}, {Commodity: 7, Born: -1, Durability: 8}, {Commodity: 7, Born: -2, Durability: 5}, {Commodity: 7, Born: -3, Durability: 3}}},
		{"neither", Commodity{}, [4]ledger.Key{{Commodity: 7}, {Commodity: 7}, {Commodity: 7}, {Commodity: 7}}},
	} {
		for i := range 4 {
			got, err := staggerKey(&tt.c, 7, int64(i), 4)
			if err != nil || got != tt.want[i] {
				t.Errorf("%s: staggerKey(agent %d of 4) = %+v, %v, want %+v", tt.name, i, got, err, tt.want[i])
			}
		}
	}
	if _, err := staggerKey(&Commodity{ShelfLife: 4}, 0, 4, 4); !errors.Is(err, ErrOutOfBounds) {
		t.Errorf("staggerKey(agent 4 of 4) error = %v, want ErrOutOfBounds", err)
	}
	if _, err := staggerKey(&Commodity{Durability: 4}, 0, 4, 4); !errors.Is(err, ErrOutOfBounds) {
		t.Errorf("staggerKey(durable, agent 4 of 4) error = %v, want ErrOutOfBounds", err)
	}
}

// warmFarmConfig is a pack of one region and one role whose recipe "farm" takes 3
// ticks, consumes 2 seed (a perishable of shelf life 4) and needs one plow (a
// durable of 10) worn by 1, with four agents of weight 1 (scale 1) and a warm
// table that holds money, 3 seed and the plow for each, and puts 75% of the
// agents in progress: apportion gives busy 3, so the agents 0, 1 and 2 have jobs.
func warmFarmConfig(withPlow bool) Config {
	c := viaToy()
	c.World.Scale, c.World.MinAgents = 1, 1
	seed := viaCommodity("seed", 50, func(c *Commodity) { c.ShelfLife = 4 })
	plow := viaCommodity("plow", 500, func(c *Commodity) { c.Durability = 10 })
	c.Commodities = append(c.Commodities, plow, seed) // canonical order: grain, plow, seed
	c.Recipes[0].Duration = 3
	c.Recipes[0].Inputs = []Item{{Commodity: "seed", Quantity: 2}}
	c.Recipes[0].Requires = []Tool{{Commodity: "plow", WearPerUse: 1}}
	c.Recipes[0].Outputs = []Output{viaOut("grain", 12)}
	// The seed and the plow are made by source recipes of the same role, so the
	// chain is producible and the pack passes the viability rules.
	c.Recipes = append(c.Recipes,
		viaRecipe("pick", nil, []Output{viaOut("seed", 8)}, nil),
		viaRecipe("forge", nil, []Output{viaOut("plow", 1)}, nil))
	c.Roles[0].Recipes = []string{"farm", "forge", "pick"}
	c.Groups = []Group{{Region: "north", Role: "farmer", Households: 4}}
	stock := []Item{{Commodity: "seed", Quantity: 3}}
	if withPlow {
		stock = []Item{{Commodity: "plow", Quantity: 1}, {Commodity: "seed", Quantity: 3}}
	}
	c.WarmStart = &WarmStart{Groups: []WarmGroup{{
		Region: "north", Role: "farmer", Money: 5_000, Stock: stock,
		Jobs: []JobSpec{{Recipe: "farm", InProgressBP: 7_500}},
	}}}
	return c
}

// lotsOf returns the lots of agent id of a world.
func lotsOf(t *testing.T, w *World, id AgentID) []ledger.Lot {
	t.Helper()
	lots, err := w.ledger.AppendLots(nil, w.agents[id].account)
	if err != nil {
		t.Fatalf("AppendLots(agent %d) error = %v, want nil", id, err)
	}
	return lots
}

// seedLots returns the (Born, Quantity) pairs of a commodity among lots.
func seedLots(lots []ledger.Lot, commodity int) [][2]int64 {
	var out [][2]int64
	for _, l := range lots {
		if l.Commodity == commodity {
			out = append(out, [2]int64{l.Born, l.Quantity})
		}
	}
	return out
}

// TestWarmStart_StaggerBeforeTheOldestFirstTake verifies the order of 5.6.1 item 4
// and ruling O11 (spec-viability.md 8.5.3): the stagger vector is a property of
// the granted stock before the job's inputs are taken, and the Transform then
// consumes oldest first. Agent 2 of 4 (ID 3, the estate is 0) holds 3 seed
// born at -floor(2 * 4 / 4) = -2; the grant adds 2 seed born 0, so the vector
// before the take is [(-2, 3), (0, 2)], total 5; the oldest-first take of 2
// leaves [(-2, 1), (0, 2)], total 3, the stagger total. Agent 0 (ID 1) has
// phase 0, so its two lots are one: [(0, 5)] and then [(0, 3)]. Agent 3 has no
// job (busy 3 of 4) and holds the stagger lot [(-3, 3)] throughout. The test
// stops the construction between the grant and the take, which the two unexported
// steps allow without a hook in production code.
//
// Requirement: FR-VIA-036, FR-VIA-037, FR-VIA-038, FR-WLD-038
func TestWarmStart_StaggerBeforeTheOldestFirstTake(t *testing.T) {
	t.Parallel()
	w, err := newShell(warmFarmConfig(true), nil)
	if err != nil {
		t.Fatalf("newShell() error = %v, want nil", err)
	}
	b := w.beginConstruction()
	b.createAgents()
	if err := b.placeJobs(); err != nil {
		t.Fatalf("placeJobs() error = %v, want nil", err)
	}
	if err := b.grantInputs(); err != nil {
		t.Fatalf("grantInputs() error = %v, want nil", err)
	}
	seed := commodityID(t, &w.cfg, "seed")
	// The offsets V = [0, 1, 2] of the 3-tick recipe and busy 3: agents 0, 1 and 2
	// take V[floor(j * 3 / 3)] = j, so Remaining is 3 - j.
	cases := []struct {
		id            AgentID
		before, after [][2]int64
		remaining     int64
	}{
		{1, [][2]int64{{0, 5}}, [][2]int64{{0, 3}}, 3},
		{2, [][2]int64{{-1, 3}, {0, 2}}, [][2]int64{{-1, 1}, {0, 2}}, 2},
		{3, [][2]int64{{-2, 3}, {0, 2}}, [][2]int64{{-2, 1}, {0, 2}}, 1},
		{4, [][2]int64{{-3, 3}}, [][2]int64{{-3, 3}}, 0},
	}
	for _, tt := range cases {
		if got := seedLots(lotsOf(t, w, tt.id), int(seed)); !slices.Equal(got, tt.before) {
			t.Errorf("agent %d seed lots before the take = %v, want %v (born, quantity)", tt.id, got, tt.before)
		}
	}
	if err := b.takeInputs(); err != nil {
		t.Fatalf("takeInputs() error = %v, want nil", err)
	}
	for _, tt := range cases {
		if got := seedLots(lotsOf(t, w, tt.id), int(seed)); !slices.Equal(got, tt.after) {
			t.Errorf("agent %d seed lots after the take = %v, want %v (born, quantity)", tt.id, got, tt.after)
		}
		if a := &w.agents[tt.id]; a.hasJob() != (tt.remaining > 0) || a.job.remaining != tt.remaining {
			t.Errorf("agent %d job = (in progress %v, remaining %d), want remaining %d", tt.id, a.hasJob(), a.job.remaining, tt.remaining)
		}
	}
	if err := w.ledger.Audit(); err != nil {
		t.Errorf("Audit() after the warm start = %v, want nil: granted equals consumed", err)
	}
}

// commodityID returns the ID of a commodity of a canonical configuration.
func commodityID(t *testing.T, c *Config, id string) CommodityID {
	t.Helper()
	k, ok := c.commodityID(id)
	if !ok {
		t.Fatalf("commodity %q is not in the configuration", id)
	}
	return k
}

// TestWarmStart_JobWithoutItsToolIsRefused verifies that a warm job whose tool
// the group's stock does not hold makes NewWorld return ErrWarmStartJob and no
// world, rather than panic: Validate does not look at the stock, so the table is
// the input of this check.
//
// Guard: the wear of a tool the agent lacks would otherwise reach the ledger and
// panic on a configuration error.
//
// Requirement: FR-VIA-036, FR-VIA-041
func TestWarmStart_JobWithoutItsToolIsRefused(t *testing.T) {
	t.Parallel()
	w, err := NewWorld(warmFarmConfig(false))
	if !errors.Is(err, ErrWarmStartJob) || w != nil {
		t.Errorf("NewWorld(a warm job with no tool in the stock) = %v, %v, want nil and ErrWarmStartJob", w, err)
	}
	if w, err := NewWorld(warmFarmConfig(true), WithAudit()); err != nil || w == nil {
		t.Errorf("NewWorld(the same table with the plow) = %v, %v, want a world", w, err)
	}
}

// TestWarmStart_TextGuards verifies by text search that no rule after tick 0
// depends on the warm table and that nothing solves at construction
// (FR-VIA-021, FR-VIA-047): Config.WarmStart is read only by the files of the
// configuration (config.go, hash.go, validate.go), of the viability row
// (viability*.go), by welfare.go's seedWages, which the construction hands the
// wages to, and by construct_warm.go, and SolveSteadyState is called only by the
// viability files.
//
// Requirement: FR-VIA-021, FR-VIA-047
//
// Guard: a read of the table in a phase, a strategy or the snapshot would make a
// world with the table differ from one without it by more than its initial
// state, and a solve at construction would put the solver on every NewWorld.
func TestWarmStart_TextGuards(t *testing.T) {
	t.Parallel()
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read directory: %v", err)
	}
	read := regexp.MustCompile(`\b(\w+)\.WarmStart\b`)
	allowed := func(n string) bool {
		return slices.Contains([]string{"config.go", "hash.go", "validate.go", "welfare.go", "construct_warm.go"}, n) || strings.HasPrefix(n, "viability")
	}
	seen := false
	for _, e := range entries {
		n := e.Name()
		if !strings.HasSuffix(n, ".go") || strings.HasSuffix(n, "_test.go") {
			continue
		}
		b, err := os.ReadFile(n)
		if err != nil {
			t.Fatalf("read %s: %v", n, err)
		}
		src := string(b)
		for _, m := range read.FindAllStringSubmatch(src, -1) {
			if m[1] != "logkeys" && !allowed(n) {
				t.Errorf("%s reads %s, want Config.WarmStart read only by the configuration, viability, welfare.go and construct_warm.go files", n, m[0])
			}
		}
		if n == "construct_warm.go" && read.MatchString(src) {
			seen = true
		}
		if strings.Contains(src, "SolveSteadyState(") && !strings.HasPrefix(n, "viability") && n != "errors.go" && n != "bounds.go" {
			t.Errorf("%s calls SolveSteadyState, want only the viability files to", n)
		}
	}
	if !seen {
		t.Errorf("construct_warm.go does not read Config.WarmStart, want it to be the file that applies the table")
	}
}

// TestWarmStart_PoolIsSeededByInitial verifies that a WarmPool row creates the
// region's pool in its estate account by the reason initial, in the
// construction phase, so the money is in the construction bucket and not in tick
// 0, the agents' money is not taken from it, and the audit passes (spec-viability.md
// 8.3.3, FR-EST-015). The table holds 5 000 for each of the 4 agents of weight 1
// (20 000) and a pool of 7 000, so the bucket creates 27 000.
//
// Requirement: FR-EST-015, FR-VIA-032
func TestWarmStart_PoolIsSeededByInitial(t *testing.T) {
	t.Parallel()
	cfg := warmFarmConfig(true)
	cfg.WarmStart.Pools = []WarmPool{{Region: "north", Money: 7_000}}
	w, err := NewWorld(cfg, WithAudit())
	if err != nil {
		t.Fatalf("NewWorld() error = %v, want nil", err)
	}
	if got := w.estates.pool(w.ledger, 0); got != 7_000 {
		t.Errorf("pool of north = %d, want 7 000", got)
	}
	var created int64
	for _, f := range w.ledger.AppendFlows(nil, true) {
		if f.Reason == ledger.Initial && f.Commodity == ledger.Money {
			created += f.Total
		}
	}
	if created != 27_000 || len(w.ledger.AppendFlows(nil, false)) != 0 {
		t.Errorf("construction bucket created %d money and tick 0 has %d flows, want 27 000 (4 * 5 000 + 7 000) and none", created, len(w.ledger.AppendFlows(nil, false)))
	}
}

// TestWarmGroupOf verifies the lookup of the warm-start entry of a (region, role)
// group: the entry whose region and role both equal the names, whatever the
// order of the entries and whichever sorts before or after the names, and nil when
// there is no table or no such entry.
//
// Arithmetic: the table holds (r2, miller), (r1, miller) and (r1, weaver), so the
// first entry sorts after "r1" and the second equals it in region only for the
// miller; a comparison that took "greater than or equal" would return the first
// entry for (r1, miller).
//
// Requirement: FR-VIA-042
func TestWarmGroupOf(t *testing.T) {
	t.Parallel()
	c := &Config{WarmStart: &WarmStart{Groups: []WarmGroup{
		{Region: "r2", Role: "miller"}, {Region: "r1", Role: "miller"}, {Region: "r1", Role: "weaver"},
	}}}
	for _, tt := range []struct {
		region, role string
		want         int // index in Groups, -1 for nil
	}{
		{"r1", "miller", 1}, {"r1", "weaver", 2}, {"r2", "miller", 0},
		{"r2", "weaver", -1}, {"r0", "miller", -1}, {"r3", "miller", -1}, {"r1", "zzz", -1}, {"r1", "aaa", -1},
	} {
		got := warmGroupOf(c, tt.region, tt.role)
		var want *WarmGroup
		if tt.want >= 0 {
			want = &c.WarmStart.Groups[tt.want]
		}
		if got != want {
			t.Errorf("warmGroupOf(%q, %q) = %p, want %p (Groups[%d])", tt.region, tt.role, got, want, tt.want)
		}
	}
	if got := warmGroupOf(&Config{}, "r1", "miller"); got != nil {
		t.Errorf("warmGroupOf(no table) = %v, want nil", got)
	}
}

// TestWarmHelpers_ErrorsReturnZeroValues verifies the rejections of the pure
// helpers of the warm start at each edge of their input, with the sentinel and with
// nothing else returned: apportion refuses a share of -1 or 10 001, shares that sum
// to 10 001 and a negative count, and accepts a share of 10 000, shares that sum to
// exactly 10 000 and a count of 0; phase refuses i = -1, i = n, i beyond n, n = 0
// and p = -1 and accepts i = n - 1 and p = 0; pickOffset refuses an empty set, j =
// -1, j = busy and j beyond busy and accepts j = busy - 1.
//
// Requirement: FR-VIA-034, FR-VIA-037
func TestWarmHelpers_ErrorsReturnZeroValues(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name   string
		shares []int64
		n      int64
		bad    error
	}{
		{"negative share", []int64{-1}, 5, ErrOutOfBounds},
		{"share above the whole", []int64{10_001}, 5, ErrOutOfBounds},
		{"sum above the whole", []int64{6_000, 4_001}, 5, ErrOutOfBounds},
		{"negative count", []int64{5_000}, -1, ErrOutOfBounds},
		{"a whole share", []int64{10_000}, 5, nil},
		{"shares summing to the whole", []int64{6_000, 4_000}, 5, nil},
		{"no agents", []int64{5_000}, 0, nil},
	} {
		got, err := apportion(tt.shares, tt.n)
		if !errors.Is(err, tt.bad) || (tt.bad != nil && got != nil) || (tt.bad == nil && (err != nil || len(got) != len(tt.shares))) {
			t.Errorf("%s: apportion(%v, %d) = (%v, %v), want error %v and a nil slice, or no error and %d counts", tt.name, tt.shares, tt.n, got, err, tt.bad, len(tt.shares))
		}
	}
	for _, tt := range []struct {
		i, n, p int64
		bad     bool
	}{
		{-1, 3, 5, true}, {3, 3, 5, true}, {4, 3, 5, true}, {0, 0, 5, true}, {0, 3, -1, true}, {2, 3, 5, false}, {0, 3, 0, false},
	} {
		got, err := phase(tt.i, tt.n, tt.p)
		if tt.bad && (!errors.Is(err, ErrOutOfBounds) || got != 0) {
			t.Errorf("phase(%d, %d, %d) = (%d, %v), want (0, ErrOutOfBounds)", tt.i, tt.n, tt.p, got, err)
		}
		if !tt.bad && err != nil {
			t.Errorf("phase(%d, %d, %d) error = %v, want nil", tt.i, tt.n, tt.p, err)
		}
	}
	for _, tt := range []struct {
		v       []int64
		j, busy int64
		bad     error
	}{
		{nil, 0, 1, ErrWarmStartJob}, {[]int64{7}, -1, 2, ErrOutOfBounds}, {[]int64{7}, 2, 2, ErrOutOfBounds},
		{[]int64{7}, 3, 2, ErrOutOfBounds}, {[]int64{7, 9}, 1, 2, nil},
	} {
		got, err := pickOffset(tt.v, tt.j, tt.busy)
		if tt.bad != nil && (!errors.Is(err, tt.bad) || got != 0) {
			t.Errorf("pickOffset(%v, %d, %d) = (%d, %v), want (0, %v)", tt.v, tt.j, tt.busy, got, err, tt.bad)
		}
		if tt.bad == nil && (err != nil || got != 9) {
			t.Errorf("pickOffset(%v, %d, %d) = (%d, %v), want (9, nil): V[floor(1 * 2 / 2)]", tt.v, tt.j, tt.busy, got, err)
		}
	}
}

// TestFundWarm_ViolatedInvariants verifies that a warm group the validator would
// have rejected makes the construction panic with an invariant message instead of
// wrapping a number or indexing past a table: money times the weight overflowing, a
// stock item of an unknown commodity, a stock quantity times the weight
// overflowing, the unknown region or commodity of a price entry and the unknown
// region of a pool.
//
// Requirement: FR-VIA-032
func TestFundWarm_ViolatedInvariants(t *testing.T) {
	t.Parallel()
	newBuilder := func(t *testing.T) (*World, *builder) {
		w := newMill(t, removalConfig())
		return w, &builder{w: w, plans: groupPlans(&w.cfg), seeds: make([]beliefSeed, 3)}
	}
	for _, tt := range []struct {
		name string
		run  func(w *World, b *builder)
		want string
	}{
		{"money overflows", func(w *World, b *builder) {
			b.fundWarm(&w.agents[2], &groupPlan{warm: &WarmGroup{Money: math.MaxInt64}}, 0)
		}, "warm money of a group overflows"},
		{"unknown commodity", func(w *World, b *builder) {
			b.fundWarm(&w.agents[2], &groupPlan{warm: &WarmGroup{Stock: []Item{{Commodity: "nope", Quantity: 1}}}}, 0)
		}, "unknown commodity or overflows"},
		{"quantity overflows", func(w *World, b *builder) {
			b.fundWarm(&w.agents[2], &groupPlan{warm: &WarmGroup{Stock: []Item{{Commodity: "grain", Quantity: math.MaxInt64}}}}, 0)
		}, "unknown commodity or overflows"},
		{"price of an unknown region", func(w *World, _ *builder) {
			c := w.cfg
			c.WarmStart = &WarmStart{Prices: []SeedPrice{{Region: "nowhere", Commodity: "grain", Price: 5}}}
			warmPriceTable(&c)
		}, "unknown region or commodity"},
		{"price of an unknown commodity", func(w *World, _ *builder) {
			c := w.cfg
			c.WarmStart = &WarmStart{Prices: []SeedPrice{{Region: "home", Commodity: "nope", Price: 5}}}
			warmPriceTable(&c)
		}, "unknown region or commodity"},
		{"pool of an unknown region", func(w *World, b *builder) {
			w.cfg.WarmStart = &WarmStart{Pools: []WarmPool{{Region: "nowhere", Money: 5}}}
			b.seedPools()
		}, "unknown region"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w, b := newBuilder(t)
			if msg := invariantMessage(func() { tt.run(w, b) }); !strings.Contains(msg, "invariant") || !strings.Contains(msg, tt.want) {
				t.Errorf("%s panicked with %q, want an invariant panic containing %q", tt.name, msg, tt.want)
			}
		})
	}
}

// TestCheckTools_TheJobNeedsOneItemForEachExecution verifies the check of the tool
// stock of a warm job: the agent holds as many items of the tool as the job has
// executions or more and the job passes; one execution more than it holds is
// ErrWarmStartJob, naming the agent, the recipe and the counts.
//
// Arithmetic: the pan world gives each household 1 pan; agent 1 weighs 1, so it holds
// 1 pan; a job of 1 execution passes and a job of 2 executions needs 2 and fails.
//
// Requirement: FR-VIA-052, FR-VIA-053
func TestCheckTools_TheJobNeedsOneItemForEachExecution(t *testing.T) {
	t.Parallel()
	w := newMill(t, panConfig())
	b := &builder{w: w}
	a := &w.agents[1]
	recipe := w.recipes.row(0)
	a.job.recipe = 0
	for _, tt := range []struct {
		executions int64
		bad        bool
	}{{1, false}, {2, true}} {
		a.job.executions = tt.executions
		err := b.checkTools(a, recipe)
		if tt.bad != errors.Is(err, ErrWarmStartJob) || (tt.bad && !strings.Contains(err.Error(), "agent 1")) {
			t.Errorf("checkTools(%d executions) = %v, want ErrWarmStartJob naming agent 1: %v", tt.executions, err, tt.bad)
		}
	}
}
