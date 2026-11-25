package economysim

import (
	"bytes"
	"io"
	"log/slog"
	"slices"
	"testing"
)

// This file tests the cost and the non-interference of the settlement (plan 3-ins Phase 6;
// NFR-INS-001, NFR-INS-002, NFR-INS-003): the two positions allocate nothing once their
// scratch has grown, whether they settle nothing or pay a policy, and a world with an
// observer and a logger runs bit for bit as one with neither.

// TestCost_InsuranceSettleZeroAllocs verifies that P1.3a with no arrival and P8.1b with no
// store policy allocate nothing, and that P8.1b settling a store policy that is struck and
// paid (the pass, the transfer and the amendment of the record) allocates nothing once
// grown, with no observer and a logger that discards.
//
// Requirement: NFR-INS-001
func TestCost_InsuranceSettleZeroAllocs(t *testing.T) {
	w := newMill(t, storeInsConfig(), WithStrategy("script", &insScript{}))
	w.obs = observers{}
	in := w.insurance
	if n := testing.AllocsPerRun(20, w.p13aSettleRoutes); n != 0 {
		t.Errorf("P1.3a with no arrival allocates %v, want 0", n)
	}
	if n := testing.AllocsPerRun(20, w.p81bSettleStores); n != 0 {
		t.Errorf("P8.1b with no store policy allocates %v, want 0", n)
	}
	w.risk.strikes = []storeStrike{{agent: insHome, region: regHome, class: 0, stores: 1, lo: 0, hi: 1}}
	w.risk.amounts = []Amount{{Commodity: commodityOf(t, w, "grain"), Quantity: 1}}
	id := int64(0)
	pay := func() {
		in.add(coverPolicy{id: id, kind: CoverStore, insured: insHome, underwriter: insBacker, shipment: NoShipment, sum: 1_000, maxPayout: 100, coverBP: 8_000, premium: 10})
		id++
		w.p81bSettleStores()
	}
	pay() // the first settlement grows the scratch
	before := heldMoney(t, w, insHome)
	if n := testing.AllocsPerRun(20, pay); n != 0 {
		t.Errorf("P8.1b paying a store policy allocates %v, want 0", n)
	}
	if heldMoney(t, w, insHome) == before {
		t.Errorf("no payout moved in the measured runs, so the case measured the empty path")
	}
}

// TestDeterminism_InsuranceNonInterference verifies that the settlement and its welfare
// terms do not depend on an observer or a logger: the world of the paid shortfall run with
// both and with neither ends in the same state and the same money, and the Debug lines of
// the module appear only in the first.
//
// Requirement: NFR-INS-003, NFR-INS-004
func TestDeterminism_InsuranceNonInterference(t *testing.T) {
	t.Parallel()
	run := func(observed bool) ([]byte, []int64, string) {
		var buf bytes.Buffer
		opts := []Option{WithSeed(5)}
		if observed {
			opts = append(opts, WithLogger(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))))
		} else {
			opts = append(opts, WithLogger(slog.New(slog.NewTextHandler(io.Discard, nil))))
		}
		c := insConfig(defaultInsure(), defaultUnderwrite())
		w, _ := newInsWorld(t, c, nil, opts...)
		if !observed {
			w.obs = observers{}
		}
		insDispatch(t, w, insHome, 8)
		insDispatch(t, w, insHome2, 12)
		planCover(w, insHome, coachLane)
		planCover(w, insHome2, coachLane)
		planOffer(w, insBacker, bigOffer)
		w.p33cCoverRound()
		step(t, w, 2)
		moneyOf(t, w, insBacker, 2_001)
		step(t, w, 3)
		var money []int64
		for i := range w.agents {
			money = append(money, heldMoney(t, w, w.agents[i].id))
		}
		return stateOf(t, w), money, buf.String()
	}
	stA, moneyA, logA := run(true)
	stB, moneyB, logB := run(false)
	if !bytes.Equal(stA, stB) || !slices.Equal(moneyA, moneyB) {
		t.Errorf("the world with an observer and a Debug logger differs from the one with neither: money %v and %v", moneyA, moneyB)
	}
	if !bytes.Contains([]byte(logA), []byte("cover ended")) || logB != "" {
		t.Errorf("Debug lines: with a logger %d bytes (want a \"cover ended\" line), with none %q", len(logA), logB)
	}
}
