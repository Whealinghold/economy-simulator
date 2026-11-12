package economysim

import (
	"context"
	"fmt"
	"log/slog"

	payroll "github.com/Whealinghold/economy-sim/internal/employment"
	"github.com/Whealinghold/economy-sim/internal/ledger"
	"github.com/Whealinghold/economy-sim/internal/logkeys"
)

// This file is the employment module in its payroll form (spec-employment.md 24,
// EMP; ADR 0011): after clearing, an agent of an employer role pays a share of
// its receipts of the tick to the households of the Hireable roles that
// worked. It holds the module's state, its validator and, with the position,
// its rule; the arithmetic is internal/employment, and this file converts at the
// call (architecture.md 2.9, 2.10.1). The module has no hire, no escrow, no
// per-agent state and no checkpoint section.

// employment is the state of the module: the roles it reads, by RoleID. It is
// derived from the configuration and saved nowhere (spec-employment.md 24.1.7).
// It is nil when the module is off.
type employment struct {
	// payrollBP is the PayrollBP of each role with an employer block, 0 for a
	// role with none.
	payrollBP []int64
	// hireable says the role is Hireable.
	hireable []bool

	// The rest is scratch of one pass of P4.1b, held so that the position
	// allocates nothing once the scratch and the welfare slots have grown: it is rebuilt every tick and saved
	// nowhere. payers and workers are by RegionID, in ascending agent ID.
	payers  [][]payroll.Payer
	workers [][]payroll.Worker
	rows    []payroll.Transfer
	// byRole is the tally of the region being paid, by RoleID.
	byRole []payrollTally
}

// payrollTally is the PayrollPaid of one role in the region being paid, and
// last the worker last counted in Workers (so a worker with several rows of the
// role counts once) plus 1, 0 for none.
type payrollTally struct {
	ev   PayrollPaid
	last int
}

// newEmployment builds the state for a canonical configuration that passed
// Validate: nil when the module is off.
func newEmployment(c *Config) *employment {
	if c.Modules.Employment == nil {
		return nil
	}
	e := &employment{
		payrollBP: make([]int64, len(c.Roles)), hireable: make([]bool, len(c.Roles)),
		payers: make([][]payroll.Payer, len(c.Regions)), workers: make([][]payroll.Worker, len(c.Regions)),
		byRole: make([]payrollTally, len(c.Roles)),
	}
	for i := range c.Roles {
		e.hireable[i] = c.Roles[i].Hireable
		if b := c.Roles[i].Employer; b != nil {
			e.payrollBP[i] = b.PayrollBP
		}
	}
	return e
}

// validateEmployment checks the module and the role fields of the payroll form
// (spec-employment.md 24.4.1): the bound of PayrollBP, the module dependency of
// Hireable and of the block, a role that is both, and the module on with no
// payer role or no Hireable role (a no-op that would look like a result).
func (v *validator) validateEmployment() {
	c := &v.c
	on := c.Modules.Employment != nil
	var payers, workers int
	for i := range c.Roles {
		r := &c.Roles[i]
		v.employmentRole([]string{"roles", r.ID}, r, on)
		if r.Employer != nil {
			payers++
		}
		if r.Hireable {
			workers++
		}
	}
	if !on {
		return
	}
	base := []string{"modules", "employment"}
	if payers == 0 {
		v.rep.add(ErrEmploymentShape, path(base, "employer"), "0", "1")
	}
	if workers == 0 {
		v.rep.add(ErrEmploymentShape, path(base, "hireable"), "0", "1")
	}
}

// employmentRole checks the employment fields of one role; on says the module
// is present.
func (v *validator) employmentRole(rb []string, r *Role, on bool) {
	if r.Employer != nil {
		eb := path(rb, "employer")
		if !on {
			v.rep.add(ErrModuleDependency, eb, "employment", "")
		}
		v.inRange(path(eb, "payroll_bp"), r.Employer.PayrollBP, 1, wholeBP)
	}
	if r.Hireable && !on {
		v.rep.add(ErrModuleDependency, path(rb, "hireable"), "employment", "")
	}
	if r.Hireable && r.Employer != nil {
		v.rep.add(ErrEmploymentShape, path(rb, "hireable"), "true", "false")
	}
}

// isPayer says the role has an employer block. The pooled regional wage that the
// cost anchor reads excludes the agents of these roles (spec-employment.md
// 24.2.3, 8.4); p12Anchors calls it under the module's pointer.
func (e *employment) isPayer(role RoleID) bool { return e.payrollBP[role] > 0 }

// p41bPayroll is P4.1b (spec-employment.md 24.1): once the markets have settled,
// the agents of the employer roles pay a share of the tick's receipts to the
// households of the Hireable roles that worked, region by region in ascending
// RegionID. No random number is drawn and no state is kept. It fails the step
// only on a refusal the budget guarantees cannot happen, or on an overflow.
func (w *World) p41bPayroll() error {
	e := w.employment
	e.collect(w)
	for g := range w.cfg.Regions {
		if len(e.payers[g]) == 0 {
			continue
		}
		if err := w.payRegion(RegionID(g)); err != nil {
			return err
		}
	}
	return nil
}

