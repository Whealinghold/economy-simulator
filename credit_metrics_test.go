package economysim_test

import (
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// creditFold is the test-side folder of the credit instruments of spec-credit.md 16.3:
// each field is what the adapter would keep for the instrument of the same name, fed by
// the events alone (and, for outstanding, seeded from a snapshot).
type creditFold struct {
	formed, principal, interest, outstanding, defaults int64
	lost                                               map[economysim.Cause]int64
	refused, limited                                   map[string]int64
	due                                                map[economysim.ClaimID]int64
}

func newCreditFold() *creditFold {
	return &creditFold{lost: map[economysim.Cause]int64{}, refused: map[string]int64{}, limited: map[string]int64{}, due: map[economysim.ClaimID]int64{}}
}

// feed folds one event. Every instrument counts claims of origin loan only.
func (f *creditFold) feed(e economysim.Event) {
	switch v := e.(type) {
	case economysim.LoanFormed:
		f.formed++
		f.principal += int64(v.Principal)
		f.outstanding += int64(v.Principal)
		f.due[v.Claim] = v.Due
	case economysim.CreditSettled:
		f.interest += int64(v.InterestPaid)
		f.outstanding -= int64(v.PrincipalPaid)
		f.defaults += v.Defaulted
	case economysim.LoanEnded:
		if v.Origin != economysim.OriginLoan {
			return
		}
		f.interest += int64(v.PaidInterest)
		f.outstanding -= int64(v.PaidPrincipal + v.LostPrincipal)
		f.lost[v.Cause] += int64(v.LostPrincipal + v.LostInterest)
	case economysim.CreditRound:
		f.refused["requests"] += v.RequestsRefused
		f.refused["offers"] += v.OffersRefused
		f.limited["in_arrears"] += v.InArrears
		f.limited["exposure"] += v.Exposure
		f.limited["too_many_claims"] += v.TooMany
		f.limited["overflow"] += v.Overflow
	}
}

// seed sets the level instrument from the claim rows of a snapshot: the sum of
// PrincipalOwed of the loans, what the adapter does at WorldRestored.
func (f *creditFold) seed(s economysim.Snapshot) {
	f.outstanding = loanOutstanding(s)
}

// loanOutstanding is the principal owed over the claims of origin loan of a snapshot.
func loanOutstanding(s economysim.Snapshot) (sum int64) {
	for _, c := range s.Claims {
		if c.Origin == economysim.OriginLoan {
			sum += int64(c.PrincipalOwed)
		}
	}
	return sum
}

// TestCredit_MetricsDerivable verifies FR-CRD-077: each instrument of spec-credit.md
// 16.3 is derivable from the events and the snapshot rows alone. A folder that sees the
// events only recomputes, after every step of a run of the pack one with loans, the
// claims formed, the principal and the interest paid, the principal outstanding, the
// amount lost by cause, and the claims newly overdue, and each equals what the
// snapshot says: the count of claims formed is the count of claim IDs issued, the
// principal formed is Lent, the interest paid is InterestPaid, the outstanding is the
// sum of PrincipalOwed over the loan rows, the losses are the totals LostPrincipal and
// LostInterest, and the defaults of a tick are the claims whose Overdue is that tick
// (open in the snapshot or removed in the same tick with that due tick). The arrears
// gauge is the sum of the owed over the overdue loan rows by region. The counters of the
// round have the subsets of 16.1: the arrears and the exposure refusals are inside the
// refused requests.
//
// Setup: pack one with creditLiveDoc, seed 1, 135 ticks (loans from tick 59, the first
// repayments at 66, the removal waterfalls from 115 and the main due tick 123).
//
// Requirement: FR-CRD-077
func TestCredit_MetricsDerivable(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := loadPackBytes(t, []byte(creditLiveDoc), packOne...)
	rec := &recorder{}
	w := newPackWorld(t, cfg, 1, economysim.WithObserver(rec), economysim.WithAudit())
	fold, seen := newCreditFold(), 0
	for i := range 135 {
		if err := w.Step(); err != nil {
			t.Fatalf("Step() at tick %d error = %v, want nil", i, err)
		}
		tickEvents := rec.events[seen:]
		seen = len(rec.events)
		for _, e := range tickEvents {
			fold.feed(e)
		}
		snap, err := w.Snapshot()
		if err != nil {
			t.Fatalf("Snapshot() at tick %d error = %v, want nil", i, err)
		}
		wantTotals(t, int64(i), fold, snap)
		wantDefaults(t, int64(i), fold, tickEvents, snap)
	}
	if fold.formed == 0 || fold.lost[economysim.CauseDebtorRemoved] == 0 {
		t.Errorf("the run formed %d loans and lost %d to removed debtors, want loans and a waterfall so the derivation is exercised", fold.formed, fold.lost[economysim.CauseDebtorRemoved])
	}
	if fold.limited["in_arrears"]+fold.limited["exposure"] > fold.refused["requests"] {
		t.Errorf("arrears %d and exposure %d refusals exceed the requests refused, %d: they are subsets", fold.limited["in_arrears"], fold.limited["exposure"], fold.refused["requests"])
	}
}

// wantTotals fails the test unless the folded instruments equal the totals and the rows
// of the snapshot after the step of tick.
func wantTotals(t *testing.T, tick int64, f *creditFold, s economysim.Snapshot) {
	t.Helper()
	c := s.Credit
	if f.principal != int64(c.Lent) || f.interest != int64(c.InterestPaid) {
		t.Errorf("tick %d: principal formed %d and interest paid %d, want the totals Lent %d and InterestPaid %d", tick, f.principal, f.interest, c.Lent, c.InterestPaid)
	}
	if got, want := f.outstanding, loanOutstanding(s); got != want {
		t.Errorf("tick %d: outstanding by events %d, want the sum of PrincipalOwed of the loan rows %d", tick, got, want)
	}
	var lost int64
	for _, v := range f.lost {
		lost += v
	}
	if want := int64(c.LostPrincipal + c.LostInterest); lost != want {
		t.Errorf("tick %d: lost by events %d, want the totals %d", tick, lost, want)
	}
	if f.formed != int64(len(f.due)) {
		t.Errorf("tick %d: claims formed %d, want %d claim IDs seen", tick, f.formed, len(f.due))
	}
}

// wantDefaults fails the test unless the claims newly overdue that tick, as the events
// count them, are those the snapshot rows and the removals of the tick name: a claim
// is newly overdue at its due tick when it was not paid, so it is an open row with
// Overdue equal to the tick, or a claim removed in the same tick whose due tick it is.
func wantDefaults(t *testing.T, tick int64, f *creditFold, events []economysim.Event, s economysim.Snapshot) {
	t.Helper()
	var folded, want int64
	for _, e := range events {
		switch v := e.(type) {
		case economysim.CreditSettled:
			folded += v.Defaulted
		case economysim.LoanEnded:
			if v.Origin == economysim.OriginLoan && v.Cause == economysim.CauseDebtorRemoved && f.due[v.Claim] == tick {
				want++
			}
		}
	}
	for _, c := range s.Claims {
		if c.Origin == economysim.OriginLoan && c.Overdue == tick {
			want++
		}
	}
	if folded != want {
		t.Errorf("tick %d: claims newly overdue by events %d, want %d from the rows and the removals", tick, folded, want)
	}
}

// TestCredit_MetricsSeededAtARestore verifies the seed of economysim.credit.outstanding
// (spec-credit.md 16.3.1): an adapter built after a restore starts the level at the sum
// of PrincipalOwed over the loan rows of the snapshot WorldRestored carries, adds the
// events after it, and agrees with the snapshot of every later tick, through the main
// due tick where claims are repaid, removed and written off.
//
// Setup: checkpoint at tick 100 (claims open, the main due tick 123 ahead), 40 ticks on.
//
// Requirement: FR-CRD-077
func TestCredit_MetricsSeededAtARestore(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := loadPackBytes(t, []byte(creditLiveDoc), packOne...)
	orig := newPackWorld(t, cfg, 1)
	step(t, orig, 100)
	rec := &recorder{}
	restored, err := economysim.RestoreWorld(cfg, mustCheckpoint(t, orig), economysim.WithObserver(rec), economysim.WithAudit())
	if err != nil {
		t.Fatalf("RestoreWorld error = %v, want nil", err)
	}
	first, ok := rec.events[0].(economysim.WorldRestored)
	if !ok {
		t.Fatalf("first event of the restored world = %T, want WorldRestored", rec.events[0])
	}
	fold := newCreditFold()
	fold.seed(first.Snapshot)
	if fold.outstanding == 0 {
		t.Fatal("the restored snapshot holds no loan principal, want open loans at tick 100 so the seed is not 0")
	}
	seen := 1
	for i := range 40 {
		if err := restored.Step(); err != nil {
			t.Fatalf("Step() of the restored world at tick %d error = %v, want nil", 100+i, err)
		}
		for _, e := range rec.events[seen:] {
			fold.feed(e)
		}
		seen = len(rec.events)
		snap, err := restored.Snapshot()
		if err != nil {
			t.Fatalf("Snapshot() at tick %d error = %v, want nil", 100+i, err)
		}
		if got, want := fold.outstanding, loanOutstanding(snap); got != want {
			t.Errorf("tick %d: outstanding seeded at the restore and folded %d, want the sum of the loan rows %d", 100+i, got, want)
		}
	}
}

// TestCredit_ArrearsGaugeFromTheSnapshot verifies that the arrears gauge of
// spec-credit.md 16.3 is the owed on the overdue loan rows summed by region, and that
// at the end of a settlement pass it equals the Arrears of that tick's CreditSettled of
// the region when nothing else ended a claim in the tick.
//
// Requirement: FR-CRD-077
func TestCredit_ArrearsGaugeFromTheSnapshot(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := loadPackBytes(t, []byte(creditLiveDoc), packOne...)
	rec := &recorder{}
	w := newPackWorld(t, cfg, 1, economysim.WithObserver(rec))
	seen, compared := 0, 0
	for i := range 135 {
		step(t, w, 1)
		events := rec.events[seen:]
		seen = len(rec.events)
		snap, err := w.Snapshot()
		if err != nil {
			t.Fatalf("Snapshot() at tick %d error = %v, want nil", i, err)
		}
		if endedByRemoval(events) {
			continue
		}
		gauge := arrearsByRegion(snap)
		for _, e := range events {
			if s, ok := e.(economysim.CreditSettled); ok {
				compared++
				if int64(s.Arrears) != gauge[s.Region] {
					t.Errorf("tick %d region %d: Arrears of CreditSettled %d, want the gauge from the rows %d", i, s.Region, s.Arrears, gauge[s.Region])
				}
			}
		}
	}
	if compared == 0 {
		t.Error("no CreditSettled compared with the gauge, want the pass to report in a tick with no removal")
	}
}

// endedByRemoval reports whether a loan ended with its debtor removed in the events.
func endedByRemoval(events []economysim.Event) bool {
	for _, e := range events {
		if v, ok := e.(economysim.LoanEnded); ok && v.Cause == economysim.CauseDebtorRemoved {
			return true
		}
	}
	return false
}

// arrearsByRegion is the owed on the overdue loan rows of a snapshot, by debtor region.
func arrearsByRegion(s economysim.Snapshot) map[economysim.RegionID]int64 {
	out := map[economysim.RegionID]int64{}
	for _, c := range s.Claims {
		if c.Origin == economysim.OriginLoan && c.Overdue >= 0 {
			out[c.Region] += int64(c.PrincipalOwed + c.InterestOwed)
		}
	}
	return out
}
