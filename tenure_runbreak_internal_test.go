package economysim

import "testing"

// TestRentStep_UntitledTickBreaksTheRefusalRuns verifies that an endowment with no title at
// P8.3a has its two refusal runs reset: a tick with no title has no refusal, so the run of
// consecutive ticks with a refusal ends (spec-land.md 13.2), and a title set again later
// starts a new run with its first-of-run Warn.
//
// Requirement: FR-LND-030
//
// Guard: a stale run survives the untitled ticks and resumes from its old count, so the
// first refusal after a re-title is not the first of a run and the Warn of every tenth tick
// lands on the wrong tick.
func TestRentStep_UntitledTickBreaksTheRefusalRuns(t *testing.T) {
	t.Parallel()
	w := newMill(t, rationConfig(10, 5_000, false), WithSeed(1))
	toStart(t, w)
	w.tenure.noTitleRun[0], w.tenure.rentFundsRun[0] = 3, 2
	titles := w.tenure.holds[0]
	w.tenure.holds[0] = nil
	w.p83aRent()
	if got := [2]int64{w.tenure.noTitleRun[0], w.tenure.rentFundsRun[0]}; got != [2]int64{0, 0} {
		t.Errorf("runs after a tick with no title = %v, want [0 0]", got)
	}
	w.tenure.holds[0] = titles
}
