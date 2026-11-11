package economysim_test

import (
	"cmp"
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/config"
)

// This file is the survival-screen harness of plan 2-25 Phase 1 (D700 to D703). It
// runs a pack, with any number of overlay layers on top, and prints at fixed ticks
// the households, heads and unfed households of each role, the share of the money
// each role holds, and the price of each market against its reference; with
// -screen.x4 it also prints the X4 criteria, and always the role waterfall of
// plan 2-8 D364 (incomeObs). It reuses runScenario, incomeObs, stablePredicate and
// RoleMoneyForTest. It is test code and changes no pack (architecture 2.10.5).
//
// The flags follow indicator/stylised_helpers_test.go: no environment variable
// (N-340). With none of them given TestScreen_Packs is skipped, so it never runs in
// a gate; the trial ladder of plan 2-25 D704 runs it by hand:
//
//	go test -run TestScreen_Packs -count=1 -v -timeout 30m \
//	    -screen.pack=one-demand -screen.seeds=101-104 -screen.layers=/tmp/trial1.toml
var (
	screenPack   = flag.String("screen.pack", "", "pack to screen: one, one-demand, five, all, or a comma list of layer files under testdata/chains")
	screenLayers = flag.String("screen.layers", "", "comma list of layers put over the pack; one that starts with / is read from that path (a trial outside the repository)")
	screenSeeds  = flag.String("screen.seeds", "1-1", "seed range a-b")
	screenTicks  = flag.Int64("screen.ticks", 0, "ticks to run; 0 is the last of -screen.at")
	screenAt     = flag.String("screen.at", "100,300,600,1100", "comma list of ticks at which the table is printed")
	screenReseed = flag.String("screen.reseed", "", "read-only H4 experiment role:count:money: when the role has no household, add count external agents of weight 1 to region 0 (AddAgent), with the role's start stock and that money each")
	screenDrop   = flag.String("screen.drop", "", "comma list of layers of the pack left out (a pack at the 16 document limit takes an overlay in place of one)")
	screenCensus = flag.Bool("screen.census", false, "also print the census of the solver for the layers without the warm start: households, balanced households and static income per role (FR-VIA-060)")
	screenX4     = flag.Bool("screen.x4", false, "also print the X4 criteria (needs -screen.ticks 1095 or more; keeps a snapshot of every tick)")
)

// screenPacks are the packs -screen.pack names. The all-on pack needs the options of
// its strategies (allOptions).
var screenPacks = map[string]struct {
	layers  []string
	options func() []economysim.Option
}{
	"one":        {packOne, nil},
	"one-demand": {packOneDemand, nil},
	"five":       {packFiveWarm, nil},
	"all":        {allLayers, allOptions},
}

// screenRole is what the table keeps of one role at a tick, summed over regions.
type screenRole struct{ Households, Agents, Unfed int64 }

// screenRoles sums the (region, role) rows of s into one row per role, n roles.
func screenRoles(s economysim.Snapshot, n int) []screenRole {
	out := make([]screenRole, n)
	for _, r := range s.Roles {
		if int(r.Role) >= n {
			continue
		}
		out[r.Role].Households += r.Households
		out[r.Role].Agents += r.Agents
		out[r.Role].Unfed += r.Unfed
	}
	return out
}

// screenShares returns each entry as basis points of the total, half to even, and -1
// for every entry when the total is not above 0 (there is no share of nothing).
func screenShares(money []int64) []int64 {
	var total int64
	for _, m := range money {
		total += m
	}
	out := make([]int64, len(money))
	for i, m := range money {
		out[i] = -1
		if total > 0 {
			if bp, ok := shareBP(m, total); ok {
				out[i] = bp
			}
		}
	}
	return out
}

