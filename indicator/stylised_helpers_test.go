package indicator_test

import (
	"context"
	"flag"
	"fmt"
	"maps"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/indicator"
	"github.com/Whealinghold/economy-sim/internal/verdict"
	"github.com/Whealinghold/economy-sim/strategy"
)

// The helpers of the stylised fact tests (plan 2-19 D594, D595; STY 9). The
// recorder, loadPack and stepWindows of run_test.go, raceEnabled and updateFlag
// are plan 2-18's and are reused: this package defines each of them once (O5).

// The flags of the tiers (D595). They are test constructs: no environment
// variable and no file is read (testing.md 10.2).
var (
	stylisedTier    = flag.String("stylised.tier", "g", "the tier of the fact tests: g (seeds 1 to 10) or n (seeds 1 to 50)")
	stylisedRelease = flag.Bool("stylised.release", false, "require a value for every key of STY 8.1.1")
	stylisedPropose = flag.Bool("stylised.propose", false, "render the proposal of a setting run (STY 8.4); it never writes a value")
	stylisedReport  = flag.Bool("stylised.report", false, "render a run and assert nothing; the only mode that may read the hold-out seeds 51 to 60")
	stylisedSeeds   = flag.String("stylised.seeds", "", "the seed range a-b of a run; the default is the tier's")
)

// The run lengths of REF 8.2 (D595). Tier L is the only user of a longer run.
const (
	gateTicks    int64 = 1095
	nightlyTicks int64 = gateTicks
	yearTicks    int64 = 365
)

// The priors that stand in for an unset key as labelled working thresholds
// (STY 6.5.1, 4.3): they are used to read a measure, never to assert.
const (
	workPinnedBP  = 5000 // r5.pinned_bp
	workActiveMin = 5000 // active_min_bp
	workStaleMax  = 1000 // stale_max_bp
)

// issueCollapse is the open issue of the collapse of pack one under its own
// rules (N-312, N-314): a fact that needs a living economy is a finding, not a
// pass and not a failure of the test.
const issueCollapse = "N-314"

// seedRange reads the seed range of a run (D595). The default is the tier's:
// 1 to 10 for g, 1 to 50 for n. A flag value is a-b; the hold-out 51 to 60 is
// allowed only with report, and no range is allowed with propose, because a
// proposal is made from the setting seeds only (STY 8.2, 8.4, FR-STY-087).
func seedRange(spec, tier string, propose, report bool) (first, last uint64, err error) {
	if spec == "" {
		switch tier {
		case "g":
			return 1, 10, nil
		case "n":
			return 1, 50, nil
		}
		return 0, 0, fmt.Errorf("-stylised.tier %q is not g or n", tier)
	}
	if propose {
		return 0, 0, fmt.Errorf("-stylised.seeds cannot be combined with -stylised.propose")
	}
	a, b, ok := strings.Cut(spec, "-")
	f, errA := strconv.ParseUint(a, 10, 16)
	l, errB := strconv.ParseUint(b, 10, 16)
	if !ok || errA != nil || errB != nil || f < 1 || l < f {
		return 0, 0, fmt.Errorf("-stylised.seeds %q is not a range a-b with 1 <= a <= b", spec)
	}
	if l > 50 && (!report || f < 51 || l > 60) {
		return 0, 0, fmt.Errorf("-stylised.seeds %q reads the hold-out 51 to 60, which only -stylised.report may", spec)
	}
	return f, l, nil
}

// packSeeds returns the seeds a fact test runs on a pack that has at most
// p.maxSeeds seeds (REF 7.5): the range of the flags, cut to max unless the flag chose it,
// and under -race the first p.raceSeeds (plan 1 D227; the count of SR2 is the number
// run, no record is rewritten).
func packSeeds(t testing.TB, p stylisedPack) []uint64 {
	t.Helper()
	first, last, err := seedRange(*stylisedSeeds, *stylisedTier, *stylisedPropose, *stylisedReport)
	if err != nil {
		t.Fatalf("flags: %v", err)
	}
	if *stylisedSeeds == "" {
		last = min(last, uint64(p.maxSeeds))
	}
	if raceEnabled {
		last = min(last, first+uint64(p.raceSeeds)-1)
	}
	var out []uint64
	for s := first; s <= last; s++ {
		out = append(out, s)
	}
	return out
}

// failMsg is the message of a failed fact (testing.md 3.3, FR-STY-086): fact,
// pack, seed, measure, value and band.
func failMsg(fact, pack string, seed uint64, measure string, got any, want string) string {
	return fmt.Sprintf("%s pack %s seed %d: %s = %v, want %s", fact, pack, seed, measure, got, want)
}

// ---- the packs ---------------------------------------------------------------

// stylisedPack is a pack of testdata/chains with the seeds REF 7.5 gives it.
type stylisedPack struct {
	name     string
	files    []string
	maxSeeds int
	// raceSeeds is the seeds run under -race: the heavier the pack, the fewer
	// (plan 1 D227; the count of SR2 is the number run).
	raceSeeds int
}

var (
	packOne    = stylisedPack{"one", []string{"base.toml", "one.toml", "warm-one.toml"}, 10, 2}
	packSecond = stylisedPack{"second", []string{"second/second.toml", "second/warm-second.toml"}, 4, 1}
)

// ---- the bands ---------------------------------------------------------------

