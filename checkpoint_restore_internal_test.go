package economysim

import (
	"bytes"
	"errors"
	"reflect"
	"runtime"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/codec"
	"github.com/Whealinghold/economy-sim/internal/ledger"
)

// This file tests the restore of state that a few ticks of a small mill never
// reach: a directive, a yield shock and every diagnostic run above 0. The values
// are set on the world's own fields (as checkpoint_internal_test.go does for the
// write path), checkpointed and restored, and the restored world must hold them.

// restoreOf checkpoints w and restores a world from it with the same configuration.
func restoreOf(t *testing.T, w *World) (*World, error) {
	t.Helper()
	c, err := w.Checkpoint()
	if err != nil {
		t.Fatalf("Checkpoint() error = %v, want nil", err)
	}
	return RestoreWorld(w.Config(), c)
}

// fullDirective is a directive with every part set.
func fullDirective() *Directive {
	return &Directive{
		Job:      &JobRequest{Recipe: 0, Executions: 2},
		Intents:  []Intent{{Commodity: 1, Side: Ask, Quantity: 3, Limit: 900, Target: 4, Reserve: 1, UnitCost: 800, Purpose: PurposeInput}},
		Discards: []Discard{{Commodity: 0, Quantity: 1}},
		Hold:     []CommodityID{0},
		Sticky:   true,
	}
}

// TestRestore_SyntheticStateRoundTrips verifies that a directive (with a job
// request, an intent, a discard, a hold and Sticky, on one agent, and none on the
// others), a yield shock and every diagnostic run above 0 survive a restore, and
// that the restored world's own checkpoint equals the one it was restored from.
//
// Requirement: FR-CHK-005, FR-CHK-041, FR-CHK-042
//
// Guard: a mill of a few ticks has no directive, no shock and runs of 0, so the
// end-to-end continuation tests never read these rows; a reader that dropped or
// misplaced one would pass them.
func TestRestore_SyntheticStateRoundTrips(t *testing.T) {
	t.Parallel()
	w := ckptMill(t, 5)
	w.agents[1].directive = fullDirective()
	w.agents[2].directive = &Directive{Job: &JobRequest{NoJob: true}}
	if err := w.shocks.set(0, 0, 5_000); err != nil {
		t.Fatalf("shocks.set() error = %v, want nil", err)
	}
	st := &w.state
	st.unfedRun[0], st.floorRun[0], st.clampRun[0], st.extinctRun[0], st.belowFloorRun[0] = 2, 3, 4, 5, 1
	st.renewalCarry[0], st.unfundedRun[0] = 7, 2
	r, err := restoreOf(t, w)
	if err != nil {
		t.Fatalf("RestoreWorld() error = %v, want nil", err)
	}
	if !reflect.DeepEqual(r.agents[1].directive, w.agents[1].directive) || !reflect.DeepEqual(r.agents[2].directive, w.agents[2].directive) || r.agents[3].directive != nil {
		t.Errorf("directives = %+v, %+v, %+v, want %+v, %+v, nil", r.agents[1].directive, r.agents[2].directive, r.agents[3].directive, w.agents[1].directive, w.agents[2].directive)
	}
	if !reflect.DeepEqual(r.shocks.entries, w.shocks.entries) || !reflect.DeepEqual(r.state, w.state) {
		t.Errorf("shocks %+v and state %+v, want %+v and %+v", r.shocks.entries, r.state, w.shocks.entries, w.state)
	}
	a, _ := w.Checkpoint()
	b, _ := r.Checkpoint()
	ab, _ := a.MarshalBinary()
	bb, _ := b.MarshalBinary()
	if !bytes.Equal(ab, bb) {
		t.Errorf("the restored world's checkpoint differs from the original's")
	}
}

