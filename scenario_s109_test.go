package economysim

import (
	"fmt"
	"log/slog"
	"testing"
)

// s109Seeds are the seeds of TestScenarioS109_ClosedTenureAndTheOwnUseExemption: the world is
// four households, so every seed runs under -race too.
var s109Seeds = []uint64{1, 2, 3}

// s109Ticks is the length of a run: enough ticks for a refusal to repeat past the first
// Warn and reach the tenth tick of a run.
const s109Ticks = 12

// s109Totals are the sums over a run of what the scenario asserts and reports.
type s109Totals struct {
	blocked, started, noTitle, rentFunds, pricedOut, paid, exempt int64
	warns                                                         map[string]int
}

// s109Run runs the forage world (four own-use households, a field of 100 units titled by the
// edit) for s109Ticks ticks on a seed with the audit on, and returns the totals of the
// OwnUseChanged and RentPaid events and the "tenure refusal" lines by reason.
func s109Run(t *testing.T, seed uint64, edit func(*Config)) s109Totals {
	t.Helper()
	obs, lines := &evLog{}, newCapture()
	w := newForage(t, 4, edit, WithSeed(seed), WithAudit(), WithObserver(obs), WithLogger(slog.New(lines)))
	step(t, w, s109Ticks)
	tot := s109Totals{warns: map[string]int{}}
	for _, e := range of[OwnUseChanged](obs) {
		tot.blocked += e.Blocked
		tot.started += e.Started
	}
	for _, e := range of[RentPaid](obs) {
		tot.noTitle += e.NoTitle
		tot.rentFunds += e.RentFunds
		tot.pricedOut += e.PricedOut
		tot.paid += int64(e.Paid)
		tot.exempt += e.Exempt
	}
	for _, l := range refusalLines(lines) {
		tot.warns[l.strs["reason"]]++
	}
	return tot
}

// TestScenarioS109_ClosedTenureAndTheOwnUseExemption verifies LND-S3 of spec-land.md 16.4 on
// the world of four own-use households and a titled field (the forage world of the subsistence
// tests): when the pool holds the whole field and the endowment is closed, the default
// strategy reads the title bound of its view (no_title, 4.6.2) and asks for nothing, so the
// households are blocked and no rent is paid; when the pool lets the field at a rent the
// households cannot pay, their requests are refused for rent funds and priced out, counted in
// RentPaid and warned at the first tick of a run and at the tenth and no more often; with
// OwnUseExempt they pay no rent and start. The subsistence module counts the blocked
// households (more with closure than with the exemption) and the audit holds every tick.
// Migration is not in this world (no overlay combines it with the pack of the module: the
// question is recorded in the PR result), and the amounts are reported, not asserted (N-352).
//
// Requirement: FR-LND-014, FR-LND-018, FR-LND-030, FR-LND-046
func TestScenarioS109_ClosedTenureAndTheOwnUseExemption(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	for _, seed := range s109Seeds {
		t.Run(fmt.Sprintf("seed %d", seed), func(t *testing.T) {
			t.Parallel()
			open := s109Run(t, seed, forageField(100))
			closed := s109Run(t, seed, forageTenure(10_000, true, false))
			priced := s109Run(t, seed, forageTenure(10_000, false, false))
			exempt := s109Run(t, seed, forageTenure(10_000, false, true))
			s109Refusals(t, seed, open, closed, priced)
			s109Warns(t, seed, priced)
			s109Exemption(t, seed, closed, exempt)
			t.Logf("seed %d: closed %+v, priced out %+v, exempt %+v (reported, not asserted beyond the signs: N-352)", seed, closed, priced, exempt)
		})
	}
}

// s109Refusals checks the open, closed and priced-out runs of one seed.
func s109Refusals(t *testing.T, seed uint64, open, closed, priced s109Totals) {
	t.Helper()
	if open.blocked != 0 || open.noTitle != 0 || open.rentFunds != 0 {
		t.Errorf("seed %d: open access %+v, want nothing blocked or refused", seed, open)
	}
	if closed.blocked == 0 || closed.paid != 0 {
		t.Errorf("seed %d: closed %+v, want households blocked and no rent paid", seed, closed)
	}
	if priced.blocked == 0 || priced.rentFunds+priced.pricedOut == 0 || priced.paid != 0 {
		t.Errorf("seed %d: priced out %+v, want households blocked, refused for rent funds or priced out, and no rent paid", seed, priced)
	}
}

// s109Warns checks that the Warn "tenure refusal" of the priced-out run is written at the
// first tick of a run and at the tenth and no more often.
func s109Warns(t *testing.T, seed uint64, priced s109Totals) {
	t.Helper()
	if n := priced.warns["rent_funds"]; n < 1 || n > 1+s109Ticks/10 {
		t.Errorf("seed %d: %d Warns \"tenure refusal\" for rent_funds in %d ticks, want 1 to %d (the first and every tenth)", seed, n, s109Ticks, 1+s109Ticks/10)
	}
}

// s109Exemption checks the exempt run of one seed against the closed one.
func s109Exemption(t *testing.T, seed uint64, closed, exempt s109Totals) {
	t.Helper()
	if exempt.blocked != 0 || exempt.started == 0 || exempt.paid != 0 || exempt.exempt == 0 {
		t.Errorf("seed %d: exempt %+v, want nothing blocked, starts, rent-free units and no rent paid", seed, exempt)
	}
	if closed.blocked <= exempt.blocked {
		t.Errorf("seed %d: blocked %d with closure and %d with the exemption, want more with closure", seed, closed.blocked, exempt.blocked)
	}
}