// screenInts reads a comma list of ticks, each at least 1, in ascending order.
func screenInts(spec string) ([]int64, error) {
	var out []int64
	for _, f := range strings.Split(spec, ",") {
		v, err := strconv.ParseInt(strings.TrimSpace(f), 10, 64)
		if err != nil || v < 1 || (len(out) > 0 && v <= out[len(out)-1]) {
			return nil, fmt.Errorf("screen ticks %q: want ascending integers of at least 1", spec)
		}
		out = append(out, v)
	}
	return out, nil
}

// screenSeedRange reads a-b (or a single seed) into the list of seeds.
func screenSeedRange(spec string) ([]uint64, error) {
	lo, hi, found := strings.Cut(spec, "-")
	if !found {
		hi = lo
	}
	a, err1 := strconv.ParseUint(lo, 10, 64)
	b, err2 := strconv.ParseUint(hi, 10, 64)
	if err1 != nil || err2 != nil || a < 1 || b < a || b-a > 1000 {
		return nil, fmt.Errorf("screen seeds %q: want a-b with 1 <= a <= b", spec)
	}
	out := make([]uint64, 0, b-a+1)
	for s := a; s <= b; s++ {
		out = append(out, s)
	}
	return out, nil
}

// screenRecorded reads the line "<prefix>: 1=557 2=538 | median=..." of a scenario
// record into seed -> value.
func screenRecorded(text, prefix string) map[uint64]int64 {
	for _, line := range strings.Split(text, "\n") {
		rest, ok := strings.CutPrefix(line, prefix+": ")
		if !ok {
			continue
		}
		rest, _, _ = strings.Cut(rest, " |")
		out := map[uint64]int64{}
		for _, kv := range strings.Fields(rest) {
			k, v, _ := strings.Cut(kv, "=")
			seed, err1 := strconv.ParseUint(k, 10, 64)
			val, err2 := strconv.ParseInt(v, 10, 64)
			if err1 == nil && err2 == nil {
				out[seed] = val
			}
		}
		return out
	}
	return nil
}

// screenConfig loads the layers, one that starts with / from that path and the others
// under testdata/chains, validates the result and returns it.
func screenConfig(t testing.TB, layers []string) economysim.Config {
	t.Helper()
	docs := make([][]byte, 0, len(layers))
	for _, l := range layers {
		p := l
		if !filepath.IsAbs(l) {
			p = filepath.Join(chainsDir, l)
		}
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatalf("read layer %s: %v", l, err)
		}
		docs = append(docs, b)
	}
	cfg, err := config.Parse(docs...)
	if err != nil {
		t.Fatalf("Parse(%q) error = %v, want nil", layers, err)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate(%q) error = %v, want nil", layers, err)
	}
	return cfg
}

// screenWatch is the observer of the screen: it feeds incomeObs and keeps the first
// tick at which a role that had households has none.
type screenWatch struct {
	inc     *incomeObs
	seen    map[economysim.RoleID]bool
	extinct map[economysim.RoleID]int64
	// moves counts households by (window, role, label): "to <role>" and "from <role>"
	// for a role change, "exit <cause>" for a removal or a cut of weight, "entry" for
	// a household that joined after the start.
	moves map[screenMove]int64
	names []string // role names, to spell the labels
}

// screenMove is the key of screenWatch.moves.
type screenMove struct {
	Window int64
	Role   economysim.RoleID
	Label  string
}

// move adds households to the ledger of the window of tick.
func (o *screenWatch) move(tick int64, role economysim.RoleID, households int64, label string) {
	o.moves[screenMove{tick / incomeWindow, role, label}] += households
}

