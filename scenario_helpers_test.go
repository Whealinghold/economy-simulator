package economysim_test

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/internal/checked"
)

// This file is the helper layer of the stability scenarios S01, S04, S07, S16
// and S19 (doc/plans/plan-phase-1-12.md, phase 7; spec-reference-chain.md 7.1
// and 7.2): the streaming observer that keeps the series the measures need, the
// measures of REF 7.1.4 on plain slices, the band reader, the script reader, the
// belief interval of FR-BEL-004 and the outcome report of D181. A measure is
// computed from events and the snapshot only (FR-REF-027); the one exception is
// the Stage 1 shock seam (export_scenarios_test.go).

const (
	bandsPath      = "testdata/chains/bands/one.toml"
	scriptsDir     = "testdata/chains/scripts"
	expectedDir    = "testdata/chains/expected"
	scenarioSeeds  = 10
	scenarioWorlds = 4
	// raceSeeds is the seed count of the gate run under -race (plan 2-1 D226): seed 1
	// is the golden seed; the rest run in gate F (D229). Two seeds took 58 min (D230).
	raceSeeds = 1
)

// gateFPrefixes are the name prefixes of the tests gate F runs without -race
// (plan 2-1 D229). A root-package test that skips under raceEnabled must start
// with one, or it runs in neither gate (TestRaceSkips_HaveAnFPrefix).
var gateFPrefixes = []string{
	"TestScenario", "TestDeterminism", "TestGolden", "TestInvariants", "TestCheckpoint",
	"TestFork", "TestConfigOnly", "TestFixture", "TestPacks", "TestCensus", "TestExpected",
	"TestStylised", "TestCost", "TestStable",
}

// TestMain (the only one in the root test binary: a second is a build error)
// refuses -update under -race before any test runs, so the refusal
// does not wait for the scenarios (plan 2-1 D228); it is the only place.
func TestMain(m *testing.M) {
	flag.Parse()
	if !updateAllowed(raceEnabled, *updateFlag) {
		fmt.Fprintln(os.Stderr, "-update is refused under -race: the seed set is reduced (plan 2-1 D228); run without -race")
		os.Exit(2)
	}
	os.Exit(m.Run())
}

// seedsFor is the seed count of a test with `full` seeds: all of them, or the
// first raceSeeds when race is set; never below 1.
func seedsFor(race bool, full int) int {
	if race {
		full = min(full, raceSeeds)
	}
	return max(full, 1)
}

// scenarioSeedCount is the seed count of a scenario test in this build.
func scenarioSeedCount() int { return seedsFor(raceEnabled, scenarioSeeds) }

// invariantSeeds is the seed count of a row of invariantTable (or of a
// determinism test) that has `full` seeds, in this build.
func invariantSeeds(full int) int { return seedsFor(raceEnabled, full) }

// updateAllowed reports whether -update may write a recorded file: not under
// -race, where the seed set is reduced and a written file would say "seeds 1-10"
// of a run that did not make it (plan 2-1 D228).
func updateAllowed(race, updating bool) bool { return !(race && updating) }

// scenSem bounds the worlds the scenario tests run at once (the machine is
// shared; the tests are still parallel).
var scenSem = make(chan struct{}, scenarioWorlds)

// seedList returns 1..n.
func seedList(n int) []uint64 {
	out := make([]uint64, max(n, 0)) // a negative count is an empty list, not a make panic
	for i := range out {
		out[i] = uint64(i + 1)
	}
	return out
}

// ---- bands -----------------------------------------------------------------

// bandFor returns the value of a band key such as "s01.band_bp" in
// testdata/chains/bands/one.toml and whether the key has one (REF 7.1.5: an
// unset key asserts nothing).
func bandFor(t testing.TB, key string) (int64, bool) {
	t.Helper()
	return bandIn(t, bandsPath, key)
}

// bandIn is bandFor on a path.
func bandIn(t testing.TB, path, key string) (int64, bool) {
	t.Helper()
	for _, tb := range plainTOML(t, path) {
		if tb.name != key {
			continue
		}
		v, ok := tb.kv["value"]
		if !ok {
			return 0, false
		}
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			t.Fatalf("%s: [%s] value %q is not an integer", path, key, v)
		}
		return n, true
	}
	t.Fatalf("%s has no table [%s]", path, key)
	return 0, false
}

// ---- scripts ---------------------------------------------------------------

// shockStep is one [[shock]] of a script: before the step of Tick the yield
// shock of (Region, Recipe) is set to BP basis points (10 000 clears it).
type shockStep struct {
	Tick           int64
	Region, Recipe string
	BP             int64
}

// loadScript reads a script of testdata/chains/scripts.
func loadScript(t testing.TB, name string) []shockStep {
	t.Helper()
	out, err := scriptFrom(plainTOML(t, filepath.Join(scriptsDir, name)))
	if err != nil {
		t.Fatalf("script %s: %v", name, err)
	}
	return out
}

// scriptFrom turns the tables of a script into its shocks, in file order. A
// table other than [[shock]], a missing key or a non-numeric number is an error.
func scriptFrom(tables []tomlTable) ([]shockStep, error) {
	var out []shockStep
	for _, tb := range tables {
		if tb.name != "shock" || !tb.array {
			return nil, fmt.Errorf("table [%s] is not [[shock]]", tb.name)
		}
		var s shockStep
		var err error
		if s.Tick, err = strconv.ParseInt(tb.kv["tick"], 10, 64); err != nil {
			return nil, fmt.Errorf("shock tick %q: %w", tb.kv["tick"], err)
		}
		if s.BP, err = strconv.ParseInt(tb.kv["yield_bp"], 10, 64); err != nil {
			return nil, fmt.Errorf("shock yield_bp %q: %w", tb.kv["yield_bp"], err)
		}
		s.Region, s.Recipe = strings.Trim(tb.kv["region"], `"`), strings.Trim(tb.kv["recipe"], `"`)
		if s.Region == "" || s.Recipe == "" {
			return nil, fmt.Errorf("shock at tick %d needs a region and a recipe", s.Tick)
		}
		out = append(out, s)
	}
	return out, nil
}

// indexOf returns the position of id in names, or -1.
func indexOf(names []string, id string) int { return slices.Index(names, id) }

// applyScript sets, before the step of tick w.Tick(), the shocks the script
// lists for that tick, through the Stage 1 seam (plan 1-12 O3).
func applyScript(t testing.TB, w *economysim.World, names economysim.Names, script []shockStep) {
	t.Helper()
	for _, s := range script {
		if s.Tick != w.Tick() {
			continue
		}
		g, r := indexOf(names.Regions, s.Region), indexOf(names.Recipes, s.Recipe)
		if g < 0 || r < 0 {
			t.Fatalf("script shock %+v: region or recipe unknown", s)
		}
		if err := economysim.SetYieldShockForTest(w, economysim.RegionID(g), economysim.RecipeID(r), s.BP); err != nil {
			t.Fatalf("SetYieldShockForTest(%+v) error = %v, want nil", s, err)
		}
	}
}

// ---- the observer and the runner ---------------------------------------------

// marketSeries is what the scenarios keep of one market, indexed by tick.
type marketSeries struct {
	Price, Volume, UnmatchedBid, UnmatchedAsk, Anchor []int64
	Outcomes, AtBound                                 []int64 // BeliefsUpdated; 0 on a tick without outcomes
	Age                                               []int64 // MarketCleared.PriceAge
	EstateUnits                                       []int64 // units an estate sold in the tick (TradeExecuted)
}

// put sets s[i] = v, growing s with zeros.
func put(s *[]int64, i, v int64) {
	for int64(len(*s)) <= i {
		*s = append(*s, 0)
	}
	(*s)[i] = v
}

// scenarioObs is a streaming Observer: it keeps the series the measures need
// and drops the rest, so a 1 400 tick run does not hold its millions of events.
type scenarioObs struct {
	markets map[marketKey]*marketSeries
	unfed   []int64                          // households unfed per tick (UpkeepShortfall)
	made    map[economysim.CommodityID]int64 // gross units from RecipeExecuted
	spoiled map[economysim.CommodityID]int64 // BoundaryFlow reason spoilage
	halted  int64                            // households the halt declined, summed (ProductionHalted)
	haltEv  int64                            // ProductionHalted events
	badSeq  string                           // first MarketCleared out of order
	// wantPlanned and wantFunded are the units of WantShortfall (Planned, Funded)
	// summed per tick; a tick with no event reads as 0 for both.
	wantPlanned, wantFunded []int64
	regions                 economysim.AgentID // agent IDs below this are estates (plan 1-8 regionOf)
}

