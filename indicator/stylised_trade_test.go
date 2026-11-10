package indicator_test

import (
	"context"
	"fmt"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/indicator"
	"github.com/Whealinghold/economy-sim/internal/verdict"
)

// The fact tests of group D, trade and the open layer (plan 2-19 Phase 7, D596),
// tier N: they run only with -stylised.tier=n. P06 and T02 on `two` and `three`
// with the trade, transport, information and merchant overlays (and the control
// without the merchants), S03 with the closure of the middle link, P03 over the
// lane Spread (C-NC), S04, T01 and T03 on `one` with the external overlay, and S05
// on `one` with the faucet. A fact whose module is absent from the pack is a row
// `not applicable` (FR-STY-009).

const tradeTicks = int64(1460) // four years: T02 reads four, P06 the last two

// tradePacks are the packs of P06, S03 and T02, cold, each with the overlays of
// its variant (REF 4.5, 7.6).
var tradePacks = []stylisedPack{
	{"two", []string{"base.toml", "two.toml"}, 4, 1},
	{"three", []string{"base.toml", "three.toml"}, 4, 1},
}

// tradeLayers are the overlays of a variant, in the order their headers give:
// the edges, the transport modes, the rumours, then the merchants unless it is the
// control.
func tradeLayers(variant string, merchants bool) []string {
	l := []string{"overlay/trade-" + variant + ".toml", "overlay/transport-" + variant + ".toml", "overlay/information-" + variant + ".toml"}
	if merchants {
		m := "overlay/merchants.toml"
		if variant == "three" {
			m = "overlay/merchants-three.toml"
		}
		l = append(l, m)
	}
	return l
}

// haulOf is the haul commodity of each transport mode (NoCommodity for a
// self-carried mode), as DeliveredPrice takes it.
func haulOf(cfg economysim.Config, names economysim.Names) []economysim.CommodityID {
	if cfg.Modules.Transport == nil {
		return nil
	}
	out := make([]economysim.CommodityID, len(cfg.Modules.Transport.Modes))
	for i, m := range cfg.Modules.Transport.Modes {
		out[i] = economysim.NoCommodity
		if c := indexOf(names.Commodities, m.Haul); m.Haul != "" && c >= 0 {
			out[i] = economysim.CommodityID(c)
		}
	}
	return out
}

// laneWindow is what P06 reads from one year window: the positive part of the gap
// after freight of each lane that dispatched, in basis points of the delivered
// price, and the Spread of each connected pair and commodity.
type laneWindow struct {
	positives []int64
	spreads   map[[3]int]int64
	mean      indicator.Index
}

func readLanes(r *stylisedRun, yw runWin) (laneWindow, error) {
	lw := laneWindow{spreads: map[[3]int]int64{}}
	p, _, err := windowPrices(r, yw.w)
	if err != nil {
		return lw, err
	}
	dl, err := indicator.DeliveredPrice(yw.w, haulOf(r.cfg, r.names), p)
	if err != nil {
		return lw, err
	}
	gr, err := indicator.PriceGap(yw.w, p, dl, r.graph)
	if err != nil {
		return lw, err
	}
	lw.mean = gr.MeanSpread
	for _, row := range gr.Rows {
		if !row.Connected {
			continue
		}
		if row.Spread.OK {
			lw.spreads[[3]int{int(row.Origin), int(row.Destination), int(row.Commodity)}] = int64(row.Spread.Value)
		}
		if row.AfterFreightOK && row.GapAfterFreightBP.OK {
			lw.positives = append(lw.positives, max(int64(row.GapAfterFreightBP.Value), 0))
		}
	}
	return lw, nil
}

// tradeRun is the memoised run of a trade pack with or without the merchants.
func tradeRun(t testing.TB, p stylisedPack, merchants bool, seed uint64) *stylisedRun {
	t.Helper()
	return customRun(t, p, tradeLayers(p.name, merchants), fmt.Sprintf("trade merchants=%v", merchants), seed, tradeTicks, nil)
}