// readBands loads bands/<name>.toml. A key with a value asserts; one without
// reports with its prior beside it (FR-REF-022).
func readBands(t testing.TB, name string) map[string]verdict.BandKey {
	t.Helper()
	b, err := os.ReadFile("../testdata/chains/bands/" + name + ".toml")
	if err != nil {
		t.Fatalf("read bands %s: %v", name, err)
	}
	keys, err := verdict.LoadBands(string(b))
	if err != nil {
		t.Fatalf("bands/%s.toml: %v", name, err)
	}
	m := make(map[string]verdict.BandKey, len(keys))
	for _, k := range keys {
		m[k.Name] = k
	}
	return m
}

// readThresholds loads bands/regimes.toml.
func readThresholds(t testing.TB) verdict.Thresholds {
	t.Helper()
	b, err := os.ReadFile("../testdata/chains/bands/regimes.toml")
	if err != nil {
		t.Fatalf("read regimes: %v", err)
	}
	keys, err := verdict.LoadBands(string(b))
	if err != nil {
		t.Fatalf("bands/regimes.toml: %v", err)
	}
	th, err := verdict.ThresholdsOf(keys)
	if err != nil {
		t.Fatalf("bands/regimes.toml: %v", err)
	}
	return th
}

// workingThresholds are the priors of STY 6.2 as labelled working thresholds
// (6.5.1): they read a run, they never assert, and a record that rests on them
// says so.
func workingThresholds() verdict.Thresholds {
	th := verdict.NewThresholds()
	for k, v := range map[string]int64{
		"r1.rise_bp": 3000, "r1.windows": 2, "r1.m_factor_bp": 30000,
		"r2.vol_bp": 8000, "r2.share_bp": 5000, "r2.unsold_bp": 5000, "r2.top_bp": 7000, "r2.windows": 2,
		"r3.fall_bp": 2000, "r3.recover_bp": 1000, "r3.horizon_factor": 50000, "r3.horizon": 730, "r3.role_ticks": 365,
		"r4.vol_bp": 5000, "r4.windows": 2, "r5.pinned_bp": 5000, "r5.pinned_ticks": 60, "r5.freeze_ticks": 60,
		"r6.factor": 30000, "r6.growth_bp": 2500, "r6.windows": 2, "r6.swing_bp": 1000,
		"r7.disp_bp": 50, "r7.spike_factor": 20000, "r7.first_ticks": 60, "r8.offset_bp": 3000, "r8.unmet_bp": 3000,
		"r9.fall_bp": 3000, "r9.windows": 2, "stale_max_bp": 1000, "active_min_bp": 5000, "min_windows": 3,
	} {
		th.Set(k, v)
	}
	return th
}

// side is the side of a band: a fact is inside when its value is at most an
// upper bound or at least a lower one.
type side uint8

const (
	atMost side = iota
	atLeast
)

// stat is one number of one fact on one seed. Why is non-empty when the number
// does not exist (a blind-spot field outside its bound, an undefined ratio):
// the seed is then undetermined and counts as outside (STY 4.3).
type stat struct {
	Value int64
	Why   string
}

// factDef describes one banded or reported statistic of a fact.
type factDef struct {
	ID, Statistic string
	Class         verdict.FactClass
	Side          side
	Prior         int64
	HasPrior      bool
	Region, Role  string
}

// judged is what a fact came to over the seeds.
type judged struct {
	Row      verdict.FactRow
	Failures []string
	Notes    []string
	// Values are the numbers of the seeds that have one, in seed order, and Seeds
	// the seeds planned: the input of verdict.SetBand in the proposal of a setting
	// run (D598). Side is the side of the band.
	Values []int64
	Seeds  int
	Side   side
}

// sortedMedian returns the min, median and max of the values. The median of an
// even count is the half-to-even mean of the middle two (STY 8.3.4).
func sortedMedian(v []int64) (lo, med, hi int64) {
	r := verdict.SR3(len(v), v, 0, 0) // one median rule for the test and the record
	return r.Min, r.Median, r.Max
}

// outside reports whether value v is outside the bound b on the side.
func outside(sd side, v, b int64) bool {
	if sd == atLeast {
		return v < b
	}
	return v > b
}

// judge applies the band-or-report step (FR-REF-022): a key with a value is
// asserted by SR2 over the seeds; an unset key reports the values beside the
// prior. A seed whose number does not exist is outside. pack and the seed
// slice name the failure (FR-STY-086).
func judge(def factDef, pack string, seeds []uint64, vals []stat, band verdict.BandKey) judged {
	row := verdict.FactRow{ID: def.ID, Class: def.Class, Statistic: def.Statistic, Region: def.Region, Role: def.Role,
		N: len(seeds), BandSet: band.Set, Band: band.Value, HasPrior: def.HasPrior, Prior: def.Prior}
	var good []int64
	var out []int64
	var res judged
	why := map[string][]uint64{}
	res.Seeds, res.Side = len(seeds), def.Side
	for i, v := range vals {
		if v.Why == "" {
			res.Values = append(res.Values, v.Value)
		}
		bound, has := band.Value, band.Set
		if !has {
			bound, has = def.Prior, def.HasPrior
		}
		switch {
		case v.Why != "":
			out = append(out, int64(seeds[i]))
			why[v.Why] = append(why[v.Why], seeds[i])
		case has && outside(def.Side, v.Value, bound):
			out = append(out, int64(seeds[i]))
			good = append(good, v.Value)
		default:
			good = append(good, v.Value)
		}
	}
	for _, reason := range slices.Sorted(maps.Keys(why)) {
		res.Notes = append(res.Notes, fmt.Sprintf("%s pack %s seeds %v: %s undetermined: %s", def.ID, pack, why[reason], def.Statistic, reason))
	}
	row.Outside = out
	if len(good) > 0 {
		row.HasValues = true
		row.Min, row.Median, row.Max = sortedMedian(good)
	}
	if band.Set && def.Class != verdict.ClassR {
		if r := verdict.SR2(len(seeds), out); !r.Pass {
			res.Failures = append(res.Failures, failMsg(def.ID, pack, firstSeed(out, seeds), def.Statistic,
				fmt.Sprintf("%d..%d, %s", row.Min, row.Max, r), fmt.Sprintf("at least %d of %d seeds inside band %d", r.Need, r.N, band.Value)))
		}
	}
	res.Row = row
	return res
}