func (o *screenWatch) Observe(e economysim.Event) {
	o.inc.Observe(e)
	switch e := e.(type) {
	case economysim.RoleChanged:
		o.move(e.Tick, e.From, e.Weight, fmt.Sprintf("to %s (%v)", o.names[e.To], e.Cause))
		o.move(e.Tick, e.To, e.Weight, fmt.Sprintf("from %s (%v)", o.names[e.From], e.Cause))
	case economysim.AgentBankrupted:
		o.move(e.Tick, e.Role, e.Weight, fmt.Sprintf("exit %v", e.Cause))
	case economysim.WeightChanged:
		if e.To < e.From {
			o.move(e.Tick, e.Role, e.From-e.To, fmt.Sprintf("exit %v", e.Cause))
		} else {
			o.move(e.Tick, e.Role, e.To-e.From, fmt.Sprintf("grew %v", e.Cause))
		}
	case economysim.AgentSpawned:
		if e.Tick > 0 && !e.External { // the initial households carry tick 0
			o.move(e.Tick, e.Role, e.Weight, "entry")
		}
	}
	if e, ok := e.(economysim.TickEnded); ok {
		for role, hh := range o.inc.live {
			switch {
			case hh > 0:
				o.seen[role] = true
			case o.seen[role]:
				if _, done := o.extinct[role]; !done {
					o.extinct[role] = e.Tick
				}
			}
		}
	}
}

// screenPoint is the table of one tick.
type screenPoint struct {
	At         int64
	Households int64
	Roles      []screenRole
	MoneyBP    []int64
	Prices     []string // "region/commodity price (bp of reference)"
}

// screenOutcome is what one seed of a screen run kept.
type screenOutcome struct {
	Points  []screenPoint
	Names   economysim.Names
	Recipes []string
	Watch   *screenWatch
	Samples map[int64]economysim.Snapshot // every tick, only with x4
}

// reseedPlan is the H4 experiment of plan 2-25 D703: it undoes the absorbing
// extinction of SPN-A8 from test code, through the public AddAgent, to see whether
// the economy lives on once the role exists. It is a probe, never a fix: the
// agents are external and immortal.
type reseedPlan struct {
	Role         string
	Count, Money int64
}

// screenReseedSpec reads role:count:money.
func screenReseedSpec(spec string) (*reseedPlan, error) {
	if spec == "" {
		return nil, nil
	}
	f := strings.Split(spec, ":")
	if len(f) != 3 {
		return nil, fmt.Errorf("screen.reseed %q: want role:count:money", spec)
	}
	n, err1 := strconv.ParseInt(f[1], 10, 64)
	m, err2 := strconv.ParseInt(f[2], 10, 64)
	if err1 != nil || err2 != nil || n < 1 || m < 0 {
		return nil, fmt.Errorf("screen.reseed %q: want a count of at least 1 and money of at least 0", spec)
	}
	return &reseedPlan{f[0], n, m}, nil
}

// screenRun runs cfg for ticks steps on seed and takes a table at each tick of at.
// A non-nil re adds agents of its role whenever the role has none.
func screenRun(t testing.TB, cfg economysim.Config, opts []economysim.Option, seed uint64, ticks int64, at []int64, x4 bool, re *reseedPlan) screenOutcome {
	t.Helper()
	probe, err := economysim.NewWorld(cfg, opts...)
	if err != nil {
		t.Fatalf("NewWorld error = %v, want nil", err)
	}
	s0, err := probe.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot error = %v, want nil", err)
	}
	out := screenOutcome{Names: s0.Names, Samples: map[int64]economysim.Snapshot{}, Watch: &screenWatch{
		inc: newIncomeObs(cfg, s0.Names), seen: map[economysim.RoleID]bool{}, extinct: map[economysim.RoleID]int64{}, moves: map[screenMove]int64{}, names: s0.Names.Roles}}
	spec := scenarioSpec{Seed: seed, Ticks: ticks, Options: append(slices.Clone(opts), economysim.WithObserver(out.Watch)),
		Probe: func(tick int64, w *economysim.World) {
			if re != nil {
				screenAddAgents(t, cfg, s0.Names, out.Watch, w, tick, re)
			}
			if !slices.Contains(at, tick+1) {
				return
			}
			s, err := w.Snapshot()
			if err != nil {
				t.Fatalf("Snapshot after tick %d error = %v, want nil", tick, err)
			}
			p := screenPoint{At: tick + 1, Households: s.Households, Roles: screenRoles(s, len(s.Names.Roles)),
				MoneyBP: screenShares(economysim.RoleMoneyForTest(w, len(s.Names.Roles)))}
			for _, m := range s.Markets {
				ref := int64(w.ReferencePrice(m.State.Region, m.State.Commodity))
				bp := "-"
				if v, ok := shareBP(int64(m.State.Price), ref); ok {
					bp = strconv.FormatInt(v, 10)
				}
				p.Prices = append(p.Prices, fmt.Sprintf("%s/%s %d (%s bp of %d), volume %d, held by agents %d and estate %d", s.Names.Regions[m.State.Region], s.Names.Commodities[m.State.Commodity], m.State.Price, bp, ref, m.State.Volume, m.HeldByAgents, m.HeldByEstate))
			}
			out.Points = append(out.Points, p)
		}}
	if x4 {
		spec.Snap = func(int64) bool { return true }
		spec.OnSnap = func(tick int64, s economysim.Snapshot) { out.Samples[tick] = s }
	}
	run := runScenario(t, cfg, spec)
	out.Names = run.Names
	out.Recipes = run.Names.Recipes
	return out
}