// p06Facts judges STY-P06 and the lane part of STY-P03 on a trade pack.
func p06Facts(t testing.TB, p stylisedPack) *factSet {
	seeds := packSeeds(t, p)
	return memoFacts(seedKey("p06"+p.name, seeds), func() *factSet {
		f, bands, th := newFactSet(), readBands(t, p.name), readThresholds(t)
		order, median, worst := make([]string, len(seeds)), make([]stat, len(seeds)), make([]stat, len(seeds))
		var regimes []verdict.Verdict
		for i, s := range seeds {
			r, ctl := tradeRun(t, p, true, s), tradeRun(t, p, false, s)
			median[i], worst[i], order[i] = f.p06Seed(r, ctl, seeds[i])
			if v, ok := f.laneR6(r, th); ok {
				regimes = append(regimes, v)
			}
		}
		prior := func(id, st string, v int64) factDef {
			return factDef{ID: id, Statistic: st, Class: verdict.ClassB, Prior: v, HasPrior: true}
		}
		f.add("P06", judgeExact(factDef{ID: "STY-P06", Statistic: "Spread of the trading pair below its control's", Class: verdict.ClassO}, p.name, seeds, order))
		f.add("P06", judge(prior("STY-P06", "x1.margin_bp", 1000), p.name, seeds, median, bands["x1.margin_bp"]))
		f.add("P06", judge(prior("STY-P06", "x1.worst_bp", 3000), p.name, seeds, worst, bands["x1.worst_bp"]))
		f.notApplicable("P06", "STY-P06", "O", "traded relative prices between the autarky ratios", "the pooled autarky ratios of VIA A3 have no accessor outside the solver (ratio_outside_autarky is a finding of the solver, not read here)")
		if len(regimes) > 0 {
			f.regimes["P03 lane Spread"] = regimes[:1]
		}
		return f
	})
}

// p06Seed reads the last two year windows of a merchant run and of its control.
func (f *factSet) p06Seed(r, ctl *stylisedRun, seed uint64) (median, worst stat, order string) {
	median, worst = stat{Why: "no lane dispatched"}, stat{Why: "no lane dispatched"}
	if len(r.years) < 4 || len(ctl.years) < 4 {
		return stat{Why: "windows"}, stat{Why: "windows"}, ""
	}
	var medMax, worstMax int64
	var seen bool
	var last, lastCtl laneWindow
	for _, k := range []int{2, 3} {
		lw, err := readLanes(r, r.years[k])
		if err != nil {
			return stat{Why: err.Error()}, stat{Why: err.Error()}, err.Error()
		}
		if k == 3 {
			last = lw
			if lastCtl, err = readLanes(ctl, ctl.years[k]); err != nil {
				return stat{Why: err.Error()}, stat{Why: err.Error()}, err.Error()
			}
		}
		if len(lw.positives) == 0 {
			continue
		}
		seen = true
		_, med, hi := sortedMedian(lw.positives)
		medMax, worstMax = max(medMax, med), max(worstMax, hi)
	}
	if seen {
		median, worst = stat{Value: medMax}, stat{Value: worstMax}
	}
	switch {
	case !last.mean.OK || !lastCtl.mean.OK:
		order = ""
		f.notes = append(f.notes, fmt.Sprintf("STY-P06 pack %s seed %d: no connected pair with a Spread in the last year; the order is undetermined", r.pack, seed))
	case r.collapsed() || ctl.collapsed():
		f.notes = append(f.notes, fmt.Sprintf("FINDING %s STY-P06 pack %s seed %d: a run has collapsed (merchants %v, control %v); the Spread order is a finding, not a verdict", issueCollapse, r.pack, seed, r.collapsed(), ctl.collapsed()))
	case last.mean.Value >= lastCtl.mean.Value:
		order = fmt.Sprintf("mean Spread %d with the merchants, %d in the control without them", last.mean.Value, lastCtl.mean.Value)
	}
	return median, worst, order
}

