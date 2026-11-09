package economysim

import (
	"context"
	"log/slog"
	"slices"

	"github.com/Whealinghold/economy-sim/internal/checked"
	"github.com/Whealinghold/economy-sim/internal/logkeys"
	mig "github.com/Whealinghold/economy-sim/internal/migration"
)

// This file is the migration module (spec-migration.md, MIG): households that
// move between regions on a timed journey. It holds the module's validation and,
// below it, its state, the guard, the arrivals and the life of a journey
// (migration_pass.go is the pass of P7.2, migration_report.go the warnings, the
// events and the checkpoint). The arithmetic is internal/migration; this file
// converts at the call (architecture.md 2.9, 2.10.1).

// migrationCommodities returns K(role) of spec-migration.md 6.1 as commodity IDs
// in ascending order: the upkeep commodities of the role in any region (its own
// list or the world's, and every region's extra upkeep) and, for every recipe on
// its list, the inputs, the required tools and the non-discard outputs. It reads
// a configuration that may not validate, so an unknown recipe adds nothing.
func migrationCommodities(c *Config, lk lookup, r *Role) []string {
	set := map[string]bool{}
	for _, it := range effectiveUpkeep(c, r) {
		set[it.Commodity] = true
	}
	for i := range c.Regions {
		for _, it := range c.Regions[i].ExtraUpkeep {
			set[it.Commodity] = true
		}
	}
	for _, id := range r.Recipes {
		if rc := lk.recipe[id]; rc != nil {
			addRecipeCommodities(set, rc)
		}
	}
	out := make([]string, 0, len(set))
	for id := range set { // the sort below erases the map order
		out = append(out, id)
	}
	slices.Sort(out)
	return out
}

// addRecipeCommodities adds the inputs, the required tools and the non-discard
// outputs of a recipe to set.
func addRecipeCommodities(set map[string]bool, rc *Recipe) {
	for _, it := range rc.Inputs {
		set[it.Commodity] = true
	}
	for _, t := range rc.Requires {
		set[t.Commodity] = true
	}
	for _, o := range rc.Outputs {
		if !o.Discard {
			set[o.Commodity] = true
		}
	}
}

// mobile reports whether the module can move the households of the role: it is
// not Fixed and has a recipe (spec-migration.md 3.3, 3.3.2).
func (r *Role) mobile() bool { return !r.Fixed && len(r.Recipes) > 0 }

// validateMigration is the module's part of phases 2 and 3 of spec-config.md
// 7.1.6 (spec-migration.md 12.1, 12.2): the bounds of each field, the module
// dependency, the region rows and the derived checks. It reports nothing while
// the module is off; Role.Fixed has no error.
func (v *validator) validateMigration() {
	c := &v.c
	m := c.Modules.Migration
	if m == nil {
		return
	}
	base := []string{"modules", "migration"}
	if c.Modules.Information == nil {
		v.rep.add(ErrModuleDependency, base, "information", "")
	}
	fields := []struct {
		key         string
		val, lo, hi int64
	}{
		{"move_speed", m.MoveSpeed, 1, MaxSpeed},
		{"move_review_ticks", m.MoveReviewTicks, 1, MaxMoveReviewTicks},
		{"move_below", m.MoveBelow, 0, MaxBasisPoints},
		{"move_gain", m.MoveGain, 0, MaxBasisPoints},
		{"move_width", m.MoveWidth, 0, MaxBasisPoints},
		{"move_cooldown", m.MoveCooldown, 0, MaxWelfareWindow},
		{"move_horizon", m.MoveHorizon, 2, MaxTravelTicks},
		{"move_out_share", m.MoveOutShare, 0, wholeBP},
		{"move_in_share", m.MoveInShare, 0, wholeBP},
	}
	ok := true
	for _, f := range fields {
		ok = v.inRange(path(base, f.key), f.val, f.lo, f.hi) && ok
	}
	if len(m.Regions) > MaxRegions {
		v.rep.add(ErrTooMany, path(base, "regions"), itoa(len(m.Regions)), itoa(MaxRegions))
		return // the rest is linear in the count it just refused
	}
	rowsOK := v.migrationRows(base, m)
	if ok && rowsOK {
		v.migrationLimits(m)
	}
}

