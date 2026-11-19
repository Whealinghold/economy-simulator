package economysim

import (
	"errors"
	"fmt"
	"slices"
	"testing"
)

// seen is a compact name of the events of a removal, a role change or a move that
// concern a hire: "abandon:<reason>" for a RecipeAbandoned, "end:<hire>:<cause>" for
// a HireEnded, "bankrupt", "role" and "estate" for the others, in delivery order.
func seen(events []Event) []string {
	var out []string
	for _, e := range events {
		switch e := e.(type) {
		case RecipeAbandoned:
			out = append(out, "abandon:"+string(e.Reason))
		case HireEnded:
			out = append(out, fmt.Sprintf("end:%d:%s", e.Hire, e.Cause))
		case AgentBankrupted:
			out = append(out, "bankrupt")
		case RoleChanged:
			out = append(out, "role")
		}
	}
	return out
}

// bankruptMoney is the money the estate received in the AgentBankrupted of events.
func bankruptMoney(events []Event) Money {
	for _, e := range events {
		if b, ok := e.(AgentBankrupted); ok {
			return b.Money
		}
	}
	return -1
}

// midJob is a hire world of one farm and the given hands, the farm hiring one
// position a hand at a bid of 100 against a reserve of 40 (so a wage of 70), a job of
// 3 ticks and one tick run: the escrow holds 140 a hire and each worker 70.
func midJob(t *testing.T, hands int64, log *hireLog, extra ...hireOpts) *World {
	t.Helper()
	o := hireOpts{hands: hands, handRecipe: true}
	if len(extra) > 0 {
		o = extra[0]
	}
	w := hireWorld(t, defaultHire(o), farmWants(hands, 100, 40), WithObserver(log))
	stepN(t, w, 1)
	if want := int(hands) * max(1, o.regions); len(w.employment.hires) != want {
		t.Fatalf("hires open after the first tick = %d, want %d", len(w.employment.hires), want)
	}
	return w
}

// endAudit runs the audit the step would and fails the test on a finding.
func endAudit(t *testing.T, w *World) {
	t.Helper()
	if err := w.Audit(); err != nil {
		t.Fatalf("Audit() = %v, want nil", err)
	}
}

// TestInterrupt_WorkerRemoved_WorkerLeft verifies the row "the worker is removed" of
// 6.4: the worker's job is abandoned first (agent_removed), the hire ends with
// worker_left, the remainder of the escrow (140) returns to the employer and the
// worker's one wage (70) goes to the estate with the rest of its account.
//
// Requirement: FR-EMP-031, FR-EMP-036
func TestInterrupt_WorkerRemoved_WorkerLeft(t *testing.T) {
	t.Parallel()
	log := &hireLog{}
	w := midJob(t, 1, log)
	farm, hand := slotsOf(w, "farm")[0], slotsOf(w, "hand")[0]
	from := len(log.all)
	if err := w.removeAgent(&w.agents[hand], CauseConsumer); err != nil {
		t.Fatalf("removeAgent() = %v, want nil", err)
	}
	if got, want := seen(log.all[from:]), []string{"abandon:agent_removed", "end:0:worker_left", "bankrupt"}; !slices.Equal(got, want) {
		t.Errorf("events of the removal = %v, want %v", got, want)
	}
	if e := log.ended[0]; e.Paid != 70 || e.Refunded != 140 || e.Worked != 1 {
		t.Errorf("HireEnded = %+v, want 1 working tick, 70 paid, 140 refunded", e)
	}
	if got, want := balanceOf(t, w, farm), Money(10_000-210+140); got != want {
		t.Errorf("farm balance = %d, want %d: the unpaid wages come back", got, want)
	}
	if got := escrowHeld(t, w); got != 0 {
		t.Errorf("escrow held = %d, want 0", got)
	}
	endAudit(t, w)
	stepN(t, w, 2)
}

// TestInterrupt_WorkerRoleChanged verifies the row "the worker changes role": its job
// is abandoned with role_changed, the hire ends with worker_role and the role change
// follows, in that order.
//
// Requirement: FR-EMP-032
func TestInterrupt_WorkerRoleChanged(t *testing.T) {
	t.Parallel()
	log := &hireLog{}
	w := midJob(t, 1, log)
	hand := slotsOf(w, "hand")[0]
	from := len(log.all)
	if err := w.execSwitch(0, &w.agents[hand], Decision{Role: 0, Cause: CauseReview}); err != nil {
		t.Fatalf("execSwitch() = %v, want nil", err)
	}
	if got, want := seen(log.all[from:]), []string{"abandon:role_changed", "end:0:worker_role", "role"}; !slices.Equal(got, want) {
		t.Errorf("events of the role change = %v, want %v", got, want)
	}
	if e := log.ended[0]; e.Paid != 70 || e.Refunded != 140 {
		t.Errorf("HireEnded = %+v, want 70 paid and 140 refunded", e)
	}
	endAudit(t, w)
	stepN(t, w, 2)
}

