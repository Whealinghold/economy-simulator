package economysim

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"slices"

	"github.com/Whealinghold/economy-sim/internal/codec"
	payroll "github.com/Whealinghold/economy-sim/internal/employment"
	"github.com/Whealinghold/economy-sim/internal/ledger"
	"github.com/Whealinghold/economy-sim/internal/logkeys"
)

// This file is what the employment module reports (spec-employment.md 15), EMP:
// the counts of the labour round by region and tick, the Debug lines of its
// refusals and the events of a hire. The aggregate event of the round, LabourRound
// at P2.8b, and the rows of the snapshot are later PRs of the plan; the counts
// they read are kept here from the first.

// labourTally is the counts of one region in one tick (spec-employment.md 15.1):
// households offered, hired and unmatched, the households of a hire that started
// nothing, positions posted, filled and unfilled, the offers and vacancies the
// admission refused, the hires formed, the pairs skipped for a wage below 1, the
// working household-ticks paid and the wages paid and refunded. refused is the
// positions each reason of labourReasons closed. stamp is the tick the counts
// belong to plus 1, so a count of an earlier tick is dropped at the first use of
// the tick (tallyOf) and no position has to clear it.
type labourTally struct {
	stamp                                         int64
	offered, hired, noVacancy, notStarted         int64
	posted, filled, unfilled                      int64
	offersRefused, vacanciesRefused, matched      int64
	wageBelowUnit, householdTicks, paid, refunded int64
	refused                                       [numLabourReasons]int64
}

// tallyOf returns the counts of region g in the tick, starting them afresh at the
// first use in a tick.
func (e *employment) tallyOf(g RegionID, tick int64) *labourTally {
	t := &e.tally[g]
	if t.stamp != tick+1 {
		*t = labourTally{stamp: tick + 1}
	}
	return t
}

// countLane adds the result of the round of region g to its counts and logs, at
// Debug, each vacancy that ended with positions left (spec-employment.md 15.2.1).
func (w *World) countLane(g RegionID, l *labourLane) {
	e := w.employment
	t := e.tallyOf(g, w.Tick())
	for _, o := range e.res.Offers {
		t.offered += o.Households
		if o.Hired {
			t.hired += o.Households
		} else {
			t.noVacancy += o.Households
		}
	}
	for _, v := range e.res.Vacancies {
		t.posted += l.posts[v.Ref].positions
		t.filled += v.Filled
		t.unfilled += v.Left
		if v.Left > 0 {
			t.refused[v.Reason] += v.Left
			w.logHireRefused(g, labourReasons[v.Reason])
		}
	}
	t.wageBelowUnit += e.res.WageBelowUnit
	t.matched += int64(len(e.res.Hires))
}

// logLabourRefused writes the Debug line of an offer or a vacancy the admission
// refused, behind Enabled. The message is constant and the reason a key.
func (w *World) logLabourRefused(msg string, agent AgentID, reason Reason) {
	if ctx := context.Background(); w.log.Enabled(ctx, slog.LevelDebug) {
		w.log.LogAttrs(ctx, slog.LevelDebug, msg, slog.Int64(logkeys.Agent, int64(agent)), slog.String(logkeys.Reason, string(reason)))
	}
}

// logHireRefused writes the Debug line "hire refused" of a vacancy that closed
// with positions left, once per vacancy, behind Enabled.
func (w *World) logHireRefused(g RegionID, reason Reason) {
	if ctx := context.Background(); w.log.Enabled(ctx, slog.LevelDebug) {
		w.log.LogAttrs(ctx, slog.LevelDebug, "hire refused", slog.Int64(logkeys.Region, int64(g)), slog.String(logkeys.Reason, string(reason)))
	}
}

// reportHireFormed delivers HireFormed under the flag (spec-employment.md 15.1).
func (w *World) reportHireFormed(h *hireRec, worker, employer *agent, bill int64) {
	if !w.obs.supplied {
		return
	}
	w.obs.deliver(HireFormed{
		Tick: w.Tick(), Hire: h.id, Region: h.region, Employer: employer.id, Worker: worker.id,
		EmployerRole: employer.role, WorkerRole: worker.role, Recipe: h.recipe, Households: h.k,
		Wage: Money(h.wage), Ticks: h.ticks, Escrowed: Money(bill), Reserve: h.reserve, Bid: h.bid,
	})
}

// reportHireEnded delivers HireEnded under the flag: the remainder of the escrow is
// what the end returned to the employer (spec-employment.md 15.1).
func (w *World) reportHireEnded(h *hireRec, refunded Money, cause Cause) {
	if !w.obs.supplied {
		return
	}
	w.obs.deliver(HireEnded{
		Tick: w.Tick(), Hire: h.id, Region: h.region, Employer: h.employer, Worker: h.worker, Recipe: h.recipe,
		Households: h.k, Worked: h.worked, Paid: Money(h.paid), Refunded: refunded, Cause: cause,
	})
}

