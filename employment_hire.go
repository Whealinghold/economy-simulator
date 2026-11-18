package economysim

import (
	"cmp"
	"fmt"
	"slices"

	"github.com/Whealinghold/economy-sim/internal/ledger"
	"github.com/Whealinghold/economy-sim/internal/transit"
)

// This file is the hired job of the employment module (spec-employment.md 3.4, 6,
// 7), EMP: the hire record, the Escrow account of a hire, the release of the wage
// a working tick at a time, the cut of a start, the end of a hire and the audit.
// The labour round that forms a hire is employment_round.go and the reports are
// employment_report.go. Interruptions (a removal, a role change, a move, a fall of
// the worker's weight), the checkpoint section and the snapshot rows are later
// PRs of the plan; a hire ends here by its completion or because nothing started.

// hireRole is the employer block of a role, resolved: the hire list in ascending
// RecipeID, the establishment, the worker's share and the hired yield factor.
type hireRole struct {
	on       bool
	list     []RecipeID
	maxHires int64
	shareBP  int64
	yieldBP  int64
}

// hireRec is a hire in progress (spec-employment.md 3.4.1). The job itself is
// the worker's job, with the employer as principal: the record holds the rest.
type hireRec struct {
	id       HireID
	region   RegionID
	employer AgentID
	worker   AgentID
	recipe   RecipeID
	// escrow is the Escrow account that holds the unpaid wages.
	escrow int
	// wage is the wage per household-tick, k the executions of the job (cut at the
	// start, scaled by a fall of the weight), ticks the duration of the recipe and
	// formed the tick of the formation.
	wage, k, ticks int64
	formed         int64
	// started says the job began: the record of a hire between its formation at
	// P2.5a and its start at P2.7 is unstarted.
	started bool
	// worked is the working ticks paid and paid the wages paid so far.
	worked, paid int64
	// reserve and bid are the two limits that fixed the wage.
	reserve, bid Price
}

// initHire resolves the employer blocks and sizes the state of the hire form.
func (e *employment) initHire(c *Config) {
	e.hire, e.maxOpen = true, c.Modules.Employment.MaxOpen
	e.roles = make([]hireRole, len(c.Roles))
	e.open = make([]int64, len(c.Regions))
	e.free = make([]transit.Free, len(c.Regions))
	e.tally = make([]labourTally, len(c.Regions))
	e.lanes = make([]labourLane, len(c.Regions))
	for i := range c.Roles {
		b := c.Roles[i].Employer
		if b == nil {
			continue
		}
		r := hireRole{on: true, maxHires: b.MaxHires, shareBP: b.WorkerShareBP, yieldBP: b.HireYieldBP}
		for _, name := range b.Recipes {
			r.list = append(r.list, RecipeID(slices.IndexFunc(c.Recipes, func(x Recipe) bool { return x.ID == name })))
		}
		slices.Sort(r.list)
		e.roles[i] = r
	}
}

// isHire says the world is in the hire form; it is false for a nil module.
func (e *employment) isHire() bool { return e != nil && e.hire }

// hireOfWorker returns the hire in progress of the agent as a worker, or nil.
func (e *employment) hireOfWorker(id AgentID) *hireRec {
	if !e.isHire() || id < 0 || int(id) >= len(e.workerOf) || e.workerOf[id] == 0 {
		return nil
	}
	i, ok := slices.BinarySearchFunc(e.hires, e.workerOf[id]-1, func(h hireRec, x HireID) int { return cmp.Compare(h.id, x) })
	if !ok {
		invariant("the hire of a worker is not in the list of hires")
	}
	return &e.hires[i]
}

// unstarted returns the hire of the agent as a worker whose job has not started,
// or nil: the hire a request at P2.7 is made for.
func (e *employment) unstarted(id AgentID) *hireRec {
	if h := e.hireOfWorker(id); h != nil && !h.started {
		return h
	}
	return nil
}

// hiredBy is the executions the agent has open as an employer.
func (e *employment) hiredBy(id AgentID) int64 {
	if id < 0 || int(id) >= len(e.hired) {
		return 0
	}
	return e.hired[id]
}

// grow makes the per-agent slices cover the agent slot id.
func (e *employment) grow(id AgentID) {
	for int(id) >= len(e.workerOf) {
		e.workerOf = append(e.workerOf, 0)
		e.hired = append(e.hired, 0)
	}
}

// escrowTake returns the Escrow account of the lowest ID in the region that no
// open contract or hire uses, or opens one (spec-employment.md 5.5, 7.2). The
// free list is the merchants module's when that module is on, so the two cannot
// take the same account; with it off the module keeps its own.
func (w *World) escrowTake(region RegionID) int {
	if w.mer != nil && w.mer.on {
		return w.escrowAccount(region)
	}
	if id, ok := w.employment.free[region].Take(); ok {
		return id
	}
	id, err := w.ledger.Open(ledger.KindEscrow, 0, int(region))
	if err != nil {
		invariant("the ledger refused an Escrow account: " + err.Error())
	}
	return id
}

