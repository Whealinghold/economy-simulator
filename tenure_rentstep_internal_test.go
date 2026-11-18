package economysim

import (
	"log/slog"
	"reflect"
	"testing"
)

// The tests of P8.3a, the rent step and RentPaid (spec-land.md 5.5, 13.1, 13.2). They run
// the world of rationConfig (ten households of weight 1, each asking one execution of 3 units
// of "ore" in home, the pool of home holding the title, rent 2) up to the starts with toStart
// and then call the position itself, so each rule is read on the counts of one tick.

// stepConfig is rationConfig with the step of the rent on: 5 000 basis points of the rent
// of 2 is a step of 1, with the default floor 0 and ceiling 20 000 (a rent of 4).
func stepConfig(stock, poolBP int64, closed bool) Config {
	c := rationConfig(stock, poolBP, closed)
	c.Modules.Tenure.RentStepBP = 5_000
	return c
}

// TestStep_RisesFallsHolds verifies the rent step of FR-LND-019 on the counts of a world
// tick: the posted rent rises by one step while units are unserved and none was cut for rent,
// falls while let units lie idle, holds on a closed endowment, and a rent scale changes the
// rent in force (and so the rent the event reports) and not the posted rent the step moves.
//
// Arithmetic: base rent 2, step max(1, 2 * 5 000 / 10 000) = 1, ceiling 2 * 20 000 / 10 000 =
// 4. Stock 10, pool 5 000: the commons has 5 units and the pool 5 let units; ten households
// want 3 units each, 30: the commons serves 3, the let tier 3, 24 are unserved, so X > 0 and
// C = 0 and the rent rises to 3. Stock 100: the commons has 50 units and serves all ten
// households (30 units), the pool's 50 units lie idle and nothing is unserved, so the rent
// falls to 1. Closed: it holds at 2. A scale of 5 000 gives a rent in force of 1 with the
// posted rent 2, and the rise takes the posted rent to 3.
//
// Requirement: FR-LND-019
func TestStep_RisesFallsHolds(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name                         string
		stock                        int64
		closed                       bool
		scaleBP                      int64
		wantRent, wantAfter          int64
		wantUnserved, wantIdleOrZero int64
	}{
		{"rises while demand is unserved", 10, false, 10_000, 2, 3, 24, 2},
		{"falls while let units lie idle", 100, false, 10_000, 2, 1, 0, 50},
		{"holds on a closed endowment", 10, true, 10_000, 2, 2, 0, 0},
		{"the scale moves the rent in force, not the posted rent", 10, false, 5_000, 1, 3, 24, 2},
	} {
		t.Run(tt.name, func(t *testing.T) {
			obs := &evLog{}
			w := newMill(t, stepConfig(tt.stock, 5_000, tt.closed), WithSeed(1), WithAudit(), WithObserver(obs))
			if err := w.SetRentScales([]EndowmentScale{{Region: 0, Resource: 0, BP: tt.scaleBP}}); err != nil {
				t.Fatalf("SetRentScales(%d) error = %v, want nil", tt.scaleBP, err)
			}
			toStart(t, w)
			w.p83aRent()
			if got := w.tenure.posted[0]; got != tt.wantAfter {
				t.Errorf("posted rent after the step = %d, want %d", got, tt.wantAfter)
			}
			evs := of[RentPaid](obs)
			if len(evs) != 1 {
				t.Fatalf("RentPaid events = %d (%+v), want 1", len(evs), evs)
			}
			e := evs[0]
			if e.Rent != Money(tt.wantRent) || e.RentAfter != Money(tt.wantAfter) || e.Unserved != tt.wantUnserved {
				t.Errorf("RentPaid rent in force %d, after %d, unserved %d, want %d, %d and %d", e.Rent, e.RentAfter, e.Unserved, tt.wantRent, tt.wantAfter, tt.wantUnserved)
			}
			if !tt.closed && tt.stock == 100 && e.Idle != tt.wantIdleOrZero {
				t.Errorf("RentPaid idle = %d, want %d (the pool's 50 let units, none drawn)", e.Idle, tt.wantIdleOrZero)
			}
		})
	}
}

