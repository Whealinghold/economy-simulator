package economysim

import (
	"context"
	"log/slog"
	"math"

	"github.com/Whealinghold/economy-sim/internal/checked"
	"github.com/Whealinghold/economy-sim/internal/codec"
	"github.com/Whealinghold/economy-sim/internal/logkeys"
	mig "github.com/Whealinghold/economy-sim/internal/migration"
)

// This file is the reporting side of the migration module (spec-migration.md 12.3,
// 13, 14): the warnings of the configuration, and below them the events, logs,
// snapshot rows and the checkpoint encoder (MIG, a continuation of migration.go
// for the Lizard limits, architecture.md 2.6).

// migrationFlowFactor is the factor 3 of the warning migration_flow_window
// (spec-migration.md 11.3, 12.3.1): a rule constant of the model of 16.6, not a
// parameter.
const migrationFlowFactor = 3

// migrationWarnings returns the five warnings of spec-migration.md 12.3 that
// read the configuration alone; migration_provision and migration_far need the
// shortest distances and are migrationRouteWarnings. They read a configuration
// that may not validate, so each guards its inputs. Each carries rule and path
// only (the product of migration_flow_window has no carrier, N-329).
func migrationWarnings(c *Config, lk lookup) []Warning {
	m := c.Modules.Migration
	if m == nil {
		return nil
	}
	base := []string{"modules", "migration"}
	var out []Warning
	if len(c.Regions) < 2 || len(c.Edges) == 0 {
		out = append(out, Warning{Path: base, Rule: "migration_without_routes"})
	}
	roles, longest := migrationRoleWarnings(c, lk)
	out = append(out, roles...)
	// Warnings runs on a configuration that may not validate: the product is checked, not wrapped.
	p, pok := checked.Mul(m.MoveOutShare, longest)
	limit, lok := checked.Mul(migrationFlowFactor, m.MoveGain)
	if pok && lok && p > limit {
		out = append(out, Warning{Path: []string{"modules", "migration", "move_out_share"}, Rule: "migration_flow_window"})
	}
	if m.MoveReviewTicks >= 1 && m.MoveOutShare >= wholeBP/m.MoveReviewTicks && m.MoveInShare >= wholeBP {
		out = append(out, Warning{Path: base, Rule: "migration_uncapped"})
	}
	return out
}

// migrationRoleWarnings returns the warnings that read the mobile roles,
// migration_no_mobile_role and migration_cohort, and the longest effective welfare
// window of a mobile role (0 when none), which migration_flow_window reads.
func migrationRoleWarnings(c *Config, lk lookup) (out []Warning, longest int64) {
	mobile := false
	for i := range c.Roles {
		r := &c.Roles[i]
		if !r.mobile() {
			continue
		}
		mobile = true
		if w, ok := effectiveWindow(c, lk, r); ok {
			longest = max(longest, w)
		}
		if migrationCohort(c, r) {
			out = append(out, Warning{Path: []string{"roles", r.ID}, Rule: "migration_cohort"})
		}
	}
	if !mobile {
		out = append(out, Warning{Path: []string{"modules", "migration"}, Rule: "migration_no_mobile_role"})
	}
	return out, longest
}

// migrationCohort reports whether some group of the role has a scale above 1: its
// agents are cohorts, which move whole (spec-migration.md 3.2.2). A group with no
// scale of its own has the world's.
func migrationCohort(c *Config, r *Role) bool {
	for _, g := range c.Groups {
		scale := g.Scale
		if scale == 0 {
			scale = c.World.Scale
		}
		if g.Role == r.ID && scale > 1 {
			return true
		}
	}
	return false
}

// b2i is 1 for true and 0 for false.
func b2i(b bool) int64 {
	if b {
		return 1
	}
	return 0
}

// migrationRouteWarnings returns migration_provision and migration_far
// (spec-migration.md 12.3): the ordered pairs of regions with a route in the
// configured graph, no edge status applied, whose journey time is above CoverTicks
// and is the horizon or more. Count is the number of pairs. Warnings appends them
// after the structural phases come back clean, as it appends the horizon warning;
// the module being off, a field out of its bound or a graph that does not build
// gives nothing.
func migrationRouteWarnings(c *Config) []Warning {
	m := c.Modules.Migration
	if m == nil || m.MoveSpeed < 1 || m.MoveHorizon < 2 {
		return nil
	}
	g, err := NewGraph(*c)
	if err != nil {
		return nil
	}
	provision, far := journeyPairs(c, g, m)
	var out []Warning
	if provision > 0 {
		out = append(out, Warning{Path: []string{"modules", "migration", "move_speed"}, Rule: "migration_provision", Count: provision})
	}
	if far > 0 {
		out = append(out, Warning{Path: []string{"modules", "migration", "move_horizon"}, Rule: "migration_far", Count: far})
	}
	return out
}

