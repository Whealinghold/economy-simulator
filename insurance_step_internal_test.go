package economysim

import (
	"log/slog"
	"reflect"
	"slices"
	"testing"
)

// This file tests the cover round through a whole step (plan 3-ins Phase 5): the plan check
// hook, P3.3c in the straight line, the order of the events, the Debug lines and the
// construction summary, and that a module with no block changes nothing.

// stepPlan is the plan of a step of insConfig: the miller insHome dispatches 8 grain by
// coach and asks to cover the lane, the backer offers, on tick 0 only.
func stepPlan(grain, haul CommodityID) func(v *AgentView, p *TradePlan) {
	return func(v *AgentView, p *TradePlan) {
		if v.Tick.Tick != 0 {
			return
		}
		switch v.Agent {
		case insHome:
			p.Shipments = append(p.Shipments, ShipmentRequest{Mode: modeCoach, To: regFar, Consignee: insFar, Lines: []Line{{Commodity: grain, Quantity: 8}}})
			p.Cover = append(p.Cover, coachLane)
		case insBacker:
			p.Offer = append(p.Offer, fairOffer)
		}
	}
}

// TestInsurance_FormsInAStep verifies the wiring of the round: the requests of a plan reach
// the round through the plan check, P3.3c runs in the straight line after the plans, and the
// events come in the order of spec-insurance.md 12.1.1 (the shipment's dispatch, then
// CoverFormed, then CoverRound).
//
// Requirement: FR-INS-030, FR-INS-070, FR-INS-072
func TestInsurance_FormsInAStep(t *testing.T) {
	t.Parallel()
	c := insConfig(defaultInsure(), defaultUnderwrite())
	// The plans are built before the goods are given, so the closure reads them lazily.
	var w *World
	plan := func(v *AgentView, p *TradePlan) {
		stepPlan(commodityOf(t, w, "grain"), commodityOf(t, w, "haul"))(v, p)
	}
	w, rec := newInsWorld(t, c, plan)
	give(t, w, insHome, commodityOf(t, w, "haul"), 8)
	give(t, w, insHome, commodityOf(t, w, "grain"), 8)
	step(t, w, 1)
	formed, rounds := coverEvents(rec)
	if len(formed) != 1 || len(rounds) != 1 || formed[0].Sum != 8_000 || formed[0].Premium != 1_600 {
		t.Fatalf("CoverFormed %+v, CoverRound %+v, want one policy of sum 8000 and premium 1600", formed, rounds)
	}
	var dispatched, fm, rd = -1, -1, -1
	for i, e := range rec.events {
		switch e.(type) {
		case ShipmentDispatched:
			dispatched = i
		case CoverFormed:
			fm = i
		case CoverRound:
			rd = i
		}
	}
	if !(dispatched >= 0 && dispatched < fm && fm < rd) {
		t.Errorf("event positions: dispatch %d, CoverFormed %d, CoverRound %d, want that order", dispatched, fm, rd)
	}
	if w.insurance.liabilityOf(insBacker) != 6_400 {
		t.Errorf("liability = %d, want 6400", w.insurance.liabilityOf(insBacker))
	}
}

// TestInsurance_NeutralWithNoBlock verifies FR-INS-005: a world with the module on and no
// role with a block runs bit for bit as one with the module off: the same events but for
// WorldCreated, the same stream state and the same money, over three seeds.
//
// Guard: the module must change a run only where a role has a block; a hook that drew a
// word or opened an account for the module would move every run of a pack that merely
// configures it.
//
// Requirement: FR-INS-001, FR-INS-005
func TestInsurance_NeutralWithNoBlock(t *testing.T) {
	t.Parallel()
	for _, seed := range []uint64{1, 7, 42} {
		run := func(on bool) ([]Event, []byte, []int64) {
			c := insConfig(nil, nil)
			if !on {
				c.Modules.Insurance = nil
			}
			w, rec := newInsWorld(t, c, nil, WithSeed(seed))
			step(t, w, 12)
			var money []int64
			for i := range w.agents {
				money = append(money, heldMoney(t, w, w.agents[i].id))
			}
			ev := slices.DeleteFunc(slices.Clone(rec.events), func(e Event) bool { _, ok := e.(WorldCreated); return ok })
			return ev, stateOf(t, w), money
		}
		onEv, onPCG, onMoney := run(true)
		offEv, offPCG, offMoney := run(false)
		if !reflect.DeepEqual(onEv, offEv) || !slices.Equal(onPCG, offPCG) || !slices.Equal(onMoney, offMoney) {
			t.Errorf("seed %d: the world with the module on and no block differs from the one with it off (%d and %d events)", seed, len(onEv), len(offEv))
		}
	}
}