// TestRentPaid_HoldsNoMapAndNoPointer verifies FR-LND-044 and observability-and-logging.md
// 4.1: no field of RentPaid is a map or a pointer, so the golden rendering and every consumer
// see the same order and no live state.
//
// Guard: an event with a map would give a random order to the golden rendering.
//
// Requirement: FR-LND-044
func TestRentPaid_HoldsNoMapAndNoPointer(t *testing.T) {
	t.Parallel()
	for _, f := range reflect.VisibleFields(reflect.TypeFor[RentPaid]()) {
		if f.Type.Kind() == reflect.Map || f.Type.Kind() == reflect.Pointer {
			t.Errorf("RentPaid.%s has kind %v, want no map and no pointer", f.Name, f.Type.Kind())
		}
	}
}

// TestRentPaid_OnlyWhenNonZero_AscendingOrder verifies FR-LND-028: RentPaid is emitted for
// an endowment with a title only in a tick in which a count or the posted rent is non-zero
// or moved, and the events of two endowments come in ascending (RegionID, ResourceID) order.
//
// Guard: an event per quiet endowment would flood a run with rows of zeros.
//
// Requirement: FR-LND-028
func TestRentPaid_OnlyWhenNonZero_AscendingOrder(t *testing.T) {
	t.Parallel()
	obs := &evLog{}
	w := tenureWorld(t, nil, WithObserver(obs))
	toSnapshot(t, w)
	w.p83Regen()
	n := len(obs.events)
	w.p83aRent()
	if got := len(obs.events) - n; got != 0 {
		t.Errorf("P8.3a emitted %d events in a tick with every count 0 and no step, want none", got)
	}
	for i := range w.tenure.rt.counts {
		w.tenure.rt.counts[i].noTitle = int64(i + 1)
	}
	w.p83aRent()
	got := of[RentPaid](obs)
	if len(got) != 2 || got[0].Region != 0 || got[1].Region != 1 || got[0].NoTitle != 1 || got[1].NoTitle != 2 {
		t.Errorf("RentPaid events = %+v, want region 0 then region 1 with 1 and 2 refused for no title", got)
	}
}

// TestRentPaid_AfterEndowmentChanged verifies FR-LND-044 and the order of the events of
// spec-land.md 6.11 item 9: in a step, RentPaid follows the EndowmentChanged of the
// position before it.
//
// Requirement: FR-LND-044
func TestRentPaid_AfterEndowmentChanged(t *testing.T) {
	t.Parallel()
	obs := &evLog{}
	w := newMill(t, rationConfig(10, 5_000, false), WithSeed(1), WithObserver(obs))
	step(t, w, 1)
	ec, rp := -1, -1
	for i, e := range obs.events {
		switch e.(type) {
		case EndowmentChanged:
			ec = i
		case RentPaid:
			rp = i
		}
	}
	if ec < 0 || rp < ec {
		t.Errorf("EndowmentChanged at %d and RentPaid at %d, want RentPaid after EndowmentChanged", ec, rp)
	}
}

// TestRentPaid_IdentityOwnCommonsLetEqualsDrawn verifies FR-LND-028 and the identity of
// spec-land.md 13.1.1: in a tick in which RentPaid and EndowmentChanged are emitted for the
// endowment, the units drawn in the three tiers add to the units drawn from the stock.
//
// Arithmetic: stock 10 (commons 5, pool 5), ten households wanting 3 units: the commons
// serves one execution (3 units) and the let tier one (3 units): 0 + 3 + 3 = 6 drawn.
//
// Requirement: FR-LND-028
func TestRentPaid_IdentityOwnCommonsLetEqualsDrawn(t *testing.T) {
	t.Parallel()
	obs := &evLog{}
	w := newMill(t, rationConfig(10, 5_000, false), WithSeed(1), WithAudit(), WithObserver(obs))
	step(t, w, 3)
	rents := of[RentPaid](obs)
	if len(rents) == 0 {
		t.Fatalf("no RentPaid in three ticks, want one a tick")
	}
	drawn := map[int64]int64{}
	for _, e := range of[EndowmentChanged](obs) {
		drawn[e.Tick] = e.Drawn
	}
	var positive int
	for _, e := range rents {
		d, changed := drawn[e.Tick]
		if got := e.OwnUnits + e.CommonsUnits + e.LetUnits; changed && got != d {
			t.Errorf("tick %d: own %d + commons %d + let %d = %d, want the %d units drawn", e.Tick, e.OwnUnits, e.CommonsUnits, e.LetUnits, got, d)
		} else if got > 0 {
			positive++
		}
	}
	if positive == 0 {
		t.Errorf("no tick with units drawn in three, want the first (6 units) to be one")
	}
}

