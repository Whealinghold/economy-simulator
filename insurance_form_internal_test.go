package economysim

import (
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"testing"
)

// This file tests the order, the matching and the formation of the cover round (plan 3-ins
// Phase 5, spec-insurance.md 7.2 to 7.4) and what the round must not do: take a word of the
// stream, read a shipment's loss share, branch on an identifier or allocate.

// agentOfRole returns the first live agent of a role in a region.
func agentOfRole(t *testing.T, w *World, region RegionID, role string) AgentID {
	t.Helper()
	for i := range w.agents {
		if a := &w.agents[i]; a.weight > 0 && w.regionOf(a) == region && w.cfg.Roles[a.role].ID == role {
			return a.id
		}
	}
	t.Fatalf("no agent of role %q in region %d", role, region)
	return -1
}

// roundOf runs a world's round for a planted request and offer and returns its CoverRound
// of region home, the zero event when none was emitted.
func roundOf(t *testing.T, w *World, rec *evLog) CoverRound {
	t.Helper()
	w.p33cCoverRound()
	_, rounds := coverEvents(rec)
	if len(rounds) == 0 {
		return CoverRound{}
	}
	return rounds[len(rounds)-1]
}

// TestForm_FailuresAreCountedAndChangeNothing verifies the reasons a candidate is not
// formed (7.4 and 7.2.2), each counted in its field of CoverRound with the sum left
// unfilled, and that a failed fill moves no money and forms no policy (FR-INS-036).
//
// Worked: 8 grain is a sum of 8 000 and, at the offer of the tests, a premium of 1 600.
// MinSum 9 000 is above the sum (below_minimum); a loading of 0 makes the premium 0, below 1
// (below_minimum); an insured left with 100 cannot pay 1 600 (funds); a request that accepts
// a loading of at most 5 000 finds an offer at 10 000 above its limit, and one that wants
// 9 000 bp cover finds an offer of 8 000 below it (no_offer).
//
// Requirement: FR-INS-024, FR-INS-034, FR-INS-036
func TestForm_FailuresAreCountedAndChangeNothing(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		mut   func(c *Config)
		lane  LaneCover
		offer CoverOffer
		pay   int64 // the insured's money, 0 for what it holds
		want  CoverRound
	}{
		{"sum below the minimum", func(c *Config) { c.Modules.Insurance.MinSum = 9_000 }, coachLane, CoverOffer{MaxLiability: 10_000, LoadingBP: 10_000, CoverBP: 8_000}, 0,
			CoverRound{BelowMinimum: 1, Unfilled: 8_000}},
		{"premium below 1", nil, coachLane, CoverOffer{MaxLiability: 10_000, LoadingBP: 0, CoverBP: 8_000}, 0,
			CoverRound{BelowMinimum: 1, Unfilled: 8_000}},
		{"funds", nil, coachLane, fairOffer, 100, CoverRound{Funds: 1, Unfilled: 8_000}},
		{"no offer at the price", nil, LaneCover{Mode: modeCoach, To: regFar, MinCoverBP: 5_000, MaxLoadingBP: 5_000}, fairOffer, 0,
			CoverRound{NoOffer: 1, Unfilled: 8_000}},
		{"no offer with the cover share", nil, LaneCover{Mode: modeCoach, To: regFar, MinCoverBP: 9_000, MaxLoadingBP: 20_000}, fairOffer, 0,
			CoverRound{NoOffer: 1, Unfilled: 8_000}},
		{"no offer with the room", nil, coachLane, CoverOffer{MaxLiability: 6_399, LoadingBP: 10_000, CoverBP: 8_000}, 0,
			CoverRound{NoOffer: 1, Unfilled: 8_000}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := insConfig(defaultInsure(), defaultUnderwrite())
			if tt.mut != nil {
				tt.mut(&c)
			}
			w, rec := newInsWorld(t, c, nil)
			insDispatch(t, w, insHome, 8)
			if tt.pay > 0 {
				setBalances(t, w, map[AgentID]int64{insHome: tt.pay, insBacker: 10_000})
			}
			planCover(w, insHome, tt.lane)
			planOffer(w, insBacker, tt.offer)
			insured, under := heldMoney(t, w, insHome), heldMoney(t, w, insBacker)
			got := roundOf(t, w, rec)
			tt.want.Region, tt.want.Requests, tt.want.Offers, tt.want.Candidates = regHome, 1, 1, 1
			if got != tt.want {
				t.Errorf("CoverRound = %+v, want %+v", got, tt.want)
			}
			if formed, _ := coverEvents(rec); len(formed) != 0 || len(w.insurance.open) != 0 {
				t.Errorf("formed %d events and %d policies, want none", len(formed), len(w.insurance.open))
			}
			if heldMoney(t, w, insHome) != insured || heldMoney(t, w, insBacker) != under {
				t.Error("a fill that failed moved money, want nothing moved")
			}
			auditNow(t, w)
		})
	}
}