// TestLog_CoverFormedAndRefusedAreDebugLines verifies the Debug lines of the round, their
// constant messages and keys (spec-insurance.md 15.2): "cover formed" with the policy, the
// insured, its region, the sum and the premium, and "cover refused" with the agent and the
// reason; and that nothing is logged above Debug in the tick.
//
// Requirement: FR-INS-075
func TestLog_CoverFormedAndRefusedAreDebugLines(t *testing.T) {
	t.Parallel()
	lg := newCapture()
	w, _ := newInsWorld(t, insConfig(defaultInsure(), defaultUnderwrite()), nil, WithLogger(slog.New(lg)))
	lg.take("world created")
	insDispatch(t, w, insHome, 8)
	planCover(w, insHome, coachLane)
	planOffer(w, insBacker, fairOffer)
	planOffer(w, insHome, fairOffer) // not an underwriter
	w.p33cCoverRound()
	all := slices.Clone(*lg.lines)
	byMsg := func(msg string) (out []logRec) {
		for _, l := range all {
			if l.msg == msg {
				out = append(out, l)
			}
		}
		return out
	}
	formed, refused := byMsg("cover formed"), byMsg("cover refused")
	wantFormed := map[string]int64{"cover_policy": 0, "agent": int64(insHome), "region": int64(regHome), "sum": 8_000, "premium": 1_600}
	if len(formed) != 1 || formed[0].level != slog.LevelDebug || !reflect.DeepEqual(formed[0].attrs, wantFormed) {
		t.Errorf(`"cover formed" lines = %+v, want one Debug line with attrs %v`, formed, wantFormed)
	}
	wantRefused := map[string]int64{"agent": int64(insHome)}
	if len(refused) != 1 || refused[0].level != slog.LevelDebug || !reflect.DeepEqual(refused[0].attrs, wantRefused) || refused[0].strs["reason"] != string(ReasonNotUnderwriter) {
		t.Errorf(`"cover refused" lines = %+v, want one Debug line of the miller with the reason not_underwriter`, refused)
	}
	for _, l := range all {
		if l.level > slog.LevelDebug {
			t.Errorf("line %q at level %v in the round, want none above Debug", l.msg, l.level)
		}
	}
}

// TestLog_ConstructionSummaryInsurance verifies that the Info line "world created" lists
// insurance among the modules and carries the counts of the agents with each block
// (spec-insurance.md 15.2).
//
// Arithmetic: insConfig has two millers in home and two in far with an Insure block and one
// backer with an Underwrite block.
//
// Requirement: FR-INS-078
func TestLog_ConstructionSummaryInsurance(t *testing.T) {
	t.Parallel()
	lg := newCapture()
	newInsWorld(t, insConfig(defaultInsure(), defaultUnderwrite()), nil, WithLogger(slog.New(lg)))
	lines := lg.take("world created")
	if len(lines) != 1 || lines[0].attrs["underwriters"] != 1 || lines[0].attrs["insured"] != 4 {
		t.Fatalf(`"world created" lines = %+v, want one with underwriters 1 and insured 4`, lines)
	}
	if got := modulesOn(&Config{Modules: Modules{Insurance: &Insurance{}}}); !slices.Contains(got, "insurance") {
		t.Errorf("modulesOn = %v, want insurance named", got)
	}
}

// TestInsurance_MetricsDerivable verifies FR-INS-073 for the instruments the round can
// feed (rule table 5.3.1): policies by kind, premium, liability, refused and limited are
// folded from CoverFormed and CoverRound alone, and agree with the world. The instruments
// of CoverSettled and CoverEnded are Phase 6's.
//
// Arithmetic: the one policy of the worked round has premium 1 600 and maximum payout
// 6 400 (the liability of the backer); the miller's offer is refused (not an underwriter),
// so refused is 1 and no `cover_limit` counts.
//
// Requirement: FR-INS-073
func TestInsurance_MetricsDerivable(t *testing.T) {
	t.Parallel()
	w, rec := newInsWorld(t, insConfig(defaultInsure(), defaultUnderwrite()), nil)
	insDispatch(t, w, insHome, 8)
	planCover(w, insHome, coachLane)
	planOffer(w, insBacker, fairOffer)
	planOffer(w, insHome, fairOffer) // not an underwriter: refused
	w.p33cCoverRound()
	policies, premium, liability, refused, limited := foldCoverMetrics(rec)
	if policies[CoverRoute] != 1 || policies[CoverStore] != 0 || premium != 1_600 || refused != 1 {
		t.Errorf("policies %v, premium %d, refused %d from the events, want one route policy, 1600 and 1", policies, premium, refused)
	}
	if got := w.insurance.liabilityOf(insBacker); liability != got {
		t.Errorf("liability folded from CoverFormed = %d, world = %d, want equal", liability, got)
	}
	for k, n := range limited {
		if n != 0 {
			t.Errorf("limited[%s] = %d, want 0 in a round with no limit reached", k, n)
		}
	}
}

// foldCoverMetrics folds the instruments of rule table 5.3.1 that CoverFormed and
// CoverRound feed: policies by kind, premium, liability, refused and limited by cover_limit.
func foldCoverMetrics(rec *evLog) (policies map[CoverKind]int64, premium, liability, refused int64, limited map[string]int64) {
	formed, rounds := coverEvents(rec)
	policies, limited = map[CoverKind]int64{}, map[string]int64{}
	for _, f := range formed {
		policies[f.Kind]++
		premium += int64(f.Premium)
		liability += int64(f.MaxPayout)
	}
	for _, r := range rounds {
		refused += r.RequestsRefused + r.OffersRefused
		for k, n := range map[string]int64{"in_arrears": r.InArrears, "exposure": r.Exposure, "no_risk": r.NoRisk,
			"no_offer": r.NoOffer, "below_minimum": r.BelowMinimum, "funds": r.Funds, "too_many": r.TooMany, "overflow": r.Overflow} {
			limited[k] += n
		}
	}
	return policies, premium, liability, refused, limited
}