// TestRentPaid_RebuildsHouseholdAndPoolMoney verifies FR-LND-028 and FR-OBS-041 for rent: the
// rows From and To and the amount ToPool of RentPaid give the change of the money of the
// household sector and of the pool, so an observer with no agent table rebuilds both.
//
// Arithmetic: the pool of home is the only lessor, so the let units the starts took, 3, at a
// rent of 2 move 6 from households of home to the pool: From = [{home, 6}], To = nil, ToPool =
// 6, Paid = 6.
//
// Requirement: FR-LND-028
func TestRentPaid_RebuildsHouseholdAndPoolMoney(t *testing.T) {
	t.Parallel()
	obs := &evLog{}
	w := newMill(t, rationConfig(10, 5_000, false), WithSeed(1), WithAudit(), WithObserver(obs))
	householdsBefore, poolBefore := tnTotalMoney(t, w)-tnMoney(t, w, 0), tnMoney(t, w, 0)
	toStart(t, w)
	w.p83aRent()
	evs := of[RentPaid](obs)
	if len(evs) != 1 {
		t.Fatalf("RentPaid events = %d, want 1", len(evs))
	}
	e := evs[0]
	var from, to Money
	for _, r := range e.From {
		from += r.Money
	}
	for _, r := range e.To {
		to += r.Money
	}
	if from != e.Paid || to+e.ToPool != e.Paid || e.Paid != 6 {
		t.Errorf("From sums to %d, To to %d and ToPool %d, Paid %d, want From = To + ToPool = Paid = 6", from, to, e.ToPool, e.Paid)
	}
	if got, want := tnTotalMoney(t, w)-tnMoney(t, w, 0), householdsBefore-int64(from)+int64(to); got != want {
		t.Errorf("household money = %d, want %d (before, less From, plus To)", got, want)
	}
	if got, want := tnMoney(t, w, 0), poolBefore+int64(e.ToPool); got != want {
		t.Errorf("pool money = %d, want %d (before plus ToPool)", got, want)
	}
}

// TestMoneyRows_MergedByRegionAscending verifies the rows of RentPaid on planted transfers of
// two endowments: the transfers of the endowment are merged into one row for each region,
// in ascending RegionID, the pool's part is ToPool and not a row of To, and the transfers
// of the next endowment are left for its own event.
//
// Requirement: FR-LND-028
func TestMoneyRows_MergedByRegionAscending(t *testing.T) {
	t.Parallel()
	w := tenureWorld(t, nil)
	rt := &w.tenure.rt
	rt.flows = []rentFlow{
		{cell: 0, from: 1, to: 0, amount: 5},
		{cell: 0, from: 0, to: 1, amount: 2},
		{cell: 0, from: 1, to: 1, amount: 4},
		{cell: 0, from: 0, to: 0, pool: true, amount: 3},
		{cell: 1, from: 1, to: 1, amount: 9},
	}
	var e RentPaid
	next := w.tenure.moneyRows(&e, 0, 0)
	if next != 4 {
		t.Errorf("moneyRows returned %d, want 4 (the first transfer of the next endowment)", next)
	}
	tnEq(t, "From", e.From, []RegionMoney{{Region: 0, Money: 5}, {Region: 1, Money: 9}})
	tnEq(t, "To", e.To, []RegionMoney{{Region: 0, Money: 5}, {Region: 1, Money: 6}})
	if e.ToPool != 3 {
		t.Errorf("ToPool = %d, want 3", e.ToPool)
	}
}