func newScenarioObs() *scenarioObs {
	return &scenarioObs{
		markets: map[marketKey]*marketSeries{},
		made:    map[economysim.CommodityID]int64{},
		spoiled: map[economysim.CommodityID]int64{},
	}
}

func (o *scenarioObs) market(r economysim.RegionID, c economysim.CommodityID) *marketSeries {
	k := mkey(r, c)
	if o.markets[k] == nil {
		o.markets[k] = &marketSeries{}
	}
	return o.markets[k]
}

// Observe keeps what the measures read.
func (o *scenarioObs) Observe(e economysim.Event) {
	switch e := e.(type) {
	case economysim.MarketCleared:
		m := o.market(e.Region, e.Commodity)
		if int64(len(m.Price)) != e.Tick && o.badSeq == "" {
			o.badSeq = fmt.Sprintf("MarketCleared of (%d,%d) at tick %d after %d ticks", e.Region, e.Commodity, e.Tick, len(m.Price))
		}
		put(&m.Price, e.Tick, int64(e.Price))
		put(&m.Volume, e.Tick, int64(e.Volume))
		put(&m.UnmatchedBid, e.Tick, int64(e.UnmatchedBid))
		put(&m.UnmatchedAsk, e.Tick, int64(e.UnmatchedAsk))
		put(&m.Anchor, e.Tick, int64(e.CostAnchor))
		put(&m.Age, e.Tick, e.PriceAge)
	case economysim.TradeExecuted:
		if e.Seller < o.regions {
			m := o.market(e.Region, e.Commodity)
			for int64(len(m.EstateUnits)) <= e.Tick {
				m.EstateUnits = append(m.EstateUnits, 0)
			}
			m.EstateUnits[e.Tick] += int64(e.Quantity)
		}
	case economysim.BeliefsUpdated:
		m := o.market(e.Region, e.Commodity)
		put(&m.Outcomes, e.Tick, e.Outcomes)
		put(&m.AtBound, e.Tick, e.AtFloor+e.AtCeiling)
	case economysim.UpkeepShortfall:
		for int64(len(o.unfed)) <= e.Tick {
			o.unfed = append(o.unfed, 0)
		}
		o.unfed[e.Tick] += e.Unfed
	case economysim.RecipeExecuted:
		for _, a := range e.Outputs {
			o.made[a.Commodity] += int64(a.Quantity)
		}
	case economysim.BoundaryFlow:
		if e.Reason == economysim.ReasonSpoilage && e.Commodity != economysim.NoCommodity {
			o.spoiled[e.Commodity] += e.Total
		}
	case economysim.WantShortfall:
		for int64(len(o.wantPlanned)) <= e.Tick {
			o.wantPlanned, o.wantFunded = append(o.wantPlanned, 0), append(o.wantFunded, 0)
		}
		o.wantPlanned[e.Tick] += e.Planned
		o.wantFunded[e.Tick] += e.Funded
	case economysim.ProductionHalted:
		o.haltEv++
		o.halted += e.Halted
	}
}

// scenarioSpec describes one run of a scenario.
type scenarioSpec struct {
	Seed   uint64
	Ticks  int64
	Script []shockStep
	// Snap, when set, says after which step tick OnSnap gets a Snapshot.
	Snap   func(tick int64) bool
	OnSnap func(tick int64, s economysim.Snapshot)
	// Probe, when set, is called with the world after step tick.
	Probe func(tick int64, w *economysim.World)
}

// scenarioRun is what runScenario kept.
type scenarioRun struct {
	Obs   *scenarioObs
	Names economysim.Names
	Final economysim.Snapshot
}

// runScenario builds a world over cfg, applies the script between steps and
// runs Ticks steps with the streaming observer. It holds a slot of scenSem.
func runScenario(t testing.TB, cfg economysim.Config, spec scenarioSpec) scenarioRun {
	t.Helper()
	scenSem <- struct{}{}
	defer func() { <-scenSem }()
	obs := newScenarioObs()
	w, err := economysim.NewWorld(cfg, economysim.WithSeed(spec.Seed), economysim.WithObserver(obs))
	if err != nil {
		t.Fatalf("NewWorld(seed %d) error = %v, want nil", spec.Seed, err)
	}
	s0, err := w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot error = %v, want nil", err)
	}
	obs.regions = economysim.AgentID(len(s0.Names.Regions))
	for i := int64(0); i < spec.Ticks; i++ {
		applyScript(t, w, s0.Names, spec.Script)
		if err := w.Step(); err != nil {
			t.Fatalf("Step at tick %d (seed %d) error = %v, want nil", i, spec.Seed, err)
		}
		if spec.Probe != nil {
			spec.Probe(i, w)
		}
		if spec.Snap != nil && spec.Snap(i) {
			s, err := w.Snapshot()
			if err != nil {
				t.Fatalf("Snapshot after tick %d error = %v, want nil", i, err)
			}
			spec.OnSnap(i, s)
		}
	}
	fin, err := w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot error = %v, want nil", err)
	}
	return scenarioRun{Obs: obs, Names: s0.Names, Final: fin}
}

// ---- the belief interval (FR-BEL-004) ----------------------------------------

// beliefInterval restates FR-BEL-004 from the configuration and the cost anchor
// of the tick, with no call to production code: the floor is the larger of the
// reference-band floor and the cost floor, capped at MaxPrice; the ceiling is
// the smaller of the reference-band ceiling and the cost ceiling, raised to the
// floor. A ratio that overflows int64 reads as MaxPrice; an anchor of 0 is an
// unknown cost.
func beliefInterval(w economysim.WorldParams, ref, anchor int64) (floor, ceiling int64) {
	bp := func(v, k int64) int64 {
		q, ok := checked.MulDiv(v, k, 10_000)
		if !ok {
			return economysim.MaxPrice
		}
		return q
	}
	ref = max(ref, 1)
	refLow := max(1, bp(ref, w.RefBandLow))
	refHigh := min(economysim.MaxPrice, bp(ref, w.RefBandHigh))
	costLow, costHigh := int64(0), int64(economysim.MaxPrice)
	if anchor > 0 {
		costLow, costHigh = bp(anchor, w.CostFloor), bp(anchor, w.CostCeiling)
	}
	floor = min(economysim.MaxPrice, max(refLow, costLow))
	return floor, max(floor, min(refHigh, costHigh))
}

// refPrice returns the reference price of a commodity by its string ID.
func refPrice(cfg economysim.Config, id string) int64 {
	for _, c := range cfg.Commodities {
		if c.ID == id {
			return int64(c.ReferencePrice)
		}
	}
	return 0
}

// marketInterval is the interval a published price may lie in. FR-BEL-004
// bounds an agent's belief, and the cost band applies only to an agent that
// makes the commodity (strategy_view.go: UnitCost is 0 for any other agent), so
// a trade price, which lies between a seller's ask and a buyer's bid, is only
// known to lie in the reference band of an agent without a cost, widened up to
// the cost floor of a maker whose floor is above that band (the conflict of
// FR-BEL-004). This is the loosest interval that every agent's bounds fit in:
// the lowest floor is the reference-band floor of a seller who does not make the
// commodity, the highest ceiling is the larger of the reference-band ceiling and
// the floor of a maker.
func marketInterval(w economysim.WorldParams, ref, anchor int64) (lo, hi int64) {
	noCost, noCostHi := beliefInterval(w, ref, 0)
	mkFloor, mkHi := beliefInterval(w, ref, anchor)
	return noCost, max(noCostHi, mkFloor, mkHi)
}

