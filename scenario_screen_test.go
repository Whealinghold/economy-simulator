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
	"github.com/Whealinghold/economy-sim/internal/checked"
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
	screenFlows  = flag.Bool("screen.flows", false, "also print, per 100-tick window, the money paid by buyer role to seller role and commodity, the units short per upkeep commodity and role, and the NoInput and NoTool refusals per recipe (plan 2-27 Phase 1)")
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
	names []string         // role names, to spell the labels
	flows *screenFlowTally // nil unless -screen.flows
}

// screenEstate is the role of a seller that is no agent of a role: the estate.
const screenEstate = economysim.RoleID(-1)

// screenFlowKey is a money flow of one window: the buyer role paid the seller role
// (screenEstate for an estate) for the commodity.
type screenFlowKey struct {
	Window        int64
	Buyer, Seller economysim.RoleID
	Commodity     economysim.CommodityID
}

// screenShortKey is the units short of one role in one window for one upkeep commodity.
type screenShortKey struct {
	Window    int64
	Role      economysim.RoleID
	Commodity economysim.CommodityID
}

// screenRefusal is the households refused by a recipe in one window, by cause.
type screenRefusal struct{ NoInput, NoTool int64 }

// screenFlowTally is what -screen.flows keeps (plan 2-27 Phase 1): the evidence of
// memo 1.1 to 1.4 that a trial is read against.
type screenFlowTally struct {
	money    map[screenFlowKey]int64
	short    map[screenShortKey]int64
	refused  map[[2]int64]screenRefusal // (window, recipe)
	overflow bool                       // a sum did not fit int64: the money rows are then incomplete
}

func newScreenFlowTally() *screenFlowTally {
	return &screenFlowTally{money: map[screenFlowKey]int64{}, short: map[screenShortKey]int64{}, refused: map[[2]int64]screenRefusal{}}
}

// observe adds the flow events of e. roleOf is the role of a live agent; a seller or
// buyer it does not know is the estate (an estate is not an agent of a role).
func (f *screenFlowTally) observe(e economysim.Event, roleOf func(economysim.AgentID) (economysim.RoleID, bool)) {
	switch e := e.(type) {
	case economysim.TradeExecuted:
		buyer, ok := roleOf(e.Buyer)
		if !ok {
			buyer = screenEstate
		}
		seller, ok := roleOf(e.Seller)
		if !ok {
			seller = screenEstate
		}
		k := screenFlowKey{e.Tick / incomeWindow, buyer, seller, e.Commodity}
		v, ok1 := checked.Mul(int64(e.Price), int64(e.Quantity))
		sum, ok2 := checked.Add(f.money[k], v)
		if !ok1 || !ok2 {
			f.overflow = true
			return
		}
		f.money[k] = sum
	case economysim.UpkeepShortfall:
		for _, r := range e.Short {
			f.short[screenShortKey{e.Tick / incomeWindow, e.Role, r.Commodity}] += r.Units
		}
	case economysim.RecipeRefused:
		k := [2]int64{e.Tick / incomeWindow, int64(e.Recipe)}
		c := f.refused[k]
		c.NoInput += e.NoInput
		c.NoTool += e.NoTool
		f.refused[k] = c
	}
}

// screenRoleName spells a role of a flow, the estate included.
func screenRoleName(names []string, r economysim.RoleID) string {
	if r == screenEstate {
		return "estate"
	}
	return names[r]
}

