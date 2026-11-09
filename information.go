package economysim

import (
	"context"
	"log/slog"
	"math"

	"github.com/Whealinghold/economy-sim/internal/codec"
	"github.com/Whealinghold/economy-sim/internal/logkeys"
	"github.com/Whealinghold/economy-sim/internal/rumour"
)

// This file is the information module (spec-information.md): the price record
// of every market, and the delayed, blurred rumours of remote prices that the
// agents of an informed role are shown. The pure rules are in internal/rumour.
// The file holds, in the order of architecture.md 2.10.1: the state, the
// validator and the warnings (the configuration surface), the construction of
// the sources, the method of P5.2, the filler of the trade view and the
// function that makes rumours, and the checkpoint sections.

// Rumour is what an informed agent has heard of a remote market: a reading of
// the price that market published Lag ticks ago, blurred by a bounded error. It
// is never the market's current price, and it carries no stock, volume or quote
// (spec-information.md 5.1).
type Rumour struct {
	// Region is the region the news is about, never the agent's own.
	Region RegionID
	// Commodity is the commodity.
	Commodity CommodityID
	// Price is the market price at the vintage tick, with the error, from 1 to
	// MaxPrice.
	Price Price
	// Mean is the price window mean at the vintage tick, with the same error; 0
	// when there was none.
	Mean Price
	// Lag is the ticks from the vintage tick to the reading: at least 2.
	Lag int64
	// SourceAge is the source's price age at the vintage tick; 0 when units
	// traded that tick. Lag and SourceAge are never changed by the error.
	SourceAge int64
}