// journeyPairs counts the ordered pairs of regions with a route whose journey time
// is above CoverTicks and the pairs whose journey time is the horizon or more.
func journeyPairs(c *Config, g Graph, m *Migration) (provision, far int64) {
	for from := range RegionID(g.Len()) {
		for to := range RegionID(g.Len()) {
			d, ok := g.Distance(from, to)
			if !ok || from == to {
				continue
			}
			if k, ok := mig.Ticks(d, m.MoveSpeed); ok {
				provision += b2i(k > c.World.CoverTicks)
				far += b2i(k >= m.MoveHorizon)
			}
		}
	}
	return provision, far
}

// outcomeReason is the reason constant of each outcome of a pass (spec-migration.md
// 7.6, 13.2), in the order of the outcomes.
var outcomeReason = [mig.Outcomes]Reason{
	ReasonStarted, ReasonNoEstimate, ReasonNoDestination, ReasonNoRumour, ReasonTooFar,
	ReasonNoViable, ReasonNoGain, ReasonBound, ReasonOutCap, ReasonUnprovisioned, ReasonInCap,
}

// JourneyRow is a migration journey in flight (spec-observability.md 5.3.8): the
// agent, its role, the region its account is still in, the destination, the
// arrival tick and the households it stands for.
type JourneyRow struct {
	Agent    AgentID
	Role     RoleID
	From, To RegionID
	Arrives  int64
	Weight   int64
}

// snapshotMigration fills the journey rows, by agent ID, and the households on a
// journey from and to each region (spec-migration.md 13.4). The journeys are nil
// and the counts 0 with the module off or nobody on the way.
func (w *World) snapshotMigration(s *Snapshot) {
	if w.migration == nil {
		return
	}
	m := w.migration
	for _, id := range m.active {
		a, rec := &w.agents[id], m.journey[id]
		from := w.regionOf(a)
		s.Journeys = append(s.Journeys, JourneyRow{Agent: id, Role: a.role, From: from, To: rec.to, Arrives: rec.arrives, Weight: a.weight})
		s.Regions[from].OnJourneyFrom += a.weight
		s.Regions[rec.to].OnJourneyTo += a.weight
	}
}

// reportMigration is step 5 of the pass (spec-migration.md 10.2, 13): the Debug
// lines and the MigrationStarted events of the departures in the order of
// admission, then, per region in ascending order, the runs of the two Warns and
// MigrationReviewed. An event is built only for an observer and a line only when
// its level is enabled; neither changes a result (13.5).
func (w *World) reportMigration() {
	p := &w.migration.pass
	ctx := context.Background()
	debug := w.log.Enabled(ctx, slog.LevelDebug)
	for i := range p.started {
		s := &p.started[i]
		if debug {
			w.log.LogAttrs(ctx, slog.LevelDebug, "migration started", slog.Int64(logkeys.Agent, int64(s.agent)),
				slog.Int64(logkeys.FromRegion, int64(s.from)), slog.Int64(logkeys.ToRegion, int64(s.to)),
				slog.Int64(logkeys.Households, s.weight), slog.Int64(logkeys.Lag, s.k))
		}
		if w.obs.supplied {
			w.obs.deliver(MigrationStarted{
				Tick: w.Tick(), Agent: s.agent, Role: w.agents[s.agent].role, From: s.from, To: s.to,
				Weight: s.weight, Arrives: s.arrives, Distance: s.distance, Journey: s.k,
				EarnedRatio: s.earned, Home: s.home, Estimate: s.estimate, JobAbandoned: s.jobAbandoned,
			})
		}
	}
	for g := range p.tally {
		w.reviewedRegion(ctx, RegionID(g), &p.tally[g])
	}
}

// declined logs the Debug line of a pushed agent that did not start (spec-migration.md
// 13.2), guarded by Enabled.
func (w *World) declined(a *agent, o mig.Outcome) {
	ctx := context.Background()
	if w.log.Enabled(ctx, slog.LevelDebug) {
		w.log.LogAttrs(ctx, slog.LevelDebug, "migration declined",
			slog.Int64(logkeys.Agent, int64(a.id)), slog.String(logkeys.Reason, string(outcomeReason[o])))
	}
}

