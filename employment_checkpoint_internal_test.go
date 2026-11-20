package economysim

import (
	"bytes"
	"errors"
	"reflect"
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/codec"
)

// hireRestore restores the checkpoint of w into a world of the same configuration
// and the same scripted strategy, with the audit on.
func hireRestore(t *testing.T, w *World, plan func(v *AgentView) JobRequest) *World {
	t.Helper()
	ck, err := w.Checkpoint()
	if err != nil {
		t.Fatalf("Checkpoint() = %v, want nil", err)
	}
	r, err := RestoreWorld(w.cfg, ck, WithAudit(), WithStrategy("script", &hireScript{plan: plan}))
	if err != nil {
		t.Fatalf("RestoreWorld() = %v, want nil", err)
	}
	return r
}

// checkpointBytes is the encoding of the checkpoint of w.
func checkpointBytes(t *testing.T, w *World) []byte {
	t.Helper()
	ck, err := w.Checkpoint()
	if err != nil {
		t.Fatalf("Checkpoint() = %v, want nil", err)
	}
	b, err := ck.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary() = %v, want nil", err)
	}
	return b
}

// TestCheckpoint_EmploymentRoundTrip verifies that the section of tag 24 restores the
// hires in flight and the next hire ID exactly, in sparse rows with one spelling:
// two hires of 3 ticks two ticks in, with the runs of the Warn lines set, are
// restored with the same records (the derived fields from the workers' jobs), the same
// snapshot rows, the same bytes when checkpointed again, and the same later hires and
// wages as the original; a restore after a removal (a worker; an employer) is as exact.
//
// Requirement: FR-EMP-044, FR-EMP-043, FR-EMP-055
func TestCheckpoint_EmploymentRoundTrip(t *testing.T) {
	t.Parallel()
	plan := farmWants(2, 100, 40)
	for _, tc := range []struct {
		name string
		do   func(*World)
	}{
		{"mid-job", func(*World) {}},
		{"after a worker was removed", func(w *World) { w.removeAgent(&w.agents[slotsOf(w, "hand")[0]], CauseConsumer) }},
		{"after the employer was removed", func(w *World) { w.removeAgent(&w.agents[slotsOf(w, "farm")[0]], CauseConsumer) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			w := hireWorld(t, defaultHire(hireOpts{hands: 2, handRecipe: true, duration: 4}), plan)
			stepN(t, w, 2)
			w.employment.unmatchedRun[1], w.employment.unfilledRun[0] = 2, 1
			tc.do(w)
			checkRestoredHires(t, w, hireRestore(t, w, plan))
		})
	}
}

// checkRestoredHires fails the test unless the restored world r holds what w holds:
// the hires (without the reserve and the bid, which are not saved), the next ID, the
// runs, the snapshot rows and the bytes of its checkpoint, and keeps stepping as w does.
func checkRestoredHires(t *testing.T, w, r *World) {
	t.Helper()
	if !slices.Equal(r.employment.hires, withoutLimits(w.employment.hires)) || r.employment.next != w.employment.next {
		t.Errorf("restored hires %+v next %d, want %+v next %d", r.employment.hires, r.employment.next, withoutLimits(w.employment.hires), w.employment.next)
	}
	if !slices.Equal(r.employment.unmatchedRun, w.employment.unmatchedRun) || !slices.Equal(r.employment.unfilledRun, w.employment.unfilledRun) {
		t.Errorf("restored runs %v %v, want %v %v", r.employment.unmatchedRun, r.employment.unfilledRun, w.employment.unmatchedRun, w.employment.unfilledRun)
	}
	checkRestoredRows(t, w, r)
	for range 4 {
		stepN(t, w, 1)
		stepN(t, r, 1)
	}
	if !bytes.Equal(checkpointBytes(t, r), checkpointBytes(t, w)) || !reflect.DeepEqual(errVal(r.Snapshot()), errVal(w.Snapshot())) {
		t.Errorf("the restored world and the original differ four ticks later")
	}
}