// TestInterrupt_WorkerMoved verifies the row "the worker's account changes region":
// a forced Relocate abandons the job (region_changed) and ends the hire with
// worker_moved, the worker keeps its wage and the employer has the rest back. The
// worker needs no refusal: only the employer is bound (open_hire).
//
// Requirement: FR-EMP-033
func TestInterrupt_WorkerMoved(t *testing.T) {
	t.Parallel()
	log := &hireLog{}
	w := midJob(t, 1, log, hireOpts{hands: 1, handRecipe: true, regions: 2})
	hand := slotsOf(w, "hand")[0]
	from := len(log.all)
	if err := w.Relocate(AgentID(hand), 1); err != nil {
		t.Fatalf("Relocate() = %v, want nil: a worker is not held by its hire", err)
	}
	if got, want := seen(log.all[from:]), []string{"abandon:region_changed", "end:0:worker_moved"}; !slices.Equal(got, want) {
		t.Errorf("events of the move = %v, want %v", got, want)
	}
	if got := balanceOf(t, w, hand); got != 70 {
		t.Errorf("worker balance = %d, want 70: every wage paid is kept", got)
	}
	endAudit(t, w)
	stepN(t, w, 2)
}

// TestInterrupt_WorkerWeightFalls_NotAbandoned_RemainderRefundedAtEnd verifies the row
// "the worker's weight falls": a worker of 2 households hired for both has its job
// re-weighed to 1 (spec-production.md 5.4), is not abandoned, is paid 70 for the
// working ticks that follow, and the remainder of the escrow comes back with the end:
// the bill 420, wages 140 + 70 + 70, refunded 140.
//
// Requirement: FR-EMP-033
func TestInterrupt_WorkerWeightFalls_NotAbandoned_RemainderRefundedAtEnd(t *testing.T) {
	t.Parallel()
	log := &hireLog{}
	plan := func(v *AgentView) JobRequest {
		if v.Role == 0 {
			return JobRequest{NoJob: true, Hire: HireRequest{Recipe: 0, Positions: 2, Bid: 100}}
		}
		return JobRequest{NoJob: true, Offer: LabourOffer{Households: 2, Reserve: 40}}
	}
	w := hireWorld(t, defaultHire(hireOpts{hands: 2, handScale: 2, handRecipe: true}), plan, WithObserver(log))
	stepN(t, w, 1)
	hand := slotsOf(w, "hand")[0]
	if a := &w.agents[hand]; a.weight != 2 || a.job.executions != 2 || a.job.principal != AgentID(slotsOf(w, "farm")[0]) {
		t.Fatalf("worker = weight %d, job %+v, want weight 2 and a hired job of 2 executions", a.weight, a.job)
	}
	if err := w.execReduce(0, &w.agents[hand], Decision{Kind: DecideReduce, Agent: AgentID(hand), Households: 1, Cause: CauseStarved}); err != nil {
		t.Fatalf("execReduce() = %v, want nil", err)
	}
	if len(log.ended) != 0 || !w.agents[hand].hasJob() || w.agents[hand].job.executions != 1 {
		t.Fatalf("after the fall: hires ended %d, job %+v, want no end and 1 execution", len(log.ended), w.agents[hand].job)
	}
	endAudit(t, w)
	stepN(t, w, 2)
	want := HireEnded{Tick: 2, Hire: 0, Region: 0, Employer: AgentID(slotsOf(w, "farm")[0]), Worker: AgentID(hand), Recipe: 0,
		Households: 1, Worked: 3, Paid: 280, Refunded: 140, Cause: CauseCompleted}
	if len(log.ended) != 1 || log.ended[0] != want {
		t.Errorf("HireEnded = %+v, want %+v", log.ended, want)
	}
}

