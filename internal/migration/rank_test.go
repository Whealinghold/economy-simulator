package migration_test

import (
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/migration"
)

// riverParams are the figures of MIG 15.1 for agent 1042: horizon 180, entry
// ratio 8 000, home figure 6 393 and margin 5 957.
var riverParams = migration.Params{Horizon: 180, EntryRatio: 8_000, Home: 6_393, Margin: 5_957}

// TestRank_StageOfLatestReached verifies the four stages and the outcome of a
// household that no candidate survives: it is the latest stage at which any
// candidate failed, a candidate's stage being the first test it fails, with no
// candidate NoDestination and a role with no recipe NoEstimate.
//
// Requirement: FR-MIG-017, FR-MIG-018
func TestRank_StageOfLatestReached(t *testing.T) {
	t.Parallel()
	// With V_a 6 393 and g 5 957 a candidate needs V_eff of 10 202. Vhat 9 000 at
	// k 2 is viable (>= 8 000) and gives V_eff = 9 000 * 178 / 180 = 8 900: stage 4.
	noGain := migration.Candidate{Ref: 1, Priced: true, K: 2, Vhat: 9_000}
	notViable := migration.Candidate{Ref: 2, Priced: true, K: 2, Vhat: 7_999}
	tooFar := migration.Candidate{Ref: 3, Priced: true, K: 180, Vhat: 50_000}
	wayTooFar := migration.Candidate{Ref: 6, Priced: true, K: 181, Vhat: 50_000}
	// Vhat at the entry ratio is viable, so it falls at stage 4 and not at stage 3:
	// 8 000 * 179 / 180 = 7 955, short of 10 202.
	atEntry := migration.Candidate{Ref: 7, Priced: true, K: 1, Vhat: 8_000}
	unpriced := migration.Candidate{Ref: 4, Priced: false, K: 2, Vhat: 50_000}
	good := migration.Candidate{Ref: 5, Priced: true, K: 2, Vhat: 14_376}
	tests := []struct {
		name        string
		cands       []migration.Candidate
		noEstimate  bool
		wantOutcome migration.Outcome
		wantRefs    []int
	}{
		{"no candidate", nil, false, migration.NoDestination, nil},
		{"no recipe", []migration.Candidate{good}, true, migration.NoEstimate, nil},
		{"only unpriced", []migration.Candidate{unpriced}, false, migration.NoRumour, nil},
		{"only too far", []migration.Candidate{tooFar}, false, migration.TooFar, nil},
		{"only way too far", []migration.Candidate{wayTooFar}, false, migration.TooFar, nil},
		{"at the entry ratio is viable", []migration.Candidate{atEntry}, false, migration.NoGain, nil},
		{"only not viable", []migration.Candidate{notViable}, false, migration.NoViable, nil},
		{"only no gain", []migration.Candidate{noGain}, false, migration.NoGain, nil},
		{"too far and no gain is no gain", []migration.Candidate{tooFar, noGain}, false, migration.NoGain, nil},
		{"no gain and unpriced is no gain", []migration.Candidate{noGain, unpriced}, false, migration.NoGain, nil},
		{"not viable and too far is not viable", []migration.Candidate{tooFar, notViable, unpriced}, false, migration.NoViable, nil},
		{"unpriced and too far is too far", []migration.Candidate{unpriced, tooFar}, false, migration.TooFar, nil},
		{"one survivor among failures", []migration.Candidate{noGain, good, tooFar}, false, migration.Started, []int{5}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			p := riverParams
			p.NoEstimate = tt.noEstimate
			ranked, outcome, ok := migration.Rank(nil, tt.cands, p)
			var refs []int
			for _, r := range ranked {
				refs = append(refs, r.Ref)
			}
			if !ok || outcome != tt.wantOutcome || !slices.Equal(refs, tt.wantRefs) {
				t.Errorf("Rank = refs %v, outcome %d, ok %v, want refs %v, outcome %d, true", refs, outcome, ok, tt.wantRefs, tt.wantOutcome)
			}
		})
	}
}

// TestRank_WorkedExampleA verifies the ranking of MIG 15.1: both destinations
// pass, lower (17 043) before middle (14 216), with the V_eff values and the
// journey times carried.
//
// Requirement: FR-MIG-017
func TestRank_WorkedExampleA(t *testing.T) {
	t.Parallel()
	cands := []migration.Candidate{
		{Ref: 1, Priced: true, K: 2, Vhat: 14_376}, // middle
		{Ref: 0, Priced: true, K: 4, Vhat: 17_431}, // lower
	}
	ranked, outcome, ok := migration.Rank(nil, cands, riverParams)
	want := []migration.Ranked{{Ref: 0, K: 4, VEff: 17_043}, {Ref: 1, K: 2, VEff: 14_216}}
	if !ok || outcome != migration.Started || !slices.Equal(ranked, want) {
		t.Errorf("Rank = %v, %d, %v, want %v, Started, true", ranked, outcome, ok, want)
	}
}