// laneR6 applies R6's rule to the Spread of each lane over the year windows (C-NC,
// the lane part: STY 7.1), in place of the price ranges.
func (f *factSet) laneR6(r *stylisedRun, th verdict.Thresholds) (verdict.Verdict, bool) {
	in, err := detectorInput(r)
	if err != nil {
		f.failures = append(f.failures, failMsg("STY-P03", r.pack, r.seed, "detector input", err, "nil"))
		return verdict.Verdict{}, false
	}
	lanes := map[[3]int]*verdict.Series{}
	for _, yw := range r.years {
		lw, err := readLanes(r, yw)
		if err != nil {
			continue
		}
		for k, v := range lw.spreads {
			if lanes[k] == nil {
				lanes[k] = &verdict.Series{}
			}
			lanes[k].Points = append(lanes[k].Points, point(yw.w.From, v))
			lanes[k].Clamped = append(lanes[k].Clamped, false)
		}
	}
	in.Ranges = nil
	for _, k := range slices.SortedFunc(mapKeys3(lanes), cmp3) {
		in.Ranges = append(in.Ranges, *lanes[k])
	}
	v := verdict.Detect(verdict.R6, in, th)
	if v.Class == verdict.Detected && !r.collapsed() {
		f.failures = append(f.failures, failMsg("STY-P03", r.pack, r.seed, "R6 over the lane Spread", v.String(), "none"))
	}
	return v, true
}

func mapKeys3(m map[[3]int]*verdict.Series) func(yield func([3]int) bool) {
	return func(yield func([3]int) bool) {
		for k := range m {
			if !yield(k) {
				return
			}
		}
	}
}

func cmp3(a, b [3]int) int {
	for i := range a {
		if a[i] != b[i] {
			return a[i] - b[i]
		}
	}
	return 0
}

// TestStylised_Trade_P06 judges STY-P06 on `two` and `three` with the trade,
// transport, information and merchant overlays, each against the control without
// the merchants (O: the mean Spread is below the control's; B: the median and the
// worst lane of the gap after freight in each of the last two years), and the lane
// part of STY-P03 (R6 over the lane Spread).
//
// Requirement: FR-STY-025, FR-STY-082, FR-STY-009
func TestStylised_Trade_P06(t *testing.T) {
	t.Parallel()
	requireTierN(t)
	for _, p := range tradePacks {
		t.Run(p.name, func(t *testing.T) {
			t.Parallel()
			f := p06Facts(t, p)
			for _, v := range f.regimes["P03 lane Spread"] {
				t.Logf("pack %s: %s", p.name, v)
			}
			f.report(t)
		})
	}
}

// ---- S03 and T02 ---------------------------------------------------------------

const (
	closeAt  = int64(365)
	reopenAt = int64(545) // s28.reopen_ticks has the prior 180
	s28Ticks = int64(910)
)

// closeMiddleLink closes then reopens the link south-north of `three` (S28),
// through SetEdgeStatuses between steps.
func closeMiddleLink(t testing.TB) func(*capture) {
	return func(c *capture) {
		set := func(w *economysim.World, closed bool) {
			g := w.Graph()
			a, okA := g.Lookup("south")
			b, okB := g.Lookup("north")
			if !okA || !okB {
				t.Fatalf("pack three has no link south-north")
			}
			err := w.SetEdgeStatuses([]economysim.EdgeStatus{{From: a, To: b, Closed: closed, FactorBP: 10_000}, {From: b, To: a, Closed: closed, FactorBP: 10_000}})
			if err != nil {
				t.Fatalf("SetEdgeStatuses(closed=%v) error = %v, want nil", closed, err)
			}
		}
		c.before = func(w *economysim.World) {
			switch w.Tick() {
			case closeAt:
				set(w, true)
			case reopenAt:
				set(w, false)
			}
		}
	}
}

