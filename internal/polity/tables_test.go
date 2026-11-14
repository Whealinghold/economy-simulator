package polity_test

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/polity"
)

// TestTables_ParentAndOf verifies the two single-value readers where they answer
// "none": the parent of a root, the direct jurisdiction of a region with no row, and
// either with an index out of range, and where they answer a value: the parent of a
// child and the direct jurisdiction of a member.
//
// Requirement: FR-POL-006, FR-POL-012
func TestTables_ParentAndOf(t *testing.T) {
	t.Parallel()
	tb := exampleB(t) // east 0 and west 1 are roots; ashford and city... see exampleB
	a := exampleA(t)
	tests := []struct {
		name   string
		got    func() (int32, bool)
		want   int32
		wantOK bool
	}{
		{"parent of a child", func() (int32, bool) { return a.Parent(uplands) }, inland, true},
		{"parent of a root", func() (int32, bool) { return a.Parent(crown) }, -1, false},
		{"parent below range", func() (int32, bool) { return a.Parent(-1) }, -1, false},
		{"parent above range", func() (int32, bool) { return a.Parent(4) }, -1, false},
		{"parent of a root of example B", func() (int32, bool) { return tb.Parent(0) }, -1, false},
		{"jurisdiction of a member", func() (int32, bool) { return a.Of(hills) }, uplands, true},
		{"jurisdiction of a region with no row", func() (int32, bool) { return tb.Of(2) }, -1, false}, // city has none in example B
		{"jurisdiction of a region below range", func() (int32, bool) { return a.Of(-1) }, -1, false},
		{"jurisdiction of a region above range", func() (int32, bool) { return a.Of(5) }, -1, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got, ok := tt.got(); got != tt.want || ok != tt.wantOK {
				t.Errorf("%s = %d, %v, want %d, %v", tt.name, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

// TestBuildError_Message verifies that a BuildError names the cause and the index of
// the entry at fault, matches its sentinel with errors.Is whether the entry is a
// jurisdiction or a member row, and says which array the index is in.
//
// Requirement: FR-POL-003, FR-POL-004
func TestBuildError_Message(t *testing.T) {
	t.Parallel()
	_, err := polity.Build([]int32{1, 0}, nil, 8)
	var be *polity.BuildError
	if !errors.As(err, &be) || !errors.Is(err, polity.ErrCycle) || be.Member {
		t.Fatalf("Build(cycle) = %v, want a BuildError wrapping ErrCycle on a jurisdiction", err)
	}
	if got := be.Error(); !strings.HasSuffix(got, " at 0") || !strings.Contains(got, polity.ErrCycle.Error()) {
		t.Errorf("BuildError.Error() = %q, want the cause and \" at 0\"", got)
	}
	_, err = polity.Build([]int32{-1}, []int32{-1, 3}, 8)
	if !errors.As(err, &be) || !errors.Is(err, polity.ErrIndex) || !be.Member || be.Index != 1 {
		t.Errorf("Build(member out of range) = %v, want ErrIndex on member 1", err)
	}
	if got := be.Error(); !strings.HasSuffix(got, " at 1") {
		t.Errorf("BuildError.Error() = %q, want it to end with \" at 1\"", got)
	}
}

// TestTables_ZeroValueAnswersNone verifies that the zero Tables, which the root
// holds for a world with the module off, answers every query with none and does not
// panic.
//
// Requirement: FR-POL-001, FR-POL-012
func TestTables_ZeroValueAnswersNone(t *testing.T) {
	t.Parallel()
	var tb polity.Tables
	if tb.Len() != 0 || tb.Regions() != 0 {
		t.Errorf("zero Tables Len() = %d, Regions() = %d, want 0, 0", tb.Len(), tb.Regions())
	}
	if len(tb.Chain(0)) != 0 || tb.Covered(0) != nil || tb.Children(0) != nil || tb.Common(0, 0) != -1 {
		t.Error("zero Tables answered something, want none")
	}
	if _, ok := tb.Of(0); ok {
		t.Error("zero Tables Of(0) ok, want false")
	}
	if got := tb.Clone(); got.Len() != 0 || !slices.Equal(got.Chain(0), nil) {
		t.Error("Clone of the zero Tables is not empty")
	}
}