// TestForm_NoRiskLaneSkipped verifies that a candidate on a lane whose expected loss is 0
// is counted NoRisk and not scanned: the foot lane has no risk row here.
//
// Requirement: FR-INS-024
func TestForm_NoRiskLaneSkipped(t *testing.T) {
	t.Parallel()
	w, rec := newInsWorld(t, insConfig(defaultInsure(), defaultUnderwrite()), nil)
	if _, reason := ask(w, insHome, 0, ShipmentRequest{Mode: modeFoot, To: regFar, ToSelf: true}); reason != "" {
		t.Fatalf("trip = %q, want dispatched", reason)
	}
	planCover(w, insHome, LaneCover{Mode: modeFoot, To: regFar, MinCoverBP: 1, MaxLoadingBP: 20_000})
	planOffer(w, insBacker, fairOffer)
	want := CoverRound{Region: regHome, Requests: 1, Offers: 1, Candidates: 1, NoRisk: 1}
	if got := roundOf(t, w, rec); got != want {
		t.Errorf("CoverRound = %+v, want %+v", got, want)
	}
}

// TestRound_NoShipmentIsCountedAndCostsNothing verifies that a lane request with no shipment
// of the tick on its lane is counted NoShipment and nothing else (spec-insurance.md 5.1.2).
//
// Requirement: FR-INS-031
func TestRound_NoShipmentIsCountedAndCostsNothing(t *testing.T) {
	t.Parallel()
	w, rec := newInsWorld(t, insConfig(defaultInsure(), defaultUnderwrite()), nil)
	planCover(w, insHome, coachLane)
	planOffer(w, insBacker, fairOffer)
	want := CoverRound{Region: regHome, Requests: 1, Offers: 1, NoShipment: 1}
	if got := roundOf(t, w, rec); got != want {
		t.Errorf("CoverRound = %+v, want %+v", got, want)
	}
}

// TestRound_TooManyPoliciesEndsThePass verifies that the policies limit ends the pass: with
// MaxPolicies 1 the first of two shipments is covered, the second is not and TooMany is 1,
// and no policy is formed beyond the limit.
//
// Requirement: FR-INS-034, FR-INS-036
func TestRound_TooManyPoliciesEndsThePass(t *testing.T) {
	t.Parallel()
	c := insConfig(defaultInsure(), defaultUnderwrite())
	c.Modules.Insurance.MaxPolicies = 1
	w, rec := newInsWorld(t, c, nil)
	insDispatch(t, w, insHome, 8)
	insDispatch(t, w, insHome2, 8)
	planCover(w, insHome, coachLane)
	planCover(w, insHome2, coachLane)
	planOffer(w, insBacker, CoverOffer{MaxLiability: 20_000, LoadingBP: 10_000, CoverBP: 8_000})
	got := roundOf(t, w, rec)
	want := CoverRound{Region: regHome, Requests: 2, Offers: 1, Candidates: 2, Formed: 1, Sum: 8_000, Premium: 1_600, Unfilled: 8_000, TooMany: 1}
	if got != want || len(w.insurance.open) != 1 {
		t.Errorf("CoverRound = %+v with %d open policies, want %+v and 1", got, len(w.insurance.open), want)
	}
}

