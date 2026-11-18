package economysim

import (
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/ledger"
)

// TestHire_FormsPaysAndCompletes verifies the whole life of a hire on a hand-built
// world: a farm posts one position at a bid of 100 and a hand offers its household
// at a reserve of 40, so the wage is 40 + halfEven(5 000 * 60 / 10 000) = 70; the job
// of 3 ticks costs 70 * 3 * 1 = 210, which moves into escrow at P2.5a; the worker
// is paid 70 a tick at P2.7 and then at P2.1 of the two following ticks; at the third
// the escrow is empty and the hire ends with the cause completed.
//
// Requirement: FR-EMP-008, FR-EMP-021, FR-EMP-024, FR-EMP-029, FR-EMP-030, FR-EMP-036, FR-EMP-051, FR-EMP-052
func TestHire_FormsPaysAndCompletes(t *testing.T) {
	t.Parallel()
	log := &hireLog{}
	w := hireWorld(t, defaultHire(hireOpts{handRecipe: true}), farmWants(1, 100, 40), WithObserver(log))
	farm, hand := slotsOf(w, "farm")[0], slotsOf(w, "hand")[0]
	total := totalMoney(t, w)
	var workerPaid, inEscrow []int64
	for range 3 {
		stepN(t, w, 1)
		workerPaid = append(workerPaid, int64(balanceOf(t, w, hand)))
		inEscrow = append(inEscrow, escrowHeld(t, w))
	}
	if want := []int64{70, 140, 210}; !slices.Equal(workerPaid, want) {
		t.Errorf("worker balance after each tick = %v, want %v (a wage at P2.7, then at P2.1)", workerPaid, want)
	}
	if want := []int64{140, 70, 0}; !slices.Equal(inEscrow, want) {
		t.Errorf("escrow after each tick = %v, want %v (the bill 210 less the wages paid)", inEscrow, want)
	}
	wantFormed := []HireFormed{{Tick: 0, Hire: 0, Region: 0, Employer: AgentID(farm), Worker: AgentID(hand), EmployerRole: 0, WorkerRole: 1,
		Recipe: 0, Households: 1, Wage: 70, Ticks: 3, Escrowed: 210, Reserve: 40, Bid: 100}}
	if !slices.Equal(log.formed, wantFormed) {
		t.Errorf("HireFormed = %+v, want %+v", log.formed, wantFormed)
	}
	wantEnded := []HireEnded{{Tick: 2, Hire: 0, Region: 0, Employer: AgentID(farm), Worker: AgentID(hand), Recipe: 0, Households: 1,
		Worked: 3, Paid: 210, Refunded: 0, Cause: CauseCompleted}}
	if !slices.Equal(log.ended, wantEnded) {
		t.Errorf("HireEnded = %+v, want %+v", log.ended, wantEnded)
	}
	if got, want := []Money{totalMoney(t, w), balanceOf(t, w, farm)}, []Money{total, 10_000 - 210 + 0}; got[0] != want[0] || got[1] != want[1] {
		t.Errorf("sum of the household balances and the farm's = %v, want %v: wages plus refunds equal the bill, nothing is created", got, want)
	}
	if n, h := len(w.employment.hires), w.employment.hiredBy(AgentID(farm)); n != 0 || h != 0 {
		t.Errorf("hires open = %d, executions open as an employer = %d, want 0 and 0 after the end", n, h)
	}
}