// screenAddAgents is the body of the reseed probe: when the role has no household,
// it adds re.Count agents of weight 1 to region 0 with the role's start stock.
func screenAddAgents(t testing.TB, cfg economysim.Config, names economysim.Names, watch *screenWatch, w *economysim.World, tick int64, re *reseedPlan) {
	t.Helper()
	role := indexOf(names.Roles, re.Role)
	if role < 0 {
		t.Fatalf("screen.reseed: role %q is not in the names of the world", re.Role)
	}
	if !watch.seen[economysim.RoleID(role)] || watch.inc.live[economysim.RoleID(role)] > 0 {
		return
	}
	var stock []economysim.StockItem
	for _, it := range cfg.Roles[role].StartStock {
		stock = append(stock, economysim.StockItem{Commodity: economysim.CommodityID(indexOf(names.Commodities, it.Commodity)), Quantity: it.Quantity})
	}
	for range re.Count {
		if _, err := w.AddAgent(economysim.NewAgent{Region: 0, Role: economysim.RoleID(role), Strategy: "default", Money: economysim.Money(re.Money), Stock: stock}); err != nil {
			t.Fatalf("AddAgent(%s) after tick %d error = %v, want nil", re.Role, tick, err)
		}
	}
	t.Logf("reseed: %d agents of %s added after tick %d", re.Count, re.Role, tick)
}

// screenCensusLog prints the static view of the solver for the cold layers: the census
// rows with their households, the balanced households and the income as basis points of
// the upkeep, the figure that TestPacks_IncomeBalance gates (plan 2-8 D365).
func screenCensusLog(t *testing.T, cold []string) {
	t.Helper()
	st, err := economysim.SolveSteadyState(context.Background(), screenConfig(t, cold))
	if err != nil {
		t.Fatalf("SolveSteadyState(%q) error = %v, want nil", cold, err)
	}
	for _, rs := range st.Report.RegionSurplusBP {
		t.Logf("census surplus %s: autarky %d bp (the labour share to subsist is 10000 * 10000 / this)", rs.Region, rs.AutarkyBP)
	}
	for _, r := range st.Report.Census {
		if r.Role == "" {
			continue
		}
		income := "no income figure"
		if r.HasIncome {
			income = fmt.Sprintf("income %d bp of the upkeep", r.IncomeBP)
		}
		t.Logf("census %s/%s (%s): households %d, balanced %d, imbalance %d, short %v, %s", r.Region, r.Role, r.Recipe, r.Households, r.Balanced, r.Imbalance, r.Short, income)
	}
}