// TestRestore_RefusesWhatAnInvalidDirectiveWrote verifies that a directive with a
// negative amount, which Checkpoint writes as the unsigned number it converts to,
// is refused by the restore as ErrCheckpointCorrupt, as is one with a commodity
// or a purpose outside its set. The directive is validated when a consumer sets it
// in plan 2-17; until then the reader is the only check.
//
// Requirement: FR-CHK-012, FR-CHK-014
func TestRestore_RefusesWhatAnInvalidDirectiveWrote(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		edit func(d *Directive)
	}{
		{"negative quantity", func(d *Directive) { d.Intents[0].Quantity = -1 }},
		{"negative limit", func(d *Directive) { d.Intents[0].Limit = -5 }},
		{"negative executions", func(d *Directive) { d.Job.Executions = -1 }},
		{"negative discard", func(d *Directive) { d.Discards[0].Quantity = -2 }},
		{"quantity above the bound", func(d *Directive) { d.Intents[0].Quantity = MaxQuantity + 1 }},
		{"unknown commodity", func(d *Directive) { d.Hold[0] = 9 }},
		{"unknown recipe", func(d *Directive) { d.Job.Recipe = 9 }},
		{"unknown purpose", func(d *Directive) { d.Intents[0].Purpose = 9 }},
		{"unknown side", func(d *Directive) { d.Intents[0].Side = 5 }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w := ckptMill(t, 2)
			d := fullDirective()
			tt.edit(d)
			w.agents[1].directive = d
			if _, err := restoreOf(t, w); !errors.Is(err, ErrCheckpointCorrupt) {
				t.Errorf("RestoreWorld(%s) error = %v, want ErrCheckpointCorrupt", tt.name, err)
			}
		})
	}
}

// TestRestore_RefusesStateBeyondItsBounds verifies, for the sections whose values
// the write path writes as they stand, that a value outside the bound its owner's
// rule gives (R13) is refused by the restore as ErrCheckpointCorrupt: a job that
// runs longer than its recipe, names a principal who is not live or has a yield
// above its bound, a belief centre below 1 or a width above 10 000, a market price
// of 0 or an age beyond the run, a residue at the bound of its draw, a target
// ratio above its bound and hunger beyond the ticks run. (A welfare field beyond
// the clamp cannot be made in process: the ring clamps on a push; the external
// test edits the bytes.)
//
// Arithmetic: the mill's recipe lasts 3 ticks, so 4 remaining is one too many; the
// world is at tick 5, so a hunger or an age of 6 is one more than the ticks run.
//
// Requirement: FR-CHK-012, FR-CHK-013, FR-CHK-014, FR-CHK-042
func TestRestore_RefusesStateBeyondItsBounds(t *testing.T) {
	t.Parallel()
	job := func(a *agent) *job {
		a.job = job{recipe: 0, remaining: 2, executions: 1, principal: a.id, yield: 10_000, pending: []int64{3}}
		return &a.job
	}
	tests := []struct {
		name string
		edit func(w *World)
	}{
		{"job longer than its recipe", func(w *World) { job(&w.agents[1]).remaining = 4 }},
		{"job of a principal who is not live", func(w *World) { job(&w.agents[1]).principal = 4_000 }},
		{"job of the estate as principal", func(w *World) { job(&w.agents[1]).principal = 0 }},
		{"job yield above its bound", func(w *World) { job(&w.agents[1]).yield = MaxYieldBP + 1 }},
		{"job pending above the bound", func(w *World) { job(&w.agents[1]).pending[0] = MaxQuantity + 1 }},
		{"belief centre below 1", func(w *World) { w.agents[1].beliefs[0].Centre = 0 }},
		{"belief width above 10 000", func(w *World) { w.agents[1].beliefs[0].Width = 10_001 }},
		{"market price 0", func(w *World) { w.markets.reports[0].Price = 0 }},
		{"market age beyond the ticks run", func(w *World) { w.markets.reports[0].Age = 6 }},
		{"best bid of 0", func(w *World) { w.markets.reports[0].HasBestBid, w.markets.reports[0].BestBid = true, 0 }},
		{"bound run beyond the ticks run", func(w *World) { w.markets.boundRun[0] = 6 }},
		{"residue at the bound of its draw", func(w *World) { w.agents[1].residues[0] = residueBound }},
		{"target ratio above its bound", func(w *World) { w.agents[1].target = 2*MaxBasisPoints + 1 }},
		{"hunger beyond the ticks run", func(w *World) { w.agents[1].hunger = 6 }},
		{"unfed above the weight", func(w *World) { w.agents[1].unfed = 2 }},
		{"role-since after the tick", func(w *World) { w.agents[1].roleSince = 6 }},
		{"yield shock at the neutral 10 000", func(w *World) { w.shocks.entries = []shockEntry{{region: 0, recipe: 0, bp: 10_000}} }},
		{"yield shocks out of order", func(w *World) {
			w.shocks.entries = []shockEntry{{region: 0, recipe: 0, bp: 5_000}, {region: 0, recipe: 0, bp: 6_000}}
		}},
		{"next agent ID below the regions", func(w *World) { w.state.nextAgent = 0 }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w := ckptMill(t, 5)
			tt.edit(w)
			if _, err := restoreOf(t, w); !errors.Is(err, ErrCheckpointCorrupt) {
				t.Errorf("RestoreWorld(%s) error = %v, want ErrCheckpointCorrupt", tt.name, err)
			}
		})
	}
}