// TestHired_InputsFromPrincipalOutputToPrincipalAtHireYield verifies that a hired job
// takes its input from the employer's account and creates its output in it, at the
// factor HireYieldBP, not the worker's: with HireYieldBP 5 000 one execution of a
// recipe of output 2 makes 2 * 5 000 / 10 000 = 1 grain from 1 grain of input, so
// two hires of one household each leave the farm's 2 grain at 2 (2 - 2 + 2).
//
// Requirement: FR-EMP-027, FR-EMP-028, FR-EMP-008
func TestHired_InputsFromPrincipalOutputToPrincipalAtHireYield(t *testing.T) {
	t.Parallel()
	log := &hireLog{}
	w := hireWorld(t, defaultHire(hireOpts{hands: 2, input: true, farmGrain: 2, duration: 1, yieldBP: 5_000}), farmWants(2, 100, 10), WithObserver(log))
	farm := slotsOf(w, "farm")[0]
	stepN(t, w, 1)
	held, _, _ := w.ledger.Holding(w.agents[farm].account, 0)
	var workers []AgentID
	for _, e := range log.all {
		if r, ok := e.(RecipeExecuted); ok {
			workers = append(workers, r.Agent)
		}
	}
	hands := slotsOfID(w, "hand")
	if got, want := []int64{held, int64(len(log.ended))}, []int64{2, 2}; !slices.Equal(got, want) {
		t.Errorf("farm grain and hires ended after the one-tick jobs = %v, want %v", got, want)
	}
	if !slices.Equal(workers, hands) && !slices.Equal(workers, []AgentID{hands[1], hands[0]}) {
		t.Errorf("Agent of RecipeExecuted = %v, want the two workers %v", workers, hands)
	}
	if log.ended[0].Cause != CauseCompleted || log.ended[1].Cause != CauseCompleted {
		t.Errorf("HireEnded = %+v, want two completed hires", log.ended)
	}
}

// TestHired_OwnRequestIsWithdrawnWhenMatchedAndStandsWhenNot verifies that the own
// job of a worker with a recipe is withdrawn when its offer is matched (it starts the
// hired job only) and starts when the offer is not matched: of two hands with own
// jobs offering at reserves 10 and 20, one position is filled by the first and the
// second starts its own job at the same tick.
//
// Requirement: FR-EMP-012
func TestHired_OwnRequestIsWithdrawnWhenMatchedAndStandsWhenNot(t *testing.T) {
	t.Parallel()
	plan := func(v *AgentView) JobRequest {
		if v.Role == 0 {
			return JobRequest{NoJob: true, Hire: HireRequest{Recipe: 0, Positions: 1, Bid: 100}}
		}
		return JobRequest{Recipe: 0, Offer: LabourOffer{Households: 1, Reserve: Price(10 * (int64(v.Agent) + 1))}}
	}
	w := hireWorld(t, defaultHire(hireOpts{hands: 2, handRecipe: true, duration: 2}), plan)
	hands := slotsOf(w, "hand")
	stepN(t, w, 1)
	hired, own := &w.agents[hands[0]], &w.agents[hands[1]]
	if hired.job.principal != AgentID(slotsOf(w, "farm")[0]) || !hired.hasJob() {
		t.Errorf("the matched worker's job = %+v, want the hired job with the farm as principal", hired.job)
	}
	if !own.hasJob() || own.job.principal != own.id {
		t.Errorf("the unmatched worker's job = %+v, want its own job started", own.job)
	}
	if len(w.employment.hires) != 1 {
		t.Errorf("open hires = %d, want 1", len(w.employment.hires))
	}
}

// TestHired_CutStartRefundsTheUnstartedWages verifies the cut start of 6.1.3 on a
// hire of 2 executions whose input is spent between the formation and the start: the
// start runs 1 execution, the wages of the other, wage * ticks = 70 * 2 = 140, go back
// to the employer at once under wage_refund with no HireEnded, and the worker is paid
// for the one execution.
//
// Requirement: FR-EMP-026, FR-EMP-036
func TestHired_CutStartRefundsTheUnstartedWages(t *testing.T) {
	t.Parallel()
	log := &hireLog{}
	plan := func(v *AgentView) JobRequest {
		if v.Role == 0 {
			return JobRequest{NoJob: true, Hire: HireRequest{Recipe: 0, Positions: 2, Bid: 100}}
		}
		return JobRequest{NoJob: true, Offer: LabourOffer{Households: 2, Reserve: 40}}
	}
	// One hand agent of weight 2: the cohort is hired whole.
	w := hireWorld(t, defaultHire(hireOpts{hands: 2, handScale: 2, input: true, farmGrain: 2, duration: 2}), plan, WithObserver(log))
	farm, hand := slotsOf(w, "farm")[0], slotsOf(w, "hand")[0]
	total := totalMoney(t, w)
	runToRation(t, w)
	formed := len(log.formed)
	if err := w.ledger.Transform(w.agents[farm].account, []ledger.Line{{Commodity: 0, Quantity: 1}}, nil); err != nil {
		t.Fatalf("Transform(): %v", err)
	}
	finishStep(t, w)
	h := w.employment.hireOfWorker(AgentID(hand))
	if h == nil {
		t.Fatalf("no hire after the cut start, want one hire in progress")
	}
	got := []int64{int64(formed), h.k, int64(len(log.ended)), w.employment.tallyOf(0, 0).refunded, int64(balanceOf(t, w, hand))}
	if want := []int64{1, 1, 0, 140, 70}; !slices.Equal(got, want) {
		t.Errorf("hires formed, executions after the cut, HireEnded, refunded, worker balance = %v, want %v (70 * 2 ticks refunded, the wage of the one execution paid)", got, want)
	}
	if got := totalMoney(t, w) + Money(escrowHeld(t, w)); got != total {
		t.Errorf("household money plus escrow = %d, want %d", got, total)
	}
	// Bill 70 * 2 ticks * 2 households = 280 left the farm; 140 came back at the cut start.
	if got := balanceOf(t, w, farm); got != 10_000-280+140 {
		t.Errorf("farm balance = %d, want %d: the unstarted wages return to the employer", got, 10_000-280+140)
	}
}