// TestOrder_RationedByWillingnessToPay verifies the order of the requests of a lane: the
// highest limit is served first, so when capacity binds the others are left uncovered
// (spec-insurance.md 7.3.3). The tie by key and agent is the unit test of internal/insure.
//
// Worked: the offer's room 6 400 is one maximum payout (80% of 8 000). The request of
// insHome2 accepts a loading of 15 000 and insHome's 12 000, so insHome2 is covered.
//
// Requirement: FR-INS-032, FR-INS-034
func TestOrder_RationedByWillingnessToPay(t *testing.T) {
	t.Parallel()
	w, rec := newInsWorld(t, insConfig(defaultInsure(), defaultUnderwrite()), nil)
	insDispatch(t, w, insHome, 8)
	insDispatch(t, w, insHome2, 8)
	planCover(w, insHome, LaneCover{Mode: modeCoach, To: regFar, MinCoverBP: 5_000, MaxLoadingBP: 12_000})
	planCover(w, insHome2, LaneCover{Mode: modeCoach, To: regFar, MinCoverBP: 5_000, MaxLoadingBP: 15_000})
	planOffer(w, insBacker, CoverOffer{MaxLiability: 6_400, LoadingBP: 10_000, CoverBP: 8_000})
	got := roundOf(t, w, rec)
	formed, _ := coverEvents(rec)
	if len(formed) != 1 || formed[0].Insured != insHome2 || got.NoOffer != 1 || got.Unfilled != 8_000 {
		t.Errorf("formed %+v, round %+v, want the policy of insHome2 and one candidate left with no offer, 8000 unfilled", formed, got)
	}
}

// TestRound_LaneIsRegion verifies that a request is matched only with offers of its own
// region (spec-insurance.md 7.2.1): an underwriter of far offers, and the insured of home
// is left with no offer.
//
// Requirement: FR-INS-031
func TestRound_LaneIsRegion(t *testing.T) {
	t.Parallel()
	c := insConfig(defaultInsure(), defaultUnderwrite())
	c.Groups = append(c.Groups, Group{Region: "b_far", Role: "backer", Households: 1})
	w, rec := newInsWorld(t, c, nil)
	farBacker := agentOfRole(t, w, regFar, "backer")
	insDispatch(t, w, insHome, 8)
	planCover(w, insHome, coachLane)
	planOffer(w, farBacker, fairOffer)
	w.p33cCoverRound()
	_, rounds := coverEvents(rec)
	if len(rounds) != 2 || rounds[0].Region != regHome || rounds[0].NoOffer != 1 || rounds[0].Formed != 0 || rounds[1].Region != regFar || rounds[1].Offers != 1 {
		t.Errorf("rounds = %+v, want home with one candidate and no offer, then far with its offer unused", rounds)
	}
}

// TestCoverRound_TakesNoWord verifies that a round that forms a policy takes no word of the
// stream: the state of the world's PCG is the same before and after.
//
// Requirement: FR-INS-033
func TestCoverRound_TakesNoWord(t *testing.T) {
	t.Parallel()
	w, _ := newInsWorld(t, insConfig(defaultInsure(), defaultUnderwrite()), nil)
	insDispatch(t, w, insHome, 8)
	planCover(w, insHome, coachLane)
	planOffer(w, insBacker, fairOffer)
	before := stateOf(t, w)
	w.p33cCoverRound()
	if len(w.insurance.open) != 1 {
		t.Fatalf("%d policies formed, want 1", len(w.insurance.open))
	}
	if after := stateOf(t, w); !slices.Equal(before, after) {
		t.Error("the PCG state changed over the round, want no word taken")
	}
}

// TestInsurance_SameWorldsOppositeStrikes verifies FR-INS-025: no rule reads the loss share
// of a shipment, which the dispatch fixed from the strike. Two identical worlds, the
// shipment's share 0 in one and 10 000 in the other, form the same policy.
//
// Guard: a premium or a sum that read the strike would price the outcome and not the risk.
//
// Requirement: FR-INS-025
func TestInsurance_SameWorldsOppositeStrikes(t *testing.T) {
	t.Parallel()
	run := func(loss int64) ([]CoverFormed, []CoverRound) {
		w, rec := newInsWorld(t, insConfig(defaultInsure(), defaultUnderwrite()), nil)
		insDispatch(t, w, insHome, 8)
		w.transport.ships[0].loss = loss
		planCover(w, insHome, coachLane)
		planOffer(w, insBacker, fairOffer)
		w.p33cCoverRound()
		return coverEvents(rec)
	}
	f0, r0 := run(0)
	f1, r1 := run(10_000)
	if len(f0) != 1 || !reflect.DeepEqual(f0, f1) || !reflect.DeepEqual(r0, r1) {
		t.Errorf("a shipment struck and one spared give %+v and %+v, want the same single policy", f0, f1)
	}
}

