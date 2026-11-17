package economysim

import (
	"encoding/binary"
	"slices"
	"strings"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/codec"
	tec "github.com/Whealinghold/economy-sim/internal/technology"
)

// The save and load of tag 33 on a small Go-built world (plan 3-tec, Phase 7), without a
// checkpoint frame around them. technology_checkpoint_test.go holds the same section
// through RestoreWorld on the overlay of pack three.

// techBody encodes values as unsigned varints, as spec-checkpoint.md 4.2 writes tag 33.
func techBody(vs ...uint64) []byte {
	var b []byte
	for _, v := range vs {
		b = binary.AppendUvarint(b, v)
	}
	return b
}

// techLoad runs loadTechnology of w over body and returns the error the reader holds,
// as the caller of the section would read it.
func techLoad(w *World, body []byte) error {
	r := codec.NewReader(body)
	if err := w.loadTechnology(nil, r); err != nil {
		return err
	}
	return r.Done()
}

// techGoodBody is the body of the world of techTwoByTwo (four cells of three values, then
// two trend carries) that every load case below edits by one value. Fields are in ID
// order, so field 0 is grinding (no learning, cap 1 000) and field 1 is milling (Base 100,
// 300 basis points a doubling, cap 5 000).
//
// Arithmetic of the gains it derives: (home, grinding) learns nothing, so its gain is its
// advance, 400; (home, milling) has X + Base = 200 = 2 Base, one doubling, 300, plus an
// advance of 200 is 500; (away, grinding) is its advance at the cap, 1 000; (away,
// milling) has X + Base = 800 = 8 Base, three doublings, 900, plus 4 500 is 5 400, cut to
// the cap of 5 000.
func techGoodBody() (body []byte, gains []int64) {
	return techBody(
		0, 400, 7, // (home, grinding)
		100, 200, 0, // (home, milling)
		5, 1_000, 9, // (away, grinding)
		700, 4_500, 3, // (away, milling)
		11, 12, // trend carries of grinding and milling
	), []int64{400, 500, 1_000, 5_000}
}

// TestSaveTechnology_CellsThenCarries verifies that tag 33 writes, for each cell in
// ascending (region, field) order, its Experience, Advance and CatchUp as unsigned
// varints, then the trend carry of each field in ascending order, and nothing else: no
// count, no gain and no key.
//
// Requirement: FR-TEC-027, FR-TEC-028
func TestSaveTechnology_CellsThenCarries(t *testing.T) {
	t.Parallel()
	w := newMill(t, techTwoByTwo())
	want, _ := techGoodBody()
	w.tec.cells[0] = tec.Cell{Advance: 400, CatchUp: 7}
	w.tec.cells[1] = tec.Cell{Experience: 100, Advance: 200}
	w.tec.cells[2] = tec.Cell{Experience: 5, Advance: 1_000, CatchUp: 9}
	w.tec.cells[3] = tec.Cell{Experience: 700, Advance: 4_500, CatchUp: 3}
	w.tec.trendCarry[0], w.tec.trendCarry[1] = 11, 12
	w.tec.gain[0] = 99_999 // a gain is derived, so whatever it holds is not written
	cw := codec.NewWriter(nil)
	if err := w.saveTechnology(cw); err != nil {
		t.Fatalf("saveTechnology() error = %v, want nil", err)
	}
	if got := cw.Out(); !slices.Equal(got, want) {
		t.Errorf("saveTechnology() wrote %x, want %x (4 cells of 3 values, 2 carries)", got, want)
	}
}

// TestLoadTechnology_RoundTripDerivesGains verifies that a body of tag 33 loads into the
// cells and carries in the order it was written, that the gain of each cell is derived
// from its Experience and Advance (never read), and that saving the loaded state writes
// the body again byte for byte.
//
// Requirement: FR-TEC-027, FR-TEC-028
func TestLoadTechnology_RoundTripDerivesGains(t *testing.T) {
	t.Parallel()
	w := newMill(t, techTwoByTwo())
	body, gains := techGoodBody()
	if err := techLoad(w, body); err != nil {
		t.Fatalf("loadTechnology(good body) error = %v, want nil", err)
	}
	if !slices.Equal(w.tec.gain, gains) {
		t.Errorf("loadTechnology(good body) gains = %v, want %v", w.tec.gain, gains)
	}
	if got := w.tec.cells[3]; got != (tec.Cell{Experience: 700, Advance: 4_500, CatchUp: 3}) {
		t.Errorf("loadTechnology(good body) cell 3 = %+v, want {700 4500 3}", got)
	}
	if got := w.tec.trendCarry; !slices.Equal(got, []int64{11, 12}) {
		t.Errorf("loadTechnology(good body) trend carries = %v, want [11 12]", got)
	}
	cw := codec.NewWriter(nil)
	if err := w.saveTechnology(cw); err != nil || !slices.Equal(cw.Out(), body) {
		t.Errorf("saveTechnology() after the load = %x, %v, want %x, nil", cw.Out(), err, body)
	}
}

