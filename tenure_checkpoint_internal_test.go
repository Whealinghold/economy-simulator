package economysim

import (
	"bytes"
	"errors"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/codec"
)

// stepTenure is the module of the checkpoint tests: the posted rent steps (so it is saved),
// inside a band of 50% to 150% of the configured rent.
func stepTenure(m *TenureConfig) {
	m.Terms[0].Rent = 10
	m.RentStepBP, m.RentFloorBP, m.RentCeilingBP = 1_000, 5_000, 15_000
}

// TestCheckpoint_TenureRoundTrip verifies that the section of tag 25 restores the titles
// after a transfer, the posted rent of a stepping module, the rent scales and the two runs
// exactly, in sparse rows with one spelling: a world restored from a checkpoint writes the
// same bytes, and its queries, slots and snapshot rows equal the original's.
//
// Requirement: FR-LND-033, FR-LND-034
func TestCheckpoint_TenureRoundTrip(t *testing.T) {
	t.Parallel()
	w := tenureWorld(t, stepTenure)
	for range 6 {
		if err := w.Step(); err != nil {
			t.Fatalf("Step error = %v, want nil", err)
		}
	}
	if err := w.TransferTitles([]TitleTransfer{tr(0, 2, 5, 30, 0), tr(1, NoAgent, 8, 70, 0)}); err != nil {
		t.Fatalf("TransferTitles error = %v, want nil", err)
	}
	if err := w.SetRentScales([]EndowmentScale{{Region: 0, BP: 2_500}}); err != nil {
		t.Fatalf("SetRentScales error = %v, want nil", err)
	}
	w.tenure.posted[0], w.tenure.noTitleRun[1], w.tenure.rentFundsRun[1] = 12, 4, 0
	ck := errVal(w.Checkpoint())
	r, err := RestoreWorld(tenureWorldConfig(stepTenure), ck)
	if err != nil {
		t.Fatalf("RestoreWorld error = %v, want nil", err)
	}
	for g := RegionID(0); g < 2; g++ {
		tnEq(t, "Holders after the restore", tnHolders(t, r, g), tnHolders(t, w, g))
	}
	tnEq(t, "Slots after the restore", errVal(r.Slots()), errVal(w.Slots()))
	tnEq(t, "posted rent and runs after the restore", [3]int64{r.tenure.posted[0], r.tenure.noTitleRun[1], r.tenure.rentFundsRun[1]}, [3]int64{12, 4, 0})
	ck2 := errVal(r.Checkpoint())
	if !bytes.Equal(errVal(ck.MarshalBinary()), errVal(ck2.MarshalBinary())) {
		t.Errorf("the restored world writes other bytes: the encoding has more than one spelling")
	}
}

// tenureBody builds a section body of tag 25 for the tests of the checks: fill writes it.
func tenureBody(fill func(cw *codec.Writer)) []byte {
	cw := codec.NewWriter(nil)
	fill(cw)
	return cw.Out()
}

// emptyTail writes the three sparse and dense parts after the holdings of a world with no
// stepping: no scale and no run.
func emptyTail(cw *codec.Writer) { cw.Count(0); cw.Count(0) }

// loadBody loads a body into a fresh world and returns the error, which is the module's
// own, before the wrapping that loadSections adds.
func loadBody(t testing.TB, mut func(m *TenureConfig), body []byte) error {
	t.Helper()
	w := tenureWorld(t, mut)
	w.tenure.holds = make([][]title, len(w.tenure.holds))
	r := codec.NewReader(body)
	err := w.loadTenure(&restoreCtx{tick: 10}, r)
	if err == nil {
		err = r.Done()
	}
	return err
}

// tnRecord writes one endowment record: the key and the titles as (holder, share) pairs.
func tnRecord(cw *codec.Writer, region, resource uint64, pairs ...uint64) {
	cw.U(region)
	cw.U(resource)
	cw.Count(len(pairs) / 2)
	for _, p := range pairs {
		cw.U(p)
	}
}

