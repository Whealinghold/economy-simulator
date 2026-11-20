package economysim

import "testing"

// TestLabourRound_DerivesTheDeferredInstruments verifies that the adapter's instruments of
// N-432 (hires formed, positions left unfilled by role, wages paid) are derivable from the
// events alone, so no counter is owed by the core: over a run in which every hire ends, the
// Matched of the rounds equals the HireFormed events, the rows by role add up to the
// totals of the round, and the wages of the rounds equal the Paid of the HireEnded events.
//
// Setup: two regions, each with one farm and one hand; the plan acts at tick 0 only, a hire
// of 3 ticks forms in each region and ends by tick 3.
//
// Requirement: FR-EMP-053, FR-EMP-054
func TestLabourRound_DerivesTheDeferredInstruments(t *testing.T) {
	t.Parallel()
	log := &labourLog{}
	plan := func(v *AgentView) JobRequest {
		if v.Tick.Tick > 0 {
			return JobRequest{NoJob: true}
		}
		return farmWants(1, 100, 0)(v)
	}
	w := hireWorld(t, defaultHire(hireOpts{regions: 2}), plan, WithObserver(log))
	stepN(t, w, 5)
	got := sumRounds(t, log.rounds)
	var endedPaid int64
	for _, e := range log.ended {
		endedPaid += int64(e.Paid)
		if e.Cause != CauseCompleted {
			t.Errorf("HireEnded cause = %v, want completed: every hire of the scenario runs its term", e.Cause)
		}
	}
	if len(log.formed) != 2 || got.matched != int64(len(log.formed)) {
		t.Errorf("hires formed = %d, sum of Matched = %d, want both 2", len(log.formed), got.matched)
	}
	if got.wages == 0 || got.wages != endedPaid {
		t.Errorf("wages of the rounds = %d, Paid of the HireEnded events = %d, want equal and above 0", got.wages, endedPaid)
	}
	if got.hired != got.matched || got.posted != got.filled+got.unfilled || got.filled != got.matched {
		t.Errorf("rows: hired %d, posted %d, filled %d, unfilled %d, want hired = filled = matched %d and posted = filled + unfilled",
			got.hired, got.posted, got.filled, got.unfilled, got.matched)
	}
}

// roundSums is the totals of the rounds that an adapter would add up: the hires matched,
// the wages paid and the rows by role.
type roundSums struct {
	matched, wages, hired, posted, filled, unfilled int64
}

// sumRounds adds up the rounds and fails the test for one whose positions do not split into
// filled and unfilled.
func sumRounds(t *testing.T, rounds []LabourRound) roundSums {
	t.Helper()
	var s roundSums
	for _, r := range rounds {
		s.matched += r.Matched
		s.wages += int64(r.Paid)
		for _, row := range r.Workers {
			s.hired += row.Hired
		}
		for _, row := range r.Employers {
			s.posted += row.Posted
			s.filled += row.Filled
			s.unfilled += row.Unfilled
		}
		if r.Posted != r.Filled+r.Unfilled {
			t.Errorf("LabourRound(tick %d, region %d): posted %d != filled %d + unfilled %d", r.Tick, r.Region, r.Posted, r.Filled, r.Unfilled)
		}
	}
	return s
}