// checkRestoredRows fails the test unless the snapshot rows of the hires and the bytes
// of the checkpoint of r are those of w.
func checkRestoredRows(t *testing.T, w, r *World) {
	t.Helper()
	sr, sw := errVal(r.Snapshot()), errVal(w.Snapshot())
	if !reflect.DeepEqual(sr.Hires, sw.Hires) || !reflect.DeepEqual(sr.Roles, sw.Roles) || sr.Escrow != sw.Escrow {
		t.Errorf("restored snapshot rows differ: hires %+v roles %+v, want %+v %+v", sr.Hires, sr.Roles, sw.Hires, sw.Roles)
	}
	if !bytes.Equal(checkpointBytes(t, r), checkpointBytes(t, w)) {
		t.Errorf("the restored world writes other bytes: the encoding has more than one spelling")
	}
}

// withoutLimits returns the hires with the reserve and the bid at 0: they fixed the
// wage and only HireFormed reads them, so the section does not save them.
func withoutLimits(hires []hireRec) []hireRec {
	out := slices.Clone(hires)
	for i := range out {
		out[i].reserve, out[i].bid = 0, 0
	}
	return out
}

// hireBody is a body of tag 24 with the hires given as (hire, employer, worker, wage,
// escrow, formed, region, paid) rows and the two runs empty.
func hireBody(next uint64, recs ...[8]uint64) []byte {
	return hireBodyRuns(next, func(cw *codec.Writer) { cw.Count(0); cw.Count(0) }, recs...)
}

// hireBodyRuns is hireBody with the two runs written by runs.
func hireBodyRuns(next uint64, runs func(*codec.Writer), recs ...[8]uint64) []byte {
	cw := codec.NewWriter(nil)
	cw.U(next)
	cw.Count(len(recs))
	for _, r := range recs {
		for _, v := range r {
			cw.U(v)
		}
	}
	runs(cw)
	return cw.Out()
}

// loadHireBody loads a body of tag 24 into w after clearing the hires it read from its
// own state, and returns the error of the load and then of the end of the body.
func loadHireBody(w *World, body []byte) error {
	e := w.employment
	e.hires, e.next = nil, 0
	clear(e.workerOf)
	clear(e.hired)
	clear(e.open)
	clear(e.unmatchedRun)
	clear(e.unfilledRun)
	r := codec.NewReader(body)
	err := w.loadEmployment(&restoreCtx{tick: w.Tick()}, r)
	if err == nil {
		err = r.Done()
	}
	return err
}