// TestRestore_ClosedAccountsCostAConstantPerByte verifies that a checkpoint whose
// ledger holds many closed Agent accounts (two bytes each, and an agent slot of the
// table each) restores into at most a fixed multiple of its size.
//
// Requirement: FR-CHK-016, NFR-CHK-001
//
// Guard: the agent table is sized by the next ID, which the ledger's accounts
// bound but do not shrink; a change that made a slot dearer, or sized the table
// by another number, would turn 6.6.1's constant into a multiplier nothing
// noticed.
func TestRestore_ClosedAccountsCostAConstantPerByte(t *testing.T) {
	// No t.Parallel: the figure is the process-wide allocation counter.
	const closed = 100_000
	w := ckptMill(t, 5)
	for range closed {
		id, err := w.ledger.Open(ledger.KindAgent, int(w.state.nextAgent), 0)
		if err == nil {
			err = w.ledger.Close(id)
		}
		if err != nil {
			t.Fatalf("opening and closing an account error = %v, want nil", err)
		}
		w.state.nextAgent++
	}
	c, err := w.Checkpoint()
	if err != nil {
		t.Fatalf("Checkpoint() error = %v, want nil", err)
	}
	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	r, err := RestoreWorld(w.Config(), c)
	runtime.ReadMemStats(&after)
	if err != nil {
		t.Fatalf("RestoreWorld() error = %v, want nil", err)
	}
	runtime.KeepAlive(r)
	got, size := after.TotalAlloc-before.TotalAlloc, uint64(len(c.b))
	t.Logf("restore allocated %d bytes for %d: %d per byte", got, size, got/size)
	if got > 400*size {
		t.Errorf("RestoreWorld() allocated %d bytes for a checkpoint of %d, want at most %d times that", got, size, 400)
	}
}

// TestRestore_NoJobDirectiveWithJunkFieldsRestores verifies that a directive whose
// job request has NoJob set restores, with the ignored recipe and executions read
// back as 0 whatever they held when it was set.
//
// Fix: review of stage-2-2b. Those fields were written as given, and the restore
// bounded them like a live job's, so a recipe out of range made a checkpoint that
// the same library refused.
func TestRestore_NoJobDirectiveWithJunkFieldsRestores(t *testing.T) {
	t.Parallel()
	w := ckptMill(t, 2)
	w.agents[1].directive = &Directive{Job: &JobRequest{NoJob: true, Recipe: 9999, Executions: 7}}
	r, err := restoreOf(t, w)
	if err != nil {
		t.Fatalf("RestoreWorld() error = %v, want nil", err)
	}
	if got, want := r.agents[1].directive, (&Directive{Job: &JobRequest{NoJob: true}}); !reflect.DeepEqual(got, want) {
		t.Errorf("restored directive = %+v, want %+v", got.Job, want.Job)
	}
}