// Rumour returns the rumour about the commodity in the region, and false when
// the view carries none (no path, too old, not yet, not a source, or an agent
// that is not informed). It does not allocate (spec-information.md 6.7).
func (v *AgentView) Rumour(region RegionID, c CommodityID) (Rumour, bool) {
	lo, hi := 0, len(v.Rumours)
	for lo < hi {
		mid := int(uint(lo+hi) >> 1)
		if r := &v.Rumours[mid]; r.Region < region || (r.Region == region && r.Commodity < c) {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	if lo < len(v.Rumours) && v.Rumours[lo].Region == region && v.Rumours[lo].Commodity == c {
		return v.Rumours[lo], true
	}
	return Rumour{}, false
}

// information is the state of the information module (nil in the world while
// the module is off): the sources, the price record, which roles are informed
// and the scratch of a view. Per-agent state does not exist (spec-information.md
// 4.6); the record is the only saved state.
type information struct {
	speed, errorBP, maxAge int64
	// sources are the markets that can be the subject of a rumour, in ascending
	// (region, commodity) order, and at the source of each market, as its index
	// plus one (0 is not a source), at region * commodities + commodity.
	sources []rumourSource
	at      []int32
	rec     *rumour.Record
	// informed says, by RoleID, whether the agents of the role are shown rumours.
	informed []bool
	// buf is the scratch of the rumours of one view, sized to the largest informed
	// role's candidates; tally the per-reader-region counts of the tick.
	buf   []Rumour
	tally []rumourTally
}

// rumourSource is a source: a market whose published price is recorded each tick.
type rumourSource struct {
	region    RegionID
	commodity CommodityID
	market    int
}

// rumourQuery asks rumoursFor for the rumours a reader would be offered. all
// considers every region but the reader; otherwise regions are the source regions
// to consider, ascending, and an empty list considers none. commodities are the ascending commodities to consider. asOf is the
// tick of the reading: the current tick for a view, the next for the migration
// pass of phase 7 (spec-information.md 4.3.2).
type rumourQuery struct {
	reader      RegionID
	all         bool
	regions     []RegionID
	commodities []CommodityID
	asOf        int64
}

// rumourTally counts the candidates of the views of one reader region in a tick
// by their result (spec-information.md 5.3.1) for RumoursServed, with the
// readers given a view and the largest lag offered. A nil tally counts nothing.
type rumourTally struct {
	Readers, Offered, NoPath, TooOld, NotYet, MaxLag int64
}

// count adds one candidate with its result and, if offered, its lag.
func (t *rumourTally) count(o rumour.Outcome, lag int64) {
	if t == nil {
		return
	}
	switch o {
	case rumour.NoPath:
		t.NoPath++
	case rumour.TooOld:
		t.TooOld++
	case rumour.NotYet:
		t.NotYet++
	default:
		t.Offered++
		t.MaxLag = max(t.MaxLag, lag)
	}
}

// newInformation builds the state of the module, or nil while it is off. The
// sources are the pairs (region, commodity) whose commodity is in the tradable
// set of some role that has a group in the region (spec-information.md 4.1.1),
// derived once and fixed. It reads a configuration that passed Validate and the
// role table of the world.
func newInformation(c *Config, rt *roleTable) *information {
	m := c.Modules.Information
	if m == nil {
		return nil
	}
	commodities, regions := len(c.Commodities), len(c.Regions)
	active := make([]bool, regions*commodities)
	for _, g := range c.Groups {
		r, _ := c.regionID(g.Region)
		role, _ := c.roleID(g.Role)
		for _, k := range rt.rows[role].tradable {
			active[int(r)*commodities+int(k)] = true
		}
	}
	in := &information{speed: m.InfoSpeed, errorBP: m.RumourErrorBP, maxAge: m.RumourMaxAge,
		at: make([]int32, len(active)), informed: make([]bool, len(c.Roles)), tally: make([]rumourTally, regions)}
	for i, on := range active {
		if on {
			in.sources = append(in.sources, rumourSource{RegionID(i / commodities), CommodityID(i % commodities), i})
			in.at[i] = int32(len(in.sources))
		}
	}
	in.rec = rumour.NewRecord(len(in.sources), int(m.RumourMaxAge))
	most := 0
	for i := range c.Roles {
		if in.informed[i] = c.Roles[i].Informed; in.informed[i] {
			most = max(most, len(rt.rows[i].tradable)*max(0, regions-1))
		}
	}
	in.buf = make([]Rumour, 0, most)
	return in
}

// p52Record is P5.2: after the price windows are pushed and before the first
// outcome, one observation of tick t is appended to the record of every source,
// in source order (spec-information.md 4.3.1): the published price and price age
// of the market and the window mean after the push, 0 while the window is empty.
// Every source is written every tick, a tick in which nothing traded included. It
// emits no event, draws nothing and allocates nothing. Nothing else writes the
// record (FR-INF-006).
func (w *World) p52Record() {
	in, tick := w.info, w.Tick()
	for i := range in.sources {
		m := in.sources[i].market
		mean, _ := w.markets.windows[m].Mean()
		rep := &w.markets.reports[m]
		in.rec.Put(i, tick, rumour.Obs{Price: int64(rep.Price), Mean: mean, Age: rep.Age})
	}
	in.rec.Seal(tick)
}

// fillRumours is the filler of the trade view (spec-information.md 6.2): for an
// agent of an informed role it keys the stream of (siteInform, tick, agent), once
// and only when the error is above 0, and puts the agent's rumours in the view,
// sorted. It runs for an agent that has a view, so an agent that is away, a
// removed agent and the estates get none, and only in the view passed to Trade.
// The draws of the view come before the agent's Trade, which keys its own site
// after, so the two cannot interleave. v.Region is the reader's region.
func (w *World) fillRumours(v *AgentView, a *agent) {
	in := w.info
	if !in.informed[a.role] {
		return
	}
	if in.errorBP > 0 {
		w.rnd.keyed(siteInform, w.Tick(), int64(a.id))
	}
	var tally *rumourTally
	if w.obs.supplied {
		tally = &in.tally[v.Region]
		tally.Readers++
	}
	q := rumourQuery{reader: v.Region, all: true, commodities: w.roles.rows[a.role].tradable, asOf: w.Tick()}
	in.buf = w.rumoursFor(in.buf[:0], q, tally)
	if n := len(in.buf); n > 0 {
		v.Rumours = in.buf[:n:n]
	}
	w.logRumours(a.id, v.Region, in.buf)
}

// logRumours is the Debug line of an informed view (spec-information.md 10.2): the
// agent, its region, the rumours it was offered and the largest lag among them. No
// line is written per draw or per rumour, and the attributes are built only when
// the level is enabled.
func (w *World) logRumours(agent AgentID, region RegionID, rs []Rumour) {
	ctx := context.Background()
	if !w.log.Enabled(ctx, slog.LevelDebug) {
		return
	}
	var lag int64
	for i := range rs {
		lag = max(lag, rs[i].Lag)
	}
	w.log.LogAttrs(ctx, slog.LevelDebug, "rumours served",
		slog.Int64(logkeys.Agent, int64(agent)), slog.Int64(logkeys.Region, int64(region)),
		slog.Int64(logkeys.Rumours, int64(len(rs))), slog.Int64(logkeys.Lag, lag))
}

// p36Rumours is the information module's part of P3.6, run before the demand
// module's: one RumoursServed per reader region in ascending RegionID for each
// region in which an informed agent was given a view this tick, from the tally the
// filler kept. It emits only when an observer was supplied, which is also the only
// time the filler counts, and it clears the tally for the next tick.
func (w *World) p36Rumours() {
	if !w.obs.supplied {
		return
	}
	for r := range w.info.tally {
		t := &w.info.tally[r]
		if t.Readers > 0 {
			w.obs.deliver(RumoursServed{Tick: w.Tick(), Region: RegionID(r), Readers: t.Readers,
				Offered: t.Offered, NoPath: t.NoPath, TooOld: t.TooOld, NotYet: t.NotYet, MaxLag: t.MaxLag})
		}
		*t = rumourTally{}
	}
}

// rumoursFor appends to dst the rumours the reader is offered for the candidates
// of q, in ascending (region, commodity) order, and returns it. A candidate is a
// pair (region, commodity) that is a source, in a region other than the
// reader's. For each, in order: the error is drawn when RumourErrorBP is above 0
// (so the stream does not depend on the graph or the age of the run), then the
// candidate is decided by the cases of spec-information.md 5.3.1, then the
// observation of the vintage tick asOf - Lag is blurred by the error. It does
// not key the stream: the caller keys it (the view with siteInform, the
// migration pass with its own site), which keeps the draws of phases 3 and 7
// apart. tally may be nil.
func (w *World) rumoursFor(dst []Rumour, q rumourQuery, tally *rumourTally) []Rumour {
	if !q.all {
		for _, r := range q.regions {
			dst = w.rumoursFrom(dst, q, r, tally)
		}
		return dst
	}
	for r := range RegionID(len(w.cfg.Regions)) {
		dst = w.rumoursFrom(dst, q, r, tally)
	}
	return dst
}

// rumoursFrom is rumoursFor for the commodities of q in one source region.
func (w *World) rumoursFrom(dst []Rumour, q rumourQuery, r RegionID, tally *rumourTally) []Rumour {
	if r == q.reader {
		return dst
	}
	in := w.info
	var lag int64
	d, path := w.geo.distance(r, q.reader)
	if path {
		var ok bool
		if lag, ok = rumour.Lag(d, in.speed); !ok {
			invariant("the lag of a validated information speed and distance overflowed")
		}
	}
	for _, c := range q.commodities {
		if s := in.at[int(r)*len(w.cfg.Commodities)+int(c)]; s != 0 {
			dst = w.candidate(dst, q, r, c, int(s)-1, lag, path, tally)
		}
	}
	return dst
}

// candidate decides one candidate and appends its rumour if it is offered. lag and
// path are those of the pair (source region, reader), which rumoursFrom computes once.
//
// Only the invariant calls are excluded from coverage (testing.md 9.5); the rest of
// the function is covered. They are violated programmer invariants. The bounds of a
// validated configuration make the lag and the blur fit int64, and the Offered case
// guarantees that the record holds the vintage, so no test can reach them from a
// valid world.
func (w *World) candidate(dst []Rumour, q rumourQuery, r RegionID, c CommodityID, source int, lag int64, path bool, tally *rumourTally) []Rumour {
	in := w.info
	e := int64(0)
	if in.errorBP > 0 {
		e = w.rnd.drawRange(-in.errorBP, in.errorBP)
	}
	out := rumour.Decide(path, lag, in.maxAge, q.asOf)
	tally.count(out, lag)
	if out != rumour.Offered {
		return dst
	}
	o, ok := in.rec.At(source, q.asOf-lag)
	if !ok {
		invariant("the price record does not hold a vintage a rumour was offered for")
	}
	price, ok := rumour.Blur(o.Price, e, MaxPrice)
	mean, okMean := rumour.BlurMean(o.Mean, e, MaxPrice)
	if !ok || !okMean {
		invariant("the error of a validated rumour overflowed")
	}
	return append(dst, Rumour{Region: r, Commodity: c, Price: Price(price), Mean: Price(mean), Lag: lag, SourceAge: o.Age})
}

// validateInformation checks the information module by spec-information.md 9.1
// and 9.2: the bounds of its three fields, a role flagged Informed while the
// module is off, and, only when the bounds hold, the two derived products of
// 9.2 (the cells of the record and the candidates of an informed role). It
// reports nothing about the module when it is off beyond the flag.
func (v *validator) validateInformation() {
	c := &v.c
	m := c.Modules.Information
	if m == nil {
		for i := range c.Roles {
			if c.Roles[i].Informed {
				v.rep.add(ErrModuleDependency, path([]string{"roles", c.Roles[i].ID}, "informed"), "true", "")
			}
		}
		return
	}
	base := []string{"modules", "information"}
	ok := v.inRange(path(base, "info_speed"), m.InfoSpeed, 1, MaxInfoSpeed)
	ok = v.inRange(path(base, "rumour_error_bp"), m.RumourErrorBP, 0, MaxRumourErrorBP) && ok
	ok = v.inRange(path(base, "rumour_max_age"), m.RumourMaxAge, 2, MaxRumourAge) && ok
	if ok {
		v.informationLimits(m)
	}
}

// informationLimits is the derived checks of spec-information.md 9.2, run only
// when the bounds of the module hold: the cells of the record and the candidates
// of each informed role.
func (v *validator) informationLimits(m *Information) {
	c := &v.c
	base := []string{"modules", "information"}
	sets := tradableSets(c, v.lk)
	cells := int64(0)
	for _, set := range regionTradable(c, v.lk, sets) {
		cells += int64(len(set))
	}
	if cells *= m.RumourMaxAge; cells > MaxRumourCells {
		v.rep.add(ErrTooMany, path(base, "rumour_max_age"), i64(cells), i64(MaxRumourCells))
	}
	others := int64(max(0, len(c.Regions)-1))
	for i := range c.Roles {
		if r := &c.Roles[i]; r.Informed && int64(len(sets[i]))*others > MaxRumoursPerView {
			v.rep.add(ErrTooMany, path([]string{"roles", r.ID}, "informed"), i64(int64(len(sets[i]))*others), i64(MaxRumoursPerView))
		}
	}
}

// regionTradable returns, for each region that has a group of a role of c, the
// commodities of c that the roles of its groups can trade (spec-information.md
// 4.1.1: a market is active when some agent of the region can trade it). The
// pairs (region, commodity) are the sources of the record. sets is
// tradableSets(c, lk). It reads a configuration that may not validate, so a
// group of an unknown role or a commodity that is not defined adds nothing.
func regionTradable(c *Config, lk lookup, sets []map[string]bool) map[string]map[string]bool {
	index := make(map[string]int, len(c.Roles))
	for i := range c.Roles {
		index[c.Roles[i].ID] = i
	}
	out := make(map[string]map[string]bool)
	for _, g := range c.Groups {
		i, ok := index[g.Role]
		if !ok || lk.region[g.Region] == nil {
			continue
		}
		if out[g.Region] == nil {
			out[g.Region] = make(map[string]bool)
		}
		for id := range sets[i] {
			if lk.commodity[id] != nil {
				out[g.Region][id] = true
			}
		}
	}
	return out
}

// informationWarnings returns the three warnings of spec-information.md 9.3
// that read the configuration alone, in no particular order:
// information_without_readers, information_without_routes and
// informed_without_sources. rumour_horizon_short needs the shortest distances
// and is horizonWarnings. They read a configuration that may not validate, so
// each guards its lookups.
func informationWarnings(c *Config, lk lookup) []Warning {
	if c.Modules.Information == nil {
		return nil
	}
	base := []string{"modules", "information"}
	var out []Warning
	informed := false
	for i := range c.Roles {
		informed = informed || c.Roles[i].Informed
	}
	if !informed {
		out = append(out, Warning{Path: base, Rule: "information_without_readers"})
	}
	if len(c.Regions) < 2 || len(c.Edges) == 0 {
		out = append(out, Warning{Path: base, Rule: "information_without_routes"})
	}
	return append(out, unsourcedRoles(c, lk)...)
}

// unsourcedRoles raises informed_without_sources for each informed role that
// would never be shown a rumour: in no region where it can be (the regions of
// its groups, every region when it has none) is there another region with a
// source among the commodities it trades.
func unsourcedRoles(c *Config, lk lookup) []Warning {
	sets := tradableSets(c, lk)
	byRegion := regionTradable(c, lk, sets)
	var out []Warning
	for i := range c.Roles {
		r := &c.Roles[i]
		if r.Informed && !hasCandidate(c, r.ID, sets[i], byRegion) {
			out = append(out, Warning{Path: path([]string{"roles", r.ID}, "informed"), Rule: "informed_without_sources"})
		}
	}
	return out
}

// hasCandidate reports whether an agent of the role, in some region where the
// role has a group (any region when it has none), has a candidate: a commodity
// of its tradable set that is a source in another region.
func hasCandidate(c *Config, role string, tradable map[string]bool, byRegion map[string]map[string]bool) bool {
	var homes []string
	for _, g := range c.Groups {
		if g.Role == role {
			homes = append(homes, g.Region)
		}
	}
	if len(homes) == 0 {
		for i := range c.Regions {
			homes = append(homes, c.Regions[i].ID)
		}
	}
	for _, home := range homes {
		for region, set := range byRegion {
			if region == home {
				continue
			}
			for id := range set {
				if tradable[id] {
					return true
				}
			}
		}
	}
	return false
}

// horizonWarnings returns rumour_horizon_short (spec-information.md 9.3): the
// ordered pairs of regions with a walk in the configured graph, no edge status
// applied, whose lag is above RumourMaxAge, so that no rumour is ever offered
// for them. Count is the number of pairs. It needs the shortest distances, so
// Warnings appends it after the structural phases come back clean, as it
// appends the viability warnings; the module being off or the graph not
// building gives nothing.
func horizonWarnings(c *Config) []Warning {
	m := c.Modules.Information
	if m == nil {
		return nil
	}
	g, err := NewGraph(*c)
	if err != nil {
		return nil
	}
	pairs := int64(0)
	for from := range RegionID(g.Len()) {
		for to := range RegionID(g.Len()) {
			d, ok := g.Distance(from, to)
			if !ok || from == to {
				continue
			}
			if lag, ok := rumour.Lag(d, m.InfoSpeed); ok && lag > m.RumourMaxAge {
				pairs++
			}
		}
	}
	if pairs == 0 {
		return nil
	}
	return []Warning{{Path: []string{"modules", "information", "rumour_max_age"}, Rule: "rumour_horizon_short", Count: pairs}}
}

// saveInformation writes tag 16 (spec-checkpoint.md 4.2, spec-information.md 14):
// for each source in ascending order, the number of observations held and the
// observations from the oldest to the newest as Price, Mean and Age. The ring's
// write position, the sources, the lags and the rumours are not saved: the
// position is the tick modulo the depth, and the rest is derived.
func (w *World) saveInformation(cw *codec.Writer) error {
	in := w.info
	var held []rumour.Obs
	for i := range in.sources {
		held = in.rec.Entries(held[:0], i)
		cw.Count(len(held))
		for _, o := range held {
			cw.U(unsigned(o.Price))
			cw.U(unsigned(o.Mean))
			cw.U(unsigned(o.Age))
		}
	}
	return nil
}

// loadInformation reads tag 16 into the record of the world, which was derived
// from the configuration. Every source holds the ticks run, at most the depth: a
// count anywhere else (above the depth, unequal between sources, or not the number
// of ticks written) is corrupt, as is a price below 1 or above MaxPrice, a mean
// above MaxPrice or an age below 0. The entries are placed so that entry i of n is
// the tick T - n + i of a checkpoint at T (rumour.Record.Load).
//
// Excluded from coverage (testing.md 9.5): the two refusals of the record (Resume and
// Load) are unreachable from a restore, which hands them a fresh record, a tick of at
// least 0 and counts already checked here.
func (w *World) loadInformation(rs *restoreCtx, r *codec.Reader) error {
	in := w.info
	want := int(min(rs.tick, int64(in.rec.Depth())))
	if len(in.sources) == 0 && !in.rec.Resume(rs.tick) {
		r.Failf("the record with no source refuses tick %d", rs.tick)
	}
	var entries []rumour.Obs
	for i := range in.sources {
		n := r.Count(3)
		if r.Err() == nil && n != want {
			r.Failf("source %d holds %d observations, want %d (the ticks run, at most the depth %d)", i, n, want, in.rec.Depth())
			return nil
		}
		entries = entries[:0]
		for range n {
			entries = append(entries, readObs(r, i))
		}
		if r.Err() == nil && !in.rec.Load(i, rs.tick, entries) {
			r.Failf("source %d: the record refuses %d observations as of tick %d", i, n, rs.tick)
		}
	}
	return nil
}

// readObs reads one observation of source i of tag 16: a price from 1 to MaxPrice,
// a mean from 0 to MaxPrice and an age of at least 0.
func readObs(r *codec.Reader, i int) rumour.Obs {
	o := rumour.Obs{Price: r.Bounded("price", MaxPrice), Mean: r.Bounded("mean", MaxPrice), Age: r.Bounded("age", math.MaxInt64)}
	if r.Err() == nil && !o.Valid(MaxPrice) {
		r.Failf("source %d holds a price of %d, want 1 to %d", i, o.Price, int64(MaxPrice))
	}
	return o
}

// saveRumours writes tag 22, the integration section of the rumours kept for an
// external agent (spec-information.md 6.3.1, 14). The integration module owns its
// rows and is not built, so the section is present with the information module and
// holds no row: one count, 0. It is registered now so that the bytes of a checkpoint
// do not move when the module fills it.
func (w *World) saveRumours(cw *codec.Writer) error {
	cw.Count(0)
	return nil
}

// loadRumours reads tag 22: a count that must be 0 until the integration module is
// built.
func (w *World) loadRumours(_ *restoreCtx, r *codec.Reader) error {
	if r.Count(1) != 0 {
		r.Failf("rows in the integration section, whose module is not built")
	}
	return nil
}