// migrationRows checks the region rows of 11.2: a defined region, once, and the
// two shares in 0 to 10 000. It reports whether every row is sound.
func (v *validator) migrationRows(base []string, m *Migration) bool {
	ok := true
	seen := make(map[string]bool, len(m.Regions))
	for i := range m.Regions {
		r := &m.Regions[i]
		rb := path(base, "region", r.Region)
		if v.lk.region[r.Region] == nil {
			v.rep.add(ErrUnknownRegion, path(rb, "region"), r.Region, "")
			ok = false
		}
		if seen[r.Region] {
			v.rep.add(ErrDuplicateID, path(rb, "region"), r.Region, "")
			ok = false
		}
		seen[r.Region] = true
		ok = v.inRange(path(rb, "out_share"), r.OutShare, 0, wholeBP) && ok
		ok = v.inRange(path(rb, "in_share"), r.InShare, 0, wholeBP) && ok
	}
	return ok
}

// migrationLimits is the derived checks of spec-migration.md 12.2, run only when
// the fields hold: the candidates of each mobile role against the bound of one
// view (ErrTooMany), and the households of the largest region times the largest
// share against MaxQuantity (ErrOverflow). The estimate's own products are
// covered by the unit-cost check of spec-production.md 10.3.
func (v *validator) migrationLimits(m *Migration) {
	c := &v.c
	others := int64(max(0, len(c.Regions)-1))
	for i := range c.Roles {
		r := &c.Roles[i]
		if !r.mobile() {
			continue
		}
		if n := int64(len(migrationCommodities(c, v.lk, r))) * others; n > MaxRumoursPerView {
			v.rep.add(ErrTooMany, path([]string{"roles", r.ID}, "recipes"), i64(n), i64(MaxRumoursPerView))
		}
	}
	share := max(m.MoveOutShare, m.MoveInShare)
	for _, r := range m.Regions {
		share = max(share, r.OutShare, r.InShare)
	}
	byRegion := make(map[string]int64, len(c.Regions))
	most := int64(0)
	for _, g := range c.Groups {
		sum, ok := checked.Add(byRegion[g.Region], g.Households)
		if !ok {
			sum = MaxQuantity + 1
		}
		byRegion[g.Region] = sum
		most = max(most, sum)
	}
	p, ok := checked.Mul(most, share)
	if !ok {
		p = MaxQuantity + 1
	}
	if p > MaxQuantity {
		v.rep.add(ErrOverflow, []string{"modules", "migration", "move_out_share"}, i64(p), i64(MaxQuantity))
	}
}

// onJourneyName is the name of the region_bound refusal of the module
// (spec-regions.md 6.5.2, spec-migration.md 7.2.2); it is also the Reason
// ReasonOnJourney of a refused switch.
const onJourneyName = string(ReasonOnJourney)

// journeyRec is the journey of one agent (spec-migration.md 8.2, 14): live says
// the agent has one, to is the destination and arrives the tick of the arrival,
// fixed at the departure (t + 1 + k). The weight is the agent's own, read when it
// arrives.
type journeyRec struct {
	to      RegionID
	arrives int64
	live    bool
}