// insuranceSources reads the module's non-test files by name.
func insuranceSources(t *testing.T) map[string]string {
	t.Helper()
	files, err := filepath.Glob("insurance*.go")
	if err != nil {
		t.Fatalf("Glob error = %v, want nil", err)
	}
	out := map[string]string{}
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("ReadFile(%s) error = %v, want nil", f, err)
		}
		out[f] = string(b)
	}
	return out
}

// TestInsurance_ReadsNoLossBP verifies, by a text guard over the module's files, that no
// code reads a shipment's loss share (the expected loss of a lane, RouteRow.LossBP and the
// candidate's loss, is the module's input) or a route's strike, a draw or the common flag
// (spec-insurance.md 6.1.2).
//
// Guard: the sum, the premium and the round must stay a function of the expected loss.
//
// Requirement: FR-INS-025
func TestInsurance_ReadsNoLossBP(t *testing.T) {
	t.Parallel()
	forbidden := regexp.MustCompile(`\bs\.loss\b|ships\[[^\]]*\]\.loss|StrikeBP|SeverityBP|\.common\b|\.rnd\b|rand\.`)
	for name, text := range insuranceSources(t) {
		code := regexp.MustCompile(`(?m)//.*$`).ReplaceAllString(text, "")
		for _, loc := range forbidden.FindAllString(code, -1) {
			t.Errorf("%s names %q, want no read of a strike, a share or a draw", name, loc)
		}
	}
}

// TestInsurance_NoBranchOnIdentifier verifies, by a text guard, that no code of the module
// compares anything with a string literal: an identifier of a mode, region or commodity
// would be one (architecture.md 9.1).
//
// Requirement: FR-INS-005, NFR-INS-005
func TestInsurance_NoBranchOnIdentifier(t *testing.T) {
	t.Parallel()
	cmpLit := regexp.MustCompile(`(==|!=)\s*"[^"]|"[^"]\s*(==|!=)|case\s+"[^"]`)
	for name, text := range insuranceSources(t) {
		if cmpLit.MatchString(text) {
			t.Errorf("%s compares with a string literal, want no branch on an identifier", name)
		}
	}
}

// TestPositions_P33cRunsAfterP33bBeforeP34 verifies the place of the cover round in the
// table of positions and that it is built.
//
// Requirement: FR-INS-030
func TestPositions_P33cRunsAfterP33bBeforeP34(t *testing.T) {
	t.Parallel()
	at := map[string]int{}
	for i, p := range positionList {
		at[p.id] = i
	}
	if !(at["P3.3b"]+1 == at["P3.3c"] && at["P3.3c"]+1 == at["P3.4"]) {
		t.Errorf("positions P3.3b, P3.3c, P3.4 at %d, %d, %d, want consecutive", at["P3.3b"], at["P3.3c"], at["P3.4"])
	}
	if slices.Contains(notBuiltPositions[:], "P3.3c") {
		t.Error("P3.3c is listed as not built, want a built position")
	}
}

// TestCost_InsuranceSitesZeroAllocs verifies that the round allocates nothing once its
// scratch has grown: with the module on, no observer and no logger, an empty round, a round
// of a request with no candidate and a refused offer.
//
// Requirement: NFR-INS-001
func TestCost_InsuranceSitesZeroAllocs(t *testing.T) {
	w := newMill(t, insConfig(defaultInsure(), defaultUnderwrite()), WithStrategy("script", &insScript{}))
	w.obs = observers{}
	plan := func() {
		planCover(w, insHome, coachLane)
		planOffer(w, insBacker, fairOffer)
		w.p33cCoverRound()
	}
	plan() // the first round grows the scratch
	if n := testing.AllocsPerRun(20, func() { w.p33cCoverRound() }); n != 0 {
		t.Errorf("an empty round allocates %v, want 0", n)
	}
	if n := testing.AllocsPerRun(20, plan); n != 0 {
		t.Errorf("a round of a request with no candidate and an offer allocates %v, want 0", n)
	}
}