// TestLoadTechnology_RefusesEachBound verifies that each field of tag 33 is checked
// against its own bound and that the largest value of each is accepted: Experience at
// MaxExperience, Advance at the cap of its own field (5 000 for milling, 1 000 for
// grinding, so a value legal for one field is refused for the other), CatchUp below
// 10 000 ticks of a year and a trend carry below the ticks of a year. A refusal leaves
// the gains as they were.
//
// Requirement: FR-TEC-028, FR-CHK-021
func TestLoadTechnology_RefusesEachBound(t *testing.T) {
	t.Parallel()
	tpy := uint64(newMill(t, techTwoByTwo()).tec.tpy)
	tests := []struct {
		name    string
		index   int // position in the body
		value   uint64
		wantErr bool
	}{
		{"experience at the bound", 0, MaxExperience, false},
		{"experience above the bound", 0, MaxExperience + 1, true},
		{"advance at the cap of milling", 4, 5_000, false},
		{"advance above the cap of milling", 4, 5_001, true},
		{"advance at the cap of grinding", 7, 1_000, false},
		{"a milling advance in grinding", 7, 5_000, true},
		{"catch-up at the bound", 2, 10_000*tpy - 1, false},
		{"catch-up at 10 000 ticks of a year", 2, 10_000 * tpy, true},
		{"trend carry at the bound", 12, tpy - 1, false},
		{"trend carry at the ticks of a year", 13, tpy, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w := newMill(t, techTwoByTwo())
			good, _ := techGoodBody()
			vals := make([]uint64, 0, 14)
			for b := good; len(b) > 0; {
				v, n := binary.Uvarint(b)
				vals, b = append(vals, v), b[n:]
			}
			vals[tt.index] = tt.value
			w.tec.gain[0] = 42
			err := techLoad(w, techBody(vals...))
			if (err != nil) != tt.wantErr {
				t.Fatalf("loadTechnology(%s) error = %v, want an error: %t", tt.name, err, tt.wantErr)
			}
			if tt.wantErr && w.tec.gain[0] != 42 {
				t.Errorf("loadTechnology(%s) refused but moved the gain to %d, want it untouched at 42", tt.name, w.tec.gain[0])
			}
		})
	}
}

// TestLoadTechnology_RefusesShapeErrors verifies that a body that is short, empty or
// longer than the cells and carries of the configuration is refused: the count is
// implied by the configuration, so no length is read to check against.
//
// Requirement: FR-TEC-028, FR-CHK-021
func TestLoadTechnology_RefusesShapeErrors(t *testing.T) {
	t.Parallel()
	good, _ := techGoodBody()
	tests := []struct {
		name string
		body []byte
	}{
		{"a body that ends in the carries", good[:len(good)-1]},
		{"an empty body", nil},
		{"a byte after the last carry", append(slices.Clone(good), 0)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if err := techLoad(newMill(t, techTwoByTwo()), tt.body); err == nil {
				t.Errorf("loadTechnology(%s) error = nil, want a refusal", tt.name)
			}
		})
	}
}

// TestLoadTechnology_GainThatDoesNotFitIsCorrupt verifies that a body whose cells are in
// bounds but whose gain does not fit int64 (here a field with a negative doubling, which
// validation rules out and a checkpoint of another configuration could still meet) is
// refused as a corrupt section and leaves the gains untouched, and does not panic.
//
// Requirement: FR-TEC-028
//
// Fix: db27c8d. The load derived the gains with a check that panicked as an invariant,
// so a body the bounds let through crashed the restore instead of refusing it.
func TestLoadTechnology_GainThatDoesNotFitIsCorrupt(t *testing.T) {
	t.Parallel()
	w := newMill(t, techTwoByTwo())
	w.tec.fields[1].DoublingBP = -1
	for i := range w.tec.gain {
		w.tec.gain[i] = 42
	}
	body, _ := techGoodBody()
	err := techLoad(w, body)
	if err == nil || !strings.Contains(err.Error(), "does not fit") {
		t.Fatalf("loadTechnology(a gain that does not fit) error = %v, want a refusal that says the gain does not fit", err)
	}
	for i, g := range w.tec.gain {
		if g != 42 {
			t.Errorf("loadTechnology(a gain that does not fit) gain %d = %d, want it untouched at 42", i, g)
		}
	}
}