func firstSeed(out []int64, seeds []uint64) uint64 {
	if len(out) > 0 {
		return uint64(out[0])
	}
	return seeds[0]
}

// judgeExact applies SR1 to an exact or ordinal fact: errs[i] non-empty is a failure
// of seed i, which is asserted whether or not a band exists (FR-REF-023).
func judgeExact(def factDef, pack string, seeds []uint64, errs []string) judged {
	row := verdict.FactRow{ID: def.ID, Class: def.Class, Statistic: def.Statistic, Region: def.Region, Role: def.Role, N: len(seeds)}
	var res judged
	for i, e := range errs {
		if e == "" {
			continue
		}
		row.Outside = append(row.Outside, int64(seeds[i]))
		res.Failures = append(res.Failures, failMsg(def.ID, pack, seeds[i], def.Statistic, e, "none"))
	}
	res.Row = row
	return res
}

// ---- the record --------------------------------------------------------------

// book collects the rows and regimes of a pack under named scenarios, in the
// order the tests produce them; RenderVerdict sorts them.
type book struct {
	mu        sync.Mutex
	pack      string
	seeds     string
	scenarios map[string]*verdict.ScenarioRecord
}

func newBook(pack, seeds string) *book {
	return &book{pack: pack, seeds: seeds, scenarios: map[string]*verdict.ScenarioRecord{}}
}

func (b *book) scenario(name string) *verdict.ScenarioRecord {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.scenarios[name] == nil {
		b.scenarios[name] = &verdict.ScenarioRecord{Scenario: name}
	}
	return b.scenarios[name]
}

func (b *book) add(scenario string, j judged) {
	sc := b.scenario(scenario)
	b.mu.Lock()
	sc.Facts = append(sc.Facts, j.Row)
	b.mu.Unlock()
}

// render returns the verdict record. The commit and configuration hash are the
// words "tests" and the hash of the pack: a record of a test run names no
// commit, since a test knows none (the setting run of STY 8.4 names its own).
func (b *book) render(configHash string) string {
	b.mu.Lock()
	defer b.mu.Unlock()
	r := verdict.Record{Pack: b.pack, Seeds: b.seeds, Commit: "tests", Config: configHash}
	for _, sc := range b.scenarios {
		r.Scenarios = append(r.Scenarios, *sc)
	}
	return verdict.RenderVerdict(r)
}

// ---- the observer and the run ------------------------------------------------

// marketLog is what the facts keep of one market, indexed by tick.
type marketLog struct {
	Price, Anchor, Age, Volume []int64
	EstateUnits                []int64 // units an estate sold in the tick
}

func putAt(s *[]int64, i, v int64) {
	for int64(len(*s)) <= i {
		*s = append(*s, 0)
	}
	(*s)[i] = v
}

// runObs feeds the two tallies of a run and keeps the market series, the
// completions per recipe and tick, and the bids per commodity and tick (the
// tally ignores OrderPlaced: FR-REF-027).
type runObs struct {
	tick, year *indicator.Tally
	regions    economysim.AgentID // agent IDs below this are estates
	markets    map[[2]int]*marketLog
	complete   map[economysim.RecipeID][]int64
	bids       map[economysim.CommodityID][]int64
	badSeq     string
	// The events of the famine and trade facts: households unfed and relief paid
	// by tick, hunger removals, and the edge changes.
	unfed  []int64
	relief []int64
	// The population module's counts by tick (STY-W02, W06): births and deaths done
	// and births the pool could not fund.
	births, deaths, unfunded []int64
	removed                  []removal
	edges                    []economysim.EdgeStatusChanged
	watch                    *ladderWatch
	// quote is the limit price of the outside's line by (gate region, commodity)
	// and tick, 0 where the side was not present.
	quote map[[2]int][]int64
}

// removal is an AgentBankrupted of the run.
type removal struct {
	Tick   int64
	Agent  economysim.AgentID
	Cause  economysim.Cause
	Weight int64
}

func newRunObs() *runObs {
	return &runObs{tick: indicator.NewTally(), year: indicator.NewTally(), markets: map[[2]int]*marketLog{},
		complete: map[economysim.RecipeID][]int64{}, bids: map[economysim.CommodityID][]int64{}, quote: map[[2]int][]int64{}}
}

func (o *runObs) market(r economysim.RegionID, c economysim.CommodityID) *marketLog {
	k := [2]int{int(r), int(c)}
	if o.markets[k] == nil {
		o.markets[k] = &marketLog{}
	}
	return o.markets[k]
}

