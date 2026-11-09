package indicator

import (
	"fmt"
	"slices"
)

// Warning is a member of the closed set of cautions a result carries beside its
// value, so that a band that says "within 5% of the base" can also say "and not
// computed from stale prices" (spec-indicators.md 9.1.2). A result lists its
// warnings in ascending order. A warning never withholds a measure.
type Warning int

// The warnings.
const (
	// WarnStale marks an index computed partly from carried-forward prices.
	WarnStale Warning = iota + 1
	// WarnClamped marks an index, ratio or growth held at MaxIndexBP.
	WarnClamped
	// WarnThin marks a measure over few ticks with volume.
	WarnThin
	// WarnCohort marks a measure over household rows of weight above 1.
	WarnCohort
	// WarnAbsent marks households left out because their value does not exist.
	WarnAbsent
	// WarnNoRoute marks a pair of regions with no route between them.
	WarnNoRoute
	// WarnBasis marks a depreciation on the retirement basis (7.6.2).
	WarnBasis
	// WarnBinding marks a regional comparison in which a cap or a limit acted
	// (8.7): check the counters before reading a difference between regions.
	WarnBinding
)

// String returns the name of the warning.
func (k Warning) String() string {
	names := [...]string{"stale", "clamped", "thin", "cohort", "absent", "no_route", "basis", "binding"}
	if k >= WarnStale && int(k) <= len(names) {
		return names[k-1]
	}
	return fmt.Sprintf("warning_%d", int(k))
}

// bindingWarning returns WarnBinding when a cap or a limit acted in any region
// of the window, and nothing otherwise (spec-indicators.md 8.7).
func bindingWarning(w Window) []Warning {
	for _, r := range w.Regional {
		if r.Binding.Any() {
			return []Warning{WarnBinding}
		}
	}
	return nil
}

// clampWarning adds WarnClamped to ws, in ascending order, when any of the
// indices was held at MaxIndexBP.
func clampWarning(ws []Warning, ixs ...Index) []Warning {
	for _, ix := range ixs {
		if ix.Clamped && !slices.Contains(ws, WarnClamped) {
			ws = append(ws, WarnClamped)
			slices.Sort(ws)
		}
	}
	return ws
}

// withWarnings returns the base warnings plus the binding one, in ascending
// order.
func withWarnings(w Window, base ...Warning) []Warning {
	out := append(base, bindingWarning(w)...)
	slices.Sort(out)
	return out
}
