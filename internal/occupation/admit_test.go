package occupation_test

import (
	"cmp"
	"fmt"
	"math/rand/v2"
	"reflect"
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/occupation"
)

func q(roles ...int) []occupation.Dest {
	ds := make([]occupation.Dest, len(roles))
	for i, r := range roles {
		ds[i] = occupation.Dest{Role: r, Qualifies: true}
	}
	return ds
}

func dcs(pairs ...int64) []occupation.DestCount {
	var out []occupation.DestCount
	for i := 0; i+1 < len(pairs); i += 2 {
		out = append(out, occupation.DestCount{Dest: int(pairs[i]), Households: pairs[i+1]})
	}
	return out
}

type admitCase struct {
	name       string
	cands      []occupation.Candidate
	region     int64
	regionUsed int64
	used       []int64
	wantAdm    []occupation.Admission
	wantCounts occupation.Counts
	wantRegion int64
	wantUsed   []int64
}

func admitCapCases() []admitCase {
	return []admitCase{
		{
			name: "reach the cap is admitted, the next is refused by the region",
			// Cap 5, used 4: the weight-3 agent reaches it (4 + 3 = 7); the next finds 7 >= 5.
			cands: []occupation.Candidate{
				{Kind: occupation.CandRung1, Ref: 0, Seq: 1, Households: 3, Dests: q(2)},
				{Kind: occupation.CandRung1, Ref: 1, Seq: 2, Households: 1, Dests: q(2)},
			},
			region: 5, regionUsed: 4, used: []int64{0, 0, 0},
			wantAdm:    []occupation.Admission{{Ref: 0, Admitted: dcs(2, 3)}},
			wantCounts: occupation.Counts{RegionCap: 1},
			wantRegion: 7, wantUsed: []int64{0, 0, 3},
		},
		{
			name: "best destination full takes the next best",
			// Role 0 has cap 1, used 1: full. Role 1 has cap 2, used 0.
			cands:  []occupation.Candidate{{Kind: occupation.CandReview, Ref: 0, Households: 1, Dests: q(0, 1)}},
			region: 5, used: []int64{1, 0, 0},
			wantAdm:    []occupation.Admission{{Ref: 0, Admitted: dcs(1, 1)}},
			wantRegion: 1, wantUsed: []int64{1, 1, 0},
		},
		{
			name: "three full destinations count the households once under RoleCap",
			// The agent has 2 households; roles 0, 1, 2 are at 1, 2, 2 of caps 1, 2, 2.
			cands:  []occupation.Candidate{{Kind: occupation.CandReview, Ref: 0, Households: 2, Dests: q(0, 1, 2)}},
			region: 5, used: []int64{1, 2, 2},
			wantCounts: occupation.Counts{RoleCap: 2},
			wantUsed:   []int64{1, 2, 2},
		},
		{
			name: "region cap is counted before the role cap",
			// Region full and the only destination full: RegionCap, not RoleCap.
			cands:  []occupation.Candidate{{Kind: occupation.CandRung1, Ref: 0, Households: 2, Dests: q(0)}},
			region: 3, regionUsed: 3, used: []int64{1, 0, 0},
			wantCounts: occupation.Counts{RegionCap: 2},
			wantRegion: 3, wantUsed: []int64{1, 0, 0},
		},
		{
			name: "no qualifying destination counts nothing",
			cands: []occupation.Candidate{
				{Kind: occupation.CandRung1, Ref: 0, Households: 2, Dests: []occupation.Dest{{Role: 0}}},
				{Kind: occupation.CandReplace, Ref: 1, Seq: 1, Households: 4},
			},
			region: 5, used: []int64{0, 0, 0},
			wantUsed: []int64{0, 0, 0},
		},
		{
			name: "EntryShare 0 counts every candidate household in RegionCap",
			// Caps all 0: kind 0 (2), kind 1 (1) and a batch (3) are all RegionCap.
			cands: []occupation.Candidate{
				{Kind: occupation.CandRung1, Ref: 0, Households: 2, Dests: q(1)},
				{Kind: occupation.CandReview, Ref: 1, Households: 1, Dests: q(1)},
				{Kind: occupation.CandReplace, Ref: 2, Seq: 2, Households: 3, Dests: q(1)},
			},
			region: 0, used: []int64{0, 0, 0},
			wantCounts: occupation.Counts{RegionCap: 6},
			wantUsed:   []int64{0, 0, 0},
		},
	}
}

