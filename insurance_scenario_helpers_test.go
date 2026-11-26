package economysim_test

import (
	"fmt"
	"reflect"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// The helpers of scenarios S160 to S165 (INS-S1 to INS-S6, spec-insurance.md 17): a step loop that
// tallies the cover events of each tick and checks the books against the snapshot, and the
// comparison of a module that is on with no block against one that is off. The scenarios keep
// counters and the events, not a snapshot per seed (plan 3-ins, gate economics, the S113 lesson).

// insTrack steps a world and tallies what its events show of the module, cumulatively.
type insTrack struct {
	w    *economysim.World
	rec  *recorder
	seen int
	tl   insTally

	premiums, due, paid, short, dropped, cut int64 // sums over the policies formed and ended
	claimPaid, claimEnded                    int64 // payout claims paid at P1.7a and ended repaid
	offersRefused, exposure, noOffer         int64 // what the rounds refused
	unfilled, funds, valueCapped             int64
	roundCapped, sum                         int64 // units valued at the cap and the sum insured, from the rounds
	removed, landed, lostUnits               int64 // agents removed, units delivered and lost by arrivals
	bad                                      string
}

// newInsTrack builds a world of cfg with the recorder and the audit on.
func newInsTrack(t testing.TB, cfg economysim.Config, seed uint64, opts []economysim.Option) *insTrack {
	t.Helper()
	rec := &recorder{}
	w := newPackWorld(t, cfg, seed, append(slices.Clone(opts), economysim.WithAudit(), economysim.WithObserver(rec))...)
	return &insTrack{w: w, rec: rec, seen: len(rec.events)}
}

// add tallies the events of a step.
func (r *insTrack) add(events []economysim.Event) {
	r.tl.add(events)
	for _, e := range events {
		switch e := e.(type) {
		case economysim.CoverFormed:
			r.premiums += int64(e.Premium)
		case economysim.CoverEnded:
			r.addEnded(e)
		case economysim.CoverSettled:
			r.claimPaid, r.claimEnded = r.claimPaid+int64(e.ClaimPaid), r.claimEnded+e.ClaimEnded
			r.valueCapped += e.ValueCapped
		case economysim.CoverRound:
			r.addRound(e)
		case economysim.AgentBankrupted:
			r.removed++
		case economysim.ShipmentArrived:
			r.addArrival(e)
		}
	}
}

// addEnded tallies a policy ended and checks that its due is the sum of its parts.
func (r *insTrack) addEnded(e economysim.CoverEnded) {
	r.due, r.paid, r.short = r.due+int64(e.Due), r.paid+int64(e.Paid), r.short+int64(e.Short)
	r.dropped, r.cut = r.dropped+int64(e.Dropped), r.cut+int64(e.Cut)
	if e.Due != e.Paid+e.Short+e.Dropped+e.Cut && r.bad == "" {
		r.bad = fmt.Sprintf("tick %d policy %d: due %d, paid %d + short %d + dropped %d + cut %d", e.Tick, e.Policy, e.Due, e.Paid, e.Short, e.Dropped, e.Cut)
	}
}

// addRound tallies what a round refused and the sums it covered.
func (r *insTrack) addRound(e economysim.CoverRound) {
	r.offersRefused, r.exposure, r.noOffer = r.offersRefused+e.OffersRefused, r.exposure+e.Exposure, r.noOffer+e.NoOffer
	r.unfilled, r.funds = r.unfilled+int64(e.Unfilled), r.funds+e.Funds
	r.roundCapped, r.sum = r.roundCapped+e.ValueCapped, r.sum+int64(e.Sum)
}

// addArrival tallies the units an arrival delivered and lost.
func (r *insTrack) addArrival(e economysim.ShipmentArrived) {
	for _, l := range e.Lines {
		r.landed += int64(l.Delivered)
		r.lostUnits += int64(l.Lost)
	}
}

// step runs one step and tallies its events, and returns the first of the checks that fails: a
// policy whose due is not the sum of its parts (spec-insurance.md 9.4 item 1), or a total of the
// snapshot that is not the sum of the events so far.
func (r *insTrack) step(t testing.TB) string {
	t.Helper()
	if err := r.w.Step(); err != nil {
		t.Fatalf("Step() at tick %d error = %v, want nil (the audit ran)", r.w.Tick(), err)
	}
	r.add(r.rec.events[r.seen:])
	r.seen = len(r.rec.events)
	snap, err := r.w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot() at tick %d error = %v, want nil", r.w.Tick(), err)
	}
	got := snap.Insurance
	want := economysim.InsuranceTotals{Formed: r.tl.formed, Premiums: economysim.Money(r.premiums), Due: economysim.Money(r.due),
		Paid: economysim.Money(r.paid), Short: economysim.Money(r.short), Dropped: economysim.Money(r.dropped), Cut: economysim.Money(r.cut)}
	if got != want && r.bad == "" {
		r.bad = fmt.Sprintf("tick %d: the snapshot totals %+v, the events %+v", snap.Tick, got, want)
	}
	return r.bad
}