// escrowPut returns an Escrow account of a hire that ended to the free list.
func (w *World) escrowPut(region RegionID, id int) {
	if w.mer != nil && w.mer.on {
		w.mer.free[region].Put(id)
		return
	}
	w.employment.free[region].Put(id)
}

// markEscrow adds the Escrow accounts of the open hires to used, the set the
// merchants module's audit and restore read to tell an account in use from a free
// one (architecture.md 2.10.1: one helper, and a contract that names a hire's
// account is then "used by two contracts"). It is a no-op for a nil module.
func (e *employment) markEscrow(used map[int]bool) {
	if !e.isHire() {
		return
	}
	for i := range e.hires {
		used[e.hires[i].escrow] = true
	}
}

// release pays the wage of the working tick of the worker a, wage times the
// executions of its job, from the escrow of its hire (spec-employment.md 6.2). It
// is called at P2.7 for the start tick and at P2.1 for each later tick, once the
// job has taken the tick. It is a no-op for a worker with no started hire.
func (w *World) releaseWage(a *agent) {
	h := w.employment.hireOfWorker(a.id)
	if h == nil || !h.started {
		return
	}
	amount := h.wage * a.job.executions
	if err := w.ledger.Transfer(h.escrow, a.account, amount, ledger.WagePay); err != nil {
		invariant("the escrow of a hire cannot pay its wage: " + err.Error())
	}
	h.worked++
	h.paid += amount
	w.welfare.addEarned(int(a.id), amount)
	w.welfare.addWagePaid(int(h.employer), amount)
	t := w.employment.tallyOf(h.region, w.Tick())
	t.paid += amount
	t.householdTicks += a.job.executions
}

// hiredStart is called when the job of the worker a started k executions at P2.7:
// the hire is started and, when the ration, the rent or the room cut the job below
// the executions hired, the wages of the executions that did not start go back to
// the employer at once (spec-employment.md 6.1.3), counted in the round of the
// tick. The hire goes on and no HireEnded is emitted.
func (w *World) hiredStart(a *agent, k int64) {
	h := w.employment.hireOfWorker(a.id)
	if h == nil {
		return
	}
	h.started = true
	cut := h.k - k
	if cut <= 0 {
		return
	}
	refund := h.wage * h.ticks * cut
	w.refundHire(h, h.employer, refund)
	w.employment.tallyOf(h.region, w.Tick()).refunded += refund
	w.employment.hired[h.employer] -= cut
	h.k = k
}

// hiredNotStarted ends the hire of the worker a whose job started no execution,
// the whole escrow refunded (spec-employment.md 6.1.3). The worker is unmatched
// for the tick: it has no job and its own request was withdrawn.
func (w *World) hiredNotStarted(a *agent) {
	h := w.employment.hireOfWorker(a.id)
	if h == nil {
		return
	}
	w.employment.tallyOf(h.region, w.Tick()).notStarted += h.k
	w.endHire(h, CauseNotStarted)
}

// hiredCompleted ends the hire of the worker a whose job completed: the output
// was created in the employer's account by the ordinary completion, the last
// wage was paid and what is left of the escrow returns to the employer
// (spec-employment.md 6.3).
func (w *World) hiredCompleted(a *agent) {
	if h := w.employment.hireOfWorker(a.id); h != nil {
		w.endHire(h, CauseCompleted)
	}
}

// refundHire moves amount from the escrow of h to the account of agent to under
// wage_refund. A refund of 0 moves nothing.
func (w *World) refundHire(h *hireRec, to AgentID, amount int64) {
	if amount < 1 {
		return
	}
	if err := w.ledger.Transfer(h.escrow, w.agents[to].account, amount, ledger.WageRefund); err != nil {
		invariant("the escrow of a hire cannot refund the employer: " + err.Error())
	}
}

// endHire ends the hire h: the remainder of the escrow goes back to the employer,
// the Escrow account is released to the free list, the hire leaves the list and
// HireEnded is emitted (spec-employment.md 6.3, 6.4). The job of the worker is the
// caller's.
func (w *World) endHire(h *hireRec, cause Cause) {
	e := w.employment
	v, err := w.ledger.Account(h.escrow)
	if err != nil {
		invariant("the Escrow account of a hire is unreadable: " + err.Error())
	}
	w.refundHire(h, h.employer, v.Balance)
	w.escrowPut(h.region, h.escrow)
	rec := *h
	e.workerOf[rec.worker] = 0
	e.hired[rec.employer] -= rec.k
	e.open[rec.region]--
	i, _ := slices.BinarySearchFunc(e.hires, rec.id, func(x hireRec, id HireID) int { return cmp.Compare(x.id, id) })
	e.hires = slices.Delete(e.hires, i, i+1)
	w.reportHireEnded(&rec, Money(v.Balance), cause)
}

