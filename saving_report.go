package economysim

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/Whealinghold/economy-sim/internal/logkeys"
)

// This file is the reporting part of the saving module (spec-saving-and-investment.md 7.3,
// 8.4.2, 12): the Debug lines of the project of a household, the counts and the event
// SavingPlanned (P3.6b), the four snapshot rows and the query Saving. Every figure is the
// pure rule of saving_strategy.go on the view, so the counts, the rows, the query and the
// default cannot disagree (FR-SAV-030, 031, 037). It draws nothing and changes no saved
// state (a continuation of saving.go, architecture.md 2.6).

// SavingState is what the query Saving returns for an agent whose role has a Saver block
// (spec 7.3): the floor, the precaution buffer, the drawdown, the goal and the idle money of
// the agent in minor units, and the project chosen with its benefit, cost, outlay and funded
// flag, or the keep of a job in progress. It is a function of saved state with the tier 2
// spend of a plan read as 0, so a restored world answers as the original did.
type SavingState struct {
	// Reserve is the flat reserve ReserveMoney, Floor the greater of it and the precaution,
	// Precaution the buffer of 4.2 and Drawdown the deepest fall of the agent's welfare ring
	// (4.1).
	Reserve, Floor, Precaution, Drawdown Money
	// Goal is the money set aside for the project chosen, or the keep of a job in progress
	// (Building), or 0; Idle is max(0, Money - S*), the money no rule holds for a purpose.
	Goal, Idle Money
	// Project is the project chosen, valid when Chosen is true.
	Project ProjectChoice
	Chosen  bool
	// Building says a job of a project recipe is in progress, so none is chosen and the goal
	// is its keep (5.5.6).
	Building bool
	// Bar says why no project is chosen: one of the four bars of 12.3, or empty.
	Bar SavingBar
}

// savingCount is the counts of SavingPlanned of one (region, role) cell in a tick (spec
// 12.1), households by the sum of weights.
type savingCount struct {
	households, saving, buffered, passed, building   int64
	noReturn, standing, unreachable, survival        int64
	funded, committed, builds, repairs, goal, buffer int64
}

// quiet reports whether no triggering count is above 0: Households, Goal, Buffer, Funded and
// Committed ride along and never trigger the event.
func (c *savingCount) quiet() bool {
	return c.saving == 0 && c.buffered == 0 && c.passed == 0 && c.building == 0 && c.noReturn == 0 &&
		c.standing == 0 && c.unreachable == 0 && c.survival == 0 && c.builds == 0 && c.repairs == 0
}

// add counts one household of the view, weighted, under the pick of its project (spec 12.1).
// A household is in at most one of survival, building, passed, unreachable, no return and
// standing, in that precedence, and one with no project on its list is in none.
func (c *savingCount) add(v *AgentView, pk *savingPick) {
	w, res := v.Weight, pk.target(0)
	c.households = satAdd(c.households, w)
	if res.Goal > 0 {
		c.saving = satAdd(c.saving, w)
	}
	if buf := pk.floor - pk.reserve; buf > 0 {
		c.buffered, c.buffer = satAdd(c.buffered, w), satAdd(c.buffer, buf)
	}
	c.goal = satAdd(c.goal, res.Goal)
	if pk.first < 0 {
		return
	}
	switch {
	case pk.bar == BarSurvival:
		c.survival = satAdd(c.survival, w)
	case pk.building:
		c.building = satAdd(c.building, w)
	case pk.chosen:
		c.passed = satAdd(c.passed, w)
		if pk.choice.Funded {
			c.funded = satAdd(c.funded, w)
		}
		if pk.choice.Committed {
			c.committed = satAdd(c.committed, w)
		}
	default:
		c.addBar(pk.bar, w)
	}
}

// addBar counts a household none of whose projects was chosen under its bar.
func (c *savingCount) addBar(bar SavingBar, w int64) {
	switch bar {
	case BarUnreachable:
		c.unreachable = satAdd(c.unreachable, w)
	case BarNoReturn:
		c.noReturn = satAdd(c.noReturn, w)
	case BarStanding:
		c.standing = satAdd(c.standing, w)
	}
}