// boundsReport is the result of boundsViolations.
type boundsReport struct {
	// Violations lists the (market, tick) pairs outside marketInterval.
	Violations []string
	// MakerOutside counts the pairs outside the stricter interval of a maker's
	// own belief bounds; reported, not asserted (see marketInterval).
	MakerOutside int64
	// EstateTicks counts the pairs exempted because an estate sold in the market
	// at the tick the published price was set: an estate's ask is priced by the markdown of EST 2.7.3, not by a
	// belief, so the published price it sets is not belief-bounded.
	EstateTicks int64
}

// boundsViolations checks, for every market and tick, that the published price
// lies inside marketInterval (FR-STB-008), except where an estate sold at the tick the price was set.
// At most limit violations are listed, each naming market, tick, price and
// interval.
func boundsViolations(cfg economysim.Config, names economysim.Names, o *scenarioObs, limit int) boundsReport {
	var rep boundsReport
	keys := make([]marketKey, 0, len(o.markets))
	for k := range o.markets {
		keys = append(keys, k)
	}
	slices.SortFunc(keys, func(a, b marketKey) int {
		if a[0] != b[0] {
			return int(a[0] - b[0])
		}
		return int(a[1] - b[1])
	})
	for _, k := range keys {
		m := o.markets[k]
		ref := refPrice(cfg, names.Commodities[k[1]])
		for tick, p := range m.Price {
			if mlo, mhi := beliefInterval(cfg.World, ref, m.Anchor[tick]); p < mlo || p > mhi {
				rep.MakerOutside++
			}
			lo, hi := marketInterval(cfg.World, ref, m.Anchor[tick])
			if p >= lo && p <= hi {
				continue
			}
			// The published price is carried until the next trade: it was set at
			// tick - PriceAge.
			if set := int64(tick) - m.Age[tick]; set >= 0 && set < int64(len(m.EstateUnits)) && m.EstateUnits[set] > 0 {
				rep.EstateTicks++
				continue
			}
			if len(rep.Violations) < limit {
				rep.Violations = append(rep.Violations, fmt.Sprintf("%s/%s tick %d price %d outside [%d, %d] (anchor %d)",
					names.Regions[k[0]], names.Commodities[k[1]], tick, p, lo, hi, m.Anchor[tick]))
			}
		}
	}
	return rep
}

// ---- the measures of REF 7.1.4 -----------------------------------------------

// windowMean returns the half-even mean of s[from:to]; ok is false for an empty
// or out-of-range window.
func windowMean(s []int64, from, to int64) (int64, bool) {
	if from < 0 || to > int64(len(s)) || to <= from {
		return 0, false
	}
	var sum int64
	for _, v := range s[from:to] {
		sum += v
	}
	return checked.MulDivHalfEven(sum, 1, to-from)
}

// recoveryTicks is the recovery time of REF S01: the first tick t >= from at
// which the win-tick mean of s ending at t lies within bandBP of p0 and does so
// at every later tick to the end, as ticks after from. ok is false if the last
// tick is outside the band or the series is shorter than the window.
func recoveryTicks(s []int64, from, p0, bandBP, win int64) (int64, bool) {
	n := int64(len(s))
	in := func(t int64) bool {
		m, ok := windowMean(s, t-win+1, t+1)
		if !ok {
			return false
		}
		d := m - p0
		if d < 0 {
			d = -d
		}
		return d*10_000 <= bandBP*p0
	}
	t := n - 1
	for t >= from && in(t) {
		t--
	}
	t++
	if t >= n || from < 0 {
		return 0, false
	}
	return t - from, true
}

// offsetBP is the signed gap of mean from seed in basis points, half-even.
func offsetBP(mean, seed int64) (int64, bool) {
	return checked.MulDivHalfEven(mean-seed, 10_000, seed)
}

// shareBP is 10 000 * num / den, half-even; ok is false for den 0.
func shareBP(num, den int64) (int64, bool) {
	return checked.MulDivHalfEven(num, 10_000, den)
}

// divHE is num / den rounded half to even; ok is false for den 0.
func divHE(num, den int64) (int64, bool) { return checked.MulDivHalfEven(num, 1, den) }

// longestRun is the longest run of true.
func longestRun(flags []bool) int64 {
	var best, cur int64
	for _, f := range flags {
		if f {
			cur++
			best = max(best, cur)
		} else {
			cur = 0
		}
	}
	return best
}

// pinnedFlags marks the ticks at which more than half of a market's belief
// outcomes ended at a bound (the condition of the "price beliefs at bound"
// Warn). A tick without outcomes is not pinned and breaks a run.
func pinnedFlags(m *marketSeries, ticks int64) []bool {
	out := make([]bool, ticks)
	for i := int64(0); i < ticks && i < int64(len(m.Outcomes)); i++ {
		out[i] = m.Outcomes[i] > 0 && 2*m.AtBound[i] > m.Outcomes[i]
	}
	return out
}

// priceAtBoundFlags marks the ticks at which the published price equals the
// floor or the ceiling of its belief interval.
func priceAtBoundFlags(cfg economysim.Config, ref int64, m *marketSeries) []bool {
	out := make([]bool, len(m.Price))
	for i, p := range m.Price {
		lo, hi := marketInterval(cfg.World, ref, m.Anchor[i])
		out[i] = p <= lo || p >= hi
	}
	return out
}

// arcElasticityBP is the arc elasticity of REF S04 (STY-W05) in basis points,
// signed (negative for a shortage, which lowers the quantity and raises the
// price): rdiv(10 000 * (q1 - q0) * (p1 + p0), (p1 - p0) * (q1 + q0)) with
// half-even rounding and integer arithmetic only. A zero denominator is absent.
func arcElasticityBP(q0, p0, q1, p1 int64) (int64, bool) {
	return checked.MulDivHalfEven(10_000*(q1-q0), p1+p0, (p1-p0)*(q1+q0))
}

func absInt(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}

// medianOf returns the median of vs (the half-even mean of the middle two).
func medianOf(vs []int64) int64 {
	s := slices.Clone(vs)
	slices.Sort(s)
	n := len(s)
	if n%2 == 1 {
		return s[n/2]
	}
	m, _ := checked.MulDivHalfEven(s[n/2-1]+s[n/2], 1, 2)
	return m
}

// ---- the report of D181 -------------------------------------------------------

// scenReport collects the measures of one scenario block per seed and the
// verdicts of its checks; the block is written to expected/<pack>.scenarios.txt
// with -update and never compared (D181, FR-REF-024).
type scenReport struct {
	mu     sync.Mutex
	pack   string
	title  string
	vals   map[string]map[uint64]string
	nums   map[string][]int64
	checks map[string][]string // check -> "seed:PASS" / "seed:FAIL detail"
}

// newScenReport is the report of a block on pack `one`.
func newScenReport(title string) *scenReport { return newPackScenReport("one", title) }

// newPackScenReport is the report of a block on the named pack; flush writes
// it to that pack's file only.
func newPackScenReport(pack, title string) *scenReport {
	return &scenReport{pack: pack, title: title, vals: map[string]map[uint64]string{}, nums: map[string][]int64{}, checks: map[string][]string{}}
}

// num records a numeric measure of a seed.
func (r *scenReport) num(seed uint64, name string, v int64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.setLocked(seed, name, strconv.FormatInt(v, 10))
	r.nums[name] = append(r.nums[name], v)
}

// absent records that a measure of a seed has no value.
func (r *scenReport) absent(seed uint64, name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.setLocked(seed, name, "absent")
}

func (r *scenReport) setLocked(seed uint64, name, v string) {
	if r.vals[name] == nil {
		r.vals[name] = map[uint64]string{}
	}
	r.vals[name][seed] = v
}

// check records the verdict of an exact, O or band check for a seed.
func (r *scenReport) check(seed uint64, name string, ok bool, detail string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	v := fmt.Sprintf("%02d:PASS", seed)
	if !ok {
		v = fmt.Sprintf("%02d:FAIL %s", seed, detail)
	}
	r.checks[name] = append(r.checks[name], v)
}