// migration is the state of the migration module, nil in the world when it is
// off (spec-migration.md 14, D481). journey is indexed by AgentID and grown on
// demand; active holds the IDs of the agents with a journey in ascending order,
// the iteration of P1.5, kept by binary search with no map. ended, blockedRun and
// heldRun are per region: the households of journeys that ended by a removal or
// a cancel since the last pass (reported once), and the two runs of consecutive
// ticks of the diagnostic Warns. They are saved; everything below them is
// derived at construction or scratch of the tick and is not.
type migration struct {
	cfg *Migration

	journey    []journeyRec
	active     []AgentID
	ended      []int64
	blockedRun []int64
	heldRun    []int64

	// kset is K(role) (spec-migration.md 6.1) by role in ascending commodity order,
	// nil for a role that cannot move; outShare and inShare are the shares of each
	// region once the rows replace the world's.
	kset     [][]CommodityID
	outShare []int64
	inShare  []int64

	// skip is the skip list of the arrival's guard query, held so that the query
	// allocates nothing; arrived holds the due IDs of one P1.5; held counts the
	// households of the arrivals a refusal held in the tick, per region, for the
	// pass that reports them.
	skip    []string
	arrived []AgentID
	held    []int64
	// pass is the scratch and the tally of the pass at P7.2 (migration_pass.go).
	pass passScratch
}

// newMigration builds the state from a canonical configuration that passed
// Validate, or returns nil when the module is off. K(role) is a function of the
// role and the configuration alone (spec-migration.md 6.1).
func newMigration(c *Config) *migration {
	cfg := c.Modules.Migration
	if cfg == nil {
		return nil
	}
	lk := newLookup(c)
	regions := len(c.Regions)
	m := &migration{
		cfg:        cfg,
		ended:      make([]int64, regions),
		blockedRun: make([]int64, regions),
		heldRun:    make([]int64, regions),
		held:       make([]int64, regions),
		outShare:   make([]int64, regions),
		inShare:    make([]int64, regions),
		kset:       make([][]CommodityID, len(c.Roles)),
		skip:       []string{onJourneyName},
	}
	for g := range regions {
		m.outShare[g], m.inShare[g] = cfg.MoveOutShare, cfg.MoveInShare
	}
	for _, row := range cfg.Regions {
		g, _ := c.regionID(row.Region)
		m.outShare[g], m.inShare[g] = row.OutShare, row.InShare
	}
	m.pass = newPassScratch(c, regions)
	for i := range c.Roles {
		r := &c.Roles[i]
		if !r.mobile() {
			continue
		}
		for _, id := range migrationCommodities(c, lk, r) {
			k, _ := c.commodityID(id)
			m.kset[i] = append(m.kset[i], k)
		}
	}
	return m
}

// on reports whether the agent has a journey.
func (m *migration) on(id AgentID) bool {
	return int(id) < len(m.journey) && m.journey[id].live
}

// holds is the test of the on_journey obligation (spec-regions.md 6.5.2): the
// agent has a journey and skip does not name it, by name or by the instance key
// "on_journey:<agent>". The arrival's own query skips the bare name.
func (m *migration) holds(a *agent, skip []string) bool {
	return m.on(a.id) && !skipHas(skip, onJourneyName, int64(a.id))
}

// depart records the journey of agent id to region to, arriving at the tick
// given, and inserts the agent in the ascending list of those on a journey. It is
// called by the departure of the pass; a second journey for one agent is a
// violated invariant of the pass, which excludes agents on a journey.
func (m *migration) depart(id AgentID, to RegionID, arrives int64) {
	if m.on(id) {
		invariant("an agent that is on a journey was given a second journey")
	}
	if int(id) >= len(m.journey) {
		m.journey = slices.Grow(m.journey, int(id)+1-len(m.journey))[:int(id)+1]
	}
	m.journey[id] = journeyRec{to: to, arrives: arrives, live: true}
	at, _ := slices.BinarySearch(m.active, id)
	m.active = slices.Insert(m.active, at, id)
}

// finish deletes the journey of agent id, in the list of those on a journey and
// in the table. It reports whether the agent had one.
func (m *migration) finish(id AgentID) bool {
	if !m.on(id) {
		return false
	}
	m.journey[id] = journeyRec{}
	at, found := slices.BinarySearch(m.active, id)
	if !found {
		invariant("an agent on a journey is missing from the list of those on a journey")
	}
	m.active = slices.Delete(m.active, at, at+1)
	return true
}

