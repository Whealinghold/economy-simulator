package migration

import (
	"cmp"
	"slices"
)

// Outcome is where a pushed household ends in a pass (spec-migration.md 7.6). The
// eleven values partition the pushed households, so no clamp is silent. The root
// maps each to its reason constant.
type Outcome uint8

// The outcomes, in the order of the table of spec-migration.md 7.6.
const (
	// Started is a household that was admitted and whose journey began.
	Started Outcome = iota
	// NoEstimate is a household whose role has no recipe (6.3.4).
	NoEstimate
	// NoDestination is a household with no candidate region (6.1).
	NoDestination
	// NoRumour is a household whose every candidate failed stage 1.
	NoRumour
	// TooFar is a household whose latest failed stage was 2.
	TooFar
	// NoViable is a household whose latest failed stage was 3.
	NoViable
	// NoGain is a household whose latest failed stage was 4.
	NoGain
	// Bound is a household that the guard refused (7.2).
	Bound
	// OutCap is a household that the outflow cap refused (7.3).
	OutCap
	// Unprovisioned is a household that failed provisioning at every candidate
	// that passed the gates (7.4).
	Unprovisioned
	// InCap is a household for which at least one candidate passed provisioning
	// and every such candidate failed the pipeline cap (7.5).
	InCap

	// Outcomes is the number of outcomes.
	Outcomes = int(InCap) + 1
)

// stageOutcome maps the stage a candidate failed at (1 to 4) to the household's
// outcome when no candidate survives.
var stageOutcome = [...]Outcome{1: NoRumour, 2: TooFar, 3: NoViable, 4: NoGain}

// Candidate is one candidate destination of a pushed household, evaluated: its
// journey time and its estimate Vhat at the rumoured prices (spec-migration.md
// 6.3, 6.4). Ref is the caller's reference to the destination and must increase
// with the destination's ID, because it breaks ties in the ranking. Priced says
// the stage 1 test passed: the household's upkeep there is fully priced and at
// least one recipe of its role is.
type Candidate struct {
	Ref    int
	Priced bool
	K      int64
	Vhat   int64
}

// Params are the figures one household's stages read.
type Params struct {
	// NoEstimate says the household's role has no recipe (6.3.4).
	NoEstimate bool
	// Horizon is MoveHorizon.
	Horizon int64
	// EntryRatio is the viability gate of stage 3, read from the spawner's.
	EntryRatio int64
	// Home is the household's measured home figure V_a.
	Home int64
	// Margin is the household's own margin g_a (Margin).
	Margin int64
}

// Ranked is a candidate that passed all four stages.
type Ranked struct {
	Ref  int
	K    int64
	VEff int64
}

// Rank applies the four stages of spec-migration.md 6.5 to the candidates in
// order and returns, in dst[:0], those that passed all four, by V_eff descending
// and, for equal values, by Ref ascending, a total order. When none passed,
// outcome is the household's outcome: NoEstimate when p.NoEstimate, NoDestination
// when there is no candidate, else the latest stage any candidate failed at (a
// candidate's stage is the first test it fails). A household with a ranking has
// the outcome Started, the one admission may change. ok is false when a product
// overflowed: the household is then returned as NoGain with no ranking, since a
// gain that cannot be computed cannot be shown, and the root counts it so.
func Rank(dst []Ranked, cands []Candidate, p Params) (ranked []Ranked, outcome Outcome, ok bool) {
	ranked = dst[:0]
	if p.NoEstimate {
		return ranked, NoEstimate, true
	}
	if len(cands) == 0 {
		return ranked, NoDestination, true
	}
	latest := 0
	for _, c := range cands {
		stage, veff, ok := stageOf(c, p)
		if !ok {
			return dst[:0], NoGain, false
		}
		if stage == 0 {
			ranked = append(ranked, Ranked{Ref: c.Ref, K: c.K, VEff: veff})
			continue
		}
		latest = max(latest, stage)
	}
	if len(ranked) > 0 {
		slices.SortFunc(ranked, func(a, b Ranked) int {
			return cmp.Or(cmp.Compare(b.VEff, a.VEff), cmp.Compare(a.Ref, b.Ref))
		})
		return ranked, Started, true
	}
	return ranked, stageOutcome[latest], true
}

// stageOf returns the first stage the candidate fails (0 if it passes all four)
// and its discounted estimate. ok is false on overflow.
func stageOf(c Candidate, p Params) (stage int, veff int64, ok bool) {
	switch {
	case !c.Priced:
		return 1, 0, true
	case c.K >= p.Horizon:
		return 2, 0, true
	case c.Vhat < p.EntryRatio:
		return 3, 0, true
	}
	veff, ok = Discount(c.Vhat, c.K, p.Horizon)
	if !ok {
		return 0, 0, false
	}
	clears, ok := Clears(veff, p.Home, p.Margin)
	if !ok {
		return 0, 0, false
	}
	if !clears {
		return 4, 0, true
	}
	return 0, veff, true
}