func admitOrderCases() []admitCase {
	return []admitCase{
		{
			name: "rung 1 before reviewers before replacements whatever the input order",
			// Same region room 2 and role room: rung 1 (1), review (1) fill it; the batch waits.
			cands: []occupation.Candidate{
				{Kind: occupation.CandReplace, Ref: 7, Seq: 0, Households: 2, Dests: q(1)},
				{Kind: occupation.CandReview, Ref: 5, Gap: 900, Seq: 5, Households: 1, Dests: q(1)},
				{Kind: occupation.CandRung1, Ref: 3, Privation: 1, Seq: 3, Households: 1, Dests: q(1)},
			},
			region: 2, used: []int64{0, 0, 0},
			wantAdm: []occupation.Admission{
				{Ref: 3, Admitted: dcs(1, 1)},
				{Ref: 5, Admitted: dcs(1, 1)},
			},
			wantCounts: occupation.Counts{RegionCap: 2},
			wantRegion: 2, wantUsed: []int64{0, 2, 0},
		},
		{
			name: "rung 1 by privation, then gap, then ID",
			// Role 2 has room 2: privation 9 first, then gap 50 over 40; equal key but Seq: lower ID first.
			cands: []occupation.Candidate{
				{Kind: occupation.CandRung1, Ref: 0, Privation: 8, Gap: 99, Seq: 10, Households: 1, Dests: q(2)},
				{Kind: occupation.CandRung1, Ref: 1, Privation: 9, Gap: 40, Seq: 11, Households: 1, Dests: q(2)},
				{Kind: occupation.CandRung1, Ref: 2, Privation: 9, Gap: 50, Seq: 12, Households: 1, Dests: q(2)},
				{Kind: occupation.CandRung1, Ref: 3, Privation: 9, Gap: 50, Seq: 9, Households: 1, Dests: q(2)},
			},
			region: 9, used: []int64{0, 0, 0},
			wantAdm: []occupation.Admission{
				{Ref: 3, Admitted: dcs(2, 1)},
				{Ref: 2, Admitted: dcs(2, 1)},
			},
			wantCounts: occupation.Counts{RoleCap: 2},
			wantRegion: 2, wantUsed: []int64{0, 0, 2},
		},
		{
			name: "replacement batches in leaver-role order, not by gap",
			// Batch of role 1 (Seq 1) takes the room before role 2's (Seq 2) though listed after.
			cands: []occupation.Candidate{
				{Kind: occupation.CandReplace, Ref: 2, Seq: 2, Gap: 0, Households: 2, Dests: q(1)},
				{Kind: occupation.CandReplace, Ref: 1, Seq: 1, Gap: 0, Households: 2, Dests: q(1)},
			},
			region: 9, used: []int64{0, 0, 0},
			wantAdm:    []occupation.Admission{{Ref: 1, Admitted: dcs(1, 2)}},
			wantCounts: occupation.Counts{RoleCap: 2},
			wantRegion: 2, wantUsed: []int64{0, 2, 0},
		},
		{
			name: "a batch spans several destinations and the rest is RoleCap",
			// Left 7: role 0 room 1, role 1 room 2, role 2 room 2 -> 1, 2, 2; 2 left, region 9 not full.
			cands:  []occupation.Candidate{{Kind: occupation.CandReplace, Ref: 0, Households: 7, Dests: q(0, 1, 2)}},
			region: 9, used: []int64{0, 0, 0},
			wantAdm:    []occupation.Admission{{Ref: 0, Admitted: dcs(0, 1, 1, 2, 2, 2)}},
			wantCounts: occupation.Counts{RoleCap: 2},
			wantRegion: 5, wantUsed: []int64{1, 2, 2},
		},
		{
			name: "a batch cut by the region cap counts the rest in RegionCap",
			// Region room 3 (5 - 2): role 1 takes min(6, 3, 2) = 2, role 2 min(4, 1, 2) = 1; 3 left, region full.
			cands:  []occupation.Candidate{{Kind: occupation.CandReplace, Ref: 0, Households: 6, Dests: q(1, 2)}},
			region: 5, regionUsed: 2, used: []int64{0, 0, 0},
			wantAdm:    []occupation.Admission{{Ref: 0, Admitted: dcs(1, 2, 2, 1)}},
			wantCounts: occupation.Counts{RegionCap: 3},
			wantRegion: 5, wantUsed: []int64{0, 2, 1},
		},
		{
			name: "a negative room after an overshoot admits none",
			// Region cap 2 with 4 used (a weighted agent overshot): the differences are negative, none admitted.
			cands:  []occupation.Candidate{{Kind: occupation.CandReplace, Ref: 0, Households: 3, Dests: q(1)}},
			region: 2, regionUsed: 4, used: []int64{0, 5, 0},
			wantCounts: occupation.Counts{RegionCap: 3},
			wantRegion: 4, wantUsed: []int64{0, 5, 0},
		},
		{
			name: "a batch skips a full destination and the leftover is RoleCap when all are full",
			// Role 1 is over its cap (used 3 of 2): 0 taken; role 2 takes 2; 1 left, region 9 not full.
			cands:  []occupation.Candidate{{Kind: occupation.CandReplace, Ref: 0, Households: 3, Dests: q(1, 2)}},
			region: 9, used: []int64{0, 3, 0},
			wantAdm:    []occupation.Admission{{Ref: 0, Admitted: dcs(2, 2)}},
			wantCounts: occupation.Counts{RoleCap: 1},
			wantRegion: 2, wantUsed: []int64{0, 3, 2},
		},
	}
}

