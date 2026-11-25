package economysim

import (
	"testing"

	"github.com/Whealinghold/economy-sim/internal/insure"
	"github.com/Whealinghold/economy-sim/internal/ledger"
)

// This file tests the edges of the cover round (plan 3-ins Phase 5): the cap on the unit
// value in a sum, a sum or a balance that passes the money bound, and a round with nothing
// submitted.

// TestRound_NothingSubmittedEmitsNothing verifies that a round with no request and no offer
// emits no event and forms nothing (spec-insurance.md 15.1.1: CoverRound is never emitted
// with all its counts 0).
//
// Requirement: FR-INS-072
func TestRound_NothingSubmittedEmitsNothing(t *testing.T) {
	t.Parallel()
	w, rec := newInsWorld(t, insConfig(defaultInsure(), defaultUnderwrite()), nil)
	w.p33cCoverRound()
	if formed, rounds := coverEvents(rec); len(formed)+len(rounds) != 0 {
		t.Errorf("events = %+v and %+v, want none", formed, rounds)
	}
}

// TestRound_ValueCapBindsInTheSum verifies that the sum is valued at the capped unit value
// and that the units valued at the cap are counted.
//
// Worked: the market published 5 000 for grain, the cap is 20 000 bp of the reference 1 000,
// which is 2 000, so 8 grain are insured at 8 * 2 000 = 16 000 and ValueCapped is 8.
//
// Requirement: FR-INS-020, FR-INS-021
func TestRound_ValueCapBindsInTheSum(t *testing.T) {
	t.Parallel()
	w, rec := newInsWorld(t, insConfig(defaultInsure(), defaultUnderwrite()), nil)
	i := int(regHome)*len(w.cfg.Commodities) + int(commodityOf(t, w, "grain"))
	w.markets.windows[i].Push(5_000)
	w.markets.reports[i].Price = 5_000
	insDispatch(t, w, insHome, 8)
	planCover(w, insHome, coachLane)
	planOffer(w, insBacker, CoverOffer{MaxLiability: 20_000, LoadingBP: 10_000, CoverBP: 8_000})
	got := roundOf(t, w, rec)
	if formed, _ := coverEvents(rec); len(formed) != 1 || formed[0].Sum != 16_000 || got.ValueCapped != 8 {
		t.Errorf("formed %+v, round %+v, want a policy of sum 16000 and 8 units valued at the cap", formed, got)
	}
}

// TestRound_SumPastTheBoundIsCountedOverflow verifies that a candidate whose sum passes
// MaxMoney is counted Overflow and is not scanned against offers (spec-insurance.md 6.3.5).
//
// Requirement: FR-INS-024
func TestRound_SumPastTheBoundIsCountedOverflow(t *testing.T) {
	t.Parallel()
	w, _ := newInsWorld(t, insConfig(defaultInsure(), defaultUnderwrite()), nil)
	rd := &w.insurance.round
	rd.rq = append(rd.rq[:0], admittedRequest{region: regHome})
	w.coverValues(regHome)
	rd.values[commodityOf(t, w, "grain")] = MaxMoney
	s := &shipment{id: 1, mode: modeCoach, from: regHome, to: regFar, hired: true, lines: []Line{{Commodity: commodityOf(t, w, "grain"), Quantity: 2}}}
	w.routeCandidate(regHome, &insure.Request{}, s)
	if c := rd.counts[regHome]; c.overflow != 1 || c.candidates != 1 || len(rd.cands) != 0 {
		t.Errorf("counts = %+v with %d candidates, want one candidate counted overflow and not scanned", c, len(rd.cands))
	}
}

// TestForm_UnderwriterBalancePastTheBoundDropsTheFill verifies the overflow of 7.4 step 4:
// an underwriter whose balance plus the premium would pass MaxMoney is skipped for this
// candidate with nothing moved.
//
// Worked: the underwriter holds MaxMoney - 1 000 and the premium is 1 600.
//
// Requirement: FR-INS-036
func TestForm_UnderwriterBalancePastTheBoundDropsTheFill(t *testing.T) {
	t.Parallel()
	w, rec := newInsWorld(t, insConfig(defaultInsure(), defaultUnderwrite()), nil)
	mint, ok := w.ledger.Boundary(ledger.Initial, ledger.Money)
	if !ok {
		t.Fatal("no initial money boundary")
	}
	room := MaxMoney - heldMoney(t, w, insBacker) - 1_000
	if err := w.ledger.Transfer(mint, w.agents[insBacker].account, room, ledger.Initial); err != nil {
		t.Fatalf("Transfer to the underwriter error = %v, want nil", err)
	}
	insDispatch(t, w, insHome, 8)
	planCover(w, insHome, coachLane)
	planOffer(w, insBacker, CoverOffer{MaxLiability: 10_000, LoadingBP: 10_000, CoverBP: 8_000})
	before := heldMoney(t, w, insHome)
	got := roundOf(t, w, rec)
	if got.Overflow != 1 || got.Formed != 0 || got.NoOffer != 1 || heldMoney(t, w, insHome) != before {
		t.Errorf("CoverRound = %+v, want overflow 1, no policy, no offer left and the insured's money unchanged", got)
	}
}