// saveEmployment is tag 24 (spec-checkpoint.md 4.2, spec-employment.md 16), written
// with the hire form only: the next hire ID, the hires in progress in ascending ID
// (the hire, the employer, the worker, the wage, the Escrow account, the formation
// tick, the region and the wages paid so far), and the two runs of the Warn lines as
// sparse rows. The recipe, the executions and the ticks left are the worker's job,
// saved once by the production section; the free list, the executions held open and
// the role rows are derived.
func (w *World) saveEmployment(cw *codec.Writer) error {
	e := w.employment
	cw.U(unsigned(e.next))
	cw.Count(len(e.hires))
	for i := range e.hires {
		h := &e.hires[i]
		cw.U(unsigned(h.id))
		cw.U(unsigned(h.employer))
		cw.U(unsigned(h.worker))
		cw.U(unsigned(h.wage))
		cw.U(unsigned(h.escrow))
		cw.U(unsigned(h.formed))
		cw.U(unsigned(h.region))
		cw.U(unsigned(h.paid))
	}
	roles := len(w.cfg.Roles)
	writeSparse(cw, e.unmatchedRun, roles)
	writeSparse(cw, e.unfilledRun, roles)
	return nil
}

// loadEmployment reads tag 24: the next hire ID, the hires with strictly ascending
// IDs below it, each field bounded by the world read so far, then the two runs. The
// production section is already loaded, so the hires are completed from the jobs of
// their workers and checked whole (employmentCross) before the section returns.
func (w *World) loadEmployment(rs *restoreCtx, r *codec.Reader) error {
	e := w.employment
	e.next = HireID(r.Bounded("next hire", maxID))
	for i, n := 0, r.Count(8); i < n && r.Err() == nil; i++ {
		h := w.readHire(rs, r)
		if r.Err() != nil {
			return nil
		}
		if i > 0 && h.id <= e.hires[i-1].id || h.id >= e.next || int64(i) >= MaxHireRecords {
			r.Failf("hire %d is not in strictly ascending order below the next hire %d, or is more than MaxHireRecords", h.id, e.next)
			return nil
		}
		e.hires = append(e.hires, h)
	}
	roles := len(w.cfg.Roles)
	readSparse(r, e.unmatchedRun, len(w.cfg.Regions), roles, rs.tick)
	readSparse(r, e.unfilledRun, len(w.cfg.Regions), roles, rs.tick)
	if r.Err() != nil {
		return nil
	}
	return w.employmentCross()
}

// readHire reads one hire record of tag 24 with the bound of each field.
func (w *World) readHire(rs *restoreCtx, r *codec.Reader) hireRec {
	next := int64(w.state.nextAgent)
	h := hireRec{id: HireID(r.Bounded("hire", maxID)), started: true}
	h.employer = AgentID(r.Bounded("employer", next-1))
	h.worker = AgentID(r.Bounded("worker", next-1))
	h.wage = r.Bounded("wage", MaxPrice)
	h.escrow = int(r.Bounded("escrow account", int64(w.ledger.Len())-1))
	h.formed = r.Bounded("formation tick", rs.tick-1)
	h.region = RegionID(r.Index(uint64(len(w.cfg.Regions))))
	h.paid = r.Bounded("wages paid", math.MaxInt64)
	return h
}

// employmentCross completes the hires from the jobs of their workers and checks
// them: both parties are live agents of the region, the employer's role has the
// recipe on its hire list, the worker has the job of the hire (the employer as its
// principal, a tick or more worked and a tick or more left), the Escrow account is an
// open one of the region that no other hire uses, and no job names a principal that
// is neither a trainer nor an employer of a hire. It then derives what is not saved:
// the hire of each worker, the executions held open, the hires open by region and the
// free list of Escrow accounts when the merchants module is off (the merchants
// module's own cross-check builds the shared one when it is on). The money in the
// accounts is the audit's.
func (w *World) employmentCross() error {
	e := w.employment
	used := make(map[int]bool, len(e.hires))
	for i := range e.hires {
		h := &e.hires[i]
		if msg := w.hireShapeFault(h, used); msg != "" {
			return fmt.Errorf("%w: hire %d: %s", ErrCheckpointCorrupt, h.id, msg)
		}
		e.grow(max(h.worker, h.employer))
		e.workerOf[h.worker] = h.id + 1
		e.hired[h.employer] += h.k
		if e.hired[h.employer] > e.roles[w.agents[h.employer].role].maxHires {
			return fmt.Errorf("%w: hire %d: employer %d holds more than MaxHires %d executions", ErrCheckpointCorrupt, h.id, h.employer, e.roles[w.agents[h.employer].role].maxHires)
		}
		e.open[h.region]++
		if e.open[h.region] > e.maxOpen {
			return fmt.Errorf("%w: hire %d: region %d holds more than MaxOpen %d hires", ErrCheckpointCorrupt, h.id, h.region, e.maxOpen)
		}
	}
	if err := w.hiredJobsFault(); err != nil {
		return err
	}
	if w.mer == nil || !w.mer.on {
		w.rebuildHireFree(used)
	}
	return nil
}