// pairSpread is the mean over the commodities priced in both regions of the
// absolute difference of the two prices as a share of their mean, in basis
// points, averaged over the ticks [from, to).
func pairSpread(r *stylisedRun, a, b string, from, to int64) (int64, bool) {
	ra, rb := indexOf(r.names.Regions, a), indexOf(r.names.Regions, b)
	var sum, n int64
	for tick := from; tick < to; tick++ {
		for c := range r.names.Commodities {
			ma, mb := r.obs.markets[[2]int{ra, c}], r.obs.markets[[2]int{rb, c}]
			if ma == nil || mb == nil || tick >= int64(len(ma.Price)) || tick >= int64(len(mb.Price)) || ma.Price[tick] <= 0 || mb.Price[tick] <= 0 {
				continue
			}
			sum, n = sum+abs(ma.Price[tick]-mb.Price[tick])*20_000/(ma.Price[tick]+mb.Price[tick]), n+1
		}
	}
	if n == 0 {
		return 0, false
	}
	return sum / n, true
}

// s03Facts judges the closure of the middle link of `three` (STY-S03).
func s03Facts(t testing.TB) *factSet {
	p := tradePacks[1]
	seeds := packSeeds(t, p)
	return memoFacts(seedKey("s03", seeds), func() *factSet {
		f, bands := newFactSet(), readBands(t, "three")
		band := bandOrWork(bands, "s28.band_bp", 1500)
		order, events := make([]string, len(seeds)), make([]string, len(seeds))
		back := make([]stat, len(seeds))
		for i, s := range seeds {
			r := customRun(t, p, tradeLayers("three", true), "s28", s, s28Ticks, closeMiddleLink(t))
			events[i] = edgeEvents(r.obs.edges)
			before, okB := pairSpread(r, "south", "north", closeAt-20, closeAt)
			end, okE := pairSpread(r, "south", "north", reopenAt-20, reopenAt)
			back[i] = stat{Why: "no priced pair"}
			if !okB || !okE || r.collapsed() {
				continue
			}
			if end < before {
				order[i] = fmt.Sprintf("Spread %d at the end of the closure, %d at its start", end, before)
			}
			back[i] = stat{Why: "the gap did not return within the run"}
			for tick := reopenAt; tick < s28Ticks; tick++ {
				if m, ok := pairSpread(r, "south", "north", tick-9, tick+1); ok && m <= before+band {
					back[i] = stat{Value: tick - reopenAt}
					break
				}
			}
		}
		f.add("S03", judgeExact(factDef{ID: "STY-S03", Statistic: "Spread at the end of the closure at least its value at the start", Class: verdict.ClassO}, "three", seeds, order))
		f.add("S03", judgeExact(factDef{ID: "STY-S03", Statistic: "EdgeStatusChanged at both changes", Class: verdict.ClassX}, "three", seeds, events))
		f.add("S03", judge(factDef{ID: "STY-S03", Statistic: "s28.reopen_ticks", Class: verdict.ClassB, Prior: 180, HasPrior: true}, "three", seeds, back, bands["s28.reopen_ticks"]))
		f.notApplicable("S03", "STY-S03", "O", "households of the cut region against the closed-region variant", "the households of one region are not kept per tick; the aggregate series is, and the control is not run")
		return f
	})
}

// edgeEvents is the X part of S03: the closure and the reopening each emit one
// EdgeStatusChanged per direction.
func edgeEvents(ev []economysim.EdgeStatusChanged) string {
	var closed, open int
	for _, e := range ev {
		if e.Closed {
			closed++
		} else {
			open++
		}
	}
	if closed != 2 || open != 2 {
		return fmt.Sprintf("%d events closing and %d reopening an edge, want 2 and 2", closed, open)
	}
	return ""
}

// TestStylised_Trade_S03 closes the link south-north of `three` for 180 ticks
// (S28) and judges STY-S03: the Spread of the pair does not fall while the road is
// closed (O), EdgeStatusChanged is emitted at both changes (X) and the gap returns
// within s28.band_bp of its earlier value (B, the ticks it takes against
// s28.reopen_ticks).
//
// Requirement: FR-STY-037, FR-STY-082, FR-REG-049
func TestStylised_Trade_S03(t *testing.T) {
	t.Parallel()
	requireTierN(t)
	s03Facts(t).report(t)
}