// hiredRequest turns the start request q of the worker a into a hired one when it
// has a hire that has not started: the inputs and the tools come from the
// employer's account, the employer is the principal and the output goes to it, and
// the effective factor carries HireYieldBP in the place of the trainee term
// (spec-employment.md 6.1.1). It leaves the request of any other agent as it is.
func (w *World) hiredRequest(q *startReq, a *agent, region RegionID) {
	h := w.employment.unstarted(a.id)
	if h == nil {
		return
	}
	cr := &w.cfg.Recipes[q.recipe]
	p := &w.agents[h.employer]
	e, has, ok := w.yields.effectiveFor(&w.shocks, region, q.recipe, w.tk.moment.Season, cr.Sited,
		w.prod.abundance(region, q.recipe, false), w.prod.techFactor(region, q.recipe), w.employment.roles[p.role].yieldBP)
	if !ok {
		invariant("the effective yield of a hired job does not fit after validation")
	}
	q.account, q.principal, q.effective, q.hasProfile = p.account, p.id, e, has
}

// snapshotHireEscrow adds the money held in the Escrow accounts of the open hires
// to the contract sector of the snapshot: the total in escrow, the money of the
// region and the money of the world (spec-employment.md 7.4.1).
func (w *World) snapshotHireEscrow(s *Snapshot) {
	e := w.employment
	if !e.isHire() {
		return
	}
	for i := range e.hires {
		h := &e.hires[i]
		v, err := w.ledger.Account(h.escrow)
		if err != nil {
			invariant("the Escrow account of a hire is unreadable: " + err.Error())
		}
		held := Money(v.Balance)
		s.Escrow = satMoney(s.Escrow, held, 0)
		s.Regions[h.region].MoneyContract = satMoney(s.Regions[h.region].MoneyContract, held, 0)
		s.Money = satMoney(s.Money, held, 0)
	}
}

// auditHires is the hires' part of the audit (spec-employment.md 7.4.2): the IDs
// are ascending and below the next; each hire's Escrow account is an open Escrow
// account of its region that no other hire uses and holds at least the wages of
// the ticks left; the worker's job names the employer as principal and is the
// recipe of the hire; and, with the merchants module off, every other Escrow
// account is empty (with it on its own audit reads the same set). The error names
// the check and wraps ErrAuditFailed.
func (w *World) auditHires() error {
	e := w.employment
	used := make(map[int]bool, len(e.hires))
	for i := range e.hires {
		if msg := w.hireFault(i, used); msg != "" {
			return fmt.Errorf("audit at tick %d: %w: hires: hire %d: %s", w.Tick(), ErrAuditFailed, e.hires[i].id, msg)
		}
	}
	if w.mer != nil && w.mer.on {
		return nil
	}
	if msg := w.freeEscrowFault(used); msg != "" {
		return fmt.Errorf("audit at tick %d: %w: hires: %s", w.Tick(), ErrAuditFailed, msg)
	}
	return nil
}

// hireFault returns what is wrong with the open hire at index i, or "". It marks
// the account in used.
func (w *World) hireFault(i int, used map[int]bool) string {
	e := w.employment
	h := &e.hires[i]
	switch {
	case i > 0 && h.id <= e.hires[i-1].id:
		return fmt.Sprintf("it is not above hire %d", e.hires[i-1].id)
	case h.id >= e.next:
		return fmt.Sprintf("it is not below the next hire %d", e.next)
	case used[h.escrow]:
		return fmt.Sprintf("account %d is used by two hires", h.escrow)
	}
	used[h.escrow] = true
	return w.escrowFault(h)
}

// escrowFault returns what is wrong with the Escrow account of the hire h or with
// the job it pays for, or "".
func (w *World) escrowFault(h *hireRec) string {
	v, err := w.ledger.Account(h.escrow)
	if err != nil || v.Kind != ledger.KindEscrow || v.Closed || v.Region != int(h.region) {
		return fmt.Sprintf("account %d is not an open Escrow account of region %d", h.escrow, h.region)
	}
	left, worker := h.ticks, &w.agents[h.worker]
	if h.started {
		left = worker.job.remaining
		if !worker.hasJob() || worker.job.principal != h.employer || worker.job.recipe != h.recipe {
			return "the job of its worker is not the job of the hire"
		}
	}
	if want := h.wage * h.k * left; v.Balance < want {
		return fmt.Sprintf("account %d holds %d, want at least %d", h.escrow, v.Balance, want)
	}
	return ""
}
