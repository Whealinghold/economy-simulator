package economysim

import (
	"log/slog"

	"github.com/Whealinghold/economy-sim/internal/welfare"
)

// This file is the saving module (spec-saving-and-investment.md; architecture.md
// 2.10.1): its state, which is derived tables and scratch only, and its constructor.
// The continuation files are saving_validate.go (the errors and the warnings),
// saving_view.go (the filler of the Saving group of the Work and Trade views),
// saving_strategy.go (the rules of the default and the two exported functions) and
// saving_report.go (the bars). The arithmetic is in internal/saving. The module has no
// saved state, takes no draw, opens no account and writes no ledger row (spec 9, 13), so
// outside this file and its continuation it appears only in the catalogue files and at
// the named hooks of ADR 0010: the field of World, the construction call, the nil-safe
// call of fillAgentView, and the three guarded calls of DefaultStrategy.

// saving is the state of the module, nil when Config.Modules.Saving is nil. Nothing in
// it is saved or hashed: the tables are derived from the configuration at construction,
// the scratch is rewritten for every view.
type saving struct {
	// module is the module's configuration of the world's canonical Config and log the
	// world's logger, which the Debug lines of the Trade view go to.
	module *Saving
	log    *slog.Logger
	// world is the world the snapshot rows and the query build their views in; counts are
	// the counts of SavingPlanned per (region, role) cell, cleared at P8.6, and nil unless
	// an observer was supplied (observability-and-logging.md 4.5).
	world  *World
	counts []savingCount
	// infra is the infrastructure module the clauses and the classes are read from; nil
	// when it is off, in which case a role has no project and the buffer only.
	infra *infrastructure
	// names are the class IDs of the infrastructure module as the configuration writes
	// them, by FacilityClassID, which a Worth row names.
	names []string
	// roles are the resolved tables by RoleID; a role with no block has a nil block.
	roles []savingRole
	// The working memory of the filler, sized once from the largest role and reused for
	// every view; the world is single-threaded and a view is valid for its call only.
	view    SavingView
	worth   []ProjectWorth
	classes []SavingClass
	needs   []FacilityClassID
	recs    []welfare.Record
}

// attach gives the module its world, after the world is built; it is nil-safe so the call
// adds no branch to the constructor.
func (s *saving) attach(w *World) {
	if s != nil {
		s.world = w
	}
}

// savingRole is what the module resolves of one role at construction: its block and the
// rows of the classes that a Builds or Repairs clause of its recipes names.
type savingRole struct {
	// block is the role's Saver block of the canonical Config, nil for a role with none.
	block *Saver
	// classes are the classes the role's clauses name, ascending by FacilityClassID.
	classes []SavingClass
}

// newSaving resolves the tables of a canonical configuration that passed Validate, or
// returns nil when the module is off. It reads no agent and takes no draw.
func newSaving(c *Config, rt *roleTable, infra *infrastructure, log *slog.Logger, supplied bool) *saving {
	if c.Modules.Saving == nil {
		return nil
	}
	s := &saving{module: c.Modules.Saving, log: log, infra: infra, roles: make([]savingRole, len(c.Roles))}
	if supplied {
		s.counts = make([]savingCount, len(c.Regions)*len(c.Roles))
	}
	if m := c.Modules.Infrastructure; m != nil {
		for i := range m.Classes {
			s.names = append(s.names, m.Classes[i].ID)
		}
	}
	var worth, classes, needs int
	for i := range c.Roles {
		blk := c.Roles[i].Saver
		if blk == nil {
			continue
		}
		sr := &s.roles[i]
		sr.block = blk
		sr.classes = s.classRows(rt.row(RoleID(i)).recipes)
		worth, classes = max(worth, len(blk.Worth)), max(classes, len(sr.classes))
		needs = max(needs, s.needCount(rt.row(RoleID(i)).recipes))
	}
	s.worth, s.classes = make([]ProjectWorth, 0, worth), make([]SavingClass, 0, classes)
	s.needs = make([]FacilityClassID, 0, needs)
	return s
}

// classRows are the rows of the classes that the clauses of the recipes name, in
// ascending FacilityClassID. A role with no infrastructure module has none.
func (s *saving) classRows(recipes []RecipeID) []SavingClass {
	if s.infra == nil {
		return nil
	}
	named := make([]bool, len(s.infra.cls))
	for _, id := range recipes {
		rc := &s.infra.rec[id]
		if rc.build >= 0 {
			named[rc.build] = true
		}
		if rc.repair >= 0 {
			named[rc.repair] = true
		}
	}
	var rows []SavingClass
	for id, on := range named {
		if !on {
			continue
		}
		k := &s.infra.cls[id]
		rows = append(rows, SavingClass{
			Class: FacilityClassID(id), Name: s.names[id], Kind: k.kind,
			Durability: k.durability, Decay: k.decay, WearPerUse: k.wearPerUse, Capacity: k.capacity,
		})
	}
	return rows
}

// needCount is the number of Needs entries over the recipes: the size of the scratch of
// the per-recipe Needs of one view.
func (s *saving) needCount(recipes []RecipeID) int {
	if s.infra == nil {
		return 0
	}
	n := 0
	for _, id := range recipes {
		n += len(s.infra.rec[id].needs)
	}
	return n
}