// lines renders the block with the measures and checks in sorted order.
func (r *scenReport) lines() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := []string{"## " + r.title}
	names := make([]string, 0, len(r.vals))
	for n := range r.vals {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		var seeds []uint64
		for s := range r.vals[n] {
			seeds = append(seeds, s)
		}
		slices.Sort(seeds)
		parts := make([]string, 0, len(seeds)+1)
		for _, s := range seeds {
			parts = append(parts, fmt.Sprintf("%d=%s", s, r.vals[n][s]))
		}
		line := "measure " + n + ": " + strings.Join(parts, " ")
		if len(r.nums[n]) > 0 {
			line += fmt.Sprintf(" | median=%d n=%d", medianOf(r.nums[n]), len(r.nums[n]))
		}
		out = append(out, line)
	}
	cn := make([]string, 0, len(r.checks))
	for n := range r.checks {
		cn = append(cn, n)
	}
	sort.Strings(cn)
	for _, n := range cn {
		vs := slices.Clone(r.checks[n])
		sort.Strings(vs)
		pass := 0
		for _, v := range vs {
			if strings.HasSuffix(v, ":PASS") {
				pass++
			}
		}
		out = append(out, fmt.Sprintf("check %s: %d/%d pass", n, pass, len(vs)))
		for _, v := range vs {
			if !strings.HasSuffix(v, ":PASS") {
				out = append(out, "  "+v)
			}
		}
	}
	return out
}

// scenFile serialises the read-modify-write of the scenarios file between the
// scenario tests of one process.
var scenFile sync.Mutex

// flush logs the block and, with -update, replaces its block in the scenarios
// file (D181).
func (r *scenReport) flush(t testing.TB) {
	t.Helper()
	lines := r.lines()
	t.Log("\n" + strings.Join(lines, "\n"))
	if !*updateFlag {
		return
	}
	if err := r.write(expectedDir); err != nil {
		t.Errorf("record %s: %v", r.title, err)
	}
}

// scenariosPath is the recorded file of a pack inside dir.
func scenariosPath(dir, pack string) string { return filepath.Join(dir, pack+".scenarios.txt") }

// write records the block in the file of its pack under dir and in no other. A
// pack name that is empty or is not a bare file-name stem is refused, so a typo
// cannot write ".scenarios.txt" or a file outside dir.
func (r *scenReport) write(dir string) error {
	if r.pack == "" || r.pack != filepath.Base(r.pack) || r.pack == "." || r.pack == ".." {
		return fmt.Errorf("scenario report %q: pack %q is not a bare name", r.title, r.pack)
	}
	return writeScenarioBlock(scenariosPath(dir, r.pack), r.title, r.lines())
}

// writeScenarioBlock replaces the block that starts with "## <title>" in the
// file, or adds it, keeping the blocks in title order.
func writeScenarioBlock(path, title string, lines []string) error {
	scenFile.Lock()
	defer scenFile.Unlock()
	blocks := map[string][]string{}
	if b, err := os.ReadFile(path); err == nil {
		var cur string
		for _, l := range strings.Split(strings.TrimRight(string(b), "\n"), "\n") {
			if strings.HasPrefix(l, "## ") {
				cur = l[3:]
			}
			if cur != "" && l != "" { // the blank line between blocks is written back, not kept
				blocks[cur] = append(blocks[cur], l)
			}
		}
	}
	blocks[title] = lines
	titles := make([]string, 0, len(blocks))
	for k := range blocks {
		titles = append(titles, k)
	}
	sort.Strings(titles)
	var sb strings.Builder
	sb.WriteString("# Recorded outcomes of the Stage 1 scenarios (plan 1-12 D181), written with -update.\n")
	sb.WriteString("# Read by the reviewer of STY 8.4; no test compares it.\n")
	for _, k := range titles {
		sb.WriteString("\n" + strings.Join(blocks[k], "\n") + "\n")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(sb.String()), 0o644)
}

// ---- unit tests of the helpers ------------------------------------------------

// TestScenarioHelpers_Measures verifies the measures on synthetic series: the
// window mean, the recovery time, the offset, the share, the longest run and the
// median, each on a hand-computed case.
//
// Requirement: FR-REF-020, FR-REF-027
func TestScenarioHelpers_Measures(t *testing.T) {
	t.Parallel()
	t.Run("windowMean", func(t *testing.T) {
		t.Parallel()
		tests := []struct {
			name     string
			s        []int64
			from, to int64
			want     int64
			ok       bool
		}{
			{"exact", []int64{10, 20, 30}, 0, 3, 20, true},
			{"half even down", []int64{1, 2}, 0, 2, 2, true},  // 1.5 -> 2 (even)
			{"half even stays", []int64{2, 3}, 0, 2, 2, true}, // 2.5 -> 2 (even)
			{"empty", []int64{1}, 1, 1, 0, false},
			{"past the end", []int64{1}, 0, 2, 0, false},
		}
		for _, tt := range tests {
			got, ok := windowMean(tt.s, tt.from, tt.to)
			if got != tt.want || ok != tt.ok {
				t.Errorf("%s: windowMean = %d, %t, want %d, %t", tt.name, got, ok, tt.want, tt.ok)
			}
		}
	})
	t.Run("recoveryTicks", func(t *testing.T) {
		t.Parallel()
		// p0 = 100, band 1000 bp (10%), window 2. The price is 100, then 150
		// from tick 4 to 7 (the shock), then 120 at tick 8, 100 from tick 9 on.
		// With from = 8 the two-tick mean ending at tick 8 is 135 (out), at 9 it
		// is 110 (in, on the edge) and stays in: recovery is 1 tick after from.
		s := []int64{100, 100, 100, 100, 150, 150, 150, 150, 120, 100, 100, 100, 100}
		if got, ok := recoveryTicks(s, 8, 100, 1000, 2); !ok || got != 1 {
			t.Errorf("recoveryTicks = %d, %t, want 1, true", got, ok)
		}
		// A series that dips out again at the end has not recovered "and stays".
		s2 := append(slices.Clone(s), 200, 200)
		if got, ok := recoveryTicks(s2, 8, 100, 1000, 2); ok {
			t.Errorf("recoveryTicks after a late excursion = %d, true, want not recovered", got)
		}
		// Back in the band already at from: 0 ticks.
		if got, ok := recoveryTicks(s, 10, 100, 1000, 2); !ok || got != 0 {
			t.Errorf("recoveryTicks already in band = %d, %t, want 0, true", got, ok)
		}
	})
	t.Run("offsetBP and shareBP", func(t *testing.T) {
		t.Parallel()
		// (110 - 100) * 10 000 / 100 = 1 000; (90 - 100) -> -1 000.
		if got, _ := offsetBP(110, 100); got != 1000 {
			t.Errorf("offsetBP(110, 100) = %d, want 1000", got)
		}
		if got, _ := offsetBP(90, 100); got != -1000 {
			t.Errorf("offsetBP(90, 100) = %d, want -1000", got)
		}
		if _, ok := offsetBP(1, 0); ok {
			t.Errorf("offsetBP with a zero seed is ok, want absent")
		}
		// 1/3 of 10 000 = 3 333.33 -> 3 333; 1/6 -> 1 666.67 -> 1 667.
		if got, _ := shareBP(1, 3); got != 3333 {
			t.Errorf("shareBP(1, 3) = %d, want 3333", got)
		}
		if got, _ := shareBP(1, 6); got != 1667 {
			t.Errorf("shareBP(1, 6) = %d, want 1667", got)
		}
	})
	t.Run("longestRun and pinnedFlags", func(t *testing.T) {
		t.Parallel()
		if got := longestRun([]bool{true, true, false, true, true, true, false}); got != 3 {
			t.Errorf("longestRun = %d, want 3", got)
		}
		if got := longestRun(nil); got != 0 {
			t.Errorf("longestRun(nil) = %d, want 0", got)
		}
		// Outcomes 4, 4, 0, 4; at bound 3, 2, 0, 4: pinned needs more than half,
		// so 3 of 4 is, 2 of 4 is not, and a tick without outcomes is not.
		m := &marketSeries{Outcomes: []int64{4, 4, 0, 4}, AtBound: []int64{3, 2, 0, 4}}
		got := pinnedFlags(m, 5)
		want := []bool{true, false, false, true, false}
		if !slices.Equal(got, want) {
			t.Errorf("pinnedFlags = %v, want %v", got, want)
		}
	})
	t.Run("medianOf", func(t *testing.T) {
		t.Parallel()
		if got := medianOf([]int64{5, 1, 3}); got != 3 {
			t.Errorf("medianOf odd = %d, want 3", got)
		}
		if got := medianOf([]int64{4, 1, 2, 3}); got != 2 { // (2+3)/2 = 2.5 -> 2 (even)
			t.Errorf("medianOf even = %d, want 2", got)
		}
	})
}