// screenExecutions spells the executions of a cell by recipe name, in recipe order.
func screenExecutions(ex map[economysim.RecipeID]int64, names []string) string {
	ids := make([]economysim.RecipeID, 0, len(ex))
	for id := range ex {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	parts := make([]string, len(ids))
	for i, id := range ids {
		parts[i] = fmt.Sprintf("%s=%d", names[id], ex[id])
	}
	return strings.Join(parts, " ")
}

// screenReport logs the tables of one seed and, if given, the waterfall and the X4
// criteria.
func screenReport(t *testing.T, cfg economysim.Config, seed uint64, o screenOutcome, ticks int64, x4 bool) {
	t.Helper()
	var start int64
	for _, g := range cfg.Groups {
		start += g.Households
	}
	for _, p := range o.Points {
		var b strings.Builder
		fmt.Fprintf(&b, "seed %d tick %d: households %d of %d at the start\n", seed, p.At, p.Households, start)
		for role, r := range p.Roles {
			money := "no money"
			if p.MoneyBP[role] >= 0 {
				money = fmt.Sprintf("%d bp of the money", p.MoneyBP[role])
			}
			fmt.Fprintf(&b, "    role %-10s households %5d heads %4d unfed %5d, %s\n", o.Names.Roles[role], r.Households, r.Agents, r.Unfed, money)
		}
		for _, pr := range p.Prices {
			fmt.Fprintf(&b, "    price %s\n", pr)
		}
		t.Log(b.String())
	}
	for role, tick := range o.Watch.extinct {
		t.Logf("seed %d: role %s extinct from tick %d", seed, o.Names.Roles[role], tick)
	}
	keys := make([]screenMove, 0, len(o.Watch.moves))
	for k := range o.Watch.moves {
		keys = append(keys, k)
	}
	slices.SortFunc(keys, func(a, b screenMove) int {
		return cmp.Or(cmp.Compare(a.Window, b.Window), cmp.Compare(a.Role, b.Role), strings.Compare(a.Label, b.Label))
	})
	for _, k := range keys {
		t.Logf("seed %d window %d role %s: %s %d households", seed, k.Window, o.Names.Roles[k.Role], k.Label, o.Watch.moves[k])
	}
	// shortcut: region 0's basket for every role and window, right for the single-region packs; key it by region before reading it for five.
	upkeep := int64(cfg.Regions[0].ReferenceBasket)
	for window := int64(0); window < ticks/incomeWindow+1; window++ {
		for role, id := range o.Names.Roles {
			row := o.Watch.inc.Row(window, economysim.RoleID(role), upkeep)
			if !row.OK {
				continue
			}
			c := o.Watch.inc.cells[[2]int64{window, int64(role)}]
			t.Logf("seed %d window %d role %s: sales %d and recipe inputs %d per household-tick, margin %d bp of the upkeep %d, household-ticks %d, unfed %d, executions %s",
				seed, window, id, row.SalesPerHousehold, row.InputCostPerHousehold, row.MarginBP, upkeep, c.HouseholdTicks, c.Unfed, screenExecutions(c.Executions, o.Recipes))
		}
	}
	if x4 {
		res := result{Samples: o.Samples, Ticks: ticks}
		for _, c := range stablePredicate(newStableRun(cfg, res), x4Priors) {
			t.Logf("seed %d X4 (%s) %s: pass=%v asserted=%v: %s", seed, c.Key, c.Name, c.Pass, c.Asserted, c.Detail)
		}
	}
}

// TestScreen_Packs prints the survival table of a pack: the flags are in the comment
// of this file. It is skipped unless -screen.pack is given, so no gate runs it, and
// with -short. It asserts nothing: the ladder of plan 2-25 D704 reads its output.
//
// Requirement: FR-REF-039
func TestScreen_Packs(t *testing.T) {
	if testing.Short() {
		t.Skip("diagnostic harness, skipped with -short")
	}
	if *screenPack == "" {
		t.Skip("diagnostic harness: give -screen.pack to run it (plan 2-25 D704)")
	}
	pk, ok := screenPacks[*screenPack]
	if !ok {
		pk.layers = strings.Split(*screenPack, ",")
	}
	layers := slices.DeleteFunc(slices.Clone(pk.layers), func(l string) bool {
		return *screenDrop != "" && slices.Contains(strings.Split(*screenDrop, ","), l)
	})
	if *screenLayers != "" {
		layers = append(layers, strings.Split(*screenLayers, ",")...)
	}
	at, err := screenInts(*screenAt)
	if err != nil {
		t.Fatal(err)
	}
	seeds, err := screenSeedRange(*screenSeeds)
	if err != nil {
		t.Fatal(err)
	}
	ticks := *screenTicks
	if ticks == 0 {
		ticks = at[len(at)-1]
	}
	if *screenX4 && ticks < x4Ticks {
		t.Fatalf("-screen.x4 needs -screen.ticks >= %d, got %d", x4Ticks, ticks)
	}
	cfg := screenConfig(t, layers)
	var opts []economysim.Option
	if pk.options != nil {
		opts = pk.options()
	}
	re, err := screenReseedSpec(*screenReseed)
	if err != nil {
		t.Fatal(err)
	}
	if *screenCensus {
		screenCensusLog(t, slices.DeleteFunc(slices.Clone(layers), func(l string) bool { return strings.HasPrefix(filepath.Base(l), "warm-") }))
	}
	t.Logf("screen pack %v, %d seeds, %d ticks, at %v", layers, len(seeds), ticks, at)
	for _, seed := range seeds {
		t.Run("seed"+strconv.FormatUint(seed, 10), func(t *testing.T) {
			t.Parallel()
			screenReport(t, cfg, seed, screenRun(t, cfg, opts, seed, ticks, at, *screenX4, re), ticks, *screenX4)
		})
	}
}

// reproduceTicks are the ticks at which pack one's households are compared with
// the record of S04 (s04DemandTicks).
var reproduceTicks = []int64{100, 300, 600}

// TestScreen_Reproduces verifies that the screen harness reproduces the households of
// pack one at ticks 100, 300 and 600 on seeds 1 to 3 that
// testdata/chains/expected/one-demand.scenarios.txt records for "wants off (pack
// one)" (557, 193 and 45 on seed 1), so a harness that drifts from the engine, or a
// pack that moved without the record, is seen.
//
// Requirement: FR-REF-039
//
// Guard: plan 2-25 D700 quotes those households as the evidence of the collapse; a
// harness that cannot reproduce them measures something else.
func TestScreen_Reproduces(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test (600 ticks, three seeds), skipped with -short")
	}
	rec, err := os.ReadFile(filepath.Join(chainsDir, "expected", "one-demand.scenarios.txt"))
	if err != nil {
		t.Fatalf("read the scenario record: %v", err)
	}
	cfg := loadValidPack(t, packOne...)
	for _, seed := range []uint64{1, 2, 3} {
		t.Run("seed"+strconv.FormatUint(seed, 10), func(t *testing.T) {
			t.Parallel()
			o := screenRun(t, cfg, nil, seed, 600, reproduceTicks, false, nil)
			if len(o.Points) != len(reproduceTicks) {
				t.Fatalf("screenRun(pack one, seed %d) kept %d tables, want %d", seed, len(o.Points), len(reproduceTicks))
			}
			for i, tick := range reproduceTicks {
				want := screenRecorded(string(rec), fmt.Sprintf("measure households at tick %d, wants off (pack one)", tick))[seed]
				if got := o.Points[i].Households; want == 0 || got != want {
					t.Errorf("households of pack one at tick %d, seed %d = %d, want %d (the record of S04)", tick, seed, got, want)
				}
			}
		})
	}
}