func addAt(s *[]int64, i, v int64) {
	for int64(len(*s)) <= i {
		*s = append(*s, 0)
	}
	(*s)[i] += v
}

func (o *runObs) Observe(e economysim.Event) {
	o.tick.Observe(e)
	o.year.Observe(e)
	switch e := e.(type) {
	case economysim.MarketCleared:
		m := o.market(e.Region, e.Commodity)
		if int64(len(m.Price)) != e.Tick && o.badSeq == "" {
			o.badSeq = fmt.Sprintf("MarketCleared of (%d,%d) at tick %d after %d ticks", e.Region, e.Commodity, e.Tick, len(m.Price))
		}
		putAt(&m.Price, e.Tick, int64(e.Price))
		putAt(&m.Anchor, e.Tick, int64(e.CostAnchor))
		putAt(&m.Age, e.Tick, e.PriceAge)
		putAt(&m.Volume, e.Tick, int64(e.Volume))
	case economysim.TradeExecuted:
		if e.Seller < o.regions {
			addAt(&o.market(e.Region, e.Commodity).EstateUnits, e.Tick, int64(e.Quantity))
		}
	case economysim.RecipeExecuted:
		c := o.complete[e.Recipe]
		addAt(&c, e.Tick, e.Executions)
		o.complete[e.Recipe] = c
	case economysim.UpkeepShortfall:
		addAt(&o.unfed, e.Tick, e.Unfed)
		if o.watch != nil {
			o.watch.unfedIn[e.Region] = true
		}
	case economysim.PopulationChanged:
		addAt(&o.births, e.Tick, e.BirthsDone)
		addAt(&o.deaths, e.Tick, e.DeathsDone)
		addAt(&o.unfunded, e.Tick, e.BirthsUnfunded)
	case economysim.ReliefPaid:
		addAt(&o.relief, e.Tick, int64(e.Paid))
	case economysim.AgentBankrupted:
		o.removed = append(o.removed, removal{e.Tick, e.Agent, e.Cause, e.Weight})
		if o.watch != nil {
			o.watch.removed = append(o.watch.removed, o.removed[len(o.removed)-1])
		}
	case economysim.EdgeStatusChanged:
		o.edges = append(o.edges, e)
	case economysim.OutsideTraded:
		k := [2]int{int(e.Region), int(e.Commodity)}
		q := o.quote[k]
		putAt(&q, e.Tick, int64(e.Price))
		o.quote[k] = q
	case economysim.OrderPlaced:
		if e.Side == economysim.Bid {
			b := o.bids[e.Commodity]
			addAt(&b, e.Tick, 1)
			o.bids[e.Commodity] = b
		}
	}
}

// pinnedTick is the Pinned row of one market at one tick, from a one-tick window.
type pinnedTick struct {
	Share int64 // basis points; -1 where no outcome was processed
	Stuck bool
}

// stylisedRun is one (pack, seed) run of the fact tests.
type stylisedRun struct {
	pack       string
	seed       uint64
	cfg        economysim.Config
	names      economysim.Names
	start      economysim.Snapshot
	years      []runWin                // whole year windows counted from tick 0
	obs        *runObs                 // the series
	pinned     map[[2]int][]pinnedTick // by market, one entry per tick
	households []int64                 // the households at the close of each tick
	money      []int64                 // the money of the non-boundary accounts at each close
	stepErr    error                   // the first error of a step (the audit)
	final      economysim.Snapshot
	hash       string
	graph      economysim.Graph // the region graph at the start
	agents     []agentRead      // every agent at the end, through Agent(a)
	ratio      []int64          // the household-weighted mean welfare ratio at each close, bp (0 where none)
	roleHH     map[int][]int64  // the households of each role at each close, by role
	tops       []verdict.Point  // the share of the money held by the top decile of the households at each year close (R2)
}

// collapsed reports whether the households at the end are under a tenth of the
// start: the pack has collapsed under its own rules and a fact that needs a
// living economy is a finding (N-312, N-314).
func (r *stylisedRun) collapsed() bool {
	return r.start.Households > 0 && r.final.Households*10 < r.start.Households
}

// capture drives a world, tick by tick, feeding the tallies and the series.
type capture struct {
	t     testing.TB
	cfg   economysim.Config
	seed  uint64
	w     *economysim.World
	run   *stylisedRun
	prev  economysim.Snapshot
	yOpen economysim.Snapshot
	opts  []economysim.Option
	// before runs between steps, before the step of w.Tick() (a scripted call);
	// after runs after the step, before the tick's windows close (a poll).
	before, after func(*economysim.World)
}

// newCapture builds the world of cfg for seed with the audit on and the
// recording observer attached.
func newCapture(t testing.TB, p stylisedPack, files []string, seed uint64, extra ...economysim.Option) *capture {
	t.Helper()
	return newCaptureFrom(t, p, loadPack(t, files...), seed, extra...)
}