// pooledShares are the shares of the households of each role in each region in the
// pooled steady state (the balanced census of WithFreeTrade), in basis points.
func pooledShares(t testing.TB, cfg economysim.Config) map[[2]string]int64 {
	t.Helper()
	ss, err := economysim.SolveSteadyState(context.Background(), cfg, economysim.WithFreeTrade())
	if err != nil {
		t.Fatalf("SolveSteadyState(free trade) error = %v, want nil", err)
	}
	total := map[string]int64{}
	for _, row := range ss.Report.Census {
		if row.Role != "" {
			total[row.Region] += row.Balanced
		}
	}
	out := map[[2]string]int64{}
	for _, row := range ss.Report.Census {
		if row.Role != "" && total[row.Region] > 0 {
			out[[2]string{row.Region, row.Role}] = row.Balanced * 10_000 / total[row.Region]
		}
	}
	return out
}

// shareDistance is the sum over the roles of a region of the absolute difference
// between its role share and the pooled share, in basis points; the result is the
// largest over the regions.
func shareDistance(r *stylisedRun, w indicator.Window, pooled map[[2]string]int64, atClose bool) (int64, bool) {
	rs, err := indicator.RoleShares(w)
	if err != nil {
		return 0, false
	}
	per := map[int]int64{}
	for _, row := range rs.Rows {
		sh := row.OpenShare
		if atClose {
			sh = row.CloseShare
		}
		if !sh.OK {
			continue
		}
		reg, role := r.names.Regions[row.Region], r.names.Roles[row.Role]
		per[int(row.Region)] += abs(int64(sh.Value) - pooled[[2]string{reg, role}])
	}
	var top int64
	for _, v := range per {
		top = max(top, v)
	}
	return top, len(per) > 0
}

// t02Facts judges STY-T02 on `three` with the trade overlays: the distance of the
// role shares to the pooled census is smaller at the end of four years than at
// the start (O) and within x2.share_band_bp (B).
func t02Facts(t testing.TB) *factSet {
	p := tradePacks[1]
	seeds := packSeeds(t, p)
	return memoFacts(seedKey("t02", seeds), func() *factSet {
		f, bands := newFactSet(), readBands(t, "three")
		order, end := make([]string, len(seeds)), make([]stat, len(seeds))
		pooled := pooledShares(t, loadPack(t, append(slices.Clone(p.files), tradeLayers("three", true)...)...))
		for i, s := range seeds {
			r := tradeRun(t, p, true, s)
			end[i] = stat{Why: "windows"}
			if len(r.years) < 4 {
				continue
			}
			d0, ok0 := shareDistance(r, r.years[0].w, pooled, false)
			d1, ok1 := shareDistance(r, r.years[3].w, pooled, true)
			if !ok0 || !ok1 || r.collapsed() {
				end[i] = stat{Why: "no household in the last year"}
				continue
			}
			end[i] = stat{Value: d1}
			if d1 >= d0 {
				order[i] = fmt.Sprintf("distance to the pooled census %d at the end, %d at the start", d1, d0)
			}
		}
		f.add("T02", judgeExact(factDef{ID: "STY-T02", Statistic: "distance to the pooled census smaller at the end", Class: verdict.ClassO}, "three", seeds, order))
		f.add("T02", judge(factDef{ID: "STY-T02", Statistic: "x2.share_band_bp", Class: verdict.ClassB, Prior: 1500, HasPrior: true}, "three", seeds, end, bands["x2.share_band_bp"]))
		return f
	})
}

// TestStylised_Trade_T02 judges STY-T02 on `three` over four years with the trade
// overlays: the occupation shares move toward the pooled census of free trade.
//
// Requirement: FR-STY-047, FR-STY-082
func TestStylised_Trade_T02(t *testing.T) {
	t.Parallel()
	requireTierN(t)
	t02Facts(t).report(t)
}

// ---- the open layer: S04, T01, T03 on `one` + external -------------------------

