package economysim_test

import (
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// stepAllocBaseline is the number of allocations of one full Step of pack one
// after 50 ticks, measured under -race with the logger and observer at their
// no-op defaults and WithAudit off. Only gate R checks it: counts differ without
// -race. It is a baseline: raising it is a reviewed change with a reason (rule 8.2).
const stepAllocBaseline = 70

// moneyStepAllocBaseline is the same measure for pack one with the money supply
// module alone on (overlay/faucet-sunk.toml): 71, one above the off world. P1.7
// itself allocates nothing once the scratch has grown (TestP17Flows_AllocatesNothing),
// so the one is in the run the money makes (a different run: other trades and entrants),
// not traced further. Raising it is a reviewed change (rule 8.2).
const moneyStepAllocBaseline = 71

// externalAgentStepAllocBaseline is the same measure for pack one with one external
// directed agent added before the first step (NFR-INT-002): P1.9 itself allocates
// nothing, so the figure is the off world's 70 plus 2 for the run the agent makes (it trades
// like a household of its role, with the fallback strategy). Measured under -race.
// Raising it is a reviewed change (rule 8.2).
const externalAgentStepAllocBaseline = 72

// storageStepAllocBaseline is the same measure for pack one with the storage module
// alone on (overlay/storage.toml, limited accounts): 70, the off world's figure, as
// NFR-STO-002 asks (the baseline plus nothing for the module). The 73 it was came from
// the room a household gets when it enters (one allocation each, now cut from a block of
// 64 by the ledger) and from the lots a role change reads to cut the account to its room
// (now a scratch of the module). Measured under -race at 50 ticks, seed 1. Raising it is
// a reviewed change (rule 8.2).
const storageStepAllocBaseline = 70

// tenureStepAllocBaseline is the same measure for pack one with the resources module and the
// tenure module on (overlay/resources.toml and overlay/tenure.toml): 72, two above the off world's 70 (the run the
// two modules make, not split between them). Measured under -race at 50 ticks, seed 1; the tiers, the entitlements and the charge work in scratch
// the module keeps, so the module adds what the run it makes adds. Raising it is a reviewed
// change (rule 8.2).
const tenureStepAllocBaseline = 72

// creditStepAllocBaseline is the allocation limit of a Step of pack one with the credit module on
// (overlay/credit.toml): 94, 24 above the off world's 70. Measured under -race at 50 ticks, seed 1: the 20
// measured steps are the ticks 50 to 69, in which the first loans of the pack form (tick 58), so the 24 are
// the run the module makes (a claim record, its slot in the ledger's list and in the indexes of the module, and
// the growth of the due ring), not a cost of the settlement or the round themselves, which work in scratch
// (NFR-CRD-001, TestCredit_AllocsPerRun). Raising it is a reviewed change (rule 8.2).
const creditStepAllocBaseline = 94

// riskStepAllocBaseline and riskStoreStepAllocBaseline are the same measure for the risk module alone
// on: pack two with its trade and transport overlays and overlay/risk.toml (a unit row and a common
// row each way, so P3.1a takes its words every tick), and pack one with overlay/storage.toml and
// overlay/risk-store.toml (a unit row on the barn, so P8.1a takes a word for each barn that holds goods).
// NFR-RSK-001 asks no more than the module off: each is the figure of the same layers without the
// risk overlay (rows "two+transport" and "one+storage"). Measured under -race at 50 ticks, seed 1:
// 64 and 64 on pack two, 70 and 70 on pack one, so the module adds nothing to a step. Raising
// one is a reviewed change (rule 8.2).
const (
	twoTransportStepAllocBase  = 64
	riskStepAllocBaseline      = twoTransportStepAllocBase
	riskStoreStepAllocBaseline = storageStepAllocBaseline
)

// TestStep_AllocationsStayAtTheBaseline checks that a full Step of pack one
// allocates no more than the baseline. Not parallel: testing.AllocsPerRun
// panics in a parallel test. Allocation counts differ under -race, so the test
// is skipped (with the reason) without it; gate R runs it.
//
// Requirement: NFR-WLD-001, NFR-OBS-002.
// Guard: an allocation added to the per-tick path.
func TestStep_AllocationsStayAtTheBaseline(t *testing.T) {
	if testing.Short() {
		t.Skip("end-to-end: builds pack one and steps it 50 ticks; skipped under -short")
	}
	if !raceEnabled {
		t.Skipf("baseline measured under -race (race=%v); counts differ without it (testing.md 5.7)", raceEnabled)
	}
	for _, row := range []struct {
		name     string
		layers   []string
		baseline float64
		external bool // an external agent is added before the first step (NFR-INT-002)
	}{
		{"one", packOne, stepAllocBaseline, false},
		{"one+faucet-sunk", packOneFaucetSunk, moneyStepAllocBaseline, false},
		{"one+external-agent", packOne, externalAgentStepAllocBaseline, true},
		{"one+storage", packOneStorage, storageStepAllocBaseline, false}, // NFR-STO-002: the off world's figure, see the constant
		{"one+credit", packOneCredit, creditStepAllocBaseline, false},    // NFR-CRD-001: the settlement and the round work in scratch
		{"one+resources+tenure", append(slices.Clone(packOneResources), "overlay/tenure.toml"), tenureStepAllocBaseline, false},
		{"two+transport", packTwoTransport, twoTransportStepAllocBase, false},
		{"two+transport+risk", packTwoRisk, riskStepAllocBaseline, false},               // NFR-RSK-001: P3.1a and the dispatch work in scratch
		{"one+storage+risk-store", packOneRiskStore, riskStoreStepAllocBaseline, false}, // NFR-RSK-001: P8.1a works in scratch
		{"one+external", packExternal(), stepAllocBaseline, false},                      // NFR-EXT-001: no more than the off world (36 measured)
	} {
		cfg := loadValidPack(t, row.layers...)
		w, err := economysim.NewWorld(cfg, economysim.WithSeed(1))
		if err != nil {
			t.Fatalf("%s: NewWorld error = %v, want nil", row.name, err)
		}
		if row.external {
			if _, err := w.AddAgent(economysim.NewAgent{Region: 0, Role: 0, Money: 1_000}); err != nil {
				t.Fatalf("%s: AddAgent error = %v, want nil", row.name, err)
			}
		}
		for i := range 50 {
			if err := w.Step(); err != nil {
				t.Fatalf("%s: Step at tick %d error = %v, want nil", row.name, i, err)
			}
		}
		got := testing.AllocsPerRun(20, func() {
			if err := w.Step(); err != nil {
				t.Fatalf("%s: Step error = %v, want nil", row.name, err)
			}
		})
		t.Logf("%s: allocs per Step = %v (baseline %v)", row.name, got, row.baseline)
		if got > row.baseline {
			t.Errorf("%s: allocs per Step = %v, want <= %v; the constant is a baseline, raising it is a reviewed change with a reason (rule 8.2)", row.name, got, row.baseline)
		}
	}
}