// TestAdmit_Cases verifies the order of need, the caps, the reach-the-cap rule,
// the next-best rule, the closed-form batch and the counts of refused
// households.
//
// The caps follow SPN 15.3: a region of 1 000 households at EntryShare 50 allows
// ceil(1 000 * 50 / 10 000) = 5 changes, a destination of 50 households allows
// 1 and one of 400 allows 2. Role indexes 0, 1, 2 below have caps 1, 2, 2.
//
// Requirement: FR-SPN-021, FR-SPN-022
func TestAdmit_Cases(t *testing.T) {
	t.Parallel()
	tests := append(admitCapCases(), admitOrderCases()...)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			region := tt.regionUsed
			used := slices.Clone(tt.used)
			lim := occupation.Limits{RegionCap: tt.region, RoleCap: []int64{1, 2, 2}}
			if tt.region == 0 {
				lim.RoleCap = []int64{0, 0, 0}
			}
			adm, _, counts := occupation.Admit(nil, nil, tt.cands, lim, &region, used)
			if !reflect.DeepEqual(adm, tt.wantAdm) {
				t.Errorf("Admit admissions = %+v, want %+v", adm, tt.wantAdm)
			}
			if counts != tt.wantCounts {
				t.Errorf("Admit counts = %+v, want %+v", counts, tt.wantCounts)
			}
			if region != tt.wantRegion {
				t.Errorf("Admit region used = %d, want %d", region, tt.wantRegion)
			}
			if !slices.Equal(used, tt.wantUsed) {
				t.Errorf("Admit used = %v, want %v", used, tt.wantUsed)
			}
		})
	}
}