// refusalLines is the "tenure refusal" lines of a capture, in order.
func refusalLines(c capture) (out []logRec) {
	for _, l := range *c.lines {
		if l.msg == "tenure refusal" {
			out = append(out, l)
		}
	}
	return out
}

// TestWarn_TenureRefusalFirstAndEveryTenth verifies FR-LND-030 and the counterpart rule of
// spec-land.md 13.5: a Warn "tenure refusal" is written for an endowment and a reason at the
// first tick with requests refused and then at every tenth of the run, never in a tick with
// none, the run restarts after a quiet tick, and the line carries the keys of 13.2.
//
// Arithmetic: the ticks 1 to 21 of a run have refusals; the lines are at the runs 1, 10 and
// 20, three. After one quiet tick the run is 0 and the next refusal is run 1 again, for each
// of the two reasons: two lines more.
//
// Requirement: FR-LND-030
func TestWarn_TenureRefusalFirstAndEveryTenth(t *testing.T) {
	t.Parallel()
	c := newCapture()
	w := newMill(t, rationConfig(10, 5_000, false), WithSeed(1), WithLogger(slog.New(c)))
	toStart(t, w)
	cnt := &w.tenure.rt.counts[0]
	for range 21 {
		cnt.noTitle, cnt.rentFunds = 2, 0
		w.p83aRent()
	}
	lines := refusalLines(c)
	if len(lines) != 3 {
		t.Fatalf("%d lines after 21 ticks of refusals, want 3 (runs 1, 10 and 20)", len(lines))
	}
	want := logRec{level: slog.LevelWarn, attrs: map[string]int64{"tick": lines[0].attrs["tick"], "region": 0, "resource": 0, "rent": 2, "agents": 2},
		msg: "tenure refusal", strs: map[string]string{"reason": "no_title"}}
	tnEq(t, "first line", lines[0], want)
	cnt.noTitle = 0
	w.p83aRent()
	cnt.noTitle, cnt.rentFunds = 1, 1
	w.p83aRent()
	if got := len(refusalLines(c)); got != 5 {
		t.Errorf("%d lines after the quiet tick and one refusal for each reason, want 5 (the run of each restarts at 1)", got)
	}
}

// TestSnapshot_TenureRunsAreTheSavedRuns verifies FR-LND-032: the two runs of the snapshot
// row of an endowment are the consecutive ticks with a refusal for each reason, the state
// the checkpoint saves.
//
// Requirement: FR-LND-032
func TestSnapshot_TenureRunsAreTheSavedRuns(t *testing.T) {
	t.Parallel()
	w := newMill(t, rationConfig(10, 5_000, false), WithSeed(1))
	toStart(t, w)
	cnt := &w.tenure.rt.counts[0]
	for range 7 {
		cnt.noTitle, cnt.rentFunds = 1, 0
		w.p83aRent()
	}
	cnt.noTitle, cnt.rentFunds = 0, 1
	w.p83aRent()
	var row EndowmentRow
	w.tenure.fillRow(&row, 0)
	if row.NoTitleRun != 0 || row.RentFundsRun != 1 {
		t.Errorf("runs = %d and %d, want 0 (a quiet tick ends the run) and 1", row.NoTitleRun, row.RentFundsRun)
	}
}

// TestDebug_RentChargedOncePerEndowmentWithRentPaid verifies the Debug line of
// spec-land.md 13.2: "rent charged" is written for an endowment that paid rent, with the
// rent in force and the units let, and for none that did not.
//
// Requirement: FR-LND-030
func TestDebug_RentChargedOncePerEndowmentWithRentPaid(t *testing.T) {
	t.Parallel()
	c := newCapture()
	w := newMill(t, rationConfig(10, 5_000, false), WithSeed(1), WithLogger(slog.New(c)))
	toStart(t, w)
	w.p83aRent()
	var got []logRec
	for _, l := range *c.lines {
		if l.msg == "rent charged" {
			got = append(got, l)
		}
	}
	if len(got) != 1 || got[0].level != slog.LevelDebug || got[0].attrs["rent"] != 2 || got[0].attrs["quantity"] != 3 {
		t.Errorf("rent charged lines = %+v, want one Debug line with rent 2 and quantity 3", got)
	}
}

