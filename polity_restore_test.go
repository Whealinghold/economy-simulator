package economysim_test

import (
	"encoding/binary"
	"errors"
	"strings"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// ledgerRecords splits the body of the ledger section (tag 2) into its account
// records: the count of accounts after the boundaries, then for each account its
// kind, a closed flag and, for an open one, region, owner, balance and the lots
// (commodity, quantity, born, durability). It returns the record bytes in ID order.
func ledgerRecords(t *testing.T, body []byte) [][]byte {
	t.Helper()
	n, off := uv(t, body, 0)
	recs := make([][]byte, 0, n)
	for range n {
		start := off
		_, off = uv(t, body, off) // kind
		var closed uint64
		closed, off = uv(t, body, off)
		if closed == 0 {
			for range 3 { // region, owner, balance
				_, off = uv(t, body, off)
			}
			var lots uint64
			lots, off = uv(t, body, off)
			for range lots * 4 {
				_, off = uv(t, body, off)
			}
		}
		recs = append(recs, body[start:off])
	}
	return recs
}

// ledgerBody joins account records into a ledger section body, up to the end of the
// accounts; the bytes after them are kept from the original.
func ledgerBody(t *testing.T, orig []byte, recs [][]byte) []byte {
	t.Helper()
	old := ledgerRecords(t, orig)
	_, countLen := uv(t, orig, 0)
	skip := countLen
	for _, r := range old {
		skip += len(r)
	}
	out := binary.AppendUvarint(nil, uint64(len(recs)))
	for _, r := range recs {
		out = append(out, r...)
	}
	return append(out, orig[skip:]...)
}

// TestRestore_TreasuryAccountsCorrupt verifies FR-POL-030 on a real checkpoint of the
// governed world at tick 3, edited in the ledger section and re-digested: a restore
// rejects as ErrCheckpointCorrupt a checkpoint whose account list lacks a treasury,
// holds a second one, misplaces one (swapped with another, placed after an agent
// account), gives one the kind of an agent, another owner, another region or closes it.
// The unedited checkpoint restores, so each refusal is the edit's.
//
// Arithmetic: polityWorldConfig has six regions, so the ledger records 0 to 5 are the
// estates, 6 is crown's treasury (owner jCrown, region capital 0), 7 is inland's (owner
// jInland, region mill 4) and 8 is the first agent account. A record is kind, closed,
// region, owner, balance, lot count: the treasury of crown is the bytes 5, 0, 0, 1.
//
// Requirement: FR-POL-030
func TestRestore_TreasuryAccountsCorrupt(t *testing.T) {
	t.Parallel()
	w := polityWorld(t)
	for range 3 {
		if err := w.Step(); err != nil {
			t.Fatalf("Step() error = %v, want nil", err)
		}
	}
	raw := mustMarshal(t, mustCheckpoint(t, w))
	edit := func(f func(recs [][]byte) [][]byte) func(secs []section) []section {
		return func(secs []section) []section {
			return editSection(secs, 2, func(b []byte) []byte {
				recs := ledgerRecords(t, b)
				if recs[6][0] != 5 || recs[7][0] != 5 || recs[8][0] == 5 {
					t.Fatalf("records 6, 7, 8 start with kinds %d, %d, %d, want 5, 5 and an agent's", recs[6][0], recs[7][0], recs[8][0])
				}
				return ledgerBody(t, b, f(recs))
			})
		}
	}
	tests := []struct {
		name string
		edit func(secs []section) []section
		want string // a fragment of the message when the attach finds it, "" when another check may
	}{
		{"unedited", edit(func(r [][]byte) [][]byte { return r }), ""},
		{"treasury missing", edit(func(r [][]byte) [][]byte { return append(r[:7:7], r[8:]...) }), "Treasury account"},
		{"treasuries swapped", edit(func(r [][]byte) [][]byte { r[6], r[7] = r[7], r[6]; return r }), "Treasury account"},
		{"treasury after an agent account", edit(func(r [][]byte) [][]byte { r[7], r[8] = r[8], r[7]; return r }), "Treasury account"},
		{"treasury of the kind agent", edit(func(r [][]byte) [][]byte { r[6] = append([]byte{0}, r[6][1:]...); return r }), "Treasury account"},
		{"treasury of another owner", edit(func(r [][]byte) [][]byte { r[6] = append([]byte{5, 0, 0, jCoast}, r[6][4:]...); return r }), "Treasury account"},
		{"treasury in another region", edit(func(r [][]byte) [][]byte { r[7] = append([]byte{5, 0, rFarm}, r[7][3:]...); return r }), "Treasury account"},
		{"treasury closed", edit(func(r [][]byte) [][]byte { r[6] = []byte{5, 1}; return r }), "Treasury account"},
		{"second treasury for one jurisdiction", edit(func(r [][]byte) [][]byte { r[8] = r[7]; return r }), ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := reframe(t, raw, tt.edit)
			r, err := economysim.RestoreWorld(polityWorldConfig(), c)
			if tt.name == "unedited" {
				if err != nil || r == nil {
					t.Fatalf("RestoreWorld(unedited) = %v, %v, want a world", r, err)
				}
				return
			}
			if !errors.Is(err, economysim.ErrCheckpointCorrupt) {
				t.Fatalf("RestoreWorld(%s) error = %v, want ErrCheckpointCorrupt", tt.name, err)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("RestoreWorld(%s) error = %q, want it to name %q", tt.name, err, tt.want)
			}
		})
	}
}