// report logs the tally of one seed in window, role and commodity order, integers only.
func (f *screenFlowTally) report(t *testing.T, seed uint64, n economysim.Names) {
	t.Helper()
	if f.overflow {
		t.Logf("seed %d flows: a sum overflowed int64, the money rows are incomplete", seed)
	}
	mk := make([]screenFlowKey, 0, len(f.money))
	for k := range f.money {
		mk = append(mk, k)
	}
	slices.SortFunc(mk, func(a, b screenFlowKey) int {
		return cmp.Or(cmp.Compare(a.Window, b.Window), cmp.Compare(a.Buyer, b.Buyer), cmp.Compare(a.Seller, b.Seller), cmp.Compare(a.Commodity, b.Commodity))
	})
	for _, k := range mk {
		v := f.money[k]
		t.Logf("seed %d window %d flow %s -> %s on %s: money %d (%d a tick)", seed, k.Window, screenRoleName(n.Roles, k.Buyer), screenRoleName(n.Roles, k.Seller), n.Commodities[k.Commodity], v, v/incomeWindow)
	}
	sk := make([]screenShortKey, 0, len(f.short))
	for k := range f.short {
		sk = append(sk, k)
	}
	slices.SortFunc(sk, func(a, b screenShortKey) int {
		return cmp.Or(cmp.Compare(a.Window, b.Window), cmp.Compare(a.Role, b.Role), cmp.Compare(a.Commodity, b.Commodity))
	})
	for _, k := range sk {
		t.Logf("seed %d window %d short role %s on %s: %d units", seed, k.Window, n.Roles[k.Role], n.Commodities[k.Commodity], f.short[k])
	}
	rk := make([][2]int64, 0, len(f.refused))
	for k := range f.refused {
		rk = append(rk, k)
	}
	slices.SortFunc(rk, func(a, b [2]int64) int { return cmp.Or(cmp.Compare(a[0], b[0]), cmp.Compare(a[1], b[1])) })
	for _, k := range rk {
		c := f.refused[k]
		t.Logf("seed %d window %d refused recipe %s: no input %d, no tool %d household-ticks", seed, k[0], n.Recipes[k[1]], c.NoInput, c.NoTool)
	}
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
	if o.flows != nil {
		o.flows.observe(e, func(a economysim.AgentID) (economysim.RoleID, bool) {
			ag, ok := o.inc.agents[a]
			return ag.role, ok
		})
	}
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
	if *screenFlows {
		out.Watch.flows = newScreenFlowTally()
	}
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
	runScenario(t, cfg, spec)
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
	if len(cfg.Regions) != 1 {
		t.Fatalf("screen.reseed adds to region 0 only: the pack has %d regions, want 1", len(cfg.Regions))
	}
	for range re.Count {
		if _, err := w.AddAgent(economysim.NewAgent{Region: 0, Role: economysim.RoleID(role), Strategy: "default", Money: economysim.Money(re.Money), Stock: slices.Clone(stock)}); err != nil {
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
	for role := range o.Names.Roles { // role order, not map order, so two runs print the same text
		if tick, ok := o.Watch.extinct[economysim.RoleID(role)]; ok {
			t.Logf("seed %d: role %s extinct from tick %d", seed, o.Names.Roles[role], tick)
		}
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
	if len(cfg.Regions) > 1 {
		t.Logf("seed %d WARNING: %d regions, the margins below use the basket of region 0 for every role", seed, len(cfg.Regions))
	}
	for window := int64(0); window < ticks/incomeWindow+1; window++ {
		for role, id := range o.Names.Roles {
			row := o.Watch.inc.Row(window, economysim.RoleID(role), upkeep)
			if !row.OK {
				continue
			}
			c := o.Watch.inc.cells[[2]int64{window, int64(role)}]
			t.Logf("seed %d window %d role %s: sales %d and recipe inputs %d per household-tick, margin %d bp of the upkeep %d, household-ticks %d, unfed %d, executions %s",
				seed, window, id, row.SalesPerHousehold, row.InputCostPerHousehold, row.MarginBP, upkeep, c.HouseholdTicks, c.Unfed, screenExecutions(c.Executions, o.Names.Recipes))
		}
	}
	if o.Watch.flows != nil {
		o.Watch.flows.report(t, seed, o.Names)
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
	if last := at[len(at)-1]; last > ticks {
		t.Fatalf("-screen.at has tick %d after the last tick run (-screen.ticks %d)", last, ticks)
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
			if !*screenX4 { // x4 keeps a snapshot of every tick: one seed at a time bounds the memory
				t.Parallel()
			}
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

// TestScreen_FlowTally verifies the arithmetic of -screen.flows: the window bucketing,
// the estate rule, and the sums of the shortfall and of the refusals.
//
// Hand-built events, window 100 ticks, roles 0 (cook) and 1 (maker), agent 1 a cook and
// agent 2 a maker, agent 9 unknown (the estate): a trade of 3 units at 20 from 2 to 1 at
// tick 99 is 60 in window 0, and the same trade at tick 100 is 60 in window 1; a trade of
// 2 at 5 from 9 (estate) to 1 is 10; one from 1 to 2 of 4 at 10 is 40 and, added to a
// second of 1 at 10, 50. Short rows 7 + 5 = 12 units of commodity 0 for role 1 in window 0
// (two events) and 4 of commodity 1. Refusals no input 2 + 3 = 5 and no tool 1 + 0 = 1.
//
// Requirement: FR-REF-039
func TestScreen_FlowTally(t *testing.T) {
	t.Parallel()
	roleOf := func(a economysim.AgentID) (economysim.RoleID, bool) {
		switch a {
		case 1:
			return 0, true
		case 2:
			return 1, true
		}
		return 0, false
	}
	f := newScreenFlowTally()
	for _, e := range []economysim.Event{
		economysim.TradeExecuted{Tick: 99, Buyer: 1, Seller: 2, Commodity: 0, Price: 20, Quantity: 3},
		economysim.TradeExecuted{Tick: 100, Buyer: 1, Seller: 2, Commodity: 0, Price: 20, Quantity: 3},
		economysim.TradeExecuted{Tick: 5, Buyer: 1, Seller: 9, Commodity: 1, Price: 5, Quantity: 2},
		economysim.TradeExecuted{Tick: 6, Buyer: 2, Seller: 1, Commodity: 0, Price: 10, Quantity: 4},
		economysim.TradeExecuted{Tick: 7, Buyer: 2, Seller: 1, Commodity: 0, Price: 10, Quantity: 1},
		economysim.UpkeepShortfall{Tick: 10, Role: 1, Short: []economysim.ShortRow{{Commodity: 0, Units: 7}, {Commodity: 1, Units: 4}}},
		economysim.UpkeepShortfall{Tick: 11, Role: 1, Short: []economysim.ShortRow{{Commodity: 0, Units: 5}}},
		economysim.RecipeRefused{Tick: 3, Recipe: 2, NoInput: 2, NoTool: 1},
		economysim.RecipeRefused{Tick: 4, Recipe: 2, NoInput: 3},
	} {
		f.observe(e, roleOf)
	}
	for _, tt := range []struct {
		name      string
		got, want int64
	}{
		{"window 0 maker to cook", f.money[screenFlowKey{0, 0, 1, 0}], 60},
		{"window 1 maker to cook", f.money[screenFlowKey{1, 0, 1, 0}], 60},
		{"estate to cook", f.money[screenFlowKey{0, 0, screenEstate, 1}], 10},
		{"cook to maker, two trades", f.money[screenFlowKey{0, 1, 0, 0}], 50},
		{"short role 1 commodity 0", f.short[screenShortKey{0, 1, 0}], 12},
		{"short role 1 commodity 1", f.short[screenShortKey{0, 1, 1}], 4},
		{"no input", f.refused[[2]int64{0, 2}].NoInput, 5},
		{"no tool", f.refused[[2]int64{0, 2}].NoTool, 1},
	} {
		if tt.got != tt.want {
			t.Errorf("%s = %d, want %d", tt.name, tt.got, tt.want)
		}
	}
	if f.overflow {
		t.Error("overflow = true after small trades, want false")
	}
	f.observe(economysim.TradeExecuted{Tick: 6, Buyer: 2, Seller: 1, Commodity: 0, Price: 1 << 62, Quantity: 4}, roleOf)
	if !f.overflow {
		t.Error("overflow = false after a product beyond int64, want true")
	}
}
