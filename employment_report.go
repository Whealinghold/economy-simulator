package economysim

import (
	"context"
	"log/slog"

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