// TestCheckpoint_RestoreRejects verifies one failing section for each check of spec-land.md
// 14.2: each is an error, an unknown endowment is ErrCheckpointConfig, a body that is
// valid loads, and the module's checks are all in the load.
//
// Requirement: FR-LND-035
func TestCheckpoint_RestoreRejects(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		mut  func(m *TenureConfig)
		body func(cw *codec.Writer)
		want error // nil: no error expected, errBad: any error
	}{
		{"valid", nil, func(cw *codec.Writer) { cw.Count(1); tnRecord(cw, 0, 0, 0, 1_000, 2, 100); emptyTail(cw) }, nil},
		{"unknown region", nil, func(cw *codec.Writer) { cw.Count(1); tnRecord(cw, 5, 0, 0, 1); emptyTail(cw) }, ErrCheckpointConfig},
		{"unknown resource", nil, func(cw *codec.Writer) { cw.Count(1); tnRecord(cw, 0, 3, 0, 1); emptyTail(cw) }, ErrCheckpointConfig},
		{"share 0", nil, func(cw *codec.Writer) { cw.Count(1); tnRecord(cw, 0, 0, 2, 0); emptyTail(cw) }, errBad},
		{"share above 10000", nil, func(cw *codec.Writer) { cw.Count(1); tnRecord(cw, 0, 0, 2, 10_001); emptyTail(cw) }, errBad},
		{"dead holder", nil, func(cw *codec.Writer) { cw.Count(1); tnRecord(cw, 0, 0, 40, 5); emptyTail(cw) }, errBad},
		{"pool of another region", nil, func(cw *codec.Writer) { cw.Count(1); tnRecord(cw, 0, 0, 1, 5); emptyTail(cw) }, errBad},
		{"holders not ascending", nil, func(cw *codec.Writer) { cw.Count(1); tnRecord(cw, 0, 0, 3, 5, 2, 5); emptyTail(cw) }, errBad},
		{"holder twice", nil, func(cw *codec.Writer) { cw.Count(1); tnRecord(cw, 0, 0, 2, 5, 2, 5); emptyTail(cw) }, errBad},
		{"sum above 10000", nil, func(cw *codec.Writer) { cw.Count(1); tnRecord(cw, 0, 0, 0, 9_000, 2, 1_001); emptyTail(cw) }, errBad},
		{"agent above the cap", func(m *TenureConfig) { m.MaxHolderShareBP = 100 }, func(cw *codec.Writer) { cw.Count(1); tnRecord(cw, 0, 0, 2, 101); emptyTail(cw) }, errBad},
		{"keys not ascending", nil, func(cw *codec.Writer) {
			cw.Count(2)
			tnRecord(cw, 1, 0, 1, 5)
			tnRecord(cw, 0, 0, 0, 5)
			emptyTail(cw)
		}, errBad},
		{"no titles written", nil, func(cw *codec.Writer) { cw.Count(1); tnRecord(cw, 0, 0); emptyTail(cw) }, errBad},
		{"scale at the default", nil, func(cw *codec.Writer) {
			cw.Count(0)
			cw.Count(1)
			cw.U(0)
			cw.U(0)
			cw.U(10_000)
			cw.Count(0)
		}, errBad},
		{"scale above MaxShockBP", nil, func(cw *codec.Writer) {
			cw.Count(0)
			cw.Count(1)
			cw.U(0)
			cw.U(0)
			cw.U(MaxShockBP + 1)
			cw.Count(0)
		}, errBad},
		{"run above the tick", nil, func(cw *codec.Writer) {
			cw.Count(0)
			cw.Count(0)
			cw.Count(1)
			cw.U(0)
			cw.U(0)
			cw.U(11)
			cw.U(0)
		}, errBad},
		{"both runs zero", nil, func(cw *codec.Writer) {
			cw.Count(0)
			cw.Count(0)
			cw.Count(1)
			cw.U(0)
			cw.U(0)
			cw.U(0)
			cw.U(0)
		}, errBad},
		{"posted above the ceiling", stepTenure, func(cw *codec.Writer) { cw.Count(0); cw.U(16); cw.Count(0); cw.Count(0) }, errBad},
		{"posted below the floor", stepTenure, func(cw *codec.Writer) { cw.Count(0); cw.U(4); cw.Count(0); cw.Count(0) }, errBad},
		{"posted in the band", stepTenure, func(cw *codec.Writer) { cw.Count(0); cw.U(15); cw.Count(0); cw.Count(0) }, nil},
		{"trailing bytes", nil, func(cw *codec.Writer) { cw.Count(0); emptyTail(cw); cw.U(1) }, errBad},
		{"truncated", nil, func(cw *codec.Writer) { cw.Count(1) }, errBad},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := loadBody(t, tc.mut, tenureBody(tc.body))
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

// errBad stands for any error of the load in a table of cases: the module reports a
// malformed body through the reader.
var errBad = errors.New("any error")

// FuzzDecodeTenureSection verifies that no body makes the decoder of tag 25 panic or
// allocate in proportion to a number it reads, and that a body it accepts is read to its end.
//
// Requirement: FR-LND-035
func FuzzDecodeTenureSection(f *testing.F) {
	f.Add(tenureBody(func(cw *codec.Writer) { cw.Count(1); tnRecord(cw, 0, 0, 0, 1_000, 2, 100); emptyTail(cw) }))
	f.Add(tenureBody(func(cw *codec.Writer) { cw.Count(0); emptyTail(cw) }))
	f.Add([]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x01})
	w := tenureWorld(f, nil)
	f.Fuzz(func(t *testing.T, body []byte) {
		w.tenure.holds = make([][]title, len(w.tenure.holds))
		r := codec.NewReader(body)
		if err := w.loadTenure(&restoreCtx{tick: 10}, r); err == nil && r.Err() == nil {
			_ = r.Done()
		}
	})
}