// TestAudit_RentEqualsRentTimesChargedUnitsLessExemptAndForfeited verifies the third part of
// FR-LND-037: the rent paid and forfeited of a tick are the rent times the units charged,
// not those exempt or let to their own lessor, and the audit fails when they differ.
//
// Arithmetic: the start of rationConfig lets 3 units at 2, so 6 are due and 6 paid. A count
// of 7 paid with 6 due is a defect the audit reports.
//
// Requirement: FR-LND-037
func TestAudit_RentEqualsRentTimesChargedUnitsLessExemptAndForfeited(t *testing.T) {
	t.Parallel()
	w := newMill(t, rationConfig(10, 5_000, false), WithSeed(1), WithAudit())
	toStart(t, w)
	c := &w.tenure.rt.counts[0]
	if c.due != 6 || c.paid != 6 {
		t.Fatalf("due %d and paid %d after the start, want 6 and 6", c.due, c.paid)
	}
	if err := w.auditTenure(); err != nil {
		t.Errorf("auditTenure() error = %v, want nil with paid equal to due", err)
	}
	c.paid++
	if err := w.auditTenure(); err == nil {
		t.Errorf("auditTenure() error = nil, want an error with 7 paid and 6 due")
	}
	c.paid, c.forfeited = c.paid-1-2, 2
	if err := w.auditTenure(); err != nil {
		t.Errorf("auditTenure() error = %v, want nil with 4 paid and 2 forfeited of 6 due", err)
	}
}

// TestNonInterference_Tenure verifies FR-LND-043: a run of a world that charges rent and
// moves the posted rent, with an observer and a logger that reports every level, ends in the
// snapshot a run with neither ends in, with the same titles, rent in force and stock.
//
// Requirement: FR-LND-043
func TestNonInterference_Tenure(t *testing.T) {
	t.Parallel()
	run := func(opts ...Option) *World {
		w := newMill(t, stepConfig(10, 5_000, false), append([]Option{WithSeed(1), WithAudit()}, opts...)...)
		step(t, w, 30)
		return w
	}
	plain := run()
	seen := run(WithObserver(&evLog{}), WithLogger(slog.New(newCapture())))
	a, errA := plain.Snapshot()
	b, errB := seen.Snapshot()
	if errA != nil || errB != nil {
		t.Fatalf("Snapshot errors = %v and %v, want nil", errA, errB)
	}
	tnEq(t, "snapshot with an observer and a logger", b, a)
	tnEq(t, "holders with an observer and a logger", tnHolders(t, seen, 0), tnHolders(t, plain, 0))
	if got, want := seen.tenure.posted[0], plain.tenure.posted[0]; got != want || want == 2 {
		t.Errorf("posted rent with an observer and a logger = %d, without = %d, want equal and moved from the configured 2", got, want)
	}
}

// TestRentIsOffDual_Boundaries verifies the rule of 5.6.2 on its edges: a slack row is off
// only above a rent of 0, a binding row is off when the gap is above half the dual (exactly
// half is not), a rent of 0 on a binding row is off by 100%, and a product past int64 counts
// as off and never panics.
//
// Requirement: FR-LND-040
func TestRentIsOffDual_Boundaries(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name        string
		rent, dual  int64
		wantOffDual bool
	}{
		{"slack and free", 0, 0, false},
		{"slack and charged", 1, 0, true},
		{"exactly half above", 15, 10, false},
		{"just over half above", 16, 10, true},
		{"exactly half below", 5, 10, false},
		{"just over half below", 4, 10, true},
		{"free on a binding row", 0, 10, true},
		{"the product overflows", 1 << 62, 1, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := rentIsOffDual(tt.rent, tt.dual); got != tt.wantOffDual {
				t.Errorf("rentIsOffDual(%d, %d) = %v, want %v", tt.rent, tt.dual, got, tt.wantOffDual)
			}
		})
	}
}