// TestHired_NothingStartedEndsTheHireNotStarted verifies that a hire whose job starts
// no execution ends with the cause not_started, the whole escrow refunded, the
// worker idle, and that the households are counted NotStarted.
//
// Requirement: FR-EMP-026, FR-EMP-052
func TestHired_NothingStartedEndsTheHireNotStarted(t *testing.T) {
	t.Parallel()
	log := &hireLog{}
	w := hireWorld(t, defaultHire(hireOpts{input: true, farmGrain: 1}), farmWants(1, 100, 40), WithObserver(log))
	farm, hand := slotsOf(w, "farm")[0], slotsOf(w, "hand")[0]
	total := totalMoney(t, w)
	runToRation(t, w)
	if err := w.ledger.Transform(w.agents[farm].account, []ledger.Line{{Commodity: 0, Quantity: 1}}, nil); err != nil {
		t.Fatalf("Transform(): %v", err)
	}
	finishStep(t, w)
	want := []HireEnded{{Tick: 0, Hire: 0, Region: 0, Employer: AgentID(farm), Worker: AgentID(hand), Recipe: 0, Households: 1,
		Worked: 0, Paid: 0, Refunded: 210, Cause: CauseNotStarted}}
	if !slices.Equal(log.ended, want) {
		t.Errorf("HireEnded = %+v, want %+v (the whole escrow 210 refunded)", log.ended, want)
	}
	if got := balanceOf(t, w, farm); got != 10_000 {
		t.Errorf("farm balance = %d, want 10000: the whole escrow returns to the employer", got)
	}
	got := []int64{boolInt(w.agents[hand].hasJob()), int64(len(w.employment.hires)), w.employment.tallyOf(0, 0).notStarted, int64(totalMoney(t, w))}
	if want := []int64{0, 0, 1, int64(total)}; !slices.Equal(got, want) {
		t.Errorf("worker has a job, hires open, households not started, household money = %v, want %v", got, want)
	}
}

// TestAudit_FailsWhenTheEscrowIsBelowTheWagesLeft verifies that the audit refuses a
// hire whose escrow was drained below wage * executions * ticks left (the escrow
// holds 140 for 2 ticks of 70, and 100 is taken away), with ErrAuditFailed.
//
// Requirement: FR-EMP-041
func TestAudit_FailsWhenTheEscrowIsBelowTheWagesLeft(t *testing.T) {
	t.Parallel()
	w := hireWorld(t, defaultHire(hireOpts{}), farmWants(1, 100, 40))
	stepN(t, w, 1)
	h := &w.employment.hires[0]
	if err := w.auditHires(); err != nil {
		t.Fatalf("auditHires() = %v, want nil for a sound hire", err)
	}
	other := w.agents[slotsOf(w, "hand")[0]].account
	if err := w.ledger.Transfer(h.escrow, other, 100, ledger.WagePay); err != nil {
		t.Fatalf("Transfer(): %v", err)
	}
	if err := w.auditHires(); !errors.Is(err, ErrAuditFailed) {
		t.Errorf("auditHires() with a drained escrow = %v, want ErrAuditFailed", err)
	}
}
