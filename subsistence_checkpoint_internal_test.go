package economysim

import (
	"bytes"
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/codec"
)

// longForage is forageConfig with a gather job of three ticks, so a checkpoint
// can fall inside an own-use job: the households enter the mode at the first
// flag step (no grain, no money) and start gather at P2.7.
func longForage(households int64) Config {
	c := forageConfig(households)
	c.Recipes[0].Duration = 3
	return c
}

// section19 returns the body tag 19 saves for w.
func section19(t *testing.T, w *World) []byte {
	t.Helper()
	cw := codec.NewWriter(nil)
	if err := w.saveSubsistence(cw); err != nil {
		t.Fatalf("saveSubsistence() error = %v, want nil", err)
	}
	return cw.Out()
}

// TestCheckpoint_SubsistenceRoundTrip verifies that the flag and the counter of
// every agent (tag 3) and the two runs of the Warns (tag 19, sparse) are saved and
// restored, and that a restored world holds them exactly.
//
// Requirement: FR-SUB-044
func TestCheckpoint_SubsistenceRoundTrip(t *testing.T) {
	t.Parallel()
	c := longForage(3)
	w := newMill(t, c, WithSeed(5), WithAudit())
	step(t, w, 4)
	w.sub.runBlocked[0] = 3 // the blocked run needs a refusal; set it so the section carries both rows
	on := 0
	for id := range w.sub.ownUse {
		if w.sub.ownUse[id] {
			on++
		}
	}
	if on == 0 || w.sub.runOn[0] == 0 || !slices.ContainsFunc(w.sub.ownUseJobs, func(j int64) bool { return j > 0 }) {
		t.Fatalf("households in the mode = %d, runOn = %v, jobs = %v, want some of each so the round trip carries something", on, w.sub.runOn, w.sub.ownUseJobs)
	}
	got := restoreWith(t, w, c)
	if !slices.Equal(got.sub.ownUse[:len(w.sub.ownUse)], w.sub.ownUse) || !slices.Equal(got.sub.ownUseJobs[:len(w.sub.ownUseJobs)], w.sub.ownUseJobs) ||
		!slices.Equal(got.sub.runOn, w.sub.runOn) || !slices.Equal(got.sub.runBlocked, w.sub.runBlocked) {
		t.Errorf("restored flags %v counters %v runs %v %v, want flags %v counters %v runs %v %v",
			got.sub.ownUse, got.sub.ownUseJobs, got.sub.runOn, got.sub.runBlocked, w.sub.ownUse, w.sub.ownUseJobs, w.sub.runOn, w.sub.runBlocked)
	}
	step(t, got, 3)
	step(t, w, 3)
	if !slices.Equal(got.sub.ownUseJobs[:len(w.sub.ownUseJobs)], w.sub.ownUseJobs) {
		t.Errorf("counters three ticks after the restore = %v, want the original's %v", got.sub.ownUseJobs, w.sub.ownUseJobs)
	}
}

// TestCheckpoint_AgentRecordWritesFlagOnce verifies N-128 and F3: the flag and the
// counter are in the agent record of tag 3 and tag 19 lists the runs only. With
// every household in the mode and no run, tag 19 is two counts of 0.
//
// Requirement: FR-SUB-044, FR-CHK-041
func TestCheckpoint_AgentRecordWritesFlagOnce(t *testing.T) {
	t.Parallel()
	w := newMill(t, longForage(3), WithSeed(5))
	step(t, w, 1)
	clear(w.sub.runOn)
	clear(w.sub.runBlocked)
	if on, _ := w.sub.recordFields(w.firstHousehold()); on != 1 {
		t.Fatalf("flag of the first household = %d, want 1 (it has no grain and no money)", on)
	}
	if got, want := section19(t, w), []byte{0, 0}; !bytes.Equal(got, want) {
		t.Errorf("tag 19 body with no run = %v, want %v: the flags are in tag 3", got, want)
	}
}

// firstHousehold returns the first household agent of w.
func (w *World) firstHousehold() AgentID {
	var id AgentID
	w.eachLive(func(a *agent) {
		if id == 0 {
			id = a.id
		}
	})
	return id
}

