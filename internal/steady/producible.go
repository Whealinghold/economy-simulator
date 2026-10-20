package steady

// Maker is an existing activity, for the producibility fixpoint of
// spec-viability.md 4.2: the needs are its inputs and its tools (a tool is an
// input), the makes its non-discard outputs. The caller lists only activities
// that exist (J above 0) and leaves out an own-use-only recipe, which is no
// source (4.2). Listed marks an activity a role with households lists, whose
// needs V1 requires to be producible.
type Maker struct {
	Region int
	Needs  []int
	Makes  []int
	Listed bool
}

// Realm is the data of the producibility fixpoint. Sourced marks the commodities
// that are sourced everywhere: a Raw marker and the module sources the caller
// adds (4.2.1). A maker with no needs is a source by itself.
type Realm struct {
	Commodities int
	Regions     int
	Sourced     []bool
	Makers      []Maker
}

// Producibility is the result of the fixpoint. Region[r][c] holds when c is
// producible in region r from that region's own activities (autarky). Realm[c]
// holds when c is producible in the realm: the optimistic reading of 4.6, free
// trade, in which an input made in any region counts for a recipe in any other.
type Producibility struct {
	Region [][]bool
	Realm  []bool
}

// fixpoint runs the producibility fixpoint. With pooled false each region has
// its own vector and a maker reads only its own region's; with pooled true there
// is one vector that every region shares.
func fixpoint(r Realm, pooled bool) [][]bool {
	n := r.Regions
	if pooled {
		n = 1
	}
	known := make([][]bool, n)
	for i := range known {
		known[i] = make([]bool, r.Commodities)
		copy(known[i], r.Sourced)
	}
	for changed := true; changed; {
		changed = false
		for _, m := range r.Makers {
			slot := m.Region
			if pooled {
				slot = 0
			}
			if !allKnown(known[slot], m.Needs) {
				continue
			}
			for _, c := range m.Makes {
				if !known[slot][c] {
					known[slot][c] = true
					changed = true
				}
			}
		}
	}
	return known
}

// allKnown reports whether every commodity in need is marked in known.
func allKnown(known []bool, need []int) bool {
	for _, c := range need {
		if !known[c] {
			return false
		}
	}
	return true
}

// Producible computes the fixpoint of 4.2 on the existing activities
// (FR-VIA-004). It is O(commodities) passes over the activities and draws
// nothing.
func Producible(r Realm) Producibility {
	return Producibility{Region: fixpoint(r, false), Realm: fixpoint(r, true)[0]}
}

// Missing is a commodity that is needed and not producible in the realm.
// By is the lowest index of a Listed maker that needs it, or -1 when only an
// upkeep needs it: the one recipe that names it.
type Missing struct {
	Commodity int
	By        int
}

// Unproducible returns, in ascending commodity order, each commodity of the
// upkeep, or needed by a Listed maker, that is not producible in the realm
// (rule unproducible_in_realm, FR-VIA-004). A cycle with no source is reported
// whole, every commodity on it being needed by the next.
func Unproducible(r Realm, upkeep []int) []Missing {
	by := make([]int, r.Commodities)
	for i := range by {
		by[i] = -1
	}
	needed := make([]bool, r.Commodities)
	for _, c := range upkeep {
		needed[c] = true
	}
	for i, m := range r.Makers {
		if !m.Listed {
			continue
		}
		for _, c := range m.Needs {
			needed[c] = true
			if by[c] < 0 {
				by[c] = i
			}
		}
	}
	realm := fixpoint(r, true)[0]
	var out []Missing
	for c := 0; c < r.Commodities; c++ {
		if needed[c] && !realm[c] {
			out = append(out, Missing{Commodity: c, By: by[c]})
		}
	}
	return out
}

// RoleUse is a role for the runnable check: the regions (ascending) where it has
// households and the recipes it lists.
type RoleUse struct {
	Regions []int
	Recipes []int
}

// Stuck names a role that cannot work and the first region where none of its
// recipes exists.
type Stuck struct {
	Role   int
	Region int
}

// CannotWork returns, in role order, each role that lists a recipe yet has
// households in a region where none of its recipes exists (rule
// role_cannot_work, FR-VIA-005). exists[region][recipe] says whether the recipe
// exists in the region (J above 0). A role with no recipes is exempt.
func CannotWork(roles []RoleUse, exists [][]bool) []Stuck {
	var out []Stuck
	for i, role := range roles {
		if len(role.Recipes) == 0 {
			continue
		}
		for _, reg := range role.Regions {
			if !anyExists(exists[reg], role.Recipes) {
				out = append(out, Stuck{Role: i, Region: reg})
				break
			}
		}
	}
	return out
}

// anyExists reports whether any of the recipes exists in the row.
func anyExists(row []bool, recipes []int) bool {
	for _, r := range recipes {
		if row[r] {
			return true
		}
	}
	return false
}