// TestAdmit_SecondCallContinuesTheCounts verifies that a second call (the
// cross-role replacements) continues the counts of the first (the switches).
//
// Region cap 3 and role 0 cap 3: the first call admits 2 households of a
// switcher; the second call's batch of 4 finds a region room of 3 - 2 = 1 and
// takes 1, counting 3 in RegionCap.
//
// Requirement: FR-SPN-021
func TestAdmit_SecondCallContinuesTheCounts(t *testing.T) {
	t.Parallel()
	lim := occupation.Limits{RegionCap: 3, RoleCap: []int64{3}}
	var region int64
	used := []int64{0}
	_, _, c1 := occupation.Admit(nil, nil, []occupation.Candidate{{Kind: occupation.CandRung1, Households: 2, Dests: q(0)}}, lim, &region, used)
	adm, _, c2 := occupation.Admit(nil, nil, []occupation.Candidate{{Kind: occupation.CandReplace, Ref: 4, Households: 4, Dests: q(0)}}, lim, &region, used)
	if c1 != (occupation.Counts{}) || c2 != (occupation.Counts{RegionCap: 3}) {
		t.Errorf("Admit counts = %+v then %+v, want zero then {RegionCap:3}", c1, c2)
	}
	if want := []occupation.Admission{{Ref: 4, Admitted: dcs(0, 1)}}; !reflect.DeepEqual(adm, want) {
		t.Errorf("Admit second call = %+v, want %+v", adm, want)
	}
	if region != 3 || used[0] != 3 {
		t.Errorf("Admit state = region %d, used %v, want 3, [3]", region, used)
	}
}

// TestAdmit_AllocatesNothing verifies that Admit allocates nothing when the
// caller supplies slices of enough capacity.
//
// Guard: Decide calls Admit per region per tick and must allocate nothing with
// the world's scratch (NFR-SPN-001). A closure in the sort or a slice built per
// candidate would allocate per call.
//
// Requirement: NFR-SPN-001
func TestAdmit_AllocatesNothing(t *testing.T) {
	cs := []occupation.Candidate{
		{Kind: occupation.CandReplace, Ref: 2, Seq: 2, Households: 5, Dests: q(0, 1)},
		{Kind: occupation.CandRung1, Ref: 0, Households: 1, Dests: q(1)},
		{Kind: occupation.CandReview, Ref: 1, Households: 2, Dests: q(0)},
	}
	adm := make([]occupation.Admission, 0, 3)
	dc := make([]occupation.DestCount, 0, 8)
	lim := occupation.Limits{RegionCap: 100, RoleCap: []int64{2, 2}}
	used := make([]int64, 2)
	allocs := testing.AllocsPerRun(100, func() {
		var region int64
		clear(used)
		_, _, _ = occupation.Admit(adm, dc, cs, lim, &region, used)
	})
	if allocs != 0 {
		t.Errorf("Admit allocates %v times per call, want 0", allocs)
	}
}

// refCand is one candidate of the per-household oracle.
type refCand struct {
	kind                                  int
	ref                                   int
	privation, gap, seq, seq2, households int64
	dests                                 []occupation.Dest
}

// expandRef turns each kind-2 batch into weight-1 candidates, the one-at-a-time
// form of the rule.
func expandRef(cs []occupation.Candidate) []refCand {
	var xs []refCand
	for _, c := range cs {
		if c.Kind != occupation.CandReplace {
			xs = append(xs, refCand{c.Kind, c.Ref, c.Privation, c.Gap, c.Seq, c.Seq2, c.Households, c.Dests})
			continue
		}
		for i := range c.Households {
			xs = append(xs, refCand{c.Kind, c.Ref, c.Privation, c.Gap, c.Seq, c.Seq2 + i, 1, c.Dests})
		}
	}
	return xs
}

// oracleAdmit is the per-household reference of Admit written from the plain
// rule of spec-spawner.md 6.3.0: each kind-2 batch is expanded to weight-1
// candidates, all are sorted by the key and admitted one at a time. It shares no
// code with Admit. It returns the households admitted per (ref, role), the final
// state and the counts.
func oracleAdmit(cs []occupation.Candidate, lim occupation.Limits, regionUsed int64, used []int64) (map[[2]int]int64, int64, []int64, occupation.Counts) {
	xs := expandRef(cs)
	slices.SortFunc(xs, func(a, b refCand) int {
		return cmp.Or(cmp.Compare(a.kind, b.kind), cmp.Compare(b.privation, a.privation),
			cmp.Compare(b.gap, a.gap), cmp.Compare(a.seq, b.seq), cmp.Compare(a.seq2, b.seq2), cmp.Compare(a.ref, b.ref))
	})
	got := map[[2]int]int64{}
	used = slices.Clone(used)
	var counts occupation.Counts
	for _, x := range xs {
		if regionUsed >= lim.RegionCap {
			counts.RegionCap += x.households
			continue
		}
		anyQ, placed := false, false
		for _, d := range x.dests {
			if !d.Qualifies {
				continue
			}
			anyQ = true
			if used[d.Role] < lim.RoleCap[d.Role] {
				used[d.Role] += x.households
				regionUsed += x.households
				got[[2]int{x.ref, d.Role}] += x.households
				placed = true
				break
			}
		}
		if !placed && anyQ {
			counts.RoleCap += x.households
		}
	}
	return got, regionUsed, used, counts
}