// newCaptureFrom is newCapture for a configuration already loaded (and, for a
// provoked run, changed in Go: a halt switched off, a floor raised).
func newCaptureFrom(t testing.TB, p stylisedPack, cfg economysim.Config, seed uint64, extra ...economysim.Option) *capture {
	t.Helper()
	obs := newRunObs()
	c := &capture{t: t, cfg: cfg, seed: seed, opts: append([]economysim.Option{economysim.WithSeed(seed), economysim.WithAudit(), economysim.WithObserver(obs)}, extra...)}
	w, err := economysim.NewWorld(cfg, c.opts...)
	if err != nil {
		t.Fatalf("NewWorld(%s, seed %d) error = %v, want nil", p.name, seed, err)
	}
	c.w = w
	s0, err := w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot() error = %v, want nil", err)
	}
	obs.regions = economysim.AgentID(len(s0.Names.Regions))
	c.run = &stylisedRun{pack: p.name, seed: seed, cfg: w.Config(), names: s0.Names, start: s0, obs: obs, graph: w.Graph(),
		pinned: map[[2]int][]pinnedTick{}, hash: fmt.Sprintf("%x", w.ConfigHash())[:16]}
	c.prev, c.yOpen = s0, s0
	if err := obs.year.Open(s0); err != nil {
		t.Fatalf("Open() error = %v, want nil", err)
	}
	return c
}

// advance runs n ticks, one at a time, closing a one-tick window at each and a
// year window at each whole year from tick 0.
func (c *capture) advance(n int64) {
	c.t.Helper()
	obs := c.run.obs
	for i := int64(0); i < n && c.run.stepErr == nil; i++ {
		if err := obs.tick.Open(c.prev); err != nil {
			c.t.Fatalf("Open() at tick %d error = %v, want nil", c.prev.Tick, err)
		}
		if c.before != nil {
			c.before(c.w)
		}
		if err := c.w.Step(); err != nil {
			c.run.stepErr = fmt.Errorf("step at tick %d: %w", c.prev.Tick, err)
			return
		}
		if c.after != nil {
			c.after(c.w)
		}
		cur, err := c.w.Snapshot()
		if err != nil {
			c.t.Fatalf("Snapshot() error = %v, want nil", err)
		}
		win, err := obs.tick.Close(cur)
		if err != nil {
			c.t.Fatalf("Close() at tick %d error = %v, want nil", cur.Tick, err)
		}
		pr, err := indicator.Pinned(win, indicator.CellSel{})
		if err != nil {
			c.t.Fatalf("Pinned() at tick %d error = %v, want nil", cur.Tick, err)
		}
		c.notePinned(win, pr)
		c.run.households = append(c.run.households, cur.Households)
		c.run.money = append(c.run.money, int64(cur.Money))
		c.run.ratio = append(c.run.ratio, meanRatio(cur))
		c.noteRoles(cur)
		if cur.Tick%yearTicks == 0 {
			yw, err := obs.year.Close(cur)
			if err != nil {
				c.t.Fatalf("Close() of the year ending at tick %d error = %v, want nil", cur.Tick, err)
			}
			c.run.years = append(c.run.years, runWin{w: yw, open: c.yOpen, closing: cur})
			c.run.tops = append(c.run.tops, topShare(c.w, cur.Tick))
			if err := obs.year.Open(cur); err != nil {
				c.t.Fatalf("Open() error = %v, want nil", err)
			}
			c.yOpen = cur
		}
		c.prev = cur
	}
}

// notePinned appends the pinned share and the stuck flag of every market of a
// one-tick window; a market with no outcome this tick has share -1.
func (c *capture) notePinned(win indicator.Window, pr indicator.PinnedResult) {
	seen := map[[2]int]bool{}
	for _, row := range pr.Rows {
		k := [2]int{int(row.Region), int(row.Commodity)}
		seen[k] = true
		sh := int64(-1)
		if row.Share.OK {
			sh = int64(row.Share.Value)
		}
		c.run.pinned[k] = append(c.run.pinned[k], pinnedTick{Share: sh, Stuck: row.Stuck > 0})
	}
	for r := 0; r < win.Regions; r++ {
		for m := 0; m < win.Commodities; m++ {
			k := [2]int{r, m}
			if !seen[k] {
				c.run.pinned[k] = append(c.run.pinned[k], pinnedTick{Share: -1})
			}
		}
	}
}

// restore replaces the world by a restored copy of its checkpoint, with the
// same observer: the rest of the run must equal an uninterrupted one
// (FR-STY-088).
func (c *capture) restore() {
	c.t.Helper()
	cp, err := c.w.Checkpoint()
	if err != nil {
		c.t.Fatalf("Checkpoint() error = %v, want nil", err)
	}
	w, err := economysim.RestoreWorld(c.cfg, cp, economysim.WithAudit(), economysim.WithObserver(c.run.obs))
	if err != nil {
		c.t.Fatalf("RestoreWorld() error = %v, want nil", err)
	}
	c.w = w
}

// finish returns the run.
func (c *capture) finish() *stylisedRun {
	c.t.Helper()
	fin, err := c.w.Snapshot()
	if err != nil {
		c.t.Fatalf("Snapshot() error = %v, want nil", err)
	}
	c.run.final = fin
	c.run.agents = agentsOf(c.w)
	return c.run
}

// agentRead is what the facts keep of one agent at the end of a run: the row the
// presets of indicator take, its role, and the working ticks of its effective
// welfare window (STY 4.7.2).
type agentRead struct {
	indicator.AgentRow
	Role                 economysim.RoleID
	WorkingTicks, Window int64
}