// TestCheckpoint_EmploymentRestoreRejects verifies one failing body for each check of
// the section: a body that is valid loads; a party that is not live or not of the
// roles, a worker with no job from the employer, an account that is no open Escrow
// account, a hire above the tick or out of order, a wage or a wages-paid figure that
// does not fit, a run out of bounds or at its default, trailing or missing bytes and a
// hired job with no hire are each an error, the cross-checks ErrCheckpointCorrupt.
//
// Requirement: FR-EMP-044
func TestCheckpoint_EmploymentRestoreRejects(t *testing.T) {
	t.Parallel()
	w := hireWorld(t, defaultHire(hireOpts{hands: 2, handRecipe: true, duration: 4}), farmWants(1, 100, 40))
	stepN(t, w, 1)
	h := w.employment.hires[0]
	farm, hand, other := uint64(h.employer), uint64(h.worker), uint64(slotsOf(w, "hand")[0])
	if other == hand {
		other = uint64(slotsOf(w, "hand")[1])
	}
	esc, acct := uint64(h.escrow), uint64(w.agents[h.employer].account)
	good := [8]uint64{0, farm, hand, uint64(h.wage), esc, 0, 0, uint64(h.paid)}
	mod := func(i int, v uint64) [8]uint64 { r := good; r[i] = v; return r }
	for _, tc := range []struct {
		name string
		body []byte
		want error // nil: loads; errBad: any error
	}{
		{"valid", hireBody(1, good), nil},
		{"no hire but a hired job", hireBody(0), ErrCheckpointCorrupt},
		{"employer is no live agent", hireBody(1, mod(1, 0)), ErrCheckpointCorrupt},
		{"worker is no live agent", hireBody(1, mod(2, farm+9_000)), errBad},
		{"employer and worker are one agent", hireBody(1, mod(2, farm)), ErrCheckpointCorrupt},
		{"worker is of no Hireable role", hireBody(1, [8]uint64{0, hand, farm, uint64(h.wage), esc, 0, 0, uint64(h.paid)}), ErrCheckpointCorrupt},
		{"worker has no job from the employer", hireBody(1, mod(2, other)), ErrCheckpointCorrupt},
		{"escrow is an Agent account", hireBody(1, mod(4, acct)), ErrCheckpointCorrupt},
		{"escrow is no account", hireBody(1, mod(4, 1<<40)), errBad},
		{"hire not below the next", hireBody(0, good), errBad},
		{"hires not ascending", hireBody(3, mod(0, 2), mod(0, 1)), errBad},
		{"formed at or after the tick", hireBody(1, mod(5, uint64(w.Tick()))), errBad},
		{"region unknown", hireBody(1, mod(6, 9)), errBad},
		{"wage 0", hireBody(1, mod(3, 0)), ErrCheckpointCorrupt},
		{"wages paid not a whole number of wages", hireBody(1, mod(7, uint64(h.paid)+1)), ErrCheckpointCorrupt},
		// One working tick is behind the hire (worked 1), so 0 is a whole number of wages but
		// below the floor worked * wage * executions.
		{"wages paid below the ticks worked", hireBody(1, mod(7, 0)), ErrCheckpointCorrupt},
		{"run above the tick", hireBodyRuns(1, func(cw *codec.Writer) {
			cw.Count(1)
			cw.U(0)
			cw.U(0)
			cw.U(uint64(w.Tick()) + 1)
			cw.Count(0)
		}, good), errBad},
		{"run at its default", hireBodyRuns(1, func(cw *codec.Writer) {
			cw.Count(1)
			cw.U(0)
			cw.U(0)
			cw.U(0)
			cw.Count(0)
		}, good), errBad},
		{"trailing bytes", append(hireBody(1, good), 1), errBad},
		{"truncated", hireBody(1, good)[:5], errBad},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ww := hireWorld(t, defaultHire(hireOpts{hands: 2, handRecipe: true, duration: 4}), farmWants(1, 100, 40))
			stepN(t, ww, 1)
			err := loadHireBody(ww, tc.body)
			switch {
			case tc.want == nil && err != nil:
				t.Errorf("a valid body = %v, want nil", err)
			case tc.want == errBad && err == nil:
				t.Errorf("an invalid body loaded, want an error")
			case tc.want != nil && tc.want != errBad && !errors.Is(err, tc.want):
				t.Errorf("error = %v, want %v", err, tc.want)
			}
		})
	}
}

// TestCheckpoint_EmploymentSharedAccountRejected verifies that two hires that name one
// Escrow account are refused by the section's own check, with ErrCheckpointCorrupt.
//
// Requirement: FR-EMP-044
func TestCheckpoint_EmploymentSharedAccountRejected(t *testing.T) {
	t.Parallel()
	w := hireWorld(t, defaultHire(hireOpts{hands: 2, handRecipe: true, duration: 4}), farmWants(2, 100, 40))
	stepN(t, w, 2)
	a, b := w.employment.hires[0], w.employment.hires[1]
	rec := func(h hireRec, esc int) [8]uint64 {
		return [8]uint64{uint64(h.id), uint64(h.employer), uint64(h.worker), uint64(h.wage), uint64(esc), uint64(h.formed), uint64(h.region), uint64(h.paid)}
	}
	if err := loadHireBody(w, hireBody(2, rec(a, a.escrow), rec(b, b.escrow))); err != nil {
		t.Fatalf("the saved hires = %v, want nil", err)
	}
	if err := loadHireBody(w, hireBody(2, rec(a, a.escrow), rec(b, a.escrow))); !errors.Is(err, ErrCheckpointCorrupt) {
		t.Errorf("two hires on one account = %v, want ErrCheckpointCorrupt", err)
	}
}