// TestLoadDirective_NoJobFieldsAreNeutral verifies, on hand-written bytes, that a
// NoJob request reads its recipe and executions as 0 even in a world with no
// recipe, and that any other value there is refused.
//
// Requirement: FR-CHK-041
//
// Fix: review of stage-2-2b. The restore bounded the ignored recipe like a live
// job's, so the 0 that saveDirective writes was out of range in a world with no
// recipe. TestRestore_NoJobDirectiveWithJunkFieldsRestores cannot see this: the
// save already writes 0 and its world has recipes.
func TestLoadDirective_NoJobFieldsAreNeutral(t *testing.T) {
	t.Parallel()
	// present, job, NoJob, recipe, executions, then three counts of 0 and not sticky.
	tests := []struct {
		name    string
		b       []byte
		wantErr bool
	}{
		{"zeros in a world with no recipe", []byte{1, 1, 1, 0, 0, 0, 0, 0, 0}, false},
		{"recipe above 0", []byte{1, 1, 1, 1, 0, 0, 0, 0, 0}, true},
		{"executions above 0", []byte{1, 1, 1, 0, 7, 0, 0, 0, 0}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := codec.NewReader(tt.b)
			d := (&World{}).loadDirective(r)
			err := r.Err()
			if tt.wantErr {
				if !errors.Is(err, codec.ErrMalformed) {
					t.Errorf("loadDirective(%x) error = %v, want codec.ErrMalformed", tt.b, err)
				}
				return
			}
			if want := (&Directive{Job: &JobRequest{NoJob: true}}); err != nil || !reflect.DeepEqual(d, want) {
				t.Errorf("loadDirective(%x) = %+v, error %v, want %+v and nil", tt.b, d, err, want)
			}
		})
	}
}

// TestRead_RefusesUnorderedNamesAndTruncatedLists verifies, on hand-written bytes,
// that the table of strategy names refuses a name that is not above the one before
// it (equal or lower), and that a directive whose first intent, discard or hold is
// out of range is refused as malformed, with no directive returned. The world has
// no commodity, so every commodity index is out of range.
//
// Requirement: FR-CHK-012, FR-CHK-014
func TestRead_RefusesUnorderedNamesAndTruncatedLists(t *testing.T) {
	t.Parallel()
	zeros := func(n int) []byte { return make([]byte, n) }
	// Directive bytes: present, no job, then the counts of intents, discards and
	// holds. A count of 2 needs 8 bytes per intent, 2 per discard and 1 per hold left
	// in the input, or the count check refuses it before the first item.
	tests := []struct {
		name string
		b    []byte
		read func(r *codec.Reader) bool // reports whether a value came back
	}{
		{"names equal", []byte{2, 1, 'a', 1, 'a'}, func(r *codec.Reader) bool { return readStrategyNames(r) != nil }},
		{"names descending", []byte{2, 1, 'b', 1, 'a'}, func(r *codec.Reader) bool { return readStrategyNames(r) != nil }},
		{"two intents", append([]byte{1, 0, 2}, zeros(16)...), func(r *codec.Reader) bool { return (&World{}).loadDirective(r) != nil }},
		{"two discards", append([]byte{1, 0, 0, 2}, zeros(4)...), func(r *codec.Reader) bool { return (&World{}).loadDirective(r) != nil }},
		{"two holds", append([]byte{1, 0, 0, 0, 2}, zeros(2)...), func(r *codec.Reader) bool { return (&World{}).loadDirective(r) != nil }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := codec.NewReader(tt.b)
			got := tt.read(r)
			if err := r.Err(); !errors.Is(err, codec.ErrMalformed) {
				t.Errorf("read of %x error = %v, want codec.ErrMalformed", tt.b, err)
			}
			if tt.name[:5] != "names" && got {
				t.Errorf("read of %x returned a directive, want nil with the error", tt.b)
			}
		})
	}
}