// agentsOf reads every agent of the world through Agent(a) (plan 2-17), in region
// then ID order. A removed agent is not listed by Agents.
func agentsOf(w *economysim.World) []agentRead {
	var out []agentRead
	for g := range w.Graph().Len() {
		ids, err := w.Agents(economysim.RegionID(g))
		if err != nil {
			continue
		}
		for _, id := range ids {
			st, err := w.Agent(id)
			if err != nil {
				continue
			}
			out = append(out, agentRead{
				AgentRow: indicator.AgentRow{Region: st.Region, Weight: st.Weight, External: st.External, Money: st.Money, Hunger: st.Hunger,
					WelfareRatioBP: st.Ratio, EarnedRatioBP: st.EarnedRatio, HasWelfareRatio: st.HasRatio, HasEarnedRatio: st.HasEarned},
				Role: st.Role, WorkingTicks: st.WorkingTicks, Window: st.Window,
			})
		}
	}
	return out
}

// agentRows returns the rows of the agents, for RowsOf.
func (r *stylisedRun) agentRows() []indicator.AgentRow { return agentRowsOf(r.agents) }

// runKey names a memoised run.
type runKey struct {
	pack  string
	seed  uint64
	ticks int64
}

// memoRun is a run computed once: the first caller computes it, the others
// wait for it, so tests that run in parallel share the work.
type memoRun struct {
	once sync.Once
	run  *stylisedRun
}

var (
	runsMemoMu sync.Mutex
	runsMemo   = map[runKey]*memoRun{}
)

// cachedRun returns the run of key, built once per test binary: the first caller
// builds it, the others wait for it.
func cachedRun(k runKey, build func() *stylisedRun) *stylisedRun {
	runsMemoMu.Lock()
	m := runsMemo[k]
	if m == nil {
		m = &memoRun{}
		runsMemo[k] = m
	}
	runsMemoMu.Unlock()
	m.once.Do(func() { m.run = build() })
	if m.run == nil {
		// build ended its test (t.Fatalf) inside Do; Do never calls it again.
		panic(fmt.Sprintf("stylised: the run %+v failed to build in an earlier test", k))
	}
	return m.run
}

// closedRun returns the run of a pack and seed over ticks, once per test
// binary: the tests of the closed packs read the same runs.
func closedRun(t testing.TB, p stylisedPack, seed uint64, ticks int64) *stylisedRun {
	t.Helper()
	return cachedRun(runKey{p.name, seed, ticks}, func() *stylisedRun {
		c := newCapture(t, p, p.files, seed)
		c.advance(ticks)
		return c.finish()
	})
}

// shortRun runs a pack with the given layers for ticks with the recording
// observer attached and no per-tick snapshot: the cold and warm runs of the
// warm-start facts.
func shortRun(t testing.TB, files []string, seed uint64, ticks int64) (*runObs, economysim.Config) {
	t.Helper()
	cfg := loadPack(t, files...)
	obs := newRunObs()
	w, err := economysim.NewWorld(cfg, economysim.WithSeed(seed), economysim.WithAudit(), economysim.WithObserver(obs))
	if err != nil {
		t.Fatalf("NewWorld(%v, seed %d) error = %v, want nil", files, seed, err)
	}
	s0, err := w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot() error = %v, want nil", err)
	}
	obs.regions = economysim.AgentID(len(s0.Names.Regions))
	if _, err := w.Run(context.Background(), ticks); err != nil {
		t.Fatalf("Run(%d) of %v seed %d error = %v, want nil", ticks, files, seed, err)
	}
	return obs, w.Config()
}

// ---- the larger packs ----------------------------------------------------------

// The packs of groups B, C and D with the seeds of REF 7.5 and 8.2, and the seeds
// run under -race (plan 1 D227). famine is the cold layer set of REF 4.5.
var (
	packTwo       = stylisedPack{"two", []string{"base.toml", "two.toml", "warm-two.toml"}, 4, 1}
	packThree     = stylisedPack{"three", []string{"base.toml", "three.toml", "warm-three.toml"}, 4, 1}
	packOneDemand = stylisedPack{"one-demand", []string{"base.toml", "demand.toml", "one-demand.toml", "warm-one-demand.toml"}, 4, 1}
	packFive      = stylisedPack{"five", []string{"base.toml", "lines.toml", "five.toml", "warm-five.toml"}, 2, 0}
	packFamine    = stylisedPack{"famine", []string{"base.toml", "famine.toml", "overlay/estates.toml"}, 10, 1}
)

// requireTierN skips a test of tier N unless -stylised.tier=n selects it (D595):
// the trade and open-layer groups cost more than the gate pays.
func requireTierN(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("run-level test, skipped with -short")
	}
	if *stylisedTier != "n" {
		t.Skip("tier N (plan 2-19 D595, D596 group D): run with -stylised.tier=n")
	}
}

// ---- scripts ------------------------------------------------------------------

// scriptStep is one table of a script of testdata/chains/scripts, applied between
// steps before the step of Tick: a [[shock]] sets the yield shock of (Region,
// Recipe) to BP basis points (10 000 clears it); a [[buyer]] adds a visitor of
// the role visitor with Money that bids for Commodity each tick for Ticks ticks,
// ShareBP basis points of the market's mean volume over the 100 ticks before
// (REF 7.1.3). The grammar is re-implemented here on purpose: the reader of the
// root package is a _test.go symbol of another package (D594).
type scriptStep struct {
	Tick           int64
	Region, Recipe string
	BP             int64
	Buyer          bool
	Money          int64
	Commodity      string
	ShareBP, Ticks int64
}