// TestCheckpoint_EmploymentBillOverflowRejected verifies that a hire whose bill (wage * ticks
// * executions) does not fit int64 is refused as ErrCheckpointCorrupt rather than compared
// after a wrapped product, and that the audit reports it as ErrAuditFailed.
//
// Requirement: FR-EMP-044
func TestCheckpoint_EmploymentBillOverflowRejected(t *testing.T) {
	t.Parallel()
	w := hireWorld(t, defaultHire(hireOpts{hands: 2, handRecipe: true, duration: 10}), farmWants(1, 100, 40))
	stepN(t, w, 1)
	h := w.employment.hires[0]
	w.agents[h.worker].job.executions = MaxQuantity
	body := hireBody(1, [8]uint64{0, uint64(h.employer), uint64(h.worker), MaxPrice, uint64(h.escrow), 0, 0, MaxPrice * MaxQuantity})
	if err := loadHireBody(w, body); !errors.Is(err, ErrCheckpointCorrupt) {
		t.Errorf("a bill above int64 = %v, want ErrCheckpointCorrupt", err)
	}
	h2 := &w.employment.hires[0]
	h2.wage, h2.k = MaxPrice, MaxQuantity
	if err := w.auditHires(); !errors.Is(err, ErrAuditFailed) {
		t.Errorf("the audit of a bill above int64 = %v, want ErrAuditFailed", err)
	}
}

// FuzzDecodeEmploymentSection verifies that no body makes the decoder of tag 24 panic
// or allocate without bound: any bytes load or fail with an error, whatever the world
// the section is read into.
func FuzzDecodeEmploymentSection(f *testing.F) {
	w := hireWorld(f, defaultHire(hireOpts{hands: 2, handRecipe: true, duration: 4}), farmWants(2, 100, 40))
	for range 2 {
		if err := w.Step(); err != nil {
			f.Fatalf("Step() = %v, want nil", err)
		}
	}
	cw := codec.NewWriter(nil)
	if err := w.saveEmployment(cw); err != nil {
		f.Fatalf("saveEmployment() = %v, want nil", err)
	}
	f.Add(cw.Out())
	f.Add(hireBody(0))
	f.Add([]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x01})
	f.Fuzz(func(t *testing.T, body []byte) {
		_ = loadHireBody(w, body) // any body loads or fails; the next one clears what this one set
	})
}

// TestEmployment_OffLeavesNoTrace verifies that with the module off, and with the
// payroll form, the hire form leaves no trace: no section of tag 24 is expected, the
// snapshot has no hire row and no employed or employing count, AgentState has no hire
// group, and the world takes no open_hire refusal.
//
// Requirement: FR-EMP-001, FR-EMP-044
func TestEmployment_OffLeavesNoTrace(t *testing.T) {
	t.Parallel()
	off := hireWorld(t, defaultHire(hireOpts{off: true}), nil)
	stepN(t, off, 2)
	s := errVal(off.Snapshot())
	if len(s.Hires) != 0 || s.Roles[0].Employed != 0 || s.Roles[0].Employing != 0 {
		t.Errorf("snapshot Hires %v, Employed %d, Employing %d, want none", s.Hires, s.Roles[0].Employed, s.Roles[0].Employing)
	}
	if st := errVal(off.Agent(AgentID(slotsOf(off, "farm")[0]))); st.HasEmployer || st.Employer != 0 || st.Hired != 0 {
		t.Errorf("AgentState hire group = %v %d %d, want zero", st.HasEmployer, st.Employer, st.Hired)
	}
	payroll := off.cfg
	payroll.Modules.Employment = &Employment{}
	hire := hireWorld(t, defaultHire(hireOpts{}), nil).cfg
	i := slices.IndexFunc(checkpointSections[:], func(s section) bool { return s.tag == 24 })
	if i < 0 {
		t.Fatalf("no row of tag 24 in checkpointSections")
	}
	if got := []bool{checkpointSections[i].present(&off.cfg), checkpointSections[i].present(&payroll), checkpointSections[i].present(&hire)}; !slices.Equal(got, []bool{false, false, true}) {
		t.Errorf("section 24 present with the module off, with the payroll form, with the hire form = %v, want [false false true]", got)
	}
}
