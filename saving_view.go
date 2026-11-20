package economysim

import (
	"github.com/Whealinghold/economy-sim/internal/welfare"
)

// fillView sets the Saving group of the Work and Trade views of an agent whose role has
// a Saver block, and the clause fields of its recipes (spec-saving-and-investment.md
// 5.2). It is nil-safe, so the one call of fillAgentView adds no branch, and a role with
// no block is left untouched: its view has no group and its recipes no clause field.
// Nothing here draws, logs or allocates beyond the scratch grown to the largest role.
func (s *saving) fillView(v *AgentView, a *agent, kind viewKind) {
	if s == nil {
		return
	}
	sr := &s.roles[a.role]
	if sr.block == nil {
		return
	}
	sv := &s.view
	*sv = SavingView{
		SafetyTicks: s.module.SafetyTicks, HurdleSpreadBP: s.module.HurdleSpreadBP, CommitBandBP: s.module.CommitBandBP,
	}
	sv.Block = *sr.block
	s.worth = append(s.worth[:0], sr.block.Worth...)
	sv.Block.Worth = s.worth[:len(s.worth):len(s.worth)]
	s.fillDrawdown(sv, a)
	s.fillFacilities(sv, v, a, kind)
	s.classes = append(s.classes[:0], sr.classes...)
	sv.Classes = s.classes[:len(s.classes):len(s.classes)]
	s.fillClauses(v, a)
	v.Saving = sv
}

// fillDrawdown sets the drawdown of the agent's ring and the figures the reach test reads:
// the records and their cumulative income less cost. The records are read into scratch, so
// the cost is O(window) and steady-state allocation is none (spec 4.1.4, NFR-SAV-002).
func (s *saving) fillDrawdown(sv *SavingView, a *agent) {
	sv.Drawdown, s.recs = welfare.DrawdownRing(&a.history, s.recs)
	sv.Records, sv.Surplus = int64(len(s.recs)), welfare.Cumulative(s.recs)
}

// fillFacilities sets the agent's own facilities of the Saving group. The Work view has them
// from the infrastructure filler already; the Trade view does not, so for an agent with a
// block the same filler is called for it (spec 5.2.1, N-196, N-382). The group shares the
// slice of the view.
func (s *saving) fillFacilities(sv *SavingView, v *AgentView, a *agent, kind viewKind) {
	if kind == viewTrade {
		s.infra.fillView(v, a, viewWork)
	}
	sv.Facilities = v.Facilities
}

// fillClauses sets the clause fields of every recipe of the view: the Needs, Builds and
// Repairs of the recipe's clauses and, for a recipe with a Builds or Repairs clause, the
// Standing reason that CAP derives (spec 5.2.2). It is nothing without the infrastructure
// module.
func (s *saving) fillClauses(v *AgentView, a *agent) {
	if s.infra == nil {
		return
	}
	s.needs = s.needs[:0]
	for i := range v.Recipes {
		r := &v.Recipes[i]
		if r.OwnUse {
			continue
		}
		rc := &s.infra.rec[r.Recipe]
		if n := len(rc.needs); n > 0 {
			lo := len(s.needs)
			s.needs = append(s.needs, rc.needs...)
			r.Needs = s.needs[lo : lo+n : lo+n]
		}
		if rc.build >= 0 {
			r.Builds = BuildView{On: true, Class: rc.build, Toward: rc.toward}
		}
		if rc.repair >= 0 {
			r.Repairs = RepairView{On: true, Class: rc.repair, Units: rc.units}
		}
		if rc.build >= 0 || rc.repair >= 0 {
			r.Standing = s.infra.standingReason(a.id, v.Region, r.Recipe)
		}
	}
}

// class is the row of the class of a clause, or nil.
func (sv *SavingView) class(id FacilityClassID) *SavingClass {
	for i := range sv.Classes {
		if sv.Classes[i].Class == id {
			return &sv.Classes[i]
		}
	}
	return nil
}

// worth is the Worth row that names the class, or nil.
func (sv *SavingView) worth(name string) *ProjectWorth {
	for i := range sv.Block.Worth {
		if sv.Block.Worth[i].Class == name {
			return &sv.Block.Worth[i]
		}
	}
	return nil
}