// collect fills the payers and the workers of every region, in ascending agent
// ID (24.1.2): the receipts and the free balance of a payer, the executions of a
// worker that is not on an own-use job.
func (e *employment) collect(w *World) {
	for g := range e.payers {
		e.payers[g], e.workers[g] = e.payers[g][:0], e.workers[g][:0]
	}
	for i := w.firstHousehold; i < len(w.agents); i++ {
		if a := &w.agents[i]; a.weight >= 1 && (e.payrollBP[a.role] > 0 || e.hireable[a.role]) {
			e.collectAgent(w, i)
		}
	}
}

// collectAgent adds the agent of slot i to the payers or the workers of its region.
func (e *employment) collectAgent(w *World, i int) {
	a := &w.agents[i]
	v, err := w.ledger.Account(a.account)
	if err != nil {
		invariant("the account of an agent is unreadable: " + err.Error())
	}
	if bp := e.payrollBP[a.role]; bp > 0 {
		e.payers[v.Region] = append(e.payers[v.Region], payroll.Payer{
			Ref: i, Receipts: w.welfare.receiptsOf(i), BP: bp, Balance: v.Balance - v.Reserved})
	}
	if k, ownUse, _ := w.welfare.workRead(i); e.hireable[a.role] && k >= 1 && !ownUse {
		e.workers[v.Region] = append(e.workers[v.Region], payroll.Worker{Ref: i, K: k})
	}
}

// payRegion applies the rule to one region: the budgets, the split, the transfers
// under wage_pay and the welfare feeds, then the report.
//
// Excluded from coverage (testing.md 9.5): the `return` of the overflow of Split and
// of a refused transfer of payRow. The budgets are bounded by the payers' balances,
// which the ledger bounds, and a transfer within a budget cannot be refused, so no
// test can inject either from the root.
func (w *World) payRegion(g RegionID) error {
	e := w.employment
	clear(e.byRole)
	for _, p := range e.payers[g] {
		due, budget, ok := payroll.Budget(p)
		if !ok {
			invariant("the budget of a payer does not fit in int64")
		}
		t := &e.byRole[w.agents[p.Ref].role].ev
		t.Due += Money(due)
		if budget >= 1 {
			t.Payers++
		}
		if budget < due {
			w.payrollCut(p.Ref, due, budget)
		}
	}
	var ok bool
	if e.rows, ok = payroll.Split(e.rows[:0], e.payers[g], e.workers[g]); !ok {
		return fmt.Errorf("payroll of region %d: %w", g, ErrOverflow)
	}
	for _, r := range e.rows {
		if err := w.payRow(g, r); err != nil {
			return err
		}
	}
	w.payrollReport(g)
	return nil
}

// payRow moves one row of the payroll and feeds the two welfare records: Y_t of
// the worker and V_t of the payer (24.2.1), and tallies it for the report.
func (w *World) payRow(g RegionID, r payroll.Transfer) error {
	from, to := &w.agents[r.Payer], &w.agents[r.Worker]
	if err := w.ledger.Transfer(from.account, to.account, r.Amount, ledger.WagePay); err != nil {
		return fmt.Errorf("wage from agent %d to agent %d in region %d: %w", from.id, to.id, g, err)
	}
	w.welfare.addWagePaid(r.Payer, r.Amount)
	w.welfare.addEarned(r.Worker, r.Amount)
	if !w.obs.supplied {
		return nil
	}
	t := &w.employment.byRole[from.role]
	t.ev.Paid += Money(r.Amount)
	if t.last != r.Worker+1 {
		k, _, _ := w.welfare.workRead(r.Worker)
		t.ev.Workers, t.last = t.ev.Workers+k, r.Worker+1
	}
	return nil
}

// payrollCut logs, at Debug and guarded by Enabled, a payer whose budget was cut
// at its balance (24.6.3). It is never a Warn: a cut is an event field (Cut).
func (w *World) payrollCut(agent int, due, applied int64) {
	if ctx := context.Background(); w.log.Enabled(ctx, slog.LevelDebug) {
		w.log.LogAttrs(ctx, slog.LevelDebug, "payroll cut at balance", slog.Int64(logkeys.Agent, int64(agent)),
			slog.Int64(logkeys.Role, int64(w.agents[agent].role)), slog.Int64(logkeys.Due, due), slog.Int64(logkeys.Applied, applied))
	}
}

// payrollReport delivers PayrollPaid for each employer role of region g whose Due
// or Paid is non-zero, in ascending role order (24.6.1). The event is built only
// with an observer.
func (w *World) payrollReport(g RegionID) {
	if !w.obs.supplied {
		return
	}
	for role := range w.employment.byRole {
		ev := w.employment.byRole[role].ev
		if ev.Due == 0 && ev.Paid == 0 {
			continue
		}
		ev.Tick, ev.Region, ev.Role, ev.Cut = w.Tick(), g, RoleID(role), ev.Due-ev.Paid
		w.obs.deliver(ev)
	}
}
