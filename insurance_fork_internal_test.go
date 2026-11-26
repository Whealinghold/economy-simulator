package economysim

import (
	"bytes"
	"fmt"
	"reflect"
	"testing"
)

// This file is the fork-at-N test of the insurance module (plan 3-ins Phase 7, spec-insurance.md
// 16, FR-INS-087): a world is saved at each boundary of a run that holds open route policies and a
// payout claim, restored, and run to the end beside the uninterrupted world. A restore that drops
// or rebuilds any state of the module differently shows as a first tick at which the two differ.

// forkEnd is the tick at which the run of the fork test ends.
const forkEnd = 9

// forkActs are the calls the script makes at a boundary, before the step of that tick. They are
// the same calls on every world of a comparison, so a world restored at that boundary receives
// them after its restore. The tick numbers are those of insCkWorld.
//
// Tick 2: the underwriter has 1 500, so the arrival at tick 2 pays 500 at once and leaves a
// payout claim of 1 100. Tick 3: a second shipment of 8 grain is covered (policy 1, premium
// 1 600). Tick 4: the underwriter is funded, so the claim is repaid at P1.7a of tick 4 and the
// second policy settles at its arrival at tick 5.
var forkActs = map[int64]func(t *testing.T, w *World){
	2: func(t *testing.T, w *World) { moneyOf(t, w, insBacker, 1_500) },
	3: func(t *testing.T, w *World) {
		moneyOf(t, w, insBacker, 10_000)
		insDispatch(t, w, insHome2, 8)
		planCover(w, insHome2, coachLane)
		planOffer(w, insBacker, bigOffer)
		w.p33cCoverRound()
	},
	4: func(t *testing.T, w *World) { moneyOf(t, w, insBacker, 5_000) },
}

// forkRun is a world of the fork test with its recorder and the index of its first event of the
// comparison.
type forkRun struct {
	w     *World
	rec   *evLog
	since int
}

// advance applies the calls of the boundary to the world and steps it once.
func (f *forkRun) advance(t *testing.T) {
	t.Helper()
	if act := forkActs[f.w.Tick()]; act != nil {
		act(t, f.w)
	}
	step(t, f.w, 1)
}

// forkEvents returns the events of the run from the comparison on, without the WorldRestored
// that opens the log of a restored world.
func (f *forkRun) forkEvents() []Event {
	ev := f.rec.events[f.since:]
	if len(ev) > 0 {
		if _, ok := ev[0].(WorldRestored); ok {
			ev = ev[1:]
		}
	}
	return ev
}

// TestFork_InsuranceRestoreEqualsUninterrupted verifies that a world saved at any boundary of a
// run with open route policies and a payout claim, restored, and stepped to the end gives, after
// every step, the events, the snapshot, the checkpoint bytes and the audit of the world that was
// never saved. The boundaries (tick 0 is not one: the construction phase has not ended) are before the first policy settles (policies 0 and 1 open and
// then policy 0 alone), at the arrival with a shortfall (the claim formed), with the claim open,
// at its repayment and after the last policy ended.
//
// Requirement: FR-INS-080, FR-INS-087, FR-INS-006
//
// Guard: the unit tests compare the restored state field by field; this one compares what the
// restored world does next. A field the codec of tag 29 loses, or an index it rebuilds in
// another order, shows here as a difference in the events or the bytes some ticks later.
func TestFork_InsuranceRestoreEqualsUninterrupted(t *testing.T) {
	t.Parallel()
	for n := int64(1); n < forkEnd; n++ {
		t.Run(fmt.Sprintf("saved at tick %d", n), func(t *testing.T) {
			t.Parallel()
			a := &forkRun{}
			a.w, a.rec = settleWorld(t, insCkConfig(), 8)
			for a.w.Tick() < n {
				a.advance(t)
			}
			a.since = len(a.rec.events)
			ck, err := a.w.Checkpoint()
			if err != nil {
				t.Fatalf("Checkpoint at tick %d error = %v, want nil", n, err)
			}
			b := &forkRun{rec: &evLog{}}
			b.w, err = RestoreWorld(insCkConfig(), ck, WithStrategy("script", &insScript{}), WithAudit(), WithObserver(b.rec))
			if err != nil {
				t.Fatalf("RestoreWorld at tick %d error = %v, want nil", n, err)
			}
			for a.w.Tick() < forkEnd {
				a.advance(t)
				b.advance(t)
				forkRequireSame(t, n, a, b)
			}
			if got := len(of[CoverEnded](a.rec)); got != 2 {
				t.Errorf("saved at tick %d: the run ended %d policies, want 2 (a run that settles none proves nothing)", n, got)
			}
		})
	}
}

// forkRequireSame fails the test unless the two worlds are at the same tick with the same
// events since the save, the same snapshot, the same checkpoint bytes and a passing audit.
func forkRequireSame(t *testing.T, saved int64, a, b *forkRun) {
	t.Helper()
	tick := a.w.Tick()
	if b.w.Tick() != tick {
		t.Fatalf("saved at tick %d: restored world at tick %d, want %d", saved, b.w.Tick(), tick)
	}
	if !reflect.DeepEqual(a.forkEvents(), b.forkEvents()) {
		t.Fatalf("saved at tick %d, tick %d: the events of the restored world differ (%d and %d events)", saved, tick, len(b.forkEvents()), len(a.forkEvents()))
	}
	if sa, sb := errVal(a.w.Snapshot()), errVal(b.w.Snapshot()); !reflect.DeepEqual(sa, sb) {
		t.Fatalf("saved at tick %d, tick %d: policies %+v totals %+v, want %+v %+v", saved, tick, sb.Policies, sb.Insurance, sa.Policies, sa.Insurance)
	}
	ca, cb := errVal(a.w.Checkpoint()), errVal(b.w.Checkpoint())
	if !bytes.Equal(errVal(ca.MarshalBinary()), errVal(cb.MarshalBinary())) {
		t.Fatalf("saved at tick %d, tick %d: the restored world writes other checkpoint bytes", saved, tick)
	}
	if err := b.w.Audit(); err != nil {
		t.Fatalf("saved at tick %d, tick %d: Audit of the restored world error = %v, want nil", saved, tick, err)
	}
}