// cancelJourney ends the journey of an agent that is still in its origin region
// without arriving: the record is dropped, no balance and no region changes, and
// the households are counted in ended for the next pass to report (spec-migration.md
// 8.3, 8.4.1; FR-MIG-048). It is the removal's step 2a (dropJourney) and the
// seam of INT's Relocate, and it emits nothing. It reports whether there was one.
func (m *migration) cancelJourney(a *agent, region RegionID) bool {
	if !m.finish(a.id) {
		return false
	}
	m.ended[region] += a.weight
	return true
}

// onJourney reports whether the agent has a migration journey: the migration half
// of the one predicate offStage (spec-world.md 6.1.3).
func (w *World) onJourney(a *agent) bool { return w.migration != nil && w.migration.on(a.id) }

// dropJourney is the migration module's step of a removal (spec-agents.md 3.6
// step 2a, FR-AGT-084): the journey is deleted and its households counted as
// ended in the region the account is in, with no event. The money and the lots
// follow to that region's estate in the steps after it.
func (w *World) dropJourney(a *agent, region RegionID) {
	w.migration.cancelJourney(a, region)
}

// p15Arrivals is P1.5: the journeys whose arrival tick has come, in ascending
// agent ID, each completed before the next (spec-migration.md 8.4, FR-MIG-029).
// The guard query is run skipping on_journey itself; another obligation holds
// the arrival, which is counted and retried every tick until it releases.
func (w *World) p15Arrivals() {
	m, tick := w.migration, w.Tick()
	m.arrived = m.arrived[:0]
	for _, id := range m.active {
		if m.journey[id].arrives <= tick {
			m.arrived = append(m.arrived, id)
		}
	}
	for _, id := range m.arrived {
		w.arriveJourney(&w.agents[id], m.journey[id].to)
	}
}

// arriveJourney completes the journey of agent a to region to, or holds it: the
// relocation of the ledger with the cause migration, the beliefs reseeded at the
// new market with no draw, RoleSince the tick, and the record dropped (spec-migration.md
// 8.4, 8.5). A refusal changes nothing and is counted for the pass.
func (w *World) arriveJourney(a *agent, to RegionID) {
	m := w.migration
	if reason, err := w.relocateRefusal(a.id, to, m.skip); err != nil {
		m.held[w.regionOf(a)] += a.weight
		if ctx := context.Background(); w.log.Enabled(ctx, slog.LevelDebug) {
			w.log.LogAttrs(ctx, slog.LevelDebug, "arrival held",
				slog.Int64(logkeys.Agent, int64(a.id)), slog.String(logkeys.Reason, string(reason)))
		}
		return
	}
	// Excluded from coverage (testing.md 9.5): the dry run just passed, so the ledger cannot refuse.
	if err := w.moveAgent(a, to, CauseMigration); err != nil {
		invariant("the ledger refused the relocation of an arriving migrant: " + err.Error())
	}
	w.reseedAtArrival(a, to)
	a.roleSince = w.Tick()
	m.finish(a.id)
}

// newPassScratch sizes the scratch of the pass: the per-region tables, the table of
// rumoured prices, and the arenas of the lines of the recipes of one candidate.
func newPassScratch(c *Config, regions int) passScratch {
	lines, tools := 0, 0
	for i := range c.Recipes {
		lines += len(c.Recipes[i].Inputs) + len(c.Recipes[i].Outputs)
		tools += len(c.Recipes[i].Requires)
	}
	return passScratch{
		households: make([]int64, regions), pipeline: make([]int64, regions),
		tally: make([]passTally, regions), price: make([]int64, regions*len(c.Commodities)),
		caps:  mig.Caps{Out: make([]int64, regions), In: make([]int64, regions), Pipeline: make([]int64, regions)},
		lines: make([]mig.Line, 0, lines+8), tools: make([]mig.Tool, 0, tools+1),
	}
}