// run steps the world for ticks, calling before at the start of each tick when it is set, and
// returns the first failure of step, or "".
func (r *insTrack) run(t testing.TB, ticks int64, before func(w *economysim.World)) string {
	t.Helper()
	for range ticks {
		if before != nil {
			before(r.w)
		}
		if msg := r.step(t); msg != "" {
			return msg
		}
	}
	return ""
}

// insNames names a pack of a golden set for the scenario reports: the file of pack two or of pack one.
func insNames(g insGolden) string {
	if g.name == "two" {
		return "two"
	}
	return "one"
}

// withoutBlocks returns cfg with the cover block of every role removed, and with the module
// removed too when moduleOff is set (on with no block against off, FR-INS-005).
func withoutBlocks(cfg economysim.Config, moduleOff bool) economysim.Config {
	cfg.Roles = slices.Clone(cfg.Roles)
	for i := range cfg.Roles {
		cfg.Roles[i].Cover = nil
	}
	if moduleOff {
		cfg.Modules.Insurance = nil
	}
	return cfg
}

// insNeutralRun runs cfg for ticks and returns its events without WorldCreated and its final snapshot
// without the three parts that name the module.
func insNeutralRun(t testing.TB, cfg economysim.Config, opts []economysim.Option, seed uint64, ticks int64) ([]economysim.Event, economysim.Snapshot) {
	t.Helper()
	res := runPack(t, cfg, runSpec{Seed: seed, Ticks: ticks, Drive: driveStep, Observe: true, Extra: opts})
	events := slices.DeleteFunc(slices.Clone(res.Events), func(e economysim.Event) bool { _, ok := e.(economysim.WorldCreated); return ok })
	snap := res.Final
	snap.Modules, snap.Policies, snap.Insurance, snap.ConfigHash = nil, nil, economysim.InsuranceTotals{}, [32]byte{}
	return events, snap
}

// insNeutral says the world of cfg with the module on and no block equals the one with it off in
// every event but WorldCreated and in the snapshot but the module's own parts, and returns what
// differs when it does not.
func insNeutral(t testing.TB, cfg economysim.Config, opts []economysim.Option, seed uint64, ticks int64) string {
	t.Helper()
	onEv, onSnap := insNeutralRun(t, withoutBlocks(cfg, false), opts, seed, ticks)
	offEv, offSnap := insNeutralRun(t, withoutBlocks(cfg, true), opts, seed, ticks)
	if !reflect.DeepEqual(onEv, offEv) {
		return fmt.Sprintf("the events differ (%d on, %d off)", len(onEv), len(offEv))
	}
	if !reflect.DeepEqual(onSnap, offSnap) {
		return "the final snapshots differ"
	}
	return ""
}

// withUnderwrite returns cfg with fn applied to a copy of the underwriter block of every role that
// has one, so that a variant of a scenario shares no pointer with the pack it came from.
func withUnderwrite(cfg economysim.Config, fn func(u *economysim.UnderwriteTerms)) economysim.Config {
	cfg.Roles = slices.Clone(cfg.Roles)
	for i := range cfg.Roles {
		if c := cfg.Roles[i].Cover; c != nil && c.Underwrite != nil {
			u := *c.Underwrite
			fn(&u)
			cv := *c // keep the Insure block of a role that has both
			cv.Underwrite = &u
			cfg.Roles[i].Cover = &cv
		}
	}
	return cfg
}

// withValueCap returns cfg with the module's value cap set to bp, on a copy of the module.
func withValueCap(cfg economysim.Config, bp int64) economysim.Config {
	m := *cfg.Modules.Insurance
	m.ValueCapBP = bp
	cfg.Modules.Insurance = &m
	return cfg
}