// TestScreen_Helpers verifies the arithmetic of the screen: the roles summed over
// regions, the money shares, and the readers of the flags and of the record.
//
// Hand-built snapshot: role 0 has 30 households, 3 heads and 2 unfed in region 0
// and 10, 1 and 0 in region 1, so 40, 4 and 2; role 1 has 5, 1 and 5 in region 0.
// Shares: money 300, 100 and 0 of a total 400 are 7 500, 2 500 and 0 bp; 1 of 3 is
// 3 333.33, which rounds to 3 333 and 2 of 3 to 6 667 (half to even).
//
// Requirement: FR-REF-039
func TestScreen_Helpers(t *testing.T) {
	t.Parallel()
	s := economysim.Snapshot{Roles: []economysim.RoleRow{
		{Region: 0, Role: 0, Agents: 3, Households: 30, Unfed: 2},
		{Region: 1, Role: 0, Agents: 1, Households: 10},
		{Region: 0, Role: 1, Agents: 1, Households: 5, Unfed: 5},
		{Region: 0, Role: 9, Agents: 1, Households: 1}, // outside n: ignored
	}}
	if got, want := screenRoles(s, 2), []screenRole{{40, 4, 2}, {5, 1, 5}}; !slices.Equal(got, want) {
		t.Errorf("screenRoles(s, 2) = %v, want %v", got, want)
	}
	for _, tt := range []struct {
		name  string
		money []int64
		want  []int64
	}{
		{"quarters", []int64{300, 100, 0}, []int64{7_500, 2_500, 0}},
		{"thirds", []int64{1, 2}, []int64{3_333, 6_667}},
		{"no money", []int64{0, 0}, []int64{-1, -1}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := screenShares(tt.money); !slices.Equal(got, tt.want) {
				t.Errorf("screenShares(%v) = %v, want %v", tt.money, got, tt.want)
			}
		})
	}
	if got, err := screenInts("100,300"); err != nil || !slices.Equal(got, []int64{100, 300}) {
		t.Errorf("screenInts(100,300) = %v, %v, want [100 300], nil", got, err)
	}
	for _, bad := range []string{"", "0", "5,5", "9,3", "x"} {
		if got, err := screenInts(bad); err == nil {
			t.Errorf("screenInts(%q) = %v, nil, want an error", bad, got)
		}
	}
	if got, err := screenSeedRange("101-104"); err != nil || !slices.Equal(got, []uint64{101, 102, 103, 104}) {
		t.Errorf("screenSeedRange(101-104) = %v, %v, want 101 to 104, nil", got, err)
	}
	if got, err := screenSeedRange("7"); err != nil || !slices.Equal(got, []uint64{7}) {
		t.Errorf("screenSeedRange(7) = %v, %v, want [7], nil", got, err)
	}
	for _, bad := range []string{"", "0-3", "5-3", "a-b"} {
		if got, err := screenSeedRange(bad); err == nil {
			t.Errorf("screenSeedRange(%q) = %v, nil, want an error", bad, got)
		}
	}
	rec := "measure households at tick 100, wants off (pack one): 1=557 2=538 | median=547 n=2\nmeasure other: 1=1"
	if got := screenRecorded(rec, "measure households at tick 100, wants off (pack one)"); got[1] != 557 || got[2] != 538 || len(got) != 2 {
		t.Errorf("screenRecorded(record) = %v, want map[1:557 2:538]", got)
	}
	if re, err := screenReseedSpec("grower:3:500"); err != nil || *re != (reseedPlan{"grower", 3, 500}) {
		t.Errorf("screenReseedSpec(grower:3:500) = %v, %v, want {grower 3 500}, nil", re, err)
	}
	if re, err := screenReseedSpec(""); err != nil || re != nil {
		t.Errorf("screenReseedSpec(empty) = %v, %v, want nil, nil", re, err)
	}
	for _, bad := range []string{"grower", "grower:0:5", "grower:2:-1", "grower:x:1"} {
		if re, err := screenReseedSpec(bad); err == nil {
			t.Errorf("screenReseedSpec(%q) = %v, nil, want an error", bad, re)
		}
	}
	if got := screenRecorded(rec, "absent"); got != nil {
		t.Errorf("screenRecorded(record, absent) = %v, want nil", got)
	}
}