// TestRank_TieByRegion verifies that equal V_eff are ordered by the candidate's
// reference ascending whatever the input order, so the ranking is a total order
// that depends on the estimates and the IDs only.
//
// Requirement: FR-MIG-017
func TestRank_TieByRegion(t *testing.T) {
	t.Parallel()
	mk := func(refs ...int) []migration.Candidate {
		var cs []migration.Candidate
		for _, r := range refs {
			cs = append(cs, migration.Candidate{Ref: r, Priced: true, K: 1, Vhat: 20_000})
		}
		return cs
	}
	for _, in := range [][]int{{0, 1, 2, 3}, {3, 2, 1, 0}, {2, 0, 3, 1}} {
		ranked, _, ok := migration.Rank(nil, mk(in...), riverParams)
		var got []int
		for _, r := range ranked {
			got = append(got, r.Ref)
		}
		if !ok || !slices.Equal(got, []int{0, 1, 2, 3}) {
			t.Errorf("Rank of refs %v = %v, %v, want [0 1 2 3], true", in, got, ok)
		}
	}
}

// TestRank_ReusesDst verifies that the ranking is built in the caller's buffer,
// whatever it held, so a pass allocates nothing once the buffer has grown.
//
// Requirement: NFR-MIG-001
func TestRank_ReusesDst(t *testing.T) {
	t.Parallel()
	buf := make([]migration.Ranked, 3, 8)
	for _, cands := range [][]migration.Candidate{
		{{Ref: 0, Priced: true, K: 1, Vhat: 20_000}},
		nil,
		{{Ref: 0, Priced: false}},
	} {
		got, _, ok := migration.Rank(buf, cands, riverParams)
		if !ok || len(got) > 1 || (len(got) == 1 && &got[0] != &buf[0]) {
			t.Errorf("Rank over %v returned %v (len %d), want at most the one entry, in the buffer", cands, got, len(got))
		}
		if len(got) == 0 && cap(got) != cap(buf) {
			t.Errorf("an empty ranking has capacity %d, want the buffer's %d", cap(got), cap(buf))
		}
	}
}

// TestOutcomes_Eleven verifies the eleven outcomes of MIG 7.6 are distinct and
// counted.
//
// Requirement: FR-MIG-025
func TestOutcomes_Eleven(t *testing.T) {
	t.Parallel()
	all := []migration.Outcome{migration.Started, migration.NoEstimate, migration.NoDestination, migration.NoRumour, migration.TooFar,
		migration.NoViable, migration.NoGain, migration.Bound, migration.OutCap, migration.Unprovisioned, migration.InCap}
	if migration.Outcomes != 11 || len(all) != 11 {
		t.Fatalf("Outcomes = %d over %d listed, want 11", migration.Outcomes, len(all))
	}
	for i, o := range all {
		if int(o) != i {
			t.Errorf("outcome %d has value %d, want its position in the table", i, o)
		}
	}
}

// TestRank_OverflowIsNoGain verifies that a product past int64 is reported with
// ok false and the household is returned as NoGain with no ranking, never
// wrapped.
//
// Requirement: NFR-MIG-003
func TestRank_OverflowIsNoGain(t *testing.T) {
	t.Parallel()
	p := riverParams
	p.Home = 1 << 62
	p.Margin = 1 << 62 // 10 000 + margin is fine; Home * factor overflows the quotient
	ranked, outcome, ok := migration.Rank(nil, []migration.Candidate{{Ref: 0, Priced: true, K: 1, Vhat: 1 << 40}}, p)
	if ok || outcome != migration.NoGain || len(ranked) != 0 {
		t.Errorf("Rank = %v, %d, %v, want no ranking, NoGain, false", ranked, outcome, ok)
	}
	// A negative journey time cannot be discounted: reported, with no ranking.
	ranked, outcome, ok = migration.Rank(nil, []migration.Candidate{{Ref: 0, Priced: true, K: -1, Vhat: 20_000}}, riverParams)
	if ok || outcome != migration.NoGain || len(ranked) != 0 {
		t.Errorf("Rank with K -1 = %v, %d, %v, want no ranking, NoGain, false", ranked, outcome, ok)
	}
}