// TestScenarioHelpers_ArcElasticity verifies the arc elasticity on two
// hand-computed points, its sign and its zero denominator.
//
// Requirement: FR-STB-009, FR-REF-020
func TestScenarioHelpers_ArcElasticity(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		q0, p0, q1, p1 int64
		want           int64
		ok             bool
	}{
		// 10 000 * (80 - 100) * (120 + 100) / ((120 - 100) * (80 + 100))
		// = 10 000 * (-20) * 220 / (20 * 180) = -44 000 000 / 3 600 = -12 222.2 -> -12 222.
		{"shortage, price up, quantity down", 100, 100, 80, 120, -12222, true},
		// 10 000 * (-10) * 210 / (10 * 190) = -21 000 000 / 1 900 = -11 052.6 -> -11 053.
		{"second point", 100, 100, 90, 110, -11053, true},
		// Both rise together: a positive elasticity (a demand shift, not a shortage).
		{"price and quantity both up", 100, 100, 110, 110, 10000 * 10 * 210 / (10 * 210), true},
		{"price unchanged", 100, 100, 80, 100, 0, false},
		{"both quantities zero, price changes", 0, 100, 0, 120, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := arcElasticityBP(tt.q0, tt.p0, tt.q1, tt.p1)
			if ok != tt.ok || (ok && got != tt.want) {
				t.Errorf("arcElasticityBP(q0=%d p0=%d q1=%d p1=%d) = %d, %t, want %d, %t", tt.q0, tt.p0, tt.q1, tt.p1, got, ok, tt.want, tt.ok)
			}
		})
	}
	if got, _ := arcElasticityBP(100, 100, 80, 120); absInt(got) != 12222 {
		t.Errorf("absolute value = %d, want 12222", absInt(got))
	}
}

// TestScenarioHelpers_BeliefInterval verifies the restatement of FR-BEL-004 on
// hand-computed cases: the floor is the larger of the two floors, the ceiling the
// smaller of the two ceilings, raised to the floor, and both are capped.
//
// Requirement: FR-BEL-004, FR-REF-027
func TestScenarioHelpers_BeliefInterval(t *testing.T) {
	t.Parallel()
	w := economysim.WorldParams{RefBandLow: 5000, RefBandHigh: 20000, CostFloor: 7000, CostCeiling: 30000}
	tests := []struct {
		name        string
		ref, anchor int64
		lo, hi      int64
	}{
		// ref band [500, 2 000]; no cost known.
		{"no anchor", 1000, 0, 500, 2000},
		// cost band [700, 3 000]: floor 700, ceiling min(2 000, 3 000).
		{"anchor 1 000", 1000, 1000, 700, 2000},
		// cost band [3 500, 15 000]: floor 3 500 above the ref ceiling, raised to the floor.
		{"cost floor above the ref ceiling", 1000, 5000, 3500, 3500},
		// ref band [50, 200]; cost band [7, 30]: the ref floor wins, the cost ceiling caps.
		{"cost ceiling below the ref floor", 100, 1, 50, 50},
	}
	for _, tt := range tests {
		lo, hi := beliefInterval(w, tt.ref, tt.anchor)
		if lo != tt.lo || hi != tt.hi {
			t.Errorf("%s: beliefInterval(ref %d, anchor %d) = [%d, %d], want [%d, %d]", tt.name, tt.ref, tt.anchor, lo, hi, tt.lo, tt.hi)
		}
	}
	// marketInterval: the loosest interval of any agent. Without a cost the
	// band is [500, 2 000]; a maker with anchor 5 000 has [3 500, 3 500], which
	// widens the top; a maker with anchor 1 000 has [700, 2 000], which adds
	// nothing.
	for _, tt := range []struct {
		anchor, lo, hi int64
	}{{0, 500, 2000}, {1000, 500, 2000}, {5000, 500, 3500}} {
		lo, hi := marketInterval(w, 1000, tt.anchor)
		if lo != tt.lo || hi != tt.hi {
			t.Errorf("marketInterval(ref 1000, anchor %d) = [%d, %d], want [%d, %d]", tt.anchor, lo, hi, tt.lo, tt.hi)
		}
	}
	// An overflowing ratio reads as MaxPrice.
	lo, hi := beliefInterval(w, economysim.MaxPrice, 0)
	if hi != economysim.MaxPrice || lo > hi {
		t.Errorf("beliefInterval at MaxPrice = [%d, %d], want a ceiling of MaxPrice", lo, hi)
	}
}

