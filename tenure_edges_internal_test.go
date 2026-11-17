package economysim

import (
	"bytes"
	"errors"
	"reflect"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/codec"
)

// TestTransferTitles_RemovedAgentIsRefusedOnEitherSide verifies that an agent removed
// after it held a title can neither send nor receive one: the call returns
// ErrAgentRemoved for the entry's side that names it, and changes nothing.
//
// Requirement: FR-LND-025
//
// Guard: the live check of checkHolder is the only thing that keeps a title from being
// given to an account that no longer exists; without the test a refactor of the order of
// the holder checks could let it through to the ledger.
func TestTransferTitles_RemovedAgentIsRefusedOnEitherSide(t *testing.T) {
	t.Parallel()
	w := tenureWorld(t, nil)
	gone := addHolder(t, w, 0, 100, 10)
	if err := w.RemoveAgent(gone); err != nil {
		t.Fatalf("RemoveAgent error = %v, want nil", err)
	}
	before := tnHolders(t, w, 0)
	for name, c := range map[string]TitleTransfer{
		"removed sender":   tr(0, gone, 3, 1, 0),
		"removed receiver": tr(0, 2, gone, 1, 0),
	} {
		if err := w.TransferTitles([]TitleTransfer{c}); !errors.Is(err, ErrAgentRemoved) {
			t.Errorf("%s: TransferTitles error = %v, want ErrAgentRemoved", name, err)
		}
	}
	tnEq(t, "titles after the refused calls", tnHolders(t, w, 0), before)
}

// TestCheckpoint_RestoreRejectsMalformedSparseRows verifies the checks of the sparse parts
// of tag 25 that the table of TestCheckpoint_RestoreRejects leaves out: a key of a scale or
// of a run that is not above the one before it, that appears twice, or that names a region
// the configuration does not have, is an error, and a well-formed pair loads.
//
// Requirement: FR-LND-035
func TestCheckpoint_RestoreRejectsMalformedSparseRows(t *testing.T) {
	t.Parallel()
	scales := func(rows ...[3]uint64) func(cw *codec.Writer) {
		return func(cw *codec.Writer) {
			cw.Count(0)
			cw.Count(len(rows))
			for _, r := range rows {
				cw.U(r[0])
				cw.U(r[1])
				cw.U(r[2])
			}
			cw.Count(0)
		}
	}
	runs := func(rows ...[4]uint64) func(cw *codec.Writer) {
		return func(cw *codec.Writer) {
			cw.Count(0)
			cw.Count(0)
			cw.Count(len(rows))
			for _, r := range rows {
				for _, v := range r {
					cw.U(v)
				}
			}
		}
	}
	cases := []struct {
		name string
		body func(cw *codec.Writer)
		ok   bool
	}{
		{"two scales ascending", scales([3]uint64{0, 0, 5_000}, [3]uint64{1, 0, 5_000}), true},
		{"scales descending", scales([3]uint64{1, 0, 5_000}, [3]uint64{0, 0, 5_000}), false},
		{"scale key twice", scales([3]uint64{0, 0, 5_000}, [3]uint64{0, 0, 6_000}), false},
		{"scale of an unknown region", scales([3]uint64{9, 0, 5_000}), false},
		{"scale of an unknown resource", scales([3]uint64{0, 9, 5_000}), false},
		{"two runs ascending", runs([4]uint64{0, 0, 1, 0}, [4]uint64{1, 0, 0, 2}), true},
		{"runs descending", runs([4]uint64{1, 0, 1, 0}, [4]uint64{0, 0, 0, 2}), false},
		{"run of an unknown region", runs([4]uint64{9, 0, 1, 0}), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := loadBody(t, nil, tenureBody(tc.body))
			if tc.ok && err != nil {
				t.Errorf("a well-formed body = %v, want nil", err)
			}
			if !tc.ok && err == nil {
				t.Errorf("a malformed body loaded, want an error")
			}
		})
	}
}

// TestCheckpoint_TenureOmitsEndowmentsWithoutTitles verifies that an endowment with no
// title is not written (the section lists only endowments with at least one title), that
// the restored world has the same two endowments, one holding and one not, and that its
// checkpoint is byte-equal to the original's.
//
// Requirement: FR-LND-033
func TestCheckpoint_TenureOmitsEndowmentsWithoutTitles(t *testing.T) {
	t.Parallel()
	only := func(m *TenureConfig) { m.Holdings = m.Holdings[:2] } // the pool and the role of home
	w := tenureWorld(t, only)
	ck := errVal(w.Checkpoint())
	r, err := RestoreWorld(tenureWorldConfig(only), ck)
	if err != nil {
		t.Fatalf("RestoreWorld error = %v, want nil", err)
	}
	if got := len(tnHolders(t, r, 1).Titles); got != 0 {
		t.Errorf("titles of yonder after the restore = %d, want 0", got)
	}
	tnEq(t, "titles of home after the restore", tnPairs(tnHolders(t, r, 0)), tnPairs(tnHolders(t, w, 0)))
	if !bytes.Equal(errVal(ck.MarshalBinary()), errVal(errVal(r.Checkpoint()).MarshalBinary())) {
		t.Errorf("the restored world writes other bytes")
	}
	if reflect.DeepEqual(tnPairs(tnHolders(t, w, 0)), [][2]int64(nil)) {
		t.Errorf("the original holds no title in home, want some: the test would prove nothing")
	}
}