// TestCheckpoint_OwnUseNonZeroCorruptWhenOff verifies R10: a non-zero flag or
// counter in the agent record is corrupt while the module is off, and so is a
// counter without the flag while it is on.
//
// Requirement: FR-SUB-044
func TestCheckpoint_OwnUseNonZeroCorruptWhenOff(t *testing.T) {
	t.Parallel()
	off := forageConfig(1)
	off.Modules.Subsistence = nil
	var none *subsistence
	for _, tt := range []struct {
		name     string
		s        *subsistence
		on, jobs uint64
		bad      bool
	}{
		{"off, both zero", none, 0, 0, false},
		{"off, flag", none, 1, 0, true},
		{"off, counter", none, 0, 2, true},
		{"on, flag and counter", &subsistence{}, 1, 2, false},
		{"on, flag alone", &subsistence{}, 1, 0, false},
		{"on, counter without the flag", &subsistence{}, 0, 2, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.s.recordBad(tt.on, tt.jobs); got != tt.bad {
				t.Errorf("recordBad(%d, %d) = %v, want %v", tt.on, tt.jobs, got, tt.bad)
			}
		})
	}
	// Through the check of a record: a valid record of the first household of a
	// world with the module off is refused once its flag is set (R10).
	w := newMill(t, off, WithSeed(2))
	id := w.firstHousehold()
	rec := agentRec{id: uint64(id), weight: 1, carry: w.deathCarry(id)}
	names := []string{w.cfg.Roles[0].Strategy}
	if err := w.checkAgentFields(&restoreCtx{}, &rec, names); err != nil {
		t.Fatalf("checkAgentFields(valid record, module off) error = %v, want nil", err)
	}
	rec.ownUse = 1
	if err := w.checkAgentFields(&restoreCtx{}, &rec, names); err == nil {
		t.Errorf("checkAgentFields(flag set, module off) = nil, want an error")
	}
}

// TestCheckpoint_SubsistenceRejects verifies that a body of tag 19 that breaks a
// rule is refused: a run above the ticks elapsed, a row at 0, rows out of order,
// a region or role that does not exist, a trailing byte, and (the cross-check) a
// run of a role whose covered set is empty.
//
// Requirement: FR-SUB-044
func TestCheckpoint_SubsistenceRejects(t *testing.T) {
	t.Parallel()
	body := func(on, blocked [][3]uint64) []byte {
		cw := codec.NewWriter(nil)
		for _, rows := range [][][3]uint64{on, blocked} {
			cw.Count(len(rows))
			for _, r := range rows {
				cw.U(r[0])
				cw.U(r[1])
				cw.U(r[2])
			}
		}
		return cw.Out()
	}
	for _, tt := range []struct {
		name string
		body []byte
	}{
		{"run above the ticks", body([][3]uint64{{0, 0, 1}}, nil)},
		{"run at zero", body([][3]uint64{{0, 0, 0}}, nil)},
		{"unknown role", body([][3]uint64{{0, 9, 0}}, nil)},
		{"unknown region", body(nil, [][3]uint64{{4, 0, 0}})},
		{"trailing byte", append(body(nil, nil), 0)},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w := newMill(t, forageConfig(1))
			r := codec.NewReader(tt.body)
			err := w.loadSubsistence(&restoreCtx{}, r)
			if err == nil {
				err = r.Done()
			}
			if err == nil && r.Err() == nil {
				t.Errorf("loadSubsistence(%v) = no error, want one", tt.body)
			}
		})
	}
	// Sparse rows must ascend.
	w := newMill(t, forageConfig(1))
	w.sub.runOn[0] = 1
	r := codec.NewReader(body([][3]uint64{{0, 0, 3}, {0, 0, 2}}, nil))
	_ = w.loadSubsistence(&restoreCtx{tick: 9}, r)
	if r.Err() == nil {
		t.Errorf("loadSubsistence(rows out of order) = no error, want one")
	}
	// The cross-check: a run in a role with an empty covered set.
	w = newMill(t, forageConfig(1))
	w.sub.covered[0] = nil
	w.sub.runBlocked[0] = 2
	if err := w.subsistenceCross(); !errors.Is(err, ErrCheckpointCorrupt) {
		t.Errorf("subsistenceCross() with a run and an empty covered set error = %v, want ErrCheckpointCorrupt", err)
	}
}

// FuzzDecodeSubsistence verifies that corrupt bytes of tag 19 give an error and
// never a panic, and that bytes that decode and cross-check hold runs at most the
// ticks elapsed.
//
// Requirement: FR-SUB-044, NFR-CHK-003
func FuzzDecodeSubsistence(f *testing.F) {
	w, err := NewWorld(longForage(3), WithSeed(5))
	if err != nil {
		f.Fatal(err)
	}
	for range 4 {
		if err := w.Step(); err != nil {
			f.Fatal(err)
		}
	}
	cw := codec.NewWriter(nil)
	if err := w.saveSubsistence(cw); err != nil {
		f.Fatal(err)
	}
	f.Add(cw.Out())
	f.Add(cw.Out()[:len(cw.Out())/2])
	f.Add([]byte{})
	f.Add([]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x01})
	f.Fuzz(func(t *testing.T, body []byte) {
		fresh, err := NewWorld(longForage(3))
		if err != nil {
			t.Fatal(err)
		}
		r := codec.NewReader(body)
		err = fresh.loadSubsistence(&restoreCtx{tick: 4}, r)
		if err == nil {
			err = r.Done()
		}
		if err == nil {
			err = fresh.subsistenceCross()
		}
		if err != nil {
			return
		}
		for cell := range fresh.sub.runOn {
			if fresh.sub.runOn[cell] > 4 || fresh.sub.runBlocked[cell] > 4 {
				t.Fatalf("decoded runs %d, %d above the 4 ticks elapsed", fresh.sub.runOn[cell], fresh.sub.runBlocked[cell])
			}
		}
	})
}