// readScript reads scripts/<name>. Any other table, a missing key or a number
// that is not an integer fails the test.
func readScript(t testing.TB, name string) []scriptStep {
	t.Helper()
	b, err := os.ReadFile("../testdata/chains/scripts/" + name)
	if err != nil {
		t.Fatalf("read script %s: %v", name, err)
	}
	var out []scriptStep
	var cur *scriptStep
	num := func(line, v string) int64 {
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			t.Fatalf("script %s: %q is not an integer in %q", name, v, line)
		}
		return n
	}
	for _, raw := range strings.Split(string(b), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		switch line {
		case "[[shock]]", "[[buyer]]":
			out = append(out, scriptStep{Buyer: line == "[[buyer]]"})
			cur = &out[len(out)-1]
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok || cur == nil {
			t.Fatalf("script %s: unexpected line %q", name, line)
		}
		k, v = strings.TrimSpace(k), strings.Trim(strings.TrimSpace(v), `"`)
		switch k {
		case "tick":
			cur.Tick = num(line, v)
		case "region":
			cur.Region = v
		case "recipe":
			cur.Recipe = v
		case "yield_bp":
			cur.BP = num(line, v)
		case "money":
			cur.Money = num(line, v)
		case "buy":
			for _, kv := range strings.Split(strings.Trim(v, "{} "), ",") {
				ik, iv, _ := strings.Cut(kv, "=")
				iv = strings.Trim(strings.TrimSpace(iv), `"`)
				switch strings.TrimSpace(ik) {
				case "commodity":
					cur.Commodity = iv
				case "quantity_share_bp":
					cur.ShareBP = num(line, iv)
				case "ticks":
					cur.Ticks = num(line, iv)
				}
			}
		default:
			t.Fatalf("script %s: unknown key in %q", name, line)
		}
	}
	return out
}

// shockSpan returns the ticks at which the first shock of a script is set below
// 10 000 and the first later one clears it, with its region and recipe.
func shockSpan(steps []scriptStep) (from, to int64, region, recipe string) {
	for _, s := range steps {
		switch {
		case s.Buyer:
		case s.BP < 10_000 && recipe == "":
			from, region, recipe = s.Tick, s.Region, s.Recipe
		case s.BP == 10_000 && recipe == s.Recipe && to == 0:
			to = s.Tick
		}
	}
	return from, to, region, recipe
}

type buyerRun struct {
	id    economysim.AgentID
	until int64
}

// scriptDriver applies a script between steps through the public calls of the
// integration surface (SetYieldShocks, AddAgent, Direct): the harness empties a
// buyer's directive after its ticks, since a directive is for one step only
// unless sticky (FR-INT-026).
type scriptDriver struct {
	t      testing.TB
	steps  []scriptStep
	cfg    economysim.Config
	names  economysim.Names
	obs    *runObs
	active []buyerRun
}

func indexOf(names []string, id string) int { return slices.Index(names, id) }

// apply is the capture's before hook.
func (d *scriptDriver) apply(w *economysim.World) {
	d.t.Helper()
	tick := w.Tick()
	for _, b := range d.active {
		if b.until == tick {
			if err := w.Direct(b.id, economysim.Directive{}); err != nil {
				d.t.Fatalf("Direct(%d, empty) at tick %d = %v, want nil", b.id, tick, err)
			}
		}
	}
	for _, s := range d.steps {
		if s.Tick != tick {
			continue
		}
		if s.Buyer {
			d.active = append(d.active, buyerRun{id: d.addBuyer(w, s), until: s.Tick + s.Ticks})
			continue
		}
		g, r := indexOf(d.names.Regions, s.Region), indexOf(d.names.Recipes, s.Recipe)
		if g < 0 || r < 0 {
			d.t.Fatalf("script shock %+v: region or recipe unknown", s)
		}
		if err := w.SetYieldShocks([]economysim.YieldShock{{Region: economysim.RegionID(g), Recipe: economysim.RecipeID(r), BP: s.BP}}); err != nil {
			d.t.Fatalf("SetYieldShocks(%+v) error = %v, want nil", s, err)
		}
	}
}

// addBuyer adds the visitor of a [[buyer]] step with the sticky bid of REF 7.1.3: the
// quantity is ShareBP of the mean volume of the 100 ticks before (at least 1), the
// limit the ceiling of the commodity's reference band, and the visitor holds its
// other tradable commodities so that the fallback adds nothing.
func (d *scriptDriver) addBuyer(w *economysim.World, s scriptStep) economysim.AgentID {
	d.t.Helper()
	g, c, role := indexOf(d.names.Regions, s.Region), indexOf(d.names.Commodities, s.Commodity), indexOf(d.names.Roles, "visitor")
	if g < 0 || c < 0 || role < 0 {
		d.t.Fatalf("script buyer %+v: region, commodity or the role visitor unknown", s)
	}
	id, err := w.AddAgent(economysim.NewAgent{Region: economysim.RegionID(g), Role: economysim.RoleID(role), Money: economysim.Money(s.Money)})
	if err != nil {
		d.t.Fatalf("AddAgent(%+v) = %v, want nil", s, err)
	}
	var hold []economysim.CommodityID
	for _, other := range w.Config().Roles[role].Trades {
		if i := indexOf(d.names.Commodities, other); i >= 0 && i != c {
			hold = append(hold, economysim.CommodityID(i))
		}
	}
	_, ceiling := beliefInterval(d.cfg.World, int64(w.ReferencePrice(economysim.RegionID(g), economysim.CommodityID(c))), 0)
	q := int64(1)
	if v := d.obs.market(economysim.RegionID(g), economysim.CommodityID(c)).Volume; len(v) > 0 {
		if mean, ok := meanOver(v, max(0, s.Tick-100), min(s.Tick, int64(len(v)))); ok {
			q = max(1, (mean*s.ShareBP+5_000)/10_000)
		}
	}
	dir := economysim.Directive{Sticky: true, Hold: hold, Intents: []economysim.Intent{{
		Commodity: economysim.CommodityID(c), Side: economysim.Bid, Quantity: economysim.Quantity(q), Limit: economysim.Price(ceiling),
	}}}
	if err := w.Direct(id, dir); err != nil {
		d.t.Fatalf("Direct(%d, %+v) at tick %d = %v, want nil", id, dir, s.Tick, err)
	}
	return id
}