// TestScenarioHelpers_BandFor verifies bandFor on a set and an unset key and on
// an unknown one, and that every key of the Stage 1 file is unset (REF 7.1.5).
//
// Requirement: FR-REF-022, FR-REF-024
func TestScenarioHelpers_BandFor(t *testing.T) {
	t.Parallel()
	p := filepath.Join(t.TempDir(), "b.toml")
	src := "[a.set]\nunit = \"bp\"\nvalue = 750\n[a.unset]\nunit = \"ticks\"\n"
	if err := os.WriteFile(p, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	if v, ok := bandIn(t, p, "a.set"); !ok || v != 750 {
		t.Errorf("bandIn(a.set) = %d, %t, want 750, true", v, ok)
	}
	if v, ok := bandIn(t, p, "a.unset"); ok || v != 0 {
		t.Errorf("bandIn(a.unset) = %d, %t, want 0, false", v, ok)
	}
	for _, key := range []string{"s01.band_bp", "s01.recover_ticks", "s01.offset_bp", "s01.unmet_bp", "s04.pinned_ticks",
		"s04.elasticity_max_bp", "s07.freeze_ticks", "s07.spoiled_bp", "s16.pinned_share_bp", "s19.ratio_band_bp",
		"s19.stock_bound_bp", "m05.pool_share_bp"} {
		if v, ok := bandFor(t, key); ok {
			t.Errorf("bandFor(%s) = %d, true, want unset in Stage 1 (REF 7.1.5)", key, v)
		}
	}
}

// TestScenarioHelpers_Script verifies the script reader on the shipped scripts
// and on a table that is not [[shock]].
//
// Requirement: FR-REF-021, FR-REF-039
func TestScenarioHelpers_Script(t *testing.T) {
	t.Parallel()
	want := map[string][]shockStep{
		"s01.toml": {{400, "north", "cut_fuel", 5000}, {460, "north", "cut_fuel", 10000}},
		"s04.toml": {{424, "north", "grow_staple", 5000}, {516, "north", "grow_staple", 10000}},
	}
	for name, w := range want {
		if got := loadScript(t, name); !slices.Equal(got, w) {
			t.Errorf("loadScript(%s) = %+v, want %+v", name, got, w)
		}
	}
	// Each bad input must parse (the parser accepts it) and then be refused by
	// scriptFrom; a parse error here would hide a script check that never ran.
	for _, bad := range []string{"[x]\na = 1\n", "[[shock]]\ntick = 1\nyield_bp = 5\n", "[[shock]]\ntick = \"a\"\nyield_bp = 5\n"} {
		tables, err := parsePlainTOML(bad)
		if err != nil {
			t.Errorf("parsePlainTOML(%q) error = %v, want nil: the input exists to test scriptFrom", bad, err)
			continue
		}
		if _, err := scriptFrom(tables); err == nil {
			t.Errorf("scriptFrom(%q) error = nil, want an error", bad)
		}
	}
}

// TestScenarioHelpers_Report verifies the report: the median, the verdict lines
// and the replacement of one block in the file by title.
//
// Requirement: FR-REF-024
func TestScenarioHelpers_Report(t *testing.T) {
	t.Parallel()
	r := newScenReport("S99 demo")
	r.num(1, "m", 10)
	r.num(2, "m", 30)
	r.absent(3, "m")
	r.check(1, "c", true, "")
	r.check(2, "c", false, "price too high")
	got := strings.Join(r.lines(), "\n")
	for _, want := range []string{"## S99 demo", "measure m: 1=10 2=30 3=absent | median=20 n=2", "check c: 1/2 pass", "02:FAIL price too high"} {
		if !strings.Contains(got, want) {
			t.Errorf("report lacks %q:\n%s", want, got)
		}
	}
	p := filepath.Join(t.TempDir(), "x", "s.txt")
	if err := writeScenarioBlock(p, "B", []string{"## B", "one"}); err != nil {
		t.Fatal(err)
	}
	if err := writeScenarioBlock(p, "A", []string{"## A", "two"}); err != nil {
		t.Fatal(err)
	}
	if err := writeScenarioBlock(p, "B", []string{"## B", "three"}); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(p)
	s := string(b)
	if strings.Index(s, "## A") > strings.Index(s, "## B") || !strings.Contains(s, "three") || strings.Contains(s, "one\n") {
		t.Errorf("file after replacing block B:\n%s", s)
	}
}

// seedPrice returns the warm-start price of a commodity in a region.
func seedPrice(cfg economysim.Config, region, commodity string) (int64, bool) {
	if cfg.WarmStart == nil {
		return 0, false
	}
	for _, p := range cfg.WarmStart.Prices {
		if p.Region == region && p.Commodity == commodity {
			return int64(p.Price), true
		}
	}
	return 0, false
}

// marketOf returns the series of a market by string IDs.
func marketOf(t testing.TB, run scenarioRun, region, commodity string) *marketSeries {
	t.Helper()
	g, c := indexOf(run.Names.Regions, region), indexOf(run.Names.Commodities, commodity)
	if g < 0 || c < 0 {
		t.Fatalf("market %s/%s is not in the names table", region, commodity)
	}
	m := run.Obs.markets[mkey(economysim.RegionID(g), economysim.CommodityID(c))]
	if m == nil {
		t.Fatalf("no MarketCleared for %s/%s", region, commodity)
	}
	return m
}

// checkExact records an exact or O check in the report and fails the test with
// the scenario, pack, seed and measure named (FR-REF-020).
func checkExact(t testing.TB, r *scenReport, scenario, pack string, seed uint64, check string, ok bool, format string, args ...any) {
	t.Helper()
	detail := fmt.Sprintf(format, args...)
	r.check(seed, check, ok, detail)
	if !ok {
		t.Errorf("%s pack %s seed %d: %s: %s", scenario, pack, seed, check, detail)
	}
}

// checkBand asserts a band check only when its key has a value (REF 7.1.5) and
// records the verdict; with no value it records nothing.
func checkBand(t testing.TB, r *scenReport, scenario, pack string, seed uint64, key string, measure int64, have bool, failIf func(bound int64) bool) {
	t.Helper()
	bound, set := bandFor(t, key)
	if !set || !have {
		return
	}
	bad := failIf(bound)
	r.check(seed, "band "+key, !bad, fmt.Sprintf("measure %d against %d", measure, bound))
	if bad {
		t.Errorf("%s pack %s seed %d: band %s: measure %d against %d", scenario, pack, seed, key, measure, bound)
	}
}

// TestScenarioHelpers_BoundsViolations verifies the exact bounds check on
// synthetic series: a price one above the interval ceiling and one below the
// floor are listed, prices on the edges and inside are not, a price outside
// where an estate sold at the tick it was set is exempt and counted, and the
// limit caps the list. Reference price 1 000 and a band of 5 000 to 20 000 bp
// give [500, 2 000] with no cost anchor.
//
// Requirement: FR-STB-008, FR-BEL-004
// Guard: a boundsViolations that flagged nothing would let the exact check of
// S01 and S04 pass vacuously.
func TestScenarioHelpers_BoundsViolations(t *testing.T) {
	t.Parallel()
	cfg := economysim.Config{
		World:       economysim.WorldParams{RefBandLow: 5000, RefBandHigh: 20000, CostFloor: 7000, CostCeiling: 30000},
		Commodities: []economysim.Commodity{{ID: "c", ReferencePrice: 1000}},
	}
	names := economysim.Names{Regions: []string{"r"}, Commodities: []string{"c"}}
	run := func(price, age, estate []int64, limit int) boundsReport {
		o := newScenarioObs()
		m := o.market(0, 0)
		m.Price, m.Age, m.EstateUnits = price, age, estate
		m.Anchor = make([]int64, len(price))
		return boundsViolations(cfg, names, o, limit)
	}
	zero := func(n int) []int64 { return make([]int64, n) }
	tests := []struct {
		name       string
		price, age []int64
		estate     []int64
		limit      int
		wantTicks  []string // substrings, one per violation, in order
		wantEstate int64
	}{
		{"inside and on the edges", []int64{500, 1000, 2000}, zero(3), nil, 5, nil, 0},
		{"one above the ceiling", []int64{1000, 2001}, zero(2), nil, 5, []string{"tick 1 price 2001 outside [500, 2000]"}, 0},
		{"one below the floor", []int64{499, 1000}, zero(2), nil, 5, []string{"tick 0 price 499 outside [500, 2000]"}, 0},
		{"both, in tick order", []int64{499, 2001}, zero(2), nil, 5, []string{"tick 0 price 499", "tick 1 price 2001"}, 0},
		{"limit caps the list", []int64{499, 2001}, zero(2), nil, 1, []string{"tick 0 price 499"}, 0},
		// The price of tick 1 was set at tick 1 - age 1 = 0, where an estate sold.
		{"estate sale exempts the price", []int64{1000, 2001}, []int64{0, 1}, []int64{3}, 5, nil, 1},
		{"estate sold at another tick does not", []int64{1000, 2001}, zero(2), []int64{3, 0}, 5, []string{"tick 1 price 2001"}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := run(tt.price, tt.age, tt.estate, tt.limit)
			if len(got.Violations) != len(tt.wantTicks) {
				t.Fatalf("boundsViolations(%v) violations = %q, want %d", tt.price, got.Violations, len(tt.wantTicks))
			}
			for i, w := range tt.wantTicks {
				if !strings.Contains(got.Violations[i], w) {
					t.Errorf("violation %d = %q, want it to contain %q", i, got.Violations[i], w)
				}
			}
			if got.EstateTicks != tt.wantEstate {
				t.Errorf("EstateTicks = %d, want %d", got.EstateTicks, tt.wantEstate)
			}
		})
	}
}

// TestScenarioHelpers_UnmetCounts verifies the S01 unmet measure. Arithmetic:
// ticks 10 to 12 start with 10, 8 and 4 households and have 2, 3 and 1 unfed:
// 6 of 22 households, 6 * 10 000 / 22 = 2 727.27, half-even 2 727 bp. A tick
// that starves 3 of 4 households and leaves 1 at the snapshot after the step
// stays at 7 500 bp over its own start (the old denominator, the snapshot after
// the step, gave 30 000 bp on seed 7 of pack one); more unfed than households
// at the start is reported as not within.
//
// Requirement: FR-REF-024, FR-REF-027
// Fix: S01 unmet was 30 000 bp on seed 7: the numerator counted households
// before the exits of the tick and the denominator after them.
func TestScenarioHelpers_UnmetCounts(t *testing.T) {
	t.Parallel()
	unfed := make([]int64, 14)
	unfed[10], unfed[11], unfed[12] = 2, 3, 1
	num, den, within := unmetCounts(unfed, 10, []int64{10, 8, 4})
	if num != 6 || den != 22 || !within {
		t.Fatalf("unmetCounts = %d, %d, %t, want 6, 22, true", num, den, within)
	}
	if bp, ok := shareBP(num, den); !ok || bp != 2727 {
		t.Errorf("unmet = %d, %t, want 2727, true", bp, ok)
	}
	// 3 of the 4 households at the start starved, 1 is left after the step.
	num, den, within = unmetCounts([]int64{3}, 0, []int64{4})
	if bp, _ := shareBP(num, den); bp != 7500 || !within {
		t.Errorf("collapse tick unmet = %d bp, within %t, want 7500, true", bp, within)
	}
	// More unfed than households at the start: a defect of the counts.
	if _, _, within = unmetCounts([]int64{3}, 0, []int64{1}); within {
		t.Errorf("unmetCounts with 3 unfed of 1 household within = true, want false")
	}
	// A window past the end of the series counts no unfed household.
	if num, den, _ = unmetCounts([]int64{1}, 5, []int64{4, 4}); num != 0 || den != 8 {
		t.Errorf("unmetCounts past the series = %d, %d, want 0, 8", num, den)
	}
}

