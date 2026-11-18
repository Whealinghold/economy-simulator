package economysim

import (
	"errors"
	"slices"
	"testing"
)

// facRecord returns the record with the ID, for a case that breaks one field of it.
func facRecord(w *World, id FacilityID) *facility { return &w.infra.fac[w.infra.index(id)] }

// TestAudit_FacilityIdentities verifies that the audit names each way the records can
// disagree with themselves, one rule broken at a time on a live world that passes it: the
// IDs out of order, repeated or not below the next ID, a class, a route target or an
// owner the record cannot have, an estate that holds a store facility, a list by owner
// that differs from the records, an owner above the per-agent bound, and an installed
// count with no store behind it. A world with the module off has nothing to audit.
//
// Requirement: FR-CAP-041
//
// Guard: the audit is the one place that sees a drift between the records and the lists
// and tables rebuilt from them. Without a case per rule a check could be deleted and
// every other test would still pass.
func TestAudit_FacilityIdentities(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		mut  func(w *World)
	}{
		{"IDs not ascending", func(w *World) { w.infra.fac[0], w.infra.fac[1] = w.infra.fac[1], w.infra.fac[0] }},
		{"ID repeated", func(w *World) { w.infra.fac[1].id = w.infra.fac[0].id }},
		{"ID not below the next", func(w *World) { w.infra.next = w.infra.fac[len(w.infra.fac)-1].id }},
		{"class out of range", func(w *World) { w.infra.fac[0].class = FacilityClassID(len(w.infra.cls)) }},
		{"condition above durability", func(w *World) { w.infra.fac[0].condition = MaxDurability }},
		{"route target not joined", func(w *World) { facRecord(w, roadID(w)).to = facRecord(w, roadID(w)).region }},
		{"target on a ground facility", func(w *World) { facRecord(w, facilityOf(w, clsDrain)).to = 1 }},
		{"region not a region", func(w *World) { w.infra.fac[0].region = RegionID(w.infra.regions) }},
		{"estate holds a store facility", func(w *World) {
			f := facRecord(w, barnID(w))
			f.owner = AgentID(f.region)
		}},
		{"owner not live", func(w *World) { w.infra.fac[0].owner = AgentID(len(w.agents)) }},
		{"installed owner in another region", func(w *World) { facRecord(w, barnID(w)).region = 0 }},
		{"owner missing from its list", func(w *World) { w.infra.byOwner[w.infra.fac[0].owner] = nil }},
		{"list holds an extra ID", func(w *World) {
			o := w.infra.fac[0].owner
			w.infra.byOwner[o] = append(w.infra.byOwner[o], w.infra.next+1)
		}},
		{"owner above the per-agent bound", func(w *World) {
			// Real records of a household, so that only the bound is broken and the lists
			// still match (an estate has no bound).
			o := facMiller(w)
			for w.infra.held(o) <= MaxFacilitiesPerAgent {
				w.infra.create(clsMill, 0, NoRegion, o, 1, w.Tick())
			}
		}},
		{"installed count with no store", func(w *World) {
			f := facRecord(w, barnID(w))
			w.sto.take(f.owner, 0)
		}},
		{"installed count of a pair with no record", func(w *World) {
			w.infra.installed[ownerClass{owner: w.infra.fac[0].owner, class: clsMill}] = 0
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w := plantedFacilityWorld(t)
			if err := w.auditInfrastructure(); err != nil {
				t.Fatalf("seed %d: auditInfrastructure before the case error = %v, want nil", facSeed, err)
			}
			tt.mut(w)
			if err := w.auditInfrastructure(); err == nil {
				t.Errorf("seed %d: auditInfrastructure after %q error = nil, want an error", facSeed, tt.name)
			}
		})
	}
	t.Run("module off", func(t *testing.T) {
		t.Parallel()
		w := newMill(t, facConfig(nil), WithSeed(facSeed))
		w.cfg.Modules.Infrastructure, w.infra = nil, nil
		if err := w.auditInfrastructure(); err != nil {
			t.Errorf("auditInfrastructure with the module off error = %v, want nil", err)
		}
	})
}

// TestAudit_FacilityTotalBound verifies that more facilities than MaxFacilities is an
// audit error, on a world whose records are otherwise valid.
//
// Requirement: FR-CAP-041
func TestAudit_FacilityTotalBound(t *testing.T) {
	t.Parallel()
	w := plantedFacilityWorld(t)
	w.infra.fac = slices.Grow(w.infra.fac, MaxFacilities)[:MaxFacilities+1]
	if err := w.auditInfrastructure(); err == nil {
		t.Errorf("seed %d: auditInfrastructure with %d records error = nil, want an error", facSeed, len(w.infra.fac))
	}
}

// TestCheckpoint_FacilitiesDecodeRefusals verifies the refusals of tag 27 that the decoder
// itself makes while reading a record, before the audit: a repeated or descending ID, a
// class, region or owner the frame cannot hold, a built tick above the checkpoint's tick,
// a route target on the region itself, an estate holding a store facility and an
// installed store facility whose owner is in another region. Each is ErrCheckpointCorrupt,
// each case breaking only that rule.
//
// Requirement: FR-CAP-041
func TestCheckpoint_FacilitiesDecodeRefusals(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		mut  func(w *World)
	}{
		{"ID repeated", func(w *World) { w.infra.fac[1].id = w.infra.fac[0].id }},
		{"IDs descending", func(w *World) { w.infra.fac[1].id, w.infra.fac[0].id = w.infra.fac[0].id, w.infra.fac[1].id }},
		{"class out of range", func(w *World) { w.infra.fac[0].class = FacilityClassID(len(w.infra.cls)) }},
		{"region out of range", func(w *World) { w.infra.fac[0].region = RegionID(w.infra.regions) }},
		{"built after the checkpoint", func(w *World) { w.infra.fac[0].built = w.Tick() + 1 }},
		{"route target on its own region", func(w *World) { facRecord(w, roadID(w)).to = facRecord(w, roadID(w)).region }},
		{"estate holds a store facility", func(w *World) {
			f := facRecord(w, barnID(w))
			f.owner = AgentID(f.region)
		}},
		{"installed owner in another region", func(w *World) { facRecord(w, barnID(w)).region = 0 }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w := plantedFacilityWorld(t)
			tt.mut(w)
			cp, err := w.Checkpoint()
			if err != nil {
				t.Fatalf("Checkpoint error = %v, want nil", err)
			}
			if _, err := RestoreWorld(w.Config(), cp); !errors.Is(err, ErrCheckpointCorrupt) {
				t.Errorf("seed %d: RestoreWorld = %v, want ErrCheckpointCorrupt", facSeed, err)
			}
		})
	}
}

// TestFacilityMessage_KindsWithALine verifies the constant Debug message of each kind
// that has one and that a kind with none (a facility added, transferred or adjusted: the
// event tells it) returns the empty string, so no line is logged for it.
//
// Requirement: FR-CAP-039
func TestFacilityMessage_KindsWithALine(t *testing.T) {
	t.Parallel()
	tests := []struct {
		kind FacilityKind
		want string
	}{
		{FacilityBuilt, "facility built"},
		{FacilityRepaired, "facility repaired"},
		{FacilityRuined, "facility ruined"},
		{FacilityInstalled, "facility installed"},
		{FacilityAdded, ""},
		{FacilityTransferred, ""},
		{FacilityAdjusted, ""},
	}
	for _, tt := range tests {
		if got := facilityMessage(tt.kind); got != tt.want {
			t.Errorf("facilityMessage(%q) = %q, want %q", tt.kind, got, tt.want)
		}
	}
}