// meanOver is the mean of s[from:to] rounded half up (the volumes and prices are
// not negative); ok is false for an empty range.
func meanOver(s []int64, from, to int64) (int64, bool) {
	if from < 0 || to > int64(len(s)) || from >= to {
		return 0, false
	}
	var sum int64
	for _, v := range s[from:to] {
		sum += v
	}
	n := to - from
	return (2*sum + n) / (2 * n), true
}

// customRun runs a pack with the extra layers for ticks, once per test binary
// under tag, after setup has attached the hooks of the run to the capture (a
// scripted call, a poll). The standard strategies are registered, so a layer may
// name the merchant.
func customRun(t testing.TB, p stylisedPack, layers []string, tag string, seed uint64, ticks int64, setup func(*capture)) *stylisedRun {
	t.Helper()
	return mutatedRun(t, p, layers, tag, seed, ticks, nil, setup)
}

// mutatedRun is customRun with a change to the loaded configuration, for the
// provoked runs that D594 builds as Go calls (O9).
func mutatedRun(t testing.TB, p stylisedPack, layers []string, tag string, seed uint64, ticks int64, mutate func(*economysim.Config), setup func(*capture)) *stylisedRun {
	t.Helper()
	key := runKey{p.name + "+" + strings.Join(layers, ",") + "|" + tag, seed, ticks}
	return cachedRun(key, func() *stylisedRun {
		cfg := loadPack(t, append(slices.Clone(p.files), layers...)...)
		if mutate != nil {
			mutate(&cfg)
		}
		c := newCaptureFrom(t, p, cfg, seed, strategy.Options()...)
		if setup != nil {
			setup(c)
		}
		c.advance(ticks)
		return c.finish()
	})
}

// scriptedRun runs a pack with a script of shocks and buyers; watch attaches the
// ladder watch of STY-S06.
func scriptedRun(t testing.TB, p stylisedPack, layers []string, script string, seed uint64, ticks int64, watch bool) *stylisedRun {
	t.Helper()
	return customRun(t, p, layers, script+"|"+strconv.FormatBool(watch), seed, ticks, func(c *capture) {
		if script != "" {
			d := &scriptDriver{t: t, steps: readScript(t, script), cfg: c.run.cfg, names: c.run.names, obs: c.run.obs}
			c.before = d.apply
		}
		if watch {
			lw := newLadderWatch(c.run.cfg)
			c.run.obs.watch, c.after = lw, lw.poll
		}
	})
}

// meanRatio is the household-weighted mean welfare ratio of the roles of a
// snapshot that have one, in basis points, rounded half up; 0 where none has.
func meanRatio(s economysim.Snapshot) int64 {
	var sum, n int64
	for _, row := range s.Roles {
		if row.HasMeanRatio {
			sum, n = sum+row.MeanRatioBP*row.RatioHouseholds, n+row.RatioHouseholds
		}
	}
	if n == 0 {
		return 0
	}
	return (2*sum + n) / (2 * n)
}

// noteRoles appends to each role the households it has at the close of a tick,
// summed over the regions (the per-role series of R3).
func (c *capture) noteRoles(cur economysim.Snapshot) {
	if c.run.roleHH == nil {
		c.run.roleHH = map[int][]int64{}
	}
	now := map[int]int64{}
	for _, row := range cur.Roles {
		now[int(row.Role)] += row.Households
	}
	for role := range c.run.names.Roles {
		c.run.roleHH[role] = append(c.run.roleHH[role], now[role])
	}
}

// topShare is the share of the money of the households held by the top decile
// (Inequality.Top(1 000), IND 9 row 11) at a year close, read through Agent(a); a
// close with no household has the point 0.
func topShare(w *economysim.World, tick int64) verdict.Point {
	set, err := indicator.RowsOf(agentRowsOf(agentsOf(w)), indicator.ColMoney, indicator.RowOpt{ExcludeExternal: true})
	if err != nil {
		return verdict.Point{Tick: tick}
	}
	res, err := indicator.Inequality(set)
	if err != nil || !res.SharesOK {
		return verdict.Point{Tick: tick}
	}
	return verdict.Point{Tick: tick, Value: res.Top}
}

func agentRowsOf(a []agentRead) []indicator.AgentRow {
	rows := make([]indicator.AgentRow, len(a))
	for i, x := range a {
		rows[i] = x.AgentRow
	}
	return rows
}
