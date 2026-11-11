package economysim

import (
	"errors"
	"slices"
	"testing"
)

// TestCheckLaneList_Limit verifies the bound on the length of the list: a list
// of the limit passes and one more is ErrOutOfBounds at the path solve_options.lanes.
// MaxLanes is 10^9, too many lanes to build in a test, so the limit is a
// parameter and the test lowers it.
//
// Requirement: FR-VIA-063
func TestCheckLaneList_Limit(t *testing.T) {
	t.Parallel()
	lane := func(i int64) Lane { return Lane{From: "a", To: "b", Commodity: "c", Labour: i} }
	if err := checkLaneList([]Lane{lane(0), lane(1)}, 2); err != nil {
		t.Errorf("checkLaneList(2 lanes, limit 2) = %v, want nil", err)
	}
	err := checkLaneList([]Lane{lane(0), lane(1), lane(2)}, 2)
	var ve *ValidationError
	if !errors.Is(err, ErrOutOfBounds) || !errors.As(err, &ve) || !slices.Equal(ve.Path, []string{"solve_options", "lanes"}) || ve.Value != "3" || ve.Bound != "2" {
		t.Errorf("checkLaneList(3 lanes, limit 2) = %v, want ErrOutOfBounds at solve_options.lanes with value 3 and bound 2", err)
	}
}

// TestLaneGroups verifies the components by hand over regions 0 to 5 with lanes
// 4-1 (the order of the ends is the direction), 1-3, 5-9, 9-2 and 9-0, where 9
// has no group, so it joins nothing: {1, 3, 4} are one component, 0 and 2 stand alone, 5 stands alone because
// its only lane ends in a region outside the set; the components are ordered by
// their first region and each lists its regions ascending, and a chain of lanes
// listed against the order of the regions still joins.
//
// Requirement: FR-VIA-061
func TestLaneGroups(t *testing.T) {
	t.Parallel()
	have := []RegionID{0, 1, 2, 3, 4, 5}
	lanes := []viaLane{{from: 4, to: 1}, {from: 1, to: 3}, {from: 5, to: 9}, {from: 9, to: 2}, {from: 9, to: 0}}
	want := [][]RegionID{{0}, {1, 3, 4}, {2}, {5}}
	if got := laneGroups(have, lanes); !equalGroups(got, want) {
		t.Errorf("laneGroups(%v, %v) = %v, want %v", have, lanes, got, want)
	}
	chain := []viaLane{{from: 5, to: 4}, {from: 4, to: 3}, {from: 3, to: 2}, {from: 2, to: 1}, {from: 1, to: 0}}
	if got := laneGroups(have, chain); !equalGroups(got, [][]RegionID{{0, 1, 2, 3, 4, 5}}) {
		t.Errorf("laneGroups(%v, a chain from 5 down to 0) = %v, want one component", have, got)
	}
}

// equalGroups compares two lists of region lists.
func equalGroups(a, b [][]RegionID) bool {
	return slices.EqualFunc(a, b, func(x, y []RegionID) bool { return slices.Equal(x, y) })
}

// TestCopyLanes verifies that a list within the limit is copied (a later change
// to the caller's list does not reach it) and that one over the limit is not
// copied, so the bound is enforced before a copy of that size is allocated.
//
// Requirement: FR-VIA-063
func TestCopyLanes(t *testing.T) {
	t.Parallel()
	in := []Lane{{From: "a", To: "b", Commodity: "c"}, {From: "b", To: "a", Commodity: "c"}}
	if got := copyLanes(in, 2); &got[0] == &in[0] || !slices.Equal(got, in) {
		t.Errorf("copyLanes(2 lanes, limit 2) shares memory or differs: %v", got)
	}
	if got := copyLanes(in, 1); &got[0] != &in[0] {
		t.Errorf("copyLanes(2 lanes, limit 1) copied the list, want it left as is for checkLaneList to refuse")
	}
	if got := copyLanes(nil, 0); len(got) != 0 {
		t.Errorf("copyLanes(nil, 0) = %v, want empty", got)
	}
}