// TestInterrupt_EmployerRemoved_RefundToOwnAccountBeforeWaterfall_PrincipalLeft_AscendingID
// verifies the row "the employer is removed": for each hire in ascending ID the
// worker's job is abandoned with principal_left, the escrow remainder (140 a hire)
// returns to the employer's own account and the hire ends with employer_left, all
// before AgentBankrupted; so the money the estate receives (10 000 - 420 + 280) holds
// the refunds, and each worker keeps its 70.
//
// Requirement: FR-EMP-034, FR-EMP-036, FR-EMP-038
func TestInterrupt_EmployerRemoved_RefundToOwnAccountBeforeWaterfall_PrincipalLeft_AscendingID(t *testing.T) {
	t.Parallel()
	log := &hireLog{}
	w := midJob(t, 2, log)
	farm := slotsOf(w, "farm")[0]
	from := len(log.all)
	if err := w.removeAgent(&w.agents[farm], CauseConsumer); err != nil {
		t.Fatalf("removeAgent() = %v, want nil", err)
	}
	want := []string{"abandon:principal_left", "end:0:employer_left", "abandon:principal_left", "end:1:employer_left", "bankrupt"}
	if got := seen(log.all[from:]); !slices.Equal(got, want) {
		t.Errorf("events of the removal = %v, want %v", got, want)
	}
	if got := bankruptMoney(log.all[from:]); got != 10_000-420+280 {
		t.Errorf("AgentBankrupted.Money = %d, want %d: the escrow refunds reached the account before the estate took it", got, 10_000-420+280)
	}
	for _, h := range slotsOf(w, "hand") {
		if got := balanceOf(t, w, h); got != 70 || w.agents[h].hasJob() {
			t.Errorf("worker %d: balance %d, job %v, want 70 and no job", h, got, w.agents[h].hasJob())
		}
	}
	if len(w.employment.hires) != 0 || escrowHeld(t, w) != 0 {
		t.Errorf("hires open %d, escrow held %d, want 0 and 0", len(w.employment.hires), escrowHeld(t, w))
	}
	endAudit(t, w)
	stepN(t, w, 2)
}

// TestInterrupt_EmployerRoleChanged verifies the row "the employer changes role": each
// hire in ascending ID abandons the worker's job (principal_left) and ends with
// employer_role, then the role change; the workers keep their wages.
//
// Requirement: FR-EMP-035
func TestInterrupt_EmployerRoleChanged(t *testing.T) {
	t.Parallel()
	log := &hireLog{}
	w := midJob(t, 2, log)
	farm := slotsOf(w, "farm")[0]
	from := len(log.all)
	if err := w.execSwitch(0, &w.agents[farm], Decision{Role: 1, Cause: CauseReview}); err != nil {
		t.Fatalf("execSwitch() = %v, want nil", err)
	}
	want := []string{"abandon:principal_left", "end:0:employer_role", "abandon:principal_left", "end:1:employer_role", "role"}
	if got := seen(log.all[from:]); !slices.Equal(got, want) {
		t.Errorf("events of the role change = %v, want %v", got, want)
	}
	if got := balanceOf(t, w, farm); got != 10_000-420+280 {
		t.Errorf("employer balance = %d, want %d", got, 10_000-420+280)
	}
	endAudit(t, w)
	stepN(t, w, 2)
}

// TestRemoval_EventOrder_WorkerThenOwn verifies the order of 6.5.2 for an agent that is
// a worker with a job and nothing else: its own RecipeAbandoned (agent_removed, step
// 1), then HireEnded at step 2c, then AgentBankrupted and EstateOpened last.
//
// Requirement: FR-EMP-038
func TestRemoval_EventOrder_WorkerThenOwn(t *testing.T) {
	t.Parallel()
	log := &hireLog{}
	w := midJob(t, 1, log)
	from := len(log.all)
	if err := w.removeAgent(&w.agents[slotsOf(w, "hand")[0]], CauseConsumer); err != nil {
		t.Fatalf("removeAgent() = %v, want nil", err)
	}
	var names []string
	for _, e := range log.all[from:] {
		names = append(names, fmt.Sprintf("%T", e))
	}
	want := []string{"economysim.RecipeAbandoned", "economysim.HireEnded", "economysim.AgentBankrupted", "economysim.EstateOpened"}
	if !slices.Equal(names, want) {
		t.Errorf("events of the removal = %v, want %v", names, want)
	}
}

