package calibration

import (
	"slices"
	"strings"
)

// Known is the data the record checks need from the registry and from STY
// (plan 2-21 D626). The package never reads a file: the registry test fills every
// field from the parse of the registry rows and of STY 5.11.
type Known struct {
	// Knobs holds every registry row, by its ID (K-BEL12, S-CAL02).
	Knobs map[string]KnobInfo
	// Facts maps a STY fact ID without its prefix (P02) to its class (X O B R P).
	Facts map[string]byte
	// Detectors holds the regime IDs (R5a).
	Detectors map[string]bool
	// Paths maps a configuration path of EXP 3.2.1 that names an instance of a table
	// keyed by an ID, with `*` for each instance segment (role.*.expected_work_ticks),
	// to the registry key of its row (Role.ExpectedWorkTicks). The registry test
	// builds it from the scope-path map; the package reads no file.
	Paths map[string]string
}

// KnobInfo is one registry row as the checks read it.
type KnobInfo struct {
	Class byte   // K, D, S or P
	Key   string // Scope.Field
	Stage string // C1 to C9: the stage that searches or sets it
	Owner string // the owner scope; it splits a stage into sub-stages
	V1x   bool   // flag v1x: no reader in version 1
	Z     bool   // flag z: a base of 0 has no grid
	Off   Off    // the off value of a D row, resolved by the caller
	Moves map[string]MoveSign
}

// MoveSign is the expected sign and its source for one Moves token of a row
// (4.4): a sign from + - 0 ~ ? and a source from S M E H.
type MoveSign struct{ Sign, Source byte }

// knobRef is a registry row with the ID it was found under.
type knobRef struct {
	ID string
	KnobInfo
}

// index finds registry rows by key or by configuration path, in an order that does
// not depend on map iteration.
type index struct {
	byKey    map[string]knobRef
	paths    map[string]string
	patterns []string
}

// newIndex indexes the rows by key; two rows with one key keep the lowest ID.
func newIndex(k Known) index {
	ids := make([]string, 0, len(k.Knobs))
	for id := range k.Knobs {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	x := index{byKey: make(map[string]knobRef, len(ids)), paths: k.Paths}
	for _, id := range ids {
		if _, dup := x.byKey[k.Knobs[id].Key]; !dup {
			x.byKey[k.Knobs[id].Key] = knobRef{id, k.Knobs[id]}
		}
	}
	for p := range k.Paths {
		x.patterns = append(x.patterns, p)
	}
	slices.Sort(x.patterns)
	return x
}

// find returns the row of a registry key, or of a configuration path that one of
// the patterns matches (FR-CLB-050, 8.2).
func (x index) find(key string) (knobRef, bool) {
	if ref, ok := x.byKey[key]; ok {
		return ref, true
	}
	for _, p := range x.patterns {
		if matchPath(p, key) {
			ref, ok := x.byKey[x.paths[p]]
			return ref, ok
		}
	}
	return knobRef{}, false
}

// matchPath reports whether a path has the segments of a pattern, `*` standing for
// any one segment.
func matchPath(pattern, path string) bool {
	ps, ss := strings.Split(pattern, "."), strings.Split(path, ".")
	if len(ps) != len(ss) {
		return false
	}
	for i := range ps {
		if ps[i] != "*" && ps[i] != ss[i] {
			return false
		}
	}
	return true
}