// noteTrade is the world's evaluation of a household given a Trade view: the counts of the
// event under the observer flag and the Debug line of the project, from one pick of the
// rule. It is nil-safe through its caller, takes no draw and builds nothing when neither is
// wanted.
func (s *saving) noteTrade(v *AgentView) {
	debug := s.log.Enabled(context.Background(), slog.LevelDebug)
	if s.counts == nil && !debug {
		return
	}
	reserve, upkeep := v.savingBasis()
	pk := pickProject(v, reserve, upkeep)
	if s.counts != nil {
		s.counts[int(v.Region)*len(s.roles)+int(v.Role)].add(v, &pk)
	}
	if debug {
		s.logProject(v, &pk)
	}
}

// noteRequest counts the request Work returned for an agent of a role with a block as a
// build or a repair request, one for each agent (spec 12.1). It is nil-safe, counts only
// under the observer flag, and reads the request, whatever strategy made it.
func (s *saving) noteRequest(a *agent, region RegionID, req *JobRequest) {
	if s == nil || s.counts == nil || req.NoJob || s.infra == nil || s.roles[a.role].block == nil {
		return
	}
	if req.Recipe < 0 || int(req.Recipe) >= len(s.infra.rec) {
		return
	}
	c, rc := &s.counts[int(region)*len(s.roles)+int(a.role)], &s.infra.rec[req.Recipe]
	switch {
	case rc.repair >= 0:
		c.repairs++
	case rc.build >= 0:
		c.builds++
	}
}

// clearTick empties the counts at P8.6 with the other per-tick scratch (spec 9.1). It is
// nil-safe.
func (s *saving) clearTick() {
	if s == nil {
		return
	}
	clear(s.counts)
}

// p36bSaving is P3.6b: the event SavingPlanned for each (region, role) with a triggering
// count, in ascending region then role order, after RoomLimited (spec 9.1, 12.1). The counts
// exist only when an observer was supplied, so with none it does nothing.
func (w *World) p36bSaving() {
	s := w.saving
	if s.counts == nil {
		return
	}
	roles := len(w.cfg.Roles)
	for cell := range s.counts {
		c := &s.counts[cell]
		if c.quiet() {
			continue
		}
		w.obs.deliver(SavingPlanned{
			Tick: w.Tick(), Region: RegionID(cell / roles), Role: RoleID(cell % roles),
			Households: c.households, Goal: Money(c.goal), Buffer: Money(c.buffer), Funded: c.funded,
			Committed: c.committed, Saving: c.saving, Buffered: c.buffered, Passed: c.passed,
			Building: c.building, NoReturn: c.noReturn, Standing: c.standing,
			Unreachable: c.unreachable, Survival: c.survival, Builds: c.builds, Repairs: c.repairs,
		})
	}
}

// logProject is the Debug line of the project of a household in the Trade view
// (spec-saving-and-investment.md 12.2): "project chosen" with the goal and the buffer, or
// "project declined" with the bar, or for a standing refusal the world's reason of
// spec-infrastructure.md 4.2.2. The strategy has no logger, so the world logs it where it
// builds the view; the rule is the pure one. The goal is the project's goal before the plan's
// tier 2 spend, which only Trade knows and nets from it (savingTarget), so it can be larger
// than the goal the spend cap reads in a tick that plans the project. The caller has checked
// that Debug is enabled; it takes no draw and changes nothing.
func (s *saving) logProject(v *AgentView, pk *savingPick) {
	ctx := context.Background()
	switch {
	case pk.chosen:
		res := pk.target(0)
		s.log.LogAttrs(ctx, slog.LevelDebug, "project chosen", slog.Int64(logkeys.Tick, v.Tick.Tick),
			slog.Int64(logkeys.Agent, int64(v.Agent)), slog.Int64(logkeys.Recipe, int64(pk.choice.Recipe)),
			slog.Int64(logkeys.Goal, res.Goal), slog.Int64(logkeys.Buffer, res.Precaution))
	case pk.bar != "" && pk.first >= 0:
		reason := string(pk.bar)
		if pk.bar == BarStanding {
			reason = string(pk.standing)
		}
		s.log.LogAttrs(ctx, slog.LevelDebug, "project declined", slog.Int64(logkeys.Tick, v.Tick.Tick),
			slog.Int64(logkeys.Agent, int64(v.Agent)), slog.Int64(logkeys.Recipe, int64(pk.first)),
			slog.String(logkeys.Reason, reason))
	}
}