// reviewedRegion advances the two diagnostic runs of region g, logs their Warns at
// the tenth consecutive tick and every tenth after (pattern B), and delivers
// MigrationReviewed when any household was pushed or held back by the settling
// rule, any arrival was held or any journey ended. The runs advance with or
// without an observer or a logger: they are saved state.
func (w *World) reviewedRegion(ctx context.Context, g RegionID, t *passTally) {
	m := w.migration
	var warn bool
	m.blockedRun[g], warn = advanceRun(m.blockedRun[g], t.pushed > 0 && t.outcome[mig.Started] == 0, false)
	if warn && w.log.Enabled(ctx, slog.LevelWarn) {
		w.log.LogAttrs(ctx, slog.LevelWarn, "migration blocked", slog.Int64(logkeys.Region, int64(g)),
			slog.Int64(logkeys.Households, t.pushed), slog.String(logkeys.Reason, string(outcomeReason[t.mostly()])))
	}
	m.heldRun[g], warn = advanceRun(m.heldRun[g], t.held > 0, false)
	if warn && w.log.Enabled(ctx, slog.LevelWarn) {
		w.log.LogAttrs(ctx, slog.LevelWarn, "arrivals held", slog.Int64(logkeys.Region, int64(g)), slog.Int64(logkeys.Households, t.held))
	}
	if !w.obs.supplied || t.unsettled+t.pushed+t.held+t.ended == 0 {
		return
	}
	o := &t.outcome
	w.obs.deliver(MigrationReviewed{
		Tick: w.Tick(), Region: g, Unsettled: t.unsettled, Reviewed: t.reviewed, Pushed: t.pushed,
		Started: o[mig.Started], NoEstimate: o[mig.NoEstimate], NoDestination: o[mig.NoDestination],
		NoRumour: o[mig.NoRumour], TooFar: o[mig.TooFar], NoViable: o[mig.NoViable], NoGain: o[mig.NoGain],
		Bound: o[mig.Bound], OutCap: o[mig.OutCap], Unprovisioned: o[mig.Unprovisioned], InCap: o[mig.InCap],
		Held: t.held, Ended: t.ended,
	})
}

// mostly is the outcome other than Started that held back the most households,
// the first in the order of 7.6 on a tie: the reason of the Warn "migration blocked".
func (t *passTally) mostly() mig.Outcome {
	best := mig.NoEstimate
	for o := mig.NoEstimate; int(o) < mig.Outcomes; o++ {
		if t.outcome[o] > t.outcome[best] {
			best = o
		}
	}
	return best
}

// saveMigration writes tag 20 (spec-checkpoint.md 4.2, spec-migration.md 14): the
// two diagnostic runs and the households of journeys ended since the last pass,
// each sparse by region. The journeys are in the agent record (tag 3).
func (w *World) saveMigration(cw *codec.Writer) error {
	m := w.migration
	writeSparseByRegion(cw, m.blockedRun)
	writeSparseByRegion(cw, m.heldRun)
	writeSparseByRegion(cw, m.ended)
	return nil
}

// loadMigration reads tag 20: a run from 1 to the ticks elapsed and a pending end
// count of at least 1, a row at its default or out of order being corrupt (R10).
func (w *World) loadMigration(rs *restoreCtx, r *codec.Reader) error {
	m := w.migration
	readSparseByRegion(r, m.blockedRun, rs.tick)
	readSparseByRegion(r, m.heldRun, rs.tick)
	readSparseByRegion(r, m.ended, math.MaxInt64)
	return nil
}

// journeyOf is the journey of agent id for the agent record of tag 3: the
// destination and the arrival tick, and whether there is one.
func (m *migration) journeyOf(id AgentID) (to RegionID, arrives int64, ok bool) {
	if m == nil || !m.on(id) {
		return 0, 0, false
	}
	return m.journey[id].to, m.journey[id].arrives, true
}

// badJourney reports whether the journey of an agent record cannot be one: it
// heads for the region its account is in, or arrives before tick 1 or later than
// a journey that started at the checkpoint could (the horizon beyond the tick). The
// module being off is checked by the caller.
func (w *World) badJourney(rs *restoreCtx, rec *agentRec, region int) bool {
	return rec.jto == uint64(region) || rec.jarrives < 1 || w.migration != nil && rec.jarrives > rs.tick+w.migration.cfg.MoveHorizon+1
}
