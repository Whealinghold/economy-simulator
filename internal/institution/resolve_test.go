package institution_test

import (
	"testing"

	"github.com/Whealinghold/economy-sim/internal/institution"
)

// TestResolve_InnermostChain verifies that the row in force for a subject at a place
// is the row keyed by the place, else the row of the first jurisdiction of the chain
// (innermost first) that has one, else none.
//
// Requirement: FR-ORG-014
func TestResolve_InnermostChain(t *testing.T) {
	t.Parallel()
	place := func(k int32) institution.Scope { return institution.Scope{Key: k} }
	jur := func(k int32) institution.Scope { return institution.Scope{Jurisdiction: true, Key: k} }
	rows := []institution.Row{
		{Subject: 1, Scope: jur(7)},   // 0: outer jurisdiction, subject 1
		{Subject: 1, Scope: jur(5)},   // 1: inner jurisdiction, subject 1
		{Subject: 1, Scope: place(3)}, // 2: place 3, subject 1
		{Subject: 2, Scope: jur(7)},   // 3: outer jurisdiction, subject 2 only
	}
	chain := []int32{5, 7} // innermost first
	tests := []struct {
		name    string
		subject int32
		at      int32
		want    int
		wantOK  bool
	}{
		{"the place's own row wins over its chain", 1, 3, 2, true},
		{"the first jurisdiction of the chain that has a row", 1, 4, 1, true},
		{"an outer row is used when the inner has none", 2, 4, 3, true},
		{"no row for the subject", 9, 4, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := institution.Resolve(rows, tt.subject, tt.at, chain)
			if ok != tt.wantOK || (ok && got != tt.want) {
				t.Errorf("Resolve(rows, %d, %d, %v) = %d, %v, want %d, %v", tt.subject, tt.at, chain, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

// TestResolve_SuspendedKeyOpen verifies that a suspended row leaves its key open: the
// result is none and no outer row is consulted, so the activity is open to everyone.
//
// Requirement: FR-ORG-015
func TestResolve_SuspendedKeyOpen(t *testing.T) {
	t.Parallel()
	rows := []institution.Row{
		{Subject: 1, Scope: institution.Scope{Jurisdiction: true, Key: 7}},
		{Subject: 1, Scope: institution.Scope{Jurisdiction: true, Key: 5}, Suspended: true},
	}
	// The inner row (index 1) is suspended; the outer row (index 0) must not be used.
	if got, ok := institution.Resolve(rows, 1, 4, []int32{5, 7}); ok {
		t.Errorf("Resolve with the inner row suspended = %d, true, want none", got)
	}
	// Without the suspension the inner row is in force.
	rows[1].Suspended = false
	if got, ok := institution.Resolve(rows, 1, 4, []int32{5, 7}); !ok || got != 1 {
		t.Errorf("Resolve without the suspension = %d, %v, want 1, true", got, ok)
	}
}

// TestOverlap_Detected verifies that two lists sharing both an item and a place are
// reported as the lowest pair with the lowest shared item and place, and that lists
// that share only an item or only a place are not an overlap.
//
// Requirement: FR-ORG-004
func TestOverlap_Detected(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		items, places [][]int32
		a, b          int
		item, at      int32
		ok            bool
	}{
		{"no lists", nil, nil, 0, 0, 0, 0, false},
		{"share an item but no place", [][]int32{{1, 2}, {2, 3}}, [][]int32{{0}, {1}}, 0, 0, 0, 0, false},
		{"share a place but no item", [][]int32{{1}, {2}}, [][]int32{{0, 1}, {1, 2}}, 0, 0, 0, 0, false},
		{"share both", [][]int32{{1, 2}, {2, 3}}, [][]int32{{0, 1}, {1, 2}}, 0, 1, 2, 1, true},
		{"the lowest pair is reported", [][]int32{{1}, {2}, {1, 2}}, [][]int32{{0}, {5}, {0, 5}}, 0, 2, 1, 0, true},
		{"the lowest shared item and place", [][]int32{{1, 3, 5}, {3, 5}}, [][]int32{{2, 4}, {1, 2, 4}}, 0, 1, 3, 2, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a, b, item, at, ok := institution.Overlap(tt.items, tt.places)
			if ok != tt.ok || (ok && (a != tt.a || b != tt.b || item != tt.item || at != tt.at)) {
				t.Errorf("Overlap(%v, %v) = %d, %d, %d, %d, %v, want %d, %d, %d, %d, %v",
					tt.items, tt.places, a, b, item, at, ok, tt.a, tt.b, tt.item, tt.at, tt.ok)
			}
		})
	}
}

// FuzzResolve checks that Resolve never panics and returns an index of a row that
// has the subject and is keyed by the place or by a jurisdiction of the chain.
func FuzzResolve(f *testing.F) {
	f.Add(int32(1), int32(3), int32(5), int32(7), true, false)
	f.Add(int32(0), int32(0), int32(0), int32(0), false, true)
	f.Fuzz(func(t *testing.T, subject, at, j1, j2 int32, suspend, jur bool) {
		rows := []institution.Row{
			{Subject: subject, Scope: institution.Scope{Jurisdiction: jur, Key: j1}, Suspended: suspend},
			{Subject: subject, Scope: institution.Scope{Key: at}},
		}
		ref, ok := institution.Resolve(rows, subject, at, []int32{j1, j2})
		if ok && (ref < 0 || ref >= len(rows) || rows[ref].Suspended || rows[ref].Subject != subject) {
			t.Errorf("Resolve returned %d, true for rows %v", ref, rows)
		}
	})
}
