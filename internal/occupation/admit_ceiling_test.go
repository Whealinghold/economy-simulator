package occupation_test

import (
	"reflect"
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/occupation"
)

// TestOccupation_CeilingCapRefusesWholeCandidate verifies that a role ceiling
// admits a candidate only if all its households fit the room, unlike RoleCap,
// which the candidate that reaches it may exceed.
//
// Arithmetic: role 0 has room 1 and a RoleCap of 5. A weight-2 candidate does not
// fit (2 > 1): refused and counted in Ceiling 2, the room untouched. A weight-1
// candidate fits (1 <= 1): admitted, room 0. A batch of 3 into a role of room 2
// takes 2 and counts 1 in Ceiling; with room negative (no ceiling) it takes 3.
//
// Requirement: FR-SET-023, FR-SPN-056
func TestOccupation_CeilingCapRefusesWholeCandidate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		cand       occupation.Candidate
		room       int64
		wantAdm    []occupation.Admission
		wantCounts occupation.Counts
		wantRoom   int64
	}{
		{"weight 2 against room 1 is refused whole", occupation.Candidate{Kind: occupation.CandRung1, Households: 2, Dests: q(0)}, 1, nil, occupation.Counts{Ceiling: 2}, 1},
		{"weight 1 against room 1 is admitted", occupation.Candidate{Kind: occupation.CandRung1, Households: 1, Dests: q(0)}, 1, []occupation.Admission{{Ref: 0, Admitted: dcs(0, 1)}}, occupation.Counts{}, 0},
		{"weight 2 against room 2 is admitted", occupation.Candidate{Kind: occupation.CandReview, Households: 2, Dests: q(0)}, 2, []occupation.Admission{{Ref: 0, Admitted: dcs(0, 2)}}, occupation.Counts{}, 0},
		{"room 0 is a closed gate", occupation.Candidate{Kind: occupation.CandRung1, Households: 1, Dests: q(0)}, 0, nil, occupation.Counts{Ceiling: 1}, 0},
		{"no ceiling leaves the RoleCap rule: weight 2 against cap 5 is admitted", occupation.Candidate{Kind: occupation.CandRung1, Households: 2, Dests: q(0)}, -1, []occupation.Admission{{Ref: 0, Admitted: dcs(0, 2)}}, occupation.Counts{}, -1},
		{"a batch takes the room and counts the rest", occupation.Candidate{Kind: occupation.CandReplace, Households: 3, Dests: q(0)}, 2, []occupation.Admission{{Ref: 0, Admitted: dcs(0, 2)}}, occupation.Counts{Ceiling: 1}, 0},
		{"a batch with no ceiling takes all", occupation.Candidate{Kind: occupation.CandReplace, Households: 3, Dests: q(0)}, -1, []occupation.Admission{{Ref: 0, Admitted: dcs(0, 3)}}, occupation.Counts{}, -1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var region int64
			lim := occupation.Limits{RegionCap: 10, RoleCap: []int64{5}, Room: []int64{tt.room}}
			adm, _, counts := occupation.Admit(nil, nil, []occupation.Candidate{tt.cand}, lim, &region, []int64{0})
			if !reflect.DeepEqual(adm, tt.wantAdm) {
				t.Errorf("Admit admissions = %+v, want %+v", adm, tt.wantAdm)
			}
			if counts != tt.wantCounts {
				t.Errorf("Admit counts = %+v, want %+v", counts, tt.wantCounts)
			}
			if lim.Room[0] != tt.wantRoom {
				t.Errorf("Admit left room %d, want %d", lim.Room[0], tt.wantRoom)
			}
		})
	}
}

// TestOccupation_CeilingNextDestination verifies that a candidate refused by the
// ceiling of its best destination takes the next qualifying one, and that the
// ceiling wins over RoleCap in the count when both applied.
//
// Arithmetic: destination 0 has room 0 and destination 1 room 3, a weight-2
// candidate goes to 1 (room 1 left). A second weight-2 candidate fits neither
// (0 < 2, 1 < 2) and counts 2 in Ceiling although destination 1 also has RoleCap
// left: the ceiling is the binding rule.
//
// Requirement: FR-SET-023, FR-SPN-056
func TestOccupation_CeilingNextDestination(t *testing.T) {
	t.Parallel()
	var region int64
	used := []int64{0, 0}
	lim := occupation.Limits{RegionCap: 10, RoleCap: []int64{5, 5}, Room: []int64{0, 3}}
	cs := []occupation.Candidate{
		{Kind: occupation.CandRung1, Seq: 1, Ref: 0, Households: 2, Dests: q(0, 1)},
		{Kind: occupation.CandRung1, Seq: 2, Ref: 1, Households: 2, Dests: q(0, 1)},
	}
	adm, _, counts := occupation.Admit(nil, nil, cs, lim, &region, used)
	if want := []occupation.Admission{{Ref: 0, Admitted: dcs(1, 2)}}; !reflect.DeepEqual(adm, want) {
		t.Errorf("Admit admissions = %+v, want %+v", adm, want)
	}
	if counts != (occupation.Counts{Ceiling: 2}) {
		t.Errorf("Admit counts = %+v, want {Ceiling:2}", counts)
	}
	if want := []int64{0, 1}; !slices.Equal(lim.Room, want) {
		t.Errorf("Admit left room %v, want %v", lim.Room, want)
	}
}

// TestFit verifies the split of an entry into the part under a ceiling and the
// part it cuts.
//
// Arithmetic: 5 into room 3 takes 3 and cuts 2; room 0 cuts all; a negative room
// is no ceiling; a negative n counts as 0.
//
// Requirement: FR-SET-023
func TestFit(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct{ n, room, take, cut int64 }{
		{5, 3, 3, 2}, {5, 0, 0, 5}, {5, 9, 5, 0}, {5, -1, 5, 0}, {-3, 2, 0, 0}, {0, 0, 0, 0},
	} {
		if take, cut := occupation.Fit(tt.n, tt.room); take != tt.take || cut != tt.cut {
			t.Errorf("Fit(%d, %d) = %d, %d, want %d, %d", tt.n, tt.room, take, cut, tt.take, tt.cut)
		}
	}
}