const (
	openTicks = int64(730)
	cutAt     = int64(400)
)

// cutGate closes the edges of the gate region to the outside node at cutAt and
// polls the households of the gate region from then on for their first unfed
// tick, remembering what each held of each upkeep good at the cut (STY-S04).
type cutGate struct {
	t       testing.TB
	region  economysim.RegionID
	held    map[economysim.AgentID]int64 // ticks of cover: the least over the upkeep goods that do not spoil
	first   map[economysim.AgentID]int64
	closeOK bool
}

func (g *cutGate) apply(w *economysim.World) {
	if w.Tick() == cutAt {
		g.cut(w)
	}
	if w.Tick() > cutAt {
		g.poll(w)
	}
}

// poll runs before the step of w.Tick()+1, so a household found unfed here was
// unfed in step w.Tick().
func (g *cutGate) poll(w *economysim.World) {
	ids, err := w.Agents(g.region)
	if err != nil {
		g.t.Fatalf("Agents(%d) error = %v, want nil", g.region, err)
	}
	for _, id := range ids {
		if _, seen := g.first[id]; seen {
			continue
		}
		if st, err := w.Agent(id); err == nil && st.Unfed > 0 {
			g.first[id] = w.Tick()
		}
	}
}

func (g *cutGate) cut(w *economysim.World) {
	gr := w.Graph()
	var changes []economysim.EdgeStatus
	for to := range economysim.RegionID(gr.Len()) {
		if to == g.region {
			continue
		}
		if e, ok := gr.Edge(g.region, to); ok && !e.Closed {
			changes = append(changes, economysim.EdgeStatus{From: g.region, To: to, Closed: true, FactorBP: 10_000},
				economysim.EdgeStatus{From: to, To: g.region, Closed: true, FactorBP: 10_000})
		}
	}
	if len(changes) == 0 {
		g.t.Fatalf("the gate region has no edge to close")
	}
	if err := w.SetEdgeStatuses(changes); err != nil {
		g.t.Fatalf("SetEdgeStatuses(close the gate) error = %v, want nil", err)
	}
	ids, err := w.Agents(g.region)
	if err != nil {
		g.t.Fatalf("Agents(%d) error = %v, want nil", g.region, err)
	}
	cfg := w.Config()
	for _, id := range ids {
		st, err := w.Agent(id)
		if err != nil {
			continue
		}
		cover, any, spoils := int64(0), false, false
		for _, u := range st.Upkeep {
			if u.Quantity <= 0 {
				continue
			}
			if cfg.Commodities[u.Commodity].ShelfLife > 0 {
				spoils = true // a spoiling good falls faster than the need: the household is excluded (STY 5.3)
				break
			}
			var have int64
			for _, h := range st.Holdings {
				if h.Commodity == u.Commodity {
					have = int64(h.Quantity)
				}
			}
			per := u.Quantity * st.Weight
			if per <= 0 {
				continue // a forged zero weight: no cover to read
			}
			if c := have / per; !any || c < cover {
				cover, any = c, true
			}
		}
		if any && !spoils {
			g.held[id] = cover
		}
	}
	g.closeOK = true
}

// openRun is the memoised run of `one` with the external overlay and the cut gate.
func openRun(t testing.TB, seed uint64) (*stylisedRun, *cutGate) {
	t.Helper()
	var g *cutGate
	r := customRun(t, packOne, []string{"overlay/external.toml"}, "open", seed, openTicks, func(c *capture) {
		g = &cutGate{t: t, held: map[economysim.AgentID]int64{}, first: map[economysim.AgentID]int64{}}
		g.region = economysim.RegionID(indexOf(c.run.names.Regions, "north"))
		c.before = g.apply
	})
	return r, g
}