// TestRelocation_PrincipalOfOpenHire_ErrRegionBound_OpenHire_KeysAreHireIDs verifies
// 6.5.1: the employer of an open hire is refused with ErrRegionBound naming open_hire,
// a skip by the hire's instance key ("open_hire:0") lifts that hire's refusal and a
// skip by another ID does not, and the worker is not refused. Once the hire ends the
// employer moves.
//
// Requirement: FR-EMP-037
func TestRelocation_PrincipalOfOpenHire_ErrRegionBound_OpenHire_KeysAreHireIDs(t *testing.T) {
	t.Parallel()
	log := &hireLog{}
	w := midJob(t, 1, log, hireOpts{hands: 1, handRecipe: true, regions: 2})
	farm, hand := AgentID(slotsOf(w, "farm")[0]), AgentID(slotsOf(w, "hand")[0])
	reason, err := w.RelocationRefusal(farm, 1)
	var rb *RegionBoundError
	if !errors.Is(err, ErrRegionBound) || !errors.As(err, &rb) || rb.Obligation != "open_hire" || reason == "" {
		t.Fatalf("RelocationRefusal(employer) = %q, %v, want ErrRegionBound naming open_hire", reason, err)
	}
	if _, err := w.RelocationRefusal(farm, 1, "open_hire:0"); err != nil {
		t.Errorf("RelocationRefusal(employer, skip open_hire:0) = %v, want nil: the key is the hire ID", err)
	}
	if _, err := w.RelocationRefusal(farm, 1, "open_hire:1"); !errors.Is(err, ErrRegionBound) {
		t.Errorf("RelocationRefusal(employer, skip open_hire:1) = %v, want ErrRegionBound: no hire 1", err)
	}
	if _, err := w.RelocationRefusal(hand, 1); err != nil {
		t.Errorf("RelocationRefusal(worker) = %v, want nil", err)
	}
	if err := w.Relocate(farm, 1); !errors.Is(err, ErrRegionBound) {
		t.Errorf("Relocate(employer) = %v, want ErrRegionBound", err)
	}
	stepN(t, w, 2)
	if _, err := w.RelocationRefusal(farm, 1); err != nil {
		t.Errorf("RelocationRefusal(employer) after the hire ended = %v, want nil", err)
	}
}

// TestSnapshot_HiresRoleRowsAndAgentState verifies the rows of 15.4 and the group of
// AgentState on a hire one tick into a job of 3: the HireRow (wage 70, 1 household,
// formed at 0, ends at tick 2, 2 ticks left, the escrow account holding 140), the
// households employed on the worker's role row and employing on the employer's, the
// employer of the worker and the executions hired by the farm, and nothing once the
// hire has ended.
//
// Requirement: FR-EMP-055
func TestSnapshot_HiresRoleRowsAndAgentState(t *testing.T) {
	t.Parallel()
	log := &hireLog{}
	w := midJob(t, 1, log)
	farm, hand := AgentID(slotsOf(w, "farm")[0]), AgentID(slotsOf(w, "hand")[0])
	s, err := w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot() = %v, want nil", err)
	}
	h := w.employment.hires[0]
	want := []HireRow{{Hire: 0, Region: 0, Employer: farm, Worker: hand, Recipe: 0, Households: 1, Wage: 70,
		Formed: 0, Ends: 2, TicksLeft: 2, Account: h.escrow, Held: 140}}
	if !slices.Equal(s.Hires, want) {
		t.Errorf("Snapshot().Hires = %+v, want %+v", s.Hires, want)
	}
	if got := []int64{s.Roles[0].Employing, s.Roles[0].Employed, s.Roles[1].Employed, s.Roles[1].Employing}; !slices.Equal(got, []int64{1, 0, 1, 0}) {
		t.Errorf("Employing and Employed of the farm role, then Employed and Employing of the hand role = %v, want [1 0 1 0]", got)
	}
	if s.Escrow != 140 {
		t.Errorf("Snapshot().Escrow = %d, want 140", s.Escrow)
	}
	checkHireAgentState(t, w, farm, hand)
	stepN(t, w, 2)
	s, _ = w.Snapshot()
	if len(s.Hires) != 0 || s.Roles[0].Employing != 0 || s.Roles[1].Employed != 0 {
		t.Errorf("after the end: Hires %+v, Employing %d, Employed %d, want none", s.Hires, s.Roles[0].Employing, s.Roles[1].Employed)
	}
}

// checkHireAgentState checks the employment group of AgentState of a worker and of
// its employer, one hire of 1 household open.
func checkHireAgentState(t *testing.T, w *World, farm, hand AgentID) {
	t.Helper()
	ws, err := w.Agent(hand)
	if err != nil || !ws.HasEmployer || ws.Employer != farm || ws.Hired != 0 {
		t.Errorf("Agent(worker) = HasEmployer %v, Employer %d, Hired %d (%v), want the farm as employer", ws.HasEmployer, ws.Employer, ws.Hired, err)
	}
	fs, err := w.Agent(farm)
	if err != nil || fs.HasEmployer || fs.Hired != 1 {
		t.Errorf("Agent(employer) = HasEmployer %v, Hired %d (%v), want 1 hired execution", fs.HasEmployer, fs.Hired, err)
	}
}