// genAdmitInput builds a random input from the seed: caps that include 0 and
// values around the demand, equal gaps and privations to force ties, used counts
// that may already exceed a cap, and unique Refs (agent index for kinds 0 and 1,
// 100 + leaver role for kind 2).
func genAdmitInput(r *rand.Rand) ([]occupation.Candidate, occupation.Limits, int64, []int64) {
	nRoles := 1 + r.IntN(4)
	lim := occupation.Limits{RegionCap: int64(r.IntN(12)), RoleCap: make([]int64, nRoles)}
	used := make([]int64, nRoles)
	for i := range lim.RoleCap {
		lim.RoleCap[i] = int64(r.IntN(7))
		used[i] = int64(r.IntN(9))
		if r.IntN(2) == 0 {
			used[i] = 0
		}
	}
	regionUsed := int64(r.IntN(10))
	if r.IntN(2) == 0 {
		regionUsed = 0
	}
	dests := func() []occupation.Dest {
		var ds []occupation.Dest
		for _, i := range r.Perm(nRoles)[:1+r.IntN(nRoles)] {
			ds = append(ds, occupation.Dest{Role: i, Qualifies: r.IntN(4) != 0})
		}
		return ds
	}
	var cs []occupation.Candidate
	for i := range r.IntN(7) {
		kind := r.IntN(2)
		c := occupation.Candidate{Kind: kind, Ref: i, Seq: int64(i), Households: 1 + int64(r.IntN(3)), Gap: int64(r.IntN(3)), Dests: dests()}
		if kind == occupation.CandRung1 {
			c.Privation = int64(r.IntN(3))
		}
		cs = append(cs, c)
	}
	for i := range r.IntN(4) {
		cs = append(cs, occupation.Candidate{Kind: occupation.CandReplace, Ref: 100 + i, Seq: int64(i), Households: 1 + int64(r.IntN(9)), Dests: dests()})
	}
	r.Shuffle(len(cs), func(i, j int) { cs[i], cs[j] = cs[j], cs[i] })
	return cs, lim, regionUsed, used
}

func admitTotals(adm []occupation.Admission) (map[[2]int]int64, int64) {
	got := map[[2]int]int64{}
	var sum int64
	for _, a := range adm {
		for _, d := range a.Admitted {
			got[[2]int{a.Ref, d.Dest}] += d.Households
			sum += d.Households
		}
	}
	return got, sum
}

// checkAdmit runs Admit on the input of the seed and returns a description of
// the first property that fails, or "": the same admissions per destination,
// the same remaining caps and the same counts as the per-household oracle; the
// same result for a permuted input; no cap exceeded by more than the largest
// whole candidate less one household; and every candidate household admitted or
// counted once (or in neither when it had no qualifying destination).
func checkAdmit(seed1, seed2 uint64) string {
	r := rand.New(rand.NewPCG(seed1, seed2))
	cs, lim, regionUsed, used0 := genAdmitInput(r)
	wantGot, wantRegion, wantUsed, wantCounts := oracleAdmit(cs, lim, regionUsed, used0)

	run := func(in []occupation.Candidate) (map[[2]int]int64, int64, int64, []int64, occupation.Counts) {
		in = slices.Clone(in)
		region, used := regionUsed, slices.Clone(used0)
		adm, _, counts := occupation.Admit(nil, nil, in, lim, &region, used)
		got, sum := admitTotals(adm)
		return got, sum, region, used, counts
	}
	got, sum, region, used, counts := run(cs)
	if !reflect.DeepEqual(got, wantGot) || region != wantRegion || !slices.Equal(used, wantUsed) || counts != wantCounts {
		return fmt.Sprintf("closed form %v region %d used %v counts %+v, oracle %v region %d used %v counts %+v (input %+v lim %+v region %d used %v)",
			got, region, used, counts, wantGot, wantRegion, wantUsed, wantCounts, cs, lim, regionUsed, used0)
	}
	shuffled := slices.Clone(cs)
	r.Shuffle(len(shuffled), func(i, j int) { shuffled[i], shuffled[j] = shuffled[j], shuffled[i] })
	if g2, _, r2, u2, c2 := run(shuffled); !reflect.DeepEqual(g2, got) || r2 != region || !slices.Equal(u2, used) || c2 != counts {
		return "permuting the input changed the result"
	}
	return checkBounds(cs, lim, regionUsed, used0, sum, region, used, counts)
}