// openFacts judges S04 (O: no household unfed before the cut plus the cover it
// held), T01 (X: the audit holds with the outside) and T03 (R: the share of ticks
// the price is within t03.quote_band_bp of the quote).
func openFacts(t testing.TB) *factSet {
	seeds := packSeeds(t, packOne)
	return memoFacts(seedKey("open", seeds), func() *factSet {
		f := newFactSet()
		early, audit, pull := make([]string, len(seeds)), make([]string, len(seeds)), make([]stat, len(seeds))
		for i, s := range seeds {
			r, g := openRun(t, s)
			audit[i] = errText(r.stepErr)
			for id, tick := range g.first {
				if cover, ok := g.held[id]; ok && tick < cutAt+cover {
					early[i] = fmt.Sprintf("agent %d unfed at tick %d, before the cut %d plus its cover %d", id, tick, cutAt, cover)
					break
				}
			}
			pull[i] = quoteShare(r)
		}
		f.add("T01", judgeExact(factDef{ID: "STY-T01", Statistic: "the audit holds with an outside node", Class: verdict.ClassX}, "one", seeds, audit))
		f.add("S04", judgeExact(factDef{ID: "STY-S04", Statistic: "first unfed tick at least the cut plus the cover held", Class: verdict.ClassO}, "one", seeds, early))
		f.add("T03", judge(factDef{ID: "STY-T03", Statistic: "t03.pull_bp", Class: verdict.ClassR, Side: atLeast, Prior: 5000, HasPrior: true}, "one", seeds, pull, verdict.BandKey{}))
		return f
	})
}

// quoteShare is the share of the ticks of the first year, from the first quote,
// in which the price of a commodity with a quote is within 1 000 bp (the prior of
// t03.quote_band_bp, a labelled working value) of it; the largest over the lines.
func quoteShare(r *stylisedRun) stat {
	best, seen := int64(0), false
	for k, q := range r.obs.quote {
		m := r.obs.markets[k]
		if m == nil {
			continue
		}
		var near, n int64
		for tick := range min(int64(len(q)), int64(len(m.Price)), yearTicks) {
			if q[tick] <= 0 {
				continue
			}
			n++
			if !outsideBand(m.Price[tick], q[tick], 1000) {
				near++
			}
		}
		if n > 0 {
			seen, best = true, max(best, near*10_000/n)
		}
	}
	if !seen {
		return stat{Why: "no quote was present"}
	}
	return stat{Value: best}
}

// TestStylised_Open_S04T01T03 runs `one` with the external overlay for 730 ticks,
// closes the gate edges at tick 400 and judges STY-S04 (no household is unfed
// before it has used its stock), STY-T01 (the audit holds with the outside) and
// STY-T03 (the pull of a quote, a report row with no band: the decision input of
// N-113).
//
// Requirement: FR-STY-038, FR-STY-039, FR-STY-082, FR-STY-009
func TestStylised_Open_S04T01T03(t *testing.T) {
	t.Parallel()
	requireTierN(t)
	openFacts(t).report(t)
}

// ---- S05: the faucet -------------------------------------------------------------

const (
	faucetTicks = int64(1095)
	faucetOffAt = int64(730)
)

// faucetRun is `one` with the faucet overlay whose flow is scaled to 0 at tick
// faucetOffAt (SetFlowScales).
func faucetRun(t testing.TB, seed uint64) *stylisedRun {
	t.Helper()
	return customRun(t, packOne, []string{"overlay/faucet.toml"}, "faucet", seed, faucetTicks, func(c *capture) {
		var flows []string
		if ms := c.run.cfg.Modules.MoneySupply; ms != nil {
			for _, fl := range ms.Flows {
				flows = append(flows, fl.ID)
			}
		}
		c.before = func(w *economysim.World) {
			if w.Tick() != faucetOffAt {
				return
			}
			var scales []economysim.FlowScale
			for _, id := range flows {
				scales = append(scales, economysim.FlowScale{Flow: id, BP: 0})
			}
			if err := w.SetFlowScales(scales); err != nil {
				t.Fatalf("SetFlowScales(0) error = %v, want nil", err)
			}
		}
	})
}