// TestScenarioSeedCount_RaceAware verifies the seed count of a scenario is all
// of the seeds in a normal build and raceSeeds under -race, is at least 1, and
// that seedList of it starts at seed 1, the golden seed.
//
// Guard: if the count could reach 0 a scenario would assert nothing and pass; if
// seed 1 were dropped the reduced run would not cover the golden seed.
func TestScenarioSeedCount_RaceAware(t *testing.T) {
	t.Parallel()
	want := scenarioSeeds
	if raceEnabled {
		want = raceSeeds
	}
	got := scenarioSeedCount()
	if got != want {
		t.Errorf("scenarioSeedCount() = %d with raceEnabled=%t, want %d", got, raceEnabled, want)
	}
	if got < 1 {
		t.Errorf("scenarioSeedCount() = %d, want at least 1", got)
	}
	if l := seedList(got); len(l) != got || l[0] != 1 {
		t.Errorf("seedList(%d) = %v, want %d seeds starting at 1", got, l, got)
	}
}

// TestInvariantSeeds_RaceAware verifies a row's seed count is its full count in
// a normal build and at most raceSeeds under -race, never below 1, for both
// builds (seedsFor) and for this one (invariantSeeds).
//
// Guard: a row of invariantTable with 16 seeds is the slowest test under -race;
// the count is what keeps gate R inside its budget (plan 2-1 D226, D230).
func TestInvariantSeeds_RaceAware(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		race       bool
		full, want int
	}{
		{"16 seeds, normal", false, 16, 16},
		{"4 seeds, normal", false, 4, 4},
		{"16 seeds, race", true, 16, raceSeeds},
		{"4 seeds, race", true, 4, raceSeeds},
		{"1 seed, race", true, 1, 1},
		{"1 seed, normal", false, 1, 1},
		{"0 seeds is raised to 1, race", true, 0, 1},
		{"0 seeds is raised to 1, normal", false, 0, 1},
	}
	for _, tt := range tests {
		if got := seedsFor(tt.race, tt.full); got != tt.want {
			t.Errorf("%s: seedsFor(%t, %d) = %d, want %d", tt.name, tt.race, tt.full, got, tt.want)
		}
	}
	for _, full := range []int{1, 4, 16} {
		if got, want := invariantSeeds(full), seedsFor(raceEnabled, full); got != want {
			t.Errorf("invariantSeeds(%d) = %d with raceEnabled=%t, want %d", full, got, raceEnabled, want)
		}
	}
}

// TestUpdateAllowed_NeverUnderRace verifies -update is refused only when both
// -race and -update are set.
//
// Guard: a reduced run that wrote the recorded scenarios file would put a
// "seeds 1-10" title over two seeds (plan 2-1 D228).
func TestUpdateAllowed_NeverUnderRace(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		race, updating, want bool
	}{{true, true, false}, {true, false, true}, {false, true, true}, {false, false, true}} {
		if got := updateAllowed(tt.race, tt.updating); got != tt.want {
			t.Errorf("updateAllowed(race=%t, updating=%t) = %t, want %t", tt.race, tt.updating, got, tt.want)
		}
	}
}

