package economysim

import (
	"cmp"
	"slices"
)

// This file is the resources module (spec-resources.md; architecture.md 2.10.1):
// its shapes, its canonical form and the lookup of a resource by ID. The pure
// rules (regeneration, abundance, the ration) are in internal/endowment and
// internal/curve. The validator and the warnings are in resources_check.go, the
// tick positions and the take in resources_tick.go. Outside these files the
// module appears only in the catalogue files (config.go, hash.go, bounds.go,
// defaults.go, errors.go, validate.go) and at the named hooks of ADR 0010.

// The shapes of a resource (spec-resources.md 4.2): the closed set of the rules
// by which an endowment regenerates.
const (
	// ShapeConstant is a flow of RatePerYear units a year that does not depend on
	// the stock; with a rate of 0 it is a finite deposit.
	ShapeConstant = "constant"
	// ShapeRefill closes the gap between the stock and the capacity at RateBP a
	// year.
	ShapeRefill = "refill"
	// ShapeLogistic grows in proportion to the stock and to the room left, at
	// RateBP a year, with a Refuge that regrows from nothing.
	ShapeLogistic = "logistic"
)

// canonical returns a deep copy of r in canonical order (spec-config.md 3.4,
// spec-resources.md 3.3.3): resources by ID, knots by share, season entries in
// the calendar's season order (an entry the calendar lacks, which fails
// validation, goes last by name) and rows by region then resource. A nil module
// stays nil, and a module that is on stays non-nil. A row's value that its Has
// flag does not enable is zeroed, so two rows that mean the same hash the same.
func (r *Resources) canonical(calendar []Season) *Resources {
	if r == nil {
		return nil
	}
	out := &Resources{
		Resources:    sortedBy(r.Resources, func(a, b Resource) int { return cmp.Compare(a.ID, b.ID) }),
		Endowments:   sortedBy(r.Endowments, byEndowmentKey),
		HorizonYears: r.HorizonYears,
	}
	order := func(season string) int {
		return slices.IndexFunc(calendar, func(s Season) bool { return s.ID == season })
	}
	for i := range out.Resources {
		res := &out.Resources[i]
		res.Abundance = sortedBy(res.Abundance, func(a, b AbundanceKnot) int { return cmp.Compare(a.Share, b.Share) })
		res.Seasons = sortedBy(res.Seasons, func(a, b SeasonRegen) int {
			ia, ib := order(a.Season), order(b.Season)
			if ia < 0 {
				ia = len(calendar)
			}
			if ib < 0 {
				ib = len(calendar)
			}
			return cmp.Or(cmp.Compare(ia, ib), cmp.Compare(a.Season, b.Season))
		})
	}
	for i := range out.Endowments {
		e := &out.Endowments[i]
		if !e.HasCapacity {
			e.Capacity = 0
		}
		if !e.HasInitial {
			e.Initial = 0
		}
		if !e.HasRegenScale {
			e.RegenScaleBP = 0
		}
	}
	return out
}

// byEndowmentKey orders two rows by region and then resource.
func byEndowmentKey(a, b Endowment) int {
	return cmp.Or(cmp.Compare(a.Region, b.Region), cmp.Compare(a.Resource, b.Resource))
}

// resourceIndex indexes the resources of c by ID without validating (a repeated
// ID keeps its first definition). It is nil when the module is off. The pointers
// point into the caller's slices, so an index is read-only.
func resourceIndex(c *Config) map[string]*Resource {
	r := c.Modules.Resources
	if r == nil {
		return nil
	}
	out := make(map[string]*Resource, len(r.Resources))
	for i := range r.Resources {
		if e := &r.Resources[i]; out[e.ID] == nil {
			out[e.ID] = e
		}
	}
	return out
}