// faucetAccounts is the X part of S05: over the year windows, the closing boundary
// balance of each flow's reason equals minus the cumulative total of the flow, and
// the money of the realm is its start plus what the faucets issued less the sinks.
func faucetAccounts(r *stylisedRun) string {
	flows := map[economysim.Reason]bool{}
	for _, fl := range r.cfg.Modules.MoneySupply.Flows {
		flows[economysim.Reason(fl.ID)] = true
	}
	cum := map[economysim.Reason]int64{}
	var issued int64
	for _, yw := range r.years {
		p, _, err := windowPrices(r, yw.w)
		if err != nil {
			return err.Error()
		}
		sc, err := indicator.Sectors(yw.w, yw.open, yw.closing, p)
		if err != nil {
			return err.Error()
		}
		for _, fl := range sc.Flows {
			cum[fl.Reason] += fl.Total
			if fl.Kind == economysim.KindFaucet {
				issued += fl.Total
			}
		}
		for _, b := range sc.Balances {
			if flows[b.Reason] && b.Commodity == economysim.NoCommodity && b.Balance != -cum[b.Reason] {
				return fmt.Sprintf("boundary balance of reason %s is %d, want minus the flow total %d at tick %d", b.Reason, b.Balance, -cum[b.Reason], yw.w.To)
			}
		}
		if got, want := sc.Realm.Money, int64(r.start.Money)+issued; got != want {
			return fmt.Sprintf("money %d at tick %d, want the start %d plus the issue %d", got, yw.w.To, r.start.Money, issued)
		}
	}
	return ""
}

func faucetFacts(t testing.TB) *factSet {
	seeds := packSeeds(t, packOne)
	return memoFacts(seedKey("faucet", seeds), func() *factSet {
		f := newFactSet()
		acc := make([]string, len(seeds))
		for i, s := range seeds {
			r := faucetRun(t, s)
			if acc[i] = errText(r.stepErr); acc[i] == "" {
				acc[i] = faucetAccounts(r)
			}
			f.notes = append(f.notes, faucetDrift(r, s)...)
		}
		f.add("S05", judgeExact(factDef{ID: "STY-S05", Statistic: "boundary balance equals minus the flow total and M = M_init + issued", Class: verdict.ClassX}, "one", seeds, acc))
		return f
	})
}

// faucetDrift reports the price response of S05 (R, no band): the Fisher link
// between successive year windows, the velocity and the money per household.
func faucetDrift(r *stylisedRun, seed uint64) []string {
	var out []string
	var prev indicator.Prices
	var prevW indicator.Window
	for k, yw := range r.years {
		p, _, err := windowPrices(r, yw.w)
		if err != nil {
			return append(out, fmt.Sprintf("STY-S05 seed %d: %v", seed, err))
		}
		tv, err := indicator.Turnover(yw.w, yw.closing, r.cfg.Calendar.TicksPerYear(), indicator.CellSel{})
		if err != nil {
			return append(out, fmt.Sprintf("STY-S05 seed %d: %v", seed, err))
		}
		line := fmt.Sprintf("STY-S05 (report, no band) seed %d year %d: velocity %v", seed, k+1, tv.Velocity)
		if k > 0 {
			if base, err := indicator.BaseOf(prevW, prev); err == nil {
				if res, err := indicator.PriceIndex(yw.w, base, indicator.WeightsConsumed, p, indicator.CellSel{}); err == nil {
					line += fmt.Sprintf(", Fisher link %d bp", res.Fisher.Value)
				}
			}
		}
		out = append(out, line)
		prev, prevW = p, yw.w
	}
	return out
}

// TestStylised_Faucet_S05 runs `one` with the faucet overlay for 1 095 ticks, with
// the flow scaled to 0 at tick 730, and judges STY-S05: the money a flow creates is
// exactly its boundary balance and the money supply moves by exactly it (X, every
// seed); the price drift, the velocity and the Fisher link are logged with no band.
//
// Requirement: FR-STY-039, FR-STY-082, FR-MON-043
func TestStylised_Faucet_S05(t *testing.T) {
	t.Parallel()
	requireTierN(t)
	faucetFacts(t).report(t)
}