// checkBounds is the accounting and cap-bound half of checkAdmit.
func checkBounds(cs []occupation.Candidate, lim occupation.Limits, regionUsed int64, used0 []int64, sum, region int64, used []int64, counts occupation.Counts) string {
	var total, allQ, maxWhole int64
	allQ = 1
	for _, c := range cs {
		total += c.Households
		if c.Kind != occupation.CandReplace {
			maxWhole = max(maxWhole, c.Households)
		}
		if !slices.ContainsFunc(c.Dests, func(d occupation.Dest) bool { return d.Qualifies }) {
			allQ = 0
		}
	}
	if sum+counts.RegionCap+counts.RoleCap > total || (allQ == 1 && sum+counts.RegionCap+counts.RoleCap != total) {
		return fmt.Sprintf("households admitted %d + counted %+v do not account for %d", sum, counts, total)
	}
	if region > max(regionUsed, lim.RegionCap+max(maxWhole-1, 0)) {
		return fmt.Sprintf("region used %d exceeds cap %d by more than a candidate", region, lim.RegionCap)
	}
	for i := range used {
		if used[i] > max(used0[i], lim.RoleCap[i]+max(maxWhole-1, 0)) {
			return fmt.Sprintf("role %d used %d exceeds cap %d by more than a candidate", i, used[i], lim.RoleCap[i])
		}
	}
	return ""
}

// TestAdmit_MatchesPerHouseholdOracle verifies, over 200 seeds, that the
// closed-form Admit gives the admissions per destination, the remaining caps
// and the RegionCap and RoleCap counts of a per-household reference, that the
// result does not depend on the order of the input, that a cap is exceeded by
// at most one candidate and that every candidate household is admitted or
// counted exactly once.
//
// The generator covers caps of 0, caps equal to, one below and one above the
// demand, equal gaps and privations, ties broken only by Seq, destinations that
// do not qualify, counts that already exceed a cap, and batches that span
// several destinations.
//
// Guard: the kind-2 closed form replaces an O(households) loop (O9). If its
// min/max arithmetic drifted from the one-at-a-time rule, only this comparison
// would show it, and only as a wrong plan far from the cause.
//
// Requirement: FR-SPN-021, NFR-SPN-001, NFR-SPN-002
func TestAdmit_MatchesPerHouseholdOracle(t *testing.T) {
	t.Parallel()
	for seed := range uint64(200) {
		if msg := checkAdmit(seed, seed*7+1); msg != "" {
			t.Fatalf("seed %d: %s", seed, msg)
		}
	}
}

// FuzzAdmit verifies the properties of TestAdmit_MatchesPerHouseholdOracle on
// inputs generated from two fuzzed seeds: no panic, the oracle comparison,
// permutation invariance, the cap bound and the accounting of households.
//
// Requirement: FR-SPN-021
func FuzzAdmit(f *testing.F) {
	// Seeds: the first generated inputs, and a high pair.
	for i := range uint64(8) {
		f.Add(i, i*7+1)
	}
	f.Add(^uint64(0), uint64(12345))
	f.Fuzz(func(t *testing.T, seed1, seed2 uint64) {
		if msg := checkAdmit(seed1, seed2); msg != "" {
			t.Fatalf("seeds %d, %d: %s", seed1, seed2, msg)
		}
	})
}