// TestScenReport_PerPackPath verifies a report for pack X writes only
// X.scenarios.txt and leaves the file of another pack as it was.
//
// Guard: one file per pack lets plans 5, 7, 12 and 24 record their packs without
// rewriting the recorded outcomes of pack one.
func TestScenReport_PerPackPath(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	other := scenariosPath(dir, "one")
	if err := os.WriteFile(other, []byte("untouched\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	r := newPackScenReport("x", "S00 x seeds 1-2, 1 ticks")
	r.num(1, "m", 7)
	if err := r.write(dir); err != nil {
		t.Fatalf("write: %v", err)
	}
	if b, err := os.ReadFile(other); err != nil || string(b) != "untouched\n" {
		t.Errorf("file of pack one = %q, %v, want it untouched", b, err)
	}
	b, err := os.ReadFile(scenariosPath(dir, "x"))
	if err != nil || !strings.Contains(string(b), "## S00 x seeds 1-2, 1 ticks") {
		t.Errorf("file of pack x = %q, %v, want the block of the report", b, err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 2 {
		t.Errorf("directory holds %d files, %v, want exactly one.scenarios.txt and x.scenarios.txt", len(entries), err)
	}
	if got, want := scenariosPath("d", "one"), filepath.Join("d", "one.scenarios.txt"); got != want {
		t.Errorf("scenariosPath(d, one) = %q, want %q", got, want)
	}
	if got := newScenReport("t").pack; got != "one" {
		t.Errorf("newScenReport pack = %q, want one", got)
	}
}

// TestScenReport_WriteRefusesBadPack verifies a report whose pack name is empty
// or has a path element is refused and writes nothing, and that seedList of a
// negative count is empty.
//
// Guard: the pack name becomes a file name under expected/; "" would write
// ".scenarios.txt" and "../x" would write outside the directory.
func TestScenReport_WriteRefusesBadPack(t *testing.T) {
	t.Parallel()
	for _, pack := range []string{"", ".", "..", "../x", "a/b"} {
		dir := t.TempDir()
		r := newPackScenReport(pack, "S00 bad seeds 1-1, 1 ticks")
		if err := r.write(dir); err == nil {
			t.Errorf("write with pack %q = nil, want an error", pack)
		}
		if entries, err := os.ReadDir(dir); err != nil || len(entries) != 0 {
			t.Errorf("pack %q left %d files, %v, want none", pack, len(entries), err)
		}
	}
	if got := seedList(-1); len(got) != 0 {
		t.Errorf("seedList(-1) = %v, want empty", got)
	}
}

// hasFPrefix reports whether a test name starts with a prefix of gateFPrefixes.
func hasFPrefix(name string) bool {
	for _, p := range gateFPrefixes {
		if strings.HasPrefix(name, p) {
			return true
		}
	}
	return false
}

// tri is a three-valued truth: a condition is evaluated with raceEnabled true and
// every other term unknown.
type tri int

const (
	triFalse tri = iota
	triUnknown
	triTrue
)

// negate is the three-valued NOT.
func negate(v tri) tri {
	switch v {
	case triTrue:
		return triFalse
	case triFalse:
		return triTrue
	}
	return triUnknown
}

// evalUnderRace evaluates cond with raceEnabled true and any other term unknown.
func evalUnderRace(cond ast.Expr) tri {
	switch e := cond.(type) {
	case *ast.Ident:
		if e.Name == "raceEnabled" {
			return triTrue
		}
	case *ast.ParenExpr:
		return evalUnderRace(e.X)
	case *ast.UnaryExpr:
		if e.Op == token.NOT {
			return negate(evalUnderRace(e.X))
		}
	case *ast.BinaryExpr:
		switch e.Op {
		case token.EQL, token.NEQ:
			if id, ok := e.Y.(*ast.Ident); ok && (id.Name == "true" || id.Name == "false") {
				v := evalUnderRace(e.X)
				if (id.Name == "true") == (e.Op == token.EQL) {
					return v
				}
				return negate(v)
			}
		case token.LOR:
			return max(evalUnderRace(e.X), evalUnderRace(e.Y))
		case token.LAND:
			return min(evalUnderRace(e.X), evalUnderRace(e.Y))
		}
	}
	return triUnknown
}

// Known limit: a skip behind an alias of raceEnabled (race := raceEnabled), in a
// switch, or in a helper not named skip* is not seen; name such helpers skip*.
// A condition that skips under -race only through an unknown term
// (`!raceEnabled || testing.CoverMode() != "atomic"`) reads as unknown and is
// not seen either; write the race skip as its own `if raceEnabled`.

// isSkipCall reports whether c is t.Skip*, or a call of a helper named skip*
// (skipIfRace(t)), which is how a test could hide its skip from raceSkippers.
func isSkipCall(c *ast.CallExpr) bool {
	switch f := c.Fun.(type) {
	case *ast.SelectorExpr:
		return f.Sel.Name == "Skip" || f.Sel.Name == "Skipf" || f.Sel.Name == "SkipNow"
	case *ast.Ident:
		return strings.HasPrefix(strings.ToLower(f.Name), "skip")
	}
	return false
}

// skipsOnRacePath reports whether n holds a skip call that runs under -race
// whatever the other terms are: every if on the path to it has a condition that
// is certainly true there (or certainly false, for its else), and there is at
// least one such if (in is set once a certain if has been entered). A nested
// condition that is unknown, such as testing.Short(), breaks the certainty, so
// `if raceEnabled { if testing.Short() {...} }` reads as `raceEnabled &&
// testing.Short()`.
func skipsOnRacePath(n ast.Node, in bool) (found bool) {
	ast.Inspect(n, func(m ast.Node) bool {
		switch x := m.(type) {
		case *ast.IfStmt:
			switch v := evalUnderRace(x.Cond); {
			case v == triTrue:
				found = found || skipsOnRacePath(x.Body, true)
			case v == triFalse && x.Else != nil:
				found = found || skipsOnRacePath(x.Else, true)
			}
			return false
		case *ast.CallExpr:
			if in && isSkipCall(x) {
				found = true
			}
		}
		return true
	})
	return found
}

// raceSkippers returns the Test functions of the files that call a t.Skip* in
// an if whose condition holds under -race (skipsOnRacePath).
func raceSkippers(t *testing.T, files map[string]string) []string {
	t.Helper()
	var out []string
	fset := token.NewFileSet()
	for name, src := range files {
		f, err := parser.ParseFile(fset, name, src, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		for _, d := range f.Decls {
			fd, ok := d.(*ast.FuncDecl)
			if !ok || fd.Body == nil || !strings.HasPrefix(fd.Name.Name, "Test") {
				continue
			}
			skips := skipsOnRacePath(fd.Body, false)
			if skips {
				out = append(out, fd.Name.Name)
			}
		}
	}
	return out
}

// TestRaceSkips_HaveAnFPrefix verifies every root-package test that skips under
// raceEnabled has a name that starts with a prefix of gateFPrefixes, so gate F
// (which runs those prefixes without -race) runs it (plan 2-1 D229). The matcher
// and the detector are checked on small sources first.
//
// Guard: a test skipped under -race whose name gate F does not select runs in
// neither gate, and nothing would say so.
func TestRaceSkips_HaveAnFPrefix(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name string
		want bool
	}{{"TestScenarioS19_HaltOnAndOff", true}, {"TestInvariants_Packs", true}, {"TestStep_AllocationsStayAtTheBaseline", false}} {
		if got := hasFPrefix(tt.name); got != tt.want {
			t.Errorf("hasFPrefix(%q) = %t, want %t", tt.name, got, tt.want)
		}
	}
	probe := map[string]string{"probe_test.go": `package p
func TestA(t *testing.T) { if raceEnabled { t.Skip("x") } }
func TestB(t *testing.T) { if !raceEnabled || testing.CoverMode() != "atomic" { t.Skipf("x") } }
func TestC(t *testing.T) { if testing.Short() { t.Skip("x") } }
func TestD(t *testing.T) { if raceEnabled && testing.Short() { t.Skip("x") } }
func TestE(t *testing.T) { if raceEnabled || testing.Short() { t.Skip("x") } }
func TestF(t *testing.T) { if raceEnabled { t.Log("x") } }
func TestG(t *testing.T) { if raceEnabled { skipIfRace(t) } }
func TestJ(t *testing.T) { if raceEnabled == true { t.Skip("x") } }
func TestK(t *testing.T) { if !raceEnabled { t.Log("x") } else if testing.Short() { t.Skip("x") } }
func TestI(t *testing.T) { if raceEnabled { if testing.Short() { t.Skip("x") } } }
func TestH(t *testing.T) { if !raceEnabled { t.Log("x") } else { t.Skip("x") } }
`}
	if got := raceSkippers(t, probe); !(len(got) == 5 && slices.Contains(got, "TestJ") && slices.Contains(got, "TestA") && slices.Contains(got, "TestE") && slices.Contains(got, "TestG") && slices.Contains(got, "TestH")) {
		t.Errorf("raceSkippers(probe) = %v, want TestA, TestE, TestG, TestH and TestJ only (TestI, nested under testing.Short(), skips only under -short as TestD does)", got)
	}
	doc, err := os.ReadFile("doc/rules/testing.md")
	if err != nil {
		t.Fatalf("read testing.md: %v", err)
	}
	if want := "-run '" + strings.Join(gateFPrefixes, "|") + "'"; !strings.Contains(string(doc), want) {
		t.Errorf("doc/rules/testing.md 9.1 lacks the quick-run pattern %s built from gateFPrefixes (same prefixes, same order, one line)", want)
	}
	files, err := filepath.Glob("*_test.go")
	if err != nil || len(files) == 0 {
		t.Fatalf("glob *_test.go = %v, %v, want the root test files", files, err)
	}
	srcs := map[string]string{}
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		srcs[f] = string(b)
	}
	for _, name := range unprefixedRaceSkippers(t, srcs) {
		t.Errorf("%s skips under raceEnabled but starts with none of %v, so it runs in neither gate R nor gate F; rename it", name, gateFPrefixes)
	}
}

// unprefixedRaceSkippers returns the tests of the sources that skip under -race
// and whose name no gate F prefix selects: the violations of the rule.
func unprefixedRaceSkippers(t *testing.T, files map[string]string) []string {
	t.Helper()
	var out []string
	for _, name := range raceSkippers(t, files) {
		if !hasFPrefix(name) {
			out = append(out, name)
		}
	}
	return out
}

// TestRaceSkips_ViolatingFixtureIsCaught verifies the whole check end to end on
// a fixture file with one violation (a race-skipping test no gate F prefix
// selects) among compliant tests: exactly the violation is reported.
//
// Guard: the real check passes on a clean tree, so it never shows it can fail; a
// detector that silently found nothing would also pass.
func TestRaceSkips_ViolatingFixtureIsCaught(t *testing.T) {
	t.Parallel()
	fixture := map[string]string{"fixture_test.go": `package p
func TestStep_TooSlowForRace(t *testing.T) { if raceEnabled { t.Skip("slow") } }
func TestScenarioX_Fine(t *testing.T) { if raceEnabled { t.Skip("slow") } }
func TestStep_NoRaceSkip(t *testing.T) { if testing.Short() { t.Skip("short") } }
`}
	if got := unprefixedRaceSkippers(t, fixture); !slices.Equal(got, []string{"TestStep_TooSlowForRace"}) {
		t.Errorf("unprefixedRaceSkippers(fixture) = %v, want [TestStep_TooSlowForRace]", got)
	}
}

// TestMain_UpdateRefusal runs this test binary again with -update and no tests
// and verifies TestMain exits 2 with a message under -race, and runs cleanly
// (exit 0) without it.
//
// Guard: the refusal is the only barrier between a reduced-seed run and a
// recorded file that says "seeds 1-10"; updateAllowed is unit-tested, but not
// that TestMain calls it before anything runs (plan 2-1 D228).
func TestMain_UpdateRefusal(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable() error = %v, want nil", err)
	}
	cmd := exec.Command(exe, "-update", "-test.run=^$")
	cmd.Dir = t.TempDir() // nothing under the module can be written even if the guard fails
	out, err := cmd.CombinedOutput()
	code := 0
	if ee, ok := err.(*exec.ExitError); ok {
		code = ee.ExitCode()
	} else if err != nil {
		t.Fatalf("run %s -update: %v", exe, err)
	}
	wantCode := 0
	if raceEnabled {
		wantCode = 2
	}
	if code != wantCode {
		t.Errorf("test binary with -update exit code = %d (raceEnabled=%t), want %d; output %q", code, raceEnabled, wantCode, out)
	}
	if refused := strings.Contains(string(out), "-update is refused under -race"); refused != raceEnabled {
		t.Errorf("refusal message printed = %t (raceEnabled=%t), want %t; output %q", refused, raceEnabled, raceEnabled, out)
	}
}