// stateOf is the saving state of one agent of a role with a block, from the Work view the
// boundary builds for it. The anchors and the moment of the boundary are in place
// (prepareAnchors), as for Snapshot.
func (s *saving) stateOf(a *agent) (SavingState, error) {
	w := s.world
	w.refreshViewCtx()
	v := &w.tk.view
	if err := w.fillAgentView(v, a, viewWork); err != nil {
		return SavingState{}, fmt.Errorf("saving view of agent %d: %w", a.id, err)
	}
	reserve, upkeep := v.savingBasis()
	pk := pickProject(v, reserve, upkeep)
	res := pk.target(0)
	return SavingState{
		Reserve: Money(pk.reserve), Floor: Money(res.Floor), Precaution: Money(res.Precaution), Drawdown: Money(v.Saving.Drawdown),
		Goal: Money(res.Goal), Idle: Money(max(0, int64(v.Money)-res.Target)), Project: res.Project,
		Chosen: res.Chosen, Building: res.Building, Bar: res.Bar,
	}, nil
}

// fillRoleRows sets the four saving fields of the role rows of one region from the agents of
// its view, for the agents of a role with a block (spec 8.4.2). It is nil-safe, so a world
// with the module off pays nothing, and it reads only saved state: a pass of the pure rules
// over the agents with a block (cost class Q).
func (s *saving) fillRoleRows(rows []RoleRow, v *RegionView) error {
	if s == nil {
		return nil
	}
	for i := range v.Agents {
		ra := &v.Agents[i]
		if s.roles[ra.Role].block == nil {
			continue
		}
		st, err := s.stateOf(&s.world.agents[ra.Agent])
		if err != nil {
			return err
		}
		row := &rows[ra.Role]
		if st.Goal > 0 {
			row.SavingHouseholds = satAdd(row.SavingHouseholds, ra.Weight)
		}
		row.GoalMoney = Money(satAdd(int64(row.GoalMoney), int64(st.Goal)))
		row.BufferMoney = Money(satAdd(int64(row.BufferMoney), int64(max(0, st.Floor-st.Reserve))))
		row.IdleMoney = Money(satAdd(int64(row.IdleMoney), int64(st.Idle)))
	}
	return nil
}

// Saving returns the saving state of an agent whose role has a Saver block (spec 7.3,
// spec-integration.md 6.7): the floor, the precaution, the drawdown, the goal and the idle
// money, and the project chosen with its benefit, cost, outlay and funded flag, or the keep
// of a job in progress. It returns ErrInStep, ErrWorldFailed, ErrModuleOff (the saving module
// is off), ErrUnknownAgent, ErrAgentRemoved, then ErrNotSaver with the zero value for an
// agent whose role has no block. It draws nothing, emits nothing, changes no saved state and
// answers a restored world as the original (FR-SAV-037, 040).
func (w *World) Saving(id AgentID) (SavingState, error) {
	if err := w.enter("Saving", "saving", w.saving != nil); err != nil {
		return SavingState{}, err
	}
	a, err := w.queryAgent(id)
	if err != nil {
		return SavingState{}, fmt.Errorf("Saving: agent %d: %w", id, err)
	}
	if w.saving.roles[a.role].block == nil {
		return SavingState{}, fmt.Errorf("Saving: agent %d: %w", id, ErrNotSaver)
	}
	if _, err := w.prepareAnchors(); err != nil {
		return SavingState{}, fmt.Errorf("Saving: %w", err)
	}
	st, err := w.saving.stateOf(a)
	if err != nil {
		return SavingState{}, fmt.Errorf("Saving: %w", err)
	}
	return st, nil
}