// hireShapeFault returns what is wrong with the hire h read from the checkpoint, or
// "", and completes its derived fields from the job of its worker. It marks the
// account in used.
func (w *World) hireShapeFault(h *hireRec, used map[int]bool) string {
	if msg := w.hirePartiesFault(h); msg != "" {
		return msg
	}
	if used[h.escrow] {
		return fmt.Sprintf("account %d is used by two hires", h.escrow)
	}
	used[h.escrow] = true
	if v, err := w.ledger.Account(h.escrow); err != nil || v.Kind != ledger.KindEscrow || v.Closed || v.Region != int(h.region) {
		return fmt.Sprintf("account %d is not an open Escrow account of region %d", h.escrow, h.region)
	}
	if msg := w.hireJobFault(h); msg != "" {
		return msg
	}
	return hirePaidFault(h)
}

// hirePaidFault returns what is wrong with the wages paid of h, or "": each worked tick
// paid the wage of the executions of the job at the time, and the executions only fall,
// so the wages paid are at least worked * wage * k.
func hirePaidFault(h *hireRec) string {
	if _, ok := payroll.Bill(h.wage, h.ticks, h.k); !ok {
		return fmt.Sprintf("the bill of %d ticks at wage %d for %d executions does not fit", h.ticks, h.wage, h.k)
	}
	floor, ok := payroll.Bill(h.wage, h.worked, h.k) // fits: the bill above does and worked < ticks
	if !ok || h.paid < floor {
		return fmt.Sprintf("the wages paid %d are below the %d ticks worked at wage %d for %d executions", h.paid, h.worked, h.wage, h.k)
	}
	return ""
}

// hirePartiesFault returns what is wrong with the two parties of the hire h, or "":
// both are live agents of the hire's region, the employer's role has an employer
// block, the worker's role is Hireable and the worker has no other hire.
func (w *World) hirePartiesFault(h *hireRec) string {
	e := w.employment
	if !w.liveAgent(h.employer) || !w.liveAgent(h.worker) || h.employer == h.worker {
		return "a party is not a live agent, or the employer and the worker are one agent"
	}
	er, wr := &w.agents[h.employer], &w.agents[h.worker]
	switch {
	case !e.roles[er.role].on || !w.cfg.Roles[wr.role].Hireable:
		return "the employer's role has no employer block or the worker's role is not Hireable"
	case int(h.worker) < len(e.workerOf) && e.workerOf[h.worker] != 0:
		return fmt.Sprintf("worker %d has another hire", h.worker)
	case w.regionOf(er) != h.region || w.regionOf(wr) != h.region:
		return fmt.Sprintf("a party is not in region %d", h.region)
	}
	return ""
}

// hireJobFault checks the job of the worker of h against the hire and sets the fields
// the job gives: the recipe, the duration, the executions and the ticks worked.
func (w *World) hireJobFault(h *hireRec) string {
	job := &w.agents[h.worker].job
	if !w.agents[h.worker].hasJob() || job.principal != h.employer {
		return "its worker has no job with the employer as principal"
	}
	if !slices.Contains(w.employment.roles[w.agents[h.employer].role].list, job.recipe) {
		return "the recipe of its worker's job is not on the employer's hire list"
	}
	h.recipe, h.ticks, h.k = job.recipe, w.cfg.Recipes[job.recipe].Duration, job.executions
	if job.remaining < 1 || job.remaining >= h.ticks || h.wage < 1 || h.paid%h.wage != 0 {
		return "its worker's job has no working tick behind it and one ahead, or the wages paid are not a whole number of wages"
	}
	h.worked = h.ticks - job.remaining
	if h.formed+h.worked != w.Tick() { // one wage a tick from the formation tick: P2.7, then P2.1
		return fmt.Sprintf("formed at tick %d with %d ticks worked, but the checkpoint is at tick %d", h.formed, h.worked, w.Tick())
	}
	return ""
}

// hiredJobsFault returns an error when the job of a live agent names a principal
// other than the agent that is neither the trainer of a trainee nor the employer of
// the agent's hire.
func (w *World) hiredJobsFault() error {
	var err error
	w.eachLive(func(a *agent) {
		if err != nil || !a.hasJob() || a.job.principal == a.id || w.skills != nil && w.skills.isTrainee(a.id) {
			return
		}
		if h := w.employment.hireOfWorker(a.id); h == nil || h.employer != a.job.principal {
			err = fmt.Errorf("%w: the job of agent %d names principal %d, which hires it by no hire", ErrCheckpointCorrupt, a.id, a.job.principal)
		}
	})
	return err
}

// rebuildHireFree sets the free list of Escrow accounts of each region to the open
// Escrow accounts of the region that no hire uses (spec-employment.md 16).
func (w *World) rebuildHireFree(used map[int]bool) {
	free := make([][]int, len(w.employment.free))
	for id := range w.ledger.Len() {
		if v, err := w.ledger.Account(id); err == nil && v.Kind == ledger.KindEscrow && !v.Closed && !used[id] {
			free[v.Region] = append(free[v.Region], id)
		}
	}
	for g := range free {
		w.employment.free[g].Rebuild(free[g])
	}
}
